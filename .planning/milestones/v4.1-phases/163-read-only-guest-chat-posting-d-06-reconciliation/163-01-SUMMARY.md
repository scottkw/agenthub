---
phase: 163-read-only-guest-chat-posting-d-06-reconciliation
plan: 01
subsystem: relay
tags: [go, websocket, relay, webserver, chat, security, read-only]

requires:
  - phase: 154-desktop-chat-ui
    provides: hub.HandleChatSend, ErrChatReadOnly, relay+webserver MsgChatSend dispatch
  - phase: 152-relay-protocol-identity-presence
    provides: Subscriber.ReadOnly, D-06 design decision, ErrReadOnly gate

provides:
  - Removed ErrChatReadOnly symbol from relay/hub.go (D-06 reconciliation)
  - hub.HandleChatSend no longer gates on sub.ReadOnly — RO clients post chat
  - HandleInject ErrReadOnly gate and MsgInput RO discard byte-for-byte unchanged
  - Flipped hub_chatsend_test.go RO test to RO-can-post
  - Flipped server_chatsend_test.go relay path RO test to RO-can-post
  - Flipped server_chatsend_test.go web path RO test to RO-can-post
  - New SEC-RO-01 regression guard: TestHandleChatSend_ROCanPostInjectStillGated

affects:
  - 163-02-PLAN.md (frontend SC-3 suppression removal — ChatPanel isReadOnly gate)
  - 163-03-PLAN.md (TESTING.md update — test rename delta)

tech-stack:
  added: []
  patterns:
    - "D-06 enforcement point: only MsgInput and MsgSessionInject are RO-gated; chat is not"
    - "SEC-RO-01 regression guard: single test proves RO can chat AND inject stays blocked AND PTY=0"

key-files:
  created: []
  modified:
    - internal/relay/hub.go
    - internal/relay/server.go
    - internal/webserver/server.go
    - internal/relay/hub_chatsend_test.go
    - internal/relay/server_chatsend_test.go
    - internal/webserver/server_chatsend_test.go

key-decisions:
  - "D-06 reconciliation: loosened ONLY HandleChatSend; HandleInject (ErrReadOnly) and MsgInput discard unchanged"
  - "ErrChatReadOnly deleted (was unreferenced after removing the gate); ErrReadOnly retained for inject"
  - "SEC-RO-01 regression guard combined into single test function: RO-chat-ok + inject-gated + PTY=0 in one pass"

patterns-established:
  - "When a single hub function is the sole enforcement point for both relay and webserver paths, one edit covers both surfaces"

requirements-completed: [ROCHAT-01, ROCHAT-02, SEC-RO-01]

coverage:
  - id: D1
    description: "hub.HandleChatSend allows RO subscribers to post chat (removed ErrChatReadOnly gate)"
    requirement: ROCHAT-01
    verification:
      - kind: unit
        ref: "internal/relay/hub_chatsend_test.go#TestHandleChatSend_ROCanPost"
        status: pass
      - kind: integration
        ref: "internal/relay/server_chatsend_test.go#TestChatSend_ROCanPost_RelayPath"
        status: pass
      - kind: integration
        ref: "internal/webserver/server_chatsend_test.go#TestChatSend_ROCanPost_WebPath"
        status: pass
    human_judgment: false
  - id: D2
    description: "HandleInject still returns ErrReadOnly for RO subscribers (inject gate unchanged)"
    requirement: ROCHAT-02
    verification:
      - kind: unit
        ref: "internal/relay/hub_chatsend_test.go#TestHandleChatSend_ROCanPostInjectStillGated"
        status: pass
      - kind: integration
        ref: "internal/relay/server_inject_test.go#TestInject_ROCap_RelayPath"
        status: pass
      - kind: integration
        ref: "internal/webserver/inject_test.go#TestInjectRO_WebPath"
        status: pass
    human_judgment: false
  - id: D3
    description: "Only HandleChatSend loosened; MsgInput RO discard and HandleInject gate byte-for-byte unchanged (SEC-RO-01)"
    requirement: SEC-RO-01
    verification:
      - kind: unit
        ref: "internal/relay/hub_chatsend_test.go#TestHandleChatSend_ROCanPostInjectStillGated"
        status: pass
      - kind: integration
        ref: "internal/webserver/capability_test.go#TestSecurity_ReadOnlyCapabilityBlocksMsgInput"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-06-28
status: complete
---

# Phase 163 Plan 01: Remove RO Chat-Send Gate (D-06 Reconciliation — Server Half) Summary

