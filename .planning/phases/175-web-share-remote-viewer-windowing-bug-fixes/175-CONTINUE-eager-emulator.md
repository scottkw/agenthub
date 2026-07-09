# Phase 175 — CONTINUE HERE: eager-emulator fix (M-51 header garble)

**Status when paused:** 2026-07-08. Phase 175 executed + verified (human_needed), then LIVE UAT
found + fixed 3 issues. **M-51 is the ONLY open item.** All other work is committed and clean.
`go test ./...`, relay+webserver `-race`, `tsc`, and vitest are all green.

## ✅ CODE FIX LANDED — 2026-07-08, commit `8268155c`

The eager-emulator fix is **implemented, TDD-tested, and committed** (`fix(175): eager-build live
emulator from first PTY byte`). `recordFrame` now builds + feeds the emulator from the first PTY
byte; `EnsureLiveEmulator` is a safety no-op (no more bootstrap-from-scrollback); dead
`stripMsgOutputBytes` removed. New RED→GREEN test
`TestReconnectPreamble_EagerEmulatorCapturesHeaderBeforeWrap` guards it. Gates all green: `go build`,
`go vet`, relay `-race`, webserver `-race`, full `go test ./...`.

**REMAINING = the live re-test only** (needs the user + a rebuilt prod app — see "Live re-test"
below). Once it passes, record **M-51 PASS** in `175-UAT.md` and finalize the phase (hand-edit
STATE/ROADMAP; do NOT run `gsd query phase.complete`).

## The one remaining task (DONE — kept for reference)

Make the per-hub VT emulator **eager**: build it and feed it from the **first PTY byte** (in the
Run() drain loop), instead of lazily on the first guest connect. This is the agreed fix for the
M-51 residual (see root cause below). The user chose this over shipping-with-a-follow-up.

### Why (root cause — already fully diagnosed, do NOT re-derive)

- M-51 scenario: a shared **shell** session running **`top`** (macOS `top` renders in the MAIN
  screen via absolute cursor positioning — NOT the alternate screen), viewed by a **late-joining**
  guest **after** the 256 KiB scrollback ring has **wrapped**.
- The reconnect preamble is chosen by `Hub.ReconnectPreamble()` (internal/relay/hub.go). It already
  correctly uses the **ring-wrap discriminator** (commit fbf5dd8e): raw replay while the ring is
  intact (`Scrollback.Truncated() == false`), emulator `RenderSnapshot()` once the ring wrapped.
- With that fix, `top`'s **body** now renders cleanly (top full-repaints the process rows each cycle,
  so they self-heal), but the **header** (Processes / Load Avg / PhysMem / column-header rows) stays
  **garbled**. See guest screenshot in session history (image #7).
- Reason: `EnsureLiveEmulator()` builds the emulator **lazily at first guest connect** and bootstraps
  it from `ScrollbackSnapshot()`. In M-51 that first connect is LATE — the ring already wrapped — so
  the bootstrap is **incomplete**. `top` updates its header **in place** (overwrites only the changing
  numbers, never clears+rewrites the header structure), so the garbage from the truncated bootstrap in
  the top rows is **never overwritten** → persistent header garble.
- Conclusion: the emulator only holds a COMPLETE screen if it has been alive and fed **since before
  the wrap**. Lazy-on-first-connect can't guarantee that. `Render()` itself is faithful (verified: it
  preserves column gaps), so the emulator snapshot is the right source — it just needs complete state.

### Implementation plan (TDD)

1. **Build + feed the emulator from the first PTY frame.** In `internal/relay/hub.go`:
   - Change `recordFrame(frame, raw []byte)` (the drain-loop writer that already holds `emuMu` and
     does scrollback-append + emulator-feed atomically — CR-03) so that on the FIRST frame, if
     `h.liveEmu == nil`, it **builds** the emulator (mirror `EnsureLiveEmulator`'s construction:
     `xvt.NewEmulator(h.Cols(), h.Rows())` + `SetScrollbackSize(liveEmulatorScrollbackLines)`) and
     then feeds this frame. No bootstrap-from-scrollback needed — being fed from byte 1 IS the
     complete history. (Mind lock order: `emuMu` → `hub.mu` via Cols()/Rows(), same as today; no
     inversion.)
   - Net effect: emulator is alive from the first byte of output, fed every frame, resized by CR-01's
     `resizeLiveEmulator` on every PTY resize → always holds the complete current screen.
