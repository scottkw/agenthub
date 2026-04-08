# Phase 55: Sidebar Component & Icons - Research

**Researched:** 2026-04-08
**Domain:** React sidebar layout, Heroicons SVG icons, CSS flexbox, localStorage persistence
**Confidence:** HIGH

## Summary

Phase 55 replaces the current top toolbar action buttons (Unicode characters in TabBar.tsx) with a
collapsible left sidebar using proper SVG icons from @heroicons/react. The app currently uses a
flex-column layout (`.app`) with a horizontal `.tab-bar` at the top; this phase adds a sidebar
component that sits to the left of the main content area inside a horizontal flex row.

The icon library (@heroicons/react 2.2.0) is already published and both required icons are
confirmed present in the package: `ServerStackIcon` (for Sessions) and `Bars3Icon` (hamburger for
the sidebar toggle). The sidebar's collapsed/expanded state must be persisted via `localStorage`,
which is a simple key-value read/write — no library needed.

The existing test suite has one pre-existing failure unrelated to this phase: the `TabBar.test.tsx`
UILAY-01 suite checks `font-size: 18px` but the CSS was updated to `20px` by quick-task
`260407-w91`. The planner should include a task to fix this stale test assertion.

**Primary recommendation:** Install `@heroicons/react`, create a new `Sidebar.tsx` component with
its own CSS block, restructure `.app` from flex-column to flex-row (sidebar + content column), and
persist state via `localStorage.getItem/setItem('sidebar-collapsed', ...)`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ICON-01 | All sidebar icons use Heroicons SVGs instead of Unicode characters | @heroicons/react 2.2.0 confirmed available; ServerStackIcon and Bars3Icon both exist in 24/outline |
| ICON-02 | Sessions uses ServerStackIcon (hamburger is now sidebar toggle, not sessions) | `ServerStackIcon` confirmed in package/24/outline/ServerStackIcon.js |
| SIDE-01 | User sees a collapsible left sidebar with navigation icons instead of top toolbar buttons | CSS layout restructure: `.app` flex-direction row, new `.sidebar` component |
| SIDE-02 | Sidebar toggles between collapsed (48px, icons only) and expanded (200px, icons + labels) via hamburger | CSS width transition on `.sidebar`, boolean `collapsed` state in React |
| SIDE-03 | Sidebar state persists across app restarts via localStorage | `localStorage.getItem/setItem('sidebar-collapsed')` — simple string boolean |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @heroicons/react | 2.2.0 (latest) | MIT-licensed SVG icon components for React | Official Tailwind Labs library, zero dependencies, tree-shakeable, used throughout the ecosystem |
| React | 19.2.4 (already installed) | Component framework | Project baseline |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| localStorage (Web API) | browser-native | Persist sidebar collapsed state | No library needed — `localStorage.getItem/setItem` is sufficient for a single boolean |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| @heroicons/react | Inline SVG strings in component | Heroicons package provides type-safe named imports and React components; inline SVG is more code |
| @heroicons/react | lucide-react, phosphor-icons | Heroicons is specified in requirements (ICON-01); no reason to deviate |
| localStorage | Wails Go backend preference storage | localStorage is sufficient for a pure UI state preference; Go backend is overkill |

**Installation:**
```bash
cd frontend && pnpm add @heroicons/react
```

**Version verification:** Confirmed via `npm view @heroicons/react version` → `2.2.0` (2024).

## Architecture Patterns

### Recommended Project Structure
```
frontend/src/
├── components/
│   ├── Sidebar.tsx          # NEW: collapsible sidebar component
│   ├── TabBar.tsx           # MODIFY: remove tab-bar__controls section (action buttons move to sidebar)
│   └── ... (existing)
├── style.css                # MODIFY: add sidebar CSS, restructure .app layout
└── App.tsx                  # MODIFY: add sidebar, restructure layout div
```

### Pattern 1: Sidebar Component with localStorage Persistence

**What:** A `Sidebar` component that reads initial collapsed state from localStorage, renders
icons-only or icons+labels depending on state, and writes state changes back to localStorage.

**When to use:** Single-panel navigation sidebar with a single persistent boolean state.

