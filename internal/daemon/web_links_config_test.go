package daemon

import (
	"encoding/json"
	"os"
	"testing"
)

// Phase 95 Plan 95-05 — LNK-05 / LNK-06 daemon WebLinksConfig sub-key
// writer + defaults-merge migration tests. Mirrors Phase 94 Plan 94-07
// search_config_test.go (TestSetSearchConfig) verbatim with WebLinksConfig
// substituted in. The Wave 0 Plan 95-01 t.Skip stubs are replaced here
// with real implementations.

// TestSetWebLinksConfigPreservesSiblings — Phase 95 LNK-05 / LNK-06.
//  1. WebLinksConfig sub-key updates round-trip through GetPluginSettings.
//  2. All other PluginSettings fields are PRESERVED (T-95-05-01 mitigation).
//  3. The pluginSettingsListener fires exactly once (web parity SSE preserved).
//  4. The change persists to disk (reload reflects the new WebLinksConfig).
//
// Harness mirrors TestSetSearchConfig in search_config_test.go.
func TestSetWebLinksConfigPreservesSiblings(t *testing.T) {
	dir := t.TempDir()

	// Baseline PluginSettings: every non-web-links field set to a non-default
	// value so any stomp by SetWebLinksConfig is detectable. WebLinksConfig
	// itself starts at platform/all-confirm-on (the defaults set in Plan 95-01)
	// so we can detect the post-call diff.
	baseline := PluginSettings{
		WebGL:        false,
		Unicode11:    false,
		Search:       true,
		SearchConfig: SearchConfig{Regex: true, CaseSensitive: true, WholeWord: false},
		WebLinks:     false,
		WebLinksConfig: WebLinksConfig{
			Modifier:         "platform",
			ConfirmOSC8:      true,
			ConfirmIDN:       true,
			ConfirmTyposquat: true,
		},
		Image:     false,
		Serialize: false,
		Clipboard: false,
		Progress:  true,
	}

	e := &SessionEngine{
		configDir:      dir,
		cliPaths:       make(map[string]string),
		pluginSettings: baseline,
	}

	// Listener counter — SetWebLinksConfig must invoke it exactly once.
	var listenerCount int
	e.SetPluginSettingsListener(func() { listenerCount++ })

	// Apply the web-links-config sub-key change.
	newCfg := WebLinksConfig{
		Modifier:         "ctrl",
		ConfirmOSC8:      false,
		ConfirmIDN:       true,
		ConfirmTyposquat: false,
	}
	e.SetWebLinksConfig(newCfg)

	// Listener fired exactly once.
	if listenerCount != 1 {
		t.Errorf("listener invocations: got %d, want 1", listenerCount)
	}

	got := e.GetPluginSettings()

	// WebLinksConfig sub-key updated.
	if got.WebLinksConfig != newCfg {
		t.Errorf("WebLinksConfig: got %+v, want %+v", got.WebLinksConfig, newCfg)
	}

	// Every other field preserved (T-95-05-01 mitigation — no full-PluginSettings stomp).
	if got.WebGL != baseline.WebGL {
		t.Errorf("WebGL stomped: got %v, want %v", got.WebGL, baseline.WebGL)
	}
	if got.Unicode11 != baseline.Unicode11 {
		t.Errorf("Unicode11 stomped: got %v, want %v", got.Unicode11, baseline.Unicode11)
	}
	if got.Search != baseline.Search {
		t.Errorf("Search stomped: got %v, want %v", got.Search, baseline.Search)
	}
	if got.SearchConfig != baseline.SearchConfig {
		t.Errorf("SearchConfig stomped: got %+v, want %+v", got.SearchConfig, baseline.SearchConfig)
	}
	if got.WebLinks != baseline.WebLinks {
		t.Errorf("WebLinks (boolean) stomped: got %v, want %v", got.WebLinks, baseline.WebLinks)
	}
	if got.Image != baseline.Image {
		t.Errorf("Image stomped: got %v, want %v", got.Image, baseline.Image)
	}
	if got.Serialize != baseline.Serialize {
		t.Errorf("Serialize stomped: got %v, want %v", got.Serialize, baseline.Serialize)
	}
	if got.Clipboard != baseline.Clipboard {
		t.Errorf("Clipboard stomped: got %v, want %v", got.Clipboard, baseline.Clipboard)
	}
	if got.Progress != baseline.Progress {
		t.Errorf("Progress stomped: got %v, want %v", got.Progress, baseline.Progress)
	}

	// Persistence: reload via a fresh engine pointed at the same configDir.
	e2 := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e2.loadSettingsFromDisk(dir)

	got2 := e2.GetPluginSettings()
	if got2.WebLinksConfig != newCfg {
		t.Errorf("reloaded WebLinksConfig: got %+v, want %+v", got2.WebLinksConfig, newCfg)
	}
	// Spot-check that the non-web-links fields also persisted (defaults-merge
	// could otherwise mask a stomp at the on-disk JSON layer).
	if got2.Search != baseline.Search {
		t.Errorf("reloaded Search: got %v, want %v", got2.Search, baseline.Search)
	}
	if got2.SearchConfig != baseline.SearchConfig {
		t.Errorf("reloaded SearchConfig: got %+v, want %+v", got2.SearchConfig, baseline.SearchConfig)
	}
	if got2.Progress != baseline.Progress {
		t.Errorf("reloaded Progress: got %v, want %v", got2.Progress, baseline.Progress)
	}
}

