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
// imports webserver. The caller (api.go) wires the provider callbacks
// (SetChatHistoryProvider / SetChatExportProvider) at construction time. The
// callbacks return pre-serialized bytes/string so this file has no knowledge
// of the ChatStore type.

import (
	"fmt"
	"net/http"
)

// SetChatHistoryProvider installs the callback that the cap-gated history route
// uses to fetch a session's message thread as pre-marshaled JSON. Must be
// called before Start(); not mutex-protected (single setter, set once —
// mirrors SetSessionResolver / SetFilesHandler).
//
// Split from the export provider (IN-04) so the history route never runs the
// Markdown export it does not serve.
//
// Provider contract (WR-03):
//   - err != nil           → internal failure on an existing session; the
//     route returns 500 (the error is surfaced, not masked).
//   - err == nil, !found    → session has no chat store; route returns 404.
//   - err == nil, found     → history is valid JSON bytes.
func (ws *WebServer) SetChatHistoryProvider(fn func(sessionID string) (history []byte, found bool, err error)) {
	ws.chatHistoryProvider = fn
}

// SetChatExportProvider installs the callback that the cap-gated export route
// uses to fetch a session's Markdown export. Must be called before Start();
// not mutex-protected. Split from the history provider (IN-04) so the export
// route never marshals the history JSON it does not serve.
//
// Provider contract mirrors SetChatHistoryProvider (WR-03 semantics).
func (ws *WebServer) SetChatExportProvider(fn func(sessionID string) (markdown string, found bool, err error)) {
	ws.chatExportProvider = fn
}

// handleChatHistory serves GET /api/chat/{id}/history (capability-gated).
// Writes the pre-marshaled JSON bytes from the provider with
// Content-Type: application/json. Returns 503 if the provider is not wired,
// 500 if the provider reports an internal error on an existing session, and
// 404 if the session has no registered store.
func (ws *WebServer) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	fn := ws.chatHistoryProvider
	if fn == nil {
		http.Error(w, "chat provider not configured", http.StatusServiceUnavailable)
		return
	}
	history, found, err := fn(sessionID)
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
	fn := ws.chatExportProvider
	if fn == nil {
		http.Error(w, "chat provider not configured", http.StatusServiceUnavailable)
		return
	}
	markdown, found, err := fn(sessionID)
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
