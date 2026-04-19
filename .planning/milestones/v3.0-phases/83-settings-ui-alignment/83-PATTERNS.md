# Phase 83: Settings UI Alignment - Pattern Map

**Mapped:** 2026-04-18
**Files analyzed:** 4 (2 modified source files + 2 modified test files)
**Analogs found:** 4 / 4

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/SettingsTab.tsx` | component | request-response | `frontend/src/components/SettingsTab.tsx` (self — editing existing file) | exact |
| `frontend/src/style.css` | config (CSS) | transform | `frontend/src/style.css` (self — auditing existing rules) | exact |
| `frontend/src/components/__tests__/SettingsTab.test.tsx` | test | transform | `frontend/src/components/__tests__/SettingsTab.test.tsx` (self — extending existing file) | exact |
| `frontend/src/components/__tests__/style.settings.test.ts` | test | transform | `frontend/src/components/__tests__/style.settings.test.ts` (self — extending existing file) | exact |

---

## Pattern Assignments

### `frontend/src/components/SettingsTab.tsx` (component, request-response)

**Analog:** Self — existing file being edited at lines 521–618.

**Existing structure (lines 521–561) — the two-table bug:**

```tsx
{/* Paths section (SETT-02) */}
<h3>Paths</h3>
{clis.length === 0 ? (
  <p className="settings-panel__empty">No CLIs detected. Install an AI coding CLI and restart the app.</p>
) : (
  <table className="settings-panel__table">
    <thead>
      <tr>
        <th>CLI</th>
        <th>Path</th>
      </tr>
    </thead>
    <tbody>
      {clis.map((cli) => (
        <tr key={cli.Name}>
          <td className="settings-panel__cli-name">{cli.Name}</td>
          <td>
            <div className="settings-panel__path-row">
              <input
                className="settings-panel__path-input"
                type="text"
                value={customPaths[cli.Name] ?? cli.Path}
                onChange={(e) =>
                  setCustomPaths((prev) => ({ ...prev, [cli.Name]: e.target.value }))
                }
                placeholder={cli.Path || `Path to ${cli.Name}`}
              />
              <button
                className="settings-panel__browse-btn"
                onClick={() => void handleBrowse(cli.Name)}
                title="Browse for executable"
              >
                Browse
              </button>
            </div>
          </td>
        </tr>
      ))}
    </tbody>
  </table>
)}
```

**Existing second table (lines 563–601) — separate table for tailscale:**

```tsx
{!clis.find(c => c.Name === 'tailscale') && (() => {
  const tsPath = customPaths['tailscale'] ?? ''
  return (
    <table className="settings-panel__table" style={{ marginTop: '0.75rem' }}>
      <thead>
        <tr>
          <th>Tool</th>   {/* different header text */}
          <th>Path</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td className="settings-panel__cli-name">tailscale</td>
          <td>
            <div className="settings-panel__path-row">
              <input
                className="settings-panel__path-input"
                type="text"
                value={tsPath}
                onChange={(e) =>
                  setCustomPaths((prev) => ({ ...prev, tailscale: e.target.value }))
                }
                placeholder="Path to tailscale (leave blank to auto-detect)"
              />
              <button
                className="settings-panel__browse-btn"
                onClick={() => void handleBrowse('tailscale')}
                title="Browse for executable"
              >
                Browse
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  )
})()}
```

**Target merged-table pattern (to replace both tables above):**

```tsx
{/* Paths section (SET-01 fix: single unified table) */}
<h3>Paths</h3>
{clis.length === 0 && (
  <p className="settings-panel__empty">
    No CLIs detected. Install claude, opencode, or another supported CLI and restart AgentHub to populate the Paths list.
  </p>
)}
<table className="settings-panel__table">
  <thead>
    <tr>
      <th>CLI</th>
      <th>Path</th>
    </tr>
  </thead>
  <tbody>
    {clis.map((cli) => (
      <tr key={cli.Name}>
        <td className="settings-panel__cli-name">{cli.Name}</td>
        <td>
          <div className="settings-panel__path-row">
            <input
              className="settings-panel__path-input"
              type="text"
              value={customPaths[cli.Name] ?? cli.Path}
              onChange={(e) =>
                setCustomPaths((prev) => ({ ...prev, [cli.Name]: e.target.value }))
              }
              placeholder={cli.Path || `Path to ${cli.Name}`}
            />
            <button
              className="settings-panel__browse-btn"
              onClick={() => void handleBrowse(cli.Name)}
              title="Browse for executable"
            >
              Browse
            </button>
          </div>
        </td>
      </tr>
    ))}
    {!clis.find(c => c.Name === 'tailscale') && (
      <tr key="tailscale">
        <td className="settings-panel__cli-name">tailscale</td>
        <td>
          <div className="settings-panel__path-row">
            <input
              className="settings-panel__path-input"
              type="text"
              value={customPaths['tailscale'] ?? ''}
              onChange={(e) =>
                setCustomPaths((prev) => ({ ...prev, tailscale: e.target.value }))
              }
              placeholder="Path to tailscale (leave blank to auto-detect)"
            />
            <button
              className="settings-panel__browse-btn"
              onClick={() => void handleBrowse('tailscale')}
              title="Browse for executable"
            >
              Browse
            </button>
          </div>
        </td>
      </tr>
    )}
  </tbody>
