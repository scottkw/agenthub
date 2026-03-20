# Architecture Research

**Domain:** Tailscale-only networking integration — Go/Wails desktop app (v1.2 milestone)
**Researched:** 2026-03-20
**Confidence:** HIGH (official Tailscale Go API docs, source code examples, direct codebase inspection)

---

## Context: What Already Exists

The existing architecture in `internal/webserver/` has these components:

| Component | File | Current Behavior |
|-----------|------|-----------------|
| `WebServer` struct | `server.go` | Holds `caKey`, `caCert`, `caDER`, `tlsCfg`; generates leaf cert in `Start()` |
| TLS layer | `tls.go` | `LoadOrCreateCA`, `GenerateLeafCert`, `BuildTLSConfig` — all self-signed |
| Auth layer | `auth.go` | `AuthManager` (bcrypt password + session cookies) |
| Token layer | `tokens.go` | `TokenStore` — per-session shareable tokens |
| Network detection | `network.go` | `ListInterfaces()` — generic NIC scan, `IsTailscaleIP` by CGNAT range |
| Entry point | `app.go` | `StartWebServer(bindIP, port)` — takes any IP, calls `GenerateLeafCert(bindIP)` |

**Config struct today:**
```go
type Config struct {
    BindIP    string  // any IP — Tailscale or LAN
    Port      int
    ConfigDir string  // for CA cert persistence
}
```

---

## v1.2 Target Architecture

### System Overview

```
┌───────────────────────────────────────────────────────────────────────┐
│                          app.go (App struct)                           │
│                                                                        │
│  StartWebServer(port int)     GetTailscaleStatus() TailscaleHealth     │
│  [no bindIP param — auto]     [new Wails method]                       │
└──────────────────────┬────────────────────────────────────────────────┘
                       │
                       ▼
┌───────────────────────────────────────────────────────────────────────┐
│                      internal/webserver/                               │
│                                                                        │
│  ┌──────────────────┐  ┌─────────────────────┐  ┌──────────────────┐  │
│  │   server.go      │  │   tailscale.go      │  │  network.go      │  │
│  │                  │  │   (NEW FILE)        │  │  (simplified)    │  │
│  │  WebServer       │  │                     │  │                  │  │
│  │  - mux           │◄─│  TailscaleHealth    │  │  GetTailscaleIP  │  │
│  │  - manager       │  │  CheckHealth()      │  │  (replaces       │  │
│  │  - webEnabled    │  │                     │  │  ListInterfaces) │  │
│  │  - sessionRes.   │  │  local.Client{}     │  │                  │  │
│  │                  │  │  .StatusWithout     │  └──────────────────┘  │
│  │  REMOVED:        │  │   Peers(ctx)        │                        │
│  │  - caKey/caCert  │  │  .GetCertificate    │                        │
│  │  - tlsCfg        │  │   (used in Start()) │                        │
│  │  - auth          │  └─────────────────────┘                        │
│  │  - tokens        │                                                  │
│  └────────┬─────────┘                                                  │
│           │                                                            │
│  ┌────────▼─────────┐  ┌─────────────────────┐                        │
│  │   tls.go         │  │   auth.go           │                        │
│  │   (DELETE)       │  │   (DELETE)          │                        │
│  │   tokens.go      │  │   tls_test.go       │                        │
│  │   (DELETE)       │  │   auth_test.go      │                        │
│  │                  │  │   tokens_test.go    │                        │
│  └──────────────────┘  └─────────────────────┘                        │
└───────────────────────────────────────────────────────────────────────┘
                       │
                       ▼ tls.Listen on Tailscale IP (100.x.x.x)
┌───────────────────────────────────────────────────────────────────────┐
│               tailscaled (local daemon — already running)              │
│                                                                        │
│  local.Client{}.GetCertificate(hi *tls.ClientHelloInfo)               │
│  → fetches/caches Let's Encrypt cert from tailscaled                  │
│  → cert CN = <hostname>.tail46d69a.ts.net (or similar ts.net domain)  │
│                                                                        │
│  local.Client{}.StatusWithoutPeers(ctx)                               │
│  → BackendState: "Running" | "NeedsLogin" | "Stopped" | "Starting"   │
│  → TailscaleIPs: []netip.Addr — the 100.x.x.x address                │
│  → CertDomains: []string — non-empty when HTTPS enabled in admin      │
└───────────────────────────────────────────────────────────────────────┘
```