**Example:**
```typescript
// Source: heroicons README + browser localStorage API
import { Bars3Icon, ServerStackIcon } from '@heroicons/react/24/outline'

const STORAGE_KEY = 'sidebar-collapsed'

interface SidebarProps {
  onOpenDaemonManager: () => void
  onOpenRemoteSessions: () => void
  onAdd: () => void
  onSettings: () => void
}

export function Sidebar({ onOpenDaemonManager, onOpenRemoteSessions, onAdd, onSettings }: SidebarProps) {
  const [collapsed, setCollapsed] = useState<boolean>(() => {
    return localStorage.getItem(STORAGE_KEY) === 'true'
  })

  const toggle = () => {
    setCollapsed(prev => {
      const next = !prev
      localStorage.setItem(STORAGE_KEY, String(next))
      return next
    })
  }

  return (
    <nav className={`sidebar${collapsed ? ' sidebar--collapsed' : ''}`}>
      <button className="sidebar__toggle" onClick={toggle} aria-label="Toggle sidebar">
        <Bars3Icon className="sidebar__icon" />
      </button>
      <button className="sidebar__item" onClick={onOpenDaemonManager}>
        <ServerStackIcon className="sidebar__icon" />
        {!collapsed && <span className="sidebar__label">Sessions</span>}
      </button>
      {/* ... other items */}
    </nav>
  )
}
```

### Pattern 2: App Layout Restructure (flex-row)

**What:** Change `.app` from `flex-direction: column` to `flex-direction: row`. The sidebar
occupies fixed width on the left; the content column (tab-bar + terminal-container) fills the rest.

**When to use:** Any app shell that needs a persistent side panel.

**Example:**
```css
/* style.css — restructured layout */
.app {
  display: flex;
  flex-direction: row;   /* was: column */
  height: 100%;
  width: 100%;
}

.app__content {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;          /* prevent flex child overflow */
}

.sidebar {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  width: 200px;
  background-color: #16161e;
  border-right: 1px solid #292e42;
  transition: width 0.15s ease;
  overflow: hidden;
}

.sidebar--collapsed {
  width: 48px;
}
```

### Pattern 3: Icon Sizing for Sidebar

**What:** Heroicons returns `<svg>` elements. Size them via CSS class or inline `className` with
explicit width/height. For a 48px collapsed sidebar, icons should be 20px to leave 14px padding
(7px each side).

**Example:**
```typescript
// 20x20 icons via className — matches Heroicons recommended usage
<ServerStackIcon className="sidebar__icon" />
```

```css
.sidebar__icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  color: #9aa5ce;  /* matches existing toolbar icon color */
}
```

### Anti-Patterns to Avoid

- **Rendering icon labels with `display: none` in collapsed mode:** Use conditional rendering
  (`{!collapsed && <span>...`)  OR use CSS to hide labels. Conditional rendering is simpler and
  avoids layout shift. If CSS is used, use `opacity: 0; width: 0` with overflow hidden rather than
  `display: none` (so transitions work).
- **Storing collapsed state in App.tsx state that doesn't persist:** Must use `localStorage` (SIDE-03).
  Initialize via lazy `useState(() => localStorage.getItem(...) === 'true')` so the initial render
  uses the persisted value.
- **Changing `.app` flex-direction without wrapping tab-bar + terminal-container:** The tab-bar and
  terminal-container must be wrapped in a single `.app__content` div so they still stack vertically
  inside the horizontal flex row.
- **Forgetting min-width: 0 on the flex content column:** Without this, long content (e.g. tab names)
  can cause the content area to overflow past the sidebar.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SVG icons | Custom SVG strings or Unicode emoji | @heroicons/react | Pixel-perfect, accessible, tree-shakeable, MIT licensed |
| CSS transitions for sidebar width | JavaScript-driven width changes | CSS `transition: width` | Smoother, GPU-accelerated, no JS overhead |

**Key insight:** The sidebar transition and icon rendering are pure CSS + existing React patterns.
No state management library, no animation library, no additional deps beyond @heroicons/react.

## Common Pitfalls

### Pitfall 1: Stale TabBar.test.tsx font-size assertion
**What goes wrong:** One existing test (`TabBar.test.tsx` line 135) asserts `font-size: 18px` for
`.tab-bar__btn` but the CSS currently has `font-size: 20px` (changed in quick-task 260407-w91).
The test suite currently has 1 failure before any changes are made.
**Why it happens:** Quick task updated CSS but didn't update the test assertion.
**How to avoid:** Fix the stale assertion in the same plan that installs the sidebar, or in a
dedicated Wave 0 setup task. Don't leave it for after — CI will be red otherwise.
**Warning signs:** `pnpm test` shows 1 failed, 247 passed before any sidebar changes.

### Pitfall 2: localStorage not available in test environment
**What goes wrong:** vitest with jsdom provides `localStorage`, but `localStorage.getItem` returns
`null` (not the string `'true'`) by default. Tests that render Sidebar without seeding localStorage
will get `collapsed = false`, which is correct default behavior — no action needed.
**How to avoid:** Tests should assert the default state (not collapsed) without needing to seed
localStorage. For a test that checks persistence, call `localStorage.setItem` in test setup.

