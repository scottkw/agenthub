---
phase: 24-cli-polish
plan: 01
subsystem: cli
tags: [go, cli, json, flag, daemon]

# Dependency graph
requires: []
provides:
  - "--json flag on cmdList, cmdWebStatus, cmdHealth for machine-readable output"
  - "daemon status subcommand (cmdDaemonStatus) with --json flag"
  - "Updated function signatures: cmdList(client, args, out), cmdHealth(args, out), cmdWebStatus(client, args, out)"
affects: [24-cli-polish]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "flag.NewFlagSet per command for local flag parsing without global state"
    - "json.NewEncoder(out).Encode() for streaming JSON output to io.Writer"
    - "daemon status intercepted before early-exit in main() to route through EnsureDaemon"

key-files:
  created: []
  modified:
    - cmd/agenthub-cli/main.go
    - cmd/agenthub-cli/main_test.go
    - cmd/agenthub-cli/cmd_daemon.go
    - cmd/agenthub-cli/cmd_daemon_test.go

key-decisions:
  - "Used flag.NewFlagSet per command (not flag package globals) to avoid state pollution between test runs"
  - "cmdDaemonStatus added to cmd_daemon.go (not main.go) to keep daemon-related code co-located"
  - "daemon status intercepted in main() before early-exit block; other daemon subcommands bypass EnsureDaemon"
  - "Empty session list returns '[]' not 'null' — guarded by nil check before json.Encode"

patterns-established:
  - "JSON output pattern: flag.NewFlagSet -> fs.Bool('json') -> json.NewEncoder(out).Encode(result)"
  - "Args threading pattern: switch case passes args []string to command functions for local flag parsing"

requirements-completed: [POLISH-01]

# Metrics
duration: 3min
completed: 2026-03-24
---

# Phase 24 Plan 01: --json Flag for CLI Commands Summary

**--json flag added to list/web-status/health/daemon-status commands enabling machine-readable JSON output via flag.NewFlagSet per command**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-24T19:38:01Z
- **Completed:** 2026-03-24T19:41:07Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added `--json` flag to `cmdList`, `cmdWebStatus`, `cmdHealth` with json.NewEncoder output
- Added `cmdDaemonStatus` function with `--json` flag and reachability detection via `client.Health() == nil`
- Updated all function signatures to accept `args []string` for flag parsing
- `daemon status` subcommand intercepted in `main()` before early-exit, routed through EnsureDaemon
- All 21 CLI tests pass; 4 new JSON tests + 4 daemon status tests added

## Task Commits

Each task was committed atomically:

1. **Task 1: Add --json flag to cmdList, cmdWebStatus, cmdHealth** - `aa5979e` (feat)
2. **Task 2: Add daemon status subcommand tests** - `d2b3ad5` (test)

_Note: cmdDaemonStatus implementation was included in Task 1 commit because main.go referenced it; Task 2 added only the test file._

## Files Created/Modified
- `cmd/agenthub-cli/main.go` - Updated cmdList/cmdWebStatus/cmdHealth signatures; daemon status routing; updated usage()
- `cmd/agenthub-cli/main_test.go` - Updated existing call sites; added JSON tests for list/web-status/health
- `cmd/agenthub-cli/cmd_daemon.go` - Added cmdDaemonStatus with --json flag; updated error message to include "status"
- `cmd/agenthub-cli/cmd_daemon_test.go` - Added TestCmdDaemon_Status, _JSON, _Unreachable, _JSON_Unreachable

## Decisions Made
- Used `flag.NewFlagSet` per command rather than package-level flags to avoid global state pollution between tests
- `cmdDaemonStatus` placed in `cmd_daemon.go` to keep daemon-related code co-located
- Empty slice nil check before JSON encode ensures `[]` not `null` for empty session lists

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] cmdDaemonStatus implemented in Task 1 (not Task 2)**
- **Found during:** Task 1 (--json flag implementation)
- **Issue:** main.go referenced `cmdDaemonStatus` in the switch case for "daemon", causing a build failure when running Task 1 tests
- **Fix:** Added `cmdDaemonStatus` implementation to `cmd_daemon.go` during Task 1 instead of Task 2
- **Files modified:** cmd/agenthub-cli/cmd_daemon.go
- **Verification:** `go test ./cmd/agenthub-cli/... -count=1` exits 0
- **Committed in:** aa5979e (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Implementation was moved up but no code was omitted. Task 2 focused on tests only.

## Issues Encountered
- `internal/relay TestHub_SlowClientDisconnected` fails intermittently (pre-existing timing-sensitive test, unrelated to these changes — verified by running on clean HEAD)

## Next Phase Readiness
- All four commands (list, web status, health, daemon status) support --json flag
- JSON output parseable by jq; human output unchanged when --json not passed
- Ready for plan 02 of phase 24-cli-polish

---
*Phase: 24-cli-polish*
*Completed: 2026-03-24*
