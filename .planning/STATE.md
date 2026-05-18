---
gsd_state_version: 1.0
milestone: v3.3.1
milestone_name: Bug Sweep
status: planning
stopped_at: "Phase 113 Plan 01 executed on main (5 task commits: 2 RED, 2 GREEN, 1 VERIFICATION docs); attachTouchScroll helper + touch-action: pan-y CSS wired; 10+5+922 tests green; physical iPad UAT (5 human_needed items) pending operator"
last_updated: "2026-05-18T16:50:50.473Z"
last_activity: 2026-05-18 — Phase 113 Plan 01 executed
progress:
  total_phases: 6
  completed_phases: 5
  total_plans: 5
  completed_plans: 5
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-17 — v3.3 shipped)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** v3.3 closed. Next: `/gsd-new-milestone` to scope v3.4 (likely v3.3 carry-forward issues #54/#55/#56 + Phase 101 visual UAT cosmetic items + Phase 107/108 deferred cleanups + Phase 103 process-debt retroactive fill + any new GitHub triage).

## Current Position

Phase: 113 (iPad terminal touch-scroll)
Plan: 01 — **complete-pending-human-UAT**
Status: 5 task commits on `main` (5745c02 test RED, c9a8506 feat GREEN handler, 49e88d2 test RED wiring, 8f06df7 feat GREEN wiring, 821d7e0 docs VERIFICATION). UI-03 + UI-04 closed by code: `attachTouchScroll(container, term)` helper (117 source lines) + useEffect wiring + `touch-action: pan-y` CSS companion on `.terminal-session-container`. 10/10 unit + 5/5 source-grep + 922/922 full-suite + tsc clean. Physical iPad UAT (5 `human_needed` items: iPad Safari/Chrome single-finger scrollback, two-finger gesture sanity, OSC 8 link tap probe, sub-threshold tap) flagged in `.planning/phases/113-ipad-terminal-touch-scroll/113-VERIFICATION.md`. Probes the v3.3 UAT-04 carry-over (Issue #56) — may fix as side-effect.
Last activity: 2026-05-18 — Phase 113 Plan 01 executed

## Operator Next Steps (carried into v3.4)

**Pre-next-release operator follow-ups (no coding, will be exercised by next tagged release):**

1. **Phase 106 `RELEASE_PUBLISH_TOKEN`** (one-time): create fine-grained PAT scoped to `Contents: read/write` on `scottkw/agenthub`, then `gh secret set RELEASE_PUBLISH_TOKEN`. Without this, `release.published` will not auto-trigger `distribute.yml` (workflow currently falls back to `GITHUB_TOKEN`, which mutes the trigger).
2. **Phase 106 `WINGET_FIRST_SUBMISSION=true`** (one-time, first submission only): `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Unset after microsoft/winget-pkgs accepts the first submission.

**GitHub issues filed during v3.3 (deferred to v3.4 backlog):**

- `scottkw/agenthub#54` — chafa OSC 10/11 + DA1 response leak into shell stdin (web surface only; desktop unaffected). Pre-existing, not a v3.3 regression. (UAT-05)
- `scottkw/agenthub#55` — WebGLRecoveryBanner does not render despite functional DOM fallback. Pre-existing Phase 93 bug. (UAT-01)
- `scottkw/agenthub#56` — iPad tap-on-link captured by xterm-helper-textarea instead of firing link click handler. Pre-existing iPad-touch polish cluster. (UAT-04)

## Performance Metrics

**Velocity:**

- v3.3 phases: 9 (Phases 100-108, including audit-driven mid-milestone Phases 107 + 108)
- v3.3 plans: 18 across phases with executable plans (105 + 106 are runbook/workflow-only)
- v3.3 commits: 133
- v3.3 timeline: 2026-05-12 → 2026-05-17 (5 days)
- Cumulative: 22 milestones shipped (v1.0–v3.3), 107 phases, ~204 plans

## Session Continuity

Last session: 2026-05-18T16:50:50.463Z
Stopped at: Phase 112 Plan 01 executed on main (4 task commits, RED+GREEN+VERIFICATION); manual cross-surface UAT deferred to operator (no GUI display + no Chrome in executor session); Issue #55 ready to close on v3.3.1 tag
Resume file: None
Next action: macOS operator runs the six `human_needed` items in `.planning/phases/111-web-bridge-osc-da-response-consumption/111-VERIFICATION.md` (web Chrome chafa + OSC/DA probe + regression smoke; desktop Wails chafa + OSC/DA probe + regression smoke; parity decision); records resume signal (approved / approved with desktop follow-up: #<n> / failed); then `/gsd-verify-work --phase 111` for Plan 01 phase-gate. Linux UAT for Phase 110 + Windows UAT for Phase 109 remain pending in parallel.

## Deferred Items

Items carried forward from v3.3 close on 2026-05-17 (see also `.planning/milestones/v3.3-MILESTONE-AUDIT.md` and `.planning/milestones/v3.3-ROADMAP.md` §Milestone Summary):

| Category | Item | Status |
|----------|------|--------|
| operator_runtime | Phase 106 `RELEASE_PUBLISH_TOKEN` PAT | pending (one-time, before next release) |
| operator_runtime | Phase 106 `WINGET_FIRST_SUBMISSION=true` variable | pending (one-time, first WinGet submission only) |
| github_issue | #54 chafa OSC leak (web surface) | deferred v3.4 (pre-existing) |
| github_issue | #55 WebGLRecoveryBanner missing | deferred v3.4 (pre-existing Phase 93 bug) |
| github_issue | #56 iPad tap-on-link xterm-helper-textarea | deferred v3.4 (pre-existing iPad-touch polish cluster) |
| visual_uat | Phase 101 5 visual-fidelity items | deferred (cosmetic, non-gating) |
| tech_debt | Phase 108 WR-01/WR-02 + IN-01..04 (docs/dead-code) | deferred v3.4 |
| tech_debt | Phase 107 IN-01/02/03 + Browse-button aria-label + SettingsSearch SEARCH_INDEX missing "Shell binary" | deferred v3.4 |
| tech_debt | Phase 101 advisory WR-01..09 + IN-01..06 (15 items) | deferred v3.4 |
| process_debt | Phase 103 missing `103-SUMMARY.md` + `103-IIP-DECISION.md` + `103-VERIFICATION.md` | deferred v3.4 (retroactive fill recommended) |
| process_debt | Nyquist `*-VALIDATION.md` missing for Phases 101–108 | deferred (process debt; not a blocker) |
| test_debt | TestOpenCodeANSICapture data race | deferred (pre-existing, skipped) |
| test_debt | Pre-existing `TestShellWebShareWarned_Default`-family failures (3 internal/daemon tests) | deferred v3.4 (SPEC §Out-of-scope for Phase 108) |
| test_debt | Phase 108 PARITY-CLI-03 harness limitation (one documented test skip with v3.4 `SetShellPathForTest` follow-up sketched) | deferred v3.4 |
| Phase 112 P01 | 8min | 3 tasks | 5 files |

## v3.3.1 Plan Execution Log

| Phase | Plan | Status | Duration | Commits | Notes |
|-------|------|--------|----------|---------|-------|
| 109 | 01 | code-complete; pending human Windows UAT (IPC-05) | 7min | 4 (3 cherry-picks from PR #53 by Alexandre Castro + 1 planner doc) | `phase-109-windows-named-pipe-ipc` branch; SUMMARY.md written; pre-existing ShellWebShareWarned failures documented in `deferred-items.md` (already known per line 81 deferred table above) |
| 110 | 01 | code-complete; pending human Linux UAT (PTY-01..04 runtime) | 10min | 7 task commits on `main` (768f999, f6c1b79, eafa6aa, cafd1e8, 6f6138a, 838ba83, + SUMMARY meta-commit) | Linux Wait4 exit detector + no-op stub + 3 unit tests + native.go wire-up + engine_test.go skip flip + 110-VERIFICATION.md + 110-01-SUMMARY.md + deferred-items.md; macOS race regression all PASS (excluding 4 pre-existing failures documented in deferred-items.md); closes Issue #57 once Linux UAT signs off |
| 111 | 01 | code-complete; pending human macOS cross-surface chafa UAT (WEB-02) | 5min | 4 task commits on `main` (c343a1d test RED, 9082f5a feat GREEN unit, 31e5d68 feat GREEN integration + wiring, bcf2bf5 docs VERIFICATION) | InputAbsorber 5-state machine (oscabsorb.go 117 source lines) + 26 unit subtests + 6 integration tests + 4-line server.go wiring; `internal/relay/server.go` UNTOUCHED, no new deps; closes Issue #54 web surface once macOS operator confirms web vs. desktop chafa parity; Open Question 1 (desktop empirical state) deferred to that UAT |
| 112 | 01 | code-complete; pending human cross-surface UAT (UI-01 desktop Wails + UI-01 web Chrome) | 8min | 4 task commits on `main` (b889c63 test RED, a4cdc2e fix GREEN, 99a42c7 docs VERIFICATION, 7e35bcd docs issue-cross-check) | TerminalPanel.tsx onContextLoss reorder (24 changed lines, one block): notify React FIRST then queueMicrotask-deferred dispose with try/catch. RESEARCH §5 pattern; CONTEXT closure-rot hypothesis refuted (RESEARCH §Pattern 1 — React useState setters are identity-stable). Full frontend suite 907/907 PASS; tsc --noEmit clean. UI-02 (DOM fallback) source-traced via WebglAddon.dispose() → renderService.setRenderer(_createRenderer()). Manual UAT deferred (no GUI display + no Chrome in executor session); closes Issue #55 once operator runs 112-VERIFICATION.md UAT-1 + UAT-2 |
| 113 | 01 | code-complete; pending human physical-iPad UAT (UI-03 + UI-04, 5 human_needed items) | ~30min | 5 task commits on `main` (5745c02 test RED, c9a8506 feat GREEN handler, 49e88d2 test RED wiring, 8f06df7 feat GREEN wiring + CSS, 821d7e0 docs VERIFICATION) | New frontend/src/lib/touchScrollHandler.ts (117 src lines) — `attachTouchScroll(container, term) => cleanup`. Translates single-finger touch Δy into `term.scrollLines(-lines)` against xterm public API; multi-touch bails so iOS handles pinch; sub-threshold (<8px) tap path untouched so OSC 8 WebLinksAddon click handler keeps firing; touchmove registered `passive:false` for `preventDefault` on confirmed scroll. TerminalPanel.tsx: 1 import + 1 new useEffect with `[sessionId]` dep array right after existing mount effect; binds to React-owned outer containerRef <div> so it survives `term.dispose()`. style.css: `touch-action: pan-y` on `.terminal-session-container` (companion CSS — does NOT use `touch-action: none`, preserves pinch-zoom). Tests: 10/10 unit + 5/5 source-grep + 922/922 full frontend suite + tsc --noEmit clean. No new deps. Closes Issue #56 once physical iPad UAT confirms; probes the v3.3 UAT-04 carry-over (iPad tap-on-link) as possible bonus side-effect repair. |
