---
phase: 163
slug: read-only-guest-chat-posting-d-06-reconciliation
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-06-29
---

# Phase 163 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
>
> **Scope:** Reverses the SEC-01 RO chat-send gate (Phase 154) so RO-cap guests can post chat,
> while `@session` inject and PTY/terminal input remain RO-gated. This is the SEC-RO-01 review:
> proof that ONLY `HandleChatSend` was loosened. Register authored at plan time across
> 163-01/02/03 (`<threat_model>` blocks present in all three plans).

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| RO guest (web-share / Hub modal / desktop tab) → relay/webserver WS read-pump | Untrusted RO-cap holder submits MsgChatSend / MsgSessionInject / MsgInput frames | Chat content, inject text, keystrokes |
| chat processing → PTY stdin | Release-blocking line: chat must never reach `WriteInput`; only MsgSessionInject (RW-gated) may | PTY stdin bytes |
| RO guest browser → ChatPanel composer | Client-side affordance; authoritative gate is server-side (163-01). UI must not re-introduce a PTY-writing path for RO | Composer draft, inject gesture |
| documentation → future planners | Stale "RO cannot post" guidance could cause a future phase to re-introduce the gate, regressing D-06 | Planning prose |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-163-01 | Elevation of Privilege | hub.HandleChatSend / HandleInject / MsgInput dispatch | high | mitigate | ONLY `HandleChatSend` loosened. `internal/relay/hub.go:661-696` HandleChatSend body has no `sub.ReadOnly` branch (only doc comments reference it). `HandleInject` `if sub.ReadOnly { return ErrReadOnly }` gate intact at `hub.go:585-587`. MsgInput `if !sub.ReadOnly` discard intact at `relay/server.go:343` + `webserver/server.go:1190`. `ErrChatReadOnly` symbol fully removed from non-comment Go. Behavioral proof: `go test ./internal/relay/... ./internal/webserver/... -run 'ChatSend\|Inject\|ReadOnlyCapabilityBlocksMsgInput' -race` → ok (relay 1.99s, webserver 4.04s); `TestHandleChatSend_ROCanPostInjectStillGated` asserts RO chat=nil + inject=ErrReadOnly + PTY writes=0 atomically. | closed |
| T-163-02 | Denial of Service | RO chat-send abuse (spam/flood) | low | accept | RO posts traverse the SAME path as RW: `SanitizeChatContent` + ChatStore per-session 10k cap (`ErrChatCapReached`) + per-message size limit (`ErrChatMessageTooLarge`). No new abuse surface beyond existing RW clients; no new code. | closed |
| T-163-03 | Elevation of Privilege | ChatPanel handleInjectPointerDown | high | mitigate | Removing the Send-button RO gate did NOT remove the inject-gesture RO gate. `handleInjectPointerDown`'s `if (isReadOnly) return` retained at `frontend/src/components/Hub/ChatPanel.tsx:667` — defense-in-depth atop the server gate (163-01 HandleInject ErrReadOnly). Proven by vitest gesture test (ROCHAT-02: `sendSessionInject` not called for RO). | closed |
| T-163-04 | Spoofing | chat author identity | low | accept | Unchanged from Phase 152/154 — author identity is server-stamped (TailnetID/personKey); alias is display-only. This phase does not touch identity. | closed |
| T-163-05 | Repudiation | PITFALLS.md / TESTING.md drift | low | mitigate | PITFALLS.md reconciled to the D-06 rule and the flipped tests registered in TESTING.md (Section 2 delta + Section 4 ROCHAT-01/02/SEC-RO-01 rows); Phase 163 cited. Verify: `bash tests/check-traceability-paths.sh` → "OK: all traceability paths exist" (exit 0). | closed |
| T-163-06 | Tampering | npm/pip/cargo installs | low | accept | No package installs in this phase — docs + test-registration edits only; no dependency surface introduced. | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on (high) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-163-01 | T-163-02 | RO chat-send DoS shares the existing RW abuse surface (sanitize + 10k session cap + per-message size limit); no new attack surface. | Ken Scott | 2026-06-29 |
| AR-163-02 | T-163-04 | Author identity is server-stamped; alias is display-only. Phase does not touch identity. | Ken Scott | 2026-06-29 |
| AR-163-03 | T-163-06 | No dependency installs in this phase (docs + test edits only). | Ken Scott | 2026-06-29 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-29 | 6 | 6 | 0 | Claude (gsd-secure-phase, State B from artifacts, ASVS L1) |

L1 grep-depth verification (register authored at plan time, threats_open: 0, ASVS L1 → short-circuit per workflow Step 3). Both high-severity threats (T-163-01, T-163-03) verified by file:line evidence and passing `-race` tests; low-severity threats accepted with documented rationale.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-06-29
