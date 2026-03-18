package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/agenthub/agenthub/internal/pty"
	"github.com/agenthub/agenthub/internal/relay"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SessionInfo is the JSON-serialisable representation of a session returned by ListSessions.
type SessionInfo struct {
	ID        string `json:"id"`
	CLI       string `json:"cli"`
	Name      string `json:"name"`
	State     string `json:"state"`
	CreatedAt string `json:"createdAt"`
}

// App holds all application state and exposes the Wails-bound methods.
type App struct {
	ctx      context.Context
	registry *pty.SessionRegistry
	backend  pty.SessionBackend
	manager  *relay.HubManager
	server   *relay.Server
	listener net.Listener
	trayEnd  func() // cleanup function returned by systray.RunWithExternalLoop

	mu       sync.RWMutex
	tabNames map[string]string // sessionID -> display name
	cliPaths map[string]string // cli name -> custom path override
}

// NewApp creates a new App with all subsystems initialised but not yet started.
func NewApp() *App {
	registry := pty.NewSessionRegistry()
	backend := pty.NewNativePTYBackend()
	manager := relay.NewHubManager()
	server := relay.NewServer(manager, backend)

	return &App{
		registry: registry,
		backend:  backend,
		manager:  manager,
		server:   server,
		tabNames: make(map[string]string),
		cliPaths: make(map[string]string),
	}
}

// startup is called when Wails initialises the app.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Allocate listener synchronously to avoid a race between GetRelayPort and Serve.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		// Startup failure is fatal — Wails will report it.
		panic(fmt.Sprintf("agenthub: relay listener: %v", err))
	}
	a.listener = ln
	go func() { _ = http.Serve(ln, a.server) }()

	// Start system tray (non-blocking via RunWithExternalLoop).
	a.initTray()
}

// shutdown is called when the Wails app is about to exit.
func (a *App) shutdown(_ context.Context) {
	// Stop the system tray loop before cleaning up other resources.
	if a.trayEnd != nil {
		a.trayEnd()
	}
	a.manager.Shutdown()
	if a.listener != nil {
		_ = a.listener.Close()
	}
}

// beforeClose hides the window instead of quitting so the app stays alive in
// the system tray (tray.go provides the tray icon and Quit menu item).
// When called outside a Wails context (e.g., in unit tests), the window hide
// is skipped safely — sessions are unaffected in both cases.
func (a *App) beforeClose(ctx context.Context) bool {
	// Wails stores the frontend under the "frontend" key; skip the call when
	// running outside the Wails event loop (tests, CLI helpers).
	if ctx.Value("frontend") != nil {
		runtime.WindowHide(ctx)
	}
	return true // prevent the default quit behaviour
}

// --- Wails-bound methods ---

// CreateSession spawns a new CLI session and returns its ID.
func (a *App) CreateSession(cli, name string) (string, error) {
	// Resolve CLI path: custom override → PATH lookup.
	cliPath := a.resolveCLI(cli)

	sess, err := a.backend.Create(a.ctx, pty.CreateRequest{
		CLI:  cliPath,
		Cols: 80,
		Rows: 24,
	})
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	a.registry.Add(sess)

	// Build resize function for the Hub.
	id := sess.ID
	resizeFn := func(cols, rows int) error {
		return a.backend.Resize(id, cols, rows)
	}
	a.manager.Create(id, sess, sess, resizeFn)

	a.mu.Lock()
	a.tabNames[id] = name
	a.mu.Unlock()

	return id, nil
}

// ListSessions returns a snapshot of all registered sessions.
func (a *App) ListSessions() []SessionInfo {
	sessions := a.registry.List()
	result := make([]SessionInfo, 0, len(sessions))

	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, s := range sessions {
		state := "running"
		if s.State == pty.StateStopped {
			state = "stopped"
		}
		name := a.tabNames[s.ID]
		result = append(result, SessionInfo{
			ID:        s.ID,
			CLI:       s.CLI,
			Name:      name,
			State:     state,
			CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return result
}

// RenameSession updates the display name of a session.
func (a *App) RenameSession(id, name string) error {
	if _, ok := a.registry.Get(id); !ok {
		return fmt.Errorf("session %q not found", id)
	}
	a.mu.Lock()
	a.tabNames[id] = name
	a.mu.Unlock()
	return nil
}

// KillSession terminates the session and removes it from all registries.
func (a *App) KillSession(id string) error {
	if err := a.backend.Kill(id); err != nil {
		return fmt.Errorf("kill session: %w", err)
	}
	a.manager.Remove(id)
	a.registry.Remove(id)

	a.mu.Lock()
	delete(a.tabNames, id)
	a.mu.Unlock()
	return nil
}

// DetectCLIs returns the list of supported AI coding CLIs found on PATH.
func (a *App) DetectCLIs() []pty.DetectedCLI {
	return pty.DetectCLIs()
}

// GetRelayPort returns the TCP port the relay HTTP server is listening on.
func (a *App) GetRelayPort() int {
	if a.listener == nil {
		return 0
	}
	return a.listener.Addr().(*net.TCPAddr).Port
}

// UpdateCLIPath stores a custom executable path for the named CLI.
// The path must exist on disk.
func (a *App) UpdateCLIPath(name, path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("custom CLI path %q: %w", path, err)
	}
	a.mu.Lock()
	a.cliPaths[name] = path
	a.mu.Unlock()
	return nil
}

// resolveCLI returns the executable path for the named CLI.
// It checks the custom overrides map first, then falls back to the name as-is
// (os/exec will PATH-search it at Create time).
func (a *App) resolveCLI(name string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if path, ok := a.cliPaths[name]; ok {
		return path
	}
	return name
}
