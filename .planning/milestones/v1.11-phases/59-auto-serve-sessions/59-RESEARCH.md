# Phase 59: Auto-Serve Sessions - Research

**Researched:** 2026-04-09
**Domain:** Go daemon startup (process.go), Tailscale health-check, webserver.Config, React/TypeScript frontend state wiring
**Confidence:** HIGH

## Summary

Phase 59 makes the web server start automatically when the daemon launches and makes every new session web-served by default. The existing codebase already has all of the primitives needed — `runDaemonCore` in `internal/daemon/process.go`, `handleWebServerStart` in `internal/daemon/api.go`, `StartWebServer` in `app.go`, and `handleCreateSession` in `api.go`. No new packages are required.

**SERVE-01** is entirely a daemon-side change. `runDaemonCore` must call `webserver.CheckHealth` after the API starts and, if Tailscale is ready (connected + has certs + has an IP), call `api.StartWebServerAuto()` (or an equivalent internal helper on `*API`) to start the web server before blocking on `<-ctx.Done()`. Because the daemon runs in a detached process, it has no Wails context — it must call the Tailscale health check directly using `webserver.CheckHealth(ctx)`, not through `app.go`.

**SERVE-02** is also a daemon-side change. `handleCreateSession` in `api.go` must automatically call `ws.EnableSession(id)` immediately after the session is created, when the web server is already running (`a.webServer != nil`). The frontend `webEnabled` state map is seeded from the daemon's actual web-server toggle state when the app restores sessions (via `IsWebServerRunning` + `ToggleWebServing`). Because the frontend currently has no way to ask "which sessions are currently web-enabled?", a new daemon API endpoint or response field is needed — or the frontend must call `ToggleWebServing(id, true)` for every new session it creates when `webServerRunning` is true.

**Primary recommendation:** (1) Add auto-start call at the bottom of `runDaemonCore` after `api.Start()`; (2) add auto-enable call in `handleCreateSession` when `a.webServer != nil`; (3) update frontend `createTab` to call `ToggleWebServing(id, true)` and seed `webEnabled` state when web server is running.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (`context`, `time`) | go1.26.2 | Context with timeout for Tailscale health check at startup | Already used throughout daemon |
| `github.com/scottkw/agenthub/internal/webserver` | local | `webserver.CheckHealth`, `webserver.Config` | Already used in `api.go` and `app.go` |
| `tailscale.com/client/local` | current | TLS cert / IP resolution | Already imported by webserver package |
| React/TypeScript (Vitest 4.x) | 4.1.0 | Frontend state wiring for auto-enabled sessions | Already configured |

### Supporting
None — this phase uses only existing project dependencies.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Calling `webserver.CheckHealth` in daemon at start | Polling in a goroutine | Polling adds complexity; a single check at startup is correct given Tailscale is already connected before the daemon spawns in typical use |
| Auto-enabling all sessions in daemon `handleCreateSession` | Auto-enabling in the frontend after `CreateSession` RPC returns | Daemon-side is the authoritative location — keeps `webEnabled` consistent even if the daemon is accessed by multiple clients in the future |
| Adding a new GET /sessions/{id}/web-status endpoint | Returning web-enabled flag in SessionInfo | Embedding in `SessionInfo` is simpler but changes the existing type contract. A separate per-session field or a GET /webserver/sessions endpoint is cleaner |

**Installation:**
```bash
# No new packages required
```

## Architecture Patterns

### Where Each Change Lives

```
internal/daemon/
├── process.go           # SERVE-01: auto-start web server at end of runDaemonCore
├── api.go               # SERVE-02: auto-enable session in handleCreateSession
│                        #           new autoStartWebServer() helper on *API
└── types.go             # possibly: add WebEnabled bool to SessionInfo if chosen

frontend/src/
├── App.tsx              # SERVE-02: seed webEnabled + call ToggleWebServing in createTab
│                        #           and in session restore on init
└── wailsjs/go/main/App.ts  # auto-generated — regenerated if new bound methods added
```

### Pattern 1: Auto-Start Web Server in runDaemonCore (SERVE-01)

