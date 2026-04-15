package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

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

	registry *pty.SessionRegistry
	backend  pty.SessionBackend
	manager  *relay.HubManager

	mu          sync.RWMutex
	tabNames    map[string]string // sessionID -> display name
	sessionCLIs map[string]string // sessionID -> raw CLI name (e.g. "opencode")
	cliPaths    map[string]string // cli name -> custom path override

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

// NewSessionEngine creates a SessionEngine with all subsystems initialised.
func NewSessionEngine() *SessionEngine {
	hostname, _ := os.Hostname()
	tuiConfig := ensureOpenCodeTUIConfig(daemonConfigDir())
	return &SessionEngine{
		hostname:          hostname,
		opencodeTUIConfig: tuiConfig,
		registry:          pty.NewSessionRegistry(),
		backend:           pty.NewNativePTYBackend(),
		manager:           relay.NewHubManager(),
		tabNames:          make(map[string]string),
		sessionCLIs:       make(map[string]string),
		cliPaths:          make(map[string]string),
		sessionStatuses:   make(map[string]status.SessionStatus),
	}
}

// CreateSession spawns a new CLI session and returns its ID.
// args are passed to the CLI process; pass nil if no extra arguments are needed.
// onStatus is called on each status transition; pass nil if not needed.
func (e *SessionEngine) CreateSession(ctx context.Context, cli, name, workDir string, args []string, cols, rows int, onStatus func(string, status.SessionStatus)) (string, error) {
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
		if s.State == pty.StateStopped {
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

		result = append(result, SessionInfo{
			ID:          s.ID,
			CLI:         s.CLI,
			Name:        name,
			State:       state,
			CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Hostname:    e.hostname,
			ViewerCount: viewerCount,
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
// The path must exist on disk.
func (e *SessionEngine) UpdateCLIPath(name, path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("custom CLI path %q: %w", path, err)
	}
	e.mu.Lock()
	e.cliPaths[name] = path
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
		if sess.State != pty.StateRunning {
			continue
		}
		if err := signalThemeChange(sess); err != nil {
			log.Printf("[warn] NotifyThemeChange: session %s: %v", sess.ID, err)
		}
	}
	return nil
}
