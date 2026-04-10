# Phase 64: Terminal Padding - Research

**Researched:** 2026-04-10
**Domain:** CSS + xterm.js layout — inset padding without breaking column/row calculation
**Confidence:** HIGH

## Summary

Phase 64 adds visible inset padding inside each terminal session so text does not touch the frame edges. The implementation is a pure CSS addition — no JavaScript changes are required.

The critical finding is that `fitTerminal()` in `TerminalPanel.tsx` already accounts for padding. Lines 24-29 read `paddingLeft`, `paddingRight`, `paddingTop`, and `paddingBottom` from `term.element` (the `.xterm` div) and subtract them from available space before computing `cols` and `rows`. This means the column/row calculation is already padding-aware — padding only needs to be applied via CSS on `.xterm`.

The correct target for padding is `.xterm` in `style.css`. This is the element that `term.open(container)` creates inside `containerRef`. Its computed padding is what `fitTerminal()` reads. Adding `padding: 6px 8px` (or similar) to `.xterm` in the project's CSS overrides the upstream default (no padding) and is immediately accounted for by the custom fit function. No JS change and no new dependencies are needed.

**Primary recommendation:** Add a single CSS rule `.xterm { padding: 6px 8px; }` to `frontend/src/style.css` in the xterm overrides block. The value 6px vertical / 8px horizontal gives a balanced inset that matches the app's existing 8px spacing rhythm (e.g., `.sidebar__item` uses `padding: 8px`).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PAD-01 | User sees terminal content inset from the edges with consistent padding | CSS rule `.xterm { padding: 6px 8px; }` in the xterm overrides section of `style.css`. `fitTerminal()` already reads `term.element` padding and subtracts it from available space, so no JS change is needed. All open sessions share the same `.xterm` selector — consistency across sessions is automatic. |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/xterm` | 6.0.0 | Terminal emulator — creates `.xterm` element that receives CSS padding | Already in use |
| Vitest | ^4.1.0 | Test runner (jsdom environment) | Project standard; `TerminalPanel.test.tsx` already exists |

No new dependencies are required for this phase.

### Supporting

None. This is CSS-only.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `.xterm { padding }` CSS rule | Inline `padding` style on `containerRef` div in React JSX | Padding on the container div would NOT be read by `fitTerminal()` — it reads from `term.element`, not from its parent. Wrong target. |
| `.xterm { padding }` CSS rule | Wrap xterm in an inner `<div>` with padding | Adds DOM complexity; `fitTerminal()` reads the `term.element`, not a wrapper. Extra layout layer needed to make the parent dimension calculation work. |
| `.xterm { padding }` CSS rule | xterm.js `scrollback` option / terminal options | xterm.js has no built-in padding option [VERIFIED: xterm.css, TerminalPanel.tsx source] |
| Fixed 6px/8px values | CSS custom property `--terminal-padding` | Overkill for a fixed value; REQUIREMENTS.md explicitly says "Configurable padding value" is Out of Scope |

**Installation:** No packages to install.

**Version verification:** [VERIFIED: /Users/ken/dev/agenthub/frontend/package.json — @xterm/xterm ^6.0.0, installed 6.0.0]

## Architecture Patterns

### Recommended Project Structure

No structural changes needed. The single file change is:

```
frontend/src/
└── style.css    # Add .xterm { padding: 6px 8px; } in the "xterm overrides" block
```

### Pattern: CSS Override in xterm Overrides Block

**What:** `style.css` already has a designated `/* ─── xterm overrides ───... */` section at the top (lines 3–11) that overrides xterm.js defaults. The scrollbar hide rule lives there. Padding belongs there too.

**When to use:** Any time you need to override an xterm.js default CSS behavior at the project level.

**Example:**
```css
/* Source: frontend/src/style.css — xterm overrides block */
/* Hide native scrollbar so FitAddon uses full container width. */
.xterm-viewport {
  scrollbar-width: none;           /* Firefox */
}
.xterm-viewport::-webkit-scrollbar {
  display: none;                   /* WebKit (Wails, Chrome, Safari) */
}

/* Add padding so terminal text does not touch the frame edges. */
.xterm {
  padding: 6px 8px;
}
```

### How fitTerminal() Already Handles Padding

[VERIFIED: frontend/src/components/TerminalPanel.tsx lines 20-29]

```typescript
// fitTerminal reads elStyle from term.element — this IS the .xterm div.
const elStyle = window.getComputedStyle(term.element!)
const padH = parseInt(elStyle.paddingLeft) + parseInt(elStyle.paddingRight)
const padV = parseInt(elStyle.paddingTop) + parseInt(elStyle.paddingBottom)

const cols = Math.max(2, Math.floor((parentW - padH) / dims.css.cell.width))
const rows = Math.max(1, Math.floor((parentH - padV) / dims.css.cell.height))
```

