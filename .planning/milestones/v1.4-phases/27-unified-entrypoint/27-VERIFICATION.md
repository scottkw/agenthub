---
phase: 27-unified-entrypoint
verified: 2026-03-25T18:12:24Z
status: passed
score: 7/7 must-haves verified
---

# Phase 27: Unified Entrypoint Verification Report

**Phase Goal:** A single `agenthub` binary dispatches to GUI, all CLI commands, and daemon mode based on args
**Verified:** 2026-03-25T18:12:24Z
**Status:** passed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `agenthub` with no args routes to GUI path (not panic, not CLI) | VERIFIED | `main()` checks `len(os.Args) == 1` and calls `runGUI()` — confirmed in main.go:25-32 |
| 2 | `agenthub <command>` dispatches to the correct cmd* function for all 13 commands | VERIFIED | `runCLI()` switch in main.go:89-118 covers all 12 top-level command names; `web` internally dispatches to 3 sub-functions; 13 cmd* functions confirmed across cmd_cli.go (13) + cmd_daemon.go (2) + cmd_attach.go (1) |
| 3 | `agenthub daemon <subcommand>` dispatches to cmdDaemon with full service management | VERIFIED | main.go:68-76 handles `daemon` pre-switch for non-status subcommands; `daemon status` falls through to `cmdDaemonStatus` after `EnsureDaemon`; cmd_daemon.go implements install/uninstall/start/stop/run/status |
| 4 | `agenthub --help` prints usage covering GUI launch and all CLI subcommands | VERIFIED | main.go:26-29 detects `--help`/`-h` and calls `usage()`; cmd_cli.go:21-47 contains full help text with "Run with no arguments to launch the desktop GUI" and all 12 command names plus 6 daemon subcommands |
| 5 | All existing CLI tests pass when run from root package | VERIFIED | `go test -count=1 -race . -run TestCmd` passes; cmd_cli_test.go in root contains `testSetup` helper and all CLI tests including `TestCmdNew_Success` |
| 6 | All existing daemon tests pass when run from root package | VERIFIED | cmd_daemon_test.go in root contains `TestCmdDaemon_ServiceActions` using injectable `serviceControlFunc`; passes with `-race` flag |
| 7 | All existing attach tests pass when run from root package | VERIFIED | cmd_attach_test.go in root contains `setupAttachTest` and `TestAttach*`; `safeBuf` mutex fix applied for data race; root package test suite passes with `-race` |

**Score:** 7/7 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `main.go` | Unified dispatch: GUI vs CLI vs help | VERIFIED | Contains `func runGUI()`, `func runCLI(args []string)`, `os.Args[1] == "--help"`, `strings.HasPrefix(os.Args[1], "-")` |
| `cmd_cli.go` | All non-attach, non-daemon CLI command functions + `usage()` | VERIFIED | 13 cmd* functions present; `usage()` contains GUI instruction and all subcommands; 272 lines, substantive |
| `cmd_attach.go` | `cmdAttach` + `attachSession` + `stdinPump` + `wsOutputPump` | VERIFIED | All four functions present plus `makeClientResizeFrame`; 205 lines, fully implemented |
| `cmd_attach_unix.go` | `watchResize` for Unix (SIGWINCH) | VERIFIED | Contains `//go:build !windows`, `func watchResize`, SIGWINCH signal handling |
| `cmd_attach_windows.go` | `watchResize` no-op for Windows | VERIFIED | Contains `//go:build windows`, `func watchResize(_ context.Context, _ *websocket.Conn)` no-op |
| `cmd_daemon.go` | `cmdDaemon` + `cmdDaemonStatus` + `serviceControlFunc` | VERIFIED | `var serviceControlFunc = daemon.ServiceControl`, `func cmdDaemon(`, `func cmdDaemonStatus(` all present |
| `dispatch_test.go` | Tests for ROUTE-01, ROUTE-02, ROUTE-03, CLI-04 | VERIFIED | Contains `TestDispatchHelp`, `TestDispatchHelpShort`, `TestDispatchNoArgs`, `TestDispatchFlagRouting`, `TestRunCLI_UnknownCommand` |
| `cmd_cli_test.go` | Migrated main_test.go CLI tests | VERIFIED | Contains `testSetup`, `TestCmdNew_Success`, and full CLI test suite |
| `cmd_attach_test.go` | Migrated attach tests | VERIFIED | Contains `setupAttachTest`, `TestCmdAttach_MissingArgs`, and attach session tests with `safeBuf` race fix |
| `cmd_daemon_test.go` | Migrated daemon tests | VERIFIED | Contains `TestCmdDaemon_ServiceActions` with mock injection via `serviceControlFunc` |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `main.go` | `cmd_cli.go` | `runCLI` switch dispatches to cmd* functions | WIRED | `case "new": err = cmdNew(...)`, `case "list": err = cmdList(...)` etc. — all 12 cases present |
| `main.go` | `cmd_daemon.go` | `runCLI` daemon case | WIRED | Pre-switch handler at main.go:68-76 calls `cmdDaemon`; switch case calls `cmdDaemonStatus` |
| `main.go` | `cmd_attach.go` | `runCLI` attach case | WIRED | `case "attach": err = cmdAttach(client, cmdArgs)` at main.go:98-99 |
| `dispatch_test.go` | `main.go` | Tests verify dispatch routing via `usage()` call | WIRED | `TestDispatchHelp` calls `usage()` directly; `TestDispatchFlagRouting` tests dispatch condition logic |

