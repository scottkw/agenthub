---
phase: 86-tui-visual-polish
reviewed: 2026-04-19T14:22:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - cmd_tui.go
  - internal/pty/registry.go
  - internal/tui/help.go
  - internal/tui/integration_test.go
  - internal/tui/keys.go
  - internal/tui/model.go
  - internal/tui/styles_test.go
  - internal/tui/styles.go
  - internal/tui/tui.go
  - internal/tui/update_test.go
  - internal/tui/update.go
  - internal/tui/view_test.go
  - internal/tui/view.go
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 86: Code Review Report

**Reviewed:** 2026-04-19T14:22:00Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

The Phase 86 TUI visual polish introduces a two-pane layout (sidebar + content), horizontal tab bar, Home/Settings/Remote tab views, per-agent badge colors, and a Tokyo Night color theme. The codebase is well-structured with clear separation between model, update, and view layers. Thread safety in the PTY registry is correct. Test coverage is thorough (30+ test functions covering navigation, modals, remote sessions, tab cycling, and sidebar focus).

Two warning-level issues were found: an incorrect row index comparison in the Remote tab that causes spurious selection highlighting, and an implicit coupling between sidebar item ordering and tabID enum values that could cause silent misbehavior if either changes. Three info-level items relate to dead code left over from the layout refactor and a magic number.

## Warnings

### WR-01: Remote Tab Uses Wrong Index Space for Selection Highlighting

**File:** `internal/tui/view.go:422`
**Issue:** `renderRemoteTab` passes `len(rows)` as the `idx` parameter to `renderRemoteSessionRow`, but `renderRemoteSessionRow` (line 618) compares `idx == m.selected` where `m.selected` is an index into `m.unifiedList`. These are different index spaces. When `m.selected` happens to coincidentally equal the local row count, a remote session row will be incorrectly highlighted with the selection cursor and background color, even though the user has not selected that row.
**Fix:** The Remote tab is a standalone view (not driven by `m.unifiedList` navigation), so selection highlighting does not apply. Pass `-1` as the index to disable selection, or introduce a separate selection state for tab-specific views:
```go
// In renderRemoteTab, line 422:
rows = append(rows, m.renderRemoteSessionRow(&g.Sessions[i], -1))
```

### WR-02: Fragile Implicit Coupling Between Sidebar Index and tabID Values

**File:** `internal/tui/update.go:380`
**Issue:** `handleSidebarKey` casts `m.sidebarFocus` (an int 0-3) directly to `tabID` via `tabID(m.sidebarFocus)`. This works only because the sidebar items happen to be ordered identically to the `tabID` iota values (Home=0, Sessions=1, Remote=2, Settings=3). If either the sidebar order or the tabID enum order changes, this cast will silently map to the wrong tab with no compile-time or test-time error.
**Fix:** Use an explicit mapping array instead of a raw int cast:
```go
var sidebarTabs = [4]tabID{tabHome, tabSessions, tabRemote, tabSettings}

// In handleSidebarKey:
case "enter":
    m.openTab(sidebarTabs[m.sidebarFocus])
    m.panesFocus = focusContent
```

## Info

### IN-01: Dead Code -- renderHeader, renderColumnHeaders, renderSessionList

**File:** `internal/tui/view.go:431`, `internal/tui/view.go:468`, `internal/tui/view.go:482`
**Issue:** Three functions (`renderHeader`, `renderColumnHeaders`, `renderSessionList`) are no longer called from any production rendering path. They were superseded by the new tab-based layout (`renderSessionFrame`, `renderTabBar`, etc.) in Phase 86. `renderHeader` is only called from two test files (`integration_test.go:65`, `view_test.go:353`). `renderColumnHeaders` and `renderSessionList` are entirely unreferenced.
**Fix:** Remove `renderColumnHeaders` and `renderSessionList` as dead code. If `renderHeader` is still needed for test assertions, consider keeping it or migrating those tests to assert on the Home tab content instead.

### IN-02: Magic Number for Sidebar Bounds

**File:** `internal/tui/update.go:376`
**Issue:** The sidebar navigation bound `m.sidebarFocus < 3` uses a hardcoded magic number that must stay synchronized with the number of sidebar items (currently 4: Home, Sessions, Remote, Settings). Related to WR-02's coupling concern.
**Fix:** Define a constant or derive from the sidebar items:
```go
const sidebarItemCount = 4

// In handleSidebarKey:
if m.sidebarFocus < sidebarItemCount-1 {
```

### IN-03: Redundant iota on Every tabID Constant

**File:** `internal/tui/model.go:50-55`
**Issue:** The `tabID` const block specifies `iota` on every line. In Go, `iota` only needs to appear on the first constant; subsequent constants in the same block auto-increment. While functionally correct, repeating `iota` is non-idiomatic and may mislead readers into thinking each value is explicitly assigned.
**Fix:** Use idiomatic Go iota pattern:
```go
const (
    tabHome     tabID = iota
    tabSessions
    tabRemote
    tabSettings
)
```

---

_Reviewed: 2026-04-19T14:22:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
