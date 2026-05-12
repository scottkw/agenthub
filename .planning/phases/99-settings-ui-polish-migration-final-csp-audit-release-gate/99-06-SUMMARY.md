---
phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate
plan: "06"
subsystem: frontend/settings-plugins
tags: [gap-closure, css-fix, disclosure-checkboxes, render-test, v3.2-release-gate]
dependency_graph:
  requires: [99-02]
  provides: [visible-disclosure-checkboxes, disclosure-render-test]
  affects: [frontend/src/components/PluginsSection.tsx, frontend/src/components/__tests__/PluginsSection.disclosure.render.test.tsx]
tech_stack:
  added: []
  patterns: [jsdom-createRoot-render-test, vi.mock-wails-bindings]
key_files:
  modified:
    - frontend/src/components/PluginsSection.tsx
  created:
    - frontend/src/components/__tests__/PluginsSection.disclosure.render.test.tsx
decisions:
  - "Option (a) selected: drop className from disclosure checkboxes (not option b: add track/thumb spans, not option c: CSS override). Smallest possible diff — native checkboxes in config <details> blocks are appropriate UX."
  - "Real-DOM render test uses createRoot + jsdom (NOT @testing-library/react) per project convention established in TerminalPanel.web-links.test.tsx"
metrics:
  duration: "~4 minutes"
  completed: "2026-05-12T05:05:33Z"
  tasks_completed: 2
  files_changed: 2
---

# Phase 99 Plan 06: Disclosure Checkbox Visibility Fix Summary

Option (a) applied — drop `className="settings-panel__toggle-input"` from 6 disclosure inputs; disclosure checkboxes now fall back to native browser checkbox rendering, closing UAT Tests 5 and 6.

## What Was Done

### Root Cause (from `.planning/debug/99-disclosure-checkboxes-missing.md`)

The 6 disclosure checkbox inputs in `PluginsSection.tsx` reused `className="settings-panel__toggle-input"`, which the Phase-82 CSS rule at `style.css:586-592` hides globally (`position:absolute; width:1px; height:1px; opacity:0; pointer-events:none`). The main plugin rows pair this hidden input with visible `.settings-panel__toggle-track`/`.settings-panel__toggle-thumb` spans — the disclosure helpers copied the class but omitted those spans, making the checkboxes invisible. The `<select>` and `<input type="number">` in the same disclosures rendered correctly because they never carried the hidden-toggle class.

### Fix Applied (Option a — RECOMMENDED)

Deleted the `className="settings-panel__toggle-input"` attribute from all 6 disclosure checkbox inputs. No other attributes changed.

**6 lines deleted from `PluginsSection.tsx`:**

```
renderSearchDisclosure — regex input (was line 175):
-            className="settings-panel__toggle-input"

renderSearchDisclosure — caseSensitive input (was line 185):
-            className="settings-panel__toggle-input"

renderSearchDisclosure — wholeWord input (was line 195):
-            className="settings-panel__toggle-input"

renderWebLinksDisclosure — confirmOSC8 input (was line 229):
-            className="settings-panel__toggle-input"

renderWebLinksDisclosure — confirmIDN input (was line 238):
-            className="settings-panel__toggle-input"

renderWebLinksDisclosure — confirmTyposquat input (was line 247):
-            className="settings-panel__toggle-input"
```

**Post-patch verification:**
- `grep -c 'settings-panel__toggle-input' frontend/src/components/PluginsSection.tsx` → **1** (down from 7)
- The remaining occurrence at line ~143 (`renderRow` hidden anchor) is correct and intentional — it is paired with visible `.settings-panel__toggle-track`/`.settings-panel__toggle-thumb` spans

### PUI-04 Anti-Race Contract Preserved

- `grep -c 'SetPluginSettings' frontend/src/components/PluginsSection.tsx` → **2** (1 import + 1 call inside handleSavePlugins)
- No new `SetPluginSettings` call path introduced

## New Test File

**`frontend/src/components/__tests__/PluginsSection.disclosure.render.test.tsx`** (128 lines)

Three `it` blocks:

1. `renders six <input type="checkbox"> elements inside .settings-panel__details blocks` — asserts count === 6
2. `NONE of the disclosure checkboxes carry the .settings-panel__toggle-input hidden-anchor class` — load-bearing assertion; closes the test-strategy gap
3. `the renderRow main toggles (outside .settings-panel__details) DO carry .settings-panel__toggle-input` — differential guardrail; asserts count === 8

**Mock strategy:** `vi.mock('../../wailsjs/go/main/App', ...)` and `vi.mock('../../wailsjs/go/models', ...)` — stub `GetPluginSettings` to resolve a plain-object snapshot with all 8 booleans + 3 sub-configs; stub `daemon.*` constructors as plain `class Stub { constructor(o={}) { Object.assign(this, o) } }`. Required because Wails-generated constructors fail under jsdom.

