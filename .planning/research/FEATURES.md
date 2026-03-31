# Feature Research

**Domain:** Desktop daemon manager with system tray, remote session indicators, and app branding
**Researched:** 2026-03-31 (v1.7 Daemon UX & Branding)
**Confidence:** MEDIUM-HIGH (platform behavior verified via official docs/GitHub issues; Wails v2 tray limitations from open issues)

---

## v1.7 Milestone: Daemon UX & Branding

### Scope

This section covers only what is NEW in v1.7. The existing app ships: tabbed terminal sessions,
background daemon with Unix socket IPC, 13 CLI commands, web serving via Tailscale TLS, per-tab
status bar, settings modal, session naming, and cross-platform builds. Research focus: system
tray icon behavior, daemon mini management window, remote session status bars, app icons, and
splash screen.

Prior milestone research (v1.2 through v1.6) preserved below.

---

## Table Stakes (Users Expect These)

Features users assume exist in a tray-resident background daemon app. Missing these = product
feels broken or unprofessional.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Tray icon appears in system tray (not dock/taskbar) | Daemon apps live in the tray, not in the taskbar/dock — Docker Desktop, 1Password, Tailscale all do this; presence in dock signals "foreground app" not "background service" | MEDIUM | Wails v2 lacks native tray support; kardianos/service already manages daemon lifecycle. Tray icon must run in a separate goroutine. fyne.io/systray conflicts with Wails AppDelegate (existing KEY DECISION). NSStatusBar CGO path is the current approach but needs Linux/Windows stubs. Consider `getlantern/systray` or `fyne.io/systray` with `RunWithExternalLoop` for non-Wails tray process. |
| Right-click tray menu with Open / Quit | Every tray app has right-click for context menu; left-click convention varies by platform but right-click for menu is universal | LOW | Standard items: "Open AgentHub", "Sessions" submenu or count, "Quit". Tooltip on hover with session count is a bonus. macOS: right-click = show menu (left-click behavior is platform-defined). Windows/Linux: right-click always. |
| Tray icon reflects daemon state | Static icon is acceptable minimum; animated or color-changed icon for "busy" or "error" state is common in daemon tools | MEDIUM | At minimum: one icon for running, one for error/disconnected. macOS tray icons should be template images (PDF or PNG with alpha only) so system applies correct tint for light/dark menubar. Windows/Linux can use color icons. |
| "Open" tray action shows the GUI window | Users expect left-click or "Open" menu item to bring the main window to front; if window is closed it should open | LOW | Wails `runtime.WindowShow()` + `runtime.WindowFocus()`. If daemon is running as service (no GUI), this must launch the GUI binary. Two cases: (1) GUI is running but hidden — show it; (2) GUI is not running — spawn it. |
| "Quit" stops the daemon cleanly | Users expect Quit in tray to fully exit — not just close the window | LOW | `os.Exit` or signal the daemon goroutine to shut down. Must call kardianos/service Stop path or send SIGTERM to daemon PID. GUI close button should NOT quit daemon — only tray Quit should. |
| App has proper platform icons (icns/ico/png) | Wails splash and Dock/Taskbar icons must be set; missing icon = unprofessional | MEDIUM | macOS: `.icns` with sizes 16,32,64,128,256,512,1024 (and @2x variants). Windows: `.ico` with 16,32,48,64,128,256. Linux: multiple `.png` sizes (16,32,48,64,128,256,512). Source: single 1024x1024 PNG → generate all. The existing `docs/agenthub-title-logo.png` provides the wordmark — a standalone icon (logomark only) is needed for small sizes. |
| Remote session status bar in web terminal | Web users have no tray or settings access — they need to see connection state, session name, and web-serving URL in the terminal view | MEDIUM | Existing per-tab status bar in the GUI provides the pattern. Web terminal view needs equivalent: session name, agent type, connection status (connected/disconnected/reconnecting), and optionally the host machine name. This is the xterm.js web view served via `web/`. |
| Remote session indicator in CLI attach | `agenthub attach <id>` is a full PTY proxy; users need visual confirmation they are in a remote session (not a local one) and how to detach | LOW | A one-line banner printed before the PTY stream begins: "Connected to session <name> on <host> — press Ctrl-\\ to detach". Already partially implemented (detach key documented in PROJECT.md). Make banner consistent and clear. |

---

## Differentiators (Competitive Advantage)

