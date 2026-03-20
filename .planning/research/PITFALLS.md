# Pitfalls Research

**Domain:** Tailscale-only networking — transitioning from self-signed TLS + password/token auth + generic VPN to Tailscale Let's Encrypt certs + Tailscale-only networking (Go/Wails + React desktop app)
**Researched:** 2026-03-20
**Confidence:** HIGH (critical pitfalls verified against official Tailscale docs, GitHub issues, and Go package documentation)

---

## Critical Pitfalls

Mistakes that cause broken connectivity, regression in security, or silent failure modes during and after this transition.

---

### Pitfall 1: Importing the Deprecated `tailscale.com/client/tailscale` Package

**What goes wrong:**
The `tailscale.com/client/tailscale` package is deprecated and marked "only intended for internal and transitional use." All its methods — `Status`, `GetCertificate`, `CertPair`, `ExpandSNIName` — delegate to the new `tailscale.com/client/local` package. Importing the old package will work today but Tailscale has signaled migration intent, and the package header explicitly says to use `tailscale.com/client/local` instead. Code written against the deprecated package will require migration again in a future Tailscale version, which is friction in a Go module that embeds Tailscale as a dep.

**Why it happens:**
Most existing blog posts, Stack Overflow answers, and pre-2024 examples use `tailscale.com/client/tailscale`. The new package path is only visible in the pkg.go.dev deprecation notice and is easy to miss.

**How to avoid:**
Use `tailscale.com/client/local` for all local daemon interactions from the start. The `LocalClient` struct lives there. The key methods are:
- `lc.Status(ctx)` — check BackendState
- `lc.GetCertificate(hi)` — TLS config callback
- `lc.CertPair(ctx, domain)` — get cert + key PEM bytes
- `lc.CertDomains(ctx)` — list of FQDN the daemon will cert for

For control plane (tailnet admin) API calls, use `tailscale.com/client/tailscale/v2` (separate concern from local daemon).

**Warning signs:**
- Import path is `tailscale.com/client/tailscale` (not `.../local`)
- pkg.go.dev shows "Deprecated" badge on any function being called
- `go vet` or the IDE surfaces deprecation warnings

**Phase to address:**
Health check implementation phase — establish the correct import path before writing any Tailscale-touching code; retrofit is painful if the wrong package spreads across multiple files.

---

### Pitfall 2: Calling `GetCertificate` Without Verifying Tailscale Is Running and HTTPS Is Enabled

**What goes wrong:**
`lc.GetCertificate(hi)` is designed as a `tls.Config.GetCertificate` callback and is called for every TLS handshake. If Tailscale daemon is not running, or if HTTPS certificates are not enabled in the tailnet admin console, this function returns an error. The Go `net/http` TLS stack logs the error and drops the handshake. Browser clients see "ERR_SSL_PROTOCOL_ERROR" with zero indication of what precondition is missing. The server appears broken.

Additionally, `lc.CertDomains(ctx)` returns nil when the daemon is not running — this is both a health signal and a guard condition. If you skip this check and directly wire `GetCertificate` into `tls.Config`, every failed handshake is an opaque error from the browser's perspective.

**Why it happens:**
Developers wire `GetCertificate` into their TLS config (correctly) but do not add startup checks that verify Tailscale's BackendState is "Running" AND that HTTPS certs are enabled. The server starts, listens, but fails silently on first connection from any browser.

**How to avoid:**
At server startup (before accepting connections), perform an ordered health check:
1. Call `lc.Status(ctx)` and verify `BackendState == "Running"` — if not, surface a modal to the user
2. Call `lc.CertDomains(ctx)` — if empty, HTTPS certs are not enabled in the tailnet; surface instructional modal
3. Only then start the HTTPS listener with `GetCertificate` in `tls.Config`

The health check should be non-blocking from the UI perspective: the Wails app can load normally, show a warning, and let the user follow instructions to fix Tailscale state. Do NOT block app startup on Tailscale being healthy.

