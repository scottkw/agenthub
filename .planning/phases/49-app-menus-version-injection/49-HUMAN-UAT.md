---
status: partial
phase: 49-app-menus-version-injection
source: [49-VERIFICATION.md]
started: 2026-04-07T08:25:00Z
updated: 2026-04-07T08:25:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. macOS Clipboard Operations in Terminal Tabs
expected: Launch app, create terminal session, Cmd+C/V/X/Z work in xterm.js terminal (previously silently broken)
result: [pending]

### 2. Menu Bar Appearance on macOS
expected: AgentHub, File (New Session Cmd+N, Close Tab Cmd+W), Edit (Cut/Copy/Paste/Undo), Window, Help (AgentHub on GitHub) menus visible in menu bar
result: [pending]

### 3. Help Menu GitHub Link
expected: Help > AgentHub on GitHub opens https://github.com/scottkw/agenthub in browser
result: [pending]

### 4. Version Display on Welcome Screen
expected: Build with `wails build -ldflags "-X main.Version=v1.9.0"`, welcome tab shows "v1.9.0" (not "dev" or "1.0.0")
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
