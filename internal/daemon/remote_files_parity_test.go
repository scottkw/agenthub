// Phase 122-05 — Cross-surface parity test.
//
// This is the merge-gate evidence for REMOTE-05: the desktop GUI (via the
// daemon's /api/files/remote/{sid}/... proxy) and the TUI (via the
// RemoteFilesClient HTTPS+cap path) MUST observe byte-identical responses
// when both query the same upstream "remote peer" webserver.
//
// The test lives in `package daemon_test` rather than `package daemon` so it
// can import `internal/tui` to exercise the TUI's RemoteFilesClient.
// Putting it in `package daemon` would be a cycle (tui imports daemon at the
// package level, see internal/tui/files.go).
//
// To drive the proxy through the daemon's mux from an external test package
// we use:
//   - daemon.NewAPI(engine) — already exported
//   - api.Handler() — newly exported test helper returning the mux
//   - api.SetRemoteFilesClientForTest(client) — newly exported test setter so
//     the daemon's outbound HTTPS transport trusts the upstream's self-signed
//     cert (this is just an exported wrapper around the unexported field that
//     internal daemon tests already use).
//
// Test inputs / expected behaviour follow PLAN.md Task 1's `<behavior>` block:
//   - Build a fixture upstream serving canned responses for list/stat/read.
//   - Bring up a httptest.NewServer wrapping the daemon mux.
//   - Deposit (sessionID, baseURL, capToken) via POST /api/remote-files/caps.
//   - Construct tui.NewRemoteFilesClientForTest pointed at the same upstream,
//     using the upstream's self-signed cert via the upstream's Client().
//   - Fetch list/stat/read through BOTH surfaces. Assert response bodies are
//     byte-identical.
//   - Assert cap-absence asymmetry: drop the proxy's cap (fresh daemon, no
//     deposit) → proxy returns 404 with the "no cap registered" marker; the
//     direct RemoteFilesClient still works because it carries the cap.
//   - Assert no test error string contains the cap token (defense-in-depth
//     CAP-LEAK invariant, T-122-04-01 + T-122-01).

package daemon_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/files"
	"github.com/scottkw/agenthub/internal/tui"
)

// fixtureCap is the canonical cap token used throughout the parity test.
// Both the upstream's auth gate AND the cap-leak source-grep assertion use
// this literal value.
const fixtureCap = "FIXTURE_CAP"

// canonicalListResponse is the byte-identical JSON body the upstream returns
// from /api/files/list. We encode/decode through files.FileListResponse so
// the test fails if any field renames or shape drift occur.
func canonicalListResponse() ([]byte, files.FileListResponse) {
	resp := files.FileListResponse{
		Entries: []files.FileEntry{
			{Name: "a.txt", Size: 100, IsDir: false},
			{Name: "sub", Size: 0, IsDir: true},
		},
		Truncated: false,
	}
	body, err := json.Marshal(resp)
	if err != nil {
		panic(err) // unreachable for static data
	}
	return body, resp
}

func canonicalStatResponse() ([]byte, files.FileEntry) {
	entry := files.FileEntry{Name: "a.txt", Size: 100, IsDir: false}
	body, err := json.Marshal(entry)
	if err != nil {
		panic(err)
	}
	return body, entry
}

// newFixtureRemotePeer spins up an httptest.NewTLSServer that mimics the
// remote peer's /api/files endpoints AND /join/exchange. Returns the server
// (caller defers Close).
func newFixtureRemotePeer(t *testing.T) *httptest.Server {
	t.Helper()

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

	listBody, _ := canonicalListResponse()
	statBody, _ := canonicalStatResponse()

	mux.HandleFunc("GET /api/files/list", guard(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(listBody)
	}))
	mux.HandleFunc("GET /api/files/stat", guard(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(statBody)
	}))
	mux.HandleFunc("GET /api/files/read", guard(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write([]byte("hello world"))
	}))

	// /join/exchange follows the webserver's 303 + Location shape (Phase 87)
	// — included so this fixture can be reused in cross-process scenarios.
	// Not exercised by Task 1's body but documented for completeness.
	mux.HandleFunc("POST /join/exchange", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/sessions/sid1?cap="+fixtureCap)
		w.WriteHeader(http.StatusSeeOther)
	})

	return httptest.NewTLSServer(mux)
}

