---
phase: 27-unified-entrypoint
plan: 01
subsystem: cli
tags: [go, cli, dispatch, wails, daemon, attach, testing]

# Dependency graph
requires:
  - phase: v1.3
    provides: cmd/agenthub-cli/ package with all CLI commands, daemon, and attach logic
provides:
  - Unified root package: main.go with runGUI()/runCLI() dispatch, no-args→GUI, --help→usage(), command→CLI
  - cmd_cli.go: all 13 CLI command functions + rewritten usage() per UI-SPEC contract
  - cmd_attach.go: cmdAttach, attachSession, stdinPump, wsOutputPump, makeClientResizeFrame
  - cmd_attach_unix.go / cmd_attach_windows.go: platform-specific watchResize
  - cmd_daemon.go: cmdDaemon, cmdDaemonStatus, serviceControlFunc injectable
  - Full test suite in root package: cmd_cli_test.go, cmd_attach_test.go, cmd_daemon_test.go, dispatch_test.go
  - Dispatch tests verifying ROUTE-01/02/03 and CLI-04 help text contract
affects: [28-cleanup, future-phases-using-single-binary]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Unified binary dispatch: len(os.Args)==1 or HasPrefix('-')→GUI, help flags→usage(), commands→runCLI()"
    - "safeBuf pattern: mutex-protected bytes.Buffer for concurrent test stdout assertions"
    - "runGUI() / runCLI() separation keeps Wails and CLI dispatch cleanly separated"

key-files:
  created:
    - main.go (rewritten with unified dispatch)
    - cmd_cli.go
    - cmd_attach.go
    - cmd_attach_unix.go
    - cmd_attach_windows.go
    - cmd_daemon.go
    - cmd_cli_test.go
    - cmd_attach_test.go
    - cmd_daemon_test.go
    - dispatch_test.go
  modified:
    - main.go

key-decisions:
  - "Dispatch rule: no args or flag (not --help/-h) → GUI; --help/-h → usage(); word → runCLI()"
  - "daemon status fall-through: most daemon subcommands go directly to cmdDaemon; only status needs EnsureDaemon first"
  - "safeBuf (mutex-protected buffer) used in TestAttachSession_LiveOutput to fix pre-existing data race in test"
  - "cmd/agenthub-cli/ left untouched — deletion is Phase 28"

patterns-established:
  - "runGUI() extracted from main() to isolate Wails options from CLI dispatch"
  - "runCLI(args []string) takes args without binary name (os.Args[1:]) for clean testability"
  - "cmdArgs rename: avoids shadowing the args parameter in runCLI switch body"

requirements-completed: [ROUTE-01, ROUTE-02, ROUTE-03, CLI-01, CLI-02, CLI-03, CLI-04]

# Metrics
duration: 18min
completed: 2026-03-25
---

# Phase 27 Plan 01: Unified Entrypoint Summary

**Merged cmd/agenthub-cli/ into root package: single agenthub binary dispatches no-args→GUI, flags→GUI, --help→usage(), commands→runCLI() with full migrated+new test suite**

## Performance

- **Duration:** 18 min
- **Started:** 2026-03-25T16:52:17Z
- **Completed:** 2026-03-25T17:10:11Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments

- Unified main.go dispatch: no args or non-help flag → runGUI(), --help/-h → usage(), any command → runCLI()
- All 13 CLI command functions (cmdNew, cmdList, cmdKill, cmdRename, cmdAttach, cmdWeb, cmdServe, cmdUnserve, cmdHealth, cmdQR, cmdSettings, cmdDaemon, cmdDaemonStatus) now live in root package
- Full test suite (28+ CLI tests, 7 attach tests, 7 daemon tests, 5 dispatch tests) migrated and passing in root package with -race flag

## Task Commits

Each task was committed atomically:

