---
status: partial
phase: 60-local-network-fallback
source: [60-VERIFICATION.md]
started: 2026-04-09T22:00:00Z
updated: 2026-04-09T22:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Banner visual layout
expected: Amber-bordered banner appears above sidebar+content row without shrinking terminal area
result: [pending]

### 2. Settings password display and copy
expected: "LAN Access Password" field shows ~22-char password in Settings > Web Server, clicking copies to clipboard with "Copied!" feedback
result: [pending]

### 3. Browser HTTP Basic Auth prompt
expected: Navigating from LAN device to https://<LAN-IP>:7443 shows credential dialog, generated password grants access, wrong passwords return 401
result: [pending]

### 4. No banner in Tailscale mode
expected: With Tailscale connected and certs enabled, LocalNetworkBanner is absent and HealthModal behaves normally
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
