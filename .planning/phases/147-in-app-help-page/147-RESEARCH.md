# Phase 147: In-App Help Page — Research

**Researched:** 2026-06-22
**Domain:** React 19 / Vite 8 / TypeScript frontend — in-app Help page with Markdown rendering, debounced full-text search, and IntersectionObserver-driven section nav
**Confidence:** HIGH (all key claims verified against installed code or npm registry)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Help is a 4th sidebar item — order: Home / Hub / Settings / Help.
- **D-02:** Help opens as a special tab with `id: '__help__'`, mirroring `__settings__` / `__hub__` / `__welcome__` in `App.tsx`.
- **D-03:** GUI only. CLI native `--help` satisfies CLI parity. Web-share viewers do not need the Help page.
- **D-04:** Issue #69's "TUI parity" section is obsolete and superseded. No TUI Help.
- **D-05:** Bundled Markdown files — committed in repo, rendered at runtime. No network fetch. Offline only.
- **D-06:** Maintainer-authored / hand-curated FAQ. Not auto-scraped from issues.
- **D-07:** Rich search — debounced live filtering over both doc and FAQ body content; matched terms highlighted in `<mark>` inside ~1–2 line context snippets; each snippet has a "jump to section" affordance.
- **D-08:** Empty state when no matches — message pointing to GitHub issues.
- **D-09:** Richer than `SettingsSearch` (which matches section labels only). Body-level snippet/highlight indexing is new work.
- **D-10:** Layout = left section-nav (~200px) + right content pane (flex:1). Search input sticky at top.
- **D-11:** Existing `--hub-*` token system only. No bespoke palette. Colorblind-safe + `prefers-reduced-motion`.
- **D-12:** Accessibility — visible/associated label on search; correct heading hierarchy; `aria-label` on external links.
- **D-13:** External links via `BrowserOpenURL` (Wails runtime), NOT `<a href>`.

### Claude's Discretion

- Markdown rendering approach (runtime renderer vs build-time), XSS-safe strategy for `<mark>` injection.
- Search index granularity (per-section / per-paragraph / per-heading-block) for snippet extraction.
- Exact icon for Help sidebar item (QuestionMarkCircleIcon from Heroicons outline is the default).
- Keyboard-shortcuts doc content — include only if shortcuts are actually documented.

### Deferred Ideas (OUT OF SCOPE)

