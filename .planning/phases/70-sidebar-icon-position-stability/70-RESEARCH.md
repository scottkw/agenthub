# Phase 70: Sidebar Icon Position Stability - Research

**Researched:** 2026-04-13
**Domain:** CSS layout / React sidebar component
**Confidence:** HIGH

## Summary

GitHub issue #20 reports that sidebar icons appear to shift horizontally when toggling between collapsed (48px rail) and expanded (200px panel) states. The root cause is a mismatch in icon horizontal position between the two states: in expanded mode, icons are left-aligned with 8px padding (icon center at 18px from sidebar left edge), while in collapsed mode they are flex-centered in the 48px rail (icon center at 24px). This produces a visible 6px rightward jump when collapsing.

The fix is straightforward: establish a fixed-width icon column (matching the collapsed rail width of 48px) that centers the icon within it, and keep this column the same width in both expanded and collapsed states. The label text follows after this fixed column. When the sidebar collapses to 48px, only the label disappears -- the icon stays in exactly the same position because its container has not changed.

**Primary recommendation:** Replace the current `padding: 8px` + flex-start/center toggle pattern with a fixed 48px-wide icon slot that centers the icon in both states. Remove the `justify-content: center` collapsed override entirely.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SBR-02 | Sidebar icons remain in the same horizontal position whether the sidebar is collapsed or expanded -- no perceived shift when toggling the hamburger button | Root cause identified (padding/justify-content mismatch); fix pattern defined (fixed icon slot); CSS changes specified below |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **Tech stack:** React 19, TypeScript, Heroicons, Vite, Vitest (jsdom)
- **Testing:** Vitest with jsdom; existing Sidebar.test.tsx covers component behavior
- **Code conventions:** camelCase JS, PascalCase components, ESLint + Prettier, TypeScript types
- **Package manager:** pnpm preferred for frontend
- **Core principle -- Chesterton's Fence:** Before removing anything, articulate why it exists. The Phase 63 `justify-content: center` override exists because without it, the icon would flex-start inside the 48px rail. The new approach makes this override unnecessary because the icon is always centered in a fixed-width slot.

## Standard Stack

No new libraries needed. This is a CSS-only fix within the existing codebase.

### Core (already in use)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | ^19.2.4 | UI framework | Already in use |
| @heroicons/react | ^2.2.0 | SVG icons (24px outline) | Already in use |
| Vitest | ^4.1.0 | Unit tests | Already in use |

## Architecture Patterns

### Current File Structure (relevant files only)
```
frontend/src/
  App.tsx                       # Root component, renders <Sidebar> inside .app__row
  style.css                     # All CSS (single file), sidebar rules at lines 166-240
  components/
    Sidebar.tsx                 # Sidebar component (React, ~101 lines)
    __tests__/
      Sidebar.test.tsx          # Sidebar tests (4 describe blocks, ~225 lines)
```

### Current DOM Structure
```html
<nav class="sidebar sidebar--collapsed?">
  <button class="sidebar__toggle">         <!-- hamburger -->
    <svg class="sidebar__icon" />
  </button>
  <button class="sidebar__item">           <!-- Home, Remote, Sessions, New Session -->
    <svg class="sidebar__icon" />
    <span class="sidebar__label">Home</span>  <!-- conditionally rendered -->
  </button>
  ...
  <div class="sidebar__bottom">
    <button class="sidebar__item">         <!-- Settings -->
      <svg class="sidebar__icon" />
      <span class="sidebar__label">Settings</span>
    </button>
  </div>
</nav>
```

### Root Cause: The Icon Shift Math

**Expanded state** (sidebar width: 200px):
- `.sidebar__item`: `display: flex; align-items: center; gap: 8px; padding: 8px;`
- Default `justify-content: flex-start` (no explicit value)
- Icon (20px wide) left edge at 8px padding, **icon center at 18px** from sidebar left

**Collapsed state** (sidebar width: 48px):
- `.sidebar--collapsed .sidebar__item`: `justify-content: center; padding: 8px 0;`
- Icon is flex-centered in 48px, **icon center at 24px** from sidebar left

**Hamburger toggle** (always):
- `.sidebar__toggle`: `width: 38px; margin: 4px; justify-content: center;`
- **Toggle icon center at 23px** from sidebar left (4px margin + 19px half-width)

**Result:** Three different horizontal positions (18px, 23px, 24px). When toggling, items jump 6px.

### Recommended Fix: Fixed Icon Slot

The pattern: give each sidebar item a fixed-width icon area equal to the collapsed rail width (48px), with the icon centered inside it. The label comes after this fixed area. [VERIFIED: codebase inspection]

**CSS changes to `style.css`:**

