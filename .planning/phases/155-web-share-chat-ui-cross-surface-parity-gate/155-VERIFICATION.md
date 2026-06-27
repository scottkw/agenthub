---
phase: 155-web-share-chat-ui-cross-surface-parity-gate
verified: 2026-06-26T00:00:00Z
status: gaps_found
score: 2/4 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification: false
gaps:
  - truth: "A participant on the web-share browser sees the same thread, presence indicators, typing indicators, unread badge, and @mention highlights as the desktop GUI owner — no feature drift between surfaces verified by Playwright e2e (PARITY-01 SC-1)"
    status: failed
    reason: "Live Playwright e2e fails on all 3 browsers (chromium, firefox, webkit). The broadcast path — chatAppendFn → hub.BroadcastChat → webserver WS write pump → client B RelayClient MSG_CHAT → onChat → setMessages — does not deliver messages sent by RW client A to RW client B. The unread badge test fails for the same root cause (badge depends on broadcast delivery). 8 out of 24 total tests failed."
    artifacts:
      - path: "internal/relay/hub.go"
        issue: "BroadcastChat fans out to h.subscribers — code is present and wired; the live e2e proves end-to-end delivery fails despite the implementation"
      - path: "frontend/e2e/chat-parity.spec.ts"
        issue: "PARITY-01 SC-1 broadcast test and unread badge test both fail on chromium, firefox, and webkit — 6 failures from these two tests alone"
      - path: "cmd/playwright-fixture/main.go"
        issue: "chatAppendFn wired before ws.Start() and hub-status diagnostic endpoint confirms wiring — the failure is at runtime delivery, not at wiring; exact failure mode (subscriber timing race, slow-subscriber channel drop, or write-pump delivery) requires live diagnostic output to pinpoint"
    missing:
      - "Diagnose root cause of broadcast non-delivery: inspect the hub-status diagnostic output from the failing test run (subscriberCount at send time, chatAppendFnWired flag), check whether both page1 and page2 WS connections are registered as subscribers before the send fires, and whether BroadcastChat's non-blocking channel send is dropping frames for page2 (slow subscriber / full Msgs channel)"
      - "Fix the broadcast delivery path so that a message sent by client A appears on client B within the 10s timeout on all 3 browsers"
      - "Re-run `pnpm -C frontend exec playwright test chat-parity` after fix and confirm all 8 broadcast-dependent tests pass (SC-1 broadcast + SC-1 unread badge, all 3 browsers)"

  - truth: "A RO-cap web-share viewer cannot post messages or trigger @session injection — the server rejects both actions regardless of client behavior (PARITY-01 SC-3)"
    status: failed
    reason: "SC-3 Playwright test passes on chromium but fails on firefox and webkit. The test waits (up to 8s) for the seeded history message 'Hello from the fixture (RW)' to appear before the adversarial send step; on firefox and webkit the history never renders within the 8s window. This is a separate failure from the broadcast issue — it indicates the chat history fetch or WS connection setup is too slow on those browsers, or the ChatPanel's loadChatHistory-in-onOpen timing produces a race on slower TLS handshakes. The server gate code (HandleChatSend returning ErrReadOnly, HandleInject returning ErrReadOnly) is present and correct; the client-side isReadOnly suppression is implemented and works on chromium. The e2e (the authoritative parity gate) is RED on 2 of 3 browsers."
    artifacts:
      - path: "frontend/e2e/chat-parity.spec.ts"
        issue: "PARITY-01 SC-3 test: expect(page.locator('.chat-msg').filter({ hasText: 'Hello from the fixture (RW)' })).toBeVisible({ timeout: 8_000 }) times out on firefox and webkit before reaching the adversarial send assertion"
      - path: "frontend/src/components/Hub/ChatPanel.tsx"
        issue: "loadChatHistory is called inside the RelayClient onOpen callback — the history fetch initiates only after the WS connection is established; on slower browsers the WS handshake over WSS (self-signed cert) may take long enough that the 8s total window is insufficient for WS open + TLS handshake + history fetch + React render"
    missing:
      - "Increase the seeded history wait timeout in the SC-3 test to 15s or add an explicit wait for the WS connection before starting the timer, so firefox and webkit have enough time for WSS handshake + history fetch"
      - "Alternatively: add a pre-wait step in the SC-3 test (open/close chat to confirm WS ready) analogous to the unread badge test's pre-wait pattern"
      - "Re-run SC-3 on all 3 browsers after the fix and confirm it passes"
---

