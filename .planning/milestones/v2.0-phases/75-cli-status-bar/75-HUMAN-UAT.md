---
status: partial
phase: 75-cli-status-bar
source: [75-VERIFICATION.md]
started: 2026-04-15T03:45:00Z
updated: 2026-04-15T03:45:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Status bar rendering and scroll region correctness
expected: Attach to a live session; verify the reverse-video bottom bar appears with session name, agent type, hostname, detach hint, and elapsed time. Terminal output scrolls cleanly above it without garbled lines or bar corruption.
result: [pending]

### 2. Live viewer count update
expected: Open two attach clients to the same session; verify the first client's bar updates to "2 viewers" within 1 second, then drops the field when the second client leaves.
result: [pending]

### 3. Terminal cleanup on exit
expected: Press Ctrl-\ to detach; verify the bar row is cleared, the scroll region is restored to full-terminal, and no leftover ANSI artifacts remain.
result: [pending]

### 4. Non-TTY suppression
expected: Run `agenthub attach <session-id> | cat` and confirm no bar escape sequences appear in the piped output.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
