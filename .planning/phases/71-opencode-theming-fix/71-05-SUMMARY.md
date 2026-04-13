---
phase: 71-opencode-theming-fix
plan: 05
subsystem: daemon,frontend
tags: [go, opencode, sigusr2, signal, theme, wails, integration-test, gap-closure]

# Dependency graph
requires:
  - phase: 71-02
    provides: "OPENCODE_TUI_CONFIG env injection + managed tui.json + sessionCLIs tracking"
provides:
  - "Session.Signal(sig os.Signal) error method in internal/pty/session.go"
  - "SessionEngine.sessionCLIs map tracking raw CLI names per session"
  - "SessionEngine.NotifyThemeChange() broadcasting SIGUSR2 to active opencode sessions"
  - "notify_theme_unix.go: signalThemeChange sends syscall.SIGUSR2"
  - "notify_theme_windows.go: signalThemeChange is a no-op"
  - "POST /theme/notify daemon HTTP route returning 204"
  - "DaemonClient.NotifyThemeChange() client method"
  - "App.NotifyThemeChange() Wails binding (nil-safe)"
  - "Frontend handleThemeChange calls NotifyThemeChange().catch() fire-and-forget"
  - "Integration test proving end-to-end SIGUSR2 delivery via real PTY process"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "POSIX signal delivery via os.Process.Signal from PTY layer"
    - "Platform-split build-tag files for signal vs no-op behavior"
    - "Hub subscriber pattern for reading PTY output in integration tests"
    - "Fire-and-forget Wails call with .catch() error handling"

key-files:
  created:
    - internal/pty/session_test.go
    - internal/daemon/notify_theme_unix.go
    - internal/daemon/notify_theme_windows.go
  modified:
    - internal/pty/session.go
    - internal/daemon/engine.go
    - internal/daemon/engine_test.go
    - internal/daemon/api.go
    - internal/daemon/api_test.go
    - internal/daemon/client.go
    - app.go
    - app_test.go
    - frontend/src/App.tsx
    - frontend/src/components/__tests__/App.test.tsx
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js

key-decisions:
  - "Used hub subscriber pattern (not sess.Read) for integration test output — engine.CreateSession wires sess into relay hub which consumes the PTY reader, making direct sess.Read race with the hub"
  - "Scrollback snapshot parsed by stripping MsgOutput (0x01) prefix bytes — simpler than frame-boundary parsing since 0x01 does not appear in ASCII marker strings"
  - "NotifyThemeChange returns nil (not error) when client is nil — fire-and-forget from frontend, no daemon = no sessions to signal"
  - "sessionCLIs stores raw CLI name (not resolved cliPath) — matches existing per-CLI detection pattern in status/detector.go"

patterns-established:
  - "Session.Signal: nil-safe signal delivery to PTY child process via cmd.Process.Signal"
  - "Platform-split notify helper: notify_theme_unix.go / notify_theme_windows.go with build tags"
  - "hubReadUntil: integration test helper subscribing to relay hub for PTY output verification"

requirements-completed: [THM-05]

# Metrics
duration: 20min
completed: 2026-04-13
---

# Phase 71 Plan 05: SIGUSR2 Broadcast for Live Theme Switching Summary

**Full signal pipeline from frontend theme picker through Wails -> daemon HTTP API -> engine broadcast -> per-session SIGUSR2 delivery, closing SC-1 gap for live OpenCode theme switching**

## Performance

- **Duration:** ~20 minutes
- **Completed:** 2026-04-13
- **Tasks:** 3
- **Files modified:** 12 (2 created, 10 modified)

## Accomplishments

- Added `Session.Signal(sig os.Signal) error` method with nil-safety to `internal/pty/session.go`
- Added `sessionCLIs map[string]string` to `SessionEngine` — populated in `CreateSession`, cleaned in `KillSession`
- Added `SessionEngine.NotifyThemeChange()` that walks registry, filters to opencode sessions, dispatches SIGUSR2 via platform helper
- Created `notify_theme_unix.go` (sends `syscall.SIGUSR2`) and `notify_theme_windows.go` (no-op) with build tags
- Added `POST /theme/notify` HTTP route in `api.go` returning 204 No Content
- Added `DaemonClient.NotifyThemeChange()` client method
- Added `App.NotifyThemeChange()` Wails binding (nil-safe — returns nil when daemon not connected)
- Wired frontend `handleThemeChange` to call `NotifyThemeChange().catch(err => console.warn(...))` fire-and-forget
- Added `NotifyThemeChange` to `App.js` and `App.d.ts` Wails binding stubs
- Integration test `TestNotifyThemeChange_RealProcess_Integration`: spawns real `/bin/sh` with USR2 trap, calls `NotifyThemeChange`, asserts `SIGUSR2_RECEIVED` marker appears in PTY output via hub subscriber
- Full Go test suite (9 packages) passes; frontend vitest suite (353 tests) passes

## Task Commits

Each task committed atomically:

1. **Task 1: Session.Signal + sessionCLIs + NotifyThemeChange + platform helpers** — `0e1a50d` (feat)
2. **Task 2: HTTP route + client + Wails binding + frontend wiring** — `6ae4aa0` (feat)
3. **Task 3: Integration test for real SIGUSR2 delivery** — `a7128c1` (test)

## Files Created/Modified

