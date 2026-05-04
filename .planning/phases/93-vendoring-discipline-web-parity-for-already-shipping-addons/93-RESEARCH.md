# Phase 93: Vendoring Discipline + Web Parity for Already-Shipping Addons — Research

**Researched:** 2026-05-04
**Domain:** xterm.js addon lifecycle, web vendoring pipeline, capability-gated web API, CI drift guard
**Confidence:** HIGH

---

## Summary

Phase 93 activates the plugin pipeline that Phase 92 built: the `void pluginConfig` inert-prop invariant in `TerminalPanel.tsx` (line 59) is lifted and replaced with real addon-load useEffect logic for the three addons already present in node_modules (`@xterm/addon-webgl`, `@xterm/addon-unicode11`, `@xterm/addon-clipboard`). Simultaneously, all three are vendored under `web/vendor/xterm/addons/` so the web-served terminal page (`web/terminal.html` + `web/assets/terminal.js`) can load them same-origin without CDN requests. A new capability-gated `/api/plugin-config` endpoint in the Go webserver lets the web client read plugin settings; a generalized `vendor_drift_test.go` enforces version parity between `frontend/package.json` and `web/vendor/xterm/VERSION` for every `@xterm/addon-*` package. The phase introduces no new design tokens — all visual surfaces (one italic CSS modifier, two BannerStack toasts) compose from Phase 92's locked design system.

**Primary recommendation:** Extend `TerminalPanel`'s existing `sessionId`-scoped `useEffect` to conditionally load/dispose addons based on `pluginConfig`; add a separate `pluginConfig`-scoped `useEffect` for hot-swap (WebGL and Clipboard); add a new `/api/plugin-config` Go handler that wraps `requireCapability`; copy the three addon `.js` files from `node_modules` to `web/vendor/xterm/addons/`; generalize the `pnpmXtermKeyRe` regex to cover every `@xterm/addon-*`.

---

## Project Constraints (from CLAUDE.md)

- Python venv required — not applicable to this phase (no Python).
- Node: use `pnpm` (confirmed by `frontend/pnpm-lock.yaml` and `package.json`).
- TypeScript: `camelCase` vars, `PascalCase` components, ESLint + Prettier.
- Go: `go fmt`, context-aware functions.
- No global npm installs.
- `NEVER kill node.exe` — Claude Code runs as Node.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PLUG-04 | Plugin state changes propagate to all connected web clients via `/api/plugin-config` (capability-gated, v3.1 SEC-* model) without manual page reload for hot-swappable plugins | New Go handler in `internal/webserver/server.go` wrapping `requireCapability`; web `terminal.js` polls/fetches on load and re-applies |
| WGL-01 | WebGL toggle applies live to all open desktop terminals without session restart (hot-swap both directions) | Lift Phase 92 inert-prop invariant; add `pluginConfig`-scoped useEffect in TerminalPanel that dispose/re-creates WebglAddon |
| WGL-02 | WebGL context loss falls back to DOM renderer, scrollback intact, no auto-retry, one-shot BannerStack toast | Existing `onContextLoss` handler (TerminalPanel line 87) already disposes — extend to fire `onWebGLContextLost` callback to App.tsx; new `WebGLRecoveryBanner.tsx` component |
| WGL-03 | Software-rasterized WebGL contexts detected at startup; DOM renderer used preemptively | Add `gl.getParameter(gl.RENDERER)` probe in TerminalPanel mount useEffect before WebglAddon construction |
| WGL-04 | Web-served clients get same WebGL behavior with vendored addon assets, CSP `script-src 'self'` honored | Copy `addon-webgl.js` to `web/vendor/xterm/addons/`; add `<script>` tag in `terminal.html`; web `terminal.js` conditionally loads based on `/api/plugin-config` response |
| U11-01 | Unicode 11 toggle marked "applies to new sessions"; honored only at next-session create time | Phase 92 already loads Unicode11Addon unconditionally; gate at session init (mount useEffect) only, not the hot-swap useEffect |
| U11-02 | Web-served clients use same Unicode 11 setting as daemon (server-shared, not per-client) | `/api/plugin-config` response is authoritative; web `terminal.js` reads it once at load and applies at terminal construction time |
| CLIP-01 | OSC 52 clipboard support; toggle applies live (hot-swappable) | `ClipboardAddon` constructor takes optional `IClipboardProvider`; dispose/re-load in hot-swap useEffect |
| CLIP-02 | Web-served OSC 52 honors read/write capability: read-only viewers cannot have OSC 52 writes | `/api/plugin-config` response should include `perms` or web `terminal.js` checks `window.__perms` (already set at line 111 in `terminal.js`) before loading ClipboardAddon |
| WEB-01 | All three addons vendored under `web/vendor/xterm/addons/`; zero CDN requests; CSP honored | Copy `.js` files; update `web/embed.go`; update `web/vendor/xterm/VERSION`; add `<script>` tags to `terminal.html` |
| WEB-02 | `vendor_drift_test.go` generalized to cover every `@xterm/addon-*` package | Regex change: `(?:xterm\|addon-fit)` → `(?:xterm\|addon-\w+)` plus min-count guard update |
| WEB-03 | Web terminal page conditionally instantiates addons based on `/api/plugin-config` response | Web `terminal.js` fetches `/api/plugin-config?cap=<token>` before terminal construction; branches on response fields |
</phase_requirements>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| WebGL hot-swap (desktop) | Browser / Wails WebView | — | xterm.js addon lifecycle is a browser-tier concern; WebglAddon.dispose() and re-load happen in the DOM |
| Unicode 11 initialization (desktop) | Browser / Wails WebView | — | Applied once at terminal construction time; TerminalPanel mount useEffect owns session init |
| Clipboard OSC 52 (desktop) | Browser / Wails WebView | — | ClipboardAddon wraps browser Clipboard API; hot-swappable in the addon-load useEffect |
| Context-loss detection and fallback | Browser / Wails WebView | App.tsx (callback) | onContextLoss fires in the browser tier; App.tsx receives the one-shot callback to show the toast |
| Software-WebGL preemption detection | Browser / Wails WebView | — | `gl.getParameter(gl.RENDERER)` is a browser-tier probe that runs before loading WebglAddon |
| Plugin config persistence | API / Backend (daemon) | — | Daemon owns settings.json; already wired in Phase 92 (engine.GetPluginSettings) |
| `/api/plugin-config` web endpoint | API / Backend (webserver) | — | WebServer serves the endpoint; wraps `requireCapability` middleware |
| Vendored addon assets | CDN / Static (embedded) | — | Files live in web/vendor/xterm/addons/; served via Go embed.FS same as xterm.js |
| CI version parity gate | CI / Go test | — | `vendor_drift_test.go` in `internal/webserver/` package |
| Web addon initialization | Browser (web page) | — | `web/assets/terminal.js` runs in the browser; reads plugin-config and conditionally loads addons |

---

