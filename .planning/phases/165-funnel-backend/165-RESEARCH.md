# Phase 165: Funnel Backend — Research

**Researched:** 2026-06-30
**Domain:** Go daemon (Tailscale LocalClient Funnel lifecycle) + webserver Origin/BaseURL + capability URL builders
**Confidence:** HIGH (all file paths, line numbers, and signatures verified by direct codebase inspection; Tailscale API verified against pinned v1.98.3 source on disk; project research pre-answers all design decisions)

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FNL-01 | Session owner can enable Tailscale Funnel on a shared session to expose it to the public internet; Funnel is off by default | `handleSetSessionFunnel` daemon endpoint + `funnelSessions` map; default=off because the map starts empty |
| FNL-02 | Enabling Funnel uses the embedded Tailscale LocalClient (`SetServeConfig`/`AllowFunnel`); no admin API token required | `ws.lc.SetServeConfig` on the promoted `local.Client` struct field |
| FNL-03 | A recipient not on the tailnet and with no Tailscale account can join via the Funnel URL + join code | Funnel-aware `issueCapabilitiesForSession` + `handleExchangeJoinCode` emit `https://hostname.ts.net/…` URLs (no port); existing join-code flow is unchanged |
| FNL-04 | When Funnel is active, Origin allowlist, `BaseURL()`, and share URLs use the Funnel hostname so cap-token exchange succeeds without a 403 | `FunnelBaseURL()` + dual-origin `requireAllowedOrigin`/`allowedOrigins`/`originAllowedForWrite` + conditional base in URL builders |
| FNL-05 | Funnel exposure is fully torn down on four triggers: user disables Funnel, user disables web-share, session ends naturally, daemon stops | Four teardown sites: `handleSetSessionFunnel(false)`, `handleWebServe(false)`, `runSessionExitCleanup`, `WebServer.Stop()` |
| FNL-06 | When tailnet prerequisites not met, `EnableFunnel` returns human-readable error matching `ipn.CheckFunnelAccess` output; never calls `SetServeConfig` | Pre-flight `ipn.CheckFunnelAccess(port, st.Self)` call; error surfaced verbatim |
| FNL-07 | A Funnel share auto-expires after user-chosen duration; daemon tears down Funnel at expiry server-side | `time.AfterFunc` timer per session in `funnelExpiry map[string]*time.Timer` on the API struct; **see planning decision flag below** |
</phase_requirements>

---

## Summary

Phase 165 adds the full Tailscale Funnel lifecycle to the AgentHub daemon. Zero new Go dependencies are required: every Funnel API (`SetServeConfig`, `GetServeConfig`, `CheckFunnelAccess`, `SetFunnel`, `IsFunnelOn`) is present in the already-pinned `tailscale.com v1.98.3`. The single structural change is promoting `local.Client` from a stack-local variable inside `startTailscale()` to a `ws.lc local.Client` struct field on `WebServer` — `local.Client` is zero-value usable so this is a zero-cost refactor.

The architectural rule for this phase is atomicity: `EnableFunnel`, `DisableFunnel`, `FunnelBaseURL`, the dual-origin Origin allowlist update, and the Funnel-URL-emitting share-link builders must all ship together. A partial deployment where `EnableFunnel()` works but the Origin check is not updated causes every Funnel guest to 403 before the capability token is ever inspected — the symptom resembles a token bug but is actually an Origin header mismatch (Funnel arrives at port 443 with `Origin: https://hostname.ts.net`; `BaseURL()` returns `https://hostname.ts.net:7443`).

All four teardown paths must be wired and tested: user disables Funnel toggle, user disables web-share, session exits naturally, and daemon stops cleanly. These are the phase's primary security properties. Additionally, FNL-07 (auto-expiry) was not included in the project research's Phase 165 deliverables list but is a Phase 165 requirement — see the planning decision flag in the Open Questions section.

**Primary recommendation:** Implement all six Funnel components (LocalClient promotion, EnableFunnel/DisableFunnel/FunnelBaseURL, dual-origin allowlist, share-URL builders, daemon endpoint + funnelSessions map, App bound method) in a single atomic commit sequence. Wire all four teardown sites. Add `funnelExpiry map[string]*time.Timer` for FNL-07.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Funnel enable/disable (SetServeConfig) | API/Backend (daemon webserver package) | — | Tailscale LocalClient must run in the daemon process; Wails frontend has no direct access to tailscaled socket |
| Origin allowlist dual-check | API/Backend (webserver middleware) | — | Security middleware runs in the webserver request path; must be atomic with Funnel enable |
| Share URL construction (Funnel-aware) | API/Backend (daemon api.go) | — | URL builders call `ws.BaseURL()` / `ws.FunnelBaseURL()` which live in the webserver package |
| Funnel state tracking (funnelSessions) | API/Backend (daemon api.go) | — | Daemon owns session lifecycle; cross-session reference counting lives next to the web-share toggle |
| Auto-expiry timer | API/Backend (daemon api.go) | — | Must be server-side, independent of UI connection |
| Wails bound method (SetSessionFunnel) | Frontend Server (app.go) | API/Backend | Thin bridge; no logic; calls daemon client method |
| Fallback-mode guard | API/Backend (webserver + daemon) | — | Must prevent `funnelActive` from being set when `StatusWithoutPeers` fails |

---

## Standard Stack

### Core (Phase 165 only — no new packages)

| Package | Version | Purpose | Status |
|---------|---------|---------|--------|
| `tailscale.com/client/local` | v1.98.3 (pinned) | `local.Client`: `SetServeConfig`, `GetServeConfig`, `StatusWithoutPeers` | Already in go.mod; `local.Client` zero-value usable |
| `tailscale.com/ipn` | v1.98.3 (pinned) | `ServeConfig` struct, `SetFunnel`, `IsFunnelOn`, `CheckFunnelAccess`, `NodeCanFunnel`, `CheckFunnelPort` | Already in go.mod |
| `tailscale.com/tailcfg` | v1.98.3 (pinned) | `NodeAttrFunnel`, `CapabilityHTTPS`, `CapabilityFunnelPorts` constants | Already in go.mod |

