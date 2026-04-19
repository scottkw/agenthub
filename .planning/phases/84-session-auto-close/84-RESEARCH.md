# Phase 84: Session Auto-Close - Research

**Researched:** 2026-04-19
**Domain:** Session lifecycle — PTY exit detection, Wails event emission, React countdown + toast UI
**Confidence:** HIGH

## Summary

Phase 84 adds automatic session tab closure when an agent process exits naturally (exit code 0). The implementation spans three layers: (1) the Go relay hub already signals PTY EOF via `hub.done`, which is the correct trigger point; (2) `app.go` needs a new `session:exit` Wails event emitted after PTY EOF is detected; (3) the React frontend needs countdown state, an inline terminal banner with "Keep Open", a visible tab indicator, and a global toast notification.

The architectural risk is low — the exit signal infrastructure already exists (`hub.Done()` channel closes when PTY Read returns EOF). The primary new work is wiring that signal through the app layer and building the UI components. The `status.Watch()` goroutine already monitors `hub.Done()` and returns when the channel closes, providing an exact hook for exit detection without polling.

Non-zero exit codes must skip auto-close (D-09). The settings toggle (D-11) requires a new boolean persisted in `settings.json` via the existing `daemonSettings` struct. Web viewer grace period (D-12) is a new concern but low-complexity.

**Primary recommendation:** Detect exit in `status.Watch()` (or a sibling goroutine) by observing `hub.Done()` closure, emit `session:exit` Wails event from `app.go`, and build the countdown + toast UI in React — reusing `handleCloseTab` for the actual close operation.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 5-second countdown before auto-closing the tab after process exit
- **D-02:** Countdown is cancellable — user can click "Keep Open" to prevent auto-close
- **D-03:** Tab fades out (opacity/dim animation) before removal when countdown completes
- **D-04:** Countdown appears in both the tab (visual indicator) and an inline banner inside the terminal area with a "Keep Open" action button
- **D-05:** Toast popup + inline terminal banner when agent exits — toast is visible from any tab, banner provides context in the exiting tab
- **D-06:** Notification includes: session name, agent type, exit code, session duration, and final heuristic status (running/idle/waiting/errored)
- **D-07:** Use PTY EOF detection as the signal that all output has been delivered — countdown starts only after EOF
- **D-08:** Terminal remains fully interactive (scrollable, selectable, copyable) during the countdown period
- **D-09:** Non-zero exit code skips auto-close entirely — tab stays open so user can review error output
- **D-10:** Clean exits (exit code 0) trigger the normal countdown + auto-close flow
- **D-11:** Global toggle in Settings to disable auto-close (default: enabled)
- **D-12:** GUI tab closes normally regardless of web viewers. Web serving continues briefly with an "Agent exited" message, then stops after a grace period.

