---
phase: 51-auto-update-checker
plan: 02
subsystem: ui
tags: [go, wails, updater, background-poller, menu]

# Dependency graph
requires:
  - phase: 51-01
    provides: internal/updater package with Check(), DefaultDetect, UpdateInfo type
provides:
  - startUpdatePoller goroutine wired into App startup (both success and failure paths)
  - runUpdateCheck method emitting update:available event to frontend
  - GetLastUpdateInfo bound method for startup race avoidance
  - CheckForUpdates bound method for manual check (force=true, bypasses rate limit)
  - Help menu "Check for Updates" item triggering immediate check via goroutine
  - appInstance package-level var for menu callback access to App methods
affects: [51-03, frontend-update-banner]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "goroutine+ticker update poller following startHealthPoller/startTrayPoller pattern"
    - "startup race avoidance: GetLastUpdateInfo mirrors GetDaemonError pattern for polled startup state"
    - "appInstance package var for menu callbacks needing App methods beyond runtime context"

key-files:
  created: []
  modified:
    - app.go
    - main.go
    - go.sum

key-decisions:
  - "5-second initial delay before first update check to avoid startup race with frontend event subscription"
  - "startUpdatePoller called in both daemon-success and daemon-failure paths — update checks are daemon-independent"
  - "CheckForUpdates uses fresh context.WithTimeout(Background()) to work without a.ctx (callable before full startup)"

patterns-established:
  - "GetLastUpdateInfo: bound method returning cached state for startup race avoidance (same pattern as GetDaemonError)"
  - "checkForUpdatesCallback: menu callback calls go appInstance.Method() to avoid blocking UI thread"

requirements-completed: [UPD-01, UPD-04]

# Metrics
duration: 3min
completed: 2026-04-07
---

# Phase 51 Plan 02: App Lifecycle Wiring Summary

**Update poller wired into Wails App startup with background goroutine, Help menu item, and GetLastUpdateInfo/CheckForUpdates bound methods for frontend integration**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-07T18:37:25Z
- **Completed:** 2026-04-07T18:40:43Z
- **Tasks:** 2
- **Files modified:** 3 (app.go, main.go, go.sum)

## Accomplishments
- Background update poller starts on App startup with 5s initial delay and 1-hour ticker
- GetLastUpdateInfo bound method solves startup race (frontend can poll on mount)
- CheckForUpdates bound method enables Help menu and future frontend-triggered checks
- Help > "Check for Updates" native menu item triggers immediate goroutine check
- appInstance var enables menu callbacks to call App methods

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire update poller and bound methods into app.go** - `7197777` (feat)
2. **Task 2: Add Help menu item and appInstance to main.go** - `a9d7ba2` (feat)

## Files Created/Modified
- `app.go` - Added lastUpdate/lastUpdateMu fields, startUpdatePoller, runUpdateCheck, GetLastUpdateInfo, CheckForUpdates methods; set appInstance in startup()
- `main.go` - Added appInstance var, checkForUpdatesCallback, "Check for Updates" Help menu item
- `go.sum` - Added missing golang.org/x/exp entries needed by tailscale dependency (pre-existing issue)

## Decisions Made
- 5-second initial delay for startup poller: frontend needs time to mount and subscribe to update:available events before first check fires
- startUpdatePoller called in both daemon-success and failure paths: update checking is daemon-independent
- CheckForUpdates uses context.WithTimeout(Background()) not a.ctx: makes it safe to call any time, including before Wails fully initializes

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed missing go.sum entries for tailscale x/exp dependencies**
- **Found during:** Task 1 verification (go build)
- **Issue:** go.sum was missing golang.org/x/exp entries added when Plan 01 merged go-selfupdate (transitive via tailscale). Build failed with "missing go.sum entry" errors.
- **Fix:** Ran `go get tailscale.com/util/set@v1.96.3 tailscale.com/util/syspolicy/setting@v1.96.3` to populate missing entries
- **Files modified:** go.sum
- **Verification:** `go vet -tags dev ./...` passes; `go test ./internal/updater/...` 8/8 pass
- **Committed in:** `7197777` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required fix — pre-existing go.sum gap from Plan 01 merge prevented compilation.

## Issues Encountered
- The Wails linker error (`_OBJC_CLASS_$_UTType undefined`) is a pre-existing macOS build environment issue unrelated to this plan's changes. `go vet` and all package tests pass cleanly. This issue exists on HEAD before our changes.

## Known Stubs
None — all functionality is fully wired. Update events flow from updater.Check() through runtime.EventsEmit to the frontend.

## Next Phase Readiness
- app.go and main.go ready for Plan 03 (frontend UpdateBanner component)
- GetLastUpdateInfo and CheckForUpdates Wails bindings already exported in App.d.ts/App.js (pre-committed by parallel agent)
- update:available event shape: `{currentVersion, latestVersion, releaseURL}`

---
*Phase: 51-auto-update-checker*
*Completed: 2026-04-07*
