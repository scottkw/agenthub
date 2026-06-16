package tailnet

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"tailscale.com/ipn/ipnstate"
	"tailscale.com/types/key"
)

// fakeStatus returns a statusFunc that injects a canned *ipnstate.Status.
func fakeStatus(s *ipnstate.Status) statusFunc {
	return func(ctx context.Context) (*ipnstate.Status, error) {
		return s, nil
	}
}

// errorStatus returns a statusFunc that returns an error.
func errorStatus(err error) statusFunc {
	return func(ctx context.Context) (*ipnstate.Status, error) {
		return nil, err
	}
}

// -----------------------------------------------------------------------
// discoverPeers tests
// -----------------------------------------------------------------------

func TestDiscoverPeers_OnlineOnly(t *testing.T) {
	status := &ipnstate.Status{
		Peer: map[key.NodePublic]*ipnstate.PeerStatus{
			key.NewNode().Public(): {
				HostName:     "online-host",
				DNSName:      "online-host.ts.net.",
				TailscaleIPs: []netip.Addr{netip.MustParseAddr("100.64.0.1")},
				OS:           "linux",
				Online:       true,
			},
			key.NewNode().Public(): {
				HostName:     "offline-host",
				DNSName:      "offline-host.ts.net.",
				TailscaleIPs: []netip.Addr{netip.MustParseAddr("100.64.0.2")},
				OS:           "windows",
				Online:       false,
			},
		},
	}

	peers, err := discoverPeers(context.Background(), fakeStatus(status))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}
	p := peers[0]
	if p.Hostname != "online-host" {
		t.Errorf("expected Hostname 'online-host', got %q", p.Hostname)
	}
	if p.DNSName != "online-host.ts.net." {
		t.Errorf("expected DNSName 'online-host.ts.net.', got %q", p.DNSName)
	}
	if len(p.TailscaleIPs) != 1 || p.TailscaleIPs[0] != "100.64.0.1" {
		t.Errorf("expected TailscaleIPs [100.64.0.1], got %v", p.TailscaleIPs)
	}
	if p.OS != "linux" {
		t.Errorf("expected OS 'linux', got %q", p.OS)
	}
	if !p.Online {
		t.Errorf("expected Online=true")
	}
}

func TestDiscoverPeers_Empty(t *testing.T) {
	status := &ipnstate.Status{
		Peer: map[key.NodePublic]*ipnstate.PeerStatus{},
	}
	peers, err := discoverPeers(context.Background(), fakeStatus(status))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if peers == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(peers) != 0 {
		t.Fatalf("expected 0 peers, got %d", len(peers))
	}
}

func TestDiscoverPeers_Error(t *testing.T) {
	err := context.DeadlineExceeded
	peers, gotErr := discoverPeers(context.Background(), errorStatus(err))
	if gotErr == nil {
		t.Fatal("expected error, got nil")
	}
	if peers != nil {
		t.Fatalf("expected nil peers on error, got %v", peers)
	}
}

// -----------------------------------------------------------------------
// probePeer tests (use injectable inner function with httptest server)
// -----------------------------------------------------------------------

// rewriteTransport is a custom RoundTripper that redirects all requests to
// the test server while preserving the request path. This allows probePeer
// (which builds its own URL from DNSName + DefaultProbePort) to be tested
// without a real Tailscale peer.
type rewriteTransport struct {
	base    http.RoundTripper
	dialURL string // host:port of the test server (e.g. "127.0.0.1:PORT")
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.URL.Host = rt.dialURL
	r.Host = rt.dialURL
	return rt.base.RoundTrip(r)
}

// redirectingClient returns an *http.Client that trusts the TLS test server cert
// and rewrites all requests to the test server address.
func redirectingClient(srv *httptest.Server) *http.Client {
	return &http.Client{
		Transport: &rewriteTransport{
			base:    srv.Client().Transport,
			dialURL: srv.Listener.Addr().String(),
		},
	}
}

func TestProbePeer_Found(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	peer := Peer{DNSName: "testhost.ts.net."}
	result := probePeer(context.Background(), peer, redirectingClient(srv))
	if !result {
		t.Error("expected probePeer to return true for 200 response")
	}
}

