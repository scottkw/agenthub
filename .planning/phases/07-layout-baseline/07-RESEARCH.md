# Phase 7: Layout Baseline - Research

**Researched:** 2026-03-19
**Domain:** xterm.js FitAddon layout, CSS flexbox height chains, Wails/React desktop UI hit targets
**Confidence:** HIGH

## Summary

Phase 7 fixes two independent problems in the existing codebase: (1) the terminal not filling available vertical space, and (2) toolbar buttons being too small to click comfortably.

For the terminal fill issue, the code already has most of the right pieces in place. `TerminalPanel` uses `flex: 1; minHeight: 0` on its container div, `ResizeObserver` calls `fitAddon.fit()` on dimension changes, and `requestAnimationFrame` defers fit after `display:none → flex` transitions. The root cause of any remaining dead space is almost certainly a broken flex height chain somewhere between `#root` and the terminal container — specifically the `min-height: 0` trap on intermediate flex children, or the `terminal-container` not having `height: 100%` instead of `flex: 1` when its parent has an explicit height. Diagnosis requires reading the existing CSS carefully and auditing every element in the stack.

For the toolbar buttons, the current `.tab-bar__btn` is `28x28px`, which is below the 36–44px minimum comfortable hit target recommended by Apple HIG and Material Design. The fix is a pure CSS change: increase width/height to `36px` or `40px`, adjust font size accordingly, and optionally add minimum touch-target padding. The tab bar itself is `height: 36px` — if buttons grow, the bar height must grow proportionally, which adds a few pixels to the bar but does not affect the terminal fill calculation (flex-shrink handles it).

**Primary recommendation:** Fix the flex height chain for TERM-01 (audit every ancestor of `.terminal-container` for missing `min-height: 0` or wrong height property), then bump `.tab-bar__btn` and `.tab-bar` dimensions for UILAY-01. Both are CSS-only changes with no JavaScript required beyond ensuring `fit()` is called after tab bar height changes.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TERM-01 | Terminal content fills all available space in each tab with no dead space | Flex height chain audit + FitAddon.fit() timing; existing ResizeObserver infrastructure already handles dynamic sizing |
| UILAY-01 | Toolbar buttons are visually larger and easy to click (36–44px hit target) | Pure CSS: increase `.tab-bar__btn` width/height and `.tab-bar` height; no JS changes needed |
</phase_requirements>

## Standard Stack

### Core (already installed — no new dependencies needed)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @xterm/xterm | ^6.0.0 | Terminal emulator | Already in use |
| @xterm/addon-fit | ^0.11.0 | Fit terminal to container | Already in use |
| React | ^19.2.4 | UI framework | Already in use |

### No New Dependencies

This phase requires zero new npm packages. All changes are CSS and minor TSX adjustments.

**Installation:**
```bash
# Nothing to install
```

**Version verification:** Current installed versions confirmed from `frontend/package.json` — all packages are already present and correct.

## Architecture Patterns

### Current Layout Tree (annotated)

```
html  (height: 100%)
  body  (height: 100%)
    #root  (height: 100%)
      .app  (display:flex; flex-direction:column; height:100%)
        .tab-bar  (flex-shrink:0; height:36px)
        .terminal-container  (flex:1; overflow:hidden; position:relative)
          .terminal-wrapper  (display:flex; flex-direction:column; width:100%; height:100%)
            .web-serving-bar  (flex-shrink:0)
            TerminalPanel > div  (flex:1; width:100%; minHeight:0)
              xterm.js internals
```

The `.app → .terminal-container → .terminal-wrapper → TerminalPanel div` chain looks correct on paper. However there are two things to verify:

1. `.terminal-container` uses `flex: 1` (not `height: 100%`) — in a flex column parent, this is correct and should work. But `position: relative` combined with `overflow: hidden` can interact badly if a child tries to use `height: 100%` instead of `flex: 1`.

2. The inner `TerminalPanel` div uses `flex: 1; minHeight: 0` inline — this is the correct xterm.js pattern. `minHeight: 0` is essential because flex children default to `min-height: auto`, which allows them to overflow rather than shrink.

### Pattern 1: Flex Height Chain for Full-Height Terminals

