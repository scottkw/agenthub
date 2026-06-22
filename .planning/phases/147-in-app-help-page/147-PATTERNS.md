# Phase 147: In-App Help Page - Pattern Map

**Mapped:** 2026-06-22
**Files analyzed:** 11 (4 new components, 5 modified files, 2 new content dirs/files)
**Analogs found:** 10 / 11 (HelpContent.tsx has no direct analog — it is new work)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `frontend/src/components/HelpTab.tsx` | component (tab container) | request-response | `frontend/src/components/SettingsTab.tsx` | role-match |
| `frontend/src/components/HelpSearch.tsx` | component (search UI) | request-response | `frontend/src/components/SettingsSearch.tsx` | role-match |
| `frontend/src/components/HelpSectionNav.tsx` | component (section nav) | event-driven | `frontend/src/components/SettingsJumpBar.tsx` | role-match (with new IntersectionObserver work) |
| `frontend/src/components/HelpContent.tsx` | component (Markdown renderer) | transform | none | no analog |
| `frontend/src/components/Sidebar.tsx` | component (modified) | request-response | self (existing) | self-modification |
| `frontend/src/App.tsx` | component (modified) | request-response | self (existing) | self-modification |
| `frontend/src/style.css` | config (modified) | transform | self (existing `--hub-*` token declarations) | self-modification |
| `frontend/src/test-setup.ts` | config (modified) | transform | self (existing ResizeObserver polyfill) | self-modification |
| `frontend/src/components/__tests__/Sidebar.test.tsx` | test (modified) | request-response | self (existing Hub button test block) | self-modification |
| `frontend/src/components/__tests__/HelpTab.test.tsx` | test (new) | request-response | `frontend/src/components/__tests__/Sidebar.test.tsx` | role-match |
| `frontend/src/components/__tests__/HelpSearch.test.tsx` | test (new) | request-response | `frontend/src/components/__tests__/Sidebar.test.tsx` | role-match |
| `frontend/src/components/__tests__/HelpSectionNav.test.tsx` | test (new) | request-response | `frontend/src/components/__tests__/Sidebar.test.tsx` | role-match |
| `frontend/src/components/__tests__/HelpContent.test.tsx` | test (new) | request-response | `frontend/src/components/__tests__/Sidebar.test.tsx` | role-match |
| `frontend/src/content/help/*.md` | content (new) | transform | none | no analog (plain Markdown files) |

---

## Pattern Assignments

### `frontend/src/components/HelpTab.tsx` (tab container, request-response)

**Analog:** `frontend/src/components/SettingsTab.tsx`

**Imports pattern** (SettingsTab.tsx lines 1-37 — extract relevant subset):
```typescript
import React, { useState, useMemo, useCallback, useRef, useEffect } from 'react'
import { HelpSearch } from './HelpSearch'
import { HelpSectionNav } from './HelpSectionNav'
import { HelpContent } from './HelpContent'
import gettingStartedMd from '../content/help/getting-started.md?raw'
import faqMd from '../content/help/faq.md?raw'
```
Note: the `?raw` suffix is mandatory and confirmed working in this codebase (see `frontend/src/components/FindBar/__tests__/FindBar.visual.test.tsx` line 17).

**Top-level JSX structure** (SettingsTab.tsx line 377-383 — mirror this pattern):
```typescript
// SettingsTab.tsx lines 377-383
return (
  <div className="settings-tab">
    <div className="settings-panel__body">
      <SettingsJumpBar />
      <SettingsSearch />
      {/* sections follow */}
    </div>
  </div>
)

// HelpTab mirrors this as:
return (
  <div className="help-tab">
    <div className="help-tab__search">
      <HelpSearch query={query} results={results} onQueryChange={handleQueryChange} onJumpToSection={...} />
    </div>
    <div className="help-tab__layout">
      <HelpSectionNav activeSection={activeSection} onSectionChange={setActiveSection} contentPaneRef={contentPaneRef} />
      <div className="help-tab__content" ref={contentPaneRef}>
        <HelpContent markdown={allMarkdown} />
      </div>
    </div>
  </div>
)
```

