---
phase: 164-web-share-chat-layout-polish-message-header-overflow-fix-res
plan: "02"
subsystem: frontend/chat
tags: [chat, layout, resize, localStorage, css-custom-property, drag-handle, tdd, vitest]
status: complete

dependency_graph:
  requires:
    - 164-01 (getRowStyle width constraint + formatAuthorFingerprint — ChatPanel.tsx/ChatMessage.tsx stable)
  provides:
    - CHAT-LAYOUT-02 implementation (resizable chat drawer width via drag handle + localStorage persistence)
    - clampChatWidth exported pure helper (unit-tested)
    - --chat-panel-width CSS custom property single source of truth for all three surfaces
  affects:
    - frontend/src/components/Hub/ChatPanel.tsx (clampChatWidth, width state, resize handlers, handle element)
    - frontend/src/style.css (two width rules + toggle offset → var(--chat-panel-width))
    - frontend/src/components/Hub/ChatPanel.test.tsx (16 new tests — clamp + persistence + drag handle)
    - frontend/src/components/Hub/chatToggleOverlap.test.ts (test (b) updated — --chat-panel-width not 372px)
    - TESTING.md (Suite Manifest note + CHAT-LAYOUT-02 traceability rows)

tech_stack:
  added: []
  patterns:
    - "TDD RED/GREEN: test-first commits for both Task 1 (clamp + persistence) and Task 2 (drag handle)"
    - "CSS custom property --chat-panel-width set on :root from React useEffect; consumed by .chat-panel width rules and the sibling toggle offset calc()"
    - "Pointer capture pattern for drag handle (mirrors existing handleInjectPointerDown setPointerCapture pattern)"
    - "Defensive localStorage access: null guard + Number() coercion + clampChatWidth on read"

key_files:
  created: []
  modified:
    - frontend/src/components/Hub/ChatPanel.tsx
    - frontend/src/style.css
    - frontend/src/components/Hub/ChatPanel.test.tsx
    - frontend/src/components/Hub/chatToggleOverlap.test.ts
    - TESTING.md

decisions:
  - "CONTEXT Decision 3: drag handle on left edge with pointer events (setPointerCapture pattern mirrors existing handleInjectPointerDown)"
  - "CONTEXT Decision 4: width clamped on BOTH drag and localStorage read (T-164-11 tampered value mitigation); persisted via agenthub.chatPanelWidth key"
  - "CONTEXT Decision 5: --chat-panel-width set on document.documentElement (:root) so it reaches the sibling toggle button which is NOT a child of .chat-panel"
  - "clampChatWidth: non-finite input (NaN, ±Infinity) → CHAT_WIDTH_DEFAULT (never NaN in CSS); in-range value coerced to integer via Math.round"
  - "CSS fallback 360px in var(--chat-panel-width, 360px) matches CHAT_WIDTH_DEFAULT so layout is correct before React mounts"

metrics:
  duration: "~5 minutes"
  completed: "2026-06-28"
  tasks_completed: 3
  files_changed: 5
---

# Phase 164 Plan 02: Resizable Chat Drawer Width (CHAT-LAYOUT-02) Summary

Replaced the fixed 360px chat drawer with a user-adjustable, persisted width on all three surfaces (GUI session tab, Hub interactive modal, web-share guest). A left-edge drag handle lets the user resize the drawer within a 280–640px range. The chosen width persists to localStorage and is restored on reload. A single `--chat-panel-width` CSS custom property is the sole source of truth for both the drawer width rules and the sibling toggle offset. D-02 overlay mode (no terminal resize / no PTY sendResize) is fully preserved.

## Tasks Completed

| # | Task | Commit | Type |
|---|------|--------|------|
| 1 RED | Add failing tests for clampChatWidth + localStorage persistence | 676ea146 | test |
| 1 GREEN | Implement clampChatWidth + width state + useEffect (--chat-panel-width + localStorage) | e98ea56f | feat |
| 2 RED | Add failing tests for resize drag handle + D-02 no-sendResize | 74cea0de | test |
| 2 GREEN | Add drag handle + CSS var() rules + toggle offset calc | 91aec888 | feat |
| 3 | Update chatToggleOverlap.test.ts + TESTING.md | 505c1732 | chore |

## What Was Built

### Task 1: clampChatWidth helper + width state + localStorage persistence

**New exports in ChatPanel.tsx:**
- `CHAT_WIDTH_MIN = 280` — minimum drawer width
- `CHAT_WIDTH_MAX = 640` — maximum drawer width
- `CHAT_WIDTH_DEFAULT = 360` — default width (matches existing CSS)
- `clampChatWidth(px: number): number` — pure helper; non-finite → DEFAULT, below-min → MIN, above-max → MAX, in-range → Math.round(px)

