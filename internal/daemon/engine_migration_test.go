package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixtureV31Path returns the on-disk location of the realistic v3.1 settings
// fixture. The Go test working directory is the package directory
// (internal/daemon/), so the fixture lives two levels up.
func fixtureV31Path(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "tests", "fixtures", "settings_v3.1.json")
}

// copyFixtureToTempDir reads the v3.1 fixture and writes it as settings.json
// inside dir. Returns the destination path for convenience.
func copyFixtureToTempDir(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(fixtureV31Path(t))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	dst := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(dst, data, 0600); err != nil {
		t.Fatalf("write fixture into temp: %v", err)
	}
	return dst
}

// TestSettingsMigrationV3_1ToV3_2 is the load-bearing CI gate per ROADMAP SC-1.
// A returning v3.1 user (no plugins block, no schemaVersion key) must:
//  1. Load with the 8 v3.2 plugin defaults (7 ON, Progress OFF) populated.
//  2. Trigger a re-write so the on-disk settings.json now contains both the
//     plugins block and schemaVersion: 2.
//  3. Preserve existing v3.1 keys (cliPaths) — defaults-merge must not zero
//     existing data.
func TestSettingsMigrationV3_1ToV3_2(t *testing.T) {
	dir := t.TempDir()
	copyFixtureToTempDir(t, dir)

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir)

	// In-memory: the 8 v3.2 defaults must be populated (Pitfall #14 mitigation).
	got := e.GetPluginSettings()
	want := defaultPluginSettings()
	if got != want {
		t.Errorf("GetPluginSettings after v3.1 load: got %+v, want %+v", got, want)
	}

	// Pre-existing v3.1 data must survive the migration.
	if cli := e.GetCLIPaths(); cli["claude"] != "/usr/local/bin/claude" {
		t.Errorf("cliPaths not preserved through migration: got %v", cli)
	}

	// On-disk: the upgrade re-write fired, so the file now contains both the
	// plugins block AND schemaVersion: 2.
	raw, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("re-read settings.json: %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal rewritten settings: %v", err)
	}
	if s.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("on-disk schemaVersion = %d, want %d", s.SchemaVersion, CurrentSchemaVersion)
	}
	if !s.Plugins.WebGL {
		t.Errorf("on-disk plugins.webgl = false, want true (proves the upgrade re-write fired)")
	}
}

// TestSettingsMigrationIdempotent asserts that loading a settings.json that is
// already at CurrentSchemaVersion does NOT trigger an unnecessary re-write.
// We verify this by comparing the file's mtime across two consecutive loads:
// the first load performs the v3.1→v3.2 upgrade write; the second load is a
// no-op and must leave the mtime unchanged.
func TestSettingsMigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	copyFixtureToTempDir(t, dir)

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir) // first load: upgrade re-write fires

	settingsFile := filepath.Join(dir, "settings.json")
	info1, err := os.Stat(settingsFile)
	if err != nil {
		t.Fatalf("stat after first load: %v", err)
	}
	mtime1 := info1.ModTime()

	// Sleep so a stray re-write would produce a detectably different mtime.
	time.Sleep(10 * time.Millisecond)

	e2 := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e2.loadSettingsFromDisk(dir) // second load: should be a no-op (idempotent)

	info2, err := os.Stat(settingsFile)
	if err != nil {
		t.Fatalf("stat after second load: %v", err)
	}
	mtime2 := info2.ModTime()

	if !mtime1.Equal(mtime2) {
		t.Errorf("settings.json mtime changed across idempotent loads: %v -> %v (the second load triggered an unnecessary re-write)", mtime1, mtime2)
	}
}
