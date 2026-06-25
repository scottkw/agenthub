# Phase 139: Card Rendering & Tab Strip - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Two independent GUI concerns under the v4.0 Hub-First overhaul:

1. **CARD-05 (#96) — Headless VT rendering.** Replace the regex ANSI-strip used for
   mini-preview cards and the briefing-modal tail with a real headless VT emulator, so
   agent (Claude Code / TUI-style) output renders with correct column spacing and zero
   leaked escape sequences. The mini-preview and the briefing-modal tail must share one
   rendering path (success criteria 1 + 2).
2. **TAB-01..03 — Browser-style tab strip.** Open tabs shrink as their count grows; when
   they overflow even at the minimum width, a visible scroll affordance reaches every tab;
   close / rename / progress-underline affordances keep working at the minimum width.

**In scope:** CARD-05, TAB-01, TAB-02, TAB-03.
**Not in scope:** the RDS redesign (Phases 140–141), the standalone HTML mockups in
`agenthub-v4.0-redesign/` (those feed the UI-Spec gate in Phase 140), CARD-01..04
(done in Phase 138).
</domain>

<decisions>
## Implementation Decisions

### VT render location (CARD-05)
- **D-01 (research-gated):** Target architecture is **Go-side unified** — do all headless
  VT gridding in the daemon (a Go VT library) for BOTH local and remote sessions, so the
  browser always receives a clean grid as text. Local already has raw scrollback in the
  daemon ring buffer; remote bytes are relayed through the local daemon, so render them
  daemon-side in the relay path too.
  - **Hard gate:** the planner's research step (`gsd-phase-researcher`) MUST verify that the
    local daemon actually sees the remote peer's PTY bytes before they reach the browser.
    Only commit to Go-side-unified if that is confirmed.
  - **Fallback (D-01a):** if the daemon does NOT see remote bytes, fall back to a **split**:
    render local scrollback in Go (inside/replacing `GetSessionTailLines`), render remote
    bytes JS-side in the browser. Both surfaces (mini-preview + briefing) consume whichever
    per-side output applies.
  - **AVOID:** JS-only-unified was rejected — it would force every Hub card to run a headless
    emulator on the shared 3s poll, reversing `MiniPreview`'s deliberate "never mount an xterm
    instance" design (`MiniPreview.tsx:1`, CARD-07).
  - The specific Go VT library (e.g. vt10x / vt100 / hinshun) is a research/planner choice —
    not locked here. Whatever is chosen, the mini-preview and briefing-modal tail MUST go
    through the same path.

### Preview fidelity (CARD-05)
- **D-02:** Render **color + bold** (full styling), on **both** surfaces — mini-preview and
  briefing tail — read out of the VT grid cells, not just the plain characters. Most faithful
  to the real terminal.
- **D-03 (Claude's discretion default):** Map the grid's colors through the **active xterm
  theme** (`ITheme`, already threaded down to `HubPanel` / `HubInteractiveModal`), not a fixed
  ANSI-16 palette — so previews match the interactive terminal. The card's existing
  dim-on-stopped-ok opacity still applies on top of the colored render. User may override to a
  fixed palette if the themed approach looks wrong in review.
- **D-04:** Note for accessibility — this is faithful reproduction of the agent's OWN output
  colors, NOT app-level status encoding. It does not violate the colorblind-safe rule (no app
  meaning is conveyed by color alone here). Status/origin/connection indicators added in
  Phase 138 remain icon/label-based.

### Tab shrink floor (TAB-01)
- **D-05:** Tabs **flex-shrink** from the current 180px max DOWN PAST the current 80px min to an
  **icon-only floor** — at the extreme the name disappears entirely, leaving status dot + close
  ×, Chrome "favicon-only" style. Only once tabs hit that floor does the strip scroll.
  (Today `.tab` is `flex-shrink: 0; min-width: 80px` — `style.css:108` — so tabs do not shrink
  at all; this must change.)
- The exact pixel floor is a UI-Spec / planner detail (must fit status dot + close × legibly).

### TAB-03 affordances at the floor
- **D-06:** Because the name vanishes at the icon-only floor, **rename falls back to the
  right-click context menu** (already implemented in `TabBar.tsx` — `tab__context-menu`). No
  double-click-name target exists at the floor.
- **D-07:** Each tab needs a **hover tooltip / `title`** showing its full name, since icon-only
  tabs look near-identical.
- **D-08:** The **close ×** and the **Phase 98 progress underline** (`tab__progress`,
  `TabBar.tsx:245`) MUST remain present and functional at the icon-only floor. Whether the ×
  is hover-only at the floor (to save space) is a UI-Spec / planner detail.

### Scroll affordance (TAB-02)
- **D-09:** Overflow affordance is **chevron buttons at both ends** (‹ / ›). They appear ONLY
  when the strip overflows, and are **scroll-position-aware** (hide ‹ at the start, hide › at
  the end). Clicking scrolls toward the hidden tabs. No raw/native scrollbar is exposed — the
  current `scrollbar-width: none` / `::-webkit-scrollbar { display:none }` hiding
  (`style.css:104`) stays; chevrons drive the existing `overflow-x: auto` `.tab-list`.

### Claude's Discretion
- Go VT library selection (D-01).
- Exact icon-only pixel floor and whether close × is hover-only at the floor (D-05, D-08).
- Themed-vs-fixed palette is defaulted to themed but overridable (D-03).
- Chevron styling, scroll step size, and keyboard accessibility of the chevrons.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap
- `.planning/REQUIREMENTS.md` — CARD-05 (line 33), TAB-01..03 (lines 44–46); status matrix lines 110–113.
- `.planning/ROADMAP.md` §"Phase 139: Card Rendering & Tab Strip" — goal + 5 success criteria.
- GitHub issue **#96** (scottkw/agenthub) — the source bug for CARD-05 (TUI output illegibility / leaked escapes).

### Prior phase context (carried forward)
- `.planning/phases/138-hub-first-navigation/138-CONTEXT.md` — CARD-04 card scope (Share button,
  origin/connection indicators, overflow menu) that the mini-preview lives alongside; CARD-07
  note that MiniPreview never mounts xterm.

### Code the planner must read
- `frontend/src/components/Hub/MiniPreview.tsx` — current plain-text snapshot render (consumer of
  the VT output); CARD-07 "no xterm instance / 3s shared poll" constraint.
- `frontend/src/components/Hub/HubBriefingModal.tsx` — current `stripAnsi` regex (lines 18–43) to
  be replaced; documents the local (Go-stripped) vs remote (client-stripped) byte-source split.
- `internal/daemon/engine.go` — `GetSessionTailLines` (the local tail producer; raw scrollback →
  regex strip today). Unit tests at `internal/daemon/engine_test.go:1580+`.
- `frontend/src/components/TabBar.tsx` — tab strip component (rename/close/context-menu/progress
  underline live here).
- `frontend/src/style.css` — `.tab-bar` / `.tab-list` / `.tab` rules (lines 82–135); scrollbar
  hiding at lines 104–106.

### Already-present deps (reuse before adding)
- `frontend/package.json` — `@xterm/xterm@^6`, `@xterm/addon-serialize@^0.14` ALREADY installed
  (relevant only to the rejected JS-unified path / the split fallback's remote side).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `GetSessionTailLines` (engine.go) — already extracts local scrollback; the Go-side VT render
  replaces its regex strip rather than adding a new pipeline.
- `@xterm/addon-serialize` — already a dependency; available if the split-fallback remote side
  needs JS-side gridding.
- `TabBar.tsx` context menu (`tab__context-menu`) — already provides rename; satisfies the
  icon-only rename fallback (D-06) with no new UI.
- `tab__progress` underline (Phase 98) — existing affordance to preserve at the floor (D-08).

### Established Patterns
- MiniPreview is `aria-hidden` decorative plain text on a **shared 3s poll** (`usePreviewPoller`),
  deliberately xterm-free — this is why per-card JS emulation was rejected (D-01 AVOID).
- Local-vs-remote byte-source split is a recurring architectural seam: local is daemon-side,
  remote is relayed to the browser. The VT decision (D-01) hinges on whether the daemon sees
  remote bytes.
- Card preview color must coexist with the existing dim-on-stopped-ok opacity and attention
  pulse / float-to-top (CARD-04, Phase 138).

### Integration Points
- `ITheme` is already threaded App → `HubPanel` → `HubInteractiveModal`; D-03 reuses it for
  preview colorization.
- `.tab-list` already has `overflow-x: auto`; chevrons (D-09) drive that existing scroll
  container rather than introducing new scroll mechanics.

</code_context>

<specifics>
## Specific Ideas

- "Browser-style" tabs explicitly means Chrome behavior: shrink-then-scroll, favicon-only at the
  extreme, end chevrons for overflow.
- VT rendering should be a real emulator producing a grid — NOT a smarter regex. #96's complaint
  is specifically about regex-strip artifacts (doubled lines, broken columns, leaked codes).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. (The broader visual redesign and the
`agenthub-v4.0-redesign/` mockups are already scoped to Phases 140–141.)

</deferred>

---

*Phase: 139-Card Rendering & Tab Strip*
*Context gathered: 2026-06-20*
