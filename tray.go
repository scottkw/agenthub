//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa

#include <stdlib.h>

// Function declarations — implementations live in tray_objc.m (compiled once
// by cgo to avoid duplicate ObjC class symbols during go test linking).
void initStatusItem(const void *iconData, int iconLen);
void removeStatusItem(void);
void updateTrayIcon(const void *iconData, int iconLen);
void updateTrayTooltip(const char *tooltip);
void setTraySessionData(const char **names, const char **ids, int count);
void setDockVisible(int visible);
*/
import "C"

import (
	_ "embed"
	"fmt"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets/tray_icon.png
var trayIconBytes []byte

//go:embed assets/tray_icon_error.png
var trayIconErrorBytes []byte

// trayCallbackApp is the global App reference for cgo callbacks.
// Set before initTray returns. Only accessed from the main goroutine.
var trayCallbackApp *App

//export onTrayShow
func onTrayShow() {
	if trayCallbackApp != nil && trayCallbackApp.ctx != nil {
		runtime.WindowShow(trayCallbackApp.ctx)
	}
}

//export onTrayQuit
func onTrayQuit() {
	// Capture the pointer before launching the goroutine to avoid a data race
	// between the goroutine reading trayCallbackApp and tests restoring it.
	app := trayCallbackApp
	go func() {
		if app != nil && app.client != nil {
			_ = app.client.ShutdownDaemon()
		}
		if app != nil {
			app.quitting = true // bypass beforeClose hide-on-close
			if app.ctx != nil {
				runtime.Quit(app.ctx)
			}
		}
	}()
}

//export onTraySession
func onTraySession(idx C.int) {
	if trayCallbackApp == nil || trayCallbackApp.ctx == nil {
		return
	}
	// Hand off to goroutine — cgo callback thread is not safe for all Wails calls.
	go func() {
		sessions := trayCallbackApp.ListSessions()
		i := int(idx)
		if i >= 0 && i < len(sessions) {
			runtime.WindowShow(trayCallbackApp.ctx)
			runtime.EventsEmit(trayCallbackApp.ctx, "tray:focus-session", sessions[i].ID)
		}
	}()
}

// trayTooltip returns the tooltip string for the given session count.
// Uses an em dash (U+2014) as specified in the UI-SPEC copywriting contract.
func trayTooltip(n int) string {
	switch n {
	case 0:
		return "AgentHub \u2014 no sessions"
	case 1:
		return "AgentHub \u2014 1 session"
	default:
		return fmt.Sprintf("AgentHub \u2014 %d sessions", n)
	}
}

// setDockVisible shows or hides the macOS Dock icon by toggling the application
// activation policy. Call with true when showing the window, false when hiding.
func (a *App) setDockVisible(visible bool) {
	if visible {
		C.setDockVisible(1)
	} else {
		C.setDockVisible(0)
	}
}

// updateTray updates the tray icon, tooltip, and session list based on current state.
func (a *App) updateTray(sessions []SessionInfo, connected bool) {
	// Update icon based on connectivity.
	if connected {
		ptr := unsafe.Pointer(&trayIconBytes[0])
		C.updateTrayIcon(ptr, C.int(len(trayIconBytes)))
	} else {
		ptr := unsafe.Pointer(&trayIconErrorBytes[0])
		C.updateTrayIcon(ptr, C.int(len(trayIconErrorBytes)))
	}

	// Update tooltip with session count.
	tip := trayTooltip(len(sessions))
	ctip := C.CString(tip)
	C.updateTrayTooltip(ctip)
	C.free(unsafe.Pointer(ctip))

	// Update session names for menu delegate.
	n := len(sessions)
	if n == 0 {
		C.setTraySessionData(nil, nil, 0)
		return
	}
	cNames := make([]*C.char, n)
	cIDs := make([]*C.char, n)
	for i, s := range sessions {
		cNames[i] = C.CString(s.Name)
		cIDs[i] = C.CString(s.ID)
	}
	C.setTraySessionData(&cNames[0], &cIDs[0], C.int(n))
	for i := range cNames {
		C.free(unsafe.Pointer(cNames[i]))
		C.free(unsafe.Pointer(cIDs[i]))
	}
}

// initTray creates a macOS system tray icon with a dynamic menu delegate.
func (a *App) initTray() {
	trayCallbackApp = a
	ptr := unsafe.Pointer(&trayIconBytes[0])
	C.initStatusItem(ptr, C.int(len(trayIconBytes)))
}

// cleanupTray removes the status bar item.
func (a *App) cleanupTray() {
	C.removeStatusItem()
}
