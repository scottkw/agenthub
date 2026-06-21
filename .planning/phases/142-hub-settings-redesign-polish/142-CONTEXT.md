# Phase 142: Hub & Settings Redesign Polish - Context

**Gathered:** 2026-06-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Resolve the five post-redesign UAT findings raised at the Phase 141-09
blocking checkpoint (`141-RENDER-COMPARE.md` §"Newly discovered follow-up
items"), captured as **POL-01..POL-05** in REQUIREMENTS.md. The redesign visual
language (Direction 01 "Refined Native") has landed; this phase fixes the
interaction/IA/repaint defects on top of it.

Concretely:
- **POL-01** — Hub card header icons (⋮ menu, ☰ handle) must not overlap card
  content at any width; the in-card mini-preview must be sized to be legible.
- **POL-02** — Settings → Appearance Light/Dark becomes a single slider/toggle
  switch (not two buttons), keeping persistence + colorblind-safe state.
- **POL-03** — Both Hub "New session" buttons (top-right + empty-state) styled
  to match the comp's sidebar "New Session" affordance.
- **POL-04** — A terminal session must repaint cleanly (no garble) after a theme
  switch or tab switch; root cause confirmed/fixed.
- **POL-05** — Hub group navigation restructured out of the secondary
  side-by-side panel; no two collapsible side panels sit side by side.

**Not in scope:** new Hub capabilities; re-introducing Sessions/Remote sidebar
pages or a "+ New Session" sidebar item; changing the locked accent or Direction
01 visual language; CodeMirror internal theme (`--cm-*`); the formal regression
program (that is Phase 143, TEST-01..05).

</domain>

<decisions>
## Implementation Decisions

### POL-05 — Group navigation pattern
- **D-01:** Move group navigation into the **main left sidebar (Home / Hub /
  Settings)** as an **expandable sub-list nested under the "Hub" item**.
  Selecting a group opens the Hub filtered to that group; the session grid then
  spans full width with no second collapsible panel beside it. The current
  side-by-side `GroupSidebar` panel (`HubPanel.tsx` `hub__body` flex row) is
  removed.
- **D-02:** The comp **does not depict a Groups concept at all** (the standalone
  comp predates the Hub-first restructure and shows the dropped Sessions/Remote
  pages). "Per the comp" is therefore aspirational — the nested-sidebar pattern
  is the agreed structure and the comp's *visual language* (Direction 01) is
  applied to it, not a literal comp region to copy.
- **D-03 (flag for researcher):** Preserve the existing **drag-to-assign**
  behavior (`onDropOnGroup` → `assignToGroup`/`removeFromGroup` in
  `HubPanel.tsx`/`GroupSidebar.tsx`). With groups in the main sidebar, the
  preferred path is dropping a session card onto a sidebar group item. If
  dragging a card across to the main sidebar proves infeasible/awkward, fall
  back to a per-card menu ("Add to group…") — researcher to recommend. Group
  data model (`lib/hubGroups.ts`: `HubGroupDef {id,name,memberKeys}`, localStorage
  `agenthub:hubGroups:v1`) and live running/total/attention counts must survive
  the move.

### POL-01 — Card icons + preview sizing
- **D-04:** Give the mini-preview a **taller fixed height** (more terminal lines
  visible — target ~6 lines, planner to confirm against card grid density) so it
  is legible/useful, and **reserve a dedicated header gutter** so ⋮ (menu) and ☰
  (handle) never overlap the session name / status / preview at any card width.
  Reflow at narrow widths must keep both icons clear of content.

### POL-04 — Terminal garble (theme/tab switch)
- **D-05:** **Harden the whole repaint path**, not just the minimal regression
  patch. Researcher first confirms the exact 141-08 regression source, then the
  fix systematically addresses theme/tab/font repaint coordination. Scout
  identified concrete suspects in `TerminalPanel.tsx`:
  - the `theme`-change effect (~line 696–704: `options.theme` +
    `clearTextureAtlas()` + `refresh()`) has **no `isActive` guard** — it can run
    on a `display:none` panel with zero cell dims;
  - **no `fitTerminal()` re-call** after a theme repaint (stale cell layout if
    font metrics changed);
  - **tab-switch ↔ theme-change race** between the `isActive` rAF-based fit
    effect (~647–687) and the immediate `theme` `refresh()`;
  - WebGL `clearTextureAtlas()` timing on `display:none → flex` transitions.
  Verify the fix with a real terminal in the **native `wails dev` window** — the
  `:34115` bridge has no PTY (per `141-RENDER-COMPARE.md` headless limitation).

### POL-02 / POL-03 — Controls (locked, sensible defaults)
- **D-06:** Light/Dark toggle is a single switch whose state is indicated by a
  **sun/moon icon + text label on/with the knob**, not by color or knob position
  alone (colorblind-safe — owner is colorblind). Keep the existing `uiTheme`
  persistence + `[data-ui-theme]` wiring (`App.tsx` ~282–290;
  `SettingsTab.tsx`). No two-button layout.
- **D-07:** Both "New session" buttons (HubFilterBar top-right + HubEmptyState)
  are restyled to match the comp's sidebar "+ New Session" affordance (see
  `c-sessions.png` left sidebar). Same affordance on both; existing onClick
  wiring unchanged.

### Claude's Discretion
- Exact preview pixel height and grid reflow breakpoints (planner's call,
  bounded by D-04's "legible/~6 lines" intent).
- Plan-splitting across the five POL items, and migration ordering.
- Exact CSS token / class names, following the existing `--hub-*` convention
  and the light-theme override discipline (every rule needs its
  `[data-ui-theme="light"]` counterpart).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### UAT findings that define this phase (read first)
- `.planning/phases/141-redesign-implementation/141-RENDER-COMPARE.md` §"Newly
  discovered follow-up items" — the authoritative source of POL-01..05.

### Design contract (LOCKED — inherited from Phase 141)
- `.planning/phases/141-redesign-implementation/141-UI-SPEC.md` — Direction 01
  lock, accent locks (`#7aa2f7` dark / `#3d6fe8` light), `--hub-*` token system,
  colorblind + motion contracts, copywriting contract, Phase Constraints.
- `.planning/phases/140-ui-spec-gate/140-UI-SPEC.md` — upstream decision gate
  (D-05 accent rejection, structural fences, recolor-only scope).

### Canonical visual reference
- `agenthub-v4.0-redesign/AgentHub Redesign (standalone).html` — open in a
  browser; Direction 01 visual reference. **Caveats:** accent `#7C8CFF` is
  REJECTED (use `#7aa2f7`/`#3d6fe8`); the comp predates the Hub-first restructure
  (shows dropped Sessions/Remote pages, no Groups concept).
- `agenthub-v4.0-redesign/AgentHub UI redesign/c-sessions.png` — shows the
  comp's left-sidebar "+ New Session" affordance (POL-03 styling target).
- `agenthub-v4.0-redesign/AgentHub UI redesign/screenshots/141-09/02-hub-dark.png`
  — the current Hub with the side-by-side GROUPS panel (POL-05 "before").

### Implementation source of truth
- `frontend/src/components/Hub/SessionCard.tsx` + `MiniPreview.tsx` — POL-01
  card header icons + preview.
- `frontend/src/components/Hub/HubPanel.tsx` (`hub__body` flex row, group state
  ~259–286, render ~508–519) + `GroupSidebar.tsx` + `lib/hubGroups.ts` — POL-05.
- `frontend/src/components/Sidebar.tsx` (Hub button ~60–66) + `App.tsx`
  (`HUB_TAB`, `onOpenHub`) — POL-05 destination for nested groups.
- `frontend/src/components/Hub/HubFilterBar.tsx` + `HubEmptyState.tsx` — POL-03
  New session buttons.
- `frontend/src/components/SettingsTab.tsx` + `App.tsx` ~282–290 (`uiTheme` /
  `[data-ui-theme]` effect) — POL-02 Light/Dark toggle.
- `frontend/src/components/TerminalPanel.tsx` (theme effect ~696–704, isActive
  fit effect ~647–687) + `App.tsx` ~1492 (`display:none/flex` tab gating) —
  POL-04 repaint path.
- `frontend/src/style.css` — `--hub-*` tokens (~3896–3995, dark + light blocks);
  add light-theme overrides for every new rule.

### Requirements / scope
- `.planning/REQUIREMENTS.md` — POL-01..POL-05 (lines 66–70).
- `.planning/ROADMAP.md` §Phase 142 — success criteria.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `--hub-*` token system + Direction 01 styling already applied across surfaces
  (Phase 141) — POL work extends/retunes, does not re-theme.
- `@heroicons/react 24/outline` (installed) — sun/moon icons for the POL-02
  toggle; no new icon dependency.
- `Sidebar.tsx` collapse pattern (localStorage `hub-sidebar` key) and
  `GroupSidebar.tsx` collapse pattern (`hub-group-sidebar-collapsed`) — the
  nested-group sub-list should reuse the main sidebar's idioms, and the
  group-panel collapse state likely becomes obsolete.
- `lib/hubGroups.ts` CRUD (`loadGroups/createGroup/assignToGroup/removeFromGroup`)
  and per-group count computation — reuse intact; only the *render location*
  moves.

### Established Patterns
- Theme switching is `[data-ui-theme="light"]` attribute-conditioned CSS; every
  migrated/new rule needs its matching light-theme override.
- `prefers-reduced-motion`: animations gated under
  `@media (prefers-reduced-motion: no-preference)` with static `reduce` fallbacks.
- Colorblind verification at the **hex-constant level in source** (grep), never
  by eye — owner is colorblind. State must be conveyed by icon + text, not color
  or position alone (drives POL-02 D-06).

### Integration Points
- POL-05 moves group selection state from `HubPanel` into / coordinated with the
  main `Sidebar` + `App` tab routing; `activeGroupId` filtering of
  `visibleSessions` must continue to drive the grid.
- POL-04 fix lives entirely in `TerminalPanel.tsx` effect coordination; verify
  natively (no PTY over the `:34115` bridge).

</code_context>

<specifics>
## Specific Ideas

- POL-05: nested expandable group sub-list under the "Hub" sidebar item (user
  selected the sidebar-nest layout over a FilterBar group-chip row).
- POL-01: taller fixed preview (~6 lines) + reserved header gutter for ⋮/☰.
- POL-04: harden the full theme/tab/font repaint path, not a minimal patch.
- POL-02: sun/moon icon + label knob for colorblind-safe Light/Dark state.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. The formal regression-test program
(automated suite + manual checklist + standing convention, TEST-01..05) is the
next phase (143) and is explicitly out of scope here.

</deferred>

---

*Phase: 142-hub-settings-redesign-polish*
*Context gathered: 2026-06-21*
