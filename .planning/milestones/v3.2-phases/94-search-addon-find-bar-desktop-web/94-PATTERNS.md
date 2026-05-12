# Phase 94: Search Addon + Find Bar (Desktop + Web) — Pattern Map

**Mapped:** 2026-05-04
**Files analyzed:** 25 (10 new, 15 modified)
**Analogs found:** 25 / 25

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/FindBar.tsx` | component (controlled) | event-driven (UI) | `frontend/src/components/WebGLRecoveryBanner.tsx` | role-match (controlled banner with dismiss + heroicons + aria) |
| `frontend/src/components/TerminalPanel.tsx` (mod) | component (xterm host) | event-driven | itself + Phase 93 hot-swap useEffect | exact (extend hot-swap dep array) |
| `frontend/src/style.css` (mod) | styles (BEM) | n/a | existing `.webgl-recovery-banner` + `.settings-panel__path-input` blocks | exact |
| `frontend/src/lib/isXtermFocused.ts` | utility (pure helper) | request-response | `frontend/src/lib/webglProbe.ts` (small browser-DOM helper) | role-match |
| `frontend/src/hooks/useFindBar.ts` (optional) | hook (lifecycle + debounce) | event-driven | inline pattern in TerminalPanel hot-swap useEffect | role-match |
| `frontend/src/wailsjs/go/models.ts` (mod) | type binding | n/a | existing `daemon.PluginSettings` class | exact (additive nested class) |
| `frontend/src/components/__tests__/FindBar.test.tsx` | test (component contract) | n/a | `WebGLRecoveryBanner.test.tsx` | exact |
| `frontend/src/components/__tests__/FindBar.perf.test.tsx` | test (perf benchmark) | n/a | (no analog — new test category) | none — RESEARCH-driven |
| `frontend/src/components/__tests__/TerminalPanel.test.tsx` (mod) | test (source-inspection) | n/a | `TerminalPanel.hot-swap.test.tsx` | exact |
| `frontend/src/__tests__/App.plugin-event.test.tsx` (mod) | test (source-inspection) | n/a | itself | exact (add searchConfig assertions) |
| `frontend/src/lib/__tests__/isXtermFocused.test.ts` | test (unit, pure) | n/a | (no exact analog; vitest unit test convention) | role-match |
| `internal/daemon/plugin_settings.go` (mod) | model (persisted struct + defaults) | CRUD | itself | exact (add SearchConfig nested struct + field) |
| `internal/daemon/plugin_settings_test.go` (mod) | test (defaults assertion) | n/a | itself | exact (extend TestDefaultPluginSettings) |
| `internal/daemon/engine_migration_test.go` (extend) | test (settings.json migration) | n/a | itself | exact (assert SearchConfig defaults populate) |
| `internal/webserver/find_bar_test.go` | test (web asset source-inspection) | n/a | `internal/webserver/assets_test.go::TestAssets_VendoredAddons` | exact |
| `internal/webserver/vendor_drift_test.go` (mod) | test (vendor drift) | n/a | itself | exact (bump min-count from 5 to 6) |
| `web/embed.go` (mod) | config (`//go:embed` directive) | file-I/O | itself | exact (append addon-search.js) |
| `web/terminal.html` (mod) | static template | n/a | itself + `#webgl-recovery-banner` element | exact |
| `web/assets/terminal.js` (mod) | controller (web SearchAddon init + handlers) | event-driven | `web/assets/terminal.js::applyPluginConfig` (Phase 93) | exact |
| `web/assets/terminal.css` (mod) | styles (web parity) | n/a | existing `#webgl-recovery-banner` block | exact |
| `web/vendor/xterm/VERSION` (mod) | manifest | n/a | itself | exact (append addon-search line) |
| `web/vendor/xterm/addons/addon-search.js` | vendored UMD bundle | n/a | `web/vendor/xterm/addons/addon-webgl.js` | exact (copy from `frontend/node_modules/@xterm/addon-search/lib/`) |
| `frontend/package.json` (mod) | manifest | n/a | itself (Phase 93 added 3 prior `@xterm/addon-*` deps) | exact |
| `.planning/phases/94-.../94-DESKTOP-UAT.md` | runbook | n/a | `.planning/phases/93-.../93-iPad-UAT.md` | role-match |
| `.planning/phases/94-.../94-WEB-UAT.md` | runbook | n/a | `.planning/phases/93-.../93-iPad-UAT.md` | role-match |

