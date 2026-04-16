---
phase: 75-cli-status-bar
plan: "01"
subsystem: relay
tags: [protocol, websocket, meta-push, viewer-count, hub]
dependency_graph:
  requires: []
  provides: [MsgMeta-frame-type, MetaPayload, MakeMeta, Hub.BroadcastMeta, NotifyViewerCount]
  affects: [internal/relay, internal/webserver]
tech_stack:
  added: []
  patterns: [server-to-client push frame, non-blocking broadcast with CloseSlow]
key_files:
  created: []
  modified:
    - internal/relay/protocol.go
    - internal/relay/hub.go
    - internal/relay/server.go
    - internal/relay/protocol_test.go
    - internal/relay/hub_test.go
    - internal/relay/server_test.go
    - internal/webserver/server.go
decisions:
  - NotifyViewerCount placed as package-level function in relay/server.go so webserver can call relay.NotifyViewerCount(hub) without a separate helper
  - BroadcastMeta mirrors hub.broadcast non-blocking pattern to prevent MsgMeta from ever blocking the PTY drain loop
  - MetaPayload fields use pointer types so omitempty correctly omits nil fields for future partial-update extensibility
metrics:
  duration: ~15 minutes
  completed: 2026-04-14
  tasks_completed: 3
  files_modified: 7
---

# Phase 75 Plan 01: MsgMeta Protocol Push Frame Summary

**One-liner:** Server-to-client MsgMeta push frame (0x20) with JSON MetaPayload carrying live viewer count, wired into both relay server and webserver subscribe/unsubscribe hooks.

## What Was Built

Extended the relay binary protocol with a new server-push frame type. When any WebSocket client subscribes or unsubscribes from a session, all current subscribers immediately receive a `MsgMeta` frame containing the updated viewer count as JSON. This eliminates polling from the CLI status bar (Plan 03) and TUI.

**Protocol additions (internal/relay/protocol.go):**
- `MsgMeta byte = 0x20` constant (reserved range 0x20-0x2F for future server-push types)
- `MetaPayload struct` with `ViewerCount *int` (pointer for omitempty correctness)
- `MakeMeta(p MetaPayload) []byte` function encoding MetaPayload as a MsgMeta frame

**Hub extension (internal/relay/hub.go):**
- `BroadcastMeta(frame []byte)` method using identical non-blocking send + CloseSlow pattern as private `broadcast`

**Relay server (internal/relay/server.go):**
- `NotifyViewerCount(hub *Hub)` exported package-level function (shared by relay and webserver)
- `handleSession` calls `NotifyViewerCount(hub)` after `hub.Subscribe` and in deferred `hub.Unsubscribe` wrapper

**Webserver (internal/webserver/server.go):**
- `handleWSSRelay` calls `relay.NotifyViewerCount(hub)` after `hub.Subscribe` and in deferred `hub.Unsubscribe` wrapper

**Tests (protocol_test.go, hub_test.go):**
- `TestMakeMeta_RoundTrip`: encodes MetaPayload, decodes via ParseFrame, verifies JSON
- `TestMakeMeta_OmitsNilFields`: verifies omitempty omits nil ViewerCount
- `TestBroadcastMeta_NonBlocking`: verifies frame delivery and CloseSlow on full channel

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated server_test.go to handle MsgMeta frames before expected MsgOutput**
- **Found during:** Task 3 (running full test suite after Task 2)
- **Issue:** Five existing server tests called `readFrame` expecting the first frame to be `MsgOutput`, but `NotifyViewerCount` now delivers a `MsgMeta` frame immediately after subscribe — before any PTY output arrives.
- **Fix:** Added `readDataFrame` helper to server_test.go that skips `MsgMeta` frames and returns the first non-meta frame. Updated all five affected test assertions to use `readDataFrame`.
- **Files modified:** internal/relay/server_test.go
- **Commit:** 3757e70

**2. [Rule 1 - Bug] Fixed TestHub_SlowClientDisconnected channel overflow caused by concurrent MsgMeta frames**
- **Found during:** Task 3 (full relay test suite run)
- **Issue:** The test was pre-existing flaky (was already failing at frame 29 before this plan). Adding MsgMeta frames on subscribe made it fail earlier (frame 15-20) because 2 extra MsgMeta frames consumed slots in normalClient's 256-slot channel. The test wrote 300 PTY frames after the flood completed, but normalClient's channel overflowed before the test goroutine could drain it.
- **Fix:** Changed normalClient drain to run in a concurrent goroutine during the flood (not after). This keeps normalClient's channel continuously drained, matching the actual production behavior of an active reader. The test intent (slowClient gets disconnected, normalClient stays alive) is preserved and now correctly exercised.
- **Files modified:** internal/relay/server_test.go
- **Commit:** 3757e70

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 | 6e76134 | feat(75-01): add MsgMeta protocol type, MetaPayload, MakeMeta, and BroadcastMeta |
| Task 2 | 87a67e0 | feat(75-01): wire NotifyViewerCount into relay server and webserver on subscribe/unsubscribe |
| Task 3 | 3757e70 | test(75-01): add MsgMeta round-trip tests and BroadcastMeta non-blocking test |

## Verification Results

```
ok  github.com/scottkw/agenthub/internal/relay  0.729s
go build ./...  [exit 0]
```

All 42 relay package tests pass. Full project build succeeds.

## Known Stubs

None. All viewer count data is live from `hub.SubscriberCount()`.

## Threat Flags

No new security-relevant surface introduced beyond the plan's threat model. MsgMeta frames are server-generated from `hub.SubscriberCount()` — no untrusted input reaches `MakeMeta`. Non-blocking send with CloseSlow (T-75-02) is implemented as required.

## Self-Check: PASSED

- internal/relay/protocol.go: FOUND (MsgMeta, MetaPayload, MakeMeta)
- internal/relay/hub.go: FOUND (BroadcastMeta)
- internal/relay/server.go: FOUND (NotifyViewerCount, calls in handleSession)
- internal/webserver/server.go: FOUND (relay.NotifyViewerCount calls in handleWSSRelay)
- internal/relay/protocol_test.go: FOUND (TestMakeMeta_RoundTrip, TestMakeMeta_OmitsNilFields)
- internal/relay/hub_test.go: FOUND (TestBroadcastMeta_NonBlocking)
- Commits 6e76134, 87a67e0, 3757e70: FOUND in git log
