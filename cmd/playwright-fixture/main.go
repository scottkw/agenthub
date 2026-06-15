//go:build playwrightfixture

// Package main is a Playwright e2e test fixture for Phase 93 web parity specs.
//
// This binary boots an in-process AgentHub web server with:
//   - Self-signed TLS on 127.0.0.1 (random port)
//   - One pre-seeded session ("playwright-test-session") backed by an io.Pipe
//     PTY stub (no real shell — the page only needs to attach for the addon
//     load assertions; PTY data is irrelevant).
//   - A capability token signed with a fixed test key.
//   - SetPluginSettingsProvider wired to an atomic.Value holding the latest
//     plugin settings JSON, so /api/plugin-config returns mutable state.
//   - A separate plain-HTTP admin listener on 127.0.0.1 (random port) exposing
//     POST /__test__/plugin-config for the SSE hot-swap spec to flip server-
//     side settings + trigger BroadcastPluginConfig().
//
// Build-tagged 'playwrightfixture' so it never compiles into release builds
// (T-93-iPad-UAT-EVASION mitigation: the /__test__ surface is fixture-only).
//
// Usage (from frontend/playwright.config.ts webServer):
//
//	go run -tags=playwrightfixture ./cmd/playwright-fixture
//
// On startup the binary writes one line to stdout in KEY=VALUE form (then
// continues running until SIGTERM):
//
//	BASE_URL=https://127.0.0.1:PORT
//	CAP=<token>
//	ADMIN_URL=http://127.0.0.1:ADMIN_PORT
//	READY=1
//
// The Playwright config's webServer.url field is used to wait for readiness.
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"path/filepath"

	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/files"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/webserver"
)

const (
	sessionID    = "playwright-test-session"
	sessionName  = "Playwright Test Session"
	sessionAgent = "claude"
	sessionHost  = "fixture-host"
)

