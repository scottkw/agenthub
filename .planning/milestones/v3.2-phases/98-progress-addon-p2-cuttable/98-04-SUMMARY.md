---
phase: 98-progress-addon-p2-cuttable
plan: "04"
subsystem: progress-addon
tags: [progress, tabbar, css, transform, wave-3]
dependency_graph:
  requires:
    - "98-01 (wave-0: vendored addon + RED scaffolds)"
    - "98-02 (wave-1: aggregateProgress + SetTrayProgress RPC)"
    - "98-03 (wave-2: TerminalPanel hot-swap + App.tsx registry + tabProgress wiring)"
  provides:
    - "TabBar.tsx tabProgress prop fully destructured and consumed in render"
    - "TabBar.tsx .tab__progress <div> per tab with data-testid and scaleX transform"
    - "style.css .tab__progress rule using transform:scaleX(0) initial state"
    - "style.css .tab position:relative anchor for .tab__progress absolute positioning"
    - "Wave 0 RED tests progress-underline + progress-transform flipped GREEN"
    - "PRG-02 per-tab visual affordance fully complete on desktop"
  affects:
    - "98-05 (Wave 4 — web parity + e2e + UAT, the only remaining cuttable wave)"
tech_stack:
  added: []
  patterns:
    - "transform:scaleX(0..1) for GPU compositor underline animation (Pitfall #4 invariant)"
    - "transform-origin:left for left-to-right fill growth"
    - "200ms ease-out CSS transition for smooth value changes"
    - "position:relative on .tab anchors absolute child .tab__progress"
    - "scaleX(value/100) maps 0-100 percent to 0-1 CSS transform scale"
key_files:
  created: []
  modified:
    - "frontend/src/components/TabBar.tsx (tabProgress destructured + .tab__progress element rendered)"
    - "frontend/src/style.css (.tab position:relative + .tab__progress rule)"
key-decisions:
  - ".tab__progress element placed as the LAST child inside each tab <div>, after .tab__close button — this ensures the absolute-positioned underline stacks correctly without interfering with the flex layout of the tab content"
  - ".tab already lacked position:relative — added in this wave as the anchor for .tab__progress absolute positioning; no existing property was displaced"
  - "transform:scaleX(value/100) used instead of width animation — GPU compositor handles transform without layout reflow (Pitfall #4 invariant); the CSS width:100% is a static declaration that never changes"
  - "scaleX(0) when tabProgress[sessionId] is undefined or 0 — underline is invisible at zero scale, no conditional rendering needed (single CSS transition handles enter/exit)"
patterns-established:
  - "Pattern: Unconditional underline element with scaleX(0) zero state — renders always, visible only when progress > 0; avoids conditional mount/unmount which would reset the CSS transition"
requirements-completed: [PRG-02]
duration: ~15min
completed: "2026-05-08"
---

# Phase 98 Plan 04: Wave 3 Per-Tab Progress Underline Summary

**TabBar.tsx renders a .tab__progress underline per tab using transform:scaleX(value/100) driven by the tabProgress prop wired at Wave 2 — PRG-02's per-tab visual affordance is fully complete, Wave 0 RED tests progress-underline and progress-transform flip GREEN.**

## Performance

- **Duration:** ~15 minutes
- **Started:** 2026-05-08T15:00:00Z
- **Completed:** 2026-05-08T15:14:37Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments

- Extended TabBar.tsx function destructuring to include `tabProgress` (the prop was declared in Wave 2 but not yet consumed)
- Injected `<div className="tab__progress" style={{ transform: `scaleX(...)` }} data-testid={`tab-progress-${tab.id}`} />` as the last child inside each tab's render, after `.tab__close`
- Added `position: relative` to the `.tab` rule in style.css — this wave confirmed `.tab` did NOT previously have `position: relative`
- Added the `.tab__progress` CSS rule immediately after `.tab__status--errored` (co-located with tab-related per-tab indicator styles)
- All 12 TabBar.test.tsx tests pass; 3 Wave 0 RED tests flip GREEN

## .tab__progress Injection Point

The underline element is the LAST child inside each tab `<div>`, placed after the `.tab__close` button:

```
<div className="tab ...">
  <span className="tab__status ..." />
  {/* rename input or tab__name */}
  {/* tab__countdown (conditional) */}
  <button className="tab__close" />
  <div className="tab__progress" style={{ transform: scaleX(...) }} data-testid="tab-progress-{tab.id}" />  ← ADDED
</div>
```

This placement is intentional: the element is `position: absolute` so it does not participate in the flex layout. It was placed last in source order for clarity — the element anchors to `bottom: 0` via CSS regardless of source position.

## .tab position:relative Status

The `.tab` rule at line 103 of style.css did NOT previously have `position: relative`. It was added in this wave:

```css
.tab {
  /* ... all existing properties ... */
  position: relative; /* Phase 98 PRG-02 — anchor for .tab__progress absolute positioning */
}
```

## CSS Rule

The `.tab__progress` rule was placed after `.tab__status--errored` in style.css (near line 901), co-located with other tab-element indicator rules:

```css
.tab__progress {
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 2px;
  background: #7aa2f7;
  transform: scaleX(0);
  transform-origin: left;
  transition: transform 200ms ease-out;
  pointer-events: none;
}
```

