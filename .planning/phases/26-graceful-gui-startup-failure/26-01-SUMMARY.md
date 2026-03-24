---
phase: 26-graceful-gui-startup-failure
plan: 01
subsystem: app
tags: [go, wails, daemon, error-handling, startup, nil-safety]

requires:
  - phase: 25-windows-named-pipe-fix
    provides: Windows-compatible daemon socket dial; DaemonClient used here

provides:
  - Graceful startup failure: startup() emits daemon:error event instead of panicking
  - RetryDaemon() bound method for frontend retry button
  - daemonErr field on App struct for tracking startup failure state
  - Nil-safety guards on all 13 client-calling bound methods

affects:
  - 26-02 (frontend retry wiring — must call RetryDaemon before other bound methods)
  - Any phase that adds new Wails-bound methods (must add nil guard)

tech-stack:
  added: []
  patterns:
    - "Graceful startup: store error in struct, emit event, return early (no panic)"
    - "Nil guard at top of every bound method that calls a.client"
    - "RetryDaemon re-initialises client + tray + health poller idempotently"
    - "TDD red-green: compile-fail tests committed before implementation"

key-files:
  created: []
  modified:
    - app.go
    - app_test.go

key-decisions:
  - "Emit daemon:error event AND store daemonErr field — event covers real-time notification, field covers polling via GetDaemonError (future)"
  - "RetryDaemon uses DefaultSocketPath() internally — no path injection needed for production; tests override HOME to force failure"
  - "Nil guards return zero-value/error same as existing error paths — no new failure modes introduced"
  - "Override HOME=/nonexistent-dir in TestRetryDaemonFail to force EnsureDaemon timeout — avoids relying on no daemon being present"

patterns-established:
  - "All new bound methods that call a.client must start with: if a.client == nil { return <zero>, fmt.Errorf(\"daemon not connected\") }"
  - "startup() failure path: set a.daemonErr, emit event, return — never call initTray or startHealthPoller"

requirements-completed:
  - DAEMON-05

duration: 15min
completed: 2026-03-24
---

# Phase 26 Plan 01: Graceful Startup Failure Summary

**Replaced `panic()` in `startup()` with graceful error handling — stores error in `daemonErr`, emits `daemon:error` Wails event, adds `RetryDaemon()` bound method, and nil-guards all 13 client-calling methods.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-03-24T22:00:00Z
- **Completed:** 2026-03-24T22:15:00Z
- **Tasks:** 1 (TDD: RED + GREEN)
- **Files modified:** 2

## Accomplishments

- Replaced `panic(fmt.Sprintf("agenthub: ensure daemon: %v", err))` with graceful error storage and `daemon:error` event emission
- Added `daemonErr error` field to App struct
- Added `RetryDaemon() error` Wails-bound method that re-attempts daemon spawn
- Added nil-safety guards to all 13 bound methods calling `a.client`
- Added 6 new tests: 5 nil-client guards + 1 RetryDaemon failure path

## Task Commits

Each task was committed atomically (TDD pattern):

1. **RED - Failing tests** - `0433f8b` (test)
2. **GREEN - Implementation** - `c652d3e` (feat)

## Files Created/Modified

- `/Users/ken/dev/agenthub/.claude/worktrees/agent-ac21120c/app.go` - Added `daemonErr` field, graceful startup, `RetryDaemon()`, 13 nil guards
- `/Users/ken/dev/agenthub/.claude/worktrees/agent-ac21120c/app_test.go` - Added `testAppNoDaemon` helper, 6 new tests

## Decisions Made

- Used `HOME=/nonexistent-test-dir-that-cannot-exist` in `TestRetryDaemonFail` to force reliable test failure regardless of whether a real daemon is running locally. The 3-second EnsureDaemon timeout is the natural failure mode.
- Nil guards return the same zero/error values as existing error paths — no breaking changes to the external API contract.
- `RetryDaemon()` guards tray initialisation with `if !a.trayInit` to prevent double health pollers (Pitfall 4 from research).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestRetryDaemonFail needed HOME override to force reliable failure**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** The test environment had a live daemon running at `DefaultSocketPath()`, so `RetryDaemon()` succeeded when it should have failed
- **Fix:** Added `t.Setenv("HOME", "/nonexistent-test-dir-that-cannot-exist")` to prevent socket creation and force EnsureDaemon timeout
- **Files modified:** app_test.go
- **Verification:** Test now reliably fails in 3 seconds (EnsureDaemon timeout)
- **Committed in:** c652d3e (implementation commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug in test approach)
**Impact on plan:** Test approach adjusted to account for live daemon in dev environment. No scope creep.

## Issues Encountered

- The worktree branch was forked before the daemon architecture refactoring. Required a `git merge main` at the start to bring the branch to the correct baseline state before implementing plan changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Go backend changes complete and tested
- Frontend (26-02) can now call `RetryDaemon()` via the Wails binding (JS stub already added in `feat(26-02)`)
- All nil guards in place so frontend retry flow won't cause nil panics

---
*Phase: 26-graceful-gui-startup-failure*
*Completed: 2026-03-24*
