---
phase: 03-wails-desktop-ui
plan: "03"
subsystem: ui
tags: [wails, go, cgo, macos, systray, NSStatusBar, pty, xterm]

# Dependency graph
requires:
  - phase: 03-wails-desktop-ui
    plan: "01"
    provides: App struct, Wails scaffold, relay wiring
  - phase: 03-wails-desktop-ui
    plan: "02"
    provides: React frontend, TabBar, TerminalPanel, SettingsPanel

provides:
  - System tray integration via native macOS NSStatusBar (cgo)
  - Window hide-to-tray on close — PTY sessions survive
  - Tray menu: Show AgentHub + Quit
  - Full end-to-end desktop app verified via browser UAT
  - All Phase 3 success criteria met

affects:
  - 04-remote-access
  - 05-status-awareness
  - 06-cross-platform

# Tech tracking
tech-stack:
  added:
    - native macOS cgo NSStatusBar (replaces fyne.io/systray — linker conflict with Wails AppDelegate)
  patterns:
    - Wails v2 requires main package at same directory level as wails.json (project root)
    - Native cgo tray preferred over CGO wrapper libs that define AppDelegate
    - beforeClose() returns true to suppress Wails quit; calls runtime.WindowHide only within Wails context
    - Tray icon goroutine selects on NSStatusBar event channels; runtime.Quit for clean shutdown

key-files:
  created:
    - tray.go (native cgo NSStatusBar tray integration — macOS build tag)
    - tray_test.go (TestHideWindowSessionsAlive, TestBeforeCloseReturnsTrue)
    - assets/appicon.png (app icon embedded via go:embed)
    - build/appicon.png (256x256 icon source)
    - build/gen_icon.go (programmatic icon generation helper)
    - .gitignore (Wails build artifact exclusions)
  modified:
    - app.go (moved from cmd/agenthub/; trayEnd func() field; initTray() in startup; shutdown cleanup)
    - go.mod / go.sum (fyne.io/systray removed; cgo tray needs no external module)

key-decisions:
  - "Replaced fyne.io/systray with native macOS cgo NSStatusBar — fyne defines AppDelegate which conflicts with Wails' own AppDelegate causing duplicate-symbol linker error"
  - "Moved all Go files from cmd/agenthub/ to project root — Wails v2 requires main package co-located with wails.json"
  - "System tray unit tests cover session-preservation behavior only; native tray UI cannot be exercised in headless test runner"

patterns-established:
  - "Wails root layout: main.go, app.go, tray.go, *_test.go all at project root alongside wails.json"
  - "beforeClose() guards runtime.WindowHide with ctx nil-check — lets unit tests call it without Wails context"
  - "cgo tray file has build tag darwin; a stub file provides no-op initTray for non-darwin platforms"

requirements-completed:
  - SESS-02

# Metrics
duration: ~90min
completed: 2026-03-18
---

# Phase 3 Plan 03: System Tray + End-to-End Desktop Verification Summary

**Native macOS NSStatusBar tray via cgo wires window hide-to-tray so PTY sessions survive close; full desktop UAT passes (tabs, rename, ANSI color, settings, CLI detection)**

## Performance

- **Duration:** ~90 min
- **Started:** 2026-03-18T14:59:00Z (approx)
- **Completed:** 2026-03-18
- **Tasks:** 2 (1 auto, 1 checkpoint)
- **Files modified:** 12

## Accomplishments

- System tray integration using native macOS cgo NSStatusBar — avoids the AppDelegate linker conflict that blocks fyne.io/systray with Wails
- Window close hides to tray (beforeClose returns true); all PTY sessions remain alive in registry; Show AgentHub restores window
- Wails project restructured: all Go files moved from cmd/agenthub/ to project root as required by Wails v2
- Human UAT checkpoint approved: dark theme, tab bar, +/gear buttons, CLI detection (4 CLIs), xterm.js with full ANSI color, multiple tabs, double-click rename, settings panel, tab close all verified

## Task Commits

Each task was committed atomically:

1. **Task 1: System tray integration + tests** - `3d8a3ed` (feat) — initial fyne.io/systray implementation + tests
2. **Task 2: Verify complete desktop app experience** - `d244396` (fix) — moved to project root, replaced fyne with native cgo tray
3. **Task 2 follow-up: gitignore** - `3f2a362` (chore) — Wails build artifact exclusions

