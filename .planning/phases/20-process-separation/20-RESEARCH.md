# Phase 20: Process Separation - Research

**Researched:** 2026-03-23
**Domain:** Go daemon process spawning, os/exec, Wails lifecycle, Unix socket IPC, process supervision
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DAEMON-01 | Session management runs in a standalone daemon process separate from the GUI | Implemented via `cmd/daemon` sub-process spawned from `main.go`; daemon owns SessionEngine; GUI is a DaemonClient-only consumer |
| DAEMON-03 | Sessions persist when all clients (GUI and CLI) disconnect | Daemon process keeps running after GUI window closes (beforeClose hides, shutdown does NOT kill daemon); daemon process lifecycle is independent of App struct |
| DAEMON-04 | GUI app connects to the daemon as a client; GUI and CLI see the same session pool | App struct is already a thin DaemonClient shell (Phase 19); Phase 20 removes the in-process engine/api from App, connects to the out-of-process daemon on the same DefaultSocketPath |
| DAEMON-05 | Daemon auto-starts when any CLI command is run and no daemon is running | `EnsureDaemon()` helper: probe socket with DaemonClient.Health(); if unreachable, spawn `agenthub daemon` via os/exec with Cmd.SysProcAttr for detach; poll until ready |
</phase_requirements>

---

## Summary

Phase 20 is the process separation step. Phase 19 validated the full API contract between App and SessionEngine in-process. Phase 20 moves the SessionEngine+API to a separate `agenthub daemon` sub-process. The change surface in `app.go` is tiny: remove `engine *daemon.SessionEngine` and `api *daemon.API` from the App struct, remove their construction from `NewApp()`, and remove `api.Start()` from `startup()`. Instead, `startup()` calls `EnsureDaemon()` to guarantee the daemon is running, then wires `DaemonClient` to the same `DefaultSocketPath()` it already uses. The relay server (`relay.Server`) currently takes `engine.Manager()` and `engine.Backend()` — this coupling must be resolved; these will move into the daemon process.

The trickiest design problem is the relay WebSocket server. Currently the GUI process owns the relay TCP listener and passes `engine.Manager()` and `engine.Backend()` into `relay.NewServer()`. After process separation, those objects live in the daemon process. The relay server must move into the daemon as well (served over a second TCP port that the daemon allocates and exposes via a `/relay-port` API endpoint). The GUI then fetches that port and propagates it to the frontend (replacing the current `GetRelayPort()` which reads from `a.listener`). The WebSocket connections from the browser continue to connect to the relay; the relay endpoint just moves to the daemon's port.

The web server (`webserver.WebServer`) also calls `engine.Manager()` for hub lookups. After process separation, the web server must also move into the daemon. The GUI's `StartWebServer`/`StopWebServer` Wails bindings become API calls to the daemon via new routes (`POST /webserver/start`, `POST /webserver/stop`, `GET /webserver/status`). These routes are already defined in the Phase 19 RESEARCH.md "Pattern 5" for completeness but not yet implemented.

**Primary recommendation:** Implement process separation in exactly two tasks: (1) move relay server + web server into the daemon process with new API routes, and (2) strip App of in-process engine/api, add `EnsureDaemon()`, and connect App as a pure client. Keep the daemon's socket path identical — `DefaultSocketPath()` — so DaemonClient needs zero changes.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` (stdlib) | go1.26.1 | Spawn daemon subprocess | Zero deps, exact binary path via `os.Executable()`, cross-platform |
| `syscall.SysProcAttr` (stdlib) | go1.26.1 | Detach daemon from GUI process group (Unix) | Without detach, daemon dies when GUI closes on macOS/Linux |
| `net/http` (stdlib) | go1.26.1 | New daemon HTTP routes for relay port, webserver | Already the IPC layer from Phase 19 |
| `context` (stdlib) | go1.26.1 | Shutdown coordination in daemon main loop | Already the codebase standard |

