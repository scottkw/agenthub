---
phase: 157-terminal-screen-share-semantics-issue-109
plan: "03"
subsystem: web-viewer
tags: [javascript, css, xterm, websocket, resize, scale, view-04, view-05]

requires:
  - phase: 157-01
    provides: "Hub.broadcastResize: MsgResize 0x02 fan-out on every host resize"
  - phase: 157-02
    provides: "Server join-push: MsgResize 0x02 before scrollback on both relay and web paths"

provides:
  - "web viewer 0x02 dispatch: term.resize(cols,rows) then recomputeScale() on every server-pushed MsgResize"
  - "recomputeScale(): cell-metric CSS scale (min(cw/gridW, ch/gridH) capped at 1.0) applied to term.element"
  - "Self-sizing removed: no ws.onopen 0x11 send, no term.onResize 0x11 send, no fitAddon.fit() calls"
  - "CSS container model: #terminal overflow:hidden + #terminal .xterm transform-origin:top left"

affects:
  - 157-04 (testing — VIEW-04/VIEW-05 structural gates are now in terminal.js/css)
  - 157-05 (TESTING.md — manual UAT P category item for visual garble + scale verification)

tech-stack:
  added: []
  patterns:
    - "Guest-honor dispatch: 0x02 branch in ws.onmessage; term.resize before recomputeScale (order: scale reads new dims)"
    - "Cell-metric scale: gridW = term.cols * cellW from _renderService.dimensions.css.cell (not getBoundingClientRect, avoids oscillation post-transform)"
    - "Downscale-only cap: s > 1 clamped to 1.0 before apply — VIEW-05 invariant"
    - "FitAddon kept loaded (cell-metric internals accessible) but fit() never called — host owns the grid"

key-files:
  created: []
  modified:
    - web/assets/terminal.js
    - web/assets/terminal.css

key-decisions:
  - "MsgResize = 0x02 const added beside MsgResize2 = 0x11 — single protocol source of truth in the web file"
  - "recomputeScale reads cell dims from term._core._renderService.dimensions.css.cell (same private API as touchScroll block) — not getBoundingClientRect, which oscillates after transform is applied"
  - "term.resize(cols, rows) called BEFORE recomputeScale — scale math requires the new term.cols/term.rows to compute gridW/gridH"
  - "Clamp cols/rows to >= 1 via `|| 1` — xterm rejects resize(0,0); T-157-03 mitigation"
  - "FitAddon loaded but fit() never called — the private render service dimensions are accessible without calling fit()"

requirements-completed: [VIEW-04, VIEW-05]

coverage:
  - id: D1
    description: "web viewer 0x02 dispatch: MsgResize const, payload guard, term.resize + recomputeScale"
    requirement: VIEW-04
    verification:
      - kind: structural
        ref: "node --check web/assets/terminal.js + grep gates (recomputeScale, term.resize, no fitAddon.fit)"
        status: pass
    human_judgment: true
  - id: D2
    description: "recomputeScale: cell-metric scale capped at 1.0, registered on window resize and every 0x02"
    requirement: VIEW-05
    verification:
      - kind: structural
        ref: "grep gates (recomputeScale function, transform-origin, overflow:hidden)"
        status: pass
    human_judgment: true

duration: 2min
completed: 2026-06-27
status: complete
---

# Phase 157 Plan 03: Web Guest Viewer — Honor 0x02 + CSS Scale-to-Fit Summary

**Web viewer now honors server-pushed MsgResize 0x02 (term.resize + cell-metric CSS scale capped at 1.0); all client-driven self-sizing (fitAddon.fit, 0x11 sends) removed; guest never drives the PTY**

## Performance

