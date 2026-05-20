// Phase 120-06 Task 1 — unit tests for webMode detection.
//
// Tests construct synthetic Location-like objects and pass them to the
// pure functions (detectMode / readWebModeParams) rather than mutating
// window.location. This keeps the suite hermetic — no global state leaks
// between cases. See frontend/src/lib/webMode.ts for the contract.

import { describe, it, expect } from 'vitest'
import { detectMode, readWebModeParams } from '../webMode'

// Helper — build a Location-like literal with just the fields the module reads.
function loc(pathname: string, search: string = ''): Location {
  return { pathname, search } as unknown as Location
}

describe('detectMode', () => {
  it('returns "web" when pathname starts with "/app/"', () => {
    expect(detectMode(loc('/app/'))).toBe('web')
  })

  it('returns "web" when pathname is exactly "/app" (no trailing slash)', () => {
    expect(detectMode(loc('/app'))).toBe('web')
  })

  it('returns "web" for nested paths under /app/', () => {
    expect(detectMode(loc('/app/foo'))).toBe('web')
    expect(detectMode(loc('/app/nested/deep'))).toBe('web')
  })

  it('returns "desktop" for the Wails root path "/"', () => {
    expect(detectMode(loc('/'))).toBe('desktop')
  })

  it('returns "desktop" for unrelated paths', () => {
    expect(detectMode(loc('/sessions/x'))).toBe('desktop')
    expect(detectMode(loc('/api/files/list'))).toBe('desktop')
    expect(detectMode(loc('/static/asset.js'))).toBe('desktop')
  })

  it('returns "desktop" for paths that merely contain "/app/" but do not start with it', () => {
    expect(detectMode(loc('/foo/app/bar'))).toBe('desktop')
  })

  it('returns "desktop" for the lookalike "/apps" (only "/app" + boundary counts)', () => {
    expect(detectMode(loc('/apps'))).toBe('desktop')
    expect(detectMode(loc('/apps/foo'))).toBe('desktop')
  })
})

describe('readWebModeParams', () => {
  it('returns { sessionId: null, capToken: null } when search is empty', () => {
    expect(readWebModeParams(loc('/app/', ''))).toEqual({ sessionId: null, capToken: null })
  })

  it('parses both session and cap when present', () => {
    const out = readWebModeParams(loc('/app/', '?session=sess-1&cap=tok-1'))
    expect(out).toEqual({ sessionId: 'sess-1', capToken: 'tok-1' })
  })

  it('returns partial when only session is present', () => {
    const out = readWebModeParams(loc('/app/', '?session=sess-2'))
    expect(out).toEqual({ sessionId: 'sess-2', capToken: null })
  })

  it('returns partial when only cap is present', () => {
    const out = readWebModeParams(loc('/app/', '?cap=tok-3'))
    expect(out).toEqual({ sessionId: null, capToken: 'tok-3' })
  })

  it('maps empty-string params to null (does not return empty strings)', () => {
    const out = readWebModeParams(loc('/app/', '?session=&cap='))
    expect(out).toEqual({ sessionId: null, capToken: null })
  })

  it('trims whitespace and treats whitespace-only as null', () => {
    const out = readWebModeParams(loc('/app/', '?session=%20%20&cap=%20'))
    expect(out).toEqual({ sessionId: null, capToken: null })
  })

  it('does not URL-decode beyond what URLSearchParams already does', () => {
    // URLSearchParams decodes percent-encoded characters; we don't double-decode.
    const out = readWebModeParams(loc('/app/', '?session=a%20b&cap=c%2Bd'))
    expect(out.sessionId).toBe('a b')
    expect(out.capToken).toBe('c+d')
  })
})
