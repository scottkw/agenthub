---
phase: 163-read-only-guest-chat-posting-d-06-reconciliation
verified: 2026-06-28T23:45:00Z
status: passed
score: 10/10 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "Run the Playwright ROCHAT-01/02 end-to-end test against a live daemon"
    expected: "RO web-share guest's Send button is NOT disabled; typing a unique message and pressing Enter causes the message to appear in the chat thread; no MsgInjectError NAK appears"
    why_human: "frontend/e2e/chat-parity.spec.ts ROCHAT-01/02 test requires a live daemon + fixture session; cannot run in CI without the full daemon stack. Go integration tests (TestChatSend_ROCanPost_RelayPath / WebPath) and vitest tests prove the behavior at the component level, but the browser-surface end-to-end path needs a live run."

  - test: "Confirm no separate /gsd-secure-phase 163 review artefact exists (threat model was embedded in plan)"
    expected: "Either (a) a secure-phase artefact exists confirming the review, or (b) the developer accepts the embedded STRIDE threat model (T-163-01..T-163-06) in the plan files as sufficient for this targeted single-gate change"
    why_human: "The phase plans contain STRIDE threat registers (T-163-01..T-163-06) but /gsd-secure-phase 163 was not separately invoked. Code evidence fully proves only MsgChatSend was loosened (hub.go:661-691 no ReadOnly check; inject gate hub.go:585-587 intact; MsgInput discard relay/server.go:343 + webserver/server.go:1190 intact). Human sign-off needed to confirm the embedded threat model is accepted as the security review record for this phase."
---

# Phase 163: RO Guest Chat Posting (D-06 Reconciliation) Verification Report

**Phase Goal:** RO-cap guests (web-share, Hub modal, desktop tab) can post chat messages and appear in presence/typing, while `@session` inject and PTY/terminal input remain RO-gated. Reverses the SEC-01 RO chat-send gate (Phase 154) and reconciles the D-06 vs SEC-01/PITFALLS conflict.
**Verified:** 2026-06-28T23:45:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | RO subscriber MsgChatSend persists and broadcasts on relay and webserver paths | VERIFIED | `hub.HandleChatSend` (hub.go:661-691) has no `sub.ReadOnly` check; relay/server.go:382-398 and webserver/server.go:1228-1244 both dispatch to hub.HandleChatSend; `TestHandleChatSend_ROCanPost`, `TestChatSend_ROCanPost_RelayPath`, `TestChatSend_ROCanPost_WebPath` all PASS under -race |
| 2 | RO subscriber `@session` inject returns ErrReadOnly and performs zero PTY writes | VERIFIED | `hub.HandleInject` (hub.go:582-641) has `if sub.ReadOnly { return ErrReadOnly }` at lines 585-587; `TestHandleChatSend_ROCanPostInjectStillGated`, `TestInjectRO_WebPath` (both perm shapes), `TestInject_ROCap_RelayPath` all PASS |
| 3 | RO subscriber MsgInput keystroke is discarded with zero PTY writes | VERIFIED | `relay/server.go:343` has `if !sub.ReadOnly {` discard; `webserver/server.go:1190` same pattern; `TestSecurity_ReadOnlyCapabilityBlocksMsgInput` PASS |
| 4 | ONLY HandleChatSend gate loosened; HandleInject (ErrReadOnly) and MsgInput discard byte-for-byte unchanged | VERIFIED | `grep -n "ErrChatReadOnly" internal/ --include="*.go"` returns only a comment reference in hub.go; HandleInject gate at hub.go:585-587 intact; MsgInput discard at relay/server.go:343 and webserver/server.go:1190 intact; `TestHandleChatSend_ROCanPostInjectStillGated` (dual-invariant: RO chat ok + inject blocked + PTY=0) PASS |
| 5 | ChatPanel Send button enabled for RO (no disabled, no aria-disabled, no opacity/cursor RO style); Enter and click call sendChat | VERIFIED | ChatPanel.tsx:1037 `disabled={!draft.trim()}` — no `isReadOnly` in expression; no `aria-disabled` attribute; no opacity/cursor RO style; `handleSend` (line 614-615 comment confirms guard removed); vitest: "RO viewer Send button is NOT disabled", "RO viewer clicking Send calls sendChat", "RO viewer pressing Enter calls sendChat and clears textarea" — all PASS (66/66 suite green) |
| 6 | RO `@session` press-and-hold inject gesture returns early; sendSessionInject never called | VERIFIED | ChatPanel.tsx:667 `if (isReadOnly) return` in `handleInjectPointerDown` intact; vitest "RO viewer press-and-hold does NOT call sendSessionInject (ROCHAT-02 inject gate)" PASS |
| 7 | No blanket "Read only" composer label | VERIFIED | `grep -c "Read only" frontend/src/components/Hub/ChatPanel.tsx` = 0 (zero occurrences in source); `chat-composer__readonly-label` JSX block removed |
| 8 | PITFALLS.md no longer states RO cannot post chat; records D-06 reconciled rule with Phase 163 citation | VERIFIED | `grep -n "cannot post" .planning/research/PITFALLS.md` returns only `"intentionally removed"` context; line 21 records reconciled rule; lines 273, 294, 324 all updated to reference @session/PTY as the remaining gate; Phase 163 cited throughout |
| 9 | TESTING.md Section 2 Phase 163 delta + Section 4 ROCHAT-01, ROCHAT-02, SEC-RO-01 rows with repo-relative paths | VERIFIED | TESTING.md lines 241-248 contain ROCHAT-01 (2 rows), ROCHAT-02 (5 rows), SEC-RO-01 (1 row); Section 2 line 34 has Phase 163 delta note; CHAT-01 rows updated to D-06 behavior |
| 10 | tests/check-traceability-paths.sh passes | VERIFIED | `bash tests/check-traceability-paths.sh` exits 0: "OK: all traceability paths exist" |

