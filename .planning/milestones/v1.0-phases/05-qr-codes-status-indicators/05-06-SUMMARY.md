---
phase: 05-qr-codes-status-indicators
plan: "06"
subsystem: auth
tags: [go, http, webserver, dashboard, authentication, middleware]

# Dependency graph
requires:
  - phase: 05-qr-codes-status-indicators
    provides: Dashboard HTML with client-side auth detection (JS probes /api/sessions)
provides:
  - GET /dashboard served publicly without auth middleware
  - Login form reachable on first visit (circular dependency resolved)
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Client-side auth detection: HTML shell served without auth; JS probes /api/session to decide login vs dashboard view"

key-files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/webserver/server_test.go

key-decisions:
  - "GET /dashboard serves HTML without dashboardAuth wrapper — dashboard.html JS detects auth state by probing /api/sessions; the HTML shell must be publicly accessible for the login form to render"
  - "TestWebServerDashboardRequiresAuth renamed to TestWebServerDashboardNoAuthRequired — old test reflected pre-fix broken behaviour; updated to assert correct 200 response"

patterns-established:
  - "HTML shell pages that self-manage auth via client-side JS should be served without server-side auth middleware"

requirements-completed: [WEB-02]

# Metrics
duration: 5min
completed: 2026-03-18
---

# Phase 05 Plan 06: Dashboard Auth Fix Summary

**Removed dashboardAuth middleware from GET /dashboard, breaking the circular dependency where the login form was unreachable behind the auth it was trying to bypass**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-18T21:37:40Z
- **Completed:** 2026-03-18T21:43:00Z
- **Tasks:** 1 (TDD — 2 commits: test + feat)
- **Files modified:** 2

## Accomplishments

- GET /dashboard now returns 200 to unauthenticated clients — the login form is reachable
- All data API routes remain protected by dashboardAuth (/api/sessions, /api/sessions/{id}/qr, etc.)
- Added two new tests: TestDashboardNoAuthReturns200 and TestAPISessionsStillRequiresAuth
- Updated the old TestWebServerDashboardRequiresAuth to TestWebServerDashboardNoAuthRequired to reflect correct post-fix behaviour
- Full go test ./... -race passes green

## Task Commits

1. **Task 1 RED: Failing test for unauthenticated GET /dashboard** - `bc8f38f` (test)
2. **Task 1 GREEN: Remove dashboardAuth from GET /dashboard route** - `2abd124` (feat)

## Files Created/Modified

- `internal/webserver/server.go` - Removed dashboardAuth wrapper from GET /dashboard route registration; updated comment
- `internal/webserver/server_test.go` - Added TestDashboardNoAuthReturns200 and TestAPISessionsStillRequiresAuth; renamed existing conflicting test

## Decisions Made

- GET /dashboard serves HTML without dashboardAuth wrapper — the dashboard.html JavaScript already handles auth state detection by probing GET /api/sessions. If 401, it shows the login form; if 200, it shows the session list. The HTML shell must be publicly accessible for this pattern to work.
- Renamed TestWebServerDashboardRequiresAuth to TestWebServerDashboardNoAuthRequired because the old test asserted 401 behaviour that was the root of UAT gap 7 ("all I get is Unauthorized").

## Deviations from Plan

None — plan executed exactly as written. The one additional change (renaming the old conflicting test) was necessary for the test suite to pass and is within the spirit of the plan's TDD instructions.

## Issues Encountered

None. Single-line change with clear intent.

## Next Phase Readiness

- UAT gap 7 closed: unauthenticated visitors can now load the dashboard login form
- Phase 05 gap closure complete — all three UAT gaps (05-04 status dot, 05-05 terminal height, 05-06 dashboard auth) addressed
- Ready for Phase 06 (cross-platform validation)

---
*Phase: 05-qr-codes-status-indicators*
*Completed: 2026-03-18*

## Self-Check: PASSED

- `internal/webserver/server.go` — FOUND
- `internal/webserver/server_test.go` — FOUND
- `.planning/phases/05-qr-codes-status-indicators/05-06-SUMMARY.md` — FOUND
- Commits `bc8f38f` (test RED), `2abd124` (feat GREEN), `eaf60dc` (docs metadata) — all present in git log