---

## Component Changes: New vs Modified vs Deleted

### NEW: `internal/webserver/tailscale.go`

This file does not exist yet. It owns all Tailscale health-check logic.

**Responsibility:** Query the local Tailscale daemon for connectivity and cert readiness. Surface structured status to `app.go`.

```go
package webserver

import (
    "context"
    "tailscale.com/client/local"
)

// TailscaleHealth is the result of a health check against the local tailscaled.
type TailscaleHealth struct {
    Installed bool   // tailscaled socket reachable (no error from StatusWithoutPeers)
    Connected bool   // BackendState == "Running"
    HasCerts  bool   // len(CertDomains) > 0
    IP        string // first TailscaleIP as string, empty if not connected
    Domain    string // first CertDomain (e.g. hostname.ts.net), empty if none
}

// CheckHealth queries tailscaled via local.Client and returns TailscaleHealth.
// ctx should have a short timeout (3-5 seconds) to avoid blocking the UI.
func CheckHealth(ctx context.Context) TailscaleHealth {
    var lc local.Client  // zero value uses platform default socket
    status, err := lc.StatusWithoutPeers(ctx)
    if err != nil {
        return TailscaleHealth{Installed: false}
    }
    h := TailscaleHealth{Installed: true}
    h.Connected = status.BackendState == "Running"
    h.HasCerts = len(status.CertDomains) > 0
    if len(status.TailscaleIPs) > 0 {
        h.IP = status.TailscaleIPs[0].String()
    }
    if len(status.CertDomains) > 0 {
        h.Domain = status.CertDomains[0]
    }
    return h
}
```

**Three health states the frontend modal must handle:**

| `Installed` | `Connected` | `HasCerts` | Meaning | User action |
|-------------|-------------|------------|---------|-------------|
| false | — | — | tailscaled not running | Install / start Tailscale |
| true | false | — | Installed but not connected | Log in or connect to Tailscale |
| true | true | false | Connected but HTTPS disabled | Enable HTTPS in Tailscale admin DNS settings |
| true | true | true | Ready | Start web server |

---

### MODIFIED: `internal/webserver/server.go`

**Config struct — simplify:**
```go
// v1.2 Config — no BindIP (always Tailscale), no ConfigDir (no CA cert storage)
type Config struct {
    Port int  // preferred port; 0 or unavailable → OS-assigned random
}
```

**WebServer struct — remove TLS/auth fields:**
```go
type WebServer struct {
    config  Config
    manager *relay.HubManager

    mu         sync.RWMutex
    webEnabled map[string]bool
    listener   net.Listener
    mux        *http.ServeMux

    sessionResolver func(sessionID string) (name, cliType, status string)
    // REMOVED: auth, tokens, caKey, caCert, caDER, tlsCfg
}
```

**`NewWebServer` — remove CA loading:**
```go
func NewWebServer(cfg Config, manager *relay.HubManager) (*WebServer, error) {
    // No CA setup — just allocate struct and register routes
    ws := &WebServer{
        config:     cfg,
        manager:    manager,
        webEnabled: make(map[string]bool),
        mux:        http.NewServeMux(),
    }
    ws.setupRoutes()
    return ws, nil
}
```

**`Start()` — use Tailscale TLS instead of self-signed:**
```go
func (ws *WebServer) Start(tailscaleIP string) error {
    var lc local.Client
    tlsCfg := &tls.Config{
        GetCertificate: lc.GetCertificate,
    }
    port := ws.config.Port
    addr := fmt.Sprintf("%s:%d", tailscaleIP, port)
    ln, err := tls.Listen("tcp", addr, tlsCfg)
    if err != nil {
        var opErr *net.OpError
        if errors.As(err, &opErr) && port != 0 {
            addr = fmt.Sprintf("%s:0", tailscaleIP)
            ln, err = tls.Listen("tcp", addr, tlsCfg)
        }
        if err != nil {
            return fmt.Errorf("webserver: listen: %w", err)
        }
    }
    ws.mu.Lock()
    ws.listener = ln
    ws.mu.Unlock()
    go http.Serve(ln, ws.mux) //nolint:errcheck
    return nil
}
```

**`BaseURL()` — now returns Tailscale IP-based URL:**
```go
func (ws *WebServer) BaseURL() string {
    // Returns https://100.x.x.x:port
    // Tailscale's reverse DNS resolves 100.x.x.x to hostname.ts.net,
    // but the IP URL works directly and avoids a DNS lookup dependency.
}
```

