---
phase: 167-native-notifications
plan: 05
subsystem: notifications
tags: [cgo, objective-c, macos, unusernotificationcenter, regression-fix, crash-hardening]

# Dependency graph
requires:
  - phase: 167-native-notifications (plans 01-04)
    provides: NotifyOnWaiting setting, maybeNotifyWaiting tray-poller edge-detector, per-session sendNotification wiring, Settings Behavior toggle
provides:
  - Bundle-id-guarded, @try/@catch-hardened darwin native notification path (log-and-swallow when unbundled)
  - Headless regression test locking in the bundle-id detection primitive
  - Updated TESTING.md manifest/traceability/manual-checklist entries for the gap closure
affects: [167-native-notifications, future-signed-build-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Native-API fail-safe: guard synchronously before dispatch, log-and-swallow on missing precondition, @try/@catch as defense in depth around the async native call"

key-files:
  created:
    - notification_darwin_test.go
  modified:
    - tray_objc_darwin.m
    - notification_darwin.go
    - TESTING.md

key-decisions:
  - "Guard synchronously BEFORE dispatch_async, not inside it — prevents the crash-prone UNUserNotificationCenter call from ever being reached in an unbundled process, rather than merely catching the resulting exception"
  - "@try/@catch retained as defense-in-depth around the UNUserNotificationCenter block even with the bundle-id guard, in case a signed build hits a residual entitlement fault"
  - "Only the honest, runnable assertion (hasAppBundleID() == false) was tested; the async main-dispatch-queue crash path itself is NOT unit-testable (go test never pumps the main queue), so no false-passing test was invented for it"

requirements-completed: [NTF-01, NTF-02, NTF-03, NTF-04]

coverage:
  - id: D1
    description: "sendNotification (native + Go wrapper) fails safe: log-and-swallow no-op when there is no valid app-bundle identifier, unchanged behavior on a valid bundle"
    requirement: "NTF-01"
    verification:
      - kind: unit
        ref: "notification_darwin_test.go#TestHasAppBundleID_FalseWhenUnbundled"
        status: pass
      - kind: unit
        ref: "notification_darwin_test.go#TestSendNotification_NoBundleReturnsCleanly"
        status: pass
      - kind: other
        ref: "go build ."
        status: pass
    human_judgment: false
  - id: D2
    description: "Real UNUserNotificationCenter delivery + tray-hidden behavior on a signed production build (M-41 re-run) — the actual crash-regression fix confirmed live"
    requirement: "NTF-01"
    verification: []
    human_judgment: true
    rationale: "Real OS notification delivery requires a live desktop session on a signed/bundled .app; go test never pumps the main dispatch queue, so the async native path cannot be exercised headlessly. This is the pre-existing M-41 manual item, now re-scoped to also confirm the hardening didn't regress delivery."

duration: 8min
completed: 2026-07-01
status: complete
---

# Phase 167 Plan 05: Harden Darwin Native Notification Path (M-41 Gap Closure) Summary

**Guarded `sendNotification` in `tray_objc_darwin.m`/`notification_darwin.go` with a bundle-id check (fail before dispatch) + `@try/@catch` defense-in-depth, closing the always-on tray-poller crash that killed the toast/tab-dot/Hub-glow/auto-close together under `wails dev`.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-01T12:19:53Z
- **Completed:** 2026-07-01T12:23:13Z
- **Tasks:** 2
- **Files modified:** 4 (2 modified in Task 1, 1 created + 1 modified in Task 2)

## Accomplishments
- `tray_objc_darwin.m`: added `hasValidBundleIdentifier()` and made `sendNotification` guard synchronously before `dispatch_async`, returning early (log-and-swallow) when there is no valid app-bundle identifier; wrapped the `UNUserNotificationCenter` authorization + request-add blocks in `@try/@catch` as defense in depth
- `notification_darwin.go`: exposed `hasAppBundleID()` via cgo; the Go `sendNotification` wrapper now early-returns with a single log line when `hasAppBundleID()` is false, mirroring the `notification_linux.go`/`notification_windows.go` beeep log-and-swallow contract
- Added `notification_darwin_test.go` (`//go:build darwin`) with `TestHasAppBundleID_FalseWhenUnbundled` (the honest, load-bearing assertion — false under an unbundled `go test` binary, reproducing the wails-dev condition) and `TestSendNotification_NoBundleReturnsCleanly` (callable/smoke guard on the Go wrapper's early-return)
- Updated `TESTING.md`: Suite Manifest counts 370→371 Go / 516→517 total with a 167-05 gap-closure note; new NTF-01 traceability row for `notification_darwin_test.go`; Category U M-41 item updated to record the hardening and require re-run on a SIGNED PRODUCTION BUILD

## Task Commits

Each task was committed atomically:

1. **Task 1: Harden the darwin native notification path** - `f5caa4b4` (fix)
2. **Task 2: Add headless bundle-id regression test and register it in TESTING.md** - `cebf30eb` (test)

_Note: Task 2 combined test-file creation with the TESTING.md documentation update per the plan's `<files>` list; both changes were verified together (`go test` + `bash tests/check-traceability-paths.sh`) before the single commit._

## Files Created/Modified
- `tray_objc_darwin.m` - Added `hasValidBundleIdentifier()` C helper; `sendNotification` guards before dispatch (log-and-swallow when unbundled) and wraps the `UNUserNotificationCenter` blocks in `@try/@catch`
- `notification_darwin.go` - Added `hasAppBundleID()` Go wrapper over the new cgo symbol; `sendNotification` early-returns with a log line when unbundled
- `notification_darwin_test.go` (new) - Headless regression tests locking in the bundle-id detection primitive and the wrapper's safe-return contract
- `TESTING.md` - Suite Manifest counts + 167-05 note, NTF-01 traceability row, Category U M-41 note updated with the hardening + signed-build re-run requirement

## Decisions Made
- Guard synchronously BEFORE `dispatch_async`, not only inside it — this is the primary fix, preventing the crash-prone API from ever being reached in an unbundled process (rather than merely catching an exception raised on the main queue, which `go test` cannot exercise anyway)
- Kept `@try/@catch` around the `UNUserNotificationCenter` block as defense in depth for any residual entitlement fault on a signed build (T-167-08, accepted low-severity residual risk per the plan's threat register)
- Did not invent a test asserting the async dispatch path was prevented from crashing — the abort occurs on the main dispatch queue, which a `go test` binary never pumps, so such a test would false-pass against the pre-fix code. The `hasAppBundleID() == false` detection guard is the honest, load-bearing assertion, as directed by the plan

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Code-level M-41 crash regression is closed: an unbundled process (e.g. `wails dev`) can no longer abort the GUI process via the native notification path. Valid bundled/signed-app behavior (NTF-01/02/03) is unchanged.
- **M-41 manual UAT remains the sole open Phase 167 item** and must now be RE-RUN on a SIGNED PRODUCTION BUILD (not `wails dev`) on macOS/Windows/Linux to confirm real toast delivery and tray-hidden behavior with the hardening in place, per TESTING.md Category U.
- Run `/gsd-verify-work 167` next to drive the live M-41 re-run on a signed build, then Phase 168.

---
*Phase: 167-native-notifications*
*Completed: 2026-07-01*

## Self-Check: PASSED

All created/modified files verified present on disk; both task commits (`f5caa4b4`, `cebf30eb`) verified present in git log.
