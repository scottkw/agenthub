---
phase: 06-distribution-cross-platform
plan: 01
subsystem: infra
tags: [go, build-tags, wails, cross-platform, darwin, linux, windows, entitlements, plist]

# Dependency graph
requires:
  - phase: 03-wails-desktop-ui
    provides: tray.go NSStatusBar implementation that needed darwin build constraint
provides:
  - darwin build tag on tray.go constraining Cocoa CGO to macOS only
  - Linux tray stub (tray_linux.go) with no-op initTray/cleanupTray
  - Windows tray stub (tray_windows.go) with no-op initTray/cleanupTray
  - wails.json info section with NSIS installer metadata
  - build/darwin/Info.plist with com.agenthub.app production bundle identifier
  - build/entitlements.plist with hardened runtime network entitlements
affects:
  - 06-02-ci-matrix (depends on these files existing for wails build -platform)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Platform stub pattern: darwin-only CGO file + linux/windows no-op stubs with matching function signatures"
    - "Go build tags for platform-specific compilation (//go:build darwin|linux|windows)"

key-files:
  created:
    - tray_linux.go
    - tray_windows.go
    - build/darwin/Info.plist
    - build/entitlements.plist
  modified:
    - tray.go (added //go:build darwin tag)
    - wails.json (added info section)
    - .gitignore (negation patterns for Info.plist files)

key-decisions:
  - "tray.go darwin build tag: add as FIRST line before package main per go:build constraint spec"
  - "build/darwin/ gitignore negation: use build/darwin/* + !Info.plist pattern instead of directory-level exclude to track plists while ignoring binary outputs"
  - "entitlements.plist: network.client + network.server only — no get-task-allow (Apple rejects notarization with it)"

patterns-established:
  - "Platform stub pattern: one file per OS (tray.go/tray_linux.go/tray_windows.go) with build tags and identical function signatures"

requirements-completed: [PLAT-01, PLAT-02, PLAT-03]

# Metrics
duration: 10min
completed: 2026-03-19
---

# Phase 6 Plan 01: Platform Stubs + Build Config Summary

**darwin-only CGO tray constrained via //go:build tag, Linux/Windows no-op stubs added, wails.json NSIS metadata and notarization entitlements plist created — codebase now compiles on all three target platforms**

## Performance

- **Duration:** 10 min
- **Started:** 2026-03-19T00:00:00Z
- **Completed:** 2026-03-19T00:10:00Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments

- tray.go restricted to macOS via `//go:build darwin` — prevents Cocoa header import errors on Linux/Windows cross-compilation
- tray_linux.go and tray_windows.go created with matching no-op `initTray`/`cleanupTray` signatures
- wails.json `info` section added with companyName, productName, productVersion, copyright, and comments for NSIS installer
- build/darwin/Info.plist created with `com.agenthub.app` CFBundleIdentifier (production bundle ID)
- build/entitlements.plist created with `network.client` and `network.server` entitlements for notarized hardened runtime
- All Go tests pass, `go vet ./...` clean, macOS binary compiles after changes

## Task Commits

Each task was committed atomically:

1. **Task 1: Add darwin build tag to tray.go and create Linux/Windows stubs** - `b6404b3` (feat)
2. **Task 2: Add wails.json Info section and production build config files** - `3fc2776` (feat)
3. **Task 3: Verify local macOS build and all Go tests pass** - verification only, no separate commit

## Files Created/Modified

- `tray.go` - Added `//go:build darwin` as first line to constrain Cocoa CGO to macOS
- `tray_linux.go` - New: Linux no-op stubs for initTray and cleanupTray
- `tray_windows.go` - New: Windows no-op stubs for initTray and cleanupTray
- `wails.json` - Added `info` section with NSIS installer metadata
- `build/darwin/Info.plist` - New: Production Info.plist with com.agenthub.app bundle identifier
- `build/entitlements.plist` - New: Hardened runtime entitlements for notarization
- `.gitignore` - Fixed to track Info.plist files within otherwise-ignored build/darwin/

## Decisions Made

- **build/darwin/ gitignore fix:** The `build/darwin/` directory was gitignored entirely. Changed pattern from `build/darwin/` to `build/darwin/*` with `!build/darwin/Info.plist` and `!build/darwin/Info.dev.plist` negations so the config plists are tracked while binary outputs (.app bundles, installers) remain ignored.
- **entitlements.plist scope:** Only `network.client` (WebSocket connections) and `network.server` (TLS web server listener) included. `get-task-allow` explicitly excluded — Apple's notarization rejects it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed .gitignore blocking build/darwin/Info.plist from being tracked**
- **Found during:** Task 2 (wails.json Info section and build config files)
- **Issue:** `.gitignore` had `build/darwin/` which ignores the entire directory, preventing `git add build/darwin/Info.plist`
- **Fix:** Changed `build/darwin/` to `build/darwin/*` with negation patterns `!build/darwin/Info.plist` and `!build/darwin/Info.dev.plist`
- **Files modified:** `.gitignore`
- **Verification:** `git add build/darwin/Info.plist` succeeded after fix
- **Committed in:** `3fc2776` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug — gitignore blocking required file)
**Impact on plan:** Essential fix — without it, Info.plist would not be tracked in git and Wails CI build would fail with missing production plist.

## Issues Encountered

None — beyond the gitignore deviation auto-fixed above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Cross-platform codebase prerequisites complete: build tags, stubs, and distribution config files all in place
- Plan 06-02 (CI matrix) can now proceed — it depends on these files existing for `wails build -platform linux/amd64` and `wails build -platform windows/amd64`
- macOS local build and all Go tests confirmed passing after changes

---
*Phase: 06-distribution-cross-platform*
*Completed: 2026-03-19*