**What:** After `api.Start()` succeeds, check Tailscale health synchronously. If healthy, call a new `api.autoStartWebServer()` method that mirrors the logic in `handleWebServerStart` but without an HTTP request.

**When to use:** At daemon startup only — not in a poller.

**Example:**
```go
// internal/daemon/process.go — at the bottom of runDaemonCore, after api.Start()
ctx5s, cancel := context.WithTimeout(ctx, 5*time.Second)
h := webserver.CheckHealth(ctx5s)
cancel()
if h.Connected && h.HasCerts && h.IP != "" {
    if err := api.AutoStartWebServer(h.IP, 7443, h.Domain); err != nil {
        fmt.Fprintf(os.Stderr, "daemon: auto-start web server: %v\n", err)
    } else {
        fmt.Fprintf(os.Stderr, "daemon: web server auto-started\n")
    }
}

<-ctx.Done()
```

**Example — new method on *API (api.go):**
```go
// AutoStartWebServer starts the web server if it is not already running.
// Called from runDaemonCore; mirrors handleWebServerStart without HTTP.
func (a *API) AutoStartWebServer(ip string, port int, fqdn string) error {
    a.mu.Lock()
    defer a.mu.Unlock()
    if a.webServer != nil {
        return nil // already running
    }
    ws, err := webserver.NewWebServer(webserver.Config{
        BindIP: ip,
        Port:   port,
        FQDN:   fqdn,
    }, a.engine.Manager())
    if err != nil {
        return err
    }
    ws.SetSessionResolver(func(sessionID string) (name, cliType, status, hostname string) {
        for _, s := range a.engine.ListSessions() {
            if s.ID == sessionID {
                return s.Name, s.CLI, a.engine.GetSessionStatus(sessionID), s.Hostname
            }
        }
        return sessionID, "", "", ""
    })
    if err := ws.Start(); err != nil {
        return err
    }
    a.webServer = ws
    return nil
}
```

### Pattern 2: Auto-Enable Session on Create (SERVE-02, daemon side)

**What:** In `handleCreateSession`, after the session ID is obtained, check if the web server is running. If so, call `ws.EnableSession(id)` immediately.

**Example:**
```go
// internal/daemon/api.go — at end of handleCreateSession, after id is set
a.mu.RLock()
ws := a.webServer
a.mu.RUnlock()
if ws != nil {
    ws.EnableSession(id)
}
writeJSON(w, http.StatusCreated, CreateResponse{ID: id})
```

### Pattern 3: Frontend Reflects Auto-Enabled State (SERVE-02, frontend side)

**What:** When `createTab` creates a session and `webServerRunning` is true, immediately set `webEnabled[sessionId] = true` and populate `sessionURLs`. This mirrors what `handleToggleWeb` does, but without calling `ToggleWebServing` (the daemon already auto-enabled it).

**When to use:** In the `createTab` callback in `App.tsx`, after `CreateSession` returns.

**Example:**
```typescript
// frontend/src/App.tsx — inside createTab, after sessionId is obtained
if (webServerRunning) {
  setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))
  const url = await GetWebServerURL()
  if (url) {
    setSessionURLs((prev) => ({ ...prev, [sessionId]: `${url}/sessions/${sessionId}` }))
  }
}
```

Also: the session restore path (init function in `useEffect`) must seed `webEnabled` for sessions that were auto-enabled before the window opened. The daemon needs to expose which sessions are web-enabled. See "Open Questions" below for the two options.

### Pattern 4: Restoring webEnabled State After Window Re-open

The frontend's `init()` already calls `IsWebServerRunning()`. When it restores existing sessions from `ListSessions()`, it must also populate `webEnabled` for those sessions.

**Option A (recommended):** Add a `webEnabled bool` field to `SessionInfo` in the daemon's `types.go` and populate it from `a.webServer.isSessionEnabled(s.ID)` in `handleListSessions`. The frontend then reads this field during session restore.

**Option B:** Add a new `GET /webserver/sessions` endpoint returning `[]string` of enabled session IDs. More surgical but requires a new endpoint + client method.

Option A is simpler and keeps session info self-contained.

