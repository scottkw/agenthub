package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/agenthub/agenthub/internal/daemon"
	"github.com/agenthub/agenthub/internal/pty"
	"github.com/agenthub/agenthub/internal/status"
	"github.com/agenthub/agenthub/internal/webserver"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SessionInfo is the JSON-serialisable representation of a session returned by ListSessions.
type SessionInfo struct {
	ID        string `json:"id"`
	CLI       string `json:"cli"`
	Name      string `json:"name"`
	State     string `json:"state"`
	CreatedAt string `json:"createdAt"`
	Hostname  string `json:"hostname"`
}

// App holds all application state and exposes the Wails-bound methods.
// App is a thin Wails-binding shell — all session state lives in the daemon process.
// All session operations are delegated through DaemonClient over the Unix socket.
type App struct {
	ctx       context.Context
	client    *daemon.DaemonClient // only daemon communication field; nil when startup failed
	trayInit  bool                 // true once initTray has been called
	daemonErr error                // non-nil when EnsureDaemon failed at startup
	quitting  bool                 // true when tray Quit was clicked; lets beforeClose allow exit
}

// NewApp creates a new App without starting any subsystems.
func NewApp() *App {
	return &App{}
}

// domReady is called by Wails after the WebView DOM is ready.
// Shows the window now that the static HTML splash is rendered and visible.
func (a *App) domReady(ctx context.Context) {
	runtime.WindowShow(ctx)
}

// startup is called when Wails initialises the app.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Start system tray icon immediately — it must be visible regardless of
	// daemon state. The poller will set the error icon if daemon is unreachable.
	a.initTray()
	a.trayInit = true

	socketPath := daemon.DefaultSocketPath()
	if err := daemon.EnsureDaemon(socketPath); err != nil {
		a.daemonErr = err
		// Notify frontend — window is already rendered when OnStartup runs (goroutine).
		runtime.EventsEmit(ctx, "daemon:error", err.Error())
		// Start tray poller even on failure — it will show error icon state.
		a.startTrayPoller(ctx)
		return
	}
	a.client = daemon.NewDaemonClient(socketPath)

	// Start tray state poller (updates icon, tooltip, session list every 5s).
	a.startTrayPoller(ctx)

	// Start Tailscale health check background poller.
	a.startHealthPoller(ctx)
}

// GetDaemonError returns the startup error message, or "" if startup succeeded.
// Called by the frontend on mount to detect a failed startup (the daemon:error
// event may fire before React subscribes, so this is the reliable path).
func (a *App) GetDaemonError() string {
	if a.daemonErr != nil {
		return a.daemonErr.Error()
	}
	return ""
}

// RetryDaemon re-attempts daemon spawn after a startup failure.
// Called by the frontend "Retry Connection" button.
func (a *App) RetryDaemon() error {
	socketPath := daemon.DefaultSocketPath()
	if err := daemon.EnsureDaemon(socketPath); err != nil {
		a.daemonErr = err
		return err
	}
	a.daemonErr = nil
	a.client = daemon.NewDaemonClient(socketPath)
	if a.ctx != nil {
		a.startHealthPoller(a.ctx)
	}
	return nil
}

// shutdown is called when the Wails app is about to exit.
func (a *App) shutdown(_ context.Context) {
	// Remove the system tray icon before cleaning up other resources.
	if a.trayInit {
		a.cleanupTray()
	}
	// Daemon is an independent process — GUI does NOT stop it.
	// Sessions persist after GUI exits (DAEMON-03).
}

// beforeClose hides the window instead of quitting so the app stays alive in
// the system tray (tray.go provides the tray icon and Quit menu item).
// When called outside a Wails context (e.g., in unit tests), the window hide
// is skipped safely — sessions are unaffected in both cases.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting {
		return false // allow quit — tray Quit was clicked
	}
	// Wails stores the frontend under the "frontend" key; skip the call when
	// running outside the Wails event loop (tests, CLI helpers).
	if ctx.Value("frontend") != nil {
		runtime.WindowHide(ctx)
	}
	return true // prevent the default quit behaviour — hide window instead
}

// --- Wails-bound methods ---

