/**
 * Phase 168 code review CR-01 fix (168-REVIEW.md) — createTab seeds
 * hubSessions so the footer "Share Session" button works immediately.
 *
 * App.tsx's createTab is not exported and App is not fully mounted in this
 * codebase's test suite (established convention: source-inspection via
 * App.tsx?raw for App-level logic — see App.open-remote.test.tsx,
 * App.wiring.test.tsx, App.createTab.stayOnHub.test.tsx).
 *
 * Bug: openShareModalForActiveSession (wired to StatusBar's onShareSession)
 * resolves the active session purely via `hubSessions.find(s => s.id ===
 * activeId)`. hubSessions was only populated at mount, on daemon-connect,
 * and by the Hub-tab-only 3s poll — never by createTab. A session created
 * via "New Session" (default stayOnHubAfterCreate=OFF auto-switches to its
 * terminal tab, per UX-01) was therefore invisible to hubSessions, and the
 * footer "Share Session" button silently no-op'd for it until the user
 * first visited the Hub tab.
 *
 * Fix: createTab now appends a SessionInfo-shaped entry to hubSessions in
 * its success path, synchronously with tab creation.
 */
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('CR-01 (168-REVIEW): createTab seeds hubSessions — source contract', () => {
  it('createTab calls setHubSessions in its success path (not just setTabs)', () => {
    const idx = raw.indexOf('const createTab = useCallback')
    expect(idx).toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 3200)
    expect(slice).toContain('setTabs((prev) => [...prev, tab])')
    expect(slice).toContain('setHubSessions((prev) => [')
  })

  it('the appended hubSessions entry is keyed by the same sessionId as the new tab', () => {
    const idx = raw.indexOf('const createTab = useCallback')
    const slice = raw.slice(idx, idx + 3200)
    const seedIdx = slice.indexOf('setHubSessions((prev) => [')
    expect(seedIdx).toBeGreaterThan(-1)
    const seedSlice = slice.slice(seedIdx, seedIdx + 500)
    expect(seedSlice).toContain('id: sessionId,')
  })

  it('setHubSessions runs inside the same try block as CreateSession — after the id is known, before the catch', () => {
    const idx = raw.indexOf('const createTab = useCallback')
    const slice = raw.slice(idx, idx + 3200)
    const sessionIdIdx = slice.indexOf('const sessionId = await CreateSession(')
    const seedIdx = slice.indexOf('setHubSessions((prev) => [')
    const catchIdx = slice.indexOf('} catch (err) {')
    expect(sessionIdIdx).toBeGreaterThan(-1)
    expect(seedIdx).toBeGreaterThan(sessionIdIdx)
    expect(catchIdx).toBeGreaterThan(seedIdx)
  })

  it('openShareModalForActiveSession still resolves from hubSessions + activeId (unchanged lookup contract)', () => {
    const idx = raw.indexOf('const openShareModalForActiveSession')
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 400)
    expect(block).toContain('hubSessions.find(')
    expect(block).toContain('activeId')
    expect(block).toContain('setShareModalSession(')
  })
})
