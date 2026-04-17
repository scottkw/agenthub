---
status: partial
phase: 80-tailscale-detection
source: [80-VERIFICATION.md]
started: 2026-04-16T14:30:00Z
updated: 2026-04-16T14:30:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Daemon-Stopped State Display
Launch app with Tailscale installed but daemon stopped; verify "Daemon Stopped" label with orange dot and platform instruction
expected: Orange dot, "Daemon Stopped", macOS-specific text ("open Tailscale from Applications or the menu bar")
result: [pending]

### 2. Diagnostics Checklist Visual Rendering
Click "Show diagnostics" and verify stepped pass/fail/gray indicators
expected: Green check on "Binary detected", red cross on "Daemon running", gray dashes on remaining steps
result: [pending]

### 3. Banner Daemon-Stopped State (D-06)
Start web server in local mode with daemon stopped; verify text-only banner with NO buttons
expected: "Tailscale daemon not running" text, no action buttons
result: [pending]

### 4. Not-Installed State Display
On a machine without Tailscale, verify "Not Installed" red dot state
expected: Red dot, "Not Installed", install CTA text
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
