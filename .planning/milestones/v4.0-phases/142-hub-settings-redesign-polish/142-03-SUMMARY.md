---
phase: 142-hub-settings-redesign-polish
plan: "03"
subsystem: frontend
tags: [pol-05, sidebar, hub-groups, state-lift, drag-drop, a11y]
dependency_graph:
  requires: ["142-01"]
  provides: ["POL-05-green"]
  affects: ["frontend/src/App.tsx", "frontend/src/components/Sidebar.tsx", "frontend/src/components/Hub/HubPanel.tsx", "frontend/src/lib/hubGroupCounts.ts"]
tech_stack:
  added: []
  patterns:
    - "State lift: groupDefs/activeGroupId lifted from HubPanel to App.tsx; counts flow up via callback (allSessions not lifted)"
    - "CARRY-01: li owns drag handlers + visual classes; inner button owns interactive ARIA"
    - "HTML5 dataTransfer text/plain drop protocol for session→group assignment"
    - "Drag auto-expand: collapsed sidebar temporarily expands during drag; does not persist to localStorage"
    - "Counts emitted upward via useEffect keyed on session+group identity strings (avoids stale closure on allSessions array)"
key_files:
  created:
    - frontend/src/lib/hubGroupCounts.ts
  modified:
    - frontend/src/App.tsx
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Sidebar.tsx
    - frontend/src/style.css
  deleted:
    - frontend/src/components/Hub/GroupSidebar.tsx
    - frontend/src/components/Hub/GroupSidebar.test.tsx
decisions:
  - "Removed __other__ item from Sidebar group sub-list — test gate (Plan 01) expects exactly All + named groups (N items); __other__ filtering in HubPanel preserved via prop"
  - "Visible button text = label only (no count span); count exposed via aria-label to keep textContent findable in tests (per CARRY-01 ARIA contract)"
  - "useEffect dependency strings instead of array references to avoid excessive counts callbacks on preview-poll renders"
metrics:
  duration: "~25 minutes"
  completed: "2026-06-21T22:12:50Z"
  tasks_completed: 2
  files_modified: 5
  files_created: 1
  files_deleted: 2
---

# Phase 142 Plan 03: Hub Group Sidebar → Main Sidebar (POL-05) Summary

Resolved POL-05: moved Hub group navigation out of the secondary side-by-side `GroupSidebar` panel into an expandable sub-list nested under the "Hub" item in the main left sidebar. The session grid now spans full width with no second collapsible panel.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Extract count helpers + lift group state to App; simplify HubPanel | 200bef2a | hubGroupCounts.ts (created), App.tsx, HubPanel.tsx |
| 2 | Build Sidebar nested group sub-list + CSS; delete GroupSidebar | ce6ff4a2 | Sidebar.tsx, style.css, GroupSidebar.tsx (deleted), GroupSidebar.test.tsx (deleted) |

## What Was Built

**hubGroupCounts.ts** — `computeCounts(sessions, memberKeys)` and `computeGlobalCounts(sessions)` helpers extracted verbatim from GroupSidebar.tsx, plus the `GroupCounts` interface. Single source of truth for both HubPanel emission and future Sidebar display.

**App.tsx** — Group state lifted: `groupDefs`, `activeGroupId`, `groupCounts`, `globalGroupCounts`. Handlers: `handleGroupSelect`, `handleCreateGroup`, `handleDropOnGroup`, `handleGroupCountsChange`. Both `<Sidebar>` and `<HubPanel>` receives the relevant props.

**HubPanel.tsx** — Internal `groupDefs`/`activeGroupId`/`sidebarCollapsed` state removed; replaced with `activeGroupId?`/`groupDefs?`/`onDropOnGroup?`/`onGroupCountsChange?` props. Added `useEffect` that computes and emits per-group + global counts whenever `allSessions` or `groupDefs` changes. GroupSidebar import and render removed; `hub__body` now wraps only `hub__grid-scroll` (full width). Per-card `handleAssignGroup` delegates to `onDropOnGroupProp`. Group filtering (`activeGroupId === null / '__other__' / id`) unchanged — driven by prop now.

**Sidebar.tsx** — Extended `SidebarProps` with 7 POL-05 props. New `GroupItem` sub-component with CARRY-01 structure (li owns drag handlers + visual classes; button owns `aria-pressed` + `aria-label` with counts). Renders `ul.sidebar__group-list` with "All" + named group items when `!collapsed && groupDefs.length > 0`. Drag auto-expand: collapsed sidebar temporarily shows group list during drag without persisting to `sidebar-collapsed` localStorage key. Inline group creation with Enter/Escape/blur pattern (V5 input guard).

