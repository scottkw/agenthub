# Pitfalls Research

**Domain:** Go/Wails desktop app — UI/UX polish (terminal padding, terminal theming, web server link actions, sidebar icon centering)
**Researched:** 2026-04-10
**Confidence:** HIGH (codebase fully readable; xterm.js issues verified against GitHub tracker; Wails runtime verified against official docs)

---

## Critical Pitfalls

### Pitfall 1: Terminal Padding Breaks FitAddon Column Count

**What goes wrong:**
The existing `fitTerminal()` custom function reads `term.element` padding via `getComputedStyle` and subtracts it from available width/height before calculating cols/rows. This means CSS padding applied to the `.xterm` element is already accounted for in the custom fitter. However, padding applied to the *container* (`containerRef.current`) rather than the `.xterm` element is NOT accounted for — `fitTerminal` reads the parent's width from `getComputedStyle(parent).width`, which is the full container width including its own padding.

The result: the terminal overflows the container by exactly the container's padding value, or — if the terminal clips — the right edge of text is cut off. Both regressions are invisible in unit tests since jsdom reports zero for all layout measurements.

There is a secondary failure: the project already uses a custom `fitTerminal()` that bypasses `FitAddon.fit()` precisely because `FitAddon.fit()` hardcodes `DEFAULT_SCROLL_BAR_WIDTH = 14px` even when the scrollbar is hidden via CSS. Adding `options.scrollback: 0` or `overflow: hidden` after the fact to hide the scrollbar will interact with this workaround in an unpredictable way.

**Why it happens:**
Padding is typically added to the inner `.xterm` element (the recommended approach) but developers may add it to the React container div for convenience. The custom `fitTerminal()` reads the parent's `width` from computed style — which does NOT subtract the parent's own padding. The inner `.xterm` padding IS correctly subtracted. So: container padding → overflow; `.xterm` padding → correct.

The existing comment in `TerminalPanel.tsx` ("FitAddon.fit() always subtracts DEFAULT_SCROLL_BAR_WIDTH (14px) even when the scrollbar is hidden") explains why the custom fitter exists — but a developer unfamiliar with this can accidentally revert to `FitAddon.fit()` when adding padding, restoring the scrollbar gap regression.

**How to avoid:**
- Apply padding only via `xterm.js` `options.scrollback` and `options.padding` — NOT via CSS on the container div.
- `xterm.js` exposes `terminal.options` as a live settable object: set `term.options` (or `term.setOption` in older API) with the padding value. The internal `FitAddon` source (`FitAddon.ts`) reads `elementStyle.paddingLeft/Right/Top/Bottom` and subtracts them before dividing by cell dimensions — so padding set this way is handled correctly.
- Do NOT add CSS `padding` to `containerRef.current` (the React div). It must remain a zero-padding flex container so the parent's width measurement is accurate.
- After changing `term.options.fontSize` (or any dimension-affecting option), call `fitTerminal(term)` immediately — the existing `useEffect([fontSize])` already does this; verify padding changes follow the same pattern.
- The `ResizeObserver` on the container will fire on any layout change including sidebar expand/collapse — confirm it fires and re-fits correctly after the container's parent width changes.

**Warning signs:**
- Terminal text is clipped on the right side (terminal overflows into container padding area).
- One or two blank columns appear on the right (FitAddon reverting to subtracting scrollbar width).
- `term.cols` differs from `Math.floor((containerWidth - 2*padding) / cellWidth)` by more than 1.
- Sidebar expand/collapse causes terminal to not fill the new width until a window resize occurs.

**Phase to address:** Terminal padding phase. The padding value must be applied via `terminal.options`, not CSS, and the existing `fitTerminal` function must remain in use (not replaced by `FitAddon.fit()`).

---

### Pitfall 2: Font Change Requires Re-Fit and Triggers a Timing Gap

**What goes wrong:**
When a user selects a new terminal font (e.g., switching from "Cascadia Code" to "JetBrains Mono"), three sequential steps must happen in the correct order:

1. `term.options.fontFamily = 'JetBrains Mono'` — triggers xterm to invalidate its character size cache and rebuild the texture atlas.
2. The CharSizeService inside xterm.js must remeasure the new font's cell dimensions. This is asynchronous — the service measures by rendering a hidden span in the DOM and reading its metrics. Until this completes, `dims.css.cell.width` and `dims.css.cell.height` return the old values.
3. Only after CharSizeService has updated should `fitTerminal()` be called — otherwise it calculates cols/rows using the old font's cell dimensions, producing incorrect terminal dimensions with the new font.

The existing rAF retry loop (which polls `fitAddonRef.current?.proposeDimensions()`) was built for a specific scenario: the terminal is hidden via `display:none` when opened, so CharSizeService returns zero until the panel becomes visible. The font-change scenario is different: CharSizeService returns *non-zero* but *stale* values. The retry loop exits immediately on seeing non-zero, before the new font dimensions are available.

A separate font-loading failure: web fonts (non-system fonts) may not be loaded by the time `term.options.fontFamily` is set. xterm.js uses canvas rendering and does not wait for `document.fonts.ready`. If the font is referenced but not yet downloaded, xterm renders with the fallback monospace font but reports dimensions for that fallback — so all content renders at the wrong width.

**Why it happens:**
Web font loading is asynchronous and xterm.js provides no built-in callback for "font ready." Canvas rendering does not automatically trigger a re-render when a web font becomes available (unlike DOM text nodes which benefit from font-swap). Developers test with system fonts (Courier New, monospace) during development and never encounter the race condition because system fonts are always already loaded.

**How to avoid:**
- For web font loading: use `document.fonts.load('14px "JetBrains Mono"')` (returns a Promise) before setting `term.options.fontFamily`. Only after the promise resolves should `fontFamily` be updated and `fitTerminal()` called.
- For post-change re-fit timing: after setting `term.options.fontFamily`, schedule `fitTerminal()` in a `requestAnimationFrame` callback, not synchronously. This gives xterm one frame to remeasure after the font option change. If using a web font, wait for `document.fonts.load()` resolution, THEN schedule the rAF.
- Persist the chosen font name in `localStorage` (e.g., `terminal-font-family`). On terminal creation in `useEffect([sessionId])`, read from localStorage and set the initial `fontFamily` in the `Terminal` constructor — this fires before `term.open()`, so CharSizeService measures the correct font from the start.
- When applying a saved font at startup, still call `document.fonts.load()` first if the font is a web font, since the terminal opens immediately on app launch before web fonts may be cached.
- The `FontFace` API and `@font-face` CSS declarations must be included in the app's CSS or dynamically injected before `document.fonts.load()` will resolve.

**Warning signs:**
- Columns are correct for the old font, wrong for the new font — text wraps too early or extends past the terminal edge.
- `fitTerminal()` fires with the correct new font but the wrong cell width.
- Font appears correct visually but the PTY still renders to the old column count — PTY dimensions were set before the re-fit.
- On app restart, the selected font works correctly (because it was set in the Terminal constructor), but switching fonts mid-session produces misaligned output.

**Phase to address:** Terminal theming phase. Font family changes require the `document.fonts.load()` pre-flight and a rAF-deferred re-fit. This is additive to the per-tab font size logic, not a replacement.

---

### Pitfall 3: Theme Persistence Requires Separate Storage from Font Sizing

**What goes wrong:**
The existing per-tab font size uses a `fontSizes` map keyed by session ID stored in React state (`App.tsx`). Terminal themes are not per-tab settings — they are global app preferences (all terminal tabs use the same color theme). Storing the selected theme in the same `fontSizes` pattern (per-tab, in React state) creates divergent theme state across tabs and means the theme is lost on app restart because React state does not persist.

A related failure: `xterm.js` v5 renamed `ITheme.selection` to `ITheme.selectionBackground`. Theme definitions copied from online resources or older xterm.js examples may use the old key name — the property is silently ignored (no runtime error), and the selection color shows as transparent, which is visually broken but not obviously wrong.

A third failure: applying a new theme to an already-open terminal requires `term.options.theme = newTheme` (full theme object assignment) — not `term.options.theme.background = '#...'`. Partial assignment of individual color properties does not trigger xterm's internal color cache invalidation, so the change may not appear until the next terminal clear.

**Why it happens:**
Developers conflate per-session-sizing (appropriate per-tab) with per-session-theming (not appropriate — all tabs should match). The `fontSizes` map is convenient but wrong for global prefs. ITheme key renames are a common xterm.js API version gotcha.

