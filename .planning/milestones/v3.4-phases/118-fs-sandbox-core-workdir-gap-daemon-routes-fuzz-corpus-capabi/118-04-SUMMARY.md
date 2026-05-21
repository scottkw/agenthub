---
phase: 118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi
plan: 04
subsystem: daemon
tags: [fs-sandbox, workdir-gap, settings-migration, schema-v3, files-read, defaults-merge]
requires:
  - internal/daemon/engine.go SessionEngine + CreateSession + KillSession + saveSettingsToDisk + loadSettingsFromDisk (pre-existing)
  - internal/daemon/plugin_settings.go (pre-existing CurrentSchemaVersion + defaults-merge pattern)
  - tests/fixtures/settings_v3.1.json (existing fixture pattern)
provides:
  - SessionEngine.sessionWorkDirs map populated at CreateSession with filepath.EvalSymlinks-resolved abs path (FS-02)
  - (*SessionEngine).GetSessionWorkDir(id) string (RLock; empty for unknown)
  - daemonSettings.FilesRead *bool with json tag filesRead,omitempty (FS-14)
  - CurrentSchemaVersion = 3 (bumped 2→3)
  - loadSettingsFromDisk defaults-merge pre-populates FilesRead=*true BEFORE Unmarshal (Pitfall 16 mitigation)
  - saveSettingsToDisk serializes FilesRead
  - tests/fixtures/settings_v3.2.json (no filesRead key)
affects:
  - Plan 05 (daemon routes): will build sandboxResolver closure on top of GetSessionWorkDir + read e.filesRead during token issuance
  - Plan 02 (FS sandbox core): consumers of NewSandbox feed it the resolved path from GetSessionWorkDir
tech_stack:
  added: []
  patterns:
    - "parallel-map pattern (RESEARCH.md Pattern 5): sessionWorkDirs lives alongside tabNames and sessionCLIs under e.mu"
    - "EvalSymlinks-once pattern (RESEARCH.md Pitfall 1): resolution happens ONCE at CreateSession, not per-request"
    - "defaults-merge constructor pattern (Pitfall #14 / #16 mitigation): pre-populate the struct literal BEFORE json.Unmarshal so missing keys inherit safe defaults"
key_files:
  created:
    - tests/fixtures/settings_v3.2.json
  modified:
    - internal/daemon/engine.go
    - internal/daemon/plugin_settings.go
    - internal/daemon/engine_migration_test.go
    - internal/daemon/engine_test.go
decisions:
  - "WorkDir resolved via filepath.EvalSymlinks ONCE at CreateSession; cached in sessionWorkDirs (no per-request resolution — RESEARCH.md Pitfall 1)"
  - "Resolution fallback to raw workDir on EvalSymlinks error so session creation never fails because of resolution; file browser will surface a 400 later if path is invalid"
  - "FilesRead is a daemon-wide setting (NOT a PluginSetting) per RESEARCH.md Assumption A6"
  - "FilesRead pointer type *bool (tri-state): nil = pre-v3.4 file pre-defaults-merge, *true = enabled, *false = explicitly disabled — defaults-merge converts the first to *true but ONLY for the legacy missing-key case (TestSettingsMigration_FilesReadExplicitFalse guards the reverse)"
  - "schemaVersion 2→3 bump triggers re-write via existing needsUpgradeWrite branch with zero modification to that comparison"
key_links:
  - from: "internal/daemon/engine.go CreateSession"
    to: "filepath.EvalSymlinks(workDir)"
    via: "one-time resolution AFTER $HOME substitution; result cached in sessionWorkDirs[id]"
  - from: "internal/daemon/engine.go loadSettingsFromDisk"
    to: "FilesRead: &tr defaults-merge"
    via: "pre-populate BEFORE json.Unmarshal so missing keys land *true"
  - from: "Plan 05 (next)"
    to: "engine.GetSessionWorkDir(id) + e.filesRead"
    via: "sandboxResolver closure + token-issuance Perms gating"
metrics:
  duration: 4m54s
  completed: 2026-05-20T16:37:47Z
  commits: 4
  tasks_completed: 2
  tasks_total: 2
---

# Phase 118 Plan 04: WorkDir Gap + Schema v3 Settings Migration Summary

Closed the WorkDir gap by caching a `filepath.EvalSymlinks`-resolved absolute path in a new `SessionEngine.sessionWorkDirs` map and exposed it via `GetSessionWorkDir(id) string`. Bumped `CurrentSchemaVersion` 2→3 and added a daemon-wide `FilesRead *bool` field to `daemonSettings` with the proven defaults-merge constructor pattern so v3.2 settings.json files upgrading to v3.4 land with `filesRead = true` (Pitfall 16 mitigation). Two atomic TDD task pairs (RED → GREEN), four commits, zero deviations.

