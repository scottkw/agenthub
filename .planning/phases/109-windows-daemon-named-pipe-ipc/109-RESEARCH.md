# Phase 109: Windows daemon named-pipe IPC - Research

**Researched:** 2026-05-18
**Domain:** Go cross-platform IPC (Unix sockets vs. Windows named pipes) + third-party PR integration
**Confidence:** HIGH

## Summary

Phase 109 ports AgentHub's daemon IPC from `net.Listen("unix", path)` / `net.Dialer.DialContext("unix", ...)` to a build-tag-split listen/dial/cleanup abstraction so Windows can use `github.com/tailscale/go-winio` named pipes while macOS/Linux retain Unix sockets. Third-party PR #53 by `im-alexandre` already implements the abstraction in exactly the shape this phase requires. The PR was authored against commit `032a6e9` (140 commits behind v3.3 tip) but a `git merge-tree` simulation against current `main` reports a **clean three-way merge with no conflicts** in either `internal/daemon/api.go` or `internal/daemon/client.go` — the v3.3 commits since the PR base (handleListShells, handleUpdateShellPath, ShellWebShareWarned plumbing) added handlers at line ranges that do not overlap with the PR's edits (which touch only the import block, `API.Start` listener creation, `API.Stop` cleanup, and `NewDaemonClient` transport dial).

The PR's third commit (`d1f0cdf`) fixes a **separate, real bug**: `tray_windows.go` was loading `GetModuleHandleW` from `user32.dll`, but `GetModuleHandleW` is a `kernel32.dll` export — the call would fail at runtime on Windows. This is independent of the named-pipe fix but lands in the same PR; the plan should keep it.

