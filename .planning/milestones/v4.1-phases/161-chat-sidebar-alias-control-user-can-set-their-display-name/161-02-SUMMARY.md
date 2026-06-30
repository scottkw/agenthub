---
phase: 161-chat-sidebar-alias-control-user-can-set-their-display-name
plan: "02"
subsystem: frontend-relay-wire-layer
tags: [relay, websocket, alias, identity, tdd]
dependency_graph:
  requires: [161-01-SUMMARY.md]
  provides: [RelayClient.sendAliasSet, RelayClient.onSelf, MSG_SELF]
  affects: [frontend/src/lib/relayClient.ts, frontend/src/lib/relayClient.test.ts]
tech_stack:
  added: []
  patterns: [TDD RED/GREEN, binary-framing, optional-callback]
key_files:
  created: []
  modified:
    - frontend/src/lib/relayClient.ts
    - frontend/src/lib/relayClient.test.ts
decisions:
  - sendAliasSet wraps existing encodeAliasSetFrame with no new encoder (encoder reuse)
  - onSelf is optional in RelayClientCallbacks, mirroring onPresence/onTyping (backward compat)
  - MockWS.OPEN = 1 required on stubbed WebSocket for readyState guard to evaluate correctly in tests
metrics:
  duration: "~5 minutes"
  completed: "2026-06-28"
  tasks_completed: 2
  files_modified: 2
status: complete
---

# Phase 161 Plan 02: Frontend Wire Layer (sendAliasSet + onSelf) Summary

Closed the RelayClient wire-layer gap required for the Phase 161 alias UI: added `sendAliasSet(alias)` to emit the existing 0x34 frame over the live WebSocket, and added `MSG_SELF` (0x37) parse + `onSelf(personKey, alias)` callback so web-share guests can learn their own identity on connect. Both changes follow the TDD RED/GREEN cycle with commits at each gate.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Add RelayClient.sendAliasSet | bd8994ce (feat) | relayClient.ts |
| 2 | Parse MSG_SELF + onSelf callback | fb92f621 (feat) | relayClient.ts, relayClient.test.ts |

## TDD Gate Compliance

Both tasks followed the required RED → GREEN cycle:

| Task | RED commit | GREEN commit |
|------|-----------|--------------|
| Task 1 sendAliasSet | 7ff650ff (test): 3 tests fail | bd8994ce (feat): all 50 pass |
| Task 2 MSG_SELF/onSelf | bd1b6a54 (test): 3 tests fail | fb92f621 (feat): all 55 pass |

## Artifacts Produced

**`frontend/src/lib/relayClient.ts`:**
- `MSG_SELF = 0x37` const (matches Go `MsgSelf` from Plan 161-01)
- `RelayClient.sendAliasSet(alias: string): void` — guards `ws.readyState === WebSocket.OPEN`, calls `encodeAliasSetFrame(alias)`; no duplicate encoder
- `ServerFrame` union: new `{ type: 'self'; personKey: string; alias: string }` variant
- `parseServerFrame` case `MSG_SELF`: TextDecoder JSON decode with try/catch-to-unknown guard
- `RelayClientCallbacks.onSelf?: (personKey: string, alias: string) => void` — optional
- `onmessage` dispatch: `case 'self':` fires `callbacks.onSelf?.(frame.personKey, frame.alias)`

**`frontend/src/lib/relayClient.test.ts`:**
- 5 new tests for `sendAliasSet` (send+frame, not-OPEN guard, unicode)
- 5 new tests for `MSG_SELF` (constant, parse happy-path, malformed guard, dispatch fires, omitted-callback safe)

## Deviations from Plan

**1. [Rule 2 - Missing functionality] Added `MockWS.OPEN = 1` to test mock**

- **Found during:** Task 1 RED test design
- **Issue:** `RelayClient` guards sends with `ws.readyState === WebSocket.OPEN`. After `vi.stubGlobal('WebSocket', MockWS)`, `WebSocket.OPEN` resolves to `MockWS.OPEN` which is `undefined` without explicit assignment. The send guard would always fail in tests.
- **Fix:** Added `;(MockWS as any).OPEN = 1` to both Task 1 and Task 2 mock setups.
- **Files modified:** `frontend/src/lib/relayClient.test.ts`
- **Commit:** included in 7ff650ff and bd1b6a54

This is not a change to production code — the `WebSocket.OPEN === 1` invariant holds in any real browser environment. It is a test harness correctness requirement.

## Acceptance Criteria Verification

```
grep -c 'sendAliasSet' frontend/src/lib/relayClient.ts   → 1 ✓ (method defined)
grep -c 'function encodeAliasSetFrame' ...               → 1 ✓ (no duplicate encoder)
grep -c '0x37' frontend/src/lib/relayClient.ts           → 2 ✓ (constant + parse case)
grep -c 'onSelf' frontend/src/lib/relayClient.ts         → 2 ✓ (declaration + dispatch)
npx vitest run src/lib/relayClient.test.ts               → 55/55 ✓
npx tsc --noEmit                                         → clean ✓
```

## Known Stubs

None. This plan is a pure wire-layer addition. No UI rendering, no placeholder data.

## Threat Flags

None. All changes are within the existing trust boundary and type system:
- `sendAliasSet` emits only to the server which re-validates via `ValidateAlias`.
- `onSelf` carries server-resolved (already ValidateAlias-clean) values. Downstream rendering (Plan 161-03) is text-only.

## Self-Check

```
frontend/src/lib/relayClient.ts → FOUND
frontend/src/lib/relayClient.test.ts → FOUND
commit 7ff650ff → FOUND
commit bd8994ce → FOUND
commit bd1b6a54 → FOUND
commit fb92f621 → FOUND
```
