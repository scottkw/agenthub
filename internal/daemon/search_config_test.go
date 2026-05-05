package daemon

import "testing"

// Phase 94 Wave 0 RED scaffold — Plan 94-02 implements (SRC-02 persistence).
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md row 02-daemon wave 1.
// RESEARCH §"Persistence path reuses Phase 92/93 unchanged".

func TestSearchConfig_DefaultsZero(t *testing.T) {
	t.Skip("RED scaffold — Plan 94-02 implements SearchConfig{Regex,CaseSensitive,WholeWord bool} " +
		"nested in PluginSettings with all-false defaults. " +
		"See 94-VALIDATION.md row 02-daemon wave 1.")
}

func TestSearchConfig_RoundTripJSON(t *testing.T) {
	t.Skip("RED scaffold — Plan 94-02 implements SearchConfig JSON round-trip via " +
		"json.Marshal/Unmarshal of daemon.PluginSettings. RESEARCH §\"Code Examples\" Example 3.")
}

func TestPluginSettings_DefaultsMerge_SearchConfig(t *testing.T) {
	t.Skip("RED scaffold — Plan 94-02 verifies that Phase 93 fixture (no searchConfig field) " +
		"populates SearchConfig defaults via the existing daemonSettings defaults-merge pattern " +
		"(internal/daemon/engine_migration_test.go:41-79 analog).")
}