**No new `go get` required for Phase 165.** (`github.com/gen2brain/beeep` is Phase 167, not Phase 165.)

**Package legitimacy:** No new packages — legitimacy audit not needed for Phase 165.

### Confirmed Tailscale API Signatures (v1.98.3, [VERIFIED: /Users/ken/go/pkg/mod/tailscale.com@v1.98.3/ipn/serve.go])

```go
// GetServeConfig — returns nil, nil when not configured; ETag field populated
func (lc *Client) GetServeConfig(ctx context.Context) (*ipn.ServeConfig, error)

// SetServeConfig — nil config clears all serve settings; ETag on sc is sent as If-Match
func (lc *Client) SetServeConfig(ctx context.Context, config *ipn.ServeConfig) error

// StatusWithoutPeers — use for Funnel prereq check (faster than Status)
func (lc *Client) StatusWithoutPeers(ctx context.Context) (*ipnstate.Status, error)

// CheckFunnelAccess — checks HTTPS cap + funnel nodeAttr + port policy; returns human-readable errors
func CheckFunnelAccess(port uint16, node *ipnstate.PeerStatus) error

// SetFunnel — toggle AllowFunnel for hostname:port; nils out empty map
func (sc *ServeConfig) SetFunnel(host string, port uint16, setOn bool)

// IsFunnelOn — true if any AllowFunnel entry is set
func (sc *ServeConfig) IsFunnelOn() bool
```

**Verified error strings from `CheckFunnelAccess` (load-bearing — surface verbatim):** [VERIFIED: tailscale.com@v1.98.3/ipn/serve.go:610-621]
- `"Funnel not available; HTTPS must be enabled. See https://tailscale.com/s/https."`
- `"Funnel not available; \"funnel\" node attribute not set. See https://tailscale.com/s/no-funnel."`
- `"port N is not allowed for funnel; allowed ports are: 443,8443,10000"` (port list is policy-controlled)

---

## Architecture Patterns

### System Architecture Diagram

```
User Action (Phase 166 toggle)
  ↓
App.SetSessionFunnel(sessionID, true, expiresIn)    [app.go — NEW Wails bound method]
  ↓ IPC (Unix socket / named pipe)
daemon/api.go: POST /sessions/{id}/funnel           [NEW endpoint]
  → handleSetSessionFunnel()
  ↓ ipn.CheckFunnelAccess(443, st.Self)             prereq check — stop here if error
  ↓ ws.EnableFunnel(ctx, 443)
      ↓ lc.GetServeConfig(ctx)                      read-modify-write with ETag
      ↓ modify struct: TCP[443]+Web[hp]+SetFunnel()
      ↓ lc.SetServeConfig(ctx, sc)
      ↓ cache funnelBaseURL = "https://"+hostname
  ↓ funnelSessions[sessionID] = true
  ↓ funnelExpiry[sessionID] = time.AfterFunc(expiresIn, teardown)
  ↓ 200 OK → frontend re-calls POST /sessions/{id}/capabilities
      ↓ issueCapabilitiesForSession
          ↓ base = ws.FunnelBaseURL() when funnelSessions[id]   [MODIFIED URL builder]
          ↓ → "https://hostname.ts.net/sessions/id?cap=TOKEN"   (no port)

External viewer (internet, not on tailnet):
  https://hostname.ts.net/sessions/id?cap=TOKEN
    ↓ Tailscale Funnel: public port 443 → local :7443 TCP proxy
    ↓ requireAllowedOrigin                           [MODIFIED: dual check]
        origin "https://hostname.ts.net" == ws.FunnelBaseURL() → allowed
    ↓ requireCapability: token valid → 302 /app/?session=id&cap=TOKEN
    ↓ handleWSSRelay: session streams to browser

Teardown (any of 4 triggers):
  → disableFunnelForSession(id)
      ↓ funnelExpiry[id].Stop() + delete
      ↓ delete funnelSessions[id]
      ↓ if len(funnelSessions) == 0: ws.DisableFunnel(ctx)
          ↓ lc.GetServeConfig → SetFunnel(false) + delete TCP/Web entries → lc.SetServeConfig(nil/empty)
```

### Recommended Project Structure Changes

No new directories or packages. Changes are additive to existing files:

```
internal/webserver/
├── server.go          MODIFIED — promote lc, add funnelActive/funnelBaseURL fields + 3 new methods
├── origin_mw.go       MODIFIED — dual-origin requireAllowedOrigin + allowedOrigins
└── capability_mw.go   MODIFIED — dual-origin originAllowedForWrite

internal/daemon/
└── api.go             MODIFIED — API struct funnelSessions+funnelExpiry, handleSetSessionFunnel,
                                  handleWebServe teardown, runSessionExitCleanup teardown,
                                  handleWebServerStop teardown, URL builder Funnel-awareness

app.go                 MODIFIED — SetSessionFunnel Wails bound method
```

### Pattern 1: LocalClient Promotion (no-cost struct field)

`startTailscale()` currently creates a stack-local `var lc local.Client` at line 406. Promote to struct field: [VERIFIED: internal/webserver/server.go:406-417]

```go
// Add to WebServer struct (guarded by ws.mu where needed):
type WebServer struct {
    config  Config
    manager *relay.HubManager
    lc      local.Client   // NEW — promoted from startTailscale() stack local; zero-value usable

    // Funnel state — guarded by ws.mu:
    funnelActive   bool
    funnelBaseURL  string // "https://<hostname>" (no port) when active
    funnelPort     uint16

    mu         sync.RWMutex
    // ... existing fields unchanged
}

// In startTailscale(), change:
//   var lc local.Client
//   tlsCfg = &tls.Config{GetCertificate: lc.GetCertificate, ...}
// to:
//   tlsCfg = &tls.Config{GetCertificate: ws.lc.GetCertificate, ...}
```

Note: `handleWSSRelay()` creates its own `var lc local.Client` at line 1121 for WhoIs calls — leave that unchanged.

### Pattern 2: EnableFunnel / DisableFunnel / FunnelBaseURL

