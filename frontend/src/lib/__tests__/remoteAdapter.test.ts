// Phase 146 FIX-03 — remoteAdapter.ts tests.
//
// Tests for adaptRemoteSession and adaptAllRemoteSessions, focusing on the
// Phase 146 contract: ro_join_code / rw_join_code pass-through.
//
// Wave 0 RED: the roJoinCode/rwJoinCode pass-through tests are RED until Wave 2
// extends AdaptedRemoteSessionInfo + adaptRemoteSession to carry the new fields.
// The existing structure/url tests below are GREEN from the start (they test
// already-implemented behavior).

import { describe, it, expect } from 'vitest'
import { adaptRemoteSession, adaptAllRemoteSessions } from '../remoteAdapter'
import type { RemotePeerSessions, RemoteSession } from '../remoteSession'

// ─── Test factories ──────────────────────────────────────────────────────────

function makePeer(overrides?: Partial<RemotePeerSessions>): RemotePeerSessions {
  return {
    hostname: 'peer-host',
    reachable: true,
    sessions: [],
    ...overrides,
  }
}

function makeSession(overrides?: Partial<RemoteSession> & { roJoinCode?: string; rwJoinCode?: string }): RemoteSession & { roJoinCode?: string; rwJoinCode?: string } {
  const { roJoinCode, rwJoinCode, ...rest } = overrides ?? {}
  const base: RemoteSession = {
    id: 'sess-x',
    name: 'Test Session',
    cliType: 'claude',
    status: 'running',
    url: 'https://peer-host.ts.net:9443/sessions/sess-x',
    ...rest,
  }
  const result: RemoteSession & { roJoinCode?: string; rwJoinCode?: string } = { ...base }
  if (roJoinCode !== undefined) result.roJoinCode = roJoinCode
  if (rwJoinCode !== undefined) result.rwJoinCode = rwJoinCode
  return result
}

// ─── Existing contract (GREEN from the start) ────────────────────────────────

describe('adaptRemoteSession — existing contract', () => {
  it('maps id, name, cli, status from the remote session', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.id).toBe('sess-x')
    expect(adapted.name).toBe('Test Session')
    expect(adapted.cli).toBe('claude')
    expect(adapted.status).toBe('running')
  })

  it('carries the peer hostname so Hub shows globe icon + hostname', () => {
    const adapted = adaptRemoteSession(makePeer({ hostname: 'remote-mac' }), makeSession())
    expect(adapted.hostname).toBe('remote-mac')
  })

  it('carries the session url (CR-01 pass-through)', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.url).toBe('https://peer-host.ts.net:9443/sessions/sess-x')
  })

  it('sets webEnabled=true for remote sessions', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.webEnabled).toBe(true)
  })
})

// ─── Phase 146 pass-through (RED until Wave 2) ──────────────────────────────
//
// RED: adaptRemoteSession does not yet pass roJoinCode/rwJoinCode through
// (AdaptedRemoteSessionInfo type doesn't carry those fields yet). Wave 2 will:
//   1. Add roJoinCode?/rwJoinCode? to AdaptedRemoteSessionInfo
//   2. Pass them through in adaptRemoteSession

describe('adaptRemoteSession — Phase 146 join-code pass-through (RED)', () => {
  it('passes roJoinCode through from session when present (Phase 146)', () => {
    const session = makeSession({ roJoinCode: 'ro-abc', rwJoinCode: 'rw-xyz' })
    const adapted = adaptRemoteSession(makePeer(), session)
    // Cast to access the not-yet-typed field — Wave 2 will add it to AdaptedRemoteSessionInfo.
    expect((adapted as unknown as { roJoinCode?: string }).roJoinCode).toBe('ro-abc')
    expect((adapted as unknown as { rwJoinCode?: string }).rwJoinCode).toBe('rw-xyz')
  })

  it('roJoinCode is undefined when session has no join code (D-03 not-shared path)', () => {
    const session = makeSession() // no roJoinCode / rwJoinCode fields
    const adapted = adaptRemoteSession(makePeer(), session)
    // When the peer hasn't enabled sharing, these fields must be absent.
    expect((adapted as unknown as { roJoinCode?: string }).roJoinCode).toBeUndefined()
    expect((adapted as unknown as { rwJoinCode?: string }).rwJoinCode).toBeUndefined()
  })
})

// ─── adaptAllRemoteSessions — filters unreachable peers ─────────────────────

describe('adaptAllRemoteSessions', () => {
  it('excludes sessions from unreachable peers', () => {
    const peers: RemotePeerSessions[] = [
      { hostname: 'a', reachable: true, sessions: [makeSession({ id: 'sess-1' })] },
      { hostname: 'b', reachable: false, sessions: [makeSession({ id: 'sess-2' })] },
    ]
    const result = adaptAllRemoteSessions(peers)
    expect(result.map((s) => s.id)).toEqual(['sess-1'])
  })

  it('flattens sessions from multiple reachable peers', () => {
    const peers: RemotePeerSessions[] = [
      { hostname: 'a', reachable: true, sessions: [makeSession({ id: 's1' }), makeSession({ id: 's2' })] },
      { hostname: 'b', reachable: true, sessions: [makeSession({ id: 's3' })] },
    ]
    const result = adaptAllRemoteSessions(peers)
    expect(result.length).toBe(3)
  })
})
