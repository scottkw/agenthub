# Phase 82: Minimize to Tray - Context

**Gathered:** 2026-04-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Add a "Start minimized to system tray" toggle in Settings that controls whether the app window shows on launch. When enabled, the window stays hidden and only the tray icon is visible. The preference persists across restarts via the daemon's settings.json. Three requirements: TRAY-01 (toggle exists), TRAY-02 (window hidden on launch when enabled), TRAY-03 (preference persists).

</domain>

<decisions>
## Implementation Decisions

### Toggle Placement
- **D-01:** Add a new "Behavior" section to the Settings tab for app-level behavior toggles. The "Start minimized to system tray" toggle lives here.
- **D-02:** The toggle is a simple on/off switch with a clear label.

### Section Ordering
- **D-03:** Claude's discretion on where the Behavior section sits relative to existing sections (Paths, Appearance). Choose what makes the best design sense.

### Startup Behavior
- **D-04:** When "Start minimized" is enabled, the window never shows — `domReady()` skips `WindowShow()` and `setDockVisible(true)`. Only the tray icon appears. User clicks tray to open the window.
- **D-05:** When disabled (default), current behavior preserved: `domReady()` shows window with splash as usual.

### Platform Scope
- **D-06:** Minimize-to-tray works on all 3 platforms (macOS, Linux, Windows) from the start. Each already has tray support — the change is in `domReady` and settings, not tray code.

### Persistence
- **D-07:** Preference stored in `daemonSettings` struct (add `StartMinimized bool` field) and persisted to `settings.json` via the existing `loadSettingsFromDisk`/`saveSettingsToDisk` infrastructure from Phase 79.
- **D-08:** New Wails-bound methods needed: `GetStartMinimized() bool` and `SetStartMinimized(bool)` following the existing `GetCLIPaths`/`SaveCLIPaths` pattern.

### Claude's Discretion
- Behavior section ordering within Settings tab
- Toggle component style (checkbox, switch, etc.) — should match existing Settings visual patterns
- Whether `domReady` reads the setting from the daemon client or if the backend passes it via a startup event
- How to handle the edge case where daemon is unreachable at startup (probably just show window as fallback)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Startup / Window Lifecycle
- `main.go` ~L67 — `StartHidden: true` in Wails options (already set)
- `app.go` §`domReady` (~L77-80) — Currently unconditionally calls `WindowShow` + `setDockVisible(true)` — must become conditional
- `app.go` §`beforeClose` (~L161-172) — Hides window on close; existing hide/show pattern
- `app.go` §`startup` (~L82-110) — App initialization, daemon client setup

### Tray Integration
- `tray.go` §`onTrayShow` (~L40-43) — Existing "show window from tray" callback (macOS)
- `tray_linux.go` §`onShow` (~L271-280) — Linux equivalent
- `tray_windows.go` §`wndProc` (~L272-340) — Windows tray click handling

### Settings Persistence
- `internal/daemon/engine.go` §`daemonSettings` (~L62-65) — Struct to extend with `StartMinimized`
- `internal/daemon/engine.go` §`loadSettingsFromDisk`/`saveSettingsToDisk` (~L72-98) — Persistence infrastructure
- `internal/daemon/engine.go` §`settingsPath` (~L67-69) — `settings.json` location

### Frontend Settings
- `frontend/src/components/SettingsTab.tsx` — Settings panel where Behavior section + toggle will be added
- `frontend/src/App.tsx` §`SettingsTab` usage (~L690) — Where SettingsTab is rendered with props

### Requirements
- `.planning/REQUIREMENTS.md` §TRAY-01, §TRAY-02, §TRAY-03 — Toggle, hidden launch, persistence requirements

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `daemonSettings` struct + `loadSettingsFromDisk`/`saveSettingsToDisk`: Direct extension point — add `StartMinimized bool` field
- `GetCLIPaths`/`SaveCLIPaths` Wails bindings: Pattern to follow for `GetStartMinimized`/`SetStartMinimized`
- `onTrayShow` / `onShow` callbacks: Already handle the "user wants to see the window" flow — this is the un-minimize path
- `StartHidden: true` already in Wails options: Window starts hidden by default — `domReady` controls visibility

### Established Patterns
- Settings persistence: `daemonSettings` JSON → `settings.json` → daemon API → Wails binding → frontend
- Window lifecycle: `StartHidden` → `domReady` shows → `beforeClose` hides → tray click shows
- Platform-specific tray: build tags (`tray.go` macOS, `tray_linux.go`, `tray_windows.go`) with shared `tray_common.go`

### Integration Points
- `app.go` `domReady()` — Gate `WindowShow` on the persisted setting
- `internal/daemon/engine.go` — Extend `daemonSettings` and persistence methods
- `internal/daemon/api.go` — Add API endpoint for start-minimized setting (if following CLI paths pattern)
- `frontend/src/components/SettingsTab.tsx` — Add Behavior section with toggle
- `app.go` Wails bindings — New `GetStartMinimized`/`SetStartMinimized` methods

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 82-minimize-to-tray*
*Context gathered: 2026-04-16*
