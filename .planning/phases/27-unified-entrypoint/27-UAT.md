---
status: complete
phase: 27-unified-entrypoint
source: 27-01-SUMMARY.md
started: 2026-03-25T17:20:00Z
updated: 2026-03-25T17:30:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: Build the binary fresh and run `go test -race -count=1 .` from the project root. All tests pass (28+ CLI tests, 7 attach tests, 7 daemon tests, 5 dispatch tests) with no race conditions detected.
result: pass

### 2. Help Flag Shows Usage
expected: Running `go run . --help` prints a usage message listing all available commands (new, list, kill, rename, attach, web, serve, unserve, health, qr, settings, daemon). Same output for `-h`.
result: pass

### 3. Unknown Command Error
expected: Running `go run . foobar` prints `agenthub: unknown command "foobar"` followed by `Run 'agenthub --help' for usage.` and exits with a non-zero status.
result: pass

### 4. CLI Command Routing
expected: Running `go run . list` routes to the CLI list handler (may require a running daemon — if daemon isn't running, the error should come from the list command logic, NOT from dispatch/routing). This confirms commands route correctly through runCLI().
result: pass

### 5. Dispatch Tests Pass
expected: Running `go test -run TestDispatch -v -count=1 .` shows all 5 dispatch tests passing: TestDispatchHelp, TestDispatchHelpShort, TestDispatchNoArgs, TestDispatchFlagRouting, TestRunCLI_UnknownCommand.
result: pass

### 6. Old CLI Package Still Works
expected: Running `go test -count=1 ./cmd/agenthub-cli/...` still passes — the old package was left untouched for Phase 28 cleanup.
result: pass

## Summary

total: 6
passed: 6
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
