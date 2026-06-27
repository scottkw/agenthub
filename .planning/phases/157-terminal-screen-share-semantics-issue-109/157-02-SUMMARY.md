---
phase: 157-terminal-screen-share-semantics-issue-109
plan: "02"
subsystem: relay
tags: [go, relay, webserver, websocket, resize, pty, view-02, view-03]

requires:
  - phase: 157-01
    provides: "Hub.ResizeClient host-authority arbiter + Hub.Rows() accessor used by join-push"

provides:
  - "Relay join-push: MsgResize 0x02 frame before scrollback on relay path (VIEW-03)"
  - "Web join-push: relay.MakeResizeFrame before scrollback on webserver path (VIEW-03)"
  - "Web read-pump resize drop: MsgResize2 case no-ops at call site (VIEW-02 / T-157-02)"
  - "Three integration tests: TestRelayJoin_PushesResizeBeforeScrollback, TestWebJoin_PushesResizeBeforeScrollback, TestWebReadPump_DropsGuestResize"

affects:
  - 157-03 (viewer CSS scale — guest now receives authoritative grid on join)
  - 157-04 (integration testing — two new test files in traceability map)
  - 157-05 (TESTING.md traceability update)

tech-stack:
  added: []
  patterns:
    - "Join-push-before-scrollback: direct conn.Write before snapshot write; never routed through sub.Msgs to preserve ordering"
    - "Defense-in-depth resize drop: web read-pump MsgResize2 case body is a no-op; hub origin gate is the primary enforcement, call-site drop avoids needless lock per guest resize"
    - "readDataFrame skip set extended to include MsgResize: pre-157 relay tests that only care about PTY output skip the join-push resize as housekeeping"

key-files:
  created: []
  modified:
    - internal/relay/server.go
    - internal/webserver/server.go
    - internal/relay/server_test.go
    - internal/webserver/server_test.go

key-decisions:
  - "Join-push uses direct conn.Write (not sub.Msgs) to guarantee resize arrives BEFORE the scrollback snapshot write — ordering is load-bearing for VIEW-03"
  - "Guard condition is c>0 && r>0 on hub.Cols()/hub.Rows(); the fallback values (220/50) satisfy this, so the push always fires — all joining clients get the initial grid"
  - "Web read-pump MsgResize2 case body is a comment-only no-op; the case label is retained so the frame is consumed and not misrouted"
  - "readDataFrame updated to skip MsgResize (same treatment as MsgMeta/MsgPresence housekeeping); TestRelayJoin uses a local readOrdered closure that skips only meta/presence so the ordering assertion is preserved"

requirements-completed: [VIEW-02, VIEW-03]

coverage:
  - id: D1
    description: "Relay-path join-push: guest first non-meta frame is MsgResize 0x02 with hub's authoritative grid before scrollback (VIEW-03)"
    requirement: VIEW-03
    verification:
      - kind: integration
        ref: "internal/relay/server_test.go#TestRelayJoin_PushesResizeBeforeScrollback"
        status: pass
    human_judgment: false
  - id: D2
    description: "Web-path join-push: web guest first non-meta frame is relay.MakeResizeFrame 0x02 before scrollback (VIEW-03)"
    requirement: VIEW-03
    verification:
      - kind: integration
        ref: "internal/webserver/server_test.go#TestWebJoin_PushesResizeBeforeScrollback"
        status: pass
    human_judgment: false
  - id: D3
    description: "Web read-pump MsgResize2 drop: web guest sending 0x11 leaves hub.Cols()/hub.Rows() unchanged (VIEW-02 / T-157-02 defense-in-depth)"
    requirement: VIEW-02
    verification:
      - kind: integration
        ref: "internal/webserver/server_test.go#TestWebReadPump_DropsGuestResize"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-06-27
status: complete
---

# Phase 157 Plan 02: Server-Side Join-Push and Web-Origin Resize Drop Summary

**VIEW-03 join-push (MsgResize 0x02 before scrollback) wired on both relay and web paths; VIEW-02 defense-in-depth call-site drop added to the web read-pump's MsgResize2 case; three integration tests lock ordering and drop behavior**

## Performance

- **Duration:** 8 min
- **Started:** 2026-06-27T13:03:06Z
- **Completed:** 2026-06-27T13:11:12Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- Inserted VIEW-03 grid push in `relay/server.go` immediately before the scrollback snapshot write: `if c, r := hub.Cols(), hub.Rows(); c > 0 && r > 0` → direct `conn.Write(MakeResizeFrame)`. Ordering is load-bearing — resize before scrollback so replayed bytes land in a correctly-sized terminal grid.
- Applied the same insertion to `webserver/server.go` (qualified as `relay.MakeResizeFrame`), replacing the stale MC-06 comment on the local-host resize call with "host-authority arbiter (VIEW-01/VIEW-02)".
- Dropped `hub.ResizeClient` in the webserver read-pump's `MsgResize2` case (VIEW-02 / T-157-02 defense-in-depth): the case label is retained to consume the frame; the body is a prose comment explaining that a web guest never drives the PTY grid.
- Added `TestRelayJoin_PushesResizeBeforeScrollback` proving the 0x02 frame precedes scrollback on the relay path with correct 120×30 dims.
- Added `TestWebJoin_PushesResizeBeforeScrollback` proving the same on the wss:// web path with 80×24 dims.
- Added `TestWebReadPump_DropsGuestResize` proving a web client's MsgResize2 (0x11) leaves hub.Cols()/hub.Rows() at 100×40 unchanged.
- Extended `readDataFrame` (relay tests) and `TestWebServerWSS` skip-filter (webserver tests) to also skip MsgResize — pre-157 tests only care about PTY output and the join-push resize is housekeeping. The ordering test uses a `readOrdered` closure that skips only meta/presence so the MsgResize assertion is preserved.

