---
phase: 161
slug: chat-sidebar-alias-control-user-can-set-their-display-name
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-28
---

# Phase 161 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (backend relay/webserver) + vitest (frontend ChatPanel/relayClient) |
| **Config file** | `frontend/vitest.config.ts`; Go uses standard `go test ./...` |
| **Quick run command** | `cd frontend && npx vitest run src/components/ChatPanel.test.tsx src/lib/relayClient.test.ts` |
| **Full suite command** | `go test ./internal/relay/... ./internal/webserver/... && cd frontend && npx vitest run && npx tsc --noEmit` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `{quick run command}`
- **After every plan wave:** Run `{full suite command}`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| {N}-01-01 | 01 | 1 | ALIAS-UI-01/02 | — | {expected secure behavior or "N/A"} | unit | `{command}` | ✅ / ❌ W0 | ⬜ pending |

*Planner fills concrete rows per task. Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Confirm existing tests cover the backend round-trip (`TestRelayIdentity_AliasPropagation`, `TestWebAliasPropagation`, `TestWebReadOnlyCanChat`).
- [ ] `frontend/src/lib/relayClient.test.ts` — add `sendAliasSet` send-side coverage (encoder already tested).
- [ ] `frontend/src/components/ChatPanel.test.tsx` — alias control render/validate/submit coverage.

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live cross-surface alias propagation (desktop owner ↔ web guest) | ALIAS-UI-02 | Requires live relay + real Tailnet web client; presence rebroadcast observed across two surfaces | Set alias from desktop ChatPanel; confirm web guest roster + new message author updates; repeat web→desktop |
| Web-share pre-fill shows correct self-identity | ALIAS-UI-01 (d) | Depends on live Tailnet computed name resolution | Open shared `/sessions/{id}` link as guest; confirm "chatting as" pre-fills the guest's computed name |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
