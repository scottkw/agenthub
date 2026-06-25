# Phase 140: UI-Spec Gate - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-20
**Phase:** 140-ui-spec-gate
**Areas discussed:** Direction choice, Accent & visual feel, Conflict reconciliation, #93 backlog triage

---

## Direction choice

Artifact provenance clarified by user: the standalone HTML is a single-file download
from Claude Design of the redesign they worked out; the subfolder `AgentHub UI redesign`
is the extracted export ZIP (`.dc.html` sources + `c-*.png` rendered screens).

Inspection found the standalone is **Direction 01 "Refined Native" built out** (sidebar +
tabs, blue-rendered, comfortable density, tmux footer), but on the **pre-Hub-first** nav.

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — 01 Refined Native | Record 01 as chosen; standalone HTML is reference | ✓ |
| 01 base + borrow pieces | Refined Native base + explicit elements from 02/03 | |
| Open live standalone first | Drive dev-browser walkthrough before locking | |

**User's choice:** Yes — 01 Refined Native
**Notes:** Direction was authored/exported by the user in Claude Design; rendered screens
reviewed via `c-*.png`. No separate live-browser pass required to lock.

---

## Accent & visual feel

| Option | Description | Selected |
|--------|-------------|----------|
| #7C8CFF periwinkle (standalone) | Adopt the standalone's violet accent app-wide | |
| Keep current blue | Treat violet as incidental; keep existing blue accent | ✓ |
| You decide / unsure | Defer accent validation to Phase 141 | |

**User's choice:** Keep current blue
**Notes:** Refined Native structure/density applies, no color change. Canonical blue =
`--hub-accent` #7aa2f7 (dark) / #3d6fe8 (light). Follow-up "visual feel" check →
"Move to reconciliation" (no further light/dark/motion/density specifics needed now).

---

## Conflict reconciliation

The comp predates Hub-first; structural decisions (NAV/SHARE/CARD) win. The sidebar,
Sessions page, Remote page, share-footer copy, and Hub-has-no-comp resolutions were
presented as locked (structural-wins). The one judgment call — restyle depth for the
existing share dialog + Hub cards — was put to the user (after a plain-language re-frame
when the user asked what they were deciding):

| Option | Description | Selected |
|--------|-------------|----------|
| Defer to Phase 141 | Spec notes restyle happens in 141; layout-rebuild decision deferred | |
| Recolor only, keep as-is | 141 recolors/retypesets share dialog + cards; no layout/interaction change | ✓ |
| Rebuild layout from comp | 141 rebuilds share dialog layout to match comp | |

**User's choice:** Recolor only, keep as-is
**Notes:** Preserves Phase 137 share model and shipped card interactions. Comp's
Sessions-page share-control layout NOT harvested.

---

## #93 backlog triage (CARRY-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Re-defer all, keep #93 open | None of 7 items fit a restyle phase; all re-deferred, #93 updated | ✓ |
| Pull in a subset | User names items to bring into Phase 141 scope | |

**User's choice:** Re-defer all, keep #93 open
**Notes:** In-scope subset for Phase 141 = none. Updating issue #93 with the triage
outcome is an execution action for the Phase 140 plan.

---

## Claude's Discretion

- Exact structure / file location of the UI spec artifact produced by the plan.

## Deferred Ideas

- All 7 #93 / #78 fidelity items (usage metrics, projects model, member avatars,
  structured briefings, session-detail page, tweaks panel, pixel-faithful port) —
  kept on issue #93 for future feature phases.
- Direction 02 (Command Workspace: ⌘K bar, live share rail) and Direction 03
  (Mission Control: gradient/glassy card deck) elements — inspiration for future
  phases, out of v4.0 scope.
