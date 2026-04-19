---
phase: 84-session-auto-close
reviewed: 2026-04-19T00:00:00Z
depth: standard
files_reviewed: 18
files_reviewed_list:
  - app.go
  - frontend/src/App.tsx
  - frontend/src/components/__tests__/App.exit.test.tsx
  - frontend/src/components/__tests__/ExitCountdownBanner.test.tsx
  - frontend/src/components/__tests__/ExitToast.test.tsx
  - frontend/src/components/ExitCountdownBanner.tsx
  - frontend/src/components/ExitToast.tsx
  - frontend/src/components/SettingsTab.tsx
  - frontend/src/components/TabBar.tsx
  - frontend/src/style.css
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - internal/daemon/api.go
  - internal/daemon/client.go
  - internal/daemon/engine_test.go
  - internal/daemon/engine.go
  - internal/daemon/types.go
  - internal/pty/session.go
findings:
  critical: 0
  warning: 4
  info: 4
  total: 8
status: issues_found
---

# Phase 84: Code Review Report

**Reviewed:** 2026-04-19
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found

## Summary

Phase 84 adds session auto-close on clean exit: a `session:exit` Wails event, an `ExitCountdownBanner` overlay, an `ExitToast` notification, a per-session exit-state machine in `App.tsx`, a persistent `AutoCloseSession` setting stored in the daemon, and the associated Go API endpoints and client. The architecture is sound and the data flow is well-structured.

Four warnings and four info items were found. No critical issues. The most important warnings are (1) a `Duration` field calculated from `CreatedAt` when the session is already stopped — producing a cumulative elapsed time rather than the actual session run duration — and (2) a missing `status` field in the TypeScript `SessionInfo` stub, which means `sessionStatuses` seeded from the `ListSessions` payload at startup will always be stale until the first status event fires.

---

## Warnings

### WR-01: `Duration` Calculated at List Time, Not at Exit Time

**File:** `internal/daemon/engine.go:261`

**Issue:** `Duration` is computed as `int(time.Since(s.CreatedAt).Seconds())` inside `ListSessions()`. `ListSessions` is polled every 500 ms by `pollSessionStatus` in `app.go`. The first poll that observes `State == "stopped"` will report a duration close to the actual run time, but any subsequent call to `ListSessions` (e.g., the tray poller at 5 s intervals or the daemon-manager panel poller at 3 s) will return a larger number. Because `emitExitEvent` immediately uses the value returned by that first matching poll, the duration sent to the frontend depends on polling latency and can be off by up to the 500 ms poll interval. More importantly, if polling is slow (e.g., daemon under load), the reported duration will be inflated. The correct approach is to capture exit time on the `Session` struct at the moment the process exits, not recalculate it on every list call.

**Fix:** Record an `ExitedAt time.Time` on `*pty.Session` in `SetState(StateStopped)` (or inside `WaitForExit`), then compute duration as `ExitedAt.Sub(s.CreatedAt)` in `ListSessions` instead of `time.Since(s.CreatedAt)`.

```go
// In pty/session.go — add ExitedAt field and capture it:
type Session struct {
    // ... existing fields ...
    ExitedAt time.Time // zero if still running
}

func (s *Session) SetState(state SessionState) {
    s.mu.Lock()
    s.State = state
    if state == StateStopped && s.ExitedAt.IsZero() {
        s.ExitedAt = time.Now()
    }
    s.mu.Unlock()
}

// In engine.go ListSessions — replace:
dur := int(time.Since(s.CreatedAt).Seconds())
// with:
var dur int
if !s.ExitedAt.IsZero() {
    dur = int(s.ExitedAt.Sub(s.CreatedAt).Seconds())
} else {
    dur = int(time.Since(s.CreatedAt).Seconds())
}
```

---

### WR-02: `SessionInfo` TypeScript Stub Missing `status` Field

**File:** `frontend/src/wailsjs/go/main/App.d.ts:4-12`

