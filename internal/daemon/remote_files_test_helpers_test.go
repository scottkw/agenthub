// Shared test helpers for daemon remote-files tests.
//
// These helpers were previously co-located in remote_files_parity_test.go.
// That file was deleted in Phase 136 (TUI removal). The helpers are retained
// here because relay_remote_files_test.go and other daemon_test files depend
// on them without any TUI import.
//
// Nothing in this file imports internal/tui.

package daemon_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/files"
)

// fixtureCap is the canonical cap token used throughout the daemon remote-files tests.
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

	// Write verbs (Phase 128). The relay-mount regression test
	// (relay_remote_files_test.go) needs a real upstream write endpoint so
	// a successfully-routed PUT reaches a 200 rather than an upstream 404 that
	// would be indistinguishable from a relay route-miss.
	mux.HandleFunc("PUT /api/files/write", guard(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	// /join/exchange follows the webserver's 303 + Location shape (Phase 87).
	mux.HandleFunc("POST /join/exchange", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/sessions/sid1?cap="+fixtureCap)
		w.WriteHeader(http.StatusSeeOther)
	})

	return httptest.NewTLSServer(mux)
}

func canonicalStatResponse() ([]byte, files.FileEntry) {
	entry := files.FileEntry{Name: "a.txt", Size: 100, IsDir: false}
	body, err := json.Marshal(entry)
	if err != nil {
		panic(err)
	}
	return body, entry
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