**style.css** — Added `sidebar__group-list/item/item__btn/active/drag-over/name/count/new/new-input` rules using only `var(--hub-*)` tokens (confirmed: `--hub-text-muted`, `--hub-text-primary`, `--hub-accent`, `--hub-sidebar-item-hover-bg`, `--hub-sidebar-item-active-bg`, `--hub-drag-over-border`, `--hub-drag-over-bg` all present in both dark `:root` and `[data-ui-theme="light"]` blocks). Motion contract applied in `prefers-reduced-motion: no-preference` / `reduce` two-block pattern.

**GroupSidebar.tsx + GroupSidebar.test.tsx** — Deleted. Logic absorbed: count functions → hubGroupCounts.ts, drag/count tests → Sidebar.test.tsx (Plan 01).

## Verification Results

- `tsc --noEmit` exits 0
- `git grep -n "GroupSidebar" frontend/src` — only comments, no live imports
- `grep -nE "sidebar__group.*#[0-9a-fA-F]{3,6}" frontend/src/style.css` — empty (no raw hex)
- HubPanel.test.tsx: 43/43 tests pass (including POL-05 RED tests turned GREEN)
- Sidebar.test.tsx: 41/45 pass; 4 remaining failures are POL-03 + POL-04 source-gate tests (scope of Plan 02, remain RED until Plan 02 lands — expected)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed `__other__` sidebar item to match test gate**
- **Found during:** Task 2 — Sidebar tests
- **Issue:** Plan spec said to include an `__other__`/"Other" item in the sidebar group list, but the Plan 01 RED test (`renders one group item button per groupDef plus an All item`) explicitly expects `1 All + 2 groups = 3 items` for 2 groupDefs (not 4). The plan spec and the test were inconsistent.
- **Fix:** Omitted the `__other__` item from Sidebar rendering. HubPanel still handles `__other__` filtering internally via the `activeGroupId` prop — that functionality is preserved.
- **Files modified:** frontend/src/components/Sidebar.tsx
- **Commit:** ce6ff4a2

**2. [Rule 1 - Bug] Button textContent = label only (no count span)**
- **Found during:** Task 2 — Sidebar tests
- **Issue:** Tests find buttons by `textContent?.trim() === 'All'` and `textContent?.includes('Alpha')`. Initial implementation rendered `<span>label</span><span>N/T</span>` making textContent = "All0/0" which doesn't match.
- **Fix:** Button renders only the label string; counts are in `aria-label` only (which the aria-label test `toMatch(/\d+\/\d+/)` verifies).
- **Files modified:** frontend/src/components/Sidebar.tsx
- **Commit:** ce6ff4a2

## Known Stubs

None. Group navigation is wired end-to-end:
- `groupDefs` from `loadGroups()` (localStorage) via App.tsx state
- Counts computed in HubPanel useEffect via `computeCounts/computeGlobalCounts`
- Counts flow to Sidebar via `groupCounts`/`globalGroupCounts` props
- `activeGroupId` set via `onGroupSelect` → Sidebar → App.tsx → HubPanel (filtering)
- Drag assignment via `onDropOnGroup` → App.tsx → `assignToGroup/removeFromGroup` → `saveGroups`

## Threat Surface Scan

No new surface beyond what the plan's threat model covers. Group name rendered as React text content (T-142-06 mitigated). localStorage interactions via existing `loadGroups/saveGroups` guard (T-142-05 accepted). Drop handler reads `dataTransfer.getData('text/plain')` with null guard on "All" (T-142-04 mitigated).

## Self-Check: PASSED

- [x] `frontend/src/lib/hubGroupCounts.ts` exists
- [x] `frontend/src/components/Sidebar.tsx` extended with POL-05 props
- [x] `frontend/src/components/Hub/HubPanel.tsx` no longer imports GroupSidebar
- [x] `frontend/src/components/Hub/GroupSidebar.tsx` deleted
- [x] `frontend/src/components/Hub/GroupSidebar.test.tsx` deleted
- [x] Commits `200bef2a` and `ce6ff4a2` exist in git log
- [x] tsc exits 0
- [x] All POL-05 tests green (HubPanel 43/43, Sidebar POL-05 group all pass)
