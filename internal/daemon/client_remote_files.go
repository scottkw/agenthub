package daemon

// Phase 122-03 — Remote-file-browse client helpers.
//
// These helpers underpin the Wails-exposed `ExchangeJoinCodeAtURL` and
// `RegisterRemoteCap` methods on *main.App. They are co-located here (separate
// file from client.go) so Plan 122-01 — which owns the actual daemon-side
// implementation of `/api/remote-files/caps` deposit and the upstream
// `/join/exchange` proxy — can replace these bodies without touching
// surrounding code.
//
// CURRENT BEHAVIOUR (122-03 standalone):
//   - ExchangeJoinCodeAtURL: contacts the remote webserver's /join/exchange
//     endpoint directly from this process (since the daemon-side proxy from
//     Plan 122-01 may not have landed yet on this worktree's base). The shape
//     is documented in 122-RESEARCH.md §Join Code Exchange.
//   - RegisterRemoteCap: POSTs to the local daemon's /api/remote-files/caps
//     endpoint per 122-CONTEXT.md. When Plan 122-01 lands it owns this route;
//     until then the helper surfaces a clear "endpoint not implemented" error
//     so the modal failure-path tests still get a typed error.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// remoteFilesHTTPClient is a shared http.Client used by the helpers below.
// Configured with a sane timeout and a permissive TLS verifier because the
// tailnet peers use self-signed certs that match their MagicDNS name (the
// existing tailnet HTTP fetch code in internal/tailnet uses the same shape).
var remoteFilesHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // tailnet peer with self-signed cert
	},
}

// ExchangeJoinCodeAtURL POSTs the 5-character join code to the remote
// webserver's `/join/exchange` endpoint and returns the cap token extracted
// from the response.
//
// Error strings (substrings checked by the modal UI):
//   - "expired"     → user must request a fresh code from the owner
//   - "invalid"     → code typo / wrong code
//   - "not-found"   → upstream session no longer exists
//   - "session-gone" → web-share toggled off after code was issued
//
// Plan 122-01 may move this to a proxied-by-daemon variant; the public
// signature here is the contract App.ExchangeJoinCodeAtURL depends on.
func (c *DaemonClient) ExchangeJoinCodeAtURL(remoteBaseURL, code string) (string, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return "", fmt.Errorf("join code is invalid: empty")
	}

	parsed, err := url.Parse(strings.TrimRight(remoteBaseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("remote base URL is invalid: %v", err)
	}

	endpoint := parsed.String() + "/join/exchange"

	form := url.Values{}
	form.Set("code", trimmed)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build /join/exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := remoteFilesHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("contact remote /join/exchange: %w", err)
	}
	defer resp.Body.Close()

	// Status-code → error-substring mapping that the modal pivots on.
	if resp.StatusCode == http.StatusGone {
		return "", fmt.Errorf("join code is session-gone (status 410)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("join code not-found (status 404)")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("join code is invalid (status %d)", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusRequestTimeout {
		return "", fmt.Errorf("join code expired (status 408)")
	}
	if resp.StatusCode >= 400 {
		// Defensive: any other 4xx/5xx surfaces as "invalid" so the modal
		// shows the user-friendly fallback rather than a raw status code.
		return "", fmt.Errorf("join code is invalid (status %d)", resp.StatusCode)
	}

	// Successful path: response body is JSON `{"cap":"..."}`.
	var body struct {
		Cap string `json:"cap"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decode /join/exchange response: %w", err)
	}
	if body.Cap == "" {
		return "", fmt.Errorf("join code exchange returned empty cap (invalid)")
	}
	return body.Cap, nil
}

// RegisterRemoteCap deposits a (sessionID, baseURL, capToken) tuple into the
// local daemon's RemoteCapStore via POST /api/remote-files/caps. Plan 122-01
// owns the daemon-side handler; this helper provides the client side so the
// Wails binding has a stable target.
func (c *DaemonClient) RegisterRemoteCap(sessionID, baseURL, capToken string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is invalid: empty")
	}
	if baseURL == "" {
		return fmt.Errorf("base URL is invalid: empty")
	}
	if capToken == "" {
		return fmt.Errorf("cap token is invalid: empty")
	}
	body := map[string]string{
		"sessionId": sessionID,
		"baseUrl":   baseURL,
		"capToken":  capToken,
	}
	// doJSON handles JSON encoding, the unix-socket dial, and 4xx→error
	// mapping. When Plan 122-01 lands the daemon handler this call succeeds;
	// before then the daemon returns 404 and doJSON surfaces that as an error
	// containing "status 404" — the modal handler treats this as a hard
	// failure (covered by Task 2 Test 7 — generic error path).
	return c.doJSON(http.MethodPost, "/api/remote-files/caps", body, nil)
}