</table>
```

**Inline style removal (line 339) — the SET-02 fix:**

```tsx
// BEFORE (line 339):
<p className="settings-panel__description" style={{ marginTop: '0.25rem', fontSize: '0.8rem' }}>

// AFTER: let the class rule supply font-size: 12px; remove both inline overrides
<p className="settings-panel__description">
```

**Existing inline style pattern to copy for the Appearance section (line 321) — NOT changed:**
Line 321 retains `style={{ marginTop: '0.5rem' }}` on the theme description — this is a separate element and not part of SET-02 scope.

---

### `frontend/src/style.css` (config/CSS, transform)

**Analog:** Self — existing file. No new CSS rules needed. The audit only verifies existing rules apply correctly after the inline style removal.

**Existing consistent h3 rule (lines 361–370) — already correct, no changes:**

```css
.settings-panel__body h3 {
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #9aa5ce;
  margin-bottom: 12px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #292e42;
}
.settings-panel__body h3:first-child {
  margin-top: 0;
  padding-top: 0;
  border-top: none;
}
```

**Existing description rule (lines ~420+) — will apply correctly after inline override removal:**

```css
.settings-panel__description {
  font-size: 12px;
  color: #9aa5ce;
  line-height: 1.5;
  margin-bottom: 10px;
}
```

**Existing table rule (lines 382–397) — already `width: 100%`, no column width changes needed:**

```css
.settings-panel__table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.settings-panel__table th {
  text-align: left;
  color: #9aa5ce;
  font-weight: 500;
  padding: 4px 8px;
  border-bottom: 1px solid #292e42;
}
.settings-panel__table td {
  padding: 8px 8px;
  border-bottom: 1px solid #1a1b26;
}
```

No CSS file edits required for SET-01 (the merged table uses `width: 100%` auto-layout).
No CSS file edits required for SET-02 (the class rule is already correct; inline override is removed from JSX).

---

### `frontend/src/components/__tests__/SettingsTab.test.tsx` (test, transform)

**Analog:** Self — existing file at lines 1–300 being extended. This file uses the raw source-inspection pattern established for all prior phases.

**Existing import and raw-import pattern (lines 1–5):**

```typescript
import { describe, it, expect } from 'vitest'
import raw from '../../components/SettingsTab.tsx?raw'
import appRaw from '../../App.tsx?raw'
import themesRaw from '../../themes.ts?raw'
```

**Existing `describe` / `it` / source-inspection pattern (lines 9–13):**

```typescript
describe('UI-02 Gap 1: SettingsTab exports', () => {
  it('exports SettingsTab function component', () => {
    expect(raw).toContain('export function SettingsTab')
  })
})
```

**Existing interface-block extraction pattern for scoped assertions (lines 31–36):**

```typescript
const interfaceStart = raw.indexOf('interface SettingsTabProps')
const interfaceEnd = raw.indexOf('}', interfaceStart)
expect(interfaceStart).toBeGreaterThan(-1)
const interfaceBlock = raw.slice(interfaceStart, interfaceEnd + 1)
expect(interfaceBlock).not.toContain('isOpen')
```

**New assertions to add — SET-01 table unification (append as new `describe` block):**

```typescript
describe('SET-01: Unified path table (single table for CLI + tailscale rows)', () => {
  it('has exactly one settings-panel__table in the Paths section', () => {
    // The paths section starts at <h3>Paths and ends at settings-panel__save-paths-row.
    // There should be only one opening of settings-panel__table class in that region.
    const pathsStart = raw.indexOf('<h3>Paths</h3>')
    const saveRow = raw.indexOf('settings-panel__save-paths-row', pathsStart)
    expect(pathsStart).toBeGreaterThan(-1)
    const pathsBlock = raw.slice(pathsStart, saveRow)
    const tableMatches = pathsBlock.match(/className="settings-panel__table"/g) ?? []
    expect(tableMatches.length).toBe(1)
  })

  it('tailscale path row is inside the same table as detected CLI rows', () => {
    // After the merge, "tailscale" row and cli.map must appear inside the same <table> block.
    const tableStart = raw.indexOf('className="settings-panel__table"')
    const tableEnd = raw.indexOf('</table>', tableStart)
    expect(tableStart).toBeGreaterThan(-1)
    const tableBlock = raw.slice(tableStart, tableEnd)
    expect(tableBlock).toContain('clis.map')
    expect(tableBlock).toContain("'tailscale'")
  })

  it('does NOT render a second table for tailscale with marginTop style', () => {
    expect(raw).not.toContain("className=\"settings-panel__table\" style={{ marginTop: '0.75rem' }}")
  })

  it('merged table uses CLI as column header (not Tool)', () => {
    const tableStart = raw.indexOf('className="settings-panel__table"')
    const tableEnd = raw.indexOf('</table>', tableStart)
    const tableBlock = raw.slice(tableStart, tableEnd)
    expect(tableBlock).toContain('<th>CLI</th>')
    expect(tableBlock).not.toContain('<th>Tool</th>')
  })
})
```

**New assertions to add — SET-02 inline style removal (append as new `describe` block):**

```typescript
describe('SET-02: No inline fontSize override on description elements', () => {
  it('does NOT have fontSize: 0.8rem on any description paragraph', () => {
    expect(raw).not.toContain("fontSize: '0.8rem'")
  })

  it('does NOT have marginTop: 0.25rem on Tailscale status description', () => {
    // The marginTop was paired with the fontSize override — both should be gone.
    // Check specifically near the tailscale status text to avoid false positives.
    const statusIdx = raw.indexOf('tailscaleStatusText(tailscaleHealth)')
    expect(statusIdx).toBeGreaterThan(-1)
    const nearStatus = raw.slice(statusIdx, statusIdx + 500)
    expect(nearStatus).not.toContain("marginTop: '0.25rem'")
  })
})
```

---

### `frontend/src/components/__tests__/style.settings.test.ts` (test, transform)

**Analog:** Self — existing file at lines 1–60 being extended. Uses `fs.readFileSync` (not `?raw`) because Vitest/jsdom does not support `?raw` for CSS files.

**Existing import and readFileSync pattern (lines 1–8):**

```typescript
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