```go
// Source: verified against tailscale.com@v1.98.3/ipn/serve.go
func (ws *WebServer) EnableFunnel(ctx context.Context, funnelPort uint16) error {
    ws.mu.Lock()
    defer ws.mu.Unlock()

    st, err := ws.lc.StatusWithoutPeers(ctx)
    if err != nil {
        return fmt.Errorf("funnel: tailscale status: %w", err)
    }

    // Step 1: prerequisite check — surface human-readable errors verbatim
    if err := ipn.CheckFunnelAccess(funnelPort, st.Self); err != nil {
        return err // e.g. "Funnel not available; HTTPS must be enabled..."
    }

    hostname := strings.TrimSuffix(st.Self.DNSName, ".")

    // Step 2: read-modify-write (preserves ETag for optimistic concurrency)
    sc, err := ws.lc.GetServeConfig(ctx)
    if err != nil {
        return err
    }
    if sc == nil {
        sc = new(ipn.ServeConfig)
    }

    // Step 3: TCP handler — Tailscale daemon terminates HTTPS on funnelPort
    if sc.TCP == nil {
        sc.TCP = make(map[uint16]*ipn.TCPPortHandler)
    }
    sc.TCP[funnelPort] = &ipn.TCPPortHandler{HTTPS: true}

    // Step 4: Web handler — proxy to AgentHub's local server
    _, localPort, _ := net.SplitHostPort(ws.Addr())
    hp := ipn.HostPort(net.JoinHostPort(hostname, strconv.Itoa(int(funnelPort))))
    if sc.Web == nil {
        sc.Web = make(map[ipn.HostPort]*ipn.WebServerConfig)
    }
    sc.Web[hp] = &ipn.WebServerConfig{
        Handlers: map[string]*ipn.HTTPHandler{
            "/": {Proxy: "https://localhost:" + localPort},
        },
    }

    // Step 5: enable Funnel for this host:port
    sc.SetFunnel(hostname, funnelPort, true)

    // Step 6: apply (ETag carried on sc from GetServeConfig)
    if err := ws.lc.SetServeConfig(ctx, sc); err != nil {
        return err
    }

    // Cache Funnel state
    ws.funnelActive = true
    ws.funnelPort = funnelPort
    if funnelPort == 443 {
        ws.funnelBaseURL = "https://" + hostname // no port — 443 is default HTTPS
    } else {
        ws.funnelBaseURL = fmt.Sprintf("https://%s:%d", hostname, funnelPort)
    }
    return nil
}

func (ws *WebServer) DisableFunnel(ctx context.Context) error {
    ws.mu.Lock()
    defer ws.mu.Unlock()

    sc, err := ws.lc.GetServeConfig(ctx)
    if err != nil || sc == nil {
        ws.funnelActive = false
        ws.funnelBaseURL = ""
        return err
    }

    hostname := strings.TrimSuffix(strings.TrimPrefix(ws.funnelBaseURL, "https://"), fmt.Sprintf(":%d", ws.funnelPort))
    hp := ipn.HostPort(net.JoinHostPort(hostname, strconv.Itoa(int(ws.funnelPort))))

    sc.SetFunnel(hostname, ws.funnelPort, false)
    delete(sc.TCP, ws.funnelPort)
    delete(sc.Web, hp)
    if len(sc.TCP) == 0 { sc.TCP = nil }
    if len(sc.Web) == 0 { sc.Web = nil }

    if err := ws.lc.SetServeConfig(ctx, sc); err != nil {
        return err
    }
    ws.funnelActive = false
    ws.funnelBaseURL = ""
    return nil
}

func (ws *WebServer) FunnelBaseURL() string {
    ws.mu.RLock()
    defer ws.mu.RUnlock()
    return ws.funnelBaseURL // "" when inactive; "https://hostname.ts.net" when active on port 443
}
```

### Pattern 3: Dual-Origin Allowlist

**MODIFY `internal/webserver/origin_mw.go`** — the existing code [VERIFIED: origin_mw.go:31-66] does a single byte-for-byte match against `ws.BaseURL()`. Add Funnel origin check:

```go
// requireAllowedOrigin — MODIFIED for dual origin (Funnel-aware)
func (ws *WebServer) requireAllowedOrigin(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if origin == "" {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }
        tailnetURL := ws.BaseURL()
        if tailnetURL == "" {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }
        if origin == tailnetURL {
            next(w, r)
            return
        }
        // Funnel: check secondary origin (empty string when Funnel not active)
        if funnelURL := ws.FunnelBaseURL(); funnelURL != "" && origin == funnelURL {
            next(w, r)
            return
        }
        http.Error(w, "forbidden", http.StatusForbidden)
    }
}

// allowedOrigins — MODIFIED to return both origins when Funnel active
func (ws *WebServer) allowedOrigins() []string {
    base := ws.BaseURL()
    if base == "" {
        return nil
    }
    origins := []string{base}
    if funnelBase := ws.FunnelBaseURL(); funnelBase != "" {
        origins = append(origins, funnelBase)
    }
    return origins
}
```

**MODIFY `internal/webserver/capability_mw.go`** — `originAllowedForWrite` [VERIFIED: capability_mw.go:187-198] checks only `ws.BaseURL()`:

```go
func (ws *WebServer) originAllowedForWrite(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    if origin == "" {
        return true // desktop Wails fetch — pass vacuously
    }
    allowed := ws.BaseURL()
    if allowed != "" && origin == allowed {
        return true
    }
    // Funnel origin (internet guests sending write requests)
    if funnelURL := ws.FunnelBaseURL(); funnelURL != "" && origin == funnelURL {
        return true
    }
    return false
}
```

### Pattern 4: Funnel-Aware Share URL Builders

**MODIFY `internal/daemon/api.go`** — two URL builder sites:

Site 1: `issueCapabilitiesForSession` [VERIFIED: api.go:1287-1289]
```go
// Current:
base := ws.BaseURL()
readURL = base + "/sessions/" + sessionID + "?cap=" + rTok
writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok

// Replace with:
base := ws.BaseURL()
a.mu.RLock()
isFunnelSession := a.funnelSessions[sessionID]
a.mu.RUnlock()
if isFunnelSession {
    if fb := ws.FunnelBaseURL(); fb != "" {
        base = fb
    }
}
readURL = base + "/sessions/" + sessionID + "?cap=" + rTok
writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok
```

