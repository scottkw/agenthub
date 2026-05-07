import { describe, it, expect } from 'vitest'
import { sanitizeFilename } from '../sanitizeFilename'

describe('Phase 97 SER-01: sanitizeFilename() — Plan 97-02 implementation', () => {
  it('rejects path traversal (../../etc/passwd → no slashes, no dots-prefix-traversal)', () => {
    const out = sanitizeFilename('../../etc/passwd')
    // No slashes, no upward-traversal sequences. Slashes become underscores.
    expect(out).not.toContain('/')
    expect(out).not.toContain('\\')
    // Does NOT start with .. (would be hidden / unusable)
    expect(out.startsWith('..')).toBe(false)
  })

  it('returns "session" for empty input', () => {
    expect(sanitizeFilename('')).toBe('session')
    expect(sanitizeFilename('   ')).toBe('session')
    expect(sanitizeFilename('\t\n')).toBe('session')
  })

  it('returns "session" for leading-dot hidden files', () => {
    expect(sanitizeFilename('.bashrc')).toBe('session')
    expect(sanitizeFilename('.hidden')).toBe('session')
    expect(sanitizeFilename('.')).toBe('session')
  })

  it('returns "session" for Windows reserved names (CON/PRN/AUX/NUL/COM1/LPT1, case-insensitive)', () => {
    expect(sanitizeFilename('CON')).toBe('session')
    expect(sanitizeFilename('con')).toBe('session')
    expect(sanitizeFilename('NUL')).toBe('session')
    expect(sanitizeFilename('COM1')).toBe('session')
    expect(sanitizeFilename('com9')).toBe('session')
    expect(sanitizeFilename('LPT1')).toBe('session')
    expect(sanitizeFilename('lpt9')).toBe('session')
    expect(sanitizeFilename('AUX')).toBe('session')
    expect(sanitizeFilename('PRN')).toBe('session')
  })

  it('collapses whitespace runs to underscore', () => {
    expect(sanitizeFilename('my session  name')).toBe('my_session_name')
    expect(sanitizeFilename('  trim me  ')).toBe('trim_me')
    expect(sanitizeFilename('a\tb\nc')).toBe('a_b_c')
  })

  it('preserves allowed characters [\\w\\-.] and replaces others with underscore', () => {
    expect(sanitizeFilename('agent-1.txt')).toBe('agent-1.txt')
    expect(sanitizeFilename('claude_code-2026')).toBe('claude_code-2026')
    expect(sanitizeFilename('weird/name:with*chars')).toBe('weird_name_with_chars')
  })
})
