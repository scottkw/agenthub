# Phase 19: Daemon Core (Engine + IPC) - Research

**Researched:** 2026-03-23
**Domain:** Go HTTP/Unix socket IPC, session state extraction, Go net/http server architecture
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DAEMON-02 | Daemon communicates with clients via HTTP/JSON over Unix socket (named pipe on Windows) | Go stdlib `net.Listen("unix", path)` + `http.Serve` covers full implementation; `net/http` client with custom `DialContext` provides typed client; Windows uses named pipes via `\\.\pipe\agenthub` |
</phase_requirements>

---

## Summary

Phase 19 extracts all session state from `App` into a new `internal/daemon` package containing three files: `engine.go` (SessionEngine — owns sessions, HubManager, status maps), `api.go` (HTTP routes served over a Unix socket), and `client.go` (typed Go client for App to delegate through). This phase operates **in-process** — there is no process separation yet; the daemon HTTP server starts in the same process as the Wails GUI, running on a Unix socket. Process separation comes in Phase 20. The key invariant is: after Phase 19, `App` holds **no authoritative session state** — all reads and writes go through `DaemonClient`.

The protocol is HTTP/JSON over a Unix socket, using Go's standard `net/http` package on both the server and client sides. This is debuggable with `curl --unix-socket`, requires no new dependencies, and the protocol API surface is small (~8 routes). On Windows, named pipes replace Unix sockets; Go handles both through the same `net.Listener` abstraction.

**Primary recommendation:** Use `net.Listen("unix", socketPath)` + `http.Serve(ln, mux)` on the server side, and `http.Client{Transport: &http.Transport{DialContext: unixDialer}}` on the client side. No new library dependencies needed — pure stdlib.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` (stdlib) | go1.26.1 | HTTP server and client over Unix socket | Zero deps, full HTTP/1.1, already used in `webserver` package |
| `encoding/json` (stdlib) | go1.26.1 | JSON request/response serialisation | Already used throughout codebase |
| `net` (stdlib) | go1.26.1 | `net.Listen("unix", path)` for socket creation | Platform-aware, handles both Unix and named pipe paths |
| `context` (stdlib) | go1.26.1 | Request lifecycle, shutdown coordination | Already the codebase standard |

### Supporting (already in go.mod — no new deps)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/google/uuid` | v1.6.0 | Session ID generation (already indirect dep) | Only if switching from current hex-random ID approach |
| `sync` (stdlib) | — | Mutex protection for engine state | Engine has concurrent readers (list) and writers (create/kill) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| stdlib `net/http` | gRPC/protobuf | gRPC explicitly excluded in REQUIREMENTS.md Out of Scope; HTTP/JSON debuggable with curl |
| stdlib `net/http` | Custom binary protocol | Out of Scope; HTTP reuses stdlib, no framing to maintain |
| Unix socket | TCP localhost | Unix socket: no port conflicts, OS-enforced local-only, no firewall issues; TCP: needed for Windows (named pipe preferred instead) |

**Installation:** No new dependencies needed — all stdlib.

---

## Architecture Patterns

### Recommended Package Structure
```
internal/daemon/
├── engine.go       # SessionEngine — owns all session state
├── api.go          # HTTP routes over Unix socket; starts/stops the listener
└── client.go       # DaemonClient — typed Go client used by App
```

### Socket Path Convention
```
~/.config/agenthub/daemon.sock        (macOS/Linux)
\\.\pipe\agenthub-daemon              (Windows)
```

The socket path must be under 104 characters on macOS (108 on Linux) — the `sun_path` field in `sockaddr_un` is platform-limited. `~/.config/agenthub/daemon.sock` expands to ~40 characters even on long usernames — well within limits. Assert the length at startup before calling `net.Listen`.

### Pattern 1: HTTP Server over Unix Socket (stdlib only)
**What:** Serve `net/http` routes over a Unix socket listener. The HTTP protocol is identical — only the transport changes.
**When to use:** Every Unix socket IPC server in this project.

```go
// Source: Go stdlib net/http documentation + net.Listen man page
func (a *API) Start(socketPath string) error {
    if len(socketPath) > 104 {
        return fmt.Errorf("daemon: socket path too long (%d chars, max 104): %s", len(socketPath), socketPath)
    }
    ln, err := net.Listen("unix", socketPath)
    if err != nil {
        return fmt.Errorf("daemon: listen: %w", err)
    }
    a.ln = ln
    go http.Serve(ln, a.mux) //nolint:errcheck
    return nil
}
```

