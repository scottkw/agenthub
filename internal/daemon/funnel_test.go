// Package daemon — Tailscale Funnel lifecycle tests (Phase 165, FNL-01/FNL-03/FNL-05/FNL-07).
//
// Drive state through the real HTTP handlers and ws.EnableFunnel / ws.DisableFunnel.
// Teardown is asserted via the fake funnelClient's stored serve config, never by
// reading a.funnelSessions directly (Pitfall 5/13 guard from 165-RESEARCH.md).
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/webserver"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

// daemonFakeFunnelClient is a test double for webserver.FunnelClientForTest.
// It maintains a stateful stored config so GetServeConfig returns what
// SetServeConfig last wrote — honouring the ETag read-modify-write invariant
// that EnableFunnel/DisableFunnel depend on (Pitfall 3 / T-165-04).
// All method calls are thread-safe.
type daemonFakeFunnelClient struct {
	mu           sync.Mutex
	storedConfig *ipn.ServeConfig
	hostname     string
}

// GetServeConfig returns the last config stored by SetServeConfig (nil = not configured).
func (f *daemonFakeFunnelClient) GetServeConfig(_ context.Context) (*ipn.ServeConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.storedConfig, nil
}

// SetServeConfig stores the config for subsequent GetServeConfig calls.
func (f *daemonFakeFunnelClient) SetServeConfig(_ context.Context, cfg *ipn.ServeConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.storedConfig = cfg
	return nil
}

// StatusWithoutPeers returns a node status with the full Funnel capability set
// required for ipn.CheckFunnelAccess(443, Self) to succeed.
func (f *daemonFakeFunnelClient) StatusWithoutPeers(_ context.Context) (*ipnstate.Status, error) {
	return &ipnstate.Status{
		Self: &ipnstate.PeerStatus{
			DNSName: f.hostname + ".",
			CapMap: tailcfg.NodeCapMap{
				tailcfg.CapabilityHTTPS: nil,
				tailcfg.NodeAttrFunnel:  nil,
				tailcfg.NodeCapability("https://tailscale.com/cap/funnel-ports?ports=443,8443,10000"): nil,
			},
		},
	}, nil
}

// IsFunnelOn reports whether the stored serve config has IsFunnelOn() == true.
// Use this to assert Funnel teardown without reading a.funnelSessions directly
// (Pitfall 5 / Phase 150 wrong-assumption lesson from MEMORY.md).
func (f *daemonFakeFunnelClient) IsFunnelOn() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.storedConfig == nil {
		return false
	}
	return f.storedConfig.IsFunnelOn()
}

// Compile-time assertion: daemonFakeFunnelClient implements the exported cross-package seam type.
var _ webserver.FunnelClientForTest = (*daemonFakeFunnelClient)(nil)

// makeFunnelTestWebServer creates a started WebServer with an injected fake funnelClient.
// The fake maintains stateful storedConfig so the real EnableFunnel/DisableFunnel bodies
// run correctly (Pitfall 13 — not a shallow mock that short-circuits the real code path).
// The WebServer is started (so ws.listener != nil, required by EnableFunnel Step 5).
func makeFunnelTestWebServer(t *testing.T, api *API, hostname string) (*webserver.WebServer, *daemonFakeFunnelClient) {
	t.Helper()
	lnCfg := newLoopbackTLSListener(t)
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: lnCfg,
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	fake := &daemonFakeFunnelClient{hostname: hostname}
	// Inject before Start so EnableFunnel uses the fake (not the production lc).
	ws.SetFunnelClientForTest(fake)
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })
	api.SetWebServerForTest(ws)
	return ws, fake
}

// enableFunnelViaHTTP calls POST /sessions/{id}/funnel {enabled:true} through the
// daemon Unix socket and returns the decoded SetSessionFunnelResponse.
func enableFunnelViaHTTP(t *testing.T, socketPath, sessionID string) SetSessionFunnelResponse {
	t.Helper()
	status, body := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", sessionID), `{"enabled":true}`)
	if status != http.StatusOK {
		t.Fatalf("POST /sessions/%s/funnel {enabled:true}: want 200, got %d; body: %s", sessionID, status, body)
	}
	var resp SetSessionFunnelResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode SetSessionFunnelResponse: %v; body: %s", err, body)
	}
	return resp
}

// ---------------------------------------------------------------------------
// Task 1: funnelSessions map + handleSetSessionFunnel + auto-expiry + FunnelActive
// ---------------------------------------------------------------------------

// TestFunnelSessionsMap verifies FNL-01 (Funnel off by default):
//   - A freshly-created session has FunnelActive=false in GET /sessions.
//   - funnelSessions is empty (no entries for any session).
func TestFunnelSessionsMap(t *testing.T) {
	api, _, socketPath := testDaemon(t)
	// Inject an un-started WebServer (FunnelActive check only needs the ws pointer,
	// not a running listener).
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP: "127.0.0.1",
		Port:   0,
		FQDN:   "test.local",
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	api.SetWebServerForTest(ws)

	// Create a session.
	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"funnel-off-default","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	// GET /sessions must report FunnelActive=false (default off, FNL-01).
	_, listBody := rawGet(t, socketPath, "/sessions")
	var sessions []SessionInfo
	if err := json.Unmarshal(listBody, &sessions); err != nil {
		t.Fatalf("decode sessions: %v; body: %s", err, listBody)
	}
	for _, s := range sessions {
		if s.ID == cr.ID && s.FunnelActive {
			t.Errorf("FNL-01: session %s FunnelActive=true before any funnel enable, want false", cr.ID)
		}
	}
}

