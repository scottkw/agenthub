import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import raw from '../../components/DaemonManagerPanel.tsx?raw'
import type { DaemonManagerPanelProps } from '../../components/DaemonManagerPanel'
import { DaemonManagerPanel } from '../../components/DaemonManagerPanel'

// SessionInfo type matching wailsjs/go/main/App.d.ts
interface SessionInfo {
  id: string
  cli: string
  name: string
  state: string
  createdAt: string
}

const mockSessions: SessionInfo[] = [
  { id: 'sess-1', cli: 'claude', name: 'claude 1', state: 'running', createdAt: '2026-04-01T10:00:00Z' },
  { id: 'sess-2', cli: 'codex', name: 'codex 1', state: 'idle', createdAt: '2026-04-01T11:00:00Z' },
]

function renderPanel(props: Partial<DaemonManagerPanelProps> = {}) {
  const defaults: DaemonManagerPanelProps = {
    sessions: [],
    sessionStatuses: {},
    webServerRunning: false,
    webEnabled: {},
    onKill: vi.fn(),
    onToggleWeb: vi.fn(),
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
})