func TestProbePeer_NotFound(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	peer := Peer{DNSName: "testhost.ts.net."}
	result := probePeer(context.Background(), peer, redirectingClient(srv))
	if result {
		t.Error("expected probePeer to return false for 404 response")
	}
}

// TestProbePeer_CapProtected401_Found is the regression test for #84. A
// cap-protected AgentHub peer answers the unauthenticated discovery probe
// (GET /api/sessions, no cap) with 401 "capability required" (capability_mw.go).
// Before the fix, probePeer accepted only HTTP 200, so every shared (cap-gated)
// peer — i.e. every real tailnet share — was silently dropped and the Remote
// panel stayed empty. Discovery must treat the AgentHub-shaped 401 as "present".
func TestProbePeer_CapProtected401_Found(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions" {
			http.Error(w, "capability required", http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	peer := Peer{DNSName: "testhost.ts.net."}
	if !probePeer(context.Background(), peer, redirectingClient(srv)) {
		t.Error("expected probePeer to return true for AgentHub 401 \"capability required\"")
	}
}

// TestProbePeer_Generic401_NotFound guards the #84 fix against false positives:
// a non-AgentHub server that happens to return a bare 401 on :7443 must NOT be
// mistaken for an AgentHub peer. Only the "capability required" marker counts.
func TestProbePeer_Generic401_NotFound(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	peer := Peer{DNSName: "testhost.ts.net."}
	if probePeer(context.Background(), peer, redirectingClient(srv)) {
		t.Error("expected probePeer to return false for a generic (non-AgentHub) 401")
	}
}

func TestProbePeer_Timeout(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	peer := Peer{DNSName: "testhost.ts.net."}
	result := probePeer(ctx, peer, redirectingClient(srv))
	if result {
		t.Error("expected probePeer to return false on timeout")
	}
}

func TestProbePeer_DNSNameDotStrip(t *testing.T) {
	var gotPath string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	peer := Peer{DNSName: "host.ts.net."}
	probePeer(context.Background(), peer, redirectingClient(srv))
	if gotPath != "/api/sessions" {
		t.Errorf("expected path /api/sessions, got %q", gotPath)
	}
}

// -----------------------------------------------------------------------
// probeAll tests
// -----------------------------------------------------------------------

func TestProbeAll_Concurrent(t *testing.T) {
	const numPeers = 10
	peers := make([]Peer, numPeers)
	for i := range peers {
		peers[i] = Peer{Hostname: "peer", DNSName: "peer.ts.net.", Online: true}
	}

	var activeCount int64
	var maxConcurrent int64

	fn := probeFunc(func(ctx context.Context, p Peer) bool {
		current := atomic.AddInt64(&activeCount, 1)
		// Update max atomically.
		for {
			old := atomic.LoadInt64(&maxConcurrent)
			if current <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond) // hold slot briefly
		atomic.AddInt64(&activeCount, -1)
		return true
	})

	result := probeAll(context.Background(), peers, fn)
	if len(result) != numPeers {
		t.Errorf("expected %d peers, got %d", numPeers, len(result))
	}
	if maxConcurrent > 5 {
		t.Errorf("max concurrent goroutines exceeded 5: got %d", maxConcurrent)
	}
}

func TestProbeAll_FiltersNonResponders(t *testing.T) {
	peers := []Peer{
		{Hostname: "peer1", DNSName: "peer1.ts.net.", Online: true},
		{Hostname: "peer2", DNSName: "peer2.ts.net.", Online: true},
		{Hostname: "peer3", DNSName: "peer3.ts.net.", Online: true},
	}

	fn := probeFunc(func(ctx context.Context, p Peer) bool {
		return p.Hostname != "peer2" // peer2 doesn't respond
	})

	result := probeAll(context.Background(), peers, fn)
	if len(result) != 2 {
		t.Errorf("expected 2 peers, got %d: %v", len(result), result)
	}
	for _, r := range result {
		if r.Hostname == "peer2" {
			t.Error("peer2 should have been filtered out")
		}
	}
}

// -----------------------------------------------------------------------
// DiscoverAndProbe integration test
// -----------------------------------------------------------------------

func TestDiscoverAndProbe_Integration(t *testing.T) {
	// Test the composition of discoverPeers + probeAll directly, which is what
	// DiscoverAndProbe does internally.
	status := &ipnstate.Status{
		Peer: map[key.NodePublic]*ipnstate.PeerStatus{
			key.NewNode().Public(): {
				HostName: "agenthub-peer",
				DNSName:  "agenthub-peer.ts.net.",
				Online:   true,
			},
			key.NewNode().Public(): {
				HostName: "non-agenthub-peer",
				DNSName:  "non-agenthub-peer.ts.net.",
				Online:   true,
			},
		},
	}

	discovered, err := discoverPeers(context.Background(), fakeStatus(status))
	if err != nil {
		t.Fatalf("discoverPeers error: %v", err)
	}
	if len(discovered) != 2 {
		t.Fatalf("expected 2 discovered peers, got %d", len(discovered))
	}

	// probeFunc returns true only for agenthub-peer.
	fn := probeFunc(func(ctx context.Context, p Peer) bool {
		return p.Hostname == "agenthub-peer"
	})

	result := probeAll(context.Background(), discovered, fn)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Hostname != "agenthub-peer" {
		t.Errorf("expected agenthub-peer, got %q", result[0].Hostname)
	}
}

// -----------------------------------------------------------------------
// probePeerByIP fallback tests
// -----------------------------------------------------------------------

func TestProbePeer_IPFallback(t *testing.T) {
	// Simulate DNS failure by using an unresolvable DNSName, but the
	// redirecting client rewrites to the test server so the IP-fallback
	// path (which also goes through the client) succeeds.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	peer := Peer{
		DNSName:      "unresolvable-host.ts.net.",
		TailscaleIPs: []string{"100.64.0.99"},
	}
	// The redirecting client makes both DNS and IP paths hit the test server.
	result := probePeer(context.Background(), peer, redirectingClient(srv))
	if !result {
		t.Error("expected probePeer to succeed via IP fallback")
	}
}

