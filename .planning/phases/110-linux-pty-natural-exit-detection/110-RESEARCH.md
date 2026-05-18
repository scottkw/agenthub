# Phase 110: Linux PTY natural-exit detection - Research

**Researched:** 2026-05-18
**Domain:** POSIX PTY EOF semantics on Linux vs. macOS; go-pty v0.2.2 internals; Go syscall.Wait4 + WNOHANG polling; build-tag platform split in `internal/pty`.
**Confidence:** HIGH (all critical claims verified against repo source or vendored go-pty source)

## Summary

The phase fix point is the natural-exit goroutine in `internal/daemon/engine.go:328-346`. It currently waits on `hub.Done()` (relay/hub.go:209) which closes when `Hub.Run` (relay/hub.go:135-152) returns from a non-nil error on `h.reader.Read(buf)`. On macOS, `pty.master.Read()` returns `io.EOF` shortly after the child exits — the slave FD closes, the kernel signals EOF on the master, the loop exits. On Linux with go-pty v0.2.2, that read does NOT return after a clean child exit; it blocks indefinitely. The result: `Hub.Run` never returns, `hub.Done()` never closes, the natural-exit goroutine never runs, the daemon never transitions the session to `StateStopped`, the Wails `pollSessionStatus` (app.go:261-304) never sees `state == "stopped"`, the frontend `session:exit` event is never emitted, and the GUI tab + TUI list entry stay open forever.

The chosen approach (locked in CONTEXT.md): on Linux only, spawn an exit-detector goroutine alongside the natural-exit waiter. The detector polls `syscall.Wait4(pid, &status, WNOHANG)` at ~100 ms. When the child has exited, the detector caches the real exit code on the Session and explicitly closes the PTY — which closes the master FD and forces the blocked `pty.Read()` in `Hub.Run` to return an error, unblocking the existing exit-watcher chain that's already correct on macOS.

**Primary recommendation:** Add a Linux-only `pty.Session` helper that runs a `Wait4(pid, _, WNOHANG)` polling goroutine started from `Create`. On positive detection it (a) sets the cached exit code via `sess.SetExitCode(status.ExitStatus())`, (b) calls `sess.pty.Close()` to unblock `Hub.Run`. The existing natural-exit goroutine in `engine.go:328-346` then runs and emits to `onExit` exactly as on macOS — no changes to the downstream chain.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Detecting child process exit | daemon (Go, `internal/pty`) | — | OS-level wait syscall; only the daemon holds the child PID and PTY FD. |
| Unblocking the PTY read loop | daemon (Go, `internal/pty`) | daemon (`internal/relay`) | Closing the master FD is the unblock mechanism; the relay hub's `Run` reacts to the resulting `Read` error. |
| Marking session "stopped" | daemon (`internal/daemon/engine.go`) | — | State transition logic already lives here (engine.go:342); no change needed. |
| Broadcasting `session:exit` to GUI | Wails app process (`app.go`) | daemon (`pollSessionStatus`) | App polls daemon, emits Wails event — no change needed. |
| Removing TUI list entry | TUI (`internal/tui`) | daemon (`ListSessions`) | TUI polls `ListSessions` every 2s; entry disappears once `state == "stopped"` propagates — no TUI change needed. |
| Broadcasting `session:exit` (web) | webserver | daemon | Same `onExit` callback wires web grace-period cleanup (api.go:409-413); unchanged. |

The fix is surgical: a single new Linux-only file in `internal/pty`. No file outside `internal/pty` changes except the test in `internal/daemon/engine_test.go` (flip `t.Skip()` to enabled).

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PTY-01 | Linux GUI tab / TUI list entry auto-closes on clean `exit 0` — same as macOS. | Section 7 (Cross-surface trace) confirms the downstream chain is correct once daemon transitions to `state == "stopped"`. |
| PTY-02 | Platform-aware: Linux uses `syscall.Wait4(pid, _, WNOHANG)` exit-detector goroutine that closes PTY to unblock the read loop; coordinated with go-pty's wait to avoid double-`Wait`. | Section 1 confirms read-loop blocking; Section 2 confirms go-pty v0.2.2 does NOT call `Wait()` automatically on Unix (no double-wait risk from go-pty itself); Section 3 confirms Wait4 semantics; Section 6 confirms build-tag convention. |
| PTY-03 | `TestListSessions_OnExitCallback_ReceivesNormalized` runs without `t.Skip()` on Linux under `-race -shuffle=on`. | Section 5 locates the test and confirms exact `t.Skip()` line + test contract. |
| PTY-04 | Cross-surface parity: TUI Linux + CLI Linux benefit from same fix; macOS/Windows unchanged. | Section 6 (build-tag isolation), Section 7 (TUI polling is independent of the fix mechanism — it just observes the same daemon state). |

## Project Constraints (from CLAUDE.md)

- **Chesterton's Fence:** the natural-exit goroutine in `engine.go:328-346` is correct on macOS — do not refactor it. Add a Linux-only helper that produces the same `Read` error condition `Hub.Run` already expects.
- **Silent Fallbacks:** the existing `time.Sleep(100 * time.Millisecond)` at `engine.go:337` is a known timing hack to wait for go-pty's `cmd.Wait` to populate `ProcessState`. With Wait4-based detection we can cache the real exit code BEFORE closing the PTY, eliminating that race and the silent fallback to `0`.
- **Code Navigation (LSP):** prefer `LSP.findReferences` on `Session.pty`, `Hub.reader`, `pty.Read` when planning.
- **Premature Abstraction:** do NOT generalize the build-tag split to BSD / other Unix variants. The bug is specifically Linux; macOS works. A `linux` vs `!linux` split is correct.
- **Make beliefs pay rent:** verify on Linux desktop that the exit-detector actually fires before assuming success. Issue #57 is the canonical reproduction.

