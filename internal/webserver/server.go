package webserver

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/files"
	"github.com/scottkw/agenthub/internal/relay"
	webfs "github.com/scottkw/agenthub/web"
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
// Perms is populated only on the /info endpoint (from the verified capability
// claims, D-19/D-23 — terminal.html uses this to suppress the input caret on
// read-only capabilities). It is omitted on /api/sessions listings because a
// single-item self-describe response does not need to re-export the caller's
// own perms.
type sessionListItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	CLIType  string `json:"cli_type"`
	Status   string `json:"status"`
	Hostname string `json:"hostname"`
	Perms    string `json:"perms,omitempty"`
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

	// grants tracks the set of active grant_ids per session (D-14). Populated
	// on toggle-on by Plan 04, cleared on toggle-off or session exit. Guarded
	// by ws.mu.
	grants map[string]map[string]struct{}

	// signingKey is the 32-byte HMAC-SHA256 key used by requireCapability
	// (D-04/D-16). Guarded by ws.mu; swapped race-free via SetSigningKey.
	// Read via currentSigningKey which returns the slice header under RLock.
	signingKey []byte

	// joinCodes is the in-memory store of short-lived join codes (D-09/D-11).
	// Plan 04 calls SetJoinCodes at daemon startup; Plan 06 consumes the
	// manager in handleJoinExchange. Guarded by ws.mu; swapped race-free via
	// SetJoinCodes.
	joinCodes *capability.JoinCodeManager

	// sessionResolver is set once before Start() and is not mutex-protected.
	sessionResolver func(sessionID string) (name, cliType, status, hostname string)

	// pluginSettingsProvider returns pre-marshaled JSON for the daemon's
	// current plugin settings. Set once before Start() via
	// SetPluginSettingsProvider; not mutex-protected. Phase 93 PLUG-04.
	//
	// Uses func() []byte (not daemon.PluginSettings) to avoid the daemon→
	// webserver→daemon circular import (see 93-RESEARCH Q1).
	pluginSettingsProvider func() []byte

	// filesHandler is the *files.Handler injected by the daemon via
	// SetFilesHandler before ws.Start() runs. Stateless; the same instance
	// is shared with the daemon-local mux (internal/daemon/api.go::NewAPI).
	// Read at request time inside the route closure (Pitfall 2 — must NOT
	// be captured at registration time, since setupRoutes runs BEFORE the
	// daemon can wire it). Set once before Start(); not mutex-protected
	// (mirrors sessionResolver pattern, server.go:82-83).
	filesHandler *files.Handler

	// pluginConfigSubscribers is the set of active SSE subscribers for
	// /api/plugin-config/stream. Each subscriber gets a buffered channel;
	// BroadcastPluginConfig non-blocking-sends to each. Drop-on-slow-consumer.
	// Phase 93 PLUG-04 push channel — closes ROADMAP SC#4.
	pluginConfigMu          sync.RWMutex
	pluginConfigSubscribers map[chan []byte]struct{}
}

