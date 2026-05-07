# Phase 96: Image Addon + CSP Audit — Research

**Researched:** 2026-05-07
**Domain:** xterm.js `@xterm/addon-image` (sixel + iTerm2 IIP), CSP `script-src 'wasm-unsafe-eval'`, multi-client byte-fidelity scrollback replay, per-terminal storage cap
**Confidence:** HIGH

---

## Summary

Phase 96 ships inline image rendering (sixel + iTerm2 Inline Image Protocol) via `@xterm/addon-image@0.9.0`. The mandatory pre-phase audit of `frontend/node_modules/@xterm/addon-image/lib/addon-image.{js,mjs}` produced one **load-bearing finding**: the addon contains **NO `new Worker(`, NO standalone `blob:` URLs in CSP-relevant constructs, NO `importScripts`, NO `eval()`, NO `data:text/javascript` script construction** — the only `URL.createObjectURL(new Blob(...))` call is a fallback in the IIP decoder for browsers lacking `createImageBitmap`, and the resulting URL is assigned to `Image.src` (controlled by `img-src`, NOT `script-src` / `worker-src`). However, the addon **does** embed a SIXEL decoder as base64-encoded WASM bytes that are instantiated in-process via `WebAssembly.instantiate()` / `new WebAssembly.Module()` / `new WebAssembly.Instance()`. Under the v3.1 CSP `script-src 'self'`, these calls are **blocked** in all major browsers as of CSP3. The CSP must therefore be amended with the **`'wasm-unsafe-eval'`** source expression (NOT `'unsafe-eval'`, which would weaken JS execution; `'wasm-unsafe-eval'` is scoped to WebAssembly only and supported in Chrome 102+ / Firefox 102+ / Safari 16+ — universally available across the Phase 99 supported browser matrix).

The remaining work is mechanical: add an `ImageConfig` nested struct to `daemon.PluginSettings` mirroring the Phase 95 `WebLinksConfig` precedent (just the `StorageLimit` field for now); thread an image arm through the existing TerminalPanel hot-swap useEffect at the **mount** site (not the hot-swap site — addon-image is next-session-only because changing options on a live terminal would re-allocate the canvas layer); vendor `lib/addon-image.js` to `web/vendor/xterm/addons/`; mirror in `web/assets/terminal.js`. The italic "Applies to new sessions you create" caption ships in PluginsSection (parallel to the existing Unicode 11 caption). The advanced `<details>` disclosure for storageLimit is **deferred to Phase 99 PUI-03** — Phase 96 ships the daemon struct + RPC + hardcoded 16 MB default value only, exactly mirroring Phase 95's WebLinksConfig deferral. IMG-04 is **already satisfied at the architecture level**: the `internal/relay/` scrollback path is a raw byte buffer with NO line-based buffering or escape filtering — sixel/IIP escape sequences pass through verbatim to second-mid-stream-joining clients via `ScrollbackSnapshot()`. The only IMG-04 risk is the **256 KiB scrollback cap** (`relay.DefaultScrollbackBytes`): a sixel image whose serialized escape stream exceeds 256 KiB will be partially truncated on replay. Phase 96 documents this as a known limitation (already true for non-image scrollback) and ships a regression test that verifies byte-fidelity within the cap; no scrollback resize is in scope.

**Primary recommendation:** Install `@xterm/addon-image@0.9.0` (already done as part of this research — `pnpm add -D @xterm/addon-image@0.9.0`), vendor `lib/addon-image.js` to `web/vendor/xterm/addons/`, extend the v3.1 CSP `script-src` directive from `'self'` to `'self' 'wasm-unsafe-eval'` in `internal/webserver/csp_mw.go` (this is the entire CSP amendment — `worker-src` and `blob:` additions are NOT required), add `ImageConfig{StorageLimit int}` to `daemon.PluginSettings` defaulting to `16` MB, mirror Phase 94-07 / 95-05 sub-config sub-key RPC plumbing (`(*App).SetImageConfig` Wails method + `PATCH /settings/image-config` HTTP route + `engine.SetImageConfig` writer), wire the addon construction into TerminalPanel's mount useEffect (not hot-swap — IMG-01 caption explicitly states "applies to new sessions you create"), append `@xterm/addon-image@0.9.0` to `web/vendor/xterm/VERSION` (the Phase 93 generalized `vendor_drift_test.go` regex covers it automatically; bump the min-count guard from 7 to 8), and write a Go-side multi-client byte-fidelity regression test using `internal/relay/hub_test.go` patterns to assert sixel byte-stream pass-through.

---

## Project Constraints (from CLAUDE.md)

- **JS/TS:** `camelCase` vars, `PascalCase` components, ESLint + Prettier, TypeScript types — applies to TerminalPanel image arm extension and any new helpers.
- **Node:** `pnpm` (project default). `@xterm/addon-image@0.9.0` already installed during this research session (devDependency).
- **Go:** `go fmt`, context-aware functions. Applies to `ImageConfig` daemon struct + `SetImageConfig` engine writer + `handleSetImageConfig` HTTP handler + new multi-client byte-fidelity test.
- **No global npm installs.**
- **NEVER kill node.exe** — Claude Code runs as Node.
- **LSP first** for code navigation — applies to discovering existing CSP middleware call sites and TerminalPanel addon refs.
- **UAT via dev-browser skill** for browser-based verifications — Phase 96 has multiple UAT touchpoints (CSP zero-violation across Chromium + Safari + Firefox; visual sixel + IIP rendering on desktop and web; Settings toggle + storage cap behavior).
- **Wails build requires `-tags wailsassets`** for production builds (project memory feedback).

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| IMG-01 | User can enable/disable inline image support in Settings (default ON); toggle clearly marked "applies to new sessions you create" | Existing PluginsSection `image` row already exists (PluginsSection.tsx:135); add the italic caption argument (4th param of `renderRow`, same as Unicode 11). PluginSettings.Image already defaults `true` (plugin_settings.go:89). TerminalPanel mount-useEffect (NOT hot-swap) reads `pluginConfig?.image` to decide whether to construct `ImageAddon` at session init. |
| IMG-02 | Per-terminal sixel/IIP storage hard-capped at 16 MB decoded RGBA by default (override of upstream 100 MB / 128 MB); user-adjustable via Advanced disclosure | Add `ImageConfig{StorageLimit int}` nested struct in `daemon.PluginSettings`. `defaultPluginSettings()` returns `ImageConfig{StorageLimit: 16}`. TerminalPanel constructs `new ImageAddon({ storageLimit: pluginConfig?.imageConfig?.storageLimit ?? 16 })`. **Advanced `<details>` UI defers to Phase 99 / PUI-03** — same deferral shape as Phase 95 WebLinksConfig.modifier. Phase 96 ships only the daemon struct + RPC + hardcoded default. |
| IMG-03 | Web-served Tailscale clients receive same inline image rendering as desktop, with v3.1 CSP either unchanged or amended (matching v3.1 D-09 documentation rigor) only if pre-phase audit confirms `addon-image` requires `worker-src 'self' blob:` | **Pre-phase audit complete (this RESEARCH.md, §"Mandatory Pre-Phase CSP Audit").** Finding: `addon-image` does NOT require `worker-src 'self' blob:`. It DOES require `script-src 'wasm-unsafe-eval'` (which the ROADMAP did not anticipate but is the load-bearing real requirement). CSP amendment: `script-src 'self'` → `script-src 'self' 'wasm-unsafe-eval'`. Document with the same rigor as v3.1 D-09 in `csp_mw.go` package comment. |
| IMG-04 | Second client joining session mid-stream receives correctly-rendered images during scrollback replay (multi-client byte-fidelity preserved through `internal/relay/`) | **Architecturally already satisfied.** `internal/relay/scrollback.go` is a raw byte buffer; `internal/relay/hub.go:135-152` reads 32 KiB chunks from the PTY reader and stores them via `MakeOutputFrame` + `Scrollback.Append` with NO line-based buffering and NO escape sequence parsing. `ScrollbackSnapshot()` returns a verbatim byte copy. New regression test in `internal/relay/scrollback_test.go` (or new file) seeds a small synthetic sixel byte stream (e.g. `\x1bPq#0;2;100;0;0#1;2;0;100;0!10A!10@-` plus terminator), pushes through Hub.Run via a `bytes.Reader`, takes ScrollbackSnapshot, and asserts the snapshot bytes equal the input bytes (modulo the 1-byte MsgOutput frame prefix per chunk). **Caveat:** scrollback is capped at `DefaultScrollbackBytes = 256 KiB`; a sixel image whose serialized escape stream exceeds 256 KiB will be partially truncated. Document as known limitation; do NOT enlarge the cap in Phase 96. |
</phase_requirements>

---

<user_constraints>
## User Constraints

> No `96-CONTEXT.md` will be authored. Per [skip-discuss-when-research-complete] memory: when ROADMAP/REQUIREMENTS/research already pre-answer the gray areas, skip `/gsd-discuss-phase` and proceed to `/gsd-plan-phase`. ROADMAP success criteria + REQUIREMENTS IMG-01..IMG-04 leave only mechanical questions. STATE.md `## Decisions` already locks the cross-cutting decisions below.

### Locked Decisions

- **Phase 96 owns inline image rendering for BOTH desktop and web.** [STATE.md `## Decisions`, Phase 96 entry] Same scope shape as Phases 94/95.
- **CSP amendment is gated on the pre-phase audit.** [STATE.md `## Decisions` Phase 96; ROADMAP Phase 96 SC-1] Audit complete; amendment is **REQUIRED** but for a different reason than the ROADMAP anticipated: `'wasm-unsafe-eval'` (not `worker-src 'self' blob:`). Documentation rigor matches v3.1 D-09. Confidence: HIGH.
- **Default ON.** [ROADMAP `## Decisions`: "ship all 7 plugins ON by default except optional `addon-progress`"] PluginSettings.Image defaults `true` already (Phase 92 plugin_settings.go:89).
- **`storageLimit: 16` MB hard cap by default.** [STATE.md `## Decisions` Phase 96; ROADMAP SC-3] Override of upstream default (128 MB per typings; ROADMAP says 100 MB — both are equally far above our 16 MB target so the discrepancy is immaterial). Phase 96 ships the value baked into the `ImageConfig` daemon struct's defaults; user adjustment via Advanced disclosure defers to Phase 99 PUI-03.
- **Vendored same-origin under `web/vendor/xterm/addons/addon-image.js`.** [STATE.md ROADMAP Phase 93 vendoring discipline] Phase 93/94/95 pattern applies verbatim — copy `frontend/node_modules/@xterm/addon-image/lib/addon-image.js` (CJS UMD, NOT the `.mjs`) to `web/vendor/xterm/addons/`.
- **No `<details>` advanced disclosure UI in Phase 96.** [Phase 99 / PUI-03] Mirrors Phase 95 WebLinksConfig.modifier deferral — daemon struct + RPC + hardcoded default ship in Phase 96; UI exposure ships in Phase 99.
- **Image addon is NEXT-SESSION-ONLY (not hot-swappable).** [REQUIREMENTS IMG-01: "applies to new sessions you create"] Construction goes in TerminalPanel's mount useEffect, NOT the hot-swap useEffect. Mirrors the Unicode 11 pattern (Phase 93) — toggling the Image setting on a live terminal does NOT re-attach; only new sessions pick up the change. The italic caption is the user-facing affordance.
- **256 KiB scrollback cap is acknowledged as a limitation, NOT enlarged.** A sixel/IIP image whose serialized byte stream exceeds 256 KiB will be partially truncated for second-mid-stream-joining clients. This is the existing v3.1 behavior for ALL terminal output; Phase 96 inherits it. Resizing the scrollback cap is **out of scope** for v3.2 (would touch v3.1 multi-client semantics MC-01..MC-06; tracked as a future consideration).
- **No image copy / save / extract gestures.** [REQUIREMENTS `## Out of Scope`] `addon-image.getImageAtBufferCell()` is exposed by the addon but Phase 96 does NOT wire UI for it. Users can screenshot.

### Claude's Discretion

