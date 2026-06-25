package webserver

// Cap-gated chat history and export routes for the web-share surface.
//
// Trust model: both routes are wrapped in requireCapability (registered in
// server.go:setupRoutes). The middleware enforces HMAC signature, grant-list
// membership, web-enabled status, and — because the route carries the {id}
// path parameter — claims.SID == {id}. This per-session isolation satisfies
// T-151-04 (a cap for session A cannot read session B's thread).
//
// Import-cycle avoidance (T-151-09): this file imports no package that
// imports webserver. The caller (api.go) wires a chatProvider callback
// (SetChatProvider) at construction time. The callback returns pre-serialized
// bytes so this file has no knowledge of the ChatStore type.

import (
	"fmt"
	"net/http"
)

// SetChatProvider installs the callback that the cap-gated chat routes use to
// fetch a session's message thread (as pre-marshaled JSON) and its Markdown
// export. Must be called before Start(); not mutex-protected (single setter,
// set once — mirrors SetSessionResolver / SetFilesHandler).
//
// The provider signature avoids a circular import: the caller passes a
// closure; this file sees only []byte + string + bool + error (T-151-09).
//
// Provider contract (WR-03):
//   - err != nil           → internal failure on an existing session; routes
//     return 500 (the error is surfaced, not masked).
//   - err == nil, !found    → session has no chat store; routes return 404.
//   - err == nil, found     → history is valid JSON bytes; markdown is the export.
func (ws *WebServer) SetChatProvider(fn func(sessionID string) (history []byte, markdown string, found bool, err error)) {
	ws.chatProvider = fn
}

// handleChatHistory serves GET /api/chat/{id}/history (capability-gated).
// Writes the pre-marshaled JSON bytes from the provider with
// Content-Type: application/json. Returns 503 if the provider is not wired,
// 500 if the provider reports an internal error on an existing session, and
// 404 if the session has no registered store.
func (ws *WebServer) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	fn := ws.chatProvider
	if fn == nil {
		http.Error(w, "chat provider not configured", http.StatusServiceUnavailable)
		return
	}
	history, _, found, err := fn(sessionID)
	if err != nil {
		// Internal failure on a session that exists — surface it as 500 rather
		// than masking it as 404 (WR-03). The provider logs the underlying cause.
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(history)
}

// handleChatExport serves GET /api/chat/{id}/export (capability-gated).
// Writes the Markdown export from the provider with
// Content-Type: text/markdown; charset=utf-8 and a Content-Disposition
// attachment header. Returns 503 if the provider is not wired, 500 if the
// provider reports an internal error on an existing session, and 404 if the
// session has no registered store.
func (ws *WebServer) handleChatExport(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	fn := ws.chatProvider
	if fn == nil {
		http.Error(w, "chat provider not configured", http.StatusServiceUnavailable)
		return
	}
	_, markdown, found, err := fn(sessionID)
	if err != nil {
		// Internal failure on a session that exists — surface it as 500 rather
		// than masking it as 404 (WR-03). The provider logs the underlying cause.
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="chat-%s.md"`, sessionID))
	_, _ = w.Write([]byte(markdown))
}
