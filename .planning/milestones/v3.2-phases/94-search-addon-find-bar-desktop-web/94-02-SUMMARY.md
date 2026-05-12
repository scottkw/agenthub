---
phase: 94
plan: 02
status: complete
completed: 2026-05-05
requirements: [SRC-02]
---

# Plan 94-02 Summary — Daemon SearchConfig Persistence

## What Was Built

SRC-02 data-layer foundation: nested `SearchConfig {Regex, CaseSensitive, WholeWord}` struct in `daemon.PluginSettings`, end-to-end JSON round-trip from Go through Wails models.ts to the SSE broadcast pipeline.

### Commits

1. `feat(94-02): add SearchConfig sub-struct + defaults to daemon.PluginSettings` — `ebe3873`
2. `feat(94-02): mirror SearchConfig to TS models + SSE round-trip test` — `86b7a44`

### Go (Task 1)

- `internal/daemon/plugin_settings.go` — added `SearchConfig` struct (camelCase JSON tags, no omitempty); added `SearchConfig SearchConfig \`json:"searchConfig"\`` to `PluginSettings`; updated `defaultPluginSettings()` to construct zero-value SearchConfig.
- `internal/daemon/plugin_settings_test.go` — extended `TestDefaultPluginSettings` with 3 SearchConfig assertions.
- `internal/daemon/search_config_test.go` — replaced Wave 0 t.Skip RED with 3 real tests (defaults zero, JSON round-trip, defaults-merge).
- `internal/daemon/engine_migration_test.go` — added explicit assertion that Phase 93 fixture (no searchConfig key) loads with SearchConfig zero defaults.
- `internal/daemon/engine.go` — gofmt re-aligned field tags (no semantic change).

### TS + SSE (Task 2)

- `frontend/src/wailsjs/go/models.ts` — new `daemon.SearchConfig` class; `daemon.PluginSettings.searchConfig: SearchConfig` field with inline `new SearchConfig(...)` conversion in constructor (chose inline over a `convertValues` helper member because the helper would surface as `keyof PluginSettings` and break `PluginsSection.tsx` toggle iteration).
- `frontend/src/components/PluginsSection.tsx` — narrowed `toggle`/`renderRow` key type to `PluginBooleanKey` (mapped type filtering only boolean fields), and now constructs fresh `PluginSettings` instances on toggle to preserve class identity. Promoted `daemon` import from type-only to value+type so the constructor is reachable.
- `frontend/src/__tests__/App.plugin-event.test.tsx` — 4 new tests (`SearchConfig` constructs from object, `PluginSettings` preserves nested instance, JSON round-trip, App.tsx + TerminalPanel prop-drill compatibility).
- `internal/webserver/plugin_settings_search_sse_test.go` — Wave 0 RED scaffold turned GREEN. Real `/api/plugin-config/stream` integration test (capability-gated, like Phase 93's existing tests) asserts the SSE frame body contains `"searchConfig":{"regex":true,"caseSensitive":false,"wholeWord":true}`.

## Tests Run / Outcomes

| Test | Status |
|------|--------|
| `go test ./internal/daemon/ -run "TestDefaultPluginSettings\|TestSearchConfig\|TestPluginSettings_DefaultsMerge\|TestSettingsMigration" -count=1` | ✅ PASS |
| `go test ./internal/webserver/ -run TestPluginSettingsSSE_Search -count=1` | ✅ PASS |
| `pnpm exec tsc --noEmit` | ✅ exit 0 (no errors) |
| `pnpm exec vitest --run src/__tests__/App.plugin-event.test.tsx` | ✅ 12/12 pass (8 existing + 4 new) |
| `pnpm exec vitest --run src/components/__tests__/PluginsSection` | ✅ 13/13 pass |
| `gofmt -d ./internal/daemon/...` | ✅ clean |

**Pre-existing test-env artifact (not introduced here):** 20 Sidebar tests fail with `localStorage.getItem is not a function` — confirmed via `git stash` round-trip on `a904be5` (Plan 94-01 SUMMARY commit) before my changes; failures are independent of Phase 94. Per the user's "verify test-env before declaring failure" guidance, these are jsdom environment failures unrelated to this plan and can be addressed separately.

**FindBar RED scaffolds (intentional):** 12 RED tests still fail by design (Plan 94-01 Task 2 Wave 0 contract). They will turn GREEN as Plans 94-03/94-04/94-05 implement.

## Wave 0 Scaffolds Turned GREEN

| Scaffold | Plan | Status |
|----------|------|--------|
| `TestSearchConfig_DefaultsZero` | 94-02 | ✅ GREEN |
| `TestSearchConfig_RoundTripJSON` | 94-02 | ✅ GREEN |
| `TestPluginSettings_DefaultsMerge_SearchConfig` | 94-02 | ✅ GREEN |
| `TestPluginSettingsSSE_Search` | 94-02 | ✅ GREEN |

4/13 RED scaffolds now GREEN; 9 remain (1 deferred to Plan 94-04, the rest to 94-03/94-05).

## Threat Model Status

| Threat | Severity | Status |
|--------|----------|--------|
| T-94-02 (settings tampering — malformed SearchConfig JSON crashes daemon) | MEDIUM | ✅ MITIGATED — defaults-merge load path populates SearchConfig zero-value before unmarshal; verified by `TestPluginSettings_DefaultsMerge_SearchConfig` and `TestSettingsMigrationV3_1ToV3_2` |

## Key Files Modified

- `internal/daemon/plugin_settings.go`
- `internal/daemon/plugin_settings_test.go`
- `internal/daemon/search_config_test.go`
- `internal/daemon/engine_migration_test.go`
- `internal/daemon/engine.go` (gofmt only)
- `internal/webserver/plugin_settings_search_sse_test.go`
- `frontend/src/wailsjs/go/models.ts`
- `frontend/src/components/PluginsSection.tsx`
- `frontend/src/__tests__/App.plugin-event.test.tsx`

## Notes / Deviations

- **Inline conversion vs convertValues helper:** Plan called for adding a `convertValues` private method on `PluginSettings` matching the standard Wails-generated pattern. Switched to inline `source["searchConfig"] ? new SearchConfig(source["searchConfig"]) : new SearchConfig()` because TypeScript `private` modifiers don't actually hide methods from `keyof` — the helper would surface in `keyof PluginSettings` and break `PluginsSection.tsx` toggle iteration. The functional contract (Wails-style nested deserialization) is identical.
- **PluginsSection.tsx adjustments:** required by the new nested field. Type-only changes — no runtime behavior shift.

## Self-Check

- [x] All tasks executed
- [x] Each task committed atomically (Task 1 = `ebe3873`, Task 2 = `86b7a44`)
- [x] SUMMARY.md created
- [x] No modifications to STATE.md or ROADMAP.md (orchestrator owns those)
- [x] T-94-02 (MEDIUM) mitigated via defaults-merge
- [x] 4 Wave 0 RED scaffolds turned GREEN
- [x] tsc --noEmit clean; no test regressions vs main pre-Phase-94 state
