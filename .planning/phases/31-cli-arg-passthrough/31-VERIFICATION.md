---
phase: 31-cli-arg-passthrough
verified: 2026-03-25T00:00:00Z
status: passed
score: 4/4 must-haves verified
re_verification: false
---

# Phase 31: CLI Arg Passthrough Verification Report

**Phase Goal:** Users can pass extra flags to agents from the CLI using the `--` separator
**Verified:** 2026-03-25
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `agenthub new claude /path -- --model claude-opus-4-5` starts a session with extra flags forwarded to the PTY process | VERIFIED | `splitDashDash` in main.go partitions args; `cmdNew` passes `extraArgs` to `client.CreateSession`; engine.go line 51 assigns `Args: args` to PTY session struct |
| 2 | `agenthub new claude /path` with no `--` continues to work identically to before | VERIFIED | `splitDashDash` returns `nil` for `after` when no `--` present; existing tests `TestCmdNew_Success` and `TestCmdNew_MissingArgs` pass with `nil` extraArgs |
| 3 | Args after `--` are passed as `[]string` token array, not a raw shell string | VERIFIED | `splitDashDash` returns `[]string` slice; `cmdNew` signature is `extraArgs []string`; no shell interpolation at any layer |
| 4 | Go tests cover `cmdNew` `--` separator parsing with and without trailing args | VERIFIED | `TestSplitDashDash` (5 sub-cases), `TestCmdNew_WithExtraArgs`, `TestCmdNew_NoSeparator` all present and passing |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `main.go` | `splitDashDash` helper and updated `runCLI` dispatch | VERIFIED | Lines 63-73: `func splitDashDash(args []string) (before, after []string)`; line 76: `before, extraArgs := splitDashDash(args)`; line 108: `cmdNew(client, cmdArgs, extraArgs, os.Stdout)` |
| `cmd_cli.go` | Updated `cmdNew` accepting `extraArgs` parameter | VERIFIED | Line 51: `func cmdNew(client *daemon.DaemonClient, args []string, extraArgs []string, out io.Writer) error`; line 57: `client.CreateSession(agent, name, workDir, extraArgs)` |
| `cmd_cli_test.go` | Tests for `splitDashDash`, `cmdNew` with/without extraArgs | VERIFIED | `TestSplitDashDash` (line 460), `TestCmdNew_WithExtraArgs` (line 506), `TestCmdNew_NoSeparator` (line 520) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `main.go:runCLI` | `cmd_cli.go:cmdNew` | `extraArgs` parameter from `splitDashDash` result | WIRED | `cmdNew(client, cmdArgs, extraArgs, os.Stdout)` at line 108 |
| `cmd_cli.go:cmdNew` | `client.CreateSession` | `extraArgs` forwarded as args parameter | WIRED | `client.CreateSession(agent, name, workDir, extraArgs)` at line 57 |
| `client.CreateSession` | `engine.CreateSession` | HTTP POST /sessions with `req.Args` | WIRED | api.go line 158 calls `a.engine.CreateSession(..., req.Args, nil)` |
| `engine.CreateSession` | PTY process | `Args: args` in session struct | WIRED | engine.go line 51: `Args: args` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `cmd_cli.go:cmdNew` | `extraArgs []string` | CLI via `splitDashDash` in `runCLI` | Yes — tokens from OS args slice | FLOWING |
| `engine.go:CreateSession` | `args []string` | Passed from API handler via `req.Args` | Yes — forwarded from HTTP request body | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `splitDashDash` all boundary cases | `go test . -run TestSplitDashDash -v` | 5 sub-tests PASS | PASS |
| `cmdNew` with extraArgs | `go test . -run TestCmdNew_WithExtraArgs -v` | PASS (32-char UUID returned) | PASS |
| `cmdNew` with nil extraArgs (backward compat) | `go test . -run TestCmdNew_NoSeparator -v` | PASS | PASS |
| Full suite regression | `go test ./...` | 6 packages ok, 0 failures | PASS |
| Clean build | `go build ./...` | Exit 0, no errors | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| ARGS-01 | 31-01-PLAN.md | User can pass extra arguments to an agent via `agenthub new <agent> -- --flag value` | SATISFIED | `splitDashDash` + `cmdNew` extraArgs + `CreateSession` forwarding verified end-to-end; REQUIREMENTS.md line 59 marks Phase 31 as Complete |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd_cli.go` | 140 | `"Tailscale IP not available"` | Info | Legitimate runtime error message in unrelated function; not a stub |

No blockers or warnings found in the modified files (`main.go`, `cmd_cli.go`, `cmd_cli_test.go`).

### Human Verification Required

None. All success criteria are verifiable programmatically:
- The `--` separator parsing is a pure function with comprehensive unit tests
- The args forwarding chain is traceable through static code analysis
- The test suite exercises the full path including real daemon IPC

### Gaps Summary

No gaps. All four observable truths are verified. The implementation is complete, substantive, wired, and data flows end-to-end from the CLI OS args slice through `splitDashDash`, `cmdNew`, `DaemonClient.CreateSession`, the HTTP API, `SessionEngine.CreateSession`, and finally into the PTY session `Args` field (wired in Phase 30).

---

_Verified: 2026-03-25_
_Verifier: Claude (gsd-verifier)_
