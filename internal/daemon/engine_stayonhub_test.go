package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestStayOnHubAfterCreate_Default verifies that a fresh engine with no
// settings.json reports StayOnHubAfterCreate as false (default OFF, D-09).
func TestStayOnHubAfterCreate_Default(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	if got := e.GetStayOnHubAfterCreate(); got != false {
		t.Errorf("default GetStayOnHubAfterCreate: got %v, want false", got)
	}
}

// TestStayOnHubAfterCreate_Persists verifies that SetStayOnHubAfterCreate(true)
// writes to settings.json and survives a reload from the same path.
func TestStayOnHubAfterCreate_Persists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	e.SetStayOnHubAfterCreate(true)

	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not found after SetStayOnHubAfterCreate(true): %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !s.StayOnHubAfterCreate {
		t.Errorf("settings.json after SetStayOnHubAfterCreate(true): got %v, want true", s.StayOnHubAfterCreate)
	}
	if s.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("schemaVersion after SetStayOnHubAfterCreate: got %d, want unchanged %d", s.SchemaVersion, CurrentSchemaVersion)
	}

	e2 := &SessionEngine{configDir: dir, cliPaths: make(map[string]string)}
	e2.loadSettingsFromDisk(dir)
	if got := e2.GetStayOnHubAfterCreate(); got != true {
		t.Errorf("reload after SetStayOnHubAfterCreate(true): got %v, want true", got)
	}
}

// TestStayOnHubAfterCreate_RoundTrip verifies SetStayOnHubAfterCreate(true)
// then SetStayOnHubAfterCreate(false) reports false and rewrites disk.
func TestStayOnHubAfterCreate_RoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses temp file paths")
	}
	dir := t.TempDir()
	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}

	e.SetStayOnHubAfterCreate(true)
	if got := e.GetStayOnHubAfterCreate(); got != true {
		t.Fatalf("after SetStayOnHubAfterCreate(true): got %v, want true", got)
	}

	e.SetStayOnHubAfterCreate(false)
	if got := e.GetStayOnHubAfterCreate(); got != false {
		t.Errorf("after SetStayOnHubAfterCreate(false): got %v, want false", got)
	}

	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not found after round-trip: %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.StayOnHubAfterCreate {
		t.Errorf("settings.json after round-trip to false: got %v, want false", s.StayOnHubAfterCreate)
	}
}

// TestStayOnHubAfterCreate_NoSchemaBump verifies that adding
// StayOnHubAfterCreate did not bump CurrentSchemaVersion and that loading an
// existing settings.json (already at CurrentSchemaVersion, no
// stayOnHubAfterCreate key) does not trigger an unnecessary defaults-merge
// rewrite.
func TestStayOnHubAfterCreate_NoSchemaBump(t *testing.T) {
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
	if got := e.GetStayOnHubAfterCreate(); got != false {
		t.Errorf("StayOnHubAfterCreate on pre-existing settings.json (no key): got %v, want false", got)
	}

	infoAfter, err := os.Stat(settingsFile)
	if err != nil {
		t.Fatalf("stat after load: %v", err)
	}
	if infoAfter.ModTime().After(infoBefore.ModTime()) {
		t.Errorf("loadSettingsFromDisk rewrote settings.json for an up-to-date file (unexpected defaults-merge rewrite)")
	}
}
