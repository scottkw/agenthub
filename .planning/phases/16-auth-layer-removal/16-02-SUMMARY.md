---
phase: 16-auth-layer-removal
plan: "02"
subsystem: frontend
tags: [auth-removal, frontend, react, typescript, dashboard]
dependency_graph:
  requires: [Phase 16 Plan 01 - Go backend auth removal]
  provides: [Clean frontend with no auth UI]
  affects:
    - frontend/src/App.tsx
    - frontend/src/components/SettingsPanel.tsx
    - frontend/src/components/StatusBar.tsx
    - frontend/src/components/__tests__/SettingsPanel.test.tsx
    - frontend/src/components/__tests__/StatusBar.test.tsx
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - web/dashboard.html
tech_stack:
  added: []
  patterns:
    - React functional components with auth state variables removed
    - Wails binding stubs pruned to match live Go exports
    - Static HTML dashboard with direct session load (no login gate)
key_files:
  created: []
  modified:
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/App.tsx
    - frontend/src/components/SettingsPanel.tsx
    - frontend/src/components/StatusBar.tsx
    - frontend/src/components/__tests__/SettingsPanel.test.tsx
    - frontend/src/components/__tests__/StatusBar.test.tsx
    - web/dashboard.html
  deleted: []
decisions:
  - "SettingsPanel reduced to 2 tabs (CLI Paths, Web Server); Security tab and all password state removed"
  - "StatusBar Copy Link button removed; QR and Disable Web remain as the only per-session actions"
  - "Dashboard HTML loads session list directly on DOMContentLoaded with no login gate"
  - "Double-nested wailsjs/wailsjs stubs cleaned locally (gitignored, not committed)"
metrics:
  duration: "8m"
  completed_date: "2026-03-20"
  tasks_completed: 2
  files_changed: 8
---

# Phase 16 Plan 02: Frontend Auth UI Removal Summary

**One-liner:** Deleted Security tab, password state, Copy Link button, and login/CA/token sections from frontend React components and dashboard HTML; Wails bindings pruned to match the auth-free Go backend.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Update Wails bindings, App.tsx, SettingsPanel.tsx, StatusBar.tsx | a906e5f | App.d.ts, App.js, App.tsx, SettingsPanel.tsx, StatusBar.tsx |
| 2 | Strip dashboard.html and update frontend tests | ec2036b | dashboard.html, SettingsPanel.test.tsx, StatusBar.test.tsx |

## What Was Built

Removed all frontend authentication UI to match the backend auth removal from Plan 01:

**Wails bindings (`App.d.ts`, `App.js`):**
- Deleted `SetWebPassword`, `IsWebPasswordSet`, `GenerateSessionToken` exports

**SettingsPanel.tsx:**
- Removed `SetWebPassword` and `IsWebPasswordSet` from imports
- Removed state variables: `webPassword`, `isPasswordSet`, `passwordSaving`, `passwordError`
- Removed `handleSetPassword()` async function
- Removed `IsWebPasswordSet()` call from `loadWebState` effect
- Narrowed `activeTab` type from `'cli-paths' | 'web-server' | 'security'` to `'cli-paths' | 'web-server'`
- Deleted Security tab button from tab list
- Deleted entire Security tab content panel (password input, set password button)
- Fixed Start Web Server disabled condition: `(!isServerRunning && !ctDisclosed)` only (removed `!isPasswordSet` gate)
- Fixed tooltip: now only shows CT disclosure message

**StatusBar.tsx:**
- Removed `onCopyTokenLink` from `StatusBarProps` interface
- Removed `onCopyTokenLink` from destructured props
- Deleted "Copy Link" button from WEB ON branch

**App.tsx:**
- Removed `GenerateSessionToken` from import
- Deleted `handleCopyTokenLink` function
- Removed `onCopyTokenLink={...}` prop from `<StatusBar>` JSX

**dashboard.html:**
- Deleted `<section id="login-section">` and all child elements
- Deleted `doLogin()` function
- Deleted `#login-error` element
- Made `#dashboard-section` visible by default (removed `display:none`)
- Removed 401-redirect branch from `refreshSessions()`
- Removed auto-login check on page load; replaced with direct `refreshSessions()` in `DOMContentLoaded`
- Deleted `copyTokenLink()` function
- Deleted "Copy Token Link" button from session cards in `renderSessions()`
- Deleted CA certificate `<section id="ca-section">` and all child content
- Deleted dead CSS: `.ca-section`, `.platform-tabs`, `.platform-tab`, `.platform-content`, `input[type="password"]`, `#login-error`, `#login-section`, `#copy-token-result`

**Test files:**
- `SettingsPanel.test.tsx`: updated mock (removed `SetWebPassword`, `IsWebPasswordSet`); added `HasCTDisclosure`, `AcknowledgeCTDisclosure`; updated tab count assertion from 3 to 2; replaced Security tab tests with "no Security tab" and "CT-disabled Start button" tests
- `StatusBar.test.tsx`: removed `onCopyTokenLink` from default props; updated button assertion to expect no "Copy Link" button

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Cleaned auth methods from double-nested legacy wailsjs stubs**
- **Found during:** Overall verification grep (`! grep -rq ... frontend/src/`)
- **Issue:** The `frontend/src/wailsjs/wailsjs/go/main/` directory (gitignored, Wails-generated legacy copy using `window['go']` style) still had `SetWebPassword`, `IsWebPasswordSet`, `GenerateSessionToken`, and `GetCACertPath`. The full-tree grep verification found these.
- **Fix:** Removed the three auth methods from the gitignored stubs locally. No commit made (files are gitignored).
- **Files modified:** `frontend/src/wailsjs/wailsjs/go/main/App.d.ts`, `App.js` (local only, gitignored)

## Test Results

```
# Go tests (all packages, -race):
ok  github.com/agenthub/agenthub                    2.367s
ok  github.com/agenthub/agenthub/internal/pty       3.306s
ok  github.com/agenthub/agenthub/internal/relay     3.217s
ok  github.com/agenthub/agenthub/internal/status    1.619s
ok  github.com/agenthub/agenthub/internal/webserver 3.199s

# Frontend tests (vitest):
Test Files  7 passed (7)
      Tests  84 passed (84)
```

All tests pass.

## Self-Check: PASSED

- `App.d.ts` has no SetWebPassword, IsWebPasswordSet, GenerateSessionToken: confirmed
- `App.js` has no SetWebPassword, IsWebPasswordSet, GenerateSessionToken: confirmed
- `SettingsPanel.tsx` has no Security tab, no isPasswordSet, no handleSetPassword: confirmed
- `SettingsPanel.tsx` disabled condition only checks ctDisclosed: confirmed
- `StatusBar.tsx` has no onCopyTokenLink, no Copy Link button: confirmed
- `App.tsx` has no GenerateSessionToken, no handleCopyTokenLink, no onCopyTokenLink: confirmed
- `dashboard.html` has no login-section, doLogin, copyTokenLink, ca-section: confirmed
- `SettingsPanel.test.tsx` has no SetWebPassword, IsWebPasswordSet, Security tab tests: confirmed
- `StatusBar.test.tsx` has no onCopyTokenLink: confirmed
- Commits a906e5f and ec2036b exist: confirmed
- Full test suite (Go -race + frontend vitest): green
