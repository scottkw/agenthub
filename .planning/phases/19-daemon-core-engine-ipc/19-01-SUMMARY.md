---
phase: 19-daemon-core-engine-ipc
plan: 01
subsystem: api
tags: [go, unix-socket, http, daemon, session-engine, pty]

# Dependency graph
requires: []
provides:
  - "internal/daemon package: SessionEngine owning all session state (registry, backend, manager, tabNames, cliPaths, sessionStatuses)"
  - "HTTP API (9 routes) served over Unix socket via net.Listen(unix)"
  - "DaemonClient typed Go client with Unix socket transport"
  - "Socket path utilities: DefaultSocketPath, ValidateSocketPath (103-char limit), CleanupStaleSocket with probe-and-remove"
affects:
  - "19-02: App delegation to SessionEngine (imports engine.go)"
  - "phase-20: process separation (imports all daemon package files)"
  - "phase-22: CLI attach (uses DaemonClient)"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Unix socket HTTP: http.Serve over net.Listen(unix) with Go 1.22+ r.PathValue() path parameters"
    - "Unix dialer pattern: custom http.Transport.DialContext dialing unix network to bypass DNS"
    - "Callback injection for Wails-free engine: onStatus func(string, status.SessionStatus) passed at CreateSession call site"
    - "Short socket paths in tests: /tmp/dtest{n}_{name} to stay under 103-char macOS sun_path limit"

key-files:
  created:
    - internal/daemon/types.go
    - internal/daemon/engine.go
    - internal/daemon/socket.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - internal/daemon/engine_test.go
    - internal/daemon/socket_test.go
    - internal/daemon/api_test.go
    - internal/daemon/client_test.go
  modified: []

key-decisions:
  - "onStatus callback injected at CreateSession call site (not wired in engine) so engine has zero Wails imports; App in Plan 02 supplies the EventsEmit wrapper"
  - "Socket path limit enforced at 103 chars (macOS sun_path = 104 bytes including null terminator)"
  - "CleanupStaleSocket probes with net.DialTimeout(500ms): ECONNREFUSED/timeout = stale (remove), connection success = already running (error)"
  - "testDaemon helper uses /tmp/dtest{n}_{name} short paths rather than t.TempDir() which generates paths >103 chars on macOS"

patterns-established:
  - "Unix socket HTTP server: net.Listen(unix) + http.Serve; CleanupStaleSocket called before bind"
  - "Unix socket HTTP client: http.Transport{DialContext: unix dialer} with base http://daemon"
  - "TDD: failing tests written first, then implementation to green"

requirements-completed:
  - DAEMON-02

# Metrics
duration: 25min
completed: 2026-03-23
---

# Phase 19 Plan 01: Daemon Core Engine + IPC Summary

**SessionEngine extracted from App with HTTP/JSON API over Unix socket and typed DaemonClient — 28 tests pass with -race, zero Wails imports in daemon package**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-03-23T14:30:00Z
- **Completed:** 2026-03-23T14:55:00Z
- **Tasks:** 2
- **Files created:** 9 (5 source + 4 test)

## Accomplishments
- SessionEngine owns all session state (registry, backend, manager, tabNames, cliPaths, sessionStatuses) with zero Wails imports
- HTTP API serves 9 routes over Unix socket using Go 1.22+ path parameters
- DaemonClient round-trips through Unix socket for every operation with typed returns
- Socket utilities handle path validation (103-char limit), stale socket cleanup, and DefaultSocketPath
- 28 tests pass with -race across engine, socket, API, and client layers

## Task Commits

Each task was committed atomically:

1. **Task 1: Create daemon types, SessionEngine, socket utilities, and their tests** - `ebe68ac` (feat)
2. **Task 2: Create HTTP API server, DaemonClient, and their round-trip tests** - `b191b50` (feat)

**Plan metadata:** (this commit)

_Note: TDD tasks had tests written first (RED), then implementation to GREEN._

