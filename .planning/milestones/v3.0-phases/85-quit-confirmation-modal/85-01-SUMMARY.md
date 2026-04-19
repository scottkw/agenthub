---
phase: 85-quit-confirmation-modal
plan: "01"
subsystem: backend-quit-pathways
tags: [go, wails, quit-modal, notification, cgo, objc]
dependency_graph:
  requires: []
  provides: [app-quit-requested-event, quit-gui-only-method, quit-all-method, macos-notification]
  affects: [app.go, tray.go, tray_objc_darwin.m, notification_darwin.go, notification_other.go, frontend/src/wailsjs/go/main/App.d.ts, frontend/src/wailsjs/go/main/App.js]
tech_stack:
  added: [UNUserNotificationCenter]
  patterns: [wails-event-emit, cgo-objc-wrapper, build-tag-stubs]
key_files:
  created:
    - notification_darwin.go
    - notification_other.go
  modified:
    - app.go
    - tray.go
    - tray_objc_darwin.m
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
decisions:
  - beforeClose emits app:quit-requested event instead of hiding window (D-08, D-12)
  - onTrayQuit emits app:quit-requested event instead of directly quitting (D-06)
  - sendNotification placed in separate notification_darwin.go with own LDFLAGS to avoid modifying tray.go cgo header
  - notification_other.go provides no-op stub for non-darwin builds
  - TypeScript binding stubs updated manually (Wails generates into nested wailsjs/wailsjs path, frontend imports from wailsjs/go/main)
metrics:
  duration: 275s
  completed: "2026-04-19T17:50:06Z"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 7
---

# Phase 85 Plan 01: Go Backend Quit Pathway Refactor Summary

Refactored Go backend quit pathways to emit a Wails event (app:quit-requested) instead of directly hiding/quitting, added QuitGUIOnly and QuitAll bound methods for frontend modal control, and added macOS native notification support via UNUserNotificationCenter cgo bridge.

## Task Completion

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Refactor beforeClose + add QuitGUIOnly/QuitAll | ebeb311 | app.go, notification_darwin.go, notification_other.go, tray_objc_darwin.m |
| 2 | Refactor onTrayQuit + TypeScript bindings | 43b0274 | tray.go, App.d.ts, App.js |

## Changes Made

### beforeClose refactor (app.go)
- Replaced window-hide behavior with event emission: `runtime.EventsEmit(ctx, "app:quit-requested", nil)`
- Added `runtime.WindowShow(ctx)` and `a.setDockVisible(true)` before event emission (D-08: ensure window visible for modal)
- Maintained `return true` in all non-quitting paths (TestBeforeCloseReturnsTrue passes)
- `quitting` flag bypass path unchanged (`return false`)

### QuitGUIOnly (app.go)
- Hides window via `runtime.WindowHide(a.ctx)` and dock via `a.setDockVisible(false)`
- Counts active sessions and sends macOS notification with session count (D-11)
- Singular/plural message formatting for session count
- ctx nil guard for safety

### QuitAll (app.go)
- Shuts down daemon via `a.client.ShutdownDaemon()` (nil-safe)
- Sets `a.quitting = true` then calls `runtime.Quit(a.ctx)` (same pattern as pre-refactor onTrayQuit)
- ctx nil guard for safety

### onTrayQuit refactor (tray.go)
- Replaced ShutdownDaemon + quitting + runtime.Quit with event emission
- Added `runtime.WindowShow` + `setDockVisible(true)` before event (D-08)
- Preserved goroutine safety pattern (capture trayCallbackApp before goroutine)
- TestTrayQuitNilClient continues to pass (nil guards cover both nil app and nil ctx)

### macOS notification infrastructure
- `tray_objc_darwin.m`: Added `#import <UserNotifications/UserNotifications.h>` and `sendNotification` C function using `UNUserNotificationCenter` with lazy permission request
- `notification_darwin.go`: cgo wrapper with `//go:build darwin`, `-framework UserNotifications` LDFLAGS, C string memory management
- `notification_other.go`: no-op stub with `//go:build !darwin` for cross-platform builds

### TypeScript bindings
- Added `QuitGUIOnly(): Promise<void>` and `QuitAll(): Promise<void>` to `App.d.ts`
- Added corresponding `Call('main.App.QuitGUIOnly', [])` and `Call('main.App.QuitAll', [])` to `App.js`

## Verification Results

- `go vet ./...` passes with zero errors
- `TestBeforeCloseReturnsTrue` passes (beforeClose returns true in non-quitting paths)
- `TestTrayQuitNilClient` passes (nil app/client does not panic)
- `grep -c 'app:quit-requested' app.go tray.go` returns 1 for each file
- `grep 'QuitGUIOnly\|QuitAll' App.d.ts` shows both declarations
- Note: `TestHideWindowSessionsAlive` fails on unmodified code (pre-existing issue with `runtime.EventsEmit` called from `pollSessionStatus` using `context.Background()` in tests) -- not caused by this plan's changes

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] notification_darwin.go and tray_objc_darwin.m changes committed with Task 1**
- **Found during:** Task 1
- **Issue:** app.go calls `sendNotification` which is defined in `notification_darwin.go`, which in turn requires the C function in `tray_objc_darwin.m`. These files are listed under Task 2 but are required for Task 1 to compile.
- **Fix:** Included notification_darwin.go, notification_other.go, and tray_objc_darwin.m in Task 1 commit to maintain compilable state at each commit.
- **Files modified:** notification_darwin.go, notification_other.go, tray_objc_darwin.m
- **Commit:** ebeb311

**2. [Rule 3 - Blocking] TypeScript binding stubs manually updated**
- **Found during:** Task 2
- **Issue:** `wails generate module` generates bindings into `frontend/src/wailsjs/wailsjs/go/main/` (nested path) but frontend imports from `frontend/src/wailsjs/go/main/` (manually maintained stubs).
- **Fix:** Added QuitGUIOnly and QuitAll declarations to both App.d.ts and App.js in the import path the frontend uses.
- **Files modified:** frontend/src/wailsjs/go/main/App.d.ts, frontend/src/wailsjs/go/main/App.js
- **Commit:** 43b0274

## Known Stubs

None -- all methods are fully implemented with real behavior.

## Self-Check: PASSED

All 8 files found on disk. Both task commits (ebeb311, 43b0274) verified in git log.
