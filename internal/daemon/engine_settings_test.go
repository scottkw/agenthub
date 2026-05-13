package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSettingsPersistence(t *testing.T) {
	dir := t.TempDir()

	// Create a fake executable for UpdateCLIPath's os.Stat check
	fakeBin := filepath.Join(dir, "fake-claude")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create engine with temp configDir
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	// Update a CLI path (SET-01: agent path persistence)
	if err := e.UpdateCLIPath("claude", fakeBin); err != nil {
		t.Fatalf("UpdateCLIPath: %v", err)
	}

	// Verify settings.json was written
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not found: %v", err)
	}

	var s daemonSettings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal settings.json: %v", err)
	}
	if s.CLIPaths["claude"] != fakeBin {
		t.Errorf("persisted claude path = %q, want %q", s.CLIPaths["claude"], fakeBin)
	}

	// Create a new engine from same dir — verify it loads the saved paths
	e2 := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e2.loadSettingsFromDisk(dir)

	got := e2.GetCLIPaths()
	if got["claude"] != fakeBin {
		t.Errorf("loaded claude path = %q, want %q", got["claude"], fakeBin)
	}
}

func TestTailscalePathPersistence(t *testing.T) {
	dir := t.TempDir()

	fakeTailscale := filepath.Join(dir, "tailscale")
	if err := os.WriteFile(fakeTailscale, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatal(err)
	}

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	// SET-02: tailscale path persistence
	if err := e.UpdateCLIPath("tailscale", fakeTailscale); err != nil {
		t.Fatalf("UpdateCLIPath tailscale: %v", err)
	}

	e2 := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e2.loadSettingsFromDisk(dir)

	got := e2.GetCLIPaths()
	if got["tailscale"] != fakeTailscale {
		t.Errorf("loaded tailscale path = %q, want %q", got["tailscale"], fakeTailscale)
	}
}

func TestSettingsLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	// Should not panic or error — missing file is first-run
	e.loadSettingsFromDisk(dir)
	if len(e.cliPaths) != 0 {
		t.Errorf("expected empty cliPaths, got %v", e.cliPaths)
	}
}

func TestStartMinimizedPersistence(t *testing.T) {
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	// Default is false
	if e.GetStartMinimized() {
		t.Error("expected default startMinimized=false")
	}

	// Set to true and verify settings.json
	e.SetStartMinimized(true)

	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not found: %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !s.StartMinimized {
		t.Error("expected startMinimized=true in settings.json")
	}

	// Create new engine, load from disk, verify
	e2 := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e2.loadSettingsFromDisk(dir)
	if !e2.GetStartMinimized() {
		t.Error("expected loaded startMinimized=true")
	}

	// Set back to false, reload, verify
	e2.SetStartMinimized(false)
	e3 := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e3.loadSettingsFromDisk(dir)
	if e3.GetStartMinimized() {
		t.Error("expected loaded startMinimized=false after set(false)")
	}
}

func TestStartMinimizedWithoutCLIPaths(t *testing.T) {
	dir := t.TempDir()
	// Write settings.json with only startMinimized (no cliPaths key)
	data := []byte(`{"startMinimized":true}`)
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir)
	if !e.GetStartMinimized() {
		t.Error("expected startMinimized=true when loaded without cliPaths in JSON")
	}
}

func TestLoadSettingsDropsStaleShellPaths(t *testing.T) {
	dir := t.TempDir()

	// Write settings.json with stale shell paths (e.g. "claude" → "/bin/sh")
	data := []byte(`{"cliPaths":{"claude":"/bin/sh","opencode":"/bin/sh","sh":"/bin/sh"},"startMinimized":true}`)
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), data, 0600); err != nil {
		t.Fatal(err)
	}

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir)

	// claude=/bin/sh should be dropped (basename "sh" ≠ "claude")
	if _, ok := e.cliPaths["claude"]; ok {
		t.Errorf("expected stale claude=/bin/sh to be dropped, got %q", e.cliPaths["claude"])
	}
	// opencode=/bin/sh should be dropped (basename "sh" ≠ "opencode")
	if _, ok := e.cliPaths["opencode"]; ok {
		t.Errorf("expected stale opencode=/bin/sh to be dropped, got %q", e.cliPaths["opencode"])
	}
	// sh=/bin/sh should be KEPT (basename "sh" == "sh")
	if e.cliPaths["sh"] != "/bin/sh" {
		t.Errorf("expected sh=/bin/sh to be kept, got %q", e.cliPaths["sh"])
	}
	// startMinimized should still load
	if !e.GetStartMinimized() {
		t.Error("expected startMinimized=true")
	}

	// Verify settings.json was rewritten without stale entries
	rewritten, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s daemonSettings
	if err := json.Unmarshal(rewritten, &s); err != nil {
		t.Fatalf("unmarshal rewritten: %v", err)
	}
	if _, ok := s.CLIPaths["claude"]; ok {
		t.Error("rewritten settings.json still contains claude")
	}
	if _, ok := s.CLIPaths["opencode"]; ok {
		t.Error("rewritten settings.json still contains opencode")
	}
}

// TestSetShellPathValidation covers WR-01: SetShellPath must reject directories
// (every POSIX directory has its execute bit set, so the Mode()&0111 check alone
// is insufficient — IsDir() must be checked first).
func TestSetShellPathValidation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file-mode semantics differ; IsDir check is still applied but exec-bit test is skipped")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	// WR-01: a directory path must be rejected with "is a directory" error
	err := e.SetShellPath(dir) // dir itself is a directory
	if err == nil {
		t.Fatal("SetShellPath(directory) expected error, got nil")
	}
	if !containsSubstr(err.Error(), "is a directory") {
		t.Errorf("SetShellPath(directory) error = %q, want message containing 'is a directory'", err.Error())
	}

	// A non-existent path must still be rejected
	err = e.SetShellPath(filepath.Join(dir, "no-such-file"))
	if err == nil {
		t.Fatal("SetShellPath(nonexistent) expected error, got nil")
	}

	// A file with no execute bit must be rejected
	noExec := filepath.Join(dir, "no-exec")
	if err := os.WriteFile(noExec, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	err = e.SetShellPath(noExec)
	if err == nil {
		t.Fatal("SetShellPath(non-executable file) expected error, got nil")
	}

	// A valid executable file must be accepted
	fakeBin := filepath.Join(dir, "fake-shell")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := e.SetShellPath(fakeBin); err != nil {
		t.Fatalf("SetShellPath(valid executable) unexpected error: %v", err)
	}
	if e.GetShellPath() != fakeBin {
		t.Errorf("GetShellPath() = %q, want %q", e.GetShellPath(), fakeBin)
	}

	// Clearing with empty string must be accepted (restores platform default)
	if err := e.SetShellPath(""); err != nil {
		t.Fatalf("SetShellPath(\"\") unexpected error: %v", err)
	}
	// After clearing, GetShellPath returns the platform default (non-empty)
	if e.GetShellPath() == "" {
		t.Error("GetShellPath() returned empty after clearing override — expected platform default")
	}
}

// containsSubstr is a local helper used by TestSetShellPathValidation to avoid
// importing strings just for Contains.
func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

func TestSettingsFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Unix file permissions")
	}
	dir := t.TempDir()

	fakeBin := filepath.Join(dir, "fake-bin")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatal(err)
	}

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	if err := e.UpdateCLIPath("claude", fakeBin); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("settings.json permissions = %o, want 0600", perm)
	}
}