```css
/* --- Sidebar item: fixed icon slot approach --- */
.sidebar__item {
  display: flex;
  align-items: center;
  gap: 0;                        /* was 8px -- slot handles spacing now */
  padding: 8px 0;                /* vertical only -- icon slot handles horizontal */
  width: 100%;
  border: none;
  background: transparent;
  color: #9aa5ce;
  cursor: pointer;
  font-size: 13px;
  font-family: inherit;
  text-align: left;
  border-radius: 0;
  transition: background-color 0.1s, color 0.1s;
}

.sidebar__icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  /* Center the 20px icon inside the 48px icon slot */
  margin: 0 14px;               /* (48 - 20) / 2 = 14px each side */
}

.sidebar__label {
  font-size: 13px;
  color: inherit;
  white-space: nowrap;
  overflow: hidden;
}

/* REMOVE the collapsed override entirely -- icon position is now
   identical in both states because the icon margin is fixed */
/* .sidebar--collapsed .sidebar__item { justify-content: center; padding: 8px 0; } */
```

**Why this works:**
- Expanded: icon occupies 20px + 14px margin on each side = 48px slot. Icon center = 14 + 10 = 24px.
- Collapsed: sidebar shrinks to 48px. Icon slot is still 48px (48px margin + 20px icon). Icon center = 24px.
- **Both states: icon center at 24px. Zero shift.**

**Hamburger alignment:** The toggle button also needs its icon center at 24px. Currently it's at 23px (38px wide with 4px margin). Options:
1. Change `.sidebar__toggle` to `width: 48px; margin: 0;` and `justify-content: center` (icon center = 24px).
2. Or apply the same margin approach: remove the explicit width and use `margin: 0 14px` on the toggle's icon.

Option 1 is cleaner -- make the toggle button fill the full 48px rail width.

### Alternative Approach: padding-left on sidebar__item

A simpler but less robust approach: set `padding-left: 14px` on `.sidebar__item` in both states, ensuring the icon left edge is always at 14px.

- Expanded icon center: 14 + 10 = 24px
- Collapsed icon center: 14 + 10 = 24px (with no justify-content override)

This is simpler but requires the icon to NOT be flex-centered. The margin approach is preferred because it keeps the icon self-contained.

### Anti-Patterns to Avoid

- **Different positioning strategies per state:** Using `justify-content: center` in collapsed but `flex-start` + padding in expanded guarantees misalignment. Icons should use the same positioning logic in both states. [VERIFIED: this is the current bug]
- **Conditional label rendering without fixed icon slot:** When `{!collapsed && <span>}` removes the label, the flex layout recalculates. If the icon position depended on the label's presence (via gap, justify-content), it shifts. Fixed icon margins are independent of label presence. [VERIFIED: codebase inspection]
- **Animating width without overflow:hidden:** The sidebar already has `overflow: hidden` and `transition: width 0.15s ease` -- this is correct. Do not remove these. [VERIFIED: style.css line 173-174]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Icon alignment math | Manual pixel calculations per state | Fixed margin on .sidebar__icon (14px each side = centered in 48px slot) | One rule, both states, zero drift |
| Smooth width transition | JavaScript animation / requestAnimationFrame | CSS `transition: width 0.15s ease` (already in place) | Hardware-accelerated, simpler, already works |

## Common Pitfalls

### Pitfall 1: Breaking the Toggle Button Alignment
**What goes wrong:** Fixing the nav items but forgetting the hamburger toggle, leaving it misaligned.
**Why it happens:** The toggle uses a different class (`.sidebar__toggle`) with its own width/margin, so changes to `.sidebar__item` don't propagate.
**How to avoid:** Verify the toggle icon's horizontal center matches the nav item icons' center (both should be 24px from left edge).
**Warning signs:** Hamburger icon appears offset from the nav icons below it.

### Pitfall 2: Label Text Appearing During Collapse Animation
**What goes wrong:** During the 0.15s width transition, the label text partially shows or creates a text reflow flicker.
**Why it happens:** The label is conditionally rendered (`{!collapsed && <span>}`), so it pops in/out instantly while the width animates. If the label appears before the sidebar is fully expanded, it can cause a flash.
**How to avoid:** The sidebar already has `overflow: hidden`, which clips the label during animation. Verify this still works after CSS changes. The React conditional render (`{!collapsed && ...}`) is fine because overflow:hidden hides any text that would extend beyond the 48px width during transition.
**Warning signs:** Flash of label text during the collapse/expand transition.

### Pitfall 3: Breaking the sidebar__bottom Push-Down
**What goes wrong:** `.sidebar__bottom { margin-top: auto; }` stops pushing Settings to the bottom.
**Why it happens:** If `flex-direction` or the parent flex container changes accidentally.
**How to avoid:** Do not modify `.sidebar` flex-direction or the `sidebar__bottom` class. Only modify `.sidebar__item` and `.sidebar__icon` positioning.
**Warning signs:** Settings button is no longer at the bottom of the sidebar.

### Pitfall 4: Padding Change Breaking Hover Background Width
**What goes wrong:** Changing from `padding: 8px` to `padding: 8px 0` means the hover highlight (`background-color: #1e2030`) no longer extends to the sidebar edges.
**Why it happens:** With `width: 100%` and `padding: 0` horizontal, the background fills the button. But if margin is on the icon instead, the button itself still spans full width. This should be fine. However, verify that the hover highlight visually looks correct.
**How to avoid:** Keep `width: 100%` on `.sidebar__item`. The hover background should still span the full sidebar width since the button is full-width.
**Warning signs:** Hover highlight appears narrower than the sidebar.

