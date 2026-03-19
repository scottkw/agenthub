# Phase 9: Settings Modal Overhaul - Research

**Researched:** 2026-03-19
**Domain:** React modal UI — tabbed layout, CSS BEM, component refactor
**Confidence:** HIGH

## Summary

Phase 9 is a focused UI refactor of an existing `SettingsPanel` React component. The current panel renders two logically separate groups of settings — CLI Paths and Web Serving — stacked vertically in a single scrollable body, separated only by an `<hr>` divider. The goal is to replace that flat layout with a tab switcher so only one group is visible at a time, and to replace the current dual-button footer (Cancel / Save) with a single "Close" button that has improved styling.

The component is pure React + plain CSS BEM. There are no third-party UI component libraries in this project — no Vuetify, no MUI, no Headless UI. All tabs must be hand-rolled in React with the project's existing BEM CSS pattern. This is a small, self-contained change: no new npm packages are needed, no backend changes are required, and no Wails bindings are affected.

The current footer has a semantic mismatch: "Save" saves only CLI paths (the Web Serving section has its own inline save actions), so a single "Close" button is both simpler and more honest about what the footer does. The per-section action buttons (Set Password, Start/Stop Web Server) are in-body and unaffected by footer simplification.

**Primary recommendation:** Add a `activeTab` state variable (`'cli-paths' | 'web-serving'`), render tab buttons in the header area, conditionally render the correct section in the body, and replace the footer with a single Close button. All changes stay within `SettingsPanel.tsx` and `style.css`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SETT-01 | Settings modal uses a tabbed layout to reduce crowding (e.g., CLI Paths \| Web Serving) | Tab state + conditional rendering in SettingsPanel.tsx; CSS tab button styles in style.css |
| SETT-02 | Settings modal has improved styling and visual organization | Better spacing, visual hierarchy, single-action footer, consistent color tokens matching existing design language |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.2.4 | Component state for active tab, conditional rendering | Already in project |
| Plain CSS (BEM) | — | Tab button + tab panel styles | Only CSS approach used in this project |
| Vitest | 4.1.0 | Unit tests for tab switching behavior | Already configured, used for all existing component tests |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| jsdom | 29.0.0 | Simulates DOM in Vitest | All component tests already run with jsdom environment |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled tabs | Headless UI / Radix Tabs | No external UI libs in this project; hand-rolled is ~20 lines of JSX |
| Single CSS file | CSS Modules | Project uses a flat style.css; no build changes warranted for this scope |

**Installation:** No new packages required.

## Architecture Patterns

### Relevant Project Structure
```
frontend/src/
├── components/
│   ├── SettingsPanel.tsx     # MODIFY — add tab state + tab UI
│   ├── __tests__/
│   │   └── SettingsPanel.test.tsx   # CREATE — new test file
├── style.css                 # MODIFY — add .settings-panel__tabs + __tab-btn CSS
```

### Pattern 1: Controlled Tab State in React
**What:** A single `useState` holds the active tab ID. Tab buttons set it. The body conditionally renders the matching section.
**When to use:** Any time a modal has 2+ mutually exclusive content sections.
**Example:**
```typescript
// Source: standard React controlled component pattern
type Tab = 'cli-paths' | 'web-serving'

const [activeTab, setActiveTab] = useState<Tab>('cli-paths')

// Tab buttons (in header or sub-header row):
<button
  className={`settings-panel__tab-btn ${activeTab === 'cli-paths' ? 'settings-panel__tab-btn--active' : ''}`}
  onClick={() => setActiveTab('cli-paths')}
>
  CLI Paths
</button>
<button
  className={`settings-panel__tab-btn ${activeTab === 'web-serving' ? 'settings-panel__tab-btn--active' : ''}`}
  onClick={() => setActiveTab('web-serving')}
>
  Web Serving
</button>

// Conditional body:
{activeTab === 'cli-paths' && <div className="settings-panel__tab-panel">...</div>}
{activeTab === 'web-serving' && <div className="settings-panel__tab-panel">...</div>}
```

### Pattern 2: Tab Bar Below Modal Header
**What:** The modal header stays as-is (title + close X). A second row directly beneath it holds the tab buttons, separated by a border. This avoids cramming tabs into the title row.
**When to use:** When the header already has elements (title and close button) that would compete for horizontal space.
**Example layout:**
```
┌─────────────────────────────────┐
│  Settings                    ×  │  ← .settings-panel__header
├─────────────────────────────────│
│  [CLI Paths]  [Web Serving]     │  ← .settings-panel__tabs
├─────────────────────────────────│
│  <scrollable body content>      │  ← .settings-panel__body
├─────────────────────────────────│
│                       [Close]   │  ← .settings-panel__footer (single button)
└─────────────────────────────────┘
```

