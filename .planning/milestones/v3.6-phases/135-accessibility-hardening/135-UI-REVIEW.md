---
phase: 135-accessibility-hardening
audited: 2026-06-19
baseline: 135-UI-SPEC.md
overall_score: 21/24
pillar_scores:
  copywriting: 3/4
  visuals: 4/4
  color: 4/4
  typography: 4/4
  spacing: 4/4
  experience_design: 2/4
screenshots: "not captured — no dev server; code-only audit"
resolution:
  fixed: ["Modal status unreachable by AT — added sr-only status label in HubModal header (commit 17155833)"]
  deferred: ["WR-04 / Exp.Design: GroupSidebar role=listbox roving-tabindex model → issue #97"]
  advisory_accepted:
    - "IN: prefersReducedMotion read once at mount (no matchMedia change listener) — transient modal, low impact"
    - "IN-02: aria-pressed pills vs role=radiogroup — single-select semantics upgrade opportunity; current form tested + WCAG-passing"
    - "Copywriting: hub-filter__new-session has no explicit aria-label (visible text is sufficient accessible name)"
---

# Phase 135: UI Audit (6-Pillar) — Accessibility Hardening

**Overall: 21/24.** Code-only audit against the approved 135-UI-SPEC.md (no dev server for screenshots).

| Pillar | Score | Summary |
|--------|-------|---------|
| Copywriting | 3/4 | ARIA labels correct per spec; modal status label gap (now FIXED); new-session button relies on visible text (acceptable). |
| Visuals | 4/4 | 11 `:focus-visible` selectors, uniform `2px var(--hub-accent)` ring; bare `.hub-card:focus` removed (no mouse-click ring); no regressions. |
| Color | 4/4 | All 6 statuses carry unique icon shape + text label; color reinforcement only. Both STATUS_CONFIGs verified at hex constants, not by eye (colorblind-safe). `errored`/`stopped-err` share an icon but differ by label ("Error" vs "Exited") per A11Y-01. |
| Typography | 4/4 | No new type roles; phase adds no font changes. |
| Spacing | 4/4 | `outline-offset: 2px` (spec-blessed sub-4px exception); no new spacing tokens; no arbitrary values. |
| Experience Design | 2/4 | Focus trap + scoped Escape + reduced-motion fallbacks all correct; dinged for the WR-04 listbox ARIA-model inconsistency (deferred #97) and the modal-status-AT gap (now fixed). |

## Key findings & disposition

1. **Modal status unreachable by AT (Copywriting + Exp.Design)** — the HubModal header status icon was decorative-only (Heroicons hard-code `aria-hidden="true"`), so a screen-reader user inside the dialog could not determine the session status. **FIXED** (commit `17155833`): added a visually-hidden (`.sr-only`) status label in the modal header mirroring SessionCard's visible status-label span; icon stays decorative; no visual change. Behavioral test added.

2. **WR-04: `role="listbox"`/`role="option"` model inconsistency in GroupSidebar** — every option is its own tab stop; the listbox pattern wants roving tabindex + arrow keys (or drop the roles). **DEFERRED** to issue **#97** (rewrites a pre-Phase-135 component's ARIA model; out of hardening scope).

3. **Advisory (accepted, not changed):** `prefersReducedMotion` read once at mount without a `matchMedia` change listener (transient modal); `aria-pressed` pills vs `role=radiogroup` semantic upgrade; `hub-filter__new-session` relies on visible text as its accessible name.

## Strong points
- Color pillar fully colorblind-safe and source-verified; focus rings consistent and token-driven; reduced-motion fallbacks complete across all six animated features; A11Y-04 focus trap correct (incl. WR-01 exit-phase fix) with behavioral coverage.

_Audited 2026-06-19 by gsd-ui-auditor. Registry audit skipped (shadcn not initialized)._
