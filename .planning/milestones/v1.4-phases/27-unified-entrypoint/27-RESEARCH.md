# Phase 27: Unified Entrypoint - Research

**Researched:** 2026-03-25
**Domain:** Go multi-mode binary dispatch, Wails v2 CLI integration
**Confidence:** HIGH

## Summary

Phase 27 is a pure code-restructuring exercise with no new functionality. All the pieces already exist and work:
`main.go` (root package) already handles GUI and a single `daemon` dispatch. `cmd/agenthub-cli/main.go`
contains a fully tested 13-command CLI dispatcher. The goal is to merge the CLI dispatcher into the root
`main.go` so one binary handles all three modes.

The strategy is locked in STATE.md: `len(os.Args) == 1 || os.Args[1] starts with "-"` launches GUI;
otherwise dispatches to CLI. The CLI functions (`cmdNew`, `cmdList`, etc.) live in `package main` inside
`cmd/agenthub-cli/` — they cannot be imported. They must be **copied** into the root package. Tests in
`cmd/agenthub-cli/` call these functions directly and must be migrated alongside them.

The phase ends when `go test ./...` is green with the CLI dispatch logic living in the root package, and
`cmd/agenthub-cli/` still intact (deletion is Phase 28's job).

**Primary recommendation:** Copy all non-`main()` source files from `cmd/agenthub-cli/` into the root
package, add a `runCLI(args []string)` dispatcher to root `main.go`, and migrate the test files.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ROUTE-01 | `agenthub` (no args) launches desktop GUI | Covered: existing Wails run block in main.go, dispatch guard needed |
| ROUTE-02 | `agenthub <command>` executes CLI command | Covered: existing CLI switch in agenthub-cli/main.go |
| ROUTE-03 | `agenthub daemon` starts daemon mode | Covered: cmdDaemon() already exists in agenthub-cli |
| CLI-01 | All 13 CLI commands work from unified binary | Covered: all cmd* functions exist and are tested |
| CLI-02 | `--json` flag works from unified binary | Covered: flag.NewFlagSet used per-command already |
| CLI-03 | Interactive attach works from unified binary | Covered: cmdAttach + attachSession + watchResize all exist |
| CLI-04 | `agenthub --help` shows GUI + CLI usage | Covered: existing usage() function, needs GUI preamble |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/wailsapp/wails/v2` | v2.10.2 | Desktop GUI framework | Already in use, drives root binary |
| `golang.org/x/term` | v0.41.0 | Raw terminal mode, size detection | Already in use for attach |
| `github.com/coder/websocket` | v1.8.14 | WebSocket relay for attach | Already in use |
| `github.com/kardianos/service` | v1.2.4 | Daemon service control | Already in use for install/uninstall/start/stop |
| `github.com/skip2/go-qrcode` | v0.0.0-20200617195104 | QR code terminal rendering | Already in use |

No new dependencies required. All packages are already in `go.mod`.

### Supporting
No additional packages needed — this is a code-move operation.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Manual copy of CLI files | Shared `internal/cli` package | Requires restructuring; not worth it for a merge-then-delete operation |
| `cobra` CLI framework | Current `flag.NewFlagSet` per command | cobra adds a dep; existing flag-per-command pattern is clean and well-tested |

**Installation:** No new packages to install.

## Architecture Patterns

### Recommended Final Structure (root package after Phase 27)

```
/ (root package main)
├── main.go              # Dispatch: GUI vs CLI; imports all cmd* functions
├── app.go               # Wails App struct (unchanged)
├── tray.go / tray_*.go  # Tray (unchanged)
├── assets_prod.go       # Wails assets (unchanged)
├── assets_stub.go       # Test stub (unchanged)
├── app_test.go          # GUI tests (unchanged)
├── cmd_attach.go        # MOVED from cmd/agenthub-cli/
├── cmd_attach_unix.go   # MOVED from cmd/agenthub-cli/
├── cmd_attach_windows.go# MOVED from cmd/agenthub-cli/
├── cmd_attach_test.go   # MOVED from cmd/agenthub-cli/
├── cmd_daemon.go        # MOVED from cmd/agenthub-cli/
├── cmd_daemon_test.go   # MOVED from cmd/agenthub-cli/
├── cmd_cli.go           # NEW: all non-attach, non-daemon cmd* functions + usage()
└── cmd_cli_test.go      # MOVED from cmd/agenthub-cli/main_test.go
```

`cmd/agenthub-cli/` remains untouched until Phase 28.

### Pattern 1: Multi-Mode Dispatch in main()

**What:** Check `os.Args` to decide mode at process startup.
**When to use:** Single-binary tools that behave differently based on invocation.

```go
// Source: STATE.md locked decision + existing main.go pattern
func main() {
    // GUI mode: no args, or first arg is a flag (e.g. --help)
    if len(os.Args) == 1 || strings.HasPrefix(os.Args[1], "-") {
        if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
            usage()
            return
        }
        runGUI()
        return
    }
    // CLI mode
    runCLI(os.Args[1:])
}
```

**Critical detail:** `--help` must be caught before GUI dispatch — otherwise Wails tries to open a window.

### Pattern 2: Preserve Daemon Backward Compat

The existing `main.go` has:
```go
if len(os.Args) > 1 && os.Args[1] == "daemon" {
    daemon.RunDaemon()
    return
}
```
This must NOT be the only daemon handling. The full `cmdDaemon()` function (with install/uninstall/start/stop/run/status subcommands) must be wired in. The existing root `main.go` daemon handling is a stripped-down stub that only calls `RunDaemon()` — it handles the `EnsureDaemon` spawning path (`exe daemon` with no subcommand) but not service management.

### Pattern 3: Preserve `package main` for cli functions

The CLI functions are in `package main` in `cmd/agenthub-cli/`. They cannot be imported. Moving them to the root `package main` is the only option. Change only the `package` declaration — no other changes needed to the function bodies.

### Pattern 4: Build Tag Isolation for Wails

The root package uses `//go:build wailsassets` / `//go:build !wailsassets` for assets. The CLI code must not break non-Wails builds. Since CLI code only uses stdlib and existing internal packages, there are no new build tag concerns.