### Claude's Discretion
- Exit detection mechanism details (how backend detects natural process exit and propagates it)
- Toast component implementation (new component vs. extending existing banner patterns)
- Fade-out animation duration and CSS approach
- Web serving grace period duration
- Settings toggle placement within the Settings tab

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SESS-01 | Session tab automatically closes when the agent process exits | Exit detection via hub.Done(), `session:exit` event, frontend countdown + auto-close |
| SESS-02 | Brief delay before auto-close allows final output to flush to terminal | PTY EOF = output complete (D-07); 5-second countdown satisfies flush window |
| SESS-03 | Toast or indicator notifies user that the agent exited before the tab closes | New Toast component + inline terminal banner; visible across all tabs |
</phase_requirements>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| PTY exit detection | Backend (relay/hub) | Backend (engine) | hub.Done() closes when PTY Read returns EOF — only the relay layer has this signal |
| Exit event propagation | Backend (app.go) | — | app.go owns Wails EventsEmit; must translate hub.Done() into a `session:exit` Wails event |
| Exit code capture | Backend (pty/backend) | daemon/engine | Process exit code lives in cmd.Wait() result; must be surfaced alongside the event |
| Auto-close countdown | Frontend (App.tsx) | — | Tab state and timers live in React; countdown is pure frontend state |
| Toast notification | Frontend (new component) | — | Toast must be visible across tabs — owned at App level, not inside TerminalPanel |
| Inline terminal banner | Frontend (TerminalPanel or sibling) | — | Shown inside the exiting session's content area |
| Tab fade animation | Frontend (TabBar) | CSS | CSS opacity transition + class toggling; no JS animation library needed |
| Settings toggle (auto-close) | Backend (daemon/settings) + Frontend (SettingsTab) | — | Persisted in settings.json; read at event handling time in frontend |
| Web viewer grace period | Backend (daemon/webserver) | — | Web server must delay disabling the session page; purely backend concern |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.2.4 | Component/state management | Already used — all UI is React |
| Wails v2 runtime | 2.x | Go-to-JS event bus | `runtime.EventsEmit` / `EventsOn` — established pattern in this codebase |
| xterm.js | @xterm/xterm 6.x | Terminal rendering | Already in use; terminal remains interactive during countdown (D-08) |
| Vitest | 4.1.0 | Frontend tests | Existing test runner; tests use createRoot + flushSync pattern |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| @heroicons/react | 2.x | Icon for toast dismiss / Keep Open | Already used in LocalNetworkBanner for XMarkIcon |
| CSS transitions | browser native | Tab fade-out, banner enter/exit | Use existing `.banner-exit` pattern (opacity + max-height) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom toast component | react-hot-toast / react-toastify | No external deps needed — project has simple banner patterns that can be extended |
| setInterval countdown | requestAnimationFrame | setInterval is simpler; precision is not critical for 5s UI countdown |

**Installation:** No new packages required.

## Architecture Patterns

### System Architecture Diagram

```
PTY Process exits
       |
       v
hub.Run() Read() -> io.EOF
       |
       v (defer hub.Shutdown() -> close(hub.done))
hub.done channel CLOSED
       |
       +--> status.Watch() goroutine returns (already wired)
       |
       +--> NEW: exitWatcher goroutine (in engine.CreateSession or app.go)
                  |
                  v
            cmd.Wait() -> captures exit code
                  |
                  v
            engine.onExit callback(sessionID, exitCode) called
                  |
                  v (crosses process boundary: daemon -> GUI via HTTP poll OR Wails direct)
            app.go: goroutine polls for exit OR engine callback -> emit Wails event
                  |
                  v
            runtime.EventsEmit(ctx, "session:exit", payload)
                  |
                  v
            Frontend: EventsOn("session:exit") in App.tsx useEffect
                  |
                  +--> exitCode == 0 AND autoCloseEnabled?
                  |         |
                  |         v YES
                  |    setExitCountdowns (sessionId -> countdown state)
                  |    setExitToast (show toast notification)
                  |    setTabExiting (sessionId -> true for fade CSS)
                  |         |
                  |         v (5-second setInterval ticks)
                  |    countdown reaches 0 -> handleCloseTab(sessionId)
                  |
                  +--> exitCode != 0 OR autoCloseDisabled
                            |
                            v
                       setExitToast (show toast, no countdown)
                       tab stays open
```

### Recommended Project Structure

No new directories needed. New files:
```
frontend/src/
├── components/
│   ├── ExitToast.tsx            # New: global toast notification
│   ├── ExitCountdownBanner.tsx  # New: inline terminal banner with "Keep Open"
│   └── __tests__/
│       ├── ExitToast.test.tsx
│       └── ExitCountdownBanner.test.tsx
```

Backend additions (no new files, extend existing):
```
internal/
├── daemon/
│   └── engine.go               # Add onExit callback support to CreateSession
│   └── types.go                # Add ExitEventPayload, update daemonSettings
│   └── api.go                  # No change — exit detected by engine, not HTTP
├── pty/
│   └── backend.go or native.go # Capture exit code from cmd.Wait()
app.go                          # Subscribe to onExit, emit session:exit Wails event
```

### Pattern 1: Hub Done -> Exit Detection

**What:** The relay hub's `done` channel closes when the PTY Read loop returns an error (EOF on natural process exit). This is already used by `status.Watch()`.

