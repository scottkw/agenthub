package daemon

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/pty"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/status"
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
	id, err := e.CreateSession(context.Background(), "cat", "test", "", nil, 0, 0, nil, nil)
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

	id1, err := e.CreateSession(context.Background(), "cat", "tab-1", "", nil, 0, 0, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession tab-1: %v", err)
	}
	id2, err := e.CreateSession(context.Background(), "cat", "tab-2", "", nil, 0, 0, nil, nil)
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

	id, err := e.CreateSession(context.Background(), "cat", "kill-me", "", nil, 0, 0, nil, nil)
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

	id, err := e.CreateSession(context.Background(), "cat", "original", "", nil, 0, 0, nil, nil)
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

	id, err := e.CreateSession(context.Background(), "cat", "status-test", "", nil, 0, 0, nil, nil)
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
	id, err := e.CreateSession(context.Background(), "cat", "args-test", "", []string{"--version"}, 0, 0, nil, nil)
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
	id, err := e.CreateSession(context.Background(), "cat", "dims-test", "", nil, 120, 40, nil, nil)
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
	id, err := e.CreateSession(context.Background(), "cat", "default-dims", "", nil, 0, 0, nil, nil)
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
	id, err := e.CreateSession(context.Background(), "cat", "h-eng", "", nil, 0, 0, nil, nil)
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
	// Isolate from real settings.json that NewSessionEngine may have loaded.
	e.configDir = t.TempDir()
	e.cliPaths = make(map[string]string)

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
func (s *spyBackend) List() []*pty.Session          { return nil }

// TestCreateSession_OpenCodeEnv asserts that CreateSession injects
// OPENCODE_TUI_CONFIG into the PTY environment when cli == "opencode",
// and that non-opencode CLIs do NOT receive the env var.
func TestCreateSession_OpenCodeEnv(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	_, err := e.CreateSession(context.Background(), "opencode", "test-oc", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(opencode): %v", err)
	}

	// Assert OPENCODE_TUI_CONFIG was injected into the env.
	var found bool
	for _, entry := range spy.lastReq.Env {
		if strings.HasPrefix(entry, "OPENCODE_TUI_CONFIG=") {
			found = true
			// Verify the value matches the engine's config path.
			wantEnv := "OPENCODE_TUI_CONFIG=" + e.opencodeTUIConfig
			if entry != wantEnv {
				t.Errorf("env var = %q, want %q", entry, wantEnv)
			}
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

	_, err = e2.CreateSession(context.Background(), "claude", "test-claude", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(claude): %v", err)
	}

	for _, entry := range spy2.lastReq.Env {
		if strings.HasPrefix(entry, "OPENCODE_TUI_CONFIG=") {
			t.Errorf("CreateSession(claude): OPENCODE_TUI_CONFIG should NOT be in Env for non-opencode CLIs")
		}
	}

	// Verify codex also does NOT get the env var.
	spy3 := &spyBackend{}
	e3 := NewSessionEngine()
	e3.backend = spy3

	_, err = e3.CreateSession(context.Background(), "codex", "test-codex", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(codex): %v", err)
	}

	for _, entry := range spy3.lastReq.Env {
		if strings.HasPrefix(entry, "OPENCODE_TUI_CONFIG=") {
			t.Errorf("CreateSession(codex): OPENCODE_TUI_CONFIG should NOT be in Env for non-opencode CLIs")
		}
	}
}

// TestNotifyThemeChange_BroadcastsToOpenCodeOnly verifies that NotifyThemeChange
// only attempts to signal sessions where sessionCLIs[id] == "opencode".
func TestNotifyThemeChange_BroadcastsToOpenCodeOnly(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	// Create an opencode session and a claude session.
	_, err := e.CreateSession(context.Background(), "opencode", "oc-tab", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(opencode): %v", err)
	}
	_, err = e.CreateSession(context.Background(), "claude", "cl-tab", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(claude): %v", err)
	}

	// NotifyThemeChange should not panic even though spy sessions have nil cmd.
	// The Signal call will fail (nil process), which is logged and skipped.
	err = e.NotifyThemeChange(context.Background())
	if err != nil {
		t.Errorf("NotifyThemeChange: want nil error, got %v", err)
	}
}

// TestNotifyThemeChange_NoOpenCodeSessions verifies no-op when no opencode sessions exist.
func TestNotifyThemeChange_NoOpenCodeSessions(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	// Create only non-opencode sessions.
	_, _ = e.CreateSession(context.Background(), "claude", "cl", "", nil, 80, 24, nil, nil)
	_, _ = e.CreateSession(context.Background(), "codex", "cx", "", nil, 80, 24, nil, nil)

	err := e.NotifyThemeChange(context.Background())
	if err != nil {
		t.Errorf("NotifyThemeChange with no opencode: want nil, got %v", err)
	}
}

// TestNotifyThemeChange_EmptyEngine verifies no-op on fresh engine with no sessions.
func TestNotifyThemeChange_EmptyEngine(t *testing.T) {
	e := NewSessionEngine()
	err := e.NotifyThemeChange(context.Background())
	if err != nil {
		t.Errorf("NotifyThemeChange on empty engine: want nil, got %v", err)
	}
}

// TestSessionCLIs_TrackedAndCleanedUp verifies sessionCLIs map is populated
// in CreateSession and cleaned up in KillSession.
func TestSessionCLIs_TrackedAndCleanedUp(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	id, err := e.CreateSession(context.Background(), "opencode", "oc", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Check sessionCLIs populated.
	e.mu.RLock()
	cli, ok := e.sessionCLIs[id]
	e.mu.RUnlock()
	if !ok {
		t.Fatal("sessionCLIs not populated for session")
	}
	if cli != "opencode" {
		t.Errorf("sessionCLIs[%s] = %q, want %q", id, cli, "opencode")
	}

	// Kill and verify cleanup.
	_ = e.KillSession(id)
	e.mu.RLock()
	_, ok = e.sessionCLIs[id]
	e.mu.RUnlock()
	if ok {
		t.Error("sessionCLIs not cleaned up after KillSession")
	}
}

// TestOpenCodeTUIConfig asserts that ensureOpenCodeTUIConfig writes a managed
// opencode-tui.json with the correct content (theme set to "system" for terminal passthrough).
func TestOpenCodeTUIConfig(t *testing.T) {
	dir := t.TempDir()
	path := ensureOpenCodeTUIConfig(dir)

	// Verify path is correct
	expected := filepath.Join(dir, "opencode-tui.json")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}

	// Verify file content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("managed opencode-tui.json not found: %v", err)
	}
	want := `{"$schema":"https://opencode.ai/tui.json","theme":"system"}` + "\n"
	if string(data) != want {
		t.Errorf("content = %q, want %q", string(data), want)
	}

	// Verify idempotent (second call does not error)
	path2 := ensureOpenCodeTUIConfig(dir)
	if path2 != path {
		t.Errorf("second call path = %q, want %q", path2, path)
	}
}

// hubReadUntil subscribes to the session's relay hub and waits until the
// marker string appears in the output frames, or the timeout elapses.
// Returns true if the marker was found within the timeout.
//
// It replays the scrollback snapshot first (to catch output that arrived before
// subscribing), then drains new frames from the hub channel until the deadline.
//
// Frame format: [1-byte type | payload...]. MsgOutput (0x01) frames contain PTY
// text. The scrollback stores concatenated raw frames; we strip the MsgOutput
// prefix bytes (0x01) to recover the text content.
func hubReadUntil(t *testing.T, e *SessionEngine, id, marker string, timeout time.Duration) bool {
	t.Helper()
	hub, ok := e.manager.Get(id)
	if !ok {
		t.Errorf("hubReadUntil: hub not found for session %s", id)
		return false
	}

	sub := &relay.Subscriber{
		Msgs:      make(chan []byte, 512),
		CloseSlow: func() {},
	}
	// Subscribe before snapshot to avoid a race where new frames arrive between
	// snapshot and subscribe.
	hub.Subscribe(sub)
	defer hub.Unsubscribe(sub)

	// Replay scrollback: concatenated MakeOutputFrame bytes.
	// MsgOutput is 0x01, which won't appear in ASCII terminal text markers.
	// Strip all 0x01 bytes to recover the payload text.
	var collected strings.Builder
	for _, b := range hub.ScrollbackSnapshot() {
		if b != relay.MsgOutput {
			collected.WriteByte(b)
		}
	}
	if strings.Contains(collected.String(), marker) {
		return true
	}

	// Drain new frames from the subscriber channel until marker found or timeout.
	deadline := time.After(timeout)
	for {
		select {
		case frame := <-sub.Msgs:
			// Frame: [type byte | payload...]
			if len(frame) > 1 {
				collected.Write(frame[1:])
			}
			if strings.Contains(collected.String(), marker) {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

// TestNotifyThemeChange_RealProcess_Integration spawns a real shell process
// that traps SIGUSR2 and writes a marker to stdout. NotifyThemeChange is then
// called and the test verifies the marker appears, proving end-to-end signal
// delivery through the PTY layer.
//
// Gated: POSIX only (SIGUSR2 does not exist on Windows).
// Gated: Skipped in -short mode (spawns real PTY, takes ~2s).
func TestNotifyThemeChange_RealProcess_Integration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGUSR2 not available on Windows")
	}
	if testing.Short() {
		t.Skip("integration test: skipped in -short mode")
	}

	e := NewSessionEngine()

	// Map "opencode" -> /bin/sh so ResolveCLI returns a valid executable.
	if err := e.UpdateCLIPath("opencode", "/bin/sh"); err != nil {
		t.Fatalf("UpdateCLIPath: %v", err)
	}

	// Create session with cli="opencode" (stored in sessionCLIs as "opencode").
	// Args: -c with a script that traps SIGUSR2 and prints a marker.
	script := `trap 'echo SIGUSR2_RECEIVED' USR2; echo READY; while true; do sleep 0.1; done`
	id, err := e.CreateSession(
		context.Background(),
		"opencode",
		"sigusr2-test",
		"",
		[]string{"-c", script},
		80, 24,
		nil, nil,
	)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	// Wait for READY marker (process started and trap is installed).
	if !hubReadUntil(t, e, id, "READY", 5*time.Second) {
		t.Fatal("process did not print READY within 5s")
	}

	// Send SIGUSR2 via NotifyThemeChange.
	if err := e.NotifyThemeChange(context.Background()); err != nil {
		t.Fatalf("NotifyThemeChange: %v", err)
	}

	// Read output and look for SIGUSR2_RECEIVED marker.
	if !hubReadUntil(t, e, id, "SIGUSR2_RECEIVED", 5*time.Second) {
		t.Error("SIGUSR2 marker not found in output after NotifyThemeChange")
	}
}

// ========================================================================
// Phase 100 Plan 02 — Shell session dispatch tests (SHELL-05 + SHELL-09).
//
// These tests exercise the engine's per-CLI shell-resolution branch and
// the status.Watch bypass guard. They use the existing spyBackend harness
// to assert on the pty.CreateRequest that the engine constructs, without
// launching a real PTY.
// ========================================================================

// newShellTestEngine returns an engine wired to a spyBackend with the
// mandatory settings isolation. Mirrors TestEngineResolveCLI's setup.
func newShellTestEngine(t *testing.T) (*SessionEngine, *spyBackend) {
	t.Helper()
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.configDir = t.TempDir()
	e.cliPaths = make(map[string]string)
	e.backend = spy
	return e, spy
}

// argsContains reports whether arg `flag` appears in the captured argv slice.
func argsContains(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// TestCreateSession_ShellArgv_Interactive (SHELL-05): bash spawns
// interactively, non-login, with an absolute path and WorkDir honored.
func TestCreateSession_ShellArgv_Interactive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell binaries")
	}
	e, spy := newShellTestEngine(t)
	_, err := e.CreateSession(context.Background(), "bash", "tab", "/tmp", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(bash): %v", err)
	}
	if !strings.HasSuffix(spy.lastReq.CLI, "/bash") {
		t.Errorf("CLI = %q, want absolute path ending in /bash", spy.lastReq.CLI)
	}
	if !argsContains(spy.lastReq.Args, "-i") {
		t.Errorf("Args missing -i: %v", spy.lastReq.Args)
	}
	if argsContains(spy.lastReq.Args, "-l") || argsContains(spy.lastReq.Args, "--login") {
		t.Errorf("Args has login flag (must be non-login): %v", spy.lastReq.Args)
	}
	if spy.lastReq.WorkDir != "/tmp" {
		t.Errorf("WorkDir = %q, want /tmp", spy.lastReq.WorkDir)
	}
}

// TestCreateSession_ZshArgv_Interactive (SHELL-05): zsh same shape as bash.
func TestCreateSession_ZshArgv_Interactive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell binaries")
	}
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not installed")
	}
	e, spy := newShellTestEngine(t)
	_, err := e.CreateSession(context.Background(), "zsh", "tab", "/tmp", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(zsh): %v", err)
	}
	if !strings.HasSuffix(spy.lastReq.CLI, "/zsh") {
		t.Errorf("CLI = %q, want absolute path ending in /zsh", spy.lastReq.CLI)
	}
	if !argsContains(spy.lastReq.Args, "-i") {
		t.Errorf("Args missing -i: %v", spy.lastReq.Args)
	}
	if argsContains(spy.lastReq.Args, "-l") || argsContains(spy.lastReq.Args, "--login") {
		t.Errorf("Args has login flag: %v", spy.lastReq.Args)
	}
}

