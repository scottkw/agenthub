# Phase 60: Local Network Fallback - Research

**Researched:** 2026-04-09
**Domain:** Go TLS (self-signed certs), HTTP Basic Auth middleware, LAN IP selection, React persistent banner layout, daemon password lifecycle
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NET-01 | User can serve sessions over the local network with self-signed TLS when Tailscale is not available | New `selfcert.go` generates P256 CA+leaf in memory; `Config.Mode = "local"` triggers `startLocal()` path in `Start()` |
| NET-02 | Local network mode generates a random password for all web connections via HTTP Basic Auth | `generatePassword()` in `runDaemonCore` using `crypto/rand`; passed to `NewWebServer`; `auth.go` middleware wraps all routes |
| NET-03 | Web server binds to a local network interface when operating in local mode | `GetLANIP()` helper in `internal/webserver/localip.go` — `net.InterfaceAddrs()` scan; preference order documented in code; passed as `Config.BindIP` |
| NET-04 | User sees a persistent nudge banner on each launch recommending Tailscale installation when in local mode | New `LocalNetworkBanner` component rendered as sibling to `app__content` in `App.tsx`; never inside `terminal-container` |
| NET-05 | User can view the generated password in the UI (settings/status area) | New daemon API endpoint `GET /webserver/local-password`; new Wails-bound `GetLocalNetworkPassword()` in `app.go`; displayed in `SettingsTab` Web Server section |
</phase_requirements>

---

## Summary

Phase 60 adds a local network fallback mode for users who do not have Tailscale installed or connected. When Tailscale is unavailable at daemon startup, the daemon can start the web server using a self-signed TLS certificate bound to the machine's LAN IP address. Access is protected by a randomly generated password delivered via HTTP Basic Auth (browsers prompt natively). The password persists for the daemon's lifetime — not per web server start/stop cycle. The password is visible in Settings > Web Server and a persistent nudge banner in the app recommends Tailscale installation.

The codebase already has most primitives needed. The key additions are: (1) a `selfcert.go` file for in-memory P256 CA+leaf cert generation (stdlib only), (2) an `auth.go` Basic Auth middleware, (3) a `localip.go` LAN IP selection helper, (4) mode-aware changes to `webserver.Config` and `Start()`, (5) password generation in `runDaemonCore`, (6) a new daemon API endpoint and Wails binding to expose the password to the GUI, and (7) two frontend additions (Settings password display, nudge banner).

Phase 59 implementations are already complete and in the codebase. Phase 60 builds on the same `*API.AutoStartWebServer` and `runDaemonCore` patterns established in Phase 59.

**Primary recommendation:** Add `Config.Mode string` ("tailscale" | "local") and `Config.Password string`. Generate password once in `runDaemonCore` (not in `AutoStartWebServer`). Use separate `startTailscale()` and `startLocal()` private methods in `Start()`. Never embed the password in any URL. Render the nudge banner as a sibling of `app__content` using `position: sticky` or normal flow above the tab bar — never inside `terminal-container`.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `crypto/ecdsa`, `crypto/elliptic`, `crypto/x509` | go1.26.2 | Self-signed cert generation (P256) | Already used in `server_test.go` — same CA+leaf pattern already tested |
| Go stdlib `crypto/rand` | go1.26.2 | Password entropy generation | Already imported in `internal/pty/native.go`; 16 bytes → base64url → ~22 chars |
| Go stdlib `net` | go1.26.2 | `net.InterfaceAddrs()` for LAN IP selection | Already in project; no new dep |
| Go stdlib `encoding/base64` | go1.26.2 | Password encoding from crypto/rand bytes | Already in `app.go` |
| Go stdlib `net/http` | go1.26.2 | Basic Auth middleware (`r.BasicAuth()`) | Standard library; no new dep |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/scottkw/agenthub/internal/webserver` | local | Extended with `selfcert.go`, `auth.go`, `localip.go` | All local mode infrastructure lives here |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| HTTP Basic Auth | Cookie/session token | Basic Auth is natively supported by all browsers — browser prompts for credentials automatically; cookie sessions require a login form (more code) |
| In-memory self-signed cert | Writing cert to disk | Disk write creates world-readable private key; in-memory is both simpler and more secure |
| P256 curve | P521 (Go default) | Chrome and Firefox reject P521 with `tls: illegal parameter` — a silent TLS failure, not a cert-warning prompt |
| CA+leaf two-cert chain | Single self-signed cert | Some TLS clients reject single certs that are both CA and leaf; CA+leaf is the pattern already validated in `server_test.go` |
| `net.InterfaceAddrs()` scan | Dialing `8.8.8.8:80` to get outbound IP | Dial approach fails on offline networks; `InterfaceAddrs` scan with preference order works offline |

**Installation:**
```bash
# No new packages required — all changes use Go stdlib only
```

---

## Architecture Patterns

### Recommended File Structure Changes

```
internal/webserver/
├── server.go          # MODIFIED: Config.Mode + Config.Password; startTailscale/startLocal dispatch; BaseURL() mode-aware; auth middleware applied for local mode
├── selfcert.go        # NEW: GenerateSelfSignedCert(ip string) (*tls.Config, error) — P256 CA+leaf, in-memory
├── auth.go            # NEW: basicAuthMiddleware(password string) func(http.Handler) http.Handler
├── localip.go         # NEW: GetLANIP() (string, error) — net.InterfaceAddrs() scan with preference order
├── tailscale.go       # UNCHANGED
└── tailscale_test.go  # UNCHANGED

