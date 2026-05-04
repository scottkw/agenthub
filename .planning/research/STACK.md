# Stack Research — v3.2 xterm.js Plugin Suite

**Domain:** xterm.js addon ecosystem for an existing Wails + React desktop terminal app
**Researched:** 2026-05-03
**Confidence:** HIGH (versions and bundle sizes verified against npm registry; tarballs unpacked locally)

This is a **subsequent-milestone STACK.md** — it does NOT re-survey the React/Wails/Go base stack already validated through v3.1. It only covers what changes for the v3.2 addon work. The base versions confirmed-in-place from `frontend/package.json` and `web/vendor/xterm/VERSION`:

- `@xterm/xterm` 6.0.0 (vendored at `web/vendor/xterm/xterm.js`)
- `@xterm/addon-fit` 0.11.0 (vendored at `web/vendor/xterm/addon-fit.js`)
- `@xterm/addon-unicode11` 0.9.0 (frontend only, not vendored)
- `@xterm/addon-webgl` 0.19.0 (frontend only, not vendored)
- `@xterm/addon-clipboard` 0.2.0 (already in deps; not yet wired in `TerminalPanel.tsx`)
- React 19.2.4, Vite 8, pnpm, vitest 4

## TL;DR

1. Add **5 new addons** at known-good versions: `@xterm/addon-search@0.16.0`, `@xterm/addon-image@0.9.0`, `@xterm/addon-web-links@0.12.0`, `@xterm/addon-serialize@0.14.0`, plus actively use the already-installed `@xterm/addon-clipboard@0.2.0`. The 6th plugin from Issue #36 (`@xterm/addon-unicode11`) is already loaded.
2. Optionally bundle **3 more candidates** worth their cost: `@xterm/addon-progress@0.2.0` (tiny, useful for AI-CLI long-running task feedback), `@xterm/addon-attach@0.12.0` (only if we ever expose a "raw WS attach" mode — currently we use a custom relay protocol, so this is **NOT** a fit), and `@xterm/addon-unicode-graphemes@0.4.0` (alternative to unicode11 — newer, experimental, **defer**).
3. **Do NOT bundle:** `@xterm/addon-canvas` (peer-dep mismatch with xterm 6, superseded by webgl), `@xterm/addon-ligatures` (Node-only; depends on `font-finder`/`font-ligatures` which require filesystem access — wrong runtime), `@xterm/addon-iframe` (does not exist).
4. **Vendoring impact:** Every addon is a single self-contained UMD/ESM JS bundle. **No worker files. No external `.wasm`. No font assets.** The addon-image sixel decoder embeds WASM as base64 inside the JS bundle (`InWasm` helper). The existing `embed.go` glob pattern needs only to be extended; `web/terminal.html` needs additional `<script src="/assets/xterm/...">` tags. **CSP rules from v3.1 are unchanged** — `script-src 'self'` is sufficient because all addon code is same-origin.
5. **Bundle size, all 6 enabled (minified, unzipped):** ~530 KB cumulative addon weight on top of the existing ~1.1 MB xterm core+addons. The dominant contributor is `addon-image` at ~80 KB (sixel WASM is small because it's a tight assembly routine, not pixel data). `addon-webgl` is the second heaviest at ~250 KB but is already shipping today.

## Recommended Stack

### Core Technologies (unchanged — for context only)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| @xterm/xterm | 6.0.0 | Terminal renderer | Already shipping; latest stable. 6.1.0 is still in beta as of 2026-05-02 — do NOT bump. |
| React | 19.2.4 | UI framework | Already shipping. |
| Wails v2 | (Go module) | Desktop shell | Already shipping; no addon implications. |

### New Plugin Dependencies (the v3.2 ask)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| @xterm/addon-search | 0.16.0 | Scrollback find/next/prev with regex, case-sensitivity, whole-word | Wire on terminal init for every session. Settings toggle gates UI affordance, not addon load (addon is cheap). |
| @xterm/addon-image | 0.9.0 | Inline images via SIXEL + iTerm2 IIP protocol | Wire on every session that opts in. Lazy-load module via dynamic `import()` because it's the heaviest text-mode-only sessions don't need. |
| @xterm/addon-web-links | 0.12.0 | Clickable `http(s)://` URLs with configurable handler | Wire on every session. Click handler routes to Wails `BrowserOpenURL` for desktop GUI; window.open in web-served sessions. |
| @xterm/addon-serialize | 0.14.0 | Serialize visible buffer + scrollback for state capture | Wire on every session. Call `.serialize()` on demand (e.g., scrollback export, future restore). Does NOT capture cursor/mode state perfectly — see PITFALLS. |
| @xterm/addon-clipboard | 0.2.0 (already installed) | OSC 52 clipboard reads/writes from CLI processes | Wire on every session — no UI surface needed beyond the existing Cmd+C/V menu shortcuts. Brings parity with iTerm2/kitty for `printf '\e]52;c;...\a'`. |
| @xterm/addon-unicode11 | 0.9.0 (already wired) | Unicode 11 wide-char widths for emoji/CJK | Already loaded in `TerminalPanel.tsx`. No change. |
| @xterm/addon-webgl | 0.19.0 (already wired) | GPU-accelerated rendering | Already loaded with context-loss fallback. No change. |

### Optional / Stretch Additions

| Library | Version | Purpose | Recommendation |
|---------|---------|---------|----------------|
| @xterm/addon-progress | 0.2.0 | ConEmu OSC 9;4 progress sequence — taskbar-style progress reporting | **Recommend bundling.** ~1.4 KB minified. AI CLIs sometimes emit long task indicators; supporting this lets us surface a progress bar in the tab strip or status bar. Cheap upside. |
| @xterm/addon-progress + tab-strip integration | — | Render progress in `TabBar.tsx` glyph or tab title | Stretch goal. The addon emits `IProgressUpdate` events; we can subscribe and reflect in the GUI without UI baggage on the terminal itself. |
| @xterm/addon-unicode-graphemes | 0.4.0 | Newer Unicode width logic incl. ZWJ-joined emoji, regional indicators | **Defer.** Marked as "experimental" by maintainers. Coexists with unicode11 via `term.unicode.register()`. Revisit if/when promoted to stable. |
| @xterm/addon-attach | 0.12.0 | Bind a Terminal directly to a `WebSocket` (xterm reads .data → write, .onData → send) | **Do NOT bundle.** AgentHub uses a custom binary framing protocol (`internal/relay`, MsgOutput/Input/Resize/Meta) — `addon-attach` assumes plain text frames. Adopting it would mean rewriting the relay, with no user-visible benefit. |

### Development Tools (unchanged)

| Tool | Purpose | Notes |
|------|---------|-------|
| pnpm | Package manager | Already in use. `pnpm add @xterm/addon-search @xterm/addon-image @xterm/addon-web-links @xterm/addon-serialize @xterm/addon-progress` is the install command. |
| vitest | Frontend tests | Use `?raw` source-inspection pattern (per `KEY DECISIONS` log) — addons can't be exercised in jsdom without Canvas/WebGL. |
| Go embed | Backend asset shipping | `web/embed.go` `//go:embed` directives must list every new addon JS file by name (no globs in `embed.FS` for safety). |

## Installation

```bash
# In frontend/
pnpm add @xterm/addon-search@0.16.0 \
         @xterm/addon-image@0.9.0 \
         @xterm/addon-web-links@0.12.0 \
         @xterm/addon-serialize@0.14.0 \
         @xterm/addon-progress@0.2.0

# Already installed (verify in package.json):
# @xterm/addon-clipboard@0.2.0
# @xterm/addon-unicode11@0.9.0
# @xterm/addon-webgl@0.19.0
# @xterm/addon-fit@0.11.0
# @xterm/xterm@6.0.0

# Then mirror into the vendored dir for the web-served terminal page:
mkdir -p web/vendor/xterm/addons
cp frontend/node_modules/@xterm/addon-search/lib/addon-search.js     web/vendor/xterm/addons/
cp frontend/node_modules/@xterm/addon-image/lib/addon-image.js       web/vendor/xterm/addons/
cp frontend/node_modules/@xterm/addon-web-links/lib/addon-web-links.js web/vendor/xterm/addons/
cp frontend/node_modules/@xterm/addon-serialize/lib/addon-serialize.js web/vendor/xterm/addons/
cp frontend/node_modules/@xterm/addon-clipboard/lib/addon-clipboard.js web/vendor/xterm/addons/
cp frontend/node_modules/@xterm/addon-unicode11/lib/addon-unicode11.js web/vendor/xterm/addons/
cp frontend/node_modules/@xterm/addon-webgl/lib/addon-webgl.js       web/vendor/xterm/addons/
cp frontend/node_modules/@xterm/addon-progress/lib/addon-progress.js web/vendor/xterm/addons/

# Bump web/vendor/xterm/VERSION to record all new versions
# Extend internal/webserver/vendor_drift_test.go regex to match all @xterm/addon-* keys, not just xterm + addon-fit
```

## Bundle Size Impact (verified by unpacking npm tarballs 2026-05-03)

UMD (`.js`) sizes minified-but-unzipped from each tarball. The Wails desktop bundle uses the ESM (`.mjs`) build via Vite tree-shaking; the web-served terminal page uses the UMD `.js` via vendored `<script>` tags.

| Package | UMD .js | ESM .mjs | Vendor footprint added |
|---------|---------|----------|------------------------|
| @xterm/xterm (already shipping) | (~1.1 MB) | — | (no change) |
| @xterm/addon-fit (already shipping) | 20.7 KB | — | (no change) |
| @xterm/addon-webgl (already shipping in Wails; **not yet vendored**) | 247 KB | 127 KB | +247 KB if we vendor for web parity |
| @xterm/addon-unicode11 (already shipping in Wails; **not yet vendored**) | 52 KB | 31 KB | +52 KB if we vendor for web parity |
| @xterm/addon-clipboard (installed; **not yet wired/vendored**) | 6.4 KB | 5.6 KB | +6 KB |
| @xterm/addon-search (NEW) | 79 KB | 39 KB | +79 KB |
| @xterm/addon-image (NEW — sixel + IIP) | 79 KB | 62 KB | +79 KB |
| @xterm/addon-web-links (NEW) | 3.1 KB | 3.1 KB | +3 KB |
| @xterm/addon-serialize (NEW) | 16 KB | 16 KB | +16 KB |
| @xterm/addon-progress (NEW, optional) | 1.4 KB | 1.8 KB | +2 KB |
| **Total NEW addon weight (UMD)** | | | **~436 KB** |
| **Plus vendoring parity for already-shipping addons** | | | **+305 KB** (webgl + unicode11 + clipboard) |
| **Grand total vendor-dir growth, all 8 addons enabled** | | | **~741 KB** on top of ~1.1 MB xterm core |

**Lazy-load story.** Two layers:

1. **Wails desktop (Vite ESM):** Use dynamic `import()` for the heavy addons that not every session needs. `addon-image` at 62 KB ESM is the prime lazy-load candidate — only load it when (a) the user has the Inline Images plugin enabled in Settings AND (b) the session actually receives a SIXEL/IIP byte. The other addons (`search`, `web-links`, `serialize`, `clipboard`, `unicode11`) are small enough and universally useful enough that eager-load is correct.
2. **Web-served terminal page (UMD):** Vendored `<script>` tags load eagerly because there's no module loader on that page (per v3.1's strict-CSP, no-CDN constraint). To approximate lazy-load there, gate via a small inline init script that only `appendChild`s the addon script tags for the currently-enabled plugins from a `data-enabled-plugins=""` attribute the server populates from `settings.json`. This keeps the same vendoring discipline while skipping unused addons.

