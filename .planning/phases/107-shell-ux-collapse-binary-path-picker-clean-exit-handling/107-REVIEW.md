---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
reviewed: 2026-05-13T00:00:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - internal/daemon/engine.go
  - internal/daemon/api.go
  - internal/daemon/client.go
  - app.go
  - frontend/src/App.tsx
  - frontend/src/components/NewSessionModal.tsx
  - frontend/src/components/SettingsTab.tsx
findings:
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 107: Code Review Report

**Reviewed:** 2026-05-13
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Phase 107 delivers three deltas: the NewSessionModal shell-row collapse (SHELL-10), the Settings shell binary path picker (SHELL-11), and the clean-exit tab handling (SHELL-12). The Go backend is well-structured — mutex discipline on `shellPath` is consistent with peer fields, the validation chain in `SetShellPath` is sane, and `resolveDefaultShellPath` never returns empty. The `SHELL_CLIS` set in `App.tsx` still contains `'shell'`, and `TabBar.agentBadgeModifier` correctly collapses all shell variants. No stale `shell:NAME` consumers remain in the reviewed files.

One BLOCKER was found: SHELL-12 silently dropped the `autoCloseRef` guard that previously honored the user's "Auto-close tab on exit" preference, leaving that Settings toggle as a no-op for exit-code-0 sessions. Three warnings follow — two are quality issues in the new Settings row (misleading "Saved!" when shell-path save fails; missing `MaxBytesReader` on the handler peer to its neighbors), one is a logic correctness issue in `SetShellPath` (directories pass the executable-bit check). Three info items cover dead code, a dead UI path, and an accessibility nit.

---

## Critical Issues

### CR-01: `autoCloseRef` not consulted in SHELL-12 exit handler — user preference silently ignored

**File:** `frontend/src/App.tsx:550-553`

**Issue:** The Phase 107-04 SHELL-12 rewrite of the `session:exit` handler unconditionally calls `handleCloseTabRef.current?.(data.sessionId)` for every `exitCode === 0` event, then returns. The `autoCloseRef.current` value — loaded on line 397 from `GetAutoCloseSession()` — is never checked. The pre-SHELL-12 handler used `autoCloseRef.current` to gate the 5-second countdown: when the user had set "Auto-close tab on exit" to OFF, the countdown was skipped and the tab stayed open. That gate is now gone.

Result: a user who has explicitly disabled "Auto-close tab on exit" in Settings will see tabs close immediately on clean shell or agent exits, contradicting the setting they configured. The Settings toggle is rendered, persisted to the daemon, and loaded on mount — but its value has no effect on behavior. This is a broken contract with the user.

**Fix:** Restore the `autoCloseRef` guard inside the `exitCode === 0` branch. The immediate-close path is still correct when `autoCloseRef.current` is `true`; when `false`, fall through to the `setSessionExits` path with `countdown: -1` (no countdown, no auto-close — tab stays open and shows the ExitToast):

```typescript
const offExit = EventsOn(
  'session:exit',
  (data: { sessionId: string; exitCode: number; sessionName: string; cli: string; duration: number; finalStatus: string }) => {
    if (data.exitCode === 0) {
      if (autoCloseRef.current) {
        // Auto-close enabled: close tab immediately (SHELL-12)
        void handleCloseTabRef.current?.(data.sessionId)
        return
      }
      // Auto-close disabled: fall through to show ExitToast with no countdown
    }
    const exitState: ExitState = {
      sessionId: data.sessionId,
      sessionName: data.sessionName,
      cli: data.cli,
      exitCode: data.exitCode,
      duration: data.duration,
      finalStatus: data.finalStatus,
      countdown: -1,
      cancelled: false,
    }
    setSessionExits(prev => ({ ...prev, [data.sessionId]: exitState }))
  }
)
```

---

## Warnings

### WR-01: `SetShellPath` accepts directory paths — executable-bit check insufficient

**File:** `internal/daemon/engine.go:691-697`

**Issue:** `SetShellPath` validates the path using `os.Stat` and checks `info.Mode()&0111 != 0`. On every POSIX host, directories have their execute bit set (e.g. `drwxr-xr-x`). A path like `/tmp` or `/usr/bin` passes both the existence check and the executable-bit check but is not a valid shell binary. When the daemon later tries to `exec` a directory as a shell, the process creation fails with a confusing error at session-spawn time rather than at settings-save time, and the stored path persists in `settings.json`.

`UpdateCLIPath` (the pre-existing peer) has the same gap, so this is consistent behavior — but `SetShellPath` is new code introduced in this phase and has an explicit executable-bit check that creates an expectation of thoroughness.

**Fix:** Add an `IsDir()` check after `os.Stat`:

```go
info, err := os.Stat(path)
if err != nil {
    return fmt.Errorf("path %q does not exist or is not executable", path)
}
if info.IsDir() {
    return fmt.Errorf("path %q is a directory, not an executable", path)
}
if info.Mode()&0111 == 0 {
    return fmt.Errorf("path %q does not exist or is not executable", path)
}
```

---

### WR-02: "Saved!" fires even when shell-path validation fails — misleading success feedback

**File:** `frontend/src/components/SettingsTab.tsx:257-265`

**Issue:** In `handleSaveCLIPaths`, the shell-path `SetShellPath` call is wrapped in an inner `try/catch` that swallows the error and sets `shellPathError`. Because the inner catch does not re-throw, execution continues to `setSaved(true)` on line 264 regardless of whether the shell-path save succeeded. The result: when a user enters an invalid shell binary path and clicks "Save Paths", they see both an inline error below the field ("path does not exist or is not executable") AND the "Saved!" success indicator on the button simultaneously. The success indicator is false.

