# Phase 141: Redesign Implementation — Pattern Map

**Mapped:** 2026-06-20
**Files analyzed:** 8 (1 primary CSS file, 3 TSX components, 3 test files, 1 implicit sidebar CSS scope)
**Analogs found:** 8 / 8

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `frontend/src/style.css` — sidebar block (lines 218–293) | CSS utility | transform | Hub group sidebar CSS (lines 4552–4760) — already fully tokenized | exact |
| `frontend/src/style.css` — S-01 welcome block (lines 1302–1404) | CSS utility | transform | Hub card CSS (lines 4107–4143) — fully tokenized surface | role-match |
| `frontend/src/style.css` — S-03 tab bar + status bar (lines 82–368) | CSS utility | transform | Hub card + hub group sidebar CSS — same token vocabulary | role-match |
| `frontend/src/style.css` — S-04/S-05 file browser + editor chrome (lines 2646–3380) | CSS utility | transform | Hub card + hub modal header/body CSS — same token vocabulary | role-match |
| `frontend/src/style.css` — S-06 settings panel (lines 370–771) | CSS utility | transform | Hub card + hub modal CSS — same token vocabulary | role-match |
| `frontend/src/style.css` — S-07 new `.hub-share-modal*` rules | CSS utility | request-response | `.hub-modal` / `.hub-modal__header` / `.hub-modal__body` (lines 4999–5100) | exact |
| `frontend/src/style.css` — motion guards on migrated surfaces | CSS utility | transform | `.find-bar` motion guard (lines 2440–2550); `.hub-modal` motion guard (lines 5219–5273) | exact |
| `frontend/src/style.css` — new tokens `:root` + `[data-ui-theme="light"]` | config | n/a | Existing `:root` + `[data-ui-theme="light"]` blocks (lines 3896–3995) | exact |
| `frontend/src/components/Hub/GroupSidebar.tsx` — CARRY-01 ARIA fix | component | event-driven | `.hub__group-sidebar-toggle` button pattern in GroupSidebar.tsx (lines 235–256) + `.hub__group-sidebar-item--active` CSS token usage | role-match |
| `frontend/src/components/Hub/SessionShareModal.tsx` — lift inline styles | component | request-response | No inline-style removal analog (first such lift in codebase) — use RESEARCH.md §Pitfall 2 | no analog |
| `frontend/src/components/StatusBar.tsx` — D-11 copy fix | component | request-response | n/a — single string change | no analog |
| `frontend/src/components/__tests__/StatusBar.test.tsx` — D-11 assertion update | test | n/a | Existing test file structure lines 66–70 | exact |
| `frontend/src/components/Hub/GroupSidebar.test.tsx` — CARRY-01 assertion update | test | n/a | Existing test file structure lines 261–275, 380–413 | exact |

---

## Pattern Assignments

### TOKEN MIGRATION PATTERN (used by all six CSS surface blocks)

**Analog:** `frontend/src/style.css` — Hub card + Hub group sidebar + Hub modal rules (lines 4107–5100)

This is the fully-migrated reference surface. Every property that a non-Hub surface currently sets with a hardcoded hex value has an exact counterpart in this section using `var(--hub-*)`.

**Imports pattern — how tokens are declared** (lines 3896–3943, `:root` block):
```css
:root {
  --hub-bg: #1a1b26;
  --hub-surface: #16161e;
  --hub-surface-elevated: #1e2030;
  --hub-border: #292e42;
  --hub-border-hover: #3b4261;
  --hub-text-primary: #c0caf5;
  --hub-text-secondary: #a9b1d6;
  --hub-text-muted: #9aa5ce;
  --hub-text-placeholder: #414868;
  --hub-accent: #7aa2f7;
  --hub-accent-hover: #89b4fa;
  --hub-destructive: #f7768e;
  --hub-success: #9ece6a;
  --hub-warning: #f59e0b;
  --hub-scrollbar: #3b4261;
  --hub-scrollbar-hover: #565f89;
  --hub-sidebar-item-active-bg: rgba(122,162,247,0.12);
  /* ... */
}
```

