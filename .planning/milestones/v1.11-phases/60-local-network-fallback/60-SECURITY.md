---
phase: 60
slug: local-network-fallback
status: verified
threats_open: 0
asvs_level: 1
created: 2026-04-09
---

# Phase 60 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| LAN network | Self-signed TLS between desktop app and LAN browsers | HTTP Basic Auth credentials, session data |
| Unix socket | Daemon API on user-owned socket (0600) | Local password, server control commands |
| Frontend ↔ Backend | Wails bindings between React UI and Go backend | Password display, mode state |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-60-01 | Spoofing | selfcert.go | accept | Self-signed TLS by design for local fallback; banner recommends Tailscale for trusted certs | closed |
| T-60-02 | Info Disclosure | api.go | mitigate | Unix socket 0600 permissions; threat model documented in code comment (WR-04, commit df7d419) | closed |
| T-60-03 | Tampering | auth.go | mitigate | HTTP Basic Auth over TLS encrypts credentials in transit; P256 cert with IP SAN | closed |
| T-60-04 | Tampering | WebSocket CORS | accept | Open OriginPatterns acceptable with Basic Auth + TLS; flagged for v1.12 hardening (RESEARCH.md) | closed |
| T-60-05 | Info | selfcert.go | accept | Missing SubjectKeyId/AuthorityKeyId — no behavioral impact, cosmetic standards gap | closed |

*Status: open / closed*
*Disposition: mitigate (implementation required) / accept (documented risk) / transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-60-01 | T-60-01 | Self-signed TLS is the intended design for local fallback when Tailscale is unavailable. Users are nudged via banner to install Tailscale. | Claude (security audit) | 2026-04-09 |
| AR-60-04 | T-60-04 | WebSocket open CORS is acceptable risk in local mode where Basic Auth + TLS provides authentication. Hardening deferred to v1.12. | Claude (security audit) | 2026-04-09 |
| AR-60-05 | T-60-05 | SubjectKeyIdentifier omission has no behavioral impact. Go crypto/tls and all major browsers accept the cert. Low priority improvement. | Claude (security audit) | 2026-04-09 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-09 | 5 | 5 | 0 | Claude (gsd-secure-phase) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-09
