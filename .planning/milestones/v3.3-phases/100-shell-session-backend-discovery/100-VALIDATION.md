---
phase: 100
slug: shell-session-backend-discovery
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-12
---

# Phase 100 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `100-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (existing 300+ test suite) |
| **Config file** | none — Go convention |
| **Quick run command** | `go test ./internal/pty/... ./internal/daemon/... -run Shell -race -count=1` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~15s quick / ~90s full (existing baseline) |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/pty/... ./internal/daemon/... -run Shell -race -count=1`
- **After every plan wave:** `go test ./internal/pty/... ./internal/daemon/... -race -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green on macOS dev box + CI matrix (macOS/Linux/Windows)
- **Max feedback latency:** ~15 seconds (quick command)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD-04-discover-found | discovery | 1 | SHELL-04 | — | N/A | unit | `go test ./internal/pty -run TestDiscoverShells_FindsInstalledShells -race` | ❌ W0 | ⬜ pending |
| TBD-04-discover-missing | discovery | 1 | SHELL-04 | — | N/A | unit | `go test ./internal/pty -run TestDiscoverShells_SkipsMissing -race` | ❌ W0 | ⬜ pending |
| TBD-04-etcshells-fixture | discovery | 1 | SHELL-04 | — | N/A | unit | `go test ./internal/pty -run TestDiscoverShells_EtcShellsFixture -race` | ❌ W0 | ⬜ pending |
| TBD-04-etcshells-missing | discovery | 1 | SHELL-04 | — | Silent skip on missing/empty `/etc/shells` (no panic / no error leak) | unit | `go test ./internal/pty -run TestDiscoverShells_NoEtcShells -race` | ❌ W0 | ⬜ pending |
| TBD-04-windows-pwsh | discovery | 1 | SHELL-04 | — | N/A | unit (GOOS=windows) | `go test ./internal/pty -run TestDiscoverShells_Windows -race` | ❌ W0 | ⬜ pending |
| TBD-04-api-route | api | 2 | SHELL-04 | — | N/A | integration | `go test ./internal/daemon -run TestHandleListShells -race` | ❌ W0 | ⬜ pending |
| TBD-05-bash-argv | spawn | 1 | SHELL-05 | — | argv contains `-i` and never `-l` / `--login` | unit (spyBackend) | `go test ./internal/daemon -run TestCreateSession_ShellArgv_Interactive -race` | ❌ W0 | ⬜ pending |
| TBD-05-zsh-argv | spawn | 1 | SHELL-05 | — | argv contains `-i` and never `-l` / `--login` | unit (spyBackend) | `go test ./internal/daemon -run TestCreateSession_ZshArgv_Interactive -race` | ❌ W0 | ⬜ pending |
| TBD-05-pwsh-argv | spawn | 1 | SHELL-05 | — | argv contains `-NoLogo`; no `-l` / `--login` | unit (spyBackend) | `go test ./internal/daemon -run TestCreateSession_PwshArgv -race` | ❌ W0 | ⬜ pending |
| TBD-05-workdir-pass | spawn | 1 | SHELL-05 | — | Caller-supplied `WorkDir` reaches `cmd.Dir` unchanged | unit (spyBackend) | `go test ./internal/daemon -run TestCreateSession_ShellWorkDirHonored -race` | ❌ W0 | ⬜ pending |
| TBD-05-workdir-default-home | spawn | 1 | SHELL-05 | — | Empty WorkDir resolves to `$HOME` for shells (not daemon CWD) | unit | `go test ./internal/daemon -run TestCreateSession_ShellEmptyWorkDirHome -race` | ❌ W0 | ⬜ pending |
| TBD-05-real-pty-smoke | spawn | 2 | SHELL-05 | — | N/A | integration (POSIX only) | `go test ./internal/pty -run TestShellSpawn_RealPTY -race` | ❌ W0 | ⬜ pending |
| TBD-09-no-watch | status-bypass | 1 | SHELL-09 | — | `go status.Watch(...)` is never started for shell sessions (no goroutine reads PTY output for heuristics) | unit (spyBackend + state assertion) | `go test ./internal/daemon -run TestCreateSession_ShellSkipsStatusWatch -race` | ❌ W0 | ⬜ pending |
| TBD-09-list-running | status-bypass | 1 | SHELL-09 | — | `ListSessions` returns Status="running" for live shell; never `"waiting"` or `"error"` | unit | `go test ./internal/daemon -run TestListSessions_ShellStatusRunning -race` | ❌ W0 | ⬜ pending |
| TBD-09-list-stopped | status-bypass | 2 | SHELL-09 | — | After PTY exit, ListSessions transitions to Status="stopped" | integration | `go test ./internal/daemon -run TestListSessions_ShellStatusStopped -race` | ❌ W0 | ⬜ pending |
| TBD-09-no-status-map | status-bypass | 1 | SHELL-09 | — | `engine.sessionStatuses[id]` map has no entry for shell sessions (defensive — guards against heuristic leak) | unit (engine state probe) | `go test ./internal/daemon -run TestShell_NoStatusMapEntry -race` | ❌ W0 | ⬜ pending |

*Final task IDs assigned by gsd-planner. Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/pty/shells.go` — new file (`ShellSpec`, `DetectedShell`, `DiscoverShells()`)
- [ ] `internal/pty/shells_test.go` — table-driven discovery tests + Windows-tagged test
- [ ] `internal/daemon/engine_test.go` — extend with shell argv + WorkDir + status-bypass tests (reuse existing `spyBackend`)
- [ ] `internal/daemon/api_test.go` — extend with `TestHandleListShells`
- [ ] `internal/daemon/types.go` — add `ShellsResponse` and `DetectedShell` JSON types
- [ ] No framework install needed — Go stdlib `testing` already in use

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Windows ConPTY shell spawn end-to-end | SHELL-04 / SHELL-05 | go-pty Windows real-PTY tests are fragile in CI; better gated on local UAT | On Windows: spawn pwsh session via API, attach, verify prompt renders, run `dir`, observe output, exit cleanly |
| Cross-platform `agenthub list` never shows `waiting` / `errored` for shells | SHELL-09 | Visual eye-balling of the TUI over a real session lifetime | On each OS: spawn shell, type commands for 30s, leave idle for 30s, exit; run `agenthub list` continuously and confirm Status only shows `running` then `stopped` |
| zsh sources `~/.zshrc` (real shell behavior) | SHELL-05 | Real-shell semantics depend on dotfiles, not testable headlessly | macOS: spawn zsh via daemon, observe prompt format matches local `~/.zshrc` PS1 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (5 file gaps above)
- [ ] No watch-mode flags (Go tests are one-shot by default)
- [ ] Feedback latency < 20s (quick command target: ~15s)
- [ ] `nyquist_compliant: true` set in frontmatter after Wave 0 completes

**Approval:** pending