Key design choices:
- `transform: scaleX(0)` is the initial/zero state — underline invisible at zero progress
- `transform-origin: left` — bar grows left-to-right as scaleX increases
- `transition: transform 200ms ease-out` — smooth animation on value changes without layout reflow
- `pointer-events: none` — underline does not intercept clicks on the tab
- `background: #7aa2f7` — same TokyoNight accent as `.tab--active border-bottom`

## Pitfall #4 GPU Compositor Invariant: ENFORCED

The `.tab__progress` rule uses `transform: scaleX(...)` exclusively for the width animation. The `width: 100%` CSS declaration is static and never changes. This means:

- Browser renders the underline at full width, then scales it via the transform compositor layer
- No layout reflow occurs during animation (the element's box model never changes)
- GPU handles the scaleX transform in its own composited layer
- The `transition: transform 200ms ease-out` animates on the GPU compositor thread

`style.width` is NOT set as an inline style — the plan's `progress-transform` test verifies `style.transform` matches `scaleX(...)` and `style.width` is empty (i.e., no inline width override).

## Wave 0 RED to GREEN Flip Confirmation

| Test file | Test tag | Before Wave 3 | After Wave 3 |
|-----------|----------|---------------|--------------|
| TabBar.test.tsx | progress-underline: TabBarProps interface declares tabProgress? | GREEN (flipped early at Wave 2) | GREEN |
| TabBar.test.tsx | progress-underline: renders .tab__progress element with data-testid | RED | GREEN |
| TabBar.test.tsx | progress-transform: uses transform scaleX (not width) | RED | GREEN |

All 12 TabBar.test.tsx tests pass after Wave 3.

## Cuttability State at This Wave Boundary

At the commit boundary of Wave 3 (this plan):

- **PRG-02 is complete:** Per-tab progress underline fully visible on desktop. A CLI emitting `OSC 9;4;1;47` in a terminal tab paints a ~47% accent underline at the bottom of that tab, animated smoothly via 200ms ease-out CSS transition.
- **PRG-03 remains active:** Tray icon quartile updates still work (Wave 1/2).
- **Wave 4 is the only remaining wave:** Web parity (`web/assets/terminal.js` progress arm + `#progress-underline` element), e2e Playwright scaffold, vendor_drift_test.go min-count bump, and manual UAT. Wave 4 is the canonical cuttability drop point — dropping it leaves desktop fully functional with no per-tab underline regression on web.

## Task Commits

1. **Task 1: TabBar per-tab .tab__progress element + style.css rules** — `87ed3d4` (feat)

## Files Created/Modified

- `frontend/src/components/TabBar.tsx` — tabProgress added to destructuring; .tab__progress element rendered per tab with scaleX transform and data-testid
- `frontend/src/style.css` — position:relative added to .tab rule; .tab__progress CSS rule added after .tab__status--errored

## Deviations from Plan

None — plan executed exactly as written. All sub-tasks A through F completed within Task 1:
- Sub-task A: tabProgress prop added to interface (already done at Wave 2 per deviation Rule 2)
- Sub-task B: tabProgress added to destructuring
- Sub-task C: .tab__progress element injected into tab render
- Sub-task D: style.css rules added (position:relative on .tab + .tab__progress rule)
- Sub-task E: Wave 0 tests pass as-is without test file modification (source-inspection tests; the production code now matches what they check)
- Sub-task F: PRG-OFF regression tests confirmed GREEN

## Known Stubs

None — all Wave 3 features are fully implemented. The per-tab underline renders correctly. tabProgress prop is both declared, wired (Wave 2), and consumed (this wave).

## Threat Flags

No new threat surface. T-98-01 (Tampering via untrusted CLI input to progress value) is mitigated upstream by the addon's [0,100] clamp; the CSS `transform: scaleX(value/100)` provides an additional visual clip for any value that somehow exceeds 100 (scaleX(>1) overflows the tab width but the underline is visually bounded by the tab's `overflow: hidden` if set, and the transform does not cause layout damage).

## Self-Check

- [x] `frontend/src/components/TabBar.tsx` contains `tab__progress` class name
- [x] `frontend/src/components/TabBar.tsx` contains `data-testid={`tab-progress-${tab.id}`}`
- [x] `frontend/src/components/TabBar.tsx` contains `scaleX(` transform expression
- [x] `frontend/src/components/TabBar.tsx` tabProgress added to destructuring
- [x] `frontend/src/style.css` contains `.tab__progress {` rule
- [x] `frontend/src/style.css` contains `transform: scaleX(0)` initial state
- [x] `frontend/src/style.css` contains `background: #7aa2f7` TokyoNight accent
- [x] `frontend/src/style.css` contains `position: relative` on `.tab` rule
- [x] 12/12 TabBar.test.tsx tests pass (including 3 progress-underline/progress-transform Wave 0 RED → GREEN)
- [x] Go PRG-OFF regression tests GREEN (TestPRG_OffPath_NoProgressLogic, TestPRG_NewProgressAddonIsGated, TestPRG_SetTrayProgressUsage)
- [x] TypeScript compiles cleanly (tsc --noEmit, no output)
- [x] Task commit 87ed3d4 exists with no file deletions

## Self-Check: PASSED
