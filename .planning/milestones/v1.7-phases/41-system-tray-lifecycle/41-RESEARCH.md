# Phase 41: System Tray + Lifecycle - Research

**Researched:** 2026-04-02
**Domain:** macOS NSStatusItem (cgo), Wails v2 lifecycle, Info.plist LSUIElement, daemon IPC
**Confidence:** HIGH

## Summary

Phase 41 has substantial scaffolding already committed. `tray.go` implements a macOS NSStatusItem via custom cgo with Show/Quit menu items. `main.go` sets `HideWindowOnClose: true` and `OnBeforeClose` returns `true` — so DMGR-01 (window hides instead of quits) is already functional. The existing code also already sets `[icon setTemplate:YES]` on the NSImage, which handles light/dark menu bar adaptation correctly.

What remains is: (1) adding `LSUIElement` to `build/darwin/Info.plist` for Dock/Cmd+Tab hiding (TRAY-05), (2) creating a monochrome tray icon PNG asset (BRND-03), (3) adding dynamic menu rebuild with session names (TRAY-04), (4) adding tooltip update with session count (TRAY-06), (5) adding a second icon for error/disconnected state (TRAY-03), (6) adding a daemon self-shutdown endpoint and wiring the Quit menu item to call it before `runtime.Quit` (DMGR-02), and (7) starting a background tray state poller in the GUI that feeds all the above.

The daemon is a **separate process** — Quit must send a shutdown signal to it before the GUI exits. The cleanest approach is a `POST /shutdown` endpoint on the daemon API that calls `os.Exit(0)`.

**Primary recommendation:** Extend tray.go cgo with `updateTrayMenu` and `updateTrayIcon` C functions, add a daemon `/shutdown` endpoint, and start a tray poller goroutine in `startup()`. Add LSUIElement to both `Info.plist` templates (production + dev).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TRAY-01 | AgentHub icon in macOS menu bar | NSStatusItem in tray.go already creates the icon; no new work for the icon's presence |
| TRAY-02 | Right-click menu with "Open AgentHub", session names, "Quit" | Menu exists; session names require dynamic rebuild via new `updateTrayMenu` C func |
| TRAY-03 | Tray icon reflects daemon state (running vs error) | Requires second icon asset (tray_error.png) + `updateTrayIcon` C func that swaps NSImage |
| TRAY-04 | Tray menu lists active sessions; clicking session focuses GUI | NSMenuDelegate or rebuild on `menuWillOpen:`; session click calls `onTraySession` cgo export |
| TRAY-05 | Dock icon hidden (LSUIElement) | Add `<key>LSUIElement</key><true/>` to `build/darwin/Info.plist` AND `build/darwin/Info.dev.plist` |
| TRAY-06 | Tray icon tooltip shows active session count | `updateTrayTooltip` C func sets `statusItem.button.toolTip` from Go |
| DMGR-01 | Window close hides instead of quitting | Already done: `HideWindowOnClose: true` + `beforeClose` returns `true` |
| DMGR-02 | Quit from tray stops daemon and exits | Add `POST /shutdown` to daemon API; call it from `onTrayQuit` before `runtime.Quit` |
| BRND-03 | macOS tray icon uses monochrome template image | Create `assets/tray_icon.png` (18x18 black+alpha PNG); embed it in tray.go instead of appicon.png |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **No systray libraries**: STATE.md decision — "never add fyne.io/systray or any mainstream systray library (duplicate AppDelegate symbol confirmed failure); keep custom cgo NSStatusItem for macOS"
- **LSUIElement in Info.plist only (not Info.dev.plist)**: Wait — STATE.md says "LSUIElement in Info.plist only (not Info.dev.plist)". This means LSUIElement goes in `build/darwin/Info.plist` (production) but NOT in `build/darwin/Info.dev.plist` (dev mode keeps Dock icon for debugging convenience).
- **Node/pnpm**: Frontend uses pnpm (`wails.json` confirms `frontend:install: pnpm install`)
- **Wails v2 only**: "Staying on v2; v3 is alpha" — confirmed in REQUIREMENTS.md Out of Scope
- **Single window only**: Wails v2 is single-window; no second OS window for management
- **Go conventions**: `go fmt`, context-aware, no globals except tray callback

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Cocoa/NSStatusItem | macOS SDK | System tray icon + menu | Only macOS-native API; custom cgo avoids systray symbol conflicts |
| Wails v2 | v2.10.2 (go.mod) | Window/lifecycle management | Already in use; `runtime.WindowHide`, `runtime.Quit` |
| Go cgo | stdlib | Objective-C bridge | Required for NSStatusItem; already working in tray.go |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go net/http | stdlib | Daemon shutdown endpoint | `POST /shutdown` on daemon API for DMGR-02 |
| Go image/png + image | stdlib | Generate monochrome tray PNG | If generating programmatically via gen_icon.go pattern |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom cgo NSStatusItem | systray, fyne.io/systray | Confirmed symbol conflict failure — DO NOT use |
| `POST /shutdown` daemon endpoint | PID file + SIGTERM | No PID file infrastructure exists; API endpoint is cleaner |
| NSMenuDelegate menuWillOpen | Rebuild menu on every poll cycle | Delegate is lazier (only on open), less churn, recommended |

