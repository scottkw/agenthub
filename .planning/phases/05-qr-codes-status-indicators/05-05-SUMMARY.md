---
phase: 05-qr-codes-status-indicators
plan: "05"
subsystem: ui
tags: [xterm, react, css, terminal, fit-addon, resize-observer]

# Dependency graph
requires:
  - phase: 03-wails-desktop-ui
    provides: TerminalPanel.tsx with xterm.js FitAddon integration
provides:
  - TerminalPanel with deferred fit() via requestAnimationFrame
  - ResizeObserver replacing window resize listener for reliable terminal sizing
  - .terminal-wrapper display:flex CSS fix
  - .web-serving-bar flex-shrink:0 CSS rule
affects: [05-qr-codes-status-indicators]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - requestAnimationFrame to defer xterm FitAddon.fit() until after browser layout pass
    - ResizeObserver on containerRef for reactive terminal sizing on any layout change

key-files:
  created: []
  modified:
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/style.css

key-decisions:
  - "Use requestAnimationFrame (not setTimeout) to defer fit() — fires after browser completes the current paint frame, ensuring containerRef.clientHeight is non-zero"
  - "ResizeObserver on containerRef replaces window resize listener — handles window resize, sidebar open/close, and any layout change without polling"

patterns-established:
  - "Terminal fit pattern: always defer fit() via requestAnimationFrame after term.open() and after isActive transitions"
  - "ResizeObserver cleanup pattern: store observer in local const, call ro.disconnect() in cleanup, cancelAnimationFrame on rafId"

requirements-completed: [TERM-01, TERM-03]

# Metrics
duration: 5min
completed: 2026-03-18
---

# Phase 5 Plan 05: Terminal Height Fix Summary

**xterm.js FitAddon fit() deferred via requestAnimationFrame and ResizeObserver replacing window listener, eliminating blank area below terminal**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-18T21:33:00Z
- **Completed:** 2026-03-18T21:38:17Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments

- Deferred the initial fit() call in the mount effect using requestAnimationFrame — browser layout completes before measuring, so containerRef.clientHeight is non-zero
- Replaced window resize listener in isActive effect with ResizeObserver on containerRef — handles all layout changes (window resize, panel switches, sidebar) reliably
- Added `display: flex` to `.terminal-wrapper` CSS rule — was missing, causing the flex column layout to not propagate height correctly
- Added `.web-serving-bar { flex-shrink: 0 }` rule — prevents the URL bar from stretching and consuming terminal space

## Task Commits

Each task was committed atomically:

1. **Task 1: Defer fit() calls and add ResizeObserver in TerminalPanel.tsx** - `a75d497` (fix)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `frontend/src/components/TerminalPanel.tsx` - Deferred fit() via requestAnimationFrame; ResizeObserver replaces window listener in isActive effect
- `frontend/src/style.css` - Added display:flex to .terminal-wrapper; added .web-serving-bar flex-shrink:0 rule

## Decisions Made

- Used `requestAnimationFrame` (not `setTimeout(0)`) — rAF fires synchronously after the browser completes the current layout/paint frame, which is precisely when clientHeight becomes accurate
- ResizeObserver on `containerRef` rather than `window` — more precise, handles any container size change not just window resize (e.g., the web-serving-bar appearing/disappearing changes available height)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Terminal height fix completes the UAT gap identified in test 3 (large blank area below terminal content)
- All three UAT gaps (status dot, terminal sizing, dashboard auth) now have targeted fixes via plans 05-04, 05-05, and 05-06
- Ready for Phase 6 or final UAT re-run

---
*Phase: 05-qr-codes-status-indicators*
*Completed: 2026-03-18*
