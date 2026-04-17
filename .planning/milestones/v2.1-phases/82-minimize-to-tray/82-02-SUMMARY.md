---
phase: 82-minimize-to-tray
plan: "02"
subsystem: frontend-settings
tags: [react, tsx, css, toggle, settings, tray]
dependency_graph:
  requires: [startMinimized-wails-bindings]
  provides: [startMinimized-ui, behavior-section, toggle-css]
  affects: [frontend/src/components/SettingsTab.tsx, frontend/src/style.css]
tech_stack:
  added: []
  patterns: [non-optimistic-toggle, toggleLoaded-flash-gate, inline-error-below-description]
key_files:
  created: []
  modified:
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/style.css
decisions:
  - "Toggle uses non-optimistic state: setStartMinimized only called after await SetStartMinimized succeeds — state reverts automatically on failure"
  - "toggleLoaded gate prevents flash from off to on: label only renders after GetStartMinimized resolves"
  - "Behavior section is the first h3 child in .settings-panel__body, inheriting h3:first-child rule (no border-top, no margin-top)"
metrics:
  duration: "~10 min"
  completed: "2026-04-17T12:59:27Z"
  tasks_completed: 2
  files_modified: 2
---

# Phase 82 Plan 02: Behavior Section with Start-Minimized Toggle

## One-liner

Settings tab Behavior section with a CSS toggle switch wired to GetStartMinimized/SetStartMinimized Wails bindings, with flash-free load, non-optimistic save, and inline error display.

## What Was Built

### Task 1: Toggle CSS rules (commit 6f75352)

Added 7 CSS rule blocks to `frontend/src/style.css` after the last `.settings-panel__*` rule, before the New Session Modal section:

- `.settings-panel__toggle-input` — visually hidden checkbox (`position: absolute`, `opacity: 0`, `pointer-events: none`) for accessibility
- `.settings-panel__toggle-row` — flex container (`display: flex`, `align-items: center`, `gap: 10px`, `min-height: 44px`, `user-select: none`)
- `.settings-panel__toggle-track` — 36×20px pill with `#16161e` background and `#292e42` border, smooth `background-color`/`border-color` transition
- `.settings-panel__toggle-thumb` — 14×14px circle at `#565f89`, smooth `transform`/`background-color` transition
- `.settings-panel__toggle-row--checked .settings-panel__toggle-track` — track turns `#7aa2f7` when checked
- `.settings-panel__toggle-row--checked .settings-panel__toggle-thumb` — thumb moves `translateX(16px)` and turns `#1a1b26` when checked
- `.settings-panel__toggle-label` — 13px, weight 400, `#c0caf5` text

All color, size, transition, and spacing values match UI-SPEC exactly.

### Task 2: Behavior section in SettingsTab (commit 6e9a325)

Modified `frontend/src/components/SettingsTab.tsx`:

**Imports:** Added `GetStartMinimized` and `SetStartMinimized` to the Wails import block.

**State (4 new variables):**
- `startMinimized` / `setStartMinimized` — persisted preference value
- `toggleLoaded` / `setToggleLoaded` — flash-prevention gate (false until GetStartMinimized resolves)
- `toggleSaving` / `setToggleSaving` — loading state for save in progress
- `toggleError` / `setToggleError` — inline error display on save failure

**useEffect:** Calls `GetStartMinimized().then(val => { setStartMinimized(val); setToggleLoaded(true) }).catch(() => setToggleLoaded(true))` on mount. Both success and error paths set `toggleLoaded(true)` so the toggle always appears.

**handleToggleMinimized:** Non-optimistic handler — `await SetStartMinimized(next)` before `setStartMinimized(next)`. On error, state is not updated (toggle reverts). Loading state via `setToggleSaving`. Error message prefixed with `Could not save preference —`.

**JSX structure:** Behavior section inserted before Appearance section as the first `<h3>` in `.settings-panel__body`, ensuring `h3:first-child` CSS rule applies (no top border/margin):
- `<h3>Behavior</h3>` as first child
- `{toggleLoaded && <label ...>}` gate prevents flash
- `--checked` class applied to label based on `startMinimized` state
- Hidden `<input type="checkbox" id="startMinimized">` for keyboard/screen-reader accessibility
- Description: "When enabled, AgentHub launches with the window hidden. Click the tray icon to open it."
- `{toggleError && <p className="settings-panel__error">}` for error display

## Verification Results

```
grep -c "settings-panel__toggle-track" style.css  → 2 (rule + --checked override)
grep -c "settings-panel__toggle-thumb" style.css  → 2 (rule + --checked override)
grep -c "settings-panel__toggle-row--checked" style.css → 2
grep -c "settings-panel__toggle-input" style.css  → 1
grep -c "settings-panel__toggle" style.css        → 7
grep "Behavior\|Appearance" SettingsTab.tsx        → Behavior at line 277, Appearance at line 305
grep "await SetStartMinimized\|setStartMinimized"  → await at 243, set at 244 (non-optimistic)
```

## Deviations from Plan

None — plan executed exactly as written. All 5 code changes (imports, state, useEffect, handler, JSX) applied in sequence per the plan specification.

## Known Stubs

None — toggle reads from and writes to the daemon via real Wails bindings created in Plan 01. No hardcoded or mock values.

## Threat Flags

None — no new trust boundaries introduced. Toggle sends only a boolean to the Wails layer; no user-supplied strings. T-82-05 (DoS via rapid re-clicks) mitigated by `pointerEvents: none` during save.

## Self-Check: PASSED

- `frontend/src/components/SettingsTab.tsx` — FOUND, contains GetStartMinimized, SetStartMinimized, toggleLoaded, Behavior section, Start minimized to system tray
- `frontend/src/style.css` — FOUND, contains settings-panel__toggle-track, settings-panel__toggle-thumb, settings-panel__toggle-row--checked, settings-panel__toggle-input, settings-panel__toggle-label
- Commit 6f75352 — FOUND (feat(82-02): add toggle CSS rules)
- Commit 6e9a325 — FOUND (feat(82-02): add Behavior section)
