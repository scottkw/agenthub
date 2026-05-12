---
phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate
plan: 01
status: complete
requirements: [PUI-02]
---

# 99-01 SUMMARY — Plugin toggle banner (PUI-02)

## What was built

Post-save BannerStack one-shot toast for Unicode 11 / Inline Images toggle changes. The banner reuses the `.webgl-recovery-banner` BEM class for visual continuity, auto-dismisses at 6000ms, and stacks both toasts simultaneously when the user toggles BOTH unicode11 and image in the same Save Plugins gesture.

PUI-04 invariant honored: no new save infrastructure was introduced — toggles still flow through the existing three-state Save Plugins button at `PluginsSection.tsx:152-160`. The post-save diff fires only AFTER `SetPluginSettings` resolves successfully.

## Key files

- `frontend/src/components/PluginToggleBanner.tsx` (new) — `PluginToggleBannerProps` component (kind: 'unicode11' | 'image') reusing `.webgl-recovery-banner` BEM class. Auto-dismiss timer (6000ms) clears on unmount or user-click of × button.
- `frontend/src/components/__tests__/PluginToggleBanner.test.tsx` (new) — vitest verbatim-copy assertions, a11y assertion, fake-timer 6000ms auto-dismiss, dismiss button click.
- `frontend/src/components/PluginsSection.tsx` (modified) — adds `lastSavedRef` snapshot + post-save diff comparing `prior.unicode11/image` vs current; emits `onPluginToggleSideEffect(kinds)` callback. Exports `PluginToggleKind` type and `PluginsSectionProps` interface.
- `frontend/src/components/SettingsTab.tsx` (modified) — forwards optional `onPluginToggleSideEffect` prop to `PluginsSection`.
- `frontend/src/App.tsx` (modified) — imports `PluginToggleBanner`, manages `pluginToggleBanners` state set (Array.from(new Set(...)) cap at 2), renders inside the existing `.banner-stack` block alongside other one-shot banners.
- `frontend/src/components/__tests__/PluginsSection.test.tsx` (modified) — adds 6 source-inspection assertions for the diff side-effect path; rebases the existing PUI-01 UI-SPEC order test onto the `renderRow('${kind}'` row literals so the new JSDoc / diff logic don't shift `indexOf` positions out of order.

## Commits

- `84dbb14` feat(99-01): add PluginToggleBanner component + vitest test
- `1b0ff73` feat(99-01): wire PluginToggleBanner into Settings save flow (PUI-02)

## Verification

- `pnpm test -- PluginToggleBanner PluginsSection` → 32/32 pass (2 test files)
- Source inspection: PluginsSection.tsx contains `lastSavedRef`, `prior.unicode11 !== pluginConfig.unicode11`, `prior.image !== pluginConfig.image`, `lastSavedRef.current = pluginConfig`, `kinds.length > 0`, `onPluginToggleSideEffect`.
- App.tsx renders `<PluginToggleBanner>` inside `.banner-stack` only when `pluginToggleBanners.length > 0`, alongside existing banners.
- PUI-04 anti-race contract preserved — only the Save Plugins button surface fires SetPluginSettings; the diff is read-only against `lastSavedRef`.

## Self-Check: PASSED

- All 8 must_haves[truths] from PLAN are satisfied (verified by tests + source inspection).
- Both artifacts (PluginToggleBanner.tsx, PluginToggleBanner.test.tsx) exist with min_lines met.
- No regressions in the 26 pre-existing PluginsSection tests; the rebased order test still enforces the UI-SPEC sequence.

## Notes

The agent executing this plan hit the daily Sonnet rate limit after committing task 1. The orchestrator (Opus) completed task 2 — wiring the integration into PluginsSection / SettingsTab / App.tsx, fixing the PUI-01 ordering test to anchor on row literals (so it survives the new diff JSDoc and diff logic), and committing the integration as `1b0ff73`. Tests pass at 32/32 in the worktree before merge.
