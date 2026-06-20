import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// Source-inspection tests for HubInteractiveModal (MODAL-03, MODAL-05).
// Pure ?raw import — NO render(), NO @xterm/xterm.
import raw from './HubInteractiveModal.tsx?raw'

// WR-07: Mock TerminalPanel and track the remote prop it receives.
// This lets us assert that HubInteractiveModal forwards the remote discriminator
// (Plan 07 proxy seam) without running xterm in jsdom.
interface CapturedTerminalProps {
  remote?: boolean
  sessionId?: string
  relayPort?: number
}

let capturedTerminalProps: CapturedTerminalProps | null = null

vi.mock('../TerminalPanel', () => ({
  TerminalPanel: (props: { remote?: boolean; sessionId?: string; relayPort?: number }) => {
    capturedTerminalProps = { remote: props.remote, sessionId: props.sessionId, relayPort: props.relayPort }
    return React.createElement('div', { 'data-testid': 'mock-terminal-panel' })
  },
}))

import { HubInteractiveModal } from './HubInteractiveModal'

// ---- Helpers ----

function makeSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'sess-1',
    cli: 'claude',
    name: 'Test Session',
    state: 'running',
    status: 'running',
    createdAt: new Date().toISOString(),
    hostname: '',
    webEnabled: false,
    viewerCount: 0,
    homeDir: false,
    browseEnabled: false,
    workDir: '/home/user',
    ...overrides,
  }
}

// ---- Source-inspection tests (MODAL-03, MODAL-05) — kept for contract coverage ----

describe('HubInteractiveModal (MODAL-03: TerminalPanel mounting)', () => {
  it('MODAL-03: mounts TerminalPanel inside the modal body', () => {
    expect(raw).toContain('TerminalPanel')
  })

  it('MODAL-03: passes sessionId prop to TerminalPanel', () => {
    expect(raw).toContain('sessionId=')
  })

  it('MODAL-03: passes relayPort prop to TerminalPanel', () => {
    expect(raw).toContain('relayPort=')
  })

  it('MODAL-03: passes theme prop to TerminalPanel', () => {
    expect(raw).toContain('theme=')
  })

  it('MODAL-03: passes pluginConfig prop to TerminalPanel', () => {
    expect(raw).toContain('pluginConfig=')
  })
})

describe('HubInteractiveModal (MODAL-05: isActive timing guard)', () => {
  it('MODAL-05: passes isActive prop to TerminalPanel', () => {
    expect(raw).toContain('isActive=')
  })

  it('MODAL-05: isActive is bound to the open phase (prevents 0-column layout during grow animation — Pitfall 1)', () => {
    // isActive must be gated on phase === 'open', not unconditionally true
    expect(raw).toMatch(/isActive=\{[^}]*open/)
  })
})

// ---- WR-07 behavioral tests: remote prop selects the proxy seam ----

describe('HubInteractiveModal (WR-07: remote prop routes to WS proxy seam)', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    capturedTerminalProps = null
    vi.clearAllMocks()
  })

  it('WR-07a: remote=true passes remote=true to TerminalPanel (selects daemon-proxy WS path)', () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)

    const session = makeSession({ id: 'r1', hostname: 'remote-host' })

    act(() => {
      root.render(
        React.createElement(HubInteractiveModal, {
          session,
          isOpen: true,
          relayPort: 51234,
          fontSize: 14,
          theme: { background: '#000', foreground: '#fff' },
          remote: true,
        }),
      )
    })

    // TerminalPanel mock should have received remote=true
    expect(capturedTerminalProps).not.toBeNull()
    expect(capturedTerminalProps!.remote).toBe(true)
  })

  it('WR-07b: remote=false (or omitted) passes remote=false/undefined to TerminalPanel (local path)', () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)

    const session = makeSession({ id: 'loc1', hostname: '' })

    act(() => {
      root.render(
        React.createElement(HubInteractiveModal, {
          session,
          isOpen: true,
          relayPort: 51234,
          fontSize: 14,
          theme: { background: '#000', foreground: '#fff' },
          remote: false,
        }),
      )
    })

    expect(capturedTerminalProps).not.toBeNull()
    // remote=false or undefined both mean local path
    expect(capturedTerminalProps!.remote).toBeFalsy()
  })
})