**Issue:** The `SessionInfo` interface in the hand-maintained Wails stub does not include the `status` field (the heuristic status: "running" / "idle" / "waiting" / "errored"), even though the Go `daemon.SessionInfo` type has `Status string` and the daemon serialises it as `"status"`. `App.tsx` seeds `sessionStatuses` at startup by calling `GetSessionStatus(s.id)` for each restored session (line 221), rather than using `s.status` from the list payload. This works, but is an unnecessary extra RPC per session and will not populate `sessionStatuses` until those async calls resolve. More importantly, it means any new code that tries to read `session.status` from a `SessionInfo` object will get a TypeScript compile error because the field is not declared in the stub.

**Fix:** Add `status: string` to the `SessionInfo` interface:

```typescript
export interface SessionInfo {
  id: string
  cli: string
  name: string
  state: string
  status: string        // heuristic status: running | idle | waiting | errored
  createdAt: string
  hostname: string
  webEnabled: boolean
}
```

Then the init loop in `App.tsx` can seed `sessionStatuses` directly without the extra `GetSessionStatus` RPC:

```typescript
const statusMap: Record<string, string> = {}
sessions.forEach((s) => { statusMap[s.id] = s.status })
setSessionStatuses(statusMap)
```

---

### WR-03: Countdown Timer Fires After Tab Closed in Race Window

**File:** `frontend/src/App.tsx:358-380`

**Issue:** When the `session:exit` event fires and `autoCloseRef.current` is true, a `setInterval` timer is started. The timer's callback calls `handleCloseTabRef.current?.(data.sessionId)`. However `handleCloseTab` is async: it awaits `KillSession`, then calls `setTabs`. If the user manually closes the tab (calling `handleCloseTab` themselves) between when the timer fires and when the async `KillSession` call completes, `KillSession` will be called twice for the same session ID. The second call will fail with a daemon error (session not found), which is swallowed with `console.warn`, so there is no user-visible crash — but it is a logic error.

Additionally, `handleCloseTab` is guarded only by the presence of `countdownTimers.current[id]` being non-null, but `clearInterval` has already been called before the `void handleCloseTabRef.current?.(data.sessionId)` line, so the guard (`entry.cancelled`) is checked via `setSessionExits` updater — that check is correct. The actual race is that double-close can occur if the user closes the tab while the interval callback is executing.

**Fix:** In `handleCloseTab`, check whether the session tab still exists before issuing `KillSession`:

```typescript
const handleCloseTab = useCallback(async (id: string) => {
  // Guard: tab may already be gone if auto-close and manual close race
  setTabs(prev => {
    if (!prev.find(t => t.id === id)) return prev  // already closed
    return prev  // proceed; actual removal happens below
  })
  // ... rest of existing logic
}, [activeId, webEnabled, sessionExits])
```

A simpler guard: check `tabs.find(t => t.id === id)` before the `KillSession` call and bail early if the tab is already gone. Since `tabs` is a React state snapshot, reading it directly in the callback body is safe for this guard.

---

### WR-04: `loadSettingsFromDisk` Calls `saveSettingsToDisk` While Holding `e.mu.Lock()`

**File:** `internal/daemon/engine.go:113-115`

**Issue:** `loadSettingsFromDisk` acquires `e.mu.Lock()` at line 97, then conditionally calls `e.saveSettingsToDisk()` at line 115. `saveSettingsToDisk` is documented as "Caller holds e.mu.Lock()" — so the lock ordering is intentional. However, `saveSettingsToDisk` calls `os.WriteFile`, which is a blocking I/O call under the mutex. This blocks all other goroutines that need `e.mu` for the duration of the file write. This is only called at startup (`NewSessionEngine`), so the practical impact is low, but it is a latent correctness risk if the config directory is on a slow filesystem (e.g., network mount).

This is a marginal warning given the startup-only context, but worth tracking.

