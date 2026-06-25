# Phase 148: Session Tab Chevron - Pattern Map

**Mapped:** 2026-06-22
**Files analyzed:** 2
**Analogs found:** 2 / 2

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/TabBar.tsx` | component | event-driven | `frontend/src/components/TabBar.tsx` (self — extend existing patterns) | exact |
| `frontend/src/style.css` | config/style | — | `frontend/src/style.css` (self — `.tab__close`, `.tab-bar__chevron`, `--hub-*` tokens) | exact |

---

## Pattern Assignments

### `frontend/src/components/TabBar.tsx` (component, event-driven)

**Analog:** Self — the file already contains every pattern the chevron must copy.

---

#### Imports (lines 1–5)

```tsx
import React, { useState, useRef, useEffect } from 'react'
import { agentBadgeModifier } from '../lib/agentBadge'
```

No new imports are needed. The chevron button uses only React primitives already present.

---

#### `contextMenu` state declaration (line 89)

```tsx
const [contextMenu, setContextMenu] = useState<{ tabId: string; x: number; y: number } | null>(null)
```

The chevron's `onClick` calls `setContextMenu` with rect-derived coords (D-01). The state shape is unchanged — only the `x`/`y` source differs between the chevron path (rect) and the right-click path (cursor).

---

#### Right-click trigger — existing cursor-position path (lines 226–234) — PRESERVE AS-IS (D-02)

```tsx
<span
  className="tab__name"
  onDoubleClick={(e) => startEdit(tab, e)}
  onContextMenu={(e) => {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ tabId: tab.id, x: e.clientX, y: e.clientY })
  }}
  title={titleText}
>
  {tab.name}
</span>
```

Right-click behavior must not change. The chevron is an additive element that reuses `setContextMenu` with a different coord source.

---

#### Close button — structural template for the chevron button (lines 239–249)

```tsx
<button
  className="tab__close"
  onClick={(e) => {
    e.stopPropagation()
    onClose(tab.id)
  }}
  title="Close tab"
  aria-label={`Close ${tab.name}`}
>
  ×
</button>
```

**Copy this pattern verbatim for the chevron, substituting:**
- `className="tab__chevron"` (new class)
- `onClick`: call `e.stopPropagation()`, then compute rect and call `setContextMenu({ tabId: tab.id, x: rect.left, y: rect.bottom })`
- `title="Session menu"`
- `aria-label="Session menu"` (exact string per #68)
- glyph: `▾` (U+25BE) or `&#9662;` — Claude's Discretion per CONTEXT.md
- `data-testid="tab-chevron"` (recommended, mirrors `data-testid="tab-context-browse-files"` on line 311)
- Session-gate: render only when `tab.sessionId` is truthy (D-05)

**New element position (D-03):** Insert the chevron button between the countdown span (line 237) and the close button (line 239). Tab element order becomes: `status dot · agent badge · name · (countdown) · chevron · close ×`.

**Rect-anchored open pattern** (implement inside `onClick`):
```tsx
onClick={(e) => {
  e.stopPropagation()
  const rect = (e.currentTarget as HTMLButtonElement).getBoundingClientRect()
  setContextMenu({ tabId: tab.id, x: rect.left, y: rect.bottom })
}}
```

---

#### Session-gate pattern (lines 303–319 — existing `tab.sessionId` truthiness gate)

```tsx
{(() => {
  const tab = tabs.find((t) => t.id === contextMenu.tabId)
  if (!tab?.sessionId) return null
  // ...
})()}
```

D-05 chevron gate uses the same predicate directly in JSX:
```tsx
{tab.sessionId && (
  <button className="tab__chevron" ...>▾</button>
)}
```

---

#### Context menu render block (lines 273–322) — UNCHANGED

