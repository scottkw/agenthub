# Feature Research

**Domain:** UI/UX polish for Go/Wails + React + xterm.js desktop terminal app (v1.12)
**Researched:** 2026-04-10
**Confidence:** HIGH (xterm.js API verified via official docs; CSS padding approach verified via xterm.js issue history; TokyoNight colors verified via upstream source; codebase read directly)

---

## Context: What Already Exists

This is a SUBSEQUENT MILESTONE. The following are already built and are NOT scope for v1.12:

- Tabbed xterm.js terminal sessions with per-tab font size (Shift+= / Shift+-)
- Collapsible left sidebar with Heroicons SVG icons (width: 200px expanded, 48px collapsed)
- Web serving with Tailscale HTTPS or local network (self-signed TLS + password) fallback
- Per-session QR codes and web dashboard
- Settings as sidebar tab (Web Server subtab shows running URL as a plain anchor)
- `BrowserOpenURL()` already imported and called in App.tsx (for remote session open)
- `navigator.clipboard.writeText()` already used in SettingsTab (for LAN password copy)
- TokyoNight Night palette already hardcoded as the app theme (bg `#1a1b26`, fg `#c0caf5`)
- Custom `fitTerminal()` in TerminalPanel.tsx already reads CSS padding from `term.element` via `getComputedStyle` and subtracts it from available width/height — padding-aware fit is already implemented

---

## Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Terminal padding (inset) | Professional terminal emulators (VS Code, iTerm2, Warp, Ghostty) all pad text so it does not touch container edges. Text flush to edges looks unfinished. | LOW | No built-in padding option in `@xterm/xterm` 6.x (confirmed via official ITerminalOptions docs — no `padding` property exists). The approach is CSS: add `padding: 6px` to `.xterm` in style.css. The existing custom `fitTerminal()` already reads `paddingLeft/Right/Top/Bottom` from `getComputedStyle(term.element!)` (TerminalPanel.tsx lines 24-29) and subtracts them from cols/rows — the fit logic is already padding-aware. |
| Web server URL opens in default browser | Every desktop app affords clicking a URL to open in the system browser. Wails WebView does NOT open external URLs from `<a href target="_blank">` — it renders in-app or does nothing. The current SettingsTab renders `<a href={serverURL} target="_blank" rel="noreferrer">` (line 305) which is broken for this purpose. | LOW | `BrowserOpenURL(url)` is the Wails runtime call that opens the system browser. Pattern already established in App.tsx line 412 (`BrowserOpenURL(url)` in `handleOpenRemoteSession`). Fix is replacing the anchor with a button or styled link calling `BrowserOpenURL(serverURL)`. |
| Copy-to-clipboard for web server URL | Any URL displayed in a settings UI is expected to be copyable with one click. There is no copy button for `serverURL` today. | LOW | `navigator.clipboard.writeText()` + 1.5s "Copied!" feedback already implemented for LAN password in SettingsTab (lines 89-93, 330-333). Identical pattern applies to the server URL. |
| Sidebar icons centered when collapsed | When sidebar collapses to 48px, icons should center horizontally. Current `.sidebar__item` uses `padding: 8px` with no `justify-content` set, so items default to `flex-start`. Icon (`20px` wide) sits left-aligned inside `48px` button — visually off-center. The toggle button (`.sidebar__toggle`) already uses `justify-content: center` — items should match. | LOW | Pure CSS fix: `.sidebar--collapsed .sidebar__item { justify-content: center; padding-left: 0; padding-right: 0; }`. No JS changes. |

---

## Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Terminal theme presets | The app already uses TokyoNight Night palette. Offering curated theme variants (TokyoNight Storm/Moon, Dracula, Catppuccin Mocha) in Settings lets users personalize without leaving the app. xterm.js `theme` option accepts a full `ITheme` object — all 16 ANSI colors + foreground/background/cursor — as a live-swappable option on any existing terminal instance. | MEDIUM | `term.options.theme = newTheme` updates a live terminal (confirmed via xterm.js ITheme docs). Theme should be stored in `localStorage` and applied at terminal creation and on change. Since all TerminalPanels need the theme, lift theme state to App level (same pattern as `fontSize`). Define a `THEMES` const map in a `themes.ts` file. |
| QR code for web server dashboard URL | Per-session QR codes already exist. A QR code for the dashboard root URL in Settings completes the trio (open, copy, QR) and lets users instantly load the dashboard on a phone without typing the URL. | LOW | Backend needs `GetWebServerQRCode()` Go binding using the same `skip2/go-qrcode` library already used in `GetSessionQRCode`. The dashboard URL is the same as `GetWebServerURL()`. Frontend can reuse the existing `QRModal` component or render an inline base64 SVG/PNG. |
| Font family selection | Developers have strong preferences for terminal fonts. Current font stack is hardcoded as `"Cascadia Code", "MesloLGS NF", "Fira Code", monospace`. A curated dropdown of 4 fonts in Settings lets users match their IDE. | MEDIUM | xterm.js `fontFamily` option accepts a CSS font-family string. Font change requires `fitTerminal()` re-trigger because char width changes. Lift `fontFamily` to App-level state (like `fontSize`). List only fonts likely pre-installed on developer machines; always fall back to `monospace`. No font download needed. |

---

## Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Full custom theme editor (color picker per ANSI slot) | Power users want exact control | 16 color pickers + fg/bg/cursor = complex UI, high maintenance, rarely used by most users, difficult to persist safely | Curated preset themes (4-5 options) cover 95% of use cases at 5% of implementation cost |
| Per-tab theme selection | Some users want different themes per project | Multiplies state management: theme stored per session, synced with backend, reflected on re-attach — enormous scope for marginal gain | Global theme preference is sufficient; tabs inherit app theme |
| Custom font upload or install | Users with rare fonts want to use them | Requires OS-level font installation, cross-platform paths, security concerns; out of scope for a desktop app that avoids managing system state | Ship 4-5 well-known pre-installed fonts; document others |
| Animated theme transitions | Looks polished in mockups | xterm.js re-renders full canvas on theme change; CSS transitions on canvas elements have no effect — theme swap is always instant | Instant swap is the correct behavior; no workaround is appropriate |
| Configurable padding per side (top/left/right/bottom) | Fine-grained control | Adds UI complexity for very limited visual gain; uniform padding of 4-8px looks correct in all orientations | Single uniform padding constant set to a tuned value (6-8px) |

---

## Feature Dependencies

```
Terminal Padding (CSS)
    depends on --> Custom fitTerminal() [already exists — already reads CSS padding]
    depends on --> .xterm CSS class [add padding: 6px]
    no JS changes needed

Sidebar Icon Centering
    depends on --> .sidebar--collapsed CSS class [already exists]
    no JS changes needed

Web Server URL: Open in Browser
    depends on --> BrowserOpenURL() [already imported in App.tsx]
    depends on --> isServerRunning + serverURL state [already in SettingsTab]

Web Server URL: Copy to Clipboard
    depends on --> navigator.clipboard.writeText [already used in SettingsTab]
    depends on --> serverURL state [already in SettingsTab]
    enhances --> Open in Browser (both operate on same URL, presented together)

Web Server QR Code
    depends on --> GetWebServerURL() [already exists]
    depends on --> skip2/go-qrcode [already in go.mod]
    requires --> new Go binding: GetWebServerQRCode()
    reuses --> QRModal component or inline base64 image pattern
    enhances --> Open in Browser + Copy URL (completes the URL access trio)

Terminal Theme Presets
    depends on --> xterm.js ITheme object [accepted by Terminal constructor and term.options.theme]
    depends on --> localStorage [for persistence across sessions]
    requires --> THEMES const map in themes.ts
    requires --> theme state lifted to App level (like fontSize)
    requires --> Settings UI: theme selector dropdown
    enhances --> Terminal Padding (both are terminal appearance; same settings section)

Font Family Selection
    depends on --> xterm.js fontFamily option [accepted by Terminal constructor]
    requires --> fontFamily state lifted to App level (like fontSize)
    requires --> fitTerminal() re-trigger on font change (char width changes)
    requires --> Settings UI: font family dropdown
    enhances --> Terminal Theme (both are terminal appearance settings)
```

### Dependency Notes

