// Package webserver CSP integration tests (Phase 89, SEC-08).
//
// Verifies D-18's five-assertion suite on all three HTML routes
// (/sessions/{id}, /dashboard, /join) against a live test server,
// extending the unit tests in csp_mw_test.go to real HTTP responses.
//
// Also verifies:
//   - Cache-Control: no-store flows through on all three HTML routes (D-16)
//   - CSP header is present even on 401 responses (cspHeaders is OUTERMOST — D-13)
package webserver

import (
	"net/http"
	"strings"
	"testing"
)

// assertCSPHeaderStrict runs D-18's five assertions against a response.
// Phase 89 D-18 locks these as the canonical CSP regression set.
func assertCSPHeaderStrict(t *testing.T, resp *http.Response, wsBaseURL string, routeName string) {
	t.Helper()

	// D-18.1: header present and non-empty
	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatalf("%s: missing Content-Security-Policy header (Phase 89 D-18.1)", routeName)
	}

	// D-18.2: script-src 'self' and style-src 'self' present
	for _, tok := range []string{"script-src 'self'", "style-src 'self'"} {
		if !strings.Contains(csp, tok) {
			t.Errorf("%s: CSP missing required token %q (Phase 89 D-18.2): %s", routeName, tok, csp)
		}
	}

	// D-18.3: no unsafe tokens in script-src, no 'unsafe-eval'/'unsafe-hashes'
	// globally, no bare wildcard. style-src is permitted to carry
	// 'unsafe-inline' per the D-09 amendment (xterm runtime style injection).
	for _, tok := range []string{"'unsafe-eval'", "'unsafe-hashes'", " *", "'*'"} {
		if strings.Contains(csp, tok) {
			t.Errorf("%s: CSP must not contain %q (Phase 89 D-18.3): %s", routeName, tok, csp)
		}
	}
	// script-src must stay strict
	scIdx := strings.Index(csp, "script-src ")
	if scIdx >= 0 {
		end := strings.Index(csp[scIdx:], ";")
		if end < 0 {
			end = len(csp) - scIdx
		}
		scriptSrc := csp[scIdx : scIdx+end]
		if strings.Contains(scriptSrc, "'unsafe-inline'") {
			t.Errorf("%s: script-src must not carry 'unsafe-inline' (Phase 89 D-18.3 script half): %s", routeName, scriptSrc)
		}
	}

	// D-18.4: connect-src 'self' wss://<host>:<port> present
	expectedWSS := "connect-src 'self' wss://" + strings.TrimPrefix(wsBaseURL, "https://")
	if !strings.Contains(csp, expectedWSS) {
		t.Errorf("%s: CSP missing computed connect-src %q (Phase 89 D-18.4): %s", routeName, expectedWSS, csp)
	}

	// D-18.5: frame-ancestors 'none' and base-uri 'none'
	for _, tok := range []string{"frame-ancestors 'none'", "base-uri 'none'"} {
		if !strings.Contains(csp, tok) {
			t.Errorf("%s: CSP missing clickjacking/base-uri defense %q (Phase 89 D-18.5): %s", routeName, tok, csp)
		}
	}
}

// TestCSPHeaderStrict_TerminalPage asserts D-18's five assertions on
// GET /sessions/{id} with a valid capability token.
//
// Phase 159: handleTerminalPage now issues an HTTP 302 redirect to /app/.
// The cspHeaders middleware wraps the whole handler chain, so CSP and
// Cache-Control headers are still set on the 302 response. This test uses
// a no-redirect client to observe those headers directly on the redirect.
func TestCSPHeaderStrict_TerminalPage(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-89-csp-terminal")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-89-csp-terminal", "read,write")

	// No-redirect client: reuse CA-trusting transport; stop at 3xx to
	// observe CSP/Cache-Control headers on the redirect response itself.
	noRedirect := &http.Client{
		Transport: client.Transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := noRedirect.Get(ws.BaseURL() + "/sessions/sess-89-csp-terminal?cap=" + token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	// Phase 159: route now redirects (302) to /app/.
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 for /sessions/{id} with valid cap (Phase 159), got %d", resp.StatusCode)
	}
	assertCSPHeaderStrict(t, resp, ws.BaseURL(), "/sessions/{id}")
}

// TestCSPHeaderStrict_Dashboard asserts D-18's five assertions on GET /dashboard.
func TestCSPHeaderStrict_Dashboard(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/dashboard")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /dashboard, got %d", resp.StatusCode)
	}
	assertCSPHeaderStrict(t, resp, ws.BaseURL(), "/dashboard")
}

// TestCSPHeaderStrict_Join asserts D-18's five assertions on GET /join.
func TestCSPHeaderStrict_Join(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/join")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /join, got %d", resp.StatusCode)
	}
	assertCSPHeaderStrict(t, resp, ws.BaseURL(), "/join")
}

// TestCSPHeaderStrict_CacheControl confirms Cache-Control: no-store flows
// through on all three HTML routes (Plan 03's cspHeaders sets it; this
// integration test confirms it reaches real HTTP responses — Phase 89 D-16).
//
// Phase 159: /sessions/{id} now redirects (302). cspHeaders is the outermost
// middleware and sets Cache-Control: no-store before requireCapability or
// handleTerminalPage run, so the header is still present on the 302 response.
// A no-redirect client is used for the /sessions/ path to observe that header.
func TestCSPHeaderStrict_CacheControl(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-89-csp-cc")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-89-csp-cc", "read,write")

	// No-redirect client: stops at 3xx so we observe the redirect response
	// headers (Cache-Control: no-store) set by the cspHeaders middleware.
	noRedirect := &http.Client{
		Transport: client.Transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// /dashboard and /join still return 200; use the default client.
	for _, p := range []string{"/dashboard", "/join"} {
		resp, err := client.Get(ws.BaseURL() + p)
		if err != nil {
			t.Fatalf("Get %s: %v", p, err)
		}
		cc := resp.Header.Get("Cache-Control")
		resp.Body.Close()
		if cc != "no-store" {
			t.Errorf("%s: expected Cache-Control no-store (Phase 89 D-16), got %q", p, cc)
		}
	}

	// /sessions/{id} returns 302 (Phase 159); use no-redirect client.
	sessPath := "/sessions/sess-89-csp-cc?cap=" + token
	resp, err := noRedirect.Get(ws.BaseURL() + sessPath)
	if err != nil {
		t.Fatalf("Get %s: %v", sessPath, err)
	}
	cc := resp.Header.Get("Cache-Control")
	resp.Body.Close()
	if cc != "no-store" {
		t.Errorf("%s: expected Cache-Control no-store (Phase 89 D-16), got %q", sessPath, cc)
	}
}

// TestCSPHeaderStrict_OnAuthFailure asserts that GET /sessions/{id} with an
// invalid cap returns 401 AND still carries the CSP header. This locks in
// the cspHeaders-OUTSIDE-requireCapability composition from Task 2 (Phase 89 D-13):
// the CSP header is set before requireCapability runs, so even a 401 is protected.
func TestCSPHeaderStrict_OnAuthFailure(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-89-csp-401")
	ws.SetSigningKey(capTestKey)
	resp, err := client.Get(ws.BaseURL() + "/sessions/sess-89-csp-401?cap=invalid")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid cap, got %d", resp.StatusCode)
	}
	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Errorf("expected CSP header on 401 response (cspHeaders must wrap OUTSIDE requireCapability — Phase 89 D-13)")
	}
}
