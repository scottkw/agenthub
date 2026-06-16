import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import { flushSync } from 'react-dom'
import raw from '../../components/DaemonManagerPanel.tsx?raw'
import type { DaemonManagerPanelProps } from '../../components/DaemonManagerPanel'
import { DaemonManagerPanel } from '../../components/DaemonManagerPanel'

// Mock Wails runtime modules used by DaemonManagerPanel / SessionSharePanel
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  IssueCapabilities: vi.fn().mockResolvedValue({
    readUrl: 'https://example.com/r',
    writeUrl: 'https://example.com/w',
    readCode: 'rc',
    writeCode: 'wc',
    homeDir: false,
  }),
  SetSessionFilesWrite: vi.fn().mockResolvedValue(undefined),
  GetLocalNetworkPassword: vi.fn().mockResolvedValue(''),
  GetCapabilityQRCode: vi.fn().mockResolvedValue(''),
}))

// SessionInfo type matching wailsjs/go/main/App.d.ts (Phase 131 / GRID-02 workDir added)
interface SessionInfo {
  id: string
  cli: string
  name: string
  state: string
  status: string
  createdAt: string
  hostname: string
  webEnabled: boolean
  viewerCount: number
  exitCode?: number
  duration?: number
  homeDir: boolean
  filesWrite: boolean
  workDir: string  // Phase 131 / GRID-02: working directory path
}

const mockSessions: SessionInfo[] = [
  { id: 'sess-1', cli: 'claude', name: 'claude 1', state: 'running', status: 'running', createdAt: '2026-04-01T10:00:00Z', hostname: 'macbook-pro.local', webEnabled: false, viewerCount: 0, homeDir: false, filesWrite: false, workDir: '' },
  { id: 'sess-2', cli: 'codex', name: 'codex 1', state: 'idle', status: 'idle', createdAt: '2026-04-01T11:00:00Z', hostname: 'dev-server.internal', webEnabled: false, viewerCount: 0, homeDir: false, filesWrite: false, workDir: '' },
]

function renderPanel(props: Partial<DaemonManagerPanelProps> = {}) {
  const defaults: DaemonManagerPanelProps = {
    sessions: [],
    sessionStatuses: {},
    webServerRunning: false,
    webEnabled: {},
    onKill: vi.fn(),
    onToggleWeb: vi.fn(),
    onOpenFileBrowser: vi.fn(),
  }
  const merged = { ...defaults, ...props }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(DaemonManagerPanel, merged))
  })
  return { container, root }
}

describe('DaemonManagerPanel (DMGR-03) - source inspection', () => {
  it('exports DaemonManagerPanel function component', () => {
    expect(raw).toContain('export function DaemonManagerPanel')
  })

  it('uses BEM class daemon-panel__session-row', () => {
    expect(raw).toContain('daemon-panel__session-row')
  })

  it('accepts onKill prop', () => {
    expect(raw).toContain('onKill')
  })

  it('accepts onToggleWeb prop', () => {
    expect(raw).toContain('onToggleWeb')
  })

  it('uses status class pattern daemon-panel__status--${status}', () => {
    expect(raw).toContain('daemon-panel__status--')
    expect(raw).toContain('daemon-panel__status')
  })

  it('uses BEM class daemon-panel__hostname', () => {
    expect(raw).toContain('daemon-panel__hostname')
  })
})

describe('DaemonManagerPanel (DMGR-03) - DOM tests', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders empty state message when sessions array is empty', () => {
    ;({ container, root } = renderPanel({ sessions: [] }))
    expect(container.textContent).toContain('No active sessions')
  })

  it('renders session rows matching sessions array length', () => {
    ;({ container, root } = renderPanel({ sessions: mockSessions }))
    const rows = container.querySelectorAll('.daemon-panel__session-row')
    expect(rows.length).toBe(mockSessions.length)
  })

  it('Kill button calls onKill with correct session id', () => {
    const onKill = vi.fn()
    ;({ container, root } = renderPanel({ sessions: mockSessions, onKill }))
    const killButtons = Array.from(container.querySelectorAll('button')).filter(
      (b) => b.textContent === 'Kill',
    )
    expect(killButtons.length).toBeGreaterThan(0)
    killButtons[0].click()
    expect(onKill).toHaveBeenCalledWith('sess-1')
  })

  it('Web toggle button calls onToggleWeb with correct session id', () => {
    const onToggleWeb = vi.fn()
    ;({ container, root } = renderPanel({
      sessions: mockSessions,
      webServerRunning: true,
      onToggleWeb,
    }))
    const webButtons = Array.from(container.querySelectorAll('button')).filter(
      (b) => b.textContent === 'Web On' || b.textContent === 'Web Off',
    )
    expect(webButtons.length).toBeGreaterThan(0)
    webButtons[0].click()
    expect(onToggleWeb).toHaveBeenCalledWith('sess-1')
  })

  it('Web toggle button is disabled when webServerRunning is false', () => {
    ;({ container, root } = renderPanel({
      sessions: mockSessions,
      webServerRunning: false,
    }))
    const webButtons = Array.from(container.querySelectorAll('button')).filter(
      (b) => b.textContent === 'Web On' || b.textContent === 'Web Off',
    )
    expect(webButtons.length).toBeGreaterThan(0)
    webButtons.forEach((btn) => {
      expect((btn as HTMLButtonElement).disabled).toBe(true)
    })
  })

  it('renders hostname badge per session row', () => {
    ;({ container, root } = renderPanel({ sessions: mockSessions }))
    const hostnames = container.querySelectorAll('.daemon-panel__hostname')
    expect(hostnames.length).toBe(mockSessions.length)
    expect(hostnames[0].textContent).toBe('macbook-pro.local')
    expect(hostnames[1].textContent).toBe('dev-server.internal')
  })

  it('renders em dash when hostname is empty', () => {
    const noHostSessions = [
      { id: 'sess-3', cli: 'claude', name: 'test', state: 'running', status: 'running', createdAt: '2026-04-01T12:00:00Z', hostname: '', webEnabled: false, viewerCount: 0, homeDir: false, filesWrite: false, workDir: '' },
    ]
    ;({ container, root } = renderPanel({ sessions: noHostSessions }))
    const badge = container.querySelector('.daemon-panel__hostname')
    expect(badge).not.toBeNull()
    expect(badge!.textContent).toBe('\u2014')
  })
})

