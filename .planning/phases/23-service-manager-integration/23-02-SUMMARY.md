---
phase: 23-service-manager-integration
plan: 02
subsystem: daemon
tags: [service-manager, cli, kardianos, launchd, dependency-injection]

requires:
  - phase: 23-01
    provides: [ServiceControl, RunDaemon exported from internal/daemon]
provides:
  - cmdDaemon dispatcher (cmd/agenthub-cli/cmd_daemon.go)
  - Updated daemon branch in main.go with sub-subcommand parsing
  - Usage text for daemon install/uninstall/start/stop
affects: [cmd/agenthub-cli]

tech-stack:
  added: []
  patterns: [serviceControlFunc package var for test injection, cmdXxx(args, out) command pattern]

key-files:
  created:
    - cmd/agenthub-cli/cmd_daemon.go
    - cmd/agenthub-cli/cmd_daemon_test.go
  modified:
    - cmd/agenthub-cli/main.go

key-decisions:
  - "Use package-level var serviceControlFunc = daemon.ServiceControl for test injection without interface overhead"
  - "No-args path (agenthub daemon) calls RunDaemon() directly for backward compat with EnsureDaemon spawn"
  - "os.Args[2:] is safe in main.go because cmd == 'daemon' guarantees at least 2 args"

patterns-established:
  - "serviceControlFunc injection: package var approach allows test mock without interface churn"
  - "cmdDaemon follows existing cmdXxx(args []string, out io.Writer) error pattern"

requirements-completed: [SVC-03]

duration: 8min
completed: 2026-03-24
---

# Phase 23 Plan 02: CLI Daemon Subcommands Summary

**`agenthub daemon install/uninstall/start/stop` CLI subcommands wired to kardianos/service via testable serviceControlFunc injection**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-03-24T00:00:00Z
- **Completed:** 2026-03-24T00:08:00Z
- **Tasks:** 1 auto (+ 1 checkpoint pending user verification)
- **Files modified:** 3

## Accomplishments
- Created `cmd_daemon.go` with `cmdDaemon` dispatcher handling install/uninstall/start/stop/run/no-args
- Added `serviceControlFunc` package var for mock injection in tests (no interface boilerplate)
- Updated `main.go` daemon branch to parse sub-subcommands via `cmdDaemon(os.Args[2:], os.Stdout)`
- Updated usage text to document all daemon subcommands
- TDD: 3 tests (ServiceActions subtable, UnknownSubcommand, ServiceControlError) all pass

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Failing tests** - `7ef8073` (test)
2. **Task 1 (GREEN): cmdDaemon dispatcher + main.go update** - `8d6b9aa` (feat)

_Note: Task 2 is a checkpoint:human-verify — pending user verification on macOS._

## Files Created/Modified
- `cmd/agenthub-cli/cmd_daemon.go` - cmdDaemon dispatcher with serviceControlFunc injection point
- `cmd/agenthub-cli/cmd_daemon_test.go` - Unit tests for dispatch via mock injection
- `cmd/agenthub-cli/main.go` - Updated daemon branch + usage text for daemon subcommands

## Decisions Made
- Used package-level `var serviceControlFunc = daemon.ServiceControl` for test injection — avoids defining a new interface just for one function, consistent with Go idiom for command-level testing
- Backward compat path: no-args and "run" both call `daemon.RunDaemon()` so existing `EnsureDaemon` spawn still works

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

The worktree branch `worktree-agent-a081f955` was behind `main` (missing plan 01 commits). Merged `main` into the worktree branch as a prerequisite step before implementing Task 1.

## Next Phase Readiness
- CLI dispatcher complete; ready for Task 2 manual macOS launchd verification
- Checkpoint: user needs to build CLI, run `daemon install/start/stop/uninstall`, verify plist lifecycle

## Self-Check: PASSED

Files created:
- cmd/agenthub-cli/cmd_daemon.go: FOUND
- cmd/agenthub-cli/cmd_daemon_test.go: FOUND
- .planning/phases/23-service-manager-integration/23-02-SUMMARY.md: FOUND

Commits:
- 7ef8073 (test): FOUND
- 8d6b9aa (feat): FOUND

---
*Phase: 23-service-manager-integration*
*Completed: 2026-03-24*
