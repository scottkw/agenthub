package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestShellWebShareWarningEnabled_Default verifies that a fresh engine reports
// the warning-enabled switch as ON (true) by default — D-08 requirement.
func TestShellWebShareWarningEnabled_Default(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix socket testDaemon helpers")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	// No settings.json → nil pointer → must default to true.
	if got := e.GetShellWebShareWarningEnabled(); got != true {
		t.Errorf("default GetShellWebShareWarningEnabled: got %v, want true", got)
	}
}

// TestShellWebShareWarningEnabled_Persists verifies that Set(false) round-trips
// through settings.json (reload observes false), and Set(true) round-trips as true.
func TestShellWebShareWarningEnabled_Persists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	// Set false, verify settings.json, reload, verify.
	if err := e.SetShellWebShareWarningEnabled(false); err != nil {
		t.Fatalf("SetShellWebShareWarningEnabled(false): %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not found after Set(false): %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.ShellWebShareWarningEnabled == nil || *s.ShellWebShareWarningEnabled != false {
		t.Errorf("settings.json after Set(false): got %v, want pointer-to-false", s.ShellWebShareWarningEnabled)
	}

	e2 := &SessionEngine{configDir: dir, cliPaths: make(map[string]string)}
	e2.loadSettingsFromDisk(dir)
	if got := e2.GetShellWebShareWarningEnabled(); got != false {
		t.Errorf("reload after Set(false): got %v, want false", got)
	}

	// Set true, reload, verify.
	if err := e2.SetShellWebShareWarningEnabled(true); err != nil {
		t.Fatalf("SetShellWebShareWarningEnabled(true): %v", err)
	}
	e3 := &SessionEngine{configDir: dir, cliPaths: make(map[string]string)}
	e3.loadSettingsFromDisk(dir)
	if got := e3.GetShellWebShareWarningEnabled(); got != true {
		t.Errorf("reload after Set(true): got %v, want true", got)
	}
}

// TestShellWebShareWarningEnabled_ReArm verifies D-03: setting warned=true then
// calling SetShellWebShareWarningEnabled(true) atomically resets shellWebShareWarned
// to false (re-arm).
func TestShellWebShareWarningEnabled_ReArm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	// Simulate user having acknowledged the warning.
	if err := e.SetShellWebShareWarned(true); err != nil {
		t.Fatalf("SetShellWebShareWarned(true): %v", err)
	}
	if got := e.GetShellWebShareWarned(); got != true {
		t.Errorf("pre-rearm GetShellWebShareWarned: got %v, want true", got)
	}

	// Enable the master switch — should atomically re-arm (reset warned to false).
	if err := e.SetShellWebShareWarningEnabled(true); err != nil {
		t.Fatalf("SetShellWebShareWarningEnabled(true): %v", err)
	}
	if got := e.GetShellWebShareWarned(); got != false {
		t.Errorf("GetShellWebShareWarned after re-arm: got %v, want false (D-03)", got)
	}
}

// TestShellWebShareWarningEnabled_OffBehavior verifies that Set(false) does NOT
// reset shellWebShareWarned — only enabling (true) triggers re-arm.
func TestShellWebShareWarningEnabled_OffBehavior(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	// User has acknowledged the warning.
	if err := e.SetShellWebShareWarned(true); err != nil {
		t.Fatalf("SetShellWebShareWarned(true): %v", err)
	}
	// Disable the master switch — warned flag must remain true.
	if err := e.SetShellWebShareWarningEnabled(false); err != nil {
		t.Fatalf("SetShellWebShareWarningEnabled(false): %v", err)
	}
	if got := e.GetShellWebShareWarned(); got != true {
		t.Errorf("GetShellWebShareWarned after Set(false): got %v, want true (Set(false) must NOT reset warned)", got)
	}
}