**How to avoid:**
- Store the selected theme name (not the full theme object) in `localStorage` under a single key (`terminal-theme`). All `TerminalPanel` instances read from this on mount and apply the theme in the `Terminal` constructor.
- When the user changes the theme, update `localStorage` and then iterate over all open `termRef.current` instances to apply the new theme. This requires either a React context or an event dispatch pattern — avoid storing theme in `App.tsx` state if it would cause all tabs to remount.
- Use `term.options.theme = { ...newTheme }` (new object) not property mutation. This guarantees the setter fires and xterm rebuilds its color cache.
- Validate theme objects against the xterm.js v5 `ITheme` interface: required key is `selectionBackground` (not `selection`). A compile-time TypeScript `ITheme` import from `@xterm/xterm` will catch this at build time.
- Ship a small number of curated themes (5-8) as static TypeScript objects conforming to `ITheme` rather than parsing external theme files at runtime. This avoids format compatibility issues.

**Warning signs:**
- Text selection color is transparent or invisible — indicates `selection` (old API) being used instead of `selectionBackground`.
- Theme change works on the active tab but other tabs still show the old theme.
- Theme is lost after app restart (not persisted to localStorage).
- `term.options.theme.background = '#...'` mutations have no visual effect.

**Phase to address:** Terminal theming phase. Theme storage must be global (`localStorage`), not per-tab React state.

---

### Pitfall 4: Web Server Link Opening Silently Fails Without Error Feedback

**What goes wrong:**
`runtime.BrowserOpenURL(ctx, url)` calls `github.com/pkg/browser.OpenURL` internally. This function can fail (no default browser configured, sandboxed environment, missing `xdg-open` on Linux) but `BrowserOpenURL` does not return an error and does not log the failure. The Go function signature is `BrowserOpenURL(ctx, url string)` with no return value — the error is swallowed.

On Linux, `xdg-open` may not be installed in minimal environments. On macOS inside a sandboxed app bundle (not the case for Wails apps, but worth noting), `NSWorkspace.openURL` can be restricted. The frontend has no way to know the URL failed to open.

A second failure: the Wails JS binding `BrowserOpenURL(url)` is available on the frontend as `window.go.main.App.BrowserOpenURL(url)`. Calling it on the URL string obtained from the web server may produce a call with an empty string if the web server URL is not yet populated in state — the browser silently opens a blank tab or nothing happens with no error.

**Why it happens:**
`BrowserOpenURL` was designed as a fire-and-forget convenience. Its silent failure mode is documented in the Wails GitHub issue tracker but not prominently in the official docs. Developers test on macOS (where `open` always works) and never encounter the silent failure on Linux.

**How to avoid:**
- Wrap the Wails binding call in the frontend with a guard: check that the URL string is non-empty before calling `BrowserOpenURL`. Show a toast or inline error if the URL is empty.
- On the Go side, expose a `OpenWebServerURL() error` method on `App` that calls `BrowserOpenURL` and logs the result (even though `BrowserOpenURL` swallows the error). Use `exec.Command("open", url)` on macOS directly if silent failure is observed in production — this at least surfaces the stderr.
- For the copy-to-clipboard path, use `runtime.ClipboardSetText(ctx, url)` which returns `(bool, error)` in Go and a Promise in JS. Surface failure to the user: "Could not copy to clipboard" with the URL displayed as plain text so they can copy manually.
- Do not use the clipboard path as a fallback for failed browser opening — they are independent actions triggered by separate buttons.

**Warning signs:**
- Clicking "Open in Browser" has no visible effect on Linux.
- No error is shown in the frontend when `BrowserOpenURL` fails.
- The URL displayed in the StatusBar or Settings is empty when the web server hasn't started yet, and clicking "Open" with an empty URL triggers a call that silently does nothing.
- `ClipboardSetText` returns `false` on macOS M1 (known issue in Wails v2.4.1 — verify the Wails version in use).

**Phase to address:** Web server link actions phase. Guards on empty URL and error surfacing for clipboard must be part of the initial implementation.

---

### Pitfall 5: Clipboard in Wails Context — Platform Inconsistency

**What goes wrong:**
`runtime.ClipboardSetText` has documented failures on macOS M1/M2 (Wails v2.4.1) where the function returns success but the clipboard remains empty. The root cause was a missing `LANG` environment variable for the underlying `pbcopy`/`pbpaste` subprocess — this was fixed in a subsequent Wails release, but the exact version where the fix landed is not consistently documented.

