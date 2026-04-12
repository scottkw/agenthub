---
phase: 67-cross-platform-system-tray
plan: 02
subsystem: tray
tags: [windows, win32, shell-notify-icon, system-tray, gdi, hicon]
dependency_graph:
  requires: [67-01]
  provides: [windows-tray]
  affects: [tray_windows.go, tray_windows_test.go]
tech_stack:
  added: []
  patterns: [Shell_NotifyIcon, Win32 GDI CreateDIBSection, HWND_MESSAGE message-only window, runtime PNG-to-HICON conversion, graceful degradation]
key_files:
  created:
    - tray_windows_test.go
  modified:
    - tray_windows.go
decisions:
  - "Runtime PNG-to-HICON conversion via GDI CreateDIBSection + CreateIconIndirect — reuses existing PNG assets, no .ico files needed"
  - "HWND_MESSAGE parent for message-only window — no taskbar button, no screen presence"
  - "LockOSThread in Win32 message pump goroutine — Win32 message loops must run on a fixed OS thread"
  - "menuIDForItem as testable helper — maps MenuItem to IDM_OPEN/IDM_QUIT/IDM_SESSION+index without calling Win32 APIs"
  - "T-67-06 bounds check: menuID-IDM_SESSION validated against len(sessions) before dispatch"
  - "goruntime alias for stdlib runtime to avoid collision with wails/v2/pkg/runtime"
metrics:
  duration_minutes: 3
  completed_date: "2026-04-12"
  tasks_completed: 1
  tasks_pending: 1
  files_created: 1
  files_modified: 1
---

# Phase 67 Plan 02: Windows Shell_NotifyIcon Tray Summary

**One-liner:** Replaced the Windows no-op tray stub with a full Shell_NotifyIcon Win32 API implementation using runtime PNG-to-HICON conversion, a message-only window for event handling, and popup context menus for session navigation.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Add failing Windows tray tests | cbdab6e | tray_windows_test.go |
| 1 (GREEN) | Implement Windows Shell_NotifyIcon tray | 4c7ac65 | tray_windows.go |

## Tasks Pending (awaiting checkpoint)

| Task | Name | Type | Status |
|------|------|------|--------|
| 2 | Verify cross-platform tray on Linux and Windows | checkpoint:human-verify | Awaiting human verification |

## What Was Built

### Task 1: Windows Shell_NotifyIcon Tray

Replaced the 4-line no-op stub in `tray_windows.go` with a full Win32 Shell_NotifyIcon implementation:

**Key components:**

- `createIconFromPNG(pngData []byte) (uintptr, error)` — decodes embedded PNG, creates 32-bit DIB via `CreateDIBSection`, writes BGRA pixels, creates monochrome mask via `CreateBitmap`, assembles HICON via `CreateIconIndirect`. Reuses existing `assets/tray_icon.png` and `assets/tray_icon_error.png` — no `.ico` files needed.

- `windowsTray` struct — holds hwnd, hIcon, hIconErr, nid (NOTIFYICONDATA), menuItems, mutex, app reference, ready channel, and disabled flag.

- `wndProc(hwnd, msg, wParam, lParam uintptr) uintptr` — package-level Win32 window procedure (cannot capture closures). Handles:
  - `WM_TRAYICON` + `WM_RBUTTONUP` → `showPopupMenu()`
  - `WM_TRAYICON` + `WM_LBUTTONDBLCLK` → goroutine show window
  - `WM_COMMAND` + `IDM_OPEN` → goroutine show window
  - `WM_COMMAND` + `IDM_QUIT` → goroutine shutdown daemon + quit
  - `WM_COMMAND` + `menuID >= IDM_SESSION` → T-67-06 bounds check + goroutine focus session

- `showPopupMenu()` — `CreatePopupMenu`, `AppendMenuW` for each item, `GetCursorPos`, `SetForegroundWindow`, `TrackPopupMenu`, `DestroyMenu`.

