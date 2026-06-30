---
phase: 152-relay-protocol-identity-presence
verified: 2026-06-26T21:57:00Z
status: passed
score: 4/5 must-haves verified
behavior_unverified: 1
overrides_applied: 0
re_verification: false
behavior_unverified_items:

  - truth: "A typing indicator appears for a participant within 500ms of starting to compose; auto-clears within ~500ms of the 5-second TTL expiry — in a live browser with real network"
    test: "M-19 — connect two browser clients over a live tailnet; one types (sends MsgTyping{typing:true}); observe the typing indicator appears in the observer's presence view within 500ms; stop typing and confirm the indicator clears within ~500ms of 5s TTL expiry"
    expected: "Indicator appears promptly (<500ms); clears after 5s inactivity; no lingering indicator after disconnect"
    why_human: "The server broadcasts MsgTyping immediately (BroadcastExcept, synchronous), but the 500ms guarantee is a network-latency claim. TestTypingTTL verifies the TTL mechanism with a 5ms injected timer. The absolute timing in a live browser requires human observation."
human_verification:

  - test: "M-18 — Live two-client presence distinctness over a real tailnet"
    expected: "Desktop owner appears as `local:local`; a separate tailnet peer's browser appears as `<nodeKey>:web`. Both entries are distinct in the presence roster; no silent identity merge."
    why_human: "TestWebIdentity_WhoIsFailureFallback proves the no-collision property with WhoIs failure (tailnetID='unknown:web'). Real WhoIs stamping (actual node key from a live tailscaled) cannot be tested without a running tailscaled and a live tailnet. M-18 is the only proof that real WhoIs data flows through correctly."

  - test: "M-19 — Typing indicator timing in a live browser"
    expected: "Typing indicator appears ≤500ms after MsgTyping{typing:true}; auto-clears ≤500ms after the 5-second TTL fires; clears immediately on abrupt disconnect."
    why_human: "The TTL mechanism is tested with a 5ms injected timer (TestTypingTTL passes). The 500ms wall-clock guarantee is a network-latency claim that requires a live browser and real relay to observe."
---

# Phase 152: Relay Protocol + Identity + Presence — Verification Report

**Phase Goal:** Every participant is identified and their live presence and typing status propagate in real time across the relay.
**Verified:** 2026-06-26T21:57:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Step 0: Previous Verification

No previous VERIFICATION.md found. Proceeding with initial verification.

---

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| #  | Truth | Status | Evidence |
|----|-------|--------|----------|
| 1  | Each WS connection stamped with TailnetID ("local" for owner, WhoIs-derived for web) and alias before any message stored; both visible to all participants | VERIFIED | `handleSession`: PersonKey="local:local", TailnetID="local", Alias from AliasStore before `hub.Subscribe`. `handleWSSRelay`: `lc.WhoIs` after `websocket.Accept`, non-"local" fallback on failure. `NotifyPresence` broadcasts the full roster including identity to all clients. Integration tests: `TestRelayIdentity_AliasPropagation`, `TestWebAliasPropagation` confirm receipt. |
| 2  | A participant can set/change their alias; the updated name propagates to all connected clients within one relay round-trip | VERIFIED | `case MsgAliasSet` in both read pumps: `ValidateAlias` → `sub.AliasSetFn` (AliasStore persist) → `hub.UpdateAlias` → `NotifyPresence(hub)`. All synchronous in one read-pump iteration. `TestRelayIdentity_AliasPropagation` (relay) and `TestWebAliasPropagation` (web) verify both clients receive the updated MsgPresence roster. |
| 3  | All currently-connected participants see each other's presence (connected/disconnected) update in real time when someone joins or leaves | VERIFIED | `NotifyPresence(hub)` called unconditionally after `hub.Subscribe` (join); `Unsubscribe` returns `presenceChanged bool` and `NotifyPresence` called only when `presenceChanged=true` (last connection for a PersonKey dropped). `TestPresenceRefCount`, `TestUnsubscribePresenceChanged` verify refcount + bool. `TestRelayIdentity_AliasPropagation` verifies both relay clients receive the initial MsgPresence frame. |
| 4  | Typing indicator appears within 500ms of starting to compose; auto-clears after 5s TTL or disconnect; never stored in JSONL log | PRESENT_BEHAVIOR_UNVERIFIED | Mechanism verified: `hub.UpdateTyping` broadcasts via `BroadcastExcept` immediately (no scheduling delay). `time.AfterFunc(5*time.Second, ...)` sets the TTL. `Unsubscribe` stops the timer on disconnect. `TestTypingTTL` (typingTTL=5ms) verifies auto-clear fires. No `AppendMessage`/`ChatStore` references in typing code path (grep clean). The "within 500ms" timing guarantee in a real browser over a real network is a runtime claim not provable by code inspection — requires M-19. |
| 5  | Desktop Wails owner and a same-machine browser client appear as two distinct, correctly-labelled presence entries — no silent identity merge | VERIFIED | By construction: relay path stamps PersonKey="local:local"; web path stamps PersonKey=tailnetID+":web". On WhoIs failure tailnetID="unknown", giving "unknown:web" ≠ "local:local". `TestWebIdentity_WhoIsFailureFallback` asserts Origin="web", PersonKey ends in ":web", and PersonKey ≠ "local:local". |

