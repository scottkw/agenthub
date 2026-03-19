# Phase 4: Web Serving + TLS + Auth - Research

**Researched:** 2026-03-18
**Domain:** Go TLS/HTTPS, self-signed CA, password auth, token auth, network interface detection, xterm.js web serving
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| WEB-01 | User can toggle web serving on/off per session | WebServer struct with per-session `webEnabled` map + Wails-bound toggle methods |
| WEB-02 | Web-served sessions use self-signed TLS with local CA cert pattern | Go `crypto/x509` CA + leaf cert generation at startup; `tls.X509KeyPair` in-memory |
| WEB-03 | App provides in-app guidance for installing CA cert in OS trust store | Platform-specific commands documented; in-app UI renders instructions; export CA PEM to temp file |
| WEB-04 | Web dashboard lists all web-served sessions with password authentication | bcrypt hash stored in config; `http.Cookie`-based session after password POST; dashboard HTML served from `embed.FS` |
| WEB-05 | User can generate shareable token links for specific sessions | `crypto/rand` 32-byte opaque token; `sync.Map` token→sessionID; URL `/t/{token}` bypasses dashboard auth |
| WEB-06 | Remote browser connects to session via xterm.js over WebSocket (full interaction, not read-only) | Existing relay protocol (MsgInput/MsgOutput/MsgResize2) works for remote too; serve xterm.js HTML from `embed.FS`; WSS uses same coder/websocket with `OriginPatterns` set |
| NET-01 | User can bind web server to a specific network interface | `net.Listen("tcp", ip+":0")` with user-selected IP; interface list from `net.Interfaces()` |
| NET-02 | App auto-detects Tailscale interface via CGNAT range (100.64.0.0/10) | Scan `net.Interfaces()`, check each addr with `Contains(net.ParseIP(ip))` against `100.64.0.0/10` |
| NET-03 | User can select other VPN interfaces (WireGuard, etc.) from a dropdown | Enumerate all non-loopback, non-link-local unicast interfaces; present to user; same bind path as NET-01 |
</phase_requirements>

---

## Summary

Phase 4 adds a second HTTP server (HTTPS, user-facing) running inside the same Go process alongside the existing relay server (plain HTTP, localhost-only, Wails-internal). The web server must use TLS that browsers trust: not a bare self-signed leaf certificate but a CA-signed leaf certificate where the CA is installed in the OS trust store. The CA cert is generated at first launch and stored on disk; the leaf cert is generated per-launch in memory.

Authentication has two layers: a bcrypt-hashed dashboard password stored in a config file (single-user model per REQUIREMENTS.md), and per-session opaque tokens generated with `crypto/rand` that bypass the dashboard entirely. Network binding is user-controlled: enumerate all non-loopback interfaces with `net.Interfaces()`, auto-detect the Tailscale CGNAT range (100.64.0.0/10), and bind the HTTPS listener to the selected IP.

The web terminal page serves an HTML file from `embed.FS` that loads @xterm/xterm from jsDelivr CDN, opens a WSS connection to the same Go server at `/sessions/{id}/ws`, and uses the exact same binary framing protocol (`MsgInput`, `MsgOutput`, `MsgResize2`) already implemented in Phase 2.

