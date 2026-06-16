---
status: partial
phase: 132-unified-grid-mini-preview-named-groups
source: [132-VERIFICATION.md]
started: 2026-06-16
updated: 2026-06-16
---

## Current Test

[awaiting human/live testing — 3 runtime items not drivable by source inspection or unit tests]

## Tests

### 1. Mini-preview perf at scale (CARD-07)
expected: Open Hub with 10+ active sessions; previews update smoothly on the shared 3s interval with no jank. DevTools shows ONE batch of GetSessionTailLines per tick (not one-per-session-per-second). Source-verified: exactly 1 setInterval in HubPanel; usePreviewPoller uses stable session-id-join dep.
result: [pending]

### 2. Drag-and-drop card → group sidebar (GROUP-02/04)
expected: Drag a session card onto a named group in the sidebar; card moves to that group; after app restart the membership persists (localStorage). Individual DnD pieces are unit-tested; end-to-end pointer gesture needs a live app.
result: [pending]

### 3. Remote peer card in unified grid (GRID-03/07)
expected: With a live Tailscale peer running AgentHub, the remote session appears alongside local cards with peer hostname as origin and "No output yet" preview (remote sessions intentionally excluded from GetSessionTailLines polling). Needs a live tailnet peer.
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps

- All 3 items are inherently live-runtime (scale, native pointer gesture, live peer). Code-level behavior is verified (7/7 must-haves, all critical review fixes tested). Drivable parts (group sidebar render, create named group, mini-preview empty/loading state, per-card "move to group" menu) can be smoke-tested via `wails dev` + dev-browser; the 3 above need operator confirmation on a built app with real sessions/peers. NOTE: terminal-tail PTY output for a live mini-preview won't render via the external-browser bridge — confirm preview content in the native window.