// NewWebServer creates a WebServer and sets up routes.
// Does NOT start the listener — call Start() to begin serving.
func NewWebServer(cfg Config, manager *relay.HubManager) (*WebServer, error) {
	ws := &WebServer{
		config:                  cfg,
		manager:                 manager,
		webEnabled:              make(map[string]bool),
		grants:                  make(map[string]map[string]struct{}),
		mux:                     http.NewServeMux(),
		pluginConfigSubscribers: make(map[chan []byte]struct{}),
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

// SetPluginSettingsProvider sets the callback used by handleGetPluginConfig
// to source the daemon's current plugin settings as pre-marshaled JSON.
// Plan 93-04 calls this at daemon startup so GET /api/plugin-config can
// serve capability-bearing web clients without a per-request RPC into the
// daemon. Phase 93 PLUG-04.
func (ws *WebServer) SetPluginSettingsProvider(fn func() []byte) {
	ws.pluginSettingsProvider = fn
}

// SetFilesHandler installs the *files.Handler used to serve the
// /api/files/{list,stat,read} routes on the webserver mux. Must be
// called before Start(). The handler must already be constructed with
// its sandbox resolver closure (the daemon's NewAPI does this — Phase
// 119 reuses a.filesHandler verbatim, no new construction). Mirrors
// SetSessionResolver — single setter, no mutex, set once.
func (ws *WebServer) SetFilesHandler(h *files.Handler) {
	ws.filesHandler = h
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

// AddGrant records grantID as an active grant for sessionID (D-14). Called by
// the daemon (Plan 04) after issuing a capability on toggle-on. Idempotent:
// adding an already-present grant is a no-op.
func (ws *WebServer) AddGrant(sessionID, grantID string) {
	ws.mu.Lock()
	if ws.grants[sessionID] == nil {
		ws.grants[sessionID] = make(map[string]struct{})
	}
	ws.grants[sessionID][grantID] = struct{}{}
	ws.mu.Unlock()
}

// ClearGrants removes every grant for sessionID (D-15). Called on toggle-off
// and by the session onExit callback (RESEARCH Pitfall 1). Safe to call when
// the session has no outstanding grants.
func (ws *WebServer) ClearGrants(sessionID string) {
	ws.mu.Lock()
	delete(ws.grants, sessionID)
	ws.mu.Unlock()
}

// isGrantActive reports whether grantID is currently in sessionID's active
// grant set. Read-only; uses the RLock path to avoid blocking concurrent
// requireCapability calls against other sessions.
func (ws *WebServer) isGrantActive(sessionID, grantID string) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	if ws.grants[sessionID] == nil {
		return false
	}
	_, ok := ws.grants[sessionID][grantID]
	return ok
}

// SetSigningKey installs the HMAC-SHA256 signing key used by subsequent
// requireCapability verifications (D-04/D-16). Plan 04 calls this at daemon
// startup; the RegenerateSigningKey handler calls it to rotate keys. The swap
// is race-free against concurrent currentSigningKey readers because both
// methods acquire ws.mu.
func (ws *WebServer) SetSigningKey(key []byte) {
	ws.mu.Lock()
	ws.signingKey = key
	ws.mu.Unlock()
}

// SetJoinCodes installs the join-code manager used by handleJoinExchange
// (D-09/D-11). Plan 04 wires this at daemon startup; Plan 06 consumes
// ws.joinCodes in the exchange handler. The swap is race-free against
// concurrent readers via ws.mu.
func (ws *WebServer) SetJoinCodes(jc *capability.JoinCodeManager) {
	ws.mu.Lock()
	ws.joinCodes = jc
	ws.mu.Unlock()
}

// currentSigningKey returns the current signing key under RLock. The returned
// slice header shares the backing array with the field; callers MUST NOT
// mutate it. SetSigningKey only ever reassigns the slice — the backing array
// is never mutated in place — so callers observing a pre-swap slice see
// stable bytes.
func (ws *WebServer) currentSigningKey() []byte {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.signingKey
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

// BindIP returns the server's configured bind IP. Used by the daemon's
// Tailscale-mode watcher to decide whether a config change is needed when
// Tailscale state transitions.
func (ws *WebServer) BindIP() string { return ws.config.BindIP }

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
//
// Route access model (Phase 87, SEC-02..SEC-05):
//   - /dashboard and / — open (landing page; Plan 06 rewrites the dashboard
//     to have no session list).
//   - /api/sessions, /api/sessions/{id}/info, /sessions/{id}, /sessions/{id}/ws
//     — gated by requireCapability. The session-bound cap enforces enumeration,
//     cross-session isolation, and write permission via claims.Perms (D-24).
//   - /api/sessions/{id}/qr — open (QR points at the session URL; the cap
//     token is baked into the URL, not needed to fetch the QR image).
//
// In local-network-fallback mode, basicAuthMiddleware wraps the entire mux at
// startLocal() and runs BEFORE these per-route wrappers (D-20 defense in
// depth).
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

	// GET /dashboard — open landing page (no session list per D-17, finalized
	// by Plan 06).
	mux.HandleFunc("GET /dashboard", ws.cspHeaders(ws.handleDashboard))

	// GET /join — open join-flow page (Plan 06, D-09). Reads ?code= from the
	// query string for pre-fill. Not capability-gated — it only collects the
	// code; the exchange itself runs on POST /join/exchange.
	mux.HandleFunc("GET /join", ws.cspHeaders(ws.handleJoin))

	// POST /join/exchange — open join-code exchange endpoint. Consumes a
	// single-use code and 303-redirects to /sessions/{id}?cap=<token>. Not
	// capability-gated — the code itself is the credential for this step
	// (D-09/D-11).
	mux.HandleFunc("POST /join/exchange", ws.handleJoinExchange)

	// GET /api/sessions — capability-gated; handleListSessions returns ONLY
	// the single session bound to the cap (D-18).
	mux.HandleFunc("GET /api/sessions", ws.requireCapability(ws.handleListSessions))

	// GET /api/sessions/{id}/info — capability-gated; cap must match {id}
	// (SEC-03). Used by the terminal page to populate status bar + perms
	// (D-19, D-23).
	mux.HandleFunc("GET /api/sessions/{id}/info", ws.requireCapability(ws.handleSessionInfo))

	// Phase 93 PLUG-04: capability-gated plugin config endpoint. Per
	// capability_mw.go path-ID logic, an empty path-ID short-circuits the
	// SID check, so any valid cap passes. Plugin config is global (not
	// session-specific) but the caller must still hold a verified cap.
	mux.HandleFunc("GET /api/plugin-config", ws.requireCapability(ws.handleGetPluginConfig))

	// Phase 93 PLUG-04 push channel — SSE stream of plugin-config changes.
	// Closes ROADMAP SC#4 ("no manual page reload for hot-swappable plugins").
	// Capability-gated like the read endpoint above.
	mux.HandleFunc("GET /api/plugin-config/stream", ws.requireCapability(ws.handleStreamPluginConfig))

	// Phase 119 / WEB-02..WEB-04: capability-gated read-only file API. The
	// closure captures ws (not ws.filesHandler) so the handler instance is
	// read at request time, AFTER the daemon has called SetFilesHandler.
	// Pitfall 2 — registering ws.filesHandler.List directly would bind nil
	// at setupRoutes() time. The same closure body is reused for all four
	// routes via a small helper to avoid four near-identical literal
	// blocks. Method-prefixed (GET/HEAD) per Go 1.22+ mux semantics — any
	// other verb returns 405 automatically without registering an
	// explicit handler (Pitfall 8 / WEB-02 SC#3). HEAD is registered as
	// a separate route because the Go 1.22 mux treats HEAD and GET as
	// distinct methods (Pitfall 1 / FS-06).
	filesDispatch := func(op func(*files.Handler) http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			h := ws.filesHandler
			if h == nil {
				http.Error(w, "files handler not configured", http.StatusServiceUnavailable)
				return
			}
			op(h)(w, r)
		}
	}
	mux.HandleFunc("GET /api/files/list", ws.requireFilesRead(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.List })))
	mux.HandleFunc("GET /api/files/stat", ws.requireFilesRead(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.Stat })))
	mux.HandleFunc("GET /api/files/read", ws.requireFilesRead(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.Read })))
	mux.HandleFunc("HEAD /api/files/read", ws.requireFilesRead(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.Read })))

	// GET /sessions/{id} — capability-gated terminal HTML page. The old
	// webEnabled-only pre-check is removed — requireCapability's grant-list
	// lookup already implies web-enabled (grants are cleared on toggle-off).
	mux.HandleFunc("GET /sessions/{id}",
		ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage)))

	// GET /sessions/{id}/ws — Origin allowlist + capability-gated WebSocket
	// upgrade. Phase 88 (D-10) wraps requireAllowedOrigin OUTSIDE
	// requireCapability so a cross-site Origin is rejected BEFORE any HMAC
	// verification work runs. The wrapper ordering matches composition
	// outermost->innermost: basicAuth (local only) -> requireAllowedOrigin
	// -> requireCapability -> handleWSSRelay.
	mux.HandleFunc("GET /sessions/{id}/ws",
		ws.requireAllowedOrigin(ws.requireCapability(ws.handleWSSRelay)))

	// GET /api/sessions/{id}/qr — serves QR code PNG. Open because the QR
	// encodes the capability-bearing URL; the cap itself lives in the URL.
	mux.HandleFunc("GET /api/sessions/{id}/qr", ws.handleSessionQR)

	// GET /assets/ and /assets/xterm/ — public static assets from the embedded web.WebFS.
	// Per D-14: http.FileServerFS mounted via fs.Sub. Two separate mounts because
	// vendored xterm files live at web/vendor/xterm/ on disk (user decision D-01/D-02)
	// but must be served under the /assets/xterm/ URL prefix (matches HTML references
	// from Plan 02 and keeps the URL space uniform). First-party extracted JS/CSS
	// live at web/assets/*. Go 1.22+ mux chooses the longest-matching pattern, so
	// /assets/xterm/xterm.js resolves via xtermFS and /assets/terminal.js via assetsFS.
	// Public tier (D-15): no capability gate. Cache-Control: no-store (D-16) so
	// xterm upgrades take effect immediately.
	assetsFS, err := fs.Sub(webfs.WebFS, "assets")
	if err != nil {
		panic(fmt.Sprintf("webserver: fs.Sub assets: %v", err))
	}
	xtermFS, err := fs.Sub(webfs.WebFS, "vendor/xterm")
	if err != nil {
		panic(fmt.Sprintf("webserver: fs.Sub vendor/xterm: %v", err))
	}
	mux.Handle("GET /assets/xterm/", http.StripPrefix("/assets/xterm/",
		assetsNoStore(http.FileServerFS(xtermFS))))
	mux.Handle("GET /assets/", http.StripPrefix("/assets/",
		assetsNoStore(http.FileServerFS(assetsFS))))
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

