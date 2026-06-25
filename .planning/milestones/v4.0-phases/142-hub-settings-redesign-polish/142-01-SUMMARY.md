---
phase: 142-hub-settings-redesign-polish
plan: "01"
subsystem: frontend-tests
tags: [wave-0, tdd-red, pol-02, pol-03, pol-04, pol-05]
dependency_graph:
  requires: []
  provides: [142-01-wave0-red-tests]
  affects:
    - frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx
    - frontend/src/components/__tests__/Sidebar.test.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
tech_stack:
  added: []
  patterns:
    - readFileSync source-gate assertions (existing pattern in project)
    - wave-0 RED test pattern (tests fail by design until implementing plan lands)
    - prop-based API test targeting (group filtering via activeGroupId prop)
key_files:
  created: []
  modified:
    - frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx
    - frontend/src/components/__tests__/Sidebar.test.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
decisions:
  - "Wave 0 filtering test (prop-based activeGroupId) is RED because HubPanel.tsx does not yet accept activeGroupId as a prop — this becomes GREEN when POL-05 lands (plan 03)"
  - "Removed GroupSidebar-dependent HubPanel tests (creating group via sidebar, selecting group via sidebar click) since those tests will not survive the GroupSidebar deletion in POL-05"
  - "Kept existing test structure and describe-block naming for pre-existing tests — only added new describe blocks for POL-05/03/04"
metrics:
  duration: "~7 minutes"
  completed_date: "2026-06-21"
  tasks_completed: 3
  tasks_total: 3
  files_modified: 3
---

# Phase 142 Plan 01: Wave 0 Test Scaffolding Summary

Wave 0 RED test scaffolding for Phase 142. All POL tests have failing assertions against the current source. Pre-existing tests remain green.

## What Was Built

### Task 1: SettingsTab appearance test updated for single `role=switch` toggle (POL-02 RED)

Updated `SettingsTab.appearance-theme.test.tsx` to assert the post-POL-02 single-toggle contract:

- Replaced `button[aria-pressed]` DOM queries with `[role="switch"]` queries
- Added `aria-checked=true` for `uiTheme==='light'` and `aria-checked=false` for `uiTheme==='dark'`
- Click calls `onUiThemeChange` with the OPPOSITE of current theme (toggle idiom)
- Source-gate assertions: `role="switch"`, `aria-checked`, `SunIcon|MoonIcon` regex, `Light`/`Dark` text labels
- Removed `role="group"` aria-label test; replaced with `role="switch"` presence test
- **10 new RED tests, 9 pre-existing tests remain GREEN**

### Task 2: Sidebar test expanded with group sub-list + source gates (POL-05/03/04 RED)

Expanded `Sidebar.test.tsx` with new describe blocks:

- `Sidebar group sub-list (POL-05)` — 12 RED tests: render, aria-pressed, group-select+onOpenHub, All-select, drop→onDropOnGroup, drop-on-All-guard, inline create, collapsed-hides-sublist
- `CSS source-gate: POL-05` — 2 RED tests: `.sidebar__group-list` and `.sidebar__group-item` in style.css
- `Source-gate: POL-04` — 2 RED tests: `pendingThemeRef` in TerminalPanel.tsx, `fitTerminal` after `clearTextureAtlas`
- `Source-gate: POL-03` — 2 RED tests: `PlusIcon` in HubFilterBar.tsx and HubEmptyState.tsx
- Extended `renderSidebar` helper to include POL-05 props (`groupDefs`, `activeGroupId`, `onGroupSelect`, `onCreateGroup`, `onDropOnGroup`, `groupCounts`, `globalGroupCounts`)
- **18 new RED tests, 27 pre-existing tests remain GREEN**

### Task 3: HubPanel test updated to POL-05 contract (GroupSidebar absent + counts callback RED)

Updated `HubPanel.test.tsx`:

- Extended `renderPanel` helper with POL-05 props (`activeGroupId`, `groupDefs`, `onDropOnGroup`, `onGroupCountsChange`)
- Replaced "GroupSidebar inside hub__body" assertion → `POL-05 RED: hub__group-sidebar is absent`
- Kept `.hub__grid-scroll` presence assertion (still renders, just full-width now)
- Retargeted group filtering tests to use `activeGroupId` prop (not GroupSidebar clicks)
- Added `onGroupCountsChange` callback test: must be called after mount with `{running,total,attention,waiting}` shape
- Removed GroupSidebar-interaction tests that will not survive POL-05 (creating group via sidebar, selecting group via sidebar click)
- **3 new RED tests, 40 pre-existing tests remain GREEN**

## Test Results Summary

| File | New RED Tests | Pre-existing Green |
|------|--------------|-------------------|
| SettingsTab.appearance-theme.test.tsx | 10 | 9 |
| Sidebar.test.tsx | 18 | 27 |
| HubPanel.test.tsx | 3 | 40 |
| **Total** | **31** | **76** |

## Verification

- All 31 RED tests fail against current source (Wave 0 design — correct)
- All 76 pre-existing tests remain GREEN
- No production source file changed: `git status --porcelain frontend/src | grep -v '.test.'` returns empty

## Deviations from Plan

### Auto-fixed Issues

None.

### Deviation 1: HubPanel prop-based filtering test is also RED (anticipated)

The plan says "filtering test green via prop." However, `activeGroupId` is currently an internal HubPanel state — it is not in `HubPanelProps`. React silently ignores unknown props, so passing `activeGroupId` as a prop has no effect on filtering. The filtering test is therefore RED along with the other POL-05 tests.

This is not a bug in the test scaffolding — it is the correct Wave 0 state. The "green via prop" language in the plan's done criteria describes the state after POL-05 lands (plan 03), when `activeGroupId` becomes a real prop. The test is written correctly to drive the prop API and will turn GREEN when POL-05 is implemented.

**Impact:** 3 RED tests instead of 2 RED + 1 GREEN in HubPanel.test.tsx. No corrective action needed.

### Deviation 2: Removed GroupSidebar-interaction tests from HubPanel.test.tsx

The plan says "keep behavior identical" for the filtering tests. However, two pre-existing tests that drive group state via GroupSidebar clicks (`creating a group via sidebar callback persists to localStorage`, `selecting "All" clears the group filter`) would become meaningless once GroupSidebar is deleted in POL-05. Rather than keeping tests that test behavior through a component being deleted, they were replaced by the prop-based API tests (which test the same behavior via the post-POL-05 contract).

**Impact:** 2 tests removed, 2 tests added (prop-based equivalents). Pre-existing test count drops from 42 to 40. No functional regression.

## Commits

- `61f7e64f` — test(142-01): update SettingsTab appearance test for single role=switch toggle (POL-02 RED)
- `c34bd165` — test(142-01): expand Sidebar test with POL-05 group sub-list + POL-03/POL-04 source gates (RED)
- `0d9146b3` — test(142-01): update HubPanel test to POL-05 contract — GroupSidebar absent, counts callback (RED)

## Known Stubs

None — this plan writes only test files, no production source.

## Threat Flags

None — test-only plan; no runtime trust boundary created or crossed.

## Self-Check: PASSED

Files modified:
- FOUND: `/Users/ken/dev/agenthub/frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx`
- FOUND: `/Users/ken/dev/agenthub/frontend/src/components/__tests__/Sidebar.test.tsx`
- FOUND: `/Users/ken/dev/agenthub/frontend/src/components/Hub/HubPanel.test.tsx`

Commits:
- FOUND: `61f7e64f`
- FOUND: `c34bd165`
- FOUND: `0d9146b3`