// TestHandleSetSessionFunnel_Enable verifies the happy-path enable (FNL-01/FNL-04):
//   - POST /sessions/{id}/funnel {enabled:true} drives ws.EnableFunnel via the injected fake.
//   - Response FunnelURL is non-empty and has no ":443"/":7443" port suffix.
//   - GET /sessions reports FunnelActive=true for that session.
func TestHandleSetSessionFunnel_Enable(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	_, fake := makeFunnelTestWebServer(t, api, "test.ts.net")

	// Create a session.
	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"funnel-enable","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	// Enable Funnel.
	resp := enableFunnelViaHTTP(t, socketPath, cr.ID)

	// FunnelURL must be non-empty and must NOT contain a port component.
	if resp.FunnelURL == "" {
		t.Fatal("FunnelURL must be non-empty after enable")
	}
	if strings.Contains(resp.FunnelURL, ":443") || strings.Contains(resp.FunnelURL, ":7443") {
		t.Errorf("FunnelURL must not contain port suffix for port 443: got %q", resp.FunnelURL)
	}

	// The fake config must show IsFunnelOn — the real EnableFunnel body ran.
	if !fake.IsFunnelOn() {
		t.Error("expected fake funnelClient to show IsFunnelOn=true after enable")
	}

	// GET /sessions must report FunnelActive=true.
	_, listBody := rawGet(t, socketPath, "/sessions")
	var sessions []SessionInfo
	if err := json.Unmarshal(listBody, &sessions); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.ID == cr.ID {
			found = true
			if !s.FunnelActive {
				t.Errorf("session %s FunnelActive=false after enable, want true", cr.ID)
			}
		}
	}
	if !found {
		t.Errorf("session %s not found in GET /sessions", cr.ID)
	}
}

// TestHandleSetSessionFunnel_DisableTeardown verifies that toggle-off (site 1):
//   - Removes the session from funnelSessions (no more sessions → len==0).
//   - Drives the real ws.DisableFunnel via disableFunnelForSession so the fake
//     serve config ends with IsFunnelOn=false.
func TestHandleSetSessionFunnel_DisableTeardown(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	_, fake := makeFunnelTestWebServer(t, api, "test.ts.net")

	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"funnel-disable","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	enableFunnelViaHTTP(t, socketPath, cr.ID)
	if !fake.IsFunnelOn() {
		t.Fatal("precondition: fake config must show IsFunnelOn after enable")
	}

	// Toggle off.
	status, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", cr.ID), `{"enabled":false}`)
	if status != http.StatusNoContent {
		t.Fatalf("disable: want 204, got %d", status)
	}

	// The real ws.DisableFunnel must have been called — fake config must show IsFunnelOn=false.
	if fake.IsFunnelOn() {
		t.Error("expected fake funnelClient to show IsFunnelOn=false after disable toggle-off")
	}

	// GET /sessions must report FunnelActive=false.
	_, listBody := rawGet(t, socketPath, "/sessions")
	var sessions []SessionInfo
	if err := json.Unmarshal(listBody, &sessions); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	for _, s := range sessions {
		if s.ID == cr.ID && s.FunnelActive {
			t.Errorf("session %s FunnelActive=true after disable, want false", cr.ID)
		}
	}
}

// TestFunnelAutoExpiry verifies FNL-07 auto-expiry:
//   - POST {enabled:true, expiresIn:1} registers a real time.AfterFunc timer.
//   - After the timer fires (~1s) the fake serve config shows IsFunnelOn=false
//     and funnelSessions is empty — with NO further HTTP/UI call.
//   - A re-enable before the prior expiry Stops that timer (no double-fire):
//     enabling with a long expiresIn after a short one keeps the config active
//     past the short timer's window.
func TestFunnelAutoExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener and timer wait")
	}
	api, _, socketPath := testDaemon(t)
	_, fake := makeFunnelTestWebServer(t, api, "test.ts.net")

	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"funnel-expiry","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	// Enable with 1-second auto-expiry.
	status, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", cr.ID), `{"enabled":true,"expiresIn":1}`)
	if status != http.StatusOK {
		t.Fatalf("enable with expiry: want 200, got %d", status)
	}
	if !fake.IsFunnelOn() {
		t.Fatal("precondition: fake config must show IsFunnelOn after enable")
	}

	// Poll for up to 3 seconds; the timer fires after ~1 second.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !fake.IsFunnelOn() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if fake.IsFunnelOn() {
		t.Error("FNL-07: expected Funnel to be auto-disabled after 1-second expiry timer fired")
	}

	// Re-enable guard: a short-expiry enable followed immediately by a long-expiry
	// re-enable must cancel the short timer. The config stays active past the 1s window.
	status, _ = rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", cr.ID), `{"enabled":true,"expiresIn":1}`)
	if status != http.StatusOK {
		t.Fatalf("re-enable short: want 200, got %d", status)
	}
	// Immediately override with a long expiry (should Stop the 1-second timer).
	status, _ = rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", cr.ID), `{"enabled":true,"expiresIn":60}`)
	if status != http.StatusOK {
		t.Fatalf("re-enable long: want 200, got %d", status)
	}

	// Wait 2 seconds — the short 1s timer would have fired if not cancelled.
	time.Sleep(2 * time.Second)

	// Config must still be active (short timer was cancelled; 60s timer hasn't fired).
	if !fake.IsFunnelOn() {
		t.Error("FNL-07: re-enable should have cancelled the prior 1s timer; config should still be active after 2s")
	}
}

// ---------------------------------------------------------------------------
// Phase 170-02 / FNL-08: reusable public read code — mint/idempotent/revoke
// regression suite. Task 1 (api_test.go) drives the mint site directly with
// a white-box injected funnelSessions/funnelReadCodeTTL; these tests exercise
// the full HTTP-driven Funnel lifecycle through the real handlers.
// ---------------------------------------------------------------------------

