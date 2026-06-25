---
phase: 142-hub-settings-redesign-polish
plan: "02"
subsystem: ui
tags: [terminal, xterm, repaint, theme, tab-switch, webgl, atlas, react]

# Dependency graph
requires:
  - phase: 142-01
    provides: Wave 0 test scaffolding including POL-04 source gate (pendingThemeRef assertion in Sidebar.test.tsx)
  - phase: 141-09
    provides: Rendered app-vs-comp comparison baseline and hardened repaint-path research (142-RESEARCH.md)
provides:
  - "Hardened TerminalPanel repaint path: isActive-guarded theme effect, pendingThemeRef deferral, fitTerminal-after-atlas-clear"
  - "Human-verified: active theme switch, tab switch away/back, cross-tab theme switch all repaint cleanly in native wails dev window"
affects:
  - "142-04 (POL-04 is now complete; terminal repaint correctness is settled for Phase 142)"
  - "143 regression test program (pending-theme deferral pattern should be included in the manual regression checklist)"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "pendingThemeRef deferral: stash theme in a ref when isActive=false; drain (apply options.theme + clearTextureAtlas + null reset) synchronously at the top of the isActive effect before the rAF fit loop"
    - "isActive guard on theme effect: never call clearTextureAtlas() or refresh() on a display:none panel (rows=0 corrupts WebGL atlas)"
    - "fitTerminal-after-atlas-clear: always fitTerminal() (not refresh(0,rows-1)) after clearTextureAtlas() to recalculate cell dims"

key-files:
  created: []
  modified:
    - frontend/src/components/TerminalPanel.tsx

key-decisions:
  - "D-05 confirmed: full hardened repaint path (not a minimal regression patch) — isActive guard + pendingThemeRef stash+drain + fitTerminal after clearTextureAtlas"
  - "CMD +/- font-resize case is untestable in wails dev (font-size effect was intentionally left unchanged); explicitly noted as out-of-scope for POL-04"

patterns-established:
  - "pendingThemeRef pattern: any xterm.js effect that calls clearTextureAtlas must guard on isActive; stash in ref when hidden, drain on activation"

requirements-completed: [POL-04]

# Metrics
duration: ~25min
completed: 2026-06-21
---

# Phase 142 Plan 02: Terminal Repaint Hardening (POL-04) Summary

**isActive-guarded theme effect with pendingThemeRef deferral eliminates terminal WebGL atlas garble on theme switch, tab switch, and cross-tab theme switch — human-verified clean across all five repaint cases in native wails dev**

## Performance

- **Duration:** ~25 min (includes wails dev launch and manual UAT)
- **Started:** 2026-06-21
- **Completed:** 2026-06-21
- **Tasks:** 2 (1 auto + 1 checkpoint:human-verify)
- **Files modified:** 1

## Accomplishments

- Added `pendingThemeRef = useRef<ITheme | null>(null)` to TerminalPanel.tsx — the deferral stash for hidden-panel theme changes
- Replaced the unguarded theme effect with an isActive-gated version: stashes theme when hidden, sets `options.theme + clearTextureAtlas() + fitTerminal()` when active — removes the broken `refresh(0, rows-1)` call
- Drained the pending theme at the top of the isActive fit effect (before rAF loop), so cross-tab theme switches apply cleanly on panel activation
- Human-verified all five repaint cases in native wails dev: active theme switch, tab switch away/back, cross-tab theme switch — all clean; CMD +/- font-resize case confirmed untestable/out-of-scope

## Task Commits

Each task was committed atomically:

1. **Task 1: Harden the TerminalPanel repaint path (isActive guard + pendingThemeRef + fitTerminal)** - `e45de2f6` (fix)
2. **Task 2: Human verify — terminal repaints cleanly across theme/tab switches** - CHECKPOINT PASSED (no code commit; human approval recorded)

**Plan metadata:** *(docs commit — see below)*

## Files Created/Modified

- `frontend/src/components/TerminalPanel.tsx` — pendingThemeRef declared; theme effect rewritten with isActive guard + pendingThemeRef stash/drain; fitTerminal replaces refresh(0,rows-1); isActive fit effect drains pending theme before rAF loop

## Decisions Made

- D-05 confirmed: the full hardened repaint path was implemented (not a minimal regression patch), as specified by the plan's must_haves and RESEARCH.md. The theme effect dependency array is `[theme, isActive]` per the design.
- CMD +/- font-resize: user noted in UAT that CMD +/- does not resize the terminal font in wails dev; the font-size effect was intentionally left unchanged in this plan (per D-05 scope). Noted as out-of-scope for POL-04 — not a regression or blocker.

## Deviations from Plan

None — plan executed exactly as written.

## Human Verification Result

**Task 2 checkpoint: PASSED** (approved by human 2026-06-21)

Repaint cases verified in native `wails dev` window:

| Case | Description | Result |
|------|-------------|--------|
| 1 | Active theme switch (Light ↔ Dark while terminal active) | Clean |
| 2 | Tab switch away from terminal, then back | Clean |
| 3 | Cross-tab theme switch (switch theme while on different tab, return to terminal) | Clean |
| 4 | Active theme switch (additional pass) | Clean |
| 5 (optional) | CMD +/- font-resize | Untestable — CMD +/- does not resize terminal font; out-of-scope for POL-04 |

Human quote: "approved — all repaint cases clean (active theme switch, tab switch away/back, cross-tab theme switch all show no garble). The optional CMD +/- font-resize case could not be tested because CMD +/- does not resize terminal font; this is out of scope for POL-04 (font-size effect was intentionally left unchanged) and is noted separately."

## Issues Encountered

None — tsc exited 0 on the first attempt; the POL-04 source-gate test (asserting `pendingThemeRef` in TerminalPanel.tsx) passed immediately after the implementation.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- POL-04 is fully complete: source gate green, human-verified clean in native wails dev
- TerminalPanel's pendingThemeRef pattern is the canonical approach for future xterm.js effects that touch the WebGL atlas — document in 143 regression checklist
- Phase 142 Plan 03 (POL-05 group nav restructure) is the next plan to execute; it is independent of POL-04

---
*Phase: 142-hub-settings-redesign-polish*
*Completed: 2026-06-21*
