---
phase: 132
plan: "05"
subsystem: frontend-integration
tags: [hub-panel, preview-poller, group-sidebar, remote-merge, app-tsx, style-css, CARD-07, GRID-03, GRID-07, reduced-motion, colorblind-safe]
dependency_graph:
  requires:
    - frontend/src/components/Hub/GroupSidebar.tsx (GRID-03 sidebar — Plan 03)
    - frontend/src/components/Hub/SessionCardGrid.tsx (groupByNamedGroups + preview/assign — Plan 04)
    - frontend/src/lib/hubGroups.ts (loadGroups/createGroup/assignToGroup/removeFromGroup — Plan 02)
    - frontend/src/lib/remoteAdapter.ts (adaptAllRemoteSessions — Plan 02)
    - frontend/src/wailsjs/go/main/App (GetSessionTailLines — Plan 01)
  provides:
    - frontend/src/components/Hub/HubPanel.tsx (usePreviewPoller + GroupSidebar layout + group state + remote merge)
    - frontend/src/App.tsx (remote poll gate extension + HubPanel prop wiring)
    - frontend/src/style.css (Phase 132 --hub-* tokens + hub__body/sidebar/preview/menu rules)
  affects:
    - All Hub surfaces (unified grid now merges local + remote sessions)
tech_stack:
  added: []
  patterns:
    - usePreviewPoller custom hook with stable sessionIdKey dep (prevents polling storm)
    - Single shared setInterval for all card previews (CARD-07 perf mandate)
    - localStorage-backed collapsed state (mirrors Sidebar.tsx pattern)
    - TDD red-green cycle for HubPanel integration
    - Named-group filter applied on top of status/search filters
key_files:
  created: []
  modified:
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/App.tsx
    - frontend/src/style.css
    - frontend/src/wailsjs/go/main/App.js
decisions:
  - "usePreviewPoller uses sessionIdKey (sessions.map(s=>s.id).join(',')) as stable dep — NOT the sessions array — to avoid the polling storm (Pitfall 3)"
  - "Remote sessions (hostname != '') are excluded from GetSessionTailLines fetches; they always render preview as 'No output yet' (Pitfall 4)"
  - "usePreviewPoller uses vi.advanceTimersByTimeAsync(100) in tests (not vi.runAllTimersAsync) to avoid infinite-loop detection with React 19 async effects"
  - "App.js Wails stub was missing GetSessionTailLines (only in .d.ts); added as deviation fix (Rule 3 — blocked build)"
  - "All new CSS transitions wrapped in @media (prefers-reduced-motion: no-preference) per Pitfall 7 and UI-SPEC Motion Contract"
metrics:
  duration: "~25 minutes"
  completed: "2026-06-16"
  tasks_completed: 3
  files_created: 0
  files_modified: 5
  tests_added: 20
---

# Phase 132 Plan 05: HubPanel Integration (usePreviewPoller + GroupSidebar + Remote Merge) Summary

Single shared 3s preview poller (CARD-07) wired into HubPanel, two-column hub__body layout hosting GroupSidebar with persisted named-group state (GRID-03), local+remote session merge (GRID-07), App.tsx remote-poll-gate extension, and all Phase 132 CSS tokens with prefers-reduced-motion discipline.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 RED | HubPanel failing tests (usePreviewPoller + hub__body + group state + remote merge) | 674952c0 | frontend/src/components/Hub/HubPanel.test.tsx |
| 1 GREEN | HubPanel implementation | 12d20fba | frontend/src/components/Hub/HubPanel.tsx, HubPanel.test.tsx |
| 2 | App.tsx remote-poll-gate extension + HubPanel prop wiring | c2989b04 | frontend/src/App.tsx |
| 3 | style.css Phase 132 tokens + BEM rules (reduced-motion safe) | 2f1d8072 | frontend/src/style.css, frontend/src/wailsjs/go/main/App.js, frontend/src/components/Hub/HubPanel.test.tsx |

## What Was Built

### HubPanel.tsx (CARD-07 + GRID-03 + GRID-07)

- `usePreviewPoller(sessions, isActive)`: single 3s `setInterval`; fetches only local sessions (hostname == ''); stable `sessionIdKey` dep prevents polling storm; paused when `isActive=false`; returns `Map<id, string[]>`
- New props: `remoteSessions?: SessionInfo[]` and `isActive?: boolean`
- `allSessions = [...sessions, ...(remoteSessions ?? [])]` — unified local + remote list
- Group state: `groupDefs` (init `loadGroups()`), `activeGroupId`, `sidebarCollapsed` (localStorage key `hub-group-sidebar-collapsed`)
- Named-group filter: when `activeGroupId` set, narrows `visibleSessions` by memberKey membership (or `__other__` for unmatched)
- `hub__body` flex-row wrapper containing `<GroupSidebar>` + `.hub__grid-scroll`
- GroupSidebar callbacks: `onCreateGroup`, `onDropOnGroup`, `onAssignGroup` via hubGroups.ts CRUD
- All Phase 131 behaviors preserved: status filter, search, '/' shortcut, error state, empty state

