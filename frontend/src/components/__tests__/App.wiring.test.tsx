import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

// Source inspection tests for App.tsx remote-sessions wiring (52-03-02).
// These verify the wiring contract without mounting the full component tree
// (which requires Wails runtime mocks).

describe('App.tsx remote-sessions wiring (52-03-02)', () => {
  it('defines REMOTE_SESSIONS_TAB constant', () => {
    expect(raw).toContain('REMOTE_SESSIONS_TAB')
  })

  it('REMOTE_SESSIONS_TAB uses id __remote_sessions__', () => {
    expect(raw).toContain("'__remote_sessions__'")
  })

  it('imports GetRemoteSessions from wailsjs binding', () => {
    expect(raw).toContain('GetRemoteSessions')
  })

  it('imports RemoteSessionsPanel component', () => {
    expect(raw).toContain("from './components/RemoteSessionsPanel'")
  })

  it('imports RemotePeerSessions type', () => {
    expect(raw).toContain('RemotePeerSessions')
  })

  it('imports BrowserOpenURL from wails runtime', () => {
    expect(raw).toContain('BrowserOpenURL')
  })

  it('contains 30_000 polling interval for remote sessions', () => {
    expect(raw).toContain('30_000')
  })

  it('passes onOpenRemoteSessions to TabBar', () => {
    expect(raw).toContain('onOpenRemoteSessions={handleOpenRemoteSessions}')
  })

  it('renders RemoteSessionsPanel when remote-sessions tab active', () => {
    expect(raw).toContain('<RemoteSessionsPanel')
  })

  it('passes remotePeers to RemoteSessionsPanel', () => {
    expect(raw).toContain('peers={remotePeers}')
  })

  it('passes remoteLoading to RemoteSessionsPanel', () => {
    expect(raw).toContain('loading={remoteLoading}')
  })

  it('wires BrowserOpenURL into handleOpenRemoteSession', () => {
    expect(raw).toContain('BrowserOpenURL(url)')
  })

  it('filters remote-sessions type from terminal tab list', () => {
    expect(raw).toContain("tab.type === 'remote-sessions'")
  })

  it('uses null-guard peers ?? [] to handle Go nil slice', () => {
    expect(raw).toContain('peers ?? []')
  })
})
