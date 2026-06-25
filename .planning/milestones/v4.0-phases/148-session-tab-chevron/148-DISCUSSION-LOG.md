# Phase 148: Session Tab Chevron - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-22
**Phase:** 148-session-tab-chevron
**Areas discussed:** Menu anchor, Placement, Which tabs, Menu theming

---

## Menu anchor

| Option | Description | Selected |
|--------|-------------|----------|
| Drop below chevron | Anchor the menu under the chevron button (via getBoundingClientRect) so it reads as a real dropdown. Right-click keeps cursor-position behavior. | ✓ |
| At cursor (reuse) | Open at the click's clientX/clientY — identical code path to right-click, zero new positioning logic. | |

**User's choice:** Drop below chevron
**Notes:** Right-click path unchanged (cursor coords); only the chevron path uses rect-anchored positioning. Both drive the same contextMenu state. → D-01/D-02

---

## Placement

| Option | Description | Selected |
|--------|-------------|----------|
| Before the × (name side) | Order becomes …name · chevron · close ×. Chevron groups with the title; close stays at the far edge. | ✓ |
| After the × (far edge) | Order becomes …name · close × · chevron. Chevron rightmost; moves close button inward. | |

**User's choice:** Before the × (name side)
**Notes:** New order: status · badge · name · (countdown) · chevron · close ×. → D-03

---

## Which tabs

| Option | Description | Selected |
|--------|-------------|----------|
| Session tabs only | Show chevron only on terminal/session tabs (truthy sessionId) — agent + shell. Special tabs stay clean. | ✓ |
| All tabs | Show on every tab including special ones; special-tab menu offers only Rename. | |

**User's choice:** Session tabs only
**Notes:** Gated on `tab.sessionId` truthiness, same predicate as the existing "Browse files" item. Tighter gate than the close × (intentional). → D-05

---

## Menu theming

| Option | Description | Selected |
|--------|-------------|----------|
| Fold in the fix | Convert .tab__context-menu hardcoded dark colors (#1e2030 etc.) to --hub-* tokens so the menu renders correctly in light mode. Serves Success Criterion #3. | ✓ |
| Chevron only, defer | Keep phase to just the chevron + its theming; defer the pre-existing menu light-theme bug. | |

**User's choice:** Fold in the fix
**Notes:** The chevron makes the menu materially more discoverable, so its light-theme correctness now matters. Pre-existing hardcoded colors at style.css ~1624–1651. → D-07

---

## Claude's Discretion

- Exact Unicode glyph (`▾`/`▿`/`▼`) and precise pixel sizing/padding to balance against the `×`.
- Exact `--hub-*` token mapping for the context-menu surface/border/hover.
- Dropdown vertical offset / viewport-bottom edge-clamping for the rect-anchored menu.
- Whether to add a distinct `data-testid` for the chevron (recommended for the regression test).

## Deferred Ideas

None — discussion stayed within phase scope. The context-menu light-theme fix was folded in (D-07), not deferred.
