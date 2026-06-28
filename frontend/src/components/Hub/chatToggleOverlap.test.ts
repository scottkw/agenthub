import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// CHAT-FIX-01 + CHAT-LAYOUT-02: source-gate asserting the toggle-relocation CSS rule exists
// and that the offset now tracks the live drawer width via --chat-panel-width.
// Uses readFileSync (mirrors style.hub.test.ts pattern) to read the stylesheet
// text directly — no DOM rendering required.
const css = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

describe('CHAT-FIX-01 / CHAT-LAYOUT-02: chat toggle relocation rule (chatToggleOverlap)', () => {
  it('(a) stylesheet contains the relocation selector .chat-panel--open ~ .hub-modal__chat-toggle', () => {
    expect(css).toContain('.chat-panel--open ~ .hub-modal__chat-toggle')
  })

  it('(b) relocation rule offset tracks --chat-panel-width (CHAT-LAYOUT-02 — no longer hard-coded 372px)', () => {
    // Capture the rule block for the relocation selector and assert the offset
    // references --chat-panel-width so it tracks the live resizable drawer width.
    // CHAT-LAYOUT-02 replaced the hard-coded right:372px with:
    //   right: calc(var(--chat-panel-width, 360px) + 12px)
    const match = css.match(
      /\.chat-panel--open\s*~\s*\.hub-modal__chat-toggle\s*\{([^}]*)\}/
    )
    expect(match).not.toBeNull()
    const ruleBody = match![1]
    // The offset must reference the CSS custom property, not a bare px literal.
    expect(ruleBody).toContain('--chat-panel-width')
    // It must still be a right-edge offset (right: ...).
    expect(ruleBody).toMatch(/right\s*:/)
  })

  it('(c) base .hub-modal__chat-toggle rule still sets right: 12px (closed-state unchanged)', () => {
    // The base rule (without the sibling combinator) must still have right: 12px so
    // the closed-state position is preserved.
    const baseMatch = css.match(
      /(?<!~\s*)\.hub-modal__chat-toggle\s*\{([^}]*)\}/
    )
    expect(baseMatch).not.toBeNull()
    const baseBody = baseMatch![1]
    expect(baseBody).toMatch(/right\s*:\s*12px/)
  })

  it('(d) base .hub-modal__chat-toggle rule declares a transition on right (slide sync with drawer)', () => {
    // The toggle must animate its right offset so it glides in lockstep with the
    // opening/closing drawer (transition: right 220ms ease-out) instead of teleporting.
    // Reuses the base-rule capture from test (c).
    const baseMatch = css.match(
      /(?<!~\s*)\.hub-modal__chat-toggle\s*\{([^}]*)\}/
    )
    expect(baseMatch).not.toBeNull()
    const baseBody = baseMatch![1]
    expect(baseBody).toMatch(/transition\s*:\s*right\b/)
  })
})
