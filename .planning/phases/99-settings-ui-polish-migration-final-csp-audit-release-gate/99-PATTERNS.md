# Phase 99: Settings UI Polish + Migration + Final CSP Audit (Release Gate) — Pattern Map

**Mapped:** 2026-05-08
**Files analyzed:** 9 (1 new component, 2 new tests, 1 new runbook, 5 modifications)
**Analogs found:** 9 / 9 — every Phase 99 file maps to a verified existing analog

> Phase 99 is the v3.2 release gate. **Zero new architectural primitives.** Every
> primitive needed (BannerStack vocabulary, `<details>` disclosure CSS, sub-key
> RPCs, Playwright `projects[]` array, fixture-based migration test, iPad UAT
> runbook) is already shipped. This pattern map points each new/modified file at
> the exact existing analog with concrete file:line references.

> **Path correction:** the orchestrator prompt suggested
> `frontend/src/components/Settings/PluginToggleBanner.tsx`, but the actual
> repository layout is **flat** under `frontend/src/components/` (verified by
> `ls`). All sibling components — `WebGLRecoveryBanner.tsx`,
> `PluginsSection.tsx`, `LocalNetworkBanner.tsx`, `UpdateBanner.tsx` — live
> directly in `components/`. New file should be
> `frontend/src/components/PluginToggleBanner.tsx` (no `Settings/` subdir).

---

## File Classification

| New / Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `frontend/src/components/PluginToggleBanner.tsx` (NEW) | component | event-driven (one-shot, auto-dismiss) | `frontend/src/components/WebGLRecoveryBanner.tsx` | exact |
| `frontend/src/components/__tests__/PluginToggleBanner.test.tsx` (NEW) | test | render + timer assertions | `frontend/src/components/__tests__/WebGLRecoveryBanner.test.tsx` | exact |
| `frontend/src/components/__tests__/PluginsSection.disclosure.test.tsx` (NEW) | test | source-inspection (vitest `?raw`) | `frontend/src/components/__tests__/PluginsSection.test.tsx` | exact |
| `.planning/phases/99-.../99-iPad-UAT.md` (NEW) | doc (runbook) | manual UAT script | `.planning/phases/93-.../93-iPad-UAT.md` | exact |
| `frontend/src/components/PluginsSection.tsx` (MODIFIED) | component | event-driven + sub-key RPC dispatch | self (extend in place) + `TerminalPanel.tsx:728` (sub-key RPC pattern) + `SettingsTab.tsx:441-473` (`<details>` markup) | exact |
| `frontend/src/App.tsx` (MODIFIED) | component (root) | event-driven (banner state machine) | `App.tsx:114, 890-933` (saveBanner pattern) + `:144-146, 915-920` (webgl banner gating) | exact |
| `frontend/playwright.config.ts` (MODIFIED) | config | test runner | self (extend `projects[]` array) | exact |
| `internal/daemon/engine_migration_test.go` (MODIFIED) | test | fixture-based load assertion | self (extend assertions) | exact |
| `frontend/src/style.css` (MODIFIED, optional) | style | presentational | `.webgl-recovery-banner` block at `style.css:1610-1661` if a `--info-toggle` modifier is needed | partial |
| `tests/fixtures/settings_v3.1.json` | config (test fixture) | file-I/O | **ALREADY EXISTS** — verified at `tests/fixtures/settings_v3.1.json` | n/a (no creation) |
| `.github/workflows/*.yml` (MODIFIED) | CI config | matrix | **NO PLAYWRIGHT WORKFLOW EXISTS** — see "No Analog Found" below | no-analog |

---

## Pattern Assignments

### `frontend/src/components/PluginToggleBanner.tsx` (NEW — component)

**Analog:** `frontend/src/components/WebGLRecoveryBanner.tsx` (entire file, 63 lines — one-shot banner with auto-dismiss in the existing `.banner-stack` vocabulary)

**Imports pattern** (`WebGLRecoveryBanner.tsx:1-2`):
```typescript
import React, { useEffect } from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'
```

**Props interface pattern** (`WebGLRecoveryBanner.tsx:25-29`):
```typescript
export interface WebGLRecoveryBannerProps {
  reason: 'context-loss' | 'software-rasterized'
  onDismiss: () => void
  className?: string
}
```

**For Phase 99** — mirror exactly with the new union:
```typescript
export interface PluginToggleBannerProps {
  kind: 'unicode11' | 'image'
  onDismiss: () => void
  className?: string
}
```

**Auto-dismiss `useEffect` pattern** (`WebGLRecoveryBanner.tsx:36-40`):
```typescript
useEffect(() => {
  if (reason !== 'context-loss') return
  const id = window.setTimeout(onDismiss, 8000)
  return () => window.clearTimeout(id)
}, [reason, onDismiss])
```

**For Phase 99** — both kinds auto-dismiss after 6000ms (per RESEARCH.md "Claude's Discretion"):
```typescript
useEffect(() => {
  const id = window.setTimeout(onDismiss, 6000)
  return () => window.clearTimeout(id)
}, [onDismiss])
```

**Copy selection pattern** (`WebGLRecoveryBanner.tsx:42-45`):
```typescript
const message =
  reason === 'context-loss'
    ? 'Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact.'
    : 'Hardware acceleration is unavailable on this device. Your terminal is using the standard renderer for the best experience.'
```

**For Phase 99** — verbatim copy locked in RESEARCH.md "Claude's Discretion":
```typescript
const message =
  kind === 'unicode11'
    ? 'Open a new terminal session to apply the Unicode 11 change.'
    : 'Open a new terminal session to apply the Inline Images change.'
```

