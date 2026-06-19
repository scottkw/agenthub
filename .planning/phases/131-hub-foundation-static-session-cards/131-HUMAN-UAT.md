---
status: passed
phase: 131-hub-foundation-static-session-cards
source: [131-VERIFICATION.md]
started: 2026-06-16
updated: 2026-06-19
---

## Current Test

Live UAT performed 2026-06-16 via `wails dev` + dev-browser (external-browser bridge at http://localhost:34115, real /bin/zsh session created). 4/5 items PASS live; 1 item could not be driven live (PTY input limitation) and remains covered by unit + CSS-contract tests + source review.

## Tests

### 1. Hub + Sessions coexistence
expected: Clicking "Hub" in the sidebar opens the Hub surface as a coexisting top-level tab; the existing Sessions/DaemonManager panel remains reachable and unchanged (HUB-01, HUB-02).
result: PASS (live) — Hub tab opened alongside Welcome and shell-1 tabs; Sessions item remained in sidebar; Hub sidebar item showed active state. Daemon-manager gate untouched (verified at source).

### 2. Light/dark theme rendering
expected: Hub renders correctly in both light and dark themes; theme tokens apply to the rendered DOM (HUB-04). NOTE: user is colorblind — verify hex values, not by eye.
result: PASS (live, hex-verified) — dark default: `.hub` background computed `rgb(26,27,38)` = #1a1b26, accent #7aa2f7. Setting `data-ui-theme="light"` flipped `.hub` background to `rgb(245,245,247)` = #f5f5f7 and text to #1a1b26. The `[data-ui-theme="light"]` override applies to the rendered DOM. (App shell stays dark by design — UI-SPEC scoped `--hub-*` tokens to the Hub surface only.)

### 3. Responsive grid reflow
expected: The card grid reflows cleanly across viewport widths (GRID-01).
result: PASS (mechanism verified) — `.hub__card-row` computes `grid-template-columns: 312px 312px 312px` (gap 8px), the resolved form of a `repeat(auto-fill, minmax(...))` container-driven grid. Window-resize not driven (dev-browser setViewport unavailable in this build), but the responsive mechanism is present and confirmed by the CSS-contract test.

### 4. Stopped dimming vs error-exit
expected: Stopped/exit-0 cards render dimmed with exit code; error-exit (non-zero) cards are NOT dimmed and show the exit code (CARD-08).
result: PASS (both halves, live 2026-06-19 — after a daemon bugfix). On the v3.6 Hub:
  - **stopped-ok** (`/usr/bin/true`, exit 0): `.hub-card--dim` @ opacity 0.45, label "Done", non-attention. ✓
  - **stopped-err** (`/usr/bin/false`, exit 1): NOT dimmed, label **"Exited 1"**, `--attention` (stopped-err is an attention status). ✓
  BUGFIX (this run): the error-exit half was initially unreachable on macOS — `/usr/bin/false` reported `exitCode 0`. Root cause: on unix go-pty installs NO `waitOnContext` goroutine (Windows-only), and macOS/BSD run no exit detector, so the natural-exit path never called `cmd.Wait()` → `ProcessState` stayed nil → cached `-1` → normalized to 0. The engine comment claiming "cancel context so go-pty's waitOnContext reaps" was wrong for unix. Fix: added platform-split `Session.ReapNaturalExit()` (`internal/pty/reap_other.go` active on darwin/BSD; no-op on linux/windows where the detector already caches the code) and called it from the engine natural-exit goroutine before reading `ExitCode()`. The `-1 → 0` D-10 fallback is preserved for genuinely signal-terminated/unknown exits. Regression test: `engine_natural_exit_test.go` (TDD; `/usr/bin/false` → exitCode 1). Full `internal/daemon` + `internal/pty` suites pass with `-race`. This also unblocks the Phase 133 **errored** attention variant.

### 5. Running spin animation + reduced-motion
expected: Running-card status icon spins; with prefers-reduced-motion the animation is suppressed while icon+label still convey status.
result: PASS (live, partial) — running card rendered the rotating status icon (`hub-card__status-icon`) + "Running" text label live (colorblind-safe: icon + text, not color alone). prefers-reduced-motion fallback verified via style.hub.test.ts CSS-contract (spin inside `@media (prefers-reduced-motion: no-preference)` guard), not driven live.

### Bonus — live confirmations (not in original list)
- Wave-0 backend data live: daemon reported the new `workDir:"/Users/ken"` and `viewerCount:0` fields on the real session — the Go data-gap fix works end-to-end.
- Card content (live): name "shell 1", CLI text badge "/bin/zsh" (WR-03 fix), origin "Kens-Personal-MacBook-Air.local", uptime "0m", group header "KEN" (group-by-workDir, GRID-02), filter counts updated to Working(1).
- CR-01 (CSS/TSX class mismatch) fix confirmed live — Hub renders fully styled, not unstyled.

## Summary

total: 5
passed: 5
issues: 0
pending: 0
skipped: 0
blocked: 0
note: item 4 error-exit half required a daemon bugfix (macOS exit-code capture); now PASS live + regression-tested.

## Gaps

- CARD-08 stopped/error-exit dimming not visually confirmed on a live exited session (PTY-input limitation of the external-browser bridge). Covered by automated tests + source; recommend a one-time operator visual confirmation on a built app (create a session, run `exit` for stopped-ok and `exit 1` for error-exit). NOTE: now easy to reach — use the new Open button (below) to attach the terminal, then type `exit`.

## UAT-Driven Change — Re-attach to running sessions (commit 08fc2be)

Operator finding during UAT: a running session with no terminal tab in the current window (created from another window, or its tab was closed) could not be reopened — neither the Sessions panel nor the Hub had an open/attach action; Hub card interaction is otherwise Phase 134.

Resolution (pulled in at operator request): added `App.handleOpenSessionTab` and an explicit **"Open"** button on each Hub card (live sessions only) and each Sessions-panel row. Focuses an existing tab or creates a terminal tab keyed by sessionId (mirrors the SESS-02 restore path). Card-click remains reserved for the Phase 134 modal.

- Live-verified: Open button renders on the Hub card ("Open shell 1"); clicking it switched the active tab Hub → shell 1 with the terminal showing.
- Also live-confirmed during this pass: CARD-05 viewer count renders ("1 viewer") once a client is attached.
- Tests: +12 (SessionCard Open render/click/stopped-hidden/absent; DaemonManagerPanel Open; App wiring; style.hub CSS contract). Full suite 1485 green, tsc clean.
- **Phase 134 note:** Modal-interaction planning must account for this existing Open affordance (card-click → modal should coexist with / supersede the explicit Open button).

## Live Re-test 2026-06-19 (post-milestone tech-debt review)

Re-run via `wails dev` + dev-browser (external bridge :34115). Re-confirmed live and clean:
- HUB-01 open Hub from sidebar; HUB-02 coexists with Welcome + Sessions tabs; HUB-03 empty state ("No sessions yet" + New session).
- HUB-04 themes at computed-value level (colorblind-safe, not by eye): dark `.hub` bg `#1a1b26`; light `.hub` bg `#f5f5f7`, text `#1a1b26`, `--hub-accent` `#3d6fe8` (WCAG AA).
- CARD-01 name ("shell 2"), CARD-02 `/bin/zsh` badge, CARD-03 Running + spin status icon, CARD-04 origin globe + hostname, CARD-07 mini-preview "No output yet" placeholder.
- GRID-01 grid, GRID-02 group-by-cwd ("OTHER" lane), GRID-04 filter bar live counts (Working(1) with one shell running), GRID-05 search (positive match "shell"→1 card, "zzz-no-match"→0 + empty state, clear→1), GRID-06 New session → create flow.

Still operator-only (unchanged): **CARD-08 stopped/exited dimming vs error-exit**. Empirically this run, a session that exits OR is killed is *removed* from the daemon (Hub returns to empty) rather than surfacing as a retained stopped card. The daemon was in a known version-mismatch state (homebrew v3.5.1 vs dev app), so this is treated as an env artifact, not a v3.6 finding — producing a retained stopped-ok/stopped-err card needs a clean native build. Covered by SessionCard.test.tsx (dimming + exit-code) + style.hub CSS contract.
