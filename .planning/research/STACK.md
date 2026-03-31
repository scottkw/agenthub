# Stack Research

**Domain:** AgentHub v1.7 — Daemon UX & Branding (system tray, remote session indicators, app icons, splash screen)
**Researched:** 2026-03-31
**Confidence:** MEDIUM-HIGH (system tray library choice has one CRITICAL constraint; icon tooling HIGH confidence)

---

## Context: What's New vs What's Unchanged

v1.7 adds four capability areas. Three require new Go dependencies. One (splash screen) is zero-dependency.

| Area | New Dependency? | Notes |
|------|-----------------|-------|
| System tray icon + menu | YES — CGO required | Most complex; platform-specific constraints |
| Remote session status bar | NO | Frontend-only React component; xterm.js WebSocket already in place |
| App icons (.icns, .ico, multi-PNG) | YES — build-time CLI tools | Generation scripts only, not runtime deps |
| Splash screen | NO | CSS overlay in React, hidden on `OnDomReady` |

---

## Recommended Stack

### New Go Runtime Dependency: System Tray

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| `fyne.io/systray` | v1.12.0 (Dec 2025) | Cross-platform system tray icon + right-click menu | Actively maintained fork of getlantern/systray; removed GTK dependency (uses DBus on Linux); no AppDelegate conflict with Wails on macOS when used correctly via goroutine; supports macOS, Windows, Linux, BSD |

**Critical integration constraint for macOS:** The standard `fyne.io/systray` API calls `systray.Run(onReady, onExit)` which tries to lock the main OS thread — the same thread Wails' WebView owns. Direct embedding causes an AppDelegate duplicate symbol linker error.