Site 2: `handleExchangeJoinCode` [VERIFIED: api.go:1385]
```go
// Current:
url := ws.BaseURL() + "/sessions/" + claims.SID + "?cap=" + token

// Replace with:
base := ws.BaseURL()
a.mu.RLock()
isFunnelSession := a.funnelSessions[claims.SID]
a.mu.RUnlock()
if isFunnelSession {
    if fb := ws.FunnelBaseURL(); fb != "" {
        base = fb
    }
}
url := base + "/sessions/" + claims.SID + "?cap=" + token
```

### Pattern 5: Four Teardown Sites

All four sites must call a shared `disableFunnelForSession` helper:

```go
// Add to API struct:
type API struct {
    // ... existing fields ...
    funnelSessions map[string]bool         // guarded by a.mu
    funnelExpiry   map[string]*time.Timer  // guarded by a.mu (FNL-07)
}

// Shared teardown helper — call under a.mu.Lock() or with appropriate locking:
func (a *API) disableFunnelForSession(ctx context.Context, sessionID string) {
    a.mu.Lock()
    // Cancel any expiry timer
    if t, ok := a.funnelExpiry[sessionID]; ok {
        t.Stop()
        delete(a.funnelExpiry, sessionID)
    }
    delete(a.funnelSessions, sessionID)
    remaining := len(a.funnelSessions)
    ws := a.webServer
    a.mu.Unlock()

    if ws != nil && remaining == 0 {
        _ = ws.DisableFunnel(ctx)
    }
}

// Teardown Site 1: handleSetSessionFunnel(id, false)
// Teardown Site 2: handleWebServe(id, false) — add after existing ws.DisableSession(id)
//   a.disableFunnelForSession(r.Context(), id)
// Teardown Site 3: runSessionExitCleanup — add after existing ws.DisableSession(sessionID)
//   a.disableFunnelForSession(context.Background(), sessionID)
// Teardown Site 4: WebServer.Stop() or handleWebServerStop — add DisableFunnel call
//   before ws.Stop(), call ws.DisableFunnel(ctx) to clear Tailscale serve config
```

### Pattern 6: New Daemon Endpoint

```go
// Add to registerRoutes():
a.mux.HandleFunc("POST /sessions/{id}/funnel", a.handleSetSessionFunnel)

// Add to types.go:
type SetSessionFunnelRequest struct {
    Enabled   bool `json:"enabled"`
    ExpiresIn int  `json:"expiresIn"` // seconds; 0 = no expiry (FNL-07)
}

type SetSessionFunnelResponse struct {
    FunnelURL string `json:"funnelUrl"` // e.g. "https://hostname.ts.net"
}

// Handler:
func (a *API) handleSetSessionFunnel(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    var req SetSessionFunnelRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    a.mu.RLock()
    ws := a.webServer
    a.mu.RUnlock()
    if ws == nil {
        http.Error(w, "web server not running", http.StatusBadRequest)
        return
    }

    if !req.Enabled {
        a.disableFunnelForSession(r.Context(), id)
        w.WriteHeader(http.StatusNoContent)
        return
    }

    // Enable: prereq check happens inside ws.EnableFunnel
    if err := ws.EnableFunnel(r.Context(), 443); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest) // CheckFunnelAccess errors are user-facing
        return
    }

    a.mu.Lock()
    if a.funnelSessions == nil {
        a.funnelSessions = make(map[string]bool)
    }
    a.funnelSessions[id] = true
    // FNL-07: auto-expiry timer
    if req.ExpiresIn > 0 {
        if a.funnelExpiry == nil {
            a.funnelExpiry = make(map[string]*time.Timer)
        }
        if t, ok := a.funnelExpiry[id]; ok {
            t.Stop()
        }
        dur := time.Duration(req.ExpiresIn) * time.Second
        a.funnelExpiry[id] = time.AfterFunc(dur, func() {
            a.disableFunnelForSession(context.Background(), id)
        })
    }
    a.mu.Unlock()

    writeJSON(w, http.StatusOK, SetSessionFunnelResponse{FunnelURL: ws.FunnelBaseURL()})
}
```

### Pattern 7: Wails Bound Method

```go
// Add to app.go:
// SetSessionFunnel enables or disables Tailscale Funnel for a session.
// expiresIn is the expiry duration in seconds (0 = no auto-expiry, FNL-07).
func (a *App) SetSessionFunnel(sessionID string, enabled bool, expiresIn int) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.SetSessionFunnel(sessionID, enabled, expiresIn)
}
```

### Anti-Patterns to Avoid

- **Do NOT modify `BaseURL()` to return the Funnel URL when active.** `BaseURL()` is used by the local Wails display, tray icon, QR code generator, and Settings copy button — all of which should continue showing the tailnet URL. Only the Origin allowlist and share-URL builders need Funnel-awareness. Keep `BaseURL()` unchanged; add `FunnelBaseURL()` separately. [VERIFIED: ARCHITECTURE.md Anti-Pattern 1]
- **Do NOT store `local.Client` by pointer.** Store by value: `lc local.Client`. The existing pattern in `tailscale.go` line 97-98 and `startTailscale()` line 412 confirms zero-value usability. [VERIFIED: server.go:412, tailscale.go:97]
- **Do NOT call `DisableFunnel()` on every web-serve toggle without checking the reference count.** If two sessions have Funnel active and session B disables web-share, calling `DisableFunnel()` would tear down Funnel for session A too. Only call `DisableFunnel()` when `len(funnelSessions) == 0`. [VERIFIED: ARCHITECTURE.md Anti-Pattern 3]
- **Do NOT construct a `&ipn.ServeConfig{}` literal in `EnableFunnel`.** Always call `GetServeConfig` first and modify the returned struct to preserve the ETag for optimistic concurrency. [VERIFIED: tailscale.com@v1.98.3/ipn/serve.go ETag field tagged `json:"-"`]
- **Do NOT call `CheckFunnelAccess` inside the `ws.mu.Lock()` section.** The StatusWithoutPeers call goes to the tailscaled daemon over a Unix socket and can block. Call it before acquiring the lock, or restructure the check to be outside the lock.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tailscale serve config | Custom TCP proxy / ngrok tunnel | `lc.SetServeConfig` with `AllowFunnel` | Tailscale handles TLS, cert provisioning, and public routing |
| Funnel prerequisite check | Parse nodeAttr strings manually | `ipn.CheckFunnelAccess(port, st.Self)` | Already checks HTTPS cap + funnel nodeAttr + port policy; returns exact human-readable error strings |
| Funnel hostname resolution | Parse FQDN from tailscale binary output | `st.Self.DNSName` (trailing-dot trimmed) or `st.CertDomains[0]` from `StatusWithoutPeers` | Already available in existing status API |
| ETag concurrency control | Retry loop or local mutex-only | Preserve ETag from `GetServeConfig` and send via `SetServeConfig` | Tailscale daemon uses ETag to detect concurrent modification |
| Origin allowlist logic | Custom header parsing | Extend existing `requireAllowedOrigin` / `allowedOrigins` / `originAllowedForWrite` | All three sites are already in the codebase and have existing tests |

