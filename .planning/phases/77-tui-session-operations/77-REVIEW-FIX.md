---
phase: 77-tui-session-operations
fixed_at: 2026-04-15T16:45:00Z
review_path: .planning/phases/77-tui-session-operations/77-REVIEW.md
iteration: 2
findings_in_scope: 9
fixed: 9
skipped: 0
status: all_fixed
---

# Phase 77: Code Review Fix Report

**Fixed at:** 2026-04-15T16:45:00Z
**Source review:** .planning/phases/77-tui-session-operations/77-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope: 9
- Fixed: 9
- Skipped: 0

## Iteration 1 -- Critical + Warning Fixes

### CR-01: Silently Discarded fetchErr Masks Remote Connection Failures

**Files modified:** `cmd_attach.go`
**Commit:** e30aea6
**Applied fix:** Replaced `_ = fetchErr` with an early-return error check: `if fetchErr != nil { return fmt.Errorf("attach: cannot reach remote host %q: %w", hostname, fetchErr) }`. This surfaces the actual connection error (TLS failure, 500, unreachable host) instead of silently converting it to "session not found".

### WR-01: Potential Nil Pointer Dereference in executeKill

**Files modified:** `internal/tui/update.go`
**Commit:** 80361cb
**Applied fix:** Added nil guard at the top of `executeKill()`: if `m.killTarget == nil`, the modal is closed and the function returns early without dereferencing. This prevents a panic if the method is ever called without a valid kill target.

### WR-02: AttachSession Always Returns nil, Swallowing Pump Errors

**Files modified:** `internal/attach/attach.go`
**Commit:** 7378b4a
**Applied fix:** Changed the select block to capture the error from whichever pump channel completes first (`r := <-stdinDone` / `r := <-wsDone`) and return `r.err` instead of always returning nil. Context cancellation still returns nil (not a pump error). This allows callers (CLI and TUI) to distinguish clean detach from error disconnect.

### WR-03: Attach Status Check Only Blocks "errored" Sessions

**Files modified:** `internal/tui/update.go`
**Commit:** c030b83
**Applied fix:** Replaced the denylist check (`Status == "errored"`) with an allowlist pattern using a switch statement that permits only `"running"`, `"idle"`, and `"waiting"` statuses. All other statuses (including future unknown ones) now show "Session not available". This is a logic change: `fixed: requires human verification`.

### WR-04: TUI Attach Calls WatchResize Directly in Goroutine, Doubling the Goroutine

**Files modified:** `internal/tui/attach.go`
**Commit:** 295c5bf
**Applied fix:** Removed the `go` prefix from `attach.WatchResize(ctx, conn)` since `WatchResize` already manages its own internal goroutine on Unix. This eliminates the unnecessary wrapper goroutine and the subtle race where signal channel setup might not complete before the first SIGWINCH. Matches the pattern used in `cmd_attach.go:164`.

### WR-05: renderHelpOverlay and renderKillConfirmModal Title Injection Assumes ANSI-Free Border

**Files modified:** `internal/tui/view.go`, `internal/tui/help.go`, `internal/tui/modal.go`
**Commit:** 4f766ac
**Applied fix:** Extracted a shared `injectBorderTitle()` helper function in `view.go` that strips ANSI codes from the border line before computing insertion positions, then reconstructs the line with separately-styled prefix/suffix segments. This prevents corrupting ANSI escape sequences that lipgloss may embed in the border line (for `BorderForeground` coloring). All three call sites (help overlay, new session modal, kill confirm modal) were updated to use the new helper.

## Iteration 2 -- Info Fixes

### IN-01: Unnecessary Loop Variable Capture in Go 1.26

**Files modified:** `cmd_attach.go`, `internal/tui/attach.go`
**Commit:** 0756d5d
**Applied fix:** Removed three `s := s` loop variable capture lines that are unnecessary since Go 1.22+ (project uses Go 1.26.1). The loop variable is now scoped per iteration by default, making these captures dead code noise.

### IN-02: renderWebStatus Uses len(sep) Instead of lipgloss.Width(sep)

**Files modified:** `internal/tui/view.go`
**Commit:** 9642080
**Applied fix:** Changed `len(sep)` to `lipgloss.Width(sep)` in the gap calculation for consistency with the rest of the function. For the current ASCII separator `" | "` the result is identical, but this ensures correctness if the separator ever changes to include Unicode or styled text.

### IN-03: Redundant Build Tag Delegation on Windows

**Files modified:** `cmd_attach_resize.go` (new), `cmd_attach_unix.go` (deleted), `cmd_attach_windows.go` (deleted)
**Commit:** 62ff647
**Applied fix:** Merged `cmd_attach_unix.go` and `cmd_attach_windows.go` into a single `cmd_attach_resize.go` without build tags. Both files had identical function bodies (`attach.WatchResize(ctx, conn)`) since the platform-specific behavior is already handled by `internal/attach/attach_unix.go` and `internal/attach/attach_windows.go`. The extra layer of build-tag indirection added no value.

## Skipped Issues

None -- all findings were fixed.

---

_Fixed: 2026-04-15T16:45:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
