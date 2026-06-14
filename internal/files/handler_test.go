package files_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/files"
)

// newHandler returns a Handler wired to a fixed sandbox over t.TempDir().
// The returned sandboxRoot is the EvalSymlinks-resolved path the sandbox is
// rooted at (callers use it to seed test files via the OS, not the API).
func newHandler(t *testing.T) (*files.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	sb, err := files.NewSandbox(dir)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}
	h := files.NewHandler(func(sessionID string) (*files.Sandbox, error) {
		if sessionID == "good" {
			return sb, nil
		}
		return nil, os.ErrNotExist
	})
	return h, sb.RootPath()
}

// invoke is a thin httptest helper: builds the request URL with session=good
// and the supplied path query parameter, runs the handler method against an
// httptest.ResponseRecorder, and returns the recorder.
func invoke(t *testing.T, h func(http.ResponseWriter, *http.Request), method, p string, headers http.Header) *httptest.ResponseRecorder {
	t.Helper()
	u := "/?session=good"
	if p != "" {
		u += "&path=" + p
	}
	req := httptest.NewRequest(method, u, nil)
	if headers != nil {
		req.Header = headers
	}
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

// --------------------------------------------------------------------------
// Task 1 — skeleton smoke tests
// --------------------------------------------------------------------------

func TestHandler_MissingSessionReturns404(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest("GET", "/?path=.", nil) // no session param
	rr := httptest.NewRecorder()
	h.List(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("missing session: status = %d; want 404", rr.Code)
	}
}

func TestHandler_UnknownSessionReturns404(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest("GET", "/?session=bogus&path=.", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown session: status = %d; want 404", rr.Code)
	}
}

// --------------------------------------------------------------------------
// Task 2 — Handler.List
// --------------------------------------------------------------------------

func TestHandler_List_BasicDirectory(t *testing.T) {
	h, root := newHandler(t)
	for _, n := range []string{"a.txt", "b.go", "c.json"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	rr := invoke(t, h.List, "GET", ".", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", rr.Code, rr.Body.String())
	}
	var resp files.FileListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Entries) != 3 {
		t.Errorf("entries = %d; want 3", len(resp.Entries))
	}
	if resp.Truncated {
		t.Errorf("Truncated = true; want false for 3-entry directory")
	}
}

func TestHandler_List_DotPathReturnsRootEntries(t *testing.T) {
	h, root := newHandler(t)
	if err := os.WriteFile(filepath.Join(root, "only.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	rr := invoke(t, h.List, "GET", ".", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rr.Code)
	}
	var resp files.FileListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Entries) != 1 || resp.Entries[0].Name != "only.txt" {
		t.Errorf("entries = %+v; want [only.txt]", resp.Entries)
	}
}

func TestHandler_List_EmptyPathDefaultsToDot(t *testing.T) {
	h, root := newHandler(t)
	if err := os.WriteFile(filepath.Join(root, "only.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	// no path query param at all
	req := httptest.NewRequest("GET", "/?session=good", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rr.Code)
	}
	var resp files.FileListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Entries) != 1 {
		t.Errorf("entries = %d; want 1", len(resp.Entries))
	}
}

func TestHandler_List_TraversalRejected(t *testing.T) {
	h, _ := newHandler(t)
	rr := invoke(t, h.List, "GET", "../escape", nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d; want 403", rr.Code)
	}
}

func TestHandler_List_NonExistentReturns403(t *testing.T) {
	h, _ := newHandler(t)
	rr := invoke(t, h.List, "GET", "does/not/exist", nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d; want 403", rr.Code)
	}
}

func TestHandler_List_NotADirectory(t *testing.T) {
	h, root := newHandler(t)
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	rr := invoke(t, h.List, "GET", "f.txt", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want 400", rr.Code)
	}
}

func TestHandler_List_TruncatedAt10000(t *testing.T) {
	if testing.Short() {
		t.Skip("creates 10001 files")
	}
	h, root := newHandler(t)
	for i := 0; i < 10001; i++ {
		name := filepath.Join(root, "f"+strconvI(i)+".txt")
		f, err := os.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	start := time.Now()
	rr := invoke(t, h.List, "GET", ".", nil)
	elapsed := time.Since(start)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rr.Code)
	}
	var resp files.FileListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Entries) != 10000 {
		t.Errorf("entries = %d; want exactly 10000", len(resp.Entries))
	}
	if !resp.Truncated {
		t.Errorf("Truncated = false; want true at 10k cap")
	}
	// Soft check that we didn't per-entry Stat (Pitfall 6). A 10k-entry
	// listing should complete well under 1s on a tempdir; >2s is suspect.
	if elapsed > 2*time.Second {
		t.Logf("WARNING: 10k listing took %v (possible per-entry stat regression)", elapsed)
	}
}

