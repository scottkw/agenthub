---
phase: 161
slug: chat-sidebar-alias-control-user-can-set-their-display-name
status: approved
nyquist_compliant: true
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
| 161-01-01 | 01 | 1 | ALIAS-UI-01 | T-161-05 | `MakeSelfFrame` emits byte `0x37` + JSON `{personKey,alias}` | unit | `go test ./internal/relay/ -run TestMakeSelfFrame -count=1` | ❌ W0 | ⬜ pending |
| 161-01-02 | 01 | 1 | ALIAS-UI-01 | T-161-05 | self-frame emitted on connect on both relay + webserver paths | unit | `go test ./internal/relay/... ./internal/webserver/... -run 'Self\|Identity\|Alias' -count=1` | ❌ W0 | ⬜ pending |
| 161-02-01 | 02 | 1 | ALIAS-UI-01, ALIAS-UI-02 | T-161-04 | `RelayClient.sendAliasSet` wraps existing `encodeAliasSetFrame` (no Wails binding) | unit | `npx vitest run src/lib/relayClient.test.ts` | ❌ W0 | ⬜ pending |
| 161-02-02 | 02 | 1 | ALIAS-UI-01 | T-161-05 | parse `MsgSelf` 0x37 → `onSelf(personKey,alias)`; tsc clean | unit | `npx vitest run src/lib/relayClient.test.ts && npx tsc --noEmit` | ❌ W0 | ⬜ pending |
| 161-03-01 | 03 | 2 | ALIAS-UI-02 | T-161-04 | `validateAlias` mirrors Go `ValidateAlias` (≤32 code points via `Array.from`, no C0/C1) | unit | `npx vitest run src/components/Hub/ChatPanel.test.tsx -t validateAlias` | ❌ W0 | ⬜ pending |
| 161-03-02 | 03 | 2 | ALIAS-UI-01, ALIAS-UI-02 | T-161-02, T-161-03 | header control renders + commits + pre-fills; stays enabled when `isReadOnly` (D-06); author renders as escaped text | unit | `npx vitest run src/components/Hub/ChatPanel.test.tsx && npx tsc --noEmit` | ❌ W0 | ⬜ pending |
| 161-04-01 | 04 | 3 | ALIAS-UI-01, ALIAS-UI-02 | — | cross-surface alias propagation (roster + future-message author) e2e | e2e | `npx playwright test e2e/chat-parity.spec.ts` | ❌ W0 | ⬜ pending |
| 161-04-02 | 04 | 3 | ALIAS-UI-01, ALIAS-UI-02 | — | TESTING.md Suite Manifest/Traceability/Manual updated; path check green | integration | `bash tests/check-traceability-paths.sh` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky. "File Exists" ❌ W0 = test asserted by a Wave-0 / in-plan TDD stub that does not yet exist on disk pre-execution.*

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
| Live cross-surface alias propagation (desktop owner ↔ web guest) | ALIAS-UI-02 | Requires live relay + real Tailnet web client; presence rebroadcast observed across two surfaces | Set alias from desktop ChatPanel; confirm web guest roster + new message author updates; repeat web→desktop (covered by 161-04 Task 3, gate=blocking) |
| Web-share pre-fill shows correct self-identity | ALIAS-UI-01 (d) | Depends on live Tailnet `WhoIs`/computed-name resolution at WS-upgrade | Open shared `/sessions/{id}` link as guest; confirm "chatting as" pre-fills the guest's computed name (covered by 161-04 Task 3) |

*These are the genuinely live-only behaviors (real Tailnet identity); all unit/integration-checkable behavior is automated in the per-task map above.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
