---
phase: 152-relay-protocol-identity-presence
plan: "01"
subsystem: relay/protocol
status: complete
tags: [protocol, presence, chat, wire-format, security]
completed_date: "2026-06-26"
duration: "~3 minutes"
task_count: 2
file_count: 2
dependency_graph:
  requires: []
  provides:
    - MsgChat byte = 0x30 (Phase 154 stub)
    - MsgChatSend byte = 0x31 (Phase 154 stub)
    - MsgPresence byte = 0x32
    - MsgTyping byte = 0x33
    - MsgAliasSet byte = 0x34
    - PresenceEntry, PresencePayload, TypingPayload, AliasPayload structs
    - MakePresenceFrame, MakeTypingFrame, MakeAliasSetFrame encoders
    - ValidateAlias exported function
  affects:
    - internal/relay/hub.go (Plans 02/03 consume these symbols)
    - internal/relay/server.go (Plans 05/06 dispatch on these constants)
    - internal/webserver/server.go (Plans 05/06 dispatch)
    - frontend/src/lib/relayClient.ts (Plan 04 TypeScript parser)
tech_stack:
  added: []
  patterns:
    - "1-byte-prefix frame encoding (MakeMeta pattern extended to MakePresenceFrame/MakeTypingFrame/MakeAliasSetFrame)"
    - "TDD RED/GREEN: failing tests committed before implementation"
    - "Reject-not-truncate alias validation with C0/C1 control char rejection"
key_files:
  created:
    - internal/relay/protocol_presence_test.go
  modified:
    - internal/relay/protocol.go
decisions:
  - "Five constants defined now (0x30-0x34) to lock wire protocol before consumers are written; 0x30/0x31 are Phase 154 stubs"
  - "ValidateAlias exported (capital V) so both relay and webserver read pumps import from one place (single source of truth, Pitfall 3)"
  - "Reject-not-truncate at 32-rune cap: caller must inform sender to choose a shorter alias"
  - "C0 (U+0000-U+001F) and C1 (U+007F-U+009F) rejected to prevent control-char injection into terminal render surface"
---

# Phase 152 Plan 01: Add Chat/Presence Wire Protocol Foundation — Summary

Wire-protocol constants 0x30-0x34, JSON payload structs, Make*Frame encoders, and exported ValidateAlias added to internal/relay/protocol.go using TDD red/green cycles; full relay test suite passes with -race.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| RED | Failing tests for payload round-trips and ValidateAlias | b0c8245c | internal/relay/protocol_presence_test.go |
| GREEN | Frame constants, structs, Make*Frame encoders, ValidateAlias | a28c0a46 | internal/relay/protocol.go |

## What Was Built

**internal/relay/protocol.go** — extended with:

1. **Frame constants (0x30-0x34):**
   - `MsgChat byte = 0x30` — Phase 154 dispatch stub (server→client chat delivery)
   - `MsgChatSend byte = 0x31` — Phase 154 dispatch stub (client→server chat send)
   - `MsgPresence byte = 0x32` — server→client full presence roster
   - `MsgTyping byte = 0x33` — bidirectional typing-start/stop indicator
   - `MsgAliasSet byte = 0x34` — client→server alias update

2. **Payload structs** with canonical json tags:
   - `PresenceEntry{PersonKey, TailnetID, Origin, Alias string; ConnCount int}` with json tags `personKey/tailnetID/origin/alias/connCount`
   - `PresencePayload{Participants []PresenceEntry}` with json tag `participants`
   - `TypingPayload{PersonKey string (omitempty), Alias string (omitempty), Typing bool (non-omitempty)}`
   - `AliasPayload{Alias string}`

3. **Make*Frame encoders** (identical to MakeMeta pattern — json.Marshal + 1-byte prefix):
   - `MakePresenceFrame(PresencePayload) []byte`
   - `MakeTypingFrame(TypingPayload) []byte`
   - `MakeAliasSetFrame(AliasPayload) []byte`

4. **ValidateAlias(raw string) string** — exported, mitigates T-152-01:
   - Strips leading/trailing whitespace (strings.TrimSpace)
   - Rejects empty/whitespace-only input
   - Rejects input over 32 runes (does NOT truncate)
   - Rejects any rune in C0 (U+0000-U+001F) or C1 (U+007F-U+009F) ranges
   - Accepts printable Unicode including multibyte sequences

**internal/relay/protocol_presence_test.go** — new test file:
- `TestPresencePayloadRoundTrip` — encode/decode round-trip for PresencePayload
- `TestTypingPayloadRoundTrip` — encode/decode round-trip for TypingPayload
- `TestAliasPayloadRoundTrip` — encode/decode round-trip for AliasPayload
- `TestTypingPayload_TypingFalse` — Typing=false preserved (non-omitempty bool)
- `TestValidateAlias` — 17-case table test: trim, empty, 32/33-rune boundary, C0, C1, multibyte

## Verification

```
go test -race ./internal/relay/... → PASS (all 37 tests, including 5 new)
grep MsgPresence byte = 0x32 internal/relay/protocol.go → line 82
grep MsgTyping byte = 0x33 internal/relay/protocol.go → line 83
grep MsgAliasSet byte = 0x34 internal/relay/protocol.go → line 84
grep func ValidateAlias internal/relay/protocol.go → line 157
```

## Deviations from Plan

None - plan executed exactly as written.

Both tasks were implemented in the same TDD cycle (RED commit for all tests, GREEN commit for all implementation) since ValidateAlias logically belongs with the protocol constants and the test file covers both tasks as the plan specified.

## Threat Surface Scan

No new network endpoints, auth paths, or schema changes introduced. `ValidateAlias` is the T-152-01 mitigation — control-char rejection at ingress. No unplanned threat surface found.

## Self-Check

- [x] `internal/relay/protocol.go` exists and contains the 5 constants, 4 structs, 3 Make*Frame functions, ValidateAlias
- [x] `internal/relay/protocol_presence_test.go` exists with 5 test functions
- [x] Commit b0c8245c (RED) exists
- [x] Commit a28c0a46 (GREEN) exists
- [x] `go test -race ./internal/relay/...` exits 0

## Self-Check: PASSED