```
// Current execution flow when SetShellPath returns 400:
inner catch → setShellPathError('…')
              ↓ swallowed
outer try continues → setSaved(true)   ← false positive
```

**Fix:** Track whether the shell-path save failed and suppress `setSaved(true)` if so:

```typescript
let shellPathOk = true
try {
  await SetShellPath(shellPath.trim())
  setShellPathError('')
} catch (err) {
  shellPathOk = false
  setShellPathError(err instanceof Error ? err.message : String(err))
}
if (shellPathOk) {
  setSaved(true)
  setTimeout(() => setSaved(false), 1500)
}
```

Alternatively, re-throw from the inner catch and let the outer catch surface the error in the top-level `error` state (consistent with how other path save failures are handled).

---

### WR-03: `handleUpdateShellPath` missing `http.MaxBytesReader` — inconsistent with peer handlers

**File:** `internal/daemon/api.go:580-593`

**Issue:** Every other `PATCH /settings/*` handler that accepts a body guards against oversized requests with `r.Body = http.MaxBytesReader(w, r.Body, 8192)` (see `handleSetPluginSettings` line 627, `handleSetSearchConfig` line 649, `handleSetWebLinksConfig` line 671, `handleSetImageConfig` line 714). The new `handleUpdateShellPath` handler decodes the body with no size cap. The daemon's Unix socket is owner-only (`0600`), so exploitation requires local access — but the cap is a defense-in-depth pattern established by the existing codebase for exactly this class of handler, and omitting it for the new handler creates an inconsistency that will mislead future maintainers.

The sibling `handleSetStartMinimized` and `handleUpdateShellWebShareWarned` also lack the cap (pre-existing gap), but this review focuses on new code.

**Fix:**

```go
func (a *API) handleUpdateShellPath(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 8192) // add this line
    var req struct {
        Value string `json:"value"`
    }
    // ... rest unchanged
```

---

## Info

### IN-01: `autoCloseRef` populated but never read — dead code after SHELL-12

**File:** `frontend/src/App.tsx:197, 397`

**Issue:** `autoCloseRef` is declared on line 197 and populated from `GetAutoCloseSession()` on line 397. After the SHELL-12 rewrite the ref is never read anywhere in the file. Even if CR-01 is fixed (the ref should be consulted in the exit handler), the load on line 397 should remain; this note is a reminder to confirm the fix closes the dead-read gap.

**Fix:** Restore the `autoCloseRef.current` check per the CR-01 fix. No separate change needed beyond that.

---

### IN-02: `ExitCountdownBanner` and `TabBar` `exitCountdowns` prop are now unreachable for exit-code-0

**File:** `frontend/src/App.tsx:1056-1061, 1169`

**Issue:** The `session:exit` handler with `exitCode === 0` now calls `handleCloseTab` and returns immediately without writing to `sessionExits`. The `ExitCountdownBanner` (line 1169) is rendered only when `sessionExits[tab.sessionId].exitCode === 0 && countdown > 0`, and the `TabBar` `exitCountdowns` prop (line 1056-1061) filters `sessionExits` for the same condition. Neither can ever receive a value since the exit-code-0 path bypasses `setSessionExits` entirely. Both paths are dead UI.

This is a consequence of the SHELL-12 design choice to close immediately rather than show a countdown. If CR-01 is fixed (auto-close respects the preference), exit-code-0 events with `autoCloseRef.current === false` will reach `setSessionExits` with `countdown: -1`, so the countdown UI remains unreachable regardless. The countdown infrastructure (`countdownTimers`, `ExitCountdownBanner`, the TabBar prop) can be cleaned up when these paths are confirmed intentionally retired.

**Fix:** If the countdown behavior is intentionally removed (per SHELL-12's design), delete the dead countdown UI paths (`ExitCountdownBanner` render, `exitCountdowns` TabBar prop, `countdownTimers` ref, `handleKeepOpen` callback). If not intentional, restore the countdown path for the `autoClose=true` branch. Either way, document the decision.

---

### IN-03: `aria-describedby` references a non-existent element when no shell-path error is showing

**File:** `frontend/src/components/SettingsTab.tsx:717, 728`

**Issue:** The shell-path input has `aria-describedby="settings-shell-path-desc"` unconditionally (line 717). The element with `id="settings-shell-path-desc"` is the error `<p>` tag rendered only when `shellPathError` is truthy (line 728). When there is no error, the `aria-describedby` attribute points to an element that does not exist in the DOM. Assistive technologies tolerate dangling `aria-describedby` references silently, but it is a WCAG 2.1 SC 1.3.1 advisory violation and will trip axe-core lint rules.

**Fix:** Either render a hidden (but present) description element at all times, or remove `aria-describedby` from the input and add it conditionally only when `shellPathError` is set:

```tsx
<input
  id="settings-shell-path"
  className="settings-panel__path-input"
  type="text"
  value={shellPath}
  onChange={(e) => setShellPath(e.target.value)}
  placeholder="e.g. /bin/zsh"
  aria-label="Shell binary path"
  aria-describedby={shellPathError ? 'settings-shell-path-desc' : undefined}
/>
{shellPathError && (
  <p id="settings-shell-path-desc" className="settings-panel__error" role="alert">
    {shellPathError}
  </p>
)}
```

---

_Reviewed: 2026-05-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
