# Phase 15: Tailscale TLS + Interface Binding - Research

**Researched:** 2026-03-20
**Domain:** Go TLS, Tailscale LocalAPI (`tailscale.com/client/local`), WebServer refactor
**Confidence:** HIGH

---

## Summary

Phase 15 replaces the existing self-signed CA+leaf TLS infrastructure with Tailscale-provisioned Let's Encrypt certificates, binds the web server exclusively to the Tailscale interface IP (derived from the daemon), and surfaces a Certificate Transparency disclosure before first cert use. All cert generation code in `internal/webserver/tls.go` is deleted, and `Config.BindIP` is no longer caller-supplied — it is always `TailscaleHealth.IP` from the poller.

The core swap is surgical: `server.go::Start()` replaces `GenerateLeafCert + BuildTLSConfig` with `tls.Config{GetCertificate: lc.GetCertificate}`, and `NewWebServer` drops the CA loading call entirely. The FQDN for URLs and QR codes shifts from `ws.Addr()` host-portion to the domain from `TailscaleHealth.Domain`. The startup gate in `app.go::StartWebServer` must check `TailscaleHealth.IP != ""` before proceeding.

The CT disclosure (TLS-04) is a one-time acknowledgement in the Settings UI. It informs the user that their machine's hostname will appear publicly in CT logs once the first cert is issued. This is informational only — no API call is needed; the cert issuance itself happens on the first TLS handshake via `GetCertificate`.

**Primary recommendation:** Use `lc.GetCertificate` as `tls.Config.GetCertificate`. This is the stable, documented, zero-cache API. Never cache `CertPair` at startup (already a locked decision in STATE.md).

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TLS-01 | Web server uses Let's Encrypt certificates provisioned via Tailscale daemon | `lc.GetCertificate` on `local.Client` is the direct API; plugs into `tls.Config.GetCertificate` |
| TLS-02 | Machine FQDN derived from Tailscale daemon, not hardcoded | `TailscaleHealth.Domain` (from `CertDomains[0]`) already populated by `checkHealth`; `BaseURL()` must use this domain |
| TLS-03 | Web server binds exclusively to Tailscale interface IP | `TailscaleHealth.IP` from health check; `StartWebServer` gates on its presence |
| TLS-04 | User warned about CT log exposure before first cert provisioning | One-time consent flag in Settings UI; must be acknowledged before `StartWebServer` can proceed |
| TLS-05 | Self-signed certificate infrastructure removed | Delete `tls.go`, `tls_test.go`; strip CA fields from `WebServer` struct and `Config`; remove `ca.crt`/`ca.key` write paths |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `tailscale.com/client/local` | v1.96.3 (already in go.mod) | `lc.GetCertificate` for TLS, `lc.StatusWithoutPeers` for FQDN+IP | Already vendored; stable API; zero-value `local.Client{}` queries existing daemon |
| `crypto/tls` (stdlib) | Go 1.26.1 | `tls.Config{GetCertificate: ...}`, `tls.Listen` | Standard; no new dep |

### No New Dependencies Required

Everything needed is already present in go.mod:
- `tailscale.com v1.96.3` — provides both health check (Phase 14) and cert retrieval (Phase 15)
- Go stdlib `crypto/tls` — `tls.Config.GetCertificate` hook is the integration point

---

## Architecture Patterns

### Recommended Project Structure (changes only)

```
internal/webserver/
├── tailscale.go         # existing — no changes needed
├── tailscale_test.go    # existing — no changes needed
├── server.go            # MODIFIED: Start() rewritten, BaseURL() uses FQDN, Config changes
├── server_test.go       # MODIFIED: testServer() uses fake GetCertificate hook
├── tls.go               # DELETED (TLS-05)
├── tls_test.go          # DELETED (TLS-05)
├── network.go           # may be simplified (CLEAN-01 is Phase 17, but ListInterfaces no longer needed for binding)
├── auth.go              # unchanged (auth removal is Phase 16)
└── tokens.go            # unchanged (token removal is Phase 16)

app.go                   # MODIFIED: StartWebServer() gates on TS health; new CT disclosure flag
frontend/src/components/
└── SettingsPanel.tsx    # MODIFIED: CT disclosure banner/modal; remove CA cert download UI
```

