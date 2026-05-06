package daemon

import (
	"encoding/json"
	"testing"
)

// Phase 94 Plan 94-02 — SRC-02 daemon SearchConfig persistence tests.
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md row 02-daemon wave 1.
// RESEARCH §"Pattern 3 — SearchConfig Persistence".

// TestSearchConfig_DefaultsZero — sanity guard against future re-ordering
// that flips a default. The SearchConfig zero-value is the contract: all
// three booleans must be false on a brand-new struct.
func TestSearchConfig_DefaultsZero(t *testing.T) {
	var c SearchConfig
	if c.Regex {
		t.Error("expected SearchConfig{}.Regex=false (Phase 94 SRC-02 invariant)")
	}
	if c.CaseSensitive {
		t.Error("expected SearchConfig{}.CaseSensitive=false (Phase 94 SRC-02 invariant)")
	}
	if c.WholeWord {
		t.Error("expected SearchConfig{}.WholeWord=false (Phase 94 SRC-02 invariant)")
	}
}

// TestSearchConfig_RoundTripJSON — json.Marshal then json.Unmarshal preserves
// SearchConfig field values, and the marshaled JSON keys are camelCase.
func TestSearchConfig_RoundTripJSON(t *testing.T) {
	in := SearchConfig{Regex: true, CaseSensitive: false, WholeWord: true}

	body, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	want := `{"regex":true,"caseSensitive":false,"wholeWord":true}`
	if got := string(body); got != want {
		t.Errorf("Marshal: got %q, want %q", got, want)
	}

	var out SearchConfig
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round-trip: got %+v, want %+v", out, in)
	}
}

// TestPluginSettings_DefaultsMerge_SearchConfig — Phase 93 fixture (no
// searchConfig key in saved JSON) loads via the defaults-merge pattern
// (load defaults FIRST, then json.Unmarshal user-saved JSON over them)
// and leaves PluginSettings.SearchConfig at the zero-value defaults.
//
// Mitigates T-94-02: a malformed or missing SearchConfig key in
// settings.json must not crash the daemon — the defaults-merge ensures
// the SearchConfig zero-value is present before unmarshal runs.
func TestPluginSettings_DefaultsMerge_SearchConfig(t *testing.T) {
	// Phase 93 settings.json shape — no searchConfig key.
	saved := []byte(`{"webgl":true,"unicode11":true,"search":true,"webLinks":true,"image":true,"serialize":true,"clipboard":true,"progress":false}`)

	// Defaults-merge: load defaults first, then unmarshal user-saved JSON.
	ps := defaultPluginSettings()
	if err := json.Unmarshal(saved, &ps); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if ps.SearchConfig != (SearchConfig{}) {
		t.Errorf("expected SearchConfig zero-value defaults after Phase 93 fixture load, got %+v", ps.SearchConfig)
	}
}

// TestSetSearchConfig verifies the Phase 94-07 gap-closure RPC:
//  1. SearchConfig sub-key updates round-trip through GetPluginSettings.
//  2. All other PluginSettings fields are PRESERVED (WR-03 mitigation).
//  3. The pluginSettingsListener fires (SRC-05 web parity SSE preserved).
//  4. The change persists to disk (reload reflects the new SearchConfig).
//
// Harness mirrors TestSetPluginSettingsRoundTrip in engine_plugins_test.go.
func TestSetSearchConfig(t *testing.T) {
	dir := t.TempDir()

	// Baseline PluginSettings: every non-search field set to a non-default
	// value so any stomp by SetSearchConfig is detectable. Defaults are
	// 7-ON-1-OFF; baseline below is intentionally a mixed pattern with
	// SearchConfig at zero-value so we can detect the post-call diff.
	baseline := PluginSettings{
		WebGL:        false,
		Unicode11:    false,
		Search:       true,
		SearchConfig: SearchConfig{Regex: false, CaseSensitive: false, WholeWord: false},
		WebLinks:     false,
		Image:        false,
		Serialize:    false,
		Clipboard:    false,
		Progress:     true,
	}

	e := &SessionEngine{
		configDir:      dir,
		cliPaths:       make(map[string]string),
		pluginSettings: baseline,
	}

	// Listener counter — SetSearchConfig must invoke it exactly once.
	var listenerCount int
	e.SetPluginSettingsListener(func() { listenerCount++ })

	// Apply the search-config sub-key change.
	newCfg := SearchConfig{Regex: true, CaseSensitive: true, WholeWord: false}
	e.SetSearchConfig(newCfg)

	// Listener fired exactly once (SRC-05 web parity SSE preserved).
	if listenerCount != 1 {
		t.Errorf("listener invocations: got %d, want 1", listenerCount)
	}

	got := e.GetPluginSettings()

	// SearchConfig sub-key updated.
	if got.SearchConfig != newCfg {
		t.Errorf("SearchConfig: got %+v, want %+v", got.SearchConfig, newCfg)
	}

	// Every other field preserved (WR-03 mitigation — no full-PluginSettings stomp).
	if got.WebGL != baseline.WebGL {
		t.Errorf("WebGL stomped: got %v, want %v", got.WebGL, baseline.WebGL)
	}
	if got.Unicode11 != baseline.Unicode11 {
		t.Errorf("Unicode11 stomped: got %v, want %v", got.Unicode11, baseline.Unicode11)
	}
	if got.Search != baseline.Search {
		t.Errorf("Search stomped: got %v, want %v", got.Search, baseline.Search)
	}
	if got.WebLinks != baseline.WebLinks {
		t.Errorf("WebLinks stomped: got %v, want %v", got.WebLinks, baseline.WebLinks)
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
	if got2.SearchConfig != newCfg {
		t.Errorf("reloaded SearchConfig: got %+v, want %+v", got2.SearchConfig, newCfg)
	}
	// Spot-check that the non-search fields also persisted (defaults-merge
	// could otherwise mask a stomp at the on-disk JSON layer).
	if got2.Search != baseline.Search {
		t.Errorf("reloaded Search: got %v, want %v", got2.Search, baseline.Search)
	}
	if got2.Progress != baseline.Progress {
		t.Errorf("reloaded Progress: got %v, want %v", got2.Progress, baseline.Progress)
	}
}
