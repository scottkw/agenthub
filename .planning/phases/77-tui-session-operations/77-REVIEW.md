---
phase: 77-tui-session-operations
reviewed: 2026-04-15T10:30:00Z
depth: standard
files_reviewed: 24
files_reviewed_list:
  - cmd_attach.go
  - cmd_attach_test.go
  - cmd_attach_unix.go
  - cmd_attach_windows.go
  - go.mod
  - go.sum
  - internal/attach/attach.go
  - internal/attach/attach_test.go
  - internal/attach/attach_unix.go
  - internal/attach/attach_windows.go
  - internal/tui/attach.go
  - internal/tui/attach_test.go
  - internal/tui/cmds.go
  - internal/tui/help.go
  - internal/tui/help_test.go
  - internal/tui/keys.go
  - internal/tui/modal.go
  - internal/tui/model.go
  - internal/tui/styles.go
  - internal/tui/tui.go
  - internal/tui/update.go
  - internal/tui/update_test.go
  - internal/tui/view.go
  - internal/tui/view_test.go
findings:
  critical: 1
  warning: 5
  info: 3
  total: 9
status: issues_found
---

# Phase 77: Code Review Report

**Reviewed:** 2026-04-15T10:30:00Z
**Depth:** standard
**Files Reviewed:** 24
**Status:** issues_found

## Summary

Phase 77 adds TUI session operations: attach (via tea.Exec), new session creation (modal), kill confirmation (modal), and inline rename. The CLI attach flow (`cmd_attach.go`) and shared attach library (`internal/attach/`) are well-structured with good separation of concerns and solid test coverage.

Key concerns: one silently-discarded error in a remote attach path that could mask connection failures, a potential nil pointer dereference in the kill modal, and several minor issues around error handling and robustness.

## Critical Issues

### CR-01: Silently Discarded fetchErr Masks Remote Connection Failures

