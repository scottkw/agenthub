---
created: 2026-06-30T18:28:29.721Z
title: Help Guide — document Tailscale Funnel admin prerequisites
area: docs
phase: 166
files:
  - frontend/src (Phase 166 Sharing Guide Help article — not yet created)
---

## Problem

The in-app Funnel toggle (Phase 166) cannot, by itself, make Funnel work. Before
`EnableFunnel` succeeds, the operator must complete one-time Tailscale **admin
console** setup that the app has no control over. Without it, `EnableFunnel`
returns `Funnel not available; "funnel" node attribute not set` (FNL-06) and the
toggle appears broken to the user.

This is the exact gap found during **Phase 165 UAT**: human-verification items
M-34 (external no-port Funnel URL → 200) and M-35 (`tailscale serve status` empty
after each teardown) were **blocked** because this host's tailnet had the `funnel`
node attribute unset and was in BackendState=Stopped. The backend behaves
correctly (clean error, funnelActive stays false, no SetServeConfig), but a
non-expert user has no way to know what to fix.

## Solution

The Phase 166 in-app **Sharing Guide Help article** must include a clear,
screenshot-driven walkthrough of the Funnel operator prerequisites:

1. **Grant the `funnel` node attribute** — Tailscale admin console →
   **Access Controls** → **JSON editor** → click **"Add Funnel to policy"**
   (the Funnel (Beta) helper in the right panel auto-inserts the `nodeAttrs`
   funnel block) → **Save**. (Verified working flow during Phase 165 UAT.)
2. **Enable HTTPS certificates** for the tailnet if not already on (Funnel's other
   prerequisite — DNS / Feature settings).
3. **`tailscale up`** so the node is connected (BackendState=Running), not Stopped.

Acceptance: a user who has never used Tailscale Funnel can follow the Help article
end-to-end and reach a state where the in-app toggle enables Funnel without the
"funnel node attribute not set" error. Surface the FNL-06 error text in the guide
as a "if you see this, do step 1" troubleshooting hook.

Note: this is a docs/Help requirement layered onto Phase 166's existing
"in-app Sharing Guide Help article" deliverable — fold it into that phase's scope
rather than treating it as a standalone phase.
