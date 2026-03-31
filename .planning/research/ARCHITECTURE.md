# Architecture Research

**Domain:** Desktop app — daemon tray management, remote session indicators, and app branding for AgentHub v1.7
**Researched:** 2026-03-31
**Confidence:** HIGH — conclusions drawn from direct codebase inspection + Wails v2 docs + official Apple/platform references

> This document supersedes the v1.5 architecture research. The arg-passthrough and terminal-fit architecture described there is now fully implemented in v1.6. This document focuses on v1.7 integration points only.

---

## System Overview (v1.6 baseline + v1.7 additions)

### v1.6 Existing Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  agenthub binary (GUI mode)                                                 │
│                                                                             │
│  ┌─────────────────────────────────────────────┐                           │
│  │  Wails App (main package)                   │                           │
│  │  - App struct: thin DaemonClient shell      │                           │
│  │  - startup() → EnsureDaemon → initTray()    │                           │
│  │  - beforeClose() → WindowHide (tray persist)│                           │
│  │  - React frontend via embedded assets       │                           │
│  └─────────────┬───────────────────────────────┘                           │
│                │ Unix socket (HTTP/JSON)                                    │
│                ▼                                                            │
│  ┌─────────────────────────────────────────────┐                           │
│  │  Daemon Process (internal/daemon)           │                           │
│  │  - SessionEngine: owns all PTY sessions     │                           │
│  │  - API: HTTP over Unix socket               │                           │
│  │  - Relay: WebSocket relay (127.0.0.1:rand)  │                           │
│  │  - WebServer: TLS (Tailscale interface)     │                           │
│  └─────────────────────────────────────────────┘                           │
│                                                                             │
│  ┌─────────────────────────────────────────────┐                           │
│  │  macOS System Tray (tray.go — darwin cgo)   │                           │
│  │  - NSStatusBar → "Show AgentHub" + "Quit"   │                           │
│  │  - Linux/Windows: no-op stubs               │                           │
│  └─────────────────────────────────────────────┘                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### v1.7 Target Architecture (new blocks marked NEW/MODIFIED)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  agenthub binary (GUI mode)                                                 │
│                                                                             │
│  ┌─────────────────────────────────────────────┐                           │
│  │  Wails App (main package)                   │                           │
│  │  - startup(): unchanged                     │                           │
│  │  - beforeClose(): unchanged                 │                           │
│  │  - React frontend: NEW SplashScreen +        │                           │
│  │    DaemonPanel components                   │                           │
│  └─────────────┬───────────────────────────────┘                           │
│                │                                                            │
│  ┌─────────────────────────────────────────────┐ MODIFIED                  │
│  │  macOS System Tray (tray.go cgo)            │                           │
│  │  - NSStatusBar: Show / Daemon Mgr / Quit    │  ← "Daemon Mgr" item new  │
│  │  - onTrayDaemonMgr() → EventsEmit           │                           │
│  └─────────────────────────────────────────────┘                           │
│                                                                             │
│  ┌─────────────────────────────────────────────┐ NEW                       │
│  │  Linux/Windows System Tray (tray_*.go)      │                           │
│  │  - systray-on-wails (Register, non-blocking)│                           │
│  │    OR keep no-op stubs (phase decision)     │                           │
│  └─────────────────────────────────────────────┘                           │
│                │ runtime.EventsEmit("daemon:show-manager")                 │
│                ▼                                                            │
│  ┌─────────────────────────────────────────────┐ MODIFIED                  │
│  │  React Frontend                             │                           │
│  │  - App.tsx: + initComplete state,           │                           │
│  │    + EventsOn("daemon:show-manager")        │                           │
│  │  - SplashScreen.tsx (NEW)                   │                           │
│  │  - DaemonPanel.tsx (NEW)                    │                           │
│  └─────────────────────────────────────────────┘                           │
│                │ existing Wails bindings                                   │
│                ▼                                                            │
│  ┌─────────────────────────────────────────────┐ UNCHANGED                 │
│  │  Daemon Process (internal/daemon)           │                           │
│  │  - No new IPC routes for DaemonPanel        │                           │
│  │    (uses ListSessions / KillSession /       │                           │
│  │     GetSessionStatus — all exist)           │                           │
│  └─────────────────────────────────────────────┘                           │
│                                                                             │
│  ┌─────────────────────────────────────────────┐ MODIFIED                  │
│  │  web/terminal.html (remote browser view)    │                           │
│  │  - Status bar div above terminal            │                           │
│  │  - JS polling GET /api/sessions/{id}/status │                           │
│  └─────────────────────────────────────────────┘                           │
│                                                                             │
│  ┌─────────────────────────────────────────────┐ MODIFIED                  │
│  │  cmd_attach.go (CLI raw terminal proxy)     │                           │
│  │  - ANSI status banner before raw mode       │                           │
│  └─────────────────────────────────────────────┘                           │
└─────────────────────────────────────────────────────────────────────────────┘

