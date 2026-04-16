---
phase: 74
slug: multi-client-fan-out
status: verified
threats_open: 0
asvs_level: 1
created: 2026-04-14
---

# Phase 74 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Client -> WebSocket handler | URL query params (?client=, ?readonly=) carry untrusted input from connecting clients | String (client name), boolean (readonly flag) |
| Client -> daemon API | GET /sessions returns viewerCount; no untrusted input in this direction | Integer (viewer count) |
| CLI -> relay WebSocket | CLI constructs ?readonly= and ?client= from user-supplied flags | String (client name), boolean (readonly flag) |
| Hub internal | Hub.go is internal infrastructure — no untrusted input crosses here | Subscriber metadata set at construction |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-74-01 | DoS | Hub.ResizeClient | accept | Max-wins arbiter naturally limits PTY resize syscalls — one resize per max-change event. Rate limiting deferred to server layer. | closed |
| T-74-02 | Tampering | Subscriber.ReadOnly | accept | ReadOnly is set at construction; enforcement in server read pump (`if !sub.ReadOnly` guard in relay/server.go:109 and webserver/server.go:447). | closed |
| T-74-03 | Tampering | client name query param | mitigate | `if len(clientName) > 64 { clientName = clientName[:64] }` in relay/server.go:49 and webserver/server.go:388. | closed |
| T-74-04 | Elevation of Privilege | readonly bypass | accept | Read-only is user convenience, not security boundary. Clients on the same Tailscale network can already connect without ?readonly=1. | closed |
| T-74-05 | Spoofing | client identity | accept | Client name is self-reported via ?client= param. No authentication — same trust model as existing WebSocket connections (Tailscale network membership). | closed |
| T-74-06 | Information Disclosure | viewerCount in API | accept | ViewerCount exposure is intentional per MC-04. Access requires local socket or Tailscale network access. | closed |
| T-74-07 | Tampering | --client flag value | mitigate | Server-side cap at 64 chars (T-74-03) is the enforcement point. CLI passes value through without additional validation. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-74-01 | ResizeClient called once per client resize event; max-wins limits syscall frequency. No amplification vector. | plan author | 2026-04-15 |
| AR-02 | T-74-02 | ReadOnly is convenience gating, not a security boundary. Server enforces in read pump. | plan author | 2026-04-15 |
| AR-03 | T-74-04 | Read-only is opt-in convenience. All clients on Tailscale network have full access by design. | plan author | 2026-04-15 |
| AR-04 | T-74-05 | Client identity is self-reported. No authentication needed — Tailscale network membership is the trust boundary. | plan author | 2026-04-15 |
| AR-05 | T-74-06 | ViewerCount is intentionally exposed per MC-04 for status bar and TUI. Daemon API access is already gated by network. | plan author | 2026-04-15 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-14 | 7 | 7 | 0 | gsd-secure-phase |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-14
