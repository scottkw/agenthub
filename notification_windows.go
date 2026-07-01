//go:build windows

package main

import (
	"log"

	"github.com/gen2brain/beeep"
)

// sendNotification sends a Windows toast notification via beeep (WinRT COM
// toast, falling back to a PowerShell/win32 path). Best-effort: errors are
// logged, never surfaced — mirrors the darwin implementation's contract
// (RESEARCH Pitfall 5). The identifier parameter is accepted for signature
// parity with darwin/linux but is not used by beeep (Windows AUMID branding
// is deferred).
func sendNotification(identifier, title, body string) {
	log.Printf("notification: dispatching beeep notification %q", title)
	if err := beeep.Notify(title, body, nil); err != nil {
		log.Printf("notification: delivery failed: %v", err)
	}
}

// requestNotificationAuth is the Windows no-op counterpart of the darwin
// proactive-authorization seam (Phase 167-06, M-41 gap closure). beeep needs
// no OS-level authorization step on Windows, so this exists only so
// NewApp's requestNotificationAuthFunc default compiles cross-platform.
func requestNotificationAuth() {}
