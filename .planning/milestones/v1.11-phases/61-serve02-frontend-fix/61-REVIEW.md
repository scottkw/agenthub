---
phase: 61-serve02-frontend-fix
reviewed: 2026-04-10T12:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - app.go
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/App.tsx
findings:
  critical: 1
  warning: 5
  info: 3
  total: 9
status: issues_found
---

# Phase 61: Code Review Report

**Reviewed:** 2026-04-10T12:00:00Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

This changeset migrates the App layer from direct in-process session management to a daemon-client architecture over a Unix socket. The Go backend (`app.go`) is now a thin Wails-binding shell delegating all operations to `DaemonClient`. The TypeScript declaration file (`App.d.ts`) and React root component (`App.tsx`) were updated to match the new API surface (additional parameters on `CreateSession`, new methods for daemon error handling, remote sessions, Tailscale, and local network mode).

The architecture change is sound. The primary concerns are: (1) a security issue with unbounded JSON decoding from remote peers, (2) duplicate goroutine spawning on `RetryDaemon`, (3) stale closure references in React callbacks, and (4) a missing dependency in a `useEffect` hook.

## Critical Issues

### CR-01: Unbounded JSON body decoding from remote peers (potential DoS)

**File:** `app.go:531-534`
**Issue:** `fetchRemoteSessions` uses `json.NewDecoder(resp.Body).Decode(&items)` to decode the response body from remote tailnet peers without any size limit. A malicious or compromised peer could send an arbitrarily large JSON response, consuming unbounded memory and potentially crashing the application. The 5-second context timeout limits connection time but does not limit response body size once streaming has begun.
**Fix:**
```go
// Limit the response body to a reasonable size (e.g., 1 MB)
limited := io.LimitReader(resp.Body, 1<<20)
if err := json.NewDecoder(limited).Decode(&items); err != nil {
    return nil
}
```

## Warnings

### WR-01: RetryDaemon spawns duplicate health pollers without cancelling previous ones

**File:** `app.go:133-145`
**Issue:** `RetryDaemon` calls `a.startHealthPoller(a.ctx)` each time it succeeds. Since `startHealthPoller` launches a goroutine with a `time.Ticker` that only exits when `ctx` is cancelled, repeated retries accumulate duplicate polling goroutines. The same `a.ctx` is used each time, so none of the earlier goroutines are cancelled. This applies to the health poller only (tray and update pollers are started once in `startup` and not re-started on retry), but it still results in redundant goroutines making concurrent Tailscale health checks.
**Fix:** Track whether the health poller is already running (e.g., with a `sync.Once` or a boolean flag), or use a child context per poller that can be cancelled before starting a new one:
```go
// In the App struct, add:
healthCancel context.CancelFunc

// In RetryDaemon, before starting the poller:
if a.healthCancel != nil {
    a.healthCancel()
}
healthCtx, cancel := context.WithCancel(a.ctx)
a.healthCancel = cancel
a.startHealthPoller(healthCtx)
```

### WR-02: Stale closure over `remotePeers` in useEffect dependency array

**File:** `frontend/src/App.tsx:376-397`
**Issue:** The remote sessions polling `useEffect` (line 376) references `remotePeers.length` on line 380 (`if (remotePeers.length === 0) setRemoteLoading(true)`) but does not include `remotePeers` in its dependency array (only `activeId` is listed). This means the loading indicator logic captures the initial empty array and will always evaluate `remotePeers.length === 0` as `true` on the first render of that effect, even if peers were previously loaded. While this is a minor UX issue (the loading spinner briefly re-appears when switching away and back), it also indicates a stale closure that could produce more serious bugs if the logic evolves.
**Fix:** Either add `remotePeers` to the dependency array, or use a ref for the loading-gate check:
```tsx
// Option A: use functional check via ref
const remotePeersRef = useRef(remotePeers)
remotePeersRef.current = remotePeers
// Then in the effect:
if (remotePeersRef.current.length === 0) setRemoteLoading(true)
```

### WR-03: Tab constant objects recreated on every render

