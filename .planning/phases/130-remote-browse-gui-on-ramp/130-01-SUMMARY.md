---
phase: 130-remote-browse-gui-on-ramp
plan: 01
subsystem: testing
tags: [go, tdd, webserver, tailnet, relay, rpc, security]

# Dependency graph
requires:
  - phase: 128-remote-write-parity
    provides: relay loopback + relay_remote_files_test.go harness (newFixtureRemotePeer, depositCapOnSocket, fixtureCap)
provides:
  - Wave 0 RED tests for GET /api/sessions/meta webserver endpoint contract (RB-01/RB-03)
  - Wave 0 RED tests for FetchAllPeerSessionsMeta no-silent-drop contract (RB-01)
  - Wave 0 RED test for GetRemoteSessionsWithMeta Reachable field (RB-01)
  - Wave 0 RED test for RB-05 relay-surface discover→pick→browse via api.RelayHandler()
affects: [130-03-remote-browse-gui-on-ramp]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Wave 0 TDD: write failing tests before production code to lock behavioral contract"
    - "Key-whitelist RB-03 pattern: decode to map[string]any and assert exact allowed-key set"
    - "newFixtureRemotePeerWithMeta: extend fixture peer with open /api/sessions/meta handler"
    - "tailnet.ShareableSessionMeta reference as compile-error RED trigger for daemon test package"

key-files:
  created:
    - internal/webserver/sessions_meta_test.go
  modified:
    - internal/tailnet/tailnet_test.go
    - app_test.go
    - internal/daemon/relay_remote_files_test.go

key-decisions:
  - "RED by compile failure (undefined types) preferred over RED by assert failure for test packages that mix old and new tests — lets existing passing tests remain in the file"
  - "daemon_test package imports tailnet to reference ShareableSessionMeta, making entire test package compile-fail until plan 130-03 creates the type"
  - "newFixtureRemotePeerWithMeta added as a new local helper rather than modifying newFixtureRemotePeer — keeps existing relay tests' behavior unchanged"
  - "TestFetchAllPeerSessionsMeta_IncludesUnreachablePeers uses a closed httptest.Server to produce connection-refused, validating no-silent-drop without a real network timeout"

patterns-established:
  - "RB-03 key-whitelist test: assert exact {id,name,cli_type,status,url} key set using map[string]any decode"
  - "nil-vs-empty discriminator: nil return = unreachable (Reachable=false), []ShareableSessionMeta{} = reachable empty (Reachable=true+len 0)"
  - "Relay-surface integration test structure: fixture peer with /api/sessions/meta + depositCapOnSocket + api.RelayHandler() browse"

requirements-completed: [RB-01, RB-03, RB-05]

# Metrics
duration: 20min
completed: 2026-06-16
---

# Phase 130 Plan 01: Remote Browse GUI On-Ramp Wave 0 Summary

**Wave 0 RED tests locking the discover→list→pick behavioral contract: four test files encode RB-01 no-silent-drop, RB-03 metadata-only security whitelist, and RB-05 relay-surface discover→browse path through api.RelayHandler()**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-06-16T05:00:00Z
- **Completed:** 2026-06-16T05:08:23Z
- **Tasks:** 3 of 3
- **Files modified:** 4

## Accomplishments

- Created `internal/webserver/sessions_meta_test.go` with four RED tests covering RB-01 (metadata endpoint returns web-enabled sessions) and RB-03 (key-whitelist asserts no cap/grant/content fields leak)
- Extended `internal/tailnet/tailnet_test.go` with four RED tests encoding the no-silent-drop contract (unreachable peer → Reachable=false kept, not dropped; reachable-empty → Reachable=true+len 0)
- Extended `app_test.go` with `TestGetRemoteSessionsWithMeta_ReachableField` asserting the Wails RPC exists and RemotePeerSessions carries a Reachable bool
- Extended `internal/daemon/relay_remote_files_test.go` with `TestRemoteFiles_DiscoverAndBrowse_RelaySurface` (RB-05 release-blocking) tying discover→cap-deposit→relay-browse through api.RelayHandler()

## Task Commits

