# Phase 141: Redesign Implementation - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Apply the locked **Direction 01 — "Refined Native"** visual language across all
surviving surfaces as a **recolor-only** pass (D-13). Concretely:

- Migrate the non-Hub surfaces (Welcome, terminal/tab bar/status bar, File
  Browser, Editor chrome, Settings, Share Modal) from hardcoded TokyoNight hex
  to the `--hub-*` CSS token system so the `[data-ui-theme="light"]` toggle
  works uniformly across every surface (light + dark).
- Add the missing `.hub-share-modal` / `.hub-share-modal__*` CSS rules (S-07 gap).
- Fix the Hub GroupSidebar ARIA model (CARRY-01 / #97).
- Remove/reword all "Sessions tab" copy (D-11).

**Requirements:** RDS-02, RDS-03, RDS-04, CARRY-01.

**Not in scope:** layout/structural/interaction changes to any surface;
re-introducing Sessions/Remote sidebar pages or a "+ New Session" sidebar item;
changing the locked accent; CodeMirror internal theme (`--cm-*`).

</domain>

<spec_lock>
## Design Contract (locked via UI-SPEC.md)

**`141-UI-SPEC.md` is the canonical, approved design contract** (status:
approved, reviewed 2026-06-20). It locks direction, accent, per-surface restyle
targets (S-01…S-07), the CARRY-01 ARIA resolution, the colorblind-safety table,
the motion contract, the copywriting contract, and the full Phase Constraints
(MUST / MUST NOT) list.

**Downstream agents MUST read `141-UI-SPEC.md` before planning or implementing.**
This discussion does not re-litigate any decision in that document — it only
resolves the handful of micro-decisions the spec left latitude on (below).

</spec_lock>

<decisions>
## Implementation Decisions

This phase needed almost no new discussion — the approved UI-SPEC pre-answers
nearly everything. The user chose to skip detailed discussion and proceed
straight to planning, with the three open micro-decisions resolved to their
sensible defaults:

### Hub visual scope
- **D-01:** Phase 141 Hub work is **ARIA + copy only**. The Hub already
  consumes `--hub-*` tokens (S-02), so no recolor is required. Resolve the
  S-02 wording tension ("ARIA only" vs "minor visual polish allowed") in favor
  of **no Hub visual changes** — do not touch Hub card layout or visuals. Hub
  deliverables for this phase are the CARRY-01 ARIA fix and any D-11 copy reword
  that lands on Hub surfaces.

### Editor chrome (S-05, no comp)
- **D-02:** The Editor (CodeMirror 6) chrome — breadcrumb, save controls — is
  restyled by **matching the File Browser surface** (same `--hub-*` token
  migration). No separate Editor comp is needed; deriving from File Browser is
  sufficient. CodeMirror's internal theme (`--cm-*` variables) is **out of
  scope** — only the surrounding chrome is restyled.

### Token migration risk flags (for the researcher)
- **D-03:** Researcher should confirm these boundaries hold during migration:
  - Agent badge colors (`.tab__agent-badge--*`) stay as semantic per-agent
    identifiers — **not** migrated to theme tokens.
  - Semantic status colors (Running/Idle/Waiting/Errored/Stopped/etc.) keep
    their existing hex values; they are reinforcement-only and colorblind-safe
    via icon + text. Do not fold them into `--hub-accent`/`--hub-*` chrome tokens.
  - New `--hub-*` tokens are introduced **only** when no existing token covers
    the semantic need, and must be added to **both** `:root` (dark) and
    `[data-ui-theme="light"]` blocks.

### Claude's Discretion
- Per-surface migration ordering and plan-splitting are the planner's call.
- Exact new token names (where genuinely needed) are Claude's call, following
  the existing `--hub-*` naming convention.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design contract (LOCKED — read first)
- `.planning/phases/141-redesign-implementation/141-UI-SPEC.md` — The full,
  approved design contract: direction lock, accent locks, token system,
  per-surface restyle targets (S-01…S-07), CARRY-01 ARIA contract, colorblind
  and motion contracts, copywriting contract, and Phase Constraints.
- `.planning/phases/140-ui-spec-gate/140-UI-SPEC.md` — Upstream decision gate
  (D-05 accent rejection, D-08/09/10 structural fences, D-11 copy fix, D-13
  recolor-only scope). The 141 spec inherits all of these.

### Canonical visual reference
- `agenthub-v4.0-redesign/AgentHub Redesign (standalone).html` — Open in a
  browser. Direction 01 visual reference. **Caveat:** its accent `#7C8CFF` is
  REJECTED; use `#7aa2f7` (dark) / `#3d6fe8` (light) instead.

### Implementation source of truth
- `frontend/src/style.css` — Token declarations at lines 3896–3995
  (`:root` dark + `[data-ui-theme="light"]`); status colors ~4217–4228;
  non-Hub surfaces with hardcoded hex to migrate (`.welcome-tab*`, `.tab-bar`,
  `.tab*`, `.tab-status-bar*`, `.file-browser*`, `.settings-panel*`, etc.).
- GroupSidebar component (Hub `aside`, `<ul role="listbox">` to be fixed per
  CARRY-01) — locate via the ARIA contract in 141-UI-SPEC.md §CARRY-01.

### Requirements / scope
- `.planning/REQUIREMENTS.md` — RDS-02, RDS-03, RDS-04, CARRY-01.
- `.planning/ROADMAP.md` §Phase 141 — success criteria.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `--hub-*` token system (`frontend/src/style.css` 3896–3995): the migration
  target. Hub, modals, and cards already consume it; the job is extending it to
  the remaining surfaces.
- `@heroicons/react 24/outline` (already installed): the icon set for all
  colorblind-safe status/origin signals — no new icon dependency needed.
- Existing motion-compliant examples to copy the pattern from: `.find-bar`,
  `.banner`, `.webgl-recovery-banner`, `.hub-card--attention`, `.hub-modal`.

### Established Patterns
- Theme switching is purely `[data-ui-theme="light"]` attribute-conditioned CSS.
  Every migrated rule must add the matching light-theme override.
- `prefers-reduced-motion`: animations live inside
  `@media (prefers-reduced-motion: no-preference)` with static fallbacks in
  `@media (prefers-reduced-motion: reduce)`.
- Colorblind verification is done at the **hex-constant level in source** (grep),
  never by eye — the user is colorblind.

### Integration Points
- `.hub-share-modal` classes are referenced in the SessionShareModal TSX but
  have **no CSS rules** — Phase 141 adds them (reuse the `hub-modal`
  header/body/footer layout pattern).

</code_context>

<specifics>
## Specific Ideas

The "look like X" reference is fully captured in the UI-SPEC: Direction 01
"Refined Native", canonical visual `agenthub-v4.0-redesign/AgentHub Redesign
(standalone).html` (accent excepted). The tmux-style footer status bar
(`tab-status-bar`) is a key Direction 01 characteristic to retain and restyle.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. The #93 / #78 Hub-fidelity backlog
was already triaged and re-deferred in Phase 140 (CARRY-02, complete); no items
were pulled into 141.

</deferred>

---

*Phase: 141-redesign-implementation*
*Context gathered: 2026-06-20*
