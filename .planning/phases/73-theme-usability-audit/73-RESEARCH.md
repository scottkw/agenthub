# Phase 73: Theme Usability Audit - Research

**Researched:** 2026-04-14
**Domain:** xterm.js theme palette contrast analysis + React frontend allowlist implementation
**Confidence:** HIGH

## Summary

Phase 73 removes unreadable themes from the xterm-theme picker. The current `SettingsTab.tsx` exposes all 157 entries from `xterm-theme@1.1.0` — including one non-palette artifact (`default`, which is the namespace re-export of the whole module). The requirement is to replace the dynamic `Object.keys(xtermThemes).sort()` derivation with a hardcoded allowlist of only those themes that pass a measurable readability standard.

All 157 themes were analyzed in this research session using WCAG-derived contrast ratios computed directly from each theme's palette data in `frontend/node_modules/xterm-theme/index.js`. The analysis applies three criteria: foreground-to-background contrast ratio ≥ 3.0, cursor-to-background contrast ratio ≥ 2.0, and at most 3 important ANSI colors (red, green, yellow, blue, cyan, white, brightWhite, brightGreen, brightYellow, brightCyan, brightBlue) below 2.5 contrast against the background. 138 themes pass; 19 fail. Four of the passing themes are light-background, satisfying the requirement for at least one light option.

The implementation is a targeted, low-risk code change: replace one constant in `SettingsTab.tsx`, add a fallback guard for stale `localStorage` values, and update the existing `SettingsTab.test.tsx` to assert the new shape. No backend changes, no new dependencies, no design system changes.

**Primary recommendation:** Define `ALLOWED_THEMES` as a hardcoded sorted array of 138 theme names in `SettingsTab.tsx` (or a co-located `themes.ts`). Replace `THEME_NAMES = Object.keys(xtermThemes).sort()` with `THEME_NAMES = ALLOWED_THEMES`. Add a one-line localStorage fallback. Update tests.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| THM-04 | The theme picker only lists xterm themes that render readable foreground text, cursor, and ANSI colors across all 4 supported CLIs (Claude Code, Codex, Gemini CLI, OpenCode); unusable themes are removed from the picker | Automated contrast analysis of all 157 themes completed in this session. 138 pass, 19 removed. All 4 agents use ANSI palette index colors (after Phase 71 fixes OpenCode), so contrast analysis against xterm.js palette data is authoritative for all agents. |
</phase_requirements>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Theme allowlist definition | Frontend (SettingsTab.tsx) | — | Allowlist is a UI-layer constant; no backend involvement |
| Contrast audit / analysis | Build-time / research-time | — | Analysis runs once at research time; result is hardcoded constant, not runtime computation |
| localStorage fallback | Frontend (SettingsTab.tsx) | — | Theme preference is client-side state |
| Test verification of picker contents | Frontend test suite (vitest) | — | `SettingsTab.test.tsx` already tests component source via raw import |

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `xterm-theme` | 1.1.0 | Provides 157 ANSI color palettes | Already installed; `VERIFIED: frontend/node_modules/xterm-theme/package.json` |
| `vitest` | 4.1.0 | Test runner | Already installed; `VERIFIED: frontend/package.json` |
| React | (project version) | Component framework | Existing stack — no change |

### Supporting

None. No new dependencies required.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hardcoded allowlist constant | Runtime contrast computation at startup | Runtime adds ~5ms and unnecessary complexity; result never changes unless xterm-theme package updates |
| Allowlist in `SettingsTab.tsx` | Separate `src/themes.ts` file | Separate file improves testability; either is fine for 138-entry array |
| Automated contrast only | Manual visual review per agent | Manual review is impractical (4 agents × 157 themes = 628 checks); automated analysis is reproducible and fast |

**Installation:** No new packages needed.

---

## Audit Methodology — Verified Results

All analysis was executed against the actual `xterm-theme@1.1.0` data on disk (`frontend/node_modules/xterm-theme/index.js`). [VERIFIED: direct node execution in research session]

### Criteria Applied

| Criterion | Threshold | Rationale |
|-----------|-----------|-----------|
| Foreground : Background contrast | ≥ 3.0 : 1 | WCAG AA minimum for large text; terminal primary text |
| Cursor : Background contrast | ≥ 2.0 : 1 | Cursor must be distinguishable from background |
| Important ANSI colors below 2.5 : 1 | ≤ 3 colors | Allows a few dim ANSI colors (typically `black`/`brightBlack` on dark backgrounds), rejects themes where multiple interactive colors become invisible |

Important ANSI colors checked: `red`, `green`, `yellow`, `blue`, `cyan`, `white`, `brightWhite`, `brightGreen`, `brightYellow`, `brightCyan`, `brightBlue`.

