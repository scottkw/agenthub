package daemon

import "testing"

// Phase 95 Plan 95-01 Task 2 — Wave 0 RED scaffolds for the daemon
// WebLinksConfig sub-key RPC + migration. Plan 95-05 implements both.
//
// Mirrors Phase 94 search_config_test.go pattern: t.Skip with an explicit
// "Pending until Plan 95-XX implements ..." marker so each appears as a
// visible SKIP row in CI (not silently absent).
//
// Why test files exist now (Wave 0) when the implementation is a Wave 1+
// concern: each downstream Phase 95 plan needs at least one named verify
// target waiting to flip GREEN. CI grep for these test names is part of
// the per-plan acceptance criteria — the file MUST exist on disk so the
// grep does not silently pass on a typo'd test name.

// TestSetWebLinksConfigPreservesSiblings — Plan 95-05 implements.
// Mirrors Phase 94 search_config_test.go TestSetSearchConfig: write a
// WebLinksConfig with non-default values; assert all OTHER PluginSettings
// fields (WebGL/Unicode11/Search/SearchConfig/WebLinks/Image/Serialize/
// Clipboard/Progress) are preserved; assert listener fires once.
func TestSetWebLinksConfigPreservesSiblings(t *testing.T) {
	t.Skip("Pending until Plan 95-05 implements engine.SetWebLinksConfig sub-key writer (95-VALIDATION row 95-06-01).")
}

// TestPluginSettingsMigration_WebLinksConfig — Plan 95-05 implements.
// Mirror Phase 92 migration test: load a v3.2-without-WebLinksConfig
// settings.json fixture; defaults-merge should populate WebLinksConfig
// with platform/all-confirm-on values.
func TestPluginSettingsMigration_WebLinksConfig(t *testing.T) {
	t.Skip("Pending until Plan 95-05 implements defaults-merge for WebLinksConfig (95-RESEARCH §\"Pattern 3\").")
}
