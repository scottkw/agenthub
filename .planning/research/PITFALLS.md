# Pitfalls Research

**Domain:** Desktop app — daemon tray management, remote session indicators, app branding/icons (AgentHub v1.7)
**Researched:** 2026-03-31
**Confidence:** HIGH — codebase read directly for framing protocol, relay server, and status bar; Wails issues verified from GitHub; icon requirements verified from Apple/Microsoft official docs and community post-mortems

---

## Critical Pitfalls

### Pitfall 1: fyne.io/systray (and energye/systray) Produces Duplicate Objective-C Symbols With Wails AppDelegate

**What goes wrong:**
Any Go systray library that contains its own Objective-C `AppDelegate` class — including `fyne.io/systray` and `github.com/energye/systray` — will fail to link on macOS with a fatal error:

```
duplicate symbol '_OBJC_CLASS_$_AppDelegate'
duplicate symbol '_OBJC_METACLASS_$_AppDelegate'
```

The project already hit this exact failure. The decision log in PROJECT.md records: "fyne.io/systray conflicts with Wails AppDelegate (duplicate symbol) — marked as ⚠️ Revisit."

**Why it happens:**
Wails v2's macOS backend defines its own `AppDelegate` in Objective-C via cgo. Any library that also defines `AppDelegate` produces a duplicate symbol at link time. The Go linker cannot resolve two Objective-C implementations of the same class name. This is not a version issue — it is a structural incompatibility that affects all versions of these libraries.

**How to avoid:**
The only safe approach on macOS is to implement the tray icon using a custom cgo wrapper that creates an `NSStatusItem` directly without defining any new Objective-C class that conflicts with Wails. The implementation must:
1. Create a `NSStatusItem` via `[[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength]`
2. Attach an `NSMenu` to it
3. Handle click events via an existing Wails-compatible mechanism (a category, not a new class, or a C callback)

Do NOT introduce any of these libraries: `fyne.io/systray`, `github.com/energye/systray`, `github.com/getlantern/systray`, `github.com/cratonica/trayhost`. All have their own `AppDelegate` or Cocoa main-thread assumptions that conflict with Wails.

Linux and Windows require entirely separate, stub-compatible implementations (see Pitfall 2).

**Warning signs:**
- Build error contains `duplicate symbol` referencing `AppDelegate`
- Build succeeds on Linux/Windows but fails only on macOS
- The cgo import path includes `fyne.io` anywhere in the dependency tree

**Phase to address:** Phase: System Tray — macOS implementation must use a custom cgo NSStatusItem approach, not a cross-platform library.

---

### Pitfall 2: The "No Dock Icon" Requirement Requires LSUIElement in Info.plist — Runtime API Alone Is Not Enough

