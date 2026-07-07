// Package webserver funnel lifecycle tests (Phase 165, FNL-01..FNL-06).
//
// Wave 0 scaffolding: fakeFunnelClient test double + compile-smoke test.
// Tasks 2–3 extend this file with EnableFunnel/DisableFunnel/Stop/Origin tests.
//
// All tests drive state through ws.EnableFunnel / ws.DisableFunnel (Pitfall 5
// guard: never set ws.funnelActive directly — mirrors Phase 150 shell-warning
// lesson from MEMORY.md).
package webserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

// fakeFunnelClient is a test double for the funnelClient interface.
// All fields are overridable func fields — set only the ones your test cares
// about; any nil field will panic if called (deliberate: reveals missing setup).
type fakeFunnelClient struct {
	getServeConfig     func(ctx context.Context) (*ipn.ServeConfig, error)
	setServeConfig     func(ctx context.Context, cfg *ipn.ServeConfig) error
	statusWithoutPeers func(ctx context.Context) (*ipnstate.Status, error)
}

func (f *fakeFunnelClient) GetServeConfig(ctx context.Context) (*ipn.ServeConfig, error) {
	return f.getServeConfig(ctx)
}

func (f *fakeFunnelClient) SetServeConfig(ctx context.Context, cfg *ipn.ServeConfig) error {
	return f.setServeConfig(ctx, cfg)
}

func (f *fakeFunnelClient) StatusWithoutPeers(ctx context.Context) (*ipnstate.Status, error) {
	return f.statusWithoutPeers(ctx)
}

// validFunnelStatus returns a *ipnstate.Status whose Self node has the full
// set of capabilities required for CheckFunnelAccess(443, Self) to succeed:
//
//   - tailcfg.CapabilityHTTPS ("https")
//   - tailcfg.NodeAttrFunnel ("funnel")
//   - tailcfg.CapabilityFunnelPorts with ?ports=443,8443,10000
func validFunnelStatus(hostname string) *ipnstate.Status {
	return &ipnstate.Status{
		Self: &ipnstate.PeerStatus{
			DNSName: hostname + ".",
			CapMap: tailcfg.NodeCapMap{
				tailcfg.CapabilityHTTPS: nil,
				tailcfg.NodeAttrFunnel:  nil,
				// Port allowlist URL — ipn.CheckFunnelPort scans keys with
				// HasPrefix(CapabilityFunnelPorts) and parses ?ports= query param.
				tailcfg.NodeCapability("https://tailscale.com/cap/funnel-ports?ports=443,8443,10000"): nil,
			},
		},
	}
}

// Compile-time assertion: *fakeFunnelClient satisfies the funnelClient interface.
var _ funnelClient = (*fakeFunnelClient)(nil)

// TestFunnelClient_CompileSmoke verifies that:
//  1. The funnelClient interface exists and fakeFunnelClient satisfies it.
//  2. NewWebServer sets ws.funnelClient = &ws.lc by default (FNL-02 production path).
//  3. A test can inject a fakeFunnelClient and have ws.funnelClient call it.
func TestFunnelClient_CompileSmoke(t *testing.T) {
	ws, _ := testServer(t)

	// Default: NewWebServer wires ws.funnelClient = &ws.lc.
	if ws.funnelClient == nil {
		t.Fatal("expected ws.funnelClient non-nil after NewWebServer")
	}

	// Injection path: replace with a fake and verify the seam is called.
	invoked := false
	fake := &fakeFunnelClient{
		getServeConfig: func(_ context.Context) (*ipn.ServeConfig, error) {
			invoked = true
			return nil, nil
		},
		setServeConfig: func(_ context.Context, _ *ipn.ServeConfig) error { return nil },
		statusWithoutPeers: func(_ context.Context) (*ipnstate.Status, error) {
			return &ipnstate.Status{Self: &ipnstate.PeerStatus{}}, nil
		},
	}
	ws.funnelClient = fake

	_, _ = ws.funnelClient.GetServeConfig(context.Background())
	if !invoked {
		t.Fatal("expected GetServeConfig to be routed through the injected fake")
	}
}

// ---------------------------------------------------------------------------
// Task 2: EnableFunnel / DisableFunnel / FunnelBaseURL / Stop teardown
// ---------------------------------------------------------------------------

