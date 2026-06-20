---
phase: 137-share-modal-cap-model
plan: "01"
subsystem: test-contract
tags: [tdd-red, security, capability, browse-matrix, frontend-tests]
dependency_graph:
  requires: []
  provides:
    - D-03/D-04 browse-perm matrix test contract (Go)
    - Browse-aware route enforcement tests
    - Whole-token grep gate for perm injection (T-137-06)
    - Share button + modal frontend test contract
    - Simplified SessionSharePanel post-CAP-05 test contract
  affects:
    - internal/daemon/api_test.go
    - internal/daemon/engine_migration_test.go
    - internal/webserver/files_routes_test.go
    - internal/webserver/capability_test.go
    - frontend/src/components/__tests__/SessionCard.share.test.tsx
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
    - frontend/src/components/__tests__/SessionSharePanel.test.tsx
tech_stack:
  added: []
  patterns:
    - Red-test-first contract via issueCapFor + exact perm string assertions
    - Comment-filtered static-grep gate for whole-token-only perm checks
    - vi.mock + createRoot/flushSync pattern for React component RED tests
key_files:
  created:
    - frontend/src/components/__tests__/SessionCard.share.test.tsx
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
  modified:
    - internal/daemon/api_test.go
    - internal/daemon/engine_migration_test.go
    - internal/webserver/files_routes_test.go
    - internal/webserver/capability_test.go
    - frontend/src/components/__tests__/SessionSharePanel.test.tsx
decisions:
  - "issueCapsTestSetup helper drops filesRead *bool param — D-07 deliberate removal; browse is per-session with no global flag"
  - "Exact string equality (not HasPerm) used for matrix assertions so a stray extra perm fails the test"
  - "Comment-filtered grep in TestHasPerm_NoStringsContains_Browse prevents doc-comment false positives"
  - "TestSettingsMigrationIdempotent and TestSettingsMigration_FilesWriteDefaultsFalse survive untouched"
metrics:
  duration_minutes: 20
  completed_date: "2026-06-20"
  tasks_completed: 3
  tasks_total: 3
  files_created: 2
  files_modified: 5
---

# Phase 137 Plan 01: RED Test Contract — Browse-Matrix & Share Modal Summary

**One-liner:** D-03/D-04 browse-perm matrix Go tests + browse-aware webserver route tests + Share modal/card frontend RED tests define the Phase 137 executable specification.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Retire obsolete Go tests; add D-03/D-04 browse-matrix tests (RED) | 657dbb1e | internal/daemon/api_test.go, internal/daemon/engine_migration_test.go |
| 2 | Add browse-aware webserver route tests + whole-token grep gate | af89d4cb | internal/webserver/files_routes_test.go, internal/webserver/capability_test.go |
| 3 | Create frontend test files (Share button + modal) and update SessionSharePanel tests | becb06cc | frontend/src/components/__tests__/SessionCard.share.test.tsx (new), frontend/src/components/__tests__/SessionShareModal.test.tsx (new), frontend/src/components/__tests__/SessionSharePanel.test.tsx |

## What Was Built

This plan establishes the Nyquist test contract for Phase 137 **before** any implementation. All tests in Tasks 1 and 3 are RED (expected to fail until Plans 02/03 land). Task 2 route tests pass immediately because they use literal perm strings against already-existing routes.

### Task 1 — Go browse-matrix RED tests

Retired from `api_test.go`:
- `TestIssueCapabilities_OwnerHasFilesRead_WhenSettingNil`
- `TestIssueCapabilities_ViewerNoFilesRead`
- `TestIssueCapabilities_OwnerNoFilesReadWhenDisabled`
- `TestIssueCapabilities_OwnerHasFilesReadWhenExplicitTrue`

The `issueCapsTestSetup` helper was rewritten to drop the `filesRead *bool` parameter (Plan 02 removes `e.filesRead`).

Added three RED tests (compile-fail until Plan 02):
- `TestIssueCapabilities_BrowseOff_NoFilesPerms` — D-03: browse OFF → RO="read", RW="read,write"
- `TestIssueCapabilities_BrowseOn_ROPermsExact` — D-04 + T-137-02: browse ON RO="read,files.read", no files.write
- `TestIssueCapabilities_BrowseOn_RWPermsExact` — D-04: browse ON RW="read,write,files.read,files.write" exact

Retired from `engine_migration_test.go`:
- `TestSettingsMigration_FilesReadDefaultsTrue`
- `TestSettingsMigration_FilesReadExplicitFalse`