### Anti-Patterns to Avoid

- **Don't create a new `internal/cli` package for this phase.** The migration is temporary — Phase 28 deletes the CLI package and Phase 29 does cleanup. Adding an internal package now creates unnecessary churn.
- **Don't modify `cmd/agenthub-cli/` during Phase 27.** Keep it intact as a reference and to allow `go test ./cmd/agenthub-cli/...` to still pass. Deletion is Phase 28.
- **Don't break the existing daemon stub in main.go.** The current `daemon.RunDaemon()` call in main.go is invoked by `EnsureDaemon` when it spawns a detached daemon via `exec.Command(exe, "daemon")`. The unified dispatch must preserve this — `agenthub daemon` with no subcommand (or with `run` subcommand) must still call `daemon.RunDaemon()`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI argument parsing | Custom arg parser | `flag.NewFlagSet` per command | Already in use throughout CLI package; consistent behavior |
| Service lifecycle control | Platform-specific code | `serviceControlFunc` (kardianos/service) | Already tested, cross-platform |
| Terminal raw mode | Manual termios calls | `golang.org/x/term` | Already used in cmdAttach |
| WebSocket for attach | Custom TCP relay | `github.com/coder/websocket` + relay package | Already built and tested |

## Runtime State Inventory

> Not applicable — this is a code restructuring phase. No stored data, live service config, OS-registered state, secrets, or build artifacts use the string "agenthub-cli" as a runtime key. The binary name change (from `agenthub-cli` to `agenthub`) is Phase 28's concern.

**Stored data:** None — no database records reference the binary name.
**Live service config:** The kardianos/service daemon is registered under a service name derived from the daemon package, not the binary name.
**OS-registered state:** None — Phase 27 adds dispatch logic only; no new service registration.
**Secrets/env vars:** None affected.
**Build artifacts:** `cmd/agenthub-cli/` directory survives Phase 27; cleanup is Phase 28.

