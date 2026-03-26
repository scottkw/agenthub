---
phase: 32-daemon-startup-performance
plan: 01
subsystem: daemon
tags: [go, polling, performance, pty, session-status]

# Dependency graph
requires:
  - phase: 20-process-separation
    provides: pollSessionStatus goroutine pattern (app.go)
provides:
  - pollSessionStatus with poll-first, 500ms-sleep-after pattern (eliminates 2s startup delay)
  - Two regression tests proving immediate first poll and HTTP-error exit
affects: [daemon-startup-performance, session-status-display]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Poll-first, sleep-after loop: call API first, then sleep between iterations — not sleep first"

key-files:
  created: []
  modified:
    - app.go
    - app_test.go

key-decisions:
  - "Moved time.Sleep from before GetSessionStatus to after it — poll-first is the correct pattern for responsive status display"
  - "Reduced interval from 2s to 500ms — faster status updates with no material overhead"
  - "TestPollSessionStatus_StopsOnHTTPError uses a dead socket (no listener) to test the error-exit path — simpler than stopping a live server due to HTTP transport keepalive behavior"

patterns-established:
  - "TDD with poll-first: RED test hangs on StopsOnHTTPError because buggy code loops for 60s; GREEN exits immediately on connection error"

requirements-completed:
  - PERF-01
  - PERF-02

# Metrics
duration: 15min
completed: 2026-03-26
---

# Phase 32 Plan 01: Fix pollSessionStatus Timing Summary

**Eliminated 2-second session status startup delay by restructuring pollSessionStatus to poll immediately then sleep 500ms between iterations (was sleep-2s-first)**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-03-26T05:20:00Z
- **Completed:** 2026-03-26T05:33:54Z
- **Tasks:** 1 (TDD: RED + GREEN)
- **Files modified:** 2

## Accomplishments

- Fixed `pollSessionStatus` in `app.go` — first HTTP call now fires immediately, not after 2-second sleep
- Reduced poll interval from 2 seconds to 500 milliseconds
- Added `TestPollSessionStatus_ImmediateFirstCall` — proves session status is available within 200ms
- Added `TestPollSessionStatus_StopsOnHTTPError` — proves function exits promptly when daemon is unreachable
- Full test suite passes (194+ tests, 6 packages, no regressions)

## Task Commits

Each task was committed atomically:

1. **Task 1: TDD — Fix pollSessionStatus timing (RED then GREEN)** - `a803a79` (feat)

## Files Created/Modified

- `/Users/ken/dev/agenthub/app.go` - pollSessionStatus restructured: sleep moved after GetSessionStatus, interval reduced to 500ms
- `/Users/ken/dev/agenthub/app_test.go` - Two new tests: TestPollSessionStatus_ImmediateFirstCall and TestPollSessionStatus_StopsOnHTTPError

## Decisions Made

- **Poll-first pattern**: `GetSessionStatus` → check result → `time.Sleep(500ms)` instead of `time.Sleep(2s)` → `GetSessionStatus`. This is the canonical approach for responsive status display.
- **Test for HTTP error, not "errored" status**: The plan proposed `TestPollSessionStatus_StopsOnErrored` using `KillSession` to trigger "errored" status. This doesn't work — `KillSession` deletes the status map entry, causing `GetSessionStatus` to return "running" (conservative default), not "errored". The "errored" status is only ever set by the PTY output heuristic detector. Redesigned to use a dead socket path (connection refused) which exercises the `return on error` path cleanly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Redesigned TestPollSessionStatus_StopsOnErrored test**
- **Found during:** Task 1 (GREEN phase, test verification)
- **Issue:** Plan's test design assumed `KillSession` transitions session to "errored" status. In reality, `KillSession` deletes the status map entry; `GetSessionStatus` returns "running" (not-found default) for killed sessions. The "errored" status is only produced by the PTY output heuristic detector, not by kill. The test hung for 60s.
- **Fix:** Renamed test to `TestPollSessionStatus_StopsOnHTTPError`. Points DaemonClient at a dead socket path (no listener) so every HTTP call fails immediately with connection refused, exercising the `return on err` path.
- **Files modified:** app_test.go
- **Verification:** Test completes in <3s (actually <10ms); full suite passes
- **Committed in:** a803a79 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - Bug in test design)
**Impact on plan:** Test exercises the same correctness property (pollSessionStatus exits promptly on failure) via a reliable mechanism. Plan's core goal — poll-first pattern in app.go — executed exactly as specified.

## Issues Encountered

The HTTP transport's keepalive connection pool prevented the "stop daemon mid-poll" test approach from working — existing TCP connections to the stopped server remain alive in the transport pool, so subsequent requests succeed until the connection is eventually closed by the OS. Used a dead socket path instead to reliably trigger immediate connection failure.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `pollSessionStatus` fix is complete. Sessions now get their first status event within ~100ms of creation (previously 2s minimum).
- Phase 32 Plan 02 can proceed: it addresses the remaining daemon startup items.
- No blockers.

## Self-Check: PASSED

- app.go: FOUND
- app_test.go: FOUND
- SUMMARY.md: FOUND
- commit a803a79: FOUND

---
*Phase: 32-daemon-startup-performance*
*Completed: 2026-03-26*
