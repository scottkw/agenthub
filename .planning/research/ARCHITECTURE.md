# Architecture Research

**Domain:** CLI + Daemon extraction from Go/Wails desktop app
**Researched:** 2026-03-23
**Confidence:** HIGH (direct codebase inspection + official Go/kardianos/service docs)

---

## Context: Current State (v1.2)

Everything lives in a single Wails process. The `App` struct owns all state and is the only client of the session engine:

```
┌──────────────────────────────────────────────────────────────┐
│                   agenthub binary (Wails)                     │
│                                                               │
│  App struct (app.go)                                          │
│  ├── SessionRegistry   (pty/registry.go)                      │
│  ├── NativePTYBackend  (pty/native.go)                        │
│  ├── HubManager        (relay/manager.go)                     │
│  ├── relay.Server      (relay/server.go)  127.0.0.1:random    │
│  ├── WebServer         (webserver/server.go)  Tailscale IP    │
│  └── Wails runtime → React frontend (embedded assets)        │
└──────────────────────────────────────────────────────────────┘
```

Session I/O path today:
```
React (xterm.js) → WebSocket → relay.Server (127.0.0.1) → Hub → PTY
```

**Key constraint:** The relay.Server and WebServer both live inside the Wails process. The Wails process dies when the window is closed (even with hide-on-close, it still exits on Quit). Sessions cannot outlive the GUI.

---

## Target Architecture (v1.3)

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  agenthub-daemon  (background process, survives GUI close)       │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  SessionEngine                                             │  │
│  │  ├── SessionRegistry   (existing, unchanged)              │  │
│  │  ├── NativePTYBackend  (existing, unchanged)              │  │
│  │  ├── HubManager        (existing, unchanged)              │  │
│  │  └── StatusRegistry    (new: maps sessionID→SessionStatus)│  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌────────────────────────┐  ┌───────────────────────────────┐   │
│  │  DaemonAPI             │  │  WebServer (Tailscale TLS)    │   │
│  │  HTTP/JSON on          │  │  (existing, moved into daemon)│   │
│  │  Unix socket           │  │                               │   │
│  │  (macOS/Linux)         │  │  GET /dashboard               │   │
│  │  Named pipe            │  │  GET /sessions/{id}/ws        │   │
│  │  (Windows)             │  │  GET /api/sessions            │   │
│  └──────────┬─────────────┘  └───────────────────────────────┘   │
└─────────────┼───────────────────────────────────────────────────┘
              │ IPC (HTTP/JSON)
    ┌─────────┴──────────┐
    │                    │
    ▼                    ▼