## Key Technical Decisions

**1. Inert-prop invariant lift — TerminalPanel.tsx line 59**
- [VERIFIED: codebase] `frontend/src/components/TerminalPanel.tsx:59` has `void pluginConfig` — the Phase 92 inert-prop invariant. Phase 93 deletes this line and wires `pluginConfig` into the addon-load useEffect.
- [VERIFIED: codebase] The `App.plugin-event.test.tsx:38-39` test asserts `consumesInEffect` is `false` — this test will go RED when the invariant is lifted, signaling it needs updating.

**2. Two-useEffect architecture for addon lifecycle**
- [VERIFIED: codebase] `TerminalPanel.tsx:66-144` — the `sessionId`-scoped mount useEffect currently loads Unicode11Addon unconditionally and WebglAddon with a try/catch (no `pluginConfig` gating). Phase 93 must restructure this into two useEffects:
  - **Mount useEffect** (`[sessionId]` dep): create Terminal, FitAddon, Unicode11Addon (if `pluginConfig?.unicode11`), and do the initial WebGL probe + load. This useEffect tears down and recreates the terminal when session changes.
  - **Hot-swap useEffect** (`[pluginConfig]` dep, or `[pluginConfig?.webgl, pluginConfig?.clipboard]`): respond to live config changes by disposing/recreating WebglAddon and ClipboardAddon on already-open terminals. Unicode 11 change is intentionally NOT handled here (next-session only).
- [ASSUMED] The hot-swap useEffect refs (webglAddonRef, clipboardAddonRef) need to be stable across renders; `useRef` is the right tool.

**3. WebGL context-loss: existing handler vs. Phase 93 extension**
- [VERIFIED: codebase] `TerminalPanel.tsx:87-90` already has `webglAddon.onContextLoss(() => { console.warn(...); webglAddon.dispose() })`. This is correct fallback but (a) there is no user notification and (b) the console.warn fires but the user sees nothing.
- Phase 93 extends this handler to fire an `onWebGLContextLost` callback prop up to App.tsx. App.tsx renders `WebGLRecoveryBanner` in the `.banner-stack`.
- [VERIFIED: UI-SPEC `93-UI-SPEC.md`] One-shot per session; auto-dismiss after 8 seconds for context-loss; no auto-dismiss for software-preemption toast.

**4. Software-WebGL detection: `gl.getParameter(gl.RENDERER)` patterns**
- [ASSUMED] Standard detection pattern: create an offscreen canvas, get WebGL context, read `gl.getParameter(gl.RENDERER)`. Strings to match: `SwiftShader`, `llvmpipe`, `ANGLE (.*Software Renderer)`, `ANGLE (.*SwiftShader)`. iPad Safari reports `ANGLE (Apple, ANGLE Metal Renderer, Unspecified Version)` for software-rasterized paths — but iPad Safari also frequently simply fails context creation, which the existing try/catch already handles.
- [ASSUMED] The detection should run BEFORE `new WebglAddon()` in the mount useEffect. If detected AND `pluginConfig?.webgl === true`, skip WebGL and fire the `onWebGLContextLost` callback with a `reason: 'software-rasterized'` discriminant.

**5. `vendor_drift_test.go` generalization — exact change**
- [VERIFIED: codebase] `internal/webserver/vendor_drift_test.go:17` current regex: `regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-fit))@([0-9.]+)':`)`
- Generalized regex: `regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-[\w-]+))@([0-9.]+)':`)`
- [VERIFIED: codebase] The minimum-count guard at line 33 checks `len(pnpmVersions) < 2`. After generalization this needs to cover the new addon count. Phase 93 vendors 3 addons (webgl, unicode11, clipboard) plus the existing 2 (xterm, addon-fit) = 5 entries. The guard should check `< 5` after Phase 93.
- [VERIFIED: codebase] The VERSION file format (`@xterm/xterm@6.0.0` per line) already supports arbitrary package names — the parser at lines 50-54 handles any `@scope/pkg@version` line.

**6. `web/vendor/xterm/addons/` directory — new, not existing**
- [VERIFIED: codebase] `web/vendor/xterm/` currently contains: `addon-fit.js`, `VERSION`, `xterm.css`, `xterm.js`. No `addons/` subdirectory exists.
- Phase 93 creates `web/vendor/xterm/addons/addon-webgl.js`, `addon-unicode11.js`, `addon-clipboard.js`. Source files: `frontend/node_modules/@xterm/addon-webgl/lib/addon-webgl.js`, etc. (non-module CJS UMD bundles — the `lib/*.js` files, not `lib/*.mjs`).
- [VERIFIED: codebase] `web/embed.go:9` currently embeds `vendor/xterm/addon-fit.js` by name — Phase 93 must extend the `//go:embed` directive to include the new addons directory.
- [VERIFIED: codebase] `internal/webserver/server.go:426-431` serves `vendor/xterm/` via `fs.Sub(webfs.WebFS, "vendor/xterm")` mounted at `/assets/xterm/`. So `web/vendor/xterm/addons/addon-webgl.js` would be served at `/assets/xterm/addons/addon-webgl.js`. This URL goes in the `terminal.html` `<script>` tags.

**7. `web/terminal.html` script tags — strict ordering required**
- [VERIFIED: codebase] `web/terminal.html:16-18`: load order is `xterm.js` → `addon-fit.js` → `terminal.js`. The new addon scripts must load AFTER `xterm.js` and BEFORE `terminal.js` so global namespace objects (`WebglAddon`, `Unicode11Addon`, `ClipboardAddon`) exist when `terminal.js` runs.
- [VERIFIED: codebase] `no_cdn_regression_test.go` excludes `web/vendor/xterm/` from CDN checks (line 50). The new `addons/` subdir is naturally under `vendor/xterm/` — it will be skipped by the existing guard.
- [VERIFIED: codebase] `no_cdn_regression_test.go:TestSecurity_NoInlineScriptOrStyleInHTML` checks for `<script>` without `src=`. Phase 93 adds `<script src="/assets/xterm/addons/...">` which passes this test (external src tag, not inline).

**8. `/api/plugin-config` endpoint — new Go handler in webserver**
- [VERIFIED: codebase] `WebServer` struct (`server.go:57-84`) has no `pluginSettings` field or engine reference. The webserver's existing capability-gated endpoints only deal with session/relay state. To serve plugin config, the WebServer needs a provider function (same pattern as `sessionResolver` at line 83).
- Pattern to follow: `sessionResolver func(sessionID string) (name, cliType, status, hostname string)` — add `pluginSettingsProvider func() daemon.PluginSettings` field to `WebServer`, set it from the daemon engine at startup.
- [VERIFIED: codebase] `capability_mw.go` is the template for capability-gating. The new endpoint wraps `requireCapability`.
- Endpoint: `GET /api/plugin-config?cap=<token>` — returns JSON of `PluginSettings`. No `{id}` in the path because plugin config is global (not session-scoped). However, the capability token IS session-scoped per the v3.1 model. The handler validates the cap to confirm the caller is a legitimate web client but does not filter config by session.
- [VERIFIED: codebase] `csp_mw.go` does not affect `/api/` routes — the CSP middleware is only wired to HTML-serving routes (`handleTerminalPage` etc., `server.go:398`). The `/api/plugin-config` endpoint needs no CSP header.

