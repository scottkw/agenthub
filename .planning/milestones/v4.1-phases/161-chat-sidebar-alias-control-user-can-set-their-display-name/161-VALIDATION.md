---
phase: 161
slug: chat-sidebar-alias-control-user-can-set-their-display-name
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-28
validated: 2026-06-28
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
| 161-01-01 | 01 | 1 | ALIAS-UI-01 | T-161-05 | `MakeSelfFrame` emits byte `0x37` + JSON `{personKey,alias}` | unit | `go test ./internal/relay/ -run TestMakeSelfFrame -count=1` | ✅ `internal/relay/protocol_presence_test.go` | ✅ green |
| 161-01-02 | 01 | 1 | ALIAS-UI-01 | T-161-05 | self-frame emitted on connect on both relay + webserver paths | unit | `go test ./internal/relay/... ./internal/webserver/... -run 'Self\|Identity\|Alias' -count=1` | ✅ `internal/relay/server_identity_test.go`, `internal/webserver/identity_test.go` | ✅ green |
| 161-02-01 | 02 | 1 | ALIAS-UI-01, ALIAS-UI-02 | T-161-04 | `RelayClient.sendAliasSet` wraps existing `encodeAliasSetFrame` (no Wails binding) | unit | `npx vitest run src/lib/relayClient.test.ts` | ✅ `frontend/src/lib/relayClient.test.ts` | ✅ green |
| 161-02-02 | 02 | 1 | ALIAS-UI-01 | T-161-05 | parse `MsgSelf` 0x37 → `onSelf(personKey,alias)`; tsc clean | unit | `npx vitest run src/lib/relayClient.test.ts && npx tsc --noEmit` | ✅ `frontend/src/lib/relayClient.test.ts` | ✅ green |
| 161-03-01 | 03 | 2 | ALIAS-UI-02 | T-161-04 | `validateAlias` mirrors Go `ValidateAlias` (≤32 code points via `Array.from`, no C0/C1) | unit | `npx vitest run src/components/Hub/ChatPanel.test.tsx -t validateAlias` | ✅ `frontend/src/components/Hub/ChatPanel.test.tsx` | ✅ green |
| 161-03-02 | 03 | 2 | ALIAS-UI-01, ALIAS-UI-02 | T-161-02, T-161-03 | header control renders + commits + pre-fills; stays enabled when `isReadOnly` (D-06); author renders as escaped text | unit | `npx vitest run src/components/Hub/ChatPanel.test.tsx && npx tsc --noEmit` | ✅ `frontend/src/components/Hub/ChatPanel.test.tsx` | ✅ green |
| 161-04-01 | 04 | 3 | ALIAS-UI-01, ALIAS-UI-02 | — | cross-surface alias propagation (roster + future-message author) e2e | e2e | `npx playwright test e2e/chat-parity.spec.ts` | ✅ `frontend/e2e/chat-parity.spec.ts` | ✅ green¹ |
| 161-04-02 | 04 | 3 | ALIAS-UI-01, ALIAS-UI-02 | — | TESTING.md Suite Manifest/Traceability/Manual updated; path check green | integration | `bash tests/check-traceability-paths.sh` | ✅ `tests/check-traceability-paths.sh` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky. "File Exists" ❌ W0 = test asserted by a Wave-0 / in-plan TDD stub that does not yet exist on disk pre-execution.*

> ¹ **161-04-01 e2e:** The Playwright spec exists on disk (`frontend/e2e/chat-parity.spec.ts:346`, test `ALIAS-UI-01 — alias set on client A propagates to client B presence roster`) and is documented passing 27/27 in `161-04-SUMMARY.md`. It requires a live dev server, so it was **not re-run** during this post-execution audit; status reflects file-present + SUMMARY-documented-green.

---

## Wave 0 Requirements

- [x] Confirm existing tests cover the backend round-trip (`TestRelayIdentity_AliasPropagation`, `TestWebIdentity_*`, RO-cap self-frame) — verified green in audit.
- [x] `frontend/src/lib/relayClient.test.ts` — `sendAliasSet` send-side coverage added (encoder + 0x37 parse).
- [x] `frontend/src/components/Hub/ChatPanel.test.tsx` — alias control render/validate/submit coverage added.

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved (post-execution audit 2026-06-28 — all 8 tasks COVERED by on-disk green tests)

---

## Validation Audit 2026-06-28

Post-execution audit (State A). The Per-Task Map was reconciled from pre-execution
TDD stubs (⬜ pending / ❌ W0) to verified on-disk reality. All asserted test files
exist and run green; no gaps required the nyquist auditor.

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
| Tasks COVERED | 8 / 8 |

**Suites re-run during audit (all green):**
- `go test ./internal/relay/ -run 'TestMakeSelfFrame|TestRelayIdentity_SelfFrameOnConnect|TestRelayIdentity_AliasPropagation'` → ok
- `go test ./internal/webserver/ -run 'TestWebIdentity_SelfFrameOnConnect|TestWebIdentity_ReadOnlySelfFrame|AliasPropagation'` → ok
- `npx vitest run src/lib/relayClient.test.ts src/components/Hub/ChatPanel.test.tsx` → 116/116 passed
- `npx tsc --noEmit` → exit 0
- `bash tests/check-traceability-paths.sh` → OK: all traceability paths exist

**Not re-run (live-server / live-Tailnet only, documented elsewhere):**
- `npx playwright test e2e/chat-parity.spec.ts` — file present, 27/27 per 161-04-SUMMARY (requires live dev server)
- Web-share guest pre-fill (M-33) — live Tailnet WhoIs, user-approved in 161-04 UAT (see VERIFICATION.md)
