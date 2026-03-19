---
phase: 02-session-registry-websocket-relay
plan: "02"
subsystem: relay
tags: [go, websocket, http-server, hub-manager, integration-tests, coder-websocket, fan-out, scrollback]

# Dependency graph
requires:
  - phase: 02-session-registry-websocket-relay
    plan: "01"
    provides: "Hub, Subscriber, ScrollbackSnapshot, WriteInput, Done, MakeInputFrame, ParseFrame"
provides:
  - "HubManager: per-session Hub lifecycle (Create/Get/Remove/Shutdown)"
  - "HTTP/WebSocket server with subscribe-before-snapshot anti-race pattern"
  - "4 integration tests proving all 3 SESS-03 success criteria with real WS connections"
affects: [03-electron-ui, 04-tls-tunnel, 05-status-bar]

# Tech tracking
tech-stack:
  added:
    - "github.com/coder/websocket v1.8.14"
  patterns:
    - "Subscribe-before-snapshot: hub.Subscribe called before hub.ScrollbackSnapshot to eliminate frame-loss race"
    - "Non-blocking write pump: select on Msgs/ctx.Done/hub.Done/readDone ensures clean shutdown on any failure path"
    - "InsecureSkipVerify on websocket.Accept: origin check deferred to Phase 4 CORS policy"
    - "Slow-client drain loop: test drains until error instead of single-read check, tolerating buffered frames before close frame"

key-files:
  created:
    - internal/relay/manager.go
    - internal/relay/server.go
    - internal/relay/server_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "HubManager.Create is idempotent — returns existing hub if session already exists, preventing double-Run goroutines"
  - "websocket.Accept uses InsecureSkipVerify:true — origin validation deferred to Phase 4 where CORS policy will be defined with known Electron origin"
  - "Slow-client test uses drain-until-error loop — tolerates WebSocket-level buffering between CloseSlow and close frame delivery"

# Metrics
duration: 3min
completed: 2026-03-18
---

# Phase 2 Plan 02: WebSocket Relay Server Summary

**HubManager and HTTP/WebSocket server wiring Hub fan-out to real WebSocket clients, with 4 integration tests proving all SESS-03 success criteria under the race detector**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-18T13:42:08Z
- **Completed:** 2026-03-18T13:44:44Z
- **Tasks:** 2
- **Files modified:** 5 (3 created, 2 modified)

## Accomplishments

- `HubManager` with Create/Get/Remove/Shutdown/SessionIDs — per-session Hub lifecycle with idempotent Create
- `Server` HTTP handler with WebSocket upgrade (coder/websocket), subscribe-before-snapshot anti-race ordering, bidirectional read/write pumps, JSON session list endpoint
- 4 integration tests using real WebSocket connections via `httptest.NewServer`:
  - `TestHub_TwoClientsFanOut` — proves criterion 1 (simultaneous output to all clients)
  - `TestHub_ReconnectScrollback` — proves criterion 2 (reconnect with scrollback replay then live output)
  - `TestHub_InputFanOut` — proves criterion 3 (client input reaches PTY; output broadcast to all)
  - `TestHub_SlowClientDisconnected` — slow client closed via CloseSlow while normal client continues
- Full project suite: 52 tests passing with `-race` (Phase 1 + Phase 2)

## Task Commits

1. **Task 1: HubManager and HTTP/WebSocket server** - `a761337` (feat)
2. **Task 2: Integration tests — prove all 3 SESS-03 criteria** - `1a0d449` (feat)

## Files Created/Modified

- `internal/relay/manager.go` — HubManager: mutex-protected map, Create (idempotent, starts go hub.Run()), Get, Remove, Shutdown, SessionIDs
- `internal/relay/server.go` — Server: NewServer registers routes on ServeMux; handleSession upgrades to WS, subscribe-before-snapshot, read pump (MsgInput/Resize2/Ping), write pump (select on Msgs/ctx/hub.Done/readDone); handleListSessions returns JSON array
- `internal/relay/server_test.go` — 4 integration tests with setupTestServer/dialWS/readFrame helpers; all pass under -race
- `go.mod` / `go.sum` — github.com/coder/websocket v1.8.14 added

## Decisions Made

- HubManager.Create is idempotent — returns existing hub if session already exists, preventing double-Run goroutines on repeated calls
- websocket.Accept uses InsecureSkipVerify:true — Phase 4 will add proper origin/CORS policy once Electron origin is known
- Slow-client test uses drain-until-error loop — tolerates WebSocket protocol buffering between CloseSlow invocation and close frame delivery on the client side

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed TestHub_SlowClientDisconnected false failure**
- **Found during:** Task 2
- **Issue:** Test asserted that a single `Read` call after overflow would return an error; WebSocket buffers frames at the protocol level, so the client could still successfully read buffered frames before receiving the close frame
- **Fix:** Changed assertion to a drain-until-error loop with 5-second deadline — correctly handles the buffered-frames-before-close-frame delivery model
- **Files modified:** internal/relay/server_test.go
- **Commit:** 1a0d449 (inline fix before final commit)

## Issues Encountered

None blocking.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `manager.go` and `server.go` are production-ready network layer components
- Phase 3 (Electron UI) can call `NewServer(manager)` and connect via ws://localhost:{port}/sessions/{id}/ws
- MsgResize2 dispatch stub in handleSession ready for Phase 3 to add `backend.Resize(cols, rows)`
- Phase gate satisfied: all 3 SESS-03 success criteria proven with real WebSocket integration tests under race detector

---
*Phase: 02-session-registry-websocket-relay*
*Completed: 2026-03-18*

## Self-Check: PASSED

- internal/relay/manager.go: FOUND
- internal/relay/server.go: FOUND
- internal/relay/server_test.go: FOUND
- Commit a761337: FOUND
- Commit 1a0d449: FOUND
