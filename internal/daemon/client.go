package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/scottkw/agenthub/internal/tailnet"
)

// DaemonClient is a typed Go client that communicates with the daemon API
// over a Unix socket.
type DaemonClient struct {
	http *http.Client
	base string
}

// NewDaemonClient creates a DaemonClient that dials the given Unix socket path.
// All HTTP requests use http://daemon as the base URL — the custom transport
// rewrites the actual network connection to the Unix socket.
func NewDaemonClient(socketPath string) *DaemonClient {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &DaemonClient{
		http: &http.Client{Transport: transport},
		base: "http://daemon",
	}
}

// Health returns nil if the daemon is reachable and returns {"status":"ok"}.
func (c *DaemonClient) Health() error {
	var h HealthResponse
	return c.doJSON(http.MethodGet, "/health", nil, &h)
}

// GetDaemonVersion returns the version reported by the daemon's health endpoint.
// Returns empty string if the daemon doesn't report a version (pre-v2.1.2).
func (c *DaemonClient) GetDaemonVersion() string {
	var h HealthResponse
	if err := c.doJSON(http.MethodGet, "/health", nil, &h); err != nil {
		return ""
	}
	return h.Version
}

// ListSessions returns all current sessions.
func (c *DaemonClient) ListSessions() ([]SessionInfo, error) {
	var sessions []SessionInfo
	if err := c.doJSON(http.MethodGet, "/sessions", nil, &sessions); err != nil {
		return nil, err
	}
	if sessions == nil {
		sessions = []SessionInfo{}
	}
	return sessions, nil
}

