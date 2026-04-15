---
phase: 76-tui-foundation
plan: 03
subsystem: cli
tags: [cli, bubbletea, tui, go, dispatch, tty]

# Dependency graph
requires:
  - phase: 76-tui-foundation/76-02
    provides: tui.Run(client *daemon.DaemonClient) error entry point
provides:
  - cmd_tui.go: cmdTUI function with TTY check, health check, tui.Run call
  - main.go: case "tui" dispatch in CLI switch
  - cmd_cli.go: tui command in usage string
affects: [77-tui-operations, 78-tui-remote]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - TTY check on os.Stdout (not os.Stdin) for full-screen TUI commands
    - Daemon health pre-check before launching alt-screen TUI to avoid blank screen on daemon failure

key-files:
  created:
    - cmd_tui.go
  modified:
    - main.go
    - cmd_cli.go

key-decisions:
  - "TTY check uses os.Stdout.Fd() (not os.Stdin) per UI-SPEC: full-screen TUI occupies stdout, non-TTY stdout means no terminal to render into"
  - "Health check runs before tui.Run to ensure daemon is reachable before entering alt-screen mode"

# Metrics
duration: 1min
completed: 2026-04-15
---

# Phase 76 Plan 03: TUI CLI Wiring Summary

**`agenthub tui` subcommand wired end-to-end: cmd_tui.go performs stdout TTY check and daemon health validation before launching tui.Run(client); main.go dispatches case "tui"; usage string lists tui command**

## Performance

- **Duration:** ~1 min
- **Started:** 2026-04-15T13:00:14Z
- **Completed:** 2026-04-15T13:01:40Z
- **Tasks:** 1
- **Files created/modified:** 3

## Accomplishments

- Created `cmd_tui.go` with `cmdTUI(client *daemon.DaemonClient) error`: checks `term.IsTerminal(int(os.Stdout.Fd()))` and returns descriptive error if non-TTY; calls `client.Health()` to validate daemon connectivity; delegates to `tui.Run(client)`
- Added `case "tui": err = cmdTUI(client)` to `main.go` CLI switch (after `case "daemon":`, before `default:`)
- Added `tui                                         Launch interactive terminal UI` to usage string in `cmd_cli.go` after the `daemon status` line

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Wire agenthub tui subcommand | 12da607 | cmd_tui.go (new), main.go, cmd_cli.go |

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None - cmd_tui.go is fully wired to tui.Run(client) with no placeholder logic.

## Threat Flags

None - no new network endpoints, auth paths, file access patterns, or schema changes introduced. Threat mitigations T-76-DOS (health pre-check) and T-76-INJ (boolean-only TTY/health state in cmd_tui.go) are both implemented as specified.

## Self-Check: PASSED

- cmd_tui.go: FOUND
- `func cmdTUI(client *daemon.DaemonClient) error`: FOUND
- `term.IsTerminal(int(os.Stdout.Fd()))`: FOUND
- `agenthub tui requires a terminal`: FOUND
- `client.Health()`: FOUND
- `tui.Run(client)`: FOUND
- `case "tui":` in main.go: FOUND
- `cmdTUI(client)` in main.go: FOUND
- `tui.*Launch interactive terminal UI` in cmd_cli.go: FOUND
- `go build -o /dev/null .`: PASS
- `go vet .`: PASS
- All tests (`go test ./...`): PASS (11 packages, including internal/tui)
- Commit 12da607: FOUND