**When to use:** As the PTY-EOF signal (D-07) — this channel closes only after the last byte has been read from the PTY buffer, satisfying the output-flush guarantee.

```go
// Source: internal/relay/hub.go lines 135-152
// Hub.Run() calls defer h.Shutdown() which closes h.done.
// The exit watcher pattern:
go func() {
    <-hub.Done() // blocks until PTY EOF (all output delivered)
    // Now safe to emit exit event — no output in flight
    exitCode := captureExitCode(sess)
    if onExit != nil {
        onExit(sessionID, exitCode)
    }
}()
```

**Key insight:** `hub.Done()` is the correct EOF signal. Do NOT use `cmd.Wait()` alone — the PTY master must drain first, and `hub.Run()` does that draining. After `hub.done` closes, all output has been fanned to subscribers.

### Pattern 2: Exit Code Capture

**What:** The exit code lives in `cmd.Wait()` result. On POSIX, `cmd.Process.Wait()` (via `cmd.Wait()`) returns an `*exec.ExitError` with `ExitCode()`. The hub shutdown happens in `hub.Run()` defer AFTER the Read error — but `cmd.Wait()` should complete naturally because the process has already exited (PTY closed causes the process to exit, which causes Wait to return).

**When to use:** Called in the exit watcher goroutine, after `hub.Done()` fires.

```go
// Source: ASSUMED pattern — to be implemented in engine.go
func (e *SessionEngine) watchSessionExit(hub *relay.Hub, sess *pty.Session, sessionID string, onExit func(string, int)) {
    go func() {
        <-hub.Done() // PTY EOF — all output delivered
        // cmd.Wait() should have already returned (process exited before PTY EOF),
        // but we do a non-blocking check via sess.State.
        exitCode := 0
        sess.mu.Lock()
        cmd := sess.cmd
        sess.mu.Unlock()
        if cmd != nil && cmd.ProcessState != nil {
            exitCode = cmd.ProcessState.ExitCode()
        }
        if onExit != nil {
            onExit(sessionID, exitCode)
        }
    }()
}
```

**Note:** `Session.cmd` is currently unexported in `session.go`. The exit watcher will need either (a) an exported `ExitCode()` method on Session, or (b) the watcher placed in the same package (`pty`).

### Pattern 3: Wails Event Emission (session:exit)

**What:** Follows the exact shape of `session:status` events emitted from `app.go`.

**When to use:** In `app.go`, subscribe to engine's `onExit` callback, emit Wails event.

```go
// Source: app.go lines 213-218 (adapted)
runtime.EventsEmit(a.ctx, "session:exit", map[string]any{
    "sessionId":    sessionID,
    "exitCode":     exitCode,
    "sessionName":  name,
    "cli":          cli,
    "duration":     durationSeconds,
    "finalStatus":  lastKnownStatus,
})
```

**Important:** The GUI (app.go) operates against the daemon via HTTP client. The `onExit` callback cannot be passed through the HTTP API directly. The recommended approach:

**Option A (recommended):** Poll `ListSessions()` in `app.go` and detect when a session transitions from running to absent (State: "stopped" or missing from list). This is fully compatible with the daemon architecture — no new daemon API needed.

**Option B:** Add a new long-poll or SSE endpoint to the daemon API for exit events. More complex, not warranted for this use case.

**Option C:** Extend `pollSessionStatus` in `app.go` (currently polls for 60s) to detect "stopped" state and emit the exit event. This is the lowest-friction approach given the existing code.

**Recommended: Option C** — extend `pollSessionStatus` to detect `State: "stopped"` in `ListSessions()` response and emit `session:exit`.

### Pattern 4: Frontend Countdown State

**What:** Per-session countdown state in `App.tsx`, using `useRef` for the interval to avoid stale closures.

**When to use:** When `session:exit` event arrives with `exitCode == 0` and auto-close is enabled.