## 1. PTY read-loop blocking semantics on Linux vs. macOS

### Where the read loop lives

The PTY read loop is in **`internal/relay/hub.go:135-152`** — `Hub.Run`:

```go
func (h *Hub) Run() {
    defer h.Shutdown()
    buf := make([]byte, 32*1024)
    for {
        n, err := h.reader.Read(buf)  // line 140 — THE blocking call
        if n > 0 { ... }
        if err != nil {
            return  // line 149 — only way out
        }
    }
}
```

`h.reader` is the `*pty.Session` (see `engine.go:298` — `hub := e.manager.Create(id, sess, sess, resizeFn)`). `Session.Read` (`session.go:66-74`) is a thin wrapper that calls `pty.Read(p)` on the underlying go-pty `Pty`. On Unix, go-pty's `unixPty.Read` (`pty_unix.go:58-60`) is `p.master.Read(b)` — a direct read on the PTY master `*os.File`. [VERIFIED: repo source]

### What terminates the loop on macOS

When the child closes the slave (process exit), the kernel closes any references the kernel still holds on the slave once all child FDs to it close. On macOS (Darwin), once the slave-side has no remaining writers AND the kernel's process-table entry releases the slave, `read()` on the master returns 0 bytes — which Go's `*os.File.Read` translates to `io.EOF`. `Hub.Run` sees `err != nil`, returns, defers `Shutdown` (closes `done`), and the natural-exit goroutine in `engine.go:328-346` runs. [CITED: BSD/Darwin PTY behavior; macOS works in production per CONTEXT.md root cause analysis]

### Why Linux blocks indefinitely

Linux PTY master semantics differ subtly: even after the child process exits and all slave FDs in the child are closed, `read()` on the master can continue to block because the slave-side may still be considered "open" by the kernel (the slave inode is held by the master pair until the master is closed). Linux does NOT consistently surface EOF on the master when only the slave-side process exits. Various reports including go-pty issue tracking and creack/pty (the upstream PTY library go-pty uses on Unix per `pty_unix.go:11` import) document this behavior. [CITED: creack/pty issue tracker discussions on Linux EOF behavior]

The asymmetric Read-loop-blocking behavior is precisely what Issue #57 reports and what `internal/daemon/engine_test.go:1300-1307` documents:
> "On Linux, go-pty's master Read() blocks indefinitely after the child exits cleanly (instead of returning io.EOF as macOS does), so the natural-exit goroutine in engine.go never fires."

[VERIFIED: explicit comment in repo at engine_test.go:1300-1307]

### Confirmed unblock mechanism

`internal/pty/cleanup.go:37-41` already documents and uses the standard remedy on the kill path:

```go
// Close the PTY master first — otherwise cmd.Wait() may block indefinitely
// waiting for the PTY slave to be closed, even after the child has exited.
if s.pty != nil {
    _ = s.pty.Close()
}
```

go-pty's `unixPty.Close` (`pty_unix.go:25-33`) closes BOTH master and slave with `errors.Join(p.master.Close(), p.slave.Close())`. Closing the master FD causes any blocked `Read` on it to return an error (`*PathError` wrapping `EBADF` or `os.ErrClosed`). [VERIFIED: vendored go-pty source `/Users/ken/go/pkg/mod/github.com/aymanbagabas/go-pty@v0.2.2/pty_unix.go`]

**This is the unblock pattern Phase 110 will reuse on the natural-exit path.** Confidence: HIGH.

## 2. go-pty v0.2.2 exit handling — no Unix `waitOnContext`

CONTEXT.md (and the ROADMAP scope) reference coordinating with "go-pty's internal `waitOnContext`" to avoid a double-`Wait` race. **This concern is overstated for our target platform.** Verified findings:

- **`waitOnContext` exists only on Windows** (`cmd_windows.go:174,180` in both v0.2.2 and v0.2.3). On Unix there is NO automatic `Wait()` goroutine. [VERIFIED: grep across vendored module cache]
- On Unix, `Cmd.Wait()` is a thin wrapper that calls `exec.Cmd.Wait()` exactly once (`cmd_unix.go:55-70`). It guards against double-calls: "exec: Wait was already called" if `ProcessState != nil`. [VERIFIED: cmd_unix.go:60-61]
- v0.2.2 and v0.2.3 are **byte-identical on Unix** (`diff` confirms zero changes in `cmd_unix.go` and `pty_unix.go`). The Linux blocking bug is independent of the go-pty version pin. [VERIFIED: file diff]

### What this means for the plan

- **No double-`Wait` race from go-pty itself on Linux.** The only `cmd.Wait()` caller in our codebase is `killSession` (`cleanup.go:46`), guarded by `MarkKilled()`/`IsKilled()` (`session.go:55-58`, `engine.go:330-332`). The natural-exit goroutine in `engine.go` does NOT call `cmd.Wait()`.
- **However**, `syscall.Wait4(pid, ...)` is a Wait variant that DOES race with `exec.Cmd.Wait()` if both reach the kernel. The Go runtime's `os/exec` uses its own `wait4` under the hood. The standard cross-platform `os.Process.Wait()` documentation explicitly notes this. The collision scenario for us:
  1. Our Wait4 polling goroutine reaps the child first (pid > 0 from Wait4).
  2. Later, `killSession` runs (e.g., user clicks Kill after a natural exit grace window) and calls `cmd.Wait()`. That call will return `ECHILD`-ish behavior from `exec.Cmd` — or worse, hang depending on how Go's `exec` package internally tracks state.
