import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// Phase 134 — Hub modal CSS contract tests.
// Pattern: mirrors style.hub.test.ts (reads raw CSS text, block-finder helper for structural assertions).
// These tests are intentionally RED until modal CSS is implemented in a later plan.
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

describe('Hub modal overlay (MODAL-01: overlay z-index contract)', () => {
  it('.hub-modal-overlay block contains position: fixed', () => {
    const idx = cssRaw.indexOf('.hub-modal-overlay')
    expect(idx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', idx)
    const block = cssRaw.slice(idx, blockEnd)
    expect(block).toContain('position: fixed')
  })

  it('.hub-modal-overlay block contains inset: 0', () => {
    const idx = cssRaw.indexOf('.hub-modal-overlay')
    expect(idx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', idx)
    const block = cssRaw.slice(idx, blockEnd)
    expect(block).toContain('inset: 0')
  })

  it('.hub-modal-overlay block contains z-index: 200', () => {
    const idx = cssRaw.indexOf('.hub-modal-overlay')
    expect(idx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', idx)
    const block = cssRaw.slice(idx, blockEnd)
    expect(block).toContain('z-index: 200')
  })
})

describe('Hub modal panel (MODAL-01: token-only, no hardcoded hex)', () => {
  it('.hub-modal block contains var(--hub-surface-elevated)', () => {
    const idx = cssRaw.indexOf('.hub-modal {')
    expect(idx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', idx)
    const block = cssRaw.slice(idx, blockEnd)
    expect(block).toContain('var(--hub-surface-elevated)')
  })

  it('.hub-modal block contains var(--hub-border)', () => {
    const idx = cssRaw.indexOf('.hub-modal {')
    expect(idx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', idx)
    const block = cssRaw.slice(idx, blockEnd)
    expect(block).toContain('var(--hub-border)')
  })

  it('.hub-modal { declaration block contains NO hardcoded hex values (token-only rule)', () => {
    const idx = cssRaw.indexOf('.hub-modal {')
    expect(idx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', idx)
    const block = cssRaw.slice(idx, blockEnd)
    // box-shadow uses rgba() which is permitted; only plain hex is forbidden
    expect(block).not.toMatch(/#[0-9a-fA-F]{3,6}(?![0-9a-fA-F])/)
  })
})

describe('Hub modal keyframes (animation declarations)', () => {
  it('@keyframes hub-modal-grow is declared', () => {
    expect(cssRaw).toContain('@keyframes hub-modal-grow')
  })

  it('@keyframes hub-modal-shrink is declared', () => {
    expect(cssRaw).toContain('@keyframes hub-modal-shrink')
  })

  it('@keyframes hub-modal-overlay-in is declared', () => {
    expect(cssRaw).toContain('@keyframes hub-modal-overlay-in')
  })

  it('@keyframes hub-modal-overlay-out is declared', () => {
    expect(cssRaw).toContain('@keyframes hub-modal-overlay-out')
  })
})

describe('Hub modal animation guard (MODAL-01: reduced-motion compliance)', () => {
  it('.hub-modal--entering animation assignment lives inside prefers-reduced-motion: no-preference guard', () => {
    const mediaIdx = cssRaw.indexOf('prefers-reduced-motion: no-preference')
    expect(mediaIdx).toBeGreaterThan(-1)
    const enterIdx = cssRaw.indexOf('hub-modal--entering')
    expect(enterIdx).toBeGreaterThan(mediaIdx)
  })

  it('prefers-reduced-motion: reduce block sets animation: none on .hub-modal', () => {
    const reduceIdx = cssRaw.lastIndexOf('prefers-reduced-motion: reduce')
    expect(reduceIdx).toBeGreaterThan(-1)
    // Grab enough text to cover the full block (nested rules may span multiple }s)
    const block = cssRaw.slice(reduceIdx, reduceIdx + 300)
    expect(block).toContain('animation: none')
  })
})

describe('Hub modal send button (MODAL-04: accent token)', () => {
  it('.hub-modal__send-btn block contains var(--hub-accent)', () => {
    const idx = cssRaw.indexOf('.hub-modal__send-btn')
    expect(idx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', idx)
    const block = cssRaw.slice(idx, blockEnd)
    expect(block).toContain('var(--hub-accent)')
  })
})

describe('Hub modal tail section (MODAL-04: preview background token)', () => {
  it('.hub-modal__tail block contains var(--hub-preview-bg)', () => {
    const idx = cssRaw.indexOf('.hub-modal__tail')
    expect(idx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', idx)
    const block = cssRaw.slice(idx, blockEnd)
    expect(block).toContain('var(--hub-preview-bg)')
  })
})