### Supporting (already in go.mod — no new deps)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `time` (stdlib) | — | Polling loop in `EnsureDaemon()` to wait for daemon ready | Max 3s wait with 50ms intervals |
| `sync` (stdlib) | — | Daemon process mutex (already in SessionEngine) | No new usages needed |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `os/exec` + `SysProcAttr` detach | `kardianos/service` | Service install is Phase 23; Phase 20 only needs on-demand auto-start, not service install |
| Move relay into daemon | Keep relay in GUI, proxy through daemon | Proxy adds latency and complexity; moving relay is the clean solution |
| New `cmd/daemon/main.go` entry point | Build tag on `main.go` | Separate `main.go` with `agenthub daemon` subcommand is cleaner; avoids build tag confusion with Wails |

**Installation:** No new dependencies. All stdlib.

---

## Architecture Patterns

### Recommended Project Structure After Phase 20
```
main.go                        # Wails entry: checks for "daemon" arg, else GUI
internal/daemon/
├── engine.go                  # SessionEngine (unchanged from Phase 19)
├── api.go                     # HTTP API — ADD relay port + webserver routes
├── client.go                  # DaemonClient — ADD relay port + webserver methods
├── socket.go                  # DefaultSocketPath, ValidateSocketPath, CleanupStaleSocket (unchanged)
├── types.go                   # ADD RelayPortResponse, WebServerStatusResponse
├── process.go                 # NEW: EnsureDaemon(), StartDaemonProcess(), daemon main loop
└── *_test.go                  # Existing + new tests for process.go
app.go                         # REMOVE engine/api fields; ADD EnsureDaemon call in startup()
```

### Pattern 1: Single Binary, Two Modes
**What:** `main.go` checks `os.Args` before calling `wails.Run`. If the first argument is `"daemon"`, run the daemon main loop and `os.Exit`. Otherwise launch the GUI.
**When to use:** Phase 20 entry point design.

```go
// Source: standard Go CLI subcommand pattern
func main() {
    if len(os.Args) > 1 && os.Args[1] == "daemon" {
        daemon.RunDaemon()   // blocks until signal
        return
    }
    // Wails GUI launch
    app := NewApp()
    err := wails.Run(...)
    ...
}
```

**Critical:** `daemon.RunDaemon()` must call `os.Exit(0)` or return (not `panic`) so the process exits cleanly on SIGTERM.

### Pattern 2: Daemon Process Spawning with Detach (EnsureDaemon)
**What:** Before wiring DaemonClient, check if the daemon socket is already answering. If not, spawn the daemon as a detached child process.
**When to use:** App `startup()` and (in Phase 21) CLI command preamble.

```go
// Source: standard Go daemon-spawn pattern using os/exec + SysProcAttr
func EnsureDaemon(socketPath string) error {
    client := NewDaemonClient(socketPath)
    if err := client.Health(); err == nil {
        return nil // already running
    }

    exe, err := os.Executable()
    if err != nil {
        return fmt.Errorf("EnsureDaemon: locate binary: %w", err)
    }

    cmd := exec.Command(exe, "daemon")
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setsid: true, // detach from GUI's process group (Unix)
    }
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("EnsureDaemon: start daemon: %w", err)
    }
    cmd.Process.Release() // detach from this process; daemon self-manages

    // Poll until daemon is ready (max 3 seconds).
    deadline := time.Now().Add(3 * time.Second)
    for time.Now().Before(deadline) {
        if err := client.Health(); err == nil {
            return nil
        }
        time.Sleep(50 * time.Millisecond)
    }
    return fmt.Errorf("EnsureDaemon: daemon did not start within 3s")
}
```

**Windows note:** `SysProcAttr.Setsid` does not exist on Windows — use `SysProcAttr.CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP` instead. Gate with `//go:build !windows` and `//go:build windows` files or a runtime `GOOS` check.

### Pattern 3: Relay Port Handoff via API
**What:** Daemon starts the relay TCP server on an OS-assigned port and exposes the port via `GET /relay-port`. GUI fetches this port in `startup()` and propagates it to the frontend exactly as before.
**When to use:** Replaces `a.listener` + `a.GetRelayPort()` after relay moves into daemon.