**Debounce pattern** — copy from `App.tsx` trayDebounceRef pattern (confirmed established convention):
```typescript
// HelpTab.tsx — debounce state (copy this exact pattern)
const [query, setQuery] = useState('')
const [debouncedQuery, setDebouncedQuery] = useState('')
const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

const handleQueryChange = useCallback((raw: string) => {
  setQuery(raw)
  if (debounceRef.current) clearTimeout(debounceRef.current)
  debounceRef.current = setTimeout(() => setDebouncedQuery(raw), 200)
}, [])

// Cleanup on unmount
useEffect(() => () => { if (debounceRef.current) clearTimeout(debounceRef.current) }, [])
```

**Search index pattern** (useMemo with `[]` deps — new work, no analog):
```typescript
// SearchEntry type + useMemo index (build once, never rebuild)
interface SearchEntry {
  sectionId: string       // anchor id, e.g. 'help-faq'
  sectionLabel: string    // e.g. 'Frequently Asked Questions'
  text: string            // raw paragraph text (Markdown stripped)
}

const searchIndex = useMemo<ReadonlyArray<SearchEntry>>(() => {
  // ... split each md into paragraphs, strip Markdown, push entries
}, [])  // [] = build once; gettingStartedMd and faqMd are module-level constants

const results = useMemo(() => {
  const q = debouncedQuery.trim().toLowerCase()
  if (!q) return []
  return searchIndex.filter((e) => e.text.toLowerCase().includes(q))
}, [debouncedQuery, searchIndex])
```

---

### `frontend/src/components/HelpSearch.tsx` (search UI, request-response)

**Analog:** `frontend/src/components/SettingsSearch.tsx`

**Structural divergence from analog:** SettingsSearch (lines 40-104) is synchronous, matches section labels only, and uses no debounce. HelpSearch receives pre-filtered `results` as a prop (debounce lives in HelpTab) and renders snippet-level results with `<mark>` highlighting. Mirror the JSX skeleton and event handling, not the filtering logic.

**Search input pattern** (SettingsSearch.tsx lines 69-103 — diverge on `<label>` requirement from D-12):
```typescript
// SettingsSearch.tsx line 69-78 — uses aria-label only (no visible label)
<input
  type="search"
  className="settings-search__input"
  placeholder="Search settings…"
  value={query}
  onChange={(e) => setQuery(e.target.value)}
  aria-label="Search settings"
/>

// HelpSearch MUST add a visible <label> (D-12 — aria-label alone is insufficient):
<label htmlFor="help-search-input">Search help…</label>
<input
  id="help-search-input"
  type="search"
  className="help-search__input"
  placeholder="Search help…"
  value={query}
  onChange={(e) => onQueryChange(e.target.value)}
/>
<button
  type="button"
  className="help-search__clear"
  aria-label="Clear search"
  onClick={() => onQueryChange('')}
>
  ×
</button>
```

**Results list pattern** (SettingsSearch.tsx lines 79-101 — mirror listbox/option ARIA):
```typescript
// SettingsSearch.tsx lines 79-101 — exact list pattern to mirror
{results.length > 0 && (
  <ul className="settings-search__results" role="listbox">
    {results.map((r) => (
      <li
        key={`${r.target}:${r.label}`}
        className="settings-search__result"
        role="option"
        aria-selected="false"
        tabIndex={0}
        onClick={() => jumpTo(r.target)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); jumpTo(r.target) }
        }}
      >
        {r.label}
      </li>
    ))}
  </ul>
)}

// HelpSearch extends this with snippet + <mark> injection:
<ul className="help-search__results" role="listbox">
  {results.map((r, i) => (
    <li key={i} className="help-search__result" role="option" aria-selected="false" tabIndex={0}>
      <HighlightedSnippet text={extractSnippet(r.text, debouncedQuery)} query={debouncedQuery} />
      <button type="button" onClick={() => onJumpToSection(r.sectionId)}>Go to {r.sectionLabel} →</button>
    </li>
  ))}
</ul>
```

**jumpTo pattern** (SettingsSearch.tsx lines 49-67 — use scrollIntoView branch):
```typescript
// SettingsSearch.tsx lines 54-57 — the scrollIntoView path (copy this):
const el = document.getElementById(target)
if (el) {
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
```

**Empty state** — no analog in SettingsSearch; new work. Render when `debouncedQuery.trim() !== ''` AND `results.length === 0`.