**Warning signs:**
- Browser shows "ERR_SSL_PROTOCOL_ERROR" or TLS handshake errors when server is "running"
- Go logs show `GetCertificate: ...` errors with no corresponding user-visible message
- App starts normally but no web connections can be established

**Phase to address:**
Tailscale health check phase — implement the check-and-modal flow before wiring `GetCertificate` into the server.

---

### Pitfall 3: MagicDNS Not Enabled — Cert Provisioning Fails Silently

**What goes wrong:**
Tailscale HTTPS certificates require MagicDNS to be enabled first. Without MagicDNS, `tailscale cert` and `GetCertificate` both fail. The error message from the Tailscale daemon is not self-explanatory to a user who does not know what MagicDNS is. The dependency order is: MagicDNS enabled → HTTPS certs enabled → `GetCertificate` works. Skipping MagicDNS in the health check means users who have never configured their tailnet will see cert failures with no actionable guidance.

**Why it happens:**
The health check naively checks only "is Tailscale connected?" (BackendState == "Running") without verifying the tailnet-level setting for HTTPS certs. A device can be fully connected to a tailnet that has neither MagicDNS nor HTTPS certs configured.

**How to avoid:**
Use `lc.CertDomains(ctx)` as the definitive proxy check for "is HTTPS provisioning possible?" — an empty result means HTTPS is not enabled (which implies MagicDNS may also be absent). The instructional modal for this state should include both steps: (1) enable MagicDNS in tailnet admin, (2) enable HTTPS in tailnet admin. Do not separate these into two different modals; users must do both sequentially and benefit from seeing both instructions at once.

**Warning signs:**
- `CertDomains` returns empty slice despite BackendState being "Running"
- `GetCertificate` returns error containing "HTTPS not configured" or similar
- User reports "cert error" but Tailscale is shown as connected

**Phase to address:**
Tailscale health check phase — the check must distinguish at minimum three states: not installed, installed but not connected, connected but HTTPS not configured.

---

### Pitfall 4: Certificate Transparency Exposes Machine Names — Permanent and Public

**What goes wrong:**
Every Tailscale Let's Encrypt certificate is recorded in public Certificate Transparency (CT) logs. The machine's FQDN (`machinename.tailnet-name.ts.net`) is permanently and publicly visible to anyone querying CT logs (crt.sh, Google CT, etc.). This is not reversible — even if HTTPS is later disabled, the CT entry persists forever. For AgentHub specifically, if the machine name contains the user's personal name, company name, or otherwise sensitive identifier, that information is now permanently public.

Research confirms this is not theoretical: analysis found 464 real hostnames exposed by 312 Tailscale users via `tailscale cert`.

**Why it happens:**
Let's Encrypt uses Certificate Transparency as a required part of the ACME protocol for domain-validated (DV) certificates. There is no opt-out. Tailscale explicitly warns about this in their docs but many users miss or skip the warning.

**How to avoid:**
Display a one-time warning in the instructional modal: "Your machine's Tailscale hostname will be permanently recorded in public Certificate Transparency logs. If your machine is named using your real name, company name, or other sensitive identifier, consider renaming it before enabling HTTPS." This warning should appear before the first cert provisioning attempt, not after.

This is a user education concern, not a code concern. The app cannot prevent CT disclosure — it can only ensure the user makes an informed decision.

**Warning signs:**
- No user-visible warning exists before enabling Tailscale HTTPS
- User later discovers their home PC's name (e.g., `kens-macbook.tail46d69a.ts.net`) is publicly searchable

**Phase to address:**
Tailscale health check / instructional modal phase — include the CT disclosure warning in the modal that guides users through HTTPS setup.

---

### Pitfall 5: Let's Encrypt Rate Limits Triggered by Repeated Cert Requests

**What goes wrong:**
Let's Encrypt enforces rate limits: 5 duplicate certificate requests per week for the same FQDN. If the server restarts frequently and `GetCertificate` is not properly caching, or if the cert validation check on startup has a bug that causes the daemon to treat every startup as a "need new cert" event, Let's Encrypt will return 429 errors. Once rate-limited, the machine cannot get a valid cert for up to 34 hours.

