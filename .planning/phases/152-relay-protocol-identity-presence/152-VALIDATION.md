---
phase: 152
slug: relay-protocol-identity-presence
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-25
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
| 152-01-01 | 01 | 1 | IDENT-01/02, PRESENCE-01/02 | — | Frame encode/decode lossless | unit | `go test ./internal/relay/ -run 'TestPresencePayloadRoundTrip\|TestTypingPayloadRoundTrip\|TestAliasPayloadRoundTrip'` | ❌ W0 | ⬜ pending |
| 152-01-02 | 01 | 1 | IDENT-02 | T-152-01 | Alias control-char/over-length rejected at source | unit | `go test ./internal/relay/ -run TestValidateAlias` | ❌ W0 | ⬜ pending |
| 152-02-01 | 02 | 1 | IDENT-02 | T-152-06 / T-152-01 | Persisted composite-keyed alias survives reload; fixed path | unit | `go test -race ./internal/daemon/ -run TestAliasStore` | ❌ W0 | ⬜ pending |
| 152-03-01 | 03 | 2 | PRESENCE-01 | — | Per-person refcount collapse + disambiguation | unit | `go test -race ./internal/relay/ -run 'TestPresenceRefCount\|TestPresenceCollapse\|TestCompositePersonKey\|TestUnsubscribePresenceChanged\|TestBroadcastPresence\|TestUpdateAlias'` | ❌ W0 | ⬜ pending |
| 152-03-02 | 03 | 2 | PRESENCE-02 | T-152-03/07/08 | Typing TTL, sender-exclusion, rate-limit, shutdown-safe, never persisted | unit | `go test -race ./internal/relay/ -run 'TestTyping\|TestHubShutdownWithActiveTypingTimer'` | ❌ W0 | ⬜ pending |
| 152-04-01 | 04 | 2 | IDENT-01, PRESENCE-01/02 | T-152-09 | Typed presence/typing parse; malformed→unknown | unit | `cd frontend && pnpm test -- run src/lib/relayClient.test.ts` | ❌ W0 | ⬜ pending |
| 152-05-01 | 05 | 3 | IDENT-01/02 | — | Engine constructs global AliasStore | build+unit | `go build ./... && go test -race ./internal/daemon/ -run 'TestNewSessionEngine\|TestAliasStore'` | ✅ | ⬜ pending |
| 152-05-02 | 05 | 3 | IDENT-01/02, PRESENCE-01/02 | T-152-01/05/11 | Owner stamping + dispatch; RO not gated for chat | build/vet | `go build ./... && go vet ./internal/relay/... ./internal/daemon/...` | ✅ | ⬜ pending |
| 152-05-03 | 05 | 3 | IDENT-02, PRESENCE-01/02 | T-152-05 | Alias propagation one round-trip; typing excludes sender; RO can chat | integration | `go test -race ./internal/relay/ -run 'TestRelayIdentity\|TestRelayAliasPropagation\|TestRelayTypingExcludesSender\|TestRelayReadOnlyCanChat'` | ❌ W0 | ⬜ pending |
| 152-06-01 | 06 | 4 | IDENT-01/02, PRESENCE-01/02 | T-152-01/02/04/05 | WhoIs stamping web origin; parity; RO not gated | build/vet | `go build ./... && go vet ./internal/webserver/... ./internal/daemon/...` | ✅ | ⬜ pending |
| 152-06-02 | 06 | 4 | IDENT-01/02, PRESENCE-01/02 | T-152-04/05 | Web client distinct from owner; alias propagation; RO can chat; traceability | integration | `go test -race ./internal/webserver/ -run 'TestWebIdentity\|TestWebAliasPropagation\|TestWebReadOnlyCanChat' && bash tests/check-traceability-paths.sh` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Sampling continuity: every plan has at least one task with an `<automated>` verify; no 3 consecutive tasks lack automated coverage.*

---

## Wave 0 Requirements

New test files (created by their owning plans, not a separate Wave 0 plan — each test is co-authored with its implementation task):

- [ ] `internal/relay/protocol_presence_test.go` — IDENT-01/02, PRESENCE-01/02 frame round-trip + ValidateAlias (Plan 01)
- [ ] `internal/daemon/alias_store_test.go` — IDENT-02 persist/reload/reject (Plan 02)
- [ ] `internal/relay/hub_presence_test.go` — PRESENCE-01 refcount + PRESENCE-02 typing TTL (Plan 03)
- [ ] `frontend/src/lib/relayClient.test.ts` — extend existing file with presence/typing cases (Plan 04)
- [ ] `internal/relay/server_identity_test.go` — relay WS integration (Plan 05)
- [ ] web-path cases in `internal/webserver/chat_test.go` (or sibling) — web WS integration (Plan 06)

No standalone Wave 0 plan is required: every requirement is covered by tests authored alongside the code in their owning plan. The two `✅` build/vet tasks (152-05-02, 152-06-01) are bracketed by `❌ W0` test tasks in the same plan, preserving Nyquist continuity.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Owner (Wails desktop) and a same-machine browser appear as two distinct, correctly-labelled presence entries over a LIVE tailnet | IDENT-02 / criterion 5 | Requires two real WS connections (native Wails webview + a real browser) against a live Tailscale daemon + daemon; WhoIs self-IP behavior (Assumption A1) cannot be exercised in CI | Launch the daemon + Wails app; open the session web-share URL in a browser on the same machine; confirm the roster shows a `local`-origin owner entry AND a separate `web`-origin entry (no silent merge). Register as M-NN in TESTING.md §5. |
| Typing indicator appears ≤500 ms from a keystroke and auto-clears after 5 s idle | PRESENCE-02 / criterion 4 | Wall-clock keypress→render timing requires a real browser and human input; not deterministic in unit tests | In a live session with two participants, type in one client's composer; confirm the other sees the named indicator within ~0.5 s and that it clears ~5 s after the last keystroke and immediately on abrupt disconnect. Register as M-NN in TESTING.md §5. |

The automated suite covers the server-side TTL mechanism (injected 5ms `typingTTL`), the never-persisted guarantee, sender-exclusion, refcount collapse, and the composite-key disambiguation logic (via the WhoIs-failure fallback). Only the live wall-clock timing and the live two-connection self-IP behavior remain manual.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (co-authored per plan)
- [x] No watch-mode flags (vitest invoked via `run`)
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-06-25
