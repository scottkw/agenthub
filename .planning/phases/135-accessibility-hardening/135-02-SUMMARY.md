---
phase: 135-accessibility-hardening
plan: "02"
subsystem: frontend/hub
tags: [accessibility, keyboard, aria, react, a11y-02]
dependency_graph:
  requires: [135-01]
  provides: [aria-pressed on filter pills, keyboard operability for group sidebar items]
  affects: [HubFilterBar.tsx, GroupSidebar.tsx]
tech_stack:
  added: []
  patterns: [aria-pressed toggle-button semantics, tabIndex+onKeyDown on non-button interactive elements]
key_files:
  created: []
  modified:
    - frontend/src/components/Hub/HubFilterBar.tsx
    - frontend/src/components/Hub/HubFilterBar.test.tsx
    - frontend/src/components/Hub/GroupSidebar.tsx
    - frontend/src/components/Hub/GroupSidebar.test.tsx
decisions:
  - "aria-pressed (not aria-selected) on filter pills — buttons in role=group use toggle-button semantics per RESEARCH Pattern 4"
  - "GroupSidebarItem keyboard handler copies the SessionCard.tsx Enter/Space pattern verbatim"
metrics:
  duration: "~8 minutes"
  completed: "2026-06-18"
  tasks: 2
  files_modified: 4
---

# Phase 135 Plan 02: Hub A11Y-02 Component Keyboard/ARIA Hardening Summary

**One-liner:** aria-pressed toggle semantics on HubFilterBar pills + tabIndex/onKeyDown Enter-Space on GroupSidebarItem li, completing the two non-modal A11Y-02 component gaps.

## Tasks Completed

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 (RED) | Failing aria-pressed tests for HubFilterBar | 31cd0934 | HubFilterBar.test.tsx |
| 1 (GREEN) | Implement aria-pressed on filter pills | 6c60281f | HubFilterBar.tsx |
| 2 (RED) | Failing keyboard tests for GroupSidebarItem | 27c36c34 | GroupSidebar.test.tsx |
| 2 (GREEN) | Implement tabIndex + onKeyDown on GroupSidebarItem | 361e3fcc | GroupSidebar.tsx |

## What Was Built

### Task 1: aria-pressed on HubFilterBar status filter pills (GAP-135-B)

Added `aria-pressed={activeFilter === key ? 'true' : 'false'}` to each `.hub-filter__pill` button in the `FILTER_PILLS.map`. Uses string values `'true'`/`'false'` (not booleans) per WAI-ARIA spec. The pills are `<button>` elements inside a `role="group"` container, making `aria-pressed` (toggle button semantics) correct — not `aria-selected` (which is for listbox/option). The container `role="group"` and `aria-label="Session status filter"` were left unchanged.

Screen readers can now announce which status filter is active.

### Task 2: Keyboard operability for GroupSidebarItem (A11Y-02 RESEARCH Open Question 1)

Added `tabIndex={0}` and `onKeyDown` to the `GroupSidebarItem` `<li>` element. The handler activates `onGroupSelect(id)` on `Enter` or `Space`, with `e.preventDefault()` on Space to suppress scroll. Pattern copied verbatim from `SessionCard.tsx` lines 250-257. All existing attributes preserved: `role="option"`, `aria-selected`, `onClick`, `onDragOver`, `onDragLeave`, `onDrop`.

The matching `.hub__group-sidebar-item:focus-visible` focus ring was supplied by plan 135-01 (not re-added here).

## Acceptance Criteria Verification

- [x] `container.querySelector('.hub-filter__pill--active').getAttribute('aria-pressed')` === `'true'`
- [x] Every non-active `.hub-filter__pill` has `getAttribute('aria-pressed')` === `'false'`
- [x] `aria-pressed` value derived from `activeFilter === key` (not hardcoded)
- [x] `GroupSidebarItem` `<li>` has `tabIndex` 0
- [x] `keydown` `Enter` on item calls `onGroupSelect` spy with item's id
- [x] `keydown` Space on item calls `onGroupSelect` with id AND `preventDefault` invoked
- [x] `role="option"`, `aria-selected`, `onClick`, drag handlers preserved (no regression)

## Test Results

```
HubFilterBar.test.tsx: 20 passed
GroupSidebar.test.tsx: 37 passed
Full suite: 105 test files, 1731 tests, all passed
```

## Deviations from Plan

None — plan executed exactly as written.

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| Task 1 RED | 31cd0934 | PASS — 2 tests failed before implementation |
| Task 1 GREEN | 6c60281f | PASS — all 20 HubFilterBar tests pass |
| Task 2 RED | 27c36c34 | PASS — 4 tests failed before implementation |
| Task 2 GREEN | 361e3fcc | PASS — all 37 GroupSidebar tests pass |

## Known Stubs

None.

## Threat Flags

None — this plan adds DOM attributes and a keyboard event handler to existing client-side React components. No network surface, IPC, capabilities, or untrusted input paths introduced.

## Self-Check: PASSED

- [x] `frontend/src/components/Hub/HubFilterBar.tsx` — exists, contains `aria-pressed`
- [x] `frontend/src/components/Hub/GroupSidebar.tsx` — exists, contains `tabIndex={0}` and `onKeyDown`
- [x] Commits 31cd0934, 6c60281f, 27c36c34, 361e3fcc — all present in git log
- [x] Full test suite green (1731 tests)