1. **Task 1: Write RB-01/RB-03 webserver metadata-endpoint tests (RED)** - `f4cb4da` (test)
2. **Task 2: Write RB-01 tailnet + app.go RPC tests (RED)** - `2b4c7ae` (test)
3. **Task 3: Write RB-05 relay-surface discover→pick→browse test (RED)** - `e4baf9b` (test)

## Files Created/Modified

- `/Users/ken/dev/agenthub/internal/webserver/sessions_meta_test.go` — NEW: four TestSessionsMeta_* RED tests for webserver metadata endpoint contract (RB-01/RB-03)
- `/Users/ken/dev/agenthub/internal/tailnet/tailnet_test.go` — EXTENDED: four TestFetchAllPeerSessionsMeta_*/TestFetchPeerSessionsMeta_* RED tests for no-silent-drop contract
- `/Users/ken/dev/agenthub/app_test.go` — EXTENDED: TestGetRemoteSessionsWithMeta_ReachableField RED test for Wails RPC Reachable field
- `/Users/ken/dev/agenthub/internal/daemon/relay_remote_files_test.go` — EXTENDED: TestRemoteFiles_DiscoverAndBrowse_RelaySurface RB-05 release-blocking relay-surface test; newFixtureRemotePeerWithMeta helper

## Decisions Made

- RED by compile failure is cleaner than RED by assert failure when the test package mixes existing passing tests with new failing ones. The daemon_test package import of `tailnet.ShareableSessionMeta` makes the entire package compile-fail, which is intentional and correct for Wave 0.
- `newFixtureRemotePeerWithMeta` created as a local helper rather than modifying `newFixtureRemotePeer` — existing relay tests remain unchanged and their expected behavior is preserved once plan 130-03 fixes the compile error.
- The RB-03 key-whitelist test decodes into `map[string]any` and asserts the exact allowed key set `{id, name, cli_type, status, url}` — any new field added to the response fails the test, locking the security contract.

## Deviations from Plan

None - plan executed exactly as written. All four test files are RED as required. No production code was modified.

## RED State Verification

All four RED states confirmed at plan end:

```
go test ./internal/webserver/... -run TestSessionsMeta -count=1
→ FAIL (404 — route/handler does not exist)

go test ./internal/tailnet/... -run TestFetchAllPeerSessionsMeta -count=1
→ build failed (undefined: ShareableSessionMeta, FetchPeerSessionsMeta)

go test . -run TestGetRemoteSessionsWithMeta -count=1
→ build failed (undefined: GetRemoteSessionsWithMeta, r.Reachable)

go test ./internal/daemon/... -run TestRemoteFiles_DiscoverAndBrowse_RelaySurface -count=1
→ build failed (undefined: tailnet.ShareableSessionMeta)
```

## Known Stubs

None — this plan creates only test files with no production code.

## Threat Flags

None — no new production code or network endpoints introduced. Tests only.

## Issues Encountered

None.

## Next Phase Readiness

- Plan 130-03 (Wave 1 GREEN) will implement `handleSessionsMeta`, `FetchPeerSessionsMeta`, `FetchAllPeerSessionsMeta`, `GetRemoteSessionsWithMeta`, and `ShareableSessionMeta/PeerSessionMetaGroup` types, turning all four RED states green.
- The key-whitelist test (RB-03) will gate any accidental cap/grant field additions to the metadata response.
- The relay-surface test (RB-05) will confirm the discover→browse relay path works end-to-end once the types exist.

## Self-Check

- [x] sessions_meta_test.go exists: `ls internal/webserver/sessions_meta_test.go` ✓
- [x] tailnet_test.go contains TestFetchAllPeerSessionsMeta_IncludesUnreachablePeers ✓
- [x] app_test.go contains TestGetRemoteSessionsWithMeta_ReachableField ✓
- [x] relay_remote_files_test.go contains TestRemoteFiles_DiscoverAndBrowse_RelaySurface ✓
- [x] relay_remote_files_test.go uses api.RelayHandler() in new test ✓
- [x] All three task commits exist: f4cb4da, 2b4c7ae, e4baf9b ✓

---
*Phase: 130-remote-browse-gui-on-ramp*
*Completed: 2026-06-16*
