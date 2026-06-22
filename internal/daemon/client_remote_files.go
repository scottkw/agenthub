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
// from the 303 See Other Location header.
//
// Phase 123-02 TD-5 (FSW-10): the webserver returns HTTP 303 See Other with a
// Location header of the form:
//
//	/sessions/<id>?cap=<token>       — success
//	/join?error=<kind>               — error (kind is one of the substrings below)
//
// A dedicated http.Client with CheckRedirect returning http.ErrUseLastResponse
// is constructed inside this function to prevent Go from auto-following the
// 303 into /sessions/<id> (which would lose the cap). The shared
// remoteFilesHTTPClient is NOT used on the success path — it remains unmutated
// so RegisterRemoteCap and future callers are unaffected.
//
// Error strings (substrings checked by the modal UI):
//   - "expired"      → user must request a fresh code from the owner
//   - "invalid"      → code typo / wrong code / missing cap in Location
//   - "not-found"    → upstream session no longer exists
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

	// Dedicated client: CheckRedirect returns ErrUseLastResponse so we
	// observe the 303 directly and can read the Location header. The shared
	// remoteFilesHTTPClient deliberately has no CheckRedirect set; do NOT
	// add it there (RESEARCH Pitfall 5, T-123-09).
	dedicated := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // tailnet peer with self-signed cert
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	form := url.Values{}
	form.Set("code", trimmed)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build /join/exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := dedicated.Do(req)
	if err != nil {
		return "", fmt.Errorf("contact remote /join/exchange: %w", err)
	}
	defer resp.Body.Close()

	// Expect 303 See Other — the webserver always redirects on exchange.
	// Any other status is an error path (fall through to the 4xx mapping below).
	if resp.StatusCode != http.StatusSeeOther {
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
		// Defensive: any other 4xx/5xx → "invalid" so the modal shows the
		// user-friendly fallback rather than a raw status code.
		return "", fmt.Errorf("join code is invalid (status %d)", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")

	// WR-05: parse Location once and read the error query param via the
	// url package rather than string surgery. This handles absolute error
	// URLs (e.g. https://host/join?error=expired) correctly and avoids
	// the TrimPrefix-on-Contains mismatch where an absolute URL like
	// "https://host/join?error=expired" would not strip the prefix and
	// would return the full URL as the kind.
	locURL, locParseErr := url.Parse(loc)

	// Error-shape Location: path matches /join?error=<kind>
	// Check via parsed URL path so absolute and relative forms both work.
	if locParseErr == nil && locURL.Path == "/join" && locURL.Query().Get("error") != "" {
		kind := locURL.Query().Get("error")
		return "", fmt.Errorf("join exchange: %s", kind)
	}

	// WR-04: if the Location URL is absolute (has a host), assert its host
	// and scheme match the request target (parsed.Host / parsed.Scheme) before
	// accepting the cap token. With InsecureSkipVerify on the transport, a
	// tailnet MITM could return a Location pointing at an attacker-controlled
	// host carrying a forged cap token.
	if locParseErr == nil && locURL.Host != "" {
		if locURL.Host != parsed.Host || locURL.Scheme != parsed.Scheme {
			return "", fmt.Errorf("join exchange: no cap in location (invalid)")
		}
	}

	// Success-shape Location: /sessions/<id>?cap=<token>.
	// Re-use the already-parsed locURL if parse succeeded; otherwise re-parse
	// (should not happen given the checks above, but be defensive).
	if locParseErr != nil {
		return "", fmt.Errorf("join exchange: bad location header")
	}
	capTok := locURL.Query().Get("cap")
	if capTok == "" {
		return "", fmt.Errorf("join exchange: no cap in location (invalid)")
	}
	return capTok, nil
}

// RemoteSessionOpenURL retrieves the cap-bearing open URL for a remote session
// from the local daemon's RemoteCapStore. The daemon composes
// baseURL+/sessions/{id}?cap=TOKEN from the stored entry and returns it as a
// plain string. The cap token never leaves the daemon except inside the
// returned URL (which already carries it by design in the existing exchange→open
// path — T-146-05-01 accept).
//
// Returns an error containing "status 404" when no cap is stored for sessionID;
// the caller treats this as "no cap; fall back to the join-code modal".
// Phase 146-05 / GAP-146-A.
func (c *DaemonClient) RemoteSessionOpenURL(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session id is invalid: empty")
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := c.doJSON(http.MethodGet, "/api/remote-files/caps/"+url.PathEscape(sessionID)+"/open-url", nil, &result); err != nil {
		return "", err
	}
	return result.URL, nil
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
