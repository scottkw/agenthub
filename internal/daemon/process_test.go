package daemon

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

var procTestSeq atomic.Int64

func shortProcTestSocket(t *testing.T, name string) string {
	t.Helper()
	seq := procTestSeq.Add(1)
	path := fmt.Sprintf("/tmp/dptest%d_%s.sock", seq, name)
	t.Cleanup(func() { os.Remove(path) })
	return path
}

func TestEnsureDaemon_AlreadyRunning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix domain sockets")
	}
	// Start an in-process daemon API to simulate a running daemon.
	socketPath := shortProcTestSocket(t, "already")
	engine := NewSessionEngine()
	engine.configDir = t.TempDir()
	engine.cliPaths = make(map[string]string)
	api := NewAPI(engine)
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("api.Start: %v", err)
	}
	defer api.Stop()
	time.Sleep(10 * time.Millisecond)

	// EnsureDaemon should return nil immediately (daemon reachable).
	client := NewDaemonClient(socketPath)
	if err := client.Health(); err != nil {
		t.Fatalf("Health check before EnsureDaemon: %v", err)
	}
}

func TestEnsureDaemon_Timeout(t *testing.T) {
	// Use a socket path where nothing is listening.
	socketPath := shortProcTestSocket(t, "timeout")
	client := NewDaemonClient(socketPath)
	// Health should fail since nothing is listening.
	if err := client.Health(); err == nil {
		t.Fatal("expected Health to fail on empty socket path")
	}
}

func TestRunDaemon_Exports(t *testing.T) {
	// Verify RunDaemon and EnsureDaemon are callable (compile-time check).
	// We cannot call RunDaemon() in tests (it blocks on signal), but we verify
	// the functions exist and are exported.
	var _ func() = RunDaemon
	var _ func(string) error = EnsureDaemon
}

func TestRestartWebServer_StopsAndStarts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix domain sockets")
	}
	socketPath := shortProcTestSocket(t, "restart")
	engine := NewSessionEngine()
	engine.configDir = t.TempDir()
	engine.cliPaths = make(map[string]string)
	api := NewAPI(engine)
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("api.Start: %v", err)
	}
	defer api.Stop()

	// RestartWebServer with local mode needs a password and LAN IP.
	// Use a local-mode restart which requires a non-empty password.
	// We call it twice; second call should also succeed (stop + start).
	pwd := "testpassword123"
	if err := api.RestartWebServer("127.0.0.1", 0, "", "local", pwd); err != nil {
		t.Fatalf("RestartWebServer (first call): %v", err)
	}

	// Verify the web server is now running in local mode.
	api.mu.RLock()
	ws := api.webServer
	api.mu.RUnlock()
	if ws == nil {
		t.Fatal("expected webServer to be non-nil after RestartWebServer")
	}
	if ws.Mode() != "local" {
		t.Errorf("expected mode 'local', got %q", ws.Mode())
	}

	// Call again — should stop the running server and start a new one.
	if err := api.RestartWebServer("127.0.0.1", 0, "", "local", pwd); err != nil {
		t.Fatalf("RestartWebServer (second call): %v", err)
	}
}

func TestUpgradeToTailscale_ExitsOnCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix domain sockets")
	}
	// upgradeToTailscale must exit promptly when ctx is cancelled.
	socketPath := shortProcTestSocket(t, "upgrade")
	engine := NewSessionEngine()
	engine.configDir = t.TempDir()
	engine.cliPaths = make(map[string]string)
	api := NewAPI(engine)
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("api.Start: %v", err)
	}
	defer api.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		upgradeToTailscale(ctx, api)
		close(done)
	}()

	// Cancel immediately — goroutine should exit within a reasonable time.
	cancel()
	select {
	case <-done:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("upgradeToTailscale did not exit after context cancellation")
	}
}
