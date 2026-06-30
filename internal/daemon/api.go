package daemon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/files"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/tailnet"
	"github.com/scottkw/agenthub/internal/webserver"
)

// API serves the daemon HTTP API over a Unix socket or Windows named pipe.
type API struct {
	engine        *SessionEngine
	mux           *http.ServeMux
	ln            net.Listener
	relayPort     int                  // TCP port the relay server is listening on
	relayLn       net.Listener         // TCP listener for the relay server
	mu            sync.RWMutex         // guards webServer and localPassword
	webServer     *webserver.WebServer // nil when not running
	tailnetCache  *tailnetCache
	localPassword string // generated once per daemon lifetime; non-empty in local mode

	// --- Phase 87 capability state (D-04/D-06/D-14/D-15/D-16) -----------
	// signingKey is the 32-byte HMAC-SHA256 key used to Sign capabilities.
	// Bootstrapped from disk in BootstrapCapabilityState; replaced atomically
	// by handleRegenerateSigningKey. All reads/writes go under signingKeyMu
	// (a dedicated lock separate from a.mu so a capability request does not
	// block an unrelated operation on webServer/localPassword).
	signingKeyMu sync.RWMutex
	signingKey   []byte
	// joinCodes is the in-memory short-lived join-code manager (D-09/D-11).
	// The Plan-06 /join/exchange handler on the webserver consumes the same
	// manager via ws.SetJoinCodes.
	joinCodes *capability.JoinCodeManager

	// filesHandler serves /api/files/{list,stat,read} on the daemon socket.
	// Phase 118 / FS-03..FS-07: the resolver closes over engine.GetSessionWorkDir
	// + files.NewSandbox so each request reconstructs a fresh per-session
	// Sandbox (no shared mutable state across requests). Loopback transport
	// is the trust boundary — no auth gate here; the webserver mount
	// (Phase 119) adds requireFilesRead.
	filesHandler *files.Handler

	// remoteCaps caches per-session (baseURL, capToken) tuples used by the
	// /api/files/remote/{sessionID}/... proxy routes. In-memory only;
	// daemon restart wipes the cache (Phase 122-01 / REMOTE-01).
	remoteCaps *RemoteCapStore

	// remoteFilesClientForTest, when non-nil, overrides the outbound HTTPS
	// client used by the remote-files proxy. Tests inject httptest.NewTLSServer's
	// Client() here so they can drive the proxy without setting up a real
	// trusted-cert chain. Production code leaves this nil and the proxy
	// builds a fresh client per request via newRemoteFilesHTTPClient().
	remoteFilesClientForTest *http.Client

	// --- Phase 165 Funnel state (FNL-01/FNL-05/FNL-07) ----------------------
	// funnelSessions tracks which sessions have Tailscale Funnel active.
	// DisableFunnel is called on the WebServer ONLY when len(funnelSessions)==0
	// (ref-count gate — Anti-Pattern 3 from 165-RESEARCH.md: never tear down a
	// still-active sibling session). Lazy-initialised on first enable. Guarded by a.mu.
	funnelSessions map[string]bool
	// funnelExpiry holds per-session auto-expiry timers (FNL-07).
	// A timer fires a.disableFunnelForSession asynchronously via time.AfterFunc.
	// Early teardown calls t.Stop() + delete to prevent double-fire (T-165-13).
	// Lazy-initialised on first timer registration. Guarded by a.mu.
	funnelExpiry map[string]*time.Timer
}

// NewAPI creates an API wired to the given SessionEngine and registers all routes.
func NewAPI(engine *SessionEngine) *API {
	a := &API{
		engine:       engine,
		mux:          http.NewServeMux(),
		tailnetCache: &tailnetCache{},
		remoteCaps:   NewRemoteCapStore(),
	}
	// Phase 118 / FS-03..FS-07: construct the file Handler BEFORE registerRoutes
	// so the route registrations have a non-nil target. The resolver closes
	// over a.engine.GetSessionWorkDir + files.NewSandbox — each request gets
	// a freshly-constructed Sandbox rooted at the session's resolved workDir.
	a.filesHandler = files.NewHandler(func(sessionID string) (*files.Sandbox, error) {
		if sessionID == "" {
			return nil, errors.New("missing session parameter")
		}
		wd := a.engine.GetSessionWorkDir(sessionID)
		if wd == "" {
			return nil, errors.New("session not found or has no working directory")
		}
		return files.NewSandbox(wd)
	})
	a.registerRoutes()
	return a
}

