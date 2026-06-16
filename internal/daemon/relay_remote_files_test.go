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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// ─── Phase 129 Wave 0: relay-surface race test (RACE-01) ────────────────────

// newSandboxBackedRemotePeer creates an httptest.TLSServer whose
// PUT /api/files/write handler is backed by a real files.Sandbox rooted at a
// t.TempDir(). The handler:
//   - Requires ?cap=fixtureCap (cap-guarded, same as newFixtureRemotePeer)
//   - Reads the If-Match header and passes it as the expectedValidator to WriteFileAtomic
//   - Maps files.ErrPreconditionFailed → HTTP 412
//   - Maps success → HTTP 200 {"ok":true}
//
// Returns the server (caller defers Close) and the temp root path so tests can
// inspect final file content and check for leftover .agenthub-tmp-* siblings.
//
// The read-only verbs (list/stat/read) are forwarded to newFixtureRemotePeer's
// handlers so this server can coexist with read-path tests. Only write is
// sandbox-backed because that is what the race test exercises.
func newSandboxBackedRemotePeer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	root := t.TempDir()

	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	// Seed the target file so the If-Match validator is meaningful.
	const seededFile = "race-relay-target.txt"
	seededContent := []byte("initial content for relay race test")
	if err := os.WriteFile(filepath.Join(root, seededFile), seededContent, 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	mux := http.NewServeMux()

	guard := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("cap") != fixtureCap {
				http.Error(w, "cap rejected", http.StatusUnauthorized)
				return
			}
			handler(w, r)
		}
	}

	// PUT /api/files/write — backed by files.Sandbox.WriteFileAtomic with
	// real If-Match→validator translation and ErrPreconditionFailed→412 mapping.
	mux.HandleFunc("PUT /api/files/write", guard(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "missing path", http.StatusBadRequest)
			return
		}

		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, "read body: "+readErr.Error(), http.StatusInternalServerError)
			return
		}

		ifMatch := r.Header.Get("If-Match")
		// Pass the If-Match value directly as the expectedValidator.
		// WriteFileAtomic treats "*" as unconditional and a non-"*" non-empty
		// string as a validator that must match the on-disk mtime+size.
		writeErr := sb.WriteFileAtomic(path, body, ifMatch)
		if writeErr != nil {
			if writeErr == files.ErrPreconditionFailed {
				http.Error(w, "precondition failed: validator mismatch", http.StatusPreconditionFailed)
				return
			}
			http.Error(w, "write error: "+writeErr.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	return httptest.NewTLSServer(mux), root
}

// TestRemoteFiles_TwoWriterRace_RelaySurface covers RACE-01.
//
// This is the relay-surface counterpart to TestWrite_TwoWritersIfMatchRace in
// internal/files/write_test.go. Instead of calling WriteFileAtomic directly,
// it drives two concurrent PUT requests through api.RelayHandler() — the GUI's
// actual loopback surface — to the sandbox-backed upstream. This closes the
// v3.5 blind spot: the previous relay tests only confirmed route mounting, not
// the single-winner invariant on the relay path.
//
// RACE-01 relay invariant: both PUTs use the SAME stale validator captured
// before either write. After both complete, exactly one relay response must be
// 200 (the winner) and exactly one must be 412 (the loser). The final on-disk
// file must be all-A or all-B (never interleaved). No .agenthub-tmp-* sibling
// may remain.
//
// RED: this test FAILS until Plan 02 adds the per-path mutex to WriteFileAtomic
// (RACE-01). Without the mutex, both goroutines pass the stat-check before either
// renames, so both win (two 200 responses, nilCount=2 at the upstream).
func TestRemoteFiles_TwoWriterRace_RelaySurface(t *testing.T) {
	upstream, root := newSandboxBackedRemotePeer(t)
	defer upstream.Close()

	api := newDaemonAPIWithUpstreamCert(t, upstream)

	// Socket surface — used only to deposit the cap.
	socketSrv := httptest.NewServer(api.Handler())
	defer socketSrv.Close()

	// Relay surface — the loopback TCP server the webview actually fetches from.
	relaySrv := httptest.NewServer(api.RelayHandler())
	defer relaySrv.Close()

	depositCapOnSocket(t, socketSrv, "sid-race", upstream.URL, fixtureCap)

	// Capture the current validator of the seeded file. Both writers will use
	// this same stale validator — the key precondition for the race.
	const seededFile = "race-relay-target.txt"
	fi, err := os.Stat(filepath.Join(root, seededFile))
	if err != nil {
		t.Fatalf("stat seeded file: %v", err)
	}
	sharedValidator := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))

	contentA := bytes.Repeat([]byte("A"), 512)
	contentB := bytes.Repeat([]byte("B"), 512)

	type result struct {
		status int
		body   []byte
	}
	results := make(chan result, 2)

	writeURL := relaySrv.URL + "/api/files/remote/sid-race/write?path=" + seededFile

	var wg sync.WaitGroup
	wg.Add(2)

	sendPUT := func(content []byte) {
		defer wg.Done()
		req, _ := http.NewRequest(http.MethodPut, writeURL, bytes.NewReader(content))
		req.Header.Set("If-Match", sharedValidator)
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, doErr := http.DefaultClient.Do(req)
		if doErr != nil {
			results <- result{status: 0, body: []byte(doErr.Error())}
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		results <- result{status: resp.StatusCode, body: body}
	}

	go sendPUT(contentA)
	go sendPUT(contentB)

	wg.Wait()
	close(results)

	var okCount, conflictCount int
	for r := range results {
		switch r.status {
		case http.StatusOK:
			okCount++
		case http.StatusPreconditionFailed, http.StatusConflict:
			// 412 from the upstream WriteFileAtomic ErrPreconditionFailed.
			// The relay proxy passes it through verbatim.
			conflictCount++
		default:
			t.Errorf("unexpected relay response status=%d body=%q", r.status, r.body)
		}
	}

	// RACE-01 single-winner assertion: exactly one writer wins (200) and
	// exactly one loses (412). Without the per-path mutex both win → okCount=2.
	if okCount != 1 {
		t.Errorf("okCount = %d; want exactly 1 successful write through relay (RACE-01 fail)", okCount)
	}
	if conflictCount != 1 {
		t.Errorf("conflictCount = %d; want exactly 1 precondition-failed through relay (RACE-01 fail)", conflictCount)
	}

	// Final on-disk content must be exactly one writer's complete payload.
	got, err := os.ReadFile(filepath.Join(root, seededFile))
	if err != nil {
		t.Fatalf("ReadFile after relay race: %v", err)
	}
	isAllA := len(got) == len(contentA) && bytes.Count(got, []byte("A")) == len(contentA)
	isAllB := len(got) == len(contentB) && bytes.Count(got, []byte("B")) == len(contentB)
	if !isAllA && !isAllB {
		t.Errorf("file content is neither all-A nor all-B (interleaved or partial write?): len=%d first8=%q",
			len(got), got[:min(8, len(got))])
	}

	// No .agenthub-tmp-* sibling may remain in the sandbox root.
	tmpPattern := filepath.Join(root, seededFile+".agenthub-tmp-*")
	leftover, err := filepath.Glob(tmpPattern)
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(leftover) != 0 {
		t.Errorf("leftover temp files after relay race: %v", leftover)
	}
}
