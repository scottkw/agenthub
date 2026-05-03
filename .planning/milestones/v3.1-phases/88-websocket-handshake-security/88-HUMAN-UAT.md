---
status: passed
phase: 88-websocket-handshake-security
source: [88-VERIFICATION.md]
started: 2026-04-21T00:00:00Z
updated: 2026-05-02T23:55:00Z
uat_completed: 2026-05-02T23:55:00Z
---

## Current Test

[completed]

## Tests

### 1. SC-2 local-HTTPS-fallback: open share link in browser on same LAN with self-signed cert, disable tailnet first
expected: Terminal page loads and WebSocket upgrade completes (101); devtools shows `Origin: https://<host-ip>:<port>` accepted with no user-visible error
result: PASS (2026-05-02 against v3.1.0-rc3 dmg). Tailscale daemon stopped; AgentHub correctly fell to local-network mode (self-signed TLS, LAN IP 192.168.1.186, Basic Auth password). Safari opened the Full Access Link `https://192.168.1.186:7443/sessions/<id>?cap=...`, accepted self-signed cert, terminal page rendered with live PTY echo. WebSocket upgrade verified in DevTools Network tab: Request `Origin: https://192.168.1.186:7443`, Response `Status: 101 Switching Protocols`, URL `wss://192.168.1.186:7443/sessions/<id>/ws?cap=...`. No user-visible error.

### 2. SC-2 tailscale-mode UAT: open share link from another tailnet node browser
expected: Terminal page attaches (WS 101); devtools confirms `Origin: https://<host>.<tailnet>.ts.net:<port>` accepted
result: PASS (2026-05-02 against v3.1.0-rc3 dmg). After Tailscale daemon restored + AgentHub web server stop/start, share URL switched to `https://kens-personal-macbook-air.tail46d69a.ts.net:7443/sessions/<id>?cap=...`. QR scanned from iPhone (tailnet-authenticated), terminal page rendered with no cert warning (Let's Encrypt via Tailscale), bidirectional input verified. WS upgrade Origin header confirmed via Web Inspector: `Origin: https://kens-personal-macbook-air.tail46d69a.ts.net:7443`, response 101 Switching Protocols.

## Summary

total: 2
passed: 2
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

None — both SC-2 manual-only items verified end-to-end against the v3.1.0-rc3 signed binary.

## Phase 87 follow-up findings surfaced during this UAT (out of Phase 88 scope)

1. **AgentHub doesn't auto-detect Tailscale daemon coming back online.** After `sudo launchctl load com.tailscale.tailscaled.plist` + `sudo tailscale up`, Settings still showed Tailscale as Connected but the web server URL stayed bound to the LAN IP. Manual Stop/Start Web Server was required to force re-binding to the Tailscale interface. Worth a state-change watcher in the daemon's Tailscale-mode detector.

2. **QR code in Sessions tab doesn't refresh after JoinCodeManager state is wiped.** When the web server restarts (e.g., during a manual mode switch), the in-memory join codes are invalidated, but the displayed QR continues to encode the old, now-invalid code. First scan returned "expired"; "Stop sharing / reshare" forced a fresh code and worked. Should refresh QR (or invalidate the displayed share UI) on web-server-restart events.