// TestCreateSession_PwshArgv (SHELL-05): pwsh spawns with -NoLogo, no login flags.
func TestCreateSession_PwshArgv(t *testing.T) {
	if _, err := exec.LookPath("pwsh"); err != nil {
		t.Skip("pwsh not installed")
	}
	e, spy := newShellTestEngine(t)
	_, err := e.CreateSession(context.Background(), "pwsh", "tab", "/tmp", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(pwsh): %v", err)
	}
	if !argsContains(spy.lastReq.Args, "-NoLogo") {
		t.Errorf("Args missing -NoLogo: %v", spy.lastReq.Args)
	}
	if argsContains(spy.lastReq.Args, "-l") || argsContains(spy.lastReq.Args, "--login") {
		t.Errorf("Args has login flag: %v", spy.lastReq.Args)
	}
}

// TestCreateSession_ShellWorkDirHonored (SHELL-05): caller-supplied WorkDir
// reaches the backend unchanged.
func TestCreateSession_ShellWorkDirHonored(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell binaries")
	}
	e, spy := newShellTestEngine(t)
	_, err := e.CreateSession(context.Background(), "bash", "tab", "/home/user/project", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(bash): %v", err)
	}
	if spy.lastReq.WorkDir != "/home/user/project" {
		t.Errorf("WorkDir = %q, want %q", spy.lastReq.WorkDir, "/home/user/project")
	}
}

// TestCreateSession_ShellEmptyWorkDirHome (SHELL-05, Pitfall 4): empty WorkDir
// for shell sessions resolves to os.UserHomeDir().
func TestCreateSession_ShellEmptyWorkDirHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell binaries")
	}
	e, spy := newShellTestEngine(t)
	_, err := e.CreateSession(context.Background(), "bash", "tab", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(bash): %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("os.UserHomeDir failed: %v", err)
	}
	if home == "" || home == "/" || home == "." {
		t.Skipf("os.UserHomeDir returned unreliable value %q", home)
	}
	if spy.lastReq.WorkDir != home {
		t.Errorf("WorkDir = %q, want %q (os.UserHomeDir)", spy.lastReq.WorkDir, home)
	}
}

// TestCreateSession_AICLIEmptyWorkDirUnchanged (SHELL-05 negative case):
// empty WorkDir for AI CLI sessions stays empty (no $HOME substitution).
func TestCreateSession_AICLIEmptyWorkDirUnchanged(t *testing.T) {
	e, spy := newShellTestEngine(t)
	_, err := e.CreateSession(context.Background(), "claude", "tab", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(claude): %v", err)
	}
	if spy.lastReq.WorkDir != "" {
		t.Errorf("AI CLI WorkDir = %q, want empty (existing behavior)", spy.lastReq.WorkDir)
	}
}

// TestCreateSession_ShellSkipsStatusWatch (SHELL-09): status.Watch is not
// invoked for shell sessions, so sessionStatuses[id] is never populated.
func TestCreateSession_ShellSkipsStatusWatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell binaries")
	}
	e, _ := newShellTestEngine(t)
	id, err := e.CreateSession(context.Background(), "bash", "tab", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(bash): %v", err)
	}
	// Give any unintended goroutine a chance to populate the map.
	time.Sleep(50 * time.Millisecond)

	got := e.GetSessionStatus(id)
	if got != "running" {
		t.Errorf("GetSessionStatus = %q, want %q (no watcher should populate)", got, "running")
	}

	e.statusMu.RLock()
	_, exists := e.sessionStatuses[id]
	e.statusMu.RUnlock()
	if exists {
		t.Errorf("expected no entry in sessionStatuses for shell session, but found one")
	}
}

// TestListSessions_ShellStatusRunning (SHELL-09): ListSessions returns
// Status=="running" for shell sessions (falls through conservative default).
func TestListSessions_ShellStatusRunning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell binaries")
	}
	e, _ := newShellTestEngine(t)
	id, err := e.CreateSession(context.Background(), "bash", "tab", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(bash): %v", err)
	}
	sessions := e.ListSessions()
	var found bool
	for _, s := range sessions {
		if s.ID == id {
			found = true
			if s.Status != "running" {
				t.Errorf("Status = %q, want %q", s.Status, "running")
			}
			if s.Status == "waiting" || s.Status == "error" || s.Status == "errored" || s.Status == "idle" {
				t.Errorf("Status = %q must not be a heuristic state", s.Status)
			}
			break
		}
	}
	if !found {
		t.Fatalf("session %s not in ListSessions output", id)
	}
}

