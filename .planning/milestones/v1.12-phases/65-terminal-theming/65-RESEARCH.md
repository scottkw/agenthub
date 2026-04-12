# Phase 65: Terminal Theming - Research

**Researched:** 2026-04-11
**Domain:** xterm.js theme API + xterm-theme npm package + localStorage persistence + SettingsTab extension
**Confidence:** HIGH

## Summary

Phase 65 adds a theme selector to the Settings panel. Users pick from the `xterm-theme` library's 246 iTerm2 color schemes. The selected theme persists across restarts via `localStorage` (the established project pattern for frontend-only preferences). Theme changes apply live to all open terminal sessions without reload.

The implementation is purely frontend — no Go backend changes are needed. The pattern is almost identical to how `fontSize` is already handled: a global state in `App.tsx` is passed down to `TerminalPanel` as a prop, and `TerminalPanel` applies it via `terminal.options.theme = newTheme` in a `useEffect`. The Settings tab gains a third tab ("Appearance") containing the theme `<select>`.

xterm.js 6.0.0's `terminal.options.theme` setter applies the change immediately to a live terminal (no re-open required). A new object reference must be passed — not a mutation of the existing object — but the `xterm-theme` library already provides distinct immutable objects so this is trivially satisfied.

**Primary recommendation:** Install `xterm-theme@1.1.0`. Add global `terminalTheme` state to `App.tsx` initialized from `localStorage`. Pass `theme` prop to every `TerminalPanel`. In `TerminalPanel`, apply theme changes via `terminal.options.theme = theme` in a `useEffect([theme])`. Add an "Appearance" tab to `SettingsTab` with a `<select>` listing all theme names.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| THM-01 | User can select a terminal color theme from the full xterm-theme library | `xterm-theme@1.1.0` exports 246 named themes; all keys of the default export serve as a `<select>` option list. Adding an "Appearance" tab to `SettingsTab` with a `<select>` covers this requirement. |
| THM-02 | User's selected theme persists across app restarts | `localStorage.setItem('agenthub:terminalTheme', themeName)` on selection change, read with `localStorage.getItem` in `useState` initializer. Same pattern already used for sidebar collapsed state and session working directory. |
| THM-03 | Theme change applies immediately to all open terminal sessions | `terminal.options.theme = theme` in a `useEffect([theme])` in `TerminalPanel`. Since all panels render simultaneously (display:none for inactive), assigning to `options.theme` applies to all. No reload required. |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `xterm-theme` | 1.1.0 | 246 iTerm2 color themes compatible with xterm.js ITheme interface | Only version available; exact match to REQUIREMENTS.md THM-01 wording |
| `@xterm/xterm` | 6.0.0 | Terminal emulator with live `options.theme` setter | Already installed; `terminal.options.theme` setter confirmed in v6 docs |

### Supporting

None. No additional libraries needed.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `xterm-theme` npm package | Hand-curated theme JSON | REQUIREMENTS.md says "full xterm-theme library" — this is the named package |
| `localStorage` for persistence | Go backend `GetSetting`/`SetSetting` Wails binding | localStorage is the established project pattern for frontend-only prefs (Sidebar, NewSessionModal). No backend round-trip needed. |
| Single global theme state in App | Per-tab theme state | REQUIREMENTS.md and Out of Scope section explicitly exclude per-tab theme overrides |

**Installation:**
```bash
cd /Users/ken/dev/agenthub/frontend && pnpm add xterm-theme@1.1.0
```

**Version verification:** `npm view xterm-theme version` returns `1.1.0` (only release; published 2022-05-25) [VERIFIED: npm registry 2026-04-11]

## Architecture Patterns

### Recommended Project Structure

No new files or directories needed. Changes touch:

```
frontend/src/
├── App.tsx                          # add terminalTheme state + pass theme prop down
├── components/
│   ├── SettingsTab.tsx              # add "Appearance" tab with theme <select>
│   └── TerminalPanel.tsx            # add theme prop + useEffect([theme]) to apply it
└── components/__tests__/
    ├── App.test.tsx                 # add THM-02/03 state assertions
    ├── SettingsTab.test.tsx         # add THM-01 theme selector assertions
    └── TerminalPanel.test.tsx       # add THM-03 options.theme application assertions
```

### Pattern 1: Global Theme State in App (mirrors fontSize pattern)

**What:** `App.tsx` holds a single `terminalTheme` state (theme name string + theme object). On selection, it updates state and writes to localStorage. The theme object is passed to every `TerminalPanel` instance.

**When to use:** Any global terminal appearance setting that applies to all sessions.

