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

	// D-18.3: no unsafe tokens, no bare wildcard
	for _, tok := range []string{"'unsafe-inline'", "'unsafe-eval'", " *", "'*'"} {
		if strings.Contains(csp, tok) {
			t.Errorf("%s: CSP must not contain %q (Phase 89 D-18.3): %s", routeName, tok, csp)
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
func TestCSPHeaderStrict_TerminalPage(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-89-csp-terminal")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-89-csp-terminal", "read,write")
	resp, err := client.Get(ws.BaseURL() + "/sessions/sess-89-csp-terminal?cap=" + token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /sessions/{id} with valid cap, got %d", resp.StatusCode)
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
func TestCSPHeaderStrict_CacheControl(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-89-csp-cc")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-89-csp-cc", "read,write")

	paths := []string{
		"/dashboard",
		"/join",
		"/sessions/sess-89-csp-cc?cap=" + token,
	}
	for _, p := range paths {
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
