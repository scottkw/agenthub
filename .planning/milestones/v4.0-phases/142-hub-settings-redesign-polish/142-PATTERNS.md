# Phase 142: Hub & Settings Redesign Polish — Pattern Map

**Mapped:** 2026-06-21
**Files analyzed:** 10 modified + 1 deleted + 1 new test file
**Analogs found:** 10 / 10 (all files have close codebase analogs)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `frontend/src/components/Sidebar.tsx` | component | event-driven | `frontend/src/components/Hub/GroupSidebar.tsx` (group sub-list) + itself (collapse pattern) | exact |
| `frontend/src/App.tsx` | provider/store | event-driven | itself — `uiTheme` state lift is the same idiom as `groupDefs` lift | exact |
| `frontend/src/components/Hub/HubPanel.tsx` | component | CRUD | itself — remove GroupSidebar render, receive lifted props | exact |
| `frontend/src/components/Hub/GroupSidebar.tsx` | component | event-driven | **DELETED** — logic absorbed into Sidebar.tsx | n/a |
| `frontend/src/components/TerminalPanel.tsx` | component | event-driven | itself — effect coordination hardening | exact |
| `frontend/src/components/SettingsTab.tsx` | component | request-response | itself — replace control within existing `Appearance` section | exact |
| `frontend/src/components/Hub/HubFilterBar.tsx` | component | request-response | `frontend/src/components/Hub/HubEmptyState.tsx` | role-match |
| `frontend/src/components/Hub/HubEmptyState.tsx` | component | request-response | `frontend/src/components/Hub/HubFilterBar.tsx` | role-match |
| `frontend/src/components/Hub/SessionCard.tsx` | component | CRUD | itself — CSS padding-top gutter change | exact |
| `frontend/src/style.css` | config/CSS | n/a | itself — `--hub-*` token + `[data-ui-theme="light"]` discipline | exact |
| `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx` | test | request-response | itself — update existing assertions | exact |

---

## Pattern Assignments

### `frontend/src/components/Sidebar.tsx` (component, event-driven) — POL-05

**Analog 1 (collapse/localStorage pattern):** `frontend/src/components/Sidebar.tsx` lines 1–81 (current file, itself)

**Analog 2 (group sub-list ARIA + drag-drop pattern):** `frontend/src/components/Hub/GroupSidebar.tsx` lines 89–183

**Imports pattern to add** (copy from GroupSidebar.tsx lines 4–14):
```tsx
import { useState, useCallback } from 'react'
import {
  ChevronDownIcon,  // or ChevronRightIcon/ChevronDownIcon for sub-list expand
} from '@heroicons/react/24/outline'
import type { HubGroupDef } from '../lib/hubGroups'
```

**Collapse/localStorage pattern** (Sidebar.tsx lines 9, 26–36):
```tsx
const STORAGE_KEY = 'sidebar-collapsed'

const [collapsed, setCollapsed] = useState<boolean>(
  () => localStorage.getItem(STORAGE_KEY) === 'true'
)
const toggle = () => {
  setCollapsed((prev) => {
    const next = !prev
    localStorage.setItem(STORAGE_KEY, String(next))
    return next
  })
}
```
Note: the group sub-list expand/collapse is a separate local state, NOT persisted to localStorage — it is always visible when the sidebar is expanded. Only the main `sidebar-collapsed` key controls sidebar width.

**Group sub-list expanded state pattern** — new, modeled on the existing collapse idiom (no persistence):
```tsx
const [groupsExpanded, setGroupsExpanded] = useState(true)
```

**New SidebarProps additions** (all required for POL-05):
```tsx
interface SidebarProps {
  onHome: () => void
  onSettings: () => void
  onOpenHub: () => void
  activePanel?: string
  // POL-05 additions:
  groupDefs: HubGroupDef[]
  activeGroupId: string | null
  onGroupSelect: (id: string | null) => void
  onCreateGroup: (name: string) => void
  onDropOnGroup: (groupId: string, mKey: string) => void
  groupCounts: Record<string, { running: number; total: number; attention: number; waiting: number }>
  globalGroupCounts: { running: number; total: number; attention: number; waiting: number }
}
```

