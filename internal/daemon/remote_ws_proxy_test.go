// Phase 134-06 — Cap-gated remote terminal WebSocket reverse-proxy tests.
//
// These integration tests drive the daemon RELAY surface (api.RelayHandler())
// exactly as the Wails webview does: the webview opens
// ws://127.0.0.1:<relayPort>/api/relay/remote/{sid}/ws WITHOUT a cap, and the
// daemon proxy looks up the Phase 122 cap, dials the peer's already-cap-gated
// wss://<baseURL>/sessions/{sid}/ws?cap=T (injecting the Origin the peer
// requires), and copies opaque WS frames both ways.
//
// They mirror the structure of relay_remote_files_test.go (fixture TLS peer +
// newDaemonAPIWithUpstreamCert + depositCapOnSocket) but the fixture peer here
// serves a cap-guarded WS echo endpoint at /sessions/{id}/ws that asserts the
// ?cap= query param AND a non-empty Origin equal to its own base URL.
//
// Lives in package daemon_test and reuses fixtureCap / newDaemonAPIWithUpstreamCert
// / depositCapOnSocket from the existing remote-files test files.

package daemon_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// fixtureWSPeer is a cap-guarded WebSocket echo peer that mimics the remote
// webserver's /sessions/{id}/ws contract for the proxy tests. It records the
// Origin header and cap query param it observed on the most recent upgrade so
// tests can assert the proxy injected them correctly.
type fixtureWSPeer struct {
	srv *httptest.Server

	mu         sync.Mutex
	gotOrigin  string
	gotCap     string
	sawUpgrade bool
}

// newFixtureRemotePeerWithWS spins up an httptest.NewTLSServer that serves a
// cap-guarded WS echo at GET /sessions/{id}/ws. The handler:
//   - rejects an empty Origin with 403 (mirrors the peer's requireAllowedOrigin)
//   - rejects a missing/mismatched ?cap= with 401 (mirrors requireCapability)
//   - otherwise accepts the upgrade (no inbound Origin allowlist — the dialer is
//     the daemon, which injects the peer's own base URL) and echoes every frame
//     back verbatim until the client closes.
func newFixtureRemotePeerWithWS(t *testing.T) *fixtureWSPeer {
	t.Helper()

	peer := &fixtureWSPeer{}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /sessions/{id}/ws", func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		cap := r.URL.Query().Get("cap")

		peer.mu.Lock()
		peer.gotOrigin = origin
		peer.gotCap = cap
		peer.sawUpgrade = true
		peer.mu.Unlock()

		// Mirror the peer's requireAllowedOrigin: empty Origin is rejected.
		if origin == "" {
			http.Error(w, "empty origin forbidden", http.StatusForbidden)
			return
		}
		// Mirror requireCapability: the cap arrives as the ?cap= query param.
		if cap != fixtureCap {
			http.Error(w, "cap rejected", http.StatusUnauthorized)
			return
		}

		// Accept with no OriginPatterns restriction — coder/websocket requires
		// the Origin host to match the request host unless we opt out. The proxy
		// injects Origin == baseURL, which is this server's own host, so the
		// default same-host check would already pass; "*" is belt-and-suspenders.
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			return
		}
		defer conn.CloseNow()

		ctx := r.Context()
		for {
			typ, data, rerr := conn.Read(ctx)
			if rerr != nil {
				return
			}
			if werr := conn.Write(ctx, typ, data); werr != nil {
				return
			}
		}
	})

	peer.srv = httptest.NewTLSServer(mux)
	t.Cleanup(peer.srv.Close)
	return peer
}

func (p *fixtureWSPeer) observed() (origin, cap string, sawUpgrade bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gotOrigin, p.gotCap, p.sawUpgrade
}

// dialProxy opens the inbound (webview-side) WS to the daemon relay proxy at
// ws://<relayHost>/api/relay/remote/{sid}/ws with the given Origin header.
func dialProxy(ctx context.Context, t *testing.T, relayURL, sid, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	wsURL := "ws://" + strings.TrimPrefix(relayURL, "http://") + "/api/relay/remote/" + sid + "/ws"
	hdr := http.Header{}
	if origin != "" {
		hdr.Set("Origin", origin)
	}
	return websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: hdr})
}

