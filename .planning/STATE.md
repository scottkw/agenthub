---
gsd_state_version: 1.0
milestone: v4.0
milestone_name: Hub-First Consolidation & UI/UX Overhaul
status: executing
stopped_at: Phase 143 complete; phases 144-150 added to v4.0 (issue burndown) — next /gsd:plan-phase 144
last_updated: 2026-06-22T04:02:28.274Z
last_activity: 2026-06-22
progress:
  total_phases: 15
  completed_phases: 8
  total_plans: 32
  completed_plans: 33
  percent: 53
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-19 — v4.0 milestone scoped)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** v4.0 issue-burndown — phases 144-150 defined; next is `/gsd:plan-phase 144`

## Current Position

Phase: 144 (next to plan) — phases 144-150 added, not yet planned
Plan: Not started
Status: v4.0 in progress — 8/15 phases complete; 144-150 scaffolded from GitHub issues
Last activity: 2026-06-22

Progress: [█████░░░░░] 8/15 phases complete

## Operator Next Steps (pre-release, carry-forward)

**Pre-next-release operator follow-ups (no coding, required before next tagged release):**

1. **`RELEASE_PUBLISH_TOKEN`** (one-time): create fine-grained PAT scoped to `Contents: read/write` on `scottkw/agenthub`, then `gh secret set RELEASE_PUBLISH_TOKEN`. Without this, `release.published` will not auto-trigger `distribute.yml`.
2. **`WINGET_FIRST_SUBMISSION=true`** (one-time, first WinGet submission only): `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Unset after microsoft/winget-pkgs accepts the first submission.
3. **Apply branch protection on `main` (TEST-02, Phase 143-04 deferred)**: deferred until v4.0 CI is green. v4.0 CI is currently red from #100 (daemon styled-tail data race) and #101 (internal/files Windows tests). Once both are fixed and CI passes, apply the verbatim `gh api ... --method PUT` command in TESTING.md §3, then smoke-test the gate (failing PR blocked, admin doc-push still lands).

## v4.0 Phase Plan

| Phase | Name | Requirements | Status |
|-------|------|--------------|--------|
| 136 | TUI Removal | NAV-01, TEST-06 | Complete |
| 137 | Share Modal & Cap Model | SHARE-01..06 | Complete |
| 138 | Hub-First Navigation | NAV-02..05, CARD-01..04 | Complete |
| 139 | Card Rendering & Tab Strip | CARD-05, TAB-01..03 | Complete |
| 140 | UI-Spec Gate | RDS-01, CARRY-02 | Complete |
| 141 | Redesign Implementation | RDS-02..04, CARRY-01 | Complete |
| 142 | Hub & Settings Redesign Polish | POL-01..05 | Complete |
| 143 | Regression Test Program | TEST-01..05 (TEST-02 deferred) | Complete |
| 144 | Daemon Styled-Tail Race Fix | FIX-01 (#100) | Not planned |
| 145 | Windows Files Test Fixes | FIX-02 (#101) | Not planned |
| 146 | Open Session Capability Bug | FIX-03 (#98) | Not planned |
| 147 | In-App Help Page | HELP-01 (#69) | Not planned |
| 148 | Session Tab Chevron | TAB-04 (#68) | Not planned |
| 149 | Google Antigravity Agent | AGENT-01 (#65) | Not planned |
| 150 | Shell-Sharing Warning Toggle | SET-01 (#51) | Not planned |

**Total:** 43 requirements mapped across 15 phases (100% coverage). Phases 144-150 added 2026-06-22 (issue burndown extending v4.0).

## Key Decisions (v4.0)

| Decision | Resolution |
|----------|------------|
| TUI removal sequencing | Phase 136 first — large independent deletion de-risks all downstream structural work |
| Cap model isolation | SHARE phase (137) isolated so dual RO/RW issuance + files-browse inheritance can be reviewed on its own before nav changes |
| Navigation removals sequencing | Share modal (137) must exist before Sessions page removal (138); Remote page removal needs card indicators also in 138 |
| Redesign implementation timing | UI-spec gate (140) before implementation (141) — direction chosen after browser review; skins the final Hub-first structure, not the old one |
| TAB and CARD-05 grouping | Independent cluster (139) runs in parallel-path after TUI removal; not blocked on nav restructure |
| TEST-06 placement | Bundled with TUI removal (Phase 136) — TUI tests are deleted, not migrated |
| TEST-01..05 placement | Dedicated late phase (142) after surface is fully settled |
| CARRY-01 placement | Phase 141 with redesign — Hub GroupSidebar ARIA fix is Hub redesign A11y work |
| CARRY-02 placement | Phase 140 (UI-spec gate) — #93 triage is a planning-time decision producing the scope for 141 |

## Deferred Items (carry-forward from v3.6 close)

| Category | Item | Status |
|----------|------|--------|
| operator_runtime | `RELEASE_PUBLISH_TOKEN` PAT | pending (one-time, before next release) |
| operator_runtime | `WINGET_FIRST_SUBMISSION=true` variable | pending (one-time, first WinGet submission only) |
| manual_uat | Phase 125 editor on-screen render + CodeMirror Tab/Cmd-V in WebView | pending (live app required) |
| manual_uat | Phase 126 `$EDITOR` suspend-resume terminal restore | pending (live app required) |
| manual_uat | Phase 124 home-dir warning banner on-screen | pending (live app required) |
| manual_uat | Phase 138 remote-card UATs #3 (Open-in-browser URL) + #5 (no Kill on remote) | **postponed to 2026-06-22** — needs office 2nd machine as remote peer; 5/7 other checks PASS 2026-06-21 |
| deferred_issue | #82 TUI Files upload parity | moot — TUI removed in Phase 136 |
| deferred_issue | #82 TUI Hub parity | moot — TUI removed in Phase 136 |
| github_issue | #93 Hub fidelity backlog | triaged in Phase 140; in-scope subset delivered in Phase 141 |
| github_issue | #96 tail/preview headless VT render | delivered in Phase 139 |
| github_issue | #97 Hub GroupSidebar ARIA | delivered in Phase 141 |

## Session Continuity

Last session: 2026-06-22T03:41:29.880Z
Stopped at: Completed 143-01-PLAN.md
Resume file: None
Next action: `/gsd:plan-phase 136`
