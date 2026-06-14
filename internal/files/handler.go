// Handler is the HTTP layer of the internal/files package. It is
// deliberately stateless — one *Handler can be mounted on multiple
// muxes:
//   - Plan 05 wires it to the daemon's session-keyed file routes.
//   - Phase 119 wires it to the webserver via the SetFilesHandler
//     parity point.
//
// The resolver function injected at construction is the seam that lets
// each consumer decide how a session ID maps to a Sandbox — the daemon
// looks up its in-memory session table; the webserver delegates back to
// the daemon via the same lookup. The handler itself imports nothing
// from internal/daemon, internal/relay, or internal/webserver.
//
// All routes accept these query parameters:
//
//	session=<id>   — REQUIRED; resolved via the injected resolver.
//	path=<rel>     — OPTIONAL; defaults to "." (the sandbox root).
//
// On a missing/unknown session: 404 "session not found".
// On a path that fails sandbox validation: 403 "access denied: ...".
// On a directory passed to Read: 400 "is a directory".
//
// The handler enforces Plan 02's three load-bearing invariants BEFORE
// delegating to http.ServeContent on the Read path:
//  1. 5 MiB preview cap — returns 413 BEFORE any streaming.
//  2. 0-byte short-circuit — returns 200+empty body, never 416
//     (golang/go#54794 / FS-07).
//  3. MIME cascade — extension first, then sniff if unknown.
package files

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// sandboxResolver is the session→Sandbox lookup the handler depends on.
// Plan 05 (daemon) and Phase 119 (webserver) provide concrete resolvers.
type sandboxResolver func(sessionID string) (*Sandbox, error)

// Handler is the stateless HTTP surface for FS-03..FS-07. Each method
// implements one HTTP verb/route:
//
//	List : GET  /list?session=<>&path=<>
//	Stat : GET  /stat?session=<>&path=<>
//	Read : GET  /read?session=<>&path=<>    (also handles HEAD via ServeContent)
//
// All three methods are safe to mount on a vanilla http.ServeMux.
type Handler struct {
	resolve sandboxResolver
}

// NewHandler returns a Handler that uses resolve to look up the Sandbox
// for an incoming session ID. resolve must NOT be nil — a nil resolver
// is a programming error caught by Go's nil-call panic on the first
// request.
func NewHandler(resolve sandboxResolver) *Handler {
	return &Handler{resolve: resolve}
}

// sandboxFor reads the session query parameter and resolves it to a
// Sandbox. On any failure (missing param, unknown session) it returns a
// non-nil error and the caller MUST respond 404 "session not found".
func (h *Handler) sandboxFor(r *http.Request) (*Sandbox, string, error) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		return nil, "", errors.New("missing session")
	}
	sb, err := h.resolve(sessionID)
	if err != nil {
		return nil, sessionID, err
	}
	if sb == nil {
		return nil, sessionID, errors.New("resolver returned nil sandbox")
	}
	return sb, sessionID, nil
}

// relPath reads the path query parameter, defaulting to "." (the
// sandbox root) when missing or empty.
func (h *Handler) relPath(r *http.Request) string {
	p := r.URL.Query().Get("path")
	if p == "" {
		return "."
	}
	return p
}

// maxListEntries caps a single directory listing — Pitfall 5 (large
// directory memory blowup). The cap is enforced via the streaming
// f.ReadDir(maxListEntries) form on the open os.File; we never call
// os.ReadDir which would buffer all entries.
const maxListEntries = 10000

// maxPreviewBytes is the v3.4 server-side preview cap — Pitfall 5
// (Read large file). Strictly greater-than: a file of exactly
// 5*1024*1024 bytes is allowed (boundary unit-tested).
const maxPreviewBytes = 5 * 1024 * 1024

