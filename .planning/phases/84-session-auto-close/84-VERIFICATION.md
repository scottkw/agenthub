---
phase: 84-session-auto-close
verified: 2026-04-19T11:45:00Z
status: passed
score: 4/4
overrides_applied: 0
---

# Phase 84: Session Auto-Close Verification Report

**Phase Goal:** Users experience a clean, informed session tab closure when an agent process exits
**Verified:** 2026-04-19T11:45:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | When an agent process exits, its session tab closes automatically without any manual intervention | VERIFIED | `session:exit` event → countdown → `handleCloseTabRef.current(sessionId)` in App.tsx. Exit watcher goroutine in engine.go blocks on `<-hub.Done()`, sets `StateStopped`; `pollSessionStatus` detects `State=="stopped"` and emits the event. |
| 2 | Final terminal output from the exiting agent is visible before the tab disappears (brief flush delay) | VERIFIED | `hub.Done()` closes only after PTY Read returns EOF (all output drained). 5-second countdown provides additional buffer. RESEARCH D-07 honored. |
| 3 | A toast notification or visible indicator appears when the agent exits, giving the user context before close | VERIFIED | `ExitToast` component rendered at App root (line 931 App.tsx), always populated on any `session:exit` event regardless of exit code. `ExitCountdownBanner` rendered inline in terminal area (line 891). |
| 4 | The auto-close behavior does not truncate or drop terminal output still in flight at process exit | VERIFIED | Exit detection uses `hub.Done()` (PTY EOF signal, not raw `cmd.Wait()`). Countdown cannot start until the hub has drained all output to subscribers. |

**Score:** 4/4 truths verified

### Required Artifacts (Plan 01)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/pty/session.go` | `ExitCode()`, `WaitForExit()`, `SetState()` methods | VERIFIED | All three methods present (lines 98-134). |
| `internal/daemon/types.go` | `ExitCode *int`, `Duration *int` fields on SessionInfo | VERIFIED | Both pointer fields present (lines 14-15). |
| `internal/daemon/engine.go` | `autoCloseSession *bool`, getter/setter, exit watcher goroutine in `CreateSession` | VERIFIED | Field at line 37, getter/setter at lines 374-390, exit watcher goroutine at lines 210-219. |
| `internal/daemon/api.go` | `GET/PATCH /settings/auto-close-session` routes, `onExit` callback with `time.AfterFunc(10s)` | VERIFIED | Routes registered (lines 55-56), grace period callback (lines 264-281). |
| `internal/daemon/client.go` | `GetAutoCloseSession`, `SetAutoCloseSession` client methods | VERIFIED | Both methods present (lines 126-138). |
| `app.go` | `pollSessionStatus` detects `State=="stopped"`, `emitExitEvent`, `GetAutoCloseSession`, `SetAutoCloseSession` | VERIFIED | `pollSessionStatus` (line 227), `emitExitEvent` (lines 242-263), Wails bindings (lines 387-405). |
| `frontend/src/wailsjs/go/main/App.js` | `GetAutoCloseSession`, `SetAutoCloseSession` stubs | VERIFIED | Both stubs at lines 72-73. |
| `frontend/src/wailsjs/go/main/App.d.ts` | TS type declarations for both bindings | VERIFIED | Both declarations at lines 114-115. |

