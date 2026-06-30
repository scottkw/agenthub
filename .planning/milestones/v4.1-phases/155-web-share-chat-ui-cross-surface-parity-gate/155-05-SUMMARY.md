---
phase: 155-web-share-chat-ui-cross-surface-parity-gate
plan: "05"
subsystem: e2e-parity-gate
gap_closure: true
closes: BLOCKER-1
tags: [parity, broadcast, websocket, e2e, playwright]
dependency_graph:
  requires: [155-04]
  provides: [PARITY-01-SC-1-broadcast, PARITY-01-SC-1-unread-badge]
  affects: [155-VERIFICATION]
tech_stack:
  added: []
  patterns:
    - subscriber-registration gate via /__test__/hub-status polling
    - sequential browser-context open for WebKit TLS budget
key_files:
  created: []
  modified:
    - frontend/e2e/chat-parity.spec.ts
decisions:
  - "Root cause classified as subscriber-registration race (hypothesis 1), not channel drop or client render suppression, based on live hub-status + WS-frame evidence"
  - "Fix placed in test spec (waitForHubSubscribers polling gate) per plan guidance: gate on subscriberCount reaching expected value rather than fixed sleep"
  - "Sequential page1-first open pattern adopted for WebKit TLS budget (mirrors unread-badge test pattern already proven on webkit)"
  - "history readiness check uses .first() (any message) instead of hasText filter: filter was slow on webkit (> 15s), .first() resolves in < 1s; subscriber gate provides definitive WS-ready signal"
metrics:
  duration: "~22 minutes"
  completed: "2026-06-27"
  tasks: 3
  files: 1
status: complete
---

# Phase 155 Plan 05: Broadcast Gap Closure (BLOCKER 1) Summary

**One-liner:** Fixed PARITY-01 SC-1 broadcast non-delivery via subscriber-registration race gate using `/__test__/hub-status` polling — 6/6 tests now pass on chromium/firefox/webkit.

## Objective

Close BLOCKER 1 from 155-VERIFICATION: chat messages sent by web-share RW client A never reached RW client B within the 10s window (all 3 browsers). Structured as DIAGNOSE → FIX → PROVE.

---

## Task 1: DIAGNOSE — Root Cause Evidence

### Captured Evidence

**(a) subscriberCount at send time:** Non-deterministic across runs:
- Run 1 (original instrumentation): `subscriberCount: 1` before send, `3` after page1 echo
- Run 2 (connection-indexed instrumentation): `subscriberCount: 0` before send, `2` after page1 echo

The variation (0, 1, 2, 3) proves a timing race — the count is not stable.

**(b) byte=48 (MSG_CHAT) frame presence in page2WsFrames:**
- Run 1: **PRESENT** — 1 frame received on `conn=1` (TerminalPanel WS) only; 0 frames on `conn=2` (ChatPanel WS)
- Run 2: **ABSENT** — 0 byte=48 frames on either connection

page2 has exactly **2 WS connections** (conn=1=TerminalPanel, conn=2=ChatPanel — confirmed by `page2ConnCount.total=2`).

**(c) page2WsClosed events:** **EMPTY** (no WS close events on page2 in either run) — CloseSlow was NOT triggered.

### Root Cause Classification

**Root cause: (1) subscriber-registration race**

`hub.Subscribe(sub)` is called inside the Go `handleWS` goroutine, which is scheduled AFTER the HTTP 101 Switching Protocols response is sent to the browser. The history endpoint (`/api/chat/{id}/history`) runs in a SEPARATE HTTP goroutine. The test's readiness gate — waiting for history to be visible — proves `onOpen` fired and `loadChatHistory` completed via HTTP, but the HTTP response can arrive BEFORE `handleWS` reaches `hub.Subscribe`. Result: when `BroadcastChat` fires, page2's ChatPanel WS subscriber (and sometimes its TerminalPanel WS subscriber) is not yet registered in the hub.

Evidence:
- (a) subscriberCount < 4 at send time (expected 4 for 2 pages × TerminalPanel + ChatPanel)
- (b) In Run 1, the one byte=48 frame was received on conn=1 (TerminalPanel — no `onChat` callback → silently discarded); conn=2 (ChatPanel) was not subscribed at BroadcastChat time
- (b) In Run 2, neither connection was subscribed at BroadcastChat time
- (c) No CloseSlow — the Msgs channel was NOT full; the frame was simply never sent to page2's ChatPanel subscriber because it wasn't in `h.subscribers` yet

The fix: gate the send on `subscriberCount >= 4` (all 4 subscribers registered) before triggering BroadcastChat.

---

## Task 2: FIX — Targeted Implementation

**File modified:** `frontend/e2e/chat-parity.spec.ts` (the only file implicated by the diagnosis)