### App.tsx

- Remote poll gate extended: `if (activeId !== REMOTE_SESSIONS_TAB.id && activeId !== HUB_TAB.id) return`
- Imports `adaptAllRemoteSessions` from `./lib/remoteAdapter`
- HubPanel render adds `remoteSessions={adaptAllRemoteSessions(remotePeers)}` and `isActive={activeId === HUB_TAB.id}`
- All existing props preserved (sessions, error, onNewSession, onRename, onOpenSession)

### style.css (Phase 132 tokens + BEM rules)

- Phase 132 `--hub-preview-*`, `--hub-sidebar-*`, `--hub-needs-input-badge-*`, `--hub-drag-over-*` tokens appended to EXISTING `:root` and `[data-ui-theme="light"]` blocks (no duplicate `:root`)
- COLORBLIND-SAFE source comments: dark/light hex for needs-input badge and drag-over border
- New BEM rules: `.hub__body`, `.hub__group-sidebar` (+ `--collapsed`), group list/item rules, drag-handle/menu-btn (opacity 0 → 1), preview pane (56px), menu popover
- ALL new transitions inside `@media (prefers-reduced-motion: no-preference)`: sidebar width 150ms + opacity 100ms

### App.js Wails stub

- Added `GetSessionTailLines` export (was in `.d.ts` but missing from `.js`, causing build failure)

## Verification Results

```
pnpm vitest run src/components/Hub/HubPanel.test.tsx
Test Files  1 passed (1)
Tests       31 passed (31)

pnpm vitest run (full suite)
Test Files  99 passed (99)
Tests       1597 passed (1597)

pnpm build → ✓ built in 316ms (clean)

go build ./... → (no output — clean)

grep -c "usePreviewPoller" HubPanel.tsx → 4 (def + call + type + comment)
grep -c "setInterval" HubPanel.tsx → 1 (single shared timer)
grep -c "sessionIdKey" HubPanel.tsx → 3 (stable dep — Pitfall 3)
grep -c "hub__body" HubPanel.tsx → 3
grep -c "GroupSidebar" HubPanel.tsx → 4
grep -c "adaptAllRemoteSessions" App.tsx → 2
grep -c "activeId !== REMOTE_SESSIONS_TAB.id && activeId !== HUB_TAB.id" App.tsx → 1
grep -c "isActive={activeId === HUB_TAB.id}" App.tsx → 1
grep -c "^  --hub-preview-bg" style.css → 2 (dark + light blocks)
grep -c "@media (prefers-reduced-motion: no-preference)" style.css → 4
grep -c "COLORBLIND-SAFE: needs-input badge" style.css → 4
grep -c "COLORBLIND-SAFE: drag-over border" style.css → 2
```

## Acceptance Criteria Check

### Task 1 — HubPanel

- [x] `pnpm vitest run src/components/Hub/HubPanel.test.tsx` — 31/31 green (22 Phase 131 preserved + 9 new Phase 132)
- [x] `grep -c "usePreviewPoller" HubPanel.tsx` — 4 (at least 2)
- [x] `grep -c "setInterval" HubPanel.tsx` — 1 (single shared timer — per-card forbidden)
- [x] `grep -c "sessionIdKey" HubPanel.tsx` — 3 (at least 1 — stable dep Pitfall 3)
- [x] `grep -c "hub__body" HubPanel.tsx` — 3 (at least 1)
- [x] `grep -c "GroupSidebar" HubPanel.tsx` — 4 (at least 1)

### Task 2 — App.tsx

- [x] `grep -c "adaptAllRemoteSessions" App.tsx` — 2 (at least 1)
- [x] `grep -c "activeId !== REMOTE_SESSIONS_TAB.id && activeId !== HUB_TAB.id" App.tsx` — 1
- [x] `grep -c "isActive={activeId === HUB_TAB.id}" App.tsx` — 1
- [x] Full vitest suite green — no regressions

### Task 3 — style.css

