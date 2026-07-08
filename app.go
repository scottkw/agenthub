package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"sync/atomic"
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
	// Phase 124 / CAP-06: true when the session cwd equals EvalSymlinks($HOME).
	// Server-side source of truth for the home-dir write warning banner (GUI +
	// TUI parity). Must be propagated from daemon.SessionInfo or the GUI banner
	// never fires (UAT finding).
	HomeDir bool `json:"homeDir"`
	// Phase 137 / SHARE-05: true when the per-session browse toggle is ON.
	// Single server-side source of truth for GUI modal seeding (NOT omitempty:
	// false must serialize so the modal can seed on open per RESEARCH open question 2).
	BrowseEnabled bool `json:"browseEnabled"`
	// Phase 165 / FNL-01: true when Tailscale Funnel is active for this session.
	// NOT omitempty: false must serialize so the frontend poll detects expiry
	// (same rule as BrowseEnabled / HomeDir — silent false-drop is a UAT-class bug).
	FunnelActive bool `json:"funnelActive"`
	// Phase 131 — Hub card fields (CARD-04, CARD-05, CARD-06, GRID-02).
	// Omitting any of these silently drops them to zero in the Wails RPC
	// response — same class of UAT bug documented on HomeDir above.
	ViewerCount int    `json:"viewerCount"`
	ExitCode    *int   `json:"exitCode,omitempty"`
	Duration    *int   `json:"duration,omitempty"`
	WorkDir     string `json:"workDir"`
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
// Reachable discriminates unreachable peers (Reachable=false) from reachable
// peers with zero shareable sessions (Reachable=true, len(Sessions)==0), enabling
// honest per-peer states in the Remote Sessions panel (RB-04).
type RemotePeerSessions struct {
	Hostname  string          `json:"hostname"`
	Reachable bool            `json:"reachable"`
	Sessions  []RemoteSession `json:"sessions"`
}

// App holds all application state and exposes the Wails-bound methods.
// App is a thin Wails-binding shell — all session state lives in the daemon process.
// All session operations are delegated through DaemonClient over the Unix socket.
type App struct {
	ctx              context.Context
	client           *daemon.DaemonClient // only daemon communication field; nil when startup failed
	trayInit         bool                 // true once initTray has been called
	lastTrayQuartile atomic.Int32         // Phase 98 PRG-03 — last applied tray progress quartile [0..4]; -1 = unset. Atomic because written by Wails RPC goroutine (SetTrayProgress) and read by startTrayPoller goroutine via trayIconBytesForState (CR-01).
	daemonErr        error                // non-nil when EnsureDaemon failed at startup
	quitting         bool                 // true when tray Quit was clicked; lets beforeClose allow exit
	// Update checker state
	lastUpdate   *updater.UpdateInfo
	lastUpdateMu sync.Mutex

	// saveFileDialogFunc allows unit tests to mock runtime.SaveFileDialog.
	// Defaults to runtime.SaveFileDialog. Phase 97 SER-01 / PROJECT.md
	// "Function injection" pattern, parallel to serviceControlFunc and
	// statusFunc.
	saveFileDialogFunc func(ctx context.Context, opts runtime.SaveDialogOptions) (string, error)

	// refreshTrayStateFunc allows unit tests to mock refreshTrayState().
	// Defaults to nil (production path calls a.refreshTrayState() directly).
	// Phase 98 PRG-03 / PROJECT.md "Function injection" pattern.
	refreshTrayStateFunc func()

	// notifyOnWaiting caches the persisted NotifyOnWaiting preference (Phase
	// 167 / NTF-04). Atomic because it's written by the Wails RPC goroutine
	// (SetNotifyOnWaiting) and read by the tray-poller goroutine
	// (maybeNotifyWaiting), mirroring lastTrayQuartile's concurrency pattern.
	notifyOnWaiting atomic.Bool

	// lastWaitingStatus tracks the previously-observed Status per session ID,
	// used by maybeNotifyWaiting to edge-detect the non-waiting→waiting
	// transition (NTF-02). Only ever read/written from refreshTrayState's
	// single goroutine (the 5s tray-poller ticker), so it needs no separate
	// locking. nil until the first maybeNotifyWaiting call (cold-start
	// baseline capture).
	lastWaitingStatus map[string]string

	// sendNotificationFunc allows unit tests to mock sendNotification so no
	// real OS notification fires. Defaults to sendNotification. Phase 167 /
	// NTF-01..03, mirrors the saveFileDialogFunc injection pattern.
	sendNotificationFunc func(identifier, title, body string)

	// requestNotificationAuthFunc allows unit tests to spy on the proactive
	// authorization entry point without firing a real OS permission prompt.
	// Defaults to requestNotificationAuth. Phase 167-06 (M-41 gap closure) —
	// mirrors the sendNotificationFunc injection pattern.
	requestNotificationAuthFunc func()
}

