---
phase: 79-settings-persistence-path-browsing
plan: 01
subsystem: daemon/settings, wails-bindings
tags: [persistence, settings, file-dialog, wails-binding]
dependency_graph:
  requires: []
  provides: [settings-persistence, open-file-dialog-binding, get-cli-paths-binding]
  affects: [79-02]
tech_stack:
  added: []
  patterns: [json-file-persistence, wails-bound-method, nil-client-guard]
key_files:
  created:
    - internal/daemon/engine_settings_test.go
  modified:
    - internal/daemon/engine.go
    - internal/daemon/engine_test.go
    - internal/daemon/api_test.go
    - app.go
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts
decisions:
  - "settings.json uses 0600 permissions (user-only read/write) matching existing ct_disclosed pattern"
  - "loadSettingsFromDisk silently ignores missing/corrupt files (first-run tolerance)"
  - "saveSettingsToDisk called inside mu.Lock to ensure atomic read-modify-write of cliPaths"
  - "Tests isolated from real settings.json by overriding configDir to t.TempDir()"
metrics:
  duration: "4m 13s"
  completed: "2026-04-16"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 7
---

# Phase 79 Plan 01: Settings Persistence & Wails Bindings Summary

JSON-backed settings persistence for CLI path overrides with 0600 permissions, plus OpenFileDialog and GetCLIPaths Wails-bound methods for frontend consumption.

## Task Results

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add settings.json persistence to SessionEngine | 8155283 | engine.go, engine_settings_test.go, engine_test.go, api_test.go |
| 2 | Add OpenFileDialog and GetCLIPaths Wails bindings | d2f39ba | app.go, App.js, App.d.ts |

## Implementation Details

### Task 1: Settings Persistence

Added `daemonSettings` struct with `cliPaths` JSON field, `loadSettingsFromDisk` (called at engine construction), and `saveSettingsToDisk` (called inside `UpdateCLIPath`'s lock). The `configDir` field caches the daemon config directory path on the `SessionEngine` struct.

Key design decisions:
- `saveSettingsToDisk` does NOT acquire `e.mu` -- it is called while the lock is already held by `UpdateCLIPath`, accessing `e.cliPaths` directly
- Missing settings.json on first run is silently ignored (not an error)
- Corrupt JSON is silently ignored (graceful degradation)
- File written with 0600 permissions (T-79-01 mitigation)

Four tests cover round-trip persistence (SET-01), tailscale path persistence (SET-02), missing file tolerance, and file permissions.

### Task 2: Wails Bindings

`OpenFileDialog` mirrors the existing `OpenDirectoryDialog` pattern but uses `runtime.OpenFileDialog` with `ShowHiddenFiles: true` (executables may be in hidden directories like `.local/bin`).

`GetCLIPaths` follows the nil-client guard pattern from `UpdateCLIPath`, delegating to `DaemonClient.GetCLIPaths()`.

Both methods exported in App.js and type-declared in App.d.ts following existing format conventions.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test isolation from real settings.json**
- **Found during:** Task 1 verification
- **Issue:** `NewSessionEngine()` now loads from the real `~/.config/agenthub/settings.json`, causing `TestEngineResolveCLI` and `TestAPIGetCLIPaths` to fail because a previous integration test had persisted `claude=/bin/cat` to disk
- **Fix:** Added `engine.configDir = t.TempDir()` and `engine.cliPaths = make(map[string]string)` after `NewSessionEngine()` in `TestEngineResolveCLI` and the `testDaemon` helper
- **Files modified:** internal/daemon/engine_test.go, internal/daemon/api_test.go
- **Commit:** 8155283

## Verification Results

- `go test ./internal/daemon/... -run "TestSettings|TestTailscalePath" -count=1`: PASS (4 tests)
- `go test ./internal/daemon/... -count=1 -short`: PASS (all daemon tests)
- `go build -o /dev/null .`: PASS (clean compilation)
- App.js contains OpenFileDialog and GetCLIPaths exports: verified (2 matches)
- App.d.ts contains OpenFileDialog and GetCLIPaths declarations: verified (2 matches)

## Self-Check: PASSED

All 7 files verified present. Both commit hashes (8155283, d2f39ba) verified in git log.
