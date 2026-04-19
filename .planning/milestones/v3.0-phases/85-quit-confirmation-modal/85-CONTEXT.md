# Phase 85: Quit Confirmation Modal - Context

**Gathered:** 2026-04-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Intercept all quit actions (GUI window close and tray Quit menu item) with a confirmation modal that displays active session details and offers two exit modes: quit GUI only (hide to tray with OS notification) or quit everything (stop daemon and all sessions).

</domain>

<decisions>
## Implementation Decisions

### Zero-Session Behavior
- **D-01:** Always show the quit confirmation modal regardless of session count — even when 0 sessions are active
- **D-02:** When session count is 0, display an adjusted message ("No active sessions") instead of a count. The two exit buttons remain available since daemon state is still relevant.

### Modal Visual Design
- **D-03:** Informational tone with session context — modal lists each active session by name and current status (running/idle/waiting/errored)
- **D-04:** "Quit Everything" button styled with destructive red accent to differentiate from "Quit GUI Only" (the safe default)
- **D-05:** Match existing modal patterns (dark overlay, Escape to close, Cancel button returns to app)

### Tray Quit Intercept
- **D-06:** Tray Quit emits a Wails event (e.g., `app:quit-requested`) instead of calling ShutdownDaemon + runtime.Quit directly
- **D-07:** Frontend listens for the quit-requested event and shows the modal
- **D-08:** When the quit event arrives and the window is hidden, auto-show the window so the modal is visible to the user
- **D-09:** User's modal choice calls back to Go to execute the selected quit behavior

### "Quit GUI Only" Behavior
- **D-10:** "Quit GUI Only" hides the window to tray (same as current close behavior) — daemon and sessions keep running
- **D-11:** After hiding, send an OS-level notification (macOS Notification Center) confirming "AgentHub is still running in the background. N sessions active."

### Window Close Intercept
- **D-12:** Window close (red button) triggers the same quit confirmation modal instead of silently hiding — consistent with tray Quit behavior

### Claude's Discretion
- Wails event naming convention for the quit-requested event
- Modal component structure (new component vs. extending existing patterns)
- OS notification implementation approach (Wails notification API or Go-native)
- Session list rendering within the modal (truncation strategy if many sessions)
- Cancel button placement and keyboard shortcut handling
- Animation/transition for modal appearance

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Quit and window lifecycle
- `app.go` lines 55-64 — App struct with `quitting` bool that controls beforeClose behavior
- `app.go` lines 162-177 — `beforeClose`: currently hides window on close, allows quit when `quitting` is true
- `main.go` lines 62-86 — `runGUI`: Wails app options including `HideWindowOnClose: true`, `OnBeforeClose: app.beforeClose`

### Tray menu and quit
- `tray.go` lines 47-63 — `onTrayQuit`: currently calls ShutdownDaemon + sets quitting + runtime.Quit
- `tray.go` lines 40-45 — `onTrayShow`: shows window and dock icon (pattern for auto-show on quit)
- `tray_common.go` lines 35-54 — Menu structure including "Quit" item

### Existing modal patterns
- `frontend/src/components/NewSessionModal.tsx` — Modal with form inputs, overlay pattern, Escape key handling
- `frontend/src/components/QRModal.tsx` — Simpler modal with overlay click + Escape key close

### Session listing
- `app.go` lines 265-282 — `ListSessions`: returns all sessions with state/status (data source for modal)
- `internal/daemon/types.go` — `SessionInfo` struct with State, Status, Name, CLI fields

### Wails event patterns
- `app.go` lines 240-244 — `EventsOn('session:status')` subscription pattern in frontend
- Wails `runtime.EventsEmit` for Go → frontend event emission

### Requirements
- `.planning/REQUIREMENTS.md` — APP-01, APP-02, APP-03

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `NewSessionModal` / `QRModal` — overlay + Escape key + click-outside patterns to follow for modal structure
- `onTrayShow` in tray.go — existing `runtime.WindowShow` + `setDockVisible` logic to reuse for auto-showing window on quit
- `ListSessions` in app.go — returns session list with status info for populating the modal
- `runtime.EventsEmit` / `EventsOn` — established Wails event pattern for Go ↔ frontend communication

### Established Patterns
- `beforeClose` returns true to prevent quit, false to allow — modal confirmation can use this same gate
- `quitting` bool on App struct — mechanism for tray quit to bypass beforeClose; quit modal will need similar control flow
- Dark overlay + centered panel modal styling from NewSessionModal/QRModal

### Integration Points
- `beforeClose` in app.go — needs to emit event to frontend instead of silently hiding
- `onTrayQuit` in tray.go — needs to emit event instead of directly quitting
- Frontend App.tsx event subscriptions — new `app:quit-requested` listener
- New `QuitConfirmModal` component in frontend/src/components/

</code_context>

<specifics>
## Specific Ideas

- Session list in modal should show agent name + status (e.g., "Claude Code (running)") — same format as the ASCII mockup the user selected
- When 0 sessions: "No active sessions" replaces the session list — keeps modal clean
- OS notification on GUI-only quit should include session count for at-a-glance context
- The modal should feel like a natural extension of the existing modal UX — not a jarring OS dialog

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 85-quit-confirmation-modal*
*Context gathered: 2026-04-19*
