---
phase: 137-share-modal-cap-model
plan: "02"
subsystem: capability-model
tags: [security, capability, browse-matrix, green-implementation, perm-injection]
dependency_graph:
  requires: ["137-01"]
  provides:
    - sessionBrowse map + browseEnabledFor/SetSessionBrowse (engine.go)
    - D-03/D-04 perm injection matrix (api.go)
    - POST /sessions/{id}/browse daemon endpoint + ClearGrants (api.go)
    - SessionBrowseRequest type + SessionInfo.BrowseEnabled (types.go)
    - SetSessionBrowse client method (client.go)
    - SetSessionBrowse Wails binding (app.go)
  affects:
    - internal/daemon/engine.go
    - internal/daemon/api.go
    - internal/daemon/types.go
    - internal/daemon/client.go
    - app.go
    - app_test.go
tech_stack:
  added: []
  patterns:
    - Per-session in-memory bool map (sessionBrowse) with RLock/Lock mutex pattern
    - D-03/D-04 perm injection driven solely by browseEnabledFor (no global flag)
    - ClearGrants on browse toggle (stale-cap threat mitigation parity with handleWebServe)
key_files:
  created: []
  modified:
    - internal/daemon/engine.go
    - internal/daemon/api.go
    - internal/daemon/types.go
    - internal/daemon/client.go
    - app.go
    - app_test.go
decisions:
  - "filesWriteDefault retained in engine struct (NOT removed) to preserve TestSettingsMigration_FilesWriteDefaultsFalse which directly accesses e.filesWriteDefault — this is a test-parity decision, not a functional one; filesWriteDefault is NOT wired to perm injection"
  - "FilesRead *bool field removed from daemonSettings (global kill-switch gone per D-07); FilesWrite bool retained for schema migration compatibility"
  - "ClearGrants fires on EVERY browse toggle (not just toggle-off) — safer than toggle-off-only; the next IssueCapabilities call re-mints with correct perms"
  - "app_test.go updated: TestListSessions_PropagatesHomeDirAndFilesWrite renamed to PropagatesHomeDirAndBrowseEnabled; uses SetSessionBrowse instead of SetSessionFilesWrite"
metrics:
  duration_minutes: 25
  completed_date: "2026-06-20"
  tasks_completed: 2
  tasks_total: 2
  files_created: 0
  files_modified: 6
---

# Phase 137 Plan 02: GREEN Implementation — Browse-Matrix Cap Model Summary

**One-liner:** Single per-session sessionBrowse map + D-03/D-04 perm injection replaces the global filesRead kill-switch and per-session files-write two-gate, turning the Plan 01 RED tests GREEN.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Collapse engine fields into sessionBrowse; rewrite api.go perm injection (D-02/D-03/D-04/D-07) | fad0275f | internal/daemon/engine.go, internal/daemon/api.go |
| 2 | Add browse endpoint + client method + Wails binding + SessionInfo field; retire files-write surface | 50fdded0 | internal/daemon/types.go, internal/daemon/client.go, app.go, app_test.go |

## What Was Built

### Task 1 — engine.go field collapse + api.go perm injection rewrite

**engine.go changes:**
- Removed struct fields: `filesRead *bool`, `sessionWrites map[string]bool`
- Retained (test-parity, see Deviations): `filesWriteDefault bool`
- Added: `sessionBrowse map[string]bool` — per-session browse toggle, default OFF (D-06/D-08)
- Removed from `daemonSettings`: `FilesRead *bool` (global kill-switch removed, D-07)
- Removed from `loadSettingsFromDisk`: FilesRead defaults-merge literal (`tr := true; s := daemonSettings{FilesRead: &tr, ...}`)
- Removed methods: `filesReadEnabled()`, `filesWriteEnabledFor()`, `filesWriteEnabledForUnlocked()`, `SetSessionFilesWrite()`
- Added methods: `browseEnabledFor()` (with RLock), `browseEnabledForUnlocked()` (lock-free, for ListSessions deadlock avoidance), `SetSessionBrowse()` (with Lock)
- Updated `NewSessionEngine`: `sessionWrites: make(map[string]bool)` → `sessionBrowse: make(map[string]bool)`
- Updated `ListSessions`: `FilesWrite: e.filesWriteEnabledForUnlocked(s.ID)` → `BrowseEnabled: e.browseEnabledForUnlocked(s.ID)`

**api.go perm injection rewrite (lines 1096-1116 replacement):**
- Old: `ownerPerms` built from `filesReadEnabled()` + conditional `filesWriteEnabledFor()` → single owner token
- New: D-03/D-04 matrix: `rPerms := "read"`, `wPerms := "read,write"`; if `browseEnabledFor(sessionID)` then `rPerms = "read,files.read"` and `wPerms = "read,write,files.read,files.write"`
- Security comment with Reversal 1, Reversal 3 references + "Audit: secure-phase to review" annotation

**api.go browse handler + route:**
- Retired: `POST /sessions/{id}/files-write` → `handleSetSessionFilesWrite`
- Added: `POST /sessions/{id}/browse` → `handleSetSessionBrowse`
- `handleSetSessionBrowse` decodes `SessionBrowseRequest`, calls `engine.SetSessionBrowse`, then clears grants (`ws.ClearGrants(id)`) for stale-cap threat mitigation (SHARE-05, mirrors handleWebServe pattern)

### Task 2 — types/client/app binding layer

**types.go:**
- `SessionInfo`: removed `FilesWrite bool json:"filesWrite"`, added `BrowseEnabled bool json:"browseEnabled"` (NOT omitempty — false must serialize per SHARE-05/RESEARCH open question 2)
- Removed `SessionFilesWriteRequest`; added `SessionBrowseRequest{ Enabled bool json:"enabled" }`

**client.go:**
- Removed `SetSessionFilesWrite`; added `SetSessionBrowse(sessionID string, enabled bool) error` using `doJSON(POST, "/sessions/"+sessionID+"/browse", SessionBrowseRequest{Enabled: enabled}, nil)`

**app.go:**
- Updated `SessionInfo` struct: `FilesWrite bool` → `BrowseEnabled bool`
- Updated `ListSessions`: `FilesWrite: s.FilesWrite` → `BrowseEnabled: s.BrowseEnabled`
- Removed `SetSessionFilesWrite` Wails binding; added `SetSessionBrowse` with nil-client guard

**app_test.go (deviation — see below):**
- Updated `TestListSessions_PropagatesHomeDirAndFilesWrite` → `TestListSessions_PropagatesHomeDirAndBrowseEnabled`
- Uses `SetSessionBrowse` and asserts `BrowseEnabled` propagation end-to-end

## Verification Results

```
go build ./... — PASSED (zero dangling references)

go test ./internal/daemon/... -run TestIssueCapabilities:
  TestIssueCapabilities_BrowseOff_NoFilesPerms — PASS (D-03: RO="read", RW="read,write")
  TestIssueCapabilities_BrowseOn_ROPermsExact  — PASS (D-04: RO="read,files.read", no files.write)
  TestIssueCapabilities_BrowseOn_RWPermsExact  — PASS (D-04: RW="read,write,files.read,files.write")

go test ./internal/webserver/... -run TestFilesRoutes_R:
  TestFilesRoutes_RO_BrowseOn_FilesReadRoute200 — PASS
  TestFilesRoutes_RO_BrowseOn_WriteRoute403     — PASS
  TestFilesRoutes_RW_BrowseOn_WriteRoute200     — PASS

go test ./internal/webserver/... -run TestRequireFiles:
  TestRequireFilesRead (all subtests)  — PASS
  TestRequireFilesWrite (all subtests) — PASS

go test ./internal/daemon/...  — PASS (all daemon tests)
go test ./internal/webserver/... — PASS (all webserver tests)
go test github.com/scottkw/agenthub — PASS (main package + app_test.go)
```

