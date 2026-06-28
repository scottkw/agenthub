---
phase: 161-chat-sidebar-alias-control-user-can-set-their-display-name
verified: 2026-06-28T12:10:00Z
status: human_needed
score: 5/6 must-haves verified
behavior_unverified: 1
overrides_applied: 0
re_verification: false
behavior_unverified_items:
  - truth: "Web-share guest alias input is pre-filled with the Tailscale WhoIs computed name via MsgSelf (0x37)"
    test: "Open a shared /sessions/{id} link as a Tailnet guest; open the chat drawer; observe the 'chatting as' label"
    expected: "Label shows the guest's Tailscale-resolved display name, not 'local' or empty"
    why_human: "Go test TestWebIdentity_SelfFrameOnConnect passes but uses a mock WhoIs that returns empty alias; real computed-name pre-fill requires a live Tailnet WhoIs lookup — cannot be proven by code inspection or unit tests. Already human-approved in Phase 161-04 live UAT."
human_verification:
  - test: "Web-share guest pre-fill shows Tailscale computed name (M-33)"
    expected: "Opening a shared /sessions/{id} link as a Tailnet guest and opening the chat sidebar shows 'chatting as: <Tailscale computed name>' — not empty or 'local'."
    why_human: "Depends on live Tailnet WhoIs resolution at WS-upgrade time; mock in automated tests returns empty alias. NOTE: User-approved in Phase 161-04 live UAT checkpoint — this item is already satisfied; listed here for traceability."
deferred:
  - truth: "Long authorID (nodekey:…) truncates with ellipsis on web-share surface"
    addressed_in: "Future polish phase (user-deferred per 161-04 Plan Summary)"
    evidence: "Explicitly deferred by user in 161-04 SUMMARY: 'Pre-existing cosmetic issue' — not part of ALIAS-UI-01/02 scope"
  - truth: "Chat panel is resizable by width"
    addressed_in: "Future feature phase (user-deferred per 161-04 Plan Summary)"
    evidence: "Feature request surfaced during UAT; out of scope for Phase 161"
---

# Phase 161: Chat Sidebar Alias Control Verification Report

**Phase Goal:** ALIAS-UI-01 — user sets their alias from the shared chat sidebar; available on GUI tab, Hub modal, and web-share guest via the shared `ChatPanel`. ALIAS-UI-02 — set alias persists via the Phase 152 `AliasStore`/`MsgAliasSet` path, immediately updates the user's chat author name and presence-roster name for all participants, and respects `ValidateAlias`.

**Verified:** 2026-06-28T12:10:00Z
**Status:** human_needed (1 behavior-unverified truth; already human-approved in 161-04 live UAT)
**Re-verification:** No — initial verification

---

## Goal Achievement

### Requirement Verdicts

**ALIAS-UI-01: ACHIEVED**
**ALIAS-UI-02: ACHIEVED**

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Alias control (label, input, save) appears in a single un-forked `ChatPanel` mounted by all three surfaces: GUI tab (`TerminalChatHost`), Hub modal (`HubInteractiveModal`), and web-share guest (`WebShareSessionView`) | ✓ VERIFIED | `TerminalChatHost.tsx:99`, `HubInteractiveModal.tsx:93`, `WebShareSessionView.tsx:81` all `import { ChatPanel } from './ChatPanel'` and mount `<ChatPanel>`; `.chat-panel__alias-label`, `.chat-panel__alias-input`, `.chat-panel__alias-save` exist in `ChatPanel.tsx:773-829` |
| 2 | Alias control is NOT disabled for read-only clients (D-06 exception — alias-set is distinct from chat-send) | ✓ VERIFIED | `ChatPanel.tsx:651,778`: explicit comment "NO isReadOnly early-return — the alias control is the D-06 RO exception"; `handleAliasCommit` at line 654 has no `isReadOnly` guard; unit test "stays enabled when isReadOnly" in `ChatPanel.test.tsx:763+` passes (61/61) |
| 3 | Web-share guest alias input is pre-filled with the Tailscale WhoIs computed name via MsgSelf (0x37) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Wire layer proven: `webserver/server.go:1163` emits `relay.MakeSelfFrame({PersonKey, Alias})` on WS connect; `relayClient.ts:189-291` parses 0x37 → `onSelf`; `ChatPanel.tsx:469-471` stores `selfIdentity`; `currentAlias` falls back to `selfIdentity.alias` before roster arrives. `TestWebIdentity_SelfFrameOnConnect` PASSES. However mock WhoIs returns empty alias — real Tailnet computed name requires live WhoIs. Human-approved: Phase 161-04 live UAT checkpoint. |
| 4 | Setting alias sends MsgAliasSet (0x34) which relay/webserver validates via `ValidateAlias`, persists via `AliasStore`, and broadcasts a fresh presence roster to all participants | ✓ VERIFIED | `ChatPanel.tsx:661`: `clientRef.current?.sendAliasSet(validated)`; `relayClient.ts:329-335`: `sendAliasSet` sends `encodeAliasSetFrame(alias)` (0x34 frame); `relay/server.go:365-379`: `case MsgAliasSet` → `ValidateAlias` → `sub.Alias = newAlias` → `sub.AliasSetFn(sub.PersonKey, newAlias)` (AliasStore.Set) → `hub.UpdateAlias` → `NotifyPresence(hub)`; `webserver/server.go:1211-1225`: symmetric path; `TestRelayIdentity_AliasPropagation` PASS (ran live) |
| 5 | After an alias change, the ChatPanel header label updates immediately from the live presence roster — not the frozen MsgSelf connect-time snapshot | ✓ VERIFIED | `ChatPanel.tsx:641-644`: `currentAlias = participants.find(p => p.personKey === selfPersonKey)?.alias ?? selfIdentity?.alias ?? ''`; commit 603b6e0b fixes the frozen-snapshot bug; regression tests "header reflects LIVE roster alias (desktop)" and "(web personKey)" in `ChatPanel.test.tsx:895-958` PASS (61/61) |
| 6 | Client `validateAlias` mirrors Go `ValidateAlias`: trim whitespace, empty→null, >32 Unicode code points rejected (not truncated), C0/C1 control characters rejected | ✓ VERIFIED | `ChatPanel.tsx:210-222`: `Array.from(trimmed)` (code-point counting mirrors `[]rune`), `> 32` → null, `cp < 0x20` (C0) and `cp >= 0x7f && cp <= 0x9f` (C1) → null; matches `protocol.go:221-247` exactly; 12 TDD tests in `ChatPanel.test.tsx:484-537` PASS (61/61) |

