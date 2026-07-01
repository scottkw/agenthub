//go:build darwin

package main

import "testing"

// TestHasAppBundleID_FalseWhenUnbundled locks in the bundle-id detection
// primitive that the Go wrapper's log-and-swallow branch keys off. A `go
// test` binary has no .app wrapper, so [[NSBundle mainBundle]
// bundleIdentifier] is nil — this reproduces exactly the `wails dev`
// condition that crashed the GUI process in the Phase 167 M-41 regression.
func TestHasAppBundleID_FalseWhenUnbundled(t *testing.T) {
	if hasAppBundleID() {
		t.Fatal("hasAppBundleID() = true in a go-test binary; expected false (no app-bundle identifier for an unbundled process)")
	}
}

// TestSendNotification_NoBundleReturnsCleanly is a callable/smoke guard on
// the Go wrapper's early-return: calling sendNotification in this unbundled
// test process must return (not panic/abort) instead of reaching the
// crash-prone native UNUserNotificationCenter path.
//
// This proves the wrapper is safe to call in an unbundled process. It is NOT
// a substitute for the live M-41 delivery test — real UNUserNotificationCenter
// delivery + tray-hidden behavior can only be proven on a signed production
// build, not under `go test`, because the main dispatch queue is never pumped
// in a test binary.
func TestSendNotification_NoBundleReturnsCleanly(t *testing.T) {
	sendNotification("agenthub.test", "AgentHub", "waiting")
}

// TestRequestNotificationAuth_NoBundleReturnsCleanly is a callable/smoke
// guard proving the darwin proactive-authorization wrapper (Phase 167-06,
// M-41 gap closure) is safe in an unbundled test process — the native side
// bundle-guards and returns instead of reaching the crash-prone
// UNUserNotificationCenter authorization API.
//
// This is NOT a substitute for the live M-41 delivery test: real permission
// prompts, granted/denied outcomes, and delegate-driven foreground
// presentation can only be proven on a signed production build with a
// pumped main dispatch queue, which `go test` never provides.
func TestRequestNotificationAuth_NoBundleReturnsCleanly(t *testing.T) {
	requestNotificationAuth()
}
