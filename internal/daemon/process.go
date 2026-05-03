package daemon

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/scottkw/agenthub/internal/webserver"
)

// BuildVersion is set by the main package at startup. The daemon embeds it in
// the health response so that EnsureDaemon can detect stale daemons from older
// builds and restart them.
var BuildVersion string

// upgradeToTailscale polls Tailscale health every 15s. When Tailscale becomes
// fully healthy (Connected + HasCerts + IP), it restarts the web server in
// Tailscale mode. Runs continuously — does not exit on first upgrade, so a
// later Tailscale outage that drops back to LAN can still be re-upgraded
// when Tailscale recovers. Exits only when ctx is cancelled.
//
// Polling cadence: 5s. Lower than the original 15s because the user-visible
// gap between "Tailscale up" and "AgentHub URL switches" was reported as
// long enough that users manually-restarted the web server thinking the
// upgrade was stuck (P-1).
func upgradeToTailscale(ctx context.Context, api *API) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			h := webserver.CheckHealth(checkCtx)
			cancel()
			if !(h.Connected && h.HasCerts && h.IP != "") {
				continue // Tailscale not ready
			}
			// Skip the restart if we're already serving in Tailscale mode on
			// the same IP — avoids redundant restart churn and dropped client
			// connections every tick once we've upgraded.
			if api.WebServerMode() == "tailscale" && api.WebServerBindIP() == h.IP {
				continue
			}
			if err := api.RestartWebServer(h.IP, 7443, h.Domain, "tailscale", ""); err != nil {
				fmt.Fprintf(os.Stderr, "daemon: upgrade to tailscale: %v\n", err)
				continue // retry on next tick
			}
			fmt.Fprintf(os.Stderr, "daemon: web server upgraded to tailscale on %s\n", h.IP)
		case <-ctx.Done():
			return
		}
	}
}

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

	// Phase 87 (D-04): bootstrap the capability signing key + join-code
	// manager BEFORE any web server is started. A missing key here is fatal
	// — without it, requireCapability would 401 every request and the daemon
	// is effectively a brick for web access.
	if err := api.BootstrapCapabilityState(); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: bootstrap capability state: %v\n", err)
		return
	}

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

	// Generate local-mode password once per daemon lifetime (NET-01).
	localPassword, err := generateLocalPassword()
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		return // abort — cannot serve without a secure password
	}
	api.SetLocalPassword(localPassword)

	// Auto-start web server: Tailscale mode if available, local mode fallback (NET-01, SERVE-01).
	// Diagnostic logging records the precise health-check outcome so a Finder/launchd
	// invocation that lands in an unexpected mode can be diagnosed post-hoc (P-6).
	{
		ctx5s, cancel := context.WithTimeout(ctx, 5*time.Second)
		h := webserver.CheckHealth(ctx5s)
		cancel()
		fmt.Fprintf(os.Stderr,
			"daemon: tailscale health: binaryFound=%v daemonUp=%v connected=%v hasCerts=%v ip=%q domain=%q\n",
			h.BinaryFound, h.DaemonUp, h.Connected, h.HasCerts, h.IP, h.Domain)
		if h.Connected && h.HasCerts && h.IP != "" {
			if err := api.AutoStartWebServer(h.IP, 7443, h.Domain, "tailscale", ""); err != nil {
				fmt.Fprintf(os.Stderr, "daemon: auto-start web server: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "daemon: web server auto-started on %s (tailscale)\n", h.IP)
			}
		} else {
			// Local mode fallback
			lanIP, lanErr := webserver.GetLANIP()
			if lanErr != nil {
				fmt.Fprintf(os.Stderr, "daemon: local mode: no LAN IP: %v (web server NOT started — UI will show no share URLs)\n", lanErr)
			} else {
				if err := api.AutoStartWebServer(lanIP, 7443, "", "local", localPassword); err != nil {
					fmt.Fprintf(os.Stderr, "daemon: auto-start web server (local): %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "daemon: web server auto-started on %s (local mode)\n", lanIP)
				}
			}
			// Launch background upgrader: watches for Tailscale to become healthy
			// and upgrades the web server from local to Tailscale mode automatically.
			go upgradeToTailscale(ctx, api)
		}
	}

	<-ctx.Done()

	fmt.Fprintf(os.Stderr, "daemon: shutting down\n")
	_ = api.Stop()
	engine.Manager().Shutdown()
}

// generateLocalPassword generates a cryptographically random 16-byte password
// encoded as base64url (~22 characters). Called once per daemon lifetime.
func generateLocalPassword() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate local password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil // ~22 chars
}

// EnsureDaemon checks if the daemon is reachable on socketPath. If not, it
// spawns a detached daemon subprocess using the current binary with "daemon"
// as the first argument, then polls until the daemon is ready (max 3 seconds).
func EnsureDaemon(socketPath string) error {
	client := NewDaemonClient(socketPath)
	needsSpawn := true
	if err := client.Health(); err == nil {
		// Daemon is reachable — check version compatibility first.
		// Production builds (non-"dev") compare versions; a stale daemon from
		// an older release won't have newly added API routes (e.g. start-minimized).
		if BuildVersion != "" && BuildVersion != "dev" {
			if ver := client.GetDaemonVersion(); ver != BuildVersion {
				fmt.Fprintf(os.Stderr, "EnsureDaemon: version mismatch (daemon=%q, binary=%q), restarting\n", ver, BuildVersion)
				_ = client.ShutdownDaemon()
				time.Sleep(200 * time.Millisecond)
				_ = CleanupStaleSocket(socketPath)
				// fall through to spawn
			} else {
				needsSpawn = false
			}
		} else {
			needsSpawn = false
		}
		if !needsSpawn {
			// Same version (or dev mode) — verify relay is also ready.
			if port, relayErr := client.GetRelayPort(); relayErr == nil && port > 0 {
				return nil // fully operational
			}
			// Stale daemon without relay support. Log and continue to spawn a new one.
			// The stale process will exit on its own when the socket is replaced.
			fmt.Fprintf(os.Stderr, "EnsureDaemon: stale daemon detected (relay not ready), respawning\n")
			_ = CleanupStaleSocket(socketPath)
		}
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
