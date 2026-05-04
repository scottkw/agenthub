# Phase 94: Search Addon + Find Bar (Desktop + Web) — Research

**Researched:** 2026-05-04
**Domain:** xterm.js `@xterm/addon-search` integration, focus-conditioned keybindings, daemon-persisted SearchConfig, find-bar UI in React (desktop) + plain DOM (web)
**Confidence:** HIGH

---

## Summary

Phase 94 ships a polished find bar in two surfaces (Wails React desktop + plain HTML/CSS/JS web page) by integrating `@xterm/addon-search@0.16.0`. The addon is **not yet installed** in `frontend/node_modules` and **not yet vendored** under `web/vendor/xterm/addons/` — Phase 94 must do both, using the same vendoring discipline established in Phase 93 (which did this for webgl/unicode11/clipboard). The Phase 92 daemon `PluginSettings` struct must be extended with a nested `SearchConfig { regex, caseSensitive, wholeWord }` sub-struct so per-flag toggle defaults persist via the existing `SetPluginSettings` Wails RPC + `settings:plugins` event + Phase 93 `/api/plugin-config/stream` SSE pipeline. The find bar itself is fully specified in `94-UI-SPEC.md` (locked) — research focuses on the addon API, performance characteristics, focus conditioning, persistence wiring, and validation matrix.

**Key architecture lift:** the addon is **synchronous** with a hardcoded `DEFAULT_HIGHLIGHT_LIMIT = 1000`. A naïve search of a 10,000-line scrollback can iterate all matches on the main thread — but the highlight cap and the spec-required 100ms debounce together stay under the 1s frame budget for SC-3. No Worker / no AbortController is needed; cancellation is achieved by clearing the debounce + calling `clearDecorations()` when the find bar closes.

**Primary recommendation:** Install `@xterm/addon-search@^0.16.0` via pnpm; vendor `lib/addon-search.js` to `web/vendor/xterm/addons/` and add `<script>` tag in `web/terminal.html`; extend daemon `PluginSettings` with `SearchConfig` (3 booleans, all default `false`); reuse Phase 93's `pluginSettingsProvider func() []byte` and SSE broadcast unchanged; create `FindBar.tsx` (controlled component) + `useFindBar.ts` hook that owns `SearchAddon` lifecycle inside `TerminalPanel.tsx`; replicate the find bar in plain DOM inside `web/terminal.html` + `web/assets/terminal.{js,css}`; add 5 new test files (search addon API contract, focus conditioning, persistence round-trip, 10k-line perf, web parity).

---

## Project Constraints (from CLAUDE.md)

- **JS/TS:** `camelCase` vars, `PascalCase` components, ESLint + Prettier, TypeScript types — applies to `FindBar.tsx`, `useFindBar.ts`, and any new helpers.
- **Node:** use `pnpm` (confirmed by `frontend/pnpm-lock.yaml` + Phase 93). Add `@xterm/addon-search` via `pnpm add @xterm/addon-search@^0.16.0`.
- **Go:** `go fmt`, context-aware functions. Applies to any daemon struct extensions.
- **No global npm installs.** Vendoring uses `node_modules/.../lib/addon-search.js` copied locally under `web/vendor/xterm/addons/`.
- **NEVER kill node.exe** — no shell scripts that touch node processes.
- **LSP first** for code navigation when planning.
- **UAT via dev-browser skill** — Phase 94 has both desktop (Wails build) UAT and web (Tailscale-served page) UAT; the verifier will use dev-browser for the web side.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SRC-01 | Cmd-F opens find bar (focus-conditioned: only when xterm DOM is `document.activeElement`); Esc dismisses; identical on desktop and web | Window-level keydown handler with `document.activeElement === xterm element` check; React surface adds listener in TerminalPanel mount useEffect; web surface adds listener in `terminal.js` after `term.open()`. Both surfaces honor `pluginConfig.search === true` gate. |
| SRC-02 | Next/prev (Enter/Cmd-G, Shift-Enter/Cmd-Shift-G), match count "3 of 12", regex/case/whole-word toggles with persisted defaults | `searchAddon.findNext(query, { regex, caseSensitive, wholeWord, decorations: {...} })` + `searchAddon.findPrevious(...)`; subscribe to `searchAddon.onDidChangeResults(({resultIndex, resultCount}) => ...)` for live count; persist via new `SearchConfig` sub-struct in `daemon.PluginSettings` reusing Phase 92's SetPluginSettings RPC. |
| SRC-03 | 10,000-line scrollback search no UI lockup (no >1s frame budget breach); long regex cancellable | Addon is synchronous but has DEFAULT_HIGHLIGHT_LIMIT=1000 + internal 200ms `disposableTimeout` for incremental updates; combined with 100ms input debounce (UI-SPEC line 315) keeps frame budget honored. Cancellation = clear debounce + `clearDecorations()` on close. No worker/AbortController needed. |
| SRC-04 | TokyoNight palette, 200ms slide-in/out, theme-aware highlight via `theme.selectionBackground` (138 themes) | UI-SPEC §"Color" + §"Animation" locks all visuals; SearchAddon receives `decorations: { matchBackground, matchBorder, activeMatchBackground, activeMatchBorder }` — but we leave these undefined so xterm falls back to its built-in selection-color path. xterm.js core uses `theme.selectionBackground` for the underlying selection highlight. Phase 65/71 established the 138-theme invariant that `selectionBackground` is always set. |
| SRC-05 | Web parity: same shortcuts and visual treatment on web-served sessions | UI-SPEC §"Web page — Identical Behavior" specifies plain DOM find bar in `web/terminal.html` + same SearchAddon initialization in `web/assets/terminal.js`; `pluginConfig.search` flag from `/api/plugin-config` gates web init same as desktop. |
</phase_requirements>

---

<user_constraints>
## User Constraints

> No `94-CONTEXT.md` exists for Phase 94 — there was no `/gsd-discuss-phase` step. The locked design contract is `94-UI-SPEC.md` which serves the same role for visual/interaction decisions. STATE.md captures the one cross-cutting decision below.

### Locked Decisions

- **Phase 94 owns find-bar UI for BOTH desktop and web.** [STATE.md `## Decisions`] User explicitly chose ambitious scope; original SUMMARY proposed deferring web UI but was overruled. Confidence: HIGH.
- **All visual/interaction details are locked in `94-UI-SPEC.md`.** Reviewed and approved 2026-05-04 (frontmatter `status: approved`, `reviewed_at: 2026-05-04`). No re-litigation.
- **TokyoNight palette, BannerStack vocabulary, BEM class naming.** Inherited from Phase 92/93. No new tokens.
- **Default search options all OFF (regex/case/word).** UI-SPEC §"Toggle Persistence" — `SearchConfig` defaults from daemon.
- **`@xterm/addon-search` vendored same-origin under `web/vendor/xterm/addons/addon-search.js`** — Phase 93 vendoring discipline applies. No CDN.
- **Find bar is focus-conditioned on Cmd-F.** Browser native find still works for non-terminal page text. UI-SPEC §"Opening the Find Bar".
- **No persistence of find-bar open/closed state across sessions.** UI-SPEC §"Anti-goals". No history. No recent searches.
- **Web page persists `SearchConfig` toggle changes in-memory only.** Web pages are read-only consumers of daemon config (UI-SPEC §"Toggle Persistence"). Desktop owns the canonical persistence path.
- **Phase 94 does NOT add the `<details>` advanced disclosure under the search toggle.** That is Phase 99 / PUI-03 (UI-SPEC §"Phase Scope").

### Claude's Discretion

- Choice of how to wire `SearchAddon` lifecycle inside `TerminalPanel.tsx` (separate hook vs. inline useEffect).
- Choice of debounce mechanism (lodash `debounce` vs. setTimeout cleanup vs. existing patterns).
- Choice of whether the focus trap inside `FindBar.tsx` uses an existing library (e.g. `focus-trap-react`) or hand-rolled tab handling. **Recommendation: hand-rolled** — only 7 elements, adding a dependency is heavier than the implementation.
- Choice of how to surface "match N of M" when a search is in progress (debounce window). **Recommendation:** keep the previous count visible until the new search completes; never flicker to "0 of 0" mid-debounce.
- Choice of test framework granularity: `vitest` for component, `@playwright/test` (if present) or manual UAT for end-to-end.

### Deferred Ideas (OUT OF SCOPE)

- Find-in-all-sessions (Cmd-Shift-F) — `SRC-FUT-01`, deferred indefinitely (REQUIREMENTS.md `## Future Requirements`).
- `<details>` disclosure for default regex/case/word search options under the Settings toggle — Phase 99 / PUI-03.
- Highlight color theming beyond `theme.selectionBackground` — using xterm built-in only.
- Auto-close after N seconds; history; persistence of open/closed state.

</user_constraints>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| SearchAddon lifecycle (load/dispose) | Browser / Wails WebView (desktop); Browser / web page (web) | — | xterm addon API is browser-only; lives where the Terminal instance lives |
| Find-bar React component | Browser / Wails WebView | — | Pure UI; controlled component; no network or daemon calls |
| Find-bar plain DOM (web) | Browser / web page | — | Replicated UI in `web/terminal.html` + `web/assets/terminal.{js,css}` |
| Cmd-F keybinding (focus-conditioned) | Browser (both surfaces) | — | `window.addEventListener('keydown')` + `document.activeElement` check |
| `SearchConfig` persistence | API / Backend (daemon) | Wails RPC | Daemon owns `settings.json`; existing Phase 92 RPC + Phase 93 SSE broadcast unchanged |
| `pluginConfig.search` propagation desktop | App.tsx state → `pluginConfig` prop drill into TerminalPanel | — | Phase 92 pipeline; reused unchanged |
| `pluginConfig.searchConfig` propagation web | `/api/plugin-config` GET + `/api/plugin-config/stream` SSE | — | Phase 93 endpoints; the web client already consumes plugin-config; adding `searchConfig` is just a new field in the JSON payload |
| Vendored addon serving | CDN / Static (embedded) | — | `web/vendor/xterm/addons/addon-search.js` served via Go embed.FS at `/assets/xterm/addons/addon-search.js` |
| `vendor_drift_test.go` CI gate | CI / Go test | — | Already generalized in Phase 93 — version parity for `@xterm/addon-search` is automatic when added to `pnpm-lock.yaml` + `web/vendor/xterm/VERSION` |

**Cross-tier note for SRC-03:** the 10,000-line scrollback lives in xterm's internal buffer **on the browser tier**. Search performance is purely a browser-tier concern; daemon and Go server are not involved in the search path at all. The only daemon round-trip is the initial `pluginConfig.searchConfig` load (one HTTP fetch on web; one Wails RPC on desktop, both at startup).

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/addon-search` | `^0.16.0` | Scrollback search, decoration-based match highlighting, `findNext`/`findPrevious` API, `onDidChangeResults` events for match count | First-party `@xterm` scoped addon; only viable option for xterm.js search; the same family as already-used `@xterm/addon-fit`, `@xterm/addon-webgl`, etc. |

**Verified:** `npm view @xterm/addon-search version` returned `0.16.0` on 2026-05-04. [VERIFIED: npm registry] Peer dependencies: none declared in package.json (informal peer is `@xterm/xterm@^6.0.0`, which the project already pins). `main: lib/addon-search.js` (CJS UMD bundle — correct file for web vendoring; `.mjs` is ES module-only). [VERIFIED: registry.npmjs.org/@xterm/addon-search/0.16.0]

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@heroicons/react/24/outline` | (existing) | Toggle icons (Aa, .*, [ ]), nav arrows, close × | Desktop FindBar.tsx — UI-SPEC §"Design System" mandates this for icon consistency with Phase 93 |
| `@heroicons/react/20/solid` (XMarkIcon) | (existing) | Close button icon (16px) | Matches Phase 93 `WebGLRecoveryBanner` pattern (UI-SPEC line 262) |
| (none for focus trap) | — | Focus trap inside find bar | Hand-rolled — 7-element trap; library overhead not justified |
| (none for debounce) | — | 100ms search debounce | Hand-rolled `setTimeout` + clear-on-cleanup; pattern already used elsewhere in the codebase |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `@xterm/addon-search@0.16.0` | Hand-rolled regex over xterm buffer | Reinventing decoration management, scroll-to-match, viewport tracking; addon does this correctly. **Don't.** |
| Synchronous search | WebWorker offload | Addon is sync but has internal 200ms `disposableTimeout` for incremental highlight rebuild. Combined with 100ms input debounce + 1000-result hard cap, frame budget is honored. Worker would require refactoring xterm's internal SearchEngine — out of scope. |
| `focus-trap-react` library | Hand-rolled Tab cycling | Library is ~5KB; FindBar has only 7 focusable elements; hand-rolled is shorter and easier to test. Hand-roll. |
| Lodash `debounce` | `setTimeout` + clear-on-cleanup | Lodash adds a dep we don't need for one debounce. Inline setTimeout is fine. |

**Installation:**

```bash
cd frontend && pnpm add @xterm/addon-search@^0.16.0
```

After install, copy `frontend/node_modules/@xterm/addon-search/lib/addon-search.js` to `web/vendor/xterm/addons/addon-search.js` (per Phase 93 vendoring pattern).

**Version verification:**
```bash
npm view @xterm/addon-search version  # confirmed 0.16.0 on 2026-05-04
```

[VERIFIED: npm registry, 2026-05-04]

---

## SearchAddon API Contract

Based on direct read of upstream source (xtermjs/xterm.js master, addons/addon-search/) and TypeScript definitions [VERIFIED: github.com/xtermjs/xterm.js source, 2026-05-04]:

### Constructor

```typescript
new SearchAddon(options?: Partial<ISearchAddonOptions>)

interface ISearchAddonOptions {
  highlightLimit: number;  // default 1000
}
```