---

## Runtime State Inventory

Not applicable — Phase 165 is not a rename/refactor phase. Funnel state is ephemeral (in-memory `funnelSessions` map); no data migration is needed.

---

## Common Pitfalls

### Pitfall 1: Funnel Teardown Incomplete on Non-Happy-Path Exits

**What goes wrong:** `SetServeConfig` with `AllowFunnel` registers a node-level Tailscale serve config that persists in the Tailscale daemon independent of the AgentHub process. If any of the four teardown sites is missed, the Funnel remains active after the session is gone — the public Tailscale Funnel port remains open indefinitely.

**Verified teardown sites and current state:** [VERIFIED: api.go:654-662, api.go:1198-1205, api.go:1128-1138, server.go:483-491]
- Site 1 — `handleSetSessionFunnel(id, false)`: NEW (doesn't exist yet)
- Site 2 — `handleWebServe(id, false)`: line 1203 calls `ws.DisableSession(id)` + `ws.ClearGrants(id)` — **missing Funnel teardown**
- Site 3 — `runSessionExitCleanup`: line 658-661 calls `ws.DisableSession` + `ws.ClearGrants` — **missing Funnel teardown**
- Site 4 — `WebServer.Stop()`: line 483-491 just closes listener — **missing Funnel teardown**

**How to avoid:** Use the `disableFunnelForSession` helper at all four sites. Verify with `tailscale serve status` in tests.

### Pitfall 2: Origin/BaseURL 403 Before Auth (Integration Landmine)

**What goes wrong:** Funnel guests arrive at port 443. Browser sends `Origin: https://hostname.ts.net` (no port). `requireAllowedOrigin` [VERIFIED: origin_mw.go:41-48] does a byte-for-byte match against `ws.BaseURL()` which returns `https://hostname.ts.net:7443`. Every Funnel guest gets 403 before the cap token is ever checked.

**How to avoid:** `FunnelBaseURL()` + dual-origin check must ship in the SAME COMMIT as `EnableFunnel()`. Test from a machine OUTSIDE the tailnet (not from a tailnet device — testing from tailnet hides this bug because the tailnet URL also works).

### Pitfall 3: ETag Concurrency Clobber on ServeConfig Read-Modify-Write

**What goes wrong:** If `EnableFunnel` constructs a new `&ipn.ServeConfig{}` instead of calling `GetServeConfig` first, the ETag is lost. The `ETag` field is tagged `json:"-"` so it's invisible in logs. Concurrent modification triggers a 412 error. [VERIFIED: tailscale.com@v1.98.3/ipn/serve.go:66 — ETag tagged json:"-"]

**How to avoid:** Always `GetServeConfig` immediately before `SetServeConfig`. Never construct a literal `ServeConfig`. Mutex-guard `EnableFunnel`/`DisableFunnel`.

### Pitfall 4: No Prerequisite Check Before SetServeConfig

**What goes wrong:** `SetServeConfig` with `AllowFunnel` fails with an opaque error when tailnet prerequisites are not met. The user sees a generic error with no actionable guidance.

**How to avoid:** Call `ipn.CheckFunnelAccess(port, st.Self)` before `SetServeConfig`. Surface the exact error string verbatim. The check happens inside `EnableFunnel` so callers automatically get the human-readable error.

### Pitfall 5: Tests Encoding the Same Wrong Assumption

**What goes wrong:** A test verifies the dual-origin check by setting `ws.funnelActive = true` directly (not via `ws.EnableFunnel()`), bypassing the actual wiring. CI passes; the production path is broken. This is the Phase 150 shell-warning pattern that bit this project. [VERIFIED: MEMORY.md — "Tests can encode the same wrong assumption"]

**How to avoid:** Tests for the Origin allowlist must call `ws.EnableFunnel()` (or a test double), not set struct fields directly. Teardown tests must verify `GetServeConfig` returns empty/nil, not assert a mock was called.

### Pitfall 6: Local-Network-Fallback Path Broken by Funnel Changes

**What goes wrong:** When Tailscale is not running, `ws.lc.StatusWithoutPeers(ctx)` returns an error. If `EnableFunnel` propagates this in a way that corrupts `ws.funnelActive`, `requireAllowedOrigin` will look for an origin that can never arrive and block all web-share in fallback mode.

**How to avoid:** `EnableFunnel` must return an explicit user-surfaced error when `StatusWithoutPeers` fails, and must NOT modify `ws.funnelActive`. `ws.funnelActive` is only set to `true` inside `EnableFunnel` after a successful `SetServeConfig`. Include a CI test with a mock `local.Client` that returns an error from `StatusWithoutPeers`.

---

## FNL-07 Planning Decision Flag

**FNL-07 was not in the project research's Phase 165 deliverables list** (see SUMMARY.md "Phase 165: Funnel Backend (Atomic)" — FNL-07 is not mentioned). However, it IS a Phase 165 requirement per REQUIREMENTS.md and ROADMAP.md.

**The requirement:** "A Funnel share automatically expires after a user-chosen duration; the daemon tears the Funnel exposure down at expiry (enforced server-side, independent of any connected UI)."

**Recommended implementation for FNL-07:**

```go
// Add to API struct:
funnelExpiry map[string]*time.Timer  // per-session, guarded by a.mu

// In handleSetSessionFunnel (enable path):
if req.ExpiresIn > 0 {
    dur := time.Duration(req.ExpiresIn) * time.Second
    a.funnelExpiry[id] = time.AfterFunc(dur, func() {
        a.disableFunnelForSession(context.Background(), id)
    })
}

// In disableFunnelForSession:
if t, ok := a.funnelExpiry[id]; ok {
    t.Stop()
    delete(a.funnelExpiry, id)
}
```

**FNL-07 planning decisions the planner must resolve:**

1. **Frontend notification of expiry:** When the timer fires, the daemon tears down Funnel silently. The frontend won't know until the next capabilities refresh (which returns a non-Funnel URL). Options:
   - A) **Polling** (recommended): Frontend re-fetches capabilities on a periodic heartbeat; stale Funnel URL becomes a tailnet URL — user notices Funnel indicator disappears. Simple, no new IPC.
   - B) **Wails event**: `runtime.EventsEmit(a.ctx, "funnel:expired", ...)` from `disableFunnelForSession` — requires app.go callback from daemon. Complex, fragile if frontend is detached.