### Pattern 2: Stale Socket Auto-Removal (ECONNREFUSED probe)
**What:** On startup, if the socket file already exists, probe it with a connection attempt. If ECONNREFUSED, the previous daemon crashed — remove the socket and rebind. If the probe succeeds, another daemon is running.
**When to use:** Every Unix socket server that needs crash recovery.

```go
// Source: standard Unix daemon convention, verified against Go net package behavior
func cleanupStaleSocket(path string) error {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return nil // no socket file, nothing to clean
    }
    // Probe: if ECONNREFUSED, the process is gone — safe to remove.
    conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
    if err != nil {
        // ECONNREFUSED or timeout — stale socket, remove it.
        return os.Remove(path)
    }
    conn.Close()
    // A daemon responded — do NOT remove; return error so caller can abort.
    return fmt.Errorf("daemon already running at %s", path)
}
```

### Pattern 3: HTTP Client over Unix Socket
**What:** `http.Client` with a custom `DialContext` that connects to the Unix socket regardless of the URL host.
**When to use:** `DaemonClient` — the only client in Phase 19.

```go
// Source: Go stdlib net/http documentation
func newUnixClient(socketPath string) *http.Client {
    return &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                var d net.Dialer
                return d.DialContext(ctx, "unix", socketPath)
            },
        },
    }
}
// Usage: client.Get("http://daemon/sessions") — host is a placeholder, ignored by transport
```

### Pattern 4: SessionEngine — Extracted State
**What:** A struct that owns everything currently spread across `App`: `registry`, `backend`, `manager`, `tabNames`, `cliPaths`, `sessionStatuses`, and the `webServer`.
**When to use:** This is the primary extraction target for Phase 19.

```go
// Extracted from app.go — owns authoritative session state
type SessionEngine struct {
    registry *pty.SessionRegistry
    backend  pty.SessionBackend
    manager  *relay.HubManager

    mu              sync.RWMutex
    tabNames        map[string]string
    cliPaths        map[string]string

    statusMu        sync.RWMutex
    sessionStatuses map[string]status.SessionStatus
}
```

### Pattern 5: API Route Set (HTTP/JSON)
All routes use stdlib pattern matching (Go 1.22+ path parameters with `{id}`). The project's `go.mod` already specifies `go 1.26.1`.

```
GET    /health                  → {"status":"ok"}
GET    /sessions                → []SessionInfo
POST   /sessions                → CreateRequest → {"id":"..."}
DELETE /sessions/{id}           → 204 No Content
GET    /sessions/{id}           → SessionInfo
PATCH  /sessions/{id}/name      → {"name":"..."} → 204
GET    /sessions/{id}/status    → {"status":"running|waiting|idle|errored"}
GET    /settings/cli-paths      → map[string]string
PATCH  /settings/cli-paths/{name} → {"path":"..."} → 204
```

**Web server routes** (for Phase 20+, but define now for completeness):
```
POST   /webserver/start         → {"port":N}
POST   /webserver/stop          → 204
GET    /webserver/status        → {"running":bool,"url":"..."}
POST   /sessions/{id}/web-serve → {"enabled":bool} → 204
```

Success criteria says "session CRUD, web serving, health, and settings" — all routes above satisfy this.

### Pattern 6: App → DaemonClient Delegation
**What:** After extraction, every `App` method that previously accessed `a.registry`, `a.tabNames`, etc. now calls through `a.client`.
**When to use:** Every Wails-bound method in `app.go` after Phase 19.

```go
// Before (Phase 18): direct state access
func (a *App) ListSessions() []SessionInfo {
    sessions := a.registry.List()
    // ... tabNames lookup
}

// After (Phase 19): delegate to client
func (a *App) ListSessions() []SessionInfo {
    return a.client.ListSessions()
}
```