**Score:** 4/5 truths verified (1 present, behavior-unverified — SC-4 timing guarantee)

---

## Required Artifacts

| Artifact | Status | Evidence |
|----------|--------|----------|
| `internal/relay/protocol.go` | VERIFIED | Constants MsgPresence=0x32, MsgTyping=0x33, MsgAliasSet=0x34 at lines 82-84; structs PresenceEntry/PresencePayload/TypingPayload/AliasPayload; Make*Frame encoders; ValidateAlias (line 157). |
| `internal/relay/protocol_presence_test.go` | VERIFIED | TestPresencePayloadRoundTrip, TestTypingPayloadRoundTrip, TestAliasPayloadRoundTrip, TestTypingPayload_TypingFalse, TestValidateAlias (17-case table). `go test -race ./internal/relay/`: PASS. |
| `internal/daemon/alias_store.go` | VERIFIED | AliasStore struct with RWMutex, filePath ("aliases.json" hardcoded), aliases map. NewAliasStore/Get/Set/GetOrDefault/loadFromDisk/persist. Set delegates to relay.ValidateAlias before any mutation. |
| `internal/daemon/alias_store_test.go` | VERIFIED | 9 tests including reload-persistence (D-02), composite key isolation, invalid-alias rejection, 0600 perms, fixed basename. `go test -race ./internal/daemon/ -run TestAliasStore`: PASS. |
| `internal/relay/hub.go` | VERIFIED | Subscriber identity fields (TailnetID, Origin, PersonKey, Alias, AliasSetFn). Hub: presenceRoster, typingRoster, lastTypingBcast, typingTTL (all make()-initialized in NewHub). Unsubscribe signature: `(presenceChanged bool)`. BroadcastPresence, BroadcastExcept, CurrentPresence, UpdateAlias, UpdateTyping, NotifyPresence, NotifyTyping. |
| `internal/relay/hub_presence_test.go` | VERIFIED | 14 tests: TestPresenceCollapse, TestPresenceRefCount, TestCompositePersonKey, TestUnsubscribePresenceChanged, TestBroadcastPresence, TestUpdateAlias, TestEmptyPersonKeyNoPresenceEntry, TestNotifyPresence, TestTypingSenderExclusion, TestTypingTTL, TestTypingTimerReset, TestTypingExplicitStop, TestUnsubscribeCancelsTypingTimer, TestHubShutdownWithActiveTypingTimer. `go test -race ./internal/relay/`: PASS. |
| `frontend/src/lib/relayClient.ts` | VERIFIED | MSG_PRESENCE=0x32, MSG_TYPING=0x33, MSG_ALIAS_SET=0x34 exported; PresenceEntry interface with Go-matching field names; ServerFrame union extended; parseServerFrame cases 0x32/0x33 with try/catch; encodeTypingFrame, encodeAliasSetFrame; default:unknown preserved. |
| `frontend/src/lib/relayClient.test.ts` | VERIFIED | 29/29 vitest tests pass including constant values, presence/typing decode, backward-compat 0x30/0x31, malformed body → unknown, encode leading bytes. |
| `internal/daemon/engine.go` | VERIFIED | `aliasStore *AliasStore` field; constructed in NewSessionEngine over daemonConfigDir; `Aliases() *AliasStore` accessor; ConfigDirForTest reconstructs aliasStore for test isolation. |
| `internal/relay/server.go` | VERIFIED | Server identity provider fields + SetIdentityProviders setter; handleSession stamps PersonKey="local:local"/TailnetID="local"/Origin="local" before Subscribe; NotifyPresence on join; presenceChanged gate on leave; case MsgTyping + case MsgAliasSet OUTSIDE ReadOnly gate (D-06). |
| `internal/relay/server_identity_test.go` | VERIFIED | TestRelayIdentity_AliasPropagation, TestRelayIdentity_TypingExcludesSender, TestRelayIdentity_ReadOnlyCanChat. `go test -race ./internal/relay/ -run TestRelayIdentity`: PASS. |
| `internal/daemon/api.go` | VERIFIED | RelayHandler wires engine.hostname + AliasStore.GetOrDefault + AliasStore.Set into SetIdentityProviders; setChatProviders extended to call SetAliasProviders on WebServer; nil-guarded. |
| `internal/webserver/server.go` | VERIFIED | SetAliasProviders setter; handleWSSRelay: lc.WhoIs AFTER websocket.Accept (Pitfall 1), tailnetID from WhoIs or "unknown" fallback, Origin="web", personKey=tailnetID+":web"; NotifyPresence on join; presenceChanged gate on leave; case MsgTyping + MsgAliasSet OUTSIDE ReadOnly gate (D-06). |
| `internal/webserver/identity_test.go` | VERIFIED | TestWebIdentity_WhoIsFailureFallback, TestWebAliasPropagation, TestWebReadOnlyCanChat. `go test -race ./internal/webserver/ -run TestWebIdentity\|TestWebAlias\|TestWebReadOnly`: PASS. |
| `TESTING.md` | VERIFIED | Suite Manifest updated (355→359 Go tests); Section 4 traceability rows for IDENT-01/IDENT-02/PRESENCE-01/PRESENCE-02 added; M-18 and M-19 added to Section 5 manual checklist. `bash tests/check-traceability-paths.sh`: exits 0 ("OK: all traceability paths exist"). |