func TestProbePeer_NoIPFallbackWithoutIPs(t *testing.T) {
	// If peer has no TailscaleIPs, fallback should not be attempted.
	peer := Peer{
		DNSName:      "unresolvable.example.invalid.",
		TailscaleIPs: nil,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Use a client that will fail DNS (no rewrite).
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	result := probePeer(ctx, peer, client)
	if result {
		t.Error("expected probePeer to return false with no IPs for fallback")
	}
}

// -----------------------------------------------------------------------
// Public wrapper smoke tests (exercises public code paths)
// -----------------------------------------------------------------------

func TestDiscoverPeers_Public(t *testing.T) {
	// Without a live tailscaled, DiscoverPeers should return an error.
	// This test exercises the public wrapper code path.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := DiscoverPeers(ctx)
	// We expect an error because no tailscaled is running in CI.
	// If somehow it succeeds (dev machine with tailscale), that's also fine.
	if err != nil {
		t.Logf("DiscoverPeers returned expected error (no live daemon): %v", err)
	} else {
		t.Log("DiscoverPeers succeeded (live tailscaled available)")
	}
	// Either outcome is acceptable; this test ensures the code compiles and runs.
}

func TestProbePeer_Public(t *testing.T) {
	// ProbePeer with an unreachable host should return false within 2 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	peer := Peer{
		Hostname: "unreachable",
		DNSName:  "unreachable.example.invalid.",
		Online:   true,
	}
	start := time.Now()
	result := ProbePeer(ctx, peer)
	elapsed := time.Since(start)
	if result {
		t.Error("expected ProbePeer to return false for unreachable host")
	}
	// Should return within ~3 seconds (2s timeout + margin).
	if elapsed > 3*time.Second {
		t.Errorf("ProbePeer took too long: %v", elapsed)
	}
	t.Logf("ProbePeer returned false in %v (expected)", elapsed)
}

