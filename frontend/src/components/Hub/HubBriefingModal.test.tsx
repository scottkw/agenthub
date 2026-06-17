import { describe, it, expect } from 'vitest'

// Source-inspection tests for HubBriefingModal (MODAL-04).
// Pure ?raw import — NO render(), NO @xterm/xterm.
// These tests are intentionally RED until HubBriefingModal.tsx is implemented in a later plan.

import raw from '../HubBriefingModal.tsx?raw'

describe('HubBriefingModal (MODAL-04: tail fetch + send flow)', () => {
  it('MODAL-04: calls GetSessionTailLines (fetches terminal tail for context display)', () => {
    expect(raw).toContain('GetSessionTailLines')
  })

  it('MODAL-04: uses RelayClient to send input', () => {
    expect(raw).toContain('RelayClient')
  })

  it('MODAL-04: calls sendInput (delivers response text to session)', () => {
    expect(raw).toContain('sendInput')
  })

  it('MODAL-04: sends via onOpen callback (race-safe send — Pitfall 5: never send before WS is open)', () => {
    expect(raw).toContain('onOpen')
  })

  it('MODAL-04: textarea has maxLength={4096} (V5 input-validation guard)', () => {
    expect(raw).toContain('maxLength={4096}')
  })

  it('MODAL-04: Send button disabled when response text is empty (responseText.trim())', () => {
    expect(raw).toContain('responseText.trim()')
  })

  it('MODAL-04: button copy reads "Send Response"', () => {
    expect(raw).toContain('Send Response')
  })
})
