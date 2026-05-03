package relay

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestServer_LoopbackOrigin127Accepted verifies that the relay accepts a WebSocket
// upgrade from a loopback client presenting Origin: http://127.0.0.1:<port>.
// This is the primary loopback allowlist acceptance test (Phase 88 D-09).
func TestServer_LoopbackOrigin127Accepted(t *testing.T) {
	srv, _, _, _, sessionID := setupTestServer(t)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{}
	headers.Set("Origin", "http://127.0.0.1:"+u.Port())
	wsURL := "ws://" + u.Host + "/sessions/" + sessionID + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		t.Fatalf("websocket.Dial with loopback 127 Origin: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
}

// TestServer_LoopbackOriginLocalhostAccepted verifies that the relay accepts a
// WebSocket upgrade from a loopback client presenting Origin: http://localhost:<port>.
// The allowlist includes both "localhost" and "127.0.0.1" forms (Phase 88 D-09).
func TestServer_LoopbackOriginLocalhostAccepted(t *testing.T) {
	srv, _, _, _, sessionID := setupTestServer(t)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{}
	headers.Set("Origin", "http://localhost:"+u.Port())
	wsURL := "ws://" + u.Host + "/sessions/" + sessionID + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		t.Fatalf("websocket.Dial with loopback localhost Origin: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
}

// TestServer_CrossSiteOriginRejected verifies that the relay rejects a WebSocket
// upgrade whose Origin does not match the loopback allowlist (Phase 88 SC-1 relay half).
// Uses raw HTTP (not websocket.Dial) to inspect the status code directly — websocket.Dial
// converts non-101 responses into opaque dial errors.
func TestServer_CrossSiteOriginRejected(t *testing.T) {
	srv, _, _, _, sessionID := setupTestServer(t)
	wsURL := srv.URL + "/sessions/" + sessionID + "/ws"
	req, err := http.NewRequest("GET", wsURL, nil)
	if err != nil {
		t.Fatal(err)
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
		t.Error("relay accepted cross-site Origin — Phase 88 D-09 loopback allowlist not enforced")
	}
}

// TestLoopbackOriginPatterns_DerivesPortFromHost is a unit test for the
// loopbackOriginPatterns helper. Verifies the 8-element allowlist when host
// includes a port (Wails origins + IP/loopback origins) and the 4-element
// Wails-only fallback when host is empty / malformed (Phase 88 T-88-14,
// extended for Wails GUI origin support).
func TestLoopbackOriginPatterns_DerivesPortFromHost(t *testing.T) {
	wailsBase := []string{
		"wails://wails",                // production macOS / Linux
		"wails://wails.localhost:*",    // dev macOS / Linux
		"http://wails.localhost",       // production Windows
		"http://wails.localhost:*",     // dev Windows
	}

	got := loopbackOriginPatterns("127.0.0.1:54321")
	want := append(append([]string{}, wailsBase...),
		"http://localhost:54321",
		"http://127.0.0.1:54321",
		"https://localhost:54321",
		"https://127.0.0.1:54321",
	)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loopbackOriginPatterns(127.0.0.1:54321) = %v, want %v", got, want)
	}
	// Empty host -> wails-only (Wails GUI must still reach the relay across
	// the wails:// → 127.0.0.1 origin boundary).
	if got := loopbackOriginPatterns(""); !reflect.DeepEqual(got, wailsBase) {
		t.Errorf("empty host: got %v, want wails-only %v", got, wailsBase)
	}
	// Malformed host (no port) -> wails-only fallback
	if got := loopbackOriginPatterns("no-port-here"); !reflect.DeepEqual(got, wailsBase) {
		t.Errorf("malformed host: got %v, want wails-only %v", got, wailsBase)
	}
}

// TestServer_WailsProductionOriginAcceptedDarwin verifies that the relay
// accepts the actual production macOS / Linux Wails webview Origin —
// `wails://wails` (host is "wails", NOT "wails.localhost"). The
// `.localhost` suffix is only appended by Wails in dev mode (see
// frontend.go:109 in wails v2.12.0). Without this pattern, the desktop
// GUI is locked out of its own backend on every production install.
func TestServer_WailsProductionOriginAcceptedDarwin(t *testing.T) {
	srv, _, _, _, sessionID := setupTestServer(t)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{}
	headers.Set("Origin", "wails://wails")
	wsURL := "ws://" + u.Host + "/sessions/" + sessionID + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		t.Fatalf("websocket.Dial with Wails production darwin Origin: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
}

// TestServer_WailsProductionOriginAcceptedWindows verifies the production
// Windows Wails webview Origin (http://wails.localhost — which DOES include
// .localhost, only the macOS/Linux scheme uses bare "wails").
func TestServer_WailsProductionOriginAcceptedWindows(t *testing.T) {
	srv, _, _, _, sessionID := setupTestServer(t)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{}
	headers.Set("Origin", "http://wails.localhost")
	wsURL := "ws://" + u.Host + "/sessions/" + sessionID + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		t.Fatalf("websocket.Dial with Wails production windows Origin: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
}

// TestServer_WailsDevOriginAccepted verifies that the relay accepts a
// WebSocket upgrade from a Wails dev-mode webview, whose Origin includes the
// Vite HMR port (wails://wails.localhost:34115).
func TestServer_WailsDevOriginAccepted(t *testing.T) {
	srv, _, _, _, sessionID := setupTestServer(t)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{}
	headers.Set("Origin", "wails://wails.localhost:34115")
	wsURL := "ws://" + u.Host + "/sessions/" + sessionID + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		t.Fatalf("websocket.Dial with Wails dev Origin: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
}