# Phase 155: Web-Share Chat UI + Cross-Surface Parity Gate — Verification Report

**Phase Goal:** The web-share surface delivers the identical chat experience and Markdown export is available on both surfaces.
**Verified:** 2026-06-26
**Status:** gaps_found (BLOCKER)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | PARITY-01 SC-1: Web-share browser sees same thread, presence, unread badge, @mention as desktop — verified by Playwright e2e | FAILED (BLOCKER) | Live Playwright e2e: broadcast test FAILS on chromium/firefox/webkit (message sent by client A never arrives at client B); unread badge test FAILS on all 3 browsers (same root cause). 6 of 8 broadcast-related failures. |
| 2 | EXPORT-01 SC-2: Export downloads a `.md` with YAML frontmatter + full thread from both surfaces | VERIFIED | Playwright export e2e PASSES on all 3 browsers (`[data-chat-export]` → download → frontmatter present). Go unit tests `TestChatStore_Export`, `TestChatRoutes_Export`, `TestChatExport` all pass. `Export()` implementation in `internal/daemon/chat.go` is substantive and correct. |
| 3 | PARITY-01 SC-3: RO-cap viewer cannot post or inject — server rejects regardless of client behavior | FAILED (BLOCKER) | Playwright SC-3 PASSES on chromium only; FAILS on firefox + webkit. Failure mode: seeded history message never renders within 8s on those browsers — test times out before the adversarial send step. Code is present and correct (isReadOnly, data-chat-send, HandleChatSend ErrReadOnly gate) but the authoritative e2e gate is RED on 2/3 browsers. |
| 4 | PARITY-01 SC-4: @session inject indicator (.chat-msg--inject) renders identically from both surfaces | VERIFIED | Playwright SC-4 PASSES on all 3 browsers. Seeded `SessionInject:true` message in fixture renders with `.chat-msg--inject` class in ChatPanel. |