┌──────────┐      ┌─────────────────────────────────────────────┐
│  CLI     │      │  Wails GUI (agenthub, no --cli flag)        │
│  client  │      │                                             │
│          │      │  App struct (app.go) — becomes thin client  │
│  new,    │      │  ├── DaemonClient  (new: IPC call wrapper)  │
│  list,   │      │  ├── Wails runtime → React frontend         │
│  attach, │      │  └── relay.Server 127.0.0.1 (REMOVED)      │
│  kill,   │      │                                             │
│  rename, │      │  React still uses its own relay port BUT    │
│  web,    │      │  relay.Server moves into the daemon         │
│  health, │      │  GUI connects via DaemonClient, not direct  │
│  qr      │      └─────────────────────────────────────────────┘
└──────────┘
```

### Single Binary, Two Modes

```
agenthub                    → launches Wails GUI (existing behavior)
agenthub daemon             → starts background daemon (new)
agenthub daemon install     → installs as launchd/systemd/Windows service
agenthub daemon uninstall   → removes service registration
agenthub new <cli> [dir]    → CLI: create session
agenthub list               → CLI: list sessions
agenthub attach <id>        → CLI: raw PTY attach
agenthub kill <id>          → CLI: kill session
agenthub rename <id> <name> → CLI: rename session
agenthub web start [port]   → CLI: start web server
agenthub web stop           → CLI: stop web server
agenthub web status         → CLI: web server status
agenthub health             → CLI: Tailscale health
agenthub qr <id>            → CLI: print QR for session URL
agenthub settings [key val] → CLI: read/write settings
```

`main()` dispatch:
```go
func main() {
    if len(os.Args) > 1 {
        // CLI/daemon mode: never starts Wails
        cli.Run(os.Args[1:])
        return
    }
    // GUI mode: existing Wails startup
    runGUI()
}
```

---

## Component Map: New vs Modified vs Unchanged

### NEW: `internal/daemon/` package

The session engine extracted from `App` into a self-contained struct:

```
internal/daemon/
├── engine.go        — SessionEngine: owns registry, backend, hub manager, status map
├── api.go           — DaemonAPI: HTTP handler serving JSON on Unix socket
├── server.go        — Daemon: top-level struct, starts engine + API + WebServer
└── client.go        — DaemonClient: HTTP client for Unix socket (used by GUI and CLI)
```

**SessionEngine** owns everything the current `App` struct owns except Wails bindings:
- `SessionRegistry`
- `NativePTYBackend`
- `HubManager`
- `sessionStatuses` map
- `tabNames` map (session display names)
- `webServer *webserver.WebServer`
- Config persistence (cliPaths, CT disclosure sentinel)

**DaemonAPI** is an `http.Handler` that multiplexes all daemon operations over the IPC socket:

| Method | Path | Operation |
|--------|------|-----------|
| POST | `/sessions` | Create session |
| GET | `/sessions` | List sessions |
| DELETE | `/sessions/{id}` | Kill session |
| PATCH | `/sessions/{id}` | Rename session |
| GET | `/sessions/{id}/status` | Get session status |
| GET | `/sessions/{id}/ws` | Attach WebSocket (raw PTY I/O) |
| POST | `/web/start` | Start web server |
| POST | `/web/stop` | Stop web server |
| GET | `/web/status` | Web server status |
| POST | `/web/sessions/{id}/enable` | Enable web serving for session |
| POST | `/web/sessions/{id}/disable` | Disable web serving for session |
| GET | `/health` | Tailscale health |
| GET | `/sessions/{id}/qr` | QR code base64 |
| GET | `/settings` | Read settings |
| PUT | `/settings` | Write settings |

**DaemonClient** wraps this API with typed Go methods. Both the GUI `App` struct and the CLI commands call `DaemonClient`. This is the only new dependency the GUI introduces.

### NEW: `cmd/` directory

```
cmd/
├── cli/
│   ├── main.go      — entry point for CLI dispatch (called from root main.go)
│   ├── new.go       — `agenthub new` command
│   ├── list.go      — `agenthub list` command
│   ├── attach.go    — `agenthub attach` command (PTY proxy)
│   ├── kill.go      — `agenthub kill` command
│   ├── rename.go    — `agenthub rename` command
│   ├── web.go       — `agenthub web start/stop/status` commands
│   ├── health.go    — `agenthub health` command
│   ├── qr.go        — `agenthub qr` command
│   └── settings.go  — `agenthub settings` command
└── daemon/
    └── main.go      — `agenthub daemon [install|uninstall|start|stop]`
