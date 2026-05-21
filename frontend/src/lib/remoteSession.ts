// Phase 122-03 Task 1 — remoteSession.ts
//
// Pure-function helpers translating the RemoteSessionsPanel session model into
// the URL/base-URL shape consumed by App.tsx for the daemon-proxy file-browser
// path. All functions are stateless and side-effect-free; tests live in
// frontend/src/lib/__tests__/remoteSession.test.ts.

import type {
  RemotePeerSessions,
  RemoteSession,
} from '../components/RemoteSessionsPanel'

export interface RemoteSessionWithHost extends RemoteSession {
  hostname: string
}

/**
 * Derive the remote base URL from a session URL of the shape
 * `https://{fqdn}:{port}/sessions/{id}`. Uses URL.origin so trailing paths,
 * query strings, and fragments are dropped. Phase 122 / RESEARCH §Pattern 3.
 */
export function remoteBaseURLFor(session: { url: string }): string {
  return new URL(session.url).origin
}

/**
 * Find a remote session by id across all peers. Returns the session enriched
 * with its peer hostname, or undefined if no peer has the id.
 */
export function findRemoteSession(
  sessionId: string,
  remotePeers: RemotePeerSessions[],
): RemoteSessionWithHost | undefined {
  for (const peer of remotePeers) {
    const match = peer.sessions.find((s) => s.id === sessionId)
    if (match) return { ...match, hostname: peer.hostname }
  }
  return undefined
}

/** True iff the session id is in any peer's remote sessions list. */
export function isRemoteSessionId(
  sessionId: string,
  remotePeers: RemotePeerSessions[],
): boolean {
  return findRemoteSession(sessionId, remotePeers) !== undefined
}