// fixedTestKey is a 32-byte deterministic HMAC signing key for the fixture.
// Tokens minted with this key are valid only against this fixture's webserver
// (because the webserver's SetSigningKey is called with the same value).
// Never used in production — gated by the playwrightfixture build tag.
var fixedTestKey = func() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i + 1)
	}
	return k
}()

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("[playwright-fixture] ")

	// 1. Self-signed TLS for 127.0.0.1.
	tlsCfg, err := selfSignedTLS()
	if err != nil {
		log.Fatalf("selfSignedTLS: %v", err)
	}

	// 2. Hub manager + one pre-seeded session backed by io.Pipe stub.
	manager := relay.NewHubManager()
	ptyOutR, ptyOutW := io.Pipe()
	inputCaptureR, inputCaptureW := io.Pipe()
	_ = inputCaptureR
	manager.Create(sessionID, ptyOutR, inputCaptureW, nil)

	// 3. WebServer in tailscale-mode with TLSConfig override (mirrors
	//    testServerWithHub but without *testing.T).
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		Mode:      "tailscale",
		TLSConfig: tlsCfg,
	}
	ws, err := webserver.NewWebServer(cfg, manager)
	if err != nil {
		log.Fatalf("NewWebServer: %v", err)
	}
	ws.SetSigningKey(fixedTestKey)
	// Phase 120-06 Task 3 — wire the embedded React bundle so /app/ serves
	// the SPA under playwright tests. staticAppFixture() returns nil under
	// `-tags playwrightfixture` alone (assets_stub.go) and the embedded
	// bundle under `-tags playwrightfixture,wailsassets` (assets_prod.go).
	// global-setup.ts always builds with both tags so the bundle is present
	// in normal e2e runs; scenario 12 keeps a 503-tolerant smoke for the
	// stub case.
	ws.SetStaticAppFS(staticAppFixture())
	ws.EnableSession(sessionID)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == sessionID {
			return sessionName, sessionAgent, "running", sessionHost
		}
		return id, "", "", ""
	})

	// 4. Plugin settings: atomic.Value holding pre-marshaled JSON. Default
	//    is everything-on so the vendor-parity spec sees all addons load.
	var pluginJSON atomic.Value
	pluginJSON.Store(mustMarshal(defaultPluginSettings()))
	ws.SetPluginSettingsProvider(func() []byte {
		v := pluginJSON.Load()
		if v == nil {
			return nil
		}
		return v.([]byte)
	})

	// 4a. Phase 120-05 — seed a deterministic test tree in a tempdir and
	// wire the files.Handler against a sandbox rooted there. The tree is
	// the canonical set described in Plan 01's <interfaces>:
	//   hello.txt     14 bytes plain text
	//   notes.md      GFM markdown with table + task list
	//   image.png     valid 1x1 PNG
	//   binary.bin    64 bytes of alternating 0x00 0xFF (binary)
	//   large.txt     5*1024*1024+1 bytes of 'A' (1 byte over 5 MiB cap)
	//   empty.txt     0 bytes
	//   subdir/nested.txt  "nested\n"
	sessionCwd, err := seedFixtureFiles()
	if err != nil {
		log.Fatalf("seedFixtureFiles: %v", err)
	}
	sandbox, err := files.NewSandbox(sessionCwd)
	if err != nil {
		log.Fatalf("files.NewSandbox: %v", err)
	}
	filesHandler := files.NewHandler(func(sid string) (*files.Sandbox, error) {
		if sid != sessionID {
			return nil, fmt.Errorf("unknown session: %s", sid)
		}
		return sandbox, nil
	})
	ws.SetFilesHandler(filesHandler)

	// 5. Start the WebServer's HTTPS listener.
	if err := ws.Start(); err != nil {
		log.Fatalf("ws.Start: %v", err)
	}

	// 6. Mint two capabilities for the test session:
	//   - OWNER cap: read,write,files.read (full file browser access)
	//   - VIEWER cap: read only (no files.read → /api/files/* returns 403
	//     with the "files.read capability required" body)
	ownerClaims := capability.Claims{
		SID:     sessionID,
		Perms:   "read,write,files.read",
		IAT:     time.Now().Unix(),
		GrantID: "grant-playwright-fixture-owner",
		V:       1,
	}
	token, err := capability.Sign(ownerClaims, fixedTestKey)
	if err != nil {
		log.Fatalf("capability.Sign owner: %v", err)
	}
	ws.AddGrant(sessionID, ownerClaims.GrantID)

	viewerClaims := capability.Claims{
		SID:     sessionID,
		Perms:   "read",
		IAT:     time.Now().Unix(),
		GrantID: "grant-playwright-fixture-viewer",
		V:       1,
	}
	viewerToken, err := capability.Sign(viewerClaims, fixedTestKey)
	if err != nil {
		log.Fatalf("capability.Sign viewer: %v", err)
	}
	ws.AddGrant(sessionID, viewerClaims.GrantID)

	// Phase 125-01 — third cap with files.write for the web-share write + 412
	// e2e scenarios (EDIT-13). The viewer read-only cap (no files.write) covers
	// the 403-without-cap scenario; this write cap covers the write-enabled path.
	writeClaims := capability.Claims{
		SID:     sessionID,
		Perms:   "read,files.read,files.write",
		IAT:     time.Now().Unix(),
		GrantID: "grant-playwright-fixture-write",
		V:       1,
	}
	writeToken, err := capability.Sign(writeClaims, fixedTestKey)
	if err != nil {
		log.Fatalf("capability.Sign write: %v", err)
	}
	ws.AddGrant(sessionID, writeClaims.GrantID)

	// 6b. Phase 122-05 — mock "remote peer" webserver on a second TLS port.
	//     Mimics a peer AgentHub on the tailnet for the remote-session
	//     file-browse e2e scenarios (16+17). Serves:
	//        POST /join/exchange → 303 Location: /sessions/peer-sid?cap=FIXTURE_CAP
	//        GET /api/files/list?session=peer-sid&path=.&cap=FIXTURE_CAP → 200 canned
	//        GET /api/files/list (no cap or wrong cap)                  → 401
	//        GET /api/files/stat / read (with cap)                      → 200 canned
	//     Listener address is published via the REMOTE_PEER_URL line so
	//     specs and the e2e/fixtures/remote-peer-setup.ts helper can read it.
	remotePeerURL, remotePeerStop := startRemotePeerFixture(tlsCfg)
	defer remotePeerStop()

	// 7. Admin HTTP server (plain, separate port) for /__test__/plugin-config.
	adminLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("admin listen: %v", err)
	}
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/__test__/plugin-config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var settings map[string]bool
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, "decode body: "+err.Error(), http.StatusBadRequest)
			return
		}
		// Re-marshal to ensure canonical key set is present and well-formed.
		data, err := json.Marshal(settings)
		if err != nil {
			http.Error(w, "marshal: "+err.Error(), http.StatusInternalServerError)
			return
		}
		pluginJSON.Store(data)
		// Trigger SSE broadcast so any subscribed client receives the new frame.
		ws.BroadcastPluginConfig(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
	adminMux.HandleFunc("/__test__/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	adminSrv := &http.Server{
		Handler:           adminMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := adminSrv.Serve(adminLn); err != nil && err != http.ErrServerClosed {
			log.Printf("admin Serve: %v", err)
		}
	}()

	// 8. Print the URLs/cap and READY signal so playwright.config.ts can
	//    parse stdout and pass the values via env vars to specs.
	baseURL := "https://" + ws.Addr()
	adminURL := "http://" + adminLn.Addr().String()
	fmt.Printf("BASE_URL=%s\n", baseURL)
	fmt.Printf("CAP=%s\n", token)
	fmt.Printf("VIEWER_CAP=%s\n", viewerToken)
	fmt.Printf("WRITE_CAP=%s\n", writeToken)
	fmt.Printf("SESSION_CWD=%s\n", sessionCwd)
	fmt.Printf("ADMIN_URL=%s\n", adminURL)
	fmt.Printf("REMOTE_PEER_URL=%s\n", remotePeerURL)
	fmt.Println("READY=1")

	// 9. Wait for SIGTERM/SIGINT, then clean up.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	log.Println("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = adminSrv.Shutdown(shutdownCtx)
	_ = ws.Stop()
	manager.Shutdown()
	_ = ptyOutW.Close()
	_ = inputCaptureW.Close()
}

