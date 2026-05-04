// Package webserver integration tests for /assets/* routes (Phase 89, SEC-07).
//
// Covers:
//   - /assets/xterm/ → vendor/xterm fs.Sub mount (D-01/D-02/D-14)
//   - /assets/ → assets/ fs.Sub mount (first-party JS/CSS from Plan 02)
//   - Cache-Control: no-store on both mounts (D-16)
//   - Public tier — no capability required (D-15)
//   - 404 for missing files; no directory listing
package webserver

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestAssets_XtermJSServedFromEmbed asserts that GET /assets/xterm/xterm.js
// returns 200 with a JS Content-Type and a body > 400 KB, proving the
// /assets/xterm/ → vendor/xterm fs.Sub mount works end-to-end (Phase 89 D-14).
func TestAssets_XtermJSServedFromEmbed(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/assets/xterm/xterm.js")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/javascript") && !strings.HasPrefix(ct, "application/javascript") {
		t.Errorf("expected JS Content-Type, got %q", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(body) < 400000 { // xterm.js is ~476 KB — served from vendor/xterm via /assets/xterm/ mount
		t.Errorf("expected xterm.js body > 400 KB (Phase 89 D-02 vendored + D-14 /assets/xterm/ mount), got %d bytes", len(body))
	}
}

// TestAssets_XtermCSSServedFromEmbed asserts that GET /assets/xterm/xterm.css
// returns 200 with a CSS Content-Type and a non-trivially-sized body (Phase 89 D-14).
func TestAssets_XtermCSSServedFromEmbed(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/assets/xterm/xterm.css")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/css") {
		t.Errorf("expected CSS Content-Type, got %q", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(body) < 6000 { // xterm.css is ~7 KB
		t.Errorf("expected xterm.css body >= 6 KB (Phase 89 D-14), got %d bytes", len(body))
	}
}

// TestAssets_FirstPartyJS asserts that GET /assets/terminal.js returns 200
// with a JS Content-Type and body containing "initTerminal", proving the
// /assets/ → assets/ fs.Sub mount serves the Plan 02 extracted IIFE (Phase 89 D-14).
func TestAssets_FirstPartyJS(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/assets/terminal.js")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/javascript") && !strings.HasPrefix(ct, "application/javascript") {
		t.Errorf("expected JS Content-Type, got %q", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !strings.Contains(string(body), "initTerminal") {
		t.Errorf("expected terminal.js body to contain \"initTerminal\" (Phase 89 D-14 /assets/ mount serving Plan 02 extracted IIFE)")
	}
}

// TestAssets_FirstPartyCSS asserts that GET /assets/dashboard.css returns 200
// with a CSS Content-Type (Phase 89 D-14).
func TestAssets_FirstPartyCSS(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/assets/dashboard.css")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/css") {
		t.Errorf("expected CSS Content-Type, got %q", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(body) < 1000 {
		t.Errorf("expected dashboard.css body >= 1 KB (Phase 89 D-14), got %d bytes", len(body))
	}
}

// TestAssets_CacheControlNoStore asserts that both /assets/xterm/xterm.js
// and /assets/terminal.js return Cache-Control: no-store, covering both
// the xtermFS and assetsFS mounts (Phase 89 D-16).
func TestAssets_CacheControlNoStore(t *testing.T) {
	ws, client := testServer(t)

	paths := []string{
		"/assets/xterm/xterm.js",
		"/assets/terminal.js",
	}
	for _, path := range paths {
		resp, err := client.Get(ws.BaseURL() + path)
		if err != nil {
			t.Fatalf("client.Get %s: %v", path, err)
		}
		resp.Body.Close()
		cc := resp.Header.Get("Cache-Control")
		if cc != "no-store" {
			t.Errorf("expected Cache-Control no-store on %s (Phase 89 D-16), got %q", path, resp.Header.Get("Cache-Control"))
		}
	}
}

// TestAssets_PublicNoCapNeeded asserts that GET /assets/xterm/xterm.js returns
// 200 without any capability token — the /assets/* routes are public-tier per
// Phase 89 D-15 and must not require authentication.
func TestAssets_PublicNoCapNeeded(t *testing.T) {
	ws, client := testServer(t)
	// Do NOT call ws.SetSigningKey — prove the route works regardless of signing-key state.
	// Make NO Authorization header, NO ?cap= query param.
	resp, err := client.Get(ws.BaseURL() + "/assets/xterm/xterm.js")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for public /assets/xterm/xterm.js (Phase 89 D-15: public tier, no cap needed), got %d", resp.StatusCode)
	}
}

// TestAssets_NotFound asserts that GET /assets/no-such-file.js returns 404.
func TestAssets_NotFound(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/assets/no-such-file.js")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for missing asset, got %d", resp.StatusCode)
	}
}

// TestAssets_NoDirectoryListing asserts that GET /assets/ does NOT return a
// directory listing — http.FileServerFS should return 404 or 403 for
// directory index requests when no index.html is present.
func TestAssets_NoDirectoryListing(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/assets/")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	// Both 404 and 403 are acceptable — we just want no directory listing.
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 404 or 403 for /assets/ directory (no listing), got %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s := string(body)
		if strings.Contains(s, "<pre>") || strings.Contains(s, "<html>") {
			t.Errorf("/assets/ returned a directory listing (contains <pre> or <html>): %q", s[:min(len(s), 200)])
		}
	}
}

// min returns the smaller of a and b (stdlib min not available in Go < 1.21;
// included here for safety across build targets).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestAssets_VendoredAddons asserts that the three vendored xterm addon
// bundles shipped by Plan 93-02 (addon-webgl, addon-unicode11, addon-clipboard)
// are accessible via the /assets/xterm/addons/ URL prefix and served with a
// JS Content-Type. Phase 93 PLUG-04 — the web terminal page's <script src>
// tags depend on these paths resolving.
func TestAssets_VendoredAddons(t *testing.T) {
	ws, client := testServer(t)
	for _, path := range []string{
		"/assets/xterm/addons/addon-webgl.js",
		"/assets/xterm/addons/addon-unicode11.js",
		"/assets/xterm/addons/addon-clipboard.js",
	} {
		resp, err := client.Get(ws.BaseURL() + path)
		if err != nil {
			t.Fatalf("client.Get %s: %v", path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "javascript") {
			t.Errorf("GET %s: expected javascript content-type, got %q", path, ct)
		}
		resp.Body.Close()
	}
}
