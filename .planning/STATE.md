---
gsd_state_version: 1.0
milestone: v4.0
milestone_name: Hub-First Consolidation & UI/UX Overhaul
status: planning
last_updated: "2026-06-19T19:06:31.662Z"
last_activity: 2026-06-19
progress:
  total_phases: 0
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-15 — after v3.5 milestone close)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 135 — accessibility-hardening

## Current Position

Phase: Not started (defining requirements)
Plan: —
Status: Defining requirements
Last activity: 2026-06-19 — Milestone v4.0 started

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

Last session: 2026-06-19
Stopped at: v3.6 Phase 135 fully complete (planned→executed→reviewed+fixed→live-UAT validated→secured→UI-audited+fixed). Milestone audit ran: status=tech_debt (39/39 reqs, 0 blockers, 7/7 flows). User chose to STOP before /gsd:complete-milestone to address tech debt first.
Resume file: None
Next action: address tech debt (below) then `/gsd:complete-milestone v3.6` (+ /gsd:cleanup). To resume autonomously after: /gsd-autonomous (lifecycle only — all phases done).

## v3.6 Close — Tech Debt (live UATs now CLOSED 2026-06-19)

See `.planning/v3.6-MILESTONE-AUDIT.md` "Update 2026-06-19". The live visual/animation UATs that held completion are now RESOLVED (verified live on a clean v3.6 daemon; the prior stalls were a stale homebrew v3.5.1 daemon on the socket):

- **DONE — Live UATs (131/132/133):** CARD-08 both halves; ATTN-01/02/04/06 (pulse/float/badge) live; ATTN-05 by composition; DnD card→group; 11-session scale render; **remote tailnet peer (GRID-03/07) across two real machines.** See the 131/132/133-HUMAN-UAT.md files (all passed).
- **DONE — Daemon bugfix (commit `245414c2`):** macOS natural-exit now captures the real exit code (`Session.ReapNaturalExit()`), so CARD-08 stopped-err is reachable. TDD regression test; daemon+pty suites green with `-race`.
- **New candidate issue:** mini-preview issues one `GetSessionTailLines` RPC per session per ~3s tick (not a single batched call) — scale-perf nuance, non-blocking.
- **Residual (unchanged, release-eligible):** `adaptAllRemoteSessions` inline memo; HubModal `prefersReducedMotion` no change-listener; WR-04 GroupSidebar roving-tabindex → #97; GRID-04 errored-filter edge (audit-tracked).
- **Carry-forward (v3.5):** live UATs for Phases 124/125/126; operator pre-release tasks (RELEASE_PUBLISH_TOKEN PAT, WINGET_FIRST_SUBMISSION).

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
| Phase 132 P03 | 8 | 2 tasks | 4 files |
| Phase 132 P04 | 14 | 2 tasks | 4 files |
| Phase 133 P03 | 2 | 2 tasks | 2 files |
| Phase 134-modal-interaction P02 | 4m | 2 tasks | 3 files |
| Phase 134-modal-interaction P03 | 3m | 2 tasks | 4 files |
| Phase 134 P06 | ~25 minutes | 2 tasks | 4 files |
| Phase 134-modal-interaction P07 | 5m | 3 tasks | 5 files |
| Phase 134-modal-interaction P08 | 8m | 3 tasks | 7 files |
| Phase 135 P01 | 8m | 2 tasks | 2 files |
| Phase 135 P02 | 8m | 2 tasks | 4 files |
| Phase 135 P03 | 15 | 2 tasks | 2 files |

## v3.6 Phase Plan

| Phase | Name | Requirements | Status |
|-------|------|--------------|--------|
| 131 | Hub Foundation + Static Session Cards | HUB-01..04, CARD-01..06, CARD-08, GRID-01..02, GRID-04..06 (17) | Not started |
| 132 | Unified Grid + Mini Preview + Named Groups | CARD-07, GRID-03, GRID-07, GROUP-01..04 (7) | Not started |
| 133 | Attention + Pulse | ATTN-01..06 (6) | Complete 2026-06-17 |
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
| attentionIds live vs debouncedSortKey | Live Set<string> for immediate per-card border/icon; 1000ms debounced sort key for position reorder only (RESEARCH Pitfall 5) |
| sortSessionsForDisplay placement | Applied per group INSIDE groupByWorkDir/groupByNamedGroups — flat sessions list never sorted (RESEARCH Pitfall 7) |
| Remote terminal WS proxy (134-06) | `GET /api/relay/remote/{sid}/ws` on the relay surface; cap-gated via RemoteCapStore; injects `Origin: <baseURL>` on the upstream dial; copies frames on `r.Context()`. No new perm scope / cap cache / HTTP client. |
| relay.LoopbackOriginPatterns export | Exported so the daemon WS proxy reuses the relay's inbound-Origin allowlist verbatim (T-134-06-01); no pattern slice duplicated. |

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

- Start the next milestone with /gsd-new-milestone
