---
phase: 141-redesign-implementation
plan: "03"
subsystem: ui
tags: [css, tokens, design-system, file-browser, settings, dark-mode, light-mode]

# Dependency graph
requires:
  - phase: 141-01
    provides: --hub-* token declarations including --hub-text-dim in :root and [data-ui-theme="light"] blocks
  - phase: 141-02
    provides: S-01..S-03 surface migration (Hub main, tab strip, terminal) complete
  - phase: 141-05
    provides: GroupSidebar/StatusBar test suites passing (plan 01 RED suites turned GREEN)
provides:
  - File Browser + Editor chrome (S-04/S-05) fully tokenized on --hub-* tokens
  - Settings panel, jump-bar, search, toggles (S-06) fully tokenized on --hub-* tokens
  - Motion-guarded transition blocks for both surfaces
  - link-confirm-popover btn variants tokenized (tail of 2646-range)
affects: [141-04, 142-regression-tests]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Motion guard pattern: transitions declared inside @media (prefers-reduced-motion: no-preference) with transition:none in reduce block"
    - "Toggle thumb off-state reuses --hub-scrollbar-hover (no new token) per planner decision"
    - "Shimmer keyframe uses token vars instead of hardcoded hex"

key-files:
  created: []
  modified:
    - frontend/src/style.css

key-decisions:
  - "Toggle thumb off-state uses var(--hub-scrollbar-hover) — no --hub-toggle-thumb-off introduced per plan 01 decision"
  - "file-browser__divider and col-divider transition moved inside motion guard (removed from static declaration)"
  - "link-confirm-popover__btn--cancel/continue tokenized as they fall within the 2646-3380 gated range"

patterns-established:
  - "Skeleton shimmer keyframe uses CSS custom properties (surface-elevated → border) for theme adaptability"
  - "Motion guard blocks placed at end of surface section, grouped by selector type"

requirements-completed: [RDS-02, RDS-04]

# Metrics
duration: 35min
completed: 2026-06-21
---

# Phase 141 Plan 03: File Browser + Settings Token Migration Summary

**File Browser (S-04/S-05) and Settings (S-06) migrated from hardcoded TokyoNight hex to --hub-* CSS custom properties, enabling light/dark theme toggle across both surfaces**

## Performance

- **Duration:** 35 min
- **Started:** 2026-06-21T11:20:00Z
- **Completed:** 2026-06-21T11:55:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- All .file-browser__* selectors (breadcrumb, status line, file list, col headers, dividers, preview pane, buttons, takeover/empty/error states) consume --hub-* tokens
- All .settings-panel__*, .settings-jump-bar__*, .settings-search__* selectors consume --hub-* tokens
- Hover/focus transitions for both surfaces wrapped in prefers-reduced-motion motion guard blocks
- No new tokens introduced beyond plan 01's --hub-text-dim
- Full frontend gate passed: 106 test files, 1737 tests, tsc --noEmit clean

## Task Commits

Each task was committed atomically:

1. **Task 1: Migrate File Browser + Editor chrome (2646–3380) to tokens** - `3efffd5e` (feat)
2. **Task 2: Migrate Settings panel/jump-bar/search/toggles (370–771) to tokens** - `d00d1f17` (feat)

## Files Created/Modified

- `frontend/src/style.css` - File browser (S-04/S-05) and settings (S-06) selectors migrated to --hub-* tokens; motion guard blocks added for both surfaces

## Decisions Made

- Toggle thumb off-state (`#565f89`) mapped to `var(--hub-scrollbar-hover)` — same dim-slate value, no new `--hub-toggle-thumb-off` token introduced (per plan 01 planner decision)
- `file-browser__divider` and `file-browser__col-divider` had `transition: background 120ms ease` removed from static declarations and placed inside the motion guard block
- `link-confirm-popover__btn--cancel` and `__btn--continue` tokenized as they fell within the 2646-3380 gated range (plan verification gate enforced)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Tokenized link-confirm-popover btn variants within gated range**
- **Found during:** Task 1 verification gate
- **Issue:** `.link-confirm-popover__btn--cancel` (border-color: #565f89, color: #c0caf5) and `.link-confirm-popover__btn--continue` (background: #7aa2f7, color: #1a1b26) are at lines 2658-2667 — inside the 2646-3380 gated range — causing the acceptance gate to fail
- **Fix:** Migrated to var(--hub-scrollbar-hover), var(--hub-text-primary), var(--hub-accent), var(--hub-bg)
- **Files modified:** frontend/src/style.css
- **Verification:** Gate re-run — PASS
- **Committed in:** 3efffd5e (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 3 — blocking gate)
**Impact on plan:** Necessary to satisfy the plan's own acceptance gate. The link-confirm-popover body (lines 2604-2645) has additional hex values outside the gated range that were not touched (out of scope per deviation scope boundary rule).

## Issues Encountered

None beyond the link-confirm-popover gate fix above.

## Known Stubs

None — this plan is CSS-only token migration with no data-flow changes.

## Threat Flags

None — recolor/token migration only; no change to data handling, auth, or network surfaces.

## Self-Check

- [x] `3efffd5e` exists: `git log --oneline --all | grep 3efffd5e` confirms
- [x] `d00d1f17` exists: `git log --oneline --all | grep d00d1f17` confirms
- [x] `frontend/src/style.css` modified: confirmed
- [x] Task 1 gate: `! sed -n '2646,3380p' ... | grep -E '#...' | grep -v 'rgba(0,' | grep -q .` exits 0 — PASS
- [x] Task 2 gate: `! sed -n '370,771p' ... | grep -E '#...' | grep -v 'rgba(0,' | grep -q .` exits 0 — PASS
- [x] Tests: 106/106 files, 1737/1737 tests passed
- [x] tsc --noEmit: clean (no output)

## Self-Check: PASSED

## Next Phase Readiness

- S-04, S-05, S-06 fully tokenized — ready for light theme UAT across File Browser and Settings
- Plan 04 (Share Modal new CSS) can proceed independently
- No blockers

---
*Phase: 141-redesign-implementation*
*Completed: 2026-06-21*