When `padding: 6px 8px` is applied to `.xterm`, `padH` = 16 and `padV` = 12 are subtracted before computing cols/rows. The terminal will not overflow or leave blank strips — it simply uses fewer columns and rows to fit within the padded area.

### Anti-Patterns to Avoid

- **Padding on containerRef div instead of `.xterm`:** `fitTerminal()` reads `term.element` (the `.xterm` div), not the container div. Applying padding to the container would create a gap but `fitTerminal()` would compute the wrong number of columns (it still sees the full container width), producing a right-side overrun or clipping.
- **Using `term.options` or xterm constructor options for padding:** xterm.js 6.0.0 has no `padding` terminal option. [VERIFIED: TerminalPanel.tsx constructor — no padding option used]
- **Configurable padding via settings:** REQUIREMENTS.md explicitly marks "Configurable padding value" as Out of Scope.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Accounting for padding in col/row calculation | Custom math in new utility | `fitTerminal()` already handles it | The function already reads `elStyle.paddingLeft/Right/Top/Bottom` and subtracts from available space. It was written for this exact scenario. |
| Cross-session consistency | Session-specific CSS class or prop | Single `.xterm` CSS rule | All xterm instances use the `.xterm` class — one rule covers all sessions |

**Key insight:** The hardest part (padding-aware column/row math) is already implemented. This phase is purely additive CSS.

## Runtime State Inventory

> SKIPPED — This is a greenfield CSS-only addition, not a rename/refactor/migration phase.

## Common Pitfalls

