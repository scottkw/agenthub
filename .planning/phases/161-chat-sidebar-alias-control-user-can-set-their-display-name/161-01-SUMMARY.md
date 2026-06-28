---
phase: 161-chat-sidebar-alias-control-user-can-set-their-display-name
plan: "01"
subsystem: relay-protocol
tags: [self-identity, wire-protocol, websocket, alias, go]
requires: [152-06-SUMMARY.md]
provides: [MsgSelf-0x37-wire-frame, SelfPayload-struct, on-connect-self-identity-emission]
affects: [internal/relay/protocol.go, internal/relay/server.go, internal/webserver/server.go]
tech_stack:
  added: []
  patterns: [server-push-hint, additive-frame, tdd-red-green]
key_files:
  created: []
  modified:
    - internal/relay/protocol.go
    - internal/relay/server.go
    - internal/webserver/server.go
    - internal/relay/protocol_presence_test.go
    - internal/relay/server_identity_test.go
    - internal/webserver/identity_test.go
    - internal/relay/server_test.go
    - internal/webserver/server_test.go
decisions:
  - "MsgSelf = 0x37 (next free value after MsgInjectError 0x36 in the 0x3x block)"
  - "Frame emitted unconditionally — not gated on perms — so RO caps receive the self hint"
  - "Insertion point: after MakeResizeFrame write, before scrollback replay (mirrors VIEW-03 direct conn.Write ordering)"
  - "Server-only, one-way: no dispatch/decode counterpart on server (client never sends 0x37)"
  - "Test helpers in server_test.go add MsgSelf to housekeeping-skip lists (Rule 1 auto-fix)"
metrics:
  duration: "4 minutes"
  completed: 2026-06-28
  tasks_completed: 2
  files_changed: 8
status: complete
---

# Phase 161 Plan 01: MsgSelf On-Connect Self-Identity Frame — Summary

Server-side additive wire frame (MsgSelf 0x37) emitted once per WS connect on both the relay (desktop) and webserver (web-share) paths, carrying the connecting client's own personKey and resolved alias.

## What Was Built

### Task 1: MsgSelf frame constant, SelfPayload, MakeSelfFrame (protocol.go)

Added to `internal/relay/protocol.go`:

- `MsgSelf byte = 0x37` constant in the 0x3x reserved block, documented as server→client only, Phase 161.
- `SelfPayload` struct with fields `PersonKey string \`json:"personKey"\`` and `Alias string \`json:"alias"\`` — lowerCamel JSON tags matching the TS parser wire contract (Plan 161-02 agreement point).
- `MakeSelfFrame(p SelfPayload) []byte` — mirrors `MakeAliasSetFrame`/`MakeTypingFrame` pattern exactly (json.Marshal + prepend type byte).

Test added: `TestMakeSelfFrame` in `protocol_presence_test.go` — asserts 0x37 leading byte and JSON round-trip with both `personKey` and `alias` lowerCamel keys.

### Task 2: On-connect emission on both server paths

**Relay path** (`internal/relay/server.go`): `conn.Write(MakeSelfFrame(SelfPayload{PersonKey: sub.PersonKey, Alias: sub.Alias}))` inserted immediately after the resize-frame write block (~line 301), before scrollback replay. sub.PersonKey ("local:local") and sub.Alias are already resolved above at lines 263-270.

**Webserver path** (`internal/webserver/server.go`): identical pattern using `relay.MakeSelfFrame`, inserted after the resize write (~line 1156), before scrollback replay. sub.PersonKey (tailnetID:web) and sub.Alias are resolved at lines 1142-1144. Not gated on perms — read-only caps receive the frame.

Tests added:
- `TestRelayIdentity_SelfFrameOnConnect` — relay harness, asserts personKey="local:local" + alias="host"
- `TestWebIdentity_SelfFrameOnConnect` — web harness, asserts personKey has ":web" suffix, not "local:local"
- `TestWebIdentity_ReadOnlySelfFrame` — confirms RO cap (perms="read") also receives the self frame

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Existing ordering tests broke when MsgSelf inserted between MsgResize and scrollback**

- **Found during:** Task 2 — full-package test run after implementing the on-connect emission
- **Issue:** `readDataFrame` (relay), `readOrdered` (relay), `readWebFrame` (webserver), and the inline loop in `TestWebServerWSS` all skip `MsgMeta`/`MsgPresence`/`MsgResize` as housekeeping frames, but not the new `MsgSelf`. Tests like `TestRelayJoin_PushesResizeBeforeScrollback` (relay) and `TestWebJoin_PushesResizeBeforeScrollback` + `TestWebServerWSS` (webserver) received MsgSelf where they expected MsgOutput, causing test failures.
- **Fix:** Added `MsgSelf` to the housekeeping-skip conditions in all four locations.
- **Files modified:** `internal/relay/server_test.go`, `internal/webserver/server_test.go`
- **Commit:** 60e78122

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 (RED+GREEN) | d0c66cfe | feat(161-01): add MsgSelf (0x37) frame constant, SelfPayload, MakeSelfFrame |
| Task 2 (RED+GREEN + Rule 1) | 60e78122 | feat(161-01): emit MsgSelf on connect on relay + webserver paths |

## Wire Contract (authoritative for Plan 161-02)

- Frame type byte: `0x37`
- JSON body: `{"personKey": string, "alias": string}`
- JSON tag names: lowerCamel (`personKey`, `alias`) — must match the TS `MSG_SELF` parser
- Direction: server → client only (the server never reads this frame type)
- Timing: after MsgResize, before scrollback replay — deterministic ordering

## Self-Check: PASSED

- FOUND: internal/relay/protocol.go
- FOUND: internal/relay/server.go
- FOUND: internal/webserver/server.go
- FOUND: internal/relay/protocol_presence_test.go
- FOUND: internal/relay/server_identity_test.go
- FOUND: internal/webserver/identity_test.go
- FOUND: commit d0c66cfe (task 1)
- FOUND: commit 60e78122 (task 2)
- `grep -c 'MsgSelf' internal/relay/protocol.go` = 4 (>= 2 required)
- `grep -c 'MakeSelfFrame'` = 1 on both server paths
- All tests pass: `go test ./internal/relay/... ./internal/webserver/... -count=1`
