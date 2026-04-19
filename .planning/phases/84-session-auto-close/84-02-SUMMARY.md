---
phase: 84-session-auto-close
plan: "02"
subsystem: frontend-auto-close
tags: [session, auto-close, exit-toast, countdown, react, wails-events]
dependency_graph:
  requires:
    - session-exit-detection-pipeline  # from Plan 01
    - auto-close-session-setting       # from Plan 01
  provides:
    - exit-toast-component
    - exit-countdown-banner-component
    - tab-countdown-indicator
    - auto-close-settings-toggle
    - frontend-auto-close-flow
  affects:
    - frontend/src/App.tsx
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/components/ExitToast.tsx
    - frontend/src/components/ExitCountdownBanner.tsx
    - frontend/src/style.css
tech_stack:
  added: []
  patterns:
    - useRef for handleCloseTabRef to avoid stale closure in [] useEffect
    - setAutoCloseEnabled functional update trick to read current state in [] useEffect
    - countdownTimers ref keyed by sessionId — bounded by open session count (T-84-05)
    - exitCountdowns derived map passed as prop to TabBar (no extra state)
key_files:
  created:
    - frontend/src/components/ExitToast.tsx
    - frontend/src/components/ExitCountdownBanner.tsx
  modified:
    - frontend/src/App.tsx
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/style.css
decisions:
  - "Used handleCloseTabRef pattern (ref updated after each render) to call latest handleCloseTab from [] useEffect without stale closure"
  - "Used setAutoCloseEnabled functional update trick to read current autoCloseEnabled value inside the session:exit event handler without adding it to deps"
  - "exitCountdowns passed as derived Object.fromEntries map to TabBar — avoids separate state for what is already derivable from sessionExits"
  - "D-12 guard: handleCloseTab checks !sessionExits[id] before ToggleWebServing(false) — naturally-exited sessions skip immediate disable, letting daemon's 10s timer handle web shutdown"
metrics:
  duration: "~25 minutes"
  completed: "2026-04-19"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 6
---

# Phase 84 Plan 02: Frontend Auto-Close Summary

React frontend for session auto-close: `session:exit` Wails event triggers countdown state in App.tsx, ExitToast shows fixed-position per-session notifications (clean/error variants), ExitCountdownBanner shows inline countdown inside terminal area, TabBar displays countdown badge with opacity fade, SettingsTab "Session Behavior" toggle persists preference via GetAutoCloseSession/SetAutoCloseSession Wails bindings.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | ExitToast, ExitCountdownBanner components + CSS | 5cecb50 | ExitToast.tsx, ExitCountdownBanner.tsx, style.css |
| 2 | App.tsx event handler, countdown state, TabBar, SettingsTab | e724048 | App.tsx, TabBar.tsx, SettingsTab.tsx |

## What Was Built

### Task 1: Components + CSS

**frontend/src/components/ExitToast.tsx:**
- `ExitState` interface exported: sessionId, sessionName, cli, exitCode, duration, finalStatus, countdown (-1 = no countdown), cancelled
- `ExitToast` component: fixed-position container (z-index 9999, bottom-right), one card per entry in `exits` map
- Clean exits: green left border (`.exit-toast__item--clean`), countdown badge, Keep Open button
- Error exits: red left border (`.exit-toast__item--error`), no countdown row, persists until dismissed
- Dismiss button uses XMarkIcon with accessible aria-label `"Dismiss exit notification for {sessionName}"`

**frontend/src/components/ExitCountdownBanner.tsx:**
- Inline banner rendered inside terminal-wrapper between TerminalPanel and StatusBar
- Shows "Agent exited cleanly. Tab closes in {n}s." with Keep Open button
- Follows UpdateBanner pattern (role="alert", aria-live="polite")

**frontend/src/style.css (additions):**
- `.exit-toast`: fixed container, column flex, gap 8px, pointer-events none
- `.exit-toast__item`: TokyoNight surface (#1e2030), border (#292e42), BEM children
- `.exit-toast__item--clean/--error`: green/red left border accents
- `.exit-countdown-banner`: deep background (#16161e), green left border, flex row
- `.tab--exiting`: opacity 0.5, 150ms ease transition
- `.tab__countdown`: 11px, #9ece6a, tabular-nums, min-width 18px

### Task 2: App.tsx, TabBar.tsx, SettingsTab.tsx

**App.tsx state additions:**
- `sessionExits: Record<string, ExitState>` — per-session exit map driving all three UI surfaces
- `countdownTimers: useRef<Record<string, ReturnType<typeof setInterval>>>` — keyed by sessionId
- `autoCloseEnabled: boolean` — loaded on mount via `GetAutoCloseSession()`
- `handleCloseTabRef: useRef` — keeps latest `handleCloseTab` accessible from `[]` useEffect

**session:exit event subscription (in `[]` useEffect):**
- Always adds entry to `sessionExits` (drives toast for any exit)
- For `exitCode === 0`: reads `autoCloseEnabled` via functional update trick; if enabled, starts 1-second interval that decrements countdown 5→0; at 0, calls `handleCloseTabRef.current(sessionId)` and removes entry
- If auto-close disabled: sets countdown to -1 (toast shows without countdown row)
- Cleanup: `offExit()` + clears all countdown timers

**handleKeepOpen / handleDismissExit:**
- `handleKeepOpen`: clears timer, sets `{cancelled: true, countdown: -1}` — toast stays visible as static notification, banner unmounts, tab opacity restores
- `handleDismissExit`: clears timer, removes entry from sessionExits entirely

**handleCloseTab updates (D-12):**
- Web disable guard: `if (webEnabled[id] && !sessionExits[id])` — skips `ToggleWebServing(false)` for naturally-exited sessions; daemon's 10s AfterFunc handles web shutdown
- Cleanup: clears countdown timer and removes sessionExits entry
- Deps updated: `[activeId, webEnabled, sessionExits]`

**JSX additions:**
- TabBar receives `exitCountdowns` map (derived from sessionExits, exitCode=0, not cancelled, countdown>0)
- ExitCountdownBanner rendered between TerminalPanel and StatusBar when session has active countdown
- ExitToast rendered at root app level (visible from any tab)

**TabBar.tsx:**
- `exitCountdowns?: Record<string, number>` added to `TabBarProps`
- `.tab--exiting` class conditional on `exitCountdowns?.[tab.sessionId]`
- `<span className="tab__countdown">{exitCountdowns[tab.sessionId]}s</span>` badge rendered after tab name

**SettingsTab.tsx:**
- Imports `GetAutoCloseSession`, `SetAutoCloseSession`
- State: `autoCloseSession`, `autoCloseLoaded`, `autoCloseSaving`, `autoCloseError`
- useEffect loads preference on mount
- `handleToggleAutoClose` saves via `SetAutoCloseSession` with error handling
- "Session Behavior" h3 section with toggle row (reuses `.settings-panel__toggle-*` CSS), description, error display — placed after "Behavior" section, before "Appearance"

## Deviations from Plan

None — plan executed exactly as written. All implementation decisions matched plan specifications.

## Known Stubs

None. All component props receive live data from `sessionExits` state which is populated by the `session:exit` Wails event. `countdown` ticks down from real setInterval calls. `autoCloseEnabled` loaded from real `GetAutoCloseSession()` binding.

## Threat Surface Scan

No new threat surface beyond what the plan's threat model covers.

- T-84-05 (countdown timer DoS): Mitigated — timers keyed by sessionId, cleaned up in `handleCloseTab` and useEffect cleanup. Maximum concurrent timers equals number of open terminal sessions (bounded by open tabs).
- No new network endpoints, auth paths, file access patterns, or schema changes introduced.

## Self-Check: PASSED

All created/modified files verified present:

- `frontend/src/components/ExitToast.tsx`: present
- `frontend/src/components/ExitCountdownBanner.tsx`: present
- `frontend/src/style.css`: modified (contains `.exit-toast`, `.exit-countdown-banner`, `.tab--exiting`)
- `frontend/src/App.tsx`: modified (contains `sessionExits`, `session:exit` subscription, `ExitToast`, `ExitCountdownBanner`)
- `frontend/src/components/TabBar.tsx`: modified (contains `exitCountdowns`, `tab--exiting`)
- `frontend/src/components/SettingsTab.tsx`: modified (contains `Session Behavior`, `Auto-close tab on exit`)

Both commits verified: 5cecb50, e724048.

TypeScript compilation: PASS (no errors).
