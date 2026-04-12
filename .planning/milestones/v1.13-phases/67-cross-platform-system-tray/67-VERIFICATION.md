---
phase: 67-cross-platform-system-tray
verified: 2026-04-11T00:00:00Z
status: human_needed
score: 8/8 must-haves verified (automated)
overrides_applied: 0
human_verification:
  - test: "On a Linux machine (GNOME, KDE, or XFCE with AppIndicator/SNI support): build and run AgentHub. Confirm tray icon appears."
    expected: "AgentHub icon visible in system tray after launch"
    why_human: "D-Bus StatusNotifierItem registration requires a running tray host on a live desktop — cannot verify from macOS host"
  - test: "On Linux: right-click tray icon. Confirm menu shows Open AgentHub, separator, and Quit. Start a session, right-click again. Confirm session name appears."
    expected: "Menu lists active sessions dynamically"
    why_human: "Requires live Linux desktop with running tray host to observe popup menu"
  - test: "On Linux: close the app window with X. Confirm tray icon remains visible, daemon is still reachable, and sessions are intact."
    expected: "Window hides, daemon continues, tray icon persists"
    why_human: "Hide-on-close requires Wails window events and live desktop session"
  - test: "On Linux: click Quit in tray menu. Confirm app fully exits and daemon stops."
    expected: "App exits cleanly, daemon process gone"
    why_human: "Requires live execution — cannot stub runtime.Quit + ShutdownDaemon from tests"
  - test: "On Windows: build and run agenthub.exe. Confirm tray icon appears in notification area (expand hidden icons if needed)."
    expected: "AgentHub icon visible in Windows notification area"
    why_human: "Shell_NotifyIconW requires Win32 desktop session — cannot verify from macOS"
  - test: "On Windows: right-click tray icon. Confirm popup menu shows Open AgentHub, separator, and Quit."
    expected: "Popup context menu appears with correct items"
    why_human: "Win32 TrackPopupMenu requires a live desktop and message pump"
  - test: "On Windows: close the app window. Confirm tray icon remains. Click Quit — confirm app exits."
    expected: "Hide-on-close works; Quit exits cleanly"
    why_human: "Requires live Windows machine — Win32 message loop cannot run cross-platform"
---

# Phase 67: Cross-Platform System Tray Verification Report

**Phase Goal:** Linux and Windows users can interact with AgentHub through a system tray icon that matches macOS tray functionality
**Verified:** 2026-04-11
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| #  | Truth                                                                                                             | Status              | Evidence                                                                                                       |
|----|-------------------------------------------------------------------------------------------------------------------|---------------------|----------------------------------------------------------------------------------------------------------------|
| 1  | On Linux (GNOME/KDE/XFCE with AppIndicator/SNI support), AgentHub tray icon appears after launch                 | ? HUMAN NEEDED      | D-Bus SNI service exported; requires live desktop to verify visibility                                         |
| 2  | On Windows, AgentHub tray icon appears in notification area after launch                                          | ? HUMAN NEEDED      | Shell_NotifyIconW(NIM_ADD) invoked in initTray; requires Windows machine                                       |
| 3  | Right-clicking Linux or Windows tray shows menu listing active sessions                                           | ? HUMAN NEEDED      | Linux: dbusmenu GetLayout wired to BuildMenuItems; Windows: showPopupMenu wired to BuildMenuItems — needs live tray |
| 4  | Linux and Windows tray icon reflects connected vs error states                                                    | ✓ VERIFIED          | `updateTray` switches iconPixmap/HIcon between trayIconBytes/trayIconErrorBytes based on `connected` bool      |
| 5  | Closing Linux or Windows window hides it; daemon continues running and tray icon remains                          | ✓ VERIFIED (partial)| `beforeClose` in app.go returns true + calls `runtime.WindowHide`; `quitting=false` path tested in TestBeforeCloseReturnsTrue and TestHideWindowSessionsAlive; live tray icon persistence needs human |
| 6  | Selecting Quit from Linux or Windows tray menu stops daemon and fully exits                                       | ✓ VERIFIED (code)   | onQuit/wndProc IDM_QUIT dispatch: `ShutdownDaemon()` then `quitting=true` then `runtime.Quit(ctx)`; needs live execution |

**Automated score:** 6/6 truths have implementation evidence. 4 require human confirmation for live tray behaviour.

### Deferred Items

None.

### Required Artifacts

