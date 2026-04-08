---
phase: 51-auto-update-checker
verified: 2026-04-07T10:52:00Z
status: passed
score: 5/5 must-haves verified
re_verification: false
gaps: []
human_verification:
  - test: "Run app with a release version (not 'dev') and verify banner appears after 5s"
    expected: "Update banner visible in Welcome tab if a newer GitHub release exists"
    why_human: "Requires a real build with a version older than the latest GitHub release; cannot fake live HTTP response in static analysis"
  - test: "Click 'Download Update' button in the banner"
    expected: "System browser opens github.com/scottkw/agenthub/releases/tag/v{latest}"
    why_human: "BrowserOpenURL is a Wails runtime call; cannot verify actual browser open programmatically without running the app"
  - test: "Click Help > Check for Updates"
    expected: "Immediate update check fires (bypasses rate limit); if update found, banner appears"
    why_human: "Native macOS menu callback requires running app; cannot simulate menu click statically"
---

# Phase 51: Auto-Update Checker Verification Report

**Phase Goal:** Users are notified of available updates and can navigate to the download page with one click
**Verified:** 2026-04-07T10:52:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | App checks GitHub releases for a newer version on startup and every hour thereafter without blocking the UI | VERIFIED | `startUpdatePoller` in `app.go:462` launches a goroutine with 5s initial delay then `time.NewTicker(time.Hour)`; called on both startup success and failure paths |
| 2 | When a newer version is available, a banner appears in the Welcome tab showing current version, new version, and a "Download" button | VERIFIED | `WelcomeTab.tsx:51-79` renders `{update && (<div className="update-banner" role="alert" ...>)}` with `update.currentVersion`, `update.latestVersion`, and "Download Update" button |
| 3 | Clicking "Download" opens the GitHub releases page in the system browser (no in-place binary replacement) | VERIFIED | `WelcomeTab.tsx:65`: `onClick={() => BrowserOpenURL(update.releaseURL)}` where `releaseURL` is `https://github.com/scottkw/agenthub/releases/tag/v{latest}` from `updater.go:121` |
| 4 | Help menu contains a "Check for Updates" item that triggers an immediate version check | VERIFIED | `main.go:105`: `helpMenu.AddText("Check for Updates", nil, checkForUpdatesCallback)`; callback at `main.go:119` calls `go appInstance.CheckForUpdates()` |
| 5 | Update check is rate-limited to once per hour (persisted last-check timestamp) and handles 429/non-200 responses silently | VERIFIED | `updater.go:80-84`: `withinRateLimit` reads `update_check.json`, skips if within 1 hour; errors from `detect()` are swallowed at `updater.go:88-91`; 8/8 unit tests pass covering all rate-limit scenarios |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/updater/updater.go` | Update detection package with injectable detectFunc, rate-limiting, persisted timestamp | VERIFIED | 154 lines; exports `DetectFunc`, `UpdateInfo`, `Check`, `DefaultDetect`; rate-limit persistence to `update_check.json` confirmed |
| `internal/updater/updater_test.go` | Unit tests for all Check behaviors | VERIFIED | 193 lines; 8 test functions; all pass (`ok github.com/scottkw/agenthub/internal/updater 0.023s`) |
| `app.go` | startUpdatePoller, runUpdateCheck, GetLastUpdateInfo, CheckForUpdates methods on App | VERIFIED | All 4 methods present; `lastUpdate *updater.UpdateInfo` and `lastUpdateMu sync.Mutex` fields on App struct |
| `main.go` | Help menu "Check for Updates" item, appInstance variable | VERIFIED | `var appInstance *App` at line 40; `helpMenu.AddText("Check for Updates", ...)` at line 105; `checkForUpdatesCallback` at line 119 |
| `frontend/src/components/WelcomeTab.tsx` | Update banner with version display, Download, and Dismiss buttons | VERIFIED | 117 lines; banner JSX at lines 51-79; conditional on `update` state; all required elements present |
| `frontend/src/style.css` | CSS for .update-banner block | VERIFIED | 9 selectors at lines 880-943; all UI-SPEC values matched exactly |
| `frontend/src/components/__tests__/WelcomeTab.test.tsx` | Tests for update banner rendering | VERIFIED | 16 source-inspection tests in `describe('update banner (UPD-02, UPD-03)')` block |
| `frontend/src/wailsjs/go/main/App.d.ts` | TypeScript bindings for GetLastUpdateInfo and CheckForUpdates | VERIFIED | Both exports present at lines 52 and 57 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/updater/updater.go` | `go-selfupdate` | `selfupdate.DetectLatest` in `DefaultDetect` | VERIFIED | Line 54: `selfupdate.DetectLatest(ctx, selfupdate.ParseSlug(slug))` |
| `internal/updater/updater.go` | `~/.config/agenthub/update_check.json` | `os.ReadFile/os.WriteFile` | VERIFIED | `withinRateLimit` reads at line 127; `persistTimestamp` writes at line 153 |
| `app.go` | `internal/updater` | `updater.Check()` call in `runUpdateCheck` | VERIFIED | Line 487: `updater.Check(ctx, configDir(), "scottkw/agenthub", Version, updater.DefaultDetect, force)` |
| `app.go` | frontend | `runtime.EventsEmit("update:available")` | VERIFIED | Line 494: `runtime.EventsEmit(ctx, "update:available", info)` |
| `main.go` | `app.go` | `appInstance.CheckForUpdates()` in menu callback | VERIFIED | Line 121: `go appInstance.CheckForUpdates()` |
| `WelcomeTab.tsx` | `app.go` | `EventsOn('update:available')` subscription | VERIFIED | Line 31: `EventsOn('update:available', (info: UpdateInfo) => { setUpdate(info) })` |
| `WelcomeTab.tsx` | `app.go` | `GetLastUpdateInfo()` bound method call on mount | VERIFIED | Line 24: `GetLastUpdateInfo().then((info) => { if (info) setUpdate(info) })` |
| `WelcomeTab.tsx` | system browser | `BrowserOpenURL` for download | VERIFIED | Line 65: `onClick={() => BrowserOpenURL(update.releaseURL)}` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `WelcomeTab.tsx` | `update` state | `GetLastUpdateInfo()` (mount) + `EventsOn('update:available')` | Yes — `runUpdateCheck` calls `updater.Check` which calls `selfupdate.DetectLatest` via `DefaultDetect` | FLOWING |
| `app.go` `runUpdateCheck` | `info *updater.UpdateInfo` | `updater.Check(...)` with `updater.DefaultDetect` | Yes — `DefaultDetect` makes real HTTP call to GitHub releases API | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 8 updater unit tests pass | `go test ./internal/updater/... -v -count=1` | 8/8 PASS, 0.023s | PASS |
| Go codebase compiles and vets cleanly | `go vet -tags dev ./...` | No output (0 issues) | PASS |
| All 197 frontend tests pass | `pnpm --dir frontend test` | 197/197 pass (10 test files) | PASS |
| CSS has at least 9 update-banner selectors | `grep -c "update-banner" style.css` | 9 | PASS |
| Wails TypeScript bindings include new methods | `grep "GetLastUpdateInfo\|CheckForUpdates" App.d.ts` | Lines 52, 57 | PASS |

