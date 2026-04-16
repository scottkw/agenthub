---
phase: 75-cli-status-bar
plan: "03"
subsystem: cmd_attach
tags: [go, cli, statusbar, lockedWriter, tty, msgmeta, websocket]
one_liner: "lockedWriter + bar wiring in both local/remote attach paths with MsgMeta viewer-count intercept, TTY detection, --status-top flag, and connection state watcher"
dependency_graph:
  requires: [75-01, 75-02]
  provides:
    - lockedWriter (serialized stdout for bar + PTY)
    - --status-top flag
    - Bar instantiation in local attach (cmdAttach)
    - Bar instantiation in remote attach (cmdAttachRemoteWithClient)
    - MsgMeta intercept in wsOutputPump -> bar.SetViewerCount
    - connection state watcher goroutine (remote only)
  affects:
    - cmd_attach.go
    - cmd_attach_test.go
    - internal/webserver/server_test.go (Rule 1 bug fix)
tech_stack:
  added: []
  patterns:
    - lockedWriter mutex wrapping io.Writer for concurrent PTY + bar writes
    - bar created in TTY path, printAttachBanner in non-TTY path (SB-03)
    - onFrame callback threaded through wsOutputPump for liveness tracking
    - connection state watcher goroutine with time.Ticker and sigCtx.Done() exit
key_files:
  created: []
  modified:
    - cmd_attach.go
    - cmd_attach_test.go
    - internal/webserver/server_test.go
decisions:
  - "lockedWriter placed in cmd_attach.go (not statusbar package): bar.go comment says 'writes serialized by caller (lockedWriter in cmd_attach.go)' — ownership is explicit"
  - "printAttachBanner moved to non-TTY else branch: bar shows same info persistently in TTY, banner is redundant and would corrupt display"
  - "onFrame=nil for local attach: no connection state tracking needed for local WebSocket (always 127.0.0.1)"
  - "FallbackCols/FallbackRows left as zero in production: these are test-only fields, production always has a real TTY"
metrics:
  duration_minutes: 12
  completed_date: "2026-04-15"
  tasks_completed: 3
  tasks_total: 3
  files_created: 0
  files_modified: 3
---

# Phase 75 Plan 03: CLI Attach Bar Integration Summary

lockedWriter + bar wiring in both local/remote attach paths with MsgMeta viewer-count intercept, TTY detection, --status-top flag, and connection state watcher.

## What Was Built

Wired the `internal/statusbar` package (Plan 02) and MsgMeta protocol (Plan 01) into the CLI attach command, completing the full status bar feature.

**lockedWriter (`cmd_attach.go`):**
- `type lockedWriter struct` with `mu sync.Mutex` and `w io.Writer`
- All `attachSession` and `wsOutputPump` calls pass `stdout` (`*lockedWriter`) instead of `os.Stdout` directly
- Prevents interleaving of PTY output bytes and bar ANSI draw sequences

**Local attach path (`cmdAttach`):**
- `--status-top` flag parsed and threaded through to bar `Position`
- `statusTop bool` threaded through `cmdAttachRemote` and `cmdAttachRemoteWithClient` signatures
- TTY check: `term.IsTerminal(int(os.Stdout.Fd()))` guards bar creation (SB-03)
- In TTY path: `statusbar.New` with session metadata, `bar.Start()`, `defer bar.Stop()`
- In non-TTY path: `printAttachBanner` on `os.Stderr` (unchanged banner behavior)
- `CreatedAt` parsed from `session.CreatedAt` RFC3339 string; falls back to `time.Now()` if zero
- `attachSession` called with `stdout` (lockedWriter) and `bar, nil` (no onFrame for local)

**Remote attach path (`cmdAttachRemoteWithClient`):**
- Same TTY/bar/banner split as local
- `CreatedAt: time.Now()` — `CLIRemoteSession` has no creation timestamp
- Connection state watcher goroutine (SB-05): 1-second ticker checks `time.Since(lastFrame)`, calls `bar.SetConnectionState("reconnecting")` after 5s silence, exits on `sigCtx.Done()`
- `onFrame` callback resets `lastFrame` and clears connection state on every received frame

**wsOutputPump (`cmd_attach.go`):**
- `onFrame()` called on every received frame before frame type dispatch
- `switch` on `msgType`: `MsgOutput` writes to `w` (existing behavior preserved), `MsgMeta` unmarshals JSON and calls `bar.SetViewerCount` (SB-04)
- Unknown frame types silently ignored (no change from prior behavior)