**No new dependencies needed** — all required APIs are already available.

## Architecture Patterns

### Files to Modify / Create

```
tray.go                             # Extend cgo with updateTrayMenu, updateTrayIcon, updateTrayTooltip
assets/tray_icon.png                # NEW: 18x18 monochrome (black+alpha) PNG for normal state
assets/tray_icon_error.png          # NEW: 18x18 monochrome error-state PNG (dot or X variant)
build/darwin/Info.plist             # Add LSUIElement key (production builds only)
internal/daemon/api.go              # Add POST /shutdown route
internal/daemon/types.go            # No change needed (no new request/response types for shutdown)
internal/daemon/client.go           # Add ShutdownDaemon() method
app.go                              # Extend onTrayQuit to call ShutdownDaemon; add startTrayPoller
```

### Pattern 1: Extend NSStatusItem with Dynamic Menu (NSMenuDelegate)

**What:** Add an `@interface AgentHubMenuDelegate : NSObject <NSMenuDelegate>` in tray.go cgo that rebuilds the menu before it opens. Go calls `setTrayState(sessions, connected)` which stores state in C globals; `menuWillOpen:` reads them.

**When to use:** Any time the menu content depends on live data (session names, count). This avoids a continuous rebuild loop and only runs when the user opens the menu.

**Example pattern:**
```objc
// In tray.go cgo block:
static NSArray *menuSessions = nil;  // array of NSString session names
static BOOL daemonConnected = YES;
static AgentHubMenuDelegate *delegate = nil;

@interface AgentHubMenuDelegate : NSObject <NSMenuDelegate>
@end

@implementation AgentHubMenuDelegate
- (void)menuWillOpen:(NSMenu *)menu {
    // Rebuild items between "Open AgentHub" and separator before "Quit"
    // Keep index 0 (Open AgentHub), index 1 (separator), remove old session items
    // Add fresh session items, then separator, then Quit
    [menu removeAllItems];
    // ... rebuild from menuSessions global
}
@end
```

**Go side:**
```go
// In tray.go (darwin):
//export onTraySession
func onTraySession(idx C.int) {
    // Focus window and optionally switch to the nth session tab
    if trayCallbackApp != nil && trayCallbackApp.ctx != nil {
        runtime.WindowShow(trayCallbackApp.ctx)
        // Emit event to frontend to focus session by index
        runtime.EventsEmit(trayCallbackApp.ctx, "tray:focus-session", int(idx))
    }
}

func (a *App) updateTray(sessions []SessionInfo, connected bool) {
    // Convert sessions to C strings and call C.updateTrayState(...)
}
```

