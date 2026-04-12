# Phase 67: Cross-Platform System Tray — Research

**Researched:** 2026-04-11
**Phase Goal:** Linux and Windows users can interact with AgentHub through a system tray icon that matches macOS tray functionality

## Executive Summary

The macOS tray is a mature cgo/Objective-C implementation using NSStatusBar with dynamic NSMenuDelegate, icon state toggling, and Wails event integration. Linux and Windows have no-op stubs (`tray_linux.go`, `tray_windows.go`) that implement the same 4-method interface. The task is to replace those stubs with real platform-native implementations that replicate the macOS behavior without touching the darwin codepath.

## Current Architecture

### Interface Contract (all platforms must implement)

```go
func (a *App) initTray()                                         // Create tray icon
func (a *App) cleanupTray()                                      // Remove tray icon
func (a *App) updateTray(sessions []SessionInfo, connected bool)  // Update icon + menu
func (a *App) setDockVisible(visible bool)                        // Platform-specific (no-op on Linux/Windows)
```

### macOS Implementation (tray.go + tray_objc_darwin.m)

- **Icon**: 18x18 PNG, embedded via `//go:embed`, template mode for light/dark adaptation
- **Menu**: NSMenuDelegate rebuilds on every menu open (no stale data)
- **Structure**: "Open AgentHub" → separator → session list → separator → "Quit"
- **Callbacks**: cgo exports `onTrayShow()`, `onTrayQuit()`, `onTraySession(idx)` — all dispatch to goroutines for thread safety
- **Icon states**: `tray_icon.png` (connected) and `tray_icon_error.png` (disconnected/error)
- **Polling**: `startTrayPoller()` calls `refreshTrayState()` every 5s, which calls `updateTray()`
- **Hide-on-close**: `beforeClose()` returns `true` to prevent quit, hides window + dock icon
- **Quit**: `onTrayQuit()` sets `a.quitting = true`, calls `ShutdownDaemon()`, then `runtime.Quit()`
- **Session focus**: `onTraySession(idx)` → `ListSessions()` → `EventsEmit("tray:focus-session", id)`

### Embedded Icons

- `assets/tray_icon.png` — 18x18 PNG, 507 bytes (monochrome, macOS template)
- `assets/tray_icon_error.png` — 18x18 PNG, 260 bytes (monochrome, macOS template)
- Both embedded via `//go:embed` in `tray.go` (darwin build tag)

### App Lifecycle (app.go)

1. `startup()` → `initTray()` immediately → `EnsureDaemon()` → `startTrayPoller()`
2. `beforeClose()` → if `!a.quitting`: hide window + dock, return true (prevent quit)
3. `shutdown()` → `cleanupTray()` → daemon stays alive
4. `refreshTrayState()` → if `client == nil`: `updateTray(nil, false)` (error icon); else: fetch sessions, `updateTray(sessions, connected)`

### Existing Dependencies

- `github.com/godbus/dbus/v5 v5.2.0` — already in go.mod (indirect, from Tailscale). Can be promoted to direct for Linux tray.
- No Win32 API wrappers in go.mod (will use `golang.org/x/sys/windows` or `syscall`)

### Build System

- macOS: `wails build -platform darwin/universal`
- Windows: `CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 wails build -platform windows/amd64`
- Linux: Docker (golang:1.26-bookworm) with `libgtk-3-dev libwebkit2gtk-4.0-dev`
- Build tags: `//go:build darwin|linux|windows`

## Critical Constraint

**fyne.io/systray is rejected** — causes Wails AppDelegate duplicate symbol conflict (documented in STATE.md). The solution MUST use platform-native APIs directly, keeping the existing per-platform file pattern.

## Linux Implementation Approach

### Protocol: D-Bus StatusNotifierItem (SNI)

The modern Linux system tray standard. Supported by KDE Plasma, GNOME (via AppIndicator extension), XFCE (with `xfce4-statusnotifier-plugin`).

**How it works:**
1. Register a D-Bus service implementing `org.kde.StatusNotifierItem` interface
2. Register with `org.kde.StatusNotifierWatcher` (the tray host)
3. Expose properties: `IconName`/`IconPixmap`, `ToolTip`, `Menu` (via `com.canonical.dbusmenu`)
4. Menu is a separate D-Bus object implementing `com.canonical.dbusmenu` interface

**Library: `github.com/godbus/dbus/v5`** — already in go.mod as indirect dependency.

### D-Bus Interfaces Required