### Required Artifacts (Plan 02)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/ExitToast.tsx` | Fixed-position toast for session exits, exports `ExitToast` and `ExitState` | VERIFIED | Component exists, exports both, renders clean/error variants, dismiss and keep-open buttons wired. |
| `frontend/src/components/ExitCountdownBanner.tsx` | Inline countdown banner, exports `ExitCountdownBanner` | VERIFIED | Component exists, "Agent exited cleanly. Tab closes in {n}s.", Keep Open button, role="alert". |
| `frontend/src/App.tsx` | `session:exit` event handler, countdown state, auto-close logic | VERIFIED | `sessionExits` state, `countdownTimers` ref, `autoCloseRef`, `EventsOn('session:exit')` subscription, `handleKeepOpen`, `handleDismissExit`, D-12 web guard. |
| `frontend/src/components/TabBar.tsx` | `exitCountdowns` prop, `.tab--exiting` class, countdown badge | VERIFIED | `exitCountdowns?: Record<string, number>` in props, conditional `.tab--exiting`, `tab__countdown` span. |
| `frontend/src/components/SettingsTab.tsx` | "Session Behavior" section with auto-close toggle | VERIFIED | h3 "Session Behavior", `GetAutoCloseSession`/`SetAutoCloseSession` imports, `autoCloseSession` state, `handleToggleAutoClose`, "Auto-close tab on exit" label. |
| `frontend/src/style.css` | Exit toast, countdown banner, tab exiting CSS | VERIFIED | `.exit-toast` (fixed, z-index 9999), `.exit-toast__item--clean/--error`, `.exit-countdown-banner`, `.tab--exiting` (opacity 0.5), `.tab__countdown` (tabular-nums). |

