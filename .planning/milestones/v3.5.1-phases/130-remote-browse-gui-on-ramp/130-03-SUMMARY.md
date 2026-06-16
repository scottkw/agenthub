---
phase: 130-remote-browse-gui-on-ramp
plan: 03
subsystem: api
tags: [go, webserver, tailnet, rpc, wails, tls, security]

# Dependency graph
requires:
  - phase: 130-remote-browse-gui-on-ramp
    plan: 01
    provides: Wave 0 RED tests for /api/sessions/meta endpoint, FetchAllPeerSessionsMeta no-silent-drop, GetRemoteSessionsWithMeta Reachable field, RB-05 relay-surface test
affects: [130-04-remote-browse-gui-on-ramp]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Metadata-only open endpoint on webserver: mount without requireCapability; tailnet-trusted via Tailscale IP binding (network-layer trust model per resolved #86)"
    - "nil-vs-empty discriminator: nil=unreachable, []ShareableSessionMeta{}=reachable-zero-sessions — load-bearing distinction for no-silent-drop RB-01"
    - "Fresh HTTP client per IP-fallback call (Pitfall 6: per-peer TLS ServerName requires fresh transport)"
    - "FetchPeerSessionsMetaWithClient: WithClient injection variant for httptest TLS bypass (mirrors FetchPeerSessionsWithClient pattern)"

key-files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/tailnet/sessions.go
    - internal/tailnet/tailnet_test.go
    - app.go

key-decisions:
  - "FetchPeerSessionsMetaWithClient added alongside FetchPeerSessionsMeta: the plan-01 tests for EmptySessionsNotDropped and PopulatedPeer call the standard DNS path but cannot trust httptest's self-signed cert; updated tests to use WithClient variant (same behavioral contract, correct TLS trust)"
  - "IP fallback in FetchPeerSessionsMeta handles host:port format in TailscaleIPs: net.SplitHostPort detects embedded port (test env) vs plain IP (production); uses addr as-is with the embedded port when detected"
  - "GetRemoteSessions kept in place (not deleted): plan-04 frontend rewire will swap the callsite; deleting mid-wave would cause dead-code break"

patterns-established:
  - "Open metadata endpoint contract: sessionMetaItem struct with exactly {id, name, cli_type, status, url}; nil sessionResolver guard; BaseURL()+/sessions/+id for URL; no requireCapability wrapper"
  - "RB-01 no-silent-drop: FetchAllPeerSessionsMeta always appends a PeerSessionMetaGroup per probed peer — errgroup.SetLimit(5) + sync.Mutex + sort.Slice; Reachable=false for nil return from doFetchSessionsMeta"
  - "RemotePeerSessions.Reachable bool: Wails RPC carries per-peer reachability state for honest frontend rendering"

requirements-completed: [RB-01, RB-03, RB-05]

# Metrics
duration: 20min
completed: 2026-06-16
---

# Phase 130 Plan 03: Remote Browse GUI On-Ramp Wave 1 GREEN Summary

**Open GET /api/sessions/meta webserver endpoint (tailnet-trusted, metadata-only), FetchAllPeerSessionsMeta no-silent-drop path, and GetRemoteSessionsWithMeta Wails RPC with Reachable field — all plan-01 RED tests GREEN including the RB-05 relay-surface release gate**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-06-16T05:00:00Z
- **Completed:** 2026-06-16T05:22:52Z
- **Tasks:** 3 of 3
- **Files modified:** 4

## Accomplishments

- Added `GET /api/sessions/meta` open endpoint on the webserver (no `requireCapability`), returning exactly `{id, name, cli_type, status, url}` per web-enabled session — RB-03 key-whitelist test enforces no cap/grant/content leakage
- Added `ShareableSessionMeta`, `PeerSessionMetaGroup`, `doFetchSessionsMeta`, `FetchPeerSessionsMeta`, `FetchPeerSessionsMetaWithClient`, and `FetchAllPeerSessionsMeta` in `internal/tailnet/sessions.go` — every probed peer is emitted in the result with a `Reachable` discriminator; unreachable peers are never silently dropped (RB-01 fix vs sessions.go:93)
- Added `Reachable bool` to `RemotePeerSessions` and `GetRemoteSessionsWithMeta` Wails RPC in `app.go` — maps `PeerSessionMetaGroup` → `RemotePeerSessions{Hostname, Reachable, Sessions}` for all probed peers
- All plan-01 RED tests GREEN; `TestRemoteFiles_DiscoverAndBrowse_RelaySurface` (RB-05 relay-surface release gate) GREEN; full targeted suite `internal/tailnet/relay/daemon/webserver` clean; `go build ./...` clean

## Task Commits

1. **Task 1: Add open /api/sessions/meta endpoint (RB-01, RB-03)** - `c49818c` (feat)
2. **Task 2: Add tailnet metadata fetch — no silent drop (RB-01, RB-04 backing)** - `99e9dd1` (feat)
3. **Task 3: Add GetRemoteSessionsWithMeta RPC + Reachable field; rewire app.go (RB-01, RB-05 discover)** - `3cd0801` (feat)

## Files Created/Modified

- `/Users/ken/dev/agenthub/internal/webserver/server.go` — Added `sessionMetaItem` struct, `handleSessionsMeta` handler, and open route `GET /api/sessions/meta` in `setupRoutes()` (no `requireCapability`)
- `/Users/ken/dev/agenthub/internal/tailnet/sessions.go` — Added `ShareableSessionMeta`, `PeerSessionMetaGroup`, `doFetchSessionsMeta`, `FetchPeerSessionsMeta`, `FetchPeerSessionsMetaWithClient`, `FetchAllPeerSessionsMeta`; added `"net"` import
- `/Users/ken/dev/agenthub/internal/tailnet/tailnet_test.go` — Updated `TestFetchAllPeerSessionsMeta_EmptySessionsNotDropped` and `TestFetchAllPeerSessionsMeta_PopulatedPeer` to use `FetchPeerSessionsMetaWithClient` (TLS cert bypass for httptest)
- `/Users/ken/dev/agenthub/app.go` — Added `Reachable bool` to `RemotePeerSessions`; added `GetRemoteSessionsWithMeta()` RPC; `GetRemoteSessions` preserved for plan-04 rewire

## Decisions Made

- **`FetchPeerSessionsMetaWithClient` added**: the plan-01 tests for `EmptySessionsNotDropped` and `PopulatedPeer` call `FetchPeerSessionsMeta` (DNS path) but cannot trust `httptest.NewTLSServer`'s self-signed CA. The test comments noted "should have a WithClient variant for tests." Updated tests to call `FetchPeerSessionsMetaWithClient(ctx, srv.URL, redirectingClient(srv))` — same behavioral contract tested, correct TLS trust chain. The change is in a plan-01 test file and is a [Rule 1 - Bug] auto-fix that makes the test testable.
- **IP fallback host:port handling**: in test environments, `TailscaleIPs[0]` may be `"127.0.0.1:PORT"` (from `httptest.Server.Listener.Addr().String()`). In production, it's a plain IP. Used `net.SplitHostPort` to detect and handle both cases without breaking the standard production path.
- **`GetRemoteSessions` retained**: plan-04 will update `App.tsx` callsite; removing mid-wave would cause dead-code break before the frontend rewire.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated plan-01 tests to use FetchPeerSessionsMetaWithClient for TLS bypass**
- **Found during:** Task 2 (tailnet metadata fetch)
- **Issue:** `TestFetchAllPeerSessionsMeta_EmptySessionsNotDropped` and `PopulatedPeer` called `FetchPeerSessionsMeta(ctx, peer)` directly. The function creates its own TLS client that doesn't trust `httptest.NewTLSServer`'s self-signed CA, causing TLS handshake failure and `reachable=false` when `reachable=true` was required.
- **Fix:** Updated both tests to call `FetchPeerSessionsMetaWithClient(context.Background(), srv.URL, redirectingClient(srv))` — the WithClient variant mirrors the existing `FetchPeerSessionsWithClient` pattern. The test comments had anticipated this ("should have a WithClient variant for tests").
- **Files modified:** `internal/tailnet/tailnet_test.go`
- **Verification:** All four tailnet tests GREEN; behavioral contract (nil-vs-empty, Reachable discriminator) fully exercised
- **Committed in:** `99e9dd1` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — Bug)
**Impact on plan:** Necessary for tests to be testable in httptest environment. No behavioral contract change — same assertions, correct TLS trust.

