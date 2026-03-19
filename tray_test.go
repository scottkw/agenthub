package main

import (
	"context"
	"testing"
)

// TestHideWindowSessionsAlive verifies that calling beforeClose does NOT kill
// any PTY sessions — they remain alive in the registry.  The system tray UI
// (systray package) is not exercised here because it requires a display server.
func TestHideWindowSessionsAlive(t *testing.T) {
	app := testApp(t)

	// Create two sessions using the always-available "cat" binary.
	_, err := app.CreateSession("cat", "tab-1", "")
	if err != nil {
		t.Fatalf("CreateSession tab-1: %v", err)
	}
	_, err = app.CreateSession("cat", "tab-2", "")
	if err != nil {
		t.Fatalf("CreateSession tab-2: %v", err)
	}

	// Verify 2 sessions are registered before the window hide.
	if app.registry.Len() != 2 {
		t.Fatalf("expected 2 sessions before beforeClose, got %d", app.registry.Len())
	}

	// Call beforeClose with a background context (Wails ctx not available in tests).
	// We use a minimal no-op context because runtime.WindowHide is a no-op outside Wails.
	_ = app.beforeClose(context.Background())

	// Sessions must still be alive — beforeClose must NOT kill them.
	if app.registry.Len() != 2 {
		t.Errorf("expected 2 sessions after beforeClose (window hide), got %d — sessions must survive window close", app.registry.Len())
	}
}

// TestBeforeCloseReturnsTrue verifies that beforeClose always returns true so
// that Wails suppresses its default quit behavior and keeps the process alive.
func TestBeforeCloseReturnsTrue(t *testing.T) {
	app := testApp(t)
	result := app.beforeClose(context.Background())
	if !result {
		t.Error("beforeClose returned false — must return true to prevent Wails from quitting the app")
	}
}
