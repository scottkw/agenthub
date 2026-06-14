// write.go adds the HTTP write surface to the existing stateless Handler.
// Each method mirrors the read-side session-resolution + status-mapping
// convention from handler.go:110-131:
//
//   - 404 "session not found"   — sandboxFor error
//   - 403 "access denied: ..."  — path validation / traversal failure
//   - 403 "Protected system file" — Sandbox denylist sentinel (FSW-06)
//   - 400                        — shape error (bad request body)
//   - 200 + JSON                 — success: FileWriteResponse or FileOpResponse
//
// Auth-less by design — the daemon Unix socket / Windows named pipe is the
// trust boundary (WEB-01 loopback-trust precedent). Capability gating
// (files.write) is a Phase 124 (CAP-08) concern and must NOT be added here.
//
// All five methods add methods to the existing *Handler type defined in
// handler.go — do NOT create a new type.
package files

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

// MaxUploadBytes is the server-side upload cap — FSW-12. Strictly matches
// the 50 MiB milestone-locked value. Enforced via http.MaxBytesReader BEFORE
// ParseMultipartForm so the cap fires before any bytes hit disk.
//
// WR-06: exported so tests can assert this matches the frontend MAX_UPLOAD_BYTES
// constant (filesApi.ts) — cross-surface parity is release-blocking.
//
// Mirrors the maxPreviewBytes idiom from handler.go:104.
const MaxUploadBytes = 50 << 20

// maxUploadBytes is the internal alias kept for backward compat with
// existing usages in this file (unexported name convention in handler.go).
const maxUploadBytes = MaxUploadBytes

// ErrPathValidation is the sentinel error type for all path-level validation
// failures (empty path, traversal, device names, ADS, etc.). Errors from
// validateRelativePath and validateAndClean are wrapped with this sentinel so
// callers can use errors.Is(err, ErrPathValidation) instead of brittle
// string-prefix checks. (IN-03 / WR-02 robustness fix)
var ErrPathValidation = errors.New("files: path validation error")

// Write implements PUT /api/files/write (and HEAD /api/files/write for the
// canWrite probe — CR-02). HEAD returns 200 + no body immediately after
// session resolution so the capability middleware (requireFilesWrite) fires
// before any body-read work, making the 403-with-"files.write" signal
// reachable from the frontend probeWrite call in useFilesCapability.
func (h *Handler) Write(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	// CR-02: short-circuit HEAD before reading the body. The requireFilesWrite
	// middleware already ran (perm check passed), so 200 = "write is permitted."
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	// WR-01: missing/empty path on a write verb is a client error → 400.
	rel := r.URL.Query().Get("path")
	if rel == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}
	// EDIT-05/08 optimistic concurrency. If-Match present + not wildcard → the
	// caller asserts a known on-disk validator; reject (412) if the file changed.
	// Wildcard ("*") or absent header → force-overwrite / new-file path; proceed.
	if ifMatch := r.Header.Get("If-Match"); ifMatch != "" && ifMatch != "*" {
		if fi, statErr := sb.Stat(rel); statErr == nil { // target exists; missing → new file, skip
			cur := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))
			if ifMatch != cur {
				http.Error(w, "file modified by another process", http.StatusPreconditionFailed)
				return
			}
		}
	}
	// WR-06: cap the write body the same way Upload caps multipart (FSW-12 DoS
	// mitigation). The same Handler is mounted on the relay TCP surface and the
	// webserver, so loopback trust does not justify an unbounded io.ReadAll.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Thread the expected validator into WriteFileAtomic for the CR-01 re-check
	// immediately before rename (narrowing the TOCTOU window). Wildcard ("*")
	// or absent header bypasses the re-check (force-overwrite / new-file path).
	ifMatchForWrite := r.Header.Get("If-Match")
	if err := sb.WriteFileAtomic(rel, data, ifMatchForWrite); err != nil {
		writeWriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(FileWriteResponse{Path: rel, Size: int64(len(data))})
}