---

## Pattern Assignments

### `frontend/src/components/FindBar.tsx` (component, event-driven)

**Analog:** `frontend/src/components/WebGLRecoveryBanner.tsx`

**Imports pattern** (lines 1-2):
```typescript
import React, { useEffect } from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'
```
Phase 94 will additionally import outline icons for the toggle group (`@heroicons/react/24/outline`). Keep the heroicons import grouping consistent with WebGLRecoveryBanner.

**Controlled props interface** (lines 25-29):
```typescript
export interface WebGLRecoveryBannerProps {
  reason: 'context-loss' | 'software-rasterized'
  onDismiss: () => void
  className?: string
}
```
FindBar mirrors this pattern: zero internal state beyond focus, all state owned by parent (TerminalPanel). UI-SPEC §"Component Inventory" line 447 confirms: "Manages no state internally beyond focus. Parent (TerminalPanel) owns all state."

**Auto-dismiss timer with cleanup** (lines 36-40) — pattern reused for the 100ms search debounce:
```typescript
useEffect(() => {
  if (reason !== 'context-loss') return
  const id = window.setTimeout(onDismiss, 8000)
  return () => window.clearTimeout(id)
}, [reason, onDismiss])
```
RESEARCH §"Pattern 4" specifies the same `window.setTimeout` + `clearTimeout` cleanup discipline for the 100ms search debounce. Use a `useRef<number | null>` to allow imperative clearing on dismiss.

**ARIA + className composition** (lines 47-50):
```typescript
const cls = ['webgl-recovery-banner', className].filter(Boolean).join(' ')

return (
  <div className={cls} role="status" aria-live="polite">
```
FindBar uses `role="search"` + `aria-label="Find in terminal"` per UI-SPEC §"Web — Identical Behavior" line 370 and §"Accessibility Contract" line 500. The match-count span gets `aria-live="polite" aria-atomic="true"` per UI-SPEC line 501.

**Dismiss button pattern** (lines 52-59):
```typescript
<button
  type="button"
  className="webgl-recovery-banner__dismiss"
  aria-label="Dismiss notification"
  onClick={onDismiss}
>
  <XMarkIcon style={{ width: 16, height: 16 }} aria-hidden="true" />
</button>
```
FindBar close button follows verbatim: `XMarkIcon` at 16px, `aria-label="Close find bar"`, `title="Close (Esc)"` per UI-SPEC copywriting table line 413-414.

---

### `frontend/src/components/TerminalPanel.tsx` (component, modify)

**Analog:** itself — extend the existing Phase 93 hot-swap useEffect.

**Hot-swap useEffect dep array** (line 233 — exact extension point):
```typescript
}, [pluginConfig?.webgl, pluginConfig?.clipboard, onWebGLContextLost, sessionId])
```
**Add** `pluginConfig?.search` to this dep array. RESEARCH §"Pattern 1" explicitly references this line.

**Addon ref declarations** (lines 76-78 — extension point):
```typescript
// Phase 93 WGL-01 / CLIP-01: addon refs for hot-swap useEffect.
const webglAddonRef = useRef<WebglAddon | null>(null)
const clipboardAddonRef = useRef<ClipboardAddon | null>(null)
```
Add `const searchAddonRef = useRef<SearchAddon | null>(null)` directly after `clipboardAddonRef`.

**Hot-swap on/off arms** (lines 188-218 — exact pattern to copy):
```typescript
if (pluginConfig?.webgl) {
  if (!webglAddonRef.current) {
    if (isSoftwareWebGL()) {
      onWebGLContextLost?.('software-rasterized')
    } else {
      try {
        const webglAddon = new WebglAddon()
        webglAddon.onContextLoss(() => { ... })
        term.loadAddon(webglAddon)
        webglAddonRef.current = webglAddon
      } catch (err) {
        console.warn(`[TerminalPanel] WebGL unavailable for session ${sessionId}:`, err)
      }
    }
  }
} else {
  if (webglAddonRef.current) {
    webglAddonRef.current.dispose()
    webglAddonRef.current = null
  }
}
```
Search arm is simpler (no probe, no error path): construct on enable, dispose on disable. RESEARCH §"Pattern 1" provides the exact code.

