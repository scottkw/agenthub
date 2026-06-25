# Phase 140: UI-Spec Gate - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

A **decision gate**, not an implementation phase. Phase 140 produces a **UI spec
artifact** that:

1. Records a chosen redesign direction from `./agenthub-v4.0-redesign` (chosen
   after browser review of the standalone HTML) — **RDS-01**.
2. Reconciles that choice against the already-shipped Hub-first structure, calling
   out and resolving every comp element that conflicts with shipped NAV/SHARE/CARD
   decisions (structural decisions win) — feeds **RDS-03**.
3. Triages the #93 backlog: lists the in-scope subset (assigned to Phase 141) and
   re-defers the rest, updating issue #93 — **CARRY-02**.

**Out of scope:** Actually restyling any surface. All visual implementation is
Phase 141 (RDS-02/03/04, CARRY-01). This phase only chooses + documents.

</domain>

<decisions>
## Implementation Decisions

### Redesign Direction (RDS-01)
- **D-01:** The chosen direction is **Direction 01 — "Refined Native"** (sidebar +
  tabs, comfortable density, flat/native look, tmux-style footer status bar). This
  is the direction the user worked out and built out in Claude Design.
- **D-02:** The **canonical reference is the standalone HTML** —
  `agenthub-v4.0-redesign/AgentHub Redesign (standalone).html`. Phase 141 builds
  against this as the visual source of truth (it is a JS-bundled SPA that renders
  only in a browser).
- **D-03:** Review provenance for success-criterion #1: the user authored/exported
  the design in Claude Design; rendered screens were reviewed via the exported
  `c-*.png` screenshots during this discussion. No additional live-browser pass was
  required to lock the direction. (A live walkthrough remains available in Phase 141
  if needed.)
- **D-04:** No cross-direction mix. Directions 02 (Command Workspace / violet /
  ⌘K / share rail) and 03 (Mission Control / gradient / glassy card deck) are **not**
  adopted, in whole or in part.

### Accent & Visual Feel
- **D-05:** **Keep the current blue accent.** The standalone's CSS accent var
  `#7C8CFF` (periwinkle violet) is treated as **incidental and rejected** — Phase 141
  does NOT change the accent color. Canonical blue tokens to preserve:
  `--hub-accent: #7aa2f7` (dark) / `#3d6fe8` (light, WCAG AA verified on #ffffff) in
  `frontend/src/style.css`. (User is colorblind — accent must be verified at the hex
  level, never by eye; see [[user_colorblind]].)
- **D-06:** Visual feel: comfortable density, **flat/native** (not glassy/rounded —
  that was Direction 03), tmux-style footer status bar, both light + dark themes.
- **D-07:** Phase 141 must honor `prefers-reduced-motion` and colorblind-safe
  semantics throughout (RDS-04) — carry-forward constraint, not re-litigated here.

### Conflict Reconciliation (structural decisions win — RDS-03)
The standalone was built on the **pre-Hub-first** navigation. The following conflicts
are resolved as locked; the spec MUST call each out:

- **D-08 (sidebar):** Comp shows `Home · Remote · Sessions · + New Session · Settings`.
  Shipped/winning structure is **`Home · Hub · Settings`** (Phase 138, NAV-02/03/04/05).
  Drop the Remote, Sessions, and `+ New Session` sidebar entries. Session creation
  stays on the Hub's `HubFilterBar` "New Session" button (CARD-01).
- **D-09 (Sessions page):** The comp's per-session management page (`c-sessions.png`:
  Web on/off, Browse files, Kill, Read-Only / Full-Access links, Copy/Open/QR,
  file-write/edit toggles) is **NOT reintroduced as a page**. Its controls already
  live on Hub cards + per-card overflow (Kill, Browse files — Phase 138) and the
  share modal (Phase 137 cap model).
- **D-10 (Remote page):** **NOT reintroduced.** Remote sessions live in the unified
  local+remote Hub grid; remote provenance is shown on cards (CARD-02, colorblind-safe).
- **D-11 (share copy):** The comp footer "Share links are on the Sessions tab" must be
  reworded/removed — there is no Sessions tab; sharing is via the Hub / share modal.
- **D-12 (Hub has no comp):** The standalone contains **no Hub surface** (it predates
  the Hub). Phase 141 derives the Hub's restyle by **applying Refined Native's visual
  language** (typography, spacing, blue accent, card treatment) to the existing
  shipped Hub — there is no pixel comp to copy. Spec must flag this explicitly.

### Restyle Depth
- **D-13:** **Recolor only — keep current UX.** Phase 141 applies Refined Native's
  color / typography / spacing to the **existing share dialog and Hub cards** but does
  NOT change their layout or interaction model. The comp's polished Sessions-page
  share-control layout is NOT harvested; the shipped Phase 137 share model and shipped
  card interactions are preserved. Lowest-risk instruction to downstream.