```
org.kde.StatusNotifierItem:
  - Properties: Id, Category, Status, IconPixmap, ToolTip, Menu
  - Methods: Activate(), ContextMenu(), SecondaryActivate()
  - Signals: NewIcon, NewToolTip, NewStatus

com.canonical.dbusmenu:
  - Properties: Version, TextDirection, Status, IconThemePath, Children
  - Methods: GetLayout(), GetGroupProperties(), Event(), EventGroup()
  - Signals: LayoutUpdated, ItemsPropertiesUpdated
```

### Icon Format

- D-Bus StatusNotifierItem accepts `IconPixmap` as `a(iiay)` — array of (width, height, ARGB32 pixel data)
- Can also use `IconName` for theme-aware icons, but custom PNG is more reliable
- The existing 18x18 PNGs can be decoded to ARGB32 at runtime (use `image/png` → extract pixels)
- Recommended tray size: 22x22 or 24x24 for most Linux desktop environments

### Menu Implementation

The D-Bus menu (`com.canonical.dbusmenu`) protocol:
- Menus are trees of items, each with an ID, label, type (standard/separator), and properties
- Updates pushed via `LayoutUpdated` signal (trigger when sessions change)
- Menu structure mirrors macOS: "Open AgentHub" → separator → sessions → separator → "Quit"
- Click events arrive via `Event()` method with item ID

### setDockVisible on Linux

- Linux doesn't have a macOS-style Dock. `setDockVisible()` is a no-op.
- Taskbar visibility is managed by the window manager (Wails handles `WindowHide`/`WindowShow`).

### Fallback: XEmbed (Legacy)

Older tray protocol. GNOME dropped it in 3.26. Still works on some WMs (i3, openbox). Not worth implementing as primary — SNI covers the required desktop environments (GNOME/KDE/XFCE per TRAY-01).

### Linux Testing Considerations

- No display server in CI → cannot test D-Bus tray registration in headless environments
- Unit-testable: icon format conversion, menu data structures, tooltip formatting
- Integration testing requires a running D-Bus session bus (can mock with `dbus-test-runner`)

## Windows Implementation Approach

### API: Shell_NotifyIcon (Win32 Shell Notifications)

The standard Windows notification area (system tray) API since Windows XP.

**How it works:**
1. Create a hidden window (HWND) to receive tray messages
2. Call `Shell_NotifyIconW(NIM_ADD, &NOTIFYICONDATA)` to add icon
3. Handle `WM_TRAYICON` custom messages for click events
4. Right-click → build and display popup menu with `TrackPopupMenu()`
5. Call `Shell_NotifyIconW(NIM_MODIFY, ...)` to update icon/tooltip
6. Call `Shell_NotifyIconW(NIM_DELETE, ...)` to remove on cleanup

**API access:** Use `golang.org/x/sys/windows` for syscall wrappers, or raw `syscall.NewLazyDLL("shell32.dll")`.

### Key Win32 Functions

```
Shell_NotifyIconW     — shell32.dll (add/modify/delete tray icon)
CreateWindowExW       — user32.dll (hidden message-only window)
RegisterClassExW      — user32.dll (register window class)
LoadImage             — user32.dll (load .ico from memory)
CreatePopupMenu       — user32.dll (context menu)
TrackPopupMenu        — user32.dll (show context menu at cursor)
GetCursorPos          — user32.dll (cursor position for menu placement)
DestroyMenu           — user32.dll (cleanup)
PostQuitMessage       — user32.dll (exit message loop)
```

### Icon Format

- Windows system tray requires `.ico` format (HICON handle)
- Options:
  1. **Embed .ico files** alongside existing PNGs (cleanest)
  2. **Convert PNG to HICON at runtime** using `CreateIconFromResourceEx` or GDI+ (more complex)
- Recommended: embed separate `.ico` files in `assets/` — simpler, no runtime conversion
- Windows tray icons are typically 16x16 or 32x32 (system scales automatically)

### Message Loop

Windows tray requires a message pump. Options:
1. **Goroutine with GetMessage loop** — dedicated goroutine calls `GetMessage`/`DispatchMessage` in a loop
2. **Use the Wails window's message loop** — risky, interleaves with Wails event handling
3. **Create a message-only window** (HWND_MESSAGE parent) — cleanest isolation

Recommended: Option 3 (message-only window) with its own goroutine for the message pump. The goroutine runs `GetMessage` and dispatches tray events. Menu item clicks post to a Go channel that the main app reads.

### Menu Implementation

- `CreatePopupMenu()` + `AppendMenuW()` to build menu items
- `TrackPopupMenu()` to show on right-click
- Handle `WM_COMMAND` messages for item selection
- Rebuild menu on each right-click (like macOS — always fresh session data)

