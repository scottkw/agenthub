---
phase: 160
slug: v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-06-28
---

# Phase 160 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Generated in State B (no prior SECURITY.md; built from the 5 PLAN threat models +
SUMMARY threat flags). ASVS L1, block_on=high. All 5 PLANs carried a `<threat_model>`
block (`register_authored_at_plan_time: true`), so per the secure-phase short-circuit
the register was verified at L1 grep depth and the auditor subagent was not required
(threats_open: 0, asvs_level: 1).

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Hub client → local relay WS | Background unread listener connects to `ws://127.0.0.1:{relayPort}` (loopback only) | Read-only chat frames (count/mention only) |
| Client-side prop threading | Unread count + mention boolean threaded ChatPanel → HubPanel → SessionCard | In-process React state (no boundary crossed) |
| inject WS client → PTY | Untrusted inject text crosses into the PTY write path (IN-02 surface) | Sanitized inject text |
| remote tarball + checksums.txt → local FS | Downloaded release artifacts verified before install | Release binary + SHA256 checksums |
| chat content → rendered DOM | DCS/APC/PM/SOS body plaintext surviving sanitization rendered downstream | Sanitized chat plaintext |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-160-01 | Information Disclosure | useChatUnreadListeners WS | low | accept | Read-only loopback listener; sends no frames; reuses TerminalPanel's relay path. No new surface. | closed |
| T-160-02 | Denial of Service | per-session RelayClient leak | low | mitigate | Effect cleanup closes every RelayClient on unmount/session removal (`useChatUnreadListeners.ts:72-74`); gated on `isActive`. Test "closes every RelayClient on unmount". | closed |
| T-160-03 | Information Disclosure | unread badge content | low | accept | Badge exposes only count + mention boolean for locally-owned sessions; no message content on card. | closed |
| T-160-IN-02 | Tampering | HandleInject control-only text | medium | mitigate | `strings.TrimSpace(sanitized)==""` early-return guard at `hub.go:608`; pinned by `TestInject_ControlOnlyInput` (`server_inject_test.go:231`, asserts 0 PTY writes). | closed |
| T-160-WR-01 | Tampering | checksum line match (grep) | low | mitigate | `grep -F "${TARBALL}"` at `scripts/install.sh:77` makes tarball-name match exact (dots literal); SHA256 compare remains integrity gate. Test asserts `-F` present. | closed |
| T-160-WR-03 | Denial of Service | root install dir | low | mitigate | `mkdir -p "$INSTALL_DIR"` present in BOTH privilege branches of `scripts/install.sh` (grep count = 2); no privilege change. Test asserts count ≥ 2. | closed |
| T-160-IN-04 | Information Disclosure | SanitizeChatContent surviving body bytes | low | accept | Surviving plaintext neutralized by react-markdown + rehype-sanitize before render (`sanitize.go:147`); no HTML/script injection. Doc comment corrected to document residual. | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-160-01 | T-160-01 | Background unread listener is read-only over 127.0.0.1 loopback and reuses the existing relay path; introduces no new attack surface. | Phase 160 plan (160-01) | 2026-06-28 |
| AR-160-02 | T-160-03 | Hub card badge surfaces only an unread count + mention boolean for sessions the local owner already controls — no message content is exposed. | Phase 160 plan (160-02) | 2026-06-28 |
| AR-160-03 | T-160-IN-04 | DCS/APC/PM/SOS body bytes surviving ESC-introducer stripping are inert plaintext, neutralized by react-markdown + rehype-sanitize downstream; the doc comment now documents (not hides) this residual. | Phase 160 plan (160-05) | 2026-06-28 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-28 | 7 | 7 | 0 | Claude (secure-phase, L1 short-circuit) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-06-28
