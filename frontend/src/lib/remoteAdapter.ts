/* GRID-07: remote sessions adapted via adaptRemoteSession(); hostname != '' routes to GlobeAltIcon + hostname */
import type { RemotePeerSessions, RemoteSession } from './remoteSession'
import type { SessionInfo } from '../wailsjs/go/main/App'

// A remote session adapted to the SessionInfo shape carries extra fields the
// Wails-generated SessionInfo type has no slot for: the peer-supplied `url` (needed
// by the Hub card's "Open in browser" affordance, CR-01) and the Phase 146 join codes
// (roJoinCode / rwJoinCode) for the exchange-then-open cap flow (FIX-03).
export type AdaptedRemoteSessionInfo = SessionInfo & {
  url: string
  /** Phase 146 FIX-03: read-only join code; absent when session is not shared (D-03). */
  roJoinCode?: string
  /** Phase 146 FIX-03: read-write join code; used only when viewer is the peer owner (D-06). */
  rwJoinCode?: string
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
    // Phase 146 FIX-03: pass through join codes so handleOpenRemoteSession can exchange them.
    // Optional — absent when the peer hasn't enabled sharing yet (D-03 not-shared path).
    roJoinCode: session.roJoinCode,
    rwJoinCode: session.rwJoinCode,
  }
}

export function adaptAllRemoteSessions(peers: RemotePeerSessions[]): AdaptedRemoteSessionInfo[] {
  return peers
    .filter((p) => p.reachable)
    .flatMap((p) => p.sessions.map((s) => adaptRemoteSession(p, s)))
}
