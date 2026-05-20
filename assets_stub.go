//go:build !wailsassets

package main

import (
	"io/fs"
	"os"
)

// assets provides a minimal fs.FS for non-Wails builds (go test, go run).
// It points at the current directory; the package compiles and tests run without
// a real frontend build since no test exercises the asset server.
// When building with `wails build`, the wailsassets build tag activates
// assets_prod.go which uses //go:embed all:frontend/dist instead.
var assets fs.FS = os.DirFS(".")

// staticAppForDaemon returns nil in dev builds so the daemon's /app/ route
// answers 503 instead of accidentally exposing the working directory tree
// over the web. Phase 120-04 — see internal/daemon/staticapp.go.
func staticAppForDaemon() fs.FS { return nil }
