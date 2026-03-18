package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/agenthub/agenthub/internal/pty"
	"github.com/agenthub/agenthub/internal/relay"
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
}

// App holds all application state and exposes the Wails-bound methods.
type App struct {
	ctx      context.Context
	registry *pty.SessionRegistry
	backend  pty.SessionBackend
	manager  *relay.HubManager
	server   *relay.Server
	listener net.Listener
	trayInit bool // true once initTray has been called

	mu        sync.RWMutex
	tabNames  map[string]string // sessionID -> display name
	cliPaths  map[string]string // cli name -> custom path override
	webServer *webserver.WebServer
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

	// Start system tray icon (non-blocking, macOS NSStatusBar).
	a.initTray()
	a.trayInit = true
}

// shutdown is called when the Wails app is about to exit.
func (a *App) shutdown(_ context.Context) {
	// Remove the system tray icon before cleaning up other resources.
	if a.trayInit {
		a.cleanupTray()
	}
	a.manager.Shutdown()
	if a.listener != nil {
		_ = a.listener.Close()
	}
	a.mu.Lock()
	ws := a.webServer
	a.mu.Unlock()
	if ws != nil {
		_ = ws.Stop()
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

// webPasswordPath returns the path to the persisted web password hash file.
func webPasswordPath() string {
	return filepath.Join(configDir(), "web_password")
}

// SetWebPassword sets the dashboard password for web serving. The bcrypt hash
// is persisted to disk so it survives restarts. Lazily initialises the WebServer
// if it has not been created yet (needed for password setup before server start).
func (a *App) SetWebPassword(password string) error {
	a.mu.Lock()
	ws := a.webServer
	a.mu.Unlock()

	if ws == nil {
		// Create lazily so password can be set before StartWebServer is called.
		newWS, err := webserver.NewWebServer(webserver.Config{
			BindIP:    "127.0.0.1",
			Port:      7443,
			ConfigDir: configDir(),
		}, a.manager)
		if err != nil {
			return fmt.Errorf("SetWebPassword: init server: %w", err)
		}
		a.mu.Lock()
		a.webServer = newWS
		a.mu.Unlock()
		ws = newWS
	}

	if err := ws.SetPassword(password); err != nil {
		return fmt.Errorf("SetWebPassword: %w", err)
	}

	// Persist the bcrypt hash to disk.
	hash, err := webserver.HashPassword(password)
	if err != nil {
		return fmt.Errorf("SetWebPassword: hash: %w", err)
	}
	if err := os.WriteFile(webPasswordPath(), hash, 0600); err != nil {
		return fmt.Errorf("SetWebPassword: persist: %w", err)
	}
	return nil
}

// IsWebPasswordSet returns true if a web serving password has been configured
// (either in memory or via the persisted hash on disk).
func (a *App) IsWebPasswordSet() bool {
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws != nil && ws.IsPasswordSet() {
		return true
	}
	_, err := os.Stat(webPasswordPath())
	return err == nil
}

// GetNetworkInterfaces returns all active non-loopback IPv4 network interfaces,
// including Tailscale detection. Used to populate the interface dropdown in Settings.
func (a *App) GetNetworkInterfaces() []webserver.NetworkInterface {
	ifaces, err := webserver.ListInterfaces()
	if err != nil {
		return []webserver.NetworkInterface{}
	}
	return ifaces
}

// StartWebServer creates (or re-creates) the WebServer bound to bindIP:port,
// loads the persisted password hash, and begins serving. Returns an error if no
// password has been set (web serving is gated behind password setup).
func (a *App) StartWebServer(bindIP string, port int) error {
	if !a.IsWebPasswordSet() {
		return fmt.Errorf("web serving requires a password — set one in Settings first")
	}

	// Stop any running server before creating a new one.
	a.mu.Lock()
	oldWS := a.webServer
	a.mu.Unlock()
	if oldWS != nil {
		_ = oldWS.Stop()
	}

	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:    bindIP,
		Port:      port,
		ConfigDir: configDir(),
	}, a.manager)
	if err != nil {
		return fmt.Errorf("StartWebServer: create: %w", err)
	}

	// Load persisted password hash.
	if hash, err := os.ReadFile(webPasswordPath()); err == nil {
		ws.LoadPasswordHash(hash)
	}

	if err := ws.Start(); err != nil {
		return fmt.Errorf("StartWebServer: start: %w", err)
	}

	a.mu.Lock()
	a.webServer = ws
	a.mu.Unlock()
	return nil
}

// StopWebServer stops the web server and clears the webServer field.
func (a *App) StopWebServer() error {
	a.mu.Lock()
	ws := a.webServer
	a.webServer = nil
	a.mu.Unlock()
	if ws == nil {
		return nil
	}
	return ws.Stop()
}

// ToggleWebServing enables or disables web serving for a specific session.
// Returns an error if the web server is not running.
func (a *App) ToggleWebServing(sessionID string, enabled bool) error {
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws == nil {
		return fmt.Errorf("web server is not running — start it in Settings first")
	}
	if enabled {
		ws.EnableSession(sessionID)
	} else {
		ws.DisableSession(sessionID)
	}
	return nil
}

// GenerateSessionToken generates a one-time token for the session and returns
// the full shareable URL (https://bindIP:port/sessions/{id}?token=xxx).
func (a *App) GenerateSessionToken(sessionID string) (string, error) {
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws == nil {
		return "", fmt.Errorf("web server is not running")
	}
	tok, err := ws.CreateToken(sessionID)
	if err != nil {
		return "", fmt.Errorf("GenerateSessionToken: %w", err)
	}
	baseURL := ws.BaseURL()
	return fmt.Sprintf("%s/sessions/%s?token=%s", baseURL, sessionID, tok), nil
}

// GetWebServerURL returns the base HTTPS URL of the running web server,
// or an empty string if the server is not running.
func (a *App) GetWebServerURL() string {
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws == nil {
		return ""
	}
	return ws.BaseURL()
}

// GetCACertPath returns the file path to the CA certificate used by the web server.
// This path is shown in Settings so users can install it in their OS trust store.
func (a *App) GetCACertPath() string {
	return webserver.ExportCACertPath(configDir())
}

// IsWebServerRunning returns true if the web server has been started and its
// listener is active.
func (a *App) IsWebServerRunning() bool {
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws == nil {
		return false
	}
	return ws.Addr() != ""
}

// GetSessionQRCode generates a QR code for the web-served session URL and
// returns it as a base64-encoded PNG string. The QR encodes the session URL
// (https://bindIP:port/sessions/{id}). Returns an error if the web server is
// not running.
func (a *App) GetSessionQRCode(sessionID string) (string, error) {
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws == nil {
		return "", fmt.Errorf("web server not running")
	}
	url := fmt.Sprintf("%s/sessions/%s", ws.BaseURL(), sessionID)
	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("GetSessionQRCode: encode: %w", err)
	}
	return base64.StdEncoding.EncodeToString(png), nil
}
