// Package webserver unit tests for cspHeaders middleware (Phase 89, SEC-08).
//
// The eight tests below cover the full behavioral specification of
// cspHeaders (D-09, D-10, D-06, D-16, D-13) using httptest.Recorder to
// inspect headers without running a real HTTP server:
//
//  1. TestCSPHeaders_HeaderSet               — CSP header is non-empty
//  2. TestCSPHeaders_RequiredTokens          — D-09 policy tokens all present
//  3. TestCSPHeaders_NoUnsafeTokens          — D-06 no unsafe-* keywords
//  4. TestCSPHeaders_WSSComposition          — D-10 connect-src wss:// derived from BaseURL
//  5. TestCSPHeaders_NoWildcardOutsideDataScheme — no bare * wildcard
//  6. TestCSPHeaders_CacheControlNoStore     — D-16 Cache-Control: no-store
//  7. TestCSPHeaders_CallsNext               — D-13 always delegates on success
//  8. TestCSPHeaders_FailsClosedOnEmptyBaseURL — fail-closed when listener not ready
package webserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCSPHeaders_HeaderSet asserts that wrapping a stub handler and invoking
// via httptest produces a response whose Content-Security-Policy header is
// non-empty and the 200 status propagates through.
func TestCSPHeaders_HeaderSet(t *testing.T) {
	ws, _ := testServer(t)
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Errorf("Content-Security-Policy header must not be empty (Phase 89 SEC-08)")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

// TestCSPHeaders_RequiredTokens asserts that the CSP header contains ALL
// required directive tokens specified by D-09.
func TestCSPHeaders_RequiredTokens(t *testing.T) {
	ws, _ := testServer(t)
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	required := []string{
		"default-src 'none'",
		"script-src 'self'",
		"style-src 'self'",
		"img-src 'self' data:",
		"font-src 'self'",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}
	for _, token := range required {
		if !strings.Contains(csp, token) {
			t.Errorf("CSP missing required token %q (Phase 89 D-09): %s", token, csp)
		}
	}
}

// TestCSPHeaders_NoUnsafeTokens asserts that the CSP header forbids unsafe
// tokens on script-src and forbids 'unsafe-eval' / 'unsafe-hashes' globally.
//
// D-09 amended 2026-04-22: style-src is permitted to carry 'unsafe-inline'
// because xterm.js injects runtime <style> elements (e2e finding). script-src
// stays strict — Finding 4's CDN-injection class remains blocked.
func TestCSPHeaders_NoUnsafeTokens(t *testing.T) {
	ws, _ := testServer(t)
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")

	// 'unsafe-eval' moved to TestCSPHeaders_NoUnsafeEvalToken_TokenAware
	// (Phase 96 IMG-03 Amendment 2): the substring approach used here would
	// falsely match 'wasm-unsafe-eval' which is intentionally permitted in
	// script-src. The token-aware test enforces the same defense correctly.
	globallyForbidden := []string{
		"'unsafe-hashes'",
	}
	for _, token := range globallyForbidden {
		if strings.Contains(csp, token) {
			t.Errorf("CSP must not contain %q anywhere (Phase 89 D-06): %s", token, csp)
		}
	}

	// script-src must not carry 'unsafe-inline'. Extract the script-src clause
	// and assert the keyword is absent from it. The clause ends at ';'.
	idx := strings.Index(csp, "script-src ")
	if idx < 0 {
		t.Fatalf("CSP missing script-src directive: %s", csp)
	}
	end := strings.Index(csp[idx:], ";")
	if end < 0 {
		end = len(csp) - idx
	}
	scriptSrc := csp[idx : idx+end]
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Errorf("script-src must not carry 'unsafe-inline' (Phase 89 D-06 script half): %s", scriptSrc)
	}
}

// TestCSPHeaders_WSSComposition asserts that the connect-src clause is
// composed correctly from ws.BaseURL() per D-10.
func TestCSPHeaders_WSSComposition(t *testing.T) {
	ws, _ := testServer(t)
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	expected := "connect-src 'self' wss://" + strings.TrimPrefix(ws.BaseURL(), "https://")
	if !strings.Contains(csp, expected) {
		t.Errorf("CSP missing computed connect-src %q (Phase 89 D-10): got %s", expected, csp)
	}
}

// TestCSPHeaders_NoWildcardOutsideDataScheme asserts that the CSP header
// contains no bare * wildcard source tokens.
func TestCSPHeaders_NoWildcardOutsideDataScheme(t *testing.T) {
	ws, _ := testServer(t)
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if strings.Contains(csp, " *") {
		t.Errorf("CSP must not contain a bare wildcard source \" *\" (Phase 89 D-09): %s", csp)
	}
	if strings.Contains(csp, "'*'") {
		t.Errorf("CSP must not contain a quoted wildcard \"'*'\" (Phase 89 D-09): %s", csp)
	}
}

// TestCSPHeaders_CacheControlNoStore asserts that the middleware sets
// Cache-Control: no-store alongside the CSP header (Research Pitfall 3, D-16).
func TestCSPHeaders_CacheControlNoStore(t *testing.T) {
	ws, _ := testServer(t)
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if cc != "no-store" {
		t.Errorf("Cache-Control must be exactly \"no-store\" (Phase 89 D-16): got %q", cc)
	}
}