// TestEnableFunnelCallsSetServeConfig verifies the happy path (FNL-02):
//   - EnableFunnel(ctx, 443) calls SetServeConfig with IsFunnelOn() == true.
//   - FunnelBaseURL() returns "https://<hostname>" with NO port component.
//
// Drives state through ws.EnableFunnel (Pitfall 5 guard).
func TestEnableFunnelCallsSetServeConfig(t *testing.T) {
	ws, _ := testServer(t)

	const funnelHostname = "mynode.ts.net"

	var capturedConfig *ipn.ServeConfig
	fake := &fakeFunnelClient{
		statusWithoutPeers: func(_ context.Context) (*ipnstate.Status, error) {
			return validFunnelStatus(funnelHostname), nil
		},
		getServeConfig: func(_ context.Context) (*ipn.ServeConfig, error) { return nil, nil },
		setServeConfig: func(_ context.Context, cfg *ipn.ServeConfig) error {
			capturedConfig = cfg
			return nil
		},
	}
	ws.funnelClient = fake

	if err := ws.EnableFunnel(context.Background(), 443); err != nil {
		t.Fatalf("EnableFunnel: %v", err)
	}

	if capturedConfig == nil {
		t.Fatal("expected SetServeConfig to be called with a non-nil config")
	}
	if !capturedConfig.IsFunnelOn() {
		t.Errorf("expected IsFunnelOn()=true in the config passed to SetServeConfig")
	}

	// FunnelBaseURL must be "https://<hostname>" — no port for 443 (FNL-04).
	want := "https://" + funnelHostname
	if got := ws.FunnelBaseURL(); got != want {
		t.Errorf("FunnelBaseURL() = %q, want %q", got, want)
	}
}

// TestEnableFunnel_PrereqCheckPreventsSetServeConfig verifies that when
// CheckFunnelAccess fails (node lacks funnel capabilities), EnableFunnel
// returns the error verbatim and never calls SetServeConfig (FNL-06).
func TestEnableFunnel_PrereqCheckPreventsSetServeConfig(t *testing.T) {
	ws, _ := testServer(t)

	setServeConfigCalled := false
	fake := &fakeFunnelClient{
		// Return a Self without funnel capabilities → CheckFunnelAccess fails.
		statusWithoutPeers: func(_ context.Context) (*ipnstate.Status, error) {
			return &ipnstate.Status{
				Self: &ipnstate.PeerStatus{
					DNSName: "mynode.ts.net.",
					CapMap:  tailcfg.NodeCapMap{}, // no HTTPS/funnel caps
				},
			}, nil
		},
		getServeConfig: func(_ context.Context) (*ipn.ServeConfig, error) { return nil, nil },
		setServeConfig: func(_ context.Context, _ *ipn.ServeConfig) error {
			setServeConfigCalled = true
			return nil
		},
	}
	ws.funnelClient = fake

	err := ws.EnableFunnel(context.Background(), 443)
	if err == nil {
		t.Fatal("expected an error from CheckFunnelAccess, got nil")
	}
	// Error must contain the Tailscale verbatim string (FNL-06 — surface verbatim).
	if !strings.Contains(err.Error(), "Funnel not available") {
		t.Errorf("expected verbatim CheckFunnelAccess error, got %q", err.Error())
	}
	if setServeConfigCalled {
		t.Fatal("SetServeConfig must not be called when CheckFunnelAccess fails")
	}
	if ws.FunnelBaseURL() != "" {
		t.Errorf("expected FunnelBaseURL empty (funnelActive=false) after prereq failure, got %q", ws.FunnelBaseURL())
	}
}

// TestEnableFunnel_FallbackModeSafe verifies that when StatusWithoutPeers
// returns an error (local-network fallback mode), EnableFunnel returns that
// error, funnelActive stays false, and SetServeConfig is never called (FNL-03
// guard / T-165-03).
func TestEnableFunnel_FallbackModeSafe(t *testing.T) {
	ws, _ := testServer(t)

	wantErr := errors.New("tailscaled not running")
	setServeConfigCalled := false
	fake := &fakeFunnelClient{
		statusWithoutPeers: func(_ context.Context) (*ipnstate.Status, error) {
			return nil, wantErr
		},
		getServeConfig: func(_ context.Context) (*ipn.ServeConfig, error) { return nil, nil },
		setServeConfig: func(_ context.Context, _ *ipn.ServeConfig) error {
			setServeConfigCalled = true
			return nil
		},
	}
	ws.funnelClient = fake

	err := ws.EnableFunnel(context.Background(), 443)
	if err == nil {
		t.Fatal("expected error when StatusWithoutPeers fails")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("got error %v, want to wrap %v", err, wantErr)
	}
	if setServeConfigCalled {
		t.Fatal("SetServeConfig must not be called in fallback mode")
	}
	if ws.FunnelBaseURL() != "" {
		t.Errorf("funnelActive must remain false in fallback mode; FunnelBaseURL=%q", ws.FunnelBaseURL())
	}
}