## Task Commits

1. **Task 1: Relay-path join-time grid push before scrollback** - `134bd895` (feat)
2. **Task 2: Web-path join push + drop web-origin resize at the read-pump** - `704afac8` (feat)
3. **Task 3: Server-side ordering + web-origin-drop tests** - `b81599e2` (test)

## Files Created/Modified

- `internal/relay/server.go` — VIEW-03 join-push block before scrollback; MC-06 comment updated to host-authority
- `internal/webserver/server.go` — VIEW-03 join-push block before scrollback; MsgResize2 case body replaced with no-op prose comment
- `internal/relay/server_test.go` — TestRelayJoin_PushesResizeBeforeScrollback added; readDataFrame extended to skip MsgResize
- `internal/webserver/server_test.go` — TestWebJoin_PushesResizeBeforeScrollback + TestWebReadPump_DropsGuestResize added; helper functions newWebTestHub, dialWebWS, readWebFrame added; TestWebServerWSS skip-filter updated

## Decisions Made

- Join-push uses direct `conn.Write` (not `sub.Msgs`) so the resize frame is ordered atomically before the snapshot bytes that follow on the same connection. Routing through `sub.Msgs` would interleave it after any queued live output from the hub's broadcast goroutine.
- Guard condition `c > 0 && r > 0` always fires against the hub's fallback dims (220×50), so every joining client receives an initial grid even before the first host resize event.
- Web read-pump MsgResize2 case retains the `case` label so the frame is consumed and the switch does not fall through to an unexpected branch. The body is prose-only (no call to ResizeClient).
- `readDataFrame` extended to skip MsgResize: the helper's documented purpose is "first PTY-data frame for pre-152 tests" — the join-push resize is housekeeping like MsgMeta/MsgPresence. The new ordering test avoids `readDataFrame` via a local closure.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Extended readDataFrame and TestWebServerWSS to skip MsgResize**
- **Found during:** Task 3 (test execution)
- **Issue:** The new join-push causes all pre-157 tests that use `readDataFrame` (relay) or the explicit skip loop (webserver) to receive MsgResize as the "first data frame" instead of MsgOutput. Five relay tests and one webserver test failed with type 0x02 where 0x01 was expected.
- **Fix:** Added `MsgResize` to the skip set in `readDataFrame`; added `relay.MsgResize` to the skip condition in `TestWebServerWSS`. `TestRelayJoin_PushesResizeBeforeScrollback` uses a local `readOrdered` closure (skips only meta/presence) to preserve the ordering assertion.
- **Files modified:** `internal/relay/server_test.go`, `internal/webserver/server_test.go`
- **Verification:** `go test -race -short ./internal/relay/ ./internal/webserver/` all green
- **Committed in:** `b81599e2` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — pre-plan test compatibility with new join-push behavior)
**Impact on plan:** Required fix — the join-push is correct behavior; the existing tests needed updating. No scope creep.

## Issues Encountered

None beyond the test compatibility fix above.

## Known Stubs

None — all new code paths are fully implemented and connected.

## Threat Flags

No new security surface beyond the plan's threat model.

- **T-157-02 (Tampering/DoS):** Web read-pump MsgResize2 case drops the call. Asserted by `TestWebReadPump_DropsGuestResize`.
- **T-157-05 (Information disclosure):** Join-push carries only cols/rows (already implied by subsequent output stream). Accepted.

## Next Phase Readiness

Plan 03 (viewer honor / CSS scale) can now rely on:
- Guest always receives authoritative grid on join (fallback 220×50 or real host dims)
- Web guest resize frames are dropped at the call site, so the hub's grid is host-only
- Both relay and web surfaces are symmetric for VIEW-03

New test files for Plan 05 TESTING.md traceability map:
- `internal/relay/server_test.go` — `TestRelayJoin_PushesResizeBeforeScrollback`
- `internal/webserver/server_test.go` — `TestWebJoin_PushesResizeBeforeScrollback`, `TestWebReadPump_DropsGuestResize`

---
*Phase: 157-terminal-screen-share-semantics-issue-109*
*Completed: 2026-06-27*

## Self-Check: PASSED

- `internal/relay/server.go` — FOUND (modified)
- `internal/webserver/server.go` — FOUND (modified)
- `internal/relay/server_test.go` — FOUND (modified)
- `internal/webserver/server_test.go` — FOUND (modified)
- Commit `134bd895` — FOUND (feat: Task 1)
- Commit `704afac8` — FOUND (feat: Task 2)
- Commit `b81599e2` — FOUND (test: Task 3)
- All 3 new tests pass under `-race`
- Full relay + webserver packages pass under `-race -short`
