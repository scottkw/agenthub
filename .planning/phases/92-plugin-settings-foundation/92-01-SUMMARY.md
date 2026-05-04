---
phase: 92
plan: 01
subsystem: daemon
tags:
  - daemon
  - settings
  - migration
  - go
  - http
requires: []
provides:
  - PluginSettings struct (8 bool fields, camelCase JSON tags)
  - defaultPluginSettings() — 7 ON, Progress OFF (UI-SPEC default table)
  - CurrentSchemaVersion = 2
  - daemonSettings extended with Plugins + SchemaVersion
  - SessionEngine.{Get,Set}PluginSettings methods
  - GET + PATCH /settings/plugins routes
  - DaemonClient.{Get,Set}PluginSettings methods
  - tests/fixtures/settings_v3.1.json (realistic v3.1 shape)
  - v3.1 → v3.2 defaults-merge migration (single load-bearing CI gate)
affects:
  - internal/daemon/engine.go (loadSettingsFromDisk, saveSettingsToDisk, daemonSettings, SessionEngine)
  - internal/daemon/api.go (route registration + handlers)
  - internal/daemon/client.go (RPC methods)
tech-stack:
  added: []
  patterns:
    - "defaults-merge initializer (Pitfall #14 mitigation): pre-populate struct literal before json.Unmarshal so missing JSON keys inherit non-zero defaults"
    - "defense-in-depth HTTP body parsing: MaxBytesReader + DisallowUnknownFields"
    - "handler-level test pattern: httptest.NewRecorder against api.mux (avoids DaemonClient's hard-coded Unix-socket transport)"
key-files:
  created:
    - internal/daemon/plugin_settings.go
    - internal/daemon/plugin_settings_test.go
    - internal/daemon/engine_plugins_test.go
    - internal/daemon/engine_migration_test.go
    - tests/fixtures/settings_v3.1.json
  modified:
    - internal/daemon/engine.go
    - internal/daemon/api.go
    - internal/daemon/client.go
key-decisions:
  - "SchemaVersion is NOT pre-populated in the defaults-merge literal: leaving it at 0 on a v3.1 file is what trips needsUpgradeWrite, which is the entire point of the migration. Plan-prescribed literal would have made the migration silently no-op."
  - "HTTP round-trip is exercised at the handler level (httptest.NewRecorder + api.mux.ServeHTTP), NOT via DaemonClient against an httptest.NewServer. DaemonClient's transport hard-codes a Unix-socket dialer that is incompatible with TCP-based test servers. End-to-end Unix-socket transport is covered by production startup."
  - "PATCH (not POST/PUT) for /settings/plugins for consistency with the existing /settings/start-minimized and /settings/auto-close-session routes. The body is a full PluginSettings struct (full-replace semantic) — the PATCH choice is convention, not partial-update semantics."
requirements-completed:
  - PLUG-01
  - PLUG-02
duration: "16 min"
completed: "2026-05-04"
---

# Phase 92 Plan 01: Plugin Settings Foundation Summary