const css = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')
```

**Existing `describe` / assertion pattern for CSS content checks (lines 10–26):**

```typescript
describe('UI-02 Gap 7: CSS cleanup — modal classes removed', () => {
  it('does NOT contain .settings-overlay (modal backdrop)', () => {
    expect(css).not.toContain('.settings-overlay')
  })
})
```

**New assertions to add — SET-01 CSS guard (append as new `describe` block):**

```typescript
describe('SET-01: No duplicate settings-panel__table margin override', () => {
  it('does NOT contain inline marginTop 0.75rem override (removed from JSX)', () => {
    // This is a JSX inline style, not a CSS rule — but we confirm CSS has no
    // duplicate table rule that would indicate a CSS-only workaround was applied.
    // The real check lives in SettingsTab.test.tsx; this is a belt-and-suspenders guard.
    expect(css).not.toContain('settings-panel__table--tailscale')
  })
})
```

**New assertions to add — SET-02 CSS description rule (append as new `describe` block):**

```typescript
describe('SET-02: Description font-size rule is authoritative at 12px', () => {
  it('contains .settings-panel__description rule', () => {
    expect(css).toContain('.settings-panel__description')
  })

  it('.settings-panel__description uses font-size: 12px', () => {
    const descIdx = css.indexOf('.settings-panel__description')
    const descBlock = css.slice(descIdx, css.indexOf('}', descIdx))
    expect(descBlock).toContain('font-size: 12px')
  })
})
```

---

## Shared Patterns

### Source-Inspection Test Strategy
**Source:** `frontend/src/components/__tests__/SettingsTab.test.tsx` lines 1–5, 9–13
**Apply to:** Both test files (`SettingsTab.test.tsx` and `style.settings.test.ts`)

All tests in this codebase use source-inspection (raw string matching on file content) rather than
DOM rendering or CSS computed-style checks. This is a deliberate choice — it avoids the complexity
of mocking Wails runtime bindings and jsdom CSS computation.

Pattern: use `?raw` import for TSX/TS files, use `readFileSync` for CSS files.
Assertions use `expect(raw).toContain(...)` and `expect(raw).not.toContain(...)`.
For scoped checks (verifying a property is absent from a specific block), slice the raw string to
the block boundaries before asserting.

### BEM CSS Class Convention
**Source:** `frontend/src/style.css` lines 320–420
**Apply to:** Any new JSX elements added to `SettingsTab.tsx`

All settings panel elements use BEM notation:
- Block: `settings-panel`
- Elements: `settings-panel__body`, `settings-panel__table`, `settings-panel__cli-name`, `settings-panel__path-row`, `settings-panel__path-input`, `settings-panel__browse-btn`
- Modifiers: `settings-panel__btn--save`, `settings-panel__btn--saved`, `settings-panel__btn--cancel`

Do not add inline styles for spacing or typography. Use existing class rules or add a BEM modifier class.

### No-Inline-Style Rule
**Source:** RESEARCH.md "Anti-Patterns to Avoid" section; SettingsTab.tsx line 339 (the bug)
**Apply to:** All JSX in `SettingsTab.tsx`

Inline `style={{ ... }}` overrides bypass the BEM system and create invisible inconsistencies.
The one existing legitimate use (Appearance section description `marginTop: '0.5rem'` at line 321)
is out of scope for this phase. The `marginTop: '0.25rem', fontSize: '0.8rem'` at line 339 is the
specific override being removed.

---

## No Analog Found

None. All four files are existing files being extended or edited. Pattern analogs are the files themselves.

---

## Metadata

**Analog search scope:** `frontend/src/components/`, `frontend/src/components/__tests__/`, `frontend/src/style.css`
**Files scanned:** 4 source files read directly
**Pattern extraction date:** 2026-04-18