```go
// New daemon API route (api.go)
a.mux.HandleFunc("GET /relay-port", a.handleRelayPort)

func (a *API) handleRelayPort(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, RelayPortResponse{Port: a.relayPort})
}

// New DaemonClient method (client.go)
func (c *DaemonClient) GetRelayPort() (int, error) {
    var resp RelayPortResponse
    if err := c.doJSON(http.MethodGet, "/relay-port", nil, &resp); err != nil {
        return 0, err
    }
    return resp.Port, nil
}

// Updated App.GetRelayPort (app.go)
func (a *App) GetRelayPort() int {
    port, err := a.client.GetRelayPort()
    if err != nil {
        return 0
    }
    return port
}
```

### Pattern 4: Web Server Delegation to Daemon
**What:** `StartWebServer`, `StopWebServer`, `IsWebServerRunning`, `GetWebServerURL`, `ToggleWebServing` all become API calls to daemon routes. App struct loses `webServer *webserver.WebServer` and the `mu sync.RWMutex` that guarded it.
**When to use:** After relay moves into daemon; web server move follows naturally since it depends on HubManager.

New daemon routes (already listed in Phase 19 RESEARCH.md Pattern 5):
```
POST   /webserver/start         → {"ip":"...","port":N,"fqdn":"..."} → {"url":"..."}
POST   /webserver/stop          → 204
GET    /webserver/status        → {"running":bool,"url":"...","addr":"..."}
POST   /sessions/{id}/web-serve → {"enabled":bool} → 204
```

**Tailscale health check:** The GUI App currently gates `StartWebServer` with `GetTailscaleStatus()`. This check must stay in the GUI (App) layer — the GUI knows whether the user has accepted the CT disclosure and has the Wails context to show UI. The daemon receives a pre-validated call. App calls `GetTailscaleStatus()` locally, then passes `ip`, `port`, `fqdn` to the daemon.

### Pattern 5: App Struct After Process Separation
**What:** App holds exactly one daemon communication field: `*daemon.DaemonClient`. This is the success criterion stated in the requirements.
**When to use:** Final App struct shape.

```go
// App after Phase 20
type App struct {
    ctx      context.Context
    client   *daemon.DaemonClient   // only daemon communication field
    server   *relay.Server          // REMOVED — relay now in daemon
    listener net.Listener           // REMOVED — relay port fetched via client
    trayInit bool
    // webServer, mu REMOVED — web server now in daemon
}
```

Wait — the success criterion says: "GUI App struct holds exactly one field for daemon communication: `*daemon.DaemonClient`". This is not a statement that the ENTIRE struct has exactly one field; it means the daemon communication field is exactly one. `ctx`, `trayInit` (non-daemon fields) can remain. This is confirmed by the requirements text.

### Anti-Patterns to Avoid
- **Keeping relay in the GUI process:** Relay depends on `HubManager` which lives in daemon after separation. Keeping relay in GUI means cross-process hub access — that path leads to multiplexed WebSocket proxying which is harder than moving the relay.
- **Not releasing the child process:** `cmd.Start()` without `cmd.Process.Release()` means the daemon becomes a zombie when the GUI exits. Always call `Release()` after detaching.
- **Hard-coding the binary path:** Always use `os.Executable()` to find the current binary — avoids PATH lookups and works correctly in packaged app bundles.
- **Polling without a deadline:** EnsureDaemon must have a hard timeout; an unreachable daemon that never starts should be an error, not an infinite loop.
- **Running relay in daemon without stopping it on shutdown:** Daemon must close the relay listener (and the web server listener) in its shutdown handler on SIGTERM/SIGINT.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Process daemonization | Custom fork/exec with double-fork | `os/exec` + `SysProcAttr.Setsid` | Double-fork is a Unix-specific pattern; `Setsid` achieves session leadership with one call in Go |
| Socket readiness probe | Custom TCP ping | `DaemonClient.Health()` already exists | The HTTP `/health` endpoint is the correct readiness probe |
| Relay port negotiation | Custom IPC message | New `GET /relay-port` HTTP route | Follows the existing HTTP/JSON pattern; no new protocol |
| Signal handling in daemon | `os/signal` package custom setup | `signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)` | stdlib, clean, already the pattern in Go servers |

---

## Common Pitfalls

### Pitfall 1: Relay Server Still in App After Separation
**What goes wrong:** App keeps its `relay.Server` + `net.Listener` + `GetRelayPort()`. Frontend gets a port to the GUI relay, not the daemon relay. When the GUI is hidden, WebSocket connections to the GUI relay work — but the hub is in the daemon. Two separate hub managers = split state.
**Why it happens:** "Move relay to daemon" is not stated explicitly in the requirements; it is a forced consequence of HubManager moving to daemon.
**How to avoid:** Move relay server construction into `daemon.RunDaemon()`. Remove `a.server`, `a.listener` from App. Wire `GetRelayPort()` through DaemonClient.
**Warning signs:** `go test -race ./...` passes but relay test fails when connecting after GUI restart.

### Pitfall 2: Zombie Child Process
**What goes wrong:** Daemon process becomes a zombie after GUI exits because Go's `os/exec` sets up wait-state tracking.
**Why it happens:** `cmd.Start()` registers the child; if the parent exits without `Wait()`, OS keeps the entry in the process table.
**How to avoid:** Call `cmd.Process.Release()` immediately after `cmd.Start()`. This tells Go to disown the process.
**Warning signs:** `ps aux | grep zombie` shows `<defunct>` agenthub processes.

### Pitfall 3: Windows SysProcAttr Incompatibility
**What goes wrong:** `syscall.SysProcAttr{Setsid: true}` does not compile on Windows — the `Setsid` field does not exist in the Windows `SysProcAttr`.
**Why it happens:** `SysProcAttr` is platform-specific; the Unix and Windows structs have different fields.
**How to avoid:** Create two build-tagged files: `process_unix.go` (Setsid: true) and `process_windows.go` (CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP). Both are in `internal/daemon`.
**Warning signs:** `GOOS=windows go build ./...` fails with "unknown field 'Setsid' in SysProcAttr".

### Pitfall 4: Stale Socket From Previous Daemon Still Alive
**What goes wrong:** `EnsureDaemon` probes the socket, finds it responding (a stale daemon from a previous session that never died), and returns — but the stale daemon has no session state matching what the GUI expects.
**Why it happens:** After process separation, the daemon is truly persistent. If the user force-quits the daemon or it crashes, a stale socket is left. `CleanupStaleSocket` handles the crash case. The "old daemon still alive" case is actually correct behavior — DAEMON-03 says sessions persist, so an old daemon with its sessions is correct.
**How to avoid:** This is actually correct operation. Document it: `EnsureDaemon` is correct to connect to an existing daemon even if it has state from a prior GUI session — that is the point of DAEMON-03.

### Pitfall 5: Wails OnBeforeClose vs Daemon Shutdown
**What goes wrong:** `shutdown()` is called when the user Quits from the tray. If `shutdown()` still calls `a.api.Stop()`, it will try to stop a nil API (since App no longer owns the API in Phase 20) — nil pointer panic.
**Why it happens:** `shutdown()` currently calls `a.api.Stop()`. After Phase 20, `a.api` is removed from App.
**How to avoid:** Remove `a.api.Stop()` from `App.shutdown()`. The daemon is an independent process — the GUI does not stop it. Shutdown only closes `a.listener` (if kept for backward compat, but it will be removed too) and cleans up the tray.

