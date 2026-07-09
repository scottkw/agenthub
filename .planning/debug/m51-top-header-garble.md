---
slug: m51-top-header-garble
status: deferred
deferred_to_issue: 130
resolution_summary: "Two root causes found + fixed (eager build 8268155c, rebuild-on-resize 44078d41) — these fixed the agent-TUI / static / resize-churn cases. A THIRD residual (macOS top ONLY: header garble on late-join after ring wrap, from a headless-emulator scroll-region/in-place-update fidelity gap that top never self-heals) is a P-low shell edge case; deferred to issue #130 and accepted by the user 2026-07-08 to finalize Phase 175. Agents (primary use case) pass."
trigger: "M-51 (Phase 175, BUG-04 / issue #119 P2): a late-joining web-share guest viewing a shared shell session running macOS `top` sees a GARBLED header block (Processes:/Load Avg:/CPU usage:/PhysMem: labels lost or interleaved with process-row content), while the process-row body renders reasonably. This persists AFTER the eager-emulator fix (commit 8268155c) on a freshly rebuilt prod app."
created: 2026-07-08
updated: 2026-07-08
---

# Debug: M-51 top header garble (late-join guest, post eager-emulator fix)

## Symptoms

- **Expected:** A guest joining a shared shell session AFTER `top` has been running long
  enough to wrap the 256 KiB scrollback ring should see the FULL `top` screen render
  cleanly — header block AND process-row body.