**Light theme override pattern** (lines 3946–3995):
```css
[data-ui-theme="light"] {
  --hub-bg: #f5f5f7;
  --hub-surface: #ffffff;
  --hub-surface-elevated: #ececf0;
  --hub-border: #d1d1db;
  --hub-border-hover: #9999b0;
  --hub-text-primary: #1a1b26;
  --hub-text-secondary: #3a3b50;
  --hub-text-muted: #5c5d80;
  --hub-accent: #3d6fe8;
  --hub-accent-hover: #2a56cf;
  --hub-destructive: #c0394f;
  --hub-sidebar-item-active-bg: rgba(61,111,232,0.10);
  /* ... */
}
```

**New token insertion pattern** — add immediately after the last token in each block, with both dark and light values in a single diff:
```css
/* In :root dark block (after line 3943) */
--hub-text-dim: #565f89;   /* very faint hint/separator text, dimmer than --hub-text-muted */

/* In [data-ui-theme="light"] block (after line 3995) */
--hub-text-dim: #9999b0;   /* matches --hub-text-placeholder light value — same tier */
```

**Core token usage pattern** (lines 4107–4143, fully-migrated hub-card):
```css
.hub-card {
  border: 1px solid var(--hub-border);
  background: var(--hub-surface);
}
.hub-card:hover {
  border-color: var(--hub-border-hover);
  background: var(--hub-surface-elevated);
}
.hub-card:focus-visible {
  outline: 2px solid var(--hub-accent);
  outline-offset: 2px;
}
```

**Active / selected item pattern** (lines 4697–4700):
```css
.hub__group-sidebar-item--active {
  background: var(--hub-sidebar-item-active-bg);
  border-left: 2px solid var(--hub-accent);
}
```

---

### Sidebar block migration (lines 218–293)

**Analog:** `frontend/src/style.css` — Hub group sidebar CSS (lines 4552–4760)

The Hub group sidebar is the exact already-tokenized counterpart of the app sidebar. Same structure: a container surface → item hover → item active → icon sizing.

**Current pattern to replace** (lines 219–271):
```css
/* BEFORE — raw hex */
.sidebar {
  background-color: #16161e;
  border-right: 1px solid #292e42;
  transition: width 0.15s ease;          /* layout-only — keep; consider motion guard */
}
.sidebar__toggle {
  color: #9aa5ce;
  transition: background-color 0.1s, color 0.1s;  /* must move to no-preference guard */
}
.sidebar__toggle:hover {
  background-color: #1e2030;
  color: #c0caf5;
}
.sidebar__item {
  color: #9aa5ce;
  transition: background-color 0.1s, color 0.1s;  /* must move to no-preference guard */
}
.sidebar__item:hover {
  background-color: #1e2030;
  color: #c0caf5;
}
/* line 4537 — active: partially tokenized */
.sidebar__item--active {
  background: rgba(122, 162, 247, 0.12);     /* → var(--hub-sidebar-item-active-bg) */
  color: var(--hub-accent, #7aa2f7);         /* already tokenized; keep but remove fallback hex */
}
```

**Migration target** (copy token pattern from Hub group sidebar lines 4553–4700):
```css
/* AFTER */
.sidebar {
  background-color: var(--hub-surface);
  border-right: 1px solid var(--hub-border);
}
.sidebar__toggle {
  color: var(--hub-text-muted);
}
.sidebar__toggle:hover {
  background-color: var(--hub-surface-elevated);
  color: var(--hub-text-primary);
}
.sidebar__item {
  color: var(--hub-text-muted);
}
.sidebar__item:hover {
  background-color: var(--hub-surface-elevated);
  color: var(--hub-text-primary);
}
.sidebar__item--active {
  background: var(--hub-sidebar-item-active-bg);
  color: var(--hub-accent);
}
```

**Motion guard pattern to add** (copy from `.hub__group-sidebar` guard at lines 4567–4571):
```css
@media (prefers-reduced-motion: no-preference) {
  .sidebar {
    transition: width 150ms ease;
  }
  .sidebar__toggle,
  .sidebar__item {
    transition: background-color 0.1s, color 0.1s;
  }
}
@media (prefers-reduced-motion: reduce) {
  .sidebar,
  .sidebar__toggle,
  .sidebar__item {
    transition: none;
  }
}
```

---

### S-01 Welcome Tab block migration (lines 1302–1404)

**Analog:** `frontend/src/style.css` — Hub card text and link rules (lines 4107–4200)

