---
phase: 100-shell-session-backend-discovery
plan: 02
subsystem: daemon
tags: [shell-sessions, pty, status-watch, tdd, parallel-wave]
requires:
  - internal/pty.DiscoverShells (Plan 100-01 — stub provided in this worktree)
  - internal/pty.KnownShellSpecs (Plan 100-01 — stub provided in this worktree)
  - internal/pty.DetectedShell (Plan 100-01 — stub provided in this worktree)
  - internal/pty.ShellSpec (Plan 100-01 — stub provided in this worktree)
provides:
  - internal/daemon.isShellSession(cli) bool
  - "internal/daemon.(*SessionEngine).resolveShellSpawn(cli) (string, []string, bool)"
  - SHELL-05 satisfied at the daemon layer (interactive non-login shell PTY spawn, WorkDir honor, $HOME fallback)
  - SHELL-09 satisfied at the daemon layer (status.Watch bypass for shell sessions)
affects:
  - internal/daemon/engine.go (knownShells map, CreateSession body, status.Watch call site)
  - internal/daemon/engine_test.go (+13 new tests)
  - internal/pty/shells.go (NEW stub — see "Parallel-wave dependency" below)
  - internal/pty/shells_helpers.go (NEW stub helpers — same note)
tech-stack:
  added: []
  patterns:
    - "Per-CLI dispatch inside CreateSession (mirrors existing 'opencode' env injection at engine.go:216)"
    - "exec.LookPath-based PATH discovery (mirrors internal/pty/detect.go)"
    - "Guard-wrapped goroutine launch (if !isShellSession(cli) { go status.Watch(...) })"
key-files:
  created:
    - internal/pty/shells.go
    - internal/pty/shells_helpers.go
  modified:
    - internal/daemon/engine.go
    - internal/daemon/engine_test.go
decisions:
  - "M2 contract locked in: knownShellSpecs has 4 entries (bash, zsh, pwsh, powershell). cliPaths['powershell'] override resolves cleanly via the override branch without falling through to discovery."
  - "req.Args is intentionally IGNORED for shell sessions (T-100-08 mitigation). args = shellArgs overwrites, not merges."
  - "Empty WorkDir for shells → os.UserHomeDir(). Empty WorkDir for AI CLIs → unchanged (existing behavior preserved by TestCreateSession_AICLIEmptyWorkDirUnchanged)."
metrics:
  duration: ~15min
  completed: 2026-05-12
  tasks: 2
  files_modified: 4
---

# Phase 100 Plan 02: Daemon engine wiring for shell sessions (SHELL-05 + SHELL-09) — Summary

Wired `engine.CreateSession` to dispatch shell-type CLIs (`shell`, `bash`, `zsh`, `pwsh`, `powershell`) into a per-shell argv + WorkDir resolution branch, and added a single-guard bypass of `go status.Watch(...)` for shell sessions so `ListSessions` falls through to its conservative `"running"` default. Closes SHELL-05 and SHELL-09 at the daemon layer.

## New helpers (engine.go)

| Symbol | Location | Signature |
|--------|----------|-----------|
| `isShellSession` | engine.go (near knownShells map) | `func isShellSession(cli string) bool` |
| `resolveShellSpawn` | engine.go (after `ResolveCLI`) | `func (e *SessionEngine) resolveShellSpawn(cli string) (string, []string, bool)` |

Both helpers honor the documented Plan 01 M2 contract: `pty.KnownShellSpecs()` returns 4 entries (bash, zsh, pwsh, powershell), so a `cliPaths["powershell"]` override resolves via the override branch's basename match without needing a special pwsh↔powershell fallback.

## CreateSession modifications

Two surgical inserts in `engine.go:CreateSession`:

1. **After `cliPath := e.ResolveCLI(cli)` (formerly L205):** added the shell-resolution branch that calls `e.resolveShellSpawn(cli)`, replaces `cliPath` + `args` with the resolved values, and substitutes `os.UserHomeDir()` for empty `workDir` on shell sessions only. AI CLI sessions (`opencode`, `claude`, `codex`, `gemini`) reach this branch with `isShell == false` and pass through unchanged — guarded by `TestCreateSession_AICLIEmptyWorkDirUnchanged`.

