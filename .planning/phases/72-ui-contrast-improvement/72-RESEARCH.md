# Phase 72: UI Contrast Improvement - Research

**Researched:** 2026-04-14
**Domain:** CSS color / WCAG accessibility / React/TypeScript frontend
**Confidence:** HIGH

## Summary

Phase 72 is a pure CSS color-fix phase. The app uses a TokyoNight-inspired palette
(`#16161e` / `#1e2030` / `#1a1b26` backgrounds, `#565f89` as the dim muted color).
Direct contrast measurements (performed in this session) confirm that `#565f89` fails
WCAG AA (4.5:1) against every app background it appears on: ratio ranges from 2.60 to
2.91. This single color is responsible for all failing instances.

Every other text color already passes: `#9aa5ce` (sidebar labels, 7.41:1), `#a9b1d6`
(settings labels, tab hover, 7.63:1+), `#c0caf5` (active tab, 10.59:1). The fix
consists of upgrading `#565f89` to a brighter value in the specific CSS rules that
control the four affected UI surfaces: tab bar inactive text, settings section headers
and description text, welcome version text and headings, and the status bar.

`#8088a8` (5.14:1 on the darkest background `#16161e`, 4.60:1 on `#1e2030`) is the
minimum hue-matched replacement that passes 4.5:1 AA on all three backgrounds.
`#9aa5ce` (7.41:1) is a conservative choice that matches the existing sidebar label
color and guarantees comfortable legibility everywhere.

**Primary recommendation:** Replace all `#565f89` occurrences that carry readable text
content with `#9aa5ce` in `style.css`. Retain `#565f89` only where it is used as a
decorative separator/border accent (where no text contrast requirement applies).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| UI-01 | Main GUI text elements (sidebar labels, tab titles, settings labels, welcome content, status bar text) meet WCAG AA contrast ratio (4.5:1 for normal text) against the dark background theme | Contrast audit below identifies every failing CSS rule and the replacement color needed |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| style.css | — | Single global stylesheet for all app UI | All app CSS lives here — no CSS modules |
| vitest | ^4.1.0 | Test runner for style verification tests | Already installed; `style.settings.test.ts` establishes the pattern |

### Supporting
No additional libraries are needed. This is a CSS-only change.

**Installation:** No new packages required.

**Version verification:** [VERIFIED: package.json] — vitest 4.1.0 is the installed version.

## Architecture Patterns

### Recommended Project Structure
No structural changes — all changes are in:
```
frontend/src/
└── style.css        # only file that changes
```

Test file:
```
frontend/src/components/__tests__/
└── style.contrast.test.ts    # new — follows style.settings.test.ts pattern
```

### Pattern 1: CSS Source-Inspection Tests
**What:** Vitest reads `style.css` as a raw string via `fs.readFileSync` and asserts
that specific color values appear (or do not appear) in specific rule contexts.

**When to use:** When verifying that a specific CSS property value is set on a
specific selector — since jsdom cannot compute rendered styles in this project setup.

**Example:**
```typescript
// Source: frontend/src/components/__tests__/style.settings.test.ts (existing pattern)
import { readFileSync } from 'fs'
import { resolve } from 'path'

const css = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

it('inactive tab has passing contrast color', () => {
  // The .tab rule must NOT use the failing #565f89 for color
  // and MUST use a passing color
  expect(css).not.toMatch(/\.tab\s*\{[^}]*color:\s*#565f89/)
})
```

**Note:** CSS source inspection tests are brittle against reformatting. Keep
assertions coarse — check that `#565f89` does NOT appear in text-content rules
rather than asserting the exact replacement hex.

### Pattern 2: WCAG Contrast Calculation (JavaScript)
**What:** Pure JS luminance formula for computing contrast ratios without a library.

**Example:**
```typescript
// Source: WCAG 2.1 specification (verified in this session)
function sRGB(c: number): number {
  c = c / 255
  return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
}
function relativeLuminance(hex: string): number {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return 0.2126 * sRGB(r) + 0.7152 * sRGB(g) + 0.0722 * sRGB(b)
}
function contrastRatio(fg: string, bg: string): number {
  const L1 = relativeLuminance(fg)
  const L2 = relativeLuminance(bg)
  return (Math.max(L1, L2) + 0.05) / (Math.min(L1, L2) + 0.05)
}
```

This can be embedded in a test to programmatically verify that chosen replacement
colors meet 4.5:1 on all three background colors.

