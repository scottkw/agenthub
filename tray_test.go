//go:build darwin

package main

import (
	"bytes"
	"context"
	"image/png"
	"testing"
)

// TestTrayIconAsset verifies that both tray icon PNGs are valid 18x18 images.
func TestTrayIconAsset(t *testing.T) {
	// Verify tray_icon.png is valid and correct size
	if len(trayIconBytes) == 0 {
		t.Fatal("trayIconBytes is empty")
	}
	img, err := png.Decode(bytes.NewReader(trayIconBytes))
	if err != nil {
		t.Fatalf("tray_icon.png is not valid PNG: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 18 || bounds.Dy() != 18 {
		t.Errorf("tray_icon.png expected 18x18, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Verify error icon too
	if len(trayIconErrorBytes) == 0 {
		t.Fatal("trayIconErrorBytes is empty")
	}
	imgErr, err := png.Decode(bytes.NewReader(trayIconErrorBytes))
	if err != nil {
		t.Fatalf("tray_icon_error.png is not valid PNG: %v", err)
	}
	boundsErr := imgErr.Bounds()
	if boundsErr.Dx() != 18 || boundsErr.Dy() != 18 {
		t.Errorf("tray_icon_error.png expected 18x18, got %dx%d", boundsErr.Dx(), boundsErr.Dy())
	}
}

// TestHideWindowSessionsAlive verifies that calling beforeClose does NOT kill
// any PTY sessions — they remain alive in the daemon registry.  The system tray UI
// (systray package) is not exercised here because it requires a display server.
func TestHideWindowSessionsAlive(t *testing.T) {
	app := testApp(t)

	// Create two sessions using the always-available "cat" binary.
	_, err := app.CreateSession("cat", "tab-1", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession tab-1: %v", err)
	}
	_, err = app.CreateSession("cat", "tab-2", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession tab-2: %v", err)
	}

	// Verify 2 sessions are registered before the window hide.
	sessions, err := app.client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions before beforeClose: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions before beforeClose, got %d", len(sessions))
	}

	// Call beforeClose with a background context (Wails ctx not available in tests).
	// We use a minimal no-op context because runtime.WindowHide is a no-op outside Wails.
	_ = app.beforeClose(context.Background())

	// Sessions must still be alive — beforeClose must NOT kill them.
	sessions, err = app.client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions after beforeClose: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions after beforeClose (window hide), got %d — sessions must survive window close", len(sessions))
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

// TestTrayTooltip verifies the tooltip string formatting with em dash.
func TestTrayTooltip(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "AgentHub \u2014 no sessions"},
		{1, "AgentHub \u2014 1 session"},
		{3, "AgentHub \u2014 3 sessions"},
	}
	for _, tt := range tests {
		got := trayTooltip(tt.n)
		if got != tt.want {
			t.Errorf("trayTooltip(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// TestTrayQuitNilClient verifies onTrayQuit with nil app/client doesn't panic.
func TestTrayQuitNilClient(t *testing.T) {
	old := trayCallbackApp
	trayCallbackApp = nil
	defer func() { trayCallbackApp = old }()
	onTrayQuit()
	trayCallbackApp = &App{}
	onTrayQuit()
}

// TestRefreshTrayStateNilClient verifies refreshTrayState is safe with nil client.
func TestRefreshTrayStateNilClient(t *testing.T) {
	app := &App{trayInit: false, client: nil}
	app.refreshTrayState()
}

// TestRefreshTrayStateStartupFailure verifies that when trayInit=true but
// client=nil (daemon startup failure), refreshTrayState calls updateTray
// with connected=false (error icon) rather than returning early.
// The test verifies no panic occurs — the cgo call itself is the observable
// side-effect on darwin (updateTrayIcon sets the error icon PNG).
func TestRefreshTrayStateStartupFailure(t *testing.T) {
	app := &App{trayInit: true, client: nil}
	// Must not panic. On darwin, updateTray calls cgo updateTrayIcon with the
	// error icon bytes. On Linux/Windows the stub is a no-op.
	app.refreshTrayState()
}
