---
phase: 01-pty-foundation
verified: 2026-03-17T12:00:00Z
status: passed
score: 10/10 must-haves verified
re_verification: false
---

# Phase 1: PTY Foundation Verification Report

**Phase Goal:** Spawn and manage real PTY sessions with clean lifecycle management
**Verified:** 2026-03-17
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth | Status | Evidence |
|----|-------|--------|----------|
| 1  | DetectCLIs() finds CLI binaries that exist on PATH and returns their resolved paths | VERIFIED | `detect.go:37-43` uses `exec.LookPath`; `TestDetectCLIs_FindsInstalledCLIs` PASS |
| 2  | DetectCLIs() returns empty list when no known CLIs are installed | VERIFIED | `detect.go:35` initializes with `make([]DetectedCLI, 0)`; `TestDetectCLIs_AllMissing` PASS |
| 3  | SessionRegistry can add, get, list, and remove sessions concurrently without data races | VERIFIED | `registry.go` uses `sync.RWMutex`; `TestRegistry_ConcurrentAccess` PASS with `-race` |
| 4  | Sessions persist in the registry independent of any UI lifecycle | VERIFIED | `TestRegistry_SessionPersistsAfterSimulatedWindowClose` PASS — cancel() does not remove session |
| 5  | Spawning a CLI via the PTY backend creates a real PTY (isatty passes on the child side) | VERIFIED | `native.go:35` calls `gopty.New()`; `TestSpawnPTY_RealPTY` writes to PTY and reads back echo |
| 6  | PTY I/O works: writing to the session sends input, reading from the session returns output | VERIFIED | `session.go:57-77` delegates Read/Write to `gopty.Pty`; `TestSpawnPTY_RealPTY` PASS |
| 7  | Resizing a session propagates new dimensions to the child process without error | VERIFIED | `native.go:98-110` calls `p.Resize(cols, rows)`; `TestResize` PASS |
| 8  | Killing a session terminates the entire process group — no orphan processes remain | VERIFIED | `cleanup.go:31` sends `SIGHUP` to `-pgid`; `TestKillClean_NoOrphans` PASS |
| 9  | Graceful shutdown kills all active sessions on SIGINT/SIGTERM | VERIFIED | `cmd/agenthub/main.go:50-92` uses `signal.NotifyContext` + `backend.Kill`; binary compiles and logic present |
| 10 | win32-input-mode sequences are correctly parsed into raw bytes (Windows) | VERIFIED | `win32input_parse.go` implements `parseWin32Chunk`; all 5 `TestWin32Input_*` tests PASS |