- **Terminal Padding requires no JS changes:** `fitTerminal()` in TerminalPanel.tsx lines 24-29 already calls `getComputedStyle(term.element!)` and subtracts computed `paddingLeft/Right/Top/Bottom` from parent dimensions before calculating cols/rows. Adding CSS padding to `.xterm` automatically flows through the existing calculation.
- **Theme change does NOT require fitTerminal re-trigger:** Color changes don't affect character dimensions. Font family changes DO require it — char width changes with different fonts.
- **All web server UX features share existing SettingsTab state:** `isServerRunning` and `serverURL` are already in SettingsTab. No new data fetching needed for open/copy/QR.
- **Theme and fontSize share the same lift pattern:** `fontSize` is already lifted to App-level state and passed as a prop to each TerminalPanel. Theme follows the identical pattern.

---

## MVP Definition for v1.12

### Launch With (all v1.12 scope)

- [ ] Terminal padding — CSS only, immediate visual quality improvement, zero risk
- [ ] Sidebar icon centering when collapsed — CSS only, pure fix
- [ ] Web server URL: Open in Browser button — replaces broken anchor with `BrowserOpenURL`
- [ ] Web server URL: Copy to clipboard — reuses established clipboard pattern
- [ ] Web server QR code — one new Go binding + reuse existing QR infrastructure
- [ ] Terminal theme presets — TokyoNight Night (current) + Storm/Moon + 1-2 other popular themes

### Add After Validation (v1.12.x)

- [ ] Font family selection — useful but depends on users having target fonts installed; validate demand with v1.12 theme feedback before adding
- [ ] Additional theme presets (Catppuccin Mocha, One Dark) — can add as point releases without a full milestone

### Future Consideration (v2+)

- [ ] Per-tab theme or font override — complex state propagation; defer until explicitly requested
- [ ] Custom theme editor — high complexity, low priority

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Sidebar icon centering | MEDIUM | LOW | P1 |
| Terminal padding | HIGH | LOW | P1 |
| Web server URL: open in browser | HIGH | LOW | P1 |
| Web server URL: copy to clipboard | HIGH | LOW | P1 |
| Web server QR code | MEDIUM | LOW | P1 |
| Terminal theme presets | MEDIUM | MEDIUM | P1 |
| Font family selection | LOW | MEDIUM | P2 |

**Priority key:**
- P1: Target for v1.12 milestone
- P2: Add after validation, not blocking v1.12

---

## Implementation Notes by Feature

### Terminal Padding

**Root cause of current state:** No padding CSS is applied to `.xterm`. Terminal renders edge-to-edge within its container.

**Fix:** In `style.css`, add:
```css
.xterm {
  padding: 6px;
}
```

**Why this works:** `fitTerminal()` reads `getComputedStyle(term.element!)` and explicitly extracts `paddingLeft`, `paddingRight`, `paddingTop`, `paddingBottom`, then:
- `padH = paddingLeft + paddingRight` subtracted from `parentW` before cols calculation
- `padV = paddingTop + paddingBottom` subtracted from `parentH` before rows calculation

This means cols/rows are calculated for the content area (inside padding), not the full container. PTY is resized correctly.

**Tuned value:** 6-8px is standard for terminal emulators. VS Code Terminal uses approximately 6px horizontal padding. Warp uses ~8px.

