---
phase: 131-hub-foundation-static-session-cards
plan: "03"
subsystem: frontend/Hub
tags: [react, typescript, vitest, tdd, hub, filter, empty-state]
dependency_graph:
  requires: [131-01]
  provides: [HubFilterBar, HubEmptyState, HubFilter type]
  affects: [HubPanel (Plan 04 will compose these)]
tech_stack:
  added: []
  patterns: [TDD red/green, BEM CSS classes, React controlled input, ref forwarding]
key_files:
  created:
    - frontend/src/components/Hub/HubFilterBar.tsx
    - frontend/src/components/Hub/HubFilterBar.test.tsx
    - frontend/src/components/Hub/HubEmptyState.tsx
    - frontend/src/components/Hub/HubEmptyState.test.tsx
  modified: []
decisions:
  - "Status count derivation in HubFilterBar mirrors SessionCard.deriveStatus: stopped+exitCode!=0 → stopped-err, stopped+exitCode=0 → stopped-ok"
  - "HubFilter type exported from HubFilterBar.tsx so HubPanel and consumers can import it from a single location"
  - "FILTER_PILLS array defined as a const to ensure pill order and label consistency with UI-SPEC Copywriting Contract"
metrics:
  duration: "15 minutes"
  completed: "2026-06-16"
  tasks_completed: 2
  tasks_total: 2
  files_created: 4
  files_modified: 0
  tests_added: 30
---

# Phase 131 Plan 03: Hub Filter Bar + Empty State Summary

HubFilterBar and HubEmptyState — two leaf components that HubPanel (Plan 04) composes. Both built TDD with 30 tests green.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | HubFilterBar failing tests | 68ec792 | HubFilterBar.test.tsx |
| 1 (GREEN) | HubFilterBar implementation | b0ab99d | HubFilterBar.tsx |
| 2 (RED) | HubEmptyState failing tests | e89111e | HubEmptyState.test.tsx |
| 2 (GREEN) | HubEmptyState implementation | 96f793c | HubEmptyState.tsx |

## What Was Built

### HubFilterBar (`frontend/src/components/Hub/HubFilterBar.tsx`)

Exports the `HubFilter` type union (`'all' | 'running' | 'waiting' | 'stopped-ok' | 'stopped-err' | 'idle'`) and the `HubFilterBar` component.

Key behaviors proven by tests:
- 6 filter pills (All, Working, Needs input, Complete, Error, Idle) — all with correct UI-SPEC labels
- Live counts on every non-All pill, computed via `deriveFilterStatus` (mirrors SessionCard logic)
- Active pill gets `hub-filter__pill--active` modifier
- Clicking any pill fires `onFilterChange` with the mapped HubFilter key
- Search input: `aria-label="Search sessions by name, CLI, or host"`, placeholder `"Search sessions…"`, value controlled, Escape clears + blurs
- `searchRef` forwarded to the input for the parent "/" shortcut
- "New session" button fires `onNewSession`

### HubEmptyState (`frontend/src/components/Hub/HubEmptyState.tsx`)

Two-variant empty state component:
- `no-sessions`: "No sessions yet" / "Create a session to start an AI coding agent." / "New session" CTA
- `no-matches`: "No matching sessions" / "Clear the filter or search to see all sessions." / "Clear filter" CTA

All copy is verbatim from the UI-SPEC Copywriting Contract.

## Test Results

```
Test Files  2 passed (2)
      Tests 30 passed (30)
```

## Deviations from Plan

None — plan executed exactly as written. Both components follow the PATTERNS.md analog patterns exactly.

## TDD Gate Compliance

- RED gate: test commits exist before GREEN (68ec792, e89111e)
- GREEN gate: feat commits follow each test commit (b0ab99d, 96f793c)
- Gate sequence validated for both tasks

## Known Stubs

None — both components are complete leaf components with no data stubs. HubFilterBar uses the sessions prop for live counts; HubEmptyState renders static copy.

## Threat Flags

None — no new network endpoints or trust boundaries. T-131-06 (XSS via JSX text) handled automatically by React default escaping. T-131-07 (npm installs) accepted — no new packages added.

## Self-Check: PASSED

Files exist:
- frontend/src/components/Hub/HubFilterBar.tsx: FOUND
- frontend/src/components/Hub/HubFilterBar.test.tsx: FOUND
- frontend/src/components/Hub/HubEmptyState.tsx: FOUND
- frontend/src/components/Hub/HubEmptyState.test.tsx: FOUND

Commits exist:
- 68ec792: test(131-03): add failing tests for HubFilterBar
- b0ab99d: feat(131-03): implement HubFilterBar component
- e89111e: test(131-03): add failing tests for HubEmptyState
- 96f793c: feat(131-03): implement HubEmptyState component
