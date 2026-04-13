---
phase: 71-opencode-theming-fix
plan: 01
subsystem: testing
tags: [go-test, opencode, tdd, spy-backend, xterm-theme]

# Dependency graph
requires: []
provides:
  - "RED-state test stubs for OpenCode env injection (TestCreateSession_OpenCodeEnv)"
  - "RED-state test stub for managed tui.json (TestOpenCodeTUIConfig)"
  - "Fixed TestKnownCLIs_HasExpectedEntries for 5 CLIs including tailscale"
  - "spyBackend test double for capturing CreateRequest.Env"
affects: [71-02, 71-03, 71-04]

# Tech tracking
tech-stack:
  added: []
  patterns: [spy-backend for env injection testing]

key-files:
  created: []
  modified:
    - internal/daemon/engine_test.go
    - internal/pty/detect_test.go

key-decisions:
  - "spyBackend captures CreateRequest.Env without launching real PTY processes"
  - "TestOpenCodeTUIConfig uses t.TempDir() + os.ReadFile for filesystem assertion"

patterns-established:
  - "spyBackend: lightweight SessionBackend mock for testing engine env injection behavior"

requirements-completed: [THM-05]

# Metrics
duration: 2min
completed: 2026-04-13
---

# Phase 71 Plan 01: Wave 0 Test Infrastructure Summary

**RED-state test stubs for OpenCode env injection and managed tui.json, plus fix for pre-existing TestKnownCLIs 5-CLI assertion**

## Performance

- **Duration:** 2m 32s
- **Started:** 2026-04-13T19:09:02Z
- **Completed:** 2026-04-13T19:11:34Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Two new failing test functions that define the acceptance criteria for Plan 02 OpenCode theming implementation
- spyBackend test double that captures CreateRequest.Env without launching real PTY processes
- Fixed pre-existing TestKnownCLIs_HasExpectedEntries failure caused by Phase 68 adding tailscale to knownCLIs without updating the test

## Task Commits

Each task was committed atomically:

1. **Task 1: Add failing test stubs for OpenCode env injection and managed tui.json** - `cc32c5d` (test)
2. **Task 2: Fix pre-existing TestKnownCLIs_HasExpectedEntries for 5 CLIs** - `63da45a` (fix)

## Files Created/Modified
- `internal/daemon/engine_test.go` - Added spyBackend type, TestCreateSession_OpenCodeEnv (RED), TestOpenCodeTUIConfig (RED)
- `internal/pty/detect_test.go` - Updated expected CLI list from 4 to 5 entries (added tailscale)

## Decisions Made
- Used spyBackend implementing pty.SessionBackend to capture CreateRequest.Env without launching real PTY processes -- avoids goroutine leaks and process management in tests
- TestOpenCodeTUIConfig uses t.TempDir() + os.ReadFile for a simple filesystem existence assertion that compiles cleanly and fails at runtime

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- TestCreateSession_OpenCodeEnv and TestOpenCodeTUIConfig are in RED state, ready for Plan 02 to turn GREEN
- TestKnownCLIs_HasExpectedEntries passes, removing a pre-existing regression
- All existing daemon and pty tests continue to pass

## Self-Check: PASSED

- FOUND: internal/daemon/engine_test.go
- FOUND: internal/pty/detect_test.go
- FOUND: 71-01-SUMMARY.md
- FOUND: cc32c5d (Task 1 commit)
- FOUND: 63da45a (Task 2 commit)

---
*Phase: 71-opencode-theming-fix*
*Completed: 2026-04-13*
