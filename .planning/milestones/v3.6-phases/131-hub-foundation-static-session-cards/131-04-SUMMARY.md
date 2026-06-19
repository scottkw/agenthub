---
phase: 131-hub-foundation-static-session-cards
plan: "04"
subsystem: frontend/Hub
tags: [react, typescript, vitest, tdd, hub, session-grid, filter, search, keyboard-shortcut]
dependency_graph:
  requires: [131-02, 131-03]
  provides: [SessionCardGrid, HubPanel]
  affects: [HubPanel (Plan 05 wires App.tsx polling)]
tech_stack:
  added: []
  patterns: [TDD red/green, BEM CSS classes, React controlled state, useEffect keydown listener]
key_files:
  created:
    - frontend/src/components/Hub/SessionCardGrid.tsx
    - frontend/src/components/Hub/SessionCardGrid.test.tsx
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
  modified: []
decisions:
  - "groupByWorkDir uses Map keyed by workDir||'' to preserve insertion order and handle empty workDir → 'Other'"
  - "basename helper inlined in SessionCardGrid (no node:path import) — splits on both / and \\ separators"
  - "filterSessions exported from HubPanel.tsx for testability — derives status inline mirroring SessionCard/HubFilterBar pattern"
  - "HubPanel renders error state as inline JSX (not a separate component) — matches DaemonManagerPanel analog"
  - "Case-insensitive search test fixed: both sessions must have distinct CLI values to avoid false positives (cli field is also searched)"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-16"
  tasks_completed: 2
  tasks_total: 2
  files_created: 4
  files_modified: 0
  tests_added: 28
---

# Phase 131 Plan 04: SessionCardGrid + HubPanel Summary

Wave 1 leaf components composed into the working Hub surface: SessionCardGrid groups sessions by
working directory with accessible group headers; HubPanel owns filter/search state, the '/'
shortcut, and renders the grid or the appropriate empty/error states — 28 tests green, full Hub
suite 85/85.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | SessionCardGrid failing tests | e851284 | SessionCardGrid.test.tsx |
| 1 (GREEN) | SessionCardGrid implementation | 77487b3 | SessionCardGrid.tsx |
| 2 (RED) | HubPanel failing tests | 30dbddf | HubPanel.test.tsx |
| 2 (GREEN) | HubPanel implementation | 87619a1 | HubPanel.tsx |

## What Was Built

### SessionCardGrid (`frontend/src/components/Hub/SessionCardGrid.tsx`)

Groups SessionCards by working directory with accessible group headers.

Key behaviors proven by tests:
- `groupByWorkDir(sessions)` → Map keyed by `s.workDir || ''`, preserving insertion order
- Empty workDir key → group header labeled "Other"
- Group header: `<h2>` with `role="heading"` `aria-level={2}`, `.hub__group-header` class
- Header span has `title={fullWorkDirPath}` tooltip; text shows `basename(workDir)` or "Other"
- `basename` helper: splits on `/` and `\`, takes last non-empty segment — no `node:path` import
- `.hub__card-row[role="list"]` with `div[role="listitem"]` wrappers per session (UI-SPEC rule 6)
- Each listitem wraps `<SessionCard session={s} onRename={onRename} />`
- 11 vitest specs green

### HubPanel (`frontend/src/components/Hub/HubPanel.tsx`)

Top-level Hub surface composing the filter bar, grid, and empty/error states.

Key behaviors proven by tests:
- `filterSessions(sessions, filter, search)` helper exported for testability:
  - Status filter: maps HubFilter to deriveStatus output; 'all' passes everything
  - Search: case-insensitive substring match on `name`, `cli`, and `hostname`
- Owns `activeFilter` (HubFilter, default 'all'), `searchText`, and `searchRef`
- `window.addEventListener('keydown')` effect: '/' when activeElement is not INPUT → focus+select searchRef
- `handleClearFilter` resets both `activeFilter` and `searchText`
- Rendering priority (error beats sessions beats filtered beats grid):
  - `error=true` → `hub__error-state` with exact UI-SPEC copy
  - `sessions.length === 0` → HubEmptyState variant="no-sessions"
  - `filtered.length === 0 && sessions.length > 0` → HubEmptyState variant="no-matches"
  - Otherwise → `<SessionCardGrid sessions={filtered} onRename={onRename} />`
- Passes `sessions` (not `filtered`) to HubFilterBar for live counts per pill
- Header: `.hub__header` with `.hub__title` "Hub" + "New session" button
- 17 vitest specs green

## Test Results

```
Test Files  6 passed (6)
      Tests 85 passed (85)
```

Full Hub component suite: HubPanel (17) + SessionCardGrid (11) + HubFilterBar (19) + HubEmptyState (11) + SessionCard (22) + InlineSessionName (5) = 85.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test case-insensitive search had false positives due to shared CLI default**

- **Found during:** Task 2 GREEN (test failure)
- **Issue:** The "case-insensitively filters sessions by name" test used default `cli: 'claude'` on both
  sessions. `filterSessions` searches `cli` field too, so searching 'CLAUDE' matched both sessions,
  not just the one with 'Claude' in its name. The filter logic was correct; the test fixture was wrong.
- **Fix:** Changed the two test sessions to use `cli: 'opencode'` and `cli: 'gemini'`; changed search
  term from 'CLAUDE' to 'UNIQUE' (matching only the first session's name 'My Unique Task').
- **Files modified:** `frontend/src/components/Hub/HubPanel.test.tsx` (test fixture only)
- **Commit:** 87619a1

## TDD Gate Compliance

- RED gate: test commits exist before GREEN (e851284, 30dbddf)
- GREEN gate: feat commits follow each test commit (77487b3, 87619a1)
- Gate sequence validated for both tasks

## Known Stubs

None — both components are fully wired to real data via props. HubPanel receives `sessions` and
`error` from its parent (App.tsx wires the polling in Plan 05).

## Threat Mitigations Applied

- **T-131-08 (XSS):** `workDir` path rendered in `SessionCardGrid` as JSX text child (`{headerLabel}`)
  and as the `title` attribute string (`title={titleValue}`). Both are React-escaped automatically.
  No `innerHTML` or `dangerouslySetInnerHTML` in either component.
- **T-131-09 (npm installs):** No new packages added. Both components import only from existing
  installed packages (`@heroicons/react`, `react`).

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries.

## Self-Check: PASSED

Files exist:
- frontend/src/components/Hub/SessionCardGrid.tsx: FOUND
- frontend/src/components/Hub/SessionCardGrid.test.tsx: FOUND
- frontend/src/components/Hub/HubPanel.tsx: FOUND
- frontend/src/components/Hub/HubPanel.test.tsx: FOUND

Commits exist:
- e851284: test(131-04): add failing tests for SessionCardGrid
- 77487b3: feat(131-04): implement SessionCardGrid component
- 30dbddf: test(131-04): add failing tests for HubPanel
- 87619a1: feat(131-04): implement HubPanel surface + tests

Acceptance criteria:
- `grep -c 'groupByWorkDir' SessionCardGrid.tsx` = 3 >= 1
- No `from 'path'` or `from 'node:path'` in SessionCardGrid.tsx
- `grep -c "addEventListener('keydown'" HubPanel.tsx` = 1 >= 1
- `grep -c "Couldn't load sessions" HubPanel.tsx` = 2 >= 1
- Full Hub suite: 85/85 tests green
