# Phase 96: Image Addon + CSP Audit — Pattern Map

**Mapped:** 2026-05-07
**Files analyzed:** 22 (2 new, 20 modified)
**Analogs found:** 22 / 22 (every file has a strong analog — Phase 96 is a structural mirror of Phase 95 plus the Phase 93 vendoring pipeline)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `web/vendor/xterm/addons/addon-image.js` | vendored-asset | static-file | `web/vendor/xterm/addons/addon-web-links.js` (Phase 95) | exact |
| `internal/relay/image_byte_fidelity_test.go` | test | byte-stream | `internal/relay/hub_test.go` `TestHubTwoSubscribersBothReceive` | exact |
| `frontend/package.json` | config | dependency-list | existing `@xterm/addon-web-links` entry (Phase 95) | exact |
| `frontend/pnpm-lock.yaml` | lockfile | dependency-list | (lockfile drift; mechanical) | n/a |
| `frontend/src/components/TerminalPanel.tsx` | component | event-driven | self — MOUNT useEffect, alongside Unicode 11 (lines 174-184) | exact |
| `frontend/src/components/PluginsSection.tsx` | component | request-response | self — `unicode11` `renderRow` 4th-arg italic caption (line 128-130) | exact |
| `frontend/src/components/__tests__/TerminalPanel.test.tsx` | test | static-source-scan | self — existing source-scan tests (`expect(raw).toContain(...)`) | exact |
| `frontend/src/components/__tests__/PluginsSection.test.tsx` | test | static-source-scan | self — existing italic-caption / row-order assertions | exact |
| `frontend/src/__tests__/App.plugin-event.test.tsx` | test | static-source-scan | self — `searchConfig` shape assertion (line 60-74) | exact |
| `frontend/src/wailsjs/go/models.ts` | type-defs | hand-edit | self — `WebLinksConfig` class block (lines 27-48, 78-83) | exact |
| `internal/daemon/plugin_settings.go` | model | persisted-config | self — `WebLinksConfig` struct (lines 24-45) + `defaultPluginSettings` (76-94) | exact |
| `internal/daemon/plugin_settings_test.go` | test | static | self — WebLinksConfig default-assertion block (lines 46-59) | exact |
| `internal/daemon/engine.go` | service | sub-key writer | self — `SetWebLinksConfig` (lines 508-535) | exact |
| `internal/daemon/api.go` | controller | request-response (PATCH) | self — `handleSetWebLinksConfig` (lines 583-616) + route registration line 77 | exact |
| `app.go` | controller | Wails-RPC | self — `(*App).SetWebLinksConfig` (lines 533-570) | exact |
| `internal/webserver/csp_mw.go` | middleware | request-response | self — current cspHeaders builder (lines 65-78) | exact (single-token append) |
| `internal/webserver/csp_mw_test.go` | test | request-response | self — `TestCSPHeaders_RequiredTokens` + `TestCSPHeaders_NoUnsafeTokens` (lines 47-115) | exact |
| `internal/webserver/vendor_drift_test.go` | test | static | self — min-count guard line 34 | exact (constant bump) |
| `web/embed.go` | config | static-embed | self — current `//go:embed` directive (lines 5-10) | exact |
| `web/vendor/xterm/VERSION` | manifest | static | self — existing manifest (7 entries) | exact (line append) |
| `web/terminal.html` | template | static-asset-list | self — existing addon `<script>` tags (lines 45-49) | exact |
| `web/assets/terminal.js` | controller | event-driven | self — Unicode 11 init in `initTerminal()` (lines 222-234), defaults block (lines 118-126) | exact |

---

## Pattern Assignments

### `internal/daemon/plugin_settings.go` (model — add `ImageConfig` struct + nested field)

**Analog:** self, `WebLinksConfig` block (lines 24-45) and `defaultPluginSettings()` (lines 76-94)

**Struct definition pattern** (lines 24-45):
```go
// WebLinksConfig persists per-plugin runtime configuration for the
// web-links toggle (Phase 95 LNK-02, LNK-03, LNK-05). Defaults are
// platform-correct + ALL confirmations ON (security-first posture
// per ROADMAP §"Phase 95 — v3.1-WS-Origin-allowlist rigor").
//
// JSON tags are camelCase to match daemonSettings vocabulary; field
// ordering matches the ## Files to Create / Modify table in
// .planning/phases/95-web-links-addon-security-hardening/95-RESEARCH.md
// §"Pattern 3: WebLinksConfig Persistence".
type WebLinksConfig struct {
    Modifier         string `json:"modifier"`
    ConfirmOSC8      bool   `json:"confirmOSC8"`
    ConfirmIDN       bool   `json:"confirmIDN"`
    ConfirmTyposquat bool   `json:"confirmTyposquat"`
}
```

**PluginSettings nesting pattern** (lines 55-66):
```go
type PluginSettings struct {
    WebGL          bool           `json:"webgl"`
    Unicode11      bool           `json:"unicode11"`
    Search         bool           `json:"search"`
    SearchConfig   SearchConfig   `json:"searchConfig"`
    WebLinks       bool           `json:"webLinks"`
    WebLinksConfig WebLinksConfig `json:"webLinksConfig"`
    Image          bool           `json:"image"`
    Serialize      bool           `json:"serialize"`
    Clipboard      bool           `json:"clipboard"`
    Progress       bool           `json:"progress"`
}
```