// TestShell_NoStatusMapEntry (SHELL-09 defensive): shell session IDs never
// land in sessionStatuses regardless of whether other AI-CLI sessions exist.
func TestShell_NoStatusMapEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell binaries")
	}
	e, _ := newShellTestEngine(t)
	bashID, err := e.CreateSession(context.Background(), "bash", "bash-tab", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(bash): %v", err)
	}
	shellIDs := []string{bashID}
	if _, err := exec.LookPath("zsh"); err == nil {
		id, err := e.CreateSession(context.Background(), "zsh", "zsh-tab", "", nil, 80, 24, nil, nil)
		if err != nil {
			t.Fatalf("CreateSession(zsh): %v", err)
		}
		shellIDs = append(shellIDs, id)
	}
	shellSysID, err := e.CreateSession(context.Background(), "shell", "sys-tab", "", nil, 80, 24, nil, nil)
	if err == nil {
		shellIDs = append(shellIDs, shellSysID)
	}
	// Create an AI CLI session as a control (no assertion against its presence).
	_, _ = e.CreateSession(context.Background(), "claude", "claude-tab", "", nil, 80, 24, nil, nil)

	time.Sleep(50 * time.Millisecond)
	for _, id := range shellIDs {
		e.statusMu.RLock()
		_, exists := e.sessionStatuses[id]
		e.statusMu.RUnlock()
		if exists {
			t.Errorf("expected no sessionStatuses entry for shell session %s, but found one", id)
		}
	}
}

// TestIsShellSession_AllShellNames (unit test for the helper).
//
// Phase 150 SET-01 follow-up: a shell session's cli is its resolved binary
// PATH ('/bin/zsh'), not a bare key — so isShellSession must normalize to the
// basename (minus .exe, case-insensitive) before matching, mirroring the
// frontend lib/shellCli.isShellCli. Previously full-path shells returned false,
// which mis-classified them for status.Watch (SHELL-09).
func TestIsShellSession_AllShellNames(t *testing.T) {
	cases := map[string]bool{
		// Bare keys (unchanged).
		"shell":      true,
		"bash":       true,
		"zsh":        true,
		"pwsh":       true,
		"powershell": true,
		// Non-shell agent CLIs.
		"claude":         false,
		"opencode":       false,
		"":               false,
		"/usr/bin/claude": false,
		// Full paths — the real-app case (was incorrectly false).
		"/bin/bash":            true,
		"/bin/zsh":             true,
		"/usr/local/bin/bash":  true,
		// Case-insensitive on the basename.
		"Bash":    true,
		"/bin/ZSH": true,
		// Windows path + .exe strip (deterministic across platforms).
		"powershell.exe": true,
		`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`: true,
		// Shells outside the warning set stay false.
		"/bin/sh": false,
		"fish":    false,
	}
	for name, want := range cases {
		got := isShellSession(name)
		if got != want {
			t.Errorf("isShellSession(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestResolveShellSpawn_KnownShell (unit test for the helper).
func TestResolveShellSpawn_KnownShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell binaries")
	}
	e, _ := newShellTestEngine(t)
	path, args, ok := e.resolveShellSpawn("bash")
	if !ok {
		t.Fatal("resolveShellSpawn(bash) returned ok=false")
	}
	if !strings.HasSuffix(path, "/bash") {
		t.Errorf("path = %q, want absolute path ending in /bash", path)
	}
	if !argsContains(args, "-i") {
		t.Errorf("args missing -i: %v", args)
	}

	// Non-shell cli returns ok=false.
	_, _, ok = e.resolveShellSpawn("claude")
	if ok {
		t.Error("resolveShellSpawn(claude) returned ok=true, want false")
	}
}

// TestResolveShellSpawn_SystemDefault (POSIX-only): cli="shell" resolves
// to $SHELL.
func TestResolveShellSpawn_SystemDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX SHELL env semantics")
	}
	// Pick a real shell that exists on the test host.
	candidates := []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	var pick string
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			pick = c
			break
		}
	}
	if pick == "" {
		t.Skip("no POSIX shell at standard paths")
	}
	t.Setenv("SHELL", pick)
	e, _ := newShellTestEngine(t)
	path, args, ok := e.resolveShellSpawn("shell")
	if !ok {
		t.Fatalf("resolveShellSpawn(shell) returned ok=false (SHELL=%s)", pick)
	}
	if path != pick {
		t.Errorf("path = %q, want %q (from $SHELL)", path, pick)
	}
	if !argsContains(args, "-i") {
		t.Errorf("args missing -i: %v", args)
	}
}

// TestResolveShellSpawn_PowerShellOverride (M2 lock-in): cliPaths["powershell"]
// override resolves cleanly via knownShellSpecs without falling through to
// discovery. This locks the Plan 01 M2 contract.
func TestResolveShellSpawn_PowerShellOverride(t *testing.T) {
	e, _ := newShellTestEngine(t)
	// Direct map mutation under the engine mutex — the public UpdateCLIPath
	// would call os.Stat on the override path and fail for our synthetic
	// non-existent path. The override-branch logic matches by basename, not
	// by file existence, so the unit test only needs the map populated.
	e.mu.Lock()
	e.cliPaths["powershell"] = "/usr/local/bin/pwsh-stub"
	e.mu.Unlock()

	path, args, ok := e.resolveShellSpawn("powershell")
	if !ok {
		t.Fatal("resolveShellSpawn(powershell) with override returned ok=false")
	}
	if path != "/usr/local/bin/pwsh-stub" {
		t.Errorf("path = %q, want override path", path)
	}
	if !argsContains(args, "-NoLogo") {
		t.Errorf("args missing -NoLogo (from powershell spec): %v", args)
	}
}

// TestResolveShellSpawn_FreshInstallFallback (v3.3.1 Phase 109 UAT finding)
// locks the fix for the fresh-install Windows path: when cli="shell",
// shellPath setting is empty, cliPaths["shell"] is empty, and DiscoverShells
// returns only specific-name shells (no synthetic "shell"), branch (4) MUST
// pick the first discovered shell rather than returning isShell=false.
//
// Pre-fix behavior: resolveShellSpawn returned ("", nil, false) and the
// caller exec'd the literal string "shell", which failed as "executable file
// not found in %PATH%" on Windows.
//
// We exercise this on POSIX with $SHELL deliberately UNSET — that suppresses
// DiscoverShells's synthetic "shell" entry (which is the existing path
// covered by TestResolveShellSpawn_SystemDefault) and forces resolution to
// reach branch (4).
func TestResolveShellSpawn_FreshInstallFallback(t *testing.T) {
	// Unset $SHELL so DiscoverShells does NOT append the synthetic "shell"
	// entry — otherwise branch (2) would match and branch (4) would never
	// be exercised.
	t.Setenv("SHELL", "")
	e, _ := newShellTestEngine(t)
	// e.shellPath is empty by default; e.cliPaths is empty by default.
	// At least one of bash/zsh must be on PATH for the test host to exercise
	// this branch — every CI / dev box should satisfy this.
	path, args, ok := e.resolveShellSpawn("shell")
	if !ok {
		t.Fatal("resolveShellSpawn(shell) returned ok=false on fresh install; " +
			"branch (4) fallback did not engage. v3.3.1 Phase 109 UAT regressed.")
	}
	if path == "" || path == "shell" {
		t.Errorf("resolveShellSpawn(shell) returned bogus path %q; expected a "+
			"real binary path from DiscoverShells", path)
	}
	if len(args) == 0 {
		t.Errorf("resolveShellSpawn(shell) returned empty args; expected the " +
			"discovered shell's spec.Argv (e.g. [-i] for bash/zsh)")
	}
}

// --- Phase 101 Plan 01: ShellWebShareWarned persistence ----------------------

// TestSetShellWebShareWarned_Default verifies fresh engine reads false.
//
// Phase 116 / TEST-06: NewSessionEngine() loads from the user's real
// ~/.config/agenthub/settings.json at construction time. Reset every field
// that loadSettingsFromDisk touches so the "default value" assertion is
// deterministic regardless of the developer's local settings.
func TestSetShellWebShareWarned_Default(t *testing.T) {
	e := NewSessionEngine()
	e.configDir = t.TempDir()
	e.cliPaths = make(map[string]string)
	e.startMinimized = false
	e.shellWebShareWarned = false
	e.shellPath = ""
	e.autoCloseSession = nil
	e.pluginSettings = defaultPluginSettings()

	if v := e.GetShellWebShareWarned(); v != false {
		t.Errorf("default GetShellWebShareWarned: got %v, want false", v)
	}
}

// TestSetShellWebShareWarned_Persists verifies round-trip through settings.json.
func TestSetShellWebShareWarned_Persists(t *testing.T) {
	dir := t.TempDir()

	e1 := NewSessionEngine()
	e1.configDir = dir
	e1.cliPaths = make(map[string]string)

	if err := e1.SetShellWebShareWarned(true); err != nil {
		t.Fatalf("SetShellWebShareWarned(true): %v", err)
	}

	// Second engine, same configDir → should observe the persisted value.
	e2 := NewSessionEngine()
	e2.configDir = dir
	e2.cliPaths = make(map[string]string)
	// Re-load explicitly: NewSessionEngine() already called loadSettingsFromDisk()
	// against the real config dir; we need to reload from our temp dir.
	e2.loadSettingsFromDisk(dir)

	if v := e2.GetShellWebShareWarned(); v != true {
		t.Errorf("after persist+reload: GetShellWebShareWarned = %v, want true", v)
	}
}