**Score:** 5/6 truths verified (1 present, behavior-unverified — code wired, live WhoIs not testable in unit suite; user-approved in 161-04 UAT)

---

### Deferred Items

Items not part of ALIAS-UI-01/02 scope; explicitly deferred by user in Phase 161-04.

| # | Item | Deferred To | Evidence |
|---|------|-------------|----------|
| 1 | Long `authorID` (nodekey:…) un-truncated in message secondary label on web-share causing horizontal scroll | Future polish phase | 161-04 SUMMARY: "Pre-existing cosmetic issue unrelated to the aliasing work" |
| 2 | Resizable chat width | Future feature phase | 161-04 SUMMARY: "Feature request — out of scope for Phase 161" |

---

### Required Artifacts

| Artifact | Provides | Status | Details |
|----------|---------|--------|---------|
| `internal/relay/protocol.go` | `MsgSelf byte = 0x37`, `SelfPayload{PersonKey, Alias}`, `MakeSelfFrame()` | ✓ VERIFIED | Lines 87, 153-162; `MsgSelf` referenced in constant block and frame builder |
| `internal/relay/server.go` | On-connect MsgSelf emit (relay/desktop path) | ✓ VERIFIED | Line 307: `conn.Write(ctx, websocket.MessageBinary, MakeSelfFrame(SelfPayload{PersonKey: sub.PersonKey, Alias: sub.Alias}))` |
| `internal/webserver/server.go` | On-connect MsgSelf emit (web path) | ✓ VERIFIED | Line 1163: `conn.Write(ctx, websocket.MessageBinary, relay.MakeSelfFrame(relay.SelfPayload{PersonKey: sub.PersonKey, Alias: sub.Alias}))` |
| `frontend/src/lib/relayClient.ts` | `MSG_SELF=0x37`, `onSelf` callback, `sendAliasSet` | ✓ VERIFIED | Lines 16, 189-193, 220-291, 329-335; `encodeAliasSetFrame` (0x34) exists at line 94-99 |
| `frontend/src/components/Hub/ChatPanel.tsx` | Alias control UI, `validateAlias`, `currentAlias` live-roster, `handleAliasCommit` | ✓ VERIFIED | Lines 210, 641-665, 773-829; all three surfaces import from this file |
| `frontend/e2e/chat-parity.spec.ts` | ALIAS-UI-01 cross-surface e2e (alias propagation) | ✓ VERIFIED | Lines 334-396; test `ALIAS-UI-01 — alias set on client A propagates to client B presence roster` exists |
| `TESTING.md` | Phase 161 Suite Manifest §2, Traceability §4 (ALIAS-UI-01/02 rows), Manual §5 M-32/M-33 | ✓ VERIFIED | Lines 228-237 (traceability), 471-487 (M-32/M-33 manual items); `bash tests/check-traceability-paths.sh` exits 0 |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `ChatPanel.handleAliasCommit` | `relayClient.sendAliasSet` | `clientRef.current?.sendAliasSet(validated)` (ChatPanel.tsx:661) | ✓ WIRED | Validated alias passes through — no-op on null |
| `relayClient.sendAliasSet` | WebSocket as MsgAliasSet (0x34) | `this.ws.send(encodeAliasSetFrame(alias))` (relayClient.ts:330) | ✓ WIRED | Frame byte 0x34 + UTF-8 JSON payload |
| Relay `case MsgAliasSet` | `AliasStore.Set` + `NotifyPresence` | `sub.AliasSetFn(sub.PersonKey, newAlias)` + `NotifyPresence(hub)` (server.go:375-379) | ✓ WIRED | Both persist and broadcast confirmed |
| `NotifyPresence` → all clients | `setParticipants` in ChatPanel | `onPresence: (p) => setParticipants(p)` (ChatPanel.tsx:459) | ✓ WIRED | Full broadcast-to-state path proven |
| `participants[selfPersonKey]` | `currentAlias` in header | `participants.find(p => p.personKey === selfPersonKey)?.alias` (ChatPanel.tsx:643) | ✓ WIRED | Live roster lookup, not frozen snapshot |
| Server on-connect | `selfIdentity` in ChatPanel | `MakeSelfFrame` → `relayClient onSelf` → `setSelfIdentity` (ChatPanel.tsx:469-471) | ✓ WIRED | Pre-fill seed before roster arrives |
| New chat message | `sub.Alias` at send time | `AuthorAlias: sub.Alias` in `HandleChatSend` (hub.go:688) | ✓ WIRED | `sub.Alias` updated by MsgAliasSet handler — new messages carry updated name |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| MakeSelfFrame produces 0x37 byte + round-trip SelfPayload | `go test ./internal/relay/ -run TestMakeSelfFrame -count=1` | PASS (0.014s) | ✓ PASS |
| Relay emits MsgSelf on connect with personKey + alias | `go test ./internal/relay/ -run TestRelayIdentity_SelfFrameOnConnect -count=1` | PASS | ✓ PASS |
| Relay alias propagation: client A MsgAliasSet → client B MsgPresence updated | `go test ./internal/relay/ -run TestRelayIdentity_AliasPropagation -count=1` | PASS | ✓ PASS |
| Web path emits MsgSelf on connect | `go test ./internal/webserver/ -run TestWebIdentity_SelfFrameOnConnect -count=1` | PASS | ✓ PASS |
| RO-cap web client also receives MsgSelf (D-06) | `go test ./internal/webserver/ -run TestWebIdentity_ReadOnlySelfFrame -count=1` | PASS | ✓ PASS |
| ChatPanel.test.tsx: all 61 tests (validateAlias + alias control + regression) | `cd frontend && npx vitest run src/components/Hub/ChatPanel.test.tsx` | 61/61 PASS | ✓ PASS |
| relayClient.test.ts: MSG_SELF + onSelf + sendAliasSet | `cd frontend && npx vitest run src/lib/relayClient.test.ts` | 55/55 PASS | ✓ PASS |
| Traceability path check | `bash tests/check-traceability-paths.sh` | "OK: all traceability paths exist" | ✓ PASS |
| ALIAS-UI-01 Playwright e2e (alias set → cross-surface roster propagation) | `npx playwright test e2e/chat-parity.spec.ts` | 27/27 per 161-04 SUMMARY — not re-run (requires live dev server) | ? SKIP (documented 27/27 in SUMMARY; live server required) |