**`<mark>` injection helper** — no analog; construct in HelpSearch:
```typescript
function HighlightedSnippet({ text, query }: { text: string; query: string }): React.ReactElement {
  // Plain-string split; pure React JSX — no dangerouslySetInnerHTML
  // Renders <mark className="help-search__mark"> around each match
  // See RESEARCH.md Pattern 6 for full implementation
}
```

---

### `frontend/src/components/HelpSectionNav.tsx` (section nav, event-driven)

**Analog:** `frontend/src/components/SettingsJumpBar.tsx`

**Analog gap:** SettingsJumpBar.tsx (all 47 lines) uses plain `<a href="#">` links with NO IntersectionObserver and NO active state. HelpSectionNav uses `<button>` elements (not `<a>`) for keyboard accessibility and adds IntersectionObserver scroll-spy. Mirror only the `<nav>` + section-list structure.

**Structure pattern** (SettingsJumpBar.tsx lines 31-46 — use nav/list/item pattern):
```typescript
// SettingsJumpBar.tsx lines 31-46
export function SettingsJumpBar(): React.ReactElement {
  return (
    <nav className="settings-jump-bar" aria-label="Settings sections">
      {SETTINGS_JUMP_LINKS.map((link) => (
        <a
          key={link.id}
          href={`#${link.id}`}
          className="settings-jump-bar__link"
          data-target={link.id}
        >
          {link.label}
        </a>
      ))}
    </nav>
  )
}

// HelpSectionNav replaces <a> with <button> and adds active state:
<nav className="help-tab__nav" aria-label="Help sections">
  <ul className="help-nav__list">
    {SECTIONS.map(({ id, label }) => (
      <li key={id} className="help-nav__item">
        <button
          type="button"
          className={`help-nav__link${activeSection === id ? ' help-nav__link--active' : ''}`}
          aria-current={activeSection === id ? 'true' : undefined}
          onClick={() => {
            document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
            onSectionChange(id)
          }}
        >
          {label}
        </button>
      </li>
    ))}
  </ul>
</nav>
```

**Section data array pattern** (SettingsJumpBar.tsx lines 21-29 — mirror exported SETTINGS_JUMP_LINKS):
```typescript
// SettingsJumpBar.tsx lines 21-29
export const SETTINGS_JUMP_LINKS: ReadonlyArray<JumpBarLink> = [
  { label: 'Plugins', id: 'settings-plugins' },
  // ...
]

// HelpSectionNav equivalent:
const SECTIONS = [
  { id: 'help-getting-started', label: 'Getting Started' },
  { id: 'help-faq', label: 'Frequently Asked Questions' },
] as const
```

**IntersectionObserver pattern** — new work, no analog. Key constraint: pass `root: contentPaneRef.current` (the scrollable div, NOT null/viewport). See RESEARCH.md Pattern 7 for full implementation. The `rootMargin: '-80px 0px -60% 0px'` clears the 80px sticky search bar; exact values require UAT tuning.

**Props interface:**
```typescript
interface HelpSectionNavProps {
  activeSection: string
  onSectionChange: (id: string) => void
  contentPaneRef: React.RefObject<HTMLDivElement>
}
```

---

### `frontend/src/components/HelpContent.tsx` (Markdown renderer, transform)

**Analog:** none — new work

**Import pattern:**
```typescript
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize'
import type { Schema } from 'hast-util-sanitize'
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'
import { ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline'
```
Note: `react-markdown` and `remark-gfm` are already installed. `rehype-sanitize` must be added with `cd frontend && pnpm add rehype-sanitize`.

**BrowserOpenURL pattern** (SettingsTab.tsx lines 25, 609-613 — exact copy):
```typescript
// SettingsTab.tsx line 609-613 — the external link pattern to copy exactly:
<button
  className="settings-web-server__action-btn"
  onClick={() => BrowserOpenURL(serverURL)}
  aria-label="Open dashboard in browser"
>
  <ArrowTopRightOnSquareIcon style={{ width: 14, height: 14 }} />
  Open
</button>

// HelpContent adapts this for Markdown <a> components:
a: ({ href, children }) => (
  <button
    type="button"
    className="help-content__external-link"
    onClick={() => href && BrowserOpenURL(href)}
    aria-label={`${children} (opens in browser)`}
  >
    {children}
    <ArrowTopRightOnSquareIcon style={{ width: 14, height: 14 }} aria-hidden="true" />
  </button>
),
```
Import path is `'../wailsjs/wailsjs/runtime/runtime'` (confirmed at SettingsTab.tsx line 25 — double `wailsjs/wailsjs` is intentional).

**rehype-sanitize schema** (extended to allow `<mark>` defensively):
```typescript
const sanitizeSchema: Schema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), 'mark'],
  attributes: {
    ...defaultSchema.attributes,
    mark: ['className'],
  },
}
```

**Component signature:**
```typescript
export function HelpContent({ markdown }: { markdown: string }): React.ReactElement
```

---

### `frontend/src/components/Sidebar.tsx` (modified)

**Self-modification.** Add `onOpenHelp` prop and Help button to the existing `sidebar__bottom` div.

**Existing `sidebar__bottom` pattern** (Sidebar.tsx lines 286-295 — the block to extend):
```typescript
// Sidebar.tsx lines 286-295 — current sidebar__bottom (1 item):
<div className="sidebar__bottom">
  <button
    className="sidebar__item"
    onClick={onSettings}
    aria-label="Settings"
  >
    <Cog6ToothIcon className="sidebar__icon" />
    {!collapsed && <span className="sidebar__label">Settings</span>}
  </button>
