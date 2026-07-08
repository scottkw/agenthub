---
phase: 173-share-modal-three-tab-segmented-redesign
plan: 02
subsystem: ui
tags: [react, vitest, share-modal, accessibility, prefers-reduced-motion, refactor]

# Dependency graph
requires:
  - phase: 173-share-modal-three-tab-segmented-redesign
    provides: "173-01 CSS foundation (.share-segbar, .share-linkcard*, .hub-share-modal__tabpanel classes in style.css)"
provides:
  - "Exported CodeDisplay + HoldToConfirmButton in frontend/src/components/SessionShare/shared.tsx — single source of truth for the two components the upcoming TailnetTab / InternetReadOnlyTab / InternetFullAccessTab / ShareLinkCard components will reuse (D-09)"
  - "prefers-reduced-motion plain-confirm fallback on HoldToConfirmButton (SM-07/D-07) — additive, does not alter the 3s hold-gate safety contract"
affects: [173-03, 173-04, 173-05, 173-06, 173-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared component hoisting: private components used only inside one file get promoted to a named-export module (SessionShare/shared.tsx) when multiple sibling components need them, instead of duplicating"
    - "prefers-reduced-motion runtime branch: window.matchMedia('(prefers-reduced-motion: reduce)').matches read once per render, guarded with typeof checks for non-browser/test environments (matches existing SessionShareModal.tsx pattern)"

key-files:
  created:
    - frontend/src/components/SessionShare/shared.tsx
    - frontend/src/components/__tests__/HoldToConfirmButton.test.tsx
  modified:
    - frontend/src/components/SessionSharePanel.tsx

key-decisions:
  - "D-07 (reduced-motion fallback) and D-09 (preserve HoldToConfirmButton as-is) are compatible: D-09 governs the SAFETY GATE CONTRACT (3s hold + SetSessionFunnelWrite semantics unchanged on the non-reduced path); D-07's fallback is an ADDITIVE branch that still requires a deliberate click to arm public write — resolves the tension flagged in 173-RESEARCH.md Pitfall 1"
  - "formatCountdown was NOT moved to shared.tsx — it is used directly by SessionSharePanel's own JSX (countdown display), not by either hoisted component, so it stayed in SessionSharePanel.tsx per the plan's explicit two-component scope"

patterns-established:
  - "New per-tab components (173-03+) import CodeDisplay/HoldToConfirmButton from '../SessionShare/shared' (or relative equivalent) rather than redefining them"

requirements-completed: [SM-07, SM-08]

coverage:
  - id: D1
    description: "CodeDisplay and HoldToConfirmButton hoisted into an exported shared.tsx module; SessionSharePanel.tsx repointed to import them (no local duplicate definitions)"
    requirement: "SM-08"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionSharePanel.test.tsx (31 tests, unchanged, still pass)"
        status: pass
      - kind: other
        ref: "pnpm build (vite build + esbuild) succeeds"
        status: pass
    human_judgment: false
  - id: D2
    description: "HoldToConfirmButton renders a plain single-click confirm (no timed fill) under prefers-reduced-motion: reduce, and keeps the unchanged 3s timed hold otherwise; disabled prop fully disables both paths"
    requirement: "SM-07"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/HoldToConfirmButton.test.tsx (5 tests: reduced-motion confirm/disabled, non-reduced hold-fill/disabled/full-hold-completion)"
        status: pass
    human_judgment: false

duration: 3min
completed: 2026-07-08
status: complete
---

# Phase 173 Plan 02: Hoist Shared Share Components + Reduced-Motion Fallback Summary

**Hoisted CodeDisplay/HoldToConfirmButton into an exported `SessionShare/shared.tsx` module (zero behavior change) and added an additive `prefers-reduced-motion` plain-confirm fallback to the hold button, both proven against the full existing test suite (2365 tests green).**

## Performance

- **Duration:** ~3 min (git commit timestamps 06:52:49 → 06:54:06)
- **Started:** 2026-07-08T11:52:00Z
- **Completed:** 2026-07-08T11:55:02Z
- **Tasks:** 2 completed (Task 2 followed TDD RED/GREEN)
- **Files modified:** 3 (1 created, 1 modified in Task 1; 1 test file created + 1 modified in Task 2)

## Accomplishments
- Created `frontend/src/components/SessionShare/shared.tsx` exporting `CodeDisplay` and `HoldToConfirmButton` verbatim (including `HOLD_DURATION_MS`/`HOLD_TICK_MS` constants) — single source of truth for the upcoming per-tab components (TailnetTab, InternetReadOnlyTab, InternetFullAccessTab) and ShareLinkCard
- Repointed `SessionSharePanel.tsx` to import both components from the new shared module; removed the local private definitions
- Added a `prefers-reduced-motion: reduce` branch to `HoldToConfirmButton`: renders a plain `<button>` with a "Confirm" label, no hold-fill span, no `setInterval` — a single click fires `onConfirm` exactly once, `disabled` still fully gates it
- Verified the non-reduced-motion 3s hold path is byte-behavior unchanged (existing `SessionSharePanel.test.tsx` R1 hold-gate assertions still pass unmodified)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create shared.tsx, repoint SessionSharePanel imports** - `af8a7879` (refactor)
2. **Task 2: Add prefers-reduced-motion fallback (TDD)** - `c4059c6c` (test, RED) → `836621f8` (feat, GREEN)

**Plan metadata:** (this commit, docs: complete plan)

_Note: Task 2 used TDD — RED commit `c4059c6c` (failing test asserting the reduced-motion branch), then GREEN commit `836621f8` (implementation). No REFACTOR commit was needed — the implementation was minimal and clean on first pass._

## Files Created/Modified
- `frontend/src/components/SessionShare/shared.tsx` - New shared module exporting `CodeDisplay` and `HoldToConfirmButton`, including the reduced-motion fallback branch
- `frontend/src/components/SessionSharePanel.tsx` - Removed local `HoldToConfirmButton`/`CodeDisplay` definitions; imports them from `./SessionShare/shared`; kept `formatCountdown` (not part of the hoisted scope)
- `frontend/src/components/__tests__/HoldToConfirmButton.test.tsx` - New test file: 5 tests covering reduced-motion confirm/disabled and non-reduced-motion hold-fill/disabled/full-hold-completion paths

## Decisions Made
- Resolved the D-07 vs D-09 tension per the plan's stated decision: D-09 protects the safety-gate TIMING CONTRACT (3s hold, `SetSessionFunnelWrite` semantics) on the non-reduced path; D-07's reduced-motion fallback is additive and still requires a deliberate click — both decisions are satisfied simultaneously.
- Did not move `formatCountdown` into `shared.tsx` — it's a standalone helper used only by `SessionSharePanel`'s own countdown JSX, not by either hoisted component, so moving it was out of this plan's explicit scope (avoids unrelated file churn).
- Reduced-motion detection re-checks `window.matchMedia` on every render (no memoization) — matches the existing pattern already used in `SessionShareModal.tsx`, keeping the codebase consistent rather than introducing a new pattern for one component.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `shared.tsx` is ready to be imported by 173-03's new per-tab components (TailnetTab / InternetReadOnlyTab / InternetFullAccessTab) and ShareLinkCard without any further changes.
- The reduced-motion fallback is proven at the component level; no visual/CSS work was needed in this plan (style.css already had the `transition: none` CSS-only reduced-motion rule from a prior phase, which remains valid alongside the new behavioral branch since the reduced-motion render path omits the hold-fill element entirely).
- Full frontend suite (143 test files / 2365 tests) passes; `tsc --noEmit` clean; `pnpm build` succeeds — no regressions carried forward.

---
*Phase: 173-share-modal-three-tab-segmented-redesign*
*Completed: 2026-07-08*

## Self-Check: PASSED

- FOUND: frontend/src/components/SessionShare/shared.tsx
- FOUND: frontend/src/components/__tests__/HoldToConfirmButton.test.tsx
- FOUND: frontend/src/components/SessionSharePanel.tsx
- FOUND: af8a7879 (refactor commit)
- FOUND: c4059c6c (test/RED commit)
- FOUND: 836621f8 (feat/GREEN commit)