**Mount-cleanup addon disposal** (lines 156-164 — extension point):
```typescript
if (webglAddonRef.current) {
  webglAddonRef.current.dispose()
  webglAddonRef.current = null
}
if (clipboardAddonRef.current) {
  clipboardAddonRef.current.dispose()
  clipboardAddonRef.current = null
}
```
Add an identical block for `searchAddonRef.current`.

**New useEffect for window keydown listener** — UI-SPEC §"Opening the Find Bar" + RESEARCH §"Pattern 2" Example 2:
```typescript
useEffect(() => {
  function handleKeyDown(e: KeyboardEvent) {
    if (!pluginConfig?.search) return
    const isMac = navigator.platform.toUpperCase().includes('MAC')
    const modifier = isMac ? e.metaKey : e.ctrlKey
    if (!modifier || e.key.toLowerCase() !== 'f') return
    if (!containerRef.current?.contains(document.activeElement)) return
    e.preventDefault()
    setFindBarOpen(true)
  }
  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [pluginConfig?.search])
```

---

### `frontend/src/style.css` (styles, modify)

**Analog:** existing `.webgl-recovery-banner` block (lines 1587-1638) + `.settings-panel__path-input` block (lines 400-413).

**Section header convention** (line 1587):
```css
/* ─── Phase 93 WGL-02/WGL-03 — WebGL recovery banner ───────────────────
   Parallel to .update-banner — same TokyoNight palette, same accent
   border (#7aa2f7) for the info tone, same banner-stack height (53px). */
```
Phase 94 adds: `/* ─── Phase 94 — Find bar (SRC-01/SRC-04) ─── */` per UI-SPEC line 266.

**Input pattern** (lines 400-413) — adapt for find-bar bottom-border-only input:
```css
.settings-panel__path-input {
  width: 100%;
  background-color: #16161e;
  border: 1px solid #292e42;
  border-radius: 4px;
  color: #c0caf5;
  font-size: 12px;
  padding: 6px 8px;
  outline: none;
  font-family: '"Cascadia Code"', '"Fira Code"', monospace;
}
.settings-panel__path-input:focus {
  border-color: #7aa2f7;
}
```
FindBar input deviates: `border: none; border-bottom: 1px solid #292e42` (toolbar idiom per UI-SPEC line 213); same focus color `#7aa2f7`; sans-serif font family per UI-SPEC line 215.

**Icon-button hover + focus pattern** (lines 1613-1633):
```css
.webgl-recovery-banner__dismiss {
  background: transparent;
  border: none;
  color: #9aa5ce;
  cursor: pointer;
  padding: 4px;
  min-width: 24px;
  min-height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}
.webgl-recovery-banner__dismiss:hover {
  color: #c0caf5;
  background: #1e2030;
}
.webgl-recovery-banner__dismiss:focus-visible {
  outline: 2px solid #7aa2f7;
  outline-offset: 2px;
}
```
FindBar icon buttons follow this verbatim, with dimensions adjusted to 28×28 (UI-SPEC §"Icon Buttons" lines 232-246) and a `--active` modifier rule for toggle buttons (UI-SPEC lines 248-253 — `background-color: rgba(122, 162, 247, 0.12); color: #7aa2f7;`).

**Reduced-motion guard** (lines 1634-1638):
```css
@media (prefers-reduced-motion: reduce) {
  .webgl-recovery-banner {
    transition: none;
  }
}
```
FindBar applies the same `@media` block to `.find-bar` per UI-SPEC line 201.

---

### `frontend/src/lib/isXtermFocused.ts` (utility)

**Analog:** `frontend/src/lib/webglProbe.ts` (small browser-DOM utility imported by TerminalPanel — referenced at line 9 of TerminalPanel.tsx as `import { isSoftwareWebGL } from '../lib/webglProbe'`).

**Function shape** — single named export, pure, accepts the DOM element it queries:
```typescript
export function isXtermFocused(termContainer: HTMLElement | null): boolean {
  if (!termContainer || !document.activeElement) return false
  return termContainer.contains(document.activeElement)
}
```
RESEARCH §"Pattern 2" lines 399-403 provide the canonical implementation. Keep the function `export`-named (not default) to match the `isSoftwareWebGL` export pattern at the call site.

---

### `internal/daemon/plugin_settings.go` (model, modify)