**Primary recommendation:** Generate the CA cert once at first launch, store it under `~/.config/agenthub/` (or OS equivalent), derive the leaf cert from it in memory on every launch. The web dashboard and terminal page are static HTML embedded in the binary via `//go:embed`. Use bcrypt + cookie for dashboard auth; use `crypto/rand` opaque tokens for per-session links. All of this lives in a new `internal/webserver` package.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `crypto/tls` | stdlib | TLS config, `tls.X509KeyPair`, `tls.NewListener` | No external dep; in-memory cert loading |
| `crypto/x509` | stdlib | CA + leaf certificate generation, `x509.CreateCertificate` | Only standard way to create cert chains in Go |
| `crypto/ecdsa` | stdlib | ECDSA P-256 key generation | P-256 is constant-time, best browser compat, smaller than RSA |
| `golang.org/x/crypto/bcrypt` | already in go.sum (transitive) | Password hashing | OWASP recommended; already a transitive dep via Wails |
| `crypto/rand` | stdlib | Secure random token generation | Only correct source for auth tokens |
| `embed` | stdlib | Embed dashboard HTML + xterm.js page into binary | Single binary distribution, no file deployment |
| `net` | stdlib | `net.Interfaces()`, `net.Listen()` | Interface enumeration and binding |
| `net/http` | stdlib | HTTP/S mux and middleware | Already used for relay server |
| `github.com/coder/websocket` | v1.8.14 (already in go.mod) | WSS upgrade for web clients | Already in project; supports `OriginPatterns` for CORS |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os/user` | stdlib | Resolve config dir path | Finding `~/.config/agenthub/` cross-platform |
| `encoding/pem` | stdlib | PEM encode/decode cert material | Exporting CA cert for user installation |
| `math/big` | stdlib | `big.NewInt()` for cert serial numbers | Required by `x509.CreateCertificate` |
| `@xterm/xterm` | ^6.0.0 (jsDelivr CDN) | Terminal rendering in web page | Same version as desktop; no npm build step needed |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| In-process HTTPS server | Separate ttyd/ttys process | Separate process = separate port management, no shared session state, distribution complexity |
| Opaque random tokens | JWT | JWTs are overkill for single-user local app; opaque tokens are simpler, revokable by clearing the map |
| bcrypt for password | argon2id | bcrypt is already a transitive dep; argon2id would add a new dep for marginal benefit in a single-user app |
| `embed.FS` for web assets | Runtime file serving from disk | `embed.FS` keeps the binary self-contained; no extra deployment files |
| CDN xterm.js in web page | Bundled/compiled JS | CDN avoids a separate npm build pipeline for the web page; acceptable for local VPN-only access |

**Installation:**
```bash
# No new third-party dependencies needed.
# golang.org/x/crypto is already required (via Wails). Confirm it's in go.mod:
go get golang.org/x/crypto@latest
```

---

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── webserver/
│   ├── server.go        # WebServer struct, Start/Stop, route registration
│   ├── tls.go           # CA + leaf cert generation, persistence, tls.Config builder
│   ├── auth.go          # Password hash check, session cookie middleware, token middleware
│   ├── network.go       # net.Interfaces enumeration, Tailscale detection, bind-IP logic
│   ├── tokens.go        # Opaque token generation, token→sessionID map
│   └── server_test.go   # Integration tests
web/
├── dashboard.html       # go:embed target — dashboard page
└── terminal.html        # go:embed target — xterm.js terminal page
```

The existing `internal/relay/` package stays untouched. The new `internal/webserver` package depends on `internal/relay` (for HubManager access) but not vice versa.

### Pattern 1: Two-Listener Architecture

**What:** The existing relay listener (plain HTTP, 127.0.0.1:random) stays as-is for Wails. A new HTTPS listener is started separately, bound to the user's selected IP, on a fixed or random port.

**When to use:** Always. Do not change the relay listener to TLS — Wails calls it from localhost JavaScript which cannot use a custom CA.

```go
// Source: net/http stdlib pattern
// Relay server (existing, unchanged):
ln, _ := net.Listen("tcp", "127.0.0.1:0")
go http.Serve(ln, relayServer)

// Web server (new):
tlsLn, _ := tls.Listen("tcp", bindIP+":"+port, tlsCfg)
go webServer.Serve(tlsLn)
```

### Pattern 2: In-Memory CA + Leaf Cert Generation

**What:** CA cert is generated once, stored in `~/.config/agenthub/ca.crt` + `ca.key`. On each app launch, a leaf cert is signed by the CA in memory — never written to disk.

**Why CA pattern:** Browsers (Chrome, Firefox, Safari) silently reject bare self-signed certs for WebSocket connections even after user exception. A CA-based approach where the user installs the CA root once means the browser trusts all subsequent leaf certs without further prompts.

```go
// Source: https://shaneutt.com/blog/golang-ca-and-signed-cert-go/ + crypto/x509 stdlib

// CA generation (run once, persist to disk):
caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
caTemplate := &x509.Certificate{
    SerialNumber:          big.NewInt(1),
    Subject:               pkix.Name{Organization: []string{"AgentHub Local CA"}},
    NotBefore:             time.Now(),
    NotAfter:              time.Now().AddDate(10, 0, 0),
    IsCA:                  true,
    KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
    BasicConstraintsValid: true,
}
caDER, _ := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
// PEM-encode and write caKey, caDER to ~/.config/agenthub/

// Leaf cert (in-memory, every launch):
leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
leafTemplate := &x509.Certificate{
    SerialNumber: big.NewInt(2),
    Subject:      pkix.Name{CommonName: bindIP},
    IPAddresses:  []net.IP{net.ParseIP(bindIP)},
    NotBefore:    time.Now(),
    NotAfter:     time.Now().AddDate(1, 0, 0),
    KeyUsage:     x509.KeyUsageDigitalSignature,
    ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
}
caCert, _ := x509.ParseCertificate(caDER)
leafDER, _ := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)

// Build tls.Config:
tlsCert, _ := tls.X509KeyPair(
    pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}),
    marshalECDSAKey(leafKey), // x509.MarshalECPrivateKey + pem.EncodeToMemory
)
tlsCfg := &tls.Config{Certificates: []tls.Certificate{tlsCert}}
```

