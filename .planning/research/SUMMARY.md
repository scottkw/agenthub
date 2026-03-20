# Project Research Summary

**Project:** AgentHub v1.2 — Tailscale-Only Networking
**Domain:** Go/Wails desktop app — networking and TLS architecture migration
**Researched:** 2026-03-20
**Confidence:** HIGH

## Executive Summary

AgentHub v1.2 is a targeted architectural migration of an existing Go/Wails desktop app: replacing self-signed TLS + password/token auth + generic VPN interface binding with Tailscale Let's Encrypt certs + Tailscale-only networking. The correct implementation pattern is narrow and well-documented. A single dependency addition (`tailscale.com@v1.96.3`) provides everything needed via `tailscale.com/client/local`: a zero-value `local.Client{}` struct queries the already-running `tailscaled` daemon via Unix socket for health status and certificate provisioning. No second Tailscale identity, no embedded daemon, no ACME code — the daemon handles all of it.

The recommended build order is strictly sequential because of hard safety dependencies: health check infrastructure must exist before TLS integration, and TLS integration using Tailscale IPs must be confirmed working before auth middleware is removed. Removing auth while still binding to `0.0.0.0` with self-signed certs would be a security regression. The deletion of dead code (`tls.go`, `auth.go`, `tokens.go`) comes last, after the compiler confirms all call sites are updated. This is a refactor-then-delete pattern, not a rewrite.

The primary risks are not technical — the Tailscale Go API is stable and well-documented with HIGH confidence. The risks are UX and operational: silent cert failures from missing health check preconditions, permanent exposure of machine hostnames in Certificate Transparency logs without user awareness, and the shift from explicit password auth to implicit tailnet-membership-as-access-control for users on shared tailnets. All three require user-facing disclosure copy in the health check modal, not code changes.

## Key Findings

### Recommended Stack

The entire Tailscale integration requires exactly one new `go get` command: `go get tailscale.com@v1.96.3`. Both packages needed (`tailscale.com/client/local` and `tailscale.com/ipn/ipnstate`) are in the same module. Everything else — TLS config, HTTP server, IP parsing — uses Go stdlib already in the codebase. The estimated binary size increase is 2–5 MB from the current ~15 MB baseline, acceptable for a desktop app but should be measured at the start of Phase 1.

**Core technologies:**
- `tailscale.com/client/local` (v1.96.3) — daemon health checks and cert provisioning; the current non-deprecated API; `local.Client{}` zero value works without configuration
- `tailscale.com/ipn/ipnstate` — typed status struct (`BackendState`, `CertDomains`, `TailscaleIPs`, `Self.DNSName`); pulled in transitively; no separate go.mod entry needed
- Go stdlib `crypto/tls` — `tls.Config{GetCertificate: lc.GetCertificate}` replaces the entire self-signed CA infrastructure with two lines
- Go stdlib `os/exec.LookPath` — used only as a pre-flight on platforms where the macOS App Store Tailscale variant may not put the CLI on PATH; primary check is a `StatusWithoutPeers` call

**What NOT to add:** `tailscale.com/tsnet` (embeds a full daemon — wrong model), `tailscale.com/client/tailscale` (deprecated package, all methods forward to `client/local`), `github.com/tailscale/tscert` (Caddy compatibility shim, missing health check methods), any auth or TLS cert generation library.

### Expected Features

The v1.2 feature set is a combination of net-new additions and deliberate deletions. All items are P1 — this is a self-contained milestone with a clear definition of done.

