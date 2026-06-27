import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'

// CHAT-PARITY-01: Behavioral tests for TerminalChatHost.
// TerminalPanel and ChatPanel are mocked to avoid xterm.js / WebSocket in JSDOM.

interface CapturedTerminalProps {
  sessionId?: string
  isActive?: boolean
  relayPort?: number
  fontSize?: number
  onFontSizeChange?: (delta: number) => void
  theme?: object
  pluginConfig?: object | null
  onWebGLContextLost?: (reason: string) => void
  onRegisterSaver?: (sessionId: string, fn: (() => string) | null) => void
  onProgressChange?: (sessionId: string, state: object) => void
  remote?: boolean
  [key: string]: unknown
}

let capturedTerminalProps: CapturedTerminalProps | null = null

vi.mock('../TerminalPanel', () => ({
  TerminalPanel: (props: CapturedTerminalProps) => {
    capturedTerminalProps = props
    return React.createElement('div', { 'data-testid': 'mock-terminal-panel' })
  },
}))

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

vi.mock('./ChatBadge', () => ({
  ChatBadge: (props: { count: number; hasMention: boolean }) =>
    React.createElement('span', { 'data-testid': 'mock-chat-badge' }, props.count > 0 ? String(props.count) : null),
}))

vi.mock('@heroicons/react/24/outline', () => ({
  ChatBubbleLeftRightIcon: (props: { className?: string; 'aria-hidden'?: boolean | string }) =>
    React.createElement('svg', { 'data-testid': 'chat-bubble-icon', ...props }),
}))

// TerminalChatHost is imported AFTER mocks are set up
import { TerminalChatHost } from './TerminalChatHost'

// ---- Test helpers ----

const BASE_PROPS = {
  sessionId: 'sess-test',
  isActive: true,
  relayPort: 51234,
  fontSize: 14,
  onFontSizeChange: () => {},
  theme: { background: '#000', foreground: '#fff' },
  pluginConfig: null,
}

function renderHost(overrides: Partial<typeof BASE_PROPS> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(React.createElement(TerminalChatHost, { ...BASE_PROPS, ...overrides }))
  })
  return { container, root }
}

// ---- Tests ----

describe('TerminalChatHost — Test 1: initial render', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    capturedTerminalProps = null
    capturedChatPanelProps = null
    vi.clearAllMocks()
  })

  it('mounts exactly one .hub-modal__chat-toggle button', () => {
    const { container } = renderHost()
    const toggles = container.querySelectorAll('.hub-modal__chat-toggle')
    expect(toggles).toHaveLength(1)
  })

  it('mounts ChatPanel with open=false initially (D-09 always-mounted)', () => {
    renderHost()
    expect(capturedChatPanelProps).not.toBeNull()
    expect(capturedChatPanelProps!.open).toBe(false)
  })

  it('toggle button has aria-label "Open chat" initially', () => {
    const { container } = renderHost()
    const btn = container.querySelector('.hub-modal__chat-toggle')
    expect(btn?.getAttribute('aria-label')).toBe('Open chat')
  })
})

describe('TerminalChatHost — Test 2: toggle interaction', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    capturedTerminalProps = null
    capturedChatPanelProps = null
    vi.clearAllMocks()
  })

  it('clicking the toggle opens the chat (ChatPanel open flips to true)', () => {
    const { container } = renderHost()
    const btn = container.querySelector('.hub-modal__chat-toggle') as HTMLElement | null
    act(() => { btn?.click() })
    expect(capturedChatPanelProps!.open).toBe(true)
  })

  it('clicking the toggle a second time closes the chat (open flips back to false)', () => {
    const { container } = renderHost()
    const btn = container.querySelector('.hub-modal__chat-toggle') as HTMLElement | null
    act(() => { btn?.click() })
    expect(capturedChatPanelProps!.open).toBe(true)
    act(() => { btn?.click() })
    expect(capturedChatPanelProps!.open).toBe(false)
  })

  it('aria-label flips to "Close chat" after toggle click', () => {
    const { container } = renderHost()
    const btn = container.querySelector('.hub-modal__chat-toggle') as HTMLElement | null
    act(() => { btn?.click() })
    expect(btn?.getAttribute('aria-label')).toBe('Close chat')
  })
})

describe('TerminalChatHost — Test 3: TerminalPanel prop forwarding', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    capturedTerminalProps = null
    capturedChatPanelProps = null
    vi.clearAllMocks()
  })

  it('forwards onWebGLContextLost to the mocked TerminalPanel', () => {
    const onWebGLContextLost = vi.fn()
    renderHost({ onWebGLContextLost } as Parameters<typeof renderHost>[0])
    expect(capturedTerminalProps).not.toBeNull()
    expect(capturedTerminalProps!.onWebGLContextLost).toBe(onWebGLContextLost)
  })

  it('forwards onRegisterSaver to the mocked TerminalPanel', () => {
    const onRegisterSaver = vi.fn()
    renderHost({ onRegisterSaver } as Parameters<typeof renderHost>[0])
    expect(capturedTerminalProps!.onRegisterSaver).toBe(onRegisterSaver)
  })

  it('forwards onProgressChange to the mocked TerminalPanel', () => {
    const onProgressChange = vi.fn()
    renderHost({ onProgressChange } as Parameters<typeof renderHost>[0])
    expect(capturedTerminalProps!.onProgressChange).toBe(onProgressChange)
  })

  it('isActive is forwarded from host props (not bound to chatOpen — D-02 no PTY resize)', () => {
    renderHost({ isActive: true })
    expect(capturedTerminalProps!.isActive).toBe(true)

    // After toggle: isActive must still be true (from props, not chatOpen)
    // Re-render a fresh instance to verify isActive prop binding
    document.body.innerHTML = ''
    capturedTerminalProps = null
    capturedChatPanelProps = null
    const { container } = renderHost({ isActive: true })
    const btn = container.querySelector('.hub-modal__chat-toggle') as HTMLElement | null
    act(() => { btn?.click() })
    expect(capturedTerminalProps!.isActive).toBe(true)
  })
})

describe('TerminalChatHost — Test 4: DOM order (ChatPanel before toggle)', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    capturedTerminalProps = null
    capturedChatPanelProps = null
    vi.clearAllMocks()
  })

  it('ChatPanel node appears before the toggle button in DOM order (sibling combinator requirement)', () => {
    const { container } = renderHost()
    const chatPanel = container.querySelector('[data-testid="mock-chat-panel"]')
    const toggleBtn = container.querySelector('.hub-modal__chat-toggle')
    expect(chatPanel).not.toBeNull()
    expect(toggleBtn).not.toBeNull()

    // compareDocumentPosition returns a bitmask; DOCUMENT_POSITION_FOLLOWING (4)
    // means the argument comes after the node — i.e., chatPanel is before toggleBtn.
    const position = chatPanel!.compareDocumentPosition(toggleBtn!)
    // DOCUMENT_POSITION_FOLLOWING = 4
    expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })
})