## Files Created/Modified
- `internal/daemon/types.go` - Shared types: SessionInfo, CreateRequest, CreateResponse, RenameRequest, StatusResponse, HealthResponse, CLIPathsResponse, UpdateCLIPathRequest
- `internal/daemon/engine.go` - SessionEngine with all session state; onStatus callback replaces hardcoded runtime.EventsEmit
- `internal/daemon/socket.go` - DefaultSocketPath, ValidateSocketPath (103-char limit), CleanupStaleSocket
- `internal/daemon/api.go` - HTTP API (9 routes) over net.Listen("unix") using Go 1.22+ r.PathValue()
- `internal/daemon/client.go` - DaemonClient with custom Transport dialing unix socket; doJSON shared helper
- `internal/daemon/engine_test.go` - TestNewSessionEngine through TestEngineResolveCLI (7 tests)
- `internal/daemon/socket_test.go` - TestSocketPathDefault, TestSocketPathLength, TestCleanupStaleSocket_* (5 tests)
- `internal/daemon/api_test.go` - TestAPIHealth through TestAPIUpdateCLIPath (10 tests, raw HTTP)
- `internal/daemon/client_test.go` - TestClientHealth through TestClientFullLifecycle (7 tests)

## Decisions Made
- **onStatus callback injection:** Engine receives `onStatus func(string, status.SessionStatus)` at `CreateSession` call site. Engine itself has zero Wails imports. App (Plan 02) provides a callback that wraps `runtime.EventsEmit`.
- **Socket path limit 103:** macOS `sun_path` is 104 bytes including the null terminator, so max usable length is 103 chars.
- **CleanupStaleSocket probe pattern:** `net.DialTimeout("unix", path, 500ms)` — if connection refused or times out, it's stale; if succeeds, daemon is running and we return an error with "already running".
- **Short socket paths in tests:** `t.TempDir()` on macOS generates paths >103 chars. Tests use `/tmp/dtest{n}_{name}` with an atomic counter to stay under the limit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `DialContext` interface mismatch in api_test.go helpers**
- **Found during:** Task 2 (api_test.go compilation)
- **Issue:** Test helpers used `interface{ Done() <-chan struct{} }` for context parameter instead of `context.Context`, causing type errors
- **Fix:** Added `context` import and changed parameter type to `context.Context`; extracted `dialUnix` helper to deduplicate dialer code
- **Files modified:** internal/daemon/api_test.go
- **Verification:** `go test -run NOTHING ./internal/daemon/...` compiles cleanly
- **Committed in:** b191b50 (Task 2 commit)

**2. [Rule 1 - Bug] Fixed socket path length exceeded in socket_test.go**
- **Found during:** Task 1 (socket tests)
- **Issue:** `t.TempDir()` returns paths >103 chars on macOS, causing `bind: invalid argument` when `net.Listen("unix", path)` is called in tests
- **Fix:** Added `shortSocketPath` helper using `/tmp/dtest{n}_{name}` with `atomic.Int64` counter; used in all socket tests that open real listeners
- **Files modified:** internal/daemon/socket_test.go
- **Verification:** All socket tests pass: `TestCleanupStaleSocket_StaleFile`, `TestCleanupStaleSocket_ActiveSocket`
- **Committed in:** ebe68ac (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes were necessary for tests to compile and run on macOS. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/daemon` package complete with full test coverage
- Plan 02 can now import `daemon.SessionEngine` to delegate App's session methods
- `NewSessionEngine()`, `CreateSession(ctx, cli, name, workDir, onStatusCallback)`, `ListSessions()`, `KillSession()`, `RenameSession()`, `GetSessionStatus()`, `ResolveCLI()`, `UpdateCLIPath()`, `GetCLIPaths()` all ready
- `daemon.NewAPI(engine)` and `daemon.NewDaemonClient(socketPath)` ready for IPC integration
- Blocker: none; Phase 19 Plan 02 (App delegation) can proceed immediately

---
*Phase: 19-daemon-core-engine-ipc*
*Completed: 2026-03-23*
