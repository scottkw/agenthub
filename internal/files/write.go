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
	"io"
	"net/http"
	"path/filepath"
)

// maxUploadBytes is the server-side upload cap — FSW-12. Strictly matches
// the 50 MiB milestone-locked value. Enforced via http.MaxBytesReader BEFORE
// ParseMultipartForm so the cap fires before any bytes hit disk.
//
// Mirrors the maxPreviewBytes idiom from handler.go:104.
const maxUploadBytes = 50 << 20

// Write implements PUT /api/files/write. Reads the request body and calls
// sb.WriteFileAtomic(relPath, data) to durably persist the content via the
// temp+Sync+rename atomic write pattern (FSW-01). On success it responds 200
// with a FileWriteResponse JSON body. (FSW-08 success criterion #2)
func (h *Handler) Write(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	rel := h.relPath(r)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := sb.WriteFileAtomic(rel, data); err != nil {
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
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
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
	rel := h.relPath(r)
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
	rel := h.relPath(r)
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
//   - validation/traversal    → 403 "access denied: ..." (same as read side)
//   - everything else         → 500
func writeWriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrProtectedSystemFile):
		http.Error(w, "Protected system file", http.StatusForbidden)
	case isValidationError(err):
		http.Error(w, "access denied: "+err.Error(), http.StatusForbidden)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// isValidationError reports whether err originated from validateRelativePath
// or the traversal-reject check in validateAndClean. These errors always begin
// with "files: " and describe path-level rejections that map to 403.
func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// All errors from validateRelativePath / validateAndClean start with "files: "
	// and do NOT include the ErrProtectedSystemFile message, so we can distinguish
	// them from OS errors (which don't carry the "files: " prefix).
	return len(msg) > 7 && msg[:7] == "files: " && !errors.Is(err, ErrProtectedSystemFile)
}
