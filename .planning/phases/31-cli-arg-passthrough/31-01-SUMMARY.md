---
phase: 31-cli-arg-passthrough
plan: 01
subsystem: cli
tags: [cli, args, passthrough, tdd]
dependency_graph:
  requires: [30-backend-args-wiring]
  provides: [CLI arg passthrough via -- separator]
  affects: [main.go, cmd_cli.go, cmd_cli_test.go]
tech_stack:
  added: []
  patterns: [splitDashDash helper, extraArgs parameter forwarding]
key_files:
  created: []
  modified:
    - main.go
    - cmd_cli.go
    - cmd_cli_test.go
decisions:
  - splitDashDash returns nil (not empty slice) when no -- present, matching Go idiom for "not provided"
  - extraArgs forwarded directly to CreateSession (Phase 30 already wired backend to pass args to PTY)
  - Usage text updated to reflect positional <agent> <path> syntax with optional -- trailer
metrics:
  duration: 2m 9s
  completed_date: "2026-03-26"
  tasks_completed: 2
  files_modified: 3
---

# Phase 31 Plan 01: CLI Arg Passthrough Summary

**One-liner:** splitDashDash helper + cmdNew extraArgs parameter forwards post-`--` tokens to agent PTY via CreateSession.

## What Was Built

Added `splitDashDash()` to `main.go` that partitions a CLI args slice at the first `--` element. Updated `runCLI` to call it and extract `extraArgs`, then pass them to `cmdNew`. Updated `cmdNew` signature in `cmd_cli.go` to accept `extraArgs []string` and forward them to `client.CreateSession` instead of the previous hardcoded `nil`. Updated usage text to show `new <agent> <path> [-- <extra-args>...]`.

Users can now run: `agenthub new claude /path/to/project -- --model claude-opus-4-5`

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Add failing tests for splitDashDash and cmdNew with extraArgs | 84cd075 | cmd_cli_test.go |
| 1 (GREEN) | Add splitDashDash helper and update cmdNew to forward extraArgs | 3d3b741 | main.go, cmd_cli.go, cmd_cli_test.go |
| 2 | Full suite regression — all packages green | (no separate commit, verification only) | — |

## Deviations from Plan

None — plan executed exactly as written.

## Decisions Made

- `splitDashDash` returns `(args, nil)` when no `--` present (not empty slice), matching the Go idiom that nil slice means "not provided vs provided but empty". The test for "no separator" verifies `after == nil`.
- Existing `cmdNew` test calls updated to pass `nil` as third argument — matches backward compat guarantee.
- Task 2 (regression) required no code changes; all packages passed on first run after Task 1.

## Known Stubs

None — all data flows are wired end-to-end. extraArgs passes from CLI through cmdNew to CreateSession which passes them to the PTY process (wired in Phase 30).

## Self-Check: PASSED

- `/Users/ken/dev/agenthub/.claude/worktrees/agent-af6f7466/main.go` contains `func splitDashDash` and `before, extraArgs := splitDashDash(args)` and `cmdNew(client, cmdArgs, extraArgs, os.Stdout)`
- `/Users/ken/dev/agenthub/.claude/worktrees/agent-af6f7466/cmd_cli.go` contains `func cmdNew(client *daemon.DaemonClient, args []string, extraArgs []string, out io.Writer) error` and `client.CreateSession(agent, name, workDir, extraArgs)` and usage `new <agent> <path> [-- <extra-args>...]`
- `/Users/ken/dev/agenthub/.claude/worktrees/agent-af6f7466/cmd_cli_test.go` contains `TestSplitDashDash`, `TestCmdNew_WithExtraArgs`, `TestCmdNew_NoSeparator`
- Commits 84cd075 (RED) and 3d3b741 (GREEN) exist in git log
- `go build ./...` exits 0
- `go test . -run "TestSplitDashDash|TestCmdNew" -v` passes all 9 test cases
- `go test ./...` exits 0 across all 6 packages