### Pattern 3: Footer Simplification
**What:** Replace the two-button footer (Cancel + Save) with a single "Close" button.
**Why it's correct:** The current "Save" only saves CLI paths. Web Serving section has its own inline actions (Set Password button, Start/Stop button). Making "Save" behave differently depending on active tab is confusing. A single Close button is semantically honest: the user acts inline within each tab.
**Impact on existing code:** `handleSaveCLIPaths` is currently called by the Save footer button. After this change it should be called by an inline "Save Paths" button within the CLI Paths tab panel (or triggered on input blur/change, matching the Web Serving pattern). The function itself is unchanged.

### Anti-Patterns to Avoid
- **CSS `display: none` toggling for tab switching:** The current project pattern for showing/hiding sections (e.g., StatusBar state) uses JSX conditionals, not CSS display. Use `{activeTab === 'cli-paths' && ...}` not `style={{ display: activeTab === 'cli-paths' ? 'flex' : 'none' }}`. This is documented in STATE.md as a project decision for Phase 8.
- **Moving `handleSaveCLIPaths` to the footer:** The footer becomes a single "Close" button. CLI path saving must be an inline action within the CLI Paths tab.
- **Importing new UI libraries:** This project is Wails + plain React. No external component libraries.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Focus trapping in modal | Custom focus trap logic | Not needed — this is an Electron/Wails desktop app; tab focus cycling is already acceptable | Wails webview focus model; no public keyboard users |
| Tab accessibility (ARIA) | Custom ARIA roles | Add `role="tablist"`, `role="tab"`, `aria-selected` per WAI-ARIA tabs pattern | Standard attributes, 3 lines of JSX |

**Key insight:** Tab UI for two sections is genuinely simple in this codebase. The only risk is scope creep — don't try to make it animated, keyboard-navigable beyond basic click, or dynamically generated from config.

## Common Pitfalls

### Pitfall 1: Losing Web Serving State on Tab Switch
**What goes wrong:** If the Web Serving section unmounts when switching to CLI Paths tab, any in-progress server start/stop state (loading spinners, error messages) is lost.
**Why it happens:** JSX conditional `{activeTab === 'web-serving' && <section/>}` unmounts the section when inactive. All useState inside that section resets.
**How to avoid:** Keep all web serving state in the parent `SettingsPanel` component (it already is — `isServerRunning`, `serverLoading`, `serverError`, etc. are all in the top-level component state). The conditional rendering only controls visibility of JSX; state lives above it.
**Warning signs:** Server URL disappears when switching tabs and back; "Stopping..." spinner resets mid-operation.

### Pitfall 2: Save Button Removing CLI Path Changes
**What goes wrong:** After removing the footer Save button, there's no way to save CLI path overrides.
**Why it happens:** `handleSaveCLIPaths` is currently wired only to the footer Save button.
**How to avoid:** Move a "Save Paths" button inside the CLI Paths tab panel. The function itself is unchanged. Alternatively, auto-save on blur (less discoverable). A dedicated in-panel button is clearer.

### Pitfall 3: Tab Bar Overflowing at Narrow Modal Widths
**What goes wrong:** Tab buttons push outside the modal if labels are long or modal is narrow.
**Why it happens:** The modal is fixed 520px wide but can shrink to 95vw on small screens. Tab labels "CLI Paths" and "Web Serving" are short enough to fit comfortably at 520px.
**How to avoid:** Use `display: flex` on the tab bar with `overflow: hidden`. Two short tab labels at 520px is no risk; just don't add more tabs without checking.

### Pitfall 4: Close Button Style Matches Cancel Not Save
**What goes wrong:** Close button styled as the primary (blue `--save`) button when it should be secondary.
**Why it happens:** Confusing "Close" with "confirm action."
**How to avoid:** Style Close as `settings-panel__btn--cancel` (ghost/border style), matching the existing secondary button pattern. There's no "primary action" in the footer — all primary actions are inline.

## Code Examples

Verified patterns from the existing codebase:

### Existing Color Tokens (from style.css)
```css
/* Source: /frontend/src/style.css */
--bg-deep:    #1a1b26;  /* body, app background */
--bg-mid:     #1e2030;  /* panel, picker background */
--bg-bar:     #16161e;  /* tab bar, status bar */
--border:     #292e42;  /* all borders */
--text-dim:   #565f89;  /* muted labels, inactive tabs */
--text-mid:   #a9b1d6;  /* secondary text */
--text-hi:    #c0caf5;  /* primary text */
--accent:     #7aa2f7;  /* active tab underline, links, focus borders */
--accent-hi:  #89b4fa;  /* hover on accent */
--hover-bg:   #1e2030;  /* hover state background */
--select-bg:  #3b4261;  /* selected/pressed background */
```
(These are not CSS custom properties in the file — they are used as literal hex values. Document as design tokens for consistency.)

