# Stack Research

**Domain:** Go CLI + background daemon + terminal attach/detach (v1.3 milestone additions)
**Researched:** 2026-03-23
**Confidence:** HIGH (all library versions verified via pkg.go.dev)

---

## Context: What Already Exists (Do NOT Re-add)

The v1.2 codebase already has the following in `go.mod` — none of these need to be added:

| Already Present | Version | Purpose |
|----------------|---------|---------|
| `github.com/aymanbagabas/go-pty` | v0.2.2 | PTY creation and management (cross-platform, ConPTY on Windows) |
| `github.com/coder/websocket` | v1.8.14 | WebSocket relay (nhooyr fork) |
| `github.com/wailsapp/wails/v2` | v2.10.2 | Desktop GUI shell |
| `golang.org/x/sys` | v0.40.0 | Low-level OS syscalls (SIGWINCH, etc.) |
| `tailscale.com` | v1.96.3 | Tailscale integration |
| `github.com/tailscale/go-winio` | indirect via tailscale | Windows named pipes — already in dep graph |
| `golang.org/x/term` | indirect via tailscale/crypto | Terminal raw mode — already in dep graph, needs promotion |

---

## Recommended Stack Additions (v1.3 New Dependencies)

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `github.com/spf13/cobra` | v1.9.1 | CLI command framework | Industry standard used by Kubernetes, Docker, Hugo, and GitHub CLI — 184k+ importers. Subcommand tree maps directly to `agenthub new`, `agenthub list`, `agenthub attach`, `agenthub web start`, etc. Native `context.Context` support, persistent flags, shell completion generation. No viable alternative at this adoption level. Latest release: v1.9.1 (Dec 3, 2025). |
| `github.com/kardianos/service` | v1.2.4 | Service manager integration | The only mature Go library that handles launchd (macOS), systemd (Linux), and Windows Service from one unified API. Generates correct plist/unit files, handles install/uninstall/start/stop. 1,400+ importers, actively maintained (released July 14, 2025). Supports `service.Config{}` with platform-specific overrides for LaunchAgent KeepAlive, systemd WantedBy, etc. |

### Promote to Direct Dependency

| Technology | Current Status | Action | Why |
|------------|---------------|--------|-----|
| `golang.org/x/term` | indirect via tailscale | `go get golang.org/x/term@latest` | Required directly for CLI attach: `term.MakeRaw`/`term.Restore` for raw mode, `term.GetSize` for resize events. Official Go extended library, already in binary. Promoting avoids accidental removal during dep tidy. Latest: v0.30.0+ (Mar 2026). |

### No Additional IPC Library Needed

Do NOT add `github.com/james-barrow/golang-ipc` or any third-party IPC wrapper.

The project already transitively depends on `github.com/tailscale/go-winio` (Windows named pipes). Use stdlib `net.Listen("unix", socketPath)` on macOS/Linux and the already-available `go-winio` for Windows named pipes. This keeps the dependency count minimal and avoids duplicating what Tailscale's `go-winio` already provides.

---

## Installation

```bash
# From the agenthub project root
go get github.com/spf13/cobra@v1.9.1
go get github.com/kardianos/service@v1.2.4
go get golang.org/x/term@latest   # promote from indirect to direct
```

---

## Integration Patterns with Existing Wails Binary

### Pattern 1: Single Binary, Two Entry Points

The existing `main.go` calls `wails.Run()` unconditionally. For v1.3, `main()` must inspect `os.Args` BEFORE calling `wails.Run()`. If a CLI subcommand is detected, cobra handles it and the process exits — `wails.Run()` is never called, no window opens.

```go
func main() {
    // If any non-GUI subcommand is present, run CLI path and exit.
    // Never calls wails.Run() — no window, no WebKit, no GUI overhead.
    if len(os.Args) > 1 && !isWailsInternalArg(os.Args[1]) {
        cli.Execute()  // cobra root command; os.Exit on completion
        return
    }
    // GUI path — unchanged from v1.2
    err := wails.Run(&options.App{ ... })
}
```

**Wails dev mode caveat (from wails/discussions/4175):** During `wails dev`, the binary is invoked twice — second time with `/tmp/wailsbindings` as an arg. The `isWailsInternalArg` guard must pass through these internal args. Use a `//go:build production` build tag to make arg inspection active only in `wails build` output, not in `wails dev`.

### Pattern 2: Daemon Reuses Existing Internal Packages

When invoked as `agenthub daemon` (or via service manager), the binary runs headlessly — `wails.Run()` is never called. The `SessionRegistry`, `HubManager`, and `WebServer` subsystems already live in `internal/` with no GUI dependency and can be reused directly:

```
agenthub daemon
  └── IPC socket listener (unix socket on macOS/Linux, named pipe on Windows)
  └── internal/pty.SessionRegistry  (no change needed)
  └── internal/relay.HubManager     (no change needed)
  └── internal/webserver.WebServer  (no change needed)
  └── Tailscale TLS via existing local.Client.GetCertificate
```

