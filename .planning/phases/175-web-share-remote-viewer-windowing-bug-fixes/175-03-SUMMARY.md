---
phase: 175-web-share-remote-viewer-windowing-bug-fixes
plan: 03
subsystem: ui
tags: [xterm, react, web-share, terminalScale, css]

# Dependency graph
requires:
  - phase: 157
    provides: computeGuestScale / VIEW-04/05 guest scale pipeline (recomputeScale, isGuestRef, term.resize honor)
provides:
  - "computeGuestViewport pure helper — floor-aware guest scale with horizontal-scroll signal"
  - "DEFAULT_GUEST_MIN_SCALE readability floor constant (0.7)"
  - "TerminalPanel.recomputeScale wired to the floor + .terminal-guest--scroll-x CSS fallback"
affects: [175-07 (mobile-viewport UAT M-NN item), any future guest-viewer terminal work]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Floor-aware pure scale helper returning {scale, overflowX} instead of a bare number, consumed by a CSS class toggle rather than JS-driven layout"

key-files:
  created: []
  modified:
    - frontend/src/lib/terminalScale.ts
    - frontend/src/lib/terminalScale.test.ts
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/components/__tests__/TerminalPanel.scale.test.tsx
    - frontend/src/style.css

key-decisions:
  - "DEFAULT_GUEST_MIN_SCALE = 0.7 (~10px effective text at 14px base font) as the shared floor constant"
  - "computeGuestScale left untouched for backward compatibility; TerminalPanel.tsx switches its import to the new computeGuestViewport"
  - "Updated the pre-existing TerminalPanel.scale.test.tsx source-gate test that pinned the literal 'computeGuestScale' substring, since the plan intentionally changes which helper TerminalPanel.tsx imports"

patterns-established:
  - "Guest-only CSS fallback class toggled imperatively from a useCallback via container.classList.toggle(), guarded by isGuestRef.current"

requirements-completed: [BUG-01]

coverage:
  - id: D1
    description: "computeGuestViewport pure function: above-floor natural scale, below-floor clamp+overflowX, upscale cap at 1.0, zero/negative-dim guard, default floor constant fallback"
    requirement: "BUG-01"
    verification:
      - kind: unit
        ref: "frontend/src/lib/terminalScale.test.ts#computeGuestViewport — BUG-01 floor-aware guest scale (9 tests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "recomputeScale wires computeGuestViewport into the guest transform + toggles terminal-guest--scroll-x class; host path never toggles the class or sends resize"
    requirement: "BUG-01"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/TerminalPanel.scale.test.tsx#guest: narrow container clamps to the readability floor and adds the scroll-x class"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/TerminalPanel.scale.test.tsx#guest: wide container caps at scale(1) with no scroll-x class"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/TerminalPanel.scale.test.tsx#host: never gains the terminal-guest--scroll-x class"
        status: pass
    human_judgment: false
  - id: D3
    description: "CSS .terminal-guest--scroll-x enables overflow-x: auto + touch-action: pan-x pan-y without disturbing the base container's vertical overflow:hidden / pan-y"
    requirement: "BUG-01"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/TerminalPanel.scale.test.tsx#style.css contains .terminal-guest--scroll-x with overflow-x: auto"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live mobile-viewport visual confirmation (narrow phone actually shows legible scrolling text) — deferred to 175-07's new M-NN item per RESEARCH Bug 1 Testability (not unit-assertable)"
    requirement: "BUG-01"
    verification: []
    human_judgment: true
    rationale: "Real-device/narrow-viewport visual legibility cannot be asserted from jsdom (clientWidth/clientHeight are always 0); RESEARCH explicitly defers this to a manual UAT item in 175-07."

# Metrics
duration: 8min
completed: 2026-07-08
status: complete
---

# Phase 175 Plan 03: Guest Terminal Readability Floor Summary

