---
phase: 21-cli-session-web-commands
plan: 01
subsystem: cli
tags: [cli, daemon, go, session-management]
dependency_graph:
  requires:
    - internal/daemon (DaemonClient, SessionEngine, API, EnsureDaemon)
    - internal/daemon/process.go (RunDaemon, EnsureDaemon)
    - internal/daemon/socket.go (DefaultSocketPath)
  provides:
    - cmd/agenthub-cli/main.go (CLI binary entry point)
    - cmd/agenthub-cli/main_test.go (CLI command tests)
  affects:
    - Plan 02 extends stubs in this binary with web/serve/unserve/health/qr commands
tech_stack:
  added: []
  patterns:
    - "text/tabwriter for aligned CLI table output"
    - "io.Writer parameter injection for testable stdout output"
    - "error return from cmd functions (not os.Exit) for testability"
key_files:
  created:
    - cmd/agenthub-cli/main.go
    - cmd/agenthub-cli/main_test.go
  modified: []
decisions:
  - "cmd functions return error instead of calling os.Exit directly — main() handles exits; this makes unit testing trivial without process-killing tricks"
  - "io.Writer parameter on cmdNew/cmdList — allows tests to capture stdout via bytes.Buffer; main() passes os.Stdout"
  - "Used cat as test CLI (not claude) — cat is always present on macOS/Linux and avoids CI dependency on installed AI CLIs"
  - "Session IDs are 32-char hex strings (not UUIDs) — test asserts len=32 and hex chars only"
  - "workDir=/tmp in tests (not /tmp/myproject) — nonexistent workDir causes PTY fork/exec to fail"
metrics:
  duration: ~15min
  completed: 2026-03-24
  tasks_completed: 2
  files_created: 2
  files_modified: 0
---

# Phase 21 Plan 01: CLI Binary with Session Commands Summary

**One-liner:** Standalone Go CLI binary at cmd/agenthub-cli/ with new/list/kill/rename session commands, daemon auto-start via EnsureDaemon, and 7 passing integration tests against a real test daemon.

## What Was Built

`cmd/agenthub-cli/main.go` — entry point for the `agenthub` CLI binary. Dispatches on `os.Args[1]` to:
- `daemon` — calls `daemon.RunDaemon()` (daemon sub-command, enables EnsureDaemon auto-spawn)
- `new <agent> <path>` — creates session, prints 32-char hex ID to stdout
- `list` — prints tabwriter-formatted table with ID/NAME/AGENT/STATUS columns
- `kill <id>` — terminates session silently (exit 0)
- `rename <id> <name>` — renames session silently (exit 0)
- `web`, `serve`, `unserve`, `health`, `qr` — stub cases for Plan 02

`cmd/agenthub-cli/main_test.go` — 7 tests against a real daemon on a short `/tmp/aht{pid}_{seq}.sock` path:
- `TestCmdNew_Success`, `TestCmdNew_MissingArgs`
- `TestCmdList_Empty`, `TestCmdList_WithSessions`
- `TestCmdKill_Success`
- `TestCmdRename_Success`, `TestCmdRename_MissingArgs`

## Decisions Made

1. **cmd functions return error** — instead of calling `os.Exit(1)` directly, each `cmdX()` function returns an error. `main()` prints to stderr and calls `os.Exit(1)`. This makes unit testing straightforward.

2. **io.Writer injection on cmdNew/cmdList** — functions that produce stdout output accept an `io.Writer` parameter. Tests pass `&bytes.Buffer{}`; `main()` passes `os.Stdout`.

3. **cat as test CLI** — `claude` is not installed in CI. Using `cat` (always available) with `/tmp` as workDir avoids test infrastructure dependencies.

4. **Session IDs are 32-char hex** — the daemon generates 16 random bytes encoded as hex (not UUID format). Test assertions updated accordingly.

## Verification Results

```
go build ./cmd/agenthub-cli/  → BUILD OK
go test ./cmd/agenthub-cli/... → ok (7/7 tests pass)
go test ./internal/daemon/...  → ok (existing tests unbroken)
go vet ./cmd/agenthub-cli/...  → no issues
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Session ID assertion used UUID format instead of 32-char hex**
- **Found during:** Task 2 (TDD GREEN phase)
- **Issue:** Plan specified "UUID" (36-char with dashes), but daemon generates 32-char lowercase hex strings
- **Fix:** Updated TestCmdNew_Success to assert len=32 and hex-only characters
- **Files modified:** cmd/agenthub-cli/main_test.go
- **Commit:** 43e9b19

**2. [Rule 1 - Bug] Test used nonexistent workDir /tmp/myproject**
- **Found during:** Task 2 (TDD GREEN phase)
- **Issue:** PTY fork/exec fails when workDir does not exist; `/tmp/myproject` was not created
- **Fix:** Changed test workDir to `/tmp` which always exists
- **Files modified:** cmd/agenthub-cli/main_test.go
- **Commit:** 43e9b19

**3. [Rule 1 - Bug] TestCmdNew_Success used "claude" as agent CLI**
- **Found during:** Task 2 (TDD GREEN phase)
- **Issue:** Claude CLI not installed at expected path in test environment
- **Fix:** Changed to "cat" (universally available on macOS/Linux) per existing api_test.go pattern
- **Files modified:** cmd/agenthub-cli/main_test.go
- **Commit:** 43e9b19

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 | a3c527f | feat(21-01): create CLI binary with session commands and daemon mode |
| Task 2 | 43e9b19 | test(21-01): add session command tests against real test daemon |

## Self-Check: PASSED