**Analog:** itself — additive nested struct.

**Existing struct** (lines 17-26):
```go
type PluginSettings struct {
    WebGL     bool `json:"webgl"`
    Unicode11 bool `json:"unicode11"`
    Search    bool `json:"search"`
    WebLinks  bool `json:"webLinks"`
    Image     bool `json:"image"`
    Serialize bool `json:"serialize"`
    Clipboard bool `json:"clipboard"`
    Progress  bool `json:"progress"`
}
```
Add `SearchConfig SearchConfig \`json:"searchConfig"\`` field directly after `Search`. Field placement matches RESEARCH lines 455-465.

**Existing default factory** (lines 36-47):
```go
func defaultPluginSettings() PluginSettings {
    return PluginSettings{
        WebGL:     true,
        Unicode11: true,
        Search:    true,
        WebLinks:  true,
        Image:     true,
        Serialize: true,
        Clipboard: true,
        Progress:  false,
    }
}
```
Add `SearchConfig: SearchConfig{Regex: false, CaseSensitive: false, WholeWord: false},` per RESEARCH lines 467-479. All three sub-fields default `false` per UI-SPEC §"Toggle Persistence" + REQUIREMENTS SRC-02.

**Doc-comment convention** — single-line struct purpose + multi-line rationale (lines 10-16). New `SearchConfig` struct gets the same treatment per RESEARCH lines 444-453.

**No omitempty rule** (lines 14-16) — comment "NO omitempty: missing keys must round-trip as the user's saved choice" applies equally to `SearchConfig`'s booleans.

---

### `internal/daemon/plugin_settings_test.go` (test, modify)

**Analog:** itself — extend the per-field assertion pattern.

**Existing per-field assertion** (lines 11-37):
```go
func TestDefaultPluginSettings(t *testing.T) {
    s := defaultPluginSettings()
    if !s.WebGL {
        t.Error("expected WebGL=true (UI-SPEC default ON)")
    }
    // ...
    if s.Progress {
        t.Error("expected Progress=false (UI-SPEC default OFF; flips ON in v3.3 per ROADMAP)")
    }
}
```
Add three assertions inside the same test body:
```go
if s.SearchConfig.Regex {
    t.Error("expected SearchConfig.Regex=false (Phase 94 SRC-02 default OFF)")
}
if s.SearchConfig.CaseSensitive {
    t.Error("expected SearchConfig.CaseSensitive=false (Phase 94 SRC-02 default OFF)")
}
if s.SearchConfig.WholeWord {
    t.Error("expected SearchConfig.WholeWord=false (Phase 94 SRC-02 default OFF)")
}
```
Header comment style — keep "per ROADMAP `## Decisions` and UI-SPEC §..." citation pattern.

**Migration test extension** — analog: `engine_migration_test.go::TestSettingsMigrationV3_1ToV3_2` lines 41-79. The defaults-merge already populates new struct fields automatically (RESEARCH §"Pattern 3" — "Migration: zero-effort"). Add an assertion that `e.GetPluginSettings().SearchConfig == SearchConfig{}` after loading a Phase 93 fixture.

---

### `internal/webserver/plugin_config.go` + `plugin_config_stream.go` (NO CHANGE)

**Analog:** the `func() []byte` provider pattern at `plugin_config.go` lines 25-38:
```go
func (ws *WebServer) handleGetPluginConfig(w http.ResponseWriter, r *http.Request) {
    if ws.pluginSettingsProvider == nil { ... }
    body := ws.pluginSettingsProvider()
    if body == nil { ... }
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "no-store")
    _, _ = w.Write(body)
}
```
Phase 93's `pluginSettingsProvider func() []byte` indirection (api.go line 291: `ws.SetPluginSettingsProvider(func() []byte { s := a.engine.GetPluginSettings(); ... json.Marshal(s) })`) recurses into nested structs automatically (RESEARCH Assumption A6). **No code change for Phase 94.**

The SSE broadcast at `plugin_config_stream.go` lines 100-117 (`BroadcastPluginConfig`) likewise needs no change — the entire pipeline is value-shape-agnostic.

---

### `internal/webserver/find_bar_test.go` (NEW — test, web asset source-inspection)

