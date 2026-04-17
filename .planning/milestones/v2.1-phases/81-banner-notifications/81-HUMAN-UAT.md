---
status: partial
phase: 81-banner-notifications
source: [81-VERIFICATION.md]
started: 2026-04-16T16:08:00.000Z
updated: 2026-04-16T16:08:00.000Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Visual banner stacking
expected: Both LocalNetworkBanner and UpdateBanner appear stacked vertically inside .banner-stack container when both are active
result: [pending]

### 2. LocalNetworkBanner dismiss animation
expected: Clicking X button on local network banner triggers smooth fade out + collapse animation (150ms opacity, 200ms max-height) before removal
result: [pending]

### 3. UpdateBanner dismiss animation
expected: Clicking Dismiss button on update banner triggers smooth fade out + collapse animation, banner-stack container cleans up when empty
result: [pending]

### 4. D-04 session reset
expected: After dismissing the local network banner, switching web server mode away from local and back to local causes the banner to reappear
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
