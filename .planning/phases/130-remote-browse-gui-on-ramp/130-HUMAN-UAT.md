---
status: partial
phase: 130-remote-browse-gui-on-ramp
source: [130-VERIFICATION.md, 130-UI-REVIEW.md]
started: 2026-06-16
updated: 2026-06-16
---

## Current Test

[awaiting human testing — requires a two-machine tailnet]

## Tests

### 1. Two-machine discover→list (RB-01/RB-04)
expected: On machine A, open the Remote Sessions panel. Machine B (running AgentHub with at least one web-share-enabled session) appears with its shareable sessions listed. A reachable peer with sessions is never shown as "No remote peers found". A reachable peer with zero shareable sessions shows "No shareable sessions". An unreachable peer shows the "Unreachable" text badge.
result: [pending]

### 2. Two-machine pick→browse (RB-02)
expected: From the Remote Sessions panel, click "Browse Files" on a machine-B session → the join-code/cap flow (Phase 122) → the File Browser opens that remote session's files over the relay loopback. Listing succeeds.
result: [pending]

### 3. Network-layer trust boundary (RB-03)
expected: From a host NOT on the tailnet, `GET https://<machine-B-tailscale-ip>:<port>/api/sessions/meta` is not reachable (no route to the bind IP). The metadata endpoint is only reachable by tailnet members. (Also: startup logs a WARN if AgentHub is ever bound to a non-tailnet IP in tailscale mode — WR-03 guard.)
result: [pending]

### 4. prefers-reduced-motion spinner (accessibility)
expected: With macOS "Reduce Motion" enabled, the Remote Sessions loading spinner does not animate (static fallback). Source-verified: `prefers-reduced-motion` CSS rule present; on-screen confirm.
result: [pending]

### 5. Colorblind panel-state legibility
expected: Each per-peer state (reachable-with-sessions / "No shareable sessions" / "Unreachable" / error "Could not load sessions") is distinguishable by TEXT/position, not color. Largely source-verified (UI audit Color 4/4, text-first labels); confirm on-screen / via a colorblind simulation.
result: [pending — source-verified text-first; visual confirm outstanding]

### 6. WR-01 spinner timing + WR-02 stale-peer pick (code-review follow-ups)
expected: The Remote Sessions spinner only shows on genuine first load (not on every 30s poll). If a peer drops between opening the pick modal and submitting, the panel re-polls once and, if still gone, surfaces "Remote session is no longer available — refresh peers and try again." (no silent no-op).
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps
