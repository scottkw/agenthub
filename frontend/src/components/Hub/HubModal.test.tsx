import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// Tests for HubModal (MODAL-01, MODAL-02, MODAL-03 routing).
// HubModal.tsx is fully implemented; the ?raw source-inspection assertions below are GREEN.
// Plus a jsdom behavioral test for the inert focus-trap lifecycle (IN-04 / WR-01 regression guard).
// NOTE: jsdom 29 does NOT implement `inert` focus suppression, but the `inert` property/attribute
// reflection IS observable — the behavioral test asserts on the property, not on focus behavior.

import raw from './HubModal.tsx?raw'

// ---- Mocks ----
// Stub the leaf modals so the behavioral render does not pull in @xterm / relayClient / Wails RPC.
vi.mock('./HubInteractiveModal', () => ({
  HubInteractiveModal: () => React.createElement('div', { 'data-testid': 'interactive-stub' }),
}))
vi.mock('./HubBriefingModal', () => ({
  HubBriefingModal: () => React.createElement('div', { 'data-testid': 'briefing-stub' }),
}))

import { HubModal } from './HubModal'

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

  it('excludes the entering grow animation from the inert trap (Pitfall 3), keeping it through open AND exiting (WR-01)', () => {
    expect(raw).toContain("phase === 'entering'")
  })

  it('queries the .hub background element (Assumption A1 verified selector)', () => {
    expect(raw).toContain("querySelector('.hub')")
  })

  it('close button carries ref={closeBtnRef} for initial focus placement', () => {
    expect(raw).toContain('ref={closeBtnRef}')
  })
})

// ---- Behavioral test: inert focus-trap lifecycle (IN-04 + WR-01 regression guard) ----
// The ?raw source-match tests above verify the inert text exists but NOT the effect's
// lifecycle gating — they would pass even with the WR-01 bug (trap dropped during exit).
// This jsdom behavioral test renders HubModal, drives the phase machine via onAnimationEnd,
// and asserts on the .hub `inert` PROPERTY (jsdom reflects it even though it does not
// implement inert focus suppression).

function makeSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'sess-1',
    cli: 'claude',
    name: 'Test Session',
    state: 'idle',
    status: 'idle',
    createdAt: new Date().toISOString(),
    hostname: '',
    webEnabled: false,
    viewerCount: 0,
    homeDir: false,
    filesWrite: false,
    workDir: '/home/user',
    ...overrides,
  }
}

