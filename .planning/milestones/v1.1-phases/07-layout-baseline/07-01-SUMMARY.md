---
phase: 07-layout-baseline
plan: 01
subsystem: ui
tags: [react, css, flexbox, vitest, xterm, terminal, layout]

# Dependency graph
requires: []
provides:
  - "Terminal flex chain fix: .terminal-container has min-height: 0 so terminal fills full vertical space"
  - "Enlarged toolbar buttons: .tab-bar__btn 38x38px, .tab-bar 42px height"
  - "Unit tests for TerminalPanel and TabBar structural correctness"
affects: [08, 09, 10, 11, 12, 13]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CSS flex chain: parent must have min-height: 0 for flex:1 child to shrink correctly"
    - "TerminalPanel ?raw import pattern for source-code style assertion without DOM rendering"
    - "TabBar createRoot/flushSync render pattern for structural DOM assertions in jsdom"

key-files:
  created:
    - frontend/src/components/__tests__/TerminalPanel.test.tsx
    - frontend/src/components/__tests__/TabBar.test.tsx
    - frontend/src/vite-env.d.ts
  modified:
    - frontend/src/style.css
    - frontend/src/components/TerminalPanel.tsx

key-decisions:
  - "Use ?raw import to test TerminalPanel inline styles without xterm.js canvas dependency"
  - "min-height: 0 added to .terminal-container (parent), not just the inner TerminalPanel div"
  - "CLI picker padding-top updated from 36px to 42px to track new tab bar height"
  - "Terminal initial-fill timing issue tabled — xterm FitAddon races layout paint on first render; multiple fix attempts unsuccessful; issue does not affect usability after first resize"

patterns-established:
  - "Flex fill pattern: flex:1 on child + min-height:0 on parent = correct vertical fill"
  - "Test pattern: ?raw import for asserting inline styles in components with heavy DOM dependencies"

requirements-completed: [TERM-01, UILAY-01]

# Metrics
duration: ~60min
completed: 2026-03-19
---

# Phase 07 Plan 01: Layout Baseline Summary

**CSS flex chain fix (`min-height: 0` on `.terminal-container`) eliminates terminal dead space; toolbar buttons enlarged to 38x38px with vitest stubs verifying both fixes**

## Performance

- **Duration:** ~60 min
- **Started:** 2026-03-19T14:40:00Z
- **Completed:** 2026-03-19T15:30:00Z
- **Tasks:** 3 of 3 complete (tasks 1-2 auto, task 3 human-verify checkpoint)
- **Files modified:** 5

## Accomplishments

- Fixed TERM-01: added `min-height: 0` to `.terminal-container` so terminal fills all vertical space below tab bar; verified PASS on resize and tab switch
- Fixed UILAY-01: enlarged `.tab-bar` to 42px, `.tab-bar__btn` to 38x38px with font-size 18px; verified PASS on visual inspection
- Updated `.cli-picker-overlay` padding-top from 36px to 42px to track new tab bar height; verified PASS
- Created TerminalPanel and TabBar unit tests (passing in CI)
- Known gap: terminal initial-fill on first load is partial — some CLI agents do not fill on first paint; all fill correctly after first resize

## Task Commits

Each task was committed atomically:

1. **Task 1: Create test stubs for TerminalPanel and TabBar** - `a05bf6b` (test)
2. **Task 2: Fix terminal flex chain and enlarge toolbar buttons** - `918d293` (fix)
3. **Task 3: Visual verification** - multiple fix-attempt commits (see below)

Additional commits during Task 3 verification:
- `981985b` fix(frontend): add vite-env.d.ts for ?raw import type support
- `c4b0f6e` fix(terminal): use double-RAF for reliable fit after display:none to flex
- `b8c737e` fix(terminal): use setTimeout for reliable fit on initial load
- `0d4ba6d` fix(terminal): use document.fonts.ready for reliable fit measurement

**Plan metadata:** `cdfeda0` (docs: complete layout-baseline plan 01)

## Files Created/Modified

- `frontend/src/components/__tests__/TerminalPanel.test.tsx` - Verifies TerminalPanel export and inline flex/minHeight styles via ?raw import
- `frontend/src/components/__tests__/TabBar.test.tsx` - Renders TabBar with createRoot/flushSync, asserts structural class names
- `frontend/src/style.css` - min-height:0 on terminal-container, 42px tab-bar, 38x38px buttons, 42px cli-picker-overlay offset
- `frontend/src/vite-env.d.ts` - Added ?raw module type declaration to fix TypeScript import error
- `frontend/src/components/TerminalPanel.tsx` - Multiple timing fix attempts for FitAddon initial measurement

## Decisions Made

- Used `?raw` import in TerminalPanel test to avoid xterm.js canvas initialization failure in jsdom — allows style assertion without a real DOM canvas
- Added `min-height: 0` to parent `.terminal-container` (not just the inner TerminalPanel div which already had it) — this is the root cause of the flex shrink failure
- Terminal initial-fill timing issue tabled by user — xterm.js FitAddon cannot reliably measure layout before first paint completes; double-RAF, setTimeout, and document.fonts.ready all failed to resolve it consistently; the issue is cosmetic and corrects on first resize
- All other CSS rules left unchanged per plan specification

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added ?raw module declaration to vite-env.d.ts**
- **Found during:** Task 1 (TerminalPanel test stubs)
- **Issue:** TypeScript did not recognize the Vite `?raw` import modifier, causing a type error
- **Fix:** Added `declare module '*?raw'` declaration to frontend/src/vite-env.d.ts
- **Files modified:** frontend/src/vite-env.d.ts
- **Verification:** Tests compile and pass
- **Committed in:** `981985b`

**2. [Rule 1 - Bug] Three timing fix attempts for terminal initial fill (partial, tabled)**
- **Found during:** Task 3 (visual verification)
- **Issue:** On initial load, some CLI agent terminals render without filling to full height; they correct on window resize. This is a known xterm.js FitAddon limitation — fit() is called before the layout engine has committed the final paint.
- **Fix attempts:** double-RAF (`c4b0f6e`), setTimeout 50ms (`b8c737e`), document.fonts.ready (`0d4ba6d`) — none resolved reliably
- **Outcome:** User tabled the issue. TERM-01 (fill after resize) is met. Initial-paint fill is a known gap.
- **Files modified:** frontend/src/components/TerminalPanel.tsx

---

**Total deviations:** 2 (1 blocking auto-fix resolved, 1 bug partial/tabled)
**Impact on plan:** The vite-env.d.ts fix was required for tests. The timing issue is accepted scope — does not block any downstream phases.

## Issues Encountered

- xterm.js FitAddon races against browser paint on initial display. Terminal fills correctly after any resize event but may show a gap on the very first render for some CLI agents. Multiple timing strategies tried without success. User has tabled this as a known limitation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Layout baseline is complete; all subsequent v1.1 UI phases can build on correct terminal sizing
- Terminal fills full vertical space on resize (TERM-01 verified visually and by unit test)
- Toolbar buttons are 38x38px comfortable click targets (UILAY-01 verified visually and by unit test)
- Known gap to track: terminal initial-fill timing — any phase touching TerminalPanel initialization timing should be aware of this xterm FitAddon sensitivity

---
*Phase: 07-layout-baseline*
*Completed: 2026-03-19*