2. **At `go status.Watch(...)` (formerly L245):** wrapped the goroutine launch in `if !isShellSession(cli) { ... }` so shell sessions never spawn the heuristic detector. `e.sessionStatuses[id]` stays empty; `ListSessions` falls through to its conservative `"running"` default at the existing `engine.go:308` branch.

## Extended `knownShells` map (Pitfall 6 mitigation)

Added four entries to the defensive filter at `engine.go:97` so stale `cliPaths["claude"] = "C:\Program Files\PowerShell\7\pwsh.exe"` overrides are dropped at settings load:

- `"pwsh": true`
- `"pwsh.exe": true`
- `"powershell": true`
- `"powershell.exe": true`

Both bare and `.exe` forms are necessary: on Windows `filepath.Base("/path/to/pwsh.exe")` returns `"pwsh.exe"`, not `"pwsh"`.

## Test coverage map

| Test | Requirement | What it locks |
|------|-------------|---------------|
| `TestCreateSession_ShellArgv_Interactive` | SHELL-05 | bash spawns with absolute path + `-i`, no `-l/--login`, WorkDir honored |
| `TestCreateSession_ZshArgv_Interactive` | SHELL-05 | zsh spawns with `-i`, no login flags |
| `TestCreateSession_PwshArgv` | SHELL-05 | pwsh spawns with `-NoLogo`, no login flags (skipped when pwsh not installed) |
| `TestCreateSession_ShellWorkDirHonored` | SHELL-05 | Non-empty caller WorkDir reaches backend unchanged |
| `TestCreateSession_ShellEmptyWorkDirHome` | SHELL-05 / Pitfall 4 | Empty WorkDir for shells → `os.UserHomeDir()` |
| `TestCreateSession_AICLIEmptyWorkDirUnchanged` | SHELL-05 negative | Empty WorkDir for AI CLIs stays empty — protects against regression |
| `TestCreateSession_ShellSkipsStatusWatch` | SHELL-09 | `GetSessionStatus` returns `"running"` and `sessionStatuses[id]` is absent |
| `TestListSessions_ShellStatusRunning` | SHELL-09 | `ListSessions` returns `Status=="running"` for live shell session |
| `TestShell_NoStatusMapEntry` | SHELL-09 defensive | bash + zsh + system-shell IDs all absent from `sessionStatuses` |
| `TestIsShellSession_AllShellNames` | unit | Helper allowlist is exactly {shell, bash, zsh, pwsh, powershell}, case-sensitive |
| `TestResolveShellSpawn_KnownShell` | unit | Bash resolves with `-i`; `claude` returns `ok=false` |
| `TestResolveShellSpawn_SystemDefault` | unit | `cli="shell"` resolves to `$SHELL` (POSIX) |
| `TestResolveShellSpawn_PowerShellOverride` | **M2 contract** | `cliPaths["powershell"]` resolves through override branch (no discovery fallthrough), argv from `powershell` knownShellSpec |

13 new tests; all pass under `-race`. The pwsh test self-skips on hosts without pwsh installed (POSIX dev box default).

## Regression protection

- `TestCreateSession_OpenCodeEnv` (existing) passes unchanged — confirms the `opencode` env-injection path is not affected by the new shell-dispatch branch.
- `TestCreateSession_AICLIEmptyWorkDirUnchanged` (new) confirms AI CLI sessions still see `WorkDir == ""` when caller passes empty — only shells get the `$HOME` substitution.
- Full `go test ./internal/daemon -race -count=1 -skip TestOpenCodeANSICapture` is green.

