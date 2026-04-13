---
phase: 71-opencode-theming-fix
reviewed: 2026-04-13T21:20:45Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - internal/pty/session.go
  - internal/pty/session_test.go
  - internal/daemon/engine.go
  - internal/daemon/engine_test.go
  - internal/daemon/notify_theme_unix.go
  - internal/daemon/notify_theme_windows.go
  - internal/daemon/api.go
  - internal/daemon/api_test.go
  - internal/daemon/client.go
  - app.go
  - app_test.go
  - frontend/src/App.tsx
  - frontend/src/components/__tests__/App.test.tsx
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 71: Code Review Report

**Reviewed:** 2026-04-13T21:20:45Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Phase 71 Plan 05 implements the SIGUSR2 broadcast pipeline for live OpenCode theme refresh. The pipeline spans five layers: (1) `Session.Signal()` on `pty.Session`, (2) `signalThemeChange()` with platform build tags (POSIX sends SIGUSR2, Windows no-op), (3) `SessionEngine.NotifyThemeChange()` which iterates opencode sessions and broadcasts, (4) `POST /theme/notify` HTTP handler + `DaemonClient.NotifyThemeChange()`, (5) `App.NotifyThemeChange()` Wails binding called fire-and-forget from the React `handleThemeChange` callback.

The implementation is well-structured, follows existing project patterns, and handles the key concerns correctly:

- **Signal delivery race conditions:** `Session.Signal()` correctly acquires `s.mu` to snapshot `cmd`, then calls `cmd.Process.Signal()` outside the lock. If the process has already exited, `Signal()` returns an OS error which `NotifyThemeChange` logs and swallows -- the broadcast continues to remaining sessions. This is the correct behavior.
- **Lock ordering:** `NotifyThemeChange` calls `e.registry.List()` (acquires/releases `registry.mu`) then acquires `e.mu.RLock()`. Inside the loop it calls `signalThemeChange(sess)` -> `sess.Signal()` which acquires `sess.mu`. The lock acquisition order is always: registry.mu -> engine.mu -> session.mu. Since `registry.List()` releases its lock before engine.mu is acquired, there is no nested lock and no deadlock risk.
- **Platform build tags:** `notify_theme_unix.go` (`//go:build !windows`) and `notify_theme_windows.go` (`//go:build windows`) are mutually exclusive and cover all platforms. Both define `func signalThemeChange(*pty.Session) error` with the same signature. The session_test.go correctly uses `//go:build !windows` since it references `syscall.SIGUSR2`.
- **HTTP handler error path:** `handleNotifyThemeChange` checks the error return from `NotifyThemeChange`, but since the engine method always returns nil (errors are logged per-session), the 500 path is defensive dead code. This is acceptable.
- **Frontend fire-and-forget:** `NotifyThemeChange().catch(err => console.warn(...))` correctly prevents unhandled promise rejection while logging failures for debugging.

No critical issues. Two warnings for minor robustness concerns. Three informational items.

## Warnings

### WR-01: Session.Signal TOCTOU between lock release and Process.Signal call

**File:** `internal/pty/session.go:88-96`
**Issue:** `Signal()` acquires `s.mu`, copies `s.cmd` to a local variable, releases the lock, then checks `cmd == nil || cmd.Process == nil` and calls `cmd.Process.Signal(sig)`. Between the lock release (line 91) and the `Signal` call (line 95), the process could exit and be reaped by the backend's `Kill()` path. If `Kill()` sets `s.cmd = nil` after the snapshot, the local `cmd` variable still holds the old pointer, so `Process.Signal()` would return an OS-level error like "process already finished" (on Unix) or "process already released" -- not a nil-pointer panic.

This is not a crash risk (the error is caught and logged by `NotifyThemeChange`), but it means the error message from the OS may be opaque. The current design is acceptable because:
1. The error is logged with the session ID context, making debugging possible.
2. The alternative (holding the lock across Signal) would block all other session operations during signal delivery, which is worse.