**Must have (table stakes):**
- Three-state Tailscale health check: daemon reachable → `BackendState == "Running"` → `len(CertDomains) > 0`
- Instructional modal with per-state failure guidance and "Check Again" button — three distinct failure messages, not one generic error
- `tls.Config.GetCertificate` wired to `local.Client.GetCertificate` — daemon handles cert caching and auto-renewal
- Auto-select Tailscale IP as bind address (`TailscaleIPs[0]`) — remove interface picker UI
- Auto-derive HTTPS URL from `strings.TrimSuffix(Status.Self.DNSName, ".")` — no user configuration required
- Delete `auth.go`, `tokens.go`, `tls.go`, and all auth middleware (`dashboardAuth`, `sessionAuth`)
- Remove routes: `POST /login`, `GET /ca.crt`, `POST /api/sessions/{id}/token`
- Replace interface picker in settings with Tailscale status indicator
- CT disclosure warning in modal before first cert provisioning attempt
- Tailnet-sharing access disclosure before auth removal is deployed

**Should have (differentiators — P2, add after validation):**
- Per-OS instructional text in health check modal (macOS menu bar / Linux CLI / Windows tray)
- Periodic health check goroutine (poll every 30–60s, update UI on disconnect)
- Health check re-run on web-serve toggle (not just at startup)

**Defer to v2+:**
- Tailscale Funnel integration (public internet access — different threat model, different scoping)
- Tailscale ACL-based per-session access control (only needed for multi-user tailnet sharing scenarios)

### Architecture Approach

The architecture introduces one new file (`internal/webserver/tailscale.go`) with a `TailscaleHealth` struct and `CheckHealth(ctx)` function, simplifies `server.go` and `app.go` by removing TLS/auth fields, and deletes four files entirely (`tls.go`, `auth.go`, `tokens.go`, and the current form of `network.go`). The `WebServer.Start()` signature changes to accept a `tailscaleIP string` obtained from the health check, and the `Config` struct collapses to just `Port int`. One new Wails-bound method, `GetTailscaleStatus()`, exposes `TailscaleHealth` to the React frontend for the status indicator and health modal.

**Major components:**
1. `internal/webserver/tailscale.go` (NEW) — owns all health check logic; `TailscaleHealth{Installed, Connected, HasCerts, IP, Domain}` returned from `CheckHealth(ctx)`; isolated and testable with no behavioral side effects on existing code
2. `internal/webserver/server.go` (MODIFIED) — `Config{Port int}` only; `WebServer` loses `caKey`/`caCert`/auth/tokens fields; `Start(tailscaleIP)` uses `tls.Config{GetCertificate: lc.GetCertificate}`; all auth middleware and auth routes removed
3. `app.go` (MODIFIED) — `StartWebServer(port int)` runs health check first, gates on all three conditions, passes IP from health result to `ws.Start()`; removes `SetWebPassword`, `GenerateSessionToken`, `GetCACertPath`, `GetNetworkInterfaces`; adds `GetTailscaleStatus()`
4. React health modal + status indicator (NEW) — three-state instructional UI driven by `TailscaleHealth` booleans; replaces settings interface picker with Tailscale status display
5. `tailscaled` (external, already running) — provides cert via Unix socket; app never touches cert bytes directly; handles ACME DNS-01 challenge transparently

### Critical Pitfalls

1. **`GetCertificate` without precondition check** — if `CertDomains` is empty when `GetCertificate` is wired into `tls.Config`, every TLS handshake fails silently with `ERR_SSL_PROTOCOL_ERROR`. Avoid by confirming `len(CertDomains) > 0` in the health check before starting the HTTPS listener; the modal must distinguish this state from "not connected."

2. **Machine hostname permanently in Certificate Transparency logs** — every `GetCertificate` call records the FQDN (`machinename.tailnet.ts.net`) in public CT logs permanently; this is not reversible. Avoid by displaying a one-time CT disclosure warning in the modal before the first cert provisioning attempt. This is a UI copy requirement, not a code change.

3. **Hardcoded Tailscale FQDN construction** — constructing `hostname + ".ts.net"` breaks on custom tailnet DNS names, Funnel-enabled nodes, and legacy tailnets. Always derive the FQDN from `lc.CertDomains(ctx)[0]`. Zero string-literal `.ts.net` in URL-constructing code.

