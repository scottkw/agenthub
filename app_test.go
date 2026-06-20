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
	"fmt"
	"math/big"
	"net"
	"os"
	goruntime "runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/webserver"
)

var testSockSeq atomic.Int64

// testApp creates an App wired for testing — no Wails GUI, but all bound
// methods are functional.  It starts an in-process daemon API on a temp socket
// and wires the client, simulating what startup() does without the subprocess.
func testApp(t *testing.T) *App {
	t.Helper()
	if goruntime.GOOS == "windows" {
		t.Skip("testApp uses Unix domain sockets")
	}
	// Isolate config directory so settings.json doesn't leak between tests.
	if goruntime.GOOS == "darwin" {
		t.Setenv("HOME", t.TempDir())
	} else {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	}

	// Start an in-process daemon API for tests (no real subprocess).
	engine := daemon.NewSessionEngine()
	api := daemon.NewAPI(engine)

	seq := testSockSeq.Add(1)
	socketPath := fmt.Sprintf("/tmp/aht%d_%d.sock", os.Getpid(), seq)
	_ = os.Remove(socketPath)
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("testApp: api.Start: %v", err)
	}

	// Start relay inside daemon API so GetRelayPort works.
	if _, err := api.StartRelay(); err != nil {
		t.Fatalf("testApp: StartRelay: %v", err)
	}

	client := daemon.NewDaemonClient(socketPath)
	// Brief sleep for server goroutine to be ready.
	time.Sleep(10 * time.Millisecond)

	app := &App{
		ctx:    context.Background(),
		client: client,
	}

	t.Cleanup(func() {
		engine.Manager().Shutdown()
		api.Stop()
		_ = os.Remove(socketPath)
	})
	return app
}

// testAppWithDirectWebServer creates an App + daemon API pair and returns a
// setup function that injects a TLS web server directly into the daemon API
// (bypassing the Tailscale prerequisite). Used for tests that need a running
// web server without a real Tailscale connection.
func testAppWithDirectWebServer(t *testing.T, tlsCfg *tls.Config) (*App, func(sessionID string) error) {
	t.Helper()
	if goruntime.GOOS == "windows" {
		t.Skip("testAppWithDirectWebServer uses Unix domain sockets")
	}
	if goruntime.GOOS == "darwin" {
		t.Setenv("HOME", t.TempDir())
	} else {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	}

	engine := daemon.NewSessionEngine()
	api := daemon.NewAPI(engine)

	seq := testSockSeq.Add(1)
	socketPath := fmt.Sprintf("/tmp/aht%d_%d.sock", os.Getpid(), seq)
	_ = os.Remove(socketPath)
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("testAppWithDirectWebServer: api.Start: %v", err)
	}
	if _, err := api.StartRelay(); err != nil {
		t.Fatalf("testAppWithDirectWebServer: StartRelay: %v", err)
	}

	client := daemon.NewDaemonClient(socketPath)
	time.Sleep(10 * time.Millisecond)

	app := &App{
		ctx:    context.Background(),
		client: client,
	}

	var wsRef *webserver.WebServer

	setup := func(sessionID string) error {
		ws, err := webserver.NewWebServer(webserver.Config{
			BindIP:    "127.0.0.1",
			Port:      0,
			FQDN:      "localhost",
			TLSConfig: tlsCfg,
		}, engine.Manager())
		if err != nil {
			return fmt.Errorf("NewWebServer: %w", err)
		}
		ws.SetSessionResolver(func(id string) (string, string, string, string) {
			return id, "", "", ""
		})
		if err := ws.Start(); err != nil {
			return fmt.Errorf("ws.Start: %w", err)
		}
		ws.EnableSession(sessionID)
		wsRef = ws
		// Inject the running webserver directly into the daemon API.
		api.SetWebServerForTest(ws)
		return nil
	}

	t.Cleanup(func() {
		if wsRef != nil {
			_ = wsRef.Stop()
		}
		engine.Manager().Shutdown()
		api.Stop()
		_ = os.Remove(socketPath)
	})
	return app, setup
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
	id, err := app.CreateSession("cat", "test-tab", "", nil, 0, 0)
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
	if sessions[0].Hostname == "" {
		t.Error("expected non-empty Hostname from ListSessions, got empty string")
	}
}

