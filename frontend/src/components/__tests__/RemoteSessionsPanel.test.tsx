import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import raw from '../../components/RemoteSessionsPanel.tsx?raw'
import type { RemoteSessionsPanelProps, RemotePeerSessions } from '../../components/RemoteSessionsPanel'
import { RemoteSessionsPanel } from '../../components/RemoteSessionsPanel'

// mockPeers includes `reachable` (added to the type in plan 04); cast via `as` to allow
// the field on fixtures before the type definition is extended.
const mockPeers = [
  {
    hostname: 'macbook-pro',
    reachable: true,
    sessions: [
      { id: 'sess-1', name: 'claude 1', cliType: 'claude', status: 'running', url: 'https://macbook-pro.ts.net:7443/sessions/sess-1' },
      { id: 'sess-2', name: 'codex 1', cliType: 'codex', status: 'idle', url: 'https://macbook-pro.ts.net:7443/sessions/sess-2' },
    ],
  },
  {
    hostname: 'dev-server',
    reachable: true,
    sessions: [
      { id: 'sess-3', name: 'claude 2', cliType: 'claude', status: 'waiting', url: 'https://dev-server.ts.net:7443/sessions/sess-3' },
    ],
  },
] as RemotePeerSessions[]

function renderPanel(props: Partial<RemoteSessionsPanelProps> = {}) {
  const defaults: RemoteSessionsPanelProps = {
    peers: [],
    loading: false,
    onOpen: vi.fn(),
    onBrowseFiles: vi.fn(),
  }
  const merged = { ...defaults, ...props }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(RemoteSessionsPanel, merged))
  })
  return { container, root }
}

afterEach(() => {
  document.body.innerHTML = ''
})

// --- Source Inspection Tests ---

describe('RemoteSessionsPanel source inspection', () => {
  it('exports RemoteSessionsPanel function component', () => {
    expect(raw).toContain('export function RemoteSessionsPanel')
  })

  it('exports RemoteSessionsPanelProps interface', () => {
    expect(raw).toContain('export interface RemoteSessionsPanelProps')
  })

  it('exports RemotePeerSessions interface', () => {
    expect(raw).toContain('export interface RemotePeerSessions')
  })

  it('exports RemoteSession interface', () => {
    expect(raw).toContain('export interface RemoteSession')
  })

  it('uses BEM class remote-panel__session-row', () => {
    expect(raw).toContain('remote-panel__session-row')
  })

  it('uses BEM class remote-panel__spinner', () => {
    expect(raw).toContain('remote-panel__spinner')
  })

  it('uses BEM class remote-panel__peer-header', () => {
    expect(raw).toContain('remote-panel__peer-header')
  })

  it('uses BEM class remote-panel__btn--open', () => {
    expect(raw).toContain('remote-panel__btn--open')
  })

  it('contains Open Session button text', () => {
    expect(raw).toContain('Open Session')
  })

  it('calls onOpen with session url', () => {
    expect(raw).toContain('onOpen(s.url)')
  })

  it('contains shareable sessions constraint copy', () => {
    expect(raw).toContain('Shows shareable sessions')
  })

  it('uses BEM class remote-panel__peer-meta for constraint sub-label', () => {
    expect(raw).toContain('remote-panel__peer-meta')
  })
})

// --- DOM Rendering Tests ---

describe('RemoteSessionsPanel DOM', () => {
  it('renders loading state when loading and no peers', () => {
    const { container } = renderPanel({ loading: true, peers: [] })
    const loading = container.querySelector('.remote-panel__loading')
    expect(loading).toBeTruthy()
    expect(loading!.textContent).toContain('Probing peers...')
    const spinner = container.querySelector('.remote-panel__spinner')
    expect(spinner).toBeTruthy()
  })

  it('renders empty state when not loading and no peers', () => {
    const { container } = renderPanel({ loading: false, peers: [] })
    const empty = container.querySelector('.remote-panel__empty')
    expect(empty).toBeTruthy()
    const title = container.querySelector('.remote-panel__empty-title')
    expect(title).toBeTruthy()
    expect(title!.textContent).toBe('No remote peers found')
    const body = container.querySelector('.remote-panel__empty-body')
    expect(body).toBeTruthy()
    expect(body!.textContent).toBe('No tailnet peers are running AgentHub.')
  })

  it('renders peer group headers', () => {
    const { container } = renderPanel({ peers: mockPeers })
    const headers = container.querySelectorAll('.remote-panel__peer-header')
    expect(headers.length).toBe(2)
    expect(headers[0].textContent).toBe('macbook-pro')
    expect(headers[1].textContent).toBe('dev-server')
  })

  it('renders peer-meta sub-label below each peer header', () => {
    const { container } = renderPanel({ peers: mockPeers })
    const metas = container.querySelectorAll('.remote-panel__peer-meta')
    expect(metas.length).toBe(2)
    expect(metas[0].textContent).toBe('Shows shareable sessions')
    expect(metas[1].textContent).toBe('Shows shareable sessions')
  })

  it('renders session rows with name, cli, and status', () => {
    const { container } = renderPanel({ peers: mockPeers })
    const rows = container.querySelectorAll('.remote-panel__session-row')
    expect(rows.length).toBe(3)

    // First row: claude 1
    const name0 = rows[0].querySelector('.remote-panel__name')
    expect(name0!.textContent).toBe('claude 1')
    const cli0 = rows[0].querySelector('.remote-panel__cli')
    expect(cli0!.textContent).toBe('claude')
    const status0 = rows[0].querySelector('.remote-panel__status')
    expect(status0!.classList.contains('remote-panel__status--running')).toBe(true)
  })

  it('Open button calls onOpen with session url', () => {
    const onOpen = vi.fn()
    const { container } = renderPanel({ peers: mockPeers, onOpen })
    const buttons = container.querySelectorAll('.remote-panel__btn--open')
    expect(buttons.length).toBe(3)

    // Click the first Open button
    ;(buttons[0] as HTMLButtonElement).click()
    expect(onOpen).toHaveBeenCalledWith('https://macbook-pro.ts.net:7443/sessions/sess-1')
  })

  it('shows data (not loading) when loading=true but peers exist', () => {
    const { container } = renderPanel({ loading: true, peers: mockPeers })
    // Should show peer data, not the loading spinner
    const headers = container.querySelectorAll('.remote-panel__peer-header')
    expect(headers.length).toBe(2)
    const loadingEl = container.querySelector('.remote-panel__loading')
    expect(loadingEl).toBeNull()
  })
})

