# Phase 165: Funnel Backend — Pattern Map

**Mapped:** 2026-06-30
**Files analyzed:** 7 (5 modified + 2 new)
**Analogs found:** 7 / 7

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/webserver/server.go` | service | request-response | `internal/webserver/server.go` (itself, existing `BaseURL`, `Stop`, `startTailscale`) | exact — additive to existing struct |
| `internal/webserver/funnel_client.go` (NEW) | utility/interface | request-response | `internal/webserver/tailscale.go` (`statusFunc`/`prefsFunc` injection) | exact — same injectable interface seam pattern |
| `internal/webserver/origin_mw.go` | middleware | request-response | `internal/webserver/origin_mw.go` (itself) | exact — additive dual-origin check |
| `internal/webserver/capability_mw.go` | middleware | request-response | `internal/webserver/capability_mw.go` (itself, `originAllowedForWrite`) | exact — additive dual-origin check |
| `internal/daemon/api.go` | controller | CRUD + event-driven | `internal/daemon/api.go` (itself: `handleWebServe`, `handleSetSessionBrowse`, `runSessionExitCleanup`, `issueCapabilitiesForSession`, `handleExchangeJoinCode`) | exact — additive state map + new endpoint |
| `app.go` | controller | request-response | `app.go` (`ToggleWebServing`, `SetSessionBrowse`) | exact — same thin client-delegation bound method pattern |
| `internal/webserver/funnel_test.go` (NEW) | test | — | `internal/webserver/` (existing test files) | role-match |
| `internal/daemon/funnel_test.go` (NEW) | test | — | `internal/daemon/` (existing test files) | role-match |

---

## Pattern Assignments

### `internal/webserver/server.go` — WebServer struct field promotion + new Funnel methods

**Analog:** `internal/webserver/server.go` (existing struct, `startTailscale`, `BaseURL`, `Stop`)

**Current WebServer struct** (lines 78–164) — to receive three new fields under `ws.mu`:

```go
// Current fields (excerpt, lines 78-96):
type WebServer struct {
    config  Config
    manager *relay.HubManager

    mu         sync.RWMutex
    webEnabled map[string]bool
    listener   net.Listener
    mux        *http.ServeMux
    grants     map[string]map[string]struct{}
    signingKey []byte
    joinCodes  *capability.JoinCodeManager
    // ...
}
```

**Add three Funnel fields under `ws.mu`** (after existing fields, before `mu`):

```go
// NEW — Funnel state, guarded by ws.mu:
funnelActive  bool
funnelBaseURL string  // "https://<hostname>" (no port) when active on port 443
funnelPort    uint16
funnelClient  funnelClient // injectable seam; NewWebServer sets this to &ws.lc
lc            local.Client // promoted from startTailscale() stack-local; zero-value usable
```

**`startTailscale()` pattern** (lines 406–441) — the `var lc local.Client` at line 412 changes to `ws.lc`:

```go
// BEFORE (line 412):
var lc local.Client
tlsCfg = &tls.Config{GetCertificate: lc.GetCertificate, ...}

// AFTER — promote to struct field:
tlsCfg = &tls.Config{GetCertificate: ws.lc.GetCertificate, ...}
```

Note: `handleWSSRelay` creates its own `var lc local.Client` at line ~1121 for WhoIs calls — leave that unchanged.

**`BaseURL()` pattern** (lines 504–522) — add `FunnelBaseURL()` as a parallel method, NOT modifying `BaseURL()`:

```go
// Existing BaseURL (lines 504-522) — DO NOT MODIFY:
func (ws *WebServer) BaseURL() string {
    ws.mu.RLock()
    ln := ws.listener
    ws.mu.RUnlock()
    if ln == nil { return "" }
    _, port, err := net.SplitHostPort(ln.Addr().String())
    if err != nil { return "" }
    if ws.config.Mode == "local" {
        return fmt.Sprintf("https://%s:%s", ws.config.BindIP, port)
    }
    return fmt.Sprintf("https://%s:%s", ws.config.FQDN, port)
}

