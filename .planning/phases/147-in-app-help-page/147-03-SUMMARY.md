---
phase: 147-in-app-help-page
plan: "03"
subsystem: frontend/help
tags: [react, typescript, routing, sidebar, tab-union]
dependency_graph:
  requires: ["147-01", "147-02"]
  provides: ["147-04"]
  affects: ["frontend/src/components/TabBar.tsx", "frontend/src/components/Sidebar.tsx", "frontend/src/App.tsx"]
tech_stack:
  added: []
  patterns: ["display-toggle mount (flex/none)", "find-or-add tab idempotency", "Tab type discriminant union extension"]
key_files:
  created: []
  modified:
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/Sidebar.tsx
    - frontend/src/App.tsx
decisions:
  - "Display-toggle (flex/none) mount for HelpTab — NOT conditional render — to preserve scroll position and activeSection state across tab switches (mirrors SettingsTab pattern)"
  - "handleOpenHelp is idempotent: finds existing help tab and focuses it rather than duplicating __help__ tabs"
  - "HELP_TAB constant placed inside App() function alongside SETTINGS_TAB and HUB_TAB for consistency"
  - "HelpTab render gated on mode !== 'web' — same rationale as SettingsTab (Wails RPC unreachable in web-share viewer)"
metrics:
  duration: "~15 minutes"
  completed_date: "2026-06-22"
  tasks_completed: 3
  files_created: 0
  files_modified: 3
---

# Phase 147 Plan 03: App Shell Wiring (Wave 3) Summary

**One-liner:** Extended Tab type union with 'help', added Help as 4th sidebar item via QuestionMarkCircleIcon + onOpenHelp prop, and wired HELP_TAB/handleOpenHelp/display-toggle HelpTab render/exclusion/Sidebar prop in App.tsx — turning all 11 RED gates GREEN (1843 pass → 1854 pass).

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Extend Tab type union + add Help sidebar item | 4aefe341 | `TabBar.tsx`, `Sidebar.tsx` |
| 2 | Wire HELP_TAB + handleOpenHelp + render block in App.tsx | 55b6ebc6 | `App.tsx` |
| 3 | Phase build gate — tsc + vite build | (no source change) | verified only |

## What Was Built

**TabBar.tsx** — Tab.type discriminant union extended from `'terminal' | 'welcome' | 'settings' | 'file-browser' | 'hub'` to include `'help'`. Required for tsc to accept `HELP_TAB.type: 'help'` without error.

**Sidebar.tsx** — Three additions:
1. `QuestionMarkCircleIcon` added to the `@heroicons/react/24/outline` import block
2. `onOpenHelp: () => void` added to `SidebarProps` interface and destructured in `Sidebar()`
3. Help `<button>` added inside `sidebar__bottom` directly below Settings, using the same active-state pattern as Hub (`activePanel === '__help__'`), with `aria-label="Help"` and `QuestionMarkCircleIcon`

**App.tsx** — Five wiring changes:
1. `import { HelpTab } from './components/HelpTab'`
2. `const HELP_TAB: Tab = { id: '__help__', name: 'Help', sessionId: '', cli: '', type: 'help' }` alongside HUB_TAB
3. `handleOpenHelp` useCallback (find-or-add idempotent pattern, mirrors `handleOpenSettings`)
4. `onOpenHelp={handleOpenHelp}` on `<Sidebar>` usage
5. Display-toggle HelpTab render block (`{mode !== 'web' && <div style={{ display: activeId === HELP_TAB.id ? 'flex' : 'none', ... }}><HelpTab /></div>}`)
6. `'help'` added to tab-type exclusion list (prevents terminal rendering for help tab)

## Test Results

Wave 3 gates (11 previously RED): **all GREEN**

| Gate Type | Count | Status |
|-----------|-------|--------|
| App.tsx source gates (HELP_TAB constant, handleOpenHelp, id '__help__', type 'help') | 4 | GREEN |
| Sidebar 4-item count assertions (SIDE-01, SBR-01, GAP-03) | 3 | GREEN |
| Sidebar Help item describe block (render, click, active state) | 4 | GREEN |
| **Total** | **11** | **ALL GREEN** |

Full suite: **1854 passed / 0 failing** (was 1843 pass / 11 fail at Wave 2 end)

## Build Gate

`cd frontend && npx tsc && vite build` — exits 0.
- `?raw` Markdown imports (getting-started.md, faq.md) inlined into `index-CxMIv_DD.js` — verified via grep
- No separate `.md` network request at runtime
- No TypeScript errors

## Deviations from Plan

None — plan executed exactly as written. All five App.tsx edits applied per the interfaces section. No bugs encountered, no missing functionality, no architectural issues.

## Known Stubs

None. All components are fully wired and the Help page is reachable from the sidebar.

## Threat Flags

No new security-relevant surface. This plan adds only client-side React state routing (sidebar click → tab state). No new network endpoints, auth paths, file access patterns, or schema changes.

## Self-Check: PASSED

Files modified:
- [x] `frontend/src/components/TabBar.tsx` — 'help' in type union
- [x] `frontend/src/components/Sidebar.tsx` — QuestionMarkCircleIcon, onOpenHelp, Help button
- [x] `frontend/src/App.tsx` — HelpTab import, HELP_TAB, handleOpenHelp, render block, exclusion, Sidebar wiring

Commits verified:
- [x] 4aefe341 — feat(147-03): extend Tab type union with 'help' + add Help sidebar item
- [x] 55b6ebc6 — feat(147-03): wire HELP_TAB + handleOpenHelp + render block in App.tsx

Test state verified:
- [x] 1854 passing (full suite)
- [x] 0 failing
- [x] `npx tsc --noEmit` clean
- [x] `npx tsc && vite build` exits 0