// registerRoutes wires all HTTP routes using Go 1.22+ path parameters.
func (a *API) registerRoutes() {
	a.mux.HandleFunc("GET /health", a.handleHealth)
	a.mux.HandleFunc("GET /sessions", a.handleListSessions)
	a.mux.HandleFunc("POST /sessions", a.handleCreateSession)
	a.mux.HandleFunc("GET /sessions/{id}", a.handleGetSession)
	a.mux.HandleFunc("DELETE /sessions/{id}", a.handleDeleteSession)
	a.mux.HandleFunc("PATCH /sessions/{id}/name", a.handleRenameSession)
	a.mux.HandleFunc("GET /sessions/{id}/status", a.handleGetSessionStatus)
	a.mux.HandleFunc("GET /sessions/{id}/tail", a.handleGetSessionTailLines)
	a.mux.HandleFunc("GET /sessions/{id}/styled-tail", a.handleGetSessionStyledTailLines)
	a.mux.HandleFunc("GET /settings/cli-paths", a.handleGetCLIPaths)
	a.mux.HandleFunc("GET /shells", a.handleListShells)
	a.mux.HandleFunc("PATCH /settings/cli-paths/{name}", a.handleUpdateCLIPath)
	a.mux.HandleFunc("GET /settings/start-minimized", a.handleGetStartMinimized)
	a.mux.HandleFunc("PATCH /settings/start-minimized", a.handleSetStartMinimized)
	a.mux.HandleFunc("GET /settings/shell-web-share-warned", a.handleGetShellWebShareWarned)
	a.mux.HandleFunc("PATCH /settings/shell-web-share-warned", a.handleUpdateShellWebShareWarned)
	a.mux.HandleFunc("GET /settings/shell-web-share-warning-enabled", a.handleGetShellWebShareWarningEnabled)
	a.mux.HandleFunc("PATCH /settings/shell-web-share-warning-enabled", a.handleSetShellWebShareWarningEnabled)
	a.mux.HandleFunc("GET /settings/shell-path", a.handleGetShellPath)
	a.mux.HandleFunc("PATCH /settings/shell-path", a.handleUpdateShellPath)
	a.mux.HandleFunc("GET /settings/auto-close-session", a.handleGetAutoCloseSession)
	a.mux.HandleFunc("PATCH /settings/auto-close-session", a.handleSetAutoCloseSession)
	a.mux.HandleFunc("GET /settings/plugins", a.handleGetPluginSettings)
	a.mux.HandleFunc("PATCH /settings/plugins", a.handleSetPluginSettings)
	a.mux.HandleFunc("PATCH /settings/search-config", a.handleSetSearchConfig)
	a.mux.HandleFunc("PATCH /settings/web-links-config", a.handleSetWebLinksConfig)
	a.mux.HandleFunc("PATCH /settings/image-config", a.handleSetImageConfig)
	// Relay port and web server routes.
	a.mux.HandleFunc("GET /relay-port", a.handleRelayPort)
	a.mux.HandleFunc("POST /webserver/start", a.handleWebServerStart)
	a.mux.HandleFunc("POST /webserver/stop", a.handleWebServerStop)
	a.mux.HandleFunc("GET /webserver/status", a.handleWebServerStatus)
	a.mux.HandleFunc("POST /sessions/{id}/web-serve", a.handleWebServe)
	a.mux.HandleFunc("POST /shutdown", a.handleShutdown)
	// Theme change notification — signals active OpenCode sessions.
	a.mux.HandleFunc("POST /theme/notify", a.handleNotifyThemeChange)
	// Tailnet peer discovery.
	a.mux.HandleFunc("GET /tailnet/peers", a.handleTailnetPeers)
	// Local mode password endpoint.
	a.mux.HandleFunc("GET /webserver/local-password", a.handleGetLocalPassword)
	// Phase 87 capability-based authorization endpoints (D-06, D-09, D-16).
	a.mux.HandleFunc("POST /sessions/{id}/capabilities", a.handleIssueCapabilities)
	// Phase 137 / SHARE-03: per-session browse toggle.
	// Loopback-trust (daemon socket): no auth gate — the owner's GUI is the only caller.
	// Toggle-off clears grants (ClearGrants) — stale-cap threat mitigation (SHARE-05).
	a.mux.HandleFunc("POST /sessions/{id}/browse", a.handleSetSessionBrowse)
	// Phase 165 / FNL-01: per-session Tailscale Funnel toggle.
	// Loopback-trust (daemon socket): no auth gate — only the owner's GUI calls this.
	// Disable path routes through disableFunnelForSession (ref-count gate, T-165-09).
	a.mux.HandleFunc("POST /sessions/{id}/funnel", a.handleSetSessionFunnel)
	a.mux.HandleFunc("POST /join/exchange", a.handleExchangeJoinCode)
	a.mux.HandleFunc("POST /capability/regenerate-key", a.handleRegenerateSigningKey)
	// Phase 118 / FS-03..FS-07: read-only file API on the daemon-local socket.
	// No auth here — the loopback transport (Unix socket / Windows named pipe)
	// is the trust boundary (ARCHITECTURE.md Decision 2 + REQUIREMENTS.md WEB-01).
	// The webserver mounts the same Handler under requireFilesRead in Phase 119.
	// Method-prefixed routes per Go 1.22+ mux semantics: POST and other verbs
	// auto-return 405 (Pitfall 8 — do not register a fallback handler that
	// would mask this).
	// Phase 123-03 / FSW-08: write routes added on the same loopback-trust basis.
	a.mux.HandleFunc("GET /api/files/list", a.filesHandler.List)
	a.mux.HandleFunc("GET /api/files/stat", a.filesHandler.Stat)
	a.mux.HandleFunc("GET /api/files/read", a.filesHandler.Read)
	a.mux.HandleFunc("HEAD /api/files/read", a.filesHandler.Read)
	a.mux.HandleFunc("PUT /api/files/write", a.filesHandler.Write)
	a.mux.HandleFunc("POST /api/files/upload", a.filesHandler.Upload)
	a.mux.HandleFunc("DELETE /api/files/delete", a.filesHandler.Delete)
	a.mux.HandleFunc("POST /api/files/rename", a.filesHandler.Rename)
	a.mux.HandleFunc("POST /api/files/mkdir", a.filesHandler.Mkdir)
	// Phase 122-01 / REMOTE-01: remote-files proxy + cap deposit.
	// POST /api/remote-files/caps accepts (sessionId, baseUrl, capToken) from
	// the GUI/TUI after a successful join-code exchange; the four GET/HEAD
	// routes proxy file ops to the remote peer's webserver using that cap.
	// Loopback transport (Unix socket / Windows named pipe) is the trust
	// boundary; no auth gate here.
	a.mux.HandleFunc("POST /api/remote-files/caps", a.handleRegisterRemoteCap)
	// Phase 146-05 / GAP-146-A: read-only open-url endpoint. Returns the cap-bearing
	// open URL from RemoteCapStore so the frontend can open a remote session without
	// re-prompting for a join code (held-cap reuse path). No cap is minted; the cap
	// enters only the returned URL string (T-146-05-01 / T-146-05-04 accept).
	a.mux.HandleFunc("GET /api/remote-files/caps/{sessionID}/open-url", a.handleRemoteSessionOpenURL)
	a.mux.HandleFunc("GET /api/files/remote/{sessionID}/list", a.handleRemoteFilesList)
	a.mux.HandleFunc("GET /api/files/remote/{sessionID}/stat", a.handleRemoteFilesStat)
	a.mux.HandleFunc("GET /api/files/remote/{sessionID}/read", a.handleRemoteFilesRead)
	a.mux.HandleFunc("HEAD /api/files/remote/{sessionID}/read", a.handleRemoteFilesRead)
	// Phase 124-03 / CAP-10: five remote write proxy routes.
	// These are daemon-socket loopback routes (WEB-01) — no auth middleware or
	// Origin check; the remote peer's requireFilesWrite enforces cap+CSRF.
	a.mux.HandleFunc("PUT /api/files/remote/{sessionID}/write", a.handleRemoteFilesWrite)
	a.mux.HandleFunc("POST /api/files/remote/{sessionID}/upload", a.handleRemoteFilesUpload)
	a.mux.HandleFunc("DELETE /api/files/remote/{sessionID}/delete", a.handleRemoteFilesDelete)
	a.mux.HandleFunc("POST /api/files/remote/{sessionID}/rename", a.handleRemoteFilesRename)
	a.mux.HandleFunc("POST /api/files/remote/{sessionID}/mkdir", a.handleRemoteFilesMkdir)
}

// Handler returns the API's underlying http.Handler (the registered mux).
// Exposed so external test packages (e.g. internal/daemon's package_test
// suite, the Phase 122-05 cross-surface parity test) can drive the daemon's
// routes through httptest.NewServer without needing access to the unexported
// mux field. Production callers do not use this — the daemon socket layer
// reaches into the mux directly.
func (a *API) Handler() http.Handler {
	return a.mux
}

// SetRemoteFilesClientForTest overrides the daemon's outbound HTTPS client
// used by the /api/files/remote/{sid}/... proxy with the supplied client.
// Test-only — wraps the unexported remoteFilesClientForTest field so that
// external test packages can inject httptest.NewTLSServer's Client() and
// have the proxy trust the upstream's self-signed certificate. Production
// code must never call this; the field stays nil and proxyRemoteFiles
// builds a fresh client per request via newRemoteFilesHTTPClient.
func (a *API) SetRemoteFilesClientForTest(c *http.Client) {
	a.remoteFilesClientForTest = c
}

