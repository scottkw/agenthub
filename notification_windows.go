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
	if err := beeep.Notify(title, body, nil); err != nil {
		log.Printf("notification: delivery failed: %v", err)
	}
}
