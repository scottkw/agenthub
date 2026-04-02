---
phase: 41-system-tray-lifecycle
plan: 02
subsystem: infra
tags: [cgo, macos, tray, objc, nsmenudelegate, daemon, lifecycle, frontend]

# Dependency graph
requires:
  - phase: 41-01
    provides: trayIconBytes/trayIconErrorBytes embeds, ShutdownDaemon client method, LSUIElement plist
provides:
  - tray_objc.m: AgentHubMenuDelegate NSMenuDelegate class + initStatusItem/updateTrayIcon/updateTrayTooltip/setTraySessionData C functions
  - tray.go: onTraySession export, updated onTrayQuit (calls ShutdownDaemon), trayTooltip(), updateTray()
  - app.go: startTrayPoller() and refreshTrayState() goroutine methods
  - frontend/src/App.tsx: tray:focus-session EventsOn handler
affects: [42-tray-integration, future-tray-platforms]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ObjC cgo split: put @interface/@implementation in .m file to avoid duplicate symbol linker errors during go test"
    - "NSMenuDelegate menuWillOpen: rebuilds menu dynamically on open — no polling needed for menu content"
    - "tray:focus-session via runtime.EventsEmit from cgo callback goroutine — safe because goroutine runs Go scheduler"
    - "setTabs(prev => ...) pattern in EventsOn handler avoids stale closure without adding tabs to useEffect deps"

key-files:
  created:
    - tray_objc.m
  modified:
    - tray.go
    - tray_linux.go
    - tray_windows.go
    - app.go
    - tray_test.go
    - frontend/src/App.tsx

key-decisions:
  - "ObjC class moved to tray_objc.m (separate .m file) — cgo compiles each .go file as a separate translation unit; inline ObjC @implementation in .go causes duplicate symbol errors when go test links multiple .o files"
  - "NSMenuDelegate menuWillOpen: pattern used instead of proactive menu rebuild — menu only built when user opens it, always fresh"
  - "startTrayPoller does immediate refresh before first tick — avoids blank tooltip/icon state during 5s init window"

patterns-established:
  - "cgo ObjC classes: define in .m file; declare function signatures in .go cgo comment block"

requirements-completed: [TRAY-01, TRAY-02, TRAY-03, TRAY-04, TRAY-06, DMGR-01, DMGR-02]

# Metrics
duration: 25min
completed: 2026-04-02
---

# Phase 41 Plan 02: Dynamic Tray Menu, Icon State, Tooltip, and Session Focus Summary

**AgentHubMenuDelegate NSMenuDelegate for dynamic session menu, updateTray() for icon/tooltip state, startTrayPoller() 5s background refresh, and tray:focus-session frontend event handler**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-04-02T17:10:00Z
- **Completed:** 2026-04-02T17:35:00Z
- **Tasks:** 2 complete + 1 checkpoint awaiting human verify
- **Files modified:** 6 files (+ 1 new .m file)

## Accomplishments

- Created `tray_objc.m` with `AgentHubMenuDelegate` (NSMenuDelegate) that rebuilds menu on every `menuWillOpen:` call
- Added `initStatusItem`, `updateTrayIcon`, `updateTrayTooltip`, `setTraySessionData` C functions in `.m` file
- `tray.go`: added `onTraySession` cgo export for session click; updated `onTrayQuit` to call `ShutdownDaemon` in goroutine before `runtime.Quit`
- `tray.go`: added `trayTooltip(n int)` returning em-dash strings per UI-SPEC; added `updateTray(sessions, connected)` updating icon/tooltip/session data
- `app.go`: added `startTrayPoller(ctx)` goroutine (5s tick + immediate) and `refreshTrayState()` helper
- Called `startTrayPoller` in both `startup()` and `RetryDaemon()` after `trayInit = true`
- `tray_linux.go`, `tray_windows.go`: added `updateTray` no-op stubs
- `frontend/src/App.tsx`: added `cancelTrayFocus = EventsOn('tray:focus-session', ...)` with `setTabs(prev => ...)` pattern; cleaned up in effect return
- All 6 Go packages pass `go test ./... -count=1`

## Task Commits

1. **Task 1: Extend tray.go cgo with dynamic menu, icon swap, tooltip, session click** - `9bebeb1` (feat)
2. **Task 2: Add tray:focus-session event handler in frontend App.tsx** - `1ea8886` (feat)

## Files Created/Modified