// BootstrapCapabilityState loads or generates the HMAC signing key (D-04) and
// constructs the in-memory JoinCodeManager (D-11). Must be called once during
// daemon startup, BEFORE any web server is created — otherwise requireCapability
// would see a nil signing key (Pitfall 3) and 401 every request.
//
// Callers must pair this with ws.SetSigningKey + ws.SetJoinCodes whenever a
// WebServer is constructed (AutoStartWebServer, handleWebServerStart, etc.).
// CurrentSigningKey and JoinCodes accessors expose the bootstrapped state.
func (a *API) BootstrapCapabilityState() error {
	store := capability.NewFileKeyStore(a.engine.configDir)
	key, err := capability.LoadOrGenerate(store)
	if err != nil {
		return fmt.Errorf("capability bootstrap: %w", err)
	}
	a.signingKeyMu.Lock()
	a.signingKey = key
	a.signingKeyMu.Unlock()
	// Join codes live for 5 minutes (D-11) and are NOT persisted across
	// restarts — on daemon restart, users must regenerate any outstanding
	// share codes. This is acceptable because codes are ephemeral sharing
	// artefacts, not durable credentials.
	a.joinCodes = capability.NewJoinCodeManager(5 * time.Minute)
	return nil
}

// CurrentSigningKey returns the current HMAC signing key under RLock.
// Callers MUST NOT mutate the returned slice; it shares the backing array
// with the field. Returns nil if BootstrapCapabilityState has not run.
func (a *API) CurrentSigningKey() []byte {
	a.signingKeyMu.RLock()
	defer a.signingKeyMu.RUnlock()
	return a.signingKey
}

// JoinCodes returns the bootstrapped JoinCodeManager, or nil if
// BootstrapCapabilityState has not run.
func (a *API) JoinCodes() *capability.JoinCodeManager {
	return a.joinCodes
}

// RelayHandler returns the http.Handler served on the relay loopback TCP port
// — the surface the Wails desktop GUI reaches over 127.0.0.1:<relayPort> (it
// cannot reach the daemon Unix socket). This is the relay server (sessions WS +
// local /api/files/*) wrapped first with the remote-files proxy routes
// (/api/files/remote/{sid}/...) and then with the chat history/export routes
// (/api/chat/{id}/history, /api/chat/{id}/export). Both wrap layers use *API
// methods that cannot live in the relay package without an import cycle. Exposed
// so external tests can drive the relay surface exactly as the webview does
// (mirrors Handler()).
func (a *API) RelayHandler() http.Handler {
	// Phase 120 CR-01: mount /api/files/* on the relay HTTP server so the Wails
	// desktop GUI can reach the read-only file API. The Wails webview hits
	// 127.0.0.1:<relayPort> (TCP) — it cannot reach the daemon's Unix socket,
	// where /api/files/* is also registered. Both surfaces share the same
	// *files.Handler instance, so sandbox + 5 MiB cap behaviour is identical.
	server := relay.NewServer(a.engine.Manager(), a.engine.Backend(), a.filesHandler)
	// Phase 152 / IDENT-01: wire identity providers so handleSession stamps the
	// owner's alias from the global AliasStore. Callback pattern breaks the
	// relay→daemon import cycle (mirrors setChatProviders). Guard on nil so a
	// failed AliasStore construction at startup doesn't prevent relay operation.
	if aliases := a.engine.Aliases(); aliases != nil {
		server.SetIdentityProviders(
			a.engine.hostname,
			aliases.GetOrDefault,
			func(key, alias string) { _ = aliases.Set(key, alias) },
		)
	} else {
		server.SetIdentityProviders(a.engine.hostname, nil, nil)
	}
	withFiles := a.wrapRelayWithRemoteFiles(server)
	// Phase 151-03 / PERSIST-01..02: chat history + export routes mounted
	// in the outer wrap layer (most-specific wins in Go 1.22+ mux). Like the
	// remote-files layer, these are *API methods that need a.engine and cannot
	// live in relay.NewServer without an import cycle.
	return a.wrapRelayWithChat(withFiles)
}

// StartRelay creates the relay HTTP server and starts it on a random TCP port.
// Returns the allocated port. Must be called after NewAPI.
func (a *API) StartRelay() (int, error) {
	server := a.RelayHandler()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("relay listener: %w", err)
	}
	a.relayPort = ln.Addr().(*net.TCPAddr).Port
	a.relayLn = ln
	go http.Serve(ln, server) //nolint:errcheck
	return a.relayPort, nil
}

// Start validates the socket path, cleans up any stale socket, and begins
// serving on the platform daemon socket.
func (a *API) Start(socketPath string) error {
	if err := ValidateSocketPath(socketPath); err != nil {
		return err
	}
	if err := CleanupStaleSocket(socketPath); err != nil {
		return err
	}
	ln, err := listenDaemonSocket(socketPath)
	if err != nil {
		return err
	}
	a.ln = ln
	go http.Serve(ln, a.mux) //nolint:errcheck
	return nil
}

// Stop closes the listener and removes the socket file when the platform uses
// a filesystem-backed socket.
//
// On Windows the returned error is diagnostic-only: winio.win32PipeListener.Close()
// (upstream) always returns nil, swallowing any overlapped-I/O cancellation error,
// so a nil return on Windows does not necessarily mean the close was clean. The
// Unix net.UnixListener.Close() path does propagate errors normally.
// Correspondingly, removeDaemonSocket short-circuits on Windows named-pipe paths
// (no filesystem entry to remove), so its return is also nil there by design;
// any error it surfaces today comes from os.Remove on the Unix path.
// Tests that assert api.Stop() == nil on Windows are therefore exercising a
// tautology and should be read as smoke checks, not error-path coverage.
func (a *API) Stop() error {
	// Stop web server if running.
	a.mu.Lock()
	ws := a.webServer
	a.webServer = nil
	a.mu.Unlock()
	if ws != nil {
		_ = ws.Stop()
	}

	// Close relay listener if open.
	if a.relayLn != nil {
		_ = a.relayLn.Close()
		a.relayLn = nil
	}

	if a.ln == nil {
		return nil
	}
	addr := a.ln.Addr().String()
	err := a.ln.Close()
	_ = removeDaemonSocket(addr)
	a.ln = nil
	return err
}

// Addr returns the listener address (for test inspection).
func (a *API) Addr() net.Addr {
	if a.ln == nil {
		return nil
	}
	return a.ln.Addr()
}

// SetWebServerForTest directly injects a running WebServer into the API's web
// server field. This is for use in unit tests only — it allows tests to bypass
// the Tailscale prerequisite check in handleWebServerStart.
func (a *API) SetWebServerForTest(ws *webserver.WebServer) {
	a.mu.Lock()
	a.webServer = ws
	a.mu.Unlock()
}

// SetLocalPassword stores the generated local-mode password. Called once from
// runDaemonCore before any web server is started. Thread-safe.
func (a *API) SetLocalPassword(pwd string) {
	a.mu.Lock()
	a.localPassword = pwd
	a.mu.Unlock()
}

// WebServerMode returns the running web server's mode ("tailscale" or
// "local"), or "" if no web server is running. Thread-safe.
func (a *API) WebServerMode() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.webServer == nil {
		return ""
	}
	return a.webServer.Mode()
}

// WebServerBindIP returns the running web server's bind IP, or "" if no
// web server is running. Thread-safe.
func (a *API) WebServerBindIP() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.webServer == nil {
		return ""
	}
	return a.webServer.BindIP()
}

// RestartWebServer stops the current web server (if any) and starts a new one with
// the given config. Used internally for mode upgrades (local -> tailscale).
// Unlike AutoStartWebServer, it always replaces the running server — it is not idempotent.
func (a *API) RestartWebServer(ip string, port int, fqdn, mode, password string) error {
	a.mu.Lock()
	if a.webServer != nil {
		_ = a.webServer.Stop()
		a.webServer = nil
	}
	a.mu.Unlock()
	// Reuse AutoStartWebServer which creates, configures, and starts the server.
	// It's safe because we just set webServer to nil above.
	return a.AutoStartWebServer(ip, port, fqdn, mode, password)
}

