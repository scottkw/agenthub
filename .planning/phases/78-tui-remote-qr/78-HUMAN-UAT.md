---
status: passed
phase: 78-tui-remote-qr
source: [78-VERIFICATION.md]
started: 2026-04-15T00:00:00Z
updated: 2026-04-15T16:25:00Z
---

## Current Test

[complete]

## Tests

### 1. QR readability with a real phone camera
expected: Scan QR in Terminal.app / iTerm2 / Alacritty / kitty; URL decodes and opens in browser
result: passed
notes: User approved QR scan behavior. During UAT, surfaced a per-session WebEnabled bug where `q` on an unserved session still opened the QR overlay (because `sessionURL()` only checked the global `webStatus.Running` flag, not `session.WebEnabled`). Fixed in commit ef7b51f with regression test TestUpdate_QRUnservedSession, plus tick-refresh guard TestUpdate_QRServeAfterUnserve in 3640869. Re-verified end-to-end under tmux: unserved → toast; external `serve` + 2s tick → QR overlay renders correctly.

### 2. Remote peer parity vs GUI Remote Sessions panel
expected: With a live tailnet, TUI and GUI show the same peers, same groups, same per-session status
result: deferred
notes: No active agenthub peer on the user's tailnet at UAT time (asustor idle, other peers offline). Display logic is covered by automated tests (TestTUIRemoteAndQR_FullFlow, TestView_SessionListWithRemotes). Parity check will surface via /gsd-audit-uat whenever the user has a live peer to compare against.

## Summary

total: 2
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0
deferred: 1

## Gaps

[none — unserve bug was resolved inline during UAT, not a deferred gap]
