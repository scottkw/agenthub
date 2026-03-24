---
phase: 22-cli-attach
plan: 02
subsystem: cli
tags: [go, websocket, testing, pty, relay, attach, integration-tests]

# Dependency graph
requires:
  - phase: 22-cli-attach
    plan: 01
    provides: cmdAttach, attachSession, stdinPump, wsOutputPump, makeClientResizeFrame
  - phase: 20-process-separation
    provides: relay HubManager + Server infrastructure used in test setup
provides:
  - cmd/agenthub-cli/cmd_attach_test.go with 7 unit/integration tests covering all 4 requirements
affects: [CI test suite, CLI-05, CLI-06, CLI-07, CLI-08 coverage]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "setupAttachTest helper: io.Pipe pairs backing HubManager + httptest.Server (mirrors relay.setupTestServer)"
    - "io.Reader/Writer injection into attachSession for terminal-free testing"
    - "Poll-with-timeout pattern for live output tests (avoids flaky fixed sleeps)"

key-files:
  created:
    - cmd/agenthub-cli/cmd_attach_test.go
  modified: []

key-decisions:
  - "Used polling loop with 10ms intervals instead of channel for TestAttachSession_LiveOutput — simpler than piping through extra channels, tolerates scheduler jitter"
  - "TestAttachSession_OutputReceived uses 30ms sleep before dialing so scrollback is populated — same pattern as relay/server_test.go TestHub_ReconnectScrollback"

# Metrics
duration: 5min
completed: 2026-03-24
---

# Phase 22 Plan 02: CLI Attach Tests Summary

**7 unit and integration tests covering all attach correctness properties — argument validation, MsgResize2 format, detach key clean return, scrollback replay, live output, Ctrl-C passthrough, and keyboard input forwarding**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-24T16:15:50Z
- **Completed:** 2026-03-24T16:20:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- Created `cmd/agenthub-cli/cmd_attach_test.go` with 297 lines and 7 tests covering all attach correctness properties
- `TestCmdAttach_MissingArgs`: proves cmdAttach returns error containing "usage" when no args given (CLI-05)
- `TestMakeClientResizeFrame`: proves first byte is `MsgResize2` (0x11) with correct big-endian encoding (CLI-06)
- `TestAttachSession_DetachKey`: proves detach key (0x1C) causes clean nil return from attachSession (CLI-07)
- `TestAttachSession_OutputReceived`: proves scrollback snapshot is written to stdout before live frames (CLI-08)
- `TestAttachSession_LiveOutput`: proves live PTY output reaches stdout after connection (CLI-05)
- `TestAttachSession_CtrlCPassthrough`: proves byte 0x03 is forwarded to PTY stdin, not intercepted (CLI-06)
- `TestAttachSession_InputForwarded`: proves keyboard bytes reach PTY stdin via relay (CLI-06)
- All 7 tests use real relay infrastructure (HubManager + httptest.Server) for integration coverage

## Task Commits

Each task was committed atomically:

1. **Task 1: Create cmd_attach_test.go with unit and integration tests** - `6a61591` (test)

## Files Created/Modified

- `cmd/agenthub-cli/cmd_attach_test.go` - 7 tests covering CLI-05/06/07/08 with real relay infrastructure

## Decisions Made

- Used polling loop with 10ms intervals instead of extra channels for live output test — simpler and tolerates scheduler jitter
- 30ms sleep before dial in scrollback test mirrors relay/server_test.go pattern to ensure hub processes write before client connects
- All tests use `context.WithTimeout` (5 seconds max) or `context.WithCancel` to prevent hangs

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Tests passed on first run.

## User Setup Required

None.

## Next Phase Readiness

- All 4 requirements (CLI-05 through CLI-08) have automated test coverage
- Test file is 297 lines (exceeds 100-line minimum)
- Full test suite green: 7 packages pass, 0 failures
- `go vet ./...` reports no warnings
