---
phase: 81-banner-notifications
plan: 02
subsystem: frontend/app-root
tags: [banner-stack, dismiss, update-state, integration, tests]
dependency_graph:
  requires:
    - "LocalNetworkBanner with onDismiss/className props (81-01)"
    - "UpdateBanner standalone component (81-01)"
    - "Banner stack CSS rules (81-01)"
  provides:
    - "BannerStack integration in App.tsx with vertical stacking"
    - "Lifted update state from WelcomeTab to App.tsx"
    - "Independent dismiss handlers with 200ms exit animation"
    - "D-04 dismissed-state reset on webServerMode change"
    - "Full test coverage for banner components and integration"
  affects:
    - "frontend/src/App.tsx"
    - "frontend/src/components/__tests__/LocalNetworkBanner.test.tsx"
    - "frontend/src/components/__tests__/UpdateBanner.test.tsx"
    - "frontend/src/components/__tests__/App.test.tsx"
    - "frontend/src/components/__tests__/WelcomeTab.test.tsx"
    - "frontend/vite.config.ts"
tech_stack:
  added: []
  patterns:
    - "BannerStack vertical stacking via .banner-stack CSS container"
    - "Independent dismiss with useCallback + setTimeout exit animation"
    - "Lifted state pattern (update state from WelcomeTab to App root)"
    - "Vite resolve alias for wailsjs runtime stub in test environment"
key_files:
  created:
    - "frontend/src/components/__tests__/UpdateBanner.test.tsx"
  modified:
    - "frontend/src/App.tsx"
    - "frontend/src/components/__tests__/LocalNetworkBanner.test.tsx"
    - "frontend/src/components/__tests__/App.test.tsx"
    - "frontend/src/components/__tests__/WelcomeTab.test.tsx"
    - "frontend/vite.config.ts"
decisions:
  - "Added vite resolve alias for wailsjs/wailsjs/runtime/runtime to enable DOM rendering tests for UpdateBanner (wailsjs runtime is generated at build time, not available in test environment)"
  - "Placed update subscription useEffect after remote sessions polling effect to maintain logical grouping of polling effects"
  - "Used parenthesized condition ((webServerMode === 'local' && !localBannerDismissed) || update) for correct operator precedence in BannerStack visibility"
metrics:
  duration: "5m"
  completed: "2026-04-16"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 6
---

# Phase 81 Plan 02: BannerStack Integration and Test Updates Summary

BannerStack wired into App.tsx with lifted update state, independent dismiss handlers with 200ms exit animation, and comprehensive test coverage across 4 test files (434 tests passing).

## Task Results

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Wire BannerStack into App.tsx | 39f3e9c | App.tsx |
| 2 | Update all test files | 48f9ce3 | LocalNetworkBanner.test.tsx, UpdateBanner.test.tsx, App.test.tsx, WelcomeTab.test.tsx, vite.config.ts |

## Changes Made

### Task 1: Wire BannerStack into App.tsx

**Imports added:**
- `import { UpdateBanner } from './components/UpdateBanner'`
- `import type { UpdateInfo } from './components/UpdateBanner'`
- `GetLastUpdateInfo` added to wailsjs import block

**State declarations added:**
- `update: UpdateInfo | null` -- lifted from WelcomeTab (D-06)
- `localBannerDismissed: boolean` -- session-only dismiss state (D-04)
- `localBannerExiting: boolean` -- exit animation flag
- `updateExiting: boolean` -- exit animation flag

**Callbacks added:**
- `handleDismissLocalBanner` -- sets exiting flag, then after 200ms sets dismissed and clears exiting
- `handleDismissUpdate` -- sets exiting flag, then after 200ms sets update to null and clears exiting

**Effects added:**
- Update subscription: calls `GetLastUpdateInfo()` on mount, subscribes to `update:available` event via `EventsOn`
- D-04 reset: when `webServerMode` transitions to `'local'`, resets `localBannerDismissed` to false

