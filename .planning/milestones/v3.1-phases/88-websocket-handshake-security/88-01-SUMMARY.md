---
phase: 88-websocket-handshake-security
plan: "01"
subsystem: webserver-origin-allowlist
tags: [security, websocket, middleware, origin-check, go, sec-06]
dependency_graph:
  requires: [87-06]
  provides: [origin-allowlist-gate, sc-1-cross-site-reject, sc-3-missing-origin-reject, sc-4-webserver-regression-guards]
  affects: [internal/webserver]
tech_stack:
  added: []
  patterns: [middleware-wrapping, source-grep-regression-guard, belt-and-suspenders-defense-in-depth]
key_files:
  created:
    - internal/webserver/origin_mw.go
    - internal/webserver/origin_mw_test.go
    - internal/webserver/origin_integration_test.go
    - internal/webserver/security_regression_test.go
  modified:
    - internal/webserver/server.go
    - internal/webserver/capability_test.go
    - internal/webserver/server_test.go
decisions: []
metrics:
  duration_minutes: 5
  tasks_completed: 2
  files_created: 4
  files_modified: 3
  completed_date: "2026-04-22"
requirements: [SEC-06]
---

# Phase 88 Plan 01: WebServer Origin Allowlist Summary

**One-liner:** Strict byte-for-byte Origin allowlist middleware (`requireAllowedOrigin`) wired outside `requireCapability` on the WSS route, replacing `OriginPatterns:[]string{"*"}` with `ws.allowedOrigins()`, locked by SC-4 source-grep regression guards.

## Files Created / Modified

### Created

| File | Purpose |
|------|---------|
| `internal/webserver/origin_mw.go` | `requireAllowedOrigin` middleware + `allowedOrigins()` helper (D-01/D-03/D-05/D-11/D-12) |
| `internal/webserver/origin_mw_test.go` | 7 unit tests: match, mismatch, missing Origin, case sensitivity, "null" literal, fail-closed on nil listener, allowedOrigins singleton |
| `internal/webserver/origin_integration_test.go` | 6 integration tests: SC-1 cross-site reject, SC-3 missing reject, SC-2 tailscale accept, short-circuit proof, D-07 body check, D-12 library belt-and-suspenders |
| `internal/webserver/security_regression_test.go` | SC-4 source-grep guards: `TestSecurity_NoAcceptAllOriginInWebserver` + `TestSecurity_WebserverOriginAllowlistWiredToBaseURL` |

### Modified

| File | Change |
|------|--------|
| `internal/webserver/server.go` | Route wiring: `requireAllowedOrigin(requireCapability(handleWSSRelay))` (D-10); AcceptOptions: `OriginPatterns: ws.allowedOrigins()` replacing `[]string{"*"}` (D-12) |
| `internal/webserver/capability_test.go` | Added `Origin: ws.BaseURL()` header to 3 existing WS dial calls that broke due to new middleware (Rule 1 fix) |
| `internal/webserver/server_test.go` | Added `Origin: ws.BaseURL()` header to `TestWebServerWSS` dial call (Rule 1 fix) |

## Success Criteria Satisfied

| SC | Description | Status |
|----|-------------|--------|
| SC-1 | Cross-site Origin + valid cap → 403 before capability check | PASSED — `TestSecurity_WebSocketRejectsCrossSiteOrigin` + `TestSecurity_OriginCheckShortCircuitsBeforeCapability` |
| SC-2 (tailscale half) | Origin == ws.BaseURL() + valid cap → 101 WebSocket upgrade | PASSED — `TestSecurity_WebSocketAcceptsMatchingOriginTailscaleMode` |
| SC-3 | Missing Origin → 403 | PASSED — `TestSecurity_WebSocketRejectsMissingOrigin` + `TestRequireAllowedOrigin_MissingOriginRejected` |
| SC-4 (webserver half) | `OriginPatterns:[]string{"*"}` absent; `ws.allowedOrigins()` present | PASSED — `TestSecurity_NoAcceptAllOriginInWebserver` + `TestSecurity_WebserverOriginAllowlistWiredToBaseURL` (both GREEN) |