**Migration rule** (same token mapping as hub-card):
- `#1a1b26` (background) → `var(--hub-bg)`
- `#a9b1d6` (body text) → `var(--hub-text-secondary)`
- `#9aa5ce` (muted labels) → `var(--hub-text-muted)`
- `#c0caf5` (strong/primary text) → `var(--hub-text-primary)`
- `#7aa2f7` (links, labels) → `var(--hub-accent)`
- `#16161e` (code background) → `var(--hub-surface)`
- `#292e42` (code border) → `var(--hub-border)`

No new tokens needed for this surface. No per-selector light overrides needed — all tokens above are already declared in both blocks.

---

### S-03 Tab Bar + Status Bar block migration (lines 82–368)

**Analog:** `frontend/src/style.css` — Hub card + hub group sidebar CSS (lines 4107–4760)

**Current pattern showing the shape** (lines 82–141):
```css
.tab-bar {
  background-color: #16161e;       /* → var(--hub-surface) */
  border-bottom: 1px solid #292e42; /* → var(--hub-border) */
}
.tab {
  border-right: 1px solid #292e42;  /* → var(--hub-border) */
  color: #9aa5ce;                   /* → var(--hub-text-muted) */
  transition: background-color 0.1s; /* → move to no-preference guard */
}
.tab:hover {
  background-color: #1e2030;        /* → var(--hub-surface-elevated) */
  color: #a9b1d6;                   /* → var(--hub-text-secondary) */
}
.tab--active {
  background-color: #1a1b26;        /* → var(--hub-bg) */
  color: #c0caf5;                   /* → var(--hub-text-primary) */
  border-bottom: 2px solid #7aa2f7; /* → var(--hub-accent) */
}
```

**D-03 fence — DO NOT migrate these selectors** (lines 1097–1103, approximately):
```css
/* KEEP AS-IS: semantic per-agent identifiers */
.tab__agent-badge--claude   { background: #7aa2f7; }
.tab__agent-badge--opencode { background: #9ece6a; }
.tab__agent-badge--codex    { background: #bb9af7; }
.tab__agent-badge--gemini   { background: #2ac3de; }
.tab__agent-badge--cursor   { background: #e0af68; }
.tab__agent-badge--aider    { background: #f7768e; }
.tab__agent-badge--shell    { background: #89ddff; }

/* KEEP AS-IS: colorblind-safe semantic status (text label is primary) */
.tab-status-bar__state--on       { color: #9ece6a; }
.tab-status-bar__state--off      { color: #9aa5ce; }
.tab-status-bar__state--inactive { color: #414868; }
```

**New token needed**: `.tab-status-bar__hint` uses `#545c7e` — consolidate with `#565f89` into `--hub-text-dim` (add to both `:root` and `[data-ui-theme="light"]` blocks before migrating this selector).

---

### S-04 + S-05 File Browser + Editor Chrome block migration (lines 2646–3380)

**Analog:** `frontend/src/style.css` — Hub modal header/body (lines 5013–5100), Hub card hover (lines 4119–4127)

Same token vocabulary as other surfaces. Distinguishing features:
- Focus-visible outlines use `var(--hub-accent)` — copy from hub-card line 4124
- Selected row left-border uses `var(--hub-accent)` — copy from hub__group-sidebar-item--active line 4699
- Truncated-banner `border-left` uses `var(--hub-warning)` — same as `.hub-modal__error-banner` semantic pattern
- `#565f89` breadcrumb separator → `var(--hub-text-dim)` (same new token as status bar hint)
- Editor chrome (breadcrumb, save controls) mirrors file browser token usage exactly per D-02

---

### S-06 Settings Panel block migration (lines 370–771)

**Analog:** `frontend/src/style.css` — Hub modal interactive elements (lines 5147–5213), Hub card surface (lines 4107–4143)

Special cases:
- Scrollbar: `#3b4261` → `var(--hub-scrollbar)`; `:hover` `#565f89` → `var(--hub-scrollbar-hover)` — tokens already declared
- Toggle thumb off state `#565f89` → reuse `var(--hub-scrollbar-hover)` (same dim-slate value; planner may introduce `--hub-toggle-thumb-off` if named precision desired — add to both blocks)
- Save button primary: `#7aa2f7` bg + `#1a1b26` text → `var(--hub-accent)` bg + `var(--hub-bg)` text — copy from hub-modal send-btn pattern (lines 5189–5204)
- Saved state `#9ece6a` → `var(--hub-success)` (this is a UI state, not a semantic status identifier — migration allowed per D-03)

