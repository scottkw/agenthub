//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework UserNotifications

#include <stdlib.h>

void sendNotification(const char *identifier, const char *title, const char *body);
*/
import "C"

import "unsafe"

// sendNotification sends a macOS notification using UNUserNotificationCenter.
// Called by QuitGUIOnly (fixed identifier) and, from Plan 03, per-session
// waiting notifications. The identifier is threaded through to
// UNNotificationRequest so concurrent notifications don't collapse into one
// another (RESEARCH Pitfall 2).
func sendNotification(identifier, title, body string) {
	cid := C.CString(identifier)
	ctitle := C.CString(title)
	cbody := C.CString(body)
	C.sendNotification(cid, ctitle, cbody)
	C.free(unsafe.Pointer(cid))
	C.free(unsafe.Pointer(ctitle))
	C.free(unsafe.Pointer(cbody))
}