2. **`expiresIn=0` semantics:** Planner should decide: "no expiry" (Funnel stays until explicit disable) vs. "default expiry" (e.g., 4 hours if user chose no expiry). The STATE.md says "daemon-enforced auto-expiry" — implication is the user always chooses a duration in the FUI risk dialog (FUI-02), so `expiresIn=0` may be an invalid state in practice.

3. **Daemon restart behavior:** On daemon restart, `funnelExpiry` is cleared (in-memory only). Funnel serve config persists in Tailscale daemon. Recommend: on daemon startup, call `lc.GetServeConfig` and if `AllowFunnel` is set, clear it (or accept the existing serve config as valid and re-populate `funnelSessions` from Tailscale state). **The simpler approach is to clear on startup** — analogous to the existing grant behavior where grants don't survive daemon restart.

**Include FNL-07 in Phase 165.** The `funnelExpiry` map adds minimal complexity and the alternative (deferring to Phase 166) would leave a requirement unimplemented with no natural home.

---

## Testability Seam for LocalClient

`local.Client` is a concrete struct, not an interface. For CI tests that run without a live tailscaled daemon, the options are:

**Recommended: Interface injection** [ASSUMED — standard Go testability pattern]

Define a narrow interface covering only the methods needed by `EnableFunnel`/`DisableFunnel`:

```go
// In internal/webserver/ (new file: funnel_client.go):
type funnelClient interface {
    GetServeConfig(ctx context.Context) (*ipn.ServeConfig, error)
    SetServeConfig(ctx context.Context, config *ipn.ServeConfig) error
    StatusWithoutPeers(ctx context.Context) (*ipnstate.Status, error)
}
```

Store it alongside `lc`:

```go
type WebServer struct {
    lc           local.Client  // used for GetCertificate (production)
    funnelClient funnelClient  // used for EnableFunnel/DisableFunnel (injectable for tests)
    // ...
}

// NewWebServer sets ws.funnelClient = &ws.lc (concrete uses the struct field by address)
// Tests inject a mock implementation that returns controlled responses
```

**Alternative: httptest.NewServer stub** — The `local.Client` talks to tailscaled over a Unix socket using HTTP. Tests could set `local.Client.TSMP` to an `httptest.Server`. This is more accurate but requires understanding the tailscaled HTTP API internals. Not recommended for Phase 165.

**Alternative: Build tags** — `//go:build tailscaletesting` with a fake implementation. Adds build complexity. Not recommended.

The interface approach matches the existing testability pattern in `tailscale.go` where `statusFunc` and `prefsFunc` are injected for `checkHealth`. [VERIFIED: tailscale.go:28-34]

---

## Code Drift from Project Research

The following discrepancies between the pre-existing project research and actual codebase were found during direct code inspection:

### Drift 1: Subscriber Buffer Location (Phase 168 impact)

**Research claimed:** `internal/relay/hub.go` contains the subscriber buffer 256 — change to 1024 there.

**Actual code:** [VERIFIED: internal/webserver/server.go:1085]
```go
sub := &relay.Subscriber{
    Msgs:     make(chan []byte, 256),  // buffer is HERE, in server.go
```
The `Subscriber` struct is defined in `hub.go` but the channel is allocated when the subscriber is created in `server.go:handleWSSRelay`. For Phase 168's #117 fix, the buffer increase must be in `server.go:1085`, not `hub.go`.

### Drift 2: macOS Notification Attribution (Phase 167 impact)

**Research claimed (STACK.md/PITFALLS.md):** macOS notifications via beeep show "Script Editor" attribution. This is documented as Pitfall 8.

**Actual code:** [VERIFIED: notification_darwin.go:1-25]
The existing `notification_darwin.go` uses `//go:build darwin` + CGO + `UNUserNotificationCenter` — NOT beeep/osascript. macOS already shows "AgentHub" as the notification sender. Pitfall 8 ("Script Editor" attribution) does NOT apply to macOS.

**Implication for Phase 167:** The macOS notification file does NOT need to change. Phase 167 needs:
- New `notification_windows.go` (`//go:build windows`) — beeep or go-toast/toast
- New `notification_linux.go` (`//go:build linux`) — beeep or notify-send exec
- Update `notification_other.go` build tag from `//go:build !darwin` to `//go:build !darwin && !windows && !linux`

### Drift 3: `handleWebServerStop` is a separate teardown site

**Research listed 3 "Three Teardown Sites" (ARCHITECTURE.md)** but the daemon has TWO paths for stopping the webserver:
- `handleWebServerStop` (POST /webserver/stop) — called when user stops web sharing via UI
- `api.Stop()` — called on daemon shutdown

