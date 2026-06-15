package daemon

// Mounts the remote-files proxy routes on the relay loopback surface.
//
// Background (the v3.5 remote-files relay gap): the Wails desktop GUI reaches
// files over the relay loopback TCP server (127.0.0.1:<relayPort>), never the
// daemon Unix socket — the webview cannot reach the socket. Phase 120 mounted
// the LOCAL /api/files/* routes on both surfaces, but Phase 122 (remote read)
// and Phase 128 (remote write) registered the /api/files/remote/{sid}/...
// proxy routes ONLY on the Unix-socket mux (api.go:registerRoutes). The result
// was that every remote file op from the desktop GUI 404'd, and remote file
// access never worked there — blocking the two-machine tailnet UAT (#24).
//
// The proxy handlers (handleRemoteFiles*) are *API methods (they need the
// RemoteCapStore + outbound HTTPS client), so they cannot be registered inside
// relay.NewServer without an import cycle (daemon imports relay). Instead we
// wrap the relay server in a parent mux that owns the 9 remote routes and falls
// through ("/") to the relay server for everything else.
//
// CORS: the relay surface is cross-origin from the webview (wails://… →
// 127.0.0.1:<relayPort>), so the remote routes need the same Access-Control-*
// treatment as the local file routes. The proxy handlers emit no CORS of their
// own (they were Unix-socket-only by design), so we wrap them with
// relay.FilesCORS and register relay.FilesPreflight for the OPTIONS verb.

import (
	"net/http"

	"github.com/scottkw/agenthub/internal/relay"
)

// wrapRelayWithRemoteFiles returns a handler that serves the remote-files proxy
// routes (with relay CORS) and falls through to the given relay server for all
// other paths (sessions WS, local /api/files/*). See file header for why these
// routes live here rather than in relay.NewServer.
func (a *API) wrapRelayWithRemoteFiles(relayServer http.Handler) http.Handler {
	mux := http.NewServeMux()

	// The 9 remote proxy routes — the exact set the webview's FilesApiClient
	// hits with baseURL=http://127.0.0.1:<relayPort> + pathPrefix=
	// /api/files/remote/{sid}. Mirrors api.go:registerRoutes' Unix-socket
	// registrations, wrapped in relay.FilesCORS for the cross-origin webview.
	mux.HandleFunc("GET /api/files/remote/{sessionID}/list", relay.FilesCORS(a.handleRemoteFilesList))
	mux.HandleFunc("GET /api/files/remote/{sessionID}/stat", relay.FilesCORS(a.handleRemoteFilesStat))
	mux.HandleFunc("GET /api/files/remote/{sessionID}/read", relay.FilesCORS(a.handleRemoteFilesRead))
	mux.HandleFunc("HEAD /api/files/remote/{sessionID}/read", relay.FilesCORS(a.handleRemoteFilesRead))
	mux.HandleFunc("PUT /api/files/remote/{sessionID}/write", relay.FilesCORS(a.handleRemoteFilesWrite))
	mux.HandleFunc("POST /api/files/remote/{sessionID}/upload", relay.FilesCORS(a.handleRemoteFilesUpload))
	mux.HandleFunc("DELETE /api/files/remote/{sessionID}/delete", relay.FilesCORS(a.handleRemoteFilesDelete))
	mux.HandleFunc("POST /api/files/remote/{sessionID}/rename", relay.FilesCORS(a.handleRemoteFilesRename))
	mux.HandleFunc("POST /api/files/remote/{sessionID}/mkdir", relay.FilesCORS(a.handleRemoteFilesMkdir))

	// CORS preflight for the write verbs (read verbs are simple cross-origin
	// requests, but PUT/POST/DELETE + the If-Match header trigger a preflight
	// the browser blocks on without these). list/stat/read also get a preflight
	// route so a webview that preflights GETs (e.g. with a custom header) is not
	// blocked.
	mux.HandleFunc("OPTIONS /api/files/remote/{sessionID}/list", relay.FilesPreflight)
	mux.HandleFunc("OPTIONS /api/files/remote/{sessionID}/stat", relay.FilesPreflight)
	mux.HandleFunc("OPTIONS /api/files/remote/{sessionID}/read", relay.FilesPreflight)
	mux.HandleFunc("OPTIONS /api/files/remote/{sessionID}/write", relay.FilesPreflight)
	mux.HandleFunc("OPTIONS /api/files/remote/{sessionID}/upload", relay.FilesPreflight)
	mux.HandleFunc("OPTIONS /api/files/remote/{sessionID}/delete", relay.FilesPreflight)
	mux.HandleFunc("OPTIONS /api/files/remote/{sessionID}/rename", relay.FilesPreflight)
	mux.HandleFunc("OPTIONS /api/files/remote/{sessionID}/mkdir", relay.FilesPreflight)

	// Everything else (sessions WS, local /api/files/*) falls through to the
	// relay server. Go 1.22+ mux: the "/" pattern is the least-specific match,
	// so the explicit remote routes above always win for their paths.
	mux.Handle("/", relayServer)
	return mux
}