**Risk:** Very low. The scrollbar is already hidden via `.xterm-viewport { scrollbar-width: none }` so scrollbar width (which PR #1208 identified as a concern) is 0px and does not create a mismatch.

### Sidebar Icon Centering

**Root cause:** `.sidebar__item` sets `display: flex; align-items: center; gap: 8px; padding: 8px`. No `justify-content` is set, so it defaults to `flex-start`. When label is hidden (collapsed state), the icon (`20px`) sits at the left edge of the `48px`-wide button with 8px left padding, placing icon center at `8 + 10 = 18px` from left edge — not centered in 48px.

**Fix:**
```css
.sidebar--collapsed .sidebar__item {
  justify-content: center;
  padding-left: 0;
  padding-right: 0;
}
```

**Toggle button reference:** `.sidebar__toggle` already uses `justify-content: center` and produces a correctly centered icon. Items should match.

### Web Server URL UX (Open, Copy, QR)

**Current state:** `SettingsTab.tsx` line 303-306:
```jsx
{isServerRunning && serverURL && (
  <p className="settings-panel__url">
    Server running at: <a href={serverURL} target="_blank" rel="noreferrer">{serverURL}</a>
  </p>
)}
```

This anchor does not open the system browser in Wails WebView.

**Target UX:** When server is running, show a compact row:
- URL text (truncated if long)
- "Open" icon button → `BrowserOpenURL(serverURL)`
- "Copy" icon button → `navigator.clipboard.writeText(serverURL)` + "Copied!" feedback
- "QR" icon button or toggle → shows QR code for dashboard URL

**QR code for dashboard:** Add to `App.go`:
```go
func (a *App) GetWebServerDashboardQRCode() (string, error) {
    url, err := a.daemon.GetWebServerURL()
    if err != nil { return "", err }
    return generateQRCode(url) // same as GetSessionQRCode logic
}
```

Frontend uses `GetWebServerDashboardQRCode()` when user clicks the QR button; displays base64 PNG inline (same pattern as `GetSessionQRCode` in QRModal).

### Terminal Theme Presets

**xterm.js ITheme API (confirmed from official docs):**
All properties are optional strings (CSS color values). Setting `term.options.theme` on an existing terminal immediately re-renders with new colors.

**TokyoNight Night colors (verified from folke/tokyonight.nvim upstream):**
- background: `#1a1b26`, foreground: `#c0caf5`, cursor: `#c0caf5`
- black: `#15161e`, red: `#f7768e`, green: `#9ece6a`, yellow: `#e0af68`
- blue: `#7aa2f7`, magenta: `#bb9af7`, cyan: `#7dcfff`, white: `#a9b1d6`
- brightBlack: `#414868`, brightRed: `#ff899d`, brightGreen: `#9fe044`
- brightYellow: `#faba4a`, brightBlue: `#8db0ff`, brightMagenta: `#c7a9ff`
- brightCyan: `#a4daff`, brightWhite: `#c0caf5`

**Implementation plan:**
1. Create `frontend/src/lib/themes.ts` with a `THEMES` const record mapping theme name to `ITheme` object. Include TokyoNight Night (current hardcoded colors), TokyoNight Storm (bg `#24283b`), and 1-2 others.
2. Add `theme` state to App.tsx (string key), initialized from `localStorage.getItem('terminal-theme') ?? 'tokyonight-night'`.
3. Pass `theme` as a prop to each `TerminalPanel` (like `fontSize`).
4. In `TerminalPanel`, apply `theme: THEMES[theme]` at terminal creation and via `useEffect` on theme prop change.
5. Add theme selector `<select>` to SettingsTab (new "Appearance" subtab or inline in CLI Paths tab).

**Scope clarification:** `PROJECT.md` listed "Font/theme customization beyond size" as deferred in earlier milestones. v1.12 milestone explicitly includes "Terminal theming (popular theme support for fonts and colors)" — this is intentionally scoped in.

---

## Sources

- xterm.js ITerminalOptions official docs (no padding option confirmed): https://xtermjs.org/docs/api/terminal/interfaces/iterminaloptions/
- xterm.js ITheme official docs (all theme properties confirmed): https://xtermjs.org/docs/api/terminal/interfaces/itheme/
- xterm.js padding CSS approach (merged PR, v3.1.0): https://github.com/xtermjs/xterm.js/issues/946 and https://github.com/xtermjs/xterm.js/pull/1208
- TokyoNight Night ANSI colors (verified upstream): https://github.com/folke/tokyonight.nvim/blob/main/extras/alacritty/tokyonight_night.toml
- Copy-to-clipboard UX patterns: https://cloudscape.design/components/copy-to-clipboard/
- AgentHub codebase read directly: `TerminalPanel.tsx`, `Sidebar.tsx`, `SettingsTab.tsx`, `App.tsx`, `style.css`, `App.d.ts` (HIGH confidence)

---

*Feature research for: AgentHub v1.12 — Terminal Padding, Theming, Web Server Link UX, Sidebar Icon Centering*
*Researched: 2026-04-10*