Both paths call `ws.Stop()`. The Funnel teardown must be called before `ws.Stop()` in BOTH paths (or inside `ws.Stop()` itself). The `handleWebServerStop` path is effectively "Teardown Site 4a" and `api.Stop()` is "Teardown Site 4b" — both need coverage. Adding `DisableFunnel()` to `WebServer.Stop()` before closing the listener handles both simultaneously.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` + `httptest` |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./internal/webserver/... ./internal/daemon/... -run Funnel -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FNL-01 | EnableFunnel defaults off; toggle sets funnelSessions | unit | `go test ./internal/daemon/... -run TestFunnelSessionsMap` | ❌ Wave 0 |
| FNL-02 | EnableFunnel calls SetServeConfig with AllowFunnel | unit (mock funnelClient) | `go test ./internal/webserver/... -run TestEnableFunnelCallsSetServeConfig` | ❌ Wave 0 |
| FNL-03 | issueCapabilitiesForSession emits Funnel URL (no port) when funnelSessions[id]=true | unit | `go test ./internal/daemon/... -run TestIssueCapabilities_FunnelURL` | ❌ Wave 0 |
| FNL-04 | HTTP request with `Origin: https://hostname.ts.net` gets 200, not 403, when Funnel active | integration (httptest) | `go test ./internal/webserver/... -run TestRequireAllowedOrigin_FunnelOrigin` | ❌ Wave 0 |
| FNL-05 (a) | Disable toggle calls DisableFunnel via handleSetSessionFunnel(false) | unit | `go test ./internal/daemon/... -run TestHandleSetSessionFunnel_DisableTeardown` | ❌ Wave 0 |
| FNL-05 (b) | handleWebServe(false) also tears down Funnel | unit | `go test ./internal/daemon/... -run TestHandleWebServe_FunnelTeardownOnDisable` | ❌ Wave 0 |
| FNL-05 (c) | runSessionExitCleanup tears down Funnel | unit | `go test ./internal/daemon/... -run TestRunSessionExitCleanup_FunnelTeardown` | ❌ Wave 0 |
| FNL-05 (d) | WebServer.Stop() calls DisableFunnel | unit | `go test ./internal/webserver/... -run TestWebServerStop_DisablesFunnel` | ❌ Wave 0 |
| FNL-06 | CheckFunnelAccess error is surfaced; SetServeConfig not called | unit (mock funnelClient) | `go test ./internal/webserver/... -run TestEnableFunnel_PrereqCheckPreventsSetServeConfig` | ❌ Wave 0 |
| FNL-07 | time.AfterFunc fires DisableFunnel at expiry | unit (time.AfterFunc + manual advance) | `go test ./internal/daemon/... -run TestFunnelAutoExpiry` | ❌ Wave 0 |
| FNL-01 | Fallback mode: StatusWithoutPeers error → funnelActive stays false | unit (mock funnelClient returning error) | `go test ./internal/webserver/... -run TestEnableFunnel_FallbackModeSafe` | ❌ Wave 0 |

### Key Test Design Decisions

**Test for FNL-04 (Origin allowlist):**
```go
// Tests MUST use ws.EnableFunnel() (not ws.funnelActive = true directly)
// to catch EnableFunnel→funnelActive wiring bugs (Pitfall 5 / Phase 150 pattern).
ws.EnableFunnel(ctx, 443) // drives real code path
req := httptest.NewRequest("GET", "/sessions/id/ws?cap=TOKEN", nil)
req.Header.Set("Origin", "https://hostname.ts.net") // Funnel origin
// Verify 200 (allowed), not 403
```

**Test for FNL-05 (teardown via real paths):**
```go
// Test runSessionExitCleanup via handleCreateSession → onExit path, not by calling
// runSessionExitCleanup directly (to catch onExit wiring bugs)
// Verify via fakeClient.GetServeConfig() returns empty after cleanup
```

**Mock funnelClient for CI:**
```go
type fakeFunnelClient struct {
    getServeConfig func(ctx context.Context) (*ipn.ServeConfig, error)
    setServeConfig func(ctx context.Context, cfg *ipn.ServeConfig) error
    statusWithoutPeers func(ctx context.Context) (*ipnstate.Status, error)
}
```

### Sampling Rate
- **Per task commit:** `go test ./internal/webserver/... ./internal/daemon/... -run Funnel -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/webserver/funnel_test.go` — TestEnableFunnelCallsSetServeConfig, TestRequireAllowedOrigin_FunnelOrigin, TestEnableFunnel_PrereqCheckPreventsSetServeConfig, TestEnableFunnel_FallbackModeSafe, TestWebServerStop_DisablesFunnel
- [ ] `internal/webserver/funnel_client.go` — `funnelClient` interface + `fakeFunnelClient` test double
- [ ] `internal/daemon/funnel_test.go` — TestFunnelSessionsMap, TestHandleSetSessionFunnel_DisableTeardown, TestHandleWebServe_FunnelTeardownOnDisable, TestRunSessionExitCleanup_FunnelTeardown, TestIssueCapabilities_FunnelURL, TestFunnelAutoExpiry

*(Existing test infrastructure covers all other requirements. New test files must be registered in TESTING.md per the standing convention.)*

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Funnel uses existing cap token auth |
| V3 Session Management | No | Existing session model unchanged |
| V4 Access Control | Yes | Origin allowlist must be extended (not bypassed) for Funnel origin |
| V5 Input Validation | Yes | `CheckFunnelAccess` validates port; `expiresIn` must be a positive duration |
| V6 Cryptography | No | Existing HMAC-SHA256 key unchanged |