```

### MODIFIED: `app.go`

The `App` struct becomes a thin GUI client. Current session management methods (`CreateSession`, `ListSessions`, `KillSession`, etc.) are rewritten to call `DaemonClient` instead of owning the session engine directly.

**Before (owns engine):**
```go
type App struct {
    ctx      context.Context
    registry *pty.SessionRegistry      // REMOVE
    backend  pty.SessionBackend        // REMOVE
    manager  *relay.HubManager         // REMOVE
    server   *relay.Server             // REMOVE
    listener net.Listener              // REMOVE
    tabNames  map[string]string        // REMOVE (moves to daemon)
    cliPaths  map[string]string        // REMOVE (moves to daemon)
    webServer *webserver.WebServer     // REMOVE (moves to daemon)
    sessionStatuses map[string]status.SessionStatus  // REMOVE (moves to daemon)
}
```

**After (client only):**
```go
type App struct {
    ctx    context.Context
    daemon *daemon.DaemonClient   // NEW: IPC calls
    trayInit bool
}
```

All current Wails-bound methods on `App` remain (same names, same JS binding surface), but their implementations become one-line calls to `daemon.*`. No frontend changes needed.

**Health poller** moves into the daemon. The GUI receives health events via Server-Sent Events or a polling endpoint on the daemon API.

### MODIFIED: `main.go`

Add CLI dispatch before the Wails startup block. The Wails block is unchanged — `NewApp()` now creates a thin client App rather than the full engine.

### MODIFIED: `internal/webserver/server.go`

No structural changes. The `WebServer` is instantiated inside `daemon.SessionEngine` instead of inside `app.go`. The `SetSessionResolver` callback still works the same way.

### UNCHANGED: `internal/relay/`

`Hub`, `HubManager`, `Server`, `Scrollback`, protocol — all unchanged. They move from being constructed in `App` to being constructed in `daemon.SessionEngine`.

### UNCHANGED: `internal/pty/`

`SessionRegistry`, `NativePTYBackend`, `Session`, `SessionBackend` interface — all unchanged.

### UNCHANGED: `internal/status/`

`Detector`, `Watch`, `SessionStatus` — all unchanged.

### UNCHANGED: `internal/webserver/tailscale.go`

`CheckHealth`, `TailscaleHealth` — unchanged.

### UNCHANGED: `tray.go`

The tray icon and callbacks are GUI-only. They remain in `tray.go`, calling `daemon.DaemonClient.Quit()` instead of `runtime.Quit()` directly. The daemon process does not have a tray.

---

## IPC Protocol: HTTP/JSON over Unix Socket

### Why HTTP/JSON over Unix socket (not gRPC, not raw TCP)

- **Go stdlib only.** `net.Listen("unix", socketPath)` + standard `net/http` + `encoding/json`. No new dependencies for the core IPC protocol.
- **WebSocket attach is already HTTP.** The existing relay protocol uses WebSocket over HTTP. Reusing HTTP for the control plane means the daemon runs one server with two endpoint classes: JSON control + WebSocket terminal.
- **Debuggable.** Any HTTP client can call the daemon API during development (`curl --unix-socket /tmp/agenthub.sock http://localhost/sessions`).
- **Precedent.** Docker daemon, tailscaled, and gopls all use this pattern (HTTP or JSON-RPC over Unix socket).

### Socket location

| Platform | Path |
|----------|------|
| macOS/Linux | `~/.config/agenthub/daemon.sock` |
| Windows | Named pipe `\\.\pipe\agenthub-daemon` |

The socket path is deterministic — both CLI and GUI discover it the same way: `filepath.Join(configDir(), "daemon.sock")`. No port negotiation, no PID files.

### Daemon auto-start from CLI

When a CLI command calls `DaemonClient` and the socket does not exist (daemon not running), it auto-starts the daemon:

```go
func (c *DaemonClient) ensureRunning() error {
    if c.isReachable() {
        return nil
    }
    // Exec self with "daemon" subcommand, detached from terminal
    return startDaemonDetached()
}
```

`startDaemonDetached()` runs `os.Executable()` with `daemon` argument, `Stdout/Stderr` redirected to a log file, and `SysProcAttr` set to detach from the controlling terminal (POSIX: `Setsid: true`; Windows: `CREATE_NEW_PROCESS_GROUP`). The CLI then polls the socket path until reachable (max 3 seconds, 100ms intervals).

### Terminal attach protocol

`agenthub attach <id>` is the interactive case. The CLI:

1. Connects to `/sessions/{id}/ws` on the daemon via WebSocket (same binary relay protocol already used by the browser client and the Wails relay server).
2. Puts the terminal in raw mode (`golang.org/x/term`).
3. Runs two goroutines: stdin → `MsgInput` frames; `MsgOutput` frames → stdout.
4. Installs a `SIGWINCH` handler to send `MsgResize2` frames on terminal resize.
5. Listens for a detach key sequence (default: `Ctrl-D Ctrl-D`, configurable). On detach, restores terminal mode and exits without killing the session.

The daemon's DaemonAPI WebSocket handler for attach is identical in structure to the existing `relay/server.go` `handleSession` function — subscribe, replay scrollback, pump frames. No new protocol needed; the relay protocol already handles all three frame types needed for interactive use.