// AutoStartWebServer starts the web server if not already running.
// Called from runDaemonCore at startup; mirrors handleWebServerStart without HTTP.
// Returns nil if the server is already running (idempotent).
func (a *API) AutoStartWebServer(ip string, port int, fqdn, mode, password string) error {
	// Local mode requires a non-empty password to prevent unauthenticated access.
	if mode == "local" && password == "" {
		return fmt.Errorf("AutoStartWebServer: local mode requires a non-empty password")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.webServer != nil {
		return nil // already running
	}
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:   ip,
		Port:     port,
		FQDN:     fqdn,
		Mode:     mode,
		Password: password,
	}, a.engine.Manager())
	if err != nil {
		return err
	}
	ws.SetSessionResolver(func(sessionID string) (name, cliType, status, hostname string) {
		for _, s := range a.engine.ListSessions() {
			if s.ID == sessionID {
				return s.Name, s.CLI, a.engine.GetSessionStatus(sessionID), s.Hostname
			}
		}
		return sessionID, "", "", ""
	})
	// Phase 93 PLUG-04: provide the daemon's current plugin settings as
	// pre-marshaled JSON to the webserver's /api/plugin-config handler.
	// func() []byte (not PluginSettings) avoids the daemon→webserver→daemon
	// circular import. json.Marshal failure returns nil so the handler
	// responds 503 (web client falls back to built-in defaults).
	ws.SetPluginSettingsProvider(func() []byte {
		s := a.engine.GetPluginSettings()
		b, err := json.Marshal(s)
		if err != nil {
			return nil
		}
		return b
	})
	// Phase 93 PLUG-04 push channel: register BroadcastPluginConfig as the
	// engine's plugin-settings change listener so SSE subscribers get a frame
	// on every SetPluginSettings call (closes ROADMAP SC#4 — no manual page
	// reload). The single-listener slot in Engine is safe because the two
	// NewWebServer call sites are mutually exclusive at runtime.
	a.engine.SetPluginSettingsListener(func() {
		ws.BroadcastPluginConfig(context.Background())
	})
	// Wire capability state onto the web server BEFORE Start() so requireCapability
	// has a non-nil signing key when the first request arrives (Pitfall 3). The
	// bootstrapped signing key and joinCodes MUST be populated by a prior call
	// to BootstrapCapabilityState; if that did not happen, requireCapability
	// will 401 every request — which is the correct defensive default.
	a.signingKeyMu.RLock()
	key := a.signingKey
	a.signingKeyMu.RUnlock()
	ws.SetSigningKey(key)
	ws.SetFilesHandler(a.filesHandler)
	// Phase 151-03 / PERSIST-01..02: wire the chat provider so the webserver's
	// cap-gated /api/chat/{id}/history and /api/chat/{id}/export routes can
	// reach the session's ChatStore without importing the daemon package
	// (T-151-09 — no webserver→daemon import cycle; provider callback only).
	a.setChatProviders(ws)
	ws.SetStaticAppFS(getStaticAppFS())
	ws.SetJoinCodes(a.joinCodes)
	if err := ws.Start(); err != nil {
		return err
	}
	a.webServer = ws
	return nil
}

// setChatProviders wires the webserver's cap-gated chat routes to the session
// engine's ChatStore via two narrow callbacks (IN-04): a history-only provider
// that marshals the thread and an export-only provider that renders Markdown.
// Splitting them means each request does only the work it serves — the history
// route never runs Export() and the export route never marshals history. Both
// preserve WR-03 semantics: a missing store is (false, nil) → 404, while an
// internal failure on an existing session is (true, err) → 500 (never masked as
// 404). Shared by AutoStartWebServer and handleWebServerStart; the closure form
// keeps the webserver→daemon import cycle broken (T-151-09).
//
// Also wires SetAliasProviders (Phase 152 / IDENT-01) so the web read pump can
// look up and persist per-person aliases from the same global AliasStore as the
// relay path. Guarded for nil store so early-startup or test scenarios where
// aliasStore failed to construct do not crash.
func (a *API) setChatProviders(ws *webserver.WebServer) {
	ws.SetChatHistoryProvider(func(sessionID string) (history []byte, found bool, err error) {
		store, storeOK := a.engine.ChatStoreFor(sessionID)
		if !storeOK {
			// Session has no chat store — genuine not-found (404), not an error.
			return nil, false, nil
		}
		// Messages() never returns nil (make([]…, len)), so an empty thread
		// already marshals to `[]` not `null` (IN-01).
		b, mErr := json.Marshal(store.Messages())
		if mErr != nil {
			// Internal failure on an existing session — surface it (500), never
			// mask it as 404 (WR-03 / CLAUDE.md no silent fallbacks).
			log.Printf("chat: marshal history for session %q: %v", sessionID, mErr)
			return nil, true, fmt.Errorf("chat: marshal history: %w", mErr)
		}
		return b, true, nil
	})
	ws.SetChatExportProvider(func(sessionID string) (markdown string, found bool, err error) {
		store, storeOK := a.engine.ChatStoreFor(sessionID)
		if !storeOK {
			return "", false, nil
		}
		md, eErr := store.Export()
		if eErr != nil {
			log.Printf("chat: export for session %q: %v", sessionID, eErr)
			return "", true, fmt.Errorf("chat: export: %w", eErr)
		}
		return md, true, nil
	})
	// Phase 152 / IDENT-01: wire alias persistence callbacks.
	// Plain func closures avoid the webserver→daemon import cycle (T-151-09).
	// aliasStore may be nil when AliasStore construction failed at startup;
	// SetAliasProviders accepts nil-guarded callbacks — handleWSSRelay will
	// operate with no alias persistence (empty alias fallback) rather than panic.
	aliasStore := a.engine.Aliases()
	ws.SetAliasProviders(
		func(personKey, def string) string {
			if aliasStore == nil {
				return def
			}
			return aliasStore.GetOrDefault(personKey, def)
		},
		func(personKey, alias string) {
			if aliasStore == nil {
				return
			}
			_ = aliasStore.Set(personKey, alias) // error intentionally discarded
		},
	)
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok", Version: BuildVersion})
}

func (a *API) handleShutdown(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.Exit(0)
	}()
}