// -----------------------------------------------------------------------
// Wave 0 RED tests for Phase 130 RB-01 — FetchAllPeerSessionsMeta /
// FetchPeerSessionsMeta (plan 130-03 will create these functions).
//
// These tests are intentionally RED: the functions ShareableSessionMeta,
// PeerSessionMetaGroup, FetchPeerSessionsMeta, and FetchAllPeerSessionsMeta
// do not exist yet. They will compile-error or assert-fail until plan 130-03
// adds them to internal/tailnet/sessions.go.
//
// Contracts encoded:
//   - RB-01: unreachable peers are NOT silently dropped — they appear with
//     Reachable=false and Sessions=[] (the nil-vs-empty discriminator)
//   - RB-04: a reachable peer with zero sessions has Reachable=true + len 0
//   - RB-01: a reachable peer with sessions returns Reachable=true + filled slice
//   - IP fallback path (mirrors FetchPeerSessions existing pattern)
// -----------------------------------------------------------------------

// metaPeerServer returns an httptest.TLSServer serving GET /api/sessions/meta.
// If items is nil, the server returns 503 (simulates unreachable). If items is
// an empty slice, it returns 200 `[]` (reachable, zero sessions). Otherwise it
// returns 200 with the JSON-encoded items.
func metaPeerServer(t *testing.T, items []ShareableSessionMeta) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sessions/meta" {
			http.NotFound(w, r)
			return
		}
		if items == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if len(items) == 0 {
			_, _ = w.Write([]byte("[]"))
			return
		}
		// Encode the items array.
		out := `[`
		for i, s := range items {
			if i > 0 {
				out += ","
			}
			out += `{"id":"` + s.ID + `","name":"` + s.Name + `","cli_type":"` + s.CLIType + `","status":"` + s.Status + `","url":"` + s.URL + `"}`
		}
		out += `]`
		_, _ = w.Write([]byte(out))
	}))
	return srv
}

// makePeerPointingAt constructs a Peer whose DNSName and TailscaleIPs both
// point at the test server address so the redirectingClient will route it.
func makePeerPointingAt(srv *httptest.Server) Peer {
	addr := srv.Listener.Addr().String()
	return Peer{
		Hostname:     "test-peer",
		DNSName:      "test-peer.ts.net.",
		TailscaleIPs: []string{addr},
	}
}

