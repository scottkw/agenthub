---
phase: 86-tui-visual-polish
plan: "02"
subsystem: tui
tags: [tui, two-pane, sidebar, layout, lipgloss, agent-badges, focus-routing]
dependency_graph:
  requires: [86-01]
  provides: [two-pane-layout, sidebar-nav, tab-bar, session-frame, home-tab, settings-tab, remote-tab, agent-badges, focus-aware-routing]
  affects: [internal/tui/view.go, internal/tui/update.go, internal/tui/help.go, internal/tui/update_test.go, internal/tui/view_test.go, internal/tui/integration_test.go]
tech_stack:
  added: []
  patterns: [lipgloss.JoinHorizontal two-pane, lipgloss.RoundedBorder session frame, injectBorderTitle, agentBadgeColor per-agent styling]
key_files:
  created: []
  modified:
    - internal/tui/view.go
    - internal/tui/update.go
    - internal/tui/help.go
    - internal/tui/update_test.go
    - internal/tui/view_test.go
    - internal/tui/integration_test.go
decisions:
  - "TestView_TabBar uses ansi.Strip() before string checks because active tab renders each rune in separate ANSI sequence"
  - "Existing TestView_SessionList and TestView_SingleSession updated to check Home tab for AgentHub/count since Sessions tab is default"
  - "integration_test.go updated to call renderHeader() directly for session count check instead of full View() content"
  - "renderHeader() and renderColumnHeaders() kept in file (not called from renderFull) to preserve TestView_HeaderRemoteCount compatibility"
metrics:
  duration_minutes: 19
  completed_date: "2026-04-19"
  tasks_completed: 2
  files_changed: 6
---

# Phase 86 Plan 02: Two-Pane Layout, Content Renderers, and Agent Badges Summary

**One-liner:** Two-pane TUI layout (sidebar + content) with renderSidebar/renderTabBar/renderContentPane dispatch, bordered session/remote frames, Home/Settings/Remote content tabs, colored agent badges, and focus-aware Tab/[/] key routing.

## What Was Built

### Task 1: Focus-Aware Key Routing and update.go Refactor

**update.go** — Refactored `handleKey()` to add tab cycling (PrevTab/NextTab at priority 6) and pane-focus dispatch (priority 7). Renamed `handleMainKey` to `handleContentKey` with Tab-to-sidebar toggle at the top. Added new `handleSidebarKey` function supporting up/down/enter/Q/? navigation.

**help.go** — Added `Tab` (Toggle sidebar/content) and `[/]` (Cycle tabs) bindings to the Navigation group in `buildHelpContent()`.

**update_test.go** — Added three new tests:
- `TestUpdate_TabFocusToggle`: Tab toggles focus between content and sidebar
- `TestUpdate_TabCycle`: `[` and `]` cycle through open tabs with wraparound
- `TestUpdate_SidebarNavigation`: Up/Down navigate sidebar items; Enter opens tab and switches focus to content

### Task 2: Two-Pane View Layout, Content Renderers, and Agent Badges

**view.go** — Full structural refactor:

- `renderFull()`: Replaced old vertical layout with `lipgloss.JoinHorizontal(sidebar, sep, right)` two-pane pattern
- Added `sidebarWidth()` (16) and `contentWidth()` (width - sidebar - 1) helpers
- Updated `nameColWidth()` to use `contentWidth()` instead of `m.width`
- Added `renderSidebar()`: 4 items (Home/Sessions/Remote/Settings) with focus highlighting and BgSidebar background
- Added `renderTabBar()`: horizontal tab strip with active tab underlined
- Added `renderContentPane()`: dispatches to Home/Sessions/Remote/Settings renderers
- Added `renderSessionFrame(cw, ch)`: sessions in bordered lipgloss.RoundedBorder frame with column headers, title injection via `injectBorderTitle`
- Added `wrapInFrame()`: shared helper for bordered frames with title
- Added `renderHomeTab()`: AgentHub branding, version, session stats, web/Tailscale status, hints
- Added `renderSettingsTab()`: read-only display with pointer to CLI
- Added `renderRemoteTab()`: remote sessions in bordered frame
- Updated `renderSessionRow()` and `renderRemoteSessionRow()`: replaced plain agent text with colored `[badge]` via `agentBadgeColor()`
- Updated `renderDividerRow()`, `renderWebStatus()`, `renderHintBar()` to use `contentWidth()`

**view_test.go** — Added 5 new tests:
- `TestView_Sidebar`: sidebar contains all 4 section labels
- `TestView_TabBar`: tab bar renders open tabs (strips ANSI for active tab check)
- `TestView_SessionFrame`: frame contains border chars, Sessions title, NAME column header
- `TestView_AgentBadge`: session row contains `[claude]` badge
- `TestView_HomeTab`: Home tab contains AgentHub, version, "AI coding terminal sessions", "1 running"

**integration_test.go** (updated) — Updated `TestTUIRemoteAndQR_FullFlow` to check `renderHeader()` directly for session count (moved out of full view in two-pane layout).

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | b096446 | feat(86-02): focus-aware key routing and help bindings |
| 2 | 419e641 | feat(86-02): two-pane view layout, content renderers, and agent badges |

## Verification

```
go build ./...              -> exit 0
go test ./internal/tui/... -count=1 -> PASS (all existing + 8 new tests)
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestView_SessionList checked View() for "AgentHub" and "2 sessions"**
- **Found during:** Task 2
- **Issue:** Old test checked full View() output for strings that are now only in the Home tab (not the default Sessions tab)
- **Fix:** Updated test to check `renderHomeTab()` directly for those strings; kept session-data checks in View() output
- **Files modified:** internal/tui/view_test.go
- **Commit:** 419e641

**2. [Rule 1 - Bug] TestView_SingleSession checked View() for "1 session"**
- **Found during:** Task 2
- **Issue:** Same as above — session count string now lives in Home tab
- **Fix:** Updated test to check `renderHomeTab()` for "1 running"
- **Files modified:** internal/tui/view_test.go
- **Commit:** 419e641

**3. [Rule 1 - Bug] TestTUIRemoteAndQR_FullFlow checked View() for "2 local, 2 remote"**
- **Found during:** Task 2
- **Issue:** Session count string from `renderHeader()` is no longer called from `renderFull()` in the two-pane layout
- **Fix:** Updated test to call `m.renderHeader()` directly
- **Files modified:** internal/tui/integration_test.go
- **Commit:** 419e641

**4. [Rule 1 - Bug] TestView_TabBar failed with ANSI-wrapped active tab string**
- **Found during:** Task 2
- **Issue:** lipgloss renders each rune of the active (bold+underline) tab separately in ANSI sequences, so `strings.Contains(bar, "Sessions")` fails
- **Fix:** Strip ANSI before checking; added `github.com/charmbracelet/x/ansi` import to view_test.go
- **Files modified:** internal/tui/view_test.go
- **Commit:** 419e641

## Known Stubs

None — all render functions produce real content from model state.

## Threat Flags

None — pure local TUI rendering changes with no network surface.

## Self-Check: PASSED

- internal/tui/view.go: FOUND
- internal/tui/update.go: FOUND
- internal/tui/help.go: FOUND
- internal/tui/update_test.go: FOUND
- internal/tui/view_test.go: FOUND
- internal/tui/integration_test.go: FOUND
- Commit b096446: FOUND
- Commit 419e641: FOUND
