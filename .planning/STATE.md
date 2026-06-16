---
gsd_state_version: 1.0
milestone: v3.6
milestone_name: Hub (Session Grid / Control Room)
status: ready_to_plan
stopped_at: Phase 131 complete (5/5) — ready to discuss Phase 132
last_updated: 2026-06-16T21:18:14.063Z
last_activity: 2026-06-16
progress:
  total_phases: 5
  completed_phases: 1
  total_plans: 5
  completed_plans: 5
  percent: 20
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-15 — after v3.5 milestone close)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 132 — unified grid + mini preview + named groups

## Current Position

Phase: 132
Plan: Not started
Status: Ready to plan
Last activity: 2026-06-16

## Operator Next Steps (pre-release, carry-forward)

**Pre-next-release operator follow-ups (no coding, required before next tagged release):**

1. **`RELEASE_PUBLISH_TOKEN`** (one-time): create fine-grained PAT scoped to `Contents: read/write` on `scottkw/agenthub`, then `gh secret set RELEASE_PUBLISH_TOKEN`. Without this, `release.published` will not auto-trigger `distribute.yml`.
2. **`WINGET_FIRST_SUBMISSION=true`** (one-time, first WinGet submission only): `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Unset after microsoft/winget-pkgs accepts the first submission.

## Performance Metrics

**Velocity:**

- v3.5 phases: 6 (Phases 123-128)
- v3.5 plans: 27/27
- v3.5 commits: 238
- v3.5 timeline: 2026-06-14 → 2026-06-15 (2 days)
- v3.5 source changes: ~14.9K source LOC added across 86 files (excluding `.planning/`)
- v3.5.1 phases: 2 (Phases 129-130)
- v3.5.1 plans: 7/7
- v3.5.1 timeline: 2026-06-16 (1 day)
- Cumulative: 26 milestones shipped (v1.0–v3.5.1), 130 phases, ~267 plans

## Session Continuity

Last session: 2026-06-16T19:25:14.771Z
Stopped at: Phase 131 UI-SPEC approved
Resume file: None
Next action: /gsd:plan-phase 131

## Deferred Items

Items carried forward from v3.5 close (2026-06-15) and pre-release operator tasks:

| Category | Item | Status |
|----------|------|--------|
| operator_runtime | `RELEASE_PUBLISH_TOKEN` PAT | pending (one-time, before next release) |
| operator_runtime | `WINGET_FIRST_SUBMISSION=true` variable | pending (one-time, first WinGet submission only) |
| manual_uat | Phase 125 editor on-screen render + CodeMirror Tab/Cmd-V in WebView | pending (live app required) |
| manual_uat | Phase 126 `$EDITOR` suspend-resume terminal restore | pending (live app required) |
| manual_uat | Phase 124 home-dir warning banner on-screen | pending (live app required) |
| deferred_issue | #82 TUI Files upload parity | deferred to a later milestone (formally descoped Phase 126) |
| deferred_issue | #82 TUI Hub parity (attention + float-to-top + named groups) | signed-off deferral — not a silent gap |
| bookkeeping | Nyquist frontmatter `nyquist_compliant:false` on Phases 123/125/126/127 | advisory; tests green |
| Phase 131 P04 | 15 | 2 tasks | 4 files |

## v3.6 Phase Plan

| Phase | Name | Requirements | Status |
|-------|------|--------------|--------|
| 131 | Hub Foundation + Static Session Cards | HUB-01..04, CARD-01..06, CARD-08, GRID-01..02, GRID-04..06 (17) | Not started |
| 132 | Unified Grid + Mini Preview + Named Groups | CARD-07, GRID-03, GRID-07, GROUP-01..04 (7) | Not started |
| 133 | Attention + Pulse | ATTN-01..06 (6) | Not started |
| 134 | Modal Interaction | MODAL-01..06 (6) | Not started |
| 135 | Accessibility Hardening | A11Y-01..04 (4) | Not started |

**Total:** 40 requirements mapped across 5 phases (39 v3.6 reqs; 100% coverage)

## Key Decisions (v3.6)

| Decision | Resolution |
|----------|------------|
| HubFilter type export location | Exported from HubFilterBar.tsx — single import location for HubPanel and all consumers |
| Filter count derivation | Mirrors SessionCard.deriveStatus: stopped+exitCode!=0→stopped-err, stopped+exitCode=0→stopped-ok |
| Mini preview implementation | Throttled snapshot of session's recent output tail — NEVER a live xterm per card. Performance constraint is known and non-negotiable (CARD-07). |
| Briefing modal data source | Driven by real terminal tail (actual prompt the agent printed). Structured "agent suggests options" multi-select from #78 deferred to #93 — agents don't emit that data today. |
| Remote modal interaction | Reuses locked Phase 122 design (daemon proxy + join-code exchange). No new remote-access architecture (MODAL-06). |
| TUI Hub parity | Explicitly deferred to issue #82 with user sign-off. Cross-surface parity contract remains; this is a documented deferral. |
| A11Y phase placement | Dedicated Phase 135 hardening phase validates the full surface end-to-end. Colorblind-safe constraint (user is colorblind) is release-blocking — verified at source level (hex constants), not by eye. |
| Group membership key | Session name + working directory. Survives session-id churn across restarts; unmatched sessions fall to a default lane (GROUP-04). |
| SessionCardGrid basename | Inlined helper (no node:path import) — splits on both / and \\ separators, takes last non-empty segment |
| filterSessions export | Exported from HubPanel.tsx for testability; case-insensitive substring search on name, cli, and hostname |

## v3.5.1 Completed (for reference)

| Phase | Plan | Status | Notes |
|-------|------|--------|-------|
| 129 | 01..03 | Complete 2026-06-16 | Write concurrency fix + DNS error UX (RACE-01..03, DNS-01..03) |
| 130 | 01..04 | Complete 2026-06-16 | Remote browse GUI on-ramp (RB-01..05) |

## Already-Fixed Prerequisites (do not re-scope)

| Fix | Commit | Status |
|-----|--------|--------|
| Relay routes not mounted (remote `/api/files/remote/{sid}/...` 404'd) | `58af6d6` | FIXED — proven live |
| Discovery probe rejected cap-protected peers (#84) | `3508bd7` | FIXED — proven live |
| Share-link scope relabel (#24 UX) | `e45ccba` | FIXED |

## Operator Next Steps

- Run `/gsd:plan-phase 131` to begin Phase 131 planning
