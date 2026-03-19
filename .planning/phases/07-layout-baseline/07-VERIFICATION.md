---
phase: 07-layout-baseline
verified: 2026-03-19T11:20:00Z
status: gaps_found
score: 4/5 must-haves verified
gaps:
  - truth: "Terminal content fills full available vertical height with no blank dead space below output"
    status: partial
    reason: "xterm.js FitAddon timing race on initial render — some CLI agents show dead space on first paint before any resize occurs. CSS fix (min-height: 0) is in place and correctly enables fill after window resize. Initial-paint fill is unreliable due to FitAddon measuring before layout commits. User has tabled this issue."
    artifacts:
      - path: "frontend/src/components/TerminalPanel.tsx"
        issue: "document.fonts.ready timing strategy does not guarantee FitAddon fires after first paint commits layout; dead space visible on some initial loads"
    missing:
      - "Reliable initial-fill strategy (tabled by user — not blocking downstream phases)"
human_verification:
  - test: "Terminal initial fill on app launch"
    expected: "Every CLI agent terminal fills full vertical height immediately on first render, before any resize event"
    why_human: "Timing race between xterm.js FitAddon.fit() and browser layout commit cannot be verified by grep/static analysis; requires visual inspection at app launch"
---

# Phase 07: Layout Baseline Verification Report

**Phase Goal:** Terminals fill all available space and toolbar buttons are easy to click
**Verified:** 2026-03-19T11:20:00Z
**Status:** gaps_found (1 known partial gap, tabled by user)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Terminal content fills full available vertical height with no blank dead space | PARTIAL | CSS fix verified; initial-paint race reported by user (4/5 visual checks passed) |
| 2 | Toolbar buttons (+ and gear) are 38x38px and comfortable to click | VERIFIED | style.css line 138-139: `width: 38px; height: 38px` confirmed |
| 3 | Tab bar height is 42px, accommodating larger buttons without overflow | VERIFIED | style.css line 38: `height: 42px` confirmed |
| 4 | Adding or switching tabs does not cause layout collapse or incorrect terminal sizing | VERIFIED | User confirmed: tab switching no collapse (visual check passed) |
| 5 | CLI picker overlay positions correctly below the taller tab bar | VERIFIED | style.css line 327: `padding-top: 42px` confirmed; user confirmed (visual check passed) |

**Score:** 4/5 truths verified (1 partial — terminal initial fill)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/style.css` | Fixed flex chain and enlarged button dimensions | VERIFIED | min-height:0 on .terminal-container (line 156); 42px tab-bar (line 38); 38x38px buttons (lines 138-139); 42px cli-picker-overlay (line 327) |
| `frontend/src/components/__tests__/TerminalPanel.test.tsx` | Unit tests verifying inline flex/minHeight styles | VERIFIED | 2 tests pass: export check + ?raw source assertion for `flex: 1`, `minHeight: 0`, `width: '100%'` |
| `frontend/src/components/__tests__/TabBar.test.tsx` | Unit tests verifying TabBar structural class names | VERIFIED | 5 tests pass: .tab-bar, .tab-bar__controls, .tab-bar__btn--add, .tab-bar__btn--settings, .tab-list |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `.terminal-container` in style.css | `TerminalPanel.tsx` inline `flex:1, minHeight:0` | CSS flex chain — parent min-height:0 allows child flex:1 to shrink | WIRED | App.tsx line 228 renders `<div className="terminal-container">` wrapping TerminalPanel; style.css line 154-160 has `.terminal-container { flex: 1; min-height: 0; ... }` |
| `.tab-bar` height: 42px | `.terminal-container` via flex:1 | flex-shrink:0 on tab-bar means terminal-container gets remaining space | WIRED | App.tsx renders `.app` (flex column) containing TabBar then `.terminal-container`; tab-bar has flex-shrink:0 (style.css line 37) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| TERM-01 | 07-01-PLAN.md | Terminal content fills all available space with no dead space | PARTIAL | CSS infrastructure correct; initial-paint fill unreliable (tabled). Fill works after any resize event. |
| UILAY-01 | 07-01-PLAN.md | Toolbar buttons are visually larger and easy to click | SATISFIED | .tab-bar__btn is 38x38px (width/height verified in style.css lines 138-139); user confirmed comfortable to click |

**Orphaned requirements check:** REQUIREMENTS.md traceability table maps only TERM-01 and UILAY-01 to Phase 7. No orphaned requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `frontend/src/components/TerminalPanel.tsx` | 104 | `document.fonts.ready.then(() => { if (!cancelled) fit() })` | Info | This is the final timing attempt for initial fill; works for resize but not guaranteed for initial paint |

No TODO/FIXME/placeholder comments found in phase artifacts. No stub implementations. No empty handlers.

### Test Results

All 18 vitest tests pass (3 test files):
- `src/lib/relayClient.test.ts` — 10 tests (pre-existing, unaffected)
- `src/components/__tests__/TabBar.test.tsx` — 5 tests (new in this phase)
- `src/components/__tests__/TerminalPanel.test.tsx` — 2 tests (new in this phase, using ?raw source assertion)

Note: `HTMLCanvasElement.getContext()` not implemented in jsdom — expected warning, does not cause test failures.

### Human Verification Required

#### 1. Terminal Initial Fill on Launch

**Test:** Launch `wails dev`, wait for app window, create a new terminal tab for each available CLI agent.
**Expected:** Terminal output fills the ENTIRE vertical space below the tab bar immediately, with no blank dead space, before any window resize.
**Why human:** FitAddon.fit() timing relative to browser paint cannot be verified statically. User reported 4/5 visual checks passed — the initial-fill check is the known failure. Multiple timing strategies (double-RAF, setTimeout 50ms, document.fonts.ready) were attempted without reliable success.

### Gaps Summary

One known partial gap exists and has been formally tabled by the user:

**TERM-01 initial-paint fill:** The CSS fix (`min-height: 0` on `.terminal-container`) is correctly in place and the flex chain is properly wired. Terminal fill works correctly after any window resize event. The gap is specifically on the very first render for some CLI agent terminals — xterm.js FitAddon.fit() races against the browser layout commit, causing FitAddon to measure zero or wrong dimensions before the terminal is visible. Three timing strategies were attempted (double requestAnimationFrame, setTimeout 50ms, document.fonts.ready) without consistent resolution across all CLI agents.

The user has accepted this as a known xterm.js limitation and tabled it for now. The REQUIREMENTS.md traceability table marks TERM-01 as "Complete" reflecting the CSS infrastructure being in place and fill working after resize. Downstream phases (08-13) can build on top of this layout foundation — none of them depend on the initial-paint timing behavior.

UILAY-01 is fully satisfied: 38x38px buttons are confirmed in code and by user visual testing.

---

_Verified: 2026-03-19T11:20:00Z_
_Verifier: Claude (gsd-verifier)_