(`TestOpenCodeANSICapture` is a pre-existing integration test that requires the real `opencode` binary and exhibits a race in upstream `opencode`; unaffected by this plan's changes.)

## Parallel-wave dependency on Plan 100-01 (IMPORTANT)

**This plan was executed in a worktree concurrently with Plan 100-01**, which is the canonical author of `internal/pty/shells.go`. At the time Plan 02 began, no `internal/pty/shells.go` existed on any branch — Plan 01's worktree had not yet committed.

To unblock Plan 02's build and tests, this worktree commits a **minimal stub** of `internal/pty/shells.go` plus a helper file `internal/pty/shells_helpers.go` (commit `3516a8e`). The stub honors the Plan 01 documented API contract exactly:

- `type ShellSpec struct { Name, DisplayName string; Argv []string }`
- `type DetectedShell struct { Name, DisplayName, Path string; Argv []string }` (with JSON tags for wire exposure)
- `func KnownShellSpecs() []ShellSpec` — returns the 4-entry M2 list
- `func DiscoverShells() []DetectedShell`
- `func DetectShell(name string) (*DetectedShell, error)`
- `var ErrShellNotFound`

The stub lacks production-quality features that Plan 01 owns:

- `/etc/shells` parsing on POSIX
- Windows `powershell.exe` 5.x fallback when `pwsh.exe` is absent
- Build-tagged Windows-specific PATH augmentation hooks

**Orchestrator merge action required:** at wave merge time, Plan 01's commit on `internal/pty/shells.go` MUST take precedence. The two implementations expose identical API surface, so `engine.go` continues to compile against either. The stub commit (`3516a8e`) is clearly labeled `chore(100-02): add parallel-wave shells.go stub for engine wiring` so it can be identified and dropped (or replaced wholesale) when Plan 01 lands.

If Plan 01 ships before merge: drop commit `3516a8e` via the merge strategy (`git merge -X theirs internal/pty/shells.go`) or by re-applying Plan 01's version on top.
If Plan 01 ships after merge: Plan 01's commit overwrites the stub naturally.

## Verification

```bash
# Shell-specific quick gate (all pass under -race):
go test ./internal/daemon -run 'Shell|IsShellSession|ResolveShellSpawn|ListSessions_ShellStatus|AICLIEmptyWorkDir' -race -count=1

# Pattern preservation check:
grep -B1 'go status.Watch(' internal/daemon/engine.go | grep -q 'if !isShellSession(cli)'

# Cross-package regression (pty + daemon):
go test ./internal/pty/... ./internal/daemon/... -race -count=1 -skip TestOpenCodeANSICapture

# Format / vet:
go vet ./internal/daemon/... && gofmt -l internal/daemon/engine.go internal/daemon/engine_test.go
```

All clean.

## Deviations from plan

### Auto-fixed Issues

**1. [Rule 3 — Blocking dependency] Parallel-wave shells.go absent**
- **Found during:** Task 1 (RED build attempt)
- **Issue:** `internal/pty/shells.go` was expected to exist (parallel Plan 100-01 dependency) but no worktree had committed it yet. Plan 02 cannot build, test, or commit without the package.
- **Fix:** Authored a minimal stub of `internal/pty/shells.go` + `internal/pty/shells_helpers.go` that satisfies the documented API contract (`ShellSpec`, `DetectedShell`, `KnownShellSpecs`, `DiscoverShells`, `DetectShell`, `ErrShellNotFound`). Stub is committed under a clearly-labeled `chore(100-02)` commit so the orchestrator can identify and replace it at merge time with Plan 01's canonical version.
- **Files modified:** `internal/pty/shells.go` (new), `internal/pty/shells_helpers.go` (new)
- **Commit:** `3516a8e`

No other deviations. All 13 plan-mandated tests are present, all verification commands return clean, no architectural changes were needed.

## Self-Check: PASSED

- `[ -f internal/daemon/engine.go ]` → FOUND
- `[ -f internal/daemon/engine_test.go ]` → FOUND
- `[ -f internal/pty/shells.go ]` → FOUND (stub — see parallel-wave note)
- `[ -f internal/pty/shells_helpers.go ]` → FOUND (stub helper)
- Commit `f42e914` (RED tests) → FOUND
- Commit `3516a8e` (shells stub) → FOUND
- Commit `b99b4c8` (GREEN engine.go) → FOUND
