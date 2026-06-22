---
gsd_state_version: 1.0
milestone: v4.0
milestone_name: Hub-First Consolidation & UI/UX Overhaul
status: ready_to_plan
stopped_at: Phase 146 complete (5/5) — ready to discuss Phase 147
last_updated: 2026-06-22T20:13:27.615Z
last_activity: 2026-06-22
progress:
  total_phases: 15
  completed_phases: 11
  total_plans: 41
  completed_plans: 42
  percent: 73
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-19 — v4.0 milestone scoped)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 147 — in app help page

## Current Position

Phase: 147
Plan: Not started
Status: Ready to plan
Last activity: 2026-06-22

Progress: [██████████] 100%

## Operator Next Steps (pre-release, carry-forward)

**Pre-next-release operator follow-ups (no coding, required before next tagged release):**

1. **`RELEASE_PUBLISH_TOKEN`** (one-time): create fine-grained PAT scoped to `Contents: read/write` on `scottkw/agenthub`, then `gh secret set RELEASE_PUBLISH_TOKEN`. Without this, `release.published` will not auto-trigger `distribute.yml`.
2. **`WINGET_FIRST_SUBMISSION=true`** (one-time, first WinGet submission only): `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Unset after microsoft/winget-pkgs accepts the first submission.
3. ~~**Apply branch protection on `main` (TEST-02, Phase 143-04 deferred)**~~ **DONE 2026-06-22**: #100 (Phase 144) and #101 (Phase 145) fixed → all 4 CI Build jobs green (run 27954448933) → branch protection applied via the TESTING.md §3 `gh api … PUT` (5 required checks: 4 build matrix + `playwright`; `strict:false`, `enforce_admins:false`, no PR-review). Remaining optional smoke-test: open a throwaway failing PR to confirm the gate blocks merge while admin doc-push to `main` still lands.

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
| 143 | Regression Test Program | TEST-01..05 (TEST-02 applied 2026-06-22) | Complete |
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

Last session: 2026-06-22T18:49:31.498Z
Stopped at: Phase 146 context gathered
Resume file: None
Next action: `/gsd:plan-phase 136`
