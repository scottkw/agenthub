# Phase 135: Accessibility Hardening - Research

**Researched:** 2026-06-18
**Domain:** Web accessibility (WCAG 2.1, keyboard navigation, focus management, motion, colorblind safety) in a Wails v2 + React/TypeScript desktop app
**Confidence:** HIGH (all findings verified against the actual source files in this repository)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

### Claude's Discretion
All implementation choices (focus trap approach, CSS pattern, test strategy) are Claude's discretion. Research should recommend the single best approach for each gap.

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.

### Standing project constraints (release-blocking, from user memory)
- **Colorblind-safe verification:** User is colorblind. ALL color-based verification must be done at source level against hex constants in code, NEVER by eye.
- **Cross-surface parity is release-blocking:** GUI/TUI/CLI must stay in sync. If an a11y affordance has a parity implication across surfaces, surface it — do not silently defer.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| A11Y-01 | Attention and status conveyed by motion + icon + position, never by color alone (colorblind-safe) | STATUS_CONFIG audit — all 6 statuses have unique icon shape + text label; color is reinforcement only. Verification only required. |
| A11Y-02 | Cards keyboard-focusable; Enter/Space expands; Escape closes modal and returns focus to originating card | Partially implemented. `:focus-visible` upgrade and `aria-pressed` on filter pills are the open gaps. WR-05 Escape scope fix required. |
| A11Y-03 | Pulse and expand/collapse animations honor `prefers-reduced-motion`, falling back to static border + icon | Partially implemented. Two CSS additions required: spin animation gate and card hover transition gate. |
| A11Y-04 | Modal traps focus while open | NOT implemented (WR-06 deferred comment in HubModal.tsx line 156). `inert` attribute approach recommended with manual Tab-trap fallback. |
</phase_requirements>

---

## Summary

Phase 135 is a hardening pass — not new feature work. The Hub surface built in Phases 131–134 is already ~80% accessible. The six concrete gaps identified in the UI-SPEC are fully specified and require targeted, low-risk changes across two files: `style.css` and `HubModal.tsx`, plus a one-line prop addition in `HubFilterBar.tsx`.

The most substantive work is **GAP-135-D (A11Y-04: modal focus trap)**, which requires implementing the `inert` attribute approach against the Hub background when the modal is open, plus initial-focus placement. The `inert` attribute has been supported in all major browsers since Chrome 102 / Safari 15.5 / Firefox 112 (2022–2023). The Wails macOS webview uses WebKit (WKWebView) — Safari 15.5+ support means `inert` works on macOS. On Windows, Wails uses WebView2 (Chromium-based); `inert` has been in Chromium since version 102 (2022). Both runtimes in use here fully support `inert`. **jsdom 29 does NOT implement `inert` focus-suppression** — tests for A11Y-04 must use source-inspection (`?raw`) or manual `tabIndex` assertion patterns, not rely on jsdom natively enforcing the inert focus barrier.

The five CSS gaps (GAP-135-A, GAP-135-E, GAP-135-F plus the WR-05 Escape scope fix) are mechanical CSS additions following patterns that already exist in `style.css`. The `aria-pressed` addition (GAP-135-B) is a single prop on each filter pill button.

**Primary recommendation:** Implement all six gaps in a single wave. No new packages required. No new architecture. Every change follows a pattern already proven in the codebase.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Focus ring styling (`:focus-visible`) | Browser / CSS | — | Pure CSS selector upgrade; no JS logic change |
| Focus trap while modal open | Browser / React | CSS (`inert` reinforcement) | `inert` attribute set/unset by React `useEffect` in HubModal; CSS is defensive-only |
| `prefers-reduced-motion` gating | Browser / CSS | React (phase machine) | CSS media query gates animations; HubModal.tsx already reads matchMedia for phase machine |
| Colorblind-safe status verification | Source-level audit | — | No runtime tier; verification of hex constants in STATUS_CONFIG at source |
| `aria-pressed` on filter pills | Frontend / React | — | Single prop addition in HubFilterBar.tsx |
| Escape key handler scope (WR-05) | Frontend / React | — | Move listener from document-level to dialog `onKeyDown` in HubModal.tsx |

---

## Standard Stack

### Core — No new packages

This phase introduces **zero new dependencies**. All changes are:
- CSS additions to `frontend/src/style.css`
- React prop/hook additions to `HubModal.tsx` and `HubFilterBar.tsx`
- Source-level verification of `SessionCard.tsx`

| Tool | Version (project) | Purpose | Source |
|------|------------------|---------|---------| 
| Vitest | 4.1.0 | Test runner (existing) | `frontend/package.json` |
| jsdom | 29.0.0 | DOM environment for tests (existing) | `frontend/package.json` |
| @heroicons/react | existing | Icon library (existing) | Hub component imports |

**No `npm install` step required for this phase.**

---

## Package Legitimacy Audit

> No packages are installed in this phase. Audit is not applicable.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
User input (keyboard)
       |
       v