```tsx
{contextMenu && tabs.some(t => t.id === contextMenu.tabId) && (
  <div
    className="tab__context-menu"
    role="menu"
    style={{ position: 'fixed', top: contextMenu.y, left: contextMenu.x }}
    onMouseDown={(e) => e.stopPropagation()}
  >
    <button role="menuitem" className="tab__context-menu__item"
      onClick={() => { startEditById(contextMenu.tabId); setContextMenu(null) }}>
      Rename
    </button>
    <button role="menuitem" className="tab__context-menu__item"
      onClick={() => { onRequestSave?.(contextMenu.tabId); setContextMenu(null) }}>
      Save Terminal As…
    </button>
    {(() => {
      const tab = tabs.find((t) => t.id === contextMenu.tabId)
      if (!tab?.sessionId) return null
      if (!onBrowseFiles) return null
      return (
        <button role="menuitem" className="tab__context-menu__item"
          data-testid="tab-context-browse-files"
          onClick={() => { onBrowseFiles(tab.sessionId, tab.name); setContextMenu(null) }}>
          Browse files
        </button>
      )
    })()}
  </div>
)}
```

The menu render block is unchanged. The `style` prop already drives `position: fixed` with `top: contextMenu.y, left: contextMenu.x` — when `y` comes from `rect.bottom` and `x` from `rect.left`, it naturally positions the menu below the chevron (D-01). No structural change to the menu is needed.

---

#### Scroll-chevron buttons (lines 175–182, 264–271) — precedent for `<button>` + Unicode glyph + `aria-label`

```tsx
<button
  className="tab-bar__chevron tab-bar__chevron--left"
  onClick={() => { listRef.current?.scrollBy({ left: -160, behavior: 'smooth' }) }}
  aria-label="Scroll tabs left"
  tabIndex={0}
>&#8249;</button>
```

Confirms the project's Unicode-glyph-in-button approach. The new `.tab__chevron` follows the same pattern (semantic `<button>`, Unicode glyph, `aria-label`). The scroll chevrons omit `tabIndex={0}` convention is overridden by native button focus — the tab chevron relies on native button focus without explicit `tabIndex`.

---

### `frontend/src/style.css` (config/style)

**Analog:** Self — extract three existing rule clusters as templates.

---

#### `.tab__close` — sizing/token template for `.tab__chevron` (lines 202–221)

```css
.tab__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: none;
  background: transparent;
  color: var(--hub-text-muted);
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  border-radius: 2px;
  flex-shrink: 0;
  padding: 0;
}
.tab__close:hover {
  background-color: var(--hub-border-hover);
  color: var(--hub-destructive);
}
```

**Copy for `.tab__chevron`**, substituting on hover `color: var(--hub-text-primary)` instead of `--hub-destructive` (the chevron is non-destructive; use the same hover text lightening the scroll chevron uses).

---

#### `.tab-bar__chevron` — hover color precedent (lines 234–255)

```css
.tab-bar__chevron {
  flex-shrink: 0;
  width: 24px;
  height: 100%;
  background: var(--hub-surface);
  border: none;
  color: var(--hub-text-muted);
  font-size: 16px;
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-right: 1px solid var(--hub-border);
}
.tab-bar__chevron:hover {
  color: var(--hub-text-primary);
}
```

Confirms hover pattern for non-destructive chevrons: `color: var(--hub-text-primary)` on hover. Apply same to `.tab__chevron:hover`.

---

#### `prefers-reduced-motion` transition block (lines 257–270) — the new `.tab__chevron` must be added here

```css
@media (prefers-reduced-motion: no-preference) {
  .tab,
  .tab__close,
  .tab-status-bar__btn {
    transition: background-color 0.1s, color 0.1s;
  }
}
@media (prefers-reduced-motion: reduce) {
  .tab,
  .tab__close,
  .tab-status-bar__btn {
    transition: none;
  }
}
```

Add `.tab__chevron` to both selector lists alongside `.tab__close`.

---

#### `.tab__context-menu` — D-07 tokenization target (lines 1623–1651, hardcoded dark hex values)

Current state (to be replaced with tokens):