**Group sub-list item ARIA + drag-drop pattern** (GroupSidebar.tsx lines 98–183):
```tsx
// The <li> owns visual classes + drag handlers; the inner <button> owns interactive ARIA.
// CARRY-01 pattern: kept from GroupSidebarItem — DO NOT invert this structure.
const [isDragOver, setIsDragOver] = useState(false)

const handleDragOver = useCallback((e: React.DragEvent<HTMLLIElement>) => {
  e.preventDefault()
  e.stopPropagation()
  setIsDragOver(true)
}, [])

const handleDragLeave = useCallback(() => {
  setIsDragOver(false)
}, [])

const handleDrop = useCallback((e: React.DragEvent<HTMLLIElement>) => {
  e.preventDefault()
  setIsDragOver(false)
  if (id === null) return  // Cannot drop on "All"
  const key = e.dataTransfer.getData('text/plain')
  if (key) onDropOnGroup(id, key)
}, [id, onDropOnGroup])

// In render:
<li
  className={`sidebar__group-item${isActive ? ' sidebar__group-item--active' : ''}${isDragOver ? ' sidebar__group-item--drag-over' : ''}`}
  onDragOver={handleDragOver}
  onDragLeave={handleDragLeave}
  onDrop={handleDrop}
>
  <button
    type="button"
    className="sidebar__group-item__btn"
    aria-pressed={isActive}
    aria-label={`${label} group, ${counts.running}/${counts.total} sessions`}
    onClick={() => { onGroupSelect(id); onOpenHub() }}
    onKeyDown={(e) => {
      if (e.key === ' ') { e.preventDefault(); onGroupSelect(id); onOpenHub() }
    }}
  >
    <span className="sidebar__group-item__name">{label}</span>
    <span className="sidebar__group-item__count">{counts.running}/{counts.total}</span>
  </button>
</li>
```

**Inline group creation input pattern** (GroupSidebar.tsx lines 211–243):
```tsx
const [creating, setCreating] = useState(false)
const [inputValue, setInputValue] = useState('')

const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
  if (e.key === 'Enter') {
    const trimmed = inputValue.trim()
    if (trimmed) { onCreateGroup(trimmed); setCreating(false); setInputValue('') }
    // Empty Enter: no-op (input stays open)
  } else if (e.key === 'Escape') {
    setCreating(false); setInputValue('')
  }
}
const handleInputBlur = () => {
  const trimmed = inputValue.trim()
  if (trimmed) onCreateGroup(trimmed)
  setCreating(false); setInputValue('')
}

// In render (only when !collapsed):
{creating ? (
  <input
    className="sidebar__group-new-input"
    type="text"
    placeholder="Group name…"
    value={inputValue}
    onChange={(e) => setInputValue(e.target.value)}
    onKeyDown={handleInputKeyDown}
    onBlur={handleInputBlur}
    autoFocus
  />
) : (
  <button type="button" className="sidebar__group-new" onClick={() => { setCreating(true); setInputValue('') }}>
    New group
  </button>
)}
```

**Active item styling pattern** (Sidebar.tsx line 61; style.css line 4692):
```tsx
className={`sidebar__item${activePanel === '__hub__' ? ' sidebar__item--active' : ''}`}
```
```css
/* style.css line 4692 */
.sidebar__item--active {
  background: var(--hub-sidebar-item-active-bg);
  color: var(--hub-accent);
}
```

---

### `frontend/src/App.tsx` (provider/store, event-driven) — POL-05

**Analog:** `frontend/src/components/Hub/HubPanel.tsx` lines 259–285 (current owner of group state) and `App.tsx` lines 279–294 (existing `uiTheme` state lift pattern)