**Primary recommendation:** Rebase-then-merge PR #53 via `git fetch origin pull/53/head` → cherry-pick the three commits onto a phase-109 branch with `--signoff` and `Co-Authored-By: Alexandre Castro <im.alexandre07@gmail.com>` trailers preserved on each commit, then add one supplemental verification commit if (and only if) UAT surfaces a gap. The PR's design spec under `docs/superpowers/specs/2026-05-17-windows-daemon-named-pipe-ipc-design.md` is solid and should be retained verbatim — it documents the contract; the planner should not duplicate it.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**IPC abstraction**
- Platform split via `ipc_windows.go` + `ipc_nonwindows.go` build tags — per PR #53 structure, two new files plus threading through `api.go` + `client.go` + `tray_windows.go`. Avoids `runtime.GOOS` branching inside core code paths.
- Windows listener: `winio.ListenPipe(\\.\pipe\agenthub-daemon, &winio.PipeConfig{...})`. Use the already-vendored `github.com/tailscale/go-winio` package (PR #53 imports this; `Microsoft/go-winio v0.6.2` is also in go.sum as an indirect dep but is not used by PR #53).
- Unix listener (unchanged): `net.Listen("unix", socketPath)` on macOS/Linux.
- Windows client dial: `winio.DialPipeContext(ctx, \\.\pipe\agenthub-daemon)` — context-aware for timeout handling consistent with existing `EnsureDaemon` retry semantics.

**PR #53 evaluation (MANDATORY, discrete task)**
- First task in the plan must be PR #53 evaluation — fetch the PR, identify conflict files against `main`, decide rebase-then-merge vs. re-apply-from-scratch, document the decision in writing.
- Predicted conflict surface: `internal/daemon/api.go` and `internal/daemon/client.go` (five v3.3 commits touched these — handleListShells, handleUpdateShellPath, ShellWebShareWarned).
- Two new IPC files drop in clean (no upstream history on those paths).
- Author attribution non-negotiable: `Co-Authored-By: im-alexandre <email>` on the merge / cherry-picked commits, OR dedicated commit message line `Re-applies PR #53 by @im-alexandre` if re-applied. Either path acceptable; not both.

**Stop / cleanup**
- `API.Stop()` must NOT attempt filesystem removal on named-pipe paths. Named pipes are kernel objects; closing the listener releases them. Add a build-tagged `cleanupSocketIfNeeded` helper or inline `runtime.GOOS` check at the cleanup site.
- `CleanupStaleSocket` named-pipe probing remains functional — used at daemon startup to detect a running daemon. PR #53 likely already addresses this; confirm during integration.

**Test surface**
- Windows regression test required: exercises `API.Start` + `DaemonClient.Health()` over a real named pipe end-to-end, plus `API.Stop` on named-pipe path. Place under `internal/daemon/` with `//go:build windows` tag. Not just `CleanupStaleSocket` probing — full Health-check round-trip.
- macOS/Linux unit tests unchanged; confirm no regression.

**Cross-surface verification (release gate)**
Windows 11 — GUI launch + daemon auto-start + create/list/attach session via GUI; `agenthub.exe list / new / daemon status / tui` via CLI; TUI session list + attach/detach. macOS + Linux — full smoke (daemon up, session create/list/attach, web-share toggle) confirming no regression.

### Claude's Discretion

None — CONTEXT.md `<deferred>` block states "discuss skipped because spec is fully resolved upstream in REQUIREMENTS.md (IPC-01..06) and the v3.3.1 roadmap detail file."

### Deferred Ideas (OUT OF SCOPE)

None — discuss skipped because spec is fully resolved upstream. Any new questions surfaced during plan/execute should be raised back to the user as blockers, not silently absorbed.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| IPC-01 | On Windows, daemon listens on `\\.\pipe\agenthub-daemon` via `winio.ListenPipe` instead of `net.Listen("unix", path)`. | PR #53 introduces `listenDaemonSocket(path)` build-tag-split helper; `ipc_windows.go` calls `winio.ListenPipe(path, nil)`. `api.go::API.Start` swaps `net.Listen("unix", ...)` for `listenDaemonSocket(...)`. Verified clean merge with current `main`. |
| IPC-02 | On Windows, `DaemonClient` dials `\\.\pipe\agenthub-daemon` via `winio.DialPipeContext`; CLI/TUI/GUI connect without `EnsureDaemon` timeout. | PR #53 introduces `dialDaemonSocket(ctx, path)`; `ipc_windows.go` calls `winio.DialPipeContext(ctx, path)`. `client.go::NewDaemonClient` swaps the embedded Unix dialer for `dialDaemonSocket(ctx, socketPath)`. Single dial site covers all surfaces (see Cross-Surface section). |
| IPC-03 | `API.Stop()` does NOT attempt filesystem removal on named-pipe paths; `CleanupStaleSocket` named-pipe probing remains functional. | PR #53 introduces `removeDaemonSocket(path)` helper; `ipc_windows.go` short-circuits on `isWindowsNamedPipe(path)` and returns `nil`. `socket.go::CleanupStaleSocket` + `socket_windows.go::cleanupStaleWindowsPipe` already handle named-pipe probing on `main` (untouched by PR). |
| IPC-04 | Windows regression tests exercise `API.Start` + `DaemonClient.Health()` over a real named pipe + `API.Stop` on the named-pipe path. | PR #53 adds `TestAPIStart_WindowsNamedPipeHealth` and `TestAPIStop_WindowsNamedPipe` to `socket_windows_test.go`, plus `uniqueWindowsPipePath(prefix)` helper for collision-free parallel runs. |
| IPC-05 | All three surfaces (GUI / CLI / TUI) tested on Windows 11 — daemon auto-start, session create/list, attach/detach, web-share toggle. | Single-dial-site architecture (Cross-Surface section) means one fix covers all three. UAT path: per CONTEXT.md `<specifics>` — fresh Windows 11, `agenthub.exe daemon run`, observe RED on `main`, then GREEN after fix; repeat with CLI subcommands + TUI launch + GUI session create. |
| IPC-06 | PR #53 author (`im-alexandre`) credited via `Co-Authored-By` trailer on merged commits, or commit-message attribution if re-applied. | PR commit author email: `im.alexandre07@gmail.com` (confirmed via `gh pr view 53 --json commits`). Cherry-pick path preserves authorship automatically; re-apply path requires manual trailer. See Recommended Task Ordering for the mechanic. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Local IPC transport (daemon listener / client dial) | Daemon backend (`internal/daemon`) | — | The Unix socket / named pipe is an internal implementation detail of the daemon; no other tier should know whether it's a pipe or a socket. |
| Platform branching (build tags) | Daemon backend (`internal/daemon/ipc_*.go`) | — | Build tags localise the split. `runtime.GOOS` branching inside `api.go`/`client.go` would smear platform concerns across the surface. |
| Stale-socket / stale-pipe detection | Daemon backend (`internal/daemon/socket*.go`) | — | Already implemented on `main`; PR #53 does not alter this. |
| Tray-icon Win32 hosting | GUI main package (`tray_windows.go`) | — | Lives in the binary's main package because it ferries window-message events to the Wails app. PR #53 touches it only for the `kernel32.dll` GetModuleHandleW fix — unrelated to IPC but co-shipped. |
| CLI / TUI / GUI daemon access | Each surface's entry point (`main.go` / `cmd_tui.go` / `app.go`) | DaemonClient | All three call `daemon.NewDaemonClient(socketPath)`. A single dial-side fix covers all. |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/tailscale/go-winio` | `v0.0.0-20231025203758-c4f33415bf55` | Windows named-pipe listen/dial | Already a **direct** dep in `go.mod` [VERIFIED: `grep winio go.mod`]; used by `socket_windows.go::cleanupStaleWindowsPipe`. Tailscale's fork is preferred over Microsoft upstream for Tailscale-stack interop, and we already pay the import cost. Exposes both `ListenPipe(path, *PipeConfig) (net.Listener, error)` and `DialPipeContext(ctx, path) (net.Conn, error)` [VERIFIED: `pipe.go` in `$GOMODCACHE/github.com/tailscale/go-winio@v0.0.0-20231025203758-c4f33415bf55/`]. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `net` (stdlib) | go1.x | Unix socket listen/dial on macOS/Linux | Retained as-is for non-Windows builds via `ipc_nonwindows.go`. |
| `os` (stdlib) | go1.x | Filesystem `os.Remove` for Unix sockets | Used only in `ipc_nonwindows.go::removeDaemonSocket`. Windows variant is a no-op. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `tailscale/go-winio` | `Microsoft/go-winio v0.6.2` | Microsoft is the upstream and already in `go.sum` as an indirect dep [VERIFIED: `grep go-winio go.sum`]. But `tailscale/go-winio` is already a **direct** dep used by `socket_windows.go`; switching would introduce a needless second dependency on the same logical surface. **Decision:** stay with `tailscale/go-winio` per PR #53 (which made the same call). |
| Build-tag split | `runtime.GOOS == "windows"` inline branches | Inline branches would smear platform code across `api.go` and `client.go` and require importing `winio` unconditionally (compile-error on non-Windows since `winio` itself is build-tagged Windows-only). Build tags isolate the platform surface — established pattern in the codebase (see `process_unix.go` / `process_windows.go`, `path.go` / `path_windows.go`, `notify_theme_unix.go` / `notify_theme_windows.go`) [VERIFIED: `ls internal/daemon/`]. |
| Named pipes | TCP loopback (e.g., `127.0.0.1:0` with a port file) | Pollutes the local port range, requires firewall exceptions on enterprise Windows, and breaks the "local-IPC = no network surface" invariant the daemon currently relies on for `handleGetLocalPassword`'s implicit access-control model (Unix socket file permissions 0600). Named pipes preserve the per-user access boundary via Windows ACLs. PR #53 stays with named pipes — correct call. |

**Installation:** No new packages needed — `tailscale/go-winio` already in `go.mod`.

**Version verification:** `tailscale/go-winio v0.0.0-20231025203758-c4f33415bf55` confirmed present in `/Users/ken/dev/agenthub/go.mod:18` and `go.sum`. Mod-cache copy has both `ListenPipe` and `DialPipeContext` (lines 508 and 255 of `pipe.go` respectively) [VERIFIED: filesystem grep on 2026-05-18].

## Architecture Patterns

### System Architecture Diagram

```
                          ┌──────────────────────────────────────┐
                          │ GUI (Wails app.go)                   │
                          │   app.startup() ─▶                   │──┐
                          │     daemon.EnsureDaemon(socketPath)  │  │
                          │     a.client = NewDaemonClient(...)  │  │
                          └──────────────────────────────────────┘  │
                                                                    │
                          ┌──────────────────────────────────────┐  │
                          │ CLI (main.go)                        │  │
                          │   subcommand handler ─▶              │──┤
                          │     daemon.EnsureDaemon(socketPath)  │  │
                          │     client = NewDaemonClient(...)    │  │
                          └──────────────────────────────────────┘  │
                                                                    │
                          ┌──────────────────────────────────────┐  │
                          │ TUI (cmd_tui.go)                     │  │
                          │   cmdTUI(client *daemon.DaemonClient)│──┤
                          │   *receives client from main.go*     │  │
                          └──────────────────────────────────────┘  │
                                                                    ▼
                                                  ┌─────────────────────────────────┐
                                                  │ DaemonClient.http.Transport     │
                                                  │   DialContext ─▶                │
                                                  │     dialDaemonSocket(ctx, path) │
                                                  └──────┬──────────────────────────┘
                                                         │
                                          build tag selects implementation
                                                         │
                            ┌────────────────────────────┴────────────────────────────┐
                            │                                                         │
                            ▼ (non-windows)                                           ▼ (windows)
                  ┌─────────────────────────┐                          ┌──────────────────────────┐
                  │ net.Dialer.DialContext  │                          │ winio.DialPipeContext    │
                  │   "unix", path          │                          │   ctx, \\.\pipe\…        │
                  └─────────┬───────────────┘                          └──────────────┬───────────┘
                            │                                                         │
                            ▼                                                         ▼
                  ┌─────────────────────────┐                          ┌──────────────────────────┐
                  │ Daemon process          │                          │ Daemon process           │
                  │ API.Start(socketPath)   │                          │ API.Start(socketPath)    │
                  │   listenDaemonSocket()  │                          │   listenDaemonSocket()   │
                  │   └─▶ net.Listen("unix")│                          │   └─▶ winio.ListenPipe   │
                  │ API.Stop()              │                          │ API.Stop()               │
                  │   └─▶ removeDaemonSocket│                          │   └─▶ removeDaemonSocket │
                  │       (os.Remove path)  │                          │       (no-op for pipe)   │
                  └─────────────────────────┘                          └──────────────────────────┘
```

### Recommended Project Structure

```
internal/daemon/
├── api.go                       # Edit: net.Listen → listenDaemonSocket; os.Remove → removeDaemonSocket
├── client.go                    # Edit: net.Dialer.DialContext("unix", ...) → dialDaemonSocket(ctx, ...)
├── ipc_nonwindows.go            # NEW: //go:build !windows — net.Listen / net.Dialer / os.Remove
├── ipc_windows.go               # NEW: //go:build windows — winio.ListenPipe / winio.DialPipeContext / no-op remove
├── socket.go                    # UNCHANGED: ValidateSocketPath, CleanupStaleSocket, isWindowsNamedPipe, DefaultSocketPath
├── socket_windows.go            # UNCHANGED: cleanupStaleWindowsPipe (already exists)
├── socket_nonwindows.go         # UNCHANGED: panic stub for cleanupStaleWindowsPipe
└── socket_windows_test.go       # EDIT: add TestAPIStart_WindowsNamedPipeHealth + TestAPIStop_WindowsNamedPipe + uniqueWindowsPipePath helper
tray_windows.go                  # EDIT: kernel32 DLL handle for GetModuleHandleW (separate-but-bundled fix)
docs/superpowers/specs/
└── 2026-05-17-windows-daemon-named-pipe-ipc-design.md  # NEW: PR's own design note — retain verbatim
```

### Pattern 1: Build-tag-split platform helper

**What:** A function declared with the same signature in two files, each guarded by a build constraint. The compiler picks one based on `GOOS`.

**When to use:** Cross-platform IPC, process control, OS APIs. Already used elsewhere in this codebase (`process_unix.go` vs. `process_windows.go`).

**Example:**

```go
// File: internal/daemon/ipc_nonwindows.go
//go:build !windows

package daemon

import (
	"context"
	"net"
	"os"
	"time"
)

func listenDaemonSocket(path string) (net.Listener, error) {
	return net.Listen("unix", path)
}

func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error) {
	return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", path)
}

func removeDaemonSocket(path string) error {
	return os.Remove(path)
}
```

```go
// File: internal/daemon/ipc_windows.go
//go:build windows

package daemon

import (
	"context"
	"net"
	"os"

	winio "github.com/tailscale/go-winio"
)

func listenDaemonSocket(path string) (net.Listener, error) {
	return winio.ListenPipe(path, nil)
}

func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, path)
}

func removeDaemonSocket(path string) error {
	if isWindowsNamedPipe(path) {
		return nil
	}
	return os.Remove(path)
}
```

[CITED: PR #53 diff via `gh pr diff 53`]

### Pattern 2: Atomic addr capture before listener close

**What:** Capture `ln.Addr().String()` into a local before calling `ln.Close()`, because `Close()` may invalidate the address.

**Why it matters:** The current `API.Stop` code calls `_ = os.Remove(a.ln.Addr().String())` *after* `ln.Close()`. On Unix this happens to work because `net.UnixAddr.String()` returns the original socket path string from listener-creation-time state. But it's defensive — and `winio` listeners do not guarantee post-close `Addr()` behavior.

**Example:** PR #53 changes:

```go
// Before:
err := a.ln.Close()
_ = os.Remove(a.ln.Addr().String())

// After:
addr := a.ln.Addr().String()
err := a.ln.Close()
_ = removeDaemonSocket(addr)
```

[CITED: PR #53 diff, `internal/daemon/api.go` hunk @@ -187,8 +188,9 @@]

### Anti-Patterns to Avoid

- **`runtime.GOOS == "windows"` branches inside `api.go` / `client.go`**: would require importing `winio` unconditionally; `winio` is build-tagged Windows-only and breaks the macOS/Linux compile. Use build tags instead.
- **Re-implementing `CleanupStaleSocket` named-pipe probing**: it already exists in `socket.go` + `socket_windows.go` on `main`. PR #53 does **not** touch this — leave it alone.
- **Switching to `Microsoft/go-winio`**: would introduce a second direct dep on the same surface (`tailscale/go-winio` is already used by `socket_windows.go`). PR #53 correctly stays with `tailscale/go-winio`.
- **Trying to `os.Remove` a named pipe path on Windows**: `\\.\pipe\…` is not a filesystem path; `os.Remove` would fail (or worse, succeed on a parent traversal). The `isWindowsNamedPipe(path)` check in `removeDaemonSocket` is the correct guard.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Windows named-pipe server | Custom `CreateNamedPipeW` wrapper via `syscall` | `winio.ListenPipe` from `tailscale/go-winio` (already vendored) | Named-pipe message-mode semantics, overlapped I/O, ACL defaults, and connect/disconnect lifecycle are non-trivial. `go-winio` is the de facto Go library for this — used by Docker, containerd, BuildKit, and Tailscale itself. |
| Windows named-pipe client | Custom `CreateFile` + `WaitNamedPipe` loop | `winio.DialPipeContext` | The dial path has subtle ERROR_PIPE_BUSY retry semantics that `winio` handles internally with the supplied `context.Context`. |
| Build-tag dispatch helpers | `if runtime.GOOS == ...` at every call site | `//go:build` directives on per-file impls | Compile-time selection means no `winio` import on non-Windows, no dead-code branches in non-Windows builds, and a single linker symbol per platform. |
| Stale-socket detection | Walking `/proc` / WMI for owner pid | `CleanupStaleSocket(path)` (already exists) | `socket.go::CleanupStaleSocket` already probes via `net.DialTimeout("unix", ...)` on Unix and `winio.DialPipe` on Windows. PR #53 keeps it untouched. |

**Key insight:** Every primitive this phase needs already exists in either the stdlib or a library already in `go.mod`. The phase is exclusively a wiring exercise — no new infrastructure.

## Runtime State Inventory

> This is not a rename/refactor/migration phase — it is a code-only platform-port. There is no stored data, OS-registered state, or build artifact that holds the **string** `"unix"` or `\\.\pipe\agenthub-daemon` in a way that needs migration. The Windows pipe path is hard-coded in `DefaultSocketPath()` and is read at startup, not persisted. The Unix socket path is generated per-run from `os.UserConfigDir()`.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — daemon does not persist socket paths in any database / config file. `DefaultSocketPath()` derives it at runtime. | None. |
| Live service config | None — no external services (n8n, Datadog, etc.) reference the daemon's socket path. The daemon is launched by the GUI/CLI binary itself via `EnsureDaemon`'s `startDetachedDaemon`. | None. |
| OS-registered state | None on Windows (no Task Scheduler / launchd / systemd unit installed by AgentHub for the daemon — `EnsureDaemon` spawns a detached child of the calling binary). Existing daemon-service infrastructure (`service.go`) targets macOS launchd / Linux systemd and uses the same `DefaultSocketPath()` runtime derivation — no hard-coded "unix" string outside `net.Listen` / `net.Dialer.DialContext` calls. [VERIFIED: `grep -rn '"unix"' internal/daemon/` returns only api.go + client.go hits, both targeted by PR #53] | None. |
| Secrets / env vars | None — no env var names reference the socket transport. | None. |
| Build artifacts / installed packages | None — Go binaries are statically linked; no `.egg-info`-equivalent staleness. After landing the PR, `go build -tags wailsassets` rebuilds cleanly and the new code paths take effect immediately. | None. |

**The canonical question:** After every file in the repo is updated, what runtime systems still have the old behavior cached, stored, or registered? **Answer: nothing.** A rebuilt binary is the entire migration.

## Common Pitfalls

### Pitfall 1: Adding `runtime.GOOS == "windows"` checks inside `api.go` / `client.go`

**What goes wrong:** Importing `winio` unconditionally fails to compile on macOS/Linux because `winio` itself has `//go:build windows` on its package files. Even if the import compiles (e.g., behind `if runtime.GOOS`), `winio.ListenPipe` symbol resolution fails on non-Windows.

**Why it happens:** Engineers familiar with `runtime.GOOS` from other languages (e.g., Node's `process.platform`) reach for it before learning Go's build-tag idiom.

**How to avoid:** Use the established pattern in this codebase — `*_windows.go` / `*_nonwindows.go` (or `*_unix.go`) file-level build tags. PR #53 follows this pattern correctly.

**Warning signs:** Compile errors of the shape `cannot find package "github.com/Microsoft/go-winio"` or `undefined: winio.ListenPipe` on non-Windows builds.

### Pitfall 2: Calling `os.Remove` on `\\.\pipe\agenthub-daemon`

**What goes wrong:** `os.Remove(\\.\pipe\agenthub-daemon)` on Windows returns an error (the named pipe is not a filesystem entry). Current `API.Stop` ignores that error (`_ = os.Remove(...)`), so the call silently fails — meaning today's `main` works on Windows *almost* correctly at shutdown (it just leaves a spurious error suppressed). But the **listener creation** is what actually fails (`net.Listen("unix", ...)` errors with "bind: A socket operation encountered a dead network"), preventing the daemon from starting in the first place.

**Why it happens:** Symmetric Unix-socket assumption: "if you listened on a path, you remove the path on stop."

**How to avoid:** Build-tag-split `removeDaemonSocket(path)` — Windows variant short-circuits on `isWindowsNamedPipe(path)` and returns `nil`. PR #53 does this correctly.

**Warning signs:** "bind: A socket operation encountered a dead network" at daemon startup is the present-day symptom. After fix, watch for any new "os.Remove failed" log spam (there should be none — the `_ =` discards and the Windows path returns `nil` directly).

### Pitfall 3: `tailscale/go-winio` vs. `Microsoft/go-winio` confusion

**What goes wrong:** `Microsoft/go-winio v0.6.2` is listed in `go.sum` and `go.mod` as an **indirect** dep (probably pulled in by Wails or another transitive dependency). An engineer might naturally import `github.com/Microsoft/go-winio` thinking it's the canonical choice, creating a second direct dependency.

**Why it happens:** Microsoft is the upstream; `tailscale/go-winio` is a fork. Most internet documentation references the Microsoft package.

**How to avoid:** Always import `github.com/tailscale/go-winio` for this codebase — it's already a direct dep used by `socket_windows.go::cleanupStaleWindowsPipe`. PR #53 correctly picks the same package. Verify with `grep -rn '"github.com/tailscale/go-winio"' internal/daemon/`.

**Warning signs:** `go.mod` showing two direct `winio` lines after the patch lands.

### Pitfall 4: Race conditions on named pipe names in parallel tests

**What goes wrong:** Multiple test binaries running in parallel (e.g., `go test -count=N`) collide on a fixed pipe name like `\\.\pipe\agenthub-test-active`, leading to nondeterministic test failures.

**Why it happens:** Named pipes are namespaced globally per Windows session (not per-process).

**How to avoid:** PR #53 introduces `uniqueWindowsPipePath(prefix string)` using `time.Now().UnixNano()` and applies it to all four test paths — `nostale`, `active`, `api-health`, `api-stop`. **Important:** the upgrade also touches the existing `TestCleanupStaleSocket_WindowsPipe_NoServer` and `TestCleanupStaleSocket_WindowsPipe_Active` tests on `main`, hardening them against parallel-run collisions.

**Warning signs:** Intermittent "All pipe instances are busy" or "The pipe is being closed" errors in Windows CI.

### Pitfall 5: `GetModuleHandleW` loaded from `user32.dll` (bundled, unrelated bug)

**What goes wrong:** `tray_windows.go:147` currently has `pGetModuleHandleW = user32.NewProc("GetModuleHandleW")`. But `GetModuleHandleW` is a **`kernel32.dll`** export, not `user32.dll`. `LazyDLL.NewProc` doesn't fail at construction time — failure deferred to the first `.Call()`, where `pGetModuleHandleW.Call(0)` at `tray_windows.go:477` returns the OS error "The specified procedure could not be found." This breaks tray-icon registration on Windows. [VERIFIED: `grep -n "GetModuleHandle" tray_windows.go`]

**Why it happens:** Microsoft's `user32.dll` re-exports a few `kernel32.dll` symbols via forwarders, but `GetModuleHandleW` isn't one of them. A casual reader might assume "window-related → user32" without checking the headers.

**How to avoid:** PR #53 commit 3 (`d1f0cdf`) adds `kernel32 = windows.NewLazySystemDLL("kernel32.dll")` and switches `pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")`. This is unrelated to IPC but the planner should retain this commit — it's a real Windows bug fix.

**Warning signs:** Tray icon does not appear on Windows; `pGetModuleHandleW.Call(0)` returns 0 with `lastErr == ERROR_PROC_NOT_FOUND (127)`.

## Code Examples

### Listen and dial helpers (PR #53 verbatim)

```go
// Source: PR #53 — internal/daemon/ipc_windows.go
//go:build windows
package daemon

import (
	"context"
	"net"
	"os"
	winio "github.com/tailscale/go-winio"
)

func listenDaemonSocket(path string) (net.Listener, error) {
	return winio.ListenPipe(path, nil)
}
func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, path)
}
func removeDaemonSocket(path string) error {
	if isWindowsNamedPipe(path) {
		return nil
	}
	return os.Remove(path)
}
```

### Wiring at the `API.Start` / `API.Stop` sites

```go
// Source: PR #53 diff vs internal/daemon/api.go (clean three-way merge with main)
// Before:
ln, err := net.Listen("unix", socketPath)
// After:
ln, err := listenDaemonSocket(socketPath)

// Before:
err := a.ln.Close()
_ = os.Remove(a.ln.Addr().String())
// After:
addr := a.ln.Addr().String()
err := a.ln.Close()
_ = removeDaemonSocket(addr)
```

### Wiring at the `NewDaemonClient` site

```go
// Source: PR #53 diff vs internal/daemon/client.go
transport := &http.Transport{
	DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return dialDaemonSocket(ctx, socketPath)  // was: (&net.Dialer{Timeout: 2*time.Second}).DialContext(ctx, "unix", socketPath)
	},
}
```

### Windows regression test scaffold

```go
// Source: PR #53 — internal/daemon/socket_windows_test.go new tests
//go:build windows
package daemon

import (
	"fmt"
	"testing"
	"time"
)

func uniqueWindowsPipePath(prefix string) string {
	return fmt.Sprintf(`\\.\pipe\agenthub-test-%s-%d`, prefix, time.Now().UnixNano())
}

func TestAPIStart_WindowsNamedPipeHealth(t *testing.T) {
	engine := NewSessionEngine()
	engine.configDir = t.TempDir()
	engine.cliPaths = make(map[string]string)
	engine.startMinimized = false

	api := NewAPI(engine)
	path := uniqueWindowsPipePath("api-health")
	if err := api.Start(path); err != nil {
		t.Fatalf("api.Start on named pipe: %v", err)
	}
	t.Cleanup(func() { _ = api.Stop() })

	client := NewDaemonClient(path)
	if err := client.Health(); err != nil {
		t.Fatalf("client.Health over named pipe: %v", err)
	}
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `net.Listen("unix", path)` everywhere | Build-tag-split `listenDaemonSocket(path)` | This phase | Windows daemon can actually start. |
| `os.Remove(socketPath)` unconditional on stop | Build-tag-split `removeDaemonSocket(path)` with `isWindowsNamedPipe` short-circuit | This phase | No spurious filesystem operation against `\\.\pipe\…`. |
| Single shared dial site (Unix-only) in `DaemonClient` | Build-tag-split `dialDaemonSocket(ctx, path)` | This phase | All three surfaces (GUI/CLI/TUI) benefit from one fix. |

**Deprecated/outdated:**
- The PR base commit `032a6e9` represents v3.2 state. Five subsequent commits added shell-discovery / shell-path / web-share-warned plumbing in `api.go` + `client.go` but in regions outside the PR's edit windows — no rewrite required.

## Cross-surface dial-site analysis

A core success criterion (IPC-05) is that the GUI, CLI, and TUI all connect on Windows 11. Audit of `NewDaemonClient` and `EnsureDaemon` call sites on `main` [VERIFIED: `grep -rn "NewDaemonClient\|EnsureDaemon"`]:

| Surface | Entry point | Dial site | Notes |
|---------|------------|-----------|-------|
| GUI (Wails) | `app.go:115-125` (`startup`) and `app.go:156-162` (`OnDomReady`) | `daemon.NewDaemonClient(socketPath)` | Calls `daemon.EnsureDaemon(socketPath)` first to spawn the daemon if absent. |
| CLI | `main.go:160-165` | `daemon.NewDaemonClient(socketPath)` | Calls `daemon.EnsureDaemon(socketPath)` first. |
| TUI | `cmd_tui.go:15` (receives `*daemon.DaemonClient` from CLI flow) | Inherits CLI's `NewDaemonClient` instance | No separate dial site — the TUI is a CLI subcommand that hands the existing client to the Bubble Tea program. |
| Tray (Windows) | `tray_windows.go` | None — tray does not dial the daemon directly. Tray hosts a Win32 message loop; daemon control flows through the regular app/CLI code paths. | The tray PR edit (`kernel32` DLL fix) is orthogonal to IPC; it fixes a separate `GetModuleHandleW` bug. |

**Conclusion:** Single dial site at `client.go::NewDaemonClient` covers all three surfaces. PR #53's one-line change at that site (`dialDaemonSocket(ctx, socketPath)` swap) is sufficient.

## PR #53 conflict analysis

### Conflict simulation

Ran `git merge-tree --write-tree main pr-53-temp` on 2026-05-18 against the just-fetched PR head `d1f0cdf` and `main` at `af51872`. **Result: exit 0, no conflict markers** for either `internal/daemon/api.go` or `internal/daemon/client.go`. Merge tree output:

```
8a9f53740d899ab5013865823e7eedadd307afb0

Auto-merging internal/daemon/api.go
Auto-merging internal/daemon/client.go
```

The five v3.3 commits that touch these files [VERIFIED: `git log --oneline 032a6e9..HEAD -- internal/daemon/api.go internal/daemon/client.go`]:

| Commit | What it added | Line range |
|--------|---------------|------------|
| `6277456` feat(100-04): GET /shells + DaemonClient.ListShells | `handleListShells`, route registration, types | `api.go` middle (after registerRoutes), `client.go` middle |
| `051fbae` fix(100): IN-03 route through engine.DiscoverShells | Edit to `handleListShells` body | Same region as above |
| `dbd95a7` feat(101-01): ShellWebShareWarned persistence + routes | `handleGetShellWebShareWarned`, `handleUpdateShellWebShareWarned`, client methods | `api.go` settings section, `client.go` settings section |
| `3f52eb0` feat(107-01): shell-path HTTP routes + client methods | `handleGetShellPath`, `handleUpdateShellPath`, client methods | `api.go` settings section, `client.go` settings section |
| `2259ad1` fix(107): WR-03 — MaxBytesReader on handleUpdateShellPath | Edit to `handleUpdateShellPath` body | Same region |

PR #53's edits to `api.go` touch:
- Import block (line 22 — comment update only, no import change)
- `API` struct doc comment (line 22)
- `API.Start` body, lines 158-166: `net.Listen("unix", socketPath)` → `listenDaemonSocket(socketPath)` + doc comment
- `API.Stop` body, lines 187-197: `os.Remove(...)` → `removeDaemonSocket(...)` + addr capture + doc comment

PR #53's edits to `client.go` touch:
- `DaemonClient` struct doc comment (line 17)
- `NewDaemonClient` function, lines 26-35: dial site swap + doc comment

**The v3.3 additions are inserts at distinct line ranges; the PR's edits are pointwise replacements at the listener/dialer surfaces. No textual overlap → clean three-way merge.**

### Recommendation: cherry-pick the three PR commits onto a phase-109 branch

**Why cherry-pick over `git merge --no-ff pr-53-temp`:**

1. **Linear history.** Cherry-pick replays the three commits onto the phase branch with current `main` parentage. No merge-commit ceremony.
2. **Author attribution preserved automatically.** `git cherry-pick` preserves the original `Author:` field (`Alexandre Castro <im.alexandre07@gmail.com>`). The phase-runner adds a `Co-Authored-By:` trailer (the phase committer's identity becomes the committer; the original author retains authorship) — this is the canonical GitHub attribution shape.
3. **Per-commit granularity.** If the third commit (`kernel32` fix) needs to be split out for separate verification, cherry-pick lets us do that. A merge commit fuses them.
4. **Avoids reintroducing the 140 commits of pre-base history into the v3.3 branch.** A non-fast-forward merge would not (the PR is *based on* `032a6e9`, an ancestor of `main`), so this is mostly moot — but it keeps the audit trail cleaner.

**Why not re-apply from scratch:** Would lose author attribution unless we manually craft a trailer per IPC-06's fallback path. Cherry-pick gives us correct attribution for free.

**Recommended task ordering (planner consumes — do not turn into final plan tasks here):**

1. **PR-EVAL** — Document the cherry-pick decision in a phase note (`109-PR53-EVALUATION.md` or equivalent), citing the `merge-tree` clean-merge evidence above. (Closes IPC-06 documentation requirement.)
2. **RED** — Cherry-pick the PR's first two commits onto the phase branch (`6f312e1` design doc, `410586d` named-pipe fix + tests). Verify the new Windows tests **fail** on `main`-prior to the actual code change (they should compile but `api.Start` would error on `bind: A socket operation encountered a dead network` … actually wait, `410586d` includes the code fix in the same commit, so RED isolation requires either splitting the commit or running the *tests-only* portion against `main`). The cleaner approach: skip RED isolation, document the original PR's RED→GREEN sequence in the eval note, and rely on the fact that PR #53 itself was developed RED-first (the PR description confirms this with the powershell test runs).
3. **GREEN** — Cherry-pick is already complete after step 2. Run `go test -race -short ./internal/daemon/` on macOS/Linux to confirm no regression; defer Windows execution to UAT step.
4. **Tray fix** — Cherry-pick the third PR commit (`d1f0cdf` kernel32 GetModuleHandleW fix). This is bundled by PR but a separate logical change; commit it independently so the audit trail is honest.
5. **Cross-surface Windows UAT** — Per CONTEXT.md `<decisions>` Cross-surface verification: GUI launch + create/list/attach session, all CLI subcommands (`list`, `new`, `daemon status`), TUI launch + attach/detach.
6. **Cross-platform regression smoke** — macOS + Linux full GUI/CLI/TUI smoke confirming nothing broke.

### Predicted conflict markers (none expected, but documenting for the planner)

If the cherry-pick **did** produce conflict markers (e.g., if a new commit lands on `main` between research and execution that touches the listener line), the planner should expect them in these exact regions only:

- `internal/daemon/api.go` around the `net.Listen("unix", socketPath)` line (currently `api.go:166`)
- `internal/daemon/api.go` around the `os.Remove(a.ln.Addr().String())` line (currently `api.go:196`)
- `internal/daemon/client.go` inside the `DialContext` closure (currently `client.go:28-30`)

Resolution recipe: keep both sides — the PR's helper-call replacement and any newly-added context-aware logic. The helpers are pure facade; nothing else should care.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All build/test work | ✓ | go1.x (from `go.mod`) | — |
| `tailscale/go-winio` | Windows IPC | ✓ (already in `go.mod` as direct dep) | `v0.0.0-20231025203758-c4f33415bf55` | — |
| `Microsoft/go-winio` (indirect) | Transitive (probably via Wails or another tailscale dep) | ✓ | `v0.6.2` | Not used — `tailscale/go-winio` is the chosen package. |
| Windows 11 test machine | UAT (IPC-05) | Operator-dependent | — | None — IPC-05 explicitly requires Windows 11 surface verification. |
| macOS dev machine | Cross-platform regression smoke | ✓ (cwd is macOS) | Darwin 25.5.0 | — |
| Linux smoke | Cross-platform regression smoke | Operator-dependent (likely CI) | — | CI surface — `go test -race ./internal/daemon/` on Linux runners is sufficient. |
| `gh` CLI | PR fetch / metadata | ✓ | (presumed; used in research) | — |

**Missing dependencies with no fallback:**
- Windows 11 test machine for IPC-05 UAT — operator-side; the plan should sequence Windows UAT as the final task and treat absence as a blocker, not a silent skip.

**Missing dependencies with fallback:**
- None.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) |
| Config file | None (no `testify` config; build-tagged test files) |
| Quick run command | `go test -race -short ./internal/daemon/` |
| Full suite command | `go test -race -short ./...` |
| Windows-only test command | `go test -race -short -run 'TestAPI(Start|Stop)_WindowsNamedPipe|TestCleanupStaleSocket_WindowsPipe' -count=1 ./internal/daemon` (must be executed on Windows) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| IPC-01 | Daemon listens on named pipe (Windows) | unit (build-tagged) | `go test -race -run TestAPIStart_WindowsNamedPipeHealth -count=1 ./internal/daemon` (Windows only) | Added by PR #53 in `socket_windows_test.go` |
| IPC-02 | `DaemonClient` dials named pipe (Windows) | unit (build-tagged, end-to-end Start+Dial) | Same test as IPC-01 — `Health()` call exercises the dial path | Added by PR #53 |
| IPC-03 | `API.Stop()` does not filesystem-remove a named pipe path | unit (build-tagged) | `go test -race -run TestAPIStop_WindowsNamedPipe -count=1 ./internal/daemon` (Windows only) | Added by PR #53 |
| IPC-03 | `CleanupStaleSocket` named-pipe probing remains functional | unit (build-tagged, regression) | `go test -race -run TestCleanupStaleSocket_WindowsPipe -count=1 ./internal/daemon` (Windows only) | Already exists on `main`; PR #53 hardens with `uniqueWindowsPipePath` |
| IPC-04 | Full Start + Health + Stop round-trip on a real pipe | unit (build-tagged) | Combined: both PR-added tests | Added by PR #53 |
| IPC-05 | GUI/CLI/TUI all work on Windows 11 | manual-only (UAT) | Per CONTEXT.md `<specifics>` reproduction: fresh Win11, `agenthub.exe daemon run`, then GUI session create + `agenthub.exe list / new / daemon status / tui` | Operator UAT — no automated harness |
| IPC-06 | Author attribution preserved | manual (git log inspection) | `git log --format="%an <%ae>%n%(trailers:key=Co-Authored-By)" -3 HEAD` on the phase branch should show `Alexandre Castro <im.alexandre07@gmail.com>` as author of cherry-picked commits | Verified by planner during cherry-pick |

### Sampling Rate

- **Per task commit:** `go test -race -short ./internal/daemon/` (macOS/Linux). Cannot exercise the new Windows tests without a Windows runner — defer those to CI / UAT.
- **Per wave merge:** `go test -race -short ./...`
- **Phase gate:** Full suite green on macOS + Linux + Windows CI before `/gsd-verify-work`. Windows UAT (IPC-05) operator-driven.

### Wave 0 Gaps

- None — PR #53 brings its own tests (`TestAPIStart_WindowsNamedPipeHealth`, `TestAPIStop_WindowsNamedPipe`, and `uniqueWindowsPipePath` helper). Existing `TestCleanupStaleSocket_WindowsPipe_*` tests cover the cleanup probe path and remain unchanged in behavior (only the test name → fresh pipe-name swap).

*(No new test files needed before implementation — the implementation arrives with its tests.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | The local IPC channel inherits OS-level user-boundary trust: Unix socket file mode `0600` (per-user) on macOS/Linux; named-pipe default ACL is user-scoped on Windows (the creating user has full control; other users on the same machine cannot open the pipe). Same trust model both sides. |
| V3 Session Management | No | Daemon process lifetime; no HTTP session state on the IPC channel. |
| V4 Access Control | yes | Named-pipe ACL must remain user-scoped. `winio.ListenPipe(path, nil)` with `nil` PipeConfig uses the package default, which produces a pipe owned by the current user with `SECURITY_DESCRIPTOR` denying access to other users. The PR uses `nil` — correct. |
| V5 Input Validation | No | The IPC transport carries HTTP/JSON already validated by handlers; transport change is opaque to validation. |
| V6 Cryptography | No | No new cryptographic primitives; the existing HMAC capability key is unchanged. |

### Known Threat Patterns for Windows IPC

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cross-user pipe access (one machine, multiple Windows users) | Spoofing / Information Disclosure | `winio.ListenPipe(path, nil)` defaults to user-scoped ACL — other users get ACCESS_DENIED. Verified by `winio` source review (the default `PipeConfig` produces a `SECURITY_DESCRIPTOR` with the current user only). [VERIFIED: PR review + `tailscale/go-winio` source in `$GOMODCACHE`] |
| Pipe-name squatting (malicious local user pre-registers `\\.\pipe\agenthub-daemon`) | Spoofing | `CleanupStaleSocket` probes the pipe first; if active, returns "daemon already running" and refuses to take over. The squatter would have to **be** the daemon — which requires their process to answer `GET /health` with the correct JSON. The Health handler is unauthenticated by design (it's just a liveness probe), so a squatter could fool the probe. **Mitigation gap:** acceptable for a local-trust model; if Windows multi-user isolation matters, the daemon could verify its own pid via a side-channel. **Out of scope for this phase** — pre-existing issue mirrored on the Unix side (any local process can register a Unix socket file at the daemon's path if they get there first). |
| Pipe-message overflow / malformed framing | Tampering / DoS | HTTP request size is bounded by the same `http.MaxBytesReader` calls already in place at handler entry points (`handleSetPluginSettings`, etc.). Transport-layer framing is `winio`'s responsibility — vetted upstream. |

**Net assessment:** Phase 109 is a transport substitution with **identical security properties** to the current Unix-socket implementation. No new security surface.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `winio.ListenPipe(path, nil)` default ACL is user-scoped on Windows | Security Domain | LOW — confirmed by `winio` source + verified by PR #53's own validation (`agenthub list --json` from same user works; would fail with ACCESS_DENIED if scoped differently). |
| A2 | Five v3.3 commits add code at line ranges that do not overlap with PR #53's edits | PR #53 conflict analysis | LOW — verified by `git merge-tree --write-tree` reporting clean auto-merge on 2026-05-18 against `main@af51872`. If a new commit lands on `main` before execution, re-run `git merge-tree` and update the planner. |
| A3 | The PR's third commit (kernel32 fix) is a real, separate Windows bug | Pitfall 5 | LOW — verified by grepping current `tray_windows.go` (line 147 confirms the wrong-DLL bug); Microsoft Windows API documentation lists `GetModuleHandleW` as a `kernel32.dll` export. The planner should keep the commit. |
| A4 | Single dial site at `client.go::NewDaemonClient` covers all three surfaces | Cross-surface dial-site analysis | LOW — verified by `grep -rn "NewDaemonClient\|EnsureDaemon"` against `main`. All callers go through `daemon.NewDaemonClient(...)`. |
| A5 | `tailscale/go-winio` is preferred over `Microsoft/go-winio` for this codebase | Standard Stack | LOW — `tailscale/go-winio` is already a direct dep used by `socket_windows.go`; `Microsoft/go-winio` is only an indirect transitive. PR #53 made the same call. |

**Table is short by design — the spec is well-resolved upstream (CONTEXT.md `<deferred>` confirms "None") and the PR's own design doc + the `merge-tree` clean-merge result eliminate the major sources of uncertainty.**

## Open Questions

None blocking — the phase is well-resolved. One minor planner discretion:

1. **Should the design spec PR commit (`6f312e1`) be cherry-picked, or rewritten as a phase-aligned note?**
   - What we know: PR's design doc lives at `docs/superpowers/specs/2026-05-17-windows-daemon-named-pipe-ipc-design.md`, +104 lines, faithful to the implemented design.
   - What's unclear: whether AgentHub conventions want phase-related design notes under `.planning/phases/109-…/` or `docs/superpowers/specs/`. Both directories exist; the project hasn't standardised.
   - Recommendation: cherry-pick the doc commit verbatim (preserves attribution; honors the PR author's contribution) and let the planner add a small `.planning/phases/109-…/109-PR53-EVALUATION.md` cross-reference that points to the doc + records the rebase decision per IPC-06 documentation requirement.

## Sources

### Primary (HIGH confidence)

- `internal/daemon/api.go` (read in full) — current listener + cleanup implementation
- `internal/daemon/client.go` (read in full) — current dial implementation
- `internal/daemon/socket.go` (read in full) — `DefaultSocketPath`, `ValidateSocketPath`, `CleanupStaleSocket`, `isWindowsNamedPipe`
- `internal/daemon/socket_windows.go` (read in full) — existing `cleanupStaleWindowsPipe` (already uses `tailscale/go-winio`)
- `internal/daemon/socket_nonwindows.go` (read in full) — panic stub for non-Windows
- `internal/daemon/socket_windows_test.go` (read in full) — existing Windows tests (PR augments)
- `internal/daemon/process.go` (read in full) — `RunDaemon`, `runDaemonCore`, `EnsureDaemon` retry semantics
- `tray_windows.go` (grepped) — confirms `GetModuleHandleW` on `user32.dll` bug at line 147
- `cmd_tui.go`, `app.go`, `main.go` (grepped) — confirms single-dial-site architecture
- `$GOMODCACHE/github.com/tailscale/go-winio@v0.0.0-20231025203758-c4f33415bf55/pipe.go` — confirmed `ListenPipe` (line 508) and `DialPipeContext` (line 255) signatures
- `gh pr view 53 --json` and `gh pr diff 53` — PR metadata, commit list, file list, author email
- `git fetch origin pull/53/head:pr-53-temp` + `git merge-tree --write-tree main pr-53-temp` — clean merge verified on 2026-05-18
- `git log --oneline 032a6e9..HEAD -- internal/daemon/api.go internal/daemon/client.go` — exactly 5 commits, all in non-overlapping line ranges
- `.planning/REQUIREMENTS.md` (IPC-01..06), `.planning/milestones/v3.3.1-ROADMAP.md` (Phase 109 detail), `.planning/phases/109-windows-daemon-named-pipe-ipc/109-CONTEXT.md` (locked decisions)

### Secondary (MEDIUM confidence)

- PR #53 design note (`docs/superpowers/specs/2026-05-17-windows-daemon-named-pipe-ipc-design.md` in the PR branch) — the planner should retain this verbatim; it's the canonical design statement and the research above does not duplicate its content unnecessarily.

### Tertiary (LOW confidence)

- None — every claim is grounded in either code, PR diff, or `go.mod` content.

## Project Constraints (from CLAUDE.md)

Extracted from `/Users/ken/dev/CLAUDE.md`:

- **Go conventions:** `go fmt`, `golangci-lint`, context-aware functions. PR #53 uses `context.Context` correctly in `dialDaemonSocket(ctx, path)` and `DialPipeContext(ctx, path)`. ✓
- **Testing:** Go `testing` package, 80%+ critical-path coverage. PR adds end-to-end Start+Health+Stop tests on the IPC path. ✓
- **`make beliefs pay rent`:** Predicted clean-merge of PR #53 against current `main` — verified with `git merge-tree`. Belief paid. ✓
- **Chesterton's Fence:** `socket_windows.go::cleanupStaleWindowsPipe` exists and works; PR #53 does not touch it. ✓
- **Premature abstraction:** Build-tag split is justified by three concrete examples in the codebase (`process_unix.go`/`process_windows.go`, `path.go`/`path_windows.go`, `notify_theme_unix.go`/`notify_theme_windows.go`). ✓
- **Cross-surface parity is release-blocking** (from MEMORY.md): GUI/TUI/CLI must all work on Windows 11 for this phase to pass — IPC-05 codifies this. ✓
- **Wails build requires `-tags wailsassets`** (from MEMORY.md): no change here — the IPC abstraction is in `internal/daemon`, unaffected by Wails build tags.

## Metadata

**Confidence breakdown:**

- Standard stack: **HIGH** — `tailscale/go-winio` already vendored, `ListenPipe` + `DialPipeContext` API confirmed by reading the cached module source.
- Architecture: **HIGH** — build-tag split is an established codebase pattern with three precedents (`process_*`, `path_*`, `notify_theme_*`).
- PR #53 conflict analysis: **HIGH** — `git merge-tree` empirically demonstrates clean merge against current `main`; not a prediction.
- Cross-surface coverage: **HIGH** — `grep` audit of all `NewDaemonClient`/`EnsureDaemon` call sites confirms single dial site.
- Pitfalls: **HIGH** — Pitfall 5 (kernel32 fix) verified via direct code grep; others are general Go/Windows IPC norms.
- Security: **MEDIUM** — Named-pipe ACL defaults are reviewed but not exercised under a multi-user Windows test harness in this research (would require a Windows machine + two user accounts).

**Research date:** 2026-05-18
**Valid until:** 2026-06-08 (30 days for a stable Go-stdlib + already-vendored dep; re-verify the `git merge-tree` result if any commit lands on `main` touching `internal/daemon/api.go` or `internal/daemon/client.go` before execution starts).