On Linux under Wayland (as opposed to XWayland), clipboard operations require `wl-clipboard` (`wl-copy`) to be installed. Wails on Linux defaults to GTK/X11 via XWayland when `GDK_BACKEND` is not set — but if the user is running a pure Wayland compositor and `GDK_BACKEND=wayland` is set, clipboard operations may fail or produce garbled text.

The JS-side clipboard API (`navigator.clipboard.writeText`) is an alternative but requires the page to have focus — in Wails, the WebView typically does have focus, so this may work. However, Wails' security model may restrict `navigator.clipboard` access depending on how `AllowedHosts` is configured.

**Why it happens:**
Clipboard is a system-level operation that Wails delegates to platform-native utilities. Each platform has different prerequisites, and Wails abstracts over them imperfectly. Developers test on macOS, miss the Wayland/Linux case entirely.

**How to avoid:**
- Use `runtime.ClipboardSetText` on the Go side (not `navigator.clipboard` on the JS side) so the Wails runtime handles platform differences.
- Always show the URL as selectable plain text alongside the "Copy" button, as a fallback for clipboard failure.
- After calling ClipboardSetText, show a visual confirmation ("Copied!") on the button — change button text or icon for 2 seconds. This tells the user the operation was attempted even if they need to verify the clipboard manually.
- Test clipboard on the Linux build in CI (even a basic `xclip` availability check) — do not assume macOS testing is sufficient.
- Avoid using `navigator.clipboard.writeText` as the primary mechanism; it is less reliable in Wails WebView than `runtime.ClipboardSetText`.

**Warning signs:**
- "Copied!" toast shows but paste in another app yields nothing (macOS M1 LANG bug — check Wails version).
- Clipboard copy works in GUI but not in test environment (headless Linux without X server).
- Clipboard behavior differs between macOS and Linux builds during manual QA.

**Phase to address:** Web server link actions phase. Clipboard failure UX (showing the URL as plain text) should be the default design, not an afterthought.

---

### Pitfall 6: QR Code for Dashboard URL — URL Staleness and State Coupling

**What goes wrong:**
The project already uses `skip2/go-qrcode` on the Go side for per-session QR codes. The v1.12 feature adds a QR code for the web server *dashboard* URL (not per-session). The dashboard URL is the same regardless of Tailscale vs. local mode — but the host, port, and scheme differ between the two modes.

A common failure: the QR code component is rendered with a URL captured at mount time. If the web server restarts (mode switch, Tailscale connects mid-session), the QR code encodes the stale URL. The user scans it, connects to the old address, and gets a connection refused error.

A second failure: placing the QR code rendering in the frontend (React-side, using a library like `react-qr-code` or `qrcode.react`) means the URL must be passed as a prop from the parent. If the parent's web server URL state is populated by a polling Wails binding (`GetWebServerURL` called periodically), there is a window between state update and QR render where the old QR is displayed alongside the new URL text.

**Why it happens:**
State updates in React are batched. The URL text and the QR code image derive from the same state variable, so they should update together — but if the QR component reads from a stale prop reference or uses `useMemo` with incorrect dependencies, they can diverge.

**How to avoid:**
- Derive the QR code URL from the same state variable as the URL text displayed alongside it. Do not store the QR URL separately.
- If generating the QR image on the Go side (using `skip2/go-qrcode`), return the base64-encoded PNG alongside the URL from the same API call so they are atomically consistent.
- If generating the QR image in the frontend, use `qrcode.react` or `react-qr-code` with the URL string as the `value` prop — React's reconciliation ensures the QR updates when the URL state updates.
- Do not use `useMemo` to cache the QR code unless the URL string is a dependency of the memo.
- When the web server is not running, do not render the QR code at all — render a disabled/greyed state instead.

**Warning signs:**
- URL text shows the new address but QR encodes the old address after a mode switch.
- QR code is rendered for an empty URL (scanning it yields an empty string or invalid URL).
- QR code does not update when Tailscale connects and the web server URL changes from LAN IP to FQDN.

**Phase to address:** Web server link actions phase. QR code must derive from the same state as URL text.

---

### Pitfall 7: Sidebar Icon Centering — Collapsed State Misalignment

