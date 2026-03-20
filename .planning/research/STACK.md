# Stack Research

**Domain:** Tailscale-only networking for Go desktop app (v1.2 milestone)
**Researched:** 2026-03-20
**Confidence:** HIGH

## Context: What This Is NOT

This is a **subsequent-milestone delta document** for v1.2. The following are already validated and must NOT be re-evaluated:
- Go/Wails v2, React, xterm.js, nhooyr/websocket (now coder/websocket), go-pty
- Self-signed TLS infrastructure — being **REMOVED** in this milestone
- Password auth + per-session tokens — being **REMOVED** in this milestone
- Generic VPN interface binding — being **REMOVED** in this milestone

This file covers only the **net-new dependencies** for Tailscale-only networking.

---

## Recommended Stack (New Additions Only)

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `tailscale.com/client/local` | v1.96.3 | Query running Tailscale daemon: connection status, cert domains, machine DNS name, provision TLS certs | Official current API. Zero-value `local.Client{}` works without configuration — communicates via local Unix/named-pipe socket to the already-running `tailscaled`. No embedded daemon; no second Tailscale identity. |
| `tailscale.com/ipn/ipnstate` | v1.96.3 (same module) | Typed response structs for daemon status | `ipnstate.Status` contains `BackendState`, `CertDomains`, `Self.DNSName`, `TailscaleIPs` — all fields needed for health checks. Pulled in transitively with `client/local`; no separate import needed in go.mod. |

**Module:** Both packages live in the single `tailscale.com` module. Latest stable: **v1.96.3** (released 2026-03-19).

**One `go get` command:**
```bash
go get tailscale.com@v1.96.3
```

### No Other New Libraries Required

Everything else uses existing dependencies:
- TLS config: Go stdlib `crypto/tls` — `local.Client.GetCertificate` returns a `tls.Config.GetCertificate` callback directly
- HTTP/HTTPS server: existing Go `net/http` server already in the codebase
- "Is Tailscale installed" check: Go stdlib `os/exec.LookPath` — no library needed

---

## How Each Health Check Maps to a Specific API

### Check 1: Is Tailscale Installed?

```go
import "os/exec"

_, err := exec.LookPath("tailscale")
installed := err == nil
```

This is a pre-flight check before attempting any `local.Client` calls. The local client will return a connection error if `tailscaled` is not running, but that error is indistinguishable (by error type) from "not installed." LookPath distinguishes the two cases.

**Platform note:**
- macOS (App Store variant): `tailscale` CLI may not be in PATH. Use `exec.LookPath("tailscale")` first; if not found, check for the app at `/Applications/Tailscale.app`. Alternatively, attempt a `local.Client` call and treat the connection error as "not installed" with a message directing the user to tailscale.com.
- macOS (direct download / Homebrew): `tailscale` is in PATH at `/usr/local/bin/tailscale` or `/opt/homebrew/bin/tailscale`.
- Linux: `tailscale` is in PATH if installed.
- Windows: Tailscale installer adds the binary to PATH.

**Decision:** Attempt `local.Client.StatusWithoutPeers()`. If the call fails with a connection error, treat it as "Tailscale not running or not installed" and show the instructional modal. This is simpler than trying to distinguish installed-but-stopped from not-installed — both require the same user action (open Tailscale).

### Check 2: Is Tailscale Connected?

```go
lc := &local.Client{}
st, err := lc.StatusWithoutPeers(ctx)
if err != nil {
    // Tailscale daemon not reachable (not installed or not running)
    return ErrTailscaleNotRunning
}
// st.BackendState values:
//   "Running"          → connected to tailnet (proceed)
//   "NeedsLogin"       → installed+running, not authenticated
//   "NeedsMachineAuth" → awaiting admin approval
//   "Stopped"          → installed+running, paused by user
//   "Starting"         → daemon initializing
//   "NoState"          → daemon up, no profile configured
```

`StatusWithoutPeers` is preferred over `Status` — lighter response (no peer map allocation), sufficient for health checks.