**State lift pattern** — copy `groupDefs`/`activeGroupId` state init from HubPanel.tsx lines 259–260, guard localStorage with try/catch per lines 263–271:
```tsx
// App.tsx additions — POL-05 group state lift
const [groupDefs, setGroupDefs] = useState<HubGroupDef[]>(() => loadGroups())
const [activeGroupId, setActiveGroupId] = useState<string | null>(null)

// Callback: counts flow UP from HubPanel via this callback
// (avoids lifting allSessions to App.tsx)
const [groupCounts, setGroupCounts] = useState<
  Record<string, { running: number; total: number; attention: number; waiting: number }>
>({})
const [globalGroupCounts, setGlobalGroupCounts] = useState(
  { running: 0, total: 0, attention: 0, waiting: 0 }
)
const handleGroupCountsChange = useCallback(
  (counts: typeof groupCounts, global: typeof globalGroupCounts) => {
    setGroupCounts(counts)
    setGlobalGroupCounts(global)
  },
  []
)
```

**useCallback pattern for group mutations** (modeled on `handleUiThemeChange` at App.tsx line 291):
```tsx
const handleGroupSelect = useCallback((id: string | null) => {
  setActiveGroupId(id)
}, [])

const handleCreateGroup = useCallback((name: string) => {
  setGroupDefs((prev) => createGroup(prev, name))
}, [])

const handleDropOnGroup = useCallback((groupId: string, key: string) => {
  setGroupDefs((prev) => assignToGroup(prev, groupId, key))
}, [])
```

**Sidebar render additions** (modeled on existing Sidebar render at App.tsx ~line 1460):
```tsx
<Sidebar
  onHome={handleHome}
  onSettings={handleOpenSettings}
  onOpenHub={handleOpenHub}
  activePanel={activeId}
  groupDefs={groupDefs}
  activeGroupId={activeGroupId}
  onGroupSelect={handleGroupSelect}
  onCreateGroup={handleCreateGroup}
  onDropOnGroup={handleDropOnGroup}
  groupCounts={groupCounts}
  globalGroupCounts={globalGroupCounts}
/>
```

**HubPanel render additions** — pass lifted state down:
```tsx
<HubPanel
  // ...existing props unchanged...
  activeGroupId={activeGroupId}
  groupDefs={groupDefs}
  onDropOnGroup={handleDropOnGroup}
  onGroupCountsChange={handleGroupCountsChange}
/>
```

---

### `frontend/src/components/Hub/HubPanel.tsx` (component, CRUD) — POL-05

**Analog:** itself (surgical changes only)

**Lines to remove:** 259–285 (`groupDefs` state, `activeGroupId` state, `sidebarCollapsed` state, `handleSidebarToggle` callback), line 11 (`import { GroupSidebar }`), lines 508–519 (GroupSidebar render in `hub__body`).

**Props additions** (receive from App.tsx):
```tsx
// New required props replacing the removed internal state:
activeGroupId: string | null              // was internal state
groupDefs: HubGroupDef[]                  // was internal state
onDropOnGroup: (groupId: string, mKey: string) => void  // was internal callback
onGroupCountsChange: (
  counts: Record<string, { running: number; total: number; attention: number; waiting: number }>,
  global: { running: number; total: number; attention: number; waiting: number }
) => void
```

**hub__body simplification** (lines 508–525 become):
```tsx
{/* POL-05: GroupSidebar removed — grid spans full width */}
<div className="hub__grid-scroll">
  {body}
</div>
```

**Counts emission** — add a `useEffect` that calls `onGroupCountsChange` whenever `allSessions` or `groupDefs` changes. Copy `computeCounts`/`computeGlobalCounts` functions from GroupSidebar.tsx lines 25–53 into HubPanel.tsx (or a separate `lib/hubGroupCounts.ts`).

---

### `frontend/src/components/TerminalPanel.tsx` (component, event-driven) — POL-04

**Analog:** itself — lines 647–704 (isActive fit effect + theme effect)

**Current theme effect** (lines 696–704) — the target to replace:
```tsx
// CURRENT — NO isActive guard:
useEffect(() => {
  if (!termRef.current) return
  termRef.current.options.theme = theme
  termRef.current.clearTextureAtlas()
  termRef.current.refresh(0, termRef.current.rows - 1)
}, [theme])
```

**Hardened pattern to implement:**

