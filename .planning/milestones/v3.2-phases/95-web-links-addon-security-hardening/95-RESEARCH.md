# Phase 95: Web-Links Addon + Security Hardening — Research

**Researched:** 2026-05-06
**Domain:** xterm.js `@xterm/addon-web-links`, OSC 8 hyperlinks, IDN/Punycode + typosquat detection, Wails `BrowserOpenURL`, `window.open` with `noopener,noreferrer`, live Settings toggle reuse from Phase 93/94
**Confidence:** HIGH

---

## Summary

Phase 95 ships clickable URLs in terminal output with v3.1-style security rigor. The upstream `@xterm/addon-web-links@0.12.0` provides the registration mechanism (`Terminal.registerLinkProvider` under the hood) and OSC 8 escape-sequence parsing — but its **default scheme allowlist is too permissive** for our threat model. Phase 95 must NOT use `WebLinksAddon` in its default form; it must construct it with an explicit `handler` callback that (1) refuses to open anything outside `{https, http, mailto}` and (2) routes through the platform-correct opener (`BrowserOpenURL` on desktop, `window.open(url, '_blank', 'noopener,noreferrer')` on web). Hover tooltips, modifier-key gating, and the click-confirmation popover are layered on top via a custom `linkProvider` (the WebLinksAddon's matcher gives URL detection; the activation/hover handlers give us the security gates). Phase 95 also introduces the **first user-facing security UI in v3.2** — a confirmation popover for OSC 8 mismatches, IDN/Punycode URLs, and known typosquat patterns. All seven success criteria from the ROADMAP map cleanly onto well-understood xterm.js link-provider primitives, plus one new React component (`LinkConfirmPopover.tsx`) and one new utility module (`lib/urlSafety.ts`).

**Primary recommendation:** Install `@xterm/addon-web-links@^0.12.0` via pnpm and vendor `lib/addon-web-links.js` to `web/vendor/xterm/addons/` (Phase 93 pattern, byte-identical). Construct `WebLinksAddon` with a custom `handler` and `options.urlRegex` — but for OSC 8 specifically, register a **second** custom link provider via `term.registerLinkProvider()` so the display-text-vs-href divergence can be surfaced in the hover tooltip and confirmation popover. Add a `WebLinksConfig` nested struct to `daemon.PluginSettings` mirroring Phase 94's `SearchConfig` precedent. Reuse Phase 93's hot-swap useEffect dep-array slicing in `TerminalPanel.tsx` to live-attach/dispose the addon. Reuse Phase 93's `applyPluginConfig` diff path in `web/assets/terminal.js`. The opener decision is environmental, not configurable: `window.go` (Wails) → `BrowserOpenURL`; otherwise → `window.open(url, '_blank', 'noopener,noreferrer')`. **Never inline-navigate via `location.href = url`.**

---

## Project Constraints (from CLAUDE.md)

- **JS/TS:** `camelCase` vars, `PascalCase` components, ESLint + Prettier, TypeScript types — applies to `LinkConfirmPopover.tsx`, `lib/urlSafety.ts`, hot-swap extensions in `TerminalPanel.tsx`.
- **Node:** `pnpm` (project default). Add `@xterm/addon-web-links` via `pnpm add @xterm/addon-web-links@^0.12.0` (matches Phase 94 install pattern).
- **Go:** `go fmt`, context-aware functions. Applies to daemon struct extension (`WebLinksConfig`).
- **No global npm installs.**
- **NEVER kill node.exe.**
- **LSP first** for code navigation — applies to discovering existing `BrowserOpenURL` call sites in `App.tsx`, `SettingsTab.tsx`, `SessionSharePanel.tsx`, `UpdateBanner.tsx`.
- **UAT via dev-browser skill** for browser-based verifications (the click-confirmation popover, OSC 8 hover, web-served `_blank` behavior).

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LNK-01 | Plain `https://`, `http://`, `mailto:` clickable; `file://`, `javascript:`, custom schemes never clickable by default | `WebLinksAddon` constructor accepts an `options.urlRegex` to scope detection; we ALSO check the scheme inside the `handler` (defense-in-depth, since the regex alone could be bypassed by an attacker-crafted regex-evading string). Single source of truth: `lib/urlSafety.ts isAllowedScheme()`. Scheme allowlist is HARDCODED — no Settings option to expand it (REQUIREMENTS `## Out of Scope`). |
| LNK-02 | Cmd-click (mac) / Ctrl-click (Linux/Windows) to activate; configurable in Settings; single-click never activates by default; hover tooltip shows resolved href | xterm's link provider receives the `MouseEvent` for both hover and click; check `event.metaKey` (mac) / `event.ctrlKey` (other) — return early when modifier missing. Hover tooltip: implemented in `LinkProvider.hover()` with native `title` attribute or a custom DOM tooltip. Settings switch: new `WebLinksConfig.modifier: 'platform' \| 'cmd' \| 'ctrl' \| 'none'` field on PluginSettings. |
| LNK-03 | OSC 8 mismatch (display ≠ href), IDN/Punycode URLs, typosquat patterns trigger click-confirmation popover showing full resolved URL before navigation | Three detectors in `lib/urlSafety.ts`: `osc8Mismatch(displayText, href)`, `hasIDN(href)` (looks for `xn--` or non-ASCII chars in the host), `isTypoSquat(host)` (small static list of `paypa1.com`-class tokens). Activation handler invokes `LinkConfirmPopover` with the resolved URL + reason; popover Continue button calls the platform opener; Cancel dismisses. |
| LNK-04 | Desktop: routes through Wails `BrowserOpenURL`; web: `window.open(url, '_blank', 'noopener,noreferrer')`; never current-tab navigation | Environment detection: `typeof window.go !== 'undefined' && typeof window.runtime?.BrowserOpenURL === 'function'` → desktop. Single helper `lib/openLink.ts openLink(url)` is the only function in the codebase that opens a URL from a terminal link. Regression test: grep ban on `location.href =` and `window.location =` in any link-handler code path. |
| LNK-05 | Settings toggle `webLinks` + sub-config (`modifier`, `confirmOSC8`, `confirmIDN`, `confirmTyposquat`) persists; live-applies to all open terminals; already-rendered links update on next refresh without session restart | Reuse Phase 94's `SetSearchConfig` sub-key RPC pattern verbatim: new `(*App).SetWebLinksConfig(cfg)` Wails method + `PATCH /settings/web-links-config` HTTP route; emits `settings:plugins` event identical to existing path. Hot-swap in `TerminalPanel.tsx` extends Phase 93/94 dep array with `pluginConfig?.webLinks`. The "next refresh" semantics ARE the natural addon behavior — disposing + reconstructing the addon re-scans the visible buffer on the next write/render. |
| LNK-06 | (Same as LNK-05; the ROADMAP success criteria 5 maps to this REQ) | Covered by LNK-05 above. |
</phase_requirements>

---

<user_constraints>
## User Constraints

> No `95-CONTEXT.md` will be authored. Per [skip-discuss-when-research-complete] feedback memory: when ROADMAP/REQUIREMENTS/research already pre-answer the gray areas, skip `/gsd-discuss-phase`. ROADMAP success criteria + REQUIREMENTS LNK-01..LNK-06 leave only mechanical questions. STATE.md captures the cross-cutting decision below.

### Locked Decisions

- **Phase 95 owns clickable links for BOTH desktop and web.** [STATE.md `## Decisions`, Phase 95 entry] Same scope shape as Phase 94. Confidence: HIGH.
- **Phase 95 is treated with v3.1-WS-Origin-allowlist rigor.** [STATE.md] Scheme allowlist is `{https, http, mailto}` with no user override. OSC 8 / IDN / typosquat detection ON by default. Click-confirmation popover for any of those three triggers.
- **Default modifier is platform-correct: `Cmd` on macOS, `Ctrl` on Linux/Windows.** [ROADMAP SC-2] User-configurable via Settings (LNK-02). Default value `'platform'` in `WebLinksConfig.modifier`.
- **Default ON.** [STATE.md ROADMAP `## Decisions`, "ship all 7 plugins ON by default except optional `addon-progress`"] PluginSettings.WebLinks defaults to `true` (already true on disk per Phase 92's `defaultPluginSettings()` return value — verified in `internal/daemon/plugin_settings.go:58`).
- **Vendored same-origin under `web/vendor/xterm/addons/addon-web-links.js`.** [STATE.md ROADMAP, Phase 93 vendoring discipline] Phase 93 pattern applies verbatim. No CDN.
- **Single Settings toggle + nested sub-config (no Phase 99 split).** Phase 99 / PUI-03 owns the `<details>` advanced disclosure under the toggle (modifier choice + confirmation policy checkboxes). Phase 95 ships the BOOLEAN toggle only — sub-config defaults are baked in code; `<details>` UI is deferred. Confidence: HIGH (matches Phase 94 PUI-03 deferral).
- **Hover tooltip displays the actual resolved href in real-time.** [ROADMAP SC-2] Including OSC 8 hyperlinks where display text ≠ href. Implementation: hover shows the URL the click would actually open (resolved against base URL if relative, but: relative URLs in terminal output are pathological — treat them as plain text, do not link).
- **Reduced-motion respected on popover (no slide-in if `prefers-reduced-motion: reduce`).** [Inherited from Phase 93/94 pattern.]

### Claude's Discretion

- Whether to use `WebLinksAddon` for the URL regex matching, or roll a fresh `term.registerLinkProvider` call. **Recommendation:** use `WebLinksAddon` for plain-text URL detection (its regex is well-tuned for terminal output) and supplement with a SECOND link provider for OSC 8 (which `WebLinksAddon` does NOT handle in its current version — see "SearchAddon API Contract" section).
- Whether `LinkConfirmPopover` is portal-rendered or in-flow. **Recommendation:** portal to `document.body` to escape the terminal container's overflow:hidden. Fixed positioning anchored at the click coordinates.
- Whether to debounce hover-tooltip rendering. **Recommendation:** native `title` attribute first; only build a custom tooltip if the native one feels laggy in dev-browser UAT.
- Static typosquat list contents. **Recommendation:** small, confidence-high list (~30 entries: `paypa1.com`, `goog1e.com`, `arnazon.com`, `microsft.com`, `app1e.com`, `git-hub.com`, `tw1tter.com`, etc.). Frame as "common typosquat suffix patterns" not "comprehensive". Document that this is best-effort, not a security boundary.
- How to detect IDN. **Recommendation:** `URL(href).hostname.includes('xn--')` (post-Punycode) OR check for non-ASCII codepoints in the original `href.match(/^https?:\/\/([^/]+)/)[1]`. Use both — Punycode-already-encoded AND Unicode-form both trigger.

### Deferred Ideas (OUT OF SCOPE)

- Custom URL protocol allowlists (`file://`, `vscode://`, custom schemes). [REQUIREMENTS `## Out of Scope`] Future opt-in possible but not now.
- `<details>` advanced disclosure for `WebLinksConfig` sub-fields (modifier choice, per-detector confirmation toggles). [Phase 99 / PUI-03]
- Comprehensive typosquat database (e.g., maintaining a remote-fetched list of millions of confusable domains). Phase 95 ships a small static list; "good enough" not "complete".
- Hover-href-mismatch warning glyph in-line with the link (red underline, etc.). [Phase 99 polish candidate or future] Phase 95 surfaces mismatch only at click time via the popover.
- Telemetry / logging of suspicious click events. **Privacy-by-default:** clicks never reported anywhere. The popover is the only feedback channel.
- Webhook / ACL integration with Tailscale identity for risky-link policy. Out of scope for v3.2.

</user_constraints>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| URL detection in plain terminal output | Browser / Wails WebView (desktop); Browser / web page (web) | — | xterm `WebLinksAddon` lives where the Terminal instance lives; pure browser-tier concern |
| OSC 8 hyperlink parsing | Browser (xterm.js core handles the escape sequence; we register a second link provider) | — | xterm.js `Terminal` has built-in OSC 8 support; surfaces via `term.registerLinkProvider` |
| Hover tooltip showing resolved href | Browser | — | Native `title` or custom DOM; no daemon involvement |
| Modifier-key click gating | Browser | — | `MouseEvent.metaKey` / `ctrlKey` checked in link handler |
| OSC 8 mismatch / IDN / typosquat detection | Browser (`lib/urlSafety.ts`) | — | Pure functions; no network; no daemon |
| Click-confirmation popover | Browser (`LinkConfirmPopover.tsx` desktop, plain DOM popover element on web) | — | Pure UI; no daemon involvement |
| Link activation routing | Browser, then either Wails IPC (desktop) or `window.open` (web) | Native browser opener (which delegates to OS default browser via Wails) | Single helper `lib/openLink.ts`; Wails RPC or `window.open` chosen by environment-detection |
| `WebLinksConfig` persistence | API / Backend (daemon) | Wails RPC + Phase 93 SSE broadcast | Reuses Phase 92/93/94 pipeline unchanged; nested struct under `PluginSettings` |
| `pluginConfig.webLinks` propagation desktop | App.tsx state → `pluginConfig` prop drill into TerminalPanel | — | Phase 92 pipeline; reused unchanged |
| `pluginConfig.webLinks*` propagation web | `/api/plugin-config` GET + `/api/plugin-config/stream` SSE | — | Phase 93 endpoints; new field flows through automatically |
| Vendored addon serving | CDN / Static (embedded) | — | `web/vendor/xterm/addons/addon-web-links.js` served via Go embed.FS at `/assets/xterm/addons/addon-web-links.js` |
| `vendor_drift_test.go` CI gate | CI / Go test | — | Already generalized in Phase 93; min-count guard bumps from 6 to 7 |

**Cross-tier note for LNK-04:** the daemon and Go webserver are not involved in the click path. Wails `BrowserOpenURL` is a Wails IPC into the Go App layer that calls the OS default browser opener — but our daemon code does not see the URL. On web, `window.open` is browser-native. **Search queries / link activations never traverse the network beyond the browser-native open.** Privacy by default.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/addon-web-links` | `^0.12.0` | URL regex matching in terminal output, link provider registration, OSC 8 plumbing (in xterm.js core, not the addon) | First-party `@xterm` scoped addon; drop-in compatible with the project's `@xterm/xterm@^6.0.0` core; same family as already-shipped `@xterm/addon-fit`, `@xterm/addon-webgl`, etc. |

**Verified:** `npm view @xterm/addon-web-links version` returned `0.12.0` on 2026-05-06. `main: lib/addon-web-links.js` (CJS UMD bundle — correct file for web vendoring; the `.mjs` file is ES-module-only and would require `<script type="module">` which conflicts with the existing UMD-via-`<script>` pattern). [VERIFIED: npm registry, 2026-05-06 via `npm view @xterm/addon-web-links main`]

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@heroicons/react/24/outline` | (existing) | ExclamationTriangleIcon for the popover risk indicator | UI consistency with Phase 93 / 94 (same icon family) |
| `@heroicons/react/20/solid` (XMarkIcon) | (existing) | Popover close button | Matches Phase 93 `WebGLRecoveryBanner` + Phase 94 `FindBar.close` precedent |
| (none for IDN) | — | Pure-JS Punycode detection via `URL` parser + `xn--` prefix check + Unicode codepoint scan | Native `URL` interface handles parsing; no library needed |
| (none for typosquat) | — | Static array of confusable strings | Library overhead unjustified for a 30-entry list |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `@xterm/addon-web-links@0.12.0` | Hand-rolled `term.registerLinkProvider` with custom URL regex | Reinventing the URL-matching regex (which `WebLinksAddon` does well); we still register a second link provider for OSC 8 because the addon doesn't currently expose OSC 8 hooks. Use the addon for plain text; supplement for OSC 8. |
| Native `title` attribute for hover tooltip | Custom DOM tooltip with positioning | Native is free, accessible, screen-reader-supported; custom would need ARIA + positioning logic. **Recommendation:** start native; upgrade only if dev-browser UAT shows the native tooltip is too laggy or hides the URL prematurely. |
| `idna-uts46` library for IDN detection | Hand-rolled Punycode + non-ASCII check | Library is ~30KB minified; we only need a boolean (is this IDN?), not full conversion. Native `URL.hostname` exposes the Punycode `xn--` form directly; non-ASCII check on the original href catches the Unicode form. **Hand-roll.** |
| `tldts` or `psl` for typosquat detection | Static list of ~30 known patterns | Library is heavy (10MB+ TLD database); typosquat detection is a fuzzy heuristic anyway. Static list is honest about its limitations. |
| Setting `event.preventDefault()` on every clickable URL render | Letting browser-native middle-click → "open in new tab" work | xterm renders into a canvas (or DOM); the link provider mechanism INSERTS its own click handlers, so middle-click is ALREADY a no-op without our intervention. Modifier-click semantics are entirely controlled by our handler. |

**Installation:**

```bash
cd frontend && pnpm add @xterm/addon-web-links@^0.12.0
```

After install, copy `frontend/node_modules/@xterm/addon-web-links/lib/addon-web-links.js` to `web/vendor/xterm/addons/addon-web-links.js` (Phase 93 pattern, byte-identical to source).

**Version verification:**
```bash
npm view @xterm/addon-web-links version  # confirmed 0.12.0 on 2026-05-06
```

[VERIFIED: npm registry, 2026-05-06]

---

## WebLinksAddon + Link Provider API Contract

### Constructor

```typescript
// [VERIFIED: @xterm/addon-web-links typings on npm registry, 2026-05-06]
new WebLinksAddon(
  handler?: (event: MouseEvent, uri: string) => void,
  options?: ILinkProviderOptions,
  useLinkCodes?: boolean
)
```

- `handler`: callback invoked when a detected link is **clicked** (with whatever modifier-discipline xterm core enforces — `Alt`/`Cmd`/`Ctrl` historically; recent versions just emit click + leave modifier handling to the handler).
- `options`: includes `urlRegex`, `hover` (callback for mouseover), `leave` (callback for mouseleave), `tooltipCallback`, `urlTransform`, `priority`.
- `useLinkCodes`: legacy parameter; default false.

### Default Behavior (without our override)

The addon's default `handler` calls `window.open(uri, '_blank')` — **MISSING `noopener,noreferrer`**, AND the default URL regex permits any `https?://` and `mailto:` URL but does **NOT explicitly exclude** `javascript:`, `file://`, or other schemes. We MUST construct with a custom handler that gates by scheme.

[VERIFIED: source-read of `@xterm/addon-web-links@0.12.0/lib/addon-web-links.js` patterns documented in xtermjs/xterm.js GitHub repository.]

### Lifecycle

```typescript
public activate(terminal: Terminal): void;  // called by term.loadAddon()
public dispose(): void;                       // called on unmount or toggle-off
```

No `clear` method — disposing the addon detaches all matchers.

### OSC 8 hyperlinks

[VERIFIED: xterm.js issue tracker + `Terminal.registerLinkProvider` typings] — OSC 8 escape sequences (`ESC ] 8 ; params ; URI ESC \`) are parsed by xterm.js core into "hyperlink ranges" within the buffer, and exposed via `IBufferLine.getCell(...).getHyperlinkId()`. The `WebLinksAddon` 0.12.0 does NOT register an OSC 8 link provider out of the box — it only matches plain text URLs via regex.

To handle OSC 8 with display-text-vs-href divergence:

```typescript
// [Pattern based on xtermjs/xterm.js Issue #4135 + ITerminalAddon authoring docs]
const osc8Provider: ILinkProvider = {
  provideLinks(bufferLineNumber, callback) {
    const line = term.buffer.active.getLine(bufferLineNumber - 1)
    if (!line) { callback(undefined); return }
    const links: ILink[] = []
    for (let col = 0; col < line.length; col++) {
      const cell = line.getCell(col)
      if (!cell) continue
      const hyperlinkId = cell.getHyperlinkId?.()  // returns undefined if no OSC 8
      if (hyperlinkId) {
        // ... walk forward to find end of run, collect display text + href ...
        // emit ILink with `text` (display) and `data` (href) split correctly
      }
    }
    callback(links)
  }
}
term.registerLinkProvider(osc8Provider)
```

**Practical reality:** xterm.js's `IBufferCell.getHyperlinkId()` returns a numeric ID that maps to a URL in the terminal's hyperlink registry. The xterm core attaches a tooltip + click handler when OSC 8 is rendered, but there is no direct public API to read the URL by ID. **We may need to work around this** by parsing OSC 8 sequences on the daemon side (write-stream interception), OR by registering our link provider EARLIER than the core's internal handler so we can match at the cell level using a regex over display text + correlate with the recently-emitted OSC 8 sequence.

**Recommendation:** Plan-phase task should include a 30-minute spike to verify exactly what `getHyperlinkId()` exposes and whether `term.parser.registerOscHandler(8, ...)` (the OSC handler registration API) is accessible. If neither is sufficient, fall back to plain-text-URL-only support in v3.2 and DEFER OSC 8 mismatch detection to v3.3. **Document this as `OPEN: spike-required` in the plan.** [ASSUMED: `term.parser.registerOscHandler` exists and accepts a callback — verify in Wave 0 spike.]

### Search Methods (n/a)

WebLinksAddon has no search methods — it's a passive matcher.

### Performance Envelope

URL regex matching runs on every line write + viewport scroll. With xterm's 10,000-line scrollback cap and the addon's internal viewport-only matching strategy (matches are only computed for lines in the visible viewport, not the whole scrollback at once), CPU cost is negligible. **No worker/debounce needed.**

### Default URL Regex (verified upstream)

```javascript
// [VERIFIED: @xterm/addon-web-links@0.12.0 source]
/(https?:\/\/|mailto:)(?:[^\s\\\]"'\<\>{}|^`])+/g
```

This regex matches `https?://` and `mailto:` schemes — but NOT `javascript:`, `file://`, `data:`, or `vbscript:`. **At first glance it appears safe.** However, an attacker can craft a URL where the regex match starts at position N but the rendered href (e.g. via OSC 8) starts at position M, with `javascript:` in between. Our defense-in-depth handler MUST re-validate the scheme inside the click handler, not rely solely on the regex.

```javascript
// Defense-in-depth: regex AND handler both gate scheme.
function handler(event, uri) {
  if (!isAllowedScheme(uri)) return  // refuse silently — never open
  if (!isModifierPressed(event)) return  // not modified click — refuse
  // ... continue with confirmation popover or direct open ...
}
```

---

## Architecture Patterns

### System Architecture Diagram

```
                  Terminal output stream (PTY → xterm.write)
                                  │
                                  ▼
                  ┌─────────────────────────────────────┐
                  │ xterm.js core parses bytes:         │
                  │  • plain text                        │
                  │  • OSC 8 hyperlink runs              │
                  │    (display ≠ href)                  │
                  └─────────────────────────────────────┘
                                  │
                  registered link providers (priority-ordered)
                                  │
              ┌───────────────────┴───────────────────┐
              ▼                                       ▼
    ┌───────────────────┐                    ┌───────────────────┐
    │ WebLinksAddon     │                    │ OSC8LinkProvider  │
    │ (plain-text URL   │                    │ (display-vs-href  │
    │  regex match)     │                    │  detection)       │
    └───────────────────┘                    └───────────────────┘
              │                                       │
              └───────────────────┬───────────────────┘
                                  │
              user hovers a link  │
                                  ▼
                  ┌───────────────────────────────────┐
                  │ Hover handler:                    │
                  │  set tooltip text = resolved href │
                  │  (NOT the display text)           │
                  └───────────────────────────────────┘
                                  │
              user clicks         │
                                  ▼
                  ┌───────────────────────────────────┐
                  │ Click handler:                    │
                  │  1. Modifier check                │
                  │     pluginConfig.webLinksConfig.  │
                  │     modifier resolved against     │
                  │     navigator.platform            │
                  │  2. Scheme allowlist              │
                  │     (https/http/mailto only)      │
                  │  3. Risk detectors                │
                  │     - osc8Mismatch?               │
                  │     - hasIDN?                     │
                  │     - isTypoSquat?                │
                  └───────────────────────────────────┘
                          │             │
              none risky  │             │  any risky
                          ▼             ▼
              ┌─────────────┐   ┌──────────────────┐
              │ openLink(url)│  │ Show LinkConfirm │
              └─────────────┘   │ Popover at click │
                          │     │ coords           │
                          │     │  reason: 'osc8'/ │
                          │     │  'idn'/'typo'    │
                          │     └──────────────────┘
                          │              │
                          │       Continue│   Cancel
                          │              ▼      │
                          ▼     ┌─────────────┐ ▼
                  ┌─────────────│  openLink   │ (no-op)
                  │             └─────────────┘
                  ▼
        ┌─────────────────────────────────────┐
        │ openLink(url):                      │
        │                                     │
        │   if (window.runtime?.BrowserOpenURL│
        │     ─ desktop / Wails               │
        │       BrowserOpenURL(url)           │
        │     (opens in OS default browser    │
        │      via Wails IPC)                 │
        │                                     │
        │   else                              │
        │     ─ web / Tailscale-served        │
        │       window.open(url, '_blank',    │
        │         'noopener,noreferrer')      │
        │     (new tab in current browser;    │
        │      `window.opener` is null)       │
        └─────────────────────────────────────┘

  Settings change (live toggle / sub-config update):
       │
       ▼
  ┌────────────────────────────────────────────┐
  │ Phase 94 SetSearchConfig pattern repeats:  │
  │                                            │
  │  SetWebLinksConfig({modifier, confirm*})   │
  │  → Wails (*App).SetWebLinksConfig          │
  │  → daemon engine.SetWebLinksConfig         │
  │  → settings.json + 'settings:plugins'      │
  │  → Phase 93 SSE broadcast                  │
  │                                            │
  │  pluginConfig prop drill → TerminalPanel   │
  │  hot-swap useEffect dep array reads        │
  │  pluginConfig?.webLinks (boolean) for      │
  │  ON/OFF; sub-config (modifier, confirm*)   │
  │  flows via React refs / closure capture    │
  │  read at click time (live, no re-attach    │
  │  needed for sub-config changes).            │
  └────────────────────────────────────────────┘
```

### Recommended Project Structure

**Desktop (React):**
```
frontend/src/
├── components/
│   ├── TerminalPanel.tsx                   # MODIFIED — hot-swap useEffect dep gains pluginConfig?.webLinks; addon ref + dispose
│   ├── LinkConfirmPopover.tsx              # NEW — portal-rendered popover with Continue/Cancel
│   ├── __tests__/
│   │   ├── LinkConfirmPopover.test.tsx     # NEW — copy/aria/keyboard/dismiss
│   │   ├── TerminalPanel.test.tsx          # MODIFIED — hot-swap dep array; modifier-click; scheme allowlist
│   │   └── App.plugin-event.test.tsx       # MODIFIED — webLinks*Config field added to PluginSettings shape
├── lib/
│   ├── urlSafety.ts                        # NEW — isAllowedScheme, osc8Mismatch, hasIDN, isTypoSquat (pure)
│   ├── openLink.ts                         # NEW — single platform-aware opener; the only code that opens a URL
│   └── __tests__/
│       ├── urlSafety.test.ts               # NEW — unit tests with attacker-supplied fixtures
│       └── openLink.test.ts                # NEW — verifies BrowserOpenURL on desktop, window.open on web
├── style.css                               # MODIFIED — add .link-confirm-popover BEM block + .link-confirm-popover--exiting + reduced-motion guard
└── wailsjs/go/models.ts                    # HAND-EDIT — add WebLinksConfig class to daemon namespace; add webLinksConfig field to PluginSettings (Phase 92 pin pattern)
```

**Web (plain DOM):**
```
web/
├── terminal.html                  # MODIFIED — add <div id="link-confirm-popover" hidden> + <script src="/assets/xterm/addons/addon-web-links.js"> BEFORE terminal.js
├── assets/
│   ├── terminal.js                # MODIFIED — applyPluginConfig grows web-links arm; openLink() helper; popover DOM wiring
│   └── terminal.css               # MODIFIED — add #link-confirm-popover styles parallel to desktop
├── vendor/xterm/
│   ├── VERSION                    # MODIFIED — append @xterm/addon-web-links@0.12.0
│   └── addons/
│       └── addon-web-links.js     # NEW — copied from frontend/node_modules/.../lib/addon-web-links.js
└── embed.go                       # MODIFIED — add vendor/xterm/addons/addon-web-links.js to //go:embed
```

**Daemon:**
```
internal/daemon/
├── plugin_settings.go             # MODIFIED — add WebLinksConfig struct + nested field; update defaultPluginSettings()
├── plugin_settings_test.go        # MODIFIED — assert WebLinksConfig defaults
├── engine.go                      # MODIFIED (small) — add SetWebLinksConfig sub-key writer (mirror of SetSearchConfig at engine.go:466-484)
└── api.go                         # MODIFIED — add PATCH /settings/web-links-config route
```

**Wails App:**
```
app.go                             # MODIFIED — add (*App).SetWebLinksConfig (mirror of SetSearchConfig at app.go:505-521)
```

**Go webserver:** **No change to existing code.** Phase 93 `pluginSettingsProvider func() []byte` recurses through `json.Marshal` for the new nested field.

### Pattern 1: WebLinksAddon Hot-Swap with Custom Handler

**What:** Construct `WebLinksAddon` with a custom handler that gates by scheme + modifier + risk detectors. Load/dispose via Phase 93 hot-swap useEffect dep array extension.

**When to use:** Always when `pluginConfig.webLinks === true`.

**Example:**

```typescript
// TerminalPanel.tsx — extension to existing Phase 93/94 hot-swap useEffect
// [Source: pattern matches existing webglAddonRef + clipboardAddonRef + searchAddonRef]

import { WebLinksAddon } from '@xterm/addon-web-links'
import { isAllowedScheme, getRisk } from '../lib/urlSafety'
import { openLink } from '../lib/openLink'

const webLinksAddonRef = useRef<WebLinksAddon | null>(null)

// Inside the existing hot-swap useEffect, ADD a webLinks arm AFTER search arm:
useEffect(() => {
  const term = termRef.current
  if (!term) return

  // ... existing WebGL hot-swap (unchanged) ...
  // ... existing Clipboard hot-swap (unchanged) ...
  // ... existing Search hot-swap (unchanged) ...

  // Web-links hot-swap (Phase 95 LNK-01..06)
  if (pluginConfig?.webLinks) {
    if (!webLinksAddonRef.current) {
      const handler = (event: MouseEvent, uri: string) => {
        // Step 1: scheme allowlist (defense-in-depth — regex AND handler check)
        if (!isAllowedScheme(uri)) return

        // Step 2: modifier-click gate
        if (!isModifierPressed(event, pluginConfig?.webLinksConfig?.modifier ?? 'platform')) return

        // Step 3: risk detection
        const risk = getRisk(uri, /* displayText */ uri /* same for plain-text URLs */)
        if (risk && shouldConfirm(risk, pluginConfig?.webLinksConfig)) {
          setLinkConfirmState({ url: uri, risk, x: event.clientX, y: event.clientY })
          return
        }

        // No risk: open immediately
        openLink(uri)
      }

      const addon = new WebLinksAddon(handler, {
        // urlRegex: undefined → use the addon's default scheme-restricted regex
        hover: (event, uri /*, range */) => {
          // Native title attribute is the simplest accessible tooltip; xterm
          // exposes the link element to us via this callback's `event.target`.
          if (event.target instanceof HTMLElement) {
            event.target.setAttribute('title', uri)
          }
        }
      })
      term.loadAddon(addon)
      webLinksAddonRef.current = addon
    }
  } else {
    if (webLinksAddonRef.current) {
      webLinksAddonRef.current.dispose()
      webLinksAddonRef.current = null
    }
  }
}, [pluginConfig?.webgl, pluginConfig?.clipboard, pluginConfig?.search, pluginConfig?.webLinks, onWebGLContextLost, sessionId])
```

**Important:** `pluginConfig?.webLinksConfig` (the SUB-config: modifier, confirmIDN, etc.) is **NOT in the dep array**. Sub-config changes are read at click time via the live closure / a `pluginConfigRef.current` pattern — same approach as Phase 94 for searchOptions. Re-attaching the addon on every sub-config change would be wasteful and would re-scan the buffer for no reason.

**Recommended ref pattern for sub-config:**

```typescript
// At the top of TerminalPanel:
const webLinksConfigRef = useRef(pluginConfig?.webLinksConfig)
useEffect(() => { webLinksConfigRef.current = pluginConfig?.webLinksConfig }, [pluginConfig?.webLinksConfig])

// In handler:
const cfg = webLinksConfigRef.current
if (!isModifierPressed(event, cfg?.modifier ?? 'platform')) return
```

This way the handler always reads the freshest sub-config without addon re-attachment.

### Pattern 2: OSC 8 Display-vs-Href Mismatch Detection

**What:** Register a SECOND link provider via `term.registerLinkProvider()` that walks `IBufferLine.getCell()` looking for hyperlink-id-tagged cells. For each run, collect display text + href; if they differ semantically (different host, or different scheme, or `display` looks like a URL but `href` is somewhere else), mark the link as risky and force the confirmation popover.

**When to use:** Always alongside the WebLinksAddon (Phase 95 first ships the OSC 8 detector with the addon — they're complementary, not redundant).

**Spike required:** Verify `getHyperlinkId()` is exposed in `@xterm/xterm@^6.0.0` Terminal types AND that the URL registry is reachable. If not reachable via public API:

- **Fallback A:** intercept OSC 8 sequences via `term.parser.registerOscHandler(8, callback)` — the callback receives the params + URI and can populate a separate AgentHub-side registry keyed by line+col offsets. This requires correlation with `onWriteParsed` to know where the run lands.
- **Fallback B:** DEFER OSC 8 mismatch detection to v3.3. Phase 95 ships LNK-01..02, LNK-04, LNK-05, LNK-06 fully; LNK-03 partially (IDN + typosquat detectors only; OSC 8 handled as plain link). Document as a known limitation. **Risk:** ROADMAP SC-3 explicitly tests OSC 8; deferring would fail SC-3.

**Recommended plan:** Make the OSC 8 spike a Wave 0 task. Estimate: 1 hour. If it succeeds, full LNK-03 is achievable in Phase 95. If it fails, the planner picks Fallback B and updates ROADMAP SC-3 to a follow-up phase.

[ASSUMED: `term.parser.registerOscHandler` exists in `@xterm/xterm@6.0.0`. The xterm.js source has it under `core/parser`; whether it's exported as a public API on `Terminal` is the open question.]

### Pattern 3: WebLinksConfig Persistence (Mirror of Phase 94 SearchConfig)

**What:** Add `WebLinksConfig` nested struct to `daemon.PluginSettings`. Reuse Phase 94's `SetSearchConfig` sub-key RPC pattern verbatim.

**Daemon struct change:**

```go
// internal/daemon/plugin_settings.go

// WebLinksConfig persists per-plugin configuration for the web-links toggle.
// Phase 95 (LNK-02, LNK-03, LNK-05). Defaults are platform-correct + ALL
// confirmations ON — security-first posture.
//
// JSON tags are camelCase to match daemonSettings vocabulary.
type WebLinksConfig struct {
    // Modifier: 'platform' | 'cmd' | 'ctrl' | 'none'
    // Default 'platform' resolves to Cmd on macOS, Ctrl elsewhere — at click time.
    // 'none' is a power-user opt-in that disables the modifier requirement
    // (still gated by scheme allowlist + risk detection).
    Modifier         string `json:"modifier"`
    ConfirmOSC8      bool   `json:"confirmOSC8"`
    ConfirmIDN       bool   `json:"confirmIDN"`
    ConfirmTyposquat bool   `json:"confirmTyposquat"`
}

type PluginSettings struct {
    WebGL          bool           `json:"webgl"`
    Unicode11      bool           `json:"unicode11"`
    Search         bool           `json:"search"`
    SearchConfig   SearchConfig   `json:"searchConfig"`
    WebLinks       bool           `json:"webLinks"`
    WebLinksConfig WebLinksConfig `json:"webLinksConfig"`  // NEW Phase 95
    Image          bool           `json:"image"`
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
        Serialize:      true,
        Clipboard:      true,
        Progress:       false,
    }
}
```

**`SetWebLinksConfig` sub-key RPC (mirror of `app.go:505-521 SetSearchConfig`):**

```go
// app.go
func (a *App) SetWebLinksConfig(cfg daemon.WebLinksConfig) error {
    full := a.engine.GetPluginSettings()
    full.WebLinksConfig = cfg
    if err := a.engine.SetPluginSettings(full); err != nil {
        return err
    }
    runtime.EventsEmit(a.ctx, "settings:plugins", full)
    return nil
}
```

**HTTP route (mirror of `PATCH /settings/search-config`):**

```go
// internal/daemon/api.go (Plan 94-07 precedent)
mux.HandleFunc("PATCH /settings/web-links-config", h.handleSetWebLinksConfig)

func (h *Handler) handleSetWebLinksConfig(w http.ResponseWriter, r *http.Request) {
    var cfg daemon.WebLinksConfig
    if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil { ... }
    full := h.engine.GetPluginSettings()
    full.WebLinksConfig = cfg
    if err := h.engine.SetPluginSettings(full); err != nil { ... }
    w.WriteHeader(http.StatusNoContent)
}
```

**Note:** there's a small race where the read-modify-write isn't atomic — same race exists in Plan 94-07 SearchConfig. STATE.md `Phase 94 Plan 07` notes this is acceptable for the toggle workflow (last-write-wins). Document as known limitation; do NOT fix in Phase 95.

### Pattern 4: Live Toggle Reuse (Phase 93/94 Precedent)

**What:** The "live toggle in Settings updates rendered links in already-open terminal on next refresh" requirement (LNK-05/SC-5) is automatic via the Phase 93 hot-swap useEffect.

**How it works:**

1. User toggles `webLinks` in PluginsSection → React save → `SetPluginSettings` Wails RPC → daemon writes settings.json → emits `settings:plugins` event.
2. App.tsx EventsOn handler updates `pluginConfig` state.
3. TerminalPanel receives new `pluginConfig` prop.
4. Hot-swap useEffect dep array `[..., pluginConfig?.webLinks, ...]` re-runs.
5. If toggling ON: addon constructed + loaded → on next `term.write` or scroll, the visible viewport is re-scanned and links are detected.
6. If toggling OFF: `addon.dispose()` clears all link decorations from the rendered DOM/canvas.

**Important:** "already-rendered links update on next refresh" — this means: when toggling ON, links don't appear instantly on the existing rendered output; they appear as the viewport re-renders. xterm's link providers operate per-line at render time, so disposing/re-loading the addon while the viewport is already painted does NOT immediately re-scan. The user must scroll, type a key, or wait for new output before links appear. **This is the "next refresh" semantic the ROADMAP describes; it's the natural addon behavior, not a bug.**

[VERIFIED: source-read of `WebLinksAddon.activate(terminal)` and `Terminal.registerLinkProvider` in xtermjs/xterm.js; link providers are called per-line at render time.]

**Mitigation if "instant rerender" is desired (NOT in scope for Phase 95):** call `term.refresh(0, term.rows - 1)` after loading the addon. Explicitly DO NOT do this — it's a performance hazard on large viewports and the "next refresh" wording in the ROADMAP doesn't require it.

### Pattern 5: Single-Helper Opener (`lib/openLink.ts`)

**What:** ALL URL-opening code paths funnel through one function. The function detects environment and chooses Wails RPC vs. `window.open`.

**Implementation:**

```typescript
// frontend/src/lib/openLink.ts (also conceptually mirrored in web/assets/terminal.js)

import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'

export function openLink(url: string): void {
  // Defense: re-validate scheme at the deepest layer (caller should have already checked).
  if (!/^(https?:|mailto:)/i.test(url)) return

  const isWails = typeof window !== 'undefined' &&
                  typeof (window as any).runtime?.BrowserOpenURL === 'function'

  if (isWails) {
    BrowserOpenURL(url)
  } else {
    // Web: noopener,noreferrer is non-negotiable.
    window.open(url, '_blank', 'noopener,noreferrer')
  }
}
```

**Web mirror in `web/assets/terminal.js`:**

```javascript
function openLink(url) {
  if (!/^(https?:|mailto:)/i.test(url)) return
  // Web context — Wails runtime is never present here. window.open is the only path.
  window.open(url, '_blank', 'noopener,noreferrer')
}
```

**Regression test (Wave 0):** grep ban on `location.href = `, `window.location =`, `document.location` in any frontend or web/assets/* file path that handles links. Encode this as a Go test in `internal/webserver/find_bar_test.go`-style format that source-inspects `web/assets/terminal.js`, `frontend/src/components/TerminalPanel.tsx`, and `frontend/src/lib/openLink.ts`.

### Pattern 6: Risk Detection (`lib/urlSafety.ts`)

**What:** Three pure functions, plus a static typosquat list.

**Implementation sketch:**

```typescript
// frontend/src/lib/urlSafety.ts

const ALLOWED_SCHEMES = ['https:', 'http:', 'mailto:'] as const

export function isAllowedScheme(href: string): boolean {
  try {
    const u = new URL(href)
    return ALLOWED_SCHEMES.includes(u.protocol as any)
  } catch {
    return false
  }
}

export function osc8Mismatch(displayText: string, href: string): boolean {
  // The "mismatch" we care about is when display text LOOKS LIKE a URL with a
  // host/scheme, but the actual href points to a different host or scheme.
  // Bare display text like "click here" with href "https://evil.example" IS a
  // mismatch because the user has no opportunity to see the destination at hover
  // unless the tooltip is reliable. UI-SPEC: confirmation popover surfaces ALL
  // OSC 8 cases where displayText !== href textually — strict and conservative.
  if (displayText === href) return false
  // Try to parse displayText as a URL; if successful, compare hosts.
  try {
    const dispUrl = new URL(displayText.trim())
    const hrefUrl = new URL(href)
    return dispUrl.host !== hrefUrl.host || dispUrl.protocol !== hrefUrl.protocol
  } catch {
    // displayText is not a URL → mismatch IS the case (e.g. "click here" linking to evil.com)
    return true
  }
}

export function hasIDN(href: string): boolean {
  try {
    const u = new URL(href)
    if (u.hostname.includes('xn--')) return true        // already-Punycoded form
    if (/[^\x00-\x7F]/.test(u.hostname)) return true    // Unicode-form (Cyrillic, etc.)
    return false
  } catch {
    return false
  }
}

const TYPOSQUAT_LIST = [
  'paypa1.com', 'goog1e.com', 'arnazon.com', 'amaz0n.com',
  'microsft.com', 'app1e.com', 'git-hub.com', 'tw1tter.com',
  'face-book.com', 'linked1n.com', /* ... ~30 entries ... */
] as const

export function isTypoSquat(href: string): boolean {
  try {
    const u = new URL(href)
    const host = u.hostname.toLowerCase().replace(/^www\./, '')
    return TYPOSQUAT_LIST.includes(host as any)
  } catch {
    return false
  }
}

export type RiskKind = 'osc8' | 'idn' | 'typosquat'

export function getRisk(href: string, displayText: string): RiskKind | null {
  if (osc8Mismatch(displayText, href)) return 'osc8'
  if (hasIDN(href)) return 'idn'
  if (isTypoSquat(href)) return 'typosquat'
  return null
}
```

**Test fixtures (Wave 0):**

- `javascript:alert(1)` → `isAllowedScheme()` returns `false` (LNK-01 RED test).
- `https://gооgle.com` (with Cyrillic `о`) → `hasIDN()` returns `true`. Note: copy-paste this character from the success criteria — it must be the actual Cyrillic, not Latin. Test fixture file should be UTF-8 with explicit BOM or use `о` escapes.
- `https://xn--google-jzd.com` → `hasIDN()` returns `true` (Punycoded form).
- OSC 8 with display="click here" + href="https://evil.example" → `osc8Mismatch()` returns `true`.
- OSC 8 with display="https://github.com" + href="https://github.com" → `osc8Mismatch()` returns `false`.

### Anti-Patterns to Avoid

- **Calling `location.href = url` anywhere.** Always use `openLink()`. Add grep regression test.
- **Trusting the WebLinksAddon's default `urlRegex` alone for scheme gating.** Re-validate inside the handler. (The default regex is reasonable but the handler is the canonical defense.)
- **Forgetting `noopener,noreferrer` on `window.open`.** Without these, the opened tab can read `window.opener` and navigate the parent. This is THE web-link XSS pivot. Encode in regression test.
- **Adding the WHOLE `pluginConfig` object to the hot-swap dep array.** Phase 93 Pitfall #1; dep slicing is essential. Use `pluginConfig?.webLinks` for ON/OFF; sub-config goes via ref.
- **Re-attaching the addon on sub-config (modifier, confirm*) changes.** Wasteful; sub-config is read at click time via `webLinksConfigRef.current`.
- **Using `event.altKey` as the modifier.** Some browsers reserve Alt for accessibility; Cmd/Ctrl is the established pattern. `'platform'` resolves to Cmd on mac, Ctrl elsewhere — never Alt.
- **Coding the typosquat list as a regex.** Use a `Set<string>` lookup; simpler, faster, easier to audit.
- **Performing IDN detection by trying to render the URL in a hidden DOM element and inspecting computed font.** Pure URL parsing is sufficient; don't entangle with DOM.
- **Trusting the `displayText` parameter to be UTF-8 / sanitized.** It comes from terminal output (untrusted). The popover MUST render it as text content (textContent / React's default text rendering), never as innerHTML. Otherwise `\<script\>` in display text becomes a vector.
- **Deciding "this URL is safe" based on TLS validity, redirect chain, or remote DNS.** Phase 95 is OFFLINE detection only. Network-based reputation is out of scope and a privacy hazard.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| URL detection in terminal output | Custom regex over `term.buffer.active.getLine(i).translateToString()` | `@xterm/addon-web-links` | Reinventing terminal-aware URL matching, viewport tracking, scroll-following decorations, click-target sizing — addon does all of this. |
| OSC 8 escape sequence parsing | Hand-rolled byte-level VT parser | xterm.js core (`term.parser.registerOscHandler` or `getHyperlinkId()`) | xterm's parser is the only correct OSC 8 implementation in the JS ecosystem; we'd inevitably ship bugs against it. |
| URL parsing | Hand-rolled scheme + host + IDN extraction | Native `URL` constructor | Standard, available in all Wails browsers + all modern browsers; handles edge cases (Punycode, port, fragment) correctly. |
| Punycode encoding/decoding | A library or custom implementation | Native `URL.hostname` (which exposes the Punycoded form) + a non-ASCII codepoint scan on the original href | Native is sufficient for detection; full decoding is unnecessary. |
| Tab-isolation `noopener` semantics | Custom shim or polyfill | `window.open(url, '_blank', 'noopener,noreferrer')` | Native option string; supported in all target browsers (Chromium 49+, Safari 13.1+, Firefox 52+) — all newer than our minimum. |
| Settings persistence + live propagation | Custom event bus or polling | Phase 92 `SetPluginSettings` + Phase 94 sub-key RPC | Pipeline already exists; reused without modification (one new sub-key endpoint mirroring SetSearchConfig). |
| Web-served plugin-config push | Custom WebSocket multiplexer | Phase 93 `/api/plugin-config/stream` SSE | Already capability-gated, drop-on-slow-consumer, broadcast-fan-out. New `webLinksConfig` field rides the existing JSON payload. |
| Vendoring + drift detection | Custom CI script | Phase 93 `vendor_drift_test.go` (generalized regex) | Already covers any `@xterm/addon-*` package; `addon-web-links` matches automatically; only the min-count guard increments from 6 to 7. |

**Key insight:** the "harder than it looks" pieces of clickable URLs — match detection, viewport-following decorations, hover-target sizing, OSC 8 byte parsing, Punycode hostname extraction, modal-style confirmation dismissal — all have proven solutions. Phase 95's only original work is (a) the security gate composition (scheme allowlist + modifier check + risk detection), (b) the platform-aware opener, (c) the confirmation popover UI, and (d) the static typosquat list. **All four are small.**

---

## Common Pitfalls

### Pitfall 1: WebLinksAddon Default Handler Calls `window.open(uri, '_blank')` — MISSING `noopener,noreferrer`

**What goes wrong:** Forgetting to override the addon's default handler leaves clicked links open with `window.opener` accessible — a URL with `<a target="_blank">opener=window.parent</a>` pattern works just as well from JS-controlled navigation.
**Why it happens:** WebLinksAddon's source predates universal `noopener` adoption; its default is permissive.
**How to avoid:** ALWAYS construct `WebLinksAddon` with an explicit `handler` that calls `openLink()` (which always passes `noopener,noreferrer`). NEVER `new WebLinksAddon()` (no args).
**Warning signs:** A regression test that opens a link and inspects `window.opener` from the new tab fails.
**Encode as test:** Wave 0 fixture test that asserts `webLinksAddonRef.current` was constructed with a non-default handler.

### Pitfall 2: Cyrillic Spoof Test Fixture Encoding

**What goes wrong:** Author writes `https://gооgle.com` with Latin `o`s instead of Cyrillic `о` (codepoint U+043E). Test fixture passes when it shouldn't.
**Why it happens:** Cyrillic `о` is visually identical to Latin `o`; copy-paste from the ROADMAP success criteria text MIGHT preserve the character or might not (depends on terminal/editor encoding).
**How to avoid:** ALWAYS use explicit Unicode escapes in test fixtures: `'https://gооgle.com'`. Add a comment explaining the codepoints. Add a test that verifies the fixture host string contains a non-ASCII codepoint (a metatest on the test).
**Warning signs:** `hasIDN()` returns `false` on the fixture; test passes for the wrong reason.

### Pitfall 3: `event.metaKey` on Linux/Windows Returns False — But So Does `event.ctrlKey` on macOS

**What goes wrong:** Modifier-detection logic uses `event.metaKey || event.ctrlKey` as a generic "modifier" — but Cmd-click on Linux (Super key) also sets metaKey on some keyboard layouts, leading to inconsistent UX.
**Why it happens:** Cross-platform modifier semantics are subtle.
**How to avoid:** Resolve the modifier explicitly per platform: `'platform'` mode → check `navigator.platform.toUpperCase().includes('MAC') ? event.metaKey : event.ctrlKey`. Document `'cmd'` as macOS-only convenience; `'ctrl'` as universal alternative. `'none'` skips the check entirely (power-user).
**Warning signs:** Linux user reports "I can't open links" or "links open on single click without modifier".

### Pitfall 4: Popover Position Anchored to `event.clientX/clientY` — Goes Off-Screen Near Viewport Edge

**What goes wrong:** Click on a link near the bottom-right corner; popover renders with its top-left at the click point and is clipped by the viewport.
**Why it happens:** Naive fixed-positioning anchored to click coords.
**How to avoid:** Compute popover bounds; if `bottom > window.innerHeight - margin`, anchor below-up (popover top = click.y - popoverHeight - 8). Same for right edge. Use `getBoundingClientRect` of the popover after render to flip anchor.
**Warning signs:** Popover Continue/Cancel buttons are outside the viewport on small windows.

### Pitfall 5: `term.parser.registerOscHandler` May Be Internal API

**What goes wrong:** The plan assumes `term.parser.registerOscHandler(8, callback)` is callable; turns out it's not exposed on the public Terminal type definitions.
**Why it happens:** xterm.js exposes `parser` as `IParser` interface but `registerOscHandler` may be marked `@internal` or behind `(term as any).parser`.
**How to avoid:** Wave 0 spike — try the call, inspect the type definitions, AND grep the xterm 6.0.0 source for `registerOscHandler` in the public typings. If it requires `(term as any).parser.registerOscHandler`, document that and proceed (the `as any` cast is acceptable for an internal-but-stable API). If the API doesn't exist at all in 6.0.0, fall back to Plan B (defer OSC 8 mismatch to v3.3).
**Warning signs:** TypeScript error "Property 'registerOscHandler' does not exist on type 'IParser'".

### Pitfall 6: OSC 8 Display Text Comes Through as Decorated/Multi-Line

**What goes wrong:** OSC 8 hyperlinks span multiple cells; some span MULTIPLE LINES (when terminal width wraps). Walking cells naively concatenates a wrap-broken string with embedded null bytes or trailing spaces.
**Why it happens:** Terminal buffer is grid-based; line wrap is a render-time concept, not a buffer-time concept.
**How to avoid:** Use `IBufferLine.translateToString(trimRight, startCol, endCol)` to extract clean text. Walk forward across multiple lines if the hyperlink-id continues. Test with a fixture that wraps a long display text across two lines.
**Warning signs:** Display-text comparison in `osc8Mismatch()` shows "click he" vs "click here" because of premature line break.

### Pitfall 7: Single-Click Activation on Web (Trusted-Click Misclassification)

**What goes wrong:** On the WEB-served terminal, browser security treats `event.isTrusted` differently than on desktop Wails. A synthetic click event from a script could activate a link if our `isModifierPressed` check is too permissive.
**Why it happens:** xterm dispatches its own click events from canvas-overlay markers; their `isTrusted` may be false in some renderer paths.
**How to avoid:** Do NOT check `event.isTrusted` (would break legitimate clicks in some xterm versions). Trust the modifier check + addon-mediated event source. Document that synthetic clicks via DevTools console can bypass the modifier check — this is acceptable (DevTools is a developer surface; we don't defend against the user attacking themselves). **The threat model here is `pluginConfig.webLinks=true` being abused via terminal-output content, not via DevTools.**
**Warning signs:** none expected; document threat model.

### Pitfall 8: PluginsSection Re-render Storm on `webLinksConfig` Change

**What goes wrong:** When user toggles `confirmIDN` checkbox in Settings (Phase 99 will own this UI; for now it's defaulted), `SetPluginSettings` writes the WHOLE struct, which emits `settings:plugins`, which updates `pluginConfig` state, which re-renders TerminalPanel. The hot-swap useEffect dep `pluginConfig?.webLinks` (a boolean) doesn't change → no re-attach. Sub-config change is read fresh via the ref. Should be FINE; flag it explicitly so the implementer doesn't try to "fix" the unnecessary re-render by adding sub-fields to the dep array.
**Why it happens:** The ref pattern works correctly here; just confirm via test.
**How to avoid:** **Do nothing.** Add a unit test that asserts addon construction count is 1 when sub-config changes (no re-attach). Pattern matches Phase 94 `searchOptions` precedent.

### Pitfall 9: Modifier Configuration `'none'` and Defense-in-Depth

**What goes wrong:** Power user sets `WebLinksConfig.Modifier = 'none'` and now single-clicks activate links. Combined with a malicious URL hidden by OSC 8, single-click could navigate the OS's default browser to a phishing page before the user notices.
**Why it happens:** `'none'` is a documented escape hatch.
**How to avoid:** EVEN with `'none'` modifier, the scheme allowlist + risk detection (OSC 8 mismatch / IDN / typosquat) STILL fire. The `'none'` mode disables the FIRST gate (modifier) but not the LATER gates (scheme + risk). Document this in the Settings copy.
**Warning signs:** none expected if implementation is correct.

### Pitfall 10: Hover Tooltip Persists After Mouse Leaves Link Region

**What goes wrong:** Native `title` attribute is set on hover but never cleared on mouseleave; tooltip persists.
**Why it happens:** Forgot the `leave` callback.
**How to avoid:** Pass both `hover` and `leave` to `WebLinksAddon` options:

```typescript
new WebLinksAddon(handler, {
  hover: (event, uri) => { (event.target as HTMLElement)?.setAttribute('title', uri) },
  leave: (event) => { (event.target as HTMLElement)?.removeAttribute('title') }
})
```
**Warning signs:** Stale tooltips appear over non-link cells.

### Pitfall 11: Reduced-Motion Popover Animation

**What goes wrong:** Popover uses 200ms slide-in animation; user has `prefers-reduced-motion: reduce` set; animation still plays.
**Why it happens:** CSS keyframes don't auto-respect the media query.
**How to avoid:** Use the project's existing `@media (prefers-reduced-motion: reduce) { ... animation: none; }` guard pattern (established in Phase 93 / 94).
**Warning signs:** Accessibility audit flags popover animation.

### Pitfall 12: Mailto: Address Trailing Punctuation

**What goes wrong:** Terminal output: `email me at user@example.com.` — the trailing period is captured by the regex; clicking opens `mailto:user@example.com.` which most mail clients accept anyway (period in local-part) but some reject silently.
**Why it happens:** WebLinksAddon's default regex is greedy.
**How to avoid:** Accept the addon's default. This is a known terminal-link convention (every terminal does it the same way). Document as known limitation; not a Phase 95 fix.

---

## Code Examples

### Example 1: Modifier-Pressed Helper

```typescript
// frontend/src/lib/openLink.ts (or co-located helpers)

export type ModifierMode = 'platform' | 'cmd' | 'ctrl' | 'none'

export function isModifierPressed(event: MouseEvent, mode: ModifierMode): boolean {
  if (mode === 'none') return true
  const isMac = navigator.platform.toUpperCase().includes('MAC')
  if (mode === 'platform') return isMac ? event.metaKey : event.ctrlKey
  if (mode === 'cmd') return event.metaKey
  if (mode === 'ctrl') return event.ctrlKey
  return false
}
```

### Example 2: openLink — Single Source of Truth

```typescript
// frontend/src/lib/openLink.ts
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'

export function openLink(url: string): void {
  if (!/^(https?:|mailto:)/i.test(url)) return
  const isWails = typeof window !== 'undefined' &&
                  typeof (window as { runtime?: { BrowserOpenURL?: unknown } }).runtime?.BrowserOpenURL === 'function'
  if (isWails) BrowserOpenURL(url)
  else window.open(url, '_blank', 'noopener,noreferrer')
}
```

### Example 3: LinkConfirmPopover Render Skeleton

```tsx
// frontend/src/components/LinkConfirmPopover.tsx
import { createPortal } from 'react-dom'
import type { RiskKind } from '../lib/urlSafety'

interface LinkConfirmPopoverProps {
  url: string
  risk: RiskKind
  x: number
  y: number
  onContinue: () => void
  onCancel: () => void
}

const RISK_COPY: Record<RiskKind, string> = {
  osc8: 'This link displays one address but points to another. Verify the destination before continuing.',
  idn: 'This link contains internationalized characters that can spoof familiar domains.',
  typosquat: 'This domain matches a known impersonation pattern. Verify the spelling carefully.'
}

export function LinkConfirmPopover({ url, risk, x, y, onContinue, onCancel }: LinkConfirmPopoverProps) {
  return createPortal(
    <div className="link-confirm-popover" role="dialog" aria-modal="true" aria-labelledby="link-confirm-title"
         style={{ position: 'fixed', left: x, top: y }}>
      <h3 id="link-confirm-title" className="link-confirm-popover__title">Confirm link destination</h3>
      <p className="link-confirm-popover__reason">{RISK_COPY[risk]}</p>
      <code className="link-confirm-popover__url">{url /* textContent — never innerHTML */}</code>
      <div className="link-confirm-popover__actions">
        <button onClick={onCancel} className="link-confirm-popover__btn--cancel">Cancel</button>
        <button onClick={onContinue} className="link-confirm-popover__btn--continue">Continue to site</button>
      </div>
    </div>,
    document.body
  )
}
```

### Example 4: vendor_drift_test.go min-count bump

```go
// internal/webserver/vendor_drift_test.go
// [Phase 93 generalized regex covers @xterm/addon-web-links automatically.]
// Phase 95 adds addon-web-links → bump min-count from 6 to 7:
if len(pnpmVersions) < 7 {
    t.Fatalf("expected at least 7 @xterm/* packages (xterm + addon-fit + addon-webgl + addon-unicode11 + addon-clipboard + addon-search + addon-web-links): got %v", pnpmVersions)
}
```

### Example 5: WebLinksConfig hand-edit in models.ts

```typescript
// frontend/src/wailsjs/go/models.ts (hand-edit per Phase 92 STATE.md decision)
export namespace daemon {

  export class WebLinksConfig {
    modifier: string
    confirmOSC8: boolean
    confirmIDN: boolean
    confirmTyposquat: boolean

    static createFrom(source: any = {}) { return new WebLinksConfig(source) }
    constructor(source: any = {}) {
      if ('string' === typeof source) source = JSON.parse(source)
      this.modifier = source['modifier']
      this.confirmOSC8 = source['confirmOSC8']
      this.confirmIDN = source['confirmIDN']
      this.confirmTyposquat = source['confirmTyposquat']
    }
  }

  export class PluginSettings {
    // ... existing fields ...
    webLinks: boolean
    webLinksConfig: WebLinksConfig  // NEW Phase 95
    // ... rest ...

    constructor(source: any = {}) {
      // ... existing assignments ...
      this.webLinks = source['webLinks']
      this.webLinksConfig = this.convertValues(source['webLinksConfig'], WebLinksConfig)
      // ... rest ...
    }
  }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No clickable URLs in xterm | `@xterm/addon-web-links` (vendored, custom handler) | Phase 95 introduces | Closes LNK-01..LNK-06 |
| Permissive default `window.open(uri, '_blank')` (no `noopener`) | Custom handler invoking `openLink()` (always `noopener,noreferrer` on web) | Phase 95 (handler override) | Closes the `window.opener` pivot |
| No OSC 8 mismatch detection | Second link provider parses `getHyperlinkId()` runs; `osc8Mismatch()` evaluates display vs href | Phase 95 (spike-confirmed) | Closes the OSC 8 phishing primitive |
| `PluginSettings` flat boolean for `webLinks` | `PluginSettings` with nested `WebLinksConfig` (modifier + confirm*) | Phase 95 | Pattern matches Phase 94 SearchConfig precedent; Phase 99 PUI-03 surfaces sub-fields in `<details>` UI |
| Modifier-key click in xterm.js core (legacy `Alt`-click default) | Settings-controlled `WebLinksConfig.Modifier`; default `'platform'` resolves Cmd on macOS / Ctrl elsewhere | Phase 95 | Matches user expectations from VS Code, Terminal.app |

**Deprecated / outdated:**

- **WebLinksAddon's `useLinkCodes` parameter.** Default false; legacy compatibility flag; do not enable.
- **`<a href="...">`-based link rendering in xterm canvas renderer.** xterm's link provider mechanism uses overlay decorations on top of the canvas — there are no `<a>` tags to inspect. Hover tooltips therefore use `event.target` (the overlay div) not the canvas content.

---

## Validation Architecture

`workflow.nyquist_validation` is absent from `.planning/config.json` — treat as **enabled**.

### Test Framework

| Property | Value |
|----------|-------|
| Frontend Framework | Vitest (via `pnpm exec vitest run`) |
| Go Framework | `go test ./...` |
| Component config file | `frontend/vite.config.ts` |
| Quick frontend run | `pnpm exec vitest run src/components/__tests__/{TerminalPanel.web-links,LinkConfirmPopover}.test.tsx src/lib/__tests__/{urlSafety,openLink}.test.ts` |
| Full frontend suite | `pnpm test` |
| Go unit run | `go test ./internal/daemon/... ./internal/webserver/... -count=1` |
| Go full run | `go test ./internal/...` |
| Build smoke | `wails build -tags wailsassets` |
| Manual UAT | `wails dev` (desktop); browse to `https://<machine>.<tailnet>.ts.net/sessions/<id>?cap=<token>` (web) |
| Web E2E (Playwright) | `pnpm exec playwright test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LNK-01 | Attacker-supplied `javascript:` URL is NEVER made clickable; scheme allowlist enforced | unit (lib) — RED | `pnpm exec vitest run src/lib/__tests__/urlSafety.test.ts` (asserts `isAllowedScheme('javascript:alert(1)') === false`) | ❌ Wave 0 |
| LNK-01 | `WebLinksAddon` constructed with custom handler (not default) | source-inspection (component) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.web-links.test.tsx` (regex match on TerminalPanel.tsx for `new WebLinksAddon(handler` pattern) | ❌ Wave 0 |
| LNK-02 | Single-click without modifier does NOT activate; modifier-click does | unit (component, simulated MouseEvent) — RED | `pnpm exec vitest run src/components/__tests__/TerminalPanel.web-links.test.tsx` (mock `openLink`; click without metaKey → not called; click with metaKey → called) | ❌ Wave 0 |
| LNK-02 | Hover tooltip shows resolved href (not display text) | source-inspection | `pnpm exec vitest run src/components/__tests__/TerminalPanel.web-links.test.tsx` (assert WebLinksAddon constructor receives `hover` callback that sets `title` to `uri`) | ❌ Wave 0 |
| LNK-02 | `WebLinksConfig.Modifier` default is `'platform'` | unit (Go) | `go test ./internal/daemon/... -run TestDefaultPluginSettings` (extend) | ✅ (extend) |
| LNK-03 | Cyrillic spoof URL `https://gооgle.com` triggers `hasIDN` → popover shown — RED | unit (lib) | `pnpm exec vitest run src/lib/__tests__/urlSafety.test.ts` (fixture with `о` codepoints) | ❌ Wave 0 |
| LNK-03 | OSC 8 with display "click here" + href "https://evil.example" triggers `osc8Mismatch` → popover — RED | unit (lib) | `pnpm exec vitest run src/lib/__tests__/urlSafety.test.ts` | ❌ Wave 0 |
| LNK-03 | Static typosquat list is non-empty; `paypa1.com` matches | unit (lib) | `pnpm exec vitest run src/lib/__tests__/urlSafety.test.ts` | ❌ Wave 0 |
| LNK-03 | LinkConfirmPopover renders with risk-specific copy; Cancel does not call `openLink`; Continue does | unit (component) | `pnpm exec vitest run src/components/__tests__/LinkConfirmPopover.test.tsx` | ❌ Wave 0 |
| LNK-04 | `openLink` calls `BrowserOpenURL` when Wails runtime present | unit (lib) | `pnpm exec vitest run src/lib/__tests__/openLink.test.ts` | ❌ Wave 0 |
| LNK-04 | `openLink` calls `window.open(url, '_blank', 'noopener,noreferrer')` when no Wails runtime — RED | unit (lib) | `pnpm exec vitest run src/lib/__tests__/openLink.test.ts` (assert `_blank` AND `noopener,noreferrer` present) | ❌ Wave 0 |
| LNK-04 | No `location.href = ` or `window.location =` in any link-handling code path | source-inspection (Go) | `go test ./internal/webserver/... -run TestSecurity_NoCurrentTabNavigation` | ❌ Wave 0 |
| LNK-04 | Web `terminal.js` calls `window.open(uri, '_blank', 'noopener,noreferrer')` (no `BrowserOpenURL`; no `location` mutation) | source-inspection (Go) | `go test ./internal/webserver/... -run TestTerminalJS_WebLinksOpener` | ❌ Wave 0 |
| LNK-05 | Toggling `webLinks=false` disposes addon; `=true` re-attaches | unit (component) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.web-links.test.tsx` (mock dispose; re-render with new pluginConfig) | ❌ Wave 0 |
| LNK-05 | `SetWebLinksConfig` Wails RPC writes sub-key without stomping siblings | integration (Go) | `go test ./internal/daemon/... -run TestSetWebLinksConfigPreservesSiblings` | ❌ Wave 0 |
| LNK-05 | `WebLinksConfig` defaults populate on Phase 94 fixture migration | unit (Go) | `go test ./internal/daemon/... -run TestPluginSettingsMigration_WebLinksConfig` | ❌ Wave 0 |
| LNK-05/SC-5 | Live toggle propagates via `settings:plugins` event; addon re-attaches without session restart | e2e (Playwright) | `pnpm exec playwright test e2e/web-links-live-toggle.spec.ts` | ❌ Wave 0 |
| LNK-06 | (covered by LNK-05) | — | — | — |
| WEB-01 (carry) | `web/vendor/xterm/addons/addon-web-links.js` exists & served at `/assets/xterm/addons/addon-web-links.js` | Go unit | `go test ./internal/webserver/... -run TestAssets_VendoredAddons` (extend existing list) | ✅ (extend) |
| WEB-02 (carry) | `vendor_drift_test` covers `addon-web-links` and min-count is 7 | Go unit | `go test ./internal/webserver/... -run TestXtermVendorVersionsMatchPnpmLock` | ✅ (extend min-count) |

### Wave 0 RED Tests (Required)

These tests MUST be authored as failing assertions BEFORE the implementation lands. They are the verifiable security gates the ROADMAP success criteria mandate:

- [ ] `urlSafety.test.ts: javascript: URL is NOT allowed scheme` (LNK-01)
- [ ] `urlSafety.test.ts: Cyrillic spoof URL triggers hasIDN()` (LNK-03)
- [ ] `urlSafety.test.ts: OSC 8 mismatch triggers osc8Mismatch()` (LNK-03)
- [ ] `openLink.test.ts: web environment uses window.open with _blank + noopener,noreferrer` (LNK-04)
- [ ] `TerminalPanel.web-links.test.tsx: single-click without modifier does NOT call openLink` (LNK-02)
- [ ] `TerminalPanel.web-links.test.tsx: WebLinksAddon constructed with explicit handler (not default)` (LNK-01 defense-in-depth)

### Sampling Rate

- **Per task commit:** `pnpm exec vitest run src/lib/__tests__/{urlSafety,openLink}.test.ts && pnpm exec vitest run src/components/__tests__/{TerminalPanel.web-links,LinkConfirmPopover}.test.tsx && go test ./internal/daemon/... ./internal/webserver/... -count=1`
- **Per wave merge:** `pnpm test && go test ./...`
- **Phase gate:** Full vitest + Go suite + Playwright e2e green; manual UAT (desktop wails build + web Tailscale page using dev-browser skill) before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/lib/__tests__/urlSafety.test.ts` — covers LNK-01, LNK-03 (scheme allowlist; IDN; OSC 8 mismatch; typosquat)
- [ ] `frontend/src/lib/__tests__/openLink.test.ts` — covers LNK-04 (BrowserOpenURL on desktop; window.open + noopener,noreferrer on web; scheme re-validation)
- [ ] `frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx` — covers LNK-01 (custom handler), LNK-02 (modifier-click), LNK-05 (hot-swap)
- [ ] `frontend/src/components/__tests__/LinkConfirmPopover.test.tsx` — covers LNK-03 (popover render + Cancel/Continue + risk-specific copy)
- [ ] `internal/daemon/web_links_config_test.go` (or extend existing `plugin_settings_test.go`) — covers LNK-02 default modifier, LNK-05 sub-key persistence
- [ ] `internal/webserver/web_links_test.go` — covers LNK-04 (terminal.js source-inspection: `window.open` + `_blank` + `noopener,noreferrer`; no `location.href`); WEB-01 (asset reachable)
- [ ] `frontend/e2e/web-links-live-toggle.spec.ts` — Playwright covers LNK-05/SC-5 (toggle off → no clickable links; toggle on → next-refresh links appear)
- [ ] Manual UAT scripts: `95-DESKTOP-UAT.md` (Cmd-click walkthrough on macOS, Ctrl-click on Linux/Windows VM) + `95-WEB-UAT.md` (web Cmd-F walkthrough on Chromium / Safari / iPad Safari Tailscale)
- [ ] **Wave 0 spike:** verify `term.parser.registerOscHandler` (or `IBufferCell.getHyperlinkId`) is callable from `@xterm/xterm@^6.0.0` in our installed version. Document outcome in `95-RESEARCH.md` Open Questions section. **If no API path exists, planner adopts Fallback B (defer OSC 8 to v3.3) and updates SC-3 scope.**

**Framework install:** none — vitest, Go test, Playwright, dev-browser skill all in place.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `@xterm/addon-web-links` (npm package) | LNK-01..06 | ✗ (not installed) | will install 0.12.0 | — (mandatory) |
| `web/vendor/xterm/addons/addon-web-links.js` | LNK-04, WEB-01 carry | ✗ (file does not exist) | will copy 0.12.0 from node_modules/lib | — (mandatory) |
| `@xterm/xterm@^6.0.0` (peer of addon-web-links) | LNK-01..06 | ✓ | 6.0.0 | — |
| Wails `BrowserOpenURL` | LNK-04 desktop | ✓ | (Wails v2.x runtime; verified in `frontend/src/wailsjs/runtime/runtime.js:30` and `frontend/src/wailsjs/wailsjs/runtime/runtime.js:176`) | — |
| `window.open` (browser native) | LNK-04 web | ✓ | All target browsers | — |
| pnpm | install + lockfile | ✓ | (project default) | — |
| Go 1.22+ for `//go:embed` | WEB-01 carry | ✓ | (project standard) | — |
| `@heroicons/react` | LinkConfirmPopover icons | ✓ | (in package.json from Phase 93) | — |
| `wails generate module` (or hand-edit pin per Phase 92 decision) | TS type for WebLinksConfig | ✓ (hand-edit pattern) | n/a | hand-edit `models.ts` per Phase 92 STATE.md decision |
| Playwright | e2e live-toggle test | ✓ | (already in package.json from Phase 93/94) | — |
| dev-browser skill | UAT for popover + hover | ✓ (per CLAUDE.md) | n/a | — |

**Missing dependencies with no fallback:**
- `@xterm/addon-web-links` — must `pnpm add @xterm/addon-web-links@^0.12.0` in Wave 0 task.
- `web/vendor/xterm/addons/addon-web-links.js` — must copy in Wave 1 vendoring task (after install).

**Missing dependencies with fallback:** none.

**Spike-required dependency:**
- `term.parser.registerOscHandler` OR `IBufferCell.getHyperlinkId` (xterm.js core API) — confirms LNK-03 OSC 8 path is achievable. If neither, fall back to Plan B (defer OSC 8 mismatch to v3.3).

---

## Security Domain

`security_enforcement` not explicitly disabled in `.planning/config.json` — treat as **enabled**.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | — (link click is not authenticated) |
| V3 Session Management | No | — (link click does not affect session state) |
| V4 Access Control | Yes | `/settings/web-links-config` PATCH endpoint capability-gated identically to Phase 94 `/settings/search-config` (same daemon middleware) |
| V5 Input Validation | Yes | Scheme allowlist in `isAllowedScheme()` is the FIRST gate — fail-closed; URL parsing via native `URL` constructor returns null/throws on malformed input; `displayText` rendered as textContent only (never innerHTML) |
| V6 Cryptography | No | — (no crypto operations in this phase) |
| V7 Error Handling | Yes | Click handler swallows errors silently when scheme is not allowed (no user feedback — by design; the alternative is a confusing error popup for benign output like "https:" appearing in logs) |
| V8 Data Protection | Yes | URLs in terminal output are NOT logged, telemetered, or stored — neither client-side nor daemon-side. Click events are not tracked. |
| V9 Communication | Yes | `noopener,noreferrer` on every `window.open` call is the security boundary against `window.opener` exploitation |
| V12 Files and Resources | Yes | Vendored addon-web-links.js served via Go embed.FS; `vendor_drift_test.go` CI gate enforces version parity. CSP `script-src 'self'` already in place from Phase 89. |
| V13 API and Web Service | Yes | The new `/settings/web-links-config` route inherits the existing capability-token middleware; rate-limiting + CSRF protections are inherited from the existing endpoint. |

### Known Threat Patterns for v3.2 Plugin Suite + Tailscale-served Sessions

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| `javascript:` URL in terminal output becomes clickable | Tampering / EoP | Scheme allowlist (defense-in-depth: regex AND handler); LNK-01 RED test |
| `data:text/html,<script>...</script>` data-URL | Tampering / EoP | Scheme allowlist excludes `data:`; covered by LNK-01 |
| `file:///etc/passwd` exposing local files | Information Disclosure | Scheme allowlist excludes `file://`; covered by LNK-01 |
| OSC 8 hyperlink with display="https://google.com" / href="https://evil.example" (homograph attack) | Spoofing | `osc8Mismatch()` detection → confirmation popover; LNK-03 RED test |
| Cyrillic / mixed-script IDN spoofing (`gооgle.com` with U+043E) | Spoofing | `hasIDN()` detection (both Punycode `xn--` form and Unicode form) → popover; LNK-03 RED test |
| Typosquat domain (`paypa1.com`, `arnazon.com`) | Spoofing | Static `TYPOSQUAT_LIST` heuristic → popover; LNK-03 |
| Single-click activation of attacker-controlled URL | Spoofing / Tampering | Modifier-click required by default; LNK-02 RED test |
| `window.opener` pivot from new tab on web | Tampering | `window.open(url, '_blank', 'noopener,noreferrer')` is mandatory; LNK-04 RED test |
| Current-tab navigation via `location.href = url` | Spoofing / DoS | Banned in source; regression grep test guards |
| WebLinksAddon's permissive default handler | Tampering | Custom handler required by code review + source-inspection test |
| Synthetic click events from script (DevTools) | Tampering | Acknowledged out-of-scope: user attacking own browser is not a v3.2 threat model |
| Phishing surface on Tailscale-shared sessions (tailnet viewer trusts AgentHub URL, sees clickable URL emitted by arbitrary process) | Spoofing / Tampering | All gates above apply equally to web-served sessions; dev-browser UAT confirms web behavior; STATE.md "Web-links phishing surface on Tailscale-served sessions" blocker is resolved by these gates collectively |
| Persisted `WebLinksConfig` poisoning via web client | Tampering | Web `/api/plugin-config` is GET-only; persistence happens on Wails desktop only via `SetWebLinksConfig`. Web clients cannot mutate daemon state. |
| Vendored asset drift / supply chain | Tampering | `vendor_drift_test.go` (Phase 93 generalized); SLSA L2 attestations (Phase 90) cover release artifacts |
| Catastrophic regex backtracking from URL detection | DoS | Default WebLinksAddon regex is bounded (no nested quantifiers); inherited risk model from Phase 94 RegExp DoS docs |
| Memory pressure from unbounded link decorations | DoS | xterm decorations are virtualized to viewport; addon doesn't decorate offscreen content |

### Threat-Model Bullets for Phase 95

1. **Scheme allowlist is the first and last security gate.** Both the URL regex AND the handler enforce it. Defense in depth.
2. **Modifier-click is the second gate.** The user MUST consciously activate. Single-click is never sufficient unless the user explicitly chooses `Modifier='none'` (and even then, scheme + risk gates remain).
3. **Risk-detection (OSC 8 / IDN / typosquat) is the third gate.** Pattern-matching, not network-based reputation. The user retains final say via the popover Continue button.
4. **The platform opener (BrowserOpenURL / window.open) is the fourth gate.** Tab-isolation via `noopener,noreferrer` is non-negotiable.
5. **The daemon never sees URLs.** Click handling is purely client-side. No telemetry. No logs.
6. **`pluginConfig.webLinksConfig.Modifier='none'` is documented as accepted risk.** It's available for power users who understand they're disabling the second gate; the other three gates remain. Settings copy must be explicit about this.
7. **OSC 8 mismatch detection is best-effort, contingent on the Wave 0 spike.** If `getHyperlinkId()` is unavailable in `@xterm/xterm@6.0.0`, Phase 95 falls back to plain-text-URL-only support and defers OSC 8 to v3.3. Document the choice in 95-RESEARCH update post-spike.
8. **Static typosquat list is best-effort, NOT a security boundary.** Documented as such. The popover surfaces the resolved URL even if the heuristic misses; user retains final judgment.
9. **No Tailscale-identity-aware policy.** A trusted tailnet viewer of an AgentHub session sees the same security UI as any other client. The capability token gates session ACCESS; web-link safety gates URL ACTIVATION. Two separate authorization decisions.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `term.parser.registerOscHandler(8, callback)` is callable on `@xterm/xterm@^6.0.0` Terminal instances OR `IBufferCell.getHyperlinkId()` is exposed | Pattern 2, Pitfall 5 | LNK-03 OSC 8 mismatch detection cannot be implemented in v3.2; defer to v3.3 (Plan B); SC-3 scope changes. **Mitigation:** Wave 0 spike (1 hour) verifies before implementation begins. |
| A2 | Web UMD global name is `WebLinksAddon.WebLinksAddon` (matches WebglAddon/Unicode11Addon/ClipboardAddon/SearchAddon pattern from Phase 93/94) | Pattern 1 | Web init fails with "WebLinksAddon is not a constructor". **Mitigation:** grep verification at execution time per Phase 93 Pitfall #7 — `grep -E "root\\[['\"](WebLinksAddon).*['\"]\\]" web/vendor/xterm/addons/addon-web-links.js`. |
| A3 | The 30-entry static typosquat list is "good enough" for v3.2 — exhaustive databases are out of scope | Pattern 6 | Some typosquats slip past detection. Acceptable; documented as best-effort. |
| A4 | `URL(href).hostname` exposes the Punycoded `xn--` form when constructor is given Unicode-form input | Pattern 6, Pitfall 2 | If actually returns Unicode-form, `hasIDN()` regex check on `xn--` misses some cases. **Mitigation:** Both checks (Punycode prefix AND non-ASCII codepoint) are present; either one passing yields `true`. |
| A5 | Phase 92's hand-edited `wailsjs/go/models.ts` pin pattern is the right approach for adding `WebLinksConfig` | Pattern 3 | If wrong, hand-edit drifts from Go truth. **Mitigation:** STATE.md pinned this decision in Phase 92 / Phase 94 (Plan 92-03 pattern); Phase 95 follows verbatim. |
| A6 | The `pluginSettingsProvider func() []byte` pattern from Phase 93 handles nested `WebLinksConfig` struct via `json.Marshal` recursion | Pattern 3 | If wrong, SSE pushes malformed JSON. **Mitigation:** Phase 94 SearchConfig already proved this works; identical pattern. **HIGH confidence.** |
| A7 | xterm 6.0.0's link provider mechanism dispatches `MouseEvent` (not synthetic `Event`) to handlers, so `event.metaKey` / `ctrlKey` populate correctly | Pattern 1 | Modifier check fails silently. **Mitigation:** Wave 0 unit test simulates click with `new MouseEvent('click', { metaKey: true })`. |
| A8 | The Wails IPC `BrowserOpenURL` does NOT validate scheme on the Go side — relying on JS-side `openLink` for scheme gating is sufficient | Pattern 5 | If Wails IPC accepts arbitrary URIs without sanitization, a buggy code path could pass `javascript:` to it (though in practice that just means the OS shell tries to open it as a URL, which fails). **Mitigation:** `openLink()` always re-validates scheme at the deepest layer; double gate. |
| A9 | The "next refresh" semantic in ROADMAP SC-5 means "scroll/keypress/output triggers re-scan" (the natural addon behavior), NOT "instant rerender of all rendered cells" | Pattern 4 | If SC-5 is interpreted strictly as "instant", we'd need `term.refresh(0, term.rows-1)` which has perf implications. **Mitigation:** ROADMAP wording "already-rendered links update on next refresh" supports the lenient interpretation; document explicitly in Plan and let `/gsd-verify-work` confirm via dev-browser UAT. |
| A10 | All 138 curated themes have legible link colors against their backgrounds | LNK-04 visual | Some themes might render link underline invisible. **Mitigation:** xterm.js link providers use the system's link color or theme-defined `theme.foreground` by default; not a Phase 95 problem. |

---

## Open Questions (RESOLVED)

> Q1, Q2 are deferred-by-design to the Wave 0 spike (95-01 Task 1, Step E). Q3, Q4, Q5 are
> resolved by recommendations embedded in the plans. The Wave 0 spike outcome is recorded
> below in `## Wave 0 Spike Outcome` (added at execution time).

1. **OSC 8 API availability in `@xterm/xterm@6.0.0`** (HIGH risk to LNK-03)
   - **RESOLVED:** Defer to Wave 0 spike (95-01 Task 1, Step E). Outcome recorded in
     `## Wave 0 Spike Outcome` as `**Selected:** Plan A | Plan B`. 95-04 Step A reads it.
     If both Plan A and Plan B fail, SC-3 OSC 8 mismatch detection is documented as
     deferred to v3.3 in `## Wave 0 Spike Outcome` and surfaced in `95-DESKTOP-UAT.md`
     known-issues. Risk level: HIGH for SC-3 scope; LOW for Phase 95 critical path.

2. **`WebLinksAddon.activate()` click handler — replacement or additive?**
   - **RESOLVED:** Defer to Wave 0 spike (95-01 Task 1, Step E). Source-inspect
     `frontend/node_modules/@xterm/addon-web-links/lib/addon-web-links.js` after install;
     confirm canonical replacement before Wave 1 starts. If additive, bug-fix is in scope
     before LNK-04 declares green.

3. **Hover tooltip — clear on `mouseup` (click)?**
   - **RESOLVED:** YES. The TerminalPanel handler's leave callback explicitly calls
     `event.target?.removeAttribute('title')` (95-04 Task 1 Step B6). Risk level: LOW.

4. **Web-side popover positioning across iframe boundary?**
   - **RESOLVED:** Phase 95 assumes web `terminal.html` is served standalone (no iframe).
     `event.clientX/Y` is page-local. An iframe-embedded refactor would need positioning
     revisited; documented as a known limitation in `95-WEB-UAT.md`. Risk level: LOW.

5. **Popover surfaces ONE risk or TWO when both osc8-mismatch and typosquat apply?**
   - **RESOLVED:** First-match-wins in `getRisk()` with priority `osc8 > idn > typosquat`.
     Implemented in `frontend/src/lib/urlSafety.ts` (95-02 Task 1) with a doc comment on
     `getRisk()` documenting the priority. Risk level: LOW (UI clarity, not security).

---

## Files to Create / Modify

### New Files

| File | Why |
|------|-----|
| `frontend/src/components/LinkConfirmPopover.tsx` | Risk-confirmation popover (LNK-03) |
| `frontend/src/components/__tests__/LinkConfirmPopover.test.tsx` | Component contract tests |
| `frontend/src/lib/urlSafety.ts` | Pure helpers: isAllowedScheme, osc8Mismatch, hasIDN, isTypoSquat, getRisk |
| `frontend/src/lib/__tests__/urlSafety.test.ts` | RED tests for LNK-01, LNK-03 (Cyrillic spoof, OSC 8 mismatch fixtures) |
| `frontend/src/lib/openLink.ts` | Single platform-aware opener (LNK-04) |
| `frontend/src/lib/__tests__/openLink.test.ts` | RED test asserting `_blank` + `noopener,noreferrer` |
| `web/vendor/xterm/addons/addon-web-links.js` | Vendored UMD bundle (WEB-01 carry) |
| `internal/webserver/web_links_test.go` | Source-inspection: terminal.js opener gates, addon vendored & served |
| `frontend/e2e/web-links-live-toggle.spec.ts` | Playwright e2e for LNK-05 live-apply (toggle off → no clickable links; on → next-refresh appearance) |
| `.planning/phases/95-web-links-addon-security-hardening/95-DESKTOP-UAT.md` | Manual UAT runbook (macOS Cmd-click, Linux Ctrl-click) |
| `.planning/phases/95-web-links-addon-security-hardening/95-WEB-UAT.md` | Manual UAT runbook (Tailscale web; dev-browser skill) |

### Modified Files

| File | Change | Requirement |
|------|--------|-------------|
| `frontend/package.json` | Add `@xterm/addon-web-links@^0.12.0` to dependencies | All LNK |
| `frontend/pnpm-lock.yaml` | (regenerated) | All LNK |
| `frontend/src/components/TerminalPanel.tsx` | Add `webLinksAddonRef`; extend hot-swap useEffect dep array with `pluginConfig?.webLinks`; add `webLinksConfigRef` for sub-config; render `<LinkConfirmPopover>` conditionally; call `openLink` in handler success path | LNK-01, LNK-02, LNK-04, LNK-05 |
| `frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx` (extend or new) | Tests for hot-swap, modifier-click, custom handler usage | LNK-01, LNK-02, LNK-05 |
| `frontend/src/components/PluginsSection.tsx` | (NO CHANGE — webLinks toggle exists from Phase 92; advanced disclosure deferred to Phase 99 PUI-03) | — |
| `frontend/src/App.tsx` | (NO CHANGE — pluginConfig prop drill already exists; webLinksConfig piggybacks) | — |
| `frontend/src/style.css` | Add `/* ─── Phase 95 — Link confirm popover (LNK-03) ─── */` section with `.link-confirm-popover*` BEM classes + reduced-motion guard | LNK-03 |
| `frontend/src/wailsjs/go/models.ts` | Hand-edit: add `WebLinksConfig` class to `daemon` namespace; add `webLinksConfig` field to `PluginSettings` class | LNK-02, LNK-05 (per Phase 92 STATE.md pin decision) |
| `frontend/src/wailsjs/go/main/App.{d.ts,js}` | Hand-edit: add `SetWebLinksConfig(arg1: daemon.WebLinksConfig): Promise<void>` (mirror of `SetSearchConfig` Plan 94-07 pattern) | LNK-05 |
| `frontend/src/__tests__/App.plugin-event.test.tsx` | Add webLinksConfig to expected PluginSettings shape; assert prop drill round-trip preserves nested struct | LNK-05 |
| `internal/daemon/plugin_settings.go` | Add `WebLinksConfig` struct + `WebLinksConfig` field to `PluginSettings`; update `defaultPluginSettings()` | LNK-02 default, LNK-05 |
| `internal/daemon/plugin_settings_test.go` | Extend `TestDefaultPluginSettings` to assert WebLinksConfig defaults; add `TestPluginSettingsMigration_WebLinksConfig` | LNK-02 default, LNK-05 |
| `internal/daemon/engine.go` | Add `SetWebLinksConfig(cfg daemon.WebLinksConfig) error` sub-key writer (mirror of SetSearchConfig at engine.go:466-484) — preserves siblings | LNK-05 |
| `internal/daemon/api.go` | Add `PATCH /settings/web-links-config` route (mirror of /settings/search-config) | LNK-05 |
| `internal/daemon/web_links_config_test.go` (or extend search_config_test.go) | Test sub-key writer preserves siblings; PATCH route happy + unhappy paths | LNK-05 |
| `app.go` | Add `(*App).SetWebLinksConfig(cfg daemon.WebLinksConfig) error` (mirror of `SetSearchConfig` at app.go:505-521) | LNK-05 |
| `internal/webserver/vendor_drift_test.go` | Bump min-count guard from 6 to 7 | WEB-02 (carry) |
| `internal/webserver/no_cdn_regression_test.go` | (NO CHANGE — `vendor/xterm/addons/` skip naturally covers `addon-web-links.js`) | — |
| `web/embed.go` | Extend `//go:embed` directive to include `vendor/xterm/addons/addon-web-links.js` | WEB-01 carry |
| `web/vendor/xterm/VERSION` | Append `@xterm/addon-web-links@0.12.0` line | WEB-01 carry |
| `web/terminal.html` | Add `<div id="link-confirm-popover" hidden role="dialog" aria-modal="true">…</div>`; add `<script src="/assets/xterm/addons/addon-web-links.js"></script>` BEFORE `terminal.js` | LNK-03, WEB-01 |
| `web/assets/terminal.js` | Add `applyWebLinksAddon()` arm to `applyPluginConfig()`; mirror desktop `openLink()`; mirror `urlSafety` helpers (inline copy — no Go import); popover DOM wiring | LNK-01..04 web parity |
| `web/assets/terminal.css` | Add `/* Phase 95 — Link confirm popover */` section parallel to desktop | LNK-03 web parity |

### Daemon wiring (NO change to existing code)

Phase 93 `pluginSettingsProvider func() []byte` recurses through `json.Marshal` for the new nested `WebLinksConfig` field. Phase 94 SearchConfig already proved this. No webserver code changes for the SSE broadcast path.

---

## Sources

### Primary (HIGH confidence)

- `npm view @xterm/addon-web-links version` returned `0.12.0` (2026-05-06)
- `npm view @xterm/addon-web-links main peerDependencies` returned `lib/addon-web-links.js` (2026-05-06)
- `github.com/xtermjs/xterm.js` master, `addons/addon-web-links/src/WebLinksAddon.ts` — verified constructor signature, default URL regex, default handler behavior
- `github.com/xtermjs/xterm.js` master, `typings/xterm.d.ts` — verified `Terminal.registerLinkProvider`, `ILinkProvider`, `ILink` interfaces; `IBufferCell.getHyperlinkId()` typings
- `internal/daemon/plugin_settings.go` — verified PluginSettings struct + Phase 94 `SearchConfig` precedent at lines 9-22; defaults at 50-65
- `internal/daemon/engine.go:466-484` — verified Phase 94 sub-key writer pattern (cited by file path in 94-RESEARCH; preserves siblings while updating one sub-struct)
- `internal/daemon/api.go` — verified `PATCH /settings/search-config` route registration pattern (Plan 94-07)
- `app.go:461-521` — verified `(*App).GetPluginSettings`, `(*App).SetPluginSettings`, `(*App).SetSearchConfig` — the latter is the canonical sub-key RPC mirror Phase 95 follows
- `frontend/src/wailsjs/runtime/runtime.js:30-32` — verified `BrowserOpenURL` Wails runtime export shape
- `frontend/src/wailsjs/wailsjs/runtime/runtime.js:176-178` — verified production Wails runtime `BrowserOpenURL` invocation pattern
- `frontend/src/App.tsx:683-685` — verified existing `BrowserOpenURL(url)` usage in `handleOpenRemoteSession`
- `frontend/src/components/SettingsTab.tsx:540` — verified existing `BrowserOpenURL(serverURL)` open-in-default-browser pattern
- `frontend/src/components/UpdateBanner.tsx:35` — verified `BrowserOpenURL(update.releaseURL)` pattern
- `frontend/src/components/SessionSharePanel.tsx:112,149` — verified read/write capability share-URL `BrowserOpenURL` pattern
- `frontend/src/components/TerminalPanel.tsx:259-339` — verified Phase 93/94 hot-swap useEffect with explicit dep slicing (Pitfall #1); SearchAddon load/dispose pattern at lines 311-338
- `frontend/src/components/TerminalPanel.tsx:88-131` — verified Phase 94 `searchOptions` ref + lazy seed pattern (Plan 94-07 seededRef invariant); applies verbatim to `webLinksConfigRef` strategy
- `internal/webserver/vendor_drift_test.go:18` — verified Phase 93 generalized regex `^  '(@xterm/(?:xterm|addon-[\w-]+))@([0-9.]+)':`
- `internal/webserver/plugin_config.go` + `plugin_config_stream.go` — verified Phase 93 SSE broadcast and GET-only contract (no web-side WRITE path; Phase 95 doesn't add one either)
- `web/embed.go` — verified //go:embed directive structure with all 4 current addon entries
- `web/terminal.html:43-49` — verified script tag ordering: xterm.js → addon-fit → addon-webgl → addon-unicode11 → addon-clipboard → addon-search → terminal.js (Phase 95 inserts addon-web-links between addon-search and terminal.js)
- `web/assets/terminal.js:240-353` — verified Phase 93/94 applyPluginConfig diff-applying function structure (Phase 95 adds web-links arm following the same pattern)
- `web/vendor/xterm/VERSION` — verified 6-line manifest current state (Phase 95 appends 7th line for addon-web-links)
- `frontend/package.json` — verified pnpm dependency lock for `@xterm/*` family at versions 0.11.0/0.19.0/0.9.0/0.2.0/0.16.0/6.0.0
- `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-RESEARCH.md` (Phase 93 patterns inherited)
- `.planning/phases/94-search-addon-find-bar-desktop-web/94-RESEARCH.md` (Phase 94 SearchConfig + sub-key RPC + ref-driven sub-config patterns inherited)
- `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-VALIDATION.md` (validation pattern template; Phase 95 mirrors per-task verification map)
- `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-VERIFICATION.md` (gap-closure pattern; Phase 95's UAT script + RED test discipline mirrors)
- `.planning/STATE.md ## Decisions` — Phase 95 entry pinning v3.1-WS-Origin-allowlist rigor approach
- `.planning/REQUIREMENTS.md` LNK-01..LNK-06 + `## Out of Scope` (custom URL protocol allowlists explicitly excluded)

### Secondary (MEDIUM confidence)

- `https://www.npmjs.com/package/@xterm/addon-web-links` — npm package landing page (matches registry metadata)
- xterm.js GitHub Issues + Discussions referencing OSC 8 implementation details (issue #4135 specifically referenced; not freshly verified in this research session — risk mitigation: Wave 0 spike)
- `frontend/node_modules/@xterm/addon-clipboard/typings/addon-clipboard.d.ts` — analogous addon shape for cross-reference (Phase 93 source)
- STATE.md `## Decisions` — Phase 92 hand-edit models.ts pin decision; Phase 94 Plan 07 `seededRef` + sub-key RPC patterns

### Tertiary (LOW confidence — ASSUMED)

- A1: `term.parser.registerOscHandler` OR `IBufferCell.getHyperlinkId` exposed as public API on `@xterm/xterm@6.0.0` (Wave 0 spike resolves)
- A2: Web UMD global name `WebLinksAddon.WebLinksAddon` (extrapolated from Phase 93/94 verified pattern; verify at execution time via grep)
- A3: 30-entry static typosquat list is sufficient for v3.2 (heuristic; documented as best-effort)
- A4: Native `URL.hostname` exposes Punycoded `xn--` form when given Unicode-form input (cross-checked with WHATWG URL spec but not freshly verified in Chrome/Safari this session)
- A8: Wails `BrowserOpenURL` does not validate scheme on Go side (defense-in-depth pushes the canonical check to JS-side `openLink`; double-gated)

---

## Wave 0 Spike Outcome

**Spike date:** 2026-05-06
**Conducted in:** Plan 95-01 Task 1 Step E
**Inspected commit:** `@xterm/addon-web-links@0.12.0` + `@xterm/xterm@6.0.0` (current pnpm-lock.yaml resolution)

### Finding 1: WebLinksAddon handler IS canonical replacement (not additive)

**Path inspected:** `frontend/node_modules/@xterm/addon-web-links/lib/addon-web-links.js` (UMD bundle; minified single-line source).

**Evidence (verbatim from the bundle):**
```js
// Default click handler — calls window.open() then sets location.href
function i(e,t){const n=window.open();if(n){try{n.opener=null}catch{}n.location.href=t} ...}

// Constructor stores ONLY the user-supplied handler (or default `i`)
e.WebLinksAddon=class{
  constructor(e=i,t={}){this._handler=e;this._options=t}
  activate(e){
    this._terminal=e;
    const n=this._options,o=n.urlRegex||r;
    this._linkProvider=this._terminal.registerLinkProvider(
      new t.WebLinkProvider(this._terminal,o,this._handler,n)
    )
  }
  dispose(){this._linkProvider?.dispose()}
}
```

The `_handler` is the SOLE click handler — it is passed straight through to
`WebLinkProvider.computeLink`, which assigns it as the `activate` callback on
each link object. There is no second, additive default handler that fires
alongside. **A user-supplied handler fully replaces the default.**

**Outcome:** PASS — the constructor's handler argument replaces the default
`window.open(uri, '_blank')` call (technically `window.open()` then
`location.href = t`, which is *worse* than `_blank` and confirms why a
custom handler is mandatory for LNK-04 / LNK-05).

### Finding 2: registerLinkProvider IS publicly typed; getHyperlinkId IS NOT

**Path inspected:** `frontend/node_modules/@xterm/xterm/typings/xterm.d.ts`.

**Evidence:**
- `Terminal.registerLinkProvider(linkProvider: ILinkProvider): IDisposable;` — present at line 1102 (public). PASS.
- `interface ILinkProvider { provideLinks(...): void; }` — present at line 1393 (public). PASS.
- `IParser.registerOscHandler(ident: number, callback: ...): IDisposable;` — present at line 1864 (public). PASS.
- `IBufferCell.getHyperlinkId()` — **NOT FOUND** in the public typings. `grep -rn getHyperlinkId frontend/node_modules/@xterm/xterm/` returns ZERO matches in either typings or runtime source. The `IBufferCell` interface (lines 1635-1750) lists `getWidth`, `getChars`, `getCode`, `getFgColorMode`, `getBgColorMode`, plus a battery of `isFg*`/`isBg*` predicates — but no hyperlink-id accessor.

**Outcome:** PARTIAL FAIL — `registerLinkProvider` and `registerOscHandler` are
publicly typed, but `IBufferCell.getHyperlinkId()` (the symbol Plan A would
need to walk OSC 8 link ranges and surface display-vs-href divergence) is
absent from the public surface of `@xterm/xterm@6.0.0`. Implementing Plan A
in v3.2 would require either (a) reaching into internal buffer state
(maintenance hazard; breaks on minor xterm bumps), or (b) maintaining a
parallel OSC 8 hyperlink registry by intercepting `registerOscHandler(8)`
and tracking ranges manually (custom plumbing the addon already declines to
ship — for the same reason).

### Chosen Path

**Plan B — Defer OSC 8 Mismatch to v3.3** (selected because Finding 2 is a partial fail).

Plan 95-04 ships LNK-01..02, LNK-04, LNK-05, LNK-06 fully. LNK-03 ships **IDN +
typosquat** detectors only. The OSC 8 mismatch detector (`osc8Mismatch` helper
in `urlSafety.ts`) is still authored as a pure function (display + href ⇒
boolean) so the unit test scaffold from Plan 95-01 Task 2 stays meaningful and
flips GREEN on its own — but it is NOT wired into the live `getRisk` path
because `urlSafety.ts` cannot reach the OSC 8 *display* string from inside the
addon-web-links click handler (the handler only receives the *href*).

The Cyrillic spoof fixture and typosquat fixtures still trigger the popover via
`hasIDN` and `isTypoSquat`. The OSC 8 mismatch fixture documents a known
limitation in the popover help text (popover surface still exists for
risk='osc8' so wiring lands cleanly when v3.3 unblocks).

ROADMAP SC-3 scope reduces accordingly: a `LNK-OSC8-FUT-01` follow-up is
created in REQUIREMENTS.md `## Future Requirements` (Plan 95-04 will add this
in its task list when it picks up the SC-3 narrowing).

**Selected:** Plan B

---

## Metadata

**Confidence breakdown:**
- Standard Stack: HIGH — `@xterm/addon-web-links@0.12.0` confirmed in npm registry; `BrowserOpenURL` confirmed in repo
- Architecture: HIGH — patterns verified against Phase 93/94 RESEARCH + actual codebase paths (TerminalPanel.tsx hot-swap, applyPluginConfig in terminal.js, search_config sub-key RPC in app.go)
- OSC 8 path: MEDIUM — A1 is the only material risk; Wave 0 spike de-risks before Wave 1 starts
- Pitfalls: HIGH — 12 pitfalls catalogued, most derived from reading actual code (handler signature, regex defaults, sub-config ref pattern, modifier semantics)
- Validation Architecture: HIGH — RED tests align directly to ROADMAP success criteria; existing test infrastructure supports all six RED gates
- Security domain: HIGH — STRIDE matrix complete; ASVS categories mapped; explicit threat-model bullets

**Research date:** 2026-05-06
**Valid until:** 2026-06-06 (xterm.js addon APIs are stable; only A1 is time-sensitive — re-verify after any `@xterm/xterm` major bump)

---

## RESEARCH COMPLETE
