---
phase: 157-terminal-screen-share-semantics-issue-109
plan: "01"
subsystem: relay
tags: [go, relay, pty, resize, hub, websocket]

requires:
  - phase: 156-install-links-and-distribution
    provides: no dependency — Phase 157 is orthogonal to install/chat

provides:
  - "Hub.ResizeClient: host-authority arbiter (local-origin min-among-local + D-01 freeze + VIEW-02 web-reject + VIEW-01 broadcast)"
  - "Hub.broadcastResize(cols, rows uint16): non-blocking MsgResize 0x02 fan-out after every authoritative grid change"
  - "Hub.Rows() int: PTY row count accessor with 50 fallback (mirrors Cols() 220 fallback)"

affects:
  - 157-02 (join-push, VIEW-03 — depends on Rows() and the frozen-grid state)
  - 157-03 (viewer honor / CSS scale — reads the broadcast grid)
  - 157-04 (testing — depends on the arbiter contract)

tech-stack:
  added: []
  patterns:
    - "Host-authority origin gate: non-local Origin returns immediately before acquiring hub.mu — single enforcement point for T-157-01"
    - "unlock-before-IO: hub.mu released before broadcastResize and resizeFn (mirrors pre-existing ResizeClient discipline at hub.go:254)"
    - "D-01 freeze: minCols/minRows stay zero when no local subscriber has positive dimensions; ptyCols/ptyRows unchanged"
    - "D-02 min-among-local: iterate only Origin=local subscribers with positive Cols/Rows; take per-axis minimum"

key-files:
  created: []
  modified:
    - internal/relay/hub.go
    - internal/relay/hub_test.go

key-decisions:
  - "VIEW-02 origin gate is the FIRST check in ResizeClient (before lock) — non-local returns immediately, size never recorded anywhere"
  - "broadcastResize self-acquires mu and must only be called after hub.mu.Unlock() (T-157-04: prevents self-deadlock)"
  - "Hub.Rows() fallback is 50 (engine.go emuRows), mirrors Hub.Cols() fallback of 220 (wide enough for scrollback VT extraction)"
  - "MC-06 max-wins tests fully replaced — no test references max-wins or shrink-on-unsubscribe behavior"

requirements-completed: [VIEW-01, VIEW-02]

coverage:
  - id: D1
    description: "ResizeClient host-authority arbiter: local-origin min-among-local, D-01 freeze, VIEW-02 web-origin reject"
    requirement: VIEW-02
    verification:
      - kind: unit
        ref: "internal/relay/hub_test.go#TestHub_ResizeHostAuthority_MinAmongLocal"
        status: pass
      - kind: unit
        ref: "internal/relay/hub_test.go#TestHub_ResizeClientNoOpWhenDimensionsUnchanged"
        status: pass
      - kind: unit
        ref: "internal/relay/hub_test.go#TestHub_ResizeFreezeLastHostSize"
        status: pass
      - kind: unit
        ref: "internal/relay/hub_test.go#TestHub_ResizeIgnoresWebOrigin"
        status: pass
    human_judgment: false
  - id: D2
    description: "broadcastResize non-blocking MsgResize 0x02 fan-out to all subscribers on host resize (VIEW-01)"
    requirement: VIEW-01
    verification:
      - kind: unit
        ref: "internal/relay/hub_test.go#TestHub_ResizeBroadcastsToSubscribers"
        status: pass
    human_judgment: false
  - id: D3
    description: "Hub.Rows() int accessor with 50 fallback (needed by VIEW-03 join-push in Plan 02)"
    requirement: VIEW-01
    verification:
      - kind: unit
        ref: "internal/relay/hub_test.go#TestHub_RowsFallback"
        status: pass
    human_judgment: false

duration: 3min
completed: 2026-06-27
status: complete
---

# Phase 157 Plan 01: Hub Host-Authority PTY-Size Arbiter Summary

**Host-authority `ResizeClient` replacing MC-06 max-wins: local-origin min-among-local grid, D-01 freeze, VIEW-02 web-reject, VIEW-01 broadcastResize fan-out, and Hub.Rows() accessor**

## Performance

