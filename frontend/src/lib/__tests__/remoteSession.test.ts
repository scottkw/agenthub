// Phase 122-03 Task 1 — remoteSession.ts pure-function helpers.
//
// These helpers translate between the RemoteSessionsPanel session model and
// the URL/base-URL shape consumed by App.tsx for the daemon-proxy file-browser
// path. Pure-function tests; no DOM, no React.

import { describe, it, expect } from 'vitest'
import {
  remoteBaseURLFor,
  findRemoteSession,
  isRemoteSessionId,
} from '../remoteSession'
import type { RemotePeerSessions } from '../../components/RemoteSessionsPanel'

const peers: RemotePeerSessions[] = [
  {
    hostname: 'hub-a',
    reachable: true,
    sessions: [
      {
        id: 'sess-A1',
        name: 'claude 1',
        cliType: 'claude',
        status: 'running',
        url: 'https://hub-a.tailnet.ts.net:9443/sessions/sess-A1',
      },
      {
        id: 'sess-A2',
        name: 'codex',
        cliType: 'codex',
        status: 'idle',
        url: 'https://hub-a.tailnet.ts.net:9443/sessions/sess-A2',
      },
    ],
  },
  {
    hostname: 'hub-b',
    reachable: true,
    sessions: [
      {
        id: 'sess-B1',
        name: 'aider',
        cliType: 'aider',
        status: 'running',
        url: 'https://hub-b.tailnet.ts.net:9443/sessions/sess-B1',
      },
    ],
  },
]

describe('remoteBaseURLFor', () => {
  it('drops the /sessions/{id} suffix via URL.origin', () => {
    expect(
      remoteBaseURLFor({
        url: 'https://hub-a.tailnet.ts.net:9443/sessions/abc',
      }),
    ).toBe('https://hub-a.tailnet.ts.net:9443')
  })

  it('also drops query strings and fragments', () => {
    expect(
      remoteBaseURLFor({
        url: 'https://hub-a.tailnet.ts.net:9443/sessions/abc?x=1#frag',
      }),
    ).toBe('https://hub-a.tailnet.ts.net:9443')
  })

  it('preserves the explicit port', () => {
    expect(remoteBaseURLFor({ url: 'https://h.example:7443/sessions/x' })).toBe(
      'https://h.example:7443',
    )
  })
})

describe('findRemoteSession', () => {
  it('finds a session in the first peer with hostname enriched', () => {
    const got = findRemoteSession('sess-A1', peers)
    expect(got).toBeDefined()
    expect(got!.id).toBe('sess-A1')
    expect(got!.hostname).toBe('hub-a')
    expect(got!.name).toBe('claude 1')
  })

  it('finds a session in a later peer', () => {
    const got = findRemoteSession('sess-B1', peers)
    expect(got).toBeDefined()
    expect(got!.hostname).toBe('hub-b')
  })

  it('returns undefined when no peer has the id', () => {
    expect(findRemoteSession('nope', peers)).toBeUndefined()
  })

  it('returns undefined on empty peer list', () => {
    expect(findRemoteSession('sess-A1', [])).toBeUndefined()
  })
})

describe('isRemoteSessionId', () => {
  it('is true when a matching session exists', () => {
    expect(isRemoteSessionId('sess-A2', peers)).toBe(true)
  })

  it('is false when no peer has the id', () => {
    expect(isRemoteSessionId('local-only', peers)).toBe(false)
  })

  it('is false on empty peer list', () => {
    expect(isRemoteSessionId('sess-A1', [])).toBe(false)
  })
})
