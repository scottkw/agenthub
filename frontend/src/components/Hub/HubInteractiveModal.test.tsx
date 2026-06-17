import { describe, it, expect } from 'vitest'

// Source-inspection tests for HubInteractiveModal (MODAL-03, MODAL-05).
// Pure ?raw import — NO render(), NO @xterm/xterm.
// These tests are intentionally RED until HubInteractiveModal.tsx is implemented in a later plan.

import raw from './HubInteractiveModal.tsx?raw'

describe('HubInteractiveModal (MODAL-03: TerminalPanel mounting)', () => {
  it('MODAL-03: mounts TerminalPanel inside the modal body', () => {
    expect(raw).toContain('TerminalPanel')
  })

  it('MODAL-03: passes sessionId prop to TerminalPanel', () => {
    expect(raw).toContain('sessionId=')
  })

  it('MODAL-03: passes relayPort prop to TerminalPanel', () => {
    expect(raw).toContain('relayPort=')
  })

  it('MODAL-03: passes theme prop to TerminalPanel', () => {
    expect(raw).toContain('theme=')
  })

  it('MODAL-03: passes pluginConfig prop to TerminalPanel', () => {
    expect(raw).toContain('pluginConfig=')
  })
})

describe('HubInteractiveModal (MODAL-05: isActive timing guard)', () => {
  it('MODAL-05: passes isActive prop to TerminalPanel', () => {
    expect(raw).toContain('isActive=')
  })

  it('MODAL-05: isActive is bound to the open phase (prevents 0-column layout during grow animation — Pitfall 1)', () => {
    // isActive must be gated on phase === 'open', not unconditionally true
    expect(raw).toMatch(/isActive=\{[^}]*open/)
  })
})
