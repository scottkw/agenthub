package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestNotifyOnWaiting_Default verifies that a fresh engine with no
// settings.json reports NotifyOnWaiting as false (default OFF, NTF-04).
func TestNotifyOnWaiting_Default(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	if got := e.GetNotifyOnWaiting(); got != false {
		t.Errorf("default GetNotifyOnWaiting: got %v, want false", got)
	}
}

// TestNotifyOnWaiting_Persists verifies that SetNotifyOnWaiting(true) writes
// to settings.json and survives a reload from the same path.
func TestNotifyOnWaiting_Persists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	e.SetNotifyOnWaiting(true)

	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not found after SetNotifyOnWaiting(true): %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !s.NotifyOnWaiting {
		t.Errorf("settings.json after SetNotifyOnWaiting(true): got %v, want true", s.NotifyOnWaiting)
	}
	if s.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("schemaVersion after SetNotifyOnWaiting: got %d, want unchanged %d", s.SchemaVersion, CurrentSchemaVersion)
	}

	e2 := &SessionEngine{configDir: dir, cliPaths: make(map[string]string)}
	e2.loadSettingsFromDisk(dir)
	if got := e2.GetNotifyOnWaiting(); got != true {
		t.Errorf("reload after SetNotifyOnWaiting(true): got %v, want true", got)
	}
}

// TestNotifyOnWaiting_RoundTrip verifies SetNotifyOnWaiting(true) then
// SetNotifyOnWaiting(false) reports false and rewrites disk.
func TestNotifyOnWaiting_RoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	e.SetNotifyOnWaiting(true)
	if got := e.GetNotifyOnWaiting(); got != true {
		t.Fatalf("after SetNotifyOnWaiting(true): got %v, want true", got)
	}

	e.SetNotifyOnWaiting(false)
	if got := e.GetNotifyOnWaiting(); got != false {
		t.Errorf("after SetNotifyOnWaiting(false): got %v, want false", got)
	}

	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not found after round-trip: %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.NotifyOnWaiting {
		t.Errorf("settings.json after round-trip to false: got %v, want false", s.NotifyOnWaiting)
	}
}

// TestNotifyOnWaiting_NoSchemaBump verifies that adding NotifyOnWaiting did
// not bump CurrentSchemaVersion and that loading an existing settings.json
// (already at CurrentSchemaVersion, no notifyOnWaiting key) does not trigger
// an unnecessary defaults-merge rewrite — RESEARCH Pitfall 4.
func TestNotifyOnWaiting_NoSchemaBump(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	existing := daemonSettings{
		Plugins:       defaultPluginSettings(),
		SchemaVersion: CurrentSchemaVersion,
	}
	data, err := json.Marshal(existing)
	if err != nil {
		t.Fatalf("marshal existing settings: %v", err)
	}
	settingsFile := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsFile, data, 0600); err != nil {
		t.Fatalf("write existing settings.json: %v", err)
	}
	infoBefore, err := os.Stat(settingsFile)
	if err != nil {
		t.Fatalf("stat before load: %v", err)
	}

	e := &SessionEngine{configDir: dir, cliPaths: make(map[string]string)}
	e.loadSettingsFromDisk(dir)
	if got := e.GetNotifyOnWaiting(); got != false {
		t.Errorf("NotifyOnWaiting on pre-existing settings.json (no key): got %v, want false", got)
	}

	infoAfter, err := os.Stat(settingsFile)
	if err != nil {
		t.Fatalf("stat after load: %v", err)
	}
	if infoAfter.ModTime().After(infoBefore.ModTime()) {
		t.Errorf("loadSettingsFromDisk rewrote settings.json for an up-to-date file (unexpected defaults-merge rewrite)")
	}
}