### #93 Backlog Triage (CARRY-02)
- **D-14:** **Re-defer all 7 deferred #78 items; keep #93 open.** None fit a restyle
  milestone — each is a net-new feature needing backend data or a new surface
  (usage metrics on cards [overlaps #67], formal projects model, member/collaborator
  avatars+presence, structured "agent suggests" briefings, session-detail/chat page,
  tweaks panel, pixel-faithful #78 port). **In-scope subset for Phase 141: none.**
- **D-15:** Updating issue #93 with the triage outcome (a GitHub comment recording
  "v4.0 triage: all re-deferred") is an **execution action** for the Phase 140 plan —
  not done during discussion. See [[reference_github_issues_release_planning]].

### Claude's Discretion
- Exact wording/structure and file location of the UI spec artifact itself (the plan
  decides; a natural home is `.planning/phases/140-ui-spec-gate/140-UI-SPEC.md` or the
  Phase 141 UI-spec produced by `/gsd:ui-phase`).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Redesign comps (the source material)
- `agenthub-v4.0-redesign/AgentHub Redesign (standalone).html` — **canonical chosen
  direction** (Direction 01 Refined Native, built out). JS-bundled SPA; renders only
  in a browser. Accent var `#7C8CFF` here is REJECTED (see D-05).
- `agenthub-v4.0-redesign/AgentHub UI redesign/AgentHub Redesign Directions.dc.html` —
  the "Three directions" chooser (01 Refined Native / 02 Command Workspace /
  03 Mission Control). Source for what was NOT chosen.
- `agenthub-v4.0-redesign/AgentHub UI redesign/c-welcome.png`, `c-session.png`,
  `c-settings.png`, `c-filebrowser.png`, `c-sessions.png`, `c-remote.png` — rendered
  screens of the standalone (note: these PNGs render the accent as blue; the live
  standalone CSS is violet — trust the hex tokens, not the PNG color).

### Structural decisions that WIN conflicts (RDS-03)
- `.planning/phases/138-hub-first-navigation/138-CONTEXT.md` — NAV (Home/Hub/Settings
  sidebar; Sessions+Remote pages deleted; creation on Hub) + CARD decisions.
- `.planning/phases/137-share-modal-cap-model/137-CONTEXT.md` — SHARE / share-modal
  cap model (read-only vs writer, QR, links) that supersedes the comp's Sessions-page
  share controls.
- `.planning/phases/139-card-rendering-tab-strip/139-CONTEXT.md` — card mini-preview /
  tab-strip rendering already shipped.

### Requirements / roadmap
- `.planning/ROADMAP.md` §"Phase 140" / §"Phase 141" — goal, success criteria.
- `.planning/REQUIREMENTS.md` — RDS-01, CARRY-02 (this phase); RDS-02/03/04, CARRY-01
  (Phase 141).

### GitHub issues
- Issue **#93** — the deferred #78 fidelity backlog to triage + update (CARRY-02).
- Issue **#97** — Hub GroupSidebar ARIA (CARRY-01) — **Phase 141**, not this phase,
  but relevant context.

### Codebase anchors
- `frontend/src/style.css` — `--hub-accent` blue tokens (#7aa2f7 dark / #3d6fe8 light)
  to preserve (D-05).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Shipped Hub surface (`frontend/src/components/Hub/*`, incl. `HubFilterBar`,
  `SessionCard`) — the restyle target; no new structure needed.
- Phase 137 share modal — preserved as-is, recolored only (D-13).
- `--hub-accent` token pair already theme-split (dark/light) and WCAG-verified.

### Established Patterns
- Hub-first navigation (Home/Hub/Settings) is shipped and is the structural anchor;
  the redesign restyles within it, never reintroduces removed pages.
- Colorblind-safe semantics: status/provenance conveyed by icon/label/badge, never
  color alone (CARD-02/03).

### Integration Points
- Phase 141 will apply the chosen visual language to the surviving surfaces
  (Welcome→Home, Hub, terminal/session, File Browser, Editor, Settings).

</code_context>

<specifics>
## Specific Ideas

- The redesign "look" the user wants is concretely captured in the standalone HTML —
  downstream should treat it as the literal visual target for Welcome/session/
  FileBrowser/Editor/Settings, and DERIVE the Hub's look from the same language
  (no Hub comp exists).
- Welcome screen in the comp maps to the shipped **Home** sidebar item.

</specifics>

<deferred>
## Deferred Ideas

All 7 #93 / #78 fidelity items are re-deferred (kept on issue #93), each a candidate
for its own future feature phase — not v4.0:
- Per-session usage metrics on cards (tokens/spend/context%) — overlaps #67.
- Formal "projects" model (code/color/desc) replacing working-dir grouping.
- Member/collaborator avatars + presence.
- Structured "agent suggests" briefings (structured decision data).
- Session detail / chat-thread page.
- Tweaks panel (live density/modal-layout/accent editing).
- Pixel-faithful #78 port (conflicts with Hub-first + "adapt the look").

Directions 02 (Command Workspace) and 03 (Mission Control) are not adopted; their
distinctive elements (⌘K command bar, live share rail, gradient/glassy card deck)
are available as inspiration for future phases but out of v4.0 scope.

</deferred>

---

*Phase: 140-ui-spec-gate*
*Context gathered: 2026-06-20*