**No Go server changes.** No new packages. No go.mod or package.json additions.

### Changes

1. **`waitForHubSubscribers(adminURL, minCount, timeoutMs=5000)` helper** (new module-level function):
   - Polls `/__test__/hub-status` every 100ms until `subscriberCount >= minCount`
   - Throws with a clear message if deadline exceeded
   - Timeout 5s (well under the test's 10s window)

2. **Broadcast test (`PARITY-01 SC-1 — message broadcast`):**
   - Calls `await waitForHubSubscribers(env.adminURL, 4)` after both pages have loaded their first chat message and before `page1.keyboard.press('Enter')`
   - Also resequenced to sequential open (page1 first, then page2) and changed history check from `filter({ hasText: '...' })` to `.first()` (see Deviation 1 below)

3. **Unread badge test (`PARITY-01 SC-1 — unread badge`):**
   - Calls `await waitForHubSubscribers(env.adminURL, 4)` after the page2 open-close warmup and before page1 sends

### Verification

- `go build ./...` → passes (no Go changes)
- `pnpm -C frontend exec tsc --noEmit` → passes
- RO gate (`hub.go:585 ErrChatReadOnly`) — **unchanged**
- BroadcastChat per-session fan-out (`h.subscribers` of THIS hub) — **unchanged**
- No new packages added

---

## Task 3: PROVE — Live Test Results

```
Running 6 tests using 1 worker

[broadcast-diag] hub status before send: {"subscriberCount":4}  ← all 4 registered ✓
[broadcast-diag] hub status after page1 echo: {"subscriberCount":4}
  ✓  1 [chromium] — message broadcast between two RW web-share clients (2.3s)
  ✓  2 [chromium] — unread badge appears on Page2 when Page1 sends while chat is closed (1.8s)
  ✓  3 [firefox]  — message broadcast between two RW web-share clients (3.1s)
  ✓  4 [firefox]  — unread badge appears on Page2 when Page1 sends while chat is closed (2.3s)
  ✓  5 [webkit]   — message broadcast between two RW web-share clients (2.9s)
  ✓  6 [webkit]   — unread badge appears on Page2 when Page1 sends while chat is closed (2.3s)

  6 passed (22.1s)
```

**All 6 instances (2 tests × 3 browsers) PASS.** BLOCKER 1 is closed.

The 10-second `toBeVisible` window and the `.chat-badge` visibility assertion were NOT weakened.

---

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] WebKit broadcast test fails at history wait with text filter**

- **Found during:** Task 3 (PROVE) — first attempt
- **Issue:** `await expect(page1.locator('.chat-msg').filter({ hasText: 'Hello from the fixture (RW)' })).toBeVisible({ timeout: 15_000 })` failed on webkit even with 15s timeout. The simultaneous opening of two browser contexts (ctx1 + ctx2) with concurrent TLS handshakes exhausted webkit's budget for page1's history load.
- **Fix:**
  1. Resequenced to open page1's chat first, wait for page1 history, then open page2's chat (reduces concurrent TLS load — mirrors the unread-badge pattern that already passes on webkit)
  2. Changed history check from `filter({ hasText: '...' })` to `.first()` (any message) — the specific-text filter was slow on webkit even with 10s/15s; `.first()` resolves in < 1s; the `waitForHubSubscribers` gate provides the definitive WS-ready signal
- **Files modified:** `frontend/e2e/chat-parity.spec.ts`
- **Commit:** 4ec4e88c

### No architectural changes, no RO gate changes, no cross-session isolation changes.

---

## Key Links Verified (Post-Fix)

| From | To | Via | Status |
|------|----|-----|--------|
| `page1.keyboard.press('Enter')` | `hub.BroadcastChat` | chatAppendFn → BroadcastChat | VERIFIED (hub-status=4 at send time) |
| `hub.BroadcastChat` | page2 ChatPanel WS | `sub.Msgs` channel → write pump | VERIFIED (byte=48 received on conn=2 post-fix) |
| `RelayClient MSG_CHAT (0x30)` | `ChatPanel.handleChat` | `onChat` callback → `setMessages` | VERIFIED (message appears in DOM) |

## Security Posture (unchanged)

- **T-155-05-01 (broadcast fan-out scope):** `BroadcastChat` iterates only `h.subscribers` of THIS hub — no cross-session delivery possible. UNCHANGED.
- **T-155-05-02 (RO gate):** `hub.go:585 ErrChatReadOnly` — UNCHANGED. No edits to hub.go.
- **T-155-05-03 (CloseSlow policy):** Preserved — a genuinely slow subscriber still gets closed. The fix adds no exception to CloseSlow.
- **T-155-05-SC (supply chain):** No new packages added to go.mod or frontend/package.json.

## Commits

