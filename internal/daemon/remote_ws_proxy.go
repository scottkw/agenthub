package daemon

// Phase 134-06 — Cap-gated remote terminal WebSocket reverse proxy.
//
// Phase 134's interactive modal reuses TerminalPanel/RelayClient, which for a
// LOCAL session attaches the local relay at ws://127.0.0.1:<relayPort>/sessions/
// {id}/ws. For a REMOTE session that session id only exists on the remote peer,
// so the local attach 404s (134-REVIEW CR-01/CR-02). The webview can only reach
// 127.0.0.1:<relayPort> — never the peer — so the daemon must proxy.
//
// This file adds GET /api/relay/remote/{sessionID}/ws on the relay loopback
// surface (mounted in relay_remote_files.go alongside the file-proxy routes). It
// mirrors the remote-files proxy exactly: look up the Phase 122 cap from the
// RemoteCapStore, dial the peer's already-cap-gated wss://<baseURL>/sessions/
// {sid}/ws?cap=T using the same InsecureSkipVerify tailnet transport
// (remoteFilesClient()), and copy opaque WS frames both ways. The cap stays
// server-side: the webview's URL carries NO cap (T-134-06-03 info-disclosure).
//
// Critical correctness gotchas (134-RESEARCH Pitfalls 1 & 4):
//  1. A Go WS dialer sends no Origin header; the peer's requireAllowedOrigin
//     rejects an empty Origin with 403. We inject Origin: <baseURL> byte-exact.
//  2. The bidirectional copy loop MUST use the long-lived request context, NOT
//     the 10s dial-timeout context — otherwise a healthy terminal dies after 10s.

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/relay"
)

// handleRemoteSessionWS proxies an inbound webview WebSocket to a remote peer's
// cap-gated terminal WS. It is a dumb pipe: the peer enforces HMAC cap
// verification, SID-match, grant-active, and read/write perms — this handler
// only forwards the cap (server-side) and copies frames.
func (a *API) handleRemoteSessionWS(w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("sessionID")
	if sid == "" {
		http.Error(w, "missing sessionID", http.StatusBadRequest)
		return
	}
	if a.remoteCaps == nil {
		http.Error(w, "remote cap store not initialised", http.StatusInternalServerError)
		return
	}

	// Cap lookup BEFORE the upgrade so the no-cap path returns the same JSON 404
	// contract as proxyRemoteFiles (the frontend re-prompts for a join code on
	// this marker). A bare route-miss would be Go's "404 page not found".
	baseURL, capToken, ok := a.remoteCaps.Get(sid)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no cap registered for session"})
		return
	}

	// 1. Accept the inbound (webview) WS. Reuse the relay's loopback/Wails Origin
	//    allowlist (T-134-06-01): the inbound conn is still loopback-only, so the
	//    same allowlist handleSession uses applies here. WS upgrades are not
	//    CORS-preflighted, so no cross-origin CORS wrapper is involved (Pattern 3).
	clientConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: relay.LoopbackOriginPatterns(r.Host),
	})
	if err != nil {
		// websocket.Accept already wrote the HTTP error response.
		return
	}
	defer clientConn.CloseNow()

	// 2. Build the upstream wss:// URL. RemoteCapStore.Put enforces an https://
	//    baseURL with a non-empty host, so swapping the scheme to wss is safe.
	u, perr := url.Parse(baseURL)
	if perr != nil {
		clientConn.Close(websocket.StatusInternalError, "bad base url")
		return
	}
	u.Scheme = "wss"
	u.Path = "/sessions/" + sid + "/ws"
	u.RawQuery = "cap=" + url.QueryEscape(capToken)

	// 3. Dial the peer with a BOUNDED dial context (10s) but the long-lived
	//    request context as its parent. CRITICAL (Pitfall 1): inject
	//    Origin: <baseURL> — the peer's requireAllowedOrigin rejects an empty
	//    Origin, which a Go WS dialer would otherwise send. Reuse the existing
	//    InsecureSkipVerify tailnet transport (do NOT build a new client).
	dialCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	hdr := http.Header{}
	hdr.Set("Origin", strings.TrimRight(baseURL, "/")) // byte-exact match to the peer's BaseURL()
	upstream, _, derr := websocket.Dial(dialCtx, u.String(), &websocket.DialOptions{
		HTTPClient: a.remoteFilesClient(),
		HTTPHeader: hdr,
	})
	if derr != nil {
		// Never surface the cap token in a close reason or log (T-134-06-03).
		// The reason string is a fixed literal; the redacted error is available
		// via redactCapTokenFromError(derr, capToken) if logging is added later.
		clientConn.Close(websocket.StatusTryAgainLater, "remote unreachable")
		return
	}
	defer upstream.CloseNow()

	// 4. Bidirectional opaque copy on the REQUEST context (Pitfall 4 — NOT the
	//    10s dialCtx, which would kill a healthy long-lived terminal). The first
	//    side to error/close tears down both via the deferred CloseNow calls.
	ctx := r.Context()
	errc := make(chan error, 2)
	go copyWS(ctx, upstream, clientConn, errc) // webview → peer (input/resize/ping)
	go copyWS(ctx, clientConn, upstream, errc) // peer → webview (scrollback/output)
	<-errc
}

// copyWS reads whole messages from src and writes them verbatim to dst until
// either side errors. The PTY frame protocol is end-to-end between xterm and the
// peer hub, so the proxy is transport-only: it never parses or evaluates frames.
func copyWS(ctx context.Context, dst, src *websocket.Conn, errc chan<- error) {
	for {
		typ, data, err := src.Read(ctx)
		if err != nil {
			errc <- err
			return
		}
		if err := dst.Write(ctx, typ, data); err != nil {
			errc <- err
			return
		}
	}
}