</div>

// After modification (Settings ABOVE Help per UI-SPEC §Layout Contract):
<div className="sidebar__bottom">
  <button
    className="sidebar__item"
    onClick={onSettings}
    aria-label="Settings"
  >
    <Cog6ToothIcon className="sidebar__icon" />
    {!collapsed && <span className="sidebar__label">Settings</span>}
  </button>
  <button
    className={`sidebar__item${activePanel === '__help__' ? ' sidebar__item--active' : ''}`}
    onClick={onOpenHelp}
    aria-label="Help"
  >
    <QuestionMarkCircleIcon className="sidebar__icon" />
    {!collapsed && <span className="sidebar__label">Help</span>}
  </button>
</div>
```

**Active state pattern** (Sidebar.tsx line 225 — exact pattern to copy for Help button):
```typescript
// Sidebar.tsx line 225 — Hub active state pattern (copy exactly for Help):
className={`sidebar__item${activePanel === '__hub__' ? ' sidebar__item--active' : ''}`}
```

**Props interface extension** (Sidebar.tsx lines 15-28 — add one prop):
```typescript
// Current interface (Sidebar.tsx lines 15-28) — add onOpenHelp at the end:
interface SidebarProps {
  onHome: () => void
  onSettings: () => void
  onOpenHub: () => void
  activePanel?: string
  // ... POL-05 group props ...
  onOpenHelp: () => void   // ADD THIS
}
```

**Icon import addition** (Sidebar.tsx lines 2-7 — add QuestionMarkCircleIcon):
```typescript
// Sidebar.tsx lines 2-7 — existing imports:
import {
  Bars3Icon,
  HomeIcon,
  Cog6ToothIcon,
  Squares2X2Icon,
} from '@heroicons/react/24/outline'

// Add QuestionMarkCircleIcon to this import block:
import {
  Bars3Icon,
  HomeIcon,
  Cog6ToothIcon,
  Squares2X2Icon,
  QuestionMarkCircleIcon,  // ADD
} from '@heroicons/react/24/outline'
```

---

### `frontend/src/App.tsx` (modified)

**Self-modification.** Add `HELP_TAB`, `handleOpenHelp`, render block, tab type exclusion, and Sidebar prop.

**Special-tab constant pattern** (App.tsx lines 95-98 — copy exactly):
```typescript
// App.tsx lines 95-98 — existing constants (add HELP_TAB alongside):
const WELCOME_TAB: Tab = { id: '__welcome__', name: 'Welcome', sessionId: '', cli: '', type: 'welcome' }
const SETTINGS_TAB: Tab = { id: '__settings__', name: 'Settings', sessionId: '', cli: '', type: 'settings' }
const HUB_TAB: Tab = { id: '__hub__', name: 'Hub', sessionId: '', cli: '', type: 'hub' }
// ADD:
const HELP_TAB: Tab = { id: '__help__', name: 'Help', sessionId: '', cli: '', type: 'help' }
```

**handleOpen pattern** (App.tsx lines 780-788 — copy handleOpenSettings exactly):
```typescript
// App.tsx lines 780-788 — handleOpenSettings (copy this pattern for handleOpenHelp):
const handleOpenSettings = useCallback(() => {
  const existing = tabs.find((t) => t.type === 'settings')
  if (existing) {
    setActiveId(existing.id)
    return
  }
  setTabs((prev) => [...prev, SETTINGS_TAB])
  setActiveId(SETTINGS_TAB.id)
}, [tabs])