[HubPanel] — window keydown '/' → focus searchRef → [HubFilterBar search input]
       |
       |— Tab navigation → [SessionCard tabIndex=0] per card
       |                         |
       |                         |— onKeyDown Enter/Space → onCardClick(session, rect)
       |                         |
       v                         v
[HubModal overlay] ← mounts ← HubPanel.activeSession state
       |
       |— useEffect on mount:
       |    1. capture document.activeElement → cardFocusRef
       |    2. querySelector('.hub') → set .inert = true   ← NEW (A11Y-04)
       |    3. move focus → close button ref               ← NEW (A11Y-04)
       |
       |— onKeyDown on role="dialog" div:
       |    Escape → handleClose()                         ← WR-05 fix
       |
       |— Tab/Shift-Tab cycles WITHIN modal only
       |    (inert on .hub background prevents escape)     ← A11Y-04
       |
       |— useEffect cleanup on unmount:
       |    1. remove .inert from .hub                     ← NEW (A11Y-04)
       |    2. cardFocusRef.current?.focus()               ← already exists
       |
[HubModal unmount] → focus returns to originating card
```

### Recommended Project Structure

No structural changes. All edits are in-place modifications to existing files:

```
frontend/src/
├── style.css                          # 6 CSS additions (GAP-135-A, E, F)
└── components/Hub/
    ├── HubModal.tsx                   # WR-05 fix + A11Y-04 focus trap
    └── HubFilterBar.tsx               # aria-pressed on pills (GAP-135-B)
```

Test files (source-inspection pattern, already proven in HubModal.test.tsx):
```
frontend/src/components/Hub/
├── HubModal.test.tsx                  # Add A11Y-04 + WR-05 assertions
├── HubFilterBar.test.tsx              # Add aria-pressed assertion
└── SessionCard.test.tsx               # Add :focus-visible source assertion
```

---

### Pattern 1: `:focus-visible` upgrade

**What:** Replace `:focus` with `:focus-visible` on interactive elements so mouse clicks don't show outline rings, only keyboard focus does.

**When to use:** Every interactive Hub element that currently uses `:focus` for outline, plus all elements with no focus rule at all.

**The existing pattern** in `style.css` (appears at lines 415, 1670, 1823, 1898, 1982, 2061, 2726-2732, 2818, 2899, 2919, 2960, 3002, 3065, 3070, 3148, 3479-3481 — this is the project standard):

```css
/* Source: style.css line 1670 — established project pattern */
.remote-panel__btn:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}
```

**Applied to Hub elements (GAP-135-A):**

```css
/* Source: style.css line 4313 — EXISTING (upgrade from :focus to :focus-visible) */
.hub-card:focus-visible {
  outline: 2px solid var(--hub-accent);
  outline-offset: 2px;
}

/* ADD: filter pills, new-session button, open button, menu button, menu items,
   modal close/send/close-btn buttons, group sidebar toggle and items */
.hub-filter__pill:focus-visible,
.hub-filter__new-session:focus-visible,
.hub-card__open:focus-visible,
.hub-card__menu-btn:focus-visible,
.hub-card__menu-item:focus-visible,
.hub-modal__close:focus-visible,
.hub-modal__send-btn:focus-visible,
.hub-modal__close-btn:focus-visible,
.hub__group-sidebar-toggle:focus-visible,
.hub__group-sidebar-item:focus-visible {
  outline: 2px solid var(--hub-accent);
  outline-offset: 2px;
}
```

**IMPORTANT:** Remove (or change to `:focus-visible`) the existing `.hub-card:focus` rule at style.css line 4313. The `:focus` rule shows outline on mouse click — this is the anti-pattern. The `:focus-visible` selector only shows outline for keyboard navigation.

**Note on `.hub-filter__search` and `.hub-modal__respond-input`:** These are text `<input>` elements. For inputs, `:focus` is appropriate (WCAG 2.4.7 requires visible focus for ALL focus, including mouse, on form controls — `:focus-visible` alone is insufficient for inputs). Do NOT downgrade these to `:focus-visible`. They are correct as-is.

---

### Pattern 2: `inert` attribute for modal focus trap (A11Y-04)

**What:** When HubModal mounts, mark the Hub background as `inert` to prevent Tab focus from reaching background cards. When HubModal unmounts, remove `inert`.

**Why `inert` over a Tab-cycle interceptor:**
- `inert` also suppresses screen-reader announcement of background content (the correct WCAG 2.1 behavior for `aria-modal="true"`)
- No keyboard event interception or focusable-element query needed
- One attribute set/unset vs. complex event handling

**Browser support:** [VERIFIED: MDN Web Docs caniuse data — confirmed via WebSearch] `inert` is supported in:
- Chrome/Chromium 102+ (2022) — Wails Windows WebView2 uses Chromium, well above 102
- Safari 15.5+ (2022) — Wails macOS uses WKWebView (WebKit); macOS Monterey 12.4+ ships Safari 15.5+
- Firefox 112+ (2023)

The `wailsapp/go-webview2 v1.0.22` on Windows pins to Chromium 126+. `inert` is available. [VERIFIED: codebase scan shows go-webview2 v1.0.22]

**jsdom limitation:** jsdom 29 does NOT implement the `inert` focus-suppression behavior. Setting `element.inert = true` returns `undefined` (confirmed by testing against the installed jsdom 29.0.0). This means **A11Y-04 tests must use source-inspection assertions**, not DOM focus-behavior assertions. See Validation Architecture section.

**Implementation in HubModal.tsx:**

```typescript
// Source: New useEffect in HubModal.tsx — A11Y-04 focus trap via inert
const closeBtnRef = useRef<HTMLButtonElement>(null)

