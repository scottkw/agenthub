---
gsd_state_version: 1.0
milestone: v3.1
milestone_name: Security Hardening
status: planning
stopped_at: Phase 87 context gathered
last_updated: "2026-04-20T02:35:24.009Z"
last_activity: 2026-04-19 — Roadmap generated (Phases 87-90)
progress:
  total_phases: 4
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-19)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** v3.1 Security Hardening (addresses GitHub Issue #35)

## Current Position

Phase: Not started (roadmap defined, ready for phase planning)
Plan: —
Status: Planning
Last activity: 2026-04-19 — Roadmap generated (Phases 87-90)

Progress: [          ] 0% (0/4 phases complete)

## Performance Metrics

**Velocity:**

- v3.0 plans completed: 9
- v3.0 phases: 4
- v3.0 timeline: 2026-04-18 → 2026-04-19 (2 days)
- Cumulative: 86 phases, 168 plans across 18 milestones

## Accumulated Context

### Decisions

- v3.1 scope derived from third-party security review (Codex) placed in `security-review/` (gitignored); 5 findings, all confirmed against v3.0 code
- Milestone addresses GitHub Issue #35 ("Security review")
- Skipping optional research step — findings themselves are the research
- Phase numbering continues from Phase 86 (v3.0 end) — Phases 87-90 allocated
- SEC-01..SEC-05 grouped into a single phase (Phase 87) because the capability token is the shared primitive for listing, WebSocket access, and read-only enforcement — splitting causes double-implementation of the token issuance and verification path
- Four-phase structure follows the security review's recommended implementation order: authorization first (Phase 87), handshake second (Phase 88 builds on 87's capability), vendoring and CSP third (Phase 89, independent), release pipeline fourth (Phase 90, CI/CD surface only)

### Pending Todos

- On milestone completion: comment on GitHub Issue #35 that it was addressed by v3.1, then close
- Decide capability token scheme during Phase 87 planning (HMAC with daemon-persisted secret vs. ed25519; base64-URL encoding; expiry semantics)
- Decide grant UX during Phase 87 planning (explicit Share button per session? copy-link gesture? QR contains the capability token?)
- Decide Origin-header-absent policy during Phase 88 planning (reject outright, or require capability-bearing handshake?)

### Quick Tasks Completed

(carried from v3.0)

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260410-g0p | Delete future-features.txt + clean stale worktrees | 2026-04-10 | 7ab4520 | [260410-g0p](./quick/260410-g0p-delete-future-features-txt-clean-stale-w/) |
| 260412-l7k | Fix local network banner showing when Tailscale connected | 2026-04-12 | e768272 | [260412-l7k](./quick/260412-l7k-fix-local-network-banner-showing-when-ta/) |

### Blockers/Concerns

- Tailnet-wide trust model must be replaced with explicit capability/grant model — requires fresh UX decisions (token in URL? explicit Share button? revocation list?) — deferred to Phase 87 planning
- Read-only policy must be server-bound; client-issued `readonly=1` query param becomes an untrusted hint (addressed in Phase 87)
- WinGet first submission to microsoft/winget-pkgs deferred until first release is published (carried from v3.0)
- Capability rotation/revocation UI and audit logging are v3.2+ scope — v3.1 issues and verifies, but does not surface a revoke flow

## Session Continuity

Last session: 2026-04-20T02:35:23.999Z
Stopped at: Phase 87 context gathered
Next action: `/gsd:plan-phase 87` to decompose capability-based session authorization into plans