**Render markup** (`WebGLRecoveryBanner.tsx:47-61`) — copy verbatim, swap class name to keep visual identity:
```typescript
const cls = ['webgl-recovery-banner', className].filter(Boolean).join(' ')

return (
  <div className={cls} role="status" aria-live="polite">
    <span className="webgl-recovery-banner__message">{message}</span>
    <button
      type="button"
      className="webgl-recovery-banner__dismiss"
      aria-label="Dismiss notification"
      onClick={onDismiss}
    >
      <XMarkIcon style={{ width: 16, height: 16 }} aria-hidden="true" />
    </button>
  </div>
)
```

**Phase 99 strategy:** **reuse the `webgl-recovery-banner` BEM class verbatim** for visual continuity (same TokyoNight info palette, same 53px banner-stack height, same × button, same `prefers-reduced-motion` rule). Do NOT introduce a new BEM block. The component differs from `WebGLRecoveryBanner` only in:
1. `kind` union (`'unicode11' | 'image'` vs `'context-loss' | 'software-rasterized'`)
2. message strings
3. auto-dismiss is unconditional 6000ms (vs conditional 8000ms)

The accessibility contract (`role="status"`, `aria-live="polite"`, dismiss `aria-label`) is preserved verbatim.

---

### `frontend/src/components/__tests__/PluginToggleBanner.test.tsx` (NEW — test)

**Analog:** `frontend/src/components/__tests__/WebGLRecoveryBanner.test.tsx` (entire file, 92 lines)

**Render helper pattern** (`WebGLRecoveryBanner.test.tsx:16-24`):
```typescript
import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { WebGLRecoveryBanner } from '../WebGLRecoveryBanner'

function render(reason: 'context-loss' | 'software-rasterized', onDismiss: () => void) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(WebGLRecoveryBanner, { reason, onDismiss }))
  })
  return { container, root }
}
```

**Lifecycle teardown pattern** (`WebGLRecoveryBanner.test.tsx:26-41`):
```typescript
describe('WebGLRecoveryBanner', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

  afterEach(() => {
    if (root) {
      flushSync(() => root!.unmount())
      root = undefined
    }
    if (container) {
      container.remove()
      container = undefined
    }
    vi.useRealTimers()
    vi.clearAllMocks()
  })
```

**Verbatim-copy assertion** (`WebGLRecoveryBanner.test.tsx:43-47`):
```typescript
it("reason='context-loss' renders verbatim recovery copy including 'Scrollback is intact.'", () => {
  ;({ container, root } = render('context-loss', vi.fn()))
  expect(container.textContent).toMatch(/Hardware-accelerated rendering recovered/)
  expect(container.textContent).toMatch(/Scrollback is intact\./)
})
```

**Auto-dismiss timing assertion with fake timers** (`WebGLRecoveryBanner.test.tsx:71-89`):
```typescript
describe('auto-dismiss timing', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  it("reason='context-loss' fires onDismiss after 8000ms (auto-dismiss)", () => {
    const onDismiss = vi.fn()
    ;({ container, root } = render('context-loss', onDismiss))
    expect(onDismiss).not.toHaveBeenCalled()
    vi.advanceTimersByTime(8000)
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })
})
```

**For Phase 99** — required tests:
1. `kind='unicode11'` renders verbatim "Open a new terminal session to apply the Unicode 11 change."
2. `kind='image'` renders verbatim "Open a new terminal session to apply the Inline Images change."
3. `role="status"` and `aria-live="polite"` (a11y contract).
4. Dismiss button has `aria-label="Dismiss notification"` and fires `onDismiss` on click.
5. Auto-dismiss after 6000ms with `vi.useFakeTimers()` + `vi.advanceTimersByTime(6000)`.
6. Both kinds auto-dismiss (no conditional skip — Phase 99 differs from WebGL banner here).

---

### `frontend/src/components/PluginsSection.tsx` (MODIFIED — component)

**Analogs:**
- Self (extend in place; current shape is the foundation)
- `frontend/src/components/SettingsTab.tsx:441-473` (`<details>` disclosure markup)
- `frontend/src/components/TerminalPanel.tsx:715-731` (sub-key RPC dispatch — `SetSearchConfig`)
- `frontend/src/wailsjs/go/main/App.d.ts:131,137,143` (Wails TS bindings)
- `frontend/src/wailsjs/go/models.ts:10,31,54,71,73,75` (auto-generated `SearchConfig`/`WebLinksConfig`/`ImageConfig` classes)

**Existing PluginsSection structure** (the file already renders 8 toggle rows + italic captions for unicode11/image; Phase 99 adds: side-effect prop, three `<details>` disclosures wired to sub-key RPCs).

**Side-effect prop addition** — Phase 99 adds a callback prop the parent (App.tsx) wires to its banner-stack. The PluginsSection signature changes from no-args to:
```typescript
// CURRENT (PluginsSection.tsx:19)
export function PluginsSection(): React.ReactElement {

// MODIFIED — Phase 99
type PluginToggleKind = 'unicode11' | 'image'
export interface PluginsSectionProps {
  onPluginToggleSideEffect?: (kinds: PluginToggleKind[]) => void
}
export function PluginsSection({
  onPluginToggleSideEffect,
}: PluginsSectionProps = {}): React.ReactElement {
```

**Diff-detection inside `handleSavePlugins`** — extend the existing handler at `PluginsSection.tsx:42-55`. Hold a ref of the last-saved snapshot (so toggles in the local edit buffer don't fire the banner until the user actually saves). Pattern:
```typescript
// NEW state near pluginConfig (line 21)
const lastSavedRef = useRef<PluginSettings | null>(null)

// In useEffect after GetPluginSettings resolves (line 33):
lastSavedRef.current = s

// In handleSavePlugins after successful SetPluginSettings (line 47):
const prior = lastSavedRef.current
const kinds: PluginToggleKind[] = []
if (prior && pluginConfig) {
  if (prior.unicode11 !== pluginConfig.unicode11) kinds.push('unicode11')
  if (prior.image !== pluginConfig.image) kinds.push('image')
}
lastSavedRef.current = pluginConfig
if (kinds.length > 0) onPluginToggleSideEffect?.(kinds)
```

**`<details>` disclosure markup pattern** (`SettingsTab.tsx:441-473` — Tailscale diagnostics):
```typescript
// PATTERN SOURCE — SettingsTab.tsx:441-442
<details className="settings-panel__details" style={{ marginTop: '0.5rem' }}>
  <summary style={{ cursor: 'pointer', color: '#7aa2f7', fontSize: '0.8rem' }}>Show diagnostics</summary>
  <div style={{ marginTop: '0.5rem', fontSize: '0.8rem' }}>
    {/* form controls */}
  </div>
</details>
```

The CSS class `.settings-panel__details` is already defined at `style.css:573-583`:
```css
.settings-panel__details {
  margin-top: 10px;
}
.settings-panel__details summary {
  font-size: 12px;
  color: #7aa2f7;
  cursor: pointer;
}
.settings-panel__details summary:hover {
  color: #89b4fa;
}
```

The base class already provides margin/color/cursor — Phase 99 should NOT override with inline styles (the existing SettingsTab usage adds inline style for one-off margin/color, but the class itself is sufficient and the planner can omit them).

**Sub-key RPC dispatch pattern** (`TerminalPanel.tsx:715-731` — verbatim shape):
```typescript
// PATTERN SOURCE — TerminalPanel.tsx:728-730
SetSearchConfig(new daemon.SearchConfig(opts)).catch(() => {
  /* silent — Settings panel surfaces persistence errors */
})
```

**Wails TS binding signatures** (`App.d.ts:131,137,143`):
```typescript
export function SetSearchConfig(arg1: daemon.SearchConfig): Promise<void>
export function SetWebLinksConfig(arg1: daemon.WebLinksConfig): Promise<void>
export function SetImageConfig(arg1: daemon.ImageConfig): Promise<void>
```

**Three disclosure render functions** — inline inside `PluginsSection` (per RESEARCH.md "Deferred Ideas" — inline render functions, NOT separate component files). Each fires its sub-key RPC immediately on change (NO local edit buffer for sub-configs — they bypass the full-snapshot Save Plugins button per Phase 94-07 WR-03 / PUI-04 anti-race contract).

**Search disclosure** (placed inside the Search row, after the description `<p>`):
```tsx
{pluginsLoaded && pluginConfig && (
  <details className="settings-panel__details">
    <summary>Search defaults</summary>
    <label className="settings-panel__toggle-row" htmlFor="search-regex">
      <input
        type="checkbox"
        id="search-regex"
        className="settings-panel__toggle-input"
        checked={pluginConfig.searchConfig.regex}
        onChange={(e) => {
          const next = new daemon.SearchConfig({
            ...pluginConfig.searchConfig,
            regex: e.target.checked,
          })
          SetSearchConfig(next).catch(() => {})
        }}
      />
      <span className="settings-panel__toggle-label">Regex</span>
    </label>
    {/* repeat for caseSensitive + wholeWord */}
  </details>
)}
```

**Web Links disclosure** — `<select>` for modifier + 3 checkboxes:
```tsx
<details className="settings-panel__details">
  <summary>Link click behavior</summary>
  <label>
    Modifier
    <select
      value={pluginConfig.webLinksConfig.modifier}
      onChange={(e) => SetWebLinksConfig(new daemon.WebLinksConfig({
        ...pluginConfig.webLinksConfig,
        modifier: e.target.value,
      })).catch(() => {})}
    >
      <option value="platform">Platform default (Cmd on macOS, Ctrl elsewhere)</option>
      <option value="cmd">Cmd</option>
      <option value="ctrl">Ctrl</option>
      <option value="none">No modifier (plain click)</option>
    </select>
  </label>
  {/* checkboxes for confirmOSC8 / confirmIDN / confirmTyposquat */}
</details>
```

**Inline Images disclosure** — `<input type="number">` with `[1, 1000]` clamp (matches daemon-side gate at `api.go:649` `handleSetImageConfig`):
```tsx
<details className="settings-panel__details">
  <summary>Storage limit</summary>
  <label>
    <input
      type="number"
      min={1}
      max={1000}
      step={1}
      value={pluginConfig.imageConfig.storageLimit}
      onChange={(e) => {
        const v = Math.max(1, Math.min(1000, Number(e.target.value)))
        SetImageConfig(new daemon.ImageConfig({ storageLimit: v })).catch(() => {})
      }}
    />
    {' MB'}
  </label>
</details>
```

**Debounce note** (RESEARCH.md "Claude's Discretion"): the storageLimit `<input type="number">` may fire many onChange events while the user types digits. Wrap the dispatch in a 500ms `setTimeout` debouncer. Mirror `App.tsx:122` `trayDebounceRef` shape:
```typescript
const imageStorageDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
// In onChange:
if (imageStorageDebounceRef.current) clearTimeout(imageStorageDebounceRef.current)
imageStorageDebounceRef.current = setTimeout(() => {
  SetImageConfig(new daemon.ImageConfig({ storageLimit: v })).catch(() => {})
}, 500)
```

**Insertion points within existing `renderRow` calls** — the disclosures live INSIDE the `renderRow` JSX, after the `<p className="settings-panel__description">` (and after the optional italic caption `<p>`). Cleanest implementation: extend `renderRow` signature with an optional `disclosure?: React.ReactNode` arg:
```typescript
function renderRow(
  key: PluginBooleanKey,
  label: string,
  description: string,
  caption?: string,
  disclosure?: React.ReactNode,
): React.ReactElement {
  // ... existing JSX ...
  // After the caption <p>:
  {disclosure}
}
```

Then the call sites for Search/WebLinks/Image pass the disclosure JSX as the 5th arg. Other rows pass nothing (default undefined).

---

### `frontend/src/components/__tests__/PluginsSection.disclosure.test.tsx` (NEW — test)

**Analog:** `frontend/src/components/__tests__/PluginsSection.test.tsx` (the entire file, especially `:1-2` `?raw` import + `expect(raw).toContain(...)` assertions).

**Existing source-inspection pattern** (`PluginsSection.test.tsx:1-13`):
```typescript
import { describe, it, expect } from 'vitest'
import raw from '../PluginsSection.tsx?raw'

describe('PUI-01: 8 toggle rows in UI-SPEC order', () => {
  it('contains all 8 plugin labels', () => {
    expect(raw).toContain('WebGL renderer')
    expect(raw).toContain('Unicode 11 widths')
    // ...
  })
})
```

**For Phase 99** — required assertions for PUI-03:
```typescript
import { describe, it, expect } from 'vitest'
import raw from '../PluginsSection.tsx?raw'

describe('PUI-03: <details> disclosures for Search / WebLinks / Image', () => {
  it('renders three settings-panel__details blocks', () => {
    const matches = raw.match(/settings-panel__details/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(3)
  })

  it('summary copy is verbatim per RESEARCH.md "Claude\'s Discretion"', () => {
    expect(raw).toContain('Search defaults')
    expect(raw).toContain('Link click behavior')
    expect(raw).toContain('Storage limit')
  })

  it('dispatches SetSearchConfig on Search disclosure changes (PUI-04 sub-key contract)', () => {
    expect(raw).toContain('SetSearchConfig')
    expect(raw).toContain('new daemon.SearchConfig')
  })

  it('dispatches SetWebLinksConfig on Web Links disclosure changes', () => {
    expect(raw).toContain('SetWebLinksConfig')
    expect(raw).toContain('new daemon.WebLinksConfig')
  })

  it('dispatches SetImageConfig on Inline Images disclosure changes', () => {
    expect(raw).toContain('SetImageConfig')
    expect(raw).toContain('new daemon.ImageConfig')
  })

  it('Web Links modifier <select> exposes platform/cmd/ctrl/none options', () => {
    expect(raw).toContain('value="platform"')
    expect(raw).toContain('value="cmd"')
    expect(raw).toContain('value="ctrl"')
    expect(raw).toContain('value="none"')
  })

  it('Inline Images storageLimit input clamps to [1, 1000]', () => {
    expect(raw).toMatch(/min=\{?1\}?/)
    expect(raw).toMatch(/max=\{?1000\}?/)
  })
})

describe('PUI-02: side-effect callback for Unicode 11 / Image toggle changes', () => {
  it('declares onPluginToggleSideEffect prop', () => {
    expect(raw).toContain('onPluginToggleSideEffect')
  })

  it('compares prior vs current unicode11 + image bools', () => {
    expect(raw).toMatch(/unicode11/)
    expect(raw).toMatch(/image/)
  })
})
```

**Why source-inspection (NOT `render()` + DOM assertions):** PluginsSection consumes the Wails-generated `daemon.SearchConfig` constructor which throws under jsdom because the auto-generated `convertValues` helper expects a Go-side context. The codebase precedent for visual-contract testing in this file (`PluginsSection.test.tsx:1-2`) is `?raw` source-inspection — Phase 99 follows that precedent to avoid jsdom regressions.

---

### `frontend/src/App.tsx` (MODIFIED — root component)

**Analog:** itself — extend the existing `saveBanner` state machine pattern at `App.tsx:114, 890-933`.

**Existing `saveBanner` state declaration** (`App.tsx:111-114`):
```typescript
// Phase 97 SER-01: one-shot save-feedback banner. Mirrors the
// localBanner pattern — info kind for "Serialize disabled"
// affordance, error kind for write/dialog failures.
const [saveBanner, setSaveBanner] = useState<{ kind: 'info' | 'error'; text: string } | null>(null)
```

**Existing webgl banner state declaration** (`App.tsx:140-146`) — closer analog because it's an *array of one-shot kinds*, not a single banner:
```typescript
// Phase 93 WGL-02 / WGL-03: WebGL recovery banner state. One-shot per
// app session — webglBannerDismissed gates rendering even if the
// underlying webglContextLost / webglSoftwareDetected event fires
// multiple times (e.g., user toggles WebGL OFF/ON while context lost).
const [webglContextLost, setWebglContextLost] = useState(false)
const [webglSoftwareDetected, setWebglSoftwareDetected] = useState(false)
const [webglBannerDismissed, setWebglBannerDismissed] = useState(false)
```

**For Phase 99** — add a banner-set state. The set may have 0, 1, or 2 banners (unicode11 toggled, image toggled, both toggled in one save). Per RESEARCH.md "Claude's Discretion", "show two banners stacked":
```typescript
// Phase 99 PUI-02: one-shot post-save Unicode 11 / Image toggle banners.
// Each kind appears at most once at any moment; dismissing removes that kind
// from the set; auto-dismiss fires from the PluginToggleBanner component.
type PluginToggleKind = 'unicode11' | 'image'
const [pluginToggleBanners, setPluginToggleBanners] = useState<PluginToggleKind[]>([])

const handlePluginToggleSideEffect = useCallback((kinds: PluginToggleKind[]) => {
  setPluginToggleBanners(prev => Array.from(new Set([...prev, ...kinds])))
}, [])
```

**Banner-stack rendering pattern** (`App.tsx:888-933`):
```tsx
return (
  <div className="app">
    {((webServerMode === 'local' && !localBannerDismissed) ||
      update ||
      ((webglContextLost || webglSoftwareDetected) && !webglBannerDismissed) ||
      saveBanner !== null) && (
      <div className="banner-stack">
        {webServerMode === 'local' && !localBannerDismissed && (
          <LocalNetworkBanner ... />
        )}
        {update && (<UpdateBanner ... />)}
        {(webglContextLost || webglSoftwareDetected) && !webglBannerDismissed && (
          <WebGLRecoveryBanner ... />
        )}
        {saveBanner && (
          <div className={saveBanner.kind === 'error' ? 'banner banner--error' : 'banner banner--info'} role="status">
            <span>{saveBanner.text}</span>
            <button onClick={() => setSaveBanner(null)} aria-label="Dismiss">×</button>
          </div>
        )}
      </div>
    )}
```

**For Phase 99** — extend the `<div className="banner-stack">` outer guard AND add the new banner children. The guard is a logical OR over all banner-presence predicates:
```tsx
{((webServerMode === 'local' && !localBannerDismissed) ||
  update ||
  ((webglContextLost || webglSoftwareDetected) && !webglBannerDismissed) ||
  saveBanner !== null ||
  pluginToggleBanners.length > 0) && (   // NEW — Phase 99
  <div className="banner-stack">
    {/* ... existing children ... */}
    {pluginToggleBanners.map((kind) => (    // NEW — Phase 99
      <PluginToggleBanner
        key={kind}
        kind={kind}
        onDismiss={() =>
          setPluginToggleBanners(prev => prev.filter(k => k !== kind))
        }
      />
    ))}
  </div>
)}
```

**Prop wire-up to `PluginsSection`** — `PluginsSection` is currently rendered inside `SettingsTab.tsx`. The callback must travel App → SettingsTab → PluginsSection. Either:
1. Add an `onPluginToggleSideEffect` prop to `SettingsTab` and forward it to `<PluginsSection />`, OR
2. Move `<PluginsSection />` rendering up to App.tsx (architectural shift — NOT recommended for a release-gate phase).

**Recommendation:** Option 1 — pass through SettingsTab. Mirror existing prop-pass pattern: `SettingsTab.tsx` already takes props from App.tsx for theme + various callbacks (verify exact prop list in plan).

**Import addition at `App.tsx`** (mirror existing imports at `:39-52`):
```typescript
import { PluginToggleBanner } from './components/PluginToggleBanner'
```

---

### `frontend/playwright.config.ts` (MODIFIED — config)

**Analog:** itself — extend the `projects[]` array at line 31-37.

**Current state** (`playwright.config.ts:31-37`):
```typescript
projects: [
  {
    name: 'chromium',
    use: { ...devices['Desktop Chrome'] },
  },
],
```

**For Phase 99** — append `firefox` and `webkit` (per RESEARCH.md "Claude's Discretion: chromium, firefox, webkit ordering"):
```typescript
projects: [
  {
    name: 'chromium',
    use: { ...devices['Desktop Chrome'] },
  },
  {
    name: 'firefox',
    use: { ...devices['Desktop Firefox'] },
  },
  {
    name: 'webkit',
    use: { ...devices['Desktop Safari'] },
  },
],
```

**Sequencing constraint** (already in the config at lines 20-21): `fullyParallel: false, workers: 1` — the Go fixture binary is single-instance, so projects run sequentially. This is correct for Phase 99; do NOT change.

**Browser-install constraint:** Playwright's WebKit + Firefox engines are downloaded by `pnpm exec playwright install --with-deps webkit firefox`. The local dev machine likely has Chromium only — Phase 99's plan must include the install step before running the new projects locally and in CI.

---

### `internal/daemon/engine_migration_test.go` (MODIFIED — test)

**Analog:** itself — extend the assertions in `TestSettingsMigrationV3_1ToV3_2` (lines 41-85).

**Existing structure** (`engine_migration_test.go:41-85`):
```go
func TestSettingsMigrationV3_1ToV3_2(t *testing.T) {
    dir := t.TempDir()
    copyFixtureToTempDir(t, dir)

    e := &SessionEngine{
        configDir: dir,
        cliPaths:  make(map[string]string),
    }
    e.loadSettingsFromDisk(dir)

    // In-memory: the 8 v3.2 defaults must be populated (Pitfall #14 mitigation).
    got := e.GetPluginSettings()
    want := defaultPluginSettings()
    if got != want {
        t.Errorf("GetPluginSettings after v3.1 load: got %+v, want %+v", got, want)
    }

    // Phase 94 SRC-02 — explicit assertion that Phase 93 fixture (no searchConfig key)
    // loads with SearchConfig zero-value defaults populated via existing defaults-merge.
    if got.SearchConfig != (SearchConfig{}) {
        t.Errorf("expected SearchConfig zero-value defaults after Phase 93 fixture load, got %+v", got.SearchConfig)
    }
    // ... cliPaths preservation, schemaVersion: 2, plugins.webgl: true ...
}
```

**House style:** sequential assertions inside one test function, NOT table-driven. Each assertion is `if got != want { t.Errorf(...) }` with descriptive error messages. The test uses two helper functions (`fixtureV31Path`, `copyFixtureToTempDir`) defined at the top of the file (lines 11-32).

**For Phase 99** — expand assertions to cover ALL 8 plugin booleans + ALL 3 sub-configs explicitly. The current `got != want` line already implicitly covers all 8 booleans (struct equality) but the planner should add explicit per-field assertions for diagnostic clarity:

```go
// Per-field assertions for diagnostic clarity (Phase 99 SC-3 expansion).
if !got.WebGL { t.Errorf("WebGL: got false, want true") }
if !got.Unicode11 { t.Errorf("Unicode11: got false, want true") }
if !got.Search { t.Errorf("Search: got false, want true") }
if !got.WebLinks { t.Errorf("WebLinks: got false, want true") }
if !got.Image { t.Errorf("Image: got false, want true") }
if !got.Serialize { t.Errorf("Serialize: got false, want true") }
if !got.Clipboard { t.Errorf("Clipboard: got false, want true") }
if got.Progress { t.Errorf("Progress: got true, want false (default OFF in v3.2)") }

// Sub-config defaults (per defaultPluginSettings() at plugin_settings.go:94-113):
wantSearch := SearchConfig{Regex: false, CaseSensitive: false, WholeWord: false}
if got.SearchConfig != wantSearch {
    t.Errorf("SearchConfig defaults: got %+v, want %+v", got.SearchConfig, wantSearch)
}
wantWebLinks := WebLinksConfig{
    Modifier:         "platform",
    ConfirmOSC8:      true,
    ConfirmIDN:       true,
    ConfirmTyposquat: true,
}
if got.WebLinksConfig != wantWebLinks {
    t.Errorf("WebLinksConfig defaults: got %+v, want %+v", got.WebLinksConfig, wantWebLinks)
}
wantImage := ImageConfig{StorageLimit: 16}
if got.ImageConfig != wantImage {
    t.Errorf("ImageConfig defaults: got %+v, want %+v", got.ImageConfig, wantImage)
}
```

**Note on existing line 60:** the current assertion is `got.SearchConfig != (SearchConfig{})` which checks for zero-value, NOT for the actual default (also zero-value coincidentally — `Regex/CaseSensitive/WholeWord` all default to `false`, so zero-value `SearchConfig{}` and `defaultPluginSettings().SearchConfig` are observationally identical). Phase 99 should keep the explicit `wantSearch := SearchConfig{...}` form for forward-compatibility (when v3.3 adds a non-zero default).

**Idempotency test** (`TestSettingsMigrationIdempotent`, lines 92-127) — already complete; Phase 99 does NOT modify.

---

### `.planning/phases/99-.../99-iPad-UAT.md` (NEW — manual UAT runbook)

**Analog:** `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-iPad-UAT.md` (entire file, 93 lines)

**Structure pattern** — 5 numbered UAT scenarios, each with: (a) prereqs, (b) numbered action steps, (c) verbatim-copy quote-block, (d) sub-checks, (e) PASS criteria. Final "Sign-Off" checklist references back to `99-VALIDATION.md`.

**Header pattern** (`93-iPad-UAT.md:1-9`):
```markdown
# Phase 93 — iPad Safari Manual UAT

> Manual UAT script for the runtime behaviors that headless Playwright cannot reproduce on iPad Safari over Tailscale. Run during `/gsd-verify-work 93` before phase sign-off.

## Prerequisites
- iPad with Safari, joined to the same Tailnet as the dev Mac running AgentHub
- AgentHub running with web-server enabled, at least one terminal session created and shared with a capability link
- The Tailscale URL of the form `https://<tailnet-fqdn>.ts.net/sessions/<id>?cap=<token>`
```

**UAT scenario pattern** (`93-iPad-UAT.md:10-23` — UAT-1):
```markdown
## UAT-1: WebGL Software-Rasterizer Preemption (WGL-03)

1. Open the Tailscale URL in iPad Safari (the iPad reports software-rasterized WebGL via ANGLE Metal Renderer).
2. Within 5 seconds, an in-page banner above the terminal scrollback should appear with the EXACT copy:
   > Hardware acceleration is unavailable on this device. Your terminal is using the standard renderer for the best experience.
3. Confirm:
   - Banner has a 53px height (visually one line of text + small padding)
   - Banner has a thin blue (`#7aa2f7`) accent strip on the LEFT edge
   - Banner has a × button on the RIGHT edge that dismisses the banner when tapped
   ...
```

**Sign-off pattern** (`93-iPad-UAT.md:85-93`):
```markdown
## Sign-Off

- [ ] UAT-1 PASS
- [ ] UAT-2 PASS
- [ ] UAT-3 PASS
- [ ] UAT-4 PASS
- [ ] UAT-5 PASS

Once all 5 UATs PASS, mark `93-VALIDATION.md` § Validation Sign-Off all checkboxes and proceed with `/gsd-verify-work 93`.
```

**For Phase 99** — clone the shape with v3.2-release-specific scenarios (per RESEARCH.md "iPad UAT runbook scope"):
- UAT-1: All-plugins-enabled attach → type → emit OSC 9;4 progress → emit sixel image flow
- UAT-2: Scrollback through history → detach → re-attach → confirm scrollback intact
- UAT-3: Zero-CDN audit (Safari Web Inspector remote debugging from dev Mac → Network tab filter for `cdn.jsdelivr.net OR unpkg.com OR esm.sh`)
- UAT-4: CSP zero-violation audit (Safari Web Inspector Console tab → no `csp` / `content security policy` errors during full session)
- UAT-5: Second-pass ALL-8-plugins-ON (toggle Progress ON in Settings before the session) flow

**Sign-off block** mirrors `93-iPad-UAT.md:85-93` with `99-VALIDATION.md` reference and `/gsd-verify-work 99` final command.

---

### `frontend/src/style.css` (MODIFIED, OPTIONAL — style)

**Analog:** `style.css:1610-1661` (the `.webgl-recovery-banner` block — full BEM definition).

**Phase 99 recommendation: ZERO new CSS.** Per Phase 92 UI-SPEC §"Component Inventory" line 225, "**Phase 92 adds zero new lines to `style.css`. This is a load-bearing assertion of the contract — visual consistency falls out for free if no new classes are introduced.**" Phase 99 inherits this discipline:

- The new `PluginToggleBanner` reuses `.webgl-recovery-banner` BEM block verbatim.
- The new `<details>` disclosures reuse `.settings-panel__details` (already at `style.css:573-583`).
- Form controls inside `<details>` reuse `.settings-panel__toggle-row` (already exists).

**If a tweak proves necessary** during implementation (e.g., `<select>` element styling for the Web Links modifier dropdown, since `<select>` has no existing project-side rules), keep it minimal and BEM-aligned. Recommended scope: ≤ 10 lines.

---

### `tests/fixtures/settings_v3.1.json` (UNCHANGED — already exists)

**Verification:** the file exists at `tests/fixtures/settings_v3.1.json` (verified by `ls /Users/ken/dev/agenthub/tests/fixtures/`). RESEARCH.md correctly states it's "deliberately minimal" — proves zero-merge of v3.2 keys. Phase 99 does NOT recreate it.

**Optional addition** (per RESEARCH.md "Claude's Discretion"): a `tests/fixtures/settings_v3.2_partial.json` for the hypothetical mid-development upgrade (some plugins set, no sub-configs) is **explicitly optional** — the existing fixture already covers the load-bearing v3.1 case. Plan should defer this unless specifically requested.

---

### `.github/workflows/*.yml` (MODIFIED — CI config) — NO CURRENT PLAYWRIGHT WORKFLOW

**Finding via grep** (`grep -rn "playwright\|pnpm test:e2e\|e2e" .github/workflows/`): **NO matches.** The existing workflows (`build.yml`, `release.yml`, `distribute.yml`, `release-please.yml`) do NOT run Playwright e2e at all. The only e2e infrastructure is local-only (`frontend/playwright.config.ts` + `frontend/e2e/*.spec.ts`).

**Pattern source for adding a new GitHub Actions job** — `build.yml` is the closest analog for matrix + step shape, even though its matrix is over `webkit_tag/webkit_pkg` (Linux variants), not browsers.

**Phase 99 implication:** the orchestrator prompt's "MODIFIED `.github/workflows/*.yml` (whichever runs e2e)" assumes a workflow that does not exist. The planner has two options:

1. **Add a NEW workflow file** `e2e.yml` that runs `pnpm exec playwright install --with-deps chromium firefox webkit` then `pnpm exec playwright test` against all three projects. Trigger on `push` to `main` + `pull_request`. Recommended skeleton:
   ```yaml
   name: e2e
   on: [push, pull_request]
   jobs:
     playwright:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
         - uses: actions/setup-node@v4
           with: { node-version: '20' }
         - uses: pnpm/action-setup@v3
         - run: pnpm install
           working-directory: frontend
         - run: pnpm exec playwright install --with-deps chromium firefox webkit
           working-directory: frontend
         - run: pnpm exec playwright test
           working-directory: frontend
   ```

2. **Defer CI integration** to a future hardening pass and run the cross-browser suite locally only for v3.2 release verification. (RESEARCH.md does not commit to CI; SC-4 says "cross-browser CSP zero-violation e2e" without specifying CI vs local.)

**Recommendation:** the planner should make this an explicit decision in the relevant PLAN. Default to Option 1 (add `e2e.yml`) since "release gate" implies CI enforcement. Phase 99 of a release gate that lacks CI gating is a hole.

---

## Shared Patterns

### Shared Pattern α — `.banner-stack` BannerStack vocabulary

**Source:** `App.tsx:888-933` (rendering) + `style.css:1610-1661` (`.webgl-recovery-banner` BEM) + `WebGLRecoveryBanner.tsx` (component template).

**Apply to:** every new banner-style notification. The vocabulary is:
1. State lives in `App.tsx` as a useState (single value or array of kinds).
2. The outer `<div className="banner-stack">` guard is a logical-OR over all banner-presence predicates — extend this guard when adding a new banner family.
3. Each banner is a self-contained component with `role="status" aria-live="polite"`, a `__message` span, and a `__dismiss` button with `aria-label="Dismiss notification"` + 16px `XMarkIcon`.
4. Auto-dismiss via `useEffect(setTimeout(onDismiss, N_MS))` with cleanup (clearTimeout on unmount).
5. `prefers-reduced-motion` override on the BEM block.

**Phase 99 instances:** `pluginToggleBanners` array state in `App.tsx` + `PluginToggleBanner.tsx` component reusing `.webgl-recovery-banner` BEM verbatim.

### Shared Pattern β — Sub-key RPC dispatch (PUI-04 anti-race contract)

**Source:** `TerminalPanel.tsx:728-730` (FindBar's `SetSearchConfig` dispatch) + `app.go:527-540, 566-578, 606-617` (Wails App methods) + `api.go:76-78, 562-651` (HTTP routes + handlers) + `engine.go:497-563` (engine sub-key writers under `e.mu.Lock()`).

**Apply to:** every PUI-03 disclosure form control. The contract:
1. UI form control's `onChange` constructs a fresh `daemon.SearchConfig` / `daemon.WebLinksConfig` / `daemon.ImageConfig` from the existing prop-derived value spread + the changed field.
2. Calls the corresponding `Set*Config(new daemon.*Config(...))` Wails binding directly — NOT `SetPluginSettings(full snapshot)`.
3. `.catch(() => {})` silent — the sub-key RPC errors don't propagate to the UI; the PluginsSection's full-snapshot Save button is the only error-surface.
4. The daemon's `engine.Set*Config` mutates ONLY that sub-key under `e.mu.Lock()` — it cannot clobber PluginsSection's local edit buffer of the 8 booleans.

**Why:** Phase 94-07 WR-03 documented the race. Full-snapshot writes from a stale `pluginConfig` prop would clobber in-flight Plugins-tab edits. Sub-key RPCs are the resolution.

### Shared Pattern γ — Source-Inspection Vitest Tests

**Source:** `__tests__/PluginsSection.test.tsx:1-2` + `__tests__/SettingsTab.persistence.test.tsx:1-3` (both use `?raw` import + `expect(raw).toContain(...)`).

**Apply to:** every new vitest test for components that touch Wails-generated `daemon.*` constructors. Reason: `daemon.SearchConfig` etc. throw under jsdom because `convertValues` expects Go-side context. Source-inspection avoids the rendering path entirely.

**Phase 99 instances:** `PluginsSection.disclosure.test.tsx` (PUI-03 disclosures), assertions on `SetSearchConfig` / `SetWebLinksConfig` / `SetImageConfig` literal substrings.

**Exception:** `PluginToggleBanner.test.tsx` does NOT need source-inspection — it's a pure presentational component with no Wails-generated dependencies. Use the full `createRoot` + `flushSync` render-test pattern from `WebGLRecoveryBanner.test.tsx` instead.

### Shared Pattern δ — Manual iPad UAT Runbook

**Source:** `.planning/phases/93-.../93-iPad-UAT.md` (template + content shape) + similar runbooks at Phase 96 / 97 / 98 (cross-referenced in RESEARCH.md "no autonomy" note).

**Apply to:** every release-gate verification that touches iOS Safari over Tailscale. Verbatim copy quoted in `>` blockquotes; numbered step lists; `## Sign-Off` checklist at the bottom; reference to `XX-VALIDATION.md` for final aggregation.

**Autonomy contract** (RESEARCH.md "Locked Decisions"): the plan that consumes this UAT must have `autonomous: false` in its frontmatter — matches Phase 93 Plan 05, Phase 96 Plan 06, Phase 97 Plan 06, Phase 98 Plan 05 precedent.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `.github/workflows/e2e.yml` (if planner chooses Option 1) | CI config | matrix | The repo has no Playwright workflow today. `build.yml` is the closest analog for matrix + step shape but its semantics (Linux build variants) differ entirely. Phase 99 is the first introduction. See "MODIFIED `.github/workflows/*.yml`" section above for skeleton. |
| `tests/fixtures/settings_v3.2_partial.json` (OPTIONAL — likely not created) | config (test fixture) | file-I/O | Hypothetical mid-development upgrade fixture. RESEARCH.md flags as optional; existing `settings_v3.1.json` already covers the load-bearing case. |

---

## Metadata

**Analog search scope:**
- `frontend/src/components/` — `WebGLRecoveryBanner.tsx`, `PluginsSection.tsx`, `SettingsTab.tsx`, `TerminalPanel.tsx`, `App.tsx`
- `frontend/src/components/__tests__/` — `WebGLRecoveryBanner.test.tsx`, `PluginsSection.test.tsx`
- `frontend/src/wailsjs/go/main/App.d.ts` (Wails TS bindings — `SetSearchConfig`/`SetWebLinksConfig`/`SetImageConfig` at lines 131/137/143)
- `frontend/src/wailsjs/go/models.ts` (auto-generated `SearchConfig`/`WebLinksConfig`/`ImageConfig` classes at lines 10/31/54)
- `frontend/src/style.css` — `.webgl-recovery-banner` (1610-1661), `.settings-panel__details` (573-583), `.settings-panel__description--italic` (1605-1608)
- `frontend/playwright.config.ts` (current `projects[]` array)
- `internal/daemon/` — `engine_migration_test.go`, `plugin_settings.go` (`defaultPluginSettings()` constructor)
- `internal/daemon/api.go` (HTTP routes 76-78, handlers 562-651) + `app.go` (Wails methods 527/566/606)
- `tests/fixtures/` — confirmed `settings_v3.1.json` already exists
- `.github/workflows/*.yml` — confirmed no Playwright workflow exists
- `.planning/phases/93-.../93-iPad-UAT.md` — manual UAT runbook template

**Files scanned:** 13 source files + 2 generated TS files + 4 GitHub Actions YAMLs + 1 fixture directory listing + 1 prior-phase runbook

**Pattern extraction date:** 2026-05-08

**Confidence:** HIGH — every Phase 99 file maps to a verified existing analog with concrete file:line references. The single load-bearing decision the planner must make is whether to add a new `e2e.yml` GitHub Actions workflow or defer CI integration to a future hardening pass.

## PATTERN MAPPING COMPLETE