- `initTray()` — creates windowsTray, converts PNGs to HICONs, launches goroutine with `LockOSThread`, registers "AgentHubTrayClass" window class, creates HWND_MESSAGE window, calls `Shell_NotifyIconW(NIM_ADD)`, closes ready channel, enters GetMessage/TranslateMessage/DispatchMessage loop.

- `updateTray(sessions, connected)` — swaps HIcon (connected/error), updates SzTip via trayTooltip, calls `Shell_NotifyIconW(NIM_MODIFY)`, rebuilds menuItems via BuildMenuItems.

- `cleanupTray()` — `Shell_NotifyIconW(NIM_DELETE)`, `DestroyIcon` x2, `PostMessageW(WM_QUIT)` to exit pump, `DestroyWindow`.

- `setDockVisible(bool)` — no-op (Windows has no Dock equivalent).

- `menuIDForItem(item MenuItem) uintptr` — testable helper mapping MenuItem to IDM_OPEN (1000), IDM_QUIT (1001), or IDM_SESSION+index (1100+).

**Security:** T-67-06 session index bounds check — `menuID - IDM_SESSION` validated as `>= 0 && < len(sessions)` before dispatch to prevent out-of-bounds access.

**Graceful degradation:** If `createIconFromPNG` fails for either icon, logs warning and sets `disabled=true`. If `Shell_NotifyIconW(NIM_ADD)` fails, logs warning and sets `disabled=true`. All subsequent calls become no-ops.

**Tests (tray_windows_test.go, `//go:build windows`):**
- `TestCreateIconFromPNG` — valid PNG → non-zero HICON handle
- `TestCreateIconFromPNGInvalid` — invalid bytes → error, zero handle
- `TestWindowsMenuFromBuildMenuItems` — 2 sessions → 6 items, correct IDM_* values
- `TestWindowsMenuEmpty` — no sessions → 4 items, IDM_OPEN first, IDM_QUIT last

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] syscall.LockOSThread does not exist**
- **Found during:** Task 1 first GOOS=windows go vet run
- **Issue:** Plan specified `runtime.LockOSThread()` but the correct package is Go's stdlib `runtime`, which collides with `github.com/wailsapp/wails/v2/pkg/runtime` already imported as `runtime`.
- **Fix:** Added `goruntime "runtime"` alias (matching existing pattern in app.go) and used `goruntime.LockOSThread()`.
- **Files modified:** tray_windows.go
- **Commit:** 4c7ac65 (fix applied before commit)

**2. [Rule 1 - Bug] Plan spec says 7 items with 2 sessions — actual BuildMenuItems returns 6**
- **Found during:** Task 1 test writing
- **Issue:** Plan test spec stated "BuildMenuItems with 2 sessions produces 7 items" but the committed tray_common.go BuildMenuItems returns 6 items (Open + sep + s0 + s1 + sep + Quit). The plan description was an off-by-one error.
- **Fix:** Test written to match the actual committed implementation (6 items with 2 sessions), not the erroneous plan spec. The Plan 01 implementation is the ground truth.
- **Files modified:** tray_windows_test.go
- **Commit:** cbdab6e

## Known Stubs

None — all four interface methods are implemented. `setDockVisible` is an intentional no-op (Windows has no Dock).

## Threat Flags

None — no new network endpoints or auth paths introduced. Win32 message-only window is local-only (T-67-05 accepted, T-67-06 mitigated in wndProc).

## Self-Check: PASSED

- tray_windows.go: FOUND
- tray_windows_test.go: FOUND
- Commit cbdab6e (RED): FOUND
- Commit 4c7ac65 (GREEN): FOUND
- `GOOS=windows go vet ./...`: PASSED
- `GOOS=windows go build ./...`: PASSED
- `go vet ./...` (darwin): PASSED
- `GOOS=linux go vet ./...`: PASSED
- No changes to tray.go (darwin) or tray_linux.go: CONFIRMED
