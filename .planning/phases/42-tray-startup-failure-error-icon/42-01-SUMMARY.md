---
phase: 42-tray-startup-failure-error-icon
plan: "01"
subsystem: ui
tags: [tray, cgo, macos, go, testing]

# Dependency graph
requires:
  - phase: 41-system-tray-lifecycle
    provides: tray infrastructure (initTray, updateTray, trayIconErrorBytes, ObjC NSStatusItem null-checks)
provides:
  - refreshTrayState correctly shows error icon when daemon is unreachable at startup (trayInit=true, client=nil)
  - TestRefreshTrayStateStartupFailure unit test covering the startup-failure path
affects:
  - 43-gui-hostname-forwarding (tray state machine is stable, no further tray changes expected)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Split compound nil-guard into two separate if blocks to distinguish 'tray not ready' from 'daemon unreachable'"

key-files:
  created: []
  modified:
    - app.go
    - tray_test.go

key-decisions:
  - "Split '!a.trayInit || a.client == nil' guard into two separate if blocks — trayInit=false means tray not ready (skip), client=nil means startup failed (show error icon)"
  - "Use updateTray(nil, false) for startup-failure path — existing connected=false code path already sets trayIconErrorBytes and trayTooltip(0)"

patterns-established:
  - "refreshTrayState guard pattern: check trayInit first (readiness), then client (connectivity) — never compound them with ||"

requirements-completed:
  - TRAY-03

# Metrics
duration: 3min
completed: "2026-04-03"
---

# Phase 42 Plan 01: Tray Startup-Failure Error Icon Summary

**Split refreshTrayState nil-client guard so tray shows error icon (trayIconErrorBytes) and updated tooltip when daemon fails to start**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-04-03T04:37:00Z
- **Completed:** 2026-04-03T04:37:56Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments

- Fixed TRAY-03: `refreshTrayState` now calls `updateTray(nil, false)` when `trayInit=true` and `client=nil` (daemon startup failure)
- Error icon (`trayIconErrorBytes`) is displayed and tooltip updates to "AgentHub — no sessions" on startup failure
- No regression: `trayInit=false` path still returns early without calling `updateTray`
- Added `TestRefreshTrayStateStartupFailure` unit test; full test suite green (7 packages)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add startup-failure test and fix refreshTrayState nil-client guard** - `a793f3d` (fix)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `app.go` - Split compound guard in `refreshTrayState`; added `updateTray(nil, false)` on startup-failure path
- `tray_test.go` - Added `TestRefreshTrayStateStartupFailure` (trayInit=true, client=nil no-panic test)

## Decisions Made

- Used `updateTray(nil, false)` directly — no new helper function needed; existing `connected=false` path already handles error icon and tooltip correctly
- Test is darwin-only by inheritance (lives in `tray_test.go` alongside `tray.go` which has `//go:build darwin`)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The fix and test were straightforward — one function body change in `app.go`, one test function added to `tray_test.go`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- TRAY-03 closed. Tray now correctly reflects daemon error state on startup failure.
- Phase 43 (gui-hostname-forwarding) is unrelated to tray — no blockers.

---
*Phase: 42-tray-startup-failure-error-icon*
*Completed: 2026-04-03*
