---
phase: 77-tui-session-operations
plan: 02
subsystem: tui
tags: [bubbletea, tea-exec, websocket, raw-pty, attach, internal-attach]

# Dependency graph
requires:
  - phase: 77-tui-session-operations (plan 01)
    provides: TUI foundation with session list, keybindings, model/update/view architecture
  - phase: 75-cli-status-bar
    provides: internal/statusbar package used during attach
  - phase: 74-multi-client-fan-out
    provides: relay Hub with fan-out WebSocket support
provides:
  - internal/attach/ package with extracted shared attach logic (AttachSession, StdinPump, WsOutputPump, MakeClientResizeFrame, LockedWriter, WatchResize)
  - tea.ExecCommand implementation for TUI attach flow
  - Enter key dispatches tea.Exec for running sessions
  - attachDoneMsg triggers session list refresh after detach
affects: [77-tui-session-operations plans 03-04, cmd_attach.go consumers]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "tea.ExecCommand interface for terminal handoff (TUI suspends, raw PTY runs, TUI resumes)"
    - "internal/attach/ shared package imported by both CLI and TUI"
    - "Type-assert stdin to *os.File for raw mode/status bar (graceful fallback in tests)"

key-files:
  created:
    - internal/attach/attach.go
    - internal/attach/attach_unix.go
    - internal/attach/attach_windows.go
    - internal/attach/attach_test.go
    - internal/tui/attach.go
    - internal/tui/attach_test.go
  modified:
    - cmd_attach.go
    - cmd_attach_unix.go
    - cmd_attach_windows.go
    - cmd_attach_test.go
    - internal/tui/update.go
    - go.mod
    - go.sum

key-decisions:
  - "Extract shared attach logic to internal/attach/ (Option A from RESEARCH.md) — clean import by both CLI and TUI, no duplication"
  - "cmd_attach_unix.go and cmd_attach_windows.go become thin wrappers delegating to attach.WatchResize"
  - "attachCmd.Run() type-asserts stdin to *os.File with ok pattern — graceful fallback when stdin is not a file (test path)"

patterns-established:
  - "tea.ExecCommand pattern: struct with SetStdin/SetStdout/SetStderr + Run() for terminal handoff"
  - "Shared internal package extraction: move package-main functions to internal/ for cross-package import"

requirements-completed: [TUI-03]

# Metrics
duration: 13min
completed: 2026-04-15
---

# Phase 77 Plan 02: Attach ExecCommand & Shared Attach Package Summary

**Extracted CLI attach logic into shared internal/attach/ package and implemented tea.ExecCommand for TUI session attach via Enter key**

## Performance

- **Duration:** 13 min
- **Started:** 2026-04-15T15:30:11Z
- **Completed:** 2026-04-15T15:43:19Z
- **Tasks:** 2
- **Files modified:** 14

## Accomplishments
- Extracted 5 functions + 1 type from cmd_attach.go into internal/attach/ package (AttachSession, StdinPump, WsOutputPump, MakeClientResizeFrame, LockedWriter, WatchResize)
- Created attachCmd implementing tea.ExecCommand — full PTY attach with WebSocket dial, raw mode, status bar, I/O pumps
- Wired Enter key to tea.Exec dispatch for running sessions, with "Session not available" toast for errored sessions
- All 200+ existing tests pass without regression after extraction

## Task Commits

Each task was committed atomically:

1. **Task 1: Extract shared attach logic into internal/attach/ package** - `d371b27` (feat)
2. **Task 2: Implement attachCmd ExecCommand and wire Enter key to tea.Exec** - `1fd5e18` (feat)

## Files Created/Modified
- `internal/attach/attach.go` - Shared attach logic: AttachSession, StdinPump, WsOutputPump, MakeClientResizeFrame, LockedWriter
- `internal/attach/attach_unix.go` - WatchResize with SIGWINCH handler for Unix
- `internal/attach/attach_windows.go` - WatchResize no-op stub for Windows
- `internal/attach/attach_test.go` - Unit tests for LockedWriter and MakeClientResizeFrame
- `internal/tui/attach.go` - attachCmd struct implementing tea.ExecCommand with full attach flow
- `internal/tui/attach_test.go` - Tests for ExecCommand interface, dispatch, errored session, done handling
- `cmd_attach.go` - Replaced inline functions with attach.* calls
- `cmd_attach_unix.go` - Delegates to attach.WatchResize
- `cmd_attach_windows.go` - Delegates to attach.WatchResize
- `cmd_attach_test.go` - Updated to use attach.AttachSession and attach.MakeClientResizeFrame
- `internal/tui/update.go` - Replaced attach placeholder with tea.Exec dispatch

## Decisions Made
- Chose Option A (extract to internal/attach/) over Option B (duplication) from RESEARCH.md — eliminates code duplication between CLI and TUI attach paths
- cmd_attach_unix.go and cmd_attach_windows.go kept as thin wrappers to maintain the build-tag compilation pattern
- attachCmd.Run() uses type assertion with ok pattern for stdin — allows graceful test path without *os.File

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- internal/attach/ package ready for any future attach consumers
- TUI attach flow complete — Enter key dispatches full PTY attach, Ctrl-\ detach returns to TUI
- Ready for Plan 77-03 (new-session modal) and Plan 77-04 (modal form fields)

---
*Phase: 77-tui-session-operations*
*Completed: 2026-04-15*