- Remote/dynamic Help content (Option B).
- `agenthub help` content CLI command.
- Web-share-viewer Help surface.
- FAQ auto-sourced from closed GitHub issues.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| HELP-01 | An in-app Help page provides documentation, an FAQ, search, and external links (#69) | All research findings below directly enable this; see Standard Stack, Architecture Patterns, Code Examples |
</phase_requirements>

---

## Summary

Phase 147 delivers an in-app Help page inside the existing Wails desktop React frontend. The technical problem decomposes into five independent workstreams: (1) special-tab routing (`__help__`) identical to the established `__settings__` / `__hub__` pattern; (2) bundled Markdown content loaded via Vite's native `?raw` import (already proven in this codebase for `.tsx` files — works equally for `.md`); (3) Markdown rendering with `react-markdown` (already installed at 10.1.0) + `remark-gfm` (already installed at 4.0.1) — NO new packages required for rendering because `react-markdown` is XSS-safe by default for maintainer-authored content; (4) full-text search with per-paragraph indexing, 200ms debounce (mirrors existing `SettingsSearch`/`App.tsx` debounce pattern), and plain-string snippet extraction with manual `<mark>` injection; and (5) a sticky left section-nav with `scrollIntoView`-based anchor jumping (the existing `SettingsJumpBar` uses plain `<a href="#">` + CSS smooth-scroll, which is directly reusable but DOES NOT use IntersectionObserver — the active-section scroll-spy is new work).

The `rehype-sanitize` package listed in the UI-SPEC is NOT currently installed, but it is also NOT strictly required for this phase. The content is maintainer-authored and bundled at build time — not user input. `react-markdown` renders via a React virtual DOM (no `dangerouslySetInnerHTML`) and escapes raw HTML by default, which is sufficient for this trust model. `rehype-sanitize` is still RECOMMENDED as defense-in-depth but is an optional add; the planner should install it and apply the extended schema that allows `<mark>`. The `<mark>` injection for search highlights does NOT go through `react-markdown` at all — it occurs in the plain-text snippet list rendered by `HelpSearch`, using a string-split approach, not a hast plugin.

**Primary recommendation:** Use `react-markdown` (already installed) + `remark-gfm` (already installed) for the full-content pane. Use plain-string search with manual `<mark>` injection for the results-list snippets. Add `rehype-sanitize` (one new package) for defense-in-depth on the content pane. Total new npm installs: 1 package (`rehype-sanitize`).

---

## Project Constraints (from CLAUDE.md)

| Directive | Implication for This Phase |
|-----------|---------------------------|
| Package manager: `pnpm` preferred | Use `pnpm add` for new installs |
| TypeScript types + ESLint/Prettier | All new components must have full TS types; no `any` |
| `tsc && vite build` must pass | Test gate MUST run `tsc` not just vitest (see memory note) |
| Regression test convention (TESTING.md) | New test files MUST be registered in Suite Manifest §2, Traceability §4; run `bash tests/check-traceability-paths.sh` before commit |
| Colorblind user | Color-based UAT verified at source level (hex tokens in CSS), not by eye |
| Wails production: `-tags wailsassets` | Bundled content must work via embed.FS; Vite `?raw` import bundles strings into the JS output — confirmed compatible |
| `prefers-reduced-motion` | All `.help-*` CSS transitions gated with `@media (prefers-reduced-motion: no-preference)` |
| Heading hierarchy | h1 → h2 → h3, no skips |

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Navigation routing (`__help__` tab) | Frontend / App.tsx | — | Follows identical `__settings__` / `__hub__` special-tab pattern; pure React state |
| Markdown content (bundled) | Build tool (Vite `?raw`) | Frontend (React state) | `?raw` import bundles content at build time into the JS bundle → available in embed.FS offline |
| Markdown rendering | Frontend (`HelpContent.tsx`) | — | `react-markdown` renders in browser; no server involvement |
| Full-text search index | Frontend (`HelpTab.tsx` or `HelpSearch.tsx`) | — | Index is built from bundled strings at module load; no API call |
| Search highlight (`<mark>`) in results | Frontend (`HelpSearch.tsx`) | — | Plain-string split/wrap in the snippet list; NOT a rehype plugin |
| Section nav active state | Frontend (`HelpSectionNav.tsx`) | — | IntersectionObserver on section headings; new work (SettingsJumpBar uses pure CSS, no observer) |
| External links | Frontend (`HelpContent.tsx` / `HelpTab.tsx`) | Wails runtime bridge | `BrowserOpenURL` from Wails runtime — identical to SettingsTab.tsx:609 |
| CSS theming | `style.css` (`--hub-*` tokens) | — | One new token `--hub-search-highlight-bg`; both `:root` and `[data-ui-theme="light"]` |

---

## Standard Stack

### Core (already installed — ZERO new packages for core rendering)

| Library | Version | Purpose | Status |
|---------|---------|---------|--------|
| `react-markdown` | 10.1.0 | Markdown → React elements (XSS-safe virtual DOM) | [VERIFIED: pnpm-lock.yaml] — already in `dependencies` |
| `remark-gfm` | 4.0.1 | GitHub Flavored Markdown (tables, strikethrough, task lists) | [VERIFIED: pnpm-lock.yaml] — already in `dependencies` |
| `@heroicons/react` | ^2.2.0 | `QuestionMarkCircleIcon` for Help sidebar item | [VERIFIED: pnpm-lock.yaml] — already in `dependencies` |

### Supporting (one new package required)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `rehype-sanitize` | 6.0.0 | Defense-in-depth sanitization on rendered Markdown | Add to `dependencies`; apply extended schema that allows `<mark>` and `className` attrs |

**Version verification:** `npm view rehype-sanitize version` → `6.0.0`, published 2017-02-23 (8+ years old), maintained by `wooorm` (Titus Wormer, core unified/remark/rehype maintainer). [VERIFIED: npm registry]

### Installation (one command)

```bash
cd frontend && pnpm add rehype-sanitize
```

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `react-markdown` + `remark-gfm` | `marked` | `react-markdown` already installed; `marked` needs `dangerouslySetInnerHTML` |
| Plain-string `<mark>` injection | Custom rehype plugin | rehype plugin is overkill for snippet list which is plain text, not rendered MD; plugin adds complexity without benefit since the content pane does NOT need `<mark>` injection |
| `rehype-sanitize` | None (rely on `react-markdown` default safety) | Both are safe for maintainer-authored bundled content; `rehype-sanitize` adds defense-in-depth if a rehype plugin is ever added later |

---

## Package Legitimacy Audit

> `slopcheck` was unavailable at research time. All packages below are tagged `[ASSUMED]` for provenance and the planner must gate the install behind a `checkpoint:human-verify` task (which is trivially satisfied given the evidence below).

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| `react-markdown` | npm | 8+ yrs | Very high | github.com/remarkjs/react-markdown | [ASSUMED] | Approved — already installed in project |
| `remark-gfm` | npm | 4+ yrs | Very high | github.com/remarkjs/remark-gfm | [ASSUMED] | Approved — already installed in project |
| `rehype-sanitize` | npm | 8+ yrs | High | github.com/rehypejs/rehype-sanitize | [ASSUMED] | Approved — official unified ecosystem package, maintained by Titus Wormer (`wooorm`), same org as `remark-gfm`; 0 postinstall scripts [VERIFIED: npm view] |

**Packages removed due to slopcheck [SLOP] verdict:** none

**Packages flagged as suspicious [SUS]:** none — all three are core unified ecosystem packages from the `remarkjs`/`rehypejs` GitHub organizations.

*Since slopcheck was unavailable, the planner should add a `checkpoint:human-verify` before the `pnpm add rehype-sanitize` step, with the verification note: "confirm package is from rehypejs org at github.com/rehypejs/rehype-sanitize."*

---

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────────────────┐
                         │  App.tsx — tab state                     │
                         │  HELP_TAB = { id: '__help__', type: 'help' } │
                         │  handleOpenHelp → find-or-add HELP_TAB   │
                         └──────────────┬──────────────────────────┘
                                        │ renders when activeId === '__help__'
                                        ▼
                         ┌─────────────────────────────────────────┐
                         │  HelpTab.tsx                             │
                         │  - owns: query state, activeSection state │
                         │  - debounces: 200ms useRef timer         │
                         │  - builds: searchIndex (at module load)  │
                         └──┬──────────────────┬───────────────────┘
                            │                  │
               ┌────────────▼──────┐   ┌───────▼──────────────────────┐
               │ HelpSearch.tsx    │   │ HelpSectionNav.tsx            │
               │ - search input    │   │ - left 200px panel            │
               │ - result snippets │   │ - section anchor links        │
               │ - <mark> in       │   │ - IntersectionObserver        │
               │   plain-text      │   │   active state on scroll      │
               │   snippets        │   │ - aria-current on active item │
               └────────────┬──────┘   └───────────────────────────────┘
                            │
               ┌────────────▼──────────────────────────────────────────┐
               │ HelpContent.tsx                                        │
               │ - <Markdown> (react-markdown) + rehype-sanitize       │
               │ - components prop: code → <code className="help__kbd"> │
               │ - section anchors with id attrs for IntersectionObserver │
               │ - external link buttons → BrowserOpenURL(url)         │
               └────────────────────────────────────────────────────────┘
                            ▲
               ┌────────────┴──────────────────────────────────────────┐
               │ frontend/src/content/help/*.md (bundled via ?raw)      │
               │ - getting-started.md                                   │
               │ - faq.md                                               │
               │ - external-links.md (or inline in HelpContent)        │
               └────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
frontend/src/
├── content/
│   └── help/
│       ├── getting-started.md     # Getting Started section content
│       ├── faq.md                 # FAQ content (6 seed questions)
│       └── keyboard-shortcuts.md  # Omit section if no shortcuts to document
├── components/
│   ├── HelpTab.tsx                # Top-level: owns query + activeSection state
│   ├── HelpSearch.tsx             # Search input + snippet results with <mark>
│   ├── HelpSectionNav.tsx         # Left sticky section-nav; IntersectionObserver
│   └── HelpContent.tsx            # Full Markdown renderer via react-markdown
└── components/__tests__/
    ├── HelpTab.test.tsx            # Sidebar wiring, HELP_TAB constant, handleOpenHelp
    ├── HelpSearch.test.tsx         # Debounce, snippet extraction, <mark> injection
    ├── HelpSectionNav.test.tsx     # Section-nav rendering, aria-current, anchor links
    └── HelpContent.test.tsx        # react-markdown renders headings, BrowserOpenURL called
```

### Pattern 1: Special-Tab Routing in App.tsx

The `__help__` tab follows the IDENTICAL pattern as `__settings__` and `__hub__`. [VERIFIED: reading App.tsx lines 780-788, 1200-1210]

```typescript
// In App.tsx — add alongside WELCOME_TAB, SETTINGS_TAB, HUB_TAB
const HELP_TAB: Tab = { id: '__help__', name: 'Help', sessionId: '', cli: '', type: 'help' }

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

Render inside the `terminal-container` div, mirroring SettingsTab's display-toggle pattern:

```typescript
{mode !== 'web' && (
  <div style={{ display: activeId === HELP_TAB.id ? 'flex' : 'none', flexDirection: 'column', height: '100%' }}>
    <HelpTab />
  </div>
)}
```

> NOTE: The SettingsTab uses `display: 'flex' / 'none'` to keep it mounted (so settings state survives tab switching). Do the same for HelpTab — do NOT use a simple `{activeId === HELP_TAB.id && <HelpTab />}` conditional or scrollposition/activeSection state is lost on tab switch.

Also add `'help'` to the tab filter that skips terminal rendering:
```typescript
// Existing line 1597 — add 'help' to the exclusion list
if (tab.type === 'welcome' || tab.type === 'settings' || tab.type === 'file-browser' || tab.type === 'hub' || tab.type === 'help') return null
```

### Pattern 2: Sidebar 4th Item (Help)

[VERIFIED: reading Sidebar.tsx — current structure confirmed]

Add `onOpenHelp` prop and Help button to `sidebar__bottom` div, BELOW Settings (Settings is accessed more often per UI-SPEC):

```typescript
// Sidebar.tsx — new prop
interface SidebarProps {
  // ... existing props ...
  onOpenHelp: () => void   // new
}

// Inside the sidebar__bottom div — Settings first, Help second:
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

Import `QuestionMarkCircleIcon` from `@heroicons/react/24/outline` (already installed).

Wire in App.tsx: `<Sidebar ... onOpenHelp={handleOpenHelp} />`.

### Pattern 3: Bundled Markdown via Vite `?raw` Import

[VERIFIED: existing codebase uses `?raw` imports in `FindBar/__tests__/FindBar.visual.test.tsx` line 17 and multiple test files — the Vite + Vitest config supports `?raw` for any text file natively, including `.md` files. The `vite.config.ts` uses `base: './'` and `@vitejs/plugin-react` — no additional configuration needed for `?raw` imports.]

```typescript
// HelpContent.tsx or HelpTab.tsx
import gettingStartedMd from '../content/help/getting-started.md?raw'
import faqMd from '../content/help/faq.md?raw'
// Each import is a string constant bundled into the JS output at build time.
// Works correctly with -tags wailsassets (embed.FS) because Vite inlines
// the string into the JS bundle — no separate network fetch at runtime.
```

**Production build verification:** The `?raw` suffix causes Vite to inline the file as a JavaScript string literal. With `-tags wailsassets`, the Go build embeds the entire `frontend/dist/` tree via embed.FS. The inlined strings are part of `dist/assets/index-*.js` — no separate `.md` file requests, no network calls, fully offline. [CITED: Vite docs — `?raw` suffix, bundled asset handling]

**Vitest note:** The `?raw` import is natively supported in Vitest too (same Vite transform pipeline). Tests that import `.md?raw` files will receive the string content directly. [VERIFIED: existing test pattern in `FindBar/__tests__/FindBar.visual.test.tsx`]

### Pattern 4: Markdown Rendering with XSS-Safe rehype-sanitize

```typescript
// HelpContent.tsx
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize'
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'
import { ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline'

// Extended schema: allow <mark> with className (for .help-search__mark)
const sanitizeSchema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), 'mark'],
  attributes: {
    ...defaultSchema.attributes,
    mark: ['className'],
  },
}

// NOTE: For the CONTENT PANE (HelpContent), <mark> injection is NOT needed.
// The <mark> elements only appear in HelpSearch snippet results (plain-text path).
// So in practice, rehype-sanitize without schema extension is sufficient for HelpContent.
// The extended schema is included defensively in case future content uses <mark>.

export function HelpContent({ markdown }: { markdown: string }): React.ReactElement {
  return (
    <Markdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[[rehypeSanitize, sanitizeSchema]]}
      components={{
        // Render inline code / keyboard shortcuts with monospace style
        code: ({ children, ...props }) => (
          <code className="help__kbd" {...props}>{children}</code>
        ),
        // External links — all <a> in Markdown become BrowserOpenURL buttons
        a: ({ href, children }) => (
          <button
            type="button"
            className="help-content__external-link"
            onClick={() => href && BrowserOpenURL(href)}
            aria-label={`${children} (opens in browser)`}
          >
            {children}
            <ArrowTopRightOnSquareIcon
              style={{ width: 14, height: 14 }}
              aria-hidden="true"
            />
          </button>
        ),
      }}
    >
      {markdown}
    </Markdown>
  )
}
```

### Pattern 5: Search Index + Snippet Extraction (Per-Paragraph Granularity)

**Recommendation:** Index at **per-paragraph granularity** — each entry is one paragraph or FAQ answer of 1–3 sentences. Per-section is too coarse (snippets overflow the ~2-line context target). Per-word is unnecessary overhead. A FAQ with 6 questions has 6 entries; Getting Started has ~10–15 paragraph entries. Total index size: ~25–40 entries. [ASSUMED — based on content estimate from UI-SPEC FAQ seed set]

**Build the index at module load time** (not on every render), using `useMemo` with `[]` deps:

```typescript
// HelpTab.tsx
interface SearchEntry {
  sectionId: string       // anchor id, e.g. 'help-faq'
  sectionLabel: string    // e.g. 'Frequently Asked Questions'
  text: string            // raw paragraph text (Markdown stripped)
}

// Strip Markdown syntax for plain-text indexing (headings, bold, backticks, links)
function stripMd(md: string): string {
  return md
    .replace(/^#{1,6}\s+/gm, '')          // headings
    .replace(/\*\*([^*]+)\*\*/g, '$1')    // bold
    .replace(/\*([^*]+)\*/g, '$1')        // italic
    .replace(/`([^`]+)`/g, '$1')          // inline code
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1') // links
    .replace(/^\s*[-*+]\s+/gm, '')        // list bullets
    .trim()
}

// Build search index once on mount
const searchIndex = useMemo<ReadonlyArray<SearchEntry>>(() => {
  const entries: SearchEntry[] = []
  // Split each Markdown file into paragraphs by double-newline
  const sections = [
    { id: 'help-getting-started', label: 'Getting Started', md: gettingStartedMd },
    { id: 'help-faq', label: 'Frequently Asked Questions', md: faqMd },
  ]
  for (const section of sections) {
    const paragraphs = section.md.split(/\n\n+/).filter(Boolean)
    for (const para of paragraphs) {
      const text = stripMd(para)
      if (text.length > 20) {  // skip very short paragraphs (headings, etc.)
        entries.push({ sectionId: section.id, sectionLabel: section.label, text })
      }
    }
  }
  return entries
}, [])
```

