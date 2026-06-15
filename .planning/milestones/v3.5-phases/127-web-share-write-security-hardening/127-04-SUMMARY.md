---
phase: 127-web-share-write-security-hardening
plan: 04
subsystem: testing
tags: [playwright, e2e, csrf, security, web-share, origin-check]

# Dependency graph
requires:
  - phase: 124-web-share-write-opt-in
    provides: originAllowedForWrite middleware in capability_mw.go
  - phase: 125-write-surface-ui
    provides: files-write.spec.ts with 14-scenario cross-browser write e2e suite

provides:
  - SEC-07 CSRF Origin-mismatch e2e cell in files-write.spec.ts
  - Browser-level proof that a valid files.write cap with a mismatched Origin is rejected 403

affects: [127-SECURITY.md, 127-validation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Standalone APIRequestContext for direct HTTP assertion without browser page context"
    - "CSRF origin check tested by injecting Origin: https://evil.example.com with a valid cap"

key-files:
  created: []
  modified:
    - frontend/e2e/files-write.spec.ts

key-decisions:
  - "Use env.writeCap (VALID files.write cap) for the Origin-mismatch cell so 403 proves CSRF rejection, not a cap failure"
  - "Append as a standalone top-level test (outside the serial describe) to mirror the pattern of the existing write-api smoke test"

patterns-established:
  - "Origin-mismatch CSRF test pattern: standalone APIRequestContext + valid cap + evil.example.com Origin -> assert 403"

requirements-completed: [SEC-07]

# Metrics
duration: 15min
completed: 2026-06-14
---

# Phase 127 Plan 04: CSRF Origin-Mismatch E2E Summary

**SEC-07 e2e cell added to files-write.spec.ts: PUT with valid files.write cap and Origin: https://evil.example.com returns 403, proving CSRF rejection by originAllowedForWrite independent of cap validity**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-06-14
- **Completed:** 2026-06-14
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- Added SEC-07 CSRF Origin-mismatch test cell to `frontend/e2e/files-write.spec.ts`
- The new cell uses `env.writeCap` (a VALID files.write cap) and sends `Origin: https://evil.example.com`; the expected 403 proves `originAllowedForWrite` (capability_mw.go:187-198) rejects the request at the CSRF check, not at the capability check
- All 51 tests pass green across Chromium, Firefox, and WebKit (the new cell runs as tests 17, 34, 51 in the three-browser sweep)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add CSRF Origin-mismatch e2e scenario to files-write.spec.ts** - `d72b4f4` (test)

## Files Created/Modified

- `frontend/e2e/files-write.spec.ts` - Added SEC-07 CSRF Origin-mismatch test cell (33 lines) after the standalone write-api smoke test

## Decisions Made

- Used a top-level standalone `test(...)` cell (same pattern as the write-api smoke at line 570) rather than inserting inside the `test.describe.configure({ mode: 'serial' })` block. This keeps the new cell consistent with the spec's serial execution model while making its standalone nature explicit.
- Used `env.writeCap` explicitly -- the comment in the test makes clear that a 403 here proves CSRF rejection (Origin check), not a cap failure, closing the ambiguity gap SEC-07 identifies.

## Deviations from Plan

None - plan executed exactly as written.

The worktree required a `git merge main` before execution (worktree started from an older base before Phase 125's `files-write.spec.ts` was merged). This is expected worktree setup, not a deviation.

## Issues Encountered

None. The merge from main brought in the full `files-write.spec.ts` (Phase 125-06), the updated `fixture-env.ts` with `writeCap`, and all required Go write-surface code. The spec ran cleanly on first attempt.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SEC-07 is closed: `files-write.spec.ts` now covers viewer-with-cap writes 200, viewer-without-cap gets 403, AND a CSRF Origin-mismatch PUT is rejected with 403 even with a valid files.write cap
- The remaining Phase 127 plans (127-01, 127-02, 127-03) address denylist hardening; this plan has no blocking dependencies on them

---
*Phase: 127-web-share-write-security-hardening*
*Completed: 2026-06-14*