Add ref near other refs (after `fitAddonRef`):
```tsx
const pendingThemeRef = useRef<ITheme | null>(null)
```

Replace theme effect (lines 696–704) with:
```tsx
// POL-04: Hardened theme effect — defers atlas rebuild to when panel is active.
// A display:none panel has rows=0; clearTextureAtlas() + refresh on zero-dim corrupts WebGL state.
useEffect(() => {
  if (!termRef.current) return
  if (!isActive) {
    // Stash for application when this panel next becomes active
    pendingThemeRef.current = theme
    return
  }
  termRef.current.options.theme = theme
  termRef.current.clearTextureAtlas()
  fitTerminal(termRef.current)   // NOT refresh() — recalculates cell dims after atlas clear
}, [theme, isActive])
```

**isActive fit effect additions** (lines 647–687) — drain pendingThemeRef BEFORE the rAF loop starts (lines 648–651 become):
```tsx
useEffect(() => {
  if (!isActive || !containerRef.current) return

  // POL-04: drain any theme that arrived while this panel was hidden.
  // Must happen synchronously before the rAF loop so the first fit uses the correct theme.
  if (pendingThemeRef.current && termRef.current) {
    termRef.current.options.theme = pendingThemeRef.current
    termRef.current.clearTextureAtlas()
    pendingThemeRef.current = null
    // fitTerminal is called by the rAF loop below — no extra call needed here
  }

  // ... existing rAF loop unchanged (lines 651–687) ...
}, [isActive])
```

**Font-size effect** (lines 690–694) — already correct, no change:
```tsx
useEffect(() => {
  if (!termRef.current || !fitAddonRef.current) return
  termRef.current.options.fontSize = fontSize
  fitTerminal(termRef.current)
}, [fontSize])
```

**tab display:none/flex pattern** (App.tsx line 1492) — unchanged, no modification:
```tsx
style={{ display: isActive ? 'flex' : 'none' }}
```

---

### `frontend/src/components/SettingsTab.tsx` (component, request-response) — POL-02

**Analog:** itself — lines 438–461 (current two-button control)

**Imports to add** (modeled on GroupSidebar.tsx lines 8–10, heroicons pattern):
```tsx
import { SunIcon, MoonIcon } from '@heroicons/react/24/outline'
```
Note: `@heroicons/react` is already installed; `SunIcon`/`MoonIcon` are confirmed in `24/outline`. SessionCard.tsx and GroupSidebar.tsx both use this import pattern.

**Current control** (lines 438–461) — target to replace:
```tsx
{/* Appearance section (SETT-02) */}
<h3 id="settings-appearance">Appearance</h3>
<div className="settings-panel__field-group">
  <label className="settings-panel__label">Interface Theme</label>
  <div role="group" aria-label="Interface theme" style={{ display: 'flex', gap: '0.5rem' }}>
    <button type="button" className={`settings-panel__btn${uiTheme === 'light' ? ' settings-panel__btn--active' : ''}`}
      aria-pressed={uiTheme === 'light'} onClick={() => onUiThemeChange('light')}>Light</button>
    <button type="button" className={`settings-panel__btn${uiTheme === 'dark' ? ' settings-panel__btn--active' : ''}`}
      aria-pressed={uiTheme === 'dark'} onClick={() => onUiThemeChange('dark')}>Dark</button>
  </div>
  ...
</div>
```

**Replacement pattern** (D-06: `role="switch"` + icon + text, colorblind-safe):
```tsx
{/* Appearance section (SETT-02) — POL-02: single toggle switch */}
<h3 id="settings-appearance">Appearance</h3>
<div className="settings-panel__field-group">
  <label className="settings-panel__label">Interface Theme</label>
  <button
    type="button"
    role="switch"
    aria-checked={uiTheme === 'light'}
    aria-label={uiTheme === 'light' ? 'Light mode — click to switch to dark' : 'Dark mode — click to switch to light'}
    className={`settings-panel__theme-toggle${uiTheme === 'light' ? ' settings-panel__theme-toggle--light' : ''}`}
    onClick={() => onUiThemeChange(uiTheme === 'light' ? 'dark' : 'light')}
  >
    <span className="settings-panel__theme-toggle-track">
      <span className="settings-panel__theme-toggle-knob" aria-hidden="true">
        {uiTheme === 'light'
          ? <><SunIcon className="settings-panel__theme-toggle-icon" /><span>Light</span></>
          : <><MoonIcon className="settings-panel__theme-toggle-icon" /><span>Dark</span></>
        }
      </span>
    </span>
  </button>
  <p className="settings-panel__description" style={{ marginTop: '0.5rem' }}>
    Switches the whole app between light and dark appearance. Default is dark.
  </p>
</div>
```

