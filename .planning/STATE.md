---
gsd_state_version: 1.0
milestone: v3.3.1
milestone_name: Bug Sweep
status: planning
stopped_at: Phase 110 Plan 01 executed on `main` (7 task commits); SUMMARY.md written; awaiting Linux-host UAT sign-off in 110-VERIFICATION.md
last_updated: "2026-05-18T15:30:00.000Z"
last_activity: 2026-05-18 — Phase 110 Plan 01 executed (Linux PTY natural-exit detection; closes Issue #57)
progress:
  total_phases: 6
  completed_phases: 2
  total_plans: 3
  completed_plans: 3
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-17 — v3.3 shipped)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** v3.3 closed. Next: `/gsd-new-milestone` to scope v3.4 (likely v3.3 carry-forward issues #54/#55/#56 + Phase 101 visual UAT cosmetic items + Phase 107/108 deferred cleanups + Phase 103 process-debt retroactive fill + any new GitHub triage).

## Current Position

Phase: 110 (Linux PTY natural-exit detection)
Plan: 01 — **complete-pending-human-UAT**
Status: 7 task commits on `main` (768f999, f6c1b79, eafa6aa, cafd1e8, 6f6138a, 838ba83, plus the SUMMARY meta-commit). PTY-01..04 closed by this plan code; Linux GUI/TUI/CLI runtime UAT + two Linux automated test runs flagged `human_needed` in `.planning/phases/110-linux-pty-natural-exit-detection/110-VERIFICATION.md`. macOS regression auto-checks all PASS.
Last activity: 2026-05-18 — Phase 110 Plan 01 executed

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

Last session: 2026-05-18T15:30:00.000Z
Stopped at: Phase 110 Plan 01 executed on `main` (7 task commits); SUMMARY.md written; awaiting Linux-host UAT sign-off in 110-VERIFICATION.md
Resume file: None
Next action: Linux operator runs the six `human_needed` items in `.planning/phases/110-linux-pty-natural-exit-detection/110-VERIFICATION.md` (Linux GUI clean/non-zero exit, Linux TUI, Linux CLI smoke, TestStartExitDetector_* under -race -shuffle=on x10, TestListSessions_OnExitCallback_ReceivesNormalized under -race -shuffle=on x10); flips `human_needed` → `human_verified`; then `/gsd-verify-work --phase 110` for Plan 01 phase-gate. Windows UAT for Phase 109 remains pending in parallel.

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

## v3.3.1 Plan Execution Log

| Phase | Plan | Status | Duration | Commits | Notes |
|-------|------|--------|----------|---------|-------|
| 109 | 01 | code-complete; pending human Windows UAT (IPC-05) | 7min | 4 (3 cherry-picks from PR #53 by Alexandre Castro + 1 planner doc) | `phase-109-windows-named-pipe-ipc` branch; SUMMARY.md written; pre-existing ShellWebShareWarned failures documented in `deferred-items.md` (already known per line 81 deferred table above) |
| 110 | 01 | code-complete; pending human Linux UAT (PTY-01..04 runtime) | 10min | 7 task commits on `main` (768f999, f6c1b79, eafa6aa, cafd1e8, 6f6138a, 838ba83, + SUMMARY meta-commit) | Linux Wait4 exit detector + no-op stub + 3 unit tests + native.go wire-up + engine_test.go skip flip + 110-VERIFICATION.md + 110-01-SUMMARY.md + deferred-items.md; macOS race regression all PASS (excluding 4 pre-existing failures documented in deferred-items.md); closes Issue #57 once Linux UAT signs off |
