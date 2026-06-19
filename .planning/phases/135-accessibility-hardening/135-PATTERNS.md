# Phase 135: Accessibility Hardening - Pattern Map

**Mapped:** 2026-06-18
**Files analyzed:** 6 files (3 modified source, 3 modified test)
**Analogs found:** 6 / 6

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/style.css` | config/style | transform | `frontend/src/style.css` (existing hub + prefers-reduced-motion blocks) | self-analog (additions to existing file) |
| `frontend/src/components/Hub/HubModal.tsx` | component | request-response (modal lifecycle) | `frontend/src/components/Hub/HubModal.tsx` (self) + `frontend/src/components/LinkConfirmPopover.tsx` (focus-on-mount) | self-analog (targeted additions) |
| `frontend/src/components/Hub/HubFilterBar.tsx` | component | request-response | `frontend/src/components/Hub/HubFilterBar.tsx` (self) | self-analog (one-prop addition) |
| `frontend/src/components/Hub/GroupSidebar.tsx` | component | event-driven | `frontend/src/components/Hub/SessionCard.tsx` (keyboard `onKeyDown` on non-button) | role-match |
| `frontend/src/components/Hub/HubModal.test.tsx` | test | — | `frontend/src/components/Hub/HubModal.test.tsx` (self, existing `?raw` pattern) | exact |
| `frontend/src/components/Hub/HubFilterBar.test.tsx` | test | — | `frontend/src/components/Hub/HubFilterBar.test.tsx` (self, existing DOM-render pattern) | exact |

---

## Pattern Assignments

### `frontend/src/style.css` — GAP-135-A, GAP-135-E, GAP-135-F

**Analog:** Self — existing `:focus-visible` and `prefers-reduced-motion` blocks within the same file.

#### Focus-visible project pattern (lines 1670-1673 — the established standard across the entire codebase)

```css
/* Source: style.css line 1670 — remote-panel reference implementation */
.remote-panel__btn:focus-visible {
  outline: 2px solid #7aa2f7;
  outline-offset: 2px;
}
```

The Hub surface uses the custom-property alias `var(--hub-accent)` instead of the raw hex. Both are equivalent (`--hub-accent` resolves to `#7aa2f7` dark / `#3d6fe8` light). Use `var(--hub-accent)` in all new Hub rules to stay token-consistent.

Multi-selector grouping pattern (lines 2726-2735 — used when many elements share one focus rule):

```css
/* Source: style.css lines 2726-2735 */
.find-bar__btn--prev:focus-visible,
.find-bar__btn--next:focus-visible,
.find-bar__toggle--case:focus-visible,
.find-bar__toggle--regex:focus-visible,
.find-bar__toggle--word:focus-visible,
.find-bar__close:focus-visible,
.find-bar__input:focus-visible {
  outline: 2px solid #7aa2f7;
  outline-offset: 2px;
}
```

#### Existing `.hub-card:focus` rule to REPLACE (line 4313-4316)

```css
/* Source: style.css lines 4313-4316 — THIS rule must be changed to :focus-visible */
.hub-card:focus {
  outline: 2px solid var(--hub-accent);
  outline-offset: 2px;
}
```

REPLACE `.hub-card:focus` with `.hub-card:focus-visible`. Do NOT add `:focus-visible` alongside the existing `:focus` rule — remove/change it in place (Pitfall 2 in RESEARCH.md: equal-specificity rules mean the old `:focus` ring still fires on mouse click if both rules coexist).

#### Existing `prefers-reduced-motion` pair to use as template (lines 4927-4956)

```css
/* Source: style.css lines 4927-4956 — the attention-pulse no-preference / reduce pair */
/* no-preference block: */
@media (prefers-reduced-motion: no-preference) {
  .hub-card--attention {
    animation: hub-attn-pulse 2s ease-in-out infinite;
  }
  /* ATTN-03: attention clear — override 100ms base transition for motion-enabled users only */
  .hub-card {
    transition: border-color 400ms ease, box-shadow 400ms ease, background 100ms ease;
  }
  .hub-card--attention:hover {
    border-color: var(--hub-attn-border);
  }
}

/* reduce block: */
@media (prefers-reduced-motion: reduce) {
  .hub-card--attention {
    border-color: var(--hub-attn-static-border);
    box-shadow: none;
    animation: none;
  }
}
```