### Pitfall 6: CreateSession Still Calls Engine Directly
**What goes wrong:** After process separation, `a.engine` is gone. `app.go` line 144 `a.engine.CreateSession(...)` becomes a nil pointer dereference.
**Why it happens:** CreateSession was the one intentional exception to the delegation pattern (Phase 19-02 decision) — it calls the engine directly to pass the onStatus callback.
**How to avoid:** After process separation, `CreateSession` must go through the daemon's HTTP API (no callback). The status events are pushed from daemon → GUI via a polling mechanism or SSE. For Phase 20, the simplest solution is: App polls `GetSessionStatus` on a ticker for sessions it just created (or the frontend's existing polling is sufficient). The `onStatus` callback emission of `session:status` Wails events must be re-examined.

**Options for status events after process separation:**
1. **Polling (simplest):** App polls `/sessions/{id}/status` every 2s after CreateSession. Frontend already handles status events; App emits Wails events on change. No new protocol.
2. **SSE endpoint on daemon:** `GET /sessions/events` Server-Sent Events stream. DaemonClient long-polls and emits Wails events. More responsive but adds complexity.
3. **No change in Phase 20:** Keep App's status polling as a Wails-event emitter separate from the daemon's internal Watch goroutine. Daemon's Watch still updates `sessionStatuses` map; GUI polls via `GetSessionStatus`.

**Recommendation:** Option 3 (polling) for Phase 20 — the frontend already uses the polling pattern. Status events are best-effort UX, not correctness-critical. SSE can be Phase 21+ if needed.

---

## Code Examples

### Daemon Entry Point (RunDaemon)
```go
// Source: standard Go signal-based server lifecycle
// internal/daemon/process.go

func RunDaemon() {
    socketPath := DefaultSocketPath()
    if err := CleanupStaleSocket(socketPath); err != nil {
        fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
        os.Exit(1)
    }

    engine := NewSessionEngine()
    api := NewAPI(engine) // starts relay + web server inside

    if err := api.Start(socketPath); err != nil {
        fmt.Fprintf(os.Stderr, "daemon: start api: %v\n", err)
        os.Exit(1)
    }

    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGTERM, syscall.SIGINT)
    defer stop()

    <-ctx.Done()

    _ = api.Stop()
    engine.Manager().Shutdown()
}
```

### main.go Subcommand Dispatch
```go
// Source: standard Go CLI subcommand pattern
func main() {
    if len(os.Args) > 1 && os.Args[1] == "daemon" {
        daemon.RunDaemon()
        return
    }
    // existing wails.Run(...)
}
```

### EnsureDaemon (process_unix.go)
```go
//go:build !windows

package daemon

import (
    "fmt"
    "os"
    "os/exec"
    "syscall"
    "time"
)

func startDetachedDaemon(exe string) error {
    cmd := exec.Command(exe, "daemon")
    cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
    if err := cmd.Start(); err != nil {
        return err
    }
    return cmd.Process.Release()
}
```

### App.startup() After Separation
```go
func (a *App) startup(ctx context.Context) {
    a.ctx = ctx

    // Ensure daemon is running; auto-start if not.
    if err := daemon.EnsureDaemon(a.socketPath); err != nil {
        panic(fmt.Sprintf("agenthub: ensure daemon: %v", err))
    }
    a.client = daemon.NewDaemonClient(a.socketPath)

    a.initTray()
    a.trayInit = true
    a.startHealthPoller(ctx)
}
```

