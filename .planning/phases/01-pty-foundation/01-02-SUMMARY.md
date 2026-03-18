---
phase: 01-pty-foundation
plan: "02"
subsystem: pty
tags: [go, go-pty, pty, native-backend, kill, process-group, win32-input, cleanup, windows-job-object]

# Dependency graph
requires:
  - 01-01 (SessionBackend interface, Session struct, SessionRegistry, go-pty v0.2.2)
provides:
  - NativePTYBackend implementing SessionBackend (Create/Resize/Kill/List)
  - POSIX process group kill via killSession (SIGHUP + SIGKILL with 2s timeout)
  - Windows Job Object cleanup (JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE)
  - win32-input-mode CSI parser (parseWin32Chunk, ParseWin32Input)
  - cmd/agenthub smoke-test binary demonstrating full PTY lifecycle
affects:
  - Phase 2 (WebSocket relay will consume Session.Read/Write)
  - Phase 3 (Electron shell will call backend.Create/Kill)
  - All subsequent phases (NativePTYBackend is the production implementation)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - PTY opened before cmd.Start; Close before cmd.Wait prevents Wait deadlock
    - go-pty sets Setsid:true (new session = new process group); Setpgid must NOT be combined
    - Platform split via job_windows.go/job_other.go keeps native.go build-tag-free
    - session.job (any) holds Windows Job Object handle without forcing build tags on Session struct
    - parseWin32Chunk stateless; ParseWin32Input accumulates pending across reads
    - Test helpers in kill_posix_test.go/kill_windows_test.go use _test.go suffix to stay out of production binary

key-files:
  created:
    - internal/pty/native.go
    - internal/pty/native_test.go
    - internal/pty/cleanup.go
    - internal/pty/cleanup_windows.go
    - internal/pty/job_other.go
    - internal/pty/job_windows.go
    - internal/pty/win32input_parse.go
    - internal/pty/win32input.go
    - internal/pty/win32input_other.go
    - internal/pty/win32input_test.go
    - internal/pty/kill_posix_test.go
    - internal/pty/kill_windows_test.go
    - cmd/agenthub/main.go
  modified:
    - internal/pty/session.go (added job any field)

key-decisions:
  - "Do NOT combine Setpgid:true with go-pty — it sets Setsid:true internally; combining causes EPERM on macOS"
  - "Close PTY master before cmd.Wait to prevent indefinite block after child exits (PTY slave still open)"
  - "parseWin32Chunk is in win32input_parse.go with no build tag so unit tests run on all platforms"
  - "Session.job stored as any — avoids build-tag spread across session.go while keeping type assertion in cleanup_windows.go"

requirements-completed:
  - CLI-02
  - TERM-06
  - TERM-07

# Metrics
duration: 10min
completed: "2026-03-18"
---

# Phase 1 Plan 2: NativePTYBackend — Spawn, Resize, Kill, win32-input Parser Summary

**NativePTYBackend with real PTY spawn (isatty passes), TERM=xterm-256color injection, POSIX process-group kill, Windows Job Object cleanup, win32-input-mode CSI parser — 24 tests pass with race detector, cross-compiles for linux and windows**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-03-18T00:11:47Z
- **Completed:** 2026-03-18T00:21:30Z
- **Tasks:** 3 of 3
- **Files modified:** 14

## Accomplishments