Grep gates:
- Zero `func.*filesReadEnabled` in engine.go
- Zero `func.*filesWriteEnabledFor` in engine.go
- Zero `func.*SetSessionFilesWrite` in engine.go
- Zero `SetSessionFilesWrite` references in internal/daemon/ and app.go
- `browseEnabledFor` present in engine.go (2 methods: locked + unlocked)
- `SetSessionBrowse` present in engine.go (1 method)
- api.go perm block: `browseEnabledFor` call + Reversal 1 + Reversal 3 audit comment

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Retained filesWriteDefault field (not removed)**
- **Found during:** Task 1
- **Issue:** The plan's behavior spec says "filesWriteDefault no longer exists anywhere in the package." But `engine_migration_test.go:251` (package `daemon` internal test) directly accesses `e.filesWriteDefault`. This is a "survivor" test from Plan 01 (`TestSettingsMigration_FilesWriteDefaultsFalse`) that must remain passing. Removing the field would produce a compile error.
- **Decision:** Retain `filesWriteDefault` in the engine struct as a settings-persistence field, NOT wired to perm injection. Comment clearly documents this is test-parity retention only (D-07 reversal does NOT affect FilesWrite schema).
- **Files modified:** internal/daemon/engine.go (field retained with revised comment)
- **Impact:** Zero security impact — `filesWriteDefault` is loaded from disk but never read by `issueCapabilitiesForSession`. The perm injection reads only `browseEnabledFor`. The surviving migration test continues to pass.
- **Commits:** fad0275f

**2. [Rule 1 - Bug] app_test.go updated (SetSessionFilesWrite → SetSessionBrowse)**
- **Found during:** Task 2
- **Issue:** `app_test.go` used `SetSessionFilesWrite` and `FilesWrite` in `TestListSessions_PropagatesHomeDirAndFilesWrite`. After removing the binding, the test failed to compile.
- **Fix:** Updated test to use `SetSessionBrowse` and `BrowseEnabled`. Test renamed to `TestListSessions_PropagatesHomeDirAndBrowseEnabled`. Behavior is identical (prove true value propagates end-to-end).
- **Files modified:** app_test.go
- **Commits:** 50fdded0

## Pre-existing Deferred Issues

`internal/release TestSER03_NoAutoSavePatterns` fails due to a playwright-fixture JS build artifact (`cmd/playwright-fixture/dist/assets/index-Dklc5ak1.js`) containing an auto-save vocabulary pattern. This failure is pre-existing (not introduced by this plan) and unrelated to the browse-cap model changes. Logged to `deferred-items.md` scope boundary; no fix attempted.

## Known Stubs

None — this plan is backend Go implementation only. No frontend stubs introduced.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: stale-cap-on-browse-on | internal/daemon/api.go | ClearGrants fires on browse toggle-ON as well as toggle-off. This is intentional and safer (old files.read caps from before-browse-on are invalidated); documented in handler comment. |

Note: The above is not a new threat — it is the correct (conservative) interpretation of the stale-cap mitigation. The plan only required ClearGrants on toggle-off; the implementation fires on every toggle, which is strictly safer and matches the handleWebServe pattern exactly.

## Self-Check: PASSED

Files modified:
- internal/daemon/engine.go — exists, contains `func (e *SessionEngine) browseEnabledFor`
- internal/daemon/api.go — exists, contains `browseEnabledFor(sessionID)` in perm block + `handleSetSessionBrowse`
- internal/daemon/types.go — exists, contains `BrowseEnabled bool json:"browseEnabled"`
- internal/daemon/client.go — exists, contains `func (c *DaemonClient) SetSessionBrowse`
- app.go — exists, contains `func (a *App) SetSessionBrowse`
- app_test.go — exists, contains `TestListSessions_PropagatesHomeDirAndBrowseEnabled`

Commits:
- fad0275f — Task 1 (engine.go + api.go)
- 50fdded0 — Task 2 (types.go, client.go, app.go, app_test.go)

All three Plan 01 RED tests now GREEN:
- TestIssueCapabilities_BrowseOff_NoFilesPerms: PASS
- TestIssueCapabilities_BrowseOn_ROPermsExact: PASS
- TestIssueCapabilities_BrowseOn_RWPermsExact: PASS