func (a *API) handleNotifyThemeChange(w http.ResponseWriter, r *http.Request) {
	if err := a.engine.NotifyThemeChange(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := a.engine.ListSessions()
	if sessions == nil {
		sessions = []SessionInfo{}
	}

	// Enrich with web-enabled state from the running web server (SERVE-02).
	a.mu.RLock()
	ws := a.webServer
	// Snapshot funnelSessions under the same read lock to avoid a separate lock
	// acquisition — both reads are already serialised by a.mu.RLock.
	funnelSnap := make(map[string]bool, len(a.funnelSessions))
	for k, v := range a.funnelSessions {
		funnelSnap[k] = v
	}
	a.mu.RUnlock()
	if ws != nil {
		for i := range sessions {
			sessions[i].WebEnabled = ws.IsSessionEnabled(sessions[i].ID)
		}
	}
	// Populate FunnelActive from the snapshot (FNL-01).
	// NOT omitempty: false must serialise so frontend polling detects expiry.
	for i := range sessions {
		sessions[i].FunnelActive = funnelSnap[sessions[i].ID]
	}

	writeJSON(w, http.StatusOK, sessions)
}

func (a *API) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Build onExit callback for web serving grace period (D-12).
	// When session exits naturally, disable web serving after 10 seconds so
	// web viewers see final output before serving stops. DisableSession is a
	// no-op for sessions that were never enabled.
	//
	// Also clear grants for the session (D-15, RESEARCH Pitfall 1): session
	// end invalidates all outstanding capabilities. Leaving grants in the map
	// would leak memory AND could (theoretically) allow a recycled session ID
	// to inherit stale grants.
	onExit := func(sessionID string, exitCode int) {
		time.AfterFunc(10*time.Second, func() {
			a.runSessionExitCleanup(sessionID)
		})
	}

	// Use background context — the PTY must outlive the HTTP request.
	// r.Context() would kill the session when the response is sent.
	id, err := a.engine.CreateSession(context.Background(), req.CLI, req.Name, req.WorkDir, req.Args, req.Cols, req.Rows, nil, onExit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Phase 87 / SEC-01: creating a session does NOT auto-enable web serving.
	// The user must explicitly toggle web-serving ON via POST
	// /sessions/{id}/web-serve (D-06 grant gesture) to expose the session.
	// This closes the SEC-01 finding: tailnet peers can no longer discover
	// newly-created sessions without an explicit share gesture.

	writeJSON(w, http.StatusCreated, CreateResponse{ID: id})
}

// runSessionExitCleanup disables web serving for a session and clears all of
// its grants. Invoked 10 seconds after the session PTY exits (handleCreateSession
// onExit). Extracted so tests can invoke the cleanup synchronously without
// waiting for the 10-second grace timer.
func (a *API) runSessionExitCleanup(sessionID string) {
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws != nil {
		ws.DisableSession(sessionID)
		ws.ClearGrants(sessionID) // D-15 also applies on natural exit (Pitfall 1)
	}
}

// runSessionExitCleanupForTest is a test-only alias for runSessionExitCleanup.
// Documented on the test surface so the test package can invoke the exact same
// cleanup routine that the onExit callback uses, without waiting on the
// 10-second time.AfterFunc grace period.
func (a *API) runSessionExitCleanupForTest(sessionID string) {
	a.runSessionExitCleanup(sessionID)
}

func (a *API) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sessions := a.engine.ListSessions()
	for _, s := range sessions {
		if s.ID == id {
			writeJSON(w, http.StatusOK, s)
			return
		}
	}
	http.Error(w, "session not found", http.StatusNotFound)
}

func (a *API) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.engine.KillSession(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleRenameSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req RenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := a.engine.RenameSession(id, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleGetSessionStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s := a.engine.GetSessionStatus(id)
	writeJSON(w, http.StatusOK, StatusResponse{Status: s})
}

// handleGetSessionTailLines returns the last n plain-text lines from the
// session's scrollback buffer. Phase 132 / CARD-07.
// CR-02: clamp n to [1..20] at the HTTP layer, mirroring the app.go Wails binding.
// The spec documents a [1..20] contract; without this clamp any caller on the
// Unix socket could request n=1000000 and bypass the intent of the spec.
func (a *API) handleGetSessionTailLines(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n := 4 // default
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
			n = parsed
		}
	}
	if n > 20 {
		n = 20 // mirror app.go clamp — enforce [1..20] at the daemon HTTP boundary
	}
	lines := a.engine.GetSessionTailLines(id, n)
	if lines == nil {
		lines = []string{}
	}
	writeJSON(w, http.StatusOK, TailLinesResponse{Lines: lines})
}

// handleGetSessionStyledTailLines returns the last n styled-cell lines from
// the session's scrollback buffer. Phase 139 / CARD-05.
// Mirrors handleGetSessionTailLines exactly — same n-clamp [1..20] defense.
func (a *API) handleGetSessionStyledTailLines(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n := 4 // default
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
			n = parsed
		}
	}
	if n > 20 {
		n = 20 // mirror app.go clamp — enforce [1..20] at the daemon HTTP boundary
	}
	spans := a.engine.GetSessionStyledTailLines(id, n)
	if spans == nil {
		spans = [][]StyledSpan{}
	}
	writeJSON(w, http.StatusOK, StyledTailLinesResponse{Lines: spans})
}

func (a *API) handleGetCLIPaths(w http.ResponseWriter, r *http.Request) {
	paths := a.engine.GetCLIPaths()
	if paths == nil {
		paths = map[string]string{}
	}
	writeJSON(w, http.StatusOK, paths)
}

// handleListShells returns the daemon's view of installed shells per
// pty.DiscoverShells. Read-only; no engine state mutated.
//
// The response always serialises to `{"shells":[...]}` — never
// `{"shells":null}` — because the slice is constructed via `make([]T, 0, n)`.
// Argv is defensively copied per entry so callers cannot mutate
// pty.knownShellSpecs via the response (T-100-09 in the plan's threat model).
func (a *API) handleListShells(w http.ResponseWriter, r *http.Request) {
	// IN-03: route through the engine wrapper (SessionEngine.DiscoverShells)
	// rather than calling pty.DiscoverShells directly, so HTTP tests can
	// substitute a fake engine and future engine-level caching applies.
	discovered := a.engine.DiscoverShells()
	out := make([]DetectedShell, 0, len(discovered))
	for _, s := range discovered {
		out = append(out, DetectedShell{
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Path:        s.Path,
			Argv:        append([]string(nil), s.Argv...),
		})
	}
	writeJSON(w, http.StatusOK, ShellsResponse{Shells: out})
}

func (a *API) handleUpdateCLIPath(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req UpdateCLIPathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := a.engine.UpdateCLIPath(name, req.Path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleGetStartMinimized(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"startMinimized": a.engine.GetStartMinimized()})
}

