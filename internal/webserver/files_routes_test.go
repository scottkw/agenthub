// Phase 119 / WEB-02..WEB-04: integration tests for the four file routes
// mounted on the webserver mux under requireFilesRead.
//
// These tests drive the REAL TLS-listening WebServer (testServer helper) with
// REAL signed capability tokens (issueCapFor helper). They cover the
// integration delta between Phase 118 (Handler + middleware existed
// standalone) and Phase 119 (routes mounted on the public mux):
//
//   - Owner cap + files.read → 200 on GET list/stat/read + HEAD read
//   - Viewer cap (no files.read) → 403 with literal "files.read" in body
//   - Missing ?cap= → 401 (NOT 404 — route-existence-leak guard)
//   - POST/PUT/DELETE → 405 (Go 1.22+ method-prefix auto-rejection)
//   - Nil filesHandler → 503 with "files handler not configured"
//   - Path traversal → 403 (sandbox layer still active, Phase 118 invariant)
//
// Wrapper internals (HMAC bad-sig, malformed tokens, etc.) are covered by
// Phase 118's TestRequireFilesRead in capability_test.go — NOT duplicated here.
package webserver

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/files"
)

// newFilesTestServer constructs a started WebServer with the signing key
// wired, a session enabled, and a *files.Handler backed by t.TempDir()
// containing a 3-byte "hi\n" file at "hello.txt". Returns the server,
// HTTP client, session ID, and the resolved tempdir path so tests can
// assert on bytes/sizes.
func newFilesTestServer(t *testing.T) (ws *WebServer, client *http.Client, sid, tmp string) {
	t.Helper()
	ws, client = testServer(t)
	ws.SetSigningKey(capTestKey)

	sid = "files-sess"
	ws.EnableSession(sid)

	tmp = t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "hello.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatalf("write hello.txt: %v", err)
	}

	h := files.NewHandler(func(sessionID string) (*files.Sandbox, error) {
		if sessionID != sid {
			return nil, errors.New("unknown session")
		}
		return files.NewSandbox(tmp)
	})
	ws.SetFilesHandler(h)
	return ws, client, sid, tmp
}

// fileURL builds a URL against the test server for one of the file routes.
// path defaults to "." if empty; sessionID defaults to sid if empty (set by
// caller). The cap token is appended as ?cap=.
func fileURL(ws *WebServer, route, sid, path, token string) string {
	q := "?session=" + sid + "&path=" + path
	if token != "" {
		q += "&cap=" + token
	}
	return ws.BaseURL() + route + q
}

// doRequest issues req via client, reads the body, and returns the
// response (with Body already closed). The body bytes are populated into
// resp.Body via a bytes.Reader so tests can re-read as needed.
func doRequest(t *testing.T, client *http.Client, method, url string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("NewRequest(%s %s): %v", method, url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do(%s %s): %v", method, url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

// ---------------------------------------------------------------------------
// Owner cap → 200 (WEB-02)
// ---------------------------------------------------------------------------

func TestFilesRoutes_OwnerCapReturns200_List(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")

	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/list", sid, ".", token))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", resp.StatusCode, string(body))
	}
	// The List handler returns a FileListResponse JSON whose Entries contains
	// at least one FileEntry — verify a "Name" field is present.
	if !strings.Contains(string(body), `"name":`) {
		t.Errorf("expected JSON FileEntry with \"name\" field, body=%s", string(body))
	}
}

func TestFilesRoutes_OwnerCapReturns200_Stat(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")

	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/stat", sid, "hello.txt", token))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), `"name":`) {
		t.Errorf("expected JSON FileEntry, body=%s", string(body))
	}
}

func TestFilesRoutes_OwnerCapReturns200_Read_Get(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")

	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/read", sid, "hello.txt", token))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", resp.StatusCode, string(body))
	}
	if string(body) != "hi\n" {
		t.Errorf("expected body \"hi\\n\", got %q", string(body))
	}
}

func TestFilesRoutes_OwnerCapReturns200_Read_Head(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")

	resp, body := doRequest(t, client, http.MethodHead, fileURL(ws, "/api/files/read", sid, "hello.txt", token))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	// HEAD semantics: empty body, but Content-Length header reflects the size
	// http.ServeContent would have returned for the GET.
	if len(body) != 0 {
		t.Errorf("expected empty body for HEAD, got %d bytes", len(body))
	}
	if resp.ContentLength != 3 {
		t.Errorf("expected Content-Length=3 for hello.txt (\"hi\\n\"), got %d", resp.ContentLength)
	}
}

// ---------------------------------------------------------------------------
// Viewer cap (no files.read) → 403 with "files.read" in body (WEB-03 / FS-13)
// ---------------------------------------------------------------------------