Features that set AgentHub apart from generic tray utilities. Should align with Core Value: "One
app to launch, manage, and share AI coding terminal sessions."

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Daemon mini management window from tray | "Manage sessions" menu item opens a compact window showing session list with start/stop/kill and web-serve toggle — no need to open full GUI for basic ops | HIGH | This is a second Wails window or a separate lightweight webview. Session list already served via daemon HTTP API. Mini window needs its own React page or a simple HTML page hitting daemon API. Alternative: open the existing full GUI to the sessions tab (simpler, less novel). |
| Session count badge on tray icon | Tray icon shows number of active sessions as a badge/overlay — common in chat apps (unread count) but novel for terminal managers | MEDIUM | Platform support varies. macOS: NSStatusItem can overlay text. Windows: overlay icon on taskbar icon (different from tray). Linux: not universally supported. Consider tooltip text as fallback: "AgentHub — 3 sessions active". |
| Tray menu lists active sessions by name | Each running session appears as a named menu item — click to show/focus that session in the GUI | MEDIUM | Dynamic menu items require the tray library to support runtime menu updates. fyne.io/systray supports `AddMenuItem` / menu item visibility toggle. Session names come from daemon API. Limit to 5-10 items to avoid menu overflow. |
| Splash screen using title logo on first load | Masks WebView2/WebKit initialization latency with branded experience; sets tone for app quality | MEDIUM | Wails v2 does not have a built-in splash screen API. Common pattern: render splash as React route shown during `loading` state before `domReady` Wails event fires. Can use `wails:ready` event or `useEffect` on mount to transition out. Duration: show during Go initialization (~0.5-1s) then fade out. Do NOT use a native pre-webview splash (complex, platform-specific). |
| Web session status bar shows machine/host name | Remote users know WHICH machine they are accessing — important when user has multiple tailnet nodes running AgentHub | LOW | Daemon already knows machine hostname (`os.Hostname()`). Include in session metadata served via HTTP API. Web terminal view reads this and displays "AgentHub on <hostname>" or similar. |
| Dark/light adaptive tray icon | macOS menu bar uses template icons (monochrome with alpha) that auto-adapt to light/dark mode; Windows 11 also supports this | LOW | macOS: provide two PNG variants named `tray_icon_template.png` (or use PDF); mark as template image via NSImage. Windows: provide color icon (system doesn't auto-adapt). Linux: provide standard color icon. This is a polish detail but visually noticeable. |

---

## Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Tray icon on daemon binary (separate process) | "Tray should be the daemon, not the GUI" | kardianos/service-managed daemons run as background services with no UI access; service process cannot create GUI elements (macOS launchd daemons are not allowed NSStatusItem). Additionally, tray library + kardianos/service + Wails all fight over the main thread on macOS. | Tray icon runs in the GUI binary (Wails app). GUI binary handles tray; daemon binary handles sessions. This is how Docker Desktop works (separate Engine daemon, Desktop GUI owns the tray). |
| Animated/spinning tray icon while sessions are active | "Shows the app is doing something" | Animation via tray icon on macOS requires updating the icon at 10-30fps via NSStatusItem, burning CPU for a cosmetic effect; on Windows, animated tray icons are deprecated in Win10+. | Use static state icons (running = solid, error = X overlay). Tooltip text can say "3 sessions active". |
| Full session terminal in mini management window | "Manage and view sessions from tray popup" | Embedding xterm.js in a popup window is a full re-implementation of the main GUI in a small panel; double the maintenance surface. xterm.js requires canvas rendering which is heavyweight for a "quick glance" popup. | Mini window shows session list with metadata (name, status, URL, agent). Double-click to open that session in the full GUI. |
| Splash screen with fixed minimum duration | "Brand exposure time" | Splash screens with artificial delays frustrate users; if the app loads fast, a forced 2-3s splash is friction. App startup is currently fast (~500ms). | Show splash only while actually loading (until Wails domReady + daemon connection confirmed). Immediately dismiss when ready. |
| Tray icon replacement for full GUI | "Minimize to tray, live there forever" | Tray-only interaction limits discoverability; new users won't find features. | Tray provides quick ops (quit, show sessions count, open GUI). Full session management always in the main GUI window. |
| Per-platform native tray icon implementations | "Best platform integration" | Three separate codebase paths (NSStatusBar CGO for macOS, Windows API for Win, AppIndicator for Linux) is significant ongoing maintenance. The existing NSStatusBar CGO approach already has this problem — KEY DECISION flags it as "Revisit". | Single Go tray library (fyne.io/systray or getlantern/systray) abstracts platform differences behind one API. Accept small UX compromises for single codebase. |

---

## Feature Dependencies

```
[App Icons (icns/ico/png)]
    └──required by──> [Wails Build System] (wails.json appicon path)
    └──required by──> [Tray Icon] (tray icon is a smaller crop of app icon)
    └──required by──> [Splash Screen] (uses title logo, not app icon)

[Daemon HTTP API — session list endpoint]
    └──required by──> [Tray Session List Menu items]
    └──required by──> [Mini Management Window]
    └──required by──> [Tray Session Count Badge]

[Existing CLI attach (agenthub attach)]
    └──enhanced by──> [CLI Attach Session Banner]

[Existing Web Terminal View (web/)]
    └──enhanced by──> [Web Session Status Bar]
    └──enhanced by──> [Machine/Host Name in Web Status]

[Tray Icon (any library)]
    └──requires──> [Dock/Taskbar Hide on macOS (LSUIElement or activation policy)]
    └──requires──> [GUI binary stays running after window close (no window-all-closed quit)]
```

### Dependency Notes

- **App icons required by tray icon:** The tray icon is derived from the app icon; generate both from the same 1024x1024 source PNG. Tray icons are typically 16x16 or 22x22 (macOS) — need a standalone logomark (not wordmark) that reads at small sizes.
- **Daemon HTTP API required by tray session list:** The daemon already exposes a REST API over Unix socket for the GUI. A mini management window or tray menu listing sessions both poll this same API. No new protocol work needed — just HTTP calls.
- **LSUIElement required by dock-free tray:** Setting LSUIElement=1 in macOS Info.plist hides both dock icon AND app menu bar. This is the correct behavior for a background service app. Wails v2 has a known issue (GitHub #3700) where `StartHidden: true` still shows dock icon — LSUIElement override in Info.plist is the correct fix.
- **GUI binary must not quit on window close:** Standard Wails behavior quits the app when the last window closes. For tray apps this must be overridden — close hides the window, tray Quit actually exits. Wails `runtime.EventsOn("quit")` or `OnBeforeClose` callback handles this.
- **Web status bar independent of GUI:** Web terminal runs in browser, not in Wails WebView. The status bar for web sessions is pure HTML/JS in the `web/` served static assets — no Wails runtime available. Must use WebSocket or polling for live status.

---

## MVP Definition for v1.7

### Launch With (v1.7)

Minimum viable to call the milestone complete.

- [ ] App icon set (icns, ico, png variants) — required for professional distribution, unblocks any icon-related work
- [ ] Tray icon with right-click menu: "Open AgentHub", "Quit" — minimum tray presence
- [ ] Tray icon reflects daemon running vs error state — two icons minimum
- [ ] GUI window hides (not quits) on close — tray app lifecycle requirement
- [ ] macOS dock icon hidden (LSUIElement or activation policy) — daemon should not live in dock
- [ ] CLI attach session banner — low complexity, high clarity value for remote users
- [ ] Web terminal session status bar — shows session name, agent, connection state, host

### Add After Core Tray Works (v1.7.x)

Features to add once the basic tray lifecycle is stable.

- [ ] Splash screen — add after icon assets exist (depends on icon work)
- [ ] Tray session count in tooltip — depends on tray library event loop working
- [ ] Tray menu lists active session names — depends on stable tray menu dynamic updates

### Future Consideration (v2+)

Defer until core is validated.

- [ ] Mini management window from tray — high complexity, explore after tray basics ship
- [ ] Session count badge overlay on tray icon — platform support inconsistent, low priority
- [ ] Dark/light adaptive tray icon (template PNG) — polish, after basic tray works

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| App icon set (all platforms) | HIGH | LOW | P1 |
| Tray icon + right-click Open/Quit | HIGH | MEDIUM | P1 |
| GUI hides on window close (not quits) | HIGH | LOW | P1 |
| macOS dock icon hidden | HIGH | LOW | P1 |
| Web terminal session status bar | HIGH | MEDIUM | P1 |
| CLI attach session banner | MEDIUM | LOW | P1 |
| Daemon state icon (running/error) | MEDIUM | LOW | P1 |
| Splash screen (title logo) | MEDIUM | MEDIUM | P2 |
| Tray tooltip with session count | MEDIUM | LOW | P2 |
| Tray menu lists session names | MEDIUM | MEDIUM | P2 |
| Session count badge on tray | LOW | HIGH | P3 |
| Mini management window from tray | HIGH | HIGH | P3 |
| Dark/light adaptive tray icon | LOW | LOW | P2 |

---

## Platform-Specific Notes

### macOS

- Tray icon lives in the menu bar (top-right area), not in a "system tray"
- Template icons: PNG with transparency only, named with "Template" suffix or marked via NSImage API; system applies light/dark tint automatically
- LSUIElement=1 in Info.plist: hides dock icon AND disables the default app menu bar (correct for daemon apps; Wails app menu still accessible via tray)
- fyne.io/systray calls `NSStatusItem` via CGo; this conflicts with Wails's own NSApplicationDelegate if both run in the same process — known issue documented in existing KEY DECISION
- Solution: run fyne.io/systray from a separate goroutine with `RunWithExternalLoop`; OR use `ra1phdd/systray-on-wails` which is explicitly designed for Wails coexistence
- Wails v2 GitHub issue #3700: dock icon cannot be hidden via `wails.json` alone; must set `LSUIElement` in Info.plist

### Windows

- System tray area is the notification area (bottom-right)
- `.ico` format required; include 16, 32, 48, 64, 128, 256px sizes in single file
- Left-click on tray icon: show/hide main window (conventional)
- Right-click: context menu
- Color icons (not monochrome templates)
- Windows 11: tray icons can be hidden in overflow; tooltip is important for discoverability

### Linux

- GNOME 40+: native system tray removed; requires GNOME Shell extension (AppIndicator/KStatusNotifierItem)
- KDE Plasma: full tray support via StatusNotifierItem
- Elementary OS, Xfce, MATE: traditional tray support
- fyne.io/systray uses AppIndicator on GTK desktops and StatusNotifierItem spec on modern DEs
- Multiple PNG sizes required (16, 22, 32, 48, 64, 128, 256)
- Desktop entry file (`.desktop`) required for proper integration

---

## Icon Asset Requirements

### Source Material

- Existing: `docs/agenthub-title-logo.png` — horizontal wordmark (AgentHub text + icon mark at left)
- Needed: Square icon mark (logomark only, without wordmark text) — required for small sizes where wordmark is unreadable
- The existing logo shows a geometric/abstract mark to the left of "AgentHub" text — extract or redraw as standalone square icon

### Required Output Files

| File | Format | Sizes | Platform |
|------|--------|-------|----------|
| `appicon.icns` | ICNS | 16,32,64,128,256,512,1024 (@1x + @2x) | macOS |
| `appicon.ico` | ICO | 16,32,48,64,128,256 | Windows |
| `appicon.png` | PNG | 256x256 (Wails default) | Linux / Wails |
| `tray_icon_template.png` | PNG (alpha only) | 22x22 @1x, 44x44 @2x | macOS tray |
| `tray_icon.ico` / `tray_icon.png` | ICO/PNG | 16x16, 32x32 | Windows/Linux tray |

### Generation Process

1. Start with 1024x1024 square logomark PNG (transparent background)
2. macOS: `sips` + `iconutil` → `.iconset/` directory → `iconutil -c icns`
3. Windows: `convert` (ImageMagick) or online tool → multi-size `.ico`
4. Linux: Copy PNG at required sizes to `/usr/share/icons/` paths in `.desktop` integration
5. Wails: set `appicon.png` path in `wails.json` `info` section

---

## Web Terminal Status Bar Requirements

### What to Show

The web terminal status bar appears at the top or bottom of the xterm.js terminal in the browser-served view. It is separate from the Wails GUI status bar.

| Field | Source | Why |
|-------|--------|-----|
| Session name | Daemon session metadata | Identifies which session this is |
| Agent type (Claude Code / Gemini / etc.) | Daemon session metadata | Disambiguates agent behavior |
| Connection status (Connected / Reconnecting / Disconnected) | WebSocket state | Shows liveness, especially critical on reconnect |
| Host machine name | `os.Hostname()` in daemon | Remote users need to know which machine they're on |
| Web URL (optional) | Daemon serve URL | Quick copy for sharing |

### Implementation Pattern

- Status bar is a thin HTML div above or below the xterm.js terminal in the served web page
- WebSocket already carries session output; add a control channel or heartbeat ping to detect disconnection
- Connection state changes: WebSocket `onopen`, `onclose`, `onerror` events
- Status bar updates via JavaScript DOM manipulation (no framework required — served as static HTML/JS)
- Session metadata fetched once on page load from daemon HTTP API (`GET /api/sessions/<id>`)
- Host name included in session metadata response from daemon

---

## CLI Attach Session Banner Requirements

### Current State

`agenthub attach <id>` does: raw I/O PTY proxy, detach key (Ctrl-\\), resize propagation. Banner behavior is undocumented or absent.

### Required Banner

Print to stderr (not stdout, to avoid polluting session output) before PTY stream begins:

```
[AgentHub] Connected to "<session-name>" (<agent>) on <hostname>
[AgentHub] Press Ctrl-\ to detach
```

### Implementation

- Already have session metadata from daemon API at attach time
- `os.Hostname()` available in CLI process
- Print banner to `os.Stderr` before entering raw mode
- On detach, print `[AgentHub] Detached.` to stderr

---

## Splash Screen Requirements

### Purpose

Masks the 200-800ms WebKit/WebView2 initialization period. Provides brand exposure. Sets quality
tone on first launch.

### Design Pattern (Wails v2 approach)

Wails v2 has no native splash API. Standard approach:

1. React renders a full-screen splash component as the initial `App` state
2. Splash shows the title logo (`docs/agenthub-title-logo.png`) centered on a solid background
3. Wails `runtime.EventsEmit("app:ready")` fired from Go after daemon connection established
4. React `useEffect` + Wails `EventsOn("app:ready")` transitions splash → main UI
5. Fade-out animation (200ms CSS transition) smooths the handoff
6. No artificial minimum duration — dismiss as soon as app is truly ready

### Timing Targets

- Show: immediately on WebView load (before daemon connection)
- Hide: when daemon IPC confirmed + initial session list loaded
- Maximum: 3 seconds (timeout fallback to avoid infinite splash on daemon failure)
- If daemon fails: transition to existing error banner (not infinite splash)

---

## Competitor / Reference App Analysis

| Feature | Docker Desktop | 1Password | Tailscale | Our Approach |
|---------|---------------|-----------|-----------|--------------|
| Tray icon | Yes, Docker whale, color-coded | Yes, lock icon, color-coded | Yes, Tailscale logo | Icon from logomark, 2 states (running/error) |
| Dock icon hidden | No (has GUI mode too) | No | Yes (macOS: menu bar only mode) | Yes — daemon should be menu-bar only |
| Tray menu | Dashboard, Settings, Restart, Quit | Lock/Unlock, Open, Accounts, Quit | Connect/Disconnect, Admin, Quit | Open GUI, session list (names), Quit |
| Session list in tray | Container list (long) | No | Devices | Active sessions, capped at 5-10 items |
| Status in web view | N/A | N/A | N/A | Session name, agent, connection, host |
| Splash screen | Yes (Docker whale animation) | Yes (branded) | No | Title logo, dismiss on ready |

---

## Sources

- fyne.io/systray conflicts with Wails: existing KEY DECISION in PROJECT.md ("Native macOS cgo NSStatusBar for tray")
- Wails v2 dock icon issue: https://github.com/wailsapp/wails/issues/3700
- Wails v2 tray menu request: https://github.com/wailsapp/wails/issues/1010
- systray-on-wails (Wails-compatible): https://pkg.go.dev/github.com/ra1phdd/systray-on-wails
- fyne.io/systray (cross-platform, RunWithExternalLoop): https://pkg.go.dev/fyne.io/systray
- macOS ICNS format: https://en.wikipedia.org/wiki/Apple_Icon_Image_format
- macOS icon sizes guide (2025): https://appicongenerator.org/app-icon-sizes-guide
- Icon sizes across platforms: https://blog.icons8.com/articles/choosing-the-right-size-and-format-for-icons/
- LSUIElement for menu-bar-only apps: fyne-io/systray README (mentions LSUIElement in Info.plist)
- Splash screen best practices: https://htmlburger.com/blog/splash-screen/ (duration: <2s, dismiss when ready)
- Wails v3 tray (alpha): https://v3alpha.wails.io/features/menus/systray/ (not used — staying on v2)
- AgentHub future features notes: docs/future-agenthub-features.txt

---

## Prior Milestone Research (v1.0 – v1.6)

The sections below preserve feature research from earlier milestones for reference.
See git history for the full content of each milestone's FEATURES.md.

---

*Feature research for: v1.7 Daemon UX & Branding — system tray, remote session indicators, app icons, splash screen*
*Researched: 2026-03-31*
