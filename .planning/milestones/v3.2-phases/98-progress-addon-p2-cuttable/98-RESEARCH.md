# Phase 98: Progress Addon (P2 — Cuttable) — Research

**Researched:** 2026-05-08
**Domain:** xterm.js `@xterm/addon-progress` (OSC 9;4 ConEmu progress sequence), per-tab progress underline UI on `TabBar.tsx`, cross-platform Wails v2 system-tray icon update with rate-limited aggregate quartile glyph, Phase-92 Settings persistence pipeline (default-OFF), and the explicit "P2 / cuttable" architecture invariant
**Confidence:** HIGH

---

## Summary

Phase 98 ships an OSC 9;4 progress reporter end-to-end: the addon parses the sequence emitted by long-running CLIs (e.g. `pip install`, AI CLIs reporting percent), the parsed value drives a per-tab underline animation, and the cross-session aggregate drives a quartile-glyph swap on the system-tray icon. The phase is **explicitly cuttable** — if Phases 95 or 96 over-run, the entire phase defers to v3.3 with zero impact on other v3.2 phases. **Five loud findings shape the plan:**

1. **`@xterm/addon-progress@0.2.0` exists, MIT-licensed, zero deps, zero CSP-relevant constructs.** [VERIFIED: `pnpm view @xterm/addon-progress` 2026-05-08 — `latest: 0.2.0, deps: none, license: MIT, unpackedSize: 23.1 kB, main: lib/addon-progress.js, types: typings/addon-progress.d.ts`.] [VERIFIED: `/tmp/progress-inspect/package/lib/addon-progress.{js,mjs}` and `package/src/ProgressAddon.ts` direct read 2026-05-08 — zero matches for `WebAssembly|new Worker|URL.createObjectURL|new Blob|importScripts|eval\(|new Function|blob:|data:text/javascript|setInterval|setTimeout|requestAnimationFrame`.] No CSP amendment is required; Phase 96's `script-src 'self' 'wasm-unsafe-eval'` carries forward unchanged.

2. **The addon API is a single source of truth: `progressAddon.onChange((state: IProgressState) => void)` plus a getter/setter `progress: { state: 0|1|2|3|4, value: 0..100 }`.** [VERIFIED: `package/typings/addon-progress.d.ts` source-read 2026-05-08.] State semantics: `0=remove, 1=set, 2=error, 3=indeterminate, 4=pause`. The addon already clamps `value` to `[0,100]` and ignores out-of-range states with a `console.warn`. **The plan does NOT write its own OSC 9;4 parser**; the addon does it (registers a `parser.registerOscHandler(9, ...)` for `9;4;<state>;<value>`).

3. **OSC 9;4 events arrive in the renderer (frontend xterm.js side), not in the Go daemon.** The `parser.registerOscHandler` is an xterm.js core method; sequences are intercepted post-relay-frame, in the Terminal instance's parser. Implications: (a) per-tab progress lives in `TerminalPanel` state (or an App-level registry mirror of Phase 97's saver-registry); (b) cross-session aggregation MUST happen frontend-side; (c) the Go daemon only learns of progress via a thin Wails RPC if the frontend pushes an aggregate to it (for tray update). The Go-side relay byte stream is opaque to the daemon — it does NOT and should NOT parse OSC 9;4.

4. **Cross-platform tray icon update API is a single Go signature already used in this codebase: `(*App).updateTray(sessions, connected)` calling out to platform-specific `updateTrayIcon([]byte)` (darwin cgo), `wt.SetIcon(hIcon)` (Windows wail-style), and `tray.iconPixmap` swap (Linux D-Bus).** [VERIFIED: `tray.go` (darwin) line 89-122; `tray_windows.go` line 392-399; `tray_linux.go` line 325-326, 413-415 source-read 2026-05-08.] Existing icons are 18×18 PNG files at `assets/tray_icon.png` and `assets/tray_icon_error.png`. The phase needs **5 new tray-icon PNGs** (or 4 + reuse of the base icon) for quartiles `0% / 25% / 50% / 75% / 100%` (or just 4 if 0% means "no progress active → show base icon"). The right tray-API call to extend is `(*App).updateTray` — adding a new optional aggregate-progress quartile parameter and routing the right PNG bytes through the existing `updateTrayIcon` cgo / `SetIcon` Win32 / `iconPixmap` D-Bus paths.