---

### S-07: New `.hub-share-modal*` CSS rules

**Analog:** `frontend/src/style.css` — `.hub-modal` + `.hub-modal__header` + `.hub-modal__body` (lines 4999–5100)

This is the closest structural match. The share modal already uses `hub-modal-overlay` (overlay already has CSS). Copy the panel/header/body layout from hub-modal, scale to 480px width, use simpler fade animation.

**Panel container pattern** (copy from `.hub-modal` lines 4999–5010):
```css
.hub-share-modal {
  position: relative;
  width: min(480px, calc(100vw - 48px));   /* narrower than hub-modal's 1100px */
  background: var(--hub-surface-elevated);
  border: 1px solid var(--hub-border);
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
```

**Header pattern** (copy from `.hub-modal__header` lines 5013–5021):
```css
.hub-share-modal__header {
  height: 48px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 16px;
  border-bottom: 1px solid var(--hub-border);
  flex-shrink: 0;
}
```

**Title pattern** (copy from `.hub-modal__session-name` lines 5023–5030):
```css
.hub-share-modal__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--hub-text-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
```

**Body pattern** (copy from `.hub-modal__body` lines 5094–5100):
```css
.hub-share-modal__body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 16px;
  overflow-y: auto;
  gap: 12px;
}
```

**Credential block pattern** (new — no analog; use RESEARCH.md §Pattern 3):
```css
.hub-share-modal__lan-creds {
  font-size: 12px;
  color: var(--hub-text-secondary);
}
.hub-share-modal__lan-creds code {
  background: var(--hub-surface);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
```

**Animation pattern** (copy keyframe + guard structure from `.hub-modal` lines 5219–5273, but fade-only not grow):
```css
/* Keyframes at root scope — no media query wrapper */
@keyframes hub-share-modal-in  { from { opacity: 0; } to { opacity: 1; } }
@keyframes hub-share-modal-out { from { opacity: 1; } to { opacity: 0; } }

/* Animation assignment — inside no-preference guard (copy guard structure from lines 5244–5257) */
@media (prefers-reduced-motion: no-preference) {
  .hub-share-modal--entering {
    animation: hub-share-modal-in 150ms ease forwards;
  }
  .hub-share-modal--exiting {
    animation: hub-share-modal-out 120ms ease forwards;
  }
}

/* Reduced-motion: instant appear/disappear (copy from hub-modal guard lines 5261–5273) */
@media (prefers-reduced-motion: reduce) {
  .hub-share-modal {
    animation: none;
    transition: none;
    opacity: 1;
  }
}
```

**Overlay phase classes** (overlay already has CSS; share modal reuses `hub-modal-overlay` and `hub-modal-overlay--${phase}` — these CSS rules already exist at lines 4988–4996 and the overlay animation keyframes are already declared at lines 5233–5241. No new overlay CSS needed.)

---

### Motion Guards on migrated surfaces (hover transitions)

**Analog 1:** `.find-bar` motion guard (lines 2440–2550) — static-first: transition declared on element, overridden with `transition: none` in reduce block.

**Analog 2:** `.hub__group-sidebar` motion guard (lines 4567–4571) — no-preference guard wraps the entire `transition` declaration.

This codebase uses **both** styles. For Phase 141, follow the Hub group sidebar pattern (no-preference guard wraps the transition declaration) since it is the more recent established pattern.

**Pattern to copy for hover transitions on migrated surfaces** (lines 4567–4571):
```css
@media (prefers-reduced-motion: no-preference) {
  .sidebar__toggle,
  .sidebar__item,
  .tab,
  .tab__close,
  .settings-panel__btn--cancel,
  .settings-panel__btn--save,
  .file-browser__list-row {
    transition: background-color 0.1s, color 0.1s;
  }
}
@media (prefers-reduced-motion: reduce) {
  .sidebar__toggle,
  .sidebar__item,
  .tab,
  .tab__close,
  .settings-panel__btn--cancel,
  .settings-panel__btn--save,
  .file-browser__list-row {
    transition: none;
  }
}
```

---