**What goes wrong:**
The current `sidebar__item` CSS uses `display: flex; align-items: center; gap: 8px; padding: 8px`. When collapsed (sidebar width 48px), the label span is conditionally removed via `{!collapsed && <span>}`. The icon (20px) plus left padding (8px) = 28px. The remaining 20px (48 - 28) goes to the right side — the icon is not centered.

To center the icon, the item needs `justify-content: center` when collapsed. But `justify-content: center` in expanded state would center the icon+label group, not left-align the text as currently designed.

Applying `justify-content: center` globally breaks the expanded state. Using a CSS class modifier (`.sidebar--collapsed .sidebar__item { justify-content: center; }`) is the correct approach, but developers often miss that `padding` also affects centering: with `padding: 8px` on both sides and `justify-content: center`, the effective centering is correct (8px left + 20px icon + 8px right = 36px, centered in 48px). However if the icon has `flex-shrink: 0` (it does) and there is no `justify-content: center`, the icon will left-align in the 32px remaining after left padding.

A second failure: the sidebar toggle button (`sidebar__toggle`) is styled separately from `sidebar__item` and is already correctly centered (`display: flex; align-items: center; justify-content: center; width: 38px; height: 38px; margin: 4px`). It looks correct while the nav items look off — creating an inconsistent appearance that confuses users about the intended design.

The `sidebar__bottom` uses `margin-top: auto` to pin Settings to the bottom. When collapsed, the Settings icon must also be centered. If `justify-content` is only applied to `.sidebar__item` via a collapsed class modifier but not propagated into `.sidebar__bottom .sidebar__item`, Settings may remain misaligned.

**Why it happens:**
The `justify-content` vs `align-items` distinction is frequently confused (`align-items` centers on the cross axis, `justify-content` centers on the main axis). For a `flex-direction: row` item, centering horizontally requires `justify-content: center`, not `align-items: center`. The sidebar toggle is centered because it has explicit `justify-content: center` — the nav items lack it.

