package tui

import (
	"context"
	"encoding/json"
	"errors"
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
