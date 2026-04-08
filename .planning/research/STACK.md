# Stack Research

**Domain:** Desktop app (Go/Wails) — v1.11 new feature additions
**Researched:** 2026-04-08
**Confidence:** HIGH

## Context: What This Research Covers

This is a subsequent-milestone research file. The existing stack (Go/Wails v2, React,
xterm.js, nhooyr/websocket, go-pty, kardianos/service, tailscale.com/client/local) is
validated and not re-researched here. This file covers only what is NEW for v1.11:

1. Self-signed TLS cert generation (local network fallback)
2. Random password generation and display
3. Auto-serve on session creation
4. Claude Code native install path detection fix

---

## Feature 1: Self-Signed TLS Cert Generation

**Verdict:** No new library needed. Go standard library covers this completely.

### Required packages (all already imported in codebase)

| Package | Source | Already in codebase? |
|---------|--------|----------------------|
| `crypto/ecdsa` | Go stdlib | YES — `app_test.go`, `internal/webserver/server_test.go` |
| `crypto/elliptic` | Go stdlib | YES — `app_test.go` |
| `crypto/rand` | Go stdlib | YES — `internal/pty/native.go`, tests |
| `crypto/tls` | Go stdlib | YES — `internal/webserver/server.go` |
| `crypto/x509` | Go stdlib | YES — tests |
| `crypto/x509/pkix` | Go stdlib | YES — tests |
| `encoding/pem` | Go stdlib | YES — tests |
| `math/big` | Go stdlib | YES — tests |

### Why no external library

The test suite in `app_test.go` and `internal/webserver/server_test.go` already
generates self-signed ECDSA certs with the CA+leaf pattern using only stdlib.
The same code pattern moves into production in a new `internal/webserver/localnet.go`
file. No new dependency needed.

### Cert generation pattern (confirmed from existing test code)

```go
// ECDSA P-256 key, self-signed, SANs = machine LAN IPs
key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
tmpl := &x509.Certificate{
    SerialNumber: big.NewInt(1),
    Subject:      pkix.Name{CommonName: "AgentHub Local"},
    NotBefore:    time.Now(),
    NotAfter:     time.Now().Add(365 * 24 * time.Hour),
    KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
    ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
    IPAddresses:  localIPs, // from net.InterfaceAddrs()
}
certDER, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
tlsCfg := &tls.Config{
    Certificates: []tls.Certificate{{
        Certificate: [][]byte{certDER},
        PrivateKey:  key,
    }},
    MinVersion: tls.VersionTLS12,
}
```

### Integration point

`webserver.Config` already supports a `TLSConfig *tls.Config` override field (used for
tests). Local-network mode passes a locally-generated `*tls.Config` through this same
field. The `BindIP` field becomes `"0.0.0.0"` in local mode instead of the Tailscale
IP. `BaseURL()` returns `https://<first-LAN-IP>:<port>` in local mode.

No structural changes to `WebServer` are required.

---

## Feature 2: Random Password Generation

**Verdict:** No new library needed. `crypto/rand` is sufficient.

### Implementation

18 random bytes encoded as URL-safe base64 = 24 characters, ~143 bits of entropy.
Adequate for a temporary local-network session password.

```go
import (
    "crypto/rand"
    "encoding/base64"
)

func generatePassword() string {
    b := make([]byte, 18)
    if _, err := rand.Read(b); err != nil {
        panic("crypto/rand unavailable: " + err.Error())
    }
    return base64.URLEncoding.EncodeToString(b)
}
```

### Storage and display

The password lives only in the daemon's in-memory `API` state alongside `webServer`.
It is:
- Generated once when local-network mode starts (`handleWebServerStart`)
- Returned in `WebServerStartResponse` (new `Password string` field, `omitempty`)
- Exposed via `WebServerStatusResponse` so GUI can display it after restart
- Never written to disk — regenerated on each `POST /webserver/start`

### Auth middleware

Standard `net/http` middleware: check `Authorization: Bearer <password>` header.
Applied only when `webServer` is in local-network mode. Tailscale mode stays
unauthenticated (tailnet membership = access control).