// TestPluginSettingsMigration_WebLinksConfig — Phase 95 LNK-02 default
// + LNK-05 defaults-merge. A v3.2 settings.json missing the webLinksConfig
// sub-object MUST defaults-merge to platform/all-confirm-on (the defaults
// set in Plan 95-01).
//
// Mitigates the Phase 92 Pitfall #14 surface: a malformed or missing
// webLinksConfig key in settings.json must not crash the daemon — the
// defaults-merge ensures the WebLinksConfig populated default is present
// before json.Unmarshal runs.
//
// Harness mirrors the existing TestPluginSettings_DefaultsMerge_SearchConfig
// in search_config_test.go: write a Phase 94-shape settings.json (no
// webLinksConfig key) to disk, load via loadSettingsFromDisk, assert the
// defaults populated.
func TestPluginSettingsMigration_WebLinksConfig(t *testing.T) {
	dir := t.TempDir()

	// Phase 94 settings.json shape — has searchConfig but no webLinksConfig key.
	// Wrap in the daemonSettings shape (plugins block) so loadSettingsFromDisk
	// observes it as a real v3.2-with-search-but-no-web-links-config file.
	saved := []byte(`{"schemaVersion":2,"plugins":{"webgl":true,"unicode11":true,"search":true,"searchConfig":{"regex":false,"caseSensitive":false,"wholeWord":false},"webLinks":true,"image":true,"serialize":true,"clipboard":true,"progress":false}}`)
	if err := os.WriteFile(settingsPath(dir), saved, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	e := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e.loadSettingsFromDisk(dir)

	defaults := defaultPluginSettings()
	got := e.GetPluginSettings()

	if got.WebLinksConfig != defaults.WebLinksConfig {
		t.Errorf("after migration, WebLinksConfig = %+v; want %+v",
			got.WebLinksConfig, defaults.WebLinksConfig)
	}
	if got.WebLinksConfig.Modifier != "platform" {
		t.Errorf("Modifier = %q; want \"platform\"", got.WebLinksConfig.Modifier)
	}
	if !got.WebLinksConfig.ConfirmOSC8 {
		t.Error("ConfirmOSC8 = false; want true (security-first default)")
	}
	if !got.WebLinksConfig.ConfirmIDN {
		t.Error("ConfirmIDN = false; want true (security-first default)")
	}
	if !got.WebLinksConfig.ConfirmTyposquat {
		t.Error("ConfirmTyposquat = false; want true (security-first default)")
	}

	// Sanity: the existing keys present in the fixture survived too.
	if !got.WebLinks {
		t.Error("WebLinks (boolean) = false; want true (fixture had it true)")
	}
	if !got.Search {
		t.Error("Search = false; want true (fixture had it true)")
	}

	// Belt-and-suspenders: the in-memory PluginSettings should serialize back
	// out with the populated webLinksConfig key (Pitfall #14 — a future load
	// observes the merged state, not the upstream gap).
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var rt PluginSettings
	if err := json.Unmarshal(raw, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rt.WebLinksConfig != defaults.WebLinksConfig {
		t.Errorf("post-marshal round-trip WebLinksConfig = %+v; want %+v",
			rt.WebLinksConfig, defaults.WebLinksConfig)
	}
}