### Anti-Patterns to Avoid

- **Tailscale check in a poller at startup:** Adds unnecessary goroutine complexity. A single synchronous check with a short timeout is sufficient for the auto-start case. If Tailscale is not ready at daemon start, the server is not started (user can still start manually from Settings).
- **Starting the web server before `api.Start()` returns:** The relay must be up before the web server; keep the ordering `StartRelay` → `Start` (socket) → `AutoStartWebServer`.
- **Calling `app.go`'s `StartWebServer` from the daemon:** `app.go` is Wails-only. The daemon uses `webserver.CheckHealth` and `webserver.NewWebServer` directly.
- **Forgetting `SetSessionResolver` in `AutoStartWebServer`:** The existing `handleWebServerStart` always sets the resolver before `ws.Start()`. `AutoStartWebServer` must do the same or the session list on the web dashboard will show only IDs.
- **Double-starting if user manually starts in Settings:** `AutoStartWebServer` must guard with `if a.webServer != nil { return nil }`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tailscale health check | Custom tailscaled socket probe | `webserver.CheckHealth(ctx)` | Already implemented, tested, and handles all edge cases |
| Web server construction | New constructor | `webserver.NewWebServer(cfg, manager)` | Already implements TLS, port fallback, session resolver pattern |
| Session enable/disable | Custom registry | `ws.EnableSession(id)` / `ws.DisableSession(id)` | Already mutex-protected on `WebServer` |

**Key insight:** All primitives exist. This phase is exclusively wiring, not new infrastructure.

## Common Pitfalls

### Pitfall 1: Tailscale Not Connected at Daemon Startup
**What goes wrong:** Daemon starts (e.g., on login via LaunchAgent) before Tailscale daemon is fully up. `webserver.CheckHealth` returns `Connected: false`. Web server does not auto-start.
**Why it happens:** macOS LaunchAgent start order is not strictly guaranteed. Tailscale daemon may take a few seconds after login before `BackendState == "Running"`.
**How to avoid:** This is acceptable behavior per the requirements — the requirements say "when the server is running." If Tailscale is not ready at daemon start, the user can still start the web server manually from Settings. A retry loop is NOT needed for this phase.
**Warning signs:** Integration test environment where Tailscale is absent — ensure test does not fail, just skips auto-start.

### Pitfall 2: webEnabled State Drift Between Daemon and Frontend
**What goes wrong:** Daemon auto-enables sessions; frontend's `webEnabled` map is empty (initial state). After re-opening the window, the toggle buttons show "Web Off" for sessions that are actually web-enabled.
**Why it happens:** `webEnabled` is frontend React state, not persisted. The daemon's `WebServer.webEnabled` map is the source of truth.
**How to avoid:** Choose Option A (embed `webEnabled` in `SessionInfo`) or Option B (new endpoint). Session restore path in `init()` must seed React state from daemon truth.

### Pitfall 3: Auto-Enable on Create Races With Session Registration
**What goes wrong:** `ws.EnableSession(id)` is called before the session is fully registered in the relay hub.
**Why it happens:** `EnableSession` only sets a flag in `webEnabled map[string]bool` — it doesn't interact with the hub. The hub is created in `engine.CreateSession` before `api.handleCreateSession` returns. So there is no race; `EnableSession` just marks the session as web-accessible.
**How to avoid:** No special ordering needed. The flag is a simple map write; the hub will exist by the time a browser actually connects via WebSocket.

### Pitfall 4: Daemon Restart Behavior (Success Criterion #4)
**What goes wrong:** `runDaemonCore` calls `api.Stop()` which calls `ws.Stop()` which closes the TLS listener. The next `runDaemonCore` call (new daemon process) then calls `AutoStartWebServer` again — this is correct.
**Why it happens:** Each daemon process owns its `*API` and `*WebServer`. A restart is a new process with a fresh `API.webServer = nil`, so the auto-start logic runs again cleanly.
**How to avoid:** No special logic needed. The existing `api.Stop()` → new process → `AutoStartWebServer` flow handles this naturally.