### Pitfall 3: Heroicons import path
**What goes wrong:** Importing from `@heroicons/react` directly (no sub-path) will fail — the
package requires the size/style sub-path.
**How to avoid:** Always use `@heroicons/react/24/outline` or `@heroicons/react/24/solid`.
```typescript
// WRONG: import { ServerStackIcon } from '@heroicons/react'
// RIGHT:
import { ServerStackIcon, Bars3Icon } from '@heroicons/react/24/outline'
```

### Pitfall 4: TabBar action buttons not removed
**What goes wrong:** If the new sidebar adds nav buttons but the old TabBar action buttons are not
removed, users see duplicate controls.
**How to avoid:** Phase 55 should remove `onOpenDaemonManager`, `onOpenRemoteSessions`, `onSettings`
props from TabBar (they move to Sidebar). Phase 56 will wire the actual navigation actions.
However — check if Phase 56 depends on them. Per the roadmap, Phase 55 builds the sidebar shell
and Phase 56 wires navigation. The safest split: Phase 55 removes the buttons from TabBar and
adds them to Sidebar with the same callback props; Phase 56 can then re-wire the callbacks.

### Pitfall 5: Terminal resize after layout change
**What goes wrong:** xterm.js FitAddon sizes the terminal to its container. Adding a sidebar
reduces the available width. If the terminal isn't refitted after the sidebar layout change,
the terminal width will be wrong.
**How to avoid:** The FitAddon uses ResizeObserver internally in TerminalPanel — verify it
re-fits when the container width changes. If not, trigger a manual fit on sidebar toggle.
The Sidebar component can call a callback `onToggle` if App.tsx needs to react.

## Code Examples

Verified patterns from official sources:

### Heroicons React Import (24/outline)
```typescript
// Source: @heroicons/react 2.2.0 package verified
import { ServerStackIcon, Bars3Icon, HomeIcon, GlobeAltIcon, PlusIcon, Cog6ToothIcon } from '@heroicons/react/24/outline'
```

### Icon Inventory for Phase 55 (confirmed in package)
| Icon Name | Purpose | Confirmed in package |
|-----------|---------|---------------------|
| `Bars3Icon` | Hamburger — sidebar toggle | Yes, package/24/outline/Bars3Icon.js |
| `ServerStackIcon` | Sessions (Daemon Manager) | Yes, package/24/outline/ServerStackIcon.js |
| `HomeIcon` | Welcome tab (Phase 56) | Yes, package/24/outline/HomeIcon.js |
| `GlobeAltIcon` | Remote Sessions (Phase 56) | Yes, package/24/outline/GlobeAltIcon.js |
| `PlusIcon` | New tab (Phase 56) | Yes, package/24/outline/PlusIcon.js |
| `Cog6ToothIcon` | Settings (Phase 56) | Yes, package/24/outline/Cog6ToothIcon.js |

### localStorage Persistence Pattern
```typescript
// Source: browser Web API — no library needed
// Initial state from localStorage (lazy initializer runs once on mount)
const [collapsed, setCollapsed] = useState<boolean>(
  () => localStorage.getItem('sidebar-collapsed') === 'true'
)

// Toggle and persist
const handleToggle = () => {
  setCollapsed(prev => {
    const next = !prev
    localStorage.setItem('sidebar-collapsed', String(next))
    return next
  })
}
```

### CSS Sidebar Width Transition
```css
.sidebar {
  width: 200px;
  transition: width 0.15s ease;
  overflow: hidden;
}
.sidebar--collapsed {
  width: 48px;
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Unicode emoji/symbols in buttons | Heroicons SVG React components | This phase | Crisp rendering at all DPI, accessible, tree-shakeable |
| Toolbar buttons in horizontal tab bar | Left sidebar with icons+labels | This phase | Standard app shell pattern |

**Deprecated/outdated (in this codebase):**
- `tab-bar__controls` section in TabBar — will be removed (buttons move to Sidebar)
- Unicode `&#9776;` hamburger for sessions — replaced by `ServerStackIcon` (hamburger becomes sidebar toggle)
- Unicode `&#9881;` gear for settings — replaced by `Cog6ToothIcon`
- Unicode `&#127760;` globe for remote — replaced by `GlobeAltIcon`

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| pnpm | Package install | Checked via CLAUDE.md | preferred | npm |
| @heroicons/react | ICON-01, ICON-02 | Not installed (needs `pnpm add`) | 2.2.0 available on npm | — |
| localStorage | SIDE-03 | Provided by jsdom in tests, Wails WebView in app | browser-native | — |
| vitest | Testing | Installed in devDependencies | 4.1.0 | — |

