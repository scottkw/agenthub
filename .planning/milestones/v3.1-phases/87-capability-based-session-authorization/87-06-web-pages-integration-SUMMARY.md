---
phase: 87-capability-based-session-authorization
plan: 06
subsystem: web
tags: [security, web, html, capability]

# Dependency graph
requires:
  - 87-03 WebServer setters (signingKey, joinCodes field, requireCapability middleware)
  - 87-04 Wails bindings and daemon /join/exchange handler contract
  - 87-05 QR URL shape (SessionSharePanel encodes /join?code= which this plan serves)
provides:
  - /dashboard public landing page (D-17 — no session list)
  - /join?code= 5-state client-side routed page
  - POST /join/exchange handler with 303/410/404 error mapping
  - /api/sessions/{id}/info perms field sourced from verified capability claims (D-23)
  - terminal.html caret suppression driven solely by perms (no ?readonly query)
affects:
  - 87-VERIFICATION phase gate — end-to-end capability flow now live web-side

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Client-side 5-state routing in join.html driven by ?code= and ?error= query params — one embed file, five visual surfaces, no server branching cost"
    - "Fail-safe default perms='read' in terminal.html — if GET /api/sessions/{id}/info fails for any reason, the UI assumes read-only and disables stdin; caret is never enabled during the async fetch"
    - "Perms sourced from claims.ClaimsFromContext in handleSessionInfo — the only input is the signed capability; no query parameter can promote read to write"
    - "HTTP 303 See Other on /join/exchange success — preserves the idempotent GET on the redirect target (/sessions/{id}?cap=<token>) and prevents form resubmission on refresh"

key-files:
  created:
    - web/join.html
  modified:
    - web/dashboard.html
    - web/embed.go
    - web/terminal.html
    - internal/webserver/server.go
    - internal/webserver/capability_test.go

key-decisions:
  - "Dashboard replaces the legacy session-list page entirely (D-17). Per-user discovery is gone — capability tokens must be explicitly shared. Keeps the threat model clean: a tailnet-reachable attacker at /dashboard sees zero session metadata."
  - "Public routes (GET /dashboard, GET /join, POST /join/exchange) are registered in setupRoutes WITHOUT requireCapability. All three are designed to be pre-authorization surfaces; any attempted session access still goes through the signed capability."
  - "handleJoinExchange nil-checks ws.joinCodes and ws.signingKey defensively and returns 500 if un-wired rather than silently accepting. Prefers hard failure over silent corruption (CLAUDE.md principle)."
  - "terminal.html reads ?cap= from the URL (NOT ?token=) aligning with the capability model (D-08). Legacy ?readonly= query is deleted — read-only state is derived exclusively from the server-verified perms claim (D-23, Pitfall 7)."
  - "xterm initialization wrapped inside the async perms-fetch IIFE so the caret is never briefly enabled for a read-only cap during the fetch round-trip — eliminates the TOCTOU flicker."
  - "Session lookup in handleJoinExchange treats 'session no longer web-enabled' distinctly from 'invalid code' — returns /join?error=session-gone (State E) instead of /join?error=invalid, surfacing honest state to the user."

requirements-completed: [SEC-01, SEC-04]

# Metrics
duration: ~15m
completed: 2026-04-20
---

# Phase 87 Plan 06: Web Pages Integration Summary

**Ships the web-side capability flow: a public /dashboard landing page (D-17) with no session list, a five-state /join?code= page with client-side routing, and a POST /join/exchange handler that 303-redirects to /sessions/{id}?cap=<token> on success (410 on expired, 404 on invalid, session-gone error path for toggled-off sessions). The terminal page is hardened: caret suppression, stdin disabling, and the READ ONLY badge are now driven exclusively by the verified perms claim returned from /api/sessions/{id}/info?cap=... — the legacy ?readonly query parameter is deleted from the write-gate path. TestEndToEnd_CapabilityFlow is GREEN, asserting perms round-trips through the cap and that ClearGrants revokes access.**

## Performance

- Duration: ~15 minutes (three atomic commits + metadata)
- Commits: 3 code (`1a0fb60`, `e7f315e`, `a87e2bb`) + 1 metadata
- Files created: 1 (`web/join.html`)
- Files modified: 5 (`web/dashboard.html`, `web/embed.go`, `web/terminal.html`, `internal/webserver/server.go`, `internal/webserver/capability_test.go`)

## Verification

- `go test ./internal/webserver/ -count=1 -v -run TestEndToEnd_CapabilityFlow` — PASS
- `go test ./... -count=1` — all project packages GREEN (pre-existing gitignored `security-review/` scaffold excluded)
- `grep -q 'Join a Shared Session' web/dashboard.html` — PASS
- `! grep -q 'session-list\|renderSessions' web/dashboard.html` — PASS (0 matches)
- `! grep -qE "get\\s*\\(\\s*['\\\"]readonly['\\\"]" web/terminal.html` — PASS (only `params.get('cap')` remains)

## Deviations

None. All three tasks executed per plan. No auto-fixes required. Agent session timed out before writing this SUMMARY.md file; orchestrator completed the metadata commit after verifying all code commits and gate checks passed.

## Self-Check: PASSED