```typescript
// Source: ASSUMED — standard React countdown pattern
interface ExitState {
  sessionId: string
  sessionName: string
  cli: string
  exitCode: number
  duration: number
  finalStatus: string
  countdown: number          // seconds remaining (5 -> 0)
  cancelled: boolean         // true = user clicked "Keep Open"
}

// In App.tsx state:
const [sessionExits, setSessionExits] = useState<Record<string, ExitState>>({})
const countdownTimers = useRef<Record<string, ReturnType<typeof setInterval>>>({})
```

### Pattern 5: Toast Notification Component

**What:** A floating notification visible regardless of active tab. New `ExitToast` component, mounted at App level above all content.

**When to use:** Always on exit — for both clean exits (with countdown) and error exits (tab stays open). Multiple toasts may stack.

**Design follows `UpdateBanner` pattern** — simple div with flex layout, dismiss button (`XMarkIcon`), message, and action button.

```typescript
// CSS class pattern (follows .update-banner):
// .exit-toast                — fixed position overlay container
// .exit-toast__item          — individual toast card
// .exit-toast__item--error   — red accent for non-zero exit
// .exit-toast__item--clean   — green/neutral accent for exit code 0
// .exit-toast__countdown     — countdown badge ("3s")
// .exit-toast__keep-open     — "Keep Open" button
// .exit-toast__dismiss       — X dismiss button
```

**Position recommendation:** Fixed, bottom-right, stacking upward. Uses CSS `position: fixed; bottom: 16px; right: 16px`.

### Pattern 6: Inline Terminal Banner (ExitCountdownBanner)

**What:** A banner rendered inside the terminal panel of the exiting session, below the terminal output and above the status bar. Follows `LocalNetworkBanner` / `UpdateBanner` pattern.

**When to use:** Only visible when the session has a pending countdown (exitCode == 0 and countdown active).

```typescript
interface ExitCountdownBannerProps {
  sessionName: string
  cli: string
  exitCode: number
  finalStatus: string
  duration: number
  countdown: number    // seconds remaining
  onKeepOpen: () => void
}
```

**Placement:** TerminalPanel needs to accept an `exitState` prop and render the banner between the xterm container and the status bar.

### Pattern 7: Settings Toggle

**What:** Boolean `autoCloseEnabled` field in `daemonSettings` (persisted to `settings.json`). Frontend reads it via a new `GetAutoClose` / `SetAutoClose` Wails method pair. Displayed in the existing Settings tab.

**When to use:** Added to `daemonSettings` struct in `engine.go`. Default: `true`.

```go
// Source: ASSUMED — extend daemonSettings in engine.go
type daemonSettings struct {
    CLIPaths        map[string]string `json:"cliPaths,omitempty"`
    StartMinimized  bool              `json:"startMinimized,omitempty"`
    AutoCloseSession bool             `json:"autoCloseSession,omitempty"` // default true when absent
}
```

**Frontend:** The setting is read once on mount in App.tsx and stored as `autoCloseEnabled` boolean. When `false`, `session:exit` still fires the toast but skips the countdown/auto-close.

**Note on default:** JSON omitempty means an absent field deserializes to `false`. Since default should be `true`, the getter must return `true` when the field is absent (treat missing == enabled).

### Anti-Patterns to Avoid

- **Relying on cmd.Wait() alone for exit detection:** cmd.Wait() returns when the process exits, but PTY output buffered in the kernel may not have been drained. Always use `hub.Done()` as the flush signal (D-07).
- **Calling KillSession on natural exit:** `handleCloseTab` calls `KillSession` which sends SIGHUP. For natural exits, the process is already gone — `KillSession` will fail. The auto-close flow should call a lighter "remove session" that only removes the daemon registry entry without signaling the process.
- **Starting countdown before `session:exit` event:** Don't start the timer on `session:status == errored` — that status fires during the polling loop and doesn't indicate natural PTY exit.
- **Storing countdown in ref instead of state:** The countdown number must be in React state to trigger re-renders (visual countdown update). The `setInterval` handle goes in a ref.
- **Global state for settings toggle in frontend:** Don't add `autoCloseEnabled` to `daemonSettings` and read it from the daemon on every `session:exit` event. Read once on mount, store in React state.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CSS countdown animation | Custom canvas/SVG progress ring | CSS transition on width or opacity | Simpler, matches project's existing animation approach |
| Toast stacking logic | Custom z-index / position manager | CSS flexbox column with fixed positioning | 1-3 toasts max; no library needed |
| PTY EOF detection | Custom process polling | `hub.Done()` channel | Already implemented — closes exactly when last byte is read |
| Settings persistence | Custom JSON file | Extend existing `daemonSettings` struct | Saves/loads already wired in `engine.go` |

