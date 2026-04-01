# Phase 40: Daemon Management Panel — Research

**Researched:** 2026-04-01
**Phase Requirement IDs:** DMGR-03

## Summary

Phase 40 adds a Daemon Management Panel — a React component accessible within the existing GUI window that lists all active sessions with their status and provides kill/web-serve operations. The critical constraint is **no new Go IPC routes** — the panel must use only existing Wails bindings.

## Existing Wails Bindings (Available Now)

All of these are already bound in `app.go` and have TypeScript stubs in `frontend/src/wailsjs/go/main/App.d.ts`:

| Binding | Signature | Purpose for Panel |
|---------|-----------|-------------------|
| `ListSessions()` | `→ SessionInfo[]` | Populate session list |
| `GetSessionStatus(id)` | `→ string` | Get live status (running/waiting/idle/errored) |
| `KillSession(id)` | `→ void` | Kill a session from the panel |
| `ToggleWebServing(id, enabled)` | `→ void` | Toggle web serving per session |
| `RenameSession(id, name)` | `→ void` | Rename sessions inline |
| `IsWebServerRunning()` | `→ boolean` | Check if web server is active (needed for toggle availability) |
| `GetWebServerURL()` | `→ string` | Get base URL for web-served sessions |

The `SessionInfo` type from `App.d.ts`:
```typescript
interface SessionInfo {
  id: string
  cli: string
  name: string
  state: string
  createdAt: string
}
```

Note: `state` in `SessionInfo` is the last-known state from the daemon's session list. For live status, `GetSessionStatus(id)` should be polled or the `session:status` Wails event should be used (already subscribed in App.tsx).

## Live Status Events (Already Wired)

App.tsx already subscribes to `session:status` events:
```typescript
EventsOn('session:status', (data: { sessionId: string; status: string }) => {
  setSessionStatuses((prev) => ({ ...prev, [data.sessionId]: data.status }))
})
```

The `sessionStatuses` state (`Record<string, string>`) is already maintained in App.tsx. The panel component can receive this as a prop — no additional event subscription needed.

## Architecture Decision: Panel Location in UI

The panel must be "accessible within the existing GUI window" (Success Criterion 1). Options:

### Option A: New Tab Type (like WelcomeTab) — RECOMMENDED
- Add a `type: 'daemon-manager'` tab to the TabBar
- Render `DaemonManagerPanel` when this tab is active (same pattern as WelcomeTab)
- Button in TabBar controls area (gear icon area) or as a dedicated icon
- User can keep it open alongside terminal tabs
- Fits existing architecture perfectly — WelcomeTab is the exact same pattern

### Option B: Sidebar Panel
- Slide-in panel from the left or right
- Requires new layout considerations
- More complex CSS and state management
- Would need careful handling to not affect terminal/FitAddon layout

### Option C: Settings Panel Tab
- Add as a third tab in the existing SettingsPanel modal
- Settings modal is a modal overlay, not great for a persistent management view
- Awkward UX — settings and management are different concerns

**Decision: Option A (Tab Type)** — aligns with existing WelcomeTab pattern, no new layout complexity, reuses Tab/TabBar infrastructure. The panel opens as a closeable tab.

## UI Component Design

### DaemonManagerPanel.tsx

The component renders a session table/list with:
- **Session name** (with cli type badge)
- **Status indicator** (reuse `.tab__status--{running|waiting|idle|errored}` CSS classes)
- **Created timestamp**
- **Actions**: Kill button, Web toggle button

### Data Flow

```
App.tsx (owns all state)
  ├── tabs, sessionStatuses, webEnabled, webServerRunning (existing state)
  └── DaemonManagerPanel (new component)
       ├── sessions: SessionInfo[] (from ListSessions, refreshed)
       ├── sessionStatuses: Record<string, string> (existing App state)
       ├── webServerRunning: boolean (existing)
       ├── webEnabled: Record<string, boolean> (existing)
       ├── onKill: (id) => void (calls existing handleCloseTab or KillSession directly)
       └── onToggleWeb: (id) => void (calls existing handleToggleWeb)
```