```css
/* ─── Tab context menu ─────────────────────────────────────────── */
.tab__context-menu {
  background: #1e2030;           /* → var(--hub-surface-elevated)  */
  border: 1px solid #292e42;    /* → var(--hub-border)             */
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
  min-width: 120px;
  z-index: 500;
  padding: 4px 0;
}

.tab__context-menu__item {
  display: block;
  width: 100%;
  padding: 8px 12px;
  font-size: 13px;
  font-weight: 400;
  color: #a9b1d6;               /* → var(--hub-text-secondary)     */
  background: none;
  border: none;
  text-align: left;
  cursor: pointer;
  font-family: inherit;
}

.tab__context-menu__item:hover {
  color: #c0caf5;               /* → var(--hub-text-primary)       */
  background: #292e42;          /* → var(--hub-border-hover)       */
}
```

**Token substitution map for D-07:**

| Hardcoded hex | Token | Dark value | Light value |
|---------------|-------|------------|-------------|
| `#1e2030` (menu bg) | `var(--hub-surface-elevated)` | `#1c1e28` | `#ececf0` |
| `#292e42` (border + hover bg) | `var(--hub-border)` / `var(--hub-border-hover)` | `#41454f` / `#54586a` | `#d1d1db` / `#9999b0` |
| `#a9b1d6` (item text) | `var(--hub-text-secondary)` | `#c7cad6` | `#3a3b50` |
| `#c0caf5` (hover text) | `var(--hub-text-primary)` | `#f4f5f8` | `#1a1b26` |

Notes: `box-shadow`, `border-radius`, `min-width`, `z-index`, `padding` are theme-independent — keep as-is. The border uses `var(--hub-border)` (not `--hub-border-hover`) since it's a resting state. The hover `background` maps to `var(--hub-border-hover)` which is the same token `.tab__close:hover` uses, establishing consistency.

---

## Shared Patterns

### Token usage (applies to both the chevron button and the tokenized context menu)

**Source:** `frontend/src/style.css` `:root` block (lines 4519–4586) and `[data-ui-theme="light"]` block (lines 4589–4657).

**Key tokens for this phase:**

```
--hub-text-muted        dark: #9398a8   light: #5c5d80   (chevron default color)
--hub-text-secondary    dark: #c7cad6   light: #3a3b50   (menu item text)
--hub-text-primary      dark: #f4f5f8   light: #1a1b26   (chevron hover + menu item hover text)
--hub-border            dark: #41454f   light: #d1d1db   (menu border)
--hub-border-hover      dark: #54586a   light: #9999b0   (chevron hover bg + menu item hover bg)
--hub-surface-elevated  dark: #1c1e28   light: #ececf0   (menu background)
```

**Apply to:** `.tab__chevron` default/hover + `.tab__context-menu` tokenization (D-07).

---

### `stopPropagation` on interactive tab children

**Source:** `TabBar.tsx` line 241 (close button), line 228 (context menu trigger).

Every button inside a `.tab` div must call `e.stopPropagation()` in its `onClick` to prevent the tab's outer `onClick={() => onSelect(tab.id)}` from firing. The chevron must do the same.

---

### `prefers-reduced-motion` compliance

**Source:** `style.css` lines 257–270.

Any new CSS rule with `transition` on interactive tab elements must be added to both the `no-preference` (enable) and `reduce` (disable) `@media` blocks. Add `.tab__chevron` alongside `.tab__close`.

---

## No Analog Found

None — both files being modified are already in the codebase and serve as their own analogs. All patterns are extracted from existing code in `TabBar.tsx` and `style.css`.

---

## Metadata

**Analog search scope:** `frontend/src/components/TabBar.tsx`, `frontend/src/style.css`
**Files scanned:** 2 (full reads; TabBar.tsx is 325 lines, style.css targeted reads at lines 195–270, 1618–1666, 4515–4657)
**Pattern extraction date:** 2026-06-22
