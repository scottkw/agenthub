package daemon

// Mounts the chat history and export routes on the relay loopback surface.
//
// Background: the Wails desktop GUI reaches the relay server over a loopback
// TCP connection (127.0.0.1:<relayPort>); it cannot reach the daemon Unix
// socket. Phase 151 Plan 03 adds two read-only chat routes that the GUI will
// call for the scrollback load path (PERSIST-02) and the export UI
// (PERSIST-01). Like the remote-files routes in relay_remote_files.go, these
// routes are *API methods — they need access to the session engine's
// ChatStore — and therefore cannot be mounted inside the relay package's
// Server without creating a daemon↔relay import cycle. Instead we wrap the
// relay server in a parent mux that owns the chat routes and falls through to
// the relay server for all other paths.
//
// Trust boundary: the relay server is bound to 127.0.0.1 (loopback-only).
// No capability gate is needed on this surface; loopback reachability is the
// trust boundary (WEB-01 / ARCHITECTURE.md Decision 2).

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/scottkw/agenthub/internal/relay"
)

// handleChatHistory serves the full chat thread for the session identified by
// the {id} path parameter as a JSON array of relay.ChatMessage objects.
//
// Empty thread returns `[]` (not null). Unknown session returns 404.
func (a *API) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Defense-in-depth: reject ids outside the strict allowlist before doing
	// anything with the value, mirroring NewChatStore (IN-02). A valid session
	// id is crypto-random hex, so this can only reject a malformed/forged id —
	// which can never have a registered store anyway (404).
	if !validChatSessionID(id) {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	store, ok := a.engine.ChatStoreFor(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	// Messages() always returns a non-nil slice (make([]…, len)), so an empty
	// thread already serializes as `[]` not `null` — no nil guard needed (IN-01).
	msgs := store.Messages()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(msgs); err != nil {
		_ = err // header already committed; log nothing (content-only body)
	}
}

// handleChatExport renders the full chat thread as a Markdown document and
// sends it as an attachment. Content-Type is text/markdown; charset=utf-8
// and Content-Disposition names the download file.
//
// Unknown session returns 404.
func (a *API) handleChatExport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Defense-in-depth: reject ids outside the strict allowlist before building
	// the Content-Disposition filename from the value, mirroring NewChatStore
	// (IN-02). This makes header-injection unreachable locally rather than only
	// via the upstream "ids are crypto-random hex" invariant.
	if !validChatSessionID(id) {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	store, ok := a.engine.ChatStoreFor(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	md, err := store.Export()
	if err != nil {
		http.Error(w, "export failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="chat-%s.md"`, id))
	_, _ = w.Write([]byte(md))
}

// wrapRelayWithChat returns a handler that serves the chat read routes (with
// relay CORS) and falls through to the given handler for all other paths. See
// relay_remote_files.go for the canonical import-cycle rationale and the CORS
// requirement: the relay surface is cross-origin from the Wails webview
// (wails://… → 127.0.0.1:<relayPort>).
func (a *API) wrapRelayWithChat(next http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/chat/{id}/history", relay.FilesCORS(a.handleChatHistory))
	mux.HandleFunc("GET /api/chat/{id}/export", relay.FilesCORS(a.handleChatExport))

	// OPTIONS preflight for the GET routes. Although simple GET requests
	// without custom headers do not trigger a browser preflight, a custom
	// Accept header or a future POST verb would. Registering the preflight
	// handler now prevents a silent breakage and matches the pattern used by
	// the remote-files wrap routes.
	mux.HandleFunc("OPTIONS /api/chat/{id}/history", relay.FilesPreflight)
	mux.HandleFunc("OPTIONS /api/chat/{id}/export", relay.FilesPreflight)

	// All other paths (sessions WS, /api/files/*, /api/files/remote/*, etc.)
	// fall through to the wrapped handler. Go 1.22+ mux: "/" is the
	// least-specific match so the explicit chat routes above always win.
	mux.Handle("/", next)
	return mux
}