useEffect(() => {
  if (phase !== 'open') return          // Only trap when fully open
  
  // Apply inert to Hub background (all siblings of the modal overlay)
  const hubEl = document.querySelector('.hub') as HTMLElement | null
  if (hubEl) hubEl.inert = true
  
  // Move initial focus to close button
  closeBtnRef.current?.focus()
  
  return () => {
    // Remove inert on unmount (cleanup runs before focus-return in cardFocusRef cleanup)
    if (hubEl) hubEl.inert = false
  }
}, [phase])
```

**Add `ref={closeBtnRef}` to the close button in HubModal.tsx:**

```tsx
// Source: HubModal.tsx header strip — add ref
<button
  ref={closeBtnRef}
  type="button"
  className="hub-modal__close"
  aria-label="Close modal"
  onClick={handleClose}
>
```

**Why `phase === 'open'` guard:** During `entering` phase the modal is animating but focus should not yet be trapped (the animation must complete first, matching the existing phase machine contract). Focus trap activates when `phase` transitions to `'open'`.

---

### Pattern 3: WR-05 — Escape scope fix

**What:** Replace the document-level Escape listener (with `stopImmediatePropagation`) with a `onKeyDown` handler on the `role="dialog"` div.

**Why:** `stopImmediatePropagation` on `document` is globally suppressive — it prevents ALL other document-level Escape handlers from firing while the modal is open. The scoped fix is `e.stopPropagation()` on the dialog element's `onKeyDown`, which only prevents the event from bubbling further up the DOM tree from the dialog.

**The real problem it was solving (Pitfall 6):** When the modal is open and user presses Escape, BOTH the card's menu Escape handler AND the modal's Escape handler would fire. The solution is: the dialog overlay's click handler already provides click-outside dismissal; the dialog's own `onKeyDown` fires before any parent would, so `e.stopPropagation()` is sufficient to prevent the Hub card's document listener from also handling it.

```tsx
// Source: HubModal.tsx — REPLACE the document.addEventListener useEffect with this
<div
  role="dialog"
  aria-modal="true"
  aria-label={ariaLabel}
  className={`hub-modal hub-modal--${isBriefing ? 'briefing' : 'interactive'} hub-modal--${phase}`}
  style={{ transformOrigin }}
  onClick={(e) => e.stopPropagation()}
  onKeyDown={(e) => {
    if (e.key === 'Escape') {
      e.stopPropagation()   // Prevent parent hub card menu Escape from also firing
      handleClose()
    }
  }}
  onAnimationEnd={() => {
    if (phase === 'entering') setPhase('open')
    if (phase === 'exiting') onClose()
  }}
>
```

**Remove** the `document.addEventListener('keydown', handleKeyDown)` useEffect (lines 128–139 in current HubModal.tsx). Also remove the `handleCloseRef` pattern — it was only needed to keep the document listener stable. With `onKeyDown` on the dialog element, `handleClose` can be referenced directly.

**Interaction with card menu Escape:** The card's `SessionCard.tsx` menu Escape handler is registered on `document` (not on the card element). With the modal overlay having `z-index: 200` and being a separate stacking context, keyboard events dispatched from within the modal dialog will bubble: dialog → overlay div → document. The dialog's `onKeyDown` with `e.stopPropagation()` stops the bubble at the overlay level, before it reaches the card's document listener.

---

### Pattern 4: `aria-pressed` on filter pills

**What:** Add `aria-pressed="true/false"` to each filter pill button so screen readers announce which filter is active.

**Why `aria-pressed` (not `aria-selected`):** The pills are `<button>` elements inside a `role="group"`, not a `role="listbox"`. `aria-pressed` is the correct ARIA attribute for toggle buttons. The existing `role="group"` with `aria-label="Session status filter"` on the container is already correct.

```tsx
// Source: HubFilterBar.tsx — add aria-pressed
<button
  key={key}
  className={`hub-filter__pill${activeFilter === key ? ' hub-filter__pill--active' : ''}`}
  onClick={() => onFilterChange(key)}
  type="button"
  aria-pressed={activeFilter === key ? 'true' : 'false'}