// handleOpenHelp — identical pattern:
const handleOpenHelp = useCallback(() => {
  const existing = tabs.find((t) => t.type === 'help')
  if (existing) {
    setActiveId(existing.id)
    return
  }
  setTabs((prev) => [...prev, HELP_TAB])
  setActiveId(HELP_TAB.id)
}, [tabs])
```

**Display-toggle mount pattern** (App.tsx lines 1532-1556 — copy for HelpTab; DO NOT use conditional render):
```typescript
// App.tsx lines 1532-1556 — SettingsTab mounted with display:none toggle (copy for Help):
{mode !== 'web' && (
  <div style={{ display: activeId === SETTINGS_TAB.id ? 'flex' : 'none', flexDirection: 'column', height: '100%' }}>
    <SettingsTab ... />
  </div>
)}

// HelpTab — same pattern (keep mounted, use display none to preserve state):
{mode !== 'web' && (
  <div style={{ display: activeId === HELP_TAB.id ? 'flex' : 'none', flexDirection: 'column', height: '100%' }}>
    <HelpTab />
  </div>
)}
```
CRITICAL: Do NOT use `{activeId === HELP_TAB.id && <HelpTab />}` — this loses scroll position on tab switch.

**Tab type exclusion** (App.tsx line 1597 — add `'help'` to existing exclusion list):
```typescript
// App.tsx line 1597 — current exclusion list:
if (tab.type === 'welcome' || tab.type === 'settings' || tab.type === 'file-browser' || tab.type === 'hub') return null
// ADD 'help':
if (tab.type === 'welcome' || tab.type === 'settings' || tab.type === 'file-browser' || tab.type === 'hub' || tab.type === 'help') return null
```

**Sidebar wiring** (App.tsx lines 1373-1385 — add onOpenHelp):
```typescript
// App.tsx lines 1373-1385 — Sidebar usage; add onOpenHelp prop:
<Sidebar
  onHome={handleHome}
  onSettings={handleOpenSettings}
  onOpenHub={handleOpenHub}
  onOpenHelp={handleOpenHelp}   // ADD
  activePanel={activeId ?? undefined}
  // ... rest unchanged ...
