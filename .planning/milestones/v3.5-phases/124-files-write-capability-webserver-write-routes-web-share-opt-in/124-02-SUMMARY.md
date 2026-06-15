---
phase: 124-files-write-capability-webserver-write-routes-web-share-opt-in
plan: 02
subsystem: daemon settings + capability minting + per-session write state
tags: [capability, settings, migration, per-session, go, security, homeDir]
requirements: [CAP-04, CAP-06, CAP-08]

dependency_graph:
  requires:
    - internal/capability/capability.go (PermFilesWrite — Phase 124-01)
    - internal/daemon/engine.go (loadSettingsFromDisk, GetSessionWorkDir, filesReadEnabled pattern)
    - internal/daemon/api.go (issueCapabilitiesForSession, ownerPerms mint pattern)
    - internal/files/sandbox.go (EvalSymlinks $HOME normalization pattern)
  provides:
    - CurrentSchemaVersion = 4 (schemaVersion bump)
    - FilesWrite bool in daemonSettings (plain bool, zero=false, opt-in for all)
    - filesWriteDefault + sessionWrites map in SessionEngine
    - filesWriteEnabledFor (per-session check, T-124-07)
    - SetSessionFilesWrite (per-session toggle setter for GUI binding)
    - sessionCwdIsHome + cwdEqualsHome (EvalSymlinks-normalized homeDir signal)
    - HomeDir + FilesWrite fields on SessionInfo (TUI + GUI parity)
    - HomeDir field on IssueCapabilitiesResponse (GUI warning signal)
    - Cap mint appends files.write only when per-session toggle is ON
  affects:
    - internal/daemon/plugin_settings.go (schemaVersion bump)
    - internal/daemon/engine.go (new fields, new methods)
    - internal/daemon/types.go (SessionInfo + IssueCapabilitiesResponse)
    - internal/daemon/api.go (cap mint + response population)
    - internal/daemon/engine_migration_test.go (new migration tests)

tech_stack:
  added: []
  patterns:
    - Per-session write map (keyed by sessionID under e.mu) — inversion of global filesRead *bool
    - EvalSymlinks($HOME) before comparing to already-resolved session cwd (T-124-08)
    - Append-only ownerPerms string mutation before Sign (HMAC field order invariant)
    - Unlocked variants of hot-path methods called inside ListSessions RLock

key_files:
  created: []
  modified:
    - internal/daemon/plugin_settings.go
    - internal/daemon/engine.go
    - internal/daemon/types.go
    - internal/daemon/api.go
    - internal/daemon/engine_migration_test.go

decisions:
  - "FilesWrite is plain bool (NOT *bool like FilesRead) — zero-value false is the opt-in default; no pre-population needed in the defaults-merge literal"
  - "Per-session write state uses an in-memory map keyed by sessionID under e.mu; persisted default seeds the fallback for sessions without explicit entries"
  - "sessionCwdIsHomeUnlocked + filesWriteEnabledForUnlocked variants added to avoid deadlock inside ListSessions which already holds e.mu.RLock"
  - "Worktree was behind main (missing Phase 124-01 PermFilesWrite); merged main before finalizing Task 2 cap mint code"

metrics:
  duration: ~45 minutes
  completed: 2026-06-14
  tasks_completed: 3
  files_changed: 5
---

# Phase 124 Plan 02: Per-Session files.write Opt-In + Settings Migration + Cap Mint Summary

**One-liner:** schemaVersion 3→4 migration with FilesWrite default false, per-session write map with filesWriteEnabledFor, EvalSymlinks-normalized homeDir signal, and cap mint that appends files.write only when the per-session toggle is ON.

## Tasks Completed

| # | Name | Commit | Status |
|---|------|--------|--------|
| 0 | Write Wave-0 failing migration tests (RED) | `6928c46` | DONE |
| 1 | schemaVersion 4 + FilesWrite settings + per-session write map + homeDir signal | `82a6f31` | DONE |
| 2 | Cap mint appends files.write + HomeDir wire field | `1e293fb` | DONE |

## What Was Built

### `internal/daemon/plugin_settings.go`
Bumped `CurrentSchemaVersion` from 3 to 4. Extended the doc comment with Phase 124 / FilesWrite rationale.

### `internal/daemon/engine.go`

**Settings structure (`daemonSettings`):**
Added `FilesWrite bool json:"filesWrite,omitempty"` — plain bool (NOT `*bool`), zero-value false. This is Critical Inversion 2: files.write is opt-in for all (CAP-08), not opt-out like FilesRead.

**SessionEngine struct fields:**
- `filesWriteDefault bool` — persisted default loaded from settings; seeds new sessions
- `sessionWrites map[string]bool` — per-session write toggle, in-memory, keyed by sessionID under e.mu (T-124-07)

**New methods:**
- `filesWriteEnabledFor(sessionID string) bool` — acquires RLock, reads per-session map, falls back to filesWriteDefault (NOT a global flag — Inversion 2)
- `filesWriteEnabledForUnlocked(sessionID string) bool` — lock-free variant for ListSessions
- `SetSessionFilesWrite(sessionID string, enabled bool)` — per-session toggle setter for GUI binding (CAP-04)
- `sessionCwdIsHome(sessionID string) bool` — EvalSymlinks($HOME) normalized comparison
- `sessionCwdIsHomeUnlocked(sessionID string) bool` — lock-free variant for ListSessions
- `cwdEqualsHome(cwd string) bool` — extracted helper with EvalSymlinks call (T-124-08: macOS /var→/private/var trap)

