---
phase: 26-graceful-gui-startup-failure
plan: 02
subsystem: ui
tags: [react, wails, typescript, event-subscription, daemon-error]

# Dependency graph
requires:
  - phase: 26-graceful-gui-startup-failure
    provides: Plan 01 Go-side startup() emitting daemon:error event via EventsEmit and RetryDaemon() bound method

provides:
  - Frontend subscribes to daemon:error event and shows error banner with actual error string
  - Retry button calls RetryDaemon() before re-running init methods
  - RetryDaemon Wails binding stubs (JS + d.ts)
  - Phase 26 daemon error handling tests in App.test.tsx

affects:
  - graceful-gui-startup-failure (completes the frontend half of the feature)
  - future frontend phases using EventsOn subscription pattern

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "EventsOn cleanup: each subscription returns an off() function, all collected and called in cleanup return"
    - "Retry with pre-flight: call RetryDaemon() before Promise.all, early-return on pre-flight failure"
    - "Dynamic banner copy: render {daemonError} state directly instead of hardcoded static strings"

key-files:
  created: []
  modified:
    - frontend/src/App.tsx
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/components/__tests__/App.test.tsx

key-decisions:
  - "Early-return pattern in retryInit: if RetryDaemon() fails, set error and return immediately rather than trying other methods that would also fail with nil client"
  - "Banner body uses {daemonError} directly — actual Go error strings (path not found, timeout) are more actionable than any hardcoded message"

patterns-established:
  - "Wails binding stubs added near the end of App.js/App.d.ts grouped by comment section"

requirements-completed: [DAEMON-05]

# Metrics
duration: 2min
completed: 2026-03-24
---

# Phase 26 Plan 02: Graceful GUI Startup Failure — Frontend Wiring Summary

**Frontend subscribes to daemon:error Wails event, retryInit calls RetryDaemon() before init methods, and banner shows actual Go error string instead of hardcoded text**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-24T21:54:32Z
- **Completed:** 2026-03-24T21:55:45Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added RetryDaemon Wails binding stubs (App.js + App.d.ts) delegating to `main.App.RetryDaemon`
- Wired `EventsOn('daemon:error')` subscription in App.tsx mount effect with cleanup unsubscribe
- Refactored retryInit to call `await RetryDaemon()` first, early-returning on failure before Promise.all
- Replaced hardcoded banner message with `{daemonError}` for actionable Go error strings
- Added Phase 26 daemon error handling test suite (5 new tests, 126 total passing)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add RetryDaemon Wails binding stubs** - `b666779` (feat)
2. **Task 2: Wire daemon:error event subscription, retry logic, and banner copy** - `603aca0` (feat)

## Files Created/Modified
- `frontend/src/wailsjs/go/main/App.js` - Added RetryDaemon export stub
- `frontend/src/wailsjs/go/main/App.d.ts` - Added RetryDaemon type declaration
- `frontend/src/App.tsx` - Import, event subscription, retryInit pre-flight, dynamic banner copy
- `frontend/src/components/__tests__/App.test.tsx` - Phase 26 daemon error handling describe block

## Decisions Made
- Early-return pattern in retryInit: RetryDaemon() failing means the daemon client is still nil, so all subsequent calls (GetRelayPort, ListSessions, etc.) would also fail. Early-return avoids redundant errors.
- Banner uses `{daemonError}` directly — Go error strings like "daemon binary not found at /path/to/agenthub" or "daemon did not respond within 3s" are more actionable than any generic message.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Frontend and Go backend are both wired: startup() emits daemon:error, frontend catches it and shows the error banner with the actual error message, retry button re-spawns daemon before re-running init
- Phase 26 graceful GUI startup failure feature is complete end-to-end
- 126 frontend tests passing

---
*Phase: 26-graceful-gui-startup-failure*
*Completed: 2026-03-24*
