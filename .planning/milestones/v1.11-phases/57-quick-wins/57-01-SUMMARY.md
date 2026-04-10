---
phase: 57-quick-wins
plan: 01
subsystem: daemon
tags: [go, path, cli-detection, augment-service-path]

# Dependency graph
requires: []
provides:
  - "~/.local/bin added to AugmentServicePath candidates (first in list)"
  - "TestAugmentServicePath_AddsLocalBin test in path_test.go"
affects: [daemon-startup, cli-detection, session-creation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "New path candidates prepended at front of candidates slice for highest precedence"

key-files:
  created: []
  modified:
    - internal/daemon/path.go
    - internal/daemon/path_test.go

key-decisions:
  - "~/.local/bin placed as FIRST candidate so Anthropic native installer binary takes precedence over volta/homebrew paths"

patterns-established:
  - "Add new path candidates as first entry to give highest lookup precedence"

requirements-completed:
  - DET-01

# Metrics
duration: 8min
completed: 2026-04-08
---

# Phase 57 Plan 01: Quick Wins Summary

**~/.local/bin added as first AugmentServicePath candidate so Claude Code installed via Anthropic native installer is discoverable by exec.LookPath at daemon/GUI startup**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-08T18:30:00Z
- **Completed:** 2026-04-08T18:38:00Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments

- Added `filepath.Join(home, ".local", "bin")` as first entry in AugmentServicePath candidates slice
- Added comment `// Anthropic native installer (macOS/Linux)` to document the intent
- Added `TestAugmentServicePath_AddsLocalBin` test verifying new candidate is prepended when directory exists
- All 57 existing daemon tests pass without regressions

## Task Commits

1. **Task 1: Add ~/.local/bin to AugmentServicePath candidates and test** - `3bba922` (feat)

**Plan metadata:** (docs commit below)

_Note: Task 1 used TDD: RED (test added, ran failing) → GREEN (implementation added, ran passing)_

## Files Created/Modified

- `internal/daemon/path.go` - Added `filepath.Join(home, ".local", "bin")` as first candidate in AugmentServicePath
- `internal/daemon/path_test.go` - Added TestAugmentServicePath_AddsLocalBin test

## Decisions Made

- Placed `~/.local/bin` as FIRST candidate (before `.volta/bin`) to give Anthropic installer binary highest priority in PATH lookup
- Used `// Anthropic native installer (macOS/Linux)` comment for clarity on Windows note (Windows support TBD but code is safe)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Path fix is committed and tested; daemon will find `~/.local/bin/claude` on next startup
- Plan 57-02 can proceed independently

## Self-Check: PASSED

- `internal/daemon/path.go` exists with new candidate: FOUND
- `internal/daemon/path_test.go` exists with TestAugmentServicePath_AddsLocalBin: FOUND
- Commit `3bba922` exists: FOUND
- All daemon tests pass: CONFIRMED (57 PASS, 0 FAIL)

---
*Phase: 57-quick-wins*
*Completed: 2026-04-08*
