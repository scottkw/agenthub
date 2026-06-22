---
status: partial
phase: 146-open-session-capability-bug
source: [146-VERIFICATION.md]
started: 2026-06-22T17:20:02Z
updated: 2026-06-22T17:20:02Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. RO join-code open flow (M-13, two-Mac tailnet)
On Mac A, start a session, enable Share, copy the **RO** join code from the Share modal and send it out of band (e.g. paste in chat) to Mac B. On Mac B, click "Open in browser" on Mac A's remote Hub card. Paste the join code into RemoteJoinCodeModal. Confirm.
expected: Browser opens Mac A's live session at `baseURL/sessions/{id}?cap=TOKEN` — no "capability required" page. Session is in RO mode.
why_human: Requires two real Macs on one tailnet. The Wails `BrowserOpenURL` call and the actual HTTP response from the remote peer's `requireCapability` middleware cannot be driven by vitest or go test. The :34115 wails-dev bridge has no real tailnet peer.
result: [pending]

### 2. RW join-code open flow (M-13, two-Mac tailnet)
Repeat test 1 with the **RW** join code from the Share modal.
expected: Browser opens at the RW cap-bearing URL. Session is in RW mode (can send terminal input).
why_human: Same reason — requires two physical Macs and a live tailnet.
result: [pending]

### 3. No-share error UX
Click "Open in browser" on a remote card for a session whose owner has NOT shared it.
expected: A clear error banner appears ("Cannot open session — the remote peer URL is unavailable" or equivalent), not a raw 401 page.
why_human: Can only be fully verified with a live remote card where the owner has not shared. The handler logic is tested by source inspection but the error-banner UX requires a running GUI.
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
