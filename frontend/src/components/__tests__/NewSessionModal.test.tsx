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
