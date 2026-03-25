---
phase: 25-windows-named-pipe-fix
plan: 01
subsystem: daemon/socket
tags: [windows, named-pipe, bug-fix, build-tags, cross-platform]
dependency_graph:
  requires: [internal/daemon/socket.go, github.com/tailscale/go-winio]
  provides: [cleanupStaleWindowsPipe, isWindowsNamedPipe]
  affects: [CleanupStaleSocket, EnsureDaemon]
tech_stack:
  added: []
  patterns: [build-tagged platform files, winio.DialPipe for Windows named pipes]
key_files:
  created:
    - internal/daemon/socket_windows.go
    - internal/daemon/socket_nonwindows.go
    - internal/daemon/socket_windows_test.go
  modified:
    - internal/daemon/socket.go
    - internal/daemon/socket_test.go
decisions:
  - "Return nil (not error) for any dial failure on named pipes — pipes vanish when last server handle closes, so no os.Remove needed"
  - "Stub on non-Windows panics rather than returns nil — unreachable code should fail loudly if somehow reached"
metrics:
  duration: "~10 minutes"
  completed: "2026-03-24"
  tasks: 2
  files_created: 3
  files_modified: 2
---

# Phase 25 Plan 01: Windows Named Pipe Dial Fix Summary

**One-liner:** Platform-aware CleanupStaleSocket that uses winio.DialPipe for Windows named pipe paths (`\\` prefix) and the existing net.DialTimeout unix path for all other platforms, with build-tagged implementation files following the process_windows.go/process_unix.go pattern.

## What Was Built

`CleanupStaleSocket` previously used `net.DialTimeout("unix", ...)` unconditionally, which always fails for Windows named pipe paths like `\\.\pipe\agenthub-daemon`, causing every probe to report "stale" and triggering redundant daemon spawns.

The fix adds an `isWindowsNamedPipe` helper that detects `\\`-prefixed paths, and dispatches to a build-tagged `cleanupStaleWindowsPipe` function in `socket_windows.go` that uses `winio.DialPipe`. Since named pipes are kernel objects (not filesystem entries), there is no `os.Remove` cleanup needed — if no server is present, the pipe simply doesn't exist and the function returns nil.

## Tasks Completed

| Task | Description | Commit |
|------|-------------|--------|
| 1 | isWindowsNamedPipe helper + build-tagged implementations | 3ff2b5a |
| 2 | Unit tests: isWindowsNamedPipe + Windows pipe test stubs | a7b841b |

## Key Files

**Created:**
- `internal/daemon/socket_windows.go` — Windows-only `cleanupStaleWindowsPipe` using `winio.DialPipe`
- `internal/daemon/socket_nonwindows.go` — Non-Windows stub (panics if unreachably called)
- `internal/daemon/socket_windows_test.go` — Windows-only tests for absent/active pipe scenarios (auto-skipped on macOS/Linux via build tag)

**Modified:**
- `internal/daemon/socket.go` — Added `isWindowsNamedPipe` helper and dispatch in `CleanupStaleSocket`; added `"strings"` import
- `internal/daemon/socket_test.go` — Added `TestIsWindowsNamedPipe` with 5 cases

## Decisions Made

1. **Return nil for all dial failures on named pipes** — When a named pipe is not present (no server), DialPipe returns an error. Returning nil is correct because there is nothing to clean up and a fresh daemon start should be allowed.

2. **Non-Windows stub panics, not returns nil** — The stub is unreachable because `isWindowsNamedPipe` only returns true for `\\`-prefixed paths which never appear on Unix. A panic makes accidental invocation visible.

## Deviations from Plan

None — plan executed exactly as written.

## Verification Results

- `go build ./internal/daemon/...` — exits 0 on macOS
- `go test ./internal/daemon/ -count=1` — full suite green (all tests pass)
- `go test ./internal/daemon/ -run TestIsWindowsNamedPipe -count=1 -v` — PASS (5 cases)
- `go test ./internal/daemon/ -run TestCleanupStale -count=1 -v` — 3/3 existing Unix tests PASS
- `grep -n "isWindowsNamedPipe" internal/daemon/socket.go` — dispatch present on line 58
- `grep -n "winio.DialPipe" internal/daemon/socket_windows.go` — correct Windows API used
- `grep -rn "os.Remove" internal/daemon/socket_windows.go` — only in comment, not as a call

## Self-Check: PASSED

Files created/exist:
- internal/daemon/socket_windows.go: FOUND
- internal/daemon/socket_nonwindows.go: FOUND
- internal/daemon/socket_windows_test.go: FOUND

Commits:
- 3ff2b5a: FOUND (feat(25-01): add isWindowsNamedPipe helper...)
- a7b841b: FOUND (test(25-01): add isWindowsNamedPipe unit test...)
