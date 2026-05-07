import { describe, it, expect } from 'vitest'

describe('Phase 97 SER-01: sanitizeFilename() — Plan 97-02 implements', () => {
  it('rejects path traversal (../../etc/passwd)', () => {
    expect.fail('RED scaffold — Plan 97-02 implements src/lib/sanitizeFilename.ts (97-VALIDATION row SER-01 sanitizeFilename handles path traversal)')
  })
  it('returns "session" for empty input', () => {
    expect.fail('RED scaffold — Plan 97-02 implements empty-input fallback (97-PATTERNS §sanitizeFilename Pattern 4)')
  })
  it('returns "session" for leading-dot hidden files (.bashrc)', () => {
    expect.fail('RED scaffold — Plan 97-02 implements leading-dot guard')
  })
  it('returns "session" for Windows reserved names (CON/PRN/AUX/NUL/COM1/LPT1)', () => {
    expect.fail('RED scaffold — Plan 97-02 implements Windows reserved-name guard')
  })
  it('collapses whitespace runs to underscore (my session  name → my_session_name)', () => {
    expect.fail('RED scaffold — Plan 97-02 implements whitespace collapse')
  })
  it('preserves allowed characters (agent-1.txt → agent-1.txt)', () => {
    expect.fail('RED scaffold — Plan 97-02 implements allowed-character preservation [\\w\\-.]')
  })
})
