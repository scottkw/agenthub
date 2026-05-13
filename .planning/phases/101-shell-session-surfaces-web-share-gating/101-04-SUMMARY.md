---
phase: 101-shell-session-surfaces-web-share-gating
plan: 04
subsystem: CLI subcommand + TUI agent picker + BadgeShell color
tags:
  - shell
  - cli
  - tui
  - lipgloss
  - tdd
dependency_graph:
  requires:
    - 101-01 (daemon ListShells, ShellWebShareWarned, DaemonClient bindings — already merged)
    - 100 (pty.DiscoverShells, resolveShellSpawn, knownShellSpecs)
  provides:
    - cmdNewShell CLI subcommand
    - Model.detectedShells + agentEntries unified picker
    - Styles.BadgeShell color token (slate-cyan)
  affects:
    - main.go dispatch (new → shell sub-subcommand)
    - cmd_cli.go usage() output
    - internal/tui/* (modal.go, model.go, styles.go, tui.go, update.go)
tech_stack:
  added: []
  patterns:
    - "Phase 100 shell-resolution dispatch (resolveShellSpawn) re-used verbatim — CLI passes raw cli string, daemon resolves to absolute path + non-login interactive argv"
    - "lipgloss adaptive light/dark color via ld() helper, mirroring all 6 existing BadgeXxx tokens"
    - "Unified agentEntry slice pattern for picker — single index space across AI CLIs + shells"
key_files:
  created:
    - .planning/phases/101-shell-session-surfaces-web-share-gating/deferred-items.md
  modified:
    - cmd_cli.go
    - cmd_cli_test.go
    - main.go
    - internal/tui/tui.go
    - internal/tui/model.go
    - internal/tui/modal.go
    - internal/tui/styles.go
    - internal/tui/styles_test.go
    - internal/tui/update.go
    - internal/tui/update_test.go
decisions:
  - "CLI flag-set pattern matches existing cmdList (flag.NewFlagSet with ContinueOnError + io.Discard for self-managed error copy)"
  - "Empty --shell= detection via fs.Visit (distinguishes 'flag absent' from 'flag set to empty string')"
  - "Shell sessions cleared of args at submit time in submitNewSession (mirrors cmdNewShell behavior and Phase 100 A6)"
  - "testModel() fixture clears detectedShells by default so legacy TUI tests retain pre-Phase-101 semantics"
  - "view.go session-list-row needed no change — already routes through agentBadgeColor() which now covers shells"
metrics:
  duration: "~45 minutes"
  tasks_completed: 3
  files_modified: 10
  files_created: 1
  tests_added: 19
  completed_date: 2026-05-12
---

# Phase 101 Plan 04: CLI new shell subcommand + TUI agent picker + BadgeShell Summary

`agenthub new shell` CLI subcommand with --shell=bash|zsh|pwsh|powershell flag + TUI agent picker shell entries + TUI BadgeShell lipgloss style — SHELL-02 + SHELL-03 + SHELL-06 TUI half closed.

## Tasks Completed

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | CLI new shell subcommand (TDD) | `0156845` | cmd_cli.go, cmd_cli_test.go, main.go |
| 2 | TUI agent picker + BadgeShell color (TDD) | `b6ab235` | internal/tui/{tui,model,modal,styles,update}.go + matching _test.go files |
| 3 | Full build + cross-platform smoke + verification | (no commit — verification only) | — |

## Implementation Notes

### CLI: `cmdNewShell` (cmd_cli.go)

**Signature:** `func cmdNewShell(client *daemon.DaemonClient, args []string, extraArgs []string, out io.Writer) error`

**Dispatch site (main.go runCLI):**
```go
case "new":
    if len(cmdArgs) > 0 && cmdArgs[0] == "shell" {
        err = cmdNewShell(client, cmdArgs[1:], extraArgs, os.Stdout)
    } else {
        err = cmdNew(client, cmdArgs, extraArgs, os.Stdout)
    }
```

**usage() lines added (verbatim from UI-SPEC §CLI):**
```
  new shell [<path>]                          Create a new raw shell session
    --shell=bash|zsh|pwsh|powershell           Pick a specific shell (default: system default)
```

**Locked stderr error strings (verbatim per UI-SPEC §CLI):**

| Trigger | Stderr text |
|---------|-------------|
| Unknown --shell value | `agenthub new shell: unknown shell "VALUE" (allowed: bash, zsh, pwsh, powershell, or omit for system default)` |
| Empty --shell= | `agenthub new shell: --shell flag requires a value (one of: bash, zsh, pwsh, powershell)` |
| Extra args after -- | `agenthub new shell: extra arguments are not forwarded to shell sessions; ignoring [ARGS]` |
| Daemon unreachable | `agenthub new shell: daemon unreachable: ERR` |

### TUI changes

- **detectedShells field** (model.go): `[]pty.DetectedShell` — populated by `pty.DiscoverShells()` in newModel (tui.go), in-process per RESEARCH A1.
- **sortShellsForPicker** (modal.go): priority map `{shell:0, bash:1, zsh:2, pwsh:3, powershell:4}` with stable sort. Mirrors the GUI Plan 02 sort so picker order is consistent across all three surfaces.
- **agentEntries** (modal.go): unified picker entry slice — AI CLIs first (in detectedCLIs order), then shells (in sortShellsForPicker order) with `"Shell — "` display prefix.
- **renderAgentPicker / cycleAgent** (modal.go): now walk agentEntries instead of detectedCLIs. cycleAgent returns immediately when the unified slice is empty.
- **submitNewSession** (update.go): reads cli from `entries[idx].cliKey`. For shell sessions (isShellCLI true), args are dropped before dispatch — mirrors Phase 100 Anti-Pattern A6.
- **BadgeShell** (styles.go): new Styles field initialised with `ld(lipgloss.Color("#3d5a80"), lipgloss.Color("#89ddff"))` (slate-cyan, TokyoNight palette). agentBadgeColor switch adds `case "shell", "bash", "zsh", "pwsh", "powershell": return s.BadgeShell`.
- **view.go**: no change — `renderSessionRow` and `renderRemoteSessionRow` already route through `agentBadgeColor()`, which now covers shells via the new switch case.

### Shell-cli-set duplication note

The shell cli identifier set `{shell, bash, zsh, pwsh, powershell}` is duplicated across five surfaces in v3.3:

| Surface | Symbol | File |
|---------|--------|------|
| CLI allowlist | `allowed` map in `cmdNewShell` | cmd_cli.go |
| Daemon dispatch | `isShellSession`, `resolveShellSpawn` | internal/daemon/engine.go (Phase 100) |
| TUI badge color | `agentBadgeColor` switch case | internal/tui/styles.go |
| TUI submit drop-args | `isShellCLI` helper | internal/tui/update.go |
| (frontend Plan 02/03) | `SHELL_CLIS`, `agentBadgeModifier` | frontend/src/App.tsx, TabBar.tsx |

Accept v3.3 duplication per Phase 100 RESEARCH Pitfall note. Revisit in v3.4 if a common pattern emerges (e.g., a shared Go constants package + a generated TypeScript constant).

## Verification

```bash
$ go build ./...           # exit 0
$ go vet ./...             # exit 0
$ gofmt -l <modified files> # empty (no diffs)
$ go test ./... -count=1   # PASS (all packages)
```

| Check | Result |
|-------|--------|
| `go build ./...` | OK |
| `go test ./...` (without -race) | PASS — main, internal/tui, all packages |
| 12 new CLI tests (TestCmdNewShell_* + TestUsage_IncludesNewShell) | PASS |
| 7 new TUI tests (TestAgentBadgeColor_*, TestBadgeShell_*, TestAgentPicker_*) | PASS |
| Phase 100 + Plan 101-01 regression | PASS |
| gofmt clean | OK |
| go vet clean | OK |
| `grep -c 'func cmdNewShell' cmd_cli.go` | 1 ✓ |
| `grep -c 'cmdNewShell' main.go` | 2 ✓ |
| `grep -c 'new shell \[<path>\]' cmd_cli.go` | 2 ✓ (usage + tests inspect via os.ReadFile) |
| `grep -c '"shell", "bash", "zsh", "pwsh", "powershell"' internal/tui/styles.go` | 1 ✓ |
| `grep -c 'BadgeShell' internal/tui/styles.go` | 4 ✓ (struct field + init + switch + doc) |
| `grep -c '#89ddff' / '#3d5a80' in styles.go` | 1 each ✓ |
| `grep -c 'pty.DiscoverShells' internal/tui/tui.go` | 2 ✓ (call + import-line implicit) |
| `grep -c 'Shell — ' internal/tui/modal.go` | 4 ✓ (sortShellsForPicker comments + agentEntries body) |
| `grep -c 'unknown shell %q' cmd_cli.go` | 2 ✓ |
| `grep -c 'extra arguments are not forwarded' cmd_cli.go` | 2 ✓ |

## Acceptance criteria

- [x] All 12 new CLI tests pass (11 cmdNewShell + 1 usage)
- [x] All 7 new TUI tests pass (4 styles + 3 picker)
- [x] Phase 100 and Plan 101-01 tests still pass (no regression)
- [x] All locked CLI stderr strings appear verbatim per UI-SPEC §CLI
- [x] usage() includes literal `new shell [<path>]` and `--shell=bash|zsh|pwsh|powershell` lines
- [x] TUI agent picker cycle order matches UI-SPEC §Interaction TUI flow
- [x] TUI BadgeShell color resolves to #89ddff (dark) / #3d5a80 (light)
- [x] TUI session-list shell row badge uses BadgeShell, NOT FgMuted (via existing agentBadgeColor route)
- [x] SHELL-02 (CLI), SHELL-03 (TUI), SHELL-06 TUI half all closed
- [x] Zero file overlap with Plan 02 (frontend) or Plan 03 (frontend banner) — runs Wave 2 in parallel

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] testModel() fixture leaked host shells into legacy tests**
- **Found during:** Task 2 (TUI tests)
- **Issue:** After adding `pty.DiscoverShells()` to `newModel()` in tui.go, three legacy TUI tests (`TestModal_AgentCycle`, `TestModal_SubmitNoAgents`, `TestView_NewSessionModal_NoAgents`) started failing because they assumed `len(detectedCLIs)` was the entire picker universe. `pty.DiscoverShells()` ran on the host and populated `detectedShells`, changing the cycle modulus.
- **Fix:** Updated `testModel()` to explicitly clear `m.detectedShells = nil` so legacy tests retain pre-Phase-101 semantics. New shell-picker tests opt in by setting `m.detectedShells` directly.
- **Files modified:** internal/tui/update_test.go (testModel helper)
- **Commit:** `b6ab235`

**2. [Rule 2 - Missing functionality] Drop args for shell sessions in TUI submitNewSession**
- **Found during:** Task 2 (TUI implementation)
- **Issue:** The original `submitNewSession` always forwarded the args field to the daemon. For shell sessions, Phase 100 A6 explicitly drops caller-supplied args (T-100-08 mitigation). Without this fix, a user could potentially type args into the TUI modal and have them forwarded — bypassing the locked Phase 100 behavior at the CLI but allowing it via TUI.
- **Fix:** Added `isShellCLI` helper. `submitNewSession` checks the cli identifier and skips arg parsing for shell sessions.
- **Files modified:** internal/tui/update.go
- **Commit:** `b6ab235`

### Out-of-scope items deferred

- **`TestOpenCodeANSICapture` data race** (`internal/daemon/opencode_ansi_test.go`) — pre-existing race detected by `go test -race`. Unrelated to Phase 101-04 (which does not touch `internal/daemon/`). Logged to `deferred-items.md` for triage in a future plan or v3.4 polish.

## Threat surface scan

No new threat surfaces introduced beyond those documented in the plan's `<threat_model>`. T-101-04-01 (Tampering on --shell flag) and T-101-04-03 (Elevation of Privilege on -- extra-args) mitigations are locked in tests `TestCmdNewShell_UnknownShellFlag`, `TestCmdNewShell_EmptyShellFlag`, and `TestCmdNewShell_ExtraArgsWarning`. The new TUI `isShellCLI` helper closes a parallel surface in the TUI submit path that the original plan did not call out explicitly (logged as Deviation #2 above).

## Self-Check: PASSED

**File existence:**
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/cmd_cli.go (+ cmdNewShell)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/cmd_cli_test.go (+12 tests)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/main.go (+ dispatch)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/internal/tui/styles.go (+ BadgeShell)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/internal/tui/model.go (+ detectedShells)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/internal/tui/tui.go (+ DiscoverShells call)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/internal/tui/modal.go (+ agentEntries, sortShellsForPicker)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/internal/tui/update.go (+ isShellCLI)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/internal/tui/styles_test.go (+4 tests)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/internal/tui/update_test.go (+3 tests)
- FOUND: /Users/ken/dev/agenthub/.claude/worktrees/agent-a2b4b793fe90968c4/.planning/phases/101-shell-session-surfaces-web-share-gating/deferred-items.md

**Commits:**
- FOUND: 0156845 feat(101-04): add agenthub new shell CLI subcommand
- FOUND: b6ab235 feat(101-04): add TUI shell picker entries + BadgeShell color