```go
func passwordMiddleware(password string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Authorization") != "Bearer "+password {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

No external auth library. The codebase explicitly removed password auth in v1.2;
re-adding it as a minimal stdlib middleware is correct and intentional.

---

## Feature 3: Auto-Serve on Session Creation

**Verdict:** Pure logic change. No new stack additions.

### Current flow (manual)

1. `POST /sessions` creates session, returns ID
2. User separately calls `POST /webserver/start` if needed
3. User separately calls `POST /sessions/{id}/web-serve` with `{"enabled": true}`

### New flow (auto)

1. `POST /sessions` creates session, returns ID
2. In `handleCreateSession`: if `a.webServer != nil`, call `a.webServer.EnableSession(id)` immediately
3. Auto-start the web server: if no webServer is running, start it automatically
   (using the appropriate mode — Tailscale or local-network — depending on health state)

### Config impact

The `API` struct needs an `autoServe bool` field or the behavior is always-on. The
`WebServerStartRequest` (IPC type) may gain an `AutoServe bool` field so the GUI can
configure this at startup. No new library, no new IPC patterns — this is logic-only
within `internal/daemon/api.go`.

---

## Feature 4: Claude Code Native Install Path Detection

**Verdict:** Pure path augmentation change in `internal/daemon/path.go`. No new library.

### Problem

The Anthropic native installer places the `claude` binary at:

| Platform | Path |
|----------|------|
| macOS / Linux / WSL | `~/.local/bin/claude` |
| Windows | `%USERPROFILE%\.local\bin\claude.exe` |

**Source:** Official Anthropic docs — https://code.claude.com/docs/en/setup, uninstall
section explicitly says `rm -f ~/.local/bin/claude`. Verified on this machine: binary
exists at `/Users/ken/.local/bin/claude` and is the active `claude` in PATH.

The current `AugmentServicePath()` in `internal/daemon/path.go` prepends only:
- `~/.volta/bin`
- `/opt/homebrew/bin`
- `/usr/local/bin`
- `/home/linuxbrew/.linuxbrew/bin`
- nvm active bin

`~/.local/bin` is missing. When the daemon runs as a launchd service (or is launched
from Finder), the shell's PATH is not sourced, so `exec.LookPath("claude")` returns
`ErrCLINotFound` even when the native-installed binary is present.

### Fix — single insertion

```go
// In AugmentServicePath(), add as first candidate:
candidates := []string{
    filepath.Join(home, ".local", "bin"),   // Anthropic native installer (claude)
    filepath.Join(home, ".volta", "bin"),
    "/opt/homebrew/bin",
    "/usr/local/bin",
    "/home/linuxbrew/.linuxbrew/bin",
    nvmActiveBin(home),
}
```

`filepath.Join(home, ".local", "bin")` is correct on all three platforms
(`os.UserHomeDir()` returns `%USERPROFILE%` on Windows, which expands correctly).

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| External self-signed cert library | Subprocess dep, no API surface | Go stdlib `crypto/x509` |
| `crypto/md5` or `math/rand` for passwords | Not cryptographically secure | `crypto/rand` |
| External auth library (gorilla/sessions etc.) | 10,000+ LOC for a single Bearer check | stdlib `net/http` middleware (~10 lines) |
| Persistent cert storage (disk writes) | Complicates restart; unnecessary for local fallback | In-memory `tls.Config`, regenerate on each server start |
| Storing password in settings file | Plaintext credential on disk; overkill for session-scoped access | In-memory only, regenerate each server start |
| `net.InterfaceAddrs()` replacement library | Already in stdlib | `net.InterfaceAddrs()` directly |
| `tsnet` for local binding | Creates a second Tailscale node; wrong tool | Standard `net.Listen("tcp", "0.0.0.0:<port>")` |

---

## New Files

| File | Purpose |
|------|---------|
| `internal/webserver/localnet.go` | `GenerateLocalCert() *tls.Config`, `LocalLANIPs() []net.IP` |
| `internal/webserver/localnet_test.go` | Tests for cert generation and IP enumeration |

## Files Modified

| File | Change |
|------|--------|
| `internal/daemon/path.go` | Add `~/.local/bin` to `AugmentServicePath()` candidates |
| `internal/daemon/path_test.go` | Test coverage for `~/.local/bin` candidate |
| `internal/daemon/api.go` | Auto-serve logic in `handleCreateSession`; `Password` in `WebServerStartResponse` / `WebServerStatusResponse` |
| `internal/daemon/client.go` | Expose `Password` from `WebServerStatus()` response |
| `internal/daemon/types.go` | `WebServerStartRequest.Mode` field (tailscale vs localnet) |
| `internal/webserver/server.go` | Password auth middleware wired when in local-network mode |
| `app.go` | Wire local-network fallback to `StartWebServer` Wails binding |
| Frontend | Nudge banner (Tailscale not found), display local URL + password |

---

## Installation

No new `go get` commands. All new capabilities use Go standard library packages
already imported in this module.

```bash
# Confirm no new deps introduced:
go mod tidy
```

---

## Stack Patterns by Variant

**Tailscale healthy (installed + connected + certs enabled) — unchanged:**
- `local.Client{}.GetCertificate` hook for TLS
- Bind to Tailscale IP only
- No password, no nudge banner

**Tailscale not healthy — new local-network fallback:**
- `GenerateLocalCert()` from `internal/webserver/localnet.go` for TLS
- Bind to `0.0.0.0` (all LAN interfaces)
- Generate password via `crypto/rand`
- Show persistent nudge banner in GUI pointing to Tailscale onboarding
- Both modes share the same `WebServer` struct, same session enable/disable API

---

## Sources

- Official Anthropic Claude Code docs — https://code.claude.com/docs/en/setup
  Confirms native install path `~/.local/bin/claude` (macOS/Linux). HIGH confidence.
- Verified on local machine at `/Users/ken/.local/bin/claude`. HIGH confidence.
- Existing `app_test.go` and `internal/webserver/server_test.go` — confirm Go stdlib
  self-signed cert generation pattern works in this codebase. HIGH confidence.
- `internal/daemon/path.go` — source of the omission (no `~/.local/bin`). HIGH confidence.
- `internal/webserver/server.go` — `TLSConfig` override field already present and wired;
  confirms local cert injection approach requires no structural change. HIGH confidence.

---

*Stack research for: AgentHub v1.11 — local network fallback, auto-serve, Claude Code detection*
*Researched: 2026-04-08*