func TestFilesRoutes_ViewerCapReturns403_List(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read")

	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/list", sid, ".", token))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer cap, got %d (body=%s)", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "files.read") {
		t.Errorf("expected body to contain literal \"files.read\", got %q", string(body))
	}
}

func TestFilesRoutes_ViewerCapReturns403_Stat(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read")

	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/stat", sid, "hello.txt", token))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "files.read") {
		t.Errorf("expected \"files.read\" in body, got %q", string(body))
	}
}

func TestFilesRoutes_ViewerCapReturns403_Read_Get(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read")

	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/read", sid, "hello.txt", token))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "files.read") {
		t.Errorf("expected \"files.read\" in body, got %q", string(body))
	}
}

func TestFilesRoutes_ViewerCapReturns403_Read_Head(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read")

	resp, _ := doRequest(t, client, http.MethodHead, fileURL(ws, "/api/files/read", sid, "hello.txt", token))
	// HEAD response bodies are empty per HTTP semantics; we assert status only.
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for viewer HEAD, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Missing ?cap= → 401 (WEB-02 SC#5, route-existence-leak guard)
// ---------------------------------------------------------------------------

func TestFilesRoutes_MissingCapReturns401_List(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	// No token argument — fileURL omits the cap query param.
	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/list", sid, ".", ""))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without cap, got %d (body=%s)", resp.StatusCode, string(body))
	}
}

func TestFilesRoutes_MissingCapReturns401_Read_Head(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	resp, _ := doRequest(t, client, http.MethodHead, fileURL(ws, "/api/files/read", sid, "hello.txt", ""))
	// Critical: must be 401, NOT 404. A 404 would leak whether the route exists.
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without cap on HEAD /read, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Method 405 — Go 1.22+ method-prefix mux auto-rejection (WEB-02 SC#3)
// ---------------------------------------------------------------------------

func TestFilesRoutes_PostReturns405(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")
	resp, _ := doRequest(t, client, http.MethodPost, fileURL(ws, "/api/files/list", sid, ".", token))
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST /api/files/list, got %d", resp.StatusCode)
	}
}

func TestFilesRoutes_PutReturns405_Read(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")
	resp, _ := doRequest(t, client, http.MethodPut, fileURL(ws, "/api/files/read", sid, "hello.txt", token))
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for PUT /api/files/read, got %d", resp.StatusCode)
	}
}

func TestFilesRoutes_DeleteReturns405_Stat(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")
	resp, _ := doRequest(t, client, http.MethodDelete, fileURL(ws, "/api/files/stat", sid, "hello.txt", token))
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for DELETE /api/files/stat, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// CR-02: HEAD /api/files/write probe — canWrite gate
// ---------------------------------------------------------------------------

// TestFilesRoutes_HeadWrite_WithFilesWrite_Returns200 asserts that
// HEAD /api/files/write with a cap carrying files.write returns 200.
// This is the web-share probeWrite path: the middleware fires, perm passes,
// Handler.Write short-circuits HEAD with 200+no-body.
func TestFilesRoutes_HeadWrite_WithFilesWrite_Returns200(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read,files.write")

	resp, body := doRequest(t, client, http.MethodHead, fileURL(ws, "/api/files/write", sid, ".", token))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for HEAD /write with files.write cap, got %d (body=%s)", resp.StatusCode, string(body))
	}
	if len(body) != 0 {
		t.Errorf("expected empty body for HEAD, got %d bytes: %q", len(body), body)
	}
}

// TestFilesRoutes_HeadWrite_WithoutFilesWrite_Returns403 asserts that
// HEAD /api/files/write with a cap lacking files.write returns 403 with
// body containing "files.write" — the exact signal useFilesCapability uses
// to resolve canWrite=false. Without the HEAD registration (CR-02 bug),
// the mux returned 405 before requireFilesWrite fired, making the probe
// always fail-open to canWrite=true.
func TestFilesRoutes_HeadWrite_WithoutFilesWrite_Returns403(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	// files.write perm is intentionally absent; viewer has only files.read.
	token := issueCapFor(t, ws, sid, "read,files.read")

	resp, body := doRequest(t, client, http.MethodHead, fileURL(ws, "/api/files/write", sid, ".", token))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for HEAD /write without files.write cap, got %d", resp.StatusCode)
	}
	// HEAD body is empty in HTTP semantics, but Go's http.Error DOES set the
	// body — doRequest reads it; just verify the status code is 403 which is
	// the load-bearing signal for isMissingFilesWritePerm().
	_ = body // status code alone is the contract for HEAD
}

