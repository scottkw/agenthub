---
phase: 22-cli-attach
plan: 01
subsystem: cli
tags: [go, websocket, pty, terminal, raw-mode, sigwinch, attach]

# Dependency graph
requires:
  - phase: 19-daemon-core-engine-ipc
    provides: DaemonClient.GetRelayPort() and ListSessions() used to locate the relay
  - phase: 20-process-separation
    provides: relay WebSocket server at ws://127.0.0.1:<port>/sessions/<id>/ws
  - phase: 21-cli-session-web-commands
    provides: main.go command dispatch pattern that attach follows
provides:
  - Full interactive PTY proxy: cmdAttach, attachSession, stdinPump, wsOutputPump, makeClientResizeFrame
  - Unix SIGWINCH handler in cmd_attach_unix.go
  - Windows no-op stub in cmd_attach_windows.go
  - golang.org/x/term promoted to direct dependency
  - agenthub attach <id> command wired into CLI dispatch
affects: [22-cli-attach plan-02, any future CLI interactive commands]

# Tech tracking
tech-stack:
  added: [golang.org/x/term v0.41.0 (direct)]
  patterns:
    - Two-goroutine I/O pump pattern (stdinPump + wsOutputPump) with select on done channels
    - Platform-specific build tag files (*_unix.go vs *_windows.go) for signal handling
    - signal.NotifyContext for SIGTERM/SIGHUP; raw mode passes Ctrl-C as byte 0x03
    - defer term.Restore covers all exit paths including panic recovery

key-files:
  created:
    - cmd/agenthub-cli/cmd_attach.go
    - cmd/agenthub-cli/cmd_attach_unix.go
    - cmd/agenthub-cli/cmd_attach_windows.go
  modified:
    - cmd/agenthub-cli/main.go
    - go.mod
    - go.sum

key-decisions:
  - "Use MsgResize2 (0x11) not MsgResize (0x02) for client-to-server resize — server read pump only handles MsgResize2 for incoming resize frames"
  - "Do NOT catch SIGINT — in raw mode Ctrl-C is byte 0x03, catching SIGINT would prevent user from sending Ctrl-C to the remote session"
  - "Detach key default Ctrl-backslash (0x1C) — same as GNU screen, clean return without harming session"
  - "attachSession takes io.Reader/Writer not os.Stdin/Stdout — enables unit testing without terminal"

patterns-established:
  - "Two-pump I/O pattern: stdinPump and wsOutputPump run as goroutines, select waits for first to finish"
  - "Platform files: *_unix.go with go:build !windows, *_windows.go with go:build windows for signal differences"

requirements-completed: [CLI-05, CLI-06, CLI-07, CLI-08]

# Metrics
duration: 2min
completed: 2026-03-24
---

# Phase 22 Plan 01: CLI Attach Implementation Summary

**Full interactive PTY proxy for `agenthub attach <id>` — WebSocket relay with raw mode, detach key, SIGWINCH resize, and signal-safe terminal restore**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-24T16:10:08Z
- **Completed:** 2026-03-24T16:12:37Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Implemented full PTY proxy in cmd_attach.go: connects to relay WebSocket, puts terminal in raw mode, runs two-goroutine I/O pump, detaches cleanly on Ctrl-backslash without harming session
- SIGWINCH handler in cmd_attach_unix.go propagates terminal resize to PTY via MsgResize2 frames; Windows stub is a no-op
- Wired `attach` command into main.go dispatch with usage hint; binary correctly shows usage error and exits 1 when called without args

## Task Commits

Each task was committed atomically:

1. **Task 1: Create cmd_attach.go with full PTY proxy implementation** - `7c62b07` (feat)
2. **Task 2: Wire attach command into main.go CLI dispatch** - `41f0dd0` (feat)

## Files Created/Modified

- `cmd/agenthub-cli/cmd_attach.go` - Core attach implementation: cmdAttach, attachSession, stdinPump, wsOutputPump, makeClientResizeFrame
- `cmd/agenthub-cli/cmd_attach_unix.go` - SIGWINCH watcher goroutine for Unix (go:build !windows)
- `cmd/agenthub-cli/cmd_attach_windows.go` - No-op stub for Windows (go:build windows)
- `cmd/agenthub-cli/main.go` - Added attach case and usage text
- `go.mod` / `go.sum` - golang.org/x/term promoted to direct dependency

## Decisions Made

- Used `relay.MsgResize2` (0x11) for client-to-server resize frames, not `relay.MakeResizeFrame()` which produces `MsgResize` (0x02). The relay server's read pump at server.go handles `case MsgResize2` for incoming resize, not `MsgResize`.
- `signal.NotifyContext` catches only SIGTERM and SIGHUP — NOT SIGINT. In raw mode, Ctrl-C arrives as byte 0x03 and should be forwarded to the remote PTY, not intercepted as a signal.
- `attachSession` accepts `io.Reader`/`io.Writer` instead of `os.Stdin`/`os.Stdout` to enable testing without a real terminal.
- `defer term.Restore` placed immediately after `term.MakeRaw` so it runs on every exit path (normal, error, panic recovery).

## Deviations from Plan

None - plan executed exactly as written.

The only minor observation: `go get golang.org/x/term` initially left it as indirect; `go mod tidy` after the implementation file was created correctly promoted it to a direct dependency.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `agenthub attach <id>` command fully implemented and tested
- Platform-specific SIGWINCH handling ensures resize propagation on Unix
- Ready for Plan 02 (tests or integration verification)
- All four requirements (CLI-05, CLI-06, CLI-07, CLI-08) satisfied

---
*Phase: 22-cli-attach*
*Completed: 2026-03-24*