Asset pipeline:
  docs/agenthub-title-logo.png (805x208)  → SplashScreen.tsx import
  assets/appicon.png (256x256, placeholder → NEW branded 1024x1024)
    → macOS: iconutil → AppIcon.icns → build/darwin/AppIcon.icns
    → Windows: ImageMagick → build/windows/icon.ico (REPLACE existing)
    → Linux: 256x256 PNG → build/linux/appicon.png
  assets/tray_icon.png (NEW 18x18 monochrome template) → tray.go embed
  build/darwin/Info.plist → add LSUIElement (optional, per-build decision)
```

---

## Component Responsibilities

| Component | Responsibility | Change Type |
|-----------|---------------|-------------|
| `tray.go` (darwin cgo) | NSStatusBar with Show / Daemon Mgr / Quit | MODIFIED |
| `tray_linux.go` | Linux tray (systray-on-wails or no-op) | MODIFIED |
| `tray_windows.go` | Windows tray (systray-on-wails or no-op) | MODIFIED |
| `app.go` | Wails App struct | UNCHANGED (no new bindings needed) |
| `SplashScreen.tsx` | Branding cover during init | NEW |
| `DaemonPanel.tsx` | Daemon status + session list controls | NEW |
| `App.tsx` | Root component — add initComplete + panel state | MODIFIED |
| `web/terminal.html` | Remote browser terminal | MODIFIED |
| `cmd_attach.go` | CLI raw I/O proxy | MODIFIED |
| `internal/webserver/server.go` | Add `/api/sessions/{id}/status` route | MODIFIED |
| `assets/appicon.png` | App icon source | REPLACE |
| `assets/tray_icon.png` | Tray icon (18x18 monochrome) | NEW |
| `build/darwin/AppIcon.icns` | macOS icon bundle | NEW (generated) |
| `build/darwin/Info.plist` | macOS bundle config | MODIFIED (optional) |
| `build.sh` | Build script | MODIFIED (icon generation steps) |
| `internal/relay/protocol.go` | Binary framing protocol | UNCHANGED |
| `internal/daemon/api.go` | Daemon IPC API | UNCHANGED |
| `internal/daemon/types.go` | Daemon wire types | UNCHANGED |

---

## Feature-by-Feature Integration Analysis

### 1. System Tray — Daemon Management Item

**Current state:** `tray.go` implements a darwin-only cgo NSStatusBar with two items: "Show AgentHub" and "Quit". `tray_linux.go` and `tray_windows.go` are no-op stubs. The callbacks use a package-level `trayCallbackApp *App` global, set in `initTray()`, to call `runtime.WindowShow` / `runtime.Quit`.

**v1.7 change (macOS):**

Add a "Daemon Manager" NSMenuItem between "Show AgentHub" and the separator. Its action calls a new exported cgo callback `onTrayDaemonMgr()`, which emits the Wails event `"daemon:show-manager"` and also calls `runtime.WindowShow` to bring the window forward.

This is the only correct approach under Wails v2 constraints: **Wails v2 is single-window only** (confirmed via GitHub issue #1480). Multiple OS windows require Wails v3 (alpha). The "daemon mini management window" from the milestone description must be implemented as a React panel inside the existing Wails window — not a second OS window.

```
Tray click "Daemon Manager"
  → cgo onTrayDaemonMgr()
  → runtime.WindowShow(ctx)             // raise existing window
  → runtime.EventsEmit(ctx, "daemon:show-manager", nil)
  → React EventsOn("daemon:show-manager") → setShowDaemonPanel(true)
  → DaemonPanel renders inside Wails window
