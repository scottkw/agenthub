---
phase: 104
plan: settings-hyperlinked-index
subsystem: frontend/settings
tags: [ui, settings, navigation, react]
requires: []
provides: [settings-jump-bar, settings-search, settings-section-anchors]
affects:
  - frontend/src/components/SettingsTab.tsx
  - frontend/src/components/PluginsSection.tsx
  - frontend/src/style.css
tech-stack:
  added: []
  patterns: [browser-hash-navigation, native-scrollIntoView, case-insensitive-substring-filter]
key-files:
  created:
    - frontend/src/components/SettingsJumpBar.tsx
    - frontend/src/components/SettingsSearch.tsx
    - frontend/src/components/__tests__/SettingsTab.hyperlinked-index.test.tsx
  modified:
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/components/PluginsSection.tsx
    - frontend/src/style.css
    - frontend/src/components/__tests__/SettingsTab.test.tsx
    - frontend/src/components/__tests__/SettingsTab.start-minimized.test.tsx
decisions:
  - "Use native anchor links (a href=#...) instead of JS click handlers — gives browser-managed smooth scroll + scroll-margin-top out of the box."
  - "Static SEARCH_INDEX array (not DOM scraping via data-setting-label) — simpler, decoupled from render-time presence of conditional toggles."
  - "Search index excludes deep plugin sub-options per spec — keeps coupling between SettingsSearch and PluginsSection internals at zero."
metrics:
  duration: ~8min
  completed: 2026-05-12
  tasks: 2
  commits: 2
---

# Phase 104: Settings Hyperlinked Index Summary

Adds a sticky jump-link bar and an autocomplete search to the Settings tab so users can navigate the now-tall single-page Settings view without manual scrolling. Implements SETUI-01 (jump bar), SETUI-02 (smooth scroll on click), and SETUI-03 (label search with anchor jump).

## What Changed

### New components

- **`SettingsJumpBar.tsx`** — Sticky `<nav>` rendered at the top of the Settings body. Renders 7 `<a href="#settings-{slug}">` anchors (Plugins, Behavior, Session Behavior, Appearance, Web Server, Security, Paths). Exports a `SETTINGS_JUMP_LINKS` constant so the search index can re-use the same canonical labels and slugs.
- **`SettingsSearch.tsx`** — Controlled `<input type="search">` with an absolutely-positioned dropdown of matching results. Filtering is plain case-insensitive substring matching over a static `SEARCH_INDEX` array. Clicking a result calls `el.scrollIntoView({ behavior: 'smooth' })` plus `history.replaceState` so the URL hash stays in sync; clears the input afterwards so the dropdown closes.

### Section anchor IDs

Each of the 7 settings section headers now carries an `id` attribute matching the JumpBar slug:

| Section | id |
| --- | --- |
| Plugins | `settings-plugins` |
| Behavior | `settings-behavior` |
| Session Behavior | `settings-session-behavior` |
| Appearance | `settings-appearance` |
| Web Server | `settings-web-server` |
| Security | `settings-security` |
| Paths | `settings-paths` |

### CSS additions (`frontend/src/style.css`)

- `.settings-panel__body h3 { scroll-margin-top: 80px }` — keeps anchored headers below the sticky bar.
- `.settings-panel__body { scroll-behavior: smooth }` — enables native smooth scrolling for hash anchors.
- `.settings-jump-bar` — flex row, `position: sticky; top: 0; z-index: 5`, 16px gap between links, matching TokyoNight palette (`#1a1b26` background, `#7aa2f7` links, `#292e42` hover).
- `.settings-search` + `.settings-search__input` + `.settings-search__results` + `.settings-search__result` — input max-width 320px, dropdown anchored to the input with shadow + 240px max-height + scroll.

### SettingsTab.tsx integration

Imports both components and renders them at the top of `.settings-panel__body`, before the first section header (which is now Behavior, since Plugins is rendered last in the existing order).

## Tests

`SettingsTab.hyperlinked-index.test.tsx` — 19 new test cases:

- 7 source-inspection tests confirming each section h3 has the correct `id` attribute (mix of `SettingsTab.tsx` and `PluginsSection.tsx`).
- 1 CSS test confirming `scroll-margin-top` is set on `.settings-panel__body h3`.
- 1 CSS test confirming `.settings-jump-bar` uses `position: sticky`.
- 4 render tests for `SettingsJumpBar` (export shape, 7 links, correct hrefs, position-sticky).
- 4 render tests for `SettingsSearch` (export shape, input placeholder, substring filtering, data-target on results).
- 4 source-inspection tests confirming `SettingsTab.tsx` imports + mounts both components.

Two existing tests that hard-coded `<h3>Foo</h3>` substrings (`SettingsTab.test.tsx`, `SettingsTab.start-minimized.test.tsx`) were updated to use `/<h3[^>]*>Foo<\/h3>/` regex matches, since the headers now carry an `id` attribute.

## Verification

- `pnpm test` — 801 passing, 20 failing. The 20 failures are pre-existing `Sidebar.test.tsx` failures unrelated to this work (baseline before the phase was 782 passing / 20 failing — net +19 passing, no regressions).
- `npx tsc --noEmit` — clean.
- `go build ./...` — clean.

## Commits

| Commit | Description |
| --- | --- |
| `b0183e8` | test(104-1): add failing tests for Settings hyperlinked index (SETUI-01..03) |
| `e3d5201` | feat(104-2): implement Settings hyperlinked index (SETUI-01..03) |

## Deviations from Plan

**[Rule 1 — Test maintenance] Updated existing literal-string assertions in `SettingsTab.test.tsx` and `SettingsTab.start-minimized.test.tsx`.** Those tests asserted `<h3>Behavior</h3>` etc. via `toContain`; adding the `id` attribute to each h3 broke the literal substring match without breaking the rendered behaviour. Switched the assertions to regex (`/<h3[^>]*>Behavior<\/h3>/`) so they remain robust to additional attributes. No behavioural change to those tests; they still pin the header presence.

## Known Stubs

None. The search index is a static array (intentional per spec) — not a stub.

## Threat Flags

None. No new network endpoints, auth paths, file access, or trust-boundary schema changes.

## Self-Check: PASSED

- frontend/src/components/SettingsJumpBar.tsx — FOUND
- frontend/src/components/SettingsSearch.tsx — FOUND
- frontend/src/components/__tests__/SettingsTab.hyperlinked-index.test.tsx — FOUND
- Commit b0183e8 — FOUND in git log
- Commit e3d5201 — FOUND in git log
- All 19 new tests pass; full suite has only pre-existing Sidebar failures.
