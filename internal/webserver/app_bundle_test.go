// Phase 120-04 Task 4 — /app/ route integration tests.
//
// Covers:
//   - Default state (no SetStaticAppFS call) → 503 with informative body
//   - With a wired fs.FS containing index.html → 200, HTML body, SPA fallback
//   - Unknown path under /app/ → SPA fallback returns index.html
//   - Cache-Control: no-store on the index.html response
//
// These tests run on every `go test ./internal/webserver/...` invocation —
// they DON'T require the real frontend/dist embed because the test builds a
// synthetic fs.FS via fstest.MapFS. The wailsassets-tagged
// TestAppBundle_ServesRealReactBundle below complements this by exercising
// the real embed when present.

package webserver

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

const fakeIndexHTML = `<!doctype html><html><head><title>AgentHub</title></head><body><div id="root"></div></body></html>`

func TestAppBundle_503WithoutFS(t *testing.T) {
	ws, client := testServer(t)
	// Intentionally do NOT call ws.SetStaticAppFS — covers the dev fallback.
	_ = ws

	resp, err := client.Get(ws.BaseURL() + "/app/")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without SetStaticAppFS, got %d", resp.StatusCode)
	}
}

func TestAppBundle_ServesIndex(t *testing.T) {
	ws, client := testServer(t)
	ws.SetStaticAppFS(fstest.MapFS{
		"index.html":           {Data: []byte(fakeIndexHTML)},
		"assets/index-abc.js":  {Data: []byte("// minified js")},
		"assets/index-abc.css": {Data: []byte("/* css */")},
	})

	resp, err := client.Get(ws.BaseURL() + "/app/")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html Content-Type, got %q", ct)
	}
	cc := resp.Header.Get("Cache-Control")
	if cc != "no-store" {
		t.Errorf("expected Cache-Control: no-store, got %q", cc)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !strings.Contains(string(body), "<!doctype html>") {
		t.Errorf("expected index.html body, got %q", string(body))
	}
	if !strings.Contains(string(body), `<div id="root">`) {
		t.Errorf("expected React root div in body, got %q", string(body))
	}
}

func TestAppBundle_SPAFallbackForUnknownPath(t *testing.T) {
	ws, client := testServer(t)
	ws.SetStaticAppFS(fstest.MapFS{
		"index.html": {Data: []byte(fakeIndexHTML)},
	})

	// A client-side React route (e.g. /app/files/abc) is not a real file in
	// the bundle but must serve index.html so the SPA router can resolve it.
	resp, err := client.Get(ws.BaseURL() + "/app/files/abc")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (SPA fallback), got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<!doctype html>") {
		t.Errorf("expected index.html SPA fallback, got %q", string(body))
	}
}

func TestAppBundle_ServesRealAsset(t *testing.T) {
	ws, client := testServer(t)
	ws.SetStaticAppFS(fstest.MapFS{
		"index.html":          {Data: []byte(fakeIndexHTML)},
		"assets/index-abc.js": {Data: []byte("// minified js")},
	})

	resp, err := client.Get(ws.BaseURL() + "/app/assets/index-abc.js")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for real asset, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "minified js") {
		t.Errorf("expected JS body, got %q", string(body))
	}
}
