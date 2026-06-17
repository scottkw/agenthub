package relay

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/files"
	"github.com/scottkw/agenthub/internal/pty"
)

// Server is an HTTP handler that exposes relay sessions over WebSocket.
type Server struct {
	manager *HubManager
	backend pty.SessionBackend
	mux     *http.ServeMux
}

// NewServer creates a Server backed by the given HubManager and SessionBackend
// and registers routes. backend is used to forward resize events from connected
// WebSocket clients to the underlying PTY.
//
// filesHandler, if non-nil, is mounted under /api/files/{list,stat,read} so the
// Wails desktop GUI (which reaches the daemon over TCP at 127.0.0.1:<relayPort>,
// NOT the Unix socket) can hit the read-only file API. The daemon's mux already
// registers the same routes for the Unix-socket transport; both surfaces share
// the same *files.Handler instance, so behaviour (sandbox enforcement, 5 MiB
// cap, etc.) is identical. Pass nil in tests that do not need file API coverage.
//
// Phase 120 CR-01 (REVIEW.md): before this change, FileBrowserTab in Wails mode
// 404'd on every /api/files/* call because the relay mux only knew /sessions/*.
//
// Phase 120.1 hotfix (CORS): the relay listens on 127.0.0.1 only, but a Wails
// webview's own origin (typically http://wails.localhost on macOS) is cross-
// origin from 127.0.0.1:<relayPort>. The browser sends a CORS preflight OPTIONS
// before the GET; without Access-Control-Allow-* headers the browser blocks
// the actual request. WebSocket relay traffic bypasses CORS preflight, which is
// why this gap was invisible until the first HTTP fetch (FileBrowserTab) was
// added. We allow loopback / Wails / Tauri origins only; relay is bound to
// 127.0.0.1 so the network boundary is already loopback-trusted.
func NewServer(manager *HubManager, backend pty.SessionBackend, filesHandler *files.Handler) *Server {
	s := &Server{
		manager: manager,
		backend: backend,
		mux:     http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /sessions/{id}/ws", s.handleSession)
	s.mux.HandleFunc("GET /sessions", s.handleListSessions)
	if filesHandler != nil {
		s.mux.HandleFunc("GET /api/files/list", withCORS(filesHandler.List))
		s.mux.HandleFunc("GET /api/files/stat", withCORS(filesHandler.Stat))
		s.mux.HandleFunc("GET /api/files/read", withCORS(filesHandler.Read))
		s.mux.HandleFunc("HEAD /api/files/read", withCORS(filesHandler.Read))
		s.mux.HandleFunc("OPTIONS /api/files/list", handleFilesPreflight)
		s.mux.HandleFunc("OPTIONS /api/files/stat", handleFilesPreflight)
		s.mux.HandleFunc("OPTIONS /api/files/read", handleFilesPreflight)
		// Write verbs (v3.5 Phases 123-125). These ride the same loopback-trusted
		// surface as the read routes above — the desktop GUI owner has full local
		// FS access, identical to the daemon unix socket (the opt-in files.write
		// capability gate is a webserver-only concern for web-share viewers, not
		// the local owner; internal/webserver/server.go). Omitting these is the
		// bug that made every desktop-GUI local save/upload/delete/rename/mkdir
		// 404 — see TestServer_FilesWriteAPI_MountedOnRelay.
		s.mux.HandleFunc("PUT /api/files/write", withCORS(filesHandler.Write))
		s.mux.HandleFunc("HEAD /api/files/write", withCORS(filesHandler.Write))
		s.mux.HandleFunc("POST /api/files/upload", withCORS(filesHandler.Upload))
		s.mux.HandleFunc("DELETE /api/files/delete", withCORS(filesHandler.Delete))
		s.mux.HandleFunc("POST /api/files/rename", withCORS(filesHandler.Rename))
		s.mux.HandleFunc("POST /api/files/mkdir", withCORS(filesHandler.Mkdir))
		s.mux.HandleFunc("OPTIONS /api/files/write", handleFilesPreflight)
		s.mux.HandleFunc("OPTIONS /api/files/upload", handleFilesPreflight)
		s.mux.HandleFunc("OPTIONS /api/files/delete", handleFilesPreflight)
		s.mux.HandleFunc("OPTIONS /api/files/rename", handleFilesPreflight)
		s.mux.HandleFunc("OPTIONS /api/files/mkdir", handleFilesPreflight)
	}
	return s
}

// isAllowedFilesOrigin returns true for loopback / Wails / Tauri webview
// origins. The relay listens on 127.0.0.1 only so this is defense-in-depth.
func isAllowedFilesOrigin(origin string) bool {
	if origin == "" {
		// Same-origin or non-browser caller (curl, Go HTTP client). Allow —
		// the relay's 127.0.0.1 bind is the actual network boundary.
		return true
	}
	switch origin {
	case "http://wails.localhost", "https://wails.localhost",
		"wails://wails", // macOS Wails v2 custom URL scheme (frontend.go:40)
		"http://tauri.localhost", "https://tauri.localhost",
		"http://localhost", "https://localhost",
		"http://127.0.0.1", "https://127.0.0.1",
		"null": // file:// pages and opaque custom-scheme origins report "null"
		return true
	}
	// Allow any wails:// origin (custom URL scheme on macOS — different Wails
	// versions/builds may vary the host segment).
	if len(origin) >= len("wails://") && origin[:len("wails://")] == "wails://" {
		return true
	}
	// Allow http(s)://127.0.0.1:<port> and http(s)://localhost:<port> with any port.
	for _, prefix := range []string{
		"http://127.0.0.1:", "https://127.0.0.1:",
		"http://localhost:", "https://localhost:",
	} {
		if len(origin) > len(prefix) && origin[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// withCORS wraps a handler to emit Access-Control-Allow-Origin on responses
// when the request Origin is in the loopback allowlist.
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isAllowedFilesOrigin(origin) && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		next(w, r)
	}
}

// handleFilesPreflight responds to CORS preflight (OPTIONS) for /api/files/*.
func handleFilesPreflight(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if !isAllowedFilesOrigin(origin) {
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Vary", "Origin")
	// Read verbs (GET/HEAD) plus the v3.5 write verbs (PUT/POST/DELETE). If-Match
	// is the optimistic-concurrency validator the editor sends on write (EDIT-05/08);
	// it must be advertised here or the browser blocks the PUT after preflight.
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, PUT, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range, If-Modified-Since, If-Match")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.WriteHeader(http.StatusNoContent)
}

// FilesCORS wraps a handler with the relay's loopback-origin CORS policy
// (Access-Control-Allow-Origin for allow-listed Wails/loopback origins).
// Exported so the daemon can apply identical CORS to the remote-files proxy
// routes it mounts on the relay surface — those handlers are *daemon.API
// methods and cannot live in this package without an import cycle, but they
// ride the same cross-origin webview→127.0.0.1:<relayPort> boundary as the
// local file routes and so need the same CORS treatment.
func FilesCORS(next http.HandlerFunc) http.HandlerFunc {
	return withCORS(next)
}

// FilesPreflight responds to a CORS preflight (OPTIONS) for /api/files/* on the
// relay surface. Exported for the daemon's remote-files proxy routes; advertises
// the same verb/header set as the local file routes (GET/HEAD/PUT/POST/DELETE,
// If-Match), so the cross-origin webview can issue remote writes.
func FilesPreflight(w http.ResponseWriter, r *http.Request) {
	handleFilesPreflight(w, r)
}

// ServeHTTP implements http.Handler by delegating to the internal mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleSession upgrades the HTTP connection to WebSocket and starts pumping
// messages between the client and its session Hub.
//
// Subscribe-before-snapshot ordering is critical here: we subscribe to the hub
// first, then replay the scrollback snapshot. Any frames written between snapshot
// time and the first live frame arrive via the Msgs channel — no frames are lost.
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	// MC-03, MC-05: parse client metadata from URL query params at upgrade time.
	readonly := r.URL.Query().Get("readonly") == "1" || r.URL.Query().Get("readonly") == "true"
	clientName := r.URL.Query().Get("client")
	if len(clientName) > 64 {
		clientName = clientName[:64] // cap identity name to prevent injection
	}

	hub, ok := s.manager.Get(sessionID)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Phase 88 (CONTEXT D-08/D-09): loopback-only Origin allowlist
		// replaces InsecureSkipVerify. The relay is bound to 127.0.0.1
		// by daemon/api.go, so in normal operation all clients are
		// loopback. This allowlist is belt-and-suspenders for the
		// landmine case where the listener is ever rebound to a
		// non-loopback interface — a future maintainer must consciously
		// add the new origin, which is the friction we want.
		//
		// Port derivation (RESEARCH Pitfall 7 option a): the Server
		// struct does not own its listener (daemon/api.go does), so we
		// read the port from r.Host. Empty port -> empty allowlist ->
		// library falls back to same-Host check, which is also correct
		// for a loopback-only deployment.
		OriginPatterns: loopbackOriginPatterns(r.Host),
	})
	if err != nil {
		// websocket.Accept already wrote an HTTP error response.
		return
	}

	ctx := r.Context()

	// Create subscriber with a buffered channel. CloseSlow disconnects the client
	// when the buffer fills up, preventing a slow client from blocking fan-out.
	sub := &Subscriber{
		Msgs:     make(chan []byte, 256),
		ReadOnly: readonly,
		Name:     clientName,
	}
	sub.CloseSlow = func() {
		conn.Close(websocket.StatusPolicyViolation, "too slow")
	}

	// Subscribe FIRST — anti-race pattern. Frames arrive in Msgs from now on,
	// so the snapshot taken below cannot cause a gap in the output stream.
	hub.Subscribe(sub)
	NotifyViewerCount(hub) // push updated viewer count to all clients
	defer func() {
		hub.Unsubscribe(sub)
		NotifyViewerCount(hub)
	}()
	defer conn.CloseNow()

	// Replay scrollback snapshot to bring the client up to date.
	if snapshot := hub.ScrollbackSnapshot(); len(snapshot) > 0 {
		if err := conn.Write(ctx, websocket.MessageBinary, snapshot); err != nil {
			return
		}
	}

	// Read pump — runs in a separate goroutine, parsing client -> PTY frames.
	//
	// Phase 115 / Issue #60: per-subscriber InputAbsorber filters OSC 10/11
	// + DA1 replies emitted by xterm.js in response to terminal queries
	// before they reach PTY stdin. The webserver-layer handleWSSRelay
	// applies the same absorber for the web-share path; this one catches
	// the daemon-direct attach path used by the Wails desktop GUI and by
	// CLI `agenthub attach`. State must persist across MsgInput frames
	// because envelopes may split across frames — the zero-value
	// &InputAbsorber{} is the legitimate "outside, empty buffers" state.
	readDone := make(chan struct{})
	absorber := &InputAbsorber{}
	go func() {
		defer close(readDone)
		for {
			_, msg, err := conn.Read(ctx)
			if err != nil {
				return
			}
			msgType, payload, err := ParseFrame(msg)
			if err != nil {
				continue
			}
			switch msgType {
			case MsgInput:
				if !sub.ReadOnly { // MC-03: discard input for read-only clients
					filtered := absorber.Filter(payload)
					if len(filtered) > 0 {
						_ = hub.WriteInput(filtered)
					}
				}
			case MsgResize2:
				if len(payload) >= 4 {
					cols := uint16(payload[0])<<8 | uint16(payload[1])
					rows := uint16(payload[2])<<8 | uint16(payload[3])
					_ = hub.ResizeClient(sub, int(cols), int(rows)) // MC-06: max-wins arbiter
				}
			case MsgPing:
				// Keep-alive — no-op.
			}
		}
	}()

	// Write pump — forwards Hub broadcast frames to the WebSocket client.
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

// loopbackOriginPatterns returns the allowlist of origin patterns
// authorised to open a relay WebSocket. Three classes of caller exist:
//
//  1. Browsers connecting via http(s)://127.0.0.1:<port> or
//     http(s)://localhost:<port> — the dashboard and (in future) any
//     locally-served HTML.
//  2. The Wails desktop webview, whose origin is wails://wails.localhost
//     (macOS, Linux) or http://wails.localhost (Windows). Production
//     builds have no port; dev builds expose a Vite HMR port. Both
//     forms must be matched. The wails.localhost host is reserved by
//     the Wails runtime and cannot be impersonated by an external
//     browser, so allow-listing it does not weaken the loopback
//     security boundary.
//  3. CLI / Go HTTP clients (no Origin header) — coder/websocket allows
//     these unconditionally per its Accept() docs ("empty Origin
//     header is allowed").
//
// If host cannot be split into host:port, the loopback IP-port
// patterns are omitted but the Wails patterns are kept — the daemon
// always binds the relay to 127.0.0.1 anyway, so the GUI must still
// reach it across the wails:// → 127.0.0.1 origin boundary.
//
// CONTEXT D-09: schemes included because the relay may be fronted by
// TLS in a future deployment; both "localhost" and "127.0.0.1" included
// because browsers and tools emit either form.
func loopbackOriginPatterns(host string) []string {
	// Production startURL by platform (Wails v2.12.0):
	//   macOS / Linux: wails://wails/         → Origin: wails://wails
	//   Windows:       http://wails.localhost → Origin: http://wails.localhost
	// Dev mode appends ".localhost" + the Vite HMR port:
	//   macOS / Linux: wails://wails.localhost:<port>
	//   Windows:       http://wails.localhost:<port>
	// All four combinations must be permitted; only `wails.localhost` and
	// `wails` are reserved Wails-runtime hosts that an external browser
	// cannot impersonate.
	wails := []string{
		"wails://wails",             // production macOS / Linux
		"wails://wails.localhost:*", // dev macOS / Linux
		"http://wails.localhost",    // production Windows
		"http://wails.localhost:*",  // dev Windows
	}
	_, port, err := net.SplitHostPort(host)
	if err != nil || port == "" {
		return wails
	}
	return append(wails,
		"http://localhost:"+port,
		"http://127.0.0.1:"+port,
		"https://localhost:"+port,
		"https://127.0.0.1:"+port,
	)
}

// LoopbackOriginPatterns is the exported wrapper around loopbackOriginPatterns,
// letting the daemon's remote-WS proxy (internal/daemon/remote_ws_proxy.go) reuse
// the exact same inbound-Origin allowlist that handleSession uses for the local
// relay WebSocket. It delegates to the unexported helper so the pattern slice is
// defined in exactly one place (T-134-06-01 spoofing mitigation: the proxy's
// inbound websocket.Accept must apply the same loopback/Wails allowlist).
func LoopbackOriginPatterns(host string) []string { return loopbackOriginPatterns(host) }

// NotifyViewerCount pushes a MsgMeta frame with the current viewer count
// to all subscribers. Must be called AFTER Subscribe/Unsubscribe returns
// (outside hub.mu) to avoid deadlock.
func NotifyViewerCount(hub *Hub) {
	count := hub.SubscriberCount()
	frame := MakeMeta(MetaPayload{ViewerCount: &count})
	hub.BroadcastMeta(frame)
}

// handleListSessions returns a JSON array of currently registered session IDs.
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	ids := s.manager.SessionIDs()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ids); err != nil {
		http.Error(w, "failed to encode sessions", http.StatusInternalServerError)
	}
}