### Event streaming (GUI health updates)

The GUI replaces its Wails `EventsEmit` health poller with a Server-Sent Events endpoint on the daemon:

```
GET /events   (daemon API, text/event-stream)
```

Events:
- `session:status` — `{"sessionId":"...","status":"running"}`
- `tailscale:health` — `TailscaleHealth` struct
- `session:created` — session metadata
- `session:killed` — sessionId

The Wails frontend subscribes to these via a `useEffect` that calls a Wails-bound `SubscribeDaemonEvents()` method, which in turn opens an HTTP SSE connection to the daemon and forwards events via `runtime.EventsEmit`. This keeps the React event model unchanged while moving the source of truth to the daemon.

---

## Data Flow Changes

### Session creation (after migration)

```
React frontend
    ↓ Wails call: app.CreateSession(cli, name, workDir)
App.CreateSession (thin wrapper)
    ↓ HTTP POST /sessions  (Unix socket)
daemon.SessionEngine.CreateSession
    ├── backend.Create(ctx, req) → *pty.Session
    ├── registry.Add(sess)
    ├── manager.Create(id, sess, sess, resizeFn) → *relay.Hub
    ├── go status.Watch(hub, id, cli, onTransit)
    └── returns sessionID
    ↓ HTTP 201 JSON: {"id":"..."}
app.go returns sessionID to Wails JS binding
React frontend uses sessionID to connect to relay WebSocket
```

### React terminal connection (after migration)

React currently calls `GetRelayPort()` to find the Wails-embedded relay server port and connects to `ws://127.0.0.1:{port}/sessions/{id}/ws`. After migration:

- The relay.Server moves from `App.startup` into the daemon.
- The daemon listens on a stable TCP port (or a second Unix socket) for relay WebSocket connections from the GUI.
- `GetRelayPort()` becomes a call to the daemon that returns the daemon's relay port.

**Simplest option:** Daemon binds relay.Server on `127.0.0.1:0` (random port, same as today), daemon API exposes `GET /relay-port`. The GUI calls this once at startup. React behavior is unchanged. This is a one-line change in the App struct.

### Terminal attach (CLI, new flow)

```
agenthub attach <id>
    ↓ DaemonClient.Attach(id)
    ↓ WebSocket connect to daemon API /sessions/{id}/ws
    ↓ Terminal raw mode on
    ↓ stdin → MsgInput frames → daemon → PTY
    ↓ PTY output → MsgOutput frames → daemon → stdout
    ↓ SIGWINCH → MsgResize2 frames
    ↓ Detach key sequence → restore terminal → exit (session stays alive)
```

---

## Migration Path: Wails-Owns-Everything → Daemon-Centric

### Phase 1: Extract SessionEngine into daemon package (no behavior change)

Create `internal/daemon/engine.go` with `SessionEngine` that wraps `SessionRegistry`, `NativePTYBackend`, `HubManager`, and the status/tabNames maps. Copy-paste `App`'s session methods into `SessionEngine` with identical logic.

In `App`, wire `NewSessionEngine()` and delegate all session methods to it. `App` still owns `SessionEngine` directly — no IPC yet. All existing tests pass unchanged.

**Why first:** Establishes the module boundary. Daemon code is now testable in isolation before any process separation happens. Compiler enforces the interface.

### Phase 2: DaemonAPI HTTP server on Unix socket

Add `internal/daemon/api.go` implementing the HTTP handler for the daemon API routes. Add `internal/daemon/client.go` implementing `DaemonClient`.

Wire `App` to use `DaemonClient` instead of holding `SessionEngine` directly. The daemon is still in-process (same binary, no fork), but all calls go through the HTTP layer over a Unix socket. This validates the protocol without multi-process complexity.

**Confidence check:** Run all existing tests with a local in-process daemon. If anything breaks here, it breaks trivially (socket path, JSON serialization) not architecturally.

### Phase 3: Fork the daemon process

Add `cmd/daemon/main.go`. The daemon starts as a separate process. `DaemonClient.ensureRunning()` forks if the socket is not present.

`main.go` gets the CLI dispatch block. Running `agenthub` with no args starts Wails as before; running with `daemon` or a session command takes the CLI path.