## Code Examples

### Fix Pattern: Fixed Icon Slot via Margin

```css
/* Source: Derived from root cause analysis of current codebase */

/* sidebar__item: same rules in both states */
.sidebar__item {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 8px 0;
  width: 100%;
  border: none;
  background: transparent;
  color: #9aa5ce;
  cursor: pointer;
  font-size: 13px;
  font-family: inherit;
  text-align: left;
  border-radius: 0;
  transition: background-color 0.1s, color 0.1s;
}

/* Icon always centered in 48px slot: 14px + 20px + 14px = 48px */
.sidebar__icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  margin: 0 14px;
}

/* Toggle button: fill the 48px rail width exactly */
.sidebar__toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 38px;
  margin: 4px 0;
  border: none;
  background: transparent;
  color: #9aa5ce;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.1s, color 0.1s;
}

/* DELETE this rule -- no longer needed */
/* .sidebar--collapsed .sidebar__item {
  justify-content: center;
  padding: 8px 0;
} */

.sidebar__label {
  font-size: 13px;
  color: inherit;
  white-space: nowrap;
  overflow: hidden;
}
```

### Verification: Manual Pixel Check
```
Expanded (200px sidebar):
  Icon left-margin: 14px
  Icon width: 20px
  Icon center: 14 + 10 = 24px from sidebar left edge

Collapsed (48px sidebar):
  Icon left-margin: 14px
  Icon width: 20px  
  Icon center: 14 + 10 = 24px from sidebar left edge

Toggle button:
  Button width: 48px, centered icon
  Icon center: 24px from sidebar left edge

All three: 24px. Zero shift.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `justify-content: center` in collapsed, `flex-start` in expanded | Fixed icon margin in both states | Phase 70 (this fix) | Eliminates 6px icon shift |
| `.sidebar__toggle` 38px + 4px margin | Full 48px width toggle button | Phase 70 (this fix) | Aligns hamburger with nav icons |

**Phase 63 context:** Phase 63 (Sidebar Icon Centering) added the `.sidebar--collapsed .sidebar__item { justify-content: center; padding: 8px 0; }` rule to center icons in collapsed mode. This fixed the centering within the rail but introduced the position mismatch between states. Phase 70 supersedes that rule. [VERIFIED: git show aed142d]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 48px collapsed rail width and 200px expanded width are the correct current values | Root Cause | Fix math would be wrong -- but verified in style.css lines 170, 178 |

All other claims were verified via codebase inspection. The assumption above was also verified, so effectively no unverified claims remain.

## Open Questions

None. The root cause, fix strategy, and verification approach are all well-defined from codebase inspection.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 + jsdom |
| Config file | `frontend/vitest.config.ts` |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SBR-02a | Icon horizontal position same in both states | CSS unit test (computed style) | `cd frontend && pnpm test -- Sidebar` | Partially (Sidebar.test.tsx exists, needs new test) |
| SBR-02b | No perceived horizontal jump | Visual / manual | Manual toggle check in running app | N/A -- manual-only |
| SBR-02c | Smooth transition, no reflow flicker | Visual / manual | Manual toggle check in running app | N/A -- manual-only |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm test`
- **Per wave merge:** `cd frontend && pnpm test`
- **Phase gate:** Full suite green + manual visual verification of sidebar toggle

### Wave 0 Gaps
- [ ] New test case in `Sidebar.test.tsx` -- verify `.sidebar__icon` has `margin: 0 14px` (CSS unit test for position stability contract)
- [ ] Alternatively: test that `.sidebar--collapsed .sidebar__item` does NOT have `justify-content: center` rule (anti-regression)

Note: True pixel-position verification requires a real browser (not jsdom). The CSS-level test verifies the structural contract. Visual verification is manual.

## Security Domain

Not applicable -- this phase is a CSS-only layout fix with no security surface.

## Sources

### Primary (HIGH confidence)
- `frontend/src/style.css` lines 166-240 -- complete sidebar CSS rules [VERIFIED: Read tool]
- `frontend/src/components/Sidebar.tsx` -- full component source, 101 lines [VERIFIED: Read tool]
- `frontend/src/App.tsx` lines 576-584 -- sidebar integration in app layout [VERIFIED: Read tool]
- `git show aed142d` -- Phase 63 commit that added the collapsed centering rule [VERIFIED: git]
- GitHub issue #20 -- original bug report describing the icon shift [VERIFIED: gh issue view]
- `frontend/src/components/__tests__/Sidebar.test.tsx` -- existing test coverage [VERIFIED: Read tool]

### Secondary (MEDIUM confidence)
None needed -- all research was from direct codebase inspection.

### Tertiary (LOW confidence)
None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new libraries, CSS-only change
- Architecture: HIGH -- full codebase inspection, exact pixel math verified
- Pitfalls: HIGH -- identified from actual code structure and prior Phase 63 history

**Research date:** 2026-04-13
**Valid until:** Indefinite (CSS layout fundamentals, no external dependency versioning concerns)
