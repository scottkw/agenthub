/**
 * @vitest-environment jsdom
 *
 * Phase 122-03 Task 3 — App.tsx remote-branch tab gate + cap caching.
 *
 * Source-inspection tests in the same style as App.fileBrowserMode.test.tsx —
 * mounting the real <App /> requires stubbing 30+ Wails imports and the xterm
 * runtime, so we inspect App.tsx?raw against the contract.
 *
 * Behaviour pinned (122-03-PLAN.md Task 3 Tests 1-7):
 *   - remoteCapsCached state initialised as empty Set
 *   - handleBrowseFilesRemote (or equivalent) callback
 *   - handleModalExchange (or equivalent) wires ExchangeJoinCodeAtURL → RegisterRemoteCap
 *   - Tab gate branches on isRemoteSessionId / findRemoteSession
 *   - Remote-path uses pathPrefix=/api/files/remote/{sid}, isRemote=true
 *   - No cap-token state ever crosses the remote-path render (T-122-03-01)
 *   - 'v3.5 follow-on' deferral comment removed
 */

import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App.tsx — Phase 122-03 module imports', () => {
  it('imports RemoteJoinCodeModal', () => {
    expect(raw).toMatch(/import\s*\{[^}]*\bRemoteJoinCodeModal\b/)
  })

  it('imports EnableWebSharingTakeover', () => {
    expect(raw).toMatch(/import\s*\{[^}]*\bEnableWebSharingTakeover\b/)
  })

  it('imports findRemoteSession + remoteBaseURLFor from lib/remoteSession', () => {
    expect(raw).toMatch(
      /import\s*\{[^}]*\bfindRemoteSession\b[^}]*\}\s*from\s*['"]\.\/lib\/remoteSession['"]/,
    )
    expect(raw).toContain('remoteBaseURLFor')
  })

  it('imports ExchangeJoinCodeAtURL + RegisterRemoteCap from the Wails bindings', () => {
    expect(raw).toMatch(/ExchangeJoinCodeAtURL/)
    expect(raw).toMatch(/RegisterRemoteCap/)
  })
})

describe('App.tsx — Phase 122-03 cap-cache state', () => {
  it('declares a remoteCapsCached state as a Set<string>', () => {
    // Allow Set<string> OR Record<string, true> — both are documented options.
    expect(raw).toMatch(/remoteCapsCached/)
    // And the initialiser must use useState (destructured pair form).
    expect(raw).toMatch(/setRemoteCapsCached[\]\s]*=\s*useState/)
  })

  it('declares joinModalForSession state for the modal trigger', () => {
    expect(raw).toMatch(/joinModalForSession/)
    expect(raw).toMatch(/setJoinModalForSession/)
  })
})

describe('App.tsx — Phase 122-03 remote-vs-local branching', () => {
  it('the file-browser tab gate calls findRemoteSession', () => {
    // Locate the tab gate block (begins at __files__ branch)
    const idx = raw.indexOf("activeId.startsWith('__files__')")
    expect(idx, '__files__ tab gate must exist').toBeGreaterThan(0)
    const slice = raw.slice(idx, idx + 4000)
    expect(slice).toContain('findRemoteSession')
  })

  it('the remote branch builds baseURL = http://127.0.0.1:${relayPort}', () => {
    const idx = raw.indexOf("activeId.startsWith('__files__')")
    const slice = raw.slice(idx, idx + 4000)
    expect(slice).toMatch(/http:\/\/127\.0\.0\.1:\$\{relayPort/)
  })

  it('the remote branch passes pathPrefix=/api/files/remote/${sid}', () => {
    expect(raw).toMatch(/pathPrefix=\{[^}]*\/api\/files\/remote\/\$\{[^}]+\}/)
  })

  it('the remote branch passes isRemote={true}', () => {
    const idx = raw.indexOf("activeId.startsWith('__files__')")
    const slice = raw.slice(idx, idx + 4000)
    expect(slice).toMatch(/isRemote=\{true\}/)
  })

  it('the v3.5 follow-on deferral comment is removed', () => {
    expect(raw).not.toContain('v3.5 follow-on')
  })
})

describe('App.tsx — Phase 122-03 cap-token confinement (T-122-03-01)', () => {
  it('the remote branch does NOT pass a capToken prop to FileBrowserTab', () => {
    // Locate the remote-FileBrowserTab render and verify capToken is absent
    const idx = raw.indexOf("activeId.startsWith('__files__')")
    const slice = raw.slice(idx, idx + 4000)
    // The slice contains both the remote branch and the local branch fallback;
    // the local branch DOES pass capToken={fbCapToken}, so we must scope to
    // just the remote branch. The remote branch is bounded by 'if (remote)'
    // through its 'return ' for FileBrowserTab.
    const remoteIdx = slice.indexOf('if (remote)')
    expect(remoteIdx).toBeGreaterThan(-1)
    const remoteSlice = slice.slice(remoteIdx, remoteIdx + 2000)
    // The remote FileBrowserTab block must contain pathPrefix but NOT capToken.
    expect(remoteSlice).toMatch(/pathPrefix=/)
    expect(remoteSlice).not.toMatch(/capToken=/)
  })
})

describe('App.tsx — Phase 122-03 modal handler wiring', () => {
  it('declares a handler that calls ExchangeJoinCodeAtURL followed by RegisterRemoteCap', () => {
    // The handler must invoke both Wails RPCs sequentially.
    expect(raw).toMatch(/ExchangeJoinCodeAtURL\(/)
    expect(raw).toMatch(/RegisterRemoteCap\(/)
    // And the order is exchange → register → mark cached
    const exchangeIdx = raw.indexOf('ExchangeJoinCodeAtURL(')
    const registerIdx = raw.indexOf('RegisterRemoteCap(')
    expect(exchangeIdx).toBeLessThan(registerIdx)
  })

  it('renders RemoteJoinCodeModal conditionally when joinModalForSession is set', () => {
    expect(raw).toMatch(/\{\s*joinModalForSession\s*&&\s*\(\s*\n?\s*<RemoteJoinCodeModal/)
  })

  // Phase 138 / NAV-04: RemoteSessionsPanel deleted — handleBrowseFilesRemote is now
  // wired directly to HubPanel's onBrowseFiles prop (App.hub.test.tsx covers this).
})

describe('App.tsx — Phase 122-03 EnableWebSharingTakeover fallback', () => {
  it('renders EnableWebSharingTakeover when remote session has no cap cached (defensive guard)', () => {
    expect(raw).toContain('<EnableWebSharingTakeover')
  })
})
