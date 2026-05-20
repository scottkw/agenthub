//go:build wailsassets

package main

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend/dist
var embeddedAssets embed.FS

var assets, _ = fs.Sub(embeddedAssets, "frontend/dist")

// staticAppForDaemon returns the embedded React bundle so the daemon process
// can mount it on the webserver's /app/ route (Phase 120-04). In prod builds
// the same FS underlies both the Wails desktop bundle and the webserver mount.
func staticAppForDaemon() fs.FS { return assets }