4. **Removing auth before Tailscale IP binding is confirmed** — removing `dashboardAuth`/`sessionAuth` while the server still binds to `0.0.0.0` or uses self-signed certs removes the only access gate. Auth removal is only safe after Phase 2 (Tailscale TLS + IP binding) is verified working end-to-end.

5. **File-based cert via `tailscale cert` CLI or `CertPair` stored at startup** — loading cert PEM from disk means the app owns renewal; Let's Encrypt certs expire in 90 days with no automatic update. Use `lc.GetCertificate` as a `tls.Config` callback exclusively — the daemon handles caching and renewal. Never call `lc.CertPair()` at startup and cache the result.

## Implications for Roadmap

Based on research, the suggested phase structure is 5 sequential phases with one parallel track. All technical questions are fully resolved — no phases require additional pre-implementation research.

### Phase 1: Tailscale Health Check Infrastructure

**Rationale:** Health check logic is used by every subsequent phase. It can be built, tested, and deployed without touching existing TLS or auth code — zero regression risk. Adding `tailscale.com` to `go.mod` here also surfaces any binary size concern before the rest of the work proceeds.
**Delivers:** `internal/webserver/tailscale.go` with `TailscaleHealth` struct and `CheckHealth(ctx)`; new `GetTailscaleStatus()` Wails method; binary size delta measurement
**Addresses:** Three-state health detection (FEATURES.md table stakes); periodic health goroutine design (FEATURES.md P2)
**Avoids:** `GetCertificate` without precondition check (Pitfall 2); health check state machine incomplete — must be designed as a state machine from the start, not a one-shot startup gate (Pitfall 9)

### Phase 2: Tailscale TLS + Interface Binding

**Rationale:** This is the load-bearing change. Replacing `BuildTLSConfig` with `lc.GetCertificate` and binding to `TailscaleIPs[0]` instead of user-selected interface must be validated with a live integration test — cert provisioned, browser connects without warning — before auth is touched. `tls.go` is not deleted yet; it stays until integration confirms success.
**Delivers:** Modified `WebServer.Start(tailscaleIP)` using Tailscale cert; modified `app.go:StartWebServer()` calling `CheckHealth()` first; verified end-to-end HTTPS on `.ts.net` domain
**Uses:** `local.Client.GetCertificate` (STACK.md); `GetCertificate` as `tls.Config` callback pattern (ARCHITECTURE.md Pattern 1); `StatusWithoutPeers` for health checks (ARCHITECTURE.md Pattern 2)
**Avoids:** Binding to `0.0.0.0` (ARCHITECTURE.md Anti-Pattern 2); file-based cert with no auto-renewal (Pitfall 6); FQDN hardcoding (Pitfall 8)

### Phase 3: Auth Layer Removal

**Rationale:** Only safe after Phase 2 confirms the server is operating correctly on a Tailscale IP with Tailscale-issued certs. Auth middleware and token infrastructure must be removed together — orphaned server-side token validation with no corresponding UI is a security hole in the wrong direction.
**Delivers:** Deleted `auth.go`, `tokens.go`, auth middleware, `POST /login`, `GET /ca.crt`, `POST /api/sessions/{id}/token` routes; removed Wails methods `SetWebPassword`, `GenerateSessionToken`, `GetCACertPath`; modal disclosure about tailnet-sharing access scope
**Avoids:** Removing auth before IP binding is confirmed (dependency order from FEATURES.md); tailnet-sharing access regression without user disclosure (Pitfall 7)

### Phase 4: Dead Code Deletion + Config Simplification