**uiTheme prop wiring unchanged** (App.tsx lines 279–294 — do NOT touch):
```tsx
const [uiTheme, setUiTheme] = useState<'dark' | 'light'>(() =>
  localStorage.getItem(UI_THEME_STORAGE_KEY) === 'light' ? 'light' : 'dark'
)
useEffect(() => {
  if (uiTheme === 'light') {
    document.documentElement.setAttribute('data-ui-theme', 'light')
    document.documentElement.style.colorScheme = 'light'
  } else {
    document.documentElement.removeAttribute('data-ui-theme')
    document.documentElement.style.colorScheme = 'dark'
  }
}, [uiTheme])
const handleUiThemeChange = useCallback((t: 'dark' | 'light') => {
  localStorage.setItem(UI_THEME_STORAGE_KEY, t)
  setUiTheme(t)
}, [])
```

---

### `frontend/src/components/Hub/HubFilterBar.tsx` (component, request-response) — POL-03

**Analog:** `frontend/src/components/Hub/HubEmptyState.tsx` (same button restyle target)

**Imports to add:**
```tsx
import { PlusIcon } from '@heroicons/react/24/outline'
```

**Current button** (HubFilterBar.tsx lines 134–141):
```tsx
<button className="hub-filter__new-session" onClick={onNewSession} type="button">
  New session
</button>
```

**Replacement pattern** (comp sidebar affordance — minimal weight, accent `+` prefix):
```tsx
<button className="hub-filter__new-session" onClick={onNewSession} type="button">
  <PlusIcon className="hub-filter__new-session-icon" aria-hidden="true" />
  New session
</button>
```
The `onClick` wiring is unchanged. Only the button interior and CSS change.

---

### `frontend/src/components/Hub/HubEmptyState.tsx` (component, request-response) — POL-03

**Analog:** `frontend/src/components/Hub/HubFilterBar.tsx` (same button restyle pattern)

**Imports to add:**
```tsx
import { PlusIcon } from '@heroicons/react/24/outline'
```

**Current button** (HubEmptyState.tsx line 43):
```tsx
<button className="hub__empty-cta" onClick={onNewSession} type="button">
  New session
</button>
```

**Replacement pattern:**
```tsx
<button className="hub__empty-cta" onClick={onNewSession} type="button">
  <PlusIcon className="hub__empty-cta-icon" aria-hidden="true" />
  New session
</button>
```

---

### `frontend/src/components/Hub/SessionCard.tsx` (component, CRUD) — POL-01

**Analog:** itself — the TSX is unchanged for POL-01. The drag handle and menu-btn elements already have correct class names; only the CSS changes.

**Drag handle / menu-btn current positions** (SessionCard.tsx lines 338–358):
```tsx
<span className="hub-card__drag-handle" aria-label="Drag to reorder" aria-hidden="true">
  <Bars3Icon className="w-4 h-4" />
</span>
<button
  ref={menuBtnRef}
  type="button"
  className="hub-card__menu-btn"
  aria-label={`Card options for ${name}`}
  ...
>
  <EllipsisHorizontalIcon className="w-4 h-4" />
</button>
```
Both are `position: absolute; top: 8px` in style.css (lines 5038–5060). The CSS fix adds `padding-top` to `.hub-card` to prevent overlap — no TSX change.

---

### `frontend/src/style.css` (config/CSS) — all POL items

**Analog:** itself — follows the established `--hub-*` token + `[data-ui-theme="light"]` discipline.

