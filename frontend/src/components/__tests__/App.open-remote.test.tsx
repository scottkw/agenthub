// Wave 0 RED tests for Phase 146 FIX-03 — handleOpenRemoteSession contract.
//
// Source-inspection tests asserting that App.tsx's handleOpenRemoteSession
// implements the exchange-then-open flow after Wave 2 rewrites it.
//
// Mounting the real <App /> requires stubbing ~30 wailsjs imports + xterm;
// source-inspection via App.tsx?raw is the established pattern (see
// App.fileBrowserMode.test.tsx and App.wiring.test.tsx).
//
// These tests are intentionally RED until Wave 2 rewrites handleOpenRemoteSession
// to call ExchangeJoinCodeAtURL instead of bare BrowserOpenURL.

import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App.tsx — Phase 146 handleOpenRemoteSession (FIX-03)', () => {
  // ─── D-01: Exchange-then-open flow ────────────────────────────────────────
  // The handler must call ExchangeJoinCodeAtURL (not a bare BrowserOpenURL on
  // session.url), so the viewer gets a scoped cap token rather than a 401.
  it('calls ExchangeJoinCodeAtURL (not BrowserOpenURL(session.url) directly) (D-01)', () => {
    const idx = raw.indexOf('handleOpenRemoteSession')
    expect(idx, 'handleOpenRemoteSession must be present in App.tsx').toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 900)
    expect(slice).toContain('ExchangeJoinCodeAtURL')
  })

  // ─── D-06: RW preference when viewer is the peer owner ────────────────────
  // When the viewer's own tailscale node matches the session's peer hostname,
  // prefer rwJoinCode so the owner gets full read+write access.
  it('selects rwJoinCode when isPeerSelf to give owner read+write access (D-06)', () => {
    expect(raw).toContain('rwJoinCode')
    expect(raw).toContain('isPeerSelf')
  })

  // ─── D-03: Error banner when session is not shared ─────────────────────────
  // If roJoinCode is absent (owner has not enabled sharing), the handler must
  // show an informative error banner via setSaveBanner — not silently fail or
  // open a 401 page.
  it('shows setSaveBanner error when roJoinCode is absent (D-03 not-shared path)', () => {
    expect(raw).toMatch(/roJoinCode.*setSaveBanner|setSaveBanner.*roJoinCode/s)
  })

  // ─── Pitfall 4: Expired / session-gone error handling ─────────────────────
  // ExchangeJoinCodeAtURL may throw if the join code expired or the session
  // was killed. The handler must catch this and surface an informative banner
  // (not an unhandled rejection that leaves the user stuck).
  it('handles expired/session-gone exchange errors with an informative banner (Pitfall 4)', () => {
    const handlerStart = raw.indexOf('handleOpenRemoteSession')
    expect(handlerStart, 'handleOpenRemoteSession declaration must be present').toBeGreaterThan(-1)
    // Slice 1200 chars — enough to capture the full async handler body.
    const slice = raw.slice(handlerStart, handlerStart + 1200)
    // Handler must reference 'expired' or 'session-gone' for error classification.
    expect(slice).toMatch(/expired|session-gone/)
    // Handler must call setSaveBanner on exchange failure.
    expect(slice).toContain('setSaveBanner')
  })

  // ─── URL shape: cap-bearing URL ───────────────────────────────────────────
  // The handler constructs the final URL as baseURL + '/sessions/' + id + '?cap=' + token.
  // Asserting the pattern is present (not the literal values) so the test is
  // resilient to formatting changes.
  it('constructs a cap-bearing URL matching /sessions/{id}?cap=TOKEN', () => {
    expect(raw).toMatch(/\/sessions\/.*\?cap=/)
  })
})