- **Mitigation:** only run the Wait4 detector in the natural-exit path; suppress it once `IsKilled()` is true. On the kill path, `killSession` continues to use `cmd.Wait()` and we never let Wait4 fire concurrently. The detector goroutine's loop body checks `sess.IsKilled()` each tick and bails out.
- **Alternative (more bulletproof):** the Wait4 detector caches the exit code and closes the PTY, then exits. It does NOT call `cmd.Wait()`. `killSession` still calls `cmd.Wait()` only on the kill path. But on the **natural-exit** path, no second `Wait` happens. The existing `engine.go:335-337` calls `sess.CancelContext()` then sleeps 100 ms — that cancel triggers `exec.Cmd.Cancel` (set by `CommandContext`), which calls `Process.Kill()` (`cmd_unix.go:27-29`). If the process is already exited (Wait4-reaped), `Process.Kill()` returns "process already finished" (harmless). The `time.Sleep(100ms)` becomes a no-op and can stay (no churn) or be removed as a follow-up.

[VERIFIED: source inspection of vendored go-pty v0.2.2 and v0.2.3 cmd_unix.go]

### Exported API to suppress go-pty's internal waiter?

No such hook exists in go-pty v0.2.2/v0.2.3 on Unix. None is needed because no internal waiter runs on Unix.

## 3. syscall.Wait4 + WNOHANG pattern in Go

### Signature

```go
func Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (wpid int, err error)
```

Located in `syscall` (stdlib). For more reliable cross-arch behavior, `golang.org/x/sys/unix.Wait4` is preferred when extra portability is needed; for Linux-only build-tagged file the stdlib `syscall.Wait4` is sufficient. [VERIFIED: stdlib godoc]

### Return semantics with `WNOHANG`