// newDaemonAPIWithUpstreamCert constructs a daemon API with a tempdir configDir
// and injects the upstream's self-signed cert via SetRemoteFilesClientForTest
// so the proxy's outbound HTTPS calls succeed.
func newDaemonAPIWithUpstreamCert(t *testing.T, upstream *httptest.Server) *daemon.API {
	t.Helper()
	engine := daemon.NewSessionEngine()
	engine.ConfigDirForTest(t.TempDir())
	api := daemon.NewAPI(engine)
	api.SetRemoteFilesClientForTest(upstream.Client())
	return api
}

// TestRemoteFiles_CrossSurface_Parity is the REMOTE-05 merge-gate evidence:
// the daemon proxy and the TUI's RemoteFilesClient observe byte-identical
// upstream responses when pointed at the same fixture peer.
func TestRemoteFiles_CrossSurface_Parity(t *testing.T) {
	upstream := newFixtureRemotePeer(t)
	defer upstream.Close()

	api := newDaemonAPIWithUpstreamCert(t, upstream)
	daemonSrv := httptest.NewServer(api.Handler())
	defer daemonSrv.Close()

	// 1. Deposit the cap via the public POST endpoint (proves the
	//    daemon-side registration flow, not just direct field mutation).
	depositBody := map[string]string{
		"sessionId": "sid1",
		"baseUrl":   upstream.URL,
		"capToken":  fixtureCap,
	}
	depositRaw, _ := json.Marshal(depositBody)
	depositResp, err := http.Post(
		daemonSrv.URL+"/api/remote-files/caps",
		"application/json",
		bytes.NewReader(depositRaw),
	)
	if err != nil {
		t.Fatalf("cap deposit: %v", err)
	}
	depositResp.Body.Close()
	if depositResp.StatusCode != http.StatusOK {
		t.Fatalf("cap deposit status: want 200, got %d", depositResp.StatusCode)
	}

	// 2. Construct a direct tui.RemoteFilesClient using the upstream's TLS
	//    client (so it trusts the self-signed cert).
	directClient := tui.NewRemoteFilesClientForTest(upstream.URL, fixtureCap, upstream.Client())

	// ─── LIST parity ────────────────────────────────────────────────
	t.Run("list parity", func(t *testing.T) {
		// Surface A: daemon proxy
		proxyResp, err := http.Get(daemonSrv.URL + "/api/files/remote/sid1/list?path=.")
		if err != nil {
			t.Fatalf("proxy list: %v", err)
		}
		defer proxyResp.Body.Close()
		if proxyResp.StatusCode != http.StatusOK {
			t.Fatalf("proxy list status: want 200, got %d", proxyResp.StatusCode)
		}
		proxyBody, err := io.ReadAll(proxyResp.Body)
		if err != nil {
			t.Fatalf("proxy list body: %v", err)
		}

		// Surface B: direct RemoteFilesClient
		entries, truncated, err := directClient.ListFiles(context.Background(), "sid1", ".")
		assertNoCapInError(t, err)
		if err != nil {
			t.Fatalf("direct list: %v", err)
		}

		// Decode the proxy body to a FileListResponse for shape comparison.
		var proxyParsed files.FileListResponse
		if err := json.Unmarshal(proxyBody, &proxyParsed); err != nil {
			t.Fatalf("proxy list decode: %v", err)
		}

		// Byte-identical to the canonical body the upstream serves
		canonical, _ := canonicalListResponse()
		if !bytes.Equal(bytes.TrimSpace(proxyBody), bytes.TrimSpace(canonical)) {
			t.Errorf("proxy body != canonical upstream body\nproxy=%s\ncanonical=%s",
				proxyBody, canonical)
		}

		// Both surfaces observed the same entries + truncated flag.
		if len(entries) != len(proxyParsed.Entries) {
			t.Fatalf("entry count mismatch: direct=%d proxy=%d",
				len(entries), len(proxyParsed.Entries))
		}
		for i := range entries {
			if entries[i] != proxyParsed.Entries[i] {
				t.Errorf("entry[%d] mismatch:\n direct=%+v\n proxy=%+v",
					i, entries[i], proxyParsed.Entries[i])
			}
		}
		if truncated != proxyParsed.Truncated {
			t.Errorf("truncated mismatch: direct=%v proxy=%v", truncated, proxyParsed.Truncated)
		}
	})

	// ─── STAT parity ────────────────────────────────────────────────
	t.Run("stat parity", func(t *testing.T) {
		proxyResp, err := http.Get(daemonSrv.URL + "/api/files/remote/sid1/stat?path=a.txt")
		if err != nil {
			t.Fatalf("proxy stat: %v", err)
		}
		defer proxyResp.Body.Close()
		proxyBody, _ := io.ReadAll(proxyResp.Body)

		entry, err := directClient.StatFile(context.Background(), "sid1", "a.txt")
		assertNoCapInError(t, err)
		if err != nil {
			t.Fatalf("direct stat: %v", err)
		}

		canonical, canonicalEntry := canonicalStatResponse()
		if !bytes.Equal(bytes.TrimSpace(proxyBody), bytes.TrimSpace(canonical)) {
			t.Errorf("stat body mismatch\nproxy=%s\ncanonical=%s", proxyBody, canonical)
		}
		if entry != canonicalEntry {
			t.Errorf("direct stat entry != canonical\ndirect=%+v canonical=%+v",
				entry, canonicalEntry)
		}
	})

	// ─── READ parity ────────────────────────────────────────────────
	t.Run("read parity", func(t *testing.T) {
		proxyResp, err := http.Get(daemonSrv.URL + "/api/files/remote/sid1/read?path=a.txt")
		if err != nil {
			t.Fatalf("proxy read: %v", err)
		}
		defer proxyResp.Body.Close()
		proxyBody, _ := io.ReadAll(proxyResp.Body)

		directBody, contentType, err := directClient.ReadFile(context.Background(), "sid1", "a.txt")
		assertNoCapInError(t, err)
		if err != nil {
			t.Fatalf("direct read: %v", err)
		}

		if !bytes.Equal(proxyBody, []byte("hello world")) {
			t.Errorf("proxy read body: want 'hello world', got %q", proxyBody)
		}
		if !bytes.Equal(directBody, []byte("hello world")) {
			t.Errorf("direct read body: want 'hello world', got %q", directBody)
		}
		if !bytes.Equal(proxyBody, directBody) {
			t.Errorf("proxy vs direct read body byte mismatch\nproxy=%q direct=%q",
				proxyBody, directBody)
		}
		if !strings.HasPrefix(contentType, "text/plain") {
			t.Errorf("direct read content-type: want text/plain*, got %q", contentType)
		}
	})
}