### App.GetRelayPort() After Separation
```go
func (a *App) GetRelayPort() int {
    port, err := a.client.GetRelayPort()
    if err != nil {
        return 0
    }
    return port
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Session state in App struct | SessionEngine in daemon package (in-process) | Phase 19 | Zero Wails imports in engine; protocol validated |
| In-process daemon server | Out-of-process daemon process | Phase 20 | Sessions survive GUI close; GUI is a pure client |
| App owns relay listener | Daemon owns relay listener; GUI fetches port | Phase 20 | RelayPort API call replaces a.listener.Addr() |
| App.StartWebServer calls webserver.NewWebServer | App delegates to daemon `POST /webserver/start` | Phase 20 | Web server lifecycle moves to daemon |

**Deprecated after Phase 20:**
- `App.engine *daemon.SessionEngine`: removed; engine lives in daemon process
- `App.api *daemon.API`: removed; API is a daemon-owned server
- `App.server *relay.Server`: removed; relay moves to daemon
- `App.listener net.Listener`: removed; relay port fetched via DaemonClient
- `App.webServer *webserver.WebServer`: removed; web server moves to daemon
- `App.mu sync.RWMutex`: removed; no more App-owned state it was guarding

---

## Open Questions

1. **Status events after CreateSession callback removal**
   - What we know: App.CreateSession currently calls `engine.CreateSession` directly with an `onStatus` callback. After process separation, `a.engine` is gone.
   - What's unclear: What is the minimum viable replacement for Phase 20? Options: (a) polling via `GetSessionStatus`, (b) App polls on its own ticker, (c) SSE.
   - Recommendation: For Phase 20, App polls `GetSessionStatus` on a per-session ticker (2s interval) for N seconds after CreateSession, emitting Wails `session:status` events on change. This is the minimum viable approach with zero new protocol. The frontend already handles `session:status` events. Stop polling when status reaches `errored` or after 60s.

2. **Relay port handoff timing with Wails startup**
   - What we know: The frontend calls `GetRelayPort()` during its initial `useEffect`. Currently, `a.listener` is allocated synchronously in `startup()` before Wails renders the frontend, so the port is always ready. After Phase 20, the port comes from `DaemonClient.GetRelayPort()` — this is an HTTP call to the daemon's `/relay-port` endpoint.
   - What's unclear: Does the Wails frontend call `GetRelayPort()` before the daemon's relay is fully started?
   - Recommendation: The daemon starts before `startup()` returns (EnsureDaemon polls until `/health` is ready, and relay starts before health is ready). So by the time the frontend calls `GetRelayPort()`, the daemon relay is running. Add a short wait in `EnsureDaemon` after health check to ensure relay is bound. Alternatively, return the relay port in `/health` response so a single probe confirms both.

3. **Daemon persistence on macOS app bundle close**
   - What we know: On macOS, closing all windows of an `.app` bundle by default terminates the process if `applicationShouldTerminateAfterLastWindowClosed` returns YES. Wails handles this via `HideWindowOnClose: true` + `beforeClose`.
   - What's unclear: When the user uses Cmd+Q (not the tray Quit), does Wails still call `shutdown()`? If so, the current `shutdown()` stops the daemon API — but after Phase 20 the daemon is a separate process, so shutdown stops nothing problematic.
   - Recommendation: After removing `a.api.Stop()` from `shutdown()`, the daemon persists across all GUI close events including Cmd+Q. Only the tray "Quit" (which calls `runtime.Quit`) triggers actual process exit. Verify this in the Phase 20 GUI regression test.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (go1.26.1) |
| Config file | None — standard `go test` |
| Quick run command | `go test -race ./internal/daemon/... .` |
| Full suite command | `go test -race ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DAEMON-01 | Daemon process spawned by EnsureDaemon runs as separate process | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemon` | ❌ Wave 0 |
| DAEMON-01 | RunDaemon starts API, exits cleanly on SIGTERM | integration | `go test -race ./internal/daemon/... -run TestRunDaemon` | ❌ Wave 0 |
| DAEMON-03 | Sessions survive after App.shutdown() is called | unit | `go test -race . -run TestShutdownSessionSurvive` | ❌ Wave 0 |
| DAEMON-03 | App reconnects to running daemon on second startup | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemonAlreadyRunning` | ❌ Wave 0 |
| DAEMON-04 | App.ListSessions matches daemon sessions after separation | integration | `go test -race . -run TestListSessions` (existing, already green) | ✅ existing |
| DAEMON-04 | App holds no engine/api/relay fields after migration | unit | `go test -race . -run TestAppStructFields` (reflection-based) | ❌ Wave 0 |
| DAEMON-05 | EnsureDaemon starts daemon when socket absent | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemon_NoSocket` | ❌ Wave 0 |
| DAEMON-05 | EnsureDaemon returns quickly when daemon already running | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemon_AlreadyRunning` | ❌ Wave 0 |
| DAEMON-05 | EnsureDaemon returns error if daemon doesn't start within timeout | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemon_Timeout` | ❌ Wave 0 |

