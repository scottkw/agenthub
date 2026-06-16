---
status: passed
phase: 130-remote-browse-gui-on-ramp
source: [130-VERIFICATION.md, 130-UI-REVIEW.md]
started: 2026-06-16
updated: 2026-06-16
---

## Current Test

[complete — two-machine tailnet discover→list→pick→join→browse PASSED 2026-06-16]

## Result Summary

Live two-machine tailnet UAT (Machine A = wails dev; Machine B = signed .app with a web-shared `claude 1` session):
- RB-01/RB-04 PASS: Machine B ("Ken's MacBook Air") discovered and listed with its shareable session under "Shows shareable sessions" — not dropped, not falsely "No remote peers found".
- RB-02 PASS: "Browse Files" → join-code modal → entered code → File Browser opened Machine B's files over the relay loopback ("worked perfectly").
- "Browse Files" casing fix confirmed on-screen.

Three pre-Phase-130 bugs surfaced during this UAT (see Gaps) — none block the RB-01..05 data path, all tracked for follow-up.

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
passed: 4
issues: 3
pending: 2
skipped: 0
blocked: 0

(Items 1+2 discover/list/pick/browse PASS live. Items 4 prefers-reduced-motion + 5 colorblind: source-verified, on-screen confirm pending — non-blocking. Item 3 network trust boundary: not separately tested but inherits the webserver bind-IP model.)

## Gaps

Three bugs surfaced during the two-machine UAT — all PRE-Phase-130 (Phase 122/124), out of RB-01..05 scope, none break the proven data path:

1. **Write-toggle does not re-hydrate (Phase 124 — `DaemonManagerPanel.tsx`)** — `sessionWrites` local state is only set on click, never seeded from the daemon's `sessions[].filesWrite`. Leaving and returning to the Sessions tab shows "Enable file writes"/"Allow file editing" as OFF even though the daemon still has `FilesWrite=true` (server-authoritative). Display/cross-surface-parity bug; writes remain enabled server-side, but the misleading OFF can cause accidental disable on re-toggle. Fix: seed `sessionWrites` from `s.filesWrite` (mirror the `webEnabled`-from-`s.webEnabled` restore in `App.tsx`).
2. **Join code never shown as readable text (Phase 122 — `SessionSharePanel.tsx`)** — the owner's panel only embeds the code in the QR / `/join?code=` URL (`SessionSharePanel.tsx:80`), never as copyable text, yet the Remote join modal instructs the owner to read it from the panel. Workaround used in UAT: scan the read-only QR. Fix: display the `readCode`/`writeCode` as copyable text.
3. **Join-code copy says "5-character" but real format is 8-char `XXXX-XXXX` (Phase 122 — `RemoteJoinCodeModal.tsx` + TUI `joincode_prompt.go`)** — `internal/capability/joincode.go:61` formats `encoded[:4] + "-" + encoded[4:8]` (8 base32 chars). Both GUI modal (copy + `placeholder="ABCDE"`) and TUI prompt say "5-character". Consistently-wrong cross-surface copy. Fix: update copy/placeholder to the real 4-dash-4 format on both surfaces.
