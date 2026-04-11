---
phase: 66-web-server-link-ux
plan: 01
subsystem: frontend/settings-tab, app.go
tags: [ux, web-server, qr-code, clipboard, settings]
dependency_graph:
  requires: []
  provides: [GetWebServerQRCode Go method, URL action row in Settings, WEB-01, WEB-02, WEB-03]
  affects: [app.go, SettingsTab.tsx, style.css, App.d.ts, App.js]
tech_stack:
  added: []
  patterns: [source-inspection tests, Wails runtime clipboard/browser APIs, qrcode.Encode]
key_files:
  created:
    - frontend/src/components/__tests__/SettingsTab.web-link-ux.test.tsx
  modified:
    - app.go
    - app_test.go
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/style.css
decisions:
  - "Used fs.readFileSync for CSS inspection (not ?raw) following established project pattern from style.settings.test.ts — vitest/jsdom does not support ?raw imports for CSS"
  - "QR cache survives hide/show toggle — cleared only when server stops (avoids re-fetch on repeated toggles)"
metrics:
  duration: "~15 minutes"
  completed: "2026-04-11"
  tasks_completed: 2
  files_modified: 6
  files_created: 1
---

# Phase 66 Plan 01: Web Server Link UX Summary

## One-liner

URL action row (Open/Copy/QR) added to Settings web server section using Wails BrowserOpenURL, ClipboardSetText, and a new GetWebServerQRCode Go method backed by skip2/go-qrcode.

## What Was Built

### Task 1: Go backend + Wails stubs

Added `GetWebServerQRCode()` to `app.go` — mirrors `GetSessionQRCode` but encodes `resp.URL` directly (no session sub-path). Backed by existing `qrcode.Encode` + `base64` imports; no new dependencies.

Added two Go tests in `app_test.go`:
- `TestGetWebServerQRCode` — starts a direct TLS web server, calls the method, verifies PNG magic bytes
- `TestGetWebServerQRCode_NoServer` — verifies error returned when server not running

Added Wails binding stubs:
- `App.d.ts`: `export function GetWebServerQRCode(): Promise<string>`
- `App.js`: `export const GetWebServerQRCode = () => Call('main.App.GetWebServerQRCode', [])`

### Task 2: Frontend URL action row + CSS + tests

Replaced the old `<p className="settings-panel__url">` URL display in `SettingsTab.tsx` with a flex row containing three action buttons:
- **Open** — calls `BrowserOpenURL(serverURL)` via Wails runtime
- **Copy** — calls `ClipboardSetText(serverURL)` with 1500ms "Copied!" feedback via `urlCopied` state
- **QR** — toggles a 200x200 inline `<img>` from base64 PNG; cache survives hide/show, resets on server stop

Added state: `urlCopied`, `showDashQR`, `dashQRb64`, `qrError`
Added `useEffect` to reset QR state when `isServerRunning` goes false.
Added handlers: `handleCopyURL`, `handleToggleDashQR` (with error state on fetch failure).

Added 6 CSS classes to `style.css`:
- `.settings-web-server__url-row` — flex container with gap
- `.settings-web-server__url-text` — truncates with ellipsis
- `.settings-web-server__action-btn` — base button style
- `.settings-web-server__action-btn:hover` — hover state
- `.settings-web-server__action-btn--active` — QR-on state (blue tint)
- `.settings-web-server__action-btn--copy-done` — Copied! state (green)
- `.settings-web-server__qr` — 200x200 block image with rounded corners

Created `SettingsTab.web-link-ux.test.tsx` with 26 source-inspection tests covering WEB-01, WEB-02, WEB-03 requirements plus CSS class presence.

## Verification Results

- `go test ./... -run TestGetWebServerQRCode -count=1` — PASS (both tests)
- `cd frontend && pnpm test` — all tests pass
- All 7 grep checks from plan verification section: PASS

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] CSS ?raw import does not work in vitest/jsdom**
- **Found during:** Task 2 test run
- **Issue:** `import cssRaw from '../../style.css?raw'` returned empty string in vitest — jsdom does not support `?raw` imports for CSS files
- **Fix:** Changed to `readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')` following the existing `style.settings.test.ts` pattern already established in the project
- **Files modified:** `frontend/src/components/__tests__/SettingsTab.web-link-ux.test.tsx`

## Known Stubs

None — all three actions (Open, Copy, QR) are fully wired to their backends.

## Threat Flags

None — no new network endpoints or auth paths introduced. GetWebServerQRCode uses the same daemon IPC boundary as all existing methods.

## Self-Check: PASSED

- app.go contains `func (a *App) GetWebServerQRCode()`: FOUND
- app_test.go contains `TestGetWebServerQRCode`: FOUND
- App.d.ts contains `GetWebServerQRCode(): Promise<string>`: FOUND
- App.js contains `GetWebServerQRCode`: FOUND
- SettingsTab.tsx contains `handleCopyURL`: FOUND
- SettingsTab.tsx contains `BrowserOpenURL(serverURL)`: FOUND
- style.css contains `.settings-web-server__url-row`: FOUND
- Test file exists: FOUND
