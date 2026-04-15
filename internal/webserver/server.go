package webserver

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/scottkw/agenthub/internal/relay"
	webfs "github.com/scottkw/agenthub/web"
	"github.com/coder/websocket"
	qrcode "github.com/skip2/go-qrcode"
	"tailscale.com/client/local"
)

// Config holds configuration for the WebServer.
type Config struct {
	// BindIP is the IP address to bind the HTTPS listener on (Tailscale IP, e.g. 100.x.x.x).
	BindIP string
	// Port is the preferred HTTPS port. If 0 or unavailable, a random port is used.
	Port int
	// FQDN is the Tailscale MagicDNS hostname (from CertDomains[0]), used in BaseURL.
	FQDN string
	// TLSConfig is an override for tests; nil in production (uses lc.GetCertificate).
	TLSConfig *tls.Config
	// Mode controls which start path to use: "tailscale" (default) or "local".
	// In "local" mode the server uses a self-signed cert and HTTP Basic Auth.
	Mode string
	// Password is the HTTP Basic Auth password used when Mode == "local".
	// Must be non-empty when Mode is "local".
	Password string
}

// sessionListItem is the JSON shape returned by GET /api/sessions and GET /api/sessions/{id}/info.
type sessionListItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	CLIType  string `json:"cli_type"`
	Status   string `json:"status"`
	Hostname string `json:"hostname"`
}

// WebServer serves the AgentHub dashboard and relays terminal I/O over WSS to
// remote browser clients. Access control is provided at the network level by
// Tailscale — no application-layer authentication is required.
type WebServer struct {
	config  Config
	manager *relay.HubManager

	mu         sync.RWMutex
	webEnabled map[string]bool // sessionID -> enabled (WEB-01 toggle)
	listener   net.Listener
	mux        *http.ServeMux

	// sessionResolver is set once before Start() and is not mutex-protected.
	sessionResolver func(sessionID string) (name, cliType, status, hostname string)
}

// NewWebServer creates a WebServer and sets up routes.
// Does NOT start the listener — call Start() to begin serving.
func NewWebServer(cfg Config, manager *relay.HubManager) (*WebServer, error) {
	ws := &WebServer{
		config:     cfg,
		manager:    manager,
		webEnabled: make(map[string]bool),
		mux:        http.NewServeMux(),
	}
	ws.setupRoutes()
	return ws, nil
}

// SetSessionResolver sets the callback used by handleListSessions and
// handleSessionInfo to resolve session metadata (name, CLI type, status,
// hostname). Must be called before Start().
func (ws *WebServer) SetSessionResolver(fn func(string) (string, string, string, string)) {
	ws.sessionResolver = fn
}

// EnableSession marks a session as web-served (WEB-01 toggle).
func (ws *WebServer) EnableSession(sessionID string) {
	ws.mu.Lock()
	ws.webEnabled[sessionID] = true
	ws.mu.Unlock()
}

// DisableSession removes a session from web-serving (WEB-01 toggle).
func (ws *WebServer) DisableSession(sessionID string) {
	ws.mu.Lock()
	delete(ws.webEnabled, sessionID)
	ws.mu.Unlock()
}

// IsSessionEnabled returns true if the session is marked as web-enabled.
func (ws *WebServer) IsSessionEnabled(sessionID string) bool {
	ws.mu.RLock()
	ok := ws.webEnabled[sessionID]
	ws.mu.RUnlock()
	return ok
}

// webEnabledSessions returns a snapshot of all web-enabled session IDs.
func (ws *WebServer) webEnabledSessions() []string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	ids := make([]string, 0, len(ws.webEnabled))
	for id, enabled := range ws.webEnabled {
		if enabled {
			ids = append(ids, id)
		}
	}
	return ids
}

// Start opens the TLS listener and begins serving.
// Dispatches to startLocal() or startTailscale() based on Config.Mode.
func (ws *WebServer) Start() error {
	switch ws.config.Mode {
	case "local":
		return ws.startLocal()
	default:
		return ws.startTailscale()
	}
}

// Mode returns the server's configured mode ("tailscale" or "local").
func (ws *WebServer) Mode() string { return ws.config.Mode }

// startTailscale opens a TLS listener using Tailscale's lc.GetCertificate hook
// (or the TLSConfig test override). Falls back to a random port on EADDRINUSE.
func (ws *WebServer) startTailscale() error {
	tlsCfg := ws.config.TLSConfig
	if tlsCfg == nil {
		var lc local.Client
		tlsCfg = &tls.Config{
			GetCertificate: lc.GetCertificate,
			MinVersion:     tls.VersionTLS12,
		}
	}

	// Try configured port first; fall back to :0 (random) on EADDRINUSE.
	port := ws.config.Port
	addr := fmt.Sprintf("%s:%d", ws.config.BindIP, port)
	ln, err := tls.Listen("tcp", addr, tlsCfg)
	if err != nil {
		var opErr *net.OpError
		if errors.As(err, &opErr) && port != 0 {
			// Port in use — try random port
			addr = fmt.Sprintf("%s:0", ws.config.BindIP)
			ln, err = tls.Listen("tcp", addr, tlsCfg)
		}
		if err != nil {
			return fmt.Errorf("webserver: listen: %w", err)
		}
	}

	ws.mu.Lock()
	ws.listener = ln
	ws.mu.Unlock()

	go http.Serve(ln, ws.mux) //nolint:errcheck
	return nil
}