### Pitfall 5: Frontend Shows "Web Off" Button While Server Is Running
**What goes wrong:** `DaemonManagerPanel` renders the "Web On/Off" button as disabled when `webServerRunning` is false. If the auto-started server is running but the frontend polled `IsWebServerRunning()` before it finished starting, the button appears greyed.
**Why it happens:** The `init()` function polls `IsWebServerRunning()` at window open. If `AutoStartWebServer` has not yet returned when the GUI's `EnsureDaemon` completes and `init()` fires, the poll sees `false`.
**How to avoid:** `AutoStartWebServer` is called synchronously in `runDaemonCore` before `<-ctx.Done()`. `EnsureDaemon` in `app.go` polls until the daemon is health-checked AND relay port is ready. By the time the frontend's `init()` fires, `AutoStartWebServer` will have completed. This pitfall is therefore not a real risk in practice — but should be confirmed with an integration test.

## Code Examples

### Existing handleWebServerStart (reference pattern for AutoStartWebServer)
```go
// Source: internal/daemon/api.go — handleWebServerStart
func (a *API) handleWebServerStart(w http.ResponseWriter, r *http.Request) {
    var req WebServerStartRequest
    // ...decode...
    ws, err := webserver.NewWebServer(webserver.Config{
        BindIP: req.IP,
        Port:   req.Port,
        FQDN:   req.FQDN,
    }, a.engine.Manager())
    // ...
    ws.SetSessionResolver(func(sessionID string) (name, cliType, status, hostname string) { ... })
    if err := ws.Start(); err != nil { ... }
    a.mu.Lock()
    a.webServer = ws
    a.mu.Unlock()
    writeJSON(w, http.StatusOK, WebServerStartResponse{URL: ws.BaseURL()})
}
```

### Existing handleCreateSession (target for auto-enable injection)
```go
// Source: internal/daemon/api.go — handleCreateSession
func (a *API) handleCreateSession(w http.ResponseWriter, r *http.Request) {
    // ...decode req...
    id, err := a.engine.CreateSession(...)
    if err != nil { ... }
    writeJSON(w, http.StatusCreated, CreateResponse{ID: id})
    // ^^^ inject ws.EnableSession(id) here, before writeJSON
}
```

### Existing createTab (target for webEnabled seeding)
```typescript
// Source: frontend/src/App.tsx — createTab
const sessionId = await CreateSession(cliName, defaultName, workDir, args, cols, rows)
const tab: Tab = { id: sessionId, name: defaultName, sessionId, cli: cliName }
setTabs((prev) => [...prev, tab])
setActiveId(sessionId)
// ^^^ inject webEnabled seeding here
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual "Start Web Server" in Settings tab | Auto-start at daemon launch (this phase) | Phase 59 | Server always available without user action |
| Manual "Web On/Off" per session | Auto-enabled for new sessions (this phase) | Phase 59 | Sessions are web-served by default |

**Deprecated/outdated:**
- None — the Settings "Start/Stop Web Server" button remains valid for manual override.

## Open Questions

1. **How to expose web-enabled state per session to the frontend for session restore?**
   - What we know: `WebServer.webEnabled` is the source of truth inside the daemon. `SessionInfo` in `types.go` is the DTO returned by `GET /sessions`.
   - What's unclear: Option A (add `WebEnabled bool` to `SessionInfo`) vs Option B (new endpoint `GET /webserver/sessions` returning `[]string`).
   - Recommendation: Option A — embed in `SessionInfo`. The field is naturally co-located with the session. The Go type and the Wails-generated TypeScript type both update together. Impact is minimal: one new boolean field, backward-compatible JSON.

2. **Should the DaemonManagerPanel web toggle button remain visible/interactive when web server is auto-started?**
   - What we know: The button is currently disabled when `webServerRunning` is false. With auto-start, `webServerRunning` will be true by the time the panel is shown.
   - What's unclear: Should the toggle be allowed to turn OFF for individual sessions (diverging from "all on by default")?
   - Recommendation: Keep the toggle functional. Users should be able to disable web serving per session if they want. The "default ON" behavior applies only at creation time.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build + test | Yes | 1.26.2 | — |
| Node.js | Frontend tests | Yes | 20.19.3 | — |
| pnpm | Frontend deps | Yes | 9.12.3 | — |
| Tailscale daemon | Production auto-start | Not required for tests | — | `webserver.CheckHealth` returns `Connected: false`, auto-start skipped |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** Tailscale is only needed in production; unit tests inject a fake health response.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `testing` package + `go test` |
| Framework (frontend) | Vitest 4.1.0 |
| Config file (Go) | None (standard `go test`) |
| Config file (frontend) | `frontend/vite.config.ts` |
| Quick run command (Go) | `go test ./internal/daemon/... -run TestAutoServe` |
| Full suite command (Go) | `go test ./...` |
| Full suite command (frontend) | `cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SERVE-01 | Web server starts automatically when daemon launches (Tailscale connected) | unit (Go) | `go test ./internal/daemon/... -run TestAutoStartWebServer` | No — Wave 0 |
| SERVE-01 | Web server does NOT start when Tailscale not connected | unit (Go) | `go test ./internal/daemon/... -run TestAutoStartWebServer_NoTailscale` | No — Wave 0 |
| SERVE-02 | New session auto-enables web serving when server is running | unit (Go) | `go test ./internal/daemon/... -run TestCreateSession_AutoWebEnable` | No — Wave 0 |
| SERVE-02 | New session does NOT auto-enable when server is not running | unit (Go) | `go test ./internal/daemon/... -run TestCreateSession_NoAutoEnable` | No — Wave 0 |
| SERVE-02 | Frontend webEnabled seeded for new session when webServerRunning | unit (TS) | `cd frontend && pnpm test --run App` | Partial (App.test.tsx exists) |

