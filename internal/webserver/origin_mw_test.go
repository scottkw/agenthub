// Package webserver unit tests for requireAllowedOrigin middleware and
// allowedOrigins helper (Phase 88, SEC-06).
package webserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/relay"
)

// TestRequireAllowedOrigin_MatchingOriginPasses asserts that an Origin header
// exactly matching ws.BaseURL() causes the inner handler to be called.
func TestRequireAllowedOrigin_MatchingOriginPasses(t *testing.T) {
	ws, _ := testServer(t)
	called := false
	handler := ws.requireAllowedOrigin(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x/ws", nil)
	req.Header.Set("Origin", ws.BaseURL())
	rec := httptest.NewRecorder()
	handler(rec, req)
	if !called {
		t.Fatal("expected inner handler to be called for matching Origin")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for matching Origin, got %d", rec.Code)
	}
}

// TestRequireAllowedOrigin_MismatchRejected asserts that a cross-site Origin
// is rejected with 403.
func TestRequireAllowedOrigin_MismatchRejected(t *testing.T) {
	ws, _ := testServer(t)
	called := false
	handler := ws.requireAllowedOrigin(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x/ws", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if called {
		t.Fatal("expected inner handler NOT to be called for mismatched Origin")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for mismatched Origin, got %d", rec.Code)
	}
	if !strings.HasPrefix(rec.Body.String(), "forbidden") {
		t.Errorf("expected body starting with \"forbidden\", got %q", rec.Body.String())
	}
}

// TestRequireAllowedOrigin_MissingOriginRejected asserts that a request with
// no Origin header is rejected with 403 (D-05).
func TestRequireAllowedOrigin_MissingOriginRejected(t *testing.T) {
	ws, _ := testServer(t)
	called := false
	handler := ws.requireAllowedOrigin(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x/ws", nil)
	// No Origin header set.
	rec := httptest.NewRecorder()
	handler(rec, req)
	if called {
		t.Fatal("expected inner handler NOT to be called for missing Origin")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for missing Origin, got %d", rec.Code)
	}
	if !strings.HasPrefix(rec.Body.String(), "forbidden") {
		t.Errorf("expected body starting with \"forbidden\", got %q", rec.Body.String())
	}
}

// TestRequireAllowedOrigin_StrictCaseSensitive asserts that an Origin with an
// uppercase scheme is rejected even though the library's case-insensitive check
// would accept it (D-03: strict byte-for-byte; middleware is tighter than library).
func TestRequireAllowedOrigin_StrictCaseSensitive(t *testing.T) {
	ws, _ := testServer(t)
	called := false
	handler := ws.requireAllowedOrigin(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x/ws", nil)
	// Uppercase scheme — ws.BaseURL() returns "https://...", so "HTTPS://..."
	// is a different byte sequence and must be rejected.
	uppercased := strings.ToUpper(ws.BaseURL())
	req.Header.Set("Origin", uppercased)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if called {
		t.Fatal("expected inner handler NOT to be called for case-mismatched Origin")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for case-mismatched Origin, got %d", rec.Code)
	}
	if !strings.HasPrefix(rec.Body.String(), "forbidden") {
		t.Errorf("expected body starting with \"forbidden\", got %q", rec.Body.String())
	}
}

// TestRequireAllowedOrigin_OriginNullRejected asserts that the literal string
// "null" (emitted by sandboxed iframes and file:// contexts) is rejected (Pitfall 2).
func TestRequireAllowedOrigin_OriginNullRejected(t *testing.T) {
	ws, _ := testServer(t)
	called := false
	handler := ws.requireAllowedOrigin(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x/ws", nil)
	req.Header.Set("Origin", "null")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if called {
		t.Fatal("expected inner handler NOT to be called for Origin: null")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Origin: null, got %d", rec.Code)
	}
	if !strings.HasPrefix(rec.Body.String(), "forbidden") {
		t.Errorf("expected body starting with \"forbidden\", got %q", rec.Body.String())
	}
}

// TestRequireAllowedOrigin_FailClosedWhenListenerNotReady asserts that when
// ws.BaseURL() returns "" (listener not yet started), any Origin value results
// in 403 — fail-closed, never silently pass (Pitfall 1; CLAUDE.md "Silent
// Fallbacks Forbidden").
func TestRequireAllowedOrigin_FailClosedWhenListenerNotReady(t *testing.T) {
	// Construct a bare WebServer without calling Start() — listener stays nil.
	cfg := Config{
		BindIP: "127.0.0.1",
		FQDN:   "127.0.0.1",
		Mode:   "tailscale",
	}
	ws, err := NewWebServer(cfg, relay.NewHubManager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	// DO NOT call ws.Start() — BaseURL() must return "".
	if got := ws.BaseURL(); got != "" {
		t.Fatalf("expected BaseURL() == \"\" before Start(), got %q", got)
	}

	called := false
	handler := ws.requireAllowedOrigin(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	req := httptest.NewRequest("GET", "/sessions/x/ws", nil)
	req.Header.Set("Origin", "https://anything.example")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if called {
		t.Fatal("expected inner handler NOT to be called when listener is not ready")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 when listener not ready, got %d", rec.Code)
	}
	if !strings.HasPrefix(rec.Body.String(), "forbidden") {
		t.Errorf("expected body starting with \"forbidden\", got %q", rec.Body.String())
	}
}

// TestAllowedOrigins_ReturnsBaseURLSingleton asserts that allowedOrigins()
// returns []string{ws.BaseURL()} when the listener is up, and nil when it is not.
func TestAllowedOrigins_ReturnsBaseURLSingleton(t *testing.T) {
	// Case 1: listener up — allowedOrigins() == []string{BaseURL()}.
	wsUp, _ := testServer(t)
	origins := wsUp.allowedOrigins()
	base := wsUp.BaseURL()
	if base == "" {
		t.Fatal("expected BaseURL() to be non-empty after Start()")
	}
	if len(origins) != 1 || origins[0] != base {
		t.Errorf("allowedOrigins() = %v, want %v", origins, []string{base})
	}

	// Case 2: listener not yet started — allowedOrigins() == nil.
	cfg := Config{
		BindIP: "127.0.0.1",
		FQDN:   "127.0.0.1",
		Mode:   "tailscale",
	}
	wsDown, err := NewWebServer(cfg, relay.NewHubManager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	// DO NOT call wsDown.Start().
	nilOrigins := wsDown.allowedOrigins()
	if nilOrigins != nil {
		t.Errorf("allowedOrigins() before Start() = %v, want nil", nilOrigins)
	}
}