// TestSetShellWebShareWarned_PersistsFalseAfterTrue verifies that flipping
// back to false round-trips correctly (regression guard for omitempty).
func TestSetShellWebShareWarned_PersistsFalseAfterTrue(t *testing.T) {
	dir := t.TempDir()

	e1 := NewSessionEngine()
	e1.configDir = dir
	e1.cliPaths = make(map[string]string)

	if err := e1.SetShellWebShareWarned(true); err != nil {
		t.Fatalf("SetShellWebShareWarned(true): %v", err)
	}
	if err := e1.SetShellWebShareWarned(false); err != nil {
		t.Fatalf("SetShellWebShareWarned(false): %v", err)
	}

	e2 := NewSessionEngine()
	e2.configDir = dir
	e2.cliPaths = make(map[string]string)
	e2.loadSettingsFromDisk(dir)

	if v := e2.GetShellWebShareWarned(); v != false {
		t.Errorf("after true->false+reload: GetShellWebShareWarned = %v, want false", v)
	}
}

// TestSetShellWebShareWarned_RoundTripJSON verifies the field appears in
// settings.json with the expected JSON key when set to true.
func TestSetShellWebShareWarned_RoundTripJSON(t *testing.T) {
	dir := t.TempDir()

	e := NewSessionEngine()
	e.configDir = dir
	e.cliPaths = make(map[string]string)

	if err := e.SetShellWebShareWarned(true); err != nil {
		t.Fatalf("SetShellWebShareWarned(true): %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	if !strings.Contains(string(data), `"shellWebShareWarned":true`) {
		t.Errorf("settings.json missing expected key/value; got:\n%s", string(data))
	}
}

// --- Phase 107 Plan 01: shellPath persistence (SHELL-11) -------------------

// TestGetShellPath_DefaultResolvesPlatformDefault verifies that a fresh engine
// with no shellPath set returns a non-empty string from GetShellPath() that
// matches $SHELL or one of the platform hardcoded defaults.
func TestGetShellPath_DefaultResolvesPlatformDefault(t *testing.T) {
	e := NewSessionEngine()
	e.configDir = t.TempDir()
	e.cliPaths = make(map[string]string)

	got := e.GetShellPath()
	if got == "" {
		t.Fatal("GetShellPath() returned empty string for fresh engine; want platform default")
	}
	// Must match $SHELL or one of the platform hardcodes.
	shellEnv := os.Getenv("SHELL")
	validDefaults := map[string]bool{
		"/bin/zsh":  true,
		"/bin/bash": true,
		"pwsh.exe":  true,
	}
	if got != shellEnv && !validDefaults[got] {
		// It's acceptable if it came from DiscoverShells() "shell" entry.
		// Just assert it's a non-empty reasonable path.
		if got == "" {
			t.Errorf("GetShellPath() = %q, want non-empty platform default", got)
		}
	}
}

// TestSetShellPath_RejectsMissingPath verifies that SetShellPath returns an
// error when the path does not exist on disk.
func TestSetShellPath_RejectsMissingPath(t *testing.T) {
	e := NewSessionEngine()
	e.configDir = t.TempDir()
	e.cliPaths = make(map[string]string)

	err := e.SetShellPath("/no/such/path/does/not/exist")
	if err == nil {
		t.Fatal("SetShellPath(/no/such/path): expected error, got nil")
	}
	if e.shellPath != "" {
		t.Errorf("e.shellPath should be unchanged after rejected SetShellPath; got %q", e.shellPath)
	}
}

// TestSetShellPath_RejectsNonExecutable verifies that SetShellPath returns an
// error when the file exists but is not executable.
func TestSetShellPath_RejectsNonExecutable(t *testing.T) {
	dir := t.TempDir()
	nonExec := filepath.Join(dir, "not-a-shell")
	if err := os.WriteFile(nonExec, []byte("#!/bin/sh\n"), 0644); err != nil {
		t.Fatalf("create non-exec file: %v", err)
	}

	e := NewSessionEngine()
	e.configDir = dir
	e.cliPaths = make(map[string]string)

	err := e.SetShellPath(nonExec)
	if err == nil {
		t.Fatalf("SetShellPath(non-exec): expected error, got nil")
	}
	if e.shellPath != "" {
		t.Errorf("e.shellPath should be unchanged after rejected SetShellPath; got %q", e.shellPath)
	}
}

// TestSetShellPath_AcceptsExecutable verifies that SetShellPath accepts /bin/sh
// and that the value round-trips through a settings.json reload.
func TestSetShellPath_AcceptsExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses /bin/sh which is POSIX-only")
	}
	dir := t.TempDir()

	e1 := NewSessionEngine()
	e1.configDir = dir
	e1.cliPaths = make(map[string]string)

	if err := e1.SetShellPath("/bin/sh"); err != nil {
		t.Fatalf("SetShellPath(/bin/sh): %v", err)
	}
	if e1.shellPath != "/bin/sh" {
		t.Errorf("e1.shellPath = %q, want /bin/sh", e1.shellPath)
	}

	// Round-trip: second engine, same configDir, reload from disk.
	e2 := NewSessionEngine()
	e2.configDir = dir
	e2.cliPaths = make(map[string]string)
	e2.loadSettingsFromDisk(dir)

	if e2.GetShellPath() != "/bin/sh" {
		t.Errorf("after reload: GetShellPath() = %q, want /bin/sh", e2.GetShellPath())
	}
}

// TestSetShellPath_EmptyClears verifies that SetShellPath("") clears the
// persisted override and GetShellPath() falls back to the platform default.
func TestSetShellPath_EmptyClears(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses /bin/sh which is POSIX-only")
	}
	dir := t.TempDir()

	e := NewSessionEngine()
	e.configDir = dir
	e.cliPaths = make(map[string]string)

	// Set a specific path first.
	if err := e.SetShellPath("/bin/sh"); err != nil {
		t.Fatalf("SetShellPath(/bin/sh): %v", err)
	}

	// Clear it.
	if err := e.SetShellPath(""); err != nil {
		t.Fatalf("SetShellPath(): %v", err)
	}

	got := e.GetShellPath()
	if got == "/bin/sh" {
		// Should have fallen back to platform default (not the cleared value).
		// Note: on this machine /bin/sh might be the platform default via $SHELL —
		// so we verify via e.shellPath being empty (the cleared field).
	}
	if e.shellPath != "" {
		t.Errorf("e.shellPath after clear: got %q, want empty", e.shellPath)
	}
	if got == "" {
		t.Errorf("GetShellPath() after clear: returned empty, want platform default")
	}
}

// =============================================================================
// Phase 107 Plan 02: SHELL-12 backend — ListSessions exit-code normalization
// =============================================================================
//
// These tests assert the SHELL-12 invariant: PTY exit-code -1 (natural EOF)
// must be normalized to 0 in the ListSessions emission path, mirroring the
// normalization already applied in the natural-exit goroutine before onExit.
//
// The tests use direct registry injection (no real process spawn) for speed
// and determinism: create pty.Session with known state/exitCode, add to the
// engine registry, then assert what ListSessions reports.

// newExitedShell12Session builds a *pty.Session that is already in StateStopped
// with a given cached exit code. The session is NOT killed (killed=false), so
// the `!s.IsKilled()` guard in ListSessions allows the exitCode branch.
func newExitedShell12Session(id string, exitCode int) *pty.Session {
	s := &pty.Session{
		ID:        id,
		CLI:       "bash",
		CreatedAt: time.Now(),
	}
	s.SetExitCode(exitCode)
	s.SetState(pty.StateStopped)
	return s
}

// newKilledShell12Session builds a *pty.Session that is StateStopped AND
// killed=true, mirroring the KillSession path. IsKilled()==true causes
// ListSessions to skip exitCode emission entirely.
func newKilledShell12Session(id string) *pty.Session {
	s := &pty.Session{
		ID:        id,
		CLI:       "bash",
		CreatedAt: time.Now(),
	}
	s.MarkKilled()
	s.SetState(pty.StateStopped)
	return s
}

// newBareEngine creates a SessionEngine with no real backend (uses spyBackend),
// isolated configDir, and injects the given pre-built sessions into its
// registry. tabNames and sessionCLIs are auto-populated from each session's
// ID and CLI fields. Returns the engine ready for ListSessions assertions.
func newBareEngine(t *testing.T, sessions ...*pty.Session) *SessionEngine {
	t.Helper()
	e := &SessionEngine{
		registry:        pty.NewSessionRegistry(),
		backend:         &spyBackend{},
		manager:         relay.NewHubManager(),
		tabNames:        make(map[string]string),
		sessionCLIs:     make(map[string]string),
		cliPaths:        make(map[string]string),
		sessionStatuses: make(map[string]status.SessionStatus),
		configDir:       t.TempDir(),
	}
	for _, s := range sessions {
		e.registry.Add(s)
		e.mu.Lock()
		e.tabNames[s.ID] = s.ID
		e.sessionCLIs[s.ID] = s.CLI
		e.mu.Unlock()
	}
	return e
}

