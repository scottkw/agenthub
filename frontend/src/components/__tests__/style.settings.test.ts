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

  it('retains .settings-panel__tabs (inner tab nav)', () => {
    expect(css).toContain('.settings-panel__tabs')
  })
})
