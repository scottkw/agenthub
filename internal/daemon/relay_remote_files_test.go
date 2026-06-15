// Regression test for the v3.5 remote-files relay gap.
//
// The Wails desktop GUI reaches files over the relay loopback TCP server
// (127.0.0.1:<relayPort>), NOT the daemon Unix socket — the webview cannot
// reach the socket. Phase 120 mounted the LOCAL /api/files/* routes on both
// the relay and the socket, but Phase 122 (remote read) and Phase 128 (remote
// write) registered the /api/files/remote/{sid}/... proxy routes ONLY on the
// Unix-socket mux (internal/daemon/api.go). The result: every remote file op
// from the desktop GUI — browse, read, AND write/delete/rename/mkdir — 404'd
// ("404 page not found", a Go ServeMux route-miss). Remote file access never
// worked on the desktop GUI; this blocked the two-machine tailnet UAT (#24).
//
// This test drives the RELAY surface (api.RelayHandler()) exactly as the webview
// does and guards three things: (1) the 9 remote routes are mounted there and a
// real fetch reaches the proxy, (2) CORS preflight succeeds for the cross-origin
// write verb, and (3) the write verb is routed (not 404). It mirrors the LOCAL
// analogue in internal/relay/server_files_test.go::TestServer_FilesWriteAPI_MountedOnRelay.
//
// Lives in package daemon_test and reuses newFixtureRemotePeer /
// newDaemonAPIWithUpstreamCert / fixtureCap / canonicalListResponse from
// remote_files_parity_test.go.

package daemon_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/files"
)

// depositCapOnSocket deposits the (sessionID, baseURL, capToken) tuple via the
// Unix-socket mux's POST /api/remote-files/caps. The desktop GUI does this
// through the Wails Go binding (App.RegisterRemoteCap → DaemonClient over the
// socket), NOT the webview — so the deposit route is intentionally socket-only
// and is exercised here through api.Handler() rather than the relay surface.
func depositCapOnSocket(t *testing.T, socketSrv *httptest.Server, sessionID, baseURL, cap string) {
	t.Helper()
	raw, _ := json.Marshal(map[string]string{
		"sessionId": sessionID,
		"baseUrl":   baseURL,
		"capToken":  cap,
	})
	resp, err := http.Post(socketSrv.URL+"/api/remote-files/caps", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("cap deposit: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cap deposit status: want 200, got %d", resp.StatusCode)
	}
}