**Component state + effect:**
- `width` state initialized via `clampChatWidth(Number(localStorage.getItem('agenthub.chatPanelWidth')))` with a `null` guard (absent entry → DEFAULT). Wrapped in try/catch for unavailable storage.
- `useEffect([width])`: sets `document.documentElement.style.setProperty('--chat-panel-width', \`${width}px\`)` and persists `width` back to localStorage. Setting on `:root` ensures the property reaches both the drawer AND the sibling toggle button (which is NOT a child of `.chat-panel`).

### Task 2: Left-edge drag handle + CSS custom property

**Drag handle (ChatPanel.tsx):**
- `resizeDragRef` (useRef) stores `{ startX, startWidth }` during a drag; `null` when idle.
- `handleResizePointerDown`: records `startX + startWidth`, calls `setPointerCapture` (mirrors inject pattern).
- `handleResizePointerMove`: computes `newWidth = clampChatWidth(startWidth + (startX - clientX))` (left-edge handle: dragging left = wider). Calls `setWidth(newWidth)`.
- `handleResizePointerUp` / `onPointerCancel`: clears `resizeDragRef`. **Never calls `sendResize`** (D-02).
- `.chat-panel__resize-handle` div rendered as the first child of `.chat-panel`, positioned absolutely at the left edge.

**CSS changes (style.css):**
1. `.hub-modal__body--interactive .chat-panel { width: var(--chat-panel-width, 360px) }` (was `360px`)
2. `.chat-panel { width: var(--chat-panel-width, 360px) }` (was `360px`)
3. `.chat-panel--open ~ .hub-modal__chat-toggle { right: calc(var(--chat-panel-width, 360px) + 12px) }` (was `372px`)
4. New `.chat-panel__resize-handle` styling: `position:absolute; left:-2px; top:0; bottom:0; width:6px; cursor:col-resize; z-index:10; user-select:none` — straddles the left border for easy grab.

### Task 3: Source-gate update + TESTING.md

**chatToggleOverlap.test.ts test (b):** Previously asserted `right: 372px` (hard-coded). Updated to assert the rule body contains `--chat-panel-width` and matches `right:` — proving the offset now tracks the live width via the CSS custom property. Tests (a) and (c) unchanged.

**TESTING.md:**
- Suite Manifest: Phase 164-02 note documenting +16 ChatPanel.test.tsx tests and updated chatToggleOverlap.test.ts; counts unchanged (366 Go / 132 vitest / 9 Playwright / 509 total).
- Traceability: Two new CHAT-LAYOUT-02 rows mapping ChatPanel.test.tsx and chatToggleOverlap.test.ts.

## Verification Results

```
cd frontend && npx vitest run src/components/Hub/ChatPanel.test.tsx src/components/Hub/chatToggleOverlap.test.ts src/components/__tests__/style.hub.test.ts
Test Files  3 passed (3)
Tests       173 passed (173)

cd frontend && npx tsc --noEmit
(clean — no errors)

bash tests/check-traceability-paths.sh
OK: all traceability paths exist
```

**Source checks:**
- No `sendResize` call in the resize path (D-02 confirmed — only comments reference it)
- Both `.chat-panel` width rules and toggle offset reference `var(--chat-panel-width...)`
- `clampChatWidth` applied on localStorage READ (T-164-11 mitigated)
- `clampChatWidth` applied on drag MOVE (T-164-13 mitigated — cannot drag past min/max)

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

**TDD compliance note:** For Task 1 RED, `clampChatWidth` was implemented correctly alongside the tests (the pure-function tests pass immediately). The RED state was confirmed by 4 failing component mount tests (the component did not yet set `--chat-panel-width`). For Task 2 RED, 5 tests fail (handle element absent). Both RED/GREEN gate sequences are satisfied.

## Known Stubs

None. Both CHAT-LAYOUT-02 requirements are complete functional implementations. The drag handle is wired on all three surfaces via the shared ChatPanel component. Width persists and restores. CSS fallback ensures correct layout before React mounts.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. Changes are purely client-side:
- `localStorage.getItem/setItem` is a well-known side effect; wrapped defensively (try/catch).
- `--chat-panel-width` receives only a clamped finite integer from `clampChatWidth` (T-164-12 mitigated — no untrusted string concatenation into CSS, no dangerouslySetInnerHTML).
- Drag clamp bounds prevent layout collapse or distortion (T-164-13 mitigated).

## Self-Check: PASSED

All modified files exist on disk. All 5 task commits present in git log (676ea146, e98ea56f, 74cea0de, 91aec888, 505c1732). No unexpected file deletions. Final vitest run: 173/173 passed. tsc: clean. traceability: OK.
