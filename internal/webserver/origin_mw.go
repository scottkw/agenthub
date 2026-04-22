// Package webserver origin-allowlist middleware (Phase 88, SEC-06).
//
// requireAllowedOrigin closes the cross-site WebSocket hijacking vector
// by strictly comparing the browser's Origin header to the server's own
// current serving URL (ws.BaseURL()). Any mismatch — including a missing
// Origin header (D-05) — is rejected with 403 "forbidden" BEFORE any
// capability work runs.
//
// The check is byte-for-byte exact (D-03). No case-folding, no
// port-stripping, no normalization. Browsers emit a canonical Origin and
// ws.BaseURL() already returns the canonical form; any divergence is
// treated as a hijack attempt and the user re-opens the link from the
// share panel.
//
// Composition (D-10): outermost->innermost is basicAuth (local only) ->
// requireAllowedOrigin -> requireCapability -> handleWSSRelay. The
// middleware sits OUTSIDE requireCapability so Origin rejection
// short-circuits the HMAC signature verification work.
//
// Fail-closed on listener-not-ready: if ws.BaseURL() returns "" (listener
// nil — theoretically unreachable because handlers don't run until
// Start() succeeds, but documented defensively), the middleware rejects
// rather than silently passing (CLAUDE.md "Silent Fallbacks Forbidden").
package webserver

import "net/http"

// requireAllowedOrigin gates the WebSocket upgrade route on an exact
// match between r.Header.Get("Origin") and ws.BaseURL(). Missing or
// mismatched Origin -> 403 "forbidden".
func (ws *WebServer) requireAllowedOrigin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// D-05: missing Origin is always rejected. Browsers always
			// send Origin on WS upgrade; the tailnet-facing WSS route is
			// not a non-browser interface.
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		allowed := ws.BaseURL()
		if allowed == "" || origin != allowed {
			// D-03: strict byte-for-byte match.
			// Pitfall 1: BaseURL() == "" means listener-not-ready — fail
			// closed, never silently pass.
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// allowedOrigins returns the strict single-element allowlist used by
// both requireAllowedOrigin and websocket.AcceptOptions.OriginPatterns
// (D-12 belt-and-suspenders). Returns nil when ws.BaseURL() is empty so
// the library layer does not silently accept based on its Host-header
// default — any request reaching the library layer with an empty
// allowlist has already bypassed the middleware, which is a bug we want
// to fail, not paper over.
func (ws *WebServer) allowedOrigins() []string {
	base := ws.BaseURL()
	if base == "" {
		return nil
	}
	return []string{base}
}
