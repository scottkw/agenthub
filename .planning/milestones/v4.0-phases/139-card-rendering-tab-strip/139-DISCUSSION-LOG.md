# Phase 139: Card Rendering & Tab Strip - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-20
**Phase:** 139-Card Rendering & Tab Strip
**Areas discussed:** VT render location, Preview fidelity, Tab shrink floor, Scroll affordance

---

## VT render location

The user initially asked to clarify the question. A plain-language explanation was given:
the regex ANSI-strip used today produces broken columns / doubled lines / leaked escapes
(#96); the fix is a real headless VT emulator; the open decision is WHERE it runs given local
bytes live in the Go daemon and remote bytes arrive in the browser.

| Option | Description | Selected |
|--------|-------------|----------|
| Go-side for local, JS for remote (split) | Render each where its bytes live; two emulator impls | fallback |
| JS-side unified (@xterm/headless) | One JS impl; Go ships raw bytes; per-card cost on 3s poll | |
| Go-side unified | All gridding in Go incl. relayed remote bytes; zero browser cost; needs daemon to see remote bytes | ✓ |

**User's choice:** Go-side unified (Option C) — **but only after the research step confirms
viability.** If C is not viable (daemon doesn't see remote bytes), fall back to the split (A).
**Notes:** Recorded as a research-gated decision. Go VT library left to research/planner.

---

## Preview fidelity

| Option | Description | Selected |
|--------|-------------|----------|
| Plain text, both | Characters only, no color | |
| Plain mini-preview, colored briefing | Mini stays plain; briefing colored | |
| Colored, both | Color + bold on both surfaces, read from the VT grid | ✓ |

**User's choice:** Colored, both.
**Notes:** Claude default — map colors through the active xterm theme (`ITheme` already threaded),
overridable to a fixed palette. Faithful reproduction of agent output colors does not conflict
with the colorblind-safe rule (no app status meaning encoded by color).

---

## Tab shrink floor

| Option | Description | Selected |
|--------|-------------|----------|
| Shrink to ~80px, keep name | Truncate name; never go below existing 80px min | |
| Shrink smaller, icon-only at floor | Name disappears at the extreme; status dot + close × remain | ✓ |
| Shrink to ~80px, active tab pinned wider | Inactive shrink; active tab exempt | |

**User's choice:** Shrink smaller, icon-only at floor (Chrome favicon-only style).
**Notes:** Drives TAB-03 constraints — rename → context menu, hover tooltip for name, close × +
progress underline must survive at the floor.

---

## Scroll affordance

| Option | Description | Selected |
|--------|-------------|----------|
| Chevron buttons at both ends | ‹ / › shown only on overflow, position-aware | ✓ |
| Visible scrollbar | Un-hide a thin native h-scrollbar | |
| Both chevrons + scrollbar | Belt-and-suspenders | |

**User's choice:** Chevron buttons at both ends.
**Notes:** Keep the current scrollbar hiding; chevrons drive the existing `overflow-x: auto`
`.tab-list`.

---

## Claude's Discretion

- Go VT library selection.
- Exact icon-only pixel floor; whether close × is hover-only at the floor.
- Themed-vs-fixed preview palette (defaulted to themed, overridable).
- Chevron styling, scroll step size, keyboard accessibility.

## Deferred Ideas

None — discussion stayed within phase scope. The broader visual redesign and the
`agenthub-v4.0-redesign/` mockups are already scoped to Phases 140–141.
