/**
 * WebShareSessionView.test.tsx — Phase 155-03 (PARITY-01)
 *
 * Tests:
 *   1. Component renders without throwing given minimal props.
 *   2. wsURL construction: wss://{host}/sessions/{id}/ws?cap={encoded} is
 *      forwarded to BOTH TerminalPanel and ChatPanel (Pitfall 6 guard).
 *   3. capToken is forwarded to ChatPanel's capToken prop.
 *   4. apiBaseURL=window.location.origin is forwarded to ChatPanel.
 *   5. Root element carries hub-modal__body--interactive CSS class (parity gate).
 *   6. Chat toggle button carries hub-modal__chat-toggle class (parity gate).
 *
 * Architecture: TerminalPanel and ChatPanel are mocked so xterm / WebSocket side
 * effects do not run in jsdom. Props are captured via module-level variables so
 * assertions run after render without needing react-testing-library.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'

// ── Mock TerminalPanel ────────────────────────────────────────────────────────
// Captures all props forwarded by WebShareSessionView so we can assert on
// sessionId and wsURL without importing xterm (absent in jsdom).

interface CapturedTerminalProps {
  sessionId?: string
  relayPort?: number
  isActive?: boolean
  wsURL?: string
  [key: string]: unknown
}

let capturedTerminalProps: CapturedTerminalProps | null = null

vi.mock('../TerminalPanel', () => ({
  TerminalPanel: (props: CapturedTerminalProps) => {
    capturedTerminalProps = props
    return React.createElement('div', { 'data-testid': 'mock-terminal-panel' })
  },
}))

// ── Mock ChatPanel ─────────────────────────────────────────────────────────────
// Captures all props so we can assert wsURL, capToken, and apiBaseURL wiring.

interface CapturedChatPanelProps {
  sessionId?: string
  relayPort?: number
  open?: boolean
  wsURL?: string
  apiBaseURL?: string
  capToken?: string
  onUnreadChange?: (count: number, hasMention: boolean) => void
  [key: string]: unknown
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

// ── Mock ChatBadge ─────────────────────────────────────────────────────────────
vi.mock('./ChatBadge', () => ({
  ChatBadge: () => React.createElement('span', { 'data-testid': 'mock-chat-badge' }),
}))

// ── Import under test (after vi.mock — vitest hoists vi.mock automatically) ──

import { WebShareSessionView } from './WebShareSessionView'

// ── Helpers ───────────────────────────────────────────────────────────────────

function mountView(
  overrides: Partial<React.ComponentProps<typeof WebShareSessionView>> = {},
) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(
      <WebShareSessionView
        sessionId="sess-123"
        capToken="tok-abc"
        relayPort={0}
        {...overrides}
      />,
    )
  })
  return {
    container,
    root,
    unmount: () => {
      act(() => { root.unmount() })
      container.remove()
    },
  }
}

// ── Tests ────────────────────────────────────────────────────────────────────

afterEach(() => {
  capturedTerminalProps = null
  capturedChatPanelProps = null
  vi.clearAllMocks()
})

describe('WebShareSessionView — renders without error', () => {
  it('mounts without throwing given minimal props', () => {
    const { unmount } = mountView()
    // If we get here without throwing, the render succeeded.
    unmount()
  })

  it('renders the hub-modal__body--interactive root class (parity gate)', () => {
    const { container, unmount } = mountView()
    const root = container.querySelector('.hub-modal__body--interactive')
    expect(root).not.toBeNull()
    unmount()
  })

  it('renders the hub-modal__chat-toggle button (parity gate)', () => {
    const { container, unmount } = mountView()
    const toggle = container.querySelector('.hub-modal__chat-toggle')
    expect(toggle).not.toBeNull()
    unmount()
  })
})

describe('WebShareSessionView — wsURL construction (Pitfall 6 guard)', () => {
  it('constructs {wsScheme}://{host}/sessions/{id}/ws?cap={encoded} and forwards to TerminalPanel', () => {
    const { unmount } = mountView({ sessionId: 'sess-123', capToken: 'tok-abc' })
    // Derive expected URL from window.location so the test is hermetic regardless
    // of jsdom's configured origin (vitest configures http://localhost:3000 by
    // default). Phase 168 FIX-01: wsURL's scheme now derives from the resolved
    // origin's http/https scheme (ws/wss) instead of a hardcoded 'wss'.
    const wsScheme = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const expectedURL = `${wsScheme}://${window.location.host}/sessions/${encodeURIComponent('sess-123')}/ws?cap=${encodeURIComponent('tok-abc')}`
    expect(capturedTerminalProps?.wsURL).toBe(expectedURL)
    unmount()
  })

  it('constructs the same wsURL and forwards it to ChatPanel (Pitfall 6: both children need it)', () => {
    const { unmount } = mountView({ sessionId: 'sess-123', capToken: 'tok-abc' })
    const wsScheme = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const expectedURL = `${wsScheme}://${window.location.host}/sessions/${encodeURIComponent('sess-123')}/ws?cap=${encodeURIComponent('tok-abc')}`
    expect(capturedChatPanelProps?.wsURL).toBe(expectedURL)
    unmount()
  })

  it('percent-encodes special characters in sessionId and capToken', () => {
    const { unmount } = mountView({ sessionId: 'sess/123', capToken: 'tok=abc&x=1' })
    const wsScheme = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const expectedURL = `${wsScheme}://${window.location.host}/sessions/${encodeURIComponent('sess/123')}/ws?cap=${encodeURIComponent('tok=abc&x=1')}`
    expect(capturedTerminalProps?.wsURL).toBe(expectedURL)
    expect(capturedChatPanelProps?.wsURL).toBe(expectedURL)
    unmount()
  })

  it('wsURL shape matches {ws|wss}://{host}/sessions/{id}/ws?cap= pattern', () => {
    const { unmount } = mountView({ sessionId: 'sess-abc', capToken: 'cap-xyz' })
    const wsURL = capturedTerminalProps?.wsURL as string
    expect(wsURL).toMatch(/^wss?:\/\//)
    expect(wsURL).toContain('/sessions/')
    expect(wsURL).toContain('/ws?cap=')
    expect(wsURL).toBe(capturedChatPanelProps?.wsURL)
    unmount()
  })
})

describe('WebShareSessionView — ChatPanel prop wiring', () => {
  it('forwards capToken to ChatPanel', () => {
    const { unmount } = mountView({ capToken: 'tok-abc' })
    expect(capturedChatPanelProps?.capToken).toBe('tok-abc')
    unmount()
  })

  it('forwards apiBaseURL=window.location.origin to ChatPanel', () => {
    const { unmount } = mountView()
    // Derive expected origin from window.location so the test is hermetic.
    expect(capturedChatPanelProps?.apiBaseURL).toBe(window.location.origin)
    unmount()
  })

  it('forwards sessionId to both TerminalPanel and ChatPanel', () => {
    const { unmount } = mountView({ sessionId: 'my-session' })
    expect(capturedTerminalProps?.sessionId).toBe('my-session')
    expect(capturedChatPanelProps?.sessionId).toBe('my-session')
    unmount()
  })
})

describe('WebShareSessionView — chat toggle interaction', () => {
  it('chat toggle opens/closes the drawer (ChatPanel open prop flips)', () => {
    const { container, unmount } = mountView()
    // Initially closed
    const chatPanel = container.querySelector('[data-testid="mock-chat-panel"]') as HTMLElement
    expect(chatPanel?.getAttribute('data-open')).toBe('false')

    // Click the toggle to open
    const toggle = container.querySelector('.hub-modal__chat-toggle') as HTMLElement
    expect(toggle).not.toBeNull()
    act(() => {
      toggle.click()
    })

    // ChatPanel should now be open
    expect(container.querySelector('[data-open="true"]')).not.toBeNull()
    unmount()
  })
})