// TestRemoteSessionWS_MountedOnRelay (WS-PROXY-01): the route is mounted on the
// relay surface and the handler runs (a successful upgrade proves it, since a
// route-miss would 404 the upgrade).
func TestRemoteSessionWS_MountedOnRelay(t *testing.T) {
	peer := newFixtureRemotePeerWithWS(t)
	api := newDaemonAPIWithUpstreamCert(t, peer.srv)

	socketSrv := httptest.NewServer(api.Handler())
	defer socketSrv.Close()
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	depositCapOnSocket(t, socketSrv, "sid1", peer.srv.URL, fixtureCap)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := dialProxy(ctx, t, relaySrv.URL, "sid1", "wails://wails")
	if err != nil {
		t.Fatalf("dial proxy: %v (route not mounted on relay surface?)", err)
	}
	defer conn.CloseNow()

	// Round-trip one frame to guarantee the proxy completed the upstream upgrade
	// before we inspect what the peer observed (the upstream Accept runs in a
	// separate goroutine on the peer side).
	if err := conn.Write(ctx, websocket.MessageBinary, []byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("read: %v", err)
	}

	if _, _, saw := peer.observed(); !saw {
		t.Fatalf("upstream peer never saw the upgrade — proxy handler did not dial through")
	}
}

// TestRemoteSessionWS_NoCap (WS-PROXY-02): with no cap deposited the handler is
// still reached and returns the "no cap registered" 404 contract (the same as
// proxyRemoteFiles), distinguishable from a bare route-miss 404.
func TestRemoteSessionWS_NoCap(t *testing.T) {
	peer := newFixtureRemotePeerWithWS(t)
	api := newDaemonAPIWithUpstreamCert(t, peer.srv) // no cap deposited
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	// Use a plain HTTP GET (no upgrade) so we can read the JSON 404 body — the
	// handler writes the no-cap response before any Accept.
	resp, err := http.Get(relaySrv.URL + "/api/relay/remote/sid1/ws")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	got := string(body[:n])

	if strings.Contains(got, "404 page not found") {
		t.Fatalf("relay returned the bare route-miss 404 — WS route not mounted; body=%s", got)
	}
	if !strings.Contains(got, "no cap registered") {
		t.Errorf("no-cap request: want proxy 'no cap registered' marker (handler reached); body=%s", got)
	}
}

// TestRemoteSessionWS_FrameCopy (WS-PROXY-03): with a cap, frames written by the
// webview-side conn arrive at the peer and the peer's echo frames arrive back.
func TestRemoteSessionWS_FrameCopy(t *testing.T) {
	peer := newFixtureRemotePeerWithWS(t)
	api := newDaemonAPIWithUpstreamCert(t, peer.srv)

	socketSrv := httptest.NewServer(api.Handler())
	defer socketSrv.Close()
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	depositCapOnSocket(t, socketSrv, "sid1", peer.srv.URL, fixtureCap)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := dialProxy(ctx, t, relaySrv.URL, "sid1", "wails://wails")
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.CloseNow()

	payload := []byte("hello-remote-terminal")
	if err := conn.Write(ctx, websocket.MessageBinary, payload); err != nil {
		t.Fatalf("write to proxy: %v", err)
	}
	typ, got, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read echo from proxy: %v", err)
	}
	if typ != websocket.MessageBinary {
		t.Errorf("echo message type = %v; want binary", typ)
	}
	if string(got) != string(payload) {
		t.Errorf("echo payload = %q; want %q (bidirectional copy broken)", got, payload)
	}
}