// CreateSession spawns a new CLI session and returns its ID.
// args are passed to the CLI process; pass nil if no extra arguments are needed.
// cols and rows specify the initial PTY dimensions; pass 0 for defaults (80x24).
// Creates the session through the daemon client, then polls for status updates
// and emits Wails events (replaces the onStatus callback used in earlier phases).
func (a *App) CreateSession(cli, name, workDir string, args []string, cols, rows int) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("daemon not connected")
	}
	id, err := a.client.CreateSession(cli, name, workDir, args, cols, rows)
	if err != nil {
		return "", err
	}
	// Poll session status for up to 60s to emit Wails events (replaces onStatus callback).
	go a.pollSessionStatus(id)
	return id, nil
}

// pollSessionStatus polls the daemon for status changes on a newly created
// session and emits Wails "session:status" events. Replaces the onStatus
// callback that was used when CreateSession called the engine directly.
// Stops when status reaches "errored" or after 60 seconds.
func (a *App) pollSessionStatus(sessionID string) {
	var last string
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		s, err := a.client.GetSessionStatus(sessionID)
		if err != nil {
			return
		}
		if s != last {
			last = s
			if a.ctx != nil && a.ctx.Value("frontend") != nil {
				runtime.EventsEmit(a.ctx, "session:status", map[string]string{
					"sessionId": sessionID,
					"status":    s,
				})
			}
			if s == string(status.StatusErrored) {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// ListSessions returns a snapshot of all registered sessions.
func (a *App) ListSessions() []SessionInfo {
	if a.client == nil {
		return []SessionInfo{}
	}
	sessions, err := a.client.ListSessions()
	if err != nil {
		return []SessionInfo{}
	}
	result := make([]SessionInfo, len(sessions))
	for i, s := range sessions {
		result[i] = SessionInfo{
			ID:        s.ID,
			CLI:       s.CLI,
			Name:      s.Name,
			State:     s.State,
			CreatedAt: s.CreatedAt,
			Hostname:  s.Hostname,
		}
	}
	return result
}

// RenameSession updates the display name of a session.
func (a *App) RenameSession(id, name string) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.RenameSession(id, name)
}

// KillSession terminates the session and removes it from all registries.
func (a *App) KillSession(id string) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	err := a.client.KillSession(id)
	if err != nil {
		return err
	}
	// Emit the status event for the frontend (same payload shape as before).
	if a.ctx != nil && a.ctx.Value("frontend") != nil {
		runtime.EventsEmit(a.ctx, "session:status", map[string]string{
			"sessionId": id,
			"status":    string(status.StatusErrored),
		})
	}
	return nil
}

// GetSessionStatus returns the current heuristic status of the session as a
// string ("running", "waiting", "idle", or "errored").
// Returns "running" if the session is not found (conservative default).
func (a *App) GetSessionStatus(sessionID string) string {
	if a.client == nil {
		return string(status.StatusRunning)
	}
	s, err := a.client.GetSessionStatus(sessionID)
	if err != nil {
		return string(status.StatusRunning) // conservative default
	}
	return s
}

// DetectCLIs returns the list of supported AI coding CLIs found on PATH.
func (a *App) DetectCLIs() []pty.DetectedCLI {
	return pty.DetectCLIs()
}

// GetRelayPort returns the TCP port the daemon's relay HTTP server is listening on.
func (a *App) GetRelayPort() int {
	if a.client == nil {
		return 0
	}
	port, err := a.client.GetRelayPort()
	if err != nil {
		return 0
	}
	return port
}

// UpdateCLIPath stores a custom executable path for the named CLI.
// The path must exist on disk.
func (a *App) UpdateCLIPath(name, path string) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.UpdateCLIPath(name, path)
}

// configDir returns the path to the agenthub config directory (~/.config/agenthub).
// Creates the directory if it does not exist.
func configDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "agenthub")
	_ = os.MkdirAll(dir, 0700)
	return dir
}

// StartWebServer tells the daemon to start the Tailscale web server.
// Returns an error if Tailscale is not connected with HTTPS certs enabled.
func (a *App) StartWebServer(port int) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	h := a.GetTailscaleStatus()
	if !h.Connected {
		return fmt.Errorf("Tailscale is not connected")
	}
	if h.IP == "" {
		return fmt.Errorf("Tailscale IP not available")
	}
	if !h.HasCerts {
		return fmt.Errorf("Tailscale HTTPS certificates not enabled — enable in Tailscale admin")
	}
	_, err := a.client.StartWebServer(h.IP, port, h.Domain)
	return err
}