---

## Key Link Verification

| From | To | Via | Status |
|------|----|-----|--------|
| `relay/protocol.go` constants (0x32-0x34) | `relay/hub.go` BroadcastPresence/UpdateTyping | Used in MakePresenceFrame/MakeTypingFrame calls inside hub methods | WIRED |
| `relay/protocol.go` ValidateAlias | `daemon/alias_store.go` Set | `relay.ValidateAlias(alias)` call at alias_store.go:80 | WIRED |
| `relay/protocol.go` constants | `relay/server.go` read pump | case MsgTyping/MsgAliasSet in the switch | WIRED |
| `relay/protocol.go` constants | `webserver/server.go` read pump | case relay.MsgTyping/relay.MsgAliasSet in the switch | WIRED |
| `relay/protocol.go` constants | `frontend/src/lib/relayClient.ts` | MSG_PRESENCE=0x32, MSG_TYPING=0x33, MSG_ALIAS_SET=0x34 mirroring Go values | WIRED |
| `daemon/alias_store.go` AliasStore | `daemon/engine.go` SessionEngine | aliasStore field + NewAliasStore in NewSessionEngine + Aliases() accessor | WIRED |
| `daemon/engine.go` Aliases() | `daemon/api.go` RelayHandler + setChatProviders | SetIdentityProviders + SetAliasProviders wired via closures | WIRED |
| `relay/hub.go` Unsubscribe(bool) | `internal/status` HubLike interface | `Unsubscribe(sub *relay.Subscriber) (presenceChanged bool)` in detector.go:221 | WIRED |
| `relay/hub.go` Unsubscribe(bool) | `relay/server.go` handleSession | `presenceChanged := hub.Unsubscribe(sub)` + conditional NotifyPresence | WIRED |
| `relay/hub.go` Unsubscribe(bool) | `webserver/server.go` handleWSSRelay | `presenceChanged := hub.Unsubscribe(sub)` + conditional relay.NotifyPresence | WIRED |

---

## Behavioral Spot-Checks (Step 7b)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Protocol round-trip tests | `go test -race ./internal/relay/ -run TestPresence\|TestTyping\|TestAlias\|TestValidateAlias` | PASS | PASS |
| AliasStore persistence | `go test -race ./internal/daemon/ -run TestAliasStore` | PASS | PASS |
| Hub presence/typing layer | `go test -race ./internal/relay/ -run TestPresenceRefCount\|TestTyping\|TestHubShutdown` | PASS | PASS |
| Relay identity integration | `go test -race ./internal/relay/ -run TestRelayIdentity` | PASS | PASS |
| Web identity + parity | `go test -race ./internal/webserver/ -run TestWebIdentity\|TestWebAlias\|TestWebReadOnly` | PASS | PASS |
| TypeScript wire protocol | `pnpm test -- run src/lib/relayClient.test.ts` (29 tests) | PASS | PASS |
| Full Go internal suite | `go test -race ./internal/...` (all 13 packages) | PASS | PASS |
| Full vitest suite | `pnpm test` (120 files, 1943 tests) | PASS | PASS |
| Build clean | `go build ./...` | No output (clean) | PASS |
| Traceability check | `bash tests/check-traceability-paths.sh` | "OK: all traceability paths exist" | PASS |

---

## Orchestrator-Fixed Integration Issues — Soundness Check

The orchestrator noted two cross-package integration issues caught and fixed after per-plan checks:

**Issue 1: `relay.Hub.Unsubscribe(bool)` signature breaking `status.HubLike` interface**