## Tasks Completed

### Task 1: sessionWorkDirs map + GetSessionWorkDir (FS-02)

- **RED** (commit `bda6030`): five failing tests — Populated, ResolvesSymlink, ClearedOnKill, EmptyForUnknown, FallbackOnEvalSymlinksError — all using `spyBackend` (no real PTY).
- **GREEN** (commit `d0dd47e`): added `sessionWorkDirs map[string]string` field, initialized it in `NewSessionEngine`, populated at `CreateSession` via `filepath.EvalSymlinks(workDir)` with raw-workDir fallback on error, cleared at `KillSession`, exposed `GetSessionWorkDir(id) string` with `e.mu.RLock()`.

Verification: `go test -run '^TestEngine_SessionWorkDirs' ./internal/daemon/ -count=1 -v` → 5/5 PASS; full daemon-package suite → PASS; `go vet ./internal/daemon/` clean.

### Task 2: FilesRead field + schemaVersion 2→3 + defaults-merge (FS-14)

- **RED** (commit `585c596`): three failing tests — `TestSettingsMigration_FilesReadDefaultsTrue`, `TestSettingsMigration_FilesReadExplicitFalse`, `TestSettingsMigration_FilesReadSchemaVersionRewrite` — plus the new `tests/fixtures/settings_v3.2.json` fixture (schemaVersion=2, plugins block, NO filesRead key) and `fixtureV32Path` + `copyV32FixtureToTempDir` helpers mirroring the v3.1 pattern.
- **GREEN** (commit `fb97fdb`): bumped `CurrentSchemaVersion` 2→3 in `plugin_settings.go`; added `filesRead *bool` to `SessionEngine`; added `FilesRead *bool \`json:"filesRead,omitempty"\`` to `daemonSettings` between `AutoCloseSession` and `Plugins`; pre-populated `FilesRead: &tr` in `loadSettingsFromDisk`'s defaults-merge literal BEFORE `json.Unmarshal`; copied `s.FilesRead` into `e.filesRead` after Unmarshal; serialized `FilesRead: e.filesRead` in `saveSettingsToDisk`.

Verification: `go test -run '^TestSettingsMigration' ./internal/daemon/ -count=1 -v` → 5/5 PASS (3 new + 2 existing v3.1→v3.2 / Idempotent tests still green); full daemon-package suite → PASS; `go vet` clean; `go build ./internal/daemon/` exits 0.

## Verification Results

| Check | Result |
| --- | --- |
| `go build ./internal/daemon/` | exit 0 |
| `go vet ./internal/daemon/` | no diagnostics |
| `go test -run '^TestEngine_SessionWorkDirs' ./internal/daemon/ -count=1` | PASS (5/5) |
| `go test -run '^TestSettingsMigration_FilesRead' ./internal/daemon/ -count=1` | PASS (3/3) |
| `go test -run '^TestSettingsMigration' ./internal/daemon/ -count=1` (existing v3.1→v3.2 + Idempotent) | PASS (no regression) |
| `go test ./internal/daemon/ -count=1 -short` | PASS (full package, 3.4s) |
| Schema bump `CurrentSchemaVersion = 3` | confirmed via grep |
| Fixture `tests/fixtures/settings_v3.2.json` exists and lacks `filesRead` key | confirmed (`python3 -c "..."` ok) |
| `GetSessionWorkDir` is RLock-guarded | confirmed via awk-scoped grep |

## Acceptance Criteria

All criteria from the plan are met:

- `grep -c 'sessionWorkDirs *map\[string\]string' internal/daemon/engine.go` → 1
- `grep -c 'sessionWorkDirs:.*make' internal/daemon/engine.go` → 1
- `grep -c 'e\.sessionWorkDirs\[id\] = resolvedWD' internal/daemon/engine.go` → 1
- `grep -c 'delete(e.sessionWorkDirs, id)' internal/daemon/engine.go` → 1
- `grep -c '^func (e \*SessionEngine) GetSessionWorkDir' internal/daemon/engine.go` → 1
- `awk '/^func \(e \*SessionEngine\) GetSessionWorkDir/,/^}/{ print }' internal/daemon/engine.go | grep -c 'e.mu.RLock()'` → 1
- `grep -c 'filepath.EvalSymlinks(workDir)' internal/daemon/engine.go` → 1
- `grep -c '^const CurrentSchemaVersion = 3' internal/daemon/plugin_settings.go` → 1
- `grep -c '^const CurrentSchemaVersion = 2' internal/daemon/plugin_settings.go` → 0
- `grep -c 'FilesRead.*\*bool\|filesRead.*\*bool' internal/daemon/engine.go` → 2 (daemonSettings field + SessionEngine mirror)
- `grep -c '\`json:"filesRead,omitempty"\`' internal/daemon/engine.go` → 1
- `awk '/^func \(e \*SessionEngine\) loadSettingsFromDisk/,/^}/{ print }' internal/daemon/engine.go | grep -c 'FilesRead: &tr'` → 1
- `awk '/^func \(e \*SessionEngine\) saveSettingsToDisk/,/^}/{ print }' internal/daemon/engine.go | grep -c 'FilesRead:'` → 1

