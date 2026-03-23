package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/agenthub/agenthub/internal/relay"
	"github.com/agenthub/agenthub/internal/webserver"
)

// API serves the daemon HTTP API over a Unix socket.
type API struct {
	engine    *SessionEngine
	mux       *http.ServeMux
	ln        net.Listener
	relayPort int               // TCP port the relay server is listening on
	relayLn   net.Listener      // TCP listener for the relay server
	mu        sync.RWMutex      // guards webServer
	webServer *webserver.WebServer // nil when not running
}

// NewAPI creates an API wired to the given SessionEngine and registers all routes.
func NewAPI(engine *SessionEngine) *API {
	a := &API{
		engine: engine,
		mux:    http.NewServeMux(),
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

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (a *API) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := a.engine.ListSessions()
	if sessions == nil {
		sessions = []SessionInfo{}
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (a *API) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	id, err := a.engine.CreateSession(r.Context(), req.CLI, req.Name, req.WorkDir, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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

	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP: req.IP,
		Port:   req.Port,
		FQDN:   req.FQDN,
	}, a.engine.Manager())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set session resolver so the web server can look up session metadata.
	ws.SetSessionResolver(func(sessionID string) (name, cliType, status string) {
		for _, s := range a.engine.ListSessions() {
			if s.ID == sessionID {
				return s.Name, s.CLI, a.engine.GetSessionStatus(sessionID)
			}
		}
		return sessionID, "", ""
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
	})
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