// seedFixtureFiles creates the deterministic test tree described in
// Plan 120-01's <interfaces> block, in a fresh tempdir. Returns the
// absolute path to the dir so it can be passed to files.NewSandbox AND
// printed on stdout (SESSION_CWD=) so specs can correlate.
//
// The dir is NOT cleaned up on shutdown — Playwright's harness reruns
// reuse the same fixture binary across runs in this session, and the
// dir is small (<6 MiB) and lives under os.TempDir() which the system
// reaps on its own schedule.
func seedFixtureFiles() (string, error) {
	dir, err := os.MkdirTemp("", "playwright-fixture-cwd-*")
	if err != nil {
		return "", fmt.Errorf("MkdirTemp: %w", err)
	}
	// 1. hello.txt — 14 bytes plain text
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("Hello, world!\n"), 0o644); err != nil {
		return "", fmt.Errorf("write hello.txt: %w", err)
	}
	// 2. notes.md — GFM markdown with table + task list
	notesMD := "# Notes\n\n" +
		"A GFM document used by the file browser e2e suite.\n\n" +
		"## Table\n\n" +
		"| Header A | Header B |\n" +
		"| --- | --- |\n" +
		"| cell 1 | cell 2 |\n" +
		"| cell 3 | cell 4 |\n\n" +
		"## Task list\n\n" +
		"- [x] First task done\n" +
		"- [ ] Second task open\n"
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(notesMD), 0o644); err != nil {
		return "", fmt.Errorf("write notes.md: %w", err)
	}
	// 3. image.png — valid 1×1 transparent PNG (literal byte sequence)
	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41, 0x54, // IDAT
		0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00, 0x05,
		0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, // IEND
		0xAE, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(filepath.Join(dir, "image.png"), pngBytes, 0o644); err != nil {
		return "", fmt.Errorf("write image.png: %w", err)
	}
	// 4. binary.bin — 64 bytes of alternating 0x00 0xFF
	binBytes := make([]byte, 64)
	for i := range binBytes {
		if i%2 == 1 {
			binBytes[i] = 0xFF
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "binary.bin"), binBytes, 0o644); err != nil {
		return "", fmt.Errorf("write binary.bin: %w", err)
	}
	// 5. large.txt — 5*1024*1024+1 bytes of 'A' (1 byte over 5 MiB cap)
	largeBytes := make([]byte, 5*1024*1024+1)
	for i := range largeBytes {
		largeBytes[i] = 'A'
	}
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), largeBytes, 0o644); err != nil {
		return "", fmt.Errorf("write large.txt: %w", err)
	}
	// 6. empty.txt — 0 bytes
	if err := os.WriteFile(filepath.Join(dir, "empty.txt"), []byte{}, 0o644); err != nil {
		return "", fmt.Errorf("write empty.txt: %w", err)
	}
	// 7. subdir/nested.txt
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir subdir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "nested.txt"), []byte("nested\n"), 0o644); err != nil {
		return "", fmt.Errorf("write subdir/nested.txt: %w", err)
	}
	// 8. emptydir — an empty subdirectory for the empty-state test
	if err := os.Mkdir(filepath.Join(dir, "emptydir"), 0o755); err != nil {
		return "", fmt.Errorf("mkdir emptydir: %w", err)
	}
	return dir, nil
}