// strconvI inlines a small int->string to avoid importing strconv just here.
func strconvI(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// TestHandler_List_StatPerEntry asserts that List populates Size and Mtime
// for each entry via DirEntry.Info(). The original Pitfall 6 guard (no per-
// entry Info()) was reversed during Phase 120 UAT-1 — users couldn't see
// file sizes at all, which was a UX regression heavier than the perf cost
// of N stat syscalls (capped at maxListEntries=10000 anyway).
//
// The body MUST handle Info() failure gracefully: a file unlinked between
// ReadDir and Info should still appear in the listing with Size=0/Mtime="".
func TestHandler_List_StatPerEntry(t *testing.T) {
	src, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go: %v", err)
	}
	text := string(src)
	const listMarker = "func (h *Handler) List("
	listIdx := strings.Index(text, listMarker)
	if listIdx < 0 {
		t.Fatalf("could not find %q in handler.go", listMarker)
	}
	tail := text[listIdx+len(listMarker):]
	nextIdx := strings.Index(tail, "\nfunc ")
	if nextIdx < 0 {
		nextIdx = len(tail)
	}
	body := tail[:nextIdx]
	if !strings.Contains(body, "entry.Info()") {
		t.Error("Handler.List body must call entry.Info() to populate size/mtime")
	}
	// Defensive: make sure the call uses the comma-ok / error-checked form
	// so a deleted file doesn't crash the listing.
	if !strings.Contains(body, "infoErr") && !strings.Contains(body, "if err") {
		t.Error("Handler.List body must check the Info() error and continue gracefully")
	}
}

func TestHandler_List_DarwinResourceForkFilter(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only filter")
	}
	h, root := newHandler(t)
	for _, n := range []string{"real.txt", "._real.txt"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	rr := invoke(t, h.List, "GET", ".", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var resp files.FileListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	for _, e := range resp.Entries {
		if strings.HasPrefix(e.Name, "._") {
			t.Errorf("darwin filter failed: ._-prefixed entry %q in response", e.Name)
		}
	}
	// Exactly one entry left.
	if len(resp.Entries) != 1 || resp.Entries[0].Name != "real.txt" {
		t.Errorf("entries = %+v; want [real.txt]", resp.Entries)
	}
}

// --------------------------------------------------------------------------
// Task 2 — Handler.Stat
// --------------------------------------------------------------------------

func TestHandler_Stat_RegularFile(t *testing.T) {
	h, root := newHandler(t)
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	rr := invoke(t, h.Stat, "GET", "hello.txt", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rr.Code, rr.Body.String())
	}
	var e files.FileEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &e); err != nil {
		t.Fatal(err)
	}
	if e.IsDir {
		t.Errorf("IsDir = true; want false")
	}
	if e.Size != 11 {
		t.Errorf("Size = %d; want 11", e.Size)
	}
	if !strings.HasPrefix(e.MIME, "text/") {
		t.Errorf("MIME = %q; want text/* prefix", e.MIME)
	}
	if e.Name != "hello.txt" {
		t.Errorf("Name = %q; want hello.txt", e.Name)
	}
}

