package daemon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/tailnet"
	"github.com/scottkw/agenthub/internal/webserver"
)

// API serves the daemon HTTP API over a Unix socket.
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
}

// NewAPI creates an API wired to the given SessionEngine and registers all routes.
func NewAPI(engine *SessionEngine) *API {
	a := &API{
		engine:       engine,
		mux:          http.NewServeMux(),
		tailnetCache: &tailnetCache{},
	}
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
	a.mux.HandleFunc("GET /settings/cli-paths", a.handleGetCLIPaths)
	a.mux.HandleFunc("PATCH /settings/cli-paths/{name}", a.handleUpdateCLIPath)
	a.mux.HandleFunc("GET /settings/start-minimized", a.handleGetStartMinimized)
	a.mux.HandleFunc("PATCH /settings/start-minimized", a.handleSetStartMinimized)
	a.mux.HandleFunc("GET /settings/auto-close-session", a.handleGetAutoCloseSession)
	a.mux.HandleFunc("PATCH /settings/auto-close-session", a.handleSetAutoCloseSession)
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
	a.mux.HandleFunc("POST /join/exchange", a.handleExchangeJoinCode)
	a.mux.HandleFunc("POST /capability/regenerate-key", a.handleRegenerateSigningKey)
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

// StartRelay creates the relay HTTP server and starts it on a random TCP port.
// Returns the allocated port. Must be called after NewAPI.
func (a *API) StartRelay() (int, error) {
	server := relay.NewServer(a.engine.Manager(), a.engine.Backend())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("relay listener: %w", err)
	}
	a.relayPort = ln.Addr().(*net.TCPAddr).Port
	a.relayLn = ln
	go http.Serve(ln, server) //nolint:errcheck
	return a.relayPort, nil
}

// Start validates the socket path, cleans up any stale socket, creates the
// parent directory, and begins serving on the Unix socket.
func (a *API) Start(socketPath string) error {
	if err := ValidateSocketPath(socketPath); err != nil {
		return err
	}
	if err := CleanupStaleSocket(socketPath); err != nil {
		return err
	}
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	a.ln = ln
	go http.Serve(ln, a.mux) //nolint:errcheck
	return nil
}

// Stop closes the listener and removes the socket file.
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
	err := a.ln.Close()
	_ = os.Remove(a.ln.Addr().String())
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
	// Wire capability state onto the web server BEFORE Start() so requireCapability
	// has a non-nil signing key when the first request arrives (Pitfall 3). The
	// bootstrapped signing key and joinCodes MUST be populated by a prior call
	// to BootstrapCapabilityState; if that did not happen, requireCapability
	// will 401 every request — which is the correct defensive default.
	a.signingKeyMu.RLock()
	key := a.signingKey
	a.signingKeyMu.RUnlock()
	ws.SetSigningKey(key)
	ws.SetJoinCodes(a.joinCodes)
	if err := ws.Start(); err != nil {
		return err
	}
	a.webServer = ws
	return nil
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
	a.mu.RUnlock()
	if ws != nil {
		for i := range sessions {
			sessions[i].WebEnabled = ws.IsSessionEnabled(sessions[i].ID)
		}
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

func (a *API) handleGetCLIPaths(w http.ResponseWriter, r *http.Request) {
	paths := a.engine.GetCLIPaths()
	if paths == nil {
		paths = map[string]string{}
	}
	writeJSON(w, http.StatusOK, paths)
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

	// Wire capability state BEFORE Start() so requireCapability has a key on
	// the first request (Pitfall 3).
	a.signingKeyMu.RLock()
	key := a.signingKey
	a.signingKeyMu.RUnlock()
	ws.SetSigningKey(key)
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
	rClaims := capability.Claims{SID: sessionID, Perms: "read", IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
	wClaims := capability.Claims{SID: sessionID, Perms: "read,write", IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}

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
