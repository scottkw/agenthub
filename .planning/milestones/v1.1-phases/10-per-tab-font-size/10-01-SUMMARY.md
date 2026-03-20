---
phase: 10-per-tab-font-size
plan: 01
subsystem: ui
tags: [react, xterm, terminal, font-size, keyboard-shortcuts]

# Dependency graph
requires:
  - phase: 07-layout-baseline
    provides: TerminalPanel ?raw test pattern, fitAddon usage, isActive/sessionId props
  - phase: 08-per-tab-status-bar
    provides: per-tab Record<string, T> state pattern in App.tsx
provides:
  - Per-tab font size control via SHIFT+= (increase) and SHIFT+- (decrease)
  - attachCustomKeyEventHandler in TerminalPanel with PTY suppression (return false)
  - fontSizes Record<string, number> state in App.tsx with DEFAULT_FONT_SIZE=14
  - Font size clamped to [6, 32] via Math.max/Math.min
  - Cleanup on tab close via setFontSizes delete
affects: [11-build-script, any future terminal feature plans]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - attachCustomKeyEventHandler for intercepting keyboard shortcuts without PTY injection
    - Separate useEffect([fontSize]) for applying controlled prop to mutable xterm options
    - ?raw source-inspection tests for verifying key handler and effect dependency arrays

key-files:
  created:
    - frontend/src/components/__tests__/App.test.tsx
  modified:
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/App.tsx
    - frontend/src/components/__tests__/TerminalPanel.test.tsx

key-decisions:
  - "onFontSizeChange intentionally omitted from [sessionId] effect deps — stable callback captured once per session avoids re-running full terminal setup"
  - "Key handler uses ev.key === '=' not ev.key === '+' — SHIFT+= produces '=' as the key value, not '+'"
  - "Separate useEffect([fontSize]) applies options.fontSize + fit() independently of setup effect"

patterns-established:
  - "?raw source inspection for xterm effects: import raw from '../Component.tsx?raw' then expect(raw).toContain(...)"
  - "attachCustomKeyEventHandler: register in [sessionId] effect, return false to suppress, return true to pass through"
  - "Per-tab controlled prop pattern: App owns state Record<string,T>, passes prop + callback to TerminalPanel"

requirements-completed: [TERM-02, TERM-03, TERM-04]

# Metrics
duration: 2min
completed: 2026-03-19
---

# Phase 10 Plan 01: Per-Tab Font Size Summary

**SHIFT+=/- font size control via attachCustomKeyEventHandler with PTY suppression, per-tab fontSizes state in App.tsx, and useEffect([fontSize]) applying term.options.fontSize + fitAddon.fit()**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-19T20:41:33Z
- **Completed:** 2026-03-19T20:43:31Z
- **Tasks:** 2 (TDD: RED + GREEN)
- **Files modified:** 4

## Accomplishments

- TerminalPanel intercepts SHIFT+= and SHIFT+- via attachCustomKeyEventHandler, returning false to suppress PTY injection
- App.tsx manages per-tab fontSizes Record<string,number> with DEFAULT_FONT_SIZE=14 and clamp [6,32]
- useEffect([fontSize]) applies term.options.fontSize then fitAddon.fit() for correct reflow after resize
- 17 new source-inspection tests all pass; all 56 tests green

## Task Commits

Each task was committed atomically:

1. **Task 1: Write failing source-inspection tests (RED)** - `c9343ea` (test)
2. **Task 2: Implement per-tab font size in TerminalPanel and App.tsx (GREEN)** - `c6a24ea` (feat)

**Plan metadata:** (docs commit below)

_Note: TDD tasks — test commit followed by feat commit_

## Files Created/Modified

- `frontend/src/components/TerminalPanel.tsx` - Extended props (fontSize, onFontSizeChange), attachCustomKeyEventHandler, useEffect([fontSize])
- `frontend/src/App.tsx` - DEFAULT_FONT_SIZE, fontSizes state, handleFontSizeChange with clamp, prop drilling to TerminalPanel, cleanup on tab close
- `frontend/src/components/__tests__/TerminalPanel.test.tsx` - Added describe('font size control') block with 10 source-inspection tests
- `frontend/src/components/__tests__/App.test.tsx` - New file: 7 source-inspection tests for App fontSizes state

## Decisions Made

- `onFontSizeChange` is intentionally omitted from the `[sessionId]` effect dependency array. The handler is a stable useCallback with no session-specific deps, captured once at terminal setup. Adding it would re-run the entire terminal setup (reconnect relay, recreate xterm instance) on every font size change — incorrect behavior.
- `ev.key === '='` not `ev.key === '+'`: when SHIFT is held and `=` key is pressed, the browser reports `ev.key = '='` (the physical key), not `'+'` (the shifted character). The plan specifies this explicitly.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Font size feature complete and tested
- TerminalPanel now fully controlled for font size — ready for any future visual customization
- Phase 13 (build script) can proceed as planned

## Self-Check: PASSED

- FOUND: frontend/src/components/TerminalPanel.tsx
- FOUND: frontend/src/App.tsx
- FOUND: frontend/src/components/__tests__/TerminalPanel.test.tsx
- FOUND: frontend/src/components/__tests__/App.test.tsx
- FOUND: .planning/phases/10-per-tab-font-size/10-01-SUMMARY.md
- FOUND: commit c9343ea (test RED phase)
- FOUND: commit c6a24ea (feat GREEN phase)