// TestRemoteSessionWS_InjectsOrigin (WS-PROXY-04): the peer asserts it received
// a non-empty Origin equal to its base URL on the upstream dial, plus the cap.
func TestRemoteSessionWS_InjectsOrigin(t *testing.T) {
	peer := newFixtureRemotePeerWithWS(t)
	api := newDaemonAPIWithUpstreamCert(t, peer.srv)

	socketSrv := httptest.NewServer(api.Handler())
	defer socketSrv.Close()
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	depositCapOnSocket(t, socketSrv, "sid1", peer.srv.URL, fixtureCap)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := dialProxy(ctx, t, relaySrv.URL, "sid1", "wails://wails")
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.CloseNow()

	// Round-trip one frame to guarantee the upstream upgrade completed before we
	// inspect what the peer observed.
	if err := conn.Write(ctx, websocket.MessageBinary, []byte("x")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("read: %v", err)
	}

	origin, cap, saw := peer.observed()
	if !saw {
		t.Fatalf("peer never saw the upgrade")
	}
	wantOrigin := strings.TrimRight(peer.srv.URL, "/")
	if origin == "" {
		t.Errorf("peer saw EMPTY Origin — proxy must inject Origin: <baseURL> (Pitfall 1)")
	}
	if origin != wantOrigin {
		t.Errorf("peer Origin = %q; want %q (byte-exact baseURL)", origin, wantOrigin)
	}
	if cap != fixtureCap {
		t.Errorf("peer cap = %q; want %q forwarded as ?cap=", cap, fixtureCap)
	}
}

// TestRemoteSessionWS_RejectsCrossSiteOrigin (WS-PROXY-05): a cross-site inbound
// Origin is rejected at websocket.Accept (mirror relay origin_test.go). Use raw
// HTTP so the non-101 status is observable (websocket.Dial hides it).
func TestRemoteSessionWS_RejectsCrossSiteOrigin(t *testing.T) {
	peer := newFixtureRemotePeerWithWS(t)
	api := newDaemonAPIWithUpstreamCert(t, peer.srv)

	socketSrv := httptest.NewServer(api.Handler())
	defer socketSrv.Close()
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	depositCapOnSocket(t, socketSrv, "sid1", peer.srv.URL, fixtureCap)

	req, err := http.NewRequest("GET", relaySrv.URL+"/api/relay/remote/sid1/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusSwitchingProtocols {
		t.Error("proxy accepted cross-site Origin — inbound loopback allowlist not enforced (T-134-06-01)")
	}
}

// TestRemoteSessionWS_LongLived (WS-PROXY-06): the copy loop uses r.Context(),
// not the 10s dial-timeout context, so a healthy terminal survives past 10s.
// Rather than sleeping 10+ real seconds, this test asserts the connection is
// still live and echoing after a delay comfortably exceeding any dial deadline
// derivation that would have leaked into the copy loop.
func TestRemoteSessionWS_LongLived(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long-lived WS test in -short mode")
	}
	peer := newFixtureRemotePeerWithWS(t)
	api := newDaemonAPIWithUpstreamCert(t, peer.srv)

	socketSrv := httptest.NewServer(api.Handler())
	defer socketSrv.Close()
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	depositCapOnSocket(t, socketSrv, "sid1", peer.srv.URL, fixtureCap)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	conn, _, err := dialProxy(ctx, t, relaySrv.URL, "sid1", "wails://wails")
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.CloseNow()

	// Prime the connection.
	if err := conn.Write(ctx, websocket.MessageBinary, []byte("first")); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("read first: %v", err)
	}

	// Wait past the 10s dial deadline. If the copy loop wrongly used the dial
	// context, the upstream side would have torn down at ~10s and this second
	// round-trip would fail.
	time.Sleep(11 * time.Second)

	if err := conn.Write(ctx, websocket.MessageBinary, []byte("after-deadline")); err != nil {
		t.Fatalf("write after 11s: %v (copy loop used the dial deadline, not r.Context()?)", err)
	}
	typ, got, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read after 11s: %v (terminal died at the 10s dial deadline — Pitfall 4)", err)
	}
	if typ != websocket.MessageBinary || string(got) != "after-deadline" {
		t.Errorf("post-deadline echo = (%v, %q); want (binary, %q)", typ, got, "after-deadline")
	}
}
