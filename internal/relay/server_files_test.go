package relay

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/files"
)

// TestServer_FilesAPI_MountedOnRelay is the regression test for Phase 120 CR-01.
// Before the fix, FileBrowserTab in the Wails desktop GUI hit the relay over TCP
// at 127.0.0.1:<relayPort> and got 404 on every /api/files/* call because the
// relay mux only knew /sessions/*. This test verifies the relay HTTP surface now
// mounts the file API and that a real fetch against /api/files/list reaches the
// handler and returns a valid FileListResponse.
//
// The companion daemon-socket coverage lives in internal/files/handler_test.go.
// What this test specifically guards is the routing wiring on the relay's mux:
// a regression that removed the s.mux.HandleFunc calls in NewServer would 404
// this request even though the handler itself is correct.
func TestServer_FilesAPI_MountedOnRelay(t *testing.T) {
	// Build a real *files.Handler over a temp directory with one known file.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	sb, err := files.NewSandbox(dir)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}
	const sessionID = "files-relay-session"
	fh := files.NewHandler(func(id string) (*files.Sandbox, error) {
		if id == sessionID {
			return sb, nil
		}
		return nil, os.ErrNotExist
	})

	// Build a relay Server with the files handler mounted. nil HubManager is
	// fine because this test never hits /sessions/*; the test only exercises
	// the /api/files/list route, which is independent of session-WS plumbing.
	mgr := NewHubManager()
	t.Cleanup(mgr.Shutdown)
	srv := httptest.NewServer(NewServer(mgr, nil, fh))
	t.Cleanup(srv.Close)

	// Hit GET /api/files/list?session=...&path=. against the relay TCP listener,
	// mirroring exactly what FilesApiClient.list() does from the Wails webview.
	url := srv.URL + "/api/files/list?session=" + sessionID + "&path=."
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("relay /api/files/list status = %d; want 200; body=%s", resp.StatusCode, string(body))
	}

	var parsed files.FileListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal response: %v; body=%s", err, string(body))
	}
	if len(parsed.Entries) != 1 || parsed.Entries[0].Name != "hello.txt" {
		t.Errorf("entries = %+v; want exactly [hello.txt]", parsed.Entries)
	}
}

// TestServer_FilesWriteAPI_MountedOnRelay is the v3.5 analogue of the Phase 120
// CR-01 regression above, for the WRITE verbs. Phases 123-125 added write/upload/
// delete/rename/mkdir to the daemon unix-socket mux (internal/daemon/api.go) and
// the webserver (internal/webserver/server.go) but NOT to this relay loopback
// server — the surface the Wails desktop GUI actually uses for local file ops.
// The result: every desktop-GUI local save/upload/delete/rename/mkdir returned
// 404 ("Couldn't save the file"). Discovered during the deferred Phase 125
// desktop UAT. This test guards the write-route wiring + CORS preflight.
func TestServer_FilesWriteAPI_MountedOnRelay(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	sb, err := files.NewSandbox(dir)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}
	const sessionID = "files-relay-write-session"
	fh := files.NewHandler(func(id string) (*files.Sandbox, error) {
		if id == sessionID {
			return sb, nil
		}
		return nil, os.ErrNotExist
	})

	mgr := NewHubManager()
	t.Cleanup(mgr.Shutdown)
	srv := httptest.NewServer(NewServer(mgr, nil, fh))
	t.Cleanup(srv.Close)

	// PUT /api/files/write — the exact call FilesApiClient.write() makes from the
	// Wails webview. Must reach the handler (200), not 404.
	writeURL := srv.URL + "/api/files/write?session=" + sessionID + "&path=hello.txt"
	req, _ := http.NewRequest(http.MethodPut, writeURL, strings.NewReader("edited via relay"))
	req.Header.Set("If-Match", "*")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", writeURL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("relay PUT /api/files/write status = %d; want 200; body=%s", resp.StatusCode, string(body))
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "hello.txt")); string(got) != "edited via relay" {
		t.Errorf("file content = %q; want %q", string(got), "edited via relay")
	}

	// CORS preflight for the write verb must succeed and advertise PUT + If-Match,
	// or the browser blocks the actual request before it is sent.
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
		t.Fatalf("write preflight status = %d; want 204", preResp.StatusCode)
	}
	if m := preResp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(m, "PUT") {
		t.Errorf("preflight Allow-Methods = %q; want it to include PUT", m)
	}
	if h := preResp.Header.Get("Access-Control-Allow-Headers"); !strings.Contains(h, "If-Match") {
		t.Errorf("preflight Allow-Headers = %q; want it to include If-Match", h)
	}

	// The other write verbs must also be routed (not 404). A DELETE of a missing
	// path reaches the handler and returns a handler-level status, never 404-route.
	delURL := srv.URL + "/api/files/delete?session=" + sessionID + "&path=hello.txt"
	delReq, _ := http.NewRequest(http.MethodDelete, delURL, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	delResp.Body.Close()
	if delResp.StatusCode == http.StatusNotFound {
		t.Errorf("DELETE /api/files/delete returned 404 — route not mounted on relay")
	}
}

// TestServer_FilesAPI_NilHandler_404 confirms that callers passing nil for the
// filesHandler (e.g. existing tests, or any production surface that does not
// need the file API) get a clean 404 on /api/files/* rather than a nil-deref
// panic. This is the documented contract of NewServer.
func TestServer_FilesAPI_NilHandler_404(t *testing.T) {
	mgr := NewHubManager()
	t.Cleanup(mgr.Shutdown)
	srv := httptest.NewServer(NewServer(mgr, nil, nil))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/files/list?session=x&path=.")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("nil filesHandler: status = %d; want 404 (route not mounted)", resp.StatusCode)
	}
}
