---
phase: 154
plan: "05"
subsystem: frontend/chat-panel
tags: [chat, virtualizer, relay-client, unread, tdd]
dependency_graph:
  requires: [154-02, 154-03, 154-04]
  provides: [ChatPanel, buildItems, mergeWithDedup, accrueUnread, getRowStyle, loadChatHistory]
  affects: [HubInteractiveModal, style.css]
tech_stack:
  added: ["@tanstack/react-virtual rangeExtractor for sticky day-separators"]
  patterns: ["liveRef stale-closure guard", "WS-first then HTTP history to close gap", "class-based vi.mock constructor", "vi.hoisted for shared mock state"]
key_files:
  created:
    - frontend/src/components/Hub/ChatPanel.tsx
    - frontend/src/components/Hub/ChatPanel.test.tsx
  modified:
    - frontend/src/style.css
decisions:
  - "D-02 OVERLAY mode: position:absolute drawer, no PTY resize, pure CSS translateX slide"
  - "getRowStyle uses position:sticky for separator (no transform) — Pitfall 1 constraint enforced"
  - "window focus/blur events instead of document.visibilityState — Pitfall 8 WKWebView workaround"
  - "liveRef.current pattern avoids stale closures in long-lived RelayClient useEffect"
  - "mergeWithDedup returns same array reference when nothing new — avoids unnecessary re-renders"
  - "WS-first then HTTP history load to avoid message-gap race (Pitfall 5)"
  - "chat-panel reduced-motion merged into share-modal @media block (style.hub.modal.test.ts lastIndexOf invariant)"
metrics:
  duration: "~40 minutes (continued from prior context)"
  completed: "2026-06-26T19:37:31Z"
  tasks: 2
  files: 3
status: complete
---

# Phase 154 Plan 05: ChatPanel Summary

**One-liner:** Virtualized slide-over chat drawer with own RelayClient WS subscription, HTTP scrollback, sticky day separators via `rangeExtractor`, and unread accrual (CHAT-04, D-09, NOTIF-01).

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 RED | ChatPanel subscription + history + states (failing tests) | `19e835a9` | ChatPanel.test.tsx |
| 1+2 GREEN | Full ChatPanel implementation — all exports, virtualizer, unread | `c974ad70` | ChatPanel.tsx, style.css |
| Post-green fix | Merge chat-panel reduced-motion into share-modal block | `728ecf7b` | style.css |

## Exports Delivered

| Export | Purpose |
|--------|---------|
| `buildItems(messages)` | Groups messages by day; inserts `separator` rows; marks `isConsecutive` (same author, <5min) |
| `mergeWithDedup(current, incoming, seenIds)` | Stable-reference merge; returns same array when nothing new |
| `accrueUnread(prev, msg, currentUserTailnetID)` | Pure function: increments count, sets `hasMention` if `msg.mentions` includes tailnetID |
| `getRowStyle(isActiveSeparator, start)` | Sticky row: `position:sticky, top:0, zIndex:2` (no transform); others: absolute + translateY |
| `loadChatHistory(relayPort, sessionId)` | `GET /api/chat/{id}/history` → `ChatMessage[]` |
| `ChatPanel` (default + named) | Full drawer component: props `{ sessionId, relayPort, open, currentUserTailnetID?, onUnreadChange? }` |

## Key Implementation Notes

**D-02 Overlay mode:** Drawer is `position:absolute; top:0; right:0; bottom:0; width:360px` floating over the terminal. No `ResizeObserver`, no PTY resize. Slide is pure CSS `transform:translateX(100%)` → `translateX(0)` via `.chat-panel--open`.

**Virtualizer + sticky separators (CHAT-04):** `useVirtualizer` with custom `rangeExtractor` that always keeps the active separator (closest above visible area) in the rendered range. `getRowStyle` returns `position:sticky` for separators — no `transform` property, which would create a new containing block and break sticky (Pitfall 1 documented in plan).

