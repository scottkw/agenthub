---
phase: 153-session-pty-bridge
verified: 2026-06-26T12:00:00Z
status: human_needed
score: 4/4 must-haves verified (2 with portions deferred to Phase 154)
behavior_unverified: 0
overrides_applied: 0
deferred:
  - truth: "The chat thread shows a '→ injected into terminal' indicator for all participants (SC-1 visual half)"
    addressed_in: "Phase 154"
    evidence: "Phase 154 goal: 'fully functional chat panel'; the SessionInject:true flag is broadcast by Phase 153 and Phase 154 renders it. Phase 153-02 PLAN explicitly: 'The visual RENDERING is Phase 154.'"
  - truth: "The @session injection path requires a deliberate confirm step — a single accidental keypress or Enter-on-autocomplete does not trigger a PTY write (SC-4 UI half)"
    addressed_in: "Phase 154"
    evidence: "Phase 154 requirements include MENTION-01 (the @session autocomplete/trigger UX). Phase 153 delivers the structural guarantee (dedicated verb); Phase 154 delivers the press-and-hold or equivalent confirm affordance."
human_verification:
  - test: "Verify the chat thread renders a '→ injected into terminal' indicator for messages with SessionInject:true"
    expected: "When a RW participant uses @session, all chat clients see a visually distinct inject-confirmation indicator (not a plain chat bubble)"
    why_human: "Indicator rendering is Phase 154 chat UI work; SessionInject:true data is wired in Phase 153 but the visual treatment requires Phase 154's component to be implemented and inspected"
  - test: "Verify the @session injection UX requires a deliberate confirm step — not just selecting @session + pressing Enter in the composer"
    expected: "A user cannot accidentally inject into the PTY with a single casual keypress or Enter-on-autocomplete; the client-side affordance must force an explicit additional confirmation before sending MsgSessionInject"
    why_human: "Phase 153 proves only MsgSessionInject writes to PTY (structural guarantee), but whether Phase 154 accidentally sends MsgSessionInject on a normal Enter keypress depends entirely on Phase 154's UX implementation — no backend check can verify this"
---

# Phase 153: @Session PTY Bridge Verification Report

**Phase Goal:** RW-capable participants can inject a prompt into the agent PTY via `@session`; the injection is sanitized and requires deliberate confirmation.
**Verified:** 2026-06-26
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1a | RW-cap holder sending `@session <text>` causes exactly that text (sanitized) to appear in PTY stdin | VERIFIED | `TestInject_RWCap_WritesToPTY` passes: `ptyWriteCount > 0`, content matches `SanitizePTYText(injectText)` |
| 1b | Chat thread broadcasts `SessionInject:true` message to all subscribers (data layer for indicator) | VERIFIED | `TestInject_RWCap_WritesToPTY` asserts `MsgChat` broadcast with `ChatMessage{SessionInject:true}`; rendering deferred Phase 154 |
| 2 | RO-cap holder's `@session` attempt is server-side rejected; zero PTY writes on relay AND web paths | VERIFIED | `TestInject_ROCap_RelayPath` (relay, `?readonly=1`): NAK + 0 writes; `TestInjectRO_WebPath` (web, table-driven over `"read"` AND `"read,files.read"`): NAK + 0 writes in both cases |
| 3 | C0 controls, embedded newlines, escape sequences, bidi overrides stripped at daemon handler; only printable text + one trailing `\n` reaches PTY stdin | VERIFIED | `TestSanitizePTYText` passes 19-case corpus under `-race`: covers null byte, C0 BEL, DEL, C1 NEL/U+0080, CSI clear+color, OSC BEL+ST-terminated, bidi RLO+LRM, LF/CR/CRLF collapse, empty, only-newlines, mixed attack vector |
| 4a | Only `MsgSessionInject` (0x35) triggers a PTY write; `MsgChatSend` (0x31) and stray frames never do | VERIFIED | `TestInject_OnlyDedicatedFrame` passes: `ptyWriteCount == 0` after sending `MsgChatSend` + stray frame |
| 4b | UI confirm affordance prevents accidental Enter-on-autocomplete triggering inject | DEFERRED | Phase 154 (MENTION-01 / @session UX). Phase 153 provides the structural guarantee; the press-and-hold or equivalent gesture is Phase 154 UI work |

