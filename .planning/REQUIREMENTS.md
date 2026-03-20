# Requirements: AgentHub

**Defined:** 2026-03-20
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.2 Requirements

Requirements for Tailscale-Only Networking milestone. Each maps to roadmap phases.

### Tailscale Health

- [x] **HEALTH-01**: App detects whether Tailscale is installed on the system
- [x] **HEALTH-02**: App detects whether Tailscale is connected to a tailnet
- [x] **HEALTH-03**: App detects whether HTTPS certificates are enabled in the tailnet
- [ ] **HEALTH-04**: User sees a modal with clear, actionable instructions when any health check fails
- [ ] **HEALTH-05**: Modal instructions are platform-specific (macOS, Linux, Windows)
- [ ] **HEALTH-06**: Health checks run periodically in background; modal updates automatically when user resolves issues

### TLS & Certificates

- [ ] **TLS-01**: Web server uses Let's Encrypt certificates provisioned via Tailscale daemon
- [ ] **TLS-02**: Machine FQDN is derived from Tailscale daemon, not hardcoded
- [ ] **TLS-03**: Web server binds exclusively to the Tailscale network interface IP
- [ ] **TLS-04**: User is warned about Certificate Transparency log exposure before first cert provisioning
- [ ] **TLS-05**: Self-signed certificate infrastructure is removed (CA+leaf generation, cert files)

### Auth Removal

- [ ] **AUTH-01**: Password authentication is removed from the web dashboard
- [ ] **AUTH-02**: Per-session shareable tokens and links are removed
- [ ] **AUTH-03**: Web dashboard is accessible without authentication to any tailnet member

### Cleanup

- [ ] **CLEAN-01**: Generic VPN interface binding code is removed
- [ ] **CLEAN-02**: Auth middleware, token generation, and related backend routes are removed
- [ ] **CLEAN-03**: Settings UI for password, tokens, and VPN interface selection is removed

## Future Requirements

### Session Access Control

- **ACL-01**: Per-session access control for shared tailnets
- **ACL-02**: Tailscale ACL tag-based session visibility

## Out of Scope

| Feature | Reason |
|---------|--------|
| Port 443 for web server | User chose to keep current port; avoids conflicts with other services |
| tsnet embedded daemon | Creates second Tailscale node; AgentHub uses existing daemon via client/local |
| Tailscale installation by app | User handles Tailscale setup; app provides instructions only |
| Non-Tailscale VPN support | Milestone explicitly removes generic VPN in favor of Tailscale-only |
| Per-session token expiry | Tokens being removed entirely |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| HEALTH-01 | Phase 14 | Complete |
| HEALTH-02 | Phase 14 | Complete |
| HEALTH-03 | Phase 14 | Complete |
| HEALTH-04 | Phase 18 | Pending |
| HEALTH-05 | Phase 18 | Pending |
| HEALTH-06 | Phase 14 | Pending |
| TLS-01 | Phase 15 | Pending |
| TLS-02 | Phase 15 | Pending |
| TLS-03 | Phase 15 | Pending |
| TLS-04 | Phase 15 | Pending |
| TLS-05 | Phase 15 | Pending |
| AUTH-01 | Phase 16 | Pending |
| AUTH-02 | Phase 16 | Pending |
| AUTH-03 | Phase 16 | Pending |
| CLEAN-01 | Phase 17 | Pending |
| CLEAN-02 | Phase 17 | Pending |
| CLEAN-03 | Phase 17 | Pending |

**Coverage:**
- v1.2 requirements: 17 total
- Mapped to phases: 17
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-20*
*Last updated: 2026-03-20 — traceability complete after roadmap creation*