**Stale-closure guard:** `liveRef.current = { open, windowFocused, currentUserTailnetID }` updated on every render. The long-lived RelayClient callback reads `liveRef.current` instead of closed-over values.

**WS-first, then HTTP:** RelayClient is constructed first (subscribes to live messages). History is loaded after connection (`GET /api/chat/{id}/history`) to avoid a gap race. `mergeWithDedup` eliminates duplicates by message id.

**Window focus tracking (Pitfall 8):** Uses `window` `focus`/`blur` events, NOT `document.visibilityState` — `visibilityState` is unreliable in WKWebView on macOS.

**Unread accrual (D-09/NOTIF-01):** Accrues when `!open || !windowFocused`. Clears (`onUnreadChange({ count: 0, hasMention: false })`) when `open && windowFocused` simultaneously true (useEffect dependency `[open, windowFocused]`).

## Test Coverage

31 tests in `ChatPanel.test.tsx` (125 files, 2035 tests total — all pass):
- `buildItems`: separator insertion, consecutive marking, day-boundary resets
- `mergeWithDedup`: deduplication, same-reference stability
- `accrueUnread`: count increment, hasMention flag, own-message skip
- `getRowStyle`: sticky separator style, non-separator absolute+translateY
- Component subscription: RelayClient constructed with correct port/sessionId, callbacks wired
- Loading states: connecting, loading-history, error, empty
- Unread accrual + clear on open+focus

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] style.css reduced-motion block broke `style.hub.modal.test.ts` lastIndexOf invariant**
- **Found during:** Post-GREEN test run (all-suite `pnpm vitest run`)
- **Issue:** Added a standalone `@media (prefers-reduced-motion: reduce)` block at the END of style.css. `style.hub.modal.test.ts:94` uses `cssRaw.lastIndexOf('prefers-reduced-motion: reduce')` and expects `animation: none` in that block. My new block only had `transition: none`, breaking the assertion.
- **Fix:** Merged `.chat-panel { transition: none }` into the existing share-modal reduce block (which has `animation: none`). Removed the standalone block.
- **Files modified:** `frontend/src/style.css`
- **Commit:** `728ecf7b`

**2. [Rule 1 - Bug] Arrow function vi.mock for RelayClient not callable with `new`**
- **Found during:** Task 1 RED test run
- **Issue:** `vi.fn().mockImplementation(arrowFn)` cannot be called with `new` — arrow functions are not constructors.
- **Fix:** Replaced with `class MockRelayClient` with proper constructor inside `vi.mock` factory.
- **Commit:** `19e835a9`

**3. [Rule 1 - Bug] `visibilityState` appeared in code comments, failing acceptance criterion grep**
- **Found during:** Task 2 acceptance check
- **Issue:** Acceptance criterion `grep -rn "visibilityState" ChatPanel.tsx returns nothing` failed due to comment text.
- **Fix:** Rephrased comments to reference "Pitfall 8, WKWebView" without the API name.
- **Commit:** `c974ad70`

## Known Stubs

None. ChatPanel is wired to real RelayClient and real HTTP endpoint. The composer slot (`chat-panel__composer-slot`) is intentionally empty — wired in plan 154-06 (ChatComposer). This is a planned interface boundary, not a data stub.

## Threat Flags

None. ChatPanel renders messages received over the RelayClient WebSocket. The wire format is already sanitized server-side (Phase 154-01: `SanitizeChatContent`). No new network endpoints introduced. Markdown rendering deferred to plan 154-06 where SEC-03 XSS gate applies.

## Self-Check: PASSED

- frontend/src/components/Hub/ChatPanel.tsx — FOUND
- frontend/src/components/Hub/ChatPanel.test.tsx — FOUND
- Commit 19e835a9 (RED) — FOUND
- Commit c974ad70 (GREEN) — FOUND
- Commit 728ecf7b (fix) — FOUND