## Deviations from Plan

None — plan executed exactly as written. The plan's prescribed pattern names (parallel-map per RESEARCH.md Pattern 5, EvalSymlinks-once per RESEARCH.md Pitfall 1, defaults-merge constructor per Pitfall #14/#16) all mapped cleanly onto the existing engine.go structure. The plan's pre-flight interface verification (engine.go line numbers for `tabNames`/`sessionCLIs`/`CreateSession`/`KillSession`/`loadSettingsFromDisk` blocks) was accurate within ±5 lines.

## Files Changed

| File | Change | Commit |
| --- | --- | --- |
| `internal/daemon/engine_test.go` | +125 (5 new TestEngine_SessionWorkDirs* subtests) | bda6030 |
| `internal/daemon/engine.go` | +34 / −4 (struct field + constructor + CreateSession resolution + KillSession cleanup + GetSessionWorkDir method) | d0dd47e |
| `tests/fixtures/settings_v3.2.json` | +32 (new fixture, no filesRead key) | 585c596 |
| `internal/daemon/engine_migration_test.go` | +115 (3 new tests + 2 helpers) | 585c596 |
| `internal/daemon/engine.go` | +12 / −3 (filesRead field + FilesRead struct field + defaults-merge + copy + save) | fb97fdb |
| `internal/daemon/plugin_settings.go` | +4 / −2 (schemaVersion 2→3 + doc-comment) | fb97fdb |

## Commit History

| # | Hash | Phase | Description |
| --- | --- | --- | --- |
| 1 | `bda6030` | RED  | `test(118-04): add failing tests for sessionWorkDirs + GetSessionWorkDir` |
| 2 | `d0dd47e` | GREEN | `feat(118-04): implement sessionWorkDirs map + GetSessionWorkDir` |
| 3 | `585c596` | RED  | `test(118-04): add failing migration tests for FilesRead defaults-merge` |
| 4 | `fb97fdb` | GREEN | `feat(118-04): add FilesRead field + bump schemaVersion 2→3 + defaults-merge` |

## Threat Surface

No new attack surface beyond the plan's `<threat_model>`. Mitigations in place:

- **T-118-17 (EoP via missing filesRead key)** — mitigated by defaults-merge `FilesRead: &tr` BEFORE Unmarshal; TestSettingsMigration_FilesReadDefaultsTrue verifies.
- **T-118-18 (tampering: explicit filesRead=false)** — accepted as explicit user choice; TestSettingsMigration_FilesReadExplicitFalse verifies the defaults-merge does NOT clobber.
- **T-118-19 (info-disclosure of sessionWorkDirs)** — accepted; map never crosses process boundary (Plan 05 wires GetSessionWorkDir only to the in-process file handler).
- **T-118-20 (DoS via bad workDir at CreateSession)** — mitigated by fallback to raw workDir on EvalSymlinks error; CreateSession never fails because of resolution; TestEngine_SessionWorkDirsFallbackOnEvalSymlinksError verifies.

## TDD Gate Compliance

Both tasks followed the RED → GREEN cycle:

- Task 1: `test(118-04): ...` (bda6030) → `feat(118-04): ...` (d0dd47e) ✓
- Task 2: `test(118-04): ...` (585c596) → `feat(118-04): ...` (fb97fdb) ✓

No REFACTOR commits required (both GREEN implementations were minimal and matched the plan's prescribed structure verbatim).

## Self-Check: PASSED

Files verified to exist:

- FOUND: `internal/daemon/engine.go` (modified)
- FOUND: `internal/daemon/plugin_settings.go` (modified)
- FOUND: `internal/daemon/engine_test.go` (modified)
- FOUND: `internal/daemon/engine_migration_test.go` (modified)
- FOUND: `tests/fixtures/settings_v3.2.json` (created)

Commits verified to exist:

- FOUND: `bda6030` (test RED Task 1)
- FOUND: `d0dd47e` (feat GREEN Task 1)
- FOUND: `585c596` (test RED Task 2)
- FOUND: `fb97fdb` (feat GREEN Task 2)