- `tray_objc.m` - New file: AgentHubMenuDelegate, all C helper functions for tray updates
- `tray.go` - Refactored: ObjC moved to .m; added onTraySession, updated onTrayQuit, added trayTooltip/updateTray
- `tray_linux.go` - Added `updateTray` no-op stub
- `tray_windows.go` - Added `updateTray` no-op stub
- `app.go` - Added `startTrayPoller`, `refreshTrayState`; wired poller into startup and RetryDaemon
- `tray_test.go` - Added `TestTrayTooltip`, `TestTrayQuitNilClient`, `TestRefreshTrayStateNilClient`
- `frontend/src/App.tsx` - Added `tray:focus-session` EventsOn handler and cancelTrayFocus cleanup

## Decisions Made

- ObjC `@interface`/`@implementation` moved to `tray_objc.m` — inline cgo ObjC classes in `.go` files cause duplicate symbol linker errors because cgo compiles each `.go` file into a separate translation unit; `#ifndef` guards don't prevent linker-level duplicate symbols across separate object files
- `NSMenuDelegate menuWillOpen:` used for dynamic menu — menu always reflects current state at open time; no separate push-update needed
- `startTrayPoller` fires immediately before first tick — ensures tray shows correct tooltip/icon/sessions from first second of app life

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Moved ObjC class definition to separate .m file**
- **Found during:** Task 1 verification (go test)
- **Issue:** Inline `@interface AgentHubMenuDelegate` in cgo comment of `tray.go` caused duplicate symbol linker errors (`duplicate symbol '_OBJC_CLASS_$_AgentHubMenuDelegate'`). Go's cgo compiles each `.go` file in a package as a separate C translation unit; `#ifndef` guards don't prevent linker-level symbol conflicts.
- **Fix:** Extracted all ObjC `@interface`/`@implementation` blocks and C helper function bodies to `tray_objc.m`. `tray.go` cgo block now only has `#include <stdlib.h>` and forward declarations. cgo automatically compiles `.m` files once.
- **Files modified:** `tray.go` (stripped to declarations), `tray_objc.m` (new)
- **Commit:** `9bebeb1`

## Test Results

```
ok  github.com/agenthub/agenthub              6.632s
ok  github.com/agenthub/agenthub/internal/daemon  1.247s
ok  github.com/agenthub/agenthub/internal/pty     1.696s
ok  github.com/agenthub/agenthub/internal/relay   1.089s
ok  github.com/agenthub/agenthub/internal/status  0.994s
ok  github.com/agenthub/agenthub/internal/webserver 1.622s
```

## Checkpoint: Task 3 Human Verification — PASSED

Production build (`wails build -tags wailsassets`) manually verified. All 12 UAT items passed:
- Tray icon visible in macOS menu bar (monochrome, adapts to theme) ✓
- No Dock icon (LSUIElement + NSApplicationActivationPolicyAccessory) ✓
- Tooltip shows "AgentHub — no sessions" / "AgentHub — 1 session" ✓
- Menu shows "Open AgentHub", session names, "Quit" ✓
- Window close hides window; tray + daemon remain ✓
- "Open AgentHub" reopens window ✓
- Session name click focuses that tab ✓
- Quit from tray fully exits app and daemon ✓
- No leftover PIDs after quit ✓

UAT fixes committed in `b658cdd`:
- Moved initTray() before EnsureDaemon (tray shows regardless of daemon state)
- Added quitting flag to bypass beforeClose hide-on-close during tray Quit
- Copied Go pointers to NSData/NSString before dispatch_async (SIGSEGV fix)
- Set NSApplicationActivationPolicyAccessory programmatically (Wails overrides LSUIElement)

## Known Stubs

None — all tray features are wired to real daemon data.

## Self-Check: PASSED
- `tray_objc.m`: FOUND
- `tray.go`: contains `onTraySession` — FOUND
- `tray.go`: contains `updateTray` — FOUND
- `tray.go`: contains `trayTooltip` — FOUND
- `app.go`: contains `startTrayPoller` — FOUND
- `app.go`: contains `refreshTrayState` — FOUND
- `tray_linux.go`: contains `updateTray` stub — FOUND
- `tray_windows.go`: contains `updateTray` stub — FOUND
- `frontend/src/App.tsx`: contains `tray:focus-session` — FOUND
- Commit 9bebeb1: FOUND
- Commit 1ea8886: FOUND

---
*Phase: 41-system-tray-lifecycle*
*Completed: 2026-04-02*