**Key insight:** The exit signal infrastructure (hub.Done) is already built. The main work is UI plumbing, not new backend infrastructure.

## Common Pitfalls

### Pitfall 1: KillSession Called on Already-Exited Process
**What goes wrong:** `handleCloseTab` calls `KillSession` which sends SIGHUP to the process group. If the process has already exited, this returns an error (no such process). The error is currently caught with `console.warn` but the session registry may be in an inconsistent state.
**Why it happens:** The auto-close flow reuses `handleCloseTab` which was designed for manual close (where the process is still alive).
**How to avoid:** Either (a) `KillSession` in the daemon must gracefully handle "already exited" (return nil if process not running), or (b) the auto-close path calls a new `RemoveSession` that only removes registry entries without process signaling.
**Warning signs:** "kill session" errors in console during auto-close.

### Pitfall 2: Exit Code Unavailable After PTY Close
**What goes wrong:** On some platforms, `cmd.ProcessState` may be nil if `cmd.Wait()` was not called, or if the process exited before the PTY read loop caught the EOF.
**Why it happens:** `hub.Run()` calls `Read()` on the PTY — when the PTY master reads EOF, the process has already exited and `Wait()` has been called by the cleanup logic. However, `Session.cmd.ProcessState` is only populated after `cmd.Wait()` returns.
**How to avoid:** In the exit watcher goroutine, wait until `hub.Done()` fires (guaranteeing `Run()` has returned), then access `cmd.ProcessState.ExitCode()`. If `ProcessState` is nil, default to exit code 0 (conservative — don't block auto-close on a nil state).
**Warning signs:** Exit code always showing as 0 even for error exits.

### Pitfall 3: Race Between Tab Close and Auto-Close
**What goes wrong:** User manually closes a tab while the auto-close countdown is running. Both `handleCloseTab` (manual) and the countdown timer fire, leading to double-close.
**Why it happens:** Timer fires after manual close; session ID no longer in tabs list.
**How to avoid:** In the countdown timer callback, check if the session tab still exists before calling `handleCloseTab`. Also clear the timer in `handleCloseTab` when it fires.
**Warning signs:** "session not found" errors, React state update on unmounted component warnings.

### Pitfall 4: daemonSettings omitempty False Default
**What goes wrong:** `AutoCloseSession bool` with `omitempty` serializes `false` as absent (omitted). On next load, absent field deserializes to `false` (Go zero value). If default should be `true`, this means new users start with auto-close disabled after any save operation that writes `false`.
**Why it happens:** JSON `omitempty` omits zero values, and `false` is the zero value for bool.
**How to avoid:** Use `*bool` (pointer) for `AutoCloseSession` in daemonSettings, or use a non-omitempty tag and always write the field. The getter (`GetAutoCloseSession`) should return `true` when the field is nil/false (absent in settings file = enabled).
**Warning signs:** Settings toggle appears to reset to disabled after restart.

### Pitfall 5: Toast Visible After Tab Is Closed
**What goes wrong:** The exit toast notification persists on screen after the user manually dismisses the tab or after auto-close completes.
**Why it happens:** Toast state is independent of tab state.
**How to avoid:** In `handleCloseTab`, also remove any toast entry for the closed session ID from `sessionExits` state.
**Warning signs:** Ghost toasts referencing sessions that no longer exist.

### Pitfall 6: pollSessionStatus vs. session:exit Event Ordering
**What goes wrong:** `pollSessionStatus` currently stops when status reaches "errored" (line 219 in app.go). For natural exits, the status may transition through "idle" before the process exits — the poller exits early without ever detecting the stopped state.
**Why it happens:** The poller is designed for the status heuristic, not for process lifecycle.
**How to avoid:** The exit detection needs a separate mechanism from status polling. Recommended: in `pollSessionStatus` or a sibling goroutine, also check `SessionInfo.State == "stopped"` from `ListSessions()`. Alternatively, extend the daemon to support a separate `GET /sessions/{id}/exit` long-poll endpoint.
**Warning signs:** `session:exit` events never fire despite agent completing normally.

## Code Examples

### Extending pollSessionStatus to Detect Exit (app.go)

```go
// Source: app.go lines 202-225 (existing pattern, extended)
func (a *App) pollSessionStatus(sessionID string) {
    var last string
    deadline := time.Now().Add(60 * time.Second)
    for time.Now().Before(deadline) {
        // Check session state (running/stopped)
        sessions, err := a.client.ListSessions()
        if err == nil {
            found := false
            for _, s := range sessions {
                if s.ID == sessionID {
                    found = true
                    // Status heuristic update (existing behavior)
                    if s.Status != last {
                        last = s.Status
                        a.emitStatusEvent(sessionID, s.Status)
                    }
                    // Exit detection: daemon marks session as "stopped"
                    if s.State == "stopped" {
                        a.emitExitEvent(sessionID, s)
                        return
                    }
                    break
                }
            }
            if !found {
                // Session removed from daemon registry — already exited and cleaned up
                return
            }
        }
        time.Sleep(500 * time.Millisecond)
    }
}
```

**Note:** This approach requires exit code to be added to `SessionInfo` in the daemon, or a separate `GET /sessions/{id}/exit-code` endpoint. Alternatively, the daemon can set `State: "stopped"` and populate an `ExitCode int` field in `SessionInfo`.

### Frontend: session:exit Event Handler

```typescript
// Source: ASSUMED — follows EventsOn('session:status') pattern in App.tsx line 240
const offExit = EventsOn(
  'session:exit',
  (data: {
    sessionId: string
    exitCode: number
    sessionName: string
    cli: string
    duration: number
    finalStatus: string
  }) => {
    // Always show toast
    setSessionExits(prev => ({
      ...prev,
      [data.sessionId]: {
        ...data,
        countdown: 5,
        cancelled: false,
      }
    }))

    // Only start auto-close for clean exits when enabled
    if (data.exitCode === 0 && autoCloseEnabled) {
      const timer = setInterval(() => {
        setSessionExits(prev => {
          const entry = prev[data.sessionId]
          if (!entry || entry.cancelled) {
            clearInterval(timer)
            return prev
          }
          if (entry.countdown <= 1) {
            clearInterval(timer)
            // Auto-close
            void handleCloseTab(data.sessionId)
            const { [data.sessionId]: _, ...rest } = prev
            return rest
          }
          return { ...prev, [data.sessionId]: { ...entry, countdown: entry.countdown - 1 } }
        })
      }, 1000)
      countdownTimers.current[data.sessionId] = timer
    }
  }
)
```

### CSS: Tab Countdown Indicator

```css
/* Source: ASSUMED — extends existing .tab CSS in style.css */
.tab--exiting {
  opacity: 0.5;
  transition: opacity 150ms ease;
}

.tab__countdown {
  font-size: 11px;
  color: #9ece6a;
  font-variant-numeric: tabular-nums;
  min-width: 18px;
  text-align: center;
}

.exit-toast {
  position: fixed;
  bottom: 16px;
  right: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 9999;
  pointer-events: none;
}

.exit-toast__item {
  pointer-events: all;
  background: #1e2030;
  border: 1px solid #292e42;
  border-radius: 6px;
  padding: 10px 14px;
  min-width: 280px;
  max-width: 380px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: #c0caf5;
  font-size: 13px;
}

.exit-toast__item--clean { border-left: 3px solid #9ece6a; }
.exit-toast__item--error { border-left: 3px solid #f7768e; }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Polling status for exit detection | hub.Done() channel signal | Phase 84 (new) | Exact EOF signal, no polling delay |
| Manual tab-only close | Auto-close with countdown | Phase 84 (new) | Better UX for agent workflows |

**Deprecated/outdated:**
- None — no existing patterns are being replaced, only extended.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `cmd.ProcessState.ExitCode()` is accessible after `hub.Done()` fires | Pattern 2, Pitfall 2 | Exit code may be unavailable; default to 0 (conservative) |
| A2 | `State: "stopped"` appears in `SessionInfo` from `ListSessions()` when process exits naturally | Pattern 3 Option C, Code Examples | If daemon removes session immediately, poll may miss the stopped state; need to add exit code to daemon registry |
| A3 | The daemon keeps a session in the registry with `State: "stopped"` briefly after natural exit (not immediately removing it) | Code Examples | If daemon removes session on process exit, the GUI poll will see "not found" instead of "stopped" + exit code |
| A4 | Settings toggle stored in daemon `daemonSettings` is accessible to the GUI via a Wails-bound method, not read at event time | Pattern 7 | If the setting must be per-session-launch (not global), architecture changes; global is the locked decision (D-11) |

**A2 and A3 are the highest-risk assumptions.** The current daemon flow in `KillSession` removes from registry immediately. For NATURAL exits, there is no equivalent registry-removal path — the session remains in `StateStopped` state in `pty.SessionRegistry` but the daemon does NOT currently detect or handle natural process exit. This means:

**Critical gap:** The daemon has no mechanism to detect when a session's process exits naturally. `Session.State` will remain `StateRunning` even after the process exits, because nothing calls `registry.Remove()` or transitions `State` to `StateStopped` for natural exits. **The planner must include a backend task to add this exit detection.**

## Open Questions (RESOLVED)

1. **How does the daemon currently detect natural process exit?** (RESOLVED)
   - What we know: `KillSession` calls `backend.Kill()` which calls `killSession()` (SIGHUP -> Wait -> SIGKILL). Natural exits are not handled in the daemon -- `Session.State` stays `StateRunning`.
   - What's unclear: Is there an existing goroutine that calls `cmd.Wait()` for natural exits, or does the process just become a zombie?
   - Recommendation: Add a goroutine in `SessionEngine.CreateSession` (or the backend's `Create`) that calls `cmd.Wait()` after hub.Done() fires and updates `sess.State = StateStopped`. This is prerequisite to any exit event emission.
   - **Resolution:** Plan 01 Task 1 adds an exit watcher goroutine in `CreateSession` that blocks on `<-hub.Done()`, calls `sess.WaitForExit()` to capture the exit code, then transitions `sess.State = StateStopped`. An `onExit` callback is also invoked for the web grace period (D-12).

2. **Should exit event go through daemon HTTP API or be detected in app.go polling?** (RESOLVED)
   - What we know: The GUI is a thin shell (app.go) that communicates with daemon via HTTP. The daemon knows when hub.Done() fires.
   - What's unclear: The cleanest path -- new daemon API endpoint or enhanced polling in app.go.
   - Recommendation: Extend `SessionInfo` with `ExitCode *int` (nil = still running). When session exits naturally, daemon sets `State: "stopped"` and `ExitCode: &code`. The existing `pollSessionStatus` goroutine in app.go polls `ListSessions()` and detects `State: "stopped"`, then emits `session:exit`. This avoids new API routes.
   - **Resolution:** Plan 01 implements the recommendation exactly. `SessionInfo` gains `ExitCode *int` and `Duration *int`. `pollSessionStatus` in app.go is rewritten to poll `ListSessions()` and detect `State == "stopped"`, then emit the `session:exit` Wails event. No new daemon API endpoints needed for exit detection.

3. **How should `handleCloseTab` handle already-exited sessions?** (RESOLVED)
   - What we know: `handleCloseTab` calls `KillSession` which sends SIGHUP. For dead processes, this returns an error.
   - What's unclear: Whether the error is fatal or silently ignorable.
   - Recommendation: In `KillSession` in the daemon, if the process is already dead, skip signal sending but still clean up registry entries. OR: add a `RemoveSession` API that only does registry cleanup without process signaling.
   - **Resolution:** Plan 02 Task 2 handles this: `handleCloseTab` already catches `KillSession` errors with `console.warn` (line 394), so the SIGHUP failure for dead processes is silently ignored. For web serving (D-12), `handleCloseTab` now checks `sessionExits[id]` and skips `ToggleWebServing(id, false)` for naturally-exited sessions, letting the daemon's 10-second grace period timer handle web serving shutdown.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — purely code changes to existing Go/TypeScript codebase)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 |
| Config file | vite.config.ts (inferred — no separate vitest.config) |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test:coverage` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SESS-01 | Tab closes automatically after countdown | unit | `cd frontend && pnpm test -- ExitToast` | No - Wave 0 |
| SESS-01 | "Keep Open" cancels countdown | unit | `cd frontend && pnpm test -- ExitCountdownBanner` | No - Wave 0 |
| SESS-01 | Non-zero exit code skips auto-close (D-09) | unit | `cd frontend && pnpm test -- App.exit` | No - Wave 0 |
| SESS-02 | Countdown starts after session:exit event (not before) | unit | `cd frontend && pnpm test -- App.exit` | No - Wave 0 |
| SESS-03 | Toast appears and shows session name, CLI, exit code | unit | `cd frontend && pnpm test -- ExitToast` | No - Wave 0 |
| SESS-03 | Toast visible when user is on different tab | unit | `cd frontend && pnpm test -- ExitToast` | No - Wave 0 |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm test`
- **Per wave merge:** `cd frontend && pnpm test:coverage`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/ExitToast.test.tsx` -- covers SESS-03
- [ ] `frontend/src/components/__tests__/ExitCountdownBanner.test.tsx` -- covers SESS-01/SESS-02
- [ ] `frontend/src/components/__tests__/App.exit.test.tsx` -- covers SESS-01 (D-09 non-zero skip)

*(Note: Go unit tests for the exit detection logic in engine.go may also be needed -- follow existing `engine_test.go` pattern)*

## Security Domain

This phase involves no authentication, cryptography, session management, or input validation beyond what already exists. The `session:exit` Wails event carries session metadata (name, CLI, duration, exit code) that is already known to the frontend. No new attack surface introduced.

ASVS categories: Not applicable — internal event routing between owned processes, no user-controlled input, no new network endpoints.

## Sources

### Primary (HIGH confidence)
- `internal/relay/hub.go` — hub.Done() channel mechanics, Run() EOF detection [VERIFIED: codebase grep]
- `internal/relay/manager.go` — HubManager.Create() starts hub.Run() goroutine [VERIFIED: codebase grep]
- `internal/status/detector.go` — Watch() function blocks on hub.Done() [VERIFIED: codebase grep]
- `app.go` lines 202-225 — pollSessionStatus pattern (existing polling + event emission) [VERIFIED: codebase grep]
- `app.go` lines 384-412 — handleCloseTab logic (reuse target) [VERIFIED: codebase grep]
- `internal/daemon/engine.go` — CreateSession, KillSession, daemonSettings pattern [VERIFIED: codebase grep]
- `frontend/src/components/UpdateBanner.tsx` — banner component pattern [VERIFIED: codebase grep]
- `frontend/src/components/LocalNetworkBanner.tsx` — multi-state banner, XMarkIcon dismiss [VERIFIED: codebase grep]
- `frontend/src/components/TabBar.tsx` — tab rendering, existing status indicator [VERIFIED: codebase grep]
- `frontend/src/style.css` — .banner-exit pattern, .tab CSS [VERIFIED: codebase grep]

### Secondary (MEDIUM confidence)
- React useState + setInterval countdown pattern [ASSUMED — standard React pattern]
- CSS fixed-position toast stacking [ASSUMED — follows existing CSS conventions in project]

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in use, verified in package.json
- Architecture: HIGH — all integration points verified in codebase; gap in natural exit detection explicitly identified
- Pitfalls: HIGH — race conditions and API limitations verified by code inspection

**Research date:** 2026-04-19
**Valid until:** 2026-05-19 (stable codebase — no external dependencies changing)
