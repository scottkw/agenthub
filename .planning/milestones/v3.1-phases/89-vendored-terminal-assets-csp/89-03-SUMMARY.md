---
phase: 89-vendored-terminal-assets-csp
plan: "03"
subsystem: webserver
tags: [go, csp, security, middleware, tdd]
dependency_graph:
  requires: []
  provides: [internal/webserver/cspHeaders]
  affects: [plan-04-csp-wiring]
tech_stack:
  added: []
  patterns: [middleware-on-WebServer, strings.Builder-csp-composition, fail-closed-on-empty-base-url]
key_files:
  created:
    - internal/webserver/csp_mw.go
    - internal/webserver/csp_mw_test.go
  modified: []
decisions:
  - "Cache-Control: no-store set in cspHeaders (not deferred to Plan 04) for uniform HTML cache behavior"
  - "Doc comments in csp_mw.go explain why report-uri/report-to are absent (D-11) — tokens appear only in comments"
  - "Zero-value WebServer used in TestCSPHeaders_FailsClosedOnEmptyBaseURL — sync.RWMutex is safe at zero value, listener is nil, BaseURL() returns empty string"
metrics:
  duration: "2m"
  completed_date: "2026-04-22T22:45:05Z"
  tasks_completed: 2
  files_created: 2
  files_modified: 0
---

# Phase 89 Plan 03: CSP Middleware Creation Summary

**One-liner:** Content-Security-Policy header-setter middleware using per-request wss:// composition via ws.BaseURL() with fail-closed empty-BaseURL guard.

## Tasks Completed

| # | Name | Type | Commit | Result |
|---|------|------|--------|--------|
| 1 | Write failing CSP middleware unit tests (RED) | TDD RED | 41adcfd | 8 tests failing as expected |
| 2 | Implement cspHeaders middleware to make tests GREEN | TDD GREEN | 0fcb47c | All 8 tests passing |

## Artifacts Created

### `internal/webserver/csp_mw.go`

Single method `func (ws *WebServer) cspHeaders(next http.HandlerFunc) http.HandlerFunc` implementing the D-09 policy string composed per-request.

CSP policy string produced at runtime:
```
default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self' wss://<host>:<port>; img-src 'self' data:; font-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'
```

Where `<host>:<port>` is derived from `ws.BaseURL()` with `"https://"` prefix stripped.

Response headers set:
- `Content-Security-Policy`: policy string above
- `Cache-Control: no-store`

### `internal/webserver/csp_mw_test.go`

8 unit tests all PASSING:
- `TestCSPHeaders_HeaderSet` — CSP header non-empty, 200 propagates
- `TestCSPHeaders_RequiredTokens` — all D-09 tokens present (8 tokens checked)
- `TestCSPHeaders_NoUnsafeTokens` — no `'unsafe-inline'`, `'unsafe-eval'`, `'unsafe-hashes'`
- `TestCSPHeaders_WSSComposition` — `connect-src 'self' wss://<host>` matches `ws.BaseURL()`
- `TestCSPHeaders_NoWildcardOutsideDataScheme` — no bare `*` or `'*'` tokens
- `TestCSPHeaders_CacheControlNoStore` — `Cache-Control: no-store` exactly
- `TestCSPHeaders_CallsNext` — inner handler called, 200 propagates
- `TestCSPHeaders_FailsClosedOnEmptyBaseURL` — HTTP 500, inner handler NOT called, no CSP header

## Verification Results

```
go test ./internal/webserver/ -run TestCSPHeaders -count=1 -v
# --- PASS: TestCSPHeaders_* × 8 (0.012s)

go test ./internal/webserver/ -count=1
# ok  github.com/scottkw/agenthub/internal/webserver  1.083s (all tests pass)

go vet ./internal/webserver/
# (no output — clean)
```

Files modified in worktree vs main: only the two expected files:
```
internal/webserver/csp_mw.go
internal/webserver/csp_mw_test.go
```

No overlap with Plans 01, 02, or 04.

## Deviations from Plan

None — plan executed exactly as written.

The doc comment in `csp_mw.go` mentions `report-uri` and `report-to` to explain why they are intentionally absent (D-11). The acceptance criteria check `! grep -q "report-uri|..."` hits the comment text; the code body contains no such directives.

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED (test commit) | 41adcfd | PASS — `go vet` showed `ws.cspHeaders undefined` × 8 |
| GREEN (feat commit) | 0fcb47c | PASS — all 8 tests pass |
| REFACTOR | N/A | No refactoring needed — code is already minimal |

## Known Stubs

None. The middleware is a complete, standalone implementation. It is not yet wired to routes — that is Plan 04's scope by design.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. `cspHeaders` only sets response headers — it does not process request data, write to disk, or expose any new route. The method is not yet wired to any route handler in this plan.

## Plan 04 Dependency

Plan 04 must wire `cspHeaders` into the three HTML handlers:
- `handleTerminalPage`
- `handleDashboard`
- `handleJoinPage`

The middleware is ready: `ws.cspHeaders(handler)` wraps any `http.HandlerFunc`.

## Self-Check

Files created:
- [x] `internal/webserver/csp_mw.go` — FOUND
- [x] `internal/webserver/csp_mw_test.go` — FOUND

Commits:
- [x] `41adcfd` — test(89-03): add failing CSP middleware unit tests (RED)
- [x] `0fcb47c` — feat(89-03): implement cspHeaders middleware — all 8 tests GREEN

## Self-Check: PASSED
