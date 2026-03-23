package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// RunDaemon is the daemon's main entry point. It creates a SessionEngine,
// starts the API (with relay server inside), and blocks until SIGTERM or SIGINT.
// This function is called from main.go when os.Args[1] == "daemon".
func RunDaemon() {
	socketPath := DefaultSocketPath()
	if err := CleanupStaleSocket(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		os.Exit(1)
	}

	engine := NewSessionEngine()
	api := NewAPI(engine)

	// Start the relay TCP server inside the daemon.
	relayPort, err := api.StartRelay()
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: start relay: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "daemon: relay listening on port %d\n", relayPort)

	if err := api.Start(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: start api: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "daemon: listening on %s\n", socketPath)

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	<-ctx.Done()

	fmt.Fprintf(os.Stderr, "daemon: shutting down\n")
	_ = api.Stop()
	engine.Manager().Shutdown()
}

// EnsureDaemon checks if the daemon is reachable on socketPath. If not, it
// spawns a detached daemon subprocess using the current binary with "daemon"
// as the first argument, then polls until the daemon is ready (max 3 seconds).
func EnsureDaemon(socketPath string) error {
	client := NewDaemonClient(socketPath)
	if err := client.Health(); err == nil {
		return nil // already running
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("EnsureDaemon: locate binary: %w", err)
	}

	if err := startDetachedDaemon(exe); err != nil {
		return fmt.Errorf("EnsureDaemon: start daemon: %w", err)
	}

	// Poll until daemon is ready (max 3 seconds).
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.Health(); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("EnsureDaemon: daemon did not start within 3s")
}