**Debounce pattern** (200ms, matches D-07 and the existing App.tsx debounce with `useRef`):

```typescript
// HelpTab.tsx
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

// Filter results
const results = useMemo(() => {
  const q = debouncedQuery.trim().toLowerCase()
  if (!q) return []
  return searchIndex.filter((e) => e.text.toLowerCase().includes(q))
}, [debouncedQuery, searchIndex])
```

### Pattern 6: `<mark>` Injection in Snippet Results (XSS-Safe, Plain-Text Path)

**Decision: Option B — pre-render snippet extraction with manual `<mark>` injection in the results list.** The full content pane (`HelpContent`) does NOT inject `<mark>` — it renders the full section cleanly. Only the `HelpSearch` snippet list injects highlights.

This is the correct split: the UI-SPEC §"Search behavior" says the results list shows "~1–2 lines of surrounding context with matched term wrapped in `<mark>`" and the content pane shows the full rendered Markdown for the active section.

**Why Option B over Option A (rehype plugin):**
- Option A (rehype plugin walking hast text nodes) requires `rehype-raw` + complex node-walker for the content pane, adding ~60kb bundle and significant complexity.
- The content pane does NOT need `<mark>` highlights — the user jumped to the section, full content is shown.
- The snippet list is plain text (not rendered Markdown), so a simple string-split approach is both simpler and XSS-safe without any sanitization concern.

