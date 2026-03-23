package daemon

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
)

// API serves the daemon HTTP API over a Unix socket.
type API struct {
	engine *SessionEngine
	mux    *http.ServeMux
	ln     net.Listener
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
