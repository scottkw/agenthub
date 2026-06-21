---
gsd_state_version: 1.0
milestone: v4.0
milestone_name: Hub-First Consolidation & UI/UX Overhaul
status: verifying
stopped_at: Phase 139 context gathered
last_updated: "2026-06-21T01:34:41.284Z"
last_activity: 2026-06-21
progress:
  total_phases: 7
  completed_phases: 4
  total_plans: 13
  completed_plans: 13
  percent: 57
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-19 — v4.0 milestone scoped)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 139 — card-rendering-tab-strip

## Current Position

Phase: 139 (card-rendering-tab-strip) — EXECUTING
Plan: 4 of 4
Status: Phase complete — ready for verification
Last activity: 2026-06-21

Progress: [██████████] 100%

## Operator Next Steps (pre-release, carry-forward)

**Pre-next-release operator follow-ups (no coding, required before next tagged release):**

1. **`RELEASE_PUBLISH_TOKEN`** (one-time): create fine-grained PAT scoped to `Contents: read/write` on `scottkw/agenthub`, then `gh secret set RELEASE_PUBLISH_TOKEN`. Without this, `release.published` will not auto-trigger `distribute.yml`.
2. **`WINGET_FIRST_SUBMISSION=true`** (one-time, first WinGet submission only): `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Unset after microsoft/winget-pkgs accepts the first submission.

## v4.0 Phase Plan

| Phase | Name | Requirements | Status |
|-------|------|--------------|--------|
| 136 | TUI Removal | NAV-01, TEST-06 | Complete |
| 137 | Share Modal & Cap Model | SHARE-01..06 | Complete |
| 138 | Hub-First Navigation | NAV-02..05, CARD-01..04 | Planned |
| 139 | Card Rendering & Tab Strip | CARD-05, TAB-01..03 | Not started |
| 140 | UI-Spec Gate | RDS-01, CARRY-02 | Not started |
| 141 | Redesign Implementation | RDS-02..04, CARRY-01 | Not started |
| 142 | Regression Test Program | TEST-01..05 | Not started |

**Total:** 31 requirements mapped across 7 phases (100% coverage)

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
| deferred_issue | #82 TUI Files upload parity | moot — TUI removed in Phase 136 |
| deferred_issue | #82 TUI Hub parity | moot — TUI removed in Phase 136 |
| github_issue | #93 Hub fidelity backlog | triaged in Phase 140; in-scope subset delivered in Phase 141 |
| github_issue | #96 tail/preview headless VT render | delivered in Phase 139 |
| github_issue | #97 Hub GroupSidebar ARIA | delivered in Phase 141 |

## Session Continuity

Last session: 2026-06-21T01:34:41.279Z
Stopped at: Phase 139 context gathered
Resume file: None
Next action: `/gsd:plan-phase 136`
