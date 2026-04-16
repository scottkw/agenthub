---
phase: 78-tui-remote-qr
reviewed: 2026-04-15T14:22:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - cmd_tui.go
  - internal/tui/attach_test.go
  - internal/tui/cmds.go
  - internal/tui/help.go
  - internal/tui/help_test.go
  - internal/tui/integration_test.go
  - internal/tui/keys.go
  - internal/tui/model.go
  - internal/tui/qr.go
  - internal/tui/tui.go
  - internal/tui/update.go
  - internal/tui/update_test.go
  - internal/tui/view.go
  - internal/tui/view_test.go
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
---

# Phase 78: Code Review Report

**Reviewed:** 2026-04-15T14:22:00Z
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Phase 78 adds two features to the TUI: (1) remote session display via tailnet peer discovery and (2) QR code overlay for session URLs. The implementation is well-structured with clean separation of concerns: `FetchRemoteFn` callback injection avoids the import cycle between `internal/tui` and `package main`, the unified list model correctly interleaves local sessions, dividers, and remote sessions, and the QR overlay follows established modal patterns.

The code quality is high overall. Key dispatch priority is correctly ordered (editing > kill confirm > new session > QR > help > main), navigation correctly skips divider rows, selection identity is preserved across list rebuilds, and operations (kill, rename) are properly blocked on remote sessions. Test coverage is thorough with 14 test files covering all new behaviors including integration flow tests.

Two warnings and two informational items were found. No critical issues.

## Warnings

### WR-01: Silent error discarding in fetchRemoteFn callback (cmd_tui.go)

**File:** `cmd_tui.go:32-39`
**Issue:** Both `client.ListTailnetPeers()` (line 32) and `fetchPeerSessions()` (line 39) silently discard errors with `_, _`. While graceful degradation is intentional here (the CLI uses the same pattern in `cmd_cli.go:113`), this violates the project's "Silent Fallbacks" principle from CLAUDE.md: "or {} converts hard failures (informative) into silent corruption (expensive)." If tailnet connectivity fails or a peer returns an HTTP error, the user sees an empty remote section with no indication that remote discovery was attempted and failed. A previous code review (Phase 74) flagged the identical pattern in `cmd_attach.go` as a critical issue.

**Fix:** At minimum, log the error at debug level so users can diagnose connectivity issues. Alternatively, return an error indicator alongside groups so the TUI can display a toast like "Remote session fetch failed":
```go
peers, err := client.ListTailnetPeers()
if err != nil {
    // At minimum, a debug log. Better: surface to user.
    slog.Debug("tailnet peer discovery failed", "err", err)
}
// ...
peerSessions, err := fetchPeerSessions(ctx, fqdn, tailnet.DefaultProbePort)
if err != nil {
    slog.Debug("peer session fetch failed", "peer", fqdn, "err", err)
    continue
}
```

### WR-02: Potential nil pointer dereference if entryLocal has nil session (view.go)

**File:** `view.go:168`
**Issue:** `m.renderSessionRow(*entry.session, i)` dereferences `entry.session` without a nil check. The invariant that `entry.session != nil` when `entry.kind == entryLocal` is maintained today (only one construction site in `rebuildUnifiedList` at `update.go:370`), but the `listEntry` struct has no enforcement mechanism -- it relies purely on convention. If a future code path creates an `entryLocal` without setting `session`, this line will panic at runtime. The same pattern exists at `view.go:170` for `entry.remote` on `entryRemote`. In contrast, `handleMainKey` in `update.go:263-264` correctly performs `entry.session == nil` checks before using the value.

**Fix:** Add a defensive nil guard in the render loop, consistent with what `handleMainKey` does:
```go
case entryLocal:
    if entry.session != nil {
        rows = append(rows, m.renderSessionRow(*entry.session, i))
    }
case entryRemote:
    if entry.remote != nil {
        rows = append(rows, m.renderRemoteSessionRow(entry.remote, i))
    }
```

## Info

### IN-01: QR overlay width calculation uses raw byte length for URL (qr.go)

**File:** `qr.go:28`
**Issue:** `urlLen := len(m.qrURL)` uses byte length rather than display width. For URLs this is almost always equivalent since URLs are ASCII, but the rest of the file consistently uses `lipgloss.Width()` for measuring display widths (e.g., line 22 for QR lines, line 75 for the hint). Using `lipgloss.Width(m.qrURL)` would be more consistent with the surrounding code.

**Fix:**
```go
urlLen := lipgloss.Width(m.qrURL)
```

### IN-02: Unused return from `newModel` call in test helper triggers `pty.DetectCLIs()` (update_test.go)

**File:** `update_test.go:14`
**Issue:** `testModel()` calls `newModel(nil, nil)` which in turn calls `pty.DetectCLIs()` at `tui.go:27`. This performs actual filesystem lookups (searching PATH for CLI binaries) on every test invocation. While not a correctness issue, it adds unnecessary I/O to every test. The detected CLIs are then overwritten by tests that care about them (e.g., `TestModal_AgentCycle` sets `m.detectedCLIs` explicitly).

**Fix:** Override `detectedCLIs` in `testModel()` to avoid filesystem probing in tests:
```go
func testModel() Model {
    m := newModel(nil, nil)
    m.width = 120
    m.height = 24
    m.hasDark = true
    m.styles = newStyles(true)
    m.loading = false
    m.detectedCLIs = nil // avoid filesystem probing in unit tests
    return m
}
```

---

_Reviewed: 2026-04-15T14:22:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
