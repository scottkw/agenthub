---
phase: 96
type: human-uat
created: 2026-05-07
requirements: [IMG-01, IMG-02, IMG-03, IMG-04]
plans: [96-01, 96-02, 96-03, 96-04, 96-05, 96-06]
---

# Phase 96 — Human UAT Runbook

Manual verifications for Phase 96 behaviors that automated tests cannot fully
cover: visual fidelity at the renderer layer, multi-client image rendering
consistency, the next-session-only Settings toggle affordance, and tab-OOM
behavior under storage-cap pressure.

Run these AFTER Plans 96-01..96-06 are all complete and the standard test
gates are green:

```bash
go test ./... -count=1
cd frontend && pnpm test
go test -tags=e2e ./internal/webserver/ -run TestBrowserCSP_TerminalImage_NoViolations -count=1
```

The chromedp e2e (Plan 96-06 Task 2) proves NO CSP violations occur when
addon-image's WASM SIXEL decoder bootstraps. The four scenarios below close
the loop on visual + behavioral fidelity that automated tests cannot assert.

Sign off the four scenarios individually; flip 96-VERIFICATION SC-* GREEN per
the validation map in `96-VALIDATION.md`.

---

## Scenario 1 — chafa --format=iterm2 image rendering (IMG-01)

**Why manual:** Visual fidelity at the rendering layer cannot be asserted by
source-scan or unit tests. The chromedp e2e proves NO CSP violations; this
scenario proves the canvas actually paints pixels.

**Setup:**

1. Build the desktop app in production form:
   ```bash
   cd frontend && pnpm install && pnpm run build
   cd .. && wails build -tags wailsassets
   ```
2. Launch the built app from `build/bin/`
   (macOS: `open build/bin/AgentHub.app`).
3. Open a new terminal session.
4. Open the web-served Tailscale URL in a separate browser (Chromium-based).
5. Confirm `chafa` is installed: `chafa --version`
   (`brew install chafa` on macOS, distro equivalent on Linux). If chafa is
   unavailable, fall back to `img2sixel` from `libsixel-bin` or any other
   sixel-emitting CLI; the test is structurally about "the addon paints
   pixels", not chafa specifically.

**Procedure:**

1. Pick a small PNG (e.g., a screenshot, `~/Pictures/chart.png`).
2. In the **desktop terminal**, run:
   ```bash
   chafa --format=iterm2 ~/Pictures/chart.png
   ```
3. Repeat the same command in the **web terminal**.
4. Open browser DevTools → Console on the web terminal.

**Pass criteria:**

- [ ] Desktop terminal renders the image inline, occupying multiple text rows.
- [ ] Web terminal renders the same image inline; visually indistinguishable
      from the desktop render (same colors, same proportions).
- [ ] No browser DevTools console errors mentioning `Content Security Policy`,
      `Refused to`, `WebAssembly`, or `'wasm-unsafe-eval'`.
- [ ] No "ghost" placeholder cells (the image renders fully, not as gray
      boxes) on first paint.

**Fail criteria:**

- Any pass criterion above is not met.
- Any console error containing `'wasm-unsafe-eval'` (means Plan 96-03's CSP
  amendment did not land or was reverted).
- Image renders on desktop but blank/garbled on web (means UMD load order in
  `web/terminal.html` is wrong, or `web/embed.go` did not pick up the new
  asset, or the addon UMD construction silently caught an exception).

---

## Scenario 2 — Two-client mid-stream image join (IMG-04 visual)

**Why manual:** The byte-fidelity unit test (Plan 96-01 Task 2) proves the
relay tier preserves bytes. This scenario proves the second-client RENDERER
paints the same image as the first.

**Setup:**

1. Launch the desktop app.
2. Open the web-served URL in a separate browser tab — **but do not navigate
   to the session yet**.
3. Open a new terminal session in the desktop app.

**Procedure:**

1. In the **desktop terminal**, render an image:
   ```bash
   chafa --format=iterm2 small.png
   ```
   Use a small image — well under the 256 KiB scrollback cap (e.g.,
   100×100 px, ≤30 KiB PNG → much smaller serialized sixel).
2. **Without closing the desktop**, navigate the web browser to the SAME
   session URL (mid-stream join via the relay's ScrollbackSnapshot replay).
3. Observe the web terminal as it loads.
4. Open browser DevTools → Console on the web terminal.

**Pass criteria:**

- [ ] The web terminal, on page load, displays the image rendered by the
      earlier `chafa` invocation as part of its scrollback replay.
- [ ] The image appears identical to the desktop's render (same colors, same
      dimensions in cell-rows).
- [ ] No "ghost" cells; no broken rendering.
- [ ] No CSP / WebAssembly errors in the console.

**Fail criteria:**

- Web terminal shows blank cells where the image should be.
- Web terminal shows different colors or dimensions than desktop.
- Browser console reports CSP violations.

