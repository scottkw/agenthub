---
phase: 172-hub-card-layout-badge-refinement
fixed_at: 2026-07-08T05:16:30Z
review_path: .planning/phases/172-hub-card-layout-badge-refinement/172-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 172: Code Review Fix Report

**Fixed at:** 2026-07-08T05:16:30Z
**Source review:** .planning/phases/172-hub-card-layout-badge-refinement/172-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 2 (Warning tier; 3 Info findings out of scope for `critical_warning`)
- Fixed: 2
- Skipped: 0

## Fixed Issues

### WR-01: `timeText` still uses the raw `hostname` heuristic that CARD-02 deprecated

**Files modified:** `frontend/src/components/Hub/SessionCard.tsx`
**Commit:** 7a812120
**Status:** fixed: requires human verification
**Applied fix:** Changed the `timeText` leading condition from the raw
`hostname && hostname !== ''` check to the provenance-aware `!isLocal` signal, so uptime/
duration display is now driven by the same `isRemote`/`isLocal` model as the origin chip,
aria-label, and Connected/Available meta item. This closes the latent inconsistency where a
local session carrying a non-empty `os.Hostname()` rendered a "Local" origin chip but silently
dropped its uptime.

**Human-verification note:** This is a conditional/logic change. The current test fixtures use
`hostname: ''` for local sessions, so the specific regressed case (local session WITH a non-empty
hostname) is not directly exercised by the suite. TypeScript typecheck passes and all 94
SessionCard tests pass, but a human should confirm the intended behavior for a
local-session-with-hostname before final sign-off.

### WR-02: `.hub-card__meta` reserves an empty vertical strip when `metaItems` is empty

**Files modified:** `frontend/src/components/Hub/SessionCard.tsx`
**Commit:** 6c0c21ad
**Status:** fixed
**Applied fix:** Wrapped the `.hub-card__meta` div in a `{metaItems.length > 0 && (...)}` guard
so the wrapper (which carries `min-height: 14px; margin-bottom: 10px`) is no longer rendered when
there are no meta items, eliminating the ~24px empty strip regression in vertical rhythm. Added
an explanatory comment referencing WR-02.

## Skipped Issues

None. All in-scope (Warning-tier) findings were fixed. IN-01, IN-02, and IN-03 are Info-tier and
outside the `critical_warning` fix scope.

## Verification

- **TypeScript:** `frontend/node_modules/.bin/tsc --noEmit` → exit 0 (clean, no errors).
- **Vitest:** `vitest run src/components/Hub/SessionCard.test.tsx src/components/__tests__/SessionCard.share.test.tsx`
  → 2 files passed, 94 tests passed.
- Both checks were run inside the isolated review-fix worktree against the fixed source.

---

_Fixed: 2026-07-08T05:16:30Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
