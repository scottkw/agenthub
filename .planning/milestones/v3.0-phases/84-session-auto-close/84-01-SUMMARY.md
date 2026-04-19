---
phase: 84-session-auto-close
plan: "01"
subsystem: backend-exit-detection
tags: [session, exit-detection, auto-close, wails-events, web-grace-period]
dependency_graph:
  requires: []
  provides:
    - session-exit-detection-pipeline
    - auto-close-session-setting
    - web-grace-period-D12
  affects:
    - internal/pty/session.go
    - internal/daemon/engine.go
    - internal/daemon/types.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - app.go
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts
tech_stack:
  added: []
  patterns:
    - hub.Done() channel used as PTY-exit signal
    - onExit callback pattern (same shape as onStatus)
    - time.AfterFunc for non-blocking grace period timer
    - pointer-typed settings fields (*bool) for nil=default semantics
key_files:
  created: []
  modified:
    - internal/pty/session.go
    - internal/daemon/types.go
    - internal/daemon/engine.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - internal/daemon/engine_test.go
    - app.go
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts
decisions:
  - "Used Session.SetState() helper to transition state from daemon package (mu is unexported from pty package)"
  - "Used simpler unconditional DisableSession in onExit callback — no IsSessionEnabled needed; DisableSession is a no-op for non-enabled sessions"
  - "pollSessionStatus now polls ListSessions instead of GetSessionStatus to detect State=stopped, extending deadline to 5min for long-running agents"
metrics:
  duration: "~20 minutes"
  completed: "2026-04-19"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 9
---

# Phase 84 Plan 01: Exit Detection Infrastructure Summary

Backend exit detection pipeline and auto-close settings: PTY EOF triggers session state transition to "stopped", daemon captures exit code, GUI detects via polling and emits `session:exit` Wails event with full payload, auto-close-session setting persisted via settings.json with getter/setter/API/client/Wails bindings, 10-second web serving grace period on natural exit (D-12).

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Exit detection infrastructure | fa91dc5 | session.go, types.go, engine.go, api.go, client.go, engine_test.go |
| 2 | GUI layer — session:exit event, auto-close bindings | 2786e92 | app.go, App.js, App.d.ts |

## What Was Built

### Task 1: Backend Exit Detection Pipeline

**internal/pty/session.go:**
- `Session.SetState(state)` — thread-safe state transition via internal mutex
- `Session.WaitForExit() int` — blocks until process exits, returns exit code (0 for nil ProcessState)
- `Session.ExitCode() int` — returns current exit code or -1 if still running

**internal/daemon/types.go:**
- `SessionInfo.ExitCode *int` — nil while running, set when stopped; pointer allows 0 as valid code
- `SessionInfo.Duration *int` — seconds since CreatedAt; set when stopped

**internal/daemon/engine.go:**
- `SessionEngine.autoCloseSession *bool` field with nil=true default semantics
- `daemonSettings.AutoCloseSession *bool` for persistence
- `GetAutoCloseSession() bool` — returns true when nil (D-11)
- `SetAutoCloseSession(val bool)` — persists to settings.json
- `CreateSession` signature extended: `onExit func(string, int)` parameter added
- Exit watcher goroutine: `<-hub.Done()` → `WaitForExit()` → `SetState(StateStopped)` → `onExit()`
- `ListSessions` now populates ExitCode/Duration for stopped sessions

**internal/daemon/api.go:**
- `GET /settings/auto-close-session` and `PATCH /settings/auto-close-session` routes
- `handleCreateSession` passes `onExit` callback with `time.AfterFunc(10s, DisableSession)` (D-12)

**internal/daemon/client.go:**
- `GetAutoCloseSession() (bool, error)` — GET /settings/auto-close-session
- `SetAutoCloseSession(val bool) error` — PATCH /settings/auto-close-session

### Task 2: GUI Layer

**app.go:**
- `SessionInfo` struct extended with `Status string` field
- `ListSessions` copies `Status` from daemon.SessionInfo
- `pollSessionStatus` rewritten: polls `ListSessions()` instead of `GetSessionStatus()`, detects `State=="stopped"`, emits `session:exit`
- `emitExitEvent(sessionID, daemon.SessionInfo)` — emits `session:exit` with sessionId, exitCode, sessionName, cli, duration, finalStatus
- `GetAutoCloseSession() bool` — Wails binding with true default on error
- `SetAutoCloseSession(val bool) error` — Wails binding

**frontend/src/wailsjs/go/main/App.js:**
- `GetAutoCloseSession` and `SetAutoCloseSession` stubs added

**frontend/src/wailsjs/go/main/App.d.ts:**
- `GetAutoCloseSession(): Promise<boolean>` and `SetAutoCloseSession(val: boolean): Promise<void>` declarations added

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Session.mu is unexported — cannot be accessed from daemon package**
- **Found during:** Task 1 compile
- **Issue:** Plan specified `sess.mu.Lock(); sess.State = pty.StateStopped; sess.mu.Unlock()` directly in the daemon package's exit goroutine, but `mu` is an unexported field in `internal/pty`
- **Fix:** Added `Session.SetState(state SessionState)` method to `internal/pty/session.go` that acquires the internal mutex. The exit watcher calls `sess.SetState(pty.StateStopped)` instead.
- **Files modified:** internal/pty/session.go
- **Commit:** fa91dc5

**2. [Rule 3 - Blocking] engine_test.go call sites needed new onExit nil parameter**
- **Found during:** Task 1 compile
- **Issue:** 18 `CreateSession` call sites in engine_test.go used the old 8-parameter signature; the new signature has 9 parameters
- **Fix:** Updated all call sites to pass `nil` for the `onExit` parameter
- **Files modified:** internal/daemon/engine_test.go
- **Commit:** fa91dc5

**3. [Rule 2 - Missing functionality] Used simpler onExit callback (no IsSessionEnabled check)**
- **Found during:** Task 1, api.go
- **Issue:** Plan offered two options: check `IsSessionEnabled` first (preferred) or use unconditional `DisableSession`. Checked webserver package — `IsSessionEnabled` does exist on `WebServer`, but using the unconditional form is cleaner and equally correct since `DisableSession` is a no-op for never-enabled sessions.
- **Fix:** Used the simpler unconditional `time.AfterFunc(10s, DisableSession)` form as specified in the plan's fallback option.
- **Commit:** fa91dc5

## Known Stubs

None — all fields are wired to real data sources. `ExitCode` and `Duration` are populated from actual process state in `ListSessions`. `session:exit` event payload uses live daemon data.

## Threat Surface Scan

No new threat surface beyond what the plan's threat model covers. All new routes (`/settings/auto-close-session`) follow the same localhost-only Unix socket pattern as existing settings routes. The `session:exit` event payload contains only data already visible in the GUI.

## Self-Check: PASSED

All 17 content checks passed. Both commits verified (fa91dc5, 2786e92). All created/modified files present.
