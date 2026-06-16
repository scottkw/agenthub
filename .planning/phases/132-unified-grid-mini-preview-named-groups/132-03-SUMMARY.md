---
phase: 132
plan: "03"
subsystem: frontend-components
tags: [hub-components, mini-preview, group-sidebar, tdd, vitest, CARD-07, GRID-03, GROUP-01, GROUP-02, colorblind-safe, aria]
dependency_graph:
  requires:
    - frontend/src/lib/hubGroups.ts (HubGroupDef, memberKey — Plan 02)
    - frontend/src/lib/hubStatus.ts (deriveHubStatus)
  provides:
    - frontend/src/components/Hub/MiniPreview.tsx (CARD-07 plain-text preview pane)
    - frontend/src/components/Hub/GroupSidebar.tsx (GRID-03 sidebar + GroupSidebarItem + GROUP-01 create + GROUP-02 drop target)
  affects:
    - Wave-3 SessionCard (Plan 04 — consumes MiniPreview as ROW 6)
    - Wave-4 HubPanel (Plan 05 — mounts GroupSidebar, owns collapsed state)
tech_stack:
  added: []
  patterns:
    - TDD red-green cycle for React components using createRoot + act
    - BEM modifier classes via conditional className strings
    - HTML5 drag-and-drop drop-target pattern (onDragOver preventDefault + state, onDrop getData)
    - Controlled input inline create flow (Enter/Escape/blur handlers)
    - ARIA listbox/option role pattern for group filter sidebar
key_files:
  created:
    - frontend/src/components/Hub/MiniPreview.tsx
    - frontend/src/components/Hub/MiniPreview.test.tsx
    - frontend/src/components/Hub/GroupSidebar.tsx
    - frontend/src/components/Hub/GroupSidebar.test.tsx
  modified: []
decisions:
  - MiniPreview renders empty-string lines as ' ' (non-breaking space) to prevent row height collapse
  - GroupSidebar collapsed state is fully controlled (no internal localStorage) — HubPanel owns persistence (Plan 05)
  - DnD tests use plain Event instead of DragEvent (jsdom does not support DragEvent natively)
  - Input simulation uses HTMLInputElement.prototype native setter to trigger React controlled state
  - All item shows aria-selected=true when no group is selected (activeGroupId===null)
metrics:
  duration: "~8 minutes"
  completed: "2026-06-16"
  tasks_completed: 2
  files_created: 4
  tests_added: 36
---

# Phase 132 Plan 03: MiniPreview + GroupSidebar Components Summary

Plain-text session preview pane (CARD-07) and collapsible group sidebar with colorblind-safe needs-input badge (GRID-03), inline group create flow (GROUP-01), and drop-target DnD half (GROUP-02). Both standalone with full vitest coverage. Zero new dependencies.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 RED | MiniPreview.test.tsx — failing tests | 0321795 | frontend/src/components/Hub/MiniPreview.test.tsx |
| 1 GREEN | MiniPreview.tsx — plain-text pane | 62a4cd8 | frontend/src/components/Hub/MiniPreview.tsx |
| 2 RED | GroupSidebar.test.tsx — failing tests | 6d891b7 | frontend/src/components/Hub/GroupSidebar.test.tsx |
| 2 GREEN | GroupSidebar.tsx — sidebar + create + drop | e44f95b | frontend/src/components/Hub/GroupSidebar.tsx, GroupSidebar.test.tsx (updated) |

## What Was Built

### MiniPreview.tsx (CARD-07)

- Three render branches: `lines === undefined` → Loading… + `hub-card__preview--loading`; `lines.length === 0` → No output yet + `hub-card__preview--empty`; else renders up to N `.hub-card__preview-line` divs
- `aria-hidden="true"` on outer pane in all states (decorative, not interactive)
- Empty-string lines rendered as `' '` (space) to preserve row height
- NO xterm import, NO setInterval, NO new Terminal() — purely presentational
- Required `/* CARD-07: mini preview is plain text snapshot — NO xterm instance; polling interval 3s shared across all cards */` comment

### GroupSidebar.tsx (GRID-03 + GROUP-01 + GROUP-02)

- `GroupSidebar` + `GroupSidebarItem` exported from same file
- Controlled collapse: `collapsed`/`onToggle` are props (not internal state) — HubPanel will own in Plan 05
- "All" item at top, one `GroupSidebarItem` per `HubGroupDef`
- Counts: `running` = sessions in group where `deriveHubStatus ∈ {running, idle, waiting}`; `total` = all; `waiting` = deriveHubStatus === 'waiting'
- Needs-input badge: `PauseCircleIcon` + count + aria-label ("N session needs input" / "N sessions need input") — colorblind-safe shape-first design
- ARIA: `role="listbox"` on `<ul>`, `role="option"` + `aria-selected` on each `<li>`
- Toggle `aria-expanded` + `aria-controls` pointing to list id
- Inline create flow: "New group" click → `<input>` with placeholder "Group name…"; Enter (non-empty trimmed) → `onCreateGroup`; Escape → cancel; empty Enter → no-op
- Drop target per FileBrowserTab pattern: `onDragOver` (preventDefault + isDragOver=true), `onDragLeave`, `onDrop` (reads text/plain, calls `onDropOnGroup`)
- No `draggable` attribute — drop target only (drag source on SessionCard in Plan 04)
- COLORBLIND-SAFE source comments: dark hex `#f59e0b` + light hex `#b45309`

