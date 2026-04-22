// Package webserver CSP header middleware (Phase 89, SEC-08).
//
// cspHeaders implements the Content-Security-Policy response header setter
// for all HTML-serving routes (terminal.html, dashboard.html, join.html).
// It is wired by Plan 04 — this file only defines the method.
//
// Policy specification (D-09):
//
//	default-src 'none'; script-src 'self'; style-src 'self';
//	connect-src 'self' wss://<host>; img-src 'self' data:;
//	font-src 'self'; base-uri 'none'; form-action 'self';
//	frame-ancestors 'none'
//
// Per-request composition (D-10): the wss:// origin is derived from
// ws.BaseURL() on every request so it tracks the listener address
// automatically — including the random-port fallback that Phase 87/88
// already use for the same reason.
//
// No CSP violation reporting (D-11): report-uri and report-to directives are
// intentionally absent. The policy is enforced silently; violations surface
// in browser DevTools only. No report-uri endpoint is exposed.
//
// Middleware shape (D-13): func(http.HandlerFunc) http.HandlerFunc, matching
// requireAllowedOrigin in origin_mw.go. Unlike the gate-style middlewares,
// cspHeaders ALWAYS delegates to next when BaseURL is non-empty — it sets
// response headers and passes control through.
//
// Fail-closed on empty BaseURL: if ws.BaseURL() returns "" (listener not
// yet ready — theoretically unreachable because handlers don't run until
// Start() succeeds), the middleware responds HTTP 500 rather than silently
// passing without a CSP header (CLAUDE.md "Silent Fallbacks Forbidden").
//
// Cache-Control: no-store is set alongside the CSP header (Research
// Pitfall 3 guidance) so HTML pages are refetched on every request. This
// ensures CSP changes propagate immediately and aligns HTML caching with
// the /assets/ no-store policy added in Plan 04.
package webserver

import (
	"net/http"
	"strings"
)

// cspHeaders sets the Content-Security-Policy and Cache-Control: no-store
// response headers, then delegates to next. When ws.BaseURL() returns ""
// (listener not ready), it responds HTTP 500 and does NOT delegate.
func (ws *WebServer) cspHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := ws.BaseURL()
		if base == "" {
			// Fail-closed: listener not ready. Theoretically unreachable
			// because handlers don't run until Start() succeeds, but
			// documented defensively (CLAUDE.md Silent Fallbacks Forbidden).
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		wssOrigin := "wss://" + strings.TrimPrefix(base, "https://")
		var b strings.Builder
		b.Grow(256)
		b.WriteString("default-src 'none'; ")
		b.WriteString("script-src 'self'; ")
		b.WriteString("style-src 'self'; ")
		b.WriteString("connect-src 'self' ")
		b.WriteString(wssOrigin)
		b.WriteString("; ")
		b.WriteString("img-src 'self' data:; ")
		b.WriteString("font-src 'self'; ")
		b.WriteString("base-uri 'none'; ")
		b.WriteString("form-action 'self'; ")
		b.WriteString("frame-ancestors 'none'")
		w.Header().Set("Content-Security-Policy", b.String())
		w.Header().Set("Cache-Control", "no-store")
		next(w, r)
	}
}
