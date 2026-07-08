# Phase 172: Hub-card layout & badge refinement - Context

**Gathered:** 2026-07-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Frontend-only visual refinement of the Hub session card (`frontend/src/components/Hub/SessionCard.tsx` + `frontend/src/style.css`). Today the card stacks THREE inconsistent metadata treatments — status (`Running`, icon+plain-text), CLI/agent (`/bin/zsh`, outlined pill), origin (`Local`, icon+colored-text) — plus the `INTERNET` filled-green pill on its own row and (Phase 171) a `FULL ACCESS` filled-red notched badge on another. This phase consolidates the *quiet* identity metadata into ONE consistent chip row (agent · origin · exposure) with tighter vertical rhythm, while deliberately keeping the exposure badge(s) the prominent filled chips so they pop MORE by contrast.

**In scope:** SessionCard.tsx markup restructuring + style.css chip/badge/row styling (dark + light theme tokens). Throwaway HTML mockups first.

**Out of scope:** Any backend/daemon change, new session data, changes to the exposure *semantics* (funnelActive / funnelWriteActive logic is fixed), the Share modal, MiniPreview rendering, ChatBadge, the attention-pulse behavior itself. No new capabilities — this is design polish only.

</domain>

<decisions>
## Implementation Decisions

### Quiet-chip style
- **D-01:** Agent + origin become **outlined ghost pills** — bordered, transparent background, muted/quiet. Origin gets pillified to match the existing `.hub-card__badge` (agent) treatment so the two read as one consistent chip family. Rationale: everything quiet = outlined; the filled exposure chip becomes the single loud element by contrast.
- **D-02:** Colorblind-safe carries forward unchanged — every chip keeps an icon shape + text label; color is reinforcement only. The contrast that makes INTERNET/FULL ACCESS pop is **structural** (outline vs fill), not merely chromatic — good for the colorblind requirement. Verify all hex at source, never by eye ([[user_colorblind]]).

### Status placement
- **D-03:** Status (`Running` / `Needs input` / etc.) **stays the primary top-line signal**, ABOVE the new chip row. It keeps its spin animation and the attention pulse. The chip row (agent · origin · exposure) sits below as secondary metadata. Status is NOT chipified — it remains the card's #1 state signal.

### Exposure badges (INTERNET + FULL ACCESS)
- **D-04:** The exposure badges **join the chip row** as its prominent right-end cluster (the "exposure" in agent · origin · exposure). They are the only FILLED chips in the row.
- **D-05:** INTERNET (read) and FULL ACCESS (write) **coexist** when write is active — matching current read-many/write-one logic (`funnelActive` renders INTERNET; `funnelWriteActive` additionally renders FULL ACCESS). Do NOT collapse to a supersede model. FULL ACCESS's load-bearing shape distinction from INTERNET (notched `clip-path` vs pill radius, heavier font weight, LockOpenIcon vs GlobeAltIcon) is a Phase-171 colorblind guarantee and MUST be preserved.

### Rhythm & meta
- **D-06:** Uptime, viewer count, and the remote Connected/Available indicator move to a **muted meta line below the chip row** (close to today's `.hub-card__row2-meta`, just tightened). Clear separation: chips = "what it is" (identity/exposure), meta line = "live stats." Tighten vertical rhythm across the card (reduce the loose row stacking that prompted the critique).

### Process
- **D-07:** Produce **2-3 throwaway HTML mockups** BEFORE touching component/CSS code (per ROADMAP approach note). The chip-STYLE direction is already locked (outlined ghost pills), so the mockups compare **variations WITHIN that direction** — e.g. border weight / radius, chip gap & spacing, meta-line density, and critically how the exposure cluster + a long remote hostname WRAP when both badges coexist. Mockups pick the final polish, not re-litigate the direction. Use `/gsd-sketch` (throwaway HTML) or the frontend-design skill; render for review before planning code changes.

### Claude's Discretion
- Exact border weight, corner radius, gap values, and padding for the outlined pills — to be pinned via the mockups (D-07).
- Long remote-hostname handling inside its origin pill (truncate/ellipsis vs wrap) — surface in the mockups; pick the cleanest.
- Whether the origin pill keeps its color-coded text (green local / blue remote) or goes fully muted — resolve visually in mockups, keeping color as reinforcement-only.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase target files
- `frontend/src/components/Hub/SessionCard.tsx` — the card component. Current row structure: header (name + ChatBadge) → row1 (status indicator + `.hub-card__badge` agent chip pushed right) → row2 (origin + row2-meta uptime/viewers) → row2b (remote conn) → row4 (exit chip) → `.hub-internet-badge` → `.hub-fullaccess-badge` → row5 (Open/Share) → MiniPreview.
- `frontend/src/style.css` — chip/badge/row styles. Key anchors: `.hub-card__row1` (~4987), `.hub-card__row2`/`__row2-meta` (~4994–5017), `.hub-card__badge` + `[data-agent]` tints (~5132–5154), `.hub-card__origin` (~5157), `.hub-card__conn` (~5177), `.hub-internet-badge` (~7194), `.hub-fullaccess-badge` (~7509). Both dark and light theme token sets (`--hub-internet-badge-bg/-text`, `--hub-fullaccess-badge-bg/-text`) must stay verified.

### Design intent record
- `.planning/STATE.md` — Phase 172 entry captures the full critique (three inconsistent treatments) + direction (one chip row, INTERNET stays prominent). Origin: commit `4402b44e`.
- ROADMAP.md Phase 172 section — goal statement + approach note (mockups-first).

### Colorblind constraint
- MEMORY `[[user_colorblind]]` — verify color-based decisions at source (hex constants in code), not by eye. Every status/chip/badge already carries an icon shape + text label; color is reinforcement only.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `.hub-card__badge` + the `.hub-card[data-agent="..."]` tint rules: the existing outlined-pill treatment for the agent chip — the template the origin pill should match (D-01).
- `.hub-card__row2-meta`: existing muted right-aligned meta group (uptime + viewers) — the basis for the consolidated meta line (D-06).
- `STATUS_CONFIG` map in SessionCard.tsx: status icon/label/spin — unchanged; status stays primary (D-03).
- `.hub-internet-badge` / `.hub-fullaccess-badge`: keep their filled/notched geometry; only their placement moves into the chip row (D-04/D-05).

### Established Patterns
- Colorblind-safe: icon shape + text label on every state carrier; color reinforcement only (STATUS_CONFIG comments, badge CSS comments). New chips must follow this.
- `data-agent` drives the card left-spine + agent chip tint + matches the tab agent-badge dot — keep this coupling.

### Integration Points
- Remote-only branches: `isRemote`/`isConnected` drive the origin (hostname) and the Connected/Available indicator — both must survive the meta-line consolidation (D-06).
- `session.funnelActive` / `session.funnelWriteActive` gate the two exposure badges — logic unchanged; only their DOM position/row membership changes.

</code_context>

<specifics>
## Specific Ideas

- Mental model for contrast: quiet outlined pills + one/two FILLED exposure chips → the filled chip is unmistakably the loudest thing on the card. "Making the other chips quieter makes INTERNET pop MORE."
- Target chip row reads: `( ⬚ agent ) · ( ⬚ origin )   [● INTERNET] [⭢ FULL ACCESS]` with status on the line above and a muted `2h 14m · 3 viewers · Connected` line below.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope (frontend card polish only).

</deferred>

---

*Phase: 172-hub-card-layout-badge-refinement*
*Context gathered: 2026-07-07*