**Example:**
```typescript
// Source: pattern mirroring App.tsx handleFontSizeChange / fontSizes state
import xtermThemes from 'xterm-theme'
import type { ITheme } from '@xterm/xterm'

const THEME_STORAGE_KEY = 'agenthub:terminalTheme'
const DEFAULT_THEME_NAME = 'Tomorrow_Night'

// In App():
const [terminalThemeName, setTerminalThemeName] = useState<string>(
  () => localStorage.getItem(THEME_STORAGE_KEY) ?? DEFAULT_THEME_NAME
)
const terminalTheme: ITheme = (xtermThemes as Record<string, ITheme>)[terminalThemeName]
  ?? (xtermThemes as Record<string, ITheme>)[DEFAULT_THEME_NAME]

const handleThemeChange = useCallback((name: string) => {
  localStorage.setItem(THEME_STORAGE_KEY, name)
  setTerminalThemeName(name)
}, [])
```

### Pattern 2: Live Theme Application in TerminalPanel

**What:** `TerminalPanel` receives a `theme` prop (ITheme object). A `useEffect` keyed on `theme` assigns it to `terminal.options.theme`. xterm.js immediately repaints the terminal with the new colors.

**When to use:** Applying any live terminal option change without re-creating the terminal instance.

**Example:**
```typescript
// Source: xterm.js official docs (xtermjs.org/docs) + mirrors fontSize effect in TerminalPanel.tsx line 180-184
// [VERIFIED: terminal.options.theme setter works in xterm.js v6; new object reference required]

// In TerminalPanel:
interface TerminalPanelProps {
  sessionId: string
  isActive: boolean
  relayPort: number
  fontSize: number
  onFontSizeChange: (delta: number) => void
  theme: ITheme   // NEW
}

// Effect:
useEffect(() => {
  if (!termRef.current) return
  termRef.current.options.theme = theme
}, [theme])
```

**Critical detail:** xterm.js v6 requires a new object reference for the theme to take effect. The `xterm-theme` library returns a distinct object for each theme name, so passing `xtermThemes[name]` directly satisfies this requirement. Do NOT mutate `termRef.current.options.theme` in-place.

### Pattern 3: Appearance Tab in SettingsTab

**What:** `SettingsTab` gains a third tab button ("Appearance") and a corresponding body panel with a `<select>` element populated from all keys of the xterm-theme default export.

**When to use:** Any new global appearance setting added to the app.

**Example:**
```typescript
// In SettingsTab:
// Import the theme name list:
import xtermThemes from 'xterm-theme'
const THEME_NAMES = Object.keys(xtermThemes).sort()

// Props addition:
interface SettingsTabProps {
  // ... existing props
  selectedTheme: string
  onThemeChange: (name: string) => void
}

// Tab button:
<button
  className={`settings-panel__tab-btn ${activeTab === 'appearance' ? 'settings-panel__tab-btn--active' : ''}`}
  onClick={() => setActiveTab('appearance')}
  role="tab"
  aria-selected={activeTab === 'appearance'}
>
  Appearance
</button>

// Tab body:
{activeTab === 'appearance' && (
  <div className="settings-panel__field-group">
    <label className="settings-panel__label">Terminal Theme</label>
    <select
      className="settings-panel__path-input"
      value={selectedTheme}
      onChange={(e) => onThemeChange(e.target.value)}
    >
      {THEME_NAMES.map(name => (
        <option key={name} value={name}>{name.replace(/_/g, ' ')}</option>
      ))}
    </select>
  </div>
)}
```

### Pattern 4: Passing Theme Through TerminalPanel Render Site

**What:** In `App.tsx` JSX, `TerminalPanel` receives the theme object as a prop.

**Example:**
```typescript
// In App.tsx — mirrors fontSize={fontSizes[tab.sessionId] ?? DEFAULT_FONT_SIZE} pattern
<TerminalPanel
  sessionId={tab.sessionId}
  isActive={isActive}
  relayPort={relayPort}
  fontSize={fontSizes[tab.sessionId] ?? DEFAULT_FONT_SIZE}
  onFontSizeChange={(delta) => handleFontSizeChange(tab.sessionId, delta)}
  theme={terminalTheme}   // NEW — same object for all sessions (global setting)
/>
```

### Anti-Patterns to Avoid