**Rationale:** Files are deleted only after `go build ./...` passes without them — the compiler enforces completeness. This sequencing eliminates the risk of accidentally leaving dead code paths that could be re-enabled. `Config` struct collapses to `Port int`. `network.go` is replaced with a simplified `GetTailscaleIP` wrapper.
**Delivers:** Deleted `tls.go`, `tls_test.go`, `auth_test.go`, `tokens_test.go`; simplified `Config` struct; removed `ConfigDir`, `BindIP`, `GetNetworkInterfaces`; clean codebase with no dead code
**Implements:** Complete deletion phase from ARCHITECTURE.md build order (Phase 4 in the suggested build order)

### Phase 5: Frontend Health Modal + Status Indicator

**Rationale:** Can begin in parallel with Phases 3–4 once `GetTailscaleStatus()` is available from Phase 1. Completes the user-facing side of the migration: the health check logic is already in the backend, the frontend catches up here.
**Delivers:** React health check modal (three distinct instructional states, "Check Again" button, CT disclosure warning, tailnet-sharing disclosure); Tailscale status indicator in settings replacing interface picker; auto-derived HTTPS URL and QR code using `CertDomains()[0]`
**Addresses:** Instructional modal (FEATURES.md table stakes); auto-derive URL from DNSName (FEATURES.md table stakes); per-OS modal text after validation (FEATURES.md differentiator P2)
**Avoids:** Health check modal shown every app launch (UX pitfall — persist "health acknowledged" flag); WebSocket URL hardcoded to old self-signed hostname (Pitfall 10); status bar showing Tailscale URL while Tailscale is offline (UX pitfall)

### Phase Ordering Rationale

- Phases 1–4 are strictly sequential due to the safety dependency chain: health check → TLS integration → auth removal → deletion. Each phase can only proceed once the previous is verified.
- Phase 5 (frontend) can start after Phase 1 delivers `GetTailscaleStatus()`, running in parallel with Phases 3–4.
- The ARCHITECTURE.md "Suggested Build Order" directly maps to this phase structure — it was designed for safe incremental delivery with explicit verification gates.
- Each phase has a clear, independent verification step: Phase 1 (unit tests with mocked `local.Client`), Phase 2 (live integration test on dev machine — manual verification required), Phase 3 (confirm all routes open, no orphaned middleware), Phase 4 (clean `go build ./...`), Phase 5 (manual UAT on all three platforms).

### Research Flags

All phases have fully resolved technical questions — no phase requires `/gsd:research-phase` during planning. The research files contain exact API signatures, exact code samples, and architecture decision rationale.

One phase requires live manual verification that cannot be scripted:
- **Phase 2:** First real test against a live `tailscaled` daemon on the dev machine. Needs confirmation that the machine's tailnet has MagicDNS and HTTPS certs enabled before integration testing can proceed. This is a user-environment dependency, not a code complexity issue.

