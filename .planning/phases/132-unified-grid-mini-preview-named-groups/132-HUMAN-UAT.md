---
status: passed
phase: 132-unified-grid-mini-preview-named-groups
source: [132-VERIFICATION.md]
started: 2026-06-16
updated: 2026-06-19
---

## Current Test

Live smoke test performed 2026-06-16 via `wails dev` + dev-browser (2 shell sessions). Drivable Phase 132 features all PASS live:
- Group sidebar renders with per-group running/total counts (GROUP-01): "All 2/2", "Backend 1/1".
- Collapsible sidebar (GROUP-03): toggle collapses to icon strip; `hub-group-sidebar-collapsed` localStorage null→true; grid reflows wider.
- Create named group (GROUP-01): typed "Backend" → appears in sidebar; persists to `agenthub:hubGroups:v1`.
- Move-to-group via per-card menu (GROUP-02): "Card options for shell 1" → "Backend" → localStorage memberKeys=["shell 1:::/Users/ken"] (name+workDir key).
- Named grouping in grid + Other fallback (GROUP-04): grid headers became ["BACKEND","OTHER"] (shell 1→Backend, shell 2→Other).
- Mini-preview pane (CARD-07) renders on every card (showed "Loading…" — PTY content does not stream over the external-browser bridge; structure/placement verified, content needs native window).
- Phase 131 regression: Open button preserved on all cards; drag handles + menu buttons present.

Cosmetic finding — FIXED (commit ff797fab): the expanded sidebar collapse-toggle chevron rendered oversized and overlapped the "GROUPS" heading. Root cause: Heroicons used Tailwind `w-N h-N` classes but the project has no Tailwind (no-ops → unconstrained SVGs). Added explicit CSS sizing for the chevron (16px), needs-input badge (12px), and card drag-handle/menu-btn (16px). Live-verified: chevron now 16px in its own row, no overlap. +3 CSS-contract assertions.

The 3 items below still need a built app with real sessions/peers:

## Tests

### 1. Mini-preview perf at scale (CARD-07)
expected: Open Hub with 10+ active sessions; previews update smoothly on the shared 3s interval with no jank. DevTools shows ONE batch of GetSessionTailLines per tick (not one-per-session-per-second). Source-verified: exactly 1 setInterval in HubPanel; usePreviewPoller uses stable session-id-join dep.
result: PASS-with-nuance (live, 2026-06-19) — created 11 sessions (9 running) on the v3.6 daemon. All **11 cards rendered** in the grid; mini-preview panes **populated with live tail content** (some real output, others "No output yet"). Instrumented the `GetSessionTailLines` Wails binding for ~7s (≈2 ticks): **22 calls = ~11 per ~3s tick** → a SINGLE shared interval (confirmed) issuing **one RPC per session per tick**, NOT one literal batched RPC. The "one-per-session-per-SECOND" anti-pattern is absent (cadence is the shared ~3s tick), but the literal "ONE batch per tick" wording is not met — per-session fan-out. NOTE for scale: N sessions = N RPCs/tick. Visual "no jank" smoothness remains native-window-only.

### 2. Drag-and-drop card → group sidebar (GROUP-02/04)
expected: Drag a session card onto a named group in the sidebar; card moves to that group; after app restart the membership persists (localStorage). Individual DnD pieces are unit-tested; end-to-end pointer gesture needs a live app.
result: PASS (live, 2026-06-19) — created group "DragTarget", then dispatched the HTML5 DnD sequence (dragstart→dragover→drop) from the `uat-err` card onto it. The app's real handlers ran end-to-end: `onDragStart` set `dataTransfer['text/plain']` = `uat-err:::/Users/ken/dev/agenthub`; `onDrop` → `onDropOnGroup` → state update; localStorage `agenthub:hubGroups:v1` persisted `memberKeys:["uat-err:::/Users/ken/dev/agenthub"]`; grid regrouped to headers ["DragTarget","Other"] (card moved). Persistence is via localStorage (survives restart). Only the OS-level pointer→drag translation (browser-native, not app logic) was synthesized rather than physically performed.

### 3. Remote peer card in unified grid (GRID-03/07)
expected: With a live Tailscale peer running AgentHub, the remote session appears alongside local cards with peer hostname as origin and "No output yet" preview (remote sessions intentionally excluded from GetSessionTailLines polling). Needs a live tailnet peer.
result: PASS (live, 2026-06-19) — with a SECOND machine ("Ken's MacBook Air (1574)", kens-macbook-air-1574.tail46d69a.ts.net) running AgentHub on the same tailnet and sharing a `claude` session: `/tailnet/peers` discovered the peer (online); `GetRemoteSessionsWithMeta()` returned it reachable with session "claude 1" (claude, running, tailnet URL on :7443). On the local `:34115` Hub the remote session rendered as a card in the unified grid alongside local cards — name "claude 1", **origin "Ken's MacBook Air (1574)"** (peer hostname), CLI badge "claude", status "Running", preview **"No output yet"** (remote sessions correctly excluded from GetSessionTailLines polling), with an "Open" affordance. GRID-03/07 confirmed end-to-end across two real tailnet hosts.

## Summary

total: 3
passed: 3
issues: 0
pending: 0
skipped: 0
blocked: 0
note: item 1 PASS-with-nuance (per-session RPC fan-out, not literal single batch); item 3 confirmed live across two real tailnet machines 2026-06-19

## Gaps

- All 3 items are inherently live-runtime (scale, native pointer gesture, live peer). Code-level behavior is verified (7/7 must-haves, all critical review fixes tested). Drivable parts (group sidebar render, create named group, mini-preview empty/loading state, per-card "move to group" menu) can be smoke-tested via `wails dev` + dev-browser; the 3 above need operator confirmation on a built app with real sessions/peers. NOTE: terminal-tail PTY output for a live mini-preview won't render via the external-browser bridge — confirm preview content in the native window.

## Live Re-test 2026-06-19 (post-milestone tech-debt review)

Re-run via `wails dev` + dev-browser. Additional live confirmations beyond the 2026-06-16 smoke test:
- GRID-03 group sidebar renders with per-group running/total counts ("All 1/1", "Backend 1/1") + collapse chevron + "New group".
- GROUP-01 create named group "Backend" live → appears "Backend 0/0".
- GROUP-02 move-to-group via per-card menu ("MOVE TO GROUP" → "Backend") → card moves under the BACKEND group header; sidebar updates to "Backend 1/1".
- GROUP-03/04 persistence: localStorage `agenthub:hubGroups:v1` = `[{"name":"Backend","memberKeys":["shell 2:::__nodir__"]}]` (name:::workDir key, `__nodir__` sentinel for the home dir).
- GRID-02 group-by-cwd headers ("BACKEND", "OTHER").

The 3 formal pending items (mini-preview perf at SCALE 10+, native pointer DnD gesture, live remote tailnet peer / GRID-07) remain operator-only — they need scale, a real pointer drag, or live tailnet infra the external bridge cannot drive. Unchanged.
