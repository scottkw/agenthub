//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework UserNotifications

#include <stdlib.h>

void sendNotification(const char *title, const char *body);
*/
import "C"

import "unsafe"

// sendNotification sends a macOS notification using UNUserNotificationCenter.
// Called by QuitGUIOnly to inform the user the app is still running (D-11).
func sendNotification(title, body string) {
	ctitle := C.CString(title)
	cbody := C.CString(body)
	C.sendNotification(ctitle, cbody)
	C.free(unsafe.Pointer(ctitle))
	C.free(unsafe.Pointer(cbody))
}
