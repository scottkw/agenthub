# Milestone v3.1 Requirements — Security Hardening

**Addresses:** GitHub Issue #35 ("Security review")
**Derived from:** Third-party security review (Codex) in `security-review/` — 5 findings, all confirmed against v3.0 code on 2026-04-19.
**Milestone goal:** Close the 5 confirmed findings so tailnet sharing is a real permission boundary, not an implicit trust fence.

---

## Active Requirements (v3.1)

### Session Authorization

Replaces the tailnet-wide trust model. Sessions become explicitly granted, and both metadata enumeration and PTY access require a server-issued capability.

- [ ] **SEC-01
** — User can explicitly grant share access to a specific session; the daemon issues a signed, server-scoped capability token bound to that session (no auto-exposure of newly created sessions when the web server is running)
- [ ] **SEC-02
** — `GET /api/sessions` rejects requests that do not carry a valid session capability (listing becomes capability-scoped, not tailnet-wide)
- [ ] **SEC-03
** — `GET /sessions/{id}/ws` and `GET /sessions/{id}` reject requests that do not carry a valid capability token for that exact session ID

### Read-Only Enforcement

Replaces the client-asserted `?readonly=1` query parameter with a server-bound permission on the capability token.

- [ ] **SEC-04
** — Read-only permission is a property of the capability token issued by the server, not a query parameter; the `?readonly=1` parameter (if retained as a view hint) cannot grant write access it lacks
- [ ] **SEC-05
** — The relay rejects `MsgInput` frames from any subscriber whose capability does not include write permission (previous bypass via reconnect-without-readonly is blocked by regression test)

### WebSocket Handshake Security

Removes the `OriginPatterns: ["*"]` / `InsecureSkipVerify: true` accept-all behavior and requires a short-lived handshake capability.

- [ ] **SEC-06** — WebSocket upgrade rejects requests whose `Origin` is not in the server allowlist (Tailscale FQDN, local-mode host URL, and configured same-origin), closing the cross-site WebSocket hijacking vector

### Frontend Supply Chain

Removes the runtime dependency on `cdn.jsdelivr.net` for the interactive terminal page.

- [ ] **SEC-07** — Terminal page loads xterm JavaScript and CSS from assets served by the embedded app binary (no `https://cdn.jsdelivr.net/...` references at runtime)
- [ ] **SEC-08** — Terminal page sets a `Content-Security-Policy` response header restricting `script-src` / `style-src` / `connect-src` to `self` plus the WebSocket origin

### Release Pipeline Hardening

Removes mutable-tag third-party code execution while release jobs hold signing and publish secrets.

- [ ] **SEC-09** — All third-party GitHub Actions in `.github/workflows/` are pinned to immutable commit SHAs (no `@main`, no `@master`, no floating branch refs)
- [ ] **SEC-10** — Go build tools used by workflows and `build.sh` (wails, nfpm, and any other `go install` targets) are pinned to exact versions (no `@latest`)
- [ ] **SEC-11** — Release pipeline is restructured so the unsigned build step cannot access signing, notarization, or publish secrets; signing/publish runs in a separate job that receives only the already-built artifacts

---

## Future Requirements

Work that naturally follows v3.1 but is out of scope for this milestone:

- **Capability rotation and revocation UI** — current model issues capabilities but lacks explicit revoke/rotate flows in the GUI/CLI/TUI
- **Per-user identity (not just capability tokens)** — would replace the capability model with authenticated users; deferred until multi-user demand emerges
- **Rate limiting on session enumeration and handshake endpoints** — useful hardening once capability model is stable
- **Audit log of session grants and capability issuance** — observability for security-sensitive actions
- **SLSA provenance attestation** for release artifacts — follow-on to release pipeline hardening
- **Third-party re-review of the capability design** — external audit after v3.1 ships

---

## Out of Scope

Explicit exclusions for v3.1 with reasoning:

- **Full SSO / OIDC user authentication** — capability tokens are sufficient for the single-user-desktop + shared-session threat model; SSO adds an auth server and database surface out of proportion with the use case
- **End-to-end encryption beyond TLS** — Tailscale Let's Encrypt TLS + capability-based authz covers the threat model; E2EE would require key exchange UX that undermines the zero-config value prop
- **Removing Tailscale network as a trust boundary entirely** — Tailscale membership remains a *reachability* gate; capabilities become the *access* gate. Defense in depth, not replacement
- **Rewriting the relay framing protocol** — the existing `MsgOutput/MsgInput/MsgResize/MsgMeta` protocol stays; only the handshake and per-frame authz behavior changes
- **Capability storage in an external vault / KMS** — tokens are symmetric-signed by the daemon using a key persisted alongside existing `settings.json`; external KMS is overkill for a local desktop app
- **Moving off `cdn.jsdelivr.net` for documentation-only pages** — only the interactive terminal page (which has command execution consequences) must be vendored; pure docs/landing pages are lower risk

---

## Traceability

Each v3.1 requirement maps to exactly one phase. All 11 requirements mapped (100% coverage, no orphans).

| REQ-ID | Phase | Status |
|--------|-------|--------|
| SEC-01 | Phase 87 — Capability-Based Session Authorization | Pending |
| SEC-02 | Phase 87 — Capability-Based Session Authorization | Pending |
| SEC-03 | Phase 87 — Capability-Based Session Authorization | Pending |
| SEC-04 | Phase 87 — Capability-Based Session Authorization | Pending |
| SEC-05 | Phase 87 — Capability-Based Session Authorization | Pending |
| SEC-06 | Phase 88 — WebSocket Handshake Security | Pending |
| SEC-07 | Phase 89 — Vendored Terminal Assets + CSP | Pending |
| SEC-08 | Phase 89 — Vendored Terminal Assets + CSP | Pending |
| SEC-09 | Phase 90 — Release Pipeline Hardening | Pending |
| SEC-10 | Phase 90 — Release Pipeline Hardening | Pending |
| SEC-11 | Phase 90 — Release Pipeline Hardening | Pending |

---

*Last updated: 2026-04-19 — Traceability filled by gsd-roadmapper; Phases 87-90 allocated for v3.1*