- [x] `pnpm build` — clean
- [x] `grep -c "^  --hub-preview-bg" style.css` — 2 (dark + light)
- [x] `grep -c "@media (prefers-reduced-motion: no-preference)" style.css` — 4 (2 new blocks)
- [x] No new `transition:` outside reduced-motion blocks on hub__group-sidebar / hub-card__drag-handle / hub-card__menu-btn
- [x] `grep -c "COLORBLIND-SAFE: needs-input badge" style.css` — 4 (dark + light in CSS + tokens)
- [x] `grep -c "COLORBLIND-SAFE: drag-over border" style.css` — 2 (dark + light)

## TDD Gate Compliance

- RED commit (HubPanel): 674952c0 — `test(132-05): add failing tests for HubPanel usePreviewPoller + hub__body + group state + remote merge`
- GREEN commit (HubPanel): 12d20fba — `feat(132-05): implement HubPanel usePreviewPoller + GroupSidebar layout + group state + remote merge`

Tasks 2 and 3 are type `auto` (non-TDD), so no RED/GREEN gates required for those.

## Threat Model Compliance

| Threat ID | Mitigation Status |
|-----------|-------------------|
| T-132-12 | MITIGATED — Single `setInterval` (grep confirms count=1); stable `sessionIdKey` dep prevents rebinding storm; local-only fetch (remote excluded); paused when Hub inactive |
| T-132-13 | MITIGATED — Remote hostname/name/preview rendered via React text children only; no `dangerouslySetInnerHTML` in HubPanel or SessionCardGrid |
| T-132-14 | MITIGATED — `adaptAllRemoteSessions` (Plan 02) filters `reachable===false` before merge; unreachable peers contribute zero sessions |
| T-132-SC | N/A — Zero new npm dependencies |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] GetSessionTailLines missing from App.js Wails stub**
- **Found during:** Task 3 (pnpm build)
- **Issue:** `GetSessionTailLines` was declared in `App.d.ts` (Plan 01 artifact) but not exported from `App.js`; Vite/rolldown threw `MISSING_EXPORT` during production build
- **Fix:** Added `export const GetSessionTailLines = (id, n) => Call('main.App.GetSessionTailLines', [id, n])` to `App.js`
- **Files modified:** `frontend/src/wailsjs/go/main/App.js`
- **Committed in:** 2f1d8072

**2. [Rule 1 - Bug] vi.runAllTimersAsync() causes infinite-loop with React 19 async effects**
- **Found during:** Task 1 GREEN (first test run of timer-based tests)
- **Issue:** `vi.runAllTimersAsync()` triggers the `setInterval` recursively until vitest detects an infinite loop (>10000 timer fires); not compatible with React 19's async `useEffect` scheduling
- **Fix:** Switched to `vi.advanceTimersByTimeAsync(100)` which advances time by 100ms — enough to trigger the initial `poll()` call without entering a loop
- **Files modified:** `frontend/src/components/Hub/HubPanel.test.tsx`
- **Committed in:** 12d20fba

**3. [Rule 1 - Bug] Unused `beforeEach` import in HubPanel.test.tsx**
- **Found during:** Task 3 (`pnpm build` runs tsc on test files)
- **Issue:** `beforeEach` imported but never used, causing TS6133 error during `tsc`
- **Fix:** Removed from the import list
- **Files modified:** `frontend/src/components/Hub/HubPanel.test.tsx`
- **Committed in:** 2f1d8072

## Known Stubs

None — all data connections wired: `usePreviewPoller` fetches from `GetSessionTailLines`; `adaptAllRemoteSessions` converts `remotePeers`; group state persists to localStorage via `hubGroups.ts`. Manual UAT still needed per 132-VALIDATION.md (mini-preview perf at scale, drag-and-drop, remote peer card in grid).

## Threat Flags

None — no new network endpoints, auth paths, or trust boundary crossings beyond what is in the plan's threat model.

## Self-Check: PASSED

- [x] frontend/src/components/Hub/HubPanel.tsx — FOUND
- [x] frontend/src/components/Hub/HubPanel.test.tsx — FOUND
- [x] frontend/src/App.tsx — FOUND (adaptAllRemoteSessions import + gate + HubPanel props)
- [x] frontend/src/style.css — FOUND (Phase 132 tokens + BEM rules)
- [x] frontend/src/wailsjs/go/main/App.js — FOUND (GetSessionTailLines export)
- [x] commit 674952c0 — FOUND (test 132-05 RED)
- [x] commit 12d20fba — FOUND (feat 132-05 HubPanel GREEN)
- [x] commit c2989b04 — FOUND (feat 132-05 App.tsx)
- [x] commit 2f1d8072 — FOUND (feat 132-05 style.css)
