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
  isActive?: boolean
}

let capturedTerminalProps: CapturedTerminalProps | null = null

vi.mock('../TerminalPanel', () => ({
  TerminalPanel: (props: CapturedTerminalProps) => {
    capturedTerminalProps = props
    return React.createElement('div', { 'data-testid': 'mock-terminal-panel' })
  },
}))

// D-02 overlay: Mock ChatPanel to track open prop without opening a WebSocket.
interface CapturedChatPanelProps {
  open?: boolean
  sessionId?: string
  relayPort?: number
  onUnreadChange?: (count: number, hasMention: boolean) => void
}

let capturedChatPanelProps: CapturedChatPanelProps | null = null

vi.mock('./ChatPanel', () => ({
  ChatPanel: (props: CapturedChatPanelProps) => {
    capturedChatPanelProps = props
    return React.createElement('div', {
      'data-testid': 'mock-chat-panel',
      'data-open': String(props.open),
    })
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
    capturedChatPanelProps = null
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

// ── D-02 overlay layout: drawer + toggle + badge ──────────────────────────

/** Shared render helper for overlay tests */
function renderModal(isOpen = true) {
  const session = makeSession()
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(
      React.createElement(HubInteractiveModal, {
        session,
        isOpen,
        relayPort: 51234,
        fontSize: 14,
        theme: { background: '#000', foreground: '#fff' },
      }),
    )
  })
  return { container, root, session }
}

describe('HubInteractiveModal — D-02 overlay drawer + chat toggle', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    capturedTerminalProps = null
    capturedChatPanelProps = null
    vi.clearAllMocks()
  })

  it('renders the chat toggle button with aria-label "Open chat" initially', () => {
    const { container } = renderModal()
    const toggleBtn = container.querySelector('.hub-modal__chat-toggle')
    expect(toggleBtn).not.toBeNull()
    expect(toggleBtn?.getAttribute('aria-label')).toBe('Open chat')
  })

  it('ChatPanel is mounted with open=false when the modal first opens (D-09: unread accrues while closed)', () => {
    renderModal()
    // ChatPanel must be present regardless of chatOpen (always-mounted for unread accrual)
    expect(capturedChatPanelProps).not.toBeNull()
    expect(capturedChatPanelProps!.open).toBe(false)
  })

  it('clicking the toggle sets ChatPanel open=true (overlay slides in)', () => {
    const { container } = renderModal()

    const toggleBtn = container.querySelector('.hub-modal__chat-toggle') as HTMLElement | null
    act(() => { toggleBtn?.click() })

    expect(capturedChatPanelProps!.open).toBe(true)
    // Toggle aria-label flips to Close chat
    expect(toggleBtn?.getAttribute('aria-label')).toBe('Close chat')
  })

  it('TerminalPanel isActive equals the modal-open prop regardless of chatOpen (no PTY resize on toggle)', () => {
    const { container } = renderModal(true)

    // Initially: isActive should equal the isOpen prop (true)
    expect(capturedTerminalProps!.isActive).toBe(true)

    // Toggle chat open
    const toggleBtn = container.querySelector('.hub-modal__chat-toggle') as HTMLElement | null
    act(() => { toggleBtn?.click() })

    // After toggling chat: TerminalPanel isActive must STILL be true (modal is still open)
    // The overlay does NOT change isActive — no PTY resize is triggered (D-02)
    expect(capturedTerminalProps!.isActive).toBe(true)
  })

  it('terminal is NOT wrapped in a shrinking column — source has no hub-modal__terminal-col', () => {
    // Overlay mode: the terminal stays full-bleed; no hub-modal__terminal-col wrapper
    expect(raw).not.toContain('hub-modal__terminal-col')
  })

  it('onUnreadChange from ChatPanel updates the toggle ChatBadge (NOTIF-01)', () => {
    const { container } = renderModal()

    // Initially no badge (count=0)
    expect(container.querySelector('.chat-badge')).toBeNull()

    // Simulate ChatPanel reporting 3 unread messages
    act(() => {
      capturedChatPanelProps?.onUnreadChange?.(3, false)
    })

    // Badge should now appear on the toggle button
    const badge = container.querySelector('.chat-badge')
    expect(badge).not.toBeNull()
    expect(badge?.textContent).toBe('3')
  })

  it('onUnreadChange with hasMention=true shows @ glyph badge (D-10)', () => {
    const { container } = renderModal()

    act(() => {
      capturedChatPanelProps?.onUnreadChange?.(1, true)
    })

    const badge = container.querySelector('.chat-badge--mention')
    expect(badge).not.toBeNull()
    expect(badge?.textContent).toBe('@')
  })
})