| Hash | Description |
|------|-------------|
| 77a63abb | test(155-05): add connection-indexed WS diagnostics for broadcast root-cause diagnosis |
| 0d7340d6 | fix(155-05): gate SC-1 broadcast/badge send on hub subscriber count >= 4 |
| 4ec4e88c | test(155-05): SC-1 broadcast + unread-badge green on chromium/firefox/webkit — 6/6 pass |

## Self-Check

- frontend/e2e/chat-parity.spec.ts modified: confirmed (git log shows 3 commits)
- BLOCKER 1 closed: confirmed (6/6 live Playwright tests pass)
- RO gate unchanged: confirmed (hub.go not modified, grep for ErrChatReadOnly shows line 585 unmodified)
- No new packages: confirmed (go.mod and package.json unmodified)

---

## Server-side broadcast race fix (post-sign-off)

The Playwright-side fix above closed the *test flake* by gating the SC-1 send on a hub subscriber count, but the underlying SERVER race remained. With explicit user sign-off, the real server-side fix landed.

**Root cause (WhoIs window).** `handleWSSRelay` (`internal/webserver/server.go`) called `websocket.Accept` (sends the 101 → client `onOpen` fires) and only THEN performed `lc.WhoIs(ctx, r.RemoteAddr)` — a Tailscale identity IPC with real production latency — before building `sub` and calling `hub.Subscribe`. Chat history is served by a SEPARATE HTTP endpoint (`GET /api/chat/{id}/history`), so a joining viewer B could render history (proving its socket open client-side) while the server was still blocked inside `WhoIs` and had NOT yet added B to the broadcast fan-out set. If user A sent a chat message in that window, `BroadcastChat` fanned out to current subscribers only — B was not one yet, B's history fetch had already returned, so B never saw the message until reload. The old `// Subscribe FIRST — anti-race pattern` comment was inaccurate: `WhoIs` preceded `Subscribe`.

**Fix (two-phase subscribe).** Register the subscriber in the broadcast set IMMEDIATELY after `Accept`, BEFORE `WhoIs`:

1. Build `sub` with WhoIs-independent fields only (`Msgs`, `ReadOnly`, `Name`, `Origin:"web"`, `CloseSlow`, `AliasSetFn`); leave `TailnetID`/`PersonKey`/`Alias` empty.
2. `hub.Subscribe(sub)` at once — delivery is live; empty `PersonKey` ⇒ no presence-roster entry yet.
3. Register `defer hub.Unsubscribe(sub)` + `defer conn.CloseNow()` (Unsubscribe is symmetric: empty PersonKey on a drop inside the window cleanly removes from subscribers, skips the roster).
4. THEN do `WhoIs`, compute `tailnetID`/`personKey`/`alias`, assign them to `sub`.
5. THEN call the new `hub.RegisterPresence(sub)` (presence-roster registration only — the former `Subscribe` `if PersonKey != ""` block, ref-counting `ConnCount`), followed by `NotifyViewerCount` + `NotifyPresence`.

**Why race-free.** `BroadcastChat` fan-out reads only `sub.Msgs` (immutable). The presence snapshot `CurrentPresence` reads roster entries — a `presenceState` COPY made under lock at `RegisterPresence` — not the live `sub`. The read pump that reads `sub`'s identity fields starts only AFTER identity is set. So no concurrent reader observes the identity fields while they transition empty → resolved. The relay loopback path (`internal/relay/server.go`) is UNTOUCHED: it sets identity before `Subscribe` and registers presence in one shot. The read-only capability gate (signed-cap–derived `readonly`) is unchanged.

**New hub method.** `Hub.RegisterPresence(sub *Subscriber)` (`internal/relay/hub.go`) — performs only the presence-roster registration `Subscribe` used to do inline, for the two-phase web path (subscribe-early, identity-later). No-op when `PersonKey` is empty.

**Tests.** `internal/relay/hub_subscribe_race_test.go` (3 tests, run under `-race`): `TestBroadcastDeliversBeforeIdentity` (subscriber added before identity still receives a broadcast frame; roster empty during the window, correct entry after `RegisterPresence`), `TestRegisterPresenceRefCounts` (ConnCount increments for a second conn sharing PersonKey), `TestUnsubscribeEmptyPersonKeyIsClean` (drop inside the WhoIs window is clean).

**Verification.** `go test -race ./internal/relay/... ./internal/webserver/...` PASS; `go build ./...` PASS; `go vet ./internal/relay/... ./internal/webserver/...` clean. The Playwright `waitForHubSubscribers` deterministic backstop is retained.

| Hash | Description |
|------|-------------|
| 90bbbcab | test(155-05): add failing two-phase subscribe race test |
| 84a49c66 | fix(155-05): close WhoIs-window broadcast race with two-phase subscribe |
