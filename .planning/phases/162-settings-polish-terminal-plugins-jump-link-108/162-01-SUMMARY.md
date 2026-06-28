---
phase: 162-settings-polish-terminal-plugins-jump-link-108
plan: "01"
subsystem: frontend/settings-ui
tags: [settings, jump-bar, plugins, label-rename, bug-fix]
dependency_graph:
  requires: []
  provides: [SETTINGS-UI-01]
  affects: [SettingsJumpBar, PluginsSection, SettingsSearch]
tech_stack:
  added: []
  patterns: [shared-constant-propagation]
key_files:
  created: []
  modified:
    - frontend/src/components/SettingsJumpBar.tsx
    - frontend/src/components/PluginsSection.tsx
    - frontend/src/components/__tests__/SettingsTab.hyperlinked-index.test.tsx
decisions:
  - "Terminal Plugins jump link moved to last in SETTINGS_JUMP_LINKS — render-order-matching instead of top-listing"
  - "id settings-plugins preserved byte-for-byte on both jump-link entry and section h3"
  - "SettingsSearch.tsx not modified — SEARCH_INDEX spreads SETTINGS_JUMP_LINKS so label propagates automatically"
  - "TESTING.md left unchanged — no stale traceability rows for these files"
metrics:
  duration: "3 minutes"
  completed: "2026-06-28"
  tasks: 3
  files: 3
status: complete
requirements:
  - SETTINGS-UI-01
---

# Phase 162 Plan 01: Settings Polish — Terminal Plugins Jump Link Summary

**One-liner:** Moved the "Plugins" jump link to last position and renamed it "Terminal Plugins" in both the sticky jump bar and the section header, fixing GitHub issue #108 with a single shared-constant change that propagates to both GUI and web-share surfaces.

## Objective

Resolve GitHub issue #108: the Settings "Plugins" jump link was listed FIRST in the sticky jump bar while its section rendered LAST, and the label "Plugins" was ambiguous. This plan reordered the link to match on-screen render order, renamed it to "Terminal Plugins", kept the `settings-plugins` anchor id stable, and updated the test fixture to match.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Move Plugins jump link to last + rename to "Terminal Plugins" in SettingsJumpBar.tsx | 20de5475 | frontend/src/components/SettingsJumpBar.tsx |
| 2 | Rename plugins section header to "Terminal Plugins" in PluginsSection.tsx | 39b3e9e3 | frontend/src/components/PluginsSection.tsx |
| 3 | Update hyperlinked-index test fixture and run full frontend gate | e5310d0b | frontend/src/components/__tests__/SettingsTab.hyperlinked-index.test.tsx |

## Key Changes

### SettingsJumpBar.tsx
- `SETTINGS_JUMP_LINKS`: plugins entry moved from index 0 (first) to index 6 (last); label changed from `'Plugins'` to `'Terminal Plugins'`; `id` unchanged (`settings-plugins`)
- Array final order: Behavior, Session Behavior, Appearance, Web Server, Security, Paths, Terminal Plugins
- Block comment updated: now states bar order mirrors on-screen render order; stale "listed first" prose removed

### PluginsSection.tsx
- `<h3 id="settings-plugins">Plugins</h3>` → `<h3 id="settings-plugins">Terminal Plugins</h3>`
- id attribute, Save Plugins button copy, and toggle-row order all unchanged

### SettingsTab.hyperlinked-index.test.tsx
- `sections` fixture plugins entry: `label: 'Plugins'` → `label: 'Terminal Plugins'`
- `id`, `file`, `raw` fields unchanged; `expectedTargets` array unchanged (set-membership check, order-agnostic)

## Verification Results

- Jump-bar order: `settings-plugins` is the LAST id; label is `'Terminal Plugins'` — PASS
- Section header: `<h3 id="settings-plugins">Terminal Plugins</h3>` — PASS
- Anchor integrity: jump-link `href="#settings-plugins"` still targets section h3 `id="settings-plugins"` — PASS
- Cross-surface parity: SettingsSearch derives from SETTINGS_JUMP_LINKS (no separate edit) — PASS
- `cd frontend && pnpm build` (tsc + vite build) — PASS (exit 0)
- `cd frontend && pnpm test` (vitest run) — PASS (2170 tests, 132 files, 0 failures)
- `bash tests/check-traceability-paths.sh` — PASS (exit 0)
- TESTING.md: unchanged — grep found no rows referencing these file paths; no stale rows

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all label changes are wired to the live UI; no placeholder text or empty data sources.

## Threat Flags

None — this is a static client-side label and array-order edit. No new trust boundary, data flow, auth surface, network call, or persisted state introduced.

## Self-Check: PASSED

- frontend/src/components/SettingsJumpBar.tsx — FOUND
- frontend/src/components/PluginsSection.tsx — FOUND
- frontend/src/components/__tests__/SettingsTab.hyperlinked-index.test.tsx — FOUND
- Commit 20de5475 — FOUND
- Commit 39b3e9e3 — FOUND
- Commit e5310d0b — FOUND
