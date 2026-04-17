---
status: partial
phase: 82-minimize-to-tray
source: [82-VERIFICATION.md]
started: 2026-04-17T00:00:00Z
updated: 2026-04-17T00:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Window hidden on launch
expected: Enable toggle in Settings > Behavior, quit app, relaunch — main window does not appear, only tray icon is visible
result: [pending]

### 2. Window appears normally when disabled
expected: Disable toggle in Settings > Behavior, quit app, relaunch — window opens normally on launch
result: [pending]

### 3. Persistence across restarts
expected: Full toggle-on/relaunch/toggle-off/relaunch cycle — preference survives via settings.json round-trip with daemon startup
result: [pending]

### 4. Loading and error UI states
expected: Observe opacity-0.6 loading state during save; trigger daemon error to confirm state revert and error message display
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
