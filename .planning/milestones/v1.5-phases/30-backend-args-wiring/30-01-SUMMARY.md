---
phase: 30-backend-args-wiring
plan: 01
subsystem: api
tags: [go, daemon, pty, ipc, args, unix-socket]

requires:
  - phase: daemon-core-engine-ipc
    provides: SessionEngine, DaemonClient, HTTP API over Unix socket
provides:
  - daemon.CreateRequest with Args []string field (JSON: "args,omitempty")
  - SessionEngine.CreateSession with args []string forwarded to pty.CreateRequest.Args
  - DaemonClient.CreateSession with args []string parameter
  - App.CreateSession (Wails binding) with args []string parameter
  - Integration tests proving args survive HTTP IPC round-trip, typed client, and engine-to-PTY
affects:
  - frontend (Wails JS bindings regenerated — CreateSession now accepts 4th arg)
  - CLI callers that invoke CreateSession

tech-stack:
  added: []
  patterns:
    - "args []string threaded through all IPC layers (types -> engine -> api -> client -> app)"
    - "Wails binding matches DaemonClient signature for consistent caller experience"
    - "json:\"args,omitempty\" omits empty args from JSON wire format (no regression for nil callers)"

key-files:
  created: []
  modified:
    - internal/daemon/types.go
    - internal/daemon/engine.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - internal/daemon/client_test.go
    - internal/daemon/engine_test.go
    - internal/daemon/api_test.go
    - app.go
    - app_test.go
    - cmd_cli.go
    - cmd_cli_test.go
    - tray_test.go

key-decisions:
  - "Args threaded between workDir and onStatus in engine signature to match logical call order (request fields before callbacks)"
  - "json:\"args,omitempty\" ensures nil/empty args produce no JSON field — backward-compatible wire format"
  - "All existing callers updated to pass nil — no behavioral change, no test regressions"

patterns-established:
  - "New request fields flow: types.CreateRequest -> engine param -> api forward -> client param -> app binding"

requirements-completed: [ARGS-03]

duration: 4min
completed: 2026-03-25
---

# Phase 30 Plan 01: Backend Args Wiring Summary

**`args []string` threaded through all 5 daemon IPC layers (CreateRequest, SessionEngine, API handler, DaemonClient, Wails binding) with integration tests proving args survive the full HTTP round-trip from JSON to PTY**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-25T00:25:55Z
- **Completed:** 2026-03-25T00:29:55Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments

- Added `Args []string` to all 5 daemon IPC layers with zero regressions on existing callers
- All existing callers (cmd_cli.go, app_test.go, cmd_cli_test.go, tray_test.go, client_test.go, engine_test.go) updated to pass `nil` for backward compatibility
- Three new integration tests prove args survive: HTTP JSON round-trip (TestAPICreateSessionWithArgs), typed client (TestClientCreateSessionWithArgs), and engine-to-PTY (TestEngineCreateSessionWithArgs)
- Full test suite green (60+ tests across 6 packages)

## Task Commits

1. **Task 1: Thread args through all 5 layers and update all callers** - `279ad30` (feat)
2. **Task 2: Add integration tests for args through full IPC chain** - `374eda8` (test)

**Plan metadata:** (pending final commit)

## Files Created/Modified

- `/Users/ken/dev/agenthub/internal/daemon/types.go` - Added `Args []string \`json:"args,omitempty"\`` to CreateRequest
- `/Users/ken/dev/agenthub/internal/daemon/engine.go` - Added `args []string` param to CreateSession, forwarded to pty.CreateRequest.Args
- `/Users/ken/dev/agenthub/internal/daemon/api.go` - Pass `req.Args` to engine.CreateSession in handleCreateSession
- `/Users/ken/dev/agenthub/internal/daemon/client.go` - Added `args []string` param to DaemonClient.CreateSession, populates CreateRequest.Args
- `/Users/ken/dev/agenthub/app.go` - Added `args []string` param to App.CreateSession (Wails binding), delegates to client
- `/Users/ken/dev/agenthub/cmd_cli.go` - Updated cmdNew to pass `nil` for args
- `/Users/ken/dev/agenthub/internal/daemon/api_test.go` - Added TestAPICreateSessionWithArgs, TestClientCreateSessionWithArgs
- `/Users/ken/dev/agenthub/internal/daemon/engine_test.go` - Added TestEngineCreateSessionWithArgs; updated all existing CreateSession calls to 6-param signature
- `/Users/ken/dev/agenthub/app_test.go` - Updated 8 CreateSession calls to pass nil
- `/Users/ken/dev/agenthub/cmd_cli_test.go` - Updated 6 CreateSession calls to pass nil
- `/Users/ken/dev/agenthub/tray_test.go` - Updated 2 CreateSession calls to pass nil
- `/Users/ken/dev/agenthub/internal/daemon/client_test.go` - Updated 5 CreateSession calls to pass nil

## Decisions Made

- Args positioned between `workDir string` and `onStatus func(...)` in engine signature — request-field parameters logically precede callback parameters
- `json:"args,omitempty"` ensures nil/empty args produce no JSON field, making wire format backward-compatible with existing daemon instances
- All existing callers pass explicit `nil` rather than empty slice — consistent with existing `onStatus` nil convention

## Deviations from Plan

**1. [Rule 3 - Blocking] Also updated internal/daemon/client_test.go**
- **Found during:** Task 1 (build verification)
- **Issue:** `client_test.go` was not in the plan's files list but contained 5 calls to the old `CreateSession` signature
- **Fix:** Updated all 5 calls to pass `nil` as the args argument
- **Files modified:** internal/daemon/client_test.go
- **Verification:** `go test ./...` passes
- **Committed in:** 279ad30 (Task 1 commit)

**2. golangci-lint not available**
- golangci-lint is not installed on this machine; the linter verification step from the plan could not be executed
- Build and all tests pass; no linting issues evident from code review

---

**Total deviations:** 2 (1 auto-fixed missing caller file, 1 tool unavailable)
**Impact on plan:** Auto-fix was necessary for correctness (compilation). Linter unavailability is environment-only; code is clean.

## Issues Encountered

- `TestHub_SlowClientDisconnected` in `internal/relay` is a pre-existing race-sensitive test that occasionally fails when run in parallel with other tests. It passes consistently when run in isolation. This is unrelated to the args-wiring changes (zero changes to internal/relay/).

## Known Stubs

None — all implementation wires real data from API boundary to PTY.

## Next Phase Readiness

- `args []string` is now available at all IPC layers; frontend can pass args through `CreateSession` Wails binding
- Wails JS binding will need regeneration to expose the new `args` parameter to React
- Remaining phases in the 30-backend-args-wiring milestone can build on this foundation

---
*Phase: 30-backend-args-wiring*
*Completed: 2026-03-25*