**Score:** 4/4 truths verified for Phase 153 scope (SC-1 indicator rendering + SC-4 UI confirm deferred to Phase 154)

### Deferred Items

Items not yet satisfied but explicitly addressed in later milestone phases.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | "→ injected into terminal" visual indicator in chat thread | Phase 154 | Phase 153-02 PLAN: "The visual RENDERING is Phase 154." SessionInject:true is persisted + broadcast; Phase 154 chat UI renders it. |
| 2 | Deliberate UI confirm step (not just Enter-on-autocomplete) for @session inject | Phase 154 | Phase 153-02 PLAN: "There is NO chat composer, trigger affordance, or press-and-hold gesture in this phase — those are Phase 154." MENTION-01 (autocomplete + confirm UX) is a Phase 154 requirement. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/sanitize.go` | SanitizePTYText state machine | VERIFIED | 5-state (normal/escape/CSI/OSC/OSCEscape); strips C0/C1/DEL/CSI/OSC/bidi; CRLF dedup guard |
| `internal/relay/sanitize_test.go` | 19-case SEC-02 corpus | VERIFIED | All categories present and passing under `-race` |
| `internal/relay/protocol.go` (additions) | MsgSessionInject (0x35), MsgInjectError (0x36), InjectPayload, InjectErrorPayload, MakeChatFrame, MakeInjectErrorFrame | VERIFIED | All 6 symbols compile; 0x35/0x36 verified free before allocation |
| `internal/relay/hub.go` (additions) | ErrReadOnly, chatAppendFn, SetChatAppendFn, BroadcastChat, HandleInject | VERIFIED | 5+ grep hits confirmed; no hub.mu held across WriteInput/chatAppendFn/BroadcastChat |
| `internal/relay/server.go` (case) | `case MsgSessionInject:` in read-pump | VERIFIED | Grep returns 1; routes through `hub.HandleInject` — no direct WriteInput in case |
| `internal/daemon/engine.go` (wiring) | `hub.SetChatAppendFn(...)` in CreateSession | VERIFIED | At line ~459, nil-guarded, after chatStore registered; no import cycle (relay never imports daemon) |
| `internal/relay/server_inject_test.go` | 3 relay-path tests | VERIFIED | TestInject_RWCap_WritesToPTY, TestInject_OnlyDedicatedFrame, TestInject_ROCap_RelayPath all pass |
| `internal/webserver/server.go` (case) | `case relay.MsgSessionInject:` in handleWSSRelay | VERIFIED | Grep returns 1 at line 1164 |
| `internal/webserver/server.go` (gate) | `readonly := !capability.HasPerm(claims.Perms, "write")` | VERIFIED | Line 1016; CR-01 fix (fcdfd03a) replaced broken `claims.Perms == "read"` |
| `internal/webserver/inject_test.go` | TestInjectRO_WebPath table-driven over "read" + "read,files.read" | VERIFIED | Table over {browse_off:"read", browse_on:"read,files.read"}; both subtests pass |
| `TESTING.md` | Go count 362, traceability rows for MENTION-02/03/SEC-01/SEC-02 | VERIFIED | Count 362 confirmed; 5 new traceability rows; `bash tests/check-traceability-paths.sh` exits 0 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| relay `case MsgSessionInject:` | `Hub.HandleInject` | `hub.HandleInject(sub, ip.Text)` — no direct WriteInput | WIRED | server.go line 373 |
| webserver `case relay.MsgSessionInject:` | `Hub.HandleInject` | `hub.HandleInject(sub, ip.Text)` — same gate as relay path | WIRED | server.go line 1176 |
| `Hub.HandleInject` | `SanitizePTYText` | `sanitized := SanitizePTYText(text)` — always runs, no bypass | WIRED | hub.go line 472 |
| `Hub.HandleInject` | `Hub.WriteInput` | `h.WriteInput([]byte(sanitized))` | WIRED | hub.go line 473 |
| `Hub.HandleInject` | `chatAppendFn` (daemon `ChatStore.AppendMessage`) | closure captures `*ChatStore`; relay never imports daemon | WIRED | hub.go lines 478–493; engine.go lines 458–461 |
| `engine.go CreateSession` | `hub.SetChatAppendFn` | called after chatStore registered, nil-guarded | WIRED | engine.go line 459 |
| web-path `readonly` | `capability.HasPerm(claims.Perms, "write")` | whole-token semantics; holds for "read" + "read,files.read" | WIRED | server.go line 1016 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `Hub.HandleInject` | `sanitized` text → PTY stdin | `SanitizePTYText(text)` on user-supplied string | Yes — pure transform of real user input | FLOWING |
| `Hub.HandleInject` | `ChatMessage{SessionInject:true}` → broadcast | `chatAppendFn` → `ChatStore.AppendMessage` (engine.go wiring) | Yes — persisted to JSONL, returned with ID/TimestampMs | FLOWING |
| `sub.ReadOnly` (web) | capability gate | `!HasPerm(claims.Perms, "write")` from signed JWT claims | Yes — sourced from verified JWT, not URL params | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| SEC-02 sanitizer corpus (19 cases) | `go test -race -short -run TestSanitizePTYText ./internal/relay/...` | `ok (1.030s)` | PASS |
| Relay inject tests (RW write, dedicated verb, RO relay rejection) | `go test -race -short -run TestInject ./internal/relay/...` | `ok (cached)` | PASS |
| Web-path RO rejection — both "read" and "read,files.read" (CR-01 regression guard) | `go test -race -short -run TestInjectRO_WebPath ./internal/webserver/...` | `ok (3.203s)` | PASS |
| Full project build | `go build ./...` | no output (exit 0) | PASS |
| go vet on modified packages | `go vet ./internal/relay/... ./internal/daemon/... ./internal/webserver/...` | no output (exit 0) | PASS |
| Traceability path check | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist` | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| MENTION-02 | 153-02 | @session injects to PTY; gated RW; sanitized; chat indicator | SATISFIED | HandleInject + BroadcastChat wired; TestInject_RWCap_WritesToPTY; indicator data in SessionInject:true broadcast |
| MENTION-03 | 153-02 | @session requires deliberate confirm; dedicated verb only | SATISFIED (backend) | TestInject_OnlyDedicatedFrame: MsgChatSend/stray = 0 PTY writes; UI confirm UX deferred Phase 154 |
| SEC-01 | 153-02 (relay), 153-03 (web) | RO clients rejected server-side; no PTY writes | SATISFIED | TestInject_ROCap_RelayPath + TestInjectRO_WebPath (table-driven "read"/"read,files.read") |
| SEC-02 | 153-01 | Sanitize C0/C1/CSI/OSC/bidi; exactly one trailing newline | SATISFIED | TestSanitizePTYText 19-case corpus passing under -race |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/relay/hub.go` | 487 | `Content: text` — original (un-sanitized) text persisted and broadcast | Warning (WR-01, OPEN) | Bidi/C0 bytes in chat storage + broadcast; SEC-02 PTY mitigations bypass the chat display surface; does not violate Phase 153 SCs but should be fixed before Phase 154 renders content |
| `internal/relay/hub.go` | 490 | `if err == nil { BroadcastChat(...) }` — persist/broadcast error swallowed silently | Warning (WR-02, OPEN) | Terminal injects after PTY write + append failure with no NAK or log; terminal and chat thread can diverge silently; violates CLAUDE.md "let it crash" / "make beliefs pay rent" |

No `TBD`, `FIXME`, or `XXX` markers found in any modified file. No stub/placeholder patterns detected.

### CR-01 Fix Verification (SEC-01 web-path bypass)

The code review found a blocker: `readonly := claims.Perms == "read"` does not hold when browse is ON (daemon mints `"read,files.read"`). Commit `fcdfd03a` (2026-06-26) fixes this:

- **Before:** `readonly := claims.Perms == "read"` — exact match; `"read,files.read"` evaluates `false` → RO viewer treated as RW
- **After:** `readonly := !capability.HasPerm(claims.Perms, "write")` — whole-token semantics via `strings.Split`; both `"read"` and `"read,files.read"` correctly produce `readonly = true`

`capability.HasPerm` implementation confirmed: splits on `,`, checks each token for exact equality with `"write"` — no substring match risk.

`TestInjectRO_WebPath` is now table-driven over `{"browse_off", "read"}` and `{"browse_on", "read,files.read"}`. Both subtests pass (the `browse_on` subcase was the regression that injected 1 PTY write against the old gate; it is now correctly NAK'd with 0 PTY writes).

### Human Verification Required

#### 1. "→ injected into terminal" indicator rendering in chat thread

**Test:** In Phase 154's chat UI, have a RW participant send `@session run tests`. Observe the resulting message in all connected chat clients.
**Expected:** A visually distinct "→ injected into terminal" indicator (not a plain chat bubble) appears in the thread for all participants, showing the injected text and the author.
**Why human:** The data layer (SessionInject:true broadcast) is wired in Phase 153. Whether Phase 154's chat component actually renders it differently from a regular message requires running the UI and visual inspection.

#### 2. Deliberate confirm step for @session injection

**Test:** In Phase 154's chat UI, type `@session run tests` into the composer. Select `@session` from the autocomplete with arrow keys and press Enter (a normal chat-send action). Then also test the dedicated inject trigger.
**Expected:** A single Enter-on-autocomplete does NOT send a MsgSessionInject frame to the server; only an additional deliberate confirm step (e.g., press-and-hold, explicit modal confirm, or a second keystroke) triggers the inject. The PTY write must not occur on the first Enter.
**Why human:** Phase 153 proves only MsgSessionInject writes to PTY (structural). Whether Phase 154's @session UX accidentally sends MsgSessionInject on a casual Enter press depends entirely on how Phase 154 implements the trigger — not verifiable by backend grep or test.

### Open Review Findings (Non-Blocking for Phase 153 Goal)

These findings from `153-REVIEW.md` remain OPEN and should be resolved before or during Phase 154:

- **WR-01** — Original (un-sanitized) text persisted/broadcast: bidi-override characters that SEC-02 strips from PTY input still reach `chat.jsonl`, all chat clients, and the Markdown export. The Trojan-Source mitigation is defeated on the chat display surface. **Recommended resolution before Phase 154 renders chat content:** sanitize `Content` with at minimum `isBidiOverride` before `chatAppendFn`/`BroadcastChat`, or introduce a `ValidateChatContent` analogous to `ValidateAlias`.

- **WR-02** — Silent post-PTY-write persist/broadcast failure: when `chatStore.AppendMessage` returns `ErrChatCapReached` or `ErrChatMessageTooLarge`, the inject already occurred but no NAK is sent and no log is emitted. Terminal and chat thread diverge silently. **Recommended resolution:** on `fn` error, send a `MakeInjectErrorFrame` NAK to the originating subscriber and/or log server-side; decide explicitly whether PTY write should be gated behind a successful append.

- **IN-02** — Control-only inject text (e.g., `"\x1b[2J"`) sanitizes to a bare `"\n"` and still presses Enter at the PTY. The `ip.Text == ""` guard runs before sanitization. Low-impact (RW clients can already send Enter) but the inject is a no-op command submission.

- **IN-04** — Sanitizer doc comment overstates coverage: DCS/APC/PM/SOS string bodies survive as plaintext (only the two-byte introducer is discarded). No control sequence reaches PTY but the comment needs correction.

### Gaps Summary

No gaps for Phase 153 scope. The backend inject pipeline is fully implemented and tested. The two remaining items (visual indicator, UI confirm UX) are legitimate cross-phase dependencies on Phase 154's chat UI, explicitly acknowledged in the Phase 153-02 plan.

The open code-review findings (WR-01, WR-02) are noteworthy but do not block the phase goal as stated. WR-01 should be addressed before Phase 154 renders chat content from stored inject messages.

---

_Verified: 2026-06-26_
_Verifier: Claude (gsd-verifier)_