Finding: `internal/status/detector.go` line 221 shows the HubLike interface now correctly declares `Unsubscribe(sub *relay.Subscriber) (presenceChanged bool)`. `go build ./...` clean. `go test -race ./internal/...` passes including the status package. **Sound — properly fixed.**

**Issue 2: Subscribe-time MsgPresence push breaking pre-existing `TestWebServerWSS` frame reader**

Finding: `internal/relay/server_test.go` line 85: `readDataFrame` helper skips `MsgMeta || MsgPresence` housekeeping frames. `internal/webserver/server_test.go` line 303: inline reader also skips `relay.MsgMeta || relay.MsgPresence`. Both pre-existing test suites pass (`go test -race ./internal/relay/`, `go test -race ./internal/webserver/`). **Sound — properly fixed.**

---

## D-06 Compliance Verification (ReadOnly gate)

**relay/server.go:** `case MsgTyping` at line 339 and `case MsgAliasSet` at line 347 appear AFTER the closing brace of the `if !sub.ReadOnly { case MsgInput }` block. Both cases are top-level switch branches. Confirmed by grep showing MsgAliasSet/MsgTyping are NOT nested inside the `if !sub.ReadOnly` check.

**webserver/server.go:** Same structure: `case relay.MsgInput` at line 1116 is inside `if !sub.ReadOnly { }`. `case relay.MsgTyping` at line 1131 and `case relay.MsgAliasSet` at line 1139 are outside this gate. RO clients can set alias and type; only PTY input remains gated.

`TestRelayIdentity_ReadOnlyCanChat` and `TestWebReadOnlyCanChat` both pass, confirming RO clients receive confirming MsgPresence after MsgAliasSet.

---

## Requirements Coverage

| Requirement | Description | Covered By | Status |
|-------------|-------------|-----------|--------|
| IDENT-01 | Each participant identified by tailnet ID and alias, both visible | Plans 01, 05, 06; tests: server_identity_test.go, identity_test.go | SATISFIED |
| IDENT-02 | Alias settable/changeable; local owner and web client correctly disambiguated | Plans 02, 05, 06; tests: alias_store_test.go, server_identity_test.go, identity_test.go | SATISFIED |
| PRESENCE-01 | Presence (connected/disconnected) shown to all participants | Plans 03, 05, 06; tests: hub_presence_test.go, server_identity_test.go | SATISFIED |
| PRESENCE-02 | Typing indicators: debounced, volatile, never stored, server-side TTL clears on disconnect | Plans 03, 05, 06; tests: hub_presence_test.go | SATISFIED (mechanism); M-19 covers ≤500ms timing |

---

## Anti-Patterns Scan

Files modified by Phase 152 scanned: `protocol.go`, `hub.go`, `relay/server.go`, `alias_store.go`, `engine.go`, `api.go`, `webserver/server.go`, `relayClient.ts`.

No `TBD`, `FIXME`, or `XXX` markers found in any modified file. No placeholder implementations. No hardcoded empty state that is not an intentional initial value (AliasStore initializes with empty map on first run — correct first-run behavior). No stubs in typing or presence dispatch paths.

---

## Human Verification Required

### 1. Live Two-Client Presence Distinctness (M-18)

**Test:** Connect the desktop Wails owner to a session and open the session's web-share URL on a separate machine (or as a tailnet peer's browser). Both see each other in the presence roster.
**Expected:** Desktop owner appears with PersonKey "local:local". Web peer appears with PersonKey `<actualNodeKey>:web` (a real Tailscale node key, not "unknown"). Both entries are distinct; no silent merge.
**Why human:** TestWebIdentity_WhoIsFailureFallback proves the no-collision property via the "unknown:web" fallback (no live tailscaled). Verifying that a real WhoIs response populates the actual node key requires a running tailscaled and a live tailnet — not automatable without those services.

### 2. Typing Indicator Timing in Live Browser (M-19)

**Test:** Open two browser windows (or one desktop app + one web-share browser). Start typing in client A's composer; observe client B's presence view. Stop typing and wait.
**Expected:** Typing indicator appears in client B's view within ≤500ms of client A sending MsgTyping{typing:true}. After client A stops typing for 5 seconds, the indicator clears within ≤500ms of TTL expiry. On abrupt tab close (disconnect), the indicator also clears (server TTL fires within 5s).
**Why human:** The TTL mechanism is proven by TestTypingTTL with a 5ms injected timer. The "within 500ms" claim is a network-latency guarantee observable only in a live browser with a real relay connection.

---

## Gaps Summary

No automated gaps found. All code artifacts exist, are substantive, and are correctly wired. All tests pass. The two human verification items (M-18, M-19) are existing TESTING.md checklist entries that require a live tailnet — they represent release-gating UAT, not code defects.

---

_Verified: 2026-06-26T21:57:00Z_
_Verifier: Claude (gsd-verifier) — Phase 152 initial verification_
