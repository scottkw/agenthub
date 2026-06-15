package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/files"
)

// TestRemoteFilesClient_New_StripsTrailingSlash verifies the constructor
// normalises the baseURL so callers can pass either shape (Behavior 1).
func TestRemoteFilesClient_New_StripsTrailingSlash(t *testing.T) {
	c := NewRemoteFilesClient("https://peer.example:9443/", "captok")
	if c == nil {
		t.Fatal("NewRemoteFilesClient returned nil")
	}
	if c.baseURL != "https://peer.example:9443" {
		t.Errorf("expected baseURL stripped, got %q", c.baseURL)
	}
}

// TestRemoteFilesClient_ListFiles_RoundTrip exercises the canonical happy path
// against an httptest.NewTLSServer fixture serving /api/files/list (Behavior 2).
func TestRemoteFilesClient_ListFiles_RoundTrip(t *testing.T) {
	const wantCap = "test-cap-token"
	expected := files.FileListResponse{
		Entries: []files.FileEntry{
			{Name: "a.txt", IsDir: false, MIME: "text/plain"},
			{Name: "sub", IsDir: true},
		},
		Truncated: false,
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/list" {
			t.Errorf("expected path /api/files/list, got %q", r.URL.Path)
		}
		if r.URL.Query().Get("session") != "sid" {
			t.Errorf("expected session=sid, got %q", r.URL.Query().Get("session"))
		}
		if r.URL.Query().Get("cap") != wantCap {
			t.Errorf("expected cap=%q, got %q", wantCap, r.URL.Query().Get("cap"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := newRemoteFilesClientWithHTTP(srv.URL, wantCap, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	entries, trunc, err := c.ListFiles(ctx, "sid", "rel")
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if trunc {
		t.Error("expected truncated=false")
	}
	if len(entries) != 2 || entries[0].Name != "a.txt" || !entries[1].IsDir {
		t.Errorf("entries mismatch: %+v", entries)
	}
}

// TestRemoteFilesClient_StatFile_RoundTrip mirrors ListFiles for the single-
// entry /stat shape (Behavior 3).
func TestRemoteFilesClient_StatFile_RoundTrip(t *testing.T) {
	expected := files.FileEntry{Name: "a.txt", Size: 5, MIME: "text/plain"}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/stat" {
			t.Errorf("expected /api/files/stat, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := c.StatFile(ctx, "sid", "a.txt")
	if err != nil {
		t.Fatalf("StatFile: %v", err)
	}
	if got.Name != "a.txt" || got.Size != 5 || got.MIME != "text/plain" {
		t.Errorf("StatFile result mismatch: %+v", got)
	}
}

// TestRemoteFilesClient_ReadFile_RoundTrip exercises the read happy path —
// body bytes + Content-Type round-trip (Behavior 4).
func TestRemoteFilesClient_ReadFile_RoundTrip(t *testing.T) {
	body := []byte("hello world")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/read" {
			t.Errorf("expected /api/files/read, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data, mime, err := c.ReadFile(ctx, "sid", "a.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("data mismatch: got %q", data)
	}
	if !strings.HasPrefix(mime, "text/plain") {
		t.Errorf("mime mismatch: got %q", mime)
	}
}

// TestRemoteFilesClient_HeadFile_RoundTrip verifies Content-Length +
// Content-Type + Last-Modified parsing (Behavior 5). HTTP HEAD must NOT
// receive a body — the test handler writes only headers.
func TestRemoteFilesClient_HeadFile_RoundTrip(t *testing.T) {
	when := time.Date(2025, 5, 20, 12, 0, 0, 0, time.UTC)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/markdown")
		w.Header().Set("Content-Length", "1234")
		w.Header().Set("Last-Modified", when.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	size, mime, mtime, err := c.HeadFile(ctx, "sid", "b.md")
	if err != nil {
		t.Fatalf("HeadFile: %v", err)
	}
	if size != 1234 {
		t.Errorf("expected size=1234, got %d", size)
	}
	if mime != "text/markdown" {
		t.Errorf("expected mime=text/markdown, got %q", mime)
	}
	if !mtime.Equal(when) {
		t.Errorf("expected mtime=%v, got %v", when, mtime)
	}
}

// TestRemoteFilesClient_Status401_NoCapLeak proves the CAP-LEAK invariant
// (T-122-04-01): a 401 upstream response surfaces an error that contains
// "401" but never the cap token (Behavior 6).
func TestRemoteFilesClient_Status401_NoCapLeak(t *testing.T) {
	const sensitiveCap = "SUPER-SECRET-CAP-TOKEN"
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newRemoteFilesClientWithHTTP(srv.URL, sensitiveCap, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := c.ListFiles(ctx, "sid", ".")
	if err == nil {
		t.Fatal("expected error on 401")
	}
	msg := err.Error()
	if !strings.Contains(msg, "401") {
		t.Errorf("expected error to contain status code 401, got %q", msg)
	}
	if strings.Contains(msg, sensitiveCap) {
		t.Errorf("CAP-LEAK INVARIANT VIOLATED — error contains cap token: %q", msg)
	}
}

// TestRemoteFilesClient_Status403 verifies error surface for permission
// denied (Behavior 7).
func TestRemoteFilesClient_Status403(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()
	c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
	_, _, err := c.ListFiles(context.Background(), "sid", ".")
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error containing 403, got %v", err)
	}
}

// TestRemoteFilesClient_Status413 verifies over-cap error preserves the
// Phase 121 classification path (Behavior 8).
func TestRemoteFilesClient_Status413(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
	}))
	defer srv.Close()
	c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
	_, _, err := c.ReadFile(context.Background(), "sid", "big.bin")
	if err == nil || !strings.Contains(err.Error(), "413") {
		t.Errorf("expected error containing 413, got %v", err)
	}
}

// TestRemoteFilesClient_CtxCancel verifies context cancellation propagates
// to the request and surfaces as a wrapped context.Canceled (Behavior 9).
func TestRemoteFilesClient_CtxCancel(t *testing.T) {
	// Server blocks until the test cancels the context.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()
	c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, _, err := c.ListFiles(ctx, "sid", ".")
	if err == nil {
		t.Fatal("expected error on ctx cancel, got nil")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected wrapped context.Canceled, got %v", err)
	}
}

// TestRemoteFilesClient_UsesURLValues asserts via source-grep that URL
// construction goes through net/url.Values (RESEARCH §Don't Hand-Roll)
// (Behavior 10).
func TestRemoteFilesClient_UsesURLValues(t *testing.T) {
	src, err := os.ReadFile("remote_files_client.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if !strings.Contains(string(src), "url.Values") {
		t.Error("expected remote_files_client.go to use url.Values for URL construction")
	}
}

// TestRemoteFilesClient_EmptyRelPathNormalisedToDot mirrors DaemonClient
// behavior (Behavior 11).
func TestRemoteFilesClient_EmptyRelPathNormalisedToDot(t *testing.T) {
	var observedPath string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedPath = r.URL.Query().Get("path")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"entries":[],"truncated":false}`))
	}))
	defer srv.Close()
	c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
	_, _, err := c.ListFiles(context.Background(), "sid", "")
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if observedPath != "." {
		t.Errorf("expected empty relPath normalised to '.', got %q", observedPath)
	}
}

// TestRemoteFilesClient_SatisfiesInterface is a compile-time guard via source
// grep that the var-binding line is present (Behavior 12). The actual
// type-system guarantee is already covered by the var _ FilesClient = ...
// declaration in remote_files_client.go.
func TestRemoteFilesClient_SatisfiesInterface(t *testing.T) {
	src, err := os.ReadFile("remote_files_client.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if !strings.Contains(string(src), "var _ FilesClient = (*RemoteFilesClient)(nil)") {
		t.Error("expected compile-time guard `var _ FilesClient = (*RemoteFilesClient)(nil)`")
	}
}

// TestRemoteFilesClient_TLSMinVersion12 source-greps for the TLS pin
// (Behavior 13).
func TestRemoteFilesClient_TLSMinVersion12(t *testing.T) {
	src, err := os.ReadFile("remote_files_client.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if !strings.Contains(string(src), "MinVersion: tls.VersionTLS12") {
		t.Error("expected `MinVersion: tls.VersionTLS12` in remote_files_client.go")
	}
}

// TestRedactCapFromURL verifies the defense-in-depth helper strips the cap
// query parameter without affecting other URL components.
func TestRedactCapFromURL(t *testing.T) {
	in := "https://peer.example:9443/api/files/list?session=sid&path=.&cap=secret"
	got := redactCapFromURL(in)
	if strings.Contains(got, "secret") {
		t.Errorf("redactCapFromURL leaked cap: %q", got)
	}
	if !strings.Contains(got, "session=sid") || !strings.Contains(got, "path=.") {
		t.Errorf("redactCapFromURL clobbered non-cap params: %q", got)
	}
}

// TestRemoteFilesClient_Write verifies WriteFile sends PUT with
// application/octet-stream body and decodes the FileWriteResponse (TUIW-01).
func TestRemoteFilesClient_Write(t *testing.T) {
	const wantCap = "write-cap-token"
	payload := []byte("hello, world")
	want := files.FileWriteResponse{Path: "notes.txt", Size: int64(len(payload))}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/write" {
			t.Errorf("expected path /api/files/write, got %q", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Query().Get("session") != "sid1" {
			t.Errorf("expected session=sid1, got %q", r.URL.Query().Get("session"))
		}
		if r.URL.Query().Get("path") != "notes.txt" {
			t.Errorf("expected path=notes.txt, got %q", r.URL.Query().Get("path"))
		}
		if r.URL.Query().Get("cap") != wantCap {
			t.Errorf("expected cap=%q, got %q", wantCap, r.URL.Query().Get("cap"))
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/octet-stream" {
			t.Errorf("expected Content-Type application/octet-stream, got %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Equal(body, payload) {
			t.Errorf("body mismatch: got %q, want %q", body, payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewRemoteFilesClientForTest(srv.URL, wantCap, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := c.WriteFile(ctx, "sid1", "notes.txt", payload)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if got.Path != want.Path {
		t.Errorf("WriteFile Path = %q, want %q", got.Path, want.Path)
	}
	if got.Size != want.Size {
		t.Errorf("WriteFile Size = %d, want %d", got.Size, want.Size)
	}
}

// TestRemoteFilesClient_Delete verifies DeleteFile sends DELETE with nil body
// and decodes FileOpResponse (TUIW-01).
func TestRemoteFilesClient_Delete(t *testing.T) {
	const wantCap = "delete-cap-token"
	want := files.FileOpResponse{OK: true}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/delete" {
			t.Errorf("expected path /api/files/delete, got %q", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Query().Get("session") != "sid2" {
			t.Errorf("expected session=sid2, got %q", r.URL.Query().Get("session"))
		}
		if r.URL.Query().Get("path") != "old.txt" {
			t.Errorf("expected path=old.txt, got %q", r.URL.Query().Get("path"))
		}
		if r.URL.Query().Get("cap") != wantCap {
			t.Errorf("expected cap=%q, got %q", wantCap, r.URL.Query().Get("cap"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewRemoteFilesClientForTest(srv.URL, wantCap, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := c.DeleteFile(ctx, "sid2", "old.txt")
	if err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if !got.OK {
		t.Errorf("DeleteFile OK = false, want true")
	}
}

// TestRemoteFilesClient_Rename verifies RenameFile sends POST with JSON body
// {oldRel,newRel} and decodes FileOpResponse (TUIW-01).
func TestRemoteFilesClient_Rename(t *testing.T) {
	const wantCap = "rename-cap-token"
	want := files.FileOpResponse{OK: true}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/rename" {
			t.Errorf("expected path /api/files/rename, got %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Query().Get("cap") != wantCap {
			t.Errorf("expected cap=%q, got %q", wantCap, r.URL.Query().Get("cap"))
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		var body struct {
			OldRel string `json:"oldRel"`
			NewRel string `json:"newRel"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode rename body: %v", err)
		}
		if body.OldRel != "old.txt" {
			t.Errorf("oldRel = %q, want old.txt", body.OldRel)
		}
		if body.NewRel != "new.txt" {
			t.Errorf("newRel = %q, want new.txt", body.NewRel)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewRemoteFilesClientForTest(srv.URL, wantCap, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := c.RenameFile(ctx, "sid3", "old.txt", "new.txt")
	if err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if !got.OK {
		t.Errorf("RenameFile OK = false, want true")
	}
}

// TestRemoteFilesClient_Mkdir verifies MkdirFile sends POST with nil body and
// decodes FileOpResponse (TUIW-01).
func TestRemoteFilesClient_Mkdir(t *testing.T) {
	const wantCap = "mkdir-cap-token"
	want := files.FileOpResponse{OK: true}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/mkdir" {
			t.Errorf("expected path /api/files/mkdir, got %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Query().Get("session") != "sid4" {
			t.Errorf("expected session=sid4, got %q", r.URL.Query().Get("session"))
		}
		if r.URL.Query().Get("path") != "newdir" {
			t.Errorf("expected path=newdir, got %q", r.URL.Query().Get("path"))
		}
		if r.URL.Query().Get("cap") != wantCap {
			t.Errorf("expected cap=%q, got %q", wantCap, r.URL.Query().Get("cap"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewRemoteFilesClientForTest(srv.URL, wantCap, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := c.MkdirFile(ctx, "sid4", "newdir")
	if err != nil {
		t.Fatalf("MkdirFile: %v", err)
	}
	if !got.OK {
		t.Errorf("MkdirFile OK = false, want true")
	}
}

// TestRemoteFilesClient_405_ErrRemotePeerNoWriteSupport asserts that all 4 write
// methods (WriteFile, DeleteFile, RenameFile, MkdirFile) return the sentinel
// ErrRemotePeerNoWriteSupport (errors.Is match) on a 405 upstream response,
// that the sentinel message is verbatim SC3, and that a non-405 error (500)
// still returns the existing generic error. (RMW-04 RED gate)
func TestRemoteFilesClient_405_ErrRemotePeerNoWriteSupport(t *testing.T) {
	const verbatimMsg = "The remote session is running an older version of AgentHub that does not support file writes."

	make405Srv := func(t *testing.T) *httptest.Server {
		t.Helper()
		return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}))
	}
	make500Srv := func(t *testing.T) *httptest.Server {
		t.Helper()
		return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
	}

	ctx := context.Background()

	t.Run("WriteFile 405 returns ErrRemotePeerNoWriteSupport", func(t *testing.T) {
		srv := make405Srv(t)
		defer srv.Close()
		c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
		_, err := c.WriteFile(ctx, "sid", "f.txt", []byte("data"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrRemotePeerNoWriteSupport) {
			t.Errorf("expected ErrRemotePeerNoWriteSupport, got %v", err)
		}
		if err.Error() != verbatimMsg {
			t.Errorf("message mismatch:\n  got:  %q\n  want: %q", err.Error(), verbatimMsg)
		}
	})

	t.Run("DeleteFile 405 returns ErrRemotePeerNoWriteSupport", func(t *testing.T) {
		srv := make405Srv(t)
		defer srv.Close()
		c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
		_, err := c.DeleteFile(ctx, "sid", "f.txt")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrRemotePeerNoWriteSupport) {
			t.Errorf("expected ErrRemotePeerNoWriteSupport, got %v", err)
		}
	})

	t.Run("RenameFile 405 returns ErrRemotePeerNoWriteSupport", func(t *testing.T) {
		srv := make405Srv(t)
		defer srv.Close()
		c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
		_, err := c.RenameFile(ctx, "sid", "old.txt", "new.txt")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrRemotePeerNoWriteSupport) {
			t.Errorf("expected ErrRemotePeerNoWriteSupport, got %v", err)
		}
	})

	t.Run("MkdirFile 405 returns ErrRemotePeerNoWriteSupport", func(t *testing.T) {
		srv := make405Srv(t)
		defer srv.Close()
		c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
		_, err := c.MkdirFile(ctx, "sid", "newdir")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrRemotePeerNoWriteSupport) {
			t.Errorf("expected ErrRemotePeerNoWriteSupport, got %v", err)
		}
	})

	t.Run("WriteFile 500 returns generic error (no regression)", func(t *testing.T) {
		srv := make500Srv(t)
		defer srv.Close()
		c := newRemoteFilesClientWithHTTP(srv.URL, "tok", srv.Client())
		_, err := c.WriteFile(ctx, "sid", "f.txt", []byte("data"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errors.Is(err, ErrRemotePeerNoWriteSupport) {
			t.Error("500 should NOT return ErrRemotePeerNoWriteSupport")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("expected generic error to contain '500', got %q", err.Error())
		}
	})

	t.Run("405 error string has no cap token", func(t *testing.T) {
		const sensitiveCap = "SUPER-SECRET-WRITE-CAP-405"
		srv := make405Srv(t)
		defer srv.Close()
		c := newRemoteFilesClientWithHTTP(srv.URL, sensitiveCap, srv.Client())
		_, err := c.WriteFile(ctx, "sid", "f.txt", []byte("data"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if strings.Contains(err.Error(), sensitiveCap) {
			t.Errorf("CAP-LEAK on 405 — error contains cap token: %q", err.Error())
		}
	})
}

// TestRemoteFilesClient_WriteCapLeak proves the CAP-LEAK invariant (T-126-01):
// on a non-200 response from a write method, the returned error string must
// contain the status code but NOT the cap token.
func TestRemoteFilesClient_WriteCapLeak(t *testing.T) {
	const sensitiveCap = "SUPER-SECRET-WRITE-CAP"

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	c := NewRemoteFilesClientForTest(srv.URL, sensitiveCap, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.WriteFile(ctx, "sid", "file.txt", []byte("data"))
	if err == nil {
		t.Fatal("expected error on 403, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "403") {
		t.Errorf("expected error to contain status code 403, got %q", msg)
	}
	if strings.Contains(msg, sensitiveCap) {
		t.Errorf("CAP-LEAK INVARIANT VIOLATED — error contains cap token: %q", msg)
	}
}
