---
phase: 73-theme-usability-audit
plan: 01
status: complete
started: 2026-04-14
completed: 2026-04-14
---

## Summary

Replaced the dynamic theme picker (all 157 xterm-theme keys) with a curated allowlist of 138 themes that pass WCAG-derived readability criteria. Added a localStorage fallback guard for stale theme names and updated tests to verify the new behavior.

## What Was Built

- **frontend/src/themes.ts** — New module exporting `ALLOWED_THEMES`, a sorted array of 138 theme names that passed the contrast audit (fg:bg >= 3.0, cursor:bg >= 2.0, at most 3 important ANSI colors below 2.5)
- **frontend/src/components/SettingsTab.tsx** — Theme picker now uses `ALLOWED_THEMES` import instead of `Object.keys(xtermThemes).sort()`, removed direct xterm-theme dependency
- **frontend/src/App.tsx** — localStorage fallback guard validates stored theme against allowlist before use; stale/removed themes silently fall back to Tomorrow_Night
- **frontend/src/components/__tests__/SettingsTab.test.tsx** — Updated THM-01 assertions for new import pattern, added THM-04 describe block with 5 assertions, added App.tsx fallback guard tests

## Key Decisions

- Moved ALLOWED_THEMES to a separate `themes.ts` module (not inline in SettingsTab) for reusability and cleaner imports
- Removed `import * as xtermThemes from 'xterm-theme'` from SettingsTab.tsx since it only needs the name list, not the theme objects
- Tests use `?raw` imports on both `themes.ts` and `App.tsx` to verify source-level contracts

## Deviations

- Test file needed an additional `themesRaw` import from `themes.ts?raw` since the THM-04 assertions check theme name presence in the allowlist source, which lives in themes.ts (not SettingsTab.tsx as originally planned)
- THM-01 "imports xterm-theme library" assertion updated to "imports ALLOWED_THEMES from themes module" since SettingsTab no longer imports xterm-theme directly

## Self-Check: PASSED

### key-files
- created:
  - frontend/src/themes.ts
- modified:
  - frontend/src/components/SettingsTab.tsx
  - frontend/src/App.tsx
  - frontend/src/components/__tests__/SettingsTab.test.tsx

### verification
- 377/377 tests pass (19 test files)
- 138 theme entries in allowlist (verified via grep count)
- No removed themes present in allowlist
- No Object.keys(xtermThemes) in SettingsTab.tsx
- localStorage fallback guard present in App.tsx
