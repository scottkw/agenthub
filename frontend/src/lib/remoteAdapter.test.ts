import { describe, it, expect } from 'vitest'
import { adaptRemoteSession, adaptAllRemoteSessions } from './remoteAdapter'
import type { RemotePeerSessions, RemoteSession } from './remoteSession'

// Fixture builders
function makePeer(overrides?: Partial<RemotePeerSessions>): RemotePeerSessions {
  return {
    hostname: 'dev-box.local',
    reachable: true,
    sessions: [],
    ...overrides,
  }
}

function makeSession(overrides?: Partial<RemoteSession>): RemoteSession {
  return {
    id: 'sess-001',
    name: 'my-claude',
    cliType: 'claude',
    status: 'running',
    url: 'https://dev-box.local/session/sess-001',
    ...overrides,
  }
}

describe('adaptRemoteSession', () => {
  it('maps id → id, name → name, cliType → cli, status → status', () => {
    const peer = makePeer()
    const session = makeSession()
    const adapted = adaptRemoteSession(peer, session)
    expect(adapted.id).toBe('sess-001')
    expect(adapted.name).toBe('my-claude')
    expect(adapted.cli).toBe('claude')
    expect(adapted.status).toBe('running')
  })

  it('sets state to "running" (conservative default)', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.state).toBe('running')
  })

  it('sets hostname to peer.hostname', () => {
    const peer = makePeer({ hostname: 'remote-host.example.com' })
    const adapted = adaptRemoteSession(peer, makeSession())
    expect(adapted.hostname).toBe('remote-host.example.com')
  })

  it('sets workDir to empty string (falls into "Other" group)', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.workDir).toBe('')
  })

  it('sets webEnabled to true', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.webEnabled).toBe(true)
  })

  it('sets viewerCount to 0', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.viewerCount).toBe(0)
  })

  it('sets homeDir to false', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.homeDir).toBe(false)
  })

  it('sets browseEnabled to false', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.browseEnabled).toBe(false)
  })

  it('carries session.url through so "Open in browser" can resolve it (CR-01)', () => {
    const session = makeSession({ url: 'https://dev-box.local/session/sess-001' })
    const adapted = adaptRemoteSession(makePeer(), session)
    expect((adapted as { url?: string }).url).toBe('https://dev-box.local/session/sess-001')
  })

  it('defaults status to "running" when session.status is empty', () => {
    const session = makeSession({ status: '' })
    const adapted = adaptRemoteSession(makePeer(), session)
    expect(adapted.status).toBe('running')
  })

  it('produces a non-empty createdAt ISO string', () => {
    const adapted = adaptRemoteSession(makePeer(), makeSession())
    expect(adapted.createdAt).toBeTruthy()
    expect(() => new Date(adapted.createdAt)).not.toThrow()
  })
})

describe('adaptAllRemoteSessions', () => {
  it('flattens sessions across reachable peers', () => {
    const peers: RemotePeerSessions[] = [
      makePeer({ hostname: 'host-a', sessions: [makeSession({ id: 'a1' }), makeSession({ id: 'a2' })] }),
      makePeer({ hostname: 'host-b', sessions: [makeSession({ id: 'b1' })] }),
    ]
    const all = adaptAllRemoteSessions(peers)
    expect(all).toHaveLength(3)
    expect(all.map((s) => s.id)).toEqual(['a1', 'a2', 'b1'])
  })

  it('excludes peers where reachable === false (T-132-06)', () => {
    const peers: RemotePeerSessions[] = [
      makePeer({ hostname: 'reachable', reachable: true, sessions: [makeSession({ id: 'r1' })] }),
      makePeer({ hostname: 'unreachable', reachable: false, sessions: [makeSession({ id: 'u1' })] }),
    ]
    const all = adaptAllRemoteSessions(peers)
    expect(all).toHaveLength(1)
    expect(all[0].id).toBe('r1')
  })

  it('returns empty array when all peers are unreachable', () => {
    const peers: RemotePeerSessions[] = [
      makePeer({ reachable: false, sessions: [makeSession()] }),
    ]
    expect(adaptAllRemoteSessions(peers)).toEqual([])
  })

  it('returns empty array for empty peers list', () => {
    expect(adaptAllRemoteSessions([])).toEqual([])
  })
})
