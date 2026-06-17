# v3.6 Hub — Resume Handoff (paused 2026-06-17, weekly quota limit)

Autonomous run (`/gsd-autonomous`) **paused mid-Phase-134** — hit the weekly Claude usage limit. **Resets Jun 20, 8am (America/Chicago).** Resume in a fresh session after that.

## Status
- **Phases 131, 132, 133** — ✅ complete & shipped (see git history). Attention live-UAT for 133 still pending (HUMAN-UAT).
- **Phase 134 (Modal Interaction)** — ⏳ IN PROGRESS.
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