- **Per-session theme state (Record<string, ITheme>):** REQUIREMENTS.md explicitly puts per-tab theme overrides Out of Scope. Use a single global `terminalThemeName` / `terminalTheme`.
- **Mutating terminal.options.theme in-place:** xterm.js uses reference comparison; mutation does not trigger a repaint. Always assign a new object.
- **Fetching theme on every render:** Import the themes object once at module level; do not call `require('xterm-theme')` dynamically.
- **Storing the theme object in localStorage:** `localStorage` only stores strings. Store the theme _name_ string; resolve the object at runtime from the import.
- **Adding a Go backend method for theme persistence:** localStorage is the correct layer for frontend-only preferences in this codebase.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 246 terminal color themes | Custom theme editor or hardcoded palette | `xterm-theme` npm package | Already packages all iTerm2 themes in xterm.js-compatible format; THM-01 explicitly names this library |
| Theme persistence | Custom Wails binding (GetTheme/SetTheme) | `localStorage` | Established project pattern; no Go backend round-trip needed for a string preference |
| Sorted theme list | Custom sort logic | `Object.keys(xtermThemes).sort()` | Trivial one-liner; already available |

**Key insight:** The xterm.js `options.theme` setter eliminates the need to re-create or re-attach the terminal to change colors. All 246 themes are already packaged and ship with correct ITheme-shaped objects.

## Runtime State Inventory

> SKIPPED — This is a greenfield feature addition, not a rename/refactor/migration phase.

## Common Pitfalls

### Pitfall 1: Theme with missing background causes transparent terminal background
**What goes wrong:** Some themes in xterm-theme may not define a `background` property, leaving the terminal background transparent/unset and showing whatever is behind it.
**Why it happens:** ITheme properties are all optional; a theme that omits `background` lets the CSS background show through.
**How to avoid:** On theme change, also check that `theme.background` exists; if not, fall back to a safe default. Additionally, the `.terminal-session-container` CSS already sets `background-color: #1a1b26` (PAD-01), which would show through as a visible but acceptable fallback.
**Warning signs:** Terminal background appears as a different color from what the theme intends.

### Pitfall 2: xterm-theme default export is CommonJS-style in bundler context
**What goes wrong:** Depending on how `xterm-theme` packages its exports, some bundlers may struggle with `import xtermThemes from 'xterm-theme'` if the package has a CommonJS default.
**Why it happens:** xterm-theme 1.1.0 was published in 2022; its package.json may not have an `"exports"` field or ESM entry. Vite handles CJS interop, but the import syntax matters.
**How to avoid:** Use `import xtermThemes from 'xterm-theme'` (default import). If TypeScript complains, add `// @ts-expect-error` or `import * as xtermThemes from 'xterm-theme'`. Check the Vite dev build for any "Cannot find module" warnings after install. [ASSUMED: Vite handles xterm-theme CJS interop cleanly — verify after `pnpm add`]
**Warning signs:** Vite build error referencing xterm-theme module resolution.

### Pitfall 3: `<select>` with 246 options has performance concern at render time
**What goes wrong:** Rendering 246 `<option>` elements is trivial for the browser but should not be done in a tight render loop.
**Why it happens:** If `THEME_NAMES` is computed inside the component body (not at module level), it gets recreated on every render.
**How to avoid:** Compute `const THEME_NAMES = Object.keys(xtermThemes).sort()` once at module level, outside the component function.
**Warning signs:** Unnecessary re-renders when the Appearance tab is open.

### Pitfall 4: localStorage key collision
**What goes wrong:** If the storage key is not namespaced, it could collide with other apps or future keys in this app.
**Why it happens:** No global key registry exists.
**How to avoid:** Use the established `agenthub:` prefix pattern (already used by `agenthub:lastWorkDir` in NewSessionModal). Key: `agenthub:terminalTheme`.
**Warning signs:** Settings bleed between apps running on the same domain/origin (unlikely in Wails but good hygiene).

### Pitfall 5: Theme does not apply to terminals opened after theme selection
**What goes wrong:** If `terminalTheme` is not passed to `TerminalPanel` at creation time, newly opened sessions start with no theme, then jump to the selected theme after a render.
**Why it happens:** The `useEffect([sessionId])` in TerminalPanel creates the `Terminal` with a `theme` option — if that option is not the current theme object at creation time, the terminal initializes with wrong colors.
**How to avoid:** Pass `theme` as a constructor argument to `new Terminal({ ..., theme })` in TerminalPanel, AND also apply it in `useEffect([theme])` for live changes. The `sessionId` effect captures `theme` at creation time via the prop.
**Warning signs:** New tabs open with default black-on-white or wrong colors momentarily.

## Code Examples

