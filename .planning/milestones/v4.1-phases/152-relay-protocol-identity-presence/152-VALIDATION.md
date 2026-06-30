---
phase: 152
slug: relay-protocol-identity-presence
status: verified
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-25
audited: 2026-06-26
---

# Phase 152 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `-race` (backend); vitest (frontend relayClient) |
| **Config file** | none for Go (standard `go test`); frontend uses package.json `test` → `vitest run` |
| **Quick run command** | `go test -race -short ./internal/relay/... ./internal/daemon/...` |
| **Full suite command** | `go test -race ./...` (+ `cd frontend && pnpm test` for relayClient) |
| **Estimated runtime** | ~30-60 seconds backend; ~5 seconds frontend |

---

## Sampling Rate

- **After every task commit:** Run `go test -race -short ./internal/relay/... ./internal/daemon/...` (or `pnpm test -- run src/lib/relayClient.test.ts` for Plan 04)
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd-verify-work`:** Full suite (Go + frontend) must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 152-01-01 | 01 | 1 | IDENT-01/02, PRESENCE-01/02 | — | Frame encode/decode lossless | unit | `go test ./internal/relay/ -run 'TestPresencePayloadRoundTrip\|TestTypingPayloadRoundTrip\|TestAliasPayloadRoundTrip'` | ✅ | ✅ green |
| 152-01-02 | 01 | 1 | IDENT-02 | T-152-01 | Alias control-char/over-length rejected at source | unit | `go test ./internal/relay/ -run TestValidateAlias` | ✅ | ✅ green |
| 152-02-01 | 02 | 1 | IDENT-02 | T-152-06 / T-152-01 | Persisted composite-keyed alias survives reload; fixed path | unit | `go test -race ./internal/daemon/ -run TestAliasStore` | ✅ | ✅ green |
| 152-03-01 | 03 | 2 | PRESENCE-01 | — | Per-person refcount collapse + disambiguation | unit | `go test -race ./internal/relay/ -run 'TestPresenceRefCount\|TestPresenceCollapse\|TestCompositePersonKey\|TestUnsubscribePresenceChanged\|TestBroadcastPresence\|TestUpdateAlias'` | ✅ | ✅ green |
| 152-03-02 | 03 | 2 | PRESENCE-02 | T-152-03/07/08 | Typing TTL, sender-exclusion, rate-limit, shutdown-safe, never persisted | unit | `go test -race ./internal/relay/ -run 'TestTyping\|TestHubShutdownWithActiveTypingTimer'` | ✅ | ✅ green |
| 152-04-01 | 04 | 2 | IDENT-01, PRESENCE-01/02 | T-152-09 | Typed presence/typing parse; malformed→unknown | unit | `cd frontend && pnpm test -- run src/lib/relayClient.test.ts` | ✅ | ✅ green |
| 152-05-01 | 05 | 3 | IDENT-01/02 | — | Engine constructs global AliasStore | build+unit | `go build ./... && go test -race ./internal/daemon/ -run 'TestNewSessionEngine\|TestAliasStore'` | ✅ | ✅ green |
| 152-05-02 | 05 | 3 | IDENT-01/02, PRESENCE-01/02 | T-152-01/05/11 | Owner stamping + dispatch; RO not gated for chat | build/vet | `go build ./... && go vet ./internal/relay/... ./internal/daemon/...` | ✅ | ✅ green |
| 152-05-03 | 05 | 3 | IDENT-02, PRESENCE-01/02 | T-152-05 | Alias propagation one round-trip; typing excludes sender; RO can chat | integration | `go test -race ./internal/relay/ -run 'TestRelayIdentity\|TestRelayAliasPropagation\|TestRelayTypingExcludesSender\|TestRelayReadOnlyCanChat'` | ✅ | ✅ green |
| 152-06-01 | 06 | 4 | IDENT-01/02, PRESENCE-01/02 | T-152-01/02/04/05 | WhoIs stamping web origin; parity; RO not gated | build/vet | `go build ./... && go vet ./internal/webserver/... ./internal/daemon/...` | ✅ | ✅ green |
| 152-06-02 | 06 | 4 | IDENT-01/02, PRESENCE-01/02 | T-152-04/05 | Web client distinct from owner; alias propagation; RO can chat; traceability | integration | `go test -race ./internal/webserver/ -run 'TestWebIdentity\|TestWebAliasPropagation\|TestWebReadOnlyCanChat' && bash tests/check-traceability-paths.sh` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Sampling continuity: every plan has at least one task with an `<automated>` verify; no 3 consecutive tasks lack automated coverage.*

---

## Wave 0 Requirements

New test files (created by their owning plans, not a separate Wave 0 plan — each test is co-authored with its implementation task):

- [x] `internal/relay/protocol_presence_test.go` — IDENT-01/02, PRESENCE-01/02 frame round-trip + ValidateAlias (Plan 01)
- [x] `internal/daemon/alias_store_test.go` — IDENT-02 persist/reload/reject (Plan 02)
- [x] `internal/relay/hub_presence_test.go` — PRESENCE-01 refcount + PRESENCE-02 typing TTL (Plan 03)
- [x] `frontend/src/lib/relayClient.test.ts` — extended with presence/typing cases (Plan 04)
- [x] `internal/relay/server_identity_test.go` — relay WS integration (Plan 05)
- [x] `internal/webserver/identity_test.go` — web WS integration (Plan 06; landed as `identity_test.go`, the planned "sibling" of chat_test.go)

No standalone Wave 0 plan was required: every requirement is covered by tests authored alongside the code in its owning plan. All six test files now exist and run green (audited 2026-06-26). The two build/vet tasks (152-05-02, 152-06-01) are bracketed by test tasks in the same plan, preserving Nyquist continuity.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Owner and a same-machine browser appear as two distinct presence entries over a LIVE tailnet (M-18) | IDENT-02 / criterion 5 | Requires a live Tailscale daemon with a real `lc.WhoIs` response; in CI `WhoIs` always fails so `tailnetID` stays `"unknown"` | ✅ **VERIFIED 2026-06-26** — live `lc.WhoIs` against running tailscaled (exact `webserver/server.go` path) resolved real distinct node keys (self `nodekey:456d9361…:web`, peer kens-inspiron `nodekey:ec65d9ee…:web`), both ≠ owner `local:local`; loopback → `unknown:web`. Recorded in 152-UAT.md (Test 1) + TESTING.md M-18. |
| Typing indicator appears ≤500 ms from a keystroke and auto-clears after 5 s idle, in a real browser (M-19) | PRESENCE-02 / criterion 4 | The visual indicator UI that renders typing frames does not exist in Phase 152 (152-04 parses frames only; ChatPanel ships in 154/155) | Wire-level mechanism VERIFIED by automated suite. **Browser-visual observation DEFERRED to Phases 154/155 UAT** (per user decision 2026-06-26) — recorded in 152-UAT.md (Test 2) + TESTING.md M-19. |

The automated suite covers the server-side TTL mechanism (injected 5ms `typingTTL`), the never-persisted guarantee, sender-exclusion, refcount collapse, and the composite-key disambiguation logic. M-18's live self/peer WhoIs behavior was verified live 2026-06-26; only M-19's browser-visual rendering remains, and it is deferred to the chat-UI phases (154/155).

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (co-authored per plan)
- [x] No watch-mode flags (vitest invoked via `run`)
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-06-25; post-execution audit verified 2026-06-26

---

## Validation Audit 2026-06-26

Post-execution audit (State A) — re-ran every command in the Per-Task Verification Map against the executed code.

| Metric | Count |
|--------|-------|
| Requirements audited | 4 (IDENT-01, IDENT-02, PRESENCE-01, PRESENCE-02) |
| Task entries | 11 |
| COVERED (green) | 11 |
| PARTIAL | 0 |
| MISSING | 0 |
| Gaps found | 0 |
| Gaps filled | 0 (none needed) |
| Escalated to manual-only | 0 |

All 6 Wave 0 test files exist and run green; `go build ./...`, `go vet` (relay/daemon/webserver), traceability check, and 29 frontend relayClient tests all pass. M-18 manual item verified live; M-19 browser-visual deferred to 154/155. **Phase 152 is Nyquist-compliant.**