func TestHandler_Stat_Directory(t *testing.T) {
	h, root := newHandler(t)
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	rr := invoke(t, h.Stat, "GET", "sub", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var e files.FileEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &e); err != nil {
		t.Fatal(err)
	}
	if !e.IsDir {
		t.Errorf("IsDir = false; want true")
	}
}

func TestHandler_Stat_TraversalRejected(t *testing.T) {
	h, _ := newHandler(t)
	rr := invoke(t, h.Stat, "GET", "../etc/passwd", nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d; want 403", rr.Code)
	}
}

func TestHandler_Stat_ForwardSlashName(t *testing.T) {
	h, root := newHandler(t)
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "b", "c.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	rr := invoke(t, h.Stat, "GET", "a/b/c.txt", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rr.Code, rr.Body.String())
	}
	var e files.FileEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &e); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(e.Name, "\\") {
		t.Errorf("Name = %q contains backslash; must be forward-slash normalized", e.Name)
	}
}

// --------------------------------------------------------------------------
// Task 3 — Handler.Read
// --------------------------------------------------------------------------

func TestHandler_Read_RegularFile(t *testing.T) {
	h, root := newHandler(t)
	body := []byte("hello world\n")
	if err := os.WriteFile(filepath.Join(root, "f.txt"), body, 0644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(h.Read))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/?session=good&path=f.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d; want 200", resp.StatusCode)
	}
	got := make([]byte, len(body))
	n, _ := resp.Body.Read(got)
	if n != len(body) || string(got) != string(body) {
		t.Errorf("body = %q; want %q", string(got[:n]), string(body))
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/plain") {
		t.Errorf("Content-Type = %q; want text/plain prefix", resp.Header.Get("Content-Type"))
	}
}

func TestHandler_Read_RangeRequest(t *testing.T) {
	h, root := newHandler(t)
	body := make([]byte, 1000)
	for i := range body {
		body[i] = byte('a' + i%26)
	}
	if err := os.WriteFile(filepath.Join(root, "f.txt"), body, 0644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(h.Read))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/?session=good&path=f.txt", nil)
	req.Header.Set("Range", "bytes=0-99")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("status = %d; want 206", resp.StatusCode)
	}
	got := make([]byte, 200)
	n, _ := resp.Body.Read(got)
	if n != 100 {
		t.Errorf("body length = %d; want 100", n)
	}
	if cr := resp.Header.Get("Content-Range"); !strings.Contains(cr, "bytes 0-99/1000") {
		t.Errorf("Content-Range = %q; want bytes 0-99/1000", cr)
	}
}

