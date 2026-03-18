package web

import "embed"

//go:embed dashboard.html terminal.html
var WebFS embed.FS
