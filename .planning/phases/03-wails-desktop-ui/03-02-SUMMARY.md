---
phase: 03-wails-desktop-ui
plan: 02
subsystem: ui
tags: [react, xterm, typescript, websocket, binary-framing, tabs, settings, vitest]

# Dependency graph
requires:
  - phase: 03-wails-desktop-ui
    plan: 01
    provides: Wails App struct with 7 bound methods, React scaffold, xterm.js dependencies

provides:
  - RelayClient class with binary framing matching Phase 2 protocol.go constants
  - encodeInputFrame, encodeResizeFrame, parseServerFrame pure functions
  - TabBar component with inline rename on double-click
  - TerminalPanel component with xterm.js scrollback:10000, unicode11, WebGL addons
  - SettingsPanel modal for custom CLI path configuration
  - Full App.tsx with tab state management wired to Wails bindings
  - Wails TypeScript binding stubs (App.d.ts, App.js, runtime.js)

affects:
  - 03-03 (system tray — uses the same App.tsx layout and beforeClose hook)

# Tech tracking
tech-stack:
  added:
    - jsdom 29.0.0 (vitest test environment dependency)
  patterns:
    - display:none for hidden terminals (buffer preserved without DOM unmounting)
    - fitAddon.fit() only on active terminal (avoids sizing hidden panels)
    - RelayClient per TerminalPanel instance (TERM-01 independent sessions)
    - Wails TypeScript stubs in wailsjs/ (replaced by wails dev at runtime)
    - WebGL addon with onContextLoss fallback to canvas renderer

key-files:
  created:
    - frontend/src/lib/relayClient.ts
    - frontend/src/lib/relayClient.test.ts
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/components/SettingsPanel.tsx
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/runtime/runtime.js
  modified:
    - frontend/src/App.tsx
    - frontend/src/style.css
    - frontend/package.json
    - frontend/pnpm-lock.yaml

key-decisions:
  - "Wails TypeScript stubs created manually in wailsjs/ — wails dev regenerates them at runtime; stubs allow tsc + vite build to succeed without running the Go backend"
  - "jsdom added as dev dependency — vitest requires it for the jsdom test environment configured in vite.config.ts"
  - "display:none for inactive TerminalPanel divs — xterm.js terminal buffer is tied to DOM node lifecycle; unmounting destroys scrollback buffer"
  - "fitAddon.fit() only called on active terminal resize — calling fit() on hidden terminals produces 0x0 sizing"

# Metrics
duration: 5min
completed: 2026-03-18
---

# Phase 3 Plan 02: React Frontend — Terminal UI Summary

**Complete React frontend: tabbed xterm.js terminal UI with binary framing WebSocket client, inline tab rename, settings panel for custom CLI paths, and Wails binding stubs enabling TypeScript compilation**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-18T14:49:09Z
- **Completed:** 2026-03-18T14:53:24Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments

- RelayClient with full binary framing matching Phase 2 protocol.go constants — 11 unit tests pass
- TerminalPanel with xterm.js scrollback:10000, unicode11 (emoji/CJK), WebGL addon with graceful fallback
- display:none pattern for inactive tabs preserves terminal buffer state
- TabBar with double-click inline rename, + button, gear settings button
- SettingsPanel modal with CLI path override inputs calling UpdateCLIPath
- App.tsx restores existing sessions on mount, manages tab CRUD via Wails bindings
- Frontend builds to dist/ via `tsc && vite build` with zero errors

## Task Commits

1. **Task 1 (RED)** - `197849b` (test: add failing tests for RelayClient binary framing)
2. **Task 1 (GREEN)** - `47e8e73` (feat: implement RelayClient with binary framing functions)
3. **Task 2** - `a31757f` (feat: TabBar + TerminalPanel + SettingsPanel + App.tsx wiring)

## Files Created/Modified

- `frontend/src/lib/relayClient.ts` — RelayClient class + encodeInputFrame/encodeResizeFrame/parseServerFrame
- `frontend/src/lib/relayClient.test.ts` — 11 vitest unit tests for framing functions
- `frontend/src/components/TabBar.tsx` — Horizontal tab bar with inline rename
- `frontend/src/components/TerminalPanel.tsx` — xterm.js with relay WebSocket connection
- `frontend/src/components/SettingsPanel.tsx` — CLI path settings modal
- `frontend/src/App.tsx` — Root component with full tab state management
- `frontend/src/style.css` — Full dark theme (tokyonight), @import xterm.css
- `frontend/src/wailsjs/go/main/App.d.ts` — TypeScript types for Wails-bound methods
- `frontend/src/wailsjs/go/main/App.js` — Wails bridge stub
- `frontend/src/wailsjs/runtime/runtime.js` — Wails runtime bridge stub

## Decisions Made

- **Wails TypeScript stubs**: The `wails dev` command generates TypeScript bindings automatically during development, but since the Go backend isn't running during the frontend-only build, stubs were created to allow `tsc` to resolve types. These stubs are overwritten by Wails on `wails dev` and are correct no-op shims in the interim.
- **jsdom installation**: vitest's `jsdom` environment is declared in vite.config.ts but the package wasn't in devDependencies. Added `jsdom@29.0.0` to fix the test runner startup failure.
- **display:none over unmounting**: xterm.js `Terminal.open()` attaches the terminal to a specific DOM container. If that container is removed from the DOM (React unmounting), the terminal instance must be destroyed and recreated on re-mount, losing the scrollback buffer. Using `display:none` preserves the DOM node and buffer across tab switches.
- **fitAddon.fit() guard on isActive**: Calling `fitAddon.fit()` on a hidden terminal returns 0 cols × 0 rows, which would send a malformed resize frame. The resize listener is only registered while `isActive === true`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] jsdom missing from devDependencies**
- **Found during:** Task 1 (RED phase — test runner startup failure)
- **Issue:** vite.config.ts declares `environment: 'jsdom'` but `jsdom` package was not installed. vitest threw `Cannot find package 'jsdom'`.
- **Fix:** Added `jsdom@29.0.0` to devDependencies via `pnpm add -D jsdom`
- **Files modified:** `frontend/package.json`, `frontend/pnpm-lock.yaml`
- **Committed in:** `197849b` (RED commit)

---

**Total deviations:** 1 auto-fixed (blocking)
**Impact on plan:** Required to run tests. No scope creep.

## Verification Results

- `pnpm run test` — 11/11 tests pass
- `pnpm run build` — TypeScript compilation + Vite build succeeds, dist/ created
- `display:none` confirmed in TerminalPanel.tsx line 111
- `scrollback: 10000` confirmed in TerminalPanel.tsx line 30
- MSG_OUTPUT (0x01) matches protocol.go constants exactly

## Next Phase Readiness

- Plan 03-03 (system tray) can use the App.tsx structure as-is — it adds a tray icon that calls the existing `startup`/`shutdown`/`beforeClose` hooks
- `wails dev` will regenerate wailsjs/ bindings from the live Go backend, replacing the stubs
- Terminal UI is fully functional once started via `wails dev` and a relay session is created

---
*Phase: 03-wails-desktop-ui*
*Completed: 2026-03-18*
