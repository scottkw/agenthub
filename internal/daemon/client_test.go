package daemon

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClientHealth(t *testing.T) {
	_, client, _ := testDaemon(t)
	if err := client.Health(); err != nil {
		t.Errorf("Health: unexpected error: %v", err)
	}
}

func TestClientListSessions(t *testing.T) {
	_, client, _ := testDaemon(t)
	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if sessions == nil {
		t.Fatal("ListSessions returned nil, want empty slice")
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestClientCreateSession(t *testing.T) {
	_, client, _ := testDaemon(t)
	id, err := client.CreateSession("cat", "test-tab", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession returned empty ID")
	}
	t.Cleanup(func() { client.KillSession(id) })
}

func TestClientKillSession(t *testing.T) {
	_, client, _ := testDaemon(t)
	id, err := client.CreateSession("cat", "kill-me", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := client.KillSession(id); err != nil {
		t.Errorf("KillSession: %v", err)
	}
}

func TestClientRenameSession(t *testing.T) {
	_, client, _ := testDaemon(t)
	id, err := client.CreateSession("cat", "original", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { client.KillSession(id) })

	if err := client.RenameSession(id, "renamed"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}

	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions after rename: %v", err)
	}
	var found SessionInfo
	for _, s := range sessions {
		if s.ID == id {
			found = s
			break
		}
	}
	if found.ID == "" {
		t.Fatal("session not found after rename")
	}
	if found.Name != "renamed" {
		t.Errorf("name after rename: got %q, want %q", found.Name, "renamed")
	}
}

func TestClientGetSessionStatus(t *testing.T) {
	_, client, _ := testDaemon(t)
	id, err := client.CreateSession("cat", "status-tab", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { client.KillSession(id) })

	status, err := client.GetSessionStatus(id)
	if err != nil {
		t.Fatalf("GetSessionStatus: %v", err)
	}
	valid := map[string]bool{"running": true, "waiting": true, "idle": true, "errored": true}
	if !valid[status] {
		t.Errorf("GetSessionStatus returned invalid status %q", status)
	}
}

func TestClientGetRelayPort(t *testing.T) {
	api, client, _ := testDaemon(t)
	port, err := api.StartRelay()
	if err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	if port <= 0 {
		t.Fatalf("StartRelay returned invalid port: %d", port)
	}

	got, err := client.GetRelayPort()
	if err != nil {
		t.Fatalf("GetRelayPort: %v", err)
	}
	if got <= 0 {
		t.Errorf("GetRelayPort: want > 0, got %d", got)
	}
	if got != port {
		t.Errorf("GetRelayPort: got %d, want %d", got, port)
	}
}

func TestClientWebServerStatus(t *testing.T) {
	_, client, _ := testDaemon(t)
	resp, err := client.GetWebServerStatus()
	if err != nil {
		t.Fatalf("GetWebServerStatus: %v", err)
	}
	if resp.Running {
		t.Errorf("GetWebServerStatus: want Running=false, got true")
	}
}

func TestShutdownDaemon(t *testing.T) {
	// Create a test HTTP server that records the shutdown call
	var shutdownCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/shutdown" {
			shutdownCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	// Create client pointing to test server (TCP, not Unix socket)
	client := &DaemonClient{
		http: srv.Client(),
		base: srv.URL,
	}
	err := client.ShutdownDaemon()
	if err != nil {
		t.Fatalf("ShutdownDaemon returned error: %v", err)
	}
	if !shutdownCalled {
		t.Error("expected /shutdown endpoint to be called")
	}
}

// TestClientFullLifecycle tests the full round-trip: create -> list -> rename -> list -> kill -> list.
func TestClientFullLifecycle(t *testing.T) {
	_, client, _ := testDaemon(t)

	// Create.
	id, err := client.CreateSession("cat", "tab-one", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// List — should have 1 session.
	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Name != "tab-one" {
		t.Errorf("name: got %q, want %q", sessions[0].Name, "tab-one")
	}

	// Rename.
	if err := client.RenameSession(id, "tab-renamed"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}

	// List — should show new name.
	sessions, err = client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions after rename: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Name != "tab-renamed" {
		t.Errorf("after rename: got %+v", sessions)
	}

	// Kill.
	if err := client.KillSession(id); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// Give the session a moment to be removed.
	time.Sleep(50 * time.Millisecond)

	// List — should be empty.
	sessions, err = client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions after kill: %v", err)
	}
	for _, s := range sessions {
		if s.ID == id {
			t.Errorf("killed session %q still in list", id)
		}
	}
}

// -------------------------------------------------------------------------
// Phase 118 / Plan 05: DaemonClient.ListFiles / StatFile / ReadFile / HeadFile
// integration tests against a real running daemon socket.
//
// Each test reuses newFilesAPI (api_test.go) to spin up a tempdir-backed
// session, then constructs a DaemonClient on the same socket and calls the
// new methods. The seams under test are:
//
//   - URL construction (url.Values.Encode, path encoding)
//   - HTTP method (GET vs HEAD)
//   - Status-code → typed-error mapping (413 / 403 / 404 → non-nil error)
//   - JSON decode of files.FileEntry / files.FileListResponse over the wire
// -------------------------------------------------------------------------

func TestDaemonClient_ListFiles(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"a.txt", "b.go", "c.md"} {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte("x"), 0600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	entries, truncated, err := c.ListFiles(context.Background(), sid, ".")
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if truncated {
		t.Errorf("truncated = true; want false for 3-entry dir")
	}
	if len(entries) < 3 {
		t.Errorf("entries len = %d, want >= 3; got %+v", len(entries), entries)
	}
	var foundGo bool
	for _, e := range entries {
		if e.Name == "b.go" {
			foundGo = true
		}
	}
	if !foundGo {
		t.Errorf("expected b.go in listing; got %+v", entries)
	}
}

func TestDaemonClient_StatFile(t *testing.T) {
	tmp := t.TempDir()
	payload := []byte("hello world")
	if err := os.WriteFile(filepath.Join(tmp, "hello.txt"), payload, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	entry, err := c.StatFile(context.Background(), sid, "hello.txt")
	if err != nil {
		t.Fatalf("StatFile: %v", err)
	}
	if entry.Name != "hello.txt" {
		t.Errorf("Name = %q, want 'hello.txt'", entry.Name)
	}
	if entry.Size != int64(len(payload)) {
		t.Errorf("Size = %d, want %d", entry.Size, len(payload))
	}
	if entry.IsDir {
		t.Errorf("IsDir = true; want false")
	}
}

func TestDaemonClient_ReadFile(t *testing.T) {
	tmp := t.TempDir()
	payload := []byte("some bytes here")
	if err := os.WriteFile(filepath.Join(tmp, "data.bin"), payload, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	data, ct, err := c.ReadFile(context.Background(), sid, "data.bin")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Errorf("ReadFile data = %q, want %q", string(data), string(payload))
	}
	if ct == "" {
		t.Errorf("ReadFile content-type empty; want a non-empty value (sniffed)")
	}
}

func TestDaemonClient_ReadFile_OverCapReturnsError(t *testing.T) {
	tmp := t.TempDir()
	// 5 MiB + 1 byte to trip the maxPreviewBytes cap.
	oversize := bytes.Repeat([]byte{'A'}, 5*1024*1024+1)
	if err := os.WriteFile(filepath.Join(tmp, "big.bin"), oversize, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	_, _, err := c.ReadFile(context.Background(), sid, "big.bin")
	if err == nil {
		t.Fatalf("ReadFile on >5MiB returned nil err; want non-nil with 413 or 'too large'")
	}
	msg := err.Error()
	if !strings.Contains(msg, "413") && !strings.Contains(strings.ToLower(msg), "too large") {
		t.Errorf("ReadFile oversize err = %q; want substring '413' or 'too large'", msg)
	}
}

func TestDaemonClient_HeadFile(t *testing.T) {
	tmp := t.TempDir()
	payload := bytes.Repeat([]byte{'B'}, 250)
	p := filepath.Join(tmp, "p.bin")
	if err := os.WriteFile(p, payload, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	size, ct, mtime, err := c.HeadFile(context.Background(), sid, "p.bin")
	if err != nil {
		t.Fatalf("HeadFile: %v", err)
	}
	if size != int64(len(payload)) {
		t.Errorf("HeadFile size = %d, want %d", size, len(payload))
	}
	if ct == "" {
		t.Errorf("HeadFile content-type empty; want a non-empty value")
	}
	if mtime.IsZero() {
		t.Errorf("HeadFile mtime IsZero; want a real timestamp")
	}
}

func TestDaemonClient_ListFiles_TraversalReturns403Error(t *testing.T) {
	_, sock, sid := newFilesAPI(t, t.TempDir())
	c := NewDaemonClient(sock)

	_, _, err := c.ListFiles(context.Background(), sid, "../../escape")
	if err == nil {
		t.Fatalf("ListFiles traversal returned nil err; want non-nil with 403")
	}
	msg := err.Error()
	if !strings.Contains(msg, "403") && !strings.Contains(strings.ToLower(msg), "access denied") {
		t.Errorf("ListFiles traversal err = %q; want substring '403' or 'access denied'", msg)
	}
}

// -------------------------------------------------------------------------
// Phase 123 / Plan 04: DaemonClient write method round-trip tests (FSW-09).
//
// Each test spins up a tempdir-backed API via newFilesAPI and exercises the
// five new DaemonClient write methods against the real Plan-03 routes:
//
//   WriteFile   → PUT  /api/files/write
//   UploadFile  → POST /api/files/upload  (multipart)
//   DeleteFile  → DELETE /api/files/delete
//   RenameFile  → POST /api/files/rename  (JSON body)
//   MkdirFile   → POST /api/files/mkdir
//
// Error and cancellation behaviour are also covered per the plan's
// acceptance criteria.
// -------------------------------------------------------------------------

func TestDaemonClientWrite_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	content := []byte("hello from WriteFile")
	resp, err := c.WriteFile(context.Background(), sid, "roundtrip.txt", content)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if resp.Path != "roundtrip.txt" {
		t.Errorf("WriteFile resp.Path = %q, want %q", resp.Path, "roundtrip.txt")
	}
	if resp.Size != int64(len(content)) {
		t.Errorf("WriteFile resp.Size = %d, want %d", resp.Size, len(content))
	}

	// Follow-up read must return identical bytes.
	got, _, err := c.ReadFile(context.Background(), sid, "roundtrip.txt")
	if err != nil {
		t.Fatalf("ReadFile after Write: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("ReadFile after Write = %q, want %q", string(got), string(content))
	}
}

func TestDaemonClientUpload_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	content := []byte("uploaded bytes content")
	resp, err := c.UploadFile(context.Background(), sid, ".", "upload.txt", content)
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if resp.Size != int64(len(content)) {
		t.Errorf("UploadFile resp.Size = %d, want %d", resp.Size, len(content))
	}

	got, _, err := c.ReadFile(context.Background(), sid, "upload.txt")
	if err != nil {
		t.Fatalf("ReadFile after Upload: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("ReadFile after Upload = %q, want %q", string(got), string(content))
	}
}

func TestDaemonClientDelete_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	// Pre-populate a file to delete.
	if err := os.WriteFile(filepath.Join(tmp, "todelete.txt"), []byte("bye"), 0600); err != nil {
		t.Fatalf("WriteFile seed: %v", err)
	}
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	opResp, err := c.DeleteFile(context.Background(), sid, "todelete.txt")
	if err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if !opResp.OK {
		t.Errorf("DeleteFile resp.OK = false, want true")
	}

	// Stat should now return a 404 error.
	_, err = c.StatFile(context.Background(), sid, "todelete.txt")
	if err == nil {
		t.Fatal("StatFile after Delete returned nil err; want non-nil (file gone)")
	}
}

func TestDaemonClientRename_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "old.txt"), []byte("rename me"), 0600); err != nil {
		t.Fatalf("WriteFile seed: %v", err)
	}
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	opResp, err := c.RenameFile(context.Background(), sid, "old.txt", "new.txt")
	if err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if !opResp.OK {
		t.Errorf("RenameFile resp.OK = false, want true")
	}
	if opResp.Path != "new.txt" {
		t.Errorf("RenameFile resp.Path = %q, want %q", opResp.Path, "new.txt")
	}

	// new.txt must be readable; old.txt must be gone.
	got, _, err := c.ReadFile(context.Background(), sid, "new.txt")
	if err != nil {
		t.Fatalf("ReadFile new.txt after Rename: %v", err)
	}
	if string(got) != "rename me" {
		t.Errorf("ReadFile new.txt = %q, want %q", string(got), "rename me")
	}
	_, err = c.StatFile(context.Background(), sid, "old.txt")
	if err == nil {
		t.Fatal("StatFile old.txt after Rename returned nil err; want non-nil (file moved)")
	}
}

func TestDaemonClientMkdir_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	_, sock, sid := newFilesAPI(t, tmp)
	c := NewDaemonClient(sock)

	opResp, err := c.MkdirFile(context.Background(), sid, "newdir/subdir")
	if err != nil {
		t.Fatalf("MkdirFile: %v", err)
	}
	if !opResp.OK {
		t.Errorf("MkdirFile resp.OK = false, want true")
	}

	// The directory must appear in the listing.
	entries, _, err := c.ListFiles(context.Background(), sid, ".")
	if err != nil {
		t.Fatalf("ListFiles after Mkdir: %v", err)
	}
	var found bool
	for _, e := range entries {
		if e.Name == "newdir" && e.IsDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("newdir not found in listing after Mkdir; entries = %+v", entries)
	}
}

func TestDaemonClientWrite_NonOKError(t *testing.T) {
	_, sock, sid := newFilesAPI(t, t.TempDir())
	c := NewDaemonClient(sock)

	// Traversal path triggers 403 from the server — surfaces as a typed error.
	_, err := c.WriteFile(context.Background(), sid, "../../escape.txt", []byte("x"))
	if err == nil {
		t.Fatal("WriteFile traversal returned nil err; want non-nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "403") && !strings.Contains(strings.ToLower(msg), "access denied") {
		t.Errorf("WriteFile traversal err = %q; want substring '403' or 'access denied'", msg)
	}
}

func TestDaemonClientWrite_ContextCancel(t *testing.T) {
	// Use a cancelled context — the HTTP dial should abort before any bytes fly.
	_, sock, sid := newFilesAPI(t, t.TempDir())
	c := NewDaemonClient(sock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.WriteFile(ctx, sid, "noop.txt", []byte("x"))
	if err == nil {
		t.Fatal("WriteFile with cancelled ctx returned nil err; want non-nil")
	}
}
