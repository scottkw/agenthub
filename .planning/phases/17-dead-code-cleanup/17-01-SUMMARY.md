---
phase: 17-dead-code-cleanup
plan: 01
subsystem: api
tags: [go, dead-code, cleanup, webserver, network-interfaces]

# Dependency graph
requires:
  - phase: 15-tailscale-tls-interface-binding
    provides: local.Client direct IP query replacing ListInterfaces usage
  - phase: 16-auth-layer-removal
    provides: cleaned webserver package — no auth routes, no auth middleware
provides:
  - internal/webserver/network.go deleted (NetworkInterface struct, IsTailscaleIP, ListInterfaces gone)
  - internal/webserver/network_test.go deleted
  - app.go GetNetworkInterfaces method removed
  - app_test.go TestGetNetworkInterfaces removed
  - Zero references to NetworkInterface, IsTailscaleIP, ListInterfaces, GetNetworkInterfaces in codebase
affects: [frontend-cleanup, ui-settings-panel]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Delete files together when test file depends on implementation file symbols to avoid compilation break"
    - "Retain package import when other symbols from that package are still used"

key-files:
  created: []
  modified:
    - app.go
    - app_test.go
  deleted:
    - internal/webserver/network.go
    - internal/webserver/network_test.go

key-decisions:
  - "Delete network.go and network_test.go atomically — test file references network.go symbols so they must go together"
  - "Retain webserver package import in app.go — used by TailscaleHealth, CheckHealth, Config, NewWebServer"
  - "No new code introduced — pure deletion, build and tests verified after each step"

patterns-established:
  - "Dead code deletion: confirm no remaining references via grep before deleting"
  - "Verify go build ./... and go test ./... after each deletion step, not just at the end"

requirements-completed: [CLEAN-01, CLEAN-02]

# Metrics
duration: 4min
completed: 2026-03-20
---

# Phase 17 Plan 01: Dead Code Cleanup (network.go) Summary

**Deleted generic VPN interface picker: network.go, network_test.go, GetNetworkInterfaces() from app.go and its test — zero dead symbol references remain**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-03-20T22:08:00Z
- **Completed:** 2026-03-20T22:11:37Z
- **Tasks:** 2
- **Files modified/deleted:** 4

## Accomplishments

- Deleted `internal/webserver/network.go` (NetworkInterface struct, tailscaleCIDR init, IsTailscaleIP, ListInterfaces — all superseded by Phase 15's local.Client direct IP query)
- Deleted `internal/webserver/network_test.go` (TestIsTailscaleIP, TestListInterfaces, TestNetworkInterfaceStruct, TestTailscaleIPDetectedInList — all testing dead code)
- Removed `GetNetworkInterfaces()` method from `app.go` (dead wrapper that called ListInterfaces) and its test from `app_test.go`
- `go build ./...` and `go test ./...` pass cleanly; all auth-absence regression tests (TestLoginRouteNotRegistered, TestTokenRouteNotRegistered, TestSessionAccessWithoutAuth) remain green

## Task Commits

Each task was committed atomically:

1. **Task 1: Delete network.go and network_test.go** - `7662510` (chore)
2. **Task 2: Remove GetNetworkInterfaces from app.go and app_test.go** - `a9e6e8b` (chore)

## Files Created/Modified

- `internal/webserver/network.go` - DELETED (196 lines of dead VPN interface picker code)
- `internal/webserver/network_test.go` - DELETED (121 lines of tests for deleted code)
- `app.go` - GetNetworkInterfaces method (lines 299-307) removed
- `app_test.go` - TestGetNetworkInterfaces test (lines 164-172) removed

## Decisions Made

- Deleted both network files atomically in a single commit — network_test.go references network.go symbols so deleting one without the other would break compilation
- Preserved the webserver import in app.go — it is still needed for webserver.TailscaleHealth, webserver.CheckHealth, webserver.Config, webserver.NewWebServer

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - straight deletion, build passed immediately after both removals.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Backend Go dead code (VPN interface picker) fully removed; CLEAN-01 fulfilled
- Auth-absence regression tests remain green; CLEAN-02 confirmed
- Ready for Plan 02: frontend dead code cleanup (Settings panel interface picker, VPN bind control, SecurityTab)
- No blockers

---
*Phase: 17-dead-code-cleanup*
*Completed: 2026-03-20*