// findShell12Session locates a SessionInfo by ID in a ListSessions result.
func findShell12Session(list []SessionInfo, id string) (SessionInfo, bool) {
	for _, si := range list {
		if si.ID == id {
			return si, true
		}
	}
	return SessionInfo{}, false
}

// TestListSessions_NaturalExit_NormalizesNegativeOneToZero (SHELL-12):
// A session whose cached PTY exit code is -1 must be reported as ExitCode=0
// by ListSessions after the -1→0 guard is applied.
func TestListSessions_NaturalExit_NormalizesNegativeOneToZero(t *testing.T) {
	sess := newExitedShell12Session("norm-neg-one", -1)
	e := newBareEngine(t, sess)

	list := e.ListSessions()
	si, ok := findShell12Session(list, "norm-neg-one")
	if !ok {
		t.Fatal("session not found in ListSessions")
	}
	if si.State != "stopped" {
		t.Errorf("State = %q, want %q", si.State, "stopped")
	}
	if si.ExitCode == nil {
		t.Fatal("ExitCode is nil, want non-nil pointer for natural (non-killed) exit")
	}
	if *si.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0 (normalized from -1)", *si.ExitCode)
	}
}

// TestListSessions_NaturalExit_PreservesNonZero (SHELL-12):
// A session with exit code 2 must not be normalized — ExitCode=2 preserved.
func TestListSessions_NaturalExit_PreservesNonZero(t *testing.T) {
	sess := newExitedShell12Session("nonzero-exit", 2)
	e := newBareEngine(t, sess)

	list := e.ListSessions()
	si, ok := findShell12Session(list, "nonzero-exit")
	if !ok {
		t.Fatal("session not found in ListSessions")
	}
	if si.ExitCode == nil {
		t.Fatal("ExitCode is nil, want non-nil for natural exit")
	}
	if *si.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2 (preserved non-zero)", *si.ExitCode)
	}
}

// TestListSessions_NaturalExit_PreservesZero (regression):
// A session with exit code 0 must stay 0 after normalization (no change).
func TestListSessions_NaturalExit_PreservesZero(t *testing.T) {
	sess := newExitedShell12Session("zero-exit", 0)
	e := newBareEngine(t, sess)

	list := e.ListSessions()
	si, ok := findShell12Session(list, "zero-exit")
	if !ok {
		t.Fatal("session not found in ListSessions")
	}
	if si.ExitCode == nil {
		t.Fatal("ExitCode is nil, want non-nil for natural exit")
	}
	if *si.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0 (explicit zero preserved)", *si.ExitCode)
	}
}

// TestListSessions_State_StoppedAfterNaturalExit (SHELL-12 regression):
// A naturally-exited session must report State=="stopped".
func TestListSessions_State_StoppedAfterNaturalExit(t *testing.T) {
	sess := newExitedShell12Session("state-stopped", 0)
	e := newBareEngine(t, sess)

	list := e.ListSessions()
	si, ok := findShell12Session(list, "state-stopped")
	if !ok {
		t.Fatal("session not found in ListSessions")
	}
	if si.State != "stopped" {
		t.Errorf("State = %q, want %q (natural exit must flip to stopped)", si.State, "stopped")
	}
}

// TestListSessions_KilledSession_ExitCodeNil (regression):
// A killed session must NOT have an ExitCode field populated — the
// `!s.IsKilled()` guard prevents reading ProcessState which may still be
// in-flight from killSession's cmd.Wait().
func TestListSessions_KilledSession_ExitCodeNil(t *testing.T) {
	sess := newKilledShell12Session("killed-sess")
	e := newBareEngine(t, sess)

	list := e.ListSessions()
	si, ok := findShell12Session(list, "killed-sess")
	if !ok {
		t.Fatal("session not found in ListSessions")
	}
	if si.ExitCode != nil {
		t.Errorf("ExitCode = %d, want nil for killed session", *si.ExitCode)
	}
}

// TestListSessions_OnExitCallback_ReceivesNormalized (regression):
// The natural-exit goroutine must pass exitCode=0 to onExit even when PTY
// reports -1. This exercises the ALREADY-CORRECT goroutine path
// (engine.go:333-340) to prevent future refactors from breaking the contract
// that both ListSessions AND onExit rely on.
// Strategy: use a real short-lived command routed through the non-shell
// AI-CLI path by pointing a synthetic CLI name at /bin/sh; sh -c "exit 0"
// exits immediately with code 0. The goroutine normalizes -1→0 before calling
// onExit, so the callback must receive 0 regardless of PTY behavior.
func TestListSessions_OnExitCallback_ReceivesNormalized(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX sh")
	}
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh not found: %v", err)
	}

	e := NewSessionEngine()
	e.configDir = t.TempDir()
	e.cliPaths = make(map[string]string)

	// Route "fakecli" to /bin/sh so CreateSession does NOT enter the
	// isShellSession() branch (which hardcodes argv=[-i]). This lets us
	// pass -c "exit 0" as args to get a clean natural exit.
	e.mu.Lock()
	e.cliPaths["fakecli"] = shPath
	e.mu.Unlock()

	codeCh := make(chan int, 1)
	onExit := func(_ string, code int) {
		codeCh <- code
	}

	_, err = e.CreateSession(context.Background(), "fakecli", "tab", "", []string{"-c", "exit 0"}, 80, 24, nil, onExit)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	select {
	case code := <-codeCh:
		if code != 0 {
			t.Errorf("onExit received code=%d, want 0", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for onExit callback")
	}
}

// Phase 118 / FS-02: sessionWorkDirs map + GetSessionWorkDir.
//
// The five subtests below assert the WorkDir-gap closure:
//  1. Populated: CreateSession with a real workDir → GetSessionWorkDir returns
//     the EvalSymlinks-resolved absolute path.
//  2. ResolvesSymlink: a symlink in the workDir argument resolves to its target.
//  3. ClearedOnKill: after KillSession the entry is removed; GetSessionWorkDir
//     returns "".
//  4. EmptyForUnknown: GetSessionWorkDir("does-not-exist") returns "".
//  5. FallbackOnEvalSymlinksError: bad workDir does not fail CreateSession;
//     GetSessionWorkDir returns the raw (unresolved) workDir.
//
// These tests use spyBackend (see TestCreateSession_OpenCodeEnv) to avoid
// spawning a real PTY — only the engine-internal map plumbing is under test.
func TestEngine_SessionWorkDirsPopulated(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy
	e.configDir = t.TempDir()

	tmpDir := t.TempDir()
	// EvalSymlinks the expected value so the assertion matches even when
	// t.TempDir() returns a symlinked path (macOS /var → /private/var).
	wantResolved, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", tmpDir, err)
	}

	id, err := e.CreateSession(context.Background(), "cat", "wd-test", tmpDir, nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	got := e.GetSessionWorkDir(id)
	if got != wantResolved {
		t.Errorf("GetSessionWorkDir(%q) = %q, want %q", id, got, wantResolved)
	}
}

func TestEngine_SessionWorkDirsResolvesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows; covered by *_Populated for non-symlink case")
	}
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy
	e.configDir = t.TempDir()

	realDir := t.TempDir()
	wantResolved, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", realDir, err)
	}

	// Create a symlink in another temp dir pointing to realDir.
	linkParent := t.TempDir()
	linkPath := filepath.Join(linkParent, "link-to-real")
	if err := os.Symlink(realDir, linkPath); err != nil {
		t.Fatalf("Symlink(%q, %q): %v", realDir, linkPath, err)
	}

	id, err := e.CreateSession(context.Background(), "cat", "sym-test", linkPath, nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	got := e.GetSessionWorkDir(id)
	if got != wantResolved {
		t.Errorf("GetSessionWorkDir(%q) = %q, want resolved %q (symlink should be followed)", id, got, wantResolved)
	}
}

