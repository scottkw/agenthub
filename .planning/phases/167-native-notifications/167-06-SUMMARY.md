---
phase: 167-native-notifications
plan: 06
subsystem: notifications
tags: [macos, UNUserNotificationCenter, cgo, wails, observability, gap-closure]

# Dependency graph
requires:
  - phase: 167-native-notifications (plan 05)
    provides: darwin bundle-id guard + @try/@catch fail-safe on the native notification path (M-41 crash regression hardening)
provides:
  - Full logging coverage on every native-notification branch (attempt, authorization not-granted, authorization error, delivery-accepted/failed)
  - Proactive UNUserNotificationCenter authorization request wired to SetNotifyOnWaiting(true), surfacing the macOS permission prompt at toggle-time
  - A UNUserNotificationCenterDelegate (willPresentNotification) so notifications present while AgentHub is frontmost
  - onNotificationAuthResult exported callback emitting a notification:permission-denied Wails event on denial
  - maybeNotifyWaiting edge-fire logging and startup cache-load-failure logging
  - Cross-platform requestNotificationAuth() no-op stubs (Windows/Linux) + beeep attempt logging
affects: [167-07 (frontend denial hint), phase-167-verify-work (M-41 live re-test)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "requestNotificationAuthFunc injection seam mirrors sendNotificationFunc/saveFileDialogFunc pattern for testability"
    - "cgo exported callback hands off to a goroutine before touching appInstance.ctx (mirrors onTraySession in tray.go)"

key-files:
  created: []
  modified:
    - tray_objc_darwin.m
    - notification_darwin.go
    - app.go
    - notification_windows.go
    - notification_linux.go
    - app_test.go
    - notification_darwin_test.go
    - TESTING.md

key-decisions:
  - "Instrumentation-first: every native-notification branch (attempt/not-granted/error/delivery) now logs, so the M-41 signed-build re-test is diagnosable from captured logs instead of a black box."
  - "Proactive authorization (SetNotifyOnWaiting(true) -> requestNotificationAuthFunc) is the leading suspected fix for M-41 — during UAT the toggle showed as Off in System Settings with no prompt ever seen; this surfaces the prompt at toggle-time instead of lazily on first waiting transition."
  - "willPresentNotification delegate registered so notifications also display while AgentHub is the frontmost app, not just when backgrounded."
  - "hasValidBundleIdentifier() forward-declared near the top of tray_objc_darwin.m so the new delegate/authorization helpers (declared before the function's definition point in the file) can call it without reordering the whole file."
  - "No live-delivery test was invented — go test never pumps the main dispatch queue, so any assertion about real authorization/delivery/foreground-presentation would false-pass. Only the injectable seam call-count and the safe-return smoke guard are unit-tested."

patterns-established:
  - "Native-path proactive-authorization seam: requestNotificationAuthFunc defaults per-platform (darwin real call, Windows/Linux no-op) exactly like sendNotificationFunc."

requirements-completed: [NTF-01, NTF-04]

coverage:
  - id: D1
    description: "Every native-notification path (attempt, authorization not-granted, authorization error, delivery-completion) emits a log line for signed-build diagnosability"
    verification:
      - kind: unit
        ref: "go build . (cgo compiles the new logging/delegate/authorization code)"
        status: pass
    human_judgment: true
    rationale: "Log-line correctness on a live signed build (actually diagnosing M-41 from Console.app/stderr output) can only be confirmed during the M-41 manual re-test — go test cannot pump the main dispatch queue or produce real NSLog output from UNUserNotificationCenter."
  - id: D2
    description: "SetNotifyOnWaiting(true) proactively invokes requestNotificationAuthFunc (native authorization), disabling does not"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "app_test.go#TestSetNotifyOnWaiting_RequestsAuthorizationWhenEnabled"
        status: pass
    human_judgment: false
  - id: D3
    description: "AgentHubNotificationDelegate registered (willPresentNotification -> banner/list/sound) so notifications present while AgentHub is frontmost"
    verification:
      - kind: unit
        ref: "go build . (compiles the delegate + registerNotificationDelegate())"
        status: pass
    human_judgment: true
    rationale: "Real foreground-presentation behavior requires a live signed .app and a pumped main run loop; only compilability is unit-verifiable."
  - id: D4
    description: "onNotificationAuthResult exported callback emits notification:permission-denied Wails event on denial"
    verification:
      - kind: unit
        ref: "go build . (compiles the //export callback + goroutine hand-off + appInstance nil-guard)"
        status: pass
    human_judgment: true
    rationale: "Real denial-path event emission requires a live signed .app with an actual denied permission prompt; the frontend consumer lands in 167-07."
  - id: D5
    description: "darwin proactive-authorization wrapper safe-returns (no panic/abort) in an unbundled process"
    requirement: "NTF-01"
    verification:
      - kind: unit
        ref: "notification_darwin_test.go#TestRequestNotificationAuth_NoBundleReturnsCleanly"
        status: pass
    human_judgment: false
  - id: D6
    description: "maybeNotifyWaiting edge-fire logging and startup cache-load-failure logging added without changing trigger/edge-detection logic"
    verification:
      - kind: unit
        ref: "go test -race -short -run MaybeNotifyWaiting ."
        status: pass
    human_judgment: false
  - id: D7
    description: "Windows/Linux beeep wrappers keep parity: no-op requestNotificationAuth() + attempt log line, delivery semantics unchanged"
    verification:
      - kind: unit
        ref: "go build . (darwin build only in this environment; Windows/Linux build tags compile per go:build guards, not executed here)"
        status: pass
    human_judgment: true
    rationale: "This executor ran on darwin only; Windows/Linux compilation is asserted by go:build-tag correctness and code review, not a cross-compiled build in this session."
  - id: D8
    description: "TESTING.md updated per the standing rule (Suite Manifest note, traceability rows, Category U M-41 note) and traceability path checker passes"
    verification:
      - kind: unit
        ref: "bash tests/check-traceability-paths.sh"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-07-01
status: complete
---

# Phase 167 Plan 06: Instrument Native Notification Path + Proactive Authorization Summary

**Every native macOS notification branch (attempt, not-granted, authorization error, delivery-accepted/failed) now logs, and enabling the NotifyOnWaiting toggle proactively requests UNUserNotificationCenter authorization + registers a foreground-presentation delegate — closing the observability gap that blocked root-causing the M-41 live-delivery blocker.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-07-01T13:37:39Z (approx, first commit)
- **Completed:** 2026-07-01T13:40:32Z (approx, last commit)
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- `tray_objc_darwin.m` / `notification_darwin.go`: every native-notification branch (dispatch attempt, authorization not-granted + error, delivery-accepted/failed) now emits a log line; a `AgentHubNotificationDelegate` is registered so notifications present while AgentHub is frontmost; a new `requestNotificationAuthorization()` proactively surfaces the macOS permission prompt and reports the outcome to Go via `onNotificationAuthResult`, which emits `notification:permission-denied` for the frontend (consumed in 167-07).
- `app.go`: `SetNotifyOnWaiting(true)` now calls the new `requestNotificationAuthFunc` injection seam before the daemon-connectivity check, so the permission prompt surfaces the moment the toggle is enabled regardless of daemon state. `maybeNotifyWaiting` logs the non-waiting→waiting edge fire (session id, status, cache value); `startup` logs when the initial NotifyOnWaiting cache load fails.
- `notification_windows.go` / `notification_linux.go`: added a no-op `requestNotificationAuth()` for cross-platform parity plus a beeep attempt log line; delivery semantics unchanged.
- `app_test.go` / `notification_darwin_test.go`: `TestSetNotifyOnWaiting_RequestsAuthorizationWhenEnabled` (spy called once on enable, zero on disable) and `TestRequestNotificationAuth_NoBundleReturnsCleanly` (safe-return smoke guard) lock in the honest, headlessly-verifiable behavior.
- `TESTING.md`: Suite Manifest note, two new traceability rows (NTF-04, NTF-01), and an updated Category U M-41 note requiring log capture on the signed-build re-run.

## Task Commits

Each task was committed atomically:

1. **Task 1: Instrument the darwin native path, register a foreground presentation delegate, and add a proactive authorization entry point** - `ba59b060` (feat)
2. **Task 2: Wire proactive authorization + edge-fire logging into App, add cache-load observability, and keep cross-platform parity** - `2e205b6a` (feat)
3. **Task 3: Add headless Go tests for the proactive-authorization seam and register everything in TESTING.md** - `f93072d5` (test)

_Note: no TDD RED/GREEN split was used — this is a `type="execute"` (not `type="tdd"`) plan; tests were added alongside/after the corresponding implementation task per the plan's task boundaries._

## Files Created/Modified
- `tray_objc_darwin.m` - Added `AgentHubNotificationDelegate` (willPresentNotification), `registerNotificationDelegate()`, `requestNotificationAuthorization()`; instrumented `sendNotification`'s attempt/not-granted/delivery-completion branches with NSLog.
- `notification_darwin.go` - Added `requestNotificationAuth()` wrapper, `//export onNotificationAuthResult` callback (emits `notification:permission-denied`), and a dispatch-path log line.
- `app.go` - Added `requestNotificationAuthFunc` injection seam (defaulted in `NewApp`), wired into `SetNotifyOnWaiting(true)`; added edge-fire logging in `maybeNotifyWaiting` and cache-load-failure logging in `startup`; added the `log` import.
- `notification_windows.go` / `notification_linux.go` - Added no-op `requestNotificationAuth()` and a beeep attempt log line.
- `app_test.go` - Added `TestSetNotifyOnWaiting_RequestsAuthorizationWhenEnabled`.
- `notification_darwin_test.go` - Added `TestRequestNotificationAuth_NoBundleReturnsCleanly`.
- `TESTING.md` - Suite Manifest note, NTF-04/NTF-01 traceability rows, Category U M-41 note update.

## Decisions Made
- Instrumentation-first approach: every native path branch logs before anything else, since the plan explicitly scoped this as root-causing an invisible failure via observability rather than guessing at a single fix.
- Proactive authorization on `SetNotifyOnWaiting(true)` is the leading suspected fix (per the plan's diagnosis of the UAT finding: toggle showed "Off" in System Settings with no prompt ever seen) — implemented as the primary behavioral change, not just logging.
- `hasValidBundleIdentifier()` was forward-declared near the top of `tray_objc_darwin.m` (rather than moving the new delegate/authorization block later in the file) to keep the new code adjacent to `menuDelegate` as instructed by the plan, while still compiling given C's declare-before-use requirement.
- No test was written asserting real authorization/delivery/foreground-presentation — per the plan's explicit instruction, such a test would false-pass under `go test` (main dispatch queue never pumped). Only the injectable seam call-count and safe-return smoke guard are asserted.

## Deviations from Plan

**1. [Rule 3 - Blocking] Forward-declared `hasValidBundleIdentifier()` to fix compile ordering**
- **Found during:** Task 1 (instrumenting the darwin native path)
- **Issue:** The plan's instruction to place `registerNotificationDelegate()`/`requestNotificationAuthorization()` "alongside the existing `menuDelegate`" put those functions earlier in the file than `hasValidBundleIdentifier()`'s definition (which stays near `sendNotification`, per the plan's own reference numbering). C requires a function to be declared before use; without a forward declaration this would fail to compile.
- **Fix:** Added `int hasValidBundleIdentifier(void);` as a forward declaration directly under the `onNotificationAuthResult` extern block near the top of the file, then left `hasValidBundleIdentifier`'s actual definition untouched in its original location.
- **Files modified:** tray_objc_darwin.m
- **Verification:** `go build .` compiles cleanly.
- **Committed in:** ba59b060 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking — compile-order fix, no behavior change)
**Impact on plan:** No scope creep; purely a C-language ordering fix required to implement the plan's instructions as written.

## Issues Encountered
None beyond the compile-ordering deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 167-07 can now subscribe to the `notification:permission-denied` Wails event emitted by `onNotificationAuthResult` and build the frontend denial hint.
- The sole remaining open Phase 167 item is the M-41 live-delivery manual re-test on a SIGNED PRODUCTION BUILD (`wails build -tags wailsassets`), which must now capture the new log output (Console.app/stderr) to either confirm delivery or pinpoint the exact failing branch (not-granted, authorization error, or delivery error).

---
*Phase: 167-native-notifications*
*Completed: 2026-07-01*

## Self-Check: PASSED

All 8 modified/created files confirmed present on disk; all 3 task commits (ba59b060, 2e205b6a, f93072d5) confirmed present in git history.