**Minimal code sketch — XSS-safe string highlight:**

```typescript
// HelpSearch.tsx
function HighlightedSnippet({ text, query }: { text: string; query: string }): React.ReactElement {
  if (!query) return <span>{text}</span>
  const lowerText = text.toLowerCase()
  const lowerQuery = query.toLowerCase()
  const parts: React.ReactNode[] = []
  let lastIndex = 0
  let idx = lowerText.indexOf(lowerQuery)
  while (idx !== -1) {
    if (idx > lastIndex) {
      parts.push(<span key={`t-${lastIndex}`}>{text.slice(lastIndex, idx)}</span>)
    }
    parts.push(
      <mark key={`m-${idx}`} className="help-search__mark">
        {text.slice(idx, idx + query.length)}
      </mark>
    )
    lastIndex = idx + query.length
    idx = lowerText.indexOf(lowerQuery, lastIndex)
  }
  if (lastIndex < text.length) {
    parts.push(<span key={`t-${lastIndex}`}>{text.slice(lastIndex)}</span>)
  }
  return <>{parts}</>
}

// Snippet extraction (~120 chars centred on first match)
function extractSnippet(text: string, query: string): string {
  const idx = text.toLowerCase().indexOf(query.toLowerCase())
  if (idx === -1) return text.slice(0, 120)
  const start = Math.max(0, idx - 40)
  const end = Math.min(text.length, idx + query.length + 80)
  const snippet = text.slice(start, end)
  return (start > 0 ? '…' : '') + snippet + (end < text.length ? '…' : '')
}
```

**XSS safety:** This approach creates React elements via JSX — React escapes all string values automatically. No `dangerouslySetInnerHTML` is used. No sanitization library is needed for the snippet list. [VERIFIED: React docs — JSX string values are escaped]

### Pattern 7: IntersectionObserver Active Section (New Work)

The existing `SettingsJumpBar` uses plain `<a href="#">` + CSS `scroll-behavior: smooth` on `.settings-panel__body`. It does NOT use IntersectionObserver. The active-section scroll-spy for the Help section-nav is new work. [VERIFIED: reading SettingsJumpBar.tsx — no IntersectionObserver present]

