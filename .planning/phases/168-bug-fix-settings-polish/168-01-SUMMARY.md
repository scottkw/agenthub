---
phase: 168-bug-fix-settings-polish
plan: 01
subsystem: relay
tags: [go, websocket, hub, viewer-count, session-metadata]

# Dependency graph
requires:
  - phase: 155
    provides: PersonKey/Origin identity fields on relay.Subscriber (D-01..D-04)
provides:
  - Hub.RemoteViewerCount() — counts only Origin=="web" subscribers
  - ListSessions SessionInfo.ViewerCount now reflects remote (web-origin) viewers only
affects: [168-02, 168-03, 168-04, 168-05, 168-06, 168-07, hub-card-frontend]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hub method filtering subscribers by Origin field, mirroring SubscriberCount's exact lock discipline (h.mu.Lock/defer h.mu.Unlock)"

key-files:
  created: []
  modified:
    - internal/relay/hub.go
    - internal/relay/hub_test.go
    - internal/daemon/engine.go
    - internal/daemon/types.go
    - internal/daemon/api_test.go

key-decisions:
  - "RemoteViewerCount is a new, separate method — SubscriberCount is left completely unchanged (still consumed by relay/server.go's NotifyViewerCount for the MsgMeta frame the frontend never parses)."
  - "Raw per-connection count (no PersonKey collapse), consistent with D-01/D-02 — matches the plan's explicit prohibition against collapsing web subscribers by PersonKey."

patterns-established:
  - "Hub-card-facing viewer counts must filter on Subscriber.Origin==\"web\"; any future call site that feeds user-visible viewer/connection counts should use RemoteViewerCount(), not SubscriberCount()."

requirements-completed: [FIX-04]

coverage:
  - id: D1
    description: "Hub.RemoteViewerCount() returns the count of Origin==\"web\" subscribers only (0 for empty/local-only hubs, correct count for mixed/web-only hubs)"
    requirement: "FIX-04"
    verification:
      - kind: unit
        ref: "internal/relay/hub_test.go#TestRemoteViewerCount"
        status: pass
    human_judgment: false
  - id: D2
    description: "SessionInfo.ViewerCount (as returned by ListSessions, the Hub card's data source) is populated from RemoteViewerCount, so a never-shared local session reads 0 and a session with N web-origin subscribers reads N"
    requirement: "FIX-04"
    verification:
      - kind: unit
        ref: "internal/daemon/engine_test.go#TestListSessions_ViewerCount"
        status: pass
      - kind: integration
        ref: "internal/daemon/api_test.go#TestAPI_ListSessionsViewerCount"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-07-01
status: complete
---

# Phase 168 Plan 01: Fix phantom Hub viewer count Summary