1. **Task 1: Copy CLI source files to root package and wire unified dispatch in main.go** - `a497f23` (feat)
2. **Task 2: Migrate test files and add dispatch routing tests** - `b5ff55b` (test)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `main.go` - Rewritten with unified dispatch: runGUI(), runCLI(args), dispatch logic
- `cmd_cli.go` - All non-attach, non-daemon CLI functions + rewritten usage() per UI-SPEC CLI-04
- `cmd_attach.go` - cmdAttach, attachSession, stdinPump, wsOutputPump, makeClientResizeFrame
- `cmd_attach_unix.go` - watchResize with SIGWINCH (build tag !windows)
- `cmd_attach_windows.go` - watchResize no-op (build tag windows)
- `cmd_daemon.go` - serviceControlFunc (injectable), cmdDaemon, cmdDaemonStatus
- `cmd_cli_test.go` - Migrated from cmd/agenthub-cli/main_test.go; testSetup + all CLI tests
- `cmd_attach_test.go` - Migrated from cmd/agenthub-cli/cmd_attach_test.go; safeBuf fix applied
- `cmd_daemon_test.go` - Migrated from cmd/agenthub-cli/cmd_daemon_test.go
- `dispatch_test.go` - New: TestDispatchHelp, TestDispatchHelpShort, TestDispatchNoArgs, TestDispatchFlagRouting, TestRunCLI_UnknownCommand

## Decisions Made

- Dispatch uses `strings.HasPrefix(os.Args[1], "-")` for flag detection — non-help flags route to GUI (e.g., future `--debug` flag)
- `daemon status` falls through to EnsureDaemon because it needs a running daemon; other daemon subcommands (install/uninstall/start/stop/run) skip EnsureDaemon
- `cmdArgs := args[1:]` rename in runCLI avoids shadowing the `args` parameter in the switch body
- Unknown commands print `"agenthub: unknown command %q\nRun 'agenthub --help' for usage.\n"` (no re-printing full usage)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed pre-existing data race in TestAttachSession_LiveOutput**
- **Found during:** Task 2 (migrate test files)
- **Issue:** `bytes.Buffer` used as both stdout writer (goroutine via wsOutputPump) and reader (polling loop in test) without synchronization — data race detected by -race
- **Fix:** Introduced `safeBuf` type (mutex-protected bytes.Buffer) implementing `io.Writer` and `String()` with locking
- **Files modified:** cmd_attach_test.go
- **Verification:** `go test -count=1 -race -run TestAttachSession_LiveOutput .` passes
- **Committed in:** b5ff55b (Task 2 commit)
- **Note:** Same race exists in original cmd/agenthub-cli/cmd_attach_test.go (untouched per plan — deletion is Phase 28)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug fix)
**Impact on plan:** Pre-existing race in test infrastructure fixed in migrated copy. No scope creep.

## Issues Encountered

- Pre-existing data race in `TestAttachSession_LiveOutput` (also present in original `cmd/agenthub-cli/` — not fixed there per plan constraint that directory is untouched until Phase 28)
- Pre-existing flaky test `TestHub_SlowClientDisconnected` in `internal/relay` — unrelated to this plan, not fixed

## Known Stubs

None — all functions wired to real implementations.

## Next Phase Readiness

- Root package now contains all CLI command functions and unified dispatch
- `cmd/agenthub-cli/` still exists and its tests still pass independently (as required by plan)
- Phase 28 can safely delete `cmd/agenthub-cli/` — all logic is now in root package
- `go vet ./...` and root package `go test -race .` both pass

## Self-Check: PASSED

- All 10 files exist: main.go, cmd_cli.go, cmd_attach.go, cmd_attach_unix.go, cmd_attach_windows.go, cmd_daemon.go, cmd_cli_test.go, cmd_attach_test.go, cmd_daemon_test.go, dispatch_test.go
- Commit a497f23 found: feat(27-01): copy CLI source files to root package and wire unified dispatch in main.go
- Commit b5ff55b found: test(27-01): migrate CLI test files to root package and add dispatch routing tests

---
*Phase: 27-unified-entrypoint*
*Completed: 2026-03-25*