### Pattern 2: Daemon Shutdown Endpoint

**What:** `POST /shutdown` on daemon API calls `os.Exit(0)` (or signals the daemon's context). The GUI calls this before `runtime.Quit`.

**Critical detail:** The daemon is a **separate process** (detached subprocess). `runtime.Quit` only quits the GUI. The daemon continues running after GUI exits unless explicitly told to stop. This is by design for normal window-close (DMGR-01), but Quit must terminate it.

```go
// internal/daemon/api.go
a.mux.HandleFunc("POST /shutdown", a.handleShutdown)

func (a *API) handleShutdown(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNoContent)
    // Flush response before os.Exit so GUI gets the 204 before connection closes
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
    go func() {
        time.Sleep(50 * time.Millisecond) // let response flush
        os.Exit(0)
    }()
}

// internal/daemon/client.go
func (c *DaemonClient) ShutdownDaemon() error {
    // POST with short timeout — daemon will exit and close connection
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/shutdown", nil)
    resp, err := c.http.Do(req)
    if err != nil {
        return err // connection reset is expected if daemon exits immediately
    }
    resp.Body.Close()
    return nil
}
```

**Wiring in app.go:**
```go
//export onTrayQuit
func onTrayQuit() {
    if trayCallbackApp != nil {
        if trayCallbackApp.client != nil {
            _ = trayCallbackApp.client.ShutdownDaemon()
        }
        if trayCallbackApp.ctx != nil {
            runtime.Quit(trayCallbackApp.ctx)
        }
    }
}
```

### Pattern 3: Tray State Poller Goroutine

**What:** Background goroutine started in `startup()` that polls the daemon every 5 seconds and calls `updateTray(sessions, connected)`.

```go
// app.go
func (a *App) startTrayPoller(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                a.refreshTrayState()
            case <-ctx.Done():
                return
            }
        }
    }()
}

func (a *App) refreshTrayState() {
    if !a.trayInit || a.client == nil {
        return
    }
    connected := a.client.Health() == nil
    var sessions []SessionInfo
    if connected {
        sessions, _ = a.ListSessions()
    }
    a.updateTray(sessions, connected)
}
```

### Pattern 4: Monochrome Tray Icon Asset

**What:** A 18x18 PNG with alpha channel where the icon is **black** pixels on transparent background. macOS `setTemplate:YES` inverts to white in dark menu bars automatically.

**Generation approach:** Use `build/gen_icon.go` pattern — add a `build/gen_tray_icon.go` (build-ignore) that programmatically creates an 18x18 PNG. Or use ImageMagick to convert the existing 16x16.png:

```bash
# Extract grayscale + alpha from existing iconset 16x16, resize to 18x18
convert build/AppIcon.iconset/icon_16x16.png \
    -resize 18x18 \
    -colorspace Gray \
    -alpha on \
    -channel Alpha -separate -negate -compose Copy_Opacity -composite \
    assets/tray_icon.png
```

The result is embedded in `tray.go` via `//go:embed assets/tray_icon.png`.

For the error state icon, a simpler approach: create a solid circle or "!" mark.

### Pattern 5: NSStatusItem Dynamic Update Functions

**What:** Add C functions to tray.go cgo block for updating tooltip, icon, and rebuilding menu from Go.

```objc
// Update tooltip from Go goroutine
static void updateTrayTooltip(const char *tooltip) {
    NSString *tip = [NSString stringWithUTF8String:tooltip];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem != nil) {
            statusItem.button.toolTip = tip;
        }
    });
}

// Swap icon (normal vs error)
static void updateTrayIcon(const void *iconData, int iconLen) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem == nil) return;
        NSData *data = [NSData dataWithBytes:iconData length:iconLen];
        NSImage *icon = [[NSImage alloc] initWithData:data];
        [icon setSize:NSMakeSize(18, 18)];
        [icon setTemplate:YES];
        statusItem.button.image = icon;
    });
}

// Update session names for menu delegate
static void setTraySessionNames(const char **names, int count) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSMutableArray *arr = [NSMutableArray arrayWithCapacity:count];
        for (int i = 0; i < count; i++) {
            [arr addObject:[NSString stringWithUTF8String:names[i]]];
        }
        menuSessions = [arr copy];
    });
}
```

### Anti-Patterns to Avoid

- **Adding systray/fyne.io/systray**: Confirmed AppDelegate symbol conflict at link time. Do not add.
- **Calling NSStatusItem APIs from a non-main thread without dispatch_async**: Crashes. All Cocoa UI calls must dispatch to main queue.
- **Setting LSUIElement in Info.dev.plist**: STATE.md decision — dev mode should keep Dock icon for debugging. Only add to `build/darwin/Info.plist`.
- **Polling sessions too frequently**: 5-second interval is appropriate; 500ms would cause unnecessary daemon API load.
- **Relying on `beforeClose` to stop daemon**: `beforeClose` returns `true` (prevents quit) — it correctly hides the window. Do not add daemon shutdown there. Only Quit triggers daemon stop.
- **Using `runtime.EventsEmit` in cgo callback**: The cgo callback thread is not safe for Wails runtime calls except `runtime.WindowShow` and `runtime.Quit`. Use a channel to hand off to a Go goroutine if needed.
- **Connection reset error from ShutdownDaemon**: Expected — daemon process exits, causing the TCP connection to reset before response completes. Treat this as success, not error.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Light/dark menu bar adaptation | Custom dark-mode detection | `[icon setTemplate:YES]` | Already set; macOS handles inversion automatically |
| System tray | Any third-party systray lib | Existing custom cgo NSStatusItem | Symbol conflict confirmed; existing code works |
| Daemon process management | PID file + SIGTERM | `POST /shutdown` API endpoint | No PID infrastructure; API is consistent with existing patterns |
| Icon resize for tray | Manual pixel manipulation | `sips` or ImageMagick from iconset | `icon_16x16.png` already exists in iconset |

**Key insight:** The existing cgo scaffold handles the hardest parts (NSStatusItem lifecycle, main thread dispatch, light/dark adaptation). Phase 41 is primarily extension work on tray.go, not new infrastructure.

## Common Pitfalls

### Pitfall 1: LSUIElement Removes Cmd+Tab Access Too
**What goes wrong:** Adding `LSUIElement = true` to Info.plist hides the app from both the Dock AND Cmd+Tab / Mission Control.
**Why it happens:** `LSUIElement` classifies the app as a "background application" — no Dock presence, no window switcher presence.
**How to avoid:** This is the intended behavior per TRAY-05 ("app lives in menu bar only"). STATE.md confirms: "LSUIElement removes app from Cmd+Tab entirely — confirm this product behavior is acceptable." This was pre-approved.
**Warning signs:** Users cannot switch to AgentHub via Cmd+Tab — they must click the menu bar icon. This is by design.

### Pitfall 2: Wails Dev Mode Won't Show Tray (wails dev vs wails build)
**What goes wrong:** `wails dev` uses `Info.dev.plist` which will NOT have LSUIElement. The app appears in the Dock in dev mode. Also, `wails dev` rebuilds the frontend but does NOT rebuild cgo — changes to tray.go require `wails build` to test.
**Why it happens:** `wails dev` uses a different plist and hot-reloads frontend only.
**How to avoid:** Test tray behavior only with `wails build` production builds. STATE.md confirms: "tray and ICNS changes require `wails build` production test cycle."

### Pitfall 3: cgo Callbacks Must Not Block Main Thread
**What goes wrong:** If `onTrayShow` or `onTrayQuit` calls long-running operations on the main thread, the app hangs.
**Why it happens:** NSMenu callbacks arrive on the main thread. `runtime.WindowShow` and `runtime.Quit` trigger UI operations.
**How to avoid:** Keep cgo callbacks minimal. For ShutdownDaemon (HTTP call), call it from a goroutine inside the cgo export:
```go
//export onTrayQuit
func onTrayQuit() {
    go func() {
        if trayCallbackApp != nil && trayCallbackApp.client != nil {
            _ = trayCallbackApp.client.ShutdownDaemon()
        }
        if trayCallbackApp != nil && trayCallbackApp.ctx != nil {
            runtime.Quit(trayCallbackApp.ctx)
        }
    }()
}
```

### Pitfall 4: NSMenu Rebuild in Wrong Thread
**What goes wrong:** Calling `setTraySessionNames` or `updateTrayIcon` from a Go goroutine without `dispatch_async` crashes with `[NSMenu]` thread-safety violations.
**Why it happens:** Cocoa requires UI operations on the main thread.
**How to avoid:** All C functions that touch NSStatusItem or NSMenu MUST wrap in `dispatch_async(dispatch_get_main_queue(), ^{ ... })`. Already established pattern in `initStatusItem` and `removeStatusItem`.

### Pitfall 5: NSMenuDelegate menuWillOpen vs Static Menu
**What goes wrong:** If sessions are added to a static NSMenu array without delegate, the menu shows stale data from last rebuild.
**Why it happens:** NSMenu items are rebuilt once, not on each open.
**How to avoid:** Use `NSMenuDelegate menuWillOpen:` to rebuild session items just before the menu is shown. The delegate has access to the C global `menuSessions` array updated by Go.

### Pitfall 6: Daemon Shutdown Race on Tray Quit
**What goes wrong:** GUI calls `runtime.Quit` before `ShutdownDaemon()` returns, and the daemon keeps running.
**Why it happens:** `runtime.Quit` triggers async app termination.
**How to avoid:** Call `ShutdownDaemon()` synchronously before `runtime.Quit`. The 50ms sleep in the daemon's shutdown handler ensures the HTTP 204 response flushes before `os.Exit`. Accept connection-reset errors as success.

### Pitfall 7: `trayCallbackApp.ctx` nil in Tests
**What goes wrong:** `onTrayQuit` dereferences `trayCallbackApp.ctx` which is nil in unit tests.
**Why it happens:** `trayCallbackApp` is a global set at init time; tests don't call `initTray`.
**How to avoid:** Existing nil checks (`if trayCallbackApp != nil && trayCallbackApp.ctx != nil`) already guard this. Maintain the pattern in all new cgo exports.

## Code Examples

### Add LSUIElement to Info.plist (TRAY-05)
```xml
<!-- build/darwin/Info.plist — add inside the top-level <dict> -->
<key>LSUIElement</key>
<true/>
```
**Note:** Add to `build/darwin/Info.plist` ONLY. Not `build/darwin/Info.dev.plist`.

### Monochrome Tray Icon Generation
```bash
# Run once to create assets/tray_icon.png from existing iconset
# Uses sips (macOS built-in) to extract grayscale+alpha 18x18 from 16x16@2x iconset
convert build/AppIcon.iconset/icon_16x16@2x.png \
    -resize 18x18 \
    -colorspace Gray \
    -background none \
    assets/tray_icon.png
```
Then embed in `tray.go`:
```go
//go:embed assets/tray_icon.png
var trayIconBytes []byte

//go:embed assets/tray_icon_error.png
var trayIconErrorBytes []byte
```

### UpdateTray Go function
```go
// tray.go (darwin build tag)
func (a *App) updateTray(sessions []SessionInfo, connected bool) {
    // Update icon
    if connected {
        ptr := unsafe.Pointer(&trayIconBytes[0])
        C.updateTrayIcon(ptr, C.int(len(trayIconBytes)))
    } else {
        ptr := unsafe.Pointer(&trayIconErrorBytes[0])
        C.updateTrayIcon(ptr, C.int(len(trayIconErrorBytes)))
    }

    // Update tooltip
    n := len(sessions)
    var tip string
    switch n {
    case 0:
        tip = "AgentHub — no sessions"
    case 1:
        tip = "AgentHub — 1 session"
    default:
        tip = fmt.Sprintf("AgentHub — %d sessions", n)
    }
    C.updateTrayTooltip(C.CString(tip))

    // Update session names for menu delegate
    // ... pass names as C string array
}
```

### NSMenuDelegate session rebuild (Objective-C in cgo)
```objc
// Inside tray.go cgo block
@interface AgentHubMenuDelegate : NSObject <NSMenuDelegate>
@end

@implementation AgentHubMenuDelegate
- (void)menuWillOpen:(NSMenu *)menu {
    [menu removeAllItems];

    NSMenuItem *openItem = [[NSMenuItem alloc] initWithTitle:@"Open AgentHub"
        action:@selector(showClicked:) keyEquivalent:@""];
    openItem.target = statusItem;
    [menu addItem:openItem];

    if (menuSessions != nil && menuSessions.count > 0) {
        [menu addItem:[NSMenuItem separatorItem]];
        for (NSUInteger i = 0; i < menuSessions.count; i++) {
            NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:menuSessions[i]
                action:@selector(sessionClicked:) keyEquivalent:@""];
            item.target = statusItem;
            item.tag = (NSInteger)i;
            [menu addItem:item];
        }
    }

    [menu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit"
        action:@selector(quitClicked:) keyEquivalent:@""];
    quitItem.target = statusItem;
    [menu addItem:quitItem];
}
@end
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Third-party systray libs | Custom cgo NSStatusItem | Phase 41 design | Avoids AppDelegate symbol conflict |
| Wails default app lifecycle (quit on close) | HideWindowOnClose + beforeClose | Already implemented | Window hides, daemon persists |
| appicon.png as tray icon (256x256 RGB) | Dedicated 18x18 monochrome alpha PNG | Phase 41 | BRND-03: correct template behavior |

**What's already done (no work needed):**
- `HideWindowOnClose: true` in `main.go` — DMGR-01 window behavior
- `beforeClose` returning `true` — prevents default Wails quit
- `[icon setTemplate:YES]` in tray.go cgo — handles light/dark bar adaptation
- `initTray()` / `cleanupTray()` called from `startup()` / `shutdown()`
- `tray_linux.go` / `tray_windows.go` no-op stubs for cross-compile

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| macOS Cocoa SDK | tray.go cgo | ✓ | macOS 25.3.0 | — |
| ImageMagick `convert` | Tray icon generation | ✓ | 7.1.2-18 | Use `sips` built-in |
| `sips` | Tray icon generation | ✓ | 316 | Use ImageMagick |
| Wails v2 CLI | Production build testing | ✓ | v2.10.2 | — |
| Go 1.26+ | Build | ✓ | 1.26.1 (go.mod) | — |

**Missing dependencies with no fallback:** None.

**Note on testing:** Testing tray changes requires `wails build` (production build), not `wails dev`. Per STATE.md: "tray and ICNS changes require `wails build` production test cycle."

## Validation Architecture

> `workflow.nyquist_validation` key is absent from `.planning/config.json` — treating as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./... -run TestHide -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DMGR-01 | `beforeClose` returns true AND does not kill sessions | unit | `go test . -run TestBeforeCloseReturnsTrue -v` | ✅ tray_test.go |
| DMGR-01 | Sessions survive `beforeClose` | unit | `go test . -run TestHideWindowSessionsAlive -v` | ✅ tray_test.go |
| DMGR-02 | `ShutdownDaemon` calls `/shutdown` endpoint | unit | `go test ./internal/daemon/... -run TestShutdownDaemon -v` | ❌ Wave 0 |
| DMGR-02 | `onTrayQuit` calls `ShutdownDaemon` before `runtime.Quit` | unit | `go test . -run TestTrayQuitShutdownsDaemon -v` | ❌ Wave 0 |
| TRAY-03/06 | `updateTray(sessions, connected)` updates tooltip + icon fields | unit | `go test . -run TestUpdateTray -v` | ❌ Wave 0 |
| TRAY-05 | Info.plist contains LSUIElement | manual | Inspect `build/darwin/Info.plist` + built app plist | manual-only |
| BRND-03 | `assets/tray_icon.png` exists and is < 32x32 with alpha | unit | `go test . -run TestTrayIconAsset -v` | ❌ Wave 0 |

**Manual-only justification (TRAY-05):** Info.plist changes only take effect in a full `wails build` production build and require visual inspection of the running app (no Dock icon visible).

**NSStatusItem calls are untestable in unit tests** — Cocoa requires a display server. Tests verify Go-side behavior (data passed to C functions) through testable wrapper functions, not actual NSStatusItem calls.

### Sampling Rate
- **Per task commit:** `go test ./...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `tray_test.go` — add `TestUpdateTray`, `TestTrayQuitShutdownsDaemon`, `TestTrayIconAsset`
- [ ] `internal/daemon/client_test.go` — add `TestShutdownDaemon`

## Open Questions

1. **Session click in tray menu → which GUI action?**
   - What we know: TRAY-04 says "clicking a session focuses it in the GUI"
   - What's unclear: Should this switch to the tab for that session, or just show the window? Tab switching requires the frontend to handle a `tray:focus-session` Wails event.
   - Recommendation: Emit `tray:focus-session` event with session ID; the frontend's existing `setActiveId` handles tab focus. No new frontend code needed if the event maps to the session's tab ID.

2. **Error icon visual design**
   - What we know: TRAY-03 requires "error/disconnected visual state"
   - What's unclear: What should the error icon look like? A badge? A different glyph?
   - Recommendation: Use a simple "!" or "×" overlaid on the standard tray icon, or generate a separate icon with a dot. Since this is monochrome template, a hollow circle (disconnected) vs filled circle (connected) or "!" is simplest to generate programmatically.

3. **Info.dev.plist and LSUIElement during development**
   - What we know: STATE.md says "LSUIElement in Info.plist only (not Info.dev.plist)"
   - What's unclear: The existing `Info.dev.plist` is a template with Wails `{{.}}` placeholders — confirming LSUIElement is NOT there matches the decision.
   - Recommendation: Follow STATE.md. LSUIElement in production `Info.plist` only. Dev mode shows Dock icon for easier debugging.

## Sources

### Primary (HIGH confidence)
- Direct code inspection: `/Users/ken/dev/agenthub/tray.go` — existing cgo implementation
- Direct code inspection: `/Users/ken/dev/agenthub/app.go` — `startup`, `shutdown`, `beforeClose`
- Direct code inspection: `/Users/ken/dev/agenthub/main.go` — `HideWindowOnClose: true`
- Direct code inspection: `/Users/ken/dev/agenthub/internal/daemon/api.go` — no `/shutdown` route confirmed
- Direct code inspection: `/Users/ken/dev/agenthub/internal/daemon/process.go` — daemon is a detached process
- Direct code inspection: `/Users/ken/dev/agenthub/build/darwin/Info.plist` — no LSUIElement confirmed
- Apple NSStatusItem documentation (macOS SDK) — `setTemplate:YES` behavior for light/dark
- `/Users/ken/dev/agenthub/.planning/STATE.md` decisions section

### Secondary (MEDIUM confidence)
- `sips -g all assets/appicon.png` — confirmed color_type=2 (RGB, no alpha)
- `go test ./...` — all existing tests pass (5.2s baseline)
- ImageMagick 7.1.2 available at `/opt/homebrew/bin/convert`

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — existing code confirmed, no new dependencies
- Architecture: HIGH — patterns follow existing cgo scaffold exactly
- Pitfalls: HIGH — cgo thread safety rules are well-established; symbol conflict risk documented in STATE.md
- Test mapping: HIGH — existing tests found and confirmed passing

**Research date:** 2026-04-02
**Valid until:** 2026-05-02 (stable macOS APIs, no external dependencies)
