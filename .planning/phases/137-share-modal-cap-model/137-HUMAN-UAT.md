---
status: partial
phase: 137-share-modal-cap-model
source: [137-VERIFICATION.md]
started: 2026-06-20T00:00:00Z
updated: 2026-06-20T00:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Share modal opens with RO + RW link rows
expected: With a running local session, clicking Share and toggling "Share the session" ON renders both link rows (read-only and full-access) with copyable URLs and QR codes; URLs are non-empty and distinct.
result: [pending]

### 2. Read-only token grants read-only file browse (SHARE-03 inheritance)
expected: With sharing ON and "Enable remote file browsing" ON, opening the read-only share URL in a second browser loads the file browser in read mode; write operations return 403.
result: [pending]

### 3. LAN password shown in local-network mode (SHARE-04)
expected: In local/LAN web-server mode, the Share modal displays the Basic Auth password under the share links.
result: [pending]

### 4. Home-dir write warning renders (D-09)
expected: Opening the Share modal on a session whose working directory is the user's home directory shows the HomeDirWriteWarning banner before the browse toggle.
result: [pending]

### 5. Remote peer card Share button disabled, colorblind-safe (D-13)
expected: On a remote peer card the Share button is disabled, shows a lock icon, and the tooltip/aria-label reads "Only the session owner can share" — shape + text convey the state, not color alone.
result: [pending]

### 6. Web-server restart clears cached share URLs (SHARE-05)
expected: Toggling the web server off then on while the Share modal is open clears the cached URLs and auto-issues new ones; old links become invalid.
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps
