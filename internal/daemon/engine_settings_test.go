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
