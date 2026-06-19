---
phase: 131-hub-foundation-static-session-cards
plan: "05"
subsystem: frontend/Hub
tags: [react, typescript, vitest, hub, css, sidebar, app-wiring, colorblind-safe, reduced-motion, wcag]
dependency_graph:
  requires: [131-04]
  provides: [HubPanel-wiring, Hub-CSS, Sidebar-Hub-item, App-HUB_TAB]
  affects: [App.tsx (session polling, tab routing), Sidebar.tsx (Hub item + active state), style.css (Hub theme system)]
tech_stack:
  added: []
  patterns: [BEM CSS custom properties, prefers-reduced-motion guard, CSS token system dark+light, source-inspection tests via ?raw, CSS-text contract tests]
key_files:
  created:
    - frontend/src/components/__tests__/App.hub.test.tsx
    - frontend/src/components/__tests__/style.hub.test.ts
  modified:
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/Sidebar.tsx
    - frontend/src/components/__tests__/Sidebar.test.tsx
    - frontend/src/App.tsx
    - frontend/src/style.css
    - frontend/src/components/Hub/HubFilterBar.tsx
    - frontend/src/components/__tests__/DaemonManagerPanel.test.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/components/Hub/InlineSessionName.test.tsx
    - frontend/src/components/Hub/SessionCard.test.tsx
    - frontend/src/components/Hub/SessionCardGrid.test.tsx
decisions:
  - "Hub tab uses id '__hub__' with type 'hub' — mirrors DAEMON_MANAGER_TAB and REMOTE_SESSIONS_TAB pattern"
  - "Two terminal-exclusion sites use different variable names (t.type vs tab.type); App.hub.test.tsx asserts both patterns separately"
  - "RefObject<HTMLInputElement | null> adopted in HubFilterBar (React 19 useRef returns null-typed ref) — enables future React 19 strict compat"
  - "hub-spin @keyframes declared at root scope; animation only fires when spin class is applied inside the prefers-reduced-motion media query"
metrics:
  duration: "~25 minutes"
  completed: "2026-06-16"
  tasks_completed: 2
  tasks_total: 2
  files_created: 2
  files_modified: 11
  tests_added: 87
---

# Phase 131 Plan 05: Hub App Shell Integration + CSS Summary

Hub wired into the app shell as a reachable, coexisting, themed top-level surface: HUB_TAB constant + handleOpenHub handler + 3s poll useEffect + HubPanel render in App.tsx; Squares2X2Icon Hub button with active-state indicator in Sidebar.tsx; full Phase 131 CSS token system (dark + light themes, responsive grid, dim rule, reduced-motion-guarded spin, sidebar active) appended to style.css — proven by 87 new tests (49 wiring + 38 CSS-contract), full suite 1473/1473 green, clean typecheck.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | TabBar type + Sidebar Hub item + App.tsx wiring | 6081673 | TabBar.tsx, Sidebar.tsx, Sidebar.test.tsx, App.tsx, App.hub.test.tsx + pre-existing type error fixes |
| 2 | Hub CSS + CSS-contract test | 4e852c0 | style.css, style.hub.test.ts |

## What Was Built

### Task 1: App Shell Wiring

**TabBar.tsx:** Added `'hub'` to the Tab `type?` union.

**Sidebar.tsx:** Added `Squares2X2Icon` import; added `onOpenHub: () => void` and `activePanel?: string` to `SidebarProps`; added Hub `<button>` with `sidebar__item--active` conditional class when `activePanel === '__hub__'`.

**App.tsx:**
- `HUB_TAB: Tab` constant with `id: '__hub__'` and `type: 'hub'`
- `hubSessions: SessionInfo[]` and `hubError: boolean` state
- `handleOpenHub` callback (find-or-create pattern, mirrors `handleOpenRemoteSessions`)
- Hub poll `useEffect` gated on `activeId === HUB_TAB.id` + `mode !== 'web'` (3s interval, cancelled flag, `setHubSessions`/`setHubError`) — satisfies T-131-10 DoS prevention
- `onOpenHub={handleOpenHub}` and `activePanel={activeId ?? undefined}` passed to `<Sidebar>`
- `<HubPanel sessions={hubSessions} error={hubError} onNewSession={() => setShowNewSessionModal(true)} onRename={handleRenameTab} />` rendered when `activeId === HUB_TAB.id`
- `t.type !== 'hub'` added to the `daemonError` empty-filter (first terminal-exclusion site)
- `tab.type === 'hub'` added to the terminal map early-return (second terminal-exclusion site)
- `import { HubPanel } from './components/Hub/HubPanel'`

**App.hub.test.tsx:** 24 source-inspection assertions covering HUB_TAB constant, Sidebar wiring, poll gating, HubPanel render props, and coexistence (HUB-02 — both terminal-exclusion sites verified).

**Sidebar.test.tsx:** 5 new Hub item tests (button present, fires onOpenHub, active class applied when `activePanel === '__hub__'`, not active when other panel active). Updated existing item/icon count tests from 5→6 items and 6→7 icons.

### Task 2: Hub CSS

**style.css:** 370-line Phase 131 Hub block appended after existing file-browser rules:

- `:root` dark theme block with all 19 `--hub-*` custom properties
- `[data-ui-theme="light"]` light theme block with all 19 overrides including HUB-04 WCAG AA inline comments for `--hub-accent` (#3d6fe8 = 4.5:1) and `--hub-destructive` (#c0394f = 4.7:1)
- 12 COLORBLIND-SAFE source comments for all status dot hex constants (dark + light), with WCAG contrast ratios
- Hub surface layout: `.hub`, `.hub__header`, `.hub__grid-scroll`, `.hub__group`
- `.hub__group-header` (11px/600/uppercase/letter-spacing 0.08em — mirrors `.remote-panel__peer-header`)
- GRID-01: `.hub__card-row` with `repeat(auto-fill, minmax(240px, 1fr))`, gap 8px, max-width 1440px
- `.hub-card` (min-width 240px, max-width 360px, border `var(--hub-border)`, radius 8px, hover + focus rules)
- CARD-08: `.hub-card--dim` with `opacity: var(--hub-dim-opacity)` + `background: var(--hub-card-dim-bg)`; "Error-exit cards are NOT dimmed" comment
- Running spin under `@media (prefers-reduced-motion: no-preference)` guard; `@keyframes hub-spin` at root scope
- Filter pill, search input, empty state, error state, badge, origin, viewer, uptime, exit chip CSS
- Pitfall-8: `.sidebar__item--active` with `var(--hub-accent, #7aa2f7)` fallback

**style.hub.test.ts:** 38 CSS-contract assertions covering dark/light tokens, GRID-01 grid declaration, CARD-08 dim rule, reduced-motion-guarded spin, sidebar active class, group header typography, and source-level colorblind-safe comments.

## Test Results

```
Test Files  95 passed (95)
      Tests 1473 passed (1473)
```

Full suite green. New tests: 49 wiring (App.hub.test.tsx + Sidebar.test.tsx Hub describe) + 38 CSS-contract (style.hub.test.ts) = 87 new tests.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-existing type errors blocked `pnpm exec tsc --noEmit` clean**

- **Found during:** Task 1 (typecheck verification)
- **Issue:** `DaemonManagerPanel.test.tsx` local `SessionInfo` interface was missing `workDir: string` (added to `App.d.ts` in Plan 01/02). `HubFilterBar.tsx` used `React.RefObject<HTMLInputElement>` but React 19 `useRef` returns `RefObject<HTMLInputElement | null>`. Four Hub test files had unused `import React from 'react'` (tsconfig `noUnusedLocals: true`, `jsx: react-jsx` handles JSX transform).
- **Fix:** Added `workDir: string` to DaemonManagerPanel test interface and all 9 inline session fixtures; changed `RefObject<HTMLInputElement>` to `RefObject<HTMLInputElement | null>` in HubFilterBar.tsx; removed unused `import React from 'react'` from HubPanel.test.tsx, InlineSessionName.test.tsx, SessionCard.test.tsx, SessionCardGrid.test.tsx.
- **Files modified:** DaemonManagerPanel.test.tsx, HubFilterBar.tsx, HubPanel.test.tsx, InlineSessionName.test.tsx, SessionCard.test.tsx, SessionCardGrid.test.tsx
- **Commit:** 6081673

**2. [Rule 1 - Bug] App.hub.test.tsx grep count pattern mismatch**

- **Found during:** Task 1 (first test run)
- **Issue:** Initial test asserted `t.type !== 'hub'` appears ≥ 2 times, but the two terminal-exclusion sites use different variable names: `t.type !== 'hub'` (filter) and `tab.type === 'hub'` (terminal map early-return). The second site uses a positive inclusion check with `tab.type`.
- **Fix:** Split into two separate assertions checking `t.type !== 'hub'` (≥ 1) and `tab.type === 'hub'` (≥ 1) respectively.
- **Files modified:** App.hub.test.tsx
- **Commit:** 6081673

**3. [Rule 1 - Bug] style.hub.test.ts comment text mismatch**

- **Found during:** Task 2 (CSS test run)
- **Issue:** Test asserted `'Error cards are NOT dimmed'` but the CSS comment text is `'Error-exit cards are NOT dimmed'`.
- **Fix:** Updated test assertion to match actual CSS comment text.
- **Files modified:** style.hub.test.ts
- **Commit:** 4e852c0

## Threat Mitigations Applied

- **T-131-10 (DoS — Hub poll active while Hub inactive):** Poll `useEffect` early-returns when `activeId !== HUB_TAB.id` and when `mode === 'web'`. Verified by App.hub.test.tsx `HUB-POLL: Hub poll is gated on HUB_TAB.id` assertions.
- **T-131-11 (Tampering — Hub replaces Sessions panel):** HUB_TAB is a distinct tab object; `'hub'` is added to terminal-exclusion sites only. The daemon-manager gate (`activeId === DAEMON_MANAGER_TAB.id`) is untouched. Verified by App.hub.test.tsx `HUB-02 coexistence` assertions.
- **T-131-12 (npm installs):** No new packages added.

## Known Stubs

None. HubPanel receives live `hubSessions` from App.tsx's ListSessions() poll (3s interval, gated on active Hub tab). The CSS theme system uses real design tokens from the UI-SPEC.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries. The Hub poll uses the existing `ListSessions()` RPC already used by the DaemonManagerPanel poll.

## Self-Check: PASSED

Files exist:
- frontend/src/components/__tests__/App.hub.test.tsx: FOUND
- frontend/src/components/__tests__/style.hub.test.ts: FOUND
- frontend/src/components/TabBar.tsx: FOUND (modified)
- frontend/src/components/Sidebar.tsx: FOUND (modified)
- frontend/src/App.tsx: FOUND (modified)
- frontend/src/style.css: FOUND (modified)

Commits exist:
- 6081673: feat(131-05): TabBar type + Sidebar Hub item + App.tsx wiring
- 4e852c0: feat(131-05): Hub CSS (theme tokens, grid, dim, reduced-motion, sidebar active) + CSS-contract test

Acceptance criteria:
- `grep -c "t.type !== 'hub'" frontend/src/App.tsx` = 1 (daemonError filter site)
- `grep -c "tab.type.*'hub'" frontend/src/App.tsx` = 1 (terminal map site; both exclusion sites verified by App.hub.test.tsx)
- `pnpm exec tsc --noEmit` = clean (no output)
- `pnpm vitest run` = 1473/1473 tests pass
- `grep -c "prefers-reduced-motion" frontend/src/style.css` = 14 (multiple existing + new hub guard)
- `grep -c 'data-ui-theme="light"' frontend/src/style.css` = 2
- `grep -c "repeat(auto-fill, minmax(240px, 1fr))" frontend/src/style.css` = 1