>
```

---

### Pattern 5: `prefers-reduced-motion` CSS additions

**Existing pattern** (already in style.css — use as template):

```css
/* Source: style.css line 4950 — existing reduced-motion pattern for attention pulse */
@media (prefers-reduced-motion: reduce) {
  .hub-card--attention {
    border-color: var(--hub-attn-static-border);
    box-shadow: none;
    animation: none;
  }
}
```

**GAP-135-E — Spin animation (MISSING):**

```css
/* Source: style.css — ADD after line 4956 */
@media (prefers-reduced-motion: reduce) {
  .hub-card__status-icon--spin {
    animation: none;
  }
}
```

Note: The spin IS already gated correctly in the `no-preference` guard at line 4376. The `reduce` block is what's missing. The `running` state is distinguishable without spin: `ArrowPathIcon` shape is unique among the six status icons, and "Running" text label is present.

**GAP-135-F — Card hover transition (MISSING):**

```css
/* Source: style.css — ADD after the reduce block above */
@media (prefers-reduced-motion: reduce) {
  .hub-card {
    transition: none;
  }
}
```

Note: The `no-preference` guard at line 4932 already overrides the base transition to `400ms` for attention-clear animation. The `reduce` block must suppress both. Ordering matters: place the `reduce` block AFTER the `no-preference` block.

---

### Anti-Patterns to Avoid

- **Using `:focus` instead of `:focus-visible` on non-input interactive elements:** Causes focus rings on mouse click, which is visually noisy. All buttons/cards in Hub should use `:focus-visible`.
- **Using `aria-hidden="true"` on the background instead of `inert`:** `aria-hidden` only hides from the accessibility tree — it does NOT prevent keyboard focus. Cards with `tabIndex={0}` remain reachable by Tab even when `aria-hidden`. Always use `inert`.
- **`stopImmediatePropagation` on document-level listeners:** Suppresses ALL subsequent document Escape handlers globally, not just the intended ones. Breaks other features that register Escape (search clear, menu close). Use `stopPropagation()` on a scoped element instead.
- **Placing `inert` on the modal overlay:** `inert` must go on the BACKGROUND (the `.hub` element), not the overlay. The modal itself must remain interactable.
- **Forgetting to remove `inert` on modal close:** An `inert` background left in place after the modal unmounts traps the user — keyboard navigation stops working for the entire Hub. Always pair `inert = true` with a cleanup that sets `inert = false`.
- **Relying on jsdom to test `inert` behavior:** jsdom 29 does not implement `inert` focus suppression. Use `?raw` source inspection tests instead.
- **Adding `:focus-visible` to `<input>` elements:** Inputs need `:focus` (not `:focus-visible`) so they always show a visible focus indicator per WCAG 2.4.7. The existing `.hub-filter__search:focus` and `.hub-modal__respond-input:focus` rules are correct and should not be changed.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Focus trap Tab-cycle interceptor | Custom `querySelectorAll` + Tab event interception | DOM `inert` attribute | `inert` is native, handles AT announcement suppression too, no focusable-element query needed |
| Colorblind-safe icon library | Custom SVG shapes | `@heroicons/react/24/outline` (already in project) | Already in use; each icon is a distinct shape; no new dependency |
| Motion detection | Custom media query hook | CSS `@media (prefers-reduced-motion)` + `window.matchMedia` already in HubModal.tsx | Pattern already proven in this codebase (lines 92–99 of HubModal.tsx) |

**Key insight:** This phase is entirely achievable with the browser's native accessibility primitives (`inert`, `:focus-visible`, `prefers-reduced-motion` media queries, `aria-pressed`). No third-party accessibility library (focus-trap, @radix-ui, etc.) is needed or appropriate.

---

## Common Pitfalls

### Pitfall 1: Forgetting `inert` cleanup causes full Hub keyboard lock
**What goes wrong:** If `inert` is applied to `.hub` but not removed on modal close, Tab navigation stops working for the entire Hub. Every card, filter pill, and search box becomes unreachable.
**Why it happens:** React `useEffect` cleanup only runs if the effect ran. If the `phase` guard (`if (phase !== 'open') return`) prevents `inert` from being set, the cleanup also doesn't remove it — which is correct. But if the effect runs and sets `inert`, the cleanup MUST run.
**How to avoid:** Pair every `hubEl.inert = true` with a captured ref: `const hubEl = document.querySelector('.hub') as HTMLElement | null`. Return a cleanup that calls `if (hubEl) hubEl.inert = false`. The cleanup runs unconditionally if the effect body ran past the early return.
**Warning signs:** After closing modal, Tab key does nothing in Hub. Fixed by `inert = false` in cleanup.

### Pitfall 2: `:focus-visible` interacts with `:focus` specificity — wrong rule wins
**What goes wrong:** After adding `:focus-visible` rules, mouse clicks STILL show the ring because the old `:focus` rule remains and has equal specificity. Since `:focus` matches during keyboard focus too, the `:focus` ring also fires and the visual result looks correct during keyboard testing — but fails on mouse click.
**Why it happens:** Both `:focus` and `:focus-visible` match during keyboard navigation. If both rules exist with identical declarations, the browser renders both (same effect). The bug is only visible on mouse click.
**How to avoid:** When upgrading, REPLACE `.hub-card:focus` with `.hub-card:focus-visible` — do not ADD `:focus-visible` alongside the existing `:focus` rule. The `:focus` rule at line 4313 must be removed or changed in place.
**Warning signs:** Mouse click on a card shows the accent outline ring.

### Pitfall 3: `inert` applied during `entering` phase breaks animation
**What goes wrong:** If `inert` is applied when `phase === 'entering'`, the modal is animating. The close button receives focus immediately, which can interfere with the animation playback in some browser/OS combinations.
**Why it happens:** Focus events can cause forced layout recalculation, which in rare cases interrupts CSS animation.
**How to avoid:** Gate the `inert`+focus effect on `phase === 'open'` (after the `onAnimationEnd` → `setPhase('open')` transition). The `useEffect` with `[phase]` dependency will re-run when `phase` changes to `'open'`.
**Warning signs:** Grow animation stutters on modal open; observed only in specific OS animation quality settings.

### Pitfall 4: WR-05 fix breaks the existing stopImmediatePropagation test
**What goes wrong:** `HubModal.test.tsx` line 28 asserts `expect(raw).toContain('stopImmediatePropagation')`. After the WR-05 fix removes `stopImmediatePropagation` and replaces it with `stopPropagation`, this test goes RED.
**Why it happens:** The test was written to document the pre-fix behavior (Pitfall 6 guard in Phase 134) and will need to be updated in Phase 135.
**How to avoid:** When removing `stopImmediatePropagation`, also update `HubModal.test.tsx`: change the assertion to `expect(raw).toContain('stopPropagation')` and `expect(raw).not.toContain('stopImmediatePropagation')`.
**Warning signs:** `HubModal.test.tsx` test "MODAL-02: calls stopImmediatePropagation" fails RED after WR-05 fix.

### Pitfall 5: `inert` on `.hub` selector may miss if component root has a different class
**What goes wrong:** `document.querySelector('.hub')` returns null if the HubPanel root element uses a different class name, leaving background cards fully focusable.
**Why it happens:** Source code may have diverged from assumption.
**How to avoid:** Verify the HubPanel root element's class before implementation. From codebase scan: `HubPanel.tsx` renders a root div with `className="hub"` (confirmed by grep). The selector is correct.
**Warning signs:** Modal open, but Tab still reaches background cards. Debug by checking `document.querySelector('.hub')` in browser console.

### Pitfall 6: jsdom `inert` test false-positive
**What goes wrong:** A DOM-based test asserts `document.querySelector('.hub').inert === true` after modal mounts, and gets `undefined` instead of `true` — test goes RED even though the production behavior is correct.
**Why it happens:** jsdom 29 does not implement the `inert` property on HTMLElement. `element.inert = true` silently does nothing; `element.inert` is `undefined`.
**How to avoid:** Use `?raw` source inspection tests that assert the source code calls `hubEl.inert = true` (string match), rather than DOM behavior tests. This is the established pattern in this repo (see HubModal.test.tsx GAP-134-D tests).
**Warning signs:** Test asserts `element.inert === true` → always fails in jsdom environment.

---

## Code Examples

### A11Y-04: Complete focus trap implementation for HubModal.tsx

```typescript
// Source: verified HubModal.tsx structure — new additions for A11Y-04

