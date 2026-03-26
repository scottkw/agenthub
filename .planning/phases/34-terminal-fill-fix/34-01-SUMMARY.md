---
phase: 34-terminal-fill-fix
plan: 01
subsystem: ui
tags: [xterm, pty, wails, terminal, fit-addon]

requires:
  - phase: 33-gui-args-field
    provides: CreateSession with args parameter wiring
provides:
  - cols/rows threading from frontend to PTY spawn
  - double-rAF initial fit pattern for Wails WebView
  - container dimension estimation in createTab
affects: []

tech-stack:
  added: []
  patterns:
    - "double-rAF deferral for CSS layout commit timing in Wails WebView"
    - "container dimension estimation with Math.floor(clientWidth/charWidth)"

key-files:
  created: []
  modified:
    - internal/daemon/types.go
    - internal/daemon/engine.go
    - internal/daemon/client.go
    - internal/daemon/api.go
    - app.go
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/App.tsx
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts

key-decisions:
  - "Keep document.fonts.ready inside double-rAF rather than removing it entirely — font safety with layout timing"
  - "Use 8px/17px char estimates (14px monospace font) — exact values come from FitAddon post-mount"
  - "Default fallback 220x50 for unmeasurable containers — avoids 80x24 on large screens"

patterns-established:
  - "double-rAF pattern: two nested requestAnimationFrame before measuring layout in Wails WebView"

requirements-completed: [TERM-01, TERM-02, TERM-03, TERM-04]

duration: 15min
completed: 2026-03-26
---

# Phase 34: Terminal Fill Fix Summary

**Thread cols/rows from React frontend through Wails/Go stack to PTY spawn with double-rAF initial fit timing**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-03-26
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Added Cols/Rows to CreateRequest and threaded through entire Go backend stack with 80x24 defaults
- Replaced document.fonts.ready primary trigger with double-rAF pattern in TerminalPanel isActive effect
- App.tsx createTab estimates container dimensions and passes cols/rows to CreateSession
- Updated Wails JS/TS bindings with cols/rows parameters
- Added 2 new Go dimension tests and 5 new frontend source-inspection tests

## Task Commits

1. **Task 1: Thread cols/rows through Go backend** - `b38177f` (feat)
2. **Task 2: Double-rAF fit + dimension estimation** - `02ac223` (feat)

## Files Created/Modified
- `internal/daemon/types.go` - Added Cols, Rows fields to CreateRequest
- `internal/daemon/engine.go` - Added cols, rows params with defaults
- `internal/daemon/client.go` - Threading cols, rows to request body
- `internal/daemon/api.go` - Pass req.Cols, req.Rows to engine
- `app.go` - Wails binding with cols, rows params
- `frontend/src/components/TerminalPanel.tsx` - Double-rAF initial fit
- `frontend/src/App.tsx` - Container dimension estimation in createTab
- `frontend/src/wailsjs/go/main/App.js` - Updated binding call
- `frontend/src/wailsjs/go/main/App.d.ts` - Updated type signature
- `frontend/src/components/__tests__/TerminalPanel.test.tsx` - TERM-04 tests
- `frontend/src/components/__tests__/App.test.tsx` - Updated args threading test

## Decisions Made
- Kept document.fonts.ready inside double-rAF for font safety
- Used 8px/17px character cell estimates based on 14px monospace font

## Deviations from Plan
None - plan executed as written.

## Issues Encountered
- Worktree agent based on older commit caused merge conflicts; changes applied directly instead

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Terminal viewport sizing complete for all CLIs
- Manual verification recommended: `wails build -tags wailsassets` then launch and check terminal fills viewport

---
*Phase: 34-terminal-fill-fix*
*Completed: 2026-03-26*