## Common Pitfalls

### Pitfall 1: Daemon Dispatch Ambiguity

**What goes wrong:** `agenthub daemon` should go to `cmdDaemon()` (full subcommand dispatch), but the existing root `main.go` only calls `daemon.RunDaemon()` for `daemon` arg.
**Why it happens:** The root `main.go` was written before service management was added to `agenthub-cli`.
**How to avoid:** Replace the existing `if os.Args[1] == "daemon"` stub in `main.go` with the full `runCLI()` path. The `cmdDaemon()` function already handles `len(args) == 0 → RunDaemon()` as a fallback, so backward compat with EnsureDaemon's spawn pattern is preserved.
**Warning signs:** `agenthub daemon install` runs GUI instead of installing service.

### Pitfall 2: `--help` Launching the GUI

**What goes wrong:** `strings.HasPrefix(os.Args[1], "-")` passes `--help` to Wails, opening a window briefly before printing help.
**Why it happens:** `--help` starts with `-`, and the dispatch logic could short-circuit to GUI mode.
**How to avoid:** Explicitly check for `-h` and `--help` before delegating to GUI. The `usage()` function already exists in `cmd/agenthub-cli/main.go` — copy it to root and call it for help flags.
**Warning signs:** Running `agenthub --help` opens a window.

### Pitfall 3: Test File Package Conflicts

**What goes wrong:** `cmd_cli_test.go` (moved from `cmd/agenthub-cli/main_test.go`) references `testSetup` and `testSetupWithWebServer` helpers that were in the same file. After splitting across multiple files, the helpers may be missing.
**Why it happens:** Go test helpers in `_test.go` files are only visible within the same package's test compilation unit.
**How to avoid:** Keep `testSetup` and `testSetupWithWebServer` in a shared `cli_test_helpers_test.go` file (or in `cmd_cli_test.go` itself), accessible to `cmd_attach_test.go` and `cmd_daemon_test.go` since all are `package main` tests.
**Warning signs:** `undefined: testSetup` compile errors in `cmd_attach_test.go` or `cmd_daemon_test.go`.

### Pitfall 4: `serviceControlFunc` Variable Scope

**What goes wrong:** The `var serviceControlFunc = daemon.ServiceControl` package-level var is used by `cmd_daemon.go` and reassigned in `cmd_daemon_test.go`. If moved to root package without care, the declaration may conflict with existing root-package vars.
**Why it happens:** Root package `app.go` uses function injection patterns too (e.g. `statusFunc`).
**How to avoid:** Direct copy — the variable name `serviceControlFunc` is distinct from anything in the current root package. No rename needed.
**Warning signs:** `serviceControlFunc redeclared in this block` compile error.

### Pitfall 5: Wails Build Tags and `wails dev`

**What goes wrong:** Adding CLI files to the root package that import non-GUI packages may affect `wails dev` hot-reload behavior.
**Why it happens:** `wails dev` compiles the root package with its dev server. CLI imports (like `golang.org/x/term`) are fine — they're already in go.sum.
**How to avoid:** No action needed. All CLI dependencies are already in `go.mod`.
**Warning signs:** `wails dev` fails to compile after adding CLI files.

## Code Examples

### Unified main() Dispatch Pattern