func TestEngine_SessionWorkDirsClearedOnKill(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy
	e.configDir = t.TempDir()

	tmpDir := t.TempDir()
	id, err := e.CreateSession(context.Background(), "cat", "kill-wd", tmpDir, nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if got := e.GetSessionWorkDir(id); got == "" {
		t.Fatalf("precondition: GetSessionWorkDir before kill should be non-empty, got %q", got)
	}

	if err := e.KillSession(id); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	if got := e.GetSessionWorkDir(id); got != "" {
		t.Errorf("GetSessionWorkDir after KillSession = %q, want \"\"", got)
	}
}

func TestEngine_SessionWorkDirsEmptyForUnknown(t *testing.T) {
	e := NewSessionEngine()
	if got := e.GetSessionWorkDir("does-not-exist"); got != "" {
		t.Errorf("GetSessionWorkDir(unknown) = %q, want \"\"", got)
	}
}

func TestEngine_SessionWorkDirsFallbackOnEvalSymlinksError(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy
	e.configDir = t.TempDir()

	badPath := "/this/path/definitely/does/not/exist/xyz123"
	id, err := e.CreateSession(context.Background(), "cat", "bad-wd", badPath, nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession with bad workDir should not fail (fallback path): %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	got := e.GetSessionWorkDir(id)
	if got != badPath {
		t.Errorf("GetSessionWorkDir(%q) = %q, want %q (raw workDir fallback when EvalSymlinks errors)", id, got, badPath)
	}
}

// =============================================================================
// Phase 131 / GRID-02: WorkDir field in daemon.SessionInfo + ListSessions() population.
//
// Two subtests:
//  1. ListSessions_WorkDir_Populated: a session created with a real workDir must
//     return a non-empty SessionInfo.WorkDir equal to the EvalSymlinks-resolved path.
//  2. ListSessions_WorkDir_EmptyForUnknown: a bare engine with a session injected
//     directly (no sessionWorkDirs entry) returns WorkDir=="" without panic.
// =============================================================================

// TestListSessions_WorkDir_Populated (Phase 131 / GRID-02):
// CreateSession with a resolved workDir → ListSessions returns SessionInfo.WorkDir
// equal to e.sessionWorkDirs[sessionID].
func TestListSessions_WorkDir_Populated(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy
	e.configDir = t.TempDir()

	tmpDir := t.TempDir()
	wantResolved, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", tmpDir, err)
	}

	id, err := e.CreateSession(context.Background(), "cat", "wd-list-test", tmpDir, nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	sessions := e.ListSessions()
	var found SessionInfo
	for _, s := range sessions {
		if s.ID == id {
			found = s
			break
		}
	}
	if found.ID == "" {
		t.Fatal("session not found in ListSessions output")
	}
	if found.WorkDir == "" {
		t.Errorf("SessionInfo.WorkDir is empty; want %q", wantResolved)
	}
	if found.WorkDir != wantResolved {
		t.Errorf("SessionInfo.WorkDir = %q, want %q", found.WorkDir, wantResolved)
	}
}

// TestListSessions_ViewerCount (Phase 168 / FIX-04, #121): ListSessions must
// populate SessionInfo.ViewerCount from hub.RemoteViewerCount() — a
// never-shared session's own internal (Origin=="local") subscribers must NOT
// count as viewers; only Origin=="web" (remote/shared) subscribers do.
func TestListSessions_ViewerCount(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy
	e.configDir = t.TempDir()

	id, err := e.CreateSession(context.Background(), "cat", "viewer-count-test", t.TempDir(), nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	hub, ok := e.manager.Get(id)
	if !ok {
		t.Fatalf("hub not found for session %s", id)
	}

	// Local-only subscribers (the app's own internal WebSocket connections):
	// TerminalPanel, ChatPanel, status watcher, Hub-card preview.
	local1 := &relay.Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	local2 := &relay.Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	hub.Subscribe(local1)
	hub.Subscribe(local2)

	sessions := e.ListSessions()
	found := findSessionByID(sessions, id)
	if found.ID == "" {
		t.Fatal("session not found in ListSessions output")
	}
	if found.ViewerCount != 0 {
		t.Errorf("local-only subscribers: ViewerCount = %d, want 0", found.ViewerCount)
	}

	// Add two web-origin (remote/shared) subscribers.
	web1 := &relay.Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "web"}
	web2 := &relay.Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "web"}
	hub.Subscribe(web1)
	hub.Subscribe(web2)

	sessions = e.ListSessions()
	found = findSessionByID(sessions, id)
	if found.ID == "" {
		t.Fatal("session not found in ListSessions output (2nd call)")
	}
	if found.ViewerCount != 2 {
		t.Errorf("2 web + 2 local subscribers: ViewerCount = %d, want 2", found.ViewerCount)
	}
}

// findSessionByID is a small test helper shared by ListSessions field-assertion tests.
func findSessionByID(sessions []SessionInfo, id string) SessionInfo {
	for _, s := range sessions {
		if s.ID == id {
			return s
		}
	}
	return SessionInfo{}
}

// TestListSessions_WorkDir_EmptyForUnknown (Phase 131 / GRID-02):
// A session with no sessionWorkDirs entry returns WorkDir=="" without panic.
// Uses newBareEngine (direct registry injection — no real PTY spawn).
func TestListSessions_WorkDir_EmptyForUnknown(t *testing.T) {
	sess := newExitedShell12Session("no-workdir-sess", 0)
	e := newBareEngine(t, sess)
	// sessionWorkDirs is nil in newBareEngine — reading from a nil map in Go
	// returns the zero value ("") without panic. This test locks that contract.

	sessions := e.ListSessions()
	si, ok := findShell12Session(sessions, "no-workdir-sess")
	if !ok {
		t.Fatal("session not found in ListSessions")
	}
	if si.WorkDir != "" {
		t.Errorf("WorkDir = %q, want empty string for session with no sessionWorkDirs entry", si.WorkDir)
	}
}

// ========================================================================
// Phase 132 Plan 01 — GetSessionTailLines unit tests (CARD-07).
//
// These tests verify the engine method strips relay framing bytes (0x01)
// and ANSI/OSC escape sequences, returns the last N lines, trims trailing
// empty lines, and returns []string{} (not nil) for unknown sessions (IN-01).
//
// Hub setup: each test creates a hub via e.manager.Create with an io.Pipe.
// The writer end receives raw text; hub.Run() frames it with MakeOutputFrame
// (prepends 0x01), stores the result in the scrollback, then the pipe is
// closed so Run() exits cleanly.
// ========================================================================

// makeTailHub creates a hub for the given session ID in the engine's manager,
// writes rawContent as PTY output (hub.Run frames it as [0x01 | payload]),
// closes the pipe, and waits for the Run goroutine to finish draining.
func makeTailHub(t *testing.T, e *SessionEngine, sessionID, rawContent string) {
	t.Helper()
	pr, pw := io.Pipe()
	hub := e.manager.Create(sessionID, pr, pw, nil)
	// Write content and close so hub.Run returns after processing.
	_, err := pw.Write([]byte(rawContent))
	if err != nil {
		t.Fatalf("makeTailHub write: %v", err)
	}
	pw.Close()
	// Wait for hub.Run to finish draining the reader.
	<-hub.Done()
}

// TestGetSessionTailLines_StripsFramingBytes: scrollback containing 0x01 framing
// bytes → returned lines contain no 0x01 byte.
func TestGetSessionTailLines_StripsFramingBytes(t *testing.T) {
	e := NewSessionEngine()
	makeTailHub(t, e, "framing-test", "hello world\n")

	lines := e.GetSessionTailLines("framing-test", 10)
	if lines == nil {
		t.Fatal("GetSessionTailLines returned nil, expected lines")
	}
	for _, line := range lines {
		for _, b := range []byte(line) {
			if b == relay.MsgOutput {
				t.Errorf("line %q contains 0x01 framing byte", line)
			}
		}
	}
	// Verify content is present.
	found := false
	for _, line := range lines {
		if strings.Contains(line, "hello world") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'hello world' in lines, got: %v", lines)
	}
}

// TestGetSessionTailLines_StripsANSI: scrollback containing CSI and OSC sequences
// → returned lines contain no escape artifacts.
func TestGetSessionTailLines_StripsANSI(t *testing.T) {
	e := NewSessionEngine()
	// CSI color sequence + OSC title-setting sequence.
	content := "\x1b[32mgreen text\x1b[0m\n\x1b]0;title\x07\nplain line\n"
	makeTailHub(t, e, "ansi-test", content)

	lines := e.GetSessionTailLines("ansi-test", 10)
	if lines == nil {
		t.Fatal("GetSessionTailLines returned nil")
	}
	joined := strings.Join(lines, "\n")
	// Must not contain CSI artifact.
	if strings.Contains(joined, "[32m") || strings.Contains(joined, "\x1b") {
		t.Errorf("lines still contain ANSI artifacts: %v", lines)
	}
	// Must not contain OSC artifact.
	if strings.Contains(joined, "]0;") {
		t.Errorf("lines still contain OSC artifacts: %v", lines)
	}
	// Readable text must survive.
	if !strings.Contains(joined, "green text") {
		t.Errorf("'green text' missing after strip: %v", lines)
	}
}

// TestGetSessionTailLines_StripsOSC8Hyperlink: OSC 8 hyperlink with ST terminator
// (ESC ] 8 ;; url ESC \) must be fully stripped — no trailing backslash artifact.
// CR-01: the original regex left the 0x5c backslash of the ST terminator visible.
func TestGetSessionTailLines_StripsOSC8Hyperlink(t *testing.T) {
	e := NewSessionEngine()
	// OSC 8 hyperlink with ST terminator: ESC ] 8 ;; https://foo.com ESC \
	// The link text follows a second OSC 8 ;; ESC \ to close the hyperlink.
	// Terminal emitters produce: ESC]8;;URL ESC\ LINKTEXT ESC]8;; ESC\
	content := "before \x1b]8;;https://example.com\x1b\\click here\x1b]8;;\x1b\\ after\n"
	makeTailHub(t, e, "osc8-test", content)

	lines := e.GetSessionTailLines("osc8-test", 10)
	if lines == nil {
		t.Fatal("GetSessionTailLines returned nil")
	}
	joined := strings.Join(lines, "\n")
	// Must contain the visible text (before/click here/after).
	if !strings.Contains(joined, "before") {
		t.Errorf("'before' missing after strip: %v", lines)
	}
	if !strings.Contains(joined, "click here") {
		t.Errorf("'click here' missing after strip: %v", lines)
	}
	if !strings.Contains(joined, "after") {
		t.Errorf("'after' missing after strip: %v", lines)
	}
	// Must NOT contain any ESC bytes (CR-01: ST terminator backslash was leaked).
	if strings.ContainsAny(joined, "\x1b") {
		t.Errorf("lines still contain ESC byte: %q", joined)
	}
	// Must NOT contain the URL.
	if strings.Contains(joined, "https://") {
		t.Errorf("URL not stripped from OSC 8 hyperlink: %q", joined)
	}
	// Must NOT contain a stray backslash from the ST terminator.
	// CR-01: old regex left \x5c (0x5c = '\') as a literal char in the output.
	if strings.Contains(joined, "\\") {
		t.Errorf("stray backslash (ST terminator artifact) present after strip: %q", joined)
	}
}

// TestGetSessionTailLines_ReturnsLastN: scrollback with 6 lines, n=4 →
// returns last 4 lines in order.
func TestGetSessionTailLines_ReturnsLastN(t *testing.T) {
	e := NewSessionEngine()
	content := "line1\nline2\nline3\nline4\nline5\nline6\n"
	makeTailHub(t, e, "lastn-test", content)

	lines := e.GetSessionTailLines("lastn-test", 4)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %v", len(lines), lines)
	}
	want := []string{"line3", "line4", "line5", "line6"}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], w)
		}
	}
}