**`setupRoutes()` — remove auth routes:**

| Route | v1.1 | v1.2 |
|-------|------|------|
| `POST /login` | present | REMOVE |
| `GET /ca.crt` | present | REMOVE |
| `POST /api/sessions/{id}/token` | present | REMOVE |
| `GET /dashboard` | no auth | keep unchanged |
| `GET /api/sessions` | requires dashboardAuth | open — network is the perimeter |
| `GET /sessions/{id}` | requires sessionAuth (cookie or token) | only `isSessionEnabled` check remains |
| `GET /sessions/{id}/ws` | requires sessionAuth | only `isSessionEnabled` check remains |
| `GET /api/sessions/{id}/qr` | requires dashboardAuth | keep or open — no token needed |

**Auth middleware removal:** `dashboardAuth` and `sessionAuth` functions are deleted. The `isSessionEnabled` guard on session routes stays — it controls which sessions are web-accessible regardless of auth.

---

### MODIFIED: `internal/webserver/network.go`

**Remove entirely:** `ListInterfaces()`, `IsTailscaleIP()`, `tailscaleCIDR` init block, `NetworkInterface` struct, `network_test.go`.

**Add:**
```go
// GetTailscaleIP is a convenience wrapper around CheckHealth for use in app.go.
// Returns the Tailscale IP or an error with a user-friendly message.
func GetTailscaleIP(ctx context.Context) (string, error) {
    h := CheckHealth(ctx)
    if !h.Installed {
        return "", fmt.Errorf("Tailscale is not installed or not running")
    }
    if !h.Connected {
        return "", fmt.Errorf("Tailscale is not connected")
    }
    if h.IP == "" {
        return "", fmt.Errorf("no Tailscale IP address assigned")
    }
    return h.IP, nil
}
```

---

### DELETED: `internal/webserver/tls.go` and `tls_test.go`

All functions (`GenerateCA`, `LoadOrCreateCA`, `GenerateLeafCert`, `BuildTLSConfig`, `ExportCACertPath`, `loadECPrivateKey`, `loadCertificate`) become dead code once `Start()` switches to `local.Client.GetCertificate`. Delete both files.

---

### DELETED: `internal/webserver/auth.go`, `auth_test.go`, `tokens.go`, `tokens_test.go`

`AuthManager`, `TokenStore`, `HashPassword`, `CheckPassword` — all removed. No replacement needed; Tailscale network membership is the auth boundary.

---

### MODIFIED: `app.go`

**Remove Wails methods:**
- `SetWebPassword(password string) error`
- `IsWebPasswordSet() bool`
- `GenerateSessionToken(sessionID string) (string, error)`
- `GetCACertPath() string`
- `GetNetworkInterfaces() []webserver.NetworkInterface`

**Remove internal helpers:**
- `webPasswordPath() string`
- `configDir()` — only needed for CA cert storage; keep only if used elsewhere

**Modify `StartWebServer`:**
```go
// StartWebServer no longer takes bindIP — always uses the Tailscale IP.
func (a *App) StartWebServer(port int) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    health := webserver.CheckHealth(ctx)
    if !health.Installed {
        return fmt.Errorf("Tailscale is not installed or running")
    }
    if !health.Connected {
        return fmt.Errorf("Tailscale is not connected")
    }
    if !health.HasCerts {
        return fmt.Errorf("Tailscale HTTPS certificates not enabled — enable HTTPS in Tailscale admin DNS settings")
    }

    // Stop any running server first.
    a.mu.Lock()
    oldWS := a.webServer
    a.mu.Unlock()
    if oldWS != nil {
        _ = oldWS.Stop()
    }

    ws, err := webserver.NewWebServer(webserver.Config{Port: port}, a.manager)
    if err != nil {
        return fmt.Errorf("StartWebServer: %w", err)
    }
    ws.SetSessionResolver(...)  // unchanged

    if err := ws.Start(health.IP); err != nil {
        return fmt.Errorf("StartWebServer: start: %w", err)
    }

    a.mu.Lock()
    a.webServer = ws
    a.mu.Unlock()
    return nil
}
```

**Add new Wails method:**
```go
// GetTailscaleStatus returns Tailscale health for the settings UI and health modal.
func (a *App) GetTailscaleStatus() webserver.TailscaleHealth {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return webserver.CheckHealth(ctx)
}
```

---

