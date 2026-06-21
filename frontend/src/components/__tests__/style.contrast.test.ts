import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// UI-01: WCAG AA contrast regression tests for non-terminal GUI text.
const css = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

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

describe('UI-01: replacement color passes WCAG AA on all backgrounds', () => {
  it('#9398a8 achieves >= 4.5 contrast on #16181f (comp sidebar/tab-bar bg)', () => {
    expect(contrastRatio('#9398a8', '#16181f')).toBeGreaterThanOrEqual(4.5)
  })

  it('#9398a8 achieves >= 4.5 contrast on #14151b (comp main area bg)', () => {
    expect(contrastRatio('#9398a8', '#14151b')).toBeGreaterThanOrEqual(4.5)
  })

  it('#9398a8 achieves >= 4.5 contrast on #1c1e28 (comp settings panel bg)', () => {
    expect(contrastRatio('#9398a8', '#1c1e28')).toBeGreaterThanOrEqual(4.5)
  })
})

describe('UI-01: tab bar contrast — no failing #565f89 text color', () => {
  it('.tab rule does not use #565f89 for color', () => {
    expect(css).not.toMatch(/\.tab\s*\{[^}]*color:\s*#565f89/)
  })

  it('.tab__close rule does not use #565f89', () => {
    expect(css).not.toMatch(/\.tab__close\s*\{[^}]*color:\s*#565f89/)
  })

  it('.tab-status-bar rule does not use #565f89', () => {
    expect(css).not.toMatch(/\.tab-status-bar\s*\{[^}]*color:\s*#565f89/)
  })

  it('.tab-status-bar__state--off does not use #565f89', () => {
    expect(css).not.toMatch(/\.tab-status-bar__state--off\s*\{[^}]*color:\s*#565f89/)
  })
})

describe('UI-01: settings panel contrast — no failing #565f89 text color', () => {
  it('.settings-panel__body h3 does not use #565f89', () => {
    expect(css).not.toMatch(/\.settings-panel__body\s+h3\s*\{[^}]*color:\s*#565f89/)
  })

  it('.settings-panel__description does not use #565f89', () => {
    expect(css).not.toMatch(/\.settings-panel__description\s*\{[^}]*color:\s*#565f89/)
  })

  it('.settings-panel__empty does not use #565f89', () => {
    expect(css).not.toMatch(/\.settings-panel__empty\s*\{[^}]*color:\s*#565f89/)
  })

  it('.settings-panel__table th does not use #565f89', () => {
    expect(css).not.toMatch(/\.settings-panel__table\s+th\s*\{[^}]*color:\s*#565f89/)
  })

  it('.settings-panel__url does not use #565f89', () => {
    expect(css).not.toMatch(/\.settings-panel__url\s*\{[^}]*color:\s*#565f89/)
  })
})

describe('UI-01: welcome tab contrast — no failing #565f89 text color', () => {
  it('.welcome-tab__version does not use #565f89', () => {
    expect(css).not.toMatch(/\.welcome-tab__version\s*\{[^}]*color:\s*#565f89/)
  })

  it('.welcome-tab__heading does not use #565f89', () => {
    expect(css).not.toMatch(/\.welcome-tab__heading\s*\{[^}]*color:\s*#565f89/)
  })
})

describe('UI-01: modal contrast — no failing #565f89 text color', () => {
  it('.new-session-modal__section-label does not use #565f89', () => {
    expect(css).not.toMatch(/\.new-session-modal__section-label\s*\{[^}]*color:\s*#565f89/)
  })
})

describe('UI-01: intentionally-dim elements preserved', () => {
  it('.tab-status-bar__state--inactive STILL uses #414868 (must NOT be changed to #9aa5ce)', () => {
    expect(css).toMatch(/\.tab-status-bar__state--inactive\s*\{[^}]*color:\s*#414868/)
  })
})
