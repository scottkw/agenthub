# v3.6 Hub — Resume Handoff (paused 2026-06-17, Phase 134 live-UAT gate)

Autonomous run (`/gsd-autonomous`) **paused at the Phase-134 human-UAT gate** by user choice ("Pause for live UAT now"). Phase 134 code + automated verification are COMPLETE; phase is NOT yet marked complete in ROADMAP — it stays `human_needed` until the 6 live-UAT items pass (see `134-HUMAN-UAT.md`).

## Resume after UAT
Once the live UAT passes: rerun `/gsd-autonomous`. It will mark 134 complete and proceed to **Phase 135 (Accessibility Hardening)** → milestone audit/complete/cleanup. If UAT finds issues, run `/gsd:plan-phase 134 --gaps` for closure first.

## Phase 134 — what landed this session (2026-06-17)
- **134-07 DONE** (commits 9ba5ee7f→baeecbcb): RelayClient `{remote}` seam (daemon-proxy URL, cap stays server-side — CR-01); TerminalPanel/HubInteractiveModal thread `remote`; HubBriefingModal remote tail via WS snapshot (CR-02) + CR-03 lifecycle fix (close-on-timeout, `settled` guard, unmount cleanup).
- **134-08 DONE** (commits 135175ea→ade06c32): `isRemote` threaded HubPanel→HubModal→both leaves; WR-01..04 fixes; WR-07 behavioral tests; WR-05/06 in-code deferred to 135.
- **Re-review (134-REVIEW.md)**: 0 blockers (CR-01/02/03 confirmed fixed; cap-gating verified secure), 4 warn, 3 info.
- **Fixes applied** (commits 8f3e16d2→93b8c208): WR-01 (tail RelayClient unmount leak — the CR-03 leak class reintroduced on the tail path), WR-02 (stranded pendingModalSessionId), WR-04 (fixed-500ms tail window → frame-quiescence idle timer), IN-01 (uniform timeout clear).
- **Verification (134-VERIFICATION.md)**: status `human_needed`, 18/18 automated must-haves verified. 1699/1699 frontend tests, Go daemon+relay `-race` pass, tsc clean.
- **Still DEFERRED to Phase 135** (signed off, NOT gaps): WR-03 read-only-cap silent-drop indicator (needs colorblind-safe non-color cue — release-blocking), WR-05/WR-06, IN-03 aria origin, focus-trap/A11Y-04. IN-02 (memory bound) optional.

## Status (historical)
- **Phases 131, 132, 133** — ✅ complete & shipped (see git history). Attention live-UAT for 133 still pending (HUMAN-UAT).
- **Phase 134 (Modal Interaction)** — ⏳ code done, awaiting live UAT (see above).
  - Plans **134-01..05 DONE** (modal works for LOCAL sessions): card-click→modal, grow/shrink animation, Escape/focus-return, interactive (TerminalPanel) + briefing (tail+Send) bodies, HubPanel/App wiring, CSS. Full suite was 1686 green + tsc clean at 134-05.
  - **Code review (134-REVIEW.md)** found 3 blockers + 7 warnings. CR-01/CR-02: the remote-session modal could never connect (mounts LOCAL relay with a REMOTE id; Phase 122 cap only proxies file-browse, no relay-WS proxy). CR-03: RelayClient/WS leak + untrusted-text race + no unmount cleanup.
  - **User decision: "Build remote relay-WS proxy now"** — expands MODAL-06 past its "no new remote-access architecture" constraint. See memory `project_modal06_remote_ws_proxy`.
  - Research `134-RESEARCH-remote-ws-proxy.md` (HIGH confidence) + 3 expansion plans **134-06/07/08** (waves 5/6/7) created & plan-checked (VALIDATION.md regenerated to cover them).
  - **134-06 DONE** (commit `e96a1ef7` chain): Go cap-gated remote terminal WS proxy on the daemon (`internal/daemon/remote_ws_proxy.go`, route `GET /api/relay/remote/{sid}/ws`, exported `relay.LoopbackOriginPatterns`). WS-PROXY-01..06 green under `-race`, build/vet/fmt clean.
  - **134-07 NOT DONE** (quota hit mid-execution; partial uncommitted test was reverted — tree clean, no SUMMARY).

## Resume here (after Jun 20 8am)
Run `/gsd-autonomous` (or `/gsd-autonomous --from 134`) in a fresh session. It will detect 134-01..06 complete and resume at **134-07**, then **134-08**, then phase verify → **Phase 135 (A11Y Hardening)** → milestone audit/complete/cleanup.

Config note: `workflow.use_worktrees=false` is set (sequential executors on main tree — harness worktree branch-namespace mismatch). Keep it.

### Remaining Phase 134 plans
- **134-07** (wave 6, deps 134-06): Frontend remote wiring. RelayClient `remote`/proxy-URL seam (remote→daemon proxy route, cap stays in daemon NOT React state; fixes CR-01); HubInteractiveModal mounts remote-routed TerminalPanel; HubBriefingModal reads remote tail from WS snapshot frame (NOT GetSessionTailLines — that's the CR-02 bug); **CR-03 fix** (close RelayClient/WS on timeout reject, `settled` guard vs late-onOpen send, unmount ping-interval cleanup); FE-URL-01 behavioral test. Run `pnpm test --run` + `pnpm exec tsc --noEmit`.
- **134-08** (wave 7, deps 134-07): Thread `isRemote` to the seam; fix WR-01 (join-modal pending-session strand on dismiss), WR-02 (don't overwrite in-flight file-browse intent), WR-03 (cast removal), WR-04 (real per-session fontSize, not hardcoded 14); WR-07 behavioral tests (remote gate, briefing send/timeout, remote tail). **WR-05/WR-06 explicitly DEFERRED to Phase 135 a11y** — note in-code, do not fix here.

### Carry-forward warning (from plan-check)
- **Read-only cap UX gap:** briefing Send presents as enabled for a read-only remote cap (input silently dropped at peer). Read-only non-color indicator deferred to **Phase 135** (colorblind-safe is release-blocking). 134-08 should document the silent-drop at the call site.

## Open UAT / findings to carry
1. **Phase 134 live UAT (manual, at phase gate):** real two-machine tailnet — open a remote session card → join-code cap exchange → modal mounts terminal; type a command, confirm it runs on the peer; verify resize/scrollback/copy-paste; briefing respond round-trip; read-only-cap behavior. No automated substitute.
2. **Attention live UAT (Phase 133)** — still pending; ATTN-05 (modal-resolve clears attention) can now be tested once 134 modal lands.
3. **"Stuck Running"/status detector** — pre-existing heuristic (`internal/daemon/engine.go`), candidate for a separate issue.
4. Pre-existing (do NOT flag as regression): `internal/release` SER-03 stale gitignored `cmd/playwright-fixture/dist` artifact.

## Live UAT recipe (Wails)
`wails dev` → external browser at http://localhost:34115 runs Go RPC + CSS inspection but NOT terminal PTY. For terminal/attention/preview UATs use the NATIVE wails dev window. For remote, need a second peer.