// TestRemoteFiles_CrossSurface_CapAbsenceAsymmetry proves the proxy gate is
// daemon-local: with no cap deposited, the proxy returns 404 "no cap
// registered" while the direct RemoteFilesClient (carrying its own cap)
// still succeeds. This is the load-bearing property of the architecture
// from Plan 04's deviation note ("the TUI talks DIRECTLY to the remote
// webserver — no daemon proxy").
func TestRemoteFiles_CrossSurface_CapAbsenceAsymmetry(t *testing.T) {
	upstream := newFixtureRemotePeer(t)
	defer upstream.Close()

	// Fresh daemon WITHOUT any cap deposit.
	api := newDaemonAPIWithUpstreamCert(t, upstream)
	daemonSrv := httptest.NewServer(api.Handler())
	defer daemonSrv.Close()

	// Surface A: proxy must 404.
	proxyResp, err := http.Get(daemonSrv.URL + "/api/files/remote/sid1/list?path=.")
	if err != nil {
		t.Fatalf("proxy GET: %v", err)
	}
	defer proxyResp.Body.Close()
	if proxyResp.StatusCode != http.StatusNotFound {
		t.Fatalf("proxy without cap: want 404, got %d", proxyResp.StatusCode)
	}
	proxyBody, _ := io.ReadAll(proxyResp.Body)
	if !strings.Contains(string(proxyBody), "no cap registered") {
		t.Errorf("proxy 404 body missing marker: %s", proxyBody)
	}

	// Surface B: direct RemoteFilesClient still works (carries its own cap).
	directClient := tui.NewRemoteFilesClientForTest(upstream.URL, fixtureCap, upstream.Client())
	entries, _, err := directClient.ListFiles(context.Background(), "sid1", ".")
	assertNoCapInError(t, err)
	if err != nil {
		t.Fatalf("direct list (with cap, no daemon deposit): %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("direct list: want 2 entries, got %d", len(entries))
	}
}

// TestRemoteFiles_CrossSurface_401Propagation proves that when the upstream
// rejects the cap, both surfaces observe a 401-shaped error:
//   - proxy: status 401 verbatim
//   - direct: error containing "401"
// Cap-leakage source-grep is also asserted on the direct path.
func TestRemoteFiles_CrossSurface_401Propagation(t *testing.T) {
	upstream := newFixtureRemotePeer(t)
	defer upstream.Close()

	api := newDaemonAPIWithUpstreamCert(t, upstream)
	daemonSrv := httptest.NewServer(api.Handler())
	defer daemonSrv.Close()

	// Deposit a stale cap that the upstream will reject (401).
	depositBody := map[string]string{
		"sessionId": "sid1",
		"baseUrl":   upstream.URL,
		"capToken":  "STALE_CAP", // != FIXTURE_CAP
	}
	depositRaw, _ := json.Marshal(depositBody)
	depositResp, _ := http.Post(
		daemonSrv.URL+"/api/remote-files/caps",
		"application/json",
		bytes.NewReader(depositRaw),
	)
	depositResp.Body.Close()

	// Surface A: proxy must propagate the 401.
	proxyResp, err := http.Get(daemonSrv.URL + "/api/files/remote/sid1/list?path=.")
	if err != nil {
		t.Fatalf("proxy GET: %v", err)
	}
	defer proxyResp.Body.Close()
	if proxyResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("proxy 401 propagation: want 401, got %d", proxyResp.StatusCode)
	}

	// Surface B: direct RemoteFilesClient with a stale cap must error
	// containing "401" but NOT containing the stale cap value.
	staleClient := tui.NewRemoteFilesClientForTest(upstream.URL, "STALE_CAP", upstream.Client())
	_, _, err = staleClient.ListFiles(context.Background(), "sid1", ".")
	if err == nil {
		t.Fatal("direct list with stale cap: want error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("direct error missing 401: %v", err)
	}
	// Cap-leak: the cap MUST NOT appear in the error string.
	if strings.Contains(err.Error(), "STALE_CAP") {
		t.Errorf("CAP-LEAK: stale cap appeared in error: %v", err)
	}
}

// TestRemoteFiles_CrossSurface_CapInjectionPrevented proves the daemon proxy
// strips a caller-supplied ?cap= from the inbound URL so a malicious local
// caller cannot smuggle a different token. This re-states the existing
// TestRemoteFiles_CallerCapStripped assertion in a cross-surface frame.
func TestRemoteFiles_CrossSurface_CapInjectionPrevented(t *testing.T) {
	// Build an upstream that records every cap it sees so we can assert
	// only the registered cap reached it.
	var sawCaps []string
	mux := http.NewServeMux()
	listBody, _ := canonicalListResponse()
	mux.HandleFunc("GET /api/files/list", func(w http.ResponseWriter, r *http.Request) {
		sawCaps = append(sawCaps, r.URL.Query().Get("cap"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(listBody)
	})
	upstream := httptest.NewTLSServer(mux)
	defer upstream.Close()

	api := newDaemonAPIWithUpstreamCert(t, upstream)
	daemonSrv := httptest.NewServer(api.Handler())
	defer daemonSrv.Close()

	depositBody := map[string]string{
		"sessionId": "sid1",
		"baseUrl":   upstream.URL,
		"capToken":  "REGISTERED_CAP",
	}
	depositRaw, _ := json.Marshal(depositBody)
	depositResp, _ := http.Post(
		daemonSrv.URL+"/api/remote-files/caps",
		"application/json",
		bytes.NewReader(depositRaw),
	)
	depositResp.Body.Close()

	// Caller smuggles a different cap via the proxy query string.
	smuggleResp, err := http.Get(
		daemonSrv.URL + "/api/files/remote/sid1/list?path=.&cap=SMUGGLED_CAP",
	)
	if err != nil {
		t.Fatalf("proxy GET (smuggle): %v", err)
	}
	smuggleResp.Body.Close()

	if len(sawCaps) != 1 {
		t.Fatalf("upstream saw %d caps; want exactly 1: %v", len(sawCaps), sawCaps)
	}
	if sawCaps[0] != "REGISTERED_CAP" {
		t.Errorf("CAP-INJECTION leaked: upstream saw cap %q; want REGISTERED_CAP",
			sawCaps[0])
	}
	if sawCaps[0] == "SMUGGLED_CAP" {
		t.Error("CAP-INJECTION: smuggled cap reached upstream — proxy stripping broken")
	}
}

// assertNoCapInError fails the test if the cap token appears in the error's
// rendered string. Asserts the T-122-04-01 CAP-LEAK invariant uniformly
// across all parity sub-tests. nil err is fine.
func assertNoCapInError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), fixtureCap) {
		t.Errorf("CAP-LEAK: fixture cap %q appeared in error: %v",
			fixtureCap, err)
	}
}