// TestFunnelPublicCode_ReadOnlyScope verifies T-170-01: the public read
// code's token (resolved via Exchange + Verify) carries read-only Perms —
// never "write" or capability.PermFilesWrite — in BOTH perm-matrix states
// (browse OFF and browse ON), proving the read-only invariant holds
// regardless of the browse toggle.
func TestFunnelPublicCode_ReadOnlyScope(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}

	run := func(t *testing.T, browseOn bool) {
		api, _, socketPath := testDaemon(t)
		_, _ = makeFunnelTestWebServer(t, api, "test.ts.net")
		key := configureCapabilityStateForTest(t, api, api.webServer)

		_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"public-code-scope","workDir":""}`)
		var cr CreateResponse
		if err := json.Unmarshal(body, &cr); err != nil {
			t.Fatalf("decode create: %v", err)
		}
		if browseOn {
			api.engine.SetSessionBrowse(cr.ID, true)
		}
		rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)
		enableFunnelViaHTTP(t, socketPath, cr.ID)

		_, _, _, _, publicReadCode, err := api.issueCapabilitiesForSession(cr.ID)
		if err != nil {
			t.Fatalf("issueCapabilitiesForSession: %v", err)
		}
		if publicReadCode == "" {
			t.Fatal("PublicReadCode must be non-empty for a Funnel session")
		}

		tok, err := api.joinCodes.Exchange(publicReadCode)
		if err != nil {
			t.Fatalf("Exchange(publicReadCode): %v", err)
		}
		claims, err := capability.Verify(tok, key)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if capability.HasPerm(claims.Perms, "write") {
			t.Errorf("T-170-01: Perms = %q must NOT contain write (browseOn=%v)", claims.Perms, browseOn)
		}
		if capability.HasPerm(claims.Perms, capability.PermFilesWrite) {
			t.Errorf("T-170-01: Perms = %q must NOT contain files.write (browseOn=%v)", claims.Perms, browseOn)
		}
	}

	t.Run("browse_off", func(t *testing.T) { run(t, false) })
	t.Run("browse_on", func(t *testing.T) { run(t, true) })
}

// TestIssueCapabilities_FunnelPublicCode_Idempotent verifies T-170-04: two
// consecutive issueCapabilitiesForSession calls for the same active Funnel
// session return the IDENTICAL PublicReadCode — a code already handed to a
// viewer must not silently rotate on a repeat capability re-issue (browse
// toggle, warm-up re-issue, modal reopen).
func TestIssueCapabilities_FunnelPublicCode_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	_, _ = makeFunnelTestWebServer(t, api, "test.ts.net")
	configureCapabilityStateForTest(t, api, api.webServer)

	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"public-code-idempotent","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)
	enableFunnelViaHTTP(t, socketPath, cr.ID)

	_, _, _, _, code1, err := api.issueCapabilitiesForSession(cr.ID)
	if err != nil {
		t.Fatalf("issueCapabilitiesForSession (1st): %v", err)
	}
	_, _, _, _, code2, err := api.issueCapabilitiesForSession(cr.ID)
	if err != nil {
		t.Fatalf("issueCapabilitiesForSession (2nd): %v", err)
	}
	if code1 == "" || code2 == "" {
		t.Fatalf("PublicReadCode must be non-empty: code1=%q code2=%q", code1, code2)
	}
	if code1 != code2 {
		t.Errorf("T-170-04: PublicReadCode rotated across re-issue: %q != %q", code1, code2)
	}
}

// TestFunnelAutoExpiry_RevokesPublicReadCode extends FNL-07/FNL-08: once the
// auto-expiry timer fires (disableFunnelForSession, trigger 5), the cached
// public read code must no longer Exchange — the timer-driven teardown path
// is one of the in-process triggers that must revoke the code (T-170-02),
// not just leave it to its own per-code TTL.
func TestFunnelAutoExpiry_RevokesPublicReadCode(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener and timer wait")
	}
	api, _, socketPath := testDaemon(t)
	_, fake := makeFunnelTestWebServer(t, api, "test.ts.net")
	configureCapabilityStateForTest(t, api, api.webServer)

	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"public-code-expiry","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)

	// Enable with 1-second auto-expiry.
	status, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", cr.ID), `{"enabled":true,"expiresIn":1}`)
	if status != http.StatusOK {
		t.Fatalf("enable with expiry: want 200, got %d", status)
	}

	_, _, _, _, publicReadCode, err := api.issueCapabilitiesForSession(cr.ID)
	if err != nil {
		t.Fatalf("issueCapabilitiesForSession: %v", err)
	}
	if publicReadCode == "" {
		t.Fatal("PublicReadCode must be non-empty before expiry")
	}

	// Poll for the timer to fire.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !fake.IsFunnelOn() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if fake.IsFunnelOn() {
		t.Fatal("precondition: expiry timer did not fire within 3s")
	}

	if _, err := api.joinCodes.Exchange(publicReadCode); !errors.Is(err, capability.ErrCodeNotFound) {
		t.Errorf("FNL-08 / T-170-02: after auto-expiry, Exchange(publicReadCode) = %v, want ErrCodeNotFound", err)
	}
}

// TestIssueCapabilities_NonFunnelSession_EmptyPublicReadCode verifies FNL-08
// at the public HTTP boundary: an ordinary web-enabled (non-Funnel) session's
// IssueCapabilitiesResponse carries PublicReadCode == "" — the reusable code
// exists only for internet (Funnel) shares.
func TestIssueCapabilities_NonFunnelSession_EmptyPublicReadCode(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	_, _ = makeFunnelTestWebServer(t, api, "test.ts.net")
	configureCapabilityStateForTest(t, api, api.webServer)

	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"non-funnel-public-code","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)

	st, issueBody := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/capabilities", cr.ID), ``)
	if st != http.StatusOK {
		t.Fatalf("capabilities: want 200, got %d; body: %s", st, issueBody)
	}
	var resp IssueCapabilitiesResponse
	if err := json.Unmarshal(issueBody, &resp); err != nil {
		t.Fatalf("decode IssueCapabilitiesResponse: %v", err)
	}
	if resp.PublicReadCode != "" {
		t.Errorf("FNL-08: non-Funnel session PublicReadCode = %q, want empty", resp.PublicReadCode)
	}
}

// ---------------------------------------------------------------------------
// Task 2: All five teardown triggers + ref-count guard
// ---------------------------------------------------------------------------

// TestFunnelTeardown_AllTriggers verifies FNL-05 — all five teardown triggers
// route through disableFunnelForSession and leave the fake serve config empty.
// Each sub-test starts from a freshly-enabled Funnel session and drives the
// specific trigger through its REAL entry point.
func TestFunnelTeardown_AllTriggers(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}

	// setup also mints the FNL-08 public read code (via configureCapabilityStateForTest
	// + issueCapabilitiesForSession) so each in-process trigger sub-test can assert it
	// was revoked (T-170-02), not just that the serve config emptied. Site 4
	// (daemon_stop) is deliberately excluded from this assertion below: ws.Stop()
	// calls DisableFunnel directly at the webserver layer, bypassing
	// disableFunnelForSession entirely (by design, per that function's doc comment) —
	// the plan's must_haves list only 4 in-process triggers (toggle-off,
	// web-share-off, session-exit, auto-expiry timer) for this invariant.
	setup := func(t *testing.T) (api *API, socketPath string, ws *webserver.WebServer, fake *daemonFakeFunnelClient, sessionID, publicReadCode string) {
		t.Helper()
		api, _, socketPath = testDaemon(t)
		ws, fake = makeFunnelTestWebServer(t, api, "test.ts.net")
		configureCapabilityStateForTest(t, api, ws)
		_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"teardown","workDir":""}`)
		var cr CreateResponse
		if err := json.Unmarshal(body, &cr); err != nil {
			t.Fatalf("decode create: %v", err)
		}
		enableFunnelViaHTTP(t, socketPath, cr.ID)
		if !fake.IsFunnelOn() {
			t.Fatal("precondition: fake config must show IsFunnelOn after enable")
		}
		_, _, _, _, publicReadCode, err := api.issueCapabilitiesForSession(cr.ID)
		if err != nil {
			t.Fatalf("issueCapabilitiesForSession: %v", err)
		}
		if publicReadCode == "" {
			t.Fatal("precondition: PublicReadCode must be non-empty after Funnel enable")
		}
		return api, socketPath, ws, fake, cr.ID, publicReadCode
	}

	t.Run("1_toggle_off", func(t *testing.T) {
		api, socketPath, _, fake, sid, publicReadCode := setup(t)
		status, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", sid), `{"enabled":false}`)
		if status != http.StatusNoContent {
			t.Fatalf("toggle-off: want 204, got %d", status)
		}
		if fake.IsFunnelOn() {
			t.Error("FNL-05: toggle-off did not clear fake config (IsFunnelOn still true)")
		}
		if _, err := api.joinCodes.Exchange(publicReadCode); !errors.Is(err, capability.ErrCodeNotFound) {
			t.Errorf("FNL-08 / T-170-02: toggle-off did not revoke public read code: Exchange = %v, want ErrCodeNotFound", err)
		}
	})

	t.Run("2_web_share_off", func(t *testing.T) {
		api, socketPath, _, fake, sid, publicReadCode := setup(t)
		// Enable web-share first so disable is meaningful.
		rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", sid), `{"enabled":true}`)
		// Disable web-share — must also teardown Funnel (site 2).
		status, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", sid), `{"enabled":false}`)
		if status != http.StatusNoContent {
			t.Fatalf("web-share-off: want 204, got %d", status)
		}
		if fake.IsFunnelOn() {
			t.Error("FNL-05: web-share-off did not clear fake config (IsFunnelOn still true)")
		}
		if _, err := api.joinCodes.Exchange(publicReadCode); !errors.Is(err, capability.ErrCodeNotFound) {
			t.Errorf("FNL-08 / T-170-02: web-share-off did not revoke public read code: Exchange = %v, want ErrCodeNotFound", err)
		}
	})

	t.Run("3_session_natural_end", func(t *testing.T) {
		api, _, _, fake, sid, publicReadCode := setup(t)
		// Invoke the same cleanup routine that the real onExit callback uses.
		api.runSessionExitCleanupForTest(sid)
		if fake.IsFunnelOn() {
			t.Error("FNL-05: runSessionExitCleanup did not clear fake config (IsFunnelOn still true)")
		}
		if _, err := api.joinCodes.Exchange(publicReadCode); !errors.Is(err, capability.ErrCodeNotFound) {
			t.Errorf("FNL-08 / T-170-02: runSessionExitCleanup did not revoke public read code: Exchange = %v, want ErrCodeNotFound", err)
		}
	})

	t.Run("4_daemon_stop", func(t *testing.T) {
		_, _, ws, fake, _, _ := setup(t)
		// ws.Stop() calls DisableFunnel (165-01 wired it); do NOT call
		// disableFunnelForSession here (that would double-fire).
		if err := ws.Stop(); err != nil {
			t.Fatalf("ws.Stop: %v", err)
		}
		if fake.IsFunnelOn() {
			t.Error("FNL-05: ws.Stop() did not clear fake config via 165-01 wiring (IsFunnelOn still true)")
		}
		// No public-read-code revocation assertion here: ws.Stop() intentionally
		// bypasses disableFunnelForSession (see that function's doc comment /
		// the plan's must_haves, which list only 4 in-process triggers for this
		// invariant) — the web listener itself is gone, so the code is moot.
	})

	t.Run("5_expiry_timer", func(t *testing.T) {
		if testing.Short() {
			t.Skip("requires timer wait")
		}
		api, socketPath, _, fake, sid, publicReadCode := setup(t)
		// Re-enable with a 1-second auto-expiry.
		status, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", sid), `{"enabled":true,"expiresIn":1}`)
		if status != http.StatusOK {
			t.Fatalf("enable with expiry: want 200, got %d", status)
		}
		// Poll for timer to fire.
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if !fake.IsFunnelOn() {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if fake.IsFunnelOn() {
			t.Error("FNL-05: expiry timer did not clear fake config (IsFunnelOn still true)")
		}
		if _, err := api.joinCodes.Exchange(publicReadCode); !errors.Is(err, capability.ErrCodeNotFound) {
			t.Errorf("FNL-08 / T-170-02: expiry timer did not revoke public read code: Exchange = %v, want ErrCodeNotFound", err)
		}
	})
}

