---
phase: 141-redesign-implementation
plan: 02
subsystem: ui
tags: [css, tokens, theming, light-dark, colorblind, reduced-motion, sidebar, tab-bar, status-bar]

# Dependency graph
requires:
  - phase: 141-01
    provides: "--hub-* token declarations for both :root (dark) and [data-ui-theme='light'] blocks, including --hub-text-dim"

provides:
  - "Sidebar (lines 218–308) fully tokenized: --hub-surface/border/text-muted/surface-elevated/text-primary"
  - "sidebar__item--active uses var(--hub-sidebar-item-active-bg) and var(--hub-accent) (no fallback hex)"
  - "Welcome tab (lines 1330–1430) fully tokenized: bg/secondary/muted/primary/accent/surface/border"
  - "Tab bar, tabs, chevrons (lines 82–216) tokenized; transition extracted to motion guard"
  - "Terminal container background tokenized: var(--hub-bg)"
  - "Status bar fully tokenized including --hub-text-dim for hint; D-03 fences intact"
  - "All hover transitions for sidebar/tab/status-btn wrapped in prefers-reduced-motion guards"

affects:
  - "141-03 (file browser + editor chrome migration)"
  - "141-04 (Hub card + modal chrome migration)"
  - "141-05 (CARRY-01 GroupSidebar ARIA)"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Selector-by-selector token migration (not global find/replace) per Pitfall 4"
    - "Motion guard: no-preference block wraps transition declarations; reduce block sets transition:none"
    - "D-03 fence: agent-badge and status-state hex intentionally preserved as semantic colorblind-safe signals"

key-files:
  created: []
  modified:
    - "frontend/src/style.css"

key-decisions:
  - "Tab bar / sidebar / welcome: recolor-only (D-13) — no layout, spacing, or structural changes"
  - "D-03 fences preserved: .tab__agent-badge--* (7 per-agent hex) and .tab-status-bar__state--* (3 status hex) remain hardcoded per colorblind contract"
  - "Transitions moved out of base selector rules into motion-guard @media blocks per Hub group sidebar pattern"

patterns-established:
  - "Motion guard pattern (no-preference/reduce) applied to all migrated hover transitions"
  - "--hub-text-dim consumed for .tab-status-bar__hint (dimmer than --hub-text-muted)"

requirements-completed: [RDS-02, RDS-04]

# Metrics
duration: 20min
completed: 2026-06-21
---

# Phase 141 Plan 02: Sidebar, Welcome, Tab Bar & Status Bar Token Migration Summary

**Sidebar, Welcome tab (S-01), and terminal tab bar + status bar (S-03) migrated from TokyoNight hex to --hub-* token system with motion-guarded transitions and D-03 fences preserved**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-06-21T16:00:00Z
- **Completed:** 2026-06-21T16:20:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- Sidebar chrome (background, border, toggle/item color, hover states, active indicator) fully tokenized; `sidebar__item--active` now uses `var(--hub-sidebar-item-active-bg)` with no fallback hex
- Welcome tab (S-01) all 11 hex values replaced with `--hub-*` tokens across `background-color`, text colors, code-block chrome, and link color
- Tab bar, tab selectors, chevrons, terminal container, and status bar (S-03) fully tokenized; `--hub-text-dim` consumed for `.tab-status-bar__hint`
- D-03 fences intact: 7 `.tab__agent-badge--*` per-agent hex and 3 `.tab-status-bar__state--*` semantic status hex are untouched
- All hover/width transitions on migrated surfaces extracted from base selectors and wrapped in `@media (prefers-reduced-motion: no-preference)` / `reduce` blocks

## Task Commits

Each task was committed atomically:

1. **Task 1: Migrate sidebar (218–293) and Welcome tab (1302–1404) to tokens** - `29ab2a71` (feat)
2. **Task 2: Migrate tab bar + status bar + terminal container (82–368); preserve D-03 fences** - `066d813f` (feat)

**Plan metadata:** (see final commit)

## Files Created/Modified

- `frontend/src/style.css` - Sidebar, Welcome tab, tab bar, status bar, terminal container chrome tokenized; motion guards added

## Decisions Made

- Transition declarations removed from base selector rules and placed exclusively inside `@media (prefers-reduced-motion: no-preference)` guards, matching the Hub group sidebar pattern established in Plan 01
- `.sidebar` width transition placed in no-preference guard (layout-only, still guarded per plan spec)
- D-03 fences: agent badge and status-state hex retained verbatim — these are semantic colorblind-safe signals, not chrome colors

## Deviations from Plan

None — plan executed exactly as written. Token substitutions followed the RESEARCH hex tables and PATTERNS migration targets.

### Known False Positive in Acceptance Criteria Gate

The plan's gate `! sed -n '82,368p' frontend/src/style.css | grep -E '#[0-9a-fA-F]{3,6}' | grep -viE ...` has a pre-existing false positive: the comment `(#139 regression)` in the tab flex-basis comment contains `#139` which matches the regex but is not a CSS color value. All actual CSS color properties in that range have been tokenized. The D-03 fenced values (`#9ece6a`, `#9aa5ce`, `#414868`) are correctly excluded from the gate. This comment existed before Plan 02's changes.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 03 (file browser + editor chrome, lines 2646–3380+) can proceed: token vocabulary is identical, same pattern applies
- Plan 04 (Hub card + modal chrome) likewise unblocked
- TypeScript (`tsc --noEmit`) passes — CSS-only change has no TS impact
- All migrated surfaces will automatically recolor under `[data-ui-theme="light"]` toggle via the Plan 01 token overrides

## Threat Flags

None — CSS recolor only; no new network endpoints, auth paths, or data-flow changes introduced.

## Self-Check: PASSED

- `frontend/src/style.css` modified: confirmed (git log shows 066d813f)
- Task 1 commit `29ab2a71`: confirmed present
- Task 2 commit `066d813f`: confirmed present
- D-03 fences: `grep -A1 'tab__agent-badge--claude' | grep '#7aa2f7'` passes
- `--hub-text-dim` consumed: `grep -c 'var(--hub-text-dim)'` returns 1
- `var(--hub-accent, #7aa2f7)` fallback hex: removed (grep returns 0)
- `tsc --noEmit`: clean (no output)

---
*Phase: 141-redesign-implementation*
*Completed: 2026-06-21*