```typescript
// HelpSectionNav.tsx — IntersectionObserver active-section detection
const SECTIONS = [
  { id: 'help-getting-started', label: 'Getting Started' },
  { id: 'help-faq', label: 'Frequently Asked Questions' },
  // Conditionally include keyboard shortcuts and external links
]

interface HelpSectionNavProps {
  activeSection: string
  onSectionChange: (id: string) => void
  contentPaneRef: React.RefObject<HTMLDivElement>
}

export function HelpSectionNav({ activeSection, onSectionChange, contentPaneRef }: HelpSectionNavProps): React.ReactElement {
  useEffect(() => {
    const root = contentPaneRef.current
    if (!root) return

    // rootMargin: slightly negative top so the section heading must be "past"
    // the sticky search bar (80px) before it fires as intersecting.
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            onSectionChange(entry.target.id)
          }
        }
      },
      {
        root,                           // scope to the scrollable content pane
        rootMargin: '-80px 0px -60% 0px', // heading must be in top 40% of pane
        threshold: 0,
      }
    )

    SECTIONS.forEach(({ id }) => {
      const el = document.getElementById(id)
      if (el) observer.observe(el)
    })

    return () => observer.disconnect()
  }, [contentPaneRef, onSectionChange])

  return (
    <nav className="help-tab__nav" aria-label="Help sections">
      <ul className="help-nav__list">
        {SECTIONS.map(({ id, label }) => (
          <li key={id} className="help-nav__item">
            <button
              type="button"
              className={`help-nav__link${activeSection === id ? ' help-nav__link--active' : ''}`}
              aria-current={activeSection === id ? 'true' : undefined}
              onClick={() => {
                const el = document.getElementById(id)
                el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
                onSectionChange(id)
              }}
            >
              {label}
            </button>
          </li>
        ))}
      </ul>
    </nav>
  )
}
```

**IntersectionObserver pitfall:** The `root` must be the scrollable content pane `<div>`, not `document` (the default). Without `root`, IntersectionObserver fires against the viewport, but the content pane has `overflow-y: auto` — scroll events inside it won't change the viewport intersection. Always pass `root: contentPaneRef.current`. [ASSUMED — standard IntersectionObserver scoping requirement; applies universally]

**Vitest polyfill needed:** jsdom does not implement IntersectionObserver. Add to `frontend/src/test-setup.ts` (alongside the existing `ResizeObserver` polyfill at line 14):

```typescript
if (typeof globalThis.IntersectionObserver === 'undefined') {
  globalThis.IntersectionObserver = class IntersectionObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof IntersectionObserver
}
```

### Pattern 8: BrowserOpenURL External Links

[VERIFIED: reading SettingsTab.tsx line 609 — exact pattern]

```typescript
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'
import { ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline'

// External links section (dedicated section in content OR rendered via components prop)
<button
  type="button"
  className="help-content__external-link"
  onClick={() => BrowserOpenURL('https://github.com/scottkw/agenthub')}
  aria-label="View AgentHub on GitHub (opens in browser)"
>
  View on GitHub
  <ArrowTopRightOnSquareIcon style={{ width: 14, height: 14 }} aria-hidden="true" />
</button>
```

### Pattern 9: CSS Token for Search Highlight

Add to `style.css` alongside existing `--hub-*` declarations:

```css
/* Phase 147 — Help page search highlight token */
:root {
  --hub-search-highlight-bg: rgba(122, 162, 247, 0.25);
}
[data-ui-theme="light"] {
  --hub-search-highlight-bg: rgba(61, 111, 232, 0.20);
}
```

Apply in CSS:
```css
.help-search__mark {
  background: var(--hub-search-highlight-bg);
  color: var(--hub-text-primary);
  border-radius: 2px;
  border: 1px solid var(--hub-accent);
}
```

### Anti-Patterns to Avoid

- **Using `dangerouslySetInnerHTML` for Markdown**: `react-markdown` + `rehype-sanitize` avoids this entirely. Never use `innerHTML` or `dangerouslySetInnerHTML` for Markdown output.
- **Injecting `<mark>` via rehype plugin into the content pane**: Unnecessary. The content pane shows the full section — no highlight needed there. Only the snippet list needs `<mark>`.
- **Conditionally mounting HelpTab** (`{activeId === HELP_TAB.id && <HelpTab />}`): This loses scroll position and active-section state when the user switches tabs. Use `display: none` like SettingsTab does.
- **Using `<a href>` for external links**: Opens inside the Wails webview. Always use `BrowserOpenURL` button.
- **IntersectionObserver with default `root`** (null = viewport): Won't fire when the content pane scrolls independently. Always pass `root: contentPaneRef.current`.
- **Skipping the `?raw` suffix**: `import gettingStartedMd from '../content/help/getting-started.md'` without `?raw` will fail — Vite needs the suffix to treat the file as a raw string.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Markdown → React elements | Custom parser / regex-to-JSX | `react-markdown` (already installed) | Handles CommonMark spec correctly; XSS-safe virtual DOM; handles edge cases in GFM tables, code blocks, nested formatting |
| XSS sanitization | Custom allowlist filter | `rehype-sanitize` with `defaultSchema` | OWASP XSS allowlists are complex to get right; `hast-util-sanitize` has 8 years of security hardening |
| Debounce | `setInterval` loop or `Date.now()` comparison | `useRef` + `setTimeout` (established pattern in App.tsx) | Already used in App.tsx trayDebounceRef — copy the pattern exactly |

**Key insight:** The full rendering pipeline (`react-markdown` → `remark-gfm` → `rehype-sanitize` → React elements) handles all CommonMark + GFM edge cases that would take hundreds of lines to replicate correctly. The search is deliberately simple (plain substring matching) — do not over-engineer it into Fuse.js or Lunr.

---

## Common Pitfalls

### Pitfall 1: Sidebar test count assertion will break

**What goes wrong:** `Sidebar.test.tsx` line 95 asserts `items.length === 3` (the Phase 138 / NAV-05 invariant). Adding a 4th item to the sidebar (Help) changes this count to 4. The test will fail after the sidebar change.

**Why it happens:** D-01 explicitly reopens the NAV-05 decision — the sidebar is now 4 items. The test is asserting the old invariant.

**How to avoid:** Update `Sidebar.test.tsx` to assert `items.length === 4` and update the describe-block comment. Also update the line 238 assertion ("all 3 sidebar items remain in DOM when collapsed") to expect 4. These tests are REGRESSION tests protecting the new state — update them as part of the phase, not workarounds.

**Warning signs:** `vitest` shows `expected 3 to equal 4` in the Sidebar test.

### Pitfall 2: rehype-sanitize schema strips `<mark>` by default

**What goes wrong:** `rehype-sanitize` with `defaultSchema` does not include `mark` in `tagNames` — it strips `<mark>` elements from the output, converting them to their text content.

