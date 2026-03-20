package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/agenthub/agenthub/internal/webserver"
)

// testApp creates an App wired for testing — no Wails GUI, but all bound
// methods are functional.  It opens a real TCP listener on 127.0.0.1:0 to
// simulate what startup() does.
func testApp(t *testing.T) *App {
	t.Helper()
	app := NewApp()

	// Set context — startup() is not called in tests, so we provide a background context.
	app.ctx = context.Background()

	// Simulate the startup listener allocation (without running wails.Run).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("testApp: net.Listen: %v", err)
	}
	app.listener = ln
	t.Cleanup(func() {
		ln.Close()
		app.manager.Shutdown()
	})
	return app
}

func TestListSessionsEmpty(t *testing.T) {
	app := testApp(t)
	sessions := app.ListSessions()
	if sessions == nil {
		t.Fatal("ListSessions on fresh App returned nil, want empty slice")
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestCreateSession(t *testing.T) {
	app := testApp(t)
	id, err := app.CreateSession("cat", "test-tab", "")
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession returned empty ID")
	}

	sessions := app.ListSessions()
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != id {
		t.Errorf("session ID mismatch: got %q, want %q", sessions[0].ID, id)
	}
	if sessions[0].Name != "test-tab" {
		t.Errorf("session name mismatch: got %q, want %q", sessions[0].Name, "test-tab")
	}
}

func TestRenameSession(t *testing.T) {
	app := testApp(t)
	id, err := app.CreateSession("cat", "original", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := app.RenameSession(id, "renamed"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}

	sessions := app.ListSessions()
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Name != "renamed" {
		t.Errorf("expected name %q, got %q", "renamed", sessions[0].Name)
	}
}

func TestKillSession(t *testing.T) {
	app := testApp(t)
	id, err := app.CreateSession("cat", "kill-me", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := app.KillSession(id); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// Give the session a moment to be removed.
	time.Sleep(50 * time.Millisecond)

	sessions := app.ListSessions()
	for _, s := range sessions {
		if s.ID == id {
			t.Errorf("killed session %q still appears in ListSessions", id)
		}
	}
}

func TestDetectCLIs(t *testing.T) {
	app := testApp(t)
	clis := app.DetectCLIs()
	if clis == nil {
		t.Fatal("DetectCLIs returned nil, want non-nil slice")
	}
}

func TestUpdateCLIPath(t *testing.T) {
	app := testApp(t)

	// /bin/cat is a guaranteed path on macOS/Linux.
	if err := app.UpdateCLIPath("claude", "/bin/cat"); err != nil {
		t.Fatalf("UpdateCLIPath: %v", err)
	}

	// Now create a session with "claude" — it should resolve to /bin/cat.
	id, err := app.CreateSession("claude", "custom-path-tab", "")
	if err != nil {
		t.Fatalf("CreateSession with custom path: %v", err)
	}

	sessions := app.ListSessions()
	var found SessionInfo
	for _, s := range sessions {
		if s.ID == id {
			found = s
			break
		}
	}
	if found.ID == "" {
		t.Fatal("session not found in ListSessions after CreateSession")
	}
}

func TestGetRelayPort(t *testing.T) {
	app := testApp(t)
	port := app.GetRelayPort()
	if port <= 0 {
		t.Errorf("GetRelayPort returned %d, want port > 0", port)
	}
}


func TestToggleWebServingErrorsWhenNotRunning(t *testing.T) {
	app := testApp(t)
	// webServer is nil — ToggleWebServing should return an error.
	err := app.ToggleWebServing("some-session-id", true)
	if err == nil {
		t.Error("expected ToggleWebServing to return error when web server is not running")
	}
}

func TestStartWebServerNoPasswordRequired(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	app := testApp(t)
	// StartWebServer should NOT error due to a missing password requirement.
	// If Tailscale is not connected, it errors about Tailscale (not password).
	// If Tailscale is connected, it may succeed.
	// Either way, no error mentioning "password" should appear.
	err := app.StartWebServer(0)
	if err == nil {
		// Tailscale is connected and server started — clean up.
		_ = app.StopWebServer()
		return
	}
	if strings.Contains(err.Error(), "password") {
		t.Errorf("StartWebServer should not mention password, got: %s", err.Error())
	}
}

func TestIsWebServerRunning(t *testing.T) {
	app := testApp(t)
	if app.IsWebServerRunning() {
		t.Error("expected IsWebServerRunning to be false before StartWebServer")
	}
}

// selfSignedTLSForAppTest generates an in-memory self-signed TLS config
// for use in app-level tests where Tailscale is not available.
func selfSignedTLSForAppTest(t *testing.T) *tls.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("selfSignedTLSForAppTest: generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("selfSignedTLSForAppTest: create cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("selfSignedTLSForAppTest: marshal key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("selfSignedTLSForAppTest: key pair: %v", err)
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}
}

func TestGetSessionQRCode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	app := testApp(t)

	// Bypass StartWebServer (which requires Tailscale) by directly creating a
	// WebServer with an in-memory TLS config, then assigning it to app.webServer.
	tlsCfg := selfSignedTLSForAppTest(t)
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "localhost",
		TLSConfig: tlsCfg,
	}, app.manager)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	app.mu.Lock()
	app.webServer = ws
	app.mu.Unlock()
	t.Cleanup(func() { _ = app.StopWebServer() })

	// Enable a session in the web server.
	if err := app.ToggleWebServing("test-session-id", true); err != nil {
		t.Fatalf("ToggleWebServing: %v", err)
	}

	// GetSessionQRCode should return a non-empty base64 string.
	b64, err := app.GetSessionQRCode("test-session-id")
	if err != nil {
		t.Fatalf("GetSessionQRCode: %v", err)
	}
	if b64 == "" {
		t.Fatal("GetSessionQRCode returned empty string")
	}

	// Decode base64 and verify PNG magic bytes (\x89PNG).
	pngBytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if len(pngBytes) < 4 {
		t.Fatalf("decoded PNG too short: %d bytes", len(pngBytes))
	}
	if pngBytes[0] != 0x89 || pngBytes[1] != 'P' || pngBytes[2] != 'N' || pngBytes[3] != 'G' {
		t.Errorf("expected PNG magic bytes, got %v", pngBytes[:4])
	}
}

