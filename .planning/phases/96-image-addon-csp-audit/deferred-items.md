# Deferred Items — Phase 96

## Out-of-Scope Findings

### TS6133 in FindBar.animation.test.tsx (pre-existing, not Phase 96)

- **File:** `frontend/src/components/FindBar/__tests__/FindBar.animation.test.tsx:15`
- **Error:** `'beforeEach' is declared but its value is never read.`
- **Origin:** Phase 94 commit `f9e6d90` (`test(94-06): add failing RED tests for FindBar slide-in animation wiring`).
- **Found during:** Plan 96-01 Task 1 `pnpm tsc --noEmit` verification.
- **Disposition:** Out of scope for Phase 96. Pre-existing on Phase 96 base (`cbfa565`); reproduced before any Plan 96-01 edits via `git stash` round-trip. Should be cleaned up by a future Phase 94 follow-up or general housekeeping. Phase 96 verification gates that depend on `pnpm tsc --noEmit` exiting 0 are blocked by this single unrelated unused-import warning, not by Phase 96 code.