/>
```

---

### `frontend/src/style.css` (modified)

**Self-modification.** Add `--hub-search-highlight-bg` token and `.help-*` CSS classes.

**Token declaration pattern** (style.css `:root` — add alongside existing `--hub-*` tokens):
```css
/* Phase 147 — Help page search highlight token */
:root {
  --hub-search-highlight-bg: rgba(122, 162, 247, 0.25);
}
[data-ui-theme="light"] {
  --hub-search-highlight-bg: rgba(61, 111, 232, 0.20);
}
```

**Layout container pattern** (style.css lines 553-559 — `.settings-tab` to mirror as `.help-tab`):
```css
/* style.css lines 553-559 — .settings-tab */
.settings-tab {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
/* Mirror: */
.help-tab {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
```

**Scrollable pane pattern** (style.css lines 561-601 — `.settings-panel__body` maps to `.help-tab__content`):
```css
/* style.css lines 561-601 — mirror key properties: */
.help-tab__content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 24px;        /* UI-SPEC: lg spacing */
  scroll-behavior: smooth;
  scrollbar-width: thin;
  scrollbar-color: var(--hub-scrollbar) transparent;
}
```

**Sticky bar pattern** (style.css lines 605-617 — `.settings-jump-bar` maps to `.help-tab__search`):
```css
/* style.css lines 605-617: */
.settings-jump-bar {
  position: sticky;
  top: 0;
  z-index: 5;
  padding: 10px 0;    /* NOTE: UI-SPEC corrects this to 8px 0 for .help-tab__search */
  background-color: var(--hub-bg);
  border-bottom: 1px solid var(--hub-border);
}
/* Help search sticky bar: */
.help-tab__search {
  position: sticky;
  top: 0;
  z-index: 5;
  padding: 8px 0;     /* UI-SPEC r1: 8px (corrected from 10px non-multiple-of-4) */
  background-color: var(--hub-surface);
  border-bottom: 1px solid var(--hub-border);
  margin-bottom: 16px;
}
```

**Anchor scroll-margin pattern** (style.css line 594 — `scroll-margin-top: 80px`):
```css
/* style.css line 594 — applied to section headings: */
scroll-margin-top: 80px;
/* Apply same value to .help-content__section headings */
```

**h3 styling pattern** (style.css lines 582-595 — `.settings-panel__body h3`):
```css
/* style.css lines 582-595 — exact pattern for h3 sub-sections: */
.settings-panel__body h3 {
  font-size: var(--hub-font-size-sm);
  font-weight: var(--hub-font-weight-emphasis);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--hub-text-muted);
  margin-bottom: 12px;
  margin-top: 24px;
  scroll-margin-top: 80px;
}
/* Mirror as .help-tab__content h3 */
```

**focus-visible pattern** (style.css lines 629-632 — copy for `.help-search__input:focus-visible`):
```css
/* style.css lines 629-632: */
.settings-jump-bar__link:focus-visible {
  outline: 2px solid var(--hub-accent);
  outline-offset: 2px;
}
/* Copy to all interactive .help-* elements */
```

**prefers-reduced-motion gate pattern** (style.css lines 1002-1015 — copy exact structure):
```css
/* style.css lines 1002-1015 — gate ALL transitions on no-preference: */
@media (prefers-reduced-motion: no-preference) {
  .settings-jump-bar__link,
  /* ... */
  {
    transition: background-color 0.15s ease, color 0.15s ease;
  }
}
@media (prefers-reduced-motion: reduce) {
  .settings-jump-bar__link,
  /* ... */
  {
    transition: none;
  }
}
/* Apply same gate to all .help-nav__link, .help-content__external-link hover transitions */
```

**mark element pattern** (new):
```css
.help-search__mark {
  background: var(--hub-search-highlight-bg);
  color: var(--hub-text-primary);
  border-radius: 2px;
  border: 1px solid var(--hub-accent);
}
```

---

### `frontend/src/test-setup.ts` (modified)

**Self-modification.** Add IntersectionObserver polyfill, following the existing ResizeObserver polyfill pattern at lines 12-22.

**Existing polyfill pattern** (test-setup.ts lines 12-22 — copy exactly, substituting IntersectionObserver):
```typescript
// test-setup.ts lines 12-22 — ResizeObserver polyfill pattern:
// Phase 139-02: ResizeObserver is not implemented in jsdom; provide a no-op
// polyfill so components that wire ResizeObserver (e.g. TabBar chevrons) mount
// without throwing. The polyfill is intentionally inert — behavioural tests
// use source-level checks (raw import) rather than DOM simulation.
if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}

// ADD IntersectionObserver polyfill (same pattern, Phase 147):
if (typeof globalThis.IntersectionObserver === 'undefined') {
  globalThis.IntersectionObserver = class IntersectionObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof IntersectionObserver
}
```
Note: `as unknown as typeof IntersectionObserver` is required because the stub does not implement the full interface (no constructor signature with callback + options).

---

### `frontend/src/components/__tests__/Sidebar.test.tsx` (modified)

**Self-modification.** Update two count assertions and add Help button test block.

**Assertions to change** (Sidebar.test.tsx lines 92-96 and 238-243):
```typescript
// Line 92-96 — change 3 → 4:
// OLD: it('renders exactly 3 sidebar__item buttons (Home, Hub, Settings)', () => {
// OLD:   expect(items.length).toBe(3)
// NEW:
it('renders exactly 4 sidebar__item buttons (Home, Hub, Settings, Help)', () => {
  ;({ container, root } = renderSidebar())
  const items = container.querySelectorAll('button.sidebar__item')
  expect(items.length).toBe(4)
})

// Line 238-243 — change 3 → 4:
// OLD: it('all 3 sidebar items remain in DOM when collapsed', () => {
// NEW:
it('all 4 sidebar items remain in DOM when collapsed', () => {
  // ...
  expect(items.length).toBe(4)
})
```

Also update `renderSidebar` helper to include `onOpenHelp: vi.fn()` in `defaultProps`.

**Hub button test block pattern** (Sidebar.test.tsx lines 263-308 — copy exactly as template for Help button tests):
```typescript
// Sidebar.test.tsx lines 263-308 — Hub button test block (copy structure for Help):
describe('Sidebar Hub item (HUB-01, Phase 131)', () => {
  it('renders a Hub button with aria-label="Hub"', () => { ... })
  it('Hub button fires onOpenHub when clicked', () => { ... })
  it('Hub button does NOT have sidebar__item--active when activePanel is not __hub__', () => { ... })
  it('Hub button has sidebar__item--active when activePanel === "__hub__"', () => { ... })
})

