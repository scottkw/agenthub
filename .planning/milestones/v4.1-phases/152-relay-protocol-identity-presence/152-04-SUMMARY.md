---
phase: 152-relay-protocol-identity-presence
plan: "04"
subsystem: frontend/relay-protocol
tags: [typescript, relay, presence, typing, wire-protocol]
status: complete

dependency_graph:
  requires: ["152-01"]
  provides: ["MSG_PRESENCE", "MSG_TYPING", "MSG_ALIAS_SET", "parseServerFrame-presence", "parseServerFrame-typing", "encodeTypingFrame", "encodeAliasSetFrame", "PresenceEntry"]
  affects: ["frontend/src/lib/relayClient.ts"]

tech_stack:
  added: []
  patterns:
    - "TDD RED/GREEN cycle — failing tests committed before implementation"
    - "Try/catch JSON.parse in parseServerFrame with {type:'unknown'} fallback for T-152-09 mitigation"
    - "Nullish fallback (participants ?? []) for missing roster field"

key_files:
  modified:
    - frontend/src/lib/relayClient.ts
    - frontend/src/lib/relayClient.test.ts

decisions:
  - "Field names in PresenceEntry interface (personKey, tailnetID, origin, alias, connCount) match Go json tags in protocol.go exactly — zero-diff wire interop"
  - "All JSON.parse of frame bodies wrapped in try/catch returning {type:'unknown'} — T-152-09 mitigation, malformed body cannot break message loop"
  - "default: unknown branch preserved for 0x30/0x31 (Phase 154 chat stubs) — backward-compatible"

metrics:
  duration: "5 minutes"
  completed: "2026-06-26"
  tasks_completed: 1
  tasks_total: 1
  files_modified: 2
---

# Phase 152 Plan 04: TypeScript Relay Protocol Wire Layer Summary

TypeScript side of the Phase 152 wire protocol: presence/typing/alias-set constants (0x32/0x33/0x34) mirroring Go `internal/relay/protocol.go` byte-for-byte, typed parse and encode surface, vitest-covered at 29/29 tests green, build passes.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| RED | Failing test cases for presence/typing/alias protocol | 0028ba5f | relayClient.test.ts |
| GREEN | Constants, PresenceEntry, ServerFrame variants, parse cases, encoders | 15b7653e | relayClient.ts |

## What Was Built

**`frontend/src/lib/relayClient.ts`** extended with:

- `MSG_PRESENCE = 0x32` (server→client), `MSG_TYPING = 0x33` (bidirectional), `MSG_ALIAS_SET = 0x34` (client→server) — values match Go `protocol.go` exactly
- `PresenceEntry` interface: `{ personKey, tailnetID, origin, alias, connCount }` — field names match Go `json:"..."` tags
- `ServerFrame` union extended with `{ type: 'presence'; participants: PresenceEntry[] }` and `{ type: 'typing'; personKey: string; alias: string; typing: boolean }`
- `parseServerFrame` case `MSG_PRESENCE` (0x32): try/catch JSON.parse → presence variant; malformed → `{type:'unknown'}`
- `parseServerFrame` case `MSG_TYPING` (0x33): try/catch JSON.parse → typing variant
- `default: {type:'unknown'}` preserved — 0x30/0x31 (Phase 154 stubs) and any future bytes are non-breaking
- `encodeTypingFrame(typing: boolean)`: `[0x33, ...JSON({typing})]` UTF-8 frame
- `encodeAliasSetFrame(alias: string)`: `[0x34, ...JSON({alias})]` UTF-8 frame

**`frontend/src/lib/relayClient.test.ts`** extended with 12 new vitest cases:

- Constant value assertions (0x32/0x33/0x34)
- Presence frame decode with full PresenceEntry field coverage
- Typing frame decode
- 0x30/0x31 backward-compat unknown assertions
- Malformed JSON body → unknown (no throw)
- Missing participants field → empty array fallback
- encodeTypingFrame leading byte and JSON body (true/false)
- encodeAliasSetFrame leading byte, JSON body, unicode alias

## Verification

- `pnpm test -- run src/lib/relayClient.test.ts`: 29/29 passed
- `pnpm run build`: clean (no TypeScript errors)
- `grep -nE 'MSG_PRESENCE *= *0x32'`: exits 0

## Deviations from Plan

None — plan executed exactly as written.

## Threat Mitigations Applied

| Threat | Mitigation |
|--------|-----------|
| T-152-09: JSON.parse of untrusted frame bodies | All parse cases wrapped in try/catch; malformed body → {type:'unknown'} not throw |
| T-152-10: alias rendering | This plan parses alias text into typed value only; rendering/escaping deferred to Phases 154/155 |
| T-152-SC: npm installs | No packages added — uses TextEncoder/TextDecoder/JSON/vitest from existing project |

## Known Stubs

None — this plan is protocol/parsing layer only. The chat UI panels (Phases 154/155) that consume these frames are intentionally deferred.

## Self-Check: PASSED

- `frontend/src/lib/relayClient.ts`: FOUND
- `frontend/src/lib/relayClient.test.ts`: FOUND
- RED commit 0028ba5f: verified in git log
- GREEN commit 15b7653e: verified in git log
- All 29 tests green: verified
- Build passes: verified