// TestCSPHeaders_CallsNext asserts that the inner handler IS called when
// BaseURL is valid and the 200 status propagates (D-13: always delegates on
// success, unlike gate-style middleware).
func TestCSPHeaders_CallsNext(t *testing.T) {
	ws, _ := testServer(t)
	called := false
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if !called {
		t.Errorf("inner handler must be called when BaseURL is valid (Phase 89 D-13)")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

// TestCSPHeaders_FailsClosedOnEmptyBaseURL asserts that when ws.BaseURL()
// returns "" (listener not ready), the middleware returns HTTP 500 and the
// inner handler is NOT called (CLAUDE.md "Silent Fallbacks Forbidden").
//
// Uses a zero-value WebServer — sync.RWMutex is safe at zero value, and
// ws.listener will be nil, causing BaseURL() to return "".
func TestCSPHeaders_FailsClosedOnEmptyBaseURL(t *testing.T) {
	var ws WebServer
	// Precondition: BaseURL() must return "" for this test to be meaningful.
	if got := ws.BaseURL(); got != "" {
		t.Fatalf("expected BaseURL() == \"\" for zero-value WebServer, got %q", got)
	}

	called := false
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	req := httptest.NewRequest("GET", "/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when BaseURL is empty (Phase 89 D-13 fail-closed): got %d", rec.Code)
	}
	if called {
		t.Errorf("inner handler must NOT be called when BaseURL is empty (Phase 89 D-13 fail-closed)")
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp != "" {
		t.Errorf("CSP header must not be set on fail path (Phase 89 D-13): got %q", csp)
	}
}

// TestCSPHeaders_HasWasmUnsafeEval — Plan 96-03 implements.
// Phase 96 IMG-03 CSP amendment: script-src must include
// 'wasm-unsafe-eval' to permit @xterm/addon-image's SIXEL/IIP
// WASM decoder to instantiate (per 96-RESEARCH §"Mandatory
// Pre-Phase CSP Audit Finding 2"; CSP3 §6.3 directive).
func TestCSPHeaders_HasWasmUnsafeEval(t *testing.T) {
	// Phase 96 IMG-03: script-src must include 'wasm-unsafe-eval' to
	// permit @xterm/addon-image's WASM decoder to instantiate.
	// See csp_mw.go package comment Amendment 2 for full rationale.
	ws, _ := testServer(t)
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header is empty")
	}

	// 'wasm-unsafe-eval' is unique enough that strings.Contains is safe
	// for THIS direction (it must appear; the false-match concern is
	// for the inverse check in TestCSPHeaders_NoUnsafeEvalToken_TokenAware).
	if !strings.Contains(csp, "'wasm-unsafe-eval'") {
		t.Errorf("CSP missing 'wasm-unsafe-eval' (Phase 96 IMG-03 Amendment 2): %s", csp)
	}

	// Defensive: ensure the token lives inside the script-src clause,
	// not anywhere else (CSP directives are semicolon-delimited).
	scriptSrcClause := extractDirective(csp, "script-src")
	if !strings.Contains(scriptSrcClause, "'wasm-unsafe-eval'") {
		t.Errorf("'wasm-unsafe-eval' must appear inside script-src directive, not elsewhere: %s", csp)
	}
}

// TestCSPHeaders_NoUnsafeEvalToken_TokenAware — Plan 96-03 tightens
// the existing TestCSPHeaders_NoUnsafeTokens substring check to be
// token-aware. After Plan 96-03, the script-src directive contains
// 'wasm-unsafe-eval'; a naive strings.Contains(csp, "'unsafe-eval'")
// check would FALSELY MATCH because 'unsafe-eval' is a substring of
// 'wasm-unsafe-eval'. This test asserts that the script-src clause,
// tokenized on whitespace, does NOT contain a bare "'unsafe-eval'"
// token (only "'wasm-unsafe-eval'" — distinct CSP3 source expression).
//
// Per 96-PATTERNS.md §`internal/webserver/csp_mw_test.go` Adapt block:
// pull script-src clause; tokenize; compare per-token equality.
func TestCSPHeaders_NoUnsafeEvalToken_TokenAware(t *testing.T) {
	// Phase 96 IMG-03 defense regression: after Amendment 2, the script-src
	// clause contains 'wasm-unsafe-eval'. A naive strings.Contains check
	// for "'unsafe-eval'" would FALSELY MATCH because 'unsafe-eval' is a
	// substring of 'wasm-unsafe-eval'. This test extracts the script-src
	// clause, tokenizes on whitespace, and asserts no token equals exactly
	// "'unsafe-eval'" — the BARE form that would broaden script execution
	// to JS eval() + new Function(). Per 96-PATTERNS.md §`internal/
	// webserver/csp_mw_test.go` "Adapt — critical defense regression".
	ws, _ := testServer(t)
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	scriptSrcClause := extractDirective(csp, "script-src")
	if scriptSrcClause == "" {
		t.Fatal("script-src directive missing from CSP")
	}

	for _, token := range strings.Fields(scriptSrcClause) {
		if token == "'unsafe-eval'" {
			t.Errorf("script-src contains bare 'unsafe-eval' token (Phase 89 D-06 forbidden — must be 'wasm-unsafe-eval' only): %s", csp)
		}
	}

	// Sanity check: 'wasm-unsafe-eval' IS expected (Amendment 2). If it
	// is absent, TestCSPHeaders_HasWasmUnsafeEval will already fail —
	// we don't duplicate that assertion here.
}

// extractDirective returns the value portion of a single CSP directive
// (e.g. extractDirective("default-src 'none'; script-src 'self'", "script-src")
// returns "'self'"). Returns empty string if the directive is not present.
func extractDirective(csp, name string) string {
	for _, clause := range strings.Split(csp, ";") {
		clause = strings.TrimSpace(clause)
		if strings.HasPrefix(clause, name+" ") || clause == name {
			return strings.TrimSpace(strings.TrimPrefix(clause, name))
		}
	}
	return ""
}