// TestFunnelTeardown_RefCountKeepsSiblingUp verifies the reference-count guard
// (Anti-Pattern 3 from 165-RESEARCH.md / T-165-09):
//   - With sessions A and B both Funnel-enabled, tearing down A leaves the
//     fake config STILL active (len(funnelSessions)==1 → DisableFunnel NOT called).
//   - Tearing down B (len==0) calls DisableFunnel so the config goes empty.
func TestFunnelTeardown_RefCountKeepsSiblingUp(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	_, fake := makeFunnelTestWebServer(t, api, "test.ts.net")

	// Create two sessions.
	_, bodyA := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"refcount-a","workDir":""}`)
	var crA CreateResponse
	if err := json.Unmarshal(bodyA, &crA); err != nil {
		t.Fatalf("decode create A: %v", err)
	}
	_, bodyB := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"refcount-b","workDir":""}`)
	var crB CreateResponse
	if err := json.Unmarshal(bodyB, &crB); err != nil {
		t.Fatalf("decode create B: %v", err)
	}

	// Enable Funnel for both sessions.
	enableFunnelViaHTTP(t, socketPath, crA.ID)
	enableFunnelViaHTTP(t, socketPath, crB.ID)
	if !fake.IsFunnelOn() {
		t.Fatal("precondition: fake config must show IsFunnelOn after both enables")
	}

	// Tear down A — sibling B keeps config alive (ref-count > 0).
	status, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", crA.ID), `{"enabled":false}`)
	if status != http.StatusNoContent {
		t.Fatalf("disable A: want 204, got %d", status)
	}
	if !fake.IsFunnelOn() {
		t.Error("T-165-09: disabling session A must NOT disable Funnel while session B is still active")
	}

	// Tear down B — now len==0, DisableFunnel fires.
	status, _ = rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/funnel", crB.ID), `{"enabled":false}`)
	if status != http.StatusNoContent {
		t.Fatalf("disable B: want 204, got %d", status)
	}
	if fake.IsFunnelOn() {
		t.Error("T-165-09: disabling session B (last active) must call DisableFunnel (fake config still active)")
	}
}

