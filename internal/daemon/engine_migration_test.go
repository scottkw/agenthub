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

	// Phase 99 SC-3 — per-field assertions for diagnostic clarity when a v3.3 default
	// change breaks the migration contract. The struct-equality check above is the
	// "fast fail" sentinel; these per-field assertions name the failing field.
	if !got.WebGL {
		t.Errorf("plugin defaults: WebGL = false, want true")
	}
	if !got.Unicode11 {
		t.Errorf("plugin defaults: Unicode11 = false, want true")
	}
	if !got.Search {
		t.Errorf("plugin defaults: Search = false, want true")
	}
	if !got.WebLinks {
		t.Errorf("plugin defaults: WebLinks = false, want true")
	}
	if !got.Image {
		t.Errorf("plugin defaults: Image = false, want true")
	}
	if !got.Serialize {
		t.Errorf("plugin defaults: Serialize = false, want true")
	}
	if !got.Clipboard {
		t.Errorf("plugin defaults: Clipboard = false, want true")
	}
	if got.Progress {
		t.Errorf("plugin defaults: Progress = true, want false (default OFF in v3.2 — flips ON in v3.3 after field validation)")
	}

	// SearchConfig defaults (replaces the previous zero-value check — forward-compatible
	// for v3.3 when a non-zero default may be introduced).
	wantSearch := SearchConfig{Regex: false, CaseSensitive: false, WholeWord: false}
	if got.SearchConfig != wantSearch {
		t.Errorf("SearchConfig defaults: got %+v, want %+v", got.SearchConfig, wantSearch)
	}

	// WebLinksConfig defaults.
	wantWebLinks := WebLinksConfig{
		Modifier:         "platform",
		ConfirmOSC8:      true,
		ConfirmIDN:       true,
		ConfirmTyposquat: true,
	}
	if got.WebLinksConfig != wantWebLinks {
		t.Errorf("WebLinksConfig defaults: got %+v, want %+v", got.WebLinksConfig, wantWebLinks)
	}

	// ImageConfig defaults — StorageLimit override (16 MB; upstream default is 100 MB).
	wantImage := ImageConfig{StorageLimit: 16}
	if got.ImageConfig != wantImage {
		t.Errorf("ImageConfig defaults: got %+v, want %+v (v3.2 overrides upstream 100 MB to 16 MB to prevent tab OOM)", got.ImageConfig, wantImage)
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

// fixtureV32Path returns the on-disk location of the v3.2 settings fixture.
// Mirrors fixtureV31Path: the Go test working directory is the package
// directory (internal/daemon/), so the fixture lives two levels up.
// Phase 118 / FS-14: v3.2 fixture has plugins block + schemaVersion=2 but
// NO filesRead key — exercises the defaults-merge mitigation for Pitfall 16.
func fixtureV32Path(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "tests", "fixtures", "settings_v3.2.json")
}

// copyV32FixtureToTempDir reads the v3.2 fixture and writes it as settings.json
// inside dir.
func copyV32FixtureToTempDir(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(fixtureV32Path(t))
	if err != nil {
		t.Fatalf("read v3.2 fixture: %v", err)
	}
	dst := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(dst, data, 0600); err != nil {
		t.Fatalf("write v3.2 fixture into temp: %v", err)
	}
	return dst
}

// TestSettingsMigration_FilesReadDefaultsTrue verifies that a v3.2 settings.json
// (schemaVersion=2, plugins block, no filesRead key) loads with FilesRead = *true
// via the defaults-merge constructor pattern. Pitfall 16 mitigation: Go's
// encoding/json leaves missing keys at zero value, so without the pre-populated
// default, upgrading users would silently land with filesRead=false and lose
// access to their own session-cwd file lists.
func TestSettingsMigration_FilesReadDefaultsTrue(t *testing.T) {
	dir := t.TempDir()
	copyV32FixtureToTempDir(t, dir)

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir)

	if e.filesRead == nil {
		t.Fatalf("e.filesRead = nil after load; want *true (defaults-merge should populate)")
	}
	if !*e.filesRead {
		t.Errorf("e.filesRead = false after load; want true (v3.2 fixture has no filesRead key, default must be true)")
	}
}

