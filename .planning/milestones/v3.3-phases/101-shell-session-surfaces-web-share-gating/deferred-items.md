# Phase 101 Deferred Items

Items discovered during plan execution that are out-of-scope for the current plan but should be tracked.

## From Plan 101-04

- **Pre-existing data race in `TestOpenCodeANSICapture`** (`internal/daemon/opencode_ansi_test.go`)
  — `go test ./internal/daemon -race -count=1` fails with a race detected during the test. The race is in OpenCode ANSI capture code, unrelated to Phase 101-04 (shell sessions). The test passes without `-race`. Out of scope per the executor scope boundary (Plan 101-04 touches `cmd_cli.go`, `main.go`, `internal/tui/*` only; does not touch `internal/daemon/`).
  - Triage owner: future plan or v3.4 polish.
  - Verified pre-existing: 2026-05-12 via plan 101-04 execution.