Daemon source-of-truth for v3.2 plugin settings — `PluginSettings` Go struct, defaults-merge load (Pitfall #14 mitigation), engine Get/Set methods, HTTP PATCH/GET routes with V5 input validation, DaemonClient methods, and a fixture-based v3.1→v3.2 migration test that is the load-bearing CI gate for ROADMAP SC-1.

**Duration:** 16 min · **Started:** 2026-05-04T14:56:33Z · **Completed:** 2026-05-04T15:12:40Z · **Tasks:** 3/3 · **Files:** 8 (5 created, 3 modified) · **Commits:** 3

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | RED — fixture + test scaffolds | `8ba46d3` | `tests/fixtures/settings_v3.1.json`, `internal/daemon/plugin_settings_test.go`, `internal/daemon/engine_plugins_test.go`, `internal/daemon/engine_migration_test.go` |
| 2 | GREEN core — PluginSettings + defaults-merge load + engine Get/Set | `6956ce7` | `internal/daemon/plugin_settings.go` (new), `internal/daemon/engine.go` |
| 3 | HTTP routes + DaemonClient + V5 input validation | `30b667c` | `internal/daemon/api.go`, `internal/daemon/client.go`, `internal/daemon/engine_plugins_test.go` |

## File Roles

| File | Role |
|------|------|
| `internal/daemon/plugin_settings.go` | `PluginSettings` struct (8 bool fields, camelCase JSON), `defaultPluginSettings()` constructor (7 ON / Progress OFF per UI-SPEC), `CurrentSchemaVersion = 2`. Single source of truth for plugin defaults. |
| `internal/daemon/engine.go` | `daemonSettings` extended with `Plugins` + `SchemaVersion` (no `omitempty`). `loadSettingsFromDisk` performs defaults-merge: pre-populates `Plugins` (NOT SchemaVersion) before Unmarshal so v3.1 files inherit defaults while `schemaVersion=0` still trips `needsUpgradeWrite`. `saveSettingsToDisk` extended; lock contract unchanged. `SessionEngine.{Get,Set}PluginSettings` mirror `Set/GetStartMinimized`. |
| `internal/daemon/api.go` | `GET /settings/plugins` returns the struct directly. `PATCH /settings/plugins` applies `http.MaxBytesReader(8 KiB)` (T-92-02 mitigation) + `dec.DisallowUnknownFields()` (T-92-03 mitigation), responds 204. |
| `internal/daemon/client.go` | `DaemonClient.GetPluginSettings` / `SetPluginSettings`. `Set` sends the full struct as the PATCH body (full-replace). |
| `tests/fixtures/settings_v3.1.json` | Realistic v3.1 shape: `cliPaths` + `startMinimized` + `autoCloseSession`; **no** `plugins`, **no** `schemaVersion`. Drives the migration test. |
| `internal/daemon/plugin_settings_test.go` | `TestDefaultPluginSettings` — asserts each of the 8 default values individually so a regression names the broken field. |
| `internal/daemon/engine_plugins_test.go` | `TestSetPluginSettingsRoundTrip` (engine-level + reload-engine round-trip), `TestPluginSettingsHTTPRoundTrip` (handler-level PATCH→GET via `httptest.NewRecorder`), `TestSetPluginSettingsRejectsUnknownFields` (400 on schema poisoning), `TestSetPluginSettingsRejectsOversizedBody` (400 on >8 KiB). |
| `internal/daemon/engine_migration_test.go` | `TestSettingsMigrationV3_1ToV3_2` — the load-bearing CI gate per ROADMAP SC-1: defaults populated in memory, `cliPaths` preserved, on-disk file rewritten with `plugins` block + `schemaVersion: 2`. `TestSettingsMigrationIdempotent` — mtime-equality check after a second consecutive load proves the migration write fires exactly once. |

## Defaults-Merge Load Pattern

Anchor: `92-PATTERNS.md` Shared Pattern A (defaults-merge initializer).

```go
// loadSettingsFromDisk
data, err := os.ReadFile(settingsPath(dir))
if err != nil {
    return // first run — no file
}
// Pre-populate ONLY Plugins. Leave SchemaVersion = 0 so a v3.1 file
// (no schemaVersion key) still trips needsUpgradeWrite below.
s := daemonSettings{
    Plugins: defaultPluginSettings(),
}
if json.Unmarshal(data, &s) != nil {
    return
}
// ... existing CLIPaths shell-mismatch cleanup and field assignments ...
e.pluginSettings = s.Plugins
needsUpgradeWrite := s.SchemaVersion < CurrentSchemaVersion
if dirty || needsUpgradeWrite {
    e.saveSettingsToDisk()
}
```

Why: `json.Unmarshal` leaves keys absent from the JSON untouched, so a v3.1 file with no `plugins` block inherits the defaults pre-populated in the struct literal. This defeats Pitfall #14 (zero-value plugin defaults silently disabling all addons on v3.1 upgrade).

## PluginSettings JSON Shape (for Plans 02 + 03 to consume)

```json
{
  "webgl": true,
  "unicode11": true,
  "search": true,
  "webLinks": true,
  "image": true,
  "serialize": true,
  "clipboard": true,
  "progress": false
}
```

camelCase tags match daemonSettings vocabulary (`cliPaths`, `startMinimized`). `webLinks` is the single multi-word JSON key. NO `omitempty` — every field round-trips even when at its zero value, and the parent `plugins` key always serializes (Pitfall #14).

HTTP shape:

| Method | Path | Request | Response |
|--------|------|---------|----------|
| `GET`  | `/settings/plugins` | — | 200, JSON body = `PluginSettings` |
| `PATCH` | `/settings/plugins` | JSON body = `PluginSettings` (≤ 8 KiB, no unknown fields) | 204 No Content |

## Verification

- `go test ./internal/daemon/... -run "TestDefaultPluginSettings|TestSetPluginSettingsRoundTrip|TestSettingsMigrationV3_1ToV3_2|TestSettingsMigrationIdempotent" -count=1` — **PASS**
- `go test ./internal/daemon/... -run "TestPluginSettingsHTTPRoundTrip|TestSetPluginSettingsRejectsUnknownFields|TestSetPluginSettingsRejectsOversizedBody" -count=1` — **PASS**
- `go test ./internal/daemon/... -count=1` — **PASS** (full daemon suite, 6.585 s; no v3.1 regression including TestSettingsPersistence, TestStartMinimizedPersistence, TestLoadSettingsDropsStaleShellPaths)
- `go vet ./internal/daemon/...` — **PASS**
- `go test ./...` — daemon and all other in-scope packages PASS. The pre-existing `security-review/` package fails to set up (mixed-package directory `relay`+`webserver` test files; out of scope for this plan).
- `golangci-lint run` — **SKIPPED** (binary not installed in this environment; `go vet` substitutes as baseline static analysis).

## Tasks 2 + 3 Did Not Regress v3.1 Daemon Tests

The full daemon suite (`go test ./internal/daemon/... -count=1`) passes after every per-task commit. The pre-existing v3.1 tests preserved:
- `TestSettingsPersistence` (cliPaths round-trip)
- `TestTailscalePathPersistence`
- `TestSettingsLoadMissingFile`
- `TestStartMinimizedPersistence`
- `TestStartMinimizedWithoutCLIPaths`
- `TestLoadSettingsDropsStaleShellPaths` (CLIPaths cleanup logic preserved verbatim per W1 guardrail)
- `TestSettingsFilePermissions` (0600 perms unchanged)

## Deviations from Plan

### [Rule 1 — Bug] Plan literal pre-populated SchemaVersion: CurrentSchemaVersion

- **Found during:** Task 2 (after running `go test`)
- **Issue:** The plan's prescribed `loadSettingsFromDisk` defaults-merge literal was:
  ```go
  s := daemonSettings{
      SchemaVersion: CurrentSchemaVersion,
      Plugins:       defaultPluginSettings(),
  }
  ```
  With `SchemaVersion` pre-populated to `CurrentSchemaVersion` (= 2), a v3.1 file (which omits `schemaVersion`) leaves `s.SchemaVersion` at 2 after Unmarshal — so `needsUpgradeWrite := s.SchemaVersion < CurrentSchemaVersion` evaluates to `2 < 2 == false`. The migration write would never fire. `TestSettingsMigrationV3_1ToV3_2` failed exactly this way: on-disk `schemaVersion = 0` and `plugins.webgl = false` (file unchanged after load).
- **Fix:** Pre-populate ONLY `Plugins`. Leave `SchemaVersion` at its zero value so a v3.1 file (no `schemaVersion` key) keeps `s.SchemaVersion = 0` after Unmarshal, which trips `needsUpgradeWrite` and fires the upgrade re-write exactly once.
- **Files modified:** `internal/daemon/engine.go`
- **Verification:** All four Wave 0 tests pass post-fix; the idempotent test confirms the second load is a no-op (mtime unchanged).
- **Commit:** `6956ce7` (rolled into the GREEN commit; the bug-fix and the original implementation were inseparable since the planned literal would never compile a passing test).

### Total deviations

**1 auto-fixed** (Rule 1 — Bug). **Impact:** None to the user-facing surface; the fix is internal to the load function and is required for the migration to function. The plan's narrative around the migration (in the action block, the verification, and the threat model) is unchanged — only one line of the prescribed Go literal needed correction.

## Authentication Gates

None — this plan is entirely local (Go tests, no network calls, no CLI tools requiring login).

## Threat Flags

None — no new attack surface beyond the threat model in `92-01-PLAN.md`.

## Self-Check: PASSED

- File checks (existence):
  - `internal/daemon/plugin_settings.go` — FOUND
  - `internal/daemon/plugin_settings_test.go` — FOUND
  - `internal/daemon/engine_plugins_test.go` — FOUND
  - `internal/daemon/engine_migration_test.go` — FOUND
  - `tests/fixtures/settings_v3.1.json` — FOUND
- Commit checks (`git log --oneline --all | grep`):
  - `8ba46d3` — FOUND
  - `6956ce7` — FOUND
  - `30b667c` — FOUND
- Verification commands re-run after writing this SUMMARY:
  - Wave 0 four-test gate — PASS
  - Three HTTP-level tests — PASS
  - Full daemon suite — PASS

## Ready for 92-02

Plan 02 (Wave 2) consumes:
- The HTTP shape `GET /settings/plugins` / `PATCH /settings/plugins` (camelCase JSON, full struct body).
- `DaemonClient.{Get,Set}PluginSettings` for the Wails GUI to read/write the persisted state.
- The 8 plugin defaults (used to seed the toggle UI before the daemon round-trip completes).
