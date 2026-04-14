# Phase 72: UI Contrast Improvement - Pattern Map

**Mapped:** 2026-04-14
**Files analyzed:** 2 (1 modified, 1 created)
**Analogs found:** 2 / 2

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/style.css` | config (CSS) | transform (color token replacement) | `frontend/src/style.css` itself — surgical edit | self |
| `frontend/src/components/__tests__/style.contrast.test.ts` | test | request-response (CSS source inspection) | `frontend/src/components/__tests__/style.settings.test.ts` | exact |

## Pattern Assignments

### `frontend/src/style.css` (config, transform)

**Analog:** `frontend/src/style.css` — surgical color replacement in existing rules.

**The 11 failing `color:` declarations to replace** (each is `color: #565f89;` → `color: #9aa5ce;`):

| Line | Selector | Background | Current Ratio |
|------|----------|------------|---------------|
| 109 | `.tab { color: ... }` | `#16161e` | 2.91:1 FAIL |
| 152 | `.tab__close { color: ... }` | `#16161e` | 2.91:1 FAIL |
| 270 | `.tab-status-bar { color: ... }` | `#16161e` | 2.91:1 FAIL |
| 282 | `.tab-status-bar__state--off { color: ... }` | `#16161e` | 2.91:1 FAIL |
| 350 | `.settings-panel__body h3 { color: ... }` | `#1e2030` | 2.60:1 FAIL |
| 363 | `.settings-panel__empty { color: ... }` | `#1e2030` | 2.60:1 FAIL |
| 374 | `.settings-panel__table th { color: ... }` | `#1e2030` | 2.60:1 FAIL |
| 464 | `.settings-panel__description { color: ... }` | `#1e2030` | 2.60:1 FAIL |
| 496 | `.settings-panel__url { color: ... }` | `#1e2030` | 2.60:1 FAIL |
| 600 | `.new-session-modal__section-label { color: ... }` | `#1e2030` | 2.60:1 FAIL |
| 1014 | `.welcome-tab__version { color: ... }` | `#1a1b26` | 2.76:1 FAIL |
| 1030 | `.welcome-tab__heading { color: ... }` | `#1a1b26` | 2.76:1 FAIL |

**Current pattern** (lines 99–114 — `.tab` rule as representative example):
```css
.tab {
  display: flex;
  align-items: center;
  padding: 0 10px;
  height: 100%;
  min-width: 80px;
  max-width: 180px;
  cursor: pointer;
  background-color: transparent;
  border-right: 1px solid #292e42;
  color: #565f89;          /* <-- FAILING: 2.91:1 on #16161e */
  font-size: 13px;
  gap: 6px;
  transition: background-color 0.1s;
  flex-shrink: 0;
}
```

**After fix** (only the `color:` line changes):
```css
  color: #9aa5ce;          /* 7.41:1 on #16161e — PASS */
```

**Current pattern** (lines 144–158 — `.tab__close`):
```css
.tab__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: none;
  background: transparent;
  color: #565f89;          /* <-- FAILING: 2.91:1 on #16161e */
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  border-radius: 2px;
  flex-shrink: 0;
  padding: 0;
}
```

**Current pattern** (lines 259–272 — `.tab-status-bar`):
```css
/* --- Per-tab status bar (Phase 8) ---------------------------------- */
.tab-status-bar {
  flex-shrink: 0;
  height: 32px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  background-color: #16161e;
  border-top: 1px solid #292e42;
  overflow: hidden;
  font-size: 12px;
  color: #565f89;          /* <-- FAILING: 2.91:1 on #16161e */
  font-family: inherit;
}
```

**Current pattern** (line 282 — `.tab-status-bar__state--off`):
```css
.tab-status-bar__state--on     { color: #9ece6a; }
.tab-status-bar__state--off    { color: #565f89; }    /* <-- FAILING */
.tab-status-bar__state--inactive { color: #414868; }
```

**Current pattern** (lines 346–365 — settings section headers + empty state):
```css
.settings-panel__body h3 {
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #565f89;          /* <-- FAILING: 2.60:1 on #1e2030 */
  margin-bottom: 12px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #292e42;
}
.settings-panel__empty {
  color: #565f89;          /* <-- FAILING: 2.60:1 on #1e2030 */
  font-size: 13px;
}
```

**Current pattern** (lines 372–378 — settings table header):
```css
.settings-panel__table th {
  text-align: left;
  color: #565f89;          /* <-- FAILING: 2.60:1 on #1e2030 */
  font-weight: 500;
  padding: 4px 8px;
  border-bottom: 1px solid #292e42;
}
```

**Current pattern** (lines 462–467 — settings description):
```css
.settings-panel__description {
  font-size: 12px;
  color: #565f89;          /* <-- FAILING: 2.60:1 on #1e2030 */
  line-height: 1.5;
  margin-bottom: 10px;
}
```