**Score:** 10/10 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` | Module definition with go-pty v0.2.2 | VERIFIED | `github.com/aymanbagabas/go-pty v0.2.2` present at line 6 |
| `internal/pty/backend.go` | SessionBackend interface contract | VERIFIED | Exports `SessionBackend`, `CreateRequest`, `ErrSessionNotFound`; 36 lines |
| `internal/pty/session.go` | Session struct with ID, state, PTY handle | VERIFIED | Exports `Session`, `SessionState`; Read/Write/String methods; 83 lines |
| `internal/pty/detect.go` | CLI detection via exec.LookPath | VERIFIED | Exports `DetectCLIs`, `DetectedCLI`, `CLISpec`, `DetectCLI`; 68 lines |
| `internal/pty/registry.go` | Thread-safe in-memory session registry | VERIFIED | Exports `SessionRegistry`; Add/Get/List/Remove/Len/KillAll; sync.RWMutex; 77 lines |
| `internal/pty/detect_test.go` | Unit tests for CLI detection | VERIFIED | 4 tests; 87 lines (min 30 required) |
| `internal/pty/registry_test.go` | Unit tests for session registry | VERIFIED | 6 tests; 123 lines (min 40 required) |
| `internal/pty/native.go` | NativePTYBackend implementing SessionBackend | VERIFIED | Exports `NativePTYBackend`, `NewNativePTYBackend`; Create/Resize/Kill/List; 177 lines (min 80 required) |
| `internal/pty/cleanup.go` | POSIX process group kill (build tag: !windows) | VERIFIED | `//go:build !windows`; `syscall.Kill(-pgid` at line 31; 66 lines |
| `internal/pty/cleanup_windows.go` | Windows Job Object cleanup (build tag: windows) | VERIFIED | `//go:build windows`; `jobObjectLimitKillOnJobClose = 0x00002000` present; 104 lines |
| `internal/pty/win32input.go` | win32-input-mode parser (build tag: windows) | VERIFIED | `//go:build windows`; exports `ParseWin32Input` |
| `internal/pty/win32input_other.go` | No-op stub for non-Windows | VERIFIED | `//go:build !windows`; `io.Copy` passthrough |
| `internal/pty/win32input_parse.go` | Stateless parser, no build tag | VERIFIED | No build tag; `parseWin32Chunk`, `findCSIEnd`, `decodeWin32Key`; 163 lines |
| `internal/pty/native_test.go` | Integration tests for PTY spawn, resize, kill | VERIFIED | 9 tests; 254 lines (min 80 required) |
| `internal/pty/win32input_test.go` | Unit tests for win32-input-mode parser | VERIFIED | 5 tests; 75 lines (min 40 required) |
| `cmd/agenthub/main.go` | Smoke-test binary | VERIFIED | detect + spawn + I/O relay + clean shutdown; 117 lines (min 40 required) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/pty/registry.go` | `internal/pty/session.go` | Registry stores Session structs | VERIFIED | `registry.go:9` — `sessions map[string]*Session` |
| `internal/pty/backend.go` | `internal/pty/session.go` | Backend returns Session from Create | VERIFIED | `backend.go:26` — `Create(…) (*Session, error)` |
| `internal/pty/native.go` | `internal/pty/backend.go` | NativePTYBackend implements SessionBackend | VERIFIED | `native.go:34,98,114,129` — all four interface methods present with correct signatures |
| `internal/pty/native.go` | `internal/pty/registry.go` | Backend stores sessions in registry | VERIFIED | `native.go:93` — `b.registry.Add(sess)` |
| `internal/pty/native.go` | `internal/pty/cleanup.go` | Kill delegates to platform-specific cleanup | VERIFIED | `native.go:125` — `return killSession(sess)` |
| `cmd/agenthub/main.go` | `internal/pty/native.go` | main creates NativePTYBackend and calls Create | VERIFIED | `main.go:41,54` — `pty.NewNativePTYBackend()` and `backend.Create(…)` |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| CLI-01 | 01-01 | App detects installed AI coding CLIs via PATH | SATISFIED | `detect.go` + 4 passing tests; `TestDetectCLIs_FindsInstalledCLIs` PASS |
| CLI-02 | 01-02 | User can launch a new session by selecting from detected CLIs | SATISFIED | `native.go:34-95` `Create()` accepts CLI name from `CreateRequest.CLI`; `cmd/agenthub/main.go` demonstrates end-to-end |
| TERM-06 | 01-02 | Terminal resizes correctly when window is resized (SIGWINCH propagation) | SATISFIED | `native.go:98-110` calls `p.Resize(cols, rows)`; `TestResize` PASS |
| TERM-07 | 01-02 | User can close/kill a session cleanly (process group cleanup) | SATISFIED | `cleanup.go` SIGHUP + SIGKILL to `-pgid`; `TestKillClean_NoOrphans` PASS |
| SESS-01 | 01-01 | Sessions persist when the app window is closed (Go-native PTY backend) | SATISFIED | Registry is independent of context lifecycle; `TestRegistry_SessionPersistsAfterSimulatedWindowClose` PASS |

**Coverage note:** REQUIREMENTS.md traceability table maps exactly CLI-01, CLI-02, TERM-06, TERM-07, SESS-01 to Phase 1. No orphaned requirements. All 5 requirements accounted for and satisfied.

### Anti-Patterns Found

None. Grep across all `internal/pty` files found zero TODO/FIXME/HACK/PLACEHOLDER comments. No stub return values (`return null`, `return {}`, `return []`) present. All handlers contain real logic.

### Human Verification Required

None required for this phase. All behaviors are verifiable programmatically:
- PTY spawn and I/O echo are covered by integration tests
- Process group kill is verified by signal-0 probe in `TestKillClean_NoOrphans`
- win32-input parsing is fully unit-tested on all platforms via untagged test file

### Commit Verification

All 6 task commits from SUMMARY.md are present in git log:

| Commit | Plan | Description |
|--------|------|-------------|
| `3a8817a` | 01-01 | feat(01-01): initialize Go module and define PTY interface contracts |
| `0b167ec` | 01-01 | feat(01-01): implement CLI detection with tests |
| `e6bcc1d` | 01-01 | feat(01-01): implement thread-safe session registry with tests |
| `9e63ab3` | 01-02 | feat(01-02): implement NativePTYBackend with spawn, resize, and I/O |
| `197f3c2` | 01-02 | feat(01-02): implement clean kill, graceful shutdown, and win32-input parser |
| `d958d42` | 01-02 | feat(01-02): add smoke-test binary and fix cross-compilation for all 3 tasks |

### Test Run Results

Full test suite run against actual codebase:

```
go test ./internal/pty/... -v -race -timeout 60s
```

24 tests, 24 PASS, 0 FAIL. Race detector clean.

Cross-compilation:
- `GOOS=linux go build ./internal/pty/...` — OK
- `GOOS=windows go build ./internal/pty/...` — OK
- `go build ./cmd/agenthub/` — OK

---

_Verified: 2026-03-17_
_Verifier: Claude (gsd-verifier)_