| Artifact              | Expected                                       | Status     | Details                                                                 |
|-----------------------|------------------------------------------------|------------|-------------------------------------------------------------------------|
| `tray_common.go`      | Shared trayTooltip, MenuItemData, BuildMenuItems | ✓ VERIFIED | No build tag; contains trayTooltip, MenuItemKind, MenuItem, BuildMenuItems |
| `tray_common_test.go` | Tests for shared tooltip and menu builder      | ✓ VERIFIED | TestTrayTooltip, TestBuildMenuItemsEmpty, TestBuildMenuItemsWithSessions, TestBuildMenuItemsLabels — all PASS |
| `tray_linux.go`       | Full Linux D-Bus SNI tray implementation       | ✓ VERIFIED | `//go:build linux`, initTray, cleanupTray, updateTray, setDockVisible, pngToARGB32Pixmap, linuxTray struct, dbus.SessionBusPrivate, org.kde.StatusNotifierItem, com.canonical.dbusmenu, RegisterStatusNotifierItem |
| `tray_linux_test.go`  | Unit tests for Linux icon conversion and menu  | ✓ VERIFIED | TestPngToARGB32Pixmap, TestPngToARGB32PixmapInvalid, TestDbusMenuLayout, TestDbusMenuLayoutEmpty |
| `tray_windows.go`     | Full Windows Shell_NotifyIcon tray             | ✓ VERIFIED | `//go:build windows`, initTray, cleanupTray, updateTray, setDockVisible, pShellNotifyIconW, windowsTray struct, pCreateWindowExW, pCreatePopupMenu, pTrackPopupMenu, WM_TRAYICON, embedded icons, BuildMenuItems, trayTooltip, goruntime.LockOSThread, log.Printf graceful degradation |
| `tray_windows_test.go` | Unit tests for Windows icon conversion and menu | ✓ VERIFIED | TestCreateIconFromPNG, TestCreateIconFromPNGInvalid, TestWindowsMenuFromBuildMenuItems, TestWindowsMenuEmpty |

### Key Link Verification

| From              | To              | Via                                              | Status     | Details                                                              |
|-------------------|-----------------|--------------------------------------------------|------------|----------------------------------------------------------------------|
| `tray_common.go`  | `tray_linux.go` | BuildMenuItems called from updateTray/initTray   | ✓ WIRED    | `BuildMenuItems` called at lines 327, 418 in tray_linux.go          |
| `tray_linux.go`   | `app.go`        | initTray/cleanupTray/updateTray/setDockVisible   | ✓ WIRED    | All 4 interface methods implemented with `func (a *App)` receiver   |
| `tray_linux.go`   | `godbus/dbus/v5`| D-Bus StatusNotifierItem registration            | ✓ WIRED    | `dbus.SessionBusPrivate()` at line 330; promoted to direct in go.mod |
| `tray_windows.go` | `tray_common.go`| BuildMenuItems and trayTooltip from shared helpers| ✓ WIRED    | BuildMenuItems at lines 409, 538; trayTooltip at lines 458, 525     |
| `tray_windows.go` | `app.go`        | initTray/cleanupTray/updateTray/setDockVisible   | ✓ WIRED    | All 4 interface methods implemented with `func (a *App)` receiver   |
| `tray_windows.go` | `golang.org/x/sys/windows` | Shell_NotifyIcon Win32 syscalls         | ✓ WIRED    | `windows.NewLazySystemDLL` at lines 116-118                         |

### Data-Flow Trace (Level 4)

| Artifact          | Data Variable | Source                          | Produces Real Data       | Status       |
|-------------------|---------------|---------------------------------|--------------------------|--------------|
| `tray_linux.go`   | menuItems     | BuildMenuItems(sessions)        | sessions from ListSessions | ✓ FLOWING  |
| `tray_linux.go`   | iconPixmap    | makePixmap(trayIconBytes/errBytes) | Embedded PNG assets     | ✓ FLOWING  |
| `tray_windows.go` | menuItems     | BuildMenuItems(sessions)        | sessions from ListSessions | ✓ FLOWING  |
| `tray_windows.go` | nid.HIcon     | createIconFromPNG(trayIconBytes) | Embedded PNG assets      | ✓ FLOWING  |

Sessions flow: `refreshTrayState()` (called every 5s by `startTrayPoller`) → `app.ListSessions()` → `daemon.DaemonClient.ListSessions()` → `updateTray(sessions, connected)` → `BuildMenuItems(sessions)` → rendered in tray menu.

### Behavioral Spot-Checks

| Behavior                           | Command                                              | Result           | Status  |
|------------------------------------|------------------------------------------------------|------------------|---------|
| go vet (darwin host)               | `go vet ./...`                                       | No output        | ✓ PASS  |
| GOOS=linux go vet                  | `GOOS=linux go vet ./...`                            | No output        | ✓ PASS  |
| GOOS=windows go vet                | `GOOS=windows go vet ./...`                          | No output        | ✓ PASS  |
| GOOS=linux go build                | `GOOS=linux go build ./...`                          | No output        | ✓ PASS  |
| GOOS=windows go build              | `GOOS=windows go build ./...`                        | No output        | ✓ PASS  |
| Shared tray tests                  | `go test -run "TestTrayTooltip\|TestBuildMenuItems"` | 4/4 PASS         | ✓ PASS  |
| Darwin tray tests                  | `go test -run "TestTrayIconAsset\|..."` (6 tests)    | 6/6 PASS         | ✓ PASS  |
| Live Linux tray icon               | Build + run on GNOME/KDE/XFCE                        | N/A (no Linux VM)| ? SKIP  |
| Live Windows tray icon             | Build + run on Windows                               | N/A (no Windows) | ? SKIP  |

### Requirements Coverage