**Fix:** Collect the dirty state under the lock, then perform the write outside it:

```go
e.mu.Lock()
// ... populate fields ...
dirty := false
// ... detect dirty ...
e.mu.Unlock()

if dirty {
    e.mu.Lock()
    e.saveSettingsToDisk()
    e.mu.Unlock()
}
```

Alternatively, refactor `saveSettingsToDisk` to snapshot the data under the lock and write outside.

---

## Info

### IN-01: `pollSessionStatus` Deadline Does Not Reset on Activity

**File:** `app.go:203-240`

**Issue:** `pollSessionStatus` uses a fixed 5-minute deadline (`300 * time.Second`) from the time the session is created. For a session that runs for more than 5 minutes, the poller will exit and never emit `session:exit`. The comment says "extended to 5min for long-running agents", but real agent tasks routinely exceed this. If the session exits after 5 minutes, the frontend will never receive the `session:exit` event and the auto-close feature (Phase 84) will silently not fire.

**Fix:** Consider a loop-based approach that does not use a wall-clock deadline, or use a very large timeout (e.g., 24 h), or poll indefinitely until the session is found in state "stopped" or removed from the daemon. Since the goroutine exits when the session disappears from `ListSessions`, there is no goroutine leak risk with an indefinite poll.

---

### IN-02: `ExitToast` Renders All Exits Including Non-Active-Tab Sessions

**File:** `frontend/src/App.tsx:931-935`, `frontend/src/components/ExitToast.tsx`

**Issue:** `ExitToast` is passed the full `sessionExits` map, which includes exits for sessions that may have had their tab closed by the auto-close mechanism (the tab is removed but the toast entry persists until `handleDismissExit` is called, or it is cleared by the countdown). This is intentional design (toasts persist after tab close), but when a session exits with `exitCode !== 0`, the `ExitToast` entry is created (line 343) but the countdown stays at `-1` and there is no auto-dismiss timer for error exits. The user must manually dismiss every error-exit toast. For long-running daemon sessions that crash repeatedly, this could accumulate many toast entries with no automatic cleanup.

**Fix (suggestion, not a bug):** Consider adding an auto-dismiss timeout (e.g., 30 s) for error exits without a countdown, or cap the number of visible toast entries.

---

### IN-03: `handleCopyURL` Has No Error Handling

**File:** `frontend/src/components/SettingsTab.tsx:173-177`

**Issue:** `handleCopyURL` calls `ClipboardSetText(serverURL)` but does not catch the returned promise rejection, unlike the adjacent `handleCopyPassword` which wraps the call in a `try/catch`. An unhandled promise rejection in a Wails WebView will be silently ignored in production but may produce a console warning.

**Fix:**

```typescript
async function handleCopyURL() {
  if (!serverURL) return
  try {
    await ClipboardSetText(serverURL)
    setUrlCopied(true)
    setTimeout(() => setUrlCopied(false), 1500)
  } catch {
    // Clipboard write failed — no user-visible action needed
  }
}
```

---

### IN-04: `spyBackend` in `engine_test.go` Returns Nil Hub — `KillSession` Panics

**File:** `internal/daemon/engine_test.go:245-257`

**Issue:** `spyBackend.Create` returns a `*pty.Session` with no `cmd` field set (`cmd` is nil). `engine.go`'s `CreateSession` goroutine does `go func() { <-hub.Done(); exitCode := sess.WaitForExit() ... }()`. `WaitForExit` handles `cmd == nil` safely (returns 0). However, `spyBackend.Kill` returns nil without actually stopping anything, so the hub read goroutine (if any) may leak. In the existing tests this is not observable because the spy tests don't call `KillSession` on spy-backend sessions through the full cleanup path. Not a correctness issue in the test output, but a minor test hygiene gap.

This is low severity and only relevant if tests are expanded to use `KillSession` with `spyBackend`.

---

_Reviewed: 2026-04-19_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