**Token system location** (lines 4014–4080 dark, lines 4082–4149 light):
```css
/* Dark theme defaults in :root (line 4015) */
:root {
  --hub-accent: #7aa2f7;
  --hub-surface-elevated: #1c1e28;
  /* ... */
}

/* Light theme in [data-ui-theme="light"] (line 4083) */
[data-ui-theme="light"] {
  --hub-accent: #3d6fe8;
  --hub-surface-elevated: #ececf0;
  /* ... */
}
```
**Rule:** Every new CSS rule that references a color or background MUST have a corresponding `[data-ui-theme="light"]` override in the light block. Token-only references (e.g., `var(--hub-accent)`) adapt automatically — no override needed unless the token itself isn't already in the light block.

**Motion contract** (sidebar lines 328–343):
```css
@media (prefers-reduced-motion: no-preference) {
  .sidebar { transition: width 150ms ease; }
  .sidebar__toggle, .sidebar__item { transition: background-color 0.1s, color 0.1s; }
}
@media (prefers-reduced-motion: reduce) {
  .sidebar, .sidebar__toggle, .sidebar__item { transition: none; }
}
```
All new animated rules (toggle-knob slide, group-item hover) follow this exact two-block pattern.

**POL-01 CSS targets:**
- `.hub-card` — add `padding-top: 36px` (icon-gutter: 8px top + 20px icon-height + 8px gap)
- `.hub-card__drag-handle` — keep `top: 8px; left: 8px` (now in the gutter, not overlapping content)
- `.hub-card__menu-btn` — keep `top: 8px; right: 8px` (same)
- `.hub-card__preview` — change `height: 56px` → `height: 88px` (≈6 lines at 11px/1.3lh)

Current preview rule (line 5063):
```css
.hub-card__preview {
  height: 56px;   /* change to 88px */
  overflow: hidden;
  border-top: 1px solid var(--hub-preview-border);
  background: var(--hub-preview-bg);
  padding: 4px 8px;
}
```

**POL-02 CSS targets** (new toggle switch rules — follow `.settings-panel__btn` existing pattern):
```css
/* Dark theme default */
.settings-panel__theme-toggle {
  /* outer container — acts as switch track wrapper */
  background: transparent;
  border: none;
  padding: 0;
  cursor: pointer;
}

.settings-panel__theme-toggle-track {
  display: flex;
  align-items: center;
  width: 120px;
  height: 32px;
  border-radius: var(--hub-radius-pill);
  background: var(--hub-surface-elevated);
  border: 1px solid var(--hub-border);
  position: relative;
}

.settings-panel__theme-toggle-knob {
  position: absolute;
  left: 4px;   /* dark: knob on left */
  display: flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 6px;
  border-radius: var(--hub-radius-pill);
  background: var(--hub-surface);
  color: var(--hub-text-secondary);
  font-size: 12px;
}

/* Light: knob on right side */
[data-ui-theme="light"] .settings-panel__theme-toggle-knob {
  left: auto;
  right: 4px;
  color: var(--hub-text-secondary);
}

/* Motion contract */
@media (prefers-reduced-motion: no-preference) {
  .settings-panel__theme-toggle-knob {
    transition: left 150ms ease, right 150ms ease;
  }
}
@media (prefers-reduced-motion: reduce) {
  .settings-panel__theme-toggle-knob { transition: none; }
}
```

**POL-03 CSS targets** — restyle both buttons to minimal text-link affordance:
```css
/* New hub-filter__new-session — comp sidebar affordance */
.hub-filter__new-session {
  background: transparent;
  border: none;
  color: var(--hub-text-secondary);
  font-size: 12px;
  padding: 0 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  margin-left: auto;
}
.hub-filter__new-session:hover { color: var(--hub-text-primary); }
.hub-filter__new-session-icon { width: 14px; height: 14px; color: var(--hub-accent); }

/* hub__empty-cta — same affordance, centered */
.hub__empty-cta {
  background: transparent;
  border: none;
  color: var(--hub-text-secondary);
  font-size: 12px;
  padding: 0 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  height: 32px;
}
.hub__empty-cta:hover { color: var(--hub-text-primary); }
.hub__empty-cta-icon { width: 14px; height: 14px; color: var(--hub-accent); }
```
Light-theme overrides: `color: var(--hub-text-secondary)` and `var(--hub-accent)` already resolve correctly from the `[data-ui-theme="light"]` token block — no extra rule needed unless computed values need override.

