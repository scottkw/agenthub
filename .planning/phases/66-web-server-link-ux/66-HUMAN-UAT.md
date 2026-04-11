---
status: partial
phase: 66-web-server-link-ux
source: [66-VERIFICATION.md]
started: 2026-04-11T23:55:00Z
updated: 2026-04-11T23:55:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Open button opens dashboard in system browser
expected: Clicking Open button calls BrowserOpenURL and opens the dashboard URL in the system's default browser
result: [pending]

### 2. Copy button writes URL to clipboard with Copied! feedback
expected: Clicking Copy calls ClipboardSetText, URL appears in system clipboard, button shows "Copied!" for 1.5s
result: [pending]

### 3. QR code is scannable and resolves to correct URL
expected: The 200x200 QR code image can be scanned by a phone and resolves to the dashboard URL
result: [pending]

### 4. QR cache persists across toggle cycles
expected: Toggling QR off and back on shows the image immediately without re-fetching from the Go backend
result: [pending]

### 5. QR cache clears when server stops
expected: Stopping the web server hides QR image and clears cache; restarting and toggling QR fetches fresh data
result: [pending]

### 6. URL action row works in both Tailscale and local network modes
expected: Open/Copy/QR buttons appear and function correctly in both web server modes
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps
