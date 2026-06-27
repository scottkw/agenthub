import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// CHAT-FIX-01: source-gate asserting the toggle-relocation CSS rule exists.
// Uses readFileSync (mirrors style.hub.test.ts pattern) to read the stylesheet
// text directly — no DOM rendering required.
const css = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

describe('CHAT-FIX-01: chat toggle relocation rule (chatToggleOverlap)', () => {
  it('(a) stylesheet contains the relocation selector .chat-panel--open ~ .hub-modal__chat-toggle', () => {
    expect(css).toContain('.chat-panel--open ~ .hub-modal__chat-toggle')
  })

  it('(b) relocation rule body sets right: 372px (clears the 360px drawer + 12px gutter)', () => {
    // Capture the rule block for the relocation selector and assert the offset
    // inside it, so the test fails if the rule is removed or the offset regresses.
    const match = css.match(
      /\.chat-panel--open\s*~\s*\.hub-modal__chat-toggle\s*\{([^}]*)\}/
    )
    expect(match).not.toBeNull()
    // The captured block must contain right: 372px (at least 360px — wider than the drawer).
    const ruleBody = match![1]
    expect(ruleBody).toMatch(/right\s*:\s*372px/)
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
})
