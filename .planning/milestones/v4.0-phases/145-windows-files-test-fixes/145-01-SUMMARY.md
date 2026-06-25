---
phase: 145-windows-files-test-fixes
plan: "01"
subsystem: internal/files
tags: [test-fix, windows-ci, denylist, home-rooting]
dependency_graph:
  requires: []
  provides: [windows-files-test-compat]
  affects: [internal/files]
tech_stack:
  added: []
  patterns: [setHomeEnv-fake-home-redirect]
key_files:
  created: []
  modified:
    - internal/files/handler_test.go
    - internal/files/write_test.go
decisions:
  - "Use fakeHome redirect via setHomeEnv rather than OS-specific skip guards — setHomeEnv already existed as the canonical helper for home isolation in this test file"
  - "Retain os and strings imports in write_test.go — both are used by other tests; no import pruning needed"
metrics:
  duration: "~4 minutes"
  completed: "2026-06-22T05:39:10Z"
  tasks_completed: 3
  files_modified: 2
---

# Phase 145 Plan 01: Windows Files Test Fixes (Home-Rooting) Summary

Fix two POSIX-assumption test failures in internal/files — redirect $HOME via setHomeEnv so denylistCheck reads a home genuinely outside the sandbox root on every OS including Windows CI.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Redirect $HOME in TestHandlerUpload_FilenameSanitized | 36f61a85 | internal/files/handler_test.go |
| 2 | Redirect $HOME and drop skip guard in TestDenylist_NonHomeRootedUnaffected | 0e5b5e80 | internal/files/write_test.go |
| 3 | Local no-regression + Windows cross-compile confidence check | (verification only) | — |

## What Was Built

**handler_test.go — TestHandlerUpload_FilenameSanitized (Task 1)**

Added `fakeHome := t.TempDir()` + `setHomeEnv(t, fakeHome)` before `newHandler(t)`. On Windows, `t.TempDir()` lands under `%USERPROFILE%` (i.e. `$HOME`), so without the redirect `denylistCheck` fires and returns 403 instead of the expected 200. By pointing `$HOME` at a separate `fakeHome` directory, `denylistCheck` correctly treats the sandbox root as non-home-rooted. The two security assertions (file lands at `root/.bashrc`; nothing written to `escapeTarget`) are unchanged.

**write_test.go — TestDenylist_NonHomeRootedUnaffected (Task 2)**

Replaced the `root := t.TempDir()` + `home, _ := os.UserHomeDir()` / case-sensitive `strings.HasPrefix(root, home)` skip guard pattern with: `fakeHome := t.TempDir()`, `sandboxRoot := t.TempDir()`, `setHomeEnv(t, fakeHome)`. This makes the sandbox provably non-home-rooted on every OS without relying on OS-specific tmpdir placement or a case-sensitive prefix check. The skip guard is deleted — the assertion now always runs. The `os` and `strings` imports are retained because other tests in the file use them.

**Task 3 — verification only**

- `go test -race -short ./internal/files/`: PASS
- `GOOS=windows GOARCH=amd64 go vet ./internal/files/`: PASS
- `GOOS=windows GOARCH=amd64 go test -c ./internal/files/ -o /dev/null`: PASS
- No production `.go` files under internal/files modified.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None. The fix redirects `$HOME` in test setup only; production `denylistCheck` logic in `sandbox.go` is unchanged. `TestDenylist_HomeRooted` and `TestDenylist_CaseVariation` still assert the denylist fires for home-rooted sandboxes (confirmed in Task 2 acceptance criteria).

## Self-Check: PASSED

- `internal/files/handler_test.go` exists and modified: confirmed
- `internal/files/write_test.go` exists and modified: confirmed
- Commit 36f61a85 exists: confirmed (git log shows it)
- Commit 0e5b5e80 exists: confirmed (git log shows it)
- `go test -race -short ./internal/files/` exits 0: confirmed
- `GOOS=windows GOARCH=amd64 go vet ./internal/files/` exits 0: confirmed
- `GOOS=windows GOARCH=amd64 go test -c ./internal/files/ -o /dev/null` exits 0: confirmed
- No production `.go` files modified: confirmed (grep returned empty)
