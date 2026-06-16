---
status: partial
phase: 129-write-concurrency-fix-dns-error-ux
source: [129-VERIFICATION.md]
started: 2026-06-15
updated: 2026-06-15
---

## Current Test

[awaiting human testing]

## Tests

### 1. DNS-03 proactive RemoteBrowseDNSWarning banner renders on-screen
expected: On a machine whose Tailscale client has `accept-dns=false` (MagicDNS off) while Tailscale is connected, opening the app shows the proactive warning banner with the message "Enable Tailscale DNS (accept-dns) to browse remote sessions" BEFORE any remote browse is attempted. When `accept-dns=true`, the banner does not appear.
result: [pending]
notes: Component logic, conditional guard (`connected===true && acceptDns===false`), exact message wording, and `role="status"` are source-verified; `tsc --noEmit` clean. Only the live on-screen render with a real `accept-dns=false` peer is unverifiable without a two-machine tailnet. Naturally co-verifiable with Phase 130 (Remote Browse GUI On-Ramp) on the same two-machine setup.

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