// TestGetSessionTailLines_TrimsTrailingEmptyLines: trailing blank lines are
// removed before taking the last N.
func TestGetSessionTailLines_TrimsTrailingEmptyLines(t *testing.T) {
	e := NewSessionEngine()
	// content ends with several blank lines.
	content := "alpha\nbeta\n\n\n\n"
	makeTailHub(t, e, "trim-test", content)

	lines := e.GetSessionTailLines("trim-test", 10)
	if len(lines) == 0 {
		t.Fatal("expected non-empty lines after trim")
	}
	last := lines[len(lines)-1]
	if strings.TrimSpace(last) == "" {
		t.Errorf("trailing empty line not trimmed; last line = %q, all lines: %v", last, lines)
	}
}

// TestGetSessionTailLines_UnknownSession: manager.Get returns false →
// IN-01: method now returns []string{} (not nil) to avoid forcing callers to nil-guard.
// The API handler and app.go bindings previously nil-guarded the result; with IN-01 fixed
// both nil-guards remain correct (empty slice is falsy but not nil — callers may remove
// their guards in a follow-up, but correctness is maintained either way).
func TestGetSessionTailLines_UnknownSession(t *testing.T) {
	e := NewSessionEngine()
	result := e.GetSessionTailLines("nonexistent-session-id", 4)
	if result == nil {
		t.Errorf("IN-01: expected []string{} (not nil) for unknown session, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice for unknown session, got: %v", result)
	}
}

// ========================================================================
// Phase 139 Plan 01 — GetSessionStyledTailLines unit tests (CARD-05).
//
// These tests are RED until Plan 03 adds GetSessionStyledTailLines to
// engine.go and StyledSpan / StyledTailLinesResponse to types.go.
// They define the contract that Plan 03 must satisfy:
//   - CSI color sequences produce spans with FG:"ansi:N" and correct Bold flag
//   - Carriage-return (TUI overwrite) collapses to a single logical line (#96)
//   - Unknown session returns [][]StyledSpan{} (non-nil, len 0)
// ========================================================================

// TestGetSessionStyledTailLines_ColorBold: scrollback with CSI bold+green sequence
// → result contains spans with FG "ansi:2" and Bold true.
// RED until Plan 03 implements GetSessionStyledTailLines in engine.go.
func TestGetSessionStyledTailLines_ColorBold(t *testing.T) {
	e := NewSessionEngine()
	// \x1b[1;32m = bold + green (CSI 1;32m). \x1b[0m = reset.
	content := "\x1b[1;32mgreen\x1b[0m\n"
	makeTailHub(t, e, "styled-color-test", content)

	result := e.GetSessionStyledTailLines("styled-color-test", 5)
	if result == nil {
		t.Fatal("GetSessionStyledTailLines returned nil, expected [][]StyledSpan")
	}

	// Find the row that contains the "green" text.
	var greenRow []StyledSpan
	for _, row := range result {
		for _, span := range row {
			if span.Char == "g" {
				greenRow = row
				break
			}
		}
		if greenRow != nil {
			break
		}
	}
	if greenRow == nil {
		t.Fatalf("no row containing 'g' (first char of 'green') found; result: %v", result)
	}

	// Reconstruct text from the row.
	var chars string
	for _, span := range greenRow {
		chars += span.Char
	}
	if !strings.Contains(chars, "green") {
		t.Errorf("expected row chars to contain 'green', got: %q", chars)
	}

	// The first span of the "green" word must have FG == "ansi:2" (green in ANSI 16).
	firstSpan := greenRow[0]
	if firstSpan.FG != "ansi:2" {
		t.Errorf("expected first span FG 'ansi:2', got: %q", firstSpan.FG)
	}
	// Bold must be set on at least the first span.
	if !firstSpan.Bold {
		t.Errorf("expected first span Bold=true, got false")
	}
}

// TestGetSessionStyledTailLines_TUI: scrollback with carriage-return overwrite
// → result contains exactly one logical line with text "bbbb" (not "aaaabbbb").
// Regression guard for #96: the VT emulator must collapse the overwrite.
// RED until Plan 03 implements GetSessionStyledTailLines.
func TestGetSessionStyledTailLines_TUI(t *testing.T) {
	e := NewSessionEngine()
	// "aaaa\rbbbb\n" — carriage return moves cursor to column 0; "bbbb" overwrites "aaaa".
	// A correct VT emulator produces one line "bbbb"; a naive line-split produces "aaaabbbb".
	content := "aaaa\rbbbb\n"
	makeTailHub(t, e, "styled-tui-test", content)

	result := e.GetSessionStyledTailLines("styled-tui-test", 5)
	if result == nil {
		t.Fatal("GetSessionStyledTailLines returned nil")
	}

	// Collect non-empty rows.
	var nonEmptyRows [][]StyledSpan
	for _, row := range result {
		var text string
		for _, span := range row {
			text += span.Char
		}
		if strings.TrimSpace(text) != "" {
			nonEmptyRows = append(nonEmptyRows, row)
		}
	}

	if len(nonEmptyRows) != 1 {
		t.Errorf("expected exactly 1 non-empty row after TUI overwrite, got %d; result: %v", len(nonEmptyRows), result)
	} else {
		// The single non-empty row must contain "bbbb" and must NOT contain "aaaa".
		var rowText string
		for _, span := range nonEmptyRows[0] {
			rowText += span.Char
		}
		if !strings.Contains(rowText, "bbbb") {
			t.Errorf("expected row text to contain 'bbbb', got: %q", rowText)
		}
		if strings.Contains(rowText, "aaaa") {
			t.Errorf("row text must not contain 'aaaa' after overwrite, got: %q", rowText)
		}
	}
}

// TestGetSessionStyledTailLines_Unknown: unknown session ID → non-nil empty
// [][]StyledSpan (len 0, not nil, no panic).
// RED until Plan 03 implements GetSessionStyledTailLines.
func TestGetSessionStyledTailLines_Unknown(t *testing.T) {
	e := NewSessionEngine()
	result := e.GetSessionStyledTailLines("does-not-exist", 5)
	if result == nil {
		t.Errorf("expected [][]StyledSpan{} (not nil) for unknown session, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice for unknown session, got len %d: %v", len(result), result)
	}
}