---

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| ALIAS-UI-01 | 161-01, 161-02, 161-03, 161-04 | User sets alias from shared ChatPanel sidebar; available on GUI tab, Hub modal, web-share guest | ✓ SATISFIED | Shared component parity verified; alias control UI present; e2e test `ALIAS-UI-01` in chat-parity.spec.ts exists and documented passing 27/27 |
| ALIAS-UI-02 | 161-01, 161-02, 161-03, 161-04 | Alias persists via AliasStore/MsgAliasSet, immediately updates author name + roster, respects ValidateAlias | ✓ SATISFIED | Full wire path: sendAliasSet → relay ValidateAlias+AliasStore+NotifyPresence → setParticipants; currentAlias live-roster tracking (603b6e0b); validateAlias mirror with 12 TDD tests |

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | None found | — | — |

No TBD, FIXME, XXX, or placeholder markers in any Phase 161 modified files.

---

### Human Verification Required

#### 1. Web-share guest alias pre-fill (M-33) — ALREADY USER-APPROVED in 161-04 UAT

**Test:** Open a shared `/sessions/{id}` link as a Tailnet guest (i.e., a user whose machine appears in the host's Tailnet). Open the chat drawer.

**Expected:** The "chatting as" header label shows the guest's Tailscale-resolved display name — not empty, not "local".

**Why human:** Requires a live Tailnet WhoIs lookup at WS-upgrade time. The automated test (`TestWebIdentity_SelfFrameOnConnect`) uses a mock WhoIs that returns empty alias; real pre-fill behavior cannot be observed in the unit suite.

**Prior result:** APPROVED by user in Phase 161-04 live UAT checkpoint.

---

### Gaps Summary

No gaps. All ALIAS-UI-01 and ALIAS-UI-02 must-haves are either fully VERIFIED or covered by user-approved live UAT (Truth 3, web pre-fill). The status is `human_needed` solely because the web pre-fill truth is behavior-dependent on a live Tailnet WhoIs call that automated tests cannot exercise — not because any implementation is missing or broken.

The two deferred items (authorID truncation, resizable chat) are pre-existing cosmetic issues explicitly out of scope for this phase by user decision.

---

_Verified: 2026-06-28T12:10:00Z_
_Verifier: Claude (gsd-verifier)_
