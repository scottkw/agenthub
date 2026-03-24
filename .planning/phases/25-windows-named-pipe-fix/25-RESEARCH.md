# Phase 25: Windows Named Pipe Dial Fix - Research

**Researched:** 2026-03-24
**Domain:** Go Windows named pipe IPC / daemon socket probe
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DAEMON-05 | Daemon auto-starts when any CLI command is run and no daemon is running | Fix enables `CleanupStaleSocket` to correctly probe a Windows named pipe; without the fix `EnsureDaemon` always sees a dial error and attempts a duplicate spawn |
</phase_requirements>

---

## Summary

`CleanupStaleSocket` in `internal/daemon/socket.go` calls `net.DialTimeout("unix", path, 500ms)` unconditionally. On Windows, `DefaultSocketPath()` returns `\\.\pipe\agenthub-daemon` — a Windows named pipe path, not a Unix domain socket path. Go's `"unix"` network type does not understand the `\\.\pipe\` namespace; the dial always fails with an error that is not `os.IsNotExist`, so `CleanupStaleSocket` removes the (non-existent) path and returns nil, treating every probe as "stale". `EnsureDaemon` therefore always thinks no daemon is running, regardless of whether one is, and attempts a redundant spawn.

The fix is minimal: detect whether the socket path is a Windows named pipe (prefix `\\.\pipe\` or `\\server\pipe\`) and use `winio.DialPipe` from the already-present `github.com/tailscale/go-winio` dependency instead of `net.DialTimeout("unix", ...)`. Because `winio.DialPipe` returns a `net.Conn` the rest of the function logic (connection succeeded → daemon running, connection error → stale) stays identical. On Windows, named pipes have no filesystem entry to remove; `os.Remove` on a named pipe path silently fails or is a no-op, which is correct behavior (the pipe ceases to exist when the last server-side handle closes).

The change belongs entirely in `socket.go`. A build-tag–free helper `isWindowsNamedPipe(path string) bool` is the cleanest approach: `strings.HasPrefix(path, `\\`)`. The Windows-specific dial is called only when this helper returns true; Unix paths continue to use the existing `net.DialTimeout("unix", ...)` path.

**Primary recommendation:** Add `isWindowsNamedPipe` path detection to `CleanupStaleSocket` and use `winio.DialPipe` for the Windows branch; add a matching test using a real `winio.ListenPipe` listener and a build-tag `//go:build windows` test file.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/tailscale/go-winio` | `v0.0.0-20231025203758-c4f33415bf55` | Windows named pipe dial/listen | Already in go.mod as indirect dep; `DialPipe` and `ListenPipe` are exactly what's needed |
| Go stdlib `net` | go1.26.1 | Unix socket dial (unchanged) | Current path for non-Windows |
| Go stdlib `strings` | go1.26.1 | `HasPrefix` for pipe detection | Zero-dependency path detection |

No new dependencies required — `tailscale/go-winio` is already in `go.sum` and available in the module cache.

**Installation:** No new packages needed. `go-winio` is already an indirect dependency. To promote it to direct use in the daemon package:

```bash
# In /Users/ken/dev/agenthub:
go get github.com/tailscale/go-winio@v0.0.0-20231025203758-c4f33415bf55
```

**Version verification:** Confirmed `v0.0.0-20231025203758-c4f33415bf55` in `go.mod` and local module cache at `$(go env GOPATH)/pkg/mod/github.com/tailscale/go-winio@v0.0.0-20231025203758-c4f33415bf55/`.

### Relevant API (verified from module cache source)

```go
// DialPipe — connects to named pipe, returns net.Conn
func DialPipe(path string, timeout *time.Duration) (net.Conn, error)

// ListenPipe — creates net.Listener on named pipe (for tests)
func ListenPipe(path string, c *winio.PipeConfig) (net.Listener, error)
```

`DialPipe` error behavior on Windows:
- Pipe not found (no server): returns `*os.PathError` wrapping `windows.ERROR_FILE_NOT_FOUND` — not `os.IsNotExist` matching
- Pipe busy (all instances taken): returns `ErrTimeout` after deadline — treated as "stale" (correct)
- Connection success: returns valid `net.Conn` — daemon is running

---

## Architecture Patterns

### Recommended Project Structure

No new files needed. Changes are confined to:

```
internal/daemon/
├── socket.go           # Add isWindowsNamedPipe helper + winio dial branch
└── socket_test.go      # Add build-tagged Windows pipe tests (or skip on non-Windows)
```

If Windows tests need platform-specific setup, add:

```
internal/daemon/
└── socket_windows_test.go   # //go:build windows — ListenPipe + CleanupStaleSocket tests
```

### Pattern 1: Platform-Detected Dial in CleanupStaleSocket

**What:** Single function handles both Unix sockets and Windows named pipes by detecting the path format, not with build tags in the core logic.

**When to use:** When the function must compile on all platforms but take a different code path on Windows.

**Example:**
```go
// Source: internal/daemon/socket.go (proposed change)

// isWindowsNamedPipe reports whether path is a Windows named pipe path.
// Named pipe paths start with \\ (double backslash).
func isWindowsNamedPipe(path string) bool {
    return strings.HasPrefix(path, `\\`)
}

func CleanupStaleSocket(path string) error {
    if isWindowsNamedPipe(path) {
        return cleanupStaleWindowsPipe(path)
    }
    // existing Unix logic unchanged below
    conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
    ...
}
```

### Pattern 2: Build-Tagged Windows Dial Helper

**What:** `cleanupStaleWindowsPipe` lives in a `_windows.go` file so `winio` is only imported on Windows, keeping the Unix build free of Windows deps.

**When to use:** When a dependency (winio) is only meaningful on one platform.

**Example:**
```go
// Source: internal/daemon/socket_windows.go  (new file)
//go:build windows

package daemon

import (
    "fmt"
    "time"
    winio "github.com/tailscale/go-winio"
)

func cleanupStaleWindowsPipe(path string) error {
    timeout := 500 * time.Millisecond
    conn, err := winio.DialPipe(path, &timeout)
    if err != nil {
        // Pipe does not exist or is not listening — nothing to clean up.
        // Named pipes have no filesystem entry; no os.Remove needed.
        return nil
    }
    conn.Close()
    return fmt.Errorf("daemon already running at %s", path)
}
```

```go
// Source: internal/daemon/socket_nonwindows.go  (new file, or fold into socket.go)
//go:build !windows

package daemon

// cleanupStaleWindowsPipe is unreachable on non-Windows; isWindowsNamedPipe
// always returns false for Unix paths. Stub keeps the compiler happy if
// isWindowsNamedPipe were ever true on non-Windows (it won't be).
```

**Alternative (simpler):** Keep everything in `socket.go` using `runtime.GOOS` guard inside `CleanupStaleSocket` with a `//go:build windows` file only for the winio import. The build-tag file pattern avoids a `runtime.GOOS` string check at runtime for the dispatch. Both approaches are valid; build-tag split is idiomatic Go and matches the existing `process_windows.go` / `process_unix.go` pattern in this package.

### Anti-Patterns to Avoid

- **Importing winio in non-build-tagged socket.go:** winio has `//go:build windows` guards inside itself but importing it without a build tag creates a confusing dependency. Follow the existing `process_windows.go` pattern.
- **Using `os.IsNotExist` to detect "pipe not found" on Windows:** Windows named pipe errors use `windows.ERROR_FILE_NOT_FOUND` wrapped in `*os.PathError`; `os.IsNotExist` may or may not unwrap correctly. The safe approach is: any `DialPipe` error → not connected → stale/absent → return nil.
- **Calling `os.Remove(path)` for named pipes:** Named pipes are kernel objects, not filesystem files. `os.Remove` on `\\.\pipe\name` will fail silently or return an error. Do not call it for named pipe paths.
- **Using `net.DialTimeout("unix", \\.\pipe\name, ...)` — the original bug:** `"unix"` network type does not understand the `\\` namespace on Windows. This always fails.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Windows named pipe dial | Custom Win32 `CreateFile` syscall wrapper | `winio.DialPipe` | `tailscale/go-winio` already handles overlapped I/O, timeout, context propagation, and the exact Windows API surface |
| Windows named pipe listen (tests) | Custom server socket | `winio.ListenPipe` | Same package; provides `net.Listener` interface compatible with `go test` teardown patterns |
| Pipe path detection | Complex path parsing | `strings.HasPrefix(path, `\\`)` | Named pipe paths always start with `\\`; no other Go IPC path format uses this prefix |

**Key insight:** `tailscale/go-winio` is already in `go.sum` — the module is fetched. Adding a build-tagged import in one new file costs zero new network fetches and zero new review burden.

---

## Common Pitfalls

### Pitfall 1: Treating All DialPipe Errors as "Stale"

**What goes wrong:** `winio.DialPipe` returns `winio.ErrTimeout` when the pipe path exists but all instances are busy. If this is treated the same as "not found," the code tries to `os.Remove` the path (wrong) or returns nil (claiming stale when a live daemon is actually saturated).

**Why it happens:** The error type is `winio.ErrTimeout`, not an `os.PathError`, and does not indicate absence.

**How to avoid:** For `CleanupStaleSocket`, the semantics are: success → daemon running → return "already running" error. Any error → no active connection established → return nil (allow fresh start). This is correct for both "not found" and "all instances busy" because in either case we should not block a daemon spawn. The saturated case is extremely unlikely for a single-process daemon.

**Warning signs:** Test shows `CleanupStaleSocket` returns "already running" when no daemon is listening.

### Pitfall 2: Missing Build Tag on winio Import

**What goes wrong:** Adding `import "github.com/tailscale/go-winio"` to `socket.go` (no build tag) causes compilation failure on Linux/macOS if winio has platform guards for its internal Windows types.

**Why it happens:** `tailscale/go-winio` exports Windows-only types; the package itself has `//go:build windows` at file level for most of its functionality.

**How to avoid:** Import winio only in `socket_windows.go` with `//go:build windows`. This matches the existing `process_windows.go` pattern already in the package.

**Warning signs:** `go build ./...` fails on macOS with "undefined: windows.Handle" or similar.

### Pitfall 3: os.Remove on Named Pipe Path

**What goes wrong:** Calling `os.Remove(`\\.\pipe\agenthub-daemon`)` either silently fails (returns error, ignored with `_ =`) or panics. No cleanup is needed — Windows automatically destroys the pipe when the last server handle closes.

**Why it happens:** Copy-paste from Unix path where `os.Remove` removes the socket inode.

**How to avoid:** The Windows branch of `CleanupStaleSocket` must NOT call `os.Remove`. Named pipes are ephemeral kernel objects. Document this explicitly in the code comment.

**Warning signs:** Test teardown sees errors about "The system cannot find the path specified."

### Pitfall 4: Test Uses Unix Socket on Windows Build

**What goes wrong:** `socket_test.go` uses `net.Listen("unix", path)` to create the test listener. On Windows this fails because Unix sockets are only available in recent Windows builds (Windows 10 build 17063+) and not universally available in CI.

**Why it happens:** Existing tests were written for macOS/Linux. The new Windows tests need `winio.ListenPipe` / `winio.DialPipe`.

**How to avoid:** New Windows-specific tests go in `socket_windows_test.go` with `//go:build windows`. Existing `socket_test.go` tests continue to use `net.Listen("unix", ...)` since they only run on non-Windows.

---

## Code Examples

### Proposed CleanupStaleSocket Fix (socket.go)

```go
// Source: internal/daemon/socket.go (proposed)
// isWindowsNamedPipe reports whether path is a Windows named pipe path.
func isWindowsNamedPipe(path string) bool {
    return strings.HasPrefix(path, `\\`)
}

func CleanupStaleSocket(path string) error {
    if isWindowsNamedPipe(path) {
        return cleanupStaleWindowsPipe(path)
    }
    conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        _ = os.Remove(path)
        return nil
    }
    conn.Close()
    return fmt.Errorf("daemon already running at %s", path)
}
```

### Windows Dial Helper (socket_windows.go — new file)

```go
// Source: internal/daemon/socket_windows.go (new)
//go:build windows

package daemon

import (
    "fmt"
    "time"

    winio "github.com/tailscale/go-winio"
)

// cleanupStaleWindowsPipe probes a Windows named pipe to determine if a daemon
// is actively listening. Named pipes are kernel objects; there is no filesystem
// file to remove when no server is present.
func cleanupStaleWindowsPipe(path string) error {
    timeout := 500 * time.Millisecond
    conn, err := winio.DialPipe(path, &timeout)
    if err != nil {
        // Any dial error (pipe absent, timeout, etc.) means nothing is listening.
        // No cleanup needed — named pipes vanish when the last server handle closes.
        return nil
    }
    conn.Close()
    return fmt.Errorf("daemon already running at %s", path)
}
```

### Windows Test (socket_windows_test.go — new file)

```go
// Source: internal/daemon/socket_windows_test.go (new)
//go:build windows

package daemon

import (
    "strings"
    "testing"

    winio "github.com/tailscale/go-winio"
)

func TestCleanupStaleSocket_WindowsPipe_NoServer(t *testing.T) {
    path := `\\.\pipe\agenthub-test-nostale`
    // Nothing listening — should return nil.
    if err := CleanupStaleSocket(path); err != nil {
        t.Errorf("CleanupStaleSocket on absent pipe: unexpected error: %v", err)
    }
}

func TestCleanupStaleSocket_WindowsPipe_Active(t *testing.T) {
    path := `\\.\pipe\agenthub-test-active`
    ln, err := winio.ListenPipe(path, nil)
    if err != nil {
        t.Fatalf("ListenPipe: %v", err)
    }
    defer ln.Close()
    // Accept in background so DialPipe succeeds.
    go func() {
        for {
            c, err := ln.Accept()
            if err != nil {
                return
            }
            c.Close()
        }
    }()

    err = CleanupStaleSocket(path)
    if err == nil {
        t.Error("CleanupStaleSocket on active pipe: expected error, got nil")
    }
    if !strings.Contains(err.Error(), "already running") {
        t.Errorf("error should mention 'already running', got: %v", err)
    }
}
```

---

## Scope Boundary: What Phase 25 Does NOT Cover

The audit identified a related issue in `api.go:79`:

```go
ln, err := net.Listen("unix", socketPath)  // also wrong on Windows
```

This is the server-side listen when the daemon starts. On Windows, `net.Listen("unix", `\\.\pipe\...`)` would also fail. However, INT-01 is **specifically scoped to `CleanupStaleSocket`** (the probe dial), and INT-02 (GUI panic) is a separate gap. The `api.go` server listen is not called out as a separate INT gap; it may be addressed in Phase 26 or may already be handled at a higher level (kardianos/service on Windows might route the listen differently). Phase 25 is scoped to `CleanupStaleSocket` only.

**Recommendation for planner:** Confirm whether `api.go` `net.Listen("unix", ...)` also needs a Windows fix. If yes, include a `socket_listen_windows.go` with `winio.ListenPipe`. If no (it already works or is deferred), keep Phase 25 strictly to `CleanupStaleSocket`.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `net.DialTimeout("unix", path, ...)` for all paths | `winio.DialPipe` for Windows named pipes, `net.DialTimeout("unix", ...)` for Unix | This phase | Correct probe behavior on Windows; eliminates false "stale" detection |
| Single `socket.go` with no platform split | `socket.go` + `socket_windows.go` build-tag split | This phase | Follows existing `process_windows.go`/`process_unix.go` pattern |

---

## Open Questions

1. **Does `api.go`'s `net.Listen("unix", socketPath)` also need to be fixed for Windows?**
   - What we know: `DefaultSocketPath()` returns `\\.\pipe\...` on Windows; `net.Listen("unix", ...)` will fail with that path
   - What's unclear: Whether Phase 25 should also fix the server-side listen, or if Phase 26 covers it
   - Recommendation: The phase description says "CleanupStaleSocket" only. If the listen side is broken too, the planner should add a task to fix `api.go` as well — ideally with `winio.ListenPipe` in `api_windows.go`.

2. **Is `\\` prefix detection robust enough, or should we use `runtime.GOOS == "windows"`?**
   - What we know: `DefaultSocketPath()` already uses `runtime.GOOS == "windows"` to pick the path format; `isWindowsNamedPipe` based on string prefix is equivalent for paths produced by `DefaultSocketPath` but could false-positive on a Unix path starting with `\\` (extremely unlikely; UNC paths are not used as Unix socket paths)
   - Recommendation: Both work. String prefix is fast and requires no OS syscall. Alternatively, call `runtime.GOOS == "windows"` directly in `CleanupStaleSocket` — this is equally readable and eliminates any theoretical ambiguity. Either is fine; pick one and be consistent.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (go test) |
| Config file | none — standard `go test ./...` |
| Quick run command | `go test ./internal/daemon/... -run TestCleanupStale -v` |
| Full suite command | `go test ./internal/daemon/...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DAEMON-05 | `CleanupStaleSocket` returns nil when Windows named pipe has no server | unit | `go test ./internal/daemon/... -run TestCleanupStaleSocket_WindowsPipe_NoServer -v` | ❌ Wave 0 (socket_windows_test.go) |
| DAEMON-05 | `CleanupStaleSocket` returns "already running" error when named pipe is active | unit | `go test ./internal/daemon/... -run TestCleanupStaleSocket_WindowsPipe_Active -v` | ❌ Wave 0 (socket_windows_test.go) |
| DAEMON-05 | Existing Unix socket tests continue to pass (regression) | unit | `go test ./internal/daemon/... -run TestCleanupStale -v` | ✅ socket_test.go |

### Sampling Rate
- **Per task commit:** `go test ./internal/daemon/... -run TestCleanupStale`
- **Per wave merge:** `go test ./internal/daemon/...`
- **Phase gate:** Full suite green on target platform before `/gsd:verify-work`

**Note:** Windows pipe tests (`//go:build windows`) will only execute in a Windows environment. On macOS/Linux CI, they are skipped. The planner should note that complete verification of the Windows-specific path requires a Windows build.

### Wave 0 Gaps
- [ ] `internal/daemon/socket_windows.go` — `cleanupStaleWindowsPipe` implementation
- [ ] `internal/daemon/socket_windows_test.go` — `TestCleanupStaleSocket_WindowsPipe_NoServer` + `TestCleanupStaleSocket_WindowsPipe_Active`

---

## Sources

### Primary (HIGH confidence)
- Module cache: `$(go env GOPATH)/pkg/mod/github.com/tailscale/go-winio@v0.0.0-20231025203758-c4f33415bf55/pipe.go` — `DialPipe`, `ListenPipe` function signatures verified directly
- `/Users/ken/dev/agenthub/go.mod` — `github.com/tailscale/go-winio` confirmed as existing indirect dependency
- `/Users/ken/dev/agenthub/internal/daemon/socket.go` — exact bug location: line 51 `net.DialTimeout("unix", path, 500ms)`
- `/Users/ken/dev/agenthub/.planning/v1.3-MILESTONE-AUDIT.md` — INT-01 gap description with file/line reference

### Secondary (MEDIUM confidence)
- [github.com/microsoft/go-winio pipe.go](https://github.com/microsoft/go-winio/blob/main/pipe.go) — confirmed `DialPipe(path string, timeout *time.Duration) (net.Conn, error)` and `DialPipeContext` signatures (upstream of tailscale fork)
- [pkg.go.dev/github.com/tailscale/go-winio](https://pkg.go.dev/github.com/tailscale/go-winio) — confirmed package identity

### Tertiary (LOW confidence)
- WebSearch results on Windows named pipe OS.Remove behavior — confirmed via Microsoft documentation that named pipes are kernel objects and are automatically destroyed; no Remove needed

---

## Metadata

**Confidence breakdown:**
- Bug location: HIGH — read directly from source file
- Fix approach: HIGH — verified `DialPipe` signature in local module cache; `winio` already in dependency tree
- `os.Remove` no-op for Windows pipes: MEDIUM — confirmed via Microsoft docs (indirect) and general named pipe kernel object semantics
- Windows test behavior in CI: LOW — Windows CI environment not confirmed; tests tagged `//go:build windows` will only run on Windows

**Research date:** 2026-03-24
**Valid until:** 2026-06-24 (stable — go-winio API surface is stable; no breaking changes expected)