// NEW — separate method; returns "" when Funnel not active:
func (ws *WebServer) FunnelBaseURL() string {
    ws.mu.RLock()
    defer ws.mu.RUnlock()
    return ws.funnelBaseURL
}
```

**`Stop()` pattern** (lines 482–491) — the existing Stop is the teardown-site 4 analog:

```go
// Current Stop (lines 482-491):
func (ws *WebServer) Stop() error {
    ws.mu.RLock()
    ln := ws.listener
    ws.mu.RUnlock()
    if ln != nil {
        return ln.Close()
    }
    return nil
}
// MODIFIED: call ws.DisableFunnel(ctx) BEFORE ln.Close() — add ctx context.Context param
// OR call from handleWebServerStop before ws.Stop() (see api.go teardown sites below).
```

**`EnableFunnel` / `DisableFunnel` locking pattern** — use `ws.mu.Lock()` consistently with `BaseURL`'s `ws.mu.RLock()` pattern:

```go
// Pattern: read funnelActive/funnelBaseURL under RLock (FunnelBaseURL),
// write funnelActive/funnelBaseURL under Lock (EnableFunnel/DisableFunnel).
// Same mutex as ws.listener (existing: see lines 435-437, 484-486).
```

---

### `internal/webserver/funnel_client.go` (NEW) — injectable interface seam

**Analog:** `internal/webserver/tailscale.go` (lines 28–34 — `statusFunc`/`prefsFunc` injection pattern)

**Existing injection pattern in tailscale.go** (lines 28–34):

```go
// statusFunc is the injectable status function type for testability.
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

// prefsFunc is the injectable prefs-probe function type for testability.
type prefsFunc func(ctx context.Context) (bool, error)

// checkHealth accepts injected fn/pf so tests can pass fakes without a live daemon.
func checkHealth(ctx context.Context, fn statusFunc, ...) TailscaleHealth {
```

**Copy this pattern** — define a narrow interface covering only what `EnableFunnel`/`DisableFunnel` need:

```go
// funnelClient is the injectable interface for Funnel lifecycle operations.
// Production: &ws.lc (concrete local.Client). Tests: fakeFunnelClient.
// Mirrors the statusFunc/prefsFunc injection idiom in tailscale.go:28-34.
type funnelClient interface {
    GetServeConfig(ctx context.Context) (*ipn.ServeConfig, error)
    SetServeConfig(ctx context.Context, config *ipn.ServeConfig) error
    StatusWithoutPeers(ctx context.Context) (*ipnstate.Status, error)
}
```

**In `NewWebServer`** (lines 168–179), add after existing field initialization:

```go
ws.funnelClient = &ws.lc  // concrete implementation; tests override via ws.funnelClient = &fakeFunnelClient{...}
```

**Test double pattern** (copy from `checkHealth` test pattern):

```go
type fakeFunnelClient struct {
    getServeConfig     func(ctx context.Context) (*ipn.ServeConfig, error)
    setServeConfig     func(ctx context.Context, cfg *ipn.ServeConfig) error
    statusWithoutPeers func(ctx context.Context) (*ipnstate.Status, error)
}
func (f *fakeFunnelClient) GetServeConfig(ctx context.Context) (*ipn.ServeConfig, error) {
    return f.getServeConfig(ctx)
}
// ... etc.
```

---

### `internal/webserver/origin_mw.go` — dual-origin allowlist

**Analog:** `internal/webserver/origin_mw.go` (itself, lines 31–66)

**Existing `requireAllowedOrigin`** (lines 31–51):

```go
func (ws *WebServer) requireAllowedOrigin(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if origin == "" {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }
        allowed := ws.BaseURL()
        if allowed == "" || origin != allowed {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }
        next(w, r)
    }
}
```

**MODIFIED** — add Funnel secondary origin check (preserve existing fail-closed logic):

```go
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
        // Funnel: secondary origin (empty string when Funnel not active — fail closed).
        if funnelURL := ws.FunnelBaseURL(); funnelURL != "" && origin == funnelURL {
            next(w, r)
            return
        }
        http.Error(w, "forbidden", http.StatusForbidden)
    }
}
```

**Existing `allowedOrigins()`** (lines 60–66):

```go
func (ws *WebServer) allowedOrigins() []string {
    base := ws.BaseURL()
    if base == "" {
        return nil
    }
    return []string{base}
}
```

**MODIFIED** — add Funnel origin to list:

```go
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

