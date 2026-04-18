import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// Source-inspection tests for style.css CSS cleanup (UI-02: Settings as sidebar tab).
// Verifies modal-specific CSS was removed and sidebar-tab CSS was added.
// Uses fs.readFileSync because vitest/jsdom does not support ?raw imports for CSS.
const css = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

describe('UI-02 Gap 7: CSS cleanup — modal classes removed', () => {
  it('does NOT contain .settings-overlay (modal backdrop)', () => {
    expect(css).not.toContain('.settings-overlay')
  })

  it('does NOT contain .settings-panel__header (modal header)', () => {
    expect(css).not.toContain('.settings-panel__header')
  })

  it('does NOT contain .settings-panel__footer (modal footer)', () => {
    expect(css).not.toContain('.settings-panel__footer')
  })

  it('does NOT contain .settings-panel__close (modal close button)', () => {
    expect(css).not.toContain('.settings-panel__close')
  })
})

describe('UI-02 Gap 7: CSS cleanup — sidebar-tab class added', () => {
  it('contains .settings-tab (sidebar tab outer wrapper)', () => {
    expect(css).toContain('.settings-tab')
  })
})

describe('UI-02 Gap 7: CSS cleanup — inner classes retained', () => {
  it('retains .settings-panel__body (inner content area)', () => {
    expect(css).toContain('.settings-panel__body')
  })

  it('does NOT contain .settings-panel__tabs (tab nav removed)', () => {
    expect(css).not.toContain('.settings-panel__tabs')
  })

  it('does NOT contain .settings-panel__tab-btn (tab button removed)', () => {
    expect(css).not.toContain('.settings-panel__tab-btn')
  })
})

describe('SETT-02: Section header CSS', () => {
  it('contains .settings-panel__body h3 rule', () => {
    expect(css).toContain('.settings-panel__body h3')
  })

  it('contains h3:first-child override rule', () => {
    expect(css).toContain('.settings-panel__body h3:first-child')
  })

  it('h3 rule includes section divider border', () => {
    expect(css).toContain('border-top: 1px solid #292e42')
  })
})

describe('SET-01: No duplicate settings-panel__table margin override', () => {
  it('does NOT contain settings-panel__table--tailscale modifier class', () => {
    expect(css).not.toContain('settings-panel__table--tailscale')
  })
})

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
