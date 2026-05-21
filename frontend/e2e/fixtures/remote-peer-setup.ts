// Phase 122-05 — remote-peer fixture helper.
//
// The playwright fixture binary (cmd/playwright-fixture/main.go) now spins up
// a SECOND TLS listener (alongside the main webserver on BASE_URL) that
// mimics a remote AgentHub peer on the tailnet. The listener's address is
// published via the REMOTE_PEER_URL line in the fixture's stdout (parsed by
// global-setup.ts and persisted in fixture-env.json).
//
// This helper exposes:
//   - remotePeerURL()           — the URL of the mock remote peer
//   - fixtureRemoteSessionId    — the session ID the mock advertises ("peer-sid")
//   - fixtureRemoteCap          — the cap token the mock validates ("FIXTURE_CAP")
//   - remoteFilesListURL(...)   — URL builder for the canned /api/files/list response
//   - remoteFilesStatURL(...)   — URL builder for /api/files/stat
//   - remoteFilesReadURL(...)   — URL builder for /api/files/read
//
// Scenarios 16 + 17 in files-browser.spec.ts use these to verify the
// remote-session upstream contract (the byte-shape both the desktop GUI's
// daemon-proxy path and the TUI's RemoteFilesClient depend on).

import { loadFixtureEnv } from '../fixture-env'

export const fixtureRemoteSessionId = 'peer-sid'
export const fixtureRemoteCap = 'FIXTURE_CAP'

/**
 * The URL of the mock remote peer (second TLS listener inside the fixture
 * binary). Throws if global-setup did not parse the REMOTE_PEER_URL line —
 * this surfaces an out-of-sync fixture build promptly rather than producing
 * silent ECONNREFUSED in every scenario.
 */
export function remotePeerURL(): string {
  const env = loadFixtureEnv()
  if (!env.remotePeerURL) {
    throw new Error(
      'remotePeerURL: fixture env has no remotePeerURL — rebuild the fixture (Phase 122-05 added it)',
    )
  }
  return env.remotePeerURL
}

/**
 * remoteFilesListURL builds the canonical /api/files/list URL against the
 * mock remote peer with the canned cap + session. Passing cap='' (or any
 * non-FIXTURE_CAP value) exercises the 401 path.
 */
export function remoteFilesListURL(opts?: { path?: string; cap?: string }): string {
  const path = opts?.path ?? '.'
  const cap = opts?.cap ?? fixtureRemoteCap
  const params = new URLSearchParams({
    session: fixtureRemoteSessionId,
    path,
  })
  if (cap !== '') params.set('cap', cap)
  return `${remotePeerURL()}/api/files/list?${params.toString()}`
}

export function remoteFilesStatURL(opts?: { path?: string; cap?: string }): string {
  const path = opts?.path ?? 'a.txt'
  const cap = opts?.cap ?? fixtureRemoteCap
  const params = new URLSearchParams({
    session: fixtureRemoteSessionId,
    path,
  })
  if (cap !== '') params.set('cap', cap)
  return `${remotePeerURL()}/api/files/stat?${params.toString()}`
}

export function remoteFilesReadURL(opts?: { path?: string; cap?: string }): string {
  const path = opts?.path ?? 'a.txt'
  const cap = opts?.cap ?? fixtureRemoteCap
  const params = new URLSearchParams({
    session: fixtureRemoteSessionId,
    path,
  })
  if (cap !== '') params.set('cap', cap)
  return `${remotePeerURL()}/api/files/read?${params.toString()}`
}

/**
 * The /join/exchange URL against the mock remote peer. Tests POST a body
 * with `code=ABCDE` (any code is accepted by the mock) and observe the
 * 303 Location header containing the cap.
 */
export function remoteJoinExchangeURL(): string {
  return `${remotePeerURL()}/join/exchange`
}