5. **The "default-OFF + cuttable" architecture has a clean shape: every Phase 98 artifact is gated on `pluginConfig.progress`; nothing constructs the addon, listens for events, mounts the underline, or updates the tray-icon-with-quartile-bytes when the toggle is OFF.** This means: (a) the addon-progress import in `TerminalPanel.tsx` must use a hot-swap useEffect (mirror of WebGL/Clipboard/Search/WebLinks/Serialize arms — NOT mount-only) so toggling Progress in Settings adds/removes the addon live without restart; (b) tray-icon PNG assets ship under a Phase 98 build tag or an `embed.FS` directive that always embeds them but only consumes them when `pluginConfig.progress` is true — the smaller cost wins (5 × ~1 KB PNGs add <5 KB to the binary); (c) the App-level aggregate-progress store is an empty `Map<sessionId, IProgressState>` when nobody is emitting progress, and the tray-icon update fires `cleared → base icon` only the first time the map empties out (a "transition guard" — see Pitfall #3).

The remaining work is mechanical:

- Install `@xterm/addon-progress@^0.2.0` (run `cd frontend && pnpm add @xterm/addon-progress@^0.2.0`)
- Vendor `lib/addon-progress.js` to `web/vendor/xterm/addons/addon-progress.js`
- Append `@xterm/addon-progress@0.2.0` to `web/vendor/xterm/VERSION`
- Bump `vendor_drift_test.go` min-count from 9 to 10 (Phase 97 closed at 9)
- Add `<script src="/assets/xterm/addons/addon-progress.js"></script>` to `web/terminal.html`
- Hot-swap useEffect arm in `TerminalPanel.tsx` keyed on `pluginConfig?.progress` (mirror of Phase 97 serialize arm)
- App-level `progressRegistry: Map<sessionId, IProgressState>` + `handleProgressChange(sessionId, state)` callback; per-tab state used to drive an underline class on `<div class="tab tab--progress" style="--progress: 47%">` (or via the existing animation/state pattern)
- New CSS rule: `.tab__progress` (a thin underline with `width: var(--progress)` transition) — pattern parallels `.tab--exiting` and the existing `.tab--active` border-bottom
- Cross-session aggregate computed in App.tsx (median or average across all sessions in `state: 1` or `state: 4`; ignore `state: 0` (cleared) and `state: 3` (indeterminate))
- New Wails RPC `(*App).SetTrayProgress(quartile int)` — quartile ∈ {0, 1, 2, 3, 4} where 0 = no progress active (use base icon); throttled at 200ms via App-side `requestAnimationFrame`-or-`setTimeout` debounce before the RPC fires (Pitfall #5)
- New tray PNG assets: `assets/tray_icon_progress_25.png`, `..._50.png`, `..._75.png`, `..._100.png` — same 18×18 dims, same TokyoNight palette, single horizontal fill bar across the bottom (or a quarter-arc, designer's discretion)
- New PluginsSection caption under the Progress toggle: `"Default OFF in v3.2 — flips ON in v3.3 after field validation."`
- New Playwright e2e fixture that emits OSC 9;4 sequences and asserts the underline appears
- Negative regression test — the OFF path must produce zero addon construction (mirror of Phase 97 SER-03 negative grep pattern)

**Web parity scope:** the desktop tray glyph has no web equivalent (a browser tab favicon could in principle do this, but it's a separate-and-larger problem and not in PRG-03 scope). The web client SHOULD honor the underline (PRG-02 reads as a visual affordance, not a desktop-specific one), but the tray-icon aggregate (PRG-03) is desktop-only. **Recommendation:** ship the underline on both desktop and web (vendored UMD copy + per-client OSC handler in `web/assets/terminal.js`), and ship the tray glyph desktop-only (Wails-runtime-bound, no web equivalent). The vendored `addon-progress.js` ships under `web/vendor/xterm/addons/` regardless to keep the WEB-01 vendoring discipline whole and `vendor_drift_test.go` green.

**Primary recommendation:** Install `@xterm/addon-progress@^0.2.0`; vendor it under `web/vendor/xterm/addons/addon-progress.js`; bump `vendor_drift_test.go` min-count 9→10; construct the addon in TerminalPanel's hot-swap useEffect (specific-key dep `pluginConfig?.progress`, mirror of WebGL/Clipboard/Search/WebLinks/Serialize arms); subscribe to `progressAddon.onChange` and forward each `IProgressState` to App.tsx via a new `onProgressChange(sessionId, state)` callback prop; in App.tsx maintain `progressRegistry: Map<sessionId, IProgressState>`, derive a CSS-custom-property-driven `--progress: <value>%` per tab, and compute a cross-session aggregate (round-down to quartile 0/1/2/3/4); throttle the aggregate update at 200ms before invoking new `(*App).SetTrayProgress(quartile)` Wails RPC; in app.go implement `SetTrayProgress` to swap the current tray PNG bytes between 5 embedded variants and call the existing `updateTrayIcon` / `SetIcon` / `iconPixmap` paths; write 4 new 18×18 PNG quartile glyph assets at `assets/tray_icon_progress_{25,50,75,100}.png`; ship the v3.2 toggle default OFF (already wired in Phase 92's `defaultPluginSettings` — `Progress: false`); add the italic caption noting v3.3 default-flip; and make every artifact gate-able on `pluginConfig.progress` so cuttability is structural, not promised.

**No CSP amendment needed.** The pre-phase audit (§"Mandatory Pre-Phase CSP Audit") found 0 occurrences of `WebAssembly|new Worker|URL.createObjectURL|new Blob|eval|new Function|blob:|data:text/javascript|importScripts` in `addon-progress.{js,mjs,ts}`. The addon is pure JS string parsing over the xterm.js OSC handler hook plus an internal Emitter (borrowed via `(terminal as any)._core._onData.constructor()` — a known upstream FIXME pending xterm#5283). [VERIFIED: `package/src/ProgressAddon.ts` line 85 — `this._onChange = new (terminal as any)._core._onData.constructor();`]

---

## Project Constraints (from CLAUDE.md)

- **JS/TS:** `camelCase` vars, `PascalCase` components, ESLint + Prettier, TypeScript types — applies to TerminalPanel hot-swap arm, `lib/aggregateProgress.ts` helper, `TabBar.tsx` underline rendering.
- **Node:** `pnpm` (project default). Add `@xterm/addon-progress@^0.2.0` as a regular dependency (not devDep), matching the runtime-dep pattern of all other `@xterm/addon-*` packages (Phase 96 IMG-01 + Phase 97 SER-01 precedent).
- **Go:** `go fmt`, context-aware functions; new `(*App).SetTrayProgress(quartile int) error` accepts an int, returns an error (maintains the existing `(*App).Foo` cgo-callout pattern).
- **No global npm installs.**
- **NEVER kill node.exe** — Claude Code runs as Node.
- **LSP first** for code navigation — applies to discovering existing tray-icon update sites (verified: `(*App).updateTray` at `app.go:946-947`, `tray.go:89-122`, `tray_windows.go`, `tray_linux.go`; this is the canonical extension point).
- **UAT via dev-browser skill** for browser-based verifications — relevant for the web-parity underline check.
- **Wails build requires `-tags wailsassets`** for production builds (project memory feedback). Tray icon PNG embedding via `//go:embed` does NOT require the wailsassets tag — it's a separate `embed.FS` directive in `tray.go` / `tray_linux.go` / `tray_windows.go`.
- **Don't delete test artifacts early** — applies to OSC 9;4 fixture scripts and Playwright artifacts; preserve until user confirms verification complete.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PRG-01 | User can enable OSC 9;4 progress support in Settings (default OFF in v3.2; flips to ON in v3.3 after field validation). | Phase 92's `PluginSettings.Progress: bool` already exists and defaults `false` (verified via `internal/daemon/plugin_settings.go:83, 111`). The PluginsSection toggle row already exists (verified via `frontend/src/components/PluginsSection.tsx:143-144`). New work: add an italic caption under the Progress toggle row using the existing `settings-panel__description settings-panel__description--italic` class (Phase 93/96/97 precedent), text: `"Default OFF in v3.2 — flips ON in v3.3 after field validation."`. Toggle persistence is already wired through `(*App).SetPluginSettings` → daemon settings.json → `settings:plugins` Wails event → App.tsx `EventsOn` subscription → `pluginConfig` prop on TerminalPanel (Phase 92 PLUG-01..PLUG-03). No daemon migration needed (the field has been present and zero-valued false since Phase 92). |
| PRG-02 | When enabled, terminals emitting OSC 9;4 progress sequences show a subtle progress underline on their tab in the tab strip. | Three pieces wire together: (1) `TerminalPanel.tsx` hot-swap useEffect arm (specific-key dep `pluginConfig?.progress`) constructs/disposes a `ProgressAddon` and subscribes to `progressAddon.onChange` (mirror of webgl/clipboard/search/webLinks/serialize arms — Phase 93 P03 + Phase 97 P04 precedent); each `IProgressState` event is forwarded to App.tsx via a new callback prop `onProgressChange(sessionId, state)`; (2) App.tsx maintains `progressRegistry: Map<sessionId, IProgressState>` and threads a per-tab `progress` value (0..100) into the existing `tabs` array shape (or via a parallel `tabProgress: Record<sessionId, number>` map); (3) `TabBar.tsx` renders an underline element scoped to each tab — recommended approach: a `<div class="tab__progress" style={{ width: `${progress}%` }}>` absolutely-positioned at the bottom of `.tab`, with a 200ms `width` CSS transition for smoothness; uses the existing TokyoNight `theme.selectionBackground` palette (#7aa2f7 — same as the `.tab--active` border-bottom). State semantics from `IProgressState`: `state:1` (set value) → show underline at `value%`; `state:2` (error) → show full-width red; `state:3` (indeterminate) → show animated shimmer (CSS keyframe); `state:4` (paused) → show muted at `value%`; `state:0` (remove) → hide underline. **Recommendation: v3.2 ships ONLY `state:1` (set value) and `state:0` (remove). `state:2/3/4` are explicit "Cuttable Inside Cuttable" — the SerializeAddon shape (one option, one default behavior) is the safer scope.** See Pitfall #1 + "Cuttable Inside Cuttable" below. |
| PRG-03 | When enabled, the system tray icon reflects an aggregate progress glyph (quartile indicator) summarizing across all sessions emitting progress; updates do not cause tray icon flicker or excessive system-tray-API churn. | Aggregation lives in App.tsx (the renderer that already owns `progressRegistry`); compute a single 0..100 value as `mean(progressRegistry.values().filter(s => s.state === 1).map(s => s.value))` (median is also defensible — see "Aggregation math" below). Round to a quartile: `0..0 → 0`, `1..25 → 1`, `26..50 → 2`, `51..75 → 3`, `76..100 → 4`. The "no progress active" condition (registry is empty OR all entries have `state:0`) maps to quartile 0 → revert to base tray icon. Quartile transitions fire a new Wails RPC `(*App).SetTrayProgress(quartile int)` — but **debounced/throttled in the renderer at 200ms** (Pitfall #5) so a stream of progress events on a chatty CLI doesn't churn the platform tray API. The Go-side `SetTrayProgress` selects from 5 embedded PNG byte slices (`trayIconBytes`, `trayIconProgress25Bytes`, `..._50`, `..._75`, `..._100`) and routes them through the existing `(*App).updateTray` infrastructure — a small refactor extending `updateTray(sessions, connected)` to `updateTray(sessions, connected, progressQuartile int)`. The cross-platform path is uniform: darwin `C.updateTrayIcon(ptr, len)`, Windows `wt.SetIcon(hIcon)`, Linux `tray.iconPixmap = makePixmap(bytes)`. Flicker mitigation = "transition guard" (Pitfall #3): only call the cgo/C function when the quartile *changes* relative to the last applied value (idempotency check at the Go layer). |
</phase_requirements>

---

<user_constraints>
## User Constraints

> No `98-CONTEXT.md` will be authored. Per [skip-discuss-when-research-complete] memory: when ROADMAP/REQUIREMENTS already pre-answer the gray areas, skip `/gsd-discuss-phase` and proceed to `/gsd-plan-phase`. ROADMAP success criteria + REQUIREMENTS PRG-01..PRG-03 + the explicit "P2 cuttable" framing leave only mechanical and small-discretion questions, all answered below.

### Locked Decisions

- **Phase 98 is explicitly cuttable.** [ROADMAP Phase 98 SC-1 verbatim: "if Phases 95 (web-links security) or 96 (image + CSP) over-run their scope, this entire phase can be deferred to v3.3 with no impact on v3.2 release readiness — its absence does not block any other phase."] The plan must produce a tight, defensible, minimum-viable scope; no over-engineering. Every Phase 98 artifact must be gate-able on `pluginConfig.progress` — landing the toggle OFF (Phase 92 default) leaves zero observable side effects.
- **Default OFF in v3.2.** [ROADMAP Phase 98 SC-2; REQUIREMENTS PRG-01 verbatim: "default OFF in v3.2; flips to ON in v3.3 after field validation"] PluginSettings.Progress already defaults `false` (Phase 92, verified at `internal/daemon/plugin_settings.go:111`). No daemon change needed for defaults. Italic caption under the toggle: `"Default OFF in v3.2 — flips ON in v3.3 after field validation."` (existing `settings-panel__description--italic` class).
- **`@xterm/addon-progress@0.2.0` (latest stable, not 0.3.0-beta).** [Mirror of Phase 96 IMG-01 + Phase 97 SER-01 stable-wins decisions — betas in security-sensitive code paths are a non-starter.] [VERIFIED: `pnpm view @xterm/addon-progress`, 2026-05-08 — `latest: 0.2.0, beta: 0.3.0-beta.216, deps: none, license: MIT, main: lib/addon-progress.js`.]
- **Vendored same-origin under `web/vendor/xterm/addons/addon-progress.js`.** [Phase 93 vendoring discipline → Phase 94/95/96/97 verbatim pattern.] Copy `frontend/node_modules/@xterm/addon-progress/lib/addon-progress.js` (CJS UMD, NOT the `.mjs`) to `web/vendor/xterm/addons/addon-progress.js`. Append `@xterm/addon-progress@0.2.0` to `web/vendor/xterm/VERSION`. Bump `internal/webserver/vendor_drift_test.go` min-count guard from 9 to 10. Add `<script src="/assets/xterm/addons/addon-progress.js"></script>` to `web/terminal.html` (after `addon-serialize.js` to preserve alphabetical script-tag order — see Phase 97 web/terminal.html line 51).
- **Addon construction is HOT-SWAPPABLE (not next-session-only).** [Mirror of WebGL/Clipboard/Search/WebLinks/Serialize hot-swap arms.] Toggling Progress in Settings while a terminal is open should attach/detach the addon live — there is no buffer-state implication. The italic caption under the Progress toggle is the v3.3-FLIP warning, NOT a "next-session-only" caption.
- **OSC 9;4 events arrive in the renderer (frontend), NOT in the Go daemon.** [VERIFIED: `package/src/ProgressAddon.ts` line 50 — `terminal.parser.registerOscHandler(9, ...)`. The xterm.js parser runs in the browser/renderer process; OSC sequences are intercepted post-relay-frame, after the daemon byte stream has been delivered to the renderer.] The Go daemon does NOT and SHALL NOT parse OSC 9;4 from the relay byte stream. Cross-session aggregation lives in App.tsx; the tray-icon update is a thin frontend → Go RPC.
- **MIT license, zero deps, zero CSP-relevant constructs.** [VERIFIED: `pnpm view @xterm/addon-progress license` returned `MIT`; `peerDependencies` is empty; CSP audit (this RESEARCH §"Mandatory Pre-Phase CSP Audit") found 0 occurrences of `WebAssembly|new Worker|URL.createObjectURL|new Blob|eval|new Function|blob:|data:text/javascript|importScripts`.] No CSP amendment needed; Phase 96's `script-src 'self' 'wasm-unsafe-eval'` carries forward unchanged.
- **Web parity scope: underline on web YES, tray glyph on web N/A.** [PRG-03 reads as a desktop-only requirement — system tray is a desktop concept.] The vendored UMD copy of `addon-progress.js` ships in v3.2 to keep the WEB-01 vendoring discipline whole; the web `terminal.js` constructs the addon and renders an underline (parallel of the desktop path). Web client never calls `SetTrayProgress` (no Wails runtime; no system tray to update).
- **Aggregation across sessions = mean of "set value" sessions, ignoring `state:0/3` and treating `state:2/4` as if they were `state:1` for value purposes.** [REQUIREMENTS PRG-03 verbatim: "aggregate progress glyph (quartile indicator) summarizing across all sessions emitting progress"; the most defensible aggregation is mean for cross-session — median is more robust to outliers but harder to explain to users.] Quartile bucketing: `[0, 0] → 0`, `[1, 25] → 1`, `[26, 50] → 2`, `[51, 75] → 3`, `[76, 100] → 4`. Empty registry OR all `state:0` → quartile 0 → revert to base tray icon.
- **Tray-icon update throttle: 200ms debounce in App.tsx before the Wails RPC fires.** [REQUIREMENTS PRG-03 verbatim: "updates do not cause tray icon flicker or excessive system-tray-API churn"; 200ms is the project standard for visual transitions (Phase 94 BannerStack 200ms slide-in/out)] Mirror of `lodash.debounce` style helper but written inline (zero new deps). Final flush on registry-empties-out fires immediately so the user sees the icon revert promptly.
- **Negative-OFF regression test.** [Mirror of Phase 97 SER-03 negative-grep guard.] When `pluginConfig.progress` is false, the codebase MUST: (a) NOT construct a `ProgressAddon`; (b) NOT subscribe to OSC 9 handler; (c) NOT call `SetTrayProgress`. A grep-based test asserts these three negatives — the test is green-by-default and red if a future change accidentally fires progress logic on the OFF path.

### Claude's Discretion

- **State coverage in v3.2: ship `state:1` (set) + `state:0` (remove) only; defer `state:2` (error), `state:3` (indeterminate), `state:4` (pause) to v3.3.** **Recommendation: minimum-viable.** Rationale: (1) the phase is explicitly P2 cuttable, so adding state-2/3/4 work is over-engineering; (2) field-validating the most common path (a CLI reporting a percent) before shipping the rare paths (errored, indeterminate, paused) matches the v3.3 default-flip story (PROG-FUT-01); (3) error-state UX is design-heavy (full-width red bar? red dot? just stop the underline?) and needs real-user feedback; (4) indeterminate-state UX needs an animated shimmer keyframe + an off-frame fallback for users with `prefers-reduced-motion: reduce` — non-trivial CSS work; (5) pause-state is rare in practice. Alternative: ship all 5 states. **Strongly recommend: NO. Ship 1+0; capture states 2/3/4 in deferred-items.md.**

- **Aggregation function: mean vs median.** **Recommendation: mean.** Rationale: (1) easier to explain in the UI-SPEC ("the tray icon shows the average percent across all running tasks"); (2) more responsive to early progress on a fresh session (a single 5%-emitting session pulls the aggregate down predictably); (3) median requires sorting on every event — slightly more compute, marginal but present. Alternative: median (robust to one stuck session at 100%). Both are defensible. Pick mean for simplicity.

- **Tray icon glyph design: bottom horizontal fill bar vs quarter-arc.** **Recommendation: bottom horizontal fill bar — same width as the icon, ~3px tall, accent color from TokyoNight palette (`#7aa2f7`).** Rationale: (1) horizontal bars at the bottom of an icon are a system-tray convention (download progress in macOS Big Sur, app activity indicators on iOS); (2) quarter-arc / radial designs require an existing icon shape that supports them visually — the AgentHub icon doesn't have a natural arc anchor; (3) bottom-bar design is fail-safe at small sizes (18×18 native; 36×36 Retina). Alternative: 4 distinct quarter-rectangle glyphs in the 4 corners (top-left = 25%, top-right = 50%, etc.) — visually clever but cognitively expensive. Designer (or you) finalizes; the plan should accept either as a parameter, not bake the design into the code path. The Go side just consumes 4 PNG byte slices.

- **Whether to ship 0%-state PNG (separate from base icon).** **Recommendation: NO — quartile 0 means "no progress active", so revert to the base `tray_icon.png` that Phase 92 already ships.** Saves one PNG, simplifies the state machine, and avoids the user-confusion case where "0% complete" looks indistinguishable from "no task running." Alternative: ship a fifth PNG showing "no progress" with a subtle empty-bar overlay — over-engineering.

- **Where the throttle/debounce lives — App.tsx (frontend) or app.go (backend).** **Recommendation: App.tsx via a small `useDebouncedCallback` pattern (or inline `useRef<NodeJS.Timeout>`).** Rationale: (1) the renderer can drop intermediate values cheaply (a debounce timer in TS is 2 lines); (2) keeping the throttle in TS avoids round-tripping every progress event over the Wails RPC bridge — only quartile transitions cross the boundary; (3) the Go side becomes simpler — `SetTrayProgress(q)` is idempotent and applies immediately (with the Pitfall #3 transition guard for the cross-platform tray-API churn protection). Alternative: backend-side debounce via a Go ticker — works but doubles the round-trip cost and complicates Go-side state.

- **Whether to expose the underline on the tab strip via CSS custom properties or via a child element.** **Recommendation: child element `<div class="tab__progress" style={{ transform: `scaleX(${progress / 100})` }}>` absolutely-positioned at the bottom of `.tab`, with `transition: transform 200ms`.** Rationale: (1) `transform` is GPU-accelerated and avoids layout reflow on every update — critical for low-flicker (PRG-03 invariant); (2) `transform-origin: left` makes the bar grow left-to-right naturally; (3) the child element approach is testable (`getByTestId("progress-underline")` in vitest); (4) no `style` attribute thrash on the parent `.tab`. Alternative: a CSS custom property (`--progress`) on `.tab` itself with the underline as a `::after` pseudo-element — terser but less testable. Both work; child element is the safer bet.

- **Whether the OSC 9;4 progress event also fires a Wails event for desktop OS-level integration (e.g., macOS dock badge, Windows taskbar progress overlay).** **Recommendation: NO — out of scope for v3.2.** PRG-03 is satisfied by tray glyph alone; dock badge / taskbar progress overlay are different surfaces with different APIs (NSApplication.dockTile, ITaskbarList3) and would extend the phase well past P2. Capture as a future PRG-FUT-* if real users ask. The phase plan mentions this only to lock the boundary.

- **Whether to make the underline color configurable.** **Recommendation: NO — use the TokyoNight accent palette (`#7aa2f7`) verbatim, matching `.tab--active` border-bottom (verified at `frontend/src/style.css:127`).** v3.2 doesn't ship per-plugin theme tokens; adding a `progressColor` setting would balloon the PluginsSection schema. Alternative: pull from `theme.selectionBackground` for visual consistency — defensible but adds runtime indirection. Verbatim accent is simpler.

- **Web-side aggregation/tray-icon.** **Recommendation: skip both — the underline is the only PRG-02 piece on web; PRG-03 is desktop-only.** Web client doesn't have a system tray, so no aggregation is needed. Rationale: keeps web parity scope tight (matches Phase 97's "web vendoring only, no UI" precedent — the addon ships in `web/vendor/xterm/addons/` so vendor_drift_test stays green, and the web `terminal.js` instantiates the addon for the underline only).

- **Where the Playwright e2e fixture script lives.** **Recommendation: extend the existing `frontend/e2e/` (or `frontend/tests-e2e/`) directory; add a single test asserting an OSC 9;4 sequence (`echo -ne "\x1b]9;4;1;47\x07"`) drives the underline to ~47% width.** This mirrors the Phase 94 chromedp e2e and Phase 95 Playwright e2e patterns. Real OSC sequences are tricky to inject via a browser fixture — easier path: write a JS test fixture that calls `progressAddon.progress = { state: 1, value: 47 }` directly and asserts the tab `transform: scaleX(0.47)` style applies. Both paths work; the direct-API path is faster and less flaky.

- **Whether "stuck progress" (a CLI that emits 50% then dies without emitting `state:0`) needs a session-end cleanup.** **Recommendation: YES — when a TerminalPanel unmounts (session closes), the App.tsx progress handler removes that session's entry from `progressRegistry` and recomputes the aggregate.** This is the SerializeAddon-on-detach unregister pattern (Phase 97 Pattern 2 line). Without this, a closed-but-stuck session contributes 50% to the aggregate forever. Alternative: time-out stale entries (drop after N seconds of no events) — overkill; session unmount is the natural boundary.

### Deferred Ideas (OUT OF SCOPE)

- **`state:2` (error) / `state:3` (indeterminate) / `state:4` (pause) UX.** [REQUIREMENTS deferred] Ship in v3.3 along with the default-ON flip; collect field feedback on which states matter.
- **macOS dock badge / Windows taskbar progress overlay.** [Out of scope; PRG-03 is satisfied by tray glyph alone.] Capture as PRG-FUT-OS-CHROME if real users ask.
- **OS-level notification on completion (e.g., bouncing dock icon when a 100% is hit).** Out of scope; the user is already in the app — they see the underline reach 100%.
- **Per-plugin progress color customization.** v3.2 doesn't ship theme tokens for plugins; accent color is verbatim.
- **Aggregating progress across web + desktop sessions on the same daemon.** Out of scope; the daemon doesn't track progress (renderer-side architecture). Cross-client aggregation would require a daemon-side store and a fan-out via the existing `settings:plugins`-style event channel — non-trivial; skip.
- **OSC 9;4 progress on the tab strip in the welcome / settings tab.** No terminal = no progress; the tab only shows the underline when `tab.type === 'terminal'`.
- **Playing a sound on 100%.** Strongly out of scope; sound design isn't AgentHub's surface.
- **Configurable throttle interval.** v3.2 hardcodes 200ms (matches BannerStack precedent).
- **Webshell live-mode tray-icon-equivalent (favicon update).** Browser tab favicons can encode progress (some sites do), but it's a separate problem with cross-browser quirks. Defer.

</user_constraints>

---

## Mandatory Pre-Phase CSP Audit

**Audit target:** Upstream `@xterm/addon-progress@0.2.0` source — both `package/lib/addon-progress.js` (CJS UMD, used for web vendoring) and `package/lib/addon-progress.mjs` (ESM, used by frontend bundler). Audit performed on the upstream tarball at `/tmp/progress-inspect/package/` (downloaded via `npm pack @xterm/addon-progress@0.2.0`, 2026-05-08).

**Audit method:** `grep -cE "WebAssembly|new Worker|URL.createObjectURL|new Blob|importScripts|eval\(|new Function|blob:|data:text/javascript|setInterval|setTimeout|requestAnimationFrame"` against both bundles. Cross-checked against the human-readable upstream source `package/src/ProgressAddon.ts` (102 lines) for any patterns the minifier might have transformed.

### Findings Table

| Pattern Searched | Count in `.mjs` | Count in `.js` | Count in `src/ProgressAddon.ts` | CSP Directive Affected | Mitigation Required |
|------------------|----------------|---------------|---------------------------------|------------------------|---------------------|
| `WebAssembly.*` | 0 | 0 | 0 | `script-src 'wasm-unsafe-eval'` | None |
| `new Worker(` | 0 | 0 | 0 | `worker-src` | None |
| `URL.createObjectURL(` | 0 | 0 | 0 | `img-src` / `connect-src` | None |
| `new Blob(` | 0 | 0 | 0 | n/a | None |
| literal `blob:` URL strings | 0 | 0 | 0 | n/a | None |
| `data:text/javascript` | 0 | 0 | 0 | `script-src` | None |
| `data:application/...` script construction | 0 | 0 | 0 | `script-src` | None |
| `importScripts(` | 0 | 0 | 0 | n/a (no Worker) | None |
| `eval(` | 0 | 0 | 0 | `script-src 'unsafe-eval'` | None |
| `new Function(` | 0 | 0 | 0 | `script-src 'unsafe-eval'` | None |
| `setInterval(` | 0 | 0 | 0 | n/a | None |
| `setTimeout(` | 0 | 0 | 0 | n/a | None |
| `requestAnimationFrame(` | 0 | 0 | 0 | n/a | None |

**Audit Conclusion:** addon-progress is pure JS string parsing over xterm.js's OSC handler hook, plus an internal Emitter borrowed from xterm core via `(terminal as any)._core._onData.constructor()` (a known upstream FIXME pending xterm/xterm.js#5283 — not a CSP concern). Zero CSP-relevant constructs, zero timers, zero workers, zero WASM, zero `blob:`, zero `data:`. No CSP changes required for Phase 98; the Phase 96 IMG-03 amendment (`script-src 'self' 'wasm-unsafe-eval'`) carries forward unchanged.

**Confidence:** HIGH (verified via direct source read of the upstream npm tarball; cross-checked against `package/src/ProgressAddon.ts` 102-line TypeScript source; no new attack surface introduced). [VERIFIED: `/tmp/progress-inspect/package/` 2026-05-08]

---

## ProgressAddon API Contract

### Constructor

```typescript
// [VERIFIED: /tmp/progress-inspect/package/typings/addon-progress.d.ts]
new ProgressAddon()  // no constructor args
```

### Methods + Properties

```typescript
// [VERIFIED: typings/addon-progress.d.ts]

// Activate the addon (called automatically by term.loadAddon).
public activate(terminal: Terminal): void;

// Disposes the addon (cleans up the OSC handler + Emitter).
public dispose(): void;

// Subscribe to progress changes — fires every time the OSC 9;4 sequence updates state or value.
public readonly onChange: IEvent<IProgressState>;

// Get/set the current progress (clamps value to [0, 100]; ignores out-of-bounds state).
public progress: IProgressState;

// Where:
interface IProgressState {
  state: 0 | 1 | 2 | 3 | 4   // 0=remove, 1=set, 2=error, 3=indeterminate, 4=pause
  value: number              // 0..100; clamped on set; ignored when state=3
}
```

### State semantics (verified from `src/ProgressAddon.ts:65-81`)

| state | Meaning | Value | Phase 98 v3.2 Action |
|-------|---------|-------|----------------------|
| 0 | REMOVE — clear progress | resets to 0 | Hide underline; remove from registry |
| 1 | SET — normal percent 0..100 | required | Show underline at value% |
| 2 | ERROR — task errored at value | optional (uses last value if missing) | **Defer to v3.3** (recommendation: hide underline, same as REMOVE) |
| 3 | INDETERMINATE — unknown progress | ignored (uses last value) | **Defer to v3.3** (recommendation: hide underline, same as REMOVE) |
| 4 | PAUSE — paused at value | optional (uses last value if missing) | **Defer to v3.3** (recommendation: hide underline, same as REMOVE) |

### Phase 98 call shape (frontend, TerminalPanel.tsx)

```typescript
// In the hot-swap useEffect (mirror of webgl/clipboard/search/webLinks/serialize arms).
// On attach:
const progressAddon = new ProgressAddon()
term.loadAddon(progressAddon)
progressAddonRef.current = progressAddon

// Subscribe to changes; forward to App.tsx for cross-session aggregation.
progressOnChangeDisposable.current = progressAddon.onChange((s: IProgressState) => {
  onProgressChange?.(sessionId, s)
})

// On detach:
progressOnChangeDisposable.current?.dispose()
progressAddonRef.current?.dispose()
progressAddonRef.current = null
onProgressChange?.(sessionId, { state: 0, value: 0 })  // clear from registry
```

### OSC 9;4 sequence format (for fixture/UAT)

```
\x1b]9;4;<state>;<value>\x07
```

Where `\x1b]` = OSC introducer (`ESC ]`), `9;4;` = identifier, `<state>` = `0..4`, `<value>` = `0..100` integer (optional — defaults to last value when absent), `\x07` = ST (BEL terminator). Example fixture:

```bash
# Set progress to 47%
echo -ne "\x1b]9;4;1;47\x07"
# Clear progress
echo -ne "\x1b]9;4;0\x07"
```

[VERIFIED: `package/src/ProgressAddon.ts:50-83` source-read 2026-05-08; OSC 9 + payload-prefix-check `data.startsWith('4;')` confirm the precise shape.]

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| OSC 9;4 sequence parsing | Browser (xterm.js parser inside `ProgressAddon.activate`) | — | The addon registers `terminal.parser.registerOscHandler(9, ...)` and parses the `4;<state>;<value>` payload itself; no custom parser code needed. |
| Per-session progress event delivery | Browser (`progressAddon.onChange((state) => ...)` inside TerminalPanel hot-swap useEffect) | — | xterm.js Emitter; in-process JS; no native involvement. |
| Cross-session aggregation | Browser (App.tsx maintains `progressRegistry: Map<sessionId, IProgressState>` and computes mean across `state:1` entries) | — | App.tsx is the parent of all TerminalPanels and already owns multi-session state (saver registry, status registry); progressRegistry is a parallel artifact. No daemon-side aggregation. |
| Per-tab underline rendering | Browser (`TabBar.tsx` renders `<div class="tab__progress" style={{ transform: scaleX(...) }}>` per tab; CSS transition handles smooth animation) | — | Pure DOM/CSS; uses `transform` to avoid layout reflow (Pitfall #4 mitigation). |
| Progress underline color (TokyoNight accent) | Browser (CSS — `frontend/src/style.css` `.tab__progress { background: #7aa2f7 }`) | — | Verbatim from existing `.tab--active` border-bottom; no theme indirection. |
| Throttled tray-quartile dispatch | Browser (App.tsx debounces 200ms via `useRef<ReturnType<typeof setTimeout> \| null>` then calls Wails RPC) | — | Throttle in TS; only quartile transitions cross the Wails bridge; minimal RPC chatter. |
| Wails RPC: `SetTrayProgress(quartile)` | Browser → Wails bridge → Go (`(*App).SetTrayProgress` in `app.go`) | — | Mirror of `(*App).SaveTerminalSession` (Phase 97). Accepts `quartile int`, returns `error`. Idempotent: stores last quartile, no-ops if unchanged (Pitfall #3 transition guard). |
| Tray icon byte-slice swap | Go (`tray.go` darwin: `C.updateTrayIcon(ptr, len)` cgo; `tray_windows.go`: `wt.SetIcon(hIcon)` Win32; `tray_linux.go`: `tray.iconPixmap = makePixmap(bytes)` D-Bus) | — | Existing infrastructure already loads `trayIconBytes` and `trayIconErrorBytes`; phase 98 adds `trayIconProgress25Bytes`, `..._50`, `..._75`, `..._100` via `//go:embed` (parallel to existing `//go:embed assets/tray_icon.png`). |
| Tray icon PNG assets (4 quartile glyphs) | CDN / Static (`assets/tray_icon_progress_{25,50,75,100}.png` embedded via `//go:embed`) | — | 4 × ~1 KB = ~4 KB binary growth; negligible. Designer-supplied or generated from base icon + horizontal-fill overlay. |
| Vendored addon serving (web parity) | CDN / Static (`web/vendor/xterm/addons/addon-progress.js` served via Go embed.FS at `/assets/xterm/addons/addon-progress.js`) | — | WEB-01 vendoring discipline. `<script>` tag in `web/terminal.html`. |
| Web-side per-tab underline | Browser (web `terminal.js` constructs `ProgressAddon`, subscribes to `onChange`, dispatches a `progress:change` DOM CustomEvent the page listener consumes) | — | Mirror of desktop path but at IIFE-scope inside `web/assets/terminal.js`; no React. |
| `vendor_drift_test.go` CI gate | CI / Go test | — | Phase 93's generalized regex matches `@xterm/addon-progress` automatically; bump min-count guard from 9 to 10. |
| OFF-path negative regression test | CI / Go test (`internal/release/no_progress_when_off_test.go`) | — | filepath.Walk + regex; verifies `pluginConfig.progress` gates every progress-related code site. |

**Cross-tier note on flicker prevention:** The "no flicker" invariant (PRG-03) is enforced at TWO layers — frontend 200ms debounce (drops intermediate values) AND Go-side idempotency check (no-ops if quartile unchanged). Either layer alone is insufficient: frontend debounce alone could still fire identical-quartile RPCs from a flapping aggregate; Go-side alone would still receive every event over the bridge. Both layers are cheap and additive.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/addon-progress` | `^0.2.0` | Parses OSC 9;4 ConEmu progress sequences and emits `IProgressState` events; getter/setter on `progress` for programmatic control | First-party `@xterm` scoped addon; drop-in compatible with the project's `@xterm/xterm@^6.0.0` core; same family as already-shipped `@xterm/addon-fit`, `@xterm/addon-webgl`, `@xterm/addon-search`, `@xterm/addon-web-links`, `@xterm/addon-image`, `@xterm/addon-unicode11`, `@xterm/addon-clipboard`, `@xterm/addon-serialize`. Latest non-beta release. |

**Verified:** `pnpm view @xterm/addon-progress` returned `0.2.0` (latest), `0.3.0-beta.216` (beta), `deps: none`, `license: MIT`, `main: lib/addon-progress.js`, `module: lib/addon-progress.mjs`, `types: typings/addon-progress.d.ts`, `unpackedSize: 23.1 kB` (verified 2026-05-08). [VERIFIED: npm registry, 2026-05-08 via `pnpm view @xterm/addon-progress`]

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none — addon-progress has zero runtime dependencies) | — | — | Verified by `pnpm view @xterm/addon-progress peerDependencies` returning empty and inspection of `package.json` showing no `dependencies` entry. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `@xterm/addon-progress@0.2.0` | `0.3.0-beta.216` | Beta tag implies API instability; v3.2 ships stable, v3.3 reconsiders. |
| `@xterm/addon-progress` | Custom OSC 9;4 parser via `terminal.parser.registerOscHandler(9, ...)` directly | Re-implements the parsing logic the addon already provides correctly; the addon is 102 lines of careful int parsing + state machine. **Don't hand-roll.** |
| Tray-icon byte-swap PNG assets | SVG rasterized at runtime to PNG bytes | macOS NSStatusItem expects raster PNG/TIFF; runtime SVG rasterization adds a Go dependency (e.g. `github.com/srwiley/oksvg`). Static PNGs at build time are simpler. |
| 4 quartile PNGs | 5 PNGs (separate "0% / no progress" glyph) | "0% complete" looks identical to "no task running" — confusing; revert to base icon at 0%. |
| Cross-platform Wails-tray API | `github.com/getlantern/systray` | Project already has `tray.go` / `tray_windows.go` / `tray_linux.go` with direct platform-API integration (cgo darwin, win32 Windows, D-Bus Linux). Switching to systray would lose existing Phase-82 minimize-to-tray work. |
| Render underline via `transform: scaleX()` | Render underline via `width: NN%` | Width changes cause layout reflow on every animation frame; transform stays on the GPU compositor. **Use transform.** |
| Aggregation in App.tsx | Aggregation in Go daemon | Go daemon doesn't see OSC 9;4 (parsed renderer-side); piping every progress event to Go and back would double the bridge traffic. Frontend aggregation is correct. |

**Installation:**

```bash
cd frontend && pnpm add @xterm/addon-progress@^0.2.0
```

After install, copy `frontend/node_modules/@xterm/addon-progress/lib/addon-progress.js` to `web/vendor/xterm/addons/addon-progress.js` (Phase 93/94/95/96/97 pattern; byte-identical to source). Append `@xterm/addon-progress@0.2.0` to `web/vendor/xterm/VERSION`. Bump `internal/webserver/vendor_drift_test.go` min-count guard from 9 to 10. Add `<script src="/assets/xterm/addons/addon-progress.js"></script>` to `web/terminal.html` after `addon-serialize.js` (alphabetical-ish; see Phase 97 web/terminal.html line 51).

**Version verification:**

```bash
pnpm view @xterm/addon-progress version  # confirmed 0.2.0 on 2026-05-08
```

[VERIFIED: npm registry, 2026-05-08]

---

## Architecture Patterns

### System Architecture Diagram

```
                    ┌──────────────────────────────────────────────┐
                    │  Running CLI emits OSC 9;4 sequence:         │
                    │  ESC ] 9 ; 4 ; <state> ; <value> BEL          │
                    └────────────────┬─────────────────────────────┘
                                     │ via the relay byte stream → renderer
                                     ▼
       ┌────────────────────────────────────────────────────┐
       │ Terminal (xterm.js, in TerminalPanel.tsx)          │
       │  parser.registerOscHandler(9, ...)                 │
       │  ProgressAddon parses 4;<state>;<value>            │
       │  Sets internal _st + _pr; fires onChange Emitter   │
       └─────────────────────┬──────────────────────────────┘
                             │ progressAddon.onChange → IProgressState
                             ▼
       ┌────────────────────────────────────────────────────┐
       │ TerminalPanel.tsx (hot-swap useEffect arm)         │
       │  - if pluginConfig?.progress: construct addon      │
       │  - subscribe; forward to App via                   │
       │     onProgressChange(sessionId, state)             │
       │  - on detach: dispose; emit { state: 0, value: 0 } │
       └─────────────────────┬──────────────────────────────┘
                             │ onProgressChange(sessionId, state)
                             ▼
       ┌────────────────────────────────────────────────────┐
       │ App.tsx                                            │
       │  - progressRegistry: Map<sessionId,IProgressState> │
       │  - tabProgress: Record<sessionId,number> for tabs  │
       │  - on each event:                                  │
       │      registry.set(sessionId, state)                │
       │      tabProgress[sessionId] = state.value          │
       │      computeAggregate() → quartile [0..4]          │
       │      debouncedSetTrayProgress(quartile, 200ms)     │
       └────┬──────────────────────────────────┬───────────┘
            │ tabProgress prop                  │ Wails Call
            ▼                                   ▼
       ┌──────────────────────────┐  ┌────────────────────────────┐
       │ TabBar.tsx               │  │ app.go (Go)                │
       │  per tab:                │  │  (*App).SetTrayProgress(q) │
       │  <div class="tab">       │  │  - if q == lastQ: return   │
       │   ... tab__name ...      │  │  - select PNG bytes        │
       │   <div class="tab__      │  │     q=0 → trayIconBytes    │
       │     progress" style=     │  │     q=1 → progress25Bytes  │
       │     {transform:scaleX(   │  │     q=2 → progress50Bytes  │
       │     progress/100)}/>     │  │     q=3 → progress75Bytes  │
       │  </div>                  │  │     q=4 → progress100Bytes │
       └──────────────────────────┘  │  - call updateTrayIcon     │
                                     │     (cgo darwin /          │
                                     │      SetIcon Windows /     │
                                     │      iconPixmap Linux)     │
                                     └────────────────────────────┘

  Web parity (no tray):

       ┌────────────────────────────────────────────────────┐
       │ web/assets/terminal.js                             │
       │  if (pluginConfig.progress) {                      │
       │    var pa = new ProgressAddon.ProgressAddon()      │
       │    term.loadAddon(pa)                              │
       │    pa.onChange(function(s) {                       │
       │      // dispatch CustomEvent on document so        │
       │      // a parent page (if any) can hear it; or     │
       │      // mutate a DOM element directly.             │
       │      // For v3.2: the web client's terminal page   │
       │      // doesn't have a tab strip → underline goes  │
       │      // on the <h1> session title or a thin        │
       │      // <progress> bar at the top of the page.     │
       │    })                                              │
       │  }                                                 │
       └────────────────────────────────────────────────────┘
```

### Recommended Project Structure (additions only)

```
frontend/src/
├── components/
│   ├── TabBar.tsx                # EXTEND: render <div class="tab__progress"> per tab
│   ├── TerminalPanel.tsx         # EXTEND: hot-swap arm + progressAddonRef + onProgressChange
│   ├── PluginsSection.tsx        # EXTEND: italic v3.3-flip caption under Progress toggle
│   └── __tests__/
│       ├── TabBar.test.tsx                # EXTEND: progress underline renders at correct width
│       ├── TerminalPanel.test.tsx         # EXTEND: addon attach/detach + onProgressChange dispatch
│       └── PluginsSection.test.tsx        # EXTEND: v3.3-flip caption present
└── lib/
    ├── aggregateProgress.ts      # NEW: pure helper — Map<id,IProgressState> → quartile [0..4]
    ├── debounce.ts               # NEW (optional) — useDebouncedCallback hook (or inline in App.tsx)
    └── __tests__/
        └── aggregateProgress.test.ts  # NEW: vitest cases for empty/single/multi-session aggregation

internal/release/
└── no_progress_when_off_test.go  # NEW: PRG-OFF negative-grep regression test

web/vendor/xterm/
├── addons/
│   └── addon-progress.js         # NEW: vendored UMD copy
└── VERSION                       # EXTEND: append @xterm/addon-progress@0.2.0

internal/webserver/
└── vendor_drift_test.go          # EXTEND: bump min-count from 9 to 10

frontend/src/wailsjs/go/main/
├── App.d.ts                      # EXTEND: SetTrayProgress signature
└── App.js                        # EXTEND: SetTrayProgress Call() stub

assets/
├── tray_icon_progress_25.png     # NEW: 18×18 PNG, base + bottom 25% fill bar
├── tray_icon_progress_50.png     # NEW: 18×18 PNG, base + bottom 50% fill bar
├── tray_icon_progress_75.png     # NEW: 18×18 PNG, base + bottom 75% fill bar
└── tray_icon_progress_100.png    # NEW: 18×18 PNG, base + bottom full fill bar

app.go                            # EXTEND: (*App).SetTrayProgress method + lastTrayQuartile field
tray.go                           # EXTEND: //go:embed for 4 quartile PNGs + updateTray quartile arg
tray_windows.go                   # EXTEND: same //go:embed + SetIcon path through quartile selector
tray_linux.go                     # EXTEND: same //go:embed + iconPixmap path through quartile selector
tray_common.go                    # OPTIONAL: helper to map quartile int → PNG bytes selector

web/terminal.html                 # EXTEND: <script src="…/addon-progress.js"></script> tag
web/assets/terminal.js            # EXTEND: ProgressAddon construction + onChange handler

frontend/src/style.css            # EXTEND: .tab__progress rule + transition + accent color
```

### Pattern 1: Hot-Swap Addon Arm (mirror of WebGL/Clipboard/Search/WebLinks/Serialize)

**What:** ProgressAddon attaches/detaches live based on `pluginConfig?.progress`; no terminal re-mount required.

**When to use:** Any plugin whose load/unload has no buffer-state implications (the ProgressAddon doesn't modify the buffer; it only registers an OSC handler).

**Example:**

```typescript
// Source: extends TerminalPanel.tsx hot-swap useEffect
// Pattern verified against existing webgl/clipboard/search/webLinks/serialize arms.

// ADD to dep array:
//   [pluginConfig?.webgl, pluginConfig?.clipboard, pluginConfig?.search,
//    pluginConfig?.webLinks, pluginConfig?.serialize, pluginConfig?.progress,
//    onWebGLContextLost, onProgressChange, sessionId]

if (pluginConfig?.progress) {
  if (!progressAddonRef.current) {
    const progressAddon = new ProgressAddon()
    term.loadAddon(progressAddon)
    progressAddonRef.current = progressAddon
    // Subscribe; capture disposable for clean teardown.
    progressOnChangeDisposable.current = progressAddon.onChange((state) => {
      onProgressChange?.(sessionId, state)
    })
  }
} else {
  if (progressAddonRef.current) {
    progressOnChangeDisposable.current?.dispose()
    progressOnChangeDisposable.current = null
    progressAddonRef.current.dispose()
    progressAddonRef.current = null
    // Clear from registry on detach so a stuck progress doesn't linger.
    onProgressChange?.(sessionId, { state: 0, value: 0 })
  }
}
```

### Pattern 2: Progress Registry (TerminalPanel → App.tsx) — mirror of Phase 97 saver registry

**What:** App.tsx holds `progressRegistry: Map<sessionId, IProgressState>`. Each TerminalPanel reports state changes via the `onProgressChange(sessionId, state)` callback. App.tsx derives per-tab progress AND a cross-session aggregate.

```typescript
// In App.tsx:
const progressRegistry = useRef(new Map<string, IProgressState>())
const [tabProgress, setTabProgress] = useState<Record<string, number>>({})

const handleProgressChange = useCallback((sessionId: string, state: IProgressState) => {
  // Phase 98 v3.2 scope: only state:1 (set) and state:0 (remove) drive UI.
  // state:2/3/4 logged as state:0 for v3.2 (deferred to v3.3).
  if (state.state === 1) {
    progressRegistry.current.set(sessionId, state)
    setTabProgress((prev) => ({ ...prev, [sessionId]: state.value }))
  } else {
    progressRegistry.current.delete(sessionId)
    setTabProgress((prev) => {
      const { [sessionId]: _, ...rest } = prev
      return rest
    })
  }

  // Recompute aggregate quartile and dispatch (debounced).
  const quartile = aggregateProgress(progressRegistry.current)
  scheduleSetTrayProgress(quartile)  // 200ms debounce — see Pattern 4
}, [])
```

### Pattern 3: Aggregation Helper (pure function, easy to test)

**What:** A single function `aggregateProgress(registry) → quartile [0..4]`. Mean of `state:1` entries, bucketed.

```typescript
// frontend/src/lib/aggregateProgress.ts
import type { IProgressState } from '@xterm/addon-progress'

export function aggregateProgress(
  registry: Map<string, IProgressState>
): 0 | 1 | 2 | 3 | 4 {
  const values: number[] = []
  for (const s of registry.values()) {
    if (s.state === 1) values.push(s.value)
  }
  if (values.length === 0) return 0
  const mean = values.reduce((a, b) => a + b, 0) / values.length
  if (mean <= 0) return 0
  if (mean <= 25) return 1
  if (mean <= 50) return 2
  if (mean <= 75) return 3
  return 4
}
```

### Pattern 4: Debounced Tray Update (App.tsx)

**What:** A useRef-backed setTimeout that batches quartile transitions at 200ms. Mirror of any standard React debounce idiom.

```typescript
// In App.tsx
const trayDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
const lastDispatchedQuartileRef = useRef<number>(-1)

const scheduleSetTrayProgress = useCallback((quartile: 0 | 1 | 2 | 3 | 4) => {
  if (trayDebounceRef.current) clearTimeout(trayDebounceRef.current)
  trayDebounceRef.current = setTimeout(() => {
    if (lastDispatchedQuartileRef.current === quartile) return  // idempotent
    lastDispatchedQuartileRef.current = quartile
    void SetTrayProgress(quartile)  // Wails RPC
  }, 200)
}, [])

// Cleanup on unmount.
useEffect(() => () => {
  if (trayDebounceRef.current) clearTimeout(trayDebounceRef.current)
}, [])
```

### Pattern 5: Per-Tab Underline Element (TabBar.tsx)

**What:** A child `<div class="tab__progress">` inside each `.tab` element, scaled via `transform`.

```tsx
// In TabBar.tsx — add a progress prop or take it from a tabProgress map.
// Render inside the existing .tab div, after the .tab__name span:

<div
  className="tab__progress"
  style={{
    transform: `scaleX(${(tabProgress?.[tab.sessionId] ?? 0) / 100})`,
  }}
  data-testid={`tab-progress-${tab.id}`}
/>
```

```css
/* frontend/src/style.css */
.tab__progress {
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 2px;
  background: #7aa2f7; /* TokyoNight accent — same as .tab--active border-bottom */
  transform: scaleX(0);
  transform-origin: left;
  transition: transform 200ms ease-out;
  pointer-events: none;
}

.tab {
  position: relative; /* anchor for .tab__progress */
}
```

### Pattern 6: Cross-Platform Tray Icon Update (Go side)

**What:** Extend the existing `(*App).updateTray(sessions, connected)` to a new signature OR add a parallel `(*App).SetTrayProgress(quartile int)` that selects a PNG byte slice and routes through the existing platform-specific update path.

```go
// app.go — new method (recommended; less invasive than changing updateTray's signature)
func (a *App) SetTrayProgress(quartile int) error {
    if !a.trayInit {
        return nil // tray not yet initialized; silent no-op
    }
    if quartile < 0 || quartile > 4 {
        return fmt.Errorf("SetTrayProgress: quartile out of range [0,4]: %d", quartile)
    }
    if a.lastTrayQuartile == quartile {
        return nil // idempotent — Pitfall #3 transition guard
    }
    a.lastTrayQuartile = quartile
    a.refreshTrayState() // triggers updateTray with new quartile-aware logic
    return nil
}
```

```go
// tray.go (darwin) — add embedded PNG bytes
//go:embed assets/tray_icon_progress_25.png
var trayIconProgress25Bytes []byte

//go:embed assets/tray_icon_progress_50.png
var trayIconProgress50Bytes []byte

//go:embed assets/tray_icon_progress_75.png
var trayIconProgress75Bytes []byte

//go:embed assets/tray_icon_progress_100.png
var trayIconProgress100Bytes []byte

// In (*App).updateTray, before C.updateTrayIcon, select the bytes:
func (a *App) trayIconBytesForState(connected bool) []byte {
    if !connected {
        return trayIconErrorBytes
    }
    switch a.lastTrayQuartile {
    case 1: return trayIconProgress25Bytes
    case 2: return trayIconProgress50Bytes
    case 3: return trayIconProgress75Bytes
    case 4: return trayIconProgress100Bytes
    default: return trayIconBytes // 0 or unset → base
    }
}
```

The Linux + Windows paths get equivalent helpers; the existing `updateTray` body changes one line: `bytes := a.trayIconBytesForState(connected)` instead of branching directly on `connected`.

### Pattern 7: Negative Regression Test (PRG-OFF)

**What:** A Go test asserting that when the codebase is built with the OFF path active, no progress-related code runs.

```go
// internal/release/no_progress_when_off_test.go
package release

// Asserts that all references to ProgressAddon construction, OSC 9 handler
// registration, and SetTrayProgress are gated on pluginConfig?.progress.
//
// This is a structural test — it greps source files for forbidden ungated patterns.
// Pattern (a): "new ProgressAddon" must appear inside a block where the same file
//              also has "pluginConfig?.progress" within ~200 chars before it.
// Pattern (b): no top-level (module-scope) construction.
// Pattern (c): "SetTrayProgress(" must appear inside a callback or guard
//              that includes "pluginConfig?.progress" or "tabProgress" in scope.
//
// In practice the simplest assertion: every "new ProgressAddon(" occurrence
// in TS files must be preceded (within the same useEffect block) by
// "if (pluginConfig?.progress)" or "pluginConfig?.progress &&".
//
// Mirror of Phase 97 SER-03 negative-grep regression test pattern.
```

### Anti-Patterns to Avoid

- **Don't parse OSC 9;4 in the Go daemon.** The renderer-side addon does it correctly; piping every progress event to Go and back is wasted bridge traffic. (PRG-03 architecture: aggregation lives in App.tsx; only the quartile transition crosses the bridge.)
- **Don't fire SetTrayProgress on every progress event.** A chatty CLI emits dozens per second; debounce at 200ms in App.tsx (Pitfall #5).
- **Don't change the tray icon if the quartile hasn't changed.** Idempotency check at the Go layer (Pattern 6 + Pitfall #3).
- **Don't animate the underline via `width` changes.** Use `transform: scaleX()` to keep the work on the GPU compositor; `width` causes layout reflow per frame (Pitfall #4).
- **Don't ship `state:2/3/4` UX in v3.2.** P2 cuttable means "minimum viable"; field-validate state:1 before adding the rare paths (locked decision).
- **Don't conflate tray-icon-progress with tray-icon-error.** Both swap the icon bytes, but they're independent concerns. The connected/disconnected error state takes precedence (you can't show progress when the daemon is unreachable).
- **Don't forget to clear the registry entry on TerminalPanel unmount.** A closed-but-stuck session contributes 50% to the aggregate forever (Pattern 1 detach branch + Claude's Discretion #stuck-progress).
- **Don't use `setInterval` to poll progress.** The addon is event-driven; polling is wasteful and adds flicker (Pitfall #6).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| OSC 9;4 sequence parsing | Custom `terminal.parser.registerOscHandler(9, ...)` with manual state-machine | `@xterm/addon-progress.activate()` | The addon is 102 lines of carefully-tested int parsing + state machine; reinventing is silently buggy on edge cases (e.g., `ProgressType.PAUSE` with no value falls back to `_pr` — easy to miss) |
| Progress event emission | Custom `EventTarget` / `BroadcastChannel` between TerminalPanel and App.tsx | `progressAddon.onChange((state) => ...)` callback prop pattern | The IEvent interface is xterm.js's standard; React callback props are the established pattern from Phase 93/97 |
| Cross-session mean computation | Custom statistics class with edge-case handling | `aggregateProgress` pure helper (Pattern 3) | One pure function; vitest covers all edge cases (empty, single, multi, all `state:0`) |
| Tray icon native API | Cgo NSStatusItem / Win32 / D-Bus calls in Phase 98 code | Existing `(*App).updateTray` infrastructure (`tray.go`, `tray_windows.go`, `tray_linux.go`) | Phase 82 already shipped cross-platform tray support; Phase 98 just adds new PNG byte slices and a quartile selector |
| Tray-update debounce | Lodash debounce or rxjs throttle | Inline `useRef<setTimeout>` pattern (Pattern 4) | 8 lines; zero new deps; Phase 94 BannerStack 200ms is already the project's debounce convention |
| Per-tab underline animation | JavaScript-driven animation loop (`requestAnimationFrame`) | CSS `transition: transform 200ms` | The browser's compositor is GPU-accelerated; JS-driven animations break frame budget under tab-strip thrash |
| Aggregate quartile bucketing | Floating-point math with branching | `if (mean <= 25) return 1` ladder (Pattern 3) | 4 conditionals; readable; testable; no off-by-one risk |
| Tray-PNG binary embedding | Runtime file load from disk | `//go:embed assets/tray_icon_progress_*.png` | Existing pattern — Phase 82 `tray_icon.png` and `tray_icon_error.png` use the same `//go:embed` directive |
| Stuck-progress cleanup | Background timer scanning the registry | `onProgressChange(sessionId, { state:0 })` on TerminalPanel unmount | Session lifecycle is the natural boundary; no daemon required |

**Key insight:** Phase 98 is a small phase whose risk is *over-engineering*. The temptation is to ship "all 5 OSC states", "configurable colors", "dock badge integration", "median aggregation", "background poller for stuck progress", "OS-level notifications", "favicon update on web". Each is a feature, none are required, and the phase is explicitly P2 cuttable. **Stick to the prescribed minimum-viable path: state:1 + state:0, mean aggregation, 4 PNG quartile glyphs, 200ms debounce, hot-swap useEffect, vendored UMD copy. Defer everything else.**

---

## Common Pitfalls

### Pitfall 1: Shipping `state:2/3/4` UX in v3.2

**What goes wrong:** The plan tries to handle error/indeterminate/pause states. Each one is a separate UX decision (color? animation? tooltip?). Phase scope balloons; "P2 cuttable" becomes "P2 we wish we'd cut."

**Why it happens:** The IProgressState type union has 5 values; "completeness" tempts the implementer to handle all 5.

**How to avoid:** Treat `state:2`, `state:3`, `state:4` as `state:0` (remove) in v3.2 — log a warning if a CLI emits them so we know the field shape; capture `state:2/3/4` UX as deferred items for v3.3. The IProgressState comparison should be `state.state === 1 ? show : hide` only.

**Warning signs:** A vitest case asserting "indeterminate-state shows shimmer animation" appears in the plan — that's scope creep.

### Pitfall 2: Aggregating Across `state:0` and `state:3` Entries

**What goes wrong:** A naive `mean(values)` includes sessions with `state:0` (cleared, value=0) or `state:3` (indeterminate, value=last) in the average, dragging the aggregate to a misleading number.

**Why it happens:** `state:0` resets value to 0 but leaves the entry in the registry; `state:3` keeps the last value but should not contribute to the running aggregate.

**How to avoid:** Filter the registry to `state === 1` only when computing the aggregate (Pattern 3 enforces this). Sessions with state 0/3 are excluded from the mean.

**Warning signs:** Aggregate bouncing to 0% when one session clears progress while another is at 50% — classic state:0 inclusion bug.

### Pitfall 3: Tray Icon Flicker from Identical-Quartile Updates

**What goes wrong:** A flapping aggregate (e.g., 50.4% → 50.6% → 50.4% → 50.6%) crosses the 50% bucket boundary back and forth, sending identical quartile values to `SetTrayProgress(2)` repeatedly. The frontend debounce drops the duplicates within a 200ms window, but a slow flap (every 250ms) bypasses the debounce and reaches the platform tray API.

**Why it happens:** Float-arithmetic flapping at bucket boundaries is normal; the frontend can't see "I just sent quartile 2" without state.

**How to avoid:** Idempotency check at the Go layer (`if a.lastTrayQuartile == quartile { return nil }`). This is the second-layer "transition guard" (Pattern 6 / Architectural Responsibility Map cross-tier note).

**Warning signs:** macOS Console.app shows repeated `NSStatusItem.button.image = ...` setter calls; visual flicker on the menu bar even when the aggregate is stable.

### Pitfall 4: Underline Animation Causes Layout Reflow

**What goes wrong:** Animating `width: NN%` on the underline causes the browser to re-layout the tab and its descendants on every frame, which (with 8+ open tabs all updating at once) can drop animation frames and cause visible jank.

**Why it happens:** `width` is a layout-affecting property; `transform: scaleX()` is a compositor-only property. Browsers GPU-accelerate transforms but not widths.

**How to avoid:** Use `transform: scaleX(progress / 100)` with `transform-origin: left`. The `transition: transform 200ms ease-out` runs on the compositor thread.

**Warning signs:** DevTools Performance timeline shows "Layout" tasks per animation frame; FPS drops below 60 during heavy progress updates.

### Pitfall 5: Tray-API Churn from Unthrottled Progress Events

**What goes wrong:** A CLI emitting `pip install` progress sends ~10 OSC 9;4 events per second. Without throttling, the frontend fires 10 SetTrayProgress RPCs per second, each crossing the Wails bridge, parsed by Go, dispatched to the platform tray API. macOS NSStatusItem updates within ~16ms but isn't designed for 10 Hz; Linux D-Bus with notify-watcher daemons may rate-limit the publisher; Windows Shell_NotifyIconW serializes queue.

**Why it happens:** The addon fires onChange on every parsed sequence; nothing inherently throttles the cross-tier path.

**How to avoid:** Frontend 200ms debounce (Pattern 4). Most `pip install`-class progress reports are coarse-grained anyway (1% increments at most); 200ms ≈ 5 Hz which is plenty smooth.

**Warning signs:** macOS Activity Monitor shows AgentHub CPU spike on a long pip install; Linux dbus-monitor shows 10+ tray icon updates per second.

### Pitfall 6: Polling Instead of Subscribing

**What goes wrong:** A maintainer adds a `setInterval(() => { ... }, 100)` to "make sure we don't miss any progress events", which runs forever even when no terminal is open and adds CPU overhead.

**Why it happens:** Misunderstanding of the addon's event model — it IS event-driven via `onChange`, no polling needed.

**How to avoid:** PRG-OFF negative regression test (Pattern 7) MUST include a forbidden-pattern assertion: zero matches for `setInterval.*[Pp]rogress` or `setTimeout.*[Pp]rogress.*[0-9]{2,}` (long-delay timers).

**Warning signs:** A `setInterval` reference in any progress-related file other than the existing `startTrayPoller` (which is a 5-second daemon health poll, unrelated).

### Pitfall 7: Missing Cleanup on TerminalPanel Unmount

**What goes wrong:** Session A reaches 50%, then the user closes the tab. The `progressRegistry` still has Session A at 50%. Session B opens, never emits progress. The aggregate stays at 50% forever, tray icon stuck at quartile 2.

**Why it happens:** The detach branch of the hot-swap useEffect must also report `{ state: 0, value: 0 }` to App.tsx so the registry entry is removed.

**How to avoid:** Pattern 1 detach branch explicitly calls `onProgressChange?.(sessionId, { state: 0, value: 0 })` AND App.tsx's handleProgressChange treats state:0 as a registry delete (Pattern 2).

**Warning signs:** Tray icon stuck at non-base quartile after all terminals are closed.

### Pitfall 8: Tray Icon Error State Trampled by Progress

**What goes wrong:** Daemon disconnects → tray icon should show error (red); a pending tray-progress update fires concurrently, overwriting the error icon with a quartile glyph. User sees a 25% progress glyph during a daemon outage.

**Why it happens:** The two paths are independent — error icon swap from `refreshTrayState` and progress icon swap from `SetTrayProgress` race.

**How to avoid:** `trayIconBytesForState(connected)` (Pattern 6) gives error precedence: if `!connected`, always return error bytes regardless of quartile. The progress quartile is only consulted when connected.

**Warning signs:** Tray icon doesn't go red during a deliberate daemon-kill UAT; users report "I lost connection but the icon stayed green-with-progress."

### Pitfall 9: Vendored Addon Drift on Pnpm Update

**What goes wrong:** Future `pnpm update` bumps `@xterm/addon-progress` from 0.2.0 → 0.2.1 in `pnpm-lock.yaml` but doesn't re-copy the vendored UMD; CI fails (Phase 93 generalized vendor_drift_test guards this).

**Why it happens:** Vendoring is a manual cp step; pnpm doesn't know about it.

**How to avoid:** Phase 89 D-04 / Phase 93 WEB-02 guard catches this — the test fails red if `pnpm-lock.yaml` and `web/vendor/xterm/VERSION` disagree on any `@xterm/addon-*` version. The plan must update VERSION + re-copy the UMD whenever the version bumps.

**Warning signs:** CI failure with message "version drift for @xterm/addon-progress: pnpm-lock=0.2.1, VERSION=0.2.0".

### Pitfall 10: Web Client Has No Tab Strip — Where Does the Underline Go?

**What goes wrong:** PRG-02 says "underline on tab strip". The desktop has a tab strip in `TabBar.tsx`. The web client (web-served Tailscale terminal) doesn't have one — each session is its own page.

**Why it happens:** The web client's UI vocabulary is "session viewer" not "tab manager."

**How to avoid:** On web, render the underline at the top of the terminal page (above the xterm container) — a thin horizontal bar matching the desktop tab__progress visually. This satisfies PRG-02's *intent* (visible affordance) without forcing a tab strip into the web layout. Document explicitly so the planner can call it out.

**Warning signs:** Web e2e Playwright assertion fails because no `.tab__progress` element exists on the web page.

---

## Code Examples

Verified patterns from official sources and the AgentHub codebase:

### Constructing the addon (TerminalPanel.tsx)

```typescript
// Source: VERIFIED against /tmp/progress-inspect/package/typings/addon-progress.d.ts
// + extends frontend/src/components/TerminalPanel.tsx hot-swap useEffect (Phase 97 P04 pattern)

import { ProgressAddon, IProgressState } from '@xterm/addon-progress'

const progressAddonRef = useRef<ProgressAddon | null>(null)
const progressOnChangeDisposable = useRef<{ dispose(): void } | null>(null)

// Inside the existing hot-swap useEffect:
if (pluginConfig?.progress) {
  if (!progressAddonRef.current) {
    const pa = new ProgressAddon()
    term.loadAddon(pa)
    progressAddonRef.current = pa
    progressOnChangeDisposable.current = pa.onChange((state) => {
      onProgressChange?.(sessionId, state)
    })
  }
} else if (progressAddonRef.current) {
  progressOnChangeDisposable.current?.dispose()
  progressOnChangeDisposable.current = null
  progressAddonRef.current.dispose()
  progressAddonRef.current = null
  onProgressChange?.(sessionId, { state: 0, value: 0 })
}
```

### Web parity construction (web/assets/terminal.js)

```javascript
// Source: VERIFIED against /Users/ken/dev/agenthub/web/assets/terminal.js Phase 97 SER-01 lines 259-272
// UMD global shape: window.ProgressAddon is a namespace object with .ProgressAddon class
// (same pattern as ImageAddon.ImageAddon and SerializeAddon.SerializeAddon).

if (pluginConfig.progress) {
  try {
    var progressAddon = new ProgressAddon.ProgressAddon();
    term.loadAddon(progressAddon);
    progressAddon.onChange(function (state) {
      // For web v3.2: render an underline on the page header (no tab strip).
      var bar = document.getElementById('progress-underline');
      if (!bar) return;
      if (state.state === 1) {
        bar.style.transform = 'scaleX(' + (state.value / 100) + ')';
      } else {
        bar.style.transform = 'scaleX(0)';
      }
    });
  } catch (e) { /* addon UMD may not be present — silent */ }
}
```

### OSC 9;4 fixture for testing

```bash
# Source: VERIFIED against /tmp/progress-inspect/package/src/ProgressAddon.ts:50 OSC handler
# format. ESC ] 9 ; 4 ; <state> ; <value> BEL

# Set progress to 47%
printf '\x1b]9;4;1;47\x07'

# Move to 75%
printf '\x1b]9;4;1;75\x07'

# Set indeterminate (deferred to v3.3 in scope but valid sequence to test parsing)
printf '\x1b]9;4;3;0\x07'

# Clear
printf '\x1b]9;4;0\x07'
```

### Wails RPC handler (Go)

```go
// Source: handcrafted; mirrors Phase 97 (*App).SaveTerminalSession at app.go pattern.
// File: app.go

// SetTrayProgress is called by App.tsx with a quartile [0..4] derived from the
// cross-session aggregate. Quartile 0 = no progress active → revert to base icon.
// The throttle/debounce lives in App.tsx (200ms); this method is idempotent —
// calling it with the same quartile twice is a no-op (Pitfall #3 transition guard).
func (a *App) SetTrayProgress(quartile int) error {
    if !a.trayInit {
        return nil
    }
    if quartile < 0 || quartile > 4 {
        return fmt.Errorf("SetTrayProgress: quartile out of range [0,4]: %d", quartile)
    }
    if a.lastTrayQuartile == quartile {
        return nil
    }
    a.lastTrayQuartile = quartile
    a.refreshTrayState()
    return nil
}
```

---

## Cuttable Inside Cuttable: Minimum-Viable Scope Path

The phase is P2 cuttable at the milestone level. **Within the phase**, also identify a minimum-viable inner scope so the planner can drop work if Phase 95/96 leak into Phase 98's window:

| Work item | v3.2 in scope | v3.3 deferred |
|-----------|---------------|---------------|
| OSC 9;4 parsing via `@xterm/addon-progress` | YES | — |
| `state:1` (set value) underline on tab | YES | — |
| `state:0` (remove) hides underline | YES | — |
| `state:2` (error) UX | NO | YES |
| `state:3` (indeterminate) animated shimmer | NO | YES |
| `state:4` (paused) muted underline | NO | YES |
| Cross-session mean aggregation | YES | — |
| Tray icon quartile glyph swap | YES | — |
| 200ms debounce of tray-API calls | YES | — |
| 4 PNG quartile glyph assets | YES | — |
| Web parity: vendored UMD copy | YES | — |
| Web parity: per-page underline | YES (recommended) | DEFERRABLE if scope tight |
| Default-OFF in v3.2 (PRG-01) | YES (already wired Phase 92) | — |
| Italic v3.3-flip caption under toggle | YES | — |
| Negative-OFF regression test | YES | — |
| Playwright e2e for OSC 9;4 → underline | YES (recommended) | DEFERRABLE if scope tight |
| Stuck-progress cleanup on session unmount | YES | — |
| OS-level dock badge / taskbar progress | NO | NO (PRG-FUT-OS-CHROME) |
| Median aggregation option | NO | NO (lock mean) |
| Configurable underline color | NO | NO |
| Per-plugin theme tokens | NO | NO |

**Recommended drop order if Phase 95/96 over-runs eat Phase 98 time:**

1. Drop Playwright e2e test (defer to Phase 99 or v3.3 — manual UAT covers it)
2. Drop web-side underline (vendored UMD copy stays for vendor_drift_test; web client just embeds the script, doesn't construct addon)
3. Drop tray quartile glyph (PRG-03) — ship PRG-01 + PRG-02 (toggle + per-tab underline) only
4. **DO NOT DROP PRG-01 + PRG-02.** If both must drop, the phase cuts entirely (the canonical "cuttable" exit).

The plan should structure Waves so that the drop order above corresponds to "cancel the last wave" — i.e. earliest waves ship the highest-value work, last wave ships the e2e/cosmetic polish.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Custom OSC 9;4 parser per app | `@xterm/addon-progress` 0.2.0 (xterm.js first-party) | xterm.js 5.5+ (~2024) | Drop the custom parser; addon handles all 5 states + clamping |
| Per-tab progress as text label "47%" | Per-tab visual underline with CSS transform | iTerm2 introduced OSC 9;4 with visual mode (~2018); xterm.js followed | Less screen real estate; matches iTerm2/WezTerm/Windows Terminal vocabulary |
| System tray icon updates per-event | Throttled/debounced quartile glyph swap | macOS NSStatusItem rate guidance (~2010s) | Avoids platform-tray-API churn; smoother UX |

**Deprecated/outdated:**
- iTerm2's pre-OSC 9;4 progress sequence (DCS-style) — replaced by ConEmu's OSC 9;4 which xterm.js implements (this addon).
- xterm.js < 5.5 didn't ship a progress addon — apps had to write their own; deprecated by `@xterm/addon-progress`.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Frontend framework | vitest (existing — Phase 92+) |
| Backend framework | go test (existing — `internal/release/`, `internal/daemon/`, `internal/webserver/`) |
| E2E framework | Playwright (existing — Phase 95/96/97) OR chromedp (existing — Phase 89/94) — pick one consistent with Wave 4 |
| Quick run command | `cd frontend && pnpm vitest run --reporter=basic` (frontend) + `go test ./internal/...` (backend) |
| Full suite command | `cd frontend && pnpm test && go test ./... && cd frontend && pnpm e2e` |
| Phase gate | All three above green; `wails build -tags wailsassets` succeeds; manual UAT smoke (toggle ON in Settings, run OSC 9;4 fixture, observe underline + tray glyph) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PRG-01 | Progress toggle persists default OFF; flips state via Settings save | unit (frontend) | `pnpm vitest run components/__tests__/PluginsSection.test.tsx -t progress` | ❌ Wave 0 (extend existing test) |
| PRG-01 | Italic v3.3-flip caption present under Progress toggle | unit (frontend) | `pnpm vitest run components/__tests__/PluginsSection.test.tsx -t v3.3-flip` | ❌ Wave 0 |
| PRG-01 | Toggle OFF leaves zero progress addon construction (negative path) | integration (backend grep) | `go test ./internal/release -run TestPRG_OffPath_NoProgressLogic` | ❌ Wave 0 |
| PRG-02 | ProgressAddon attaches when pluginConfig.progress flips ON; detaches on flip OFF | unit (frontend) | `pnpm vitest run components/__tests__/TerminalPanel.test.tsx -t progress-hot-swap` | ❌ Wave 0 |
| PRG-02 | onChange events forwarded via onProgressChange callback prop | unit (frontend) | `pnpm vitest run components/__tests__/TerminalPanel.test.tsx -t progress-onchange-forward` | ❌ Wave 0 |
| PRG-02 | TabBar renders underline at correct width based on tabProgress | unit (frontend) | `pnpm vitest run components/__tests__/TabBar.test.tsx -t progress-underline` | ❌ Wave 0 |
| PRG-02 | Underline uses transform (not width) for animation | unit (frontend; CSS or DOM check) | `pnpm vitest run components/__tests__/TabBar.test.tsx -t progress-transform` | ❌ Wave 0 |
| PRG-02 | OSC 9;4 sequence drives addon → underline (e2e) | e2e | `pnpm playwright test tests-e2e/progress.spec.ts` (or `go test -tags=e2e ./internal/...`) | ❌ Wave 4 |
| PRG-03 | aggregateProgress helper buckets correctly | unit (frontend) | `pnpm vitest run lib/__tests__/aggregateProgress.test.ts` | ❌ Wave 0 |
| PRG-03 | Empty registry → quartile 0 (revert to base) | unit (frontend) | (above) | (above) |
| PRG-03 | All-state-0 registry → quartile 0 | unit (frontend) | (above) | (above) |
| PRG-03 | Mean of [50, 75] = 62.5 → quartile 3 | unit (frontend) | (above) | (above) |
| PRG-03 | SetTrayProgress idempotency (no-op on identical quartile) | unit (backend) | `go test ./. -run TestApp_SetTrayProgress_Idempotent` | ❌ Wave 1 |
| PRG-03 | SetTrayProgress quartile bounds check | unit (backend) | `go test ./. -run TestApp_SetTrayProgress_BoundsCheck` | ❌ Wave 1 |
| PRG-03 | SetTrayProgress error precedence (disconnected → error icon, ignore quartile) | unit (backend) | `go test ./. -run TestApp_SetTrayProgress_ErrorPrecedence` | ❌ Wave 1 |
| PRG-03 | Frontend 200ms debounce drops within-window bursts | unit (frontend) | `pnpm vitest run lib/__tests__/debounce.test.ts` (or App-level integration) | ❌ Wave 0 |
| PRG-03 | Manual UAT: open 3 terminals, emit progress in 2, observe tray glyph 50% quartile | manual | (manual UAT) | ❌ Wave 4 |

### Sampling Rate

- **Per task commit:** `cd frontend && pnpm vitest run --reporter=basic` + `go test ./...` (skip e2e for speed)
- **Per wave merge:** Above + `pnpm playwright test` (or chromedp e2e) for the wave's covered features
- **Phase gate:** Full suite green, including manual UAT for PRG-03 cross-session aggregation + tray glyph
- **Cuttability gate:** Verify that toggling Progress OFF in Settings (already default) produces a binary build identical-to-Phase-97-state in all behavioral surfaces — i.e. the OFF path doesn't load the addon, doesn't subscribe to onChange, doesn't call SetTrayProgress, doesn't render `.tab__progress`.

### Wave 0 Gaps

- [ ] `frontend/src/lib/aggregateProgress.ts` — pure helper for cross-session mean → quartile bucketing
- [ ] `frontend/src/lib/__tests__/aggregateProgress.test.ts` — vitest cases (empty / single / multi / all-state-0 / boundary values)
- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — extend with progress-hot-swap + onchange-forward cases
- [ ] `frontend/src/components/__tests__/TabBar.test.tsx` — extend with progress-underline render case
- [ ] `frontend/src/components/__tests__/PluginsSection.test.tsx` — extend with v3.3-flip caption case
- [ ] `internal/release/no_progress_when_off_test.go` — negative regression test (mirror Phase 97 SER-03 pattern)
- [ ] `frontend/tests-e2e/progress.spec.ts` — Playwright OSC 9;4 → underline e2e (Wave 4 / cuttable last)
- [ ] OSC 9;4 fixture script `tests/fixtures/osc94-progress-fixture.sh` for manual UAT (4-line shell script)
- [ ] 4 × 18×18 PNG quartile glyph assets at `assets/tray_icon_progress_{25,50,75,100}.png` — designer-supplied or generated; build-script optional

*(If no gaps: NOT applicable — gaps exist.)*

---

## Security Domain

> Required when `security_enforcement` is enabled. Including for completeness; Phase 98 is low-security-surface (the addon parses CLI-emitted output, doesn't touch network/auth/storage), but the audit MUST be documented.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | n/a |
| V3 Session Management | no | n/a |
| V4 Access Control | yes | Capability-token model already protects `/api/plugin-config` (Phase 93 WEB-03); Phase 98 introduces no new endpoints |
| V5 Input Validation | yes | OSC 9;4 sequence parsing — addon already validates state ∈ {0..4}, value clamped to [0, 100] (verified `package/src/ProgressAddon.ts:65-100`); SetTrayProgress quartile bounds check on Go side |
| V6 Cryptography | no | n/a |
| V12 File Resources | no | n/a (no file I/O introduced; tray PNGs are //go:embed compile-time) |
| V14 Configuration | yes | CSP unchanged (Phase 96 amendment carries forward); script-src 'self' continues to gate addon-progress.js loading from same-origin only |

### Known Threat Patterns for {Phase 98 stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious CLI floods OSC 9;4 to DoS the renderer | Denial-of-Service | Frontend debounce + Go idempotency gate; OSC handler runs on the xterm.js parser thread which is rate-limited by relay frame delivery (the daemon→relay→browser pipeline IS the throttle) |
| Untrusted CLI sends out-of-range progress values | Tampering | Addon clamps value to [0, 100] (verified line 99); state out-of-range ignored with console.warn (verified line 95) |
| Tray-API quartile manipulation by frontend bug | Tampering | Go-side bounds check `quartile < 0 || quartile > 4 → error` |
| Malicious vendored UMD copy (supply chain) | Tampering | Phase 89 vendor-drift test asserts pnpm-lock.yaml == VERSION; Phase 90 SHA-pinned dependabot for `@xterm/*` updates (carries forward) |
| Cross-session progress leak (one user's progress visible to another) | Information Disclosure | The progressRegistry lives in the local renderer's App.tsx state; the daemon never sees it; cross-client web-served sessions each get their own renderer state. No shared store. |
| Tray glyph reveals task progress to a user with screen-share active | Information Disclosure | User-controlled (default OFF); the toggle copy notes "OSC 9;4 progress" so security-conscious users can opt out |

---

## Open Questions

1. **Designer-supplied vs generated PNG quartile glyphs.**
   - What we know: 4 PNGs needed at 18×18 (matching existing tray_icon.png dims); TokyoNight accent palette established.
   - What's unclear: whether the designer is producing these or whether the plan should include a build-time generation step (e.g. Go script using `image/draw` to overlay a horizontal bar on the base PNG — `build/gen_icon.go` already exists and could be extended).
   - Recommendation: plan should ship a `build/gen_progress_icons.go` that generates the 4 PNGs from `assets/tray_icon.png` + a horizontal-fill overlay; commit the resulting PNGs to the repo (per the project's existing `embed`-able-asset pattern). Designer can override the overlay color/shape if they want; the generation script gets us unblocked.

2. **Whether a chromedp e2e test (Phase 89/94 pattern) or Playwright (Phase 95/96/97 pattern) is preferred for OSC 9;4 → underline.**
   - What we know: project has both; chromedp is already wired with `//go:build e2e` tags and runs in CI.
   - What's unclear: which the planner prefers for Phase 98's e2e Wave.
   - Recommendation: Playwright — recent precedent (Phase 95/96/97) leans Playwright; the test is "open the desktop app, emit OSC 9;4 via a fixture script, assert the .tab__progress element transforms." chromedp would force a CDP-only path that's harder to reuse for the manual-UAT runbook.

3. **Whether to include `state:3` (indeterminate) shimmer in v3.2 since it's the only sub-1% extra effort.**
   - What we know: v3.2 scope locked to `state:1` + `state:0` only.
   - What's unclear: whether the planner deems shimmer trivially achievable (a CSS keyframe + `animation: shimmer 2s linear infinite`).
   - Recommendation: stay locked. The risk is that shimmer + `prefers-reduced-motion` + theme variation expands the test surface, and Phase 98 is P2 cuttable. Defer.

4. **Whether the web-served terminal page should render the underline at all in v3.2.**
   - What we know: web client has no tab strip; PRG-02's intent is "visible progress affordance"; the underline at the top of the page is a defensible interpretation.
   - What's unclear: whether the user wants this or whether deferring web-side underline is fine.
   - Recommendation: ship the underline on the web page (it's a 5-line addition to `web/assets/terminal.js` + a single CSS rule + an HTML element in `web/terminal.html`). If Phase 95/96 over-runs, this is the first thing to drop (per the cuttability ladder above).

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `pnpm` | frontend addon install + lockfile | ✓ | (already in use) | — |
| `Wails v2.12.0` runtime API | SetTrayProgress RPC + cgo tray | ✓ | v2.12.0 | — |
| Go `image/draw` (stdlib) | optional generated quartile PNGs | ✓ | go1.x stdlib | hand-supplied PNGs |
| Playwright | e2e progress underline test | ✓ | (existing — Phase 95+) | chromedp (existing — Phase 89/94) |
| OSC 9;4-emitting CLI for UAT | manual UAT | depends on user env | — | shell `printf '\x1b]9;4;1;47\x07'` always works |
| Tailscale | web parity UAT | ✓ (existing — v3.1+) | — | local-network-fallback |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None — every dependency is satisfied or has a viable fallback.

---

## Assumptions Log

> The research above explicitly verified the major claims via direct source-read of the upstream tarball, the project codebase, and the npm registry. The remaining `[ASSUMED]` items are stated below; they should be confirmed during planning if any are load-bearing.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 200ms debounce interval is sufficient to prevent perceptible tray-API churn. | Pitfall #5; Locked Decisions | Could increase to 500ms if real CLIs emit at >5 Hz; trivial to tune. |
| A2 | Mean-based aggregation is the user-intuitive choice over median. | Locked Decisions; Pattern 3 | Median is also defensible; switching is a one-line change in aggregateProgress.ts. |
| A3 | Bottom horizontal fill bar is the optimal tray glyph design at 18×18. | Claude's Discretion | Designer review may prefer a different shape; Go side is design-agnostic (it just consumes 4 PNG byte slices). |
| A4 | Web-side underline at the top of the terminal page is a sufficient PRG-02 affordance. | Pitfall #10; Open Questions #4 | If users find it unobtrusive, can move to a more prominent placement; only `web/terminal.html` + `web/assets/terminal.js` need to change. |
| A5 | `state:2/3/4` UX deferral to v3.3 is acceptable to the user. | Locked Decisions; Cuttable Inside Cuttable | If v3.2 needs full state coverage, scope expands; P2 framing should hold. |
| A6 | TerminalPanel unmount is the correct boundary for stuck-progress cleanup. | Pitfall #7; Claude's Discretion | If sessions can outlive panels (web reconnect?), may need timer-based cleanup; web client has page-reload as the natural boundary already. |

**If this table is empty:** N/A — six assumptions listed; planner should validate A1, A2, A4 with the user if uncertain.

---

## Sources

### Primary (HIGH confidence)
- `@xterm/addon-progress@0.2.0` typings — `/tmp/progress-inspect/package/typings/addon-progress.d.ts` (verified 2026-05-08)
- `@xterm/addon-progress@0.2.0` source — `/tmp/progress-inspect/package/src/ProgressAddon.ts` (verified 2026-05-08)
- `@xterm/addon-progress@0.2.0` UMD bundle — `/tmp/progress-inspect/package/lib/addon-progress.js` (verified 2026-05-08)
- npm registry — `pnpm view @xterm/addon-progress` (verified 2026-05-08)
- AgentHub `internal/daemon/plugin_settings.go` lines 72-113 — PluginSettings struct + Progress field default
- AgentHub `frontend/src/components/PluginsSection.tsx` lines 143-144 — existing Progress toggle row
- AgentHub `frontend/src/components/TabBar.tsx` lines 105-191 — existing tab strip + context menu structure
- AgentHub `frontend/src/components/TerminalPanel.tsx` — existing hot-swap useEffect pattern (Phase 93/97)
- AgentHub `frontend/src/style.css` lines 77-128, 881-898 — tab-bar + tab__status styling patterns
- AgentHub `app.go` lines 921-960 — startTrayPoller + refreshTrayState + updateTray contract
- AgentHub `tray.go` (darwin), `tray_windows.go`, `tray_linux.go` — cross-platform tray API integration
- AgentHub `web/embed.go` — embed.FS directives for vendored xterm assets
- AgentHub `web/terminal.html` lines 43-51 — vendored addon script-tag include order
- AgentHub `web/assets/terminal.js` lines 240-272 — Phase 96/97 web-parity addon construction patterns
- AgentHub `internal/webserver/vendor_drift_test.go` — Phase 93 generalized version-parity gate
- Phase 97 RESEARCH.md — addon-serialize precedent (vendoring, hot-swap, secrets caption, web parity scope, negative regression test patterns)

### Secondary (MEDIUM confidence)
- `xtermjs/xterm.js` GitHub repository — addon-progress was added in xterm.js 5.5+ (2024)
- iTerm2 Proprietary Escape Codes documentation — OSC 9;4 ConEmu progress sequence origin

### Tertiary (LOW confidence)
- (none — every load-bearing claim was verified against primary sources)

---

## Metadata

**Confidence breakdown:**
- Standard stack (addon-progress version, license, deps): HIGH — verified via npm registry + direct tarball read
- Architecture (hot-swap arm, registry pattern, tray quartile swap): HIGH — Phase 93/97 precedent verified in source
- CSP audit: HIGH — direct grep of upstream tarball found zero CSP-relevant constructs
- Pitfalls: HIGH (mostly) — 200ms debounce / mean aggregation / quartile bucketing are reasoned defaults backed by project precedent (Phase 94 BannerStack 200ms convention)
- API contract: HIGH — typings + source-read of all 102 lines of ProgressAddon.ts
- Cross-platform tray API: HIGH — direct read of tray.go / tray_windows.go / tray_linux.go
- Web parity scope: MEDIUM — Phase 97 precedent supports "vendor-and-light-construction" but the underline placement on a tab-strip-less page is a discretionary call (Open Question #4)
- Aggregation function (mean vs median): MEDIUM — both are defensible; mean is locked but planner may revisit

**Research date:** 2026-05-08
**Valid until:** 2026-06-07 (30 days for a stable upstream addon at v0.2.0; bump if `pnpm view @xterm/addon-progress` shows a new latest release before Phase 98 Wave 0 starts)
