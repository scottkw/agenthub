import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// Phase 141-07 — Comp design-language adoption source gate.
// Pattern: mirrors style.hub.test.ts / style.contrast.test.ts (reads raw CSS text, asserts structure).
// These tests verify that the comp font/radii/type tokens are APPLIED at source level — not merely
// declared as custom properties. A future "token values reverted" or "font-family removed from surface"
// regression cannot silently pass this gate.
//
// User is colorblind: all color and font verification is done at source hex/var level, not by eye.
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

// ─── Font infrastructure: @font-face must be vendored, not CDN ───────────────

describe('Phase 141-07: @font-face — vendored fonts, no CDN', () => {
  it('declares at least 5 @font-face rules for Plus Jakarta Sans + JetBrains Mono', () => {
    const matches = cssRaw.match(/@font-face\s*\{/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(5)
  })

  it('@font-face src URLs reference ./assets/fonts/ (same-origin, not CDN)', () => {
    // All @font-face src lines must use the vendored path
    expect(cssRaw).toContain('./assets/fonts/')
  })

  it('no Google Fonts CDN URL present (fonts.googleapis.com)', () => {
    expect(cssRaw).not.toContain('fonts.googleapis.com')
  })

  it('no Google Static CDN URL present (fonts.gstatic.com)', () => {
    expect(cssRaw).not.toContain('fonts.gstatic.com')
  })
})

// ─── Font tokens: declared in BOTH :root and [data-ui-theme="light"] ─────────

describe('Phase 141-07: font tokens declared in :root (dark theme)', () => {
  it('--hub-font-ui is declared in :root', () => {
    // The :root block comes before the light block
    const rootIdx = cssRaw.indexOf(':root')
    const lightIdx = cssRaw.indexOf('[data-ui-theme="light"]')
    const rootBlock = cssRaw.slice(rootIdx, lightIdx)
    expect(rootBlock).toContain('--hub-font-ui')
  })

  it('--hub-font-mono is declared in :root', () => {
    const rootIdx = cssRaw.indexOf(':root')
    const lightIdx = cssRaw.indexOf('[data-ui-theme="light"]')
    const rootBlock = cssRaw.slice(rootIdx, lightIdx)
    expect(rootBlock).toContain('--hub-font-mono')
  })
})

describe('Phase 141-07: font tokens declared in [data-ui-theme="light"]', () => {
  it('[data-ui-theme="light"] block is present', () => {
    expect(cssRaw).toContain('[data-ui-theme="light"]')
  })

  it('--hub-font-ui is declared inside [data-ui-theme="light"] block', () => {
    const lightIdx = cssRaw.indexOf('[data-ui-theme="light"]')
    expect(lightIdx).toBeGreaterThan(-1)
    // Find the closing brace — the light block is long so look for the next top-level block
    const blockEnd = cssRaw.indexOf('\n}', lightIdx + 1)
    const lightBlock = cssRaw.slice(lightIdx, blockEnd)
    expect(lightBlock).toContain('--hub-font-ui')
  })

  it('--hub-font-mono is declared inside [data-ui-theme="light"] block', () => {
    const lightIdx = cssRaw.indexOf('[data-ui-theme="light"]')
    expect(lightIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('\n}', lightIdx + 1)
    const lightBlock = cssRaw.slice(lightIdx, blockEnd)
    expect(lightBlock).toContain('--hub-font-mono')
  })
})

// ─── Dark palette — comp values, NOT TokyoNight ──────────────────────────────

describe('Phase 141-07: dark palette — comp surface values (not TokyoNight)', () => {
  it('dark --hub-bg is comp value #14151b (not TokyoNight #1a1b26)', () => {
    expect(cssRaw).toContain('--hub-bg: #14151b')
    // TokyoNight bg must NOT appear as a --hub-bg token value
    expect(cssRaw).not.toMatch(/--hub-bg:\s*#1a1b26/)
  })

  it('dark --hub-surface is comp value #16181f', () => {
    expect(cssRaw).toContain('--hub-surface: #16181f')
  })

  it('dark --hub-surface-elevated is comp value #1c1e28', () => {
    expect(cssRaw).toContain('--hub-surface-elevated: #1c1e28')
  })
})

// ─── Locked blue accent + violet rejection ────────────────────────────────────

describe('Phase 141-07: locked blue accent D-05 (colorblind constraint)', () => {
  it('--hub-accent: #7aa2f7 is present (comp blue)', () => {
    expect(cssRaw).toContain('--hub-accent: #7aa2f7')
  })

  it('#7C8CFF (rejected violet accent) is absent', () => {
    // Case-insensitive: reject both uppercase and lowercase
    expect(cssRaw.toLowerCase()).not.toContain('#7c8cff')
  })
})

// ─── Semantic colors ──────────────────────────────────────────────────────────

describe('Phase 141-07: comp semantic color tokens', () => {
  it('--hub-success: #4ade80 (comp green, not TokyoNight #9ece6a)', () => {
    expect(cssRaw).toContain('--hub-success: #4ade80')
  })

  it('--hub-warning: #fbbf24 (comp amber)', () => {
    expect(cssRaw).toContain('--hub-warning: #fbbf24')
  })
})

// ─── Font application: surface selectors must consume the tokens ─────────────

describe('Phase 141-07: ≥8 surface selectors apply var(--hub-font-ui) or var(--hub-font-mono)', () => {
  it('at least 8 font-family declarations use var(--hub-font-ui) or var(--hub-font-mono)', () => {
    const matches = cssRaw.match(/font-family:\s*var\(--hub-font-(ui|mono)\)/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(8)
  })

  it('var(--hub-font-ui) is applied to at least one surface selector', () => {
    expect(cssRaw).toContain('font-family: var(--hub-font-ui)')
  })

  it('var(--hub-font-mono) is applied to at least one code/path/credential context', () => {
    expect(cssRaw).toContain('font-family: var(--hub-font-mono)')
  })

  it('no remaining system-ui literal stack on a chrome surface (no -apple-system or Segoe UI standalone stacks)', () => {
    // After restyle, no chrome selector should still have the old literal UI stack
    // (the token declarations themselves contain 'system-ui' as fallback — those are fine)
    // Verify that standalone font-family properties use the token, not the literal
    // Check that no 'font-family: system-ui' declarations (without var()) exist
    expect(cssRaw).not.toMatch(/font-family:\s*system-ui,\s*-apple-system/)
    expect(cssRaw).not.toMatch(/font-family:\s*-apple-system,\s*BlinkMacSystemFont/)
  })
})

// ─── Radii tokens: consumed by surface selectors ─────────────────────────────

describe('Phase 141-07: comp radii tokens applied to surfaces', () => {
  it('--hub-radius-pill is consumed by at least one selector (filter chips/pills)', () => {
    expect(cssRaw).toContain('border-radius: var(--hub-radius-pill)')
  })

  it('--hub-radius-md is consumed by at least one selector (cards/panels)', () => {
    expect(cssRaw).toContain('border-radius: var(--hub-radius-md)')
  })

  it('--hub-radius-sm is consumed by at least one selector (inputs/buttons)', () => {
    expect(cssRaw).toContain('border-radius: var(--hub-radius-sm)')
  })

  it('--hub-radius-lg is consumed by at least one selector (large modal containers)', () => {
    expect(cssRaw).toContain('border-radius: var(--hub-radius-lg)')
  })
})

// ─── Type scale: heading token consumed ──────────────────────────────────────

describe('Phase 141-07: comp type scale tokens applied', () => {
  it('--hub-font-size-heading is consumed by at least one heading selector', () => {
    expect(cssRaw).toContain('font-size: var(--hub-font-size-heading)')
  })

  it('--hub-font-size-sm is consumed by at least one small-text selector', () => {
    expect(cssRaw).toContain('font-size: var(--hub-font-size-sm)')
  })

  it('--hub-font-weight-emphasis is consumed by at least one emphasis selector', () => {
    expect(cssRaw).toContain('font-weight: var(--hub-font-weight-emphasis)')
  })
})

// ─── D-03 colorblind fences — MUST remain unchanged ──────────────────────────

describe('Phase 141-07: D-03 colorblind-safe fences — agent-badge + status-state', () => {
  it('.tab__agent-badge--claude is still present (D-03 fence — per-agent semantic hex)', () => {
    expect(cssRaw).toContain('.tab__agent-badge--claude')
  })

  it('.tab-status-bar__state--on is still present (D-03 fence — tmux state color)', () => {
    expect(cssRaw).toContain('.tab-status-bar__state--on')
  })

  it('.tab-status-bar__state--off is still present (D-03 fence)', () => {
    expect(cssRaw).toContain('.tab-status-bar__state--off')
  })

  it('.tab-status-bar__state--inactive is still present (D-03 fence)', () => {
    expect(cssRaw).toContain('.tab-status-bar__state--inactive')
  })
})