---

### `internal/webserver/capability_mw.go` — dual-origin write check

**Analog:** `internal/webserver/capability_mw.go` (itself, `originAllowedForWrite` lines 187–198)

**Existing `originAllowedForWrite`** (lines 187–198):

```go
func (ws *WebServer) originAllowedForWrite(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    if origin == "" {
        // Desktop Wails fetch() sends no Origin — pass vacuously (CAP-03).
        return true
    }
    // Present Origin: strict byte-for-byte match required (D-03).
    allowed := ws.BaseURL()
    return allowed != "" && origin == allowed
}
```

**MODIFIED** — add Funnel secondary check after existing tailnet check:

```go
func (ws *WebServer) originAllowedForWrite(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    if origin == "" {
        return true // Desktop Wails fetch() — pass vacuously (CAP-03)
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

---

### `internal/daemon/api.go` — new state maps, endpoint, URL builders, teardown sites

**Analogs (all in api.go itself):**
- API struct pattern: lines 26–69 (existing `webServer`, `mu`, `signingKey`, `joinCodes` fields)
- New endpoint pattern: `handleSetSessionBrowse` (lines 1427–1453) and `handleWebServe` (lines 1181–1206)
- URL builder pattern: `issueCapabilitiesForSession` lines 1287–1289; `handleExchangeJoinCode` line 1385
- Teardown pattern: `runSessionExitCleanup` (lines 654–662); `handleWebServerStop` (lines 1128–1138)

**Add to API struct** (after existing fields, before closing brace):

```go
// funnelSessions tracks which sessions have Funnel active (reference count).
// DisableFunnel is called only when the map reaches len==0. Guarded by a.mu.
funnelSessions map[string]bool
// funnelExpiry holds per-session auto-expiry timers (FNL-07). Guarded by a.mu.
funnelExpiry   map[string]*time.Timer
```

**New endpoint registration** — follow `handleSetSessionBrowse` routing pattern (line 144):

```go
// Existing analog (line 144):
a.mux.HandleFunc("POST /sessions/{id}/browse", a.handleSetSessionBrowse)

