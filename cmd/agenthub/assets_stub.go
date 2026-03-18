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