// TestWebServerStop_DisablesFunnel verifies teardown site 4 (FNL-05):
//   - After EnableFunnel succeeds, ws.Stop() calls DisableFunnel.
//   - The fake serve config has IsFunnelOn()==false after Stop.
//   - FunnelBaseURL() returns "" after Stop.
func TestWebServerStop_DisablesFunnel(t *testing.T) {
	ws, _ := testServer(t)

	const funnelHostname = "mynode.ts.net"

	// Stateful fake: SetServeConfig stores the last-written config so
	// GetServeConfig returns it on the next call (ETag read-modify-write).
	var storedConfig *ipn.ServeConfig
	fake := &fakeFunnelClient{
		statusWithoutPeers: func(_ context.Context) (*ipnstate.Status, error) {
			return validFunnelStatus(funnelHostname), nil
		},
		getServeConfig: func(_ context.Context) (*ipn.ServeConfig, error) {
			return storedConfig, nil
		},
		setServeConfig: func(_ context.Context, cfg *ipn.ServeConfig) error {
			storedConfig = cfg
			return nil
		},
	}
	ws.funnelClient = fake

	// Enable Funnel via the real path (Pitfall 5 guard).
	if err := ws.EnableFunnel(context.Background(), 443); err != nil {
		t.Fatalf("EnableFunnel: %v", err)
	}
	if storedConfig == nil || !storedConfig.IsFunnelOn() {
		t.Fatal("precondition: EnableFunnel must produce IsFunnelOn=true")
	}

	// Stop() must call DisableFunnel before closing the listener.
	if err := ws.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}

	// The fake's stored config must no longer have Funnel on.
	if storedConfig != nil && storedConfig.IsFunnelOn() {
		t.Error("expected IsFunnelOn=false after Stop(); Funnel teardown missing")
	}
	// FunnelBaseURL must be cleared.
	if got := ws.FunnelBaseURL(); got != "" {
		t.Errorf("expected FunnelBaseURL=\"\" after Stop(), got %q", got)
	}
}