### Known Threat Patterns for Phase 165

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cross-site WS hijack from Funnel origin | Tampering | Dual-origin allowlist in `requireAllowedOrigin` + `allowedOrigins` |
| Funnel enabled for wrong session (cap mismatch) | Elevation | `requireCapability` already checks `claims.SID == pathID`; Funnel URL uses session-scoped tokens |
| Stuck Funnel after daemon crash | Denial of Service | Document `tailscale serve reset` in risk dialog; mitigate via 4-site teardown on clean paths |
| ETag race on concurrent SetServeConfig | Tampering | Mutex-guard Enable/Disable + read-modify-write pattern |
| funnelActive=true in local fallback mode | Elevation | Guard: `EnableFunnel` returns error and does not set `funnelActive` when `StatusWithoutPeers` fails |
| Log leak of Funnel URL | Information Disclosure | Log `funnelBaseURL` at DEBUG level only; omit from INFO/WARN structured logs |

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| tailscaled daemon socket | `lc.StatusWithoutPeers`, `lc.SetServeConfig` | Context-dependent | v1.98.3 module pinned | Return user-friendly error; guard fallback mode |
| Tailscale Funnel nodeAttr | `CheckFunnelAccess` | Tailnet-dependent | — | `CheckFunnelAccess` returns human-readable error; FNL-06 handles this |
| Go 1.26.3 | Build | ✓ | 1.26.3 (confirmed in go.mod) | — |

**Missing dependencies with no fallback:** None for Phase 165 itself (Tailscale prereqs are handled by FNL-06 error surfacing, not by code fallback).

---

## Open Questions

1. **FNL-07: Frontend notification of expiry via polling vs Wails event**
   - What we know: Daemon fires `disableFunnelForSession` at expiry; this clears the funnelSessions map and calls ws.DisableFunnel
   - What's unclear: How does the frontend learn that Funnel expired? Should Phase 165 add `funnelActive bool` + `funnelExpiresAt int64` fields to `SessionInfo` (or a new `/funnel-status` endpoint) that the frontend polls? Or does Phase 166 wire the Wails event callback?
   - Recommendation: Add `funnelActive bool` to `SessionInfo` (returned by `GET /sessions`) — simplest approach with zero new IPC. Phase 166 frontend reads this on the existing session-info poll cycle. Keep Wails event approach as a nice-to-have if polling latency is unacceptable.

2. **Port selection: hard-code 443 or detect from CheckFunnelPort?**
   - What we know: Tailscale allows ports 443, 8443, 10000 (policy-controlled); 443 gives a clean URL with no port component
   - What's unclear: Should the code try 443 first, then fall back to 8443 on failure, or always 443 and let CheckFunnelAccess fail with a message?
   - Recommendation: Always use 443. `CheckFunnelAccess(443, st.Self)` will fail with the `"port 443 is not allowed"` error if 443 is blocked. Surface that error. Do not add port-fallback logic — the Tailscale admin can enable 443 and most tailnets default to it.

3. **Daemon restart behavior for active Funnel configs**
   - What we know: `funnelSessions` is in-memory only; daemon restart clears it; but Tailscale serve config persists in tailscaled
   - What's unclear: Should daemon startup probe `lc.GetServeConfig` and clear any lingering Funnel config, or leave it as-is?
   - Recommendation: Clear on startup. In `AutoStartWebServer` (or a new startup hook), call `lc.GetServeConfig(ctx)` and if `sc.IsFunnelOn()`, call `lc.SetServeConfig(ctx, nil)`. This mirrors how existing grants don't survive daemon restart — Funnel state should be equally ephemeral.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Interface injection via `funnelClient` interface is the right testability seam | Testability Seam | Wrong seam choice could make tests hard to write or require CGO in tests |
| A2 | Adding `funnelActive bool` to `SessionInfo` is sufficient for FNL-07 frontend notification | Open Questions #1 | If polling latency is unacceptable, a Wails event mechanism is needed in Phase 166 |
| A3 | Port 443 should always be the default and no fallback to 8443/10000 is needed | Open Questions #2 | If some tailnets don't allow 443, users get an error instead of falling back gracefully |

---

## Sources

### Primary (HIGH confidence)

- `internal/webserver/server.go` (direct inspection: struct lines 78-164, `startTailscale` lines 406-441, `BaseURL` lines 504-522, `Stop` lines 483-491, subscriber creation line 1085) — [VERIFIED]
- `internal/webserver/origin_mw.go` (direct inspection: `requireAllowedOrigin` lines 31-51, `allowedOrigins` lines 60-66) — [VERIFIED]
- `internal/webserver/capability_mw.go` (direct inspection: `originAllowedForWrite` lines 187-198) — [VERIFIED]
- `internal/daemon/api.go` (direct inspection: API struct lines 26-69, `issueCapabilitiesForSession` lines 1217-1299, `handleExchangeJoinCode` lines 1340-1387, `handleWebServe` lines 1181-1206, `runSessionExitCleanup` lines 654-662, `WebServer.Stop` line 483) — [VERIFIED]
- `/Users/ken/go/pkg/mod/tailscale.com@v1.98.3/ipn/serve.go` — `CheckFunnelAccess` line 601, `NodeCanFunnel` line 610, error strings lines 618-621, `SetFunnel` line 469, `IsFunnelOn` line 576 — [VERIFIED]
- `notification_darwin.go` (direct inspection: CGO + UNUserNotificationCenter, NOT osascript) — [VERIFIED]
- `internal/webserver/tailscale.go` (direct inspection: zero-value `local.Client` pattern lines 97-105) — [VERIFIED]

### Secondary (MEDIUM confidence)

- `.planning/research/SUMMARY.md` — project-level Phase 165 deliverables and architecture decisions
- `.planning/research/STACK.md` — Tailscale API reference (verified against module source)
- `.planning/research/PITFALLS.md` — pitfall catalog (verified against direct code inspection)
- `.planning/research/ARCHITECTURE.md` — component change map and data flows

### Tertiary (LOW confidence)

- Interface injection pattern for testability (A1) — standard Go pattern; [ASSUMED] as the right choice for this codebase

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all Funnel APIs verified against tailscale.com@v1.98.3 source on disk; no new packages
- Architecture: HIGH — all file paths and line numbers verified by direct code inspection; code drift documented
- Pitfalls: HIGH — grounded in direct code inspection + project post-mortems (MEMORY.md)

**Research date:** 2026-06-30
**Valid until:** 2026-07-30 (tailscale.com is pinned; Funnel API is stable)
