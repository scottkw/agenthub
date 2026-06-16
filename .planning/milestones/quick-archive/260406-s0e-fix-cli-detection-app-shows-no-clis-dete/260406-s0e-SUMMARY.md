---
status: complete
completed: 2026-04-06
commit: 56ccc97
phase: quick
plan: 260406-s0e
subsystem: daemon/gui
tags: [cli-detection, path, gui, daemon, fix]
dependency_graph:
  requires: []
  provides: [exported-augment-path]
  affects: [gui-cli-detection, daemon-startup]
tech_stack:
  added: []
  patterns: [exported-function-shared-between-daemon-and-gui]
key_files:
  created: []
  modified:
    - internal/daemon/path.go
    - internal/daemon/path_test.go
    - internal/daemon/process.go
    - main.go
decisions:
  - Export augmentServicePath as AugmentServicePath so both daemon and GUI process can call it
  - Update test file (same package) to use exported name to keep go vet passing
metrics:
  duration: "~5 minutes"
  completed: "2026-04-07"
  tasks_completed: 1
  files_modified: 4
---

# Quick Task 260406-s0e: Fix CLI Detection (App Shows No CLIs Detected) Summary

**One-liner:** Export daemon's PATH augmentation function and call it at GUI startup so Finder/Dock launches find CLIs in Homebrew, volta, nvm, and /usr/local/bin.

## What Was Done

When macOS launches an app from Finder or Dock, it provides a minimal PATH (`/usr/bin:/bin:/usr/sbin:/sbin`). This caused `exec.LookPath` to fail to find CLIs installed in user directories, resulting in the "No CLIs detected" UI state.

The daemon already had `augmentServicePath()` that prepends well-known user tool directories to PATH. This fix exports that function (`AugmentServicePath`) and calls it at the very start of `runGUI()` — before `NewApp()` — so the GUI process also benefits from the augmented PATH.

## Changes

| File | Change |
|------|--------|
| `internal/daemon/path.go` | Renamed `augmentServicePath` -> `AugmentServicePath` (exported); updated godoc to mention GUI usage |
| `internal/daemon/process.go` | Updated call from `augmentServicePath()` -> `AugmentServicePath()` (no behavior change) |
| `main.go` | Added `daemon.AugmentServicePath()` as first line of `runGUI()` |
| `internal/daemon/path_test.go` | Updated test calls from `augmentServicePath()` -> `AugmentServicePath()` (Rule 1 auto-fix) |

## Verification

- `go build -tags wailsassets -o /dev/null .` — passes with no errors
- `go vet ./...` — passes with no errors
- Grep confirms function exported and called in both process.go and main.go

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated path_test.go to use exported function name**
- **Found during:** Task 1 verification (`go vet ./...`)
- **Issue:** `path_test.go` is in `package daemon` (same package, not `_test`), so it called `augmentServicePath()` directly. Renaming to `AugmentServicePath` broke the test file, causing `go vet` to fail with `undefined: augmentServicePath`.
- **Fix:** Replaced all 3 occurrences of `augmentServicePath()` in `path_test.go` with `AugmentServicePath()`. No behavior change — tests still exercise the same code path.
- **Files modified:** `internal/daemon/path_test.go`
- **Commit:** eb90fa6

## Known Stubs

None.

## Self-Check: PASSED

- `internal/daemon/path.go` exists and contains `func AugmentServicePath()`
- `internal/daemon/process.go` calls `AugmentServicePath()`
- `main.go` calls `daemon.AugmentServicePath()` at start of `runGUI()`
- Commit eb90fa6 exists