**ListSessions:**
Populates `HomeDir` and `FilesWrite` on each `SessionInfo` using the unlocked variants — single server-side source of truth for both GUI and TUI (cross-surface parity is release-blocking per MEMORY.md).

**Migration path:**
`loadSettingsFromDisk` populates `e.filesWriteDefault = s.FilesWrite` (zero-value false on v3 fixture with no `filesWrite` key). The existing `needsUpgradeWrite := s.SchemaVersion < CurrentSchemaVersion` path fires on the 3→4 bump and rewrites on-disk schemaVersion to 4.

### `internal/daemon/types.go`

**`SessionInfo`:**
Added `HomeDir bool json:"homeDir"` and `FilesWrite bool json:"filesWrite"` — populated from `sessionCwdIsHomeUnlocked` and `filesWriteEnabledForUnlocked` in `ListSessions`. Both fields are the cross-surface source of truth read by GUI session list and TUI (plan 124-05).

**`IssueCapabilitiesResponse`:**
Added `HomeDir bool json:"homeDir"` — populated from `engine.sessionCwdIsHome(id)` at cap issuance. The frontend reads this field to decide whether to show the home-write warning banner (plan 124-04).

### `internal/daemon/api.go`

**`issueCapabilitiesForSession`:**
After the existing `filesReadEnabled()` block, appended:
```go
if a.engine.filesWriteEnabledFor(sessionID) {
    ownerPerms += "," + capability.PermFilesWrite
}
```
Uses `capability.PermFilesWrite` constant (never `strings.Contains`), reads the per-session toggle (not a global flag), and appends to the ownerPerms string before `Sign` (HMAC field order invariant preserved — T-124-09). rClaims `Perms: "read"` is never modified.

**`handleIssueCapabilities`:**
Populates `HomeDir: a.engine.sessionCwdIsHome(id)` in the response JSON.

### `internal/daemon/engine_migration_test.go`
Added two new tests:
- `TestSettingsMigration_FilesWriteDefaultsFalse` — uses `copyV32FixtureToTempDir` harness, loads via `loadSettingsFromDisk`, asserts `e.filesWriteDefault == false` (opt-in for all)
- `TestSettingsMigration_FilesWriteSchemaVersionRewrite` — asserts on-disk `schemaVersion == 4` after load from v3 fixture

## Verification

All success criteria met:

- `go test -race ./internal/daemon/ -count=1` GREEN
- `go test -run 'TestSettingsMigration_FilesWriteDefaultsFalse' ./internal/daemon/` exits 0
- `go vet ./internal/daemon/` clean
- `gofmt -l` reports no diffs on all touched files

## Deviations from Plan

### Pre-execution: Worktree behind main — merged main before finalizing Task 2

**Found during:** Task 2 build failure (`undefined: capability.PermFilesWrite`)
**Issue:** The worktree was initialized at `d725107` (pre-Phase-124-01). `capability.PermFilesWrite` was missing — Task 2 cap mint would not compile.
**Fix:** Applied Task 2 changes to `api.go` and `types.go` as a WIP commit (cff5d4e), then ran `git merge --no-edit main` to bring in Phase 124-01 (`PermFilesWrite`, `requireFilesWrite`, route mounts). Auto-merged cleanly.
**Impact:** No plan-specified files were modified by the merge unexpectedly; the merge only added Phase 124-01 implementation + Phase 123 write engine that were prerequisites.

### [Rule 1 - Bug] Added unlocked variants to avoid deadlock in ListSessions

**Found during:** Task 1 implementation
**Issue:** `ListSessions` holds `e.mu.RLock` via `defer e.mu.RUnlock()`. Calling `sessionCwdIsHome` or `filesWriteEnabledFor` inside the loop would try to acquire `e.mu.RLock` again — on a non-reentrant mutex this would deadlock.
**Fix:** Extracted `sessionCwdIsHomeUnlocked` and `filesWriteEnabledForUnlocked` variants that assume caller holds the lock. ListSessions calls the unlocked variants directly.
**Files modified:** `internal/daemon/engine.go`
**Commit:** `82a6f31`

## Known Stubs

None — no stub values, placeholder text, or unwired data sources introduced in this plan.

## Threat Flags

No new network endpoints or auth paths beyond the plan's threat model. Mitigations applied:
- T-124-06: `FilesWrite bool` zero-value false; `TestSettingsMigration_FilesWriteDefaultsFalse` asserts opt-in default
- T-124-07: Per-session map keyed by sessionID; `filesWriteEnabledFor` reads only that session's entry
- T-124-08: `cwdEqualsHome` calls `filepath.EvalSymlinks(home)` before comparing to the already-resolved `GetSessionWorkDir` result
- T-124-09: Only the `ownerPerms` string is mutated before `Sign`; `Claims` struct field order unchanged

## Self-Check: PASSED

- FOUND: internal/daemon/plugin_settings.go (CurrentSchemaVersion = 4)
- FOUND: internal/daemon/engine.go (filesWriteEnabledFor, sessionCwdIsHome, sessionWrites map)
- FOUND: internal/daemon/types.go (HomeDir + FilesWrite on SessionInfo; HomeDir on IssueCapabilitiesResponse)
- FOUND: internal/daemon/api.go (filesWriteEnabledFor call + HomeDir population)
- FOUND: internal/daemon/engine_migration_test.go (TestSettingsMigration_FilesWriteDefaultsFalse)
- FOUND: commit 6928c46 (RED migration tests)
- FOUND: commit 82a6f31 (schemaVersion 4 + per-session write map + homeDir signal)
- FOUND: commit 1e293fb (cap mint + IssueCapabilitiesResponse.HomeDir)