// ---------------------------------------------------------------------------
// Nil filesHandler → 503 (defense-in-depth, Pitfall 2 mitigation)
// ---------------------------------------------------------------------------

func TestFilesRoutes_NilHandlerReturns503(t *testing.T) {
	// Construct the harness WITHOUT calling SetFilesHandler so the closure
	// observes ws.filesHandler == nil at request time.
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	sid := "nil-handler-sess"
	ws.EnableSession(sid)

	token := issueCapFor(t, ws, sid, "read,write,files.read")
	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/list", sid, ".", token))
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when filesHandler is nil, got %d (body=%s)", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "files handler not configured") {
		t.Errorf("expected body \"files handler not configured\", got %q", string(body))
	}
}

// ---------------------------------------------------------------------------
// WEB-05 defense-in-depth: no CSP header / no HTML Content-Type on file routes
// ---------------------------------------------------------------------------

// TestFilesRoutes_NoCSPHeader is a WEB-05 defense-in-depth regression guard.
// The CSP middleware (csp_mw.go) is intentionally mounted only on HTML
// routes — /api/files/* serves JSON/octet-stream and does not need (and
// must not have) CSP headers. If a future maintainer accidentally wraps
// these routes in cspHeaders, this test fires.
func TestFilesRoutes_NoCSPHeader(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")

	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/list", sid, ".", token))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", resp.StatusCode, string(body))
	}
	if got := resp.Header.Get("Content-Security-Policy"); got != "" {
		t.Errorf("unexpected CSP header on JSON route: %q", got)
	}
}

// TestFilesRoutes_NoHTMLContentType is a WEB-05 defense-in-depth regression
// guard. File routes MUST return JSON (List/Stat) or the resolved file MIME
// (Read) — never text/html. If a future change accidentally serves an HTML
// error page from a file route, the WEB-05 promise ("file browser flow
// adds no CSP surface") breaks silently. This test catches that at the
// package-test layer without needing a browser.
func TestFilesRoutes_NoHTMLContentType(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")

	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/list", sid, ".", token))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", resp.StatusCode, string(body))
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected Content-Type to start with application/json, got %q", ct)
	}
}

// ---------------------------------------------------------------------------
// Phase 137 / SHARE-03: Browse-aware perm matrix route tests (D-03/D-04)
//
// These tests pin the T-137-02/T-137-03 security delta using literal perm
// strings via issueCapFor. They pass immediately (the routes already exist);
// their value is pinning the RO-no-write invariant against future regressions.
// ---------------------------------------------------------------------------

// TestFilesRoutes_RO_BrowseOn_FilesReadRoute200 asserts that a browse-ON RO
// cap ("read,files.read") reaches the /api/files/list route with 200.
// D-04: files.read is present so requireFilesRead passes.
func TestFilesRoutes_RO_BrowseOn_FilesReadRoute200(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,files.read") // browse ON, RO code

	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/list", sid, ".", token))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for RO-browse-on cap on /list, got %d (body=%s)", resp.StatusCode, string(body))
	}
}

// TestFilesRoutes_RO_BrowseOn_WriteRoute403 asserts that a browse-ON RO cap
// ("read,files.read") is rejected (403) on the write route because files.write
// is absent. Pins T-137-03: RO code never writes even with browse ON.
func TestFilesRoutes_RO_BrowseOn_WriteRoute403(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,files.read") // browse ON, RO code — no files.write

	resp, _ := doRequest(t, client, http.MethodHead, fileURL(ws, "/api/files/write", sid, ".", token))
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for RO-browse-on cap on HEAD /write, got %d (T-137-03)", resp.StatusCode)
	}
}

// TestFilesRoutes_RW_BrowseOn_WriteRoute200 asserts that a browse-ON RW cap
// ("read,write,files.read,files.write") reaches the write route with 200.
// D-04: files.write present, requireFilesWrite passes.
func TestFilesRoutes_RW_BrowseOn_WriteRoute200(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read,files.write") // browse ON, RW code

	resp, _ := doRequest(t, client, http.MethodHead, fileURL(ws, "/api/files/write", sid, ".", token))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for RW-browse-on cap on HEAD /write, got %d (D-04)", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Sandbox traversal still rejected (Phase 118 invariant unchanged) — T-119-03
// ---------------------------------------------------------------------------

func TestFilesRoutes_TraversalRejected(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read")

	// Sandbox.validateAndClean rejects any path containing ".." atomically;
	// the handler converts that to a 403 "access denied: ...".
	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/list", sid, "../../etc/passwd", token))
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for traversal, got %d (body=%s)", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "access denied") {
		t.Errorf("expected sandbox rejection (\"access denied\"), got body %q", string(body))
	}
}