**JSX replaced:**
- Single `<LocalNetworkBanner>` render replaced with `<div className="banner-stack">` containing both `<LocalNetworkBanner>` (with `onDismiss` and `className` props) and `<UpdateBanner>` (with `onDismiss` and `className` props)
- Banner order: LocalNetworkBanner first (top), UpdateBanner second (bottom)

### Task 2: Update All Test Files

**LocalNetworkBanner.test.tsx (5 new tests):**
- Test dismiss button renders when onDismiss provided
- Test dismiss button absent when onDismiss not provided
- Test onDismiss called on click
- Test className applied to banner element
- Test dismiss button aria-label correctness

**UpdateBanner.test.tsx (9 new tests, new file):**
- Tests version information rendering
- Tests Download Update button presence
- Tests Dismiss button with aria-label
- Tests onDismiss callback on click
- Tests role="alert" accessibility
- Tests aria-live="polite"
- Tests className application for banner-exit animation
- Tests BrowserOpenURL called with releaseURL on download
- Tests "Update available:" message text

**App.test.tsx (18 new tests):**
- BannerStack integration describe block testing all structural requirements
- Tests imports (UpdateBanner, UpdateInfo, GetLastUpdateInfo)
- Tests state declarations (update, localBannerDismissed, localBannerExiting, updateExiting)
- Tests callbacks (handleDismissLocalBanner, handleDismissUpdate)
- Tests JSX structure (banner-stack div, both banners inside, onDismiss props, className exit animation)
- Tests event subscription (update:available)
- Tests D-04 reset (setLocalBannerDismissed(false))

**WelcomeTab.test.tsx (cleanup):**
- Removed entire `describe('update banner (UPD-02, UPD-03)')` block (15 tests) -- these assertions are now covered by UpdateBanner.test.tsx and App.test.tsx
- Fixed version import assertion from `GetVersion, GetLastUpdateInfo` to just `GetVersion`

**vite.config.ts (test infrastructure):**
- Added resolve aliases mapping `wailsjs/wailsjs/runtime/runtime` paths to the existing `src/wailsjs/runtime/runtime.js` stub
- Enables DOM rendering tests for components that import from the Wails-generated runtime path

## Verification Results

- TypeScript compilation: passes (no new errors; only pre-existing wailsjs module resolution warnings)
- Test suite: 21 files, 434 tests, all passing
- Previous 12 failing WelcomeTab tests: resolved (removed outdated assertions, replaced with new coverage)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added vite.config.ts resolve alias for wailsjs runtime**
- **Found during:** Task 2
- **Issue:** UpdateBanner.test.tsx could not render the component because `wailsjs/wailsjs/runtime/runtime` is a Wails-generated path that doesn't exist at test time. `vi.mock` alone was insufficient because Vite resolves imports before vitest mock hoisting.
- **Fix:** Added resolve aliases in vite.config.ts mapping the doubled `wailsjs/wailsjs/runtime/runtime` paths to the existing `src/wailsjs/runtime/runtime.js` stub file. This allows `vi.mock` to intercept and override the resolved module.
- **Files modified:** `frontend/vite.config.ts`
- **Commit:** 48f9ce3

## Self-Check: PASSED

- [x] frontend/src/App.tsx contains banner-stack, UpdateBanner, handleDismissLocalBanner, handleDismissUpdate
- [x] frontend/src/components/__tests__/UpdateBanner.test.tsx exists with 9 tests
- [x] frontend/src/components/__tests__/LocalNetworkBanner.test.tsx has 5 new dismiss tests
- [x] frontend/src/components/__tests__/App.test.tsx has BannerStack integration block with 18 tests
- [x] frontend/src/components/__tests__/WelcomeTab.test.tsx no longer contains update banner describe block
- [x] Commit 39f3e9c exists (Task 1)
- [x] Commit 48f9ce3 exists (Task 2)
- [x] pnpm test passes: 434 tests, 21 files, 0 failures
