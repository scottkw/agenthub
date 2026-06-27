---
phase: 155-web-share-chat-ui-cross-surface-parity-gate
plan: "06"
subsystem: e2e-parity-gate
gap_closure: true
closes: BLOCKER-2
tags: [parity, read-only-gate, playwright, timing, firefox, webkit]
dependency_graph:
  requires: [155-05]
  provides: [PARITY-01-SC-3-all-browsers, PARITY-01-full-gate]
  affects: [155-VERIFICATION]
tech_stack:
  added: []
  patterns:
    - .first() warm-up gate before adversarial send (replaces text-filter wait unreliable in full suite)
    - sequential context open for webkit TLS budget (carried from 155-05)
key_files:
  created: []
  modified:
    - frontend/e2e/chat-parity.spec.ts
decisions:
  - "SC-3 readiness gate uses .first() (any message) not text-filter: text-filter fails in full suite because broadcast tests add messages that auto-scroll the virtualizer past the seeded row"
  - "Text-filter wait removed entirely (readiness role superseded by .first() warm-up; adversarial assertion unchanged)"
  - "Timeout increased to 15_000 for .first() warm-up to cover worst-case Firefox/WebKit WSS+TLS startup"
metrics:
  duration: "~6 minutes"
  completed: "2026-06-27"
  tasks: 2
  files: 1
status: complete
requirements: [PARITY-01]
---

# Phase 155 Plan 06: SC-3 RO-Gate Timing Fix + Final Parity Gate Summary

**One-liner:** Closed BLOCKER 2 (PARITY-01 SC-3) with a `.first()` warm-up replacing the text-filtered history wait — full 24/24 chat-parity Playwright gate green on chromium/firefox/webkit.

## Objective

Close BLOCKER 2 from 155-VERIFICATION: the SC-3 RO-gate test was timing out on Firefox and WebKit while waiting (8s) for the seeded history message `Hello from the fixture (RW)` to render before the adversarial send step. The server-side `ErrChatReadOnly`/`ErrReadOnly` gate code was correct and working on chromium. The failure was purely connection/handshake timing on slower browsers.

---

## Task 1: FIX — SC-3 RO-gate warm-up for Firefox/WebKit

**File modified:** `frontend/e2e/chat-parity.spec.ts`

### Changes

Added a `.chat-msg.first()` wait with 15 000 ms timeout immediately after the composer textarea becomes visible. This confirms the WSS handshake completed and at least one history message rendered before checking the send button and counting messages for the adversarial assertion.

Removed the previous `filter({ hasText: 'Hello from the fixture (RW)' }).toBeVisible({ timeout: 8_000 })` readiness check (see Deviation 1 below).

**Unchanged semantics:**
- `expect(sendBtn).toBeDisabled()` — client-side RO gate check
- `expect(afterCount).toBe(beforeCount)` — adversarial server-side gate assertion
- `await page.locator('.chat-panel__composer textarea').fill('adversarial-ro-send')` + `Enter` — adversarial attempt
- `await page.waitForTimeout(500)` — dwell time before counting

**No Go files modified. `ErrChatReadOnly` (hub.go:585) and `isReadOnly` (ChatPanel.tsx) untouched.**

### Verification (Task 1 targeted run)

```
Running 3 tests using 1 worker
  ✓  1 [chromium] PARITY-01 SC-3 — RO viewer Send button is disabled and server gate holds (1.2s)
  ✓  2 [firefox]  PARITY-01 SC-3 — RO viewer Send button is disabled and server gate holds (1.7s)
  ✓  3 [webkit]   PARITY-01 SC-3 — RO viewer Send button is disabled and server gate holds (1.5s)

  3 passed (8.8s)
```

---

## Task 2: PROVE — Full chat-parity suite 24/24 + TESTING.md guard

### Full Playwright Gate (authoritative release-blocking evidence)

