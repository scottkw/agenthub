// Phase 147-01: HelpTab source-gate test stubs (RED until Plans 02/03).
//
// These tests assert that App.tsx declares HELP_TAB/handleOpenHelp and that
// style.css declares the --hub-search-highlight-bg token in both :root and
// [data-ui-theme="light"]. They WILL fail until Plans 02/03 add those
// implementations — this is the intended RED state for Wave 0.

import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

const appSrc = readFileSync(resolve(__dirname, '../../App.tsx'), 'utf-8')
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

// ============================================================
// App.tsx source gates — HELP_TAB wiring (RED until Plan 02)
// ============================================================

describe('HelpTab source gate: HELP_TAB constant in App.tsx (Phase 147)', () => {
  it("App.tsx declares HELP_TAB with id '__help__'", () => {
    expect(appSrc).toContain("id: '__help__'")
  })

  it("App.tsx declares HELP_TAB with type 'help'", () => {
    expect(appSrc).toContain("type: 'help'")
  })

  it('App.tsx declares HELP_TAB constant', () => {
    expect(appSrc).toContain('HELP_TAB')
  })

  it('App.tsx declares handleOpenHelp callback', () => {
    expect(appSrc).toContain('handleOpenHelp')
  })
})

// ============================================================
// style.css source gates — --hub-search-highlight-bg token (RED until Plan 03)
// ============================================================

describe('HelpTab source gate: --hub-search-highlight-bg CSS token (Phase 147)', () => {
  it('style.css declares --hub-search-highlight-bg in :root', () => {
    // Token must be present in the :root block
    expect(cssRaw).toContain('--hub-search-highlight-bg')
  })

  it('style.css declares --hub-search-highlight-bg in [data-ui-theme="light"]', () => {
    // Token must also be overridden in the light-theme block
    const lightThemeIdx = cssRaw.indexOf('[data-ui-theme="light"]')
    expect(lightThemeIdx).toBeGreaterThan(-1)
    const lightThemeSection = cssRaw.slice(lightThemeIdx, lightThemeIdx + 500)
    expect(lightThemeSection).toContain('--hub-search-highlight-bg')
  })
})
