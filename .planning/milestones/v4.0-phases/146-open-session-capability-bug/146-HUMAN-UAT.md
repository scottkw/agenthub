---
status: passed
phase: 146-open-session-capability-bug
source: [146-VERIFICATION.md]
started: 2026-06-22T17:20:02Z
updated: 2026-06-22
---

## Current Test

[complete — all human UAT items approved 2026-06-22 on live two-Mac tailnet build (Developer ID signed, universal)]

## Tests

### 1. End-to-End First Open (RO code) — M-13 Sub-scenario A
On Mac A, start a session, enable Share, copy the **RO** join code from the Share modal and send it out of band to Mac B. On Mac B, click "Open in browser" on Mac A's remote Hub card. Paste the code into RemoteJoinCodeModal. Confirm.
expected: Browser opens at `baseURL/sessions/{id}?cap=TOKEN` in RO mode. No "capability required" page. Modal closes after successful exchange.
why_human: Requires two real Macs on a live tailnet. `BrowserOpenURL` and the remote peer's `requireCapability` HTTP response cannot be exercised by vitest or `go test`. The `:34115` wails-dev bridge has no real tailnet peer.
result: PASS (2026-06-22, user-approved on live two-Mac tailnet)

### 2. End-to-End Second Open (Held-Cap Reuse) — M-13 Sub-scenario B
After completing Test 1 (cap deposited in-app), click "Open in browser" on the SAME remote card WITHOUT obtaining a fresh join code. Repeat with RW.
expected: Browser opens directly (no join-code modal), reusing the held cap. The single-use code is already consumed (D-11) — second open must work without prompting. RW repeat: same held-cap reuse behavior with RW permissions.
why_human: The held-cap reuse path is code-verified and test-locked (Plan 05). The live behavior on a real two-Mac tailnet after in-app connect must be confirmed. This is the literal user-reported failure that GAP-146-A was filed to fix.
result: PASS (2026-06-22, user-approved on live two-Mac tailnet)

### 3. No-Share Error UX
Click "Open in browser" on a remote card where the owner has NOT shared (Share toggle off — no valid code exists).
expected: Error banner appears ("Cannot open session — the remote peer URL is unavailable" or equivalent). No raw 401 page.
why_human: Requires a live remote peer with a non-shared session. The banner-vs-401 outcome is a UX observation that requires a running GUI.
result: PASS (2026-06-22, user-approved on live two-Mac tailnet)

## Summary

total: 3
passed: 3
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

### GAP-146-A — "Open in browser" forced a second single-use code instead of reusing a held cap
status: resolved
discovered: 2026-06-22 (live two-Mac tailnet UAT, test 1)
resolved: 2026-06-22 (Plan 146-05)
detail: Join codes are single-use (D-11). The in-app connect consumed the code; "Open in browser" unconditionally re-prompted and re-exchanged rather than reusing the held cap. The consumed code's re-exchange returned `ErrCodeNotFound`, mislabeled "Code invalid" (WR-03).
resolution: Plan 146-05 — `handleOpenRemoteSession` now checks `remoteCapsCached.has(session.id)` and opens the cap-bearing URL directly via new `App.OpenRemoteSessionURL` binding + daemon endpoint `GET /api/remote-files/caps/{sessionID}/open-url` when a cap is held; modal only when none held. WR-01 (SID-correct daemon-composed URL) and WR-03 (correct "Code already used or expired" copy) also fixed; WR-02 behavior tests added. Live confirmation is Test 2 above.
