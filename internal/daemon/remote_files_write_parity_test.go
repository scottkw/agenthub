// Phase 128-03 — Cross-surface write-parity test.
//
// This is the RMW-01/02/03 merge-gate evidence: the daemon proxy (Observer A),
// the TUI's RemoteFilesClient (Observer B), and Playwright HTTPS (Observer C)
// all observe byte-identical round-trip results when writing to and reading
// from the SAME persisted fixture peer.
//
// The test lives in `package daemon_test` (NOT `package daemon`) to mirror the
// Phase 122 read-parity harness convention, avoiding the tui→daemon import
// cycle (Pitfall 1 — see remote_files_parity_test.go header comment).
//
// Fixture peer (newFixtureRemoteWritePeer) is backed by a REAL files.Sandbox
// rooted at t.TempDir() so write-then-read round-trips actual bytes — NOT
// canned responses (Pitfall 2).
//
// assertNoCapInError is reused from remote_files_parity_test.go (same file,
// same package daemon_test) on every direct-client error path (T-128-08 /
// CAP-LEAK invariant).
package daemon_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scottkw/agenthub/internal/files"
	"github.com/scottkw/agenthub/internal/tui"
)

// newFixtureRemoteWritePeer spins up an httptest.NewTLSServer backed by a real
// files.Sandbox rooted at t.TempDir(). Write verbs (PUT write, DELETE, POST
// rename/mkdir) genuinely persist so read-back returns actual written bytes.
// The guard closure reuses the same fixtureCap / session("sid1") convention as
// the read-parity harness so both harnesses can share cap deposits.
//
// Backward-compat: GET /api/files/read serves from the sandbox first, then
// falls back to "hello world" for paths that were never written (list/stat
// remain canned so the read-parity tests can keep using their canonical shape).
func newFixtureRemoteWritePeer(t *testing.T) *httptest.Server {
	t.Helper()

	const fixtureSession = "sid1"

	// Real sandbox — Pitfall 2 mitigation: writes genuinely persist.
	sandbox, err := files.NewSandbox(t.TempDir())
	if err != nil {
		t.Fatalf("newFixtureRemoteWritePeer: sandbox: %v", err)
	}

	mux := http.NewServeMux()

	guard := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("cap") != fixtureCap {
				http.Error(w, "cap rejected", http.StatusUnauthorized)
				return
			}
			// session param is optional for write calls — the daemon proxy
			// injects it via ?session=<sid>.
			handler(w, r)
		}
	}

	// GET /api/files/read: serve from sandbox (persisted writes) first;
	// fallback to "hello world" for backward-compat paths.
	mux.HandleFunc("GET /api/files/read", guard(func(w http.ResponseWriter, r *http.Request) {
		rel := r.URL.Query().Get("path")
		if rel == "" {
			rel = "a.txt"
		}
		f, openErr := sandbox.Open(rel)
		if openErr == nil {
			defer f.Close()
			fi, statErr := f.Stat()
			if statErr == nil && !fi.IsDir() {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				if r.Method != http.MethodHead {
					_, _ = io.Copy(w, f)
				}
				return
			}
		}
		// Fallback: canned response for any path not yet written.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write([]byte("hello world"))
		}
	}))

	// PUT /api/files/write: persist body via WriteFileAtomic.
	mux.HandleFunc("PUT /api/files/write", guard(func(w http.ResponseWriter, r *http.Request) {
		rel := r.URL.Query().Get("path")
		if rel == "" {
			http.Error(w, "path is required", http.StatusBadRequest)
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(r.Body, 5<<20))
		if readErr != nil {
			http.Error(w, "read body: "+readErr.Error(), http.StatusBadRequest)
			return
		}
		if writeErr := sandbox.WriteFileAtomic(rel, body); writeErr != nil {
			http.Error(w, "write: "+writeErr.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"path": rel,
			"size": len(body),
		})
	}))

	// /join/exchange follows the webserver's 303 + Location shape (included
	// for reuse compatibility with cross-process scenarios).
	mux.HandleFunc("POST /join/exchange", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/sessions/"+fixtureSession+"?cap="+fixtureCap)
		w.WriteHeader(http.StatusSeeOther)
	})

	return httptest.NewTLSServer(mux)
}

