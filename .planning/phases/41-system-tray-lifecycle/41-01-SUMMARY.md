---
phase: 41-system-tray-lifecycle
plan: 01
subsystem: infra
tags: [cgo, macos, tray, daemon, plist, png, go-embed]

# Dependency graph
requires:
  - phase: 40-daemon-management-panel
    provides: DaemonClient infrastructure and API protocol patterns
provides:
  - POST /shutdown daemon endpoint with 50ms delayed os.Exit(0)
  - DaemonClient.ShutdownDaemon() method treating connection-reset as success
  - assets/tray_icon.png: 18x18 monochrome "A" letterform on transparent background
  - assets/tray_icon_error.png: 18x18 monochrome "!" glyph on transparent background
  - tray.go embeds tray_icon.png and tray_icon_error.png (not appicon.png)
  - build/darwin/Info.plist with LSUIElement=true for Dock hiding in production
affects: [42-tray-integration, future-daemon-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Daemon shutdown: flush response before goroutine delay so client receives 204 before exit"
    - "ShutdownDaemon nil-on-connection-error: connection-reset is expected success signal"
    - "Go embed pattern for multiple tray icons: separate //go:embed var per file"

key-files:
  created:
    - assets/tray_icon.png
    - assets/tray_icon_error.png
  modified:
    - internal/daemon/api.go
    - internal/daemon/client.go
    - internal/daemon/client_test.go
    - tray.go
    - tray_test.go
    - build/darwin/Info.plist

key-decisions:
  - "Daemon shutdown flushes response before goroutine delay — ensures client receives 204 before os.Exit(0) fires"
  - "ShutdownDaemon treats all connection errors as success — daemon may exit before response body completes"
  - "Tray icons generated with Go image/draw for pixel-exact 18x18 letterforms rather than ImageMagick resize"
  - "LSUIElement added to Info.plist only (not Info.dev.plist) — dev mode retains Dock icon for debugging"

patterns-established:
  - "Tray icon embed: separate //go:embed var for normal and error state icons"
  - "Test httptest.NewServer for daemon API tests that would otherwise call os.Exit"

requirements-completed: [TRAY-05, DMGR-02, BRND-03]

# Metrics
duration: 12min
completed: 2026-04-02
---

# Phase 41 Plan 01: System Tray Lifecycle Foundation Summary

**Daemon POST /shutdown endpoint, two 18x18 monochrome tray icon PNGs embedded in tray.go, and LSUIElement=true in production Info.plist**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-04-02T16:53:00Z
- **Completed:** 2026-04-02T17:03:17Z
- **Tasks:** 3
- **Files modified:** 6 files (+ 2 new PNG assets)

## Accomplishments
- Added `POST /shutdown` route to daemon API with 50ms delayed `os.Exit(0)` after flushing 204 response
- Added `DaemonClient.ShutdownDaemon()` method treating connection-reset errors as success
- Generated two 18x18 monochrome PNG icons: `tray_icon.png` (A letterform) and `tray_icon_error.png` (! glyph)
- Updated `tray.go` embeds from `appicon.png` to the new tray-specific icons; added `trayIconErrorBytes` var
- Added `LSUIElement=true` to `build/darwin/Info.plist` (production only — dev plist unchanged)
- All new functionality covered by tests: `TestShutdownDaemon` and `TestTrayIconAsset`

## Task Commits

Each task was committed atomically:

1. **Task 1: Add daemon /shutdown endpoint and client method** - `3d7fd1a` (feat)
2. **Task 2: Generate monochrome tray icon PNG assets** - `74cfc33` (feat)
3. **Task 3: Add LSUIElement to production Info.plist** - `79e9c04` (feat)

## Files Created/Modified
- `internal/daemon/api.go` - Added `POST /shutdown` route and `handleShutdown` method with `time` import
- `internal/daemon/client.go` - Added `ShutdownDaemon()` method using `context.WithTimeout` + nil-on-error
- `internal/daemon/client_test.go` - Added `TestShutdownDaemon` using `httptest.NewServer`
- `assets/tray_icon.png` - 18x18 monochrome A letterform, transparent background, RGBA PNG
- `assets/tray_icon_error.png` - 18x18 monochrome ! exclamation glyph, transparent background, RGBA PNG
- `tray.go` - Updated embed from `appicon.png` to `tray_icon.png`; added `trayIconErrorBytes` embed
- `tray_test.go` - Added `TestTrayIconAsset` verifying both icons decode as valid 18x18 PNGs
- `build/darwin/Info.plist` - Added `<key>LSUIElement</key><true/>` after `NSHighResolutionCapable`

## Decisions Made
- Daemon shutdown: flush HTTP response before goroutine sleep so client receives 204 before `os.Exit(0)` fires
- `ShutdownDaemon()` treats all `http.Do` errors as success — connection reset is the expected signal the daemon exited
- Icons generated with `go run` script using `image/draw` for pixel-exact 18x18 geometry; script deleted after use
- `LSUIElement` in `Info.plist` only (not `Info.dev.plist`) per STATE.md recorded decision — dev keeps Dock icon for debugging

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 02 (tray cgo extensions) can proceed: all three foundation artifacts exist
  - `/shutdown` endpoint ready for `onTrayQuit` to call `ShutdownDaemon()`
  - `trayIconBytes` and `trayIconErrorBytes` ready for icon swap in `updateTrayIcon()`
  - `LSUIElement` set — production builds will show only in menu bar

## Self-Check: PASSED
- `assets/tray_icon.png`: FOUND (18x18 PNG)
- `assets/tray_icon_error.png`: FOUND (18x18 PNG)
- `internal/daemon/api.go`: contains `handleShutdown` — FOUND
- `internal/daemon/client.go`: contains `ShutdownDaemon` — FOUND
- `build/darwin/Info.plist`: contains `LSUIElement` — FOUND (count=1)
- Commit 3d7fd1a: FOUND
- Commit 74cfc33: FOUND
- Commit 79e9c04: FOUND

---
*Phase: 41-system-tray-lifecycle*
*Completed: 2026-04-02*