func (a *API) handleSetStartMinimized(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StartMinimized bool `json:"startMinimized"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	a.engine.SetStartMinimized(req.StartMinimized)
	w.WriteHeader(http.StatusNoContent)
}

// handleGetShellWebShareWarned returns the persisted shell-web-share-warned
// flag wrapped in `{"value": <bool>}`. Phase 101 SHELL-08.
func (a *API) handleGetShellWebShareWarned(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"value": a.engine.GetShellWebShareWarned()})
}

// handleUpdateShellWebShareWarned accepts `{"value": <bool>}` and persists
// the shell-web-share-warned flag. Phase 101 SHELL-08.
func (a *API) handleUpdateShellWebShareWarned(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value bool `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := a.engine.SetShellWebShareWarned(req.Value); err != nil {
		http.Error(w, fmt.Sprintf("persist: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetShellWebShareWarningEnabled returns the warning-enabled master switch
// wrapped in `{"value": <bool>}`. Defaults true (D-08). Phase 150 SET-01.
func (a *API) handleGetShellWebShareWarningEnabled(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"value": a.engine.GetShellWebShareWarningEnabled()})
}

// handleSetShellWebShareWarningEnabled accepts `{"value": <bool>}` and persists
// the warning-enabled master switch. When val=true the engine atomically resets
// shellWebShareWarned (D-03 re-arm). Phase 150 SET-01.
func (a *API) handleSetShellWebShareWarningEnabled(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value bool `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := a.engine.SetShellWebShareWarningEnabled(req.Value); err != nil {
		http.Error(w, fmt.Sprintf("persist: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetShellPath returns the persisted shell binary path wrapped in
// `{"value": "<string>"}`. When no path has been configured by the user,
// the engine resolves and returns the platform default (never empty).
// Phase 107 SHELL-11.
func (a *API) handleGetShellPath(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"value": a.engine.GetShellPath()})
}

// handleUpdateShellPath accepts `{"value": "<path>"}` and persists the shell
// binary path. An empty value clears the override (restores platform default).
// A non-empty path that does not exist or is not executable returns 400.
// Phase 107 SHELL-11.
func (a *API) handleUpdateShellPath(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192) // match peer handlers (handleSetPluginSettings etc.)
	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := a.engine.SetShellPath(req.Value); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleGetAutoCloseSession(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"autoCloseSession": a.engine.GetAutoCloseSession()})
}

func (a *API) handleSetAutoCloseSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AutoCloseSession bool `json:"autoCloseSession"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	a.engine.SetAutoCloseSession(req.AutoCloseSession)
	w.WriteHeader(http.StatusNoContent)
}

// handleGetPluginSettings returns the engine's current PluginSettings as a
// JSON object (direct struct shape — no wrapper map).
func (a *API) handleGetPluginSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.engine.GetPluginSettings())
}

// handleSetPluginSettings accepts a full PluginSettings struct as the body
// (full-replace semantic; the route is PATCH for consistency with surrounding
// settings routes, not for partial-update semantics) and persists it.
//
// Defense-in-depth (RESEARCH §Security):
//   - http.MaxBytesReader caps body at 8 KiB (T-92-02 mitigation).
//   - DisallowUnknownFields rejects schema poisoning attempts (T-92-03).
//   - PluginSettings has no SchemaVersion field, so a downgrade attack via
//     body crafting is impossible (T-92-04).
func (a *API) handleSetPluginSettings(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)

	var req PluginSettings
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	a.engine.SetPluginSettings(req)
	w.WriteHeader(http.StatusNoContent)
}

// handleSetSearchConfig accepts a SearchConfig struct and persists ONLY that
// sub-key of PluginSettings (Phase 94-07 WR-03 gap closure). Defense-in-depth
// mirrors handleSetPluginSettings: 8 KiB body cap + DisallowUnknownFields.
//
// The route is PATCH /settings/search-config — sibling to /settings/plugins —
// chosen for symmetry with /settings/auto-close-session (sub-key-style PATCH
// routes elsewhere in this API).
func (a *API) handleSetSearchConfig(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)

	var req SearchConfig
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	a.engine.SetSearchConfig(req)
	w.WriteHeader(http.StatusNoContent)
}

// handleSetWebLinksConfig accepts a WebLinksConfig struct and persists ONLY
// that sub-key of PluginSettings (Phase 95 LNK-05 / LNK-06). Defense-in-depth
// mirrors handleSetSearchConfig: 8 KiB body cap + DisallowUnknownFields.
//
// The route is PATCH /settings/web-links-config — sibling to
// /settings/search-config — chosen for symmetry with the other sub-key-style
// PATCH routes in this API.
func (a *API) handleSetWebLinksConfig(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)

	var req WebLinksConfig
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	// WR-02: validate Modifier against the four documented literals so a
	// typoed value or corrupted settings.json cannot silently disable the
	// entire feature (isModifierPressed falls through every if and
	// returns false → every modifier-click is gated off with no UX
	// feedback). The struct comment in plugin_settings.go documents
	// these four values; here we enforce them at the API boundary.
	switch req.Modifier {
	case "platform", "cmd", "ctrl", "none":
		// ok
	default:
		http.Error(w, "modifier must be one of: platform, cmd, ctrl, none", http.StatusBadRequest)
		return
	}
	a.engine.SetWebLinksConfig(req)
	w.WriteHeader(http.StatusNoContent)
}

// handleSetImageConfig accepts an ImageConfig struct and persists ONLY
// that sub-key of PluginSettings (Phase 96 IMG-02). Defense-in-depth
// mirrors handleSetWebLinksConfig: 8 KiB body cap (T-96-02-03) +
// DisallowUnknownFields (T-96-02-02).
//
// The route is PATCH /settings/image-config — sibling to
// /settings/web-links-config — chosen for symmetry with the other
// sub-key-style PATCH routes in this API.
//
// IMG-02 range gate (T-96-02-01): StorageLimit must be in [1, 1000] MB.
// Reject 0 (no images render — defeat-the-feature footgun) and >1000
// (defeats tab-OOM mitigation per ROADMAP Phase 96 SC-3; the upstream
// addon-image default is 128 MB but AgentHub locks 16 MB by default,
// allowing user override up to 1000 MB hard cap for hypothetical future
// power-user advanced disclosure).
func (a *API) handleSetImageConfig(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)

	var req ImageConfig
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.StorageLimit < 1 || req.StorageLimit > 1000 {
		http.Error(w, "storageLimit must be in range [1, 1000]", http.StatusBadRequest)
		return
	}
	a.engine.SetImageConfig(req)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleRelayPort(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, RelayPortResponse{Port: a.relayPort})
}

func (a *API) handleWebServerStart(w http.ResponseWriter, r *http.Request) {
	var req WebServerStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// In local mode, resolve LAN IP if caller did not provide one.
	if req.Mode == "local" && req.IP == "" {
		lanIP, err := webserver.GetLANIP()
		if err != nil {
			http.Error(w, "no LAN IP found: "+err.Error(), http.StatusInternalServerError)
			return
		}
		req.IP = lanIP
	}

	// Local mode requires a non-empty password to prevent unauthenticated access.
	if req.Mode == "local" && req.Password == "" {
		http.Error(w, "local mode requires a non-empty password", http.StatusBadRequest)
		return
	}

	// Stop any previously running server to avoid leaking its listener.
	a.mu.Lock()
	if a.webServer != nil {
		_ = a.webServer.Stop()
		a.webServer = nil
	}
	a.mu.Unlock()

	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:   req.IP,
		Port:     req.Port,
		FQDN:     req.FQDN,
		Mode:     req.Mode,
		Password: req.Password,
	}, a.engine.Manager())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set session resolver so the web server can look up session metadata.
	ws.SetSessionResolver(func(sessionID string) (name, cliType, status, hostname string) {
		for _, s := range a.engine.ListSessions() {
			if s.ID == sessionID {
				return s.Name, s.CLI, a.engine.GetSessionStatus(sessionID), s.Hostname
			}
		}
		return sessionID, "", "", ""
	})

	// Phase 93 PLUG-04: provide the daemon's current plugin settings as
	// pre-marshaled JSON to the webserver's /api/plugin-config handler.
	ws.SetPluginSettingsProvider(func() []byte {
		s := a.engine.GetPluginSettings()
		b, err := json.Marshal(s)
		if err != nil {
			return nil
		}
		return b
	})
	// Phase 93 PLUG-04 push channel: register BroadcastPluginConfig as the
	// engine's plugin-settings change listener (closes ROADMAP SC#4).
	a.engine.SetPluginSettingsListener(func() {
		ws.BroadcastPluginConfig(context.Background())
	})

	// Wire capability state BEFORE Start() so requireCapability has a key on
	// the first request (Pitfall 3).
	a.signingKeyMu.RLock()
	key := a.signingKey
	a.signingKeyMu.RUnlock()
	ws.SetSigningKey(key)
	ws.SetFilesHandler(a.filesHandler)
	// Phase 151-03 / PERSIST-01..02: wire the chat provider (mirrors AutoStartWebServer).
	a.setChatProviders(ws)
	ws.SetStaticAppFS(getStaticAppFS())
	ws.SetJoinCodes(a.joinCodes)

	if err := ws.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a.mu.Lock()
	a.webServer = ws
	a.mu.Unlock()

	writeJSON(w, http.StatusOK, WebServerStartResponse{URL: ws.BaseURL()})
}