**9. BannerStack extension in App.tsx**
- [VERIFIED: codebase] `App.tsx:748-770` renders `.banner-stack` conditionally: `{((webServerMode === 'local' && !localBannerDismissed) || update) && ...}`. Phase 93 must extend this condition to also show when `webglContextLost` or `webglSoftwareDetected` is true.
- [VERIFIED: codebase] `App.tsx:749-770` shows `LocalNetworkBanner` first, then `UpdateBanner`. The new `WebGLRecoveryBanner` renders AFTER `UpdateBanner` (UI-SPEC `93-UI-SPEC.md` line 248 requirement).
- [VERIFIED: codebase] `UpdateBanner.tsx` uses `role="alert"`, `aria-live="polite"`. `LocalNetworkBanner.tsx:33` uses `role="status"`. UI-SPEC specifies `role="status"` + `aria-live="polite"` for `WebGLRecoveryBanner` — follow `LocalNetworkBanner` shape, not `UpdateBanner`.
- [VERIFIED: codebase] CSS for banners: `.update-banner` at `style.css:990` has `padding: 12px 16px; min-height ~53px` implied. UI-SPEC mandates ≤ 53px for the new toasts.

**10. CLIP-02: read-only web client clipboard gate**
- [VERIFIED: codebase] `web/assets/terminal.js:111` sets `window.__perms = perms` from the `/api/sessions/{id}/info` response. Phase 93 web `terminal.js` reads `window.__perms` and only loads `ClipboardAddon` if `perms !== 'read'`.
- [VERIFIED: ClipboardAddon API] `ClipboardAddon` constructor signature: `constructor(base64?: IBase64, provider?: IClipboardProvider)`. For web: `new ClipboardAddon.ClipboardAddon()` (global namespace after UMD script load).

**11. Unicode 11 server-shared requirement (U11-02)**
- [VERIFIED: STATE.md] "Server-shared plugin config for buffer-interpretation plugins (Unicode 11 must match across clients to avoid scrollback divergence); per-client renderer choice (WebGL/DOM) tolerated."
- The web terminal page reads `/api/plugin-config` once at load — the daemon's current setting is authoritative. No per-client override for Unicode 11.

**12. Phase 92 tests that Phase 93 must UPDATE**
- [VERIFIED: codebase] `frontend/src/__tests__/App.plugin-event.test.tsx:38-39` asserts `consumesInEffect` is `false`. Phase 93 lifts the inert-prop invariant, making this assertion go RED. Phase 93 must update this test to assert the consumption IS present (or remove the now-stale negative assertion).
- [VERIFIED: codebase] `frontend/src/__tests__/App.plugin-event.test.tsx` is in `frontend/src/__tests__/` — separate from the component test dir.

---

## Dependencies / Preconditions from Phase 92

| Item | Status | Where |
|------|--------|--------|
| `PluginSettings` struct + defaults (`internal/daemon/plugin_settings.go`) | COMPLETE | Phase 92 Plan 01 |
| `engine.GetPluginSettings()` / `engine.SetPluginSettings()` | COMPLETE | Phase 92 Plan 01 |
| `(*App).GetPluginSettings()` / `(*App).SetPluginSettings()` Wails bindings | COMPLETE | Phase 92 Plan 02 |
| `daemon.PluginSettings` TypeScript type (generated models) | COMPLETE | Phase 92 Plan 02 |
| `settings:plugins` Wails runtime event emission on save | COMPLETE | Phase 92 Plan 02 |
| `PluginsSection.tsx` 8-toggle UI + Save Plugins button | COMPLETE | Phase 92 Plan 03 |
| `App.tsx` EventsOn subscription + `pluginConfig` state + prop drill | COMPLETE | Phase 92 Plan 03 |
| `TerminalPanel.tsx` `pluginConfig?: PluginSettings | null` prop + `void pluginConfig` inert-prop invariant | COMPLETE | Phase 92 Plan 03 |
| Phase 92 inert-prop invariant test in `App.plugin-event.test.tsx` | COMPLETE | Phase 92 Plan 03 — **must be updated in Phase 93** |

---

## Per-Requirement Implementation Sketches

### PLUG-04 — `/api/plugin-config` endpoint

**Go changes (2 files):**

1. `internal/webserver/server.go` — add `pluginSettingsProvider func() daemon.PluginSettings` field to `WebServer` struct; add a setter method `SetPluginSettingsProvider(f func() daemon.PluginSettings)`; register the route in `setupRoutes`:
   ```go
   mux.HandleFunc("GET /api/plugin-config", ws.requireCapability(ws.handleGetPluginConfig))
   ```
2. New handler method on `*WebServer`:
   ```go
   func (ws *WebServer) handleGetPluginConfig(w http.ResponseWriter, r *http.Request) {
       if ws.pluginSettingsProvider == nil {
           http.Error(w, "plugin config unavailable", http.StatusServiceUnavailable)
           return
       }
       s := ws.pluginSettingsProvider()
       writeJSON(w, http.StatusOK, s)
   }
   ```
3. Wire from daemon: in `internal/daemon/engine.go` or wherever `NewWebServer` is called (check `app.go` daemon startup path), call `ws.SetPluginSettingsProvider(engine.GetPluginSettings)`.

**Web-side (terminal.js) changes:**
- After fetching `/api/sessions/{id}/info` (which yields `perms`), also fetch `/api/plugin-config?cap=<token>` to get plugin settings.
- Store in a `pluginConfig` variable, then gate addon loading in `initTerminal()`.

**Note on `requireCapability` and session-binding:** The capability middleware checks `pathID := r.PathValue("id")` against `claims.SID`. Since `/api/plugin-config` has no `{id}` path segment, `pathID` is `""` and the SID check is skipped — any valid cap can call this endpoint. This is appropriate: plugin config is global (not session-specific) but the caller must still be an authenticated web client.

---

### WGL-01 — WebGL hot-swap (desktop)

**TerminalPanel.tsx restructuring:**

Add refs at the top of the component (alongside existing `termRef`, `fitAddonRef`):
```typescript
const webglAddonRef = useRef<WebglAddon | null>(null)
const clipboardAddonRef = useRef<ClipboardAddon | null>(null)
```