### Pattern 3: Auth Middleware Chain

**What:** Two middleware functions in front of session WebSocket handlers:
1. `dashboardAuth` — checks `agenthubd_session` cookie (set after password POST)
2. `tokenAuth` — checks `?token=` query param against token map; bypasses dashboard session

```go
// Source: net/http stdlib pattern

func (s *WebServer) sessionHandler(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    token := r.URL.Query().Get("token")

    if token != "" {
        // Token path — check token map
        mapped, ok := s.tokens.Lookup(token)
        if !ok || mapped != sessionID {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }
    } else {
        // Dashboard session cookie path
        if !s.isAuthenticated(r) {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
    }
    // Upgrade to WSS and relay...
}
```

### Pattern 4: Opaque Token Generation

```go
// Source: crypto/rand stdlib — Go 1.22+ rand.Text() pattern

func generateToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}
// Tokens stored in sync.Map[token] -> sessionID
// Revoked by deleting from map
```

### Pattern 5: Network Interface Enumeration and Tailscale Detection

```go
// Source: net stdlib

func ListInterfaces() ([]NetworkInterface, error) {
    ifaces, err := net.Interfaces()
    if err != nil {
        return nil, err
    }
    _, tailscaleCIDR, _ := net.ParseCIDR("100.64.0.0/10")

    var result []NetworkInterface
    for _, iface := range ifaces {
        if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
            continue
        }
        addrs, _ := iface.Addrs()
        for _, addr := range addrs {
            var ip net.IP
            switch v := addr.(type) {
            case *net.IPNet:
                ip = v.IP
            case *net.IPAddr:
                ip = v.IP
            }
            if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
                continue
            }
            isTailscale := tailscaleCIDR.Contains(ip)
            result = append(result, NetworkInterface{
                Name:        iface.Name,
                IP:          ip.String(),
                IsTailscale: isTailscale,
            })
        }
    }
    return result, nil
}
```

### Pattern 6: Embed Dashboard and Terminal HTML

```go
// Source: embed stdlib

//go:embed web
var webFS embed.FS

// Serve at /dashboard, /terminal/{id}
mux.Handle("/static/", http.FileServer(http.FS(webFS)))
```

The web terminal HTML page uses CDN xterm.js, connects via WSS using the same binary protocol from Phase 2:
```html
<!-- Source: xtermjs.org, jsDelivr CDN -->
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@6/css/xterm.css"/>
<script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@6/lib/xterm.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.11/lib/addon-fit.js"></script>
```

### Anti-Patterns to Avoid

