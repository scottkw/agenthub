---
phase: 44-git-migration-to-github
plan: 01
subsystem: infra
tags: [go, modules, import-paths, github]

# Dependency graph
requires: []
provides:
  - "go.mod declares module github.com/scottkw/agenthub"
  - "All 30 import sites across 16 .go files use canonical GitHub module path"
  - "Build and race-clean test suite verified against new module path"
affects:
  - 44-02 (GitHub push plan depends on correct module path being committed)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Go module path matches canonical GitHub repo import path before first push"

key-files:
  created: []
  modified:
    - go.mod
    - app.go
    - app_test.go
    - cmd_attach.go
    - cmd_attach_test.go
    - cmd_cli.go
    - cmd_cli_test.go
    - cmd_daemon.go
    - cmd_daemon_test.go
    - main.go
    - internal/daemon/api.go
    - internal/daemon/engine.go
    - internal/relay/server.go
    - internal/status/detector.go
    - internal/status/detector_test.go
    - internal/webserver/server.go
    - internal/webserver/server_test.go
    - tray.go

key-decisions:
  - "Capture trayCallbackApp pointer before goroutine in onTrayQuit to eliminate data race"

patterns-established:
  - "Capture global pointer in caller goroutine before launching background goroutine to prevent cgo callback data races"

requirements-completed:
  - GIT-02

# Metrics
duration: 5min
completed: 2026-04-03
---

# Phase 44 Plan 01: Git Migration to GitHub — Module Path Rewrite Summary

**Go module path rewritten from `github.com/agenthub/agenthub` to `github.com/scottkw/agenthub` across go.mod and all 30 import sites in 16 .go files; race detector clean**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-04-03T20:17:32Z
- **Completed:** 2026-04-03T20:22:00Z
- **Tasks:** 1
- **Files modified:** 18

## Accomplishments

- Updated `go.mod` module declaration to `github.com/scottkw/agenthub`
- Rewrote all 30 import path occurrences across 16 .go files (zero old paths remain)
- Fixed pre-existing data race in `onTrayQuit` (capture pointer before goroutine)
- `go build ./...` passes with zero errors
- `go test -race ./...` passes across all 6 packages (200+ tests race-clean)

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite Go module path in go.mod and all imports** - `bede276` (chore)

## Files Created/Modified

- `go.mod` - module declaration updated to `github.com/scottkw/agenthub`
- `app.go` - internal imports rewritten
- `app_test.go` - internal imports rewritten
- `cmd_attach.go` - internal imports rewritten
- `cmd_attach_test.go` - internal imports rewritten
- `cmd_cli.go` - internal imports rewritten
- `cmd_cli_test.go` - internal imports rewritten
- `cmd_daemon.go` - internal imports rewritten
- `cmd_daemon_test.go` - internal imports rewritten
- `main.go` - internal imports rewritten
- `internal/daemon/api.go` - internal imports rewritten
- `internal/daemon/engine.go` - internal imports rewritten
- `internal/relay/server.go` - internal imports rewritten
- `internal/status/detector.go` - internal imports rewritten
- `internal/status/detector_test.go` - internal imports rewritten
- `internal/webserver/server.go` - internal imports rewritten
- `internal/webserver/server_test.go` - internal imports rewritten
- `tray.go` - internal imports rewritten + race fix in `onTrayQuit`

## Decisions Made

- Used `go mod edit -module` for the go.mod change and `find ... sed` for the .go files — atomic and auditable
- Fixed the `onTrayQuit` data race by capturing `trayCallbackApp` in the caller goroutine before spawning the background goroutine (correct semantics: read global in caller, use snapshot in goroutine)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed data race in onTrayQuit cgo callback**
- **Found during:** Task 1 (during `go test -race ./...` verification)
- **Issue:** `onTrayQuit()` launched a goroutine that read the `trayCallbackApp` global; `TestTrayQuitNilClient` deferred restoring the global while goroutine was still running — detected by race detector
- **Fix:** Captured `app := trayCallbackApp` in the caller goroutine before `go func()`, goroutine then uses local snapshot only
- **Files modified:** `tray.go`
- **Verification:** `go test -race ./...` passes cleanly with zero race warnings
- **Committed in:** `bede276` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - Bug)
**Impact on plan:** Fix essential for race-clean test suite. No scope creep — same behavioral semantics, just race-safe.

## Issues Encountered

None beyond the race condition documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Module path is `github.com/scottkw/agenthub` everywhere — codebase is ready for Plan 02 (GitHub push)
- Build and race-clean tests confirmed on the updated module path
- No push has occurred — Plan 02 handles the GitHub remote setup and push sequence

---
*Phase: 44-git-migration-to-github*
*Completed: 2026-04-03*