### Results Summary

| Category | Count |
|----------|-------|
| Total themes in package | 157 |
| Pass — kept in picker | 138 |
| Fail — removed from picker | 19 |
| Light-background themes kept | 4 |
| Dark-background themes kept | 134 |

### 19 Themes Removed

| Theme | Failure Reason | Detail |
|-------|---------------|--------|
| C64 | Low FG contrast | fg:bg = 2.26 (purple-on-purple) |
| Royal | Low FG contrast | fg:bg = 2.34 (very dark bg, fg nearly same) |
| Shaman | Low FG contrast | fg:bg = 2.44 |
| CrayonPonyFish | Low FG contrast | fg:bg = 2.76 |
| Grass | Low cursor + many bad ANSI | cursor:bg = 1.54; 4+ important ANSI invisible |
| Later_This_Evening | Low cursor | cursor = #424242 on #222222 bg; ratio 1.58 |
| Ocean | Low cursor + many bad ANSI | cursor:bg = 1.80; blue bg makes many ANSI invisible |
| Zenburn | Low cursor | cursor:bg = 1.84 |
| Spring | Many bad ANSI | Light bg; 4+ important ANSI colors invisible |
| Tomorrow | Many bad ANSI | Light bg; 4+ important ANSI colors invisible (same palette as Spring) |
| PencilLight | Low cursor + many bad ANSI | cursor:bg = 1.94; 4+ invisible ANSI |
| Misterioso | Low cursor | cursor = #000000 on #2d3743; ratio 1.74 |
| Github | Many bad ANSI | Light bg (#f4f4f4); 4+ ANSI colors wash out |
| OneHalfLight | Low cursor + many bad ANSI | cursor = #bfceff on near-white bg; ratio 1.49 |
| BirdsOfParadise | Low cursor | cursor = #573d26 on #2a1f1d; ratio 1.60 |
| Material | Low cursor + many bad ANSI | Light bg; cursor:bg = 2.18; 6 important ANSI invisible |
| Man_Page | Many bad ANSI | Yellow bg (#fef49c); white/bright ANSI invisible |
| Terminal_Basic | Many bad ANSI | White bg; white/brightWhite/brightGreen/brightCyan invisible |
| default | Not a theme palette | This key is the ESM namespace re-export of the whole module, not a color palette |

### 4 Light-Background Themes Kept

| Theme | Background | FG Contrast |
|-------|-----------|------------|
| Novel | `#dfdbc3` | 10.4 : 1 |
| Piatto_Light | `#ffffff` | 10.2 : 1 |
| Solarized_Light | `#fcf4dc` | 5.3 : 1 |
| Violet_Light | `#fcf4dc` | 5.3 : 1 |

All four light themes have acceptable ANSI color contrast — those removed (Spring, Tomorrow, Github, OneHalfLight, PencilLight, Material, Man_Page, Terminal_Basic) failed specifically on ANSI color visibility against light backgrounds.

### Why Automated Analysis Is Valid for All 4 Agents

All 4 agents (Claude Code, Codex, Gemini CLI, OpenCode after Phase 71) output ANSI palette index escape sequences (color indices 0–15). xterm.js remaps these indices using the theme palette object. Therefore, the foreground/background/cursor/ANSI fields in xterm-theme's JSON data ARE the colors that appear on screen — the automated contrast analysis against those values is authoritative. [VERIFIED: Phase 71 research — OpenCode was fixed to use system theme / ANSI palette]

---

## Architecture Patterns

### System Architecture Diagram

```
User opens Settings tab
        |
        v
SettingsTab.tsx renders <select>
  options sourced from ALLOWED_THEMES (138 entries, sorted alphabetically)
  NOT from Object.keys(xtermThemes)
        |
  User selects theme --> onThemeChange(name) --> localStorage write
        |
  App startup: read localStorage --> if not in ALLOWED_THEMES, use 'Tomorrow_Night'
```

### Recommended Project Structure

```
frontend/src/
  components/
    SettingsTab.tsx          # Replace THEME_NAMES constant; add localStorage fallback
    __tests__/
      SettingsTab.test.tsx   # Update THM-01 tests + add new THM-04 tests
```

Optional (clean separation):
```
frontend/src/
  themes.ts                  # Export ALLOWED_THEMES constant
  components/
    SettingsTab.tsx           # Import from ../themes
    __tests__/
      SettingsTab.test.tsx    # Import ALLOWED_THEMES for assertions
```

Either layout satisfies the requirement. The planner should choose based on testability preference.

### Pattern 1: Hardcoded Allowlist Constant

**What:** Replace the dynamic derivation with a sorted array literal.

**When to use:** When the list changes only at design time (new package versions), not runtime.

**Example:**
```typescript
// Before (dynamic — includes 'default' namespace artifact, all 157 keys):
const THEME_NAMES = Object.keys(xtermThemes).sort()

// After (hardcoded allowlist of 138 verified themes):
const ALLOWED_THEMES: string[] = [
  "AdventureTime",
  "Afterglow",
  "AlienBlood",
  // ... 135 more ...
  "idleToes",
]
const THEME_NAMES = ALLOWED_THEMES  // rename or use directly
```

[VERIFIED: analysis of xterm-theme@1.1.0 on disk]

### Pattern 2: localStorage Fallback Guard

**What:** At app initialization (where `selectedTheme` is read from localStorage), check membership in the allowlist.

**When to use:** User may have stored a theme that is now removed from the picker.

**Example:**
```typescript
// In App.tsx or wherever localStorage is read for selectedTheme:
const storedTheme = localStorage.getItem('agenthub:terminalTheme') ?? 'Tomorrow_Night'
const selectedTheme = ALLOWED_THEMES.includes(storedTheme)
  ? storedTheme
  : ALLOWED_THEMES.includes('Tomorrow_Night')
    ? 'Tomorrow_Night'
    : ALLOWED_THEMES[0]
```

Per the UI-SPEC fallback contract:
- Stored theme in allowlist → use it
- Stored theme not in allowlist → `Tomorrow_Night`
- `Tomorrow_Night` itself not in allowlist → first entry of allowlist
- Allowlist empty → `Tomorrow_Night` hardcoded last resort

[ASSUMED: localStorage key is `agenthub:terminalTheme` — confirmed visible in SettingsTab.tsx line 22 comment and UI-SPEC, but the actual read location needs to be located in App.tsx during implementation]

### Anti-Patterns to Avoid

- **Re-deriving from `Object.keys(xtermThemes)`:** The `default` key is the ESM namespace export, not a palette. Dynamic derivation is also fragile if the package adds more namespace exports in future.
- **Runtime contrast computation:** Adds startup latency for a list that never changes at runtime. Compute once at research time, ship as a constant.
- **Filtering xtermThemes by allowlist at render time:** Unnecessary — the allowlist IS the source of truth. Don't use xterm-theme as the authoritative key source and filter it down; just use ALLOWED_THEMES directly.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Contrast ratio computation | Custom algorithm | WCAG formula (already applied in this research) | The W3C WCAG 2.1 relative luminance + contrast ratio formula is the standard; already applied to produce allowlist |
| Theme discovery at runtime | Scan package, filter by contrast | Hardcoded constant from this research | Package contents don't change at runtime |

**Key insight:** The audit work happens once (at research/planning time). The output is a constant. Don't make the app do audit work every render.

---

## Common Pitfalls

### Pitfall 1: `default` Key in Named Imports

**What goes wrong:** `import * as xtermThemes from 'xterm-theme'` captures a `default` key that is the entire module's default export (the object of all themes), not a color palette. If rendered as a `<option>`, the user sees a dropdown entry called "default" that applies the full module object as a theme — which silently uses undefined colors.

**Why it happens:** ESM `import *` includes the `default` export as a property named `default`. The current code already ships this bug — it shows "default" in the picker today.

**How to avoid:** The allowlist approach eliminates this entirely — `default` is simply not in ALLOWED_THEMES.

**Warning signs:** Seeing "default" as a picker option. Applying "default" leaves terminal with browser-default colors.

### Pitfall 2: Stale localStorage After Theme Removal

**What goes wrong:** User previously selected a removed theme. On next app load, `selectedTheme` is a key that doesn't exist in xtermThemes under a valid palette — or doesn't exist in ALLOWED_THEMES — so the terminal renders with whatever xterm.js default is (likely ugly).

**Why it happens:** localStorage persists across app restarts; the stored key is never validated.

**How to avoid:** The fallback guard (Pattern 2 above) must be applied wherever `selectedTheme` is initialized from localStorage — at the App.tsx level, before passing `selectedTheme` as a prop.

**Warning signs:** Terminal appears with wrong colors after upgrade; `selectedTheme` prop has a value not in `ALLOWED_THEMES`.

### Pitfall 3: Test Still Asserts `Object.keys(xtermThemes).sort()`

**What goes wrong:** `SettingsTab.test.tsx` line 88 currently asserts `THEME_NAMES = Object.keys(xtermThemes).sort()`. After the change, this test will fail because the source no longer contains that string.

**Why it happens:** Tests use raw source inspection (`?raw` import). They assert exact source strings.

**How to avoid:** Update the THM-01 describe block tests to assert the new constant pattern (e.g., `ALLOWED_THEMES` or the hardcoded array). Add new THM-04 assertions per the UI-SPEC test contract.

**Warning signs:** `pnpm test -- SettingsTab` shows THM-01 failures.

### Pitfall 4: Sorting Divergence Between Allowlist and UI

**What goes wrong:** `ALLOWED_THEMES` is defined in non-alphabetical order. The UI-SPEC requires alphabetical sort. The `<select>` relies on the array order — it does not sort at render time.

**Why it happens:** If the constant is assembled by appending names as they're analyzed rather than sorted.

**How to avoid:** Ensure `ALLOWED_THEMES` is sorted alphabetically when written (matches the current `sort()` behavior the picker already depends on).

---

## The Complete Allowlist (138 themes) [VERIFIED]

```typescript
export const ALLOWED_THEMES: string[] = [
  "AdventureTime",
  "Afterglow",
  "AlienBlood",
  "Argonaut",
  "Arthur",
  "AtelierSulphurpool",
  "Atom",
  "Batman",
  "Belafonte_Night",
  "Blazer",
  "Borland",
  "Bright_Lights",
  "Broadcast",
  "Brogrammer",
  "Chalk",
  "Chalkboard",
  "Ciapre",
  "Cobalt2",
  "Cobalt_Neon",
  "Dark_Pastel",
  "Darkside",
  "Desert",
  "DimmedMonokai",
  "DotGov",
  "Dracula",
  "Duotone_Dark",
  "ENCOM",
  "Earthsong",
  "Elemental",
  "Elementary",
  "Espresso",
  "Espresso_Libre",
  "Fideloper",
  "FirefoxDev",
  "Firewatch",
  "FishTank",
  "Flat",
  "Flatland",
  "Floraverse",
  "ForestBlue",
  "FrontEndDelight",
  "FunForrest",
  "Galaxy",
  "Glacier",
  "Grape",
  "Gruvbox_Dark",
  "Hardcore",
  "Harper",
  "Highway",
  "Hipster_Green",
  "Homebrew",
  "Hurtado",
  "Hybrid",
  "IC_Green_PPL",
  "IC_Orange_PPL",
  "IR_Black",
  "Jackie_Brown",
  "Japanesque",
  "Jellybeans",
  "JetBrains_Darcula",
  "Kibble",
  "Lavandula",
  "LiquidCarbon",
  "LiquidCarbonTransparent",
  "LiquidCarbonTransparentInverse",
  "MaterialDark",
  "Mathias",
  "Medallion",
  "Molokai",
  "MonaLisa",
  "Monokai_Soda",
  "Monokai_Vivid",
  "N0tch2k",
  "Neopolitan",
  "Neutron",
  "NightLion_v1",
  "NightLion_v2",
  "Night_3024",
  "Novel",
  "Obsidian",
  "OceanicMaterial",
  "Ollie",
  "OneHalfDark",
  "Pandora",
  "Paraiso_Dark",
  "Parasio_Dark",
  "PaulMillr",
  "PencilDark",
  "Piatto_Light",
  "Pnevma",
  "Pro",
  "Red_Alert",
  "Red_Sands",
  "Rippedcasts",
  "Ryuuko",
  "SeaShells",
  "Seafoam_Pastel",
  "Seti",
  "Slate",
  "Smyck",
  "SoftServer",
  "Solarized_Darcula",
  "Solarized_Dark",
  "Solarized_Dark_Higher_Contrast",
  "Solarized_Dark_Patched",
  "Solarized_Light",
  "SpaceGray",
  "SpaceGray_Eighties",
  "SpaceGray_Eighties_Dull",
  "Spacedust",
  "Spiderman",
  "Square",
  "Sundried",
  "Symfonic",
  "Teerb",
  "Thayer_Bright",
  "The_Hulk",
  "Tomorrow_Night",
  "Tomorrow_Night_Blue",
  "Tomorrow_Night_Bright",
  "Tomorrow_Night_Eighties",
  "ToyChest",
  "Treehouse",
  "Ubuntu",
  "UnderTheSea",
  "Urple",
  "Vaughn",
  "VibrantInk",
  "Violet_Dark",
  "Violet_Light",
  "WarmNeon",
  "Wez",
  "WildCherry",
  "Wombat",
  "Wryan",
  "ayu",
  "deep",
  "idleToes",
]
```

`Tomorrow_Night` is at index 113 — it survives the audit (fg:bg = 9.8, cursor:bg = 9.8). The default theme is safe.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (test block present) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test -- SettingsTab` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

[VERIFIED: `frontend/package.json` test script; `frontend/vite.config.ts` test config]

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| THM-04 | Picker contains only allowlisted themes, not all 157 | unit (source inspection) | `pnpm test -- SettingsTab` | ✅ exists — needs update |
| THM-04 | `Tomorrow_Night` is present in picker (default theme survives) | unit | same | ❌ Wave 0 |
| THM-04 | At least one light-background theme in picker | unit | same | ❌ Wave 0 |
| THM-04 | At least one dark-background theme in picker | unit | same | ❌ Wave 0 |
| THM-04 | Stale localStorage theme falls back to Tomorrow_Night | unit | same | ❌ Wave 0 |
| THM-04 | `THEME_NAMES` does not include `default` (namespace artifact) | unit | same | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test -- SettingsTab`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `SettingsTab.test.tsx` — THM-01 describe block: update 2 assertions that reference `Object.keys(xtermThemes).sort()` pattern (they will fail after the change)
- [ ] `SettingsTab.test.tsx` — add THM-04 describe block with 5 new assertions (see test contract in UI-SPEC)
- [ ] Locate where `selectedTheme` is initialized from localStorage (likely `App.tsx`) — add fallback guard test or inline assertion

---

## Environment Availability

Step 2.6: SKIPPED — no external dependencies. This phase is a frontend code change only (modify one constant + update tests). All required tools (Node.js, pnpm, vitest) are confirmed present from the test run executed during research.

---

## Security Domain

> This phase makes no authentication, session, cryptography, or network changes. The only change is a list of strings rendered into a `<select>` element. No ASVS categories apply. Omitting security section.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | localStorage key for theme is `agenthub:terminalTheme` — the fallback guard must target this key | Architecture Patterns - Pattern 2 | Fallback guard would read wrong key, leaving stale themes unhandled; low risk — key is visible in SettingsTab.tsx line 22 comment |
| A2 | The localStorage read for `selectedTheme` happens in App.tsx (not SettingsTab.tsx) | Architecture Patterns | Implementation detail; executor must locate the actual read site before adding fallback |

If A1-A2 table is empty: All claims in this research were verified or cited. But these two items are design details the executor must confirm by reading App.tsx.

---

## Open Questions

1. **Where is `selectedTheme` initialized from localStorage?**
   - What we know: `SettingsTab` receives `selectedTheme` as a prop from its parent
   - What's unclear: Whether the read is in `App.tsx`, a custom hook, or elsewhere — needs to be found before adding the fallback guard
   - Recommendation: Executor reads `App.tsx` to locate the read site at plan execution time; not a blocker for planning

---

## Sources

### Primary (HIGH confidence)

- `frontend/node_modules/xterm-theme/index.js` — direct analysis of all 157 theme palettes; contrast values computed from actual hex data
- `frontend/src/components/SettingsTab.tsx` — verified current `THEME_NAMES` derivation, `<select>` binding, prop interface
- `frontend/src/components/__tests__/SettingsTab.test.tsx` — verified existing test structure and THM-01 assertions that will need updating
- `frontend/vite.config.ts` — verified vitest test config
- `.planning/phases/71-opencode-theming-fix/71-RESEARCH.md` — confirmed all 4 agents use ANSI palette indices after Phase 71

### Secondary (MEDIUM confidence)

- `.planning/phases/73-theme-usability-audit/73-UI-SPEC.md` — design contract for fallback behavior, test assertions, picker format

### Tertiary (LOW confidence)

None.

---

## Metadata

**Confidence breakdown:**

- Allowlist contents: HIGH — computed from actual package data in this session
- Standard stack: HIGH — no new dependencies; all tooling verified by running tests
- Architecture: HIGH — single-file change with well-understood React pattern
- Pitfalls: HIGH — `default` namespace artifact and stale localStorage are observable facts

**Research date:** 2026-04-14
**Valid until:** Until `xterm-theme` package is updated (stable — last published 2020-era package). Allowlist stays valid for any xterm-theme@1.1.0 installation.

## Project Constraints (from CLAUDE.md)

| Directive | Impact on Phase 73 |
|-----------|-------------------|
| Node: `pnpm` preferred | All install/test commands use `pnpm` |
| JS/TS: `camelCase`, `PascalCase` components, TypeScript types | `ALLOWED_THEMES: string[]` type annotation required |
| Testing: `vitest` for JS/TS | Already the test framework in use |
| 80%+ coverage in critical components | Theme picker allowlist is testable; test contract in UI-SPEC covers the critical behaviors |
| No global package installs | No new dependencies; no installs needed |
