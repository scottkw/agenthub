package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// mockRemoteWebserver returns an httptest.NewTLSServer that mimics the remote
// peer's /api/files/{list,stat,read} endpoints. It validates the ?cap=<token>
// query parameter and returns canned responses keyed on the cap:
//
//   cap="ok-cap"     → 200 with operation-specific body
//   cap="viewer-cap" → 403 (mirrors Phase 119's requireFilesRead when the
//                      cap is valid but lacks files.read perm)
//   anything else    → 401 (cap rejected entirely)
//
// The returned cleanup closure must be deferred by the test.
func mockRemoteWebserver(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	mux := http.NewServeMux()
	handle := func(op string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cap := r.URL.Query().Get("cap")
			switch cap {
			case "ok-cap":
				// echo the path query so tests can assert pass-through
				switch op {
				case "list":
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"entries":   []map[string]any{{"name": "hello.txt"}},
						"truncated": false,
						"echoPath":  r.URL.Query().Get("path"),
					})
				case "stat":
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"name":     r.URL.Query().Get("path"),
						"size":     int64(42),
						"echoPath": r.URL.Query().Get("path"),
					})
				case "read":
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					w.Header().Set("Content-Length", "11")
					w.Header().Set("Last-Modified", "Wed, 21 Oct 2026 07:28:00 GMT")
					w.WriteHeader(http.StatusOK)
					if r.Method == http.MethodHead {
						return
					}
					_, _ = w.Write([]byte("hello world"))
				}
			case "viewer-cap":
				http.Error(w, "files.read denied", http.StatusForbidden)
			default:
				http.Error(w, "cap invalid", http.StatusUnauthorized)
			}
		}
	}
	mux.HandleFunc("GET /api/files/list", handle("list"))
	mux.HandleFunc("GET /api/files/stat", handle("stat"))
	mux.HandleFunc("GET /api/files/read", handle("read"))
	mux.HandleFunc("HEAD /api/files/read", handle("read"))

	srv := httptest.NewTLSServer(mux)
	return srv, srv.Close
}

// proxyTestAPI builds a minimal *API + httptest server wired to the proxy
// routes. The remoteFilesClientForTest is injected from `upstream.Client()`
// so the proxy trusts the mock's self-signed cert.
func proxyTestAPI(t *testing.T, upstream *httptest.Server) (*API, *httptest.Server) {
	t.Helper()
	engine := NewSessionEngine()
	engine.configDir = t.TempDir()
	api := NewAPI(engine)
	api.remoteFilesClientForTest = upstream.Client()
	srv := httptest.NewServer(api.mux)
	t.Cleanup(srv.Close)
	return api, srv
}

func TestRemoteFiles_ListRoundTrip(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)

	if err := api.remoteCaps.Put("sid-1", upstream.URL, "ok-cap"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	resp, err := http.Get(srv.URL + "/api/files/remote/sid-1/list?path=.")
	if err != nil {
		t.Fatalf("GET proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d; body=%s", resp.StatusCode, body)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got["echoPath"] != "." {
		t.Fatalf("path not forwarded; echoPath=%v", got["echoPath"])
	}
}

func TestRemoteFiles_NoCapRegistered_Returns404(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	_, srv := proxyTestAPI(t, upstream)

	resp, err := http.Get(srv.URL + "/api/files/remote/unknown-sid/list?path=.")
	if err != nil {
		t.Fatalf("GET proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404; got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !strings.Contains(body["error"], "no cap registered") {
		t.Fatalf("expected error marker; got %q", body["error"])
	}
}

func TestRemoteFiles_Upstream403_PassesThrough(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)

	_ = api.remoteCaps.Put("sid-1", upstream.URL, "viewer-cap")
	resp, err := http.Get(srv.URL + "/api/files/remote/sid-1/list?path=.")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 verbatim; got %d", resp.StatusCode)
	}
}

func TestRemoteFiles_Upstream401_PassesThrough(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)

	// "stale-cap" → mock returns 401 (any-cap-that-isn't-ok/viewer falls
	// into the default 401 branch).
	_ = api.remoteCaps.Put("sid-1", upstream.URL, "stale-cap")
	resp, err := http.Get(srv.URL + "/api/files/remote/sid-1/list?path=.")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 verbatim; got %d", resp.StatusCode)
	}
}

func TestRemoteFiles_StatRoundTrip(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)
	_ = api.remoteCaps.Put("sid-1", upstream.URL, "ok-cap")

	resp, err := http.Get(srv.URL + "/api/files/remote/sid-1/stat?path=foo.txt")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if got["echoPath"] != "foo.txt" {
		t.Fatalf("path not forwarded; echoPath=%v", got["echoPath"])
	}
}

func TestRemoteFiles_ReadRoundTrip(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)
	_ = api.remoteCaps.Put("sid-1", upstream.URL, "ok-cap")

	resp, err := http.Get(srv.URL + "/api/files/remote/sid-1/read?path=foo.txt")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello world" {
		t.Fatalf("body mismatch: %q", body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("Content-Type not forwarded; got %q", got)
	}
	if got := resp.Header.Get("Last-Modified"); got == "" {
		t.Fatal("Last-Modified header lost in transit")
	}
}

func TestRemoteFiles_HeadOnRead(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)
	_ = api.remoteCaps.Put("sid-1", upstream.URL, "ok-cap")

	req, _ := http.NewRequest(http.MethodHead, srv.URL+"/api/files/remote/sid-1/read?path=foo.txt", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Length") == "" {
		t.Fatal("Content-Length missing on HEAD")
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Fatalf("HEAD body should be empty; got %d bytes", len(body))
	}
}