### Pattern 1: GetCertificate Hook (TLS-01)

**What:** `tls.Config.GetCertificate` is a callback invoked per-handshake. The Tailscale client's `GetCertificate` handles ACME provisioning and caching internally.

**When to use:** Always for Tailscale-backed TLS. Never cache `CertPair` at startup.

**Example:**
```go
// Source: tailscale.com@v1.96.3/client/local/cert.go (verified in GOMODCACHE)
var lc local.Client  // zero value — connects to existing tailscaled

tlsCfg := &tls.Config{
    GetCertificate: lc.GetCertificate,
    MinVersion:     tls.VersionTLS12,
}
ln, err := tls.Listen("tcp", addr, tlsCfg)
```

`lc.GetCertificate` signature: `func(hi *tls.ClientHelloInfo) (*tls.Certificate, error)`
- Uses `hi.ServerName` (SNI) to identify which cert to fetch
- Falls back to `ExpandSNIName` if SNI is a bare label
- Has a 1-minute internal timeout per invocation
- Returns cached cert if still valid; triggers ACME renewal if not

### Pattern 2: FQDN Derivation (TLS-02)

**What:** `TailscaleHealth.Domain` is already populated by `checkHealth` in `tailscale.go`. This is `CertDomains[0]` from the daemon status. `BaseURL()` must return `https://<domain>:<port>` instead of `https://<ip>:<port>`.

**Example change to `server.go`:**
```go
// BEFORE (current)
func (ws *WebServer) BaseURL() string {
    host, port, _ := net.SplitHostPort(ln.Addr().String())
    return fmt.Sprintf("https://%s:%s", host, port)
}

// AFTER (Phase 15)
func (ws *WebServer) BaseURL() string {
    _, port, _ := net.SplitHostPort(ln.Addr().String())
    return fmt.Sprintf("https://%s:%s", ws.fqdn, port)
}
```

`ws.fqdn` is set at `Start()` time from `TailscaleHealth.Domain` passed in via `Config` or a new field.

**Constraint from STATE.md:** "FQDN: always derive from `lc.CertDomains(ctx)[0]`; zero hardcoded `.ts.net` strings in URL construction." Use the already-computed `TailscaleHealth.Domain` from the health system rather than calling the daemon again.

### Pattern 3: Tailscale-Only Interface Binding (TLS-03)

**What:** `StartWebServer` in `app.go` no longer accepts a `bindIP` param from the frontend. It calls `GetTailscaleStatus()` internally, checks `IP != ""`, and uses that IP. The server refuses to start if Tailscale is not connected.

**Example change to `app.go`:**
```go
func (a *App) StartWebServer(port int) error {
    h := a.GetTailscaleStatus()
    if h.IP == "" {
        return fmt.Errorf("Tailscale is not connected — cannot start web server")
    }
    if !h.HasCerts {
        return fmt.Errorf("Tailscale HTTPS certificates are not enabled")
    }
    // ... create WebServer with h.IP and h.Domain
}
```

Note: `StartWebServer(bindIP string, port int)` signature changes to `StartWebServer(port int)`. The `GetNetworkInterfaces()` method and frontend interface dropdown are removed (but that is CLEAN-01 in Phase 17 — do not delete the method in Phase 15, just stop using it for web server binding).

### Pattern 4: Certificate Transparency Disclosure (TLS-04)

**What:** Before the first Tailscale cert is issued, the user must acknowledge that their machine's FQDN will appear in public CT logs (crt.sh, Censys, etc.). This is a one-time modal/banner in the Settings UI.

**Implementation approach:**
- Persisted flag: `~/.config/agenthub/ct_disclosed` file (presence = acknowledged)
- New Wails-bound methods: `HasCTDisclosure() bool` and `AcknowledgeCTDisclosure()`
- Frontend: Settings panel shows CT disclosure banner before the "Start Server" button is enabled, if flag not set

**What CT logs reveal:** The FQDN (e.g., `kens-macbook.tail46d69a.ts.net`) is visible to anyone querying CT transparency logs. The cert contains only the domain, not the IP or owner identity (beyond domain name). This is the standard disclosure for any Let's Encrypt cert.