// Mirror as:
describe('Sidebar Help item (Phase 147)', () => {
  it('renders a Help button with aria-label="Help"', ...)
  it('Help button fires onOpenHelp when clicked', ...)
  it('Help button does NOT have sidebar__item--active when activePanel is not __help__', ...)
  it('Help button has sidebar__item--active when activePanel === "__help__"', ...)
})
```

---

### New test files (HelpTab.test.tsx, HelpSearch.test.tsx, HelpSectionNav.test.tsx, HelpContent.test.tsx)

**Analog:** `frontend/src/components/__tests__/Sidebar.test.tsx`

**Test file header pattern** (Sidebar.test.tsx lines 1-11 — copy for all new test files):
```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { ComponentUnderTest } from '../ComponentUnderTest'

// CSS source for contract/source-gate tests:
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')
// Component source for source-gate assertions:
const componentSrc = readFileSync(resolve(__dirname, '../HelpTab.tsx'), 'utf-8')
```

**Source-gate pattern** (Sidebar.test.tsx lines 643-650 — use readFileSync to assert string presence):
```typescript
// Sidebar.test.tsx lines 643-650 — CSS source-gate pattern:
it('POL-05 RED: style.css contains .sidebar__group-list rule', () => {
  expect(cssRaw).toContain('.sidebar__group-list')
})