// CreateSession creates a new session and returns its ID.
// args are passed to the CLI process; pass nil if no extra arguments are needed.
// cols and rows specify the initial PTY dimensions; pass 0 for defaults (80x24).
func (c *DaemonClient) CreateSession(cli, name, workDir string, args []string, cols, rows int) (string, error) {
	req := CreateRequest{CLI: cli, Name: name, WorkDir: workDir, Args: args, Cols: cols, Rows: rows}
	var resp CreateResponse
	if err := c.doJSON(http.MethodPost, "/sessions", req, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

// KillSession terminates the given session.
func (c *DaemonClient) KillSession(id string) error {
	return c.doJSON(http.MethodDelete, "/sessions/"+id, nil, nil)
}

// RenameSession changes the display name of a session.
func (c *DaemonClient) RenameSession(id, name string) error {
	return c.doJSON(http.MethodPatch, "/sessions/"+id+"/name", RenameRequest{Name: name}, nil)
}

// GetSessionStatus returns the current status string for the given session.
func (c *DaemonClient) GetSessionStatus(id string) (string, error) {
	var resp StatusResponse
	if err := c.doJSON(http.MethodGet, "/sessions/"+id+"/status", nil, &resp); err != nil {
		return "", err
	}
	return resp.Status, nil
}

// GetCLIPaths returns the current CLI path override map.
func (c *DaemonClient) GetCLIPaths() (map[string]string, error) {
	var paths map[string]string
	if err := c.doJSON(http.MethodGet, "/settings/cli-paths", nil, &paths); err != nil {
		return nil, err
	}
	return paths, nil
}

// ListShells returns the daemon's discovery of installed shells via
// GET /shells. The returned slice mirrors the wire body's `shells` field —
// always non-nil; empty slice (not nil) when no shells are installed.
func (c *DaemonClient) ListShells() ([]DetectedShell, error) {
	var resp ShellsResponse
	if err := c.doJSON(http.MethodGet, "/shells", nil, &resp); err != nil {
		return nil, err
	}
	if resp.Shells == nil {
		resp.Shells = []DetectedShell{}
	}
	return resp.Shells, nil
}

// UpdateCLIPath sets a custom executable path for the named CLI.
func (c *DaemonClient) UpdateCLIPath(name, path string) error {
	return c.doJSON(http.MethodPatch, "/settings/cli-paths/"+name, UpdateCLIPathRequest{Path: path}, nil)
}

// GetStartMinimized returns the persisted start-minimized preference.
func (c *DaemonClient) GetStartMinimized() (bool, error) {
	var resp map[string]bool
	if err := c.doJSON(http.MethodGet, "/settings/start-minimized", nil, &resp); err != nil {
		return false, err
	}
	return resp["startMinimized"], nil
}

// SetStartMinimized persists the start-minimized preference.
func (c *DaemonClient) SetStartMinimized(val bool) error {
	return c.doJSON(http.MethodPatch, "/settings/start-minimized",
		map[string]bool{"startMinimized": val}, nil)
}

// GetShellWebShareWarned returns the persisted shell-web-share-warned flag.
// Phase 101 SHELL-08.
func (c *DaemonClient) GetShellWebShareWarned() (bool, error) {
	var resp map[string]bool
	if err := c.doJSON(http.MethodGet, "/settings/shell-web-share-warned", nil, &resp); err != nil {
		return false, err
	}
	return resp["value"], nil
}

// SetShellWebShareWarned persists the shell-web-share-warned flag.
// Phase 101 SHELL-08.
func (c *DaemonClient) SetShellWebShareWarned(val bool) error {
	return c.doJSON(http.MethodPatch, "/settings/shell-web-share-warned",
		map[string]bool{"value": val}, nil)
}

// GetShellPath returns the persisted shell binary path from the daemon.
// When no path has been configured, the daemon returns the platform default.
// Phase 107 SHELL-11.
func (c *DaemonClient) GetShellPath() (string, error) {
	var resp map[string]string
	if err := c.doJSON(http.MethodGet, "/settings/shell-path", nil, &resp); err != nil {
		return "", err
	}
	return resp["value"], nil
}

// SetShellPath persists the shell binary path via the daemon API. An empty
// path clears the override (restores platform default). A non-empty path that
// does not exist or is not executable causes the daemon to return 400, which
// is surfaced as an error here. Phase 107 SHELL-11.
func (c *DaemonClient) SetShellPath(path string) error {
	return c.doJSON(http.MethodPatch, "/settings/shell-path",
		map[string]string{"value": path}, nil)
}

// GetAutoCloseSession returns the auto-close-on-exit preference.
func (c *DaemonClient) GetAutoCloseSession() (bool, error) {
	var resp map[string]bool
	if err := c.doJSON(http.MethodGet, "/settings/auto-close-session", nil, &resp); err != nil {
		return true, err // default true on error
	}
	return resp["autoCloseSession"], nil
}

// SetAutoCloseSession persists the auto-close-on-exit preference.
func (c *DaemonClient) SetAutoCloseSession(val bool) error {
	return c.doJSON(http.MethodPatch, "/settings/auto-close-session",
		map[string]bool{"autoCloseSession": val}, nil)
}

// GetPluginSettings fetches the persisted plugin enable/disable preferences.
func (c *DaemonClient) GetPluginSettings() (PluginSettings, error) {
	var resp PluginSettings
	if err := c.doJSON(http.MethodGet, "/settings/plugins", nil, &resp); err != nil {
		return PluginSettings{}, err
	}
	return resp, nil
}

// SetPluginSettings persists the plugin enable/disable preferences via the
// daemon API. The full PluginSettings struct is sent in the body (full
// replace semantic; the route is PATCH for consistency with surrounding
// settings routes, not for partial-update semantics).
func (c *DaemonClient) SetPluginSettings(s PluginSettings) error {
	return c.doJSON(http.MethodPatch, "/settings/plugins", s, nil)
}

// SetSearchConfig persists ONLY the SearchConfig sub-key of PluginSettings via
// the daemon API. Phase 94-07 WR-03 gap closure — the find bar must not race
// PluginsSection's stale local edit buffer by writing the full PluginSettings
// from a snapshot prop. This sub-key writer routes to the engine's
// SetSearchConfig method which mutates only e.pluginSettings.SearchConfig
// under the engine mutex.
func (c *DaemonClient) SetSearchConfig(cfg SearchConfig) error {
	return c.doJSON(http.MethodPatch, "/settings/search-config", cfg, nil)
}

// SetWebLinksConfig persists ONLY the WebLinksConfig sub-key of PluginSettings
// via the daemon API. Phase 95 LNK-05 / LNK-06 — mirrors Phase 94-07
// SetSearchConfig. Routes to engine.SetWebLinksConfig which mutates only
// e.pluginSettings.WebLinksConfig under the engine mutex, preserving
// PluginsSection's edit buffer for unrelated plugin booleans.
func (c *DaemonClient) SetWebLinksConfig(cfg WebLinksConfig) error {
	return c.doJSON(http.MethodPatch, "/settings/web-links-config", cfg, nil)
}

// SetImageConfig persists ONLY the ImageConfig sub-key of PluginSettings via
// the daemon API. Phase 96 IMG-02 — mirrors Phase 95 SetWebLinksConfig and
// Phase 94-07 SetSearchConfig. Routes to engine.SetImageConfig which mutates
// only e.pluginSettings.ImageConfig under the engine mutex, preserving
// PluginsSection's edit buffer for unrelated plugin booleans.
//
// The daemon validates StorageLimit is in [1, 1000] MB; out-of-range values
// surface as a non-nil error from this call (HTTP 400 from the handler).
func (c *DaemonClient) SetImageConfig(cfg ImageConfig) error {
	return c.doJSON(http.MethodPatch, "/settings/image-config", cfg, nil)
}

// GetRelayPort returns the TCP port the daemon's relay server is listening on.
func (c *DaemonClient) GetRelayPort() (int, error) {
	var resp RelayPortResponse
	if err := c.doJSON(http.MethodGet, "/relay-port", nil, &resp); err != nil {
		return 0, err
	}
	return resp.Port, nil
}

// StartWebServer tells the daemon to start the web server.
// mode is "tailscale" or "local"; password is non-empty for local mode.
func (c *DaemonClient) StartWebServer(ip string, port int, fqdn, mode, password string) (string, error) {
	req := WebServerStartRequest{IP: ip, Port: port, FQDN: fqdn, Mode: mode, Password: password}
	var resp WebServerStartResponse
	if err := c.doJSON(http.MethodPost, "/webserver/start", req, &resp); err != nil {
		return "", err
	}
	return resp.URL, nil
}

// GetLocalNetworkPassword returns the generated local-mode password from the daemon.
// Returns empty string when the daemon is in Tailscale mode (no password needed).
func (c *DaemonClient) GetLocalNetworkPassword() (string, error) {
	var resp map[string]string
	if err := c.doJSON(http.MethodGet, "/webserver/local-password", nil, &resp); err != nil {
		return "", err
	}
	return resp["password"], nil
}

// StopWebServer tells the daemon to stop the Tailscale web server.
func (c *DaemonClient) StopWebServer() error {
	return c.doJSON(http.MethodPost, "/webserver/stop", nil, nil)
}

// GetWebServerStatus returns the current web server state from the daemon.
func (c *DaemonClient) GetWebServerStatus() (WebServerStatusResponse, error) {
	var resp WebServerStatusResponse
	if err := c.doJSON(http.MethodGet, "/webserver/status", nil, &resp); err != nil {
		return WebServerStatusResponse{}, err
	}
	return resp, nil
}

// ShutdownDaemon sends POST /shutdown to terminate the daemon process.
// Connection-reset errors are expected (daemon exits before response completes)
// and are treated as success.
func (c *DaemonClient) ShutdownDaemon() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/shutdown", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		// Connection reset is expected — daemon exited. Treat as success.
		return nil
	}
	resp.Body.Close()
	return nil
}

