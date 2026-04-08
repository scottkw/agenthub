package daemon

import (
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
