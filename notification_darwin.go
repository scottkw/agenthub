//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework UserNotifications

#include <stdlib.h>

int hasValidBundleIdentifier(void);
void sendNotification(const char *identifier, const char *title, const char *body);
*/
import "C"

import (
	"log"
	"unsafe"
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
	cid := C.CString(identifier)
	ctitle := C.CString(title)
	cbody := C.CString(body)
	C.sendNotification(cid, ctitle, cbody)
	C.free(unsafe.Pointer(cid))
	C.free(unsafe.Pointer(ctitle))
	C.free(unsafe.Pointer(cbody))
}
