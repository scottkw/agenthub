import { describe, it, expect } from 'vitest'

// Source-inspection tests for HubModal (MODAL-01, MODAL-02, MODAL-03 routing).
// Pure ?raw import — NO render(), NO @xterm/xterm.
// These tests are intentionally RED until HubModal.tsx is implemented in a later plan.

import raw from './HubModal.tsx?raw'

describe('HubModal (MODAL-01: dialog accessibility contract)', () => {
  it('MODAL-01: uses role="dialog"', () => {
    expect(raw).toContain('role="dialog"')
  })

  it('MODAL-01: uses aria-modal="true"', () => {
    expect(raw).toContain('aria-modal="true"')
  })

  it('MODAL-01: inline style sets transformOrigin (grow-origin for animation)', () => {
    expect(raw).toContain('transformOrigin')
  })
})

describe('HubModal (MODAL-02: keyboard + focus management)', () => {
  it('MODAL-02: handles Escape key', () => {
    expect(raw).toContain('Escape')
  })

  it('MODAL-02: calls stopImmediatePropagation (Pitfall 6 guard — prevents Hub card Escape double-fire)', () => {
    expect(raw).toContain('stopImmediatePropagation')
  })

  it('MODAL-02: uses cardFocusRef for focus-return on unmount', () => {
    expect(raw).toContain('cardFocusRef')
  })
})

describe('HubModal (MODAL-03: attention-based routing)', () => {
  it('MODAL-03: renders HubBriefingModal branch (attention=true path)', () => {
    expect(raw).toContain('HubBriefingModal')
  })

  it('MODAL-03: renders HubInteractiveModal branch (attention=false path)', () => {
    expect(raw).toContain('HubInteractiveModal')
  })

  it('MODAL-03: routing predicate uses isAttentionStatus', () => {
    expect(raw).toContain('isAttentionStatus')
  })
})