**How to avoid:**
- Add `.sidebar--collapsed .sidebar__item { justify-content: center; }` to the CSS. This targets only the collapsed state.
- Remove or set `gap` to 0 in collapsed state: `.sidebar--collapsed .sidebar__item { gap: 0; }` — the gap is irrelevant when no label is present but 8px gap on a 20px icon in 48px sidebar adds 8px to the right of the icon (since the label flexes to zero), visually off-centering it. Setting `gap: 0` when collapsed ensures the calculation is `(48px - 20px) / 2 = 14px each side`, but only if padding is also 0 or equal on both sides.
- Verify that `padding: 8px` on `sidebar__item` already provides equal horizontal padding (8px left + 20px icon + 8px right = 36px, inside a 48px sidebar leaves 6px unaccounted for). The correct fix is `justify-content: center` so the flex engine centers the icon within the remaining space after padding — `(48px - 16px padding - 20px icon) / 2 = 6px extra` distributed equally.
- The cleanest solution: when collapsed, set `justify-content: center; padding-left: 0; padding-right: 0` on sidebar items, and let the 20px icon center inside 48px. The toggle button already demonstrates this pattern.
- Write a Sidebar test (extending `Sidebar.test.tsx`) that verifies the collapsed class is applied and the icon is the only child rendered — not centering itself (jsdom can't measure layout), but at minimum verifying no label is in the DOM.

**Warning signs:**
- Collapsed sidebar shows icons slightly left-of-center (gap between icon right edge and sidebar right edge smaller than gap between icon left edge and sidebar left edge).
- Toggle button is centered but nav items are not — inconsistent alignment within the sidebar.
- Settings icon in `sidebar__bottom` is misaligned even after fixing nav items (forgot to propagate the fix to the bottom section).
- Animated transition from expanded to collapsed shows icon jumping left then staying left instead of smoothly centering.

**Phase to address:** Sidebar icon centering phase. Pure CSS change — zero backend changes, minimal frontend code change, but must cover all sidebar item selectors including `sidebar__bottom`.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Apply terminal padding via CSS on container div | Trivial one-line CSS change | `fitTerminal()` calculates wrong cols — text overflows or is clipped | Never — use `terminal.options` only |
| Revert to `FitAddon.fit()` when adding padding | Removes custom fitter complexity | Restores the scrollbar gap regression (14px permanent right gap) | Never — custom fitter must stay |
| Store selected theme in per-tab React state | Mirrors existing `fontSizes` pattern | Theme diverges across tabs; lost on app restart | Never — theme is global, not per-tab |
| Set `term.options.theme.background = '#...'` (mutation) | Simple one-liner | Does not trigger xterm color cache invalidation; change may not appear | Never — always assign a new theme object |
| Skip `document.fonts.load()` for web fonts | Simpler font change code | Font renders at wrong cell width until page reload | Never for web fonts; OK for system fonts (always available) |
| Generate QR code once at modal open | Simpler lifecycle | QR encodes stale URL if web server restarts between modal open and scan | Acceptable if modal is torn down and recreated on each open (not kept in DOM) |
| `BrowserOpenURL` without empty-string guard | One fewer line of code | Silent no-op if URL is empty | Never — always guard against empty URL |
| Show only "Copy" button without displaying URL text | Cleaner UI | No fallback when clipboard fails | Never — always show URL as selectable text |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| xterm.js padding | Add CSS `padding` to container div | Set via `terminal.options` so FitAddon/custom fitter reads padding from the `.xterm` element |
| xterm.js padding + custom `fitTerminal()` | Replace custom fitter with `FitAddon.fit()` thinking padding is now handled internally | Keep custom fitter; `FitAddon.fit()` hardcodes scrollbar width subtraction regardless |
| xterm.js font change | Set `fontFamily` and immediately call `fitTerminal()` synchronously | Wait one `requestAnimationFrame` after `fontFamily` change so CharSizeService remeasures |
| Web fonts + xterm.js | Reference `fontFamily` before font is loaded | `document.fonts.load('14px "FontName"')` → Promise resolves → set `fontFamily` → rAF → `fitTerminal()` |
| xterm.js ITheme v5 | Use `selection` key (xterm.js v4 API) | Use `selectionBackground` key; import `ITheme` from `@xterm/xterm` for compile-time validation |
| xterm.js theme assignment | `term.options.theme.background = '#...'` (property mutation) | `term.options.theme = { ...newTheme }` (new object) to trigger cache invalidation |
| `runtime.BrowserOpenURL` | Pass empty string when web server not started | Guard with `if (url !== '') { BrowserOpenURL(url) }` |
| `runtime.ClipboardSetText` | Trust return value as definitive success | Always show URL as selectable text alongside copy button as fallback |
| Sidebar collapsed CSS | Add `justify-content: center` globally | Scope to `.sidebar--collapsed .sidebar__item` only; expanded state needs left alignment |
| Sidebar bottom item | Fix only top nav items | `sidebar__bottom .sidebar__item` also needs `justify-content: center` in collapsed state |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Font change triggers rAF retry loop that exits early | Terminal dims calculated with old font cell width after font change | rAF retry loop is for zero-dimension case; font change needs single rAF after change, not the retry loop | Every font change in theming phase |
| QR code re-renders on every URL poll interval | QR image flickers every 3 seconds | Derive QR from stable state; only update when URL actually changes (use `useMemo` with URL as dep) | Any QR code component inside a polling parent |
| Multiple `ResizeObserver` callbacks when sidebar animates | `fitTerminal()` fires ~9 frames during 150ms transition | `ResizeObserver` is already in place and fires naturally; this is acceptable for smooth transition | Sidebar with `transition: width 0.15s` — expected behavior |
| `document.fonts.load()` called on every font switch | Slight delay per font switch (network round-trip for web fonts) | Cache loaded font names in a Set; skip `fonts.load()` for already-loaded fonts | Every theme switch if fonts are not cached |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Include web server URL (with Basic Auth password embedded) in QR code | QR image captured in screenshot, shared, exposes password | Dashboard QR uses URL without credentials — Tailscale mode needs no creds; local mode uses Basic Auth header not URL param |
| `BrowserOpenURL` with user-controlled URL string | URL injection (javascript:, file: schemes) — blocked by Wails PR #4484 URL validation | Wails v2 validates URL scheme in `BrowserOpenURL`; frontend should still validate before calling |
| External font CDN in `@font-face` (Google Fonts, etc.) | Font request reveals user IP to third party; breaks offline use | Bundle fonts as static assets in `frontend/public/fonts/` and serve from Wails embed |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Terminal padding applied inconsistently (some tabs have it, some don't) | Visual jank when switching tabs | Padding is a global terminal option set at `Terminal` constructor time — all tabs use the same value |
| Theme switch rerenders all terminal tabs causing brief flicker | Feels buggy | Apply theme via `term.options.theme = newTheme` without dismounting the panel; xterm updates in-place without a DOM remount |
| "Copied!" confirmation shown even when clipboard API fails | User trusts confirmation and doesn't try again | Show confirmation only after verifying — for JS-side clipboard, the Promise rejects on failure; for Wails-side, rely on a truthy return value but add fallback text display |
| QR code shown when web server is stopped | Confusing — QR encodes a URL that returns connection refused | Hide QR code button/section entirely when web server is not running |
| Sidebar icons misaligned in collapsed state | Feels unfinished; reduces confidence in the app | Fix `justify-content` in CSS before shipping any other sidebar visual changes |
| Font selector lists fonts not installed on system | User picks "JetBrains Mono", gets fallback monospace rendering | Either bundle all offered fonts as web fonts in `frontend/public/fonts/` or detect availability with `document.fonts.check()` before listing |

---

## "Looks Done But Isn't" Checklist

- [ ] **Terminal padding:** Padding applied — verify `term.cols` matches `Math.floor((containerWidth - 2*padding) / cellWidth)` within ±1
- [ ] **Terminal padding + FitAddon:** After adding padding — verify the custom `fitTerminal()` is still being used (not `fitAddon.fit()`) by checking that no `14px` scrollbar gap appears on the right
- [ ] **Terminal padding + sidebar transition:** Collapse and expand sidebar — verify terminal re-fits to the new container width after the transition completes
- [ ] **Font change timing:** Switch font — verify `term.cols` matches the new font's cell width, not the old font's, by checking the PTY still renders correctly after font change
- [ ] **Web font loading:** Pick a web font that's NOT a system font — verify it loads before the terminal renders by checking there is no flash of wrong-font-width output
- [ ] **Theme ITheme API:** Apply a theme with `selectionBackground` set — select some text in the terminal and verify the selection color is visible (not transparent)
- [ ] **Theme persistence:** Change theme, close and reopen the app window — verify the theme is still applied to new terminal sessions (was persisted to localStorage)
- [ ] **Theme all tabs:** Change theme with 3 tabs open — verify all 3 tabs show the new theme, not just the active one
- [ ] **Browser open empty URL:** Web server not running, click "Open in Browser" — verify nothing happens (or an error is shown), not a browser tab opening `about:blank` or `undefined`
- [ ] **Clipboard fallback:** Simulate clipboard failure (test in environment without clipboard) — verify the URL is displayed as selectable text so user can copy manually
- [ ] **QR URL currency:** Start web server in local mode, verify QR encodes LAN IP URL; then connect Tailscale and let server restart in Tailscale mode — verify QR now encodes the FQDN URL
- [ ] **Sidebar icons collapsed:** Collapse sidebar — verify all icons (Home, Remote, Sessions, New Session, Settings) are horizontally centered within the 48px width
- [ ] **Sidebar Settings icon:** Settings is in `sidebar__bottom` — verify Settings icon is ALSO centered when collapsed, not just the top nav items

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Terminal padding via CSS container (wrong approach) | LOW | Move padding from container CSS to `terminal.options`; remove CSS padding; re-fit all open terminals |
| FitAddon.fit() accidentally restored | LOW | Restore custom `fitTerminal()` call; remove `fitAddon.fit()` call |
| Wrong font dims after font change (no rAF delay) | LOW | Add `requestAnimationFrame` wrapper around `fitTerminal()` call in font change handler |
| Web font not bundled (breaks offline, leaks IP) | MEDIUM | Download font files, add to `frontend/public/fonts/`, update `@font-face` declarations in CSS |
| Theme using old `selection` key (no selection color) | LOW | Replace `selection` with `selectionBackground` in all theme objects; TypeScript import of `ITheme` catches this |
| QR encoding stale URL | LOW | Ensure QR derives from same state variable as URL text; React re-renders both together |
| Sidebar icon off-center | LOW | Add `.sidebar--collapsed .sidebar__item { justify-content: center; }` to style.css; verify for both nav items and sidebar__bottom |
| Clipboard silent failure | LOW | Add URL-as-text display as permanent UI element alongside copy button; no code fix needed |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Terminal padding breaks FitAddon cols | Terminal padding phase | `term.cols` correct; no right-side gap; sidebar toggle triggers re-fit |
| FitAddon.fit() accidentally restored | Terminal padding phase | Custom `fitTerminal()` in TerminalPanel.tsx still used; no `fitAddon.fit()` call |
| Font change needs rAF delay | Terminal theming phase | Cols correct after font switch; PTY receives correct resize |
| Web font race condition | Terminal theming phase | `document.fonts.load()` called before `fontFamily` set for non-system fonts |
| ITheme `selection` → `selectionBackground` | Terminal theming phase | Selection text visible; TypeScript build passes with `ITheme` import |
| Theme must be global not per-tab | Terminal theming phase | All open tabs update simultaneously; theme persists after restart |
| `BrowserOpenURL` silent failure | Web server links phase | Empty URL guarded in frontend; Linux tested |
| Clipboard failure — no fallback text | Web server links phase | URL displayed as selectable text alongside copy button |
| QR encodes stale URL | Web server links phase | QR updates when web server URL changes |
| Sidebar icons not centered when collapsed | Sidebar centering phase | All icons (including Settings in bottom) centered in 48px sidebar |

---

## Sources

- xterm.js GitHub — FitAddon padding subtraction source: `addons/addon-fit/src/FitAddon.ts` (reads `elementStyle.paddingLeft/Right/Top/Bottom` from `.xterm` element; subtracts from parent dimensions) — HIGH confidence (official source, verified 2026-04-10)
- xterm.js GitHub Discussion #5299 — FitAddon does not fit to last pixel row; workaround uses `_core._renderService.dimensions.css.cell.height` private API — MEDIUM confidence (community discussion, 2025-01-19)
- xterm.js GitHub Issue #4841 — FitAddon resizes incorrectly; integer vs. float rounding; timing dependency on `term.open()` before `fit()` — HIGH confidence (official tracker)
- xterm.js GitHub Issue #1164 — Web fonts and canvas renderer: font must be loaded before `Terminal.open()`; Firefox aggressively skips font download; no built-in font-ready callback — HIGH confidence (official tracker)
- xterm.js GitHub Issue #1499 — `setOption('fontSize')` and font weight changes interact; CharSizeService remeasures asynchronously — HIGH confidence (official tracker)
- xterm.js Release 5.0.0 — `ITheme.selection` renamed to `ITheme.selectionBackground`; `selectionInactiveBackground` added — HIGH confidence (official release notes)
- xterm.js docs — `ITheme` interface: `selectionBackground`, `selectionInactiveBackground` — HIGH confidence (official docs)
- Wails GitHub Issue #3261 — `BrowserOpenURL` calls `pkg/browser.OpenURL`; errors silently ignored; no fallback — HIGH confidence (official tracker)
- Wails GitHub Issue #2534 — `ClipboardSetText` returns success but clipboard empty on macOS M1 (v2.4.1) — HIGH confidence (official tracker)
- Wails GitHub PR #4484 — URL validation added to `BrowserOpenURL` (blocks `javascript:`, `data:`, `file:`, `ftp:` schemes) — HIGH confidence (official PR)
- Wails docs — `runtime.ClipboardSetText`, `runtime.BrowserOpenURL` signatures — HIGH confidence (official docs)
- Project codebase — `frontend/src/components/TerminalPanel.tsx` (custom `fitTerminal()` function, rAF retry loop, `ResizeObserver`, `fontSize` useEffect), `frontend/src/components/Sidebar.tsx` (collapsed state logic, conditional label render), `frontend/src/style.css` (sidebar CSS — `sidebar__item` uses `display: flex; align-items: center; gap: 8px; padding: 8px`; no `justify-content: center` on items) — HIGH confidence (live codebase, read 2026-04-10)
- PROJECT.md — Key decisions: "Bounded rAF retry loop polling proposeDimensions() ✓ Good"; "Frontend cols/rows estimation at session creation ✓ Good"; "ResizeObserver + requestAnimationFrame for fit() ✓ Good"; tech debt note: "FitAddon has required careful handling" — HIGH confidence (project history)

---
*Pitfalls research for: Go/Wails desktop app v1.12 — terminal padding, terminal theming, web server link actions, sidebar icon centering*
*Researched: 2026-04-10*
