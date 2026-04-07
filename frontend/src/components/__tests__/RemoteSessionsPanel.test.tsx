import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import raw from '../../components/RemoteSessionsPanel.tsx?raw'
import type { RemoteSessionsPanelProps, RemotePeerSessions } from '../../components/RemoteSessionsPanel'
import { RemoteSessionsPanel } from '../../components/RemoteSessionsPanel'

const mockPeers: RemotePeerSessions[] = [
  {
    hostname: 'macbook-pro',
    sessions: [
      { id: 'sess-1', name: 'claude 1', cliType: 'claude', status: 'running', url: 'https://macbook-pro.ts.net:7443/sessions/sess-1' },
      { id: 'sess-2', name: 'codex 1', cliType: 'codex', status: 'idle', url: 'https://macbook-pro.ts.net:7443/sessions/sess-2' },
    ],
  },
  {
    hostname: 'dev-server',
    sessions: [
      { id: 'sess-3', name: 'claude 2', cliType: 'claude', status: 'waiting', url: 'https://dev-server.ts.net:7443/sessions/sess-3' },
    ],
  },
]

function renderPanel(props: Partial<RemoteSessionsPanelProps> = {}) {
  const defaults: RemoteSessionsPanelProps = {
    peers: [],
    loading: false,
    onOpen: vi.fn(),
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

  it('contains web-enabled sessions constraint copy', () => {
    expect(raw).toContain('Shows web-enabled sessions only')
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
    expect(metas[0].textContent).toBe('Shows web-enabled sessions only')
    expect(metas[1].textContent).toBe('Shows web-enabled sessions only')
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
