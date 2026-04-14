# Phase 73: Theme Usability Audit - Pattern Map

**Mapped:** 2026-04-14
**Files analyzed:** 3 (2 modified, 1 new optional)
**Analogs found:** 3 / 3

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/SettingsTab.tsx` | component | request-response | self (modify in place) | exact — same file |
| `frontend/src/components/__tests__/SettingsTab.test.tsx` | test | — | self (modify in place) | exact — same file |
| `frontend/src/themes.ts` (optional) | utility/constant | — | `frontend/src/App.tsx` (THEME_STORAGE_KEY/DEFAULT_THEME_NAME constants block) | role-match |

---

## Pattern Assignments

### `frontend/src/components/SettingsTab.tsx` (component, modification)

**Analog:** self — existing file at `frontend/src/components/SettingsTab.tsx`

**Current constant to replace** (line 22):
```typescript
// BEFORE — replace this line:
const THEME_NAMES = Object.keys(xtermThemes).sort()

// AFTER — replace with:
const ALLOWED_THEMES: string[] = [
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
const THEME_NAMES = ALLOWED_THEMES
```

**Import pattern** (lines 1-20) — no change needed; `xterm-theme` import stays as-is because `App.tsx` still uses `xtermThemes` for palette lookup:
```typescript
import React, { useState, useEffect } from 'react'
import * as xtermThemes from 'xterm-theme'
// ... rest of imports unchanged
```

Note: If ALLOWED_THEMES is extracted to a separate `themes.ts`, replace the `xterm-theme` import in `SettingsTab.tsx` with:
```typescript
import { ALLOWED_THEMES } from '../themes'
```
And keep the `xterm-theme` import only in `App.tsx` (which still needs it for palette object lookup at line 90).

**Select render pattern** (lines 228-230) — no change:
```typescript
{THEME_NAMES.map(name => (
  <option key={name} value={name}>{name.replace(/_/g, ' ')}</option>
))}
```

**No structural changes:** The `<select>` element, CSS classes, `onChange` handler, `selectedTheme` prop, and all other component logic remain identical. Only line 22 changes.

---

### `frontend/src/App.tsx` (component, modification — localStorage fallback guard)

**Analog:** self — existing file at `frontend/src/App.tsx`

**Current localStorage read pattern** (lines 37-91) — the read site to add the fallback guard:

```typescript
// Lines 37-38: existing constants
const THEME_STORAGE_KEY = 'agenthub:terminalTheme'
const DEFAULT_THEME_NAME = 'Tomorrow_Night'

// Lines 87-91: existing theme state initialization (where guard is added)
const [terminalThemeName, setTerminalThemeName] = useState<string>(
  () => localStorage.getItem(THEME_STORAGE_KEY) ?? DEFAULT_THEME_NAME
)
const terminalTheme: ITheme = (xtermThemes as Record<string, ITheme>)[terminalThemeName]
  ?? (xtermThemes as Record<string, ITheme>)[DEFAULT_THEME_NAME]
```

**After — add allowlist guard to the useState initializer:**
```typescript
// Import ALLOWED_THEMES at top of file (if extracted to themes.ts):
import { ALLOWED_THEMES } from './themes'

// Replace the useState initializer at lines 87-89:
const [terminalThemeName, setTerminalThemeName] = useState<string>(() => {
  const stored = localStorage.getItem(THEME_STORAGE_KEY) ?? DEFAULT_THEME_NAME
  if (ALLOWED_THEMES.includes(stored)) return stored
  if (ALLOWED_THEMES.includes(DEFAULT_THEME_NAME)) return DEFAULT_THEME_NAME
  return ALLOWED_THEMES[0] ?? DEFAULT_THEME_NAME
})
```

**If ALLOWED_THEMES stays in SettingsTab.tsx** (not extracted), App.tsx cannot import it without a circular dependency. In that case, define a minimal inline guard using the same DEFAULT_THEME_NAME constant that already exists:
```typescript
// Inline approach — no import needed:
const [terminalThemeName, setTerminalThemeName] = useState<string>(() => {
  const stored = localStorage.getItem(THEME_STORAGE_KEY) ?? DEFAULT_THEME_NAME
  // Note: full membership check requires ALLOWED_THEMES import from themes.ts
  // Minimal guard: any non-null stored value is used; themes.ts extraction is preferred
  return stored
})
```
The research recommends extracting to `themes.ts` to allow App.tsx to import `ALLOWED_THEMES` for the membership check. If kept inline in SettingsTab.tsx, the fallback guard must be duplicated or omitted (acceptable risk since removed themes still exist in the xterm-theme package and App.tsx line 91 already has a null-coalesce fallback to DEFAULT_THEME_NAME via `?? (xtermThemes as Record<string, ITheme>)[DEFAULT_THEME_NAME]`).

**Existing null-coalesce pattern** (line 91) — already handles unknown theme names for palette lookup:
```typescript
const terminalTheme: ITheme = (xtermThemes as Record<string, ITheme>)[terminalThemeName]
  ?? (xtermThemes as Record<string, ITheme>)[DEFAULT_THEME_NAME]
```
This means even without the localStorage guard, a stale theme name will fall back to `Tomorrow_Night`'s palette at render time. The localStorage guard prevents the stale name from persisting into `terminalThemeName` state and appearing as the selected option in the picker.

---

### `frontend/src/components/__tests__/SettingsTab.test.tsx` (test, modification)

**Analog:** self — existing file at `frontend/src/components/__tests__/SettingsTab.test.tsx`

**Source-inspection test pattern** (lines 1-3) — all new tests must use the same `?raw` import:
```typescript
import { describe, it, expect } from 'vitest'
import raw from '../../components/SettingsTab.tsx?raw'
```

**Existing THM-01 assertions to update** (lines 88-89) — these will fail after the change and must be replaced:
```typescript
// REMOVE these two assertions (lines 88-89):
it('computes THEME_NAMES at module level', () => {
  expect(raw).toContain('THEME_NAMES = Object.keys(xtermThemes).sort()')
})

// REPLACE with:
it('defines ALLOWED_THEMES constant at module level', () => {
  expect(raw).toContain('ALLOWED_THEMES: string[]')
})

it('sets THEME_NAMES to ALLOWED_THEMES', () => {
  expect(raw).toContain('THEME_NAMES = ALLOWED_THEMES')
})
```

**New THM-04 describe block to append** — follows exact style of existing source-inspection tests:
```typescript
describe('THM-04: Allowlist-only theme picker', () => {
  it('ALLOWED_THEMES contains Tomorrow_Night (default theme survives audit)', () => {
    expect(raw).toContain('"Tomorrow_Night"')
  })

  it('ALLOWED_THEMES contains at least one light-background theme (Novel)', () => {
    expect(raw).toContain('"Novel"')
  })

  it('ALLOWED_THEMES contains at least one dark-background theme (Dracula)', () => {
    expect(raw).toContain('"Dracula"')
  })

  it('does NOT contain "default" in ALLOWED_THEMES (namespace artifact excluded)', () => {
    // Only check within the ALLOWED_THEMES block to avoid false positives
    const allowlistStart = raw.indexOf('ALLOWED_THEMES: string[]')
    const allowlistEnd = raw.indexOf(']', allowlistStart)
    expect(allowlistStart).toBeGreaterThan(-1)
    const allowlistBlock = raw.slice(allowlistStart, allowlistEnd + 1)
    expect(allowlistBlock).not.toContain('"default"')
  })

  it('does NOT derive theme names from Object.keys(xtermThemes)', () => {
    expect(raw).not.toContain('Object.keys(xtermThemes).sort()')
  })
})
```

**Fallback guard test** — if guard is in App.tsx, add to a new `App.wiring.test.tsx` describe block or the existing `App.test.tsx`. If guard stays in `SettingsTab.tsx`, add to `SettingsTab.test.tsx`:
```typescript
// In whichever file hosts the guard:
it('localStorage fallback: stale theme not in allowlist falls back to Tomorrow_Night', () => {
  // Source-inspection: the guard pattern must appear in the source
  expect(raw).toContain('ALLOWED_THEMES.includes(stored)')
})
```

---

### `frontend/src/themes.ts` (optional new utility file)

**Analog:** `frontend/src/App.tsx` — pattern for module-level TypeScript constants with type annotations

**Constants pattern from App.tsx** (lines 36-38):
```typescript
const DEFAULT_FONT_SIZE = 14
const THEME_STORAGE_KEY = 'agenthub:terminalTheme'
const DEFAULT_THEME_NAME = 'Tomorrow_Night'
```

**New file structure to follow:**
```typescript
// frontend/src/themes.ts
// Verified allowlist of 138 xterm themes that pass WCAG-derived readability audit.
// See Phase 73 RESEARCH.md for methodology. Update only when xterm-theme package changes.
export const ALLOWED_THEMES: string[] = [
  // ... 138 entries from RESEARCH.md (alphabetically sorted) ...
]
```

**No default export** — named export only (matches project pattern: all other src/ utilities use named exports).

---

## Shared Patterns

### Source-Inspection Test Style
**Source:** `frontend/src/components/__tests__/SettingsTab.test.tsx` (all existing tests)
**Apply to:** All new test assertions in SettingsTab.test.tsx

The project exclusively uses raw source inspection (`?raw` import + `expect(raw).toContain(...)`) rather than mounting components. This avoids Wails runtime mocking complexity. All new THM-04 tests must follow this pattern — no `render()`, no `screen.getBy*()`.

```typescript
import { describe, it, expect } from 'vitest'
import raw from '../../components/SettingsTab.tsx?raw'

describe('THM-04: ...', () => {
  it('assertion name', () => {
    expect(raw).toContain('exact string from source')
  })
})
```

### Scoped String Search (interface/block isolation)
**Source:** `frontend/src/components/__tests__/SettingsTab.test.tsx` (lines 29-43)
**Apply to:** The `"default"` exclusion test in THM-04

When an assertion must avoid false positives from comments or unrelated sections, scope the search to a slice of the raw source:
```typescript
const blockStart = raw.indexOf('ALLOWED_THEMES: string[]')
const blockEnd = raw.indexOf(']', blockStart)
const block = raw.slice(blockStart, blockEnd + 1)
expect(block).not.toContain('"default"')
```

### TypeScript Type Annotation for Arrays
**Source:** `frontend/src/App.tsx` (e.g., line 158: `const restoredTabs: Tab[] = ...`)
**Apply to:** `ALLOWED_THEMES` constant declaration

All array constants in this codebase carry explicit TypeScript type annotations:
```typescript
const ALLOWED_THEMES: string[] = [ ... ]
```

---

## No Analog Found

None. All files have close analogs or are self-modifications of existing files.

---

## Key Implementation Notes for Planner

1. **Decision point:** `ALLOWED_THEMES` in `SettingsTab.tsx` vs. extracted to `themes.ts`. Extracting to `themes.ts` is the only way to let `App.tsx` import it for the localStorage guard. If keeping it inline, the localStorage fallback can be skipped — App.tsx line 91 already provides a silent palette fallback via `?? DEFAULT_THEME_NAME` (stale theme names produce the default palette, not a crash). The picker never shows the stale name again once the user opens settings.

2. **App.tsx already partially handles stale themes** at line 90-91: even if `terminalThemeName` holds a removed theme name, `terminalTheme` will be `Tomorrow_Night`'s palette. The only visible defect without the localStorage guard is that the `<select>` shows no selected option (since the stale name is not in `ALLOWED_THEMES`). This may or may not be acceptable — the UI-SPEC requires a silent fallback, which implies the guard is needed.

3. **Test runner command:** `cd /Users/ken/dev/agenthub/frontend && pnpm test -- SettingsTab`

---

## Metadata

**Analog search scope:** `frontend/src/components/`, `frontend/src/components/__tests__/`, `frontend/src/`
**Files scanned:** `SettingsTab.tsx`, `SettingsTab.test.tsx`, `App.tsx`, `App.wiring.test.tsx`, `App.test.tsx`
**Pattern extraction date:** 2026-04-14
