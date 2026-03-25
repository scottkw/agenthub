---
phase: 20-process-separation
plan: 01
subsystem: daemon
tags: [go, daemon, relay, webserver, unix-socket, process-spawn, ipc]

# Dependency graph
requires:
  - phase: 19-daemon-core-engine-ipc
    provides: SessionEngine, API, DaemonClient, relay/webserver integration

provides:
  - RunDaemon() entry point that creates SessionEngine + API + relay, blocks on SIGTERM/SIGINT
  - EnsureDaemon() helper that probes health and spawns detached subprocess
  - Platform-specific process detach (Unix Setsid, Windows CREATE_NEW_PROCESS_GROUP)
  - API.StartRelay() method that starts relay TCP listener and returns allocated port
  - 5 new API routes: GET /relay-port, POST /webserver/start, POST /webserver/stop, GET /webserver/status, POST /sessions/{id}/web-serve
  - 5 new DaemonClient methods: GetRelayPort, StartWebServer, StopWebServer, GetWebServerStatus, ToggleWebServing
  - New types: RelayPortResponse, WebServerStartRequest, WebServerStartResponse, WebServerStatusResponse, WebServeRequest

affects: [20-02-process-separation, app-startup, cli-commands]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Relay TCP server started inside API via StartRelay(), port exposed via GET /relay-port for GUI to fetch"
    - "EnsureDaemon polls health every 50ms up to 3s deadline — no sleep, fast startup detection"
    - "Platform-specific detach via build tags: process_unix.go (!windows) and process_windows.go (windows)"
    - "Web server lifecycle delegated to daemon via POST /webserver/start|stop — GUI becomes pure client"

key-files:
  created:
    - internal/daemon/process.go
    - internal/daemon/process_unix.go
    - internal/daemon/process_windows.go
    - internal/daemon/process_test.go
  modified:
    - internal/daemon/types.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - internal/daemon/api_test.go
    - internal/daemon/client_test.go

key-decisions:
  - "Relay TCP server lives inside API struct (relayLn field), started by RunDaemon before api.Start() — daemon owns the relay port lifecycle"
  - "EnsureDaemon takes socketPath as argument (not DefaultSocketPath() internally) — allows tests to inject short socket paths and avoids macOS 103-char limit"
  - "API.Stop() cleans up relay listener and web server in addition to Unix socket — single shutdown method covers all resources"

patterns-established:
  - "StartRelay() pattern: relay.NewServer -> net.Listen(tcp, 0) -> record port in API struct -> go http.Serve -> return port"
  - "Web server delegation: daemon creates webserver.WebServer, sets session resolver from engine, returns URL to client"

requirements-completed: [DAEMON-01, DAEMON-05]

# Metrics
duration: 4min
completed: 2026-03-23
---

# Phase 20 Plan 01: Process Separation - Daemon Infrastructure Summary

**Daemon process infrastructure: RunDaemon/EnsureDaemon entry points, platform-specific detach, relay + web server delegation via 5 new API routes and 5 new DaemonClient methods**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-23T16:02:01Z
- **Completed:** 2026-03-23T16:05:38Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- RunDaemon() creates SessionEngine + API + relay server, blocks on SIGTERM/SIGINT for clean shutdown
- EnsureDaemon() probes health and spawns detached subprocess (Setsid on Unix, CREATE_NEW_PROCESS_GROUP on Windows), polls up to 3s
- API extended with relay port exposure and full web server lifecycle (start/stop/status/toggle-session)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add new types + API routes + client methods for relay port and web server** - `75900a8` (feat)
2. **Task 2: Create process.go with RunDaemon + EnsureDaemon and platform-specific detach files** - `3a9122c` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `internal/daemon/types.go` - Added RelayPortResponse, WebServerStartRequest, WebServerStartResponse, WebServerStatusResponse, WebServeRequest
- `internal/daemon/api.go` - Extended API struct with relayPort/relayLn/mu/webServer fields; StartRelay(); 5 new route handlers
- `internal/daemon/client.go` - Added GetRelayPort, StartWebServer, StopWebServer, GetWebServerStatus, ToggleWebServing
- `internal/daemon/api_test.go` - Added TestAPIRelayPort, TestAPIWebServerStatus_NotRunning, TestAPIWebServe_NoServer
- `internal/daemon/client_test.go` - Added TestClientGetRelayPort, TestClientWebServerStatus
- `internal/daemon/process.go` - RunDaemon() and EnsureDaemon() implementation
- `internal/daemon/process_unix.go` - Unix startDetachedDaemon with Setsid: true
- `internal/daemon/process_windows.go` - Windows startDetachedDaemon with CREATE_NEW_PROCESS_GROUP
- `internal/daemon/process_test.go` - TestEnsureDaemon_AlreadyRunning, TestEnsureDaemon_Timeout, TestRunDaemon_Exports

## Decisions Made
- EnsureDaemon takes socketPath as argument (not hardcoded DefaultSocketPath()) — enables clean testing with short /tmp paths that stay under macOS 103-char sun_path limit
- API.Stop() is the single teardown point for all daemon resources: Unix socket, relay TCP listener, web server — no separate teardown needed in RunDaemon
- Relay server inside API struct (not a separate top-level type) — consistent with how API already owns the Unix socket lifecycle

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Daemon infrastructure is complete: RunDaemon is a drop-in entry point for `agenthub daemon` subcommand
- Phase 20 Plan 02 can now strip in-process engine/API from App struct and call EnsureDaemon() from startup
- All daemon tests pass with -race; GOOS=windows go vet passes; go build ./... passes

---
*Phase: 20-process-separation*
*Completed: 2026-03-23*
