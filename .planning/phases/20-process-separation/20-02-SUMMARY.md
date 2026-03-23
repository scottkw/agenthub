---
phase: 20-process-separation
plan: 02
subsystem: app
tags: [go, daemon, frontend, react, wails, process-separation, client-delegation]

# Dependency graph
requires:
  - phase: 20-process-separation
    plan: 01
    provides: RunDaemon, EnsureDaemon, DaemonClient with GetRelayPort/StartWebServer/StopWebServer/GetWebServerStatus/ToggleWebServing

provides:
  - main.go with daemon subcommand dispatch (os.Args[1]=="daemon" -> RunDaemon)
  - App struct stripped to ctx/client/trayInit — no engine, api, relay, webserver fields
  - startup() calls EnsureDaemon then wires DaemonClient
  - shutdown() does NOT stop daemon (sessions survive GUI close)
  - All session/relay/webserver ops delegated through DaemonClient
  - pollSessionStatus goroutine for async status events (replaces onStatus callback)
  - Frontend daemon error banner with retry button

affects: [gui-session-persistence, daemon-auto-start, relay-port-handoff]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "pollSessionStatus goroutine polls daemon every 2s up to 60s — replaces onStatus callback that could not be serialized over HTTP"
    - "SetWebServerForTest on daemon.API enables TLS injection in tests without Tailscale"
    - "testAppWithDirectWebServer helper encapsulates in-process daemon + webserver setup for QR code tests"

key-files:
  created: []
  modified:
    - main.go
    - app.go
    - app_test.go
    - tray_test.go
    - internal/daemon/api.go
    - frontend/src/App.tsx

key-decisions:
  - "pollSessionStatus polls every 2s up to 60s — replaces onStatus callback; callbacks cannot be serialized over HTTP so polling is the correct pattern for out-of-process daemon"
  - "SetWebServerForTest added to daemon.API — enables test injection of TLS webserver without requiring Tailscale in test environment"
  - "shutdown() has no daemon teardown — daemon is an independent process; GUI closing does not affect session state (DAEMON-03 requirement)"

requirements-completed: [DAEMON-01, DAEMON-03, DAEMON-04, DAEMON-05]

# Metrics
duration: 7min
completed: 2026-03-23
---

# Phase 20 Plan 02: Process Separation - App Refactor Summary

**App stripped to thin DaemonClient shell with daemon dispatch in main.go, all session/relay/webserver ops delegated through DaemonClient, frontend error banner added for daemon connection failures**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-23T16:08:23Z
- **Completed:** 2026-03-23T16:15:26Z
- **Tasks:** 3 of 3 complete (2 automated + 1 human-verify checkpoint, approved)
- **Files modified:** 6

## Accomplishments
- main.go dispatches `agenthub daemon` subcommand to RunDaemon() before Wails startup
- App struct reduced from 8 fields (engine, api, client, socketPath, server, listener, trayInit, mu, webServer) to 3 (ctx, client, trayInit)
- startup() calls EnsureDaemon(socketPath) then wires DaemonClient — no in-process engine/API/relay
- shutdown() contains no daemon teardown — sessions persist after GUI close (DAEMON-03)
- CreateSession delegates through DaemonClient; pollSessionStatus goroutine emits Wails events
- GetRelayPort fetches port from daemon via client.GetRelayPort()
- StartWebServer/StopWebServer/ToggleWebServing/GetWebServerURL/IsWebServerRunning/GetSessionQRCode all delegate through DaemonClient
- Frontend daemonError state captures init() failures; retryInit callback re-runs full init sequence
- Error banner shows "Unable to connect to session daemon" with Retry Connection button when daemon unreachable and no tabs exist

## Task Commits

Each task was committed atomically:

1. **Task 1: main.go dispatch + app.go refactor + test updates** - `55d0cda` (feat)
2. **Task 2: Frontend daemon error banner** - `9a57cde` (feat)
3. **Task 3: GUI regression verification** — human-approved (no code commit, all 10 steps passed)

**Bug fixes committed during Task 3 verification:**
- `a4c9f67`: fix(20-02): fix blank terminal from stale daemon and silent relay failure
- `fda05c5`: fix(20-02): use background context for PTY sessions in daemon API