Remove the WebGL try/catch from the mount useEffect (`sessionId` dep). Move it to a new hot-swap useEffect:
```typescript
// Hot-swap useEffect — responds to pluginConfig changes on already-open terminals
useEffect(() => {
  const term = termRef.current
  if (!term) return

  // WebGL hot-swap
  if (pluginConfig?.webgl) {
    if (!webglAddonRef.current) {
      // Try to load; software-renderer check first
      if (!isSoftwareWebGL()) {
        try {
          const webglAddon = new WebglAddon()
          webglAddon.onContextLoss(() => {
            webglAddon.dispose()
            webglAddonRef.current = null
            onWebGLContextLost?.({ reason: 'context-loss' })
          })
          term.loadAddon(webglAddon)
          webglAddonRef.current = webglAddon
        } catch (err) {
          // context creation failed — no toast (silent; user tried explicitly)
        }
      } else {
        onWebGLContextLost?.({ reason: 'software-rasterized' })
      }
    }
  } else {
    if (webglAddonRef.current) {
      webglAddonRef.current.dispose()
      webglAddonRef.current = null
    }
  }
  // Clipboard hot-swap — see CLIP-01 section
}, [pluginConfig?.webgl, pluginConfig?.clipboard])
```

**App.tsx changes:**
- Add `onWebGLContextLost` callback prop to `TerminalPanel`.
- Add `webglContextLost` state and `webglSoftwareDetected` state.
- Show `WebGLRecoveryBanner` in `.banner-stack` when either is true.
- Extend banner-stack visibility condition: `|| webglContextLost || webglSoftwareDetected`.

---

### WGL-02 — Context-loss fallback + BannerStack toast

**New component:** `frontend/src/components/WebGLRecoveryBanner.tsx`

```typescript
interface WebGLRecoveryBannerProps {
  reason: 'context-loss' | 'software-rasterized'
  onDismiss: () => void
  className?: string
}
```

- `reason='context-loss'` → message: `Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact.` + auto-dismiss after 8000ms.
- `reason='software-rasterized'` → message: `Hardware acceleration is unavailable on this device. Your terminal is using the standard renderer for the best experience.` + persistent (no auto-dismiss).
- Both: `role="status"`, `aria-live="polite"`, dismiss button `aria-label="Dismiss notification"`, accent left border `#7aa2f7`.
- CSS: new `.webgl-recovery-banner` classes parallel to `.update-banner` (copy structural shape, not extend — UI-SPEC decision).
- One-shot per session: App.tsx manages a boolean `webglBannerDismissed` that, once true, prevents re-showing even if `onWebGLContextLost` fires again.

**New CSS rule in `frontend/src/style.css`:**
- `.settings-panel__description--italic { font-style: italic }` (one new rule per UI-SPEC).
- `.webgl-recovery-banner { ... }` (parallel structure to `.update-banner`).

---

### WGL-03 — Software-rasterized WebGL preemption

**Helper function** (in `TerminalPanel.tsx` or a separate `lib/webglProbe.ts`):
```typescript
function isSoftwareWebGL(): boolean {
  try {
    const canvas = document.createElement('canvas')
    const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl')
    if (!gl) return false
    const renderer = (gl as WebGLRenderingContext).getParameter(
      (gl as WebGLRenderingContext).RENDERER
    )
    return /SwiftShader|llvmpipe|ANGLE.*Software|ANGLE.*SwiftShader/i.test(renderer)
  } catch {
    return false
  }
}
```

Called once in the hot-swap useEffect when WebGL toggle is ON and no webglAddonRef exists yet.

---

### WGL-04 — Web parity (vendored WebGL addon)

- Copy `frontend/node_modules/@xterm/addon-webgl/lib/addon-webgl.js` to `web/vendor/xterm/addons/addon-webgl.js`.
- Add `<script src="/assets/xterm/addons/addon-webgl.js"></script>` to `web/terminal.html` (after `addon-fit.js`, before `terminal.js`).
- Update `web/embed.go` to embed the new addons directory.
- Update `web/vendor/xterm/VERSION` to add the line `@xterm/addon-webgl@0.19.0`.

---

### U11-01 — Unicode 11 settings-controlled, next-session only

**TerminalPanel mount useEffect** — Unicode 11 is applied at session construction:
```typescript
// Inside the sessionId-scoped mount useEffect, AFTER creating term:
if (pluginConfig?.unicode11 !== false) { // default ON if pluginConfig is null
  const unicode11 = new Unicode11Addon()
  term.loadAddon(unicode11)
  term.unicode.activeVersion = '11'
}
```

The `unicode11` addon is NOT in the hot-swap useEffect. If the user toggles Unicode 11, already-open terminals are unaffected. The italic caption in `PluginsSection.tsx` is the affordance.

**PluginsSection.tsx change** — add italic caption below Unicode 11 row:
```tsx
<p className="settings-panel__description settings-panel__description--italic">
  Applies to new sessions you create.
</p>
```

---

### U11-02 — Server-shared Unicode 11 setting (web)

Web `terminal.js`: apply Unicode11 addon at terminal construction if `pluginConfig.unicode11 === true`. No per-client override. Config comes from `/api/plugin-config` response.

Web vendor: copy `frontend/node_modules/@xterm/addon-unicode11/lib/addon-unicode11.js` to `web/vendor/xterm/addons/addon-unicode11.js`. Add script tag. Update VERSION.

---

### CLIP-01 — Clipboard hot-swap (desktop)

Add ClipboardAddon to hot-swap useEffect alongside WebGL:
```typescript
// Clipboard hot-swap (inside same [pluginConfig?.webgl, pluginConfig?.clipboard] useEffect)
if (pluginConfig?.clipboard) {
  if (!clipboardAddonRef.current) {
    const clipAddon = new ClipboardAddon()
    term.loadAddon(clipAddon)
    clipboardAddonRef.current = clipAddon
  }
} else {
  if (clipboardAddonRef.current) {
    clipboardAddonRef.current.dispose()
    clipboardAddonRef.current = null
  }
}
```

Import: `import { ClipboardAddon } from '@xterm/addon-clipboard'` added to `TerminalPanel.tsx` imports.

---

### CLIP-02 — Web clipboard read-only gate

In web `terminal.js`, before loading ClipboardAddon:
```javascript
if (pluginConfig.clipboard && window.__perms !== 'read') {
  var clipAddon = new ClipboardAddon.ClipboardAddon()
  term.loadAddon(clipAddon)
}
```

`window.__perms` is already set at line 111 of `terminal.js` from the `/api/sessions/{id}/info` response — no new fetch needed for this gate.

---

### WEB-01 — Vendor three addons same-origin

**Files to copy:**
```
frontend/node_modules/@xterm/addon-webgl/lib/addon-webgl.js
  → web/vendor/xterm/addons/addon-webgl.js

frontend/node_modules/@xterm/addon-unicode11/lib/addon-unicode11.js
  → web/vendor/xterm/addons/addon-unicode11.js

frontend/node_modules/@xterm/addon-clipboard/lib/addon-clipboard.js
  → web/vendor/xterm/addons/addon-clipboard.js
```

