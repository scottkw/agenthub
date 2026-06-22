---
phase: 143-regression-test-program
plan: "01"
subsystem: frontend-tests
tags: [vitest, regression, gap-closure, css-contract, pure-function, component-render]
dependency_graph:
  requires: []
  provides:
    - frontend/src/lib/hubGroupCounts.test.ts (GAP-01 coverage)
    - frontend/src/lib/agentBadge.test.ts (GAP-02 coverage)
    - frontend/src/components/__tests__/Sidebar.test.tsx (GAP-03 coverage added)
    - frontend/src/components/__tests__/style.hub.test.ts (GAP-04 coverage added)
  affects:
    - .planning/phases/143-regression-test-program/143-03-PLAN.md (traceability table cites these paths)
    - .planning/phases/143-regression-test-program/143-02-PLAN.md (path-check validation)
tech_stack:
  added: []
  patterns:
    - vitest pure-function unit test (hubStatus.test.ts analog)
    - component render + cleanup with createRoot/act (Sidebar.test.tsx extension)
    - CSS source-gate with readFileSync + indexOf/slice (style.hub.test.ts extension)
key_files:
  created:
    - frontend/src/lib/hubGroupCounts.test.ts
    - frontend/src/lib/agentBadge.test.ts
  modified:
    - frontend/src/components/__tests__/Sidebar.test.tsx
    - frontend/src/components/__tests__/style.hub.test.ts
decisions:
  - "Followed hubStatus.test.ts analog exactly: import only {describe,it,expect} from vitest, one describe per function, toBe/toEqual assertions, no mocks"
  - "SessionInfo fixtures built as minimal object literals with only fields needed for deriveHubStatus (state, status, exitCode) and computeCounts (name, workDir)"
  - "GAP-03 describe placed before the existing POL-05 describe block (not inside it) — the new block proves the positive 3-item contract WHEN groups are present; POL-05 tests the full group sub-list behavior"
  - "GAP-04 CSS tokens: all exact string literals verified in style.css at the line numbers cited in the plan interfaces block before asserting; no placeholder class names used"
  - "hub-card__preview height assertion uses indexOf('.hub-card__preview {') + slice to scope the block, ruling out false positives from other rules"
metrics:
  duration_minutes: 12
  completed_date: "2026-06-22"
  tasks_completed: 3
  tasks_total: 3
  files_changed: 4
---

# Phase 143 Plan 01: Close v4.0 Coverage Gaps (GAP-01..04) Summary

**One-liner:** Four v4.0 vitest gaps closed — pure-function count/badge tests (GAP-01/02), sidebar 3-item contract with groups (GAP-03), Phase 142 comp-fidelity CSS token pinning (GAP-04).

## What Was Built

Closed the four v4.0 release-critical automated coverage gaps identified in RESEARCH.md:

**GAP-01 — computeCounts / computeGlobalCounts (`hubGroupCounts.test.ts`):** New vitest file with 11 tests covering empty list, running/idle/waiting/stopped-err status routing, multi-session accumulation, and membership exclusion via `memberKey` Set. Verified that `total === sessions.length` for global counts and membership filtering works correctly for `computeCounts`.

**GAP-02 — agentBadgeModifier (`agentBadge.test.ts`):** New vitest file with 14 tests covering all 6 known agents (claude, opencode, codex, gemini, cursor, aider), all 5 shell variants (shell, bash, zsh, pwsh, powershell) collapsing to `'shell'`, empty string, and the null fallback for unknown CLIs.

**GAP-03 — 3-item sidebar contract with groups (Sidebar.test.tsx extension):** New `describe('NAV-05 positive render contract — 3 items with groups present (GAP-03)')` block proving that `button.sidebar__item` count stays exactly 3 when groupDefs are supplied, group entries render as `.sidebar__group-item` (not top-level nav), and no Sessions/Remote buttons appear regardless of groupDefs.

**GAP-04 — Phase 142 comp-fidelity CSS tokens (style.hub.test.ts extension):** New `describe('Phase 142 comp-fidelity CSS tokens (GAP-04 anti-regression)')` block pinning: (a) `border-radius: 16px` in `.hub-card` block, (b) all 8 `data-agent` spine rules with exact hex literals, (c) `.hub-card__badge` base rule and claude chip tint `color: #7aa2f7`, (d) `.hub-card__preview` declares `height: 150px` (not min-height).

## Test File Paths (repo-relative — for Plan 03 traceability table)

| File | Gap | Coverage |
|------|-----|----------|
| `frontend/src/lib/hubGroupCounts.test.ts` | GAP-01 | computeCounts, computeGlobalCounts |
| `frontend/src/lib/agentBadge.test.ts` | GAP-02 | agentBadgeModifier (all CLI inputs) |
| `frontend/src/components/__tests__/Sidebar.test.tsx` | GAP-03 | 3-item contract with groupDefs present |
| `frontend/src/components/__tests__/style.hub.test.ts` | GAP-04 | Phase 142 comp-fidelity CSS tokens |

## Commits

| Task | Description | Hash |
|------|-------------|------|
| 1 | GAP-01 hubGroupCounts + GAP-02 agentBadge tests | 1a8d06e1 |
| 2 | GAP-03 sidebar 3-item contract with groups | 6ec02016 |
| 3 | GAP-04 Phase 142 CSS token anti-regression | 8482606e |

## Verification

Full vitest suite: **1804 tests across 110 files — all green**. No regressions from the new/extended files.

```
Test Files  110 passed (110)
      Tests 1804 passed (1804)
```

## Deviations from Plan

None — plan executed exactly as written. All four source interfaces verified against the live codebase before writing assertions:
- `hubGroupCounts.ts` imports confirmed; `memberKey` from `hubGroups.ts` used for `computeCounts` Set construction
- `agentBadge.ts` export name `agentBadgeModifier` confirmed (not `agentColor`/`agentLabel` as earlier PATTERNS.md noted)
- Sidebar.tsx: `showGroupList = effectiveExpanded && groupDefs.length > 0` confirmed; expanded by default (localStorage default)
- `style.css` line numbers verified: border-radius:16px at ~4423, data-agent spines at ~4437, .hub-card__badge at ~4639, .hub-card__preview height:150px at ~5367

## Known Stubs

None.

## Threat Flags

None — test-only changes; no new production code, endpoints, or auth paths introduced.

## Self-Check: PASSED

- [x] `frontend/src/lib/hubGroupCounts.test.ts` exists
- [x] `frontend/src/lib/agentBadge.test.ts` exists
- [x] `frontend/src/components/__tests__/Sidebar.test.tsx` extended with GAP-03 describe block
- [x] `frontend/src/components/__tests__/style.hub.test.ts` extended with GAP-04 describe block
- [x] All 3 task commits exist: 1a8d06e1, 6ec02016, 8482606e
- [x] `cd frontend && pnpm test` exits 0 (1804/1804 passing)