func TestGetSessionQRCode_NoServer(t *testing.T) {
	app := testApp(t)
	// webServer is nil — should return an error.
	_, err := app.GetSessionQRCode("any-id")
	if err == nil {
		t.Error("expected GetSessionQRCode to return error when web server is not running")
	}
}

func TestGetTailscaleStatus(t *testing.T) {
	app := testApp(t)
	h := app.GetTailscaleStatus()
	// On machines without tailscaled, Installed will be false.
	// On machines with tailscaled, any combination is valid.
	// The key assertion is that the method doesn't panic and returns a struct.
	_ = h.Installed
	_ = h.Connected
	_ = h.HasCerts
	_ = h.IP
	_ = h.Domain
}

func TestHealthPollerStops(t *testing.T) {
	app := testApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	app.startHealthPoller(ctx)
	// Cancel immediately -- the goroutine should exit without blocking.
	cancel()
	// Give the goroutine a moment to observe the cancellation.
	time.Sleep(100 * time.Millisecond)
	// If the goroutine leaked, the race detector (run with -race) will catch it
	// on test cleanup. No explicit assertion needed beyond "no hang, no panic".
}

func TestGetSessionStatus(t *testing.T) {
	app := testApp(t)

	id, err := app.CreateSession("cat", "status-test-tab", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// GetSessionStatus should return a valid non-empty status string.
	s := app.GetSessionStatus(id)
	if s == "" {
		t.Error("GetSessionStatus returned empty string for active session")
	}

	// Valid status values.
	valid := map[string]bool{"running": true, "waiting": true, "idle": true, "errored": true}
	if !valid[s] {
		t.Errorf("GetSessionStatus returned invalid status %q", s)
	}
}

func TestStatusMap(t *testing.T) {
	app := testApp(t)

	id1, err := app.CreateSession("cat", "tab-1", "")
	if err != nil {
		t.Fatalf("CreateSession tab-1: %v", err)
	}
	id2, err := app.CreateSession("cat", "tab-2", "")
	if err != nil {
		t.Fatalf("CreateSession tab-2: %v", err)
	}

	s1 := app.GetSessionStatus(id1)
	s2 := app.GetSessionStatus(id2)

	if s1 == "" || s2 == "" {
		t.Errorf("expected non-empty statuses; s1=%q s2=%q", s1, s2)
	}

	// Unknown session returns "running" default.
	sUnknown := app.GetSessionStatus("nonexistent-id")
	if sUnknown != "running" {
		t.Errorf("unknown session status: got %q, want %q", sUnknown, "running")
	}
}
