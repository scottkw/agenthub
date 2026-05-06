# Phase 95 Deferred Items

Items discovered during Phase 95 execution that are out-of-scope per the
"SCOPE BOUNDARY: Only auto-fix issues DIRECTLY caused by the current task's
changes" rule.

## Pre-existing TypeScript warning (Phase 94 leftover)

- **File:** `frontend/src/components/FindBar/__tests__/FindBar.animation.test.tsx`
- **Line:** 15
- **Error:** `TS6133: 'beforeEach' is declared but its value is never read.`
- **Discovered during:** Plan 95-01 Task 1 (running `pnpm tsc --noEmit` for verification)
- **Cause:** Pre-existing — present before any Phase 95 changes; unrelated to
  WebLinksConfig hand-edit on `models.ts`.
- **Disposition:** Defer to a Phase 94 cleanup or hygiene plan. Removing the
  unused import would be trivial but is outside the Plan 95-01 task surface.