## Data Flow: TLS Cert Provisioning

```
User enables web serving in settings
            ↓
app.go: StartWebServer(port)
            ↓
webserver.CheckHealth(ctx)  [5s timeout]
  → local.Client{}.StatusWithoutPeers(ctx)
  → confirm BackendState == "Running"
  → confirm len(CertDomains) > 0
  → extract TailscaleIPs[0] as bindIP
            ↓
webserver.NewWebServer(Config{Port: port}, manager)
            ↓
ws.Start(tailscaleIP = "100.x.x.x")
  → tls.Config{ GetCertificate: local.Client{}.GetCertificate }
  → tls.Listen("tcp", "100.x.x.x:port", tlsCfg)
            ↓
First HTTPS request arrives
  → TLS handshake triggers GetCertificate callback
  → local.Client talks to tailscaled via Unix socket
  → tailscaled serves cached or fetches fresh Let's Encrypt cert
    for <hostname>.ts.net via DNS-01 ACME challenge
  → Returns *tls.Certificate to the TLS stack
  → Browser receives browser-trusted cert — no CA install required
```

**Cert caching:** `local.Client.GetCertificate` caches on disk inside the tailscaled data directory. Automatic renewal is handled by tailscaled. No cert files in `~/.config/agenthub/`.

---

## Data Flow: Health Check for UI Modal

```
Frontend settings panel loads, or user clicks "Check Status"
            ↓
Wails: app.GetTailscaleStatus()
            ↓
webserver.CheckHealth(ctx)  [5s timeout]
            ↓
Returns TailscaleHealth {
    Installed: bool   → false: show "Install Tailscale" instructions
    Connected: bool   → false: show "Connect to tailnet" instructions
    HasCerts:  bool   → false: show "Enable HTTPS in admin console" link
    IP:        string → display to user when server is running
    Domain:    string → display Tailscale hostname when available
}
            ↓
Frontend renders instructional modal based on first false field
(three distinct user actions, each with specific copy and link)
```

---

## Interface Binding: Old vs New

| Aspect | v1.1 | v1.2 |
|--------|------|------|
| How IP is obtained | `ListInterfaces()` scans all NICs; user picks from dropdown | `lc.StatusWithoutPeers().TailscaleIPs[0]` — direct from daemon |
| Bind address | Any IPv4 on the machine (user selected) | Tailscale CGNAT IP only (100.x.x.x) |
| User choice | Interface dropdown in Settings | None — auto-detected |
| VPN fallback | Generic VPN interface support | Removed — Tailscale only |
| TLS | Self-signed CA + leaf cert | Let's Encrypt via tailscaled |
| Auth | bcrypt password + session cookies + per-session tokens | Network membership (Tailscale ACLs as perimeter) |
| Port | Configurable default 7443 | Configurable, recommend default 443 |

**Port recommendation:** Defaulting to 443 gives clean URLs (`https://hostname.ts.net`) and works without custom firewall rules. The existing port-fallback logic (`EADDRINUSE` → `:0`) handles conflicts. If 443 is taken by another service, fallback to a high port like 7443.

---

## Architectural Patterns

### Pattern 1: GetCertificate as tls.Config Callback (HIGH confidence)

**What:** Set `tls.Config.GetCertificate = lc.GetCertificate` instead of generating or loading cert files. The TLS stack invokes this callback per-connection handshake; `local.Client` returns a cached valid cert or fetches a fresh one from tailscaled.

**When to use:** Any Go HTTPS server that only serves on a Tailscale IP. This is the official pattern documented by Tailscale with a reference implementation in their repository.

**Trade-offs:** Requires tailscaled running when the server starts (first connection fails if tailscaled goes down post-start; TLS handshake errors). No cert management code in the app. Automatic renewal by tailscaled.

**Example:**
```go
var lc local.Client  // zero value uses platform default socket
tlsCfg := &tls.Config{
    GetCertificate: lc.GetCertificate,
}
ln, err := tls.Listen("tcp", tailscaleIP+":443", tlsCfg)
```

### Pattern 2: StatusWithoutPeers for Health Checks (HIGH confidence)

**What:** Use `StatusWithoutPeers` (not `Status`) for connectivity checks. Omits the peer list, which can be large.

**When to use:** Any time you need to check local Tailscale state, not peer routing.

