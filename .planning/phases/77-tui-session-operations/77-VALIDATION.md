---
phase: 77
slug: tui-session-operations
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-15
---

# Phase 77 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `77-RESEARCH.md` → `## Validation Architecture`.
> Task IDs are populated after `gsd-planner` finalizes PLAN.md files.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (stdlib, go1.26.2) + `charm.land/x/exp/teatest/v2` for headless Bubble Tea model tests |
| **Config file** | none — Go's built-in test runner |
| **Quick run command** | `go test ./internal/tui/... ./internal/attach/... -count=1 -timeout 30s` |
| **Full suite command** | `go test ./... -count=1 -timeout 120s` |
| **Estimated runtime** | ~20s quick / ~75s full |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/tui/... ./internal/attach/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -count=1 -timeout 120s`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

*Populated after finalized plans. Task IDs follow `77-NN-TM` convention (plan NN, task M within plan). The planner MUST link every task here after PLAN.md generation and set `nyquist_compliant: true`.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| *pending planner link* | — | — | — | — | — | — | — | — | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Coverage requirement:** Every phase requirement (TUI-03, TUI-04, TUI-05, TUI-06) must have ≥ 2 verifying tasks (one build/vet, one unit/integration).

---

## Wave 0 Requirements

Wave 0 installs the test scaffolding and any new dependencies before implementation waves.

- [ ] Dependency install — add `charm.land/x/exp/teatest/v2` for headless Bubble Tea tests (if not already present via bubbletea/v2 transitive)
- [ ] `internal/attach/` package directory created — for extracted attach logic (shared between `cmd/` and TUI `tea.Exec` path)
- [ ] `internal/tui/modal/` package directory created (or `internal/tui/newsession_*.go` / `internal/tui/confirm_*.go` files) — modal state machines
- [ ] `internal/tui/update_test.go` extended — rename editing state, kill confirm, modal open/close tests
- [ ] `internal/tui/newsession_test.go` — new session modal tests (agent picker cycling, focus cycling, submit/cancel)
- [ ] `internal/tui/confirm_test.go` — kill confirmation dialog tests (default No, y/n shortcuts, toggle)
- [ ] `internal/tui/attach_test.go` — `tea.Exec` wiring test (ExecCommand interface contract, detach → refresh msg)
- [ ] `internal/tui/toast_test.go` — toast kind rendering + auto-dismiss via tea.Tick

---

## Manual-Only Verifications

TUI rendering and interactive attach/detach cannot be fully covered by headless unit tests. The following behaviors require live-terminal UAT:

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Attach flow end-to-end (TUI → raw PTY → detach → TUI) | TUI-03 | `tea.Exec` requires real terminal for alt-screen exit/re-enter transition | In a real terminal: `agenthub tui`, select a running session, press Enter, confirm raw PTY attach and status bar display, press Ctrl-\ to detach, confirm TUI resumes without visual artifacts |
| Attach to errored / stopped session | TUI-03 | Toast rendering depends on real terminal | In TUI select a stopped/errored session, press Enter, verify toast "Session not available" shown briefly and UI does not attempt attach |
| Attach resume selection preserved | TUI-03 | Requires real detach event | After detach, verify previously-attached session remains highlighted in list |
| New-session modal creation end-to-end | TUI-04 | Daemon RPC visible only against live daemon | Open modal with `n`, cycle agent picker with ←/→, Tab between fields, fill dir + args, Enter; verify new session appears in list |
| New-session modal cancel | TUI-04 | Alt-screen rendering | Open modal, press Esc; verify modal closes and no session created |
| New-session agent picker empty state | TUI-04 | Depends on absence of CLI binaries | Run TUI on a machine with no detected agents in PATH; verify `(none found)` label shown in danger color |
| Kill confirmation — confirm path | TUI-05 | Daemon RPC visible only against live daemon | Select session, press `d`, toggle to Yes (or press `y`), confirm session removed from list |
| Kill confirmation — cancel path | TUI-05 | Default-No safety check | Press `d`, press Enter (default No) or Esc or `n`; verify dialog closes with session intact |
| Kill confirmation focus default is No | TUI-05 | Visual focus indicator on real terminal | Open dialog; observe the `[ No ]` option is highlighted on first render |
| Inline rename — submit | TUI-06 | Real terminal cursor placement | Press `r` on selected row, observe Name column becomes textinput, type new name, press Enter; verify session renamed in list |
| Inline rename — cancel | TUI-06 | Real terminal cursor placement | Press `r`, type, press Esc; verify original name restored |
| Inline rename suppresses navigation | TUI-06 | Only visible via live typing | While editing, press `j`/`k` — verify those characters enter the text field instead of moving selection |
| Duplicate rename error | TUI-06 | Daemon returns error | Rename a session to an existing session name; verify toast "Rename failed: {daemon message}" |
| Toast auto-dismiss timing | TUI-03/04/05/06 | Requires real clock tick | Trigger any success/error toast; verify it disappears after ~2 seconds |
| Keybinding reassignment (`r` rename, `R` refresh) | UI-SPEC §Keybindings | Help overlay content | Press `?`; verify `r rename`, `R refresh`, `n new`, `d kill` lines present with no `r refresh` regression |

Each of the above MUST be listed as a human UAT item in `77-HUMAN-UAT.md` after implementation.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (test stubs + any new dependencies + `internal/attach/` extraction)
- [ ] No watch-mode flags (always `-count=1`)
- [ ] Feedback latency < 30s
- [ ] Manual UAT items from this file are mirrored in `77-HUMAN-UAT.md` after execution
- [ ] `nyquist_compliant: true` set in frontmatter after planner links all Task IDs

**Approval:** pending