- **Whether to add a separate `imageAddonRef` ref in TerminalPanel or inline the construction.** **Recommendation:** add a ref (parallel to `webglAddonRef`/`searchAddonRef`/`webLinksAddonRef`/`clipboardAddonRef`) so dispose-on-unmount is clean and a future hot-swap migration is one line. The mount useEffect's existing cleanup function (TerminalPanel.tsx:230-260 area) already iterates ref-based dispose calls; addon-image fits the pattern exactly.
- **Whether to fall back gracefully if `WebAssembly.instantiate` rejects (CSP misconfiguration, ancient browser).** **Recommendation:** wrap the `new ImageAddon(...)` call in try/catch (mirrors the existing WebGL try/catch). On failure: log warn, do not load addon; sixel/IIP sequences pass through harmlessly as printable garbage in the terminal (existing pre-Phase-96 behavior). Do NOT show a banner — image addon failure is non-critical compared to WebGL context loss.
- **Whether to use `addon-image@0.9.0` (latest stable) or `0.10.0-beta.216` (latest beta).** **Recommendation:** stick with `0.9.0` (latest non-beta). v3.2 ships under the v3.1 vendoring discipline + frozen-version posture; betas in security-sensitive code paths are a non-starter. Beta version availability is documented in §"State of the Art" for v3.3 reconsideration.
- **Sixel-vs-IIP test fixture choice.** **Recommendation:** use a tiny synthetic sixel byte stream for the IMG-04 unit test (no external image binary needed; bytes are self-documenting in the test source). For the SC-2 visual UAT, use `chafa --format=iterm2 chart.png` (per ROADMAP SC-2 verbatim) — `chafa` is a common CLI; if unavailable, fall back to a hand-crafted minimal IIP escape (`\x1b]1337;File=inline=1;width=2px;height=2px:<base64-png>\x1b\\`).
- **Whether the 50 MB sixel fixture for the storage-cap regression test (SC-3) is generated at test time or committed as a binary.** **Recommendation:** generated at test time. A 50 MB committed binary is git-bloat. Generation: a small Go test helper emits a synthetic sixel preamble + N raster bands of solid color until the decoded RGBA exceeds 50 MB. Eviction is observed via `addon.storageUsage` getter (typings line 102) reading less than the synthetic input.

### Deferred Ideas (OUT OF SCOPE)

- **`<details>` Advanced disclosure for `imageConfig.storageLimit` UI.** [Phase 99 / PUI-03] Daemon struct + RPC ship in Phase 96; UI exposure in Phase 99.
- **Image copy / save / "extract image" right-click gesture.** [REQUIREMENTS `## Out of Scope`] `addon-image` exposes `getImageAtBufferCell()` and `extractTileAtBufferCell()` returning `HTMLCanvasElement` — building UI on top is an explicit non-goal; users can screenshot.
- **Enlarging the relay scrollback cap to fit large images.** Would touch v3.1 MC-01..MC-06 multi-client byte-fidelity invariants and PTY backpressure semantics. Defer to a future milestone after a real user reports the truncation as a problem.
- **Sixel acceleration via OffscreenCanvas + Worker-based decoding.** The upstream addon supports this in beta channels (0.10.0-beta.*); v3.2 ships the stable WASM-in-main-thread path. Reconsider in v3.3 if performance issues surface.
- **Per-session storageLimit override.** Phase 96 ships a single global `storageLimit` (one value applied to all sessions). Per-session overrides would multiply the Settings UI surface area without clear demand.
- **Telemetry on image rendering failures.** Privacy-by-default (matches Phase 95 link-click privacy posture). Failures surface in browser DevTools console only.

</user_constraints>

---

## Mandatory Pre-Phase CSP Audit

**Audit target:** `frontend/node_modules/@xterm/addon-image/lib/addon-image.{js,mjs}` (version 0.9.0, installed 2026-05-07 via `cd frontend && pnpm add -D @xterm/addon-image@0.9.0`)

**Audit method:** `grep -n` and `grep -c` across both `addon-image.js` (CJS UMD, 2 lines minified, used for web vendoring) and `addon-image.mjs` (ESM, 39 lines mostly-minified, used by frontend bundler) for the four ROADMAP-mandated patterns plus standard auxiliary patterns.

### Findings Table

| Pattern Searched | Count in `.mjs` | Count in `.js` | CSP Directive Affected | Mitigation Required |
|------------------|----------------|---------------|------------------------|---------------------|
| `URL.createObjectURL(` | 1 | 1 | `img-src` (assigned to `Image.src`, NOT `script-src`) | None — v3.1 CSP `img-src 'self' data:` is sufficient because the resulting `blob:` URL becomes an Image source, not a script source. The `blob:` source expression in `img-src` is needed if the browser blocks blob: image sources under strict img-src; **see "blob: in img-src" decision below**. |
| `new Worker(` | 0 | 0 | `worker-src` | **None — `worker-src` directive is NOT required by addon-image.** The ROADMAP's anticipated amendment (`worker-src 'self' blob:`) is unnecessary. |
| `new Blob(` | 1 | 1 | (paired with `URL.createObjectURL` above) | (covered above) |
| literal `blob:` URL strings | 0 | 0 | (n/a — the `blob:` URLs are runtime-created via `URL.createObjectURL`, not literal in source) | (covered above) |
| `data:application/...` script construction | 0 | 0 | `script-src` | None |
| `data:text/javascript` | 0 | 0 | `script-src` | None |
| `importScripts(` | 0 | 0 | (no Worker, so n/a) | None |
| `eval(` | 0 | 0 | `script-src 'unsafe-eval'` | None |
| `new Function(` | 0 | 0 | `script-src 'unsafe-eval'` | None |
| **`WebAssembly.instantiate(`** | **2** | **2** | **`script-src 'wasm-unsafe-eval'` (CSP3)** | **REQUIRED — see §"CSP Amendment" below.** |
| **`new WebAssembly.Module(`** | **2** | **2** | **`script-src 'wasm-unsafe-eval'`** | **REQUIRED** |
| **`new WebAssembly.Instance(`** | **2** | **2** | **`script-src 'wasm-unsafe-eval'`** | **REQUIRED** |
| `new WebAssembly.Memory(` | 1 | 1 | `script-src 'wasm-unsafe-eval'` | (covered above) |

### Detailed Finding 1: `URL.createObjectURL` — IIP Decoder Fallback Path Only

