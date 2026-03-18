package main

import (
	_ "embed"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets/appicon.png
var trayIconBytes []byte

// initTray starts the system tray using RunWithExternalLoop so that the tray
// runs alongside (not blocking) the Wails event loop.  It is called from
// startup() after the relay listener is ready.
func (a *App) initTray() {
	start, end := systray.RunWithExternalLoop(a.onTrayReady, a.onTrayExit)
	a.trayEnd = end
	start()
}

// onTrayReady sets up the tray icon and menu items.  It runs inside the systray
// event loop goroutine — do not call Wails runtime functions directly here;
// use goroutines for click handlers.
func (a *App) onTrayReady() {
	systray.SetIcon(trayIconBytes)
	systray.SetTooltip("AgentHub")

	mShow := systray.AddMenuItem("Show AgentHub", "Show the AgentHub window")
	mQuit := systray.AddMenuItem("Quit", "Quit AgentHub")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				runtime.WindowShow(a.ctx)
			case <-mQuit.ClickedCh:
				systray.Quit()
				runtime.Quit(a.ctx)
				return
			}
		}
	}()
}

// onTrayExit is called when the systray loop exits.  Cleanup is handled by
// shutdown() via the trayEnd function; nothing additional needed here.
func (a *App) onTrayExit() {}