// NewApp creates a new App without starting any subsystems.
func NewApp() *App {
	a := &App{
		saveFileDialogFunc:          runtime.SaveFileDialog,
		sendNotificationFunc:        sendNotification,
		requestNotificationAuthFunc: requestNotificationAuth,
	}
	a.lastTrayQuartile.Store(-1) // Phase 98 PRG-03 — ensure first SetTrayProgress call always updates
	return a
}

// displayNameForCLI maps a CLI identifier to its human-readable display name
// (Phase 167 / NTF-03). Static mirror of internal/pty/detect.go's knownCLIs
// table — deliberately does NOT call pty.DetectCLI, which performs a live
// PATH scan unsuitable for this per-notification lookup (RESEARCH LOCKED
// decision #5). Unknown CLIs (including shells) fall back to the raw input.
func displayNameForCLI(cli string) string {
	switch cli {
	case "claude":
		return "Claude Code"
	case "codex":
		return "OpenAI Codex"
	case "gemini":
		return "Gemini CLI"
	case "opencode":
		return "OpenCode"
	case "agy":
		return "Google Antigravity"
	default:
		return cli
	}
}

// maybeNotifyWaiting fires a native notification for every session whose
// Status transitioned from non-"waiting" to "waiting" since the previous
// call (Phase 167 / NTF-02). Must be called from a single goroutine (the
// tray-poller ticker) — a.lastWaitingStatus is not otherwise synchronized.
// No-op when the NotifyOnWaiting preference is off (NTF-04). The first call
// (lastWaitingStatus nil) only baselines current statuses without firing, so
// sessions already waiting at cold-start don't trigger a notification burst.
func (a *App) maybeNotifyWaiting(sessions []SessionInfo) {
	if !a.notifyOnWaiting.Load() {
		return
	}
	firstRun := a.lastWaitingStatus == nil
	if firstRun {
		a.lastWaitingStatus = make(map[string]string, len(sessions))
	}
	seen := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		seen[s.ID] = true
		prev, known := a.lastWaitingStatus[s.ID]
		a.lastWaitingStatus[s.ID] = s.Status
		if firstRun {
			continue // baseline capture only — no notification on cold start
		}
		if s.Status == string(status.StatusWaiting) && known && prev != string(status.StatusWaiting) {
			body := fmt.Sprintf("%s (%s) is waiting for your input.", s.Name, displayNameForCLI(s.CLI))
			// Phase 167-06 (M-41 gap closure): confirms from outside whether
			// the trigger actually reached the native send on a signed build.
			log.Printf("notification: non-waiting->waiting edge fired for session %s (status=%s notifyOnWaiting=%v)", s.ID, s.Status, a.notifyOnWaiting.Load())
			a.sendNotificationFunc("agenthub.session-waiting."+s.ID, "AgentHub", body)
		}
	}
	for id := range a.lastWaitingStatus {
		if !seen[id] {
			delete(a.lastWaitingStatus, id) // prune sessions no longer present
		}
	}
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
	appCtx = ctx    // expose to menu callbacks (openGitHubCallback)
	appInstance = a // expose to menu callbacks (checkForUpdatesCallback)

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

	// Cache the persisted NotifyOnWaiting preference (Phase 167 / NTF-04).
	// On error, leave the atomic at its zero value (false — the safe default).
	if val, err := a.client.GetNotifyOnWaiting(); err == nil {
		a.notifyOnWaiting.Store(val)
	} else {
		log.Printf("notification: initial NotifyOnWaiting cache load failed (%v) — defaulting to false until the toggle is set", err)
	}

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
		runtime.WindowShow(ctx) // D-08: ensure window visible for modal
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
	sendNotification("agenthub.quit-gui-only", "AgentHub", body)
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

// shouldContinuePolling is the pure, injectable-clock deadline check driving
// pollSessionStatus's loop. It returns true while now is still within
// maxWindow of pollStart, matching the original inline
// `time.Now().Before(deadline)` semantics exactly (deadline == pollStart +
// maxWindow). Extracted as a side-effect-free helper (175-02 Task 1 / BUG-03
// Wave 0 scaffolding) so the exit-poll deadline math is unit-testable without
// a live daemon or wall-clock sleeps; 175-05 is the plan that may change the
// semantics (e.g. re-arming the deadline) — this extraction is
// behavior-preserving only.
func shouldContinuePolling(pollStart, now time.Time, maxWindow time.Duration) bool {
	return now.Before(pollStart.Add(maxWindow))
}

// pollSessionStatus polls the daemon for status changes on a newly created
// session and emits Wails "session:status" events. Also detects natural exit
// (State == "stopped") and emits "session:exit". Replaces the onStatus
// callback that was used when CreateSession called the engine directly.
func (a *App) pollSessionStatus(sessionID string) {
	var last string
	var consecutiveErrors int
	pollStart := time.Now()
	const maxPollWindow = 300 * time.Second // extended to 5min for long-running agents
	for shouldContinuePolling(pollStart, time.Now(), maxPollWindow) {
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
			// Phase 137 / SHARE-05 + CAP-06: propagate browse-enabled and home-dir
			// flags so the GUI's Share modal seeds from daemon source of truth.
			// Omitting these silently drops them to false (same UAT class as FilesWrite bug).
			HomeDir:       s.HomeDir,
			BrowseEnabled: s.BrowseEnabled,
			// Phase 165 / FNL-01: propagate Funnel state so the frontend poll
			// can detect expiry. Omitting silently drops to false (T-165-15).
			FunnelActive: s.FunnelActive,
			// Phase 131 / CARD-04..06, GRID-02: propagate Hub card fields from
			// daemon source of truth. Omitting these silently drops them to zero
			// — the same class of silent-corruption bug documented on HomeDir above.
			ViewerCount: s.ViewerCount,
			ExitCode:    s.ExitCode,
			Duration:    s.Duration,
			WorkDir:     s.WorkDir,
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

// GetSessionTailLines returns the last n plain-text lines from the session's
// scrollback buffer, with ANSI escape sequences and relay framing bytes stripped.
// Returns an empty slice if the session has no scrollback (e.g. remote sessions)
// or if the daemon is unreachable. n is clamped to [1..20] to bound response size.
// Used by the Hub mini-preview poller (CARD-07). Phase 132.
func (a *App) GetSessionTailLines(id string, n int) []string {
	if a.client == nil {
		return []string{}
	}
	if n < 1 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	lines, err := a.client.GetSessionTailLines(id, n)
	if err != nil || lines == nil {
		return []string{}
	}
	return lines
}

// GetSessionStyledTailLines returns the last n styled-cell lines from the
// session's scrollback buffer, rendered through a headless VT emulator with
// per-cell color and bold attributes. Returns an empty slice for remote
// sessions, unreachable daemon, or unknown session IDs.
// n is clamped to [1..20] (defense in depth — second enforcement layer
// mirroring the HTTP handler clamp in daemon/api.go).
// Phase 139 / CARD-05.
func (a *App) GetSessionStyledTailLines(id string, n int) [][]daemon.StyledSpan {
	if a.client == nil {
		return [][]daemon.StyledSpan{}
	}
	if n < 1 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	lines, err := a.client.GetSessionStyledTailLines(id, n)
	if err != nil || lines == nil {
		return [][]daemon.StyledSpan{}
	}
	return lines
}

// DetectCLIs returns the list of supported AI coding CLIs found on PATH.
func (a *App) DetectCLIs() []pty.DetectedCLI {
	return pty.DetectCLIs()
}

// ListShells returns the daemon's discovery of installed shells. Phase 101-01
// prerequisite for plan 101-02 (NewSessionModal shell rows). Mirrors the
// thin-delegation pattern used by ListSessions/DetectCLIs.
//
// Returns an empty slice (not nil) on error or when the daemon is unreachable —
// the GUI degrades gracefully by omitting shell rows from the new-session modal
// (per UI-SPEC §Edge Cases "silent absence" pattern).
func (a *App) ListShells() []daemon.DetectedShell {
	if a.client == nil {
		return []daemon.DetectedShell{}
	}
	shells, err := a.client.ListShells()
	if err != nil {
		return []daemon.DetectedShell{}
	}
	return shells
}

// GetShellWebShareWarned returns the persisted "user has been warned about shell
// web-share security implications" flag. Used by ShellWebShareBanner (plan 101-03)
// to suppress the one-time confirmation banner on subsequent toggles.
func (a *App) GetShellWebShareWarned() bool {
	if a.client == nil {
		return false
	}
	warned, err := a.client.GetShellWebShareWarned()
	if err != nil {
		return false
	}
	return warned
}

// SetShellWebShareWarned persists the "user has been warned" flag. Called by
// ShellWebShareBanner (plan 101-03) when the user dismisses the one-time
// confirmation banner.
func (a *App) SetShellWebShareWarned(v bool) error {
	if a.client == nil {
		return nil
	}
	return a.client.SetShellWebShareWarned(v)
}

// GetShellWebShareWarningEnabled returns the warning-enabled master switch.
// Returns true (enabled) when the daemon is not connected — safe degradation per D-08.
// Phase 150 SET-01.
func (a *App) GetShellWebShareWarningEnabled() bool {
	if a.client == nil {
		return true // default ON (not false like GetShellWebShareWarned)
	}
	val, err := a.client.GetShellWebShareWarningEnabled()
	if err != nil {
		return true // default: enabled (safe degradation per D-08)
	}
	return val
}

// SetShellWebShareWarningEnabled persists the warning-enabled master switch.
// When enabling, the daemon atomically resets shellWebShareWarned (D-03 re-arm).
// Phase 150 SET-01.
func (a *App) SetShellWebShareWarningEnabled(v bool) error {
	if a.client == nil {
		return nil
	}
	return a.client.SetShellWebShareWarningEnabled(v)
}

// GetShellPath returns the persisted shell binary path. When no path has been
// set by the user, the daemon returns the platform default. Called by the
// Settings → Paths "Shell binary" field (plan 107-03) on mount.
// Phase 107 SHELL-11.
func (a *App) GetShellPath() string {
	if a.client == nil {
		return ""
	}
	path, err := a.client.GetShellPath()
	if err != nil {
		return ""
	}
	return path
}

// SetShellPath persists the shell binary path. An empty path clears the
// override (restores platform default). A non-empty path that does not exist
// or is not executable causes the daemon to return an error. Called by the
// Settings → Paths "Shell binary" field Save Paths click (plan 107-03).
// Phase 107 SHELL-11.
func (a *App) SetShellPath(v string) error {
	if a.client == nil {
		return nil
	}
	return a.client.SetShellPath(v)
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

// GetStayOnHubAfterCreate returns the persisted "stay on Hub after creating
// a session" preference (Phase 168 / UX-01). Returns false (auto-switch,
// today's behavior) when daemon is not connected. Plain client passthrough —
// no atomic cache needed since there is no background reader (unlike
// NotifyOnWaiting's tray poller).
func (a *App) GetStayOnHubAfterCreate() bool {
	if a.client == nil {
		return false
	}
	val, err := a.client.GetStayOnHubAfterCreate()
	if err != nil {
		return false
	}
	return val
}

// SetStayOnHubAfterCreate persists the "stay on Hub after creating a
// session" preference (Phase 168 / UX-01).
func (a *App) SetStayOnHubAfterCreate(val bool) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.SetStayOnHubAfterCreate(val)
}

// GetNotifyOnWaiting returns the cached native-notification-on-waiting
// preference (Phase 167 / NTF-04). Reads the atomic cache rather than the
// daemon so the tray-poller's edge detector and the Settings toggle always
// agree on the current value without an extra round trip.
func (a *App) GetNotifyOnWaiting() bool {
	return a.notifyOnWaiting.Load()
}

// SetNotifyOnWaiting updates the cached preference immediately (so
// maybeNotifyWaiting picks it up on the very next tick) and persists it via
// the daemon client. When notifications are newly enabled, the tray
// poller's next tick performs a fresh baseline capture (Task 1), preventing
// a cold-start notification burst for sessions already waiting.
func (a *App) SetNotifyOnWaiting(val bool) error {
	a.notifyOnWaiting.Store(val)
	// Phase 167-06 (M-41 gap closure): proactively surface the macOS
	// permission prompt the moment the toggle is enabled, regardless of
	// daemon connectivity — the leading suspected fix (during UAT the app
	// showed as "Off" in System Settings with NO prompt ever seen).
	if val && a.requestNotificationAuthFunc != nil {
		a.requestNotificationAuthFunc()
	}
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.SetNotifyOnWaiting(val)
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

// GetPluginSettings returns the persisted plugin enable/disable preferences.
// Returns the zero-value PluginSettings when the daemon is not connected
// or when the RPC fails — callers MUST also gate on a toggleLoaded-style
// guard (the React Settings UI does this via `pluginsLoaded`).
//
// PLUG-03 / Phase 92.
func (a *App) GetPluginSettings() daemon.PluginSettings {
	if a.client == nil {
		return daemon.PluginSettings{}
	}
	s, err := a.client.GetPluginSettings()
	if err != nil {
		return daemon.PluginSettings{}
	}
	return s
}

// SetPluginSettings persists plugin preferences via the daemon AND
// broadcasts the change to all open desktop terminals via the
// "settings:plugins" Wails runtime event (PLUG-03).
//
// EventsEmit lives in app.go ONLY — internal/daemon has no Wails runtime
// context (Pitfall #2). The event fires AFTER the daemon RPC succeeds:
// a failed save MUST NOT emit the event, otherwise consumers would see
// a state the daemon never persisted.
func (a *App) SetPluginSettings(s daemon.PluginSettings) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	if err := a.client.SetPluginSettings(s); err != nil {
		return err
	}
	// WR-05: guard against nil a.ctx (test harness or early Wails-bound
	// RPC fired before startup) — runtime.EventsEmit panics on nil ctx.
	// Mirrors the existing pattern at app.go:266 / app.go:355 / app.go:1006.
	if a.ctx != nil && a.ctx.Value("frontend") != nil {
		runtime.EventsEmit(a.ctx, "settings:plugins", s)
	}
	return nil
}

// SetSearchConfig persists ONLY the find-bar SearchConfig sub-key of
// PluginSettings via the daemon AND broadcasts the resulting full
// PluginSettings to all open desktop terminals via the "settings:plugins"
// Wails runtime event.
//
// Phase 94-07 WR-03 (gap closure) — used by TerminalPanel.handleSearchOptionsChange
// instead of SetPluginSettings(full snapshot from prop), which raced
// PluginsSection's stale local edit buffer.
//
// The event payload is the full PluginSettings (re-fetched via the daemon)
// because App.tsx's EventsOn('settings:plugins') subscription expects a
// PluginSettings shape — same listener consumes both Set methods. The
// re-fetch happens on the App side so the event reflects the post-write
// truth (including the unchanged non-search fields).
func (a *App) SetSearchConfig(cfg daemon.SearchConfig) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	if err := a.client.SetSearchConfig(cfg); err != nil {
		return err
	}
	// Re-fetch the full PluginSettings so the event payload matches the
	// SetPluginSettings event shape (App.tsx listener expects PluginSettings).
	full, err := a.client.GetPluginSettings()
	if err != nil {
		// Persistence succeeded but readback failed — synthesize a payload
		// from defaults + new SearchConfig so listeners still receive a frame.
		// The next GetPluginSettings call will reconcile.
		full = daemon.PluginSettings{SearchConfig: cfg}
	}
	// WR-05: guard against nil a.ctx (test harness or pre-startup RPC).
	if a.ctx != nil && a.ctx.Value("frontend") != nil {
		runtime.EventsEmit(a.ctx, "settings:plugins", full)
	}
	return nil
}

// SetWebLinksConfig persists ONLY the WebLinksConfig sub-key of
// PluginSettings via the daemon AND broadcasts the resulting full
// PluginSettings to all open desktop terminals via the "settings:plugins"
// Wails runtime event.
//
// Phase 95 LNK-05 / LNK-06 — mirror of Phase 94-07 SetSearchConfig. The
// sub-key writer preserves PluginsSection's edit buffer semantics: a
// future Settings advanced-disclosure write (Phase 99 / PUI-03) cannot
// stomp an in-flight Plugins-tab boolean edit.
//
// The event payload is the full PluginSettings (re-fetched via the daemon)
// because App.tsx's EventsOn('settings:plugins') subscription expects a
// PluginSettings shape — the same listener consumes SetPluginSettings,
// SetSearchConfig, and SetWebLinksConfig events. The re-fetch happens on
// the App side so the event reflects the post-write truth (including the
// unchanged non-web-links fields).
func (a *App) SetWebLinksConfig(cfg daemon.WebLinksConfig) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	if err := a.client.SetWebLinksConfig(cfg); err != nil {
		return err
	}
	// Re-fetch the full PluginSettings so the event payload matches the
	// SetPluginSettings event shape (App.tsx listener expects PluginSettings).
	full, err := a.client.GetPluginSettings()
	if err != nil {
		// Persistence succeeded but readback failed — synthesize a payload
		// from defaults + new WebLinksConfig so listeners still receive a
		// frame. The next GetPluginSettings call will reconcile.
		full = daemon.PluginSettings{WebLinksConfig: cfg}
	}
	// WR-05: guard against nil a.ctx (test harness or pre-startup RPC).
	if a.ctx != nil && a.ctx.Value("frontend") != nil {
		runtime.EventsEmit(a.ctx, "settings:plugins", full)
	}
	return nil
}

// SetImageConfig persists ONLY the ImageConfig sub-key of PluginSettings
// via the daemon, then re-fetches the full PluginSettings and emits
// 'settings:plugins' so the React listener consumes it like a
// SetPluginSettings frame (same shape, same handler).
//
// Phase 96 IMG-02 — mirror of Phase 95 SetWebLinksConfig and Phase 94-07
// SetSearchConfig. Concurrency contract is delegated to the daemon's
// engine.SetImageConfig sub-key writer (mutate under e.mu.Lock(); save;
// capture listener; release lock; invoke listener after release).
//
// Note on next-session-only semantics: the event fires (so future
// <details> UIs and the web SSE consumer reflect the new persisted
// value), but the desktop TerminalPanel mount useEffect intentionally
// does NOT include `imageConfig` in any hot-swap dep array — newly-
// mounted sessions pick up the new StorageLimit; already-open sessions
// do not. The PluginsSection italic caption is the user-facing
// affordance for this constraint.
func (a *App) SetImageConfig(cfg daemon.ImageConfig) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	if err := a.client.SetImageConfig(cfg); err != nil {
		return err
	}
	// Re-fetch the full PluginSettings so the event payload matches the
	// SetPluginSettings event shape (App.tsx listener expects PluginSettings).
	full, err := a.client.GetPluginSettings()
	if err != nil {
		// Persistence succeeded but readback failed — synthesize a payload
		// from defaults + new ImageConfig so listeners still receive a
		// frame. The next GetPluginSettings call will reconcile.
		full = daemon.PluginSettings{ImageConfig: cfg}
	}
	// WR-05: guard against nil a.ctx (test harness or pre-startup RPC).
	// Phase 95 code-review fix carried forward.
	if a.ctx != nil && a.ctx.Value("frontend") != nil {
		runtime.EventsEmit(a.ctx, "settings:plugins", full)
	}
	return nil
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

// DisconnectViewers force-closes every remote (web-origin) viewer currently
// connected to the session, without revoking the share capability. Phase 168
// / FIX-02, #117.
func (a *App) DisconnectViewers(sessionID string) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.DisconnectViewers(sessionID)
}

// SetSessionBrowse enables or disables the per-session file-browse capability
// for a specific session. Phase 137 / SHARE-03. Mirrors ToggleWebServing but
// targets the engine's per-session browse map (sole driver of file-perm
// injection per D-02/D-03/D-04). Toggle-off clears outstanding grants on the
// daemon (stale-cap threat mitigation per SHARE-05).
func (a *App) SetSessionBrowse(sessionID string, enabled bool) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.SetSessionBrowse(sessionID, enabled)
}

// SetSessionFunnel enables or disables Tailscale Funnel for a session.
// Phase 165 / FNL-01. Thin bridge — all Funnel logic lives in the daemon
// (165-02). expiresIn is the auto-expiry in seconds (0 = no expiry, FNL-07)
// so Phase 166's expiry picker can drive the full FNL-07 path.
func (a *App) SetSessionFunnel(sessionID string, enabled bool, expiresIn int) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.SetSessionFunnel(sessionID, enabled, expiresIn)
}

// SetSessionFunnelWrite mints the gate-minted, terminal-only public write
// capability for a session (FNL-09). Phase 171-02. Thin bridge — all
// RW-gate lifecycle logic lives in the daemon; expiresIn is the requested
// TTL in seconds (server-clamped unconditionally to (0, 3600]).
func (a *App) SetSessionFunnelWrite(sessionID string, expiresIn int) (daemon.SetSessionFunnelWriteResponse, error) {
	if a.client == nil {
		return daemon.SetSessionFunnelWriteResponse{}, fmt.Errorf("daemon not connected")
	}
	writeURL, writeCode, expiresAt, err := a.client.SetSessionFunnelWrite(sessionID, expiresIn)
	if err != nil {
		return daemon.SetSessionFunnelWriteResponse{}, err
	}
	return daemon.SetSessionFunnelWriteResponse{WriteURL: writeURL, WriteCode: writeCode, ExpiresAt: expiresAt}, nil
}

// DisableSessionFunnelWrite revokes the gate-minted write grant/code/gate
// for a session (FNL-09) without disturbing the reusable public read share.
// Phase 171-02. Thin bridge — mirrors SetSessionFunnelWrite.
func (a *App) DisableSessionFunnelWrite(sessionID string) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.DisableSessionFunnelWrite(sessionID)
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

// --- Phase 87 capability Wails bindings (D-06, D-07, D-09, D-16) ---------

// IssueCapabilities asks the daemon to mint two capabilities (read + read,write)
// for the given session (D-07). The session must be web-enabled (caller must
// have already called ToggleWebServing). Returns the read/write URLs and the
// matching single-use join codes.
func (a *App) IssueCapabilities(sessionID string) (daemon.IssueCapabilitiesResponse, error) {
	if a.client == nil {
		return daemon.IssueCapabilitiesResponse{}, fmt.Errorf("daemon not connected")
	}
	return a.client.IssueCapabilities(sessionID)
}

// ExchangeJoinCode consumes a single-use join code and returns the
// capability-bearing URL the client should follow. Called by the /join page
// after the user taps "Join Session".
func (a *App) ExchangeJoinCode(code string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("daemon not connected")
	}
	return a.client.ExchangeJoinCode(code)
}

// RegenerateSigningKey rotates the HMAC signing key (D-16 panic button in
// Settings > Security). All previously-issued capabilities become invalid.
func (a *App) RegenerateSigningKey() error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.RegenerateSigningKey()
}

// GetCapabilityQRCode encodes a join-code exchange URL as a base64 PNG QR
// code (D-09). The caller must pass the join-code URL (e.g.
// "https://host/join?code=A7K-4P2N"), NOT the raw capability token URL.
// Mirrors GetWebServerQRCode; the encoder call is identical.
func (a *App) GetCapabilityQRCode(joinURL string) (string, error) {
	if joinURL == "" {
		return "", fmt.Errorf("joinURL required")
	}
	png, err := qrcode.Encode(joinURL, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("GetCapabilityQRCode: encode: %w", err)
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

// SaveTerminalSession opens a native Save File dialog and writes the supplied
// terminal scrollback content to the user-chosen path. Cancellation is silent
// success. Returns wrapped errors for dialog setup or write failures.
//
// Phase 97 SER-01. Mirrors OpenFileDialog (lines 815-829) for the cancel=""
// pattern. Uses saveFileDialogFunc indirection (function-injection per
// PROJECT.md "Key Decisions") so unit tests can mock the Wails runtime.
func (a *App) SaveTerminalSession(defaultDir, defaultName, content string) error {
	if defaultDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			defaultDir = home
		}
	}
	path, err := a.saveFileDialogFunc(a.ctx, runtime.SaveDialogOptions{
		Title:                "Save Terminal As… (file will include any printed secrets)",
		DefaultDirectory:     defaultDir,
		DefaultFilename:      defaultName,
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{DisplayName: "Text File (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return fmt.Errorf("SaveTerminalSession: dialog: %w", err)
	}
	if path == "" {
		return nil // user cancelled — silent success per OpenFileDialog precedent
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("SaveTerminalSession: write: %w", err)
	}
	return nil
}

// DownloadFile opens a native Save File dialog and writes bytes fetched from
// the given URL (typically the local relay's /api/files/read endpoint) to
// the user-chosen path. Cancellation is silent success.
//
// Phase 120 UAT-1: the React <a download> attribute is ignored by Wails'
// WKWebView — clicking a download link just navigates the webview to the
// URL, replacing the React app with raw file content. This method bridges
// the gap by performing the save server-side with a native dialog.
//
// url MUST be a loopback URL (the relay at 127.0.0.1 or the daemon's
// local /api/files/remote/{sid}/* proxy). The URL is fetched without auth
// because both endpoints are bound to 127.0.0.1 and trust the loopback
// transport (or, for the remote proxy, the cap is injected server-side
// from RemoteCapStore).
func (a *App) DownloadFile(url, suggestedName string) error {
	defaultDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		defaultDir = filepath.Join(home, "Downloads")
		if _, statErr := os.Stat(defaultDir); statErr != nil {
			defaultDir = home
		}
	}
	path, err := a.saveFileDialogFunc(a.ctx, runtime.SaveDialogOptions{
		Title:                "Save File As…",
		DefaultDirectory:     defaultDir,
		DefaultFilename:      suggestedName,
		CanCreateDirectories: true,
	})
	if err != nil {
		return fmt.Errorf("DownloadFile: dialog: %w", err)
	}
	if path == "" {
		return nil // user cancelled — silent success per SaveTerminalSession precedent
	}
	// Loopback fetch with a generous timeout sized for the 5 MiB cap +
	// network slack. The relay/daemon-proxy never blocks longer than this.
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("DownloadFile: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DownloadFile: server returned %d", resp.StatusCode)
	}
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("DownloadFile: create: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("DownloadFile: write: %w", err)
	}
	return nil
}

