---
gsd_state_version: 1.0
milestone: v3.6
milestone_name: Hub (Session Grid)
status: planning
last_updated: "2026-06-16T16:39:50.222Z"
last_activity: 2026-06-16
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
**Current focus:** Milestone complete

## Current Position

Phase: Not started (defining requirements)
Plan: —
Status: Defining requirements
Last activity: 2026-06-16 — Milestone v3.6 started

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
- Cumulative: 25 milestones shipped (v1.0–v3.5), 128 phases, ~260 plans

## Session Continuity

Last session: 2026-06-16T06:03:19.577Z
Stopped at: Completed 130-03-PLAN.md
Resume file: None
Next action: Execute plan 130-04 (frontend rewire)

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
| bookkeeping | Nyquist frontmatter `nyquist_compliant:false` on Phases 123/125/126/127 | advisory; tests green |
| Phase 129-write-concurrency-fix-dns-error-ux P01 | 4m | 3 tasks | 3 files |
| Phase 130-remote-browse-gui-on-ramp P01 | 20 | 3 tasks | 4 files |
| Phase 130 P03 | 20 | 3 tasks | 4 files |
| Phase 130-remote-browse-gui-on-ramp P04 | 15 | 2 tasks | 5 files |

## v3.5.1 Phase Plan

| Phase | Name | Requirements | Status |
|-------|------|--------------|--------|
| 129 | Write Concurrency Fix + DNS Error UX | RACE-01..03, DNS-01..03 (6) | Not started |
| 130 | Remote Browse GUI On-Ramp | RB-01..05 (5) | Not started |

## Key Decisions (v3.5.1)

| Decision | Resolution |
|----------|------------|
| #87 If-Match concurrency contract | RESOLVED 2026-06-15 → **(a) per-path lock for true single-winner guarantee**. Serialize writes per path; loser gets a clean 412/conflict. Matches standard If-Match optimistic-concurrency semantics. Code, comments, and remote-write proxy must all assert single-winner (RACE-02). |
| #86 remote-browse architecture | RESOLVED 2026-06-15 → **(a) tailnet-trusted metadata-only discovery endpoint**. Returns shareable-session metadata to tailnet-trusted callers; content/caps stay locked (preserves Phase 87/88 no-enumeration model, RB-03). Satisfies RB-01 (see sessions) + RB-04 (honest states). |
| 130-03: FetchPeerSessionsMetaWithClient | Added alongside FetchPeerSessionsMeta for httptest TLS bypass — mirrors FetchPeerSessionsWithClient pattern; tests updated to use WithClient where httptest cert trust is required |
| 130-03: GetRemoteSessions retained | Not deleted during plan-03; plan-04 frontend rewire will swap the App.tsx callsite to GetRemoteSessionsWithMeta |

## Already-Fixed Prerequisites (do not re-scope)

| Fix | Commit | Status |
|-----|--------|--------|
| Relay routes not mounted (remote `/api/files/remote/{sid}/...` 404'd) | `58af6d6` | FIXED — proven live |
| Discovery probe rejected cap-protected peers (#84) | `3508bd7` | FIXED — proven live |
| Share-link scope relabel (#24 UX) | `e45ccba` | FIXED |

## v3.5 Plan Execution Log

| Phase | Plan | Status | Notes |
|-------|------|--------|-------|
| 123 | 01..04 | Complete 2026-06-14 | Write primitives + denylist + daemon routes (FSW-01..12) |
| 124 | 01..05 | Complete 2026-06-14 | files.write cap + requireFilesWrite + CSRF + schemaV4 (CAP-01..10) |
| 125 | 01..06 | Complete 2026-06-14 | CodeMirror 6 editor + 51/51 cross-browser e2e (EDIT-01..13) |
| 126 | 01..04 | Complete 2026-06-15 | $EDITOR shell-out + d/r/m + 8-method interface (TUIW-01..07) |
| 127 | 01..04 | Complete 2026-06-15 | Security hardening audit (SEC-01..07) |
| 128 | 01..04 | Complete 2026-06-15 | Remote write parity + 3-observer proof (RMW-01..06) |
| UAT | Two-machine tailnet (RMW-06) | EXECUTED 2026-06-15 | Data path proven live; 4-layer GUI on-ramp breakage discovered; layers 1+3 fixed; layer 4 (#86) → Phase 130 |

## Operator Next Steps

- Start the next milestone with /gsd-new-milestone