// startLocal opens a TLS listener using a self-signed certificate for
// ws.config.BindIP, wraps the mux with HTTP Basic Auth, and begins serving.
// Falls back to a random port on EADDRINUSE.
func (ws *WebServer) startLocal() error {
	tlsCfg, err := GenerateSelfSignedCert(ws.config.BindIP)
	if err != nil {
		return fmt.Errorf("webserver: generate self-signed cert: %w", err)
	}
	// Test override: if a TLSConfig was provided, use it instead.
	if ws.config.TLSConfig != nil {
		tlsCfg = ws.config.TLSConfig
	}

	// Try configured port first; fall back to :0 (random) on EADDRINUSE.
	port := ws.config.Port
	addr := fmt.Sprintf("%s:%d", ws.config.BindIP, port)
	ln, err := tls.Listen("tcp", addr, tlsCfg)
	if err != nil {
		var opErr *net.OpError
		if errors.As(err, &opErr) && port != 0 {
			// Port in use — try random port
			addr = fmt.Sprintf("%s:0", ws.config.BindIP)
			ln, err = tls.Listen("tcp", addr, tlsCfg)
		}
		if err != nil {
			return fmt.Errorf("webserver: listen: %w", err)
		}
	}

	ws.mu.Lock()
	ws.listener = ln
	ws.mu.Unlock()

	// Wrap the mux with Basic Auth so unauthenticated requests get 401.
	handler := basicAuthMiddleware(ws.config.Password)(ws.mux)
	go http.Serve(ln, handler) //nolint:errcheck
	return nil
}

// Stop closes the listener, stopping the HTTP server.
func (ws *WebServer) Stop() error {
	ws.mu.RLock()
	ln := ws.listener
	ws.mu.RUnlock()
	if ln != nil {
		return ln.Close()
	}
	return nil
}

// Addr returns the listener's network address (host:port).
func (ws *WebServer) Addr() string {
	ws.mu.RLock()
	ln := ws.listener
	ws.mu.RUnlock()
	if ln == nil {
		return ""
	}
	return ln.Addr().String()
}

// BaseURL returns the base HTTPS URL for the server.
// In "local" mode it uses the bind IP (e.g. https://192.168.1.50:7443).
// In "tailscale" mode (or when Mode is unset) it uses the FQDN (e.g. https://hostname.ts.net:7443).
func (ws *WebServer) BaseURL() string {
	ws.mu.RLock()
	ln := ws.listener
	ws.mu.RUnlock()
	if ln == nil {
		return ""
	}
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return ""
	}
	if ws.config.Mode == "local" {
		return fmt.Sprintf("https://%s:%s", ws.config.BindIP, port)
	}
	return fmt.Sprintf("https://%s:%s", ws.config.FQDN, port)
}

// setupRoutes registers all HTTP routes on the server mux.
// All routes are open — network-level access control is provided by Tailscale.
func (ws *WebServer) setupRoutes() {
	mux := ws.mux

	// GET / → redirect to /dashboard
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// GET /dashboard
	mux.HandleFunc("GET /dashboard", ws.handleDashboard)

	// GET /api/sessions
	mux.HandleFunc("GET /api/sessions", ws.handleListSessions)

	// GET /api/sessions/{id}/info — single-session metadata
	mux.HandleFunc("GET /api/sessions/{id}/info", ws.handleSessionInfo)

	// GET /sessions/{id} — checks web-enabled toggle only
	mux.HandleFunc("GET /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !ws.IsSessionEnabled(r.PathValue("id")) {
			http.NotFound(w, r)
			return
		}
		ws.handleTerminalPage(w, r)
	})

	// GET /sessions/{id}/ws — checks web-enabled toggle only; WebSocket upgrade
	mux.HandleFunc("GET /sessions/{id}/ws", func(w http.ResponseWriter, r *http.Request) {
		if !ws.IsSessionEnabled(r.PathValue("id")) {
			http.NotFound(w, r)
			return
		}
		ws.handleWSSRelay(w, r)
	})

	// GET /api/sessions/{id}/qr — serves QR code PNG
	mux.HandleFunc("GET /api/sessions/{id}/qr", ws.handleSessionQR)
}