**Analog:** `internal/webserver/assets_test.go::TestAssets_VendoredAddons` (lines 205-230):
```go
func TestAssets_VendoredAddons(t *testing.T) {
    ws, client := testServer(t)
    for _, path := range []string{
        "/assets/xterm/addons/addon-webgl.js",
        "/assets/xterm/addons/addon-unicode11.js",
        "/assets/xterm/addons/addon-clipboard.js",
    } {
        resp, err := client.Get(ws.BaseURL() + path)
        if err != nil {
            t.Fatalf("client.Get %s: %v", path, err)
        }
        if resp.StatusCode != http.StatusOK {
            t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
        }
        ct := resp.Header.Get("Content-Type")
        if !strings.Contains(ct, "javascript") {
            t.Errorf("GET %s: expected javascript content-type, got %q", path, ct)
        }
        resp.Body.Close()
    }
}
```
Phase 94 adds a `TestAssets_AddonSearch` (or extends the existing test) — same shape, with `addon-search.js` appended. Plus source-inspection tests:

- `TestTerminalHTML_FindBar` — read `web/terminal.html` (via `web.WebFS` embed.FS), assert it contains `<div id="find-bar"` and `<script src="/assets/xterm/addons/addon-search.js">`.
- `TestTerminalJS_SearchAddon` — read `web/assets/terminal.js`, assert it contains `new SearchAddon.SearchAddon()` (or whatever the verified UMD constructor expression is — see Pitfall #7 in RESEARCH).

---

### `internal/webserver/vendor_drift_test.go` (modify)

**Analog:** itself.

**Min-count guard** (line 34):
```go
if len(pnpmVersions) < 5 {
    t.Fatalf("failed to parse at least 5 @xterm/* packages (xterm, addon-fit, addon-webgl, addon-unicode11, addon-clipboard) from pnpm-lock.yaml: ...")
}
```
**Bump to 6**: add `addon-search` to the package list in the error message; change `< 5` → `< 6`. RESEARCH §"Code Examples" Example 5 confirms the regex auto-matches `addon-search`; only the count is hand-edited.

---

### `web/embed.go` (modify)

**Analog:** itself (lines 5-10):
```go
//go:embed dashboard.html terminal.html join.html
//go:embed assets/terminal.js assets/terminal.css
//go:embed assets/dashboard.js assets/dashboard.css
//go:embed assets/join.js assets/join.css
//go:embed vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION
//go:embed vendor/xterm/addons/addon-webgl.js vendor/xterm/addons/addon-unicode11.js vendor/xterm/addons/addon-clipboard.js
```
Append `vendor/xterm/addons/addon-search.js` to the last directive (or add a new `//go:embed` line) — keep the line under 200 chars to match existing formatting.

---

### `web/terminal.html` (modify)

**Analog:** itself (lines 15-21):
```html
<div id="webgl-recovery-banner" hidden role="status" aria-live="polite"></div>
<div id="terminal"></div>
<script src="/assets/xterm/xterm.js"></script>
<script src="/assets/xterm/addon-fit.js"></script>
<script src="/assets/xterm/addons/addon-webgl.js"></script>
<script src="/assets/xterm/addons/addon-unicode11.js"></script>
<script src="/assets/xterm/addons/addon-clipboard.js"></script>
<script src="/assets/terminal.js"></script>
```

**Hidden-element sentinel** — UI-SPEC line 389 explicitly cites the `#webgl-recovery-banner[hidden]` pattern as the model:
```html
<div id="webgl-recovery-banner" hidden role="status" aria-live="polite"></div>
```
Phase 94 adds `<div id="find-bar" hidden role="search" aria-label="Find in terminal"> ... </div>` (full DOM structure in UI-SPEC lines 369-385) just before `<div id="terminal">` OR inside it. UI-SPEC §"Web: terminal.html structure" lines 481-486 specifies inside `#terminal` (which becomes `position: relative`).

**Script tag insertion** — append `<script src="/assets/xterm/addons/addon-search.js"></script>` directly before `<script src="/assets/terminal.js"></script>` (matches the addon-clipboard insertion order).

---

### `web/assets/terminal.js` (modify)

**Analog:** existing `applyPluginConfig` function (lines 233-273) and the SSE EventSource handler (lines 357-396).

**Addon hot-swap pattern** (lines 237-256) — exact template for SearchAddon:
```javascript
// WebGL hot-swap.
if (!newConfig.webgl && webglAddonHandle) {
  try { webglAddonHandle.dispose(); } catch (e) {}
  webglAddonHandle = null;
} else if (newConfig.webgl && !webglAddonHandle && !isSoftwareWebGL()) {
  try {
    webglAddonHandle = new WebglAddon.WebglAddon();
    webglAddonHandle.onContextLoss(function() { ... });
    term.loadAddon(webglAddonHandle);
  } catch (e) { /* silent */ }
}
```
Phase 94 adds an analogous `searchAddonHandle` block. RESEARCH §"Code Examples" Example 4 lines 733-749 provides the exact code:
```javascript
var searchAddonHandle = null
function applySearchAddon(enabled) {
  if (enabled) {
    if (!searchAddonHandle) {
      try {
        searchAddonHandle = new SearchAddon.SearchAddon()
        term.loadAddon(searchAddonHandle)
      } catch (e) {}
    }
  } else {
    if (searchAddonHandle) {
      try { searchAddonHandle.dispose() } catch (e) {}
      searchAddonHandle = null
    }
  }
}
```
**Note** RESEARCH Pitfall #7: verify the UMD global name at exec time via `grep -o "root\[[\"'].*[\"']\]" web/vendor/xterm/addons/addon-search.js | head -3`. Likely `SearchAddon.SearchAddon` (matches the Phase 93 pattern for WebglAddon/Unicode11Addon/ClipboardAddon).

**Cmd-F handler** — RESEARCH §"Code Examples" Example 4 lines 754-764:
```javascript
window.addEventListener('keydown', function(e) {
  if (!pluginConfig.search) return
  var isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0
  var modifier = isMac ? e.metaKey : e.ctrlKey
  if (!modifier || e.key.toLowerCase() !== 'f') return
  var termEl = document.getElementById('terminal')
  if (!termEl || !termEl.contains(document.activeElement)) return
  e.preventDefault()
  showFindBar()
})
```

**Banner show-hide pattern** (lines 182-209) — `showWebGLBanner` shows the find bar shape: query DOM, set content + properties, toggle `hidden`. The web FindBar's `showFindBar() / hideFindBar()` follows the same idiom — set `el.hidden = false`, focus the input, etc.

---

### `web/assets/terminal.css` (modify)

**Analog:** existing `#webgl-recovery-banner` block (terminal.css lines 50-91).

**Section header** (line 50):
```css
/* Phase 93 WGL-02 / WGL-04 — web parity with desktop .webgl-recovery-banner. */
```
Phase 94 adds `/* Phase 94 — Find bar */` per UI-SPEC line 387.

**Hidden-attribute sentinel** (line 69):
```css
#webgl-recovery-banner[hidden] { display: none; }
```
Phase 94 mirrors with `#find-bar[hidden] { display: none; }` and `#find-bar:not([hidden]) { display: flex; }` per UI-SPEC line 389.

**Web parity color/spacing** (lines 51-69) — exact same TokyoNight token values as desktop `style.css`. Per UI-SPEC line 451: "Exact same token values."

---

### `frontend/src/wailsjs/go/models.ts` (modify)

**Analog:** itself — existing `daemon.PluginSettings` class shape (lines 8-37).

**Hand-edit pin pattern** (lines 1-7 — header comment cites the convention):
```typescript
// Auto-generated by Wails (`wails generate module`) and pinned in-repo.
// This file mirrors the `daemon.PluginSettings` Go struct surfaced through
// the (*App).GetPluginSettings / SetPluginSettings Wails bindings (PLUG-03).
```
RESEARCH §"Pattern 3" + Assumption A5: hand-edit additive — add a new `SearchConfig` exported class to the `daemon` namespace, add `searchConfig: SearchConfig` field + constructor line to `PluginSettings`. RESEARCH §"Code Examples" Example 3 lines 678-724 provides the exact diff.

**Field-init constructor pattern** (lines 24-34):
```typescript
constructor(source: any = {}) {
    if ('string' === typeof source) source = JSON.parse(source);
    this.webgl = source["webgl"];
    this.unicode11 = source["unicode11"];
    this.search = source["search"];
    // ...
}
```
For `searchConfig` (a nested class), use Wails-generated `convertValues` helper. RESEARCH Example 3 line 715: `this.searchConfig = this.convertValues(source['searchConfig'], SearchConfig)`.

---

### `frontend/src/components/__tests__/FindBar.test.tsx` (NEW — test)

**Analog:** `frontend/src/components/__tests__/WebGLRecoveryBanner.test.tsx` (lines 1-80).

**Test harness pattern** (lines 16-24):
```typescript
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
FindBar test uses `createRoot` + `flushSync` + manual mount/unmount in `afterEach` — same recipe. UI-SPEC §"Copywriting Contract" lines 397-414 dictates the verbatim copy assertions (`Find…`, `Previous match`, `Next match`, `Case sensitive`, `Regular expression`, `Whole word`, `Close find bar`).

**Verbatim-copy assertion pattern** (lines 43-47):
```typescript
it("reason='context-loss' renders verbatim recovery copy including 'Scrollback is intact.'", () => {
  ;({ container, root } = render('context-loss', vi.fn()))
  expect(container.textContent).toMatch(/Hardware-accelerated rendering recovered/)
  expect(container.textContent).toMatch(/Scrollback is intact\./)
})
```
FindBar tests the placeholder (`Find…`), the match-count format (`3 of 12`, `0 of 0`), and aria-labels.

**ARIA assertion pattern** (lines 55-60):
```typescript
it('has role="status" and aria-live="polite" (accessibility contract)', () => {
  ;({ container, root } = render('context-loss', vi.fn()))
  const statusEl = container.querySelector('[role="status"]')
  expect(statusEl).not.toBeNull()
  expect(statusEl?.getAttribute('aria-live')).toBe('polite')
})
```
FindBar asserts `[role="search"]`, `[aria-label="Find in terminal"]`, `[aria-pressed]` on each toggle, `[aria-live="polite"]` on the count span.

**Fake timers for debounce** (lines 71-80):
```typescript
beforeEach(() => {
  vi.useFakeTimers()
})
it("reason='context-loss' fires onDismiss after 8000ms (auto-dismiss)", () => {
  // ...
  vi.advanceTimersByTime(8000)
})
```
FindBar uses identical pattern with `vi.advanceTimersByTime(100)` for the search debounce.

---

### `frontend/src/components/__tests__/TerminalPanel.test.tsx` (modify) + new source-inspection tests

**Analog:** `frontend/src/components/__tests__/TerminalPanel.hot-swap.test.tsx` (entire file, 67 lines).

**Source-inspection convention** (line 19):
```typescript
import src from '../TerminalPanel.tsx?raw'
```
RESEARCH §"Validation Architecture" calls out this exact pattern. New assertions to add (extending hot-swap test or creating `TerminalPanel.search.test.tsx`):

```typescript
it('declares searchAddonRef ref', () => {
  expect(src).toContain('searchAddonRef')
})
it('imports SearchAddon from @xterm/addon-search', () => {
  expect(src).toMatch(/import\s+\{\s*SearchAddon\s*\}\s+from\s+['"]@xterm\/addon-search['"]/)
})
it('hot-swap useEffect dep array includes pluginConfig?.search', () => {
  expect(src).toMatch(/\[\s*pluginConfig\?\.webgl\s*,\s*pluginConfig\?\.clipboard\s*,\s*pluginConfig\?\.search/)
})
it('contains focus-conditioned Cmd-F keydown listener (window.addEventListener)', () => {
  expect(src).toContain("window.addEventListener('keydown'")
  expect(src).toMatch(/containerRef\.current\?\.contains\(document\.activeElement\)/)
})
```

---

## Shared Patterns

### Authentication / Capability
**Source:** `internal/webserver/plugin_config.go` lines 25-38
**Apply to:** New `internal/webserver/find_bar_test.go` web-asset checks
- The `requireCapability` middleware already gates `/api/plugin-config` and the `/assets/xterm/addons/*` static paths via existing route registration. Phase 94 adds zero new endpoints. No new auth code.

### Error Handling
**Source:** `frontend/src/components/PluginsSection.tsx` lines 41-54
```typescript
async function handleSavePlugins(): Promise<void> {
  if (!pluginConfig) return
  setSaving(true)
  setError(null)
  try {
    await SetPluginSettings(pluginConfig)
    setSaved(true)
    setTimeout(() => setSaved(false), 1500)
  } catch (err) {
    setError(err instanceof Error ? err.message : String(err))
  } finally {
    setSaving(false)
  }
}
```
**Apply to:** FindBar's toggle persistence path. RESEARCH §"Pattern 3" frontend-consumption: optimistic local update + fire-and-forget `SetPluginSettings`. On failure, surface via the same error toast vocabulary used in PluginsSection (line 132: `Could not save plugin settings — {error}`).

### Defensive Merge of Daemon Config
**Source:** `web/assets/terminal.js` lines 117-136 + lines 367-373
```javascript
var pluginConfig = {
  webgl: true, unicode11: true, clipboard: true,
  search: true, webLinks: true, image: true,
  serialize: true, progress: false
};
// ...
for (var k in pc) {
  if (Object.prototype.hasOwnProperty.call(pc, k)) pluginConfig[k] = pc[k];
}
```
**Apply to:** Web `terminal.js` SearchConfig handling. Add `searchConfig: { regex: false, caseSensitive: false, wholeWord: false }` to the defaults object; defensive merge already handles the new key automatically (RESEARCH §"Pattern 3" web-consumption).

### Hidden-Attribute Sentinel
**Source:** `web/terminal.html` line 15 + `web/assets/terminal.css` line 69
- Web banners use `<div hidden>` + CSS `[hidden] { display: none; }` rather than CSS `display: none` directly. This is the load-bearing pattern for the web find bar (UI-SPEC line 389).

### Reduced-Motion Guard
**Source:** `frontend/src/style.css` lines 1634-1638 + `web/assets/terminal.css` lines 89-91
```css
@media (prefers-reduced-motion: reduce) {
  .webgl-recovery-banner { transition: none; }
}
```
**Apply to:** Both desktop `.find-bar` and web `#find-bar` per UI-SPEC line 201.

### Focus-Visible Outline
**Source:** `frontend/src/style.css` line 1556 + 1630-1633 (recurring pattern across the entire stylesheet)
```css
outline: 2px solid #7aa2f7;
outline-offset: 2px;
```
**Apply to:** All find-bar focusable elements (input, toggles, nav buttons, close button) per UI-SPEC line 246 + Accessibility Contract line 499.

### Vendor Drift CI Gate
**Source:** `internal/webserver/vendor_drift_test.go` lines 18-73
**Apply to:** Phase 94's vendored `addon-search.js` is auto-covered by the regex; only the min-count guard at line 34 needs a bump (5 → 6).

### Source-Inspection Test Style
**Source:** `frontend/src/components/__tests__/TerminalPanel.hot-swap.test.tsx` (entire file)
- Use `?raw` Vite import for whole-file source inspection.
- Each test is a single regex/contains assertion with a Pitfall citation in the test name.
- Apply to all FindBar source-inspection tests (focus listener presence, dep-array shape, debounce setup).

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `frontend/src/components/__tests__/FindBar.perf.test.tsx` | benchmark test | n/a | No perf-budget tests exist in the codebase yet. Use Vitest + `performance.now()` per RESEARCH §"Validation Architecture" Sampling Rate. Borrow harness shape from `WebGLRecoveryBanner.test.tsx` mount/unmount; the assertion shape is novel (`expect(elapsed).toBeLessThan(1000)`). |

All other files have clear analogs.

---

## Metadata

**Analog search scope:**
- `frontend/src/components/` (TerminalPanel, WebGLRecoveryBanner, PluginsSection)
- `frontend/src/components/__tests__/` (TerminalPanel.hot-swap, WebGLRecoveryBanner)
- `frontend/src/__tests__/` (App.plugin-event)
- `frontend/src/lib/` (webglProbe — not read but referenced as analog target)
- `frontend/src/wailsjs/go/models.ts`
- `frontend/src/style.css` (terminal-session-container, settings-panel__path-input, settings-panel__toggle-row, banner-stack, webgl-recovery-banner sections)
- `internal/daemon/` (plugin_settings.go, plugin_settings_test.go, engine_migration_test.go, engine_plugins_test.go, engine.go, api.go)
- `internal/webserver/` (plugin_config.go, plugin_config_stream.go, vendor_drift_test.go, assets_test.go)
- `web/` (terminal.html, embed.go, assets/terminal.js, assets/terminal.css, vendor/xterm/VERSION, vendor/xterm/addons/)

**Files scanned:** 23

**Pattern extraction date:** 2026-05-04
