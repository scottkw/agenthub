---
phase: 15-tailscale-tls-interface-binding
plan: "01"
subsystem: webserver
tags: [tls, tailscale, refactor, go]
dependency_graph:
  requires: []
  provides: [tailscale-tls-backend, fqdn-base-url, tls-config-override-for-tests]
  affects: [internal/webserver]
tech_stack:
  added: [tailscale.com/client/local (GetCertificate)]
  patterns: [GetCertificate hook for dynamic cert provisioning, TLSConfig override for test isolation]
key_files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/webserver/server_test.go
  deleted:
    - internal/webserver/tls.go
    - internal/webserver/tls_test.go
decisions:
  - "Use InsecureSkipVerify for fresh token-test clients rather than sharing CA across selfSignedTLSForTest calls; these tests verify auth logic, not TLS trust"
  - "testCookieJar defined in server_test.go (external test package); equivalent to simpleCookieJar in server.go"
metrics:
  duration: 215s
  completed: "2026-03-20"
  tasks_completed: 2
  files_modified: 4
---

# Phase 15 Plan 01: Tailscale TLS Infrastructure Replacement Summary

**One-liner:** Replaced self-signed CA+leaf TLS with Tailscale `lc.GetCertificate` hook; added FQDN-based BaseURL and TLSConfig test override; deleted `tls.go` and `tls_test.go` entirely.

## What Was Built

- **Config struct** now has `FQDN string` and `TLSConfig *tls.Config` fields; `ConfigDir` removed
- **WebServer struct** stripped of `caKey`, `caCert`, `caDER`, `tlsCfg` fields
- **NewWebServer** no longer calls `LoadOrCreateCA` or writes cert files
- **Start()** uses `lc.GetCertificate` in production; `TLSConfig` override in tests
- **BaseURL()** returns `https://{ws.config.FQDN}:{port}` instead of IP-based URL
- **Deleted**: `TestClient()`, `handleCACert`, `GenerateCA`, `LoadOrCreateCA`, `GenerateLeafCert`, `BuildTLSConfig`, `ExportCACertPath`
- **Tests**: `selfSignedTLSForTest` helper generates per-test in-memory CA+leaf; all 19 tests pass

## Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Rewrite server.go, delete tls.go | c9402e4 | server.go (modified), tls.go (deleted) |
| 2 | Update server_test.go, delete tls_test.go | 018f86d | server_test.go (modified), tls_test.go (deleted) |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fresh test clients used different CAs than server**
- **Found during:** Task 2
- **Issue:** `TestWebServerTokenAccess` and `TestWebServerTokenAccessInvalid` called `selfSignedTLSForTest` for a fresh client, which generated a different CA than the server — causing TLS verification failures
- **Fix:** Used `tls.InsecureSkipVerify` for freshClient in token auth tests; these tests validate token auth logic, not TLS trust, so this is the appropriate solution
- **Files modified:** `internal/webserver/server_test.go`
- **Commit:** 018f86d

## Success Criteria Verification

- [x] `go build ./internal/webserver/...` passes
- [x] `go test ./internal/webserver/... -count=1` passes (19 tests green)
- [x] `tls.go` and `tls_test.go` do not exist
- [x] `server.go` uses `lc.GetCertificate` in `Start()`
- [x] `server.go` `BaseURL()` returns FQDN-based URL (`ws.config.FQDN`)
- [x] `server.go` Config has `FQDN string` and `TLSConfig *tls.Config` fields
- [x] No references to `GenerateCA`, `LoadOrCreateCA`, `GenerateLeafCert`, `BuildTLSConfig`, `ExportCACertPath` in any remaining `.go` file

## Self-Check: PASSED
