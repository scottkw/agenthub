---
status: passed
phase: 129-write-concurrency-fix-dns-error-ux
source: [129-VERIFICATION.md]
started: 2026-06-15
updated: 2026-06-15
---

## Current Test

[complete]

## Tests

### 1. DNS-03 proactive RemoteBrowseDNSWarning banner renders on-screen
expected: On a machine whose Tailscale client has `accept-dns=false` (MagicDNS off) while Tailscale is connected, opening the app shows the proactive warning banner with the message "Enable Tailscale DNS (accept-dns) to browse remote sessions" BEFORE any remote browse is attempted. When `accept-dns=true`, the banner does not appear.
result: PASS (2026-06-15, live two-machine tailnet, `accept-dns=false`)
notes: UAT initially showed NO banner despite backend correctly reporting `connected:true, acceptDns:false` (confirmed via DevTools `GetTailscaleStatus()`). Root cause: `RemoteBrowseDNSWarning` was rendered inside `.banner-stack`, but the stack's mount-gating expression in App.tsx did not include the DNS-warning trigger — so when no other banner was active the whole stack was unmounted and the warning never rendered. Fixed by adding `tailscaleHealth?.connected === true && tailscaleHealth?.acceptDns === false` to the gating expression (commit on 2026-06-15) plus an INVARIANT comment guarding the gating site. After the fix the banner renders correctly at `accept-dns=false` (verified on-screen). DNS-03 fully delivered in Phase 129.

## Summary

total: 1
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
