package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/scottkw/agenthub/internal/files"
)

// TestServer_FilesAPI_CORS verifies the Phase 120.1 hotfix: relay /api/files/*
// responds to CORS preflight (OPTIONS) from loopback / Wails / Tauri origins
// and emits Access-Control-Allow-Origin on successful responses.
//
// Before this fix the Wails webview's HTTP fetch to 127.0.0.1:<relayPort> was
// blocked by the browser because the preflight returned 405 with no
// Access-Control-* headers. This test guards the wiring so a regression that
// removed withCORS / handleFilesPreflight would surface immediately.
func TestServer_FilesAPI_CORS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sb, _ := files.NewSandbox(dir)
	const sid = "cors-session"
	fh := files.NewHandler(func(id string) (*files.Sandbox, error) {
		if id == sid {
			return sb, nil
		}
		return nil, os.ErrNotExist
	})
	mgr := NewHubManager()
	t.Cleanup(mgr.Shutdown)
	srv := httptest.NewServer(NewServer(mgr, nil, fh))
	t.Cleanup(srv.Close)

	t.Run("preflight from macOS wails://wails returns 204 + ACAO", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/files/list", nil)
		req.Header.Set("Origin", "wails://wails")
		req.Header.Set("Access-Control-Request-Method", "GET")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("preflight: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusNoContent {
			t.Fatalf("status: got %d want 204", res.StatusCode)
		}
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != "wails://wails" {
			t.Errorf("ACAO: got %q want %q", got, "wails://wails")
		}
	})

	t.Run("preflight from wails.localhost returns 204 + ACAO", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/files/list", nil)
		req.Header.Set("Origin", "http://wails.localhost")
		req.Header.Set("Access-Control-Request-Method", "GET")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("preflight: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusNoContent {
			t.Fatalf("status: got %d want 204", res.StatusCode)
		}
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != "http://wails.localhost" {
			t.Errorf("ACAO: got %q want %q", got, "http://wails.localhost")
		}
		if got := res.Header.Get("Access-Control-Allow-Methods"); got == "" {
			t.Error("Access-Control-Allow-Methods missing")
		}
	})

	t.Run("preflight from disallowed origin returns 403", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/files/list", nil)
		req.Header.Set("Origin", "http://evil.example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("preflight: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Errorf("status: got %d want 403", res.StatusCode)
		}
	})

	t.Run("GET with allowed Origin echoes ACAO", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/files/list?session="+sid+"&path=.", nil)
		req.Header.Set("Origin", "http://wails.localhost")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(res.Body)
			t.Fatalf("status: got %d want 200, body=%s", res.StatusCode, string(body))
		}
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != "http://wails.localhost" {
			t.Errorf("ACAO: got %q want %q", got, "http://wails.localhost")
		}
	})

	t.Run("GET with no Origin (curl-style) still works", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/files/list?session="+sid+"&path=.", nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("status: got %d want 200", res.StatusCode)
		}
		// No Origin header → no ACAO header (correct).
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("ACAO unexpected: %q", got)
		}
	})
}
