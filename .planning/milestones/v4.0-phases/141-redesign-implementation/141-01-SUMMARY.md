---
phase: 141-redesign-implementation
plan: "01"
subsystem: ui
tags: [css-tokens, testing, aria, accessibility, hub, vitest, react]

# Dependency graph
requires: []
provides:
  - "--hub-text-dim CSS custom property in both :root (dark) and [data-ui-theme=light] blocks"
  - "GroupSidebar CARRY-01 ARIA contract encoded in tests (RED until plan 05)"
  - "StatusBar D-11 copy assertion encoded in tests (RED until plan 05)"
  - "SessionShareModal S-07 hub-share-modal structure smoke test"
affects: [141-02, 141-03, 141-04, 141-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "New --hub-* token insertion: add after last same-block token in both :root and [data-ui-theme=light]"
    - "RED test pattern: update test assertions to target contract before component ships"

key-files:
  created: []
  modified:
    - "frontend/src/style.css"
    - "frontend/src/components/Hub/GroupSidebar.test.tsx"
    - "frontend/src/components/__tests__/StatusBar.test.tsx"
    - "frontend/src/components/__tests__/SessionShareModal.test.tsx"

key-decisions:
  - "One new token only (--hub-text-dim); --hub-toggle-thumb-off reuses --hub-scrollbar-hover per D-03 latitude and RESEARCH A3"
  - "S-07 smoke assertion passes GREEN immediately because SessionShareModal already emits __header/__body; this is acceptable since the test encodes the structure contract"
  - "GroupSidebar keyboard tests use .click() on inner button instead of keydown dispatch, matching jsdom native-button activation semantics"

patterns-established:
  - "CARRY-01 ARIA model: <ul> no role; aria-labelledby on <ul>; <aside> aria-label='Session groups'; <button type=button> inside each <li> carries aria-pressed"
  - "D-11 copy: 'Share — open the Hub card' replaces 'Share links are on the Sessions tab'"

requirements-completed: [RDS-02, RDS-04, CARRY-01, RDS-03]

# Metrics
duration: 25min
completed: 2026-06-21
---

# Phase 141 Plan 01: Foundation Tokens & Wave-0 Test Contracts Summary

**--hub-text-dim token added to both theme blocks; GroupSidebar/StatusBar/SessionShareModal tests updated to Phase 141 ARIA and copy contracts (RED against current components)**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-06-21T10:30:00Z
- **Completed:** 2026-06-21T11:00:00Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- Added `--hub-text-dim` CSS custom property in both `:root` (dark: `#565f89`) and `[data-ui-theme=light]` (light: `#9999b0`) blocks in `style.css` — exactly 2 occurrences, no other token values changed
- Rewrote GroupSidebar.test.tsx to encode CARRY-01 ARIA contract: `aria-pressed` on inner `<button>`, `aria-labelledby` on `<ul>`, `aria-label="Session groups"` on `<aside>`, no `role=listbox/option` — suite is RED (10 failures) pending plan 05
- Updated StatusBar.test.tsx D-11 assertion to `'Share — open the Hub card'` — test is RED pending plan 05
- Added S-07 render-smoke describe block to SessionShareModal.test.tsx asserting `.hub-share-modal__header`, `__body`, and panel container — passes GREEN because component already emits these classes
- `tsc --noEmit` clean throughout

## Task Commits

1. **Task 1: Add --hub-text-dim token to both theme blocks** - `e908c348` (feat)
2. **Task 2: Update GroupSidebar tests to CARRY-01 ARIA contract** - `30cb6571` (test)
3. **Task 3: Update StatusBar D-11 + add S-07 smoke assertions** - `9f69a21a` (test)

## Files Created/Modified

- `frontend/src/style.css` — Added `--hub-text-dim` in `:root` dark block (line 3919) and `[data-ui-theme=light]` block (line 3972)
- `frontend/src/components/Hub/GroupSidebar.test.tsx` — CARRY-01 ARIA contract: aria-pressed, aria-labelledby, aside aria-label, inner-button tabIndex/click dispatch
- `frontend/src/components/__tests__/StatusBar.test.tsx` — D-11 copy assertion updated
- `frontend/src/components/__tests__/SessionShareModal.test.tsx` — S-07 hub-share-modal structure smoke describe block added

## Decisions Made

- One new token only (`--hub-text-dim`); consolidates `#545c7e` status-bar hint and `#565f89` breadcrumb separator per RESEARCH A2. No `--hub-toggle-thumb-off` token — plan 03 reuses `--hub-scrollbar-hover` per D-03 latitude.
- S-07 smoke assertion passes GREEN immediately because `SessionShareModal.tsx` already emits `__header`/`__body` wrapper divs. This is the correct behavior per the plan's conditional ("if the existing TSX does emit `__header`/`__body` wrappers, assert on those").
- GroupSidebar keyboard tests updated to use `.click()` on the inner `<button>` instead of `keydown` dispatch on `<li>`, since native buttons in jsdom respond to `.click()` without requiring synthetic keyboard events. Space `keydown` test retained on the button for coverage.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `--hub-text-dim` token is available for plans 02, 03, 04 to consume in CSS surface migrations
- GroupSidebar test suite (RED) defines the exact ARIA contract plan 05 must implement to turn GREEN
- StatusBar D-11 test (RED) defines the exact copy string plan 05 must render
- S-07 smoke assertions (GREEN) establish structural baseline for plan 04's inline style lift work

---
*Phase: 141-redesign-implementation*
*Completed: 2026-06-21*
