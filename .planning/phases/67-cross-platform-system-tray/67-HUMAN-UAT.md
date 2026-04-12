---
status: partial
phase: 67-cross-platform-system-tray
source: [67-VERIFICATION.md]
started: 2026-04-12T00:00:00Z
updated: 2026-04-12T00:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Linux tray icon visible
expected: AgentHub icon visible in system tray after launch on GNOME/KDE/XFCE
result: [pending]

### 2. Linux session menu
expected: Right-click shows Open AgentHub, separator, dynamic sessions, separator, Quit
result: [pending]

### 3. Linux hide-on-close and Quit
expected: Closing window hides it (tray persists, daemon alive); Quit fully exits
result: [pending]

### 4. Windows tray icon visible
expected: AgentHub icon appears in Windows notification area after launch
result: [pending]

### 5. Windows menu, hide-on-close, Quit
expected: Right-click popup menu with sessions; close hides window; Quit fully exits
result: [pending]

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps
