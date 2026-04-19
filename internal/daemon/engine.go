package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/scottkw/agenthub/internal/pty"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/status"
)

// SessionEngine owns all session state: registry, backend, hub manager,
// tab names, CLI path overrides, and per-session status.
// It is intentionally free of Wails imports — callers supply an onStatus
// callback if they need event emission.
type SessionEngine struct {
	hostname          string // machine hostname, captured at startup
	opencodeTUIConfig string // path to managed opencode-tui.json (set at init)
	configDir         string // cached config dir for settings persistence

	registry *pty.SessionRegistry
	backend  pty.SessionBackend
	manager  *relay.HubManager

	mu          sync.RWMutex
	tabNames    map[string]string // sessionID -> display name
	sessionCLIs map[string]string // sessionID -> raw CLI name (e.g. "opencode")
	cliPaths    map[string]string // cli name -> custom path override

	startMinimized  bool  // persisted start-minimized preference
	autoCloseSession *bool // nil = default (true); persisted pointer

	statusMu        sync.RWMutex
	sessionStatuses map[string]status.SessionStatus // sessionID -> current status
}

// daemonConfigDir returns ~/.config/agenthub/, creating it if needed.
// Mirrors app.go configDir() — internal packages cannot import main.
func daemonConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "agenthub")
	_ = os.MkdirAll(dir, 0700)
	return dir
}

// ensureOpenCodeTUIConfig writes a managed tui.json that forces OpenCode's
// "system" theme. The system theme uses ANSI palette colors (0-15), making
// OpenCode respect xterm.js theme remapping. The file is overwritten on every
// call (content is a hardcoded constant, no user data). Returns the file path.
func ensureOpenCodeTUIConfig(dir string) string {
	path := filepath.Join(dir, "opencode-tui.json")
	content := []byte("{\"$schema\":\"https://opencode.ai/tui.json\",\"theme\":\"system\"}\n")
	_ = os.WriteFile(path, content, 0644)
	return path
}

// daemonSettings is the persisted settings structure.
type daemonSettings struct {
	CLIPaths         map[string]string `json:"cliPaths,omitempty"`
	StartMinimized   bool              `json:"startMinimized,omitempty"`
	AutoCloseSession *bool             `json:"autoCloseSession,omitempty"`
}

// settingsPath returns the path to settings.json inside the config dir.
func settingsPath(dir string) string {
	return filepath.Join(dir, "settings.json")
}

// knownShells maps basenames of common shell interpreters. A stored CLI path
// whose basename is a shell and whose key is NOT that shell is almost certainly
// stale/wrong (e.g. "claude" → "/bin/sh") and should be discarded on load.
var knownShells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "fish": true,
	"csh": true, "tcsh": true, "dash": true, "ksh": true,
}

// loadSettingsFromDisk reads settings.json and populates engine state.
// Missing file is not an error (first run).
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
	data, err := os.ReadFile(settingsPath(dir))
	if err != nil {
		return // file not found or unreadable — not an error
	}
	var s daemonSettings
	if json.Unmarshal(data, &s) != nil {
		return
	}
	e.mu.Lock()
	dirty := false
	if s.CLIPaths != nil {
		for k, v := range s.CLIPaths {
			base := filepath.Base(v)
			if knownShells[base] && base != k {
				log.Printf("daemon: dropping stale CLI path override %q=%q (shell mismatch)", k, v)
				delete(s.CLIPaths, k)
				dirty = true
				continue
			}
			e.cliPaths[k] = v
		}
	}
	e.startMinimized = s.StartMinimized
	e.autoCloseSession = s.AutoCloseSession
	if dirty {
		// Rewrite settings.json without the stale entries.
		e.saveSettingsToDisk()
	}
	e.mu.Unlock()
}

// saveSettingsToDisk writes current settings to settings.json.
// Caller holds e.mu.Lock().
func (e *SessionEngine) saveSettingsToDisk() {
	s := daemonSettings{
		CLIPaths:         e.cliPaths,
		StartMinimized:   e.startMinimized,
		AutoCloseSession: e.autoCloseSession,
	}
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	_ = os.WriteFile(settingsPath(e.configDir), data, 0600)
}

// NewSessionEngine creates a SessionEngine with all subsystems initialised.
func NewSessionEngine() *SessionEngine {
	hostname, _ := os.Hostname()
	cfgDir := daemonConfigDir()
	tuiConfig := ensureOpenCodeTUIConfig(cfgDir)
	e := &SessionEngine{
		hostname:          hostname,
		configDir:         cfgDir,
		opencodeTUIConfig: tuiConfig,
		registry:          pty.NewSessionRegistry(),
		backend:           pty.NewNativePTYBackend(),
		manager:           relay.NewHubManager(),
		tabNames:          make(map[string]string),
		sessionCLIs:       make(map[string]string),
		cliPaths:          make(map[string]string),
		sessionStatuses:   make(map[string]status.SessionStatus),
	}
	e.loadSettingsFromDisk(cfgDir)
	return e
}

// CreateSession spawns a new CLI session and returns its ID.
// args are passed to the CLI process; pass nil if no extra arguments are needed.
// onStatus is called on each status transition; pass nil if not needed.
// onExit is called with (sessionID, exitCode) when the process exits naturally; pass nil if not needed.
func (e *SessionEngine) CreateSession(ctx context.Context, cli, name, workDir string, args []string, cols, rows int, onStatus func(string, status.SessionStatus), onExit func(string, int)) (string, error) {
	cliPath := e.ResolveCLI(cli)

	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	// Per-agent environment configuration.
	var env []string
	if cli == "opencode" && e.opencodeTUIConfig != "" {
		env = append(env, "OPENCODE_TUI_CONFIG="+e.opencodeTUIConfig)
	}

	sess, err := e.backend.Create(ctx, pty.CreateRequest{
		CLI:     cliPath,
		Args:    args,
		Env:     env,
		Cols:    cols,
		Rows:    rows,
		WorkDir: workDir,
	})
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	e.registry.Add(sess)

	id := sess.ID
	resizeFn := func(cols, rows int) error {
		return e.backend.Resize(id, cols, rows)
	}
	hub := e.manager.Create(id, sess, sess, resizeFn)

	e.mu.Lock()
	e.tabNames[id] = name
	e.sessionCLIs[id] = cli // raw CLI name, NOT cliPath
	e.mu.Unlock()

	go status.Watch(hub, id, cli, func(sid string, s status.SessionStatus) {
		e.statusMu.Lock()
		e.sessionStatuses[sid] = s
		e.statusMu.Unlock()
		if onStatus != nil {
			onStatus(sid, s)
		}
	})

	// Watch for natural process exit (PTY EOF -> hub.Done closes).
	// Transitions session state to StateStopped and captures exit code.
	// Calls onExit callback if provided (used by API layer for web grace period per D-12).
	go func() {
		<-hub.Done() // blocks until PTY read loop returns EOF (D-07)
		exitCode := sess.WaitForExit()
		sess.SetState(pty.StateStopped)
		if onExit != nil {
			onExit(id, exitCode)
		}
	}()

	return id, nil
}

// ListSessions returns a snapshot of all registered sessions.
func (e *SessionEngine) ListSessions() []SessionInfo {
	sessions := e.registry.List()
	result := make([]SessionInfo, 0, len(sessions))

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, s := range sessions {
		state := "running"
		if s.GetState() == pty.StateStopped {
			state = "stopped"
		}
		name := e.tabNames[s.ID]

		// MC-04: populate viewer count from hub subscriber count.
		// manager.Get acquires HubManager.mu; SubscriberCount acquires hub.mu.
		// Both are safe to call while holding e.mu.RLock (no lock ordering conflict).
		viewerCount := 0
		if hub, ok := e.manager.Get(s.ID); ok {
			viewerCount = hub.SubscriberCount()
		}

		// Heuristic status from detector (running/idle/waiting/errored).
		heuristicStatus := string(status.StatusRunning) // conservative default
		e.statusMu.RLock()
		if hs, ok := e.sessionStatuses[s.ID]; ok {
			heuristicStatus = string(hs)
		}
		e.statusMu.RUnlock()

		var exitCodePtr *int
		var durationPtr *int
		if state == "stopped" {
			ec := s.ExitCode()
			exitCodePtr = &ec
			dur := int(time.Since(s.CreatedAt).Seconds())
			durationPtr = &dur
		}

		result = append(result, SessionInfo{
			ID:          s.ID,
			CLI:         s.CLI,
			Name:        name,
			State:       state,
			Status:      heuristicStatus,
			CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Hostname:    e.hostname,
			ViewerCount: viewerCount,
			ExitCode:    exitCodePtr,
			Duration:    durationPtr,
		})
	}
	return result
}

// RenameSession updates the display name of a session.
func (e *SessionEngine) RenameSession(id, name string) error {
	if _, ok := e.registry.Get(id); !ok {
		return fmt.Errorf("session %q not found", id)
	}
	e.mu.Lock()
	e.tabNames[id] = name
	e.mu.Unlock()
	return nil
}

// KillSession terminates the session and removes it from all registries.
func (e *SessionEngine) KillSession(id string) error {
	if err := e.backend.Kill(id); err != nil {
		return fmt.Errorf("kill session: %w", err)
	}
	e.manager.Remove(id)
	e.registry.Remove(id)

	e.mu.Lock()
	delete(e.tabNames, id)
	delete(e.sessionCLIs, id)
	e.mu.Unlock()

	e.statusMu.Lock()
	delete(e.sessionStatuses, id)
	e.statusMu.Unlock()

	return nil
}

// GetSessionStatus returns the current heuristic status of the session.
// Returns "running" if the session is not found (conservative default).
func (e *SessionEngine) GetSessionStatus(sessionID string) string {
	e.statusMu.RLock()
	s, ok := e.sessionStatuses[sessionID]
	e.statusMu.RUnlock()
	if !ok {
		return string(status.StatusRunning)
	}
	return string(s)
}

// ResolveCLI returns the executable path for the named CLI.
// Checks custom overrides first, then returns the name as-is.
func (e *SessionEngine) ResolveCLI(name string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if path, ok := e.cliPaths[name]; ok {
		return path
	}
	return name
}

// UpdateCLIPath stores a custom executable path for the named CLI.
// The path must exist on disk. Persists to settings.json.
func (e *SessionEngine) UpdateCLIPath(name, path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("custom CLI path %q: %w", path, err)
	}
	e.mu.Lock()
	e.cliPaths[name] = path
	e.saveSettingsToDisk()
	e.mu.Unlock()
	return nil
}

// GetCLIPaths returns a copy of the current CLI path overrides map.
func (e *SessionEngine) GetCLIPaths() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]string, len(e.cliPaths))
	for k, v := range e.cliPaths {
		out[k] = v
	}
	return out
}

// GetStartMinimized returns the persisted start-minimized preference.
func (e *SessionEngine) GetStartMinimized() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.startMinimized
}

// SetStartMinimized updates and persists the start-minimized preference.
func (e *SessionEngine) SetStartMinimized(val bool) {
	e.mu.Lock()
	e.startMinimized = val
	e.saveSettingsToDisk()
	e.mu.Unlock()
}

// GetAutoCloseSession returns the auto-close-on-exit preference.
// Returns true when the setting is absent (nil pointer = default enabled per D-11).
func (e *SessionEngine) GetAutoCloseSession() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.autoCloseSession == nil {
		return true // default: enabled
	}
	return *e.autoCloseSession
}

// SetAutoCloseSession updates and persists the auto-close-on-exit preference.
func (e *SessionEngine) SetAutoCloseSession(val bool) {
	e.mu.Lock()
	e.autoCloseSession = &val
	e.saveSettingsToDisk()
	e.mu.Unlock()
}

// Manager returns the HubManager (needed by webserver and relay server).
func (e *SessionEngine) Manager() *relay.HubManager {
	return e.manager
}

// Registry returns the session registry (needed for Get lookups).
func (e *SessionEngine) Registry() *pty.SessionRegistry {
	return e.registry
}

// Backend returns the session backend (needed for resize in relay).
func (e *SessionEngine) Backend() pty.SessionBackend {
	return e.backend
}

// NotifyThemeChange signals all active OpenCode sessions to re-query the
// terminal palette. On POSIX this sends SIGUSR2; on Windows this is a no-op.
// Errors on individual sessions are logged and do not abort the broadcast.
// Safe to call when no opencode sessions exist (returns nil).
func (e *SessionEngine) NotifyThemeChange(ctx context.Context) error {
	sessions := e.registry.List()

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, sess := range sessions {
		if e.sessionCLIs[sess.ID] != "opencode" {
			continue
		}
		if sess.GetState() != pty.StateRunning {
			continue
		}
		if err := signalThemeChange(sess); err != nil {
			log.Printf("[warn] NotifyThemeChange: session %s: %v", sess.ID, err)
		}
	}
	return nil
}