**Mount strategy:** `createRoot(container).render(<PluginsSection />)` + `await act(async () => { await new Promise(r => setTimeout(r, 0)) })` to flush the `GetPluginSettings` promise. Forces `<details open>` before querying.

## RED → GREEN Evidence

### RED (pre-patch source, 7 toggle-input occurrences)

```
 ❯ src/components/__tests__/PluginsSection.disclosure.render.test.tsx (3 tests | 1 failed) 45ms
     × NONE of the disclosure checkboxes carry the .settings-panel__toggle-input hidden-anchor class 11ms

─────────────── Failed Tests 1 ───────────────

 FAIL  src/components/__tests__/PluginsSection.disclosure.render.test.tsx > Phase 99 gap-closure: disclosure checkboxes render with class hygiene > NONE of the disclosure checkboxes carry the .settings-panel__toggle-input hidden-anchor class
AssertionError: expected false to be true // Object.is equality

- Expected
+ Received

- true
+ false

  ❯ src/components/__tests__/PluginsSection.disclosure.render.test.tsx:113:22

 Test Files  1 failed (1)
      Tests  1 failed | 2 passed (3)
```

### GREEN (post-patch source, 1 toggle-input occurrence)

```
 Test Files  1 passed (1)
      Tests  3 passed (3)
   Duration  599ms
```

## Test Results

### After Task 1 + Task 2

**PluginsSection tests (all 3 files):**
```
 Test Files  3 passed (3)
      Tests  36 passed (36)
```

- `PluginsSection.test.tsx` — 24 tests (unchanged, all pass)
- `PluginsSection.disclosure.test.tsx` — 9 source-inspection tests (all pass, including PUI-04 SetPluginSettings count=2)
- `PluginsSection.disclosure.render.test.tsx` — 3 new render tests (all pass)

**Full suite:** 782 tests pass across 52 files. `Sidebar.test.tsx` shows 20 pre-existing failures (TypeError: Cannot read properties of undefined (reading 'unmount') + localStorage not available) that are unrelated to this plan and present in the base commit `f518197`. These are out of scope per deviation rules and logged to deferred items.

## Deviations from Plan

None — plan executed exactly as written. Option (a) applied as specified. Test file uses createRoot + vi.mock per the specified constraints.

## Pre-Existing Test Failures (Out of Scope)

`Sidebar.test.tsx` — 20 failures pre-existing in base commit `f518197`. Root cause: `localStorage` not available (Node.js experimental warning) and `root` undefined in `afterEach`. Not caused by this plan. Logged here per scope-boundary rule.

## Commits

| Task | Commit | Files | Description |
|------|--------|-------|-------------|
| 1 | `247a7b4` | `PluginsSection.tsx` | Drop 6 hidden-toggle className attributes from disclosure inputs |
| 2 | `6cfe32d` | `PluginsSection.disclosure.render.test.tsx` | Add 128-line real-DOM render test (3 it blocks, RED→GREEN verified) |

## Key Decisions

- **Option (a) over (b) and (c):** Smallest diff (6 lines deleted); native checkboxes in config `<details>` blocks are appropriate UX — they are not status indicators like the main iOS-pill toggles, so the pill visual is unnecessary. Option (b) would add markup complexity; option (c) CSS override was explicitly rejected by the plan.
- **`createRoot` over `@testing-library/react`:** Project convention documented in `TerminalPanel.web-links.test.tsx` — `@testing-library/react` is intentionally not in devDependencies.

## References

- Root cause record: `.planning/debug/99-disclosure-checkboxes-missing.md`
- UAT gaps closed: 99-UAT.md Tests 5 and 6 (Search disclosure checkboxes, Web Links disclosure checkboxes)

## Next Action

Re-run manual UAT Tests 5 and 6 in `99-UAT.md` after `wails build -tags wailsassets`. Confirm VISIBLE native checkboxes in both Search and Web Links disclosures. Flip both tests from `issue` to `pass`. V3.2 release gate unblocked.

## Known Stubs

None.

## Threat Flags

None — this plan only removes a CSS class attribute from 6 existing checkbox inputs and adds a test file. No new network endpoints, auth paths, file access patterns, or schema changes.

## Self-Check: PASSED

- `frontend/src/components/PluginsSection.tsx` — modified (6 deletions) FOUND
- `frontend/src/components/__tests__/PluginsSection.disclosure.render.test.tsx` — created (128 lines) FOUND
- Commit `247a7b4` — FOUND
- Commit `6cfe32d` — FOUND
- `grep -c 'settings-panel__toggle-input' frontend/src/components/PluginsSection.tsx` → 1 VERIFIED
- `grep -c 'SetPluginSettings' frontend/src/components/PluginsSection.tsx` → 2 VERIFIED
- PluginsSection tests (36/36) PASSED
