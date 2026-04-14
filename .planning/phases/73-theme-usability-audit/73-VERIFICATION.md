---
phase: 73-theme-usability-audit
verified: 2026-04-14T13:27:00Z
status: human_needed
score: 6/6
overrides_applied: 0
human_verification:
  - test: "Open the app, navigate to Settings > Appearance. Count the themes visible in the theme picker dropdown."
    expected: "Exactly 138 themes listed — fewer than before (was 157). No 'default' entry visible."
    why_human: "Picker contents depend on running app UI — can verify count programmatically but visual confirmation of no namespace artifact is best done in app."
  - test: "In the theme picker, select 'Novel'. Open a terminal session and run a command (e.g., ls). Verify foreground text is clearly readable against the light background."
    expected: "Terminal text is readable. The light theme displays without contrast issues."
    why_human: "Visual readability across terminal rendering cannot be confirmed by static analysis — requires rendered terminal output."
  - test: "Set localStorage key 'agenthub:terminalTheme' to a removed theme name (e.g., 'Github') via browser devtools, then reload the app."
    expected: "App silently falls back to Tomorrow_Night — no error, no removed theme loaded."
    why_human: "localStorage fallback behavior is runtime state that depends on the stored value being absent from ALLOWED_THEMES at startup."
---

# Phase 73: Theme Usability Audit — Verification Report