### Anti-Patterns to Avoid
- **Split state:** After Phase 19, `App` must contain ZERO authoritative session maps — no `tabNames`, no `sessionStatuses`, no `registry` field. Any leftover local map creates silent divergence.
- **Direct struct embedding of Engine in App:** Engine must be behind the client interface, not embedded; Phase 20 will replace the in-process socket with an out-of-process one without changing `App`.
- **Long socket paths:** Do NOT put sockets in `/tmp/agenthub-<username>/` style paths with long usernames — check length before binding.
- **Ignoring EADDRINUSE on the socket:** `net.Listen("unix", path)` returns `EADDRINUSE` if the file exists even if no process owns it. Always probe + remove before rebinding.
- **Missing socket cleanup on shutdown:** Remove the socket file in `shutdown()` so the next startup finds a clean slate.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP routing | Custom router | Go 1.22+ stdlib `net/http` with `{id}` path params | Already used in `webserver` package; no new dep |
| JSON codec | Manual marshal | `encoding/json` stdlib | Already used everywhere |
| Unix socket server | Custom frame protocol | `net.Listen("unix")` + `http.Serve` | Identical to TCP; debuggable with curl |
| Unix socket client | Custom dialer | `http.Transport.DialContext` pointing at Unix socket | Stdlib; established pattern |
| Session ID generation | UUID library | Current `generateID()` in `internal/pty/native.go` (hex-rand) | Already correct; do not change format mid-migration |

**Key insight:** The entire IPC layer is stdlib HTTP. The hard part is not the transport — it is ensuring `App` holds absolutely no authoritative session state after extraction.

---

## Common Pitfalls

### Pitfall 1: Residual State in App
**What goes wrong:** After migration, `App` still has `tabNames` or `sessionStatuses` maps that drift from the engine's truth.
**Why it happens:** Incremental migration — you move some methods but leave old maps in place "temporarily."
**How to avoid:** Plan migration as a single wave: add all engine methods, update all App methods, then delete all state fields from `App` in the same task.
**Warning signs:** Tests that pass by hitting `App`'s local state rather than going through the client.

### Pitfall 2: Socket Path Length Panic
**What goes wrong:** `net.Listen("unix", path)` returns `bind: invalid argument` or `ENAMETOOLONG` with no useful context.
**Why it happens:** macOS `sun_path` is 104 bytes max (including null terminator), Linux is 108.
**How to avoid:** Assert `len(socketPath) <= 103` before calling `net.Listen`, returning a clear error message.
**Warning signs:** Tests pass on CI (short paths) but fail for users with long home directory paths.

### Pitfall 3: Stale Socket File After Crash
**What goes wrong:** Second startup fails with `EADDRINUSE` because the socket file exists but no process is listening.
**Why it happens:** Crash or `kill -9` — the socket file is not cleaned up.
**How to avoid:** Implement the ECONNREFUSED probe pattern (see Pattern 2 above) before every `net.Listen("unix", ...)`.
**Warning signs:** Tests pass (they clean up) but manual crash testing fails on second start.

### Pitfall 4: Status Watcher Goroutine Leak
**What goes wrong:** `go status.Watch(hub, ...)` goroutines created in `CreateSession` never stop when the session is killed, because they block on `hub.Done()` which was already closed — but new goroutines started after engine restart create duplicates.
**Why it happens:** Moving `Watch` goroutine launch into `SessionEngine.CreateSession` without tracking the goroutine count.
**How to avoid:** `status.Watch` already returns when `hub.Done()` is closed (see `internal/status/detector.go` line 231). Verify `hub.Done()` is closed on `KillSession`.
**Warning signs:** Race detector flags access to `sessionStatuses` after `KillSession`.

### Pitfall 5: In-Process Socket — Port 0 vs Socket Path
**What goes wrong:** Calling `net.Listen("unix", socketPath)` from within the Wails process blocks if `socketPath` is used before the engine starts.
**Why it happens:** `App.startup()` is called by Wails before first render; engine must be started inside `startup()`.
**How to avoid:** Start `SessionEngine` and `DaemonAPI` inside `App.startup()`, same pattern as the relay listener.

### Pitfall 6: Windows Named Pipe
**What goes wrong:** Shipping macOS/Linux Unix socket code fails on Windows.
**Why it happens:** Windows does not have `AF_UNIX` sockets in older builds; `net.Listen("unix", ...)` may work on Windows 10 1803+ (build 17063+), but named pipes are the conventional Windows IPC.
**How to avoid:** The STATE.md notes that Windows CI should be established during Phase 19. For Phase 19, use a build-tag-guarded socket path: `//./pipe/agenthub-daemon` on Windows, `~/.config/agenthub/daemon.sock` elsewhere. `net.Listen("unix", ...)` actually works on modern Windows builds — verify in CI before deciding. Named pipes are fallback.
**Warning signs:** `go test ./internal/daemon/... -run TestSocket` fails on the Windows CI runner.

---

## Code Examples