// ToggleWebServing enables or disables web serving for a session.
func (c *DaemonClient) ToggleWebServing(sessionID string, enabled bool) error {
	return c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/web-serve", WebServeRequest{Enabled: enabled}, nil)
}

// IssueCapabilities mints the read + read,write capability pair for a
// web-enabled session (D-07). Returns the URLs and single-use join codes
// (D-09) for each. Called by the GUI/CLI/TUI after toggle-on.
func (c *DaemonClient) IssueCapabilities(sessionID string) (IssueCapabilitiesResponse, error) {
	var resp IssueCapabilitiesResponse
	err := c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/capabilities", nil, &resp)
	return resp, err
}

// ExchangeJoinCode consumes a single-use join code and returns the
// capability-bearing URL the client should follow.
func (c *DaemonClient) ExchangeJoinCode(code string) (string, error) {
	var resp ExchangeJoinCodeResponse
	if err := c.doJSON(http.MethodPost, "/join/exchange", ExchangeJoinCodeRequest{Code: code}, &resp); err != nil {
		return "", err
	}
	return resp.URL, nil
}

// RegenerateSigningKey rotates the HMAC signing key (D-16 panic button).
// All previously-issued capabilities become invalid globally.
func (c *DaemonClient) RegenerateSigningKey() error {
	return c.doJSON(http.MethodPost, "/capability/regenerate-key", nil, nil)
}

// NotifyThemeChange tells the daemon to signal active OpenCode sessions
// to re-query the terminal palette. Returns nil on success (204 No Content).
func (c *DaemonClient) NotifyThemeChange() error {
	return c.doJSON(http.MethodPost, "/theme/notify", nil, nil)
}

// ListTailnetPeers returns discovered tailnet peers running AgentHub.
func (c *DaemonClient) ListTailnetPeers() ([]tailnet.Peer, error) {
	var peers []tailnet.Peer
	if err := c.doJSON(http.MethodGet, "/tailnet/peers", nil, &peers); err != nil {
		return nil, err
	}
	if peers == nil {
		peers = []tailnet.Peer{}
	}
	return peers, nil
}

// doJSON is a shared request/response helper. It marshals body (if non-nil),
// sends the request, checks the status code, and decodes result (if non-nil).
func (c *DaemonClient) doJSON(method, path string, body, result any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.base+path, reqBody)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon API %s %s: status %d: %s", method, path, resp.StatusCode, string(b))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