**Missing dependencies with no fallback:**
- @heroicons/react: must be installed (`cd frontend && pnpm add @heroicons/react`)

**Missing dependencies with fallback:**
- None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | frontend/vite.config.ts (test.environment = 'jsdom') |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SIDE-01 | Sidebar renders with `.sidebar` class and navigation icon buttons | unit | `cd frontend && pnpm test -- --reporter=verbose Sidebar` | No — Wave 0 gap |
| SIDE-02 | Sidebar collapses to 48px / expands to 200px on toggle; CSS classes `.sidebar--collapsed` applied | unit (CSS assertion) | `cd frontend && pnpm test -- --reporter=verbose Sidebar` | No — Wave 0 gap |
| SIDE-03 | localStorage key `sidebar-collapsed` is read on init and written on toggle | unit | `cd frontend && pnpm test -- --reporter=verbose Sidebar` | No — Wave 0 gap |
| ICON-01 | Sidebar renders SVG elements (not Unicode text) for icons | unit | `cd frontend && pnpm test -- --reporter=verbose Sidebar` | No — Wave 0 gap |
| ICON-02 | Sessions button contains ServerStackIcon SVG (not hamburger) | unit | `cd frontend && pnpm test -- --reporter=verbose Sidebar` | No — Wave 0 gap |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm test`
- **Per wave merge:** `cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/Sidebar.test.tsx` — covers SIDE-01, SIDE-02, SIDE-03, ICON-01, ICON-02
- [ ] Fix stale assertion in `frontend/src/components/__tests__/TabBar.test.tsx:135` — `font-size: 18px` should be `20px`

## Open Questions

1. **Terminal refit on sidebar toggle**
   - What we know: FitAddon uses ResizeObserver on the terminal container div; changing sidebar width
     changes the container width.
   - What's unclear: Does Wails' embedded WebView and the ResizeObserver fire reliably on CSS
     transition width changes? In standard browser environments it does.
   - Recommendation: Implement the sidebar first; verify terminal refit during UAT. If ResizeObserver
     doesn't fire on CSS width transition, add a window `resize` event dispatch after toggle.

2. **TabBar prop cleanup scope for Phase 55 vs Phase 56**
   - What we know: Phase 55 builds the sidebar shell; Phase 56 wires navigation actions.
   - What's unclear: Should Phase 55 remove `onOpenDaemonManager`, `onOpenRemoteSessions`, and
     `onSettings` from TabBar, or keep them until Phase 56?
   - Recommendation: Remove the three action buttons from TabBar in Phase 55 (they are replaced by
     sidebar items). Keep the callback props wired from App.tsx to Sidebar in Phase 55 so Phase 56
     only needs to add more items, not restructure.

## Sources

### Primary (HIGH confidence)
- `npm pack @heroicons/react` — inspected package/24/outline/ file listing to confirm
  ServerStackIcon, Bars3Icon, HomeIcon, GlobeAltIcon, PlusIcon, Cog6ToothIcon all exist in 2.2.0
- `npm view @heroicons/react version` → confirmed 2.2.0 is latest
- MDN Web API (localStorage) — native browser API, universally available in jsdom and Wails WebView
- Project source: `/Users/ken/dev/agenthub/frontend/src/App.tsx`, `TabBar.tsx`, `style.css` — direct inspection

### Secondary (MEDIUM confidence)
- heroicons GitHub README (via WebFetch) — confirmed import pattern `@heroicons/react/24/outline`
- WebSearch heroicons 2.x — confirmed Bars3Icon = hamburger, ServerStackIcon = server stack

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — versions confirmed via npm registry, package contents inspected directly
- Architecture: HIGH — existing codebase fully read; sidebar pattern is standard React/CSS
- Pitfalls: HIGH — pre-existing test failure verified by running `pnpm test`

**Research date:** 2026-04-08
**Valid until:** 2026-05-08 (stable library, no fast-moving parts)

## Project Constraints (from CLAUDE.md)

| Directive | Applies To |
|-----------|------------|
| Use `pnpm` (preferred) for Node package management | `pnpm add @heroicons/react` |
| TypeScript strict mode, `camelCase`/`PascalCase` | Sidebar.tsx must use proper types and naming |
| `noUnusedLocals`, `noUnusedParameters` (tsconfig) | All props accepted by Sidebar must be used |
| ESLint + Prettier | Format new files consistently |
| No global package installs | `cd frontend && pnpm add ...` scoped to project |
| Testing: vitest, 80%+ coverage in critical components | Sidebar test file required before implementation |