### Engine Constructor (from extracted App state)
```go
// Source: Derived from app.go NewApp() — direct extraction
func NewSessionEngine() *SessionEngine {
    registry := pty.NewSessionRegistry()
    backend := pty.NewNativePTYBackend()
    manager := relay.NewHubManager()

    return &SessionEngine{
        registry:        registry,
        backend:         backend,
        manager:         manager,
        tabNames:        make(map[string]string),
        cliPaths:        make(map[string]string),
        sessionStatuses: make(map[string]status.SessionStatus),
    }
}
```

### CreateSession on Engine (from App.CreateSession)
```go
// Source: Derived from app.go lines 123-169
func (e *SessionEngine) CreateSession(ctx context.Context, cli, name, workDir string) (string, error) {
    cliPath := e.resolveCLI(cli)
    sess, err := e.backend.Create(ctx, pty.CreateRequest{
        CLI: cliPath, Cols: 80, Rows: 24, WorkDir: workDir,
    })
    if err != nil {
        return "", fmt.Errorf("create session: %w", err)
    }
    e.registry.Add(sess)
    id := sess.ID
    resizeFn := func(cols, rows int) error { return e.backend.Resize(id, cols, rows) }
    hub := e.manager.Create(id, sess, sess, resizeFn)

    e.mu.Lock()
    e.tabNames[id] = name
    e.mu.Unlock()

    go status.Watch(hub, id, cli, func(sid string, s status.SessionStatus) {
        e.statusMu.Lock()
        e.sessionStatuses[sid] = s
        e.statusMu.Unlock()
        // NOTE: no Wails EventsEmit here — App layer handles events via polling or
        // a separate SSE/WebSocket channel in Phase 20+
    })
    return id, nil
}
```

### DaemonClient ListSessions
```go
// Source: Derived from Go net/http stdlib transport pattern
func (c *DaemonClient) ListSessions() ([]SessionInfo, error) {
    resp, err := c.http.Get("http://daemon/sessions")
    if err != nil {
        return nil, fmt.Errorf("daemon: list sessions: %w", err)
    }
    defer resp.Body.Close()
    var sessions []SessionInfo
    if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
        return nil, fmt.Errorf("daemon: decode sessions: %w", err)
    }
    return sessions, nil
}
```

### API Health Handler
```go
// Source: Standard Go net/http handler pattern
func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) //nolint:errcheck
}
```

