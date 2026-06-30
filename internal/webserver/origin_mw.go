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
// match between r.Header.Get("Origin") and either ws.BaseURL() (tailnet) or
// ws.FunnelBaseURL() (Funnel — secondary check active only when Funnel is
// enabled; fail-closed when FunnelBaseURL()==""). Missing or mismatched
// Origin -> 403 "forbidden".
//
// Dual-origin extension (Phase 165, FNL-04 / T-165-01):
//
//   - Primary check: tailnet BaseURL (ws.BaseURL()) — unchanged behaviour.
//   - Secondary check: Funnel base URL (ws.FunnelBaseURL()) — exact byte match;
//     only consulted when FunnelBaseURL() is non-empty; never a prefix/substring
//     widen (T-165-07). Secondary branch is inert (fail-closed) until
//     165-02 daemon endpoint calls ws.EnableFunnel.
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
		tailnetURL := ws.BaseURL()
		if tailnetURL == "" {
			// Pitfall 1: BaseURL() == "" means listener-not-ready — fail
			// closed, never silently pass.
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if origin == tailnetURL {
			next(w, r)
			return
		}
		// Secondary: Funnel origin (exact match; empty string when inactive — fail closed).
		if funnelURL := ws.FunnelBaseURL(); funnelURL != "" && origin == funnelURL {
			next(w, r)
			return
		}
		// D-03: strict byte-for-byte match required for all origins.
		http.Error(w, "forbidden", http.StatusForbidden)
	}
}

// allowedOrigins returns the strict allowlist used by both requireAllowedOrigin
// and websocket.AcceptOptions.OriginPatterns (D-12 belt-and-suspenders). Returns
// nil when ws.BaseURL() is empty so the library layer does not silently accept
// based on its Host-header default — any request reaching the library layer with
// an empty allowlist has already bypassed the middleware, which is a bug we want
// to fail, not paper over.
//
// When Funnel is active (FunnelBaseURL() != ""), the Funnel URL is appended to
// the list so the websocket library's secondary origin check also allows Funnel
// guests (FNL-04 / T-165-01).
func (ws *WebServer) allowedOrigins() []string {
	base := ws.BaseURL()
	if base == "" {
		return nil
	}
	origins := []string{base}
	if funnelBase := ws.FunnelBaseURL(); funnelBase != "" {
		origins = append(origins, funnelBase)
	}
	return origins
}