**Key fields:**
```go
status, err := lc.StatusWithoutPeers(ctx)
// err != nil            → tailscaled unreachable → Installed: false
// status.BackendState   → "Running" | "NeedsLogin" | "Stopped" | "Starting" | "NoState"
// status.TailscaleIPs   → []netip.Addr — the 100.x.x.x address(es)
// status.CertDomains    → []string — non-empty only when HTTPS enabled in admin DNS settings
// status.Health         → []string — known problems; empty = healthy
```

### Pattern 3: Network-as-Auth-Perimeter (MEDIUM confidence)

**What:** Remove application-layer password auth and per-session tokens. Rely on Tailscale network membership as the security boundary. The server only listens on the Tailscale IP; public internet cannot reach it.

**When to use:** Single-user or small-team tailnets where tailnet membership is controlled. For stricter access control, Tailscale ACLs at the network layer are the right tool (not app-layer auth).

**Trade-offs:** Simpler code. Any authenticated tailnet member can reach the dashboard — if multiple people are on the same tailnet, they share access. For personal use this is fine. For team use, Tailscale ACLs restrict which devices can reach specific IPs/ports.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Calling `tailscale cert` as a Subprocess

**What people do:** `exec.Command("tailscale", "cert", domain)` to write cert files, then `tls.LoadX509KeyPair`.

**Why it's wrong:** Requires the `tailscale` CLI binary on PATH (not guaranteed). Cert files written to disk need cleanup. `local.Client.GetCertificate` is the designed Go API that `tailscale cert` calls internally.

**Do this instead:** Use `local.Client.GetCertificate` as a `tls.Config.GetCertificate` callback.

### Anti-Pattern 2: Binding to `0.0.0.0` and Filtering in Middleware

**What people do:** Bind to all interfaces, then check `r.RemoteAddr` in middleware to reject non-Tailscale connections.

**Why it's wrong:** Exposes the port on all network interfaces at the OS level. The TLS handshake (including `GetCertificate`) happens before any middleware sees the request — non-Tailscale connections still trigger cert fetching and consume a handshake.

**Do this instead:** Bind the listener to the Tailscale IP specifically. The OS rejects connections from other interfaces before they reach Go.

### Anti-Pattern 3: Hardcoding the Tailscale IP

**What people do:** Read `100.x.x.x` from a config file or environment variable.

**Why it's wrong:** Tailscale IPs are stable per-device but can change on re-key or re-install. Also breaks on machines enrolled in multiple tailnets.

**Do this instead:** Always call `lc.StatusWithoutPeers(ctx).TailscaleIPs[0]` at server start time. One call, always current.

### Anti-Pattern 4: Importing the Deprecated `tailscale.com/client/tailscale` Package

**What people do:** `import "tailscale.com/client/tailscale"` and use package-level functions like `tailscale.GetCertificate` or `tailscale.Status`.

**Why it's wrong:** These are deprecated aliases. The `pkg.go.dev` docs mark them deprecated in favor of `tailscale.com/client/local`.

**Do this instead:** Import `tailscale.com/client/local`. The `local.Client` type is the current API.

---

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| tailscaled (local daemon) | `local.Client{}` zero value — uses platform default socket | Linux/macOS: `/var/run/tailscale/tailscaled.sock`; Windows: named pipe. Platform detection is handled inside `local.Client`. |
| Let's Encrypt (via tailscaled) | Indirect — tailscaled handles ACME DNS-01 challenge. App only calls `lc.GetCertificate`. | Prerequisite: HTTPS must be enabled in Tailscale admin console DNS settings. First cert fetch may take 1–2 seconds. Cert files stored in tailscaled data dir, not `~/.config/agenthub/`. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `app.go` → `webserver.TailscaleHealth` | Direct struct return from `CheckHealth()` | Replaces the `[]webserver.NetworkInterface` that was returned to the frontend |
| `WebServer.Start()` → `local.Client` | Direct call; `local.Client` is a value type | Not an interface — wrap in a `getCertFunc` parameter for testability: `type certFunc func(*tls.ClientHelloInfo) (*tls.Certificate, error)` |
| `tls.Listener` → tailscaled | Unix socket via `local.Client` | If tailscaled stops after `Start()`, `GetCertificate` returns an error on the next handshake. Existing connections are unaffected. |

---

## go.mod Impact

Adding `tailscale.com` as a direct dependency is significant:

- `tailscale.com` is a large module with many transitive dependencies
- Only `tailscale.com/client/local` and `tailscale.com/ipn/ipnstate` are needed
- **Verify binary size delta before committing** — the Tailscale module pulls in networking/crypto packages that may substantially increase binary size from the current ~15 MB baseline