func defaultPluginSettings() map[string]bool {
	return map[string]bool{
		"webgl":     true,
		"unicode11": true,
		"clipboard": true,
		"search":    true,
		"webLinks":  true,
		"image":     true,
		"serialize": true,
		"progress":  false,
	}
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// startRemotePeerFixture stands up a second TLS listener that mimics a remote
// AgentHub peer on the tailnet for the Phase 122-05 remote-session e2e
// scenarios. Returns the base URL ("https://127.0.0.1:PORT") and a stop
// closure the caller defers.
//
// Endpoints (all under fixtureRemoteCap="FIXTURE_CAP", fixtureRemoteSession="peer-sid"):
//
//	POST /join/exchange                                          → 303 Location: /sessions/peer-sid?cap=FIXTURE_CAP
//	GET  /api/files/list?session=peer-sid&path=.&cap=FIXTURE_CAP → 200 canned FileListResponse
//	GET  /api/files/list (no/wrong cap)                          → 401 "cap rejected"
//	GET  /api/files/stat?session=peer-sid&path=a.txt&cap=...     → 200 canned FileEntry
//	GET  /api/files/read?session=peer-sid&path=a.txt&cap=...     → 200 "hello world"
//
// The canned shape matches the Go cross-surface parity test
// (internal/daemon/remote_files_parity_test.go) so both observers agree on
// the byte-identical expected response.
//
// We reuse the same self-signed TLS config as the main webserver because
// (a) it's already valid for 127.0.0.1 and (b) the e2e specs use
// ignoreHTTPSErrors anyway. The shared config is fine — TLS handshake is
// stateless per connection.
//
// Phase 128-03 (RMW-01/02/03): write verbs are backed by a real files.Sandbox
// rooted at a temp directory so write-then-read round-trips actual bytes
// (Pitfall 2 — NOT canned responses). The canned read handlers for list/stat
// remain for backward-compat with scenarios 16+17.
func startRemotePeerFixture(tlsCfg *tls.Config) (baseURL string, stop func()) {
	const (
		fixtureRemoteCap     = "FIXTURE_CAP"
		fixtureRemoteSession = "peer-sid"
	)

	// Phase 128-03: real sandbox so writes genuinely persist (Pitfall 2).
	sandboxDir, mkErr := os.MkdirTemp("", "remote-peer-fixture-*")
	if mkErr != nil {
		log.Fatalf("remote-peer sandbox dir: %v", mkErr)
	}
	sandbox, sbErr := files.NewSandbox(sandboxDir)
	if sbErr != nil {
		log.Fatalf("remote-peer sandbox: %v", sbErr)
	}

	canonicalList := map[string]any{
		"entries": []map[string]any{
			{"name": "a.txt", "size": 100, "isDir": false},
			{"name": "sub", "size": 0, "isDir": true},
		},
		"truncated": false,
	}
	canonicalStat := map[string]any{
		"name":  "a.txt",
		"size":  100,
		"isDir": false,
	}

	mux := http.NewServeMux()

	guard := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("cap") != fixtureRemoteCap {
				http.Error(w, "cap rejected", http.StatusUnauthorized)
				return
			}
			if got := r.URL.Query().Get("session"); got != fixtureRemoteSession {
				http.Error(w, "wrong session", http.StatusNotFound)
				return
			}
			handler(w, r)
		}
	}

	mux.HandleFunc("POST /join/exchange", func(w http.ResponseWriter, r *http.Request) {
		// Accept any code (the fixture isn't validating the code itself —
		// the test asserts on the redirect contract).
		w.Header().Set("Location", "/sessions/"+fixtureRemoteSession+"?cap="+fixtureRemoteCap)
		w.WriteHeader(http.StatusSeeOther)
	})
	mux.HandleFunc("GET /api/files/list", guard(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(canonicalList)
	}))
	mux.HandleFunc("GET /api/files/stat", guard(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(canonicalStat)
	}))

	// GET /api/files/read: serves from the persisted sandbox first (for
	// files written by the write verbs); falls back to canned "hello world"
	// for backward-compat with scenarios 16+17 which read the canonical a.txt.
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
		// Fallback: canned "hello world" (backward-compat with scenarios 16+17).
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Length", "11")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write([]byte("hello world"))
	}))
	mux.HandleFunc("HEAD /api/files/read", guard(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Length", "11")
		w.WriteHeader(http.StatusOK)
	}))

	// PUT /api/files/write: persist body via WriteFileAtomic so read-back
	// returns actual written bytes (Phase 128-03 RMW-01 persistence).
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

	// DELETE /api/files/delete: remove a file from the sandbox.
	mux.HandleFunc("DELETE /api/files/delete", guard(func(w http.ResponseWriter, r *http.Request) {
		rel := r.URL.Query().Get("path")
		if rel == "" {
			http.Error(w, "path is required", http.StatusBadRequest)
			return
		}
		if delErr := sandbox.Delete(rel); delErr != nil {
			http.Error(w, "delete: "+delErr.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))

	// POST /api/files/rename: rename a file inside the sandbox.
	mux.HandleFunc("POST /api/files/rename", guard(func(w http.ResponseWriter, r *http.Request) {
		oldRel := r.URL.Query().Get("path")
		newRel := r.URL.Query().Get("newPath")
		if oldRel == "" || newRel == "" {
			http.Error(w, "path and newPath are required", http.StatusBadRequest)
			return
		}
		if renErr := sandbox.Rename(oldRel, newRel); renErr != nil {
			http.Error(w, "rename: "+renErr.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))

	// POST /api/files/mkdir: create a directory inside the sandbox.
	mux.HandleFunc("POST /api/files/mkdir", guard(func(w http.ResponseWriter, r *http.Request) {
		rel := r.URL.Query().Get("path")
		if rel == "" {
			http.Error(w, "path is required", http.StatusBadRequest)
			return
		}
		if mkdirErr := sandbox.Mkdir(rel); mkdirErr != nil {
			http.Error(w, "mkdir: "+mkdirErr.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))

	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		log.Fatalf("remote-peer listen: %v", err)
	}

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("remote-peer Serve: %v", err)
		}
	}()

	return "https://" + ln.Addr().String(), func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		_ = os.RemoveAll(sandboxDir)
	}
}

// selfSignedTLS generates an in-memory self-signed CA + leaf for 127.0.0.1.
// Mirrors the helper used by `testServerWithHub` in the non-test package.
func selfSignedTLS() (*tls.Config, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Playwright Fixture CA"}},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(2 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, err
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(2 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	leafCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		return nil, err
	}
	leafKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})
	tlsCert, err := tls.X509KeyPair(leafCertPEM, leafKeyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
