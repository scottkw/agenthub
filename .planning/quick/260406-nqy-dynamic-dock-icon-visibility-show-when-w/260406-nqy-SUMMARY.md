---
phase: quick
plan: 260406-nqy
subsystem: tray/window-lifecycle
tags: [macos, dock, tray, cgo, objc, wails]
dependency_graph:
  requires: []
  provides: [dynamic-dock-icon-visibility]
  affects: [tray, app-lifecycle]
tech_stack:
  added: []
  patterns: [cgo-objc-bridge, platform-stubs, activation-policy-toggle]
key_files:
  created: []
  modified:
    - tray_objc.m
    - tray.go
    - tray_linux.go
    - tray_windows.go
    - app.go
decisions:
  - Keep initStatusItem's NSApplicationActivationPolicyAccessory set — dock stays hidden at startup until domReady fires setDockVisible(true)
  - Use dispatch_async(main_queue) for setDockVisible consistent with other ObjC functions in tray_objc.m
  - Add activateIgnoringOtherApps:YES when showing dock so window comes forward
metrics:
  duration: 8m
  completed: 2026-04-06
  tasks_completed: 2
  files_modified: 5
---

# Quick Task 260406-nqy: Dynamic Dock Icon Visibility Summary

**One-liner:** macOS dock icon now tracks window state via NSApplicationActivationPolicy toggling bridged through cgo — visible when window is shown, hidden when closed to tray.

## Objective

Implement dynamic dock icon visibility so the macOS dock icon appears when the app window is shown and disappears when the window is hidden to the system tray.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add setDockVisible C function and Go wrappers | ba447f9 | tray_objc.m, tray.go, tray_linux.go, tray_windows.go |
| 2 | Wire dock visibility into window lifecycle events | 6c0fe50 | app.go, tray.go |

## What Was Built

### Task 1: setDockVisible infrastructure

- `tray_objc.m`: New `setDockVisible(int visible)` C function that toggles `NSApplicationActivationPolicy` on the main queue. When showing, also calls `activateIgnoringOtherApps:YES` to bring the window forward.
- `tray.go`: Added `void setDockVisible(int visible)` declaration to cgo preamble and Go wrapper method `(a *App) setDockVisible(visible bool)`.
- `tray_linux.go`, `tray_windows.go`: Added no-op stubs `func (a *App) setDockVisible(visible bool) {}` so all platforms compile.

### Task 2: Lifecycle wiring

Four call sites cover all window show/hide transitions:

- `app.go domReady`: `a.setDockVisible(true)` after `runtime.WindowShow` — dock shows on startup
- `app.go beforeClose`: `a.setDockVisible(false)` before `runtime.WindowHide` — dock hides when window closes to tray
- `tray.go onTrayShow`: `trayCallbackApp.setDockVisible(true)` after `runtime.WindowShow` — dock restores when user clicks "Open AgentHub" in tray menu
- `tray.go onTraySession`: `trayCallbackApp.setDockVisible(true)` after `runtime.WindowShow` — dock restores when user clicks a session in tray menu

## Verification

- `go vet -tags dev ./...` passes with no errors on both tasks
- The pre-existing Wails linker error (`_OBJC_CLASS_$_UTType`) confirmed to exist before these changes — not caused by this task

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. The Linux/Windows stubs are intentional no-ops documented as such; the feature is macOS-only by design.

## Self-Check: PASSED

- tray_objc.m modified: FOUND (setDockVisible function added)
- tray.go modified: FOUND (cgo declaration + Go wrapper + onTrayShow/onTraySession wired)
- tray_linux.go modified: FOUND (no-op stub added)
- tray_windows.go modified: FOUND (no-op stub added)
- app.go modified: FOUND (domReady + beforeClose wired)
- Commit ba447f9: FOUND
- Commit 6c0fe50: FOUND
