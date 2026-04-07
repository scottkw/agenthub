---
phase: 50-tailscale-peer-discovery
plan: 02
subsystem: api
tags: [tailscale, daemon, cache, http, peer-discovery]

# Dependency graph
requires:
  - phase: 50-01
    provides: internal/tailnet package with DiscoverAndProbe, Peer struct
provides:
  - GET /tailnet/peers daemon route returning JSON array of Peer objects
  - 30-second tailnetCache with thundering-herd-safe getOrRefresh
  - DaemonClient.ListTailnetPeers() method for Go callers
  - discoverFunc injectable type for testability
affects:
  - 50-tailscale-peer-discovery (phases 52+: GUI and CLI consumers)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Cache with injectable discoverFunc for test isolation without mocks"
    - "Full Mutex (not RWMutex) for getOrRefresh to prevent thundering herd"
    - "Pre-populate cache in tests to avoid needing live external daemon"

key-files:
  created:
    - internal/daemon/tailnet_cache.go
  modified:
    - internal/daemon/api.go
    - internal/daemon/client.go
    - internal/daemon/api_test.go

key-decisions:
  - "Full Mutex for getOrRefresh: prevents multiple goroutines from all calling DiscoverAndProbe simultaneously on cache expiry"
  - "discoverFunc injectable type: enables test isolation by pre-populating cache instead of mocking Tailscale daemon"
  - "Empty array (not error) on Tailscale unavailability: graceful degradation for offline peers"

patterns-established:
  - "tailnetCache.set(testPeers) in tests: pre-populate pattern to avoid external daemon dependency"
  - "context.WithTimeout(r.Context(), 10s) in handler: bounds long-running discovery calls"

requirements-completed: [REM-01]

# Metrics
duration: 2min
completed: 2026-04-07
---

# Phase 50 Plan 02: Tailscale Daemon Route Summary

**GET /tailnet/peers daemon route with 30s cache, thundering-herd-safe getOrRefresh, and DaemonClient.ListTailnetPeers() wired to internal/tailnet package**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-04-07T18:13:46Z
- **Completed:** 2026-04-07T18:15:46Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Created tailnetCache struct with 30-second TTL and full Mutex preventing thundering herd on cache expiry
- Wired GET /tailnet/peers route in daemon API calling tailnet.DiscoverAndProbe with 10s timeout
- Added DaemonClient.ListTailnetPeers() consuming the new route
- Three sub-tests covering cached peers, empty array on unavailability, and client method

## Task Commits

Each task was committed atomically:

1. **Task 1: Create tailnet cache and daemon route** - `ce73164` (feat)
2. **Task 2: Add DaemonClient method and daemon route test** - `2ba62bf` (feat)

## Files Created/Modified
- `internal/daemon/tailnet_cache.go` - tailnetCache with get/set/getOrRefresh and discoverFunc type
- `internal/daemon/api.go` - Added tailnetCache field, initialization, route registration, handleTailnetPeers handler
- `internal/daemon/client.go` - Added ListTailnetPeers() method calling GET /tailnet/peers
- `internal/daemon/api_test.go` - TestHandleTailnetPeers with 3 sub-tests

## Decisions Made
- Used full Mutex (not RWMutex) for getOrRefresh: a single Mutex serializes all callers, preventing multiple concurrent DiscoverAndProbe calls when cache expires
- discoverFunc injectable type allows tests to pre-populate the cache via set() instead of requiring a live Tailscale daemon
- Route returns empty array on error to match the graceful degradation established in plan 01

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Daemon exposes GET /tailnet/peers returning typed peer array via Unix socket
- DaemonClient.ListTailnetPeers() is ready for GUI (phase 52) and CLI (phase 53) consumers
- Cache initialized at NewAPI time — no lazy init needed by callers

---
*Phase: 50-tailscale-peer-discovery*
*Completed: 2026-04-07*
