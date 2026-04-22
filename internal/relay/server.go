package relay

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/coder/websocket"
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
func NewServer(manager *HubManager, backend pty.SessionBackend) *Server {
	s := &Server{
		manager: manager,
		backend: backend,
		mux:     http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /sessions/{id}/ws", s.handleSession)
	s.mux.HandleFunc("GET /sessions", s.handleListSessions)
	return s
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
	readDone := make(chan struct{})
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
					_ = hub.WriteInput(payload)
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

// loopbackOriginPatterns returns the 4-element allowlist of loopback
// origin patterns for the given host (from r.Host). If host cannot be
// split into host:port, returns nil — letting the library fall back to
// its same-Host default, which is still loopback-safe given the
// daemon-side bind to 127.0.0.1.
//
// CONTEXT D-09: schemes included because the relay may be fronted by
// TLS in a future deployment; both "localhost" and "127.0.0.1" included
// because browsers and tools emit either form.
func loopbackOriginPatterns(host string) []string {
	_, port, err := net.SplitHostPort(host)
	if err != nil || port == "" {
		return nil
	}
	return []string{
		"http://localhost:" + port,
		"http://127.0.0.1:" + port,
		"https://localhost:" + port,
		"https://127.0.0.1:" + port,
	}
}

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
