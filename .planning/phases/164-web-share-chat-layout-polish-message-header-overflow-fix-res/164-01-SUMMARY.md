---
phase: 164-web-share-chat-layout-polish-message-header-overflow-fix-res
plan: "01"
subsystem: frontend/chat
tags: [chat, layout, overflow, fingerprint, vitest, tdd]
status: complete

dependency_graph:
  requires:
    - 163-01 (RO chat gate loosened — ChatPanel.tsx stable)
    - 159-05 (WEBCHAT-06 CSS ellipsis already in style.css)
  provides:
    - CHAT-LAYOUT-01 implementation (getRowStyle width constraint + formatAuthorFingerprint)
    - formatAuthorFingerprint exported helper (pure, unit-tested)
  affects:
    - frontend/src/components/Hub/ChatPanel.tsx (getRowStyle non-separator branch)
    - frontend/src/components/Hub/ChatMessage.tsx (secondary label render + new helper)
    - frontend/src/components/Hub/ChatPanel.test.tsx (extended)
    - frontend/src/components/Hub/ChatMessage.test.tsx (extended)
    - TESTING.md (Suite Manifest note + CHAT-LAYOUT-01 traceability rows)

tech_stack:
  added: []
  patterns:
    - "TDD RED/GREEN: test-first commits before each implementation commit"
    - "Reference-element pattern for jsdom hsl→rgb normalization in hue guard test"

key_files:
  created: []
  modified:
    - frontend/src/components/Hub/ChatPanel.tsx
    - frontend/src/components/Hub/ChatMessage.tsx
    - frontend/src/components/Hub/ChatPanel.test.tsx
    - frontend/src/components/Hub/ChatMessage.test.tsx
    - TESTING.md

decisions:
  - "Decision 1 (CONTEXT): getRowStyle non-separator branch adds width:'100%' + right:0 — bounds the absolutely-positioned row to the virtualizer container so the existing WEBCHAT-06 ellipsis on .chat-msg__tailnet-id has a constrained ancestor"
  - "Decision 2 (CONTEXT): formatAuthorFingerprint returns last 6 chars of authorID — compact stable disambiguator that avoids displaying the full raw nodekey in the chat header; degrades gracefully for short inputs"
  - "jsdom normalizes hsl→rgb in inline styles — avatar hue guard uses reference-element comparison instead of hsl string matching"

metrics:
  duration: "~6 minutes"
  completed: "2026-06-28"
  tasks_completed: 3
  files_changed: 5
---

# Phase 164 Plan 01: Chat Header Overflow Fix (CHAT-LAYOUT-01) Summary

Web-share chat layout polish: constrain virtualizer row width so the existing CSS ellipsis engages, and replace the full raw `nodekey:…` secondary label with a 6-char fingerprint for compact readable display.

## Tasks Completed

| # | Task | Commit | Type |
|---|------|--------|------|
| 1 RED | Add failing tests for getRowStyle width constraint | 8fe797be | test |
| 1 GREEN | Constrain virtualizer row width in getRowStyle | 901ea73a | feat |
| 2 RED | Add failing tests for formatAuthorFingerprint + render | 8647806d | test |
| 2 GREEN | Implement formatAuthorFingerprint + secondary label render | d47018ea | feat |
| 3 | Update TESTING.md — Suite Manifest note + traceability rows | 66731b02 | chore |

## What Was Built

### Task 1: getRowStyle width constraint (CHAT-LAYOUT-01 root-cause fix)

`getRowStyle()` in `ChatPanel.tsx` previously returned `position:absolute` rows with no explicit width, causing the virtualizer row to shrink-to-fit its content. This meant the row had no bounded container, so the existing `WEBCHAT-06` CSS ellipsis on `.chat-msg__tailnet-id` (added in Phase 159-05) had nothing to shrink against.

**Fix:** Non-separator branch now returns `width: '100%'` and `right: 0` in addition to the existing `position:'absolute'`, `top:0`, `left:0`, `transform:translateY(...)`. The separator branch is unchanged (no width, no transform — CHAT-04 Pitfall 1 preserved).

### Task 2: formatAuthorFingerprint helper + secondary label

Added a pure exported helper `formatAuthorFingerprint(authorID: string): string` to `ChatMessage.tsx`:
- Returns the last 6 characters of `authorID` for long inputs (e.g. `'nodekey:456d9361bab4eb7'` → `'ab4eb7'`)
- Returns the input unchanged for inputs ≤ 6 chars (e.g. `'local'` → `'local'`)
- Returns `''` for empty string (no throw)

The `.chat-msg__tailnet-id` secondary label now renders `({formatAuthorFingerprint(authorID)})` instead of `({authorID})`. The avatar hue (`tailnetIdToHue(authorID)`) and all mention matching continue to receive the full unchanged `authorID`.

### Task 3: TESTING.md update

Added Phase 164-01 Suite Manifest note and two CHAT-LAYOUT-01 traceability rows mapping the requirement to both extended test files.

## Verification Results

```
cd frontend && npx vitest run src/components/Hub/ChatMessage.test.tsx src/components/Hub/ChatPanel.test.tsx src/components/__tests__/style.hub.test.ts
Test Files  3 passed (3)
Tests       187 passed (187)
```

`bash tests/check-traceability-paths.sh` → `OK: all traceability paths exist`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] jsdom hsl→rgb normalization in avatar hue guard test**

- **Found during:** Task 2 RED phase
- **Issue:** The initial avatar guard test assertion `expect(avatar?.style.background).toContain('hsl(272,')` failed with `AssertionError: expected 'rgb(119, 52, 178)' to contain 'hsl(272,'` — jsdom normalizes `hsl(H, S%, L%)` to `rgb(r, g, b)` even on `getAttribute('style')`.
- **Fix:** Used a reference-element pattern: create a `div`, set its `style.background` to the expected hsl value, then compare `avatar.style.background` to `refElement.style.background`. Both go through jsdom normalization consistently, making the comparison valid.
- **Files modified:** `frontend/src/components/Hub/ChatMessage.test.tsx`
- **Commit:** 8647806d (in the RED commit — test was written correctly from the start)

No other deviations. Plan executed as written.

## Known Stubs

None. Both changes are complete functional implementations.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. The changes are purely client-side display helpers operating on data that already flows through the existing chat pipeline. T-164-01 and T-164-02 from the plan threat model are both addressed: the fingerprint is a strict REDUCTION of exposed data (T-164-01 accept), and the render uses a React text node with no `dangerouslySetInnerHTML` (T-164-02 mitigate — unchanged from before).

## Self-Check: PASSED

All implementation files exist on disk. All 5 task commits present in git log. No unexpected file deletions. Final vitest run: 187/187 passed.