The relay.Server moves from `App.startup` into the daemon. `GetRelayPort()` is wired to the daemon. The GUI no longer owns a relay listener.

**This is the first phase where sessions outlive the GUI window.** Verify: close the GUI, reopen it, and confirm sessions are still listed and attachable.

### Phase 4: CLI commands

Implement CLI commands in `cmd/cli/`. Each command is a thin wrapper:
- Parse args
- Call `DaemonClient.EnsureRunning()`
- Call the relevant `DaemonClient` method
- Print results

`attach` is the only complex command (raw PTY proxy, `golang.org/x/term`, SIGWINCH handler, detach key).

### Phase 5: Service manager integration

Add `agenthub daemon install / uninstall` using `kardianos/service`. The service just runs `agenthub daemon` in the foreground — the service manager handles restart-on-failure.

Service definition is generated at install time and written to the platform's config directory:
- macOS: `~/Library/LaunchAgents/com.agenthub.daemon.plist`
- Linux: `~/.config/systemd/user/agenthub-daemon.service`
- Windows: Windows service registry via `kardianos/service`

---

## Structural Changes to the Codebase

### Recommended final file tree delta

```
agenthub/
├── main.go                    MODIFIED — add CLI dispatch
├── app.go                     MODIFIED — thin client, delegates to DaemonClient
├── tray.go                    UNCHANGED
├── cmd/
│   ├── cli/
│   │   ├── root.go            NEW — cobra/flag CLI entry point
│   │   ├── new.go             NEW
│   │   ├── list.go            NEW
│   │   ├── attach.go          NEW — PTY proxy, raw mode
│   │   ├── kill.go            NEW
│   │   ├── rename.go          NEW
│   │   ├── web.go             NEW
│   │   ├── health.go          NEW
│   │   ├── qr.go              NEW
│   │   └── settings.go        NEW
│   └── daemon/
│       └── main.go            NEW — daemon subcommand + service install/uninstall
├── internal/
│   ├── daemon/
│   │   ├── engine.go          NEW — SessionEngine (session logic extracted from App)
│   │   ├── api.go             NEW — HTTP handler on Unix socket
│   │   ├── server.go          NEW — top-level Daemon struct
│   │   └── client.go          NEW — DaemonClient (used by GUI and CLI)
│   ├── pty/                   UNCHANGED
│   ├── relay/                 UNCHANGED
│   ├── status/                UNCHANGED
│   └── webserver/             UNCHANGED
```

No existing internal packages are modified. The extraction is additive until Phase 3, when `app.go` delegates to `DaemonClient`.

---

## Integration Points

### Wails GUI ↔ Daemon

| Point | Current | After Migration |
|-------|---------|-----------------|
| Session CRUD | Direct method calls on `App` fields | `DaemonClient` HTTP calls to Unix socket |
| Relay WebSocket port | `app.listener` (in-process) | `DaemonClient.GetRelayPort()` → daemon API |
| Health events | `app.startHealthPoller` → `runtime.EventsEmit` | Daemon SSE stream → GUI proxies via `runtime.EventsEmit` |
| Web server control | `app.StartWebServer` / `StopWebServer` | `DaemonClient.WebStart/Stop` |
| Session status | `app.sessionStatuses` map | `DaemonClient.GetSessionStatus` |
| Tab names | `app.tabNames` map | `DaemonClient.RenameSession` + `ListSessions` includes name |
| Wails JS bindings | Same method names on `App` | Same method names, different implementation bodies |

**Critical invariant:** All Wails-bound method signatures on `App` remain identical. Zero changes to the React frontend are required in Phase 2–3.

### CLI ↔ Daemon

All CLI commands go through `DaemonClient`. The CLI has no direct knowledge of PTYs, hubs, or the relay protocol except in `attach.go`, which uses the existing binary relay framing protocol over WebSocket.

### Service Manager ↔ Daemon

The service manager invokes `agenthub daemon` and manages the process lifecycle (restart-on-crash, start-on-login). The daemon must not daemonize itself (no double-fork) — service managers expect the process to run in the foreground.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Shared In-Process State After Phase 2

