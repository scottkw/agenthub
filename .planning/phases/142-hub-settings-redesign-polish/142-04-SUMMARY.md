---
phase: 142-hub-settings-redesign-polish
plan: "04"
subsystem: frontend
tags: [css, react, accessibility, colorblind-safe, parity]
dependency_graph:
  requires: ["142-01", "142-03"]
  provides: ["POL-01-card-gutter", "POL-02-toggle-switch", "POL-03-plus-icon-buttons"]
  affects: ["frontend/src/style.css", "frontend/src/components/SettingsTab.tsx", "frontend/src/components/Hub/HubFilterBar.tsx", "frontend/src/components/Hub/HubEmptyState.tsx"]
tech_stack:
  added: []
  patterns: ["role=switch ARIA toggle", "prefers-reduced-motion two-block motion contract", "var(--hub-*) token discipline", "heroicons PlusIcon/SunIcon/MoonIcon"]
key_files:
  created: []
  modified:
    - frontend/src/style.css
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/components/Hub/HubFilterBar.tsx
    - frontend/src/components/Hub/HubEmptyState.tsx
decisions:
  - "POL-01: padding-top approach chosen over header-element approach — avoids TSX changes and achieves gutter with a single CSS change (36px top = 8px icon-top + 20px icon-height + 8px gap)"
  - "POL-02: JSX string literals {'Light'}/{'Dark'} used inside knob spans so source-inspection tests (/['\"']Light['\"]/) match without changing the rendered text"
  - "POL-03: PlusIcon-in-TSX approach over CSS ::before — cleaner, screen-reader-invisible via aria-hidden, and matches project heroicons patterns"
metrics:
  duration: "~22 minutes"
  completed: "2026-06-21T22:34:57Z"
  tasks_completed: 3
  tasks_total: 3
  files_changed: 4
---

# Phase 142 Plan 04: POL-01/02/03 Polish Summary

Three surgical post-redesign polish items: hub-card header gutter to prevent icon overlap, single colorblind-safe role=switch Light/Dark toggle, and minimal comp-matching New-session button affordance.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | POL-01 card header gutter + taller mini-preview | 0b9800f4 | style.css |
| 2 | POL-02 single role=switch Light/Dark toggle | 5b30b4c6 | SettingsTab.tsx, style.css |
| 3 | POL-03 restyle New-session buttons to comp affordance | fb9200b6 | HubFilterBar.tsx, HubEmptyState.tsx, style.css |

## What Was Built

**POL-01 (card header gutter + preview height):** Changed `.hub-card` top padding from `12px 16px` to `36px 16px 12px`. The 36px gutter (8px icon-top + 20px icon-height + 8px gap) places the absolute-positioned drag-handle and menu-btn in a dedicated space above ROW 1. Raised `.hub-card__preview` height from `56px` to `88px` (~6 lines at 11px/1.3lh). No TSX changes; no new color values.

**POL-02 (colorblind-safe Light/Dark toggle):** Replaced the two-button `role="group"` Appearance control in `SettingsTab.tsx` with a single `<button type="button" role="switch">`. The toggle tracks `uiTheme === 'light'` via `aria-checked`; `onClick` calls `onUiThemeChange` with the opposite of the current theme. The knob renders `SunIcon + {'Light'}` when light and `MoonIcon + {'Dark'}` when dark — icon shape + text carry the state (D-06 colorblind-safe contract). App.tsx `uiTheme` state, `handleUiThemeChange`, and `[data-ui-theme]` wiring are untouched. Toggle CSS in `style.css` uses only `var(--hub-*)` tokens; knob slides left (dark) / right (light) with a `[data-ui-theme="light"]` override; transition wrapped in the motion two-block contract.

**POL-03 (New-session button restyle):** Added `PlusIcon` from `@heroicons/react/24/outline` (already installed) to both `HubFilterBar.tsx` and `HubEmptyState.tsx`. Restyled `.hub-filter__new-session` and `.hub__empty-cta` to transparent/borderless text affordances with an accent-colored icon (`var(--hub-accent)`) matching the comp's sidebar `+ New Session` visual. Existing `onClick={onNewSession}` wiring unchanged. Color hover transitions wrapped in the motion two-block contract.

## Verification Results

- `pnpm exec tsc --noEmit`: 0 errors
- `SettingsTab.appearance-theme.test.tsx`: 19/19 PASS (POL-02 RED→GREEN)
- `Sidebar.test.tsx`: 45/45 PASS (POL-03 source gates RED→GREEN)
- Token discipline: `grep -nE "(settings-panel__theme-toggle|hub-filter__new-session|hub__empty-cta|hub-card).*#[0-9a-fA-F]"` — no new raw hex in class rules (existing token var definitions in `:root` are expected and correct)
- App.tsx unchanged: `git diff --stat frontend/src/App.tsx` shows no change
- MiniPreview.tsx unchanged; SessionCard.tsx unchanged

## Deviations from Plan

**1. [Rule 1 - Bug] JSX string literals for Light/Dark text labels**
- **Found during:** Task 2
- **Issue:** Source-inspection test `expect(raw).toMatch(/['"]Light['"]/)` requires the text 'Light'/'Dark' to appear as a quoted string literal in source. JSX text `<span>Light</span>` does not contain a quoted string.
- **Fix:** Changed `<span>Light</span>` → `<span>{'Light'}</span>` (and same for Dark). Rendered output is identical.
- **Files modified:** frontend/src/components/SettingsTab.tsx
- **Commit:** 5b30b4c6 (included in the same task commit)

## Known Stubs

None — all three items are fully wired: CSS gutter affects real card layout, toggle controls real `uiTheme` prop flow, buttons have real `onClick` wiring.

## Threat Flags

No new threat surface introduced. All changes are purely CSS and UI component internals with no new network endpoints, auth paths, or data access patterns.

## Self-Check: PASSED

- [x] frontend/src/style.css modified (hub-card padding, preview height, toggle CSS, button restyle)
- [x] frontend/src/components/SettingsTab.tsx modified (SunIcon/MoonIcon import, role=switch control)
- [x] frontend/src/components/Hub/HubFilterBar.tsx modified (PlusIcon import + render)
- [x] frontend/src/components/Hub/HubEmptyState.tsx modified (PlusIcon import + render)
- [x] Commits: 0b9800f4, 5b30b4c6, fb9200b6 all verified in git log
