# Phase 157: Terminal Screen-Share Semantics (Issue #109) - Context

**Gathered:** 2026-06-27
**Status:** Ready for planning
**Source:** Inline decision capture (open decisions from Issue #109 thread, resolved with user 2026-06-27)

<domain>
## Phase Boundary

Eliminate the cross-viewer PTY grid-size garble (overlapping/doubled/rewrapped characters
when viewers have different window sizes) by adopting **"screen-share semantics" (Option B,
locked in Issue #109)**: the host's terminal is the single source of truth for the PTY grid;
every other connected viewer renders **that same grid** and reconciles to its own viewport by
**CSS pixel-scaling (downscale-only)**, never by re-gridding. Covers the full Option B scope —
all six change-layers — across relay/webserver PTY arbitration and the web + desktop terminal
viewers.

**The garble invariant (why this works):** Raw PTY bytes carry grid-specific wrap points and
absolute cursor moves. Rendering them in a *different*-sized grid collides (garble) or re-flows
(wrap). The single cure is that **every connected viewer renders the exact PTY grid the bytes
were laid out for.** Viewports are reconciled by scaling *pixels* (`transform: scale`), which
shrinks the picture, not the grid — so wrapping is byte-for-byte identical to the host and no
one garbles. Arbitration (which grid is "the one") is therefore a content-authority decision,
**orthogonal to garble-freedom**.

**In scope (Phase 157):**
- **Host-authority PTY arbitration** replacing MC-06 max-wins (`internal/relay/hub.go`
  `ResizeClient`): PTY grid tracks the host (local-origin) subscriber; web/remote-origin
  `MsgResize` is ignored by the arbiter.
- On host resize, broadcast `MakeResizeFrame(ptyCols,ptyRows)` to web guests.
- On guest join, push the host's current grid (`MakeResizeFrame(hub.Cols(),hub.Rows())`)
  **before** scrollback replay (fixes the static-screen garble case).
- Web viewer (`web/assets/terminal.js`) + desktop viewer
  (`frontend/src/components/TerminalPanel.tsx`) honor server-pushed `MsgResize` (0x02) →
  `term.resize(cols,rows)`, stop self-sizing, and CSS-downscale the host grid to fit
  (`s = min(containerW/gridW, containerH/gridH)`, capped at `s ≤ 1.0`, recomputed on window
  resize and every `MsgResize`). `web/assets/terminal.css` + desktop parity.
- Replace MC-06 max-wins hub tests with host-authority tests; update TESTING.md suite manifest
  + traceability map for every new/changed test file (standing rule).

**Out of scope / explicitly designed out:**
- Any change to guest **input** semantics — this is a **sizing-only** phase (see decisions).
- Upscaling a guest viewport above the host grid (cap `s ≤ 1.0` — never upscale).
- Disturbing the host view (the host renders natively; it never scales — except the rare
  multi-local-host case, see D-02).

</domain>

<decisions>
## Implementation Decisions

### Core model (locked upstream in Issue #109 — Option B)
- Host (local origin) is the single source of truth for the PTY grid.
- Guests `term.resize()` to the host grid and CSS-**downscale** to fit; cap `s ≤ 1.0`
  (downscale-only, never upscale).
- Host view is never disturbed (renders natively at the PTY grid). Accepted tradeoff: a guest
  on a much smaller viewport gets small text (screen-share readability tradeoff — correct for
  presenter semantics; prefer a crisp renderer / sensible scale so downscaled text isn't also
  blurry).

### D-01 — Host-not-connected behavior
**Freeze last host size.** When no local (host) subscriber is driving the PTY, retain the most
recent host-reported `(cols,rows)` until a host reconnects. Avoids a reflow jump and matches
"host is source of truth" continuity. (Not a fixed 120×32 default.)

### D-02 — Multiple local (host-origin) subscribers
**Smallest-among-local arbitration** — PTY grid = `min` of host-origin subscriber sizes
(swap the existing `max` for `min`, scoped to **local origin only**). Rationale: local hosts
render natively (not scaled), so `max` would garble the smaller local host (mini-#109) and
latest-wins could garble the non-latest one; `min` guarantees the grid fits inside every native
local viewport — no local host ever garbles; the larger one simply gets padding. Keeps the host
pixel-pristine and does NOT relax "host view never disturbed." Same complexity as the max code
being replaced — just `min`. (Common case is a single local host, where this is moot.)

### D-03 — Guest input model (sizing-only scope)
**Leave existing guest input behavior exactly as-is** — no new read-write model added, nothing
removed. None of VIEW-01..05 require guest input; the phase is scoped strictly to the grid-size
garble fix. The one invariant this phase adds: **guests never drive PTY resize** (web-origin
`MsgResize` is ignored). Whatever input path exists today (`MsgInput`/inject) is untouched.

### Claude's Discretion
- Exact scale-recompute hook wiring (ResizeObserver vs. window resize listener), CSS transform
  origin, and container layout for the scaled grid — implement to the cleanest result that
  honors `s ≤ 1.0` and no host disturbance.
- Test structure/naming for the host-authority hub tests, within the TESTING.md convention.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Issue & requirements
- GitHub Issue #109 (scottkw/agenthub) — Option B decision + the garble screenshot scenario.
- `.planning/REQUIREMENTS.md` — VIEW-01..05 (lines ~69–73).
- `.planning/ROADMAP.md` — Phase 157 section (goal, six success criteria).

### Code touchpoints (verify exact symbols during planning)
- `internal/relay/hub.go` — `ResizeClient` (MC-06 max-wins arbiter to replace), `Cols()`,
  `Rows()`, `Resize`, `Subscriber.Cols/Rows`, `ptyCols/ptyRows`.
- `internal/webserver/server.go`, `internal/relay/server.go` — web-origin `MsgResize` handling.
- `web/assets/terminal.js`, `web/assets/terminal.css` — web guest viewer + scaling.
- `frontend/src/components/TerminalPanel.tsx` — desktop guest viewer (cross-surface parity).
- Frame helpers: `MakeResizeFrame`, `MsgResize` (0x02) — locate the relay frame package.

### Standing conventions
- `TESTING.md` — Section 2 suite manifest, Section 4 traceability map, Section 5 manual
  checklist; run `bash tests/check-traceability-paths.sh` before committing (repo CLAUDE.md).

</canonical_refs>

<specifics>
## Specific Ideas

- Success criterion 5 (cross-surface parity): desktop `TerminalPanel` must achieve guest-path
  parity, OR the gap is explicitly documented IF hosts are always desktop / guests always web.
  Parity is a release-blocking project rule — do not silently defer.
- Success criterion 1 is verified against the **Issue #109 screenshot scenario** (host + a
  smaller-windowed guest → no overlapping/doubled characters).

</specifics>

<deferred>
## Deferred Ideas

- Collaborative read-write guest typing as a *new* designed feature (D-03 keeps existing
  behavior; a deliberate RW model is a future phase if wanted).
- Relaxing "host view never disturbed" so local hosts also downscale (the alternative to D-02's
  smallest-among-local) — not adopted; revisit only if multi-local-host garble surfaces.

</deferred>

---

*Phase: 157-terminal-screen-share-semantics-issue-109*
*Context captured: 2026-06-27 via inline decision capture (/gsd-plan-phase)*