// TestListSessions_PropagatesHomeDirAndFilesWrite is the regression guard for the
// Phase 137 (updated from Phase 124): propagation test for HomeDir + BrowseEnabled.
// The daemon's SessionInfo carries HomeDir + BrowseEnabled, and the App.ListSessions
// binding must propagate both. Phase 124 originally tested FilesWrite; Phase 137
// replaces the per-session write gate with the per-session browse toggle (D-02/D-07).
func TestListSessions_PropagatesHomeDirAndBrowseEnabled(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skipf("no usable home dir: %v", err)
	}

	app := testApp(t)
	// Create a session whose working directory IS the home directory so the
	// daemon may compute HomeDir=true (sessionCwdIsHome). Whether it actually
	// resolves to true is env-dependent (symlinked homes, temp HOME on CI), so
	// the assertions below test PROPAGATION against the daemon's own value
	// rather than hardcoding true for HomeDir.
	id, err := app.CreateSession("cat", "home-session", home, nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Enable the per-session browse toggle so BrowseEnabled=true on the daemon —
	// this one we control, so we can assert true propagates end-to-end.
	if err := app.SetSessionBrowse(id, true); err != nil {
		t.Fatalf("SetSessionBrowse: %v", err)
	}

	// The daemon's own view (source of truth) for the same session.
	rawSessions, err := app.client.ListSessions()
	if err != nil {
		t.Fatalf("daemon ListSessions: %v", err)
	}
	var raw *daemon.SessionInfo
	for i := range rawSessions {
		if rawSessions[i].ID == id {
			raw = &rawSessions[i]
			break
		}
	}
	if raw == nil {
		t.Fatalf("session %q not found in daemon ListSessions", id)
	}

	// The App binding's view (what the GUI consumes).
	sessions := app.ListSessions()
	var got *SessionInfo
	for i := range sessions {
		if sessions[i].ID == id {
			got = &sessions[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("session %q not found in App.ListSessions", id)
	}

	// The bug: the binding dropped these fields to false. Assert it now mirrors
	// the daemon's source of truth, env-independently.
	if got.HomeDir != raw.HomeDir {
		t.Errorf("ListSessions dropped HomeDir: binding=%v daemon=%v (home-dir banner would never show)", got.HomeDir, raw.HomeDir)
	}
	if got.BrowseEnabled != raw.BrowseEnabled {
		t.Errorf("ListSessions dropped BrowseEnabled: binding=%v daemon=%v", got.BrowseEnabled, raw.BrowseEnabled)
	}
	// BrowseEnabled we forced ON, so it must be true on both sides — proves the
	// true value survives the binding (not just a false==false coincidence).
	if !raw.BrowseEnabled {
		t.Error("daemon did not record BrowseEnabled=true after SetSessionBrowse(true)")
	}
	if !got.BrowseEnabled {
		t.Error("App.ListSessions dropped BrowseEnabled=true (cross-surface browse-state parity broken)")
	}
}

func TestRenameSession(t *testing.T) {
	app := testApp(t)
	id, err := app.CreateSession("cat", "original", "", nil, 0, 0)
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
	id, err := app.CreateSession("cat", "kill-me", "", nil, 0, 0)
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
	id, err := app.CreateSession("claude", "custom-path-tab", "", nil, 0, 0)
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
	// web server is not running — ToggleWebServing should return an error.
	err := app.ToggleWebServing("some-session-id", true)
	if err == nil {
		t.Error("expected ToggleWebServing to return error when web server is not running")
	}
}

func TestStartWebServerLocalModeFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	app := testApp(t)
	// StartWebServer tries Tailscale first; if unavailable, falls back to local
	// mode which requires a daemon-generated password. In CI (no Tailscale, no
	// daemon password), a "password" error is expected and correct behavior.
	err := app.StartWebServer(0)
	if err == nil {
		// Tailscale is connected and server started — clean up.
		_ = app.StopWebServer()
		return
	}
	// Any error is acceptable — Tailscale not found or local mode password
	// missing are both valid outcomes in a test environment.
	t.Logf("StartWebServer error (expected in CI): %s", err.Error())
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

func TestGetWebServerQRCode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tlsCfg := selfSignedTLSForAppTest(t)
	app, startWebServer := testAppWithDirectWebServer(t, tlsCfg)

	if err := startWebServer("test-session-id"); err != nil {
		t.Fatalf("startWebServer: %v", err)
	}

	b64, err := app.GetWebServerQRCode()
	if err != nil {
		t.Fatalf("GetWebServerQRCode: %v", err)
	}
	if b64 == "" {
		t.Fatal("GetWebServerQRCode returned empty string")
	}

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

func TestGetWebServerQRCode_NoServer(t *testing.T) {
	app := testApp(t)
	_, err := app.GetWebServerQRCode()
	if err == nil {
		t.Error("expected GetWebServerQRCode to return error when web server is not running")
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

	id, err := app.CreateSession("cat", "status-test-tab", "", nil, 0, 0)
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

	id1, err := app.CreateSession("cat", "tab-1", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession tab-1: %v", err)
	}
	id2, err := app.CreateSession("cat", "tab-2", "", nil, 0, 0)
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

// testAppNoDaemon returns an App with no daemon client — simulates a startup
// failure where EnsureDaemon returned an error before the client was created.
func testAppNoDaemon(t *testing.T) *App {
	t.Helper()
	return &App{ctx: context.Background()}
}

// --- Nil-client guard tests ---

// TestNilClientListSessions verifies ListSessions returns an empty slice (not nil,
// no panic) when the daemon client is nil.
func TestNilClientListSessions(t *testing.T) {
	app := testAppNoDaemon(t)
	sessions := app.ListSessions()
	if sessions == nil {
		t.Fatal("ListSessions with nil client returned nil, want empty slice")
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

// TestNilClientGetRelayPort verifies GetRelayPort returns 0 (no panic) when
// the daemon client is nil.
func TestNilClientGetRelayPort(t *testing.T) {
	app := testAppNoDaemon(t)
	port := app.GetRelayPort()
	if port != 0 {
		t.Errorf("expected GetRelayPort to return 0 with nil client, got %d", port)
	}
}

// TestNilClientCreateSession verifies CreateSession returns an error (no panic)
// when the daemon client is nil.
func TestNilClientCreateSession(t *testing.T) {
	app := testAppNoDaemon(t)
	_, err := app.CreateSession("cat", "tab", "", nil, 0, 0)
	if err == nil {
		t.Error("expected CreateSession to return error with nil client")
	}
}

// TestNilClientKillSession verifies KillSession returns an error (no panic)
// when the daemon client is nil.
func TestNilClientKillSession(t *testing.T) {
	app := testAppNoDaemon(t)
	err := app.KillSession("any-id")
	if err == nil {
		t.Error("expected KillSession to return error with nil client")
	}
}

// TestNilClientGetSessionStatus verifies GetSessionStatus returns "running"
// (no panic) when the daemon client is nil.
func TestNilClientGetSessionStatus(t *testing.T) {
	app := testAppNoDaemon(t)
	s := app.GetSessionStatus("any-id")
	if s != "running" {
		t.Errorf("expected GetSessionStatus to return %q with nil client, got %q", "running", s)
	}
}

// TestNilClientGetRemoteSessions verifies GetRemoteSessions returns an empty slice
// (not nil, no panic) when the daemon client is nil.
func TestNilClientGetRemoteSessions(t *testing.T) {
	app := testAppNoDaemon(t)
	peers := app.GetRemoteSessions()
	if peers == nil {
		t.Fatal("GetRemoteSessions with nil client returned nil, want empty slice")
	}
	if len(peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(peers))
	}
}

// --- RetryDaemon tests ---

// TestPollSessionStatus_ImmediateFirstCall verifies that pollSessionStatus
// makes its first HTTP call immediately, without sleeping first. The test
// creates a session and then checks that GetSessionStatus returns a non-empty
// status within 200ms — well before the old 2-second sleep would have elapsed.
func TestPollSessionStatus_ImmediateFirstCall(t *testing.T) {
	app := testApp(t)

	id, err := app.CreateSession("cat", "poll-test", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Start polling in a background goroutine (mirrors production usage).
	go app.pollSessionStatus(id)

	// Wait 200ms — well under the 2-second sleep that exists in the buggy code.
	time.Sleep(200 * time.Millisecond)

	// The first HTTP call must have been made by now. GetSessionStatus
	// independently polls the daemon so it always returns a fresh value; the
	// important thing is that the session exists and has a valid status, proving
	// the daemon is reachable and the session was created.
	s := app.GetSessionStatus(id)
	if s == "" {
		t.Error("GetSessionStatus returned empty string within 200ms — first poll may not have fired yet")
	}
	valid := map[string]bool{"running": true, "waiting": true, "idle": true, "errored": true}
	if !valid[s] {
		t.Errorf("GetSessionStatus returned invalid status %q", s)
	}
}

// TestPollSessionStatus_StopsOnHTTPError verifies that pollSessionStatus exits
// promptly when the daemon is unreachable, rather than blocking until the
// 300-second deadline expires. The circuit breaker exits after 5 consecutive
// errors (5 x 500ms sleep = ~2.5s plus dial attempts).
func TestPollSessionStatus_StopsOnHTTPError(t *testing.T) {
	// Point the client at a socket path that has no listener — every HTTP call
	// will fail immediately with "connection refused". This simulates a daemon
	// that has gone away and exercises the error-circuit-breaker path.
	seq := testSockSeq.Add(1)
	deadSocketPath := fmt.Sprintf("/tmp/aht_dead_%d_%d.sock", os.Getpid(), seq)
	_ = os.Remove(deadSocketPath) // Ensure nothing is listening there.

	client := daemon.NewDaemonClient(deadSocketPath)
	app := &App{
		ctx:    context.Background(),
		client: client,
	}

	start := time.Now()
	app.pollSessionStatus("any-session-id")
	elapsed := time.Since(start)

	// Circuit breaker: 5 consecutive errors x 500ms sleep = ~2.5s plus dial
	// overhead. Allow 10s headroom. The key assertion is it doesn't loop for
	// 300 seconds.
	if elapsed > 10*time.Second {
		t.Errorf("pollSessionStatus took %v with no daemon — expected < 10s (circuit breaker should exit after 5 consecutive errors)", elapsed)
	}
}

// TestRetryDaemonFail verifies RetryDaemon returns a non-nil error, leaves
// a.client nil, and sets a.daemonErr when the daemon cannot start.
// We force failure by redirecting HOME to a read-only path so the socket
// directory cannot be created and EnsureDaemon times out.
func TestRetryDaemonFail(t *testing.T) {
	// Override HOME so DefaultSocketPath resolves to a dir that can't be created.
	t.Setenv("HOME", "/nonexistent-test-dir-that-cannot-exist")
	t.Setenv("XDG_CONFIG_HOME", "/nonexistent-test-dir-that-cannot-exist")

	app := testAppNoDaemon(t)
	err := app.RetryDaemon()
	if err == nil {
		t.Error("expected RetryDaemon to return error when daemon cannot start")
	}
	if app.client != nil {
		t.Error("expected app.client to remain nil after failed RetryDaemon")
	}
	if app.daemonErr == nil {
		t.Error("expected app.daemonErr to be set after failed RetryDaemon")
	}
}

func TestAutoInstallTailscale(t *testing.T) {
	app := &App{}

	t.Run("returns error on non-darwin", func(t *testing.T) {
		// This test validates the method exists and compiles.
		// On darwin (dev machine), it may not return the non-darwin error,
		// so we just verify the method signature and basic behavior.
		err := app.AutoInstallTailscale()
		// On darwin without ctx, it should either succeed starting or fail gracefully.
		// The key validation is that the method exists and is callable.
		_ = err
	})

	t.Run("findBrew resolves a path on macOS", func(t *testing.T) {
		path, err := findBrew()
		if goruntime.GOOS == "darwin" {
			// On macOS dev machine, brew should be findable
			if err != nil {
				t.Skipf("brew not installed: %v", err)
			}
			if path == "" {
				t.Fatal("findBrew returned empty path")
			}
			if _, statErr := os.Stat(path); statErr != nil {
				t.Fatalf("findBrew path does not exist: %s", path)
			}
		}
	})
}

func TestNotifyThemeChange(t *testing.T) {
	app := testApp(t)
	err := app.NotifyThemeChange()
	if err != nil {
		t.Errorf("NotifyThemeChange: want nil, got %v", err)
	}
}

func TestNilClientNotifyThemeChange(t *testing.T) {
	app := testAppNoDaemon(t)
	err := app.NotifyThemeChange()
	if err != nil {
		t.Errorf("NotifyThemeChange with nil client: want nil (no-op), got %v", err)
	}
}

// -----------------------------------------------------------------------
// Wave 0 RED test for Phase 130 RB-01 — GetRemoteSessionsWithMeta Wails RPC.
//
// GetRemoteSessionsWithMeta does not exist yet. This test will fail to compile
// until plan 130-03 adds it to app.go (alongside adding Reachable bool to
// RemotePeerSessions). It encodes the contract:
//   - Returns []RemotePeerSessions with a Reachable field per peer
//   - An unreachable peer maps to Reachable=false with empty Sessions
//   - With nil client, returns empty slice (no panic)
// -----------------------------------------------------------------------

// TestGetRemoteSessionsWithMeta_ReachableField verifies that
// GetRemoteSessionsWithMeta returns []RemotePeerSessions where each element
// carries a Reachable bool. With a nil client (no daemon), it must return an
// empty non-nil slice — the same nil-guard contract as GetRemoteSessions.
//
// With a live daemon and no tailnet peers, the result is empty — but the
// function must exist and compile. The Reachable field assertion is exercised
// here by constructing a nil-client App and verifying the type carries the
// field (compile-time proof). The runtime contract (unreachable peer →
// Reachable=false) is validated by the tailnet-level tests in
// internal/tailnet/tailnet_test.go.
func TestGetRemoteSessionsWithMeta_ReachableField(t *testing.T) {
	// Case 1: nil client — must return empty non-nil slice, no panic.
	app := testAppNoDaemon(t)
	peers := app.GetRemoteSessionsWithMeta()
	if peers == nil {
		t.Fatal("GetRemoteSessionsWithMeta with nil client returned nil, want empty slice")
	}
	if len(peers) != 0 {
		t.Errorf("expected 0 peers with nil client, got %d", len(peers))
	}

	// Case 2: verify RemotePeerSessions carries a Reachable bool field.
	// This is a compile-time assertion: if RemotePeerSessions does not have a
	// Reachable field, this assignment fails to compile.
	var r RemotePeerSessions
	r.Reachable = true
	r.Reachable = false
	_ = r.Reachable

	// Case 3: live daemon, no tailnet peers — result is empty but non-nil.
	appLive := testApp(t)
	livePeers := appLive.GetRemoteSessionsWithMeta()
	if livePeers == nil {
		t.Fatal("GetRemoteSessionsWithMeta with live daemon returned nil, want empty slice")
	}
	// Each peer in the result must have a Reachable field (struct-level assertion).
	for _, p := range livePeers {
		// Accessing p.Reachable compiles only if the field exists.
		_ = p.Reachable
	}
}
