---
phase: 60-local-network-fallback
plan: 01
subsystem: webserver
tags: [tls, auth, networking, local-mode]
dependency_graph:
  requires: []
  provides: [local-mode-webserver-infra]
  affects: [internal/webserver]
tech_stack:
  added: []
  patterns: [self-signed-tls-p256, http-basic-auth-middleware, lan-ip-selection]
key_files:
  created:
    - internal/webserver/selfcert.go
    - internal/webserver/selfcert_test.go
    - internal/webserver/auth.go
    - internal/webserver/auth_test.go
    - internal/webserver/localip.go
    - internal/webserver/localip_test.go
  modified:
    - internal/webserver/server.go
    - internal/webserver/server_test.go
decisions:
  - "Export IsTailscaleIP (not just isTailscaleIP) so _test package can validate CGNAT boundary conditions without internal symbol access"
  - "Export BasicAuthMiddleware alongside unexported basicAuthMiddleware alias so tests validate via exported form while internal code uses unexported form per plan spec"
  - "startLocal() checks ws.config.TLSConfig != nil for test override before calling GenerateSelfSignedCert, matching startTailscale() test override pattern"
metrics:
  duration: ~8 minutes
  completed: 2026-04-09
  tasks_completed: 2
  files_changed: 8
requirements: [NET-01, NET-02, NET-03]
---

# Phase 60 Plan 01: Webserver Local Mode Infrastructure Summary

**One-liner:** P256 self-signed TLS cert generation, HTTP Basic Auth middleware, and LAN IP selection with Tailscale CGNAT exclusion — plus mode-aware Config/Start/BaseURL dispatch in WebServer.

## What Was Built

Three new Go files provide the foundational infrastructure for local-network operation when Tailscale is unavailable:

- **selfcert.go** — `GenerateSelfSignedCert(ip string) (*tls.Config, error)`: generates an in-memory P256 CA+leaf cert with IP SAN. Uses `elliptic.P256()` explicitly (not P521, which Chrome rejects).
- **auth.go** — `basicAuthMiddleware(password string) func(http.Handler) http.Handler`: wraps any `http.Handler`, returns 401 + `WWW-Authenticate: Basic realm="AgentHub"` on missing/wrong credentials.
- **localip.go** — `GetLANIP() (string, error)`: returns the best private IPv4 address, preferring en0/eth0/wlan0, skipping Tailscale CGNAT (100.64.0.0/10), loopback, and link-local addresses.

**server.go** was updated with:
- `Config.Mode string` and `Config.Password string` fields
- `Mode()` accessor method
- `Start()` dispatches to `startLocal()` or `startTailscale()` based on `Config.Mode`
- `startLocal()` calls `GenerateSelfSignedCert` + wraps mux with `basicAuthMiddleware`
- `startTailscale()` preserves the existing Tailscale `lc.GetCertificate` path
- `BaseURL()` returns IP-based URL in local mode, FQDN-based in tailscale mode

## Test Coverage

36 total webserver tests pass (all existing + 12 new):

| Test | What it verifies |
|------|-----------------|
| TestGenerateSelfSignedCert | P256 cert, IP SAN present, TLS handshake succeeds |
| TestGenerateSelfSignedCert_TLSConfig | HTTPS server starts and returns 200 |
| TestBasicAuthMiddleware_Unauthorized | 401 + WWW-Authenticate on no credentials |
| TestBasicAuthMiddleware_WrongPassword | 401 on wrong password |
| TestBasicAuthMiddleware_Authorized | 200 with correct password |
| TestBasicAuthMiddleware_EmptyUsername | 200 with empty username (only password matters) |
| TestGetLANIP | Non-empty, non-loopback IPv4 returned |
| TestGetLANIP_ExcludesTailscale | CGNAT boundary conditions (100.63.x, 100.64.x, 100.127.x, 100.128.x) |
| TestLocalModeStart | Server starts in local mode; 401 without auth, 200 with correct password |
| TestBaseURL_LocalMode | IP-based URL returned (not FQDN) |
| TestBaseURL_TailscaleMode | FQDN-based URL returned (existing behavior preserved) |
| TestMode_Accessor | Mode() returns configured mode string |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] nil context in TestGenerateSelfSignedCert**
- **Found during:** Task 1 GREEN phase
- **Issue:** Test called `dialer.DialContext(nil, ...)` which panics — Go's net.Dialer requires a non-nil context
- **Fix:** Changed to `context.Background()`
- **Files modified:** internal/webserver/selfcert_test.go
- **Commit:** 6507ecb

**2. [Design - Exported helpers for test package access]**
- **Found during:** Task 1 implementation
- **Issue:** Plan specified `isTailscaleIP` as unexported, but tests in `webserver_test` package (external test package) cannot access unexported symbols
- **Fix:** Exported `IsTailscaleIP` and `BasicAuthMiddleware`; added unexported aliases (`isTailscaleIP`, `basicAuthMiddleware`) for internal use per plan spec
- **Files modified:** internal/webserver/localip.go, internal/webserver/auth.go
- **Commit:** 6507ecb

## Known Stubs

None. All implemented functions are fully wired and testable.

## Threat Flags

None. This plan adds internal Go infrastructure with no new network endpoints, auth paths, or schema changes. The webserver's routes and external surface are unchanged — mode dispatch happens at server startup time and is configured by the daemon layer (Phase 60 Plan 02).

## Self-Check

### File existence:
- internal/webserver/selfcert.go: FOUND
- internal/webserver/auth.go: FOUND
- internal/webserver/localip.go: FOUND
- internal/webserver/selfcert_test.go: FOUND
- internal/webserver/auth_test.go: FOUND
- internal/webserver/localip_test.go: FOUND

### Commits:
- 6507ecb: feat(60-01): add selfcert.go, auth.go, localip.go with tests
- 48bf72c: feat(60-01): add Mode+Password to Config, startLocal/startTailscale dispatch, mode-aware BaseURL

## Self-Check: PASSED