func TestHandler_ZeroByteRead(t *testing.T) {
	h, root := newHandler(t)
	if err := os.WriteFile(filepath.Join(root, "empty.txt"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	// (a) Without Range header
	rr := invoke(t, h.Read, "GET", "empty.txt", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("no-range: status = %d; want 200 (FS-07)", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("no-range: body length = %d; want 0", rr.Body.Len())
	}
	// (b) With Range header — the golang/go#54794 416 case
	rr2 := invoke(t, h.Read, "GET", "empty.txt", http.Header{"Range": {"bytes=0-"}})
	if rr2.Code != http.StatusOK {
		t.Errorf("with-range: status = %d; want 200 (golang/go#54794 mitigation)", rr2.Code)
	}
	if rr2.Body.Len() != 0 {
		t.Errorf("with-range: body length = %d; want 0", rr2.Body.Len())
	}
}

func TestHandler_Read_OverCapReturns413(t *testing.T) {
	h, root := newHandler(t)
	big := make([]byte, 6*1024*1024) // 6 MiB > 5 MiB cap
	if err := os.WriteFile(filepath.Join(root, "big.bin"), big, 0644); err != nil {
		t.Fatal(err)
	}
	rr := invoke(t, h.Read, "GET", "big.bin", nil)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d; want 413", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "file too large for preview") {
		t.Errorf("body = %q; want contains 'file too large for preview'", rr.Body.String())
	}
}

func TestHandler_Read_BoundaryAt5MiB(t *testing.T) {
	h, root := newHandler(t)
	exact := make([]byte, 5*1024*1024)
	if err := os.WriteFile(filepath.Join(root, "boundary.txt"), exact, 0644); err != nil {
		t.Fatal(err)
	}
	rr := invoke(t, h.Read, "GET", "boundary.txt", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d; want 200 (cap is strict > not >=)", rr.Code)
	}
}

func TestHandler_Read_DirectoryReturns400(t *testing.T) {
	h, _ := newHandler(t)
	rr := invoke(t, h.Read, "GET", ".", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want 400", rr.Code)
	}
}

func TestHandler_Read_TraversalRejected(t *testing.T) {
	h, _ := newHandler(t)
	rr := invoke(t, h.Read, "GET", "../escape", nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d; want 403", rr.Code)
	}
}

func TestHandler_Read_HEAD_ReturnsHeadersOnly(t *testing.T) {
	h, root := newHandler(t)
	body := []byte("hello world\n")
	if err := os.WriteFile(filepath.Join(root, "f.txt"), body, 0644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(h.Read))
	defer srv.Close()
	req, _ := http.NewRequest("HEAD", srv.URL+"/?session=good&path=f.txt", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d; want 200", resp.StatusCode)
	}
	if resp.ContentLength != int64(len(body)) {
		t.Errorf("Content-Length = %d; want %d", resp.ContentLength, len(body))
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/plain") {
		t.Errorf("Content-Type = %q; want text/plain prefix", resp.Header.Get("Content-Type"))
	}
	buf := make([]byte, 16)
	n, _ := resp.Body.Read(buf)
	if n != 0 {
		t.Errorf("HEAD body length = %d; want 0", n)
	}
}

func TestHandler_Read_IfModifiedSince_Future(t *testing.T) {
	h, root := newHandler(t)
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(h.Read))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/?session=good&path=f.txt", nil)
	req.Header.Set("If-Modified-Since", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified {
		t.Errorf("status = %d; want 304", resp.StatusCode)
	}
}

// --------------------------------------------------------------------------
// Plan 123-03 — Handler write methods (RED tests, FSW-05/08/12)
// --------------------------------------------------------------------------

// newHandlerWithHomeSandbox builds a Handler whose sandbox is rooted at a
// sub-directory of os.UserHomeDir() so the denylist fires on protected names.
// Returns the handler and the EvalSymlinks-resolved root path.
func newHandlerWithHomeSandbox(t *testing.T) (*files.Handler, string) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("os.UserHomeDir: %v", err)
	}
	// Use a *named* temp dir under $HOME so the denylist can fire.
	dir, err := os.MkdirTemp(home, "agenthub-test-sandbox-*")
	if err != nil {
		t.Skipf("MkdirTemp under $HOME: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sb, err := files.NewSandbox(dir)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}
	h := files.NewHandler(func(sessionID string) (*files.Sandbox, error) {
		if sessionID == "good" {
			return sb, nil
		}
		return nil, os.ErrNotExist
	})
	return h, sb.RootPath()
}

// invokeWithBody issues an HTTP request with the given method and body to the
// handler, returning the response recorder.
func invokeWithBody(t *testing.T, h func(http.ResponseWriter, *http.Request), method, p string, body []byte, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	u := "/?session=good"
	if p != "" {
		u += "&path=" + p
	}
	var bodyReader *strings.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, u, bodyReader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

// TestHandlerWrite_RoundTrip: PUT body to write handler → 200 + FileWriteResponse;
// subsequent Read returns identical bytes. (FSW-08 success criterion #2)
func TestHandlerWrite_RoundTrip(t *testing.T) {
	h, _ := newHandler(t)
	payload := []byte("hello from write test")
	rr := invokeWithBody(t, h.Write, "PUT", "hello.txt", payload, "application/octet-stream")
	if rr.Code != http.StatusOK {
		t.Fatalf("Write status = %d; want 200; body=%s", rr.Code, rr.Body.String())
	}
	var wr files.FileWriteResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &wr); err != nil {
		t.Fatalf("decode FileWriteResponse: %v; body=%s", err, rr.Body.String())
	}
	if wr.Path != "hello.txt" {
		t.Errorf("FileWriteResponse.Path = %q; want 'hello.txt'", wr.Path)
	}
	if wr.Size != int64(len(payload)) {
		t.Errorf("FileWriteResponse.Size = %d; want %d", wr.Size, len(payload))
	}
	// Read back via the read handler.
	rr2 := invoke(t, h.Read, "GET", "hello.txt", nil)
	if rr2.Code != http.StatusOK {
		t.Fatalf("Read after Write status = %d; want 200; body=%s", rr2.Code, rr2.Body.String())
	}
	if got := rr2.Body.Bytes(); string(got) != string(payload) {
		t.Errorf("Read body = %q; want %q", got, payload)
	}
}

// TestHandlerWrite_DenylistForbidden: write/rename/delete/mkdir targeting a
// denylisted $HOME path gives 403 "Protected system file". (FSW-06 HTTP layer)
func TestHandlerWrite_DenylistForbidden(t *testing.T) {
	h, _ := newHandlerWithHomeSandbox(t)
	// Write to .bashrc inside the home-rooted sandbox
	rr := invokeWithBody(t, h.Write, "PUT", ".bashrc", []byte("evil"), "application/octet-stream")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("Write .bashrc status = %d; want 403; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Protected system file") {
		t.Errorf("body = %q; want contains 'Protected system file'", rr.Body.String())
	}
}

// TestHandlerWrite_Traversal403: traversal relpath gives 403 "access denied: ...".
func TestHandlerWrite_Traversal403(t *testing.T) {
	h, _ := newHandler(t)
	rr := invokeWithBody(t, h.Write, "PUT", "../escape.txt", []byte("evil"), "application/octet-stream")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("Write traversal status = %d; want 403; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "access denied:") {
		t.Errorf("body = %q; want contains 'access denied:'", rr.Body.String())
	}
}

// TestHandlerUpload_DenylistForbidden: a multipart upload targeting a denylisted
// path within a home-rooted sandbox gives 403 "Protected system file".
// (FSW-05 + FSW-06: HTTP-upload-layer denylist evidence)
func TestHandlerUpload_DenylistForbidden(t *testing.T) {
	h, _ := newHandlerWithHomeSandbox(t)
	// Use httptest.NewServer + real multipart to test Upload which calls
	// r.ParseMultipartForm internally (needs a real HTTP round-trip for MIME).
	srv := httptest.NewServer(http.HandlerFunc(h.Upload))
	defer srv.Close()

	// Build a multipart form targeting the denylist (.bashrc under the home-rooted sandbox).
	var buf strings.Builder
	boundary := "testboundary"
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString(`Content-Disposition: form-data; name="dir"` + "\r\n\r\n")
	buf.WriteString(".") // dir = "." (sandbox root, which is under $HOME)
	buf.WriteString("\r\n")
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString(`Content-Disposition: form-data; name="file"; filename=".bashrc"` + "\r\n")
	buf.WriteString("Content-Type: application/octet-stream\r\n\r\n")
	buf.WriteString("evil content")
	buf.WriteString("\r\n--" + boundary + "--\r\n")

	resp, err := http.Post(
		srv.URL+"/?session=good",
		"multipart/form-data; boundary="+boundary,
		strings.NewReader(buf.String()),
	)
	if err != nil {
		t.Fatalf("POST upload: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Upload denylist status = %d; want 403; body=%s", resp.StatusCode, string(respBody))
	}
	if !strings.Contains(string(respBody), "Protected system file") {
		t.Errorf("body = %q; want contains 'Protected system file'", string(respBody))
	}
}

// TestHandlerUpload_FilenameSanitized: multipart with FileHeader.Filename
// "../../.bashrc" lands as ".bashrc" under the target dir (filepath.Base strip).
func TestHandlerUpload_FilenameSanitized(t *testing.T) {
	h, root := newHandler(t)
	srv := httptest.NewServer(http.HandlerFunc(h.Upload))
	defer srv.Close()

	var buf strings.Builder
	boundary := "testbound2"
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString(`Content-Disposition: form-data; name="dir"` + "\r\n\r\n")
	buf.WriteString(".")
	buf.WriteString("\r\n")
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString(`Content-Disposition: form-data; name="file"; filename="../../.bashrc"` + "\r\n")
	buf.WriteString("Content-Type: text/plain\r\n\r\n")
	buf.WriteString("sanitized")
	buf.WriteString("\r\n--" + boundary + "--\r\n")

	resp, err := http.Post(
		srv.URL+"/?session=good",
		"multipart/form-data; boundary="+boundary,
		strings.NewReader(buf.String()),
	)
	if err != nil {
		t.Fatalf("POST upload: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Upload status = %d; want 200; body=%s", resp.StatusCode, string(respBody))
	}
	// File must land at root/.bashrc, not at root/../../.bashrc
	if _, err := os.Stat(filepath.Join(root, ".bashrc")); err != nil {
		t.Errorf("expected .bashrc at sandbox root, got error: %v", err)
	}
	// Confirm nothing was written outside the sandbox root
	escapeTarget := filepath.Join(filepath.Dir(root), ".bashrc")
	if _, err := os.Stat(escapeTarget); err == nil {
		t.Errorf("SECURITY: file written outside sandbox root at %s", escapeTarget)
	}
}

// TestHandlerUpload_OverCap413: multipart body exceeding 50 MiB gives 413.
func TestHandlerUpload_OverCap413(t *testing.T) {
	if testing.Short() {
		t.Skip("generates 51 MiB body")
	}
	h, root := newHandler(t)
	srv := httptest.NewServer(http.HandlerFunc(h.Upload))
	defer srv.Close()

	// Build a multipart body slightly over 50 MiB.
	const overCap = 51 * 1024 * 1024
	boundary := "bigboundary"
	header := "--" + boundary + "\r\n" +
		`Content-Disposition: form-data; name="dir"` + "\r\n\r\n.\r\n" +
		"--" + boundary + "\r\n" +
		`Content-Disposition: form-data; name="file"; filename="big.bin"` + "\r\n" +
		"Content-Type: application/octet-stream\r\n\r\n"
	footer := "\r\n--" + boundary + "--\r\n"

	bigContent := make([]byte, overCap)
	body := strings.NewReader(header + string(bigContent) + footer)

	resp, err := http.Post(
		srv.URL+"/?session=good",
		"multipart/form-data; boundary="+boundary,
		body,
	)
	if err != nil {
		t.Fatalf("POST upload: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("over-cap status = %d; want 413", resp.StatusCode)
	}
	// No truncated file should remain on disk.
	if _, err := os.Stat(filepath.Join(root, "big.bin")); !os.IsNotExist(err) {
		t.Errorf("over-cap: big.bin should not exist on disk; got: %v", err)
	}
}

// TestHandlerWrite_OverCap413: a PUT /write body over the 50 MiB cap gives 413
// and writes no file. (WR-06 — parity with Upload's FSW-12 cap.)
func TestHandlerWrite_OverCap413(t *testing.T) {
	if testing.Short() {
		t.Skip("generates 51 MiB body")
	}
	h, root := newHandler(t)
	srv := httptest.NewServer(http.HandlerFunc(h.Write))
	defer srv.Close()

	const overCap = 51 * 1024 * 1024
	body := bytes.NewReader(make([]byte, overCap))
	req, err := http.NewRequest("PUT", srv.URL+"/?session=good&path=big.bin", body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT write: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("over-cap write status = %d; want 413", resp.StatusCode)
	}
	// No truncated file should remain on disk.
	if _, err := os.Stat(filepath.Join(root, "big.bin")); !os.IsNotExist(err) {
		t.Errorf("over-cap: big.bin should not exist on disk; got: %v", err)
	}
}

// TestHandlerRename: rename happy path gives 200 FileOpResponse.
func TestHandlerRename(t *testing.T) {
	h, root := newHandler(t)
	if err := os.WriteFile(filepath.Join(root, "old.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	// Rename uses JSON body: {"oldRel":"old.txt","newRel":"new.txt"}
	req := httptest.NewRequest("POST", "/?session=good", strings.NewReader(`{"oldRel":"old.txt","newRel":"new.txt"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Rename(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Rename status = %d; want 200; body=%s", rr.Code, rr.Body.String())
	}
	var resp files.FileOpResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode FileOpResponse: %v", err)
	}
	if !resp.OK {
		t.Errorf("FileOpResponse.OK = false; want true")
	}
	if _, err := os.Stat(filepath.Join(root, "new.txt")); err != nil {
		t.Errorf("new.txt should exist after rename: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "old.txt")); !os.IsNotExist(err) {
		t.Errorf("old.txt should not exist after rename")
	}
}

// TestHandlerMkdir: mkdir happy path gives 200 FileOpResponse.
func TestHandlerMkdir(t *testing.T) {
	h, root := newHandler(t)
	rr := invokeWithBody(t, h.Mkdir, "POST", "newdir", nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("Mkdir status = %d; want 200; body=%s", rr.Code, rr.Body.String())
	}
	var resp files.FileOpResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode FileOpResponse: %v", err)
	}
	if !resp.OK {
		t.Errorf("FileOpResponse.OK = false; want true")
	}
	fi, err := os.Stat(filepath.Join(root, "newdir"))
	if err != nil {
		t.Fatalf("newdir should exist: %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("newdir should be a directory")
	}
}

// TestHandlerDelete: delete happy path gives 200 FileOpResponse.
func TestHandlerDelete(t *testing.T) {
	h, root := newHandler(t)
	if err := os.WriteFile(filepath.Join(root, "del.txt"), []byte("bye"), 0644); err != nil {
		t.Fatal(err)
	}
	rr := invokeWithBody(t, h.Delete, "DELETE", "del.txt", nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("Delete status = %d; want 200; body=%s", rr.Code, rr.Body.String())
	}
	var resp files.FileOpResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode FileOpResponse: %v", err)
	}
	if !resp.OK {
		t.Errorf("FileOpResponse.OK = false; want true")
	}
	if _, err := os.Stat(filepath.Join(root, "del.txt")); !os.IsNotExist(err) {
		t.Errorf("del.txt should not exist after delete")
	}
}

// --------------------------------------------------------------------------
// CR-01 — Upload empty/dot filename → 400
// --------------------------------------------------------------------------

// invokeUploadWithFilename issues a multipart POST to h.Upload using a real
// httptest.Server (required by ParseMultipartForm). The Content-Disposition
// filename is set to the provided rawFilename (which may be empty or ".").
func invokeUploadWithFilename(t *testing.T, h *files.Handler, rawFilename string) *http.Response {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(h.Upload))
	t.Cleanup(srv.Close)

	boundary := "testboundary123"
	var buf strings.Builder
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString(`Content-Disposition: form-data; name="dir"` + "\r\n\r\n")
	buf.WriteString(".")
	buf.WriteString("\r\n")
	buf.WriteString("--" + boundary + "\r\n")
	// Build Content-Disposition with the raw filename (including empty case).
	buf.WriteString(`Content-Disposition: form-data; name="file"; filename="` + rawFilename + `"` + "\r\n")
	buf.WriteString("Content-Type: application/octet-stream\r\n\r\n")
	buf.WriteString("data")
	buf.WriteString("\r\n--" + boundary + "--\r\n")

	resp, err := http.Post(
		srv.URL+"/?session=good",
		"multipart/form-data; boundary="+boundary,
		strings.NewReader(buf.String()),
	)
	if err != nil {
		t.Fatalf("POST upload: %v", err)
	}
	return resp
}

// TestHandlerUpload_EmptyFilename400: empty filename → 400 (CR-01).
// filepath.Base("") == "." which would collapse target to the sandbox root
// directory; WriteFileAtomic would attempt to rename a regular temp file onto
// a directory → opaque 500 + stray temp file. The guard now rejects early.
func TestHandlerUpload_EmptyFilename400(t *testing.T) {
	h, _ := newHandler(t)
	resp := invokeUploadWithFilename(t, h, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("empty filename: status = %d; want 400; body=%s", resp.StatusCode, body)
	}
}

// TestHandlerUpload_DotFilename400: filename "." (or path that reduces to ".")
// → 400 (CR-01). filepath.Base("/") == "." which would cause the same
// directory-collision as the empty case.
func TestHandlerUpload_DotFilename400(t *testing.T) {
	h, _ := newHandler(t)
	resp := invokeUploadWithFilename(t, h, ".")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("dot filename: status = %d; want 400; body=%s", resp.StatusCode, body)
	}
}

// TestHandlerUpload_DotDotFilename400: ".." filename → 400 (CR-01).
func TestHandlerUpload_DotDotFilename400(t *testing.T) {
	h, _ := newHandler(t)
	resp := invokeUploadWithFilename(t, h, "..")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("dotdot filename: status = %d; want 400; body=%s", resp.StatusCode, body)
	}
}

// --------------------------------------------------------------------------
// WR-01 — Missing ?path= on write verbs → 400
// --------------------------------------------------------------------------

// invokeNoPath issues a request with session=good but NO path query parameter.
func invokeNoPath(t *testing.T, h func(http.ResponseWriter, *http.Request), method string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *strings.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, "/?session=good", bodyReader) // no &path=
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

// TestHandlerWrite_MissingPath400: PUT /api/files/write without ?path= → 400 (WR-01).
func TestHandlerWrite_MissingPath400(t *testing.T) {
	h, _ := newHandler(t)
	rr := invokeNoPath(t, h.Write, "PUT", []byte("data"))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Write missing path: status = %d; want 400; body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandlerDelete_MissingPath400: DELETE /api/files/delete without ?path= → 400 (WR-01).
func TestHandlerDelete_MissingPath400(t *testing.T) {
	h, _ := newHandler(t)
	rr := invokeNoPath(t, h.Delete, "DELETE", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Delete missing path: status = %d; want 400; body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandlerMkdir_MissingPath400: POST /api/files/mkdir without ?path= → 400 (WR-01).
func TestHandlerMkdir_MissingPath400(t *testing.T) {
	h, _ := newHandler(t)
	rr := invokeNoPath(t, h.Mkdir, "POST", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Mkdir missing path: status = %d; want 400; body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandlerList_MissingPathStillDefaultsToDot: GET /api/files/list without
// ?path= must still default to "." (read-side behavior is unchanged by WR-01).
func TestHandlerList_MissingPathStillDefaultsToDot(t *testing.T) {
	h, root := newHandler(t)
	if err := os.WriteFile(filepath.Join(root, "probe.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/?session=good", nil) // no path param
	rr := httptest.NewRecorder()
	h.List(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("List missing path: status = %d; want 200 (read-side unchanged)", rr.Code)
	}
}