**SC-2 local-HTTPS-fallback half** deferred to phase-level UAT checkpoint (88-VALIDATION.md manual items) — requires a live self-signed cert + browser trust dismissal.

## Key Decisions / Rationale

All implementation decisions were pre-locked in CONTEXT.md D-01 through D-14. No new decisions were required during execution.

Cross-references to locked decisions in the implementation:

- **D-01/D-11** — `ws.BaseURL()` called per-request under existing `ws.mu.RLock()` (negligible cost; avoids snapshot invalidation complexity)
- **D-03** — byte-for-byte exact match; no case-folding, port-stripping, or normalization
- **D-05** — missing `Origin` header always rejected with 403 (non-browser clients use localhost relay, Plan 02 scope)
- **D-07** — single generic `"forbidden"` body for all Origin rejections; no distinction leaks rejection reason (T-88-05 information-disclosure defense)
- **D-10** — composition order: `requireAllowedOrigin` outside `requireCapability`; cross-site rejection short-circuits before HMAC work
- **D-12** — belt-and-suspenders: `OriginPatterns: ws.allowedOrigins()` at AcceptOptions site (library-layer second check)
- **D-13** — source-grep regression guards lock SC-4 contract; future maintainer cannot silently reintroduce `"*"` without failing the test
- **D-14** — no logs/metrics on Origin rejection (matches Phase 87 minimal-observability stance)

## Regression Test Outcomes

Both SC-4 source-grep guards are GREEN after Task 2:

- `TestSecurity_NoAcceptAllOriginInWebserver` — was RED (expected) during Task 1 TDD RED phase; turned GREEN when `server.go` was updated in Task 2
- `TestSecurity_WebserverOriginAllowlistWiredToBaseURL` — GREEN throughout (server.go retains `OriginPatterns:` field; now references `ws.allowedOrigins()`)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Phase 87 tests broke when Origin middleware was wired**

- **Found during:** Task 2 full suite run
- **Issue:** Three `dialWebServerWS` calls in `capability_test.go` and one `websocket.Dial` in `server_test.go` passed `nil` headers (no `Origin`). Pre-Phase-88 these succeeded because `OriginPatterns:[]string{"*"}` accepted everything. After wiring `requireAllowedOrigin`, missing Origin returns 403, breaking those 4 tests.
- **Fix:** Added `Origin: ws.BaseURL()` header to each affected dial call. This is the correct fix — those tests are asserting Phase 87 write-gate behavior; the Origin header is now a prerequisite for any WS upgrade, so the tests must supply it.
- **Files modified:** `internal/webserver/capability_test.go`, `internal/webserver/server_test.go`
- **Commit:** f9dc908

### Library Belt-and-Suspenders Test (Task 2 test 6)

`TestSecurity_LibraryLayerRejectsCrossSiteOriginWhenMiddlewareBypassed` was kept — the `httptest.NewTLSServer` approach was clean and the test passed on first attempt. The test uses `ws.allowedOrigins()` directly against the library's `AcceptOptions` in a bare handler (no middleware), confirming D-12 defense-in-depth works independently.

## Known Stubs

None. All data flows are wired; no placeholder values or TODO markers in the delivered code.

## Threat Flags

None. No new network endpoints, auth paths, or trust boundary surface was introduced. This plan only adds restrictions to an existing endpoint.

## Self-Check

### Files Exist
- `internal/webserver/origin_mw.go` — FOUND
- `internal/webserver/origin_mw_test.go` — FOUND
- `internal/webserver/origin_integration_test.go` — FOUND
- `internal/webserver/security_regression_test.go` — FOUND

### Commits Exist
- Task 1: 66bf80e — FOUND
- Task 2: f9dc908 — FOUND

### Test Suite
- `go test ./internal/webserver/ -count=1` — PASSED (no Phase 87 regression)
- `go vet ./internal/webserver/` — CLEAN
- `gofmt -l` — CLEAN

## Self-Check: PASSED