**Disclosure text (suggested):**
> "When you start the web server, Tailscale will provision a Let's Encrypt TLS certificate for your device's hostname (e.g., `hostname.ts.net`). This hostname will be permanently visible in public Certificate Transparency logs. This is normal and expected for any Let's Encrypt certificate."

### Pattern 5: WebServer Config Changes

The `Config` struct gains a `FQDN` field and drops nothing (backward compat for tests). The `ConfigDir` field can be removed since CA files are no longer written.

```go
// BEFORE
type Config struct {
    BindIP    string
    Port      int
    ConfigDir string  // used for CA cert files
}

// AFTER (Phase 15)
type Config struct {
    BindIP string  // must be Tailscale IP (100.64.x.x range)
    Port   int
    FQDN   string  // derived from CertDomains[0], used in BaseURL
}
```

`ConfigDir` removal is safe because `NewWebServer` no longer calls `LoadOrCreateCA`.

### Anti-Patterns to Avoid

- **Caching CertPair at startup:** `lc.CertPair(ctx, domain)` called once and stored in a field will serve stale/expired certs. Always use `GetCertificate` hook — it handles caching internally.
- **Hardcoding `.ts.net` in URL construction:** State machine decision. Use `ws.fqdn` field.
- **Calling daemon on every `BaseURL()` call:** Compute `fqdn` once at `Start()` time from the `Config.FQDN` field (which was set from `TailscaleHealth.Domain` before `Start()` is called).
- **Deleting network.go in this phase:** `network.go` cleanup is CLEAN-01 (Phase 17). Leave it in place.
- **Deleting auth.go/tokens.go in this phase:** Auth removal is Phase 16. Leave them in place.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ACME cert provisioning | Custom Let's Encrypt ACME client | `lc.GetCertificate` | Tailscale daemon handles ACME negotiation, DNS-01 challenge, renewal |
| Cert caching/rotation | In-process cert store | `lc.GetCertificate` internal cache | Daemon manages disk cache; GetCertificate returns valid cert or renews |
| FQDN discovery | DNS lookup, hostname parsing | `status.CertDomains[0]` from `lc.StatusWithoutPeers` | Already computed in `checkHealth`; authoritative from Tailscale control plane |
| Tailscale IP detection | `net.Interfaces()` CGNAT range scan | `status.TailscaleIPs[0]` from `lc.StatusWithoutPeers` | More reliable; `network.go::ListInterfaces()` is a heuristic fallback (Phase 17 cleans it) |

**Key insight:** The Tailscale daemon is a full ACME client. `GetCertificate` is a thin hook that delegates everything to it.

---

## Common Pitfalls

### Pitfall 1: SNI Required for GetCertificate
**What goes wrong:** Browser connects to `https://100.64.x.x:port` (IP address URL) — TLS handshake has no SNI, `GetCertificate` returns `errors.New("no SNI ServerName")`, connection fails.
**Why it happens:** `lc.GetCertificate` requires SNI (domain name in TLS ClientHello). IP-address URLs do not send SNI.
**How to avoid:** After Phase 15, `BaseURL()` must return the FQDN URL (e.g., `https://myhost.ts.net:7443`), never the IP URL. All QR codes and links must use the FQDN. This is why TLS-02 and TLS-03 are bundled — you can't bind to Tailscale IP with a domain cert unless the browser is told to connect via domain name.
**Warning signs:** `no SNI ServerName` errors in server logs; browser shows `ERR_SSL_PROTOCOL_ERROR`.

### Pitfall 2: Startup Before Tailscale Is Ready
**What goes wrong:** `StartWebServer` is called before Tailscale is connected; `GetCertificate` invoked on first connection fails because daemon has no cert; server silently serves TLS errors.
**Why it happens:** `Start()` itself doesn't fail — the TLS listener opens fine. The failure is deferred to first handshake.
**How to avoid:** Gate `StartWebServer` on `TailscaleHealth.IP != ""` AND `TailscaleHealth.HasCerts`. Return an error before creating the listener. The health poller (Phase 14) already provides this data.
**Warning signs:** Server starts but browser immediately shows certificate error.

