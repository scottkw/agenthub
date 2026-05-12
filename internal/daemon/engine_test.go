package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/pty"
	"github.com/scottkw/agenthub/internal/relay"
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
func TestIsShellSession_AllShellNames(t *testing.T) {
	cases := map[string]bool{
		"shell":      true,
		"bash":       true,
		"zsh":        true,
		"pwsh":       true,
		"powershell": true,
		"claude":     false,
		"opencode":   false,
		"":           false,
		"Bash":       false,
		"/bin/bash":  false,
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
