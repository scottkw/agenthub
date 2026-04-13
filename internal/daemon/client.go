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

// UpdateCLIPath sets a custom executable path for the named CLI.
func (c *DaemonClient) UpdateCLIPath(name, path string) error {
	return c.doJSON(http.MethodPatch, "/settings/cli-paths/"+name, UpdateCLIPathRequest{Path: path}, nil)
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