**ErrChatReadOnly removed from hub.HandleChatSend so RO-cap guests can post chat on both relay and webserver paths; HandleInject ErrReadOnly and MsgInput RO discard unchanged and regression-proven in a single combined test.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-06-28T20:08:14Z
- **Completed:** 2026-06-28T20:14:30Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Removed `ErrChatReadOnly` symbol and the `if sub.ReadOnly { return ErrChatReadOnly }` early-return from `hub.HandleChatSend` — the single enforcement point covering both the relay and webserver MsgChatSend dispatch paths (no per-server edit needed)
- Updated `HandleChatSend` doc comment to reflect D-06: RO clients are full chat participants; only MsgInput and MsgSessionInject remain RO-gated
- Rewrote MsgChatSend case comments in `relay/server.go` and `webserver/server.go` to reference D-06 instead of the removed SEC-01 chat gate
- Flipped all three RO-cannot-post test functions to RO-can-post: `TestHandleChatSend_ROCanPost`, `TestChatSend_ROCanPost_RelayPath`, `TestChatSend_ROCanPost_WebPath`
- Added `TestHandleChatSend_ROCanPostInjectStillGated` — SEC-RO-01 dual-invariant regression guard proving in one test that RO HandleChatSend succeeds AND HandleInject returns ErrReadOnly AND PTY write count is 0

## Test Count Delta (for 163-03 TESTING.md update)

- `internal/relay/hub_chatsend_test.go`: renamed `TestHandleChatSend_ROReturnsError` → `TestHandleChatSend_ROCanPost`; added `TestHandleChatSend_ROCanPostInjectStillGated` (+1 net)
- `internal/relay/server_chatsend_test.go`: renamed `TestChatSend_RODropped_RelayPath` → `TestChatSend_ROCanPost_RelayPath` (0 net)
- `internal/webserver/server_chatsend_test.go`: renamed `TestChatSend_RODropped_WebPath` → `TestChatSend_ROCanPost_WebPath` (0 net)

## Task Commits

1. **Task 1: Remove the RO chat-send gate (hub.HandleChatSend, relay/server.go, webserver/server.go)** - `dce5559c` (feat)
2. **Task 2: Flip RO-cannot-post tests to RO-can-post; add SEC-RO-01 regression guard** - `57411937` (test)

## Files Created/Modified

- `internal/relay/hub.go` — removed ErrChatReadOnly symbol; removed RO gate from HandleChatSend; updated doc comment per D-06
- `internal/relay/server.go` — rewrote MsgChatSend dispatch comment (D-06 reference, no code change)
- `internal/webserver/server.go` — rewrote MsgChatSend dispatch comment (D-06 reference, no code change)
- `internal/relay/hub_chatsend_test.go` — replaced ROReturnsError test with ROCanPost; added ROCanPostInjectStillGated
- `internal/relay/server_chatsend_test.go` — replaced RODropped_RelayPath with ROCanPost_RelayPath
- `internal/webserver/server_chatsend_test.go` — replaced RODropped_WebPath with ROCanPost_WebPath

## Decisions Made

- D-06 reconciliation: loosened ONLY `HandleChatSend`; `HandleInject` (`ErrReadOnly`) and the MsgInput `!sub.ReadOnly` discard are byte-for-byte unchanged
- `ErrChatReadOnly` deleted entirely — it was only referenced at the removed gate and in the test being flipped; keeping a dead exported symbol violates project no-silent-fallbacks principle
- SEC-RO-01 regression guard combined into a single test (`TestHandleChatSend_ROCanPostInjectStillGated`) using `makePTYCountingHub` to prove all three invariants atomically

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

Go build cache (ENOENT) on the first full test run after a prior background process; `go clean -cache` resolved it. Not a code issue.

## Next Phase Readiness

- Server gate removed; tests green under `-race`
- 163-02 can now remove the frontend SC-3 suppression (`isReadOnly` gate in `ChatPanel.tsx`)
- 163-03 must update `TESTING.md` with the renamed/added test functions per the delta table above

## Self-Check

- `internal/relay/hub.go` modified: EXISTS
- `internal/relay/hub_chatsend_test.go` modified: EXISTS
- `internal/relay/server_chatsend_test.go` modified: EXISTS
- `internal/webserver/server_chatsend_test.go` modified: EXISTS
- Commit `dce5559c` (Task 1): VERIFIED
- Commit `57411937` (Task 2): VERIFIED
- `go build ./internal/...`: PASSED
- `go test ./internal/relay/... ./internal/webserver/... -race -count=1`: PASSED (relay 4.7s, webserver 12.1s)
- `grep -rn "ErrChatReadOnly" internal/ --include="*.go" | grep -v hub.go`: EMPTY (only comment reference in hub.go)

## Self-Check: PASSED

---
*Phase: 163-read-only-guest-chat-posting-d-06-reconciliation*
*Completed: 2026-06-28*