### setDockVisible on Windows

- Windows doesn't have a macOS-style Dock. `setDockVisible()` is a no-op.
- Taskbar button visibility is managed by the window manager (Wails handles window show/hide).

### Windows Build Considerations

- Already cross-compiled with MinGW-w64: `CC=x86_64-w64-mingw32-gcc`
- CGO_ENABLED=1 already set for Windows builds
- Win32 API calls via `syscall` don't require additional C code
- `golang.org/x/sys/windows` provides typed wrappers (cleaner than raw syscall)

## Shared Concerns

### Icon Asset Strategy

| Platform | Format | Size | Source |
|----------|--------|------|--------|
| macOS | PNG (template) | 18x18 | `assets/tray_icon.png`, `assets/tray_icon_error.png` |
| Linux | PNG → ARGB32 | 22x22 | New `assets/tray_icon_linux.png`, `assets/tray_icon_error_linux.png` (or reuse 18x18 and let DE scale) |
| Windows | ICO | 16x16+32x32 | New `assets/tray_icon.ico`, `assets/tray_icon_error.ico` |

**Recommendation:** Create new icon assets at appropriate sizes per platform. The 18x18 macOS template icons won't look right on other platforms — they're designed for macOS Retina menu bar rendering.

Alternative: Reuse the existing 18x18 PNGs and convert/scale at runtime. Simpler but may produce blurry icons on Windows/Linux.

### Tooltip Format (shared)

All platforms use the same `trayTooltip(n int) string` function:
- "AgentHub — no sessions" (U+2014 em dash)
- "AgentHub — 1 session"
- "AgentHub — N sessions"

Currently defined in `tray.go` (darwin only). Must be extracted to a shared file or duplicated per platform.

### Menu Structure (shared)

All platforms:
1. "Open AgentHub" — show window
2. Separator
3. Dynamic session list (if any sessions exist)
4. Separator
5. "Quit" — shutdown daemon + exit

### Callback Pattern (shared)

All platforms use the same callback logic:
- **Show**: `runtime.WindowShow(ctx)` + `setDockVisible(true)`
- **Quit**: `client.ShutdownDaemon()` → `a.quitting = true` → `runtime.Quit(ctx)`
- **Session**: `ListSessions()` → index lookup → `EventsEmit("tray:focus-session", id)`

### Hide-on-Close (Wails handles it)

`beforeClose()` in `app.go` already returns `true` to prevent quit. This is platform-independent. On Linux/Windows, it hides the window but the tray icon keeps the process alive. `setDockVisible()` is no-op on non-macOS.

The Wails config already has `HideWindowOnClose: true` which reinforces this behavior.

## Validation Architecture

### Testable Units

1. **Icon conversion** (Linux ARGB32, Windows ICO) — pure function tests
2. **Menu data structure** — verify items, separators, session count
3. **Tooltip formatting** — already tested, extract to shared
4. **D-Bus message construction** (Linux) — mock bus
5. **NOTIFYICONDATA struct population** (Windows) — struct validation

### Integration Tests

- Linux: Requires D-Bus session bus (available in CI Docker with `dbus-launch`)
- Windows: Requires Win32 APIs (only testable on Windows CI)
- Both: The actual tray rendering requires a display server

### Cross-Platform Test Strategy

- Extract shared logic (tooltip, menu data, callbacks) to `tray_common.go` (no build tag)
- Platform-specific files test their own icon format handling
- Existing `tray_test.go` (darwin) patterns serve as reference

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| D-Bus SNI not available on minimal Linux | Medium | Low | Fail gracefully — log warning, tray functions become no-ops |
| Windows message loop conflicts with Wails | Low | High | Use message-only window (HWND_MESSAGE) on separate goroutine |
| Icon rendering quality on different DPIs | Medium | Low | Provide multiple icon sizes; let platform scale |
| godbus/dbus version incompatibility | Low | Low | Already v5.2.0 in go.mod; pin version |
| Cross-compilation issues for Linux cgo | Low | Medium | Docker build already handles GTK deps; D-Bus headers available in Debian |

## Recommended Implementation Order

1. **Shared code extraction** — Move `trayTooltip()` and callback logic to `tray_common.go`
2. **Linux tray** — D-Bus SNI + dbusmenu using godbus (already a dependency)
3. **Windows tray** — Shell_NotifyIcon via syscall/x/sys/windows
4. **Icon assets** — Create platform-appropriate icons
5. **Tests** — Unit tests per platform, integration tests where CI allows

## RESEARCH COMPLETE
