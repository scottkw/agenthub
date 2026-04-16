---
phase: 76-tui-foundation
reviewed: 2026-04-15T12:00:00Z
depth: standard
files_reviewed: 16
files_reviewed_list:
  - cmd_cli.go
  - cmd_tui.go
  - internal/daemon/engine.go
  - internal/daemon/types.go
  - internal/tui/cmds.go
  - internal/tui/help.go
  - internal/tui/help_test.go
  - internal/tui/keys.go
  - internal/tui/model.go
  - internal/tui/styles.go
  - internal/tui/tui.go
  - internal/tui/update.go
  - internal/tui/update_test.go
  - internal/tui/view.go
  - internal/tui/view_test.go
  - main.go
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 76: Code Review Report

**Reviewed:** 2026-04-15T12:00:00Z
**Depth:** standard
**Files Reviewed:** 16
**Status:** issues_found

## Summary

This review covers the Phase 76 TUI foundation: the `internal/tui/` package (Bubble Tea terminal UI), the `cmd_tui.go` entry point, and supporting files in `internal/daemon/` and `main.go`. Overall the code is well-structured, idiomatic Go, with clean separation of concerns (model/update/view/cmds/styles/keys/help). Tests cover key behaviors including navigation, message handling, rendering states, and the help overlay. Two warnings and three informational items were found. No security issues or critical bugs were identified.

## Warnings

### WR-01: Rune-level copy on ANSI-styled string in help overlay title injection

**File:** `internal/tui/help.go:36-39`
**Issue:** The title-injection logic converts the rendered border line to a `[]rune` slice and uses `copy` to insert the ANSI-styled title string at a fixed rune offset. Since both the border line and the title contain ANSI escape sequences (from lipgloss styling), operating on raw runes is unreliable. The guard `borderWidth > titleWidth+4` compares visual widths (ignoring escape code rune counts), but `copy(runes[insertPos:insertPos+len(titleRunes)], titleRunes)` operates on rune offsets that include escape codes. If the styled title has more runes (including escape codes) than the remaining runes in the border line after position 3, this will panic with an index-out-of-bounds. Even when it doesn't panic, the copy corrupts the ANSI escape sequences in the border, potentially producing garbled terminal output.
**Fix:** Add a bounds check before the copy, and consider using lipgloss's `PlaceOverlay` or string-based insertion that accounts for ANSI sequences. At minimum, guard against out-of-bounds:
```go
if insertPos+len(titleRunes) <= len(runes) {
    copy(runes[insertPos:insertPos+len(titleRunes)], titleRunes)
    lines[0] = string(runes)
}
```
For a more robust solution, strip ANSI from the border line, insert the raw title at the correct visual position, then re-apply border styling. Or use lipgloss's overlay facilities which handle ANSI-aware placement.

### WR-02: Toast message has no dedicated expiry tick, causing stale display

**File:** `internal/tui/update.go:108-117` and `internal/tui/view.go:237-239`
**Issue:** When a reserved key (Enter, n) is pressed, a toast message is set with a 2-second expiry (`time.Now().Add(2*time.Second)`). The toast is rendered in `renderWebStatus` only if `time.Now().Before(m.toastExp)`. However, there is no tea.Cmd scheduled to trigger a re-render when the toast expires. The UI only re-renders on incoming messages (key press, tick, session data). The background tick fires every 2 seconds, so the toast could remain visible for up to ~4 seconds (2s toast + up to 2s until next tick triggers a re-render). The toast field also retains its string value after expiry; it is never cleared from the model.
**Fix:** Schedule a dedicated tea.Tick that fires at toast expiry to force a re-render, and clear the toast string on expiry:
```go
case key.Matches(msg, m.keys.Attach):
    m.toast = "Coming in next update"
    m.toastExp = time.Now().Add(2 * time.Second)
    return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
        return toastExpiredMsg{}
    })
```
Then handle `toastExpiredMsg` in Update to clear `m.toast`.

## Info

### IN-01: Inconsistent width calculation: len() vs lipgloss.Width() for separator

**File:** `internal/tui/view.go:242`
**Issue:** The gap calculation uses `len(sep)` for the separator string `" | "` while using `lipgloss.Width()` for `webPart` and `right`. For this specific ASCII string, `len()` and `lipgloss.Width()` return the same value (3), so there is no visual bug. However, the inconsistency could lead to layout bugs if the separator is ever changed to include styled or multi-byte characters.
**Fix:** Use `lipgloss.Width(sep)` for consistency:
```go
gap := m.width - lipgloss.Width(webPart) - lipgloss.Width(sep) - lipgloss.Width(right)
```

### IN-02: Column header layout lacks status column label

**File:** `internal/tui/view.go:94-96`
**Issue:** The column headers format string starts with 4 spaces of padding (`"    "`) to account for the cursor (2) and status glyph (2) columns, but there is no visible label for the status glyph column. This is likely intentional (status is a color-coded glyph, not a text column), but the magic number `4` in the format string and `53` in `nameColWidth()` are undocumented and could drift from the actual row layout in `renderSessionRow`.
**Fix:** Add a brief comment documenting the padding breakdown:
```go
// Padding: 2 (cursor "> ") + 2 (status glyph + space) = 4 leading spaces
row := fmt.Sprintf("    %-*s  %-12s  %-20s  %7s",
```

### IN-03: Unused fields in WebServerStatusResponse propagated to TUI model

**File:** `internal/tui/model.go:13` and `internal/daemon/types.go:74-80`
**Issue:** The `Model.webStatus` field stores the full `WebServerStatusResponse` struct which includes `Mode` and `Addr` fields. The TUI only uses `Running` and `URL`. This is not a bug but the TUI carries unused data in its model on every tick cycle.
**Fix:** No action needed for Phase 76. If the TUI grows more complex, consider a TUI-specific status struct that only carries the fields it renders.

---

_Reviewed: 2026-04-15T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