- **Duration:** 3 min
- **Started:** 2026-06-27T12:52:35Z
- **Completed:** 2026-06-27T12:55:35Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Rewrote `Hub.ResizeClient` with a four-part host-authority policy: VIEW-02 origin gate (non-local returns immediately), D-02 min-among-local (PTY grid = smallest local subscriber), D-01 freeze (last host grid persists when no local subscriber), VIEW-01 broadcast (MsgResize 0x02 fan-out after every grid change)
- Added `Hub.broadcastResize(cols, rows uint16)` — non-blocking fan-out helper mirroring BroadcastMeta; self-acquires mu, called only after hub.mu.Unlock() to prevent T-157-04 deadlock
- Added `Hub.Rows() int` with fallback 50 (mirrors Cols() 220), required by VIEW-03 join-push in Plan 02
- Replaced three MC-06 max-wins hub tests with six host-authority tests covering min-among-local, no-op, D-01 freeze, web-origin ignore, broadcast verification, and Rows() fallback

## Task Commits

1. **Task 1: Rewrite ResizeClient + add broadcastResize/Rows()** - `3c23414c` (feat)
2. **Task 2: Replace MC-06 hub tests with host-authority suite** - `8fd77291` (test)

## New Symbols

| Symbol | Location | Description |
|--------|----------|-------------|
| `Hub.broadcastResize(cols, rows uint16)` | `internal/relay/hub.go` | Non-blocking MsgResize fan-out; called after hub.mu.Unlock() |
| `Hub.Rows() int` | `internal/relay/hub.go` | PTY row count accessor; returns 50 fallback before first resize |
| `TestHub_ResizeHostAuthority_MinAmongLocal` | `internal/relay/hub_test.go` | D-02 min-among-local: two local subs → min grid |
| `TestHub_ResizeClientNoOpWhenDimensionsUnchanged` | `internal/relay/hub_test.go` | No-op when same dimensions; Origin:local stamped |
| `TestHub_ResizeFreezeLastHostSize` | `internal/relay/hub_test.go` | D-01 freeze: host disconnects, web guest cannot shrink |
| `TestHub_ResizeIgnoresWebOrigin` | `internal/relay/hub_test.go` | VIEW-02 / T-157-01: web sub never drives PTY grid |
| `TestHub_ResizeBroadcastsToSubscribers` | `internal/relay/hub_test.go` | VIEW-01: 0x02 frame with correct dims reaches all subs |
| `TestHub_RowsFallback` | `internal/relay/hub_test.go` | Rows() returns 50 before resize, ptyRows after |

## Files Created/Modified

- `internal/relay/hub.go` — ResizeClient rewritten, broadcastResize added, Rows() added, ptyCols/ptyRows comment updated
- `internal/relay/hub_test.go` — Three MC-06 tests replaced with six host-authority tests

## Decisions Made

- VIEW-02 origin gate placed as the FIRST check in ResizeClient, before acquiring hub.mu. Non-local origin returns immediately; the guest's reported size is not stored anywhere (not even in sub.Cols/sub.Rows). This is the single authoritative enforcement point for T-157-01.
- broadcastResize must be called only after hub.mu.Unlock() because it self-acquires mu. Calling while holding mu causes a self-deadlock (T-157-04 mitigation).
- Hub.Rows() fallback is 50, matching the `emuRows` default in engine.go (line 780). This ensures Plan 02's join-push sends a meaningful initial size even before the first resize event.
- MC-06 max-wins logic is fully removed; no test asserts max-wins or shrink-on-unsubscribe behavior.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## Known Stubs

None — all new symbols are fully implemented and connected.

## Threat Flags

No new security surfaces beyond the plan's threat model. T-157-01 and T-157-04 are both mitigated by the implementation:

- **T-157-01 (Tampering/DoS):** Origin gate at `ResizeClient` line 1 — non-local returns immediately. Asserted by `TestHub_ResizeIgnoresWebOrigin`.
- **T-157-04 (Self-deadlock):** `broadcastResize` acquires mu internally; called only after `hub.mu.Unlock()` per the unlock-before-IO discipline. No broadcast occurs while holding mu.

## Next Phase Readiness

Plan 02 (join-push, VIEW-03) can now:
- Call `hub.Rows()` to get the current authoritative PTY row count (or 50 fallback)
- Call `hub.Cols()` to get the current authoritative PTY column count (or 220 fallback)
- Rely on the frozen grid (D-01) when the host is temporarily disconnected during a guest join

---
*Phase: 157-terminal-screen-share-semantics-issue-109*
*Completed: 2026-06-27*

## Self-Check: PASSED

- `internal/relay/hub.go` — FOUND (modified)
- `internal/relay/hub_test.go` — FOUND (modified)
- Commit `3c23414c` — FOUND (feat: Task 1)
- Commit `8fd77291` — FOUND (test: Task 2)
- All 6 hub tests pass under `-race`
- Full relay package passes under `-race -short`
