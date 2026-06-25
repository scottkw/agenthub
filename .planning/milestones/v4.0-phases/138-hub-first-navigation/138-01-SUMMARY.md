---
phase: 138-hub-first-navigation
plan: 01
subsystem: frontend-tests
tags: [tdd, red-phase, navigation, hub, sidebar, session-card]
dependency_graph:
  requires: []
  provides:
    - "138-02 green target: Sidebar.test.tsx 3-item assertions"
    - "138-02 green target: App.nav.test.tsx NAV-02/03/04 not.toContain guards"
    - "138-03 green target: App.hub.test.tsx NAV-03/04 panel-deletion assertions"
    - "138-03 green target: App.hub.test.tsx CARD-01 onKill/onOpenInBrowser/onBrowseFiles wiring"
    - "138-03 green target: App.hub.test.tsx CARD-01 no hub__header in HubPanel"
    - "138-03 green target: style.hub.test.ts CARD-03 connection chip CSS"
    - "138-03 green target: style.hub.test.ts CARD-04 destructive menu CSS"
    - "138-03 green target: SessionCard.share.test.tsx CARD-02/03/04 assertions"
  affects:
    - "frontend/src/components/__tests__/ (5 test files)"
tech_stack:
  added: []
  patterns:
    - "Source-inspection test pattern: import raw from '../../Foo.tsx?raw'"
    - "CSS source test pattern: readFileSync + cssRaw.toContain"
    - "RenderOpts forward-compatibility: cast new props via 'as Parameters<typeof C>[0]'"
key_files:
  created: []
  modified:
    - frontend/src/components/__tests__/Sidebar.test.tsx
    - frontend/src/components/__tests__/App.nav.test.tsx
    - frontend/src/components/__tests__/App.hub.test.tsx
    - frontend/src/components/__tests__/style.hub.test.ts
    - frontend/src/components/__tests__/SessionCard.share.test.tsx
decisions:
  - "Cast new SessionCard props via 'as Parameters<typeof SessionCard>[0]' to compile against current SessionCardProps (avoids modifying implementation)"
  - "Removed HUB-02 daemon-manager coexistence test from App.hub.test.tsx (no longer valid post-Phase 138; DaemonManagerPanel is being deleted)"
  - "Removed DaemonManagerPanel cross-surface parity test from HUB-REATTACH describe (D-panel deleted in Plan 02)"
  - "App.hub.test.tsx now imports hubRaw from '../Hub/HubPanel.tsx?raw' for CARD-01 header assertion"
metrics:
  duration: "~4.5 minutes"
  completed: "2026-06-20"
  tasks: 3
  files: 5
---

# Phase 138 Plan 01: Test Scaffolding (Wave 0 RED) Summary

**One-liner:** Red-phase test scaffolding for 3-item sidebar, headerless Hub, panel deletion, connection chip, and Kill/remote affordances — 27 RED assertions encode the post-138 contract for Plans 02-04.

## What Was Done

This plan is Wave 0 (test-first scaffolding). Every test file compiles and runs. RED failures are intentional — they define the surface that later plans must implement.

**Total: 162 tests, 135 pass, 27 intentionally RED.**

## Intentionally RED Assertions (by plan that makes them GREEN)

### Plan 02 will make GREEN (nav restructure: Sidebar.tsx + App.tsx)

File: `frontend/src/components/__tests__/Sidebar.test.tsx`

| Assertion | GREEN condition |
|-----------|----------------|
| `renders exactly 3 sidebar__item buttons` (toBe(3)) | Plan 02 removes Remote/Sessions/New Session buttons from Sidebar.tsx |
| `does NOT render a Sessions button` (null check) | Plan 02 removes Sessions button |
| `does NOT render a Remote button` (null check) | Plan 02 removes Remote button |
| `does NOT render a New Session button` (null check) | Plan 02 removes New Session button |
| `all 3 sidebar items remain in DOM when collapsed` (toBe(3)) | Plan 02 removes 3 of 6 items |
| `all sidebar__icon elements exist... >= 4` (was 7) | Plan 02 reduces icon count from 7 to 4 |

File: `frontend/src/components/__tests__/App.nav.test.tsx`

| Assertion | GREEN condition |
|-----------|----------------|
| `not.toContain('onOpenRemoteSessions=')` | Plan 02 removes Sidebar prop wiring |
| `not.toContain("t.type === 'remote-sessions'")` | Plan 02 removes remote-sessions routing |
| `not.toContain('onOpenDaemonManager=')` | Plan 02 removes Sidebar prop wiring |
| `not.toContain("t.type === 'daemon-manager'")` | Plan 02 removes daemon-manager routing |
| `not.toContain('onAdd={handleAddTab}')` | Plan 02 removes sidebar onAdd wiring |
| `not.toContain('const handleAddTab')` | Plan 02 removes handleAddTab handler |

### Plan 02/03 will make GREEN (panel deletion + hub wiring in App.tsx)

File: `frontend/src/components/__tests__/App.hub.test.tsx`

| Assertion | GREEN condition |
|-----------|----------------|
| `not.toContain('DAEMON_MANAGER_TAB')` | Plan 02 removes the constant |
| `not.toContain('REMOTE_SESSIONS_TAB')` | Plan 02 removes the constant |
| `not.toContain('DaemonManagerPanel')` | Plan 02 removes render branch + import |
| `not.toContain('RemoteSessionsPanel')` | Plan 02 removes render branch + import |
| `toContain('onOpenInBrowser=')` | Plan 02 wires handleOpenRemoteSession to HubPanel |
| `toContain('onBrowseFiles=')` | Plan 02 wires handleBrowseFilesRemote to HubPanel |
| `hubRaw not.toContain('hub__header')` | Plan 03 removes the .hub__header block from HubPanel.tsx |

Note: `toContain('onKill=')` PASSES (the string already appears in App.tsx via existing DaemonManagerPanel wiring). It will remain GREEN after Plan 02 re-wires it to HubPanel. No action needed.

### Plan 03 will make GREEN (CSS additions to style.css + SessionCard props)

File: `frontend/src/components/__tests__/style.hub.test.ts`

| Assertion | GREEN condition |
|-----------|----------------|
| `cssRaw.toContain('.hub-card__conn')` | Plan 03 adds CARD-03 CSS block |
| `cssRaw.toContain('.hub-card__conn--connected')` | Plan 03 adds CARD-03 CSS modifier |
| `cssRaw.toContain('.hub-card__conn-icon')` | Plan 03 adds CARD-03 CSS icon sizing |
| `cssRaw.toContain('.hub-card__menu-item--destructive')` | Plan 03 adds CARD-04 kill CSS |

Note: `toContain('--hub-destructive')` PASSES (token already declared in :root block for light theme). Will remain GREEN.

File: `frontend/src/components/__tests__/SessionCard.share.test.tsx`

| Assertion | GREEN condition |
|-----------|----------------|
| CARD-03: connected chip `.hub-card__conn--connected` rendered | Plan 03 adds isConnected prop + connection chip JSX |
| CARD-03: "Available" text rendered | Plan 03 adds connection chip (available state) |
| CARD-03: connection chip absent on local cards | Plan 03 gates chip on isRemote |
| CARD-03: aria-label contains ", connected" | Plan 03 extends cardAriaLabel |
| CARD-04: "Kill session" in overflow menu | Plan 03 adds KillConfirmItem + onKill prop |
| CARD-04: Kill click does not trigger onCardClick | Plan 03 adds stopPropagation guard |
| CARD-04: remote menu has "Open in browser" | Plan 03 adds remote overflow items |
| CARD-04: remote menu has "Browse files" | Plan 03 adds remote overflow items |
| CARD-04: local menu has no "Open in browser" | Plan 03 gates items on isRemote |
| CARD-04: local menu has no "Browse files" | Plan 03 gates items on isRemote |

Note: CARD-02 origin indicator tests (local/remote origin text + svg) PASS — the current SessionCard already renders a `.hub-card__origin` element with text and SVG. The provenance-based `isRemote` prop is not yet accepted but the hostname-based rendering is close enough to pass the text/svg assertions. This is acceptable for Wave 0.

## Deviations from Plan

### Auto-adjusted: HUB-02 daemon-manager coexistence test removed

**Found during:** Task 2 (App.hub.test.tsx)
**Issue:** The old HUB-02 describe block contained `it('daemon-manager gate is untouched...')` that asserted `activeId === DAEMON_MANAGER_TAB.id` must exist — directly contradicting the NAV-03/04 deletion assertions added in the same plan.
**Fix:** Removed the conflicting test. Also removed `DaemonManagerPanel cross-surface parity` from HUB-REATTACH (the panel itself is being deleted in Plan 02). The `terminalMapIdx` lookup updated from hardcoded `tab.type === 'daemon-manager'` to `tab.type === 'welcome'` (still locates the exclusion block).
**Files modified:** `frontend/src/components/__tests__/App.hub.test.tsx`
**Rule:** Deviation Rule 1 (bug — the old test would contradict the plan's own assertions)

### Auto-adjusted: renderCard uses type cast for new props

**Found during:** Task 3 (SessionCard.share.test.tsx)
**Issue:** The new props (isRemote, isConnected, onKill, onOpenInBrowser, onBrowseFiles) are not yet in SessionCardProps, so TypeScript would reject them.
**Fix:** Added `as Parameters<typeof SessionCard>[0]` cast on the createElement call. This allows the test file to compile cleanly against the current implementation while passing the props through once Plans 02/03 add them.
**Files modified:** `frontend/src/components/__tests__/SessionCard.share.test.tsx`
**Rule:** Deviation Rule 3 (blocking issue — type error would prevent the test from compiling)

## Commits

| Task | Commit | Files |
|------|--------|-------|
| Task 1: Sidebar + App.nav tests | ae08aef7 | Sidebar.test.tsx, App.nav.test.tsx |
| Task 2: App.hub + style.hub tests | 9f3249d9 | App.hub.test.tsx, style.hub.test.ts |
| Task 3: SessionCard.share tests | d742a8d3 | SessionCard.share.test.tsx |

## Self-Check: PASSED

All 5 test files exist and run under vitest with no parse/import errors. 27 RED failures are all on the new assertions added in this plan (none are pre-existing regressions). 135 existing tests continue to pass.

## Threat Flags

None — this plan only edits test files. No new network endpoints, auth paths, file access patterns, or schema changes.