**Hub session cards now read 0 viewers for never-shared local sessions — `Hub.RemoteViewerCount()` filters on `Origin=="web"`, replacing the raw `SubscriberCount()` (which included the app's own internal TerminalPanel/ChatPanel/status-watcher/preview WebSocket connections) as the source for `SessionInfo.ViewerCount`.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-01T15:09:33-05:00 (first commit)
- **Completed:** 2026-07-01T15:12:05-05:00 (last commit)
- **Tasks:** 2 completed
- **Files modified:** 5

## Accomplishments
- New `(*Hub).RemoteViewerCount() int` method in `internal/relay/hub.go`, mirroring `SubscriberCount`'s exact lock discipline but counting only `Origin=="web"` subscribers.
- `engine.go`'s `ListSessions` — the sole call site feeding the Hub card — now calls `hub.RemoteViewerCount()` instead of `hub.SubscriberCount()`.
- `types.go`'s `SessionInfo.ViewerCount` doc comment reworded to state "remote (web-origin) viewers only".
- Fixed a pre-existing integration test (`TestAPI_ListSessionsViewerCount`) that encoded the old (pre-fix) semantics by subscribing with no `Origin` — updated to `Origin: "web"` so it still exercises a real remote-viewer scenario under the new filtered count.

## Task Commits

Each task followed the TDD RED → GREEN cycle:

1. **Task 1: Add Hub.RemoteViewerCount() filtered on Origin=="web"**
   - `9ae4dfe` (test) — RED: `TestRemoteViewerCount` added, fails to compile (method doesn't exist yet)
   - `02daff6` (feat) — GREEN: `RemoteViewerCount()` implemented, test passes
2. **Task 2: Repoint engine.go ViewerCount at RemoteViewerCount + update types.go comment**
   - `c0b60b3` (test) — RED: `TestListSessions_ViewerCount` added, fails behaviorally (2 local + 2 web → got 4, want 2)
   - `e1786d3` (feat) — GREEN: call-site swap + comment update + pre-existing `TestAPI_ListSessionsViewerCount` fix, all tests pass

**Plan metadata:** (this commit, docs: complete plan)

_Note: Both tasks used TDD (RED commit → GREEN commit); no separate REFACTOR commits were needed._

## Files Created/Modified
- `internal/relay/hub.go` - Added `RemoteViewerCount()` method (Origin=="web" filter), placed next to `SubscriberCount`
- `internal/relay/hub_test.go` - Added `TestRemoteViewerCount` (empty/local-only/mixed/web-only/unsubscribe cases)
- `internal/daemon/engine.go` - `ListSessions` viewerCount call-site: `hub.SubscriberCount()` → `hub.RemoteViewerCount()`
- `internal/daemon/types.go` - `SessionInfo.ViewerCount` comment reworded to "remote (web-origin) viewers only"
- `internal/daemon/engine_test.go` - Added `TestListSessions_ViewerCount` + `findSessionByID` test helper
- `internal/daemon/api_test.go` - Fixed `TestAPI_ListSessionsViewerCount` to subscribe with `Origin: "web"` (was subscribing with no Origin, which is no longer countable)

## Decisions Made
- Kept `RemoteViewerCount` as a wholly separate method rather than modifying `SubscriberCount` in place, per the plan's explicit prohibition — `SubscriberCount` is still consumed unchanged by `relay/server.go`'s `NotifyViewerCount`.
- Used a raw per-connection count (no `PersonKey` collapse) for `RemoteViewerCount`, matching D-01/D-02 and the plan's explicit prohibition against collapsing web subscribers.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed pre-existing test `TestAPI_ListSessionsViewerCount` that encoded the old (pre-fix) viewer-count semantics**
- **Found during:** Task 2 (wave-merge verification gate: `go test -race -short ./internal/relay/... ./internal/daemon/...`)
- **Issue:** This integration test subscribed a client with no `Origin` field set (empty string) and asserted `ViewerCount` rose by 1. Under the new `RemoteViewerCount` filter (`Origin=="web"` only), an empty-Origin subscriber no longer counts, so the test timed out waiting for a count that would never arrive — exactly the kind of "test encodes the same wrong assumption" pitfall this fix exists to correct (the same class of gap the FIX-04 requirement targets).
- **Fix:** Set `Origin: "web"` on the test's subscriber, so it represents a genuine remote/shared viewer — the scenario the test was actually meant to exercise. Updated the doc comment to note the Phase 168 semantics change.
- **Files modified:** internal/daemon/api_test.go
- **Verification:** `go test ./internal/daemon/... -run TestAPI_ListSessionsViewerCount -v` passes; full `go test -race -short ./internal/relay/... ./internal/daemon/...` passes clean afterward.
- **Committed in:** e1786d3 (part of Task 2 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix, Rule 1)
**Impact on plan:** Directly in scope — same call site (`ListSessions` → `SessionInfo.ViewerCount`) the plan's Task 2 modifies. No scope creep; without this fix the wave-merge race gate would fail.

## Issues Encountered
None beyond the deviation documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `Hub.RemoteViewerCount()` is available for any future call site that needs a real remote-viewer count (e.g. the multi-viewer kick/disconnect UI work in a later 168 plan, per #117).
- `SubscriberCount()` and `NotifyViewerCount` are fully untouched — no risk to existing MsgMeta frame consumers.
- No blockers for subsequent 168 plans.

---
*Phase: 168-bug-fix-settings-polish*
*Completed: 2026-07-01*
