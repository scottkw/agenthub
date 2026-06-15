---
phase: 128-remote-write-parity-cross-surface-integration
plan: "02"
subsystem: remote-write-parity
tags: [cap-expiry, 401, write-error, upload-queue, tui, gui, rw-05, tdd]
dependency_graph:
  requires: ["128-01"]
  provides: ["128-03", "128-04"]
  affects: ["internal/tui/remote_files_client.go", "frontend/src/lib/useFilesWrite.ts", "frontend/src/components/FileBrowserTab.tsx"]
tech_stack:
  added: []
  patterns:
    - "Sentinel error with errors.Is for typed 401 detection (Go)"
    - "TypeScript catch-branch ordering: specific predicates before generic (401 before error)"
    - "Upload queue filter-on-error vs map-to-failed for non-retryable errors"
key_files:
  created: []
  modified:
    - internal/tui/remote_files_client.go
    - internal/tui/remote_files_client_test.go
    - internal/tui/update.go
    - frontend/src/lib/filesApi.ts
    - frontend/src/lib/useFilesWrite.ts
    - frontend/src/lib/__tests__/useFilesWrite.test.tsx
    - frontend/src/components/FileBrowserTab.tsx
    - frontend/src/components/FileBrowser/__tests__/Upload.test.tsx
decisions:
  - "ErrRemoteCapExpired uses a cap-free fixed string (no interpolated URL or token) — CAP-LEAK invariant preserved"
  - "Upload queue entry is removed (filter) on 401, not mapped to 'failed' — no stuck progress bar"
  - "Buffer preservation on 401 (T-125-08) verified by checking the 401 branch produces ACCESS_EXPIRED_MESSAGE not the generic copy"
  - "Server temp-cleanup on interrupted write confirmed already correct via existing TestWriteFileAtomic (no new server code)"
metrics:
  duration: "~25 minutes"
  completed: "2026-06-14"
  tasks_completed: 3
  files_modified: 8
---

# Phase 128 Plan 02: Cap-expiry mid-edit "access expired" + upload-abort queue cleanup (RMW-05) Summary

**One-liner:** HTTP 401 mid-edit now returns ErrRemoteCapExpired (Go) / WriteOutcome 'expired' (TS) with buffer preserved and a distinct "access expired" message; in-flight upload 401s remove the queue entry instead of leaving a stuck progress bar.

## Tasks Completed

| # | Task | Commit | Result |
|---|------|--------|--------|
| 1 RED | Go 401 sentinel tests (6 sub-tests) | 04eb990 | RED confirmed — ErrRemoteCapExpired undefined |
| 1 GREEN | ErrRemoteCapExpired + 401 checks in 4 write methods + TUI surface | 9f468a0 | GREEN — all TestRemoteFilesClient_401 pass |
| 2 RED | TS expired outcome + ACCESS_EXPIRED_MESSAGE tests (8 tests) | 86365c8 | RED confirmed — 4 tests fail |
| 2 GREEN | WriteOutcome 'expired' + isUnauthorized catch branch + ACCESS_EXPIRED_MESSAGE | 6bc0d89 | GREEN — 35/35 vitest pass |
| 3 RED | Upload-abort queue cleanup tests (source-inspection) | bb0bbe8 | RED confirmed — filter test fails |
| 3 GREEN | Upload isUnauthorized branch removes queue entry; server cleanup confirmed | 8eec6a5 | GREEN — 37/37 vitest pass |

## Key Changes

### Go Surface (TUI)

**`internal/tui/remote_files_client.go`**
- Added `ErrRemoteCapExpired = errors.New("your access to this remote session has expired")` sentinel
- Added `if resp.StatusCode == http.StatusUnauthorized` check BEFORE the generic `!= http.StatusOK` block in all 4 write methods (WriteFile, DeleteFile, RenameFile, MkdirFile)
- CAP-LEAK invariant preserved: fixed string, no token/URL interpolation

