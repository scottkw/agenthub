---
phase: 19-daemon-core-engine-ipc
plan: 02
subsystem: api
tags: [go, daemon, unix-socket, ipc, wails, delegation]

requires:
  - phase: 19-01
    provides: SessionEngine, API, DaemonClient, socket utilities

provides:
  - App struct delegating all session ops through DaemonClient over in-process Unix socket
  - App holds no authoritative session state (engine owns it all)
  - In-process daemon wiring in testApp() test helper

affects:
  - 20-daemon-process-separation
  - any phase referencing app.go session state

tech-stack:
  added: []
  patterns:
    - "App is a thin Wails binding shell — all session state lives in daemon.SessionEngine"
    - "DaemonClient delegation pattern: client.Method() for all session ops except CreateSession"
    - "CreateSession calls engine directly for onStatus callback (HTTP cannot serialize callbacks)"
    - "Short /tmp socket paths in tests to stay under macOS 103-char sun_path limit"

key-files:
  created: []
  modified:
    - app.go
    - app_test.go
    - tray_test.go

key-decisions:
  - "CreateSession calls engine.CreateSession directly (not client) because onStatus callback cannot be serialized over HTTP — this is the one intentional exception to the delegation pattern"
  - "testApp() uses /tmp/aht{pid}_{seq}.sock for socket paths — macOS t.TempDir() produces paths > 103 chars which exceed sun_path limit"
  - "tray_test.go updated to use app.engine.Registry().Len() since registry field moved into engine"

patterns-established:
  - "App thin-shell pattern: App struct has no maps or direct session state, only engine/api/client/socketPath"
  - "Phase 20 migration path: change only daemon.DefaultSocketPath() to point to out-of-process socket — App code stays identical"

requirements-completed:
  - DAEMON-02

duration: 11min
completed: 2026-03-23
---

# Phase 19 Plan 02: App Daemon Delegation Summary

**App refactored from direct PTY/registry state to thin Wails shell delegating all session ops through DaemonClient over an in-process Unix socket**

## Performance

- **Duration:** ~11 min
- **Started:** 2026-03-23T14:29:10Z
- **Completed:** 2026-03-23T14:47:19Z
- **Tasks:** 2 of 2 complete (Task 1 auto + Task 2 human-verify checkpoint approved)
- **Files modified:** 3

## Accomplishments

- Removed all direct session state from App (registry, backend, manager, tabNames, cliPaths, sessionStatuses, statusMu)
- App struct now holds engine, api, client, socketPath — delegates all session operations through DaemonClient
- startup() starts daemon API over Unix socket and wires DaemonClient
- shutdown() stops daemon API after relay/webserver cleanup
- testApp() in app_test.go starts in-process daemon on short /tmp socket path
- All 19 existing main package tests pass with -race; all 6 packages green

## Task Commits

1. **Task 1: Migrate App to delegate through DaemonClient and update tests** - `f132335` (feat)
2. **Task 2: Verify GUI behavior identical to v1.2** - checkpoint:human-verify approved (no code commit)

## Files Created/Modified

- `/Users/ken/dev/agenthub/app.go` - App struct migrated to daemon delegation pattern; all session methods delegate through a.client or a.engine
- `/Users/ken/dev/agenthub/app_test.go` - testApp() wires in-process daemon API on /tmp short socket; daemon import added
- `/Users/ken/dev/agenthub/tray_test.go` - app.registry.Len() → app.engine.Registry().Len()

## Decisions Made

- CreateSession is the one exception to the client delegation pattern: it calls engine.CreateSession directly to pass the onStatus callback that wraps runtime.EventsEmit. This cannot be serialized over HTTP.
- Test socket paths use `/tmp/aht{pid}_{seq}.sock` pattern to stay under macOS's 103-char sun_path limit. macOS t.TempDir() paths are longer than 103 chars.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed tray_test.go referencing removed app.registry field**
- **Found during:** Task 1 (post-write test run)
- **Issue:** `tray_test.go` referenced `app.registry.Len()` which no longer exists after migration
- **Fix:** Updated to `app.engine.Registry().Len()` — semantically identical, routes through engine
- **Files modified:** tray_test.go
- **Verification:** `go test -race ./...` passes
- **Committed in:** f132335 (Task 1 commit)

**2. [Rule 3 - Blocking] Fixed socket path length exceeding macOS sun_path limit in tests**
- **Found during:** Task 1 (test run after testApp update)
- **Issue:** macOS t.TempDir() returns paths like `/var/folders/...` that exceed 103 chars; api.Start() returns ValidateSocketPath error
- **Fix:** Use `/tmp/aht{pid}_{seq}.sock` pattern in testApp() — same approach as Phase 19-01 daemon tests
- **Files modified:** app_test.go
- **Verification:** All tests pass with -race
- **Committed in:** f132335 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes necessary to get tests green. No scope creep — both fixes were anticipated in the STATE.md decision log from Phase 19-01.

## Issues Encountered

None beyond the two auto-fixed deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 19 Plan 02 is complete. All automated tests pass with -race (19 app tests + 28 daemon tests). GUI verified identical to v1.2.
- Phase 20 (daemon process separation) can proceed: changing only `daemon.DefaultSocketPath()` to point to an out-of-process socket will work without App code changes — the thin-shell invariant holds.
- Remaining open item for Phase 20 planning: relay port handoff sequence (daemon → GUI) needs to be pinned with respect to Wails lifecycle hooks.

---
*Phase: 19-daemon-core-engine-ipc*
*Completed: 2026-03-23*
