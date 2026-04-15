---
status: partial
phase: 77-tui-session-operations
source: [77-VERIFICATION.md]
started: 2026-04-15T16:15:00Z
updated: 2026-04-15T16:15:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Attach to a running session from TUI and detach with Ctrl-\
expected: TUI suspends, raw PTY attach runs with status bar, Ctrl-\ returns to TUI with session list refreshed
result: [pending]

### 2. Create a new session via n key modal
expected: Modal opens with agent picker, directory pre-filled, Tab cycles focus, Enter creates session, list refreshes with new entry
result: [pending]

### 3. Kill a session via d key confirmation dialog
expected: Confirmation dialog appears with session name, default focus on No, y confirms kill, session disappears from list
result: [pending]

### 4. Rename a session via r key inline edit
expected: Name column replaced with textinput pre-filled with current name, Enter submits, new name reflected immediately
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
