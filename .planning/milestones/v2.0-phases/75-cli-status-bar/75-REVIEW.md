---
phase: 75-cli-status-bar
reviewed: 2026-04-14T18:30:00Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - cmd_attach.go
  - cmd_attach_test.go
  - internal/relay/hub.go
  - internal/relay/hub_test.go
  - internal/relay/protocol.go
  - internal/relay/protocol_test.go
  - internal/relay/server.go
  - internal/relay/server_test.go
  - internal/statusbar/bar.go
  - internal/statusbar/bar_test.go
  - internal/webserver/server.go
  - internal/webserver/server_test.go
findings:
  critical: 0
  warning: 4
  info: 4
  total: 8
status: issues_found
---

# Phase 75: Code Review Report

**Reviewed:** 2026-04-14T18:30:00Z
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

Phase 75 adds a CLI status bar with DECSTBM scroll regions, MsgMeta protocol frames for server-to-client metadata push, viewer count broadcasting, and connection state tracking. The implementation is well-structured with good separation of concerns (protocol framing in relay/protocol.go, hub fan-out in hub.go, ANSI rendering in statusbar/bar.go, and integration in cmd_attach.go). Terminal injection is properly handled via the sanitize function, and the non-blocking broadcast pattern correctly prevents slow subscribers from blocking the PTY drain loop.

Four warnings were identified: a silently suppressed fetch error that produces misleading diagnostics, discarded pump errors in attachSession, a stale reference to a future phase for origin checking, and unnecessary loop variable captures. Four informational items cover dead code in tests, a redundant struct wrapper, and cleanup opportunities.

## Warnings

### WR-01: Silently suppressed fetchErr masks real network failures

**File:** `cmd_attach.go:225`
**Issue:** When `fetchPeerSessions` or `fetchPeerSessionsWithClient` returns an error (network timeout, DNS failure, TLS handshake error), the error is silently discarded with `_ = fetchErr`. The code then iterates an empty slice, finds no matching session, and returns `"attach: session %q not found on remote host %q"`. This misleads the user into thinking the session does not exist when the real problem is a network failure. Per CLAUDE.md: "Silent Fallbacks: `or {}` converts hard failures (informative) into silent corruption (expensive). Let it crash."

**Fix:** Return the fetch error if non-nil before searching sessions:
```go
if fetchErr != nil {
    return fmt.Errorf("attach: fetch sessions from %s: %w", hostname, fetchErr)
}
```

### WR-02: attachSession always returns nil, discarding pump errors

**File:** `cmd_attach.go:361-368`
**Issue:** The `select` block receives from `stdinDone` or `wsDone` channels, each carrying a `result{err error}`, but the error value is never read. The function unconditionally returns `nil`. If the WebSocket connection drops with an error (e.g., server shutdown, TLS error), the caller `cmdAttach` reports a clean detach instead of the actual failure. For a clean detach (user pressed the detach key), returning nil is correct -- but for connection loss it should propagate the error.

**Fix:** Read the result and return the error for non-detach cases:
```go
var err error
select {
case r := <-stdinDone:
    err = r.err
case r := <-wsDone:
    err = r.err
case <-ctx.Done():
    err = ctx.Err()
}
conn.Close(websocket.StatusNormalClosure, "detach")
return err
```

### WR-03: InsecureSkipVerify on WebSocket origin check with stale TODO

**File:** `internal/relay/server.go:60-62`
**Issue:** The `websocket.AcceptOptions` sets `InsecureSkipVerify: true` with the comment "Phase 4 will add proper CORS/origin policy." The project is at Phase 75, so this phase reference is significantly stale. While the relay server binds to `127.0.0.1` (local only), the comment suggests this was intended to be temporary. For local-only relay this is low risk, but the misleading TODO should be updated to either document why origin checking is permanently unnecessary for the local relay, or to reference the actual phase/issue where it will be addressed.

**Fix:** Update the comment to reflect the current design decision:
```go
// InsecureSkipVerify: relay server is bound to 127.0.0.1 only.
// Origin checking is not required for localhost-only connections.
InsecureSkipVerify: true,
```

