package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testApp creates an App wired for testing — no Wails GUI, but all bound
// methods are functional.  It opens a real TCP listener on 127.0.0.1:0 to
// simulate what startup() does.
func testApp(t *testing.T) *App {
	t.Helper()
	app := NewApp()

	// Set context — startup() is not called in tests, so we provide a background context.
	app.ctx = context.Background()

	// Simulate the startup listener allocation (without running wails.Run).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("testApp: net.Listen: %v", err)
	}
	app.listener = ln
	t.Cleanup(func() {
		ln.Close()
		app.manager.Shutdown()
	})
	return app
}

func TestListSessionsEmpty(t *testing.T) {
	app := testApp(t)
	sessions := app.ListSessions()
	if sessions == nil {
		t.Fatal("ListSessions on fresh App returned nil, want empty slice")
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestCreateSession(t *testing.T) {
	app := testApp(t)
	id, err := app.CreateSession("cat", "test-tab")
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession returned empty ID")
	}

	sessions := app.ListSessions()
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != id {
		t.Errorf("session ID mismatch: got %q, want %q", sessions[0].ID, id)
	}
	if sessions[0].Name != "test-tab" {
		t.Errorf("session name mismatch: got %q, want %q", sessions[0].Name, "test-tab")
	}
}

func TestRenameSession(t *testing.T) {
	app := testApp(t)
	id, err := app.CreateSession("cat", "original")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := app.RenameSession(id, "renamed"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}

	sessions := app.ListSessions()
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Name != "renamed" {
		t.Errorf("expected name %q, got %q", "renamed", sessions[0].Name)
	}
}

func TestKillSession(t *testing.T) {
	app := testApp(t)
	id, err := app.CreateSession("cat", "kill-me")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := app.KillSession(id); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// Give the session a moment to be removed.
	time.Sleep(50 * time.Millisecond)

	sessions := app.ListSessions()
	for _, s := range sessions {
		if s.ID == id {
			t.Errorf("killed session %q still appears in ListSessions", id)
		}
	}
}

func TestDetectCLIs(t *testing.T) {
	app := testApp(t)
	clis := app.DetectCLIs()
	if clis == nil {
		t.Fatal("DetectCLIs returned nil, want non-nil slice")
	}
}

func TestUpdateCLIPath(t *testing.T) {
	app := testApp(t)

	// /bin/cat is a guaranteed path on macOS/Linux.
	if err := app.UpdateCLIPath("claude", "/bin/cat"); err != nil {
		t.Fatalf("UpdateCLIPath: %v", err)
	}

	// Now create a session with "claude" — it should resolve to /bin/cat.
	id, err := app.CreateSession("claude", "custom-path-tab")
	if err != nil {
		t.Fatalf("CreateSession with custom path: %v", err)
	}

	sessions := app.ListSessions()
	var found SessionInfo
	for _, s := range sessions {
		if s.ID == id {
			found = s
			break
		}
	}
	if found.ID == "" {
		t.Fatal("session not found in ListSessions after CreateSession")
	}
}

func TestGetRelayPort(t *testing.T) {
	app := testApp(t)
	port := app.GetRelayPort()
	if port <= 0 {
		t.Errorf("GetRelayPort returned %d, want port > 0", port)
	}
}

// testAppWithConfigDir returns an App with an isolated temp config directory
// so web server tests don't pollute ~/.config/agenthub.
func testAppWithConfigDir(t *testing.T) (*App, string) {
	t.Helper()
	dir := t.TempDir()

	// Override configDir resolution by pointing the config to a temp path.
	// We use an env var approach: set XDG_CONFIG_HOME so os.UserConfigDir returns our dir.
	t.Setenv("XDG_CONFIG_HOME", dir)
	// On macOS, os.UserConfigDir uses $HOME/Library/Application Support, not XDG.
	// Override by pointing HOME to dir so configDir() returns dir/agenthub.
	// Simpler: just call functions directly and use the temp dir as the override.

	app := testApp(t)
	return app, filepath.Join(dir, "agenthub")
}

func TestSetWebPasswordPersistsAndReloads(t *testing.T) {
	// Use a temp dir to isolate password persistence.
	dir := t.TempDir()
	// Override HOME so configDir() returns dir + "/agenthub" (macOS uses Library subdir).
	// Instead, we directly test the full round-trip by writing/reading using the same path.
	hashPath := filepath.Join(dir, "web_password")

	// Write a known bcrypt hash manually using the same logic as SetWebPassword.
	// We can't override configDir() directly, so we test the underlying mechanism:
	// SetWebPassword calls os.WriteFile(webPasswordPath(), ...) and IsWebPasswordSet
	// checks os.Stat(webPasswordPath()).
	//
	// For integration, create the app and temporarily redirect the config dir.
	t.Setenv("HOME", dir)

	app := testApp(t)

	// Initially, no password set.
	if app.IsWebPasswordSet() {
		t.Error("expected IsWebPasswordSet to be false before SetWebPassword")
	}

	if err := app.SetWebPassword("testpassword123"); err != nil {
		t.Fatalf("SetWebPassword: %v", err)
	}

	// Check that a hash file was written somewhere under dir.
	// The actual path will be dir/Library/... on macOS or dir/.config/agenthub/...
	// We verify via IsWebPasswordSet which should read from disk.
	if !app.IsWebPasswordSet() {
		t.Error("expected IsWebPasswordSet to be true after SetWebPassword")
	}

	// Verify the web_password file exists somewhere under the temp dir.
	found := false
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if filepath.Base(path) == "web_password" {
			found = true
			hashPath = path
		}
		return nil
	})
	if !found {
		t.Fatal("web_password file not created under temp HOME")
	}
	_ = hashPath // verified it exists
}

func TestGetNetworkInterfaces(t *testing.T) {
	app := testApp(t)
	ifaces := app.GetNetworkInterfaces()
	// GetNetworkInterfaces may return empty on minimal CI environments.
	// The important thing is it does NOT return nil and does not panic.
	if ifaces == nil {
		t.Error("GetNetworkInterfaces returned nil, want non-nil slice")
	}
}

func TestToggleWebServingErrorsWhenNotRunning(t *testing.T) {
	app := testApp(t)
	// webServer is nil — ToggleWebServing should return an error.
	err := app.ToggleWebServing("some-session-id", true)
	if err == nil {
		t.Error("expected ToggleWebServing to return error when web server is not running")
	}
}

func TestStartWebServerErrorsWhenPasswordNotSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	app := testApp(t)
	// No password set — StartWebServer should return an error.
	err := app.StartWebServer("127.0.0.1", 0)
	if err == nil {
		t.Error("expected StartWebServer to return error when password is not set")
	}
}

func TestIsWebServerRunning(t *testing.T) {
	app := testApp(t)
	if app.IsWebServerRunning() {
		t.Error("expected IsWebServerRunning to be false before StartWebServer")
	}
}