**Why it happens:** The GFM `defaultSchema` is conservative (it mirrors GitHub's HTML sanitizer policy). `<mark>` is not a standard GFM output element.

**How to avoid:** Use the extended schema shown in Pattern 4 above:
```typescript
const sanitizeSchema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), 'mark'],
  attributes: { ...defaultSchema.attributes, mark: ['className'] },
}
```

**Note:** This only matters if `<mark>` is ever injected via a rehype plugin before sanitize. In the current plan (plain-text `<mark>` injection in snippet list only), this pitfall does NOT apply to `HelpContent` — but the extended schema should be applied defensively.

### Pitfall 3: `?raw` import fails without `?raw` suffix (runtime, not build-time error)

**What goes wrong:** Vite tries to treat the `.md` file as a JS module (it has no JS exports) and throws a module parse error. The error surfaces at runtime in dev, or at build time with a cryptic Rollup error.

**Why it happens:** Vite's default behavior for unknown file types is to try to parse them as JavaScript. Without the `?raw` suffix, `.md` files are not handled.

**How to avoid:** Always write `import foo from '../content/help/file.md?raw'`. The `?raw` suffix is mandatory. [VERIFIED: Vite docs pattern; existing codebase uses `?raw` in 8+ places without any additional config]

### Pitfall 4: HelpSectionNav IntersectionObserver misfires when `root` is null

**What goes wrong:** Active section indicator never changes on scroll, or always shows the last section.

**Why it happens:** Default IntersectionObserver `root` is the viewport. The content pane (`overflow-y: auto`) scrolls independently — its children never cross viewport boundaries unless the container itself is larger than the viewport.

**How to avoid:** Pass `{ root: contentPaneRef.current }` to the IntersectionObserver constructor. The `contentPaneRef` must point to the div with `overflow-y: auto`. Pass it down from `HelpTab` via a `React.RefObject`.

### Pitfall 5: `tsc` rejects before `vitest` catches it

**What goes wrong:** `pnpm test` (vitest) passes but `pnpm build` (tsc + vite build) fails with a TypeScript error in a new file.

**Why it happens:** vitest with jsdom tolerates some TypeScript errors that `tsc --strict` catches — notably missing props on components, incorrect event handler types, or the `Tab['type']` discriminant union not including `'help'`.

**How to avoid:** After writing all new components, run `cd frontend && npx tsc --noEmit` before committing. The Tab type in `TabBar.tsx` or App.tsx may need `'help'` added to the union. [VERIFIED: memory note "Run tsc in the frontend gate, not just vitest"]

### Pitfall 6: Tab type discriminant union missing `'help'`

**What goes wrong:** TypeScript narrows `tab.type` in the tab-rendering switch and emits an error about `'help'` not being assignable to the existing union.

**Where to look:** The `Tab` interface in `frontend/src/components/TabBar.tsx`. Check if `type` is typed as a string literal union. If so, add `'help'` to it.

```bash
grep -n "type.*welcome\|type.*settings\|type.*hub" /Users/ken/dev/agenthub/frontend/src/components/TabBar.tsx
```

---

## Code Examples

### Complete HelpTab.tsx skeleton

```typescript
// Source: Pattern 1 (App.tsx special-tab) + Pattern 5 (search index) + Pattern 7 (IntersectionObserver)
import React, { useState, useMemo, useCallback, useRef, useEffect } from 'react'
import { HelpSearch } from './HelpSearch'
import { HelpSectionNav } from './HelpSectionNav'
import { HelpContent } from './HelpContent'
import gettingStartedMd from '../content/help/getting-started.md?raw'
import faqMd from '../content/help/faq.md?raw'

// ... (SearchEntry type, stripMd, searchIndex useMemo as in Pattern 5)

export function HelpTab(): React.ReactElement {
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [activeSection, setActiveSection] = useState('help-getting-started')
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const contentPaneRef = useRef<HTMLDivElement>(null)

  // ... debounce, searchIndex, results as in Patterns 5+6

  const allMarkdown = useMemo(() => `${gettingStartedMd}\n\n${faqMd}`, [])

  return (
    <div className="help-tab">
      <div className="help-tab__search">
        <HelpSearch
          query={query}
          results={results}
          onQueryChange={handleQueryChange}
          onJumpToSection={(id) => {
            const el = document.getElementById(id)
            el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
            setActiveSection(id)
            setQuery('')
          }}
        />
      </div>
      <div className="help-tab__layout">
        <HelpSectionNav
          activeSection={activeSection}
          onSectionChange={setActiveSection}
          contentPaneRef={contentPaneRef}
        />
        <div className="help-tab__content" ref={contentPaneRef}>
          <HelpContent markdown={allMarkdown} />
        </div>
      </div>
    </div>
  )
}
```

### rehype-sanitize import pattern (TypeScript-safe)

```typescript
// Source: rehype-sanitize 6.0.0 — ESM-only package, typed
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize'
import type { Schema } from 'hast-util-sanitize'

const sanitizeSchema: Schema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), 'mark'],
  attributes: {
    ...defaultSchema.attributes,
    mark: ['className'],
  },
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `marked` + `dangerouslySetInnerHTML` | `react-markdown` (virtual DOM, no innerHTML) | ~2019+ | react-markdown 10.1.0 is already installed; use it |
| `remark-html` (output HTML string) | `remark-rehype` + `hast-util-to-jsx-runtime` (react-markdown's pipeline) | ~2022 | Already handled internally by react-markdown 10 |
| IntersectionObserver docs: pass `threshold: 1` for "fully visible" | `threshold: 0` + `rootMargin` to control trigger zone | Ongoing | `threshold: 0` fires when any pixel enters the root; control active-section via rootMargin instead |

**Deprecated/outdated:**
- `react-markdown` v6/v7 API (`renderers` prop): replaced by `components` prop in v8+. Version 10.1.0 uses `components`. [VERIFIED: react-markdown readme in node_modules]
- `rehype-sanitize` with `allowedTags` option: that was `xss-filters` API. rehype-sanitize uses `Schema` with `tagNames` and `attributes`.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Per-paragraph granularity yields ~25–40 search index entries | Standard Stack / Pattern 5 | Low — the exact count doesn't affect implementation; planner can adjust |
| A2 | Keyboard Shortcuts section should be omitted (no currently documented shortcuts found) | Architecture Patterns / Recommended Structure | Medium — if shortcuts exist, a `keyboard-shortcuts.md` file needs creating and the section nav needs a 3rd entry |
| A3 | IntersectionObserver with `root: contentPaneRef.current` + `rootMargin: '-80px 0px -60% 0px'` works for scroll-spy | Pattern 7 | Medium — the exact rootMargin values need empirical tuning; -60% may need adjustment; planner should note this as requiring UAT |
| A4 | The `Tab` type union in `TabBar.tsx` requires adding `'help'` | Pitfall 6 | Low if wrong — tsc will catch it at build time |
| A5 | Total index size (~25–40 entries) is fast enough for synchronous in-memory search | Pattern 5 | Low — even 200 entries would be fast with simple `String.includes` |

**If this table is empty:** it is not — there are 5 assumed claims above. All are low-to-medium risk and either verified at test time or guided by UAT.

---

## Open Questions

1. **Keyboard Shortcuts content**
   - What we know: The UI-SPEC says to omit the section if no shortcuts are documented. The codebase has no discovered keyboard shortcut documentation file.
   - What's unclear: Are there undocumented keyboard shortcuts that should be in the Help page?
   - Recommendation: Omit the Keyboard Shortcuts section for v1. Create the section nav entry and `.md` file only if the maintainer adds content.

2. **`frontend/src/content/help/` directory location**
   - What we know: The CONTEXT.md says "a new content directory under `frontend/src/` (or assets)".
   - What's unclear: Whether there's a style preference for `src/content/` vs `src/assets/`.
   - Recommendation: Use `frontend/src/content/help/` — this is a source-code concern (the Markdown is imported as a module via `?raw`), not a static asset.

3. **Tab type union location**
   - What we know: App.tsx defines `SETTINGS_TAB`, `HUB_TAB` etc. The `Tab` type is in `TabBar.tsx`.
   - What's unclear: Whether `type` on `Tab` is an open `string` or a closed literal union.
   - Recommendation: Check `TabBar.tsx` for the `Tab` interface definition before implementing. [Action: planner should add this as a Wave 0 investigation step]

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Vite build, pnpm, vitest | ✓ | v24.14.1 | — |
| pnpm | Package install | ✓ | 9.15.9 | — |
| `react-markdown` | HelpContent rendering | ✓ | 10.1.0 (installed) | — |
| `remark-gfm` | GFM tables/strikethrough | ✓ | 4.0.1 (installed) | — |
| `rehype-sanitize` | Defense-in-depth sanitize | ✗ | — (not installed) | Skip defense-in-depth (acceptable for maintainer content) |
| `@heroicons/react` | QuestionMarkCircleIcon | ✓ | ^2.2.0 (installed) | — |
| IntersectionObserver (browser) | HelpSectionNav active-section | ✓ | Native in Wails WebView (Chromium) | jsdom test polyfill needed |

**Missing dependencies with no fallback:** none — `rehype-sanitize` has a viable fallback (omit it, content is still safe via react-markdown's virtual DOM).

**Missing dependencies with fallback:** `rehype-sanitize` — optional defense-in-depth layer. Install it anyway; it's a simple one-package add.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (inline `test:` block) |
| Environment | jsdom 29.0.0 |
| Setup file | `frontend/src/test-setup.ts` |
| Quick run command | `cd frontend && pnpm test` |
| Full suite + type check | `cd frontend && npx tsc --noEmit && pnpm test` |
| Build gate | `cd frontend && npx tsc && vite build` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HELP-01 | Sidebar renders Help button (4th item, aria-label="Help") | unit | `cd frontend && pnpm test` → Sidebar.test.tsx | ❌ Wave 0 (update existing) |
| HELP-01 | Help button fires onOpenHelp when clicked | unit | `cd frontend && pnpm test` → Sidebar.test.tsx | ❌ Wave 0 (update existing) |
| HELP-01 | Help button has sidebar__item--active when activePanel === '__help__' | unit | `cd frontend && pnpm test` → Sidebar.test.tsx | ❌ Wave 0 (update existing) |
| HELP-01 | `HELP_TAB` constant and `handleOpenHelp` exist in App.tsx (source gate) | unit | `cd frontend && pnpm test` → HelpTab.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | HelpSearch debounce: query change sets debouncedQuery after 200ms | unit | `cd frontend && pnpm test` → HelpSearch.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | HelpSearch renders search input with visible label "Search help…" | unit | `cd frontend && pnpm test` → HelpSearch.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | HelpSearch renders clear button with aria-label="Clear search" | unit | `cd frontend && pnpm test` → HelpSearch.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | HelpSearch shows empty-state when query non-empty AND 0 results | unit | `cd frontend && pnpm test` → HelpSearch.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | HelpSearch does NOT show empty-state for empty query | unit | `cd frontend && pnpm test` → HelpSearch.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | HighlightedSnippet wraps matched term in `<mark class="help-search__mark">` | unit | `cd frontend && pnpm test` → HelpSearch.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | HelpSectionNav renders buttons for each section with correct aria-current | unit | `cd frontend && pnpm test` → HelpSectionNav.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | style.css declares `--hub-search-highlight-bg` in `:root` and `[data-ui-theme="light"]` | unit (CSS source gate) | `cd frontend && pnpm test` → HelpTab.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | HelpContent renders `<Markdown>` (source gate: react-markdown import present) | unit | `cd frontend && pnpm test` → HelpContent.test.tsx | ❌ Wave 0 (create) |
| HELP-01 | BrowserOpenURL is called when external link button is clicked | unit | `cd frontend && pnpm test` → HelpContent.test.tsx | ❌ Wave 0 (create) |

**Manual-only behaviors (add to TESTING.md §5):**
- **M-NN** Help page opens in the live Wails native webview when the Help sidebar button is clicked. Full Markdown content renders (headings, paragraphs, code spans). Section-nav active state updates on scroll. External links open the system browser.
  - _Why not automatable:_ Wails native webview is not accessible to Playwright or headless browser automation.
  - _Source:_ Phase 147

### Wave 0 Gaps (files to create before implementation begins)

- [ ] Add `IntersectionObserver` polyfill to `frontend/src/test-setup.ts`
- [ ] Update `Sidebar.test.tsx` — change "exactly 3 sidebar\_\_item buttons" assertions to 4; add Help button assertions
- [ ] Create `frontend/src/components/__tests__/HelpTab.test.tsx` — HELP_TAB constant, handleOpenHelp source gates, CSS token source gate
- [ ] Create `frontend/src/components/__tests__/HelpSearch.test.tsx` — debounce, highlight, empty-state, clear button
- [ ] Create `frontend/src/components/__tests__/HelpSectionNav.test.tsx` — section-nav render, aria-current
- [ ] Create `frontend/src/components/__tests__/HelpContent.test.tsx` — react-markdown source gate, BrowserOpenURL call
- [ ] Create `frontend/src/content/help/` directory with placeholder `.md` files (Wave 0 or Wave 1)

### Sampling Rate

- **Per task commit:** `cd frontend && pnpm test`
- **Per wave merge:** `cd frontend && npx tsc --noEmit && pnpm test`
- **Phase gate:** Full suite green (`tsc` + `pnpm test`) before `/gsd:verify-work`

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | not applicable (no auth surface) |
| V3 Session Management | no | not applicable |
| V4 Access Control | no | not applicable (Help is read-only content) |
| V5 Input Validation | yes — search input | Plain substring match on in-memory index; no server call; no eval; no innerHTML; React escapes all string values in JSX |
| V6 Cryptography | no | not applicable |
| V7 Error Handling | no | not applicable |
| V8 Data Protection | no | not applicable (no PII in Help content) |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| XSS via rendered Markdown | Spoofing / Tampering | `react-markdown` (virtual DOM, no innerHTML) + `rehype-sanitize` with `defaultSchema` extended for `<mark>` |
| XSS via search highlight injection | Tampering | Plain-string React JSX approach (React escapes all values); no `dangerouslySetInnerHTML` used |
| External link hijack (webview navigation) | Spoofing | All links rendered as `<button>` + `BrowserOpenURL`; no `<a href>` anywhere in new code |

**Net security posture:** The Help page is a read-only surface rendering maintainer-authored, bundled content. The only user-supplied input is the search query, which is processed entirely in-memory via string matching and rendered via React JSX (auto-escaped). No network calls, no user-generated content, no authentication. Security risk level: LOW.

---

## Sources

### Primary (HIGH confidence)

- `frontend/package.json` — confirmed `react-markdown: 10.1.0`, `remark-gfm: ^4.0.1` already installed; `rehype-sanitize` NOT installed
- `frontend/pnpm-lock.yaml` — confirmed resolved versions; `rehype-sanitize` absent from lockfile
- `frontend/node_modules/react-markdown/readme.md` — confirmed v10 API (`components` prop, safe-by-default, `rehype-sanitize` recommended for extra safety)
- `frontend/src/App.tsx` (lines 95-99, 780-788, 1200-1210, 1532-1556) — confirmed `SETTINGS_TAB`, `HUB_TAB`, `handleOpenSettings`, `handleOpenHub` patterns; confirmed display-toggle mount strategy for settings
- `frontend/src/components/Sidebar.tsx` — confirmed current 3-item sidebar structure; `sidebar__bottom` div; `activePanel === '__hub__'` active pattern
- `frontend/src/components/SettingsJumpBar.tsx` — confirmed NO IntersectionObserver; plain `<a href="#">` + CSS; active-section scroll-spy is new work
- `frontend/src/components/SettingsSearch.tsx` — confirmed `useMemo` + `filter` pattern; no debounce in SettingsSearch (it's synchronous); help needs debounce because body search is heavier
- `frontend/vite.config.ts` — confirmed `?raw` support is available (base config; no additional plugin needed)
- `frontend/src/components/FindBar/__tests__/FindBar.visual.test.tsx` line 17 — confirmed `?raw` import pattern works in this codebase
- `frontend/src/test-setup.ts` — confirmed `ResizeObserver` polyfill pattern; `IntersectionObserver` polyfill NOT yet present (must be added)
- `frontend/src/components/__tests__/Sidebar.test.tsx` (line 95) — confirmed the "exactly 3 items" assertion that will break
- `frontend/src/components/SettingsTab.tsx` (line 609) — confirmed `BrowserOpenURL` pattern
- `npm view rehype-sanitize` — confirmed package exists, v6.0.0, maintained by `wooorm` (core unified maintainer), 0 postinstall scripts, MIT license

### Secondary (MEDIUM confidence)

- `TESTING.md` §4 Traceability Map — confirmed new test files for HELP-01 must be added as rows; confirmed `bash tests/check-traceability-paths.sh` must be run before commit
- `TESTING.md` §5 Manual Checklist (M-01 pattern) — confirmed format for new M-NN entry for Wails-native UAT

### Tertiary (LOW confidence)

- IntersectionObserver `rootMargin: '-80px 0px -60% 0px'` for active-section detection — exact values need UAT tuning [ASSUMED]

---

## Metadata

**Confidence breakdown:**
- Standard Stack (installed packages, versions): HIGH — verified against `pnpm-lock.yaml` and `package.json`
- App.tsx routing pattern (`HELP_TAB`, `handleOpenHelp`): HIGH — verified against existing `SETTINGS_TAB`/`HUB_TAB` patterns
- `?raw` import for `.md` files: HIGH — verified against existing `?raw` usage in codebase + Vite docs
- Markdown rendering safety: HIGH — verified against `react-markdown` readme + virtual DOM analysis
- `<mark>` injection strategy (plain-text vs rehype plugin): HIGH — verified by reading the UI-SPEC surface separation (results list vs content pane)
- IntersectionObserver active-section rootMargin values: LOW — needs empirical tuning; functional pattern is HIGH
- Search index granularity / entry count: MEDIUM — estimated from known content (6 FAQ + ~10-15 getting-started paragraphs)

**Research date:** 2026-06-22
**Valid until:** 2026-07-22 (30 days — stable packages, no fast-moving components)
