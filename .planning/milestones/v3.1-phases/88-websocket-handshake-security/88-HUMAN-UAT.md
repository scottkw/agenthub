---
status: partial
phase: 88-websocket-handshake-security
source: [88-VERIFICATION.md]
started: 2026-04-21T00:00:00Z
updated: 2026-04-21T00:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. SC-2 local-HTTPS-fallback: open share link in browser on same LAN with self-signed cert, disable tailnet first
expected: Terminal page loads and WebSocket upgrade completes (101); devtools shows `Origin: https://<host-ip>:<port>` accepted with no user-visible error
result: [pending]

### 2. SC-2 tailscale-mode UAT: open share link from another tailnet node browser
expected: Terminal page attaches (WS 101); devtools confirms `Origin: https://<host>.<tailnet>.ts.net:<port>` accepted
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