#### GAP-135-E: spin animation reduce block — ADD after line 4956

Copy the `reduce` block pattern above. The `no-preference` guard for the spin is already at lines 4376-4380:

```css
/* Source: style.css lines 4376-4380 — EXISTING no-preference guard (already correct) */
@media (prefers-reduced-motion: no-preference) {
  .hub-card__status-icon--spin {
    animation: hub-spin 0.8s linear infinite;
  }
}
```

New `reduce` block to add after line 4956:

```css
@media (prefers-reduced-motion: reduce) {
  .hub-card__status-icon--spin {
    animation: none;
  }
}
```

#### GAP-135-F: card hover transition reduce block — ADD after the spin reduce block

```css
@media (prefers-reduced-motion: reduce) {
  .hub-card {
    transition: none;
  }
}
```

Ordering matters: the `no-preference` override at lines 4932-4934 sets `.hub-card` transition to `400ms`. The `reduce` block must come AFTER the `no-preference` block so the cascade resolves correctly.

---

### `frontend/src/components/Hub/HubModal.tsx` — GAP-135-C (WR-05) + GAP-135-D (A11Y-04)

**Analog:** Self. The existing file has all the patterns to build on; the changes are targeted additions and one substitution.

#### Imports pattern (lines 1-19 — copy exactly, add nothing)

```typescript
/* Source: HubModal.tsx lines 1-19 */
import React, { useCallback, useEffect, useRef, useState } from 'react'
// ... (no new imports needed for this phase)
```

`closeBtnRef` requires no new import — `useRef` is already imported.

#### Existing `cardFocusRef` useEffect to use as template (lines 111-117)

```typescript
/* Source: HubModal.tsx lines 111-117 — focus-return on unmount pattern */
const cardFocusRef = useRef<HTMLElement | null>(null)
useEffect(() => {
  cardFocusRef.current = document.activeElement as HTMLElement
  return () => {
    cardFocusRef.current?.focus()
  }
}, [])
```

The new `closeBtnRef` useEffect for A11Y-04 follows this same `useEffect` + cleanup structure. Add it AFTER the `cardFocusRef` block, keyed on `[phase]`:

```typescript
/* Source: New addition — A11Y-04 focus trap via inert */
const closeBtnRef = useRef<HTMLButtonElement>(null)

useEffect(() => {
  if (phase !== 'open') return

  const hubEl = document.querySelector('.hub') as HTMLElement | null
  if (hubEl) hubEl.inert = true

  closeBtnRef.current?.focus()

  return () => {
    if (hubEl) hubEl.inert = false
  }
}, [phase])
```

#### Existing Escape useEffect to REMOVE (lines 119-139)

```typescript
/* Source: HubModal.tsx lines 119-139 — REMOVE THIS ENTIRE BLOCK */
// ---- Escape key handler (MODAL-02, T-134-04-02) ----
const handleCloseRef = useRef(handleClose)
useEffect(() => {
  handleCloseRef.current = handleClose
})
useEffect(() => {
  function handleKeyDown(e: KeyboardEvent): void {
    if (e.key === 'Escape') {
      e.stopImmediatePropagation()
      handleCloseRef.current()
    }
  }
  document.addEventListener('keydown', handleKeyDown)
  return () => document.removeEventListener('keydown', handleKeyDown)
}, [])
```

Remove `handleCloseRef` (both the ref declaration and the sync effect). Replace the `document.addEventListener` effect with the `onKeyDown` prop on the dialog div.

#### Existing `role="dialog"` div to MODIFY (lines 167-176)

```tsx
/* Source: HubModal.tsx lines 167-176 — CURRENT (before WR-05 fix) */
<div
  role="dialog"
  aria-modal="true"
  aria-label={ariaLabel}
  className={`hub-modal hub-modal--${isBriefing ? 'briefing' : 'interactive'} hub-modal--${phase}`}
  style={{ transformOrigin }}
  onClick={(e) => e.stopPropagation()}
  onAnimationEnd={() => {
    if (phase === 'entering') setPhase('open')
    if (phase === 'exiting') onClose()
  }}
>
```

Add `onKeyDown` prop (WR-05 fix). Keep all existing props unchanged:

```tsx
/* Target state after WR-05 fix */
<div
  role="dialog"
  aria-modal="true"
  aria-label={ariaLabel}
  className={`hub-modal hub-modal--${isBriefing ? 'briefing' : 'interactive'} hub-modal--${phase}`}
  style={{ transformOrigin }}
  onClick={(e) => e.stopPropagation()}
  onKeyDown={(e) => {
    if (e.key === 'Escape') {
      e.stopPropagation()
      handleClose()
    }
  }}
  onAnimationEnd={() => {
    if (phase === 'entering') setPhase('open')
    if (phase === 'exiting') onClose()
  }}
>
```

#### Existing close button to add `ref` prop to (lines 196-203)

```tsx
/* Source: HubModal.tsx lines 196-203 — CURRENT */
<button
  type="button"
  className="hub-modal__close"
  aria-label="Close modal"
  onClick={handleClose}
>
  <XMarkIcon aria-hidden="true" />
</button>
```

Add `ref={closeBtnRef}`:

```tsx
/* Target state — add ref */
<button
  ref={closeBtnRef}
  type="button"
  className="hub-modal__close"
  aria-label="Close modal"
  onClick={handleClose}
>
  <XMarkIcon aria-hidden="true" />
</button>
```

---

### `frontend/src/components/Hub/HubFilterBar.tsx` — GAP-135-B (`aria-pressed`)

**Analog:** Self. One `aria-pressed` prop addition to the pill `<button>`.

#### Existing pill button (lines 104-113 — CURRENT)

```tsx
/* Source: HubFilterBar.tsx lines 104-113 */
{FILTER_PILLS.map(({ key, label }) => (
  <button
    key={key}
    className={`hub-filter__pill${activeFilter === key ? ' hub-filter__pill--active' : ''}`}
    onClick={() => onFilterChange(key)}
    type="button"
  >
    {key === 'all' ? label : `${label} (${counts[key] ?? 0})`}
  </button>
))}
```

Add `aria-pressed`:

```tsx
/* Target state — add aria-pressed */
{FILTER_PILLS.map(({ key, label }) => (
  <button
    key={key}
    className={`hub-filter__pill${activeFilter === key ? ' hub-filter__pill--active' : ''}`}
    onClick={() => onFilterChange(key)}
    type="button"
    aria-pressed={activeFilter === key ? 'true' : 'false'}
  >
    {key === 'all' ? label : `${label} (${counts[key] ?? 0})`}
  </button>
))}
```

---

### `frontend/src/components/Hub/GroupSidebar.tsx` — A11Y-02 open question (keyboard for `<li>` items)

**Analog:** `frontend/src/components/Hub/SessionCard.tsx` — `tabIndex={0}` + `onKeyDown` Enter/Space on an `<article>` element (lines 250-257).

#### SessionCard keyboard pattern to copy (lines 250-257)

```tsx
/* Source: SessionCard.tsx lines 250-257 — keyboard-operable non-button element */
onKeyDown={(e) => {
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    onCardClick?.(session, (e.currentTarget as HTMLElement).getBoundingClientRect())
  }
}}
aria-label={cardAriaLabel}
tabIndex={0}
```

Apply to `GroupSidebarItem` `<li>` (lines 126-161 of GroupSidebar.tsx). The `<li>` already has `role="option"` and `onClick`. Add `tabIndex={0}` and `onKeyDown`:

```tsx
/* Target state for GroupSidebarItem <li> — add tabIndex and onKeyDown */
<li
  className={itemClass}
  role="option"
  aria-selected={isActive ? 'true' : 'false'}
  tabIndex={0}
  onClick={() => onGroupSelect(id)}
  onKeyDown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onGroupSelect(id)
    }
  }}
  onDragOver={handleDragOver}
  onDragLeave={handleDragLeave}
  onDrop={handleDrop}
>
```

---

### `frontend/src/components/Hub/HubModal.test.tsx` — A11Y-04 + WR-05 test updates

**Analog:** Self — the `?raw` source-inspection pattern is already established at lines 1-77.

#### Existing `?raw` import and describe/it pattern (lines 1-8, 9-35)

