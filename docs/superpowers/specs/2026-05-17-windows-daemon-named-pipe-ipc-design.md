# Windows Daemon Named Pipe IPC Fix

## Summary

AgentHub's Windows daemon path is a named pipe (`\\.\pipe\agenthub-daemon`), but the daemon API server and client currently open that path with Go's Unix socket network. This makes the daemon fail during startup on Windows with:

```text
listen unix \\.\pipe\agenthub-daemon: bind: A socket operation encountered a dead network.
```

The fix is a minimal platform-specific IPC abstraction inside `internal/daemon`. Unix-like platforms keep the existing Unix socket behavior. Windows uses `github.com/tailscale/go-winio` for named pipe listen and dial operations.

## Goals

- Make `agenthub daemon run` start successfully on Windows when using `DefaultSocketPath()`.
- Make daemon clients connect to Windows named pipes through the same `DaemonClient` API.
- Preserve current macOS/Linux Unix socket behavior.
- Add regression coverage before changing production behavior.
- Keep the patch limited to daemon local IPC code.

## Non-Goals

- Do not change GUI, TUI, session management, web sharing, Tailscale, release, or packaging behavior.
- Do not replace local IPC with TCP loopback.
- Do not change the default Windows path away from `\\.\pipe\agenthub-daemon`.
- Do not rewrite broader daemon startup or process management logic.

## Design

Introduce small helpers in `internal/daemon`:

```go
func listenDaemonSocket(path string) (net.Listener, error)
func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error)
func removeDaemonSocket(path string) error
```

The exact helper names may change during implementation if an existing naming pattern fits better.

On macOS and Linux, the helpers preserve the current behavior:

```go
net.Listen("unix", path)
net.Dialer{Timeout: 2 * time.Second}.DialContext(ctx, "unix", path)
os.Remove(path)
```

On Windows, the helpers use named pipes:

```go
winio.ListenPipe(path, nil)
winio.DialPipeContext(ctx, path)
```

`removeDaemonSocket` is a no-op for Windows named pipe paths because named pipes are kernel objects, not filesystem socket files.

`API.Start` will call `listenDaemonSocket` after `ValidateSocketPath` and `CleanupStaleSocket`. `NewDaemonClient` will call `dialDaemonSocket` from its custom HTTP transport. `API.Stop` will close the listener and call `removeDaemonSocket` instead of unconditionally removing `a.ln.Addr().String()`.

## Tests

Add Windows-only regression tests before changing production code:

- `TestAPIStart_WindowsNamedPipeHealth`: start an `API` on a unique `\\.\pipe\agenthub-test-api-*` path, create a `DaemonClient`, call `Health`, and expect success.
- `TestAPIStop_WindowsNamedPipe`: start an `API` on a unique named pipe, stop it, and verify the stop path succeeds without treating the pipe path as a filesystem path.

Existing Unix tests should continue to pass without behavior changes. Existing Windows `CleanupStaleSocket` tests remain useful but are not sufficient because they only cover stale-pipe probing, not the API listener or client dial path.

## Validation

Primary Windows validation:

```powershell
go test -race -short ./internal/...
```

Targeted daemon validation:

```powershell
go test -race -short ./internal/daemon
```

Manual smoke check after tests:

```powershell
.\agenthub.exe daemon run
.\agenthub.exe list --json
```

If the local build path is not used, rely on CI Windows jobs to validate the Wails build and package-level test matrix.

## PR Plan

1. Add failing Windows regression tests for API listen/client health over named pipe.
2. Add platform-specific listen/dial/remove helpers.
3. Wire `API.Start`, `NewDaemonClient`, and `API.Stop` to the helpers.
4. Update comments that currently say only "Unix socket" so they mention Unix socket or Windows named pipe.
5. Run targeted and broader Go validation.
6. Open a PR from `im-alexandre:fix/windows-daemon-named-pipe-ipc` to `scottkw:main` with `Fixes #52`.

## Risks

- `winio.DialPipeContext` availability must be verified against the pinned `github.com/tailscale/go-winio` version. If unavailable, use the package's supported timeout-based dial API with a context-compatible wrapper.
- Tests that create named pipes must use unique pipe names to avoid collisions across parallel Windows test runs.
- `CleanupStaleSocket` already treats any named pipe dial error as absent or stale; this remains unchanged unless tests reveal a busy-pipe false negative.