### Pitfall 3: Test Infrastructure Breaks
**What goes wrong:** `server_test.go` calls `testServer()` which calls `NewWebServer` → old code called `LoadOrCreateCA` → now removed. Tests fail to compile.
**Why it happens:** The test helper assumed self-signed cert infrastructure.
**How to avoid:** `testServer()` must be updated to inject a fake `GetCertificate` function. The new `Config` needs a `GetCertificate` field (or equivalent) for testability, OR tests use a real self-signed cert directly without going through the Tailscale client. Best pattern: add `TLSConfig *tls.Config` override field to `Config` for testing.

**Recommended test approach:**
```go
// In tests, pass a pre-built self-signed tls.Config to bypass GetCertificate
cfg := webserver.Config{
    BindIP:    "127.0.0.1",
    Port:      0,
    FQDN:      "localhost",
    TLSConfig: selfSignedTLSConfigForTest(),  // new escape hatch
}
```

### Pitfall 4: CT Disclosure Flag Race
**What goes wrong:** User restarts the app, flag file exists, but they don't remember acknowledging. Or the flag file gets deleted by a clean install.
**Why it happens:** Purely file-system based state.
**How to avoid:** The disclosure is informational, not a security gate. If the flag is missing (new install, config wipe), show the disclosure again. If a cert already exists (file in Tailscale's cert cache), the disclosure is moot — already logged. The check is best-effort UX, not a blocker.

### Pitfall 5: Cert Files in Working Directory
**What goes wrong:** `kens--personal-macbook-air.tail46d69a.ts.net.crt` and `.key` files currently sit in the repo root (visible in git status). These must be gitignored, not committed.
**Why it happens:** Tailscale writes cert files to `$HOME/Library/Preferences/Tailscale/certs/` on macOS when provisioned via daemon. The files in the repo root are likely from local manual testing. They are NOT written by AgentHub code.
**How to avoid:** Add `*.ts.net.crt` and `*.ts.net.key` to `.gitignore`. TLS-05 removes the AgentHub CA files (`ca.crt`, `ca.key` in `~/.config/agenthub`), but does NOT touch Tailscale's own cert cache.

---

## Code Examples

Verified patterns from official sources (GOMODCACHE v1.96.3):

### Building the TLS Config for Tailscale Certs
```go
// Source: tailscale.com@v1.96.3/client/local/cert.go (GetCertificate method)
var lc local.Client // zero value; connects to existing tailscaled

tlsCfg := &tls.Config{
    GetCertificate: lc.GetCertificate,
    MinVersion:     tls.VersionTLS12,
}
addr := fmt.Sprintf("%s:%d", bindIP, port)  // bindIP = TailscaleHealth.IP
ln, err := tls.Listen("tcp", addr, tlsCfg)
```

### Updated Start() Signature Pattern
```go
// server.go Start() — after Phase 15
func (ws *WebServer) Start() error {
    tlsCfg := ws.config.TLSConfig  // test override
    if tlsCfg == nil {
        var lc local.Client
        tlsCfg = &tls.Config{
            GetCertificate: lc.GetCertificate,
            MinVersion:     tls.VersionTLS12,
        }
    }

    port := ws.config.Port
    addr := fmt.Sprintf("%s:%d", ws.config.BindIP, port)
    ln, err := tls.Listen("tcp", addr, tlsCfg)
    // ... fallback to :0 on EADDRINUSE as before
    ws.mu.Lock()
    ws.listener = ln
    ws.mu.Unlock()
    go http.Serve(ln, ws.mux)
    return nil
}
```

### app.go StartWebServer() — Tailscale-gated
```go
// app.go — Phase 15 StartWebServer
func (a *App) StartWebServer(port int) error {
    h := a.GetTailscaleStatus()  // existing method from Phase 14
    if !h.Connected {
        return fmt.Errorf("Tailscale is not connected")
    }
    if h.IP == "" {
        return fmt.Errorf("Tailscale IP not available")
    }
    if !h.HasCerts {
        return fmt.Errorf("Tailscale HTTPS certificates not enabled — enable in Tailscale admin")
    }

    // Stop running server
    a.mu.Lock()
    oldWS := a.webServer
    a.mu.Unlock()
    if oldWS != nil {
        _ = oldWS.Stop()
    }

    ws, err := webserver.NewWebServer(webserver.Config{
        BindIP: h.IP,
        Port:   port,
        FQDN:   h.Domain,
    }, a.manager)
    // ... SetSessionResolver, Start(), store in a.webServer
}
```

### CT Disclosure Persistence
```go
// In app.go
func ctDisclosurePath() string {
    return filepath.Join(configDir(), "ct_disclosed")
}

func (a *App) HasCTDisclosure() bool {
    _, err := os.Stat(ctDisclosurePath())
    return err == nil
}

func (a *App) AcknowledgeCTDisclosure() error {
    return os.WriteFile(ctDisclosurePath(), []byte("1"), 0600)
}
```

### Test Escape Hatch — Self-Signed TLS for Tests
```go
// server_test.go helper — bypasses GetCertificate
func selfSignedTLSForTest(t *testing.T) (*tls.Config, *http.Client) {
    t.Helper()
    caKey, caCert, caDER, _ := generateTestCA()
    bindIP := net.ParseIP("127.0.0.1")
    leafCert, _ := generateTestLeafCert(caKey, caCert, bindIP)
    tlsCfg := &tls.Config{
        Certificates: []tls.Certificate{leafCert},
        MinVersion:   tls.VersionTLS12,
    }
    pool := x509.NewCertPool()
    pool.AddCert(caCert)
    _ = caDER
    client := &http.Client{Transport: &http.Transport{
        TLSClientConfig: &tls.Config{RootCAs: pool},
    }}
    return tlsCfg, client
}
```

Note: The CA+leaf generation functions can be moved to a `_test.go` file or a `testutil` package since they're no longer needed in production code after TLS-05.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Self-signed CA + leaf cert | Tailscale Let's Encrypt cert via `GetCertificate` | Phase 15 | No browser warnings; no CA install step |
| `ConfigDir` for CA persistence | No cert files written by AgentHub | Phase 15 | Simpler Config; fewer file I/O failure modes |
| `BindIP` from frontend dropdown | `BindIP` hardcoded to `TailscaleHealth.IP` | Phase 15 | Eliminates misconfiguration; enforces Tailscale-only |
| IP-based BaseURL | FQDN-based BaseURL | Phase 15 | Required for SNI; enables valid Let's Encrypt cert |
| `StartWebServer(bindIP, port)` | `StartWebServer(port)` | Phase 15 | Callers no longer choose interface |

**Deprecated/outdated after Phase 15:**
- `GenerateCA()`, `LoadOrCreateCA()`, `GenerateLeafCert()`, `BuildTLSConfig()`, `ExportCACertPath()` — all in `tls.go`, all deleted
- `GetCACertPath()` Wails binding — no longer meaningful; remove from `app.go` and `App.js`
- `GetNetworkInterfaces()` usage in `SettingsPanel.tsx` for interface selection — stays in Go layer until CLEAN-01 (Phase 17), but Settings UI no longer uses it for server binding
- CA cert download instruction block in `SettingsPanel.tsx` — removed (no CA cert to install)

---

## Open Questions

1. **Config.TLSConfig escape hatch vs. separate NewWebServerForTest()**
   - What we know: Tests need a way to start the server without a live Tailscale daemon. Current `testServer()` helper must change.
   - What's unclear: Whether adding `TLSConfig *tls.Config` to `Config` is the cleanest approach vs. a `WithTLSConfig()` option function or keeping CA generation in `tls_test.go` and re-implementing locally there.
   - Recommendation: Add `TLSConfig *tls.Config` to `Config` as an override (nil in production, set in tests). This keeps the production path clean and tests fully functional without any daemon.

2. **Wails binding signature for StartWebServer**
   - What we know: Frontend currently calls `StartWebServer(selectedInterface, selectedPort)` from `SettingsPanel.tsx`. Changing to `StartWebServer(port)` requires updating the JS binding in `App.js` and the frontend call site.
   - What's unclear: Whether the planner should treat the Wails binding update as part of the same plan as the Go change, or separate.
   - Recommendation: Same plan — Wails binding is just renaming one export; do it atomically with the Go change.

3. **Port selection in Settings UI after Phase 15**
   - What we know: The interface dropdown is going away. The port input stays.
   - What's unclear: The full Settings UI redesign scope — that belongs to Phase 17 (CLEAN-03). Phase 15 should make the minimal change: remove the interface dropdown, add CT disclosure.
   - Recommendation: In Phase 15, hide/remove the interface dropdown in the UI only; full Settings cleanup deferred to Phase 17.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib) |
