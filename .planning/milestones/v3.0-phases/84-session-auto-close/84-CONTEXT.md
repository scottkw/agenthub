# Phase 84: Session Auto-Close - Context

**Gathered:** 2026-04-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Automatically close a session tab when the agent process exits, with output preservation, countdown with cancel, and user notification. This phase adds exit detection from the backend, a new Wails event, and frontend auto-close behavior with toast + inline banner.

</domain>

<decisions>
## Implementation Decisions

### Close Timing & Animation
- **D-01:** 5-second countdown before auto-closing the tab after process exit
- **D-02:** Countdown is cancellable — user can click "Keep Open" to prevent auto-close
- **D-03:** Tab fades out (opacity/dim animation) before removal when countdown completes
- **D-04:** Countdown appears in both the tab (visual indicator) and an inline banner inside the terminal area with a "Keep Open" action button

### Exit Notification
- **D-05:** Toast popup + inline terminal banner when agent exits — toast is visible from any tab, banner provides context in the exiting tab
- **D-06:** Notification includes: session name, agent type, exit code, session duration, and final heuristic status (running/idle/waiting/errored)

### Output Flush Guarantee
- **D-07:** Use PTY EOF detection as the signal that all output has been delivered — countdown starts only after EOF
- **D-08:** Terminal remains fully interactive (scrollable, selectable, copyable) during the countdown period

### Error Exit Behavior
- **D-09:** Non-zero exit code skips auto-close entirely — tab stays open so user can review error output
- **D-10:** Clean exits (exit code 0) trigger the normal countdown + auto-close flow

### Settings & Opt-Out
- **D-11:** Global toggle in Settings to disable auto-close (default: enabled)

### Web Viewer Handling
- **D-12:** GUI tab closes normally regardless of web viewers. Web serving continues briefly with an "Agent exited" message, then stops after a grace period.

### Claude's Discretion
- Exit detection mechanism details (how backend detects natural process exit and propagates it)
- Toast component implementation (new component vs. extending existing banner patterns)
- Fade-out animation duration and CSS approach
- Web serving grace period duration
- Settings toggle placement within the Settings tab

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Session lifecycle
- `internal/pty/session.go` — Session state model (StateRunning/StateStopped), PTY read/write
- `internal/pty/cleanup.go` — killSession process termination (POSIX: SIGHUP → Wait → SIGKILL)
- `internal/daemon/engine.go` — SessionEngine: CreateSession with onStatus callback, ListSessions with state mapping, KillSession cleanup
- `internal/daemon/types.go` — SessionInfo struct (State, Status fields), API request/response types

### Status detection & events
- `internal/status/detector.go` — SessionStatus enum (running/idle/waiting/errored), Watch function, Detector pattern matching
- `app.go` lines 184-225 — pollSessionStatus: polls daemon for 60s, emits `session:status` Wails events
- `app.go` lines 259-276 — KillSession: emits `session:status` with "errored" on kill

### Frontend tab management
- `frontend/src/App.tsx` lines 384-412 — handleCloseTab: kills session, removes tab, cleans up state
- `frontend/src/App.tsx` lines 240-244 — EventsOn('session:status') subscription
- `frontend/src/components/TabBar.tsx` — Tab bar rendering with close buttons

### Notification patterns
- `frontend/src/components/UpdateBanner.tsx` — Existing banner notification pattern
- `frontend/src/components/LocalNetworkBanner.tsx` — Existing conditional banner pattern

### Requirements
- `.planning/REQUIREMENTS.md` — SESS-01, SESS-02, SESS-03

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `EventsOn('session:status', ...)` pattern in App.tsx — existing Wails event subscription for status changes; a new `session:exit` event can follow the same pattern
- `UpdateBanner.tsx` / `LocalNetworkBanner.tsx` — banner components that can inform toast/inline banner design
- `handleCloseTab` in App.tsx — existing close logic that the auto-close flow can reuse

### Established Patterns
- Wails event emission from Go backend (`runtime.EventsEmit`) with map payload
- Status polling via `pollSessionStatus` with daemon client HTTP calls
- Session state tracked as `sessionStatuses` Record in React state

### Integration Points
- Backend: `SessionEngine.CreateSession` already has an `onStatus` callback — exit events could use a similar callback or a new channel
- Backend: PTY read loop in relay hub — EOF from PTY read signals process exit
- Frontend: `App.tsx` useEffect for event subscriptions — new `session:exit` event listener
- Frontend: Tab bar — needs countdown indicator/badge support
- Settings: Existing settings persistence in `daemonSettings` struct and `settings.json`

</code_context>

<specifics>
## Specific Ideas

- Toast should be visible even when user is on a different tab — ensures exit is noticed without tab-switching
- "Keep Open" button should be prominent and easy to click within the 5-second window
- Exit code display should distinguish clean (0) vs error (non-zero) visually — e.g., green vs red accent

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 84-session-auto-close*
*Context gathered: 2026-04-19*