### Anti-Patterns to Avoid
- **Targeting every `#565f89` instance:** Some uses of `#565f89` are decorative
  (border colors, icon color when not conveying text). Only replace it where the
  color is on a text-content CSS property (`color:`), not `border-color:`,
  `background-color:`, etc.
- **Introducing a new color not in the palette:** Use `#9aa5ce` (already the sidebar
  label color) or `#a9b1d6` (already the settings label / tab hover color) to keep
  the palette coherent.
- **Using inline styles:** All color changes belong in `style.css`, not JSX
  `style={{}}` props, to keep color maintenance centralized.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Contrast ratio math | Custom algorithm | WCAG formula (see code example above) | Single standard formula; no library needed for 2 colors |
| Contrast verification | Manual eyeballing | Automated test with computed ratio | Prevents regression |
| Color picker / audit tool | New UI | Node one-liner (used in this research) | All values are hardcoded hex — computed once at design time |

**Key insight:** This is a design-constants problem, not a runtime problem. All
colors are static hex values in one CSS file. The fix is edit + test; no new
abstractions are needed.

## Failing Contrast Audit

> All ratios computed in this session. [VERIFIED: direct computation using WCAG formula]

### Background Color Key
- `#16161e` — sidebar and tab-bar background
- `#1a1b26` — main area background (welcome tab, terminal container)
- `#1e2030` — settings panel body, modal bodies

### Failing CSS Rules (using `#565f89` as text `color:`)

| CSS Selector / Rule | Current Color | Background | Current Ratio | AA Required | Fix |
|--------------------|---------------|------------|---------------|-------------|-----|
| `.tab { color: ... }` (inactive tab text) | `#565f89` | `#16161e` | **2.91:1** | 4.5:1 | Replace with `#9aa5ce` |
| `.tab__close { color: ... }` (close button) | `#565f89` | `#16161e` | **2.91:1** | 4.5:1 | Replace with `#9aa5ce` |
| `.settings-panel__body h3 { color: ... }` (section headers) | `#565f89` | `#1e2030` | **2.60:1** | 4.5:1* | Replace with `#9aa5ce` |
| `.settings-panel__description { color: ... }` | `#565f89` | `#1e2030` | **2.60:1** | 4.5:1 | Replace with `#9aa5ce` |
| `.settings-panel__empty { color: ... }` | `#565f89` | `#1e2030` | **2.60:1** | 4.5:1 | Replace with `#9aa5ce` |
| `.settings-panel__table th { color: ... }` | `#565f89` | `#1e2030` | **2.60:1** | 4.5:1 | Replace with `#9aa5ce` |
| `.tab-status-bar { color: ... }` (status bar) | `#565f89` | `#16161e` | **2.91:1** | 4.5:1 | Replace with `#9aa5ce` |
| `.welcome-tab__version { color: ... }` | `#565f89` | `#1a1b26` | **2.76:1** | 4.5:1 | Replace with `#9aa5ce` |
| `.welcome-tab__heading { color: ... }` | `#565f89` | `#1a1b26` | **2.76:1** | 4.5:1* | Replace with `#9aa5ce` |
| `.settings-panel__url { color: ... }` | `#565f89` | `#1e2030` | **2.60:1** | 4.5:1 | Replace with `#9aa5ce` |
| `.new-session-modal__section-label { color: ... }` | `#565f89` | `#1e2030` | **2.60:1** | 4.5:1* | Replace with `#9aa5ce` |

*Section headers and section labels are uppercase+small caps at 11-13px. Per WCAG 2.1,
"large text" (18pt or 14pt bold) gets the relaxed 3:1 threshold. These elements are
small-caps at 11-13px CSS pixels, which does NOT qualify as large text — 4.5:1 applies.

### Passing Rules (no change needed)
| CSS Selector | Color | Lowest Background | Ratio | Status |
|-------------|-------|-------------------|-------|--------|
| `.sidebar__item { color: ... }` | `#9aa5ce` | `#16161e` | 7.41:1 | PASS |
| `.settings-panel__label { color: ... }` | `#a9b1d6` | `#1e2030` | 7.63:1 | PASS |
| `.tab--active { color: ... }` | `#c0caf5` | `#1a1b26` | 10.59:1 | PASS |
| `.welcome-tab__tagline { color: ... }` | `#a9b1d6` | `#1a1b26` | 8.10:1 | PASS |
| `.welcome-tab__text { color: ... }` | `#a9b1d6` | `#1a1b26` | 8.10:1 | PASS |