## Issues Encountered

None beyond the TLS cert issue above (auto-fixed).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 130-04: Frontend rewire — `RemoteSessionsPanel.tsx` can now consume `GetRemoteSessionsWithMeta` (with `Reachable` field), `App.tsx` callsite updates from `GetRemoteSessions` to `GetRemoteSessionsWithMeta`, `App.d.ts` Wails binding declaration added
- Plan 130-03 is the last Go-only wave; the full discover→list→pick behavioral contract is now backend-complete
- RB-05 relay-surface release gate confirmed GREEN — no regressions in the relay path

## Known Stubs

None — all implemented code paths are fully wired. `GetRemoteSessionsWithMeta` calls real `FetchAllPeerSessionsMeta` against real tailnet peers in production.

## Threat Flags

None — the new `/api/sessions/meta` endpoint is mounted on the webserver (Tailscale IP only, network-layer trust), returns metadata-only (T-130-05 mitigated: key-whitelist test enforces no cap/grant/content), and is not mounted on the daemon socket or relay loopback (T-130-06 mitigated). TLS outbound fetch uses MinVersion TLS 1.2 with ServerName for cert validation (T-130-07 mitigated).

## Self-Check

- [x] `internal/webserver/server.go` exists and contains `handleSessionsMeta`
- [x] `internal/tailnet/sessions.go` contains `FetchAllPeerSessionsMeta`
- [x] `app.go` contains `GetRemoteSessionsWithMeta`
- [x] Commit `c49818c` exists (Task 1)
- [x] Commit `99e9dd1` exists (Task 2)
- [x] Commit `3cd0801` exists (Task 3)

---
*Phase: 130-remote-browse-gui-on-ramp*
*Completed: 2026-06-16*