// List implements GET /list. Returns a FileListResponse JSON body with
// up to maxListEntries entries (Truncated=true when capped). On darwin
// runtime, names beginning with "._" are filtered out (macOS resource
// fork — Pitfall 15).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	rel := h.relPath(r)
	dir, err := sb.Open(rel)
	if err != nil {
		http.Error(w, "access denied: "+err.Error(), http.StatusForbidden)
		return
	}
	defer dir.Close()
	fi, err := dir.Stat()
	if err != nil {
		http.Error(w, "stat failed", http.StatusInternalServerError)
		return
	}
	if !fi.IsDir() {
		http.Error(w, "not a directory", http.StatusBadRequest)
		return
	}
	// Streaming cap — Pitfall 5. ReadDir(n>0) reads at most n entries
	// from the open file's directory stream; io.EOF means "fewer than
	// n entries existed" and is treated as success.
	//
	// Read one extra entry beyond the cap to disambiguate the exactly-
	// maxListEntries case (WR-01): if we asked for exactly maxListEntries
	// and got that many back, we cannot tell whether more existed. Probing
	// one past the cap lets us set Truncated correctly: len > cap means
	// truncated; len <= cap means we saw the entire directory.
	entries, err := dir.ReadDir(maxListEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "read directory failed", http.StatusInternalServerError)
		return
	}
	truncated := len(entries) > maxListEntries
	if truncated {
		entries = entries[:maxListEntries]
	}
	result := FileListResponse{
		Entries:   make([]FileEntry, 0, len(entries)),
		Truncated: truncated,
	}
	for _, entry := range entries {
		name := entry.Name()
		// darwin-only ._-resource-fork filter — Pitfall 15.
		if runtime.GOOS == "darwin" && strings.HasPrefix(name, "._") {
			continue
		}
		mode := entry.Type()
		ext := extensionMIME(name)
		// Phase 120 UAT-1: per-entry Info() so size/mtime show up in the
		// UI without requiring a follow-up /stat round-trip per row. The
		// original Pitfall 6 ban on per-entry stat was overly conservative
		// — users couldn't see file sizes at all. Graceful fallback: if
		// Info() fails (the file was unlinked between ReadDir and Info),
		// we still emit the entry with zeroed size/mtime; the directory
		// listing remains useful.
		var size int64
		var mtime string
		if fi, infoErr := entry.Info(); infoErr == nil {
			size = fi.Size()
			mtime = fi.ModTime().UTC().Format(time.RFC3339)
		}
		result.Entries = append(result.Entries, FileEntry{
			Name:      name,
			Size:      size,
			Mtime:     mtime,
			Mode:      mode.String(),
			IsDir:     entry.IsDir(),
			IsSymlink: mode&os.ModeSymlink != 0,
			IsBinary:  ext == "", // extension-only heuristic
			MIME:      ext,       // may be ""
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Stat implements GET /stat. Returns a single FileEntry (not wrapped
// in a list) describing the named path. MIME is populated via the full
// cascade: extension first, then magic-byte sniff if the extension is
// unknown.
func (h *Handler) Stat(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	rel := h.relPath(r)
	f, err := sb.Open(rel)
	if err != nil {
		http.Error(w, "access denied: "+err.Error(), http.StatusForbidden)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "stat failed", http.StatusInternalServerError)
		return
	}
	// Name must be forward-slash normalized — Pitfall 14. filepath.Base
	// returns a basename, but on Windows it can contain backslashes if
	// the input had them; we defend in depth via ReplaceAll.
	name := strings.ReplaceAll(filepath.Base(rel), "\\", "/")
	if name == "." || name == "" {
		// "." refers to the sandbox root itself; use the root's
		// basename for a friendlier label.
		name = strings.ReplaceAll(filepath.Base(sb.RootPath()), "\\", "/")
	}
	mime := extensionMIME(rel)
	if mime == "" && !fi.IsDir() {
		mime = sniffMIME(f)
		// Caller contract for sniffMIME: rewind to start so subsequent
		// reads see the full content. Stat does not stream the body
		// itself, but defense-in-depth.
		_, _ = f.Seek(0, io.SeekStart)
	}
	entry := FileEntry{
		Name:      name,
		Size:      fi.Size(),
		Mtime:     fi.ModTime().UTC().Format(time.RFC3339),
		Mode:      fi.Mode().String(),
		IsDir:     fi.IsDir(),
		IsSymlink: fi.Mode()&os.ModeSymlink != 0,
		IsBinary:  !strings.HasPrefix(mime, "text/"),
		MIME:      mime,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entry)
}

// Read implements GET /read (and HEAD /read via http.ServeContent's
// automatic dispatch). Three load-bearing checks fire BEFORE delegation:
//
//  1. fi.Size() > maxPreviewBytes → 413 (FS-05 / Pitfall 5).
//  2. fi.Size() == 0              → 200 + empty body (FS-07,
//     golang/go#54794 mitigation; ServeContent would otherwise return
//     416 when a Range header accompanies a 0-byte file).
//  3. MIME cascade is resolved and Content-Type set so ServeContent
//     does not invoke its own DetectContentType (which would re-read
//     the file head and may misclassify).
//
// All Range/If-Modified-Since/Last-Modified/HEAD semantics are handled
// by http.ServeContent.
func (h *Handler) Read(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	rel := h.relPath(r)
	f, err := sb.Open(rel)
	if err != nil {
		http.Error(w, "access denied: "+err.Error(), http.StatusForbidden)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "stat failed", http.StatusInternalServerError)
		return
	}
	if fi.IsDir() {
		http.Error(w, "is a directory", http.StatusBadRequest)
		return
	}
	// (1) 5 MiB preview cap BEFORE any streaming — Pitfall 5 / FS-05.
	if fi.Size() > maxPreviewBytes {
		http.Error(w, "file too large for preview", http.StatusRequestEntityTooLarge)
		return
	}
	// (3) Resolve Content-Type via the cascade FIRST so we can short-
	// circuit zero-byte files with the correct header, and so
	// ServeContent does not invoke its own sniff.
	contentType := extensionMIME(rel)
	if contentType == "" {
		contentType = sniffMIME(f)
		_, _ = f.Seek(0, io.SeekStart)
	}
	w.Header().Set("Content-Type", contentType)
	// EDIT-05/08 ETag contract — emit the same "<UnixNano>-<size>" validator
	// that Handler.Write compares against If-Match. The client echoes this
	// header verbatim as If-Match on the next write (RESEARCH Open Q6 resolution:
	// server emits ETag, client echoes, eliminating RFC3339-vs-UnixNano mismatch).
	w.Header().Set("ETag", fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size())))
	// (2) 0-byte short-circuit BEFORE http.ServeContent — FS-07 /
	// golang/go#54794. ServeContent on a 0-byte file with a Range
	// header returns 416, which breaks the FS-07 contract.
	if fi.Size() == 0 {
		w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		return
	}
	// http.ServeContent handles Range / If-Modified-Since /
	// Last-Modified / ETag-by-modtime / HEAD (FS-05, FS-06).
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}