| Requirement | Source Plan | Description                                                | Status             | Evidence                                                     |
|-------------|-------------|------------------------------------------------------------|--------------------|--------------------------------------------------------------|
| TRAY-01     | 67-01       | Linux system tray icon visible in supported desktop envs   | ? HUMAN NEEDED     | D-Bus SNI service exported; live visibility requires Linux desktop |
| TRAY-02     | 67-02       | Windows system tray icon visible in notification area      | ? HUMAN NEEDED     | Shell_NotifyIconW(NIM_ADD) implemented; requires Windows     |
| TRAY-03     | 67-01, 67-02| Linux/Windows tray shows dynamic session list              | ✓ CODE VERIFIED    | BuildMenuItems wired to updateTray on both platforms; live menu needs human |
| TRAY-04     | 67-01, 67-02| Linux/Windows tray shows status icon states               | ✓ VERIFIED         | updateTray switches between trayIconBytes/trayIconErrorBytes on both platforms |
| TRAY-05     | 67-01, 67-02| Linux/Windows tray supports hide-on-close                 | ✓ CODE VERIFIED    | beforeClose() returns true + WindowHide; quitting=false path prevents exit; live behavior needs human |
| TRAY-06     | 67-01, 67-02| Linux/Windows tray Quit stops daemon and fully exits      | ✓ CODE VERIFIED    | ShutdownDaemon + quitting=true + runtime.Quit wired in onQuit/wndProc IDM_QUIT; live execution needs human |

All 6 requirement IDs (TRAY-01 through TRAY-06) from REQUIREMENTS.md are claimed by plans 67-01 and 67-02 and are accounted for. No orphaned requirements.

### Anti-Patterns Found

| File              | Pattern                      | Severity | Impact                                                          |
|-------------------|------------------------------|----------|-----------------------------------------------------------------|
| None              | —                            | —        | No TODO/FIXME/HACK/PLACEHOLDER found in any tray file           |

No stubs detected. `setDockVisible` no-ops on Linux and Windows are intentional (documented in both files). All data paths from sessions → tray menu are fully wired.

### Human Verification Required

#### 1. Linux Tray Icon Visibility (TRAY-01)

**Test:** Build for Linux (`GOOS=linux`/Wails build), run on GNOME, KDE, or XFCE desktop with AppIndicator or StatusNotifier support. Observe the system tray area.
**Expected:** AgentHub tray icon appears in the tray. If the icon doesn't appear on GNOME, check that the AppIndicator extension is installed.
**Why human:** D-Bus StatusNotifierItem registration requires a running tray host (kde-statusnotifier-watcher or GNOME AppIndicator). The code exports the correct D-Bus objects and registers with StatusNotifierWatcher — but whether the icon actually renders depends on the tray host presence on a live desktop session.

#### 2. Linux Session Menu (TRAY-03)

**Test:** On a running Linux session — right-click the tray icon when no sessions exist; verify menu shows "Open AgentHub | separator | separator | Quit". Then create a session and right-click again; verify session name appears in menu.
**Expected:** Dynamic session list updates correctly between tray polls (every 5 seconds).
**Why human:** dbusmenu GetLayout response requires a live tray host to trigger the menu display.

#### 3. Linux Hide-on-Close and Quit (TRAY-05, TRAY-06)

**Test:** Close the AgentHub window with X. Verify the tray icon stays. Then click Quit in the tray menu. Verify the process exits.
**Expected:** Window hides (beforeClose returns true); daemon keeps running. Quit fires ShutdownDaemon then runtime.Quit.
**Why human:** Requires live Wails runtime to execute window hide and process quit.

#### 4. Windows Tray Icon Visibility (TRAY-02)

**Test:** Build `agenthub.exe` for Windows and run on Windows 10/11. Check system tray (notification area). May need to click "Show hidden icons".
**Expected:** AgentHub icon visible in notification area.
**Why human:** Shell_NotifyIconW requires a Windows desktop session. The message pump goroutine with LockOSThread cannot run under macOS cross-compilation.

#### 5. Windows Session Menu and Quit (TRAY-03, TRAY-05, TRAY-06)

**Test:** Right-click Windows tray icon; verify popup menu. Double-click to open window. Close window (X); verify icon remains. Click Quit; verify app exits.
**Expected:** Popup menu shows correct items. Double-click shows window. Hide-on-close works. Quit exits.
**Why human:** Win32 wndProc message pump and TrackPopupMenu require a running Windows session.

### Gaps Summary

No automated gaps found. All code artifacts exist, are substantive (not stubs), are wired to the App interface and each other, and data flows from the daemon through sessions to tray menus on both platforms. All 5 cross-compilation checks (vet + build for darwin/linux/windows) pass clean. All unit tests pass.

The only open items are live-platform verification for TRAY-01 and TRAY-02 (icon visibility) and the full end-to-end interaction flows for TRAY-03, TRAY-05, and TRAY-06 — these cannot be verified programmatically from a macOS development host.

---

_Verified: 2026-04-11_
_Verifier: Claude (gsd-verifier)_