**Default-merge pattern** (lines 76-94):
```go
func defaultPluginSettings() PluginSettings {
    return PluginSettings{
        WebGL:        true,
        Unicode11:    true,
        Search:       true,
        SearchConfig: SearchConfig{Regex: false, CaseSensitive: false, WholeWord: false},
        WebLinks:     true,
        WebLinksConfig: WebLinksConfig{
            Modifier:         "platform",
            ConfirmOSC8:      true,
            ConfirmIDN:       true,
            ConfirmTyposquat: true,
        },
        Image:     true,
        Serialize: true,
        Clipboard: true,
        Progress:  false,
    }
}
```

**Mirror exactly:**
- `ImageConfig` struct comment header citing Phase 96 IMG-02 + 96-RESEARCH §"Pattern 2: ImageConfig Persistence"
- Single field `StorageLimit int \`json:"storageLimit"\`` (per RESEARCH `Important Defaults` block)
- NO `omitempty` on the JSON tag (Pitfall #14 — round-trip the user's saved choice)
- `ImageConfig ImageConfig \`json:"imageConfig"\`` field inserted in `PluginSettings` AFTER `Image bool`, BEFORE `Serialize bool` (preserves UI-SPEC ordering and matches the WebLinks/WebLinksConfig adjacency)
- `defaultPluginSettings()` returns `ImageConfig: ImageConfig{StorageLimit: 16}` per STATE.md/ROADMAP locked decision

**Adapt:** Just one numeric field, not four. No "Modifier" enum validation needed at struct level — the enum-like constraint lives in api.go (see below).

---

### `internal/daemon/plugin_settings_test.go` (test — assert `ImageConfig{StorageLimit: 16}` defaults)

**Analog:** self, `WebLinksConfig` default-assertion block (lines 46-59)

**Pattern** (lines 46-59):
```go
// Phase 95 LNK-02/LNK-03/LNK-05: WebLinksConfig defaults are
// platform-correct + ALL confirmations ON (security-first posture).
if got := s.WebLinksConfig.Modifier; got != "platform" {
    t.Errorf("WebLinksConfig.Modifier = %q, want \"platform\"", got)
}
if !s.WebLinksConfig.ConfirmOSC8 {
    t.Error("WebLinksConfig.ConfirmOSC8 should default true")
}
```

**Mirror exactly:** New assertion block at end of `TestDefaultPluginSettings`, citing Phase 96 IMG-02. Single comparison: `if got := s.ImageConfig.StorageLimit; got != 16 { t.Errorf(...) }`.

---

### `internal/daemon/engine.go` (service — add `SetImageConfig` sub-key writer)

**Analog:** self, `SetWebLinksConfig` (lines 508-535)

**Pattern** (lines 508-535):
```go
// SetWebLinksConfig updates and persists ONLY the WebLinksConfig sub-key
// of PluginSettings, leaving the rest of PluginSettings (WebGL, Unicode11,
// Search, SearchConfig, WebLinks boolean, Image, Serialize, Clipboard,
// Progress) untouched.
//
// Phase 95 LNK-05 / LNK-06 — mirrors Phase 94 Plan 07's SetSearchConfig
// sub-key writer verbatim. Concurrency / persistence contract is identical
// to SetPluginSettings: mutate under e.mu.Lock(), saveSettingsToDisk while
// held, capture and invoke listener after release.
func (e *SessionEngine) SetWebLinksConfig(cfg WebLinksConfig) {
    e.mu.Lock()
    e.pluginSettings.WebLinksConfig = cfg
    e.saveSettingsToDisk()
    listener := e.pluginSettingsListener
    e.mu.Unlock()
    if listener != nil {
        listener()
    }
}
```

**Mirror exactly:**
- Comment block listing every other sub-key left untouched (now including `WebLinksConfig` itself in the spared-list)
- `e.mu.Lock()` → mutate `e.pluginSettings.ImageConfig = cfg` → `saveSettingsToDisk()` → capture listener → `Unlock()` → invoke listener after release (concurrency contract is structurally important)
- Cite Phase 96 IMG-02 and call out next-session-only semantics — the listener still fires (web SSE consumers should receive the frame so dashboards reflect the new persisted config), but `applyPluginConfig` on the desktop side (TerminalPanel.tsx) intentionally does NOT include `imageConfig` in its hot-swap dep array.

**Adapt:** Param type is `ImageConfig`, not `WebLinksConfig`.

---

### `internal/daemon/api.go` (controller — add `PATCH /settings/image-config` handler)

**Analog:** self, `handleSetWebLinksConfig` (lines 583-616) + route registration (line 77)

**Route registration pattern** (line 77):
```go
a.mux.HandleFunc("PATCH /settings/web-links-config", a.handleSetWebLinksConfig)
```

**Handler pattern** (lines 583-616):
```go
func (a *API) handleSetWebLinksConfig(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 8192)

    var req WebLinksConfig
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()

    if err := dec.Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    // WR-02: validate Modifier against the four documented literals so a
    // typoed value or corrupted settings.json cannot silently disable the
    // entire feature ...
    switch req.Modifier {
    case "platform", "cmd", "ctrl", "none":
        // ok
    default:
        http.Error(w, "modifier must be one of: platform, cmd, ctrl, none", http.StatusBadRequest)
        return
    }
    a.engine.SetWebLinksConfig(req)
    w.WriteHeader(http.StatusNoContent)
}
```

**Mirror exactly:**
- Route insertion order: ADD `a.mux.HandleFunc("PATCH /settings/image-config", a.handleSetImageConfig)` immediately after the web-links-config route (line 77), preserving the alphabetic-ish PATCH grouping
- `http.MaxBytesReader(w, r.Body, 8192)` — same 8 KiB cap (overkill for a single-int payload but identical defense-in-depth)
- `dec.DisallowUnknownFields()` — required (defense-in-depth against forward-evolved clients)
- `http.StatusNoContent` (204) on success
- Error envelope `http.Error(w, "invalid request body", http.StatusBadRequest)` for decode failure

**Adapt — value validation:** Replace the `Modifier` switch with a numeric range check. Per RESEARCH (storage cap rationale + Pitfall #6: "Storage Cap Too High → Tab OOM"), validate `req.StorageLimit > 0 && req.StorageLimit <= 128`. Reject zero/negative ("storage-limit must be > 0") and reject > 128 MB ("storage-limit must be <= 128 MiB"). Do NOT accept the upstream addon-image default of 128 MB silently — the daemon's chosen default is 16 MB; 128 is the absolute upper bound.

---

### `app.go` (controller — add `(*App).SetImageConfig` Wails method)

**Analog:** self, `(*App).SetWebLinksConfig` (lines 533-570)

**Pattern** (lines 549-570):
```go
func (a *App) SetWebLinksConfig(cfg daemon.WebLinksConfig) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    if err := a.client.SetWebLinksConfig(cfg); err != nil {
        return err
    }
    // Re-fetch the full PluginSettings so the event payload matches the
    // SetPluginSettings event shape (App.tsx listener expects PluginSettings).
    full, err := a.client.GetPluginSettings()
    if err != nil {
        // Persistence succeeded but readback failed — synthesize a payload
        // from defaults + new WebLinksConfig so listeners still receive a
        // frame. The next GetPluginSettings call will reconcile.
        full = daemon.PluginSettings{WebLinksConfig: cfg}
    }
    // WR-05: guard against nil a.ctx (test harness or pre-startup RPC).
    if a.ctx != nil && a.ctx.Value("frontend") != nil {
        runtime.EventsEmit(a.ctx, "settings:plugins", full)
    }
    return nil
}
```

**Mirror exactly:**
- `if a.client == nil` daemon-not-connected guard
- Re-fetch full `PluginSettings` after sub-key write (App.tsx EventsOn handler expects the full shape)
- Synthesize-from-defaults fallback on readback failure (`full = daemon.PluginSettings{ImageConfig: cfg}`)
- WR-05 nil-ctx guard before `EventsEmit` (carry the lesson from Phase 95 code review forward)
- Event name stays `"settings:plugins"` — the same listener consumes WebLinks AND Image AND Search AND full SetPluginSettings frames

**Adapt:** Type is `daemon.ImageConfig`. The daemon-client method on `a.client` will be `SetImageConfig` (a corresponding addition to `internal/daemon/client.go` may be required — verify when implementing).

---

### `internal/webserver/csp_mw.go` (middleware — append `'wasm-unsafe-eval'` to script-src)

**Analog:** self, line 68 + package comment block (lines 1-44)

**Current pattern** (lines 65-78):
```go
b.Grow(256)
b.WriteString("default-src 'none'; ")
b.WriteString("script-src 'self'; ")
b.WriteString("style-src 'self' 'unsafe-inline'; ")
b.WriteString("connect-src 'self' ")
b.WriteString(wssOrigin)
b.WriteString("; ")
b.WriteString("img-src 'self' data:; ")
b.WriteString("font-src 'self'; ")
b.WriteString("base-uri 'none'; ")
b.WriteString("form-action 'self'; ")
b.WriteString("frame-ancestors 'none'")
```

**Existing v3.1 D-09 amendment documentation pattern** (lines 7-19):
```go
// Policy specification (D-09, amended 2026-04-22 after Phase 89 e2e finding):
//
//	default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline';
//	connect-src 'self' wss://<host>; img-src 'self' data:;
//	font-src 'self'; base-uri 'none'; form-action 'self';
//	frame-ancestors 'none'
//
// Amendment rationale: xterm.js injects <style> elements at runtime (cursor,
// selection, theme hooks) via document.createElement('style'). CSP3 classifies
// these as style-src-elem, which style-src 'self' blocks. Chromium e2e test
// TestBrowserCSP_TerminalNoViolations surfaced 12 violations on /sessions/{id}.
// User disposition: allow 'unsafe-inline' for style-src only. script-src
// remains strict ('self') — the Finding 4 CDN class of attack is unchanged.
```

**Mirror exactly:**
- Single-line code change on line 68: `b.WriteString("script-src 'self' 'wasm-unsafe-eval'; ")`
- Update the policy spec comment block on lines 7-13 to include the new directive
- Add an "Amendment 2 (2026-05-XX, after Phase 96 pre-phase audit)" block matching the v3.1 D-09 rigor: cite RESEARCH §"Mandatory Pre-Phase CSP Audit Finding 2", state the rationale (`@xterm/addon-image` SIXEL decoder bootstraps via `WebAssembly.instantiate`/`Module`/`Instance` and CSP3 §6.3 requires `'wasm-unsafe-eval'` to permit dynamic WASM module instantiation), explicitly contrast with `'unsafe-eval'` (which is BROADER and MUST NOT be used — defense-in-depth distinction), and cite browser support floors (Chrome 102 / Firefox 102 / Safari 16).

**Adapt:** Nothing structural. This is a single-token append plus a documentation-rigor matching exercise.

---

### `internal/webserver/csp_mw_test.go` (test — assert `'wasm-unsafe-eval'` present + `'unsafe-eval'` absent)

**Analog:** self, `TestCSPHeaders_RequiredTokens` (lines 47-72) + `TestCSPHeaders_NoUnsafeTokens` (lines 80-115)

**Required-token pattern** (lines 56-71):
```go
csp := rec.Header().Get("Content-Security-Policy")
required := []string{
    "default-src 'none'",
    "script-src 'self'",
    "style-src 'self'",
    "img-src 'self' data:",
    "font-src 'self'",
    "base-uri 'none'",
    "form-action 'self'",
    "frame-ancestors 'none'",
}
for _, token := range required {
    if !strings.Contains(csp, token) {
        t.Errorf("CSP missing required token %q (Phase 89 D-09): %s", token, csp)
    }
}
```

**Forbidden-token pattern** (lines 89-99):
```go
globallyForbidden := []string{
    "'unsafe-eval'",
    "'unsafe-hashes'",
}
for _, token := range globallyForbidden {
    if strings.Contains(csp, token) {
        t.Errorf("CSP must not contain %q anywhere (Phase 89 D-06): %s", token, csp)
    }
}
```

**Mirror exactly:**
- Add `"'wasm-unsafe-eval'"` to the `required` slice in `TestCSPHeaders_RequiredTokens` (or create a new sub-test `TestCSPHeaders_HasWasmUnsafeEval`).
- KEEP `"'unsafe-eval'"` in the `globallyForbidden` slice — `'wasm-unsafe-eval'` is a NARROWER directive distinct from `'unsafe-eval'`, and `strings.Contains(csp, "'unsafe-eval'")` returns true if `'wasm-unsafe-eval'` appears (because the former is a substring of the latter). Therefore the existing forbidden-token check MUST be tightened to use a token-aware comparison (split on whitespace inside the script-src clause and compare per-token equality), or use a regex like `\b'unsafe-eval'\b` that excludes the `wasm-` prefix. **This is a non-trivial adaptation** — flag it explicitly to the planner as a regression-defense item.

**Adapt — critical defense regression:** The existing `strings.Contains(csp, "'unsafe-eval'")` check will FALSELY match the new `'wasm-unsafe-eval'` token. Without tightening, the test passes vacuously and the regression guard is lost. Recommended: pull the script-src clause out the same way the existing `TestCSPHeaders_NoUnsafeTokens` already does for `'unsafe-inline'` (lines 101-114), tokenize on whitespace, and assert exact-token equality.

---

### `internal/webserver/vendor_drift_test.go` (test — bump min-count guard 7 → 8)

**Analog:** self, line 34

**Pattern** (lines 33-36):
```go
if len(pnpmVersions) < 7 {
    t.Fatalf("failed to parse at least 7 @xterm/* packages (xterm, addon-fit, addon-webgl, addon-unicode11, addon-clipboard, addon-search, addon-web-links) from pnpm-lock.yaml: found %v (Phase 95 SRC-95-06 — addon-web-links joined the manifest; T-95-06-01 mitigation)", pnpmVersions)
}
```

**Mirror exactly:**
- Change `< 7` → `< 8`
- Append `addon-image` to the inline package list in the error message
- Update the citation suffix: `(Phase 95 SRC-95-06 — addon-web-links joined the manifest; Phase 96 IMG-03 — addon-image joined the manifest; T-96-XX-XX mitigation)`

**Adapt:** Nothing. Mechanical bump.

---

### `web/embed.go` (config — append addon-image to `//go:embed` directive)

**Analog:** self, line 10

**Pattern** (lines 5-10):
```go
//go:embed dashboard.html terminal.html join.html
//go:embed assets/terminal.js assets/terminal.css
//go:embed assets/dashboard.js assets/dashboard.css
//go:embed assets/join.js assets/join.css
//go:embed vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION
//go:embed vendor/xterm/addons/addon-webgl.js vendor/xterm/addons/addon-unicode11.js vendor/xterm/addons/addon-clipboard.js vendor/xterm/addons/addon-search.js vendor/xterm/addons/addon-web-links.js
```

**Mirror exactly:** Append `vendor/xterm/addons/addon-image.js` to the existing line 10 (or split into two `//go:embed` lines if the ~115-char limit is hit — the directive accepts multiple lines and the dev who edits this is encouraged to wrap once line length crosses 120 cols).

---

### `web/vendor/xterm/VERSION` (manifest — append version line)

**Analog:** self, lines 1-7

**Pattern:**
```
@xterm/xterm@6.0.0
@xterm/addon-fit@0.11.0
@xterm/addon-webgl@0.19.0
@xterm/addon-unicode11@0.9.0
@xterm/addon-clipboard@0.2.0
@xterm/addon-search@0.16.0
@xterm/addon-web-links@0.12.0
```

**Mirror exactly:** Append `@xterm/addon-image@0.9.0` as the 8th line, preserving the trailing newline. The drift-test parses each non-comment, non-empty line as `@<scope>/<pkg>@<version>` (vendor_drift_test.go lines 50-55).

---

### `web/terminal.html` (template — add `<script>` tag for addon-image)

**Analog:** self, lines 45-49

**Pattern:**
```html
<script src="/assets/xterm/xterm.js"></script>
<script src="/assets/xterm/addon-fit.js"></script>
<script src="/assets/xterm/addons/addon-webgl.js"></script>
<script src="/assets/xterm/addons/addon-unicode11.js"></script>
<script src="/assets/xterm/addons/addon-clipboard.js"></script>
<script src="/assets/xterm/addons/addon-search.js"></script>
<script src="/assets/xterm/addons/addon-web-links.js"></script>
```

**Mirror exactly:** Insert `<script src="/assets/xterm/addons/addon-image.js"></script>` after the addon-web-links line (line 49), BEFORE the link-confirm-popover `<div>` block, BEFORE `<script src="/assets/terminal.js"></script>` (line 63). Order matters: terminal.js depends on the addon UMD globals being defined.

---

### `web/assets/terminal.js` (controller — add ImageAddon construction in `initTerminal()`)

**Analog:** self, Unicode 11 init block (lines 222-234) and pluginConfig defaults (lines 118-126)

**Defaults pattern** (lines 118-126):
```javascript
var pluginConfig = {
    webgl: true, unicode11: true, clipboard: true,
    search: true, webLinks: true, image: true,
    serialize: true, progress: false,
    searchConfig: { regex: false, caseSensitive: false, wholeWord: false }
};
```

**Next-session-only construction pattern** (lines 222-234):
```javascript
// Phase 93 U11-02: server-shared Unicode 11. Applied at construction
// ONLY — server-shared semantics mean a running session must NOT switch
// its width tables mid-buffer (would corrupt scrollback on existing
// characters). The next page load picks up any change automatically.
if (pluginConfig.unicode11) {
    try {
        var u11 = new Unicode11Addon.Unicode11Addon();
        term.loadAddon(u11);
        term.unicode.activeVersion = '11';
    } catch (e) { /* addon UMD may not be present — silent */ }
}
```

**Mirror exactly:**
- Add `imageConfig: { storageLimit: 16 }` to the defaults object on line 118-126 (parallel to `searchConfig`)
- Add a NEXT-SESSION-ONLY image construction block right after Unicode 11 (lines 222-234), citing Phase 96 IMG-03
- `try { ... } catch (e) { /* addon UMD may not be present — silent */ }` wrapping is mandatory (Pitfall #4 — UMD global namespace failure is silent and per-session)
- Read `pluginConfig.imageConfig.storageLimit ?? 16` for defense against missing key (web is a downstream consumer; daemon may push frames before the schema-merge has happened on this page load)
- UMD global namespace pitfall #4: `new ImageAddon.ImageAddon({ storageLimit: ... })` (NOT `new ImageAddon(...)`) — verify the actual UMD shape during implementation by inspecting the vendored file

**Adapt — `applyPluginConfig` diff path is INTENTIONALLY UNCHANGED.** Image is next-session-only on the web side too (page reload triggers fresh init). RESEARCH explicitly calls this out as `(intentional non-change)` on line 1281 of 96-RESEARCH.md.

---

### `frontend/src/wailsjs/go/models.ts` (type-defs — add `ImageConfig` class)

**Analog:** self, `WebLinksConfig` block (lines 27-48) + `PluginSettings` constructor nesting (lines 78-83)

**Class pattern** (lines 27-48):
```typescript
// Phase 95 LNK-02/LNK-03/LNK-05: nested per-plugin runtime config
// for the web-links toggle. Defaults: modifier="platform" +
// ALL confirmations true (security-first; daemon owns defaults via
// internal/daemon/plugin_settings.go defaultPluginSettings()).
export class WebLinksConfig {
    modifier: string;
    confirmOSC8: boolean;
    confirmIDN: boolean;
    confirmTyposquat: boolean;

    static createFrom(source: any = {}) {
        return new WebLinksConfig(source);
    }

    constructor(source: any = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.modifier = source["modifier"];
        this.confirmOSC8 = source["confirmOSC8"];
        this.confirmIDN = source["confirmIDN"];
        this.confirmTyposquat = source["confirmTyposquat"];
    }
}
```

**PluginSettings nesting pattern** (lines 56, 78-83):
```typescript
webLinksConfig: WebLinksConfig;
// ...
this.webLinksConfig = source["webLinksConfig"]
    ? new WebLinksConfig(source["webLinksConfig"])
    : new WebLinksConfig();
```

**Mirror exactly:**
- Add `export class ImageConfig` BETWEEN `WebLinksConfig` (line 48) and `PluginSettings` (line 50), with one numeric field `storageLimit: number`
- Add `imageConfig: ImageConfig;` field to `PluginSettings` AFTER `image: boolean` (preserve the Go struct ordering — line 84 in models.ts)
- Add the inline `new ImageConfig(source["imageConfig"])` construction in the constructor body after `this.image = source["image"];`
- Cite Phase 96 IMG-02 in the class doc-comment; mirror the daemon-owns-defaults caveat
- HAND-EDIT preservation header: this file's existing comment (lines 1-6) already documents the Phase 92 pin pattern — do not touch it

**Adapt:** Single field `storageLimit: number` (default 16). Constructor body is a one-liner: `this.storageLimit = source["storageLimit"];`.

---

### `frontend/src/components/TerminalPanel.tsx` (component — wire ImageAddon in MOUNT useEffect)

**Analog:** self, Unicode 11 mount block (lines 174-184)

**Pattern** (lines 174-184):
```typescript
// Phase 93 U11-01: Unicode 11 honors next-session-only semantics. The
// pluginConfig?.unicode11 flag is read at session init; toggling it in
// Settings does NOT mutate already-open terminals (UI-SPEC § Interaction
// Contract — italic caption "Applies to new sessions you create."
// explains this affordance to users). Default true if pluginConfig
// hasn't loaded yet (preserves Phase 92 always-on behavior).
if (pluginConfig?.unicode11 !== false) {
    const unicode11 = new Unicode11Addon()
    term.loadAddon(unicode11)
    term.unicode.activeVersion = '11'   // TERM-03: emoji + CJK + box-drawing
}
```

**Cleanup pattern** (lines 235-242 — for hot-swap addons; image follows mount-cleanup pattern instead, see below):
```typescript
if (webglAddonRef.current) {
    webglAddonRef.current.dispose()
    webglAddonRef.current = null
}
```

**Mirror exactly:**
- Import: `import { ImageAddon } from '@xterm/addon-image'` alongside other addon imports (lines 4-9)
- Add `const imageAddonRef = useRef<ImageAddon | null>(null)` alongside other addon refs (lines 84-100, group with the HOT-SWAP refs visually but the instance is mount-only)
- Construction block: place IMMEDIATELY AFTER the Unicode 11 block (line 184), gated by `if (pluginConfig?.image !== false)`, reading `storageLimit` from `pluginConfig?.imageConfig?.storageLimit ?? 16`
- Constructor: `new ImageAddon({ storageLimit: <value>, enableSizeReports: false })` — `enableSizeReports: false` is REQUIRED per Pitfall #8 (CSI Response Pollution risk if true)
- Cleanup: dispose `imageAddonRef.current` in the mount useEffect's RETURN cleanup (lines 227-276), alongside `webLinksAddonRef.current.dispose()` (line 257). Wrap in `try { ... } catch { /* ignore */ }` — match the Phase 95 web-links cleanup style.

**Adapt — critical: Pitfall #1.** The image addon belongs in the MOUNT useEffect alongside Unicode 11 (lines 158-280), NOT in the hot-swap useEffect. RESEARCH §"Pitfall 1: Wrong useEffect — Image in Hot-Swap Instead of Mount" makes this point explicit. Toggling Image in Settings does NOT re-attach the addon on already-open terminals. The PluginsSection italic caption ("Applies to new sessions you create.") is the user-facing affordance for this constraint.

---

### `frontend/src/components/PluginsSection.tsx` (component — add italic caption to image renderRow)

**Analog:** self, `unicode11` renderRow (lines 128-130)

**Pattern** (lines 128-130, 135-136):
```typescript
{renderRow('unicode11', 'Unicode 11 widths',
    'Correct cell widths for emoji and CJK characters using the Unicode 11 width tables.',
    'Applies to new sessions you create.')}
// ...
{renderRow('image', 'Inline images',
    'Render images sent via sixel or the iTerm2 inline image protocol directly inside the terminal.')}
```

**`renderRow` signature** (lines 100-114):
```typescript
{caption && (
    <p className="settings-panel__description settings-panel__description--italic">
        {caption}
    </p>
)}
```

**Mirror exactly:**
- Add the literal string `'Applies to new sessions you create.'` as the 4th argument to the existing image `renderRow` call on line 135-136
- Identical wording to the unicode11 caption (line 130) — same affordance, same string, intentional consistency
- The `renderRow` body already supports the 4th argument (lines 107-111) — no signature change needed

---

### `frontend/src/components/__tests__/TerminalPanel.test.tsx` (test — assert ImageAddon construction)

**Analog:** self, existing source-scan tests (lines 11-22 and others)

**Pattern** (lines 11-22):
```typescript
describe('TerminalPanel', () => {
    it('exports TerminalPanel function', () => {
        // ...
    })
    it('source contains flex:1 and minHeight:0 inline styles', () => {
        // ...
    })
})
```

**Mirror exactly:** Add a new `describe('IMG-01/IMG-02 ImageAddon construction', () => { ... })` block. Source-scan assertions:
1. `expect(raw).toContain("import { ImageAddon } from '@xterm/addon-image'")`
2. `expect(raw).toContain('imageAddonRef')`
3. `expect(raw).toContain('new ImageAddon(')`
4. `expect(raw).toContain('enableSizeReports: false')` — Pitfall #8 regression guard
5. `expect(raw).toContain('imageConfig?.storageLimit ?? 16')` (or fragment — exact text depends on implementation)
6. Next-session-only invariant: assert that the ImageAddon construction text appears WITHIN the mount useEffect range AND does NOT appear in the hot-swap useEffect range — practical implementation: split on the `useEffect` markers, scan only the mount section for `ImageAddon`. Mirror the testing style already used for unicode11 in this file if such a test exists; otherwise add a comment-anchor regex check.

**Adapt:** Tests are source-scans (per project convention shown in PluginsSection.test.tsx and elsewhere) — no React Testing Library render needed.

---

### `frontend/src/components/__tests__/PluginsSection.test.tsx` (test — assert italic caption under Image row)

**Analog:** self, existing row-order/description assertions (lines 4-35)

**Pattern** (lines 4-13):
```typescript
describe('PUI-01: 8 toggle rows in UI-SPEC order', () => {
    it('contains all 8 row labels', () => {
        expect(raw).toContain('WebGL renderer')
        expect(raw).toContain('Unicode 11 widths')
        // ...
    })
})
```

**Mirror exactly:** Add a new `it('IMG-01: Image row carries next-session-only italic caption', () => { ... })` test. Source-scan assertion: assert that the substring `'Applies to new sessions you create.'` appears AT LEAST TWICE in the raw source (once for unicode11, once for image). For tighter regression guarding, slice the source at the `'Inline images'` index and assert the next-session-only caption text appears within ~200 characters AFTER that point (proves it's in the image renderRow call, not the unicode11 one).

---

### `frontend/src/__tests__/App.plugin-event.test.tsx` (test — extend PluginSettings shape)

**Analog:** self, `searchConfig` shape assertion (lines 59-74)

**Pattern** (lines 59-74):
```typescript
it('daemon.PluginSettings preserves nested searchConfig as a SearchConfig instance', () => {
    const ps = new daemon.PluginSettings({
        // ...
        searchConfig: { regex: true, caseSensitive: false, wholeWord: true },
        // ...
    })
    expect(ps.searchConfig).toBeInstanceOf(daemon.SearchConfig)
    expect(ps.searchConfig.regex).toBe(true)
    expect(ps.searchConfig.caseSensitive).toBe(false)
    expect(ps.searchConfig.wholeWord).toBe(true)
})
```

**Mirror exactly:** Extend the test fixture object passed to `new daemon.PluginSettings({ ... })` to include `imageConfig: { storageLimit: 16 }`. Add a parallel assertion: `expect(ps.imageConfig).toBeInstanceOf(daemon.ImageConfig); expect(ps.imageConfig.storageLimit).toBe(16);`. Also extend the JSON round-trip test (lines 77-91) similarly.

---

### `frontend/package.json` (config — add `@xterm/addon-image: ^0.9.0`)

**Analog:** self, existing `@xterm/addon-web-links` entry

**Mirror exactly:** Add `"@xterm/addon-image": "^0.9.0"` in alphabetical order under `dependencies` (NOT `devDependencies` — promoted from research-time devDep install). The exact key position: after `@xterm/addon-fit` and before `@xterm/addon-search` (alphabetical: fit < image < search < unicode11 < web-links < webgl).

---

### `frontend/pnpm-lock.yaml` (lockfile — auto-regenerated)

**Mechanical:** Run `pnpm install` after the package.json edit; the lockfile updates itself. Verify the resulting `@xterm/addon-image@0.9.0: {}` entry has zero transitive deps (per RESEARCH `Sources` line 1319).

---

### `web/vendor/xterm/addons/addon-image.js` (NEW FILE — vendored UMD bundle)

**Analog:** `web/vendor/xterm/addons/addon-web-links.js` (Phase 95 vendoring) and the broader Phase 93 vendoring pipeline

**Mechanical recipe** (Phase 93 verbatim — RESEARCH §"Pattern 4: Web Vendoring (Phase 93/94/95 Pattern Verbatim)"):
1. `pnpm install @xterm/addon-image@^0.9.0` (in `frontend/`)
2. `cp frontend/node_modules/@xterm/addon-image/lib/addon-image.js web/vendor/xterm/addons/addon-image.js`
3. Verify same-origin (no remote `<script>` references — required for CSP `script-src 'self'`)
4. Confirm UMD global is exposed: `grep ImageAddon web/vendor/xterm/addons/addon-image.js | head -3` should show the UMD `(function (global, factory) { ... })` preamble assigning `global.ImageAddon`

---

### `internal/relay/image_byte_fidelity_test.go` (NEW FILE — IMG-04 multi-client byte-fidelity)

**Analog:** `internal/relay/hub_test.go` `TestHubTwoSubscribersBothReceive` (lines 55-89)

**Pattern** (lines 55-89):
```go
func TestHubTwoSubscribersBothReceive(t *testing.T) {
    hub, ptyWriter := makeTestHub(t)

    sub1 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
    sub2 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
    hub.Subscribe(sub1)
    hub.Subscribe(sub2)
    go hub.Run()

    data := []byte("broadcast data")
    if _, err := ptyWriter.Write(data); err != nil {
        t.Fatalf("write failed: %v", err)
    }

    timeout := time.After(2 * time.Second)
    var got1, got2 []byte
    for got1 == nil || got2 == nil {
        select {
        case f := <-sub1.Msgs:
            got1 = f
        // ...
        }
    }

    // Both should have received the same frame
    if string(got1) != string(got2) {
        t.Errorf("subscriber frames differ: sub1=%v sub2=%v", got1, got2)
    }
}
```

**Mirror exactly:**
- Same `makeTestHub(t)` helper, same `Subscriber` shape, same channel-based assertion structure
- Cite Phase 96 IMG-04 in the test docstring; explain that the relay is a byte-buffer (not a parser tier — RESEARCH "IMG-04 Architecture: HIGH — relay scrollback path read end-to-end; byte-fidelity is structurally guaranteed (no parsing tier in the relay)")
- Synthetic sixel-shaped byte payload: a small but recognizable sequence such as `\x1bPq#0;2;100;0;0#1~~~~\x1b\\` (parser-validity is NOT material — the test asserts byte-for-byte equality on both subscribers, NOT that the bytes are parsed correctly anywhere)
- Two subscribers, both receive identical frames after the same write
- Optionally include a third assertion that `ScrollbackSnapshot()` (replay path for late-joining clients — RESEARCH `internal/relay/server.go:104-109`) returns the same bytes verbatim

**Adapt:** The test should call out that the 256 KiB scrollback truncation cap (`DefaultScrollbackBytes = 256 * 1024` in scrollback.go line 6) means second-mid-stream-joining clients may receive a truncated payload — but this is by design (Pitfall #3) and is the documented v3.2 limitation per Assumption A4. Test only the byte-fidelity, NOT no-truncation.

---

## Shared Patterns

### Sub-key writer concurrency contract
**Source:** `internal/daemon/engine.go` `SetWebLinksConfig` (lines 526-535)
**Apply to:** `SetImageConfig` engine writer
```go
e.mu.Lock()
e.pluginSettings.WebLinksConfig = cfg
e.saveSettingsToDisk()
listener := e.pluginSettingsListener
e.mu.Unlock()
if listener != nil {
    listener()
}
```
**Why exactly:** Mutate under lock, persist while held, capture listener BEFORE release, invoke listener AFTER release. Reordering this risks deadlock (listener might re-enter the engine) or lost updates (listener fires before persist).

---

### Wails-method daemon-fanout pattern
**Source:** `app.go` `(*App).SetWebLinksConfig` (lines 549-570)
**Apply to:** `(*App).SetImageConfig`
```go
if a.client == nil { return fmt.Errorf("daemon not connected") }
if err := a.client.SetX(cfg); err != nil { return err }
full, err := a.client.GetPluginSettings()
if err != nil { full = daemon.PluginSettings{XConfig: cfg} }
if a.ctx != nil && a.ctx.Value("frontend") != nil {
    runtime.EventsEmit(a.ctx, "settings:plugins", full)
}
return nil
```
**Why exactly:** App.tsx `EventsOn('settings:plugins')` expects `PluginSettings`. Re-fetch full state after sub-key write so the event payload is the post-write truth. WR-05 nil-ctx guard (Phase 95 code-review fix) is mandatory.

---

### PATCH handler defense-in-depth
**Source:** `internal/daemon/api.go` `handleSetWebLinksConfig` (lines 590-616)
**Apply to:** `handleSetImageConfig` and any future PATCH `/settings/*-config` route
```go
r.Body = http.MaxBytesReader(w, r.Body, 8192)
var req XConfig
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
if err := dec.Decode(&req); err != nil {
    http.Error(w, "invalid request body", http.StatusBadRequest)
    return
}
// ... value validation ...
a.engine.SetXConfig(req)
w.WriteHeader(http.StatusNoContent)
```
**Why exactly:** `MaxBytesReader` caps payload size at 8 KiB (overkill for small configs but uniform across PATCH routes). `DisallowUnknownFields` catches forward-evolved client schemas before they silently corrupt state. `204 No Content` is the chosen success status for sub-key PATCH.

---

### Wailsjs nested config typing
**Source:** `frontend/src/wailsjs/go/models.ts` `WebLinksConfig` block + `PluginSettings` constructor inline conversion (lines 27-48, 78-83)
**Apply to:** `ImageConfig` class + `imageConfig` field in PluginSettings
**Why exactly:** This file is a HAND-EDITED PIN (Phase 92 pattern documented in the file's leading comment). Any nested config must avoid `convertValues` because a `convertValues` member would surface as `keyof PluginSettings` and break PluginsSection's `keyof PluginSettings` toggle iteration (line 73-76 of models.ts documents this).

---

### Vendoring + drift-test triplet (Phase 93/94/95 verbatim)
**Sources:** `web/embed.go`, `web/vendor/xterm/VERSION`, `internal/webserver/vendor_drift_test.go`, `web/terminal.html`
**Apply to:** Any new `@xterm/addon-*` vendored package
**Pattern:** Every addon-vendor introduction touches FOUR files in lockstep:
1. `web/vendor/xterm/addons/addon-X.js` — the file itself (copied from `frontend/node_modules`)
2. `web/embed.go` — `//go:embed` directive includes the new file
3. `web/vendor/xterm/VERSION` — `@xterm/addon-X@<version>` line appended
4. `web/terminal.html` — `<script src="/assets/xterm/addons/addon-X.js"></script>` tag added before terminal.js
5. `internal/webserver/vendor_drift_test.go` — bump min-count guard
**Skipping any one of these breaks vendor_drift_test, breaks `script-src 'self'` CSP, or breaks the UMD load in the browser.**

---

### CSP amendment documentation rigor (v3.1 D-09 precedent)
**Source:** `internal/webserver/csp_mw.go` lines 7-19 (the v3.1 D-09 amendment block)
**Apply to:** Phase 96's Amendment 2 (`'wasm-unsafe-eval'`) documentation
**Why exactly:** The existing D-09 amendment block has a specific rigor — explicit policy spec line, "Amendment rationale" prose paragraph citing the discovery vector (e2e test name), the user-disposition statement, and the contrast-with-broader-directive note. Phase 96's amendment must match this rigor: cite the pre-phase audit (RESEARCH §"Mandatory Pre-Phase CSP Audit Finding 2"), explain WASM bootstrap, contrast `'wasm-unsafe-eval'` (narrow, just WASM module instantiation) vs `'unsafe-eval'` (broad, includes JS `eval()`), cite browser support floors.

---

### Source-scan test convention
**Source:** `frontend/src/components/__tests__/PluginsSection.test.tsx` lines 4-35; `frontend/src/components/__tests__/TerminalPanel.test.tsx` lines 11-22
**Apply to:** All new vitest assertions for TerminalPanel + PluginsSection in Phase 96
**Why exactly:** Tests in this codebase favor `expect(raw).toContain(...)` source-scan assertions over React Testing Library renders, because the components depend on browser APIs (Terminal, WebGL, Wails runtime) that vitest cannot stand up cleanly. New IMG-01/02 tests must follow this convention.

---

## No Analog Found

**None.** Every Phase 96 file has a clear analog already in the codebase. Phase 96 is structurally a Phase-95 mirror plus the Phase-93 vendoring pipeline plus a single CSP amendment.

The only file with a non-trivial adaptation flag is `internal/webserver/csp_mw_test.go` — the existing `'unsafe-eval'` substring check will FALSELY match the new `'wasm-unsafe-eval'` token and must be tightened to a token-aware comparison. This is documented in detail in that file's section above.

---

## Metadata

**Analog search scope:**
- `internal/daemon/` (engine.go, api.go, plugin_settings.go, plugin_settings_test.go)
- `app.go` (root)
- `internal/webserver/` (csp_mw.go, csp_mw_test.go, vendor_drift_test.go)
- `internal/relay/` (scrollback.go, hub_test.go, scrollback_test.go)
- `web/` (embed.go, terminal.html, assets/terminal.js, vendor/xterm/VERSION)
- `frontend/src/components/` (TerminalPanel.tsx, PluginsSection.tsx, __tests__/)
- `frontend/src/wailsjs/go/models.ts`
- `frontend/src/__tests__/App.plugin-event.test.tsx`

**Files scanned (non-exhaustive):** 17 source files + 5 test files

**Pattern extraction date:** 2026-05-07

**Confidence:** HIGH — every analog was read directly with file:line citations; every "mirror exactly" instruction maps to a code-block excerpt above; every "adapt" callout names a specific divergence with rationale.