```typescript
/* Source: HubModal.test.tsx lines 1-8 — the ?raw import is the project standard for source-inspection tests */
import { describe, it, expect } from 'vitest'
import raw from './HubModal.tsx?raw'

describe('HubModal (MODAL-01: dialog accessibility contract)', () => {
  it('MODAL-01: uses role="dialog"', () => {
    expect(raw).toContain('role="dialog"')
  })
  // ...
})
```

#### Existing test that must be UPDATED (lines 28-30)

```typescript
/* Source: HubModal.test.tsx lines 28-30 — this assertion goes RED after WR-05 fix */
it('MODAL-02: calls stopImmediatePropagation (Pitfall 6 guard — prevents Hub card Escape double-fire)', () => {
  expect(raw).toContain('stopImmediatePropagation')
})
```

Replace with two assertions: one confirming the old approach is gone, one confirming the new scoped approach:

```typescript
/* Target — updated WR-05 assertions */
it('MODAL-02: uses stopPropagation (scoped, not global) for Escape on dialog element', () => {
  expect(raw).toContain('stopPropagation')
})

it('MODAL-02: does NOT use stopImmediatePropagation (WR-05 fix — scoped handler replaces global guard)', () => {
  expect(raw).not.toContain('stopImmediatePropagation')
})

it('MODAL-02: Escape handled via onKeyDown on dialog element, not document.addEventListener', () => {
  expect(raw).not.toMatch(/document\.addEventListener\s*\(\s*['"]keydown/)
})
```

#### New A11Y-04 describe block — source-inspection pattern following the exact style of existing describe blocks

```typescript
/* Pattern: add as a new describe block after existing describes */
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

  it('gates inert trap on phase === "open" (not during entering animation)', () => {
    expect(raw).toContain("phase !== 'open'")
  })
})
```

Note: NO DOM render tests for `inert` behavior. jsdom 29 does not implement `inert` focus suppression (confirmed: `element.inert` returns `undefined`). All A11Y-04 tests are `?raw` string assertions only.

---

### `frontend/src/components/Hub/HubFilterBar.test.tsx` — GAP-135-B `aria-pressed`

**Analog:** Self — the DOM-render test pattern is established at lines 1-307. See the `aria-label` test as the closest structural match (lines 164-168).

#### Existing DOM attribute assertion pattern (lines 164-168)

```typescript
/* Source: HubFilterBar.test.tsx lines 164-168 — DOM attribute assertion pattern */
it('renders the search input with correct aria-label', () => {
  ;({ container, root } = renderFilterBar())
  const input = container.querySelector('.hub-filter__search') as HTMLInputElement
  expect(input.getAttribute('aria-label')).toBe('Search sessions by name, CLI, or host')
})
```

Apply the same `.getAttribute()` pattern for `aria-pressed`. Add inside the existing `'HubFilterBar — filter pills'` describe block:

```typescript
/* Target — add to existing 'HubFilterBar — filter pills' describe block */
it('active pill has aria-pressed="true"', () => {
  ;({ container, root } = renderFilterBar({ activeFilter: 'running' }))
  const activePill = container.querySelector('.hub-filter__pill--active') as HTMLButtonElement
  expect(activePill.getAttribute('aria-pressed')).toBe('true')
})

it('inactive pills have aria-pressed="false"', () => {
  ;({ container, root } = renderFilterBar({ activeFilter: 'running' }))
  const allPills = Array.from(container.querySelectorAll('.hub-filter__pill'))
  const inactivePills = allPills.filter((p) => !p.classList.contains('hub-filter__pill--active'))
  for (const pill of inactivePills) {
    expect(pill.getAttribute('aria-pressed')).toBe('false')
  }
})
```

---

## Shared Patterns

### `?raw` Source-Inspection Test
**Source:** `frontend/src/components/Hub/HubModal.test.tsx` lines 7-8
**Apply to:** HubModal.test.tsx additions (A11Y-04), and any style.css source assertions.

```typescript
import raw from './HubModal.tsx?raw'
// For style.css source assertions (if needed in a new test file):
import cssRaw from '../../../style.css?raw'
```

Vite supports `?raw` for any file type. The pattern `expect(raw).toContain(...)` and `expect(raw).not.toContain(...)` and `expect(raw).not.toMatch(/regex/)` are all established in this repo. These are the ONLY valid test strategies for things jsdom cannot enforce natively (inert, CSS media queries).

