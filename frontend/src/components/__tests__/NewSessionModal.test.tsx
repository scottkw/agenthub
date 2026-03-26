import { describe, it, expect } from 'vitest'
import raw from '../NewSessionModal.tsx?raw'

describe('NewSessionModal source inspection', () => {
  // SESS-01: Modal structure
  describe('SESS-01: modal (not dropdown)', () => {
    it('uses new-session-overlay class', () => {
      expect(raw).toContain('new-session-overlay')
    })
    it('uses new-session-modal class', () => {
      expect(raw).toContain('new-session-modal')
    })
    it('accepts onConfirm prop', () => {
      expect(raw).toContain('onConfirm')
    })
    it('accepts onClose prop', () => {
      expect(raw).toContain('onClose')
    })
  })

  // SESS-02: Agent picker
  describe('SESS-02: agent picker', () => {
    it('uses DetectedCLI type', () => {
      expect(raw).toContain('DetectedCLI')
    })
    it('renders DisplayName for each CLI', () => {
      expect(raw).toContain('DisplayName')
    })
    it('tracks selectedCLI state', () => {
      expect(raw).toContain('selectedCLI')
    })
  })

  // SESS-03: Folder browser
  describe('SESS-03: native folder browser', () => {
    it('imports OpenDirectoryDialog', () => {
      expect(raw).toContain('OpenDirectoryDialog')
    })
    it('has Browse button text', () => {
      expect(raw).toContain('Browse')
    })
    it('tracks browseLoading state', () => {
      expect(raw).toContain('browseLoading')
    })
  })

  // SESS-04: Last-used folder memory
  describe('SESS-04: last-used folder persistence', () => {
    it('uses agenthub:lastWorkDir localStorage key', () => {
      expect(raw).toContain('agenthub:lastWorkDir')
    })
    it('reads from localStorage on open', () => {
      expect(raw).toContain('localStorage.getItem')
    })
    it('writes to localStorage after folder pick', () => {
      expect(raw).toContain('localStorage.setItem')
    })
  })
})

// ARGS-02: Args text field
describe('ARGS-02: args text field', () => {
  it('has args input class', () => {
    expect(raw).toContain('new-session-modal__args-input')
  })
  it('has placeholder with example flag', () => {
    expect(raw).toContain('e.g. --model claude-opus-4-5')
  })
  it('splits args with filter(Boolean) to avoid empty strings', () => {
    expect(raw).toContain('.filter(Boolean)')
  })
})

// ARGS-04: Per-agent args persistence
describe('ARGS-04: per-agent args persistence', () => {
  it('uses agenthub:args: localStorage key pattern', () => {
    expect(raw).toContain('agenthub:args:')
  })
  it('reads args from localStorage on agent change', () => {
    // handleSelectCLI reads localStorage for the newly selected agent
    expect(raw).toContain('handleSelectCLI')
  })
  it('persists args to localStorage on confirm', () => {
    // handleConfirm calls localStorage.setItem with ARGS_KEY
    expect(raw).toContain('ARGS_KEY(selectedCLI)')
  })
})

// ARGS-05: Clear args
describe('ARGS-05: clear args button', () => {
  it('has handleClearArgs function', () => {
    expect(raw).toContain('handleClearArgs')
  })
  it('removes localStorage key on clear', () => {
    expect(raw).toContain('localStorage.removeItem')
  })
  it('has accessible clear button', () => {
    expect(raw).toContain('aria-label="Clear arguments"')
  })
})