func TestRemoteFiles_CallerCapStripped(t *testing.T) {
	// A malicious caller cannot smuggle a different cap by appending
	// ?cap=other to the proxy URL. Build an upstream that records which cap
	// it saw and assert it always sees the registered one.
	var sawCaps []string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/files/list", func(w http.ResponseWriter, r *http.Request) {
		sawCaps = append(sawCaps, r.URL.Query().Get("cap"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	})
	upstream := httptest.NewTLSServer(mux)
	defer upstream.Close()

	api, srv := proxyTestAPI(t, upstream)
	_ = api.remoteCaps.Put("sid-1", upstream.URL, "registered-cap")

	_, _ = http.Get(srv.URL + "/api/files/remote/sid-1/list?path=.&cap=injected-cap")

	if len(sawCaps) != 1 || sawCaps[0] != "registered-cap" {
		t.Fatalf("upstream saw caps %v; want only [registered-cap]", sawCaps)
	}
}

func TestRemoteFiles_PathQueryEncoding(t *testing.T) {
	// Path containing space and `#` must survive the proxy untouched.
	var sawPath string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/files/list", func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Query().Get("path")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	})
	upstream := httptest.NewTLSServer(mux)
	defer upstream.Close()

	api, srv := proxyTestAPI(t, upstream)
	_ = api.remoteCaps.Put("sid-1", upstream.URL, "ok-cap")

	want := "weird path with #hash.txt"
	u := fmt.Sprintf("%s/api/files/remote/sid-1/list?path=%s", srv.URL, urlEncode(want))
	_, _ = http.Get(u)

	if sawPath != want {
		t.Fatalf("path encoding lost; got %q want %q", sawPath, want)
	}
}

func TestRemoteFiles_TLSMinVersionInSource(t *testing.T) {
	// Source-grep guard: newRemoteFilesHTTPClient must pin TLS 1.2 minimum.
	src, err := readFile("remote_files.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(src, []byte("tls.VersionTLS12")) {
		t.Fatal("remote_files.go must reference tls.VersionTLS12 for outbound HTTPS")
	}
}

func TestRemoteFiles_405OnUnsupportedMethods(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)
	_ = api.remoteCaps.Put("sid-1", upstream.URL, "ok-cap")

	// POST/PUT/DELETE on a GET-only route must 405 (Go 1.22 mux semantics).
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req, _ := http.NewRequest(method, srv.URL+"/api/files/remote/sid-1/list", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("%s expected 405; got %d", method, resp.StatusCode)
		}
	}
}

// ---- POST /api/remote-files/caps -----------------------------------------

func TestRegisterRemoteCap_HappyPath(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)

	body := bytes.NewBufferString(`{"sessionId":"sid-1","baseUrl":"https://peer.example","capToken":"tok-abc"}`)
	resp, err := http.Post(srv.URL+"/api/remote-files/caps", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d; body=%s", resp.StatusCode, b)
	}
	var out map[string]bool
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if !out["ok"] {
		t.Fatalf("expected {ok:true}; got %v", out)
	}
	base, tok, ok := api.remoteCaps.Get("sid-1")
	if !ok || base != "https://peer.example" || tok != "tok-abc" {
		t.Fatalf("cap not stored: (%q, %q, %v)", base, tok, ok)
	}
}

func TestRegisterRemoteCap_MalformedBody(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	_, srv := proxyTestAPI(t, upstream)

	resp, err := http.Post(srv.URL+"/api/remote-files/caps", "application/json", bytes.NewBufferString("{not json"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on malformed body; got %d", resp.StatusCode)
	}
}

func TestRegisterRemoteCap_EmptyFieldsRejected(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	_, srv := proxyTestAPI(t, upstream)

	cases := []string{
		`{"sessionId":"","baseUrl":"https://p","capToken":"t"}`,
		`{"sessionId":"s","baseUrl":"","capToken":"t"}`,
		`{"sessionId":"s","baseUrl":"https://p","capToken":""}`,
	}
	for _, body := range cases {
		resp, _ := http.Post(srv.URL+"/api/remote-files/caps", "application/json", bytes.NewBufferString(body))
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("body=%s expected 400; got %d", body, resp.StatusCode)
		}
	}
}

func TestRegisterRemoteCap_IntegrationWithDaemonClient(t *testing.T) {
	// Confirm DaemonClient.RegisterRemoteCap (from 122-03's
	// client_remote_files.go) successfully posts to the new endpoint.
	// This is the integration test required by the recovery plan's
	// success criteria.
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)

	// Build a DaemonClient pointed at the test server. The client uses
	// doJSON which builds http://daemon{path}; override by constructing
	// directly with a custom socketPath/http.Client. Simplest: tunnel
	// through srv.URL via a tiny shim instead of the unix-socket dial.
	client := &DaemonClient{http: &http.Client{}, base: srv.URL}
	if err := client.RegisterRemoteCap("sid-integration", "https://peer.example", "tok-int"); err != nil {
		t.Fatalf("RegisterRemoteCap: %v", err)
	}
	base, tok, ok := api.remoteCaps.Get("sid-integration")
	if !ok || base != "https://peer.example" || tok != "tok-int" {
		t.Fatalf("cap not stored via DaemonClient: (%q, %q, %v)", base, tok, ok)
	}
}

// ---- helpers --------------------------------------------------------------

// urlEncode is a thin wrapper around url.QueryEscape so the test reads
// naturally (the call site is "encode this path before sending").
func urlEncode(s string) string {
	return url.QueryEscape(s)
}

// readFile reads a file from the package's source directory. Used by the
// TLS-min-version source-grep guard.
func readFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