// ---------------------------------------------------------------------------
// Task 3: Funnel-aware share-URL builders + daemon-restart lingering-Funnel clear
// ---------------------------------------------------------------------------

// TestIssueCapabilities_FunnelURL verifies FNL-03 in issueCapabilitiesForSession:
//   - With funnelSessions[id]==true and ws.FunnelBaseURL() non-empty, the issued
//     readURL/writeURL use the funnel host (no port component, not the tailnet :7443 port).
//   - Without funnelSessions[id] the URLs use the tailnet BaseURL (with port).
func TestIssueCapabilities_FunnelURL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	_, _ = makeFunnelTestWebServer(t, api, "test.ts.net")
	configureCapabilityStateForTest(t, api, api.webServer)

	// Create + web-enable a session.
	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"cap-url","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)

	// Issue capabilities WITHOUT Funnel enabled — URLs must use tailnet BaseURL (with port).
	st, issueBody := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/capabilities", cr.ID), ``)
	if st != http.StatusOK {
		t.Fatalf("capabilities without Funnel: want 200, got %d; body: %s", st, issueBody)
	}
	var noFunnelResp IssueCapabilitiesResponse
	if err := json.Unmarshal(issueBody, &noFunnelResp); err != nil {
		t.Fatalf("decode IssueCapabilitiesResponse: %v", err)
	}
	tailnetBase := api.webServer.BaseURL()
	if !strings.HasPrefix(noFunnelResp.ReadURL, tailnetBase) {
		t.Errorf("without Funnel: ReadURL %q must start with tailnet BaseURL %q", noFunnelResp.ReadURL, tailnetBase)
	}

	// Enable Funnel for the session.
	enableFunnelViaHTTP(t, socketPath, cr.ID)
	funnelBase := api.webServer.FunnelBaseURL()
	if funnelBase == "" {
		t.Fatal("FunnelBaseURL must be non-empty after enable")
	}

	// Issue capabilities WITH Funnel enabled — URLs must use FunnelBaseURL (no port).
	st, issueBody = rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/capabilities", cr.ID), ``)
	if st != http.StatusOK {
		t.Fatalf("capabilities with Funnel: want 200, got %d; body: %s", st, issueBody)
	}
	var funnelResp IssueCapabilitiesResponse
	if err := json.Unmarshal(issueBody, &funnelResp); err != nil {
		t.Fatalf("decode IssueCapabilitiesResponse: %v", err)
	}
	if !strings.HasPrefix(funnelResp.ReadURL, funnelBase) {
		t.Errorf("with Funnel: ReadURL %q must start with FunnelBaseURL %q", funnelResp.ReadURL, funnelBase)
	}
	if strings.Contains(funnelResp.ReadURL, ":443") || strings.Contains(funnelResp.ReadURL, ":7443") {
		t.Errorf("with Funnel: ReadURL %q must not contain port suffix", funnelResp.ReadURL)
	}
	// ?cap= token must still be present (capability gate intact).
	if !strings.Contains(funnelResp.ReadURL, "?cap=") {
		t.Errorf("with Funnel: ReadURL %q missing ?cap= token", funnelResp.ReadURL)
	}
}