// NEW — same pattern:
a.mux.HandleFunc("POST /sessions/{id}/funnel", a.handleSetSessionFunnel)
```

**`handleSetSessionFunnel` handler** — copy structure from `handleSetSessionBrowse` (lines 1435–1453) and `handleWebServe` (lines 1181–1206):

```go
// Pattern (from handleSetSessionBrowse lines 1435-1453):
func (a *API) handleSetSessionFunnel(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    var req SetSessionFunnelRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    // webServer nil-guard (from handleWebServe lines 1189-1196):
    a.mu.RLock()
    ws := a.webServer
    a.mu.RUnlock()
    if ws == nil {
        http.Error(w, "web server not running", http.StatusBadRequest)
        return
    }
    // ... enable/disable logic (see RESEARCH.md Pattern 6)
}
```

**`disableFunnelForSession` helper** — analogous to `runSessionExitCleanup` (lines 654–662):

```go
// Analog: runSessionExitCleanup (lines 654-662) — same ws nil-guard + cleanup pattern.
func (a *API) disableFunnelForSession(ctx context.Context, sessionID string) {
    a.mu.Lock()
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
```

**URL builder modification in `issueCapabilitiesForSession`** (lines 1287–1289):

```go
// BEFORE (line 1287):
base := ws.BaseURL()
readURL = base + "/sessions/" + sessionID + "?cap=" + rTok
writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok

// AFTER — Funnel-aware:
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

**URL builder modification in `handleExchangeJoinCode`** (line 1385):

```go
// BEFORE (line 1385):
url := ws.BaseURL() + "/sessions/" + claims.SID + "?cap=" + token

// AFTER — Funnel-aware:
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

**Teardown Site 2 — `handleWebServe` disable path** (after line 1204, `ws.ClearGrants(id)`):

```go
// Existing teardown (lines 1203-1205):
ws.DisableSession(id)
ws.ClearGrants(id)
w.WriteHeader(http.StatusNoContent)

// ADD after ClearGrants:
a.disableFunnelForSession(r.Context(), id)
```

**Teardown Site 3 — `runSessionExitCleanup`** (after line 660, `ws.ClearGrants(sessionID)`):

```go
// Existing teardown (lines 658-661):
ws.DisableSession(sessionID)
ws.ClearGrants(sessionID)

// ADD after ClearGrants:
a.disableFunnelForSession(context.Background(), sessionID)
```

**Teardown Site 4 — `handleWebServerStop`** (lines 1128–1138):

```go
// Existing (lines 1128-1138):
func (a *API) handleWebServerStop(w http.ResponseWriter, r *http.Request) {
    a.mu.Lock()
    ws := a.webServer
    a.webServer = nil
    a.mu.Unlock()
    if ws != nil {
        _ = ws.Stop()
    }
    w.WriteHeader(http.StatusNoContent)
}

// MODIFIED — DisableFunnel before Stop:
func (a *API) handleWebServerStop(w http.ResponseWriter, r *http.Request) {
    a.mu.Lock()
    ws := a.webServer
    a.webServer = nil
    a.mu.Unlock()
    if ws != nil {
        _ = ws.DisableFunnel(r.Context()) // clear Tailscale serve config first
        _ = ws.Stop()
    }
    w.WriteHeader(http.StatusNoContent)
}
```

---

### `app.go` — `SetSessionFunnel` Wails bound method

**Analog:** `app.go` `ToggleWebServing` (line 868) and `SetSessionBrowse` (line 880)

**Existing thin-delegation pattern** (lines 868–885):

```go
func (a *App) ToggleWebServing(sessionID string, enabled bool) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.ToggleWebServing(sessionID, enabled)
}

func (a *App) SetSessionBrowse(sessionID string, enabled bool) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.SetSessionBrowse(sessionID, enabled)
}
```

**Copy this pattern exactly** for `SetSessionFunnel`:

```go
// SetSessionFunnel enables or disables Tailscale Funnel for a session.
// expiresIn is the expiry duration in seconds (0 = no auto-expiry, FNL-07).
func (a *App) SetSessionFunnel(sessionID string, enabled bool, expiresIn int) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.SetSessionFunnel(sessionID, enabled, expiresIn)
}
```

Also add `FunnelActive bool` to the `SessionInfo` struct (lines 29–54) following the same pattern as `WebEnabled`, `BrowseEnabled`, `HomeDir` — no `omitempty` (false must serialize so frontend polling detects expiry):

```go
// Analog: WebEnabled bool `json:"webEnabled"` (line 37)
FunnelActive bool `json:"funnelActive"`
```

---

## Shared Patterns

### Mutex Guard Pattern
**Source:** `internal/webserver/server.go` lines 484–490 and 494–501
**Apply to:** All new `ws.*` methods (`EnableFunnel`, `DisableFunnel`, `FunnelBaseURL`)

```go
// Read pattern (e.g., FunnelBaseURL, BaseURL):
ws.mu.RLock()
val := ws.someField
ws.mu.RUnlock()
return val