### Check 3: Are HTTPS Certs Enabled?

```go
certsEnabled := len(st.CertDomains) > 0
```

`CertDomains` is the authoritative signal. It is populated by the Tailscale control plane when both:
1. MagicDNS is enabled in the admin console (tailscale.com/admin → DNS)
2. HTTPS certificates are enabled in the admin console (tailscale.com/admin → DNS → HTTPS)

If `CertDomains` is empty: show instructional modal pointing user to admin console. The machine's FQDN for constructing the web server address:

```go
// st.Self.DNSName is e.g. "my-machine.tail1234.ts.net." (trailing dot)
// Strip trailing dot before use as a hostname
hostname := strings.TrimSuffix(st.Self.DNSName, ".")
```

### Cert Provisioning for the HTTP/TLS Server

```go
lc := &local.Client{}

// Drop-in tls.Config callback — preferred approach
// Handles caching and auto-renewal; daemon manages ACME DNS-01 challenge
tlsConfig := &tls.Config{
    GetCertificate: lc.GetCertificate,
}

// Alternative: get raw PEM bytes (if cert needs inspection or disk writing)
certPEM, keyPEM, err := lc.CertPair(ctx, hostname)
```

`GetCertificate` is the correct choice for the existing Go HTTP server — it is a stable API, handles cert caching, and triggers renewal when the cert is close to expiry. No cert files to manage on disk. The daemon handles the ACME DNS-01 challenge via the Tailscale control plane; the app never sees ACME directly.

**Replacing self-signed TLS:** Remove the existing `generateSelfSignedCert` / CA+leaf infrastructure and replace the `tls.Config` construction with the above two lines. The `net.Listen("tcp", addr)` and `http.Server{TLSConfig: tlsConfig}` pattern is unchanged.

---

## Installation

```bash
go get tailscale.com@v1.96.3
```

**Expected binary size impact:** `tailscale.com/client/local` is a thin HTTP-over-Unix-socket client. It does NOT embed tsnet or the Tailscale daemon. The module pulls in Tailscale's internal types and some crypto, but the bulk of the Tailscale module (wireguard engine, DNS, routing) is not linked unless imported. Binary size increase is estimated at 2-5MB. Acceptable for a desktop app currently at ~15MB.

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `tailscale.com/client/local` | `tailscale.com/tsnet` | tsnet embeds a full Tailscale node inside the Go binary — correct for headless servers that need their own Tailscale identity and tailnet presence. Wrong for AgentHub: it is a desktop app that authenticates via the user's existing Tailscale installation. Adding tsnet would create a second Tailscale node on the user's tailnet, require separate auth, and massively bloat the binary. |
| `tailscale.com/client/local` | `github.com/tailscale/tscert` | tscert is a stripped-down cert-only shim maintained for Caddy's older-Go-version compatibility requirements. It lacks the status/health check methods needed for this milestone. No reason to use it when the full `client/local` is available. |
| `tailscale.com/client/local` | `tailscale.com/client/tailscale` (old package) | Deprecated. All methods on the old package delegate to `local.Client` internally. Official docs say to migrate to `client/local`. Do not use. |
| `local.Client.GetCertificate` | `exec.Command("tailscale", "cert", hostname)` subprocess | Running `tailscale cert` as a subprocess works but requires parsing stdout, error handling across platforms, no automatic renewal, and no in-process cert caching. The Go API is strictly better. |

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `tailscale.com/tsnet` | Embeds a full Tailscale daemon; creates a second tailnet node; wrong architectural model for a desktop app using an existing daemon | `tailscale.com/client/local` |
| `tailscale.com/client/tailscale` (old) | Deprecated; all methods forward to `local.Client`; wrong dependency to establish | `tailscale.com/client/local` |
| `github.com/tailscale/tscert` | Caddy compatibility shim; no status/health methods; `client/local` is the recommended path | `tailscale.com/client/local` |
| Any auth library (bcrypt, JWT, session tokens) | Auth is being removed entirely in v1.2 — Tailscale network membership is the access control | Remove existing auth; no replacement |
| Any TLS cert generation library | Self-signed cert infrastructure is being removed — Tailscale provides Let's Encrypt certs | Remove existing cert gen; use `local.Client.GetCertificate` |
| Any VPN interface detection library | Generic VPN binding is being removed; Tailscale-specific APIs replace it | `local.Client.StatusWithoutPeers` for IP/hostname |