// --- Phase 122-03 Task 2: "Browse files" button per remote session ---

describe('RemoteSessionsPanel — Phase 122-03 Browse files', () => {
  it('source: declares the onBrowseFiles prop', () => {
    expect(raw).toContain('onBrowseFiles')
  })

  it('source: uses the BEM class remote-panel__btn--browse', () => {
    expect(raw).toContain('remote-panel__btn--browse')
  })

  it('renders a Browse files button for every remote session', () => {
    const { container } = renderPanel({ peers: mockPeers })
    const buttons = container.querySelectorAll('.remote-panel__btn--browse')
    expect(buttons.length).toBe(3)
  })

  it('Browse files button has visible "Browse Files" label', () => {
    const { container } = renderPanel({ peers: mockPeers })
    const button = container.querySelector('.remote-panel__btn--browse') as HTMLButtonElement
    expect(button.textContent).toContain('Browse Files')
  })

  it('Browse files button has aria-label including the session name', () => {
    const { container } = renderPanel({ peers: mockPeers })
    const buttons = Array.from(
      container.querySelectorAll('.remote-panel__btn--browse'),
    ) as HTMLButtonElement[]
    expect(buttons[0].getAttribute('aria-label')).toBe('Browse files on claude 1')
  })

  it('Browse files click calls onBrowseFiles(sessionId, sessionName)', () => {
    const onBrowseFiles = vi.fn()
    const { container } = renderPanel({ peers: mockPeers, onBrowseFiles })
    const buttons = Array.from(
      container.querySelectorAll('.remote-panel__btn--browse'),
    ) as HTMLButtonElement[]
    buttons[0].click()
    expect(onBrowseFiles).toHaveBeenCalledWith('sess-1', 'claude 1')

    buttons[2].click()
    expect(onBrowseFiles).toHaveBeenCalledWith('sess-3', 'claude 2')
  })
})

// --- RB-04: Per-peer honest states (Wave 0 RED — these tests FAIL until plan 04 implementation) ---

describe('RemoteSessionsPanel — RB-04 unreachable peer state', () => {
  it('renders "Unreachable" text when peer has reachable=false', () => {
    const unreachablePeer: RemotePeerSessions[] = [
      {
        hostname: 'offline-host',
        reachable: false,
        sessions: [],
      },
    ]
    const { container } = renderPanel({ peers: unreachablePeer })
    // The word "Unreachable" must appear in the DOM as text (colorblind-safe, text-first)
    expect(container.textContent).toContain('Unreachable')
  })

  it('renders the peer hostname alongside the "Unreachable" badge', () => {
    const unreachablePeer: RemotePeerSessions[] = [
      {
        hostname: 'offline-host',
        reachable: false,
        sessions: [],
      },
    ]
    const { container } = renderPanel({ peers: unreachablePeer })
    const header = container.querySelector('.remote-panel__peer-header')
    expect(header).toBeTruthy()
    expect(header!.textContent).toBe('offline-host')
    expect(container.textContent).toContain('Unreachable')
  })

  it('does not render session rows for an unreachable peer', () => {
    const unreachablePeer: RemotePeerSessions[] = [
      {
        hostname: 'offline-host',
        reachable: false,
        sessions: [],
      },
    ]
    const { container } = renderPanel({ peers: unreachablePeer })
    const rows = container.querySelectorAll('.remote-panel__session-row')
    expect(rows.length).toBe(0)
  })
})