**POL-05 CSS additions** — new `sidebar__group-*` rules, follow `.hub__group-sidebar-item` pattern (GroupSidebar.tsx CSS at style.css ~line 4697+):
```css
/* sidebar__group-list — nested under Hub item */
.sidebar__group-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.sidebar__group-item {
  /* mirrors .hub__group-sidebar-item visual weight */
}

.sidebar__group-item__btn {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 5px 8px 5px 48px;  /* 48px left = icon slot width, aligns under Hub label */
  border: none;
  background: transparent;
  color: var(--hub-text-muted);
  font-size: 12px;
  cursor: pointer;
}

.sidebar__group-item__btn:hover {
  background: var(--hub-sidebar-item-hover-bg);
  color: var(--hub-text-primary);
}

.sidebar__group-item--active .sidebar__group-item__btn {
  background: var(--hub-sidebar-item-active-bg);
  color: var(--hub-accent);
}

.sidebar__group-item--drag-over .sidebar__group-item__btn {
  border: 1px solid var(--hub-drag-over-border);
  background: var(--hub-drag-over-bg);
}
/* All tokens above have [data-ui-theme="light"] overrides already in the token block */
```

---

### `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx` (test, request-response) — POL-02

**Analog:** itself — update existing assertions

**Current DOM query pattern** (lines 168–170, 175–179):
```tsx
const buttons = container.querySelectorAll<HTMLButtonElement>('button[aria-pressed]')
const darkBtn = Array.from(buttons).find(b => b.textContent?.trim() === 'Dark')
```

**Replacement query pattern** (after POL-02, one element, `role="switch"`):
```tsx
const toggle = container.querySelector<HTMLButtonElement>('[role="switch"]')
// aria-checked reflects current uiTheme:
expect(toggle?.getAttribute('aria-checked')).toBe('true')   // uiTheme==='light'
expect(toggle?.getAttribute('aria-checked')).toBe('false')  // uiTheme==='dark'
toggle?.click()
expect(onUiThemeChange).toHaveBeenCalledWith('light')  // or 'dark', depending on initial state
```

**Source-inspection assertions to update** (lines 109–118):
```tsx
// OLD (failing after POL-02):
it('Appearance section contains aria-pressed attribute', () => {
  expect(raw).toContain('aria-pressed')
})
it('SettingsTab calls onUiThemeChange with light', () => {
  expect(raw).toContain("onUiThemeChange('light')")
})

// NEW:
it('Appearance section contains role="switch" toggle', () => {
  expect(raw).toContain('role="switch"')
})
it('Appearance section contains aria-checked attribute', () => {
  expect(raw).toContain('aria-checked')
})
// onUiThemeChange call assertions survive unchanged — the callback call
// "onUiThemeChange(uiTheme === 'light' ? 'dark' : 'light')" still contains both strings.
```

**`role="group"` assertion to remove** (line 215–219 — `role="group"` is gone after POL-02):
```tsx
// DELETE:
it('the Interface theme control has role=group aria-label', () => {
  const group = container.querySelector('[role="group"][aria-label="Interface theme"]')
  expect(group).not.toBeNull()
})
// REPLACE WITH:
it('the Interface theme control is a role=switch element', () => {
  const toggle = container.querySelector('[role="switch"]')
  expect(toggle).not.toBeNull()
})
```

**Vitest mock setup** (lines 15–68) — unchanged; all mocked Wails bindings are still needed.

---

## Shared Patterns

### `--hub-*` Token Convention
**Source:** `frontend/src/style.css` lines 4014–4080 (dark) + 4082–4149 (light)
**Apply to:** Every new CSS rule in all POL items

