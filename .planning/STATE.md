---
gsd_state_version: 1.0
milestone: v3.1
milestone_name: Security Hardening
status: executing
stopped_at: Completed 87-03-webserver-enforcement-PLAN.md
last_updated: "2026-04-20T16:58:26Z"
last_activity: 2026-04-20
progress:
  total_phases: 4
  completed_phases: 0
  total_plans: 6
  completed_plans: 3
  percent: 50
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-19)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 87 — capability-based session authorization

## Current Position

Phase: 87 (capability-based-session-authorization) — EXECUTING
Plan: 4 of 6 (next: 87-04-daemon-api)
Status: Plans 01-03 complete — capability package landed and wired into webserver. All 9 Wave 0 SEC tests GREEN (SEC-02..SEC-05 implemented at HTTP layer).
Last activity: 2026-04-20 -- Phase 87 Plan 03 (webserver enforcement) complete

Progress: [█████░░░░░] 50% (3/6 Phase 87 plans complete)

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
- Phase 87 Plan 01: adopted a build-tag protocol (`phase87_wave1`, `phase87_wave2`) for Wave 0 RED skeletons so the capability package tests and webserver tests can reference yet-to-exist production symbols without breaking `go test ./...` on main — each wave's Plan un-tags its own files when production code lands
- Phase 87 Plan 01: placed webserver test helpers in `package webserver` (internal test helpers) rather than `package webserver_test` to avoid duplicating `selfSignedTLSForTest`/`testServer` names already present in `server_test.go`; both packages coexist in the same directory with no Go-level collision
- Phase 87 Plan 01: gated `FuzzVerify` by build tag instead of `f.Skip()` because fuzz entry-point Skip is silently dropped by the fuzzer harness
- Phase 87 Plan 02: source-grep constant-time regression guard (TestVerify_ConstantTimeComparison reads capability.go and asserts hmac.Equal is present, bytes.Equal literal absent) — forced a doc-comment rewrite but correctly guards against future maintainers replacing hmac.Equal with bytes.Equal
- Phase 87 Plan 02: SetClockForTest seam lives in export_test.go (only compiled during go test) rather than a production setter — clock injection is hermetically sealed to test builds, production code cannot import the helper
- Phase 87 Plan 02: JoinCodeManager uses plain sync.Mutex not RWMutex — RWMutex read lock would allow two goroutines to observe the entry before either deletes it, breaking single-use invariant (RESEARCH Pitfall 4); verified by 100-goroutine TestJoinCodeManager_ConcurrentExchangeIsAtomic
- Phase 87 Plan 03: Middleware shape is func(http.HandlerFunc) http.HandlerFunc (not func(http.Handler) http.Handler) so the wrapper can inspect r.PathValue("id") — critical for SEC-03 session-ID binding inside requireCapability
- Phase 87 Plan 03: All capability.Verify failure modes collapse to a single 401 body "capability required" — do not distinguish malformed / bad-sig / bad-claims to the caller (T-87-08 information-disclosure defense)
- Phase 87 Plan 03: requireCapability performs a defense-in-depth check of BOTH isGrantActive AND IsSessionEnabled — either alone is sufficient in the current code path but the redundancy guards against a future code path that touches only one (admin-revoke-without-disable, partial onExit cleanup, etc.)
- Phase 87 Plan 03: handleListSessions returns zero-or-one items (D-18) — /api/sessions is now a self-describe endpoint rather than an enumeration endpoint. No caller ever receives a list longer than one via HTTPS
- Phase 87 Plan 03: Subscriber.ReadOnly sourced from claims.Perms == "read" (D-24 / SEC-04) — ?readonly query string is completely removed from the write-gate path
- Phase 87 Plan 03: OriginPatterns ["*"] intentionally retained — Phase 88 handles WebSocket Origin allowlisting; removing it here would front-run Phase 88 scope
- Phase 87 Plan 03: TestWebServerToggle expects 403 (not 404) post-toggle-off — the cap is structurally valid; the middleware's revoked-path response is 403, not 404. This reflects D-15's "toggle-off revokes, doesn't make the URL a 404" contract
- Phase 87 Plan 03: TestSessionAccessWithoutAuth was inverted to assert 401 — the pre-Phase-87 "tailnet membership is sufficient" behavior is exactly what SEC-02/SEC-03 remove

### Pending Todos

- On milestone completion: comment on GitHub Issue #35 that it was addressed by v3.1, then close
- Decide Origin-header-absent policy during Phase 88 planning (reject outright, or require capability-bearing handshake?)

### Quick Tasks Completed

(carried from v3.0)

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260410-g0p | Delete future-features.txt + clean stale worktrees | 2026-04-10 | 7ab4520 | [260410-g0p](./quick/260410-g0p-delete-future-features-txt-clean-stale-w/) |
| 260412-l7k | Fix local network banner showing when Tailscale connected | 2026-04-12 | e768272 | [260412-l7k](./quick/260412-l7k-fix-local-network-banner-showing-when-ta/) |

### Plan Execution Metrics

| Phase | Plan | Duration | Tasks | Files | Commits |
|-------|------|----------|-------|-------|---------|
| 87 | 01 | 8min | 2 | 6 | a35e963, 5ca1f3e |
| 87 | 02 | 12min | 2 | 10 | dd1c15e, 6d2cf8f |
| 87 | 03 | 8min | 2 | 5 | 1703ccd, d269e62 |

### Blockers/Concerns

- Tailnet-wide trust model must be replaced with explicit capability/grant model — requires fresh UX decisions (token in URL? explicit Share button? revocation list?) — deferred to Phase 87 planning
- Read-only policy must be server-bound; client-issued `readonly=1` query param becomes an untrusted hint (addressed in Phase 87)
- WinGet first submission to microsoft/winget-pkgs deferred until first release is published (carried from v3.0)
- Capability rotation/revocation UI and audit logging are v3.2+ scope — v3.1 issues and verifies, but does not surface a revoke flow

## Session Continuity

Last session: 2026-04-20T16:58:26Z
Stopped at: Completed 87-03-webserver-enforcement-PLAN.md
Next action: `/gsd:execute-phase 87` to run Plan 04 (daemon API — wire signingKey + joinCodes at startup, issue capabilities on toggle-on, clear grants on toggle-off / session exit, IPC handlers for IssueCapabilities / ExchangeJoinCode / RegenerateSigningKey)

**Planned Phase:** 87 (capability-based-session-authorization) — 6 plans — 2026-04-20T13:47:57.212Z