There is a documented bug in older Tailscale versions where the cert cache invalidation logic was wrong: even if a cert existed on disk and was valid, the daemon could repeatedly request renewals due to a validation function that always considered the cached cert invalid. This triggered Let's Encrypt rate limits under normal operation (issue #14690).

**Why it happens:**
`GetCertificate` is called on every TLS handshake. The Tailscale daemon handles caching and renewal internally, but bugs in the renewal heuristic have existed. In normal single-user desktop app use (few daily restarts, few connections), this is unlikely to trigger. But during development — many restarts, testing cert path — it can become a real problem quickly.

**How to avoid:**
- Use `GetCertificate` via the `LocalClient` (daemon-mediated) rather than calling `CertPair` on every startup and wiring it manually. The daemon handles caching; bypass the daemon cache and you own the caching problem.
- Ensure you are on a recent Tailscale version (post-fix for issue #14690 / #8725, fixed in v1.56+).
- Do not call `CertPair` on every TLS handshake — only use it if you are implementing file-based cert rotation (which is unnecessary in this case since `GetCertificate` handles the full lifecycle).

**Warning signs:**
- Tailscale daemon logs show "certificate: too many requests" or Let's Encrypt 429 responses
- `GetCertificate` errors that include "rate limit" strings
- Multiple restarts per day during development with cert checks on each startup

**Phase to address:**
Let's Encrypt cert integration phase — use `GetCertificate` via `LocalClient` and verify the daemon handles caching by inspecting logs for "async renewal" vs "serving cached" messages.

---

### Pitfall 6: Cert Renewal: `tailscale cert` File-Based Path Does NOT Auto-Renew

**What goes wrong:**
If you use `tailscale cert` CLI to obtain cert + key PEM files and load them as static files into the Go TLS config, you own renewal. Let's Encrypt certs expire in 90 days. The daemon does not know where you put the files and will not update them. The server will start serving an expired certificate, and all browser connections will fail with cert expiry errors — with no warning until the cert actually expires.

This is the wrong pattern for AgentHub. File-based certs are for nginx/Caddy use cases where the web server manages cert files separately from the daemon.

**Why it happens:**
Documentation for `tailscale cert` (the CLI command) is prominent and feels like the natural way to get certs. `GetCertificate` via the Go API is the correct pattern for in-process Go servers but requires knowing it exists.

**How to avoid:**
Use `lc.GetCertificate` wired into `tls.Config.GetCertificate`. This delegates certificate lifecycle entirely to the Tailscale daemon, which handles caching and renewal automatically. Never load cert PEM from a file obtained via `tailscale cert` into the Go server directly.

```go
tlsConfig := &tls.Config{
    GetCertificate: lc.GetCertificate,
}
```

This is the correct and complete integration. The daemon fetches, caches, and renews — the server code never touches cert bytes directly.

**Warning signs:**
- Code opens cert/key PEM files from disk and loads them into `tls.Config{Certificates: [...]}`
- Code calls `lc.CertPair()` on every startup and caches the result in a variable
- No test plan for "what happens 88 days after launch?"

**Phase to address:**
Let's Encrypt cert integration phase — the implementation must use `GetCertificate` callback, not static cert loading.

---

### Pitfall 7: Removing Password Auth Leaves Sessions Accessible to All Tailnet Members

**What goes wrong:**
The v1.1 password protects the web dashboard and per-session tokens protect individual terminal sessions. Removing password auth means any device on the user's tailnet — which can include family members, work colleagues, or shared devices depending on how the tailnet is configured — can access all terminal sessions directly by visiting the AgentHub URL. The Tailscale network-level authentication (WireGuard identity) is the only remaining gate.

For most single-person tailnets this is the intended behavior. For shared tailnets (family, team), this is a significant regression in access control that the user may not expect.

**Why it happens:**
Tailscale's security model assumes that tailnet membership itself is sufficient access control. This is true for single-user tailnets. But tailnets can be shared, and Tailscale's own docs note that ACLs do not affect local network access — only tailnet-to-tailnet routing.

**How to avoid:**
Include a one-time disclosure in the health check modal: "AgentHub will be accessible to all devices on your Tailscale network. If you share your Tailscale network with others, anyone on it can access your terminal sessions." This is user education, not a code change.

Optionally defer: if a future milestone wants per-device ACLs, Tailscale supports them via ACL policy files — but that is out of scope for v1.2.

**Warning signs:**
- No user-visible disclosure that "all tailnet members can access sessions"
- Family/team member reports accessing sessions they did not expect to be able to access

**Phase to address:**
Auth removal phase — the informational modal about what is being removed should include the tailnet-sharing disclosure.

---

### Pitfall 8: Hardcoded Tailscale FQDN Assumptions Breaking on Non-Standard Tailnets

**What goes wrong:**
Tailscale FQDNs follow the pattern `<machinename>.<tailnet-dns-name>.ts.net`. Code that constructs the FQDN by string-concatenation (e.g., `hostname + ".ts.net"`) breaks on:
- Tailnets with custom DNS names (not the default `ts.net` suffix)
- Funnel-enabled nodes (different DNS namespace)
- Older tailnets with legacy naming

If the constructed FQDN does not match what the daemon reports via `CertDomains`, the TLS server will answer connections on the wrong hostname and browsers will see a cert name mismatch error.

**Why it happens:**
`ts.net` looks like the canonical suffix. The daemon can report a different FQDN. Developers construct URLs for the QR code and share links by hardcoding the suffix instead of querying the daemon.

**How to avoid:**
Never construct the Tailscale FQDN by string manipulation. Always derive it from `lc.CertDomains(ctx)` — the first element is the canonical FQDN for this node. Use this same value for:
- The TLS server name (SNI check)
- The URL shown in the UI and QR code
- The shareable link

```go
domains, err := lc.CertDomains(ctx)
if err != nil || len(domains) == 0 {
    // not configured — show health check modal
}
fqdn := domains[0] // canonical FQDN, use everywhere
```

**Warning signs:**
- URLs shown in UI contain `.ts.net` suffix hardcoded in source
- QR code URL and actual served cert have different hostnames
- cert name mismatch browser errors (`ERR_CERT_COMMON_NAME_INVALID`)

**Phase to address:**
Let's Encrypt cert integration phase AND URL generation / QR code generation — derive FQDN from daemon at all code sites.

---

### Pitfall 9: Health Check State Machine Is Incomplete — No Periodic Re-Check

**What goes wrong:**
A health check that runs once at startup and shows a modal is insufficient. Tailscale can disconnect after startup (network change, sleep/wake, logout). If the health check runs only at app launch, the server continues attempting to serve over Tailscale after the daemon has disconnected. Connections silently time out. The status bar shows a valid Tailscale URL that no longer resolves.

**Why it happens:**
Startup-only health checks are simpler to implement. Ongoing polling feels like over-engineering for a desktop app.

**How to avoid:**
Implement a lightweight background goroutine that polls `lc.Status(ctx)` every 30–60 seconds. On BackendState transition from "Running" to anything else, update the UI status indicator and optionally suppress the Tailscale URL from the status bar (replace with a "Tailscale disconnected" message). Re-check every interval and restore when BackendState returns to "Running".

This goroutine should be cancellable (using a context tied to app shutdown) and should not surface a modal for every transient disconnect — only after a sustained disconnect (2+ consecutive failed checks).

**Warning signs:**
- No goroutine in the codebase that polls Tailscale status after startup
- Status bar shows Tailscale URL even after manually stopping Tailscale daemon
- User shares a Tailscale URL that times out immediately because Tailscale is disconnected

**Phase to address:**
Tailscale health check phase — design the health check as a state machine from the beginning, not just a one-shot startup gate.

---

### Pitfall 10: WebSocket Connections Break During TLS Certificate Change (Migration Transition)

**What goes wrong:**
During the transition from self-signed TLS to Tailscale Let's Encrypt certs, any browser that has an existing WebSocket (`wss://`) connection to the old self-signed cert endpoint will not automatically reconnect to the new HTTPS endpoint. The browser caches the old CA trust (added by the user during v1.0 setup) and may reject the new cert or route to a stale URL. Users with an open browser tab on the web dashboard during the update will lose their connection with no automatic reconnect.

**Why it happens:**
Changing the TLS certificate (CA, domain, hostname) is not transparent to existing WebSocket clients. The client has a connection to `wss://192.168.x.x` or `wss://agenthub.local` and after the update the server is at `wss://machinename.tailnet.ts.net`. These are different origins.

**How to avoid:**
This is a one-time migration concern. The handling is:
1. Accept that existing connections break at the moment of update — this is unavoidable
2. Ensure the frontend WebSocket reconnect logic handles connection loss gracefully with a user-visible "reconnecting..." state (already required for general robustness)
3. After the v1.2 update, the web dashboard URL changes from IP/self-signed to FQDN/LE cert — document this in the release notes

The deeper mitigation: ensure the frontend's WebSocket reconnect logic does not hard-code the old `wss://` URL. The URL should be derived dynamically from `window.location.host` or passed from the backend as part of the initial session metadata.

**Warning signs:**
- Frontend WebSocket URL is hardcoded to an IP address or `localhost`
- No "reconnecting..." UI state for WebSocket disconnects
- Web dashboard shows stale data after server restart with no reload prompt

**Phase to address:**
TLS migration / cert integration phase — verify the frontend derives the WebSocket URL dynamically and handles reconnects.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Run health check only at startup | Simple code | App shows stale "connected" state after Tailscale disconnects | Never — daemon can disconnect; add periodic polling |
| Construct Tailscale FQDN by string concatenation | No async call needed | Breaks on custom tailnet domains, Funnel, legacy tailnets | Never — always use `CertDomains()` |
| Show one combined "not ready" error | Less UX work | User can't tell whether Tailscale is missing, disconnected, or just HTTPS-unconfigured | Never — three distinct states need three distinct messages |
| Load cert from file via `tailscale cert` CLI output | Works today | Cert expires in 90 days with no auto-renewal | Never for a daemon-integrated Go server — use `GetCertificate` callback |
| Skip CT disclosure warning | Less UI friction | User's machine name permanently public in CT logs without their knowledge | Never |
| Import `tailscale.com/client/tailscale` (deprecated) | Works today | Migration required when deprecated package is removed | Only if already in codebase and refactor is too costly — document the debt |

---

## Integration Gotchas

Common mistakes when wiring Tailscale into the existing Go server.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| `tls.Config.GetCertificate` | Wire in without health check guard | Check `CertDomains()` before starting TLS listener; surface modal if empty |
| `lc.Status()` — BackendState | Check only for non-nil error | Check `Status.BackendState == "Running"` explicitly; error-free ≠ connected |
| Tailscale FQDN for URLs | Construct with `hostname + ".ts.net"` | Derive from `lc.CertDomains(ctx)[0]` always |
| VPN interface binding removal | Delete generic interface code first | First confirm Tailscale is the only interface in use; then remove generic code |
| Auth removal — token validation | Delete token middleware before removing token UI | Remove token middleware and token UI together; orphaned middleware is a security hole |
| WebSocket URL on frontend | Hardcode `wss://` + IP address | Derive from `window.location.host` or backend-provided session URL |

---

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Calling `lc.Status()` on every request to verify Tailscale is up | High IPC overhead on Go<→daemon socket | Cache status; recheck every 30–60s in background goroutine | > a few requests per second |
| Calling `lc.CertDomains()` on every TLS handshake | Redundant IPC; adds latency to every connection | Cache domains at startup; invalidate on health-check polling interval | Every TLS handshake |
| Polling health check at 1-second interval | Tailscale daemon IPC load, unnecessary wake-ups | Poll at 30–60 second intervals; only shorten on user-triggered check | Constant when app is open |

---

## Security Mistakes

Domain-specific security issues introduced by this migration.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Removing password auth without informing user that tailnet = access gate | Shared tailnet members silently gain full terminal access | Display disclosure in health check modal before auth is removed |
| Continuing to serve on all interfaces after switching to Tailscale-only | Traffic bypasses Tailscale if direct IP access is possible | Bind the HTTPS listener exclusively to the Tailscale interface IP (from `lc.Status`) not `0.0.0.0` |
| No check that `GetCertificate` domain matches the request's SNI | Wrong cert served if server is reached via non-Tailscale hostname | `GetCertificate` via `LocalClient` handles this; ensure custom cert code does not bypass it |
| Leaving self-signed CA trust instructions in documentation post-v1.2 | Users install unnecessary CA; creates trust confusion | Remove or archive all self-signed TLS onboarding docs after v1.2 ships |

---

## UX Pitfalls

Common user experience mistakes specific to the Tailscale networking migration.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Health check modal appears every app launch even after user has fixed Tailscale | Annoying; modal fatigue | Persist a "health acknowledged" flag; re-show only when state regresses |
| Instructional modal just says "Enable HTTPS in Tailscale" with no link | User has to search for where to do this | Include a direct link to `https://login.tailscale.com/admin/dns` with instructions |
| URL and QR code in status bar update before cert is ready | Browser shows cert error when user scans QR code immediately | Only show the Tailscale URL/QR code after `CertDomains()` returns a non-empty domain AND a test TLS handshake succeeds |
| No status differentiation: "Tailscale not installed" vs "not connected" vs "no HTTPS" | User tries wrong fix for each state | Three distinct health states: MISSING, DISCONNECTED, NO_HTTPS — each with its own message and action |
| Status bar shows Tailscale FQDN URL even when Tailscale is offline | User copies and shares a broken URL | Status bar should conditionally show Tailscale URL only when health check goroutine confirms "Running" |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Health check:** Distinguishes three states (not installed, disconnected, no HTTPS) — not just "connected vs not"
- [ ] **Cert provisioning:** Uses `lc.GetCertificate` callback in `tls.Config` — NOT static cert file loading
- [ ] **FQDN derivation:** All URL construction uses `CertDomains()[0]` — no hardcoded `.ts.net` suffix
- [ ] **Auth removal:** Both token middleware AND token UI removed together — no orphaned server-side token validation
- [ ] **Periodic health check:** Background goroutine polls status and updates UI — not startup-only check
- [ ] **Interface binding:** HTTPS listener bound to Tailscale IP only — not `0.0.0.0`
- [ ] **CT disclosure:** Warning shown to user before first cert provisioning attempt
- [ ] **WebSocket URL:** Frontend derives `wss://` URL dynamically — not hardcoded to IP or old self-signed hostname
- [ ] **Self-signed code removal:** CA generation, leaf cert generation, and CA install instructions all removed — no dead code that might accidentally be re-enabled

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Wrong import package (deprecated) | LOW | Search-replace import paths; regenerate if any API shape differences |
| `GetCertificate` errors at runtime — HTTPS not enabled | LOW | Show modal; user follows link to enable HTTPS in tailnet admin; health check passes on next poll |
| Let's Encrypt rate limited (429) | HIGH — up to 34 hour wait | Wait; do not re-request during rate limit window; add logging to detect early |
| CT disclosure missed — machine name exposed | NONE — irreversible | Document the disclosure warning; rename machine in Tailscale if possible (new cert CT entry, old one remains) |
| File-based cert expires (90 days) | MEDIUM | Switch to `GetCertificate` callback; redeploy; previous cert was expired and caused outage |
| Tailnet-sharing auth issue (unintended access) | MEDIUM | Add Tailscale ACL policy to restrict access to specific Tailscale users/tags; or add optional in-app PIN |
| FQDN hardcoded — cert mismatch on non-standard tailnet | LOW-MEDIUM | Replace hardcoded FQDN derivation with `CertDomains()`; deploy patch |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Deprecated package import | Health check / first Tailscale integration | All imports use `tailscale.com/client/local`; no `client/tailscale` in source |
| `GetCertificate` without precondition check | Health check phase | Server refuses to start HTTPS listener until `CertDomains()` is non-empty |
| MagicDNS prerequisite missing | Health check phase | Modal shown when `CertDomains()` is empty with MagicDNS + HTTPS setup instructions |
| CT disclosure | Health check / instructional modal | Warning text visible in modal before cert provisioning enabled |
| Rate limit from repeated cert requests | Cert integration phase | Verify daemon-mediated `GetCertificate` is used; logs show "cached" not "requesting new" |
| File-based cert with no auto-renewal | Cert integration phase | Code review confirms no `CertPair()` called at startup + stored; only `GetCertificate` callback |
| Auth removal — tailnet sharing disclosure | Auth removal phase | Informational text present in modal about tailnet access scope |
| Hardcoded FQDN | Cert integration + URL generation phase | Grep confirms no `.ts.net` string literals in URL-constructing code |
| Health check not periodic | Health check phase | Background goroutine present; disconnect Tailscale while app is open and verify UI updates |
| WebSocket URL static | Cert/URL migration phase | Disconnect old self-signed cert; WebSocket URL updates dynamically to new FQDN |
| Interface binding | Interface removal phase | `ss -tlnp` or netstat confirms server not listening on `0.0.0.0`; only on Tailscale IP |

---

## Sources

- [Tailscale Enabling HTTPS — official docs](https://tailscale.com/kb/1153/enabling-https) — prerequisites (MagicDNS required), CT disclosure, rate limits, 90-day expiry, "you own renewal" when using CLI certs
- [Tailscale TLS Certs blog post](https://tailscale.com/blog/tls-certs) — `GetCertificate` integration pattern for Go
- [tailscale.com/client/tailscale — pkg.go.dev](https://pkg.go.dev/tailscale.com/client/tailscale) — deprecation notice; confirmed all methods deprecated in favor of `tailscale.com/client/local`
- [tsnet package — pkg.go.dev](https://pkg.go.dev/tailscale.com/tsnet) — in-process Tailscale node; alternative to LocalClient for embedded use
- [GitHub issue #14690 — tsnet cert validation bug causing rate limits](https://github.com/tailscale/tailscale/issues/14690) — cached cert treated as invalid on every handshake → rate limit
- [GitHub issue #8725 — TLS cert not renewing before expiry](https://github.com/tailscale/tailscale/issues/8725) — confirmed bug fixed in PR #8731; renewal heuristic was wrong
- [GitHub issue #8204 — cert renewal uses hard-coded 14-day threshold](https://github.com/tailscale/tailscale/issues/8204) — improved to 2/3rds validity period heuristic
- [Analyzing public hostnames of Tailscale users (CT logs research)](https://iter.ca/post/hostnames/) — 464 real hostnames exposed; CT exposure is real, not theoretical
- [GitHub issue #12199 — FR: allowlist for HTTPS certs to avoid CT log leaks](https://github.com/tailscale/tailscale/issues/12199) — community discussion of CT concern; no resolution implemented
- [GitHub issue #15650 — Certificate transparency concern](https://github.com/tailscale/tailscale/issues/15650) — upstream acknowledgment
- [Tailscale best practices — security hardening](https://tailscale.com/kb/1196/security-hardening) — ACLs do not affect local network access; tailnet membership = access
- [ipnstate package — pkg.go.dev](https://pkg.go.dev/tailscale.com/ipn/ipnstate) — BackendState values: NoState, NeedsLogin, NeedsMachineAuth, Stopped, Starting, Running
- [Tailscale Serve docs](https://tailscale.com/docs/features/tailscale-serve) — interactive HTTPS enable prompt UX reference

---
*Pitfalls research for: v1.2 Tailscale-only networking — transitioning from self-signed TLS + password/token auth to Tailscale Let's Encrypt certs*
*Researched: 2026-03-20*
