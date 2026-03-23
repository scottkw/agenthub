package daemon

import (
	"testing"
	"time"
)

func TestClientHealth(t *testing.T) {
	_, client, _ := testDaemon(t)
	if err := client.Health(); err != nil {
		t.Errorf("Health: unexpected error: %v", err)
	}
}

func TestClientListSessions(t *testing.T) {
	_, client, _ := testDaemon(t)
	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if sessions == nil {
		t.Fatal("ListSessions returned nil, want empty slice")
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestClientCreateSession(t *testing.T) {
	_, client, _ := testDaemon(t)
	id, err := client.CreateSession("cat", "test-tab", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession returned empty ID")
	}
	t.Cleanup(func() { client.KillSession(id) })
}

func TestClientKillSession(t *testing.T) {
	_, client, _ := testDaemon(t)
	id, err := client.CreateSession("cat", "kill-me", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := client.KillSession(id); err != nil {
		t.Errorf("KillSession: %v", err)
	}
}

func TestClientRenameSession(t *testing.T) {
	_, client, _ := testDaemon(t)
	id, err := client.CreateSession("cat", "original", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { client.KillSession(id) })

	if err := client.RenameSession(id, "renamed"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}

	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions after rename: %v", err)
	}
	var found SessionInfo
	for _, s := range sessions {
		if s.ID == id {
			found = s
			break
		}
	}
	if found.ID == "" {
		t.Fatal("session not found after rename")
	}
	if found.Name != "renamed" {
		t.Errorf("name after rename: got %q, want %q", found.Name, "renamed")
	}
}

func TestClientGetSessionStatus(t *testing.T) {
	_, client, _ := testDaemon(t)
	id, err := client.CreateSession("cat", "status-tab", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { client.KillSession(id) })

	status, err := client.GetSessionStatus(id)
	if err != nil {
		t.Fatalf("GetSessionStatus: %v", err)
	}
	valid := map[string]bool{"running": true, "waiting": true, "idle": true, "errored": true}
	if !valid[status] {
		t.Errorf("GetSessionStatus returned invalid status %q", status)
	}
}

// TestClientFullLifecycle tests the full round-trip: create -> list -> rename -> list -> kill -> list.
func TestClientFullLifecycle(t *testing.T) {
	_, client, _ := testDaemon(t)

	// Create.
	id, err := client.CreateSession("cat", "tab-one", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// List — should have 1 session.
	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Name != "tab-one" {
		t.Errorf("name: got %q, want %q", sessions[0].Name, "tab-one")
	}

	// Rename.
	if err := client.RenameSession(id, "tab-renamed"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}

	// List — should show new name.
	sessions, err = client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions after rename: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Name != "tab-renamed" {
		t.Errorf("after rename: got %+v", sessions)
	}

	// Kill.
	if err := client.KillSession(id); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// Give the session a moment to be removed.
	time.Sleep(50 * time.Millisecond)

	// List — should be empty.
	sessions, err = client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions after kill: %v", err)
	}
	for _, s := range sessions {
		if s.ID == id {
			t.Errorf("killed session %q still in list", id)
		}
	}
}
