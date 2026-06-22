// Phase 146 FIX-03 (revised) — remoteAdapter.ts tests.
//
// Tests for adaptRemoteSession and adaptAllRemoteSessions focusing on the
// CARD-04 contract: hostname pass-through (globe icon) and URL pass-through
// (CR-01 "Open in browser" affordance).
//
// Broadcast pass-through tests REMOVED — D-10: discovery carries no join codes
// in the out-of-band design. See CONTEXT.md D-10.

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

function makeSession(overrides?: Partial<RemoteSession>): RemoteSession {
  return {
    id: 'sess-x',
    name: 'Test Session',
    cliType: 'claude',
    status: 'running',
    url: 'https://peer-host.ts.net:9443/sessions/sess-x',
    ...overrides,
  }
}

// ─── Existing contract (CARD-04 / CR-01) ─────────────────────────────────────

describe('adaptRemoteSession — existing contract', () => {
  it('maps id, name, cli, status from the remote session', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.id).toBe('sess-x')
    expect(adapted.name).toBe('Test Session')
    expect(adapted.cli).toBe('claude')
    expect(adapted.status).toBe('running')
  })

  it('carries the peer hostname so Hub shows globe icon + hostname (CARD-04)', () => {
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