```go
// Source: STATE.md locked decision
// File: main.go (root package)
func main() {
    // GUI mode: no args, or first arg is a flag
    if len(os.Args) == 1 || strings.HasPrefix(os.Args[1], "-") {
        // Intercept --help / -h before Wails opens a window
        if len(os.Args) > 1 {
            usage()
            return
        }
        runGUI()
        return
    }
    // CLI mode: dispatch to command handler
    runCLI(os.Args[1:])
}

func runGUI() {
    app := NewApp()
    err := wails.Run(&options.App{ /* existing options */ })
    if err != nil {
        panic(err)
    }
}

func runCLI(args []string) {
    cmd := args[0]

    // Daemon sub-commands that don't need EnsureDaemon
    if cmd == "daemon" && (len(args) < 2 || args[1] != "status") {
        if err := cmdDaemon(args[1:], os.Stdout); err != nil {
            fmt.Fprintf(os.Stderr, "%v\n", err)
            os.Exit(1)
        }
        return
    }

    // All other commands: auto-start daemon, create client, dispatch
    socketPath := daemon.DefaultSocketPath()
    if err := daemon.EnsureDaemon(socketPath); err != nil {
        fmt.Fprintf(os.Stderr, "agenthub: %v\n", err)
        os.Exit(1)
    }
    client := daemon.NewDaemonClient(socketPath)

    cmdArgs := args[1:]
    var err error
    switch cmd {
    case "new":
        err = cmdNew(client, cmdArgs, os.Stdout)
    // ... all 13 commands
    default:
        fmt.Fprintf(os.Stderr, "agenthub: unknown command %q\n", cmd)
        usage()
        os.Exit(1)
    }

    if err != nil {
        fmt.Fprintf(os.Stderr, "%v\n", err)
        os.Exit(1)
    }
}
```

### File Migration (no changes to function bodies)

```go
// In each moved file, only change:
// FROM: package main   (in cmd/agenthub-cli/)
// TO:   package main   (in root — same text, different directory)
// That's the only required change.
```

### Test Migration: Shared Helpers

```go
// cmd_cli_test.go (was cmd/agenthub-cli/main_test.go)
// testSetup and testSetupWithWebServer must be in this file or a shared _test.go
// because cmd_attach_test.go and cmd_daemon_test.go also use them.
// Keep them here — they're already in main_test.go alongside the other tests.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Two binaries (`agenthub` + `agenthub-cli`) | One binary with arg-based dispatch | v1.4 Phase 27 | Simpler distribution, no PATH confusion |
| Root main.go: stub daemon dispatch only | Root main.go: full CLI dispatch | Phase 27 | Removes split-brain between binaries |

**Deprecated after Phase 27:**
- The stripped-down `daemon.RunDaemon()` stub in current `main.go` — replaced by full `cmdDaemon()` dispatch (though `cmdDaemon` calls `RunDaemon` internally for the no-args case).

## Open Questions

1. **Should `usage()` show both GUI and CLI sections?**
   - What we know: CLI-04 requires `agenthub --help` to cover both GUI launch and all CLI subcommands.
   - What's unclear: Format — two sections ("GUI mode" + "CLI commands") vs. integrated.
   - Recommendation: Two-section format with a preamble: `"Usage: agenthub [command]"` + `"Run with no arguments to launch the desktop GUI."` + existing command list.

2. **Should the 13 CLI functions go into one new file or multiple?**
   - What we know: They currently live in `cmd/agenthub-cli/main.go` (one large file).
   - Recommendation: Split into `cmd_cli.go` (cmdNew/cmdList/cmdKill/cmdRename/cmdWeb/cmdServe/cmdUnserve/cmdHealth/cmdQR/cmdSettings + usage) and keep `cmd_attach.go` and `cmd_daemon.go` as separate files (they already are in agenthub-cli). This matches the existing decomposition.

## Environment Availability

Step 2.6: All dependencies are satisfied by the existing project environment.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Build | ✓ | 1.26.1 | — |
| Wails | GUI build | ✓ | v2.10.2 | — |
| `golang.org/x/term` | attach raw mode | ✓ | v0.41.0 (in go.mod) | — |
| `github.com/coder/websocket` | attach websocket | ✓ | v1.8.14 (in go.mod) | — |

No missing dependencies. No fallbacks needed.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib) |
| Config file | none — `go test` convention |
| Quick run command | `go test ./... -count=1` |
| Full suite command | `go test -race ./... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ROUTE-01 | No-args launches GUI (not panic) | unit | `go test . -run TestDispatch_NoArgs` | ❌ Wave 0 |
| ROUTE-02 | `agenthub list` dispatches to cmdList | unit | `go test . -run TestDispatch_CLICommands` | ❌ Wave 0 |
| ROUTE-03 | `agenthub daemon install` calls serviceControlFunc | unit | `go test . -run TestCmdDaemon_ServiceActions` | ❌ Wave 0 (migrated) |
| CLI-01 | All 13 cmds callable from root package | unit | `go test . -run TestCmd.*` | ❌ Wave 0 (migrated) |
| CLI-02 | `--json` flag works on list/health/web/daemon | unit | `go test . -run TestCmdList_JSON.*` | ❌ Wave 0 (migrated) |
| CLI-03 | attach: raw PTY, detach key, resize, Ctrl-C | unit | `go test . -run TestAttach.*` | ❌ Wave 0 (migrated) |
| CLI-04 | `agenthub --help` prints both GUI and CLI usage | unit | `go test . -run TestDispatch_Help` | ❌ Wave 0 |