Dark defaults in `:root`:
- Accent: `--hub-accent: #7aa2f7` (LOCKED — do NOT override with raw hex in rules)
- Surfaces: `--hub-surface`, `--hub-surface-elevated`, `--hub-bg`
- Borders: `--hub-border`, `--hub-border-hover`, `--hub-drag-over-border`
- Text tiers: `--hub-text-primary`, `--hub-text-secondary`, `--hub-text-muted`
- Sidebar-specific: `--hub-sidebar-item-active-bg`, `--hub-sidebar-item-hover-bg`, `--hub-drag-over-bg`

Light counterparts in `[data-ui-theme="light"]` — same token names, different values.

**Rule:** No hardcoded hex values in `.hub-*`, `.hub__*`, or `.sidebar*` rules. Use tokens only.

### Motion Contract
**Source:** `frontend/src/style.css` lines 328–343
**Apply to:** Any new animated rule (toggle knob, group sub-list expand, hover transitions)

```css
@media (prefers-reduced-motion: no-preference) {
  /* animated rules here */
}
@media (prefers-reduced-motion: reduce) {
  /* matching static overrides here */
}
```

### Heroicons Import Pattern
**Source:** `frontend/src/components/Hub/GroupSidebar.tsx` lines 4–10; `Sidebar.tsx` lines 2–7
**Apply to:** Any new icon in SettingsTab.tsx (SunIcon/MoonIcon), HubFilterBar.tsx (PlusIcon), HubEmptyState.tsx (PlusIcon)

```tsx
import { IconName } from '@heroicons/react/24/outline'
// Then in TSX: <IconName aria-hidden="true" /> (CSS controls size, NO Tailwind w-*/h-* classes)
// Exception: existing SessionCard.tsx uses "w-4 h-4" Tailwind — do NOT propagate to new icons.
```

### localStorage Guard Pattern (try/catch)
**Source:** `frontend/src/components/Hub/HubPanel.tsx` lines 263–271
**Apply to:** Any new localStorage read in Sidebar.tsx or App.tsx

```tsx
const [value, setValue] = useState<boolean>(() => {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'true'
  } catch {
    return false  // SecurityError (private browsing) or QuotaExceededError
  }
})
```

### Colorblind-Safe State Pattern
**Source:** `frontend/src/components/Hub/GroupSidebar.tsx` lines 1–3 (comment header)
**Apply to:** All state-bearing UI (toggle knob, group active state, attention badges)

Rule: State MUST be conveyed by icon shape + text label as primary signals. Color is a reinforcement signal only. In practice:
- POL-02 toggle: `SunIcon`/`MoonIcon` + "Light"/"Dark" text label — not knob position or color alone
- POL-05 group active: `aria-pressed` + visual highlight — not color alone

### `useCallback` for Stable Event Handlers
**Source:** `frontend/src/components/Hub/GroupSidebar.tsx` lines 101–119; `App.tsx` line 291
**Apply to:** All new event handlers threaded as props (group state mutations in App.tsx, drag handlers in Sidebar.tsx)

```tsx
const handleX = useCallback((arg: Type) => {
  // mutation
}, [stableDeps])
```

---

## No Analog Found

All files in this phase have direct codebase analogs. No file requires patterns from RESEARCH.md alone.

---

## Files Deleted

| File | Reason | Logic destination |
|---|---|---|
| `frontend/src/components/Hub/GroupSidebar.tsx` | POL-05: entire component removed | Group sub-list → `Sidebar.tsx`; counts computation → `HubPanel.tsx` (via callback) or new `lib/hubGroupCounts.ts` |
| `frontend/src/components/Hub/__tests__/GroupSidebar.test.tsx` | deleted with GroupSidebar.tsx | Drag-drop + counts tests migrate to new `Sidebar.test.tsx` |

---

## Metadata

**Analog search scope:** `frontend/src/components/`, `frontend/src/lib/`, `frontend/src/style.css`, `frontend/src/App.tsx`
**Files read:** 14 source files + 1 test file
**Pattern extraction date:** 2026-06-21
