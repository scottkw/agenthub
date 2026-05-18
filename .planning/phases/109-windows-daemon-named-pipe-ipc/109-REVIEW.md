---
phase: 109
status: findings
critical_count: 0
high_count: 0
medium_count: 2
low_count: 3
medium_fixed: 2
low_fixed: 0
---

# Phase 109 Code Review — Windows daemon named-pipe IPC (PR #53 cherry-pick)

**Reviewed:** 2026-05-18
**Depth:** standard (cross-file analysis on `internal/daemon` + verification against PR #53)
**Reviewer:** Claude (gsd-code-reviewer)
**Scope:** Commits `68b2421`, `2f25e63`, `fc50cd4` on `phase-109-windows-named-pipe-ipc` (changes since `9cc1087`)

## Files reviewed

- `internal/daemon/ipc_windows.go` (new)
- `internal/daemon/ipc_nonwindows.go` (new)
- `internal/daemon/api.go` (modified: `Start`, `Stop`)
- `internal/daemon/client.go` (modified: `NewDaemonClient` dial closure)
- `internal/daemon/socket_windows_test.go` (modified: added two tests + helper)
- `tray_windows.go` (modified: `GetModuleHandleW` DLL fix)

## Summary

The cherry-pick lands cleanly. The build-tag split is correct, function signatures match across both files, the kernel32 tray fix is genuine, and the round-trip test exercises `Start → Health → Stop` as IPC-04 requires. No bugs introduced relative to `main` behavior.

Two **MEDIUM** issues are worth surfacing for follow-up but neither blocks shipping Phase 109:

1. **Windows dial path loses the 2-second fast-fail timeout** that the Unix path retains, so a `BUSY` pipe instance can spin-retry under `context.Background()` indefinitely on Windows. (Behavior parity gap — not present on `main` because `main` doesn't dial pipes at all.)
2. **Windows `API.Stop()` always returns `nil` regardless of close errors**, because `winio.win32PipeListener.Close()` swallows errors upstream. Callers cannot distinguish a clean stop from a forced one. Minor — affects only diagnostic surfaces, not correctness.

Three **LOW** findings are stylistic / hygiene observations.

The plan's `must_haves` and `threat_model` mitigations (T-109-04 atomic addr capture, T-109-05 `os.Remove` short-circuit on pipe paths, IPC-06 author preservation) all hold against the landed code.

---

## MEDIUM findings

### MR-01: Windows dial path has no fast-fail timeout — `ERROR_PIPE_BUSY` spins forever under `context.Background()`

**Status:** fixed in `dd73b61` (post-cherry-pick adjustment, 2026-05-18)
**Severity:** MEDIUM (Behavior parity gap — Windows-only)
**File:** `internal/daemon/ipc_windows.go:17-19`, `internal/daemon/client.go:353-389` (caller)

**Code:**

```go
// ipc_windows.go
func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error) {
    return winio.DialPipeContext(ctx, path)
}
```

```go
// ipc_nonwindows.go (for comparison)
func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error) {
    return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", path)
}
```

**Issue:** The Unix side defends against a slow/missing socket with a hard-coded 2-second dial timeout in addition to whatever `ctx` says. The Windows side relies entirely on `ctx` — and `DaemonClient.doJSON` (`client.go:363`) creates requests via `http.NewRequest` (no context), which means the dial runs under `context.Background()`.

`winio.tryDialPipe` (verified in `$GOMODCACHE/.../go-winio@v0.0.0-20231025203758/pipe.go:207-231`) returns `ERROR_FILE_NOT_FOUND` immediately for missing pipes — so the "no daemon" case is safe. But for `ERROR_PIPE_BUSY` (the daemon is up but every pipe instance is currently held by other clients before the server has called `Accept` for a fresh one), `tryDialPipe` loops with `time.Sleep(10 * time.Millisecond)` until `ctx` cancels — and `context.Background()` never cancels.

In practice this is rare with a single-pipe-instance daemon, but the original pre-PR code had a 2s ceiling on the Unix side; the cherry-pick silently dropped it on Windows.

**Recommended fix:** Add a deadline in the Windows variant so it mirrors the Unix 2-second cap:

```go
func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error) {
    if _, ok := ctx.Deadline(); !ok {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
        defer cancel()
    }
    return winio.DialPipeContext(ctx, path)
}
```

Alternative: lift the timeout to the caller by switching `client.go:363` to `http.NewRequestWithContext(context.Background(), ...)` and applying a per-request deadline. The helper-side fix is more local and preserves the existing call sites untouched.

---

### MR-02: `API.Stop()` returns `nil` from Windows close path regardless of underlying error

**Status:** fixed in `9994151` (doc comment added on `API.Stop`, 2026-05-18)
**Severity:** MEDIUM (Diagnostic — error-suppression on Windows)
**File:** `internal/daemon/api.go:196-200`

**Code:**

```go
addr := a.ln.Addr().String()
err := a.ln.Close()
_ = removeDaemonSocket(addr)
a.ln = nil
return err
```

**Issue:** `winio.win32PipeListener.Close()` (verified in `$GOMODCACHE/.../go-winio@v0.0.0-20231025203758/pipe.go:575-582`) is structured as:

```go
func (l *win32PipeListener) Close() error {
    select {
    case l.closeCh <- 1:
        <-l.doneCh
    case <-l.doneCh:
    }
    return nil
}
```

`Close()` always returns `nil` on Windows. Any error from the underlying overlapped-I/O cancellation is swallowed by `winio` itself, so `API.Stop()` on Windows cannot ever report a close failure — callers see `nil` even if the listener routine panicked or the OS handle was already torn down.

This is upstream behavior (not introduced by PR #53), and the Unix `net.UnixListener.Close()` does propagate errors — so Windows is **less** observable than Unix. Worth flagging because:

1. `runDaemonCore` ignores the error already (no caller currently consumes it), so today's impact is zero.
2. `TestAPIStop_WindowsNamedPipe` asserts `api.Stop()` returns `nil`. That assertion is **tautologically true** on Windows (always nil) and therefore doesn't actually exercise the error path on the platform where this code matters.

**Recommended fix:** None required. Document the asymmetry in `api.go::Stop` doc comment so future readers don't mistake the `nil`-return for a successful close on Windows:

```go
// Stop closes the listener and (on Unix) removes the socket file.
//
// Note: on Windows, winio's listener Close() always returns nil — the
// return value of Stop() reflects Unix close errors only.
func (a *API) Stop() error {
```

Optionally, harden `TestAPIStop_WindowsNamedPipe` to also verify the listener stops accepting (the post-Stop Health() failure already implicitly tests this).

---

## LOW findings

### LR-01: `isWindowsNamedPipe` matches all UNC paths, not just pipe paths

**Severity:** LOW (Over-broad guard — no current real-world impact)
**File:** `internal/daemon/socket.go:46-48` (pre-existing; reused by Phase 109's `removeDaemonSocket`)

**Code:**

```go
func isWindowsNamedPipe(path string) bool {
    return strings.HasPrefix(path, `\\`)
}
```

**Issue:** The check returns `true` for any path starting with two backslashes — `\\.\pipe\agenthub-daemon` (intended) but also `\\server\share\foo` (UNC file path) or `\\?\C:\very\long\path` (Win32 long-path namespace). On Windows, `removeDaemonSocket` would silently no-op on a UNC file path the user might pass via `--socket-path` for some reason.

Because the daemon only ever derives socket paths via `DefaultSocketPath()` (which hard-codes `\\.\pipe\agenthub-daemon` on Windows), this is purely theoretical today. But the function name promises "named pipe" — and it would lie about a UNC share.

**Recommended fix:** Tighten the prefix to the actual pipe namespace:

```go
func isWindowsNamedPipe(path string) bool {
    return strings.HasPrefix(path, `\\.\pipe\`) || strings.HasPrefix(path, `\\?\pipe\`)
}
```

The second form covers the rarely-used long-path pipe alias; both are sanctioned by Windows. Not Phase 109's bug — flagging here because the cherry-pick now consumes this guard in a place (`removeDaemonSocket`) where a mis-classification would cause a different failure mode than before.

---

### LR-02: `TestAPIStart_WindowsNamedPipeHealth` and `TestAPIStop_WindowsNamedPipe` reach into unexported engine fields

**Severity:** LOW (Test hygiene)
**File:** `internal/daemon/socket_windows_test.go:54-57`, `:74-77`

**Code:**

```go
engine := NewSessionEngine()
engine.configDir = t.TempDir()
engine.cliPaths = make(map[string]string)
engine.startMinimized = false
```

**Issue:** The tests poke unexported fields on `SessionEngine` to side-step the normal init path. This works (same package) but couples the test to the engine's internal field layout — any rename of `configDir` / `cliPaths` / `startMinimized` silently breaks Windows-only tests that the macOS executor cannot run. Other tests in the package (`api_test.go`, `engine_test.go`) use the same shape, so this is consistent with the local convention; flagging only as a maintenance risk.

**Recommended fix:** If a test-only constructor is added later (e.g., `newTestEngine(t)`), migrate these two tests onto it. Not required for Phase 109.

---

### LR-03: `socket_windows_test.go` does not exercise a non-trivial HTTP payload over the pipe

**Severity:** LOW (Test depth — meets IPC-04 letter but not spirit)
**File:** `internal/daemon/socket_windows_test.go:53-91`

**Issue:** The round-trip test calls `client.Health()` which sends `GET /health` (no body) and receives a `~30-byte` JSON response (`{"status":"ok","version":""}`). It does verify the wire path works, but it doesn't exercise:

- Request bodies (e.g., `POST /sessions` with a `CreateRequest`) — would catch framing / chunking bugs.
- Path parameters (`GET /sessions/{id}`) — would catch URL-rewriting edge cases through the custom transport's `http://daemon` base.
- Sustained traffic / multiple concurrent requests over a single pipe instance.

IPC-04 is satisfied as worded ("Start + Health + Stop over a real named pipe"), so this is not a plan miss. But for a transport substitution, a single-request smoke is shallow — the kinds of failure mode the named-pipe transport can introduce (message vs byte mode mismatches, half-close behavior, ACL surprises) won't surface here.

**Recommended fix:** Add one optional follow-up test that round-trips `ListSessions()` or `CreateSession()`+`KillSession()` on the named pipe. Defer to a future phase or to operator UAT (WIN-CLI-01 in `109-VERIFICATION.md` already covers this manually).

---

## Notes (no severity assigned)

- **Kernel32 tray fix verified.** `GetModuleHandleW` is documented by Microsoft as a `kernel32.dll` export; the assertion that `user32.dll` re-exports it is false. `windows.NewLazySystemDLL("kernel32.dll")` is the canonical idiom (used elsewhere in `tray_windows.go` for `user32`, `gdi32`, `shell32`). The diff is minimal and correct.

- **`winio.ListenPipe(path, nil)` ACL.** Confirmed via `pipe.go:323-371`: passing `nil` PipeConfig triggers `rtlDefaultNpAcl` for the security descriptor, yielding the Windows default named-pipe DACL (creator + administrators read/write; other users denied). Matches the T-109-01 mitigation in the plan's threat model.

- **`os.Remove` short-circuit on pipes (T-109-05).** `ipc_windows.go::removeDaemonSocket` checks `isWindowsNamedPipe(path)` before `os.Remove` — correct and tested by `TestAPIStop_WindowsNamedPipe` via the absence of any filesystem operation after Stop on a `\\.\pipe\` path.

- **Atomic addr capture (T-109-04).** `api.go:196-198` captures `addr := a.ln.Addr().String()` before `a.ln.Close()`. Strictly defensive against `winio` post-close semantics; `winio.win32PipeListener.Addr()` (`pipe.go:584-586`) returns the stored path verbatim, so the capture-order doesn't currently change behavior — but it's the right pattern.

- **Author preservation (IPC-06).** `git log --format='%an <%ae>' main..HEAD | grep -c 'Alexandre Castro'` returns 3 — confirmed across the three cherry-picked commits. No `Co-Authored-By:` trailers added (correct; the `Author:` field carries attribution canonically per the plan's D-04 / Task 2 step 3).

- **Cross-compile.** `GOOS=windows GOARCH=amd64 go build -tags wailsassets ./...` is expected to pass per Task 2/3 verifications. Not re-run in this review (Reviewer is read-only).

---

_Reviewed: 2026-05-18_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
