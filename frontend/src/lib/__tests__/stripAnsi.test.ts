import { describe, it, expect } from 'vitest'
import { stripAnsi } from '../stripAnsi'

describe('Phase 97 SER-01: stripAnsi() — Plan 97-02 implementation', () => {
  it('strips SGR (Select Graphic Rendition) sequences', () => {
    // Bold + red foreground: \x1b[1;31m ... \x1b[0m
    const input = '\x1b[1;31mError\x1b[0m: something failed'
    expect(stripAnsi(input)).toBe('Error: something failed')
  })

  it('strips ECH (erase character) sequences', () => {
    // Erase 5 characters: \x1b[5X
    const input = 'before\x1b[5Xafter'
    expect(stripAnsi(input)).toBe('beforeafter')
  })

  it('strips cursor-move sequences (CUF/CUB/CUU/CUD)', () => {
    // CUF (forward), CUB (backward), CUU (up), CUD (down)
    const input = 'a\x1b[3Cb\x1b[2Dc\x1b[1Ad\x1b[4Be'
    expect(stripAnsi(input)).toBe('abcde')
  })

  it('strips DEC private mode sequences (\\x1b[?25h, \\x1b[?7l)', () => {
    const input = '\x1b[?25hcontent\x1b[?7l'
    expect(stripAnsi(input)).toBe('content')
  })

  it('preserves plain text without escape sequences', () => {
    expect(stripAnsi('plain text')).toBe('plain text')
    expect(stripAnsi('')).toBe('')
    expect(stripAnsi('multi\nline\ttext')).toBe('multi\nline\ttext')
  })

  it('round-trips a serializeAddon-style fixture (mixed SGR + cursor + ECH + plain)', () => {
    // Fixture mimicking serialize({ excludeModes: true }) output — a mix
    // of color resets, cursor moves, ECH erasures, and printable text.
    const fixture =
      '\x1b[0m\x1b[38;2;255;128;0mAgentHub\x1b[0m \x1b[1;32m$\x1b[0m ' +
      'echo hello\x1b[3C\x1b[2X\x1b[?25h\nhello\x1b[0m'
    const out = stripAnsi(fixture)
    expect(out).toBe('AgentHub $ echo hello\nhello')
    // Defensive: no \x1b bytes remain.
    expect(out).not.toContain('\x1b')
  })
})
