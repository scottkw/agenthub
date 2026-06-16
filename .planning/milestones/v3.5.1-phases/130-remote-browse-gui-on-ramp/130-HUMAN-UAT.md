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
result: PASS (live 2026-06-16) — Machine B ("Ken's MacBook Air") discovered and listed its shareable "claude 1" session under "Shows shareable sessions"; not dropped, not falsely "No remote peers found".

### 2. Two-machine pick→browse (RB-02)
expected: From the Remote Sessions panel, click "Browse Files" on a machine-B session → the join-code/cap flow (Phase 122) → the File Browser opens that remote session's files over the relay loopback. Listing succeeds.
result: PASS (live 2026-06-16) — "worked perfectly"; File Browser opened Machine B's files over the relay loopback.

### 3. Network-layer trust boundary (RB-03)
expected: From a host NOT on the tailnet, `GET https://<machine-B-tailscale-ip>:<port>/api/sessions/meta` is not reachable (no route to the bind IP). The metadata endpoint is only reachable by tailnet members.
result: VERIFIED (architecture + source) — integration check confirmed `/api/sessions/meta` is mounted ONLY on the Tailscale-bound webserver mux (not the daemon socket or relay loopback), so it is unreachable off-tailnet by construction; the WR-03 startup guard warns if ever bound to a non-tailnet IP in tailscale mode. A live non-tailnet probe was not run (no off-tailnet host in the UAT); accepted as architecturally guaranteed.

### 4. prefers-reduced-motion spinner (accessibility)
expected: With macOS "Reduce Motion" enabled, the Remote Sessions loading spinner does not animate (static fallback).
result: SOURCE-VERIFIED — `prefers-reduced-motion` CSS fallback rule present (added per UI-SPEC accessibility contract). On-screen toggle confirm optional; non-blocking.

### 5. Colorblind panel-state legibility
expected: Each per-peer state (reachable-with-sessions / "No shareable sessions" / "Unreachable" / error "Could not load sessions") is distinguishable by TEXT/position, not color.
result: PASS — text-first labels confirmed (UI audit Color 4/4); the colorblind operator used the panel live and read the states by text ("Shows shareable sessions", session rows, "Browse Files") during the two-machine UAT. Distinguishable without color.

### 6. WR-01 spinner timing + WR-02 stale-peer pick (code-review follow-ups)
expected: The Remote Sessions spinner only shows on genuine first load (not on every 30s poll). If a peer drops between opening the pick modal and submitting, the panel re-polls once and, if still gone, surfaces "Remote session is no longer available — refresh peers and try again." (no silent no-op).
result: VERIFIED-BY-FIX — WR-01 (ref-gated spinner, `de9728f`) + WR-02 (re-poll + error banner, `b3907f6`) committed and covered by the green frontend suite. On-screen edge-timing confirm optional; non-blocking.

## Summary

total: 6
passed: 6
issues: 0
pending: 0
skipped: 0
blocked: 0

(Items 1+2 PASS live two-machine. Item 5 confirmed by colorblind operator during live use. Item 3 verified architecturally (tailnet-bound mux + WR-03 guard). Items 4+6 source/test-verified; optional on-screen edge confirms are non-blocking.)

## Gaps

Three bugs surfaced during the two-machine UAT — all PRE-Phase-130 (Phase 122/124) — were FIXED in-milestone per user decision (folded into v3.5.1):

1. **Write-toggle did not re-hydrate (Phase 124 — `DaemonManagerPanel.tsx`)** — FIXED `667807b`: added a `useEffect` seeding `sessionWrites` from server-authoritative `s.filesWrite` on session arrival (mirrors the `webEnabled` restore), with a re-hydration unit test.
2. **Join code not shown as readable text (Phase 122 — `SessionSharePanel.tsx`)** — FIXED `9fb33b0`: added a `CodeDisplay` (monospace text + Copy) for `readCode`/`writeCode`.
3. **Join-code copy said "5-character" (Phase 122 — `RemoteJoinCodeModal.tsx` + TUI `joincode_prompt.go`)** — FIXED `7b85c16`: corrected to 8-char `XXXX-XXXX` on both GUI and TUI surfaces.

No open gaps.