// StopWebServer tells the daemon to stop the web server.
func (a *App) StopWebServer() error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.StopWebServer()
}

// ToggleWebServing enables or disables web serving for a specific session.
// Returns an error if the web server is not running.
func (a *App) ToggleWebServing(sessionID string, enabled bool) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.ToggleWebServing(sessionID, enabled)
}

// GetWebServerURL returns the base HTTPS URL of the running web server,
// or an empty string if the server is not running.
func (a *App) GetWebServerURL() string {
	if a.client == nil {
		return ""
	}
	resp, err := a.client.GetWebServerStatus()
	if err != nil || !resp.Running {
		return ""
	}
	return resp.URL
}

// ctDisclosurePath returns the path to the CT disclosure acknowledgement file.
func ctDisclosurePath() string {
	return filepath.Join(configDir(), "ct_disclosed")
}

// HasCTDisclosure returns true if the user has acknowledged the CT log disclosure.
func (a *App) HasCTDisclosure() bool {
	_, err := os.Stat(ctDisclosurePath())
	return err == nil
}

// AcknowledgeCTDisclosure persists the CT disclosure acknowledgement.
func (a *App) AcknowledgeCTDisclosure() error {
	return os.WriteFile(ctDisclosurePath(), []byte("1"), 0600)
}

// IsWebServerRunning returns true if the daemon's web server is active.
func (a *App) IsWebServerRunning() bool {
	if a.client == nil {
		return false
	}
	resp, err := a.client.GetWebServerStatus()
	if err != nil {
		return false
	}
	return resp.Running
}

// GetSessionQRCode generates a QR code for the web-served session URL and
// returns it as a base64-encoded PNG string. The QR encodes the session URL
// (https://bindIP:port/sessions/{id}). Returns an error if the web server is
// not running.
func (a *App) GetSessionQRCode(sessionID string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("daemon not connected")
	}
	resp, err := a.client.GetWebServerStatus()
	if err != nil || !resp.Running {
		return "", fmt.Errorf("web server not running")
	}
	url := fmt.Sprintf("%s/sessions/%s", resp.URL, sessionID)
	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("GetSessionQRCode: encode: %w", err)
	}
	return base64.StdEncoding.EncodeToString(png), nil
}

// OpenDirectoryDialog opens a native OS folder picker and returns the selected path.
// Returns "" if the user cancels. Falls back to the user's home directory when
// defaultDir is empty.
func (a *App) OpenDirectoryDialog(defaultDir string) (string, error) {
	if defaultDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			defaultDir = home
		}
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Working Directory",
		DefaultDirectory: defaultDir,
	})
}

// GetTailscaleStatus returns the current Tailscale health state.
// Called by the frontend on-demand; also available as a Wails-bound method.
func (a *App) GetTailscaleStatus() webserver.TailscaleHealth {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return webserver.CheckHealth(ctx)
}

// startTrayPoller starts a background goroutine that refreshes tray state
// (icon, tooltip, session list) immediately and then every 5 seconds.
// The goroutine exits when ctx is cancelled (Wails shutdown).
func (a *App) startTrayPoller(ctx context.Context) {
	go func() {
		// Do an immediate refresh before the first tick.
		a.refreshTrayState()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.refreshTrayState()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// refreshTrayState reads daemon connectivity and session list, then updates
// the tray icon, tooltip, and session menu entries.
func (a *App) refreshTrayState() {
	if !a.trayInit {
		return // tray not yet initialised
	}
	if a.client == nil {
		// Startup failed — tray is visible but daemon is unreachable.
		// Show error icon and appropriate tooltip.
		a.updateTray(nil, false)
		return
	}
	connected := a.client.Health() == nil
	var sessions []SessionInfo
	if connected {
		sessions = a.ListSessions()
	}
	a.updateTray(sessions, connected)
}

// startHealthPoller starts a background goroutine that polls Tailscale health
// every 10 seconds and emits "tailscale:health" events when the state changes.
// The goroutine exits when ctx is cancelled (Wails shutdown).
func (a *App) startHealthPoller(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		var last webserver.TailscaleHealth
		for {
			select {
			case <-ticker.C:
				checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				h := webserver.CheckHealth(checkCtx)
				cancel()
				if h != last {
					last = h
					if a.ctx != nil && a.ctx.Value("frontend") != nil {
						runtime.EventsEmit(a.ctx, "tailscale:health", h)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