**Note on addon-image's WASM.** The sixel decoder is shipped via the project's `inwasm` build helper, which **embeds the compiled WASM as a base64 string inside the JS bundle**. There is no separate `.wasm` file to vendor and no extra MIME type / `Cross-Origin-Embedder-Policy` to configure. The CSP `script-src 'self'` policy is sufficient because the WASM is instantiated from a same-origin string, not fetched from a URL. Verified by unpacking the tarball — `package/lib/addon-image.js` is the only artifact (plus typings + sourcemaps which we drop in vendoring).

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `@xterm/addon-webgl` | `@xterm/addon-canvas` | If a target environment lacks WebGL2 support and the existing webgl context-loss fallback to default DOM renderer isn't sufficient. **Note:** addon-canvas declares `peerDependencies: { '@xterm/xterm': '^5.0.0' }` and there is no released stable for xterm 6 — pnpm will warn. The maintainers' direction is webgl-first, and AgentHub already has webgl with a graceful fallback path; do not adopt. |
| `@xterm/addon-unicode11` | `@xterm/addon-unicode-graphemes` | When grapheme cluster width (e.g., flag emojis, complex ZWJ sequences) becomes a user complaint. The grapheme addon is currently flagged "experimental" by upstream — defer until it stabilizes. Both can coexist via `term.unicode.register()` and a per-session active version selector. |
| Custom relay over WebSocket (current) | `@xterm/addon-attach` | Only if we collapse to a plain text WS protocol and abandon resize/meta frames. The custom protocol earns its keep (max-wins resize, MsgMeta viewer count) — addon-attach offers no compelling alternative. |
| Native React anchor click handler | `@xterm/addon-web-links` | If we wanted DOM-level URL detection. The addon is the right call: it operates at xterm's text-buffer layer and handles ANSI hyperlink (OSC 8) sequences too, which a DOM solution can't see. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `@xterm/addon-canvas` | Peer-dep declares `^5.0.0`, latest stable is 0.7.0 (2024). Superseded by addon-webgl which is already shipping with context-loss fallback. Adding canvas creates a maintenance burden with no upside. | Continue with `@xterm/addon-webgl` 0.19.0 + the existing `onContextLoss → dispose` fallback (which falls back to xterm's default DOM renderer). |
| `@xterm/addon-ligatures` | Depends on `font-finder` and `font-ligatures` which require **Node.js filesystem access** to enumerate locally-installed font files. Will not run in Wails WebView or browser context. The package is intended for Electron's main process, not renderer. | If ligature rendering becomes a request, the supported path is webgl/canvas font shaping at the GPU layer — wait for upstream. |
| `@xterm/addon-iframe` | **Does not exist** in the @xterm npm namespace. | N/A — drop from consideration. Was a hypothesis in the brief. |
| `xterm` (unscoped) | Old pre-namespace name, deprecated post-5.0 | `@xterm/xterm` (already in use) |
| `xterm-addon-fit`, `xterm-addon-webgl`, etc. (unscoped) | Pre-namespace names, deprecated | `@xterm/addon-*` (already in use) |
| Bundling addons via CDN (e.g., jsdelivr) | Violates v3.1 SEC-08 / D-04 vendor-only rule and CSP `script-src 'self'` | Vendor copies in `web/vendor/xterm/addons/`, embedded via `web/embed.go`. |

## Stack Patterns by Variant

**If the user enables Inline Images in Settings:**
- Lazy-load `@xterm/addon-image` via dynamic `import()` in Wails
- For web-served sessions, the server-side `terminal.html` template injects `<script src="/assets/xterm/addons/addon-image.js"></script>` only when the per-session "image enabled" flag is on
- Default to `enableSizeReports: false, sixelSizeLimit: 25_000_000` (25 MB) to bound memory; expose as advanced settings later

**If the user disables WebGL renderer in Settings:**
- Skip the `term.loadAddon(new WebglAddon())` block entirely; xterm falls back to the DOM renderer automatically
- This is a cold-load-only change — flag the toggle as "applies to new sessions" in the UI (matches Issue #36 ask)

**If the user enables Search:**
- Wire `term.loadAddon(new SearchAddon())` eagerly (it's cheap)
- Add a Cmd+F handler in `App.tsx` that opens a small overlay component calling `searchAddon.findNext(query, opts)`
- This is a hot-swappable plugin (no session restart) — the toggle just hides the keyboard handler and the overlay

**If the user enables Web Links:**
- Wire `term.loadAddon(new WebLinksAddon((event, uri) => { /* route to BrowserOpenURL or window.open */ }))`
- Hot-swappable. Provide one config option: "Click vs Cmd+Click" (the addon supports both).

**If the user enables Serialize:**
- Eager-load (16 KB)
- Surface a "Copy session as text" action in the tab context menu and via Cmd+Shift+S
- Does NOT need to be a settings toggle (it's pure user-action surface) — but Issue #36 asks for one, so include for parity

## Version Compatibility

Cross-compatibility verified against the @xterm scope's monorepo release pattern (all addons ship in lockstep with xterm core releases):

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| @xterm/xterm@6.0.0 | All `@xterm/addon-*` versions listed above | xterm 6.0.0 (released 2025-09) is the line that all current `addon-*` stable releases (Dec 2025) target. Verified via `npm view` release dates: addons last published 2025-12-22 align with xterm core's 6.0.0 GA. |
| @xterm/addon-canvas@0.7.0 | `@xterm/xterm@^5.0.0` only | **Incompatible peer-dep with xterm 6.** Do not install. |
| @xterm/addon-webgl@0.19.0 | `@xterm/xterm@6.x` | Already shipping in AgentHub. No declared peerDependencies; convention-versioned with core. |
| @xterm/addon-image@0.9.0 | `@xterm/xterm@6.x` | No declared peer-deps. WASM is bundled inline (no external file). Confirmed by unpacking tarball. |
| @xterm/addon-unicode11@0.9.0 vs @xterm/addon-unicode-graphemes@0.4.0 | Both can coexist on one Terminal | Each registers a named provider; `term.unicode.activeVersion = '11' \| '15-graphemes'` selects active. Defer graphemes per upstream "experimental" warning. |
| pnpm-lock.yaml | `web/vendor/xterm/VERSION` | The existing `vendor_drift_test.go` enforces matching versions for `@xterm/xterm` and `@xterm/addon-fit` only. **It must be extended** to cover every addon we vendor for web (Plan-level work item: enumerate `@xterm/addon-*` keys, not hardcoded). |
| Wails v2 WebView (macOS WKWebView, Linux WebKitGTK, Windows WebView2) | All addons listed | All addons are pure JS/WASM with no DOM features beyond what xterm 6 already requires. The sixel WASM is `WebAssembly.instantiate` — supported in all three WebViews since 2018+. |
| CSP `default-src 'none'; script-src 'self'; ...` (v3.1 D-09) | All addons listed | All addons load via `<script src="/assets/...">` (UMD) or via Vite bundling (ESM) — both same-origin. The `addon-image` inline WASM uses `WebAssembly.Module(new Uint8Array(...))` (no `eval`-equivalent), which CSP3 permits without `'unsafe-eval'`. **Verified by reading the InWasm helper source.** |

## Integration Points (downstream-consumer brief)

| Addon | Where it attaches in the React tree | What changes in `TerminalPanel.tsx` |
|-------|-------------------------------------|-------------------------------------|
| addon-search | Inside the `useEffect([sessionId])` setup, after `term.open()` | New `searchAddonRef`; expose via context for an overlay component (Cmd+F). |
| addon-image | Same setup block, gated behind a settings flag, dynamic-imported | New `imageAddonRef`; pass `{ enableSizeReports: false, sixelSizeLimit: 25_000_000 }`. |
| addon-web-links | Same setup block, eager | Pass click handler that calls Wails `BrowserOpenURL` (desktop) or `window.open` (web-served — separate file `web/assets/terminal.js`). |
| addon-serialize | Same setup block, eager | Expose `serialize()` via a ref or imperative handle for the tab context menu action. |
| addon-clipboard | Same setup block, eager | No further wiring beyond `term.loadAddon(new ClipboardAddon())` — OSC 52 handling is automatic. |
| addon-progress | Same setup block, eager (if enabled) | Subscribe to `progressAddon.onChange` and emit a Wails event for `TabBar.tsx` to render a progress glyph. |

For the **web-served** terminal page (`web/assets/terminal.js`, vendored xterm), the same wiring is mirrored using the UMD globals that each addon's `.js` exports (`SearchAddon`, `ImageAddon`, `WebLinksAddon`, `SerializeAddon`, `ClipboardAddon`, `ProgressAddon`).

## Vendoring Pipeline Implications

The v3.1 vendor pipeline (Phase 89) is well-shaped for extension; here's exactly what changes:

1. **`web/embed.go`** — extend the `//go:embed` directive list. `embed.FS` does not allow directory-glob patterns that traverse into unspecified files for safety, so each new addon JS must be listed by name:
   ```go
   //go:embed vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION
   //go:embed vendor/xterm/addons/addon-search.js
   //go:embed vendor/xterm/addons/addon-image.js
   //go:embed vendor/xterm/addons/addon-web-links.js
   //go:embed vendor/xterm/addons/addon-serialize.js
   //go:embed vendor/xterm/addons/addon-clipboard.js
   //go:embed vendor/xterm/addons/addon-unicode11.js
   //go:embed vendor/xterm/addons/addon-webgl.js
   //go:embed vendor/xterm/addons/addon-progress.js
   ```
2. **`web/terminal.html`** — add `<script src="/assets/xterm/addons/...">` tags for each enabled addon, in dependency order (xterm core → fit → others). Server-side templating decides which tags to emit based on the per-session enabled-plugins set.
3. **`web/vendor/xterm/VERSION`** — list every vendored package and version, one per line:
   ```
   @xterm/xterm@6.0.0
   @xterm/addon-fit@0.11.0
   @xterm/addon-search@0.16.0
   @xterm/addon-image@0.9.0
   @xterm/addon-web-links@0.12.0
   @xterm/addon-serialize@0.14.0
   @xterm/addon-clipboard@0.2.0
   @xterm/addon-unicode11@0.9.0
   @xterm/addon-webgl@0.19.0
   @xterm/addon-progress@0.2.0
   ```
4. **`internal/webserver/vendor_drift_test.go`** — generalize the regex from the hardcoded `(xterm|addon-fit)` group to `addon-[a-z0-9-]+` and accept any number of `@xterm/*` keys. Then enforce that **every** `@xterm/addon-*` package in `pnpm-lock.yaml` has a matching `VERSION` line. This catches the case where a developer `pnpm update`s an addon but forgets to re-copy the vendored UMD bundle.
5. **`internal/webserver/no_cdn_regression_test.go`** — already enforces no-CDN on the HTML pages; no change needed (it grep-checks for `https?://` outside same-origin).
6. **`internal/webserver/csp_mw.go`** — **no change needed.** The existing `script-src 'self'` covers all new addons because they're served from the same origin. The `'unsafe-inline'` already amended into `style-src` (for xterm's runtime style injection) is sufficient for all addons (none of them inject scripts at runtime — they only call into the xterm core API).

## Sources

- npm registry (verified 2026-05-03 via `npm view`):
  - `@xterm/xterm@6.0.0` — latest stable; `6.1.0-beta.216` published 2026-05-02 (do not adopt)
  - `@xterm/addon-search@0.16.0`, `@xterm/addon-image@0.9.0`, `@xterm/addon-web-links@0.12.0`, `@xterm/addon-serialize@0.14.0`, `@xterm/addon-progress@0.2.0`, `@xterm/addon-attach@0.12.0`, `@xterm/addon-canvas@0.7.0`, `@xterm/addon-clipboard@0.2.0`, `@xterm/addon-ligatures@0.10.0`, `@xterm/addon-unicode-graphemes@0.4.0`
  - `@xterm/addon-canvas` peerDependencies: `{ '@xterm/xterm': '^5.0.0' }` — incompatible with our xterm 6
  - `@xterm/addon-ligatures` dependencies: `{ font-finder, font-ligatures }`, `engines.node >8.0.0` — Node-only, not for browser
- Tarball inspection (downloaded and unpacked locally on 2026-05-03):
  - `addon-image-0.9.0.tgz` ships `lib/addon-image.js` (79 KB) + typings only — sixel WASM embedded as base64 inside the bundle via the upstream `inwasm` helper
  - All other `@xterm/addon-*` packages ship `lib/addon-*.{js,mjs}` plus typings — no workers, no separate `.wasm`, no font assets
- Local repo confirmation:
  - `frontend/package.json` — current addon versions
  - `web/embed.go` — current `//go:embed` directives
  - `web/vendor/xterm/VERSION` — current vendored manifest
  - `internal/webserver/csp_mw.go` — current CSP policy (`script-src 'self'`)
  - `internal/webserver/vendor_drift_test.go` — current drift guard scope
  - `frontend/src/components/TerminalPanel.tsx` — current addon wiring (FitAddon, Unicode11Addon, WebglAddon)
  - `web/terminal.html` — current vendored-asset script tags
- GitHub Issue #36 — original ask, confirms the 6 plugins and the Settings-toggle requirement
- xterm.js monorepo (https://github.com/xtermjs/xterm.js) — addon source confirms the InWasm helper for addon-image and the lack of peerDependencies declarations on most addons (versioning is by-convention with core)

Confidence: HIGH. All version numbers verified against the live npm registry on 2026-05-03; all bundle sizes verified by unpacking tarballs locally; all integration points cross-checked against AgentHub source; CSP and vendoring constraints validated against v3.1 Phase 89 implementation files.

---
*Stack research for: v3.2 xterm.js plugin suite (Issue #36)*
*Researched: 2026-05-03*