- Implemented `NativePTYBackend` — `Create` opens PTY via go-pty, spawns process, merges environment with `TERM=xterm-256color` and `COLORTERM=truecolor` always set. `Resize` delegates to `gopty.Pty.Resize`. `Kill` calls `killSession` and removes from registry.
- POSIX cleanup (`cleanup.go`): sends `SIGHUP` to the process group (`-pgid`), closes PTY master (to prevent `cmd.Wait` deadlock), waits 2 seconds, then `SIGKILL`. No orphan processes.
- Windows cleanup (`cleanup_windows.go`): `jobObject` wraps a Windows Job Object with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` — closing the handle terminates all assigned processes atomically.
- `win32input_parse.go` (no build tag): stateless `parseWin32Chunk` parser handles key-down events (emit Unicode char), key-up filtering (Kd=0, drop), literal passthrough, incomplete sequence buffering, and mixed content. `ParseWin32Input` is the platform-specific streaming wrapper.
- `cmd/agenthub/main.go`: smoke-test binary — detects CLIs, spawns `cat` demo session (or `AGENTHUB_DEMO_CLI`), relays PTY output to stdout and stdin to PTY, shuts down cleanly on SIGINT/SIGTERM.

## Task Commits

1. **Task 1: NativePTYBackend with spawn, resize, and I/O** — `9e63ab3`
2. **Task 2: Clean kill, graceful shutdown, and win32-input parser** — `197f3c2`
3. **Task 3: Smoke-test binary and full suite** — `d958d42`

## Files Created/Modified

- `internal/pty/native.go` — `NativePTYBackend`, `NewNativePTYBackend`, `Create/Resize/Kill/List`, `mergeEnv`, `generateID`
- `internal/pty/native_test.go` — 9 integration tests: spawn, real PTY echo, env vars, resize, not-found, list, kill (3)
- `internal/pty/cleanup.go` — POSIX `killSession`: SIGHUP + PTY close + 2s timeout + SIGKILL
- `internal/pty/cleanup_windows.go` — `jobObject` struct, `newJobObject`, `Assign`, `Close`, `killSession` for Windows
- `internal/pty/job_other.go` — `assignJobObject` no-op for POSIX
- `internal/pty/job_windows.go` — `assignJobObject` creates Job Object and assigns process
- `internal/pty/win32input_parse.go` — `parseWin32Chunk`, `findCSIEnd`, `decodeWin32Key` (no build tag)
- `internal/pty/win32input.go` — `ParseWin32Input` streaming wrapper (Windows only)
- `internal/pty/win32input_other.go` — `ParseWin32Input` passthrough (POSIX)
- `internal/pty/win32input_test.go` — 5 unit tests for win32-input-mode parser
- `internal/pty/kill_posix_test.go` — `killSignal0` test helper (POSIX)
- `internal/pty/kill_windows_test.go` — `killSignal0` stub (Windows)
- `internal/pty/session.go` — Added `job any` field for platform-specific cleanup handle
- `cmd/agenthub/main.go` — Smoke-test binary with detect, spawn, I/O relay, clean shutdown

## Decisions Made

- **Do not combine Setpgid with go-pty**: go-pty sets `Setsid:true` internally; adding `Setpgid:true` causes `EPERM` on macOS. Since `Setsid` already creates a new session (PGID == PID), process group kill via `-pid` works without explicit `Setpgid`.
- **Close PTY before Wait**: On POSIX, `cmd.Wait` blocks until the PTY slave is closed even after the child exits. Closing the PTY master before `Wait` is mandatory to prevent deadlock.
- **No build tag on win32input_parse.go**: The stateless chunk parser lives in an untagged file so tests run on all platforms without conditional compilation gymnastics.
- **session.job as `any`**: Avoids propagating Windows build tags into `session.go`. The Windows cleanup code does a type assertion to `*jobObject`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed Setpgid:true from SysProcAttr**
- **Found during:** Task 1 RED → GREEN
- **Issue:** Plan specified `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` but go-pty's `cmd_unix.go` already sets `Setsid:true` on the same struct. Combining both causes `fork/exec: operation not permitted` on macOS.
- **Fix:** Removed `Setpgid` entirely. go-pty's `Setsid` creates a new session where PGID == PID, so negative-PGID kill still works as intended.
- **Files modified:** `internal/pty/native.go`
- **Commit:** `9e63ab3`

**2. [Rule 1 - Bug] Close PTY master before cmd.Wait in killSession**
- **Found during:** Task 1 GREEN — `TestSpawnPTY_EnvSet` caused `killSession` to hang indefinitely
- **Issue:** `env` exits immediately after printing, but `cmd.Wait` blocks because the PTY master fd is still open. The PTY slave cannot deliver EOF to `Wait` until all master references are closed.
- **Fix:** `cleanup.go` now closes the PTY master BEFORE calling `cmd.Wait` in the goroutine.
- **Files modified:** `internal/pty/cleanup.go`
- **Commit:** `9e63ab3`

**3. [Rule 1 - Bug] Fixed Windows cross-compilation errors in cleanup_windows.go**
- **Found during:** Task 3 cross-compilation check
- **Issue 1:** `windows.SetInformationJobObject` returns `(int, error)` — assignment mismatch with single-return assumption.
- **Issue 2:** `windows.PROCESS_ALL_ACCESS` does not exist in x/sys; replaced with `PROCESS_TERMINATE | SYNCHRONIZE | PROCESS_DUP_HANDLE`.
- **Fix:** Updated function call to capture both return values; replaced constant.
- **Files modified:** `internal/pty/cleanup_windows.go`
- **Commit:** `d958d42`

**4. [Rule 2 - Missing critical functionality] Added platform-split job assignment files**
- **Found during:** Task 1 implementation
- **Issue:** Plan said to use `runtime.GOOS == "windows"` in `native.go` to call `newJobObject()`, but `newJobObject` is only defined in `cleanup_windows.go`. Non-Windows builds fail.
- **Fix:** Extracted `assignJobObject(sess, proc)` into `job_windows.go` (real) and `job_other.go` (no-op). `native.go` calls `assignJobObject` unconditionally — no build tags needed in the main file.
- **Files modified/created:** `internal/pty/job_windows.go`, `internal/pty/job_other.go`, `internal/pty/native.go`
- **Commit:** `9e63ab3`

## Issues Encountered

None remaining.

## User Setup Required

None.

## Next Phase Readiness

- Phase 2 (WebSocket relay) can call `backend.Create` and consume `Session.Read`/`Session.Write` directly
- `ParseWin32Input` is ready to be wired into the WebSocket input path on Windows
- `cmd/agenthub` binary can be used for manual smoke-testing before Phase 2

## Self-Check: PASSED

All files exist. Commits 9e63ab3, 197f3c2, d958d42 verified in git log. 24 tests pass with race detector. GOOS=linux and GOOS=windows cross-compilation succeeds.
