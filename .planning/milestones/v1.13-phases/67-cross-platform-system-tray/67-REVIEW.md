---
phase: 67-cross-platform-system-tray
reviewed: 2026-04-11T12:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - go.mod
  - tray_common.go
  - tray_common_test.go
  - tray_linux.go
  - tray_linux_test.go
  - tray_test.go
  - tray_windows.go
  - tray_windows_test.go
  - tray.go
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
status: issues_found
---

# Phase 67: Code Review Report

**Reviewed:** 2026-04-11T12:00:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

The cross-platform system tray implementation spans macOS (cgo/Cocoa via `tray.go`), Linux (D-Bus StatusNotifierItem via `tray_linux.go`), and Windows (Shell_NotifyIconW via `tray_windows.go`), with shared menu construction logic in `tray_common.go`. The architecture is clean -- a single `BuildMenuItems` function is the source of truth for menu layout, and each platform adapter consumes it. Bounds checking on session indices is present in all three platform implementations.

The main concerns are a data race on the `App.quitting` field written from unsynchronized goroutines, a defensive gap in `trayTooltip` for negative inputs, and redundant PNG re-decoding on every Linux tray update.

## Warnings

### WR-01: Data race on App.quitting field

**File:** `tray_linux.go:290`, `tray_windows.go:314`, `tray.go:57`
**Issue:** The `app.quitting` field (declared as a plain `bool` in `app.go:64`) is written from goroutines spawned by tray callbacks (`onQuit` on Linux, `wndProc` WM_COMMAND/IDM_QUIT on Windows, `onTrayQuit` on macOS) and read on the Wails main goroutine in `beforeClose` (`app.go:162`). There is no synchronization (no mutex, no atomic). This is a data race detectable by `go test -race`.

**Fix:** Use `atomic.Bool` instead of `bool`:
```go
// In app.go:
import "sync/atomic"

type App struct {
    // ...
    quitting  atomic.Bool  // true when tray Quit was clicked
    // ...
}

// Writers (tray callbacks):
app.quitting.Store(true)

// Reader (beforeClose):
if a.quitting.Load() {
    return false
}
```

### WR-02: trayTooltip does not handle negative input

**File:** `tray_common.go:7-16`
**Issue:** If `trayTooltip` is called with a negative value (e.g., `n = -1`), it falls through to the `default` case and returns `"AgentHub -- -1 sessions"`. While the current callers always pass `len(sessions)` (which is non-negative), this is a latent bug that would produce nonsensical UI text if a future caller passes an invalid value. Defensive programming would clamp or handle this.

**Fix:** Treat negative values the same as zero:
```go
func trayTooltip(n int) string {
    switch {
    case n <= 0:
        return "AgentHub \u2014 no sessions"
    case n == 1:
        return "AgentHub \u2014 1 session"
    default:
        return fmt.Sprintf("AgentHub \u2014 %d sessions", n)
    }
}
```

### WR-03: Windows tooltip truncation may lose null terminator

**File:** `tray_windows.go:459-462`, `tray_windows.go:526-532`
**Issue:** `syscall.StringToUTF16` returns a null-terminated UTF-16 slice. The copy is bounded by `len(nid.SzTip)` (128 uint16 elements). If the tooltip string (including null terminator) exceeds 128 UTF-16 code units, the null terminator is silently dropped, producing an unterminated wide string passed to `Shell_NotifyIconW`. While current tooltip strings are short enough to never hit this (max ~30 characters), this is a latent bug.

**Fix:** Reserve space for the null terminator:
```go
copyLen := len(tipUTF16)
if copyLen > len(tipArr) {
    copyLen = len(tipArr)
}
// Ensure null termination.
if copyLen == len(tipArr) && tipUTF16[copyLen-1] != 0 {
    tipArr[copyLen-1] = 0
} else {
    copy(tipArr[:], tipUTF16[:copyLen])
}
```
Or more simply, cap at `len(tipArr) - 1` to always leave room for a trailing null.

## Info

### IN-01: Double separator when no sessions

**File:** `tray_common.go:39-57`
**Issue:** When `sessions` is empty, `BuildMenuItems` produces: `[Open AgentHub, separator, separator, Quit]` -- two consecutive separators with nothing between them. The code comment at line 37-38 explicitly documents this as intentional behavior. Tests verify it. This is noted for awareness; some platform tray hosts may render double separators as a visible gap.

**Fix:** No fix needed if intentional. If the visual double-separator is undesirable on any platform, conditionally omit the first separator when sessions is empty:
```go
if len(sessions) > 0 {
    for i, s := range sessions { ... }
}
```

### IN-02: Unused variables in Linux test

**File:** `tray_linux_test.go:54-55`
**Issue:** Lines 54-55 assign `children` and `ok` from the `layout.props` map lookup, then immediately discard both with blank identifiers (`_ = children; _ = ok`). These two lines serve no purpose and are dead code.

**Fix:** Remove the unused map lookup:
```go
// Delete these two lines:
// children, ok := layout.props["children-display"]
// _ = children
// _ = ok
```

### IN-03: Commented-out plan reference in Windows test

**File:** `tray_windows_test.go:48-60`
**Issue:** Lines 48-60 contain a long inline comment discussing a discrepancy between the plan spec ("7 items") and the actual implementation (6 items). While useful during development, this reasoning block should be condensed to a single-line comment now that the implementation is settled.

**Fix:** Replace the 12-line comment block with a concise note:
```go
// With 2 sessions: Open + sep + session0 + session1 + sep + Quit = 6 items.
wantLen := 6
```

---

_Reviewed: 2026-04-11T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