### Refresh Strategy

The panel needs to refresh the session list periodically since sessions can be created/killed from other sources (CLI, other windows). Two approaches:

1. **Poll ListSessions every 3s** — Simple, matches the web status bar polling pattern from Phase 39
2. **React to session:status events + manual refresh** — Events already fire on status changes, but don't fire for new sessions created externally

**Recommendation:** Poll `ListSessions()` every 3 seconds when the panel tab is active (use `useEffect` with `setInterval`, cleanup on unmount/inactive). This is consistent with the polling pattern used in the web terminal status bar.

## Entry Point for Panel

Add a button/icon in the TabBar controls area (next to + and gear):
- Small icon or text button "Sessions" / list icon
- Clicking creates/focuses the daemon manager tab
- If tab already exists, focus it; don't create duplicates

Alternative: Add it as a menu item in the gear icon dropdown. But there's no dropdown currently — gear goes directly to SettingsPanel. A third button in the controls area is simpler.

## Styling Approach

Follow existing patterns exactly:
- Dark theme: bg `#1a1b26`, panel bg `#16161e`, borders `#292e42`
- Text: primary `#c0caf5`, secondary `#a9b1d6`, muted `#565f89`
- Accent blue `#7aa2f7`, green `#9ece6a`, red `#f7768e`, amber `#f59e0b`
- Status dots: reuse `.tab__status--{state}` pattern
- Buttons: reuse `.tab-status-bar__btn` or `.settings-panel__btn` pattern
- BEM naming: `.daemon-panel`, `.daemon-panel__session-row`, etc.
- Font: inherit monospace family from body

## Validation Architecture

### Test Strategy

1. **Unit tests** (`DaemonManagerPanel.test.tsx`):
   - Renders session list from mock data
   - Status indicators show correct colors/classes
   - Kill button calls onKill with correct session ID
   - Web toggle calls onToggleWeb with correct session ID
   - Web toggle disabled when web server not running
   - Empty state message when no sessions

2. **Integration verification**:
   - Panel accessible from TabBar button
   - Panel shows real sessions from daemon
   - Kill works and session disappears
   - Web toggle works and reflects state change

### Key Links (what can break)
- `ListSessions()` binding → panel data (if binding changes, panel shows stale/no data)
- `sessionStatuses` prop → status dots (if event subscription breaks, dots stuck)
- `KillSession()` → kill button (if binding fails, sessions can't be killed from panel)
- `ToggleWebServing()` → web toggle (if web server not running, toggle should be disabled)

## Files to Create/Modify

| File | Action |
|------|--------|
| `frontend/src/components/DaemonManagerPanel.tsx` | Create — new panel component |
| `frontend/src/components/__tests__/DaemonManagerPanel.test.tsx` | Create — unit tests |
| `frontend/src/App.tsx` | Modify — add daemon manager tab type, button, rendering logic |
| `frontend/src/components/TabBar.tsx` | Modify — add Tab type 'daemon-manager', add sessions button |
| `frontend/src/style.css` | Modify — add `.daemon-panel` CSS classes |

## Risks & Mitigations

1. **Risk**: ListSessions polling could cause performance issues
   - **Mitigation**: Only poll when panel tab is active; 3s interval is light
   
2. **Risk**: Kill from panel vs close from tab could have different behaviors
   - **Mitigation**: Panel kill should call the same `handleCloseTab` logic to clean up tabs + status + fonts + QR state, or at minimum `KillSession` binding + emit cleanup events

3. **Risk**: Tab type extension could break existing tab logic
   - **Mitigation**: WelcomeTab already proves `type` field works; daemon-manager follows same pattern

## What NOT To Do

- Do NOT add new Go IPC routes (Success Criterion 5)
- Do NOT create a separate OS window (Wails v2 is single-window, SC 1)
- Do NOT duplicate state — reuse App.tsx's existing state via props
- Do NOT add rename to the panel — it already works via tab double-click/context-menu