**Score:** 10/10 truths verified (0 present, behavior-unverified)

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| ROCHAT-01 | 163-01, 163-02 | RO can post chat across all surfaces via the shared ChatPanel | SATISFIED | hub.HandleChatSend RO gate removed (hub.go:661-691); ChatPanel Send unblocked (ChatPanel.tsx:1037); Go + vitest tests PASS |
| ROCHAT-02 | 163-01, 163-02 | RO @session inject + PTY input remain gated | SATISFIED | HandleInject ErrReadOnly gate intact (hub.go:585-587); MsgInput discard intact (relay/server.go:343, webserver/server.go:1190); handleInjectPointerDown gate intact (ChatPanel.tsx:667); Go + vitest tests PASS |
| SEC-RO-01 | 163-01, 163-03 | Security re-review: only MsgChatSend was loosened | SATISFIED (code-verified) | File:line proof: HandleChatSend (hub.go:661-691) — no ReadOnly check; HandleInject (hub.go:585-587) — ErrReadOnly gate intact; relay/server.go:343 — MsgInput discard intact; webserver/server.go:1190 — MsgInput discard intact; `TestHandleChatSend_ROCanPostInjectStillGated` proves dual-invariant under -race. Note: `/gsd-secure-phase 163` not separately run — STRIDE threat model (T-163-01..T-163-06) embedded in plan files; human sign-off requested below. |