The real question is whether `cmd.Process` itself can become nil between the check and the call. Looking at `os/exec`, `Process` is set during `Start()` and never nilled by the runtime -- only `ProcessState` changes after `Wait()`. So this is safe.

**Fix:** No code change required. This is a "reviewed and accepted" TOCTOU. If desired, wrap the error with more context:
```go
if err := cmd.Process.Signal(sig); err != nil {
    return fmt.Errorf("session %s: signal %v: %w", s.ID, sig, err)
}
```

### WR-02: Unused context parameter in NotifyThemeChange

**File:** `internal/daemon/engine.go:255`
**Issue:** `NotifyThemeChange(ctx context.Context)` accepts a context but never uses it. The HTTP handler passes `r.Context()` (line 227 of api.go), but if the HTTP client disconnects mid-broadcast, the signal delivery continues unaware. This is actually the correct behavior for a fire-and-forget broadcast (you want all signals delivered even if the caller disconnects), but the unused parameter is misleading -- it suggests the function respects cancellation when it does not.

**Fix:** Either remove the context parameter (breaking change if the API contract needs it for future use) or document why it is intentionally unused. Since the function signature mirrors other engine methods that accept context, keeping it for consistency is reasonable. Add a brief doc note:
```go
// NotifyThemeChange signals all active OpenCode sessions to re-query the
// terminal palette. On POSIX this sends SIGUSR2; on Windows this is a no-op.
// Errors on individual sessions are logged and do not abort the broadcast.
// Safe to call when no opencode sessions exist (returns nil).
// ctx is accepted for API consistency but not currently used (broadcast is non-cancellable).
func (e *SessionEngine) NotifyThemeChange(ctx context.Context) error {
```

## Info

### IN-01: NotifyThemeChange always returns nil -- dead error path in handler

**File:** `internal/daemon/api.go:226-232`
**Issue:** `handleNotifyThemeChange` checks `if err := a.engine.NotifyThemeChange(r.Context()); err != nil` and returns 500. However, `NotifyThemeChange` always returns `nil` -- individual signal errors are logged but not propagated. The `http.Error` path on line 228 is unreachable dead code.

This is defensively correct (if a future change makes NotifyThemeChange return errors, the handler is ready). No change needed, but it could benefit from a comment explaining the defensive nature.

**Fix:** Optional: add a comment `// NotifyThemeChange currently always returns nil; this guard is defensive.`

### IN-02: session_test.go build-tagged !windows but tests could run on all platforms

**File:** `internal/pty/session_test.go:1`
**Issue:** The test file uses `//go:build !windows` because `TestSession_Signal_NilCmd` references `syscall.SIGUSR2`. However, `TestSession_Signal_NilProcess` uses `os.Kill` which is available on all platforms. The build constraint excludes both tests from Windows. This is acceptable since the primary purpose (testing Signal with SIGUSR2) is POSIX-only, but if Windows test coverage of the nil-guard path is desired, the tests could be split into separate files.

**Fix:** No change required for this phase. If Windows coverage is desired later, extract `TestSession_Signal_NilProcess` to a separate file without the build constraint (using `os.Kill` or `os.Interrupt`).

### IN-03: Integration test assumes /bin/sh trap semantics

**File:** `internal/daemon/engine_test.go:518`
**Issue:** `TestNotifyThemeChange_RealProcess_Integration` uses a shell script with `trap 'echo SIGUSR2_RECEIVED' USR2`. This works on standard POSIX shells but relies on `/bin/sh` supporting the `trap` builtin with signal names (not numbers). All common `/bin/sh` implementations (bash, dash, zsh) support this, and the test is already gated with `runtime.GOOS == "windows"` skip and `testing.Short()` skip. The test is well-designed with READY/SIGUSR2_RECEIVED markers, hub subscription before snapshot (avoiding race), and generous 5-second timeouts.

**Fix:** No change needed. This is informational -- the test is robust and well-gated.

---

_Reviewed: 2026-04-13T21:20:45Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
