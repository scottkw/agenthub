---
phase: 04-web-serving-tls-auth
plan: 01
subsystem: infra
tags: [tls, x509, ecdsa, ca-cert, network, tailscale, go-stdlib]

requires:
  - phase: 03-wails-desktop-ui
    provides: App struct and project structure for internal/webserver package placement

provides:
  - ECDSA P-256 CA cert generation, disk persistence, and reload via LoadOrCreateCA
  - In-memory leaf cert signing with SAN IPAddresses via GenerateLeafCert
  - tls.Config builder via BuildTLSConfig with TLS 1.2 minimum
  - Network interface enumeration via ListInterfaces (IPv4, non-loopback, non-link-local)
  - Tailscale CGNAT range detection via IsTailscaleIP (100.64.0.0/10)
  - NetworkInterface struct with Name, IP, IsTailscale fields

affects:
  - 04-02-PLAN (WebServer struct uses tls.go and network.go)
  - 04-03-PLAN (auth and token integration)

tech-stack:
  added: []
  patterns:
    - "CA-signed leaf pattern: GenerateCA persists once; GenerateLeafCert runs in-memory each launch"
    - "SAN IPAddresses required: modern browsers reject certs without Subject Alternative Name"
    - "IPv4-only interface enumeration: keeps dropdown clean; VPN interfaces are always IPv4"
    - "make([]NetworkInterface, 0) for empty default: callers can range safely without nil check"

key-files:
  created:
    - internal/webserver/tls.go
    - internal/webserver/tls_test.go
    - internal/webserver/network.go
    - internal/webserver/network_test.go
  modified: []

key-decisions:
  - "Leaf key never written to disk — CA key on disk is already a risk; leaf generated in-memory each launch"
  - "IPv4-only in ListInterfaces — VPN/Tailscale interfaces are always IPv4; keeps interface dropdown clean"
  - "clock skew buffer: NotBefore = time.Now().Add(-time.Minute) prevents immediate rejection on machines with slight clock drift"

patterns-established:
  - "TLS: CA cert persisted to ~/.config/agenthub/ca.crt + ca.key; leaf cert generated in-memory each launch"
  - "Network: Package-level tailscaleCIDR initialized in init() for CGNAT detection"

requirements-completed: [WEB-02, NET-01, NET-02, NET-03]

duration: 15min
completed: 2026-03-18
---

# Phase 4 Plan 01: TLS CA Infrastructure and Network Interface Enumeration Summary

**ECDSA P-256 CA-signed TLS cert generation with disk persistence and Tailscale CGNAT detection via stdlib only**

## Performance

- **Duration:** 15 min
- **Started:** 2026-03-18T17:47:00Z
- **Completed:** 2026-03-18T18:02:12Z
- **Tasks:** 2
- **Files modified:** 4 created

## Accomplishments
- CA cert generation with IsCA=true, BasicConstraintsValid=true, KeyUsageCertSign persisted to disk once
- Leaf cert signing with SAN IPAddresses (mandatory for modern browser trust), ExtKeyUsageServerAuth, in-memory only
- LoadOrCreateCA idempotent: first call writes ca.crt/ca.key, subsequent calls load identically
- Network interface enumeration excluding loopback, link-local, and IPv6, with Tailscale CGNAT auto-detection
- 22 tests passing across all webserver package files

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: TLS failing tests** - `c5f7852` (test)
2. **Task 1 GREEN: tls.go implementation** - `f757ce5` (feat)
3. **Task 2 RED: Network failing tests** - `df0398b` (test)
4. **Task 2 GREEN: network.go implementation** - `151bf87` (feat)

_Note: TDD tasks have separate test (RED) and feat (GREEN) commits_

## Files Created/Modified
- `internal/webserver/tls.go` - GenerateCA, LoadOrCreateCA, GenerateLeafCert, BuildTLSConfig, ExportCACertPath
- `internal/webserver/tls_test.go` - 5 TLS test cases (CA properties, leaf SAN, disk persistence, tls.Config)
- `internal/webserver/network.go` - NetworkInterface struct, IsTailscaleIP, ListInterfaces with IPv4 filter
- `internal/webserver/network_test.go` - 4 network test cases (Tailscale classification, interface enumeration, struct fields)

## Decisions Made
- Leaf key never written to disk — CA key on disk is already a risk; leaf generated in-memory each launch
- IPv4-only in ListInterfaces — VPN/Tailscale interfaces are always IPv4; keeps interface dropdown clean
- clock skew buffer: NotBefore = time.Now().Add(-time.Minute) prevents immediate rejection on machines with slight clock drift

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] tokens_test.go referenced unimplemented GenerateToken/NewTokenStore**
- **Found during:** Task 2 (network test run — all package tests run together)
- **Issue:** tokens_test.go existed from prior plan work but tokens.go was already present (not missing); build succeeded after verifying tokens.go exists
- **Fix:** No fix needed — tokens.go already existed; initial diagnostic was incorrect
- **Files modified:** None
- **Verification:** Full test suite passes: 22 tests passing
- **Committed in:** 151bf87 (Task 2 commit)

---

**Total deviations:** 0 auto-fixes required — tokens.go pre-existed from prior plan
**Impact on plan:** None — plan executed as written

## Issues Encountered
- tokens_test.go appeared to be missing its implementation (tokens.go) during Task 2 test run — turned out tokens.go already existed from a previous commit (086438b). Build succeeded once implementation was confirmed present.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- TLS infrastructure ready for Plan 02/03 WebServer struct to use BuildTLSConfig and GenerateLeafCert
- Network interface enumeration ready for Plan 02/03 bind-IP selection UI
- CA cert path (ExportCACertPath) ready for Plan 02/03 trust store installation guidance (WEB-03)
- No blockers for next wave plans

## Self-Check: PASSED

- FOUND: internal/webserver/tls.go
- FOUND: internal/webserver/network.go
- FOUND: internal/webserver/tls_test.go
- FOUND: internal/webserver/network_test.go
- FOUND: commit c5f7852 (test RED - TLS)
- FOUND: commit f757ce5 (feat GREEN - TLS)
- FOUND: commit df0398b (test RED - network)
- FOUND: commit 151bf87 (feat GREEN - network)
