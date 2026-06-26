---
phase: 154-desktop-chat-ui
plan: "02"
subsystem: frontend/relay-client
tags: [relay-protocol, chat, typescript, vitest]
dependency_graph:
  requires: [154-01]
  provides: [ChatMessage-type, encodeChatSendFrame, encodeSessionInjectFrame, MSG_CHAT, MSG_CHAT_SEND, MSG_SESSION_INJECT, MSG_INJECT_ERROR, relay-dispatch-switch]
  affects: [154-03, 154-04, 154-05, 154-06]
tech_stack:
  added: ["@tanstack/react-virtual@3.14.4", "react-textarea-autosize@8.5.9"]
  patterns: [1-byte-prefix-JSON-frame-encoder, try-catch-parseServerFrame, optional-callback-chaining]
key_files:
  created: [frontend/src/lib/relayClient.test.ts]
  modified: [frontend/package.json, frontend/pnpm-lock.yaml, frontend/src/lib/relayClient.ts]
decisions:
  - alias field in ChatMessage mirrors Go json:"alias" tag (AuthorAlias field) — not authorAlias
  - all new RelayClientCallbacks members are optional (?) for TerminalPanel backward compat
  - ws.onmessage replaced with switch to dispatch all frame types via optional chaining
metrics:
  duration: "~4 minutes"
  completed: "2026-06-26"
  tasks_completed: 2
  files_changed: 4
status: complete
---

# Phase 154 Plan 02: Relay Client Wire-Protocol Foundation Summary

## One-liner
Installed @tanstack/react-virtual + react-textarea-autosize and extended relayClient.ts with MSG_CHAT/MSG_CHAT_SEND/MSG_SESSION_INJECT/MSG_INJECT_ERROR constants, ChatMessage interface, frame encoders, try/catch parse cases, and a switch-based onmessage dispatcher — backward compatible with TerminalPanel.

## Tasks Completed

| # | Name | Commit | Files |
|---|------|--------|-------|
| 1 | Install @tanstack/react-virtual + react-textarea-autosize | 723948ef | frontend/package.json, frontend/pnpm-lock.yaml |
| 2 | Extend relayClient.ts + tests | fb553936 | frontend/src/lib/relayClient.ts, frontend/src/lib/relayClient.test.ts |

## What Was Built

**Task 1 — Package installs:**
- `@tanstack/react-virtual@3.14.4` (satisfies `^3.14.3`) — virtualizer for ChatPanel message list
- `react-textarea-autosize@8.5.9` — auto-growing textarea for chat composer
- `rehype-raw` confirmed absent (SEC-03 guard)

**Task 2 — relayClient.ts extensions:**

New constants (following one-liner hex+arrow-comment style):
- `MSG_CHAT = 0x30` (server → client: chat message broadcast)
- `MSG_CHAT_SEND = 0x31` (client → server: post chat message)
- `MSG_SESSION_INJECT = 0x35` (client → server: inject text into PTY)
- `MSG_INJECT_ERROR = 0x36` (server → client: inject rejected)

`ChatMessage` interface — mirrors Go struct json tags exactly:
- `alias: string` (NOT `authorAlias` — the Go field is `AuthorAlias string json:"alias"`)
- All fields: v, id, sessionID, authorID, alias, content, mentions?, sessionInject?, ts

New encoders (copy of `encodeTypingFrame` 1-byte-prefix+JSON pattern):
- `encodeChatSendFrame(content)` → `[0x31, ...JSON({content})]`
- `encodeSessionInjectFrame(text)` → `[0x35, ...JSON({text})]`

Extended `ServerFrame` union: added `chat` and `inject_error` variants.

Extended `RelayClientCallbacks`: added optional `onPresence?`, `onTyping?`, `onChat?`, `onInjectError?` — all optional for TerminalPanel backward compat.

`parseServerFrame` — added MSG_CHAT and MSG_INJECT_ERROR cases using decode-slice(1)+JSON.parse+try/catch template; malformed frames return `{ type:'unknown' }`.

`ws.onmessage` — replaced if-chain with switch; dispatches all frame types via optional chaining (`this.callbacks.onChat?.(frame.message)`).

Added `sendChat(content)` and `sendSessionInject(text)` convenience methods to `RelayClient`.

**Tests (relayClient.test.ts):**
- Updated existing "Phase 152 stub" tests that now expect correct behavior for 0x30 (chat) and 0x31 (still unknown — client-to-server frame)
- Added Phase 154 behavior tests covering all 6 behaviors from plan:
  1. encodeChatSendFrame byte[0] === 0x31, body decodes to `{content}`
  2. encodeSessionInjectFrame byte[0] === 0x35, body decodes to `{text}`
  3. parseServerFrame 0x30 → `{type:'chat', message}` with `message.alias` populated
  4. parseServerFrame malformed 0x30 → `{type:'unknown'}` (try/catch guard)
  5. parseServerFrame 0x36 → `{type:'inject_error', reason}`
  6. RelayClient with only `onOutput` constructs successfully and does not throw on chat frame arrival
- **42/42 tests passing**
- `tsc --noEmit` clean — zero errors

## Verification Results

```
pnpm test run src/lib/relayClient.test.ts  →  42/42 tests passed
pnpm exec tsc --noEmit                      →  zero errors
grep rehype-raw frontend/package.json       →  returns nothing (SEC-03 satisfied)
pnpm ls @tanstack/react-virtual             →  3.14.4
pnpm ls react-textarea-autosize            →  8.5.9
```

## Deviations from Plan

None — plan executed exactly as written.

The package-legitimacy checkpoint (Task 0) was pre-approved by the human operator before this agent was spawned. The operator confirmed @tanstack/react-virtual is the canonical TanStack virtualizer (publisher: TanStack org, repo: github.com/TanStack/virtual, millions of weekly downloads, v3.14.3 exists).

## Known Stubs

None. All symbols produced by this plan are fully implemented:
- Constants: defined with correct hex values
- ChatMessage: all fields typed and correct
- Encoders: produce correct binary frames
- Parse cases: fully handle 0x30 and 0x36 frames
- Callbacks: optional chaining wired in dispatch switch

## Threat Surface Scan

No new network endpoints or trust boundaries introduced beyond what the plan's threat model covers. The `parseServerFrame` try/catch guards (T-154-05) are implemented as required. `rehype-raw` remains absent (SEC-03).

## Self-Check: PASSED
- `frontend/src/lib/relayClient.ts` exists and contains MSG_CHAT, ChatMessage, encodeChatSendFrame, encodeSessionInjectFrame
- `frontend/src/lib/relayClient.test.ts` exists with 42 passing tests
- Commits 723948ef and fb553936 confirmed in git log
- `alias: string` (not authorAlias) confirmed in ChatMessage interface