describe('RemoteSessionsPanel — RB-04 reachable peer with zero sessions', () => {
  it('renders "No shareable sessions" title for a reachable peer with sessions=[]', () => {
    const emptyReachablePeer: RemotePeerSessions[] = [
      {
        hostname: 'empty-host',
        reachable: true,
        sessions: [],
      },
    ]
    const { container } = renderPanel({ peers: emptyReachablePeer })
    expect(container.textContent).toContain('No shareable sessions')
  })

  it('renders body text "This peer has no sessions with web-sharing enabled."', () => {
    const emptyReachablePeer: RemotePeerSessions[] = [
      {
        hostname: 'empty-host',
        reachable: true,
        sessions: [],
      },
    ]
    const { container } = renderPanel({ peers: emptyReachablePeer })
    expect(container.textContent).toContain('This peer has no sessions with web-sharing enabled.')
  })

  it('does not render session rows for a reachable peer with empty sessions', () => {
    const emptyReachablePeer: RemotePeerSessions[] = [
      {
        hostname: 'empty-host',
        reachable: true,
        sessions: [],
      },
    ]
    const { container } = renderPanel({ peers: emptyReachablePeer })
    const rows = container.querySelectorAll('.remote-panel__session-row')
    expect(rows.length).toBe(0)
  })

  it('does not render "Unreachable" badge for a reachable peer with empty sessions', () => {
    const emptyReachablePeer: RemotePeerSessions[] = [
      {
        hostname: 'empty-host',
        reachable: true,
        sessions: [],
      },
    ]
    const { container } = renderPanel({ peers: emptyReachablePeer })
    expect(container.textContent).not.toContain('Unreachable')
  })
})

describe('RemoteSessionsPanel — RB-04 no false "No remote peers found" when ≥1 peer probed', () => {
  it('does not render "No remote peers found" when an unreachable peer is present', () => {
    const probedPeers: RemotePeerSessions[] = [
      {
        hostname: 'offline-host',
        reachable: false,
        sessions: [],
      },
    ]
    const { container } = renderPanel({ peers: probedPeers })
    expect(container.textContent).not.toContain('No remote peers found')
  })

  it('does not render "No remote peers found" when a reachable-but-empty peer is present', () => {
    const probedPeers: RemotePeerSessions[] = [
      {
        hostname: 'empty-host',
        reachable: true,
        sessions: [],
      },
    ]
    const { container } = renderPanel({ peers: probedPeers })
    expect(container.textContent).not.toContain('No remote peers found')
  })

  it('does not render "No remote peers found" when mixed peers (reachable + unreachable) are present', () => {
    const mixedPeers: RemotePeerSessions[] = [
      {
        hostname: 'offline-host',
        reachable: false,
        sessions: [],
      },
      {
        hostname: 'empty-host',
        reachable: true,
        sessions: [],
      },
      {
        hostname: 'live-host',
        reachable: true,
        sessions: [
          { id: 'sess-x', name: 'claude x', cliType: 'claude', status: 'running', url: 'https://live-host.ts.net:7443/sessions/sess-x' },
        ],
      },
    ]
    const { container } = renderPanel({ peers: mixedPeers })
    expect(container.textContent).not.toContain('No remote peers found')
  })

  it('still renders "No remote peers found" only when peers array is truly empty', () => {
    const { container } = renderPanel({ peers: [] })
    expect(container.textContent).toContain('No remote peers found')
  })
})

// --- RB-04: Error state — honest failure vs. zero-peers ---

describe('RemoteSessionsPanel — RB-04 error state', () => {
  it('renders "Could not load sessions" heading when error=true and peers is empty', () => {
    const { container } = renderPanel({ error: true, peers: [] })
    const title = container.querySelector('.remote-panel__empty-title')
    expect(title).toBeTruthy()
    expect(title!.textContent).toBe('Could not load sessions')
  })

  it('renders error body copy when error=true and peers is empty', () => {
    const { container } = renderPanel({ error: true, peers: [] })
    const body = container.querySelector('.remote-panel__empty-body')
    expect(body).toBeTruthy()
    expect(body!.textContent).toBe(
      'An error occurred loading remote sessions. Check your tailnet connection.',
    )
  })

  it('does NOT render "No remote peers found" when error=true and peers is empty', () => {
    const { container } = renderPanel({ error: true, peers: [] })
    expect(container.textContent).not.toContain('No remote peers found')
  })

  it('renders "No remote peers found" (not error state) when error=false and peers is empty', () => {
    const { container } = renderPanel({ error: false, peers: [] })
    expect(container.textContent).toContain('No remote peers found')
    expect(container.textContent).not.toContain('Could not load sessions')
  })

  it('renders peers normally when error=true but peers are present (error clears on next poll success)', () => {
    // Defensive: if a stale error flag survives while peers are populated, show peers.
    // The component only activates the error branch when peers.length === 0.
    const { container } = renderPanel({ error: true, peers: mockPeers })
    const headers = container.querySelectorAll('.remote-panel__peer-header')
    expect(headers.length).toBe(2)
    expect(container.textContent).not.toContain('Could not load sessions')
  })
})
