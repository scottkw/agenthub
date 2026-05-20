# Feature Research

**Domain:** xterm.js plugin/addon suite for AI-coding-CLI terminal app (AgentHub v3.2)
**Researched:** 2026-05-03
**Confidence:** HIGH — official xterm.js README + typings on master + npm metadata; all addon names/versions verified against the official `xtermjs/xterm.js` repo and `@xterm/*` npm scope. Competitor comparisons MEDIUM (public docs + comparison articles).

> Scope reminder: AgentHub v3.2 (Issue #36) — bundle a curated set of xterm.js addons + Settings UI to enable/disable each, with per-addon config where it matters. Existing features (138 themes, web serving, multi-client fan-out, BannerStack, three-state Save, vendored xterm assets, strict CSP) are not re-researched — only their *interactions* with the new addons are called out.

---

## Feature Landscape

### Table Stakes (Users Expect These)

A 2026-era xterm.js-based terminal is expected to ship these. Missing any makes AgentHub feel like a toy compared to VS Code's integrated terminal, Hyper, Tabby, or iTerm2.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **WebGL renderer** (`@xterm/addon-webgl`) | VS Code's terminal defaults to WebGL ("auto" GPU acceleration); Hyper/Tabby ship it; without it scrolling/painting feels visibly laggy at 80x24+. xterm.js DOM renderer is the slow fallback. | M | xterm.js core ships only the DOM renderer by default. Loading WebGL is one `terminal.loadAddon(new WebglAddon())` line. **Must** subscribe to `webglcontextlost` / `onContextLoss` and dispose+fall back to DOM — browsers drop the GL context on OOM, sleep/wake, GPU driver hiccup. AgentHub already calls `clearTextureAtlas()` + `refresh()` on theme change for WebGL (THM-02 / Phase 65). |
| **Find/search in scrollback** (`@xterm/addon-search`) | Cmd-F / Ctrl-F is the universal "find in this view" muscle memory. AI sessions scroll thousands of lines; users *will* try Cmd-F immediately. | M (UI) / S (wiring) | API exposes `findNext(term, opts)` / `findPrevious(term, opts)` with `regex`, `caseSensitive`, `wholeWord`, `incremental`, `decorations` options. AgentHub must build the find bar UI itself — the addon does not draw one. Standard pattern: floating bar top-right of active terminal, `Esc` to close, `Enter`/`Shift+Enter` for next/prev, three toggle icons (Aa, .*, ⌊ab⌋). |
| **Clickable web links** (`@xterm/addon-web-links`) | When an AI agent prints a URL (PR link, doc link, error trace), users expect Cmd-click to open. iTerm2, VS Code terminal, Hyper all do this. | S | Default detection regex covers `http(s)://…`. **Default activation is plain click** — for AgentHub on macOS the right default is Cmd-click (Ctrl-click on Linux/Win) to avoid mis-clicks during selection. Modifier check goes in the `activate` callback. |
| **Correct Unicode 11+ widths** (`@xterm/addon-unicode11`) | CJK users, emoji ZWJ sequences (👨‍👩‍👧, 🏳️‍🌈, skin-tone modifiers), and Powerline glyphs all break alignment without it. AI agents emit emoji + box-drawing constantly (Claude Code TUI, Codex spinners). | XS | One addon, one `terminal.unicode.activeVersion = '11'` line. xterm.js core defaults to Unicode 6 widths (1991-era tables). Bundling this is essentially mandatory for any modern terminal. Caveat: tables are reportedly closer to Unicode 12; `@xterm/addon-unicode-graphemes` (experimental) is the future path. |
| **Settings toggle for each plugin** | Users expect "I can turn this off" — VS Code has `terminal.integrated.gpuAcceleration` (auto/on/off/canvas), iTerm2 has per-feature checkboxes throughout Preferences. Plugins that misbehave on a given GPU/driver must be disablable without code edits. | S | AgentHub already has `settings.json` daemon persistence (Phase 79) and three-state Save button. Add a "Plugins" section with one row per addon (label, short description, toggle). |
| **"Applies to new sessions only" indicator** | xterm.js Terminal instances bind addons at construction; many addons (image, unicode11, webgl on some setups) cannot be hot-swapped on a live terminal without disposing it. Users will toggle and wonder why nothing changed. | S | VS Code uses an inline italic note ("Restart the terminal for this to take effect"). Pattern: small grey italic text below the toggle, only shown for plugins that can't hot-swap. AgentHub's terminal lifecycle is tab-scoped, so "applies to new tabs/sessions" is the accurate copy. |

### Differentiators (Competitive Advantage)

Where AgentHub can lean into its specific audience (developers running AI coding CLIs over Tailscale) and ship something that feels more thoughtful than VS Code or Hyper.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Inline images** (`@xterm/addon-image`) | AI coding CLIs increasingly emit charts/diagrams; `chafa --format=iterm2 chart.png` Just Works. AgentHub serving over Tailscale means a remote tablet user sees images inline — no separate viewer. | L | Supports both **sixel** (DEC, what `img2sixel`/older tools emit) and **iTerm2 IIP** (modern, what `imgcat`/`chafa --format=iterm2`/`viu` emit). Default `iipSupport: true`. **Storage cost:** FIFO cache, default `storageLimit: 128 MB` per terminal — must drop default since users keep many tabs open. **Pixel cost:** `pixelLimit` defaults to 16M pixels (~4096×4096). **CSP interaction:** images decode to canvas/WebGL inside xterm — does NOT need `img-src` directives because no `<img>` tags are created; the existing `script-src 'self'` + `connect-src 'self' wss://…` CSP from Phase 89 is sufficient *if* the addon's worker is bundled same-origin (verify). |
| **OSC 9;4 progress** (`@xterm/addon-progress`) | AI CLIs and tools they invoke (`pip`, `npm`, `cargo`, `mise`, model loaders) emit OSC 9;4. Surfacing this in AgentHub's tab bar (subtle progress underline) and tray (quartile glyph) is a unique fit. Ghostty has the GUI version; no other tabbed terminal app surfaces it per-tab. | M | Addon is small — fires events on progress state/value. AgentHub-specific opportunity: route the event into the existing tab bar (subtle progress bar under tab title) and tray (1/4/2/4/3/4 quartile glyph). Real differentiator vs VS Code/iTerm2. |
| **Serialize for "Copy session as text/HTML"** (`@xterm/addon-serialize`) | AI debugging workflow: "send me what your terminal looks like right now." Today users screenshot. With `serialize → HTML` plus a "Copy as HTML" / "Save session..." menu item, they get pixel-perfect sharable output (colors, cursor pos, scrollback). | S | API: `serialize(opts?)` returns VT-sequence string (replay into a fresh terminal); `serializeAsHTML()` returns standalone HTML. **What it does NOT preserve:** PTY state, env, running processes, alt-screen contents (with caveats). It is a *visual* snapshot, not session migration. Frame this honestly. |
| **Find bar matching AgentHub's BannerStack visual language** | Phase 81 established a stacking-banner visual style with 200ms exit animations. The find bar should feel like a sibling — same border-radius, typography, dismiss animation. | S (CSS) | Don't ship a generic find bar — match existing TokyoNight + BannerStack vocabulary. Highlight color must come from the active xterm-theme palette so it's visible across all 138 themes (use `theme.selectionBackground` consistently). |
| **Per-tab "Find in this session" + (P3) "Find in all sessions"** | AI agents run for hours and produce content across many tabs. "Find in all" — iterating each terminal's `searchAddon.findNext` and aggregating — is something no other terminal app ships. | M | P3/future. Each tab carries its own SearchAddon; a global Cmd-Shift-F opens a panel that runs the term across every open terminal and lists matches with tab name + line number, click to jump. |
| **Vendored addon assets matching v3.1 CSP discipline** | Phase 89 established `web/vendor/xterm/` and `script-src 'self'`. Vendoring each addon (no CDN, no dynamic imports from outside origin) is what makes the entire suite *shippable* under that CSP. | S | All `@xterm/addon-*` packages are pure JS with zero runtime fetches *except* `@xterm/addon-image` (loads worker code) and `@xterm/addon-ligatures` (Node-only — won't work in web embed). Image addon must be bundled with its worker as a same-origin asset. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Open arbitrary `file://`, `vscode://`, `mailto:`, custom protocols on click** | "I want my error stack traces to open in $EDITOR." | Tailnet-shared sessions = remote tailnet viewer can be tricked into clicking a link that fires a custom-protocol handler on their machine. Custom URL handlers also drift between OSes. | Default: **HTTP(S) only**. Add a *separate* "Custom protocols (advanced)" textbox in Plugin settings, off by default, with warning text. Never enable `file://` by default. |
| **Plain-click activation for links (no modifier)** | "Cmd-click is annoying." | Drag-to-select is the #1 terminal interaction. Plain-click activation collides with selection start. iTerm2, Terminal.app, VS Code all require Cmd/Ctrl. | Cmd on macOS, Ctrl on Linux/Windows — match the platform. Optional hidden setting for users who insist, but default to modifier-required. |
| **Always-on serialization / autosave-to-disk on every keystroke** | "I want to never lose a session." | Disk I/O on every output frame is wasteful; the buffer can be 100KB+ × N sessions. Serialize is meant to be on-demand. AgentHub's PTY persistence + scrollback replay (MC-04) already covers "don't lose output." | Manual `serialize()` triggered by user action only ("Export buffer"), or one-shot on graceful tab close → store last-N-bytes in `settings.json`. Don't run on a timer. |
| **Bundle `@xterm/addon-ligatures`** | Cool feature on paper (Fira Code arrows). | The addon **requires Node.js APIs** to read the font file from disk. AgentHub's web-served terminal (Tailscale browser viewer) has no Node — would silently no-op for half of users. Inconsistent rendering between desktop and web is worse than no ligatures. | Skip for v3.2. Revisit only if a pure-browser ligature addon emerges. Document the decision. |
| **Custom theme editor in addition to plugin toggles** | "Plugins are settings, themes are settings, why not let me edit themes too?" | Already explicitly Out of Scope in PROJECT.md ("138 curated xterm-theme schemes cover the need; custom editing adds complexity"). | Defer indefinitely. Don't let plugin work re-open this question. |
| **Per-tab WebGL on/off** | "WebGL works on tab 1 but flickers on tab 2 because of an extension." | Renderer choice is a global GPU/driver characteristic, not per-tab. Per-tab adds 4× state surface and breaks the "one renderer" intuition. | Global setting only. If WebGL fails on context loss, fall back globally and surface a one-shot banner: "GPU rendering unavailable, switched to software." |
| **Clicking inline images opens a viewer modal / Copy/Save image** | "It's an image, I want to see it bigger / save it." | Image addon doesn't expose image extraction. Re-implementing right-click → Save / Copy on top of canvas pixels is real engineering for marginal benefit. AI sessions usually emit small previews, not full screenshots. | v3.2: ship inline images read-only (no copy/save gestures). Track a follow-up issue if users ask. |
| **Real-time collaborative cursor like Warp** | Warp showed cursors-in-terminal is possible. | AgentHub's multi-client model is *fan-out of the same PTY*, not collaborative editing. Cursor sharing in a single PTY is meaningless (there's only one cursor). | Already addressed — independent scrollback per client (Phase 74) is the correct model. Don't graft Warp's design onto a shared-PTY system. |
| **Synchronized search across all clients viewing the same session** | "I'm pairing — when I find a thing, my pair sees the highlight." | Violates "independent scrollback per client" contract (Phase 74 / MC-04). Search highlight is a client-side overlay; broadcasting it would couple clients in a way users explicitly didn't want. | Search is purely local to each browser/desktop client. Document this. |
| **Replace AgentHub's WS protocol with `@xterm/addon-attach`** | "There's an addon for this." | AgentHub uses its own binary-framing protocol (MsgOutput/MsgInput/MsgResize/MsgMeta — type 0x04 for viewer count) supporting MC-01..MC-06. addon-attach is a basic raw-PTY-over-WS bridge with none of that. | Skip the addon entirely. Existing protocol stays. |
| **Replace native Cmd+C/V with `@xterm/addon-clipboard`** | "There's an addon for this." | Phase 49 already wired Cmd+C/V via the macOS app menu and works on all platforms. The clipboard addon adds OSC 52 read/write — a *different* capability (agents pushing to user clipboard) which is a separate scope question. | Skip for v3.2. Consider OSC 52 support as v3.3 if AI agents start using it. |

---

## Feature Dependencies

```
┌─────────────────────────────────────────────────────────┐
│ Vendored xterm core (v3.1 / Phase 89)                   │
│ + strict CSP (script-src 'self', connect-src 'self')    │
└──────┬──────────────────────────────────────────────────┘
       │
       ├─required by──> All @xterm/addon-* (vendor to web/vendor/xterm/addons/)
       │
       ├──> WebGL renderer ──enhances──> ALL renderers (perf)
       │         └──conflicts──> Canvas, DOM (only one active at a time)
       │         └──fallback──> Canvas → DOM
       │
       ├──> Search addon
       │         └──depends──> Find bar UI (custom, not in addon)
       │         └──enhances──> WebGL (decorations API needs renderer hooks)
       │
       ├──> Web-Links addon
       │         └──security-gate──> Modifier-key requirement (Cmd/Ctrl)
       │         └──enhances──> Hover affordance via existing tooltip primitive
       │         └──routes-to──> Wails BrowserOpenURL (existing, Phase 52)
       │
       ├──> Image addon
       │         └──depends──> CSP allows worker-src 'self' (verify)
       │         └──depends──> Vendored worker bundle (no remote import)
       │         └──conflicts-soft──> Memory budget when many tabs open
       │
       ├──> Unicode11 addon
       │         └──enhances──> ALL renderers (correct widths everywhere)
       │         └──prerequisite-for──> Reliable emoji/CJK in AI agent output
       │
       ├──> Serialize addon
       │         └──enhances──> "Export buffer" / "Copy as HTML" UI
       │         └──independent-of──> All other addons
       │
       ├──> Progress addon (OSC 9;4)
       │         └──enhances──> Tab bar (per-tab progress underline)
       │         └──enhances──> Tray icon (aggregate progress glyph)
       │
       └──> Settings UI (Plugins section)
                 └──depends──> Daemon settings.json (existing, Phase 79)
                 └──depends──> Three-state Save button (existing)
                 └──pattern──> Mirror BannerStack visual language for find bar


SKIPPED for v3.2 (with reasons):
   addon-ligatures        ──blocked-by──> Requires Node APIs, won't work in web embed
   addon-attach           ──blocked-by──> AgentHub uses its own binary-framing WS protocol
   addon-clipboard        ──blocked-by──> Phase 49 already covers Cmd+C/V; OSC 52 is separate scope
   addon-fit              ──already-using──> Existing dependency; no toggle needed
   addon-canvas           ──optional──> Skip; DOM is fine 2nd-tier fallback in 2026 WebViews
   addon-web-fonts        ──blocked-by──> "Font family selection" is Out of Scope in PROJECT.md
   addon-unicode-graphemes ──experimental──> Successor to unicode11, not yet stable; v3.3
```

### Dependency Notes

- **WebGL → DOM fallback chain:** xterm.js's official guidance is to listen for `onContextLoss` on `WebglAddon`, dispose it, and load DOM (or `@xterm/addon-canvas`) instead. AgentHub should fall straight to DOM (skip Canvas) for v3.2 to keep the matrix small — Canvas2D is only valuable on systems where WebGL is broken but Canvas2D is fast, which is rare in 2026.
- **Search ↔ WebGL interaction:** `addon-search`'s `decorations` option draws highlights via xterm's decoration API. Works on WebGL/Canvas/DOM uniformly; visuals inherit the active xterm-theme — already a solved problem in AgentHub since Phase 65/THM-02.
- **Image ↔ existing initial-paint timing:** v1.6 (Phase 35) fixed the initial fit timing with a bounded rAF retry loop on `proposeDimensions()`. The image addon writes into the buffer and triggers redraws — verify it doesn't re-trigger the rAF loop or cause measurement flicker on first paint when an image is in the scrollback at session restore. Likely fine, but flag for UAT.
- **Unicode11 must be loaded before any output is written:** xterm.js evaluates char widths at insert time. Loading unicode11 mid-session leaves all prior output sized using Unicode 6 tables. This is the canonical "applies to new sessions only" case. UI must say so.
- **Progress addon ↔ tab bar / tray:** Existing tab bar (Phase 56) and tray (Phases 41/67) are React/native respectively. Plumbing OSC 9;4 events from xterm.js into both is moderate work but reuses existing event-emit patterns (`session:exit` → `app:quit-requested` is the precedent).
- **CSP ↔ image worker:** Phase 89 set `script-src 'self'`. Web Workers spawned from same-origin URLs inherit `worker-src` from `default-src` (which Phase 89 likely set to `'self'`). If the image addon ships its worker as a `Blob`/`data:` URL, CSP will block it — would require `worker-src 'self' blob:` (a CSP loosening worth scrutinizing). Verify in Phase 1 of v3.2.

---

## MVP Definition

### Launch With (v3.2)

The minimum to call this milestone done. Each item must ship together — Settings UI is meaningless without the addons it toggles, and the addons are friction-prone without a way to disable them.

- [ ] **WebglAddon with context-loss → DOM fallback** — table stakes; one banner via BannerStack on fallback ("GPU rendering unavailable, switched to software").
- [ ] **SearchAddon + custom find bar** (Cmd-F open, Esc close, Enter/Shift-Enter next/prev, regex/case/word toggles, match count "3 of 12") — per-tab.
- [ ] **WebLinksAddon with Cmd/Ctrl-click activation, HTTPS/HTTP only** — hover affordance reusing existing tooltip CSS.
- [ ] **Unicode11Addon, on by default** — flip `terminal.unicode.activeVersion = '11'` at construction.
- [ ] **ImageAddon with sixel + IIP, default `storageLimit` dropped to 32 MB per terminal** (saves memory across N tabs; raisable in advanced settings).
- [ ] **SerializeAddon with one user-facing entry point** — terminal tab right-click → "Copy session as text" (`serialize()`); secondary "Copy as HTML" (`serializeAsHTML()`).
- [ ] **Settings → Plugins section** — toggles for: WebGL, Search, Web Links, Inline Images, Unicode 11, Serialize. Each row: label + 1-line description + toggle + (italic) "Applies to new sessions" where addon can't hot-swap.
- [ ] **Persistence via existing daemon `settings.json`** — extend the `daemonSettings` struct (Phase 79); reuse three-state Save.
- [ ] **Vendored addon assets under `web/vendor/xterm/addons/`** — same model as Phase 89 vendored core; no CDN imports; CI test that no `unpkg.com`/`jsdelivr.net` strings exist in built assets.

### Add After Validation (v3.2.x)

- [ ] **ProgressAddon (OSC 9;4) with tab-bar progress underline + tray glyph** — small fit-and-finish; high differentiator value. *Trigger: confirm at least one bundled CLI emits OSC 9;4 in normal use.*
- [ ] **Per-plugin advanced settings panel** — collapsible "Advanced" reveal under each plugin row for users who want `pixelLimit`, search-regex-on default, link click handler choice. *Trigger: 2+ user requests for a knob behind a default.*
- [ ] **Find-in-all-sessions (Cmd-Shift-F)** — global panel iterating each tab's SearchAddon. *Trigger: per-tab find bar shipped and stable for ≥1 release.*
- [ ] **"Export buffer" Settings UI** — explicit button to save `serializeAsHTML()` to a file via Wails save dialog. *Trigger: serialize is in right-click menu and used.*

### Future Consideration (v3.3+)

- [ ] **`@xterm/addon-unicode-graphemes`** — once it leaves experimental, supersedes unicode11 and adds proper grapheme-cluster awareness (fixes ZWJ emoji rendering).
- [ ] **`@xterm/addon-canvas` as 2nd-tier fallback** — only if user reports of WebGL→DOM falling all the way to slow path become common.
- [ ] **OSC 8 hyperlink support** (different from web-links — actual escape-sequence-defined links from the program) — VT-level link semantics; tracked in xterm.js core, not yet a separate addon.
- [ ] **Image copy/save gesture** — right-click an inline image to copy/save. Requires extending image addon or building a click-hit-test layer.
- [ ] **Custom protocol handlers for web-links** (`file://`, editor:// schemes) — opt-in advanced toggle per protocol. Defer until concrete user need.
- [ ] **Plugin registry / extensibility** — explicitly Out of Scope per PROJECT.md. The same applies to user-installable xterm.js addons.
- [ ] **OSC 52 clipboard (via `@xterm/addon-clipboard`)** — let agents push to user clipboard. Watch for AI CLI adoption.

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---|---|---|---|
| WebGL renderer + fallback | HIGH | MEDIUM (XS code, M for fallback testing) | P1 |
| Search + find bar | HIGH | MEDIUM (XS for addon, M for find-bar UI) | P1 |
| Web links (HTTPS only, Cmd-click) | HIGH | LOW | P1 |
| Unicode 11 widths | HIGH (silent — *prevents* breakage) | LOW (XS) | P1 |
| Inline images | MEDIUM | MEDIUM (CSP/worker verification + memory budget tuning) | P1 |
| Serialize (right-click "Copy session") | MEDIUM | LOW | P1 |
| Settings UI (Plugins section, toggles, "applies to new sessions") | HIGH (gates everything else from a polish standpoint) | MEDIUM | P1 |
| Vendored addon assets (CSP-clean) | HIGH (gates entire shipping of v3.2) | MEDIUM | P1 |
| Progress (OSC 9;4) + tab/tray surfacing | MEDIUM | MEDIUM | P2 |
| Per-plugin advanced settings | LOW–MEDIUM | LOW | P2 |
| Find-in-all-sessions | MEDIUM | MEDIUM | P3 |
| Canvas fallback (between WebGL and DOM) | LOW | LOW | P3 |
| Unicode graphemes (when stable) | MEDIUM | LOW | P3 |
| Custom URL protocols | LOW | LOW | P3 (opt-in, behind warning) |
| Ligatures | LOW (broken in web mode) | n/a — skip | OUT |
| Web fonts | n/a — out of scope | n/a | OUT |
| Attach addon | n/a — duplicates existing | n/a | OUT |
| Clipboard addon | n/a — duplicates Phase 49 | n/a | OUT (revisit OSC 52 in v3.3) |

**Priority key:**
- **P1**: Must ship in v3.2 milestone — defines "Plugin Suite shipped"
- **P2**: Should ship if budget allows; otherwise v3.2.x
- **P3**: Future consideration, tracked but not committed
- **OUT**: Investigated and rejected for v3.2

---

## Per-Plugin "What Good Looks Like"

### 1. WebGL Renderer (`@xterm/addon-webgl`)

**When it kicks in:** Always, by default, on any browser/WebView with WebGL2. AgentHub's Wails WebView (Chromium-based on Win/Linux, WKWebView on macOS) has WebGL2 universally as of macOS 11+/Windows 10+/modern WebKitGTK.

**UX of context loss:** Browser fires `webglcontextlost` on the canvas (GPU OOM, system sleep/wake, driver crash, tab in background too long on some platforms). Without handling, terminal goes blank. With handling: dispose `WebglAddon`, attach a fallback (DOM is simplest), show a one-shot dismissable banner via the existing **BannerStack** (Phase 81): *"GPU rendering unavailable on this device — switched to software rendering. Restart AgentHub to retry."*

**Performance expectation:** ~10× over DOM at 80×24, larger margin at 200×60. xterm.js maintainers have published benchmarks showing WebGL sustains 60fps on output spam where DOM drops to 5–15fps.

**Settings UI:** Toggle. Hot-swap NOT supported reliably — flip "applies to new sessions only." Future advanced reveal: dropdown matching VS Code (`auto` / `on` / `off`) — for v3.2 keep it binary (on/off) with `auto` semantics baked in (try WebGL, fall back on context loss).

**AgentHub-specific concern:** Existing `clearTextureAtlas() + refresh()` for theme-change is in the codebase (THM-02 / Phase 65). That code path must continue to work — verify it tolerates `WebglAddon` being null when WebGL is disabled.

### 2. Scrollback Search (`@xterm/addon-search`)

**Keyboard shortcut conventions:**
- Open: **Cmd-F** (macOS), **Ctrl-F** (Linux/Win) — universally expected.
- Next: **Enter** (or **Cmd-G** / **F3**).
- Previous: **Shift-Enter** (or **Cmd-Shift-G** / **Shift-F3**).
- Close: **Esc**.

**Find bar UI (canonical pattern, used by VS Code, Sublime, every browser):**
- Floating bar pinned to top-right of the active terminal pane.
- Single text input + match count ("12 of 327") + three small toggle buttons: **Aa** (case sensitive), **\\b** or **⌊ab⌋** (whole word), **.\*** (regex).
- Up-arrow / down-arrow icon buttons for prev/next.
- × close button.
- On match, scroll into view + highlight via the addon's `decorations` API (works across DOM/Canvas/WebGL).

**Toggles in detail:**
- **Regex:** Off by default (avoid surprising special-char behavior). Caveat: empty regex matches everywhere — guard against this in input handler.
- **Case sensitivity:** Off by default (matches user expectation from browsers/editors).
- **Whole word:** Off by default. Implementation in addon checks for non-word boundaries (`_`, `(`, `)`, space).
- **Highlight all matches:** Use the addon's `decorations` option — it draws all matches with a dim color, current match with a bright color. AgentHub should pick from the active xterm-theme: `theme.selectionBackground` for "all", `theme.yellow` (or accent) for "current."

**Settings UI:** Toggle for whether the addon is loaded at all (binary). Advanced (P2): defaults for the three search toggles (so power-users can have regex always-on).

**AgentHub-specific:** Find bar visual style must match the BannerStack vocabulary (Phase 81). Use the same border-radius, drop shadow, 200ms slide-in/out animation. Don't ship a Bootstrap/material-style bar that clashes.

### 3. Inline Images (`@xterm/addon-image`)

**Sixel vs iTerm2 IIP — what's the difference?**

| | Sixel | iTerm2 IIP |
|---|---|---|
| Origin | DEC, late 1980s | iTerm2, 2014 |
| Encoding | Custom 6-pixels-per-byte ANSI extension | Base64-encoded image inside `OSC 1337` |
| Image formats | Sixel-encoded raw pixel data | PNG, JPEG, GIF (first frame only in addon) |
| Color space | Indexed palette (default 256, addon caps at 4096) | Full-color via native image decoder |
| Animation | No | Static only (addon-side) |
| What emits it? | `img2sixel` (libsixel), `chafa --format=sixels`, mpv/lsix | `imgcat` (iTerm2 distro), `chafa --format=iterm2`, `viu`, `wezterm imgcat` |

**What shells/CLIs emit them in practice (AI coding context):**
- `chafa` is the dominant general-purpose tool — auto-detects best protocol.
- `imgcat` is the pure-iTerm2 path (works because xterm.js's IIP support is iTerm2-compatible).
- AI coding CLIs themselves (Claude Code, Codex, Gemini CLI, OpenCode) **do not currently emit images natively** as of 2026-05. But: users do `cat screenshot.png | chafa` or pipe matplotlib output through `imgcat` constantly. Real demand is from human-issued commands, not agent output.
- Growing pattern: `agent | tee >(some-tool-that-renders)` for agent-generated diagrams.

**Copy/save image gestures:** **Anti-feature for v3.2.** Addon exposes no first-class image extraction; building hit-test + save flow is real work. Ship inline rendering only.

**AgentHub-specific:**
- **Memory budget:** default `storageLimit: 128 MB` × N open tabs is too much. Drop default to **32 MB per terminal**; expose in advanced.
- **CSP:** Verify worker assets bundled same-origin. If addon uses `Blob`/`data:` workers, will need `worker-src 'self' blob:` (a CSP relaxation worth scrutinizing).
- **Tailscale-served viewer:** Images render the same in the browser-view as in the desktop app — verify in UAT (test with `chafa` + Tailscale-served session).

### 4. Web Links (`@xterm/addon-web-links`)

**Hover affordance:** xterm.js does not draw a tooltip itself — addon exposes `hover` and `leave` callbacks. AgentHub should reuse the existing tooltip primitive (already used elsewhere in React) to show:
> `https://example.com/path` *(Cmd-click to open)*

**Modifier-key requirement:** Cmd on macOS, Ctrl on Linux/Windows. Default is plain click — *override* in the `activate` callback:
```ts
activate: (event, uri) => {
  const isMac = navigator.platform.includes('Mac');
  const ok = isMac ? event.metaKey : event.ctrlKey;
  if (!ok) return;
  // route to BrowserOpenURL via Wails (existing pattern, Phase 52)
}
```

**URL detection regex:** Addon defaults to a robust HTTP(S) regex (handles trailing punctuation, parens-in-Wikipedia-URLs, etc.). Don't override unless adding protocols.

**Security:**
- Default = HTTPS/HTTP only. ✓
- `file://` = anti-feature. Don't enable.
- For AgentHub's tailnet model: a remote tailnet user viewing a session can be tricked into clicking a link to a phishing site. Same risk as any browser, mitigated by the modifier requirement (no accidental clicks). Document in security notes.
- Open links via Wails `BrowserOpenURL` (already used by Remote Sessions panel — Phase 52) so they go to the user's default browser, not into the WebView.

**Settings UI:** Toggle (binary). Advanced (P2): "Click handler" dropdown — `Default browser` (current). Skip "New AgentHub tab" (this is a terminal app, not a browser) and "Custom command" (anti-feature).

### 5. Unicode 11 (`@xterm/addon-unicode11`)

**What breaks today without it:**
- **CJK alignment in box-drawing UIs:** Asian users running Claude Code or Codex see misaligned tables/borders because Unicode 6 (xterm.js default) doesn't know about CJK ranges added in Unicode 7–11.
- **Emoji ZWJ sequences:** 👨‍👩‍👧 (man+woman+girl) renders as three separate characters consuming 6 cells instead of 2; 🏳️‍🌈 (rainbow flag) shows as flag + rainbow side-by-side.
- **Skin-tone modifiers:** 👍🏽 occupies wrong width; layout breaks.
- **Powerline glyphs and Nerd Font ranges:** Most are in Private Use Area, unaffected — but adjacent CJK and emoji glyphs *are* affected, and a single misplaced char shifts an entire status line.

**Visibility to users:** Subtle but real. Most western developers won't notice; CJK users notice immediately; everyone notices when emoji-using AI agents render TUI elements.

**Caveat (HIGH-confidence pitfall):** xterm.js's `unicode11` tables are reportedly closer to Unicode 12 than 11 (see [xtermjs#4753](https://github.com/xtermjs/xterm.js/issues/4753)), and still lack proper grapheme-cluster awareness. ZWJ sequences are still imperfect. The successor `@xterm/addon-unicode-graphemes` exists but is experimental. For v3.2: ship unicode11, document the limitation, plan unicode-graphemes for v3.3.

**Settings UI:** Toggle (binary). On by default. **"Applies to new sessions only"** — flipping mid-session leaves prior output sized using old tables.

### 6. Serialize (`@xterm/addon-serialize`)

**Capture format:** Two modes:
- `serialize()` → string of VT escape sequences. Replay into a fresh `Terminal` via `terminal.write(str)` reproduces the buffer pixel-perfectly (within renderer fidelity).
- `serializeAsHTML()` → standalone HTML with inline `<style>`. Drop into any browser; looks like the terminal looked.

**Restore semantics:** Replaying VT into a new Terminal restores **visible buffer + scrollback contents + cursor position + colors + attributes**. Does NOT restore: alt-screen state across the boundary, mouse mode, application keypad mode, OSC-set window title, OSC-set palette overrides (in some cases).

**What it's good for:**
- **Debug capture:** "Show me what your terminal looked like when it broke." — share `serializeAsHTML()` output, much smaller than a screenshot, copy/paste-able.
- **Share-as-text:** Pasting AI output into a chat/PR description with colors preserved.
- **Lightweight session snapshot:** Save buffer to disk on tab close; restore visual state when user reopens. Note that this is *visual* — the underlying PTY still has its own state (which AgentHub's daemon already persists via go-pty + scrollback replay, MC-04).

**What it explicitly does NOT preserve:**
- PTY state (cwd, env vars, running processes).
- Active shell history.
- Open file descriptors / job control state.
- Cursor blink phase, animation timing.
- Xterm decorations (search highlights, custom marker overlays).

**AgentHub fit:**
- Right-click on terminal tab → "Copy session as text" / "Copy session as HTML."
- (P2) Settings → Session Behavior → "Export buffer..." → Wails save dialog → `.html` file.
- (Future, not v3.2) On graceful tab close, store last-N-bytes of `serialize()` in `settings.json`; on session reopen, write it into the new terminal as a "previous session" header.

**Settings UI:** Toggle (binary, very low cost — addon is tiny and inert until called).

### 7. Settings UI for Plugins

**Patterns from mature apps:**

- **VS Code terminal:** Setting per feature, mostly dropdowns/booleans. `terminal.integrated.gpuAcceleration` is `auto` / `on` / `off` / `canvas`. Search is its own command palette command. Terminal feature toggles live alongside hundreds of other settings — no dedicated "Plugins" section, but each setting has a clear scope ("Restart terminal for this to take effect" appears as italic note).
- **iTerm2:** Tabbed Preferences (General / Appearance / Profiles / Keys / Pointer / Advanced). Profile-scoped feature toggles (e.g., per-profile inline image support). No "addon registry" — features are built-in.
- **Hyper:** `~/.hyper.js` plain-JS config file. `plugins: ['hyper-tab-icons', ...]` array; live reload on save. Power-user friendly, opaque to GUI users.
- **Tabby:** Settings panel with categories (Appearance, Hotkeys, Plugins). The "Plugins" pane has *per-plugin enable toggle + Configure button* (often opens a sub-panel) + Install/Uninstall (Tabby has an actual plugin registry). This is the closest model to what AgentHub needs, minus install/uninstall.
- **Warp:** Settings → Features panel; per-feature toggles ("AI Command Suggestions", "Workflows"). Heavy on copy explaining what each toggle does.
- **Alacritty:** TOML config file only. No GUI. Relevant negative example — power users love it, casual users hate it.

**Recommended AgentHub pattern:**

A new section in the existing Settings tab (currently Appearance / Web Server / Paths sections per Phase 69's h3 layout). Add a fourth section:

```
## Plugins

### Rendering
WebGL renderer                                          [ON ●]
GPU-accelerated terminal rendering. Falls back to
software rendering automatically on unsupported devices.

### Search
Find in scrollback                                      [ON ●]
Press Cmd-F in any terminal to find text in its history.
> Advanced ▾  (collapsible — P2)
    Default: Match case          [ ]
    Default: Whole word          [ ]
    Default: Regular expression  [ ]

### Web Links
Clickable web links                                     [ON ●]
Cmd-click HTTP(S) URLs in terminal output to open in
your default browser.

### Inline Images
Inline image rendering                                  [ON ●]
Renders sixel and iTerm2 inline images in terminal output.
Applies to new sessions only.
> Advanced ▾  (P2)
    Memory limit per session: [32] MB

### Unicode
Unicode 11 character widths                             [ON ●]
Correct widths for CJK, emoji, and modern Unicode glyphs.
Applies to new sessions only.

### Session Capture
Buffer serialization                                    [ON ●]
Enables "Copy session" actions in the terminal right-click menu.
```

**Per-plugin enable/disable mechanics:**
- Backend: extend `daemonSettings` struct in `internal/daemon/settings.go` (Phase 79 introduced this) with a `Plugins map[string]bool` (or named fields for type safety).
- Frontend: each toggle is a simple `<input type="checkbox">` styled to match existing settings; persists via existing three-state Save.
- Hot-swap reality:
  - `WebglAddon`: disposing+reattaching is unreliable on live terminals → "applies to new sessions."
  - `SearchAddon`: hot-swap fine (just stop showing the find bar).
  - `WebLinksAddon`: hot-swap fine.
  - `ImageAddon`: must be loaded before any image data hits the buffer → "applies to new sessions."
  - `Unicode11Addon`: must be loaded before any output → "applies to new sessions."
  - `SerializeAddon`: hot-swap fine.

**Per-plugin config patterns:**
- Use a `<details><summary>Advanced</summary>...</details>` collapsible reveal under the plugin row. Default collapsed.
- For Search: defaults for case/word/regex toggles.
- For Web Links: (v3.2.x) click handler choice — currently single value, ship binary toggle for v3.2.
- For Inline Images: memory limit (sensible bounds, 8–256 MB).

**"Applies to new sessions only" indicator pattern:**
- Italic, dimmed text directly under the toggle, on its own line.
- Always-present (not just when toggle changes), so users see it before clicking.
- Same italic small-text style used for descriptions, but different color (use `#ffd479`-ish accent or whatever matches BannerStack warning style — verify with existing CSS tokens).
- Optional polish: when user toggles a non-hot-swappable plugin and hits Save, show a one-shot banner via BannerStack: *"Plugin changes will apply to new sessions you create. Existing tabs are unaffected."*

### 8. Plugin Survey — Other Addons

| Addon | Purpose | Real for AgentHub audience? | Verdict |
|---|---|---|---|
| **`@xterm/addon-fit`** | Resize terminal to container | YES — already a dependency | Keep using; not in scope of v3.2 toggle UI |
| **`@xterm/addon-attach`** | Connects xterm to a WebSocket server speaking raw PTY | NO — AgentHub uses its own binary-framing protocol with multi-client metadata, scrollback replay, max-wins resize. Replacing it = giving up MC-01..MC-06. | **Skip** |
| **`@xterm/addon-clipboard`** | OSC 52 clipboard read/write | NO — Phase 49 already wired Cmd+C/V via app menu. OSC 52 from agents would be a *new* capability (potentially useful but separate scope) | Skip for v3.2; consider as v3.3 if AI agents start using OSC 52 to push to user clipboard |
| **`@xterm/addon-canvas`** | Canvas2D renderer (fallback path for WebGL) | LOW — modern WebViews universally have WebGL2; DOM is fine as 2nd-tier fallback | Skip for v3.2; revisit if WebGL→DOM jump turns out too jarring |
| **`@xterm/addon-image`** | Sixel + IIP | YES (covered above) | **Ship in v3.2** |
| **`@xterm/addon-ligatures`** | Programming ligatures | NO — requires Node APIs, breaks in web embed | Skip — broken model for AgentHub |
| **`@xterm/addon-progress`** | OSC 9;4 progress reporting | YES — strong AgentHub-specific differentiator (tab/tray surfacing) | **P2 (v3.2.x)**; bundle in v3.2 only if budget allows |
| **`@xterm/addon-search`** | Buffer search | YES (covered above) | **Ship in v3.2** |
| **`@xterm/addon-serialize`** | Buffer serialization | YES (covered above) | **Ship in v3.2** |
| **`@xterm/addon-unicode-graphemes`** | Grapheme-cluster aware widths (experimental) | YES (eventually) | **Defer to v3.3** when stable |
| **`@xterm/addon-unicode11`** | Unicode 11 widths | YES (covered above) | **Ship in v3.2** |
| **`@xterm/addon-web-fonts`** | Async web-font integration | NO — "Font family selection" is Out of Scope in PROJECT.md | Skip per existing decision |
| **`@xterm/addon-web-links`** | Clickable URLs | YES (covered above) | **Ship in v3.2** |
| **`@xterm/addon-webgl`** | GPU rendering | YES (covered above) | **Ship in v3.2** |

**Community / non-`@xterm` addons of note:**
- `xterm-addon-shell-integration` (community) — would parse OSC 133 prompts to enable per-prompt navigation. Not maintained by xterm core, scope risk; skip.
- `xterm-addon-zmodem` — file-transfer over terminal. Niche; AgentHub users have Tailscale + scp/rsync. Skip.
- No widely-used `xterm-addon-shellintegration` or `xterm-addon-jumplist` analogue exists for the modern `@xterm/*` scope. The official addon set is the relevant universe.

---

## Competitor Feature Analysis

| Feature | VS Code Terminal | iTerm2 | Hyper | Tabby | Warp | AgentHub v3.2 (proposed) |
|---|---|---|---|---|---|---|
| GPU rendering | WebGL default + auto fallback | Native Metal | WebGL (Electron) | WebGL | Native (Rust+wgpu) | WebGL + DOM fallback |
| Find/search | Cmd-F + find bar; regex/case/word toggles | Cmd-F + find bar | Cmd-F via plugin | Cmd-F built-in | Cmd-F + AI-aware search | Cmd-F per-tab + (P3) Cmd-Shift-F across tabs |
| Inline images | Yes (xterm-addon-image, sixel+IIP) | Yes (native IIP) | Plugin | Yes | No (deliberate) | Sixel + IIP via addon-image |
| Web links | HTTPS only, Cmd-click | HTTPS+file, Cmd-click toggle | Plugin | Yes | Yes, AI-augmented | HTTPS only, Cmd/Ctrl-click |
| Unicode/emoji | Unicode 11 default | Mature, 14+ | Inherited from xterm.js | Inherited | Native | Unicode 11 (tracking unicode-graphemes for v3.3) |
| Buffer serialize | Internal (workspace state) | Save Selected Text | No | No | No | Right-click "Copy as HTML/text" + Settings export |
| Plugin toggles UI | Settings search, per-feature | Tabbed Prefs, per-profile | Plain JS config file | Per-plugin toggles + registry | Settings → Features panel | h3-section in existing Settings, per-plugin toggle + advanced reveal |
| OSC 9;4 progress | Yes (Windows taskbar) | No | No | No | No | (P2) Tab-bar underline + tray glyph — AgentHub differentiator |
| Multi-client viewer | No | No | No | No | Drives shared blocks (different model) | Existing (Phase 74); search/find stays per-client |

**Where AgentHub leads on this matrix:**
- Tab-level OSC 9;4 progress + tray surfacing.
- Buffer serialize as a first-class user gesture ("Copy session"), not a hidden API.
- Per-plugin advanced reveal pattern in a coherent Settings tab (Tabby is closest, AgentHub matches).

**Where AgentHub deliberately doesn't compete:**
- AI-augmented search (Warp).
- Custom protocol handlers (iTerm2).
- Plugin marketplace (Hyper, Tabby).
- Native rendering performance (Warp's wgpu).

---

## Sources

**Official xterm.js documentation & code (HIGH confidence):**

- [xterm.js — Using addons (official guide)](https://xtermjs.org/docs/guides/using-addons/)
- [xterm.js — Link Handling guide](https://xtermjs.org/docs/guides/link-handling/)
- [xterm.js GitHub — addons directory](https://github.com/xtermjs/xterm.js/tree/master/addons)
- [@xterm/addon-webgl README](https://github.com/xtermjs/xterm.js/blob/master/addons/addon-webgl/README.md)
- [@xterm/addon-search typings](https://github.com/xtermjs/xterm.js/blob/master/addons/addon-search/typings/addon-search.d.ts)
- [@xterm/addon-image (npm)](https://www.npmjs.com/package/@xterm/addon-image)
- [@xterm/addon-image upstream (jerch/xterm-addon-image)](https://github.com/jerch/xterm-addon-image)
- [@xterm/addon-web-links (npm)](https://www.npmjs.com/package/@xterm/addon-web-links)
- [@xterm/addon-unicode11 source](https://github.com/xtermjs/xterm.js/tree/master/addons/addon-unicode11)
- [@xterm/addon-serialize typings](https://github.com/xtermjs/xterm.js/blob/master/addons/addon-serialize/typings/addon-serialize.d.ts)
- [@xterm/addon-progress (npm)](https://www.npmjs.com/package/@xterm/addon-progress)
- [@xterm/addon-fit (npm)](https://www.npmjs.com/package/@xterm/addon-fit)
- [@xterm/addon-canvas (npm)](https://www.npmjs.com/package/@xterm/addon-canvas)

**Standards / protocol references (HIGH confidence):**

- [OSC 9;4 — Progress Bar Sequence (rockorager.dev)](https://rockorager.dev/misc/osc-9-4-progress-bars/)
- [Set the progress bar in Windows Terminal (MS Learn)](https://learn.microsoft.com/en-us/windows/terminal/tutorials/progress-bar-sequences)
- [iTerm2 Inline Images Protocol](https://iterm2.com/documentation-images.html)
- [Are We Sixel Yet?](https://www.arewesixelyet.com/) — Sixel adoption tracker
- [Terminal Graphics Protocols: Kitty, Sixel, iTerm2, and Beyond (Akmatori)](https://akmatori.com/blog/terminal-graphics-protocols)

**Unicode width / xterm.js limitations (HIGH confidence — issue threads):**

- [xtermjs#4753 — xterm.js lies about Unicode 11, really uses Unicode 12](https://github.com/xtermjs/xterm.js/issues/4753)
- [xtermjs#3304 — grapheme cluster & unicode v13 support](https://github.com/xtermjs/xterm.js/issues/3304)
- [xtermjs#1709 — Unicode handling in xterm.js](https://github.com/xtermjs/xterm.js/issues/1709)
- [Terminal Emulators Battle Royale – Unicode Edition (Jeff Quast)](https://www.jeffquast.com/post/ucs-detect-test-results/)

**VS Code terminal as reference impl (MEDIUM confidence — public docs + code):**

- [VS Code Terminal Appearance docs](https://code.visualstudio.com/docs/terminal/appearance)
- [microsoft/vscode#182442 — Terminal image support PR](https://github.com/microsoft/vscode/pull/182442)
- [microsoft/vscode#15211 — Add a setting to disable gpu acceleration](https://github.com/microsoft/vscode/issues/15211)

**Inline image tooling for the AI-CLI audience (MEDIUM confidence):**

- [chafa GitHub](https://github.com/hpjansson/chafa/)
- [BourgeoisBear/rasterm — Go encoder for iTerm/Kitty/Sixel](https://github.com/BourgeoisBear/rasterm)

**Competitor / market context (MEDIUM confidence — comparison articles, dated):**

- [Warp vs iTerm2 comparison](https://www.warp.dev/compare-terminal-tools/iterm2-vs-warp)
- [Terminal Emulators Comparison (Terminal Trove 2026)](https://terminaltrove.com/compare/terminals/)
- [iTerm2 Preferences — Keys/Profiles (official)](https://iterm2.com/documentation-preferences-profiles-keys.html)

---
*Feature research for: xterm.js plugin suite (AgentHub v3.2, Issue #36)*
*Researched: 2026-05-03*
