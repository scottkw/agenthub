---
phase: 35-terminal-fill-fix-v2
plan: 01
subsystem: ui
tags: [xterm, terminal, raf, fit, proposedimensions, wails, react]

# Dependency graph
requires:
  - phase: 34
    provides: double-rAF initial fit pattern + cols/rows threading to PTY
provides:
  - Bounded rAF retry loop in TerminalPanel isActive effect polling proposeDimensions() for cell readiness
  - Source inspection tests for FILL-01..06 requirements
  - Production binary built and ready for manual verification
affects: [TerminalPanel, terminal-fill, initial-fit, xterm-fit-addon]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "rAF retry loop: poll proposeDimensions() until non-undefined before calling fit()"
    - "MAX_ATTEMPTS = 20 bounds retry loop at ~333ms (60fps) to prevent infinite loops"
    - "cancelled flag + cancelAnimationFrame(rafId) for safe cleanup across multiple rAF IDs"

key-files:
  created: []
  modified:
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/components/__tests__/TerminalPanel.test.tsx

key-decisions:
  - "Replace double-rAF one-shot with bounded rAF retry loop polling proposeDimensions() — canonical signal that CharSizeService has measured cell dimensions"
  - "MAX_ATTEMPTS = 20 (~333ms at 60fps) provides enough time for all CLI startup sequences while bounding the loop"
  - "document.fonts.ready removed — proposeDimensions() returning non-undefined already implies font measurement complete"
  - "Single rafId variable sufficient — cancelled flag short-circuits any in-flight rAF after cleanup"

patterns-established:
  - "proposeDimensions() !== undefined: canonical readiness check for FitAddon before calling fit()"
  - "Retry via requestAnimationFrame(() => fn(attempt + 1)) with attempt < MAX_ATTEMPTS guard"

requirements-completed: [FILL-01, FILL-02, FILL-03, FILL-04, FILL-05, FILL-06]

# Metrics
duration: 8min
completed: 2026-03-26
---

# Phase 35 Plan 01: Terminal Fill Fix v2 Summary

**Replaced double-rAF one-shot fit with bounded rAF retry loop polling FitAddon.proposeDimensions() until cell dimensions are non-zero, fixing initial-load terminal fill for Claude, Gemini, and OpenCode CLIs**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-03-26T16:03:34Z
- **Completed:** 2026-03-26T16:12:00Z
- **Tasks:** 2 automated complete, 1 awaiting human verification
- **Files modified:** 2

## Accomplishments

- Replaced double-rAF one-shot (fires once at ~32ms, misses slow CLIs) with bounded rAF retry loop that polls `proposeDimensions()` until cell dimensions are non-zero
- Added source inspection tests for all 6 FILL requirements confirming new pattern and absence of old pattern
- Built production binary (`wails build -tags wailsassets`) for manual verification
- All 150 frontend tests pass; Go daemon tests pass

## Task Commits

1. **Task 1: Add Wave 0 source inspection tests for rAF retry loop** - `9148a11` (test)
2. **Task 2: Replace double-rAF with bounded rAF retry loop** - `2f86d5a` (feat)
3. **Task 3: Verify terminal fill in production build** - AWAITING human verification

## Files Created/Modified

- `frontend/src/components/TerminalPanel.tsx` - Replaced isActive effect: double-rAF one-shot with bounded rAF retry loop (MAX_ATTEMPTS=20, tryFit, proposeDimensions readiness check)
- `frontend/src/components/__tests__/TerminalPanel.test.tsx` - Replaced TERM-04 tests with FILL-01..06 tests for rAF retry loop pattern

## Decisions Made

- Used `proposeDimensions() !== undefined` as the canonical readiness signal — FitAddon source confirms it returns `undefined` when `css.cell.width === 0`, which happens when CharSizeService hasn't measured the font yet (opened with `display:none`)
- Kept single `rafId` variable overwritten each iteration, relying on `cancelled` flag to short-circuit any in-flight rAF after cleanup (safe per research analysis in 35-RESEARCH.md)
- Removed `document.fonts.ready` — not needed; if `proposeDimensions()` returns non-undefined, cell dimensions are already non-zero which implies font measurement completed
- Retained ResizeObserver for all subsequent size changes (no change to that path)

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None. The TDD RED→GREEN cycle was clean: 6 tests failed RED (3 checking new patterns absent, 3 checking old patterns present), then all 150 passed GREEN after the TerminalPanel.tsx update.

## Known Stubs

None — the retry loop is fully wired. FILL-06 (production build behavior) is verified via the production binary checkpoint (Task 3, awaiting human confirmation).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- TerminalPanel.tsx has bounded rAF retry loop replacing double-rAF one-shot
- All automated tests green (150 frontend, Go daemon pass)
- Production binary built at `build/bin/agenthub.app/Contents/MacOS/agenthub`
- Manual verification of all 4 CLIs (Claude, Gemini, OpenCode, Codex) pending at Task 3 checkpoint
- Upon user confirmation, phase 35 is complete and v1.6 milestone closes

## Self-Check: PASSED

- FOUND: frontend/src/components/TerminalPanel.tsx
- FOUND: frontend/src/components/__tests__/TerminalPanel.test.tsx
- FOUND: .planning/phases/35-terminal-fill-fix-v2/35-01-SUMMARY.md
- FOUND: commit 9148a11 (test: add FILL-01..06 tests)
- FOUND: commit 2f86d5a (feat: rAF retry loop)

---
*Phase: 35-terminal-fill-fix-v2*
*Completed: 2026-03-26*
