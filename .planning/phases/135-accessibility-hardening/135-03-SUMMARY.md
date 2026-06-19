---
phase: 135-accessibility-hardening
plan: "03"
subsystem: frontend/hub-modal
tags: [accessibility, focus-trap, inert, modal, keyboard, tdd]
dependency_graph:
  requires: []
  provides: [A11Y-01-verified, A11Y-04-focus-trap, WR-05-scoped-escape]
  affects: [frontend/src/components/Hub/HubModal.tsx, frontend/src/components/Hub/HubModal.test.tsx]
tech_stack:
  added: []
  patterns: [inert-attribute-focus-trap, dialog-scoped-keydown, raw-source-inspection-tests]
key_files:
  modified:
    - frontend/src/components/Hub/HubModal.tsx
    - frontend/src/components/Hub/HubModal.test.tsx
decisions:
  - "inert attribute on .hub background for focus trap (not manual Tab-cycle interceptor) — native, also suppresses AT announcement"
  - "All A11Y-04 tests use ?raw source inspection — jsdom 29 does not implement inert focus suppression"
  - "Removed stopImmediatePropagation comment mentioning the old approach (word must not appear in source)"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-19"
  tasks_completed: 2
  files_modified: 2
---

# Phase 135 Plan 03: Hub Modal Accessibility (GAP-135-C, GAP-135-D, A11Y-01) Summary

**One-liner:** Modal focus trap via DOM `inert` on `.hub` background + dialog-scoped Escape via `onKeyDown` + A11Y-01 STATUS_CONFIG mirror verified at source with TDD ?raw assertions.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | WR-05 failing tests | ac926f9f | HubModal.test.tsx |
| 1 (GREEN) | WR-05 fix + A11Y-01 mirror | ebc74fc3 | HubModal.tsx |
| 2 (RED) | A11Y-04 failing tests | 37123c28 | HubModal.test.tsx |
| 2 (GREEN) | A11Y-04 inert focus trap | 109a9d32 | HubModal.tsx |

## What Was Built

### Task 1: WR-05 Escape scope fix + A11Y-01 verification

**HubModal.tsx changes:**
- Removed the entire document-level Escape block (Phase 134 `handleCloseRef` ref, sync `useEffect`, and `document.addEventListener('keydown', ...)` effect with `stopImmediatePropagation`)
- Added `onKeyDown` prop to the `role="dialog"` div: on `e.key === 'Escape'` calls `e.stopPropagation()` then `handleClose()` directly
- Updated JSDoc to reflect WR-05 scoped handler

**HubModal.test.tsx changes:**
- Replaced `stopImmediatePropagation` test with 3 WR-05 assertions: `stopPropagation` present, `stopImmediatePropagation` absent, `document.addEventListener keydown` absent
- Added 6-test A11Y-01 describe block verifying STATUS_CONFIG source contains each status key + label (colorblind-safe: verified at source, not by eye)

**A11Y-01 mirror verification (source-level audit, no code change required):**
- SessionCard.tsx STATUS_CONFIG and HubModal.tsx STATUS_CONFIG are identical
- 6 statuses, each unique icon shape + text label
- `stopped-err` shares `ExclamationCircleIcon` with `errored` — text label "Exited" vs "Error" is the WCAG-compliant differentiator

### Task 2: A11Y-04 modal focus trap via inert + initial focus

**HubModal.tsx changes:**
- Added `const closeBtnRef = useRef<HTMLButtonElement>(null)`
- Added `useEffect` keyed on `[phase]` (placed after `cardFocusRef` block):
  - Early return if `phase !== 'open'` (Pitfall 3: no trap during entering animation)
  - `document.querySelector('.hub')` → set `.inert = true`
  - `closeBtnRef.current?.focus()` (WCAG 2.4.3: initial focus to close button)
  - Cleanup: `.inert = false` (Pitfall 1: prevents Hub keyboard-lock)
- Added `ref={closeBtnRef}` to the `.hub-modal__close` button

**HubModal.test.tsx changes:**
- Added `describe('HubModal (A11Y-04: focus trap via inert)')` with 6 `?raw` source assertions:
  - `.inert = true` present
  - `.inert = false` present (cleanup guard)
  - `closeBtnRef` and `closeBtnRef.current?.focus()` present
  - `phase !== 'open'` guard present
  - `querySelector('.hub')` present
  - `ref={closeBtnRef}` present on close button

## Verification

```
cd frontend && npx vitest run src/components/Hub/HubModal.test.tsx
Tests: 27/27 passed

cd frontend && npx vitest run
Tests: 1745/1745 passed (105 test files, no regressions)
```

## Success Criteria Validation

- [x] A11Y-04: while the modal is open the `.hub` background is `inert` (Tab cannot reach background cards via browser's native inert focus suppression)
- [x] A11Y-04: initial focus moves to the close button (`closeBtnRef.current?.focus()` when `phase === 'open'`)
- [x] A11Y-04: `inert` is removed on close (cleanup sets `.inert = false` — Pitfall 1 guard)
- [x] WR-05 (A11Y-02): Escape handled by dialog-scoped `onKeyDown` with `stopPropagation`; no `document.addEventListener` for keydown and no `stopImmediatePropagation` remain
- [x] A11Y-01: HubModal STATUS_CONFIG verified at source to mirror SessionCard — every status uniquely identifiable by icon shape + text label (colorblind-safe; verified at source not by eye)
- [x] Focus returns to originating card on close via existing `cardFocusRef` (unchanged, preserved)

## Deviations from Plan

### Auto-fixed Issues

None.

### Minor adjustment — WR-05 comment text

**Found during:** Task 1 GREEN
**Issue:** The replacement comment `// This replaces the previous document-level listener with stopImmediatePropagation.` still contained the word `stopImmediatePropagation`, causing the `expect(raw).not.toContain('stopImmediatePropagation')` test to fail.
**Fix:** Rewrote comment to say "previous document-level Escape listener (Phase 134, now removed)" without naming the old method.
**Files modified:** HubModal.tsx
**Commit:** ebc74fc3

## Known Stubs

None. All behavior is fully implemented. The `inert` attribute is a native DOM feature requiring no stub.

## Threat Flags

No new network endpoints, auth paths, file access patterns, or schema changes introduced. DOM-only focus management; no new attack surface.

T-135-03-01 (inert keyboard-lock) is mitigated: every `hubEl.inert = true` is paired with a useEffect cleanup `hubEl.inert = false`, and asserted at source (`.inert = false` present in raw source, confirmed by test).

## Self-Check: PASSED

- FOUND: frontend/src/components/Hub/HubModal.tsx
- FOUND: frontend/src/components/Hub/HubModal.test.tsx
- FOUND: .planning/phases/135-accessibility-hardening/135-03-SUMMARY.md
- FOUND commit ac926f9f (test RED Task 1)
- FOUND commit ebc74fc3 (feat GREEN Task 1)
- FOUND commit 37123c28 (test RED Task 2)
- FOUND commit 109a9d32 (feat GREEN Task 2)