**Phase Goal:** The theme picker only offers themes that produce readable, usable terminal output across all four supported agents
**Verified:** 2026-04-14T13:27:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Theme picker shows exactly 138 themes (not 157) | VERIFIED | `grep -c '"' frontend/src/themes.ts` = 138. ALLOWED_THEMES array has 138 entries. |
| 2 | No 'default' namespace artifact appears in the picker | VERIFIED | `grep '"default"' frontend/src/themes.ts` returns no matches. ALLOWED_THEMES does not contain "default". |
| 3 | At least one light-background theme (Novel, Piatto_Light, Solarized_Light, Violet_Light) is available | VERIFIED | All four confirmed present in themes.ts lines 84, 94, 111, 135. |
| 4 | At least one dark-background theme (e.g., Dracula, Tomorrow_Night) is available | VERIFIED | Both confirmed present in themes.ts lines 30, 123. |
| 5 | User with a previously-stored removed theme sees Tomorrow_Night on next launch | VERIFIED | App.tsx lines 88-93: `ALLOWED_THEMES.includes(stored)` guard — falls back to DEFAULT_THEME_NAME ('Tomorrow_Night') when stored theme is absent from allowlist. |
| 6 | Tomorrow_Night remains the default theme | VERIFIED | App.tsx line 39: `DEFAULT_THEME_NAME = 'Tomorrow_Night'`. themes.ts line 123 confirms it is in the allowlist. |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/themes.ts` | ALLOWED_THEMES constant — 138-entry sorted string array | VERIFIED | 144 lines, exports `ALLOWED_THEMES: string[]` with 138 entries. No banned themes. |
| `frontend/src/components/SettingsTab.tsx` | Theme picker using ALLOWED_THEMES instead of Object.keys(xtermThemes) | VERIFIED | Line 2: imports ALLOWED_THEMES. Line 22: `const THEME_NAMES = ALLOWED_THEMES`. No Object.keys(xtermThemes) reference. |
| `frontend/src/App.tsx` | localStorage fallback guard using ALLOWED_THEMES.includes() | VERIFIED | Line 35: imports ALLOWED_THEMES. Lines 90-91: `ALLOWED_THEMES.includes(stored)` and `ALLOWED_THEMES.includes(DEFAULT_THEME_NAME)`. |
| `frontend/src/components/__tests__/SettingsTab.test.tsx` | THM-04 test assertions for allowlist behavior | VERIFIED | Lines 212-246: full THM-04 describe block with 5 allowlist assertions + 2 localStorage fallback guard assertions. `themesRaw` import from themes.ts confirmed. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `frontend/src/themes.ts` | `frontend/src/components/SettingsTab.tsx` | `import { ALLOWED_THEMES }` | WIRED | Line 2 of SettingsTab.tsx: `import { ALLOWED_THEMES } from '../themes'` — confirmed present and used at line 22 |
| `frontend/src/themes.ts` | `frontend/src/App.tsx` | `import { ALLOWED_THEMES }` | WIRED | Line 35 of App.tsx: `import { ALLOWED_THEMES } from './themes'` — used at lines 90-91 in useState initializer |
| `frontend/src/components/__tests__/SettingsTab.test.tsx` | `frontend/src/components/SettingsTab.tsx` | `?raw` import for source inspection | WIRED | Line 2 of test: `import raw from '../../components/SettingsTab.tsx?raw'` — used in all source-inspection assertions |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| `SettingsTab.tsx` | `THEME_NAMES` | `ALLOWED_THEMES` imported from `themes.ts` | Yes — 138-entry static allowlist (intentional: static is the design) | FLOWING |
| `App.tsx` | `terminalThemeName` | localStorage + ALLOWED_THEMES guard | Yes — reads actual stored value, falls back to allowlist member | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| themes.ts exports ALLOWED_THEMES | `grep -c '"' frontend/src/themes.ts` | 138 | PASS |
| Banned themes absent from allowlist | `grep '"default"\|"C64"\|"Github"...' themes.ts` | No matches | PASS |
| Object.keys(xtermThemes) removed from SettingsTab | `grep 'Object.keys' SettingsTab.tsx` | No matches | PASS |
| localStorage fallback guard present | `grep -n 'ALLOWED_THEMES.includes' App.tsx` | Lines 90-91 | PASS |
| All THM-04 tests pass | `pnpm test -- SettingsTab` | 73/73 passed | PASS |
| Full test suite — no regressions | `pnpm test` | 377/377 passed (19 files) | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| THM-04 | 73-01-PLAN.md | Theme picker only lists xterm themes that render readable foreground text, cursor, and ANSI colors across all 4 supported CLIs | SATISFIED (programmatic) / NEEDS HUMAN (visual) | ALLOWED_THEMES constant implements the filtering; 19 themes removed; THM-04 test block passes. Visual readability across 4 agents requires human confirmation. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `SettingsTab.tsx` | 404, 436 | `placeholder=` HTML attribute | Info | Input field placeholder text — not a stub. Standard HTML attribute for path input UX. |

No blockers or warnings found. The `placeholder=` matches are HTML `<input placeholder="">` attributes used for UX guidance in form fields — not stub indicators.

### Human Verification Required

#### 1. Theme picker count and "default" exclusion

**Test:** Open the app, navigate to Settings > Appearance. Open the Terminal Theme dropdown and count items (or scroll to confirm no "default" entry at top).
**Expected:** 138 themes visible, sorted alphabetically, no "default" entry.
**Why human:** Theme picker rendering is a UI behavior that involves the running app. Static analysis confirms the allowlist is correctly wired; visual confirmation checks that the select element renders as expected.

#### 2. Light-background theme readability in terminal

**Test:** Select "Novel" from the theme picker. Open a new terminal session (any CLI). Run a command and observe output.
**Expected:** Terminal foreground text is clearly readable against the light background. No invisible or low-contrast text.
**Why human:** Terminal text readability is a visual quality judgment that cannot be confirmed by static code analysis. The contrast ratios were validated in 73-RESEARCH.md but runtime rendering may differ.

#### 3. localStorage fallback for removed themes

**Test:** Open browser devtools for the Wails app. Set `localStorage.setItem('agenthub:terminalTheme', 'Github')` (a removed theme). Reload the app. Check which theme is active in Settings > Appearance.
**Expected:** Theme picker shows "Tomorrow_Night" selected — the removed theme name was silently discarded and replaced with the default.
**Why human:** Requires runtime manipulation of localStorage state. The code path is verified to exist (lines 89-93 of App.tsx) but end-to-end runtime behavior requires a running app.

### Gaps Summary

No gaps found. All 6 must-have truths are verified, all artifacts exist and are substantive, all key links are wired, all key data flows through correctly, and the full test suite (377 tests) passes with no regressions.

Three items are routed to human verification because they involve runtime behavior, visual rendering quality, and localStorage state manipulation that cannot be confirmed by static code analysis. These are quality confirmation items, not blockers — the code implementation is correct.

---

_Verified: 2026-04-14T13:27:00Z_
_Verifier: Claude (gsd-verifier)_
