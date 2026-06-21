# Phase 142: Hub & Settings Redesign Polish - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-21
**Phase:** 142-hub-settings-redesign-polish
**Areas discussed:** Group navigation (POL-05), Card preview + icons (POL-01), Terminal garble fix (POL-04), Controls (POL-02/03)

---

## Group navigation (POL-05)

| Option | Description | Selected |
|--------|-------------|----------|
| Nest under Hub in sidebar | Groups as expandable sub-list under the "Hub" item in the main left sidebar; grid full-width | ✓ |
| Group row in FilterBar | Groups as a chip/dropdown row inside HubFilterBar next to status chips | |
| Discuss / undecided | Talk through trade-offs first | |

**User's choice:** Nest under Hub in sidebar.
**Notes:** The standalone comp never depicted a Groups concept (it predates the Hub-first restructure and still shows the dropped Sessions/Remote pages), so "per the comp" is aspirational. Nested-sidebar pattern is the agreed structure. Drag-to-assign preservation flagged for researcher (drop onto sidebar group item, else per-card menu fallback).

---

## Card preview + icons (POL-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Taller fixed preview + reserved gutter | Larger fixed preview height (~6 lines) + dedicated header gutter so ⋮/☰ never overlap | ✓ |
| Aspect-ratio scaled preview | Preview scales with card width; icons pinned to corners | |
| Discuss / undecided | Decide sizing + placement together | |

**User's choice:** Taller fixed preview + reserved gutter.
**Notes:** Exact pixel height left to planner, bounded by "legible/~6 lines" intent.

---

## Terminal garble fix (POL-04)

| Option | Description | Selected |
|--------|-------------|----------|
| Root-cause, fix minimally | Confirm 141-08 regression source, patch just that | |
| Harden the whole repaint path | Fix regression + systematically address theme/tab/font repaint coordination | ✓ |
| Discuss / undecided | Scope after research | |

**User's choice:** Harden the whole repaint path.
**Notes:** Scout flagged TerminalPanel.tsx theme effect lacking isActive guard, no re-fit after repaint, tab-switch/theme race, WebGL atlas timing. Verify natively (`:34115` bridge has no PTY).

---

## Controls (POL-02/03)

| Option | Description | Selected |
|--------|-------------|----------|
| Locked — sensible defaults | Sun/moon icon + text label knob (colorblind-safe); both New session buttons styled like comp's "+ New Session" | ✓ |
| Discuss toggle/button details | Specify state indication / button styling precisely | |

**User's choice:** Locked — sensible defaults.
**Notes:** Owner is colorblind — toggle state must read via icon + text, not color/position alone. Existing uiTheme persistence + onClick wiring unchanged.

---

## Claude's Discretion

- Exact preview pixel height and grid reflow breakpoints (bounded by D-04).
- Plan-splitting across the five POL items and migration ordering.
- Exact CSS token/class names, following `--hub-*` convention + light-theme override discipline.

## Deferred Ideas

None — discussion stayed within phase scope. The formal regression-test program (TEST-01..05) is Phase 143, explicitly out of scope.