**What:** Every ancestor in a vertical flex stack must participate correctly. The most common failure is a missing `min-height: 0` on intermediate flex children.

**When to use:** Any time xterm.js (or any content-sized element) must fill remaining space in a flex container.

**The canonical chain:**
```css
/* Parent flex container */
.parent {
  display: flex;
  flex-direction: column;
  height: 100%;        /* or flex: 1 if parent is also a flex child */
}

/* Flex child that must fill remaining space */
.child {
  flex: 1;
  min-height: 0;       /* CRITICAL — prevents flex child from overflowing */
  overflow: hidden;
}
```

**Source:** MDN Flexbox documentation — `min-height: auto` is the default for flex items, causing them to size to content rather than container.

### Pattern 2: xterm.js FitAddon Timing Requirements

**What:** `fitAddon.fit()` measures the container's DOM dimensions. It MUST be called after the container has a real size in the DOM.

**When fit() must be called:**
1. After initial mount (use `requestAnimationFrame` to defer past browser paint)
2. After `display: none → display: flex` transition (tab switch)
3. After any container resize (use `ResizeObserver`)
4. After tab bar height changes (if tab bar grows, terminal container shrinks)

**Current code already does this correctly** in `TerminalPanel.tsx`:
- Mount: `requestAnimationFrame(() => fitAddon.fit())`
- Active change: `requestAnimationFrame(() => fitAddonRef.current?.fit())` + `ResizeObserver`

**The one gap:** If the tab bar height changes (UILAY-01 increases it), the ResizeObserver on the terminal container will fire automatically — no additional code needed.

### Pattern 3: Minimum Touch/Click Target Sizes

**What:** Buttons should be at least 36–44px in their smaller dimension for comfortable clicking.

**Reference targets:**
- Apple HIG: 44x44pt minimum touch target
- Material Design 3: 48x48dp minimum, can go to 40dp with extra padding
- Practical desktop minimum: 36px is acceptable for mouse-only (no touch)

**Current state:** `.tab-bar__btn` is `28x28px` — too small.

**Target state:** `36x36px` minimum, `40x40px` preferred. This requires the tab bar itself to grow from `36px` to at least `40px` height.

**CSS change:**
```css
/* Before */
.tab-bar {
  height: 36px;
}
.tab-bar__btn {
  width: 28px;
  height: 28px;
  font-size: 16px;
}

/* After */
.tab-bar {
  height: 42px;   /* accommodate 38px buttons with padding */
}
.tab-bar__btn {
  width: 38px;
  height: 38px;
  font-size: 18px;
}
```

### Recommended Project Structure

No structural changes needed. All changes are within:
```
frontend/src/
├── style.css          # Primary change target (tab bar + button sizing)
└── components/
    └── TerminalPanel.tsx  # Possibly: verify inline styles are correct
```

### Anti-Patterns to Avoid

- **`height: 100%` on a flex child:** Works only when the parent has an explicit height, not when parent uses `flex: 1`. Use `flex: 1; min-height: 0` instead.
- **Calling `fit()` synchronously after mount:** xterm.js container has no dimensions yet. Always wrap in `requestAnimationFrame`.
- **Observing the window resize event instead of ResizeObserver:** Window resize doesn't capture all layout changes (e.g., sibling element grows/shrinks). Current code already uses ResizeObserver — do not regress to window listener.
- **Setting fixed height on `.terminal-container`:** Breaks responsiveness. Keep it as `flex: 1`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal sizing to container | Custom dimension calculation | `@xterm/addon-fit` FitAddon | Handles rows/cols calculation, char metrics, padding |
| Container resize detection | `window.addEventListener('resize', ...)` | `ResizeObserver` | Fires on any dimension change, not just window resize |

**Key insight:** FitAddon already handles all the math for mapping pixel dimensions to terminal columns/rows. The only job here is ensuring the container div has correct CSS dimensions before `fit()` is called.

## Common Pitfalls

### Pitfall 1: The `min-height: 0` Trap
**What goes wrong:** Terminal renders at 0px height or doesn't grow to fill space.
**Why it happens:** Flex items default to `min-height: auto`, which means they size to their content minimum. xterm.js internal elements have intrinsic size, so the terminal div expands past its container instead of being constrained.
**How to avoid:** Every flex child in the height chain that needs to grow/shrink must have `min-height: 0` (or `overflow: hidden`).
**Warning signs:** Terminal has correct columns but very few rows; blank space appears below the last rendered line.

