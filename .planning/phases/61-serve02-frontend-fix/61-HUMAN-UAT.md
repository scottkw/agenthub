---
status: partial
phase: 61-serve02-frontend-fix
source: [61-VERIFICATION.md]
started: 2026-04-10T07:58:00Z
updated: 2026-04-10T07:58:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. New session with web server running
expected: Launch app, start web server, create a new session — StatusBar for the new session shows "WEB ON" (not "WEB OFF") because createTab seeds webEnabled=true when webServerRunning=true
result: [pending]

### 2. Restored session with web-enabled state
expected: Close and re-open app with a previously web-enabled session — StatusBar shows "WEB ON" for the restored session because init() seeds webEnabled from s.webEnabled on ListSessions result
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