### CARRY-01: GroupSidebar.tsx ARIA fix

**Analog file:** `frontend/src/components/Hub/GroupSidebar.tsx` — the toggle button at lines 235–256 is the only native `<button>` in this component, showing the correct button pattern.

**Current broken pattern** (`GroupSidebarItem` render, lines 126–173):
```tsx
// GroupSidebar.tsx lines 126–147 — current (broken)
return (
  <li
    className={itemClass}
    role="option"                              // WRONG: <ul> has role="listbox"
    aria-selected={isActive ? 'true' : 'false'} // WRONG: listbox model
    tabIndex={0}                               // WRONG: <li> should be inert
    onClick={() => onGroupSelect(id)}
    onKeyDown={(e) => {                        // WRONG: not needed for native button
      if (e.target !== e.currentTarget) return
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

**CARRY-01 target pattern** (per 141-UI-SPEC.md §CARRY-01 and 141-RESEARCH.md §Pattern 4):
```tsx
// AFTER: <li> is inert; <button> is the interactive element
return (
  <li className={itemClass}>
    <button
      type="button"
      className="hub__group-sidebar-item"
      aria-pressed={isActive}
      aria-label={`${label} group, ${counts.running}/${counts.total} sessions`}
      onClick={() => onGroupSelect(id)}
      // NO onKeyDown needed — native button handles Enter/Space natively
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      {/* children unchanged */}
    </button>
  </li>
)
```

**`<ul>` change** (GroupSidebar.tsx lines 264–268):
```tsx
// BEFORE:
<ul id={SIDEBAR_LIST_ID} className="hub__group-sidebar-list" role="listbox">

// AFTER:
<ul
  id={SIDEBAR_LIST_ID}
  className="hub__group-sidebar-list"
  aria-labelledby="hub-group-sidebar-heading"
  // NO role="listbox"
>
```

**`<aside>` label change** (GroupSidebar.tsx — find the `<aside>` wrapper and add `aria-label`):
```tsx
// AFTER:
<aside aria-label="Session groups" className={`hub__group-sidebar${collapsed ? ' hub__group-sidebar--collapsed' : ''}`}>
```

**`<span>` heading id** (GroupSidebar.tsx line 260):
```tsx
// AFTER:
<span id="hub-group-sidebar-heading" className="hub__group-sidebar-heading">Groups</span>
```

---

### GroupSidebar.test.tsx — CARRY-01 assertion updates

**Analog:** The existing test structure in `GroupSidebar.test.tsx` (lines 206–275, 380–413).

**Tests that assert the OLD (broken) pattern — must be updated:**

Lines 206–221 (active state + aria-selected):
```tsx
// BEFORE:
it('active group item has hub__group-sidebar-item--active class and aria-selected="true"', () => {
  // ...
  expect(items[1].getAttribute('aria-selected')).toBe('true')
})
it('non-active items have aria-selected="false"', () => {
  // ...
  expect(items[0].getAttribute('aria-selected')).toBe('true') // All is active
  expect(items[1].getAttribute('aria-selected')).toBe('false')
})
```

```tsx
// AFTER: items are now <li>s; the <button> inside carries aria-pressed
it('active group item has hub__group-sidebar-item--active class and aria-pressed="true"', () => {
  // ...
  const btn = items[1].querySelector('button')
  expect(btn!.getAttribute('aria-pressed')).toBe('true')  // or boolean true
})
it('non-active items have aria-pressed="false"', () => {
  // ...
  const allBtn = items[0].querySelector('button')
  expect(allBtn!.getAttribute('aria-pressed')).toBe('true')
  const alphaBtn = items[1].querySelector('button')
  expect(alphaBtn!.getAttribute('aria-pressed')).toBe('false')
})
```

Lines 261–275 (ARIA role tests):
```tsx
// BEFORE:
it('list has role="listbox"', () => {
  expect(list!.getAttribute('role')).toBe('listbox')
})
it('each item has role="option"', () => {
  items.forEach((item) => {
    expect(item.getAttribute('role')).toBe('option')
  })
})
```

```tsx
// AFTER:
it('list has no role attribute (plain <ul>)', () => {
  expect(list!.getAttribute('role')).toBeNull()
})
it('each item has an inner button with role="button" and aria-pressed', () => {
  items.forEach((item) => {
    const btn = item.querySelector('button')
    expect(btn).not.toBeNull()
    expect(btn!.getAttribute('type')).toBe('button')
    expect(btn!.hasAttribute('aria-pressed')).toBe(true)
  })
})
```

Lines 380–388 (keyboard operability — tabIndex on items):
```tsx
// BEFORE:
it('group sidebar items have tabIndex 0 (keyboard-focusable)', () => {
  items.forEach((item) => {
    expect((item as HTMLElement).tabIndex).toBe(0)
  })
})
```

```tsx
// AFTER: <li> is inert; the <button> inside is keyboard-focusable
it('inner buttons inside group sidebar items have tabIndex 0 (keyboard-focusable)', () => {
  items.forEach((item) => {
    const btn = item.querySelector('button') as HTMLButtonElement
    expect(btn.tabIndex).toBe(0)  // native button default
  })
})
```

Lines 389–413 (Enter/Space keyboard dispatch must target the inner button, not the `<li>`):
```tsx
// AFTER: dispatch keydown on the <button>, not the <li>
const alphaItem = items[1] as HTMLElement
const alphaBtn = alphaItem.querySelector('button') as HTMLButtonElement
act(() => {
  alphaBtn.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
})
// Note: native button fires onClick on Enter/Space; test may need to use .click() instead
// if jsdom's keyboard simulation does not activate native button
```

Also add new assertion (lines 261–275 section) to verify `aria-labelledby` on `<ul>`:
```tsx
it('list has aria-labelledby="hub-group-sidebar-heading"', () => {
  const list = container.querySelector('.hub__group-sidebar-list')
  expect(list!.getAttribute('aria-labelledby')).toBe('hub-group-sidebar-heading')
})
```

---

### StatusBar.tsx — D-11 copy fix

**Location:** `frontend/src/components/StatusBar.tsx`

Line 15 (JSDoc comment — editorial reword):
```tsx
// BEFORE: "is the Sessions tab's job (SessionSharePanel renders cap-bearing Read-Only"
// AFTER:  "is the Hub card's job (SessionShareModal renders cap-bearing Read-Only"
```

Line 49 (functional string — must match the test assertion):
```tsx
// BEFORE:
Share links are on the Sessions tab