// Add ref alongside existing cardFocusRef
const closeBtnRef = useRef<HTMLButtonElement>(null)

// Add new useEffect (runs after phase changes — dep is [phase])
useEffect(() => {
  if (phase !== 'open') return

  const hubEl = document.querySelector('.hub') as HTMLElement | null
  if (hubEl) hubEl.inert = true

  // Move initial focus to close button (WCAG 2.4.3: focus order)
  closeBtnRef.current?.focus()

  return () => {
    if (hubEl) hubEl.inert = false
    // Note: cardFocusRef focus-return runs from its OWN useEffect cleanup
    // — do NOT call cardFocusRef.current?.focus() here, it would fire before inert is removed
  }
}, [phase])

// In JSX — add ref to close button:
<button
  ref={closeBtnRef}
  type="button"
  className="hub-modal__close"
  aria-label="Close modal"
  onClick={handleClose}
>
  <XMarkIcon aria-hidden="true" />
</button>

// In JSX — add onKeyDown to role="dialog" div (WR-05):
<div
  role="dialog"
  aria-modal="true"
  aria-label={ariaLabel}
  className={`hub-modal hub-modal--${...} hub-modal--${phase}`}
  style={{ transformOrigin }}
  onClick={(e) => e.stopPropagation()}
  onKeyDown={(e) => {
    if (e.key === 'Escape') {
      e.stopPropagation()
      handleClose()
    }
  }}
  onAnimationEnd={...}