Phases with standard patterns (execution is straightforward, research is complete):
- **Phase 1:** Official API, canonical usage pattern, zero ambiguity in implementation
- **Phase 3:** Deletion work guided by compiler errors; no new API surface
- **Phase 4:** Compiler-guided deletion; `go build` is the verification
- **Phase 5:** React modal against a defined struct; all state values and modal branches are specified in FEATURES.md and ARCHITECTURE.md

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Official pkg.go.dev docs; latest stable version confirmed (v1.96.3 released 2026-03-19); Go version compatibility verified against existing go.mod; deprecated package alternatives confirmed and documented |
| Features | HIGH | Built from direct codebase inspection of all affected files plus official Tailscale API docs; feature dependency graph explicitly traced with safety ordering |
| Architecture | HIGH | Exact code samples verified against the official Tailscale `servetls` reference implementation; direct codebase inspection of all modified files (`server.go`, `tls.go`, `auth.go`, `tokens.go`, `network.go`, `app.go`) |
| Pitfalls | HIGH | Sourced from official Tailscale docs, confirmed GitHub issue bugs with fix versions (#14690, #8725, #8204), and independent Certificate Transparency log analysis with real data (464 hostnames exposed) |

**Overall confidence:** HIGH

### Gaps to Address

- **Binary size delta must be measured in Phase 1:** STACK.md estimates 2–5 MB increase, but `tailscale.com` is a large module. If the binary grows beyond an acceptable threshold (e.g., >25 MB), the documented fallback is `github.com/tailscale/tscert` for `GetCertificate` only plus raw HTTP-over-Unix-socket for health checks — this avoids pulling the full Tailscale dependency tree. Do not defer this measurement; it gates the rest of the work.

- **macOS App Store Tailscale variant behavior:** The App Store version may not put the `tailscale` CLI on PATH, making `exec.LookPath` unreliable as an "is installed" signal. The recommended mitigation — attempt `StatusWithoutPeers`; treat connection error as "not installed or not running" — is documented in STACK.md but should be verified during Phase 2 UAT on a macOS machine using the App Store variant.

- **Per-OS health modal copy:** The exact instructional text for macOS/Linux/Windows is deferred to P2 (add after basic modal works on all three platforms). The feature structure is fully specified; only the final copy and per-platform string values are TBD. Verify on all three platforms during Phase 5 UAT before marking P2 items complete.

## Sources

### Primary (HIGH confidence)
- [pkg.go.dev — tailscale.com/client/local](https://pkg.go.dev/tailscale.com/client/local) — Client methods, GetCertificate, CertPair, StatusWithoutPeers signatures; stable API markers present
- [pkg.go.dev — tailscale.com/ipn/ipnstate](https://pkg.go.dev/tailscale.com/ipn/ipnstate) — Status struct fields: BackendState, CertDomains, TailscaleIPs, Self.DNSName
- [Tailscale — Enabling HTTPS](https://tailscale.com/kb/1153/enabling-https) — MagicDNS prerequisite, CT disclosure, rate limits, 90-day cert expiry; official docs
- [Tailscale — TLS Certs blog](https://tailscale.com/blog/tls-certs) — GetCertificate integration pattern for Go; official Tailscale blog
- [GitHub — tailscale/tailscale servetls example](https://github.com/tailscale/tailscale/blob/main/client/tailscale/example/servetls/servetls.go) — canonical `tls.Config{GetCertificate: lc.GetCertificate}` usage
- [pkg.go.dev — tailscale.com versions](https://pkg.go.dev/tailscale.com?tab=versions) — v1.96.3 confirmed latest stable as of 2026-03-19
- [pkg.go.dev — tailscale.com/client/tailscale (deprecated)](https://pkg.go.dev/tailscale.com/client/tailscale) — deprecation confirmed; all methods forward to client/local
- Existing codebase: `internal/webserver/server.go`, `auth.go`, `tokens.go`, `tls.go`, `network.go`, `app.go`, `go.mod`

### Secondary (MEDIUM confidence)
- [GitHub issue #14690](https://github.com/tailscale/tailscale/issues/14690) — cert cache validation bug causing Let's Encrypt rate limits; fixed in v1.56+; use v1.96.3 to avoid
- [GitHub issue #8725](https://github.com/tailscale/tailscale/issues/8725) — TLS cert not renewing before expiry; fixed in PR #8731; resolved
- [GitHub issue #12199](https://github.com/tailscale/tailscale/issues/12199) — CT log disclosure feature request; community-confirmed concern; no Tailscale-side resolution implemented
- [github.com/tailscale/tscert](https://pkg.go.dev/github.com/tailscale/tscert) — minimal cert-only alternative; documented as fallback only if full tailscale.com module binary size is unacceptable
- [Tailscale security hardening](https://tailscale.com/kb/1196/security-hardening) — tailnet membership = access; ACLs context for shared tailnet use case

### Tertiary (MEDIUM-LOW confidence)
- [iter.ca — CT log hostname analysis](https://iter.ca/post/hostnames/) — 464 real hostnames exposed via tailscale cert; confirms CT exposure is real and not theoretical; independent researcher, not Tailscale official

---
*Research completed: 2026-03-20*
*Ready for roadmap: yes*
