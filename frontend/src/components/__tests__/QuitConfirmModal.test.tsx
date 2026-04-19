import { describe, it, expect } from 'vitest'
import raw from '../QuitConfirmModal.tsx?raw'
import appRaw from '../../App.tsx?raw'

describe('QuitConfirmModal source inspection', () => {
  describe('APP-01: modal structure', () => {
    it('uses quit-modal-overlay class', () => {
      expect(raw).toContain('quit-modal-overlay')
    })
    it('uses quit-modal class with stopPropagation', () => {
      expect(raw).toContain('stopPropagation')
    })
    it('handles Escape key via keydown listener', () => {
      expect(raw).toContain('Escape')
    })
    it('sets role dialog and aria-modal', () => {
      expect(raw).toContain('role="dialog"')
      expect(raw).toContain('aria-modal="true"')
    })
    it('sets aria-labelledby pointing to title', () => {
      expect(raw).toContain('aria-labelledby="quit-modal-title"')
    })
    it('has close button with aria-label', () => {
      expect(raw).toContain('aria-label="Close"')
    })
    it('displays modal title "Quit AgentHub?"', () => {
      expect(raw).toContain('Quit AgentHub?')
    })
  })

  describe('APP-02: exit mode buttons', () => {
    it('renders Keep Running button', () => {
      expect(raw).toContain('Keep Running')
    })
    it('renders Quit GUI Only button', () => {
      expect(raw).toContain('Quit GUI Only')
    })
    it('renders Quit Everything button', () => {
      expect(raw).toContain('Quit Everything')
    })
    it('renders quit-modal__btn--cancel class', () => {
      expect(raw).toContain('quit-modal__btn--cancel')
    })
    it('renders quit-modal__btn--quit-gui class', () => {
      expect(raw).toContain('quit-modal__btn--quit-gui')
    })
    it('renders quit-modal__btn--quit-all class', () => {
      expect(raw).toContain('quit-modal__btn--quit-all')
    })
    it('disables buttons after click (acting state)', () => {
      expect(raw).toContain('acting')
      expect(raw).toContain('disabled={acting}')
    })
  })

  describe('APP-03: session list display', () => {
    it('accepts sessions prop', () => {
      expect(raw).toContain('sessions')
    })
    it('renders session name', () => {
      expect(raw).toContain('quit-modal__session-name')
    })
    it('renders session status', () => {
      expect(raw).toContain('quit-modal__session-status')
    })
    it('renders session dot', () => {
      expect(raw).toContain('quit-modal__session-dot')
    })
    it('shows no-sessions text when empty (D-02)', () => {
      expect(raw).toContain('No active sessions')
    })
    it('truncates session list at 5 entries', () => {
      expect(raw).toContain('slice(0, 5)')
    })
    it('shows overflow count', () => {
      expect(raw).toContain('quit-modal__overflow')
    })
  })
})

describe('App.tsx quit-modal wiring (Phase 85)', () => {
  it('subscribes to app:quit-requested event', () => {
    expect(appRaw).toContain("'app:quit-requested'")
  })
  it('defines showQuitModal state', () => {
    expect(appRaw).toContain('showQuitModal')
  })
  it('imports QuitConfirmModal component', () => {
    expect(appRaw).toContain('QuitConfirmModal')
  })
  it('renders QuitConfirmModal in JSX', () => {
    expect(appRaw).toContain('<QuitConfirmModal')
  })
  it('unsubscribes via offQuit in cleanup', () => {
    expect(appRaw).toContain('offQuit')
  })
  it('imports QuitGUIOnly bound method', () => {
    expect(appRaw).toContain('QuitGUIOnly')
  })
  it('imports QuitAll bound method', () => {
    expect(appRaw).toContain('QuitAll')
  })
})
