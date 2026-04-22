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

// TestCSPHeaders_NoUnsafeTokens asserts that the CSP header contains NONE of
// the forbidden unsafe-* keywords (D-06).
func TestCSPHeaders_NoUnsafeTokens(t *testing.T) {
	ws, _ := testServer(t)
	handler := ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	forbidden := []string{
		"'unsafe-inline'",
		"'unsafe-eval'",
		"'unsafe-hashes'",
	}
	for _, token := range forbidden {
		if strings.Contains(csp, token) {
			t.Errorf("CSP must not contain %q (Phase 89 D-06): %s", token, csp)
		}
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