**`web/embed.go` extension:**
```go
//go:embed vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION
//go:embed vendor/xterm/addons/addon-webgl.js vendor/xterm/addons/addon-unicode11.js vendor/xterm/addons/addon-clipboard.js
var WebFS embed.FS
```

**`web/vendor/xterm/VERSION` additions:**
```
@xterm/addon-webgl@0.19.0
@xterm/addon-unicode11@0.9.0
@xterm/addon-clipboard@0.2.0
```

**`web/terminal.html` script tag additions** (after `addon-fit.js`, before `terminal.js`):
```html
<script src="/assets/xterm/addons/addon-webgl.js"></script>
<script src="/assets/xterm/addons/addon-unicode11.js"></script>
<script src="/assets/xterm/addons/addon-clipboard.js"></script>
```

**URL mapping:** `web/vendor/xterm/addons/` is under `vendor/xterm/` which is served via `fs.Sub(webfs.WebFS, "vendor/xterm")` at `/assets/xterm/` — so the URLs are `/assets/xterm/addons/addon-webgl.js` etc. Correct.

---

### WEB-02 — Generalized `vendor_drift_test.go`

**File:** `internal/webserver/vendor_drift_test.go`

**Current regex (line 17):**
```go
var pnpmXtermKeyRe = regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-fit))@([0-9.]+)':`)
```

**Generalized regex:**
```go
var pnpmXtermKeyRe = regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-[\w-]+))@([0-9.]+)':`)
```

**Minimum count guard (line 33)** — update from `< 2` to `< 5` (xterm + addon-fit + addon-webgl + addon-unicode11 + addon-clipboard):
```go
if len(pnpmVersions) < 5 {
    t.Fatalf("failed to parse at least 5 @xterm/* packages from pnpm-lock.yaml: found %v ...", pnpmVersions)
}
```

**Comment update (lines 3-5):** Update to say "every `@xterm/addon-*` package" instead of just "addon-fit".

**VERSION parser (lines 50-54):** Already handles arbitrary `@scope/pkg@version` lines — no change needed.

---

### WEB-03 — Web terminal conditionally loads addons from `/api/plugin-config`

**`web/assets/terminal.js` restructuring:**

Current `initTerminal()` IIFE (line 95-208) creates the terminal and loads FitAddon unconditionally. Phase 93 extends the preamble to also fetch plugin config:

```javascript
(async function initTerminal() {
  // ... existing perms fetch (lines 97-110) ...

  // Phase 93: fetch plugin config
  var pluginConfig = { webgl: true, unicode11: true, clipboard: true }
  if (cap && sessionID) {
    try {
      var pcResp = await fetch(withCap('/api/plugin-config'))
      if (pcResp.ok) {
        pluginConfig = await pcResp.json()
      }
    } catch (e) {
      // Fall through with defaults ON
    }
  }

  // ... existing terminal construction ...
  var term = new Terminal({ ... })
  var fitAddon = new FitAddon.FitAddon()
  term.loadAddon(fitAddon)

  // Phase 93: conditional addon loading
  if (pluginConfig.unicode11) {
    var unicode11Addon = new Unicode11Addon.Unicode11Addon()
    term.loadAddon(unicode11Addon)
    term.unicode.activeVersion = '11'
  }

  if (pluginConfig.webgl && !isSoftwareWebGL()) {
    try {
      var webglAddon = new WebglAddon.WebglAddon()
      webglAddon.onContextLoss(function() {
        webglAddon.dispose()
        showWebGLContextLossBanner()
      })
      term.loadAddon(webglAddon)
    } catch (e) {
      showWebGLContextLossBanner()
    }
  }

  if (pluginConfig.clipboard && window.__perms !== 'read') {
    var clipAddon = new ClipboardAddon.ClipboardAddon()
    term.loadAddon(clipAddon)
  }

  term.open(document.getElementById('terminal'))
  // ...
})()
```

**Banner div in terminal.html** (for web context-loss toast):
```html
<div id="webgl-recovery-banner" style="display:none" role="status" aria-live="polite">
  <!-- Content injected by terminal.js -->
</div>
```

Per UI-SPEC, the web banner must visually match the desktop banner. CSS goes in `web/assets/terminal.css`.

---

## Validation Architecture

`workflow.nyquist_validation` is absent from `.planning/config.json` — treat as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Frontend Framework | Vitest (via `pnpm exec vitest run`) |
| Go Framework | `go test ./...` |
| Config file | `frontend/vite.config.ts` |
| Quick frontend run | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` |
| Full frontend suite | `pnpm test` |
| Go unit run | `go test ./internal/webserver/... -count=1 -run TestXtermVendor` |
| Go full run | `go test ./internal/...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PLUG-04 | `/api/plugin-config` returns plugin settings | unit (Go) | `go test ./internal/webserver/... -run TestPluginConfig -count=1` | ❌ Wave 0 |
| WGL-01 | TerminalPanel hot-swaps WebGL on pluginConfig change | source-inspection | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ (extend) |
| WGL-02 | Context-loss callback fires `onWebGLContextLost` | source-inspection | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ (extend) |
| WGL-03 | `isSoftwareWebGL` probe present in source | source-inspection | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ (extend) |
| WGL-04 | Vendored webgl addon served at `/assets/xterm/addons/` | Go integration | `go test ./internal/webserver/... -run TestAssets_Addons -count=1` | ❌ Wave 0 |
| U11-01 | PluginsSection renders italic caption under Unicode 11 | source-inspection | `pnpm exec vitest run src/components/__tests__/PluginsSection.test.tsx` | ✅ (extend) |
| U11-02 | Web terminal.js applies unicode11 from plugin-config | source-inspection (Go) | `go test ./internal/webserver/... -run TestTerminalJS_PluginConfig -count=1` | ❌ Wave 0 |
| CLIP-01 | ClipboardAddon hot-swap wired in TerminalPanel | source-inspection | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ (extend) |
| CLIP-02 | Web ClipboardAddon gated by `window.__perms` | source-inspection (Go) | `go test ./internal/webserver/... -run TestTerminalJS_ClipboardGate -count=1` | ❌ Wave 0 |
| WEB-01 | Three addon files vendored and embedded | Go unit | `go test ./internal/webserver/... -run TestXtermVendor -count=1` | ✅ (extend) |
| WEB-02 | `vendor_drift_test` fails on addon version mismatch | Go unit | `go test ./internal/webserver/... -run TestXtermVendorVersions -count=1` | ✅ (modify) |
| WEB-03 | terminal.js fetches `/api/plugin-config` at load | source-inspection (Go) | `go test ./internal/webserver/... -run TestTerminalJS_PluginConfig -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx && go test ./internal/webserver/... -count=1`
- **Per wave merge:** `pnpm test && go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/webserver/plugin_config_test.go` — covers PLUG-04 (`GET /api/plugin-config` with/without cap)
- [ ] `internal/webserver/assets_test.go` — extend to cover `/assets/xterm/addons/addon-webgl.js` (WGL-04)
- [ ] `internal/webserver/no_cdn_regression_test.go` — verify `vendor/xterm/addons/` subdir is excluded from CDN check (should work via existing `strings.HasSuffix(path, "vendor/xterm")` guard, but confirm `vendor/xterm/addons` matches the existing skip logic)

---

## Common Pitfalls

### Pitfall 1: Mount useEffect vs. Hot-Swap useEffect — Wrong Dep Array
**What goes wrong:** Putting `pluginConfig` in the `[sessionId]` dep array causes the entire terminal to be destroyed and recreated on every config change.
**Why it happens:** The mount useEffect tears down and recreates the Terminal instance when deps change.
**How to avoid:** Keep two separate useEffects. Session init (Terminal + FitAddon + Unicode11) in `[sessionId]`. Addon hot-swap (WebGL + Clipboard) in `[pluginConfig?.webgl, pluginConfig?.clipboard]`. Tear-down only in cleanup of the mount useEffect.
**Warning signs:** Terminal flickers on Settings save; scrollback is lost when toggling WebGL.

### Pitfall 2: `void pluginConfig` Line Must Be Deleted
**What goes wrong:** Forgetting to delete `TerminalPanel.tsx:59` — the line compiles fine (TypeScript doesn't care), but the test at `App.plugin-event.test.tsx:38-39` stays green (still no useEffect consumption) and the Phase 92 invariant test never flips, hiding the missed implementation.
**How to avoid:** The Phase 93 plan must explicitly include "delete `void pluginConfig` (line 59)" as a task step. The `App.plugin-event.test.tsx` test must be updated to assert consumption IS present.

### Pitfall 3: `pnpmXtermKeyRe` Min-Count Guard
**What goes wrong:** Generalizing the regex without updating the `len(pnpmVersions) < 2` guard causes the test to silently pass even if the generalized regex misses some packages.
**How to avoid:** Update the guard to `< 5` after Phase 93 (xterm + addon-fit + addon-webgl + addon-unicode11 + addon-clipboard). Comment why the number is 5.

### Pitfall 4: embed.FS Wildcard vs. Explicit Paths
**What goes wrong:** Using `//go:embed vendor/xterm/addons/` (directory-only embed) might not work as expected with `fs.Sub`; or missing files fail at build time with unhelpful errors.
**How to avoid:** List each addon file explicitly in `web/embed.go`, matching the existing pattern for `vendor/xterm/addon-fit.js`. Alternatively, use `vendor/xterm/addons/*.js` glob — Go embed supports `*` but NOT `**`.

