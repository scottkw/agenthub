---
phase: 132-unified-grid-mini-preview-named-groups
plan: "04"
subsystem: frontend-components
tags: [hub-components, session-card, session-card-grid, mini-preview, drag-and-drop, group-menu, named-groups, tdd, vitest, CARD-07, GROUP-02, GROUP-04]
dependency_graph:
  requires:
    - frontend/src/components/Hub/MiniPreview.tsx (CARD-07 plain-text preview pane — Plan 03)
    - frontend/src/lib/hubGroups.ts (HubGroupDef, memberKey — Plan 02)
    - frontend/src/lib/hubStatus.ts (deriveHubStatus — Plan 01)
  provides:
    - frontend/src/components/Hub/SessionCard.tsx (ROW 6 MiniPreview + drag source + overflow group menu)
    - frontend/src/components/Hub/SessionCardGrid.tsx (groupByNamedGroups + Other fallback + preview/assign threading)
  affects:
    - Wave-4 HubPanel (Plan 05 — wires previewTails + groupDefs + onAssignGroup into the grid)
tech_stack:
  added: []
  patterns:
    - TDD red-green cycle for React components using createRoot + act
    - HTML5 DnD drag-source pattern (draggable + onDragStart setData + onDragEnd)
    - Controlled menu open/close with Escape keydown + click-outside via useEffect
    - Named-group grouping: Map<groupId, {label, sessions}> in definition order, Other last
    - Optional prop threading: previewTails/groupDefs/onAssignGroup flow from grid to card
key_files:
  created: []
  modified:
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/Hub/SessionCard.test.tsx
    - frontend/src/components/Hub/SessionCardGrid.tsx
    - frontend/src/components/Hub/SessionCardGrid.test.tsx
key_decisions:
  - "Menu Escape handler uses document-level keydown listener (not card-level) — fires reliably regardless of focused element inside the card"
  - "Click-outside uses mousedown (not click) to close before any nested element receives click"
  - "groupByNamedGroups is exported function (not internal) to enable direct unit testing"
  - "Nested block comment syntax error in JSDoc fixed — use plain text in JSDoc instead of embedded /* */ comments"
requirements-completed: [CARD-07, GROUP-02, GROUP-04]
duration: ~14min
completed: "2026-06-16"
---

# Phase 132 Plan 04: SessionCard + SessionCardGrid Extension Summary

SessionCard extended with ROW 6 MiniPreview (CARD-07), HTML5 drag source with memberKey payload (GROUP-02), and a keyboard-reachable overflow group menu; SessionCardGrid gains groupByNamedGroups with Other fallback and threads preview tails + assign callback to every card.

## Performance

- **Duration:** ~14 min
- **Completed:** 2026-06-16
- **Tasks:** 2 (each via TDD red-green cycle)
- **Files modified:** 4

## Accomplishments

- SessionCard gains ROW 6 MiniPreview powered by `previewLines` prop (undefined=loading, []=empty, lines=data); NO xterm instance (CARD-07)
- SessionCard is a drag source: `draggable="true"` on `<article>`, `onDragStart` sets `text/plain` to `memberKey(name, workDir)`, `effectAllowed='move'`
- `.hub-card__drag-handle` (Bars3Icon, aria-hidden) and `.hub-card__menu-btn` (EllipsisHorizontalIcon, aria-expanded, aria-haspopup="menu") added — visible via CSS hover/focus-within
- Overflow group menu (`role="menu"`) with group sub-items, "Other (default)", conditional "Remove from group" (only when session is in a named group); keyboard-closeable via Escape
- SessionCardGrid exports `groupByNamedGroups`: named groups in definition order, `__other__` last; unmatched sessions guaranteed to land in Other (GROUP-04 correctness)
- Phase 131 fallback preserved: empty/undefined `groupDefs` → `groupByWorkDir` path unchanged (all 11 existing Phase 131 assertions still green)
- All Phase 131 behaviors intact: rows 1-5, Open button (`handleOpenSessionTab` wiring), `hub-card--dim`, exit chip, origin marker

## Task Commits

Each TDD cycle committed separately (RED then GREEN):

1. **Task 1 RED: SessionCard failing tests** — `3be5f2e4` (test)
2. **Task 1 GREEN: SessionCard implementation** — `89ec944f` (feat)
3. **Task 2 RED: SessionCardGrid failing tests** — `8bfc12cd` (test)
4. **Task 2 GREEN: SessionCardGrid implementation** — `2dbd7f3d` (feat)

## TDD Gate Compliance