// Mirror for Help tests — e.g. CSS token source gate:
it('style.css declares --hub-search-highlight-bg in :root', () => {
  expect(cssRaw).toContain('--hub-search-highlight-bg')
})
// Component source gate:
it('HelpContent.tsx imports BrowserOpenURL', () => {
  expect(helpContentSrc).toContain('BrowserOpenURL')
})
```

**Mock pattern for Wails imports** — Sidebar tests avoid mocking BrowserOpenURL (SettingsTab imports it, but Sidebar doesn't call it). HelpContent tests need a vi.mock:
```typescript
// Mock Wails runtime — check how other tests handle it:
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}))
```

**NAV-05 renderHelper pattern** (Sidebar.test.tsx lines 40-61 — copy renderHelper idiom):
```typescript
// Standard render helper — copy for each Help component:
function renderHelpSearch(overrides = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const defaultProps = { query: '', results: [], onQueryChange: vi.fn(), onJumpToSection: vi.fn() }
  act(() => { root.render(<HelpSearch {...defaultProps} {...overrides} />) })
  return { container, root }
}
```

---

### `frontend/src/content/help/*.md` (new content)

**No analog.** Plain Markdown files. Key constraint: import using `?raw` suffix (`import gettingStartedMd from '../content/help/getting-started.md?raw'`). File names to create:
- `getting-started.md` — Getting Started section
- `faq.md` — FAQ section (6 seed questions from UI-SPEC Copywriting Contract)

Keyboard Shortcuts section is omitted for v1 (no documented shortcuts found — RESEARCH.md A2).

---

## Shared Patterns

### BrowserOpenURL (External Links)
**Source:** `frontend/src/components/SettingsTab.tsx` lines 25, 609-613
**Apply to:** `HelpContent.tsx` (all Markdown `<a>` links), any explicit external link buttons
**Import path:** `'../wailsjs/wailsjs/runtime/runtime'` (double `wailsjs` is intentional — confirmed)
```typescript
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'
// Usage:
onClick={() => BrowserOpenURL(url)}
```
**Anti-pattern:** NEVER use `<a href="...">` for external links — this opens inside the Wails webview.

### Active Sidebar Item
**Source:** `frontend/src/components/Sidebar.tsx` line 225
**Apply to:** `Sidebar.tsx` Help button, all future sidebar items
```typescript
className={`sidebar__item${activePanel === '__hub__' ? ' sidebar__item--active' : ''}`}
```

### Tab Type Discriminant Union
**Source:** `frontend/src/components/TabBar.tsx` line 12
**Apply to:** `App.tsx` (HELP_TAB constant), TabBar.tsx (type union update)
```typescript
// TabBar.tsx line 12 — current union:
type?: 'terminal' | 'welcome' | 'settings' | 'file-browser' | 'hub'
// ADD 'help':
type?: 'terminal' | 'welcome' | 'settings' | 'file-browser' | 'hub' | 'help'
```
If this is not updated, `tsc --noEmit` will fail. Run `cd frontend && npx tsc --noEmit` after all files are written.

### Display-Toggle Mount Strategy
**Source:** `frontend/src/App.tsx` lines 1532-1556
**Apply to:** `App.tsx` HelpTab render block
```typescript
// Keep HelpTab mounted; use display:none to preserve state across tab switches:
<div style={{ display: activeId === HELP_TAB.id ? 'flex' : 'none', flexDirection: 'column', height: '100%' }}>
  <HelpTab />
</div>
```

### CSS Token Usage (No Hardcoded Hex)
**Source:** `frontend/src/style.css` (all existing `.hub-*` rules)
**Apply to:** All `.help-*` CSS classes
All new CSS must use `var(--hub-*)` tokens. No hardcoded hex values, no inline rgba. The only new token added in this phase is `--hub-search-highlight-bg`.

### prefers-reduced-motion Gate
**Source:** `frontend/src/style.css` lines 1002-1015 (and lines 257-264, 328-337, 454-459)
**Apply to:** All `.help-*` CSS transitions
```css
@media (prefers-reduced-motion: no-preference) {
  .help-nav__link { transition: background-color 0.15s ease, color 0.15s ease; }
}
@media (prefers-reduced-motion: reduce) {
  .help-nav__link { transition: none; }
}
```

### focus-visible Outline
**Source:** `frontend/src/style.css` lines 629-632
**Apply to:** `.help-search__input`, `.help-nav__link`, `.help-content__external-link`
```css
:focus-visible {
  outline: 2px solid var(--hub-accent);
  outline-offset: 2px;
}
```

### Test CSS Source-Gate Pattern
**Source:** `frontend/src/components/__tests__/Sidebar.test.tsx` lines 10-11
**Apply to:** All new test files that assert CSS or source-level properties
```typescript
import { readFileSync } from 'fs'
import { resolve } from 'path'
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')
```

---

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `frontend/src/components/HelpContent.tsx` | component | transform | No existing Markdown renderer component; `react-markdown` is in `package.json` but unused in any component today |
| `frontend/src/content/help/*.md` | content | transform | No existing bundled Markdown content in the project |

For `HelpContent.tsx`, use RESEARCH.md Pattern 4 (Markdown rendering with `react-markdown` + `rehype-sanitize`) and Pattern 8 (BrowserOpenURL for links) as the implementation guide.

---

## Critical Anti-Patterns (Do Not Repeat)

| Anti-Pattern | Why | Source of Truth |
|---|---|---|
| `<a href="...">` for external links | Opens in Wails webview, not system browser | SettingsTab.tsx line 609 shows correct `BrowserOpenURL` pattern |
| `{activeId === HELP_TAB.id && <HelpTab />}` conditional mount | Loses scroll position and activeSection state on tab switch | App.tsx lines 1532-1533 show correct display-toggle pattern |
| `import foo from '../content/help/file.md'` (no `?raw`) | Vite tries to parse `.md` as JS, throws module error | FindBar.visual.test.tsx line 17 shows correct `?raw` usage |
| IntersectionObserver with `root: null` (default) | Won't fire when content pane scrolls independently of viewport | RESEARCH.md Pitfall 4 |
| `dangerouslySetInnerHTML` for Markdown output | XSS risk; `react-markdown` virtual DOM avoids this entirely | RESEARCH.md Anti-Patterns section |
| Hardcoded hex colors in `.help-*` CSS rules | Breaks light/dark theme; fails colorblind-safe source check | UI-SPEC §Color |

---

## Metadata

**Analog search scope:** `frontend/src/components/`, `frontend/src/`, `frontend/src/components/__tests__/`
**Files read:** Sidebar.tsx, SettingsSearch.tsx, SettingsJumpBar.tsx, SettingsTab.tsx (imports + lines 375-413, 600-630), App.tsx (lines 90-119, 778-800, 1370-1385, 1527-1610), style.css (lines 553-632, 1002-1015), test-setup.ts, Sidebar.test.tsx, TabBar.tsx (grep)
**Pattern extraction date:** 2026-06-22
