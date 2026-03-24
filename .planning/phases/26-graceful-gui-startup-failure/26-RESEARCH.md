# Phase 26: Graceful GUI Startup Failure - Research

**Researched:** 2026-03-24
**Domain:** Wails v2 startup lifecycle, Go error propagation, React event-driven error UI
**Confidence:** HIGH

---

## Summary

Phase 26 closes the final gap in DAEMON-05: when the daemon binary is missing or fails to start within 3 seconds, the current `startup()` in `app.go` calls `panic()`. Because Wails invokes `OnStartup` in a goroutine, this panic propagates to the goroutine root and crashes the entire process before any UI renders.

The fix has two coupled halves. On the Go side, `startup()` must convert the panic into a graceful return — storing the error and emitting a `daemon:error` Wails event. On the frontend side, a subscriber for `daemon:error` must set `daemonError` state so the already-coded error banner renders. A new `RetryDaemon()` Wails-bound method enables the existing "Retry Connection" button to actually attempt daemon re-spawn on the Go side, rather than just re-calling daemon RPC methods against a nil client.

**Primary recommendation:** Replace `panic()` with `runtime.EventsEmit(ctx, "daemon:error", message)` and add `RetryDaemon() error` as a Wails-bound method. Wire the frontend retry button to call `RetryDaemon()` before re-running `init()`.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DAEMON-05 | Daemon auto-starts when any CLI command is run and no daemon is running | The EnsureDaemon path must not panic the GUI on failure; graceful error + retry covers the GUI startup flow of this requirement |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/wailsapp/wails/v2` | v2.10.2 (project) | Desktop app framework with Go/JS bridge | Already in use; provides EventsEmit for Go→JS messaging |
| `github.com/wailsapp/wails/v2/pkg/runtime` | same | `runtime.EventsEmit` and `runtime.EventsOn` for Go→frontend events | Canonical Wails event bus |
| `vitest` | ^4.1.0 | Frontend unit tests | Already configured; existing `App.test.tsx` uses raw source scanning |
| `go test ./...` | go 1.21+ | Go unit tests | Existing `app_test.go` pattern; `testApp()` helper available |

### No New Dependencies
This phase requires zero new imports. All primitives already exist:
- `runtime.EventsEmit` (already imported in `app.go` via `"github.com/wailsapp/wails/v2/pkg/runtime"`)
- `EventsOn` (already imported in `App.tsx` from `./wailsjs/wailsjs/runtime/runtime`)
- `useState`/`useCallback` (already used in `App.tsx`)

---

## Architecture Patterns

### Wails OnStartup Execution Model

**CRITICAL FINDING (verified from Wails v2.11.0 source):**

On all three desktop platforms (darwin, windows, linux), `OnStartup` is invoked **inside a goroutine** after the window is shown:

```go
// darwin/frontend.go line 248-252
go func() {
    if f.frontendOptions.OnStartup != nil {
        f.frontendOptions.OnStartup(f.ctx)
    }
}()
mainWindow.Run(f.startURL.String())
```

This means:
1. The frontend IS running and can receive events before startup completes
2. A `panic()` inside `OnStartup` kills the goroutine and crashes the process (the window closes)
3. A graceful return from `OnStartup` leaves the window open with whatever state was emitted

### Wails `OnStartup` Signature Constraint

The `OnStartup` type is `func(ctx context.Context)` — it returns nothing. There is no way to communicate startup failure back to Wails through the return value. The only channel available is:
- `runtime.EventsEmit(ctx, "daemon:error", message)` — sends event to frontend JS
- Store error on `App` struct for later `RetryDaemon()` to check

### Recommended Pattern: Emit-and-Return

```go
// app.go — startup()
func (a *App) startup(ctx context.Context) {
    a.ctx = ctx

    socketPath := daemon.DefaultSocketPath()
    if err := daemon.EnsureDaemon(socketPath); err != nil {
        // Store for RetryDaemon to know we are in failed state
        a.daemonErr = err
        // Notify frontend — window is already rendered at this point
        runtime.EventsEmit(ctx, "daemon:error", err.Error())
        // Return gracefully — do NOT call initTray or startHealthPoller
        return
    }
    a.client = daemon.NewDaemonClient(socketPath)
    a.initTray()
    a.trayInit = true
    a.startHealthPoller(ctx)
}
```

### RetryDaemon Bound Method

The frontend "Retry Connection" button currently calls `retryInit` which calls `GetRelayPort()`, `ListSessions()`, etc. — all of which call `a.client.XXX()` against a **nil client** if startup failed. This will panic with a nil pointer dereference.

A `RetryDaemon()` method must be added that re-runs EnsureDaemon and re-initialises `a.client`:

```go
// app.go — RetryDaemon() error (Wails-bound)
func (a *App) RetryDaemon() error {
    socketPath := daemon.DefaultSocketPath()
    if err := daemon.EnsureDaemon(socketPath); err != nil {
        a.daemonErr = err
        return err
    }
    a.daemonErr = nil
    a.client = daemon.NewDaemonClient(socketPath)
    // Also start tray and health poller if not already started
    if !a.trayInit {
        a.initTray()
        a.trayInit = true
    }
    if a.ctx != nil {
        a.startHealthPoller(a.ctx)
    }
    return nil
}
```

### Frontend: Subscribe to daemon:error Event

The frontend already has `daemonError` state (line 60 of `App.tsx`) and already renders the error banner (lines 297-333). The missing piece is:

1. Subscribe to `daemon:error` on mount to catch the Go startup failure path
2. Wire the "Retry Connection" button to call `RetryDaemon()` first, then re-run `init()`

```typescript
// In useEffect mount block, alongside offStatus/offHealth:
const offDaemonError = EventsOn('daemon:error', (msg: string) => {
  setDaemonError(msg)
})
// In cleanup return:
offDaemonError()
```

```typescript
// retryInit — prepend RetryDaemon call:
const retryInit = useCallback(async () => {
  setDaemonError(null)
  try {
    await RetryDaemon()  // NEW: re-spawn daemon on Go side
  } catch (err) {
    setDaemonError(String(err))
    return
  }
  // ... existing Promise.all([GetRelayPort(), ...]) ...
}, [])
```

### App Struct Change: daemonErr field

Add `daemonErr error` field to `App` struct to track startup failure state:

```go
type App struct {
    ctx      context.Context
    client   *daemon.DaemonClient
    trayInit bool
    daemonErr error  // non-nil when EnsureDaemon failed at startup
}
```

### Nil-Safety Guard (Optional Defense)

All Wails-bound methods that call `a.client.XXX()` will nil-panic if called before startup succeeds. `RetryDaemon()` prevents this in the normal retry flow, but a nil-guard in `ListSessions()` and `GetRelayPort()` is a defensive option:

```go
func (a *App) ListSessions() []SessionInfo {
    if a.client == nil {
        return []SessionInfo{}
    }
    // ... existing ...
}
```

This matches the existing pattern where `ListSessions` already returns `[]SessionInfo{}` on error — the failure mode is the same.

### Recommended Project Structure (No Change)

No new files needed. All changes are in:
- `app.go` — modify `startup()`, add `RetryDaemon()`, add `daemonErr` field, optionally add nil guards
- `frontend/src/App.tsx` — subscribe to `daemon:error`, wire retry button

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Go→Frontend error notification | Custom IPC channel | `runtime.EventsEmit(ctx, "daemon:error", msg)` | Wails event bus is the correct channel for Go→JS async messages |
| JS→Go retry call | localStorage polling | Wails-bound `RetryDaemon() error` method | Synchronous RPC pattern matches all other App methods |
| Frontend error display | New error component | Extend existing `daemonError` state + banner in App.tsx | Banner is already coded at lines 297-333; reuse it |

---

## Common Pitfalls

### Pitfall 1: Nil Client on Retry
**What goes wrong:** Frontend calls `GetRelayPort()` before `RetryDaemon()` succeeds; `a.client` is still nil; method panics.
**Why it happens:** `init()` in App.tsx calls multiple Go methods in `Promise.all`; if the frontend retries without calling `RetryDaemon()` first, all calls crash.
**How to avoid:** `retryInit` MUST call `RetryDaemon()` and await it before calling any other bound method. Early return on `RetryDaemon` error.
**Warning signs:** "null pointer dereference" panic logs in stderr when retry is clicked.

### Pitfall 2: Race Between EventsEmit and EventsOn Subscription
**What goes wrong:** Startup fires `daemon:error` event before the frontend's `useEffect` subscribes to it; event is lost; UI shows blank screen with no error message.
**Why it happens:** Wails `OnStartup` runs in a goroutine; on fast machines, it may fail and emit before the React tree has mounted and registered the listener.
**How to avoid:** Two-pronged defense:
  1. Also check for daemon error in `init()` by catching exceptions from `GetRelayPort()` (already done via `setDaemonError(String(err))` in the existing catch block — `GetRelayPort()` will error if `a.client == nil`)
  2. Alternatively, add a `GetDaemonError() string` bound method that `init()` calls to poll for startup error state
**Recommended:** Guard in `init()` already covers this via the catch block. The `daemon:error` event is a bonus for real-time notification if startup is slow.

### Pitfall 3: startHealthPoller Called After Startup Failure
**What goes wrong:** If `startHealthPoller` runs with a nil client, the goroutine starts background polling but all calls to `a.client.XXX()` will panic.
**Why it happens:** Forgot to return early from startup before the poller is started.
**How to avoid:** In the modified `startup()`, the `return` after emitting the error event MUST come before `a.initTray()` and `a.startHealthPoller()`.

### Pitfall 4: Double Health Poller on Retry
**What goes wrong:** `RetryDaemon()` calls `startHealthPoller()` again; now two pollers are running.
**Why it happens:** `startHealthPoller` starts a goroutine; calling it twice starts two goroutines.
**How to avoid:** Guard with a started flag, or pass the context and let both exit when ctx is cancelled (the duplicate just wastes resources but doesn't corrupt state). Simplest: check `a.trayInit` as a proxy since both are set together.

### Pitfall 5: shutdown() nil-panics on Cleanup
**What goes wrong:** If startup fails and the user closes the window, `shutdown()` runs; `a.trayInit` is false so tray cleanup is skipped (correct); but other cleanup code might assume `a.client != nil`.
**Why it happens:** `shutdown()` only guards `a.trayInit`; no guard for client.
**How to avoid:** `shutdown()` currently does nothing except tray cleanup and already has the `a.trayInit` guard. No additional change needed.

---

## Code Examples

### Verified Pattern: Existing EventsEmit Usage (app.go)

```go
// Source: app.go line 114-117 — EventsEmit in pollSessionStatus
runtime.EventsEmit(a.ctx, "session:status", map[string]string{
    "sessionId": sessionID,
    "status":    s,
})
```

Emitting a plain string for `daemon:error` follows the same pattern:
```go
runtime.EventsEmit(ctx, "daemon:error", err.Error())
```

### Verified Pattern: EventsOn in Frontend (App.tsx)

```typescript
// Source: App.tsx line 108-113 — existing EventsOn usage
const offStatus = EventsOn(
  'session:status',
  (data: { sessionId: string; status: string }) => {
    setSessionStatuses((prev) => ({ ...prev, [data.sessionId]: data.status }))
  },
)
```

New listener follows same pattern:
```typescript
const offDaemonError = EventsOn('daemon:error', (msg: string) => {
  setDaemonError(msg)
})
```

### Verified Pattern: Wails Bound Method Returning Error

All existing App methods that can fail return `error`. `RetryDaemon()` follows:
```go
func (a *App) RetryDaemon() error {
    // ...
    return err  // nil on success
}
```

The Wails code-gen produces TypeScript as `RetryDaemon(): Promise<void>` when error is the only return value (non-nil error becomes a rejected Promise).

### Verified Pattern: Existing Error Banner (App.tsx lines 297-333)

```tsx
{daemonError && tabs.length === 0 && (
  <div style={{ ... borderLeft: '3px solid #f7768e', ... }}>
    <div>Unable to connect to session daemon</div>
    <div>{/* error description */}</div>
    <button onClick={retryInit}>Retry Connection</button>
  </div>
)}
```

The banner is already complete and styled. No new component needed. The message text should reference the actual `daemonError` string (currently shows a hardcoded message). Consider displaying `daemonError` directly or alongside the static text.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `panic()` on EnsureDaemon fail | Emit event + return gracefully | Phase 26 | GUI stays alive; user can retry |
| Frontend only catches `init()` failures | Also subscribes to `daemon:error` event | Phase 26 | Catches failures before init() runs |
| `retryInit` calls Go methods directly | Calls `RetryDaemon()` first, then Go methods | Phase 26 | Prevents nil client panic on retry |

---

## Open Questions

1. **Should `GetDaemonError() string` be added as a bound method?**
   - What we know: The race condition between `EventsEmit` and `EventsOn` subscription could cause the event to be missed on fast failures
   - What's unclear: Whether Wails startup goroutine can actually outrun the React mount cycle in practice
   - Recommendation: Add `GetDaemonError() string` as a defensive poll in `init()` alongside the event subscription; this costs 1 extra RPC on startup but eliminates the race entirely

2. **Display raw error string or user-friendly message in the banner?**
   - What we know: `daemonError` is currently displayed only as a static "The background daemon did not start in time" message; the actual error is logged to console
   - What's unclear: User preference for technical vs friendly messages
   - Recommendation: Show the `daemonError` string as a detail below the static heading (same pattern as browser dev tool errors)

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test` (stdlib) |
| Framework (Frontend) | vitest ^4.1.0 |
| Config file (Go) | none — `go test ./...` |
| Config file (Frontend) | `frontend/vitest.config.ts` or inline in `vite.config.ts` |
| Quick run command (Go) | `cd /path/to/agenthub && go test -run TestStartup -v ./...` |
| Quick run command (Frontend) | `cd /path/to/agenthub/frontend && pnpm test` |
| Full suite (Go) | `go test ./...` |
| Full suite (Frontend) | `pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DAEMON-05 | `startup()` returns instead of panicking when EnsureDaemon fails | unit | `go test -run TestStartupEnsureDaemonFailure -v .` | ❌ Wave 0 |
| DAEMON-05 | `startup()` emits `daemon:error` event on failure | unit | `go test -run TestStartupEmitsDaemonError -v .` | ❌ Wave 0 |
| DAEMON-05 | `RetryDaemon()` succeeds when daemon is available | unit | `go test -run TestRetryDaemon -v .` | ❌ Wave 0 |
| DAEMON-05 | `RetryDaemon()` returns error when daemon unavailable | unit | `go test -run TestRetryDaemonFail -v .` | ❌ Wave 0 |
| DAEMON-05 | Frontend renders error banner on daemonError state | unit (raw scan) | `cd frontend && pnpm test` | ❌ Wave 0 |
| DAEMON-05 | Frontend retry calls RetryDaemon before init methods | unit (raw scan) | `cd frontend && pnpm test` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -run TestStartup -v . && cd frontend && pnpm test`
- **Per wave merge:** `go test ./... && cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `app_test.go` — add `TestStartupEnsureDaemonFailure`, `TestStartupEmitsDaemonError`, `TestRetryDaemon`, `TestRetryDaemonFail`
- [ ] `frontend/src/components/__tests__/App.test.tsx` — add tests for `daemon:error` subscription, retry wiring, `RetryDaemon` import

*(No new framework install needed — both Go test and vitest are already in place)*

---

## Sources

### Primary (HIGH confidence)
- Wails v2.11.0 source — `internal/frontend/desktop/darwin/frontend.go` lines 248-254: `OnStartup` runs inside `go func()`; window is already running when startup executes
- Wails v2.11.0 source — `pkg/options/options.go` line 61: `OnStartup func(ctx context.Context)` — returns nothing; no error channel back to Wails
- Project source — `app.go` lines 43-58: current `startup()` with the `panic()`
- Project source — `app.go` line 16: `runtime` package already imported
- Project source — `frontend/src/App.tsx` lines 60, 100-103, 246-281, 297-333: `daemonError` state, init catch block, `retryInit`, error banner — all already present

### Secondary (MEDIUM confidence)
- Wails v2.11.0 source — `internal/frontend/desktop/windows/frontend.go` line 223, `internal/frontend/desktop/linux/frontend.go` line 264: same pattern on Windows/Linux — startup runs in goroutine on all platforms

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; all existing
- Architecture: HIGH — verified from Wails source and existing project patterns
- Pitfalls: HIGH — derived directly from reading call sites and nil checks
- Frontend changes: HIGH — `daemonError` state already exists; verified banner code

**Research date:** 2026-03-24
**Valid until:** 2026-04-24 (stable domain; Wails version pinned in go.mod)