**Note on DAEMON-01 and DAEMON-03 integration tests:** Testing a subprocess spawn in `go test` requires either spawning the real binary (needs a build step) or using `os/exec.Command(os.Args[0], "-test.run=TestDaemonMain")` with the `TestMain` trick. The simpler approach for Phase 20: test `EnsureDaemon` by starting a real in-process daemon API on a test socket (same pattern as Phase 19 testDaemon helper), and test that `App.shutdown()` does NOT stop that daemon. Full subprocess spawning tests can be Phase 21 scope (CLI testing requires subprocess anyway).

### Sampling Rate
- **Per task commit:** `go test -race ./internal/daemon/... .`
- **Per wave merge:** `go test -race ./...`
- **Phase gate:** `go test -race ./...` green + manual GUI regression (sessions survive window close, reopen shows same sessions)

### Wave 0 Gaps
- [ ] `internal/daemon/process.go` — EnsureDaemon, startDetachedDaemon, RunDaemon
- [ ] `internal/daemon/process_unix.go` — SysProcAttr Setsid (build tag `!windows`)
- [ ] `internal/daemon/process_windows.go` — SysProcAttr CREATE_NEW_PROCESS_GROUP (build tag `windows`)
- [ ] `internal/daemon/process_test.go` — TestEnsureDaemon_*, TestRunDaemon
- [ ] `internal/daemon/types.go` additions — RelayPortResponse, WebServerStartRequest, WebServerStatusResponse
- [ ] `internal/daemon/api.go` additions — relay port route, webserver routes
- [ ] `internal/daemon/client.go` additions — GetRelayPort, StartWebServer, StopWebServer, GetWebServerStatus, ToggleWebServing

---

## Sources

### Primary (HIGH confidence)
- `app.go` — authoritative current App struct; all fields to be removed identified by direct inspection
- `internal/daemon/engine.go`, `api.go`, `client.go`, `socket.go` — Phase 19 output, directly read
- `main.go` — current Wails entry point; subcommand dispatch pattern derived from direct inspection
- `frontend/src/App.tsx` — confirms `GetRelayPort()` usage in frontend; relay port handoff timing verified
- `.planning/STATE.md` — `[Phase 20 risk]` and `[Phase 20 research flag]` entries directly inform pitfalls
- `.planning/phases/19-daemon-core-engine-ipc/19-01-SUMMARY.md`, `19-02-SUMMARY.md` — Phase 19 decisions confirmed

### Secondary (MEDIUM confidence)
- `os/exec` + `SysProcAttr.Setsid` for daemon detach: standard Go pattern, documented in stdlib; `Setsid` field confirmed present in `syscall.SysProcAttr` on Unix (darwin, linux)
- `signal.NotifyContext` for graceful shutdown: Go 1.16+ stdlib, confirmed available in go1.26.1
- `cmd.Process.Release()` to disown child: documented in `os.Process.Release` godoc

### Tertiary (LOW confidence)
- Windows `CREATE_NEW_PROCESS_GROUP` as SysProcAttr equivalent for daemon detach: conventional pattern, not verified against this project's CI

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all stdlib, no new deps, verified against go.mod and existing code
- Architecture: HIGH — derived directly from reading Phase 19 code and requirements; relay/webserver coupling is mechanically evident
- Pitfalls: HIGH for process lifecycle (verified patterns), MEDIUM for Windows SysProcAttr (not CI-verified yet)
- Status event replacement: MEDIUM — polling approach is correct but exact implementation details need verification during planning

**Research date:** 2026-03-23
**Valid until:** 2026-06-23 (stdlib-only, stable patterns)