**File:** `frontend/src/App.tsx:40-43`
**Issue:** `WELCOME_TAB`, `DAEMON_MANAGER_TAB`, `REMOTE_SESSIONS_TAB`, and `SETTINGS_TAB` are declared inside the `App` function body, meaning new object references are created on every render. These objects are compared by reference in `tabs.find()` calls within `useCallback` hooks (lines 265, 401, 416, 426) and used in `useEffect` dependency comparisons (lines 355, 377). While not a crash bug today, any future change that adds these to a dependency array will cause infinite re-render loops. The existing `handleOpenSettings`, `handleOpenDaemonManager`, `handleOpenRemoteSessions`, and `handleHome` callbacks already reference these per-render objects, making them unstable.
**Fix:** Move the tab constant definitions outside the component function:
```tsx
const WELCOME_TAB: Tab = { id: '__welcome__', name: 'Welcome', sessionId: '', cli: '', type: 'welcome' }
const DAEMON_MANAGER_TAB: Tab = { id: '__daemon_manager__', name: 'Sessions', sessionId: '', cli: '', type: 'daemon-manager' }
// ... etc.

function App(): React.ReactElement {
  // use WELCOME_TAB etc. directly
}
```

### WR-04: Missing `tabs` in dependency array of handleOpenSettings causes stale closure

**File:** `frontend/src/App.tsx:264-272`
**Issue:** `handleOpenSettings` depends on `tabs` (via `tabs.find()`) and correctly lists `[tabs]` as a dependency. However, `handleAddTab` (line 274) depends on `handleOpenSettings` and `detectedCLIs`, and `handleOpenSettings` depends on `tabs`. This creates a correct transitive dependency chain. The concern is actually with `handleOpenDaemonManager` (line 399), `handleOpenRemoteSessions` (line 415), and `handleHome` (line 425) -- all three list `[tabs]` in their dependency array but reference the per-render `DAEMON_MANAGER_TAB`/`REMOTE_SESSIONS_TAB`/`WELCOME_TAB` constants. If `tabs` does not change but those constants are new objects each render, the `setTabs((prev) => [...prev, CONSTANT])` call will push a new-identity object each time. This is masked because `tabs` in the dependency array does trigger re-creation, but it means every `tabs` state change regenerates all four callbacks.
**Fix:** See WR-03 fix -- hoisting the constants eliminates this problem.

### WR-05: `webEnabled` state not cleared for sessions that were enabled before frontend disconnect

**File:** `frontend/src/App.tsx:155-175` and `frontend/src/App.tsx:477-498`
**Issue:** When restoring sessions on mount (line 155) or retry (line 477), the code seeds `webEnabled` from `s.webEnabled` but only sets state if `Object.keys(enabledMap).length > 0`. It does NOT clear previously-set `webEnabled` entries for sessions that are no longer web-enabled on the daemon side. After a `retryInit`, if a session was previously web-enabled but is now disabled on the daemon, the frontend `webEnabled` state will retain the stale `true` value because `setWebEnabled(enabledMap)` replaces the entire map only when there are enabled sessions, but leaves the old map untouched when there are none.
**Fix:** Always replace the `webEnabled` and `sessionURLs` state during restore, not conditionally:
```tsx
// Replace the conditional block with:
setWebEnabled(enabledMap)
setSessionURLs(urlMap)
```

## Info

### IN-01: console.error / console.warn calls left in production code

**File:** `frontend/src/App.tsx:178,260,293,317,341,364,500`
**Issue:** Seven `console.error` / `console.warn` calls remain in the component. These are useful during development but will emit to the WebView console in production builds.
**Fix:** Consider using a conditional logger or removing these before release builds. Low priority since they do not affect functionality.

### IN-02: http.Client created per fetchRemoteSessions call without connection pooling

**File:** `app.go:509-512`
**Issue:** A new `http.Client` with a new `http.Transport` is created for every call to `fetchRemoteSessions`. Since `GetRemoteSessions` can call this up to 5 times concurrently, each call creates its own TLS connection pool. This is wasteful but not a correctness issue.
**Fix:** Consider creating a package-level `http.Client` with the desired TLS config and reusing it across calls.

### IN-03: Magic numbers for PTY dimension estimation

**File:** `frontend/src/App.tsx:231-236`
**Issue:** The character-width (8px) and line-height (17px) constants used to estimate initial PTY dimensions are magic numbers. The comment notes these are approximations overridden by a later fit, so this is informational only.
**Fix:** Extract to named constants for clarity:
```tsx
const CHAR_WIDTH_PX = 8
const LINE_HEIGHT_PX = 17
const STATUS_BAR_HEIGHT_PX = 32
```

---

_Reviewed: 2026-04-10T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