- **Bare self-signed leaf cert:** Browsers reject these for WSS even with a user exception. Always use CA-signed leaf.
- **Writing leaf key to disk:** The CA key on disk is already a risk. Never write the leaf key; generate in memory each launch.
- **Reusing the relay listener for web traffic:** The Wails-internal relay runs plain HTTP on 127.0.0.1 and must stay that way. The web server is a separate listener on the VPN IP.
- **Calling `InsecureSkipVerify: true` on the WebSocket acceptor for the web server:** Phase 2 deferred CORS policy to Phase 4. Now that origins are known (the web server's own HTTPS base URL), use `OriginPatterns: []string{bindIP}` instead.
- **Storing passwords in plaintext:** Always bcrypt hash. Use `bcrypt.DefaultCost` (10) — sufficient for a single-user local app.
- **Using JWT for tokens:** Opaque tokens are simpler, immediately revocable (delete from map), and sufficient for this use case.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TLS cert chain validation | Custom x509 parser | `crypto/x509.CreateCertificate` + `tls.X509KeyPair` | Getting ASN.1 encoding wrong causes browser rejection |
| Secure random tokens | `math/rand` base10 IDs | `crypto/rand.Read` + `base64.RawURLEncoding` | `math/rand` is predictable; tokens must be cryptographically random |
| Password hashing | SHA-256 or MD5 of password | `golang.org/x/crypto/bcrypt` | bcrypt has built-in salt; SHA-256 is trivially rainbow-attacked |
| Session cookie generation | Sequential int IDs | `crypto/rand` 32-byte token | Sequential IDs allow session fixation |
| Network interface listing | Parsing `/proc/net/if_inet6` | `net.Interfaces()` + `iface.Addrs()` | stdlib is cross-platform; /proc only exists on Linux |
| xterm.js in-process bundling | npm build pipeline for web page | CDN xterm.js in HTML | Web terminal page is for local VPN access; CDN acceptable; avoids second build pipeline |

**Key insight:** The Go standard library handles all the hard crypto correctly. The surface area for mistakes is in the integration logic (two-listener architecture, CORS policy, cert chain assembly), not in any individual primitive.

---

## Common Pitfalls

### Pitfall 1: Leaf Cert Missing SAN (Subject Alternative Name)

**What goes wrong:** `x509.Certificate` has a `DNSNames` / `IPAddresses` field for SANs. If the leaf cert only sets `Subject.CommonName` but not `IPAddresses`, modern browsers (Chrome 58+, Firefox 66+) reject the cert with `ERR_CERT_COMMON_NAME_INVALID` even when the CA is trusted.

**Why it happens:** RFC 2818 deprecated CN-based hostname validation. All modern browsers now require SAN.

**How to avoid:** Always populate `IPAddresses: []net.IP{net.ParseIP(bindIP)}` in the leaf template. If the user might access the server by hostname, also add `DNSNames`.

**Warning signs:** Browser shows certificate error despite CA being installed; `openssl verify` passes but browser fails.

### Pitfall 2: CA Cert Not Marked as CA

**What goes wrong:** `IsCA: false` or `BasicConstraintsValid: false` in the CA template means the CA cert cannot sign the leaf cert — `x509.CreateCertificate` will succeed but browsers will reject the chain.

**How to avoid:** CA template must have `IsCA: true`, `BasicConstraintsValid: true`, and `KeyUsage` including `x509.KeyUsageCertSign`.

### Pitfall 3: WSS Connection Fails Because Relay Listener Is Upgraded to TLS

**What goes wrong:** If the developer tries to make the existing relay listener serve TLS, Wails's internal JavaScript (from `127.0.0.1`) will fail because the browser's WebView2/WebKit cannot verify a self-signed cert at `127.0.0.1` without the CA installed in the OS trust store, and even then the Wails WebView uses an embedded browser with different trust settings.

**How to avoid:** Two separate listeners, always. Relay stays plain HTTP on 127.0.0.1. Web server is TLS on VPN IP.

### Pitfall 4: macOS CA Trust Installation Requires sudo + User Confirmation

**What goes wrong:** `security add-trusted-cert -d -k /Library/Keychains/System.keychain ca.crt` requires `sudo`. On macOS Big Sur and later, even running as root is not sufficient — the system requires an explicit user confirmation dialog (as of 2024/2025, this is a deliberate Apple security hardening).

**Why it happens:** Apple added this restriction to prevent malicious apps from silently trusting their own certs.

**How to avoid:** Phase 4 should provide in-app instructions (WEB-03) that copy the CA cert to a file and show the user the exact command/manual step. The app should NOT try to auto-install — it will fail or prompt unexpectedly. Provide instructions for:
- **macOS:** "Open Keychain Access, drag this file in, double-click and set Trust to Always"  _or_ run `sudo security add-trusted-cert -d -k /Library/Keychains/System.keychain ~/agenthub-ca.crt` (still works but requires password prompt in terminal)
- **Linux:** Copy to trust dir + run update command (varies by distro — show the relevant command)
- **Windows:** `certutil -addstore root ca.crt` in an elevated command prompt

**Warning signs:** Silent failure, browser still shows cert warning after running the command.

### Pitfall 5: WebSocket CORS Rejected When Origin Doesn't Match Host

**What goes wrong:** `github.com/coder/websocket` rejects WebSocket upgrades by default if `Origin` header != `Host` header. The web terminal page served from `https://100.x.x.x:PORT/` will have `Origin: https://100.x.x.x:PORT`, which matches `Host`, so this is fine by default. But if the user accesses via a different URL or the port differs, it will fail with 403.

**How to avoid:** Replace `InsecureSkipVerify: true` in `server.go` with `OriginPatterns: []string{bindIP}` for the web server's WebSocket handler. The relay server's `InsecureSkipVerify: true` is correct to keep — it's localhost-only and serving Wails.

### Pitfall 6: Token URL Exposes Session ID in Logs

**What goes wrong:** URL-based tokens appear in server access logs and browser history. If the token is the session ID, that's worse — it also exposes internal identifiers.

**How to avoid:** The token must be a random opaque value that maps to a session ID internally (`tokens.Lookup(token) -> sessionID`). Never put the session ID in the token URL.

### Pitfall 7: go:embed Fails for Files Outside the Package Directory

**What goes wrong:** The Wails project has a known constraint: `//go:embed` cannot use `..` paths or symlinks. This was already hit in Phase 3 (assets_stub.go workaround). The web HTML files must live inside (or under) the package that declares the `//go:embed` directive.

**How to avoid:** Create a `web/` directory at the project root, and a `webembed` sub-package (or place the embed declaration in `internal/webserver/`) with the HTML files as subdirectory. Verify with `go build` before writing all the serving logic.

---

## Code Examples

Verified patterns from official sources:

### CA + Leaf Cert Generation (in-memory leaf, persisted CA)

```go
// Source: crypto/x509 stdlib, https://shaneutt.com/blog/golang-ca-and-signed-cert-go/

func generateCA() (*ecdsa.PrivateKey, *x509.Certificate, []byte, error) {
    key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, nil, nil, err
    }
    tmpl := &x509.Certificate{
        SerialNumber:          big.NewInt(1),
        Subject:               pkix.Name{Organization: []string{"AgentHub Local CA"}},
        NotBefore:             time.Now().Add(-time.Minute), // clock skew buffer
        NotAfter:              time.Now().AddDate(10, 0, 0),
        IsCA:                  true,
        KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
        BasicConstraintsValid: true,
    }
    der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
    if err != nil {
        return nil, nil, nil, err
    }
    cert, err := x509.ParseCertificate(der)
    if err != nil {
        return nil, nil, nil, err
    }
    return key, cert, der, nil
}

func generateLeafCert(caKey *ecdsa.PrivateKey, caCert *x509.Certificate, bindIP net.IP) (tls.Certificate, error) {
    leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return tls.Certificate{}, err
    }
    tmpl := &x509.Certificate{
        SerialNumber: big.NewInt(time.Now().UnixNano()),
        Subject:      pkix.Name{CommonName: bindIP.String()},
        IPAddresses:  []net.IP{bindIP},
        NotBefore:    time.Now().Add(-time.Minute),
        NotAfter:     time.Now().AddDate(1, 0, 0),
        KeyUsage:     x509.KeyUsageDigitalSignature,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
    }
    der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &leafKey.PublicKey, caKey)
    if err != nil {
        return tls.Certificate{}, err
    }
    keyDER, err := x509.MarshalECPrivateKey(leafKey)
    if err != nil {
        return tls.Certificate{}, err
    }
    certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
    return tls.X509KeyPair(certPEM, keyPEM)
}
```

### Start HTTPS Listener on Specific IP

```go
// Source: net/http + crypto/tls stdlib

func startWebServer(bindIP string, port int, handler http.Handler, tlsCert tls.Certificate) (net.Listener, error) {
    tlsCfg := &tls.Config{
        Certificates: []tls.Certificate{tlsCert},
        MinVersion:   tls.VersionTLS12,
    }
    addr := net.JoinHostPort(bindIP, strconv.Itoa(port))
    ln, err := tls.Listen("tcp", addr, tlsCfg)
    if err != nil {
        return nil, err
    }
    go http.Serve(ln, handler)
    return ln, nil
}
```

### bcrypt Password Verification Middleware

```go
// Source: golang.org/x/crypto/bcrypt

func (s *WebServer) checkPassword(plaintext string) bool {
    err := bcrypt.CompareHashAndPassword(s.passwordHash, []byte(plaintext))
    return err == nil
}

func hashPassword(plaintext string) ([]byte, error) {
    return bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
}
```

### Token Generation

```go
// Source: crypto/rand stdlib (Go 1.22+ rand.Text() equivalent)

func generateToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}
```

### Tailscale Detection

```go
// Source: net stdlib, Tailscale docs confirming 100.64.0.0/10

var tailscaleCIDR *net.IPNet

func init() {
    _, tailscaleCIDR, _ = net.ParseCIDR("100.64.0.0/10")
}

func IsTailscaleIP(ip net.IP) bool {
    return tailscaleCIDR.Contains(ip)
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Bare self-signed cert (CN only) | CA-signed leaf cert with SAN | Chrome 58 (2017), Firefox 66 (2019) | Old approach fails silently; CA pattern required |
| `math/rand` for tokens | `crypto/rand` + base64 | Go 1.0+ (always correct approach) | Predictable tokens = session hijack |
| SHA1 cert signatures | ECDSA P-256 or RSA-2048 | Browser deprecations 2016-2019 | SHA1 certs rejected by all modern browsers |
| HTTP Basic Auth | Cookie session + bcrypt hash | N/A — design choice | Basic Auth sends credentials on every request; cookie session sends once |
| `gorilla/websocket` | `github.com/coder/websocket` | Already decided in Phase 2 | nhooyr's websocket was adopted by Coder; maintained, better API |

**Deprecated/outdated:**
- Bare self-signed certs: Rejected by all modern browsers for WebSocket connections
- RSA-4096 for local certs: Excessive; ECDSA P-256 is faster and smaller with equivalent security
- `security add-trusted-cert` silent auto-install: Apple blocked this on macOS Big Sur+; must provide manual instructions

---

## CA Trust Installation: Per-Platform Reference

This is the critical blocker flagged in STATE.md. Here is the design decision:

**Do NOT auto-install the CA cert.** Provide in-app copy-to-clipboard instructions.

| Platform | Manual Step | Automated Fallback |
|----------|------------|-------------------|
| **macOS** | Keychain Access: drag cert in, double-click, set "Always Trust" | `sudo security add-trusted-cert -d -k /Library/Keychains/System.keychain [path]` — works but prompts for password in terminal |
| **Linux (Debian/Ubuntu)** | `sudo cp ca.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates` | Show dist-specific commands |
| **Linux (RHEL/Fedora)** | `sudo cp ca.crt /etc/pki/ca-trust/source/anchors/ && sudo update-ca-trust extract` | Show dist-specific commands |
| **Linux (Firefox)** | `certutil -A -n "AgentHub CA" -t "TCu,Cu,Tu" -i ca.crt -d sql:$HOME/.mozilla/firefox/*.default` | Show command |
| **Windows** | Run in elevated cmd: `certutil -addstore root ca.crt` | Can also try `CertAddEncodedCertificateToStore` via syscall (mkcert does this) |

WEB-03 specifically requires in-app guidance. The app should: (1) export CA cert to a temp file on button click, (2) display platform-appropriate instructions, (3) have a "copy command" button.

---

## Open Questions

1. **Port for HTTPS web server — fixed or random?**
   - What we know: A fixed port (e.g., 7443) is user-friendly for bookmarks and QR codes; random avoids conflicts.
   - What's unclear: Whether users will have port 7443 occupied by something else.
   - Recommendation: Start with a configurable port, default 7443, fall back to random if 7443 is taken. Store effective port in app state for URL display.

2. **CA cert regeneration policy — when to regenerate?**
   - What we know: The CA cert is stored on disk. If it expires (10 years) or is deleted, the app must regenerate and the user must reinstall it.
   - What's unclear: How to notify the user.
   - Recommendation: Check expiry at startup; if within 30 days of expiry or absent, regenerate and show a prompt.

3. **Dashboard password setup — first-run or in settings?**
   - What we know: There's no user account system; password is single-user.
   - What's unclear: Whether to prompt for password on first web-serve toggle or require setting it in advance in settings.
   - Recommendation: Require password to be set in Settings before web serving can be enabled. Gate the toggle with a check. This avoids an open dashboard.

4. **Windows CryptoAPI auto-install feasibility**
   - What we know: mkcert uses `CertAddEncodedCertificateToStore` via crypt32.dll, which works on Windows without certutil.
   - What's unclear: Whether a Wails app (not running elevated) can call this API.
   - Recommendation: Attempt Windows syscall install on button click; if it fails due to permissions, fall back to showing `certutil` instructions. Flag for Phase 6 validation.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` + `net/http/httptest` |
| Config file | none — Go tests use `go test ./...` |
| Quick run command | `go test ./internal/webserver/... -timeout 30s` |
| Full suite command | `go test ./... -timeout 60s` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WEB-01 | Toggle web serving on/off changes served session list | unit | `go test ./internal/webserver/... -run TestToggleWebServing` | Wave 0 |
| WEB-02 | Generated TLS cert is CA-signed with SAN, trusted by x509.Verify | unit | `go test ./internal/webserver/... -run TestTLSCertGeneration` | Wave 0 |
| WEB-03 | CA export produces valid PEM file, instructions rendered per platform | unit | `go test ./internal/webserver/... -run TestCAExport` | Wave 0 |
| WEB-04 | Dashboard returns 401 without auth, 200 after POST /login with correct password | unit | `go test ./internal/webserver/... -run TestDashboardAuth` | Wave 0 |
| WEB-05 | Token grants access to exactly one session; invalid token returns 401 | unit | `go test ./internal/webserver/... -run TestTokenAuth` | Wave 0 |
| WEB-06 | WSS connection with valid auth upgrades and receives MsgOutput frames | integration | `go test ./internal/webserver/... -run TestWebSocketRelay` | Wave 0 |
| NET-01 | Web server binds to specified IP, not 0.0.0.0 | unit | `go test ./internal/webserver/... -run TestBind` | Wave 0 |
| NET-02 | 100.64.x.x IP is classified as Tailscale in enumeration results | unit | `go test ./internal/webserver/... -run TestTailscaleDetection` | Wave 0 |
| NET-03 | All non-loopback, non-link-local interfaces appear in the list | unit | `go test ./internal/webserver/... -run TestInterfaceList` | Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/webserver/... -timeout 30s`
- **Per wave merge:** `go test ./... -timeout 60s`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/webserver/server_test.go` — covers WEB-01, WEB-04, WEB-06
- [ ] `internal/webserver/tls_test.go` — covers WEB-02, WEB-03
- [ ] `internal/webserver/auth_test.go` — covers WEB-04, WEB-05
- [ ] `internal/webserver/network_test.go` — covers NET-01, NET-02, NET-03

---

## Sources

### Primary (HIGH confidence)

- Go stdlib `crypto/x509`, `crypto/tls`, `crypto/ecdsa`, `crypto/rand`, `net`, `embed` — verified against pkg.go.dev
- `github.com/coder/websocket` v1.8.14 — `OriginPatterns` and `InsecureSkipVerify` fields verified at pkg.go.dev/github.com/coder/websocket
- https://shaneutt.com/blog/golang-ca-and-signed-cert-go/ — CA + leaf cert code patterns verified against crypto/x509 stdlib docs
- Tailscale CGNAT range 100.64.0.0/10 — verified at https://tailscale.com/docs/concepts/tailscale-ip-addresses
- `golang.org/x/crypto/bcrypt` — verified at pkg.go.dev/golang.org/x/crypto/bcrypt
- mkcert truststore_darwin.go / truststore_windows.go / truststore_linux.go source — confirms platform-specific trust install commands

### Secondary (MEDIUM confidence)

- macOS `security add-trusted-cert` requires sudo + user prompt on Big Sur+ — confirmed via Apple Developer Forums thread 671582 and Twocanoes blog
- jsDelivr CDN for @xterm/xterm@6 — verified at jsdelivr.com/package/npm/@xterm/xterm
- Go `embed.FS` constraint: cannot use `..` paths — already confirmed by Phase 3 implementation (assets_stub.go)

### Tertiary (LOW confidence)

- Windows `CertAddEncodedCertificateToStore` without elevation may work — mkcert uses this approach but Wails privilege context unverified; flag for Phase 6

---

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — all libraries are stdlib or already in go.mod
- TLS cert generation: HIGH — verified against official crypto/x509 docs and working mkcert source
- CA trust installation UX: MEDIUM — macOS Big Sur behavior confirmed; exact experience may vary by OS version; deliberate in-app guidance approach avoids automated failure
- Architecture (two listeners): HIGH — follows existing pattern from Phase 2/3; avoids Wails relay disruption
- Token auth: HIGH — standard `crypto/rand` + opaque token pattern; no external deps
- Windows auto-install: LOW — unverified whether Wails process can call CertAddEncodedCertificateToStore without elevation

**Research date:** 2026-03-18
**Valid until:** 2026-09-18 (stable domain — stdlib crypto APIs are stable; macOS behavior could shift with OS updates)
