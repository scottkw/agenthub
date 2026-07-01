//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework UserNotifications

#include <stdlib.h>

int hasValidBundleIdentifier(void);
void sendNotification(const char *identifier, const char *title, const char *body);
void requestNotificationAuthorization(void);
*/
import "C"

import (
	"log"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// hasAppBundleID reports whether the running process has a valid app-bundle
// identifier. A go-test binary is itself unbundled, so this returns false in
// tests, exactly reproducing the `wails dev` condition that crashed the GUI
// process (Phase 167 M-41 regression).
func hasAppBundleID() bool {
	return C.hasValidBundleIdentifier() != 0
}

// sendNotification sends a macOS notification using UNUserNotificationCenter.
// Called by QuitGUIOnly (fixed identifier) and, from Plan 03, per-session
// waiting notifications. The identifier is threaded through to
// UNNotificationRequest so concurrent notifications don't collapse into one
// another (RESEARCH Pitfall 2).
//
// Fail-safe (Phase 167-05 gap closure, M-41): if there is no valid app-bundle
// identifier (e.g. `wails dev`, a go-test binary), log-and-swallow instead of
// calling into the native path — mirrors the beeep Windows/Linux wrappers'
// contract (notification_windows.go, notification_linux.go).
func sendNotification(identifier, title, body string) {
	if !hasAppBundleID() {
		log.Printf("notification: skipping — no valid app-bundle identifier (unbundled/unsigned process)")
		return
	}
	log.Printf("notification: dispatching native notification for identifier %s", identifier)
	cid := C.CString(identifier)
	ctitle := C.CString(title)
	cbody := C.CString(body)
	C.sendNotification(cid, ctitle, cbody)
	C.free(unsafe.Pointer(cid))
	C.free(unsafe.Pointer(ctitle))
	C.free(unsafe.Pointer(cbody))
}

// requestNotificationAuth is the darwin implementation of the cross-platform
// proactive-authorization seam (app.go's requestNotificationAuthFunc). Called
// when the user enables the NotifyOnWaiting toggle (Phase 167-06, M-41 gap
// closure) so the macOS permission prompt surfaces at toggle-time instead of
// only lazily on the first waiting transition.
func requestNotificationAuth() {
	log.Printf("notification: requesting proactive native authorization")
	C.requestNotificationAuthorization()
}

//export onNotificationAuthResult
func onNotificationAuthResult(granted C.int) {
	log.Printf("notification: proactive authorization result granted=%d", int(granted))
	// Emit denied OR granted so the Settings hint can self-heal (Phase 167 WR-01):
	// a bare return on grant left a stale denied-hint stuck for the session after
	// the user fixed permissions and re-toggled. Hand off to a goroutine — the
	// completion-handler thread is not safe for all Wails calls (mirrors
	// onTraySession in tray.go).
	event := "notification:permission-granted"
	if granted == 0 {
		event = "notification:permission-denied"
	}
	go func() {
		if appInstance != nil && appInstance.ctx != nil {
			runtime.EventsEmit(appInstance.ctx, event, nil)
		}
	}()
}