```

**v1.7 change (Linux/Windows):**

`systray-on-wails` (published Nov 2024, Apache-2.0) provides cross-platform tray without conflicting with the existing macOS cgo code. It uses `Register()` instead of `Run()` — non-blocking, coexists with Wails event loop. The macOS implementation is separate (keeps existing cgo), Linux/Windows get `systray-on-wails`.

Linux build dependency: `libgtk-3-dev libayatana-appindicator3-dev` — acceptable since Linux builds already use Docker via `build.sh`.

Platform decision for v1.7: whether to implement real Linux/Windows tray or keep stubs is a scope call. The architecture supports both; the stubs can remain for a phase and real tray can follow.

**macOS dock icon (LSUIElement):**

For a tray-first app, the dock icon should ideally disappear when the user closes the window (app lives in tray). This requires `<key>LSUIElement</key><true/>` in `build/darwin/Info.plist`. However:
- `LSUIElement` also removes the app from Cmd+Tab (app switcher) — significant UX change
- Known Wails issue #3700: this is a product decision, not a blocker
- Recommendation: add to `Info.plist` (production) only, NOT to `Info.dev.plist` (dev mode). Gate behind a `build.sh` flag or note it as a phase-level decision.

### 2. Remote Session Status Bar — Web Terminal

**Existing architecture:** `web/terminal.html` is a self-contained HTML file served by `internal/webserver/server.go`. It loads xterm.js from jsDelivr CDN and connects via WSS to the relay. The binary framing protocol (relay/protocol.go) carries `MsgOutput` (0x01), `MsgInput` (0x10), `MsgResize2` (0x11), `MsgPing` (0x12). There is currently no status indicator in the remote terminal view.

**Two implementation approaches for remote status:**

Option A — New WebSocket frame type `MsgStatus` (0x03, unused):
- Server pushes status frames from engine's heuristic goroutine
- Web client handles 0x03 to update a DOM badge
- Pros: push semantics, immediate updates
- Cons: modifies relay/protocol.go + relay/server.go + relay/hub.go + terminal.html + cmd_attach.go — 5 files across 3 packages

Option B — Polling REST endpoint in terminal.html:
- Add `GET /api/sessions/{id}/status` to webserver/server.go
- terminal.html calls `fetch()` every 3 seconds
- Pros: 2 files touched, completely isolated change surface
- Cons: 3-second latency on status change

**Recommendation: Option B (polling REST).** The status bar shows ambient context — running/waiting/idle/errored. A 3-second poll delay is not perceptible to users. The `sessionResolver` callback in webserver already returns `(name, cliType, status)` from `engine.GetSessionStatus()`. The new route is three lines of Go. Option A requires touching the hot I/O path across multiple packages.

**New webserver route:**

```go
// GET /api/sessions/{id}/status
// Returns: {name, cli, status} for the session.
// Used by terminal.html status bar polling.
mux.HandleFunc("GET /api/sessions/{id}/status", ws.handleSessionStatus)
```

**terminal.html layout change:**

Current CSS: `#terminal { width: 100%; height: 100%; }` with `body { overflow: hidden }`. The status bar adds a fixed-height element. Change to flex column layout:
```css
body { display: flex; flex-direction: column; }
#status-bar { height: 28px; flex-shrink: 0; }
#terminal { flex: 1; min-height: 0; }
```