// TestEnableFunnel_ProxyTargetReachable is the FNL-03 502 regression guard (165-05 loopback-HTTP fix):
//
// Asserts the loopback-HTTP proxy target SHAPE and real loopback reachability:
//   - EnableFunnel's serve-config Proxy equals http://127.0.0.1:<loopbackPort> — scheme
//     is plain "http" (NOT "https", NOT "https+insecure"), host is "127.0.0.1" (NOT
//     ws.config.BindIP/FQDN), and the port is the plain-HTTP loopback listener's port
//     (NOT the TLS listener's port). The 165-04 Option A target
//     https+insecure://<bindIP>:<tlsPort> is dead: tailscaled sends SNI=IP-literal on
//     hop 2, the SNI-driven cert (ws.lc.GetCertificate) has no cert for an IP literal →
//     TLS internal_error → 502. https+insecure only disables the CLIENT's cert verification,
//     not the SERVER's SNI-based cert selection.
//   - The proxy target is actually reachable: a PLAIN http.Client (no TLS transport)
//     GETs http://127.0.0.1:<loopbackPort> and receives a real HTTP response — NOT
//     connection-refused (the exact 502 condition this tests against).
//   - Anti-regression guard: scheme=="https" or scheme=="https+insecure" or
//     port==TLS-listener-port fails the test explicitly with a 165-04-regression message.
//
// NOTE: this unit test guards target SHAPE + loopback reachability only.
// The SNI/ingress failure (FNL-03 root cause) is only reproducible on a live tailnet.
// Live M-34 (off-tailnet device, no Tailscale client) is the real end-to-end gate.
// Given the 165-04 false-green precedent (loopback self-signed cert answered any SNI),
// the automated test alone is NOT sufficient to certify FNL-03 closed.
//
// State is driven through ws.EnableFunnel (Pitfall 5 guard; never set ws.funnelActive directly).
func TestEnableFunnel_ProxyTargetReachable(t *testing.T) {
	if testing.Short() {
		t.Skip("requires listeners and network dial")
	}
	ws, _ := testServer(t)

	const funnelHostname = "mynode.ts.net"

	// Get the TLS listener port — the loopback port MUST differ from this.
	_, tlsPort, err := net.SplitHostPort(ws.Addr())
	if err != nil {
		t.Fatalf("ws.Addr() %q: SplitHostPort: %v", ws.Addr(), err)
	}

	// Get the loopback listener port directly (package-internal access — funnel_test.go
	// is package webserver). The loopback listener is started by startTailscale (invoked
	// by testServer's ws.Start call) and is already live.
	ws.mu.RLock()
	loopbackLn := ws.loopbackListener
	ws.mu.RUnlock()
	if loopbackLn == nil {
		t.Fatal("loopbackListener is nil — Task 1 must set it in startTailscale")
	}
	_, loopbackPort, err := net.SplitHostPort(loopbackLn.Addr().String())
	if err != nil {
		t.Fatalf("loopbackListener.Addr() %q: SplitHostPort: %v", loopbackLn.Addr(), err)
	}
	if loopbackPort == tlsPort {
		t.Fatalf("loopback port %q == TLS port %q — they must be different ephemeral ports", loopbackPort, tlsPort)
	}

	// Stateful fake: SetServeConfig stores the last-written config so GetServeConfig
	// returns it on the next call (ETag read-modify-write invariant — Pitfall 3 / T-165-04,
	// mirrors TestWebServerStop_DisablesFunnel pattern).
	var storedConfig *ipn.ServeConfig
	fake := &fakeFunnelClient{
		statusWithoutPeers: func(_ context.Context) (*ipnstate.Status, error) {
			return validFunnelStatus(funnelHostname), nil
		},
		getServeConfig: func(_ context.Context) (*ipn.ServeConfig, error) {
			return storedConfig, nil
		},
		setServeConfig: func(_ context.Context, cfg *ipn.ServeConfig) error {
			storedConfig = cfg
			return nil
		},
	}
	ws.funnelClient = fake

	if err := ws.EnableFunnel(context.Background(), 443); err != nil {
		t.Fatalf("EnableFunnel: %v", err)
	}

	if storedConfig == nil {
		t.Fatal("expected SetServeConfig to be called with a non-nil config")
	}

	// Extract the Proxy value from the Web handler.
	hp := ipn.HostPort(net.JoinHostPort(funnelHostname, "443"))
	web, ok := storedConfig.Web[hp]
	if !ok {
		t.Fatalf("no Web entry for HostPort %q in serve config", hp)
	}
	handler, ok := web.Handlers["/"]
	if !ok {
		t.Fatal("no '/' Web handler in serve config")
	}

	// Parse the proxy URL for fine-grained assertions.
	proxyURL, err := url.Parse(handler.Proxy)
	if err != nil {
		t.Fatalf("url.Parse(handler.Proxy=%q): %v", handler.Proxy, err)
	}

	// Scheme must be plain "http" — NOT "https" or "https+insecure".
	// https+insecure is the dead 165-04 Option A target (SNI-IP 502 root cause unchanged).
	if proxyURL.Scheme != "http" {
		t.Errorf("EnableFunnel Proxy scheme = %q, want \"http\"\n"+
			"  \"https\" or \"https+insecure\" is the DEAD 165-04 target: tailscaled sends SNI=IP\n"+
			"  on hop 2; the SNI-driven cert has no cert for an IP literal → TLS internal_error\n"+
			"  → HTTP 502 for every external Funnel guest (FNL-03 165-05 anti-regression guard).",
			proxyURL.Scheme)
	}

	// Host must be 127.0.0.1 — NOT ws.config.BindIP/FQDN.
	proxyHost, proxyPort, err := net.SplitHostPort(proxyURL.Host)
	if err != nil {
		t.Fatalf("net.SplitHostPort(proxyURL.Host=%q): %v", proxyURL.Host, err)
	}
	if proxyHost != "127.0.0.1" {
		t.Errorf("EnableFunnel Proxy host = %q, want \"127.0.0.1\"\n"+
			"  A non-loopback host would expose plaintext traffic on a real network interface.",
			proxyHost)
	}

	// Port must equal the loopback HTTP listener's port — NOT the TLS listener's port.
	if proxyPort != loopbackPort {
		t.Errorf("EnableFunnel Proxy port = %q, want loopback port %q (not TLS port %q)\n"+
			"  If the port equals the TLS port, hop 2 targets the HTTPS listener (SNI-IP → 502).",
			proxyPort, loopbackPort, tlsPort)
	}
	if proxyPort == tlsPort {
		t.Errorf("EnableFunnel Proxy port = TLS port %q — must target the loopback HTTP listener, not the TLS listener",
			tlsPort)
	}

	// Full canonical form check (belt-and-suspenders).
	wantProxy := "http://" + net.JoinHostPort("127.0.0.1", loopbackPort)
	if handler.Proxy != wantProxy {
		t.Errorf("EnableFunnel Proxy = %q, want %q", handler.Proxy, wantProxy)
	}

	// Reachability leg: a PLAIN http.Client (no TLS transport) must connect to the
	// loopback HTTP target and receive a real HTTP response.
	// Connection-refused or timeout is the exact dead-target condition and must fail.
	//
	// DO NOT use an InsecureSkipVerify HTTPS client here — that was the 165-04
	// false-green: a loopback self-signed cert answered any SNI, so the SNI/cert
	// mismatch never surfaced. This test must use plain HTTP to match the target scheme.
	dialURL := "http://" + net.JoinHostPort("127.0.0.1", loopbackPort)
	plain := &http.Client{Timeout: 2 * time.Second}
	resp, dialErr := plain.Get(dialURL)
	if dialErr != nil {
		t.Fatalf("reachability: GET %q failed (connection-refused = dead target, the 502 condition): %v",
			dialURL, dialErr)
	}
	defer resp.Body.Close()
	// Any HTTP status is acceptable — the point is the connection was accepted and served.
	t.Logf("reachability: GET %q → %d (loopback HTTP target live, FNL-03 proxy-target layer closed)", dialURL, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Task 3: Dual-origin allowlist
// ---------------------------------------------------------------------------

// TestRequireAllowedOrigin_FunnelOrigin verifies the dual-origin allowlist (FNL-04 / T-165-01):
//
//   - Funnel origin → 403 when Funnel inactive (fail-closed).
//   - Funnel origin → 200 after ws.EnableFunnel(ctx,443) via the fake.
//   - Tailnet BaseURL origin → 200 (unaffected by Funnel).
//   - Unrelated origin → 403 (fail-closed).
//
// State driven through ws.EnableFunnel (not ws.funnelActive directly — Pitfall 5).
func TestRequireAllowedOrigin_FunnelOrigin(t *testing.T) {
	ws, _ := testServer(t)

	const funnelHostname = "testhost.ts.net"
	funnelURL := "https://" + funnelHostname

	fake := &fakeFunnelClient{
		statusWithoutPeers: func(_ context.Context) (*ipnstate.Status, error) {
			return validFunnelStatus(funnelHostname), nil
		},
		getServeConfig: func(_ context.Context) (*ipn.ServeConfig, error) { return nil, nil },
		setServeConfig: func(_ context.Context, _ *ipn.ServeConfig) error { return nil },
	}
	ws.funnelClient = fake

	// invokeMiddleware calls requireAllowedOrigin with the given origin and
	// returns (innerHandlerCalled, statusCode).
	invokeMiddleware := func(origin string) (bool, int) {
		var called bool
		h := ws.requireAllowedOrigin(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/sessions/x/ws", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		h(rec, req)
		return called, rec.Code
	}

	t.Run("FunnelOrigin_Inactive_403", func(t *testing.T) {
		called, code := invokeMiddleware(funnelURL)
		if called {
			t.Fatal("expected inner handler NOT called for Funnel origin when Funnel inactive")
		}
		if code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", code)
		}
	})

	// Enable Funnel via the real code path (Pitfall 5 guard).
	if err := ws.EnableFunnel(context.Background(), 443); err != nil {
		t.Fatalf("EnableFunnel: %v", err)
	}

	t.Run("FunnelOrigin_Active_200", func(t *testing.T) {
		called, code := invokeMiddleware(funnelURL)
		if !called {
			t.Fatal("expected inner handler called for Funnel origin when Funnel active")
		}
		if code != http.StatusOK {
			t.Errorf("expected 200, got %d", code)
		}
	})

	t.Run("TailnetOrigin_Unaffected_200", func(t *testing.T) {
		called, code := invokeMiddleware(ws.BaseURL())
		if !called {
			t.Fatal("expected inner handler called for tailnet BaseURL origin")
		}
		if code != http.StatusOK {
			t.Errorf("expected 200, got %d", code)
		}
	})

	t.Run("UnrelatedOrigin_403", func(t *testing.T) {
		called, code := invokeMiddleware("https://evil.example")
		if called {
			t.Fatal("expected inner handler NOT called for unrelated origin")
		}
		if code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", code)
		}
	})
}

// TestOriginAllowedForWrite_FunnelOrigin verifies originAllowedForWrite dual-origin
// logic (capability_mw.go, T-165-01 / CAP-03), now RW-gate aware (Phase 171
// Plan 01, FNL-09 D-02):
//
//   - Empty Origin passes vacuously (desktop Wails fetch).
//   - Funnel origin fails when Funnel inactive (fail-closed).
//   - Funnel origin STILL fails after ws.EnableFunnel when the session has
//     not passed the RW gate (T-171-05 — Funnel-active alone is no longer
//     sufficient).
//   - Funnel origin passes once SetRWGate(sessionID, true) has run.
//   - Tailnet BaseURL origin still passes UNAFFECTED by the RW gate (the
//     gate is a Funnel-origin-only defense-in-depth check).
//   - Unrelated origin still fails.
func TestOriginAllowedForWrite_FunnelOrigin(t *testing.T) {
	ws, _ := testServer(t)

	const funnelHostname = "testhost.ts.net"
	const sessionID = "sess-171-origin-gate"
	funnelURL := "https://" + funnelHostname

	fake := &fakeFunnelClient{
		statusWithoutPeers: func(_ context.Context) (*ipnstate.Status, error) {
			return validFunnelStatus(funnelHostname), nil
		},
		getServeConfig: func(_ context.Context) (*ipn.ServeConfig, error) { return nil, nil },
		setServeConfig: func(_ context.Context, _ *ipn.ServeConfig) error { return nil },
	}
	ws.funnelClient = fake

	reqWithOrigin := func(origin string) *http.Request {
		req := httptest.NewRequest("GET", "/", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		return req
	}

	// Empty Origin passes vacuously (CAP-03 desktop path).
	if !ws.originAllowedForWrite(reqWithOrigin(""), sessionID) {
		t.Fatal("expected empty Origin to pass vacuously (CAP-03)")
	}

	// Funnel origin fails when inactive (fail-closed, T-165-07).
	if ws.originAllowedForWrite(reqWithOrigin(funnelURL), sessionID) {
		t.Fatal("expected Funnel origin to be rejected when Funnel inactive")
	}

	// Enable Funnel via real path (Pitfall 5 guard).
	if err := ws.EnableFunnel(context.Background(), 443); err != nil {
		t.Fatalf("EnableFunnel: %v", err)
	}

	// Funnel origin STILL fails after EnableFunnel when the session has not
	// passed the RW gate (T-171-05 defense-in-depth).
	if ws.originAllowedForWrite(reqWithOrigin(funnelURL), sessionID) {
		t.Fatal("expected Funnel origin to be rejected for a non-gated session even with Funnel active")
	}

	// Once gated, Funnel origin passes.
	ws.SetRWGate(sessionID, true)
	if !ws.originAllowedForWrite(reqWithOrigin(funnelURL), sessionID) {
		t.Fatal("expected Funnel origin to pass once the session is RW-gated")
	}

	// Tailnet BaseURL origin still passes, unaffected by gate state.
	if !ws.originAllowedForWrite(reqWithOrigin(ws.BaseURL()), sessionID) {
		t.Fatal("expected tailnet BaseURL origin to still pass")
	}

	// Unrelated origin still fails.
	if ws.originAllowedForWrite(reqWithOrigin("https://evil.example"), sessionID) {
		t.Fatal("expected unrelated origin to still fail")
	}
}