### Pitfall 1: Wrong padding target — container div instead of .xterm
**What goes wrong:** Padding applied to the `containerRef` div (the React-rendered wrapper) creates a visual gap, but `fitTerminal()` computes cols/rows from `parentW` (the container's full width) minus `term.element` padding (zero). Result: terminal renders more columns than fit, causing text to extend into the padding zone or clip.
**Why it happens:** The `.xterm` element is created by `term.open()` inside the container; the container and `term.element` are different DOM nodes.
**How to avoid:** Apply padding only to `.xterm` (the element `term.element` points to), never to the React container wrapper div.
**Warning signs:** Terminal text still touches or overruns the frame edge after the change.

### Pitfall 2: Adding padding after term.open() causes a stale dimension read
**What goes wrong:** CSS is applied after the terminal initializes but before the first `fitTerminal()` call during the `isActive` effect's rAF loop. If `getComputedStyle` is called on `.xterm` before the CSS rule is parsed and applied, `padH`/`padV` would be zero.
**Why it happens:** Style sheet parsing is synchronous when styles are in the same CSS file imported at module load time. Since the padding rule lives in `style.css` (imported at the top of `main.tsx`), the rule is always applied before any `term.open()` call.
**How to avoid:** Put the padding rule in `style.css` (not a dynamically injected style sheet). This is already the intended approach.
**Warning signs:** Only a concern if padding were injected dynamically at runtime.

### Pitfall 3: Padding breaks the ResizeObserver-driven refit
**What goes wrong:** On window resize, `ResizeObserver` triggers `fitTerminal()`. If padding is on the container div (not `.xterm`), `parentW`/`parentH` already includes padding deduction by the browser, but `padH`/`padV` from `term.element` would be zero — double-deducting the space.
**Why it happens:** The CSS box model: if the container has `box-sizing: border-box` (set by `* { box-sizing: border-box }` in `style.css`), `clientWidth` already excludes the padding. But `fitTerminal()` reads `parentStyle.width` via `getComputedStyle` which returns the content-box width.
**How to avoid:** Apply padding to `.xterm` only, which is what `fitTerminal()` is designed to read. With `box-sizing: border-box` on `*`, the `.xterm` element's own layout is unaffected because padding on `.xterm` is correctly read as `padH`/`padV`.
**Warning signs:** Columns miscalculated after window resize.

## Code Examples

### CSS rule to add (verified approach)
```css
/* Source: frontend/src/style.css — xterm overrides block */
/* Inset terminal text from the container edges (PAD-01). */
.xterm {
  padding: 6px 8px;
}
```

### fitTerminal() padding path (no changes required)
```typescript
// Source: frontend/src/components/TerminalPanel.tsx lines 20-29
// [VERIFIED] — already reads padding from term.element
const elStyle = window.getComputedStyle(term.element!)
const padH = parseInt(elStyle.paddingLeft) + parseInt(elStyle.paddingRight)
const padV = parseInt(elStyle.paddingTop) + parseInt(elStyle.paddingBottom)

const cols = Math.max(2, Math.floor((parentW - padH) / dims.css.cell.width))
const rows = Math.max(1, Math.floor((parentH - padV) / dims.css.cell.height))
```

### Vitest source-text test pattern (from TerminalPanel.test.tsx)
```typescript
// Source: frontend/src/components/__tests__/TerminalPanel.test.tsx lines 6, 9
import raw from '../TerminalPanel.tsx?raw'
const cssRaw = readFileSync(resolve(__dir, '../../style.css'), 'utf-8')

// Test CSS rule presence:
it('xterm element has padding for PAD-01', () => {
  expect(cssRaw).toMatch(/\.xterm\s*\{[^}]*padding/)
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| FitAddon.fit() for sizing | Custom `fitTerminal()` that reads `term.element` padding | Phase implementation (existing) | Padding is automatically accounted for — no additional fit math needed |
| Scrollbar width hard-coded in FitAddon | Hidden scrollbar + full-width fit | Phase before 64 | Adjacent concern — not affected by padding change |

**Deprecated/outdated:**
- `FitAddon.fit()`: Not used in this project. The custom `fitTerminal()` replaced it specifically because `FitAddon` hard-codes a 14px scrollbar deduction. [VERIFIED: TerminalPanel.tsx comment lines 8-10]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `6px 8px` is an appropriate padding value aesthetically | Standard Stack / Code Examples | Purely visual — if wrong, just change the numbers; no logic impact |

**All other claims in this research were verified against the source code.**

## Open Questions

1. **Exact padding value (6px 8px vs other values)**
   - What we know: 6px vertical / 8px horizontal matches the app's 8px spacing rhythm (`.sidebar__item padding: 8px`, `.tab-status-bar padding: 0 8px`) [VERIFIED: style.css]
   - What's unclear: Whether 6px vertical feels correct at various font sizes (14px default)
   - Recommendation: Use 6px/8px; if the reviewer wants to adjust, it is a one-line change

## Environment Availability

> SKIPPED — Phase 64 is a pure CSS change. No external dependencies beyond the existing frontend build toolchain (Vite + Vitest, confirmed already installed at `/Users/ken/dev/agenthub/frontend/node_modules`).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest ^4.1.0 |
| Config file | `frontend/vite.config.ts` (test.environment: jsdom) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PAD-01 | `.xterm` CSS rule contains `padding` property | unit (source text) | `pnpm test` | ❌ Wave 0 — add to `TerminalPanel.test.tsx` |
| PAD-01 | `fitTerminal()` reads `paddingLeft`/`paddingRight`/`paddingTop`/`paddingBottom` from `term.element` | unit (source text) | `pnpm test` | ✅ (implicitly via `TerminalPanel.test.tsx` source checks — can add explicit assertion) |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — add PAD-01 assertion: `expect(cssRaw).toMatch(/\.xterm\s*\{[^}]*padding/)` (file already exists, needs one new `it` block)

## Security Domain

> This phase makes no network requests, handles no user input, processes no credentials, and introduces no new dependencies. No ASVS categories apply.

## Sources

### Primary (HIGH confidence)
- [VERIFIED: /Users/ken/dev/agenthub/frontend/src/components/TerminalPanel.tsx] — `fitTerminal()` implementation; padding subtraction in col/row math (lines 20-29)
- [VERIFIED: /Users/ken/dev/agenthub/frontend/src/style.css] — xterm overrides block, terminal-container/wrapper layout, 8px spacing rhythm
- [VERIFIED: /Users/ken/dev/agenthub/frontend/node_modules/@xterm/xterm/css/xterm.css] — `.xterm` rule has no padding by default; `.xterm-viewport` and `.xterm-screen` layout
- [VERIFIED: /Users/ken/dev/agenthub/.planning/REQUIREMENTS.md] — PAD-01 definition; "Configurable padding value" is Out of Scope
- [VERIFIED: /Users/ken/dev/agenthub/frontend/src/components/__tests__/TerminalPanel.test.tsx] — existing test patterns (source-text assertions, cssRaw reads)

### Secondary (MEDIUM confidence)
- None needed — all critical claims verified from source.

### Tertiary (LOW confidence)
- [ASSUMED: A1] — 6px/8px is visually appropriate padding value (aesthetic judgment, not a logic fact)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified from package.json and installed node_modules
- Architecture: HIGH — verified by reading fitTerminal() source; padding path is explicit
- Pitfalls: HIGH — derived from reading the actual fitTerminal() logic; not speculation
- Test patterns: HIGH — existing test file uses identical pattern (cssRaw source-text assertions)

**Research date:** 2026-04-10
**Valid until:** Stable indefinitely — depends only on xterm.js 6.0.0 (locked) and local source files