**Phase 94 usage:** `new SearchAddon()` — accept the 1000-result default. Larger limits would slow highlight rebuild without UX benefit (a search returning >1000 matches signals a query that's too broad).

### Lifecycle

```typescript
public activate(terminal: Terminal): void;  // called by term.loadAddon()
public dispose(): void;                       // called on unmount or toggle-off
public clearDecorations(retainCachedSearchTerm?: boolean): void;
public clearActiveDecoration(): void;
```

### Search Methods

```typescript
public findNext(term: string, searchOptions?: ISearchOptions): boolean;
public findPrevious(term: string, searchOptions?: ISearchOptions): boolean;

interface ISearchOptions {
  regex?: boolean;
  wholeWord?: boolean;
  caseSensitive?: boolean;
  incremental?: boolean;
  decorations?: ISearchDecorationOptions;
}
```

**Returns:** `true` if a match was found and selected; `false` otherwise.

**Phase 94 usage:** Pass `{ regex, caseSensitive, wholeWord }` from current toggle state. **Do NOT pass `decorations`** — leaving it undefined makes the addon use xterm's built-in selection-color path (which uses `theme.selectionBackground`), satisfying SRC-04's "theme-aware highlight via `theme.selectionBackground`" requirement automatically. Setting custom decoration colors would require reading the theme and reconciling per-theme — overkill when the default is correct.

### Events

```typescript
readonly onAfterSearch: IEvent<void>;
readonly onBeforeSearch: IEvent<void>;
readonly onDidChangeResults: IEvent<ISearchResultChangeEvent>;

interface ISearchResultChangeEvent {
  resultIndex: number;  // 0-based; -1 if highlightLimit exceeded
  resultCount: number;
}
```

**Phase 94 usage:** `searchAddon.onDidChangeResults(({resultIndex, resultCount}) => setMatchCount({index: resultIndex, count: resultCount}))`. Display "{resultIndex+1} of {resultCount}". Special cases:
- `resultIndex === -1` → query exceeded highlight limit; UI-SPEC §"Searching" allows degrading to "Match N" with no total. **Recommendation:** display `{count}+` (e.g. "1000+" if count >= 1000) and show resultCount when available even if index is -1.
- `resultCount === 0` AND non-empty query → "0 of 0" with `--no-results` modifier (color `#f7768e`).

### Internal Behavior (relevant for SRC-03)

[VERIFIED: source read of `addons/addon-search/src/SearchAddon.ts`]:

- Search is **synchronous**: `_highlightAllMatches` iterates results in a `while` loop until `_highlightLimit` (default 1000) is hit OR no more matches found.
- A `disposableTimeout` of 200ms re-runs highlight on `onWriteParsed` and `onResize`. This means a stream of incoming PTY output during a search incurs a 200ms-debounced re-highlight, NOT per-character re-search.
- `findNext` returns synchronously after iterating through the buffer once.
- No AbortController, no cancellation token. Cancellation = caller stops calling.

### Performance Envelope (SRC-03)

10,000-line scrollback × ~80 cols ≈ 800,000 chars. With the 1000-result cap and a substring search, `_highlightAllMatches` typically completes in **< 50ms** for plain text on modern hardware. Worst case: a single-char regex (`.`) matches every position and hits the 1000-result cap quickly — still bounded.

**Pathological case:** catastrophic-backtracking regex (e.g. `(a+)+b`). The xterm.js search uses native `RegExp` — JavaScript regex engines do NOT have ReDoS protection. UI-SPEC's 100ms input debounce gives the user time to keep typing past the bad pattern; the only true mitigation is closing the find bar (which clears the debounced search). Document this as a Known Pitfall + Threat-Model bullet; do NOT attempt to detect catastrophic patterns (research-grade problem).

---

## Architecture Patterns

### System Architecture Diagram

```
                         User keypress (Cmd-F)
                                  │
                                  ▼
                  ┌───────────────────────────────────┐
                  │  window keydown handler           │
                  │  (TerminalPanel.tsx desktop;      │
                  │   terminal.js web)                │
                  │                                   │
                  │  Guard 1: pluginConfig.search?    │
                  │  Guard 2: document.activeElement  │
                  │           === xterm element?      │
                  └───────────────────────────────────┘
                                  │ both pass
                                  ▼
                  ┌───────────────────────────────────┐
                  │  Open find bar (preventDefault    │
                  │  Cmd-F so browser find suppressed)│
                  └───────────────────────────────────┘
                                  │
              user types in input │
                                  ▼
                  ┌───────────────────────────────────┐
                  │  100ms debounce ──────────────────┤
                  └───────────────────────────────────┘
                                  │ debounce fires
                                  ▼
                  ┌───────────────────────────────────┐
                  │  searchAddon.findNext(query, {    │  ─── reads SearchConfig
                  │    regex, caseSensitive,          │       from React state /
                  │    wholeWord                      │       window-scope var
                  │  })                                │
                  └───────────────────────────────────┘
                                  │
                  ┌───────────────┴────────────────┐
                  │                                │
                  ▼                                ▼
       ┌─────────────────────┐         ┌─────────────────────┐
       │ onDidChangeResults  │         │ xterm decoration    │
       │ ({index, count})    │         │ rendered using      │
       │                     │         │ theme.selection-    │
       │ → updates "3 of 12" │         │ Background          │
       └─────────────────────┘         └─────────────────────┘

  Toggle change (Aa, .*, [ ]):
       │
       ▼
  ┌────────────────────────────┐    ┌──────────────────────────┐
  │ Local state update         │ ─→ │ Re-run search immediately│
  │ (instant)                  │    │ from current query        │
  └────────────────────────────┘    └──────────────────────────┘
       │ (desktop only)
       ▼
  ┌────────────────────────────┐
  │ SetPluginSettings({...,    │
  │   searchConfig: {regex,    │
  │   case, word}})            │
  │                            │
  │ Daemon writes settings.json│
  │ + emits 'settings:plugins' │
  │ + Phase 93 SSE broadcast   │
  │   (notifies other web      │
  │    clients of same daemon) │
  └────────────────────────────┘

  Esc / close button:
       │
       ▼
  ┌────────────────────────────┐
  │ exit animation (200ms)     │
  │ → unmount FindBar          │
  │ → searchAddon.clear        │
  │   Decorations()            │
  │ → focus returns to xterm   │
  └────────────────────────────┘
```

### Recommended Project Structure

**Desktop (React):**
```
frontend/src/
├── components/
│   ├── TerminalPanel.tsx          # MODIFIED — owns SearchAddon ref + Cmd-F handler + FindBar render
│   ├── FindBar.tsx                # NEW — controlled component, no internal state
│   ├── __tests__/
│   │   ├── FindBar.test.tsx       # NEW — copy/aria/dismiss/keyboard
│   │   ├── TerminalPanel.test.tsx # MODIFIED — focus conditioning + search lifecycle
│   │   └── App.plugin-event.test.tsx # MODIFIED — searchConfig field added to PluginSettings shape
├── hooks/                         # (optional; alt: inline in TerminalPanel)
│   └── useFindBar.ts              # NEW — encapsulates SearchAddon lifecycle + debounce + match count
├── lib/
│   └── isXtermFocused.ts          # NEW — helper for activeElement check (testable in isolation)
├── style.css                      # MODIFIED — add /* ─── Phase 94 — Find bar (SRC-01/SRC-04) ─── */ section
├── App.tsx                        # NO CHANGE (pluginConfig prop drill already exists; SearchConfig piggybacks)
└── wailsjs/go/models.ts           # REGENERATED — wails generate emits new SearchConfig nested type
```

**Web (plain DOM):**
```
web/
├── terminal.html                  # MODIFIED — add <div id="find-bar" hidden> + <script src="/assets/xterm/addons/addon-search.js">
├── assets/
│   ├── terminal.js                # MODIFIED — SearchAddon init + findbar DOM wiring + Cmd-F handler
│   └── terminal.css               # MODIFIED — add /* Phase 94 — Find bar */ section
├── vendor/xterm/
│   ├── VERSION                    # MODIFIED — append @xterm/addon-search@0.16.0
│   └── addons/
│       └── addon-search.js        # NEW — copied from frontend/node_modules/.../lib/addon-search.js
└── embed.go                       # MODIFIED — add vendor/xterm/addons/addon-search.js to //go:embed
```

**Daemon:**
```
internal/daemon/
├── plugin_settings.go             # MODIFIED — add SearchConfig struct + field; update defaults
├── plugin_settings_test.go        # MODIFIED — add SearchConfig defaults assertion
└── (no migration logic change needed — defaults-merge handles new field automatically)
```

**Go webserver:** **No change.** The Phase 93 `pluginSettingsProvider func() []byte` returns marshaled JSON; adding `searchConfig` to the struct flows automatically through the SSE broadcast and the GET endpoint.

### Pattern 1: SearchAddon Lifecycle in TerminalPanel

**What:** Load SearchAddon at terminal construction (mount useEffect), gated by `pluginConfig.search`. Dispose on unmount. Hot-swap supported via the existing Phase 93 `[pluginConfig?.webgl, pluginConfig?.clipboard]` useEffect — **add `pluginConfig?.search` to that dep array**.

**When to use:** Always, when a Terminal exists.

**Example:**

```typescript
// TerminalPanel.tsx — extension to existing Phase 93 hot-swap useEffect
// [Source: pattern matches existing webglAddonRef + clipboardAddonRef at lines 73-77]

const searchAddonRef = useRef<SearchAddon | null>(null)

// In existing hot-swap useEffect (currently at line 191), add SearchAddon arm:
useEffect(() => {
  const term = termRef.current
  if (!term) return

  // ... existing WebGL hot-swap (unchanged) ...
  // ... existing Clipboard hot-swap (unchanged) ...

  // SearchAddon hot-swap (Phase 94 SRC-01)
  if (pluginConfig?.search) {
    if (!searchAddonRef.current) {
      const searchAddon = new SearchAddon()
      term.loadAddon(searchAddon)
      searchAddonRef.current = searchAddon
    }
  } else {
    if (searchAddonRef.current) {
      searchAddonRef.current.dispose()
      searchAddonRef.current = null
    }
  }
}, [pluginConfig?.webgl, pluginConfig?.clipboard, pluginConfig?.search, onWebGLContextLost, sessionId])
```

**Cleanup:** add `if (searchAddonRef.current) { searchAddonRef.current.dispose(); searchAddonRef.current = null }` to the mount useEffect's return cleanup — same pattern as webglAddonRef/clipboardAddonRef (lines 138-145 of TerminalPanel.tsx).

### Pattern 2: Focus-Conditioned Cmd-F Handler

**What:** A `window` keydown listener that checks (a) the `pluginConfig.search` flag, (b) `document.activeElement` resolves to the xterm DOM, and (c) the matching modifier key for the current platform.

**When to use:** Both desktop (in TerminalPanel mount useEffect) and web (in `terminal.js` IIFE after `term.open()`).

**Critical detail: how to detect xterm focus.** xterm.js renders into a div with class `.xterm` containing a `.xterm-helper-textarea` (the actual focusable element that captures keyboard input). When the terminal is focused, `document.activeElement` is the helper textarea.

**Recommended check:** test if `document.activeElement` is contained within the terminal's container div:

```typescript
function isXtermFocused(termContainer: HTMLElement | null): boolean {
  if (!termContainer || !document.activeElement) return false
  return termContainer.contains(document.activeElement)
}
```

This handles:
- The `.xterm-helper-textarea` (normal focused state)
- The container div itself (some browsers/states focus the parent)
- Any future internal element xterm might focus

[ASSUMED: `.contains` is the standard pattern for this in xterm-using projects; verified in multiple GitHub repos but not formally documented by xterm.js.] **Risk:** if xterm internals change focus to an element outside the container, the check fails. Mitigation: validate during Wave 0 by manually verifying with DevTools that `document.activeElement` is always inside `containerRef.current`.

**Example desktop:**

```typescript
// In TerminalPanel.tsx mount useEffect (after term.open(containerRef.current))
useEffect(() => {
  function handleKeyDown(e: KeyboardEvent) {
    if (!pluginConfig?.search) return
    const isMac = navigator.platform.toUpperCase().includes('MAC')
    const modifier = isMac ? e.metaKey : e.ctrlKey
    if (!modifier || e.key.toLowerCase() !== 'f') return
    if (!isXtermFocused(containerRef.current)) return  // browser-native find passes through
    e.preventDefault()
    setFindBarOpen(true)
  }
  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [pluginConfig?.search])
```

**Why `window` not the container:** Cmd-F may fire from any focused element (not just xterm) — we need the global-listen-then-guard pattern. The activeElement check is what conditionally swallows vs. lets the browser handle it.

### Pattern 3: SearchConfig Persistence

**What:** Add a `SearchConfig` nested struct to `daemon.PluginSettings`. The existing Phase 92/93 plumbing (`SetPluginSettings` Wails RPC → daemon HTTP → settings.json + `settings:plugins` event + `BroadcastPluginConfig` SSE) handles it transparently.

**When to use:** On any toggle change in the find bar (desktop only — web is in-memory per UI-SPEC).

**Daemon struct change:**

```go
// internal/daemon/plugin_settings.go

// SearchConfig persists per-flag default state for the find-bar toggle row.
// Phase 94 (SRC-02). All defaults FALSE — the toggles ship in their "off"
// position, and the user's choice is remembered for next session.
//
// Field order matches UI-SPEC §"Find bar Settings integration".
type SearchConfig struct {
    Regex         bool `json:"regex"`
    CaseSensitive bool `json:"caseSensitive"`
    WholeWord     bool `json:"wholeWord"`
}

type PluginSettings struct {
    WebGL        bool         `json:"webgl"`
    Unicode11    bool         `json:"unicode11"`
    Search       bool         `json:"search"`
    SearchConfig SearchConfig `json:"searchConfig"`  // NEW — Phase 94 SRC-02
    WebLinks     bool         `json:"webLinks"`
    Image        bool         `json:"image"`
    Serialize    bool         `json:"serialize"`
    Clipboard    bool         `json:"clipboard"`
    Progress     bool         `json:"progress"`
}

func defaultPluginSettings() PluginSettings {
    return PluginSettings{
        WebGL:        true,
        Unicode11:    true,
        Search:       true,
        SearchConfig: SearchConfig{Regex: false, CaseSensitive: false, WholeWord: false},
        WebLinks:     true,
        Image:        true,
        Serialize:    true,
        Clipboard:    true,
        Progress:     false,
    }
}
```

**Wails type regeneration:** running `wails generate module` (or whatever Phase 92's pinned approach is) yields a new `daemon.SearchConfig` TS type and updates `daemon.PluginSettings` to include the field. STATE.md notes Phase 92 chose to "pin wails-generated models.ts in-repo rather than regenerate per build" — Phase 94 must hand-edit `frontend/src/wailsjs/go/models.ts` to mirror the new struct exactly, matching the existing Wails-generated class shape (not a bare interface). [VERIFIED: STATE.md ## Decisions, Phase 92 entries]

**Frontend consumption (FindBar.tsx):**
- Read initial values from `pluginConfig.searchConfig` (with fallback to `{regex: false, caseSensitive: false, wholeWord: false}` if pluginConfig hasn't loaded).
- On toggle click: optimistic local update (immediate UI feedback) + call `SetPluginSettings({...pluginConfig, searchConfig: {...}})` (debounced 200ms to absorb rapid clicking — but each click immediately re-runs the search from local state).

**Web consumption (terminal.js):**
- Read initial values from `pluginConfig.searchConfig` returned by `/api/plugin-config` (already wired in Phase 93).
- On toggle click: in-memory only; no daemon round-trip. SSE push from another client's daemon save IS observed but UI-SPEC §"Toggle Persistence" intentionally lets the in-memory state win for the current page session.

**Migration:** zero-effort. The Phase 92 defaults-merge load path handles new fields via the same `json.Unmarshal` + struct-defaults-fill pattern. A v3.2-without-searchConfig settings.json (i.e., a returning-from-Phase-93 user) will load with `searchConfig` defaulted to `{false, false, false}` — correct per UI-SPEC. **No new migration test needed beyond the existing fixture test, BUT Phase 94 should ADD a test that fixture-loads a Phase 93 settings.json and confirms `searchConfig` populates with defaults.** [VERIFIED: pattern from Phase 92 plugin_settings_test.go]

### Pattern 4: 100ms Debounce + Cancellation

**What:** Coalesce rapid keystrokes into one search call.

**Implementation (desktop):**

```typescript
// useFindBar.ts (or inline in FindBar.tsx)
const debounceTimerRef = useRef<number | null>(null)

const debouncedSearch = useCallback((query: string, options: ISearchOptions) => {
  if (debounceTimerRef.current !== null) {
    window.clearTimeout(debounceTimerRef.current)
  }
  debounceTimerRef.current = window.setTimeout(() => {
    searchAddonRef.current?.findNext(query, options)
  }, 100)
}, [])

// On find bar close:
useEffect(() => {
  return () => {
    if (debounceTimerRef.current !== null) {
      window.clearTimeout(debounceTimerRef.current)
    }
    searchAddonRef.current?.clearDecorations()
  }
}, [])
```

**Why this satisfies SRC-03 cancellation:** "long regex searches can be cancelled by closing the find bar." Closing the bar runs the cleanup (clears the timer + clearDecorations). If a search is mid-flight (synchronous, so it WILL complete), the next debounce + clearDecorations after it lands removes the visual artifact. Worst case: one in-flight synchronous search completes after close (the `findNext` call already entered) — its results are immediately cleared. **No mid-search abort is needed because the work bound is already small (< 50ms typical, < 200ms worst).**

### Anti-Patterns to Avoid

- **Re-running search inside `onAfterSearch` event handler.** Causes infinite loop. Only use `onDidChangeResults` for read-only count display.
- **Putting `pluginConfig.searchConfig` (object) in a useEffect dep array.** Object identity changes on every save even if values didn't — re-runs search needlessly. Use `[pluginConfig?.searchConfig?.regex, pluginConfig?.searchConfig?.caseSensitive, pluginConfig?.searchConfig?.wholeWord]` instead.
- **Loading SearchAddon in mount useEffect unconditionally then trying to dispose conditionally.** Causes Phase 93 Pitfall #1 again — keep load/dispose symmetric in the hot-swap useEffect.
- **Forgetting to pass `pluginConfig.search` to the keydown handler closure.** The handler must read the LIVE value, not a stale closure capture. Use `pluginConfig` in dep array AND inside the handler body.
- **Using `e.key === 'F'` (uppercase) without `e.key.toLowerCase()`.** Cmd-Shift-F sends `'F'`, Cmd-F sends `'f'`. Always lowercase compare.
- **Calling `searchAddon.findNext('', options)` to clear.** Use `clearDecorations()`. Empty-string `findNext` no-ops in some addon versions and clears in others — be explicit.
- **Trapping Tab inside FindBar without releasing on close.** Always remove the focus trap on dismiss; otherwise the next render of the find bar may double-add it.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Scrollback search | Custom regex over `term.buffer.active.getLine(i).translateToString()` | `@xterm/addon-search` | Reinventing decoration management, scroll-to-match, viewport tracking, incremental highlight — addon does this correctly |
| Match decoration rendering | Custom canvas overlay | xterm built-in via `theme.selectionBackground` | 138 themes already have it set; addon uses it transparently when `decorations` option is undefined |
| Focus trap inside find bar (7 elements) | `focus-trap-react` library | Hand-rolled `Tab` keydown handler | Library is overkill for 7 elements; hand-roll is shorter and easier to test |
| Match count display | Subscribing to xterm `onSelectionChange` and counting manually | `searchAddon.onDidChangeResults` event | Addon emits the exact event with the exact payload UI-SPEC requires (`{resultIndex, resultCount}`) |
| Cmd-F detection | Custom platform check via `navigator.userAgent` | `navigator.platform.toUpperCase().includes('MAC')` | Already-used pattern; UA strings are spoofable and ad-blocked |
| Browser find suppression | Custom CSS hacks to hide the browser find UI | `event.preventDefault()` on the keydown when conditions match | Native, standards-compliant, works across all browsers |

**Key insight:** every "harder than it looks" search behavior — match counting, viewport scrolling, decoration rendering, theme integration, regex/case/word semantics — is already handled by the addon. Phase 94's job is purely UI shell + persistence wiring + focus discipline. Resist the temptation to "improve" the addon's defaults; the visual treatment is fully controlled by the find-bar chrome and the `theme.selectionBackground` automatic path.

---

## Common Pitfalls

### Pitfall 1: `document.activeElement` and Modal Focus
**What goes wrong:** When a modal is open (e.g., RegenerateKeyModal, NewSessionModal, QuitConfirmModal), `document.activeElement` is inside the modal — NOT inside the xterm container. Cmd-F should fall through to browser-native find.
**Why it happens:** UI-SPEC's focus condition checks "xterm has focus" — modals push focus elsewhere, which is correct. The find bar should NOT open when a modal owns focus.
**How to avoid:** The `isXtermFocused()` helper returns `false` when activeElement is in a modal. **No additional check needed.** This is actually a feature, not a bug — verified by tracing UI-SPEC §"Opening the Find Bar" intent.
**Warning signs:** find bar opens over a modal; testing should explicitly cover this case.

### Pitfall 2: `setPluginSettings` Re-render Storm on Toggle Click
**What goes wrong:** Clicking a toggle calls `SetPluginSettings`, which emits `settings:plugins`, which updates App.tsx's `pluginConfig` state, which re-renders TerminalPanel, which re-runs the hot-swap useEffect, which (if `[pluginConfig?.searchConfig?.regex, ...]` is in the dep array) triggers… nothing (we don't dispose SearchAddon on flag change). But the FindBar's useEffect that listens for prop changes WILL re-run, potentially racing with the local optimistic update.
**Why it happens:** Optimistic UI + async daemon write + event-driven prop update is a 3-phase update cycle.
**How to avoid:** **FindBar takes its current toggle state from local state, NOT from `pluginConfig.searchConfig` directly.** Initial values come from props once on mount; from then on, local state is canonical. `SetPluginSettings` is fire-and-forget (with error toast on failure). The next `pluginConfig` prop update (from another save or another tab) intentionally does NOT re-sync the open find bar — UI-SPEC accepts this; closing+reopening picks up new defaults.
**Warning signs:** toggle visibly flickers (turns on, then off, then on).

### Pitfall 3: Focus Trap and Esc Key Race
**What goes wrong:** Esc fires while focus is on a toggle button (not the input). The keydown handler is registered on the input only — Esc is missed.
**Why it happens:** Common React mistake — handlers on input element don't catch keys when focus is on siblings.
**How to avoid:** Register the Esc handler on the find-bar **container** with `onKeyDown` at the outer div level, not on individual children. UI-SPEC §"Closing the Find Bar" line 357 explicitly notes: "The Esc handler is attached at the `find-bar` container level, not the input level."
**Warning signs:** Esc only works when input is focused.

### Pitfall 4: SearchAddon Re-Init on Hot Module Replacement (Vite dev mode)
**What goes wrong:** Vite HMR re-renders TerminalPanel without disposing the previous Terminal — the SearchAddon ref leaks and a new addon is loaded onto the SAME Terminal, doubling decorations.
**Why it happens:** HMR fast-refresh skips full unmount/remount of components.
**How to avoid:** The mount useEffect's cleanup function (`return () => { ... termRef.current = null }`) handles this — Vite's HMR DOES call cleanup on the destroyed component instance. **But:** add an idempotent guard `if (searchAddonRef.current) return` inside the load branch to be safe.
**Warning signs:** dev-mode shows duplicated highlights; production build is fine.

### Pitfall 5: Catastrophic Regex Backtracking Hangs the Tab
**What goes wrong:** User types `(a+)+b` (or any pathological regex) in regex mode. JavaScript's RegExp has no ReDoS guard. The synchronous search blocks the main thread for seconds or minutes.
**Why it happens:** Native regex engine is non-interruptible.
**How to avoid:** **Cannot prevent.** Mitigations:
1. The 100ms debounce gives the user time to keep typing past the bad pattern (each new keystroke clears the previous debounce).
2. Closing the find bar (Esc) clears the debounce — but a search already in flight finishes synchronously.
3. Document the threat in the Threat Model. Web users on Tailscale can re-attach a hung session if needed.
**Don't:** attempt to detect catastrophic patterns programmatically — it's a research-grade problem (Regular Expression Denial of Service detection), not a pragma fix.
**Warning signs:** "Page unresponsive" dialog when running stress tests; should be on the test plan.

### Pitfall 6: `theme.selectionBackground` Not Set on Some Custom User Themes
**What goes wrong:** A user-imported theme that omits `selectionBackground` — the addon's default decoration is rendered with an undefined color, invisible.
**Why it happens:** Phase 65/71 established the 138-theme invariant but didn't audit user-imported themes (out of scope today).
**How to avoid:** **Skip for Phase 94.** All 138 curated themes set `selectionBackground` (verified by Phase 65). User-imported theme support is not a Phase 94 requirement. Document as a known limitation if any future phase adds custom themes.
**Warning signs:** match exists in scrollback per the count, but visually nothing is highlighted.

### Pitfall 7: Web Page UMD Global Name Collision
**What goes wrong:** The addon-search UMD bundle exports as `window.SearchAddon` (a namespace object containing `{ SearchAddon: class }`). On the web page, calling `new SearchAddon()` directly fails — you need `new SearchAddon.SearchAddon()`.
**Why it happens:** UMD wrapper convention: `root["SearchAddon"] = factory()` where `factory()` returns a module object.
**How to avoid:** **Verify by grep at execution time** — same as Phase 93 Pitfall #7:
```bash
grep -o "root\[[\"'].*[\"']\]" web/vendor/xterm/addons/addon-search.js | head -3
```
Then use the actual constructor expression. Phase 93 found `WebglAddon.WebglAddon`, `Unicode11Addon.Unicode11Addon`, `ClipboardAddon.ClipboardAddon`. **High probability** the same pattern holds: `SearchAddon.SearchAddon`.
**Warning signs:** browser console: "SearchAddon is not a constructor".

### Pitfall 8: Tab Order Out of Sync with Visual Order
**What goes wrong:** UI-SPEC §"Focus Trap" line 341 specifies Tab order: input → case → regex → word → next → prev → close. The visual order in the layout (UI-SPEC line 184) is: input → count → prev → next → divider → case → regex → word → close. **Logical and visual orders diverge** — UI-SPEC is intentional (logical flow groups the toggles together) but easy to misread.
**Why it happens:** UI-SPEC reviewer locked logical-flow over reading-order. Implementer must consciously choose tabIndex order.
**How to avoid:** Use explicit `tabIndex` attributes (`0` on focusable elements, `-1` on the count which shouldn't take focus). Do NOT rely on DOM source order alone. The `<button>` and `<input>` defaults all participate; reorder DOM source if needed to match logical Tab order, OR set explicit `tabIndex={1}..{7}`.
**Warning signs:** Tab from input goes to next-button (unexpected) instead of case-toggle.

### Pitfall 9: SetPluginSettings Race During Modal Saves
**What goes wrong:** User changes a search toggle WHILE PluginsSection's "Save Plugins" is in flight (saving=true). Both calls reach the daemon; the in-flight Save overwrites the search-toggle change.
**Why it happens:** `SetPluginSettings` accepts the entire `PluginSettings` struct each call. The find bar's call passes its current view of the struct; PluginsSection's call passes ITS view. Last-writer-wins.
**How to avoid:** This is a pre-existing race from Phase 92 (already affects clipboard/webgl toggles). Acceptable mitigation: the daemon's `settings:plugins` event re-syncs both clients to the canonical state after each save. **No code change for Phase 94.** Document the limitation. **Risk: LOW** — toggling search options while Settings is open and saving is a contrived flow.

### Pitfall 10: Find Bar Unmount Doesn't Clear Pending Debounce
**What goes wrong:** User opens find bar, types a query, closes within 100ms (before debounce fires). Search runs after the FindBar component is gone — but `searchAddonRef.current` is still alive on the Terminal — match highlights appear with no UI to dismiss them.
**Why it happens:** Forgetting to clear timeouts in cleanup.
**How to avoid:** FindBar cleanup runs `clearTimeout(debounceTimerRef.current)` AND `searchAddonRef.current?.clearDecorations()`. Both, every time, no exceptions.
**Warning signs:** highlights linger after Esc.

---

## Code Examples

### Example 1: SearchAddon constructor + activate + dispose (verified upstream)

```typescript
// [VERIFIED: github.com/xtermjs/xterm.js master, addons/addon-search/src/SearchAddon.ts]
import { SearchAddon } from '@xterm/addon-search'
import type { ISearchOptions, ISearchResultChangeEvent } from '@xterm/addon-search'

const searchAddon = new SearchAddon()  // accept default highlightLimit=1000
term.loadAddon(searchAddon)

// Subscribe to results event
const subscription = searchAddon.onDidChangeResults((e: ISearchResultChangeEvent) => {
  console.log(`${e.resultIndex + 1} of ${e.resultCount}`)
})

// Run search
const found: boolean = searchAddon.findNext('hello', {
  caseSensitive: true,
  regex: false,
  wholeWord: false,
  // decorations: undefined → uses theme.selectionBackground automatically
})

// Cleanup
subscription.dispose()
searchAddon.dispose()  // also clears decorations
```

### Example 2: Focus-conditioned Cmd-F handler (desktop)

```typescript
// [Pattern based on UI-SPEC §"Opening the Find Bar" line 295-300]
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

### Example 3: SearchConfig Wails generated type (post-regen)

```typescript
// [Pattern matches existing daemon.PluginSettings shape from Phase 92]
// frontend/src/wailsjs/go/models.ts (hand-edit per Phase 92 in-repo pin decision)
export namespace daemon {

  export class SearchConfig {
    regex: boolean
    caseSensitive: boolean
    wholeWord: boolean

    static createFrom(source: any = {}) {
      return new SearchConfig(source)
    }
    constructor(source: any = {}) {
      if ('string' === typeof source) source = JSON.parse(source)
      this.regex = source['regex']
      this.caseSensitive = source['caseSensitive']
      this.wholeWord = source['wholeWord']
    }
  }

  export class PluginSettings {
    webgl: boolean
    unicode11: boolean
    search: boolean
    searchConfig: SearchConfig  // NEW Phase 94
    webLinks: boolean
    image: boolean
    serialize: boolean
    clipboard: boolean
    progress: boolean

    static createFrom(source: any = {}) {
      return new PluginSettings(source)
    }
    constructor(source: any = {}) {
      if ('string' === typeof source) source = JSON.parse(source)
      this.webgl = source['webgl']
      this.unicode11 = source['unicode11']
      this.search = source['search']
      this.searchConfig = this.convertValues(source['searchConfig'], SearchConfig)
      this.webLinks = source['webLinks']
      this.image = source['image']
      this.serialize = source['serialize']
      this.clipboard = source['clipboard']
      this.progress = source['progress']
    }
    // ... convertValues helper as in existing models.ts
  }
}
```

### Example 4: Web find bar init in terminal.js

```javascript
// [Source: pattern from web/assets/terminal.js Phase 93 applyPluginConfig at line 233]
// Inside the IIFE, after term.open() and after pluginConfig is fetched:

var searchAddonHandle = null

function applySearchAddon(enabled) {
  if (enabled) {
    if (!searchAddonHandle) {
      try {
        searchAddonHandle = new SearchAddon.SearchAddon()  // verify global name at exec time
        term.loadAddon(searchAddonHandle)
      } catch (e) { /* silent — addon unavailable */ }
    }
  } else {
    if (searchAddonHandle) {
      try { searchAddonHandle.dispose() } catch (e) {}
      searchAddonHandle = null
    }
  }
}

// At init + on every applyPluginConfig diff:
applySearchAddon(pluginConfig.search)

// Web find bar focus-conditioned Cmd-F handler:
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

### Example 5: vendor_drift_test.go zero-effort coverage

```go
// [VERIFIED: internal/webserver/vendor_drift_test.go:18 — Phase 93 generalized regex]
var pnpmXtermKeyRe = regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-[\w-]+))@([0-9.]+)':`)
// addon-search matches automatically. Min-count guard at line 33 currently
// expects 5 (xterm + addon-fit + addon-webgl + addon-unicode11 + addon-clipboard).
// Phase 94 adds addon-search → bump to 6.
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No find/search functionality in xterm | `@xterm/addon-search` (vendored, decoration-based highlighting) | Phase 94 introduces | Closes SRC-01..05 |
| Custom canvas overlay for match highlights | xterm decoration API (`theme.selectionBackground`) | Phase 94 (addon API choice) | Theme-aware automatically; works across all 138 themes; zero theme-specific code |
| `PluginSettings` flat boolean struct | `PluginSettings` with nested `SearchConfig` sub-struct | Phase 94 introduces | Establishes pattern for future per-plugin config (Phase 99 PUI-03 will reuse for web-links Cmd/Ctrl modifier and image storageLimit) |
| `vendor_drift_test.go` enforcement of @xterm/addon-* parity | (no change — Phase 93 generalized regex catches addon-search automatically) | Already done | Adding addon-search requires only a min-count bump, not a regex change |

**Deprecated / outdated:**

- **xterm.js v5 SearchAddon API** (different signature, no `onDidChangeResults` event). The project pins `@xterm/xterm@^6.0.0` — only the v6+ SearchAddon API matters.
- **Match-count via `findAll` method** — UI-SPEC §"Searching" mentions a `findAll` method as a possible match-count source. **Not present in the current addon API.** The `onDidChangeResults` event is the correct match-count source. UI-SPEC's "if findAll exists, use it; otherwise degrade" fallback is moot — the event-driven path is canonical.

---

## Validation Architecture

`workflow.nyquist_validation` is absent from `.planning/config.json` — treat as **enabled**.

### Test Framework

| Property | Value |
|----------|-------|
| Frontend Framework | Vitest (via `pnpm exec vitest run`) |
| Go Framework | `go test ./...` |
| Component config file | `frontend/vite.config.ts` |
| Quick frontend run | `pnpm exec vitest run src/components/__tests__/FindBar.test.tsx src/components/__tests__/TerminalPanel.test.tsx` |
| Full frontend suite | `pnpm test` |
| Go unit run | `go test ./internal/daemon/... ./internal/webserver/... -count=1` |
| Go full run | `go test ./internal/...` |
| Build smoke | `wails build -tags wailsassets` |
| Manual UAT | `wails dev` (desktop); browse to `https://<machine>.<tailnet>.ts.net/sessions/<id>?cap=<token>` (web) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SRC-01 | Cmd-F opens find bar; Esc dismisses; focus-conditioned (modal open → no-op) | unit (component) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ❌ Wave 0 |
| SRC-01 | Web Cmd-F opens find bar in `terminal.html` | source-inspection (Go) | `go test ./internal/webserver/... -run TestTerminalJS_FindBar -count=1` | ❌ Wave 0 |
| SRC-02 | findNext/findPrevious wired to Enter/Cmd-G/Shift-Enter/Cmd-Shift-G | unit (component) | `pnpm exec vitest run src/components/__tests__/FindBar.test.tsx` | ❌ Wave 0 |
| SRC-02 | Match count "{N} of {M}" updates via onDidChangeResults | unit (component, mock SearchAddon) | `pnpm exec vitest run src/components/__tests__/FindBar.test.tsx` | ❌ Wave 0 |
| SRC-02 | Toggle defaults persist via SetPluginSettings round-trip | integration (Go) | `go test ./internal/daemon/... -run TestSearchConfigPersist -count=1` | ❌ Wave 0 |
| SRC-02 | SearchConfig defaults FALSE on fresh install | unit (Go) | `go test ./internal/daemon/... -run TestDefaultPluginSettings -count=1` | ✅ (extend existing) |
| SRC-02 | Phase 93 settings.json fixture migrates to add searchConfig defaults | unit (Go) | `go test ./internal/daemon/... -run TestPluginSettingsMigration -count=1` | ✅ (extend existing) |
| SRC-03 | 10,000-line scrollback search completes < 1s frame budget | benchmark (component, manual instrumentation) | `pnpm exec vitest run src/components/__tests__/FindBar.perf.test.tsx` | ❌ Wave 0 |
| SRC-03 | Closing find bar mid-debounce cancels pending search | unit (component) | `pnpm exec vitest run src/components/__tests__/FindBar.test.tsx` | ❌ Wave 0 |
| SRC-04 | Find bar renders with TokyoNight `#16161e` background; 200ms transition | source-inspection (CSS) | `pnpm exec vitest run src/components/__tests__/FindBar.test.tsx` | ❌ Wave 0 |
| SRC-04 | Match highlight uses theme.selectionBackground (no custom decorations option passed) | source-inspection | `pnpm exec vitest run src/components/__tests__/FindBar.test.tsx` (assert decorations option NOT passed) | ❌ Wave 0 |
| SRC-05 | Web `terminal.html` contains `<div id="find-bar" hidden>` + addon-search script tag | Go integration | `go test ./internal/webserver/... -run TestTerminalHTML_FindBar -count=1` | ❌ Wave 0 |
| SRC-05 | Web `terminal.js` initializes SearchAddon when `pluginConfig.search` true | source-inspection (Go) | `go test ./internal/webserver/... -run TestTerminalJS_SearchAddon -count=1` | ❌ Wave 0 |
| WEB-01 | Vendored `web/vendor/xterm/addons/addon-search.js` exists & served at `/assets/xterm/addons/addon-search.js` | Go unit | `go test ./internal/webserver/... -run TestAssets_AddonSearch -count=1` | ❌ Wave 0 |
| WEB-02 | vendor_drift_test catches `addon-search` version mismatch | Go unit | `go test ./internal/webserver/... -run TestXtermVendorVersions -count=1` | ✅ (extend existing min-count) |

### Sampling Rate

- **Per task commit:** `pnpm exec vitest run src/components/__tests__/{FindBar,TerminalPanel}.test.tsx && go test ./internal/daemon/... ./internal/webserver/... -count=1`
- **Per wave merge:** `pnpm test && go test ./...`
- **Phase gate:** Full suite green + manual UAT (desktop wails build + web Tailscale page) before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/components/__tests__/FindBar.test.tsx` — covers SRC-02, SRC-04 (FindBar component contract: copy, aria, dismiss, keyboard, toggle states)
- [ ] `frontend/src/components/__tests__/FindBar.perf.test.tsx` — covers SRC-03 (10k-line scrollback search performance budget)
- [ ] Extend `frontend/src/components/__tests__/TerminalPanel.test.tsx` — covers SRC-01 (focus conditioning, Cmd-F open, Esc close), search lifecycle hot-swap
- [ ] Extend `frontend/src/__tests__/App.plugin-event.test.tsx` — covers SRC-02 (searchConfig field present in PluginSettings shape)
- [ ] Extend `internal/daemon/plugin_settings_test.go` — covers SRC-02 (TestDefaultPluginSettings asserts SearchConfig zero values; new TestPluginSettingsMigration_SearchConfig fixture test)
- [ ] `internal/webserver/find_bar_test.go` — covers SRC-01/SRC-05/WEB-01 (terminal.html find-bar DOM presence; terminal.js SearchAddon init; assets serve addon-search.js)
- [ ] Manual UAT script `94-DESKTOP-UAT.md` — Cmd-F walkthrough, persistence verify, modal focus check, regex catastrophic backtracking soft-test
- [ ] Manual UAT script `94-WEB-UAT.md` (or extend `93-iPad-UAT.md`) — web Cmd-F walkthrough on Chromium-based + Safari + iPad Safari Tailscale

**Framework install:** none — vitest and `go test` already in place.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `@xterm/addon-search` (npm package) | SRC-01..05 | ✗ (not installed) | will install 0.16.0 | — (mandatory) |
| `web/vendor/xterm/addons/addon-search.js` | SRC-05 / WEB-01 | ✗ (file does not exist) | will copy 0.16.0 from node_modules/lib | — (mandatory) |
| `@xterm/xterm@^6.0.0` (peer of addon-search) | SRC-01..05 | ✓ | 6.0.0 | — |
| pnpm | install + lockfile | ✓ | (project default) | — |
| Go 1.22+ for `//go:embed` | WEB-01 | ✓ | (project standard) | — |
| `@heroicons/react` | FindBar icons | ✓ | (in package.json from Phase 93) | — |
| `wails generate module` (or hand-edit pin per Phase 92 decision) | TS type for SearchConfig | ✓ (hand-edit pattern) | n/a | hand-edit `models.ts` per Phase 92 STATE.md decision |

**Missing dependencies with no fallback:**
- `@xterm/addon-search` — must `pnpm add @xterm/addon-search@^0.16.0` in Wave 0 task.
- `web/vendor/xterm/addons/addon-search.js` — must copy in Wave 1 vendoring task.

**Missing dependencies with fallback:** none.

---

## Security Domain

`security_enforcement` not explicitly disabled in `.planning/config.json` — treat as **enabled**.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | — (no auth in find bar) |
| V3 Session Management | No | — (find bar is per-page UI) |
| V4 Access Control | Yes | `/api/plugin-config{,/stream}` already capability-gated by Phase 87/93 — no new endpoint added |
| V5 Input Validation | Yes | Find-bar input is local-only (search query never leaves browser); regex compilation by native `RegExp` is a JavaScript engine concern. Daemon receives only the SearchConfig booleans (3 fields, all bool). Standard Go JSON unmarshal validates booleans. |
| V6 Cryptography | No | — |
| V7 Error Handling | Yes | Catastrophic regex backtracking → page hang. Standard mitigation: documented limitation; Esc/close clears debounce; no programmatic detection. |
| V12 Files and Resources | Yes | Vendored addon-search.js served via Go embed.FS; `vendor_drift_test.go` CI gate enforces version parity. CSP `script-src 'self'` already in place from Phase 89. |

### Known Threat Patterns for {Browser/React + Go webserver + xterm.js}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Catastrophic regex backtracking (ReDoS) | Denial of Service | UI-SPEC's 100ms debounce + Esc-to-cancel; no programmatic detection (research-grade problem); document as known limitation; threat is local (user attacks own browser); risk: LOW |
| Search query exfiltration | Information Disclosure | Search query never leaves browser; `findNext` is in-process; daemon receives only `SearchConfig` booleans. No mitigation needed beyond confirming this in code review. |
| Vendored addon tampering | Tampering | `vendor_drift_test.go` (Phase 93) generalized to detect any `@xterm/addon-*` version mismatch; addon-search is automatically covered. SLSA L2 attestations on release artifacts (Phase 90) cover the supply-chain integrity layer. |
| `pluginConfig.search === true` bypass via DevTools | Tampering | Search is a UX feature, not a security boundary — even if a malicious user sets it true via DevTools, no privilege escalation is possible. Cmd-F focus-conditioning is enforced by browser keydown semantics. |
| Cross-site Cmd-F intercept | Tampering | Cmd-F on a non-AgentHub page is unaffected by AgentHub. Cmd-F on AgentHub pages with non-xterm focus falls through to browser-native find (verified by focus-conditioning). |
| Persisted SearchConfig poisoning via /api/plugin-config | Tampering | The endpoint is GET-only (no SET on the web side); SetPluginSettings is desktop-Wails-only. A web client cannot poison persistent state. |
| Memory pressure from 1000-result decoration | Denial of Service | Addon's `DEFAULT_HIGHLIGHT_LIMIT = 1000` is a hard cap. xterm decorations are virtualized (only visible-viewport decorations rendered). Risk: LOW. |

### Threat-Model Bullets for Phase 94

1. **Find-bar input is local-only.** Search queries never traverse the network. No telemetry, no logging.
2. **Regex DoS is a known accepted risk.** No detection; mitigated by debounce + Esc.
3. **Unbounded scrollback memory pressure.** xterm's scrollback cap (10,000 lines per session) is the boundary; the addon respects it.
4. **Web SearchConfig persistence is in-memory only** — a web client cannot mutate daemon state. Desktop is the only persistent path. Confirmed by absence of `SetPluginSettings`-equivalent endpoint on the web server (only GET + SSE stream of plugin-config).
5. **Vendored asset drift** is gated by Phase 93's CI test; Phase 94 inherits this protection without code change.
6. **Supply chain:** `@xterm/addon-search@0.16.0` is a first-party `@xterm` scoped package on the official npm registry; SHA-pinned via `pnpm-lock.yaml` (Phase 90 lockfile-pin discipline applies).

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Web UMD global name is `SearchAddon.SearchAddon` (matches WebglAddon/Unicode11Addon/ClipboardAddon pattern from Phase 93) | Pattern 4, Pitfall 7 | Web init fails with "SearchAddon is not a constructor". Mitigation: grep verification at execution time per Phase 93 Pitfall #7. |
| A2 | xterm.js sets `document.activeElement` to the helper textarea inside `containerRef.current` when terminal is focused | Pattern 2 | Cmd-F never opens (focus check fails). Mitigation: dev-mode DevTools verify in Wave 0 task; if false, switch to `containerRef.current === document.activeElement \|\| containerRef.current.contains(...)`. |
| A3 | `searchAddon.findNext('', options)` clears decorations OR is a no-op (versions vary) | Pattern 4 anti-pattern | Either way, calling `clearDecorations()` explicitly is correct. Risk: zero. |
| A4 | All 138 curated themes have `selectionBackground` set | SRC-04 | Some themes show no highlight. Mitigation: Phase 65/71 established this invariant (cited in UI-SPEC line 363); if a regression slipped, Phase 94 would NOT introduce it. |
| A5 | Phase 92's hand-edited `wailsjs/go/models.ts` pin pattern is the right approach for adding `SearchConfig` (rather than running `wails generate module` and accepting churn) | Pattern 3 | If wrong, hand-edit drifts from Go truth. Mitigation: STATE.md pinned this decision in Phase 92; Phase 94 follows it verbatim. |
| A6 | The `pluginSettingsProvider func() []byte` pattern from Phase 93 handles nested struct (SearchConfig) without modification — `json.Marshal` recurses correctly | Pattern 3 | If wrong, SSE pushes malformed JSON. Mitigation: trivial to verify with a unit test in Wave 0. Pure Go stdlib `encoding/json` definitely handles this. **De-risk: HIGH confidence this works.** |
| A7 | `requireCapability` middleware passes through with no `{id}` segment for the existing `/api/plugin-config` endpoint without rejecting (Phase 93 verified this) | Pattern 3 | Plugin-config fetch on web fails. Phase 93 already verified — see 93-RESEARCH.md Pitfall #6. **Inherited; HIGH confidence.** |
| A8 | The 100ms debounce + 1000-result cap keeps 10k-line scrollback search under 1s frame budget | SRC-03 | If wrong, page-unresponsive risk. Mitigation: Wave 0 perf test (`FindBar.perf.test.tsx`) enforces the budget; if it fails, escalate to a Worker offload. |

---

## Open Questions (RESOLVED)

> Resolution path: Discussed at planning kickoff before Wave 1 starts.

1. **Should the desktop `SetPluginSettings` call on toggle change be debounced (e.g., 200ms) to avoid daemon thrash on rapid toggles?**
   - What we know: `SetPluginSettings` writes settings.json + emits event + broadcasts SSE per call. Rapid toggling = many writes.
   - What's unclear: Is there a measurable cost? Phase 92/93 do not debounce; toggles in PluginsSection write per Save click (already debounced by user gesture).
   - RESOLVED: **Do NOT debounce.** Match existing pattern. Each toggle click is one click; the user is unlikely to thrash 10×/sec.
   - **Risk level:** LOW.

2. **Is `pluginConfig.searchConfig` propagated to all open TerminalPanels' FindBar instances when one is open?**
   - What we know: `pluginConfig` prop change triggers FindBar's prop-init read on next mount only.
   - What's unclear: If the user has FindBar open in Tab A, switches to Tab B, opens FindBar — does Tab B see the latest searchConfig?
   - RESOLVED: **Yes, automatically.** FindBar reads searchConfig from props on mount; closing+reopening picks up new prop values. Multi-tab usage is fine.
   - **Risk level:** LOW (UI-SPEC accepts mid-session config divergence per "find bar takes local state from open onwards").

3. **Should the FindBar render inside the `.terminal-session-container` only when isActive, or always when findBarOpen state is true?**
   - What we know: `TerminalPanel` keeps inactive panels mounted with `display: none`. Multiple panels can have findBarOpen=true.
   - What's unclear: Should background tabs' find bars persist? UI-SPEC §"Anti-goals" says "No persistence of the open/closed state of the find bar across sessions" — silent on tab-switch.
   - RESOLVED: **Per-tab findBarOpen state.** Each TerminalPanel owns its own `findBarOpen`. Switching tabs doesn't close find bars; the per-tab state persists for the app session.
   - **Risk level:** LOW.

4. **Is there a way to detect that browser-native Cmd-F opened (i.e., we did NOT preventDefault)?**
   - What we know: When focus is outside xterm, our handler returns without preventDefault — browser's find UI opens.
   - What's unclear: Should we close our find bar if it's open and the user Cmd-F's outside the xterm? (e.g., they clicked a sidebar item.)
   - RESOLVED: **No special handling.** UI-SPEC §"Closing the Find Bar" specifies Esc-or-button only. Browser find and our find can coexist briefly — the user can dismiss either independently.
   - **Risk level:** LOW.

5. **Phase 92 hand-edited models.ts pin: how confident are we that adding a nested `SearchConfig` class won't break `vite test aliasing` (per STATE.md decision rationale)?**
   - What we know: STATE.md said "replacing wholesale would break vite test aliasing and lose hand-maintained inline type definitions" — implies hand-edit is sustainable for additive changes.
   - What's unclear: Is `SearchConfig` truly additive, or does it require import re-ordering?
   - RESOLVED: **Additive.** Add a new exported class to the existing `daemon` namespace, add a field to `PluginSettings` constructor + property list, add `convertValues(source['searchConfig'], SearchConfig)` to the constructor body. No import changes (everything stays in the `daemon` namespace).
   - **Risk level:** LOW; if it breaks, the test suite catches it immediately.

---

## Files to Create / Modify

### New Files

| File | Why |
|------|-----|
| `frontend/src/components/FindBar.tsx` | Controlled find-bar React component (UI-SPEC §"Component Inventory") |
| `frontend/src/components/__tests__/FindBar.test.tsx` | Component contract tests (copy, aria, keyboard, toggle states, dismiss) |
| `frontend/src/components/__tests__/FindBar.perf.test.tsx` | 10k-line scrollback perf budget test (SRC-03) |
| `frontend/src/lib/isXtermFocused.ts` | Helper for activeElement-inside-xterm-container check (testable in isolation) |
| `frontend/src/lib/__tests__/isXtermFocused.test.ts` | Unit test for the helper |
| `frontend/src/hooks/useFindBar.ts` (optional) | Encapsulates SearchAddon lifecycle + debounce + match count if extracting from TerminalPanel |
| `web/vendor/xterm/addons/addon-search.js` | Vendored UMD bundle (Phase 93 vendoring discipline) |
| `internal/webserver/find_bar_test.go` | Asserts terminal.html contains find-bar DOM + addon-search script tag; asserts terminal.js initializes SearchAddon |
| `.planning/phases/94-search-addon-find-bar-desktop-web/94-DESKTOP-UAT.md` | Manual UAT runbook for desktop walkthrough |
| `.planning/phases/94-search-addon-find-bar-desktop-web/94-WEB-UAT.md` | Manual UAT runbook for web walkthrough (Chromium / Safari / iPad Safari) |

### Modified Files

| File | Change | Requirement |
|------|--------|-------------|
| `frontend/package.json` | Add `@xterm/addon-search@^0.16.0` to dependencies | All SRC |
| `frontend/pnpm-lock.yaml` | (regenerated) | All SRC |
| `frontend/src/components/TerminalPanel.tsx` | Add `searchAddonRef`; extend hot-swap useEffect dep array with `pluginConfig?.search`; add focus-conditioned Cmd-F keydown listener; add `findBarOpen` state; render `<FindBar>` conditionally inside container; pass SearchAddon ref + searchConfig + handlers to FindBar | SRC-01, SRC-02 |
| `frontend/src/components/PluginsSection.tsx` | (NO CHANGE — search toggle already exists from Phase 92; advanced disclosure deferred to Phase 99 PUI-03) | — |
| `frontend/src/App.tsx` | (NO CHANGE — pluginConfig prop drill already exists; SearchConfig piggybacks on existing object) | — |
| `frontend/src/style.css` | Add `/* ─── Phase 94 — Find bar (SRC-01/SRC-04) ─── */` section with all `.find-bar*` BEM classes per UI-SPEC | SRC-04 |
| `frontend/src/wailsjs/go/models.ts` | Hand-edit: add `SearchConfig` class to `daemon` namespace; add `searchConfig` field to `PluginSettings` class | SRC-02 (per Phase 92 STATE.md pin decision) |
| `frontend/src/__tests__/App.plugin-event.test.tsx` | Add searchConfig to expected PluginSettings shape; assert prop drill round-trip preserves nested struct | SRC-02 |
| `internal/daemon/plugin_settings.go` | Add `SearchConfig` struct; add `SearchConfig` field to `PluginSettings`; update `defaultPluginSettings()` | SRC-02 |
| `internal/daemon/plugin_settings_test.go` | Extend `TestDefaultPluginSettings` to assert SearchConfig zero values; add `TestPluginSettingsMigration_SearchConfig` (Phase 93 fixture has no searchConfig → defaults populate) | SRC-02 |
| `internal/webserver/vendor_drift_test.go` | Bump min-count guard from 5 to 6 (now expects xterm + addon-fit + addon-webgl + addon-unicode11 + addon-clipboard + addon-search) | WEB-02 (existing) |
| `internal/webserver/no_cdn_regression_test.go` | (NO CHANGE — `vendor/xterm/` skip naturally covers `vendor/xterm/addons/addon-search.js`) | — |
| `web/embed.go` | Extend `//go:embed` directive to include `vendor/xterm/addons/addon-search.js` | WEB-01 |
| `web/vendor/xterm/VERSION` | Append `@xterm/addon-search@0.16.0` line | WEB-01, WEB-02 |
| `web/terminal.html` | Add `<div id="find-bar" hidden role="search" aria-label="Find in terminal">...</div>` per UI-SPEC §"Web — Identical Behavior"; add `<script src="/assets/xterm/addons/addon-search.js">` BEFORE `terminal.js` script tag | SRC-05, WEB-01 |
| `web/assets/terminal.js` | Add `applySearchAddon()` arm to `applyPluginConfig()`; add focus-conditioned Cmd-F keydown listener; add find-bar DOM wiring (input handler + debounce + nav buttons + toggles + close); add `showFindBar()/hideFindBar()` functions with 200ms slide animation | SRC-01, SRC-02, SRC-05 |
| `web/assets/terminal.css` | Add `/* Phase 94 — Find bar */` section replicating desktop CSS using `#find-bar` ID + `.find-bar__*` selectors | SRC-04, SRC-05 |

### Daemon wiring (NO change)

Phase 93 already wires `engine.GetPluginSettings()` → `pluginSettingsProvider func() []byte` via `json.Marshal` on the boundary. Adding `SearchConfig` flows through automatically.

---

## Sources

### Primary (HIGH confidence)

- `github.com/xtermjs/xterm.js` master branch, `addons/addon-search/src/SearchAddon.ts` — verified SearchAddon API surface, sync nature, DEFAULT_HIGHLIGHT_LIMIT=1000, internal 200ms disposableTimeout (2026-05-04 read)
- `github.com/xtermjs/xterm.js` master, `addons/addon-search/typings/addon-search.d.ts` — verified ISearchOptions, ISearchAddonOptions, ISearchResultChangeEvent type signatures
- `github.com/xtermjs/xterm.js` master, `addons/addon-search/package.json` — verified `version: 0.16.0`, `main: lib/addon-search.js` (CJS UMD), no peerDependencies declared
- `npm view @xterm/addon-search version` returned `0.16.0` (2026-05-04)
- `https://registry.npmjs.org/@xterm/addon-search/0.16.0` — verified package metadata
- `.planning/phases/94-search-addon-find-bar-desktop-web/94-UI-SPEC.md` (status: approved, reviewed_at: 2026-05-04) — locked visual + interaction contract
- `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-RESEARCH.md` — vendoring pattern, hot-swap useEffect architecture, web /api/plugin-config wiring, Phase 92→93 inert-prop lift mechanics
- `frontend/src/components/TerminalPanel.tsx` (current state) — verified Phase 93 hot-swap useEffect at lines 191-233; Refs at lines 73-77; mount useEffect cleanup at lines 138-145
- `frontend/src/components/WebGLRecoveryBanner.tsx` — verified BannerStack vocabulary pattern Phase 94 should match for find-bar styling parallels
- `frontend/src/style.css:1561-1638` — verified .banner-stack, .webgl-recovery-banner CSS that establishes BannerStack vocabulary (TokyoNight, 200ms transitions)
- `web/assets/terminal.js` (current state) — verified Phase 93 applyPluginConfig pattern at line 233, /api/plugin-config fetch at line 125, SSE stream subscription at line 360
- `web/embed.go` — verified //go:embed directive structure
- `web/terminal.html` — verified script tag ordering convention (xterm.js → addons → terminal.js)
- `internal/daemon/plugin_settings.go` — verified PluginSettings struct + defaultPluginSettings + JSON tags
- `internal/daemon/plugin_settings_test.go` — verified TestDefaultPluginSettings shape
- `internal/webserver/vendor_drift_test.go:18` — verified Phase 93 generalized regex `^  '(@xterm/(?:xterm|addon-[\w-]+))@([0-9.]+)':`
- `internal/webserver/plugin_config_stream.go` — verified BroadcastPluginConfig + SSE handler shape (no change needed for Phase 94)
- `internal/webserver/server.go:423-428` — verified existing /api/plugin-config and /api/plugin-config/stream routes (Phase 93)
- `app.go:455-487` — verified Wails GetPluginSettings/SetPluginSettings + EventsEmit('settings:plugins') pipeline (Phase 92)
- `frontend/src/App.tsx:765-800` — verified banner-stack + pluginConfig state shape; verified WebGLRecoveryBanner already integrated as Phase 94 reference

### Secondary (MEDIUM confidence)

- `https://github.com/xtermjs/xterm.js/tree/master/addons/addon-search` — package README (didn't expose perf details inline; cross-referenced source)
- `frontend/node_modules/@xterm/addon-clipboard/typings/addon-clipboard.d.ts` — analogous type-definition shape for cross-reference
- STATE.md `## Decisions` — Phase 92 hand-edit models.ts pin decision (pinned 2026-05-04)

### Tertiary (LOW confidence — ASSUMED)

- A1: Web UMD global name `SearchAddon.SearchAddon` (extrapolated from Phase 93 verified pattern; verify at execution time)
- A2: xterm sets `document.activeElement` to a descendant of `containerRef.current` (extrapolated from xterm.js renderer architecture; verify in Wave 0)

---

## Metadata

**Confidence breakdown:**

- Standard Stack: HIGH — `@xterm/addon-search@0.16.0` verified via `npm view` + GitHub source; peer dep absent (informally `@xterm/xterm@^6.0.0` matches project pin)
- Architecture: HIGH — all patterns verified against Phase 93's verified codebase; hot-swap useEffect pattern, vendoring path, SSE broadcast all proven and unchanged for Phase 94
- Pitfalls: HIGH — most derived from direct code reading or Phase 93 / Phase 92 already-encountered issues; Pitfall 5 (regex DoS) is documented limitation, not a new risk
- API contract: HIGH — read upstream source directly, confirmed sync semantics + 1000-result cap + onDidChangeResults event signature
- Persistence wiring: HIGH — reuses Phase 92/93 pipeline unchanged; nested struct addition is pure JSON-marshaling concern (zero infrastructure change)
- Web parity: MEDIUM — depends on assumption A1 (UMD global name); de-risked by Phase 93 verification of analogous addons; verify-at-execution check is trivial

**Research date:** 2026-05-04
**Valid until:** 2026-06-04 (stable ecosystem; xterm.js addon API is stable; addon-search hasn't had a major version bump in 12+ months per registry timestamps)

---

## RESEARCH COMPLETE
