package daemon

import (
	"context"
	"testing"
	"time"
)

func TestNewSessionEngine(t *testing.T) {
	e := NewSessionEngine()
	if e == nil {
		t.Fatal("NewSessionEngine returned nil")
	}
	if e.registry == nil {
		t.Error("registry is nil")
	}
	if e.backend == nil {
		t.Error("backend is nil")
	}
	if e.manager == nil {
		t.Error("manager is nil")
	}
	if e.tabNames == nil {
		t.Error("tabNames is nil")
	}
	if e.cliPaths == nil {
		t.Error("cliPaths is nil")
	}
	if e.sessionStatuses == nil {
		t.Error("sessionStatuses is nil")
	}
}

func TestEngineCreateSession(t *testing.T) {
	e := NewSessionEngine()
	id, err := e.CreateSession(context.Background(), "cat", "test", "", nil, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession returned empty ID")
	}
	t.Cleanup(func() { _ = e.KillSession(id) })
}

func TestEngineListSessions(t *testing.T) {
	e := NewSessionEngine()

	id1, err := e.CreateSession(context.Background(), "cat", "tab-1", "", nil, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession tab-1: %v", err)
	}
	id2, err := e.CreateSession(context.Background(), "cat", "tab-2", "", nil, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession tab-2: %v", err)
	}
	t.Cleanup(func() {
		_ = e.KillSession(id1)
		_ = e.KillSession(id2)
	})

	sessions := e.ListSessions()
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	nameMap := make(map[string]string)
	for _, s := range sessions {
		nameMap[s.ID] = s.Name
	}
	if nameMap[id1] != "tab-1" {
		t.Errorf("session %s: expected name %q, got %q", id1, "tab-1", nameMap[id1])
	}
	if nameMap[id2] != "tab-2" {
		t.Errorf("session %s: expected name %q, got %q", id2, "tab-2", nameMap[id2])
	}
}

func TestEngineKillSession(t *testing.T) {
	e := NewSessionEngine()

	id, err := e.CreateSession(context.Background(), "cat", "kill-me", "", nil, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := e.KillSession(id); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// Give the session a moment to be removed.
	time.Sleep(50 * time.Millisecond)

	sessions := e.ListSessions()
	for _, s := range sessions {
		if s.ID == id {
			t.Errorf("killed session %q still appears in ListSessions", id)
		}
	}
}

func TestEngineRenameSession(t *testing.T) {
	e := NewSessionEngine()

	id, err := e.CreateSession(context.Background(), "cat", "original", "", nil, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	if err := e.RenameSession(id, "renamed"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}

	sessions := e.ListSessions()
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
		t.Errorf("expected name %q, got %q", "renamed", found.Name)
	}
}

func TestEngineGetSessionStatus(t *testing.T) {
	e := NewSessionEngine()

	// Unknown session returns "running" conservative default.
	s := e.GetSessionStatus("nonexistent-id")
	if s != "running" {
		t.Errorf("unknown session status: got %q, want %q", s, "running")
	}

	id, err := e.CreateSession(context.Background(), "cat", "status-test", "", nil, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	// Known session returns a valid status.
	status := e.GetSessionStatus(id)
	valid := map[string]bool{"running": true, "waiting": true, "idle": true, "errored": true}
	if !valid[status] {
		t.Errorf("GetSessionStatus returned invalid status %q", status)
	}
}

func TestEngineCreateSessionWithArgs(t *testing.T) {
	e := NewSessionEngine()
	id, err := e.CreateSession(context.Background(), "cat", "args-test", "", []string{"--version"}, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession with args: %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession with args returned empty ID")
	}
	t.Cleanup(func() { _ = e.KillSession(id) })
}

func TestEngineCreateSessionWithDimensions(t *testing.T) {
	e := NewSessionEngine()
	id, err := e.CreateSession(context.Background(), "cat", "dims-test", "", nil, 120, 40, nil)
	if err != nil {
		t.Fatalf("CreateSession with dimensions: %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession with dimensions returned empty ID")
	}
	t.Cleanup(func() { _ = e.KillSession(id) })
}

func TestEngineCreateSessionDefaultDimensions(t *testing.T) {
	e := NewSessionEngine()
	id, err := e.CreateSession(context.Background(), "cat", "default-dims", "", nil, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession with zero dims: %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession with zero dims returned empty ID")
	}
	t.Cleanup(func() { _ = e.KillSession(id) })
}

func TestEngineListSessionsHostname(t *testing.T) {
	e := NewSessionEngine()
	id, err := e.CreateSession(context.Background(), "cat", "h-eng", "", nil, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	sessions := e.ListSessions()
	if len(sessions) == 0 {
		t.Fatal("expected 1 session")
	}
	if sessions[0].Hostname == "" {
		t.Error("SessionInfo.Hostname empty — os.Hostname() must have failed or field not populated")
	}
}

func TestEngineResolveCLI(t *testing.T) {
	e := NewSessionEngine()

	// By default: returns name as-is.
	got := e.ResolveCLI("claude")
	if got != "claude" {
		t.Errorf("ResolveCLI default: got %q, want %q", got, "claude")
	}

	// After UpdateCLIPath: returns custom path.
	if err := e.UpdateCLIPath("claude", "/bin/cat"); err != nil {
		t.Fatalf("UpdateCLIPath: %v", err)
	}
	got = e.ResolveCLI("claude")
	if got != "/bin/cat" {
		t.Errorf("ResolveCLI after update: got %q, want %q", got, "/bin/cat")
	}
}
