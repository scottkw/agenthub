---
phase: 84-session-auto-close
plan: "03"
subsystem: frontend-auto-close-tests
tags: [session, auto-close, exit-toast, countdown, react, tests, vitest]
dependency_graph:
  requires:
    - exit-toast-component       # from Plan 02
    - exit-countdown-banner-component  # from Plan 02
    - frontend-auto-close-flow   # from Plan 02
  provides:
    - exit-toast-unit-tests
    - exit-countdown-banner-unit-tests
    - app-exit-wiring-inspection-tests
  affects:
    - frontend/src/components/__tests__/ExitToast.test.tsx
    - frontend/src/components/__tests__/ExitCountdownBanner.test.tsx
    - frontend/src/components/__tests__/App.exit.test.tsx
tech_stack:
  added: []
  patterns:
    - createRoot + flushSync for synchronous React rendering in tests
    - Source inspection via ?raw import for wiring verification without mounting
    - afterEach cleanup: root.unmount() + container.remove() + vi.clearAllMocks()
key_files:
  created:
    - frontend/src/components/__tests__/ExitToast.test.tsx
    - frontend/src/components/__tests__/ExitCountdownBanner.test.tsx
    - frontend/src/components/__tests__/App.exit.test.tsx
  modified: []
decisions:
  - "Used autoCloseRef (not autoCloseEnabled) in App.exit.test.tsx — matches post-wave-2 fix where state was converted to a ref to avoid stale closures"
  - "Removed SetAutoCloseSession test from App.exit.test.tsx — it lives in SettingsTab.tsx, not App.tsx, per important_context note"
  - "13 ExitToast tests cover all BEM class variants and interaction paths — sufficient to detect regressions in countdown logic and dismiss behavior"
metrics:
  duration: "~2 minutes"
  completed: "2026-04-19"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 3
---

# Phase 84 Plan 03: Test Suite for Session Auto-Close Summary

Three test files verifying Phase 84 frontend components and wiring: ExitToast unit tests (13 cases covering clean/error exits, countdown, callbacks, accessibility), ExitCountdownBanner unit tests (5 cases), and App.tsx source inspection tests (12 cases confirming session:exit event wiring).

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | ExitToast and ExitCountdownBanner unit tests | f2981ec | ExitToast.test.tsx, ExitCountdownBanner.test.tsx |
| 2 | App.tsx source inspection tests for session:exit wiring | 27a2c12 | App.exit.test.tsx |

## What Was Built

### Task 1: Component Unit Tests

**frontend/src/components/__tests__/ExitToast.test.tsx (13 tests):**
- Empty state returns null (no `.exit-toast` element)
- Clean exit renders `.exit-toast__item--clean` class
- Error exit renders `.exit-toast__item--error` class
- Session name displayed in toast header
- Meta line contains CLI, exit status, final status
- Exit code and duration displayed
- Countdown and Keep Open button shown for clean exit with active countdown
- Countdown row hidden for error exits (countdown=-1)
- Countdown row hidden when cancelled=true
- `onKeepOpen` callback called with correct sessionId
- `onDismiss` callback called with correct sessionId
- Multiple toast items rendered for multiple exits
- Dismiss button has correct `aria-label` attribute
- Error exit shows "exited with error" in meta line

**frontend/src/components/__tests__/ExitCountdownBanner.test.tsx (5 tests):**
- Banner renders with countdown message ("Agent exited cleanly", countdown value)
- Keep Open button rendered with correct text
- `onKeepOpen` callback invoked on click
- `role="alert"` for accessibility
- Countdown value in dedicated `.exit-countdown-banner__countdown` span

### Task 2: App.tsx Source Inspection Tests

**frontend/src/components/__tests__/App.exit.test.tsx (12 tests):**
- `session:exit` Wails event subscription present
- `sessionExits` state defined
- `countdownTimers` ref defined
- `autoCloseRef` defined (ref-based to avoid stale closures)
- `ExitCountdownBanner` component rendered
- `ExitToast` component rendered
- `handleKeepOpen` callback defined
- `handleDismissExit` callback defined
- `GetAutoCloseSession` imported from Wails bindings
- `exitCountdowns` prop passed to TabBar
- `setSessionExits` cleanup in handleCloseTab
- `offExit` event unsubscription in useEffect cleanup

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Context Correction] Adjusted App.exit.test.tsx for post-wave-2 App.tsx changes**
- **Found during:** Task 2 execution (per important_context in prompt)
- **Issue:** Plan's Task 2 template checked for `autoCloseEnabled` (state) and `SetAutoCloseSession` in App.tsx, but wave 2 post-merge fix changed `autoCloseEnabled` state to `autoCloseRef` ref, and `SetAutoCloseSession` was moved to SettingsTab.tsx only
- **Fix:** Test checks for `autoCloseRef` instead of `autoCloseEnabled`; removed `SetAutoCloseSession` test (it's in SettingsTab, not App.tsx); replaced with `autoCloseRef` test
- **Files modified:** frontend/src/components/__tests__/App.exit.test.tsx
- **Commit:** 27a2c12

## Known Stubs

None. These are test files only — no components or data flows introduced.

## Threat Surface Scan

No new threat surface. Test files only — no network endpoints, auth paths, file access patterns, or schema changes.

## Self-Check: PASSED

All created files verified present:

- `frontend/src/components/__tests__/ExitToast.test.tsx`: present
- `frontend/src/components/__tests__/ExitCountdownBanner.test.tsx`: present
- `frontend/src/components/__tests__/App.exit.test.tsx`: present

Both commits verified:

- f2981ec: test(84-03): add ExitToast and ExitCountdownBanner unit tests
- 27a2c12: test(84-03): add App.tsx source inspection tests for session:exit wiring

Full test suite: 496 tests pass across 25 test files (pnpm test -- --run).