**Requirements traceability gap (WARNING):** ROCHAT-01, ROCHAT-02, and SEC-RO-01 are declared in PLAN frontmatter but do not appear in `.planning/REQUIREMENTS.md`. They are tracked in TESTING.md Section 4 (traceability rows added by plan 163-03) but REQUIREMENTS.md has no entries for them. These IDs amend the existing SEC-01 requirement (Phase 153). Not a blocker — phase goal is achieved and TESTING.md is the canonical regression home — but REQUIREMENTS.md is stale regarding this reconciliation.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/hub.go` | ErrChatReadOnly removed; HandleChatSend no sub.ReadOnly check; HandleInject ErrReadOnly gate intact | VERIFIED | ErrChatReadOnly absent from all non-comment Go code; HandleChatSend (lines 661-691) has no ReadOnly branch; HandleInject (lines 582-641) has `if sub.ReadOnly { return ErrReadOnly }` at line 585 |
| `internal/relay/server.go` | MsgChatSend dispatch comment updated (D-06 reference); MsgInput discard unchanged | VERIFIED | Lines 382-398: D-06 comment updated; line 343: `if !sub.ReadOnly {` discard intact |
| `internal/webserver/server.go` | MsgChatSend dispatch comment updated (D-06); MsgInput discard unchanged | VERIFIED | Lines 1228-1244: D-06 comment updated; line 1190: `if !sub.ReadOnly {` discard intact |
| `internal/relay/hub_chatsend_test.go` | TestHandleChatSend_ROCanPost + TestHandleChatSend_ROCanPostInjectStillGated present and PASS | VERIFIED | Both tests PASS under -race |
| `internal/relay/server_chatsend_test.go` | TestChatSend_ROCanPost_RelayPath present and PASS | VERIFIED | PASS under -race |
| `internal/webserver/server_chatsend_test.go` | TestChatSend_ROCanPost_WebPath (both perm shapes) present and PASS | VERIFIED | PASS under -race (browse_off + browse_on subtests) |
| `frontend/src/components/Hub/ChatPanel.tsx` | Send button enabled for RO; no Read-only label; inject gate retained | VERIFIED | disabled={!draft.trim()} (line 1037); no "Read only" text; handleInjectPointerDown `if (isReadOnly) return` at line 667 |
| `frontend/src/components/Hub/ChatPanel.test.tsx` | 5 new ROCHAT-01/02 tests present and PASS | VERIFIED | 66/66 vitest tests PASS; ROCHAT-01/02 describe block confirmed in output |
| `frontend/e2e/chat-parity.spec.ts` | ROCHAT-01/02 E2E test rewritten (RO Send not disabled + RO message appears) | VERIFIED (text only — live run pending) | SC-3 rewritten to ROCHAT-01/02 semantics; live daemon run is human verification item |
| `.planning/research/PITFALLS.md` | D-06 reconciled; Phase 163 cited; no stale "cannot post" claim | VERIFIED | Line 21 reconciled rule; lines 273/294/324 updated; no stale "cannot post" assertions |
| `TESTING.md` | Section 2 Phase 163 delta note; Section 4 ROCHAT-01/02/SEC-RO-01 rows; traceability paths valid | VERIFIED | All rows present; check-traceability-paths.sh passes |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| relay/server.go MsgChatSend (line 382) | hub.HandleChatSend (hub.go:661) | Direct call; no local RO gate between dispatch and HandleChatSend | WIRED | relay/server.go:395 calls `hub.HandleChatSend(sub, ip.Content)` |
| webserver/server.go MsgChatSend (line 1228) | hub.HandleChatSend (hub.go:661) | Direct call; no local RO gate between dispatch and HandleChatSend | WIRED | webserver/server.go:1240 calls `hub.HandleChatSend(sub, ip.Content)` |
| hub.HandleInject (hub.go:582) → ErrReadOnly gate (line 585) | relay/server.go MsgSessionInject dispatch + webserver/server.go MsgSessionInject dispatch | Both server files call hub.HandleInject; HandleInject gates first | WIRED | relay/server.go:399-418; webserver/server.go:1245-1257 both route MsgSessionInject to hub.HandleInject |
| ChatPanel.tsx handleInjectPointerDown (line 665) → `if (isReadOnly) return` (line 667) | sendSessionInject call (line 676) | isReadOnly early-return prevents sendSessionInject; inject gate is server-side authoritative (hub.HandleInject:585) with ChatPanel as defense-in-depth | WIRED | Line 667 check confirmed; vitest ROCHAT-02 test confirms sendSessionInject NOT called for RO |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go test suite: ChatSend + Inject + MsgInput RO gates | `go test ./internal/relay/... ./internal/webserver/... -run 'ChatSend|Inject|ReadOnlyCapabilityBlocksMsgInput' -race -count=1` | relay: ok 2.0s; webserver: ok 4.0s | PASS |
| Named test: RO can post and inject stays gated | `go test ./internal/relay/... -run 'TestHandleChatSend_ROCanPostInjectStillGated' -race -count=1 -v` | PASS 0.05s | PASS |
| Named test: RO inject still blocked (relay path) | `go test ./internal/relay/... -run 'TestInject_ROCap_RelayPath' -race -count=1 -v` | PASS | PASS |
| Named test: RO inject still blocked (web path) | `go test ./internal/webserver/... -run 'TestInjectRO_WebPath' -race -count=1 -v` | PASS (browse_off + browse_on) | PASS |
| Named test: MsgInput PTY discard unchanged | `go test ./internal/webserver/... -run 'TestSecurity_ReadOnlyCapabilityBlocksMsgInput' -race -count=1 -v` | PASS | PASS |
| vitest ChatPanel suite (66 tests including ROCHAT-01/02) | `pnpm --dir frontend test -- --run ChatPanel` | 66/66 PASS 1.63s | PASS |
| ErrChatReadOnly completely removed from non-comment Go | `grep -rn "ErrChatReadOnly" internal/ --include="*.go" \| grep -v hub.go` | (empty) | PASS |
| "Read only" text absent from ChatPanel.tsx | `grep -c "Read only" frontend/src/components/Hub/ChatPanel.tsx` | 0 | PASS |
| Traceability path check | `bash tests/check-traceability-paths.sh` | "OK: all traceability paths exist" | PASS |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | — | — | — | — |

No debt markers (TBD/FIXME/XXX), placeholder text, or stub implementations found in the 9 files modified by this phase.

### Security Gate Verification (SEC-RO-01)

Per the verification instructions, SEC-RO-01 requires proof that ONLY MsgChatSend was loosened. The following file:line citations establish this:

**Gate removed (hub.HandleChatSend):**

- `internal/relay/hub.go` lines 661-691: `HandleChatSend` function has no `sub.ReadOnly` check. The function proceeds directly from `SanitizeChatContent` to `chatAppendFn` persist to `BroadcastChat`. No conditional on `ReadOnly`.
- Doc comment at line 649-660 explicitly states the ErrChatReadOnly gate was removed to honor D-06.

**Gates confirmed intact:**

- `internal/relay/hub.go` lines 585-587: `HandleInject` has `if sub.ReadOnly { return ErrReadOnly }` — inject gate intact.
- `internal/relay/server.go` line 343: `if !sub.ReadOnly {` — MsgInput discard intact on relay path.
- `internal/webserver/server.go` line 1190: `if !sub.ReadOnly {` — MsgInput discard intact on webserver path.

**Behavioral proof:** `TestHandleChatSend_ROCanPostInjectStillGated` in `internal/relay/hub_chatsend_test.go` proves all three invariants atomically in a single test under `-race`: (a) RO HandleChatSend returns nil, (b) HandleInject returns ErrReadOnly, (c) PTY write count = 0.

**Note:** `/gsd-secure-phase 163` was not separately invoked. The plans include STRIDE threat registers (T-163-01..T-163-06). Human sign-off is requested to confirm this is acceptable.

### Human Verification Required

#### 1. Playwright ROCHAT-01/02 E2E Test (Live Daemon)

**Test:** Start AgentHub daemon, open a web-share session as an RO guest (cap without write perm), navigate to the chat panel, verify the Send button is NOT disabled, type a unique test message, press Enter, confirm the message appears in the chat thread.

**Expected:** The RO guest's message appears in the thread within 3 seconds. The Send button has no disabled state. No MsgInjectError NAK appears.

**Why human:** `frontend/e2e/chat-parity.spec.ts` ROCHAT-01/02 test was rewritten but the Playwright suite requires a live daemon + fixture session. Cannot auto-run without the full daemon stack. The 163-02 SUMMARY documents this as `human_judgment: true / status: unknown`.

#### 2. Security Review Record (SEC-RO-01)

**Test:** Confirm that the STRIDE threat registers in the three plan files (T-163-01..T-163-06) are accepted as the security review record for this phase, or run `/gsd-secure-phase 163` to produce a separate secure-phase artefact.

**Expected:** Either a secure-phase artefact exists, or the developer explicitly accepts the embedded threat model as sufficient for a targeted single-gate loosening change where code evidence fully proves the scope.

**Why human:** The code evidence for SEC-RO-01 is complete and verifiable (all file:line citations above). This is a process question — whether a separate `/gsd-secure-phase` run is required in addition to the embedded threat model.

---

## Summary

Phase 163 goal is **achieved in the codebase**. All 10 must-have truths verify against actual code and passing tests:

- Server-side gate: `hub.HandleChatSend` (hub.go:661-691) has no `sub.ReadOnly` branch. `ErrChatReadOnly` symbol is fully removed from all non-comment Go code. Both relay and webserver MsgChatSend dispatch delegate to `hub.HandleChatSend` with no local RO gate. Three Go integration tests confirm RO send works on relay path, webserver path (both perm shapes), and the hub unit level.

- Inject and PTY gates intact: `hub.HandleInject` at hub.go:585-587 returns `ErrReadOnly` for RO; relay/server.go:343 and webserver/server.go:1190 both discard `MsgInput` for RO. `TestHandleChatSend_ROCanPostInjectStillGated` proves both invariants in one test under -race. Standing guards (`TestInjectRO_WebPath`, `TestInject_ROCap_RelayPath`, `TestSecurity_ReadOnlyCapabilityBlocksMsgInput`) pass untouched.

- Frontend: ChatPanel.tsx Send button (`disabled={!draft.trim()}`) has no `isReadOnly` in the expression. `handleInjectPointerDown` retains `if (isReadOnly) return`. Five new vitest tests in the ROCHAT-01/02 block all pass (66/66 suite green).

- Presence/typing: MsgTyping and MsgAliasSet in both server files are explicitly NOT gated on `sub.ReadOnly` (relay/server.go:357-364, webserver/server.go:1203-1210) — RO clients can emit presence and typing indicators unchanged.

- Documentation: PITFALLS.md Pitfall 1 reconciled to D-06 rule with Phase 163 citation. TESTING.md Section 2 Phase 163 delta note and Section 4 ROCHAT-01/ROCHAT-02/SEC-RO-01 traceability rows added. `tests/check-traceability-paths.sh` exits 0.

One gap flagged as WARNING (non-blocking): ROCHAT-01, ROCHAT-02, SEC-RO-01 requirement IDs are in PLAN frontmatter and TESTING.md but not in `.planning/REQUIREMENTS.md`. The TESTING.md traceability is complete; the milestone REQUIREMENTS.md does not reflect the D-06 reconciliation.

Two human verification items remain: the Playwright ROCHAT-01/02 E2E test (requires live daemon) and SEC-RO-01 security review record confirmation.

---

_Verified: 2026-06-28T23:45:00Z_
_Verifier: Claude (gsd-verifier)_