**Current pattern** (lines 493–497 — settings URL label):
```css
.settings-panel__url {
  margin-top: 8px;
  font-size: 12px;
  color: #565f89;          /* <-- FAILING: 2.60:1 on #1e2030 */
}
```

**Current pattern** (lines 594–602 — new session modal section label):
```css
.new-session-modal__section-label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #565f89;          /* <-- FAILING: 2.60:1 on #1e2030 */
  margin-bottom: 8px;
}
```

**Current pattern** (lines 1012–1033 — welcome tab version + heading):
```css
.welcome-tab__version {
  font-size: 12px;
  color: #565f89;          /* <-- FAILING: 2.76:1 on #1a1b26 */
  margin-bottom: 40px;
}
.welcome-tab__heading {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: #565f89;          /* <-- FAILING: 2.76:1 on #1a1b26 */
  margin-bottom: 10px;
  font-weight: 600;
}
```

**Constraint — do NOT change these `#565f89` usages** (decorative, no text contrast requirement):
- Any `border-color:`, `border:`, `background-color:`, or `background:` property using `#565f89`
- `.tab-status-bar__state--inactive { color: #414868 }` — separate color, separate decision

---

### `frontend/src/components/__tests__/style.contrast.test.ts` (test, CSS source inspection)

**Analog:** `frontend/src/components/__tests__/style.settings.test.ts` (lines 1–61) — exact pattern match.

**File header / setup pattern** (lines 1–8 of analog):
```typescript
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// Source-inspection tests for style.css CSS cleanup (UI-02: Settings as sidebar tab).
// Verifies modal-specific CSS was removed and sidebar-tab CSS was added.
// Uses fs.readFileSync because vitest/jsdom does not support ?raw imports for CSS.
const css = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')
```

**Negative assertion pattern** (lines 11–13 of analog):
```typescript
describe('UI-02 Gap 7: CSS cleanup — modal classes removed', () => {
  it('does NOT contain .settings-overlay (modal backdrop)', () => {
    expect(css).not.toContain('.settings-overlay')
  })
```

**Positive assertion pattern** (lines 28–31 of analog):
```typescript
describe('UI-02 Gap 7: CSS cleanup — sidebar-tab class added', () => {
  it('contains .settings-tab (sidebar tab outer wrapper)', () => {
    expect(css).toContain('.settings-tab')
  })
})
```

**New test: WCAG contrast math helper** (embed directly in test file, from RESEARCH.md verified formula):
```typescript
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

**New test: selector-scoped negative assertion** (use regex, not plain `toContain`, to scope to a rule block):
```typescript
describe('UI-01: tab bar contrast', () => {
  it('.tab rule does not use failing color #565f89 for text', () => {
    expect(css).not.toMatch(/\.tab\s*\{[^}]*color:\s*#565f89/)
  })
  it('.tab__close rule does not use failing color #565f89 for text', () => {
    expect(css).not.toMatch(/\.tab__close\s*\{[^}]*color:\s*#565f89/)
  })
})
```

**New test: programmatic contrast ratio verification** (for replacement color on all backgrounds):
```typescript
describe('UI-01: replacement color passes WCAG AA', () => {
  const REPLACEMENT = '#9aa5ce'
  it('passes 4.5:1 on sidebar/tab-bar background #16161e', () => {
    expect(contrastRatio(REPLACEMENT, '#16161e')).toBeGreaterThanOrEqual(4.5)
  })
  it('passes 4.5:1 on settings background #1e2030', () => {
    expect(contrastRatio(REPLACEMENT, '#1e2030')).toBeGreaterThanOrEqual(4.5)
  })
  it('passes 4.5:1 on main area background #1a1b26', () => {
    expect(contrastRatio(REPLACEMENT, '#1a1b26')).toBeGreaterThanOrEqual(4.5)
  })
})
```

---

## Shared Patterns

### CSS Edit Rule — `color:` Only
**Source:** `frontend/src/style.css` (all failing rules listed above)
**Apply to:** Every edit in `style.css`

Only change `color:` properties. Never change `border-color:`, `border:`, `background-color:`, or `background:` uses of `#565f89` — those are decorative and have no text contrast requirement.

### Test File Structure
**Source:** `frontend/src/components/__tests__/style.settings.test.ts` (lines 1–61)
**Apply to:** `style.contrast.test.ts`

All style tests in this project:
1. Import `readFileSync` + `resolve` from Node builtins
2. Load `style.css` as a raw string with `readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')`
3. Group assertions by feature in `describe()` blocks
4. Use `expect(css).not.toMatch(regex)` for selector-scoped negative checks (not `not.toContain()`)
5. Run with: `cd /Users/ken/dev/agenthub/frontend && pnpm test`

---

## No Analog Found

None. Both files have direct analogs in the codebase.

---

## Metadata

**Analog search scope:** `frontend/src/style.css`, `frontend/src/components/__tests__/`
**Files scanned:** 2 source files + grep across style.css
**Pattern extraction date:** 2026-04-14