// Write pattern (EnableFunnel, DisableFunnel):
ws.mu.Lock()
defer ws.mu.Unlock()
// ... modify ws.funnelActive, ws.funnelBaseURL, ws.funnelPort
```

### webServer Nil-Guard Pattern
**Source:** `internal/daemon/api.go` lines 1189–1196 and 1225–1230
**Apply to:** `handleSetSessionFunnel`, `disableFunnelForSession`

```go
a.mu.RLock()
ws := a.webServer
a.mu.RUnlock()
if ws == nil {
    http.Error(w, "web server not running", http.StatusBadRequest)
    return
}
```

### JSON Decode + PathValue Pattern
**Source:** `internal/daemon/api.go` `handleSetSessionBrowse` lines 1436–1441 and `handleWebServe` lines 1182–1187
**Apply to:** `handleSetSessionFunnel`

```go
id := r.PathValue("id")
var req SomeRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, "invalid request body", http.StatusBadRequest)
    return
}
```

### writeJSON Response Pattern
**Source:** `internal/daemon/api.go` line 1324 (`handleIssueCapabilities`)
**Apply to:** `handleSetSessionFunnel` enable response

```go
writeJSON(w, http.StatusOK, SetSessionFunnelResponse{FunnelURL: ws.FunnelBaseURL()})
```

### time.AfterFunc Teardown Pattern
**Source:** No existing timer usage in api.go — this is a new pattern for FNL-07. The `funnelExpiry` map approach follows the same map-with-cleanup convention as `grants map[string]map[string]struct{}` (api.go:43) and `remoteCaps` (api.go:60).

```go
// FNL-07 — after registering funnelSessions[id]:
if req.ExpiresIn > 0 {
    if a.funnelExpiry == nil {
        a.funnelExpiry = make(map[string]*time.Timer)
    }
    if t, ok := a.funnelExpiry[id]; ok {
        t.Stop() // cancel any existing timer for this session
    }
    dur := time.Duration(req.ExpiresIn) * time.Second
    a.funnelExpiry[id] = time.AfterFunc(dur, func() {
        a.disableFunnelForSession(context.Background(), id)
    })
}
```

---

## Anti-Pattern Guard

The research identifies critical anti-patterns that must be enforced during planning:

1. **Do NOT modify `BaseURL()`** to return the Funnel URL when active. `BaseURL()` feeds the tray icon, QR code, and Settings copy button — those must stay on the tailnet URL. Add `FunnelBaseURL()` separately.

2. **Do NOT store `local.Client` by pointer.** Store by value: `lc local.Client`. Existing pattern: `tailscale.go` lines 97 and 105 confirm zero-value usability.

3. **Do NOT call `DisableFunnel()` without checking `len(funnelSessions) == 0`** — would tear down Funnel for all active sessions when only one toggles off.

4. **Do NOT construct `&ipn.ServeConfig{}` literal in `EnableFunnel`** — always `GetServeConfig` first to preserve ETag.

5. **Do NOT call `CheckFunnelAccess` inside `ws.mu.Lock()`** — `StatusWithoutPeers` is a Unix socket call that can block; call before acquiring the lock.

6. **Tests MUST call `ws.EnableFunnel()`**, not set `ws.funnelActive = true` directly (Pitfall 5 / Phase 150 wrong-assumption pattern from MEMORY.md).

---

## No Analog Found

All Phase 165 files have strong analogs in the existing codebase. No novel patterns required.

---

## Metadata

**Analog search scope:** `/Users/ken/dev/agenthub/internal/webserver/`, `/Users/ken/dev/agenthub/internal/daemon/`, `/Users/ken/dev/agenthub/app.go`
**Files read:** `server.go`, `origin_mw.go`, `capability_mw.go`, `tailscale.go`, `api.go` (key sections), `app.go` (key sections)
**Pattern extraction date:** 2026-06-30