| Config file | none |
| Quick run command | `go test ./internal/webserver/... -count=1` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TLS-01 | Server uses `lc.GetCertificate` hook in TLS config | unit | `go test ./internal/webserver/... -run TestStart_UsesTailscaleCert -count=1` | ❌ Wave 0 |
| TLS-02 | `BaseURL()` returns FQDN URL, not IP | unit | `go test ./internal/webserver/... -run TestBaseURL_UsesFQDN -count=1` | ❌ Wave 0 |
| TLS-03 | `StartWebServer` fails if Tailscale IP unavailable | unit | `go test . -run TestStartWebServer_NoTailscale -count=1` | ❌ Wave 0 |
| TLS-03 | Web server binds to Tailscale IP | unit | `go test ./internal/webserver/... -run TestStart_BindsToTailscaleIP -count=1` | ❌ Wave 0 |
| TLS-04 | `HasCTDisclosure` returns false before acknowledgement | unit | `go test . -run TestCTDisclosure -count=1` | ❌ Wave 0 |
| TLS-04 | `AcknowledgeCTDisclosure` persists flag | unit | `go test . -run TestCTDisclosure -count=1` | ❌ Wave 0 |
| TLS-05 | CA+leaf generation functions do not exist | compile-time | `go build ./...` | via TLS-05 implementation |
| TLS-05 | No cert files written to ConfigDir | unit | `go test ./internal/webserver/... -run TestNewWebServer_NoCertFiles -count=1` | ❌ Wave 0 |