**Known limitation (acceptable, not a regression):** If the image's
serialized escape stream exceeds 256 KiB, the second client may receive only
the tail of the sixel sequence (per RESEARCH §"Pitfall 3: 256 KiB Scrollback
Truncation Cap"). For this scenario, USE A SMALL IMAGE to stay within the
cap. Larger-image multi-client behavior is out of scope for v3.2 (would
touch MC-01..MC-06 multi-client invariants).

---

## Scenario 3 — Settings → Image toggle → next-session-only affordance (IMG-01)

**Why manual:** The italic caption is a visual-only UX affordance; its
presence is asserted by source-scan in `PluginsSection.test.tsx`, but its
behavioral semantics — toggling does NOT re-attach on already-open terminals;
new sessions DO pick up the change — require live interaction.

**Setup:**

1. Launch the desktop app.
2. Open a terminal session **A**.
3. Render an image in session A:
   ```bash
   chafa --format=iterm2 small.png
   ```

**Procedure:**

1. With session A still open and the image visible, navigate to
   **Settings → Plugins**.
2. Verify the **Image** toggle row shows the italic caption
   `Applies to new sessions you create.` directly under the toggle.
3. Toggle **Image OFF**. Press **Save**.
4. Return to session A. Render another image (`chafa ...`).
5. Open a NEW terminal session **B**. Render an image.
6. Toggle **Image ON** again. Press **Save**.
7. In session A, render another image.
8. Open a NEW terminal session **C**. Render an image.

**Pass criteria:**

- [ ] Italic caption `Applies to new sessions you create.` renders directly
      under the **Image** toggle in Settings → Plugins.
- [ ] After Image OFF: session A STILL renders images (next-session-only —
      live toggling does not re-attach the addon on open terminals).
- [ ] After Image OFF: session B does NOT render images (sixel sequences
      appear as printable garbage in the terminal — that's the
      pre-Phase-96 behavior, intentionally preserved when the addon is
      disabled at session-init time).
- [ ] After Image ON again: session A still renders images (no re-attach
      on already-open terminals — same invariant as toggling OFF).
- [ ] After Image ON again: session C renders images (new session picks up
      the toggle).

**Fail criteria:**

- Italic caption missing or has different wording.
- Toggling Image OFF/ON causes session A's image rendering to change live
  (would mean the construction is in the wrong useEffect — Pitfall #1).
- Session B (created when Image=OFF) renders images (would mean the gate
  is missing in the desktop addon-attach path).

---

## Scenario 4 — 50 MB sixel fixture FIFO eviction at 16 MB cap (IMG-02)

**Why manual:** Tab-OOM is a browser-side resource-pressure outcome; the
addon's FIFO eviction is internal to its WASM decoder. Asserting "no tab
OOM" requires live observation in the browser process monitor.

**Setup:**

1. Generate (or locate) a sequence of synthetic sixel inputs whose decoded
   RGBA output sums to ~50 MB. A small Go test helper or shell script can
   emit N raster bands of solid color until the decoded RGBA exceeds 50 MB.
   Example shape: 1024×1024 px solid color = 4 MiB RGBA per band; 13 bands
   = ~52 MiB total decoded. The fixture script is NOT committed to the
   repo — generate at UAT time. A simple recipe:
   ```bash
   # quick-and-dirty: 13 chafa renders of a solid-color PNG
   for i in $(seq 1 13); do
     chafa --format=iterm2 ~/Pictures/solid_1024.png
   done > sixel_fixture.bin
   ```
2. Launch the desktop app.
3. Open a fresh terminal session.

**Procedure:**

1. In the terminal, pipe the 50 MB sixel fixture:
   ```bash
   cat sixel_fixture.bin
   ```
   (or whatever script generates the synthetic stream).
2. Observe the terminal as the fixture renders.
3. Open browser DevTools → Performance / Memory tab (web client) or
   Activity Monitor / `top -pid <pid>` (desktop client) and observe memory
   usage during and after.
4. After the fixture completes, scroll the terminal back through the
   rendered images.

**Pass criteria:**

- [ ] The terminal does not crash, freeze, or report "page unresponsive".
- [ ] Memory usage stabilizes well under any obvious tab-OOM threshold
      (subjective: if memory grows linearly with image count without
      flattening, FIFO eviction is broken).
- [ ] Older images in the scrollback may show as "placeholder" boxes
      (gray cells) — this is the addon's `showPlaceholder: true` upstream
      default for evicted images. EXPECTED behavior, not a bug.
- [ ] Most recent images render fully.

**Fail criteria:**

- Tab crashes, freezes, or shows "Aw, Snap" / "page unresponsive".
- Memory grows unbounded (would mean `storageLimit` is not being enforced).
- All images render fully without any placeholders (would mean
  `storageLimit` is set absurdly high — possibly the upstream 128 MB
  default leaked through because the construction did not pass our
  16 MiB override).

---

## Sign-off

- [ ] Scenario 1 (chafa rendering) — pass on desktop AND web
- [ ] Scenario 2 (multi-client mid-stream byte-fidelity) — pass
- [ ] Scenario 3 (Settings toggle next-session-only affordance) — pass
- [ ] Scenario 4 (50 MB sixel storage cap, no tab OOM) — pass
- [ ] `go test ./... -count=1` exits 0
- [ ] `cd frontend && pnpm test` exits 0
- [ ] `go test -tags=e2e ./internal/webserver/ -run TestBrowserCSP_TerminalImage_NoViolations -count=1`
      exits 0 (or t.Skip on missing Chromium — UAT scenarios cover the gap)

**Approver:** _______________
**Date:** _______________

---

## Known Limitations Documented Here (NOT regressions)

- **256 KiB scrollback cap:** images whose serialized escape streams exceed
  256 KiB are truncated for second-mid-stream-joining clients. Pre-existing
  v3.1 behavior; out of scope for v3.2 (would touch MC-01..MC-06
  multi-client invariants).
- **No image copy/save gestures:** the addon's `getImageAtBufferCell` API
  is intentionally not wired in v3.2. Users can screenshot.
- **Storage cap UI:** the Advanced `<details>` disclosure exposing
  `storageLimit` defers to Phase 99 / PUI-03. Phase 96 ships the daemon
  struct + RPC + a hardcoded 16 MiB default only.
- **Live toggle re-attach for Image:** intentionally NOT implemented (next-
  session-only on both desktop and web). The italic caption in
  PluginsSection is the user-facing affordance for this constraint.