```
Running 24 tests using 1 worker

[broadcast-diag] hub status before send: {"chatAppendFnWired":true,"hubFound":true,"subscriberCount":4}
[broadcast-diag] hub status after page1 echo: {"chatAppendFnWired":true,"hubFound":true,"subscriberCount":4}
  ✓   1 [chromium] PARITY-01 SC-1 — message broadcast between two RW web-share clients (2.3s)
  ✓   2 [chromium] PARITY-01 SC-1 — presence roster element renders on both clients (479ms)
  ✓   3 [chromium] PARITY-01 SC-1 — unread badge appears on Page2 when Page1 sends while chat is closed (3.4s)
  ✓   4 [chromium] PARITY-01 SC-1 — typing indicator slot is present in the DOM (254ms)
  ✓   5 [chromium] PARITY-01 SC-1 — @mention message renders with .chat-msg--mention class (345ms)
  ✓   6 [chromium] PARITY-01 SC-3 — RO viewer Send button is disabled and server gate holds (975ms)
  ✓   7 [chromium] EXPORT-01 SC-2 — export downloads .md with YAML frontmatter (338ms)
  ✓   8 [chromium] PARITY-01 SC-4 — @session inject indicator (.chat-msg--inject) renders from history (350ms)

[broadcast-diag] hub status before send: {"chatAppendFnWired":true,"hubFound":true,"subscriberCount":4}
[broadcast-diag] hub status after page1 echo: {"chatAppendFnWired":true,"hubFound":true,"subscriberCount":4}
  ✓   9 [firefox]  PARITY-01 SC-1 — message broadcast between two RW web-share clients (2.9s)
  ✓  10 [firefox]  PARITY-01 SC-1 — presence roster element renders on both clients (930ms)
  ✓  11 [firefox]  PARITY-01 SC-1 — unread badge appears on Page2 when Page1 sends while chat is closed (2.9s)
  ✓  12 [firefox]  PARITY-01 SC-1 — typing indicator slot is present in the DOM (519ms)
  ✓  13 [firefox]  PARITY-01 SC-1 — @mention message renders with .chat-msg--mention class (548ms)
  ✓  14 [firefox]  PARITY-01 SC-3 — RO viewer Send button is disabled and server gate holds (1.5s)
  ✓  15 [firefox]  EXPORT-01 SC-2 — export downloads .md with YAML frontmatter (820ms)
  ✓  16 [firefox]  PARITY-01 SC-4 — @session inject indicator (.chat-msg--inject) renders from history (519ms)

[broadcast-diag] hub status before send: {"chatAppendFnWired":true,"hubFound":true,"subscriberCount":4}
[broadcast-diag] hub status after page1 echo: {"chatAppendFnWired":true,"hubFound":true,"subscriberCount":4}
  ✓  17 [webkit]   PARITY-01 SC-1 — message broadcast between two RW web-share clients (2.5s)
  ✓  18 [webkit]   PARITY-01 SC-1 — presence roster element renders on both clients (839ms)
  ✓  19 [webkit]   PARITY-01 SC-1 — unread badge appears on Page2 when Page1 sends while chat is closed (2.8s)
  ✓  20 [webkit]   PARITY-01 SC-1 — typing indicator slot is present in the DOM (481ms)
  ✓  21 [webkit]   PARITY-01 SC-1 — @mention message renders with .chat-msg--mention class (459ms)
  ✓  22 [webkit]   PARITY-01 SC-3 — RO viewer Send button is disabled and server gate holds (1.0s)
  ✓  23 [webkit]   EXPORT-01 SC-2 — export downloads .md with YAML frontmatter (798ms)
  ✓  24 [webkit]   PARITY-01 SC-4 — @session inject indicator (.chat-msg--inject) renders from history (572ms)

  24 passed (32.3s)
```

**24/24 passed. PARITY-01 is satisfied by live e2e evidence.**

### Traceability Guard

```
bash tests/check-traceability-paths.sh
OK: all traceability paths exist
Exit: 0
```

No test file added/renamed/removed — TESTING.md counts and traceability rows unchanged.

---

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Text-filter wait fails in full suite due to virtualizer scroll**

- **Found during:** Task 2 (first full-suite run)
- **Issue:** After adding the `.first()` warm-up + extending text-filter timeout to 15 000 ms, the full 24-test suite still failed SC-3 on Firefox and WebKit. The text-filter locator `filter({ hasText: 'Hello from the fixture (RW)' }).toBeVisible()` timed out despite `.first()` passing. Root cause: the broadcast tests (SC-1) that run earlier in the suite send new messages to the fixture session; `ChatPanel` auto-scrolls to the bottom on new messages, pushing the seeded "Hello from the fixture (RW)" row out of the virtualizer's render window. The text-filter locator saw zero visible elements.
- **Fix:** Removed the text-filter wait entirely. `.first()` already proves the WSS handshake and history fetch completed — which is the sole purpose the text-filter readiness check served. The adversarial assertion (`afterCount == beforeCount`) is unchanged.
- **Files modified:** `frontend/e2e/chat-parity.spec.ts`
- **Commit:** 6b3f100a

### No architectural changes, no RO gate changes, no Go files modified.

---

## Security Posture (unchanged from plan)

| Threat | Status |
|--------|--------|
| T-155-06-01 (Elevation of Privilege — SC-3 RO gate) | UNCHANGED: `hub.go:585 ErrChatReadOnly` + `ChatPanel.isReadOnly` not modified. RO cap still cannot post or inject. |
| T-155-06-02 (self-signed cert) | UNCHANGED: `ignoreHTTPSErrors` is fixture-only, no production trust change. |
| T-155-06-SC (supply chain) | UNCHANGED: no new packages. |

---

## Commits

| Hash | Description |
|------|-------------|
| b29c5470 | fix(155-06): add WS warm-up to SC-3 RO-gate test for firefox/webkit |
| 6b3f100a | fix(155-06): drop text-filter wait in SC-3 — use .first() as sole WS gate |

---

## Self-Check

- frontend/e2e/chat-parity.spec.ts modified: confirmed (git log shows 2 commits)
- BLOCKER 2 closed: confirmed (SC-3 passes 3/3 browsers: chromium 975ms, firefox 1.5s, webkit 1.0s)
- Full gate passes: confirmed (24/24 passed, 32.3s)
- RO gate unchanged: confirmed (no Go files modified, ErrChatReadOnly at hub.go:585 untouched)
- TESTING.md unchanged: confirmed (no test file added/renamed/removed; traceability check exits 0)
- No new packages: confirmed (go.mod and package.json unmodified)

## Self-Check: PASSED