### Pitfall 5: `web/terminal.html` Inline Script Regression
**What goes wrong:** Adding inline JavaScript to initialize addons or banners triggers `TestSecurity_NoInlineScriptOrStyleInHTML` which finds `<script>` tags without `src=`.
**How to avoid:** ALL JavaScript for the web terminal goes in `web/assets/terminal.js` (extracted per Phase 89 D-06). No inline `<script>` blocks ever. The recovery banner logic must live in `terminal.js`, not inlined in `terminal.html`.

### Pitfall 6: `/api/plugin-config` Route Has No `{id}` — requireCapability Path Check
**What goes wrong:** `requireCapability` does `r.PathValue("id")` and compares to `claims.SID`. If this were to match an empty path ID against the SID, it could falsely reject valid caps.
**Why it's not a problem:** `r.PathValue("id")` returns `""` when no `{id}` param is in the route pattern. The check is `if pathID != "" && claims.SID != pathID` — the condition short-circuits on empty string. Any valid cap passes.
**Confirmation needed:** Verify this in `capability_mw.go:57` (confirmed: `if pathID := r.PathValue("id"); pathID != "" && ...`).

### Pitfall 7: ClipboardAddon in Web — Global Namespace Name
**What goes wrong:** The UMD bundle exports the addon as `ClipboardAddon.ClipboardAddon` (module wrapper pattern — `window.ClipboardAddon = { ClipboardAddon: class }`) not as `ClipboardAddon` directly.
**How to avoid:** Check the UMD bundle's actual global name. Confirm by grepping the `.js` file: `grep -o "root\[[\"'].*[\"']\]" web/vendor/xterm/addons/addon-clipboard.js | head -3`. Same applies to webgl (`WebglAddon.WebglAddon`) and unicode11 (`Unicode11Addon.Unicode11Addon`).

### Pitfall 8: WebGL Context Loss in Desktop vs. Web — Different Callback Paths
**What goes wrong:** Desktop fires `onWebGLContextLost` callback to App.tsx (React state). Web fires `showWebGLContextLossBanner()` (a plain JS function manipulating the `#webgl-recovery-banner` div). Treating them as the same code path leads to confusion.
**How to avoid:** Keep desktop and web implementations completely separate. Desktop: React callback prop pattern. Web: plain DOM manipulation in `terminal.js`.

---

## Code Examples

### Example 1: Hot-swap useEffect dependency slicing

```typescript
// [VERIFIED: TerminalPanel.tsx structure — Pattern 2 useEffects]
// Hot-swap: responds to specific config flags without terminal rebuild
useEffect(() => {
  const term = termRef.current
  if (!term) return
  // WebGL + Clipboard hot-swap logic here
}, [pluginConfig?.webgl, pluginConfig?.clipboard])
// Note: pluginConfig?.unicode11 is intentionally NOT in the dep array.
```

### Example 2: `vendor_drift_test.go` generalized regex

```go
// [VERIFIED: current regex at internal/webserver/vendor_drift_test.go:17]
// CURRENT:
var pnpmXtermKeyRe = regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-fit))@([0-9.]+)':`)
// GENERALIZED (Phase 93):
var pnpmXtermKeyRe = regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-[\w-]+))@([0-9.]+)':`)
```

### Example 3: WebServer plugin settings provider pattern

```go
// [ASSUMED — follows sessionResolver func pattern at server.go:83]
// In WebServer struct:
pluginSettingsProvider func() PluginSettings  // set via SetPluginSettingsProvider

// In setupRoutes:
mux.HandleFunc("GET /api/plugin-config", ws.requireCapability(ws.handleGetPluginConfig))