- **Duration:** 2 min
- **Started:** 2026-06-27T13:14:34Z
- **Completed:** 2026-06-27T13:16:38Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added `MsgResize = 0x02` const beside `MsgResize2 = 0x11` in terminal.js — single protocol source of truth for the server-pushed resize frame.
- Added 0x02 branch in `ws.onmessage`: guards `payload.length >= 4`, decodes big-endian cols/rows, clamps each to `>= 1` (T-157-03 / xterm resize(0,0) protection), calls `term.resize(cols, rows)` then `recomputeScale()` (order is load-bearing: scale reads the new term.cols/rows).
- Added `recomputeScale()`: reads `cellW/cellH` from `term._core._renderService.dimensions.css.cell` (same private API already used by the touchScroll block); computes `gridW = term.cols * cellW`, `gridH = term.rows * cellH`; derives `s = Math.min(cw/gridW, ch/gridH)` capped at `s <= 1.0` (VIEW-05 downscale-only); applies `term.element.style.transform = 'scale(' + s + ')'`.
- Removed all self-sizing: `ws.onopen` initial 0x11 send removed; `term.onResize → 0x11` handler removed; `window.resize → fitAddon.fit()` replaced with `window.resize → recomputeScale()`; initial `fitAddon.fit()` at open removed.
- FitAddon kept loaded (its private render-service internals provide the cell dimensions recomputeScale needs) but `fit()` is never called — host PTY owns the grid.
- Added `overflow: hidden` to `#terminal` CSS rule — clips sub-pixel overhang from the scaled xterm grid.
- Added `#terminal .xterm { transform-origin: top left; }` — pins the scaled grid to the top-left corner so it scales toward bottom-right with no negative offset.

## Task Commits

1. **Task 1: Honor 0x02 resize + recomputeScale in terminal.js; disable self-sizing** - `0b4def19` (feat)
2. **Task 2: terminal.css transform container rules** - `32d81b9f` (feat)

## Files Created/Modified

- `web/assets/terminal.js` — MsgResize const; 0x02 dispatch; recomputeScale(); ws.onopen 0x11 send removed; term.onResize handler removed; window.resize handler replaced; fitAddon.fit() removed
- `web/assets/terminal.css` — `#terminal` gains `overflow: hidden`; new `#terminal .xterm { transform-origin: top left; }`

## Decisions Made

- `recomputeScale` reads cell dims from `term._core._renderService.dimensions.css.cell`, not `getBoundingClientRect` — the bounding rect oscillates after a CSS transform is applied, causing a feedback loop. The private API is stable across all xterm.js v4/v5 versions already used by this project.
- `term.resize(cols, rows)` is called BEFORE `recomputeScale()` — the scale calculation reads `term.cols` and `term.rows`, which are only updated after `term.resize` returns. Reversing the order would scale against the old grid dimensions.
- Cols/rows clamped via `|| 1` — xterm.js throws when either dimension is 0. A server delivering a zero-dim frame would otherwise crash the renderer; the clamp is the T-157-03 mitigation.
- FitAddon kept loaded to preserve cell-metric accessibility. Its internal `_renderService` reference only becomes usable after `term.open()` + first render, which happens before any 0x02 frame arrives (WebSocket opens after term is ready).
- No `term.onResize` handler installed — the handler was the mechanism by which the web guest was previously driving 0x11 sends. Removing it entirely prevents any future code path from accidentally re-enabling client-driven resizes.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## Known Stubs

None — recomputeScale is fully wired: called from 0x02 dispatch and from window resize. The CSS transform-origin and overflow rules are in place. Behavioral proof (host + smaller guest browser, no garble, correct pixel scale) is the manual Category P UAT item tracked in Plan 05.

## Threat Flags

No new security surface beyond the plan's threat model.

- **T-157-03 (Tampering):** `payload.length >= 4` guard + `|| 1` clamp fully implemented. A crafted zero-dim frame from a compromised server cannot crash the renderer.
- **T-157-06 (Usability, accepted):** Pointer hit-test offset under CSS scale is a documented limitation (A3). Not a VIEW-01..05 blocker; noted for UAT.

## Next Phase Readiness

Plan 04 (testing) and Plan 05 (TESTING.md / manual UAT) can rely on:
- `terminal.js` never sends a 0x11 from the web surface
- `recomputeScale()` is the single scale-apply path, registered on both 0x02 and window resize
- CSS container model (overflow:hidden + transform-origin:top left) in terminal.css

---
*Phase: 157-terminal-screen-share-semantics-issue-109*
*Completed: 2026-06-27*

## Self-Check: PASSED

- `web/assets/terminal.js` — FOUND (modified)
- `web/assets/terminal.css` — FOUND (modified)
- Commit `0b4def19` — verified (feat: Task 1)
- Commit `32d81b9f` — verified (feat: Task 2)
- `node --check web/assets/terminal.js` — PASS
- `function recomputeScale` — FOUND in terminal.js
- `term.resize` call in 0x02 dispatch — FOUND in terminal.js
- `fitAddon.fit()` count — 0 (PASS)
- `transform-origin: top left` — FOUND in terminal.css
- `overflow: hidden` on `#terminal` — FOUND in terminal.css
