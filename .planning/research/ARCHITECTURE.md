# Architecture Research

**Domain:** UI/UX polish features for Go/Wails + React + xterm.js desktop app
**Researched:** 2026-04-10
**Confidence:** HIGH (all integration points verified from live codebase)

## Standard Architecture

### System Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                        React Frontend (WebView)                   │
├────────────┬────────────────────────────────────┬────────────────┤
│  Sidebar   │  App.tsx (root state + wiring)      │  TabBar        │
│  (nav)     │  - tab state                        │  (session tabs)│
│            │  - font sizes per session           │                │
│            │  - web serving state per session    │                │
│            │  - theme per session (NEW)           │                │
│            │  - padding per session (NEW)         │                │
├────────────┴──────────┬─────────────────────────┴────────────────┤
│   terminal-container  │  Non-terminal panels                      │
│   ┌──────────────┐    │  (Welcome, DaemonManager,                 │
│   │ TerminalPanel│    │   RemoteSessions, Settings)               │
│   │  xterm.js    │    │                                           │
│   │  FitAddon    │    │                                           │
│   └──────────────┘    │                                           │
│   ┌──────────────┐    │                                           │
│   │  StatusBar   │    │                                           │
│   │  (32px flex) │    │                                           │
│   └──────────────┘    │                                           │
├───────────────────────┴───────────────────────────────────────────┤
│              Wails Runtime Bindings (JS to Go bridge)             │
│  BrowserOpenURL · ClipboardSetText · EventsOn                     │
│  Go-generated App.* bindings (GetWebServerURL, GetSessionQRCode…) │
├───────────────────────────────────────────────────────────────────┤
│                     Go Backend (App struct)                        │
│  GetWebServerURL · GetSessionQRCode · GetWebServerMode            │
│  GetLocalNetworkPassword · StartWebServer · StopWebServer         │
│  DaemonClient (Unix socket) → internal/daemon SessionEngine       │
│  internal/webserver · skip2/go-qrcode                            │
└───────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Integration Notes |
|-----------|----------------|-------------------|
| `App.tsx` | Root state: tabs, fontSizes, webEnabled, sessionURLs | Owns new state: `themes` and `paddings` per session |
| `Sidebar.tsx` | Navigation: collapsed (48px) / expanded (200px), localStorage | CSS-only fix for icon centering when collapsed |
| `TerminalPanel.tsx` | xterm.js lifecycle, fit, font size prop | Receives `theme` and `padding` props; pass to `new Terminal({})` |
| `StatusBar.tsx` | 32px flex bar per terminal tab — web on/off, URL, QR button | Add open-in-browser button and copy-URL button |
| `SettingsTab.tsx` | CLI paths and web server config (two sub-tabs) | Add Appearance sub-tab for theme/padding defaults |
| `QRModal.tsx` | Modal showing session QR code (base64 PNG) | QR for dashboard URL is a new second use case |
| `App.go` | Wails-bound methods: GetWebServerURL, GetSessionQRCode | Add `GetDashboardQRCode()` for the dashboard URL QR |

## Feature Integration Points

### Feature 1: Terminal Padding

**What it is:** An inset margin so terminal text does not touch the container edges.

**Integration surface:**

- `TerminalPanel.tsx` has a custom `fitTerminal()` function that already reads `paddingLeft/Right/Top/Bottom` from `window.getComputedStyle(term.element!)`. This means CSS padding on the xterm element flows through `fitTerminal()` automatically — cols and rows are recalculated after subtracting `padH` and `padV`. This was built intentionally to support padding.
- The container `<div ref={containerRef}>` currently has `style={{ flex: 1, width: '100%', minHeight: 0 }}`. The xterm element rendered inside it (accessed as `term.element`) receives the padding.

**The correct approach:** Apply padding via `term.element.style.padding` after `term.open()`. The custom `fitTerminal` reads this via `window.getComputedStyle(term.element!)`. Do not apply padding to the outer container div — that div's padding is not what `fitTerminal` reads.

