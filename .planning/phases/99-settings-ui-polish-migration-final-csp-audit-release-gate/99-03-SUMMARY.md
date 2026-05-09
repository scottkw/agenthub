---
phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate
plan: "03"
subsystem: testing
tags: [go, migration, settings, plugin-defaults, daemon]

# Dependency graph
requires:
  - phase: 92-plugin-settings-foundation
    provides: defaultPluginSettings() constructor and PluginSettings/SearchConfig/WebLinksConfig/ImageConfig types
  - phase: 94-search-addon-find-bar
    provides: Phase 94 SRC-02 SearchConfig struct + zero-value check at line 60 (now replaced)
  - phase: 95-web-links-addon-security-hardening
    provides: WebLinksConfig struct with Modifier/ConfirmOSC8/ConfirmIDN/ConfirmTyposquat fields
  - phase: 96-image-addon-csp-audit
    provides: ImageConfig struct with StorageLimit override (16 MB vs upstream 100 MB)
provides:
  - Per-field assertions in TestSettingsMigrationV3_1ToV3_2 for all 8 plugin booleans + 3 sub-configs
  - Forward-compatible wantSearch := SearchConfig{...} replacing the zero-value check
  - SC-3 satisfied per ROADMAP release-gate requirements
affects:
  - future v3.3 default changes (any defaultPluginSettings() change now produces named field failures)
  - 99-VERIFICATION (migration gate is now more diagnostic)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Per-field test assertions with named errors as diagnostic complement to struct-equality fast-fail
    - Forward-compatible explicit struct comparison (wantX := TypeName{...}) over zero-value check (Type{})

key-files:
  created: []
  modified:
    - internal/daemon/engine_migration_test.go

key-decisions:
  - "Kept existing struct-equality fast-fail sentinel (got != want) — per-field assertions are diagnostic, not replacement"
  - "Replaced SearchConfig zero-value check with explicit wantSearch := SearchConfig{Regex: false, ...} for forward-compatibility when v3.3 adds non-zero defaults"
  - "TestSettingsMigrationIdempotent left completely unchanged — it was already correct and is not in SC-3 scope"
  - "No new test files, no new fixtures — the existing settings_v3.1.json fixture covers the load-bearing migration scenario"

patterns-established:
  - "Pattern: Per-field boolean test assertions use if !got.Field { t.Errorf(\"...: Field = false, want true\") } — not table-driven"
  - "Pattern: Sub-config assertion uses wantX := TypeName{...} over (TypeName{}) zero-value check — explicitly names expected fields for forward-compat"

requirements-completed: []

# Metrics
duration: 9min
completed: 2026-05-08
---

# Phase 99 Plan 03: Migration Test SC-3 Expansion Summary

**Per-field assertions added to TestSettingsMigrationV3_1ToV3_2 for all 8 plugin booleans + 3 sub-configs, with zero-value SearchConfig check replaced by explicit wantSearch struct for v3.3 forward-compatibility**

## Performance

- **Duration:** 9 min
- **Started:** 2026-05-08T19:05:47Z
- **Completed:** 2026-05-08T19:15:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- Expanded `TestSettingsMigrationV3_1ToV3_2` with per-field assertions for all 8 plugin booleans: WebGL (want true), Unicode11 (want true), Search (want true), WebLinks (want true), Image (want true), Serialize (want true), Clipboard (want true), Progress (want false — default OFF in v3.2)
- Replaced the Phase 94 SRC-02 zero-value `SearchConfig{}` check at line 60 with explicit `wantSearch := SearchConfig{Regex: false, CaseSensitive: false, WholeWord: false}` — observationally identical today but forward-compatible when v3.3 adds a non-zero default
- Added explicit `WebLinksConfig` defaults assertion: `Modifier: "platform", ConfirmOSC8: true, ConfirmIDN: true, ConfirmTyposquat: true`
- Added explicit `ImageConfig` defaults assertion: `StorageLimit: 16` (the v3.2 override of upstream 100 MB to prevent tab OOM)
- Confirmed `TestSettingsMigrationIdempotent` (lines 138-173) untouched and still green — mtime equality across consecutive loads verified
- SC-3 satisfied: migration test now asserts all plugin defaults are populated with named field failures on any default-flip

## Task Commits

Each task was committed atomically:

1. **Task 1: Expand TestSettingsMigrationV3_1ToV3_2 per-field assertions for SC-3** - `5a7b9b5` (test)

**Plan metadata:** (committed with SUMMARY.md below)

## Files Created/Modified

- `internal/daemon/engine_migration_test.go` - Expanded TestSettingsMigrationV3_1ToV3_2 with 11 per-field assertions (8 plugin booleans + 3 sub-configs); removed zero-value SearchConfig check; idempotency test unchanged

## Decisions Made

- Kept the existing struct-equality fast-fail sentinel `if got != want { t.Errorf(...) }` — the per-field assertions are diagnostic companions, not replacements. If a future refactor accidentally changes struct equality, the fast-fail fires immediately; the per-field assertions then name which field changed.
- Replaced `got.SearchConfig != (SearchConfig{})` with `got.SearchConfig != wantSearch` (where `wantSearch` is explicitly constructed). The zero-value check is observationally identical today (`Regex/CaseSensitive/WholeWord` all default false), but becomes a silent false-pass if v3.3 ever adds a non-false default to SearchConfig.
- Did not create optional `tests/fixtures/settings_v3.2_partial.json` (mentioned in RESEARCH.md "Claude's Discretion") — the existing v3.1 fixture covers the load-bearing migration scenario; a partial v3.2 fixture would be redundant.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

`go test ./internal/daemon/... -run TestMigration -count=1` returned "no tests to run" because the Go test runner requires the `-run` pattern to match the full test function name. Used `-run TestSettingsMigration` instead, which correctly matched both `TestSettingsMigrationV3_1ToV3_2` and `TestSettingsMigrationIdempotent`. Both pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SC-3 migration test gate satisfied — any future `defaultPluginSettings()` change that alters a default now produces a clear, named failure in CI
- `TestSettingsMigrationIdempotent` unchanged and still green
- Phase 99 verification (`99-VERIFICATION.md`) can now count SC-3 as satisfied

## Self-Check

- [x] `internal/daemon/engine_migration_test.go` exists and modified
- [x] Commit `5a7b9b5` exists
- [x] `TestSettingsMigrationV3_1ToV3_2` PASS
- [x] `TestSettingsMigrationIdempotent` PASS
- [x] `go vet ./internal/daemon/...` PASS
- [x] `gofmt` produces no diff
- [x] All 13 acceptance criteria verified

## Self-Check: PASSED

---
*Phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate*
*Completed: 2026-05-08*