### Pattern 3: IPC Socket for CLI-to-Daemon Communication

The CLI subcommands (`list`, `attach`, `kill`, `rename`, etc.) communicate with the running daemon via a local socket.

**macOS/Linux:**
```go
socketPath := filepath.Join(os.UserCacheDir(), "agenthub", "daemon.sock")
ln, err := net.Listen("unix", socketPath)
```

**Windows:** `tailscale/go-winio` is already in the dep graph. Import it directly:
```go
// Windows build tag file
import "github.com/tailscale/go-winio"

ln, err := winio.ListenPipe(`\\.\pipe\agenthub-daemon`, nil)
```

Abstract behind a single `NewDaemonListener(path string) (net.Listener, error)` function gated by `//go:build` tags — one per platform, same call site.

**Protocol:** stdlib `encoding/json` over the socket is sufficient for the ~15 command types (`new`, `list`, `attach`, `kill`, `rename`, `web start/stop/status`, `health`, `qr`, `settings`). No gRPC or protobuf needed.

### Pattern 4: Terminal Attach (PTY Proxy over IPC Socket)

The `agenthub attach <id>` command:

1. Dial daemon IPC socket; send `{"op": "attach", "session_id": "..."}`.
2. Daemon finds the session's Hub, registers a new subscriber (same subscriber interface used by WebSocket relay today).
3. Raw PTY bytes stream over the socket.
4. CLI client calls `term.MakeRaw(int(os.Stdin.Fd()))` to enter raw mode.
5. Two goroutines: `io.Copy(conn, os.Stdin)` and `io.Copy(os.Stdout, conn)`.
6. On detach prefix (configurable, default `Ctrl+\` = `0x1c`), CLI intercepts the byte in the stdin copy loop, sends `{"op": "detach"}`, and calls `term.Restore`.
7. On `SIGWINCH`, read size with `term.GetSize(int(os.Stdout.Fd()))` and send resize event to daemon which calls `hub.Resize(cols, rows)`.

The existing `relay.Hub` / `relay.Subscriber` fan-out infrastructure handles daemon-side attach with no changes — IPC socket attach is another subscriber type alongside WebSocket subscribers.

### Pattern 5: Service Manager Integration (kardianos/service)

```go
svcConfig := &service.Config{
    Name:        "agenthub-daemon",
    DisplayName: "AgentHub Daemon",
    Description: "Manages AgentHub terminal sessions in the background.",
    // macOS: installs as LaunchAgent (user-level, runs on login)
    // Linux: installs as systemd user unit
    // Windows: installs as Windows Service
    Option: service.KeyValue{
        "KeepAlive":  true,   // launchd: restart on crash
        "RunAtLoad":  true,   // launchd: start immediately on install
        "UserService": true,  // systemd: user-level unit
    },
}

prg := &daemonProgram{...}  // implements service.Interface
svc, err := service.New(prg, svcConfig)
```

`kardianos/service` also provides `service.Interactive()` — returns `true` when running from a terminal vs. from a service manager. Use this to decide whether to write logs to stderr (interactive) or the system logger (service manager).

---

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| `spf13/cobra v1.9.1` | `urfave/cli v2` | urfave/cli is fine but cobra has wider ecosystem adoption, superior persistent-flag inheritance for nested command groups (`web start/stop/status`), and better shell completion generation. The Kubernetes/Docker ecosystem tooling patterns align with cobra. |
| `spf13/cobra v1.9.1` | `urfave/cli v3` | v3 is still gaining adoption (API stabilized late 2024). Cobra is the safe, proven choice. |
| `kardianos/service` | Manual plist/systemd/SCM authoring | Correct plist and systemd unit files require platform-specific knowledge and maintenance across 3 platforms. `kardianos/service` generates them from a single `service.Config{}` struct with well-tested templates. |
| stdlib `net.Listen("unix",...)` + existing `tailscale/go-winio` | `james-barrow/golang-ipc` | `golang-ipc` abstracts what the project already has. Adding it introduces 3+ new transitive deps and version skew risk for something fully covered by stdlib + an existing dep. |
| stdlib `encoding/json` over socket | gRPC + protobuf | Massive overkill for ~15 local command types. Adds code generation, `google.golang.org/grpc` (large dep), proto files to maintain. No performance benefit for a local IPC use case. |
| `golang.org/x/term` | `github.com/pkg/term` | `x/term` is the official Go extended library, already in the binary transitively, actively maintained (v0.41.0 released March 2026). `pkg/term` is unmaintained. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `github.com/james-barrow/golang-ipc` | Wraps what the project already has (go-winio for Windows, stdlib for Unix). Adds transitive deps with no benefit. | stdlib `net` + existing `tailscale/go-winio` |
| `github.com/creack/pty` direct use | The project already uses `aymanbagabas/go-pty` which wraps `creack/pty` on Unix and adds ConPTY on Windows. Mixing both packages causes confusion in the PTY lifecycle. | Continue using `aymanbagabas/go-pty` exclusively |
| gRPC / protobuf for daemon IPC | Overkill for a local socket with ~15 command types. Adds code generation, large deps, build complexity. | stdlib `encoding/json` over Unix socket |
| `tmux` as session backend | Out of scope per PROJECT.md ("Configurable session backend deferred to future milestone"). Adds an external process dependency. | Existing Go-native PTY in `internal/pty` |
| Wails v3 alpha | In alpha as of March 2026; breaking API changes likely before stable release. Project is on Wails v2.10.2 which is stable and battle-tested. | Stay on Wails v2.10.2 |
| Separate daemon binary | Contradicts the "single binary, two modes" requirement in PROJECT.md. Complicates distribution, signing, and PATH management. | Single binary with cobra dispatch in `main()` |

---

## Stack Patterns by Scenario

**If running as GUI (no CLI subcommand in os.Args):**
- `main()` proceeds to `wails.Run()` — unchanged from v1.2.
- Daemon subsystems (registry, relay, webserver) start inside the Wails process as goroutines.
- IPC socket opened so CLI clients can communicate with the running GUI process.

**If running as CLI subcommand (`agenthub list`, `agenthub attach <id>`, etc.):**
- cobra `Execute()` runs; `wails.Run()` is never called.
- CLI dials daemon IPC socket; daemon process must already be running (or CLI auto-starts it).
- No Wails/WebKit overhead; fast startup, stdout/stderr output.

**If running as daemon (`agenthub daemon` or via service manager):**
- `kardianos/service` detects service manager invocation vs. interactive terminal.
- `wails.Run()` never called; no window.
- Sessions managed via existing `SessionRegistry`; web server started via existing `webserver` package.
- IPC socket opened on a well-known path.

**If Windows named pipe needed for IPC:**
- Import `github.com/tailscale/go-winio` directly (already in dep graph — just needs a direct import).
- Use `winio.ListenPipe` / `winio.DialPipe` behind a `//go:build windows` file.
- The rest of the codebase uses `net.Listener` / `net.Conn` — the abstraction boundary is at socket creation only.

---

## Version Compatibility

| Package | Version | Compatible With | Notes |
|---------|---------|-----------------|-------|
| `github.com/spf13/cobra` | v1.9.1 | Go 1.22+ | v1.9.0 released Feb 2025, v1.9.1 bugfix shortly after; both stable |
| `github.com/kardianos/service` | v1.2.4 | Go 1.17+, macOS 12+, Linux systemd v221+, Windows 10+ | July 2025 release. Launchd UserAgent mode supported. |
| `golang.org/x/term` | v0.30.0+ | `golang.org/x/sys v0.40.0` | Must match same golang.org/x/* release family; no conflict expected |
| `github.com/tailscale/go-winio` | existing indirect | Windows 10+ | Named pipe API stable; do not upgrade independently of tailscale dep |

---

## Sources

- [pkg.go.dev/github.com/spf13/cobra](https://pkg.go.dev/github.com/spf13/cobra) — latest version v1.9.1 (Dec 3, 2025), feature set verified. HIGH confidence.
- [github.com/spf13/cobra/releases/tag/v1.9.0](https://github.com/spf13/cobra/releases/tag/v1.9.0) — v1.9.0 release date confirmed Feb 2025. HIGH confidence.
- [pkg.go.dev/github.com/kardianos/service](https://pkg.go.dev/github.com/kardianos/service) — v1.2.4 (July 14, 2025), platform support and API verified. HIGH confidence.
- [pkg.go.dev/golang.org/x/term](https://pkg.go.dev/golang.org/x/term) — v0.41.0 (Mar 10, 2026), MakeRaw/Restore/GetSize API confirmed. HIGH confidence.
- [pkg.go.dev/github.com/aymanbagabas/go-pty](https://pkg.go.dev/github.com/aymanbagabas/go-pty) — v0.2.2, ReadWriteCloser + Resize API verified. HIGH confidence.
- [pkg.go.dev/github.com/Microsoft/go-winio](https://pkg.go.dev/github.com/Microsoft/go-winio) — v0.6.2+, net.Listener-compatible named pipe API confirmed. HIGH confidence.
- [github.com/wailsapp/wails/discussions/4175](https://github.com/wailsapp/wails/discussions/4175) — os.Args pattern in Wails v2 and production build tag approach. MEDIUM confidence (community discussion, no official doc).
- [iximiuz.com — Linux PTY attach/detach internals](https://iximiuz.com/en/posts/linux-pty-what-powers-docker-attach-functionality/) — raw mode + PTY proxy pattern (same pattern Docker uses). MEDIUM confidence (authoritative blog, verified against stdlib docs).
- `/Users/ken/dev/agenthub/go.mod` — existing dependencies verified directly. HIGH confidence.
- `/Users/ken/dev/agenthub/internal/relay/hub.go` — existing Hub/Subscriber fan-out infrastructure verified compatible with attach use case. HIGH confidence.

---
*Stack research for: AgentHub v1.3 CLI + Daemon milestone*
*Researched: 2026-03-23*
