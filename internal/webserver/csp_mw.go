// Package webserver CSP header middleware (Phase 89, SEC-08).
//
// cspHeaders implements the Content-Security-Policy response header setter
// for all HTML-serving routes (terminal.html, dashboard.html, join.html).
// It is wired by Plan 04 — this file only defines the method.
//
// Policy specification (D-09, amended 2026-04-22 after Phase 89 e2e finding):
//
//	default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline';
//	connect-src 'self' wss://<host>; img-src 'self' data:;
//	font-src 'self'; base-uri 'none'; form-action 'self';
//	frame-ancestors 'none'
//
// Amendment rationale: xterm.js injects <style> elements at runtime (cursor,
// selection, theme hooks) via document.createElement('style'). CSP3 classifies
// these as style-src-elem, which style-src 'self' blocks. Chromium e2e test
// TestBrowserCSP_TerminalNoViolations surfaced 12 violations on /sessions/{id}.
// User disposition: allow 'unsafe-inline' for style-src only. script-src
// remains strict ('self') — the Finding 4 CDN class of attack is unchanged.
//
// Amendment 2 (Phase 96 IMG-03, 2026-05-07 after pre-phase CSP audit):
//
//	default-src 'none'; script-src 'self' 'wasm-unsafe-eval';
//	style-src 'self' 'unsafe-inline'; connect-src 'self' wss://<host>;
//	img-src 'self' data:; font-src 'self'; base-uri 'none';
//	form-action 'self'; frame-ancestors 'none'
//
// Amendment rationale: @xterm/addon-image embeds a SIXEL decoder and
// an iTerm2 IIP base64 decoder as base64-encoded WebAssembly bytecode
// (~10 KB raw), instantiated synchronously on the main thread via
// WebAssembly.instantiate / new WebAssembly.Module / new WebAssembly.
// Instance. The pre-phase audit confirmed (see RESEARCH §"Mandatory Pre-Phase CSP Audit Finding 2",
// documented in .planning/phases/96-image-addon-csp-audit/96-RESEARCH.md):
// 0 occurrences of `new Worker(`, 0 occurrences of `eval(`, and
// 6 cumulative occurrences of WASM bootstrap APIs across the addon's
// .js + .mjs bundles. CSP3 §6.3 requires 'wasm-unsafe-eval' to permit
// these instantiations under any non-default script-src directive.
//
// Defense-in-depth distinction: 'wasm-unsafe-eval' is NARROW — it
// permits only WebAssembly compilation/instantiation. It does NOT
// enable JS `eval()`, `new Function()`, or `setTimeout(string)` —
// those still require the broader 'unsafe-eval' source expression,
// which this codebase explicitly forbids via the token-aware
// TestCSPHeaders_NoUnsafeEvalToken_TokenAware regression guard
// (csp_mw_test.go). The two source expressions share a substring but
// are distinct CSP3 tokens; the regression guard tokenizes on
// whitespace within the script-src clause to distinguish them.
//
// Browser support (CSP3 'wasm-unsafe-eval' source expression):
//
//	Chrome (Chromium):  102+   (May 2022)
//	Firefox:            102+   (June 2022)
//	Safari (WebKit):    16.0+  (September 2022)
//	iPad Safari:        16.0+  (September 2022)
//
// All four browsers in Phase 99's supported matrix support
// 'wasm-unsafe-eval' with multi-year headroom; no fallback strategy
// required.
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
		// Phase 96 IMG-03 Amendment 2: 'wasm-unsafe-eval' permits the
		// @xterm/addon-image SIXEL/IIP WASM decoder to instantiate
		// (CSP3 §6.3 governs WebAssembly.compile/instantiate/Module/Instance).
		// 'wasm-unsafe-eval' is NARROW — it does NOT enable JS eval() or
		// new Function() (those still require the broader 'unsafe-eval', which
		// remains forbidden by csp_mw_test.go). See package comment Amendment 2.
		b.WriteString("script-src 'self' 'wasm-unsafe-eval'; ")
		b.WriteString("style-src 'self' 'unsafe-inline'; ")
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