### Pitfall 2: fit() Called Before Container Has Dimensions
**What goes wrong:** `fitAddon.fit()` throws or computes cols=0/rows=0; terminal appears blank or very small.
**Why it happens:** The container div is in the DOM but browser hasn't completed layout paint.
**How to avoid:** Always call `fit()` inside `requestAnimationFrame(() => ...)` for initial mount and tab switches.
**Warning signs:** Console warning from xterm.js about invalid dimensions; terminal appears with 1 row.

### Pitfall 3: Tab Switch Causes Layout Collapse
**What goes wrong:** Switching to a tab that was previously visible shows a terminal with wrong dimensions.
**Why it happens:** `display: none` makes the container invisible — when it transitions back to `display: flex`, the ResizeObserver fires but may fire before layout is complete.
**How to avoid:** The existing `requestAnimationFrame` defer on the `isActive` effect handles this. Do not remove it.
**Warning signs:** Terminal is sized to previous dimensions after tab switch; requires manual window resize to fix.

### Pitfall 4: Tab Bar Height Growth Breaks Terminal Fill
**What goes wrong:** Increasing the tab bar height (for UILAY-01) causes the terminal to overflow or have dead space.
**Why it happens:** If tab bar uses `flex-shrink: 0` and `.terminal-container` uses `flex: 1`, the terminal container will automatically shrink. The ResizeObserver on the terminal container will fire `fit()`. This is correct behavior — no special handling needed.
**How to avoid:** Do not set a fixed height on `.terminal-container`. Keep it as `flex: 1`.
**Warning signs:** Would only be a problem if `.terminal-container` had a hardcoded height.

### Pitfall 5: `web-serving-bar` Missing `flex-shrink: 0`
**What goes wrong:** The web serving bar (shown above the terminal in each tab) compresses or overlaps the terminal.
**Current state:** `web-serving-bar` has `flex-shrink: 0` — this is correct.
**Warning signs:** Web serving bar text gets cut off; terminal starts above the expected position.

## Code Examples

### Correct Full-Height Flex Chain
```css
/* Source: MDN Flexbox + xterm.js FitAddon documentation */

/* Root shell */
html, body, #root {
  height: 100%;
  overflow: hidden;
}

/* App: flex column */
.app {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* Fixed-height header */
.tab-bar {
  flex-shrink: 0;
  height: 42px;    /* increased from 36px for UILAY-01 */
}

/* Fills remaining space */
.terminal-container {
  flex: 1;
  min-height: 0;   /* belt-and-suspenders with overflow:hidden */
  overflow: hidden;
}

/* Full height within container */
.terminal-wrapper {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
}

/* Fixed-height optional bar */
.web-serving-bar {
  flex-shrink: 0;
}

/* Terminal div: fills remaining space in wrapper */
/* (This is the inline style in TerminalPanel.tsx) */
.terminal-div {
  flex: 1;
  width: 100%;
  min-height: 0;
}
```

### Correct FitAddon Timing Pattern
```typescript
// Source: @xterm/addon-fit README + xterm.js docs

// On mount — defer past browser paint
useEffect(() => {
  // ... setup ...
  requestAnimationFrame(() => {
    fitAddon.fit()
  })
}, [sessionId])

// On active change — defer + observe
useEffect(() => {
  if (!isActive || !containerRef.current) return
  const rafId = requestAnimationFrame(() => {
    fitAddonRef.current?.fit()
  })
  const ro = new ResizeObserver(() => {
    fitAddonRef.current?.fit()
  })
  ro.observe(containerRef.current)
  return () => {
    cancelAnimationFrame(rafId)
    ro.disconnect()
  }
}, [isActive])
```

