---
status: complete
phase: 21-cli-session-web-commands
source: 21-01-SUMMARY.md, 21-02-SUMMARY.md
started: 2026-03-24T14:00:00Z
updated: 2026-03-24T14:10:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Create New Session
expected: Run `go run ./cmd/agenthub-cli/ new cat /tmp` — prints a 32-character hex session ID to stdout (no dashes, lowercase).
result: pass

### 2. List Sessions
expected: Run `go run ./cmd/agenthub-cli/ list` — prints a table with columns ID, NAME, AGENT, STATUS. The session created in test 1 should appear.
result: pass

### 3. Rename Session
expected: Run `go run ./cmd/agenthub-cli/ rename <id-from-test-1> mytest` — exits silently with code 0. Running `list` again shows the session with NAME "mytest".
result: pass

### 4. Kill Session
expected: Run `go run ./cmd/agenthub-cli/ kill <id-from-test-1>` — exits silently with code 0. Running `list` again no longer shows that session.
result: pass

### 5. Web Start (Tailscale Health Gate)
expected: Run `go run ./cmd/agenthub-cli/ web start` — if Tailscale is not connected/configured, prints an error about Tailscale health (connected, IP, or certs). Does NOT attempt to start the web server on bad Tailscale state.
result: pass

### 6. Web Stop
expected: Run `go run ./cmd/agenthub-cli/ web stop` — exits silently with code 0 even if no web server is running (no error output).
result: pass

### 7. Web Status
expected: Run `go run ./cmd/agenthub-cli/ web status` — prints key-value output showing web server state (running/stopped, address if running).
result: pass

### 8. Health Check
expected: Run `go run ./cmd/agenthub-cli/ health` — prints Tailscale health info with aligned key-value labels (Connected, IP, HasCerts fields).
result: pass

### 9. QR Code Rendering
expected: Run `go run ./cmd/agenthub-cli/ qr` — renders a Unicode half-block QR code in the terminal followed by the URL below it.
result: pass

## Summary

total: 9
passed: 9
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
