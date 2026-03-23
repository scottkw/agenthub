---
phase: 14-tailscale-health-check-infrastructure
plan: 02
subsystem: app
tags: [tailscale, go, wails, health-check, goroutine, events]

# Dependency graph
requires:
  - 14-01 (TailscaleHealth struct and CheckHealth function)
provides:
  - GetTailscaleStatus() Wails-bound method on App
  - startHealthPoller() background goroutine emitting tailscale:health events
affects:
  - phase-18 (frontend receives live Tailscale health updates via tailscale:health event)
  - phase-15 (TLS binding can use GetTailscaleStatus for pre-flight checks)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Background goroutine polling with time.NewTicker and context cancellation for clean shutdown
    - EventsEmit guard pattern: a.ctx != nil && a.ctx.Value("frontend") != nil
    - On-demand method wraps long-running check with 5-second context timeout
    - State-change detection via struct equality before emitting events (avoids event storms)

key-files:
  created: []
  modified:
    - app.go (GetTailscaleStatus method, startHealthPoller goroutine, startup() call, time import)
    - app_test.go (TestGetTailscaleStatus and TestHealthPollerStops)

key-decisions:
  - "startHealthPoller uses ctx.Done() select case for clean goroutine exit on Wails shutdown"
  - "Struct equality (h != last) prevents event storms when Tailscale state is stable"
  - "EventsEmit guard reuses existing app.go pattern to avoid Wails panic outside event loop"
  - "GetTailscaleStatus uses context.Background() (not a.ctx) so it remains callable before Wails startup"

patterns-established:
  - "Poller goroutine pattern: ticker + ctx.Done() select; defer ticker.Stop(); state diff before emit"

requirements-completed: [HEALTH-06]

# Metrics
duration: 10min
completed: 2026-03-20
---

# Phase 14 Plan 02: Wails App Layer Integration — GetTailscaleStatus and startHealthPoller

**GetTailscaleStatus() Wails-bound method and startHealthPoller() background goroutine wired into app.go startup — both tests pass with race detector, full suite green**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-03-20T18:57:00Z
- **Completed:** 2026-03-20T19:07:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `GetTailscaleStatus()` Wails-bound method: wraps `webserver.CheckHealth` with a 5-second context timeout; callable on-demand from the frontend without goroutine overhead
- `startHealthPoller()` background goroutine: polls every 10 seconds, emits `"tailscale:health"` events only when state changes, exits cleanly when Wails context is cancelled
- EventsEmit guard applied (`a.ctx != nil && a.ctx.Value("frontend") != nil`) — matches existing pattern in `CreateSession` and `KillSession`; prevents runtime panic in test environments
- `a.startHealthPoller(ctx)` called at end of `startup()` after tray initialisation
- `TestGetTailscaleStatus`: exercises real `CheckHealth` path without tailscaled — no panic, all 5 struct fields accessible
- `TestHealthPollerStops`: context cancellation test passes with `-race` detector — no goroutine leak

## Task Commits

Each task was committed atomically:

1. **Task 1: Add GetTailscaleStatus method and startHealthPoller to app.go** - `4051314` (feat)
2. **Task 2: Add tests for GetTailscaleStatus and health poller shutdown** - `5ff2aa1` (test)

## Files Created/Modified

- `app.go` - Added `GetTailscaleStatus()`, `startHealthPoller()`, `a.startHealthPoller(ctx)` in `startup()`, `"time"` import
- `app_test.go` - Added `TestGetTailscaleStatus` and `TestHealthPollerStops`

## Decisions Made

- `GetTailscaleStatus` uses `context.Background()` as the timeout base (not `a.ctx`) so it remains callable even if invoked before Wails fully initialises.
- Struct equality `h != last` is the state-change gate for `EventsEmit`. Since `TailscaleHealth` contains only comparable types (bool, string), this is safe and avoids event storms during stable network conditions.
- `startHealthPoller` receives `ctx` (the Wails app context) rather than creating its own — ensures the goroutine lifecycle is bound to the Wails app lifecycle, not to an independent context.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Pre-existing `internal/relay` test suite was green on this run (the `TestHub_SlowClientDisconnected` flakiness noted in plan 01 did not manifest).

## User Setup Required

None — frontend integration (Phase 18) is the next consumer; no configuration needed at this layer.

## Self-Check: PASSED

- `app.go` — FOUND: `func (a *App) GetTailscaleStatus()` at line 538
- `app.go` — FOUND: `func (a *App) startHealthPoller()` at line 547
- `app.go` — FOUND: `a.startHealthPoller(ctx)` at line 86
- `app_test.go` — FOUND: `func TestGetTailscaleStatus(t *testing.T)`
- `app_test.go` — FOUND: `func TestHealthPollerStops(t *testing.T)`
- Task 1 commit `4051314` — FOUND
- Task 2 commit `5ff2aa1` — FOUND
- `go test . -run "TestGetTailscaleStatus|TestHealthPollerStops" -v -race` — PASS
- `go test ./...` — all packages PASS

## Next Phase Readiness

- `GetTailscaleStatus()` is registered as a Wails-bound method (added to app struct, exposed in main.go via `app.GetTailscaleStatus`)
- `"tailscale:health"` event is live and emitted from the Go layer — frontend (Phase 18) can subscribe with `EventsOn("tailscale:health", callback)`
- Completes HEALTH-06 requirement

---
*Phase: 14-tailscale-health-check-infrastructure*
*Completed: 2026-03-20*
