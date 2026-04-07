---
phase: 54-tailscale-onboarding-enhancement
plan: 01
subsystem: api
tags: [go, wails, tailscale, homebrew, events]

# Dependency graph
requires: []
provides:
  - AutoInstallTailscale() method in app.go with findBrew helper
  - tailscale:install:progress and tailscale:install:done Wails events
  - Wails JS/TS bindings for AutoInstallTailscale
  - Go test for AutoInstallTailscale and findBrew
affects: [54-02]

# Tech tracking
tech-stack:
  added: [bufio (stdlib), os/exec (stdlib), goruntime alias for stdlib runtime]
  patterns: [goruntime alias to avoid collision with wails runtime import, goroutine-based streaming via bufio.Scanner + EventsEmit]

key-files:
  created: []
  modified:
    - app.go
    - app_test.go
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js

key-decisions:
  - "goruntime alias for stdlib runtime avoids collision with wails/v2/pkg/runtime already imported as runtime"
  - "cmd.Stderr = cmd.Stdout merges stderr into stdout pipe to stream all brew output via single goroutine"

patterns-established:
  - "Goroutine-based streaming: bufio.Scanner on StdoutPipe emitting per-line EventsEmit events before cmd.Wait"

requirements-completed: [TS-02]

# Metrics
duration: 2min
completed: 2026-04-07
---

# Phase 54 Plan 01: AutoInstallTailscale Backend Method Summary

**`brew install --cask tailscale-app` Go backend method with findBrew path detection and streaming Wails events for per-line progress and done notification**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-04-07T22:06:30Z
- **Completed:** 2026-04-07T22:08:11Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added `findBrew()` helper resolving `/opt/homebrew/bin/brew` (Apple Silicon) or `/usr/local/bin/brew` (Intel)
- Added `AutoInstallTailscale()` method on `*App` with non-darwin guard, goroutine-based streaming via `bufio.Scanner`, and `tailscale:install:progress` / `tailscale:install:done` Wails events
- Added `TestAutoInstallTailscale` with `findBrew resolves a path on macOS` subtest — tests pass
- Updated Wails JS/TS bindings to expose `AutoInstallTailscale` to frontend

## Task Commits

Each task was committed atomically:

1. **Task 1: Add AutoInstallTailscale method and findBrew helper to app.go** - `33f9535` (feat)
2. **Task 2: Add Go test and update Wails bindings** - `f2c98b2` (feat)

## Files Created/Modified
- `app.go` - Added `findBrew()` helper and `AutoInstallTailscale()` method; added `bufio`, `os/exec`, `goruntime "runtime"` imports
- `app_test.go` - Added `TestAutoInstallTailscale` with two subtests; added `goruntime "runtime"` import
- `frontend/src/wailsjs/go/main/App.d.ts` - Added `AutoInstallTailscale(): Promise<void>` TypeScript binding
- `frontend/src/wailsjs/go/main/App.js` - Added `AutoInstallTailscale` JavaScript binding

## Decisions Made
- Used `goruntime "runtime"` alias to avoid collision with `github.com/wailsapp/wails/v2/pkg/runtime` already imported as `runtime`
- Used `cmd.Stderr = cmd.Stdout` to merge stderr into the stdout pipe so a single goroutine streams all brew output

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `AutoInstallTailscale` backend method is ready for Plan 02 frontend wiring (TS-02 "Try Auto-Install" button in the Tailscale health modal)
- No blockers

---
*Phase: 54-tailscale-onboarding-enhancement*
*Completed: 2026-04-07*