### WR-04: draw() writes DECSTBM inside b.mu but bar content outside b.mu

**File:** `internal/statusbar/bar.go:149-177`
**Issue:** In `draw()`, the DECSTBM scroll region escape (lines 155-157) is written while holding `b.mu`, but the actual bar content writes (lines 173-177 -- cursorSave, moveCursor, eraseLine, text, cursorRestore) happen after releasing `b.mu`. Since `b.w` is a `lockedWriter`, individual writes are atomic, but a PTY output write could insert between the DECSTBM re-issue and the bar redraw, causing a brief visual glitch where content appears in the wrong scroll region. This is a cosmetic-only race, not a data corruption risk, but it could be confusing during a terminal resize event.

**Fix:** Move the DECSTBM re-issue outside the lock (alongside the other writes), or consolidate all writes into a single buffer then flush atomically:
```go
b.mu.Lock()
needResize := cols != b.cols || rows != b.rows
if needResize {
    b.cols = cols
    b.rows = rows
}
viewerCount := b.viewerCount
connState := b.connState
localCols := b.cols
localRows := b.rows
b.mu.Unlock()

if needResize {
    // Re-issue DECSTBM outside the lock, in sequence with bar content.
    if b.opts.Position == Top {
        fmt.Fprintf(b.w, setScrollRegion, 2, rows)
    } else {
        fmt.Fprintf(b.w, setScrollRegion, 1, rows-1)
    }
}
// ... rest of bar draw writes follow immediately ...
```

## Info

### IN-01: Unnecessary loop variable capture (Go 1.22+ semantics)

**File:** `cmd_attach.go:96`, `cmd_attach.go:229`
**Issue:** The `s := s` pattern to capture the loop variable is unnecessary since Go 1.22 (project uses Go 1.26.1 per go.mod). The loop variable is now per-iteration by default.

**Fix:** Remove the `s := s` lines at both locations.

### IN-02: Dead variables in TestHubWriteInput

**File:** `internal/relay/hub_test.go:199-231`
**Issue:** `TestHubWriteInput` creates three hubs (`hub`, `hub2`, `hub3`) and multiple pipe pairs. Only `hub3` is actually used for the test assertion. `hub` and `hub2` along with their associated resources (`r`, `w`, `rIn`, `wIn`) are dead code, suppressed with blank identifier assignments at lines 228-230. This makes the test harder to understand and wastes resources.

**Fix:** Remove the unused `hub`, `hub2`, and their associated pipe pairs. Only create `hub3` with `pr`/`pw`.

### IN-03: Unused result struct wrapper in attachSession

**File:** `cmd_attach.go:349`
**Issue:** `type result struct{ err error }` wraps a single error field but the struct is never destructured (the `err` field is never accessed). The channels could use `chan error` directly, which would be simpler.

**Fix:** Change channels to `chan error` and send the error directly:
```go
stdinDone := make(chan error, 1)
wsDone := make(chan error, 1)
go func() { stdinDone <- stdinPump(ctx, conn, stdin, detachKey) }()
go func() { wsDone <- wsOutputPump(ctx, conn, stdout, bar, onFrame) }()
```

### IN-04: printDetachMessage called unconditionally after attachSession

**File:** `cmd_attach.go:181`, `cmd_attach.go:329`
**Issue:** `printDetachMessage(os.Stderr)` is called after `attachSession` returns regardless of how the session ended (clean detach, error, or context cancellation). If the session ended due to a WebSocket error or signal, printing "Detached." is slightly misleading -- the session was not voluntarily detached. This is cosmetic and does not affect correctness.

**Fix:** Conditionally print based on the return value of `attachSession` (once WR-02 is addressed and errors are propagated):
```go
err = attachSession(ctx, conn, os.Stdin, stdout, detachKey, bar, onFrame)
if err == nil {
    printDetachMessage(os.Stderr)
}
return err
```

---

_Reviewed: 2026-04-14T18:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
