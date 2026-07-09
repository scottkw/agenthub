---
phase: 176-platform-hardening-bug-fixes
plan: 02
subsystem: webserver
tags: [csp, security-headers, go, http, funnel]

requires:
  - phase: 89-csp-header-hardening
    provides: ws.cspHeaders middleware + assertCSPHeaderStrict five-assertion helper, already applied to /dashboard, /join, /sessions/{id}
provides:
  - GET /app/ now flows through ws.cspHeaders — the public guest SPA surface (exposed via Tailscale Funnel) carries the same strict Content-Security-Policy header as its sibling HTML routes
  - TestCSPHeaderStrict_App regression test proving the header is present on a 200 /app/ response
affects: [176-04 (TESTING.md traceability + manual item M-53), any future /app/ route changes]

tech-stack:
  added: []
  patterns:
    - "cspHeaders wraps a route's handler literal directly (no capability gate) when the route has no requireCapability/requireAllowedOrigin to nest inside"
    - "Integration tests for capability-agnostic static routes wire a stub fstest.MapFS via SetStaticAppFS before asserting on the response, mirroring app_bundle_test.go's pattern"

key-files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/webserver/csp_integration_test.go

key-decisions:
  - "Reused ws.cspHeaders verbatim (D-05) — no new SPA-tailored CSP policy authored, no edits to csp_mw.go"
  - "Test wires a stub fstest.MapFS via SetStaticAppFS (mirroring app_bundle_test.go) since testServer(t) does not wire a static app FS by default and /app/ would otherwise 503"

patterns-established:
  - "New TestCSPHeaderStrict_* tests should reuse the shared assertCSPHeaderStrict helper rather than hand-rolling header assertions"

requirements-completed: [BUG-06]

coverage:
  - id: D1
    description: "GET /app/ carries the strict Content-Security-Policy header (same policy as /dashboard, /join, /sessions/{id})"
    requirement: "BUG-06"
    verification:
      - kind: unit
        ref: "internal/webserver/csp_integration_test.go#TestCSPHeaderStrict_App"
        status: pass
    human_judgment: false
  - id: D2
    description: "Production /app/ SPA loads without a breaking CSP violation across inline scripts, wasm-unsafe-eval, SSE/WS connect-src, font-src, img-src data: (D-06 console sweep)"
    verification: []
    human_judgment: true
    rationale: "/app/ returns 503 under wails dev / plain go build (no embedded SPA); requires a production Vite build (wails build -tags \"webkit2_41,wailsassets\") and a browser DevTools console read — cannot be automated in this environment. Tracked as manual item M-53 in plan 176-04."

duration: 8min
completed: 2026-07-09
status: complete
---

# Phase 176 Plan 02: /app/ CSP Header Hardening Summary

**Wrapped the public guest SPA route `/app/` in the existing `ws.cspHeaders` middleware, closing the BUG-06 (#123) gap where the sole unauthenticated Funnel-exposed surface served zero Content-Security-Policy protection.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-09T13:12:35Z
- **Completed:** 2026-07-09T13:20:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `mux.HandleFunc("GET /app/", ...)` now wraps its handler literal in `ws.cspHeaders(...)`, so every response from the public guest SPA carries the same strict policy already enforced on `/dashboard`, `/join`, and `/sessions/{id}` (`default-src 'none'`, `script-src 'self' 'wasm-unsafe-eval'`, `connect-src 'self' wss://<host>`, `frame-ancestors 'none'`, etc.)
- Added `TestCSPHeaderStrict_App`, wiring a stub `fstest.MapFS` via `SetStaticAppFS` to clear the 503 no-bundle guard, then reusing the existing `assertCSPHeaderStrict` helper — confirmed as a real regression gate by temporarily reverting the Task 1 wrap and observing the test fail on the missing header
- No new CSP policy authored; `csp_mw.go` untouched

## Task Commits

Each task was committed atomically:

1. **Task 1: Wrap the /app/ route registration in ws.cspHeaders(...)** - `ec078c56` (fix)
2. **Task 2: Add TestCSPHeaderStrict_App asserting the CSP header on an /app/ response** - `e287c3dd` (test)

**Plan metadata:** (pending — this SUMMARY commit)

## Files Created/Modified
- `internal/webserver/server.go` - `/app/` route handler literal wrapped in `ws.cspHeaders(...)`; inline comment added documenting the Phase 176 BUG-06 rationale (D-05); 503 fallback and SPA routing logic unchanged
- `internal/webserver/csp_integration_test.go` - New `TestCSPHeaderStrict_App` (wires a stub `fstest.MapFS`, asserts 200, delegates to `assertCSPHeaderStrict`); added `testing/fstest` import

## Decisions Made
- Reused `ws.cspHeaders` verbatim per D-05 — no speculative loosening, no new policy
- Test wires the stub app FS via `SetStaticAppFS` (the OPEN QUESTION from the plan's `<read_first>` was confirmed: `testServer(t)` does not wire a static app FS, only `app_bundle_test.go` does)

## Deviations from Plan

None - plan executed exactly as written. One minor addition beyond the plan's literal action: added an inline code comment above the `/app/` route registration explaining the Phase 176 BUG-06 rationale, matching this codebase's existing documentation-heavy convention for security-relevant route wrappers (not a deviation rule trigger — purely explanatory, no behavior change).

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- BUG-06 (#123) is code-complete and regression-gated by `TestCSPHeaderStrict_App`
- Plan 176-04 must add a Section 4 traceability row for `csp_integration_test.go` (BUG-06) and manual item **M-53** (production-build `/app/` CSP console sweep — D-06, opportunistic/human-checked, requires `wails build -tags "webkit2_41,wailsassets"`)
- No blockers for the remaining 176 plans (176-03, 176-04)

## Self-Check: PASSED

- FOUND: internal/webserver/server.go (grep confirms `mux.HandleFunc("GET /app/", ws.cspHeaders(`)
- FOUND: internal/webserver/csp_integration_test.go (TestCSPHeaderStrict_App present, passes)
- FOUND commit ec078c56 in git log
- FOUND commit e287c3dd in git log
- `go build ./...` clean; `go test ./internal/webserver/... -count=1` all pass

---
*Phase: 176-platform-hardening-bug-fixes*
*Completed: 2026-07-09*