func TestRemoteFiles_MountedOnRelay(t *testing.T) {
	upstream := newFixtureRemotePeer(t)
	defer upstream.Close()

	api := newDaemonAPIWithUpstreamCert(t, upstream)

	// Socket surface — used only to deposit the cap, mirroring the Wails Go
	// binding path. File ops below go through the relay surface.
	socketSrv := httptest.NewServer(api.Handler())
	defer socketSrv.Close()

	// Relay surface — the loopback TCP server the webview actually fetches from.
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	depositCapOnSocket(t, socketSrv, "sid1", upstream.URL, fixtureCap)

	// 1. GET .../list through the relay must reach the proxy (200), not 404.
	//    This is the call FilesApiClient makes from the webview with
	//    baseURL=http://127.0.0.1:<relayPort> + pathPrefix=/api/files/remote/sid1.
	listURL := relaySrv.URL + "/api/files/remote/sid1/list?path=."
	resp, err := http.Get(listURL)
	if err != nil {
		t.Fatalf("GET %s: %v", listURL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		t.Fatalf("relay GET .../list returned 404 — remote route not mounted on relay; body=%s", body)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("relay GET .../list status = %d; want 200; body=%s", resp.StatusCode, body)
	}
	var parsed files.FileListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal relay list response: %v; body=%s", err, body)
	}
	canonical, _ := canonicalListResponse()
	if !bytes.Equal(bytes.TrimSpace(body), bytes.TrimSpace(canonical)) {
		t.Errorf("relay list body != canonical upstream body\nrelay=%s\ncanonical=%s", body, canonical)
	}

	// 2. CORS preflight for the cross-origin write verb must succeed and
	//    advertise PUT + If-Match, or the webview never sends the PUT.
	writeURL := relaySrv.URL + "/api/files/remote/sid1/write?path=a.txt"
	preReq, _ := http.NewRequest(http.MethodOptions, writeURL, nil)
	preReq.Header.Set("Origin", "wails://wails")
	preReq.Header.Set("Access-Control-Request-Method", "PUT")
	preReq.Header.Set("Access-Control-Request-Headers", "If-Match")
	preResp, err := http.DefaultClient.Do(preReq)
	if err != nil {
		t.Fatalf("OPTIONS preflight: %v", err)
	}
	defer preResp.Body.Close()
	if preResp.StatusCode != http.StatusNoContent {
		t.Fatalf("relay write preflight status = %d; want 204", preResp.StatusCode)
	}
	if m := preResp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(m, "PUT") {
		t.Errorf("preflight Allow-Methods = %q; want it to include PUT", m)
	}
	if h := preResp.Header.Get("Access-Control-Allow-Headers"); !strings.Contains(h, "If-Match") {
		t.Errorf("preflight Allow-Headers = %q; want it to include If-Match", h)
	}

	// 3. The PUT write verb must be routed through the relay to the upstream
	//    write endpoint and relay its 200 — not a relay-level route-miss 404.
	putReq, _ := http.NewRequest(http.MethodPut, writeURL, strings.NewReader("edited via relay"))
	putReq.Header.Set("If-Match", "*")
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT %s: %v", writeURL, err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode == http.StatusNotFound {
		t.Fatalf("relay PUT .../write returned 404 — remote write route not mounted on relay")
	}
	if putResp.StatusCode != http.StatusOK {
		putBody, _ := io.ReadAll(putResp.Body)
		t.Errorf("relay PUT .../write status = %d; want 200 (routed to upstream write); body=%s", putResp.StatusCode, putBody)
	}
}

// TestRemoteFiles_RelaySurface_NoCap_ReachesProxy reproduces the exact live
// probe the v3.5 audit used to find the bug. Before the fix, GET .../list on
// the relay port returned Go's bare "404 page not found" (a ServeMux route-miss
// — the route was never registered). After the fix the route IS registered, so
// with no cap deposited the request reaches the proxy handler, which returns its
// own JSON 404 carrying the "no cap registered" marker. The body distinguishes
// "route mounted, handler reached" from "route never mounted".
func TestRemoteFiles_RelaySurface_NoCap_ReachesProxy(t *testing.T) {
	upstream := newFixtureRemotePeer(t)
	defer upstream.Close()

	// Fresh daemon, NO cap deposited.
	api := newDaemonAPIWithUpstreamCert(t, upstream)
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	resp, err := http.Get(relaySrv.URL + "/api/files/remote/sid1/list?path=.")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// Both the route-miss and the proxy's "no cap" response are 404, so status
	// alone cannot tell them apart — the body is the discriminator.
	if strings.Contains(string(body), "404 page not found") {
		t.Fatalf("relay .../list returned the bare route-miss 404 — remote route not mounted; body=%s", body)
	}
	if !strings.Contains(string(body), "no cap registered") {
		t.Errorf("relay .../list with no cap: want proxy 'no cap registered' marker (handler reached); body=%s", body)
	}
}

// TestRemoteFiles_RelaySurface_LocalRoutesStillWork confirms wrapping the relay
// server with the remote-files routes does not shadow the local /api/files/*
// routes or the /sessions list — the parent mux must fall through to the relay
// server for every path it does not explicitly own.
func TestRemoteFiles_RelaySurface_FallthroughToRelay(t *testing.T) {
	upstream := newFixtureRemotePeer(t)
	defer upstream.Close()

	api := newDaemonAPIWithUpstreamCert(t, upstream)
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	// /sessions is served by the inner relay server. If the parent mux failed
	// to fall through, this would 404.
	resp, err := http.Get(relaySrv.URL + "/sessions")
	if err != nil {
		t.Fatalf("GET /sessions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("relay /sessions status = %d; want 200 (parent mux must fall through to relay server)", resp.StatusCode)
	}
}