internal/daemon/
├── process.go         # MODIFIED: generate password once at startup; pass to AutoStartWebServer when in local mode; detect local mode condition
├── api.go             # MODIFIED: AutoStartWebServer gains mode/password params; new handleGetLocalPassword; handleWebServerStatus returns Mode
├── types.go           # MODIFIED: WebServerStartRequest adds Mode+Password; WebServerStatusResponse adds Mode+Password (for local mode)
└── client.go          # MODIFIED: GetWebServerStatus returns Mode; new GetLocalNetworkPassword()

app.go                 # MODIFIED: StartWebServer detects fallback condition; exposes GetLocalNetworkPassword Wails binding

frontend/src/
├── App.tsx            # MODIFIED: webServerMode state ('tailscale'|'local'|null); LocalNetworkBanner rendered as sibling to app__content
└── components/
    └── SettingsTab.tsx  # MODIFIED: Web Server tab shows password field + mode indicator when isServerRunning && mode == 'local'
```

### Pattern 1: Config Mode Discrimination

**What:** Add `Mode string` ("tailscale" | "local") and `Password string` to `webserver.Config`. `Start()` dispatches to `startTailscale()` or `startLocal()` based on Mode. `BaseURL()` computes the URL from the actual listener address and mode — no FQDN for local mode.

**When to use:** Every call to `NewWebServer` must specify Mode explicitly.

**Example:**
```go
// internal/webserver/server.go
type Config struct {
    BindIP    string     // Tailscale IP (100.x.x.x) or LAN IP (192.168.x.x)
    Port      int        // preferred port; 0 → random
    FQDN      string     // Tailscale MagicDNS hostname; empty for local mode
    TLSConfig *tls.Config // test override; nil in production
    Mode      string     // "tailscale" | "local" — required
    Password  string     // non-empty when Mode == "local"; Basic Auth password
}