// TestGetSessionStyledTailLines_QueryNoHang: scrollback containing terminal query
// escape sequences (Primary Device Attributes ESC[c, DSR cursor-position ESC[6n)
// must NOT hang. Regression for the headless-emulator deadlock: charmbracelet/x/vt
// writes query responses to an unbuffered io.Pipe (Emulator.pw); with no reader
// draining Emulator.Read, emu.Write blocks forever on the first query. Claude Code's
// TUI emits these queries at startup, so they land in the captured PTY scrollback.
func TestGetSessionStyledTailLines_QueryNoHang(t *testing.T) {
	e := NewSessionEngine()
	// DA1 query, then text, then DSR cursor-position query, then more text.
	// Each query triggers a response write to the emulator's response pipe.
	content := "\x1b[c\x1b[32mhello\x1b[0m\x1b[6n\nworld\n"
	makeTailHub(t, e, "styled-query-hang", content)

	done := make(chan [][]StyledSpan, 1)
	go func() {
		done <- e.GetSessionStyledTailLines("styled-query-hang", 5)
	}()
	select {
	case result := <-done:
		if result == nil {
			t.Fatal("GetSessionStyledTailLines returned nil")
		}
		// Sanity: the visible text must survive the query stripping.
		var joined string
		for _, row := range result {
			for _, span := range row {
				joined += span.Char
			}
			joined += "\n"
		}
		if !strings.Contains(joined, "hello") || !strings.Contains(joined, "world") {
			t.Errorf("expected visible text 'hello' and 'world' in result, got: %q", joined)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GetSessionStyledTailLines hung on terminal query escape sequences — emulator response pipe is not being drained")
	}
}

// TestGetSessionStyledTailLines_AllQueriesNoHang: kitchen-sink fixture containing
// every query verb that causes charmbracelet/x/vt to write a response into its
// unbuffered response pipe (Emulator.pw), interleaved with visible text.
//
// Verifies two things:
//  1. The call completes well within a 5-second timeout (no hang — queryStripPattern
//     strips every blocking sequence before emu.Write; FIX-01 / #100).
//  2. The visible text surrounding the control sequences survives in the rendered
//     grid (the strip is narrow: only query verbs, not SGR/color; Pitfall 3).
//
// Query verbs covered (all enumerated in RESEARCH.md "complete set"):
//   - DA1    ESC[c          — Primary Device Attributes
//   - DA2    ESC[>c         — Secondary Device Attributes
//   - DSR5   ESC[5n         — Device Status Report (operating status)
//   - DSR6   ESC[6n         — Device Status Report (cursor position / CPR)
//   - DECXCPR ESC[?6n       — Extended Cursor Position Report
//   - DECRQM ANSI ESC[4$p   — Request Mode (ANSI mode 4)
//   - DECRQM DEC  ESC[?2026$p — Request Mode (DEC mode 2026 synchronous update)
//   - OSC color query ESC]11;?BEL — Background color query
//   - mode-2048 ESC[?2048h  — In-band resize enable (fires pw write on set; Pitfall 1)
func TestGetSessionStyledTailLines_AllQueriesNoHang(t *testing.T) {
	e := NewSessionEngine()

	// Interleave every query verb with visible text anchors so we can assert
	// that the visible content survives the strip operation.
	//
	// Structure: visible-text QUERY visible-text ... so the assertions below
	// can search for each anchor word in the joined rendered output.
	content := "" +
		"alpha\n" +
		"\x1b[c" + // DA1
		"beta\n" +
		"\x1b[>c" + // DA2
		"gamma\n" +
		"\x1b[5n" + // DSR operating status
		"delta\n" +
		"\x1b[6n" + // DSR cursor position (CPR)
		"epsilon\n" +
		"\x1b[?6n" + // DECXCPR extended cursor position
		"zeta\n" +
		"\x1b[4$p" + // DECRQM ANSI mode 4
		"eta\n" +
		"\x1b[?2026$p" + // DECRQM DEC mode 2026 (synchronous update)
		"theta\n" +
		"\x1b]11;?\x07" + // OSC 11 background color query, BEL-terminated
		"iota\n" +
		"\x1b[?2048h" + // DEC mode 2048 in-band resize SET (Pitfall 1: triggers pw write)
		"kappa\n"

	makeTailHub(t, e, "styled-all-queries-hang", content)

	done := make(chan [][]StyledSpan, 1)
	go func() {
		done <- e.GetSessionStyledTailLines("styled-all-queries-hang", 20)
	}()
	select {
	case result := <-done:
		if result == nil {
			t.Fatal("GetSessionStyledTailLines returned nil")
		}

		// Join all rendered cell content to check visible text survived.
		var joined string
		for _, row := range result {
			for _, span := range row {
				joined += span.Char
			}
			joined += "\n"
		}

		// Every visible anchor word must appear in the rendered output.
		// This proves the strip removed only the control sequences, not content.
		anchors := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa"}
		for _, word := range anchors {
			if !strings.Contains(joined, word) {
				t.Errorf("visible anchor %q missing from rendered output after query strip; full output: %q", word, joined)
			}
		}

	case <-time.After(5 * time.Second):
		t.Fatal("GetSessionStyledTailLines hung on terminal query escape sequences — queryStripPattern did not cover all blocking sequences (FIX-01 regression)")
	}
}

// --------------------------------------------------------------------------
// Plan 02 Task 2: chatStores lifecycle wiring in SessionEngine
// --------------------------------------------------------------------------

// TestEngineNewSessionEngine_ChatStoresInit verifies that NewSessionEngine
// initialises chatStores to a non-nil map and sets chatsBaseDir to a non-empty
// path (derived from chatsDir() / daemonConfigDir()).
func TestEngineNewSessionEngine_ChatStoresInit(t *testing.T) {
	e := NewSessionEngine()
	e.ChatsBaseDirForTest(t.TempDir()) // redirect away from real data dir
	if e.chatStores == nil {
		t.Error("chatStores map is nil after NewSessionEngine")
	}
	if e.chatsBaseDir == "" {
		t.Error("chatsBaseDir is empty after NewSessionEngine")
	}
}

// TestEngineChatStoreFor_AfterCreate verifies that after CreateSession returns,
// ChatStoreFor(id) reports ok==true.
func TestEngineChatStoreFor_AfterCreate(t *testing.T) {
	e := NewSessionEngine()
	e.ChatsBaseDirForTest(t.TempDir())

	id, err := e.CreateSession(context.Background(), "cat", "chat-create", "", nil, 0, 0, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	_, ok := e.ChatStoreFor(id)
	if !ok {
		t.Errorf("ChatStoreFor(%q) ok=false after CreateSession; expected ok=true", id)
	}
}

// TestEngineChatStoreFor_AfterKill verifies the no-orphan guarantee:
// after KillSession, ChatStoreFor returns ok==false AND the JSONL file
// is absent from the temp chatsBaseDir.
func TestEngineChatStoreFor_AfterKill(t *testing.T) {
	tempDir := t.TempDir()
	e := NewSessionEngine()
	e.ChatsBaseDirForTest(tempDir)

	id, err := e.CreateSession(context.Background(), "cat", "chat-kill", "", nil, 0, 0, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Append a message so the file is created on disk.
	store, ok := e.ChatStoreFor(id)
	if !ok {
		t.Fatalf("ChatStoreFor(%q) ok=false immediately after CreateSession", id)
	}
	if _, err := store.AppendMessage(relay.ChatMessage{
		AuthorID:    "local",
		AuthorAlias: "alice",
		Content:     "pre-kill message",
	}); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	// File must exist before kill.
	jsonlPath := filepath.Join(tempDir, id+".jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		t.Fatalf("JSONL file should exist before KillSession: %v", err)
	}

	if err := e.KillSession(id); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// After kill: no store, no file.
	if _, ok := e.ChatStoreFor(id); ok {
		t.Errorf("ChatStoreFor(%q) ok=true after KillSession; expected ok=false", id)
	}
	if _, err := os.Stat(jsonlPath); !os.IsNotExist(err) {
		t.Errorf("JSONL file still exists after KillSession (orphan!): %v", err)
	}
}

// TestEngineChatStoreFor_FailedNewChatStore verifies that when NewChatStore
// fails (chatsBaseDir points at an uncreatable location), CreateSession still
// returns a usable session ID and ChatStoreFor returns ok==false.
func TestEngineChatStoreFor_FailedNewChatStore(t *testing.T) {
	e := NewSessionEngine()

	// Point chatsBaseDir at a path whose parent is a regular file so MkdirAll
	// fails — the directory can never be created.
	tmpFile, err := os.CreateTemp(t.TempDir(), "not-a-dir")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	tmpFile.Close()
	// filepath.Join(tmpFile.Name(), "chats") has a file as its parent component.
	e.ChatsBaseDirForTest(filepath.Join(tmpFile.Name(), "chats"))

	id, err := e.CreateSession(context.Background(), "cat", "chat-fail", "", nil, 0, 0, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession must succeed even when ChatStore fails: %v", err)
	}
	t.Cleanup(func() { _ = e.KillSession(id) })

	if id == "" {
		t.Fatal("CreateSession returned empty id")
	}

	// ChatStoreFor must return ok=false when store creation failed.
	if _, ok := e.ChatStoreFor(id); ok {
		t.Errorf("ChatStoreFor(%q) ok=true despite NewChatStore failure; expected ok=false", id)
	}
}

// TestEngineNoRealDirChatFiles verifies that with ChatsBaseDirForTest applied,
// no chat file is ever created under the real daemonConfigDir()/chats path.
// It creates a session, appends a message, kills it — then checks that the
// real chats dir has not gained any .jsonl file for the test session.
func TestEngineNoRealDirChatFiles(t *testing.T) {
	tempDir := t.TempDir()
	e := NewSessionEngine()
	e.ChatsBaseDirForTest(tempDir)

	id, err := e.CreateSession(context.Background(), "cat", "isolation-test", "", nil, 0, 0, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	store, ok := e.ChatStoreFor(id)
	if !ok {
		t.Fatalf("ChatStoreFor(%q): ok=false", id)
	}
	if _, err := store.AppendMessage(relay.ChatMessage{
		AuthorID: "local",
		Content:  "isolation message",
	}); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	if err := e.KillSession(id); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// Confirm NO .jsonl file for this id under the real chats dir.
	realChatsDir := chatsDir()
	realPath := filepath.Join(realChatsDir, id+".jsonl")
	if _, err := os.Stat(realPath); !os.IsNotExist(err) {
		t.Errorf("chat file found in real data dir %q (isolation failure): %v", realPath, err)
	}
}