**Alternative if binary size is unacceptable:** Use `github.com/tailscale/tscert` (a minimal stripped-down package for cert-only use) for `GetCertificate`, and perform health checks via raw HTTP to the tailscaled local API socket. This avoids the full `tailscale.com` dependency tree but requires more manual HTTP work.

---

## Suggested Build Order

### Phase 1: New `tailscale.go` — Health Check (isolated, no behavioral changes)

Build `internal/webserver/tailscale.go` with `TailscaleHealth` and `CheckHealth()`. Wire `GetTailscaleStatus()` into `app.go` as a new Wails method. Write tests that mock the `local.Client` response.

**Why first:** Health check logic is used by everything else. Can be built and tested without touching existing TLS or auth code. Frontend can start showing Tailscale status immediately.

**This phase also:** Adds `tailscale.com` to `go.mod`. Measure binary size delta here before proceeding.

### Phase 2: Bind to Tailscale IP + Tailscale TLS

Modify `WebServer.Start()` to accept `tailscaleIP string` and use `local.Client.GetCertificate`. Modify `app.go:StartWebServer()` to call `CheckHealth()` first and pass the IP.

**Why second:** The TLS change is the load-bearing shift. Old `tls.go` is not deleted yet — kept until integration tests confirm `GetCertificate` returns a valid cert for the machine's `.ts.net` domain.

**Risk:** First real test against live tailscaled. Needs manual verification on the dev machine that cert provisioning works end-to-end.

### Phase 3: Remove Auth Layer

Remove `dashboardAuth` and `sessionAuth` middleware. Remove `AuthManager`, `TokenStore`. Remove `POST /login`, `GET /ca.crt`, `POST /api/sessions/{id}/token` routes. Remove corresponding Wails methods from `app.go` (`SetWebPassword`, `IsWebPasswordSet`, `GenerateSessionToken`, `GetCACertPath`). Remove password persistence (`web_password` file, `webPasswordPath()`, `LoadPasswordHash` calls).

**Why third:** Only safe to remove auth after confirming the server works with Tailscale TLS on the correct IP (Phase 2). Removing auth while still on self-signed certs during transition would expose the server unguarded.

### Phase 4: Delete Dead Files

Delete `tls.go`, `tls_test.go`, `auth.go`, `auth_test.go`, `tokens.go`, `tokens_test.go`. Simplify `network.go` to only `GetTailscaleIP`. Simplify `Config` struct. Remove `GetNetworkInterfaces()` from `app.go`.

**Why last:** Deleting files after the code compiles without them means the compiler enforces that all callsites are updated. Delete only when `go build ./...` passes.

### Phase 5: Frontend Health Modal

Wire `TailscaleHealth` struct to a React modal in Settings that shows three distinct instructional states. Can begin in parallel with Phase 3–4 once `GetTailscaleStatus()` is available from Phase 1.

---

## Sources

- [Tailscale enabling HTTPS docs](https://tailscale.com/docs/how-to/set-up-https-certificates)
- [tailscale.com/client/local pkg.go.dev](https://pkg.go.dev/tailscale.com/client/local) — `Client`, `GetCertificate`, `CertPair`, `StatusWithoutPeers` — confirmed stable API
- [tailscale.com/ipn/ipnstate pkg.go.dev](https://pkg.go.dev/tailscale.com/ipn/ipnstate) — `Status` struct: `BackendState`, `TailscaleIPs`, `CertDomains`, `Health`
- [Official servetls example](https://github.com/tailscale/tailscale/blob/main/client/tailscale/example/servetls/servetls.go) — canonical `GetCertificate` usage: `TLSConfig: &tls.Config{GetCertificate: lc.GetCertificate}`
- [github.com/tailscale/tscert](https://pkg.go.dev/github.com/tailscale/tscert) — minimal cert-only alternative if full `tailscale.com` dependency is too heavy
- [Tailscale TLS blog post](https://tailscale.com/blog/tls-certs) — context on the DNS-01 ACME cert provisioning model
- Direct codebase inspection: `internal/webserver/server.go`, `tls.go`, `auth.go`, `tokens.go`, `network.go`, `app.go`, `go.mod`

---

*Architecture research for: AgentHub v1.2 Tailscale-only networking*
*Researched: 2026-03-20*