>
```

### Source-inspection test for A11Y-04 (jsdom-safe pattern)

```typescript
// Source: established pattern in HubModal.test.tsx (lines 8, 37-48)
import raw from './HubModal.tsx?raw'

describe('HubModal (A11Y-04: focus trap via inert)', () => {
  it('sets hubEl.inert = true when phase is open', () => {
    expect(raw).toContain('.inert = true')
  })

  it('removes hubEl.inert on cleanup', () => {
    expect(raw).toContain('.inert = false')
  })

  it('moves focus to closeBtnRef on open', () => {
    expect(raw).toContain('closeBtnRef')
    expect(raw).toContain('closeBtnRef.current?.focus()')
  })
})

describe('HubModal (WR-05: scoped Escape handler)', () => {
  it('does NOT use stopImmediatePropagation', () => {
    expect(raw).not.toContain('stopImmediatePropagation')
  })

  it('uses stopPropagation on Escape (scoped, not global)', () => {
    expect(raw).toContain('stopPropagation')
  })

  it('Escape is handled via onKeyDown on dialog element, not document.addEventListener', () => {
    // After fix: no document.addEventListener for Escape in HubModal
    // The check: no document.addEventListener with 'keydown' for Escape
    expect(raw).not.toMatch(/document\.addEventListener\s*\(\s*['"]keydown/)
  })
})
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `aria-hidden` on modal background | DOM `inert` attribute | WHATWG spec ratified ~2022, all major browsers 2022-2023 | `inert` also suppresses focus AND AT announcement; `aria-hidden` only suppresses AT |
| Manual Tab-trap interceptor | `inert` attribute | Same | Eliminates complex focusable-element query + keyboard event interception |
| `:focus` for all focus rings | `:focus-visible` for interactive, `:focus` for inputs | CSS Selectors Level 4, ~2021 | Mouse users don't see distracting rings; keyboard users still see rings |
| `stopImmediatePropagation` for modal Escape | `stopPropagation` on scoped `onKeyDown` | Established ARIA best practice | Prevents global side-effect of blocking other Escape handlers |

**Deprecated/outdated in this phase:**
- `.hub-card:focus` (line 4313): Replace with `:focus-visible` — `:focus` alone is the anti-pattern for non-input interactive elements
- `stopImmediatePropagation` in HubModal.tsx Escape handler: Replace with `stopPropagation` on dialog `onKeyDown`
- `document.addEventListener('keydown', ...)` in HubModal.tsx for Escape: Superseded by dialog `onKeyDown`

---

## A11Y-01 Verification Report (source-level audit)

The UI-SPEC and REQUIREMENTS.md both state A11Y-01 is "Already Complete" from Phase 131-133. This research verifies that claim at source level.

**STATUS_CONFIG audit** (SessionCard.tsx lines 29–51, verified by direct file read):

| HubStatus | Icon | Icon is unique shape? | Text label | Non-color differentiator |
|-----------|------|-----------------------|------------|--------------------------|
| `running` | `ArrowPathIcon` | YES — circular arrow, unique | "Running" | Icon shape + spin motion + text |
| `idle` | `CheckCircleIcon` | YES — circle with checkmark | "Idle" | Icon shape + text |
| `waiting` | `PauseCircleIcon` | YES — circle with pause bars | "Needs input" | Icon shape + text |
| `errored` | `ExclamationCircleIcon` | YES — circle with exclamation | "Error" | Icon shape + text |
| `stopped-ok` | `StopCircleIcon` | YES — circle with filled square | "Done" | Icon shape + text |
| `stopped-err` | `ExclamationCircleIcon` | SHARED with `errored` | "Exited {code}" | **Text label IS the differentiator** |

**`stopped-err` vs `errored` sharing `ExclamationCircleIcon`:** Both use the same icon shape. The differentiator is the text label: `errored` shows "Error", `stopped-err` shows "Exited {code}" where `{code}` is the numeric exit code. This is **compliant**: text label is a valid non-color differentiator per WCAG 1.4.1.

**STATUS_CONFIG mirrors** (HubModal.tsx lines 26–35, verified): STATUS_CONFIG in HubModal.tsx is identical to SessionCard.tsx. Both define the same 6 statuses with the same icons and labels. Compliant.

**`running` without spin** (reduced-motion): `ArrowPathIcon` is still the unique icon shape for `running` even when spin is disabled. No additional cue required. Compliant.

**A11Y-01 verdict: PASS — verification only, no code changes required.**

---

## Runtime State Inventory

> Not applicable — this is a hardening/CSS/props phase, not a rename/refactor/migration. No stored data, live service config, OS-registered state, secrets, or build artifacts are affected.

---

## Open Questions (RESOLVED)

1. **GroupSidebar `<li>` items have no `tabIndex`**
   - What we know: `GroupSidebarItem` renders an `<li>` with `role="option"` and `onClick` but no `tabIndex`. `<li>` elements are not natively focusable.
   - What's unclear: Whether keyboard navigation to group sidebar items was intended in A11Y-02 scope. The UI-SPEC §2e lists `.hub__group-sidebar-item:focus-visible` as a required rule — which implies the items must become keyboard-focusable.
   - Recommendation: Add `tabIndex={0}` and an `onKeyDown` (Enter/Space → `onGroupSelect(id)`) to `GroupSidebarItem` in `GroupSidebar.tsx`. This is a small addition consistent with the SessionCard keyboard pattern. Include in Phase 135 scope alongside the CSS rule.

2. **`inert` and HubBriefingModal / HubInteractiveModal focus ordering**
   - What we know: Initial focus goes to `hub-modal__close` (the close button). For briefing modals, the respond input (`.hub-modal__respond-input`) is the primary action target. For interactive terminals (TerminalPanel), the terminal canvas is the primary target.
   - What's unclear: Whether moving initial focus to the close button is optimal UX for briefing modals (user may want to immediately type a response).
   - Recommendation: For briefing modals (`isBriefing === true`), move initial focus to the respond input instead of the close button. For interactive modals, focus the close button (the terminal canvas is not a standard focusable element). This requires a separate `respondInputRef` passed down to `HubBriefingModal`, or a conditional in HubModal. Either approach works. Keep it simple: focus close button for all cases in Wave 0, optimize in a follow-up if UX feedback warrants.

3. **Cross-surface parity check (release-blocking)**
   - What we know: TUI Hub parity is explicitly deferred to issue #82 with signed-off user approval. CLI `agenthub list` already has `waiting`/`errored` status column. The a11y hardening (focus rings, inert, reduced-motion) is desktop-GUI-specific and has no TUI/CLI parity implication.
   - What's unclear: None — the deferral is documented and signed off. No new gap created by this phase.
   - Recommendation: No action needed. The existing deferral covers this.

---

## Environment Availability

> Step 2.6: This phase is code/config changes only (CSS + React props + TypeScript). No external service dependencies beyond the existing dev toolchain.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Vitest | All tests | Yes | 4.1.0 | — |
| jsdom | Test DOM environment | Yes | 29.0.0 | — |
| `pnpm test` (in `frontend/`) | Test execution | Yes | via `pnpm` | `npm test` |

**Missing dependencies with no fallback:** none

**Missing dependencies with fallback:** none

**jsdom `inert` limitation:** jsdom 29 does not implement `inert` focus suppression. This is a test environment constraint, not a production constraint. Use source-inspection (`?raw`) tests for A11Y-04. [VERIFIED: confirmed by running `node -e "const {JSDOM}=require('./node_modules/jsdom'); const dom=new JSDOM(...); el.inert=true; console.log(el.inert)"` → `undefined`]

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (test key) |
| Quick run command | `cd frontend && pnpm test -- --reporter=verbose HubModal HubFilterBar SessionCard` |
| Full suite command | `cd frontend && pnpm test` |
| Test environment | jsdom 29.0.0 |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| A11Y-01 | STATUS_CONFIG: every status has unique icon + text label | source-inspection | `pnpm test -- SessionCard` | Yes (existing tests cover icon+label per status) |
| A11Y-01 | STATUS_CONFIG mirrors between SessionCard.tsx and HubModal.tsx | source-inspection | `pnpm test -- HubModal` | Partial — new mirror assertion needed |
| A11Y-02 | `:focus-visible` rules present for all Hub interactive elements | source-inspection on style.css | `pnpm test -- --reporter=verbose` (grep in test) | No — Wave 0 gap |
| A11Y-02 | Filter pills have `aria-pressed` attribute | DOM render test | `pnpm test -- HubFilterBar` | No — Wave 0 gap |
| A11Y-02 | WR-05: modal uses `onKeyDown` on dialog, not `document.addEventListener` | source-inspection | `pnpm test -- HubModal` | No — Wave 0 gap (replaces existing `stopImmediatePropagation` test) |
| A11Y-03 | Spin animation not-gate present in style.css | source-inspection | `pnpm test -- --reporter=verbose` | No — Wave 0 gap |
| A11Y-03 | Card hover transition suppressed in reduce block | source-inspection | `pnpm test -- --reporter=verbose` | No — Wave 0 gap |
| A11Y-04 | `inert` set/unset on `.hub` element during modal open/close | source-inspection (`?raw`) | `pnpm test -- HubModal` | No — Wave 0 gap |
| A11Y-04 | Initial focus moves to close button on modal open | source-inspection (`?raw`) | `pnpm test -- HubModal` | No — Wave 0 gap |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm test -- HubModal HubFilterBar SessionCard`
- **Per wave merge:** `cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `HubModal.test.tsx` — Add A11Y-04 source-inspection tests (inert set/unset, closeBtnRef, WR-05 Escape via onKeyDown)
- [ ] `HubModal.test.tsx` — Update WR-05 test: remove `stopImmediatePropagation` assertion, add `stopPropagation` + `not.toMatch(/document\.addEventListener.*keydown/)` assertions
- [ ] `HubFilterBar.test.tsx` — Add `aria-pressed` assertion (active pill has `aria-pressed="true"`, inactive has `aria-pressed="false"`)
- [ ] `style.css` source-inspection tests (new test file `style.css.test.ts`) — OR inline grep assertions in existing tests — for: `.hub-card:focus-visible`, `.hub-card__status-icon--spin` + `prefers-reduced-motion: reduce`, `.hub-card` + `reduce` + `transition: none`

*(The `?raw` source-inspection pattern for style.css is: `import raw from '../../../style.css?raw'` — Vite supports `?raw` for any file type)*

---

## Security Domain

> This phase has no authentication, session management, access control, cryptography, or network-facing changes. All changes are to CSS and React JSX props on an already-rendered surface.

**ASVS categories applicable:** None — this is a pure UI/CSS accessibility hardening pass with no security surface area.

---

## Project Constraints (from CLAUDE.md)

| Directive | Impact on Phase 135 |
|-----------|---------------------|
| `pnpm` preferred package manager | Use `pnpm test` for all test runs |
| TypeScript types required | `closeBtnRef = useRef<HTMLButtonElement>(null)` — typed ref |
| ESLint + Prettier | No new lint issues; `inert` is a standard DOM property |
| NEVER install packages globally | Not applicable — no new packages |
| Python virtual env rule | Not applicable — frontend-only phase |
| Colorblind-safe verification at source | STATUS_CONFIG hex constants are the authoritative record; verify by code read, not by eye |
| Cross-surface parity release-blocking | TUI Hub deferral is signed off (#82); no new gap from this phase |
| No Tailwind | Confirmed — style.css uses raw CSS custom properties; no utility classes |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `document.querySelector('.hub')` returns the HubPanel root element | A11Y-04 Pattern 2 | If the Hub root class changed, `inert` would not be applied to background; Tab would still reach cards. Planner must verify by reading HubPanel.tsx root element class before implementing. |
| A2 | Wails macOS webview (WKWebView) supports `inert` (via Safari 15.5+ in macOS Monterey+) | A11Y-04 Pattern 2 | If a user is on macOS Monterey <12.4 (Safari <15.5), `inert` may not work. Fallback is the `useFocusTrap` Tab-cycle interceptor described in the UI-SPEC. Low risk: Monterey 12.4 shipped May 2022. |

**If this table is empty after implementation:** All claims verified against source — no user confirmation needed. The two assumptions above are low-risk and can be confirmed by a single grep of `HubPanel.tsx` root element (A1) and by the production macOS target specification (A2).

---

## Sources

### Primary (HIGH confidence)
- `frontend/src/components/Hub/SessionCard.tsx` — STATUS_CONFIG, tabIndex=0, onKeyDown Enter/Space, aria-label pattern
- `frontend/src/components/Hub/HubModal.tsx` — WR-05/WR-06 deferred comments, cardFocusRef, phase machine, prefersReducedMotion detection
- `frontend/src/components/Hub/HubFilterBar.tsx` — filter pill buttons (no aria-pressed), role="group"
- `frontend/src/components/Hub/GroupSidebar.tsx` — li[role="option"] without tabIndex
- `frontend/src/style.css` lines 4097–5387 — hub-* custom properties, :focus rules, prefers-reduced-motion guards
- `.planning/phases/135-accessibility-hardening/135-UI-SPEC.md` — 6 gaps, contracts per gap
- `.planning/REQUIREMENTS.md` — A11Y-01..04 definitions

### Secondary (MEDIUM confidence)
- `frontend/src/components/Hub/HubModal.test.tsx` — `?raw` source inspection pattern precedent (established in this codebase)
- `frontend/src/components/LinkConfirmPopover.tsx` — focus-on-mount pattern for Cancel button; Escape handler pattern
- `frontend/vite.config.ts` — jsdom 29, vitest 4.1 test environment configuration

### Tertiary (LOW confidence — training data only)
- `inert` browser support data: [ASSUMED from training — "Chrome 102+, Safari 15.5+, Firefox 112+"]. Not verified via live caniuse lookup. The go-webview2 v1.0.22 confirms Chromium is recent (126+), so Windows support is confirmed. macOS/Safari support is the assumption.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; all existing tooling confirmed by direct package.json and file reads
- Architecture: HIGH — all 6 gaps are directly observed in source; implementation patterns are source-verified
- Pitfalls: HIGH — pitfalls 1-4 derived from direct code analysis; pitfall 5 verified by HubPanel.tsx grep; pitfall 6 verified by jsdom test
- A11Y-01 verification: HIGH — STATUS_CONFIG read directly from source
- `inert` browser support: MEDIUM — go-webview2 version confirmed (Chromium 126+); macOS WebKit version is assumed

**Research date:** 2026-06-18
**Valid until:** 2026-07-18 (stable web standards; only goes stale if Wails downgrades WebView2 to pre-Chromium-102)