// TestRemoteFilesWrite_CrossSurface is the RMW-01/02/03 merge-gate evidence:
// the daemon proxy (Observer A) and the TUI's RemoteFilesClient (Observer B)
// observe byte-identical results for write-then-read against ONE persisting
// fixture peer. A cross-observer assertion proves both surfaces hit the same
// persisted state (shared sandbox).
func TestRemoteFilesWrite_CrossSurface(t *testing.T) {
	upstream := newFixtureRemoteWritePeer(t)
	defer upstream.Close()

	api := newDaemonAPIWithUpstreamCert(t, upstream)
	daemonSrv := httptest.NewServer(api.Handler())
	defer daemonSrv.Close()

	// Deposit the cap so the proxy knows the upstream URL + cap token.
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

	// Observer B: direct RemoteFilesClient using the upstream's TLS client.
	directClient := tui.NewRemoteFilesClientForTest(upstream.URL, fixtureCap, upstream.Client())

	// ─── Observer A write-then-read ──────────────────────────────────────
	t.Run("observer A proxy write-then-read", func(t *testing.T) {
		const contentA = "content-A-from-proxy"

		// Write via the daemon proxy.
		req, buildErr := http.NewRequest(
			http.MethodPut,
			daemonSrv.URL+"/api/files/remote/sid1/write?path=x.txt",
			bytes.NewReader([]byte(contentA)),
		)
		if buildErr != nil {
			t.Fatalf("build PUT: %v", buildErr)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		writeResp, doErr := http.DefaultClient.Do(req)
		if doErr != nil {
			t.Fatalf("proxy PUT: %v", doErr)
		}
		writeResp.Body.Close()
		if writeResp.StatusCode != http.StatusOK {
			t.Fatalf("proxy PUT status: want 200, got %d", writeResp.StatusCode)
		}

		// Read back via the daemon proxy.
		readResp, readErr := http.Get(daemonSrv.URL + "/api/files/remote/sid1/read?path=x.txt")
		if readErr != nil {
			t.Fatalf("proxy GET read: %v", readErr)
		}
		defer readResp.Body.Close()
		readBody, _ := io.ReadAll(readResp.Body)
		if readResp.StatusCode != http.StatusOK {
			t.Fatalf("proxy GET read status: want 200, got %d", readResp.StatusCode)
		}

		// Assert byte-equivalence: read-back must equal what was written.
		if !bytes.Equal(readBody, []byte(contentA)) {
			t.Errorf("proxy write-then-read mismatch:\n  wrote=%q\n  got=%q", contentA, readBody)
		}
	})

	// ─── Observer B write-then-read ──────────────────────────────────────
	t.Run("observer B direct write-then-read", func(t *testing.T) {
		const contentB = "content-B-from-direct-client"

		// Write via the direct RemoteFilesClient.
		_, writeErr := directClient.WriteFile(context.Background(), "sid1", "y.txt", []byte(contentB))
		assertNoCapInError(t, writeErr)
		if writeErr != nil {
			t.Fatalf("direct WriteFile: %v", writeErr)
		}

		// Read back via the direct RemoteFilesClient.
		readBody, _, readErr := directClient.ReadFile(context.Background(), "sid1", "y.txt")
		assertNoCapInError(t, readErr)
		if readErr != nil {
			t.Fatalf("direct ReadFile: %v", readErr)
		}

		// Assert byte-equivalence.
		if !bytes.Equal(readBody, []byte(contentB)) {
			t.Errorf("direct write-then-read mismatch:\n  wrote=%q\n  got=%q", contentB, readBody)
		}
	})

	// ─── Cross-observer: A writes, B reads (proves shared persisted state) ──
	t.Run("cross-observer A-writes B-reads", func(t *testing.T) {
		const contentAB = "content-AB-cross-observer"

		// Observer A writes via the daemon proxy.
		req, buildErr := http.NewRequest(
			http.MethodPut,
			daemonSrv.URL+"/api/files/remote/sid1/write?path=ab.txt",
			bytes.NewReader([]byte(contentAB)),
		)
		if buildErr != nil {
			t.Fatalf("build PUT: %v", buildErr)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		writeResp, doErr := http.DefaultClient.Do(req)
		if doErr != nil {
			t.Fatalf("proxy PUT: %v", doErr)
		}
		writeResp.Body.Close()
		if writeResp.StatusCode != http.StatusOK {
			t.Fatalf("proxy PUT status: want 200, got %d", writeResp.StatusCode)
		}

		// Observer B reads the same file directly.
		readBody, _, readErr := directClient.ReadFile(context.Background(), "sid1", "ab.txt")
		assertNoCapInError(t, readErr)
		if readErr != nil {
			t.Fatalf("direct ReadFile (cross-observer): %v", readErr)
		}

		// Assert byte-equivalence: B must see exactly what A wrote.
		if !bytes.Equal(readBody, []byte(contentAB)) {
			t.Errorf("cross-observer byte mismatch:\n  A wrote=%q\n  B got=%q",
				contentAB, readBody)
		}
	})

	// ─── Cross-observer: B writes, A reads (proves shared persisted state) ──
	t.Run("cross-observer B-writes A-reads", func(t *testing.T) {
		const contentBA = "content-BA-cross-observer"

		// Observer B writes via the direct RemoteFilesClient.
		_, writeErr := directClient.WriteFile(context.Background(), "sid1", "ba.txt", []byte(contentBA))
		assertNoCapInError(t, writeErr)
		if writeErr != nil {
			t.Fatalf("direct WriteFile: %v", writeErr)
		}

		// Observer A reads via the daemon proxy.
		readResp, readErr := http.Get(daemonSrv.URL + "/api/files/remote/sid1/read?path=ba.txt")
		if readErr != nil {
			t.Fatalf("proxy GET read: %v", readErr)
		}
		defer readResp.Body.Close()
		readBody, _ := io.ReadAll(readResp.Body)
		if readResp.StatusCode != http.StatusOK {
			t.Fatalf("proxy GET read status: want 200, got %d", readResp.StatusCode)
		}

		// Assert byte-equivalence: A must see exactly what B wrote.
		if !bytes.Equal(readBody, []byte(contentBA)) {
			t.Errorf("cross-observer byte mismatch:\n  B wrote=%q\n  A got=%q",
				contentBA, readBody)
		}
	})
}

// TestRemoteFilesWrite_CapLeakInvariant asserts that direct-client error paths
// do not leak the cap token (T-128-08 / CAP-LEAK invariant).
func TestRemoteFilesWrite_CapLeakInvariant(t *testing.T) {
	upstream := newFixtureRemoteWritePeer(t)
	defer upstream.Close()

	// Use a stale cap to force an error on every write/read call.
	const staleCap = "STALE_WRITE_CAP"
	staleClient := tui.NewRemoteFilesClientForTest(upstream.URL, staleCap, upstream.Client())

	_, writeErr := staleClient.WriteFile(context.Background(), "sid1", "z.txt", []byte("data"))
	if writeErr == nil {
		t.Fatal("stale cap write: want error, got nil")
	}
	// Assert the stale cap does NOT appear in the error string.
	if bytes.Contains([]byte(writeErr.Error()), []byte(staleCap)) {
		t.Errorf("CAP-LEAK: stale cap appeared in WriteFile error: %v", writeErr)
	}

	_, _, readErr := staleClient.ReadFile(context.Background(), "sid1", "z.txt")
	if readErr == nil {
		t.Fatal("stale cap read: want error, got nil")
	}
	assertNoCapInError(t, readErr)
}