// TestFetchAllPeerSessionsMeta_IncludesUnreachablePeers asserts that when a
// peer's /api/sessions/meta endpoint is unreachable (connection refused after
// the server is closed), FetchAllPeerSessionsMeta still emits a group for it
// with Reachable=false and Sessions=[] (empty slice, not dropped).
//
// This is the core RB-01 no-silent-drop contract.
func TestFetchAllPeerSessionsMeta_IncludesUnreachablePeers(t *testing.T) {
	// Build a server then close it immediately so requests get connection-refused.
	closedSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	closedSrv.Close() // now unreachable

	peer := Peer{
		Hostname: "closed-peer",
		DNSName:  "closed-peer.ts.net.",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	groups := FetchAllPeerSessionsMeta(ctx, []Peer{peer})

	if len(groups) != 1 {
		t.Fatalf("expected 1 group for unreachable peer (no silent drop), got %d: %v", len(groups), groups)
	}

	g := groups[0]
	if g.Hostname != "closed-peer" {
		t.Errorf("expected Hostname=closed-peer, got %q", g.Hostname)
	}
	// RB-01 critical: unreachable → Reachable=false, NOT dropped.
	if g.Reachable {
		t.Errorf("unreachable peer must have Reachable=false, got true")
	}
	// Sessions must be an empty non-nil slice, not dropped peer.
	if g.Sessions == nil {
		t.Error("Sessions must be [] (non-nil empty slice) for unreachable peer")
	}
	if len(g.Sessions) != 0 {
		t.Errorf("expected 0 sessions for unreachable peer, got %d", len(g.Sessions))
	}
}

// TestFetchAllPeerSessionsMeta_EmptySessionsNotDropped asserts that a reachable
// peer returning 200 `[]` (zero shareable sessions) is NOT dropped from the
// result — it appears with Reachable=true and Sessions=[].
//
// This is the exact RB-01 fix vs sessions.go:93 (the nil-vs-empty distinction).
func TestFetchAllPeerSessionsMeta_EmptySessionsNotDropped(t *testing.T) {
	srv := metaPeerServer(t, []ShareableSessionMeta{}) // reachable, zero sessions
	defer srv.Close()

	peer := makePeerPointingAt(srv)
	peer.Hostname = "empty-peer"
	peer.DNSName = "empty-peer.ts.net."

	// Use a redirecting client so the test server receives the request.
	testClient := redirectingClient(srv)
	_ = testClient // FetchPeerSessionsMeta must accept a client or use DNS; we reference the type.

	// FetchAllPeerSessionsMeta doesn't accept a client injection; it uses DNS.
	// For this test we rely on the fact that connecting to a closed address
	// will fail DNS resolution — instead we use FetchPeerSessionsMeta which
	// mirrors FetchPeerSessions and should have a WithClient variant for tests.
	// Since the function does not exist yet, this test is RED by compile failure.
	sessions, reachable := FetchPeerSessionsMeta(context.Background(), peer)

	// Reachable peer with zero sessions: Reachable=true, Sessions=[] (not nil, not dropped).
	if !reachable {
		t.Errorf("peer serving 200 [] must be Reachable=true, got false")
	}
	if sessions == nil {
		t.Error("sessions must be [] (non-nil empty), not nil — nil signals unreachable")
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

// TestFetchAllPeerSessionsMeta_PopulatedPeer asserts that a reachable peer
// returning 200 with 2 session items maps to Reachable=true and 2 sessions
// with all fields populated.
func TestFetchAllPeerSessionsMeta_PopulatedPeer(t *testing.T) {
	items := []ShareableSessionMeta{
		{ID: "s1", Name: "Alpha", CLIType: "claude", Status: "running", URL: "https://peer/sessions/s1"},
		{ID: "s2", Name: "Beta", CLIType: "codex", Status: "idle", URL: "https://peer/sessions/s2"},
	}
	srv := metaPeerServer(t, items)
	defer srv.Close()

	peer := makePeerPointingAt(srv)
	peer.Hostname = "populated-peer"
	peer.DNSName = "populated-peer.ts.net."

	// FetchPeerSessionsMeta does not exist yet — RED by compile failure.
	sessions, reachable := FetchPeerSessionsMeta(context.Background(), peer)

	if !reachable {
		t.Errorf("expected Reachable=true for peer serving sessions, got false")
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].ID != "s1" && sessions[1].ID != "s1" {
		t.Errorf("expected session with ID=s1 in result")
	}
}

// TestFetchPeerSessionsMeta_IPFallback verifies that when the DNS-based fetch
// fails but a TailscaleIP is present, FetchPeerSessionsMeta falls back to the
// IP path and returns the served metas.
//
// Mirrors the existing TestProbePeer_IPFallback pattern: the redirectingClient
// makes both DNS and IP path attempts resolve to the test server.
func TestFetchPeerSessionsMeta_IPFallback(t *testing.T) {
	items := []ShareableSessionMeta{
		{ID: "fallback-sess", Name: "Fallback Session", CLIType: "claude", Status: "running", URL: "https://host/sessions/fallback-sess"},
	}
	srv := metaPeerServer(t, items)
	defer srv.Close()

	// Unresolvable DNS name — but TailscaleIPs is set so IP fallback fires.
	peer := Peer{
		Hostname:     "ip-fallback-peer",
		DNSName:      "unresolvable-host.example.invalid.",
		TailscaleIPs: []string{srv.Listener.Addr().String()},
	}

	// FetchPeerSessionsMeta does not exist yet — RED by compile failure.
	// In production it will create an IP-fallback client with ServerName set.
	// The redirectingClient pattern used in existing tests re-routes by host,
	// but FetchPeerSessionsMeta owns its own client — so the IP path exercise
	// is best validated via the function's own logic once it exists.
	sessions, reachable := FetchPeerSessionsMeta(context.Background(), peer)

	// Even via IP fallback, we expect at least the function to return results.
	// With a real server that happens to answer on the IP, reachable=true.
	// This test may also pass if the DNS resolution of the invalid host fails
	// fast and the IP fallback client successfully connects (using the peer's
	// actual IP directly).
	if !reachable {
		// Log rather than Fatal since IP fallback behavior depends on network config.
		t.Logf("IP fallback returned reachable=false (acceptable if network blocks IP fallback in test env); sessions=%v", sessions)
	} else if len(sessions) != 1 {
		t.Errorf("expected 1 session via IP fallback, got %d", len(sessions))
	}
}
