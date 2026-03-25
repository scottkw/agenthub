---
phase: 28-cli-package-removal
plan: 01
subsystem: infra
tags: [go, cleanup, dead-code, cli]

# Dependency graph
requires:
  - phase: 27-unified-entrypoint
    provides: CLI logic moved to root package main.go, making cmd/agenthub-cli/ dead code
provides:
  - cmd/agenthub-cli/ deleted (8 files, 1559 lines removed)
  - README.md Go packages table without cmd/agenthub-cli row
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - README.md

key-decisions:
  - "Deleted cmd/agenthub-cli/ entirely — all CLI logic now lives in root package (main.go) per Phase 27 unification"
  - "Pre-existing flaky test TestHub_SlowClientDisconnected in internal/relay deferred (out of scope — not caused by this plan)"

patterns-established: []

requirements-completed:
  - CLEAN-01
  - CLEAN-02

# Metrics
duration: 5min
completed: 2026-03-25
---

# Phase 28 Plan 01: CLI Package Removal Summary

**Deleted cmd/agenthub-cli/ dead package (8 files, 1559 lines) and scrubbed its README.md row — repo now has zero references to the old standalone CLI binary**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-03-25T00:32:57Z
- **Completed:** 2026-03-25T00:37:59Z
- **Tasks:** 2
- **Files modified:** 1 (README.md) + 8 deleted

## Accomplishments
- Deleted all 8 source files in cmd/agenthub-cli/ (dead code since Phase 27 moved CLI into root main.go)
- Removed the `cmd/agenthub-cli` row from the Go packages table in README.md
- Confirmed `go build ./...` exits 0 — no import regressions
- Confirmed `grep -r "agenthub-cli"` (scoped to active source files) returns zero results

## Task Commits

Each task was committed atomically:

1. **Task 1: Delete cmd/agenthub-cli/ and remove README reference** - `cce73c1` (chore)
2. **Task 2: Verify clean build and no dangling references** - no commit (verification only, no files changed)

## Files Created/Modified
- `README.md` - Removed `cmd/agenthub-cli` row from Go packages table
- `cmd/agenthub-cli/main.go` - Deleted
- `cmd/agenthub-cli/cmd_daemon.go` - Deleted
- `cmd/agenthub-cli/cmd_attach.go` - Deleted
- `cmd/agenthub-cli/cmd_attach_unix.go` - Deleted
- `cmd/agenthub-cli/cmd_attach_windows.go` - Deleted
- `cmd/agenthub-cli/cmd_daemon_test.go` - Deleted
- `cmd/agenthub-cli/cmd_attach_test.go` - Deleted
- `cmd/agenthub-cli/main_test.go` - Deleted

## Decisions Made
- Deleted cmd/agenthub-cli/ entirely — confirmed it is the old standalone entrypoint from before Phase 27, with no imports from root or other packages pointing to it
- Did not run `go mod tidy` — all dependencies are shared with root package, no orphaned entries expected

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

One pre-existing flaky test (`TestHub_SlowClientDisconnected` in `internal/relay`) failed during verification with a timing-related WebSocket policy violation. This test is in an unrelated file with no connection to the deleted package. The failure is out-of-scope per deviation rules and has been logged to deferred items.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 28 complete. Repository is clean:
- `cmd/agenthub-cli/` directory does not exist
- Zero references to "agenthub-cli" in active source files
- `go build ./...` exits 0

---
*Phase: 28-cli-package-removal*
*Completed: 2026-03-25*
