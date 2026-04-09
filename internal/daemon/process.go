package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/scottkw/agenthub/internal/webserver"
)

// RunDaemon is the daemon's main entry point. It creates a signal context and
// delegates to runDaemonCore. Called from main.go when os.Args[1] == "daemon".
func RunDaemon() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	runDaemonCore(ctx)
}

// runDaemonCore is the blocking core of the daemon. It creates a SessionEngine,
// starts the API (with relay server inside), and blocks until ctx is cancelled.
// Using a context parameter allows it to be driven by either signal handling
// (RunDaemon) or a service manager (daemonSvc).
func runDaemonCore(ctx context.Context) {
	AugmentServicePath() // Must be before NewSessionEngine / any exec.LookPath
	socketPath := DefaultSocketPath()
	if err := CleanupStaleSocket(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		return
	}

	engine := NewSessionEngine()
	api := NewAPI(engine)

	// Start the relay TCP server inside the daemon.
	relayPort, err := api.StartRelay()
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: start relay: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "daemon: relay listening on port %d\n", relayPort)

	if err := api.Start(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: start api: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "daemon: listening on %s\n", socketPath)

	// Auto-start web server if Tailscale is connected (SERVE-01).
	{
		ctx5s, cancel := context.WithTimeout(ctx, 5*time.Second)
		h := webserver.CheckHealth(ctx5s)
		cancel()
		if h.Connected && h.HasCerts && h.IP != "" {
			if err := api.AutoStartWebServer(h.IP, 7443, h.Domain); err != nil {
				fmt.Fprintf(os.Stderr, "daemon: auto-start web server: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "daemon: web server auto-started on %s\n", h.IP)
			}
		} else {
			fmt.Fprintf(os.Stderr, "daemon: Tailscale not ready, skipping web server auto-start\n")
		}
	}

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
		// Daemon is reachable — verify relay is also ready.
		if port, relayErr := client.GetRelayPort(); relayErr == nil && port > 0 {
			return nil // fully operational
		}
		// Stale daemon without relay support. Log and continue to spawn a new one.
		// The stale process will exit on its own when the socket is replaced.
		fmt.Fprintf(os.Stderr, "EnsureDaemon: stale daemon detected (relay not ready), respawning\n")
		_ = CleanupStaleSocket(socketPath)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("EnsureDaemon: locate binary: %w", err)
	}

	if err := startDetachedDaemon(exe); err != nil {
		return fmt.Errorf("EnsureDaemon: start daemon: %w", err)
	}

	// Poll until daemon is fully ready — health + relay port (max 3 seconds).
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.Health(); err == nil {
			if port, relayErr := client.GetRelayPort(); relayErr == nil && port > 0 {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("EnsureDaemon: daemon did not become ready within 3s")
}
