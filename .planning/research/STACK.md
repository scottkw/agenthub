# Stack Research

**Domain:** UI/UX polish — terminal theming, terminal padding, web server link UX, sidebar icon centering
**Researched:** 2026-04-10
**Confidence:** HIGH

## Context: What This Research Covers

This is a subsequent-milestone research file for v1.12. The existing stack (Go/Wails v2, React, xterm.js, nhooyr/websocket, go-pty, kardianos/service, tailscale.com/client/local) is validated and not re-researched here. This file covers only what is NEW for v1.12:

1. Terminal theming (popular color schemes for xterm.js)
2. Terminal padding (inset so text doesn't touch edges)
3. Web server link UX (open in browser, copy URL, QR code for dashboard)
4. Sidebar icon centering when collapsed

**Existing bindings already in place — no new setup needed:**
- `BrowserOpenURL(url: string): void` — Wails runtime, already in `wailsjs/wailsjs/runtime/runtime.d.ts`
- `ClipboardSetText(text: string): Promise<boolean>` — Wails runtime, already in same file
- `@xterm/xterm ^6.0.0` — already installed
- `@xterm/addon-fit ^0.11.0` — already installed (project uses custom `fitTerminal()` over stock FitAddon)
- `@heroicons/react ^2.2.0` — already installed
- `skip2/go-qrcode` — already in `go.mod`

---

## Feature 1: Terminal Theming

**Verdict:** One new npm package needed.

### Recommended: `xterm-theme@1.1.0`

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| `xterm-theme` | 1.1.0 | 217 iTerm2-derived theme definitions for xterm.js | Plain JS objects matching `ITheme` interface exactly; no runtime coupling; tree-shakeable; MIT license |

**Installation:**
```bash
cd frontend && pnpm add xterm-theme@1.1.0
```

**Compatibility:** HIGH confidence. `xterm-theme` exports plain JS objects with keys matching xterm.js `ITheme` (`foreground`, `background`, `cursor`, `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, and bright variants). The xterm.js `ITheme` interface is stable across v5 and v6. Although the package lists `xterm` (deprecated) as a peer dep, it is pure data — no xterm.js runtime imports. pnpm may warn about the peer dep; use `--legacy-peer-deps` if needed.

**Themes to expose (recommended initial set of 10):**

| Name in package | Display name | Style |
|----------------|-------------|-------|
| `Dracula` | Dracula | Dark, purple tones — most popular dark theme |
| `OneHalfDark` | One Half Dark | Dark, popular VS Code default |
| `OneHalfLight` | One Half Light | Light version |
| `Solarized Dark` | Solarized Dark | Classic, readable dark |
| `Solarized Light` | Solarized Light | Classic light |
| `Tomorrow Night` | Tomorrow Night | Dark, easy on eyes |
| `Monokai Soda` | Monokai | Dark, developer favourite |
| `Material` | Material | Google Material colors |
| `Gruvbox Dark` | Gruvbox | Retro warm dark |
| `ayu` | Ayu | Modern minimal dark |

**Current state in TerminalPanel.tsx:**
```typescript
const term = new Terminal({
  theme: { background: '#1a1b26' },  // ← replace with full ITheme from xterm-theme
  ...
})
```

**Integration pattern:**
```typescript
import { Dracula, OneHalfDark, SolarizedDark } from 'xterm-theme'

// At creation:
const term = new Terminal({ theme: Dracula, ... })

// Runtime change (no terminal recreation needed):
term.options.theme = OneHalfDark
```

**Persistence:** Store selected theme name in `localStorage`. Pass as prop `App.tsx → TerminalPanel`. Apply at `new Terminal({ theme: selectedTheme })` and on user change via `term.options.theme = newTheme`.

---

## Feature 2: Terminal Padding

**Verdict:** No new library. CSS-only change. Custom `fitTerminal()` already handles padding correctly.

**Key discovery:** The project does NOT use stock `FitAddon.fit()`. It has a custom `fitTerminal()` function that already reads CSS padding from the terminal element:

```typescript
// Existing code in TerminalPanel.tsx — already accounts for padding:
const elStyle = window.getComputedStyle(term.element!)
const padH = parseInt(elStyle.paddingLeft) + parseInt(elStyle.paddingRight)
const padV = parseInt(elStyle.paddingTop) + parseInt(elStyle.paddingBottom)

const cols = Math.max(2, Math.floor((parentW - padH) / dims.css.cell.width))
const rows = Math.max(1, Math.floor((parentH - padV) / dims.css.cell.height))
```

This means adding CSS padding to `.xterm` element automatically produces correct terminal sizing with no JS changes. This is the correct insertion point because `fitTerminal()` reads from `term.element!` (the `.xterm` div, not the container).

**Implementation:** One CSS rule:
```css
.xterm {
  padding: 8px 12px;
}
```

**Why not a config option:** xterm.js `ITerminalOptions` has no `padding` property. The feature request (GitHub issue #946) was closed in v3.1.0 via a PR that made xterm aware of CSS padding in mouse coordinate calculation — but there is no `padding: N` option in `ITerminalOptions`. CSS is the only approach, and it works correctly here because the custom fitter already reads it.

**No new Go changes. No new npm packages.**

---

## Feature 3: Web Server Link UX

**Verdict:** No new libraries. All bindings already exist. Two areas to update: `StatusBar.tsx` (session URLs) and `SettingsTab.tsx` (dashboard URL). One new Go binding needed for dashboard QR.

### Session URL (StatusBar.tsx) — current state

```typescript
// Current: opens inside Wails WebView — WRONG for URLs
<a className="tab-status-bar__url" href={sessionURL} target="_blank" rel="noreferrer">
  {sessionURL}
</a>
```

`target="_blank"` in Wails WebView opens within the WebKit view, not the system browser. `BrowserOpenURL` is the correct API.

### Changes needed

**StatusBar.tsx:**
- Replace `<a href>` with text span + "Open" button calling `BrowserOpenURL(sessionURL)` from Wails runtime
- Add copy icon button calling `ClipboardSetText(sessionURL)` from Wails runtime

**SettingsTab.tsx:**
- Replace `<a href={serverURL}>` with text + "Open" button calling `BrowserOpenURL(serverURL)`
- Add "Copy" button calling `ClipboardSetText(serverURL)`
- Add "QR" button showing QR code modal for the dashboard URL

**QR for dashboard URL:** `skip2/go-qrcode` already exists. The existing `QRModal.tsx` and `GetSessionQR(id)` binding exist for per-session QR. Add one new Go binding `GetWebServerQR() (string, error)` in `App.go` that generates a QR PNG (base64) for the web server dashboard URL. This reuses the exact same `skip2/go-qrcode` pattern already in the codebase.

**Import pattern for Wails runtime bindings:**
```typescript
import { BrowserOpenURL, ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'
```

Both are already exported from the runtime file — no `wails generate` needed.

### clipboard: `ClipboardSetText` vs `navigator.clipboard.writeText`

SettingsTab already uses `navigator.clipboard.writeText(localPassword)` for the password copy button. Either approach works in Wails (WebKit runs in a secure context). For the URL copy buttons, `ClipboardSetText` from the Wails runtime is preferred — it uses the OS clipboard API directly and is consistent with desktop patterns. Keep the existing password copy as-is to avoid unnecessary churn.

---

## Feature 4: Sidebar Icon Centering

**Verdict:** Pure CSS fix. No library, no JS changes.

**Problem:** When sidebar is collapsed, `.sidebar__item` has `display: flex; align-items: center; gap: 8px; padding: 8px`. With no `justify-content`, it defaults to `flex-start`, which pushes icons left instead of centering them in the 48px collapsed sidebar.

**Current CSS:**
```css
.sidebar--collapsed {
  width: 48px;
}

.sidebar__item {
  display: flex;
  align-items: center;  /* vertical — correct */
  gap: 8px;
  padding: 8px;
  width: 100%;
  text-align: left;
  /* NO justify-content: defaults to flex-start → icon hugs left edge */
}
```

**Fix:**
```css
.sidebar--collapsed .sidebar__item {
  justify-content: center;
  padding: 8px 0;
}
```

`padding: 8px 0` removes horizontal padding in collapsed state, ensuring the icon sits at the geometric center of the 48px sidebar.

The `.sidebar__toggle` already has `justify-content: center` (correct). Only `.sidebar__item` needs this fix.

---

## Summary: New Dependencies

| Type | Item | Action |
|------|------|--------|
| npm | `xterm-theme@1.1.0` | `pnpm add xterm-theme@1.1.0` in `frontend/` |
| Go | none | No new packages — all bindings already exist |
| CSS | none | Changes to existing `style.css` only |
| Wails bindings | `GetWebServerQR()` | New method in `App.go`, generated binding in `wailsjs/` |

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `react-color` or color picker libraries | Out of scope; this is a theme selector, not a color editor | `<select>` over predefined themes from `xterm-theme` |
| Stock `FitAddon.fit()` for padding | Subtracts hardcoded 14px scrollbar width; ignores CSS padding | Existing custom `fitTerminal()` in `TerminalPanel.tsx` — already handles padding |
| `ITerminalOptions.padding` | Does not exist in xterm.js API | CSS on `.xterm { padding: 8px 12px }` |
| `<a href target="_blank">` for opening URLs in Wails | Opens inside WebKit WebView, not system browser | `BrowserOpenURL(url)` from Wails runtime |
| New QR library | `skip2/go-qrcode` already present | Reuse existing pattern with new `GetWebServerQR()` binding |
| External CSS framework for sidebar fix | 2-line CSS change | `.sidebar--collapsed .sidebar__item { justify-content: center }` |
| `navigator.clipboard.writeText()` for new URL copy buttons | Works but less idiomatic for desktop | `ClipboardSetText()` Wails runtime binding |
| Storing theme in Go/backend | UI preference, no backend relevance | `localStorage` key in frontend only |

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `xterm-theme@1.1.0` | `@xterm/xterm@^6.0.0` | Compatible as pure data; ITheme interface stable across v5/v6; no runtime coupling |
| `@xterm/xterm@^6.0.0` (existing) | Custom `fitTerminal()` with CSS padding | Verified: custom fitter reads `getComputedStyle(term.element!)` padding |
| Wails runtime `BrowserOpenURL` | Wails v2.10.2 (existing) | Already declared in `runtime.d.ts`; no version change |
| Wails runtime `ClipboardSetText` | Wails v2.10.2 (existing) | Already declared in `runtime.d.ts`; no version change |

---

## Sources

- https://xtermjs.org/docs/api/terminal/interfaces/itheme/ — ITheme fields confirmed (HIGH confidence, official docs)
- https://xtermjs.org/docs/api/terminal/interfaces/iterminaloptions/ — Confirmed no `padding` option exists (HIGH confidence, official docs)
- https://github.com/xtermjs/xterm.js/discussions/5299 — FitAddon does not handle CSS padding; custom implementation required (MEDIUM confidence, maintainer response)
- https://github.com/xtermjs/xterm.js/issues/946 — Padding via CSS on `.xterm` element is the supported approach; merged in v3.1.0 (MEDIUM confidence, issue resolution)
- https://github.com/ysk2014/xterm-theme/blob/master/src/index.js — 217 themes enumerated by direct source inspection; all export plain objects (HIGH confidence, source code)
- Wails v2 docs — `ClipboardSetText` and `BrowserOpenURL` confirmed as runtime bindings (HIGH confidence, official Wails docs)
- `/Users/ken/dev/agenthub/frontend/src/wailsjs/wailsjs/runtime/runtime.d.ts` — Both bindings already declared in this repo (HIGH confidence, direct file inspection)
- `/Users/ken/dev/agenthub/frontend/src/components/TerminalPanel.tsx` — Custom `fitTerminal()` already reads `paddingLeft/paddingRight/paddingTop/paddingBottom` from `getComputedStyle(term.element!)` (HIGH confidence, direct code inspection)
- `/Users/ken/dev/agenthub/frontend/src/style.css` lines 154–224 — Sidebar CSS confirms missing `justify-content` on `.sidebar__item` (HIGH confidence, direct code inspection)

---

*Stack research for: AgentHub v1.12 UI/UX Polish*
*Researched: 2026-04-10*
