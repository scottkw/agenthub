// Package webserver integration tests for the Origin allowlist gate (Phase 88,
// SEC-06). Each test exercises one sampling point from the validation
// architecture in 88-VALIDATION.md.
//
// All tests that need to observe a 403 (rather than a 101 WebSocket upgrade)
// use http.Client.Do with raw Upgrade headers — websocket.Dial converts 403
// into an opaque dial error, making status-code assertions impossible via that
// path. Tests that expect a successful 101 use dialWebServerWS.
package webserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
)

// TestSecurity_WebSocketRejectsCrossSiteOrigin asserts SC-1: an upgrade
// request with a cross-site Origin header and an otherwise-valid capability
// is rejected with 403 "forbidden" at the requireAllowedOrigin middleware,
// before any capability work runs.
func TestSecurity_WebSocketRejectsCrossSiteOrigin(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-88-xsite")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-88-xsite", "read,write")

	// Use http.Client directly — websocket.Dial would obscure the 403.
	wsURL := ws.BaseURL() + "/sessions/sess-88-xsite/ws?cap=" + token
	req, err := http.NewRequest("GET", wsURL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for cross-site Origin, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.HasPrefix(string(body), "forbidden") {
		t.Errorf("expected body starting with \"forbidden\", got %q", string(body))
	}
}

// TestSecurity_WebSocketRejectsMissingOrigin asserts SC-3 / D-05: an upgrade
// request with no Origin header is rejected with 403 "forbidden".
func TestSecurity_WebSocketRejectsMissingOrigin(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-88-noorigin")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-88-noorigin", "read,write")

	wsURL := ws.BaseURL() + "/sessions/sess-88-noorigin/ws?cap=" + token
	req, err := http.NewRequest("GET", wsURL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	// No Origin header — browsers always set this on WS upgrade; absence is D-05.
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for missing Origin, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.HasPrefix(string(body), "forbidden") {
		t.Errorf("expected body starting with \"forbidden\", got %q", string(body))
	}
}

// TestSecurity_WebSocketAcceptsMatchingOriginTailscaleMode asserts SC-2
// (tailscale half): an upgrade with Origin == ws.BaseURL() and a valid
// capability completes the WebSocket handshake (101 Switching Protocols).
func TestSecurity_WebSocketAcceptsMatchingOriginTailscaleMode(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-88-match")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-88-match", "read,write")

	headers := http.Header{}
	headers.Set("Origin", ws.BaseURL())
	conn := dialWebServerWS(t, client, ws.BaseURL(),
		"/sessions/sess-88-match/ws?cap="+token, headers)
	if conn == nil {
		t.Fatal("expected successful upgrade for matching Origin")
	}
	// dialWebServerWS registers t.Cleanup to close the conn.
}

// TestSecurity_OriginCheckShortCircuitsBeforeCapability asserts SC-1
// short-circuit proof: a request with a cross-site Origin AND an invalid
// capability gets 403 (not 401), proving the Origin check runs and short-
// circuits before the capability middleware can emit its 401.
func TestSecurity_OriginCheckShortCircuitsBeforeCapability(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-88-sc")
	ws.SetSigningKey(capTestKey)
	// Do NOT register a grant; capability "garbage" is invalid in every respect.
	// Origin check should 403 before capability middleware can respond with 401.

	wsURL := ws.BaseURL() + "/sessions/sess-88-sc/ws?cap=garbage"
	req, err := http.NewRequest("GET", wsURL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	// Origin check runs FIRST (outer wrapper). 403 "forbidden" wins over
	// 401 "capability required" because the middleware short-circuits.
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 (Origin short-circuit), got %d — capability 401 would indicate wrong composition order", resp.StatusCode)
	}
}

// TestSecurity_OriginRejectionBodyIsForbidden asserts D-07 generic body:
// the rejection body for an Origin failure starts with "forbidden" and does
// NOT contain words that would leak the rejection reason (T-87-08 /
// T-88-05 information-disclosure defense).
func TestSecurity_OriginRejectionBodyIsForbidden(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-88-body")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-88-body", "read,write")

	wsURL := ws.BaseURL() + "/sessions/sess-88-body/ws?cap=" + token
	req, err := http.NewRequest("GET", wsURL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	if !strings.HasPrefix(s, "forbidden") {
		t.Errorf("expected body to start with \"forbidden\", got %q", s)
	}
	// Information-disclosure defense (T-88-05 / D-07): body must not leak
	// whether the failure was Origin-vs-capability-vs-whatever.
	for _, leak := range []string{"origin", "Origin", "mismatch", "evil", "capability"} {
		if strings.Contains(s, leak) {
			t.Errorf("rejection body leaks rejection reason (%q): %q", leak, s)
		}
	}
}

// TestSecurity_LibraryLayerRejectsCrossSiteOriginWhenMiddlewareBypassed
// asserts D-12 belt-and-suspenders: the library-layer OriginPatterns check
// (ws.allowedOrigins()) also rejects cross-site Origins independently of the
// middleware. This test bypasses the middleware by mounting a bare handler
// that calls websocket.Accept with the same allowlist.
func TestSecurity_LibraryLayerRejectsCrossSiteOriginWhenMiddlewareBypassed(t *testing.T) {
	ws, _ := testServer(t)

	// Bare handler — no requireAllowedOrigin middleware — uses only the
	// library-layer OriginPatterns allowlist (belt-and-suspenders, D-12).
	bareHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: ws.allowedOrigins(),
		})
		if err != nil {
			// Library rejected — expected for wrong Origin.
			return
		}
	})

	// Mount on a separate TLS server so the library's Origin-vs-Host check
	// can fire against a real TLS listener.
	srv := httptest.NewTLSServer(bareHandler)
	defer srv.Close()

	wsURL := srv.URL + "/ws"
	req, err := http.NewRequest("GET", wsURL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusSwitchingProtocols {
		t.Error("library accepted cross-site Origin — D-12 belt-and-suspenders broken")
	}
	// Library-layer rejection is any non-101 code; the exact value is
	// library-controlled (unlike the middleware's clean http.Error 403).
}