**New state in App.tsx:**
```typescript
const DEFAULT_PADDING = 8  // px
const [paddings, setPaddings] = useState<Record<string, number>>({})
```

**TerminalPanel prop addition:**
```typescript
interface TerminalPanelProps {
  // ...existing...
  padding: number  // px inset, applied to term.element.style.padding after open()
}
```

**fitTerminal compatibility:** No changes to `fitTerminal` logic. The function already subtracts `padH`/`padV` from parent dimensions when computing cols/rows.

**Settings UI:** Global default lives in a new "Appearance" sub-tab of `SettingsTab`. Store default in `localStorage` so it persists, same pattern as `sidebar-collapsed`.

---

### Feature 2: Terminal Theming

**What it is:** Selectable color themes (e.g., Tokyo Night, Dracula, Solarized Dark) applied to xterm.js terminal color palette and background.

**Integration surface:**

- `TerminalPanel.tsx` currently hardcodes `theme: { background: '#1a1b26' }` in the `new Terminal({})` constructor. The xterm.js `ITheme` interface supports: `background`, `foreground`, `cursor`, `cursorAccent`, `selectionBackground`, `selectionForeground`, plus 16 ANSI colors (`black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, and bright variants).
- xterm.js supports runtime theme changes via `term.options.theme = newTheme` without destroying the terminal instance. Scrollback is preserved.

**The correct approach:**

1. Define theme objects as constants in a new `frontend/src/lib/themes.ts` file. No backend involvement.
2. Pass `theme` string key as a prop to `TerminalPanel`. The panel maps it to the xterm `ITheme` object.
3. Font family is kept separate — it already implicitly comes from the hardcoded `fontFamily` string. Theme selection does not need to change font family.

**New state in App.tsx:**
```typescript
const DEFAULT_THEME = 'tokyo-night'
const [themes, setThemes] = useState<Record<string, string>>({})
```

**TerminalPanel prop addition:**
```typescript
interface TerminalPanelProps {
  // ...existing...
  theme: string  // key into THEMES map defined in lib/themes.ts
}
```

**Runtime theme switching effect (same pattern as fontSize):**
```typescript
useEffect(() => {
  if (!termRef.current) return
  termRef.current.options.theme = THEMES[theme] ?? THEMES['tokyo-night']
}, [theme])
```

**Theme file structure:**
```typescript
// frontend/src/lib/themes.ts
import type { ITheme } from '@xterm/xterm'
export const THEMES: Record<string, ITheme> = {
  'tokyo-night':   { background: '#1a1b26', foreground: '#a9b1d6', ... },
  'dracula':       { background: '#282a36', foreground: '#f8f8f2', ... },
  'solarized-dark': { ... },
  'github-dark':   { ... },
}
export const THEME_KEYS = Object.keys(THEMES) as string[]
```

**Settings UI:** Theme picker dropdown in the "Appearance" sub-tab of `SettingsTab`. The selected theme is stored in `localStorage` as the global default. Per-session overrides use `themes: Record<string, string>` state in `App.tsx`.

**No backend changes.** Themes are pure frontend.

---

### Feature 3: Web Server Link Improvements (Open in Browser, Copy URL, Dashboard QR)

**What it is:** In `SettingsTab` → Web Server sub-tab, when the server is running and the URL is shown, add: (a) open in default browser button, (b) copy-to-clipboard button, (c) QR code button for the dashboard URL.

**Integration surface:**

- `SettingsTab.tsx` already renders `serverURL` with an `<a>` tag when running. Inside Wails WebView, `target="_blank"` on anchor tags is unreliable — it does not reliably open the system browser. The correct call is `BrowserOpenURL(url)` from the Wails runtime.
- `BrowserOpenURL` is already imported and used in `App.tsx` and `WelcomeTab.tsx`. Adding it to `SettingsTab.tsx` is a one-line import change.
- `ClipboardSetText` is already in the Wails runtime bindings (`wailsjs/wailsjs/runtime/runtime.js:200`). It returns `Promise<boolean>`. The existing LAN password copy in `SettingsTab` uses `navigator.clipboard.writeText` — that should be replaced with `ClipboardSetText` for consistency and cross-platform reliability within Wails.
- For dashboard QR: the existing `GetSessionQRCode(sessionId)` generates QR for a session URL. A new `GetDashboardQRCode()` method encodes `serverURL` (the dashboard root, not a session path).

**Go backend change — add one new Wails method to `app.go`:**
```go
// GetDashboardQRCode generates a QR code for the web dashboard root URL and
// returns it as a base64-encoded PNG. Returns error if server not running.
func (a *App) GetDashboardQRCode() (string, error) {
    if a.client == nil {
        return "", fmt.Errorf("daemon not connected")
    }
    resp, err := a.client.GetWebServerStatus()
    if err != nil || !resp.Running {
        return "", fmt.Errorf("web server not running")
    }
    png, err := qrcode.Encode(resp.URL, qrcode.Medium, 256)
    if err != nil {
        return "", fmt.Errorf("GetDashboardQRCode: encode: %w", err)
    }
    return base64.StdEncoding.EncodeToString(png), nil
}
```

This is a minimal addition — it calls `qrcode.Encode` (already imported as `skip2/go-qrcode`) with `resp.URL` instead of a session URL. No daemon IPC changes.

**Wails binding generation:** `wails generate module` (or dev server restart) auto-generates the updated `frontend/src/wailsjs/go/main/App.js` and `.d.ts` with `GetDashboardQRCode`.

**Frontend (SettingsTab.tsx):** Import `BrowserOpenURL`, `ClipboardSetText` from Wails runtime; import `GetDashboardQRCode` from generated bindings. Add three controls in the server-running section. The QR display can reuse the `QRModal` pattern inline or as a new `DashboardQRModal` component.

**QR modal approach:** The existing `QRModal` accepts `sessionId` and calls `GetSessionQRCode(sessionId)` internally. The simplest extension is a second variant `DashboardQRModal` that accepts no `sessionId` and calls `GetDashboardQRCode()`. Alternatively, generalize `QRModal` with an optional `fetchFn` prop — but that adds complexity. Prefer a separate focused component.

---

### Feature 4: Sidebar Icon Centering When Collapsed

**What it is:** When the sidebar is collapsed (48px wide), icons should be horizontally centered within the 48px column. Currently, `.sidebar__toggle` uses `justify-content: center` and is centered. `.sidebar__item` uses `display: flex; align-items: center; gap: 8px; padding: 8px; width: 100%` but does NOT have `justify-content: center` — so the icon aligns to the left edge of the padding box, with 8px left padding. The 20px icon at 8px left pad leaves 20px right of center: visually off-center.

**Root cause confirmed in `style.css` lines 188-207:** `.sidebar__item` has no `justify-content` declaration. `.sidebar__toggle` (line 169-182) has `justify-content: center` — that button looks correct. The item buttons are missing it.

**The fix — CSS modifier class (zero JSX changes):**
```css
.sidebar--collapsed .sidebar__item {
  justify-content: center;
  padding: 8px 0;
}
```

The `padding: 8px 0` removes horizontal padding when collapsed — otherwise 8px left+right padding plus 20px icon = 36px in a 48px column, centering within the remaining 12px instead of the full 48px. With `padding: 0` horizontally, `justify-content: center` distributes the full 48px correctly.

The `.sidebar--collapsed` class is already on the `<nav>` element (verified in `Sidebar.tsx` line 44). CSS descendant selector works without JSX changes.

---

## Data Flow Changes

### Terminal Padding/Theming Data Flow

```
localStorage ('terminal-padding', 'terminal-theme' keys)
    ↓ initial value in useState
App.tsx state: paddings{sessionId: number}, themes{sessionId: string}
    ↓ props to each TerminalPanel instance
TerminalPanel.tsx
    useEffect([sessionId]):
      term.open(containerRef.current)
      term.element.style.padding = padding + 'px'    <- applied once on creation
    useEffect([padding]):
      term.element.style.padding = padding + 'px'    <- applied on change
    useEffect([theme]):
      term.options.theme = THEMES[theme]             <- applied on change
    fitTerminal() reads getComputedStyle(term.element)
      padH = paddingLeft + paddingRight
      padV = paddingTop + paddingBottom              <- already subtracted
    cols/rows computed correctly with padding subtracted
```

### Dashboard QR / URL Controls Data Flow

```
SettingsTab.tsx (serverURL state, already populated from GetWebServerURL)
    "Open Dashboard" → BrowserOpenURL(serverURL)      [Wails runtime, sync]
    "Copy URL"       → ClipboardSetText(serverURL)    [Wails runtime, async]
    "QR" button      → setState: showDashboardQR=true
                           ↓
                   DashboardQRModal mounts
                   GetDashboardQRCode() called         [new Go binding]
                           ↓
                   app.go GetDashboardQRCode()
                   client.GetWebServerStatus() → resp.URL
                   qrcode.Encode(resp.URL, qrcode.Medium, 256) → PNG bytes
                   base64.StdEncoding.EncodeToString(png) → string returned
                           ↓
                   <img src="data:image/png;base64,…"> displayed in modal
```

## Component Inventory: New vs Modified

| Component/File | Status | Change Summary |
|----------------|--------|----------------|
| `frontend/src/lib/themes.ts` | **NEW** | xterm ITheme constants for all supported themes |
| `frontend/src/components/DashboardQRModal.tsx` | **NEW** | Modal for dashboard URL QR; calls `GetDashboardQRCode()` |
| `frontend/src/components/TerminalPanel.tsx` | **MODIFIED** | Add `theme: string` and `padding: number` props; add `useEffect([theme])` and `useEffect([padding])` effects |
| `frontend/src/components/SettingsTab.tsx` | **MODIFIED** | Add Appearance sub-tab (theme picker, padding slider); add open/copy/QR controls in web-server sub-tab |
| `frontend/src/App.tsx` | **MODIFIED** | Add `themes` and `paddings` state; pass new props to `TerminalPanel` |
| `frontend/src/style.css` | **MODIFIED** | Add `.sidebar--collapsed .sidebar__item { justify-content: center; padding: 8px 0 }` |
| `app.go` | **MODIFIED** | Add `GetDashboardQRCode()` method |
| `frontend/src/wailsjs/go/main/App.js` + `.d.ts` | **REGENERATED** | Auto-generated after new Go method |
| `frontend/src/components/StatusBar.tsx` | **MODIFIED** (optional) | If open-in-browser / copy buttons are wanted at session level (not just dashboard level) |

## Recommended Build Order

**Ordered by dependency chain and risk:**

1. **Sidebar icon centering (Feature 4)** — CSS-only, one rule, zero risk, zero dependencies. Proves the `.sidebar--collapsed` descendant selector approach before any code changes.

2. **Terminal padding (Feature 1)** — Pure frontend. The `fitTerminal` custom function already handles padding subtraction. Add `DEFAULT_PADDING` constant, `paddings` state in `App.tsx`, `padding` prop in `TerminalPanel`. Apply via `term.element.style.padding` in the mount effect. Validate that terminal fills correctly after resize with padding active. Add global default to `localStorage` in Appearance sub-tab (first version of new settings sub-tab).

3. **Terminal theming (Feature 2)** — Pure frontend, depends on Feature 1 only for the shared Appearance sub-tab UI. Create `lib/themes.ts`, add `theme` prop to `TerminalPanel`, add theme picker to Appearance sub-tab. Validate runtime theme switching without terminal destruction.

4. **Web server link improvements (Feature 3)** — The only feature touching Go backend. Do last so the binding layer is touched once and in isolation. Add `GetDashboardQRCode` to `app.go`, regenerate bindings, update `SettingsTab.tsx` with open/copy/QR controls, add `DashboardQRModal` component.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Applying Padding to the Container Div Instead of the xterm Element

**What people do:** Add `padding` to the `containerRef` outer div in TerminalPanel.
**Why it's wrong:** `fitTerminal` reads `window.getComputedStyle(term.element!)` — not the container div. Padding on the container shrinks available space but `fitTerminal` calculates parent width from the container (not the xterm element's own box), causing mismatched sizing.
**Do this instead:** Apply padding via `term.element.style.padding = padding + 'px'` after `term.open()`. The custom `fitTerminal` already subtracts `padH`/`padV` from its calculations.

### Anti-Pattern 2: Destroying and Recreating the Terminal Instance for Theme Changes

**What people do:** Tear down the `Terminal` instance and recreate it to apply a new theme.
**Why it's wrong:** Loses the scrollback buffer, closes the relay client WebSocket, and triggers the bounded rAF retry loop (20 attempts, ~333ms) on every theme change.
**Do this instead:** `term.options.theme = THEMES[theme]` — xterm.js applies theme changes live without recreation, identical to how `term.options.fontSize = fontSize` works for the existing font size feature.

### Anti-Pattern 3: Using `<a target="_blank">` for Dashboard URL in Wails WebView

**What people do:** Render `<a href={serverURL} target="_blank">` expecting it to open the system browser.
**Why it's wrong:** Wails WebView's handling of `target="_blank"` is platform-dependent and unreliable. Links may not open the OS default browser.
**Do this instead:** `BrowserOpenURL(serverURL)` from the Wails runtime. Already used for remote sessions in `App.tsx:412` and for GitHub releases in `WelcomeTab.tsx:65`.

### Anti-Pattern 4: Applying `justify-content: center` to All Sidebar Items Unconditionally

**What people do:** Add `justify-content: center` to `.sidebar__item` globally (not scoped to collapsed state).
**Why it's wrong:** In expanded state, icon and label are flex children — centering both together shifts the text away from the left-aligned nav pattern and looks wrong.
**Do this instead:** Scope the rule to `.sidebar--collapsed .sidebar__item`. The existing `.sidebar--collapsed` class is on the `<nav>` element, making descendant selectors the right mechanism.

## Integration Points Summary

### Wails Runtime Bindings (already available, no changes to bindings layer)

| Binding | Import Path | Used For in v1.12 |
|---------|-------------|-------------------|
| `BrowserOpenURL(url)` | `wailsjs/wailsjs/runtime/runtime` | Open dashboard in OS browser (new use in SettingsTab) |
| `ClipboardSetText(text)` | `wailsjs/wailsjs/runtime/runtime` | Copy server URL to clipboard |

### Go App Methods (existing — no changes)

| Method | Used For |
|--------|----------|
| `GetWebServerURL()` | Dashboard URL source — already called in SettingsTab |
| `GetSessionQRCode(sessionId)` | Session QR (unchanged) |
| `IsWebServerRunning()` | Gate for showing link controls |

### Go App Methods (new)

| Method | Location | Notes |
|--------|----------|-------|
| `GetDashboardQRCode()` | `app.go` | Calls `qrcode.Encode(resp.URL, ...)` — uses already-imported `skip2/go-qrcode` and `base64` |

### Internal Module Boundaries (no changes needed)

| Boundary | Status |
|----------|--------|
| `App.go` → `DaemonClient` → daemon HTTP | `GetDashboardQRCode` reuses `client.GetWebServerStatus()` — no new daemon API endpoints |
| `TerminalPanel` → `fitTerminal` | Padding flows through existing computed-style reading — no changes to `fitTerminal` |
| `SettingsTab` → Wails runtime | `BrowserOpenURL` import added; `ClipboardSetText` added |

---

*Architecture research for: AgentHub v1.12 UI/UX Polish milestone*
*Researched: 2026-04-10*
