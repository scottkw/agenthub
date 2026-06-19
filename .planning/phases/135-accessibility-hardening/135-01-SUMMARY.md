---
phase: 135-accessibility-hardening
plan: 01
subsystem: frontend/css
tags: [accessibility, css, focus-visible, prefers-reduced-motion, a11y]
requirements: [A11Y-02, A11Y-03]

dependency_graph:
  requires: []
  provides:
    - GAP-135-A: :focus-visible rings on all Hub interactive elements
    - GAP-135-E: spin animation suppressed under prefers-reduced-motion: reduce
    - GAP-135-F: card hover transition suppressed under prefers-reduced-motion: reduce
  affects:
    - frontend/src/style.css
    - frontend/src/components/__tests__/style.hub.test.ts

tech_stack:
  added: []
  patterns:
    - Multi-selector :focus-visible group rule (mirroring lines 2726-2735 pattern)
    - prefers-reduced-motion: reduce block pair (mirroring lines 4927-4956 pattern)
    - readFileSync source-inspection assertions (established style.hub.test.ts pattern)

key_files:
  modified:
    - frontend/src/style.css
    - frontend/src/components/__tests__/style.hub.test.ts

decisions:
  - Replace .hub-card:focus in place with .hub-card:focus-visible (no duplicate rule per Pitfall 2)
  - Group all 10 new :focus-visible selectors into one rule block (mirrors find-bar pattern at lines 2726-2735)
  - Input selectors (.hub-filter__search:focus, .hub-modal__respond-input:focus) left unchanged (WCAG 2.4.7)
  - .hub__group-sidebar-new-input:focus kept as :focus (input), with outline added to existing rule
  - GAP-135-E and GAP-135-F reduce blocks placed AFTER the no-preference block to win the cascade

metrics:
  duration: "~8 minutes"
  completed: "2026-06-19T01:05:49Z"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 2
  tests_added: 13
  tests_total: 1725
---

# Phase 135 Plan 01: Hub CSS Accessibility Hardening Summary

**One-liner:** Hub CSS gains keyboard-only :focus-visible accent rings on all 11 interactive elements and two prefers-reduced-motion: reduce blocks that stop the running-status spinner and suppress card hover transitions.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add :focus-visible rings to all Hub interactive elements (GAP-135-A) | dbed1f1c | frontend/src/style.css |
| 2 | Add prefers-reduced-motion reduce blocks + source assertions (GAP-135-E/F) | 02b2501e | frontend/src/style.css, frontend/src/components/__tests__/style.hub.test.ts |

## What Was Built

### GAP-135-A: Focus-visible rings (Task 1)

- Changed `.hub-card:focus` to `.hub-card:focus-visible` in place (line 4313) — mouse clicks no longer trigger the outline ring
- Added a grouped `:focus-visible` rule covering 10 Hub interactive element selectors: `.hub-filter__pill`, `.hub-filter__new-session`, `.hub-card__open`, `.hub-card__menu-btn`, `.hub-card__menu-item`, `.hub-modal__close`, `.hub-modal__send-btn`, `.hub-modal__close-btn`, `.hub__group-sidebar-toggle`, `.hub__group-sidebar-item`
- Added `outline: 2px solid var(--hub-accent); outline-offset: 2px;` to the existing `.hub__group-sidebar-new-input:focus` rule (input element keeps `:focus`, not `:focus-visible`, per WCAG 2.4.7)
- All rings use `var(--hub-accent)` token (dark: `#7aa2f7`, light: `#3d6fe8`) — verified at source, never by eye (colorblind constraint)
- `.hub-filter__search:focus` and `.hub-modal__respond-input:focus` left completely unchanged

### GAP-135-E + GAP-135-F: Reduced-motion (Task 2)

Added two `prefers-reduced-motion: reduce` blocks after the existing attention-pulse reduce block (line 4974), in cascade-correct order:

1. **GAP-135-E** — `@media (prefers-reduced-motion: reduce) { .hub-card__status-icon--spin { animation: none; } }` — stops the ArrowPathIcon rotation. Running state remains identifiable by icon shape (unique among status icons) and "Running" text label.
2. **GAP-135-F** — `@media (prefers-reduced-motion: reduce) { .hub-card { transition: none; } }` — suppresses the 100ms/400ms border/background hover transition.

Extended `style.hub.test.ts` with 13 new source-inspection assertions covering GAP-135-A (focus-visible presence, Pitfall 2 guard, accent token, input :focus preservation) and GAP-135-E/F (regex span of reduce blocks, cascade ordering, no-preference block preservation).

## Verification

- `cd frontend && npx vitest run src/components/__tests__/style.hub.test.ts` — 61 passed (48 pre-existing + 13 new)
- `cd frontend && npx vitest run` — 1725 passed, 0 regressions across 105 test files
- Colorblind constraint: all focus rings use `var(--hub-accent)` token — asserted at source, never by eye
- A11Y-02 CSS portion: keyboard-only rings on all Hub interactive elements; mouse click no longer triggers ring
- A11Y-03: spin stops and card hover transition suppressed under prefers-reduced-motion: reduce

## Deviations from Plan

None — plan executed exactly as written. Both tasks followed the exact code patterns specified in PATTERNS.md and UI-SPEC.md.

## Known Stubs

None.

## Threat Flags

None — this plan modifies only client-side CSS. No new network surface, auth paths, or trust boundary crossings.

## Self-Check: PASSED

- [x] `frontend/src/style.css` — modified (focus-visible rules + reduce blocks added)
- [x] `frontend/src/components/__tests__/style.hub.test.ts` — modified (13 new Phase 135 assertions)
- [x] Commit dbed1f1c exists (Task 1)
- [x] Commit 02b2501e exists (Task 2)
- [x] All 1725 tests pass
