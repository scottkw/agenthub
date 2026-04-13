package daemon

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/pty"
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
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix path /bin/cat")
	}
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

// spyBackend records the CreateRequest from the most recent Create call.
// Used by Wave 0 tests to assert on env injection without launching a real PTY.
type spyBackend struct {
	lastReq pty.CreateRequest
}

func (s *spyBackend) Create(_ context.Context, req pty.CreateRequest) (*pty.Session, error) {
	s.lastReq = req
	return &pty.Session{
		ID:        "spy-id",
		CLI:       req.CLI,
		State:     pty.StateRunning,
		CreatedAt: time.Now(),
	}, nil
}

func (s *spyBackend) Resize(string, int, int) error { return nil }
func (s *spyBackend) Kill(string) error             { return nil }
func (s *spyBackend) List() []*pty.Session           { return nil }

// TestCreateSession_OpenCodeEnv asserts that CreateSession injects
// OPENCODE_TUI_CONFIG into the PTY environment when cli == "opencode".
// Wave 0: RED state — current engine.go does not set CreateRequest.Env.
func TestCreateSession_OpenCodeEnv(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	_, err := e.CreateSession(context.Background(), "opencode", "test-oc", "", nil, 80, 24, nil)
	if err != nil {
		t.Fatalf("CreateSession(opencode): %v", err)
	}

	// Assert OPENCODE_TUI_CONFIG was injected into the env.
	var found bool
	for _, entry := range spy.lastReq.Env {
		if strings.HasPrefix(entry, "OPENCODE_TUI_CONFIG=") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("CreateSession(opencode): expected OPENCODE_TUI_CONFIG in Env, got %v", spy.lastReq.Env)
	}

	// Also verify that non-opencode CLIs do NOT get the env var.
	spy2 := &spyBackend{}
	e2 := NewSessionEngine()
	e2.backend = spy2

	_, err = e2.CreateSession(context.Background(), "claude", "test-claude", "", nil, 80, 24, nil)
	if err != nil {
		t.Fatalf("CreateSession(claude): %v", err)
	}

	for _, entry := range spy2.lastReq.Env {
		if strings.HasPrefix(entry, "OPENCODE_TUI_CONFIG=") {
			t.Errorf("CreateSession(claude): OPENCODE_TUI_CONFIG should NOT be in Env for non-opencode CLIs")
		}
	}
}

// TestOpenCodeTUIConfig asserts that a managed opencode-tui.json file is written
// with the correct content (theme set to "system" for terminal passthrough).
// Wave 0: RED state — no code writes this file yet.
func TestOpenCodeTUIConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode-tui.json")

	// Plan 02 will add ensureOpenCodeTUIConfig(dir) that writes this file.
	// For now, assert the file exists — this will FAIL (RED state).
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("managed opencode-tui.json not found: %v", err)
	}
	if !strings.Contains(string(data), `"theme":"system"`) {
		t.Errorf("expected theme:system in tui.json, got: %s", data)
	}
}
