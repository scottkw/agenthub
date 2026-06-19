---
phase: 133-attention-pulse
plan: "04"
subsystem: frontend/components/Hub
tags: [tdd, attn-06, group-sidebar, collapsed-badge, bug-fix, inversion-fix]
dependency_graph:
  requires: [133-01 (isAttentionStatus predicate), 133-02 (attn badge CSS)]
  provides: [GroupCounts.attention field, collapsed-group attention badge render, inverted-badge condition fix]
  affects:
    - frontend/src/components/Hub/GroupSidebar.tsx
    - frontend/src/components/Hub/GroupSidebar.test.tsx
tech_stack:
  added: []
  patterns: [TDD RED/GREEN, vitest unit test, BEM badge pattern]
key_files:
  created: []
  modified:
    - frontend/src/components/Hub/GroupSidebar.tsx
    - frontend/src/components/Hub/GroupSidebar.test.tsx
decisions:
  - "GroupCounts.attention is the superset of waiting — computed via isAttentionStatus(deriveHubStatus(s))"
  - "Attention badge replaces needs-input badge when attnCount>0; needs-input only shows when attention===0 AND waiting>0"
  - "Existing needs-input badge tests updated from collapsed=false to collapsed=true to reflect corrected (fixed) behavior"
  - "BellAlertIcon has no Tailwind sizing — sized via .hub__group-sidebar-item__attn-badge svg CSS rule (Plan 02)"
metrics:
  duration: ~3 minutes
  completed: 2026-06-17
---

# Phase 133 Plan 04: Collapsed-Group Attention Badge Summary

## One-Liner

Added `GroupCounts.attention` field (superset of waiting via `isAttentionStatus`), rendered `BellAlertIcon + count` attention badge on collapsed sidebar items (ATTN-06), and fixed the Phase 132 badge-condition inversion bug (`!collapsed` → `collapsed`) in GroupSidebar.tsx.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 RED | Add failing tests for ATTN-06 attention badge | d06f55f0 | GroupSidebar.test.tsx (137 lines added) |
| 2 GREEN | Extend counts, add attention badge, fix collapsed condition | c7afc826 | GroupSidebar.tsx + GroupSidebar.test.tsx (63 lines added, 24 removed) |

## Verification

- `grep -c "attention" frontend/src/components/Hub/GroupSidebar.tsx` → 12 (≥5 required) ✓
- `grep -c "isAttentionStatus" frontend/src/components/Hub/GroupSidebar.tsx` → 3 (≥2 required) ✓
- `grep -c "hub__group-sidebar-item__attn-badge" frontend/src/components/Hub/GroupSidebar.tsx` → 3 (≥2 required) ✓
- `grep -c "collapsed && counts.attention" frontend/src/components/Hub/GroupSidebar.tsx` → 2 ✓
- `grep -c '!collapsed && counts.waiting' frontend/src/components/Hub/GroupSidebar.tsx` → 0 ✓
- No `w-3`/`h-4`/`w-4` added to new BellAlertIcon ✓
- `cd frontend && pnpm test -- --run src/components/Hub/GroupSidebar.test.tsx` → 33 passed ✓
- `cd frontend && pnpm tsc --noEmit` → no errors ✓

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated existing needs-input badge tests to use `collapsed: true`**
- **Found during:** Task 2 implementation
- **Issue:** The 5 existing needs-input badge tests rendered with `collapsed=false` (default) and expected to see the badge — they were testing the Phase 132 inverted-condition bug. After fixing `!collapsed` → `collapsed`, those tests would have failed because badges now correctly only show when collapsed.
- **Fix:** Updated the 5 existing tests to reflect the corrected behavior: added `collapsed: true` where testing badge presence, and updated aria-label assertions to reflect that `waiting` sessions trigger the attn-badge (not needs-input-badge), since `waiting` IS an attention status.
- **Files modified:** `frontend/src/components/Hub/GroupSidebar.test.tsx`
- **Commit:** c7afc826

**2. [Rule 2 - Missing critical functionality] isAttentionStatus imported in both count functions**
- The plan specified adding it in 2 places; the final implementation adds it in 3 places (computeCounts, computeGlobalCounts, and the import line). This is correct — the import is counted as a reference in the grep output.

## TDD Gate Compliance

- RED gate: commit `d06f55f0` — `test(133-04): add failing tests for ATTN-06 attention badge (RED)` (6 failing cases confirmed)
- GREEN gate: commit `c7afc826` — `feat(133-04): extend GroupCounts with attention + add collapsed attn badge (GREEN)` (33 passing cases)

## Phase 132 Regression Guard

All Phase 132 features preserved:
- Group list with "All" item ✓
- Running/total count display ✓
- Drag-over drop targets ✓
- Create-group inline input ✓
- Collapse toggle ✓
- Existing 27 Phase 132 assertions pass ✓
- `NeedsInputBadge` component reused unchanged ✓

## Known Stubs

None — all attention badge logic is wired to the real `isAttentionStatus` predicate from Plan 01.

## Threat Flags

None — presentational change only; counts derived from existing session status; no new data ingress, RPC, input, or storage.

## Self-Check: PASSED

- `frontend/src/components/Hub/GroupSidebar.tsx` modified with attention field + badge render ✓
- `frontend/src/components/Hub/GroupSidebar.test.tsx` modified with 6 new passing test cases ✓
- Commits d06f55f0 and c7afc826 exist in git log ✓
- tsc --noEmit clean ✓
- All 33 tests pass ✓