**Note:** REQ ROUTE-03, CLI-01, CLI-02, CLI-03 are already covered by the existing `cmd/agenthub-cli/` tests. Migration of those test files to the root package satisfies the requirement. New tests are only needed for ROUTE-01, ROUTE-02, and CLI-04 (dispatch logic in unified main.go).

### Sampling Rate
- **Per task commit:** `go test ./... -count=1`
- **Per wave merge:** `go test -race ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `dispatch_test.go` (root package) — covers ROUTE-01, ROUTE-02, CLI-04 with dispatch logic tests
- [ ] Migrate `cmd/agenthub-cli/main_test.go` → root `cmd_cli_test.go`
- [ ] Migrate `cmd/agenthub-cli/cmd_attach_test.go` → root `cmd_attach_test.go`
- [ ] Migrate `cmd/agenthub-cli/cmd_daemon_test.go` → root `cmd_daemon_test.go`

## Project Constraints (from CLAUDE.md)

| Directive | Application to Phase 27 |
|-----------|------------------------|
| Go: `go fmt`, `golangci-lint`, context-aware functions | All moved files must pass `go fmt` and `golangci-lint` before commit |
| Go: `go mod` for packages | No new deps needed; all in go.mod already |
| Testing: Go `testing` package | Existing test pattern maintained, no new framework |
| Cross-platform: macOS, Linux, Windows | `cmd_attach_unix.go` and `cmd_attach_windows.go` build tags must be preserved exactly |
| Premature Abstraction: need 3 real examples before abstracting | Do not create a new `internal/cli` package — not needed for a merge-then-delete |
| Chesterton's Fence: before removing, articulate why | `cmd/agenthub-cli/` stays intact in Phase 27 — deletion is Phase 28's explicit job |
| Silent Fallbacks: let it crash | Don't add `or {}` fallbacks; existing error patterns (`os.Exit(1)` on error) are correct |

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection of `cmd/agenthub-cli/main.go`, `cmd_daemon.go`, `cmd_attach.go`, `cmd_attach_unix.go`, `cmd_attach_windows.go`, `cmd_daemon_test.go`, `cmd_attach_test.go`, `main_test.go`
- Direct codebase inspection of root `main.go`, `app.go`, `assets_prod.go`, `assets_stub.go`
- `.planning/STATE.md` locked decision: dispatch strategy `len(os.Args) == 1 || os.Args[1] starts with "-" → GUI`
- `.planning/REQUIREMENTS.md` — requirement IDs and descriptions
- `go.mod` — all dependencies verified as already present

### Secondary (MEDIUM confidence)
- Wails v2 documentation (v2.10.2 already in use) — no new Wails API required

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in use, no new dependencies
- Architecture: HIGH — migration plan is direct copy-and-rewire of existing code
- Pitfalls: HIGH — derived from direct inspection of actual code, not speculation

**Research date:** 2026-03-25
**Valid until:** Indefinite — this is an internal code migration with stable, already-deployed code