### DOM-Render Test Helper Pattern
**Source:** `frontend/src/components/Hub/HubFilterBar.test.tsx` lines 29-49
**Apply to:** HubFilterBar.test.tsx additions (aria-pressed assertions use existing `renderFilterBar()` helper — no new helper needed).

```typescript
/* Render helper — already defined in HubFilterBar.test.tsx */
function renderFilterBar(
  overrides: Partial<React.ComponentProps<typeof HubFilterBar>> = {},
) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  // ...
  act(() => { root.render(<HubFilterBar {...props} />) })
  return { container, root, ...props }
}
```

### `useRef` + `useEffect` Cleanup Pattern
**Source:** `frontend/src/components/Hub/HubModal.tsx` lines 111-117
**Apply to:** HubModal.tsx — new `closeBtnRef` useEffect follows this exact structure (ref + effect + cleanup).

```typescript
const cardFocusRef = useRef<HTMLElement | null>(null)
useEffect(() => {
  cardFocusRef.current = document.activeElement as HTMLElement
  return () => {
    cardFocusRef.current?.focus()
  }
}, [])
```

### `onKeyDown` Enter/Space on Non-Button Interactive Elements
**Source:** `frontend/src/components/Hub/SessionCard.tsx` lines 250-257
**Apply to:** `GroupSidebarItem` `<li>` in GroupSidebar.tsx.

```tsx
onKeyDown={(e) => {
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    onCardClick?.(session, (e.currentTarget as HTMLElement).getBoundingClientRect())
  }
}}
tabIndex={0}
```

### `afterEach` Cleanup in DOM-Render Tests
**Source:** `frontend/src/components/Hub/HubFilterBar.test.tsx` lines 53-58
**Apply to:** Any new describe blocks that render components.

```typescript
afterEach(() => {
  root.unmount()
  container.remove()
})
```

---

## A11Y-01 Verification (No Code Changes — Source-Level Audit Only)

**Status: PASS — verified directly from source.**

Per RESEARCH.md §A11Y-01 Verification Report (confirmed by reading `SessionCard.tsx` lines 29-51 and `HubModal.tsx` lines 25-35):

- All 6 `HubStatus` values map to unique icon shape + text label in `STATUS_CONFIG`.
- `stopped-err` and `errored` share `ExclamationCircleIcon` — differentiated by text label ("Exited {code}" vs "Error"). This is WCAG-compliant: text label is a valid non-color differentiator.
- `running` uniqueness: `ArrowPathIcon` shape is distinct even without spin. Compliant under `prefers-reduced-motion`.
- Both `SessionCard.tsx` STATUS_CONFIG and `HubModal.tsx` STATUS_CONFIG are identical. No mirror drift.
- All color hex values in STATUS_CONFIG comments are labeled "(reinf.)" confirming color is reinforcement only.

**Executor action:** Run the existing SessionCard.test.tsx (icon + label assertions already exist). Add a source-inspection mirror assertion to HubModal.test.tsx if needed.

---

## Files With No New Analog Needed

All modified files have strong self-analogs or close role-match analogs within the Hub component tree. No file requires a pattern from outside the codebase.

| File | Reason |
|------|--------|
| `style.css` (GAP-135-A focus-visible) | Self-analog: identical pattern exists at lines 1670, 2726-2735, 3479-3481. No external analog needed. |
| `style.css` (GAP-135-E/F reduce blocks) | Self-analog: identical pattern exists at lines 4950-4956 (the attention reduce block). |
| `HubModal.tsx` (A11Y-04 inert) | Self-analog: `cardFocusRef` useEffect (lines 111-117) is the structural template. |
| `HubFilterBar.tsx` (aria-pressed) | Self-analog: `aria-label`, `type="button"` already on the same `<button>` element. |

---

## Metadata

**Analog search scope:** `frontend/src/components/Hub/`, `frontend/src/style.css`
**Files scanned:** 9 (HubModal.tsx, HubFilterBar.tsx, SessionCard.tsx, GroupSidebar.tsx, HubModal.test.tsx, HubFilterBar.test.tsx, GroupSidebar.test.tsx, SessionCard.test.tsx, style.css)
**Pattern extraction date:** 2026-06-18
