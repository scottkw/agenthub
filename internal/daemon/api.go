package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/tailnet"
	"github.com/scottkw/agenthub/internal/webserver"
)

// API serves the daemon HTTP API over a Unix socket.
type API struct {
	engine       *SessionEngine
	mux          *http.ServeMux
	ln           net.Listener
	relayPort    int                  // TCP port the relay server is listening on
	relayLn      net.Listener         // TCP listener for the relay server
	mu           sync.RWMutex         // guards webServer and localPassword
	webServer    *webserver.WebServer // nil when not running
	tailnetCache *tailnetCache
	localPassword string // generated once per daemon lifetime; non-empty in local mode
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
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
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
	// Use background context — the PTY must outlive the HTTP request.
	// r.Context() would kill the session when the response is sent.
	id, err := a.engine.CreateSession(context.Background(), req.CLI, req.Name, req.WorkDir, req.Args, req.Cols, req.Rows, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-enable web serving for this session if the web server is running (SERVE-02).
	a.mu.RLock()
	ws := a.webServer
	a.mu.RUnlock()
	if ws != nil {
		ws.EnableSession(id)
	}

	writeJSON(w, http.StatusCreated, CreateResponse{ID: id})
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
	} else {
		ws.DisableSession(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleTailnetPeers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	peers := a.tailnetCache.getOrRefresh(ctx, tailnet.DiscoverAndProbe)
	writeJSON(w, http.StatusOK, peers)
}