### `#565f89` Uses That Are NOT Text (keep as-is)
These use `#565f89` as a border/background/decorative value — no text contrast
requirement:
- `border-right: 1px solid #292e42` and similar `border:` rules (not text)
- `.tab-status-bar__state--off { color: #565f89 }` — used for OFF-state badge (small
  indicator pill, not primary reading content) [ASSUMED — may need review if it's
  readable state text]
- `.tab-status-bar__state--inactive { color: #414868 }` — separate color, also
  failing: 1.72:1 on `#16161e` — but this is "inactive" state which may be
  intentionally invisible [ASSUMED]

## Replacement Color Decision

`#9aa5ce` is recommended as the universal replacement for failing `#565f89` text uses:

| Color | On `#16161e` | On `#1e2030` | On `#1a1b26` | Verdict |
|-------|-------------|-------------|-------------|---------|
| `#565f89` (current) | 2.91:1 FAIL | 2.60:1 FAIL | 2.76:1 FAIL | Replace |
| `#8088a8` (minimum AA) | 5.14:1 PASS | 4.60:1 PASS | 4.89:1 PASS | Borderline |
| `#9aa5ce` (recommended) | **7.41:1 PASS** | **6.63:1 PASS** | **7.04:1 PASS** | Comfortable margin |
| `#a9b1d6` (current labels) | 8.52:1 PASS | 7.63:1 PASS | 8.10:1 PASS | Also fine |

[VERIFIED: computed in this research session using WCAG formula]

Using `#9aa5ce` (the current sidebar label color) has an additional benefit: it is
already in the palette and will look visually consistent across the app. The secondary
text will be slightly dimmer than primary text (`#c0caf5`) but clearly legible.

## Common Pitfalls

### Pitfall 1: Replacing `#565f89` Where It Is Not Text
**What goes wrong:** A global find-replace of `#565f89` → `#9aa5ce` changes border
colors, hover backgrounds, and decorative separators — making them too bright.
**Why it happens:** `#565f89` appears 15+ times in `style.css` in multiple contexts.
**How to avoid:** Only change `color:` properties, never `border-color:`,
`background-color:`, or `background:` uses.
**Warning signs:** If borders or separators become visually prominent/harsh after the
change.

### Pitfall 2: Tab Close Button Forgetting
**What goes wrong:** The inactive `.tab` text gets fixed, but `.tab__close` still uses
`#565f89` — the close button × remains dim.
**Why it happens:** Two separate rules both use `#565f89`.
**How to avoid:** Check both `.tab { color }` and `.tab__close { color }` explicitly.

### Pitfall 3: Status Bar State Colors
**What goes wrong:** The `.tab-status-bar` base color is fixed but the `--off` state
(`#565f89`) is forgotten, leaving an inconsistent element.
**How to avoid:** Audit all `tab-status-bar__state` variants — `--on` (green, passes),
`--off` (failing), `--inactive` (also failing, even dimmer).

### Pitfall 4: Forgetting New Session Modal
**What goes wrong:** Sidebar, tab bar, settings, and welcome are all fixed — but the
New Session Modal section labels (also `#565f89` on `#1e2030`) are missed because the
modal is not always visible.
**How to avoid:** The planner should include `.new-session-modal__section-label` in the
fix scope.

### Pitfall 5: Test Approach Brittleness
**What goes wrong:** A test that asserts `css.includes('#565f89')` will fail as soon as
any decorative use of `#565f89` remains — which it should, for borders. Tests must be
scoped to the specific CSS rule blocks, not the file as a whole.
**How to avoid:** Use regex that matches the selector + color together, e.g.,
`/\.tab\s*\{[^}]*color:\s*#565f89/`. Or test that the replacement color IS present on
the target selectors rather than that the old color is absent.

## Code Examples

### Verified Contrast Calculation (for test file)
```typescript
// Source: WCAG 2.1 specification — verified in this session
function sRGB(c: number): number {
  c = c / 255
  return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
}
function relativeLuminance(hex: string): number {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return 0.2126 * sRGB(r) + 0.7152 * sRGB(g) + 0.0722 * sRGB(b)
}
function contrastRatio(fg: string, bg: string): number {
  const L1 = relativeLuminance(fg)
  const L2 = relativeLuminance(bg)
  return (Math.max(L1, L2) + 0.05) / (Math.min(L1, L2) + 0.05)
}

// Key passing assertions (computed in this session):
// contrastRatio('#9aa5ce', '#16161e') === 7.41  (PASS)
// contrastRatio('#9aa5ce', '#1e2030') === 6.63  (PASS)
// contrastRatio('#9aa5ce', '#1a1b26') === 7.04  (PASS)
```

