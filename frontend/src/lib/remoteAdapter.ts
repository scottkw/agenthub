/* GRID-07: remote sessions adapted via adaptRemoteSession(); hostname != '' routes to GlobeAltIcon + hostname */
import type { RemotePeerSessions, RemoteSession } from './remoteSession'
import type { SessionInfo } from '../wailsjs/go/main/App'

// A remote session adapted to the SessionInfo shape carries extra fields the
// Wails-generated SessionInfo type has no slot for: the peer-supplied `url` (needed
// by the Hub card's "Open in browser" affordance, CR-01).
// broadcast join-code fields REMOVED — D-10: discovery carries no codes (out-of-band design).
export type AdaptedRemoteSessionInfo = SessionInfo & {
  url: string
}

export function adaptRemoteSession(
  peer: RemotePeerSessions,
  session: RemoteSession,
): AdaptedRemoteSessionInfo {
  return {
    id: session.id,
    name: session.name,
    cli: session.cliType,
    state: 'running',          // conservative default — remote status is not granular
    status: session.status || 'running',
    createdAt: new Date().toISOString(),
    hostname: peer.hostname,   // non-empty → GlobeAltIcon + hostname in SessionCard
    webEnabled: true,
    viewerCount: 0,
    workDir: '',               // remote sessions have no local workDir → fall into "Other"
    homeDir: false,
    browseEnabled: false,
    url: session.url,          // CR-01: carry the peer URL so "Open in browser" can resolve it
  }
}

export function adaptAllRemoteSessions(peers: RemotePeerSessions[]): AdaptedRemoteSessionInfo[] {
  return peers
    .filter((p) => p.reachable)
    .flatMap((p) => p.sessions.map((s) => adaptRemoteSession(p, s)))
}
