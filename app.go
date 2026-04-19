package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"time"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/pty"
	"github.com/scottkw/agenthub/internal/status"
	"github.com/scottkw/agenthub/internal/tailnet"
	"github.com/scottkw/agenthub/internal/updater"
	"github.com/scottkw/agenthub/internal/webserver"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SessionInfo is the JSON-serialisable representation of a session returned by ListSessions.
type SessionInfo struct {
	ID         string `json:"id"`
	CLI        string `json:"cli"`
	Name       string `json:"name"`
	State      string `json:"state"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
	Hostname   string `json:"hostname"`
	WebEnabled bool   `json:"webEnabled"`
}

// RemoteSession is a session on a remote tailnet peer.
type RemoteSession struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	CLIType string `json:"cliType"`
	Status  string `json:"status"`
	URL     string `json:"url"`
}

// RemotePeerSessions groups sessions by peer hostname.
type RemotePeerSessions struct {
	Hostname string          `json:"hostname"`
	Sessions []RemoteSession `json:"sessions"`
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
	// Update checker state
	lastUpdate   *updater.UpdateInfo
	lastUpdateMu sync.Mutex
}

// NewApp creates a new App without starting any subsystems.
func NewApp() *App {
	return &App{}
}

// domReady is called by Wails after the WebView DOM is ready.
// Shows the window unless the persisted start-minimized preference is set.
// Falls back to showing the window when the daemon is unreachable (safe default).
func (a *App) domReady(ctx context.Context) {
	startMinimized := false
	if a.client != nil {
		if val, err := a.client.GetStartMinimized(); err == nil {
			startMinimized = val
		}
	}
	if !startMinimized {
		runtime.WindowShow(ctx)
		a.setDockVisible(true)
	}
}

// startup is called when Wails initialises the app.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	appCtx = ctx       // expose to menu callbacks (openGitHubCallback)
	appInstance = a    // expose to menu callbacks (checkForUpdatesCallback)

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
		a.startUpdatePoller(ctx) // update checks don't depend on daemon
		return
	}
	a.client = daemon.NewDaemonClient(socketPath)

	// Start tray state poller (updates icon, tooltip, session list every 5s).
	a.startTrayPoller(ctx)

	// Start Tailscale health check background poller.
	a.startHealthPoller(ctx)

	// Start update checker background poller.
	a.startUpdatePoller(ctx)
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

// GetVersion returns the application version injected at build time.
// Returns "dev" in local development builds.
func (a *App) GetVersion() string {
	return Version
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

// beforeClose intercepts the window close button and emits a Wails event so
// the frontend can show a quit confirmation modal (D-12). The window is shown
// first if hidden (D-08) so the modal is always visible to the user.
// When called outside a Wails context (e.g., in unit tests), the event emit
// is skipped safely — sessions are unaffected in both cases.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting {
		return false // QuitAll already set flag — allow quit
	}
	// Wails stores the frontend under the "frontend" key; skip the call when
	// running outside the Wails event loop (tests, CLI helpers).
	if ctx.Value("frontend") != nil {
		runtime.WindowShow(ctx)       // D-08: ensure window visible for modal
		a.setDockVisible(true)
		runtime.EventsEmit(ctx, "app:quit-requested", nil)
	}
	return true // always prevent default quit — modal owns the decision
}

// QuitGUIOnly hides the GUI window to the system tray without stopping
// the daemon or any active sessions. Sends a macOS notification confirming
// the app is still running in the background (D-10, D-11).
func (a *App) QuitGUIOnly() {
	if a.ctx == nil {
		return
	}
	a.setDockVisible(false)
	runtime.WindowHide(a.ctx)
	// Send macOS notification with session count (D-11)
	sessions := a.ListSessions()
	count := 0
	for _, s := range sessions {
		if s.Status != "" && s.Status != "stopped" {
			count++
		}
	}
	var body string
	if count == 1 {
		body = "AgentHub is still running in the background. 1 session active."
	} else {
		body = fmt.Sprintf("AgentHub is still running in the background. %d sessions active.", count)
	}
	sendNotification("AgentHub", body)
}

// QuitAll shuts down the daemon (terminating all sessions) and quits the
// application completely (D-09, APP-02).
func (a *App) QuitAll() {
	if a.ctx == nil {
		return
	}
	if a.client != nil {
		_ = a.client.ShutdownDaemon()
	}
	a.quitting = true
	runtime.Quit(a.ctx)
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
// session and emits Wails "session:status" events. Also detects natural exit
// (State == "stopped") and emits "session:exit". Replaces the onStatus
// callback that was used when CreateSession called the engine directly.
func (a *App) pollSessionStatus(sessionID string) {
	var last string
	var consecutiveErrors int
	deadline := time.Now().Add(300 * time.Second) // extended to 5min for long-running agents
	for time.Now().Before(deadline) {
		sessions, err := a.client.ListSessions()
		if err != nil {
			consecutiveErrors++
			if consecutiveErrors >= 5 {
				return // daemon is gone — stop polling
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		consecutiveErrors = 0
		found := false
		for _, s := range sessions {
			if s.ID == sessionID {
				found = true
				// Emit heuristic status changes (existing behavior)
				if s.Status != last {
					last = s.Status
					if a.ctx != nil && a.ctx.Value("frontend") != nil {
						runtime.EventsEmit(a.ctx, "session:status", map[string]string{
							"sessionId": sessionID,
							"status":    s.Status,
						})
					}
				}
				// Exit detection: daemon marks session as "stopped" when process exits naturally
				if s.State == "stopped" {
					a.emitExitEvent(sessionID, s)
					return
				}
				break
			}
		}
		if !found {
			// Session removed from daemon (killed externally) — stop polling
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// emitExitEvent sends the session:exit Wails event to the frontend.
func (a *App) emitExitEvent(sessionID string, s daemon.SessionInfo) {
	if a.ctx == nil || a.ctx.Value("frontend") == nil {
		return
	}
	exitCode := 0
	if s.ExitCode != nil {
		exitCode = *s.ExitCode
	}
	duration := 0
	if s.Duration != nil {
		duration = *s.Duration
	}
	runtime.EventsEmit(a.ctx, "session:exit", map[string]any{
		"sessionId":   sessionID,
		"exitCode":    exitCode,
		"sessionName": s.Name,
		"cli":         s.CLI,
		"duration":    duration,
		"finalStatus": s.Status,
	})
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
			ID:         s.ID,
			CLI:        s.CLI,
			Name:       s.Name,
			State:      s.State,
			Status:     s.Status,
			CreatedAt:  s.CreatedAt,
			Hostname:   s.Hostname,
			WebEnabled: s.WebEnabled,
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

// GetCLIPaths returns all stored CLI path overrides. Used by Settings tab to
// populate path inputs with persisted values on mount.
func (a *App) GetCLIPaths() (map[string]string, error) {
	if a.client == nil {
		return nil, fmt.Errorf("daemon not connected")
	}
	return a.client.GetCLIPaths()
}

// GetStartMinimized returns the persisted start-minimized preference.
// Returns false (show window) when daemon is not connected.
func (a *App) GetStartMinimized() bool {
	if a.client == nil {
		return false
	}
	val, err := a.client.GetStartMinimized()
	if err != nil {
		return false
	}
	return val
}

// SetStartMinimized persists the start-minimized preference.
func (a *App) SetStartMinimized(val bool) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.SetStartMinimized(val)
}

// GetAutoCloseSession returns the persisted auto-close-on-exit preference.
// Returns true (enabled) when daemon is not connected (conservative default per D-11).
func (a *App) GetAutoCloseSession() bool {
	if a.client == nil {
		return true
	}
	val, err := a.client.GetAutoCloseSession()
	if err != nil {
		return true // default: enabled
	}
	return val
}

// SetAutoCloseSession persists the auto-close-on-exit preference.
func (a *App) SetAutoCloseSession(val bool) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.SetAutoCloseSession(val)
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

// StartWebServer tells the daemon to start the web server.
// Uses Tailscale mode when connected with certs; falls back to local mode otherwise.
func (a *App) StartWebServer(port int) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	h := a.GetTailscaleStatus()
	if h.Connected && h.IP != "" && h.HasCerts {
		_, err := a.client.StartWebServer(h.IP, port, h.Domain, "tailscale", "")
		return err
	}
	// Local mode fallback — daemon already holds the generated password.
	pwd, _ := a.client.GetLocalNetworkPassword()
	_, err := a.client.StartWebServer("", port, "", "local", pwd)
	return err
}

// GetLocalNetworkPassword returns the generated LAN access password from the daemon,
// or empty string if not in local mode or daemon is unreachable.
func (a *App) GetLocalNetworkPassword() string {
	if a.client == nil {
		return ""
	}
	pwd, err := a.client.GetLocalNetworkPassword()
	if err != nil {
		return ""
	}
	return pwd
}

// GetWebServerMode returns the web server mode ("tailscale", "local", or "").
// Returns "" when the web server is not running or daemon is unreachable.
func (a *App) GetWebServerMode() string {
	if a.client == nil {
		return ""
	}
	resp, err := a.client.GetWebServerStatus()
	if err != nil || !resp.Running {
		return ""
	}
	return resp.Mode
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

// NotifyThemeChange signals active OpenCode terminal sessions to re-query
// the terminal palette after a theme change in Settings > Appearance.
// Fire-and-forget from the frontend — errors are logged, not surfaced to UI.
func (a *App) NotifyThemeChange() error {
	if a.client == nil {
		return nil // daemon not connected; no sessions to signal
	}
	return a.client.NotifyThemeChange()
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

// GetWebServerQRCode returns a base64-encoded PNG QR code for the web server dashboard URL.
// Returns an error if the daemon is disconnected or the web server is not running.
func (a *App) GetWebServerQRCode() (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("daemon not connected")
	}
	resp, err := a.client.GetWebServerStatus()
	if err != nil || !resp.Running {
		return "", fmt.Errorf("web server not running")
	}
	png, err := qrcode.Encode(resp.URL, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("GetWebServerQRCode: encode: %w", err)
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

// OpenFileDialog opens a native OS file picker and returns the selected path.
// Returns "" if the user cancels. Falls back to the user's home directory when
// defaultDir is empty. Used by Settings > Paths browse buttons (SET-04).
func (a *App) OpenFileDialog(defaultDir string) (string, error) {
	if defaultDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			defaultDir = home
		}
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Executable",
		DefaultDirectory: defaultDir,
		ShowHiddenFiles:  true,
	})
}

// GetTailscaleStatus returns the current Tailscale health state.
// Called by the frontend on-demand; also available as a Wails-bound method.
func (a *App) GetTailscaleStatus() webserver.TailscaleHealth {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	customPath := ""
	if a.client != nil {
		if paths, err := a.client.GetCLIPaths(); err == nil {
			customPath = paths["tailscale"]
		}
	}
	return webserver.CheckHealthWithCustomPath(ctx, customPath)
}

// GetRemoteSessions discovers tailnet peers and fetches their session lists.
// Returns an empty slice if the daemon is unreachable or no peers are found.
// Individual peer fetch failures are silently omitted.
// Delegates to tailnet.FetchAllPeerSessions for concurrent fetch with IP fallback.
func (a *App) GetRemoteSessions() []RemotePeerSessions {
	if a.client == nil {
		return []RemotePeerSessions{}
	}
	peers, err := a.client.ListTailnetPeers()
	if err != nil || len(peers) == 0 {
		return []RemotePeerSessions{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	groups := tailnet.FetchAllPeerSessions(ctx, peers)

	results := make([]RemotePeerSessions, 0, len(groups))
	for _, g := range groups {
		sessions := make([]RemoteSession, 0, len(g.Sessions))
		for _, s := range g.Sessions {
			sessions = append(sessions, RemoteSession{
				ID:      s.ID,
				Name:    s.Name,
				CLIType: s.CLIType,
				Status:  s.Status,
				URL:     s.URL,
			})
		}
		results = append(results, RemotePeerSessions{Hostname: g.Hostname, Sessions: sessions})
	}
	return results
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

// startUpdatePoller checks for updates on startup and every hour.
// Follows the startHealthPoller/startTrayPoller goroutine+ticker pattern.
func (a *App) startUpdatePoller(ctx context.Context) {
	go func() {
		// Initial check after 5-second delay to avoid startup race
		// (frontend needs time to mount and subscribe to events).
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}
		a.runUpdateCheck(ctx, false)
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.runUpdateCheck(ctx, false)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// runUpdateCheck performs a single update check and emits an event if a newer version is found.
func (a *App) runUpdateCheck(ctx context.Context, force bool) {
	info, err := updater.Check(ctx, configDir(), "scottkw/agenthub", Version, updater.DefaultDetect, force)
	if err != nil || info == nil {
		return
	}
	a.lastUpdateMu.Lock()
	a.lastUpdate = info
	a.lastUpdateMu.Unlock()
	runtime.EventsEmit(ctx, "update:available", info)
}

// GetLastUpdateInfo returns the latest update info, or nil if no update is available.
// Called by frontend on mount to handle the startup race condition where
// update:available events may fire before React subscribes.
func (a *App) GetLastUpdateInfo() *updater.UpdateInfo {
	a.lastUpdateMu.Lock()
	defer a.lastUpdateMu.Unlock()
	return a.lastUpdate
}

// CheckForUpdates performs an immediate update check, bypassing the rate limit.
// Called from Help > Check for Updates menu item.
func (a *App) CheckForUpdates() *updater.UpdateInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, _ := updater.Check(ctx, configDir(), "scottkw/agenthub", Version, updater.DefaultDetect, true)
	if info != nil {
		a.lastUpdateMu.Lock()
		a.lastUpdate = info
		a.lastUpdateMu.Unlock()
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "update:available", info)
		}
	}
	return info
}

// findBrew returns the path to the Homebrew binary, checking both Apple Silicon
// and Intel Mac install locations.
func findBrew() (string, error) {
	if _, err := os.Stat("/opt/homebrew/bin/brew"); err == nil {
		return "/opt/homebrew/bin/brew", nil
	}
	if _, err := os.Stat("/usr/local/bin/brew"); err == nil {
		return "/usr/local/bin/brew", nil
	}
	return "", fmt.Errorf("Homebrew not found; install from https://brew.sh")
}

// AutoInstallTailscale runs `brew install --cask tailscale-app` on macOS,
// streaming stdout/stderr lines via tailscale:install:progress events and
// emitting tailscale:install:done on completion.
func (a *App) AutoInstallTailscale() error {
	if goruntime.GOOS != "darwin" {
		return fmt.Errorf("auto-install is only supported on macOS")
	}
	brewPath, err := findBrew()
	if err != nil {
		return err
	}
	cmd := exec.Command(brewPath, "install", "--cask", "tailscale-app")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "tailscale:install:progress", line)
			}
		}
		exitErr := cmd.Wait()
		if exitErr != nil {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "tailscale:install:done", map[string]interface{}{"success": false, "error": exitErr.Error()})
			}
		} else {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "tailscale:install:done", map[string]interface{}{"success": true})
			}
		}
	}()
	return nil
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
				customPath := ""
				if a.client != nil {
					if paths, err := a.client.GetCLIPaths(); err == nil {
						customPath = paths["tailscale"]
					}
				}
				checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				h := webserver.CheckHealthWithCustomPath(checkCtx, customPath)
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
