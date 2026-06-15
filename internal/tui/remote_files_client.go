package tui

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/scottkw/agenthub/internal/files"
)

// remotePeerOutdatedMessage is the verbatim SC3 user-facing copy surfaced when
// a write verb returns HTTP 405 from a v3.4 peer that has no write routes.
// MUST byte-match the TS const REMOTE_PEER_OUTDATED_MESSAGE in
// frontend/src/lib/filesApi.ts (RMW-04 cross-surface parity contract).
const remotePeerOutdatedMessage = "The remote session is running an older version of AgentHub that does not support file writes."

// ErrRemotePeerNoWriteSupport is returned by all 4 write methods when the
// remote peer responds with HTTP 405 (upstream v3.4 peer has no write routes).
// Callers use errors.Is to detect this sentinel and render the verbatim copy.
var ErrRemotePeerNoWriteSupport = errors.New(remotePeerOutdatedMessage)

// remoteCapExpiredMessage is the user-facing copy surfaced when a write verb
// returns HTTP 401 (cap token expired or revoked mid-session).
// CAP-LEAK invariant: this string must NOT interpolate the cap token or URL.
const remoteCapExpiredMessage = "your access to this remote session has expired"

// ErrRemoteCapExpired is returned by all 4 write methods when the remote peer
// responds with HTTP 401 (capability token expired or revoked). Callers use
// errors.Is to detect this sentinel and render a distinct "access expired"
// message instead of the generic write-error copy. (RMW-05)
var ErrRemoteCapExpired = errors.New(remoteCapExpiredMessage)

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

// WriteFile writes data to relPath inside the remote session sandbox via PUT
// /api/files/write with Content-Type application/octet-stream. Returns a
// FileWriteResponse on 200. Mirrors DaemonClient.WriteFile (client.go:513-533).
//
// CAP-LEAK invariant (T-126-01): error strings interpolate only (statusCode,
// body) — never the full URL (which carries cap=).
func (c *RemoteFilesClient) WriteFile(ctx context.Context, sid, rel string, data []byte) (files.FileWriteResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.filesURL("write", sid, rel), bytes.NewReader(data))
	if err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("remote files write: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("remote files write: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusMethodNotAllowed { // RMW-04: v3.4 peer has no write routes
		return files.FileWriteResponse{}, ErrRemotePeerNoWriteSupport
	}
	if resp.StatusCode == http.StatusUnauthorized { // RMW-05: cap expired/revoked mid-session
		return files.FileWriteResponse{}, ErrRemoteCapExpired
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return files.FileWriteResponse{}, fmt.Errorf("remote files write: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileWriteResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileWriteResponse{}, fmt.Errorf("remote files write: decode response: %w", err)
	}
	return out, nil
}

// DeleteFile removes relPath inside the remote session sandbox via DELETE
// /api/files/delete. Returns FileOpResponse{OK: true} on success. Mirrors
// DaemonClient.DeleteFile (client.go:582-601).
//
// CAP-LEAK invariant (T-126-01): error strings interpolate only (statusCode,
// body) — never the full URL.
func (c *RemoteFilesClient) DeleteFile(ctx context.Context, sid, rel string) (files.FileOpResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.filesURL("delete", sid, rel), nil)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files delete: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files delete: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusMethodNotAllowed { // RMW-04: v3.4 peer has no write routes
		return files.FileOpResponse{}, ErrRemotePeerNoWriteSupport
	}
	if resp.StatusCode == http.StatusUnauthorized { // RMW-05: cap expired/revoked mid-session
		return files.FileOpResponse{}, ErrRemoteCapExpired
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return files.FileOpResponse{}, fmt.Errorf("remote files delete: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileOpResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files delete: decode response: %w", err)
	}
	return out, nil
}

// remoteRenameRequest is the JSON body sent to POST /api/files/rename.
// Mirrors daemon.renameRequest (client.go:604-607).
type remoteRenameRequest struct {
	OldRel string `json:"oldRel"`
	NewRel string `json:"newRel"`
}

// RenameFile moves oldRel to newRel inside the remote session sandbox via POST
// /api/files/rename with a JSON body. Both paths are validated server-side.
// Mirrors DaemonClient.RenameFile (client.go:612-637).
//
// CAP-LEAK invariant (T-126-01): error strings interpolate only (statusCode,
// body) — never the full URL.
func (c *RemoteFilesClient) RenameFile(ctx context.Context, sid, oldRel, newRel string) (files.FileOpResponse, error) {
	body := remoteRenameRequest{OldRel: oldRel, NewRel: newRel}
	b, err := json.Marshal(body)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files rename: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.filesURL("rename", sid, "."), bytes.NewReader(b))
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files rename: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files rename: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusMethodNotAllowed { // RMW-04: v3.4 peer has no write routes
		return files.FileOpResponse{}, ErrRemotePeerNoWriteSupport
	}
	if resp.StatusCode == http.StatusUnauthorized { // RMW-05: cap expired/revoked mid-session
		return files.FileOpResponse{}, ErrRemoteCapExpired
	}
	if resp.StatusCode != http.StatusOK {
		body2, _ := io.ReadAll(resp.Body)
		return files.FileOpResponse{}, fmt.Errorf("remote files rename: %d %s", resp.StatusCode, strings.TrimSpace(string(body2)))
	}
	var out files.FileOpResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files rename: decode response: %w", err)
	}
	return out, nil
}

// MkdirFile creates relPath (and all missing parent directories) inside the
// remote session sandbox via POST /api/files/mkdir. The target path is passed
// via filesURL's path query parameter. Mirrors DaemonClient.MkdirFile
// (client.go:642-661).
//
// CAP-LEAK invariant (T-126-01): error strings interpolate only (statusCode,
// body) — never the full URL.
func (c *RemoteFilesClient) MkdirFile(ctx context.Context, sid, rel string) (files.FileOpResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.filesURL("mkdir", sid, rel), nil)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files mkdir: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files mkdir: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusMethodNotAllowed { // RMW-04: v3.4 peer has no write routes
		return files.FileOpResponse{}, ErrRemotePeerNoWriteSupport
	}
	if resp.StatusCode == http.StatusUnauthorized { // RMW-05: cap expired/revoked mid-session
		return files.FileOpResponse{}, ErrRemoteCapExpired
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return files.FileOpResponse{}, fmt.Errorf("remote files mkdir: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out files.FileOpResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return files.FileOpResponse{}, fmt.Errorf("remote files mkdir: decode response: %w", err)
	}
	return out, nil
}
