# Phase 148: Session Tab Chevron - Context

**Gathered:** 2026-06-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Add a **down-chevron affordance** to each **session tab** (agent + shell) in the AgentHub desktop GUI's `TabBar.tsx`. Left-clicking the chevron opens the **existing** tab context menu (Rename / Save Terminal As… / Browse files) — the same menu currently reachable only by right-click. The chevron makes that right-click-only menu **discoverable** (the motivation in #68: new users miss the menu because right-click isn't self-advertising).

**In scope:** the chevron icon on session tabs, its left-click → open-menu wiring, hover affordance, keyboard activation + `aria-label`, light/dark theming of the chevron, AND folding in the light-theme fix for the menu it opens (see D-07). Single shared component (`TabBar.tsx`) serves both agent and shell tabs.

**Out of scope:** changing the menu's items or behavior; altering right-click behavior (it is preserved exactly); a chevron on non-session special tabs (Welcome/Settings/Hub/Help/File-browser — see D-05); any new menu actions.

**Locked by issue #68 (not re-decided here):** down-chevron glyph as the indicator; left-click opens the same menu; right-click preserved (no regression for power users); subtle hover state; chevron focusable + Enter/Space activatable with `aria-label` "Session menu"; chevron visible on all session tabs (active AND inactive, so users can act on background sessions); matches existing tab affordance weight/size.
</domain>

<decisions>
## Implementation Decisions

### Menu anchoring (chevron vs right-click)
- **D-01:** When the menu is opened **via the chevron**, anchor it as a **dropdown below the chevron button** — compute position from the button's `getBoundingClientRect()` rather than mouse coords. This reads as a real dropdown tied to the affordance.
- **D-02:** **Right-click keeps its existing cursor-position behavior** (`clientX`/`clientY`). Only the chevron path uses rect-anchored positioning. Both paths drive the same `contextMenu` state and render the same `.tab__context-menu`.

### Chevron placement in the tab
- **D-03:** Chevron sits **before the `×` close button** on the name side. New tab order: `status dot · agent badge · name · (countdown) · chevron · close ×`. The close button stays at the far right edge where users expect it; the chevron groups with the title.

### Icon implementation
- **D-04:** Use a **Unicode down-chevron glyph** (e.g. `▾` / `&#9662;`) to match the established tab-affordance approach — the close button is a Unicode `×` and the tab-list scroll controls are Unicode `‹`/`›` (`&#8249;`/`&#8250;`). No SVG/icon-library dependency. Match the close button's size/weight (16×16 button, ~14px glyph) and theming pattern.

### Which tabs get the chevron
- **D-05:** **Session tabs only** — render the chevron only on tabs with a truthy `sessionId` (agent + shell terminal tabs). Special tabs (Welcome/Settings/Hub/Help/File-browser) do **not** get a chevron; their menu would offer only "Rename," and #68 frames the feature as "agent and shell session tabs." Keeps special tabs clean. (Note: this is a tighter gate than the close `×`, which renders on all tabs — intentional.)

