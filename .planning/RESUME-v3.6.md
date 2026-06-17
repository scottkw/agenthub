# v3.6 Hub — Resume Handoff (paused 2026-06-17)

Autonomous run (`/gsd-autonomous`) paused at ~68% context after Phase 133. Resume in a fresh session.

## Status
- **Phase 131** (Hub Foundation + Static Cards) — ✅ complete & shipped. Live-UAT'd; added a re-attach "Open" button (commit 08fc2be) on Hub cards + Sessions rows (front-runs part of Phase 134 — see memory).
- **Phase 132** (Unified Grid + Mini Preview + Named Groups) — ✅ complete & shipped. Live smoke-tested (group sidebar/create/assign/collapse all pass). Fixed: chevron/icon Tailwind no-op sizing (ff797fab); **mini-preview "Loading…" local/remote provenance bug (8ee92a0d, found during 133 UAT)**.
- **Phase 133** (Attention + Pulse) — ✅ complete & verified (1638 frontend tests green, tsc clean). Code review found+fixed 3 critical/4 warning/2 info. **Attention live-UAT still pending** (see HUMAN-UAT) — needs a session in waiting/errored/stopped-err state.

## Resume here
Run `/gsd-autonomous` (or `/gsd-autonomous --from 134`) in a fresh session. Next: **Phase 134 (Modal Interaction)** → **Phase 135 (A11Y Hardening)** → milestone audit/complete/cleanup.

Per-phase flow used: skip_discuss=true → write minimal CONTEXT → gsd-ui-phase → research → pattern-map → plan → plan-check → execute (sequential gsd-executor agents on main tree, NOT worktrees — harness worktree branch namespace mismatch) → code-review + fix → verify → (offer live UAT) → complete.

## Open UAT / findings to carry
1. **Attention live UAT (Phase 133)** — pulse border, debounced float-to-top + FLIP timing, collapsed-group attn badge. Need a real attention session. ATTN-05 (modal-resolve clear) is BLOCKED on Phase 134 — re-test once the modal exists.
2. **Phase 134 must preserve the re-attach Open button** (card-click→modal should coexist; see memory `project_phase134_reattach_button`).
3. **"Stuck Running" / status detector** — NOT a Phase 133 bug. Pre-existing heuristic detector (`internal/daemon/engine.go:428`, `status` pkg) classifies running/idle/waiting/errored from PTY output. If `waiting`/`needs-input` is unreliable, attention + Phase 131 "Needs input" filter are under-triggered. Candidate for a separate scottkw/agenthub issue + investigation (not milestone-blocking by itself).
4. **ROADMAP cosmetic**: Phase 132/133/134/135 "Plans:" checklists were stale-copied from Phase 131 boilerplate (133's was corrected during planning; check 134/135 at plan time or milestone-close).
5. Pre-existing (do NOT flag as regression): `internal/release` SER-03 fails on a stale gitignored `cmd/playwright-fixture/dist` artifact (predates milestone; no Go touched in 132/133).

## Live UAT recipe (Wails)
`wails dev` → external browser at http://localhost:34115 (dev-browser drivable) runs Go RPC + CSS inspection but NOT terminal PTY. For terminal/attention/preview-content UATs use the NATIVE wails dev window. dev-browser binary: `node "$(npm root -g)/dev-browser/bin/dev-browser.js"`.