// Handler:
func (ws *WebServer) handleGetPluginConfig(w http.ResponseWriter, r *http.Request) {
    if ws.pluginSettingsProvider == nil {
        http.Error(w, "plugin config unavailable", http.StatusServiceUnavailable)
        return
    }
    writeJSON(w, http.StatusOK, ws.pluginSettingsProvider())
}
```

### Example 4: Banner-stack condition extension in App.tsx

```tsx
// [VERIFIED: App.tsx:748-770 current condition]
// CURRENT:
{((webServerMode === 'local' && !localBannerDismissed) || update) && (
// PHASE 93:
{((webServerMode === 'local' && !localBannerDismissed) || update || webglContextLost || webglSoftwareDetected) && (
  <div className="banner-stack">
    {/* ... existing banners ... */}
    {(webglContextLost || webglSoftwareDetected) && !webglBannerDismissed && (
      <WebGLRecoveryBanner
        reason={webglSoftwareDetected ? 'software-rasterized' : 'context-loss'}
        onDismiss={() => setWebglBannerDismissed(true)}
        className={webglBannerExiting ? 'banner-exit' : undefined}
      />
    )}
  </div>
)}
```

### Example 5: Unicode 11 italic caption

```tsx
// [VERIFIED: UI-SPEC 93-UI-SPEC.md — exact copy locked]
<p className="settings-panel__description settings-panel__description--italic">
  Applies to new sessions you create.
</p>
```

### Example 6: Software WebGL probe

```typescript
// [ASSUMED — standard WebGL software detection pattern]
function isSoftwareWebGL(): boolean {
  try {
    const canvas = document.createElement('canvas')
    const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl')
    if (!gl) return false
    const renderer = (gl as WebGLRenderingContext).getParameter(
      (gl as WebGLRenderingContext).RENDERER
    ) as string
    return /SwiftShader|llvmpipe|ANGLE.*Software|ANGLE.*SwiftShader/i.test(renderer)
  } catch {
    return false
  }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Unconditional WebGL + Unicode11 load at terminal construction | Settings-gated load via `pluginConfig` prop | Phase 93 | WebGL becomes user-controllable; hot-swap without session restart |
| `vendor_drift_test.go` covers only `@xterm/xterm` + `@xterm/addon-fit` | Generalized regex covers every `@xterm/addon-*` | Phase 93 | CI enforces parity for all 5+ addons |
| Web terminal page loads no addons (vanilla xterm + FitAddon only) | Web terminal page loads webgl/unicode11/clipboard conditionally from same-origin vendor | Phase 93 | Web parity with desktop; zero CDN requests; CSP honored |
| WebGL context loss → console.warn only | Context loss → DOM fallback + BannerStack toast | Phase 93 | User is informed; no silent degradation |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Software WebGL renderer strings include `SwiftShader`, `llvmpipe`, `ANGLE.*Software`, `ANGLE.*SwiftShader` | WGL-03 sketch | iPad Safari may report different renderer string; missing detection means software-WebGL preemption never fires. Mitigation: also handle `context creation refused` path (already covered by try/catch). |
| A2 | UMD bundles export globals as `WebglAddon.WebglAddon`, `Unicode11Addon.Unicode11Addon`, `ClipboardAddon.ClipboardAddon` | WEB-03 sketch | If actual global name is different, web page throws "WebglAddon is not a constructor". Must verify by grepping the `.js` files before use. |
| A3 | `isSoftwareWebGL` probe runs quickly enough (< 16ms) not to delay terminal initialization perceptibly | WGL-03 | On some systems with no GPU, `getContext('webgl')` may take several hundred ms. Mitigation: run probe in mount useEffect before Terminal construction (already on the init path). |
| A4 | `pluginSettingsProvider func() PluginSettings` functional option is the right WebServer extensibility mechanism (vs. interface, vs. field-set) | PLUG-04 | Minor: any callable-based approach works; the pattern already exists in `sessionResolver` at `server.go:83`. |
| A5 | `frontend/node_modules/@xterm/addon-webgl/lib/addon-webgl.js` (CJS UMD) is the correct file for web vendoring (not the `.mjs` file) | WEB-01 | If `.mjs` is used, browser will try to import as ES module; `<script src="...">` without `type="module"` will get a MIME error. Must use the `.js` (CJS UMD) file. |

---

## Open Questions / Risks

1. **`PluginSettings` import path in webserver package**
   - What we know: `WebServer` is in package `webserver` (`internal/webserver/`); `PluginSettings` is in `internal/daemon/`.
   - What's unclear: Does importing `daemon.PluginSettings` in `webserver` create a circular dependency? (The daemon package already imports webserver to construct the server.)
   - Recommendation: Define a minimal `PluginConfig` struct in `internal/webserver/` that mirrors the daemon struct, OR use `func() any` + JSON marshaling, OR move `PluginSettings` to a shared `internal/pluginconfig/` package. **Most likely:** avoid the import problem entirely by passing `func() map[string]bool` or `func() []byte` (pre-marshaled JSON) instead of `daemon.PluginSettings` — the handler just writes the bytes directly.
   - **Risk level:** MEDIUM — could block PLUG-04 implementation if not resolved upfront.

2. **App.tsx banner-stack condition expansion**
   - What we know: The `.banner-stack` div is only rendered when `(webServerMode === 'local' && !localBannerDismissed) || update` (App.tsx:748).
   - What's unclear: The webgl toasts can fire on desktop where `webServerMode` is typically `'tailscale'` or `''` — the condition must be expanded or the toasts will never render on Tailscale-mode users.
   - Recommendation: Extend the condition to `|| webglContextLost || webglSoftwareDetected` as shown in Code Example 4.

3. **One-shot per session: sessionStorage vs React state**
   - What we know: UI-SPEC specifies "once dismissed, never re-shows in the same session". For desktop this is straightforward React state (reset on app restart). For web it must survive hot reloads.
   - Recommendation: For desktop, use React state in App.tsx (`webglBannerDismissed: boolean`). For web, use `sessionStorage.setItem('webgl-banner-shown', '1')` — check before calling `showWebGLContextLossBanner()`.

4. **`vendor_drift_test.go` pnpm-lock format stability**
   - The test parses `pnpm-lock.yaml` line by line with a fixed regex. The format `  '@xterm/addon-foo@version':` matches the current pnpm v8 lockfile format.
   - Risk: pnpm v9 changed the lockfile format. The test already has a comment acknowledging format drift risk (`see 89-RESEARCH.md Q3`). The generalized regex uses the same line format as the existing one — no new risk introduced.

---

## Files to Create / Modify

### New Files
| File | Why |
|------|-----|
| `frontend/src/components/WebGLRecoveryBanner.tsx` | BannerStack toast for context-loss + software-WebGL toasts (WGL-02, WGL-03) |
| `frontend/src/components/__tests__/WebGLRecoveryBanner.test.tsx` | Source-inspection tests for copy + dismiss button + aria attributes |
| `web/vendor/xterm/addons/addon-webgl.js` | Copied from node_modules for web vendoring (WEB-01) |
| `web/vendor/xterm/addons/addon-unicode11.js` | Copied from node_modules for web vendoring (WEB-01) |
| `web/vendor/xterm/addons/addon-clipboard.js` | Copied from node_modules for web vendoring (WEB-01) |

### Modified Files
| File | Change | Requirement |
|------|--------|-------------|
| `frontend/src/components/TerminalPanel.tsx` | Delete `void pluginConfig` (line 59); add hot-swap useEffect; add `webglAddonRef`/`clipboardAddonRef`; add `isSoftwareWebGL()` helper; add `onWebGLContextLost` callback prop; add `ClipboardAddon` import | WGL-01..03, CLIP-01 |
| `frontend/src/components/PluginsSection.tsx` | Add italic caption `<p>` below Unicode 11 row | U11-01 |
| `frontend/src/App.tsx` | Add `webglContextLost`/`webglSoftwareDetected`/`webglBannerDismissed` states; expand banner-stack condition; render `WebGLRecoveryBanner`; pass `onWebGLContextLost` callback to TerminalPanel | WGL-02, WGL-03 |
| `frontend/src/style.css` | Add `.settings-panel__description--italic { font-style: italic }` + `.webgl-recovery-banner { ... }` CSS block | U11-01, WGL-02 |
| `frontend/src/__tests__/App.plugin-event.test.tsx` | Update `consumesInEffect` assertion from `false` to `true` (Phase 93 lifts inert-prop invariant) | WGL-01 |
| `internal/webserver/vendor_drift_test.go` | Generalize `pnpmXtermKeyRe` regex; update min-count guard to 5; update comment | WEB-02 |
| `internal/webserver/server.go` | Add `pluginSettingsProvider` field to `WebServer`; add `SetPluginSettingsProvider` method; register `GET /api/plugin-config` route | PLUG-04 |
| `web/embed.go` | Extend `//go:embed` to include `vendor/xterm/addons/addon-webgl.js`, `addon-unicode11.js`, `addon-clipboard.js` | WEB-01 |
| `web/vendor/xterm/VERSION` | Add three new addon version lines | WEB-01, WEB-02 |
| `web/terminal.html` | Add three `<script src="/assets/xterm/addons/...">` tags | WEB-01 |
| `web/assets/terminal.js` | Add `/api/plugin-config` fetch; conditional addon loading; `isSoftwareWebGL()` probe; `showWebGLContextLossBanner()` function; recovery banner DOM manipulation | WEB-03, WGL-04, U11-02, CLIP-02 |
| `web/assets/terminal.css` | Add `.webgl-recovery-banner { ... }` CSS (53px, TokyoNight palette, matches desktop visually) | WGL-04 |

### Daemon wiring (engine startup)
| File | Change | Requirement |
|------|--------|-------------|
| Daemon startup path (check `internal/daemon/api.go` or where WebServer is constructed) | Call `ws.SetPluginSettingsProvider(func() daemon.PluginSettings { return engine.GetPluginSettings() })` | PLUG-04 |

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | — |
| V3 Session Management | No | — |
| V4 Access Control | Yes | `requireCapability` middleware; ClipboardAddon gated by `window.__perms` |
| V5 Input Validation | Yes | Handler uses existing `writeJSON` (no untrusted input parsed) |
| V6 Cryptography | No | — |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unauthenticated plugin config read | Information Disclosure | `requireCapability` wraps `/api/plugin-config` — any caller without a valid cap gets 401 |
| OSC 52 clipboard injection via read-only web session | Tampering | `window.__perms !== 'read'` gate before loading ClipboardAddon on web |
| Vendored addon file serving CDN content | Tampering | `vendor_drift_test.go` CI gate; `no_cdn_regression_test.go` guards against CDN URLs |
| WebGL context loss → silent degradation | Denial of Service | One-shot BannerStack toast informs user; no auto-retry loop |

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `@xterm/addon-webgl` | WGL-01, WGL-04 | ✓ | 0.19.0 | — (already in node_modules) |
| `@xterm/addon-unicode11` | U11-01, U11-02 | ✓ | 0.9.0 | — (already in node_modules) |
| `@xterm/addon-clipboard` | CLIP-01, CLIP-02 | ✓ | 0.2.0 | — (already in node_modules) |
| `web/vendor/xterm/addons/` | WEB-01 | ✗ (dir doesn't exist) | — | Create in Phase 93 |
| pnpm | Dependency management | ✓ | (project default) | — |
| Go 1.22+ | `//go:embed` glob support | ✓ | (project standard) | — |

---

## Sources

### Primary (HIGH confidence)
- `internal/webserver/vendor_drift_test.go` — current regex, VERSION file format, test structure
- `internal/webserver/server.go` — WebServer struct, route registration, asset serving pattern, `sessionResolver` functional option pattern
- `internal/webserver/capability_mw.go` — `requireCapability` middleware; path-ID check behavior
- `internal/webserver/csp_mw.go` — CSP policy (script-src 'self' honored; no change needed for addon JS files served via /assets/xterm/)
- `frontend/src/components/TerminalPanel.tsx` — current addon load, `void pluginConfig` inert-prop invariant at line 59, `onContextLoss` handler at lines 87-90
- `frontend/src/App.tsx:748-770` — banner-stack rendering, condition structure
- `frontend/src/style.css:990-1001, 1561-1580` — `.update-banner` shape, `.banner-stack` max-height contract (3 × 53px)
- `frontend/src/components/UpdateBanner.tsx` — banner vocabulary (role, aria-live, dismiss button shape)
- `frontend/src/components/LocalNetworkBanner.tsx:33` — `role="status"` (UI-SPEC specifies same for WebGLRecoveryBanner)
- `web/embed.go` — embed.FS directives to extend
- `web/vendor/xterm/VERSION` — VERSION file format
- `web/terminal.html` — current script tag order
- `web/assets/terminal.js` — `window.__perms` assignment at line 111; perms fetch pattern
- `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-UI-SPEC.md` — locked UI contract
- `.planning/phases/92-plugin-settings-foundation/92-PATTERNS.md` — Phase 92 pattern map with exact file:line references
- `.planning/phases/92-plugin-settings-foundation/92-03-SUMMARY.md` — Phase 92 inert-prop invariant implementation details
- `frontend/node_modules/@xterm/addon-clipboard/typings/addon-clipboard.d.ts` — ClipboardAddon constructor signature

### Secondary (MEDIUM confidence)
- `frontend/src/__tests__/App.plugin-event.test.tsx` — `consumesInEffect` assertion at line 38-39 (must be updated)
- `internal/webserver/no_cdn_regression_test.go` — vendor/xterm skip logic (applies to addons subdir automatically)

### Tertiary (LOW confidence — ASSUMED)
- Software WebGL renderer string patterns (SwiftShader, llvmpipe, ANGLE variants) — training knowledge, verify against browser documentation
- UMD global namespace names for each addon — verify by grepping the `.js` files

---

## Metadata

**Confidence breakdown:**
- Standard Stack: HIGH — all addons already in node_modules; versions verified
- Architecture: HIGH — all patterns verified against existing codebase with file:line citations
- Pitfalls: HIGH — most pitfalls derived from reading actual code, not speculation
- Web endpoint pattern: MEDIUM — WebServer struct documented, functional option pattern confirmed; circular import question is OPEN

**Research date:** 2026-05-04
**Valid until:** 2026-06-04 (stable ecosystem; xterm.js addon APIs are stable)

---

## RESEARCH COMPLETE
