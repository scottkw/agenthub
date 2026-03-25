//go:build wailsassets

package main

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend/dist
var embeddedAssets embed.FS

var assets, _ = fs.Sub(embeddedAssets, "frontend/dist")
