---
status: complete
phase: 05-qr-codes-status-indicators
source: [05-01-SUMMARY.md, 05-02-SUMMARY.md, 05-03-SUMMARY.md]
started: 2026-03-18T20:30:00Z
updated: 2026-03-18T21:15:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Status dots visible on tabs
expected: Each tab shows a small colored dot before the tab name. New sessions show a blue dot (running).
result: pass

### 2. Status dot transitions
expected: When a CLI session returns to its prompt (e.g., Claude Code shows ❯), the dot changes from blue (running) to green (idle). When you type or a command runs, it changes back to blue.
result: issue
reported: "the dot stays blue the whole time"
severity: major

### 3. Terminal fills window
expected: The active terminal fills the full available height below the tab bar. No wasted whitespace at the bottom.
result: issue
reported: "No the terminal output is not filling the window. Large blank area below terminal content."
severity: major

### 4. No web-serving bar when server is off
expected: When the web server is NOT started (default), there are no "Web Off" buttons or web-serving controls visible anywhere. The terminal area is clean.
result: pass

### 5. Web server start and web serving toggle
expected: Open Settings (gear icon). Set a password. Start Web Server. Close settings. A "Web Off" button now appears above the active terminal. Click it to enable web serving — it changes to "Web On" and shows the session URL.
result: pass

### 6. QR button and modal
expected: With web serving enabled on a session, a "QR" button appears next to "Copy Token Link". Clicking it opens a modal overlay with a 256x256 QR code image and the session URL below it. Modal closes on clicking outside or pressing Escape.
result: pass

### 7. Web dashboard accessible
expected: With the web server running, the server URL (shown in Settings) opens a dashboard in a browser. The dashboard shows a list of web-enabled sessions.
result: issue
reported: "When I open the link, all I get is Unauthorized."
severity: major

### 8. Dashboard QR thumbnails
expected: On the web dashboard, each session row shows a small (64x64) QR code thumbnail. Clicking it opens an enlarged 256x256 QR overlay with the session URL.
result: skipped
reason: Blocked by test 7 — dashboard returns Unauthorized

## Summary

total: 8
passed: 4
issues: 3
pending: 0
skipped: 1
skipped: 0

## Gaps

- truth: "Status dot changes from blue (running) to green (idle) when CLI returns to prompt"
  status: failed
  reason: "User reported: the dot stays blue the whole time"
  severity: major
  test: 2
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""
- truth: "Terminal fills full available height below tab bar with no wasted whitespace"
  status: failed
  reason: "User reported: terminal output is not filling the window, large blank area below terminal content"
  severity: major
  test: 3
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""
- truth: "Web dashboard loads in browser when navigating to server URL"
  status: failed
  reason: "User reported: When I open the link, all I get is Unauthorized."
  severity: major
  test: 7
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""