## Files Created/Modified

- `tray.go` - Native macOS cgo NSStatusBar tray (Show AgentHub + Quit menu items)
- `tray_test.go` - TestHideWindowSessionsAlive, TestBeforeCloseReturnsTrue
- `app.go` - trayEnd func() field; initTray() call in startup; shutdown cleanup; beforeClose ctx guard
- `assets/appicon.png` - 256x256 dark blue app icon (embedded via go:embed)
- `build/appicon.png` - App icon source used by Wails build
- `build/gen_icon.go` - Programmatic PNG icon generator (build helper)
- `go.mod` / `go.sum` - fyne.io/systray removed (cgo approach requires no external module)
- `.gitignore` - Wails build/ and dist/ artifact exclusions

## Decisions Made

- **fyne.io/systray replaced with native cgo NSStatusBar:** fyne.io/systray defines its own `AppDelegate` via cgo, which collides with Wails' `AppDelegate` at link time (duplicate symbol). Native cgo wrapping of `NSStatusBar`/`NSStatusItem` resolves the conflict without an external library.
- **Moved Go files to project root:** Wails v2 resolves the frontend assets path relative to `wails.json`. The `main` package must live in the same directory. Moving from `cmd/agenthub/` is the canonical Wails project layout.
- **System tray tests are behavior-only:** `systray` / NSStatusBar calls require a macOS display server and cannot run in a headless test runner. Tests cover the session-preservation invariant (`registry.Len()` unchanged after `beforeClose`) and return value only.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Replaced fyne.io/systray with native macOS cgo NSStatusBar**
- **Found during:** Task 1 (system tray integration)
- **Issue:** fyne.io/systray defines `AppDelegate` in its cgo layer; Wails also defines `AppDelegate`; the linker fails with "duplicate symbol _AppDelegate" making the binary unlinkable
- **Fix:** Removed fyne.io/systray dependency; implemented `tray.go` using direct cgo calls to `NSStatusBar`, `NSStatusItem`, `NSMenu`, `NSMenuItem` — no external library needed
- **Files modified:** tray.go, go.mod, go.sum
- **Verification:** `wails dev` builds and launches successfully; tray icon appears in macOS menu bar
- **Committed in:** d244396

**2. [Rule 3 - Blocking] Moved all Go files from cmd/agenthub/ to project root**
- **Found during:** Task 2 (wails dev launch)
- **Issue:** Wails v2 requires the `main` package to be co-located with `wails.json`. With sources in `cmd/agenthub/`, `wails dev` could not locate the main package
- **Fix:** Moved app.go, main.go, tray.go, *_test.go, assets/ from `cmd/agenthub/` to project root; updated embed paths; removed now-empty `cmd/agenthub/` directory
- **Files modified:** app.go, main.go, tray.go, tray_test.go, app_test.go, assets_stub.go, assets/appicon.png
- **Verification:** `wails dev` builds successfully; all tests pass at project root
- **Committed in:** d244396

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes were structurally necessary. The linker conflict is a hard incompatibility between fyne.io/systray and Wails; the project layout requirement is a Wails v2 constraint. No scope creep — plan objective (tray + verified desktop experience) fully delivered.

## Issues Encountered

- Initial tray icon approach used `fyne.io/systray` as specified in the plan's research notes; duplicate AppDelegate symbol was discovered at link time and required the cgo rewrite. Wails `wails.json`-relative main package requirement was discovered during `wails dev` and required project restructure.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 3 is complete. All three plans delivered: Wails scaffold (03-01), React frontend terminal UI (03-02), and system tray + end-to-end verification (03-03).
- Phase 4 (Remote Access / TLS / HTTPS) can begin. Concern: per-OS CA cert trust installation UX (macOS Keychain, Linux NSS, Windows certutil) needs design before Phase 4 execution.
- The cgo tray implementation is macOS-only (build tag `darwin`). Phase 6 cross-platform work will need Linux (AppIndicator/libayatana) and Windows (systray via win32 API) implementations.

---
*Phase: 03-wails-desktop-ui*
*Completed: 2026-03-18*
