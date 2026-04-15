---
phase: 77
slug: tui-session-operations
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-15
linked_at: 2026-04-15
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

*Linked to finalized plans 77-01, 77-02, 77-03, 77-04. Task IDs follow `77-NN-TM` convention (plan NN, task M within plan).*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 77-01-T1 | 77-01 | 1 | TUI-03, TUI-04, TUI-05, TUI-06 | T-77-RACE | Model fields store session IDs (not indices); modal/editing state immutable during tick refresh | build + vet | `go build ./internal/tui/... && go vet ./internal/tui/...` | yes | ⬜ pending |
| 77-01-T2 | 77-01 | 1 | TUI-03, TUI-04, TUI-05, TUI-06 | T-77-DOS, T-77-RACE | Priority-based key dispatch; kill/rename by ID not index; r=rename R=refresh reassignment | unit | `go test ./internal/tui/... -count=1 -timeout 30s` | yes | ⬜ pending |
| 77-02-T1 | 77-02 | 2 | TUI-03 | T-77-RACE, T-77-INJ | LockedWriter serializes concurrent writes; attach logic extracted; no inline duplication | unit + build | `go build ./... && go test ./internal/attach/... -count=1 -timeout 30s` | yes | ⬜ pending |
| 77-02-T2 | 77-02 | 2 | TUI-03 | T-77-PATH, T-77-DOS | attachCmd uses sessionID by value; tea.Exec dispatched; errored sessions blocked; defer term.Restore | unit | `go test ./internal/tui/... -count=1 -timeout 30s` | yes | ⬜ pending |
| 77-03-T1 | 77-03 | 2 | TUI-05, TUI-06 | T-77-INJ | Kill dialog renders session name via fmt.Sprintf %q + Lip Gloss; no raw ANSI; FgDanger for title | build + vet | `go build ./internal/tui/... && go vet ./internal/tui/...` | yes | ⬜ pending |
| 77-03-T2 | 77-03 | 2 | TUI-05, TUI-06 | T-77-DOS | Kill confirm state: default No, y/n shortcuts, toggle, session killed toast; rename: empty rejected, same-name no-op, nav suppressed | unit | `go test ./internal/tui/... -count=1 -timeout 30s` | yes | ⬜ pending |
| 77-04-T1 | 77-04 | 3 | TUI-04 | T-77-PATH, T-77-INJ | Agent names from DetectCLIs (controlled); dir/args via textinput (safe); no client-side path validation | build + vet | `go build ./internal/tui/... && go vet ./internal/tui/...` | yes | ⬜ pending |
| 77-04-T2 | 77-04 | 3 | TUI-04 | T-77-DOS | Focus cycling modular arithmetic; agent cycling modular; textinput delegation single Update; DetectCLIs cached | unit | `go test ./internal/tui/... -count=1 -timeout 30s` | yes | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Coverage check:**
- TUI-03: 77-01-T1 (build), 77-01-T2 (unit — key reassignment + attach dispatch), 77-02-T1 (unit — extract), 77-02-T2 (unit — ExecCommand + dispatch) = 4 tasks
- TUI-04: 77-01-T1 (build), 77-01-T2 (unit — modal open), 77-04-T1 (build — modal render), 77-04-T2 (unit — focus/agent/submit) = 4 tasks
- TUI-05: 77-01-T1 (build), 77-01-T2 (unit — kill open/confirm/cancel), 77-03-T1 (build — modal render), 77-03-T2 (unit — kill dialog + message) = 4 tasks
- TUI-06: 77-01-T1 (build), 77-01-T2 (unit — rename start/submit/cancel), 77-03-T1 (build — inline edit), 77-03-T2 (unit — rename view + message) = 4 tasks

Every requirement has >= 2 verifying tasks.

---

## Wave 0 Requirements

Wave 0 is handled by Plan 77-01 (Wave 1) — all scaffolding, types, and test infrastructure are created in the first plan.

- [ ] No new `go get` needed (all deps already in go.mod from Phase 76)
- [ ] `internal/tui/model.go` extended with Phase 77 state fields and message types (77-01-T1)
- [ ] `internal/tui/update_test.go` extended with Phase 77 tests (77-01-T2)
- [ ] `internal/tui/view_test.go` extended with Phase 77 tests (77-01-T2)
- [ ] `internal/tui/help_test.go` updated for new help content (77-01-T2)
- [ ] `internal/attach/` package directory created (77-02-T1)
- [ ] `internal/tui/attach.go` — NEW file for ExecCommand (77-02-T2)
- [ ] `internal/tui/modal.go` — NEW file for modal rendering (77-03-T1)
- [ ] `internal/tui/attach_test.go` — NEW file for attach tests (77-02-T2)

---

## Manual-Only Verifications

TUI rendering and interactive attach/detach cannot be fully covered by headless unit tests. The following behaviors require live-terminal UAT:

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Attach flow end-to-end (TUI → raw PTY → detach → TUI) | TUI-03 | `tea.Exec` requires real terminal for alt-screen exit/re-enter transition | In a real terminal: `agenthub tui`, select a running session, press Enter, confirm raw PTY attach and status bar display, press Ctrl-\ to detach, confirm TUI resumes without visual artifacts |
| Attach to errored / stopped session | TUI-03 | Toast rendering depends on real terminal | In TUI select a stopped/errored session, press Enter, verify toast "Session not available" shown briefly and UI does not attempt attach |
| Attach resume selection preserved | TUI-03 | Requires real detach event | After detach, verify previously-attached session remains highlighted in list |
| New-session modal creation end-to-end | TUI-04 | Daemon RPC visible only against live daemon | Open modal with `n`, cycle agent picker with Left/Right, Tab between fields, fill dir + args, Enter; verify new session appears in list |
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
| Keybinding reassignment (`r` rename, `R` refresh) | UI-SPEC Keybindings | Help overlay content | Press `?`; verify `r rename`, `R refresh`, `n new`, `d kill` lines present with no `r refresh` regression |

Each of the above MUST be listed as a human UAT item in `77-HUMAN-UAT.md` after implementation.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (test stubs + any new dependencies + `internal/attach/` extraction)
- [x] No watch-mode flags (always `-count=1`)
- [x] Feedback latency < 30s
- [ ] Manual UAT items from this file are mirrored in `77-HUMAN-UAT.md` after execution
- [x] `nyquist_compliant: true` set in frontmatter after planner links all Task IDs

**Approval:** pending