Note: TLS-01 and TLS-02 HTTPS browser verification require live Tailscale daemon — these are manual-only for the live check. Automated tests use the `TLSConfig` override in `Config` to validate structure without a daemon.

### Sampling Rate
- **Per task commit:** `go test ./internal/webserver/... -count=1`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/webserver/server_test.go` — update `testServer()` helper to use `TLSConfig` override; add `TestBaseURL_UsesFQDN`, `TestStart_BindsToTailscaleIP`, `TestNewWebServer_NoCertFiles`
- [ ] `app_test.go` — add `TestStartWebServer_NoTailscale`, `TestCTDisclosure_*`
- [ ] `internal/webserver/tls_test.go` — move CA/leaf test helpers to test-internal use or delete; `tls.go` itself is deleted so test file must be removed/rewritten

---

## Sources

### Primary (HIGH confidence)
- `tailscale.com@v1.96.3/client/local/cert.go` (read directly from GOMODCACHE) — `GetCertificate`, `CertPair`, `ExpandSNIName` APIs verified
- `tailscale.com@v1.96.3/client/local/local.go` (read directly from GOMODCACHE) — `Client` struct zero-value semantics, `StatusWithoutPeers`
- `tailscale.com@v1.96.3/ipn/ipnstate/ipnstate.go` (grepped directly from GOMODCACHE) — `CertDomains`, `TailscaleIPs`, `BackendState` fields
- `/Users/ken/dev/agenthub/internal/webserver/` (read directly) — current implementation: `tls.go`, `server.go`, `tailscale.go`, `network.go`, all test files
- `/Users/ken/dev/agenthub/app.go` (read directly) — `StartWebServer`, `GetTailscaleStatus`, `startHealthPoller`, Wails bindings
- `/Users/ken/dev/agenthub/.planning/STATE.md` (read directly) — locked decisions: cert pattern, FQDN derivation, binary size constraint

### Secondary (MEDIUM confidence)
- CT transparency disclosure patterns — standard practice for any Let's Encrypt cert; no special API needed, purely informational UX

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — `lc.GetCertificate` is documented as stable API in source; already in go.mod
- Architecture: HIGH — all patterns derived from reading actual source files, not documentation guesses
- Pitfalls: HIGH — SNI requirement verified in `cert.go` source; test breakage derived from reading actual test files

**Research date:** 2026-03-20
**Valid until:** 2026-06-20 (Tailscale LocalAPI is stable; cert.go API marked stable)