func (a *API) handleWebServerStop(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	ws := a.webServer
	a.webServer = nil
	a.mu.Unlock()

	if ws != nil {
		_ = ws.Stop()
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleWebServerStatus(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()

	if ws == nil {
		writeJSON(w, http.StatusOK, WebServerStatusResponse{Running: false})
		return
	}
	writeJSON(w, http.StatusOK, WebServerStatusResponse{
		Running: true,
		URL:     ws.BaseURL(),
		Addr:    ws.Addr(),
		Mode:    ws.Mode(),
	})
}

// handleGetLocalPassword returns the local-mode password over the Unix socket.
// The socket is owned by the current user (0600) so only same-UID processes
// can reach this endpoint. This is the intended access-control model.
func (a *API) handleGetLocalPassword(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	pwd := a.localPassword
	a.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]string{"password": pwd})
}

// handleWebServe toggles web serving for a session (D-06 grant gesture).
//
// Enable path (req.Enabled==true): marks the session web-enabled so that
// subsequent POST /sessions/{id}/capabilities calls can issue capabilities.
// Returns 204 No Content. The authoritative capability issuance path is the
// separate POST /sessions/{id}/capabilities endpoint — keeping this handler
// body-less avoids dead weight in DaemonClient.ToggleWebServing (which
// discards the response body). The frontend (Plan 05) follows toggle-on with
// a separate POST /sessions/{id}/capabilities.
//
// Disable path (req.Enabled==false): marks the session web-disabled AND
// clears its entire grant list (D-15). Previously-issued capabilities for
// this session become permanently invalid — the user must run the Share flow
// again to produce fresh ones.
func (a *API) handleWebServe(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req WebServeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()

	if ws == nil {
		http.Error(w, "web server not running", http.StatusBadRequest)
		return
	}

	if req.Enabled {
		ws.EnableSession(id)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ws.DisableSession(id)
	ws.ClearGrants(id) // D-15: permanent grant clear on toggle-off
	w.WriteHeader(http.StatusNoContent)
}

// issueCapabilitiesForSession mints the two capabilities (read + read,write)
// for a web-enabled session, registers both grant_ids on the WebServer, and
// issues a short-lived join code (D-09) for each. Returns the two
// capability-bearing URLs and their join codes. Called by
// handleIssueCapabilities (POST /sessions/{id}/capabilities).
//
// This is the atomic "Share" operation from the user's perspective (D-07):
// one call produces both a read-only and a read-write link, each with its
// own grant_id for future revocation granularity.
func (a *API) issueCapabilitiesForSession(sessionID string) (readURL, writeURL, readCode, writeCode string, err error) {
	a.signingKeyMu.RLock()
	key := a.signingKey
	a.signingKeyMu.RUnlock()
	if key == nil {
		return "", "", "", "", errors.New("capability: signing key not bootstrapped")
	}

	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws == nil {
		return "", "", "", "", errors.New("web server not running")
	}
	if a.joinCodes == nil {
		return "", "", "", "", errors.New("capability: join-code manager not bootstrapped")
	}

	// Generate two 128-bit grant IDs (hex-encoded to 32 chars).
	var rgid, wgid [16]byte
	if _, err := rand.Read(rgid[:]); err != nil {
		return "", "", "", "", err
	}
	if _, err := rand.Read(wgid[:]); err != nil {
		return "", "", "", "", err
	}

	now := time.Now().Unix()
	// Phase 137 / SHARE-03: perm injection driven solely by the per-session browse
	// toggle (browseEnabledFor). No global kill-switch (D-07 deliberate removal of
	// the Phase 118 filesReadEnabled / FilesRead global flag — Reversal 3 in
	// 137-RESEARCH.md).
	//
	// D-03: Browse OFF (default): RO="read", RW="read,write"
	// D-04: Browse ON:            RO="read,files.read", RW="read,write,files.read,files.write"
	//
	// Security invariants (T-137-02 / Pitfall 2): files.write NEVER appears in
	// rPerms (the RO token). The RO code holder gains read-only filesystem access
	// when browse is ON — this is intentional (D-05).
	//
	// Deliberate reversals vs Phase 118/124:
	//   Reversal 1: The dual RO/RW issuance replaces the single owner-only token
	//               (T-124-07: write is no longer separately gated per-session).
	//   Reversal 3: Global filesReadEnabled kill-switch removed; per-session default
	//               OFF (D-06) is the new gate.
	// Audit: secure-phase to review Reversal 1 and Reversal 3 in 137-RESEARCH.md.
	rPerms := "read"
	wPerms := "read,write"
	if a.engine.browseEnabledFor(sessionID) {
		rPerms = "read," + capability.PermFilesRead
		wPerms = "read,write," + capability.PermFilesRead + "," + capability.PermFilesWrite
	}
	rClaims := capability.Claims{SID: sessionID, Perms: rPerms, IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
	wClaims := capability.Claims{SID: sessionID, Perms: wPerms, IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}

	rTok, err := capability.Sign(rClaims, key)
	if err != nil {
		return "", "", "", "", err
	}
	wTok, err := capability.Sign(wClaims, key)
	if err != nil {
		return "", "", "", "", err
	}

	// Register both grants BEFORE returning URLs so the caller's first
	// request with the returned token succeeds (no TOCTOU where the token
	// arrives before the grant is registered).
	ws.AddGrant(sessionID, rClaims.GrantID)
	ws.AddGrant(sessionID, wClaims.GrantID)

	base := ws.BaseURL()
	readURL = base + "/sessions/" + sessionID + "?cap=" + rTok
	writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok

	readCode, err = a.joinCodes.Issue(rTok)
	if err != nil {
		return "", "", "", "", err
	}
	writeCode, err = a.joinCodes.Issue(wTok)
	if err != nil {
		return "", "", "", "", err
	}
	return readURL, writeURL, readCode, writeCode, nil
}

// handleIssueCapabilities issues two capabilities for a web-enabled session
// (D-07) and returns their URLs and join codes. Called by the frontend after
// a toggle-on gesture. Returns 400 when the web server is not running or
// the session is not web-enabled.
func (a *API) handleIssueCapabilities(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws == nil {
		http.Error(w, "web server not running", http.StatusBadRequest)
		return
	}
	if !ws.IsSessionEnabled(id) {
		http.Error(w, "session not web-enabled", http.StatusBadRequest)
		return
	}
	readURL, writeURL, readCode, writeCode, err := a.issueCapabilitiesForSession(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, IssueCapabilitiesResponse{
		ReadURL:   readURL,
		WriteURL:  writeURL,
		ReadCode:  readCode,
		WriteCode: writeCode,
		HomeDir:   a.engine.sessionCwdIsHome(id), // Phase 124 / CAP-06: EvalSymlinks-normalized home-dir signal for the GUI warning banner
	})
}

// handleExchangeJoinCode consumes a single-use join code (D-09/D-11) and
// returns the capability-bearing URL the client should follow. Status codes:
//   - 200: code valid, returns ExchangeJoinCodeResponse{URL}
//   - 400: bad request body or empty code
//   - 404: code not found (never issued, already exchanged, GC'd)
//   - 410: code expired past TTL
//   - 500: token verify failed or web server not running
func (a *API) handleExchangeJoinCode(w http.ResponseWriter, r *http.Request) {
	var req ExchangeJoinCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Code == "" {
		http.Error(w, "code required", http.StatusBadRequest)
		return
	}
	if a.joinCodes == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	token, err := a.joinCodes.Exchange(req.Code)
	switch {
	case errors.Is(err, capability.ErrCodeExpired):
		http.Error(w, "code expired", http.StatusGone)
		return
	case errors.Is(err, capability.ErrCodeNotFound):
		http.Error(w, "invalid code", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	a.signingKeyMu.RLock()
	key := a.signingKey
	a.signingKeyMu.RUnlock()
	if key == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	claims, err := capability.Verify(token, key)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws == nil {
		http.Error(w, "web server not running", http.StatusBadRequest)
		return
	}
	url := ws.BaseURL() + "/sessions/" + claims.SID + "?cap=" + token
	writeJSON(w, http.StatusOK, ExchangeJoinCodeResponse{URL: url})
}

// handleRegenerateSigningKey replaces capability.key on disk, updates the
// in-memory signing key, and calls ws.SetSigningKey so requireCapability
// picks up the new key on the next request (D-16 panic button). All
// previously-issued capabilities fail verification against the new key —
// this is the intended blast radius. No attempt is made to preserve
// outstanding grants: the stale grants eventually expire when their sessions
// end, and the signature check alone is sufficient to block them.
func (a *API) handleRegenerateSigningKey(w http.ResponseWriter, r *http.Request) {
	newKey, err := capability.GenerateKey()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	store := capability.NewFileKeyStore(a.engine.configDir)
	if err := store.Save(newKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.signingKeyMu.Lock()
	a.signingKey = newKey
	a.signingKeyMu.Unlock()

	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws != nil {
		ws.SetSigningKey(newKey)
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) handleTailnetPeers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	peers := a.tailnetCache.getOrRefresh(ctx, tailnet.DiscoverAndProbe)
	writeJSON(w, http.StatusOK, peers)
}

// handleSetSessionFunnel handles POST /sessions/{id}/funnel.
// It enables or disables Tailscale Funnel per-session (FNL-01/FNL-07).
// Phase 165.
//
// Enable path (req.Enabled==true): calls ws.EnableFunnel(ctx, 443) and, if
// it succeeds, records funnelSessions[id]=true and optionally registers a
// time.AfterFunc expiry timer (FNL-07). Returns 200 with SetSessionFunnelResponse.
//
// Disable path (req.Enabled==false): calls disableFunnelForSession and returns 204.
//
// CheckFunnelAccess errors (FNL-06) are surfaced verbatim as 400 so the user
// sees the human-readable "Funnel not available; ..." text.
func (a *API) handleSetSessionFunnel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req SetSessionFunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws == nil {
		http.Error(w, "web server not running", http.StatusBadRequest)
		return
	}

	if !req.Enabled {
		a.disableFunnelForSession(r.Context(), id)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Enable path: call the real WebServer.EnableFunnel — CheckFunnelAccess runs inside.
	if err := ws.EnableFunnel(r.Context(), 443); err != nil {
		// Surface CheckFunnelAccess error verbatim (FNL-06).
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	if a.funnelSessions == nil {
		a.funnelSessions = make(map[string]bool)
	}
	a.funnelSessions[id] = true

	// FNL-07: register auto-expiry timer if expiresIn > 0.
	if req.ExpiresIn > 0 {
		if a.funnelExpiry == nil {
			a.funnelExpiry = make(map[string]*time.Timer)
		}
		// Cancel any existing timer for this session before registering a new one
		// (re-enable before expiry must not double-fire — T-165-13).
		if t, ok := a.funnelExpiry[id]; ok {
			t.Stop()
		}
		dur := time.Duration(req.ExpiresIn) * time.Second
		a.funnelExpiry[id] = time.AfterFunc(dur, func() {
			a.disableFunnelForSession(context.Background(), id)
		})
	}
	a.mu.Unlock()

	writeJSON(w, http.StatusOK, SetSessionFunnelResponse{FunnelURL: ws.FunnelBaseURL()})
}

// disableFunnelForSession clears the per-session Funnel state and calls
// ws.DisableFunnel when no other Funnel sessions remain (ref-count gate).
// Invoked by: handleSetSessionFunnel toggle-off (site 1), handleWebServe
// disable path (site 2), runSessionExitCleanup (site 3), and the funnelExpiry
// timer (site 5). Site 4 (daemon stop) is covered by ws.Stop()→DisableFunnel
// in Phase 165-01 — do NOT add a second call there (would double-fire).
//
// Locking contract: acquires a.mu.Lock, then releases before calling ws.DisableFunnel
// (blocking call must not hold the mutex — mirrors the pattern in runSessionExitCleanup).
func (a *API) disableFunnelForSession(ctx context.Context, sessionID string) {
	a.mu.Lock()
	// Stop and remove the expiry timer if present (T-165-13 double-fire prevention).
	if t, ok := a.funnelExpiry[sessionID]; ok {
		t.Stop()
		delete(a.funnelExpiry, sessionID)
	}
	delete(a.funnelSessions, sessionID)
	remaining := len(a.funnelSessions)
	ws := a.webServer
	a.mu.Unlock()

	// Ref-count gate (Anti-Pattern 3 / T-165-09): only call DisableFunnel when
	// the last Funnel session is torn down so a still-active sibling is never cut off.
	if ws != nil && remaining == 0 {
		_ = ws.DisableFunnel(ctx)
	}
}

// handleSetSessionBrowse handles POST /sessions/{id}/browse.
// It toggles the per-session browse flag for the given session.
// Phase 137 / SHARE-03. Loopback-trust (daemon socket): no auth gate needed.
//
// Browse toggle-off clears outstanding grants for the session (parity with
// handleWebServe ClearGrants on toggle-off, SHARE-05 stale-cap threat):
// a browse-off cannot leave a stale files.read cap live. The next call to
// IssueCapabilities re-mints tokens with the updated (browse-OFF) perm set.
func (a *API) handleSetSessionBrowse(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req SessionBrowseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	a.engine.SetSessionBrowse(id, req.Enabled)
	// Stale-cap mitigation (SHARE-05 / stale-cap threat): clear outstanding grants
	// on any browse toggle so a toggle-off cannot leave a live files.read cap.
	// Mirrors the handleWebServe ClearGrants pattern (api.go:1056).
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws != nil {
		ws.ClearGrants(id)
	}
	w.WriteHeader(http.StatusNoContent)
}