// AFTER:
Share — open the Hub card
```

---

### StatusBar.test.tsx — D-11 assertion update

**Analog:** Existing test structure at lines 66–70 — exact string match.

Line 66 (test description):
```tsx
// BEFORE:
it('shows hint pointing to Sessions tab when web enabled (Phase 87 cleanup)', () => {
// AFTER:
it('shows hint pointing to Hub card when web enabled (Phase 87 cleanup)', () => {
```

Line 70 (assertion):
```tsx
// BEFORE:
expect(hint?.textContent).toBe('Share links are on the Sessions tab')
// AFTER:
expect(hint?.textContent).toBe('Share — open the Hub card')
```

---

### SessionShareModal.tsx — inline style lift

**Location:** `frontend/src/components/Hub/SessionShareModal.tsx` lines 292–295

No existing analog in codebase (first inline-style removal). The pattern is mechanical: remove the `color` and `background` properties from the `style={{...}}` attribute; the CSS class rule provides them.

Line 292:
```tsx
// BEFORE:
<div className="hub-share-modal__lan-creds" style={{ margin: '8px 0', fontSize: 12, color: '#a9b1d6' }}>
// AFTER: remove color from inline style; keep margin and fontSize (structural, not color)
<div className="hub-share-modal__lan-creds" style={{ margin: '8px 0', fontSize: 12 }}>
```

Line 294:
```tsx
// BEFORE:
<code style={{ background: '#16161e', padding: '2px 6px', borderRadius: 3, fontFamily: 'monospace' }}>
// AFTER: remove background; keep structural props or move all to CSS class
<code>
// (all properties moved to .hub-share-modal__lan-creds code CSS rule)
```

---

## Shared Patterns

### Token Resolution (apply to all six CSS surface blocks)
**Source:** `frontend/src/style.css` `:root` block (lines 3896–3943)
**Apply to:** Every hex constant in sidebar, S-01, S-03, S-04, S-05, S-06 blocks
```css
/* 9 values cover 95% of all occurrences: */
#1a1b26 → var(--hub-bg)
#16161e → var(--hub-surface)
#1e2030 → var(--hub-surface-elevated)
#292e42 → var(--hub-border)
#3b4261 → var(--hub-border-hover)  [also --hub-scrollbar]
#c0caf5 → var(--hub-text-primary)
#a9b1d6 → var(--hub-text-secondary)
#9aa5ce → var(--hub-text-muted)
#7aa2f7 → var(--hub-accent)
#89b4fa → var(--hub-accent-hover)
#f7768e → var(--hub-destructive)
#9ece6a → var(--hub-success)
#f59e0b → var(--hub-warning)
#565f89 → var(--hub-text-dim)       [NEW token — add to both blocks first]
#545c7e → var(--hub-text-dim)       [same new token — close enough per RESEARCH.md A2]
```

### Motion Guard (apply to all CSS surface blocks that have transitions)
**Source:** `frontend/src/style.css` lines 4567–4571 (hub__group-sidebar guard)
**Apply to:** `.sidebar`, `.sidebar__toggle`, `.sidebar__item`, `.tab`, `.tab__close`, `.settings-panel__*-btn`, `.file-browser__*` hover states
```css
@media (prefers-reduced-motion: no-preference) {
  /* transition declarations go here */
}
@media (prefers-reduced-motion: reduce) {
  /* transition: none overrides go here */
}
```

### Light Theme Auto-flip (apply to all token usages)
**Source:** `frontend/src/style.css` `[data-ui-theme="light"]` block (lines 3946–3995)
**Apply to:** All migrated hex → `var(--hub-*)` replacements
No per-selector `[data-ui-theme="light"]` overrides needed unless a NEW token is introduced. New tokens (`--hub-text-dim`) must be added to both `:root` and `[data-ui-theme="light"]` in the same diff.

### Focus-visible Ring (apply to all interactive elements in migrated surfaces)
**Source:** `frontend/src/style.css` lines 4131–4143
**Apply to:** All `:focus-visible` rules in migrated surfaces
```css
outline: 2px solid var(--hub-accent);
outline-offset: 2px;
```

### ARIA — native button replaces custom keyboard handler
**Source:** `frontend/src/components/Hub/GroupSidebar.tsx` lines 235–256 (existing toggle button)
**Apply to:** `GroupSidebarItem` render function — replace `<li onClick onKeyDown tabIndex>` with `<li><button type="button" aria-pressed>`
The native `<button>` element handles Enter and Space natively — no `onKeyDown` needed.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `SessionShareModal.tsx` inline style lift | component | n/a | No prior inline-style removal in codebase; pattern is mechanical (delete color/background from style={{}}; they are covered by new CSS rule) |
| `StatusBar.tsx` D-11 copy | component | n/a | Single string literal change — no analog needed |

---

## D-03 Fence Summary (DO NOT MIGRATE)

The following hex values appear in migrated CSS sections but must remain as hardcoded hex per D-03:

```
/* Agent badge colors — semantic per-agent identifiers */
.tab__agent-badge--claude   { background: #7aa2f7; }   /* note: same as accent, different semantic purpose */
.tab__agent-badge--opencode { background: #9ece6a; }
.tab__agent-badge--codex    { background: #bb9af7; }
.tab__agent-badge--gemini   { background: #2ac3de; }
.tab__agent-badge--cursor   { background: #e0af68; }
.tab__agent-badge--aider    { background: #f7768e; }
.tab__agent-badge--shell    { background: #89ddff; }

/* Semantic status state colors — colorblind-safe: text label is primary differentiator */
.tab-status-bar__state--on       { color: #9ece6a; }
.tab-status-bar__state--off      { color: #9aa5ce; }
.tab-status-bar__state--inactive { color: #414868; }
```

---

## Metadata

**Analog search scope:** `frontend/src/style.css` (5273+ lines), `frontend/src/components/Hub/`, `frontend/src/components/__tests__/`
**Files read:** `style.css` (targeted reads: lines 82–368, 218–293, 3890–3995, 4100–4760, 4988–5100, 5219–5273, 2421–2550), `GroupSidebar.tsx`, `GroupSidebar.test.tsx`, `StatusBar.tsx`, `StatusBar.test.tsx`, `SessionShareModal.tsx` (excerpt)
**Pattern extraction date:** 2026-06-20