**What happens:** Keeping `SessionEngine` as a direct field of `App` while also exposing it via `DaemonClient` creates two code paths. Bugs will appear in one but not the other.

**Do this instead:** After Phase 2 validates the DaemonClient interface, remove the direct field immediately. One code path only.

### Anti-Pattern 2: TCP Port for the Daemon API

**What people do:** Bind the daemon API on a localhost TCP port (e.g., 7444) and store the port number in a config file or PID file.

**Why it's wrong:** Port conflicts across users or multiple test instances. Port scanning by other processes. PID files are notorious for stale-state bugs. Unix sockets are path-addressed — no ports, no discovery needed.

**Exception:** Windows requires named pipes. `kardianos` and the Go standard library handle this transparently via `net.Listen("unix", path)` on POSIX and named pipe emulation on Windows.

### Anti-Pattern 3: Duplicating the Relay Protocol for CLI Attach

**What people do:** Create a separate "CLI attach" protocol that copies PTY bytes over stdin/stdout of the daemon process directly.

**Why it's wrong:** Redundant with the existing WebSocket relay protocol. The relay protocol already handles all three interactive frame types (MsgOutput, MsgInput, MsgResize2). The `relay/server.go` `handleSession` function is 50 lines and already tested.

**Do this instead:** The `agenthub attach` CLI command connects to `/sessions/{id}/ws` on the daemon API using the existing binary frame protocol. The daemon API WebSocket handler is identical to `relay/server.go`.

### Anti-Pattern 4: Rebuilding the Wails Frontend for CLI Output

**What people do:** Add a "headless" Wails mode or reuse React components for CLI output formatting.

**Why it's wrong:** The CLI has no WebView. React and xterm.js have no role in a terminal attach.

**Do this instead:** CLI output is plain text to stdout. `attach` connects directly to the daemon WebSocket and pipes bytes. `list` formats a table with `text/tabwriter`. No shared UI code.

### Anti-Pattern 5: Embedding the Daemon in the Wails App Process

**What people do:** Run the daemon logic in a background goroutine inside the Wails process, protected by a mutex, with the "daemon" being just a goroutine pool.

**Why it's wrong:** Sessions still die when the Wails window closes / process exits. The whole point of the daemon is process-level persistence. "Daemon as goroutine" solves nothing.

**Do this instead:** The daemon is always a separate OS process. The GUI connects to it like any other client.

---

## Scaling Considerations

This is a local desktop app. Scaling means "many sessions per user," not "many users."

| Concern | Current | With Daemon |
|---------|---------|-------------|
| Sessions outliving GUI | Not supported | Supported — daemon owns sessions |
| Multiple GUI windows | Not supported (single Wails window) | Possible future extension |
| CLI + GUI simultaneously | Not supported | Supported — both are daemon clients |
| Session count limit | No hard limit; PTY is the bottleneck | Same; each session is ~1 goroutine + PTY fd |
| IPC throughput | N/A (in-process) | Unix socket saturates at ~1 GB/s; terminal I/O is KB/s |

No scaling concerns at the local desktop level. The daemon architecture is also the right foundation for a future remote-access mode where the daemon runs headless on a server.

---

## Sources

- Direct codebase inspection: `app.go`, `main.go`, `internal/pty/`, `internal/relay/`, `internal/webserver/`, `internal/status/`
- [kardianos/service](https://pkg.go.dev/github.com/kardianos/service) — cross-platform service manager (macOS launchd, Linux systemd, Windows service) — MEDIUM confidence (well-maintained library, 4.3k stars, stable API)
- [gopls daemon architecture](https://go.dev/gopls/daemon) — Unix socket + HTTP/JSON-RPC pattern for Go daemon/client split — HIGH confidence (official Go toolchain)
- [Unix domain sockets in Go (Eli Bendersky)](https://eli.thegreenplace.net/2019/unix-domain-sockets-in-go/) — net.Listen("unix") pattern — HIGH confidence
- Docker daemon / tailscaled — precedent for HTTP over Unix socket as IPC — HIGH confidence

---

*Architecture research for: AgentHub v1.3 CLI + Daemon*
*Researched: 2026-03-23*