// SetTrayProgress sets the cross-session aggregate progress quartile and
// refreshes the system tray icon if the quartile changed.
//
// Quartile semantics: 0 = no active progress (revert to base icon);
// 1..4 = 25/50/75/100% quartile glyphs. Frontend caller is App.tsx,
// which debounces at 200ms before invoking this RPC (Pitfall #5);
// the Go-side idempotency check (Pitfall #3 transition guard) ensures
// identical quartile values do not churn the platform tray API even
// if the debounce is bypassed by a slow flap.
//
// Returns an error if quartile is out of range [0,4]; silent no-op if
// the tray subsystem hasn't initialized (a.trayInit == false).
func (a *App) SetTrayProgress(quartile int) error {
	if !a.trayInit {
		return nil
	}
	if quartile < 0 || quartile > 4 {
		return fmt.Errorf("SetTrayProgress: quartile out of range [0,4]: %d", quartile)
	}
	if a.lastTrayQuartile.Load() == int32(quartile) {
		return nil
	}
	a.lastTrayQuartile.Store(int32(quartile))
	if a.refreshTrayStateFunc != nil {
		a.refreshTrayStateFunc()
	} else {
		a.refreshTrayState()
	}
	return nil
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
		// Every group returned by FetchAllPeerSessions answered the cap-gated
		// probe, so it is reachable by definition. Set Reachable=true explicitly
		// to match the RemotePeerSessions semantics of GetRemoteSessionsWithMeta
		// (WR-04: leaving it defaulted to false mislabels reachable peers as
		// "Unreachable" in RemoteSessionsPanel for any future caller).
		results = append(results, RemotePeerSessions{Hostname: g.Hostname, Reachable: true, Sessions: sessions})
	}
	return results
}

// GetRemoteSessionsWithMeta discovers tailnet peers and fetches their shareable-session
// metadata via the open /api/sessions/meta endpoint (no cap required). Returns ALL
// probed peers including unreachable ones (Reachable=false) and peers with zero
// shareable sessions (Reachable=true, len(Sessions)==0) — enabling honest per-peer
// states in the Remote Sessions panel (RB-01, RB-04).
//
// This RPC uses the new tailnet metadata path (FetchAllPeerSessionsMeta) that never
// silently drops peers, unlike the cap-gated GetRemoteSessions path. GetRemoteSessions
// is retained for backward compatibility during the plan-04 frontend rewire.
func (a *App) GetRemoteSessionsWithMeta() []RemotePeerSessions {
	if a.client == nil {
		return []RemotePeerSessions{}
	}
	peers, err := a.client.ListTailnetPeers()
	if err != nil || len(peers) == 0 {
		return []RemotePeerSessions{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	groups := tailnet.FetchAllPeerSessionsMeta(ctx, peers)

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
		results = append(results, RemotePeerSessions{
			Hostname:  g.Hostname,
			Reachable: g.Reachable,
			Sessions:  sessions,
		})
	}
	return results
}

// ExchangeJoinCodeAtURL exchanges a 5-character join code against a REMOTE
// peer's `/join/exchange` endpoint (NOT the local daemon) and returns the cap
// token extracted from the response.
//
// Phase 122-03 / REMOTE-01 — the desktop GUI's paste-join-code modal calls
// this to obtain a cap for browsing files on a remote tailnet session. The
// returned cap is subsequently deposited via RegisterRemoteCap; it must NEVER
// be passed to the React frontend (it lives in the daemon's RemoteCapStore).
func (a *App) ExchangeJoinCodeAtURL(remoteBaseURL, code string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("daemon not connected")
	}
	return a.client.ExchangeJoinCodeAtURL(remoteBaseURL, code)
}

// RegisterRemoteCap deposits a (sessionID, remote baseURL, cap token) tuple
// into the local daemon's RemoteCapStore. Subsequent file-browser fetches
// through the local-daemon proxy route /api/files/remote/{sid}/... use this
// cap. Phase 122-03 / REMOTE-01.
func (a *App) RegisterRemoteCap(sessionID, baseURL, capToken string) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.RegisterRemoteCap(sessionID, baseURL, capToken)
}

// OpenRemoteSessionURL returns the cap-bearing open URL for a remote session
// by reading the already-stored cap from the daemon's RemoteCapStore. The
// daemon composes baseURL+/sessions/{id}?cap=TOKEN from its own store entry
// (keyed by sessionID) and returns the full URL as a string.
//
// Convention (matches ExchangeJoinCodeAtURL): the frontend receives the
// composed URL string and opens it via BrowserOpenURL; the raw cap token never
// enters React state — it lives only inside the returned URL string.
//
// Returns an error when the daemon is not connected or when no cap is stored
// for sessionID (e.g. "status 404"). The caller treats a 404-flavoured error
// as "no cap held; fall back to the join-code modal" so a stale/evicted entry
// self-heals gracefully (T-146-05-02 accept).
//
// Do NOT call BrowserOpenURL here — the frontend opens the returned URL,
// matching the existing exchange→open convention. Phase 146-05 / GAP-146-A.
func (a *App) OpenRemoteSessionURL(sessionID string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("daemon not connected")
	}
	return a.client.RemoteSessionOpenURL(sessionID)
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
		a.maybeNotifyWaiting(sessions) // Phase 167 / NTF-01..03
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
