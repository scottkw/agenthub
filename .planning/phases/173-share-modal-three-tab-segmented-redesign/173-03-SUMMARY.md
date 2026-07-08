---
phase: 173-share-modal-three-tab-segmented-redesign
plan: 03
subsystem: ui
tags: [react, aria, tablist, accessibility, hub-share-modal]

# Dependency graph
requires:
  - phase: 173-01
    provides: ".share-segbar/.share-seg* CSS classes (dark + light theme tokens, --hub-share-seg-active-bg, is-active/is-danger states, focus-visible ring)"
provides:
  - "ShareSegmentedControl — a standalone, hand-rolled role=tablist component with roving tabindex and arrow-key navigation"
  - "Exported ShareTab id union ('tailnet' | 'internet-ro' | 'internet-fa') for the shell (plan 06) to reuse"
  - "A colorblind-safe danger cue pattern (component-owned glyph prefix + CSS ring, no hue dependency) other segmented controls in this app can copy"
affects: [173-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hand-rolled WAI-ARIA APG roving-tabindex tablist (no UI library) — first tablist precedent in this codebase"

key-files:
  created:
    - frontend/src/components/Hub/ShareSegmentedControl.tsx
    - frontend/src/components/__tests__/ShareSegmentedControl.test.tsx
  modified: []

key-decisions:
  - "Component prefixes the ⚠ glyph onto the danger tab's sub label itself (not the caller-supplied sub text) — belt-and-suspenders with the is-danger CSS ring, independent of what the shell passes"
  - "Disabled sub text is always 'N/A' per the plan's must_haves (not the DESIGN sketch's 'Off') — plan is the source of truth for behavior when it conflicts with the non-normative design sketch"

patterns-established:
  - "Roving tabindex tablist: tabIndex 0 on active tab, -1 on all others; ArrowLeft/ArrowRight computed only over the enabled-tab subset so disabled segments are skipped and the ends wrap"

requirements-completed: [SM-03, SM-07]

coverage:
  - id: D1
    description: "ShareSegmentedControl renders role=tablist with three role=tab buttons, aria-selected reflecting the active tab"
    requirement: "SM-03"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareSegmentedControl.test.tsx#ShareSegmentedControl — ARIA tablist contract (SM-03)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Roving tabindex: active tab has tabIndex=0, all others tabIndex=-1"
    requirement: "SM-07"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareSegmentedControl.test.tsx#ShareSegmentedControl — roving tabindex (SM-07)"
        status: pass
    human_judgment: false
  - id: D3
    description: "ArrowLeft/ArrowRight move selection between enabled segments, wrapping at ends and skipping disabled segments"
    requirement: "SM-07"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareSegmentedControl.test.tsx#ShareSegmentedControl — arrow-key navigation (SM-07)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Disabled segments carry aria-disabled + disabled and show 'N/A' sub text; clicking them does not select"
    requirement: "SM-07"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareSegmentedControl.test.tsx#ShareSegmentedControl — disabled segments (SM-05/SM-07)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The Full-access segment carries the is-danger class and a ⚠ glyph in its label — distinguished without hue"
    requirement: "SM-07"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareSegmentedControl.test.tsx#ShareSegmentedControl — colorblind-safe danger cue (SM-07)"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-07-08
status: complete
---

# Phase 173 Plan 03: Share Segmented Control Summary

**Hand-rolled `role="tablist"` component (`ShareSegmentedControl`) with roving tabindex, arrow-key navigation that skips disabled segments, and a component-owned colorblind-safe danger glyph — this app's first tablist implementation.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-08T07:01:00-05:00 (approx, first task commit 07:01:33)
- **Completed:** 2026-07-08T07:02:16-05:00
- **Tasks:** 2
- **Files modified:** 2 (both new)

## Accomplishments
- Built `ShareSegmentedControl.tsx`: a real `role="tablist"` with three `role="tab"` segments, `aria-selected`, roving `tabIndex` (0 active / -1 others), `aria-disabled` + `disabled` for gated tabs, and an `onKeyDown` handler that moves selection among only the enabled tabs (wraps, skips disabled) on ArrowLeft/ArrowRight
- Exported the `ShareTab` id union (`'tailnet' | 'internet-ro' | 'internet-fa'`) and a `ShareSegmentedControlTab` prop shape for plan 06's shell to consume directly
- Colorblind-safe danger cue implemented at the component level: whenever `danger` is true, the component prefixes `⚠ ` onto the sub label itself (not dependent on the shell remembering to include the glyph), pairing with the `is-danger` CSS class (inset ring, from plan 01) — never a hue-only distinction
- Wrote a 9-test a11y contract suite using the codebase's established raw `react-dom/client` + `flushSync` pattern (no `@testing-library`, matching `FunnelRiskPanel.test.tsx`'s precedent) — every assertion is role/attribute/class/text based, never computed color

## Task Commits

Each task was committed atomically:

1. **Task 1: Create ShareSegmentedControl.tsx** - `34c7015f` (feat)
2. **Task 2: ShareSegmentedControl.test.tsx — a11y + roving + arrow-nav contract** - `94e0aced` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `frontend/src/components/Hub/ShareSegmentedControl.tsx` - New role=tablist component: roving tabindex, arrow-key nav, colorblind-safe danger glyph, exports `ShareTab`/`ShareSegmentedControlTab`
- `frontend/src/components/__tests__/ShareSegmentedControl.test.tsx` - 9-test a11y/roving/arrow-nav contract suite, attribute/text/role/class assertions only

## Decisions Made
- Prefix the `⚠` glyph onto the danger tab's sub label inside the component itself, rather than trusting the shell's `tabs` array to include it in the passed `sub` string — makes the colorblind-safe cue structurally guaranteed regardless of what plan 06 passes, and avoids a double-glyph bug if the shell also includes `⚠` in its own copy.
- Used `'N/A'` for the disabled sub text exactly as specified in this plan's `must_haves.truths` and `<behavior>` block, even though the non-normative DESIGN-129.md code sketch shows `'Off'` — the plan is the authoritative behavior spec here (RESEARCH's own reference impl also used `'N/A'`), so no deviation needed, just following the plan over an inconsistent illustrative sketch.

## Deviations from Plan

None - plan executed exactly as written. Both tasks' automated verify commands passed on the first attempt (`tsc --noEmit` clean, `pnpm vitest run` 9/9 green). Ran the full frontend suite as an additional check beyond the plan's per-task verification: `tsc --noEmit` clean overall, `pnpm vitest run` 144 files / 2374 tests all passing (no regressions).

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `ShareSegmentedControl` is ready to be wired into `SessionShareModal` by plan 06, along with the `.share-seg*` CSS (plan 01) and the hoisted `CodeDisplay`/`HoldToConfirmButton` (plan 02).
- No blockers. The component's `tabs` prop shape (`{ id, main, sub, disabled?, danger? }`) and `ShareTab` union are the stable contract plan 06 should import directly rather than redefining.

---
*Phase: 173-share-modal-three-tab-segmented-redesign*
*Completed: 2026-07-08*

## Self-Check: PASSED

- FOUND: frontend/src/components/Hub/ShareSegmentedControl.tsx
- FOUND: frontend/src/components/__tests__/ShareSegmentedControl.test.tsx
- FOUND commit: 34c7015f
- FOUND commit: 94e0aced