Survivors untouched: `TestSettingsMigrationIdempotent`, `TestSettingsMigrationV3_1ToV3_2`, `TestSettingsMigration_FilesWriteDefaultsFalse`, `TestSettingsMigration_FilesWriteSchemaVersionRewrite`, `TestSettingsMigration_FilesReadSchemaVersionRewrite`.

### Task 2 — Browse-aware route tests + whole-token grep gate

Added to `files_routes_test.go`:
- `TestFilesRoutes_RO_BrowseOn_FilesReadRoute200` — browse-ON RO cap → 200 on /list
- `TestFilesRoutes_RO_BrowseOn_WriteRoute403` — browse-ON RO cap → 403 on /write (T-137-03)
- `TestFilesRoutes_RW_BrowseOn_WriteRoute200` — browse-ON RW cap → 200 on HEAD /write

Added to `capability_test.go`:
- `TestHasPerm_NoStringsContains_Browse` — comment-filtered grep gate over api.go + engine.go for strings.Contains + perm literal (T-137-06)

All four tests pass immediately. Existing `TestRequireFilesWrite` / `TestRequireFilesRead` tests are unmodified.

### Task 3 — Frontend RED test files + SessionSharePanel update

**Created `SessionCard.share.test.tsx`** (RED until Plan 03):
- Share button renders on local card with accessible label "Share <name>"
- Click fires `onShare` and does NOT bubble to card click handler (stopPropagation — Pitfall 6)
- Remote card: Share button disabled + aria-label/title "Only the session owner can share" + `.hub-card__share-lock` icon (D-13 colorblind-safe)

**Created `SessionShareModal.test.tsx`** (RED until Plan 03):
- SHARE-01: "Share the session" toggle present
- SHARE-02: browse toggle disabled when sharing OFF
- SHARE-06/Pitfall 1: browse toggle calls `SetSessionBrowse` then `IssueCapabilities`
- SHARE-04: local-mode calls `GetLocalNetworkPassword`
- D-09: homeDir fixture shows home-dir warning
- SHARE-05: server-truth seeding calls `IssueCapabilities` once on open
- SHARE-05: stale-URL clear after webServerRunning flip

**Updated `SessionSharePanel.test.tsx`** (post-CAP-05 contract):
- Removed entire CAP-05 describe blocks (ownerWriteEnabled, viewer opt-in, confirm dialog, surfaceWriteLink gate, write URL gating tests)
- Added: write link always rendered when writeURL/writeCode provided
- Added: browseEnabled=false → "watch only" scope text; browseEnabled=true → file access scope text
- Kept: read code / join code assertions, v3.5 link scope clarity tests (unchanged)

## Red-Test Baseline

The following tests are expected to remain RED (compile or run fail) until the specified implementation plan:

| Test | Reason | Resolves in |
|------|--------|-------------|
| `TestIssueCapabilities_BrowseOff_NoFilesPerms` | Calls `api.engine.SetSessionBrowse` (not yet on engine) | Plan 02 |
| `TestIssueCapabilities_BrowseOn_ROPermsExact` | Same | Plan 02 |
| `TestIssueCapabilities_BrowseOn_RWPermsExact` | Same | Plan 02 |
| All `SessionCard.share.test.tsx` tests | `SessionCard` has no `onShare` prop or Share button | Plan 03 |
| All `SessionShareModal.test.tsx` tests | `SessionShareModal` component does not exist | Plan 03 |
| Some `SessionSharePanel.test.tsx` tests | Panel still has `ownerWriteEnabled` prop until Plan 03 strips it | Plan 03 |

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — this plan is test-only; no production stubs introduced.

## Threat Flags

No new network endpoints, auth paths, or schema changes introduced (test-only plan). T-137-02/T-137-03/T-137-06 audit targets are pinned as test contracts.

## Self-Check: PASSED

Files created/modified:
- internal/daemon/api_test.go — exists, contains TestIssueCapabilities_BrowseOn_ROPermsExact
- internal/daemon/engine_migration_test.go — exists, FilesReadDefaultsTrue removed
- internal/webserver/files_routes_test.go — exists, contains TestFilesRoutes_RO_BrowseOn_WriteRoute403
- internal/webserver/capability_test.go — exists, contains TestHasPerm_NoStringsContains_Browse
- frontend/src/components/__tests__/SessionCard.share.test.tsx — exists (created)
- frontend/src/components/__tests__/SessionShareModal.test.tsx — exists (created)
- frontend/src/components/__tests__/SessionSharePanel.test.tsx — exists, CAP-05 tests removed

Commits:
- 657dbb1e — Task 1
- af89d4cb — Task 2
- becb06cc — Task 3
