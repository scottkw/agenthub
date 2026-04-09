---
phase: 60-local-network-fallback
plan: 03
subsystem: frontend
tags: [local-network, banner, settings, ux, react]
dependency_graph:
  requires: ["60-02"]
  provides: ["LocalNetworkBanner component", "LAN password display in Settings", "HealthModal local-mode fix"]
  affects: ["frontend/src/App.tsx", "frontend/src/components/SettingsPanel.tsx", "frontend/src/components/HealthModal.tsx"]
tech_stack:
  added: []
  patterns: ["conditional render on webServerMode state", "polling Wails binding for mode state", "click-to-copy with timed feedback"]
key_files:
  created:
    - frontend/src/components/LocalNetworkBanner.tsx
    - frontend/src/components/__tests__/LocalNetworkBanner.test.tsx
  modified:
    - frontend/src/App.tsx
    - frontend/src/components/SettingsPanel.tsx
    - frontend/src/components/HealthModal.tsx
    - frontend/src/style.css
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
decisions:
  - "Used createRoot/flushSync test pattern (not @testing-library/react) to match existing codebase convention"
  - "SettingsPanel.tsx is the correct file (not SettingsTab.tsx — plan referred to an earlier name)"
  - "Sidebar.test.tsx localStorage failures are pre-existing and out of scope"
metrics:
  duration: "4 minutes"
  completed: "2026-04-09T20:15:57Z"
  tasks_completed: 2
  files_changed: 8
---

# Phase 60 Plan 03: Frontend Local Network UI Summary

**One-liner:** Persistent amber nudge banner above sidebar+content row with app__row layout restructure, LAN password display with click-to-copy in SettingsPanel web-server tab, and HealthModal suppression when web server is running in any mode.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create LocalNetworkBanner component with tests (TDD) | ebcb762 | LocalNetworkBanner.tsx, LocalNetworkBanner.test.tsx, style.css |
| 2 | Integrate banner into App.tsx layout, add password to SettingsPanel, fix HealthModal | 53e5beb | App.tsx, SettingsPanel.tsx, HealthModal.tsx, style.css, App.d.ts, App.js |

## What Was Built

### LocalNetworkBanner Component
- New `frontend/src/components/LocalNetworkBanner.tsx` with `visible` and `onOpenURL` props
- Returns `null` when `visible=false` (no DOM presence, no animation)
- Warning amber left border (`border-left: 3px solid #f59e0b`) matching UI spec
- `role="status"` for accessibility
- `⚠` icon in amber, message text, secondary text, "Install Tailscale" CTA button
- CTA calls `onOpenURL('https://tailscale.com/download')`
- 5 unit tests using existing `createRoot`/`flushSync` pattern — all passing

### App.tsx Layout Restructure
- Added `webServerMode` state (`'tailscale' | 'local' | null`)
- `GetWebServerMode()` polled during `init()` and `retryInit()` alongside `IsWebServerRunning()`
- `.app` `flex-direction` changed from `row` to `column` to support banner above row
- New `.app__row` div wraps Sidebar + `.app__content` (maintains `flex-direction: row`)
- `LocalNetworkBanner` rendered above `.app__row` when `webServerMode === 'local'`
- `webServerMode` prop passed to SettingsPanel
- `webServerRunning` prop passed to HealthModal

### SettingsPanel Web Server Tab — LAN Password
- New `webServerMode?` prop added to `SettingsPanelProps`
- `GetLocalNetworkPassword()` imported and called via `useEffect` when `webServerMode === 'local' && isServerRunning`
- Click-to-copy: copies password to clipboard, shows "Copied!" for 1500ms then reverts
- Password shown unmasked (desktop app, user needs to type it in browser)
- Mode indicator: "Web server mode: Local network (self-signed TLS)"
- Label: "LAN Access Password" with "(click to copy)" hint

### HealthModal Suppression
- Added `webServerRunning?: boolean` prop to `HealthModalProps`
- Early `if (webServerRunning) return null` after the `if (health === null) return null` guard
- When web server is running (any mode), HealthModal stays hidden — nudge banner handles messaging

### Wails Binding Stubs
- `App.d.ts`: added `GetLocalNetworkPassword(): Promise<string>` and `GetWebServerMode(): Promise<string>`
- `App.js`: added corresponding `Call('main.App.GetLocalNetworkPassword', [])` and `Call('main.App.GetWebServerMode', [])` exports

## CSS Added

All new CSS appended to `frontend/src/style.css`:
- `.local-network-banner` — amber left border, flex row, 12px vertical padding
- `.local-network-banner__icon`, `__message`, `__sub`, `__cta`, `__cta:hover`
- `.app__row` — flex row, flex 1, min-height 0 (replaces old `.app` row direction)
- `.settings-web-server__password-label`, `__password-field`, `__password-field:hover`
- `.settings-web-server__mode-indicator`, `__copy-hint`, `__copy-hint--copied`

## Test Results

```
Test Files: 14 passed, 1 pre-existing failure (Sidebar.test.tsx — localStorage jsdom issue, out of scope)
Tests:      260 passed, 13 pre-existing failures (all in Sidebar.test.tsx)
LocalNetworkBanner tests: 5/5 passing
```

Go build: `go build ./...` exits 0.

## Deviations from Plan

### Auto-adjusted: SettingsTab vs SettingsPanel
- **Found during:** Task 2
- **Issue:** Plan referred to `frontend/src/components/SettingsTab.tsx` but the actual file in the codebase is `SettingsPanel.tsx` (the component was renamed in a prior phase)
- **Fix:** Applied all SettingsTab changes to `SettingsPanel.tsx` instead
- **Files modified:** `frontend/src/components/SettingsPanel.tsx`

### Auto-adjusted: Test pattern (Rule 2 — consistency)
- **Found during:** Task 1
- **Issue:** Plan specified `@testing-library/react` for tests, but entire test suite uses `createRoot`/`flushSync` pattern (no @testing-library installed)
- **Fix:** Used existing `createRoot`/`flushSync` pattern for consistency with codebase — same behavior, no new dependency
- **Files modified:** `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx`

### Pre-existing test failures (out of scope)
- `Sidebar.test.tsx` fails with `localStorage.getItem is not a function` (13 tests) — jsdom localStorage setup issue unrelated to this plan's changes
- Logged to deferred-items: Sidebar.test.tsx localStorage mock setup needs fix

## Known Stubs

None — all UI elements are wired to real Wails bindings (`GetWebServerMode`, `GetLocalNetworkPassword`). Data flows from Go backend through Wails to the React components.

## Threat Flags

None — this plan adds UI-only display of an existing password (not a new auth surface). The password is generated and stored by Plan 02 backend work. No new network endpoints introduced.

## Self-Check: PASSED

| Item | Status |
|------|--------|
| LocalNetworkBanner.tsx | FOUND |
| LocalNetworkBanner.test.tsx | FOUND |
| 60-03-SUMMARY.md | FOUND |
| Commit ebcb762 (Task 1) | FOUND |
| Commit 53e5beb (Task 2) | FOUND |
