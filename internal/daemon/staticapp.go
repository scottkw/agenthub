// Phase 120-04 Task 4 — daemon-side handle on the React frontend bundle.
//
// The embed.FS (`go:embed all:frontend/dist`) lives in package main and is
// gated behind the `wailsassets` build tag. The daemon process (a separate
// invocation of the same binary via `agenthub daemon`) cannot import package
// main, so main wires the FS into this package at startup via SetStaticAppFS.
//
// Both daemon code paths that construct a *webserver.WebServer (AutoStartWebServer
// + handleWebServerStart in api.go) read this package-level FS and pass it to
// ws.SetStaticAppFS, which mounts /app/ for remote/web-share viewers.
//
// When unset (dev build, `go test`, `go run` without wailsassets), this stays
// nil and the webserver responds 503 for /app/ — the expected dev fallback.

package daemon

import (
	"io/fs"
	"sync"
)

var (
	staticAppMu sync.RWMutex
	staticAppFS fs.FS // nil by default; main calls SetStaticAppFS at startup
)

// SetStaticAppFS sets the package-level React-bundle fs.FS that the daemon's
// web-start sites pass to webserver.SetStaticAppFS. Safe to call from any
// goroutine; the daemon reads under RLock.
func SetStaticAppFS(appFS fs.FS) {
	staticAppMu.Lock()
	staticAppFS = appFS
	staticAppMu.Unlock()
}

// getStaticAppFS returns the currently-wired bundle FS, or nil if not set.
func getStaticAppFS() fs.FS {
	staticAppMu.RLock()
	defer staticAppMu.RUnlock()
	return staticAppFS
}