- **Actual:** The process-row body renders roughly correctly, but the header block is
  garbled: the structural labels (`Processes:`, `Load Avg:`, `CPU usage:`, `SharedLibs:`,
  `MemRegions:`, `PhysMem:`, `VM:`, `Networks:`, `Disks:`) are missing or interleaved with
  fragments of process rows and numbers. See guest screenshot (image #8 in session history).
- **Error messages:** None — purely a rendering defect.
- **Timeline:** Phase 175 UAT. The earlier lazy-emulator build had this same header garble;
  commit `8268155c` (eager-build the emulator from the first PTY byte) was expected to fix it
  but did NOT — the rebuilt prod app still shows the garble.
- **Reproduction:** Share a shell session; run `top`; let it run ~5 min to wrap the ring;
  open a FRESH Incognito guest tab; join. Header is garbled on the late-join render.

## Evidence

- timestamp: 2026-07-08 — Running daemon is pid 74469, launched 21:22 from
  `build/bin/agenthub.app/Contents/MacOS/AgentHub` (mtime 21:22, i.e. built AFTER commit
  8268155c which is in `git log`). So the build is temporally after the fix, but binary
  identity is NOT yet positively confirmed.
- timestamp: 2026-07-08 — `go tool nm` on the app binary shows NO symbols for
  `stripMsgOutputBytes`, `recordFrame`, OR `EnsureLiveEmulator` — the binary is
  symbol-stripped (Wails production build uses `-ldflags "-s -w"`). So the "stripMsgOutputBytes
  absent = my fix is in" check is INCONCLUSIVE (everything is absent). `go version -m` shows
  build tags `desktop,wailsassets,wv2runtime.download,production` but no clean `vcs.revision`
  line surfaced. **Binary-contains-fix is still UNCONFIRMED.**
- timestamp: 2026-07-08 — Unit test `TestReconnectPreamble_EagerEmulatorCapturesHeaderBeforeWrap`
  (internal/relay/scrollback_altscreen_test.go) PASSES: with the eager build, a one-time
  positioned header written before a ring wrap IS present in `ReconnectPreamble()`. So the fix
  works for the *synthetic* header-persistence model. The live failure means either (a) the
  binary lacks the fix, or (b) the synthetic model does NOT match how macOS `top` actually
  drives the terminal.
- timestamp: 2026-07-08 — checked: `git diff --stat HEAD -- internal/relay/hub.go` is EMPTY
  (zero uncommitted changes); `git log --oneline 8268155c..HEAD` shows exactly one commit,
  `e4b89b75`, which `git diff --stat 8268155c e4b89b75 -- '*.go'` proves touches ZERO Go files
  (docs-only, a `.md` handoff file). Commit 8268155c landed 21:19:46, docs commit e4b89b75 at
  21:20:11, binary built at 21:22. Since `dev-run.sh` does `wails build -tags wailsassets` with
  NO git checkout step (compiles whatever's on disk), and the working tree's hub.go is
  byte-identical to HEAD which includes 8268155c — **binary-contains-fix confidence raised to
  HIGH** (not 100%/forensic, but as conclusive as static analysis allows on a stripped binary).
- timestamp: 2026-07-08 — built `zzz_manual_top_capture_test.go` (temporary): captured 10s of
  REAL macOS `top -s 1` raw PTY output. `top` DOES enter alt-screen (`\x1b[?1049h`), sets a
  scroll region (`\x1b[1;24r`), does ONE full clear+header write at start, then EVERY
  subsequent cycle uses ONLY targeted VPA (`\x1b[<n>d`)/CUP/backspace repositioning + DCH
  (`\x1b[NP`, delete-char) to patch numbers in place — confirms the "header labels written
  once, only patched thereafter" model, but reveals `top`'s incremental updates are far more
  surgical (stateful DCH/VPA splicing) than the synthetic test's simple model.
- timestamp: 2026-07-08 — fed the SAME real-top capture into 3 emulators: (a) built at the
  real 80x24 size, (b) built+left at the Hub.Cols()/Rows() FALLBACK 220x50 (simulating "no
  local subscriber ever resized"), (c) built at 220x50 then resized down to 80x24 mid-stream.
  All three rendered IDENTICALLY CLEAN. **ELIMINATES pure geometry/width mismatch (fallback vs
  real PTY size) as a standalone cause** — `xvt`'s absolute positioning is self-describing and
  tolerant of a wider-than-real emulator.
- timestamp: 2026-07-08 — tested whether `xvt.Emulator.Write()` correctly buffers a partial
  escape sequence split across separate `Write()` calls (mirrors production: `recordFrame`
  calls `emu.Write()` once per PTY `Read()` chunk, not once for the whole stream), including
  byte-at-a-time feeding. Render output was IDENTICAL to a single whole-stream write in both
  cases. **ELIMINATES "parser state lost across Write() call boundaries" as a cause.**
- timestamp: 2026-07-08 — found a REAL (but likely secondary/unconfirmed-for-this-exact-bug)
  defect: `liveEmuQueryStripPattern` is applied PER-CHUNK inside `recordFrame` (once per PTY
  `Read()`, ≤32 KiB), not on the accumulated stream. If a query/DSR sequence the pattern is
  designed to strip (e.g. `\x1b[6n`) straddles a chunk boundary, each half fails the regex
  independently and the UNSTRIPPED literal bytes get fed into `emu.Write()` — proven via
  `TestManualSplitWrite_StripPerChunkVsWhole`. Real 10s and 270s captures of bare `top` alone
  contained NONE of the query-strip-matched sequences, so this bug's relevance to the EXACT
  M-51 repro (bare `top`, no themed shell prompt) is unconfirmed, but it's a legitimate latent
  defect in the fix's own anti-hang hardening layer, independent of the primary finding below.
- timestamp: 2026-07-08 — built a fully faithful E2E harness (temporary, since deleted): a REAL
  production `relay.Hub` (not a mock) wired to a REAL macOS `top -s 1` process via `go-pty`,
  using the real `DefaultScrollbackBytes` (256 KiB) ring, a real local Subscriber calling
  `ResizeClient(80,24)` IMMEDIATELY (matching the best-case "TerminalPanel connects and
  resizes before top ever draws" scenario), run for 4m30s until the ring genuinely wrapped.
  The late-join snapshot (`EnsureLiveEmulator()` + `ReconnectPreamble()`) was **PERFECTLY
  CLEAN** — full 8-line header intact, PID table correct. **This confirms the eager-build fix
  (8268155c) mechanism itself is CORRECT for a static, never-resized-after-start geometry** —
  matching the synthetic unit test's conclusion, extended to a real `top` process.
- timestamp: 2026-07-08 — separately confirmed real macOS `top` DOES emit a full
  `\x1b[H\x1b[2J` + complete header rewrite in direct response to a PTY resize (SIGWINCH) —
  tested by resizing a live `top` PTY from 154x42 down to 60x20 and inspecting the immediate
  post-resize byte stream. So `top` is NOT purely "write header once, never again" — it DOES
  restructure/redraw its header around resize events (and the field layout differs by
  available rows: a compact 5-field layout at 20 rows vs the full 8-field layout at more rows).
  This directly undermines the "header is written exactly once" mental model the M-51 fix's
  design comments assume, widening the real attack surface for state divergence.
- timestamp: 2026-07-08 — **ROOT CAUSE REPRODUCED**: same real-Hub E2E harness, but with
  deliberate mid-session geometry churn: start wide (154x42) → after 4s, `ResizeClient` to
  narrow (60x20) → after 3s, `ResizeClient` back to wide (154x42) → forced-small (20 KiB)
  scrollback ring wraps within ~1 minute → late-join snapshot inspected. Result: header LABELS
  are all present but STRUCTURALLY WRONG — 3 originally-separate header lines collapsed onto
  ONE row each ("Load Avg:...CPU usage:...SharedLibs:..." merged; "MemRegions:...PhysMem:..."
  merged; "VM:...Networks:..." merged), while the PID/process-table BODY rendered cleanly and
  correctly (including the wider 154-col column set: STATE/BOOSTS/%CPU_ME/etc). **Reproduced
  twice independently with `-count=1`** (fresh `top` processes, different PIDs/data each time,
  identical structural corruption pattern both times) — not a one-off timing fluke. This
  exactly matches the reported symptom: "structural labels... missing or interleaved with
  fragments of process rows" (header) vs "process-row body renders roughly correctly" (body).

## UPDATE 2026-07-08 (post-fix live UAT) — BOTH FIXES INSUFFICIENT; new discriminating evidence

- timestamp: 2026-07-08 — Live UAT on a rebuilt prod app containing BOTH fixes (8268155c eager
  build + 44078d41 rebuild-on-resize): `top` header STILL garbled on late-join guest (image #9).
  So neither the eager-build nor the destructive-resize theory is the (whole) cause. The
  deterministic unit tests pass but do NOT capture the real failure — my model of `top` is wrong.
- timestamp: 2026-07-08 — **KEY USER OBSERVATION: `top` is the ONLY thing that garbles. All the
  AGENT TUIs (Claude Code, Gemini, etc.) display fine on late-join** — including after ring wrap.
  Agents are AgentHub's primary use case and they pass; `top` (a shell edge case) is the sole
  failure. #119 is P2.
- timestamp: 2026-07-08 — **Refined symptom read of image #9:** the garble is NOT "missing
  labels" — it is PROCESS ROWS bleeding UP into the header region (PID 3131 `bzfilelist`,
  `Signal Helpe`, `Google Chrom` process rows appear in rows 1–7, right-hand side), while the
  body process table below renders coherently. Process content reaches header rows only via
  SCROLLING (unprotected scroll region) or absolute-cursor positioning onto a mis-sized grid.
- hypothesis (revised, UNVERIFIED): the differentiator is that agent TUIs do frequent FULL
  differential repaints (alt-screen), so ANY reconstruction error self-heals within ~1 frame;
  `top` draws its header structure ONCE, protects it with a SCROLL REGION, and thereafter only
  patches numbers in place (surgical VPA/DCH) — so a reconstruction error in the header is
  PERMANENT (top never rewrites those cells). The bleed pattern points specifically at
  scroll-region handling: if the reconstructed/continuously-fed emulator's scroll region does
  not match the host's, top's process-list scroll pushes rows into the header. This is a
  headless-emulator FIDELITY gap (charmbracelet/x/vt vs the host xterm.js) and/or a width/height
  mismatch — NOT the two things already fixed.
- OPEN GROUND-TRUTH QUESTION (must answer before any fix #3): does the garble exist in the
  SERVER-SIDE reconnect preamble bytes the guest receives (emulator render or raw replay), or is
  it introduced CLIENT-SIDE by xterm.js at a mismatched width? Also: which ReconnectPreamble
  branch fires (raw vs emulator) and is the ring even truncated at join time? Not yet captured.
- DECISION PENDING (scope): agents pass; `top` is a P2 shell edge case. Chase it further (needs
  live-path instrumentation + more rebuild cycles) vs. document as a known limitation and
  finalize. Escalated to the user.

## Current Focus

- hypothesis: CONFIRMED — mid-session PTY/emulator geometry resize churn (a local viewer's
  cols/rows changing, e.g. window resize / sidebar toggle / font-size change / the
  mount-then-settle double-fit that xterm.js's FitAddon commonly does) combined with
  `xvt.Emulator.Resize()`'s destructive non-reflow truncate/pad grid semantics causes the
  live per-hub emulator's header rows to retain STRUCTURALLY stale content from an
  intermediate geometry. `top` only performs a full header re-layout in direct response to a
  resize (SIGWINCH), so any single missed/partial capture of that redraw becomes permanently
  baked into the live emulator's state (never self-heals) — while the process-table BODY is
  fully rewritten every ~1s cycle regardless of geometry, so it self-heals immediately and
  "renders roughly correctly" exactly as reported.
- test: Built `internal/relay/zzz_manual_resize_churn_test.go` (temporary, now deleted) using
  the REAL production Hub (recordFrame/EnsureLiveEmulator/ReconnectPreamble/RenderSnapshot)
  with a REAL macOS `top` process as PTY reader/writer: wide(154x42) start → shrink to
  60x20 after 4s (via hub.ResizeClient, which resizes both the live emulator AND the real PTY,
  matching D-02 min-among-local) → grow back to 154x42 after 3s → let a small (20 KiB)
  scrollback ring wrap → simulate late-join via EnsureLiveEmulator+ReconnectPreamble.
- expecting: (met) The late-join snapshot showed header labels present but STRUCTURALLY
  merged onto wrong lines ("Load Avg:...CPU usage:...SharedLibs:..." all on ONE row instead
  of 3 separate rows; "MemRegions:...PhysMem:..." merged; "VM:...Networks:..." merged) while
  the PID/process table body rendered cleanly and correctly (including the WIDER 154-col
  column set). Reproduced twice with `-count=1` (fresh top processes, different PIDs/data,
  same structural corruption pattern both times) — NOT a one-off fluke.
- next_action: DONE (diagnose-only mode) — root cause confirmed and reported; no fix applied.
  Parent/specialist to design the fix (see Suggested Fix Direction in structured report).
- reasoning_checkpoint:
    hypothesis: "Mid-session hub.ResizeClient geometry churn (shrink then grow, or any resize
      after the header has been drawn once) corrupts the live per-hub emulator's header rows
      because xvt's Buffer.Resize is a destructive grid truncate/pad (not a reflow), and top's
      full-header-redraw-on-SIGWINCH is the ONLY mechanism that can correct it — any frame lost
      or misapplied around a resize event leaves the corruption permanently baked into the
      continuously-fed live emulator with no future correction."
    confirming_evidence:
      - "Real-Hub E2E test with a SINGLE unchanging geometry (80x24 from t=0, real top, real
        256 KiB ring wrap at t=4m30s) rendered the late-join header PERFECTLY CLEAN — proves
        the eager-build mechanism itself (commit 8268155c) works correctly absent resize churn."
      - "Real-Hub E2E test with shrink(60x20)-then-grow(154x42) resize churn, real top, small
        ring forced to wrap, showed the SAME structural merge-corruption pattern reproducibly
        across two independent runs with fresh top processes."
      - "Separately confirmed real macOS top DOES emit a full \\x1b[H\\x1b[2J + full header
        rewrite in direct response to SIGWINCH (TestManualResizeBehavior) — so the corruption
        is not simply 'top never redraws the header'; it's that the live emulator's captured
        state around a resize transition can end up structurally wrong despite that redraw."
      - "Confirmed geometry MISMATCH alone (emulator built/left at 220x50 fallback while real
        top content assumes 80 cols, fed as a complete consistent stream) renders CLEANLY —
        eliminates simple width-fallback mismatch as a standalone cause; the corruption
        specifically requires an in-flight RESIZE EVENT after header content already exists."
    falsification_test: "Feed real top's actual byte stream through a SINGLE never-resized
      emulator for the full session (Test 3) — if header renders clean (it did), this
      falsifies 'width-fallback-mismatch' as sufficient; only the ACTIVE resize-during-session
      variant (Test 5) reproduces the garble, which supports the resize-churn hypothesis over
      the pure-geometry-mismatch hypothesis."
    fix_rationale: "Not authored in this diagnose-only run — left for the parent/specialist.
      Candidate directions: (1) on any resizeLiveEmulator call, discard-and-rebuild the live
      emulator instead of calling emu.Resize() destructively, since a rebuilt-from-current-PTY
      state is safer than a truncate/pad of stale content; (2) suppress/queue emulator resize
      until the corresponding real-PTY SIGWINCH-triggered redraw frame is observed, so the
      grid dimension change and the content that assumes it land atomically; (3) treat any
      resize event as a signal to re-arm a short 'grace window' where RenderSnapshot prefers
      raw scrollback if it's still intact, rather than immediately trusting the live emulator's
      possibly-transitional state."
    blind_spots: "Did not isolate whether SHRINK alone (no grow-back) or GROW alone (no prior
      shrink) is independently sufficient — only tested the shrink+grow round trip. Did not
      instrument the EXACT byte-level moment the corruption is introduced (i.e., did not
      capture Hub.recordFrame's per-call inputs during the churn window to pinpoint whether
      the emulator resize landed before/after/interleaved with the top-driven redraw frame).
      Did not test whether a single window-resize event (the most common real trigger, e.g.
      xterm.js FitAddon's mount-then-settle double-fit) alone reproduces it without an
      explicit large shrink — the production trigger is plausibly much subtler than my 154->60
      test delta. Did not confirm on the actual deployed prod binary via live UI (relied on
      static/circumstantial confirmation of binary identity, not a live UAT click-through)."
  tdd_checkpoint: n/a — diagnose-only mode (goal: find_root_cause_only), no fix/test written.

## Key code (for the debugger)

- `internal/relay/hub.go`:
  - `recordFrame(frame, raw)` — drain-loop writer; now eager-builds `liveEmu` on first frame
    (commit 8268155c) and feeds every frame under `emuMu`. Strips query seqs via
    `liveEmuQueryStripPattern`.
  - `EnsureLiveEmulator()` — now a safety no-op (builds empty emu only if none exists).
  - `RenderSnapshot()` — `strings.TrimRight(liveEmu.Render(), "\n")`, prefixes
    `altScreenEnterSeq` when `IsAltScreen()`. This is the wrapped-ring reconnect source.
  - `ReconnectPreamble()` — discriminator: raw replay when ring NOT truncated; emulator
    `RenderSnapshot()` when ring wrapped.
  - `Cols()/Rows()` — PTY geometry (fallback 220x50); `resizeLiveEmulator` follows PTY resizes.
- `internal/daemon/engine.go` — `GetSessionStyledTailLines` is the PRECEDENT headless-emulator
  render (emuRows=50, same `queryStripPattern`). Compare its handling to RenderSnapshot's.
- Emulator lib: `github.com/charmbracelet/x/vt`.
- Synthetic test that passes: `TestReconnectPreamble_EagerEmulatorCapturesHeaderBeforeWrap`.

## Eliminated

(none yet)

## Resolution

- root_cause: The eager-build fix (8268155c) is confirmed working for a STATIC geometry (real
  Hub + real macOS top + real 256 KiB ring wrap + never-resized-after-start local viewer
  renders a perfectly clean late-join header). The remaining, empirically-reproduced defect is
  a SEPARATE, additional bug: mid-session PTY/emulator geometry resize churn (a local
  subscriber's cols/rows changing after the header has already been drawn once — e.g. window
  resize, sidebar toggle, font-size change, or xterm.js FitAddon's common mount-then-settle
  double-fit) drives `Hub.resizeLiveEmulator` → `xvt.Emulator.Resize()`, which is a
  DESTRUCTIVE, NON-REFLOW grid truncate/pad (confirmed via `ultraviolet`'s `Buffer.Resize`
  source: shrink drops columns/rows outright via slice truncation; grow pads with blank cells
  — it never reflows or preserves logical row structure). Because `top` only fully
  restructures/redraws its header in direct response to a resize (SIGWINCH) — confirmed real
  macOS top DOES emit a full `ESC[2J` + header rewrite on resize, with a DIFFERENT field
  layout depending on available rows (compact vs full) — any resize transition where the live
  emulator's grid dimensions and the arriving PTY bytes are not perfectly synchronized leaves
  the emulator's header rows in a structurally stale/hybrid state (fields from an intermediate
  geometry's layout persisting on the wrong rows) that is NEVER corrected, because top has no
  reason to believe anything is wrong and never reissues a full redraw absent another resize.
  The process-table BODY is unaffected because it is fully rewritten every ~1s cycle
  regardless of geometry, so any transient corruption there self-heals within one refresh —
  exactly matching the reported "body renders reasonably, header stays garbled" asymmetry.
  SECONDARY (unconfirmed relevance to this exact repro, but a real latent defect found along
  the way): `liveEmuQueryStripPattern` in `recordFrame` is applied per-PTY-Read-chunk rather
  than on the accumulated stream, so a query/DSR escape sequence split across a chunk boundary
  is not stripped and is fed to the emulator literally — proven via a direct unit-style test,
  though no such sequence was observed in real bare-`top` captures (10s + 270s), so it is not
  established as a contributor to M-51 specifically.
- fix: APPLIED (parent context, commit pending). Chose candidate direction (1) but rebuild
  EMPTY, not from a scrollback-tail bootstrap: `resizeLiveEmulator` now discards `h.liveEmu`
  and constructs a fresh `xvt.NewEmulator(cols, rows)` (+ `SetScrollbackSize`) at the new
  geometry instead of calling the destructive in-place `emu.Resize()`. Rationale: the real PTY
  resize (`resizeFn`, invoked immediately after `resizeLiveEmulator` in `ResizeClient`) delivers
  the SIGWINCH that makes a full-screen app fully redraw, so the correct post-resize screen is
  derivable ENTIRELY from the redraw bytes that follow — starting empty lets that redraw
  repopulate a clean screen with zero stale pre-resize rows. Rebuild-from-scrollback-tail was
  REJECTED because a wrapped ring (the M-51 precondition) makes that bootstrap incomplete —
  reintroducing the original garble. The no-resize common case is untouched: recordFrame's eager
  build still feeds ONE emulator continuously, preserving post-wrap completeness (the original
  BUG-04 fix). Candidate (2) synchronize-with-redraw was rejected as fragile (no robust signal
  for "the redraw landed"); candidate (3) raw-replay grace window does not help the exact M-51
  case where the ring has already wrapped (raw is incomplete). SECONDARY per-chunk query-strip
  boundary gap deliberately NOT bundled here (one-change-at-a-time; unconfirmed for M-51; no
  such sequence seen in real bare-top captures) — left as a standalone defense-in-depth
  follow-up.
- fix_files:
  - internal/relay/hub.go — resizeLiveEmulator: discard + rebuild empty at new geometry.
  - internal/relay/scrollback_altscreen_test.go — TestLiveEmulatorResizeDiscardsStaleContent
    (RED→GREEN guard: pre-resize positioned header must NOT survive a resize into the preamble).
- verification: Deterministic RED→GREEN unit test `TestLiveEmulatorResizeDiscardsStaleContent`
  encodes the fix contract (pre-resize positioned header must not survive a resize into
  `RenderSnapshot`): RED with destructive `emu.Resize()` (render was
  `"\x01STALE-HEADER\n\nnew-body-row"` — stale row-1 header survived), GREEN after the rebuild.
  Full gates pass: `go build ./...`, `go vet ./internal/relay/...`, `go test ./internal/relay/...
  -race`, `go test ./internal/webserver/... -race`, full `go test ./...`. All prior
  preamble/altscreen/resize/fanout tests (incl. TestLiveEmulatorFollowsResize — still 40x10
  post-resize) stay green. STILL PENDING: live UAT on the deployed desktop app — share a shell
  running `top`, RESIZE the TerminalPanel / window (or let xterm.js FitAddon's mount-then-settle
  fit occur) while it runs, wrap the ring, late-join a fresh guest, and confirm the full header
  renders cleanly. Record M-51 PASS in 175-UAT.md once confirmed.
- files_changed:
  - internal/relay/hub.go
  - internal/relay/scrollback_altscreen_test.go