**Plan metadata:** `4207532` (docs: plan summary and state update — awaiting Task 3 checkpoint)

## Files Created/Modified
- `main.go` - Added daemon subcommand dispatch before Wails startup
- `app.go` - Stripped App struct to ctx/client/trayInit; all ops delegate through DaemonClient; added pollSessionStatus
- `app_test.go` - Replaced testApp() with in-process daemon pattern; added testAppWithDirectWebServer helper
- `tray_test.go` - Replaced app.engine.Registry().Len() with app.client.ListSessions()
- `internal/daemon/api.go` - Added SetWebServerForTest for test TLS injection
- `frontend/src/App.tsx` - Added daemonError state, retryInit callback, error banner JSX

## Decisions Made
- pollSessionStatus goroutine replaces onStatus callback — callbacks cannot be serialized over HTTP, polling is the correct pattern for out-of-process daemon
- SetWebServerForTest on daemon.API enables test injection of TLS webserver without requiring Tailscale, maintaining existing QR code test coverage
- shutdown() has no daemon teardown — daemon is independent process, GUI closing must not affect sessions (DAEMON-03)

## Deviations from Plan

### Auto-added functionality

**1. [Rule 2 - Missing critical functionality] Added SetWebServerForTest to daemon.API**
- **Found during:** Task 1 (app_test.go refactor)
- **Issue:** TestGetSessionQRCode previously bypassed the App by directly setting app.webServer. After removing that field, tests needed a way to inject a TLS webserver into the daemon API without going through the Tailscale-gated HTTP route
- **Fix:** Added `SetWebServerForTest(ws *webserver.WebServer)` to daemon.API — thread-safe injection method for test use only
- **Files modified:** `internal/daemon/api.go`
- **Commit:** `55d0cda`

**2. [Rule 1 - Bug] Blank terminal from stale socket and silent relay failure**
- **Found during:** Task 3 GUI regression verification
- **Issue:** EnsureDaemon probed an old socket, got "connection refused", but stale socket cleanup left the daemon in a state where relay was not started. GetRelayPort returned 0 — frontend WebSocket never connected, terminal stayed blank.
- **Fix:** Fixed stale socket cleanup in EnsureDaemon; relay startup failure now surfaces as an error instead of being silent
- **Files modified:** `internal/daemon/api.go`, `internal/daemon/process.go`
- **Commit:** `a4c9f67`

**3. [Rule 1 - Bug] PTY sessions killed by HTTP request context cancellation**
- **Found during:** Task 3 GUI regression verification (terminal briefly showed output then went blank)
- **Issue:** daemon API passed the HTTP request's `context.Context` to PTY session goroutines. When the HTTP handler returned its response, the context was cancelled, terminating the PTY session immediately.
- **Fix:** Changed PTY session creation to use `context.Background()` — PTY goroutines must outlive the HTTP request that created them
- **Files modified:** `internal/daemon/api.go`
- **Commit:** `fda05c5`

---

**Total deviations:** 3 auto-fixed (1 missing critical functionality + 2 bugs)
**Impact on plan:** All auto-fixes necessary for correctness. SetWebServerForTest preserved test coverage. Both bugs would have caused complete feature failure in production.

## Issues Encountered

- macOS `Cmd+W` does not trigger window close event in Wails — it hides to tray only via the close button (X). Pre-existing Wails behavior, not a regression.
- Task 3 verification revealed two bugs (blank terminal, PTY context) that required post-task fixes before human approval could be given.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Process separation is complete: daemon runs as independent process, GUI is pure DaemonClient consumer
- Sessions survive GUI close/reopen once EnsureDaemon reconnects on startup
- Daemon auto-starts on GUI launch via EnsureDaemon
- Relay port is fetched from daemon via GetRelayPort()
- GUI regression verified by human: all 10 steps passed (session creation, daemon process visible, close/reopen session survival, relay port working, daemon auto-restart after manual kill)
- Process separation complete end-to-end

---
*Phase: 20-process-separation*
*Completed: 2026-03-23*