## Verification Results

```
pnpm vitest run src/components/Hub/MiniPreview.test.tsx src/components/Hub/GroupSidebar.test.tsx
Test Files  2 passed (2)
Tests       36 passed (36)

grep -rn "dangerouslySetInnerHTML" MiniPreview.tsx GroupSidebar.tsx → (empty)
grep -c "PauseCircleIcon" GroupSidebar.tsx → 4
grep -c "COLORBLIND-SAFE" GroupSidebar.tsx → 3
grep -c "role=\"listbox\"|role=\"option\"" GroupSidebar.tsx → 2
grep -c "draggable" GroupSidebar.tsx → 0
grep -c "aria-hidden" MiniPreview.tsx → 4
grep -c "CARD-07" MiniPreview.tsx → 1
```

## Acceptance Criteria Check

### Task 1 — MiniPreview

- [x] `pnpm vitest run src/components/Hub/MiniPreview.test.tsx` — 11/11 green
- [x] `grep -c "xterm\|setInterval\|Terminal(" MiniPreview.tsx` — 2 (both in required source comment/JSDoc, no functional usage)
- [x] `grep -c "aria-hidden" MiniPreview.tsx` — 4 (once per render branch + JSDoc)
- [x] `grep -c "CARD-07" MiniPreview.tsx` — 1

### Task 2 — GroupSidebar

- [x] `pnpm vitest run src/components/Hub/GroupSidebar.test.tsx` — 25/25 green (includes PauseCircleIcon assertion + empty-name no-op)
- [x] `grep -c "PauseCircleIcon" GroupSidebar.tsx` — 4 (at least 1)
- [x] `grep -c "COLORBLIND-SAFE" GroupSidebar.tsx` — 3 (at least 1)
- [x] `grep -c "role=\"listbox\"\|role=\"option\"" GroupSidebar.tsx` — 2 (at least 2)
- [x] `grep -c "draggable" GroupSidebar.tsx` — 0 (drop target only)

## TDD Gate Compliance

- RED commit (MiniPreview): 0321795 — `test(132-03): add failing tests for MiniPreview loading/empty/data/non-collapsing-line states`
- GREEN commit (MiniPreview): 62a4cd8 — `feat(132-03): implement MiniPreview plain-text snapshot pane (CARD-07)`
- RED commit (GroupSidebar): 6d891b7 — `test(132-03): add failing tests for GroupSidebar counts/badge/collapse/create/drop`
- GREEN commit (GroupSidebar): e44f95b — `feat(132-03): implement GroupSidebar + GroupSidebarItem (GRID-03, GROUP-01, GROUP-02)`

## Threat Model Compliance

| Threat ID | Mitigation Status |
|-----------|-------------------|
| T-132-07 | MITIGATED — MiniPreview renders lines as React text children only; no dangerouslySetInnerHTML |
| T-132-08 | MITIGATED — GroupSidebar trims + rejects empty/whitespace before firing onCreateGroup |
| T-132-09 | ACCEPTED — drop payload is same-page member key; no privileged effect |
| T-132-SC | N/A — Zero new npm dependencies |

## Deviations from Plan

**1. [Rule 1 - Bug] DragEvent unavailable in jsdom**
- **Found during:** Task 2 (GroupSidebar RED phase)
- **Issue:** jsdom does not implement `DragEvent` globally; tests throwing `ReferenceError: DragEvent is not defined`
- **Fix:** Used plain `Event` with `dataTransfer` property defined via `Object.defineProperty` — React's synthetic event system accepts the native drop event and reads `dataTransfer` correctly
- **Files modified:** frontend/src/components/Hub/GroupSidebar.test.tsx

**2. [Rule 1 - Bug] React controlled input simulation**
- **Found during:** Task 2 (GroupSidebar GREEN phase)
- **Issue:** Setting `input.value = 'My Group'` directly doesn't trigger React controlled state; `onCreateGroup` not called
- **Fix:** Used `HTMLInputElement.prototype value setter` via `Object.getOwnPropertyDescriptor` before dispatching the `input` event — standard pattern for testing React controlled inputs without @testing-library
- **Files modified:** frontend/src/components/Hub/GroupSidebar.test.tsx

## Known Stubs

None — both components are purely presentational with no data stubs. MiniPreview receives `lines` prop; GroupSidebar receives `sessions` prop. Data wiring happens in Plans 04/05.

## Threat Flags

None — no new network endpoints, auth paths, or trust boundary crossings.

## Self-Check: PASSED

- [x] frontend/src/components/Hub/MiniPreview.tsx — FOUND
- [x] frontend/src/components/Hub/MiniPreview.test.tsx — FOUND
- [x] frontend/src/components/Hub/GroupSidebar.tsx — FOUND
- [x] frontend/src/components/Hub/GroupSidebar.test.tsx — FOUND
- [x] commit 0321795 — FOUND (test 132-03 MiniPreview RED)
- [x] commit 62a4cd8 — FOUND (feat 132-03 MiniPreview GREEN)
- [x] commit 6d891b7 — FOUND (test 132-03 GroupSidebar RED)
- [x] commit e44f95b — FOUND (feat 132-03 GroupSidebar GREEN)