// TestSettingsMigration_FilesReadExplicitFalse verifies that an explicit
// `"filesRead": false` in settings.json is NOT clobbered by the defaults-merge
// pre-population. This guards against the defaults-merge silently overriding
// explicit user choice (which would re-introduce Pitfall 16 in reverse).
func TestSettingsMigration_FilesReadExplicitFalse(t *testing.T) {
	dir := t.TempDir()
	// Synthetic settings.json with explicit filesRead: false.
	settings := `{
		"schemaVersion": 2,
		"filesRead": false,
		"plugins": {
			"webgl": true,
			"unicode11": true,
			"search": true,
			"webLinks": true,
			"image": true,
			"serialize": true,
			"clipboard": true,
			"progress": false
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(settings), 0600); err != nil {
		t.Fatalf("write synthetic settings: %v", err)
	}

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir)

	if e.filesRead == nil {
		t.Fatalf("e.filesRead = nil after load with explicit false; want *false")
	}
	if *e.filesRead {
		t.Errorf("e.filesRead = true after load with explicit \"filesRead\": false; defaults-merge must NOT clobber explicit user values")
	}
}

// TestSettingsMigration_FilesReadSchemaVersionRewrite verifies that loading a
// v3.2 settings.json (schemaVersion=2) triggers the needsUpgradeWrite path so
// the on-disk file is rewritten with schemaVersion=3 (CurrentSchemaVersion).
// Mirrors the v3.1→v3.2 migration test pattern.
func TestSettingsMigration_FilesReadSchemaVersionRewrite(t *testing.T) {
	dir := t.TempDir()
	copyV32FixtureToTempDir(t, dir)

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir)

	raw, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("re-read settings.json: %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal rewritten settings: %v", err)
	}
	if s.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("on-disk schemaVersion = %d, want %d (v3.2→v3.3 upgrade re-write should fire)", s.SchemaVersion, CurrentSchemaVersion)
	}
}

// TestSettingsMigration_FilesWriteDefaultsFalse verifies that a v3.2
// settings.json (no filesWrite key) loads with FilesWrite = false (opt-in for
// all). This is the INVERSION of TestSettingsMigration_FilesReadDefaultsTrue:
// files.write is an explicit opt-in, so the absence of the key must yield false,
// not true. T-124-06 mitigation.
func TestSettingsMigration_FilesWriteDefaultsFalse(t *testing.T) {
	dir := t.TempDir()
	copyV32FixtureToTempDir(t, dir)

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir)

	// FilesWrite must default to false on a v3 file with no filesWrite key.
	// Zero-value false is the correct opt-in default (CAP-08); do NOT pre-populate
	// a true default as filesRead does.
	if e.filesWriteDefault {
		t.Errorf("e.filesWriteDefault = true after loading v3 fixture with no filesWrite key; want false (opt-in for all)")
	}
}

// TestSettingsMigration_FilesWriteSchemaVersionRewrite verifies that loading a
// v3.2 settings.json (schemaVersion=2) triggers the needsUpgradeWrite path so
// the on-disk file is rewritten with schemaVersion=4 (CurrentSchemaVersion after
// Phase 124 bump). Mirrors TestSettingsMigration_FilesReadSchemaVersionRewrite
// with the updated target version.
func TestSettingsMigration_FilesWriteSchemaVersionRewrite(t *testing.T) {
	dir := t.TempDir()
	copyV32FixtureToTempDir(t, dir)

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir)

	raw, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("re-read settings.json: %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal rewritten settings: %v", err)
	}
	if s.SchemaVersion != 4 {
		t.Errorf("on-disk schemaVersion = %d, want 4 (v3→v4 upgrade re-write should fire after Phase 124 bump)", s.SchemaVersion)
	}
}