**Tests (`cmd_attach_test.go`):**
- `TestWsOutputPump_MsgMeta`: verifies `relay.ParseFrame` + `json.Unmarshal` round-trip for MsgMeta payload; `bar.SetViewerCount` on non-started bar does not panic
- `TestLockedWriter_ConcurrentWrites`: 100 goroutines write concurrently, no line interleaving
- `TestWsOutputPump_IgnoresUnknownFrameTypes`: unknown frame type 0xFF does not match `MsgOutput` or `MsgMeta`
- All existing `attachSession` call sites updated to new `(ctx, conn, stdin, stdout, detachKey, bar, onFrame)` signature with `nil, nil`
- `cmdAttachRemoteWithClient` call site updated with new `statusTop bool` parameter

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Wire bar into local and remote attach | b950b8d | cmd_attach.go |
| 2+3 | Add tests + update existing signatures | 48d344b | cmd_attach_test.go, internal/webserver/server_test.go |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed TestWebServerWSS failing due to MsgMeta frame before MsgOutput**
- **Found during:** Task 3 (full test suite run)
- **Issue:** `TestWebServerWSS` in `internal/webserver/server_test.go` read the first WebSocket frame expecting `MsgOutput`, but `relay.NotifyViewerCount` (added in Plan 01) now sends a `MsgMeta` frame immediately on subscribe. The test was not updated when Plan 01 fixed the same issue in `relay/server_test.go`.
- **Fix:** Replaced single `conn.Read` with a loop that skips frames where `msg[0] == relay.MsgMeta`, identical in spirit to the `readDataFrame` fix applied to relay tests in Plan 01.
- **Files modified:** `internal/webserver/server_test.go`
- **Commit:** 48d344b

## Threat Model Coverage

| Threat ID | Mitigation | Verified By |
|-----------|-----------|-------------|
| T-75-05 | MsgMeta payload JSON unmarshal failure silently ignored | TestWsOutputPump_MsgMeta (no panic on bad payload) |
| T-75-06 | lockedWriter wraps os.Stdout only — no cross-user boundary | TestLockedWriter_ConcurrentWrites |
| T-75-07 | Connection state watcher goroutine exits on sigCtx.Done() | Code review — 1s ticker, context-gated exit |

## Verification Results

```
ok  github.com/scottkw/agenthub                    25.115s
ok  github.com/scottkw/agenthub/internal/daemon    6.129s
ok  github.com/scottkw/agenthub/internal/pty       0.534s
ok  github.com/scottkw/agenthub/internal/relay     0.719s
ok  github.com/scottkw/agenthub/internal/status    0.094s
ok  github.com/scottkw/agenthub/internal/statusbar 4.517s
ok  github.com/scottkw/agenthub/internal/tailnet   5.176s
ok  github.com/scottkw/agenthub/internal/updater   0.022s
ok  github.com/scottkw/agenthub/internal/webserver 0.126s
go build ./...  [exit 0]
go vet ./...    [exit 0]
```

All packages pass. Full project builds and vets clean.

## Known Stubs

None. All status bar fields are wired to live data:
- `SessionName` / `AgentType` / `Hostname` from session metadata
- `CreatedAt` from RFC3339 parse or `time.Now()` fallback
- `ViewerCount` updated live via MsgMeta push from relay
- Connection state updated via onFrame callback and watcher goroutine

## Self-Check: PASSED

- [x] cmd_attach.go: `type lockedWriter struct` present
- [x] cmd_attach.go: `statusTop := false` and `--status-top` flag present
- [x] cmd_attach.go: `term.IsTerminal(int(os.Stdout.Fd()))` TTY guard present
- [x] cmd_attach.go: `statusbar.New` called with session metadata
- [x] cmd_attach.go: `defer bar.Stop()` after `bar.Start()`
- [x] cmd_attach.go: `wsOutputPump` handles `MsgMeta` with `bar.SetViewerCount`
- [x] cmd_attach.go: `wsOutputPump` handles `MsgOutput` (existing behavior preserved)
- [x] cmd_attach.go: connection state watcher goroutine in remote path
- [x] cmd_attach_test.go: `TestWsOutputPump_MsgMeta` present
- [x] cmd_attach_test.go: `TestLockedWriter_ConcurrentWrites` present
- [x] cmd_attach_test.go: `TestWsOutputPump_IgnoresUnknownFrameTypes` present
- [x] internal/webserver/server_test.go: MsgMeta skip loop present
- [x] Commit b950b8d: FOUND in git log
- [x] Commit 48d344b: FOUND in git log
- [x] All tests pass: CONFIRMED