### Toolbar Button Target Size
```css
/* Source: Apple HIG (44pt), Material Design 3 (48dp recommended, 40dp minimum) */

.tab-bar {
  height: 42px;    /* was 36px — accommodate 38px buttons */
  flex-shrink: 0;
}

.tab-bar__btn {
  width: 38px;     /* was 28px */
  height: 38px;    /* was 28px */
  font-size: 18px; /* was 16px */
  /* existing: border-radius: 4px, transitions — keep */
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `window.resize` for terminal fit | `ResizeObserver` | xterm.js v4+ era | Fires on any layout change, not just window |
| `height: 100%` on flex children | `flex: 1; min-height: 0` | CSS Flexbox spec clarification | Reliable height chain without fixed-pixel ancestors |
| Synchronous `fit()` | `requestAnimationFrame(() => fit())` | xterm.js FitAddon docs | Ensures DOM layout complete before measuring |

**Deprecated/outdated:**
- `window.onresize` for terminal sizing: ResizeObserver is the correct replacement. Current code is already correct.

## Open Questions

1. **Is there actually a visual bug with terminal fill right now?**
   - What we know: TERM-01 is listed as unfixed; STATE.md notes "flex `min-height: 0` trap must be fixed"; existing code appears to have `minHeight: 0` on the TerminalPanel div
   - What's unclear: Whether the bug manifests on specific conditions (new tab, tab switch, window resize) or always; the `.terminal-container` CSS does not have `min-height: 0` explicitly
   - Recommendation: Planner should add a diagnostic step: run the app and check if blank space appears below terminal output. If `.terminal-container` is missing `min-height: 0`, add it.

2. **Does the tab bar height increase affect the web-serving-bar layout?**
   - What we know: `.web-serving-bar` is inside `.terminal-wrapper`, not inside `.tab-bar` — so tab bar height changes don't affect it
   - What's unclear: Nothing — the layout is clean
   - Recommendation: No action needed

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (vitest configured inline) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TERM-01 | Terminal container has correct flex CSS (flex:1, min-height:0) | unit | `pnpm test -- --reporter=verbose` | Wave 0 — needs test file |
| TERM-01 | Tab switch does not collapse terminal (isActive effect cleanup) | unit | `pnpm test -- --reporter=verbose` | Wave 0 — needs test file |
| UILAY-01 | Tab bar buttons are >= 36px in both dimensions | manual-only | Visual inspection in running app | N/A — CSS dimension, not logic |
| UILAY-01 | Tab bar height accommodates larger buttons without overflow | manual-only | Visual inspection | N/A — CSS |

**Note on UILAY-01:** Button hit target size is a visual/CSS property. It cannot be meaningfully unit tested (jsdom has no layout engine). Manual verification via running the app is the only reliable approach.

**Note on TERM-01:** The core correctness (flex chain, min-height) can be verified by testing the CSS classes and inline styles produced by components. Full layout behavior requires a real browser.

### Wave 0 Gaps

- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — covers TERM-01 (verifies inline styles on container div: `flex: 1`, `minHeight: 0`, `width: 100%`)
- [ ] `frontend/src/components/__tests__/TabBar.test.tsx` — covers UILAY-01 structure (tab-bar classnames present; note: pixel dimensions require visual verification)

*(Existing `vitest` infrastructure is in place — only test files are missing)*

## Sources

### Primary (HIGH confidence)
- Codebase direct read: `frontend/src/style.css`, `frontend/src/App.tsx`, `frontend/src/components/TerminalPanel.tsx`, `frontend/src/components/TabBar.tsx` — full understanding of current layout and component structure
- `frontend/package.json` — confirmed versions of xterm.js, FitAddon, React

### Secondary (MEDIUM confidence)
- MDN Flexbox documentation — `min-height: 0` on flex items, flex height chain patterns
- Apple HIG — 44pt minimum touch target recommendation
- Material Design 3 — 48dp recommended / 40dp minimum button target

### Tertiary (LOW confidence)
- None — all critical findings are based on direct codebase inspection

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — read directly from package.json and source files
- Architecture: HIGH — complete source code available; layout chain fully traceable
- Pitfalls: HIGH — identified from direct code inspection; `min-height: 0` pattern is well-documented CSS behavior
- Button sizing: HIGH — requirements spec calls for 36–44px; Apple/Material guidelines confirm range

**Research date:** 2026-03-19
**Valid until:** Stable — CSS flexbox and xterm.js FitAddon behavior are stable; no version changes expected in this timeframe