**File:** `cmd_attach.go:211`
**Issue:** The error from `fetchPeerSessions` / `fetchPeerSessionsWithClient` is explicitly discarded with `_ = fetchErr`. If the remote peer is unreachable, returns a 500, or has a TLS error, the user sees "session not found on remote host" instead of the actual connection error. This is misleading -- the session may exist but the fetch failed. The comment says "returns empty slice on error" but this converts a hard failure (informative) into silent corruption (the project's CLAUDE.md explicitly warns against this pattern under "Silent Fallbacks").

**Fix:**
```go
if fetchErr != nil {
    return fmt.Errorf("attach: cannot reach remote host %q: %w", hostname, fetchErr)
}
```

## Warnings

### WR-01: Potential Nil Pointer Dereference in executeKill

**File:** `internal/tui/update.go:269`
**Issue:** `executeKill()` accesses `m.killTarget.ID` without a nil check. While the call site (`handleKillConfirmKey`) should only be reached when `killTarget` is set, the method is exported on Model (public receiver) and could be called from test code or future refactors when `killTarget` is nil, causing a panic.

**Fix:**
```go
func (m Model) executeKill() (tea.Model, tea.Cmd) {
    if m.killTarget == nil {
        m.modal = modalNone
        return m, nil
    }
    id := m.killTarget.ID
    // ...
}
```

### WR-02: AttachSession Always Returns nil, Swallowing Pump Errors

**File:** `internal/attach/attach.go:50-58`
**Issue:** `AttachSession` discards the error from both `stdinDone` and `wsDone` channels. When `WsOutputPump` returns a WebSocket read error (e.g., connection reset, server crash), or `StdinPump` returns a write error, the caller always receives `nil`. This means `cmd_attach.go:166` and `internal/tui/attach.go:106` can never distinguish a clean detach from an error disconnect. For the TUI attach path, this means `attachDoneMsg.err` is always nil, and the error toast ("Attach error: ...") is dead code.

**Fix:**
```go
select {
case r := <-stdinDone:
    conn.Close(websocket.StatusNormalClosure, "detach")
    return r.err
case r := <-wsDone:
    conn.Close(websocket.StatusNormalClosure, "detach")
    return r.err
case <-ctx.Done():
    conn.Close(websocket.StatusNormalClosure, "detach")
    return nil
}
```

### WR-03: Attach Status Check Only Blocks "errored" Sessions

**File:** `internal/tui/update.go:167`
**Issue:** The attach guard checks `s.Status == "errored"` but allows attaching to sessions in any other status, including potentially problematic states. If new status values are added in the future (e.g., "stopped", "terminated", "creating"), they would be silently allowed. An allowlist pattern (`status == "running" || status == "idle" || status == "waiting"`) would be more defensive.

**Fix:**
```go
case key.Matches(msg, m.keys.Attach):
    if len(m.sessions) == 0 {
        m.toast = "Session not available"
        m.toastKind = toastError
        m.toastExp = time.Now().Add(2 * time.Second)
        return m, nil
    }
    s := m.sessions[m.selected]
    switch s.Status {
    case "running", "idle", "waiting":
        // OK to attach
    default:
        m.toast = "Session not available"
        m.toastKind = toastError
        m.toastExp = time.Now().Add(2 * time.Second)
        return m, nil
    }
```

### WR-04: TUI Attach Calls WatchResize Directly in Goroutine, Doubling the Goroutine

**File:** `internal/tui/attach.go:103`
**Issue:** The code calls `go attach.WatchResize(ctx, conn)` but `attach.WatchResize` (on Unix) already spawns its own internal goroutine. This means there's an unnecessary extra goroutine wrapper. While not a bug per se, the outer `go` call means the function returns immediately before the inner goroutine is set up, creating a subtle race: the signal channel setup might not complete before the first SIGWINCH arrives. Compare with `cmd_attach.go:164` which calls `watchResize(ctx, conn)` without the `go` prefix, delegating goroutine management to the callee.

**Fix:**
```go
// Remove the "go" prefix -- WatchResize manages its own goroutine.
attach.WatchResize(ctx, conn)
```

### WR-05: renderHelpOverlay and renderKillConfirmModal Title Injection Assumes ANSI-Free Border

**File:** `internal/tui/help.go:37-39` and `internal/tui/modal.go:179-181`
**Issue:** The title injection logic uses `copy(runes[insertPos:insertPos+len(titleRunes)], titleRunes)` to overwrite the top border with the title. This assumes the top border line is plain rune characters. However, lipgloss may embed ANSI escape sequences in the border line (for `BorderForeground` coloring), meaning the rune slice contains invisible escape code runes. Overwriting at `insertPos := 3` can corrupt the ANSI sequences, producing garbled output on some terminals. The risk increases if lipgloss internals change their escape code placement.

**Fix:** Consider using lipgloss layout utilities to place the title above or within the border instead of raw rune manipulation, or strip ANSI codes before computing the insertion position.

## Info

### IN-01: Unnecessary Loop Variable Capture in Go 1.26

**File:** `cmd_attach.go:82`, `cmd_attach.go:215`, `internal/tui/attach.go:45`
**Issue:** Lines like `s := s // capture loop variable` are no longer necessary as of Go 1.22+ (the project uses Go 1.26.1 per go.mod). The loop variable is now scoped per iteration by default.

**Fix:** Remove the `s := s` lines. They're harmless but add noise.

### IN-02: `renderWebStatus` Uses `len(sep)` Instead of `lipgloss.Width(sep)` for Gap Calculation

**File:** `internal/tui/view.go:266`
**Issue:** The gap calculation uses `len(sep)` which counts bytes, not display width. For the current separator `" | "` this is fine (all ASCII), but it's inconsistent with the rest of the function which uses `lipgloss.Width()` for display-width-aware calculations. If the separator ever changes to include Unicode or styled text, this would break alignment.

**Fix:**
```go
gap := m.width - lipgloss.Width(webPart) - lipgloss.Width(sep) - lipgloss.Width(right)
```

### IN-03: Redundant Build Tag Delegation on Windows

**File:** `cmd_attach_windows.go` and `cmd_attach_unix.go`
**Issue:** Both `cmd_attach_windows.go` and `cmd_attach_unix.go` contain identical function bodies (`attach.WatchResize(ctx, conn)`). The platform-specific behavior is already handled by the build tags in `internal/attach/attach_unix.go` and `internal/attach/attach_windows.go`. The `cmd_attach_*.go` wrapper files add an extra layer of indirection with no behavioral difference. Consider having a single `cmd_attach_resize.go` without build tags that calls `attach.WatchResize(ctx, conn)`.

**Fix:** Merge into a single file without build tags since the underlying package already handles platform differences.

---

_Reviewed: 2026-04-15T10:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
