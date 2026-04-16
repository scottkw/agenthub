---
phase: 79-settings-persistence-path-browsing
reviewed: 2026-04-16T12:00:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - app.go
  - frontend/src/components/SettingsTab.tsx
  - frontend/src/components/__tests__/SettingsTab.persistence.test.tsx
  - frontend/src/style.css
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - internal/daemon/api_test.go
  - internal/daemon/engine.go
  - internal/daemon/engine_settings_test.go
  - internal/daemon/engine_test.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 79: Code Review Report

**Reviewed:** 2026-04-16T12:00:00Z
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

The settings persistence and path browsing feature adds a clean settings persistence layer (settings.json backed by the daemon's SessionEngine), a Wails-bound `OpenFileDialog` for browse-to-select, and corresponding frontend UI with save confirmation. The Go backend code is well-structured with proper mutex usage and test coverage. The Wails binding stubs (App.d.ts, App.js) are correctly wired. The CSS additions are well-scoped with BEM naming.

Key concerns: one input validation gap in the CLI path update flow (no sanitization of the `name` key), an inconsistent clipboard API that will break in Wails production builds, and missing error handling on a disk write that silently loses user data.

## Critical Issues

### CR-01: Inconsistent clipboard API -- `navigator.clipboard` will fail in Wails production builds

**File:** `frontend/src/components/SettingsTab.tsx:129`
**Issue:** `handleCopyPassword` (line 129) uses `navigator.clipboard.writeText()` while `handleCopyURL` (line 136) correctly uses the Wails `ClipboardSetText` runtime binding. In Wails production builds, `navigator.clipboard` is often unavailable (requires secure context / HTTPS origin, which the Wails WebView does not always provide). This means copying the LAN password will silently fail with an unhandled promise rejection in production.
**Fix:**
```typescript
async function handleCopyPassword() {
  if (!localPassword) return
  await ClipboardSetText(localPassword)
  setCopied(true)
  setTimeout(() => setCopied(false), 1500)
}
```

## Warnings

### WR-01: No input validation on CLI name key in UpdateCLIPath

**File:** `internal/daemon/engine.go:278`
**Issue:** `UpdateCLIPath(name, path string)` validates that `path` exists on disk (via `os.Stat`), but performs no validation on the `name` parameter. The `name` is used as a key in `cliPaths` and serialized directly into `settings.json`. While the current callers pass well-known CLI names (e.g., "claude", "tailscale"), the Wails-bound `UpdateCLIPath` method in `app.go:307` passes user-supplied strings directly through to the daemon client with no sanitization. An empty string or excessively long string as a key would be persisted silently. This is a defense-in-depth concern.
**Fix:**
```go
func (e *SessionEngine) UpdateCLIPath(name, path string) error {
    if name == "" {
        return fmt.Errorf("CLI name must not be empty")
    }
    if len(name) > 64 {
        return fmt.Errorf("CLI name too long (max 64 chars)")
    }
    if _, err := os.Stat(path); err != nil {
        return fmt.Errorf("custom CLI path %q: %w", path, err)
    }
    // ... rest unchanged
}
```

### WR-02: saveSettingsToDisk silently drops write errors

**File:** `internal/daemon/engine.go:91-98`
**Issue:** Both `json.Marshal` errors (line 94) and `os.WriteFile` errors (line 97) are silently discarded. If the config directory becomes read-only, the disk is full, or any other I/O error occurs, the user's path changes appear saved (the in-memory map is updated) but will be lost on daemon restart. Per project conventions ("Silent Fallbacks: `or {}` converts hard failures (informative) into silent corruption (expensive). Let it crash."), errors should at minimum be logged.
**Fix:**
```go
func (e *SessionEngine) saveSettingsToDisk() {
    s := daemonSettings{CLIPaths: e.cliPaths}
    data, err := json.Marshal(s)
    if err != nil {
        log.Printf("[error] saveSettingsToDisk: marshal: %v", err)
        return
    }
    if err := os.WriteFile(settingsPath(e.configDir), data, 0600); err != nil {
        log.Printf("[error] saveSettingsToDisk: write %s: %v", settingsPath(e.configDir), err)
    }
}
```
Consider also propagating the error to `UpdateCLIPath` so the frontend can display it:
```go
func (e *SessionEngine) saveSettingsToDisk() error {
    // ...
    return os.WriteFile(settingsPath(e.configDir), data, 0600)
}
```

### WR-03: Port value can be NaN or 0 when passed to StartWebServer

**File:** `frontend/src/components/SettingsTab.tsx:317`
**Issue:** The port input's `onChange` handler uses `Number(e.target.value)`. When the user clears the input field, `Number("")` evaluates to `0`. The HTML `min={1}` / `max={65535}` attributes only provide visual hints -- they do not prevent submission. `handleToggleServer` (line 226) passes `selectedPort` directly to `StartWebServer()` without validating it is a valid port number (1-65535). A port of 0 will cause the Go side to bind to a random port, which may not be the intended behavior.
**Fix:**
```typescript
async function handleToggleServer() {
  setServerError(null)
  if (!isServerRunning && (selectedPort < 1 || selectedPort > 65535 || isNaN(selectedPort))) {
    setServerError('Port must be between 1 and 65535')
    return
  }
  setServerLoading(true)
  // ... rest unchanged
}
```

## Info

### IN-01: console.error left in production code

**File:** `frontend/src/components/SettingsTab.tsx:94`
**Issue:** `console.error('[SettingsTab] loadWebState:', err)` is used for error reporting in the web state loading effect. While not harmful, it is inconsistent with the rest of the codebase where Wails errors are typically caught and stored in state for UI display, or silently ignored (as in the `GetCLIPaths` `.catch(() => {})` on line 107).
**Fix:** Either remove the console.error and handle with state, or add a comment explaining why this specific error is logged to console rather than displayed in UI.

### IN-02: Frontend tests use raw string matching instead of rendering

**File:** `frontend/src/components/__tests__/SettingsTab.persistence.test.tsx:1-98`
**Issue:** All tests import the component source as a raw string (`?raw` suffix) and assert on string contents (e.g., `expect(raw).toContain('setSaved')`). This verifies code structure exists but not that it actually works correctly -- a rename, dead code elimination, or logic change could break functionality while tests still pass. This is a test quality concern, not a blocking issue -- the approach is pragmatic for verifying Wails binding wiring.
**Fix:** Consider adding at least one render-based test (via `@testing-library/react` with Wails mocks) for the critical save-and-reload flow to complement the structural tests.

---

_Reviewed: 2026-04-16T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