### xterm-theme default export shape
```typescript
// Source: [VERIFIED: github.com/ysk2014/xterm-theme/blob/master/src/index.js]
// xterm-theme exports an object with 246 keys, each being an ITheme-compatible object.
// Example (Dracula):
const Dracula = {
  foreground:     '#f8f8f2',
  background:     '#1e1f29',
  cursor:         '#bbbbbb',
  black:          '#000000',
  brightBlack:    '#555555',
  red:            '#ff5555',
  brightRed:      '#ff5555',
  green:          '#50fa7b',
  brightGreen:    '#50fa7b',
  yellow:         '#f1fa8c',
  brightYellow:   '#f1fa8c',
  blue:           '#bd93f9',
  brightBlue:     '#bd93f9',
  magenta:        '#ff79c6',
  brightMagenta:  '#ff79c6',
  cyan:           '#8be9fd',
  brightCyan:     '#8be9fd',
  white:          '#bbbbbb',
  brightWhite:    '#ffffff',
}
// 19 properties: foreground, background, cursor + 8 standard + 8 bright
```

### xterm.js live theme application
```typescript
// Source: [CITED: xtermjs.org/docs/api/terminal/classes/terminal/]
// A new object reference is required — reference comparison determines if update fires.
termRef.current.options.theme = theme  // theme is a new ITheme object from xterm-theme
```

### localStorage persistence pattern (mirrors Sidebar.tsx)
```typescript
// Source: [VERIFIED: frontend/src/components/Sidebar.tsx lines 11, 28-36]
const STORAGE_KEY = 'agenthub:terminalTheme'
const DEFAULT_THEME_NAME = 'Tomorrow_Night'

const [terminalThemeName, setTerminalThemeName] = useState<string>(
  () => localStorage.getItem(STORAGE_KEY) ?? DEFAULT_THEME_NAME
)

const handleThemeChange = useCallback((name: string) => {
  localStorage.setItem(STORAGE_KEY, name)
  setTerminalThemeName(name)
}, [])
```

### SettingsTab activeTab state — how to add third tab
```typescript
// Source: [VERIFIED: frontend/src/components/SettingsTab.tsx line 34]
// Existing: const [activeTab, setActiveTab] = useState<'cli-paths' | 'web-server'>('cli-paths')
// New type union adds 'appearance':
const [activeTab, setActiveTab] = useState<'cli-paths' | 'web-server' | 'appearance'>('cli-paths')
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hard-coded `theme: { background: '#1a1b26' }` in Terminal constructor | Global theme prop from xterm-theme library | Phase 65 | User can switch themes; default theme can be any of 246 options |
| No per-session theme variation | Single global theme (all sessions share it) | Always (per-tab themes are Out of Scope) | Simple state model |

**Deprecated/outdated:**
- `theme: { background: '#1a1b26' }` (hard-coded in TerminalPanel constructor): replaced by the `theme` prop from App.tsx.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Vite handles xterm-theme CJS/ESM interop cleanly with `import xtermThemes from 'xterm-theme'` | Common Pitfalls #2 | Build error after `pnpm add`; workaround: `import * as xtermThemes from 'xterm-theme'` |
| A2 | `Tomorrow_Night` is a reasonable default theme (dark, familiar) | Code Examples | Aesthetic only; any theme name from the library is valid |

## Open Questions

1. **Default theme name**
   - What we know: The app currently hard-codes `background: '#1a1b26'` (Tokyo Night dark background). No existing default theme is defined.
   - What's unclear: Which theme matches closest to the current app colors? `Afterglow`, `Atom`, `Tomorrow_Night` are reasonable dark defaults.
   - Recommendation: Use `Tomorrow_Night` as the default; it is a well-known dark theme. If the planner or reviewer prefers another, it is a one-line change.

2. **Option label formatting**
   - What we know: Theme names use underscores (e.g., `Tomorrow_Night`, `Solarized_Dark`). The `<select>` could display raw keys or replace `_` with spaces.
   - What's unclear: Whether the user prefers underscored or space-separated names.
   - Recommendation: Display with underscores replaced by spaces for readability (`name.replace(/_/g, ' ')`); keep the key as the value.

## Environment Availability

> Step 2.6: No external service dependencies. xterm-theme is a pure npm package. Vite, pnpm, and Node.js are confirmed present from Phase 64.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| pnpm | Package install | ✓ | (project standard) | — |
| xterm-theme | THM-01 theme list | Not yet installed | 1.1.0 (latest) | — |
| @xterm/xterm | Theme application | ✓ | 6.0.0 | — |

**Missing dependencies with no fallback:**
- `xterm-theme` — must be installed via `pnpm add xterm-theme@1.1.0`

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
| THM-01 | SettingsTab has "Appearance" tab button | unit (source-text) | `pnpm test` | ✅ `SettingsTab.test.tsx` (add assertion) |
| THM-01 | Appearance tab contains a `<select>` element | unit (source-text) | `pnpm test` | ✅ `SettingsTab.test.tsx` (add assertion) |
| THM-01 | SettingsTab receives `selectedTheme` and `onThemeChange` props | unit (source-text) | `pnpm test` | ✅ `SettingsTab.test.tsx` (add assertion) |
| THM-02 | App.tsx reads `agenthub:terminalTheme` from localStorage on init | unit (source-text) | `pnpm test` | ✅ `App.test.tsx` (add assertion) |
| THM-02 | handleThemeChange writes to localStorage | unit (source-text) | `pnpm test` | ✅ `App.test.tsx` (add assertion) |
| THM-03 | TerminalPanel has `theme` prop in interface | unit (source-text) | `pnpm test` | ✅ `TerminalPanel.test.tsx` (add assertion) |
| THM-03 | TerminalPanel applies `options.theme = theme` in useEffect | unit (source-text) | `pnpm test` | ✅ `TerminalPanel.test.tsx` (add assertion) |
| THM-03 | TerminalPanel useEffect has `[theme]` dependency | unit (source-text) | `pnpm test` | ✅ `TerminalPanel.test.tsx` (add assertion) |

All test files already exist; this phase adds new `it()` blocks to existing `describe` suites.

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

None — existing test infrastructure covers all phase requirements. No new test files needed; assertions are added to existing test files.

## Security Domain

This phase handles no user credentials, makes no network requests, and introduces no server-side code. The only data persisted is a theme name string (one of 246 known values) written to `localStorage`.

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | no | Theme name comes from a controlled `<select>` populated from library keys; no free-text input |
| V6 Cryptography | no | — |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: /Users/ken/dev/agenthub/frontend/src/components/TerminalPanel.tsx] — `terminal.options.fontSize` setter pattern (lines 180-184); confirms `options.*` setter API is used in this codebase
- [VERIFIED: /Users/ken/dev/agenthub/frontend/src/components/Sidebar.tsx] — `localStorage.getItem/setItem` with `agenthub:`-prefixed keys; established persistence pattern
- [VERIFIED: /Users/ken/dev/agenthub/frontend/src/components/NewSessionModal.tsx] — localStorage for `agenthub:lastWorkDir` and `agenthub:args:*`; confirms namespacing convention
- [VERIFIED: /Users/ken/dev/agenthub/frontend/src/components/SettingsTab.tsx] — tab structure (`activeTab` state, `settings-panel__tab-btn` classes); confirms how to add a third tab
- [VERIFIED: /Users/ken/dev/agenthub/frontend/src/App.tsx] — `fontSizes` state + `handleFontSizeChange` + prop threading to `TerminalPanel`; blueprint for theme state
- [VERIFIED: /Users/ken/dev/agenthub/frontend/package.json] — `@xterm/xterm ^6.0.0` installed; `xterm-theme` not yet installed
- [VERIFIED: npm registry 2026-04-11] — `xterm-theme` latest = 1.1.0, published 2022-05-25
- [CITED: github.com/ysk2014/xterm-theme/blob/master/src/index.js] — 246 named exports + default export object; all themes imported from `./iterm/` directory
- [CITED: github.com/ysk2014/xterm-theme/blob/master/src/iterm/Dracula.js] — theme shape: 19 properties (foreground, background, cursor, 8 standard ANSI, 8 bright ANSI)
- [CITED: xtermjs.org/docs/api/terminal/classes/terminal/] — `terminal.options.theme` setter; new object reference required; applies immediately

### Secondary (MEDIUM confidence)
- [CITED: xtermjs.org/docs/api/terminal/interfaces/itheme/] — ITheme interface properties: foreground, background, cursor, cursorAccent, selectionBackground, selectionForeground, selectionInactiveBackground, ANSI colors 0-15, extendedAnsi array

### Tertiary (LOW confidence)
- [ASSUMED: A1] — Vite CJS interop for xterm-theme import
- [ASSUMED: A2] — `Tomorrow_Night` as default theme choice

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — xterm-theme npm registry verified; xterm.js options.theme API cited from official docs
- Architecture: HIGH — mirrors verified existing patterns (fontSize, localStorage, SettingsTab tabs)
- Pitfalls: MEDIUM — #2 (Vite CJS interop) is ASSUMED; others are HIGH derived from source code inspection
- Test patterns: HIGH — all existing test files confirmed present; source-text assertion pattern is established

**Research date:** 2026-04-11
**Valid until:** Stable — xterm.js 6.0.0 is locked in package.json; xterm-theme 1.1.0 is the only version
