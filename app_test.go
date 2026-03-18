package main

import (
	"context"
	"net"
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