// React needs this flag for act() to flush effects deterministically under vitest.
;(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true

// jsdom 29 does NOT define `inert` as a reflecting accessor, so before the component
// assigns it the property reads `undefined` (not `false`). The component sets the
// property to `true` on trap, and to `false` on cleanup. So:
//   - never trapped   → undefined (falsy)
//   - trapped         → true
//   - cleaned up      → false
// We assert via truthiness to stay robust to jsdom's reflection model.

describe('HubModal (A11Y-04 behavioral: inert focus-trap lifecycle — WR-01 regression guard)', () => {
  let hubEl: HTMLElement & { inert?: boolean }
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>
  let unmounted = false

  afterEach(() => {
    if (!unmounted) {
      act(() => {
        root.unmount()
      })
    }
    container.remove()
    hubEl.remove()
    unmounted = false
    vi.restoreAllMocks()
  })

  // Two deterministic render modes:
  //   reducedMotion=true  → phase machine initializes directly to 'open' (skips the
  //                         animated grow), so we can observe the open-phase inert trap
  //                         and the cleanup-on-unmount without firing onAnimationEnd.
  //   reducedMotion=false → phase machine starts at 'entering'. A close-button click
  //                         drives handleClose → setPhase('exiting'). (React 19 + jsdom
  //                         do not deliver a hand-dispatched 'animationend' to React's
  //                         synthetic handler, so we never rely on it; clicks ARE
  //                         delivered, which is enough to reach 'exiting'.)
  function renderModal(reducedMotion: boolean) {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: reducedMotion,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
      onchange: null,
    }))

    // The background the trap targets.
    hubEl = document.createElement('div') as HTMLElement & { inert?: boolean }
    hubEl.className = 'hub'
    document.body.appendChild(hubEl)

    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)

    act(() => {
      root.render(
        React.createElement(HubModal, {
          session: makeSession(),
          sourceRect: { left: 0, top: 0, width: 100, height: 100 } as DOMRect,
          relayPort: 51234,
          fontSize: 14,
          theme: {} as never,
          onClose: vi.fn(),
        }),
      )
    })
    return container.querySelector('[role="dialog"]') as HTMLElement
  }

  function phaseOf(dialog: HTMLElement): string {
    return (dialog.className.match(/hub-modal--(entering|open|exiting)/)?.[1]) ?? 'unknown'
  }

  it('background .hub is NOT inert while entering (Pitfall 3: no trap during grow)', () => {
    const dialog = renderModal(false) // animated path → starts at 'entering'
    expect(phaseOf(dialog)).toBe('entering')
    // The effect early-returns during 'entering', so inert is never applied.
    expect(hubEl.inert).toBeFalsy()
  })

  it('background .hub is inert once the modal is open', () => {
    const dialog = renderModal(true) // reduced-motion → starts directly at 'open'
    expect(phaseOf(dialog)).toBe('open')
    expect(hubEl.inert).toBe(true)
  })

  it('background .hub STAYS inert through phase=exiting (WR-01 regression guard)', () => {
    // Animated path: start at 'entering' (inert NOT yet applied), then click close to
    // drive entering → exiting. WR-01's bug gated the trap on `phase !== 'open'`, so the
    // effect early-returned for 'exiting' and the background was left interactive while the
    // modal was still mounted. The fix gates on `phase === 'entering'` instead, so the
    // effect applies inert on 'exiting'. This asserts the post-fix behavior:
    //   pre-fix  → inert stays falsy during 'exiting'  (FAIL → catches the regression)
    //   post-fix → inert is true during 'exiting'      (PASS)
    const dialog = renderModal(false)
    expect(phaseOf(dialog)).toBe('entering')

    act(() => {
      const closeBtn = container.querySelector('.hub-modal__close') as HTMLElement
      closeBtn.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    expect(phaseOf(dialog)).toBe('exiting') // still mounted, exit animation in flight
    expect(hubEl.inert).toBe(true) // background MUST remain trapped during exit
  })

  it('background .hub inert is cleared after unmount (Pitfall 1: no Hub keyboard-lock)', () => {
    renderModal(true) // reduced-motion → 'open' → inert applied
    expect(hubEl.inert).toBe(true)

    act(() => {
      root.unmount()
    })
    unmounted = true
    // Cleanup ran inert = false — no path leaves .hub permanently inert.
    expect(hubEl.inert).toBe(false)
  })

  it('A11Y-01: modal header exposes the session status to AT via a visually-hidden label', () => {
    // UI-REVIEW finding: a screen-reader user inside the dialog could not determine the
    // session status — the header icon is decorative (Heroicons hard-code aria-hidden).
    // Status must be conveyed by a text label (not color — colorblind-safe). The icon stays
    // decorative; an sr-only span carries the STATUS_CONFIG label. Default status 'idle' → 'Idle'.
    renderModal(true) // reduced-motion → 'open'
    const statusIcon = container.querySelector('.hub-modal__status-icon') as HTMLElement
    expect(statusIcon).not.toBeNull()
    expect(statusIcon.getAttribute('aria-hidden')).toBe('true') // icon stays decorative
    const srLabel = container.querySelector('.hub-modal__header .sr-only') as HTMLElement
    expect(srLabel, 'modal header missing AT-readable status label').not.toBeNull()
    expect(srLabel.textContent).toBe('Idle')
  })
})