func (ws *WebServer) Start() error {
    switch ws.config.Mode {
    case "local":
        return ws.startLocal()
    default: // "tailscale"
        return ws.startTailscale()
    }
}
```

### Pattern 2: Self-Signed Cert Generation (selfcert.go)

**What:** Generate an in-memory P256 CA + leaf cert signed by that CA. The CA cert is stored in the WebServer struct only to populate the `TLSConfig.Certificates` field. No disk writes.

**Example:**
```go
// Source: internal/webserver/server_test.go — selfSignedTLSForTest() is the reference pattern
// internal/webserver/selfcert.go
func GenerateSelfSignedCert(ip string) (*tls.Config, error) {
    caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil { return nil, err }
    caTmpl := &x509.Certificate{
        SerialNumber:          big.NewInt(1),
        Subject:               pkix.Name{Organization: []string{"AgentHub Local"}},
        NotBefore:             time.Now().Add(-time.Minute),
        NotAfter:              time.Now().Add(365 * 24 * time.Hour),
        IsCA:                  true,
        BasicConstraintsValid: true,
        KeyUsage:              x509.KeyUsageCertSign,
    }
    caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
    if err != nil { return nil, err }
    caCert, err := x509.ParseCertificate(caDER)
    if err != nil { return nil, err }

    leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil { return nil, err }
    leafTmpl := &x509.Certificate{
        SerialNumber: big.NewInt(2),
        Subject:      pkix.Name{CommonName: ip},
        IPAddresses:  []net.IP{net.ParseIP(ip)},
        NotBefore:    time.Now().Add(-time.Minute),
        NotAfter:     time.Now().Add(365 * 24 * time.Hour),
        KeyUsage:     x509.KeyUsageDigitalSignature,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
    }
    leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
    if err != nil { return nil, err }
    leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
    if err != nil { return nil, err }

    tlsCert, err := tls.X509KeyPair(
        pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}),
        pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER}),
    )
    if err != nil { return nil, err }
    return &tls.Config{
        Certificates: []tls.Certificate{tlsCert},
        MinVersion:   tls.VersionTLS12,
    }, nil
}
```

### Pattern 3: HTTP Basic Auth Middleware (auth.go)

**What:** Wrap the entire http.ServeMux with Basic Auth for local mode. For dashboard and other routes, return `401 Unauthorized` with `WWW-Authenticate: Basic realm="AgentHub"` when Authorization header is absent or wrong. Browser will show native credential prompt.

**Example:**
```go
// internal/webserver/auth.go
func basicAuthMiddleware(password string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            _, pass, ok := r.BasicAuth()
            if !ok || pass != password {
                w.Header().Set("WWW-Authenticate", `Basic realm="AgentHub"`)
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

**Applied in `startLocal()`:**
```go
func (ws *WebServer) startLocal() error {
    tlsCfg, err := GenerateSelfSignedCert(ws.config.BindIP)
    if err != nil { return fmt.Errorf("selfcert: %w", err) }
    addr := fmt.Sprintf("%s:%d", ws.config.BindIP, ws.config.Port)
    ln, err := tls.Listen("tcp", addr, tlsCfg)
    // ... port fallback same as existing ...
    ws.mu.Lock()
    ws.listener = ln
    ws.mu.Unlock()
    handler := basicAuthMiddleware(ws.config.Password)(ws.mux)
    go http.Serve(ln, handler) //nolint:errcheck
    return nil
}
```

### Pattern 4: LAN IP Selection (localip.go)

**What:** Iterate `net.InterfaceAddrs()` to find the first non-loopback, non-link-local, non-Tailscale IPv4 address. Preference order: `en0` (primary Wi-Fi on macOS) → `eth0` (primary Ethernet on Linux) → any other private-range IPv4.

**Why explicit preference:** On machines with multiple network interfaces (Wi-Fi + Ethernet + Docker bridge + Tailscale), `net.InterfaceAddrs()` returns addresses in an undefined order. Tailscale's IP (100.64.0.0/10) must be excluded from local mode binding — it would make the "local" server only reachable on the Tailscale network, which defeats the purpose.

**Example:**
```go
// internal/webserver/localip.go
// GetLANIP returns the first suitable LAN IPv4 address.
// Preference order: named primary interface (en0/eth0) → any private-range IPv4.
// Excludes: loopback (127.x), link-local (169.254.x), Tailscale CGNAT (100.64-127.x).
func GetLANIP() (string, error) {
    ifaces, err := net.Interfaces()
    if err != nil { return "", err }
    // First pass: prefer known primary interface names
    for _, pref := range []string{"en0", "eth0", "wlan0"} {
        if ip := ipFromInterface(ifaces, pref); ip != "" {
            return ip, nil
        }
    }
    // Second pass: any non-loopback, non-link-local private IPv4
    for _, iface := range ifaces {
        if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
            continue
        }
        if ip := firstPrivateIPv4(iface); ip != "" {
            return ip, nil
        }
    }
    return "", fmt.Errorf("no suitable LAN IP found")
}

func isTailscaleIP(ip net.IP) bool {
    // Tailscale CGNAT range: 100.64.0.0/10
    return ip[0] == 100 && ip[1] >= 64 && ip[1] <= 127
}
```

### Pattern 5: Password Lifecycle — Generated Once in runDaemonCore

**What:** The password is generated once when the daemon starts. It is stored in a variable in `runDaemonCore` and passed to `AutoStartWebServer` (when in local mode) and to a new `GetLocalNetworkPassword()` API endpoint. It is never written to disk.

**Decision already recorded in STATE.md:** "Password lifetime = daemon lifetime (generated once in runDaemonCore, not per server start)".

**Example:**
```go
// internal/daemon/process.go — in runDaemonCore, after api.Start()
localPassword := generateLocalPassword() // 16 bytes crypto/rand → base64url
api.SetLocalPassword(localPassword)       // stores for GetLocalNetworkPassword handler

h := webserver.CheckHealth(ctx5s)
cancel()
if h.Connected && h.HasCerts && h.IP != "" {
    // Tailscale mode
    if err := api.AutoStartWebServer(h.IP, 7443, h.Domain, "tailscale", ""); ...
} else {
    // Local fallback
    lanIP, err := webserver.GetLANIP()
    if err == nil {
        if err := api.AutoStartWebServer(lanIP, 7443, "", "local", localPassword); ...
    }
}
```

**Password generation helper:**
```go
// internal/daemon/process.go or a small helper
func generateLocalPassword() string {
    b := make([]byte, 16)
    _, _ = rand.Read(b)
    return base64.RawURLEncoding.EncodeToString(b) // 22 chars
}
```

### Pattern 6: Frontend Nudge Banner

**What:** A persistent banner rendered as a **sibling of `app__content`**, not inside `terminal-container`. This is the same pattern that `HealthModal`, `NewSessionModal`, and `QRModal` follow in `App.tsx`. The banner appears when `webServerMode === 'local'` (a new piece of state in `App.tsx`).

**Decision already recorded in STATE.md:** "Nudge banner renders as sibling to app__content, never inside terminal flex container."

**JSX position in App.tsx:**
```tsx
return (
  <div className="app">
    <Sidebar ... />
    {webServerMode === 'local' && (
      <LocalNetworkBanner
        onOpenSettings={handleOpenSettings}
        onDismiss={() => { /* optional: localStorage dismiss */ }}
        platform={platform}
      />
    )}
    <div className="app__content">
      {/* TabBar, terminal-container, etc. — UNCHANGED */}
    </div>
    {/* existing overlays: NewSessionModal, QRModal, HealthModal */}
  </div>
)
```

**CSS for banner — does NOT shrink terminal area:**
```css
/* LocalNetworkBanner renders as a flex row sibling to .app__content */
/* .app is flex-direction: row; the banner needs to span full width */
/* Approach: render banner ABOVE the row by using a wrapper column or position:sticky */

/* Simplest: wrap the Sidebar + app__content in a column, banner at top */
/* .app layout change: flex-direction: column for a new inner wrapper */
```

Wait — the existing `.app` layout is `flex-direction: row` (Sidebar + app__content side by side). To place a banner above both, we need one of:

**Option A (recommended):** Wrap the row in a column. `<div className="app">` becomes a column; a new `<div className="app__row">` holds Sidebar + app__content.

**Option B:** Make the banner `position: fixed` at top, add `padding-top` to `.app__content`. This is fragile with the rAF terminal fit loop.

**Option C:** Render banner inside `app__content` before `terminal-container` but outside `terminal-container`. Since `.app__content` is `flex-direction: column`, a banner child would push `terminal-container` down — this shrinks the terminal.

**Recommendation: Option A.** It is the cleanest structural change and has no interaction with the terminal flex chain.

```css
/* New layout after Phase 60 */
.app {
  display: flex;
  flex-direction: column; /* was row — changed */
  height: 100%;
  width: 100%;
}

.app__row {
  display: flex;
  flex-direction: row;   /* sidebar + content */
  flex: 1;
  min-height: 0;
}

.local-network-banner {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  background-color: #2a1a00;
  border-bottom: 1px solid #ff9e3b;
  font-size: 12px;
  color: #ff9e3b;
}
```

The `Sidebar` and `<div className="app__content">` move inside `<div className="app__row">`. The banner sits above `app__row` inside `app`. Terminal rAF fit loop is unaffected because `terminal-container` is inside `app__content` inside `app__row` — banner height is consumed by the new outer column layout without touching the inner flex chain.

### Pattern 7: Settings Tab — Password Display

**What:** Add a "Local Network Password" field to the Web Server tab in `SettingsTab.tsx`. Visible only when `isServerRunning && mode === 'local'`. Shows the password masked by default with a reveal toggle. The field is read-only.

**Wails binding needed:**
```go
// app.go — new Wails-bound method
func (a *App) GetLocalNetworkPassword() string {
    if a.client == nil { return "" }
    pwd, _ := a.client.GetLocalNetworkPassword()
    return pwd
}
```

**Daemon API endpoint needed:**
```go
// internal/daemon/api.go
a.mux.HandleFunc("GET /webserver/local-password", a.handleGetLocalPassword)

func (a *API) handleGetLocalPassword(w http.ResponseWriter, r *http.Request) {
    a.mu.RLock()
    pwd := a.localPassword
    a.mu.RUnlock()
    writeJSON(w, http.StatusOK, map[string]string{"password": pwd})
}
```

**`localPassword` field on `*API`:**
```go
// internal/daemon/api.go
type API struct {
    // ... existing fields ...
    localPassword string // non-empty when web server is in local mode
}

func (a *API) SetLocalPassword(pwd string) {
    a.mu.Lock()
    a.localPassword = pwd
    a.mu.Unlock()
}
```

**`WebServerStatusResponse` mode field:**
```go
// internal/daemon/types.go
type WebServerStatusResponse struct {
    Running bool   `json:"running"`
    URL     string `json:"url"`
    Addr    string `json:"addr"`
    Mode    string `json:"mode"` // "tailscale" | "local" | ""
}
```

**BaseURL() for local mode:** Use the listener's actual IP:port, not FQDN.

```go
// internal/webserver/server.go
func (ws *WebServer) BaseURL() string {
    ws.mu.RLock()
    ln := ws.listener
    mode := ws.config.Mode
    ws.mu.RUnlock()
    if ln == nil { return "" }
    _, port, err := net.SplitHostPort(ln.Addr().String())
    if err != nil { return "" }
    if mode == "local" {
        return fmt.Sprintf("https://%s:%s", ws.config.BindIP, port)
    }
    return fmt.Sprintf("https://%s:%s", ws.config.FQDN, port)
}
```

### Anti-Patterns to Avoid

- **Password in session URL:** Never add `?pw=xxx` or `#pw=xxx` to session URLs. Use `Authorization: Basic` header only. Browsers prompt natively — no login form needed.
- **Password rolled on StopWebServer/StartWebServer:** The password lives in `runDaemonCore`, not in `WebServer`. `StopWebServer + StartWebServer` reuses the same password (daemon hasn't restarted).
- **P521 curve:** `ecdsa.GenerateKey(elliptic.P521(), ...)` fails silently in Chrome. Always use `elliptic.P256()`.
- **Single self-signed cert as both CA and leaf:** Use CA + leaf pair. The test helper `selfSignedTLSForTest` in `server_test.go` already demonstrates the correct two-cert pattern.
- **Banner inside `terminal-container`:** Shrinks terminal height, breaks the rAF fit loop. Must be a sibling to `app__content` (Option A above).
- **Starting local server before checking for LAN IP:** `GetLANIP()` can fail on unusual network configurations. Log the failure and skip local start (same pattern as Tailscale not ready) — don't crash the daemon.
- **Binding to `0.0.0.0` in local mode:** Binds to all interfaces including Tailscale. Bind to the specific LAN IP.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Self-signed TLS cert | Custom ASN.1 serialization | Go stdlib `crypto/tls`, `crypto/x509`, `crypto/ecdsa` | Already fully demonstrated in `server_test.go::selfSignedTLSForTest` — copy that exact pattern |
| Basic Auth parsing | Custom header parsing | `r.BasicAuth()` (stdlib `net/http`) | Handles all RFC 7617 encoding edge cases |
| Random password | Weak `math/rand` generation | `crypto/rand.Read` + `base64.RawURLEncoding.EncodeToString` | Already used in `internal/pty/native.go`; cryptographically secure |
| LAN IP detection | Custom netlink probing | `net.Interfaces()` + `net.InterfaceAddrs()` (stdlib) | Works on all platforms (macOS/Linux/Windows) |
| Auth middleware | Third-party middleware library | Simple `http.Handler` wrapper (stdlib) | 8 lines; no new deps; full control |

**Key insight:** Every component needed for Phase 60 is either already in the stdlib (already imported by this project) or is a direct extension of existing patterns in the codebase.

---

## Common Pitfalls

### Pitfall 1: P521 TLS Curve — Silent Browser Failure
**What goes wrong:** Chrome and Firefox reject P521 ECDSA certificates with `tls: illegal parameter`. The browser shows a connection error — NOT a "this certificate is not trusted" warning. Users cannot click through.
**Why it happens:** Go's default curve and examples use P521.
**How to avoid:** Always pass `elliptic.P256()` to `ecdsa.GenerateKey`. The `selfSignedTLSForTest` helper already uses P256 — copy that pattern exactly.
**Warning signs:** `curl` succeeds but Chrome shows `ERR_SSL_VERSION_OR_CIPHER_MISMATCH`.

### Pitfall 2: Banner Inside Terminal Flex Container
**What goes wrong:** Adding a banner inside `terminal-container` shrinks the terminal height. The rAF-based `fit` loop fires, detects the wrong dimensions, and produces a terminal that appears undersized or triggers extra resize events.
**Why it happens:** The natural place to insert a "top of content area" banner is inside the content area — but `.app__content` is a column flex container where children fill vertically. Adding a banner shrinks `terminal-container`'s available space.
**How to avoid:** Wrap `Sidebar` + `app__content` in a new `app__row` div. Place the banner as the first child of the outer `.app` column, above `app__row`. Terminal fill is unaffected.
**Warning signs:** Terminal text appears clipped at the bottom; `fit` triggers extra times on banner visibility change.

### Pitfall 3: Password Rolled on Server Restart
**What goes wrong:** If the password is generated inside `handleWebServerStart` or `AutoStartWebServer`, stopping and restarting the web server generates a new password. Browser sessions using the old password get 401 errors with no explanation.
**Why it happens:** Password generation naturally flows to the same place as server initialization.
**How to avoid:** Generate password in `runDaemonCore` once per daemon lifetime. Store in `*API.localPassword`. `AutoStartWebServer` receives the password as a parameter — it does not generate it.
**Warning signs:** Settings tab shows a different password after clicking Stop/Start Web Server.

### Pitfall 4: Binding to 0.0.0.0 in Local Mode
**What goes wrong:** `tls.Listen("tcp", "0.0.0.0:7443", ...)` makes the server reachable on all interfaces, including the Tailscale interface (100.x.x.x). Users who have Tailscale can reach the local server from remote machines without going through the intended LAN-only path.
**Why it happens:** `0.0.0.0` is the default "listen everywhere" address and appears to work.
**How to avoid:** `GetLANIP()` returns a specific LAN IP; use that as `Config.BindIP`.
**Warning signs:** `curl https://100.x.x.x:7443` succeeds when it should fail.

### Pitfall 5: `BaseURL()` Returns FQDN in Local Mode
**What goes wrong:** If `BaseURL()` always returns `https://hostname.ts.net:PORT`, the session URL shown in Settings and the QR code will point to the Tailscale hostname — which is unreachable when Tailscale is not connected.
**Why it happens:** `BaseURL()` was written for Tailscale-only operation; it uses `ws.config.FQDN` unconditionally.
**How to avoid:** Branch on `ws.config.Mode` in `BaseURL()`: local mode returns `https://LAN_IP:PORT`; Tailscale mode returns `https://FQDN:PORT`.
**Warning signs:** Settings shows a `.ts.net` URL even though Tailscale is not installed.

### Pitfall 6: AutoStartWebServer — Local Mode Condition
**What goes wrong:** `runDaemonCore` currently skips auto-start when `!h.Connected || !h.HasCerts`. For Phase 60, the else branch should attempt local mode. If `GetLANIP()` fails, the daemon should log and continue (no local web server) — not crash.
**Why it happens:** The current code treats Tailscale absence as "skip web server entirely". Phase 60 changes this to "fall back to local mode".
**How to avoid:** Add an `else` branch after the Tailscale check: get LAN IP, on failure log and skip, on success call `AutoStartWebServer` in local mode.
**Warning signs:** Local mode never starts even when Tailscale is absent; no log message about LAN IP failure.

### Pitfall 7: HealthModal Hides When It Should Not (Local Mode Context)
**What goes wrong:** `HealthModal` currently hides when `isInstalled && isConnected && hasCerts`. In local mode, the user may have Tailscale installed but not connected — the HealthModal would show, blocking the UI. The nudge banner is supposed to be the non-blocking way to encourage Tailscale.
**Why it happens:** HealthModal was designed for the "Tailscale is required" assumption. Phase 60 changes the assumption: Tailscale is now optional (local mode is the fallback).
**How to avoid:** Modify `HealthModal`'s visibility logic: only show when the web server is NOT running (regardless of mode). If the web server is running (even in local mode), the user can work — HealthModal should not block them.

Alternatively: `HealthModal` shows only when `isWebServerRunning === false`. This is a simple state check. The nudge banner separately communicates "you're in local mode, consider Tailscale."

**Warning signs:** `HealthModal` blocks the entire UI for users who have Tailscale installed but not connected, even though local mode web server is running.

---

## Code Examples

### Verified Pattern: Two-Cert TLS Config (from server_test.go)
```go
// Source: internal/webserver/server_test.go — selfSignedTLSForTest (lines 29-88)
// This is the exact CA+leaf P256 pattern to copy into selfcert.go.
// The only difference in production: use ws.config.BindIP instead of "127.0.0.1",
// and set NotAfter to 365 days instead of 1 hour.
caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
// ... CA cert template with IsCA: true ...
leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
// ... leaf cert template with IPAddresses: []net.IP{net.ParseIP(ip)} ...
tlsCert, _ := tls.X509KeyPair(leafCertPEM, leafKeyPEM)
serverTLS := &tls.Config{Certificates: []tls.Certificate{tlsCert}, MinVersion: tls.VersionTLS12}
```

### Verified Pattern: Auto-Start in runDaemonCore (from process.go)
```go
// Source: internal/daemon/process.go — lines 52-65
// Phase 60 extends the else branch:
h := webserver.CheckHealth(ctx5s)
cancel()
if h.Connected && h.HasCerts && h.IP != "" {
    // Tailscale mode (unchanged from Phase 59)
    if err := api.AutoStartWebServer(h.IP, 7443, h.Domain, "tailscale", ""); ...
} else {
    // NEW: Local mode fallback
    lanIP, lanErr := webserver.GetLANIP()
    if lanErr != nil {
        fmt.Fprintf(os.Stderr, "daemon: local mode: no LAN IP: %v\n", lanErr)
    } else {
        if err := api.AutoStartWebServer(lanIP, 7443, "", "local", localPassword); ...
    }
}
```

### Verified Pattern: handleWebServerStatus Mode Field
```go
// Source: internal/daemon/api.go — handleWebServerStatus (lines 354-368)
// Phase 60 adds Mode to the response:
if ws != nil {
    writeJSON(w, http.StatusOK, WebServerStatusResponse{
        Running: true,
        URL:     ws.BaseURL(),
        Addr:    ws.Addr(),
        Mode:    ws.config.Mode, // "tailscale" or "local"
    })
}
```

### Verified Pattern: Basic Auth in Go stdlib
```go
// Source: Go standard library net/http
// r.BasicAuth() decodes the Authorization: Basic <base64(user:pass)> header.
// For AgentHub, the username is ignored — any non-empty password match succeeds.
_, pass, ok := r.BasicAuth()
if !ok || pass != expectedPassword {
    w.Header().Set("WWW-Authenticate", `Basic realm="AgentHub"`)
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Self-signed cert infrastructure (v1.0) | Removed — Tailscale-only (v1.2 Phase 16) | v1.2 | Local mode was removed; Phase 60 re-adds it |
| Password auth (v1.0–v1.1) | Removed — Tailscale network-level auth (v1.2) | v1.2 | Phase 60 re-adds password auth for local mode only |
| No LAN IP detection | `net.InterfaceAddrs()` scan | Phase 60 (new) | Enables local mode binding to correct interface |
| Modal-based settings | Settings-as-tab (Phase 58) | Phase 58 | Settings password display goes in SettingsTab.tsx, not a modal |
| Tailscale-only auto-start (Phase 59) | Local mode fallback auto-start | Phase 60 | Daemon always tries to start a web server; local mode if no Tailscale |

**Deprecated/outdated:**
- `HealthModal` blocking behavior — with local mode fallback, Tailscale is no longer required to start the web server. HealthModal visibility logic must be updated to not block when web server is already running in local mode.

---

## Open Questions

1. **HealthModal visibility in local mode**
   - What we know: `HealthModal` currently returns `null` only when `installed && connected && hasCerts`. In local mode, users may have Tailscale installed but not connected — HealthModal would show and block the UI.
   - What's unclear: Should HealthModal (a) hide when web server is running in any mode, or (b) show in a non-blocking "advisory" form in local mode?
   - Recommendation: Hide HealthModal when `webServerRunning === true`. The nudge banner handles the "suggest Tailscale" messaging in a non-blocking way. The HealthModal remains for the case where no web server is running at all (Tailscale not connected and LAN IP also unavailable).

2. **Port for local mode: 7443 or different?**
   - What we know: Tailscale mode uses 7443. Both modes could share the same port if they never run simultaneously (which is the case — they are mutually exclusive per the Tailscale health check logic).
   - What's unclear: Should local mode use a different port to make the mode visually obvious in the URL?
   - Recommendation: Reuse 7443 for both modes. The port is a daemon-lifetime constant (falls back to random if 7443 is taken). Using the same port simplifies `AutoStartWebServer` signature and is consistent with user expectations.

3. **Nudge banner dismiss persistence**
   - What we know: STATE.md says "nudge banner appears on each launch when running in local network mode." This implies no dismiss persistence.
   - What's unclear: Should the user be able to dismiss the banner for the session (page-lifetime only)?
   - Recommendation: Per STATE.md — no dismiss. The banner appears whenever `webServerMode === 'local'` and disappears automatically if the web server switches to Tailscale mode. This is the simplest implementation and aligns with the stated requirement.

4. **WebSocket `Origin` check in local mode**
   - What we know: The existing WebSocket `AcceptOptions` sets `OriginPatterns: []string{"*"}` (open) because Tailscale provides network-level security. In local mode, there is no network-level security beyond Basic Auth over HTTPS.
   - What's unclear: Should local mode restrict WebSocket origin to the machine's own IP?
   - Recommendation: Keep `OriginPatterns: []string{"*"}` for now. The self-signed cert + Basic Auth provides authentication; CSRF via WebSocket is a lower risk than a broken WebSocket connection from different origin. Flag for v1.12 security hardening.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build + test | Yes | 1.26.2 | — |
| `crypto/ecdsa`, `crypto/x509` | Self-signed cert (selfcert.go) | Yes (stdlib) | — | — |
| `crypto/rand` | Password generation | Yes (stdlib) | — | — |
| `net.Interfaces()` | LAN IP detection | Yes (stdlib) | — | Log failure, skip local start |
| `r.BasicAuth()` | Auth middleware | Yes (stdlib net/http) | — | — |
| Node.js | Frontend tests (Vitest) | Yes | 20.19.3 | — |
| pnpm | Frontend deps | Yes | 9.12.3 | — |
| Tailscale | Tailscale mode auto-start | Not required | — | Local mode fallback (Phase 60) |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** Tailscale is only needed for Tailscale mode; local mode is the fallback for when Tailscale is absent.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `testing` package + `go test` |
| Framework (frontend) | Vitest 4.1.0 |
| Config file (Go) | None (standard `go test`) |
| Config file (frontend) | `frontend/vite.config.ts` |
| Quick run command (Go) | `go test ./internal/webserver/... ./internal/daemon/... -run TestLocal` |
| Full suite command (Go) | `go test ./...` |
| Full suite command (frontend) | `cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NET-01 | Self-signed cert generated with P256 curve | unit (Go) | `go test ./internal/webserver/... -run TestGenerateSelfSignedCert` | No — Wave 0 |
| NET-01 | Local mode `Start()` succeeds and server responds | unit (Go) | `go test ./internal/webserver/... -run TestLocalModeStart` | No — Wave 0 |
| NET-02 | Request without auth header returns 401 | unit (Go) | `go test ./internal/webserver/... -run TestBasicAuthMiddleware_Unauthorized` | No — Wave 0 |
| NET-02 | Request with correct password returns 200 | unit (Go) | `go test ./internal/webserver/... -run TestBasicAuthMiddleware_Authorized` | No — Wave 0 |
| NET-02 | Password unchanged after StopWebServer/StartWebServer | unit (Go) | `go test ./internal/daemon/... -run TestLocalPassword_StableAcrossRestart` | No — Wave 0 |
| NET-03 | `GetLANIP()` returns non-loopback IPv4 | unit (Go) | `go test ./internal/webserver/... -run TestGetLANIP` | No — Wave 0 |
| NET-03 | `GetLANIP()` excludes Tailscale CGNAT range (100.64-127.x) | unit (Go) | `go test ./internal/webserver/... -run TestGetLANIP_ExcludesTailscale` | No — Wave 0 |
| NET-04 | Nudge banner renders when `webServerMode === 'local'` | unit (TS) | `cd frontend && pnpm test --run LocalNetworkBanner` | No — Wave 0 |
| NET-04 | Nudge banner not rendered when `webServerMode === 'tailscale'` | unit (TS) | `cd frontend && pnpm test --run LocalNetworkBanner` | No — Wave 0 |
| NET-05 | `GET /webserver/local-password` returns password when in local mode | unit (Go) | `go test ./internal/daemon/... -run TestGetLocalPassword` | No — Wave 0 |
| NET-05 | `GetLocalNetworkPassword()` returns empty string in Tailscale mode | unit (Go) | `go test ./internal/daemon/... -run TestGetLocalPassword_TailscaleMode` | No — Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/webserver/... ./internal/daemon/... && cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/webserver/selfcert_test.go` — `TestGenerateSelfSignedCert`: verify cert uses P256, has correct IP SAN, passes TLS handshake
- [ ] `internal/webserver/auth_test.go` — `TestBasicAuthMiddleware_Unauthorized`, `TestBasicAuthMiddleware_Authorized`: 401 without creds, 200 with correct password
- [ ] `internal/webserver/localip_test.go` — `TestGetLANIP`: at least one non-loopback IP returned; `TestGetLANIP_ExcludesTailscale`: fake interface with 100.64.x.x address is excluded
- [ ] `internal/webserver/server_test.go` additions — `TestLocalModeStart`: server starts in local mode, responds to authed request; `TestBaseURL_LocalMode`: BaseURL returns IP-based URL, not FQDN
- [ ] `internal/daemon/api_test.go` additions — `TestGetLocalPassword`, `TestAutoStartWebServer_LocalMode`
- [ ] `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` — renders when mode=local, hidden when mode=tailscale

*(Existing test infrastructure: `selfSignedTLSForTest` in `server_test.go` and `testDaemon` in `api_test.go` provide the scaffolding — extend rather than replace)*

---

## Sources

### Primary (HIGH confidence)
- Source code: `internal/webserver/server.go` — `Config`, `Start()`, `BaseURL()`, `setupRoutes()` — current Tailscale-only implementation; base for Phase 60 changes
- Source code: `internal/webserver/server_test.go` — `selfSignedTLSForTest()` — exact P256 CA+leaf pattern to use in `selfcert.go`; verified working
- Source code: `internal/daemon/process.go` — `runDaemonCore()` — Phase 59 auto-start already in place; Phase 60 adds else branch
- Source code: `internal/daemon/api.go` — `AutoStartWebServer()`, `SetWebServerForTest()` — testability patterns; Phase 60 adds mode/password params
- Source code: `internal/daemon/types.go` — `WebServerStatusResponse`, `WebServerStartRequest` — types to extend with Mode/Password fields
- Source code: `frontend/src/App.tsx` — layout structure (`.app`, `.app__content`, modal sibling pattern) — determines banner placement strategy
- Source code: `frontend/src/style.css` — `.app`, `.app__content`, `.terminal-container` flex chain — confirms banner must be outside this chain
- Source code: `frontend/src/components/SettingsTab.tsx` — current Web Server tab structure — integration point for password display
- `.planning/STATE.md` — locked decisions: P256 curve, password=daemon lifetime, banner=sibling to app__content
- `.planning/research/PITFALLS.md` — pre-researched pitfall analysis for all Phase 60 issues (HIGH confidence, 2026-04-08)
- `.planning/research/ARCHITECTURE.md` — pre-researched integration point analysis (HIGH confidence, 2026-04-08)

### Secondary (MEDIUM confidence)
- Go stdlib documentation: `net/http.Request.BasicAuth()` — RFC 7617 Basic Auth parsing (verified via Go 1.26.2 stdlib source)
- Go stdlib: `net.Interfaces()` / `InterfaceAddrs()` — cross-platform interface enumeration

### Tertiary (LOW confidence)
- None — all Phase 60 implementation patterns are fully derived from the live codebase and Go stdlib.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are Go stdlib, already imported in this project; no new external deps
- Architecture: HIGH — all integration points verified by reading actual source files; pre-researched architecture doc confirms approach
- Pitfalls: HIGH — P256 curve, banner layout, password lifetime, mode discrimination all documented from code inspection + pre-research

**Research date:** 2026-04-09
**Valid until:** 2026-05-09 (no external API dependencies; all stdlib)

---

## Project Constraints (from CLAUDE.md)

| Directive | Application to Phase 60 |
|-----------|------------------------|
| Python virtual environment rule | N/A — this phase is Go + TypeScript |
| Go: `go fmt`, `golangci-lint`, context-aware functions | All new Go files must pass `go fmt` and `golangci-lint`; context passed to cert generation if needed |
| JS/TS: ESLint + Prettier, TypeScript types | `LocalNetworkBanner` component must have typed props; no `any` types |
| pnpm preferred | Frontend changes use `pnpm` |
| 80%+ coverage in critical components | Wave 0 test gaps listed above; all new Go files should have tests |
| No `kill node.exe` | N/A |
| Chesterton's Fence: before removing, articulate why it exists | `HealthModal` visibility change: articulate that local mode makes Tailscale optional, so HealthModal should not block when web server is already running |
| Silent Fallbacks (`or {}`) are dangerous | `GetLANIP()` failure must be logged explicitly, not silently ignored via `or ""` |
