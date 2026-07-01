//go:build linux

package main

import (
	"log"

	"github.com/gen2brain/beeep"
)

// sendNotification sends a Linux desktop notification via D-Bus (falling
// back to notify-send). Best-effort: errors are logged, never surfaced —
// mirrors the darwin implementation's contract (RESEARCH Pitfall 5). The
// identifier parameter is accepted for signature parity with darwin/windows
// but is not used by beeep (Linux app_name branding is deferred).
func sendNotification(identifier, title, body string) {
	log.Printf("notification: dispatching beeep notification %q", title)
	if err := beeep.Notify(title, body, nil); err != nil {
		log.Printf("notification: delivery failed: %v", err)
	}
}

// requestNotificationAuth is the Linux no-op counterpart of the darwin
// proactive-authorization seam (Phase 167-06, M-41 gap closure). beeep needs
// no OS-level authorization step on Linux, so this exists only so NewApp's
// requestNotificationAuthFunc default compiles cross-platform.
func requestNotificationAuth() {}
