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
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets/tray_icon.png
var trayIconBytes []byte

//go:embed assets/tray_icon_error.png
var trayIconErrorBytes []byte

//go:embed assets/tray_icon_progress_25.png
var trayIconProgress25Bytes []byte

//go:embed assets/tray_icon_progress_50.png
var trayIconProgress50Bytes []byte

//go:embed assets/tray_icon_progress_75.png
var trayIconProgress75Bytes []byte

//go:embed assets/tray_icon_progress_100.png
var trayIconProgress100Bytes []byte

// trayCallbackApp is the global App reference for cgo callbacks.
// Set before initTray returns. Only accessed from the main goroutine.
var trayCallbackApp *App

//export onTrayShow
func onTrayShow() {
	if trayCallbackApp != nil && trayCallbackApp.ctx != nil {
		runtime.WindowShow(trayCallbackApp.ctx)
		trayCallbackApp.setDockVisible(true)
	}
}

//export onTrayQuit
func onTrayQuit() {
	// Capture the pointer before launching the goroutine to avoid a data race
	// between the goroutine reading trayCallbackApp and tests restoring it.
	app := trayCallbackApp
	go func() {
		if app != nil && app.ctx != nil {
			runtime.WindowShow(app.ctx)      // D-08: auto-show if hidden
			app.setDockVisible(true)
			runtime.EventsEmit(app.ctx, "app:quit-requested", nil)
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
			trayCallbackApp.setDockVisible(true)
			runtime.EventsEmit(trayCallbackApp.ctx, "tray:focus-session", sessions[i].ID)
		}
	}()
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

// trayIconBytesForState returns the appropriate tray icon byte slice for the
// given connection state and current progress quartile (Phase 98 PRG-03).
//
// Error precedence (Pitfall #8): when connected=false, always returns
// trayIconErrorBytes regardless of a.lastTrayQuartile to ensure daemon-
// disconnect is not masked by a progress glyph.
//
// This helper is defined verbatim in tray.go (darwin), tray_linux.go, and
// tray_windows.go — three identical copies required because each file has its
// own //go:build tag and the trayIconProgress* byte slices embedded in each
// file are not visible across build-tag boundaries.
func (a *App) trayIconBytesForState(connected bool) []byte {
	if !connected {
		return trayIconErrorBytes
	}
	switch a.lastTrayQuartile {
	case 1:
		return trayIconProgress25Bytes
	case 2:
		return trayIconProgress50Bytes
	case 3:
		return trayIconProgress75Bytes
	case 4:
		return trayIconProgress100Bytes
	default:
		return trayIconBytes
	}
}

// updateTray updates the tray icon, tooltip, and session list based on current state.
func (a *App) updateTray(sessions []SessionInfo, connected bool) {
	// Update icon based on connectivity and progress quartile.
	bytes := a.trayIconBytesForState(connected)
	ptr := unsafe.Pointer(&bytes[0])
	C.updateTrayIcon(ptr, C.int(len(bytes)))

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
