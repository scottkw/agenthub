package tui

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/scottkw/agenthub/internal/files"
)

// RemoteFilesClient satisfies FilesClient by talking to a remote AgentHub
// peer's webserver over Tailscale HTTPS with a session-scoped capability
// token in the query string. It mirrors *daemon.DaemonClient's files methods
// (internal/daemon/client.go:362-484) but is pointed at
// https://{peer-fqdn}:{port} instead of the local Unix socket.
//
// Phase 122 / REMOTE-03. The TUI talks DIRECTLY to the remote webserver
// (no daemon proxy) because a Go HTTP client over Tailscale TLS has no CORS
// concerns — the daemon proxy only exists to satisfy the browser-origin
// constraint that the desktop Wails surface lives under.
//
// CAP-LEAK INVARIANT (T-122-04-01): every error returned by this type MUST
// NOT contain the cap token. We accomplish this by interpolating only
// (statusCode, body) into error strings, never the full request URL.
// redactCapFromURL is provided for the rare path where URL interpolation
// is needed; callers should pass any URL through it first.
type RemoteFilesClient struct {
	baseURL  string
	capToken string
	http     *http.Client
}

// Compile-time guard: *RemoteFilesClient satisfies FilesClient.
var _ FilesClient = (*RemoteFilesClient)(nil)

// NewRemoteFilesClient constructs a client targeting baseURL with the given
// cap token. baseURL is of shape "https://{fqdn}:{port}" (no trailing path).
// A trailing slash is stripped so the caller does not have to be careful.
//
// TLS 1.2+ is enforced via tls.Config.MinVersion. Timeout is 15 seconds —
// slightly longer than DaemonClient's 5s because Tailscale RTT is higher
// than the local socket. Per-request context timeouts in the Cmd factories
// provide defense-in-depth.
func NewRemoteFilesClient(baseURL, capToken string) *RemoteFilesClient {
	return &RemoteFilesClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		capToken: capToken,
		http: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
		},
	}
}

// newRemoteFilesClientWithHTTP is a test-only constructor that lets a unit
// test inject a pre-configured *http.Client (typically an httptest.NewTLSServer
// client) so the test can exercise the remote-file methods without performing
// a real TLS handshake. Additive surface — production code never uses this.
func newRemoteFilesClientWithHTTP(baseURL, capToken string, httpClient *http.Client) *RemoteFilesClient {
	return &RemoteFilesClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		capToken: capToken,
		http:     httpClient,
	}
}

// NewRemoteFilesClientForTest is the EXPORTED test-only constructor mirroring
// newRemoteFilesClientWithHTTP for callers outside this package. The Phase
// 122-05 cross-surface parity test lives in `package daemon_test` (because
// `package daemon` cannot import `internal/tui` — that would cycle, since
// tui already imports daemon), and needs to build a RemoteFilesClient
// pointed at an httptest.NewTLSServer with the upstream's self-signed cert.
// Production code must never call this — use NewRemoteFilesClient instead.
func NewRemoteFilesClientForTest(baseURL, capToken string, httpClient *http.Client) *RemoteFilesClient {
	return newRemoteFilesClientWithHTTP(baseURL, capToken, httpClient)
}

// filesURL builds /api/files/<op>?session=<sid>&path=<rel>&cap=<token>. Empty
// relPath is normalised to "." so callers can pass "" for "list the root"
// (mirrors *daemon.DaemonClient.filesURL line 367-368).
func (c *RemoteFilesClient) filesURL(op, sessionID, relPath string) string {
	if relPath == "" {
		relPath = "."
	}
	q := url.Values{}
	q.Set("session", sessionID)
	q.Set("path", relPath)
	if c.capToken != "" {
		q.Set("cap", c.capToken)
	}
	return c.baseURL + "/api/files/" + op + "?" + q.Encode()
}

// redactCapFromURL strips the cap= query parameter from a URL string so the
// remaining shape can be safely embedded in error messages. Defense-in-depth
// for any future error path that wants to interpolate the URL — current
// implementations interpolate only (status, body) and do not need this.
func redactCapFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	if q.Has("cap") {
		q.Set("cap", "[redacted]")
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// ListFiles fetches a directory listing over HTTPS+cap.
func (c *RemoteFilesClient) ListFiles(ctx context.Context, sid, rel string) ([]files.FileEntry, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.filesURL("list", sid, rel), nil)
	if err != nil {
		return nil, false, fmt.Errorf("remote files list: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("remote files list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("remote files list: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, false, fmt.Errorf("remote files list: decode response: %w", err)
	}
	return out.Entries, out.Truncated, nil
}

// StatFile fetches metadata for a single path inside the session sandbox.
func (c *RemoteFilesClient) StatFile(ctx context.Context, sid, rel string) (files.FileEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.filesURL("stat", sid, rel), nil)
	if err != nil {
		return files.FileEntry{}, fmt.Errorf("remote files stat: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileEntry{}, fmt.Errorf("remote files stat: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return files.FileEntry{}, fmt.Errorf("remote files stat: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileEntry
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileEntry{}, fmt.Errorf("remote files stat: decode response: %w", err)
	}
	return out, nil
}

// ReadFile fetches file bytes from the remote sandbox.
func (c *RemoteFilesClient) ReadFile(ctx context.Context, sid, rel string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.filesURL("read", sid, rel), nil)
	if err != nil {
		return nil, "", fmt.Errorf("remote files read: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("remote files read: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("remote files read: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("remote files read: read body: %w", err)
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// HeadFile preflights size + Content-Type + modtime without transferring the
// body. Mirrors DaemonClient.HeadFile lines 454-484.
func (c *RemoteFilesClient) HeadFile(ctx context.Context, sid, rel string) (int64, string, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.filesURL("read", sid, rel), nil)
	if err != nil {
		return 0, "", time.Time{}, fmt.Errorf("remote files head: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", time.Time{}, fmt.Errorf("remote files head: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", time.Time{}, fmt.Errorf("remote files head: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	size := resp.ContentLength
	if size < 0 {
		size = 0
	}
	var mtime time.Time
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if t, err := http.ParseTime(lm); err == nil {
			mtime = t
		}
	}
	return size, resp.Header.Get("Content-Type"), mtime, nil
}