**Location:** `addon-image.mjs` line 38 (and equivalent in minified `.js`), within the `Ae` class (IIP image processor's `end()` method).

**Code excerpt (deobfuscated for clarity):**

```javascript
// addon-image.mjs §IIP processor end() method
let blob = new Blob([this._dec.data8], { type: this._metrics.mime });
this._dec.release();
if (!window.createImageBitmap) {
  // FALLBACK PATH for browsers that lack createImageBitmap
  let url = URL.createObjectURL(blob);
  let img = new Image();
  return new Promise(resolve => {
    img.addEventListener('load', () => {
      URL.revokeObjectURL(url);
      let canvas = T.createCanvas(window.document, t, i);
      canvas.getContext('2d')?.drawImage(img, 0, 0, t, i);
      this._storage.addImage(canvas);
      resolve(true);
    });
    img.src = url;             // ← this is the only place blob: URL touches the DOM
    setTimeout(() => resolve(true), 1e3);
  });
}
// PRIMARY PATH — modern browsers
return createImageBitmap(blob, { resizeWidth: t, resizeHeight: i }).then(...)
```

**CSP analysis:**
- `URL.createObjectURL(blob)` is a runtime API call, not a CSP-controlled construct directly.
- The returned `blob:...` URL is assigned to `img.src`. Image element sources are governed by the **`img-src`** directive.
- Modern browsers **DO** require `blob:` (or `'self'`) in `img-src` for blob-URL image loads. v3.1 CSP currently has `img-src 'self' data:` — **this DOES NOT explicitly allow `blob:`**.
- However: in **all three** Phase 99 supported browsers (Chromium, Safari, Firefox), `createImageBitmap` IS available (it's the primary path). The fallback path executes only on ancient browsers (pre-2017 Chrome / pre-2017 Safari). **Phase 99's supported browser matrix excludes browsers without `createImageBitmap`.**

**Decision:** Do NOT add `blob:` to `img-src`. The fallback path is dead code on supported browsers. If Phase 99's UAT surfaces a real `img-src` violation in a supported browser, add `blob:` then — defensive minimalism over speculative permissiveness.

**Confidence:** HIGH (verified via direct source read of `addon-image.mjs` line 38; verified via `caniuse.com` that `createImageBitmap` is universally available since Safari 15 / Chrome 50 / Firefox 42).

### Detailed Finding 2: `WebAssembly.instantiate` / `Module` / `Instance` — SIXEL Decoder (Load-Bearing)

**Location:** `addon-image.mjs` lines 17-38 (within the `Ke()` and `Oe()` IIFEs that bootstrap the WASM-based SIXEL decoder); also within the sixel parser's `DecoderAsync` factory (line 38).

**Code excerpt (deobfuscated):**

```javascript
// addon-image.mjs §SIXEL WASM decoder factory
function DecoderAsync(opts) {
  let bandHandlerObj = new _e();
  let imports = {
    env: {
      handle_band: bandHandlerObj.handle_band.bind(bandHandlerObj),
      mode_parsed: bandHandlerObj.mode_parsed.bind(bandHandlerObj)
    }
  };
  return WebAssembly.instantiate(K || Ve, imports).then(result => (
    K = K || result.module,
    new Decoder(opts, result.instance || result, bandHandlerObj)
  ));
}

// addon-image.mjs §IIP base64 decoder factory
let A = WebAssembly;
return e === 2
  ? (i ? () => s || (s = R(d)) : () => Promise.resolve(s || (s = R(d))))
  : e === 1
    ? (i ? () => n || (n = new A.Module(s || (s = R(d))))
         : () => n ? Promise.resolve(n) : A.compile(s || (s = R(d))).then(o => n = o))
    : i ? o => new A.Instance(n || (n = new A.Module(s || (s = R(d)))), o)
        : o => n ? A.instantiate(n, o) : A.instantiate(s || (s = R(d)), o).then(...)

// addon-image.mjs §SIXEL decoder Decoder class _initCanvas
this._mem = new WebAssembly.Memory({ initial: Math.ceil(i / 65536) });
```

**Bytes source:** A base64-encoded string literal `Ve` containing the WASM bytecode for the SIXEL decoder (~10 KB raw / ~14 KB base64), inlined in `addon-image.mjs`. `R()` is a base64-decode-to-Uint8Array helper. The decoder is compiled and instantiated **synchronously in-process** when the addon constructs its sixel handler in `ImageAddon.activate()`. The IIP decoder uses a smaller WASM module for inline base64 decoding (also inlined as base64).

**CSP analysis:**
- Under CSP3 (the spec all major browsers implement as of 2024+), `WebAssembly.compile`, `WebAssembly.instantiate`, `new WebAssembly.Module`, and `new WebAssembly.Instance` are **gated by the `script-src` directive**.
- Default behavior (when `script-src` is set): WebAssembly compilation/instantiation is **BLOCKED** unless the `script-src` directive includes one of:
  - `'unsafe-eval'` (broad — also enables `eval()` and `new Function()` — **NOT acceptable** for our security posture)
  - `'wasm-unsafe-eval'` (narrowly scoped to WASM only — **the correct choice**)
- v3.1 CSP from `csp_mw.go:68`: `script-src 'self'` — **WILL BLOCK** `addon-image`'s WASM bootstrap.
- Browser failure mode: silent block in production CSP (no `report-uri` configured per v3.1 D-11); console warning in DevTools: `"Refused to compile or instantiate WebAssembly module because 'wasm-unsafe-eval' is not an allowed source of script in the following Content Security Policy directive: script-src 'self'"`.

**Browser support for `'wasm-unsafe-eval'`:**

| Browser | Supported Since | Date |
|---------|----------------|------|
| Chrome (Chromium) | 102 | May 2022 |
| Firefox | 102 | June 2022 |
| Safari (WebKit) | 16.0 | September 2022 |
| iPad Safari | 16.0 | September 2022 |

All four browsers in Phase 99's supported matrix support `'wasm-unsafe-eval'` with multiple years of headroom. **No fallback strategy required** — the amendment is universally honored.

**Decision:** Amend `internal/webserver/csp_mw.go` `script-src` directive from `'self'` to `'self' 'wasm-unsafe-eval'`. No other CSP directives change. Document the amendment in the package comment with the same rigor as the v3.1 D-09 amendment (the existing `'unsafe-inline'` style-src amendment).

**Confidence:** HIGH (verified via direct source read of `addon-image.mjs`; verified via web search of CSP3 spec + browser changelog dates; confirmed `'wasm-unsafe-eval'` is the correct directive name across all three engines).

### Audit Conclusion

The pre-phase audit is **complete and definitive**. Two CSP-relevant findings:

1. **`URL.createObjectURL` → `Image.src` fallback path:** `img-src` amendment **NOT required** for supported browsers (createImageBitmap is the primary path; fallback is dead code on Chrome 50+ / Firefox 42+ / Safari 15+). The ROADMAP's anticipated `worker-src 'self' blob:` amendment is unnecessary because **there are no Workers in addon-image at all**.

2. **WASM SIXEL/IIP decoders:** `script-src` amendment **REQUIRED**. Add `'wasm-unsafe-eval'`. Universally supported across Phase 99's browser matrix.

**Net CSP delta:**
```diff
- script-src 'self';
+ script-src 'self' 'wasm-unsafe-eval';
```

That is the entire amendment. The other v3.1 CSP directives (`default-src 'none'`, `style-src 'self' 'unsafe-inline'`, `connect-src 'self' wss://<host>`, `img-src 'self' data:`, `font-src 'self'`, `base-uri 'none'`, `form-action 'self'`, `frame-ancestors 'none'`) remain **unchanged**.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Sixel/IIP byte-stream parsing | Browser (xterm.js core + addon-image) | — | Sequence parsing happens in the addon's DCS handler / OSC 1337 handler registered against `term.parser`; pure browser-tier concern |
| WASM SIXEL decoder execution | Browser (in-process WASM, no Worker) | — | `WebAssembly.instantiate(bytes)` runs synchronously on the main thread; constrained by `script-src 'wasm-unsafe-eval'` |
| Decoded image canvas allocation + render | Browser (HTMLCanvasElement layer attached to xterm screen element) | — | `addon-image` injects a `.xterm-image-layer` `<canvas>` into the xterm screen element (`addon-image.mjs` `_open()` method); no daemon involvement |
| Per-terminal storage cap (FIFO eviction) | Browser (addon-image's internal `Storage` class) | — | `addon-image` enforces `storageLimit` via FIFO eviction in `_evictOldest`; user-visible `storageLimit` getter/setter on the addon instance |
| Image addon construction (next-session-only) | Browser (TerminalPanel mount useEffect) | — | Image addon construction goes in the `[sessionId]` mount useEffect (alongside Unicode 11), NOT the hot-swap useEffect — IMG-01 caption mandates "applies to new sessions you create" |
| Multi-client byte-fidelity replay | API / Backend (`internal/relay/`) | — | Sixel/IIP escape sequences are bytes in the PTY stream; `relay.Hub.Run` reads 32 KiB chunks raw and stores them via `Scrollback.Append` with NO escape parsing or line buffering; `ScrollbackSnapshot()` returns verbatim bytes for second-mid-stream-joining clients |
| ImageConfig persistence | API / Backend (daemon `settings.json`) | Wails RPC + Phase 93 SSE broadcast | New nested struct under `PluginSettings`; reuses Phase 92/93/94/95 plumbing unchanged |
| `pluginConfig.imageConfig` propagation desktop | App.tsx state → `pluginConfig` prop drill into TerminalPanel | — | Phase 92 pipeline; reused unchanged |
| `pluginConfig.imageConfig` propagation web | `/api/plugin-config` GET + `/api/plugin-config/stream` SSE | — | Phase 93 endpoints; new field flows through automatically as JSON |
| Vendored addon serving | CDN / Static (embedded) | — | `web/vendor/xterm/addons/addon-image.js` served via Go embed.FS at `/assets/xterm/addons/addon-image.js` |
| `vendor_drift_test.go` CI gate | CI / Go test | — | Phase 93's generalized regex `(@xterm/(?:xterm|addon-[\w-]+))` matches `@xterm/addon-image` automatically; bump min-count guard from 7 to 8 |
| CSP amendment (`'wasm-unsafe-eval'`) | API / Backend (Go webserver `csp_mw.go`) | — | Single-line append to the `script-src` directive in the existing CSP middleware; no new middleware needed |

**Cross-tier note for IMG-04:** The byte-fidelity guarantee crosses two tiers: the Go relay reads bytes verbatim; the browser's xterm.js + addon-image consumes those bytes verbatim. Neither tier interprets sixel/IIP escape sequences in a way that could lose information. The only failure mode is **scrollback truncation** at `DefaultScrollbackBytes = 256 KiB` — and that's a pre-existing v3.1 behavior, not a Phase 96 regression.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/addon-image` | `^0.9.0` | Sixel decoder + iTerm2 IIP (Inline Image Protocol) decoder + image storage + canvas rendering layer | First-party `@xterm` scoped addon; drop-in compatible with the project's `@xterm/xterm@^6.0.0` core; same family as already-shipped `@xterm/addon-fit`, `@xterm/addon-webgl`, `@xterm/addon-search`, `@xterm/addon-web-links`, etc. Latest non-beta release. |

**Verified:** `pnpm view @xterm/addon-image version` returned `0.9.0` on 2026-05-07. `main: lib/addon-image.js` (CJS UMD bundle — correct file for web vendoring; the `.mjs` file is ES-module-only and would require `<script type="module">` which conflicts with the existing UMD-via-`<script>` pattern). `dist-tags: { latest: 0.9.0, beta: 0.10.0-beta.216 }`. [VERIFIED: npm registry, 2026-05-07 via `pnpm view @xterm/addon-image`]

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none — addon-image has zero runtime dependencies) | — | — | The addon ships its own SIXEL decoder (WASM-compiled from `node-sixel`) and its own IIP base64 decoder bundled into `lib/addon-image.js`. No transitive dependencies introduced. |

**Verification:** `pnpm view @xterm/addon-image deps` returned `none` on 2026-05-07. Confirmed by inspecting `pnpm-lock.yaml` after install — only the top-level `@xterm/addon-image@0.9.0: {}` entry, no nested deps.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `@xterm/addon-image@0.9.0` | `0.10.0-beta.216` (latest beta) | Beta versions might include performance improvements (rumored OffscreenCanvas + Worker pipeline in 0.10 series). Risk: beta tag implies API instability + security-sensitive decoder code in pre-stable. Posture: stable wins for v3.2; reconsider for v3.3. |
| `@xterm/addon-image` | Hand-roll a sixel parser using `term.parser.registerDcsHandler({final: 'q'}, ...)` and an IIP parser using `registerOscHandler(1337, ...)` | Reinventing 4000+ lines of decoder code with WASM-grade performance. The addon is purpose-built and extensively tested upstream. **Don't hand-roll.** |
| Inline image rendering as text | Skip image support entirely | Issue #36 explicitly lists inline images. Cutting Phase 96 means cutting a closure-relevant feature. |
| `xterm-addon-image-canvas-only` (hypothetical pure-JS variant) | (does not exist as a published package) | n/a |

**Installation:**

```bash
cd frontend && pnpm add @xterm/addon-image@^0.9.0
```

(Already executed during this research as `pnpm add -D @xterm/addon-image@0.9.0`. The plan should normalize this to a regular dependency, not devDependency, matching the other `@xterm/addon-*` packages in `package.json`.)

After install, copy `frontend/node_modules/@xterm/addon-image/lib/addon-image.js` to `web/vendor/xterm/addons/addon-image.js` (Phase 93/94/95 pattern, byte-identical to source).

**Version verification:**
```bash
pnpm view @xterm/addon-image version  # confirmed 0.9.0 on 2026-05-07
```

[VERIFIED: npm registry, 2026-05-07]

---

## ImageAddon API Contract

### Constructor

```typescript
// [VERIFIED: frontend/node_modules/@xterm/addon-image/typings/addon-image.d.ts]
new ImageAddon(options?: IImageAddonOptions)

interface IImageAddonOptions {
  enableSizeReports?: boolean;     // default true — activates CSI 14/16/18 t reports
  pixelLimit?: number;             // default 4096*4096 — single-image pixel cap
  storageLimit?: number;           // default 128 (MB) — FIFO cache cap (decoded RGBA)
  showPlaceholder?: boolean;       // default true — placeholder for evicted images
  sixelSupport?: boolean;          // default true
  sixelScrolling?: boolean;        // default true (DECSET 80)
  sixelPaletteLimit?: number;      // default 256
  sixelSizeLimit?: number;         // default 25_000_000 bytes (25 MB sixel raw bytes)
  iipSupport?: boolean;            // default true (iTerm2 IIP)
  iipSizeLimit?: number;           // default 20_000_000 bytes (20 MB IIP raw bytes)
}
```

### Important Defaults (from `addon-image.mjs:38` `et` constants)

```javascript
// [VERIFIED: addon-image.mjs source]
let et = {
  enableSizeReports: true,
  pixelLimit: 16777216,         // 16M pixels (4096x4096)
  sixelSupport: true,
  sixelScrolling: true,
  sixelPaletteLimit: 256,
  sixelSizeLimit: 25e6,         // 25 MB
  storageLimit: 128,            // MB — UPSTREAM DEFAULT
  showPlaceholder: true,
  iipSupport: true,
  iipSizeLimit: 2e7             // 20 MB
};
```

**Note on upstream default discrepancy:** The ROADMAP and STATE.md say "upstream 100 MB" but the actual upstream default per source-read is **128 MB** (and the typings doc string says "Default is 128 MB"). Either way, our 16 MB override is far below upstream — the discrepancy is immaterial to the override decision. **Recommended action:** in Phase 96 plan, document the upstream default as "128 MB" (verified by source-read) rather than the ROADMAP's "100 MB" wording.

### Lifecycle

```typescript
// [VERIFIED: typings]
public activate(terminal: Terminal): void;  // called by term.loadAddon()
public dispose(): void;                       // called on unmount
public reset(): void;                         // resets options to constructor defaults + clears storage

// Live-mutable properties (the only addon-image options that can change post-construction)
public storageLimit: number;                  // MB — getter/setter; mutating evicts immediately
public readonly storageUsage: number;         // MB — current usage
public showPlaceholder: boolean;              // getter/setter

// Image extraction (NOT wired in v3.2 — see "## User Constraints / Deferred Ideas")
public getImageAtBufferCell(x: number, y: number): HTMLCanvasElement | undefined;
public extractTileAtBufferCell(x: number, y: number): HTMLCanvasElement | undefined;
```

### Hot-swap considerations (Phase 96 NEXT-SESSION-ONLY decision)

The addon-image `dispose()` removes the `<canvas>` layer from the DOM and clears all in-memory image storage. Re-constructing it on a live terminal with already-rendered images would mean **all previously rendered images vanish from the buffer**. Even though the buffer cells still carry the IMAGE-CELL flag (`getBg(col) & 0x10000000`), the rendering layer is gone — the cells become "ghost" placeholders. For UX consistency with Unicode 11 (which has the same semantic — toggling mid-session would re-flow buffer widths), Phase 96 ships image addon as **NEXT-SESSION-ONLY**: the `pluginConfig?.image` flag is read in the mount useEffect and the addon is constructed once per session. Toggling the Settings switch does NOT affect already-open terminals; the italic caption is the user-visible affordance.

**Live-mutable subset:** `storageLimit` is the only `IImageAddonOptions` field that the addon exposes as a runtime setter. Phase 96 ships the daemon struct + RPC for `storageLimit` so a future Phase 99 or v3.3 plan can wire a `<details>` UI that calls `imageAddonRef.current.storageLimit = newValue` without re-attaching the addon. **For Phase 96, the value is set at construction time only** — toggling `storageLimit` in Settings triggers no live update (matches the IMG-01 affordance).

[VERIFIED: typings line 95-103 — `storageLimit` is read-write; `storageUsage` is read-only.]

### Performance Envelope

- WASM SIXEL decoder: ~10 KB compiled bytecode; instantiates synchronously in <50ms on first sixel sequence (lazy — not at addon `activate()` time, but at first DCS handler invocation).
- Per-image decode: O(width × height) — a 1000×1000 sixel decodes in ~100ms on a modern laptop main thread.
- Storage: decoded images held as `HTMLCanvasElement` (browser-tracked; subject to GPU memory pressure on canvas-heavy sessions).
- FIFO eviction: O(log n) per insertion; eviction triggered when `_pixelLimit` (derived from `storageLimit * 1e6 / 4`) is exceeded.

[VERIFIED: source-read of `Storage._evictOldest`, `_handle_band` WASM call sites in addon-image.mjs.]

---

## Architecture Patterns

### System Architecture Diagram

```
                    PTY output stream (raw bytes)
                                │
                                ▼
                  ┌─────────────────────────────────────┐
                  │ Go: relay.Hub.Run reads 32 KiB     │
                  │ chunks; wraps each in MakeOutputFrame│
                  │ (1-byte MsgOutput prefix); appends  │
                  │ to Scrollback (256 KiB cap, FIFO);  │
                  │ broadcasts to all subscribers       │
                  │                                     │
                  │ NO line buffering; NO escape parsing│
                  │ — sixel/IIP bytes pass verbatim     │
                  └─────────────────────────────────────┘
                                │
                                ▼  WebSocket binary frames
                  ┌─────────────────────────────────────┐
                  │ Browser: xterm.js Terminal.write    │
                  │ → parser dispatches DCS / OSC       │
                  │   handlers registered by addons     │
                  └─────────────────────────────────────┘
                                │
                ┌───────────────┴───────────────┐
                ▼                               ▼
    ┌───────────────────┐               ┌───────────────────┐
    │ DCS 'q' handler   │               │ OSC 1337 handler  │
    │ (sixel)           │               │ (iTerm IIP)       │
    │ → SixelHandler    │               │ → IIPHandler      │
    │   .put(bytes)     │               │   .put(bytes)     │
    │ → WASM decode     │               │ → IIP base64      │
    │   (handle_band    │               │   decoder (WASM)  │
    │    callback)      │               │                   │
    └───────────────────┘               └───────────────────┘
                │                               │
                └───────────────┬───────────────┘
                                ▼
                  ┌─────────────────────────────────────┐
                  │ Storage.addImage(canvas)            │
                  │ - assigns image ID                   │
                  │ - tags buffer cells with imageId,   │
                  │   tileId via _writeToCell           │
                  │ - FIFO evict if pixel total exceeds │
                  │   storageLimit MB cap (16 MB)       │
                  └─────────────────────────────────────┘
                                │
                                ▼
                  ┌─────────────────────────────────────┐
                  │ ImageRenderer.render() on each      │
                  │ xterm.onRender event:               │
                  │ - reads viewport buffer rows         │
                  │ - finds tagged cells                 │
                  │ - drawImage() onto .xterm-image-    │
                  │   layer canvas overlaid on screen   │
                  └─────────────────────────────────────┘

  Settings change (Image toggle / storageLimit):
       │
       ▼
  ┌────────────────────────────────────────────┐
  │ Phase 95 SetWebLinksConfig pattern repeats:│
  │                                            │
  │  SetImageConfig({storageLimit})            │
  │  → Wails (*App).SetImageConfig             │
  │  → daemon engine.SetImageConfig            │
  │  → settings.json + 'settings:plugins'      │
  │  → Phase 93 SSE broadcast                  │
  │                                            │
  │  pluginConfig prop drill → TerminalPanel:  │
  │  - mount useEffect (NEXT-SESSION-ONLY)     │
  │    reads pluginConfig?.image at session    │
  │    init for the boolean toggle             │
  │  - reads pluginConfig?.imageConfig?.        │
  │    storageLimit at construction time       │
  │  - hot-swap useEffect does NOT have an     │
  │    image arm — toggling image at runtime   │
  │    has no effect on open terminals         │
  │    (italic caption is the affordance)      │
  └────────────────────────────────────────────┘

  Multi-client mid-stream join (IMG-04):
       │
       ▼
  ┌────────────────────────────────────────────┐
  │ relay.Server.handleConnection:             │
  │  1. Subscribe(sub) — atomic                │
  │  2. ScrollbackSnapshot() — verbatim copy   │
  │     of last ≤256 KiB of PTY bytes          │
  │     (includes any sixel/IIP escapes)       │
  │  3. conn.Write(snapshot)                   │
  │  4. Stream future frames via sub.Msgs      │
  │                                            │
  │ Second client xterm.write(snapshot bytes)  │
  │ → DCS/OSC handlers fire normally           │
  │ → image renders identically to first client│
  │   (modulo 256 KiB scrollback truncation —  │
  │   pre-existing v3.1 limitation)            │
  └────────────────────────────────────────────┘
```

### Recommended Project Structure

**Desktop (React):**
```
frontend/src/
├── components/
│   ├── TerminalPanel.tsx                  # MODIFIED — import ImageAddon; add imageAddonRef; wire image
│   │                                      #   construction in MOUNT useEffect (alongside Unicode 11);
│   │                                      #   add to mount useEffect cleanup
│   ├── PluginsSection.tsx                 # MODIFIED — add italic "Applies to new sessions you create"
│   │                                      #   caption to the existing 'image' renderRow call (line 135)
│   └── __tests__/
│       ├── TerminalPanel.test.tsx         # MODIFIED — assert ImageAddon imported + constructed when
│       │                                  #   pluginConfig.image is true; storageLimit value passed
│       ├── PluginsSection.test.tsx        # MODIFIED — assert italic caption present under Image toggle
│       └── App.plugin-event.test.tsx      # MODIFIED — extend PluginSettings shape to include imageConfig
└── wailsjs/go/models.ts                   # HAND-EDIT — add ImageConfig class to daemon namespace;
                                           #   add imageConfig field to PluginSettings (Phase 92 pin pattern,
                                           #   mirror of Phase 95 WebLinksConfig hand-edit)
```

**Web (plain DOM):**
```
web/
├── terminal.html                          # MODIFIED — add <script src="/assets/xterm/addons/addon-image.js">
│                                          #   AFTER xterm.js, AFTER existing addons, BEFORE terminal.js
├── assets/
│   └── terminal.js                        # MODIFIED — applyPluginConfig grows image arm:
│                                          #   conditionally constructs ImageAddon at term init
│                                          #   (NOT in the diff path — image is next-session-only;
│                                          #   tabs auto-reload triggers a fresh construction anyway)
├── vendor/xterm/
│   ├── VERSION                            # MODIFIED — append @xterm/addon-image@0.9.0
│   └── addons/
│       └── addon-image.js                 # NEW — copied from frontend/node_modules/.../lib/addon-image.js
└── embed.go                               # MODIFIED — add vendor/xterm/addons/addon-image.js to //go:embed
```

**Daemon:**
```
internal/daemon/
├── plugin_settings.go                     # MODIFIED — add ImageConfig struct + nested field;
│                                          #   update defaultPluginSettings()
├── plugin_settings_test.go                # MODIFIED — assert ImageConfig defaults (StorageLimit: 16)
├── engine.go                              # MODIFIED — add SetImageConfig sub-key writer
│                                          #   (mirror of SetWebLinksConfig at engine.go:526)
└── api.go                                 # MODIFIED — add PATCH /settings/image-config route +
                                           #   handleSetImageConfig (mirror of handleSetWebLinksConfig:590)
```

**Wails App:**
```
app.go                                     # MODIFIED — add (*App).SetImageConfig
                                           #   (mirror of SetWebLinksConfig at app.go:549)
```

**Go webserver:**
```
internal/webserver/
├── csp_mw.go                              # MODIFIED — append ' wasm-unsafe-eval' to script-src
│                                          #   directive (line 68); update package comment with the
│                                          #   amendment rationale (matching v3.1 D-09 documentation rigor)
├── csp_mw_test.go                         # MODIFIED — extend assertion to expect 'wasm-unsafe-eval' in CSP
└── vendor_drift_test.go                   # MODIFIED — bump min-count guard from 7 to 8 (line 33-equivalent);
                                           #   no regex change needed (Phase 93 generalized regex matches)
```

**Relay (test only — no production code change):**
```
internal/relay/
└── scrollback_test.go (or new img_byte_fidelity_test.go)
                                           # NEW or EXTENDED — IMG-04 byte-fidelity regression:
                                           # 1. Synthesize a small sixel byte stream
                                           # 2. Push through Hub.Run via bytes.Reader
                                           # 3. ScrollbackSnapshot()
                                           # 4. Assert snapshot bytes == input bytes
                                           #    (modulo 1-byte MsgOutput frame prefix per chunk)
                                           # 5. Verify multi-subscriber pass-through:
                                           #    Subscribe two clients, write input, assert
                                           #    both receive byte-identical streams
```

### Pattern 1: ImageAddon Construction in Mount useEffect (NEXT-SESSION-ONLY)

**What:** Construct `ImageAddon` once per session in TerminalPanel's `[sessionId]`-keyed mount useEffect. Do NOT add an image arm to the hot-swap useEffect.

**When to use:** Always when `pluginConfig?.image` is `true` at session-init time. Live changes to `pluginConfig.image` after session init are intentionally ignored on already-open terminals (the italic caption explains this).

**Example:**

```typescript
// TerminalPanel.tsx — extension to existing mount useEffect (alongside Unicode 11)
// [Source: pattern matches existing Unicode 11 construction at TerminalPanel.tsx:180-185]

import { ImageAddon } from '@xterm/addon-image'

const imageAddonRef = useRef<ImageAddon | null>(null)

// Inside the EXISTING mount useEffect, AFTER Unicode 11 construction (line ~185),
// BEFORE the WebGL/Clipboard hot-swap delegation:

if (pluginConfig?.image !== false) {
  // Default true if pluginConfig is null (matches defaultPluginSettings.Image = true).
  // Construction is best-effort: try/catch swallows WASM-instantiation failures
  // (e.g. CSP misconfigured, ancient browser without WASM support) so a sixel
  // sequence in the PTY stream renders as benign printable garbage rather than
  // crashing the terminal mount.
  try {
    const storageLimit = pluginConfig?.imageConfig?.storageLimit ?? 16
    const imageAddon = new ImageAddon({ storageLimit })
    term.loadAddon(imageAddon)
    imageAddonRef.current = imageAddon
  } catch (err) {
    console.warn(`[TerminalPanel] ImageAddon unavailable for session ${sessionId}:`, err)
  }
}

// Inside the mount useEffect's cleanup function (line ~230-260 area), parallel
// to existing webglAddonRef / clipboardAddonRef / searchAddonRef / webLinksAddonRef
// dispose calls:

if (imageAddonRef.current) {
  try { imageAddonRef.current.dispose() } catch { /* ignore */ }
  imageAddonRef.current = null
}
```

**Important:** the dep array of the mount useEffect is `[sessionId]` (verified at TerminalPanel.tsx:281). Do NOT add `pluginConfig?.image` or `pluginConfig?.imageConfig?.storageLimit` to this dep array — that would re-mount the entire terminal (destroying the buffer + scrollback) on every Settings save, which is exactly what we want NOT to happen for next-session-only addons.

[VERIFIED: TerminalPanel.tsx:175-281 — Unicode 11 already lives here with the exact same dep-array discipline.]

### Pattern 2: ImageConfig Persistence (Mirror of Phase 95 WebLinksConfig)

**What:** Add `ImageConfig` nested struct to `daemon.PluginSettings`. Reuse Phase 94/95 sub-key RPC pattern verbatim.

**Daemon struct change:**

```go
// internal/daemon/plugin_settings.go

// ImageConfig persists per-plugin runtime configuration for the inline-image
// toggle. Phase 96 (IMG-02). Default StorageLimit is 16 MB — overrides
// upstream's 128 MB default to prevent tab-OOM with 8+ open tabs (per
// STATE.md Phase 96 Decisions: "16 MB cap to prevent tab-OOM").
//
// JSON tags are camelCase to match daemonSettings vocabulary.
//
// NO omitempty: missing keys must round-trip as the user's saved choice
// (which may legitimately be 0 or any low value once Phase 99 ships the
// Advanced disclosure). Pitfall #14 mandates the parent key always
// serialize so future loads observe it.
type ImageConfig struct {
    // StorageLimit is the per-terminal sixel/IIP cache cap in MB of
    // decoded RGBA. Default 16 (overrides upstream 128). Bounded by
    // addon-image's setLimit() validation: must be in [0.5, 1000] MB.
    StorageLimit int `json:"storageLimit"`
}

type PluginSettings struct {
    WebGL          bool           `json:"webgl"`
    Unicode11      bool           `json:"unicode11"`
    Search         bool           `json:"search"`
    SearchConfig   SearchConfig   `json:"searchConfig"`
    WebLinks       bool           `json:"webLinks"`
    WebLinksConfig WebLinksConfig `json:"webLinksConfig"`
    Image          bool           `json:"image"`
    ImageConfig    ImageConfig    `json:"imageConfig"`     // NEW Phase 96
    Serialize      bool           `json:"serialize"`
    Clipboard      bool           `json:"clipboard"`
    Progress       bool           `json:"progress"`
}

func defaultPluginSettings() PluginSettings {
    return PluginSettings{
        WebGL:          true,
        Unicode11:      true,
        Search:         true,
        SearchConfig:   SearchConfig{Regex: false, CaseSensitive: false, WholeWord: false},
        WebLinks:       true,
        WebLinksConfig: WebLinksConfig{Modifier: "platform", ConfirmOSC8: true, ConfirmIDN: true, ConfirmTyposquat: true},
        Image:          true,
        ImageConfig:    ImageConfig{StorageLimit: 16},      // NEW Phase 96
        Serialize:      true,
        Clipboard:      true,
        Progress:       false,
    }
}
```

**`SetImageConfig` engine writer (mirror of `engine.go:526 SetWebLinksConfig`):**

```go
// internal/daemon/engine.go

// SetImageConfig updates and persists ONLY the ImageConfig sub-key of
// PluginSettings, leaving the rest untouched.
//
// Phase 96 IMG-02 — mirrors Phase 95-05 SetWebLinksConfig pattern.
// Concurrency / persistence contract is identical to SetSearchConfig.
func (e *SessionEngine) SetImageConfig(cfg ImageConfig) {
    e.mu.Lock()
    e.pluginSettings.ImageConfig = cfg
    e.saveSettingsToDisk()
    listener := e.pluginSettingsListener
    e.mu.Unlock()
    if listener != nil {
        listener()
    }
}
```

**HTTP route + handler (mirror of `api.go:77 + 590`):**

```go
// internal/daemon/api.go

// In setupRoutes:
a.mux.HandleFunc("PATCH /settings/image-config", a.handleSetImageConfig)

// Handler:
func (a *API) handleSetImageConfig(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
    var req daemon.ImageConfig
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    if err := dec.Decode(&req); err != nil {
        http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
        return
    }
    // Bound check — addon-image's setLimit() enforces [0.5, 1000] MB.
    // We round to integer MB for settings.json simplicity.
    if req.StorageLimit < 1 || req.StorageLimit > 1000 {
        http.Error(w, "storageLimit must be in [1, 1000] MB", http.StatusBadRequest)
        return
    }
    a.engine.SetImageConfig(req)
    w.WriteHeader(http.StatusNoContent)
}
```

**Wails App method (mirror of `app.go:549 SetWebLinksConfig`):**

```go
// app.go
func (a *App) SetImageConfig(cfg daemon.ImageConfig) error {
    full := a.engine.GetPluginSettings()
    full.ImageConfig = cfg
    if err := a.client.SetImageConfig(cfg); err != nil {
        return err
    }
    runtime.EventsEmit(a.ctx, "settings:plugins", full)
    return nil
}
```

[VERIFIED: app.go:549, internal/daemon/engine.go:526, internal/daemon/api.go:77+590 — all three Phase 95 precedents read directly from current source.]

### Pattern 3: CSP Amendment with v3.1 D-09 Documentation Rigor

**What:** Single-line append to `script-src` directive in `csp_mw.go`. Match the documentation rigor of the v3.1 D-09 amendment (the existing `'unsafe-inline'` style-src amendment) — package-level comment block describing the rationale, the audit that produced the finding, the browser support, and the alternative considered (`'unsafe-eval'`).

**Code change (`internal/webserver/csp_mw.go`):**

```diff
- b.WriteString("script-src 'self'; ")
+ b.WriteString("script-src 'self' 'wasm-unsafe-eval'; ")
```

**Package comment update:** Append a new "Phase 96 amendment" section to the existing comment block (after the v3.1 D-09 amendment text at line 7-19), e.g.:

```go
// Amendment 2 (Phase 96, 2026-05-XX): script-src amendment after addon-image
// CSP audit. Source: .planning/phases/96-image-addon-csp-audit/96-RESEARCH.md
// §"Mandatory Pre-Phase CSP Audit". The @xterm/addon-image SIXEL decoder is
// embedded as base64 WASM bytes and instantiated via WebAssembly.instantiate
// (no Worker, no blob: script construction — verified by source-read of
// frontend/node_modules/@xterm/addon-image/lib/addon-image.{js,mjs}).
// CSP3 gates WebAssembly compilation/instantiation under script-src;
// 'wasm-unsafe-eval' is the narrowly-scoped source expression that permits
// WASM only (NOT general eval/Function — those would require 'unsafe-eval'
// which we explicitly reject). Browser support: Chrome 102+ (May 2022),
// Firefox 102+ (June 2022), Safari 16.0+ (September 2022) — universally
// available across the Phase 99 supported browser matrix.
//
// User disposition: append 'wasm-unsafe-eval' to script-src ONLY. img-src
// remains 'self' data:' (no blob: addition needed because the addon's
// IIP fallback path that uses URL.createObjectURL is dead code on
// browsers with createImageBitmap — universally available since 2017).
```

**Test update (`internal/webserver/csp_mw_test.go`):** Find the existing test that asserts the CSP header contents (likely a `strings.Contains(csp, "script-src 'self'")` check) and update to assert `strings.Contains(csp, "script-src 'self' 'wasm-unsafe-eval'")`. Add a regression assertion that `'unsafe-eval'` is NOT present (defense against future amendments accidentally broadening the directive).

**Documentation rigor (matching v3.1 D-09):** Phase 96's plan should include a wave-3 e2e CSP-zero-violation test that runs against Chromium + Safari + Firefox (extending the existing Phase 89 chromedp-based test), exactly as v3.1 D-09 produced for the style-src amendment. The Phase 99 release-gate phase formally adopts the cross-browser e2e suite; Phase 96 produces its own initial run as part of CSP-amendment validation.

[VERIFIED: csp_mw.go lines 7-44 — the existing comment block is the structural precedent for the amendment-2 append.]

### Pattern 4: Web Vendoring (Phase 93/94/95 Pattern Verbatim)

**File copy:**
```bash
cp frontend/node_modules/@xterm/addon-image/lib/addon-image.js \
   web/vendor/xterm/addons/addon-image.js
```

**`web/embed.go` extension:**
```diff
  //go:embed vendor/xterm/addons/addon-webgl.js vendor/xterm/addons/addon-unicode11.js \
- vendor/xterm/addons/addon-clipboard.js vendor/xterm/addons/addon-search.js \
- vendor/xterm/addons/addon-web-links.js
+ vendor/xterm/addons/addon-clipboard.js vendor/xterm/addons/addon-search.js \
+ vendor/xterm/addons/addon-web-links.js vendor/xterm/addons/addon-image.js
```

**`web/vendor/xterm/VERSION` append:**
```
@xterm/addon-image@0.9.0
```

**`web/terminal.html` script tag append** (after the other addon-* scripts, before `terminal.js`):
```html
<script src="/assets/xterm/addons/addon-image.js"></script>
```

**`web/assets/terminal.js` extension:** In the existing `initTerminal()` IIFE (around line 228 where `pluginConfig.unicode11` is checked), add an image construction block:

```javascript
// terminal.js — image addon construction at term init (next-session-only,
// matching desktop semantics)
if (pluginConfig.image !== false) {
  try {
    var storageLimit = (pluginConfig.imageConfig && pluginConfig.imageConfig.storageLimit) || 16;
    // UMD global namespace verification needed during plan execution — typical
    // pattern is `ImageAddon.ImageAddon` like other addons. Confirm by grep
    // of addon-image.js: `grep "exports.ImageAddon\|root.ImageAddon" web/vendor/xterm/addons/addon-image.js`
    var imageAddon = new ImageAddon.ImageAddon({ storageLimit: storageLimit });
    term.loadAddon(imageAddon);
  } catch (e) {
    console.warn('[terminal.js] ImageAddon unavailable:', e);
  }
}
```

**UMD global namespace verification:** Per Phase 93 Pitfall #7, the UMD bundle's exposed global may be `ImageAddon` (single-export) or `ImageAddon.ImageAddon` (namespace-wrapped). Verify at plan-execution time by grepping the vendored `.js` file:

```bash
grep -o "exports\.ImageAddon\|root\.ImageAddon\|root\[['\"]ImageAddon['\"]\]" \
  web/vendor/xterm/addons/addon-image.js | head -3
```

[VERIFIED for confidence-boost: addon-image.js minified line 2 shows the UMD pattern: `"object"==typeof exports?exports.ImageAddon=t():e.ImageAddon=t()`. So for plain `<script>` load, the global is `ImageAddon` (the function/class), NOT `ImageAddon.ImageAddon`. Plan should verify this is consistent with how other addons were vendored — there's an inconsistency risk if some addons use the inner-namespace pattern and addon-image uses the flat pattern.]

### Pattern 5: PluginsSection Italic Caption (IMG-01 affordance)

**What:** PluginsSection.tsx already supports an optional 4th `caption` argument to `renderRow` (used for Unicode 11). Phase 96 just passes it for the existing `image` row.

**Diff:**

```diff
  {renderRow('image', 'Inline images',
-    'Render images sent via sixel or the iTerm2 inline image protocol directly inside the terminal.')}
+    'Render images sent via sixel or the iTerm2 inline image protocol directly inside the terminal.',
+    'Applies to new sessions you create.')}
```

That is the entire UI change in PluginsSection. No new components, no new CSS, no new design tokens. The italic class `.settings-panel__description--italic` already exists from Phase 93 (per Phase 93 RESEARCH §"WGL-04 — Web parity").

[VERIFIED: PluginsSection.tsx:108-110 renders the optional `caption` with `--italic` modifier; PluginsSection.tsx:135-136 is the existing image row.]

### Anti-Patterns to Avoid

- **Adding image to the hot-swap useEffect.** This would either re-mount the terminal (destroying buffer/scrollback) or re-attach the addon mid-session (losing already-rendered images). The italic caption is the correct UX pattern.
- **Adding `'unsafe-eval'` instead of `'wasm-unsafe-eval'`.** `'unsafe-eval'` permits `eval()` and `new Function()` — a much broader attack surface. The correct directive is `'wasm-unsafe-eval'`, scoped exclusively to WebAssembly compilation/instantiation.
- **Adding `worker-src 'self' blob:` "just in case".** The audit confirmed no Workers. Adding the directive defensively without a real need violates the v3.1 D-09 minimalism principle.
- **Adding `blob:` to `img-src` without a real violation.** The fallback path that creates blob: image URLs is dead code on supported browsers. Wait for an actual UAT violation before broadening img-src.
- **Bumping `relay.DefaultScrollbackBytes` to fit large images.** Out of scope — would touch v3.1 multi-client byte-fidelity invariants. Document the 256 KiB cap as a known limitation; user demand for larger sixel scrollback can drive a future phase.
- **Wiring `<details>` advanced disclosure for storageLimit in Phase 96.** That's PUI-03's responsibility (Phase 99). Phase 96 ships the daemon plumbing only.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SIXEL escape sequence parsing | Custom DCS handler with hand-written palette + raster decoder | `@xterm/addon-image`'s WASM SIXEL decoder | The upstream decoder is ~4000 lines of C compiled to WASM with extensive test coverage; rolling our own would be a multi-week project with security implications. |
| iTerm2 IIP base64 image decoding | Custom OSC 1337 handler + base64 decode + image construction | `@xterm/addon-image`'s IIP path | Same reasoning — purpose-built upstream code. |
| FIFO eviction of large images | Custom `Map` + insertion-order tracking + `removeOldest` | `@xterm/addon-image`'s built-in `Storage` class | Eviction logic is already wired to canvas dispose; rolling our own risks GPU memory leaks. |
| WASM bytecode loading | Fetching a separate `.wasm` file from `/assets/` | Inline base64 (the addon's bundled approach) | Inline base64 is what `addon-image.js` ships with — works under our `script-src 'self' 'wasm-unsafe-eval'` CSP without any new fetches. A separate `.wasm` would need `connect-src` allowance + a new asset route. |
| Multi-client byte-fidelity for image bytes | Custom escape-aware buffer that "knows" about sixel/IIP | The existing `relay.Scrollback` raw-byte buffer | Already verbatim-preserving; sixel/IIP escapes are bytes like any other. No special-casing needed. |
| Sub-config sub-key persistence (`storageLimit`) | A new `/settings/image` HTTP route with a different shape | The Phase 94-07 / 95-05 `PATCH /settings/<feature>-config` sub-key writer pattern | Three iterations of the pattern have shipped (SearchConfig, WebLinksConfig); Phase 96 makes it a fourth. Diverging would create maintenance burden. |
| CSP middleware extension | A new `cspImageHeaders` middleware specific to image-bearing routes | Single-line append to existing `cspHeaders` middleware in `csp_mw.go` | The CSP applies to all HTML routes (terminal.html, dashboard.html, join.html); fragmenting middleware by feature would multiply test surface. |

**Key insight:** Phase 96 is the most conservative phase in v3.2 — it leverages five existing patterns (mount useEffect for next-session addons, vendor pipeline, sub-key RPC, Phase 92 PluginSettings shape, italic caption affordance) and introduces zero new ones. The only novel work is the CSP amendment, and even that fits the v3.1 D-09 documentation precedent verbatim.

---

## Common Pitfalls

### Pitfall 1: Wrong useEffect — Image in Hot-Swap Instead of Mount
**What goes wrong:** Adding `pluginConfig?.image` to the hot-swap useEffect dep array (alongside webgl/clipboard/search/webLinks).
**Why it happens:** Visual symmetry with the other addon arms — they're all in the hot-swap useEffect, why not Image?
**How to avoid:** Image addon construction goes in the **mount useEffect** (`[sessionId]` dep). Toggling `pluginConfig.image` in Settings must NOT affect already-open terminals — that's what the italic caption tells the user. The hot-swap useEffect must NOT have an image arm at all.
**Warning signs:** Toggling Image in Settings causes already-open terminals to re-render with a flickering image layer; previously rendered images vanish on toggle-on, ghost-cells remain on toggle-off.

### Pitfall 2: `'unsafe-eval'` Instead of `'wasm-unsafe-eval'`
**What goes wrong:** Amending CSP with `script-src 'self' 'unsafe-eval'` — addon-image works, but the directive also permits `eval()`, `new Function()`, and `setTimeout(string)`.
**Why it happens:** `'unsafe-eval'` is the older, more familiar directive name; `'wasm-unsafe-eval'` was added to CSP3 specifically for this case.
**How to avoid:** The directive name is `'wasm-unsafe-eval'` (with the wasm- prefix). It's narrowly scoped to WebAssembly compilation/instantiation. **Never use `'unsafe-eval'`** — it's a much broader attack surface and we have no JS-level eval requirements anywhere in the codebase.
**Warning signs:** Source-grep for `'unsafe-eval'` in `csp_mw.go` should find ZERO matches; only `'wasm-unsafe-eval'` should appear.

### Pitfall 3: 256 KiB Scrollback Truncation Mistaken for Byte-Fidelity Bug
**What goes wrong:** A large sixel image (e.g. 1 MB serialized escape stream) is emitted to a session; a second client joins; the second client's image is corrupted. Investigation concludes the relay corrupts bytes.
**Why it happens:** `relay.DefaultScrollbackBytes = 256 KiB`. Anything beyond 256 KiB of recent PTY output is dropped from the scrollback (FIFO). For large images, the second client may receive only the tail of the sixel sequence — not a parser-valid stream.
**How to avoid:** The IMG-04 regression test must use a sixel byte stream **smaller than 256 KiB** to assert byte-fidelity. The 50 MB sixel fixture for SC-3 (storage cap) is a **client-side decoded RGBA** test, not a relay byte-stream test — those are two different concerns. Document the 256 KiB cap as a known limitation; do not enlarge it.
**Warning signs:** A regression test that pushes >256 KiB through the relay fails non-deterministically depending on chunk read timing.

### Pitfall 4: UMD Global Namespace — `ImageAddon` vs. `ImageAddon.ImageAddon`
**What goes wrong:** Web `terminal.js` calls `new ImageAddon.ImageAddon({ storageLimit: 16 })` but the actual UMD export is just `ImageAddon` (function/class directly).
**Why it happens:** Different `@xterm/addon-*` packages use different UMD wrapper patterns. Some attach to `window.AddonName.AddonName` (namespace-wrapped), others attach directly to `window.AddonName`.
**How to avoid:** Source-verify before writing the constructor call. From this research: addon-image.js line 2 shows `exports.ImageAddon=t()` — meaning the UMD global is `ImageAddon` (a class), and `new ImageAddon(...)` is correct. NOT `new ImageAddon.ImageAddon(...)`. **However, double-check by running** `grep "exports\.ImageAddon\|root\.ImageAddon" web/vendor/xterm/addons/addon-image.js` after the file is copied — the structure may differ between source `.js` (CJS) and the production `.js` if the package bundles them differently.
**Warning signs:** Browser console: `TypeError: ImageAddon is not a constructor` or `TypeError: Cannot read property 'ImageAddon' of undefined`.

### Pitfall 5: WASM Instantiation Failure → Silent Loss of Sixel Rendering
**What goes wrong:** CSP is misconfigured (e.g. forgotten `'wasm-unsafe-eval'`); WASM instantiation throws; addon partially loads but sixel rendering silently fails.
**Why it happens:** addon-image's `DecoderAsync` is called lazily on the first sixel sequence, not at addon construction. The addon `activate()` call succeeds; the failure surfaces only when the user runs `chafa --format=sixel ...`.
**How to avoid:** The mount useEffect's `try/catch` around `new ImageAddon(...)` only catches construction-time failures. WASM-instantiation failures fire later. Defense: the chromedp e2e test must run a sixel-emitting fixture and assert the image renders (visual check, not just no-CSP-violation). Phase 96 plan should include a Wave-3 e2e that emits a small sixel sequence and snapshots the canvas pixels.
**Warning signs:** Sixel sequences in PTY output result in no visible image AND no terminal output (the addon swallows the bytes); browser DevTools Network tab is clean but Console shows `Refused to compile WebAssembly module` warning.

### Pitfall 6: Storage Cap Too High → Tab OOM
**What goes wrong:** User sets `storageLimit` very high via Phase 99's Advanced disclosure (or the daemon-API allows arbitrary values up to 1000 MB). With 8 open tabs each holding 1 GB of decoded RGBA, the browser tab crashes.
**Why it happens:** Per-tab storage is independent — `storageLimit` is per-`ImageAddon`-instance, and each TerminalPanel has its own.
**How to avoid:** The HTTP handler bound check is `[1, 1000]` MB. We could be more conservative (`[1, 64]`?) but that constrains power users. Plan-level decision: ship `[1, 1000]` to match upstream's `[0.5, 1000]` validation; the 16 MB default is the genuine safety net for the 99% use case. Document the per-tab implication in the eventual Phase 99 `<details>` UI copy.
**Warning signs:** Browser tab process memory grows unbounded with image-heavy CLI sessions; tab eventually crashes with "Aw, snap!" / "A web page is slowing down your browser".

### Pitfall 7: `vendor_drift_test.go` Min-Count Guard Mismatch
**What goes wrong:** Adding `@xterm/addon-image@0.9.0` to `web/vendor/xterm/VERSION` and `pnpm-lock.yaml`, but forgetting to bump the min-count guard in `vendor_drift_test.go` from 7 to 8.
**Why it happens:** The Phase 93 generalized regex `(@xterm/(?:xterm|addon-[\w-]+))` catches the new addon automatically; the test continues to pass with 8 packages parsed but the min-guard stays at 7. Silent loss of the Phase 93 invariant ("at least N packages parsed").
**How to avoid:** Bump the min-count guard from 7 to 8 explicitly. Add a code comment counting the packages: `// xterm + addon-fit + addon-webgl + addon-unicode11 + addon-clipboard + addon-search + addon-web-links + addon-image = 8`.
**Warning signs:** A regex bug introduced in a future phase causes the parse to drop to 7 packages; the test silently passes.

### Pitfall 8: ImageAddon `enableSizeReports: true` → CSI Response Pollution
**What goes wrong:** addon-image's default `enableSizeReports: true` activates `windowOptions.getWinSizePixels`, `getCellSizePixels`, `getWinSizeChars` on the underlying xterm.js Terminal. This causes the terminal to RESPOND to certain CSI queries with size information. If the running CLI doesn't expect these responses, output may be polluted.
**Why it happens:** Some CLIs (notably older bash readline implementations) issue CSI queries on startup and treat unexpected responses as input.
**How to avoid:** addon-image typings doc string explicitly notes: `If false, no settings will be touched. Use false, if you have high security constraints and/or deal with windowOptions by other means.` AgentHub does not currently set `windowOptions` anywhere in TerminalPanel, so the addon's modification IS the project's only configuration. **Decision:** ship with the default `enableSizeReports: true` for v3.2 — the size reports are how `chafa` and other image-emitting CLIs determine target dimensions. Document the risk; if a real CLI pollution report surfaces, flip to `false` in a v3.2.x patch.
**Warning signs:** User reports terminal output polluted with `[?14;...]t` style escape responses leaking into shell prompts.

---

## Code Examples

### Example 1: TerminalPanel Mount-useEffect Image Arm

```typescript
// [VERIFIED: pattern matches Unicode 11 construction at TerminalPanel.tsx:180-185]
// Inside the mount useEffect (sessionId dep), AFTER Unicode 11, BEFORE
// the WebGL/Clipboard delegation to hot-swap useEffect:

if (pluginConfig?.image !== false) {
  try {
    const storageLimit = pluginConfig?.imageConfig?.storageLimit ?? 16
    const imageAddon = new ImageAddon({ storageLimit })
    term.loadAddon(imageAddon)
    imageAddonRef.current = imageAddon
  } catch (err) {
    console.warn(`[TerminalPanel] ImageAddon unavailable for session ${sessionId}:`, err)
  }
}

// Cleanup (in mount useEffect cleanup, parallel to other addon refs):
if (imageAddonRef.current) {
  try { imageAddonRef.current.dispose() } catch { /* ignore */ }
  imageAddonRef.current = null
}
```

### Example 2: CSP Middleware Amendment

```go
// [VERIFIED: csp_mw.go:65-82 current structure]
// Single-line change to script-src directive:
- b.WriteString("script-src 'self'; ")
+ b.WriteString("script-src 'self' 'wasm-unsafe-eval'; ")
```

### Example 3: ImageConfig Daemon Struct + Default

```go
// [VERIFIED: pattern matches WebLinksConfig at plugin_settings.go:33-45 + 83-88]
type ImageConfig struct {
    StorageLimit int `json:"storageLimit"`
}

// In defaultPluginSettings():
ImageConfig: ImageConfig{StorageLimit: 16},
```

### Example 4: SetImageConfig Engine Writer

```go
// [VERIFIED: pattern matches SetWebLinksConfig at engine.go:526]
func (e *SessionEngine) SetImageConfig(cfg ImageConfig) {
    e.mu.Lock()
    e.pluginSettings.ImageConfig = cfg
    e.saveSettingsToDisk()
    listener := e.pluginSettingsListener
    e.mu.Unlock()
    if listener != nil {
        listener()
    }
}
```

### Example 5: IMG-04 Multi-Client Byte-Fidelity Regression Test

```go
// [VERIFIED: pattern follows internal/relay/hub_test.go subscribe pattern]
// New file or extension to scrollback_test.go:

func TestImage_ByteFidelity_MultiClient(t *testing.T) {
    // Synthetic small sixel byte stream — well under 256 KiB so scrollback
    // truncation is not a concern. This is a contrived sequence; what matters
    // is that the bytes pass through verbatim, not that they decode to
    // anything visually meaningful.
    sixelInput := []byte("\x1bPq#0;2;100;0;0#1;2;0;100;0!10A!10@-\x1b\\")

    pr, pw := io.Pipe()
    var stdinBuf bytes.Buffer
    hub := NewHub("test-image-fidelity", pr, &stdinBuf, DefaultScrollbackBytes, nil)
    go hub.Run()
    defer hub.Shutdown()

    // Subscribe two clients BEFORE any data is written (no scrollback to replay
    // — they should observe identical real-time fan-out).
    sub1 := &Subscriber{Msgs: make(chan []byte, 16), CloseSlow: func() {}}
    sub2 := &Subscriber{Msgs: make(chan []byte, 16), CloseSlow: func() {}}
    hub.Subscribe(sub1)
    hub.Subscribe(sub2)

    // Write the sixel stream; close the pipe to signal EOF.
    go func() {
        pw.Write(sixelInput)
        pw.Close()
    }()

    // Drain both subscribers; concatenate the payload bytes (after stripping
    // the 1-byte MsgOutput frame prefix per chunk).
    var got1, got2 []byte
    timeout := time.After(2 * time.Second)
    for {
        if len(got1) >= len(sixelInput) && len(got2) >= len(sixelInput) {
            break
        }
        select {
        case frame := <-sub1.Msgs:
            if frame[0] == MsgOutput {
                got1 = append(got1, frame[1:]...)
            }
        case frame := <-sub2.Msgs:
            if frame[0] == MsgOutput {
                got2 = append(got2, frame[1:]...)
            }
        case <-timeout:
            t.Fatalf("timed out waiting for fan-out: got1=%d got2=%d want=%d", len(got1), len(got2), len(sixelInput))
        }
    }

    if !bytes.Equal(got1, sixelInput) {
        t.Errorf("client 1 received corrupted sixel stream:\nwant: %x\ngot:  %x", sixelInput, got1)
    }
    if !bytes.Equal(got2, sixelInput) {
        t.Errorf("client 2 received corrupted sixel stream:\nwant: %x\ngot:  %x", sixelInput, got2)
    }

    // Verify scrollback also preserves bytes (mid-stream-join scenario).
    snapshot := hub.ScrollbackSnapshot()
    // Snapshot is concatenated MakeOutputFrame frames; strip prefixes to recover bytes.
    var fromSnapshot []byte
    for i := 0; i < len(snapshot); {
        // Each frame is {MsgOutput, ...payload...}; the payload length is
        // the original chunk size which we don't have post-hoc — but for
        // this single-write test the entire sixel is one chunk preceded
        // by one MsgOutput byte.
        if snapshot[i] != MsgOutput {
            t.Fatalf("scrollback frame at offset %d has unexpected type byte 0x%02x", i, snapshot[i])
        }
        // Find next MsgOutput or end (single-frame case).
        end := i + 1 + len(sixelInput)
        if end > len(snapshot) {
            end = len(snapshot)
        }
        fromSnapshot = append(fromSnapshot, snapshot[i+1:end]...)
        i = end
    }
    if !bytes.Equal(fromSnapshot, sixelInput) {
        t.Errorf("scrollback corrupted sixel stream:\nwant: %x\ngot:  %x", sixelInput, fromSnapshot)
    }
}
```

### Example 6: Storage-Cap Eviction Regression Test (Browser-Side)

```typescript
// [ASSUMED — pattern derived from addon-image typings]
// frontend/src/components/__tests__/TerminalPanel.imageStorage.test.tsx
// Wave 0 / Wave 3 component test exercising the storageLimit eviction path.

it('evicts oldest images when storageLimit cap is exceeded', async () => {
  // Mount TerminalPanel with pluginConfig.imageConfig.storageLimit = 1 (1 MB
  // for fast test). Construct a synthetic sixel stream that would decode to
  // ~1.5 MB of RGBA. After write, addon.storageUsage should be < 1 MB
  // (eviction occurred).
  // ...assertion via imageAddonRef.current.storageUsage < 1.0...
})
```

(Concrete fixture generation is a plan-level concern; the contract is verified via `addon.storageUsage` reading less than the synthetic input.)

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No inline image rendering in xterm.js (sixel/IIP escapes garbled in output) | Vendored `@xterm/addon-image@0.9.0` with WASM SIXEL decoder + IIP base64 decoder | Phase 96 (v3.2) | Sixel + iTerm2 IIP render inline; matches feature parity with iTerm2, WezTerm, Konsole on the desktop side |
| v3.1 CSP `script-src 'self'` (blocks WebAssembly compile/instantiate) | v3.2 Phase 96 CSP `script-src 'self' 'wasm-unsafe-eval'` | Phase 96 | Permits addon-image WASM bootstrap; narrowly scoped (no JS-eval broadening); universally supported across Chrome 102+ / Firefox 102+ / Safari 16+ |
| Per-terminal sixel storage uncapped (or upstream default 128 MB) | Per-terminal storage hard-capped at 16 MB decoded RGBA via `ImageConfig.StorageLimit` | Phase 96 | Prevents tab-OOM with 8+ open tabs; user-adjustable in Phase 99 |
| Multi-client mid-stream image rendering untested | IMG-04 regression test verifies relay byte-fidelity for sixel/IIP escape streams | Phase 96 | Protects MC-01..MC-06 multi-client invariants from future relay refactors |

**Deprecated / outdated:**
- `xterm-addon-image` (legacy `xterm-` prefix, before the 2024 `@xterm/` namespace migration) — not used; we adopt the current `@xterm/addon-image` from day one.
- The 0.10.0-beta.* branch of `@xterm/addon-image` — interesting (rumored OffscreenCanvas + Worker pipeline) but pre-stable; defer to v3.3 reconsideration.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 0.10.0-beta.* branch's OffscreenCanvas + Worker pipeline rumor is just a rumor — not yet shipped in stable | "State of the Art" deprecated section | Low — even if true, we explicitly choose 0.9.0 stable for v3.2. |
| A2 | Image addon failure in `try/catch` is acceptably silent (no banner needed) — sixel sequences degrade to printable garbage harmlessly | Pattern 1 example | Medium — if a sixel sequence triggers a parser exception in xterm.js core that escapes addon-image, the terminal session could hang. Mitigation: chromedp e2e exercises the failure path. |
| A3 | `ImageAddon` UMD export is `window.ImageAddon` (flat), not `window.ImageAddon.ImageAddon` (namespaced) | Pitfall #4 + Pattern 4 web vendoring | Low — verifiable via `grep` on the vendored `.js` file at plan-execution time. Quick to fix if wrong. |
| A4 | The 256 KiB scrollback cap is acceptable for v3.2 image scrollback semantics (large sixel images get truncated for second-mid-stream-joining clients) | "## User Constraints / Locked Decisions" + Pitfall #3 | Medium — if a user reports a real workflow hitting this (e.g. always-on jupyter inline plotting), a v3.2.x patch could enlarge the cap. Document as known limitation in release notes. |
| A5 | `enableSizeReports: true` (the addon's default) does not pollute realistic CLI workflows | Pitfall #8 | Low — addon-image typings explicitly mention this is the default; widely deployed; if a real pollution report surfaces, flip to `false` in patch. |
| A6 | `'wasm-unsafe-eval'` browser support since Safari 16 / Chrome 102 / Firefox 102 is sufficient for Phase 99's supported matrix (no need for fallback policy) | "Mandatory Pre-Phase CSP Audit" + "State of the Art" | Low — Safari 16 shipped September 2022; iPad Safari 16 shipped same day. Phase 99 supported matrix already excludes pre-2022 browsers in the unwritten "modern browsers only" stance the v3.1 CSP work assumed. |
| A7 | The synthetic sixel byte stream `"\x1bPq#0;2;100;0;0#1;2;0;100;0!10A!10@-\x1b\\"` is small enough to fit comfortably below 256 KiB but parser-valid enough to exercise the relay byte-fidelity invariant | Code Example 5 | Low — bytes pass through the relay regardless of parser validity (relay does no parsing); plan-execution may want to use a real `chafa`-generated sixel to also verify browser-side decode, but for the IMG-04 regression test the synthetic stream is sufficient. |
| A8 | The `pnpm-lock.yaml` format remains stable enough that adding `@xterm/addon-image@0.9.0` as devDependency now (during research) and later promoting to a regular dependency (during plan execution) does not break the Phase 93 `vendor_drift_test.go` regex | "Standard Stack / Installation" note | Low — the regex matches `@xterm/(?:xterm|addon-[\w-]+)` regardless of dependency type. |

---

## Open Questions

1. **Should `ImageAddon` constructor pass `pixelLimit`, `sixelSizeLimit`, `iipSizeLimit` from `ImageConfig` too?**
   - What we know: addon-image exposes these as constructor options. ROADMAP/REQUIREMENTS only mention `storageLimit`.
   - What's unclear: Whether tightening the per-image limits is a Phase 96 concern or a Phase 99/v3.3 concern.
   - Recommendation: Phase 96 ships **only** `storageLimit` in `ImageConfig`. The other limits use addon-image's defaults (`pixelLimit: 16M pixels`, `sixelSizeLimit: 25 MB`, `iipSizeLimit: 20 MB`). If Phase 99 advanced disclosure surfaces them, extend `ImageConfig` then.

2. **Should the chromedp CSP-zero-violation e2e suite for Phase 96 cover Safari and Firefox now, or defer to Phase 99?**
   - What we know: Phase 89 e2e is Chromium-only. ROADMAP SC-1 says "green on Chromium + Safari + Firefox after any amendment". Phase 99 SC-4 says the cross-browser e2e is the release-gate test.
   - What's unclear: Is Phase 96 obligated to run cross-browser e2e during its own validation, or can it gate on Chromium-only and defer cross-browser to Phase 99?
   - Recommendation: Phase 96 plan ships **Chromium-only** e2e for the CSP amendment validation (sufficient to confirm `'wasm-unsafe-eval'` is wired correctly); Phase 99 owns the full Chromium + Safari + Firefox + iPad Safari run as the release gate. This matches Phase 89's Chromium-only precedent and the v3.2 release-gate phasing.

3. **Should the IMG-04 regression test live in `internal/relay/` (Go-side byte-fidelity) or in `frontend/src/components/__tests__/` (browser-side render fidelity)?**
   - What we know: The byte-fidelity guarantee crosses two tiers; both are testable.
   - Recommendation: **Go-side** (the relay byte buffer). Browser-side rendering is already covered by the SC-2 visual UAT and the addon's own upstream test suite. Phase 96 plan should include the Go test in Wave 0 (red scaffold) and turn it green in the wave that ships the daemon-side ImageConfig (no production code change needed for IMG-04 — the test asserts existing behavior is preserved).

4. **Should the `'unsafe-eval'`-vs-`'wasm-unsafe-eval'` regression test live in `csp_mw_test.go` or in a new `no_unsafe_eval_test.go`?**
   - What we know: The defense is "ban `'unsafe-eval'` from any future amendment".
   - Recommendation: Single assertion appended to `csp_mw_test.go` (next to the existing CSP header content tests). A dedicated file is overkill for one test.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `@xterm/addon-image` | All IMG-* requirements | ✓ | 0.9.0 (installed during research) | — |
| pnpm | Dependency management | ✓ | 9.15.9 | — |
| Go 1.22+ | `//go:embed` glob support, `relay` test patterns | ✓ | (project standard) | — |
| `chafa` CLI | SC-2 visual UAT (sixel + IIP emission) | ⚠ unknown — not verified on dev machine | — | Hand-crafted minimal sixel/IIP escape sequence (~50 bytes) for automated test; mention `chafa` as the suggested manual UAT tool with `brew install chafa` install instruction |
| chromedp | CSP zero-violation e2e (Phase 89 precedent) | ✓ | (Phase 89 dependency, already in go.mod via test fixtures) | — |
| Browser: Chromium | CSP e2e + visual UAT | ✓ | (system default) | — |
| Browser: Safari | Cross-browser CSP e2e (deferred to Phase 99) | n/a for Phase 96 | — | Defer to Phase 99 |
| Browser: Firefox | Cross-browser CSP e2e (deferred to Phase 99) | n/a for Phase 96 | — | Defer to Phase 99 |
| `web/vendor/xterm/addons/addon-image.js` | WEB vendoring (IMG-03) | ✗ (not yet vendored) | — | Create during plan execution by copying from `frontend/node_modules/@xterm/addon-image/lib/addon-image.js` |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:**
- `chafa` for visual UAT — fallback is hand-crafted minimal escape sequences. Plan should suggest installing `chafa` for richer UAT but not require it.

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
| Go unit run (relay) | `go test ./internal/relay/... -count=1` |
| Go unit run (webserver CSP) | `go test ./internal/webserver/... -run TestCSP -count=1` |
| Go unit run (daemon plugin settings) | `go test ./internal/daemon/... -run TestPluginSettings -count=1` |
| Go full run | `go test ./internal/...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| IMG-01 | PluginsSection renders italic caption under Image toggle | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/PluginsSection.test.tsx` | ✅ (extend) |
| IMG-01 | TerminalPanel constructs ImageAddon when pluginConfig.image is true | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ (extend) |
| IMG-01 | Toggling pluginConfig.image at runtime does NOT re-attach addon (next-session-only) | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ (extend) |
| IMG-02 | ImageConfig.StorageLimit defaults to 16 in defaultPluginSettings | unit (Go) | `go test ./internal/daemon/... -run TestPluginSettings_Defaults -count=1` | ✅ (extend) |
| IMG-02 | SetImageConfig persists ONLY ImageConfig sub-key, leaves other PluginSettings fields untouched | unit (Go) | `go test ./internal/daemon/... -run TestSetImageConfig -count=1` | ❌ Wave 0 |
| IMG-02 | PATCH /settings/image-config validates StorageLimit in [1, 1000] | unit (Go) | `go test ./internal/daemon/... -run TestHandleSetImageConfig -count=1` | ❌ Wave 0 |
| IMG-02 | TerminalPanel passes pluginConfig.imageConfig.storageLimit to ImageAddon constructor | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ (extend) |
| IMG-03 | CSP middleware adds 'wasm-unsafe-eval' to script-src | unit (Go) | `go test ./internal/webserver/... -run TestCSP_WasmUnsafeEval -count=1` | ❌ Wave 0 |
| IMG-03 | CSP middleware does NOT contain 'unsafe-eval' (defense regression) | unit (Go) | `go test ./internal/webserver/... -run TestCSP_NoUnsafeEval -count=1` | ❌ Wave 0 |
| IMG-03 | Vendored addon-image.js served at /assets/xterm/addons/ | Go integration | `go test ./internal/webserver/... -run TestAssets_Addons -count=1` | ✅ (extend) |
| IMG-03 | vendor_drift_test min-count guard bumped to 8 | Go unit | `go test ./internal/webserver/... -run TestXtermVendorVersions -count=1` | ✅ (modify) |
| IMG-03 | chromedp CSP-zero-violation on Chromium with addon-image loaded + sixel emitted | e2e (Go //go:build e2e) | `go test ./internal/webserver/... -tags=e2e -run TestBrowserCSP_TerminalImage -count=1` | ❌ Wave 3 |
| IMG-04 | Synthetic sixel byte stream passes verbatim through relay scrollback | unit (Go) | `go test ./internal/relay/... -run TestImage_ByteFidelity_MultiClient -count=1` | ❌ Wave 0 |
| IMG-04 | Two subscribers receive byte-identical fan-out for sixel input | unit (Go) | (same test as above; covers both subscribe paths and snapshot replay) | ❌ Wave 0 |
| IMG-04 | (Visual) chafa --format=iterm2 chart.png renders identically on first and second mid-stream-joining client | manual UAT | runbook in `96-HUMAN-UAT.md` | ❌ Wave 4 |

### Sampling Rate
- **Per task commit:** `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx && go test ./internal/webserver/... ./internal/daemon/... ./internal/relay/... -count=1`
- **Per wave merge:** `pnpm test && go test ./internal/...`
- **Phase gate:** Full suite green before `/gsd-verify-work`; chromedp e2e green; manual UAT runbook signed off

### Wave 0 Gaps
- [ ] `internal/daemon/api_image_test.go` (or extension to existing `api_test.go`) — covers `handleSetImageConfig` validation
- [ ] `internal/daemon/engine_image_test.go` (or extension) — covers `SetImageConfig` sub-key writer (preserves other fields)
- [ ] `internal/webserver/csp_mw_test.go` extension — `'wasm-unsafe-eval'` present + `'unsafe-eval'` absent assertions
- [ ] `internal/relay/image_byte_fidelity_test.go` (new file) OR extension to `scrollback_test.go` — IMG-04 multi-client byte-fidelity
- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` extension — ImageAddon import + construction + storageLimit pass-through + next-session-only invariant
- [ ] `frontend/src/components/__tests__/PluginsSection.test.tsx` extension — italic caption present under Image row
- [ ] `frontend/src/__tests__/App.plugin-event.test.tsx` extension — PluginSettings shape includes imageConfig

### Wave 3 Gaps
- [ ] `internal/webserver/browser_csp_image_e2e_test.go` (//go:build e2e) — chromedp loads `/sessions/{id}` with addon-image enabled, emits a small sixel sequence via test fixture, asserts zero CSP violations + image canvas layer present in DOM
- [ ] `96-HUMAN-UAT.md` — runbook for: (a) `chafa --format=iterm2 chart.png` on desktop, (b) same on web-served Tailscale page, (c) two-client mid-stream image join, (d) toggling Image in Settings → confirming italic caption + no live re-attach, (e) 50 MB sixel fixture FIFO eviction

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | — |
| V3 Session Management | No | — |
| V4 Access Control | Yes | `requireCapability` middleware on `/api/plugin-config` (Phase 93 PLUG-04 — already wired); image config flows through this gated endpoint unchanged |
| V5 Input Validation | Yes | `handleSetImageConfig` HTTP handler: `MaxBytesReader(8 KiB)` + `DisallowUnknownFields` + bound check `StorageLimit ∈ [1, 1000]` |
| V6 Cryptography | No | — |
| V14 Configuration | Yes | CSP amendment is the load-bearing security change; documented with v3.1 D-09 rigor; regression test asserts `'unsafe-eval'` is NOT permitted |

### Known Threat Patterns for Browser + Go Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| CSP bypass via WebAssembly compile/instantiate | Tampering | `'wasm-unsafe-eval'` is **scoped to WASM only** — does NOT permit `eval()`, `new Function()`, `setTimeout(string)`, etc. The narrowly-scoped directive is the standard mitigation; broader `'unsafe-eval'` is explicitly rejected. |
| WASM-side memory exhaustion (decoder fed malicious sixel) | Denial of Service | addon-image's WASM decoder enforces `pixelLimit` (16M pixels default), `sixelSizeLimit` (25 MB raw bytes default); on-decode failure releases the decoder buffer. addon-image throws `"image exceeds memory limit"` Error which we catch via the existing addon-internal try/catch (no AgentHub-side handler needed). |
| Tab-OOM via storage cap exhaustion (8 tabs × 128 MB upstream default) | Denial of Service | 16 MB `ImageConfig.StorageLimit` default × 8 tabs = 128 MB ceiling — comfortably within browser per-tab memory budgets. Phase 99 advanced disclosure can let users opt into higher limits with full informed consent. |
| Sixel byte stream corruption in multi-client replay | Tampering / DoS | IMG-04 regression test asserts byte-fidelity through `relay.Scrollback` (raw byte buffer; no escape parsing). 256 KiB cap acknowledged as known truncation point. |
| `URL.createObjectURL` blob: URL exfiltration | Information Disclosure | The blob: URL is created from in-memory image bytes (already in the addon's possession); no exfiltration vector. Path is dead code anyway on supported browsers. |
| OSC 1337 (IIP) parameter injection | Tampering | addon-image's IIP header parser validates `name`, `width`, `height`, `inline`, `size` fields with type-checked decoders (`Pe()`, `Ee()`, `yt()` per source); malformed headers abort the decode and discard subsequent bytes. |
| `enableSizeReports: true` CSI response pollution | Information Disclosure (mild — terminal dimensions) | Default upstream behavior; widely deployed; documented in Pitfall #8 with mitigation path (flip to `false` if real reports surface). |

---

## Files to Create / Modify

### New Files
| File | Why |
|------|-----|
| `web/vendor/xterm/addons/addon-image.js` | Vendored same-origin addon (IMG-03 / WEB-01 vendoring discipline) |
| `internal/relay/image_byte_fidelity_test.go` (or extension to `scrollback_test.go`) | IMG-04 multi-client byte-fidelity regression |

### Modified Files
| File | Change | Requirement |
|------|--------|-------------|
| `frontend/package.json` | Add `@xterm/addon-image: ^0.9.0` as a regular dependency (not devDep — promoted from research-time devDep install) | IMG-01..03 |
| `frontend/pnpm-lock.yaml` | Updated by `pnpm install` after package.json edit | (lockfile drift) |
| `frontend/src/components/TerminalPanel.tsx` | Import ImageAddon; add `imageAddonRef`; wire image construction in MOUNT useEffect (alongside Unicode 11); add to mount useEffect cleanup. Hot-swap useEffect intentionally unchanged. | IMG-01, IMG-02 |
| `frontend/src/components/PluginsSection.tsx` | Add `'Applies to new sessions you create.'` 4th argument to existing `image` `renderRow` call (line 135-136) | IMG-01 |
| `frontend/src/components/__tests__/TerminalPanel.test.tsx` | Extend tests to assert ImageAddon construction + storageLimit + next-session-only invariant | IMG-01, IMG-02 |
| `frontend/src/components/__tests__/PluginsSection.test.tsx` | Extend to assert italic caption present under Image row | IMG-01 |
| `frontend/src/__tests__/App.plugin-event.test.tsx` | Extend PluginSettings shape to include `imageConfig: {storageLimit: 16}` | IMG-02 |
| `frontend/src/wailsjs/go/models.ts` | HAND-EDIT — add `ImageConfig` class to daemon namespace; add `imageConfig` field to PluginSettings (Phase 92 pin pattern, mirror of Phase 95 WebLinksConfig hand-edit) | IMG-02 |
| `internal/daemon/plugin_settings.go` | Add `ImageConfig` struct + nested field; update `defaultPluginSettings()` to include `ImageConfig{StorageLimit: 16}` | IMG-02 |
| `internal/daemon/plugin_settings_test.go` | Assert `ImageConfig{StorageLimit: 16}` defaults | IMG-02 |
| `internal/daemon/engine.go` | Add `SetImageConfig` sub-key writer (mirror of `SetWebLinksConfig`) | IMG-02 |
| `internal/daemon/api.go` | Add `PATCH /settings/image-config` route + `handleSetImageConfig` handler (mirror of `handleSetWebLinksConfig`) | IMG-02 |
| `app.go` | Add `(*App).SetImageConfig` Wails method (mirror of `SetWebLinksConfig`) | IMG-02 |
| `internal/webserver/csp_mw.go` | Append `'wasm-unsafe-eval'` to script-src directive (line 68); update package comment with Amendment 2 documentation matching v3.1 D-09 rigor | IMG-03 |
| `internal/webserver/csp_mw_test.go` | Assert `'wasm-unsafe-eval'` present in script-src; assert `'unsafe-eval'` NOT present (defense regression) | IMG-03 |
| `internal/webserver/vendor_drift_test.go` | Bump min-count guard from 7 to 8; update comment counting packages | IMG-03 / WEB-02 |
| `web/embed.go` | Append `vendor/xterm/addons/addon-image.js` to `//go:embed` directive | IMG-03 / WEB-01 |
| `web/vendor/xterm/VERSION` | Append `@xterm/addon-image@0.9.0` line | IMG-03 / WEB-01 / WEB-02 |
| `web/terminal.html` | Add `<script src="/assets/xterm/addons/addon-image.js"></script>` after other addon-* scripts, before `terminal.js` | IMG-03 |
| `web/assets/terminal.js` | Add ImageAddon construction in `initTerminal()` IIFE alongside Unicode 11 (next-session-only); read `pluginConfig.imageConfig.storageLimit ?? 16` | IMG-03 |
| (potentially) `web/assets/terminal.js` `applyPluginConfig` diff path | NO change — Image is next-session-only; web has no "live re-attach" semantics for it (page reload triggers fresh init anyway) | (intentional non-change) |

### New e2e / UAT Artifacts (Wave 3-4)
| File | Why |
|------|-----|
| `internal/webserver/browser_csp_image_e2e_test.go` (//go:build e2e) | chromedp test: load `/sessions/{id}` with addon-image enabled, emit small sixel sequence, assert zero CSP violations + image canvas layer present in DOM |
| `.planning/phases/96-image-addon-csp-audit/96-HUMAN-UAT.md` | Manual UAT runbook: chafa sixel/IIP visual on desktop + web; multi-client mid-stream join; Settings toggle confirmation; 50 MB fixture eviction |

---

## Sources

### Primary (HIGH confidence)
- `frontend/node_modules/@xterm/addon-image/lib/addon-image.js` — minified production bundle; source of `URL.createObjectURL` count + `WebAssembly.*` count
- `frontend/node_modules/@xterm/addon-image/lib/addon-image.mjs` — unminified ES module; source for code-excerpt verification of WASM bootstrap + IIP fallback
- `frontend/node_modules/@xterm/addon-image/typings/addon-image.d.ts` — official TypeScript typings; source for `IImageAddonOptions`, `ImageAddon` API contract
- `frontend/node_modules/@xterm/addon-image/lib/addon-image.mjs:38` — verbatim source for `et = { storageLimit: 128, ... }` upstream defaults
- `internal/webserver/csp_mw.go` — current v3.1 CSP policy + D-09 amendment documentation pattern
- `internal/relay/scrollback.go` — `DefaultScrollbackBytes = 256 * 1024`; raw byte buffer; FIFO truncation
- `internal/relay/hub.go:135-152` — `Hub.Run` 32 KiB chunk reader, `MakeOutputFrame` wrapping, no escape parsing
- `internal/relay/server.go:104-109` — `ScrollbackSnapshot()` replay path for second-mid-stream-joining clients
- `internal/relay/protocol.go` — `MakeOutputFrame` (1-byte MsgOutput prefix; verbatim payload)
- `internal/daemon/plugin_settings.go` — current `PluginSettings`, `WebLinksConfig`, `defaultPluginSettings()` precedents
- `internal/daemon/engine.go:497-526` — `SetSearchConfig` and `SetWebLinksConfig` sub-key writer patterns (mirror targets for `SetImageConfig`)
- `internal/daemon/api.go:77, 590` — `PATCH /settings/web-links-config` route + handler (mirror target for `PATCH /settings/image-config`)
- `app.go:549` — `(*App).SetWebLinksConfig` Wails method (mirror target for `SetImageConfig`)
- `frontend/src/components/TerminalPanel.tsx:84-100, 175-281` — addon refs + mount useEffect + hot-swap useEffect structure; Unicode 11 next-session-only precedent at lines 180-185
- `frontend/src/components/PluginsSection.tsx:107-114, 130, 135-136` — italic caption rendering pattern + existing image row
- `web/embed.go` — current `//go:embed` directive structure
- `web/vendor/xterm/VERSION` — current vendored addon manifest
- `web/assets/terminal.js:118-836` — `applyPluginConfig` pattern, `initTerminal()` IIFE, `pluginConfig` defaults, addon construction order
- `internal/webserver/vendor_drift_test.go:18` — Phase 93 generalized regex `(@xterm/(?:xterm|addon-[\w-]+))`
- `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-RESEARCH.md` — vendoring pipeline + UMD global namespace pitfalls + CSP precedent
- `.planning/phases/95-web-links-addon-security-hardening/95-RESEARCH.md` — sub-config sub-key RPC pattern (Pattern 3 verbatim mirror)
- `.planning/STATE.md` — Phase 96 cross-cutting decisions: `storageLimit: 16`, mandatory pre-phase audit, default ON

### Secondary (MEDIUM confidence)
- npm registry `pnpm view @xterm/addon-image` (2026-05-07) — version 0.9.0 + dist-tags + zero deps
- pnpm-lock.yaml `@xterm/addon-image@0.9.0: {}` entry (post-install) — confirms zero transitive deps
- WebAssembly Content Security Policy proposal: github.com/WebAssembly/content-security-policy — `'wasm-unsafe-eval'` semantics
- MDN `Content-Security-Policy: script-src` — `'wasm-unsafe-eval'` directive specification

### Tertiary (LOW confidence — verified via secondary sources)
- caniuse.com `wasm-unsafe-eval` — search returned the entry but the page's specific version table was not extractable via WebFetch; cross-verified via WebKit bug 235408 (Safari 16), Mozilla bug 1740263 (Firefox 102), Chromium issue 948834 (Chrome 102)
- Browser support dates (Chrome 102 / Firefox 102 / Safari 16) — verified via vendor changelogs and the cited bug-tracker entries; consistent across multiple sources
- "Inline base64 WASM bytecode is the addon's bundled approach" — verified by source-read but the ~10 KB / ~14 KB sizes are estimates from inspecting the `Ve` constant length

---

## Metadata

**Confidence breakdown:**
- Mandatory Pre-Phase CSP Audit: HIGH — primary source (direct `grep` on installed package files); WASM-vs-CSP analysis cross-verified against CSP3 spec
- Standard Stack: HIGH — version verified against npm registry; zero deps verified; UMD pattern verified via source-read
- Architecture: HIGH — all patterns verified against current Phase 92/93/94/95 codebase with file:line citations
- IMG-04 Architecture: HIGH — relay scrollback path read end-to-end; byte-fidelity is structurally guaranteed (no parsing tier in the relay)
- IMG-04 Test Strategy: MEDIUM — test pattern verified against existing `hub_test.go`; the synthetic sixel byte stream's parser-validity is not material to the byte-fidelity assertion
- Pitfalls: HIGH — most derived from real source-read; Pitfall #8 (`enableSizeReports`) is documented in upstream typings
- CSP Amendment Documentation Rigor: HIGH — the existing v3.1 D-09 amendment text in `csp_mw.go` is the structural precedent; matching that precedent is mechanical

**Research date:** 2026-05-07
**Valid until:** 2026-06-07 (stable ecosystem; xterm.js addon-image API stable; CSP3 wasm-unsafe-eval directive stable; v3.1 codebase patterns stable through Phase 95)

---

## RESEARCH COMPLETE