describe('DaemonManagerPanel \u2014 write-toggle re-hydration from server truth', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
    vi.clearAllMocks()
  })

  it('renders "Enable file writes" toggle checked when session arrives with filesWrite=true', async () => {
    // Phase 124 / CAP-04: when a session has filesWrite=true (server-authoritative),
    // the toggle must render as aria-checked="true" WITHOUT any user click \u2014 pure
    // re-hydration from the sessions prop. We wrap in act() so the seeding useEffect
    // flushes before the assertion.
    const writeSessions: SessionInfo[] = [
      { id: 'sess-fw', cli: 'claude', name: 'write-on session', state: 'running', status: 'running',
        createdAt: '2026-04-01T10:00:00Z', hostname: 'host', webEnabled: false, viewerCount: 0,
        homeDir: false, filesWrite: true, workDir: '' },
    ]
    await act(async () => {
      ;({ container, root } = renderPanel({ sessions: writeSessions }))
    })

    const toggle = Array.from(container.querySelectorAll('[role="switch"]')).find(el =>
      el.getAttribute('aria-label') === 'Enable file writes'
    )
    expect(toggle).not.toBeNull()
    expect(toggle!.getAttribute('aria-checked')).toBe('true')
  })

  it('renders "Enable file writes" toggle unchecked when session has filesWrite=false', async () => {
    const noWriteSessions: SessionInfo[] = [
      { id: 'sess-nw', cli: 'claude', name: 'write-off session', state: 'running', status: 'running',
        createdAt: '2026-04-01T10:00:00Z', hostname: 'host', webEnabled: false, viewerCount: 0,
        homeDir: false, filesWrite: false, workDir: '' },
    ]
    await act(async () => {
      ;({ container, root } = renderPanel({ sessions: noWriteSessions }))
    })

    const toggle = Array.from(container.querySelectorAll('[role="switch"]')).find(el =>
      el.getAttribute('aria-label') === 'Enable file writes'
    )
    expect(toggle).not.toBeNull()
    expect(toggle!.getAttribute('aria-checked')).toBe('false')
  })
})

describe('DaemonManagerPanel WR-04 \u2014 homeDir banner sourced from SessionInfo.homeDir', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
    vi.clearAllMocks()
  })

  it('shows home-dir banner when s.homeDir=true and writes are enabled via sessionWrites state', async () => {
    // A session whose server-reported homeDir=true and whose write toggle we enable.
    // We must mock IssueCapabilities and SetSessionFilesWrite for the toggle interaction.
    const { IssueCapabilities, SetSessionFilesWrite } = await import('../../wailsjs/go/main/App')
    const mockIssue = vi.mocked(IssueCapabilities)
    const mockSetWrite = vi.mocked(SetSessionFilesWrite)
    mockSetWrite.mockResolvedValue(undefined)
    mockIssue.mockResolvedValue({
      readUrl: 'https://example.com/r',
      writeUrl: 'https://example.com/w',
      readCode: 'rc',
      writeCode: 'wc',
      homeDir: false, // IssueCapabilities response homeDir is irrelevant post-WR-04
    })

    const homeDirSession: SessionInfo[] = [
      { id: 'sess-home', cli: 'claude', name: 'home session', state: 'running', status: 'running',
        createdAt: '2026-04-01T10:00:00Z', hostname: 'host', webEnabled: true, viewerCount: 0,
        homeDir: true, filesWrite: false, workDir: '' },
    ]

    ;({ container, root } = renderPanel({
      sessions: homeDirSession,
      webServerRunning: true,
      webEnabled: { 'sess-home': true },
    }))

    // The "Enable file writes" toggle is rendered; click it to enable writes
    const toggle = Array.from(container.querySelectorAll('[role="switch"]')).find(el =>
      el.getAttribute('aria-label') === 'Enable file writes'
    ) as HTMLInputElement | null
    expect(toggle).not.toBeNull()

    await flushSync(async () => { toggle!.click() })
    // Give async handlers time to run
    await new Promise(r => setTimeout(r, 50))

    // After enabling writes, the home-dir banner should appear because
    // s.homeDir=true (from SessionInfo, server source of truth)
    // The banner contains "Warning:" per HomeDirWriteWarning colorblind contract
    expect(container.textContent).toContain('Warning:')
    expect(container.textContent).toContain('home')
  })

  it('does NOT show home-dir banner when s.homeDir=false even if writes are enabled', () => {
    // WR-04: banner must not appear for sessions where homeDir=false
    const normalSession: SessionInfo[] = [
      { id: 'sess-norm', cli: 'claude', name: 'normal session', state: 'running', status: 'running',
        createdAt: '2026-04-01T10:00:00Z', hostname: 'host', webEnabled: false, viewerCount: 0,
        homeDir: false, filesWrite: true, workDir: '' },
    ]
    ;({ container, root } = renderPanel({ sessions: normalSession }))
    // The component uses s.homeDir (SessionInfo) not share?.homeDir post-WR-04
    // With homeDir=false on the session, no banner should render
    expect(container.textContent).not.toContain('Warning: writes can affect your home directory')
  })
})