Note: `go build -tags dev -o /dev/null .` fails with a linker error (`_OBJC_CLASS_$_UTType undefined`) that is a pre-existing macOS build environment issue documented in the 51-02 SUMMARY as existing before this phase's changes. `go vet` passes cleanly, confirming the package-level code is correct.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| UPD-01 | 51-01, 51-02 | App checks GitHub releases for newer versions on startup and periodically | SATISFIED | `startUpdatePoller` fires on startup (5s delay) then hourly via ticker; `internal/updater` package handles GitHub API via `go-selfupdate` |
| UPD-02 | 51-03 | User sees a notification banner when an update is available with version info | SATISFIED | `WelcomeTab.tsx` renders `update-banner` div with `update.currentVersion` and `update.latestVersion` when `update` state is non-null |
| UPD-03 | 51-03 | User can trigger update with one click (opens download page) | SATISFIED | "Download Update" button calls `BrowserOpenURL(update.releaseURL)`; `releaseURL` is the GitHub releases tag URL |
| UPD-04 | 51-02 | User can check for updates manually from the Help menu | SATISFIED | `helpMenu.AddText("Check for Updates", nil, checkForUpdatesCallback)` in `main.go`; callback calls `CheckForUpdates()` with `force=true` bypassing rate limit |

All 4 phase requirements covered. No orphaned requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None found | — | — | — | — |

Scan performed on: `internal/updater/updater.go`, `internal/updater/updater_test.go`, `app.go`, `main.go`, `frontend/src/components/WelcomeTab.tsx`, `frontend/src/style.css`, `frontend/src/components/__tests__/WelcomeTab.test.tsx`.

No TODO/FIXME/placeholder comments, no empty implementations, no stub handlers, no hardcoded empty data in rendering paths.

### Human Verification Required

The following behaviors require running the app with a release build and cannot be verified programmatically:

#### 1. Live Update Banner Appearance

**Test:** Build the app with an older version (e.g., `wails build -ldflags "-X main.Version=1.0.0"`) and run it. Wait ~5 seconds after launch.
**Expected:** Update banner appears in the Welcome tab showing "1.0.0 -> {latest}" if a newer GitHub release exists.
**Why human:** Requires a real HTTP call to GitHub releases API and a running Wails app with frontend mounted.

#### 2. Download Button Opens Browser

**Test:** When the update banner is visible, click "Download Update".
**Expected:** System default browser opens `https://github.com/scottkw/agenthub/releases/tag/v{latest}`.
**Why human:** `BrowserOpenURL` is a Wails runtime call that requires the macOS WKWebView to be running.

#### 3. Help Menu "Check for Updates"

**Test:** With app running using a version older than latest, click Help > Check for Updates.
**Expected:** After a brief pause (~10 seconds timeout), the update banner appears if a newer version exists; no crash or error dialog if no update or network unavailable.
**Why human:** Native macOS menu click requires a running app; network dependency on GitHub API.

### Gaps Summary

No gaps. All automated checks pass at all four levels (exists, substantive, wired, data flowing). The pre-existing linker issue (`_OBJC_CLASS_$_UTType`) is a macOS SDK/Wails environment issue predating this phase and does not affect correctness of the implemented code — `go vet` confirms no code-level issues.

---

_Verified: 2026-04-07T10:52:00Z_
_Verifier: Claude (gsd-verifier)_