// TestExchangeJoinCode_FunnelURL_GateIntact verifies FNL-03 in handleExchangeJoinCode:
//   - For a Funnel session, POST /join/exchange with a VALID code returns a URL
//     on the funnel host (no port) AND includes ?cap=<token>.
//   - The single-use gate is intact: reusing the same code returns 404.
//   - An unknown code returns 404 (Funnel URL does NOT bypass the gate).
func TestExchangeJoinCode_FunnelURL_GateIntact(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	_, _ = makeFunnelTestWebServer(t, api, "test.ts.net")
	configureCapabilityStateForTest(t, api, api.webServer)

	// Create + web-enable a session.
	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"join-funnel","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)

	// Enable Funnel.
	enableFunnelViaHTTP(t, socketPath, cr.ID)
	funnelBase := api.webServer.FunnelBaseURL()

	// Issue capabilities (gets join codes).
	st, issueBody := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/capabilities", cr.ID), ``)
	if st != http.StatusOK {
		t.Fatalf("capabilities: want 200, got %d; body: %s", st, issueBody)
	}
	var issue IssueCapabilitiesResponse
	if err := json.Unmarshal(issueBody, &issue); err != nil {
		t.Fatalf("decode IssueCapabilitiesResponse: %v", err)
	}

	// Exchange the read code — must return a Funnel URL with ?cap= token.
	st, xBody := rawPost(t, socketPath, "/join/exchange",
		fmt.Sprintf(`{"code":%q}`, issue.ReadCode))
	if st != http.StatusOK {
		t.Fatalf("exchange: want 200, got %d; body: %s", st, xBody)
	}
	var xResp ExchangeJoinCodeResponse
	if err := json.Unmarshal(xBody, &xResp); err != nil {
		t.Fatalf("decode ExchangeJoinCodeResponse: %v", err)
	}
	if !strings.HasPrefix(xResp.URL, funnelBase) {
		t.Errorf("exchange URL %q must start with FunnelBaseURL %q", xResp.URL, funnelBase)
	}
	if !strings.Contains(xResp.URL, "?cap=") {
		t.Errorf("exchange URL %q must contain ?cap= token", xResp.URL)
	}

	// Single-use gate: reusing the same code must return 404.
	st2, _ := rawPost(t, socketPath, "/join/exchange",
		fmt.Sprintf(`{"code":%q}`, issue.ReadCode))
	if st2 != http.StatusNotFound {
		t.Errorf("reused code: want 404, got %d", st2)
	}

	// Unknown code must return 404 (Funnel URL does not bypass the gate).
	st3, _ := rawPost(t, socketPath, "/join/exchange", `{"code":"unknown-fake-code-99"}`)
	if st3 != http.StatusNotFound {
		t.Errorf("unknown code: want 404, got %d", st3)
	}
}

// TestStartupClearsLingeringFunnel verifies that AutoStartWebServer calls
// ws.ClearLingeringFunnel when a lingering IsFunnelOn=true config is detected
// at startup (Open Question #3 / T-165-12):
//   - The fake serves an active Funnel config on GetServeConfig at startup.
//   - After AutoStartWebServer, the stored config has IsFunnelOn=false (cleared).
func TestStartupClearsLingeringFunnel(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, _ := testDaemon(t)
	lnCfg := newLoopbackTLSListener(t)

	// Build a fake that starts with a lingering IsFunnelOn=true config.
	fake := &daemonFakeFunnelClient{hostname: "test.ts.net"}
	// Seed the stored config with a lingering Funnel entry (simulates a prior crash).
	lingering := &ipn.ServeConfig{}
	lingering.SetFunnel("test.ts.net", 443, true)
	fake.mu.Lock()
	fake.storedConfig = lingering
	fake.mu.Unlock()
	if !fake.IsFunnelOn() {
		t.Fatal("precondition: fake must start with IsFunnelOn=true (lingering config)")
	}

	// AutoStartWebServer creates a new WebServer and calls Start() + ClearLingeringFunnel.
	// We intercept by setting up the WebServer manually and then wire the fake.
	// Since AutoStartWebServer doesn't accept an external fake, we use a different approach:
	// create the WebServer, inject the fake, call ClearLingeringFunnel directly (same code path).
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: lnCfg,
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	ws.SetFunnelClientForTest(fake)
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })

	// Call ClearLingeringFunnel — this is what AutoStartWebServer calls after Start.
	if err := ws.ClearLingeringFunnel(context.Background()); err != nil {
		t.Fatalf("ClearLingeringFunnel: %v", err)
	}

	// The lingering config must now be cleared.
	if fake.IsFunnelOn() {
		t.Error("T-165-12: ClearLingeringFunnel did not clear the lingering Funnel config")
	}
}

// ---------------------------------------------------------------------------
// Task 4 (165-04 gap-closure): Kill-path Funnel teardown (GAP 2 / FNL-05)
// ---------------------------------------------------------------------------