// Upload implements POST /api/files/upload. Parses a multipart form, enforcing
// the 50 MiB cap via http.MaxBytesReader BEFORE ParseMultipartForm (FSW-12 /
// T-123-13 DoS mitigation). The "file" part's filename is sanitized with
// filepath.Base to strip any "../" traversal components (FSW-05 / T-123-12).
// The final path is routed through sb.WriteFileAtomic.
//
// Form fields:
//
//	dir  — relative directory within the sandbox (default ".")
//	file — the uploaded file part
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	// Cap MUST be applied before ParseMultipartForm — otherwise the multipart
	// parser buffers the entire body to disk before the limit is checked.
	// This is the FSW-12 / T-123-13 enforcement point.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		// IN-05: distinguish the cap error (413) from a malformed multipart body (400).
		// ParseMultipartForm wraps *http.MaxBytesError when the cap fires; any other
		// error is a shape problem in the client request (truncated boundary, etc.).
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "malformed multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}
	dir := r.FormValue("dir")
	if dir == "" {
		dir = "."
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file part: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// filepath.Base strips any directory components including "../" traversal —
	// T-123-12 / FSW-05. validateAndClean inside WriteFileAtomic adds a second
	// defense layer.
	safeName := filepath.Base(header.Filename)

	// CR-01: reject empty, dot, or separator-only filenames that filepath.Base
	// produces from empty or path-separator-only inputs. Without this guard,
	// safeName=="." collapses target to dir (a directory) and WriteFileAtomic
	// tries to rename a regular temp file onto an existing directory → opaque
	// 500 + stray .agenthub-tmp-* sibling.
	if safeName == "" || safeName == "." || safeName == ".." || strings.ContainsAny(safeName, `/\`) {
		http.Error(w, "invalid upload filename", http.StatusBadRequest)
		return
	}

	target := filepath.Join(dir, safeName)

	// io.ReadAll is bounded by the MaxBytesReader cap applied above.
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "read upload: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := sb.WriteFileAtomic(target, data); err != nil {
		writeWriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(FileWriteResponse{Path: target, Size: int64(len(data))})
}

// Delete implements DELETE /api/files/delete. Calls sb.Delete(relPath) to
// remove a file or recursively remove a directory subtree within the sandbox
// (FSW-04). Responds 200 with FileOpResponse{OK: true}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	// WR-01: missing/empty path on a write verb is a client error → 400.
	rel := r.URL.Query().Get("path")
	if rel == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}
	if err := sb.Delete(rel); err != nil {
		writeWriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(FileOpResponse{Path: rel, OK: true})
}

// renameRequest is the JSON body decoded by Rename.
type renameRequest struct {
	OldRel string `json:"oldRel"`
	NewRel string `json:"newRel"`
}

// Rename implements POST /api/files/rename. Decodes a JSON body with fields
// "oldRel" and "newRel" and calls sb.Rename. Both paths are validated by the
// Sandbox (validateAndClean on both — FSW-02 / T-123-01 destination traversal
// risk mitigation). Responds 200 with FileOpResponse{OK: true}.
func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	var req renameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "decode body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.OldRel == "" || req.NewRel == "" {
		http.Error(w, "oldRel and newRel are required", http.StatusBadRequest)
		return
	}
	if err := sb.Rename(req.OldRel, req.NewRel); err != nil {
		writeWriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(FileOpResponse{Path: req.NewRel, OK: true})
}

// Mkdir implements POST /api/files/mkdir. Reads the target path from the
// "path" query parameter and calls sb.MkdirAll so all missing parent
// directories are created in one call (FSW-03). Responds 200 with
// FileOpResponse{OK: true}.
func (h *Handler) Mkdir(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	// WR-01: missing/empty path on a write verb is a client error → 400.
	rel := r.URL.Query().Get("path")
	if rel == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}
	if err := sb.MkdirAll(rel); err != nil {
		writeWriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(FileOpResponse{Path: rel, OK: true})
}

// writeWriteError maps Sandbox write errors to appropriate HTTP status codes,
// mirroring the read-side 403/404/400 convention from handler.go (RESEARCH
// §Pattern 5). The mapping is:
//
//   - ErrProtectedSystemFile  → 403 "Protected system file" (T-123-14, FSW-06)
//   - ErrPathValidation       → 403 "access denied: ..." (traversal/validation)
//   - fs.ErrNotExist          → 404 (missing parent directory, missing source)
//   - fs.ErrExist             → 409 (rename/mkdir onto existing target)
//   - everything else         → 500
func writeWriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPreconditionFailed):
		http.Error(w, "file modified by another process", http.StatusPreconditionFailed)
	case errors.Is(err, ErrProtectedSystemFile):
		http.Error(w, "Protected system file", http.StatusForbidden)
	case errors.Is(err, ErrPathValidation):
		http.Error(w, "access denied: "+err.Error(), http.StatusForbidden)
	case errors.Is(err, fs.ErrNotExist):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, fs.ErrExist):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