**`internal/tui/update.go`**
- Added `else if errors.Is(msg.err, ErrRemoteCapExpired)` branch in `applyFilesOpMsg` that sets a distinct "Your access to this remote session has expired." toast instead of the generic "<op> failed: ..." copy

### TS Surface (GUI/Web)

**`frontend/src/lib/filesApi.ts`**
- Added `ACCESS_EXPIRED_MESSAGE` export: "Your access to this remote session has expired. Your changes are still here." (cap-free; emphasizes buffer retention)

**`frontend/src/lib/useFilesWrite.ts`**
- Extended `WriteOutcome` to include `'expired'`
- Added `isUnauthorized()` catch branch AFTER `isMethodNotAllowed()` (RMW-04) and BEFORE the generic branch — returns `'expired'`, sets `saveError(ACCESS_EXPIRED_MESSAGE)`, does NOT clear editContent (T-125-08 locked)

**`frontend/src/components/FileBrowserTab.tsx`**
- Added `isUnauthorized()` check in upload catch block: filters queue entry out (`.filter(item => item.id !== itemId)`) instead of mapping to `'failed'`
- Added documentation comment for `'expired'` outcome in UnsavedChangesModal dispatch

## Verification Results

```
go test ./internal/tui/ -run TestRemoteFilesClient -race -count=1  → ok
go test ./internal/files/ -run TestWriteFileAtomic -race -count=1  → ok
pnpm exec vitest run src/lib/__tests__/useFilesWrite.test.tsx src/lib/__tests__/filesApi.test.ts  → 35/35 pass
pnpm exec vitest run src/components/FileBrowser/__tests__/Upload.test.tsx  → 37/37 pass
tsc --noEmit  → clean
gofmt  → clean
go vet ./internal/tui/  → clean
```

## Success Criteria Check

- [x] 401 → 'expired' outcome + "access expired" msg (both surfaces)
- [x] Buffer NOT cleared on 401 (T-125-08 not regressed — asserted by test)
- [x] Upload queue entry removed on 401 (filter, not map-to-failed)
- [x] Cap NOT leaked in 401 message (fixed strings; CAP-LEAK invariant)
- [x] Server temp cleanup on interrupted write already correct (WriteFileAtomic root.Remove(tmp))
- [x] go test ./internal/tui/ -run TestRemoteFilesClient green
- [x] go test ./internal/files/ -run TestWriteFileAtomic green
- [x] pnpm exec vitest run green
- [x] tsc/gofmt/vet clean

## Deviations from Plan

**1. [Rule 1 - Bug] Test predicate used wrong occurrence index for source-inspection test**
- Found during: Task 3 GREEN
- Issue: `fileBrowserTabRaw.indexOf('isUnauthorized()')` found the first occurrence (line 317, auth gate in DaemonManagerPanel handler) rather than the upload loop occurrence (line 825). The `isOverCap()` anchor had the same problem.
- Fix: Changed to `fileBrowserTabRaw.lastIndexOf('isOverCap()')` which uniquely identifies the upload loop occurrence.
- Files modified: Upload.test.tsx
- No additional commit (fixed within Task 3 GREEN implementation cycle)

**2. [Rule 1 - Bug] TestFilesOpMsg_401_AccessExpiredToast used wrong case in string comparison**
- Found during: Task 1 GREEN
- Issue: Test checked `strings.Contains(got, "your access")` but the TUI toast uses "Your access" (capital Y).
- Fix: Changed to `strings.ToLower(got)` comparison.
- No additional commit (fixed within Task 1 GREEN implementation cycle)

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. All new error strings are fixed constants (no interpolation). CAP-LEAK invariant maintained throughout.

## Known Stubs

None. All branches are fully implemented.

## Self-Check

Commits verified:
- 04eb990: test(128-02) RED Go
- 9f468a0: feat(128-02) GREEN Go
- 86365c8: test(128-02) RED TS
- 6bc0d89: feat(128-02) GREEN TS
- bb0bbe8: test(128-02) RED upload
- 8eec6a5: feat(128-02) GREEN upload

All 8 files modified exist in the worktree.

## Self-Check: PASSED
