package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/scottkw/agenthub/internal/files"
	"github.com/scottkw/agenthub/internal/tailnet"
)

// DaemonClient is a typed Go client that communicates with the daemon API
// over a Unix socket or Windows named pipe.
type DaemonClient struct {
	http *http.Client
	base string
}

// NewDaemonClient creates a DaemonClient that dials the given daemon socket path.
// All HTTP requests use http://daemon as the base URL — the custom transport
// rewrites the actual network connection to the platform daemon socket.
func NewDaemonClient(socketPath string) *DaemonClient {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialDaemonSocket(ctx, socketPath)
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

// GetSessionTailLines returns the last n plain-text lines from the session's
// scrollback buffer, with ANSI escape sequences and framing bytes stripped.
// Phase 132 / CARD-07.
func (c *DaemonClient) GetSessionTailLines(id string, n int) ([]string, error) {
	var resp TailLinesResponse
	if err := c.doJSON(http.MethodGet, fmt.Sprintf("/sessions/%s/tail?n=%d", id, n), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Lines, nil
}

// GetSessionStyledTailLines returns the last n styled-cell lines from the
// session's scrollback buffer via GET /sessions/{id}/styled-tail.
// Phase 139 / CARD-05.
func (c *DaemonClient) GetSessionStyledTailLines(id string, n int) ([][]StyledSpan, error) {
	var resp StyledTailLinesResponse
	if err := c.doJSON(http.MethodGet, fmt.Sprintf("/sessions/%s/styled-tail?n=%d", id, n), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Lines, nil
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

// GetNotifyOnWaiting returns the persisted native-notification-on-waiting
// preference. Phase 167 NTF-04.
func (c *DaemonClient) GetNotifyOnWaiting() (bool, error) {
	var resp map[string]bool
	if err := c.doJSON(http.MethodGet, "/settings/notify-on-waiting", nil, &resp); err != nil {
		return false, err
	}
	return resp["notifyOnWaiting"], nil
}

// SetNotifyOnWaiting persists the native-notification-on-waiting preference.
// Phase 167 NTF-04.
func (c *DaemonClient) SetNotifyOnWaiting(val bool) error {
	return c.doJSON(http.MethodPatch, "/settings/notify-on-waiting",
		map[string]bool{"notifyOnWaiting": val}, nil)
}

// GetStayOnHubAfterCreate returns the persisted "stay on Hub after creating
// a session" preference. Phase 168 UX-01.
func (c *DaemonClient) GetStayOnHubAfterCreate() (bool, error) {
	var resp map[string]bool
	if err := c.doJSON(http.MethodGet, "/settings/stay-on-hub-after-create", nil, &resp); err != nil {
		return false, err
	}
	return resp["stayOnHubAfterCreate"], nil
}

// SetStayOnHubAfterCreate persists the "stay on Hub after creating a
// session" preference. Phase 168 UX-01.
func (c *DaemonClient) SetStayOnHubAfterCreate(val bool) error {
	return c.doJSON(http.MethodPatch, "/settings/stay-on-hub-after-create",
		map[string]bool{"stayOnHubAfterCreate": val}, nil)
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

// GetShellWebShareWarningEnabled returns the warning-enabled master switch.
// Phase 150 SET-01.
func (c *DaemonClient) GetShellWebShareWarningEnabled() (bool, error) {
	var resp map[string]bool
	if err := c.doJSON(http.MethodGet, "/settings/shell-web-share-warning-enabled", nil, &resp); err != nil {
		return false, err
	}
	return resp["value"], nil
}

// SetShellWebShareWarningEnabled persists the warning-enabled master switch.
// When val=true the engine atomically resets shellWebShareWarned (D-03 re-arm).
// Phase 150 SET-01.
func (c *DaemonClient) SetShellWebShareWarningEnabled(val bool) error {
	return c.doJSON(http.MethodPatch, "/settings/shell-web-share-warning-enabled",
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

// DisconnectViewers force-closes every remote (Origin=="web") viewer
// currently connected to the session, without revoking the capability or
// affecting web-serve state (D-06). Phase 168 / FIX-02, #117.
func (c *DaemonClient) DisconnectViewers(sessionID string) error {
	return c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/disconnect-viewers", nil, nil)
}

// SetSessionBrowse sets the per-session browse toggle for a session.
// Phase 137 / SHARE-03. Mirrors ToggleWebServing but routes to the engine's
// per-session browse map (not a global flag). Toggle-off clears grants on the
// daemon side (stale-cap threat mitigation per SHARE-05).
func (c *DaemonClient) SetSessionBrowse(sessionID string, enabled bool) error {
	return c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/browse",
		SessionBrowseRequest{Enabled: enabled}, nil)
}

// SetSessionFunnel enables or disables Tailscale Funnel for a session.
// Phase 165 / FNL-01. Mirrors ToggleWebServing / SetSessionBrowse — thin
// delegation, no Funnel logic here. expiresIn is the auto-expiry duration in
// seconds; 0 = no auto-expiry (FNL-07). The response body (FunnelURL) is
// discarded here; the GUI retrieves it from IssueCapabilities.
func (c *DaemonClient) SetSessionFunnel(sessionID string, enabled bool, expiresIn int) error {
	return c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/funnel",
		SetSessionFunnelRequest{Enabled: enabled, ExpiresIn: expiresIn}, nil)
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

// -------------------------------------------------------------------------
// Phase 118 / Plan 05: file-browser DaemonClient methods.
//
// All four methods use the existing c.http transport (Unix socket / Windows
// named pipe). url.Values.Encode() handles path quoting so special characters
// (spaces, '#', '?') do not break the URL. Non-2xx responses surface as
// errors containing the status code so callers can branch on 403/404/413
// without re-issuing the request.
// -------------------------------------------------------------------------

// filesURL builds /api/files/<op>?session=<sid>&path=<rel>. relPath empty
// is normalised to "." so callers can pass "" for "list the root".
func (c *DaemonClient) filesURL(op, sessionID, relPath string) string {
	if relPath == "" {
		relPath = "."
	}
	q := url.Values{}
	q.Set("session", sessionID)
	q.Set("path", relPath)
	return c.base + "/api/files/" + op + "?" + q.Encode()
}

// ListFiles fetches a directory listing for the given session via the
// daemon-local socket. The loopback transport is the trust boundary — no
// cap-token is sent. Returns entries, truncated flag, and any non-2xx
// status surfaced as a typed error.
// Phase 118 / FS-03.
func (c *DaemonClient) ListFiles(ctx context.Context, sessionID, relPath string) ([]files.FileEntry, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.filesURL("list", sessionID, relPath), nil)
	if err != nil {
		return nil, false, fmt.Errorf("files list: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("files list: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("files list: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, false, fmt.Errorf("files list: decode response: %w", err)
	}
	return out.Entries, out.Truncated, nil
}

// StatFile fetches metadata for a single path inside the session's sandbox.
// Phase 118 / FS-04.
func (c *DaemonClient) StatFile(ctx context.Context, sessionID, relPath string) (files.FileEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.filesURL("stat", sessionID, relPath), nil)
	if err != nil {
		return files.FileEntry{}, fmt.Errorf("files stat: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileEntry{}, fmt.Errorf("files stat: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return files.FileEntry{}, fmt.Errorf("files stat: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileEntry
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileEntry{}, fmt.Errorf("files stat: decode response: %w", err)
	}
	return out, nil
}

// ReadFile fetches file bytes via the daemon socket. The daemon enforces the
// 5 MiB preview cap server-side (Pitfall 5 / FS-05); a 413 surfaces here as
// a typed error so callers can distinguish "too large" from "not found"
// without re-issuing the request.
// Phase 118 / FS-05.
func (c *DaemonClient) ReadFile(ctx context.Context, sessionID, relPath string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.filesURL("read", sessionID, relPath), nil)
	if err != nil {
		return nil, "", fmt.Errorf("files read: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("files read: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("files read: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("files read: read body: %w", err)
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// HeadFile preflights size + Content-Type + modtime without transferring the
// body. Phase 120 (GUI) uses this for cap-aware previews — query the size
// before deciding to fetch. Phase 118 / FS-06.
func (c *DaemonClient) HeadFile(ctx context.Context, sessionID, relPath string) (int64, string, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.filesURL("read", sessionID, relPath), nil)
	if err != nil {
		return 0, "", time.Time{}, fmt.Errorf("files head: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", time.Time{}, fmt.Errorf("files head: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", time.Time{}, fmt.Errorf("files head: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	// Content-Length parsing — http.Response.ContentLength is already int64;
	// -1 means absent. HEAD responses on a known-size body always populate it.
	size := resp.ContentLength
	if size < 0 {
		size = 0
	}
	// Last-Modified header — parse with http.ParseTime so RFC1123/RFC850 are
	// both handled. On parse failure the zero time is returned so callers see
	// "no usable mtime" rather than a stale value.
	var mtime time.Time
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if t, err := http.ParseTime(lm); err == nil {
			mtime = t
		}
	}
	return size, resp.Header.Get("Content-Type"), mtime, nil
}

// -------------------------------------------------------------------------
// Phase 123 / Plan 04: DaemonClient write methods (FSW-09).
//
// All five methods mirror the read-method patterns established in Plan 118/05:
//   - filesURL helper for URL construction (op, sessionID, relPath)
//   - http.NewRequestWithContext for context-aware requests
//   - non-2xx status surfaced as a typed error: "files <op>: %d %s"
//   - doJSON for JSON-bodied ops (RenameFile, MkdirFile)
//
// Auth-less by design — the daemon Unix socket is the trust boundary (WEB-01).
// DaemonClient implements the files write surface for all surviving surfaces
// (GUI, CLI, web); the TUI surface was removed in Phase 136.
// -------------------------------------------------------------------------

// WriteFile writes data to relPath inside the session sandbox via PUT
// /api/files/write. Uses Content-Type application/octet-stream with the
// raw bytes as the body. Non-2xx responses surface as a typed error.
// Phase 123 / FSW-09 / T-123-18 (ctx cancellation closes the hang risk).
func (c *DaemonClient) WriteFile(ctx context.Context, sessionID, relPath string, data []byte) (files.FileWriteResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.filesURL("write", sessionID, relPath), bytes.NewReader(data))
	if err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files write: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files write: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return files.FileWriteResponse{}, fmt.Errorf("files write: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileWriteResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files write: decode response: %w", err)
	}
	return out, nil
}

// UploadFile sends data as a multipart/form-data POST to /api/files/upload.
// dir is the sandbox-relative target directory ("." for root). filename is
// the base name of the file part; the server applies filepath.Base for an
// additional traversal defence. Phase 123 / FSW-09.
func (c *DaemonClient) UploadFile(ctx context.Context, sessionID, dir, filename string, data []byte) (files.FileWriteResponse, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("dir", dir); err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files upload: write dir field: %w", err)
	}
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files upload: create form file: %w", err)
	}
	if _, err := fw.Write(data); err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files upload: write file data: %w", err)
	}
	if err := mw.Close(); err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files upload: close multipart writer: %w", err)
	}

	// filesURL adds session+path query params; path is unused for upload (dir
	// comes from the form field) but we pass "." to satisfy the helper signature.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.filesURL("upload", sessionID, "."), &buf)
	if err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files upload: new request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files upload: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return files.FileWriteResponse{}, fmt.Errorf("files upload: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileWriteResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("files upload: decode response: %w", err)
	}
	return out, nil
}

// DeleteFile removes relPath inside the session sandbox via DELETE
// /api/files/delete. Returns FileOpResponse{OK: true} on success.
// Phase 123 / FSW-09.
func (c *DaemonClient) DeleteFile(ctx context.Context, sessionID, relPath string) (files.FileOpResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.filesURL("delete", sessionID, relPath), nil)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files delete: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files delete: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return files.FileOpResponse{}, fmt.Errorf("files delete: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileOpResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files delete: decode response: %w", err)
	}
	return out, nil
}

// renameRequest is the JSON body sent to POST /api/files/rename.
type renameRequest struct {
	OldRel string `json:"oldRel"`
	NewRel string `json:"newRel"`
}

// RenameFile moves oldRel to newRel inside the session sandbox via POST
// /api/files/rename with a JSON body. Both paths are validated server-side
// (T-123-01 destination traversal risk). Phase 123 / FSW-09.
func (c *DaemonClient) RenameFile(ctx context.Context, sessionID, oldRel, newRel string) (files.FileOpResponse, error) {
	body := renameRequest{OldRel: oldRel, NewRel: newRel}
	b, err := json.Marshal(body)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files rename: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.filesURL("rename", sessionID, "."), bytes.NewReader(b))
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files rename: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files rename: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body2, _ := io.ReadAll(resp.Body)
		return files.FileOpResponse{}, fmt.Errorf("files rename: %d %s", resp.StatusCode, strings.TrimSpace(string(body2)))
	}
	var out files.FileOpResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files rename: decode response: %w", err)
	}
	return out, nil
}

// MkdirFile creates relPath (and all missing parent directories) inside the
// session sandbox via POST /api/files/mkdir. Uses the path query parameter —
// relPath is passed via filesURL. Phase 123 / FSW-09.
func (c *DaemonClient) MkdirFile(ctx context.Context, sessionID, relPath string) (files.FileOpResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.filesURL("mkdir", sessionID, relPath), nil)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files mkdir: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files mkdir: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return files.FileOpResponse{}, fmt.Errorf("files mkdir: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileOpResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileOpResponse{}, fmt.Errorf("files mkdir: decode response: %w", err)
	}
	return out, nil
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
