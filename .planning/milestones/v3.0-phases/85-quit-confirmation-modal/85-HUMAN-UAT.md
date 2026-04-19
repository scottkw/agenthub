---
status: partial
phase: 85-quit-confirmation-modal
source: [85-VERIFICATION.md]
started: 2026-04-19T13:15:00Z
updated: 2026-04-19T13:15:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Full modal appearance on window close
expected: Closing the GUI window shows the quit confirmation modal with title "Quit AgentHub?", session list, and three buttons
result: [pending]

### 2. Quit GUI Only path
expected: Clicking "Quit GUI Only" hides the window to tray, hides dock icon, and shows a macOS notification with session count
result: [pending]

### 3. Quit Everything path
expected: Clicking "Quit Everything" shuts down the daemon, terminates all sessions, and fully exits the application
result: [pending]

### 4. Tray Quit menu triggers modal
expected: Selecting "Quit" from the system tray menu auto-shows the window and displays the quit confirmation modal
result: [pending]

### 5. Cancel/Escape/overlay dismiss
expected: Pressing Escape, clicking the overlay, or clicking "Keep Running" dismisses the modal with no state change
result: [pending]

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps
