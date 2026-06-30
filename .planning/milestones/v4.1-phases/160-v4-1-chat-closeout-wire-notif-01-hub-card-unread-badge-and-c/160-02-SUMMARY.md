---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: "02"
subsystem: frontend/Hub
tags: [notif-01, unread-badge, prop-threading, tdd, hub-panel]
status: complete

dependency_graph:
  requires: [160-01]
  provides: [NOTIF-01-A]
  affects:
    - frontend/src/components/Hub/HubInteractiveModal.tsx
    - frontend/src/components/Hub/HubModal.tsx
    - frontend/src/components/Hub/SessionCardGrid.tsx
    - frontend/src/components/Hub/HubPanel.tsx

tech_stack:
  added: []
  patterns:
    - functional setState with new Map(prev) for unreadMap mutations (Pitfall 5)
    - optional prop threading through multi-level component tree
    - TDD RED/GREEN/REFACTOR cycle across 3 tasks

key_files:
  created: []
  modified:
    - frontend/src/components/Hub/HubInteractiveModal.tsx
    - frontend/src/components/Hub/HubModal.tsx
    - frontend/src/components/Hub/SessionCardGrid.tsx
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/HubInteractiveModal.test.tsx
    - frontend/src/components/Hub/SessionCardGrid.test.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx

decisions:
  - "Used vi.mock('./HubModal') in HubPanel.test.tsx to capture onUnreadChange
    without mounting the full modal stack; existing FE-ROUTE-01 and GAP-134-D
    tests preserved by rendering hub-modal-overlay + hub-modal__close in mock"
  - "handleUnreadChange defined as a plain function (not useCallback) in HubPanel
    since it is only referenced by useChatUnreadListeners and HubModal render
    site — stable enough for the exclusion-id dependency pattern"
  - "Reset unreadMap entry via m.delete(session.id) in handleCardClick before
    setModalState — both updates batch in React 18 so badge and modal state
    change in the same commit, badge never flickers visible while modal is open"

metrics:
  duration: "~30 minutes"
  completed: "2026-06-28T03:31:09Z"
  tasks_completed: 3
  tasks_total: 3
  files_changed: 7
  tdd_gate_compliance: RED+GREEN all three tasks
---

# Phase 160 Plan 02: NOTIF-01 Prop Threading Summary

Part A of the NOTIF-01 fix: full prop-threading chain from ChatPanel's
`onUnreadChange(count, hasMention)` through HubInteractiveModal (sessionId
injected) → HubModal → HubPanel's `unreadMap` → SessionCardGrid (both
render sites) → SessionCard's existing ChatBadge. Closes the v4.1 milestone
audit BLOCKER — Hub card unread badge was dead-wired.

## What Was Built

### Task 1: Lift unread out of the modal

**HubInteractiveModal** gains optional `onUnreadChange?(sessionId,count,hasMention)`
prop. `handleUnreadChange` continues to update local toggle-badge state AND
now calls `onUnreadChange?.(session.id, count, mention)` — injecting the
session id that ChatPanel's callback lacks.

**HubModal** gains the same optional prop and threads it to `<HubInteractiveModal>`
at the interactive-modal render site only (HubBriefingModal has no chat).

### Task 2: Thread unreadBySessionId through SessionCardGrid

**SessionCardGrid** gains optional `unreadBySessionId?: Map<string,{count,hasMention}>`.
Both SessionCard render sites (named-group path and workDir-fallback path)
now pass `unreadCount={unreadBySessionId?.get(s.id)?.count}` and
`hasChatMention={unreadBySessionId?.get(s.id)?.hasMention}`. SessionCard.tsx
was not modified (already renders ChatBadge from these props).

### Task 3: Wire HubPanel unreadMap, hook, reset-on-open, and props

**HubPanel** additions:
- `unreadMap` state (`Map<sessionId,{count,hasMention}>`) initialized empty
- `handleUnreadChange(sessionId,count,hasMention)` using functional setState +
  `new Map(prev)` (never mutates in place)
- `handleCardClick` resets the opening session's map entry via `m.delete(session.id)`
  before calling `setModalState`
- `useChatUnreadListeners(sessions, relayPort??0, modalState?.session.id??null,
  isActive??false, handleUnreadChange)` call — backgrounded session listener
  from Plan 160-01; exclusion id prevents double-counting
- `<HubModal onUnreadChange={handleUnreadChange}>` and
  `<SessionCardGrid unreadBySessionId={unreadMap}>` receive the new props

## Test Results

```
vitest run src/components/Hub/ — 377/377 PASS
tsc --noEmit               — CLEAN (exit 0)
```

**TDD gate compliance:**
- Task 1: test(160-02) RED → feat(160-02) GREEN ✓
- Task 2: test(160-02) RED → feat(160-02) GREEN ✓
- Task 3: test(160-02) RED → feat(160-02) GREEN ✓

New tests added:
- `HubInteractiveModal.test.tsx` +2 tests (NOTIF-01 sessionId injection)
- `SessionCardGrid.test.tsx` +4 tests (NOTIF-01 unreadBySessionId threading — both render sites)
- `HubPanel.test.tsx` +7 tests (5 source-inspection + 2 behavioral with HubModal mock)

## Commits

| Hash | Type | Subject |
|------|------|---------|
| 9bd04519 | test | add failing tests for HubInteractiveModal sessionId injection |
| 12388e1c | feat | add onUnreadChange prop to HubInteractiveModal and HubModal |
| 382373b3 | test | add failing tests for SessionCardGrid unreadBySessionId threading |
| dbb7180d | feat | thread unreadBySessionId through SessionCardGrid to both SessionCard sites |
| 1716e094 | test | add failing tests for HubPanel unreadMap wiring and reset-on-open |
| e52128b2 | feat | wire HubPanel unreadMap, useChatUnreadListeners, reset-on-open, and prop pass-down |
| 3538f854 | chore | remove stale casts in tests after GREEN phase adds props to interfaces |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Remove stale type casts after GREEN phase adds props to interfaces**
- **Found during:** Post-GREEN tsc gate
- **Issue:** RED-phase tests used `as React.ComponentType<Record<string,unknown>>` cast
  in `HubInteractiveModal.test.tsx` and spread cast `{...({unreadBySessionId} as Record<string,unknown>)}`
  in `SessionCardGrid.test.tsx` to bypass unknown-prop type errors. After GREEN added
  the props to the interfaces, these casts caused TS2352 type errors caught by tsc.
- **Fix:** Removed casts; tests now pass typed props directly.
- **Files modified:** `HubInteractiveModal.test.tsx`, `SessionCardGrid.test.tsx`
- **Commit:** 3538f854

None beyond the tsc cleanup above.

## Known Stubs

None. All unread badge data is live (from ChatPanel callbacks and the
useChatUnreadListeners hook). No placeholder values or hardcoded mock data
in the shipped implementation.

## Threat Flags

None. This plan is pure client-side prop threading with no new network
endpoints, auth paths, or trust boundary crossings (T-160-03 accepted in
plan threat model — badge exposes only count + mention boolean).

## Self-Check: PASSED

All source files verified present on disk.
All 7 plan commits verified in git history.
377/377 vitest tests pass. tsc --noEmit: clean (exit 0).