### Required Artifacts (Plan 03)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/__tests__/ExitToast.test.tsx` | 12+ unit tests for ExitToast | VERIFIED | 13 test cases covering empty state, clean/error variants, meta content, countdown display/hide, callbacks, accessibility, multiple items. |
| `frontend/src/components/__tests__/ExitCountdownBanner.test.tsx` | 5+ unit tests for ExitCountdownBanner | VERIFIED | 5 test cases: countdown message, Keep Open button, callback, role, countdown span. |
| `frontend/src/components/__tests__/App.exit.test.tsx` | 12+ source inspection tests | VERIFIED | 12 test cases checking session:exit subscription, all state/ref names, component renders, callbacks, and cleanup. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/daemon/engine.go` | `internal/relay/hub.go` | `<-hub.Done()` in exit watcher goroutine | VERIFIED | Line 214: `<-hub.Done()` |
| `internal/daemon/engine.go` | `internal/daemon/api.go` | `onExit` callback provided by API layer | VERIFIED | Line 281: `a.engine.CreateSession(..., onExit)` |
| `app.go` | `internal/daemon/client.go` | `pollSessionStatus` detects `State=="stopped"`, emits `session:exit` | VERIFIED | Lines 227-228: `if s.State == "stopped" { a.emitExitEvent(...) }` |
| `frontend/src/App.tsx` | `frontend/src/components/ExitToast.tsx` | `sessionExits` state passed as `exits` prop | VERIFIED | Lines 931-934: `<ExitToast exits={sessionExits} ...>` |
| `frontend/src/App.tsx` | `frontend/src/components/ExitCountdownBanner.tsx` | `exitState` rendered in terminal-wrapper | VERIFIED | Lines 891-895: conditional render with `countdown` prop |
| `frontend/src/App.tsx` | Wails runtime `EventsOn` | `session:exit` event subscription | VERIFIED | Lines 332-401: `const offExit = EventsOn('session:exit', ...)` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `ExitToast` | `exits: Record<string, ExitState>` | `sessionExits` React state, populated by `session:exit` Wails event | Yes — event payload populated by daemon `ListSessions()` returning live `ExitCode *int` and `Duration *int` from process state | FLOWING |
| `ExitCountdownBanner` | `countdown: number` | `sessionExits[tab.sessionId].countdown` — decremented by `setInterval` (1s ticks) | Yes — countdown is initialized to 5 from the event and decremented by real timer | FLOWING |
| `TabBar` countdown badge | `exitCountdowns?: Record<string, number>` | Derived map from `sessionExits` (entries with `exitCode === 0`, not `cancelled`, `countdown > 0`) | Yes — derived from live state | FLOWING |
| `SettingsTab` auto-close toggle | `autoCloseSession` | `GetAutoCloseSession()` Wails call on mount → daemon `settings.json` | Yes — reads from `daemonSettings.AutoCloseSession *bool`, persisted to disk | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go build succeeds | `go build ./...` | No output (pass) | PASS |
| Go vet passes | `go vet ./...` | No output (pass) | PASS |
| Backend tests pass | `go test ./internal/pty/... ./internal/daemon/... -count=1` | `ok internal/pty`, `ok internal/daemon` | PASS |
| TypeScript compiles clean | `npx tsc --noEmit` | No output (pass) | PASS |
| Frontend test suite | `pnpm test -- --run` | 496/496 tests pass across 25 files | PASS |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SESS-01 | 84-01, 84-02, 84-03 | Session tab automatically closes when the agent process exits | SATISFIED | Exit watcher (`hub.Done()` → `StateStopped`), `pollSessionStatus` emits `session:exit`, App.tsx countdown auto-closes via `handleCloseTab`. |
| SESS-02 | 84-01, 84-02, 84-03 | Brief delay before auto-close allows final output to flush to terminal | SATISFIED | `hub.Done()` ensures PTY EOF (output fully flushed). 5-second countdown adds additional buffer. D-07 honored. |
| SESS-03 | 84-02, 84-03 | Toast or indicator notifies user that the agent exited before the tab closes | SATISFIED | `ExitToast` (fixed-position, visible from any tab) + `ExitCountdownBanner` (inline in terminal area). Both shown on any exit event. |

No orphaned requirements. REQUIREMENTS.md maps only SESS-01, SESS-02, SESS-03 to Phase 84. All three are satisfied.

### Anti-Patterns Found

No blockers or warnings. Scanned modified files for TODO/FIXME/PLACEHOLDER, empty implementations, and hardcoded empty data — none found.

One note: `ExitToast` returns `null` when `exits` is empty (line 23). This is an intentional conditional render, not a stub — the empty-map case means nothing to display.

### Human Verification Required

The following cannot be verified programmatically:

1. **Countdown visual feedback**
   - **Test:** Launch an agent session, let it exit naturally (exit code 0). Observe the session tab in the GUI.
   - **Expected:** Tab fades to ~0.5 opacity, countdown badge appears (e.g., "5s"), ExitCountdownBanner appears in terminal area, ExitToast appears bottom-right. Countdown ticks 5→4→3→2→1→tab closes.
   - **Why human:** Visual UI behavior requires browser rendering.

2. **Keep Open cancels auto-close**
   - **Test:** During countdown, click "Keep Open" in either the toast or banner.
   - **Expected:** Countdown stops, tab stays open, banner disappears, tab opacity restores to 1.0. Toast remains with no countdown row.
   - **Why human:** UI interaction and state cancellation.

3. **Non-zero exit skips auto-close**
   - **Test:** Run an agent that exits with a non-zero code.
   - **Expected:** Toast appears with red left border and "exited with error". No countdown, no inline banner. Tab stays open indefinitely.
   - **Why human:** Requires running an agent with a failure exit.

4. **Settings toggle persists**
   - **Test:** Open Settings > Session Behavior, toggle "Auto-close tab on exit" to disabled. Restart the app. Verify toggle still shows disabled.
   - **Expected:** Setting persists across restarts via `settings.json`.
   - **Why human:** Requires app restart to verify persistence.

5. **Web serving grace period (D-12)**
   - **Test:** Start a session with web serving enabled. Let the agent exit naturally. Observe web serving endpoint is still accessible for ~10 seconds after exit, then becomes unavailable.
   - **Expected:** Web serving stays up for 10 seconds, then `DisableSession` fires.
   - **Why human:** Requires external HTTP request timing and web server state inspection.

### Gaps Summary

No gaps. All must-haves verified. Goal achieved.

---

_Verified: 2026-04-19T11:45:00Z_
_Verifier: Claude (gsd-verifier)_
