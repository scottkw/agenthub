# Architecture Research — v3.2 xterm.js Plugin Suite

**Milestone:** v3.2 (Issue #36) — curated xterm.js addons + Settings-controlled enable/disable
**Researched:** 2026-05-03
**Confidence:** HIGH (verified against actual repo files, package.json, embed.go, csp_mw.go, daemon engine.go, TerminalPanel.tsx, served terminal.js)

---

## 1. Existing Architecture Snapshot (what's already built)

These are the load-bearing facts the v3.2 design must integrate with:

### 1.1 Desktop Frontend (Wails-bundled React)

- `frontend/src/components/TerminalPanel.tsx` is the per-tab xterm.js host. **One Terminal instance per `sessionId`**, created in a `useEffect([sessionId])`, disposed on unmount. Inactive tabs are kept mounted with `display: none` (buffer preservation).
- **Already loads three addons today** (verified):
  - `@xterm/addon-fit` — required for sizing
  - `@xterm/addon-unicode11` — already wired with `term.unicode.activeVersion = '11'` and `allowProposedApi: true`
  - `@xterm/addon-webgl` — already loaded inside a try/catch with `onContextLoss` → `dispose()` fallback
- `@xterm/addon-clipboard` is in `package.json` but **not wired** (legacy from v1.0 TERM-05).
- Theme is a single global `ITheme` prop passed to every panel. Re-applied via `term.options.theme = ...; term.clearTextureAtlas(); term.refresh(...)` (THM-03 pattern).
- Font size is per-tab (held in `App.tsx` state, persisted to localStorage indirectly through StatusBar font controls).

### 1.2 Daemon Settings Persistence (`internal/daemon/engine.go`)

The struct (read directly from engine.go lines 67–71):

```go
type daemonSettings struct {
    CLIPaths         map[string]string `json:"cliPaths,omitempty"`
    StartMinimized   bool              `json:"startMinimized,omitempty"`
    AutoCloseSession *bool             `json:"autoCloseSession,omitempty"`
}
```

- One file: `<configDir>/settings.json`. Loaded once at `NewSessionEngine`. Saved synchronously (`saveSettingsToDisk` under `e.mu`) on every setter.
- Wails bindings on `app.go` follow the pattern `Get<X>() T` / `Set<X>(v T) error` — no batched save.
- Three-state Save button is a frontend pattern in `SettingsTab.tsx` (idle → "Saving…" → "Saved!" 1.5s) — **not** a backend RPC contract.

### 1.3 Web-Served Terminal (`web/`, `internal/webserver/`)

- `web/terminal.html` + `web/assets/terminal.js` is a **completely separate xterm.js initialization** from the desktop. It currently loads only **fit**, no unicode11, no webgl, no theme, hard-coded font.
- `web/embed.go` is a single `//go:embed` block. Vendored xterm runtime is at `web/vendor/xterm/` (xterm.js, xterm.css, addon-fit.js, VERSION manifest).
- `web/vendor/xterm/VERSION` is asserted byte-for-byte against `frontend/pnpm-lock.yaml` by `vendor_drift_test.go` (D-04/D-20). **Adding a new vendored addon means adding it both to `embed.go` AND `VERSION` AND the drift test.**
- CSP from `internal/webserver/csp_mw.go`:
  ```
  default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline';
  connect-src 'self' wss://<host>; img-src 'self' data:;
  font-src 'self'; base-uri 'none'; form-action 'self';
  frame-ancestors 'none'
  ```
  Note: `worker-src` is **not** listed → falls back to `default-src 'none'` → workers blocked. addon-image's worker support has already been removed upstream, so this is fine. `blob:` is not allowed for any directive yet.
- `no_cdn_regression_test.go` asserts no inline `<script>` or `<style>` in HTML, no CDN URLs anywhere except `vendor/xterm/`.

### 1.4 Wails Bindings (`app.go`)

Existing settings RPCs follow this exact shape:

```go
func (a *App) GetCLIPaths() (map[string]string, error)
func (a *App) GetStartMinimized() bool
func (a *App) SetStartMinimized(val bool) error
func (a *App) GetAutoCloseSession() bool
func (a *App) SetAutoCloseSession(val bool) error
```

Each delegates to `a.client.<X>` which round-trips JSON over the Unix socket / named pipe to the daemon. **No streaming, no events for settings changes** — the frontend reads on mount and writes on user action.

---

## 2. Plugin Lifecycle & Hot-Swap Matrix (Question 1 + 2)

The xterm.js docs are intentionally vague on load-order requirements ([WebGL README](https://github.com/xtermjs/xterm.js/blob/master/addons/addon-webgl/README.md) shows `terminal.open(element); terminal.loadAddon(new WebglAddon());` — i.e. **after** open). What actually matters in practice is:

- **Renderer addons** (WebGL) must be loaded **after `term.open()`** because they hook into the renderer registry that `open()` initialises.
- **Buffer-mutating addons** (Unicode11) must be loaded **before any data is written** so the unicode service uses the new tables. In our code we do this immediately after `term.open()`, which is correct because we have not written anything yet.
- **Decoder addons** (image) hook into the parser; must be loaded **before any image escape sequence is written** to be effective. After `term.open()` is fine; before any `term.write(data)` is required.
- **UI-overlay addons** (search, web-links) hook events on the renderer/dom layer; must be loaded **after `term.open()`** because they need a DOM element to attach to.
- **Buffer-reader addons** (serialize) read from buffer; can be loaded any time, including lazily on first use.

| Plugin | Must attach before… | Hot-swap on live terminal? | UI indicator |
|---|---|---|---|
| `@xterm/addon-webgl` | After `open()`, before first `write()` for clean texture atlas | **Disable hot:** call `dispose()`, falls back to canvas. **Enable hot:** call `loadAddon(new WebglAddon())` after open. ✓ Both directions hot. | None (live) |
| `@xterm/addon-search` | After `open()` | ✓ Fully hot — `loadAddon`/`dispose` at any time | None (live) |
| `@xterm/addon-web-links` | After `open()`, **before** first link should be clickable in scrollback | **Hot enable:** ✓ (matches new output only). **Hot disable:** ✓ via `dispose()`, but already-rendered link decorations only clear on next refresh. **Config change** (clickHandler, regex): requires dispose+reload of the addon — within one terminal, no full session reload. | "Existing scrollback links update on next render" (minor) |
| `@xterm/addon-unicode11` | Before any `write()` (for buffer correctness), **and** must run `term.unicode.activeVersion = '11'` to take effect | **Cannot hot-swap once buffer has content.** Switching unicode tables mid-stream re-interprets nothing — the existing buffer is already laid out. New writes use the new tables; old content keeps wrong widths. | **"Applies to new sessions only"** ⚠ |
| `@xterm/addon-image` | After `open()`, before first image escape | ✓ Hot enable: future images decoded. ✓ Hot disable: future images render as garbage escape codes (but already-rendered images stay until scrolled out). Lossy mid-stream — best treated as new-session for clean UX. | **"Applies to new sessions only"** ⚠ (recommended; technically hot but messy) |
| `@xterm/addon-serialize` | Any time | ✓ Fully hot — purely passive reader, no buffer interaction | None (live) |

**Decision:** four are truly hot-swappable (webgl, search, web-links, serialize), two require a new-session badge (unicode11, image). The Settings UI exposes one shared "applies to new sessions only" affordance — a small badge next to the toggle row.

---

## 3. State Propagation (Question 3)

```
┌──────────────────────────┐
│ SettingsTab.tsx          │
│  - PluginsSection        │  user toggles + per-plugin config
│  - per-plugin <Config/>  │
└──────────┬───────────────┘
           │ Wails: GetPluginSettings / SetPluginSettings
           ▼
┌──────────────────────────┐
│ app.go bindings          │  thin DaemonClient pass-through
└──────────┬───────────────┘
           │ HTTP/JSON over Unix socket / pipe
           ▼
┌──────────────────────────┐
│ daemon engine.go         │
│  daemonSettings.Plugins  │  persisted in settings.json
└──────────┬───────────────┘
           │ on save: emit Wails event "settings:plugins"
           ▼
┌──────────────────────────┐
│ App.tsx                  │  EventsOn("settings:plugins")
│  pluginConfig state      │  passed as prop to TerminalPanel
└──────────┬───────────────┘
           ▼
┌──────────────────────────┐
│ TerminalPanel.tsx        │
│  useEffect([pluginConfig]) loads/disposes addons on live term
│  hot-swappable: apply    │
│  non-hot: ignored until  │
│            new sessionId │
└──────────────────────────┘
```

**Key choices:**

1. **Daemon is the source of truth, not localStorage.** Plugin config is shared across desktop and (potentially) web — localStorage would split the state. This matches CLI paths / startMinimized / autoClose precedent (`daemonSettings`).
2. **Frontend mirror via Wails event.** Existing pattern: when `Set<X>` succeeds, emit a runtime event so other panels/components react. Today the event mechanism already exists (`session:exit`, `app:quit-requested`, `tray:focus-session`). Add `settings:plugins` for live propagation.
3. **TerminalPanel becomes config-aware.** Add a `pluginConfig: PluginConfig` prop. A new `useEffect([pluginConfig])` diffs against a ref of the previous config, calls `loadAddon` / `addonRef.dispose()` per plugin. Non-hot plugins are stamped at terminal create time only.
4. **No per-tab override in v3.2.** Plugin config is global (matches global theme — see Out-of-Scope decision). This keeps the Settings UI simple. Per-tab overrides can be a future milestone.

### 3.1 Live-terminal vs new-session decision logic

```ts
// inside TerminalPanel.tsx
useEffect(() => {
  if (!termRef.current) return
  // WebGL — hot both ways
  reconcileWebgl(termRef.current, webglRef, pluginConfig.webgl)
  // Search — hot
  reconcileSearch(termRef.current, searchRef, pluginConfig.search)
  // WebLinks — hot, but config change requires dispose+reload
  reconcileWebLinks(termRef.current, webLinksRef, pluginConfig.webLinks)
  // Serialize — hot
  reconcileSerialize(termRef.current, serializeRef, pluginConfig.serialize)
  // Unicode11 — IGNORED here; only applied at create time in [sessionId] effect
  // Image — IGNORED here; only applied at create time in [sessionId] effect
}, [pluginConfig])
```

The new-session-only plugins are baked in by reading `pluginConfig` inside the existing `useEffect([sessionId])` body. Toggling them after creation does not affect existing terminals; new tabs pick up the new config.

### 3.2 Web-served sessions (per-client vs per-session)

**Recommendation: per-server (one global plugin config), with a path to per-client opt-out via URL params if demand emerges.**

Argued, not assumed:

| Option | Pros | Cons | Verdict |
|---|---|---|---|
| **Per-server, server-injected** (chosen) | One source of truth; consistent UX across desktop and web; trivially implements vendoring discipline (server only ships the addon JS for enabled plugins); matches existing "tailnet membership = access control, no per-user state" model | A user on iPhone Safari can't disable WebGL even if their browser dies on it | Ship this; iterate on demand |
| Per-client localStorage | Each browser remembers its own choice | Splits UX with desktop; complicates vendored asset shipping (must ship all addon JS regardless); no migration path to authenticated multi-user | No |
| Per-session at session-create time | Plugin set frozen at session create; matches existing CLI args model | Confusing UX — same session looks different to different viewers; what's the "config" for a remote attach? | No |

**Implementation:** the served `terminal.js` is rendered (today it's static-embedded) — to inject config we have two choices:
- **A.** Add a small `/api/plugin-config` JSON endpoint; `terminal.js` fetches it before constructing Terminal (one extra round-trip, ~10ms LAN). CSP `connect-src 'self'` already permits this.
- **B.** Embed config as a `<meta name="agenthub-plugins" content='{"webgl":true,...}'>` tag in `terminal.html` rendered by the Go handler.

Option A wins because `terminal.html` is currently a static embedded file (`//go:embed terminal.html`); switching it to a template requires touching the embed.go boundary, while a JSON endpoint reuses the existing `/api/sessions/{id}/info` pattern and the existing capability gate.

---

## 4. Per-Plugin Config Plumbing (Question 4)

Two of the six plugins are configurable in v3.2 scope:

- **Search:** `regex`, `caseSensitive`, `wholeWord`, `incremental` (all booleans, addon-level options on each call to `findNext`/`findPrevious`)
- **WebLinks:** `clickModifier` (none/cmd/ctrl/alt — what does the user have to hold to click a link? defaults to `none` on web, `alt` on desktop in practice), `urlRegex` override (advanced, hide behind disclosure)

### 4.1 Go struct extension

```go
// internal/daemon/engine.go
type daemonSettings struct {
    CLIPaths         map[string]string `json:"cliPaths,omitempty"`
    StartMinimized   bool              `json:"startMinimized,omitempty"`
    AutoCloseSession *bool             `json:"autoCloseSession,omitempty"`
    Plugins          *PluginSettings   `json:"plugins,omitempty"` // NEW
}

// New file: internal/daemon/plugin_settings.go
type PluginSettings struct {
    WebGL     PluginToggle    `json:"webgl"`
    Search    SearchConfig    `json:"search"`
    Image     PluginToggle    `json:"image"`
    WebLinks  WebLinksConfig  `json:"webLinks"`
    Unicode11 PluginToggle    `json:"unicode11"`
    Serialize PluginToggle    `json:"serialize"`
}

type PluginToggle struct {
    Enabled bool `json:"enabled"`
}

type SearchConfig struct {
    Enabled        bool `json:"enabled"`
    DefaultRegex   bool `json:"defaultRegex,omitempty"`
    DefaultCaseSensitive bool `json:"defaultCaseSensitive,omitempty"`
    DefaultWholeWord     bool `json:"defaultWholeWord,omitempty"`
}

type WebLinksConfig struct {
    Enabled       bool   `json:"enabled"`
    ClickModifier string `json:"clickModifier,omitempty"` // "none"|"cmd"|"ctrl"|"alt"
    URLRegex      string `json:"urlRegex,omitempty"`      // empty = use addon default
}
```

**Defaults policy:** `*PluginSettings` is a pointer so `nil` means "first run, use built-in defaults". The defaults function lives in `plugin_settings.go` with all six enabled — matching existing user expectation (today TerminalPanel always loads webgl + unicode11; turning them off would be a regression). Image is the one fresh face — recommend default-on as well; sixel/iTerm protocol is well-established.

### 4.2 Wails bindings (one Get/Set pair, not per-field)

```go
// app.go
func (a *App) GetPluginSettings() (*daemon.PluginSettings, error) { ... }
func (a *App) SetPluginSettings(s daemon.PluginSettings) error    { ... }
```

Single struct round-trip is simpler than 12 Get/Set methods. Frontend keeps a `pluginConfig: PluginSettings` reducer; user edits → debounced (~300ms) → `SetPluginSettings` → daemon emits `settings:plugins` → all subscribers update.

### 4.3 React side

- New `frontend/src/types/plugins.ts` exports the `PluginSettings` mirror (TypeScript matches the Go JSON shape).
- New component `frontend/src/components/PluginsSection.tsx` rendered as another `<h3>` block in the existing scrollable `SettingsTab.tsx` (same pattern as Appearance, Web Server, Paths).
- Per-plugin config panels are inline collapsibles inside the toggle row — disclosure pattern keeps the section compact.
- Pass `pluginConfig` from `App.tsx` to every `<TerminalPanel>` as a prop.

---

## 5. Web vs Desktop Parity (Question 5)

**Decision: SHARE plugin config (server-injected), but accept that the web client gets a strictly-subset feature set.**

| Plugin | Desktop | Web | Reason |
|---|---|---|---|
| WebGL | ✓ | ✓ | Both have GPU |
| Unicode11 | ✓ | ✓ | Buffer-level correctness — should match |
| Search | ✓ | needs UI | Search overlay UI is more work on web; **defer search UI to a phase or future milestone**; backend support is "free" once the addon ships |
| WebLinks | ✓ | ✓ | Same code path |
| Image | ✓ | ✓ but CSP-sensitive | See §6 |
| Serialize | ✓ | not exposed | Serialization is a desktop-side action ("Save terminal as text") — no web UI in v3.2 scope |

**Rationale:** web and desktop both render the same xterm.js — divergence is technical-debt-by-design. The only legitimate split is when the surrounding app shell differs (search needs an input box, serialize needs a "Save" button — those are app-shell concerns, not addon concerns). For correctness-class plugins (webgl, unicode11, weblinks, image), parity is the simpler, more honest UX.

**Web-side wiring (`web/assets/terminal.js`):** rewrite the existing `initTerminal` IIFE to (1) fetch `/api/plugin-config?cap=...` after the existing perms fetch, (2) construct addons gated on the config, (3) load each addon at the appropriate lifecycle phase. Addon JS files are loaded as additional `<script src="/assets/xterm/addon-*.js">` tags in `terminal.html` — **all addons are loaded eagerly, the config decides which to instantiate.** This avoids dynamic script injection (would conflict with `script-src 'self'` if naïvely implemented and is unnecessary for ~50KB of vendored JS).

---

## 6. Vendoring + CSP (Question 6)

### 6.1 Vendored asset additions

Each addon ships as one or two files:

| Addon | Files (UMD bundle) | Location | embed.go addition |
|---|---|---|---|
| addon-webgl | `addon-webgl.js` | `web/vendor/xterm/` | `vendor/xterm/addon-webgl.js` |
| addon-search | `addon-search.js` | `web/vendor/xterm/` | `vendor/xterm/addon-search.js` |
| addon-image | `addon-image.js` | `web/vendor/xterm/` | `vendor/xterm/addon-image.js` |
| addon-web-links | `addon-web-links.js` | `web/vendor/xterm/` | `vendor/xterm/addon-web-links.js` |
| addon-unicode11 | `addon-unicode11.js` | `web/vendor/xterm/` | `vendor/xterm/addon-unicode11.js` |
| addon-serialize | `addon-serialize.js` | `web/vendor/xterm/` | `vendor/xterm/addon-serialize.js` |

`web/vendor/xterm/VERSION` gets six new lines, one per addon. `vendor_drift_test.go` regex (`@xterm/(?:xterm|addon-fit)`) **must be updated** to include the new packages — this is a load-bearing test, not a check-in formality (it will fail CI loudly if forgotten, which is the desired behavior).

### 6.2 CSP implications per plugin

| Plugin | CSP needs | Status today | Action |
|---|---|---|---|
| WebGL | `script-src 'self'` only — runs as ES module, GPU calls don't need CSP | ✓ already permitted | None |
| Search | `script-src 'self'`, possibly `style-src 'self' 'unsafe-inline'` if it injects highlight styles | ✓ already permitted (style-src amended in v3.1 D-09) | None |
| Image | `script-src 'self'`, **canvas** (no CSP directive — canvas is always allowed), reads from terminal stream not network | ✓ workers were removed upstream so `worker-src` not needed | **Verify in Phase research** that no `data:` script blob or `blob:` URL is constructed for image rendering. Current `img-src 'self' data:` covers `<img>`-tag fallbacks if any |
| WebLinks | `script-src 'self'`, click handler uses `window.open(url)` which is not CSP-gated for user-initiated navigation | ✓ permitted | None |
| Unicode11 | `script-src 'self'` only | ✓ permitted | None |
| Serialize | `script-src 'self'` only — pure JS, returns string | ✓ permitted | None |

**The big unknown is addon-image.** [Upstream README](https://github.com/xtermjs/xterm.js/tree/master/addons/addon-image) says workers are removed, but does not enumerate canvas/blob behavior. **Phase task: spawn a researcher in the image phase to verify by reading `addon-image.js` source whether any `URL.createObjectURL` (blob:) or `data:` script construction is present.** If yes, CSP must be amended (analogous to the v3.1 D-09 style-src amendment). If no, CSP is unchanged.

### 6.3 No-CDN regression test

`internal/webserver/no_cdn_regression_test.go` walks `web/` excluding `vendor/xterm/` looking for CDN URLs and inline `<script>`. As long as the new addon JS lives under `vendor/xterm/` and HTML stays inline-free, the test continues to pass without modification.

### 6.4 Frontend (desktop) addon bundling

Vite bundles addons into the main JS bundle — no embed.FS implication. `pnpm-lock.yaml` becomes the source of truth for desktop addon versions, and `vendor_drift_test.go` enforces that the vendored web/ versions match.

---

## 7. Component Map: New vs Modified

### 7.1 New files

| File | Purpose |
|---|---|
| `internal/daemon/plugin_settings.go` | `PluginSettings`, `SearchConfig`, `WebLinksConfig` types + defaults |
| `internal/daemon/engine_plugins_test.go` | Round-trip persistence test (load → save → re-load) |
| `internal/webserver/plugin_config.go` | `GET /api/plugin-config?cap=...` handler returning JSON |
| `internal/webserver/plugin_config_test.go` | Auth + JSON shape tests |
| `frontend/src/types/plugins.ts` | TS mirror of `PluginSettings` |
| `frontend/src/components/PluginsSection.tsx` | Settings section with toggle list |
| `frontend/src/components/PluginConfigSearch.tsx` | Per-plugin config panel for Search |
| `frontend/src/components/PluginConfigWebLinks.tsx` | Per-plugin config panel for WebLinks |
| `frontend/src/lib/pluginReconcile.ts` | Pure functions: `reconcileWebgl`, `reconcileSearch`, `reconcileWebLinks`, `reconcileSerialize` (returns `{addon: AddonRef, dispose: () => void}`) |
| `frontend/src/components/__tests__/PluginsSection.test.tsx` | Source-inspection tests |
| `web/vendor/xterm/addon-{webgl,search,image,web-links,unicode11,serialize}.js` | Vendored UMD bundles |

### 7.2 Modified files

| File | Change |
|---|---|
| `frontend/package.json` | Add `@xterm/addon-search`, `@xterm/addon-image`, `@xterm/addon-web-links`, `@xterm/addon-serialize` dependencies; remove unused `@xterm/addon-clipboard` (cleanup) |
| `frontend/src/components/TerminalPanel.tsx` | Accept `pluginConfig` prop; refactor addon loading to a reconcile pattern; new useEffect for hot-swap |
| `frontend/src/App.tsx` | `pluginConfig` state + `EventsOn("settings:plugins")` subscription; pass prop to all TerminalPanel instances |
| `frontend/src/components/SettingsTab.tsx` | Insert `<PluginsSection>` after Appearance section |
| `app.go` | Add `GetPluginSettings()` / `SetPluginSettings()` Wails bindings; emit `settings:plugins` after Set |
| `internal/daemon/engine.go` | Extend `daemonSettings`; add `GetPluginSettings`/`SetPluginSettings` methods on engine |
| `internal/daemon/api.go` | New HTTP routes `GET /settings/plugins`, `PUT /settings/plugins` |
| `internal/daemon/client.go` | Mirror RPC client methods |
| `internal/webserver/server.go` | Wire `/api/plugin-config` route through capability gate |
| `web/embed.go` | Add six addon files to `//go:embed` directives |
| `web/vendor/xterm/VERSION` | Six new package@version lines |
| `web/terminal.html` | Add six `<script src="/assets/xterm/addon-*.js">` lines (all loaded, instantiation gated by config) |
| `web/assets/terminal.js` | Fetch plugin config; load addons per config; lifecycle ordering |
| `internal/webserver/vendor_drift_test.go` | Regex extended to match all `@xterm/*` packages |
| `internal/webserver/no_cdn_regression_test.go` | No change expected (vendor/xterm/ already excluded) |
| `frontend/src/themes.ts` | No change |

---

## 8. Build Order — Suggested Phase Split (Question 7)

Dependencies dictate the order. Five phases, increasing complexity:

### Phase 92 — Plugin Settings Foundation (no addons yet)

**Why first:** all later phases plug into this. Cheap to validate.

- Extend `daemonSettings` with `Plugins *PluginSettings` and round-trip persistence
- Add Wails `GetPluginSettings` / `SetPluginSettings` bindings
- Add `PluginsSection.tsx` skeleton in Settings (six toggles, no per-plugin config yet)
- Wire `pluginConfig` from App.tsx → TerminalPanel as a prop (no consumption yet)
- Add `settings:plugins` event emission

**Verifies:** persistence round-trip; UI renders; toggles save and reload across daemon restart.

**Deferred:** actual addon loading.

### Phase 93 — WebGL & Unicode11 Migration to Plugin System

**Why second:** these are already loaded today in TerminalPanel. We're moving them under config control. Lowest-risk addon work because the code already exists.

- Refactor TerminalPanel addon loading into the reconcile pattern (`pluginReconcile.ts`)
- Make WebGL hot-swappable (currently it's load-once)
- Wire Unicode11 to be **conditional but applied-at-create-time only** (the new-session-only badge)
- Update `web/terminal.html` + `terminal.js` + `embed.go` + `VERSION` for unicode11 + webgl on web (web today has neither)
- Vendor `addon-unicode11.js` and `addon-webgl.js`; update `vendor_drift_test.go` regex
- Add `GET /api/plugin-config` endpoint with capability gate

**Verifies:** web-desktop parity for the two foundation plugins; reconcile pattern works on a live terminal; vendor-drift test still passes.

### Phase 94 — Search Addon + Search UI

**Why third:** purely additive UI, hot-swap friendly, no CSP risk.

- Add `@xterm/addon-search` to frontend + vendor
- Build `PluginConfigSearch.tsx` (default flags) and a search overlay component (`SearchOverlay.tsx` — Cmd-F shortcut, find-next/prev, regex/case/whole-word toggles)
- Wire keyboard handler in App.tsx (only when search plugin enabled)
- Web side: ship the addon vendored but **defer** the web search UI to a future phase

**Verifies:** end-to-end per-plugin config flow (Search has 4 sub-flags); hot toggle works.

### Phase 95 — Web-Links Addon + Click Handler

**Why fourth:** one new sub-config (`clickModifier`); hot-swap friendly; configures `BrowserOpenURL` on desktop vs `window.open` on web.

- Add `@xterm/addon-web-links` to frontend + vendor
- `PluginConfigWebLinks.tsx` with click-modifier dropdown
- Desktop click handler routes through Wails `BrowserOpenURL` for native browser launch
- Web click handler uses `window.open(url, '_blank', 'noopener,noreferrer')`
- Custom URL regex field hidden behind disclosure

**Verifies:** click-modifier works; desktop launches default browser; web opens new tab; URL regex override accepted and validated.

### Phase 96 — Image Addon + CSP Audit

**Why fifth:** the highest-CSP-risk plugin and the most novel. Isolating it makes phase verification clean.

- Spawn researcher first: read `addon-image.js` source for `URL.createObjectURL`, `data:` script blobs, dynamic Worker construction. Document findings in phase RESEARCH.md.
- Add `@xterm/addon-image` to frontend + vendor
- Update CSP if findings require (most likely: no change needed; possibly `img-src` needs `blob:`)
- Wire to TerminalPanel + web/terminal.js
- Test with sample sixel + iTerm IIP escape sequences (`agenthub` REPL paste test)
- Update `csp_integration_test.go` if CSP amended; update `browser_csp_e2e_test.go` to assert no violations during image render

**Verifies:** real image rendering; no CSP violations; clean disable falls back gracefully.

### Phase 97 — Serialize Addon + "Save Session" UX

**Why last:** purely additive, no dependencies on others, but the surrounding UX is the most app-shell-flavored (save dialog, file naming, mime type).

- Add `@xterm/addon-serialize` to frontend + vendor (web doesn't expose this — desktop only)
- Add "Save Terminal As…" item to TerminalPanel context menu or sidebar action
- Wails `SaveFileDialog` for path; write `term.serialize()` output (text, html, or both depending on serialize options)
- Decision: support text-only in v3.2; HTML output deferred (theme-aware, more complex)

**Verifies:** serialize captures full scrollback; file save works on macOS/Linux/Windows; large buffers don't OOM.

### Why this order works

1. **Foundation first** (92): every later phase needs `pluginConfig` plumbing.
2. **Migrate-don't-add second** (93): lowest-risk addon work — the code already runs.
3. **Cheapest new addon third** (94): search is hot-swap friendly and self-contained UI; proves the per-plugin config UI flow.
4. **One config knob fourth** (95): web-links exercises the click-handler abstraction without CSP risk.
5. **CSP-risky fifth** (96): image is isolated so phase verification can do dedicated CSP UAT without bundling other risk.
6. **App-shell-heavy last** (97): serialize needs the most surrounding UX (save dialog, file mime type), and benefits from being implemented after the toggle/config pattern is proven by all five prior phases.

**Phases 92 and 93 are blocking dependencies for everything else.** 94/95 can run in parallel if helpful. 96 is independent. 97 is independent.

---

## 9. Patterns to Follow (from existing codebase)

| Pattern | Source | Apply to v3.2 |
|---|---|---|
| `Get<X>` / `Set<X>` Wails binding pairs | `app.go` startMinimized/autoClose | `GetPluginSettings` / `SetPluginSettings` |
| `daemonSettings` extended without breaking JSON compat (`omitempty`, pointer for tri-state) | engine.go | `Plugins *PluginSettings` (nil = use defaults) |
| Three-state Save button | SettingsTab.tsx | Reuse for plugin section save (likely auto-save with debounce, no button) |
| `EventsOn` for cross-component sync | App.tsx (session:exit, app:quit-requested) | `settings:plugins` |
| Source-inspection vitest tests with `fs.readFileSync` | existing pattern (Tech Debt note) | New PluginsSection tests follow same shape |
| `//go:embed` + `fs.Sub` mounted at `/assets/xterm/` | webserver/server.go + web/embed.go | Same path for new addons |
| Vendor-drift regex test | `vendor_drift_test.go` | Extend regex; add new packages |

## 10. Anti-Patterns to Avoid

| Anti-pattern | Why bad | Do instead |
|---|---|---|
| Per-tab plugin config in v3.2 | Doubles UI complexity; existing "global theme" precedent already chose against this | Global config; revisit if user demand surfaces |
| localStorage for plugin config | Splits desktop/web state; loses on browser cache clear | `daemonSettings` + Wails event |
| Lazy-load addon JS via dynamic script injection | Conflicts with `script-src 'self'` in spirit; brittle | Eager `<script>` tags in HTML; runtime instantiation gated on config |
| Hot-swapping unicode11 mid-stream | Buffer already laid out; partial corruption | Apply at terminal-create time only; show "applies to new sessions" badge |
| Forgetting to update `vendor_drift_test.go` regex | CI passes, vendoring drifts silently | Phase checklist item — update test in same commit as VERSION bump |
| Putting plugin config in `app.go` SessionInfo | Pollutes session-level types with global config | Separate Wails binding pair, separate state |

---

## 11. Confidence & Open Questions

**HIGH confidence on:**
- Existing TerminalPanel addon loading shape (read directly from source)
- Existing daemonSettings + Wails binding pattern (verified)
- Existing CSP, embed.FS, vendor-drift contract (read source)
- Hot-swap matrix (lifecycle reasoning + xterm.js architecture knowledge)

**MEDIUM confidence on:**
- addon-image CSP requirements — upstream docs vague; resolved by Phase 96 research
- addon-search "incremental" config option — addon API surface confirmed but exact config field names need a Context7 lookup at phase time

**LOW confidence on:**
- Web search overlay UX — deferred outside v3.2 scope explicitly to avoid undertested ship

**Open questions for milestone roadmap:**
1. Should the desktop bundle the same addon JS as web (vendored), or accept that frontend uses its own pnpm-managed copy and only `vendor_drift_test.go` keeps them aligned? Current answer: keep current pattern (frontend uses pnpm; web/vendor mirrors; drift test enforces match).
2. Default-enabled set: ship all six on by default, or only the four already-loaded ones? Recommendation: all six on; image is the only potential surprise but it's well-established.
3. Where does the "applies to new sessions only" indicator render? Inline next to the toggle, or as a section-level note? Recommendation: inline tag per affected toggle (matches "READ ONLY" badge pattern from web terminal).

---

## 12. Sources

- `frontend/src/components/TerminalPanel.tsx` (verified addon loading)
- `frontend/package.json` (verified installed addon versions)
- `internal/daemon/engine.go` (verified daemonSettings shape)
- `internal/webserver/csp_mw.go` (verified CSP policy)
- `internal/webserver/no_cdn_regression_test.go` (verified anti-regression contract)
- `internal/webserver/vendor_drift_test.go` (verified vendor-drift contract)
- `web/embed.go`, `web/terminal.html`, `web/assets/terminal.js` (verified web-served terminal)
- `app.go` (verified Wails binding pattern)
- `.planning/PROJECT.md`, `.planning/MILESTONES.md` (verified v3.1 vendoring + CSP context)
- [@xterm/addon-webgl README](https://github.com/xtermjs/xterm.js/blob/master/addons/addon-webgl/README.md) — load-after-open pattern
- [@xterm/addon-image upstream](https://github.com/xtermjs/xterm.js/tree/master/addons/addon-image) — workers removed, canvas-based
- [xterm.js Using addons guide](https://xtermjs.org/docs/guides/using-addons/) — general addon lifecycle