### Accessibility
- **D-06:** Chevron is a semantic `<button>` (native keyboard focus + Enter/Space), with `aria-label="Session menu"` (per #68). Follows the existing close-button pattern (`tab__close` is already a real button with `aria-label`).

### Light/dark theming (folded fix)
- **D-07:** **Fold in the context-menu light-theme fix.** The menu `.tab__context-menu` currently uses **hardcoded dark colors** (`background:#1e2030`, `border:#292e42`, `color:#a9b1d6`, hover `#c0caf5`/`#292e42` — style.css ~lines 1624–1651) that do **not** adapt to light theme. Because the chevron makes this menu far more discoverable, convert those hardcoded values to the `--hub-*` token system so the menu renders correctly in both themes. This directly serves Success Criterion #3 ("works in light/dark themes"). The chevron's own colors must also use `--hub-*` tokens (reuse the `tab__close` token choices: `--hub-text-muted` default, `--hub-border-hover` bg + brighter text on hover).

### Claude's Discretion
- Exact Unicode glyph (`▾` vs `▿` vs `▼`) and precise pixel sizing/padding to visually balance against the `×` — pick whatever reads cleanly at tab scale in both themes.
- Exact `--hub-*` token mapping for the context-menu surface/border/hover (choose the closest-matching existing tokens; the menu is an elevated popover, so `--hub-surface-elevated` / `--hub-border` / `--hub-text-*` are the natural candidates).
- Dropdown vertical offset/edge-clamping for the rect-anchored menu (avoid clipping at the viewport bottom) — standard popover hygiene, planner's call.
- Whether the chevron uses a distinct `data-testid` for the regression test (recommended, mirroring `tab-context-browse-files`).
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirement & issue source
- `.planning/REQUIREMENTS.md` — **TAB-04** definition (Phase 148, #68).
- GitHub issue **#68** (`scottkw/agenthub`) — "Add down chevron indicator to session tabs for menu discoverability." Authoritative behavior/acceptance spec: chevron icon, left-click opens menu, right-click preserved, hover affordance, keyboard + `aria-label "Session menu"`, visible on all (active + inactive) session tabs, match existing tab affordance style.
- `.planning/ROADMAP.md` §"Phase 148: Session Tab Chevron" — goal + 3 success criteria.

### Component under change
- `frontend/src/components/TabBar.tsx` — the single shared tab component (both agent + shell tabs). Context-menu trigger at the `.tab__name onContextMenu` (~line 226); menu render block (~lines 273–322); close button (~lines 239–249); existing scroll chevrons (~lines 175–182, 264–271, Unicode `‹`/`›`).
- `frontend/src/style.css` — tab styles: `.tab__close` (~lines 202–221, the token/sizing analog), `.tab-bar__chevron` (scroll-chevron style precedent), `.tab__context-menu` (~lines 1624–1651, the hardcoded colors to tokenize per D-07), `--hub-*` token definitions (dark ~4520–4586, light ~4589–4657) + `[data-ui-theme]` theming.

### Milestone scope / norms
- `.planning/PROJECT.md` — v4.0 scope; theme-token + colorblind-safe + `prefers-reduced-motion` release norms (the chevron hover and menu colors must honor these).
- `.planning/phases/147-in-app-help-page/147-CONTEXT.md` §D-11 — prior-phase confirmation of the "respect existing theme tokens, no bespoke palette, honor colorblind-safe + reduced-motion" norm.

### Regression-test convention
- `TESTING.md` (repo root) — new test files MUST be registered (Suite Manifest §2, Traceability §4 with a repo-relative path); run `bash tests/check-traceability-paths.sh` before committing. Add an M-NN manual item only if a behavior can't be automated (the chevron + menu-open is automatable via the existing TabBar test harness; light/dark theming is verified at source level — hex/token constants — per the colorblind-source-verification norm).
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`contextMenu` state + `.tab__context-menu` render block** (`TabBar.tsx`): already drives Rename / Save Terminal As… / Browse files. The chevron reuses this verbatim — it only needs a new `onClick` that calls `setContextMenu({ tabId, x, y })` with rect-derived coords (D-01). No menu-item changes.
- **`.tab__close` button** (`TabBar.tsx` ~239–249, `style.css` ~202–221): the structural + styling template for the chevron — semantic `<button>`, `stopPropagation` on click (so it doesn't select the tab), 16×16, Unicode glyph, `--hub-*` hover tokens. Copy this pattern.
- **`--hub-*` token system** (`style.css`, `[data-ui-theme]`): supplies the light/dark values for both the chevron and the tokenized context menu (D-07).

### Established Patterns
- Tab affordances are **Unicode glyphs**, not an icon library (close `×`, scroll `‹`/`›`). The chevron follows suit (D-04) — no new dependency.
- Interactive tab elements are real `<button>`s with `aria-label` and `e.stopPropagation()` to avoid triggering the tab's `onClick` select. The chevron must `stopPropagation` so clicking it opens the menu without switching tabs.
- Session-gating uses `tab.sessionId` truthiness (the "Browse files" item is already gated this way). The chevron's session-tab-only rule (D-05) uses the same predicate.

### Integration Points
- `TabBar.tsx` is the only component touched for the chevron + wiring. `style.css` is touched for chevron styles + the context-menu tokenization (D-07).
- The menu rect-anchoring (D-01) needs the chevron button's `getBoundingClientRect()`; existing menu positioning is `position: fixed` with `top/left`, so the chevron path just supplies different `x`/`y` into the same state shape — no structural change to the menu render.
</code_context>

<specifics>
## Specific Ideas

- New tab element order (D-03): `status dot · agent badge · name · (countdown) · chevron · close ×`.
- Chevron `aria-label` text is exactly **"Session menu"** (#68).
- Menu, when chevron-opened, drops **below** the chevron (D-01); right-click stays at the cursor (D-02).
- The folded fix (D-07) is specifically the `.tab__context-menu` hardcoded hex values → `--hub-*` tokens so the menu is correct in light mode.
</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. (The context-menu light-theme fix was *folded in* per D-07 rather than deferred, because Success Criterion #3 requires light/dark correctness and the chevron makes that menu materially more discoverable.)

</deferred>

---

*Phase: 148-session-tab-chevron*
*Context gathered: 2026-06-22*