// handleDashboard serves the embedded dashboard.html.
func (ws *WebServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := webfs.WebFS.ReadFile("dashboard.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data) //nolint:errcheck
}

// handleListSessions handles GET /api/sessions.
func (ws *WebServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
	ids := ws.webEnabledSessions()
	items := make([]sessionListItem, 0, len(ids))
	for _, id := range ids {
		name, cliType, st, hostname := "", "", "", ""
		if ws.sessionResolver != nil {
			name, cliType, st, hostname = ws.sessionResolver(id)
		}
		if name == "" {
			name = id
		}
		items = append(items, sessionListItem{ID: id, Name: name, CLIType: cliType, Status: st, Hostname: hostname})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items) //nolint:errcheck
}

// handleSessionInfo handles GET /api/sessions/{id}/info.
// Returns full session metadata for a single web-enabled session.
func (ws *WebServer) handleSessionInfo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !ws.IsSessionEnabled(id) {
		http.NotFound(w, r)
		return
	}
	if ws.sessionResolver == nil {
		http.NotFound(w, r)
		return
	}
	name, cliType, status, hostname := ws.sessionResolver(id)
	// If resolver returned defaults (name == id and cliType empty), session not found
	if name == id && cliType == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessionListItem{
		ID:       id,
		Name:     name,
		CLIType:  cliType,
		Status:   status,
		Hostname: hostname,
	}) //nolint:errcheck
}

// handleTerminalPage serves the embedded terminal.html.
func (ws *WebServer) handleTerminalPage(w http.ResponseWriter, r *http.Request) {
	data, err := webfs.WebFS.ReadFile("terminal.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data) //nolint:errcheck
}

// handleSessionQR handles GET /api/sessions/{id}/qr.
// Returns a QR code PNG for the session URL. Returns 404 if the session is not
// web-enabled.
func (ws *WebServer) handleSessionQR(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if !ws.IsSessionEnabled(sessionID) {
		http.NotFound(w, r)
		return
	}
	sessionURL := fmt.Sprintf("%s/sessions/%s", ws.BaseURL(), sessionID)
	png, err := qrcode.Encode(sessionURL, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "failed to generate QR code", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png) //nolint:errcheck
}

// handleWSSRelay upgrades to WebSocket and relays frames between the hub and the browser.
// Uses subscribe-before-snapshot pattern to avoid missing frames.
func (ws *WebServer) handleWSSRelay(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	// MC-03, MC-05: parse client metadata from URL query params at upgrade time.
	readonly := r.URL.Query().Get("readonly") == "1" || r.URL.Query().Get("readonly") == "true"
	clientName := r.URL.Query().Get("client")
	if len(clientName) > 64 {
		clientName = clientName[:64] // cap identity name to prevent injection
	}

	hub, ok := ws.manager.Get(sessionID)
	if !ok {
		// Session is web-enabled but not yet in the hub (e.g. just enabled, not started)
		// Accept the connection anyway and close cleanly
		http.Error(w, "session not found in hub", http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// The server is accessible only to tailnet members via Tailscale.
		// Accept connections from any origin.
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return
	}

	ctx := r.Context()

	sub := &relay.Subscriber{
		Msgs:     make(chan []byte, 256),
		ReadOnly: readonly,
		Name:     clientName,
	}
	sub.CloseSlow = func() {
		conn.Close(websocket.StatusPolicyViolation, "too slow")
	}

	// Subscribe FIRST — anti-race pattern.
	hub.Subscribe(sub)
	defer hub.Unsubscribe(sub)
	defer conn.CloseNow()

	// Replay scrollback snapshot to bring the client up to date.
	if snapshot := hub.ScrollbackSnapshot(); len(snapshot) > 0 {
		if err := conn.Write(ctx, websocket.MessageBinary, snapshot); err != nil {
			return
		}
	}

	// Read pump — client → PTY
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, msg, err := conn.Read(ctx)
			if err != nil {
				return
			}
			msgType, payload, err := relay.ParseFrame(msg)
			if err != nil {
				continue
			}
			switch msgType {
			case relay.MsgInput:
				if !sub.ReadOnly { // MC-03: discard input for read-only clients
					_ = hub.WriteInput(payload)
				}
			case relay.MsgResize2:
				if len(payload) >= 4 {
					cols := uint16(payload[0])<<8 | uint16(payload[1])
					rows := uint16(payload[2])<<8 | uint16(payload[3])
					_ = hub.ResizeClient(sub, int(cols), int(rows)) // MC-06: max-wins arbiter
				}
			case relay.MsgPing:
				// Keep-alive — no-op.
			}
		}
	}()

	// Write pump — hub → browser
	for {
		select {
		case frame := <-sub.Msgs:
			if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
				return
			}
		case <-ctx.Done():
			return
		case <-hub.Done():
			return
		case <-readDone:
			return
		}
	}
}