- RED commit (SessionCard): 3be5f2e4 — `test(132-04): add failing tests for SessionCard ROW-6 preview + drag source + group menu`
- GREEN commit (SessionCard): 89ec944f — `feat(132-04): extend SessionCard with ROW-6 MiniPreview + drag source + group menu`
- RED commit (SessionCardGrid): 8bfc12cd — `test(132-04): add failing tests for SessionCardGrid named groups + preview/assign threading`
- GREEN commit (SessionCardGrid): 2dbd7f3d — `feat(132-04): extend SessionCardGrid with groupByNamedGroups + preview/assign threading`

## Files Created/Modified

- `frontend/src/components/Hub/SessionCard.tsx` — Added imports (Bars3Icon, EllipsisHorizontalIcon, memberKey, HubGroupDef, MiniPreview); new props (previewLines, groupDefs, onAssignGroup); draggable article; drag handle + menu button; group menu with escape/click-outside; ROW 6 MiniPreview
- `frontend/src/components/Hub/SessionCard.test.tsx` — Expanded from 25 → 43 tests: Phase 131 preserved + 18 new for ROW 6, drag, menu, assign, Escape, regression guard
- `frontend/src/components/Hub/SessionCardGrid.tsx` — Added groupByNamedGroups export; new props (groupDefs, previewTails, onAssignGroup); branched render path; preview/assign threading in both paths
- `frontend/src/components/Hub/SessionCardGrid.test.tsx` — Expanded from 11 → 28 tests: Phase 131 preserved + 17 new for named grouping, fallback, Other, preview threading, assign threading, unit tests

## Verification Results

```
pnpm vitest run src/components/Hub/SessionCard.test.tsx src/components/Hub/SessionCardGrid.test.tsx
Test Files  2 passed (2)
Tests       71 passed (71)

grep -c "MiniPreview" SessionCard.tsx → 4 (at least 1)
grep -c "onOpenSession" SessionCard.tsx → 4 (at least 1 — Open button preserved)
grep -c "draggable" SessionCard.tsx → 1
grep -c "role=\"menu\"" SessionCard.tsx → 1
grep -c "groupByNamedGroups" SessionCardGrid.tsx → 3 (at least 1)
grep -c "groupByWorkDir" SessionCardGrid.tsx → 4 (at least 1 — Phase 131 fallback preserved)
grep -c "previewTails" SessionCardGrid.tsx → 4 (at least 1)
grep -rn "dangerouslySetInnerHTML" SessionCard.tsx SessionCardGrid.tsx → (empty — PASS)
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Nested block comment inside JSDoc is invalid TypeScript**
- **Found during:** Task 2 GREEN (first compile of SessionCardGrid.tsx)
- **Issue:** The plan comment `/* GROUP-04: ... */` was embedded inside a `/** ... */` JSDoc block, producing a parse error at `*/` on line 32 (`Unexpected token`)
- **Fix:** Converted the embedded `/* */` comment to plain JSDoc text (removed `/*` and `*/` markers)
- **Files modified:** `frontend/src/components/Hub/SessionCardGrid.tsx`
- **Verification:** `pnpm vitest run` compiled successfully; all 28 tests green
- **Committed in:** 2dbd7f3d (same task commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — syntax bug)
**Impact on plan:** Trivial syntax fix; no behavior change; no scope creep.

## Threat Model Compliance

| Threat ID | Mitigation Status |
|-----------|-------------------|
| T-132-10 | MITIGATED — all remote name/hostname/preview values rendered via React text children; NO dangerouslySetInnerHTML anywhere in SessionCard or SessionCardGrid |
| T-132-11 | MITIGATED — `groupByNamedGroups` guarantees every session lands in exactly one group (matched group or __other__); covered by the Other-fallback test |
| T-132-SC | N/A — Zero new npm dependencies |

## Known Stubs

None — SessionCard and SessionCardGrid are purely presentational. `previewLines` prop is undefined (loading) until HubPanel wires `usePreviewPoller` in Plan 05.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes.

## Self-Check: PASSED

- [x] frontend/src/components/Hub/SessionCard.tsx — FOUND
- [x] frontend/src/components/Hub/SessionCard.test.tsx — FOUND
- [x] frontend/src/components/Hub/SessionCardGrid.tsx — FOUND
- [x] frontend/src/components/Hub/SessionCardGrid.test.tsx — FOUND
- [x] commit 3be5f2e4 — FOUND (test 132-04 SessionCard RED)
- [x] commit 89ec944f — FOUND (feat 132-04 SessionCard GREEN)
- [x] commit 8bfc12cd — FOUND (test 132-04 SessionCardGrid RED)
- [x] commit 2dbd7f3d — FOUND (feat 132-04 SessionCardGrid GREEN)