**What goes wrong:**
Setting `NSApplicationActivationPolicyAccessory` at runtime (via cgo or via Wails's Dock Service) causes the dock icon to appear briefly at launch, then disappear. Users see a flash of the dock icon on every app start. For a daemon tray app that is supposed to be invisible, this is unacceptable.

Additionally, when running as an accessory app, certain Wails behaviors change: `WindowShow()` may not bring the management window to the front correctly because the app is not a regular activation-policy app.

**Why it happens:**
macOS evaluates the Info.plist `LSUIElement` flag before any application code runs. Setting activation policy at runtime only takes effect after `NSApplicationMain` has already registered the dock icon. Setting `LSUIElement = YES` in Info.plist prevents the dock icon from appearing at all — it is never registered.

Wails v2 generates `Info.plist` from templates during `wails build`. The relevant key must be in the template used for macOS builds, not set via runtime Go code.

**How to avoid:**
Add `LSUIElement` to the macOS Info.plist template used by the Wails build:
```xml
<key>LSUIElement</key>
<true/>
```

This plist is located at `build/darwin/Info.plist`. Verify it is present in the built `.app` bundle after `wails build` by inspecting `Contents/Info.plist` in the output.

Important implication: with `LSUIElement = YES`, the app has no menu bar application name, no dock icon, and cannot be brought forward by clicking the dock. The management window must be opened exclusively from the tray menu click. The tray menu becomes the only entry point.

**Warning signs:**
- Dock icon appears for ~10ms on startup before vanishing
- `wails build` output does not include `LSUIElement` in `Contents/Info.plist`
- Management window does not receive focus when shown via `WindowShow()` while app has policy `Accessory`

**Phase to address:** Phase: System Tray — set LSUIElement in Info.plist template at the start; test with production `wails build` (not `wails dev`).

---

### Pitfall 3: Adding a New Binary Frame Type to the Relay Protocol Breaks Existing Web Clients That Don't Ignore Unknown Types

**What goes wrong:**
The existing relay framing protocol in `internal/relay/protocol.go` defines message types 0x01–0x03 (output, resize, title) and 0x10–0x12 (input, resize2, ping). Adding a new type — for example `MsgStatus byte = 0x20` for a remote session status indicator — will be received by any existing web terminal client connected via WebSocket.

The current web terminal JavaScript parses all binary frames. If it does not have an explicit `default: break` (or equivalent) in the type switch, receiving an unknown type byte may cause a crash, incorrect output rendering, or silent corruption.

Additionally, the scrollback buffer in `internal/relay/scrollback.go` records all frames. If status frames are recorded in scrollback, they will be replayed to new clients — which is correct for most frames but may be wrong for transient status frames (e.g., "session is connecting" should not be replayed after the session is already running).

**Why it happens:**
The protocol was designed for terminal I/O and was not explicitly versioned. The frontend JavaScript switch on type bytes is not documented to handle unknown types gracefully. Scroll back replay applies uniformly to all frames without filtering by type.

**How to avoid:**
1. Before adding any new frame type, audit the web terminal JavaScript client to verify it has a safe default handler for unrecognized type bytes. Add one if absent.
2. Classify new frame types as "non-scrollback" vs "scrollback" in the relay hub. Status frames should be classified as non-scrollback: not recorded, only broadcast to currently-connected clients.
3. Add the new frame type to `protocol.go` with a constant and a `Make*Frame` helper function.
4. Update `server.go`'s read pump switch to handle any new client-to-server types.
5. Write a `protocol_test.go` round-trip test for each new frame type.

**Warning signs:**
- Web terminal shows garbage output or a partial character after reconnecting
- Status frame content appears as raw bytes rendered in the terminal viewport
- JavaScript console shows `TypeError` when processing a frame with an unrecognized first byte

**Phase to address:** Phase: Remote Session Indicators — audit and update the web terminal JS switch before adding new frame types; mark status frames as non-scrollback.

---

### Pitfall 4: Injecting a Status Bar Above/Below the Web Terminal Breaks the Terminal's CSS Height Calculation

**What goes wrong:**
The web terminal uses `xterm.js` with `FitAddon`. FitAddon calls `proposeDimensions()` which reads the parent container's `clientHeight`. If a status bar is added above or below the terminal container without subtracting its height from the flex layout, the terminal container's `clientHeight` includes the status bar region. FitAddon then calculates more rows than physically fit, causing the terminal to overflow and produce a vertical scrollbar, or the last row is hidden behind the status bar.

This is a variant of the layout pitfall that was solved in v1.6 with the bounded rAF retry loop. Adding a status bar reintroduces the same failure mode through a different cause.

**Why it happens:**
FitAddon measures the pixel height of the container, divides by character height, and floors to get row count. If the container is `height: 100%` of a flex parent that also contains a 32px status bar, and the flex layout is not set to `flex: 1 1 auto` (with `overflow: hidden`) on the terminal div, the terminal div's computed height includes the status bar pixels.

The existing `StatusBar.tsx` for web-serving state already uses a fixed 32px height with JSX conditionals (per PROJECT.md: "Always rendered at 32px height regardless of state — no layout reflow on toggle"). The remote session status bar must follow the same pattern.

**How to avoid:**
- Add the remote session status bar as a sibling to the terminal container inside a `display: flex; flex-direction: column` wrapper
- Give the terminal container `flex: 1 1 0; min-height: 0; overflow: hidden`
- Fix the status bar at a constant pixel height (e.g., 28px or 32px) — no dynamic height, no conditional rendering that changes height
- After adding the status bar, verify `proposeDimensions()` returns the same row count as the viewport physically shows — use `console.log(fitAddon.proposeDimensions())` on connect
- The existing bounded rAF retry loop already handles timing; the layout fix is a CSS concern

**Warning signs:**
- Terminal has a vertical scrollbar when the status bar is shown
- Last row of terminal output is clipped or hidden
- Row count from `proposeDimensions()` is 1–2 rows more than the visible viewport fits

**Phase to address:** Phase: Remote Session Indicators — implement the CSS flex structure correctly from the start; test by comparing proposeDimensions() row count against visible rows in the viewport.

---

### Pitfall 5: Wails v2 Has No Native Multi-Window API — A Tray Management Window Requires a Workaround

**What goes wrong:**
Wails v2's architecture is built around a single main window created by `wails.Run()`. There is no first-class API for creating a second window at runtime (unlike v3, which has full multi-window support). Attempts to open a second Wails window at runtime either fail silently, cause a crash, or create a second Wails runtime instance, which is not supported.

**Why it happens:**
Wails v2 was designed with a single-window model. The `wails.Run()` call takes over the main thread and manages the lifecycle of exactly one `WebviewWindow`. Issue #1480 ("Support Multiple Windows") was the primary driver for the v3 rewrite. The tray mini management window feature requires either: (a) reusing the existing Wails main window (show/hide based on tray clicks), or (b) opening a native OS window via a separate cgo implementation, or (c) a lightweight approach like a system notification or a web-based popup URL.

**How to avoid:**
The only v2-compatible approach is to reuse the single Wails window as the management interface. The tray click shows/hides the main window. The main window's React app renders different content based on whether it was opened from the tray vs. from the standard launch. Do not attempt to spawn a second `wails.Run()` or a second `webview`.

Specifically:
- Use `runtime.WindowShow(ctx)` / `runtime.WindowHide(ctx)` from the tray click handler
- Use Wails events (`runtime.EventsEmit`) to notify the React frontend which "mode" it should render (tray-opened vs. full-open)
- The existing window can be centered to the screen or positioned near the tray icon (use `runtime.WindowSetPosition`)
- Do not use `runtime.WindowCenter()` + `Show()` inside a cgo callback — this can deadlock if called from a non-main thread. Use a Go channel or goroutine to dispatch to the Wails runtime context

**Warning signs:**
- App crashes on tray click with a nil pointer or "runtime not initialized"
- Second window appears but has no Wails JavaScript bridge (blank/non-functional)
- `wails.Run()` is called twice in the same process

**Phase to address:** Phase: System Tray — management window is the existing Wails window, shown/hidden from tray click; no second window.

---

### Pitfall 6: The Daemon Process and the GUI Process Both Trying to Own the Tray Icon Creates Two Icons

**What goes wrong:**
AgentHub v1.7 plans a "daemon tray icon." The daemon is a background process (`internal/daemon`) that manages sessions. The Wails GUI is a separate process. If both the daemon and the GUI try to create a tray icon (e.g., the daemon uses cgo NSStatusItem and the Wails GUI also tries to create one), two tray icons appear. Users see duplicate icons; clicking the wrong one performs unexpected actions.

**Why it happens:**
The architecture decision is ambiguous: "Daemon system tray icon (no taskbar/dock icon)" could be interpreted as the daemon process owning the tray icon, or the Wails GUI owning the tray icon while the daemon is the data source. If the daemon runs as a `kardianos/service` service (launchd/systemd), it runs as a background process without a UI session on some configurations — it cannot create UI elements at all on macOS (`LSUIElement` is a GUI-process concern; background services may not have screen access).

**How to avoid:**
The tray icon must be owned by exactly one process: the Wails GUI process. The daemon provides status data (session count, health) via the existing Unix socket IPC. The GUI polls the daemon and updates the tray menu content accordingly. The daemon itself does not create any UI elements.

Rationale: macOS services launched via launchd without a UI session (`SessionCreate false`) cannot access the screen. The GUI process always runs in the user's login session. This is consistent with how tools like Docker Desktop operate (daemon as service, tray icon in the GUI agent).

On first launch with no GUI window open, the app should run as `LSUIElement` (tray-only) — the daemon is a separate background service already, and the Wails app is the lightweight tray agent.

**Warning signs:**
- Two tray icons appear after launching the app
- Tray icon disappears when the management window is closed (because the window closing terminates the Wails process)
- Daemon service logs show errors about "no display connection" when attempting UI initialization

**Phase to address:** Phase: System Tray — establish process model explicitly: GUI owns tray, daemon is headless; document this in code comments.

---

### Pitfall 7: macOS .icns Missing Specific Sizes Causes Blurry or Missing Icons at Certain Scales

**What goes wrong:**
If the `.icns` file does not contain all required size/scale combinations, macOS will upscale or downscale an available size to fill the gap. At 16x16 and 32x32 (used in window title bars, menu bars, and Finder sidebars), upscaling from a 512px source produces blurry results that look unprofessional. The tray icon specifically renders at 18x18pt (36px on Retina) on macOS — if neither `16x16@2x` nor a purpose-built template size is present, the tray icon will appear pixelated.

**Why it happens:**
The `iconutil` + `sips` workflow requires exactly 10 files in the `.iconset` folder with specific naming:
```
icon_16x16.png       (16x16)
icon_16x16@2x.png    (32x32)
icon_32x32.png       (32x32)
icon_32x32@2x.png    (64x64)
icon_128x128.png     (128x128)
icon_128x128@2x.png  (256x256)
icon_256x256.png     (256x256)
icon_256x256@2x.png  (512x512)
icon_512x512.png     (512x512)
icon_512x512@2x.png  (1024x1024)
```
Developers commonly generate only 5 sizes (missing the `@2x` variants), which produces 5-file icns files that look fine on non-Retina displays but blurry on Retina.

Additionally, macOS system tray icons are displayed as template images (white with transparency mask) on macOS 10.14+. A full-color icon in the tray will not adapt to dark/light mode. The tray icon must be a template image (suffix `Template` in the image name when using NSImage, or set `[item.button setImageScalingFactor:NSImageScaleProportionallyDown]` with a monochrome PNG).

**How to avoid:**
- Generate the source asset at 1024x1024 minimum (ideally 1024x1024 from vector/SVG)
- Use a script that generates all 10 files with correct naming conventions, then runs `iconutil -c icns`
- For the tray icon specifically, generate a separate monochrome PNG at 18x18pt source (18px and 36px at @2x) and mark it as a template image in the cgo NSStatusItem code: `[statusItem.button setImage:[NSImage imageNamed:@"TrayIconTemplate"]]`
- Validate the `.icns` contents with `iconutil -c icns --convert iconset` and inspect with Preview — check it renders sharply at 16x16 display size

**Warning signs:**
- Tray icon appears white on white in Light Mode, or black on black in Dark Mode
- App icon in Finder or Dock appears blurry when the window is at small size
- `file AppIcon.icns` reports fewer than 10 embedded images
- `sips -g all AppIcon.icns` shows maximum dimension is 512 (missing 1024 @2x layer)

**Phase to address:** Phase: App Branding — generate all 10 icns sizes from the start; generate a separate template PNG for the tray icon.

---

### Pitfall 8: Windows .ico Missing Small Sizes Causes Blurry Taskbar and Explorer Icons

**What goes wrong:**
Windows uses the `.ico` file at multiple sizes simultaneously: 16x16 (taskbar small icons, Explorer list view), 24x24 (some system contexts), 32x32 (Explorer medium view, legacy), 48x48 (Explorer large view), 256x256 (high-DPI and Windows 10/11 tiles). If only a single 256x256 PNG is embedded in the `.ico`, Windows downscales it to 16x16 for the taskbar — the result is blurry, indistinct, and may show compression artifacts.

The electron-builder community documented this exact issue: "Icons look jagged on Windows 10+ when using 256x256 icon due to bad downscaling" (issue #7328). The Wails build pipeline requires an `.ico` file at `build/windows/icon.ico` — if this file contains only one size, the same problem occurs.

**Why it happens:**
`.ico` is a multi-image container format. Each embedded image is a separately designed bitmap at its target size. Small icons (16x16, 32x32) need hand-crafted or carefully downscaled designs with higher contrast and fewer details — not just a rescaled version of the 256px master.

**How to avoid:**
- Build the `.ico` with at minimum: 16x16, 32x32, 48x48, 256x256 (PNG-compressed format for 256x256 is acceptable and preferred)
- Use ImageMagick (`convert`): `magick source_1024.png -define icon:auto-resize=256,48,32,16 icon.ico`
- Or use a tool like `icotool` (Linux/cross-platform)
- Verify the output: `magick identify icon.ico` should list 4 entries at the expected sizes
- The 16x16 and 32x32 versions benefit from a simpler design (remove fine details, increase contrast) — use the same logo simplified, not a direct downscale

**Warning signs:**
- `magick identify icon.ico` shows only one image entry
- Taskbar icon appears blurry or shows a featureless blob at small sizes
- `wails build` completes but Windows shows a generic app icon (Wails placeholder not replaced)

**Phase to address:** Phase: App Branding — generate multi-resolution `.ico` from the start; verify with `magick identify`.

---

### Pitfall 9: Wails Splash Screen Shown Before WebView Content Is Ready Causes a White Flash, Not a Splash Screen

**What goes wrong:**
A common implementation pattern is: start with the window visible, render a splash component in React, then hide it when the app is ready. In Wails, the window opens, shows the OS-native white/gray background for ~100–300ms while the WebView is initializing, then React renders the splash. The user sees: white flash → splash screen → main app. The "white flash before splash" defeats the purpose.

Alternatively: start with `StartHidden: true` and call `runtime.WindowShow(ctx)` from the `OnDomReady` callback. This eliminates the white flash but creates a different problem: `OnDomReady` fires when the DOM is ready, not when React has finished initial rendering. If the splash is pure CSS/HTML (no JavaScript required), this is fine. If the splash requires React hydration, `OnDomReady` fires too early and the splash may render for a single frame before the React tree renders.

**Why it happens:**
Wails's `StartHidden` option hides the native OS window, but the WebView still initializes in the background. `OnDomReady` maps to the browser `DOMContentLoaded` event — not `load` or the React commit phase. On slower machines or during cold starts (no cached assets), there can be a visible lag between `DOMContentLoaded` and the first React paint.

**How to avoid:**
Use the combined approach:
1. Set `StartHidden: true` in Wails options
2. Implement the splash screen as static HTML in `index.html` (outside React) — a `<div id="splash">` with inline styles. This renders with zero JavaScript latency, immediately after the WebView initializes.
3. Call `runtime.WindowShow(ctx)` from `OnDomReady` — at this point the static splash is already visible
4. In React, after initialization is complete, hide the splash div and render the main app
5. The splash div removal triggers no layout reflow because it is absolutely positioned

Avoid using the Wails `runtime.WindowShow()` from a goroutine with a fixed `time.Sleep` delay — timing is unpredictable across machines and produces either too-early show (white flash) or too-long splash (app appears frozen).

**Warning signs:**
- White/gray OS window background visible for >50ms before any content appears
- Splash screen itself flickers or renders for a single frame with incorrect dimensions
- On Windows with WebView2 cold start, app appears frozen for 2–3 seconds (WebView2 runtime initialization)

**Phase to address:** Phase: App Branding — splash screen implementation; use static HTML splash + `StartHidden: true` + `OnDomReady` show.

---

### Pitfall 10: Linux Tray Icon Requires Extension or Proxy on Modern GNOME — No Guarantee of Visibility

**What goes wrong:**
On modern GNOME (default on Ubuntu 22.04+, Fedora, and many other distros), the system tray (StatusNotifierItem / AppIndicator spec) is not supported natively. GNOME removed the system tray in GNOME 3.26. Without the `gnome-shell-extension-appindicator` extension installed, any tray icon the app creates is silently invisible — no error, no fallback, no notification to the user.

Additionally, `gnome-shell-extension-appindicator` was flagged as incompatible with GNOME 48 (released 2025), meaning this problem will worsen over time as distros ship newer GNOME.

On KDE (Plasma), the StatusNotifierItem protocol works natively. On i3/XFCE/MATE with system tray support, legacy XEmbed systray works. The behavior varies by desktop environment in ways that are not detectable at app startup.

**Why it happens:**
GNOME's design philosophy removed the persistent system tray. The AppIndicator protocol exists as a community workaround (originally from Ubuntu/Unity) but requires an extension. The `fyne.io/systray` library (which is excluded anyway due to the Wails conflict) uses DBus with the SNI/AppIndicator spec but this does not guarantee visibility on GNOME without the extension.

**How to avoid:**
For v1.7, tray on Linux is a best-effort feature with documented limitations:
- Accept that the tray icon will not be visible on stock GNOME without the AppIndicator extension
- Display a first-run notice on Linux: "If you don't see the tray icon, install gnome-shell-extension-appindicator or run the app from your application launcher"
- The app should not depend on the tray icon for critical functionality on Linux — provide an alternative launch mechanism (e.g., the desktop launcher shows the management window directly)
- On Windows, the system tray (notification area) has full support and no extension requirement

For the custom cgo NSStatusBar approach on macOS, a separate stub is needed for Linux and Windows. The Linux stub can use a pure-Go DBus SNI implementation or be a no-op for v1.7 with a follow-up ticket.

**Warning signs:**
- Running on GNOME: no tray icon appears and no error is logged (silent failure)
- Running on GNOME 48+: even with appindicator extension installed, icon may be missing
- CI tests pass but manual test on a clean Ubuntu 24.04 VM shows no tray icon

**Phase to address:** Phase: System Tray — document Linux limitation explicitly; Linux tray as best-effort; provide non-tray launch path.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Tray icon as no-op stub on Linux | Unblocks macOS/Windows delivery | Linux users get no tray management — must use CLI | Acceptable for v1.7 with documented follow-up |
| Single Wails window show/hide instead of dedicated tray window | No multi-window complexity | Management UI shares React state with terminal tabs — messy | Acceptable for v1.7; v3 migration (whenever that happens) would enable a proper separate window |
| Using `sips` downscale for all icon sizes | Script simplicity | Small icon sizes (16, 32px) look blurry without hand-crafted artwork | Acceptable if source is a clean vector/SVG; review results manually before shipping |
| Static HTML splash screen outside React | Instant render, zero JS dependency | CSS and app fonts may not match React styles exactly | Acceptable — keep splash visually simple (logo + background color only) |
| Status frame broadcast-only (no scrollback) | Avoids stale status in scrollback replay | New clients connecting after a status change don't immediately see current status | Acceptable — clients should request status via HTTP API on connect, not via scrollback |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| cgo NSStatusItem | Define a new Objective-C class that conflicts with Wails AppDelegate | Implement tray using C functions and callbacks attached to existing Wails AppDelegate via a category; never define `@implementation AppDelegate` in tray code |
| Wails `runtime.WindowShow()` from cgo callback | Call Go runtime from NSMenu action handler directly — deadlock on main thread | Post to a Go channel or use `dispatch_async(dispatch_get_main_queue(), ...)` to defer back to Go's goroutine scheduler |
| Relay protocol new frame type | Forget to update web terminal JavaScript switch — unknown bytes rendered as terminal output | Audit JS switch before adding frame type; add `default: return` for unknown types; use protocol_test.go for every frame type |
| `iconutil` icns generation | Generate only 5 sizes (missing @2x variants) | Script must generate all 10 named files; verify with `sips -g all AppIcon.icns` |
| Windows `.ico` | Single-image ico file from a 256px PNG | Use `magick convert` with `icon:auto-resize=256,48,32,16`; verify with `magick identify icon.ico` |
| Wails Info.plist on macOS | Set `LSUIElement` via runtime Go code after `NSApplicationMain` | Edit `build/darwin/Info.plist` template before `wails build` runs |
| `kardianos/service` daemon on macOS | Assume service can create UI elements (NSStatusItem) | Service runs without screen access in launchd; all UI elements must be in the GUI process, not the service |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Tray menu rebuilt on every daemon poll | Tray menu flickers or has a noticeable delay on open | Build the NSMenu object once; update individual menu items in place via `setTitle:` rather than rebuilding the entire menu tree | Every poll cycle if menu is rebuilt each time |
| Daemon poll goroutine without back-pressure | If daemon is slow or socket reconnect is retrying, new polls stack up, creating a goroutine leak | Use a `time.Ticker` (not `time.Sleep` in a loop) with a `select` that drops missed ticks; bound reconnect retries with exponential backoff | When daemon is restarting or the socket is unavailable |
| Status bar WebSocket frame sent on every PTY output byte | If MsgStatus is sent per-output-byte (via the write pump), it floods the WebSocket channel | Only send MsgStatus frames on state transitions (connected → disconnected → reconnecting), not per output byte | Any connected web terminal client with active output |
| Splash screen asset loaded from Wails embed (MIME type) | Splash image fails to load with MIME type error in production build | Always build with `-tags wailsassets` for production; use `wails build` not a manual `go build` | Production binary without the embed build tag |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Tray menu "Open in Browser" action opens an arbitrary URL from daemon response | SSRF or URL spoofing if daemon IPC is compromised | Validate URL origin: only open URLs with `https://` scheme and a known FQDN prefix (the tailscale FQDN) |
| Tray right-click menu exposes session list with IDs | Session IDs in the tray menu are readable by any macOS accessibility process or screenshot | Not a critical risk for a local Tailscale-only deployment; acceptable for v1.7; do not expose API keys or auth tokens in menu items |
| NSStatusItem cgo code receives untrusted input from menu item actions | Menu item `tag` or `representedObject` values passed back to Go | Use a fixed integer tag per menu item; do not pass user-controlled strings as `representedObject`; map tags to Go constants |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Tray icon shows no visual difference when daemon is down | User doesn't know sessions are unavailable | Use a different tray icon image (grayed-out or with an overlay dot) when the daemon is unreachable |
| Management window opens behind other windows | User clicks tray, sees nothing, thinks the app didn't respond | Call `runtime.WindowSetAlwaysOnTop(ctx, true)` then `WindowShow()` then `WindowSetAlwaysOnTop(ctx, false)` to bring it to front once |
| Remote session status bar "DISCONNECTED" persists after reconnect | Web terminal user sees stale disconnected state | Status bar must update on both connect and disconnect events; test by simulating a WebSocket reconnect |
| CLI attach `agenthub attach <id>` shows no visual indicator that it's a remote session | User not aware of latency characteristics | Print a one-line status header to stderr before starting the PTY proxy: "Attached to remote session <id> (type Ctrl-\ to detach)" — already done for detach key, extend it |
| Splash screen blocks the window during daemon health check | Startup appears frozen if Tailscale check takes 2–5 seconds | Splash should be a brief branding moment (500ms max), not a loading gate; run health checks after the window is fully shown |

---

## "Looks Done But Isn't" Checklist

- [ ] **macOS tray icon:** Tray icon appears immediately on app launch with no dock icon flash — verify with a production `wails build`, not `wails dev`.
- [ ] **macOS tray icon:** Tray icon uses template image mode — appears white in Light Mode menu bar and adapts to Dark Mode correctly.
- [ ] **macOS tray icon:** `LSUIElement = YES` is present in the built `.app/Contents/Info.plist` — inspect file directly after `wails build`.
- [ ] **macOS tray icon:** Management window opens centered/near tray icon, receives focus, and closes/hides without terminating the process.
- [ ] **Windows tray icon:** Tray icon appears in the system notification area with correct icon (not Wails placeholder default).
- [ ] **Windows .ico:** `magick identify build/windows/icon.ico` lists 4+ size entries (16, 32, 48, 256).
- [ ] **macOS .icns:** `sips -g all build/darwin/AppIcon.icns` shows 10 size variants including 1024x1024 @2x.
- [ ] **Tray menu:** Session count in tray menu updates within 2 seconds of a session being created or killed via CLI.
- [ ] **Tray menu:** "Quit" menu item terminates both the GUI process and prompts/confirms daemon shutdown (does not orphan sessions).
- [ ] **Relay protocol:** New MsgStatus frame type renders nothing in the terminal viewport — web terminal JS ignores unknown types.
- [ ] **Remote session status bar:** Adding the status bar does not introduce a vertical scrollbar in the web terminal (`proposeDimensions()` row count unchanged).
- [ ] **Remote session status bar:** Status bar height is constant (no layout reflow on connect/disconnect state change).
- [ ] **Splash screen:** No white flash before splash — window is hidden until `OnDomReady`; static splash HTML renders before React.
- [ ] **Splash screen:** On Windows, WebView2 cold-start delay is handled gracefully — splash is visible, app does not appear frozen.
- [ ] **Linux:** App launches without crashing even when no tray is available — tray init failure is logged and swallowed, not panicked.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| fyne/energye systray duplicate symbol | LOW | Remove conflicting dependency from go.mod; implement custom cgo NSStatusItem wrapper from scratch (~150 lines of Objective-C + Go) |
| Dock icon flash (missing LSUIElement) | LOW | Add `LSUIElement` to `build/darwin/Info.plist` template; rebuild with `wails build`; no code changes required |
| New relay frame type corrupts terminal output | MEDIUM | Revert frame type addition; add JS `default: return` handler; re-add frame type after verification |
| Status bar breaks terminal row count | LOW | Fix CSS flex layout (`flex: 1 1 0; min-height: 0; overflow: hidden` on terminal container); no protocol changes needed |
| Two tray icons appear | LOW | Identify which process creates the duplicate; remove tray init from daemon process; tray owned exclusively by GUI |
| icns missing @2x layers | LOW | Rerun icon generation script with all 10 files; rebuild Wails macOS target |
| ico single-size blurry | LOW | Regenerate with `magick convert source.png -define icon:auto-resize=256,48,32,16 icon.ico`; rebuild Windows target |
| Splash shows white flash | MEDIUM | Add `StartHidden: true` to Wails options; add static HTML splash in `index.html`; wire `OnDomReady` to `WindowShow()` |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| fyne/energye duplicate symbol on macOS | System Tray | `wails build` produces binary — zero linker errors; tray icon appears in menu bar |
| Dock icon flash (missing LSUIElement) | System Tray | Production `.app/Contents/Info.plist` contains `LSUIElement = YES`; no dock icon at launch |
| Second window attempt crash | System Tray | Management window is the existing Wails window shown/hidden; no second `wails.Run()` |
| GUI vs. daemon both creating tray icons | System Tray | Only one tray icon visible; daemon process has no UI initialization code |
| Linux tray silent failure | System Tray | App runs on Ubuntu 22.04 with no extension — no crash; log message about tray not available |
| New relay frame type breaks web terminal | Remote Session Indicators | Web terminal renders no garbage bytes; protocol_test.go covers new frame type |
| Status bar breaks terminal fit | Remote Session Indicators | `proposeDimensions()` row count matches visible rows; no vertical scrollbar |
| macOS .icns missing @2x | App Branding | `sips -g all AppIcon.icns` lists 10 size entries |
| Windows .ico single-size | App Branding | `magick identify icon.ico` lists 4+ entries; taskbar icon sharp at 16px |
| Splash screen white flash | App Branding | No white OS background visible before splash; `StartHidden: true` in Wails options |
| Tray icon not a template image | App Branding | Tray icon adapts to macOS Dark/Light Mode automatically |

---

## Sources

- Wails v2 AppDelegate duplicate symbol issue (fyne/energye conflict): https://github.com/wailsapp/wails/issues/3003
- fyne.io MenuItem type conflict with systray: https://github.com/fyne-io/fyne/issues/632
- Wails v2 system tray community discussion (custom cgo approach): https://github.com/wailsapp/wails/discussions/4514
- Wails v2 dock icon hiding issue and Dock Service PR #4451: https://github.com/wailsapp/wails/issues/3700
- Wails v2 LSUIElement / activation policy: https://github.com/wailsapp/wails/issues/3374
- Wails v2 multiple windows limitation (primary driver for v3): https://github.com/wailsapp/wails/issues/1480
- macOS .icns required sizes and naming: https://gist.github.com/jamieweavis/b4c394607641e1280d447deed5fc85fc
- macOS .icns compression and modern standards: https://en.wikipedia.org/wiki/Apple_Icon_Image_format
- macOS icon margin/scaling standard: https://mjtsai.com/blog/2025/10/02/how-to-export-a-mac-icon-file-with-the-proper-margins/
- Windows .ico required sizes (Microsoft): https://learn.microsoft.com/en-us/windows/apps/design/iconography/app-icon-construction
- Windows .ico blurry downscaling (electron-builder issue #7328): https://github.com/electron-userland/electron-builder/issues/7328
- Linux GNOME AppIndicator extension: https://extensions.gnome.org/extension/615/appindicator-support/
- Linux AppIndicator GNOME 48 compatibility issue: https://bbs.archlinux.org/viewtopic.php?id=304357
- Wails splash screen via WindowShow/Hide: https://github.com/wailsapp/wails/pull/1599
- AgentHub codebase (direct read): `internal/relay/protocol.go` (frame types 0x01–0x03, 0x10–0x12), `internal/relay/server.go` (read pump switch), `internal/relay/scrollback.go` (scrollback buffer), `frontend/src/components/StatusBar.tsx` (32px fixed height JSX pattern), `internal/daemon/types.go` (IPC types), `build/darwin/Info.plist` (macOS build template)

---
*Pitfalls research for: AgentHub v1.7 — daemon tray management, remote session indicators, app branding/icons*
*Researched: 2026-03-31*