### Tab Button CSS (new, following existing BEM)
```css
/* Source: pattern from .tab-bar__btn and .tab in style.css */
.settings-panel__tabs {
  display: flex;
  flex-direction: row;
  border-bottom: 1px solid #292e42;
  padding: 0 20px;
  gap: 0;
  flex-shrink: 0;
}

.settings-panel__tab-btn {
  padding: 10px 16px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: #565f89;
  font-size: 13px;
  font-family: inherit;
  cursor: pointer;
  margin-bottom: -1px; /* overlap the tabs border-bottom */
}

.settings-panel__tab-btn:hover {
  color: #a9b1d6;
}

.settings-panel__tab-btn--active {
  color: #c0caf5;
  border-bottom-color: #7aa2f7;
}
```

### Existing Test Pattern (from StatusBar.test.tsx)
```typescript
// Source: /frontend/src/components/__tests__/StatusBar.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'

function renderComponent(props) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => { root.render(React.createElement(Component, props)) })
  return { container, root }
}

afterEach(() => { root.unmount(); container.remove() })
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single scrollable body with divider | Tabbed sections | Phase 9 | Reduces cognitive load; each section is focused |
| Cancel + Save footer | Single Close footer | Phase 9 | Removes misleading "Save" that only saved CLI paths |

## Open Questions

1. **Should CLI path saving be inline (Save Paths button in tab panel) or auto-save on blur?**
   - What we know: Web Serving uses inline buttons (Set Password, Start/Stop). Consistency favors inline buttons.
   - What's unclear: User expectation — is an explicit Save Paths button or auto-save on change more intuitive?
   - Recommendation: Use an explicit "Save Paths" button inside the CLI Paths tab panel, directly below the table. This is consistent with the Web Serving tab's pattern and makes the save action explicit.

2. **Should `handleSaveCLIPaths` still close the modal after saving?**
   - What we know: Currently it calls `onClose()` on success.
   - What's unclear: With inline save, closing after saving is surprising (user might want to stay on the tab).
   - Recommendation: Remove `onClose()` from `handleSaveCLIPaths`. Let the user close explicitly with the footer Close button. Show a brief success state instead (or just clear the dirty state silently).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (test: { environment: 'jsdom' }) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SETT-01 | Default tab "CLI Paths" is active on open | unit | `pnpm test -- SettingsPanel` | ❌ Wave 0 |
| SETT-01 | Clicking "Web Serving" tab shows web serving section, hides CLI paths | unit | `pnpm test -- SettingsPanel` | ❌ Wave 0 |
| SETT-01 | Clicking "CLI Paths" tab shows CLI paths section, hides web serving | unit | `pnpm test -- SettingsPanel` | ❌ Wave 0 |
| SETT-01 | Only one tab's content visible at a time | unit | `pnpm test -- SettingsPanel` | ❌ Wave 0 |
| SETT-02 | Footer renders single Close button (not Cancel + Save) | unit | `pnpm test -- SettingsPanel` | ❌ Wave 0 |
| SETT-02 | Close button has correct CSS class (secondary styling) | unit | `pnpm test -- SettingsPanel` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/SettingsPanel.test.tsx` — covers SETT-01 and SETT-02

*(Vitest, jsdom, and React testing infrastructure already configured. Only the test file itself is missing.)*

## Sources

### Primary (HIGH confidence)
- Direct read of `/Users/ken/dev/agenthub/frontend/src/components/SettingsPanel.tsx` — full component structure, all state variables, all handler functions
- Direct read of `/Users/ken/dev/agenthub/frontend/src/style.css` — all existing CSS, BEM class naming, color values, modal structure
- Direct read of `/Users/ken/dev/agenthub/frontend/src/App.tsx` — how SettingsPanel is used, props passed, close handler
- Direct read of `/Users/ken/dev/agenthub/frontend/src/components/__tests__/StatusBar.test.tsx` — canonical test pattern for this codebase
- Direct read of `/Users/ken/dev/agenthub/.planning/STATE.md` — Phase 8 decision: "Three states via JSX conditionals, not CSS display toggling"
- Direct read of `/Users/ken/dev/agenthub/frontend/package.json` — confirmed Vitest 4.1.0, no UI component libraries
- Direct read of `/Users/ken/dev/agenthub/frontend/vite.config.ts` — confirmed jsdom test environment

### Secondary (MEDIUM confidence)
- None required — all critical information sourced directly from project files

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — read directly from package.json and existing source
- Architecture: HIGH — tab pattern derived from existing codebase patterns (TabBar component, StatusBar conditional rendering)
- Pitfalls: HIGH — derived from reading actual component state structure and existing project decisions in STATE.md

**Research date:** 2026-03-19
**Valid until:** 2026-04-18 (stable — pure UI refactor, no external dependencies)