No build toolchain needed — plain JS + CSS injected directly into terminal.html.

### 3. Remote Session Status Bar — CLI Attach

**Existing architecture:** `cmdAttach` in `cmd_attach.go` opens WebSocket to relay, puts stdin in raw mode, runs two goroutines: `stdinPump` (stdin → WS input frames) and `wsOutputPump` (WS output frames → stdout). The `wsOutputPump` handles only `MsgOutput` frames.

**Approach:** Print an ANSI status header to stdout *before* entering raw mode. This is the cleanest approach — no protocol changes needed.

```
1. Before raw mode: client.GetSessionStatus(sessionID) → status string
2. Print ANSI banner: reverse-video line with session name, CLI, status, detach key hint
3. Enter raw mode, begin I/O pumps
```

The banner is printed once at connect time. A polling goroutine to refresh it is possible but adds complexity without enough value for v1.7 (the terminal output itself shows the agent's state).

The `DaemonClient` already exposes `GetSessionStatus` and `ListSessions` — session metadata is available before entering raw mode at no additional IPC cost.

### 4. App Icons and Branding

**Current state:**
- `assets/appicon.png`: 256x256, generated by `build/gen_icon.go` — a programmatic placeholder (dark blue background, stylized A shape)
- `docs/agenthub-title-logo.png`: 805x208 RGBA PNG — the real branded wide logo
- `build/darwin/appicon.png`: same 256x256 placeholder (Wails copies this in)
- `build/windows/icon.ico`: exists, generated from the placeholder

**Required deliverable — icon generation pipeline in build.sh:**

macOS ICNS:
```bash
# Create iconset directory with sips (built into macOS)
mkdir -p AppIcon.iconset
sips -z 16 16   appicon.png --out AppIcon.iconset/icon_16x16.png
sips -z 32 32   appicon.png --out AppIcon.iconset/icon_16x16@2x.png
sips -z 32 32   appicon.png --out AppIcon.iconset/icon_32x32.png
sips -z 64 64   appicon.png --out AppIcon.iconset/icon_32x32@2x.png
sips -z 128 128 appicon.png --out AppIcon.iconset/icon_128x128.png
sips -z 256 256 appicon.png --out AppIcon.iconset/icon_128x128@2x.png
sips -z 256 256 appicon.png --out AppIcon.iconset/icon_256x256.png
sips -z 512 512 appicon.png --out AppIcon.iconset/icon_256x256@2x.png
sips -z 512 512 appicon.png --out AppIcon.iconset/icon_512x512.png
sips -z 1024 1024 appicon.png --out AppIcon.iconset/icon_512x512@2x.png
iconutil -c icns AppIcon.iconset -o build/darwin/AppIcon.icns
```

Windows ICO (ImageMagick — already available in CI):
```bash
magick assets/appicon.png \
  -define icon:auto-resize=256,128,64,48,32,16 \
  build/windows/icon.ico
```

Linux: copy 256x256 PNG to `build/linux/appicon.png` (Wails reads it for .desktop file).

Tray icon: `assets/tray_icon.png` — 18x18 monochrome PNG. The existing cgo code uses `[icon setTemplate:YES]` which adapts to light/dark menu bar. The new tray icon should be a monochrome/outline design (solid dark renders as white on dark menu bars automatically). This is a separate design asset from the app icon.

**Info.plist `CFBundleIconFile`:** The `build/darwin/Info.plist` template already has `<key>CFBundleIconFile</key><string>iconfile</string>`. Wails maps this to the `build/darwin/appicon.icns` it generates internally, OR the `build/darwin/AppIcon.icns` we place there. Confirm the exact filename Wails expects by checking Wails build output; if it generates `iconfile.icns` from `build/darwin/appicon.png`, the explicit `AppIcon.icns` may need to be placed as `iconfile.icns` or the `CFBundleIconFile` value updated.

### 5. Splash Screen

**Wails v2 constraint:** No built-in splash screen API. The window appears as soon as Wails initializes. Wails docs suggest `StartHidden` + `OnDomReady` for deferred show, but this causes a blank hang before any visible content — worse UX than a CSS splash.

**Recommended approach — CSS-first React splash:**

`App.tsx` already has an `init()` function in `useEffect` that sets multiple state values when startup data is available. Before `init()` completes, `relayPort` is `null` and `tabs` is empty — the app renders in an empty state. The splash replaces this:

```typescript
// App.tsx
const [initComplete, setInitComplete] = useState(false)

useEffect(() => {
  async function init() {
    // ... existing logic ...
    setInitComplete(true)  // mark complete at end
  }
  void init()
}, [])

return (
  <div className="app">
    {!initComplete && <SplashScreen />}
    {initComplete && (
      <>
        <TabBar ... />
        <div className="terminal-container">...</div>
      </>
    )}
    {/* Modals render regardless (QR, Settings, Health) */}
  </div>
)
```

`SplashScreen.tsx` renders the title logo image (`docs/agenthub-title-logo.png` copied to `frontend/src/assets/`) plus a loading animation. The Wails window already has `BackgroundColour: #1a1b26` so the splash background is seamless.

No Go changes required. No `StartHidden` needed. Init typically completes in < 1 second (daemon is already running), so splash is brief.

**Daemon error state interaction:** If `GetDaemonError()` returns an error during init, `initComplete` should still become `true` to show the error banner. The error path is: `init()` catches the error, sets `daemonError`, sets `initComplete = true`. The error banner renders instead of the tab bar (same as current behavior). The splash only hides the error for the brief init window — acceptable.

---

## Data Flow

### Tray → DaemonPanel → Daemon IPC

```
[User: tray right-click → "Daemon Manager"]
  → cgo onTrayDaemonMgr()
  → runtime.WindowShow(ctx)                           // raise window
  → runtime.EventsEmit(ctx, "daemon:show-manager", nil)
  → React: EventsOn("daemon:show-manager")
  → setShowDaemonPanel(true)
  → DaemonPanel component mounts
  → ListSessions() → DaemonClient → Unix socket       // fetch sessions
  → render: session rows (name, status, kill button)
  → KillSession(id) / GetSessionStatus(id)            // via existing bindings
```

No new Wails bindings needed. `DaemonPanel` uses only: `ListSessions()`, `GetSessionStatus()`, `KillSession()`, `RenameSession()` — all present in `app.go`.

### Remote Status Polling Flow (Web)

```
[Browser opens /sessions/{id}]
  → terminal.html JS: setInterval(pollStatus, 3000)
  → fetch("/api/sessions/{id}/status")
  → webserver handleSessionStatus
  → ws.sessionResolver(id)
  → engine.GetSessionStatus(id)                       // via closure set in api.go
  → return {name, cli, status}
  → JS: update status bar DOM
```

### CLI Attach Status Flow

```
[$ agenthub attach <id>]
  → cmdAttach: client.GetSessionStatus(id)            // before raw mode
  → fmt.Fprintf(os.Stdout, "\033[7m REMOTE: %s (%s) [%s] — Ctrl-\\ to detach \033[0m\n",
      name, cli, status)
  → term.MakeRaw(stdin)
  → attachSession(ctx, conn, os.Stdin, os.Stdout, detachKey)
  → I/O pumps (unchanged)
```

### Splash Screen Flow

```
[Wails window opens]
  → React renders: initComplete=false → SplashScreen visible
  → init() begins: GetDaemonError(), GetRelayPort(), ListSessions(), ...
  → all calls return (< 1s normally)
  → setInitComplete(true)
  → React re-renders: SplashScreen unmounts, app UI mounts
```

### Icon Generation Flow (build.sh)

```
[build.sh mac target]
  → sips: generate 10 PNG sizes from assets/appicon.png → AppIcon.iconset/
  → iconutil: AppIcon.iconset → build/darwin/AppIcon.icns
  → wails build (Wails reads build/darwin/ for bundle assets)

[build.sh windows target]
  → magick: assets/appicon.png → build/windows/icon.ico
  → wails build (Wails embeds icon.ico in PE binary)

[build.sh linux target]
  → cp assets/appicon.png build/linux/appicon.png
  → wails build (Wails references PNG for .desktop AppIcon)
```

---

## Recommended Project Structure (New and Modified Files Only)

```
agenthub/
├── tray.go                              # MODIFIED: add "Daemon Manager" menu item
├── tray_linux.go                        # MODIFIED: real systray-on-wails or keep stub
├── tray_windows.go                      # MODIFIED: real systray-on-wails or keep stub
├── tray_test.go                         # likely unchanged
│
├── assets/
│   ├── appicon.png                      # REPLACE: 1024x1024 branded icon (square)
│   └── tray_icon.png                    # NEW: 18x18 monochrome template PNG
│
├── build/
│   ├── darwin/
│   │   ├── Info.plist                   # MODIFIED: add LSUIElement (optional)
│   │   └── AppIcon.icns                 # NEW: generated by build.sh
│   ├── windows/
│   │   └── icon.ico                     # REPLACE: regenerated with ImageMagick
│   └── linux/
│       └── appicon.png                  # NEW: 256x256 copy for .desktop
│
├── build.sh                             # MODIFIED: icon generation steps
│
├── frontend/src/
│   ├── App.tsx                          # MODIFIED: initComplete, DaemonPanel toggle, SplashScreen
│   ├── components/
│   │   ├── DaemonPanel.tsx              # NEW: session list + status + kill controls
│   │   └── SplashScreen.tsx            # NEW: branding cover during init
│   └── assets/
│       └── agenthub-title-logo.png     # NEW: copy of docs/ logo for splash
│
└── web/
    └── terminal.html                    # MODIFIED: status bar + polling JS
```

---

## Integration Boundaries

### Existing Wails Bindings — No New Bindings Needed

`DaemonPanel.tsx` requires four operations. All are already bound in `app.go`:

| Operation | Bound Method | File/Line |
|-----------|-------------|-----------|
| List all sessions | `ListSessions()` | `app.go:166` |
| Get session status | `GetSessionStatus(id)` | `app.go:217` |
| Kill a session | `KillSession(id)` | `app.go:196` |
| Rename a session | `RenameSession(id, name)` | `app.go:188` |

No `wails generate` / `wailsjs` regeneration needed for DaemonPanel.

### Internal Package Boundaries

| Boundary | Communication | v1.7 Impact |
|----------|---------------|-------------|
| tray.go → App (Go) | package-level `trayCallbackApp` global | Add `onTrayDaemonMgr` callback — same pattern as existing `onTrayShow` |
| App → React | `runtime.EventsEmit("daemon:show-manager")` | Add `EventsOn` listener in App.tsx |
| webserver → sessionResolver | Closure set in api.go before Start() | Reuse for new `/api/sessions/{id}/status` endpoint |
| cmd_attach → DaemonClient | Existing `client.GetSessionStatus()` | Call before raw mode entry — no new methods |
| tray_*.go (Linux/Windows) → Wails | `runtime.*` calls | Same pattern if systray-on-wails is used |

---

## Architectural Patterns

### Pattern 1: Tray-to-React via Wails Events

**What:** cgo (or systray-on-wails) menu callback → `runtime.EventsEmit` → React `EventsOn` handler.

**When to use:** Any native code outside the Wails WebView needs to trigger React UI state. Already established for `session:status`, `tailscale:health`, `daemon:error`. The `daemon:show-manager` event follows this exact pattern.

**Trade-off:** One-way (Go → React only). React-to-tray communication still goes through Wails-bound methods. Acceptable since tray state is simple (no feedback needed to tray from DaemonPanel actions).

### Pattern 2: CSS-First Splash Screen

**What:** React conditionally renders `SplashScreen` while `!initComplete`. The `init()` function in `useEffect` sets `initComplete = true` at completion.

**When to use:** App needs a loading state before startup data is available. Preferable to Wails-layer `StartHidden` because the splash content is visible immediately (no blank hang).

**Trade-off:** The splash renders inside the already-initialized Wails WebView. This is not a true OS-level splash (which would show before the WebView loads). For v1.7, the init time is < 1 second when the daemon is running, so the distinction is academic.

### Pattern 3: Polling REST for Ambient Status in Vanilla HTML

**What:** `setInterval` + `fetch` poll a JSON endpoint from `terminal.html`.

**When to use:** Status is ambient context (not real-time critical), the file has no build toolchain, and the push alternative requires touching multiple Go packages.

**Trade-off:** 3-second status latency. Negligible for session heuristic state (running/waiting/idle/errored changes on a seconds timescale anyway).

---

## Anti-Patterns

### Anti-Pattern 1: Second OS Window for Daemon Manager

**What people might attempt:** Use Wails v3's multi-window API, or open a native NSWindow via cgo, for the "daemon mini management window" described in the milestone.

**Why it's wrong:** Wails v2 (currently used, v2.10.2) has no multi-window API — confirmed in GitHub issue #1480. Wails v3 is in alpha. A native NSWindow via cgo would need its own separate HTML/rendering pipeline or require building a fully native Cocoa UI in cgo — a major architectural detour.

**Do this instead:** Implement DaemonPanel as a React modal/panel inside the existing Wails window. Call `runtime.WindowShow(ctx)` from the tray callback to raise it. The "mini management window" is a product concept — a focused panel inside the main window serves the same purpose.

### Anti-Pattern 2: New WebSocket Frame for Remote Status

**What people might attempt:** Add `MsgStatus` byte (0x03, currently the reserved `MsgTitle` slot) to relay/protocol.go and push status from the relay server to browser/CLI clients.

**Why it's wrong:** Requires coordinated changes to `relay/protocol.go`, `relay/server.go` (new push goroutine), `relay/hub.go` (status fan-out), `web/terminal.html`, and `cmd_attach.go`. Adds complexity to the hot I/O path. The relay server would need to query `SessionEngine` for status and fan out frames independently of PTY output.

**Do this instead:** Add `GET /api/sessions/{id}/status` to `internal/webserver/server.go`. Poll from `terminal.html`. Three lines of Go, one JS function. Leaves the relay protocol clean.

### Anti-Pattern 3: Mixing systray-on-wails with macOS cgo NSStatusBar

**What people might attempt:** Replace the existing darwin cgo tray with `systray-on-wails` for uniform cross-platform code.

**Why it's wrong:** `systray-on-wails` registers its own NSStatusBar on macOS. Running two NSStatusBar registrations is undefined behavior. The transition cost (remove working tested cgo code, validate systray-on-wails behavior on macOS) outweighs the uniformity benefit.

**Do this instead:** Keep macOS cgo tray, add `systray-on-wails` only for Linux/Windows stubs. Share menu item label constants between platforms.

### Anti-Pattern 4: LSUIElement in dev build

**What people might attempt:** Add `LSUIElement` to `build/darwin/Info.dev.plist` along with `Info.plist` for consistency.

**Why it's wrong:** `LSUIElement` removes the app from the macOS dock AND app switcher (Cmd+Tab). In dev mode, this makes the app nearly unfindable during debugging. The dev binary also runs without a proper `.app` bundle, making dock behavior different anyway.

**Do this instead:** Add `LSUIElement` only to `build/darwin/Info.plist` (production bundle). `Info.dev.plist` is used by `wails dev` and must remain a normal windowed app.

### Anti-Pattern 5: initComplete = true Before Daemon Error is Checked

**What people might attempt:** Set `initComplete = true` at the start of `init()` (or skip it on error paths), leaving the splash screen visible forever if the daemon fails.

**Why it's wrong:** If `GetDaemonError()` returns an error, the existing error banner logic must still show. The splash blocks it. `initComplete` must be set to `true` on all code paths through `init()` — success AND error.

**Do this instead:** Use `defer setInitComplete(true)` at the top of `init()`, or explicitly call it in both the success and error branches.

---

## Build Order (Suggested Phase Sequence)

Dependencies between features:

```
[Icons + Branding Assets]   (standalone — no code deps)
  ↓ unblocks
  ├─→ [Splash Screen]       (needs logo asset in frontend/src/assets/)
  └─→ (icon files used by all platform builds)

[Remote Status — Web]       (standalone — webserver + HTML only)
[Remote Status — CLI]       (standalone — cmd_attach only)

[DaemonPanel component]     (standalone — uses only existing Wails bindings)
  ↓ unblocks
  └─→ [Tray Daemon Manager item]  (needs DaemonPanel to exist in React)
          ↓ optional extension
          └─→ [Linux/Windows real tray]
```

**Recommended phase order:**

1. **Icons + branding** — Replace `assets/appicon.png` with branded 1024x1024, add `assets/tray_icon.png`, generate ICNS/ICO, update `build.sh`. No source code changes. Unblocks all visual work.

2. **Splash screen** — Add `SplashScreen.tsx`, add `initComplete` state to `App.tsx`, copy logo to `frontend/src/assets/`. No Go changes.

3. **Remote status bar (web terminal)** — Add `/api/sessions/{id}/status` route to webserver, modify `terminal.html` layout and add polling JS. Fully self-contained.

4. **Remote status bar (CLI attach)** — Modify `cmd_attach.go` to print ANSI banner before raw mode. Standalone.

5. **DaemonPanel component** — New React component using only existing Wails bindings. Can be validated by temporarily setting `showDaemonPanel = true` in App.tsx.

6. **Tray: Daemon Manager item** — Add menu item to `tray.go` + new cgo export `onTrayDaemonMgr`. Add `EventsOn("daemon:show-manager")` handler in `App.tsx`. Depends on: DaemonPanel exists.

7. **Linux/Windows tray** (optional in v1.7, can remain stubs) — Implement `systray-on-wails` in `tray_linux.go` / `tray_windows.go`.

Steps 1-4 have no mutual dependencies and can be executed as parallel phases. Steps 5-6 are coupled (DaemonPanel before tray wire-up). Step 7 is independent.

---

## Sources

- Codebase inspection (HIGH confidence): `tray.go`, `tray_linux.go`, `tray_windows.go`, `app.go`, `main.go`, `cmd_attach.go`, `internal/daemon/api.go`, `internal/daemon/types.go`, `internal/relay/protocol.go`, `internal/webserver/server.go`, `web/terminal.html`, `frontend/src/App.tsx`, `frontend/src/components/StatusBar.tsx`, `build/darwin/Info.plist`, `build/darwin/Info.dev.plist`, `build/gen_icon.go`, `go.mod`, `wails.json`
- [Wails v2 does not support multiple windows (issue #1480)](https://github.com/wailsapp/wails/issues/1480) — confirmed single-window constraint
- [Wails v2 macOS dock hide issue #3700](https://github.com/wailsapp/wails/issues/3700) — LSUIElement complexity in Wails
- [Wails v2 Application Options](https://wails.io/docs/reference/options/) — StartHidden, HideWindowOnClose, OnBeforeClose
- [systray-on-wails package](https://pkg.go.dev/github.com/ra1phdd/systray-on-wails) — cross-platform tray for Wails v2, published Nov 2024, Apache-2.0
- [Apple LSUIElement documentation](https://developer.apple.com/documentation/bundleresources/information-property-list/lsuielement) — agent app behavior, Dock/app switcher exclusion
- [iconutil + ImageMagick icon generation gist](https://gist.github.com/miguelsolorio/4f89bdf5bc2aabebf25ce45ca7cf8d97) — macOS iconset workflow

---

*Architecture research for: AgentHub v1.7 — Daemon UX & Branding*
*Researched: 2026-03-31*