// TestFunnelTeardown_KillPath is the FNL-05 kill-path regression guard (165-04 GAP 2):
// An explicit kill (DELETE /sessions/{id} → handleDeleteSession) must tear down Funnel
// and remove the funnelSessions[id] ref-count entry synchronously — no 10s grace period.
//
// Two sub-cases:
//  1. Single Funnel session: after DELETE, fake config is empty (IsFunnelOn false) and
//     GET /sessions reports FunnelActive=false.
//  2. Two Funnel sessions A+B (ref-count guard): killing A leaves B's Funnel up
//     (IsFunnelOn still true); B's subsequent natural-exit cleanup then clears the config.
//     This proves the stale-ref-count regression (A's leftover entry blocking B's later
//     teardown) is gone — which is the precise root cause of GAP 2.
//
// Teardown is asserted via the fake funnelClient's stored serve config (Pitfall 5/13 guard —
// never assert via a.funnelSessions directly).
func TestFunnelTeardown_KillPath(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}

	// --- Sub-case 1: single session ---
	t.Run("single_session_killed", func(t *testing.T) {
		api, _, socketPath := testDaemon(t)
		_, fake := makeFunnelTestWebServer(t, api, "test.ts.net")

		// Create a session.
		_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"kill-single","workDir":""}`)
		var cr CreateResponse
		if err := json.Unmarshal(body, &cr); err != nil {
			t.Fatalf("decode create: %v", err)
		}

		// Enable Funnel via the real HTTP handler (Pitfall 5 guard).
		enableFunnelViaHTTP(t, socketPath, cr.ID)
		if !fake.IsFunnelOn() {
			t.Fatal("precondition: fake config must show IsFunnelOn=true after enable")
		}

		// Kill the session via DELETE /sessions/{id} — the real kill-path entry point.
		status, _ := rawDelete(t, socketPath, fmt.Sprintf("/sessions/%s", cr.ID))
		if status != http.StatusNoContent {
			t.Fatalf("DELETE /sessions/%s: want 204, got %d", cr.ID, status)
		}

		// Fake config must show IsFunnelOn=false: the kill path must synchronously
		// call runSessionExitCleanup → disableFunnelForSession → ws.DisableFunnel.
		// Before the 165-04 fix, handleDeleteSession returned 204 without any cleanup,
		// leaving Funnel exposed after kill (the GAP 2 defect).
		if fake.IsFunnelOn() {
			t.Error("FNL-05 kill path: DELETE session did not clear fake config (IsFunnelOn still true after kill)")
		}

		// GET /sessions must report FunnelActive=false (or the session absent).
		_, listBody := rawGet(t, socketPath, "/sessions")
		var sessions []SessionInfo
		if err := json.Unmarshal(listBody, &sessions); err != nil {
			t.Fatalf("decode sessions: %v; body: %s", err, listBody)
		}
		for _, s := range sessions {
			if s.ID == cr.ID && s.FunnelActive {
				t.Errorf("FNL-05: session %s FunnelActive=true after kill, want false", cr.ID)
			}
		}
	})

	// --- Sub-case 2: two Funnel sessions — ref-count guard ---
	t.Run("refcount_killing_a_keeps_b_up", func(t *testing.T) {
		api, _, socketPath := testDaemon(t)
		_, fake := makeFunnelTestWebServer(t, api, "test.ts.net")

		// Create sessions A and B.
		_, bodyA := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"kill-refcount-a","workDir":""}`)
		var crA CreateResponse
		if err := json.Unmarshal(bodyA, &crA); err != nil {
			t.Fatalf("decode create A: %v", err)
		}
		_, bodyB := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"kill-refcount-b","workDir":""}`)
		var crB CreateResponse
		if err := json.Unmarshal(bodyB, &crB); err != nil {
			t.Fatalf("decode create B: %v", err)
		}

		// Enable Funnel for both sessions.
		enableFunnelViaHTTP(t, socketPath, crA.ID)
		enableFunnelViaHTTP(t, socketPath, crB.ID)
		if !fake.IsFunnelOn() {
			t.Fatal("precondition: fake config must show IsFunnelOn=true after both enables")
		}

		// Kill session A via DELETE. With the 165-04 fix, A is removed from
		// funnelSessions synchronously. B's entry remains → len == 1 → DisableFunnel
		// must NOT be called yet (ref-count gate — sibling B is still active).
		status, _ := rawDelete(t, socketPath, fmt.Sprintf("/sessions/%s", crA.ID))
		if status != http.StatusNoContent {
			t.Fatalf("DELETE session A: want 204, got %d", status)
		}

		// Config must still be ACTIVE after killing A — B's Funnel must not be cut off.
		// (This also guards against an over-eager teardown where the kill path
		// calls DisableFunnel unconditionally instead of routing through the ref-counted
		// disableFunnelForSession helper.)
		if !fake.IsFunnelOn() {
			t.Error("T-165-09 / kill ref-count: killing A must NOT disable Funnel while B is still active")
		}

		// Tear down B via the natural-exit cleanup path. With the 165-04 fix A's
		// funnelSessions entry was already removed at kill time, so B's cleanup sees
		// len == 0 and calls DisableFunnel. Without the fix, A's stale entry leaves
		// len == 1 → DisableFunnel is never called → config stays active (the GAP 2
		// stale-ref-count regression that blocked sibling teardown).
		api.runSessionExitCleanupForTest(crB.ID)

		// Config must be EMPTY now — the stale-ref-count regression is gone.
		if fake.IsFunnelOn() {
			t.Error("FNL-05 kill ref-count: B's natural-exit cleanup did not clear fake config — " +
				"stale funnelSessions[A] entry from the kill path is blocking DisableFunnel (GAP 2)")
		}
	})
}

// grantIDBehindPublicCode Exchanges a REUSABLE public-share code (a
// non-destructive read, since reusable codes survive Exchange) and returns the
// GrantID embedded in the token it resolves to. Used to assert, via probeGrant,
// whether the grant the public code currently points at is still live.
func grantIDBehindPublicCode(t *testing.T, api *API, key []byte, code string) string {
	t.Helper()
	tok, err := api.joinCodes.Exchange(code)
	if err != nil {
		t.Fatalf("Exchange(%q): %v", code, err)
	}
	claims, err := capability.Verify(tok, key)
	if err != nil {
		t.Fatalf("Verify token behind %q: %v", code, err)
	}
	return claims.GrantID
}

// TestIssueCapabilities_BrowseToggleRebindsPublicCode is the CR-01 regression:
// a reusable public-share code already handed to viewers must keep resolving to
// a LIVE grant after the owner toggles "Enable remote file browsing" — which
// calls ws.ClearGrants and then re-issues a NEW underlying read token. Before
// the fix the code stayed pinned to the first token, whose grant ClearGrants
// wiped, so every public viewer hit 403 ("capability has been revoked") while
// the UI still displayed the (dead) code. After the fix
// issueCapabilitiesForSession rebinds the stable code onto the freshly-granted
// token (T-170-04 keeps the code string; CR-01 keeps it live).
func TestIssueCapabilities_BrowseToggleRebindsPublicCode(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	ws, _ := makeFunnelTestWebServer(t, api, "test.ts.net")
	key := configureCapabilityStateForTest(t, api, api.webServer)
	// probeGrant hits /api/sessions/{id}/info, which 404s (not 200) when no
	// session resolver is set even for a live grant — supply one so a 200
	// unambiguously means "grant active".
	ws.SetSessionResolver(func(string) (string, string, string, string) {
		return "browse-toggle-rebind", "cat", "running", "localhost"
	})

	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"browse-toggle-rebind","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)
	enableFunnelViaHTTP(t, socketPath, cr.ID)

	// First issuance mints the reusable public code; the grant behind it is live.
	_, _, _, _, code1, err := api.issueCapabilitiesForSession(cr.ID)
	if err != nil {
		t.Fatalf("issueCapabilitiesForSession (1st): %v", err)
	}
	if code1 == "" {
		t.Fatal("precondition: PublicReadCode must be non-empty for a Funnel session")
	}
	grant1 := grantIDBehindPublicCode(t, api, key, code1)
	if !probeGrant(t, ws, key, cr.ID, grant1) {
		t.Fatal("precondition: grant behind the fresh public code is not active")
	}

	// Browse toggle: the real handleSetSessionBrowse clears the session's grants
	// before the frontend re-issues capabilities. Drive that clear directly.
	ws.ClearGrants(cr.ID)
	if probeGrant(t, ws, key, cr.ID, grant1) {
		t.Fatal("precondition: ClearGrants did not revoke the original grant")
	}

	// Re-issue (what the frontend does immediately after the toggle). The public
	// code must NOT rotate (T-170-04) AND must now resolve to a LIVE grant.
	_, _, _, _, code2, err := api.issueCapabilitiesForSession(cr.ID)
	if err != nil {
		t.Fatalf("issueCapabilitiesForSession (2nd): %v", err)
	}
	if code2 != code1 {
		t.Errorf("T-170-04: public code rotated across browse-toggle re-issue: %q != %q", code1, code2)
	}
	grant2 := grantIDBehindPublicCode(t, api, key, code2)
	if !probeGrant(t, ws, key, cr.ID, grant2) {
		t.Error("CR-01: after browse toggle + re-issue the public code still resolves to a revoked grant — " +
			"public viewers would 403; issueCapabilitiesForSession must Rebind it onto the fresh token")
	}
}

// TestIssueCapabilities_ExpiredPublicCodeRemints is the WR-02 regression: an
// "Until I disable" (ExpiresIn==0) share keeps its public URL up indefinitely,
// but the reusable code itself is TTL-capped at funnelReadCodeMaxTTL (8h). Once
// that backstop elapses the cached code is dead, yet its stale string stays
// cached — before the fix the mint gate keyed only on presence, so a re-issue
// handed back the dead code forever (no recovery short of a Funnel off→on
// cycle). After the fix the gate treats an expired cache as absent and re-mints
// a fresh, working code.
//
// The join-code manager's own clock cannot be advanced from this package
// (SetClockForTest is capability-internal), so the test drives the daemon's
// re-mint gate directly by forcing the cached expiry into the past — which is
// exactly the wall-clock condition WR-02 keys on.
func TestIssueCapabilities_ExpiredPublicCodeRemints(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	_, _ = makeFunnelTestWebServer(t, api, "test.ts.net")
	configureCapabilityStateForTest(t, api, api.webServer)

	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"expired-code-remint","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)
	// expiresIn omitted == 0 ("Until I disable"): no auto-expiry timer; the code
	// TTL is backstopped at funnelReadCodeMaxTTL.
	enableFunnelViaHTTP(t, socketPath, cr.ID)

	_, _, _, _, code1, err := api.issueCapabilitiesForSession(cr.ID)
	if err != nil {
		t.Fatalf("issueCapabilitiesForSession (1st): %v", err)
	}
	if code1 == "" {
		t.Fatal("precondition: PublicReadCode must be non-empty for a Funnel session")
	}
	if _, err := api.joinCodes.Exchange(code1); err != nil {
		t.Fatalf("precondition: fresh public code must Exchange: %v", err)
	}

	// Simulate the 8h backstop having elapsed: force the cached expiry into the
	// past so the re-mint gate sees the cache as stale.
	api.mu.Lock()
	api.funnelReadCodeExpiry[cr.ID] = time.Now().Add(-time.Minute)
	api.mu.Unlock()

	// Re-issue: the expired cache must be treated as absent and re-minted.
	_, _, _, _, code2, err := api.issueCapabilitiesForSession(cr.ID)
	if err != nil {
		t.Fatalf("issueCapabilitiesForSession (2nd): %v", err)
	}
	if code2 == "" {
		t.Fatal("WR-02: re-issue after backstop must yield a non-empty code")
	}
	if code2 == code1 {
		t.Errorf("WR-02: re-issue after the cached code expired returned the stale code %q; "+
			"the gate must treat an expired cache as absent and re-mint", code1)
	}
	if _, err := api.joinCodes.Exchange(code2); err != nil {
		t.Errorf("WR-02: the re-minted public code must Exchange (be live), got %v", err)
	}
}

// TestIssueCapabilities_TeardownDuringMint_NoOrphanCode is the WR-01 regression:
// the reusable public code must NOT be minted for a Funnel session that was torn
// down in the TOCTOU window between issueCapabilitiesForSession's base-URL
// membership read and its mint critical section. Before the fix the mint keyed
// only on the stale isFunnelSession read, so a disableFunnelForSession that
// interleaved there left an orphaned, resolvable public code that teardown had
// already run past and would never revoke — violating "the code dies with the
// share" (T-170-02). After the fix the mint gate re-checks funnelSessions
// membership under the same lock and skips the mint (publicReadCode == "").
//
// The interleave is driven deterministically via mintRaceHookForTest (set to the
// teardown), NOT a goroutine race: this is a TOCTOU logic race (both reads are
// correctly locked, just at different times), so -race would not flag it and a
// stress loop would be flaky.
func TestIssueCapabilities_TeardownDuringMint_NoOrphanCode(t *testing.T) {
	if testing.Short() {
		t.Skip("requires TLS listener")
	}
	api, _, socketPath := testDaemon(t)
	_, _ = makeFunnelTestWebServer(t, api, "test.ts.net")
	configureCapabilityStateForTest(t, api, api.webServer)

	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"teardown-during-mint","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)
	enableFunnelViaHTTP(t, socketPath, cr.ID)

	// Arrange the teardown to fire in the mint window on the next issuance: the
	// base-URL read will observe funnelSessions[cr.ID]==true, then the hook tears
	// the session down before the mint lock re-checks membership.
	hookFired := false
	api.mintRaceHookForTest = func() {
		hookFired = true
		api.disableFunnelForSession(context.Background(), cr.ID)
	}

	_, _, _, _, publicReadCode, err := api.issueCapabilitiesForSession(cr.ID)
	if err != nil {
		t.Fatalf("issueCapabilitiesForSession: %v", err)
	}
	if !hookFired {
		t.Fatal("precondition: mint-race hook never fired — the isFunnelSession branch was not entered")
	}
	if publicReadCode != "" {
		t.Errorf("WR-01: minted an orphan public code %q for a session torn down in the TOCTOU window; "+
			"the under-lock funnelSessions re-check must skip the mint", publicReadCode)
	}
	// Defense in depth: no cached code entry survived the teardown-during-mint.
	api.mu.Lock()
	_, cachedExists := api.funnelReadCode[cr.ID]
	api.mu.Unlock()
	if cachedExists {
		t.Error("WR-01: a cached public code entry survived teardown-during-mint")
	}
}