// handleJoin serves the embedded join.html landing page. The page reads
// ?code= and ?error= from the URL on the client side and shows the matching
// state variant (A=pre-filled, B=no-code, C=expired, D=invalid, E=session-gone).
// No capability required — the code input itself is the credential for the
// subsequent POST /join/exchange call.
func (ws *WebServer) handleJoin(w http.ResponseWriter, r *http.Request) {
	data, err := webfs.WebFS.ReadFile("join.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data) //nolint:errcheck
}

// handleJoinExchange consumes a single-use join code (D-09/D-11) and redirects
// to /sessions/{id}?cap=<token> via HTTP 303 See Other. Error contract:
//   - Empty or malformed body              -> 303 /join?error=invalid
//   - ErrCodeExpired                       -> 303 /join?error=expired
//   - ErrCodeNotFound                      -> 303 /join?error=invalid
//   - underlying verify / server error     -> 500
//   - session no longer web-enabled        -> 303 /join?error=session-gone
//
// Join-code 303 responses point at /join?error=<kind> so the user-facing UX
// (join.html State C/D/E) is the same shape regardless of whether the failure
// came from the browser back button, an expired code, or a revoked session.
func (ws *WebServer) handleJoinExchange(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/join?error=invalid", http.StatusSeeOther)
		return
	}
	code := r.FormValue("code")
	if code == "" {
		http.Redirect(w, r, "/join?error=invalid", http.StatusSeeOther)
		return
	}

	ws.mu.RLock()
	jc := ws.joinCodes
	ws.mu.RUnlock()
	if jc == nil {
		// Daemon never wired a JoinCodeManager — misconfiguration. Refuse rather
		// than silently accept: an un-wired manager means the server cannot
		// consume codes, so every request here is effectively invalid.
		http.Error(w, "join flow not available", http.StatusInternalServerError)
		return
	}

	token, err := jc.Exchange(code)
	switch {
	case errors.Is(err, capability.ErrCodeExpired):
		http.Redirect(w, r, "/join?error=expired", http.StatusSeeOther)
		return
	case errors.Is(err, capability.ErrCodeNotFound):
		http.Redirect(w, r, "/join?error=invalid", http.StatusSeeOther)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	key := ws.currentSigningKey()
	if key == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	claims, err := capability.Verify(token, key)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !ws.IsSessionEnabled(claims.SID) {
		http.Redirect(w, r, "/join?error=session-gone", http.StatusSeeOther)
		return
	}

	target := "/sessions/" + claims.SID + "?cap=" + token
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// handleListSessions handles GET /api/sessions. Per D-18 the response
// contains ONLY the single session that the caller's capability is bound to
// — there is no listing-scoped capability, so no caller can ever receive a
// list longer than one via HTTPS. This collapses the endpoint from
// "enumeration" to "self-describe".
//
// requireCapability has already verified claims and attached them to the
// request context; if the context does not carry claims, treat as 401
// (defense in depth — should be unreachable given the middleware ordering).
func (ws *WebServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := capability.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "capability required", http.StatusUnauthorized)
		return
	}
	items := make([]sessionListItem, 0, 1)
	if ws.IsSessionEnabled(claims.SID) {
		name, cliType, st, hostname := "", "", "", ""
		if ws.sessionResolver != nil {
			name, cliType, st, hostname = ws.sessionResolver(claims.SID)
		}
		if name == "" {
			name = claims.SID
		}
		items = append(items, sessionListItem{
			ID: claims.SID, Name: name, CLIType: cliType, Status: st, Hostname: hostname,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items) //nolint:errcheck
}

// handleSessionInfo handles GET /api/sessions/{id}/info.
// Returns full session metadata for a single web-enabled session. requireCapability
// has already verified the cap, matched its SID to the path {id}, and attached
// the Claims to the request context; we lift Perms out of the claims so the
// terminal page can fail-safely determine its read-only state from the
// server-verified capability (D-19 / D-23 / SEC-04).
func (ws *WebServer) handleSessionInfo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims, ok := capability.ClaimsFromContext(r.Context())
	if !ok {
		// Should be unreachable — requireCapability attaches claims on success.
		http.Error(w, "capability required", http.StatusUnauthorized)
		return
	}
	// requireCapability already cross-checked IsSessionEnabled before we got
	// here. Retain the resolver-based 404 for the "session ID isn't registered
	// with the engine" path (distinct from the 403 "revoked" response produced
	// by the middleware).
	if ws.sessionResolver == nil {
		http.NotFound(w, r)
		return
	}
	name, cliType, status, hostname := ws.sessionResolver(id)
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
		Perms:    claims.Perms,
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
//
// Phase 87: requireCapability has already verified the cap and attached the
// Claims to r.Context(). Subscriber.ReadOnly is sourced from claims.Perms
// (D-24 / SEC-04) — the old ?readonly=1 client-asserted hint has been
// removed from the write-gate path.
func (ws *WebServer) handleWSSRelay(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	// D-24 / SEC-04: readonly is bound to the signed capability, NOT the
	// query string. A caller with perms="read" cannot promote themselves to
	// write by omitting or fabricating ?readonly=; a caller with
	// perms="read,write" always has write even if someone appends
	// ?readonly=1 to the URL.
	claims, _ := capability.ClaimsFromContext(r.Context())
	readonly := claims.Perms == "read"

	// MC-05: client name is still a benign view hint from the query string.
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
		// Phase 88 (D-12) belt-and-suspenders: the requireAllowedOrigin
		// middleware already rejected wrong-Origin requests with 403
		// before we got here. Setting OriginPatterns to the same strict
		// allowlist ensures the library-layer check ALSO rejects if a
		// future route-wiring change ever bypasses the middleware.
		// ws.allowedOrigins() returns []string{ws.BaseURL()} — the
		// library does a case-insensitive path.Match against
		// u.Scheme+"://"+u.Host when the pattern contains "://", which
		// matches our canonical BaseURL form exactly.
		OriginPatterns: ws.allowedOrigins(),
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
	relay.NotifyViewerCount(hub) // push updated viewer count to all clients
	defer func() {
		hub.Unsubscribe(sub)
		relay.NotifyViewerCount(hub)
	}()
	defer conn.CloseNow()

	// Replay scrollback snapshot to bring the client up to date.
	if snapshot := hub.ScrollbackSnapshot(); len(snapshot) > 0 {
		if err := conn.Write(ctx, websocket.MessageBinary, snapshot); err != nil {
			return
		}
	}

	// Read pump — client → PTY
	readDone := make(chan struct{})
	absorber := &relay.InputAbsorber{} // Phase 111 / Issue #54: absorb OSC 10/11/DA1 replies. Phase 115: type moved to relay package.
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
					filtered := absorber.Filter(payload)
					if len(filtered) > 0 {
						_ = hub.WriteInput(filtered)
					}
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

// assetsNoStore wraps an http.Handler with Cache-Control: no-store (D-16) and
// blocks directory listing responses. http.FileServerFS returns an HTML directory
// index for requests ending with "/" when no index.html is present; we 404
// those requests because /assets/* serves individual files only.
// Scoped to /assets/* only — keeps embedded xterm + extracted JS/CSS
// fresh across deploys without content-hashing the URLs. Negligible
// bandwidth cost at single-page-load-per-session.
func assetsNoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block directory listing: any request whose (already-stripped) path
		// is empty or ends with "/" is a directory index request — return 404.
		if r.URL.Path == "" || r.URL.Path == "/" || len(r.URL.Path) > 0 && r.URL.Path[len(r.URL.Path)-1] == '/' {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