### App.CreateSession after delegation (Phase 19 result)
```go
// After Phase 19 — App has no local session state
func (a *App) CreateSession(cli, name, workDir string) (string, error) {
    return a.client.CreateSession(cli, name, workDir)
}

func (a *App) ListSessions() []SessionInfo {
    sessions, err := a.client.ListSessions()
    if err != nil {
        return []SessionInfo{}
    }
    return sessions
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Custom binary protocol | HTTP/JSON over Unix socket | Phase 19 (v1.3) | Debuggable with curl, no new deps, standard error codes |
| Session state in App struct | SessionEngine in internal/daemon | Phase 19 (v1.3) | Enables process separation in Phase 20 without changing App API |
| In-process state sharing | In-process Unix socket (Phase 19) → out-of-process (Phase 20) | Incremental across phases | Phase 19 validates protocol correctness before separation |

**Deprecated/outdated:**
- `App.registry`, `App.tabNames`, `App.cliPaths`, `App.sessionStatuses` fields: removed at the end of Phase 19 — all state lives in `SessionEngine`.
- Direct `a.backend.Create(...)` calls from `App`: replaced by `a.client.CreateSession(...)`.

---

## Open Questions

1. **Wails EventsEmit for session status after extraction**
   - What we know: `App.CreateSession` currently calls `runtime.EventsEmit` on status transitions. The callback is deep in `status.Watch`, which runs as a goroutine started from `App.CreateSession`.
   - What's unclear: After extraction, `SessionEngine` must not import Wails — it has no `ctx` with the `"frontend"` key. Options: (a) App passes a callback to `engine.CreateSession` that wraps `runtime.EventsEmit`; (b) Engine exposes a channel/subscription for status events; (c) App polls `GetSessionStatus`.
   - Recommendation: Pass an `onStatusChange func(sessionID string, status string)` callback into `engine.CreateSession`. App provides `runtime.EventsEmit` wrapper as the callback. This keeps Engine free of Wails imports while preserving real-time status events for the GUI. This is the same function-injection pattern already used in `webserver.SetSessionResolver`.

2. **Windows Unix socket support**
   - What we know: `net.Listen("unix", ...)` was added to Windows in Go 1.16+ for Windows 10 Build 17063+.
   - What's unclear: Minimum Windows version the project supports.
   - Recommendation: Try `net.Listen("unix", socketPath)` on Windows first; fail fast with a clear message if it errors. Named pipe fallback can be Phase 20 scope if needed, but modern Windows 10/11 should work.

3. **Socket path on macOS sandbox / app bundle**
   - What we know: macOS `.app` bundles run in a sandbox when distributed via the App Store; `UserConfigDir()` may resolve to a container path.
   - What's unclear: Current distribution model (direct download vs App Store).
   - Recommendation: Use `os.UserConfigDir()` for the socket path (same as `configDir()` in `app.go`). Already handles both sandboxed and non-sandboxed environments.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (go1.26.1) |
| Config file | None — standard `go test` |
| Quick run command | `go test -race ./internal/daemon/...` |
| Full suite command | `go test -race ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DAEMON-02 | HTTP/JSON over Unix socket — server binds and responds | unit | `go test -race ./internal/daemon/... -run TestAPI` | ❌ Wave 0 |
| DAEMON-02 | Session CRUD via HTTP routes | unit | `go test -race ./internal/daemon/... -run TestSessionCRUD` | ❌ Wave 0 |
| DAEMON-02 | DaemonClient round-trips (create, list, kill, rename) | unit | `go test -race ./internal/daemon/... -run TestClient` | ❌ Wave 0 |
| DAEMON-02 | Stale socket auto-removed on ECONNREFUSED | unit | `go test -race ./internal/daemon/... -run TestStaleSocket` | ❌ Wave 0 |
| DAEMON-02 | Socket path length assertion fires before bind | unit | `go test -race ./internal/daemon/... -run TestSocketPathLength` | ❌ Wave 0 |
| DAEMON-02 | App delegates through DaemonClient (no direct state) | integration | `go test -race . -run TestCreate` (existing tests must still pass) | ✅ existing |
| DAEMON-02 | SessionEngine holds all state; App holds none | integration | `go test -race . -run Test` (all existing app_test.go tests) | ✅ existing |

### Sampling Rate
- **Per task commit:** `go test -race ./internal/daemon/... ./...`
- **Per wave merge:** `go test -race ./...`
- **Phase gate:** `go test -race ./...` fully green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/daemon/engine_test.go` — unit tests for SessionEngine CRUD, covers DAEMON-02
- [ ] `internal/daemon/api_test.go` — HTTP handler tests over real Unix socket, covers DAEMON-02
- [ ] `internal/daemon/client_test.go` — DaemonClient round-trip tests, covers DAEMON-02
- [ ] `internal/daemon/socket_test.go` — stale socket + path length tests, covers DAEMON-02

---

## Sources

### Primary (HIGH confidence)
- Go stdlib `net` package — `net.Listen("unix", path)` documentation; standard library behavior verified against go1.26.1 which is already in go.mod
- `internal/relay/server.go` — existing Unix socket pattern analogous to relay HTTP server (TCP variant)
- `internal/webserver/server.go` — existing `setupRoutes()` with Go 1.22+ path parameter syntax (`{id}`)
- `app.go` — authoritative source for all state fields to be extracted to `SessionEngine`
- `app_test.go` — existing test suite that must pass unchanged after Phase 19
- `internal/status/detector.go` — Watch goroutine lifecycle (hub.Done() pattern)

### Secondary (MEDIUM confidence)
- macOS `sun_path` limit: 104 bytes (POSIX constant `sizeof(sockaddr_un.sun_path)`, macOS-specific 104 vs Linux 108)
- Windows Unix socket support: Go 1.16+ on Windows 10 Build 17063+, documented in Go release notes

### Tertiary (LOW confidence)
- Named pipe fallback for older Windows: conventional approach but not verified against this project's supported Windows versions

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all stdlib, no new deps, verified against go.mod
- Architecture: HIGH — directly derived from reading existing `App`, `SessionRegistry`, `HubManager` source
- Pitfalls: HIGH for socket/stale-file issues (verified Unix behavior), MEDIUM for Windows named pipe
- Test gaps: HIGH — confirmed no `internal/daemon/` package exists yet

**Research date:** 2026-03-23
**Valid until:** 2026-06-23 (stdlib-only, stable)