- `internal/pty/session.go` — Added `Signal(sig os.Signal) error` method
- `internal/pty/session_test.go` — New: nil-cmd and nil-process Signal edge case tests
- `internal/daemon/engine.go` — Added `sessionCLIs` field, `log` import, `NotifyThemeChange` method; populated/cleaned in CreateSession/KillSession
- `internal/daemon/notify_theme_unix.go` — New: POSIX `signalThemeChange` sending `syscall.SIGUSR2`
- `internal/daemon/notify_theme_windows.go` — New: Windows no-op `signalThemeChange`
- `internal/daemon/engine_test.go` — Added NotifyThemeChange/sessionCLIs unit tests + integration test with hub subscriber
- `internal/daemon/api.go` — Added `POST /theme/notify` route + `handleNotifyThemeChange` handler
- `internal/daemon/api_test.go` — Added `TestHandleNotifyThemeChange`, `TestHandleNotifyThemeChange_WithSessions`, `TestClientNotifyThemeChange`
- `internal/daemon/client.go` — Added `NotifyThemeChange()` client method
- `app.go` — Added `App.NotifyThemeChange()` Wails binding
- `app_test.go` — Added `TestNotifyThemeChange`, `TestNilClientNotifyThemeChange`
- `frontend/src/App.tsx` — Added `NotifyThemeChange` import; wired `handleThemeChange` to call it fire-and-forget
- `frontend/src/components/__tests__/App.test.tsx` — Added THM-05 describe block
- `frontend/src/wailsjs/go/main/App.d.ts` — Added `NotifyThemeChange(): Promise<void>` declaration
- `frontend/src/wailsjs/go/main/App.js` — Added `NotifyThemeChange` binding stub

## Decisions Made

- **Hub subscriber for integration test (not direct sess.Read):** `engine.CreateSession` wires `sess` into the relay hub which starts a goroutine consuming the PTY reader. Direct `sess.Read` in the test races with the hub's drain loop and sees no data. The hub subscriber pattern receives all PTY output reliably.
- **Scrollback parsed by stripping MsgOutput (0x01) bytes:** Each frame in the concatenated scrollback snapshot starts with `0x01` (MsgOutput). Since `0x01` doesn't appear in the ASCII marker strings (`READY`, `SIGUSR2_RECEIVED`), stripping all `0x01` bytes recovers the text payload without needing frame-boundary parsing.
- **`NotifyThemeChange` returns nil (not error) when `a.client == nil`:** Fire-and-forget from frontend; if daemon isn't connected there are no sessions to signal. Consistent with `GetWebServerURL` returning empty string for nil client.
- **`sessionCLIs` stores raw CLI name (not resolved cliPath):** Matches existing detection pattern in `internal/status/detector.go` where `cli == "claude"` checks use raw names. The resolved path varies by machine; the raw name is canonical.

## Deviations from Plan

**1. [Rule 2 - Missing functionality] Added NotifyThemeChange to Wails JS/d.ts stubs**

- **Found during:** Task 2 frontend wiring
- **Issue:** `frontend/src/wailsjs/go/main/App.d.ts` and `App.js` are hand-maintained stubs (see their headers). The plan noted the d.ts is auto-generated but in this project it's actually a manually maintained stub. `NotifyThemeChange` needed to be added to both files for TypeScript compilation and Wails runtime dispatch.
- **Fix:** Added `NotifyThemeChange` entry to both `App.d.ts` (type declaration) and `App.js` (Call stub).
- **Files modified:** `frontend/src/wailsjs/go/main/App.d.ts`, `frontend/src/wailsjs/go/main/App.js`
- **Commit:** `6ae4aa0`

**2. [Rule 1 - Bug] Replaced direct sess.Read with hub subscriber in integration test**

- **Found during:** Task 3 integration test
- **Issue:** Initial implementation used `sess.Read()` directly to observe PTY output. The test failed with "process did not print READY within 5s" because `engine.CreateSession` wraps `sess` in `relay.Hub` which starts a goroutine consuming the PTY reader — leaving nothing for `sess.Read()`.
- **Fix:** Rewrote `hubReadUntil` helper to subscribe to the relay hub using `hub.Subscribe(sub)`, replay `hub.ScrollbackSnapshot()`, and drain `sub.Msgs` channel. Integration test now passes in 0.12s.
- **Files modified:** `internal/daemon/engine_test.go`
- **Commit:** `a7128c1`

## Known Stubs

None. All wiring is live — `NotifyThemeChange` dispatches through all layers end-to-end, verified by integration test.

## Threat Surface Scan

No new threat surfaces beyond those documented in the plan's threat model (T-71-04, T-71-05, T-71-06). The `POST /theme/notify` endpoint is on the Unix socket (same-UID access only), carries no payload, and returns no session data.

## Self-Check: PASSED

- FOUND: internal/pty/session.go (Signal method)
- FOUND: internal/pty/session_test.go
- FOUND: internal/daemon/engine.go (NotifyThemeChange, sessionCLIs)
- FOUND: internal/daemon/notify_theme_unix.go
- FOUND: internal/daemon/notify_theme_windows.go
- FOUND: internal/daemon/engine_test.go (integration test)
- FOUND: internal/daemon/api.go (POST /theme/notify)
- FOUND: internal/daemon/api_test.go
- FOUND: internal/daemon/client.go (NotifyThemeChange)
- FOUND: app.go (App.NotifyThemeChange)
- FOUND: app_test.go
- FOUND: frontend/src/App.tsx (NotifyThemeChange import + call)
- FOUND: frontend/src/components/__tests__/App.test.tsx (THM-05 block)
- FOUND: 0e1a50d (Task 1 commit)
- FOUND: 6ae4aa0 (Task 2 commit)
- FOUND: a7128c1 (Task 3 commit)

---
*Phase: 71-opencode-theming-fix*
*Completed: 2026-04-13*
