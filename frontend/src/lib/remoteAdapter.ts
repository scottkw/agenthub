/* GRID-07: remote sessions adapted via adaptRemoteSession(); hostname != '' routes to GlobeAltIcon + hostname */
import type { RemotePeerSessions, RemoteSession } from './remoteSession'
import type { SessionInfo } from '../wailsjs/go/main/App'

export function adaptRemoteSession(
  peer: RemotePeerSessions,
  session: RemoteSession,
): SessionInfo {
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
  }
}

export function adaptAllRemoteSessions(peers: RemotePeerSessions[]): SessionInfo[] {
  return peers
    .filter((p) => p.reachable)
    .flatMap((p) => p.sessions.map((s) => adaptRemoteSession(p, s)))
}