### Existing Test Pattern (from style.settings.test.ts)
```typescript
// Source: frontend/src/components/__tests__/style.settings.test.ts
import { readFileSync } from 'fs'
import { resolve } from 'path'
const css = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

it('contains .settings-panel__body h3 rule', () => {
  expect(css).toContain('.settings-panel__body h3')
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual visual review | Automated WCAG contrast ratio tests | Phase 72 | Regressions caught in CI |
| `#565f89` for all muted text | `#9aa5ce` for muted text, `#565f89` only for decorative/border | Phase 72 | All text meets 4.5:1 AA |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `.tab-status-bar__state--off` (color: `#565f89`) and `--inactive` (color: `#414868`) are intentionally dim "invisible" state indicators and not primary reading content | Failing Audit table footnote | If these ARE meant to be read by users, they also need to be fixed; planner should confirm scope |
| A2 | The New Session Modal section labels (`.new-session-modal__section-label`) are in scope for UI-01 ("main GUI text elements") | Failing Audit table | If out of scope, remove from plan; but the requirement lists "settings labels" which strongly suggests all non-terminal text |

## Open Questions

1. **Status bar `--off` / `--inactive` state colors**
   - What we know: `#565f89` (off) = 2.91:1 FAIL; `#414868` (inactive) = 1.72:1 FAIL
   - What's unclear: Are these intentionally invisible (state meaning: "nothing active, ignore me") or are they readable labels?
   - Recommendation: Planner should include both states in the fix. The `--off` state means "no web server" which users need to read. The `--inactive` state means "session closed" — also readable.

2. **New Session Modal scope**
   - What we know: `.new-session-modal__section-label` uses `#565f89` on `#1e2030`
   - What's unclear: The phase success criteria list sidebar, tab bar, settings, and welcome — the modal is not explicitly mentioned
   - Recommendation: Include it anyway. It is non-terminal GUI text, and the requirement text says "main GUI text elements."

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — pure CSS color change + vitest tests already installed)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (vitest configured inline) |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UI-01 | Sidebar labels meet 4.5:1 contrast | unit (CSS inspection) | `cd frontend && pnpm test -- style.contrast` | ❌ Wave 0 |
| UI-01 | Tab text meets 4.5:1 contrast | unit (CSS inspection) | `cd frontend && pnpm test -- style.contrast` | ❌ Wave 0 |
| UI-01 | Settings labels/headers meet 4.5:1 | unit (CSS inspection) | `cd frontend && pnpm test -- style.contrast` | ❌ Wave 0 |
| UI-01 | Welcome content meets 4.5:1 | unit (CSS inspection) | `cd frontend && pnpm test -- style.contrast` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/style.contrast.test.ts` — covers all UI-01 contrast assertions
  - Embeds WCAG formula, asserts `contrastRatio('#9aa5ce', bg) >= 4.5` for each background
  - Asserts failing-selector regex patterns do NOT still contain `#565f89` as text color

*(All other test infrastructure — vitest, jsdom, test runner — already in place)*

## Security Domain

This phase involves no authentication, session management, access control, cryptography,
or input handling. Security domain does not apply.

## Sources

### Primary (HIGH confidence)
- [VERIFIED: direct computation] — All contrast ratios computed in this session using WCAG 2.1 relative luminance formula
- [VERIFIED: file read] `frontend/src/style.css` — all color values extracted from source
- [VERIFIED: file read] `frontend/src/components/__tests__/style.settings.test.ts` — test pattern confirmed

### Secondary (MEDIUM confidence)
- [CITED: https://www.w3.org/TR/WCAG21/#contrast-minimum] — WCAG 2.1 Success Criterion 1.4.3 (Level AA): 4.5:1 for normal text, 3:1 for large text

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — CSS + vitest already in place, no new dependencies
- Architecture: HIGH — single file change, established test pattern
- Pitfalls: HIGH — all derived from direct code inspection
- Contrast calculations: HIGH — computed programmatically using WCAG formula

**Research date:** 2026-04-14
**Valid until:** 2026-05-14 (stable — hardcoded CSS values)