---

## Stack Patterns by Health Check State

**If Tailscale daemon unreachable (StatusWithoutPeers returns error):**
- Show instructional modal: "AgentHub web sharing requires Tailscale. Download at tailscale.com."
- Disable all web serving controls in the UI
- Poll periodically (e.g., every 10s) and re-enable controls when daemon becomes available

**If BackendState != "Running" (e.g., NeedsLogin, Stopped):**
- Show instructional modal with state-specific message:
  - `NeedsLogin`: "Open Tailscale and sign in to your tailnet"
  - `Stopped`: "Tailscale is paused. Resume it to enable web sharing"
- Disable web serving controls

**If BackendState == "Running" but CertDomains is empty:**
- Show instructional modal: "Enable HTTPS in the Tailscale admin console: tailscale.com/admin → DNS → Enable HTTPS"
- Disable web serving controls (cannot provision certs without this)

**If all checks pass (Running + CertDomains non-empty):**
- Use `strings.TrimSuffix(st.Self.DNSName, ".")` as the server hostname
- Bind HTTP/TLS listener to `st.TailscaleIPs[0]` (Tailscale interface) on the desired port
- Configure `tls.Config{GetCertificate: lc.GetCertificate}` on the HTTP server
- The web serving URL is `https://<DNSName>:<port>/`

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `tailscale.com v1.96.3` | Go 1.26+ | go.mod already specifies `go 1.26.1`; no conflict |
| `tailscale.com v1.96.3` | `coder/websocket v1.8.14` | No conflict; different subsystems |
| `tailscale.com v1.96.3` | `wails/v2 v2.10.2` | No known conflicts; both use standard `net/http` |
| `local.Client` | tailscaled >= v1.20 | Local socket API stable for years; any recent Tailscale install is compatible |

---

## Sources

- [pkg.go.dev — tailscale.com/client/local](https://pkg.go.dev/tailscale.com/client/local) — Client methods, CertPair, GetCertificate, Status signatures. HIGH confidence, official documentation.
- [pkg.go.dev — tailscale.com/ipn/ipnstate](https://pkg.go.dev/tailscale.com/ipn/ipnstate) — Status struct fields: BackendState, CertDomains, Self.DNSName, TailscaleIPs. HIGH confidence, official documentation.
- [Tailscale Docs — Enabling HTTPS](https://tailscale.com/docs/how-to/set-up-https-certificates) — CertDomains requires MagicDNS + HTTPS enabled in admin console. HIGH confidence, official Tailscale documentation.
- [pkg.go.dev — tailscale.com versions](https://pkg.go.dev/tailscale.com?tab=versions) — v1.96.3 confirmed latest stable as of 2026-03-19. HIGH confidence, official.
- [pkg.go.dev — tailscale.com/client/tailscale (deprecated)](https://pkg.go.dev/tailscale.com/client/tailscale) — Confirmed deprecated; all methods redirect to `client/local`. HIGH confidence, official.
- [GitHub — tailscale/tscert](https://github.com/tailscale/tscert) — Confirmed as a Caddy compatibility shim, not for general use. MEDIUM confidence, official GitHub.
- [Tailscale Docs — tsnet](https://tailscale.com/kb/1244/tsnet) — Confirmed tsnet embeds a full daemon (wrong for desktop app use case). HIGH confidence, official Tailscale documentation.

---

*Stack research for: AgentHub v1.2 Tailscale-Only Networking*
*Researched: 2026-03-20*
