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
