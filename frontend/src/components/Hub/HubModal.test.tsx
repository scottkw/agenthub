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

  it('MODAL-02: uses stopPropagation (scoped, not global) for Escape on dialog element', () => {
    expect(raw).toContain('stopPropagation')
  })

  it('MODAL-02: does NOT use stopImmediatePropagation (WR-05 fix — scoped handler replaces global guard)', () => {
    expect(raw).not.toContain('stopImmediatePropagation')
  })

  it('MODAL-02: Escape handled via onKeyDown on dialog element, not document.addEventListener', () => {
    expect(raw).not.toMatch(/document\.addEventListener\s*\(\s*['"]keydown/)
  })

  it('MODAL-02: uses cardFocusRef for focus-return on unmount', () => {
    expect(raw).toContain('cardFocusRef')
  })
})

describe('HubModal (GAP-134-D: reduced-motion close does not depend on onAnimationEnd)', () => {
  // Under prefers-reduced-motion the CSS disables animations, so onAnimationEnd never
  // fires. The phase machine must detect reduced motion and skip the animated phases,
  // otherwise the modal can never close.
  it('detects prefers-reduced-motion via matchMedia', () => {
    expect(raw).toContain('prefers-reduced-motion: reduce')
    expect(raw).toContain('matchMedia')
  })

  it('handleClose closes synchronously when reduced motion is preferred', () => {
    expect(raw).toMatch(/prefersReducedMotion[\s\S]{0,80}onClose\(\)/)
  })
})

describe('HubModal (GAP-134-C: origin marker uses provenance, not hostname)', () => {
  // Local sessions carry the machine os.Hostname(); a hostname-based isLocal check
  // mislabels them with the globe icon + machine name. The colorblind-safe origin cue
  // (computer vs globe icon) must be driven by the provenance `remote` prop.
  it('isLocal is derived from the remote prop', () => {
    expect(raw).toContain('const isLocal = !remote')
  })

  it('isLocal is NOT derived from session.hostname', () => {
    expect(raw).not.toContain("const isLocal = !session.hostname")
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

describe('HubModal (A11Y-01: STATUS_CONFIG mirrors SessionCard — colorblind-safe at source)', () => {
  // Verify at source that STATUS_CONFIG contains each of the 6 status keys with
  // unique icon + text label. Color is reinforcement only (colorblind-safe).
  // Verified at source level (hex constants in code), NEVER by eye.
  it('A11Y-01: STATUS_CONFIG contains running status with "Running" label', () => {
    expect(raw).toContain("running:")
    expect(raw).toContain("'Running'")
  })

  it('A11Y-01: STATUS_CONFIG contains idle status with "Idle" label', () => {
    expect(raw).toContain("idle:")
    expect(raw).toContain("'Idle'")
  })

  it('A11Y-01: STATUS_CONFIG contains waiting status with "Needs input" label', () => {
    expect(raw).toContain("waiting:")
    expect(raw).toContain("'Needs input'")
  })

  it('A11Y-01: STATUS_CONFIG contains errored status with "Error" label', () => {
    expect(raw).toContain("errored:")
    expect(raw).toContain("'Error'")
  })

  it('A11Y-01: STATUS_CONFIG contains stopped-ok status with "Done" label', () => {
    expect(raw).toContain("'stopped-ok':")
    expect(raw).toContain("'Done'")
  })

  it('A11Y-01: STATUS_CONFIG contains stopped-err status with "Exited" label (text-differentiator from errored)', () => {
    expect(raw).toContain("'stopped-err':")
    expect(raw).toContain("'Exited'")
  })
})

describe('HubModal (A11Y-04: focus trap via inert)', () => {
  // jsdom 29 does NOT implement inert focus suppression (element.inert returns undefined).
  // ALL assertions here are ?raw source-inspection only — no DOM focus-behavior tests.

  it('sets hubEl.inert = true when phase is open', () => {
    expect(raw).toContain('.inert = true')
  })

  it('removes hubEl.inert on cleanup (Pitfall 1 guard — prevents Hub keyboard-lock)', () => {
    expect(raw).toContain('.inert = false')
  })

  it('moves focus to closeBtnRef on open (WCAG 2.4.3: focus order)', () => {
    expect(raw).toContain('closeBtnRef')
    expect(raw).toContain('closeBtnRef.current?.focus()')
  })

  it('gates inert trap on phase === "open" (not during entering animation — Pitfall 3)', () => {
    expect(raw).toContain("phase !== 'open'")
  })

  it('queries the .hub background element (Assumption A1 verified selector)', () => {
    expect(raw).toContain("querySelector('.hub')")
  })

  it('close button carries ref={closeBtnRef} for initial focus placement', () => {
    expect(raw).toContain('ref={closeBtnRef}')
  })
})