---

### Data-Flow Trace (Level 4)

Not applicable — this phase produces a CLI dispatch layer, not a component that renders dynamic data. The cmd* functions pass data to `io.Writer` arguments, not rendering pipelines. Data flow is verified by the test suite (testSetup creates a real daemon API, verifies actual output).

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Root package tests pass with race detector | `go test -count=1 -race -timeout 120s . -run TestDispatch\|TestCmdNew\|TestCmdDaemon_ServiceActions\|TestAttach` | `ok github.com/agenthub/agenthub 2.155s` | PASS |
| Full root package test suite | `go test -count=1 -race -timeout 120s .` | `ok github.com/agenthub/agenthub 6.945s` | PASS |
| `go vet` clean | `go vet ./...` | No output (exit 0) | PASS |
| `cmd/agenthub-cli/` pre-existing race | `go test -count=1 -race ./...` | FAIL in `cmd/agenthub-cli` only — pre-existing data race in `TestAttachSession_LiveOutput`, documented in SUMMARY as known issue; unfixed per plan constraint that directory is untouched until Phase 28 | NOTE (not a Phase 27 gap) |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| ROUTE-01 | 27-01-PLAN.md | User can run `agenthub` with no args to launch GUI | SATISFIED | `main()` checks `len(os.Args) == 1` → `runGUI()`; `TestDispatchNoArgs` verifies condition |
| ROUTE-02 | 27-01-PLAN.md | User can run `agenthub <command>` to execute any CLI command | SATISFIED | `runCLI(os.Args[1:])` called for non-flag args; switch covers all commands; `TestDispatchFlagRouting` verifies |
| ROUTE-03 | 27-01-PLAN.md | User can run `agenthub daemon` to start daemon mode | SATISFIED | `cmd == "daemon"` handled pre-switch (calls `cmdDaemon`) and in-switch (`cmdDaemonStatus`); `TestCmdDaemon_ServiceActions` verifies |
| CLI-01 | 27-01-PLAN.md | All 13 CLI commands work from unified binary | SATISFIED | 12 case labels in `runCLI` switch + pre-switch daemon handler; all cmd* functions in root package |
| CLI-02 | 27-01-PLAN.md | `--json` flag works on applicable commands from unified binary | SATISFIED | `cmdList`, `cmdWebStatus`, `cmdHealth`, `cmdDaemonStatus` all parse `--json` flag; covered by migrated tests |
| CLI-03 | 27-01-PLAN.md | Interactive attach (raw PTY, detach key, resize) works from unified binary | SATISFIED | `cmdAttach` in root package: raw mode, detach key scanning, `watchResize`, `makeClientResizeFrame`; `TestAttach*` pass |
| CLI-04 | 27-01-PLAN.md | `agenthub --help` shows both GUI and CLI usage | SATISFIED | `usage()` in cmd_cli.go:21-47 contains GUI launch instruction + all commands + daemon subcommands + help trailer; `TestDispatchHelp` verifies all strings |

**Orphaned requirements check:** REQUIREMENTS.md also lists CLEAN-01, CLEAN-02, BUILD-01, BUILD-02, BUILD-03 — all mapped to Phase 28/29, not Phase 27. No orphaned requirements for this phase.

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `dispatch_test.go` | 112-113 | `TestRunCLI_UnknownCommand` compares `got := fmt.Sprintf(...)` to `expected := fmt.Sprintf(...)` with identical format strings — the test always passes regardless of main.go content | Info | Test verifies message format string in isolation, not actual `runCLI` output; the real format in main.go:116 is confirmed by code review |

No blocker or warning anti-patterns found. The test weakness noted above is cosmetic — the actual error format in `main.go:116` matches exactly what the test expects, as verified by direct code inspection.

---

### Human Verification Required

None required for automated verification. The following behaviors require a running binary for full UAT but are out of scope for this phase (Phase 29 covers build system):

1. **GUI launch smoke test**
   - Test: Run `agenthub` with no args on macOS
   - Expected: Wails window opens
   - Why human: Cannot launch Wails GUI in test environment

2. **Binary dispatch end-to-end**
   - Test: Run `agenthub list` against a live daemon
   - Expected: Session list printed
   - Why human: Requires running daemon process

Both are integration concerns deferred to Phase 29 (BUILD-03).

---

### Gaps Summary

No gaps found. All 7 must-have truths are verified. All 10 required artifacts exist, are substantive (real implementations, not stubs), and are wired together. All 7 requirement IDs (ROUTE-01/02/03, CLI-01/02/03/04) have implementation evidence. The root package test suite passes with the race detector enabled.

The one known issue — a pre-existing data race in `cmd/agenthub-cli/cmd_attach_test.go` — is explicitly out of scope for Phase 27. The PLAN states "cmd/agenthub-cli/ directory is UNTOUCHED (deletion is Phase 28)". The migrated copy in the root package has the race fixed via the `safeBuf` pattern.

---

_Verified: 2026-03-25T18:12:24Z_
_Verifier: Claude (gsd-verifier)_