### Sampling Rate
- **Per task commit:** `go test ./internal/daemon/... && cd frontend && pnpm test`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/daemon/api_test.go` — `TestAutoStartWebServer`: inject `SetWebServerForTest` pattern from existing `TestAPIWebServe_NoServer`; verify `AutoStartWebServer` starts the server
- [ ] `internal/daemon/api_test.go` — `TestCreateSession_AutoWebEnable`: create session when web server is pre-set via `SetWebServerForTest`, verify `GET /webserver/status` shows session in web-enabled list

*(Frontend tests use existing App.test.tsx infrastructure — extend rather than create from scratch)*

## Sources

### Primary (HIGH confidence)
- Source code: `internal/daemon/process.go` — `runDaemonCore` entry point; confirmed current structure
- Source code: `internal/daemon/api.go` — `handleWebServerStart`, `handleCreateSession`, `SetWebServerForTest` pattern
- Source code: `internal/webserver/server.go` — `NewWebServer`, `EnableSession`, `SetSessionResolver`
- Source code: `internal/webserver/tailscale.go` — `CheckHealth` (injectable for tests)
- Source code: `app.go` — `StartWebServer` (Wails-only, not usable in daemon process)
- Source code: `frontend/src/App.tsx` — `createTab`, session restore init, `webEnabled` state
- Source code: `internal/daemon/types.go` — `SessionInfo` DTO (candidate for `WebEnabled` field)

### Secondary (MEDIUM confidence)
- Source code: `internal/daemon/api_test.go` — `SetWebServerForTest` used in existing tests, confirms testability pattern

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are already in use; no new dependencies
- Architecture: HIGH — all call sites verified by reading actual source files
- Pitfalls: HIGH — derived from reading actual code paths, not conjecture

**Research date:** 2026-04-09
**Valid until:** 2026-05-09 (stable internal codebase; no external API dependency)

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SERVE-01 | Web server starts automatically when daemon launches (no manual start required) | Add `AutoStartWebServer` method to `*API`; call it in `runDaemonCore` after `api.Start()` succeeds; use `webserver.CheckHealth` with 5s timeout to get IP/FQDN/port |
| SERVE-02 | New sessions have web serving enabled automatically when the web server is running | Call `ws.EnableSession(id)` in `handleCreateSession` after session created + web server running; seed frontend `webEnabled` state in `createTab` when `webServerRunning` is true; add `WebEnabled bool` to `SessionInfo` for session-restore path |
</phase_requirements>
