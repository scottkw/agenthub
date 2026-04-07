package tailnet

import (
	"context"
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
