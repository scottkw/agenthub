# Project Research Summary

**Project:** AgentHub v3.2 — xterm.js Plugin Suite (Issue #36)
**Domain:** xterm.js addon ecosystem integrated into an existing Wails + React + Go terminal app with multi-client web fan-out
**Researched:** 2026-05-03
**Confidence:** HIGH (versions, bundle sizes, and integration points verified against the live npm registry, unpacked tarballs, and the actual repo files; security/CSP claims tied to v3.1 hardening that already shipped)

## Executive Summary

v3.2 is a **partial refactor + targeted addition** milestone, not a greenfield feature. `frontend/src/components/TerminalPanel.tsx` already loads `addon-fit`, `addon-unicode11`, and `addon-webgl` (with `onContextLoss` fallback). `addon-clipboard@0.2.0` is in `package.json` but never wired. The v3.2 ask is therefore three things at once: (1) add **four genuinely new addons** — `addon-search`, `addon-image`, `addon-web-links`, `addon-serialize`; (2) wire the dormant `addon-clipboard`; (3) move all of the above (including the three already-shipping ones) under a single Settings-driven enable/disable system that persists via the existing daemon `settings.json` and propagates over Wails events. Web-served sessions today have **none** of these addons (only `addon-fit`); v3.2 also brings them to web parity, gated by `worker-src`/`blob:` CSP review for `addon-image` specifically.

The recommended approach is to ship the foundation first and the riskiest addon last. Establish daemon `PluginSettings` persistence + Wails RPC + a Settings UI shell with no addon work behind it, then migrate the three already-loaded addons (webgl, unicode11, clipboard) into the new reconcile pattern as a low-risk shakedown of the system, then layer in the new addons in increasing-risk order: search → web-links (security gate) → image (CSP gate, heaviest, most novel) → serialize → optional `addon-progress`. This honors the dependency truth (settings system before addons; vendoring discipline before web parity; webgl atlas state before search/image which both interact with rendering) and contains the two highest-impact risks — phishing on Tailscale-served web-links and CSP/memory exposure from `addon-image` — to dedicated phases with their own UAT gates.

The standout risks are not technical — they are **security-policy decisions**. Web-links on Tailscale-served sessions is a fresh phishing primitive: a tailnet viewer trusts the AgentHub URL, sees a clickable URL emitted by an arbitrary process, and gets redirected. OSC 8 hyperlinks let attackers display `github.com` while linking `evil.example`. This must ship with click-confirmation, OSC 8 href display on hover, IDN/Punycode warnings, and a strict scheme allowlist (`https`, `http`, `mailto`) — same posture as v3.1's WS Origin allowlist. Second-rank risk: `addon-image`'s sixel decoder defaults `storageLimit` to 100 MB of decoded RGBA per terminal — at 8 open tabs that is a tab-OOM within reach. Override to 16 MB and audit whether the addon constructs any `blob:` workers (the v3.1 CSP currently has no `worker-src`, falls back to `default-src 'none'`, would block them silently). These two gates dominate v3.2's perceived quality.

## Key Findings

### Recommended Stack

The base stack (Wails v2 + React 19.2.4 + xterm 6.0.0 + Vite 8 + pnpm + vitest) is unchanged. v3.2 adds **5 new pnpm packages** and activates 1 already-installed one. Detail in `STACK.md`.

**Core technologies (new in v3.2):**
- **`@xterm/addon-search@0.16.0`** — scrollback find with regex/case/word; cheap (39 KB ESM); hot-swappable
- **`@xterm/addon-image@0.9.0`** — sixel + iTerm2 IIP via inline-base64 WASM (no separate `.wasm` file); 62 KB ESM; lazy-load candidate
- **`@xterm/addon-web-links@0.12.0`** — clickable URLs with custom activate handler; 3 KB; security-critical
- **`@xterm/addon-serialize@0.14.0`** — buffer/scrollback capture; 16 KB; passive
- **`@xterm/addon-clipboard@0.2.0`** — already installed, never wired; OSC 52 read/write; 6 KB
- **(Optional, recommended P2)** `@xterm/addon-progress@0.2.0` — OSC 9;4 progress sequences; 1.4 KB; uniquely good fit for AI-CLI long-running task feedback (tab-strip glyph, tray badge)

**Explicitly rejected:** `addon-canvas` (peer-deps `^5.0.0`, incompatible with our xterm 6); `addon-ligatures` (Node-only — `font-finder`/`font-ligatures` need filesystem access; will silently no-op in web-served context, splitting UX); `addon-iframe` (does not exist); `addon-attach` (replaces our custom binary-framing relay protocol with no benefit and loses MC-01..MC-06 metadata); `addon-unicode-graphemes` (experimental — defer to v3.3).

**Vendoring impact:** `web/embed.go` extends with new `//go:embed` lines per addon; `web/vendor/xterm/VERSION` grows by ~6–8 lines; `internal/webserver/vendor_drift_test.go` regex must generalize from the hardcoded `(xterm|addon-fit)` group to match every `@xterm/addon-*` package. **No new fonts, no separate WASM files, no workers in any addon's published artifact** — the sixel decoder embeds WASM as a base64 string inside the JS bundle via the upstream `inwasm` helper. CSP `script-src 'self'` from v3.1 D-09 covers all addon code without amendment, **with the open question of whether `addon-image` ever constructs `blob:` URLs at runtime** — that audit is the load-bearing gate on the image phase.

### Expected Features

Detail in `FEATURES.md`.

**Must have (table stakes for a 2026-era xterm.js app):**
- WebGL renderer with context-loss → DOM fallback (already shipping; needs hot-swap support added)
- Find/search in scrollback with Cmd-F overlay, regex/case/word toggles, match count
- Clickable HTTP(S) links with Cmd/Ctrl-click activation (security-critical default)
- Unicode 11 widths for emoji/CJK alignment (already shipping; just needs Settings toggle)
- Settings UI for per-plugin enable/disable with "applies to new sessions only" indicator on non-hot-swap addons

**Should have (differentiators for AgentHub's audience):**
- Inline images via sixel + iTerm2 IIP — AI CLIs increasingly emit charts/diagrams via `chafa --format=iterm2`
- Buffer serialize as a first-class user gesture ("Copy session as text/HTML") rather than a hidden API
- `addon-progress` (OSC 9;4) surfaced in tab strip + tray — uniquely good fit for AI-CLI long-running tasks; no other tabbed terminal app does per-tab progress glyphs
- Find bar matching AgentHub's BannerStack visual language (200ms slide-in/out, TokyoNight palette, theme-aware highlight via `theme.selectionBackground`)

**Defer (v2+ / out of scope):**
- `addon-unicode-graphemes` (experimental upstream; revisit in v3.3 when stable)
- Find-in-all-sessions (Cmd-Shift-F across tabs) — needs per-tab find shipped first
- Canvas fallback addon as 2nd-tier between WebGL and DOM (only if reports flood in)
- OSC 52 clipboard adoption from agents (watch for AI CLI uptake)
- Image copy/save gestures (addon doesn't expose pixel extraction)
- Custom URL protocol handlers (`file://`, `vscode://` — security-sensitive opt-in)
- Plugin extensibility / user-installable addons (explicitly Out of Scope per PROJECT.md)

### Architecture Approach

A single source of truth in the daemon, propagated to all xterm.js consumers (desktop Wails + web-served Tailscale). Detail in `ARCHITECTURE.md`.

**Major components:**
1. **`daemonSettings.Plugins *PluginSettings`** (Go) — extend the existing struct in `internal/daemon/engine.go`; pointer for tri-state nil-defaults; new file `internal/daemon/plugin_settings.go` defines per-plugin nested config types (`SearchConfig`, `WebLinksConfig`, etc.). Persists to `<configDir>/settings.json` with a one-time migration from v3.1 (defaults merge — must NOT use `omitempty` on plugin keys).
2. **Wails bindings + event** — single `GetPluginSettings()` / `SetPluginSettings(s)` pair in `app.go`; emits `settings:plugins` runtime event on save (matching the existing `session:exit` / `app:quit-requested` pattern).
3. **`PluginsSection.tsx` + per-plugin config panels** — fourth h3 block in the existing scrollable Settings tab (Phase 69 layout); reuses the three-state Save button pattern; per-plugin config is an inline `<details>` disclosure.
4. **`pluginReconcile.ts` + `useEffect([pluginConfig])` in TerminalPanel** — pure reconcile functions diff old vs new config and call `loadAddon`/`addon.dispose()`. Hot-swappable plugins (webgl, search, web-links, serialize, clipboard) apply live; non-hot (unicode11, image) stamp at terminal-create time only and the UI shows an "applies to new sessions" badge.
5. **`/api/plugin-config` endpoint + web-side reconcile** — for Tailscale-served `web/assets/terminal.js`; reuses existing capability gate and `connect-src 'self'` CSP. All addon JS files load eagerly via `<script>` tags (vendored same-origin); the config decides which to instantiate.

**Patterns inherited from existing codebase:**
- Source-inspection vitest tests (jsdom can't run Canvas/WebGL — keep the `?raw` / `fs.readFileSync` precedent)
- `Get<X>`/`Set<X>` Wails binding pairs (no batched save)
- Vendor-drift test as a load-bearing CI gate (extends regex; not advisory)
- `clearTextureAtlas() + refresh()` after theme/renderer changes (already in code; do not regress)

### Hot-Swap Matrix (authoritative)

Cross-cut from STACK + ARCHITECTURE + PITFALLS — single table for the roadmap to reference.

| Addon | Hot-swap on live terminal? | Default | Web-shared vs local-only | New-session badge? | Notes |
|---|---|---|---|---|---|
| `addon-webgl` | **YES both directions** (dispose + re-load works) | ON | Shared (both desktop and web get WebGL) | No (live) | Was previously load-once; v3.2 makes hot. Must keep `onContextLoss` → DOM fallback path. |
| `addon-search` | **YES** (purely additive UI overlay) | ON | Shared addon load; **web search UI deferred** to future phase | No (live) | Cheapest plugin; cleanest hot-swap |
| `addon-web-links` | **YES** (config change requires dispose+reload of just the addon) | ON | Shared | No (live) | Already-rendered links update on next refresh |
| `addon-image` | Technically yes, but messy mid-stream (lossy on already-rendered images) | ON | Shared (CSP-gated) | **YES** ⚠ | Treat as new-session for clean UX |
| `addon-unicode11` | **NO** — buffer already laid out; switching tables retroactively shifts widths | ON | Shared (must be — affects line wrap, otherwise scrollback diverges across clients) | **YES** ⚠ | Settings UI must communicate clearly |
| `addon-serialize` | **YES** (passive reader, never touches buffer) | ON | Local-only (desktop "Save Terminal As…" — no web UI in v3.2) | No (live) | Library capability; no auto-save to disk (PII risk) |
| `addon-clipboard` | **YES** (OSC 52 hook is parser-level) | ON | Shared | No (live) | Wakes up the dormant package.json entry |
| `addon-progress` (P2) | **YES** | OFF (P2 — ship if budget allows) | Shared | No (live) | Surfaces via `progressAddon.onChange` → tab-strip + tray |

**Decision rule for the Settings UI:** show "(applies to new sessions)" only on `addon-unicode11` and `addon-image`. Other toggles apply live.

### Critical Pitfalls

Top 5 from `PITFALLS.md` — these are the load-bearing risks that must be designed against, not just acknowledged:

1. **Web-Links Phishing on Tailscale-Served Sessions (PITFALLS #8)** — A compromised CLI emits `https://gооgle.com` (Cyrillic) or an OSC 8 with mismatched display text vs href. Tailnet viewer clicks; their browser has full cookies/OAuth/password manager. **Mitigation:** click-confirmation popover with full resolved URL on first link per session; OSC 8 href displayed on hover (always show actual href, not display text); IDN/Punycode warning; strict scheme allowlist (`https`, `http`, `mailto` only — never `file://`, `javascript:`, custom protocols by default). Treat this with the rigor v3.1 applied to the WS Origin allowlist. **This is the single most important quality gate in v3.2.**
2. **Sixel Storage Bomb + CSP `worker-src` Audit (PITFALLS #3, #4, #20)** — `addon-image` defaults `storageLimit: 100` MB of decoded RGBA per terminal; ×8 tabs = OOM. Override to **16 MB**. Separately: audit `addon-image.js` source for `URL.createObjectURL`/`blob:`/dynamic-Worker construction. v3.1 CSP has **no `worker-src`** (falls back to `default-src 'none'` — silent block). If the addon needs blob workers, amend CSP to `worker-src 'self' blob:` with the same v3.1 D-09 documentation rigor. **Re-run zero-violation Chromium e2e suite + add Safari and Firefox** after every addon integration (Tailscale audience includes iPad Safari).
3. **WebGL Context Loss Drops Scrollback (PITFALLS #1, #10, #18)** — Naïve recovery on `onContextLoss` is "create a new Terminal," which obliterates scrollback. Correct recovery: dispose only `WebglAddon`, fall back to DOM (skip canvas — superseded), emit one-shot BannerStack toast, never auto-retry. Also: **detect software WebGL** via `gl.getParameter(RENDERER)` for SwiftShader/llvmpipe/ANGLE-software — fall back proactively (iPad Safari, GPU-blacklisted corp browsers, software-rasterized old Linux). And: do NOT regress the v1.12 `clearTextureAtlas + refresh` theme-change path (PITFALLS #18) — atlas caches glyph textures per (char, fg, bg, attrs), must clear on theme change.
4. **"Applies to New Sessions Only" Miscommunication (PITFALLS #11, #13)** — Other AgentHub Settings (theme, font size) apply live; users build a "Settings = live" mental model. Toggling unicode11 or image with no visible effect generates "the toggle is broken" reports. **Mitigation:** dedicated visual treatment for non-hot-swap toggles — italic caption directly under the toggle ("Applies to new sessions you create") **plus** a one-shot post-toggle BannerStack confirmation. Designed up-front in the Settings UI phase, not retrofitted.
5. **Settings.json Migration Zeroes Plugin Defaults (PITFALLS #14)** — Naïve `json.Unmarshal` of v3.1 settings into v3.2 struct yields Go zero values (false/0). Returning user lands with **all addons disabled and `storageLimit: 0`** — terminal looks worse than before. **Mitigation:** `defaultSettings()` constructor populates desired defaults; load = parse on top of populated default struct OR post-process zero values. Add `"schemaVersion": 2`. Fixture test loading v3.1 `settings.json` → asserting v3.2 defaults populated is non-negotiable.

**Honorable mention:** PITFALLS #15 (Multi-Client Plugin State Drift) — distinguish renderer-choice (per-client) from buffer-interpretation (server-broadcast for unicode11). Decision must land before Settings UI is designed.

## Implications for Roadmap

The four researchers proposed different phase splits (STACK 5, FEATURES 8, ARCHITECTURE 5, PITFALLS 10). Synthesized into **8 phases** that honor the dependency truth and isolate the two highest-risk addons. Phase numbering continues from v3.1 (Phase 89 was last shipped; Phase 90 was milestone close; Phase 91 follow-ups already filed and deferred).

### Phase 92: Plugin Settings Foundation (no addons yet)
**Rationale:** every later phase plugs into this; cheapest to validate; lowest risk. Establishes daemon persistence + Wails RPC + Settings UI shell with **no addon behavior wired**. If this phase reveals architectural problems, no addon work is wasted.
**Delivers:** `daemonSettings.Plugins *PluginSettings`; `GetPluginSettings`/`SetPluginSettings` Wails bindings; `settings:plugins` runtime event; `PluginsSection.tsx` skeleton with 6 disabled toggles; `pluginConfig` prop threaded App.tsx → TerminalPanel (consumed in subsequent phases); migration test from v3.1 settings.json fixture.
**Addresses:** PITFALLS #14 (migration), #13 (UI affordance design).
**Avoids:** scope creep — explicit no-addon-loading-yet.

### Phase 93: Vendoring Discipline + Web Parity for Already-Shipping Addons
**Rationale:** webgl + unicode11 + clipboard already load on desktop but **none are vendored for web** (web today only has `addon-fit`). Migrating the three already-loaded addons under the new reconcile pattern AND vendoring them for web is a controlled shakedown of the entire system before adding net-new addons. Generalizes `vendor_drift_test.go` regex to match all `@xterm/addon-*` keys (load-bearing CI gate).
**Delivers:** `pluginReconcile.ts` with `reconcileWebgl`/`reconcileUnicode11`/`reconcileClipboard`; web-served terminal page loads webgl + unicode11 + clipboard via vendored `<script>` tags; `/api/plugin-config` endpoint with capability gate; `vendor_drift_test.go` generalized; `web/vendor/xterm/VERSION` manifest extended.
**Uses:** `@xterm/addon-webgl@0.19.0`, `@xterm/addon-unicode11@0.9.0`, `@xterm/addon-clipboard@0.2.0` (all already in `package.json`).
**Implements:** state propagation pipeline (daemon → Wails event → React → reconcile).
**Avoids:** PITFALLS #1 (context loss recovery preserved), #16 (vendor drift), #18 (theme atlas regression).

### Phase 94: Search Addon + Find Bar UI
**Rationale:** purely additive UI; hot-swap friendly; no CSP risk; cheapest "new addon" to prove the per-plugin-config UI flow (search has 3 sub-flags: regex/case/word).
**Delivers:** `@xterm/addon-search@0.16.0` vendored + wired; floating find bar overlay matching BannerStack visual vocabulary; Cmd-F (focus-conditioned!) / Esc / Enter / Shift-Enter / Cmd-G keybindings; match count "3 of 12"; `PluginConfigSearch.tsx` defaults panel.
**Addresses:** FEATURES table-stakes "Find/search in scrollback".
**Avoids:** PITFALLS #6 (search lock on large scrollback — performance test with 10k-line fixture before merge), #7 (Cmd-F vs browser-find conflict — focus-conditioned `preventDefault`, only when xterm DOM is `document.activeElement`).
**Web search UI deferred** to a future phase (addon ships vendored; the input bar is desktop-only in v3.2).

### Phase 95: Web-Links Addon + Security Hardening (LOAD-BEARING SECURITY GATE)
**Rationale:** highest-impact security risk in v3.2; treat with v3.1's WS-Origin-allowlist rigor. Isolating this phase makes the security review crisp and gives the click-confirmation UX dedicated polish time.
**Delivers:** `@xterm/addon-web-links@0.12.0` vendored + wired; platform-aware click handler (Wails `BrowserOpenURL` on desktop; `window.open(_, '_blank', 'noopener,noreferrer')` on web); click-confirmation popover with full resolved URL; OSC 8 href display on hover (always actual href, never display text); strict scheme allowlist (`https`, `http`, `mailto`); IDN/Punycode warning; `PluginConfigWebLinks.tsx` with Cmd/Ctrl-click modifier dropdown.
**Addresses:** FEATURES table-stakes "Clickable web links".
**Avoids:** PITFALLS #8 (phishing — *the* defining gate of this phase), #9 (platform click conventions).
**Verification:** OSC 8 spoof fixture test; mid-trip-punctuation regex test (`(https://example.com)`); platform UAT on Wails macOS/Linux/Windows + web Chrome/Safari/Firefox + iPad Safari long-press.

### Phase 96: Image Addon + CSP Audit (LOAD-BEARING SECURITY/PERFORMANCE GATE)
**Rationale:** heaviest addon (62 KB ESM), highest CSP risk, most novel feature; isolate so phase verification can do dedicated CSP UAT and memory budgeting without bundling other risk. Spawn researcher first to audit `addon-image.js` source for `URL.createObjectURL`/`blob:`/dynamic-Worker construction.
**Delivers:** `@xterm/addon-image@0.9.0` vendored + wired with `storageLimit: 16` MB override (drop from upstream default 100); `enableSizeReports: false`; `pixelLimit: 16_000_000`; Settings → Plugins → Inline Images → Advanced reveal exposing `storageLimit`; CSP amendment if research finds blob/worker usage; multi-client image replay regression test (client B joins after client A renders sixel — confirms relay byte-fidelity).
**Uses:** `@xterm/addon-image@0.9.0` — sixel WASM embedded as base64 inside the bundle (no external `.wasm`).
**Addresses:** FEATURES differentiator "Inline images".
**Avoids:** PITFALLS #3 (storage bomb — 16 MB cap; expose in Settings; tighter on web), #4 (CSP `worker-src` — research at phase start; amend if needed; re-run e2e on Chromium + Safari + Firefox), #5 (multi-client replay — audit `internal/relay/` for byte-level filtering; sixel must survive scrollback replay).
**Open question:** does `addon-image` construct `blob:` workers? Researcher answers before any wiring work begins.

### Phase 97: Serialize Addon + Save-Session UX
**Rationale:** purely additive; passive reader; the surrounding UX (save dialog, file naming, mime type) is the most app-shell-flavored work — benefits from being implemented after the toggle/config pattern is proven by 5 prior phases.
**Delivers:** `@xterm/addon-serialize@0.14.0` vendored + wired; "Save Terminal As…" tab right-click action via Wails `SaveFileDialog`; text-only output in v3.2 (HTML output deferred — theme-aware, more complex); explicit Settings UI tooltip warning that "Serialize captures all visible terminal text including any secrets, tokens, or sensitive data printed in this session."
**Addresses:** FEATURES differentiator "Buffer serialize as a first-class user gesture".
**Avoids:** PITFALLS #12 (PII/secrets exfil — no on-disk auto-save; no automatic crash-recovery serialization in v3.2; explicit warning copy; default toggle ON for the addon-as-library but no user-facing "auto-save" surface).
**Scope discipline:** ship as a manual user action only. Future session-restore work is a separate decision with encrypted-at-rest storage requirements.

### Phase 98: Optional — Progress Addon (OSC 9;4) + Tab/Tray Surfacing
**Rationale:** tiny (1.4 KB), uniquely good fit for AgentHub's AI-CLI audience; differentiates from VS Code/iTerm2/Hyper which don't surface OSC 9;4 per-tab. P2 — ship if budget allows after Phases 92–97 stabilize.
**Delivers:** `@xterm/addon-progress@0.2.0` vendored + wired; `progressAddon.onChange` subscriber emits Wails event; `TabBar.tsx` renders a subtle progress underline per tab; tray icon shows aggregate quartile glyph (1/4 / 2/4 / 3/4); Settings toggle (default OFF in v3.2, can flip ON in v3.3 once validated).
**Addresses:** FEATURES differentiator "OSC 9;4 progress".
**Avoids:** scope creep — explicitly P2; if Phase 96 or 95 over-runs, this is the cuttable phase.

### Phase 99: Settings UI Polish + Migration + Final CSP Audit (release gate)
**Rationale:** final integration phase; bundle audit; cross-browser e2e CSP; "applies to new sessions" affordance polish; settings.json migration test; release gate before v3.2 ships.
**Delivers:** Settings → Plugins section finalized with all 7 toggles (6 from Phases 93–97 + optional Progress); per-plugin descriptions + advanced disclosures; "applies to new sessions" italic caption + post-toggle BannerStack on unicode11/image; `schemaVersion: 2` written; migration test green; cold-cache 3G first-paint benchmark vs. budget; e2e CSP zero-violation suite green on Chromium + Safari + Firefox; iPad Safari Tailscale UAT; software-WebGL detection telemetry hidden line in Settings ("Renderer: WebGL2 hardware / software / DOM").
**Addresses:** FEATURES table-stakes "Settings toggle for each plugin", "'Applies to new sessions only' indicator".
**Avoids:** PITFALLS #11, #13 (UI affordance), #14 (migration), #19 (bundle bloat — lazy-load image if budget exceeded), #20 (per-addon CSP review checklist).

### Phase Ordering Rationale

- **Foundation before features (92 first):** every later phase needs `pluginConfig` plumbing and the migration story. Validating the daemon→Wails→event→React pipeline with no addon work means addon phases are pure addon work.
- **Migrate-don't-add second (93):** the three already-loaded addons (webgl, unicode11, clipboard) become the shakedown for the reconcile pattern. The web-parity work (vendoring webgl + unicode11 for the first time) lives here too — same vendoring discipline, same drift test extension, applied once.
- **Cheapest new addon third (94):** search proves the per-plugin-config UI flow with the lowest CSP/security risk before the security-critical phases.
- **Security gate fourth (95):** web-links is the highest-impact security work. Isolating it gives the click-confirmation UX dedicated polish time and the security review a crisp scope (this phase's PR is the v3.2 equivalent of v3.1's WS Origin allowlist PR).
- **CSP/performance gate fifth (96):** image is the heaviest addon and the only one that might require CSP amendment. Isolated for dedicated CSP UAT + multi-client replay test + memory budgeting.
- **Stand-alone sixth (97):** serialize has no addon dependencies but the most app-shell-flavored UX work. After the toggle/config pattern is proven 5 times, this phase can focus on the save-dialog UX without competing concerns.
- **Optional polish seventh (98):** progress is cuttable if 95 or 96 over-runs. Bundles small but the tab/tray plumbing is real work.
- **Release gate last (99):** final cross-browser CSP audit + bundle audit + Settings UI polish + migration verification. The "is this ready to ship" phase.

**Parallelizability:** Phases 92 + 93 are blocking for everything else. Phases 94 + 95 + 97 can run in parallel if multiple researchers/implementers are available. Phase 96 is independent but should not run before Phase 93 completes (vendoring discipline). Phase 99 is a sequential release gate.

### Research Flags

Phases likely needing deeper research during planning (`/gsd-research-phase`):
- **Phase 95 (Web-Links security):** OSC 8 attack surface; IDN/Punycode detection; click-confirmation UX prior art (browsers' "this link looks suspicious" patterns); test fixtures for typosquats. Treat security review with v3.1 rigor.
- **Phase 96 (Image CSP):** **Mandatory pre-phase research** to read `frontend/node_modules/@xterm/addon-image/lib/addon-image.js` source and grep for `URL.createObjectURL`/`new Worker(`/`blob:`/`data:`. Findings determine whether CSP amendment is needed. Also: multi-client relay byte-fidelity audit of `internal/relay/`.
- **Phase 99 (Final integration):** cold-cache first-paint budgeting methodology; Safari + Firefox CSP e2e harness setup (existing v3.1 suite is Chromium-only); iPad Safari Tailscale UAT script.

Phases with standard patterns (skip dedicated research-phase):
- **Phase 92:** standard daemon-settings extension pattern; precedent in Phase 79 (settings persistence) and Phase 82 (startMinimized).
- **Phase 93:** vendoring pattern established by Phase 89; reconcile pattern is straightforward refactor.
- **Phase 94:** addon-search has stable API; find-bar UI has VS Code as reference.
- **Phase 97:** addon-serialize is a thin library; main work is Wails save dialog wiring (existing pattern).
- **Phase 98:** addon-progress has documented OSC 9;4 spec; tab-strip + tray are existing surfaces.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Versions and bundle sizes verified against live npm registry 2026-05-03; tarballs unpacked locally; CSP/vendoring constraints validated against v3.1 Phase 89 actual source |
| Features | HIGH | Official xterm.js README + typings + npm metadata (HIGH); competitor comparisons MEDIUM (public docs + comparison articles dated). Default-on/off decisions and "what good looks like" per-plugin grounded in upstream + competitor practice |
| Architecture | HIGH | Verified against actual repo files — `TerminalPanel.tsx` addon loading shape, `daemonSettings` struct in `engine.go`, CSP in `csp_mw.go`, vendor-drift contract in `vendor_drift_test.go`, embed shape in `embed.go`. Hot-swap matrix from xterm.js lifecycle reasoning + addon source inspection |
| Pitfalls | HIGH for addon-specific behavior (verified against upstream addon repos and known issues); HIGH for AgentHub-specific architecture (read directly from `.planning/PROJECT.md` and v1.6/v2.0/v3.1 milestones); MEDIUM for Safari/Tailscale-iOS WebGL software-rendering claims (single-source) |

**Overall confidence:** HIGH. The stack is verified, the integration shape is verified against actual files, the security risks are concrete and sourced. The two MEDIUM uncertainties (addon-image CSP behavior; Safari WebGL software detection) are scoped to specific phases (96 and 93 respectively) where they will be resolved by phase-time research.

### Gaps to Address

Consolidated open questions from all four researchers — these need decisions before or during roadmap planning:

- **Addon-image `blob:`/worker behavior** (PITFALLS #4, #20; ARCHITECTURE §6.2). Resolved by mandatory pre-Phase-96 source inspection. **Decision needed:** does CSP need `worker-src 'self' blob:`? Researcher answers in Phase 96.
- **Default-enabled set** (ARCHITECTURE §11 open question 2). **Recommendation:** ship all 7 ON (image included) except optional `addon-progress` which is OFF in v3.2 and flips ON in v3.3 after validation. Image being default-ON matches the "well-established sixel/IIP protocol" precedent and gives the differentiator visibility.
- **Per-client vs server-shared plugin config for web-served sessions** (PITFALLS #15; ARCHITECTURE §3.2). **Recommendation:** server-shared for buffer-interpretation plugins (unicode11 — must match across clients to avoid scrollback divergence); per-client localStorage opt-out path for renderer-only choices (webgl, image rendering capability). Decision must land in Phase 92 design (Settings UI surface differs).
- **Search keybind on web-served:** browser-find vs xterm-search collision. **Recommendation:** focus-conditioned `preventDefault` (only when xterm DOM is `document.activeElement`); add visible search affordance; offer `/` as alternative. Land in Phase 94.
- **Where the "applies to new sessions" indicator renders:** inline next to toggle, or section-level note? **Recommendation:** inline italic caption per affected toggle (matches "READ ONLY" badge pattern from existing web terminal; aligns with VS Code precedent). Plus one-shot post-toggle BannerStack confirmation. Land in Phase 92 design, polish in Phase 99.
- **Serialize HTML output in v3.2 or defer:** `serializeAsHTML()` is theme-aware and more complex than `serialize()`. **Recommendation:** v3.2 ships text-only "Save Terminal As…"; HTML output is a v3.3 follow-up (tracked, not committed).
- **Progress addon as P1 or P2:** STACK and FEATURES recommend P2 (cuttable); ARCHITECTURE skips it. **Recommendation:** P2 in Phase 98, ship if Phases 95 or 96 don't over-run. Default OFF if shipped (validate in field before flipping ON).

## Sources

### Primary (HIGH confidence)
- npm registry verified 2026-05-03 via `npm view` — all `@xterm/addon-*` versions, peer-deps, dependencies
- Tarball inspection 2026-05-03 — `addon-image-0.9.0.tgz`, `addon-search-0.16.0.tgz`, etc. locally unpacked; bundle sizes measured; sixel WASM confirmed embedded as base64 (no external `.wasm` file)
- Local repo files (verified directly): `frontend/package.json`, `frontend/pnpm-lock.yaml`, `frontend/src/components/TerminalPanel.tsx`, `internal/daemon/engine.go`, `internal/webserver/csp_mw.go`, `internal/webserver/vendor_drift_test.go`, `internal/webserver/no_cdn_regression_test.go`, `web/embed.go`, `web/terminal.html`, `web/assets/terminal.js`, `web/vendor/xterm/VERSION`, `app.go`, `.planning/PROJECT.md`, `.planning/MILESTONES.md`
- Official xterm.js documentation: Using addons guide, Link Handling guide, addon-webgl README (context-loss), addon-search/addon-serialize typings, addon-image upstream (jerch/xterm-addon-image)
- Standards: OSC 9;4 (rockorager.dev), iTerm2 IIP, OSC 8 phishing primitives (egmontkob gist), Khronos WebGL HandlingContextLost
- GitHub Issue #36

### Secondary (MEDIUM confidence)
- xterm.js issue threads: #4753 (unicode 11 actually 12), #3304 (graphemes), #5176/#4902 (search slow on wrapped scrollback), jerch/xterm-addon-image#47 (images aren't serialized)
- VS Code terminal docs + microsoft/vscode#15211 (gpu acceleration setting)
- Image tooling: chafa, Are We Sixel Yet?
- Competitor comparisons (dated): Warp vs iTerm2; Terminal Trove 2026

### Tertiary (LOW confidence)
- iPad Safari WebGL software-rendering behavior — single-source claim; needs Phase 93 UAT
- Tailscale-cellular first-paint budget impact — extrapolated from bundle sizes; needs Phase 99 benchmark

---
*Research completed: 2026-05-03*
*Ready for roadmap: yes*