**Confirmed working pattern (from Wails community discussion #4514):** Run the tray in a subprocess that communicates with the daemon over IPC (Unix socket / named pipe). The daemon already owns an IPC socket; the tray process connects as another client. This matches the existing architecture — the tray is just another DaemonClient consumer.

Alternative for macOS-only: native NSStatusBar via cgo (no third-party library; avoids all conflicts). Requires platform-specific `.m` files and build constraints. Recommended only if cross-platform tray is deprioritized.

**Linux note:** `fyne.io/systray` uses DBus / SystemNotifier/AppIndicator spec. Requires `libdbus-1-dev` on the build host. Older desktop environments (classic GNOME, some i3 setups) need `snixembed` proxy to display the icon. Modern GNOME 3+, KDE, XFCE work natively.

**Build requirement:** `CGO_ENABLED=1` (already required by Wails itself; no change needed).

### New Build-Time Tools: Icon Generation

These are one-time generation scripts (run during asset prep, not at runtime). Install as dev tools; not added to `go.mod` as runtime deps.

| Tool | Version | Purpose | Why |
|------|---------|---------|-----|
| `github.com/jackmordaunt/icns/v3` (CLI: `icnsify`) | v3 / v2.2.7 (Nov 2023) | PNG → `.icns` for macOS | Pure Go, no ImageMagick, works on Windows/Linux/macOS build hosts. Takes any PNG, outputs multi-size `.icns`. Supports piping. |
| `github.com/sergeymakinen/go-ico` | latest (Jan 2024) | PNG → `.ico` for Windows | Pure Go ICO encoder/decoder. Multi-size ICO (256, 128, 64, 48, 32, 16) from a single source PNG. |
| `golang.org/x/image/draw` | stdlib (in go.mod) | Image resizing for multi-size PNGs | Standard library; `draw.BiLinear.Scale()` handles high-quality downsample. No additional dependency. |
| macOS native `iconutil` | macOS system tool | Alternative PNG → `.icns` on macOS only | Built into macOS. Use as fallback if `icnsify` unavailable. Requires an `.iconset/` directory structure. Not cross-platform. |

**Existing state:** `build/appicon.png` is 256x256 (placeholder geometric art). `build/windows/icon.ico` exists (6 sizes, Wails-generated from appicon). `build/darwin/` has no `.icns` — this is the gap.

**Wails v2 icon behavior:**
- `build/appicon.png` → embedded in binary as the app icon (dock/taskbar)
- `build/windows/icon.ico` → Windows .exe resource; Wails auto-generates from appicon.png if missing, sizes: 256, 128, 64, 48, 32, 16
- `build/darwin/*.icns` → macOS .app bundle icon; Wails uses the `.icns` if present; falls back to `appicon.png`. **Must be pre-generated** — Wails v2 does not auto-generate `.icns` from PNG the way it does for Windows.

**Recommended icon pipeline:**
```
docs/agenthub-title-logo.png  (source — full branding asset)
  → strip to square icon crop (manual Figma/Photoshop step, or imagemagick -gravity center -extent 1:1)
  → build/appicon.png          (1024x1024 PNG, transparent background recommended)
  → build/darwin/AppIcon.icns  (via icnsify or iconutil)
  → build/windows/icon.ico     (via go-ico or Wails auto-gen)
  → build/linux/icons/         (16, 32, 48, 64, 128, 256 PNGs via x/image/draw)
```

### Frontend: Remote Session Status Bar

No new npm packages. The status bar is a React component consuming existing WebSocket state.

| Technology | Already In Project | Usage |
|------------|-------------------|-------|
| React state / props | Yes — React in frontend | Session connection state |
| xterm.js WebSocket relay | Yes — RelayClient.ts | `onOpen`/`onClose`/`latency` events already available |
| CSS flexbox | Yes | Status bar layout (same pattern as existing per-tab StatusBar) |

The web dashboard already has a status bar. The new component targets the standalone web terminal page (the page served to remote browsers and `agenthub attach` CLI sessions).

### Frontend: Splash Screen

No new dependencies. Wails v2 provides `OnDomReady` callback and CSS overlay pattern.

| Technology | How Used |
|------------|---------|
| Wails `OnDomReady` in `main.go` | Triggers a Wails event (`runtime.EventsEmit`) when DOM is fully ready |
| React `useEffect` + state | Hides splash overlay after event received |
| CSS `position: fixed` overlay | Covers WebView during load; fades out on hide |

**Pattern:** Splash overlay is a `<div>` with `position:fixed; z-index:9999; width:100vw; height:100vh` containing the title logo image. On `OnDomReady` event, set `display:none` or trigger a CSS fade-out transition. The logo image (`docs/agenthub-title-logo.png`) is embedded in the Wails frontend assets.

Wails `OnDomReady` timing note: fires when all assets in `index.html` are loaded (equivalent to `body.onload`). On first load this includes the time to parse/render React — typically 100-400ms on target hardware. Reliable for splash-hide triggering.

---

## Installation

```bash
# System tray runtime dep (add to go.mod)
go get fyne.io/systray@v1.12.0

# Icon generation tools (install as dev tools, not go.mod deps)
go install github.com/jackmordaunt/icns/v3/cmd/icnsify@latest

# go-ico is used as a library in a build script, not a standalone CLI
# Add to a build/gen_icons.go script:
go get github.com/sergeymakinen/go-ico@latest

# golang.org/x/image — already in project via indirect deps (Wails/Tailscale pull it in)
# Verify: grep "golang.org/x/image" go.sum
```

**Linux build host requirement for tray CGO:**
```bash
# Ubuntu/Debian
sudo apt-get install libdbus-1-dev

# Already required: libgtk-3-dev, libwebkit2gtk-4.0-dev (Wails itself needs these)
```

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `fyne.io/systray` (subprocess/IPC pattern) | `github.com/energye/systray` | energye is another active fork with similar API; use if fyne.io has a breaking issue. Same AppDelegate constraint applies. |
| `fyne.io/systray` subprocess | `github.com/ra1phdd/systray-on-wails` | Pre-release (v0.0.0, Nov 2024), limited adoption. Use only if Wails main-thread integration is confirmed working on all 3 platforms. |
| `fyne.io/systray` subprocess | Native NSStatusBar via cgo | Eliminates AppDelegate conflict entirely on macOS. Requires platform-specific Obj-C glue files. Choose this if dropping Linux tray or building platform-specific tray modules. PROJECT.md key decisions already note this pattern was attempted. |
| `fyne.io/systray` subprocess | Wails v3 built-in tray | Wails v3 has native systray support. Currently v3-alpha, not production-ready. Choose when v3 stabilizes (no ETA as of early 2026). |
| `icnsify` for .icns generation | macOS `iconutil` CLI | Use `iconutil` when building exclusively on macOS; it's a zero-install option. Not usable in CI (Ubuntu runners). |
| `github.com/sergeymakinen/go-ico` | Wails auto-gen | Wails auto-generates `icon.ico` from `appicon.png` if not present. Use Wails auto-gen if the 6-size default suffices. Use go-ico for custom size control or transparent backgrounds. |
| CSS splash overlay | Wails `SplashBackgroundColour` option | Wails has a `BackgroundColour` option (not a splash screen). There is no official Wails v2 splash screen API — CSS overlay is the community-standard approach. |

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `getlantern/systray` | Original library; requires GTK on Linux and causes AppDelegate duplicate symbol on macOS with Wails | `fyne.io/systray` (fork that removed GTK) |
| `github.com/energye/energy` | Full Chromium-based desktop framework; enormous dependency for just a tray icon | `fyne.io/systray` standalone |
| Wails v3 upgrade | v3 is alpha; Wails v2 is the production-validated stack in this project. Upgrading mid-milestone introduces unquantified risk | Stay on Wails v2.10.2; revisit at next major milestone |
| `nfnt/resize` for icon resizing | Unmaintained (last commit 2018). Works but no security updates | `golang.org/x/image/draw` — standard library, in project already via indirect deps |
| ImageMagick (`convert` CLI) | External binary dependency; not reliably present on all build hosts or CI runners | Pure Go tools (`icnsify`, `go-ico`) |
| `disintegration/imaging` | Last tagged release was 2019 despite later commits; no active maintainer response to issues | `golang.org/x/image/draw` for resizing; `icnsify` for .icns |
| React splash screen library (npm) | A CSS overlay + CSS transition is 10 lines. No library warranted. | Inline CSS + React state |

---

## Stack Patterns by Variant

**If system tray runs in-process (macOS only, NSStatusBar cgo):**
- Create `internal/tray/tray_darwin.go` with `//go:build darwin` and cgo NSStatusBar calls
- Create stub `internal/tray/tray_linux.go` and `tray_windows.go` with build constraints
- No external tray library needed
- Because: NSStatusBar doesn't conflict with Wails' AppDelegate (it hooks into the existing NSApp, doesn't create its own)

**If system tray runs as subprocess (cross-platform, fyne.io/systray):**
- Create `cmd/agenthub-tray/main.go` — separate binary that calls `systray.Run()`
- Tray binary communicates with daemon via existing Unix socket / named pipe
- Tray binary launched by daemon on startup, tracked as a managed child process
- Bundled in the same `.app` bundle / distribution package as the main binary
- Because: avoids main-thread ownership conflict; matches existing daemon-as-subprocess pattern

**If splash screen needs to hide before React hydration:**
- Embed splash as a static `<div>` in `index.html` (not in React), using `window.hideSplash = () => { ... }`
- Call from Wails `OnDomReady` via `runtime.WindowExecJS(ctx, "window.hideSplash()")`
- Because: React render itself takes ~50-100ms after DOM ready; if splash is a React component it may flash before rendering

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `fyne.io/systray v1.12.0` | Wails v2.10.2 | Compatible when run in subprocess (separate binary). In-process on macOS causes AppDelegate linker error. |
| `fyne.io/systray v1.12.0` | Go 1.26.1 | No known incompatibilities |
| `github.com/jackmordaunt/icns/v3` | macOS, Linux, Windows build hosts | Pure Go; no host OS requirement |
| `github.com/sergeymakinen/go-ico` | macOS, Linux, Windows build hosts | Pure Go; no host OS requirement |
| `golang.org/x/image` | Already in project (indirect dep via Wails/Tailscale) | Verify `golang.org/x/image` is reachable; if not explicit in go.mod, add it |

---

## Sources

- [fyne.io/systray pkg.go.dev](https://pkg.go.dev/fyne.io/systray) — v1.12.0 version, Linux DBus requirement, CGO requirement. MEDIUM confidence (official package page).
- [fyne-io/systray GitHub](https://github.com/fyne-io/systray) — Fork origin (from getlantern/systray), GTK removal confirmed. MEDIUM confidence.
- [Wails discussion #4514 — SysTray](https://github.com/wailsapp/wails/discussions/4514) — Confirmed: no built-in Wails v2 systray; subprocess/IPC as community-validated workaround; NSStatusBar cgo as macOS alternative. HIGH confidence (Wails maintainer + community).
- [Wails issue #1010 — macOS tray](https://github.com/wailsapp/wails/issues/1010) — Main-thread conflict root cause: "both the systray library and wails use the main thread, so it is difficult to use them together." HIGH confidence.
- [jackmordaunt/icns GitHub](https://github.com/JackMordaunt/icns) — Pure Go, v2.2.7, `icnsify` CLI, cross-platform. HIGH confidence.
- [sergeymakinen/go-ico pkg.go.dev](https://pkg.go.dev/github.com/sergeymakinen/go-ico) — Pure Go ICO encoder, Jan 2024. MEDIUM confidence.
- [Wails project config docs](https://wails.io/docs/reference/project-config/) — Icon file locations and build behavior confirmed via search. MEDIUM confidence (403 on direct fetch; cross-referenced with issue #1431).
- [Wails issue #1431](https://github.com/wailsapp/wails/issues/1431) — Windows icon.ico auto-generated from appicon.png confirmed. MEDIUM confidence.
- `/Users/ken/dev/agenthub/build/` — Direct inspection: appicon.png is 256x256, icon.ico exists (6 sizes), no .icns in darwin/. HIGH confidence (direct file read).
- `/Users/ken/dev/agenthub/go.mod` — Current dependencies: Wails v2.10.2, Go 1.26.1; fyne.io/systray not yet present. HIGH confidence (direct file read).
- `/Users/ken/dev/agenthub/.planning/PROJECT.md` — Key Decision "Native macOS cgo NSStatusBar for tray: fyne.io/systray conflicts with Wails AppDelegate (duplicate symbol)" — confirms the constraint was already encountered. HIGH confidence.

---
*Stack research for: AgentHub v1.7 — Daemon UX & Branding*
*Researched: 2026-03-31*