2. **`EnsureLiveEmulator()` becomes a safety no-op.** With eager build, `liveEmu` is already set once
   any output has flowed. Keep the function (both WS connect sites still call it) but it should NOT
   re-bootstrap from scrollback (that path is now dead / would be wrong). Simplest: if `liveEmu != nil`
   return; else build empty (fallback for a session that connected before any output). Verify the two
   call sites (relay/server.go ~321, webserver/server.go ~1492) still compile + behave.
3. **Keep** the ring-wrap discriminator in `ReconnectPreamble` (fbf5dd8e), CR-01 `resizeLiveEmulator`,
   CR-03 `recordFrame` atomicity, and `Scrollback.Truncated()` — all still needed.
4. **TDD test** (internal/relay/scrollback_altscreen_test.go): simulate top's in-place header —
   write `ESC[H` + "HEADER-XYZ" (a positioned header) + body, then WRAP the ring with filler,
   WITHOUT any pre-wrap `EnsureLiveEmulator` call. Assert `ReconnectPreamble()` (wrapped → emulator)
   still contains "HEADER-XYZ". This is RED with lazy build (emulator built late from truncated ring
   never saw HEADER-XYZ) and GREEN with eager build (emulator saw it at byte 1). Also keep all
   existing preamble/altscreen/resize/fanout tests GREEN.
5. **Gates:** `go build ./...`, `go vet ./...`, `go test ./internal/relay/... -race`, full
   `go test ./...`. Then commit.

### Perf note (the trade-off the user accepted)

A lightweight `charmbracelet/x/vt` emulator now parses every PTY byte for EVERY session continuously
(not on-demand). Bounded by `SetScrollbackSize(50)`. Acceptable per the user's choice; if it ever
matters, a middle ground is to start feeding at web-serve-enable time instead of session start (less
robust — fails if the ring wrapped before sharing was enabled).

### Live re-test (needs the user + a rebuilt prod app)

```
AgentHub daemon stop 2>/dev/null; pkill -f agenthub
bash build.sh --platform macos
open build/bin/agenthub.app
```
Then: share a shell session, run `top`, let it run ~5 min (wrap the ring), open a FRESH Incognito
guest tab, join. Expected: the late-join renders the FULL top screen cleanly — header AND body.
Record M-51 PASS in `175-UAT.md`, then finalize the phase.

## Live UAT scoreboard (175-UAT.md is authoritative)

- **M-48** mobile readability floor (BUG-01/#128) — **PASS** live (real iPhone).
- **M-49** disconnect banner (BUG-02/#125) — **PASS** all 3 triggers (exit, disconnect-viewers,
  disable-sharing). Disable-path gap fixed in commit 501aef9e.
- **M-50** BUG-03 (#126) — satisfied by 175-01 DISPROVED diagnosis (non-reproduction; instrumentation).
- **M-52** #109 remote-layout regression (host/guest size mismatch) — **PASS** live (Claude clean).
  Fixed by the hybrid raw-replay-by-default reconnect preamble (commit 460b597f).
- **M-51** late-join TUI after ring-wrap (BUG-04/#119 P2) — **PARTIAL**: body clean, header garbled.
  Ring-wrap discriminator landed (fbf5dd8e); **eager-emulator fix is the remaining work (this doc).**

## Key commits this UAT round (branch v4.2-funnel-sharing)

- `460b597f` fix(175): hybrid reconnect preamble — raw replay by default (fixes #109 regression M-52)
- `501aef9e` fix(175): disconnect web viewers on web-serve toggle-off (M-49 disable-path)
- `fbf5dd8e` fix(175): reconnect discriminator = ring-wrapped, not alt-screen (M-51 top body)
- Plus the earlier code-review fixes (CR-01/CR-02/CR-03) in d4c309db and the phase's 7 plans.

## Reminders

- Working tree has PRE-EXISTING unrelated dirt (do NOT touch/commit): `frontend/src/wailsjs/.../models.ts`,
  `node_modules/.vite/...`, untracked `agenthub-v4.0-redesign/`, `dev-run.sh`, `frontend/uat-16*/17*-harness.*`.
- Phase 175 is NOT yet marked complete in ROADMAP/STATE (correct — human_needed until UAT closes). Do
  NOT run `gsd query phase.complete` until M-51 passes; hand-edit STATE/ROADMAP per project convention.
- User is colorblind: verify any color-related behavior at the hex/source level, not by eye.