**Added a floor-aware guest-viewport scale helper (`computeGuestViewport`) and wired `TerminalPanel.recomputeScale` to clamp at a readability floor (0.7) with a `.terminal-guest--scroll-x` horizontal-scroll CSS fallback, fixing BUG-01 (#128) where narrow-phone web-share guests saw the 80-col grid shrink to unreadable text with no floor.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-08T18:37:00Z (approx)
- **Completed:** 2026-07-08T18:43:27Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- New pure helper `computeGuestViewport(containerW, containerH, gridW, gridH, minScale)` in `terminalScale.ts`: returns `{ scale, overflowX }`, clamping at `minScale` (default `DEFAULT_GUEST_MIN_SCALE = 0.7`) instead of shrinking further, while preserving the existing never-upscale (`≤1`) invariant and divide-by-zero guard. `computeGuestScale` (the original VIEW-05 helper) is untouched — all 10 pre-existing tests still pass.
- `TerminalPanel.recomputeScale` now calls `computeGuestViewport` and toggles a new `terminal-guest--scroll-x` class on the guest container when `overflowX` is true; the class is guest-only (`isGuestRef.current` guard). No `sendResize` is emitted from this path (D-03 invariant preserved).
- New CSS rule `.terminal-guest--scroll-x` in `style.css`: `overflow-x: auto` + `touch-action: pan-x pan-y`, layered over the base `.terminal-session-container` (`overflow: hidden`, `touch-action: pan-y`) so only the horizontal axis changes — vertical scroll continues to be handled by xterm's internal viewport.
- Full frontend test suite (2402 tests, 147 files) passes; `tsc --noEmit` clean; `pnpm build` succeeds.

## Task Commits

Each task was committed atomically (TDD RED→GREEN cycle per task):

1. **Task 1: Add a floor-aware guest-viewport pure function (RED → GREEN)** - `0ba4e579` (feat)
2. **Task 2: Wire recomputeScale to the floor + horizontal-scroll fallback** - `775fb8c6` (feat)

_Note: Each task's test-then-implementation change was committed as a single `feat` commit per the plan's `tdd="true"` tasks — tests and implementation were written and verified together (RED confirmed via `vitest run` before implementing, GREEN confirmed after) prior to each commit, matching this repo's existing TDD-task commit convention on this plan (single commit per task, not split test/feat commits)._

## Files Created/Modified
- `frontend/src/lib/terminalScale.ts` - added `computeGuestViewport` + `DEFAULT_GUEST_MIN_SCALE`; `computeGuestScale` untouched
- `frontend/src/lib/terminalScale.test.ts` - 9 new tests for `computeGuestViewport` (above-floor, below-floor, upscale-cap, exact-floor-boundary, zero/negative-dim guard, default-floor-constant fallback)
- `frontend/src/components/TerminalPanel.tsx` - `recomputeScale` switched from `computeGuestScale` to `computeGuestViewport`; toggles `.terminal-guest--scroll-x` on the guest container via `container.classList.toggle(..., overflowX)`
- `frontend/src/components/__tests__/TerminalPanel.scale.test.tsx` - updated the source-gate test that pinned the `computeGuestScale` import literal (now pins `computeGuestViewport`); added narrow-viewport (floor-clamp + scroll-x class), wide-viewport (scale(1), no scroll-x class), host-path (never gains scroll-x class) behavioral tests, and a CSS-gate test for `.terminal-guest--scroll-x`
- `frontend/src/style.css` - new `.terminal-guest--scroll-x` rule colocated with the existing `.terminal-session-container` / VIEW-05 guest-scale CSS

## Decisions Made
- `DEFAULT_GUEST_MIN_SCALE = 0.7` chosen as the shared floor constant (~10px effective text at the 14px base font size), exported from `terminalScale.ts` as the single source of truth for both the caller (`TerminalPanel.tsx`) and the tests.
- Left `computeGuestScale` untouched in `terminalScale.ts` for backward compatibility (per plan instruction), but `TerminalPanel.tsx` fully switches its import to `computeGuestViewport` — this is a real behavior change (not additive), so the pre-existing `TerminalPanel.scale.test.tsx` source-gate test that asserted the literal `computeGuestScale` substring was intentionally updated to assert `computeGuestViewport` instead, matching the plan's explicit "recomputeScale consumes the floor-aware helper" behavior spec.
- The `.terminal-guest--scroll-x` class is toggled inside `recomputeScale` guarded by `isGuestRef.current`, even though `recomputeScale` is in practice only ever invoked on the guest path — this makes the guest-only invariant explicit in the code rather than implicit from call-site discipline, per the plan's threat-model mitigation (T-175-03-01).

## Deviations from Plan

None - plan executed exactly as written. Both tasks matched their `behavior`/`action`/`verify`/`acceptance_criteria` blocks with no auto-fixes required.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The floor-aware guest scale pipeline is code-complete and unit-tested at both the pure-function layer and the component-integration layer (host/guest CSS class isolation proven).
- **Live/manual verification is still required**: an actual narrow-phone-viewport visual check (the terminal stays legible and scrolls horizontally below the floor) cannot be asserted from jsdom (`clientWidth`/`clientHeight` are always 0 there) — this is explicitly deferred to 175-07's new M-NN manual UAT item per the plan's `<verification>` block and RESEARCH's "Bug 1 Testability" note.
- No blockers for subsequent 175-xx plans; `terminalScale.ts` and `TerminalPanel.tsx` are otherwise unchanged in structure, so other plans touching those files are unaffected.

---
*Phase: 175-web-share-remote-viewer-windowing-bug-fixes*
*Completed: 2026-07-08*

## Self-Check: PASSED

All 5 declared files_modified exist on disk; all 3 commits (`0ba4e579`, `775fb8c6`, `7b3f9413`) verified present in `git log`.