**Score:** 2/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/daemon/chat.go` | `Export()` rewritten with YAML frontmatter | VERIFIED | Substantive implementation present (lines 337–377): YAML frontmatter fence, session/exported_at/participants fields, per-message headers, SessionInject marker. |
| `internal/daemon/chat_test.go` | `TestChatStore_Export` (5 sub-tests) | VERIFIED | Present at line 649. All 5 sub-tests: EmptyThread, SingleMessage, DeduplicatedParticipants, SessionInjectMarker, YAMLSpecialCharInAlias. |
| `internal/daemon/chat_routes_test.go` | `TestChatRoutes_Export` | VERIFIED | Present at line 213. Asserts frontmatter + Content-Disposition. |
| `internal/webserver/chat_test.go` | `TestChatExport` (valid cap + unauthorized) | VERIFIED | Present at line 264. Asserts 200 + frontmatter on valid cap; 401 + no thread bytes on missing cap (T-155-03). |
| `frontend/src/lib/relayClient.ts` | `wsURL?: string` override in constructor opts | VERIFIED | Line 215: `if (opts?.wsURL)` branch evaluates before `127.0.0.1:${port}` is built. |
| `frontend/src/components/TerminalPanel.tsx` | `wsURL?: string` prop threaded to RelayClient | VERIFIED | Props extended; threaded into `new RelayClient(..., { remote, wsURL })`. |
| `frontend/src/components/Hub/ChatPanel.tsx` | `wsURL?`/`apiBaseURL?`/`capToken?` props; Export button; isReadOnly suppression | VERIFIED | All three optional props present; `buildExportURL()` + `triggerExport()` + `[data-chat-export]` button added; `isReadOnly` state from `/info` perms with fail-safe default. |
| `frontend/src/components/Hub/WebShareSessionView.tsx` | New wrapper component with hub-modal parity classes | VERIFIED | Exists; constructs `wss://{host}/sessions/{id}/ws?cap=` and passes wsURL to both TerminalPanel and ChatPanel; uses verbatim `hub-modal__body--interactive` and `hub-modal__chat-toggle` class names. |
| `frontend/src/components/Hub/WebShareSessionView.test.tsx` | 11 vitest tests proving wsURL/cap wiring | VERIFIED | Exists; 11 tests: URL shape, forwarding to both children, cap encoding, parity class names, chat toggle interaction. |
| `frontend/src/App.tsx` | `openWebSessionTab` helper + web-mode bootstrap + render branch | VERIFIED | `openWebSessionTab` present; bootstrap opens file tab first then session tab (active); render branch on `__websession__${sessionId}` mounts `WebShareSessionView`. |
| `frontend/e2e/chat-parity.spec.ts` | 8 Playwright tests covering PARITY-01 + EXPORT-01 | EXISTS — PARTIALLY FAILS | Substantive spec with 8 tests across frozen UI-SPEC §5 selectors. 16/24 tests pass (8 fail); the 3 broadcast/unread-badge tests fail on all browsers; SC-3 fails on 2 browsers. |
| `cmd/playwright-fixture/main.go` | ChatStore + SetChatHistoryProvider + SetChatExportProvider + chatAppendFn wired | VERIFIED | All providers wired (lines 233–258); `hub.SetChatAppendFn` called before `ws.Start()`; hub-status diagnostic endpoint present (line 358). |
| `TESTING.md` | Section 2 + Section 4 PARITY-01/EXPORT-01/NOTIF-02 rows; Section 5 Phase 155 UAT | VERIFIED | Section 2: 500 total tests logged; Section 4: EXPORT-01 (3 Go rows + 1 Playwright row), PARITY-01 (vitest + Playwright rows), NOTIF-02 row. Section 5: M-24 manual UAT item added. `bash tests/check-traceability-paths.sh` exits 0. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `hub.HandleChatSend` | `hub.BroadcastChat` | `chatAppendFn` → persist → `BroadcastChat(MakeChatFrame(msg))` | WIRED (code) — NOT DELIVERING (live) | Code wiring is correct and present (hub.go:596–615). Fixture wires chatAppendFn before ws.Start. BUT live Playwright e2e: message sent by client A never appears on client B. BroadcastChat fans out to all `h.subscribers`; the failure is at runtime (possible: subscriber not registered yet, slow-subscriber channel drop, write-pump delivery failure). |
| `webserver WS write pump` | client B browser | binary WebSocket frame (`sub.Msgs` channel) | WIRED (code) — NOT VERIFIED (live) | The write pump exists and reads from `sub.Msgs`. Live e2e proves the frame never reaches client B's RelayClient. |
| `RelayClient MSG_CHAT (0x30)` | `ChatPanel.handleChat` | `onChat` callback → `setMessages` | VERIFIED | relayClient.ts:167–175 parses MSG_CHAT; line 418 wires `onChat: handleChat`; `handleChat` (line 396) calls `setMessages`. Wiring is correct at the client level. |
| `WebShareSessionView wsURL` | `TerminalPanel` + `ChatPanel` | props drilling | VERIFIED | Both children receive `wsURL`; vitest proves forwarding (11 tests in WebShareSessionView.test.tsx). |
| `ChatPanel.loadChatHistory` | `/api/chat/{id}/history?cap=` | `apiBaseURL ?? loopback` + `?cap=${encodeURIComponent(capToken)}` | VERIFIED | Lines 390–391 in ChatPanel.tsx build the URL correctly; Pitfall 2 (missing ?cap= → 401) prevented. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Playwright parity gate | `pnpm -C frontend exec playwright test chat-parity` (run by orchestrator) | 16/24 passed, 8 failed | FAIL |
| SC-1 broadcast (chromium) | playwright test run | FAILED | FAIL |
| SC-1 broadcast (firefox) | playwright test run | FAILED | FAIL |
| SC-1 broadcast (webkit) | playwright test run | FAILED | FAIL |
| SC-1 unread badge (chromium) | playwright test run | FAILED | FAIL |
| SC-1 unread badge (firefox) | playwright test run | FAILED | FAIL |
| SC-1 unread badge (webkit) | playwright test run | FAILED | FAIL |
| SC-3 RO gate (chromium) | playwright test run | PASSED | PASS |
| SC-3 RO gate (firefox) | playwright test run | FAILED (history load timeout) | FAIL |
| SC-3 RO gate (webkit) | playwright test run | FAILED (history load timeout) | FAIL |
| EXPORT-01 SC-2 download (all browsers) | playwright test run | PASSED (3/3 browsers) | PASS |
| SC-4 inject indicator (all browsers) | playwright test run | PASSED (3/3 browsers) | PASS |
| SC-1 presence roster (all browsers) | playwright test run | PASSED (3/3 browsers) | PASS |
| SC-1 @mention render (all browsers) | playwright test run | PASSED (3/3 browsers) | PASS |
| SC-1 typing slot (all browsers) | playwright test run | PASSED (3/3 browsers) | PASS |
| Go export unit tests | `go test ./internal/daemon/... -run TestChatStore_Export -count=1` | PASSED (SUMMARY-01) | PASS |
| Go route tests | `go test ./internal/daemon/... -run TestChatRoutes_Export && go test ./internal/webserver/... -run TestChatExport` | PASSED (SUMMARY-01) | PASS |
| TypeScript compilation | `pnpm -C frontend exec tsc --noEmit` | PASSED (SUMMARY-02/03) | PASS |
| Vitest suite | `pnpm -C frontend test run src/components/Hub/` | 344/344 (SUMMARY-03) | PASS |
| Go build | `go build ./...` | PASSED (SUMMARY-04) | PASS |
| Traceability check | `bash tests/check-traceability-paths.sh` | exit 0 (SUMMARY-04) | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| EXPORT-01 | 155-01, 155-02, 155-04 | User can download chat thread as Markdown from both desktop and web-share surface | VERIFIED | Go unit tests pass; Export() serializes YAML frontmatter; Playwright export download passes all 3 browsers |
| PARITY-01 | 155-02, 155-03, 155-04 | Every Session Chat feature behaves identically on desktop and web-share browser — release-blocking | FAILED (BLOCKER) | Live Playwright gate FAILS: broadcast (SC-1) fails all 3 browsers; unread badge (SC-1) fails all 3 browsers; SC-3 fails 2/3 browsers. PARITY-01 is explicitly release-blocking per the standing cross-surface parity rule and REQUIREMENTS.md. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.planning/phases/155-web-share-chat-ui-cross-surface-parity-gate/155-04-SUMMARY.md` | "Live Playwright e2e deferred to orchestrator" | E2e gate not run by executor | Warning (informational) | Executor correctly deferred due to prior hang; this is not a code anti-pattern but is why the gate failure was not caught at execution time. No TBD/FIXME/XXX markers found in phase-modified files. |

No `TBD`, `FIXME`, or `XXX` markers found in the files modified by this phase.

## Gaps Summary

**Two blockers prevent the phase goal from being achieved:**

**BLOCKER 1 — Broadcast non-delivery (PARITY-01 SC-1)**

The central PARITY-01 promise is that a message sent by web-share client A appears in web-share client B's thread. The live Playwright e2e proves this does not happen — the test fails on all three browsers (chromium, firefox, webkit). The broadcast code path is correctly wired in the Go source (`HandleChatSend` → `chatAppendFn` → `BroadcastChat` → `sub.Msgs`) and the TypeScript source (`RelayClient onChat` → `ChatPanel handleChat` → `setMessages`). The failure is at runtime: the actual frame never reaches the page2 WebSocket client within the 10s window. The diagnostic endpoint (`/__test__/hub-status`) was instrumented for exactly this failure mode; the actual diagnostic output from the failing run is needed to pinpoint whether the root cause is a subscriber-registration timing race, a slow-subscriber channel drop, or a write-pump delivery failure.

The unread badge test (SC-1) fails for the same root cause — Page2 must receive the broadcast message to trigger the unread badge.

**BLOCKER 2 — SC-3 history load timeout on Firefox and WebKit (PARITY-01 SC-3)**

The RO gate test waits for seeded history to appear (to confirm the WS connection is fully established) before the adversarial send step. On Firefox and WebKit, the seeded `Hello from the fixture (RW)` message never renders within the 8s window. This indicates the WS+TLS handshake plus history fetch is taking longer than 8s on those browsers against the fixture's self-signed cert. The RO gate code itself (server-side `ErrReadOnly`, client-side `isReadOnly` suppression) is correct and proven on chromium. The fix is to either increase the wait timeout or add a pre-connection step (analogous to the unread badge test's explicit open-then-close warm-up) to ensure the WS is established before starting the timer.

**What is fully working:**
- `Export()` serialization and both export routes (Go unit tests)
- ChatPanel Export button, buildExportURL, triggerExport wiring
- RelayClient `wsURL` override + TerminalPanel + ChatPanel prop threading
- WebShareSessionView component (parity CSS classes, wsURL/cap forwarding, vitest coverage)
- App.tsx web-mode bootstrap (web-session tab, file browser secondary, render branch)
- PARITY-01 SC-4 inject indicator (seeded history, all browsers)
- PARITY-01 SC-1 presence roster, typing slot, @mention highlight (all browsers)
- EXPORT-01 SC-2 download (all browsers)
- TESTING.md fully updated (Section 2/4/5, traceability-path check)

---

_Verified: 2026-06-26_
_Verifier: Claude (gsd-verifier)_
_Authoritative live e2e evidence: orchestrator-run Playwright cross-surface parity gate — 24 tests (chromium/firefox/webkit), 16 passed / 8 failed_
