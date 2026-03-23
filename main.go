// AgentHub — Wails v2 desktop application.
//
// This file replaces the Phase 1 smoke-test binary.  The application presents
// a Wails window backed by the Go App struct (app.go) and a React frontend
// served from the embedded frontend/dist assets.
//
// Build for production:  wails build
// Run in dev mode:       wails dev
package main

import (
	"os"

	"github.com/agenthub/agenthub/internal/daemon"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "daemon" {
		daemon.RunDaemon()
		return
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "AgentHub",
		Width:             1200,
		Height:            800,
		MinWidth:          800,
		MinHeight:         600,
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 0x1b, G: 0x26, B: 0x36, A: 0xff},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.startup,
		OnShutdown:    app.shutdown,
		OnBeforeClose: app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
