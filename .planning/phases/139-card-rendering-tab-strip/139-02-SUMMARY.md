---
phase: 139-card-rendering-tab-strip
plan: 02
subsystem: ui
tags: [react, css, tab-strip, resize-observer, container-queries, jsdom]

# Dependency graph
requires:
  - phase: 139-01
    provides: TabBar chevron + floor + rename + title tests (TAB-01..03) — RED scaffold
provides:
  - Chrome-style tab strip: flex-shrink to 32px icon-only floor, @container hide of tab__name/tab__rename-input
  - Overflow chevrons (‹/›) gated on canScrollLeft/canScrollRight, ResizeObserver-driven
  - D-07 full-name title on outer .tab div (tooltip at floor)
  - ResizeObserver jsdom polyfill in test-setup.ts for component render tests
affects:
  - 139-03 (CARD-05 VT work — no file overlap, TAB cluster now done)
  - 139-04 (remaining plan in phase)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CSS container queries (@container) for element-level responsive hide — applied to .tab"
    - "ResizeObserver + passive scroll listener for scroll-position-aware state — TerminalPanel.tsx idiom"
    - "No-op ResizeObserver polyfill in test-setup.ts for jsdom-based component tests"

key-files:
  created: []
  modified:
    - frontend/src/components/TabBar.tsx
    - frontend/src/style.css
    - frontend/src/components/__tests__/TabBar.test.tsx
    - frontend/src/test-setup.ts

key-decisions:
  - "icon-only floor: 32px min-width content box, @container threshold 59px (allows status dot + close × at floor)"
  - "ResizeObserver polyfill in test-setup.ts (no-op) — keeps jsdom tests green without mocking the full API"
  - "Fixed afterEach guard bug in TAB-02 describe (Rule 1): source-level tests never assign root, so afterEach must nil-guard before unmount"

patterns-established:
  - "ResizeObserver cleanup: observe el, passive scroll listener, cleanup removes both (TerminalPanel idiom)"
  - "Chevron buttons: native <button> with aria-label, tabIndex=0, scrollBy smooth — no extra keydown needed"

requirements-completed: [TAB-01, TAB-02, TAB-03]

# Metrics
duration: 12min
completed: 2026-06-20
---

# Phase 139 Plan 02: Tab Strip Implementation Summary

**Chrome-style flex-shrink tab strip with @container icon-only floor (32px), ResizeObserver-driven overflow chevrons, and full-name title tooltip — all 31 TabBar tests GREEN**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-06-20T18:05:00Z
- **Completed:** 2026-06-20T18:08:30Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Tab strip now flex-shrinks from 180px max past 80px down to 32px icon-only floor; `.tab__name` and `.tab__rename-input` hide via `@container (max-width: 59px)` while status dot, close ×, and progress underline remain visible
- Overflow-aware chevrons (‹/›) render only when `canScrollLeft` / `canScrollRight` — driven by a ResizeObserver + passive scroll listener on the `.tab-list` element; both are cleaned up on unmount
- Full-name `title` attribute added to the outer `.tab` div (D-07) so hover tooltip works at the icon-only floor when `.tab__name` is CSS-hidden
- ResizeObserver no-op polyfill added to `test-setup.ts` enabling component render tests in jsdom

## Task Commits

Each task was committed atomically:

1. **Task 1: Tab flex-shrink + icon-only floor (CSS) — TAB-01** - `6ad9f35f` (feat)
2. **Task 2: Overflow chevrons + scroll-position awareness — TAB-02/TAB-03** - `3846e435` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `frontend/src/style.css` - `.tab` flex-shrink:1, min-width:32px, container-type:inline-size; `@container (max-width:59px)` hiding tab__name+tab__rename-input; `.tab-bar__chevron` rules
- `frontend/src/components/TabBar.tsx` - `listRef`, `canScrollLeft/Right` state, `checkScroll()`, `useEffect` with ResizeObserver+scroll listener, chevron JSX, outer `.tab` title attribute
- `frontend/src/components/__tests__/TabBar.test.tsx` - Rule 1 fix: `afterEach` nil-guard in TAB-02 describe
- `frontend/src/test-setup.ts` - ResizeObserver no-op polyfill (Rule 1 — jsdom lacks ResizeObserver)

## Chosen Floor Values

| Value | Setting | Rationale |
|-------|---------|-----------|
| `32px` | `min-width` on `.tab` | Content box floor — accommodates status dot (8px) + gap (6px) + close × (16px) + padding (10px×2) = ~50px total; 32px content box keeps close × legible |
| `59px` | `@container` threshold | Total tab width at which label disappears: 32px content + 10px padding each side + 7px border = ~59px outer |

## Decisions Made

- Used `container-type: inline-size` on `.tab` for container queries — allows each tab to independently hide its label based on its own rendered width, not the viewport
- Used `title={tab.name}` (full tab name only) on the outer `.tab` div rather than copying the rename-hint tooltip — simpler and matches D-07 "full tab name discoverable at floor"
- `@container (max-width: 59px)` hides only `tab__name` and `tab__rename-input` — explicitly excludes `tab__status`, `tab__close`, `tab__progress` per Pitfall 4

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] ResizeObserver not defined in jsdom**
- **Found during:** Task 2 (chevron useEffect mount)
- **Issue:** `new ResizeObserver(checkScroll)` throws in jsdom test environment — jsdom 29 does not implement ResizeObserver
- **Fix:** Added no-op polyfill class in `frontend/src/test-setup.ts` (`observe/unobserve/disconnect` are no-ops)
- **Files modified:** `frontend/src/test-setup.ts`
- **Verification:** All 31 TabBar tests pass; no ResizeObserver errors
- **Committed in:** `3846e435` (Task 2 commit)

**2. [Rule 1 - Bug] TAB-02 describe `afterEach` crashes for source-level tests**
- **Found during:** Task 2 (test run after chevron implementation)
- **Issue:** `afterEach` calls `root.unmount()` but source-level `it` blocks never assign `root`, so the call throws `TypeError: Cannot read properties of undefined (reading 'unmount')` — causing all 6 TAB-02 tests to fail even though their assertions passed
- **Fix:** Added nil guard: `if (root) root.unmount(); if (container) container.remove()`
- **Files modified:** `frontend/src/components/__tests__/TabBar.test.tsx`
- **Verification:** All 31 tests GREEN
- **Committed in:** `3846e435` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs in test infrastructure, not in plan logic)
**Impact on plan:** Both fixes necessary for the tests to actually turn GREEN. No scope creep.

## Issues Encountered

None beyond the two auto-fixed issues above.

## Human Check Required (Colorblind Note)

Visual/live app UAT should verify:
- At icon-only floor, the status dot remains visible and legible at its normal size (shape/position check, not color meaning)
- The close × button is fully clickable at the 32px floor
- Chevrons (‹/›) are visually distinguishable from the tab content area

(User is colorblind — verify legibility at source/size level, not by color meaning.)

## Next Phase Readiness

- TAB-01, TAB-02, TAB-03 fully delivered and GREEN
- Phase 139-03 (CARD-05 VT work) has no file overlap with this plan — can proceed independently
- `frontend/src/test-setup.ts` now has ResizeObserver polyfill available for any future component tests that need it

---
*Phase: 139-card-rendering-tab-strip*
*Completed: 2026-06-20*