| Return | Meaning | Action |
|--------|---------|--------|
| `wpid > 0, err == nil` | Child exited and was reaped. `wstatus` is populated. | Read `wstatus.ExitStatus()`, cache, close PTY, exit detector. |
| `wpid == 0, err == nil` | Child is still running, no status to report. | Sleep, poll again. |
| `wpid == -1, err == ECHILD` | No child to wait for — already reaped by someone else (e.g., `killSession`'s `cmd.Wait`). | Exit detector silently. |
| `wpid == -1, err == EINTR` | Syscall interrupted. | Retry. |
| `wpid == -1, other err` | Unexpected. | Log + exit detector. |

[CITED: Linux wait4(2) man page; Go stdlib syscall package; cross-confirmed via search results below]

### Extracting the exit code

```go
var status syscall.WaitStatus
wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
if wpid > 0 && err == nil {
    if status.Exited() {
        exitCode := status.ExitStatus()        // 0-255 for normal exit
    } else if status.Signaled() {
        exitCode := 128 + int(status.Signal()) // POSIX convention
    }
}
```

The repo's `cleanup.go:48-50` already uses `cmd.ProcessState.ExitCode()` which collapses both cases — we should match that normalization. `cmd.ProcessState.ExitCode()` returns `-1` if signaled. For the natural-exit path (clean exit code 0 is the dominant case), `status.ExitStatus()` is the right field. Edge case: a shell that exits via SIGKILL/SIGTERM from elsewhere is unusual for `exit` typed at a prompt, so the 128+signal path is acceptable. [VERIFIED: repo code + Go stdlib docs]

### Confidence

HIGH for Wait4 mechanics. The pattern is standard and well-documented. Sources:
- [Linux wait4(2) man page](https://man7.org/linux/man-pages/man2/wait4.2.html) [CITED]
- [Go stdlib syscall.Wait4](https://pkg.go.dev/syscall#Wait4) [CITED]
- Discussion on golang-nuts confirming WNOHANG=0 means "still running" [CITED: golang-nuts thread referenced in WebSearch results]

## 4. Cadence + jitter

### Codebase precedent

| Location | Cadence | Purpose |
|----------|---------|---------|
| `internal/daemon/engine.go:337` | one-shot 100 ms | Wait for go-pty to populate ProcessState after CancelContext. **Direct precedent.** |
| `internal/daemon/api.go:361` | one-shot 50 ms | Brief settle |
| `internal/daemon/process.go:217` | 50 ms | Process loop settle |
| `internal/daemon/process.go:179` | 200 ms | Process loop |
| `internal/daemon/process.go:32` | 5 s ticker | Long-cadence daemon process supervisor |
| `internal/tui/cmds.go:31` | 2 s ticker | TUI session list refresh (user-perceptible) |
| `internal/statusbar/bar.go:131` | 1 s ticker | Status bar UI updates |
| `app.go:302` | 500 ms | GUI `pollSessionStatus` — natural-exit detection at the Wails layer |

The closest precedents to a Wait4 polling cadence are:
1. **`app.go:302` at 500 ms** — already the upstream consumer of "session is stopped" detection. If Wait4 polls faster than this, we don't gain user-perceptible latency.
2. **`engine.go:337` at 100 ms** — already used in the natural-exit path for ProcessState settle.

### Recommendation

**100 ms ticker, no jitter.** Rationale:
- The phase's user-perceptible target is "auto-close within ~1 s" (ROADMAP success criteria #1, #2). 100 ms detection latency + 500 ms Wails poll + frame render is well under 1 s in the worst case.
- 100 ms aligns with the existing `engine.go:337` constant — single source of truth via a named const (e.g., `linuxExitPollInterval = 100 * time.Millisecond`).
- Single-process Wait4 polling at 10 Hz across all active sessions is negligible CPU (one syscall per session per 100 ms, no allocations).
- **No jitter needed.** Jitter matters when many clients hit a shared service simultaneously. We poll per-session, in-process, no thundering herd.
- The detector goroutine should also accept a `context.Context` (the session's `childCtx`) and return on context cancellation to avoid leaking on `killSession`.

Confidence: HIGH (codebase precedent + back-of-envelope CPU math).

## 5. `TestListSessions_OnExitCallback_ReceivesNormalized` — exact contract

**Location:** `internal/daemon/engine_test.go:1287-1342`

**Current state:** Skipped on Linux at lines 1300-1307:
```go
if runtime.GOOS == "linux" {
    // On Linux, go-pty's master Read() blocks indefinitely after the child
    // exits cleanly (instead of returning io.EOF as macOS does), so the
    // natural-exit goroutine in engine.go never fires. Tracked in
    // scottkw/agenthub#57; production SHELL-12 auto-close is also affected
    // on Linux until the goroutine is restructured.
    t.Skip("blocked on Linux PTY EOF behavior — see scottkw/agenthub#57")
}
```

**What it tests:**
1. Routes a fake CLI name (`"fakecli"`) through `cliPaths["fakecli"] = shPath` so it does NOT enter the `isShellSession()` branch.
2. Calls `e.CreateSession(ctx, "fakecli", "tab", "", []string{"-c", "exit 0"}, 80, 24, nil, onExit)`.
3. The `sh -c "exit 0"` exits immediately with code 0.
4. Asserts `onExit` is called with `code == 0` within 5 seconds.
5. The window proves that the natural-exit goroutine fires AND that the `-1 → 0` normalization at `engine.go:339-341` runs.

**What it needs to flip the `t.Skip()`:**
- The natural-exit goroutine must actually fire on Linux for `sh -c "exit 0"` — i.e., `Hub.Run` must return after the sh process exits.
- Equivalent: the Wait4-based detector must call `sess.pty.Close()` after observing exit, which forces `Hub.Run`'s blocked `Read` to return an error.
- No other test modifications needed. The exit code is already exit-status-0 from `sh -c "exit 0"`, and the goroutine's `-1 → 0` normalization is defensive.

**`-race -shuffle=on` considerations:**
- The detector goroutine reads `sess.pty` and calls `sess.pty.Close()` — `Session.Read` (`session.go:66-74`) already holds `s.mu` while capturing `pty` before reading. The detector must acquire the same mutex (or call through a wrapper method) before closing.
- The detector must NOT call `sess.cmd.Wait()` — race against `killSession`'s `Wait`. It calls `syscall.Wait4(pid, ...)` directly.
- `sess.SetExitCode(...)` is mutex-guarded (`session.go:147-149`). Safe.
- The 5-second timeout in the test (line 1339) is generous; 100 ms poll + ~200 ms `sh` startup + `Hub.Run` Read-error propagation should be well under 1 s.

**Other Linux skips to consider:** searching for other `runtime.GOOS == "linux"` skips in the test suite that may now be reachable:

(Out of scope for this section; the planner can grep for additional skipped tests if desired.)

## 6. Build-tag convention

### Existing patterns in the codebase

The repo uses **two parallel splits** depending on what's platform-specific:

**Pattern A: `_windows.go` vs. `_nonwindows.go`** (Phase 109, the most recent precedent):
- `internal/daemon/socket_windows.go` ↔ `internal/daemon/socket_nonwindows.go`
- `internal/daemon/ipc_windows.go` ↔ `internal/daemon/ipc_nonwindows.go`

**Pattern B: `_windows.go` vs. `_other.go`** (older convention in `internal/pty`):
- `internal/pty/cleanup_windows.go` ↔ `internal/pty/cleanup.go` (header `//go:build !windows`)
- `internal/pty/job_windows.go` ↔ `internal/pty/job_other.go`
- `internal/pty/win32input.go` ↔ `internal/pty/win32input_other.go`

**No existing `_linux.go` / `_other.go` split inside `internal/`.** The only `//go:build linux` files in this repo are top-level `tray_linux.go` / `tray_linux_test.go` (system-tray code), paired with implicit `//go:build !linux` files at the package root.

### Recommendation for Phase 110

CONTEXT.md proposes `exit_linux.go` / `exit_other.go`. This is consistent with the **existing `internal/pty` convention** (Pattern B) and is the right call. Specifically:

- `internal/pty/exit_linux.go` — `//go:build linux` — contains the Wait4-polling exit-detector goroutine. Exports a single function (e.g., `startExitDetector(sess *Session)`) called from `NativePTYBackend.Create` after `cmd.Start()`.
- `internal/pty/exit_other.go` — `//go:build !linux` — defines the same function as a no-op. macOS, BSDs, Windows all use the empty stub.

This is the surgical, minimal split. It does NOT touch `cleanup.go` (already `!windows`) and it does NOT touch `cleanup_windows.go` (Windows kill path). The build matrix becomes:

| Platform | `cleanup.go` | `cleanup_windows.go` | `exit_linux.go` | `exit_other.go` |
|----------|--------------|----------------------|-----------------|-----------------|
| linux | ✓ | — | ✓ | — |
| darwin/bsd | ✓ | — | — | ✓ |
| windows | — | ✓ | — | ✓ |

The detector calls only POSIX syscalls — but since the file is `//go:build linux`, that's hermetic.

Confidence: HIGH.

### Test build-tag

The test file is `internal/daemon/engine_test.go` — package-level, no build tag. The `t.Skip()` block is the platform gate. Removing the skip is sufficient; no new test build tag needed.

If the planner wants a Linux-only unit test for the detector itself (e.g., a `TestExitDetector_*` in `internal/pty/`), the convention there is `//go:build !windows` for POSIX tests (`session_exit_test.go:1`). A Linux-only detector test should use `//go:build linux` (matching the production file).

## 7. Cross-surface trace

The full chain from process exit to UI auto-close, with file:line precision:

```
[Linux child process exits via `exit 0`]
            │
            │  Today: pty.Read() at internal/relay/hub.go:140 BLOCKS — chain stalls here.
            ▼
[Phase 110 NEW]  pty.exit_linux.go: detector goroutine notices via syscall.Wait4(pid, _, WNOHANG)
            │   - Caches exit code: sess.SetExitCode(status.ExitStatus())
            │   - Closes PTY: sess.pty.Close() — closes master+slave (pty_unix.go:25-33)
            ▼
[Hub.Run]  internal/relay/hub.go:140: pty.Read returns error (EBADF / os.ErrClosed)
            │   - Loop exits at line 149, defer Shutdown() runs (line 136)
            │   - h.done channel closed via closeOnce (line 191)
            ▼
[Engine natural-exit goroutine]  internal/daemon/engine.go:328-346
            │   - <-hub.Done() unblocks (line 329)
            │   - sess.IsKilled() == false (natural exit, not killSession)
            │   - sess.CancelContext() (line 335) — triggers exec.Cmd.Cancel = Process.Kill;
            │     harmless if already exited.
            │   - time.Sleep(100ms) — already-cached exit code makes this a no-op now.
            │   - sess.ExitCode() returns cached value (or 0 fallback at line 339-341)
            │   - sess.SetState(pty.StateStopped) (line 342)
            │   - onExit(id, exitCode) (line 343-345)
            ▼
[Daemon API onExit callback]  internal/daemon/api.go:409-413
            │   - time.AfterFunc(10*time.Second, runSessionExitCleanup) — web grace period.
            │   - NOTE: this callback does NOT broadcast `session:exit`. That comes from polling.
            ▼
[Wails app polling]  app.go:261-304 (pollSessionStatus)
            │   - Polls daemon.ListSessions() every 500ms (line 302).
            │   - Sees state == "stopped" for this session (engine.go:361-363).
            │   - Calls emitExitEvent (app.go:292).
            ▼
[Wails event emit]  app.go:307-327 (emitExitEvent)
            │   - runtime.EventsEmit(a.ctx, "session:exit", {sessionId, exitCode, ...})
            ▼
[Frontend handler]  frontend/src/App.tsx:538 — listens for `session:exit`
            │   - If exitCode == 0 AND autoCloseRef ON (SHELL-12), calls handleCloseTabRef → tab closes.
            │   - For non-zero, sets sessionExits[id] (countdown UI per shellExit.test.tsx).
            ▼
[GUI tab closes / TUI list entry disappears]
```

**TUI side (parallel, independent):**
```
[TUI ticker]  internal/tui/cmds.go:30-34 — every 2s
    → daemon.ListSessions() → state == "stopped" → next render shows no entry (or stopped pill).
```

**CLI attach-detach:** untouched. CLI uses the relay WebSocket path; the daemon-side session cleanup is the same path, so the daemon's `runSessionExitCleanup` (api.go:436-444) fires `webServer.DisableSession` + `ClearGrants` exactly as today. CLI sessions that were attached see the WebSocket close as the relay hub shuts down via `hub.Shutdown()` (hub.go:186-193). [VERIFIED]

**Emit point summary:** `session:exit` is emitted from `app.go:319` (`runtime.EventsEmit`). The fix point is upstream of that — `internal/pty/exit_linux.go` (new) calling `sess.pty.Close()`. **Everything between is unchanged.**

Confidence: HIGH for the full chain (every reference verified in repo source).

## 8. Open Questions

1. **`os.Process.Kill()` after Wait4 reap — does Go's stdlib complain?**
   - What we know: `engine.go:335` calls `sess.CancelContext()`, which triggers `cmd.Cancel` (set by go-pty's `CommandContext` at `cmd_unix.go:27-29`), which calls `cmd.Process.Kill()`. After Wait4 has reaped the child, the PID may be recycled. `os.Process.Kill` documents returning an error for "process already finished" but is otherwise safe.
   - What's unclear: whether `cmd.Cancel`'s error is surfaced anywhere (it returns to internal `os/exec` machinery and may be logged or silently dropped).
   - Recommendation: verify in plan by running the test under `-race`. If it surfaces, replace `CancelContext()` with a guarded version that checks `sess.exitCode != -1` first.

2. **PID reuse race window.**
   - What we know: Wait4 reaps the child → kernel frees PID → another process on the system could take it. If our `cmd.Cancel` then fires after a delay (e.g., the 100 ms `time.Sleep`), it could theoretically Kill an unrelated process.
   - What's unclear: practical likelihood. PID reuse on Linux typically takes thousands of forks; a 100 ms window is short.
   - Recommendation: the detector should call `sess.CancelContext()` itself BEFORE closing the PTY — that way the cancel reaches `Process.Kill` on a still-known PID. Or, suppress the `CancelContext()` call in `engine.go:335` when the detector has already reaped (by checking `sess.ExitCode() != -1`). The planner picks.

3. **Wait4 on a process that has never been started.**
   - What we know: If `cmd.Start()` succeeded, `cmd.Process.Pid > 0`. The detector reads `sess.cmd.Process.Pid` once at startup.
   - What's unclear: nothing — but the detector goroutine must be spawned AFTER `cmd.Start()` returns successfully. Plan around `NativePTYBackend.Create` ordering (`native.go:53-57` is the Start call; detector should start after line 57 but BEFORE `b.registry.Add(sess)` for symmetry with macOS).

4. **Should the detector also handle the kill path?**
   - On the kill path, `killSession` already calls `cmd.Wait()` after closing the PTY. If the detector is still running and calls `Wait4` concurrently with the kernel-side Wait reaping, the race is real but harmless (both will see ECHILD or one wins the reap). The detector should check `sess.IsKilled()` and return early. This is mentioned in §2 but called out here as a deliberate plan item.

5. **macOS regression risk.**
   - `exit_other.go` is a no-op on macOS — macOS code path is byte-for-byte unchanged. Confidence in zero-regression: HIGH.
   - Caveat: if the planner decides to ALSO call `sess.pty.Close()` on the macOS natural-exit path for symmetry, that's an unnecessary regression risk. **Don't.** Leave macOS alone.

6. **Windows considered:** ROADMAP scope explicitly defers Windows. ConPTY EOF semantics are a separate problem; `exit_other.go` is a no-op on Windows by design.

## 9. Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (Go 1.26.1 per `go.mod:3`) |
| Config file | none (Go default) |
| Quick run command | `go test ./internal/daemon -run TestListSessions_OnExitCallback_ReceivesNormalized -race` |
| Full suite command | `go test ./... -race -shuffle=on` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| PTY-01 | GUI tab auto-closes on `exit 0` (Linux) | manual UAT | (Linux desktop build, type `exit`, observe tab close) | manual-only — UAT |
| PTY-01 | TUI list entry disappears on `exit 0` (Linux) | manual UAT | (Linux TUI build, attached session, type `exit`) | manual-only — UAT |
| PTY-02 | Wait4 detector fires on natural exit | unit | `go test ./internal/pty -run TestExitDetector -race` | ❌ Wave 0 (new file) |
| PTY-02 | Detector does NOT fire on kill path | unit | `go test ./internal/pty -run TestExitDetector_SuppressedOnKill -race` | ❌ Wave 0 (new file) |
| PTY-02 | macOS code path unchanged (stub no-op compiles) | build-only | `GOOS=darwin go build ./...` | ✅ existing |
| PTY-03 | `TestListSessions_OnExitCallback_ReceivesNormalized` passes on Linux | integration | `go test ./internal/daemon -run TestListSessions_OnExitCallback_ReceivesNormalized -race` | ✅ exists, currently `t.Skip()`'d |
| PTY-04 | Cross-platform regression (no macOS break) | full suite | `go test ./... -race -shuffle=on` (run on macOS dev box) | ✅ existing |
| PTY-04 | Cross-platform regression (no Windows break) | build-only | `GOOS=windows go build ./...` | ✅ existing |

### Sampling Rate
- **Per task commit:** `go test ./internal/pty ./internal/daemon -race`
- **Per wave merge:** `go test ./... -race -shuffle=on`
- **Phase gate:** Full suite green on macOS dev + Linux CI before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] `internal/pty/exit_linux.go` — Wait4 detector implementation
- [ ] `internal/pty/exit_other.go` — no-op stub for non-Linux
- [ ] `internal/pty/exit_linux_test.go` (or shared `exit_test.go` with `//go:build linux`) — unit tests for detector lifecycle, kill-path suppression, exit-code extraction
- [ ] Flip `t.Skip()` block at `internal/daemon/engine_test.go:1300-1307` (Linux skip removed)

## 10. Common Pitfalls

### Pitfall 1: Double-`Wait` race on the kill path
**What goes wrong:** Wait4 detector reaps child concurrently with `killSession`'s `cmd.Wait()`.
**Why it happens:** Both call the kernel's wait4 under the hood; whichever wins reaps; the loser gets ECHILD.
**How to avoid:** Detector polls `sess.IsKilled()` each tick and exits immediately when killed. `MarkKilled()` is called BEFORE `killSession` runs (engine.go:330 + Kill backend at native.go:122 sets it via `sess.MarkKilled()`).
**Warning signs:** Test failures under `-race`, sporadic "wait already called" errors in CI.

### Pitfall 2: PID recycling after Wait4 reap
**What goes wrong:** Detector reaps child, then existing `engine.go:335` `CancelContext` calls `Process.Kill` after a delay — but PID may have been reused.
**Why it happens:** Wait4 reap releases the PID; another process on the system may inherit it before our delayed `Kill` fires.
**How to avoid:** Have the detector call `sess.CancelContext()` itself, immediately after caching the exit code and BEFORE closing the PTY. OR add a guard in `engine.go:335` that skips CancelContext when `sess.ExitCode() != -1`.
**Warning signs:** Unrelated processes dying mysteriously on Linux dev boxes during tests.

### Pitfall 3: Detector leak on `killSession`
**What goes wrong:** Detector goroutine never exits when session is killed before its natural exit.
**Why it happens:** Detector polls only on time-tick, doesn't check stop signal.
**How to avoid:** Detector loop selects on `time.NewTicker(100ms).C` AND on a stop channel (e.g., `childCtx.Done()` from the session's context, or check `sess.IsKilled()` each tick).
**Warning signs:** Goroutine count grows over a long test session under `-race`.

### Pitfall 4: Closing PTY while another goroutine reads
**What goes wrong:** Detector calls `sess.pty.Close()` while `Hub.Run`'s `pty.Read()` is mid-syscall.
**Why it happens:** Concurrent close on `*os.File`.
**How to avoid:** `*os.File.Close` is safe to call concurrently with `Read` per Go stdlib docs (and this is precisely how `cleanup.go:39-41` already works on the kill path — it closes mid-Read intentionally to unblock).
**Warning signs:** None expected; this is a well-trodden pattern.

### Pitfall 5: Hardcoded poll interval, no tunability
**What goes wrong:** 100 ms is baked in; CI on slow ARM emulators may need longer.
**Why it happens:** Magic number in detector source.
**How to avoid:** Declare `const linuxExitPollInterval = 100 * time.Millisecond` at top of `exit_linux.go`. Single edit point if a future tuning is needed.

## 11. Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Process exit detection | A goroutine that calls `os.FindProcess(pid)` + `proc.Signal(syscall.Signal(0))` | `syscall.Wait4(pid, _, WNOHANG)` | Signal-0 polling doesn't yield exit code, races against PID recycling worse than Wait4. |
| Reading PTY EOF | Custom timeout on `pty.Read` | Close the master FD (`pty.Close()`) — already used in `cleanup.go:39-41` | The existing kill-path mechanism is the exact pattern; reuse it. |
| Exit code extraction | Parsing process state strings | `syscall.WaitStatus.ExitStatus()` / `.Signal()` / `.Exited()` / `.Signaled()` | Stdlib already canonicalizes; we don't reimplement POSIX status decoding. |
| Goroutine lifecycle | sync.WaitGroup ceremony | `context.Context` from `sess.cancel` + check `sess.IsKilled()` each tick | Existing session lifecycle uses context; integrate, don't add. |

## 12. Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Linux master Read blocks indefinitely after slave child clean exit (vs. macOS EOF). | §1 | LOW — verified in repo comments (engine_test.go:1300-1307) and Issue #57 evidence. If wrong, no fix needed — but issue exists. |
| A2 | Closing the master FD via `pty.Close()` unblocks a Linux-blocked `Read` with an error. | §1, §10 Pitfall 4 | LOW — same pattern used successfully on kill path (`cleanup.go:37-41`). |
| A3 | `os.Process.Kill()` on an already-reaped process returns a benign error, not a kill of a recycled PID, when the PID is still tracked by `os/exec`. | §8 Q1, §10 Pitfall 2 | MEDIUM — Go's `os/exec` may handle this internally, but the docs are vague. Mitigation in §10 Pitfall 2. |
| A4 | The 100 ms `time.Sleep` in `engine.go:337` becomes a no-op once the detector caches the exit code first. | §2, §7 | LOW — caching before close means `sess.ExitCode()` returns the real value, no fallback needed. |
| A5 | `syscall.Wait4` is sufficient; `golang.org/x/sys/unix.Wait4` not required for Linux-only file. | §3 | LOW — stdlib version is well-supported on linux/amd64 and arm64. |
| A6 | The TUI list entry "auto-closes" by ticker-poll detecting `state == "stopped"`, NOT by an active push event. | §7 | LOW — verified at `tui/cmds.go:30-34` (2s ticker) and absence of any push-mechanism grep. |
| A7 | go-pty v0.2.2 and v0.2.3 are Unix-identical (no version bump needed). | §2 | LOW — verified by `diff`. |

**No `[ASSUMED]` claims require user confirmation** — all hypotheses are testable in the plan and have explicit mitigations.

## 13. Code Examples

### Detector skeleton (illustrative — planner will refine)

```go
//go:build linux

// Source: synthesizing repo's existing patterns (cleanup.go close, session.go mutex usage)
// and Go stdlib syscall.Wait4 semantics.

package pty

import (
    "syscall"
    "time"
)

const linuxExitPollInterval = 100 * time.Millisecond

// startExitDetector spawns a goroutine that polls Wait4 until the child exits.
// On detection it caches the exit code and closes the PTY, which unblocks
// any pending Read in the relay hub.
func startExitDetector(s *Session) {
    if s.cmd == nil || s.cmd.Process == nil {
        return
    }
    pid := s.cmd.Process.Pid
    go func() {
        ticker := time.NewTicker(linuxExitPollInterval)
        defer ticker.Stop()
        for range ticker.C {
            if s.IsKilled() {
                return // killSession owns the wait; we exit silently.
            }
            var status syscall.WaitStatus
            wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
            if wpid == 0 && err == nil {
                continue // child still running
            }
            if err != nil {
                // ECHILD: already reaped (race we lost) or never had a child.
                // Either way: nothing to do. Let cleanup path own state.
                return
            }
            // wpid > 0: child reaped by us.
            var exitCode int
            switch {
            case status.Exited():
                exitCode = status.ExitStatus()
            case status.Signaled():
                exitCode = 128 + int(status.Signal())
            default:
                exitCode = 0
            }
            s.SetExitCode(exitCode)
            // Closing the PTY unblocks Hub.Run's blocked Read on the master.
            s.mu.Lock()
            p := s.pty
            s.mu.Unlock()
            if p != nil {
                _ = p.Close()
            }
            return
        }
    }()
}
```

```go
//go:build !linux

// Source: convention for build-tag stubs in this codebase.

package pty

// startExitDetector is a no-op on non-Linux platforms.
// macOS and BSDs surface EOF on the PTY master after the child exits.
// Windows uses ConPTY with separate exit semantics (out of scope for Phase 110).
func startExitDetector(_ *Session) {}
```

### Wire-up point in `native.go`

```go
// Source: internal/pty/native.go:53-93 — proposed insertion point.

if err := cmd.Start(); err != nil {
    cancel()
    _ = p.Close()
    return nil, fmt.Errorf("start process: %w", err)
}

// ... existing Resize handling ...

sess := &Session{ /* ... */ }

// On Windows, assign the process to a Job Object for reliable cleanup.
if cmd.Process != nil {
    assignJobObject(sess, cmd.Process)
}

// Phase 110: on Linux, spawn an exit-detector goroutine to unblock the
// PTY read loop on natural exit. No-op on macOS/BSD/Windows.
startExitDetector(sess)

b.registry.Add(sess)
return sess, nil
```

## 14. State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Rely on PTY EOF surfacing on Linux master Read | Active Wait4 polling + explicit Close | Phase 110 | Surgical, hermetic to `internal/pty/exit_linux.go`. No upstream contracts change. |
| Hope go-pty exposes an exit hook on Unix | None exists; v0.2.2/v0.2.3 have no Unix `waitOnContext` | (n/a — never existed on Unix) | Confirms we must add detection ourselves. |

## Sources

### Primary (HIGH confidence)
- `/Users/ken/dev/agenthub/internal/relay/hub.go` — read loop (lines 135-152), Done channel (208-211)
- `/Users/ken/dev/agenthub/internal/pty/session.go` — Read wrapper, mutex usage, MarkKilled, ExitCode
- `/Users/ken/dev/agenthub/internal/pty/native.go` — Create flow, wire-up point
- `/Users/ken/dev/agenthub/internal/pty/cleanup.go` — existing PTY-close-to-unblock pattern (37-41)
- `/Users/ken/dev/agenthub/internal/daemon/engine.go` — natural-exit goroutine (328-346), ListSessions stopped-state (361-363, 384-395)
- `/Users/ken/dev/agenthub/internal/daemon/engine_test.go:1287-1342` — skipped test contract
- `/Users/ken/dev/agenthub/internal/daemon/engine_exit_test.go` — spyBackend test infra
- `/Users/ken/dev/agenthub/app.go:259-327` — pollSessionStatus + emitExitEvent (Wails layer)
- `/Users/ken/dev/agenthub/internal/tui/cmds.go:30-34` — TUI session list polling
- `/Users/ken/go/pkg/mod/github.com/aymanbagabas/go-pty@v0.2.2/cmd_unix.go` — Unix Cmd.start/wait
- `/Users/ken/go/pkg/mod/github.com/aymanbagabas/go-pty@v0.2.2/pty_unix.go` — Read/Close on master
- Confirmed v0.2.2 == v0.2.3 on Unix via `diff`
- `/Users/ken/dev/agenthub/.planning/REQUIREMENTS.md` (PTY-01..04)
- `/Users/ken/dev/agenthub/.planning/milestones/v3.3.1-ROADMAP.md` (Phase 110 section)

### Secondary (MEDIUM confidence)
- [Linux wait4(2) man page](https://man7.org/linux/man-pages/man2/wait4.2.html) — WNOHANG return semantics
- [Go stdlib syscall.Wait4](https://pkg.go.dev/syscall#Wait4) — function signature, parameters
- [golang-codereviews discussion on Wait4 + WNOHANG](https://groups.google.com/g/golang-codereviews/c/oa4wm3-XBps) — pid == 0 == "still running" semantics
- [creack/pty issue tracker](https://github.com/creack/pty/issues) (broad) — Linux master EOF blocking discussions (general ecosystem knowledge, repo bug reports)

### Tertiary (LOW confidence)
- None — all critical claims grounded in repo source or stdlib docs.

## Metadata

**Confidence breakdown:**
- Read-loop blocking root cause: HIGH — explicit repo comments + Issue #57 reproduce.
- go-pty internals on Unix: HIGH — vendored source inspection.
- Wait4 semantics: HIGH — stdlib + man page agree.
- Build-tag convention: HIGH — direct precedent in `internal/pty`.
- Cadence: HIGH — codebase precedent at 100 ms.
- Cross-surface trace: HIGH — every link verified in source.
- Pitfalls: MEDIUM-HIGH — PID-recycle and double-Wait scenarios are theoretical; mitigations are concrete.

**Research date:** 2026-05-18
**Valid until:** 2026-06-17 (30 days; codebase stable, go-pty unchanged for 2+ releases)
