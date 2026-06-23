---
phase: 148-session-tab-chevron
plan: "01"
subsystem: frontend
tags: [tab-bar, context-menu, accessibility, theming, vitest]
dependency_graph:
  requires: []
  provides: [tab__chevron button, rect-anchored menu open, context-menu token theming]
  affects: [frontend/src/components/TabBar.tsx, frontend/src/style.css]
tech_stack:
  added: []
  patterns: [Unicode-glyph button, getBoundingClientRect rect-anchoring, hub-token CSS variables]
key_files:
  created: []
  modified:
    - frontend/src/components/TabBar.tsx
    - frontend/src/style.css
    - frontend/src/components/__tests__/TabBar.test.tsx
    - TESTING.md
decisions:
  - D-01: Chevron opens menu anchored below itself via getBoundingClientRect (not cursor coords)
  - D-02: Right-click path keeps clientX/clientY unchanged — zero regression
  - D-03: Chevron inserted between countdown span and close button in DOM order
  - D-04: Unicode ▾ (U+25BE) glyph — no SVG/icon-library dependency
  - D-05: sessionId gate — special tabs (Welcome/Settings/Hub/Help/File-browser) get no chevron
  - D-06: Semantic button, aria-label="Session menu", native keyboard focus
  - D-07: context-menu hardcoded dark hex → --hub-* tokens for light/dark correctness
metrics:
  duration: "~25 minutes"
  completed: "2026-06-23"
  tasks_completed: 3
  files_changed: 4
---

# Phase 148 Plan 01: Session Tab Chevron Summary

**One-liner:** Down-chevron button (▾) on session tabs opens the existing context menu anchored below the chevron via getBoundingClientRect; context menu tokenized to `--hub-*` for light/dark correctness.

## Tasks Completed

| # | Name | Commit | Files |
|---|------|--------|-------|
| 1 (RED) | Add failing TAB-04 tests | 720ef346 | TabBar.test.tsx |
| 1 (GREEN) | Add tab__chevron button to TabBar.tsx | 5250e79d | TabBar.tsx |
| 2 | Style .tab__chevron + tokenize .tab__context-menu | df4c867f | style.css |
| 3 | Register TAB-04 in TESTING.md | 446364e9 | TESTING.md |

## What Was Built

A `tab__chevron` button (Unicode `▾`, 16×16px) is inserted after the countdown span and before the close button on every session tab (tabs with a truthy `sessionId`). Clicking it opens the existing Rename / Save Terminal As… / Browse files context menu anchored below the chevron using `getBoundingClientRect()` — a dropdown, not a cursor-position popover. Special tabs (Welcome/Settings/Hub/Help/File-browser) render no chevron. Right-click on the tab name continues to open the menu at cursor position, unchanged.

The `.tab__context-menu` CSS was tokenized: all five hardcoded dark hex values (`#1e2030`, `#292e42`, `#a9b1d6`, `#c0caf5`) replaced with `--hub-surface-elevated`, `--hub-border`, `--hub-border-hover`, `--hub-text-secondary`, `--hub-text-primary` so the menu renders correctly in both light and dark themes.

## Acceptance Criteria Verification

- TabBar.tsx contains `className="tab__chevron"`, `aria-label="Session menu"`, `data-testid="tab-chevron"` — verified
- Chevron onClick uses `getBoundingClientRect()` with `rect.left`/`rect.bottom` (not `e.clientX/clientY`) — verified at source level
- Chevron JSX guarded by `tab.sessionId &&`, placed between countdown and `.tab__close` — verified
- Pre-existing `.tab__name` `onContextMenu` handler still uses `e.clientX`/`e.clientY` — verified (D-02 preserved)
- `.tab__context-menu` rules contain zero hardcoded hex values — verified (`grep` returns empty)
- `.tab__chevron` appears in both `@media (prefers-reduced-motion: no-preference)` and `reduce` blocks — verified
- `pnpm build` passes (tsc + vite) — verified
- `pnpm test -- TabBar` exits 0, all 36 tests pass — verified
- TESTING.md §4 TAB-04 row path is `frontend/src/components/__tests__/TabBar.test.tsx` — verified
- `bash tests/check-traceability-paths.sh` exits 0 — verified

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED (test commit) | 720ef346 | PASS — 3 new tests failing before implementation |
| GREEN (feat commit) | 5250e79d | PASS — all 36 tests pass after implementation |

## Deviations from Plan

None — plan executed exactly as written. All 7 decisions (D-01 through D-07) implemented as specified.

## Known Stubs

None — the chevron wires directly to the existing `contextMenu` state and existing menu render block. No placeholder data.

## Threat Flags

No new threat surface introduced. The chevron reuses the existing `contextMenu` state path with the same shape. Position values derive from a DOM `getBoundingClientRect()` on an element this component owns — no user-supplied input rendered. Assessment matches the plan's threat model (T-148-01, T-148-02, T-148-SC: all `accept`).

## Self-Check

### Files exist:
- [x] frontend/src/components/TabBar.tsx
- [x] frontend/src/style.css
- [x] frontend/src/components/__tests__/TabBar.test.tsx
- [x] TESTING.md

### Commits exist:
- [x] 720ef346 — test(148-01): add failing tests for TAB-04 session tab chevron (RED)
- [x] 5250e79d — feat(148-01): add tab__chevron button to session tabs in TabBar.tsx
- [x] df4c867f — feat(148-01): style .tab__chevron and tokenize .tab__context-menu in style.css
- [x] 446364e9 — docs(148-01): register TAB-04 in TESTING.md traceability map

## Self-Check: PASSED
