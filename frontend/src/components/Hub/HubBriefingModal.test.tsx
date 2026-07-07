import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import type { RelayClientCallbacks } from '../../lib/relayClient'

// Source-inspection tests for HubBriefingModal (MODAL-04) — kept for contract coverage.
import raw from './HubBriefingModal.tsx?raw'

// ---- Mocks ----

// Mock Wails RPC — GetSessionStyledTailLines is the local-path tail fetch (Phase 139 / CARD-05).
vi.mock('../../wailsjs/go/main/App', () => ({
  GetSessionStyledTailLines: vi.fn().mockResolvedValue([[{ c: 'line-a' }], [{ c: 'line-b' }]]),
}))

// WR-07 / CR-03-01 / TAIL-01: mock RelayClient so behavioral tests can control
// the WS lifecycle (onOpen, onOutput, close) without a real WebSocket.
//
// The mock exposes:
//   - mockInstances: array of all constructed instances (in creation order)
//   - Each instance has: sendInput (spy), close (spy), triggerOpen(), triggerOutput(data)
//
// Usage: call triggerOpen() / triggerOutput() from the test to simulate WS events.

interface MockRelayClientInstance {
  sendInput: ReturnType<typeof vi.fn>
  close: ReturnType<typeof vi.fn>
  triggerOpen: () => void
  triggerOutput: (data: Uint8Array) => void
  opts?: { remote?: boolean }
}

const mockRelayInstances: MockRelayClientInstance[] = []

vi.mock('../../lib/relayClient', () => {
  class MockRelayClient {
    private callbacks: RelayClientCallbacks
    readonly opts?: { remote?: boolean }
    sendInput = vi.fn()
    close = vi.fn()

    constructor(
      _port: number,
      _sessionId: string,
      callbacks: RelayClientCallbacks,
      opts?: { remote?: boolean },
    ) {
      this.callbacks = callbacks
      this.opts = opts

      const instance: MockRelayClientInstance = {
        sendInput: this.sendInput,
        close: this.close,
        triggerOpen: () => {
          this.callbacks.onOpen?.()
        },
        triggerOutput: (data: Uint8Array) => {
          this.callbacks.onOutput(data)
        },
        opts,
      }
      mockRelayInstances.push(instance)
    }
  }

  return { RelayClient: MockRelayClient }
})

import { HubBriefingModal } from './HubBriefingModal'
import { GetSessionStyledTailLines } from '../../wailsjs/go/main/App'
import type { ITheme } from '@xterm/xterm'

// Minimal ITheme stub for HubBriefingModal's required theme prop (Phase 139 / CARD-05).
const STUB_THEME: ITheme = { foreground: '#c0caf5', background: '#1a1b26' }

// ---- Helpers ----

function makeSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'sess-1',
    cli: 'claude',
    name: 'Test Session',
    state: 'waiting',
    status: 'waiting',
    createdAt: new Date().toISOString(),
    hostname: '',
    webEnabled: false,
    viewerCount: 0,
    homeDir: false,
    browseEnabled: false,
    funnelActive: false,
    funnelWriteActive: false,
    workDir: '/home/user',
    ...overrides,
  }
}

function renderBriefing(props: {
  session?: SessionInfo
  relayPort?: number
  onClose?: () => void
  remote?: boolean
  theme?: ITheme
}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)

  act(() => {
    root.render(
      React.createElement(HubBriefingModal, {
        session: props.session ?? makeSession(),
        relayPort: props.relayPort ?? 51234,
        onClose: props.onClose ?? vi.fn(),
        remote: props.remote,
        theme: props.theme ?? STUB_THEME,
      }),
    )
  })

  return { container, root }
}

// ---- Source-inspection tests (MODAL-04) — kept for contract coverage ----

describe('HubBriefingModal (MODAL-04: tail fetch + send flow)', () => {
  it('MODAL-04: calls GetSessionStyledTailLines (fetches styled terminal tail for context display; Phase 139 CARD-05)', () => {
    expect(raw).toContain('GetSessionStyledTailLines')
  })

  it('MODAL-04: uses RelayClient to send input', () => {
    expect(raw).toContain('RelayClient')
  })

  it('MODAL-04: calls sendInput (delivers response text to session)', () => {
    expect(raw).toContain('sendInput')
  })

  it('MODAL-04: sends via onOpen callback (race-safe send — Pitfall 5: never send before WS is open)', () => {
    expect(raw).toContain('onOpen')
  })

  it('MODAL-04: textarea has maxLength={4096} (V5 input-validation guard)', () => {
    expect(raw).toContain('maxLength={4096}')
  })

  it('MODAL-04: Send button disabled when response text is empty (responseText.trim())', () => {
    expect(raw).toContain('responseText.trim()')
  })

  it('MODAL-04: button copy reads "Send Response"', () => {
    expect(raw).toContain('Send Response')
  })
})

// ---- CR-03-01: Behavioral tests — briefing send ordering + timeout cleanup (WR-07) ----

describe('HubBriefingModal CR-03-01: send ordering + timeout cleanup (behavioral)', () => {
  beforeEach(() => {
    mockRelayInstances.length = 0
    vi.useFakeTimers()
  })

  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('CR-03-01a: send path opens -> sendInput(text+newline) -> close in order', async () => {
    const onClose = vi.fn()
    const { container } = renderBriefing({ remote: false, onClose })

    // Wait for local tail fetch (local path: GetSessionTailLines)
    await act(async () => {
      await Promise.resolve()
    })

    // Type into the textarea and click Send
    const textarea = container.querySelector<HTMLTextAreaElement>('textarea')
    expect(textarea).not.toBeNull()

    act(() => {
      // Simulate typing
      Object.defineProperty(textarea!, 'value', {
        value: 'yes',
        writable: true,
        configurable: true,
      })
      textarea!.dispatchEvent(new Event('input', { bubbles: true }))
      textarea!.dispatchEvent(new Event('change', { bubbles: true }))
    })

    // Re-render to pick up value (simulate React's onChange)
    // Use a simpler approach: trigger the React onChange event
    act(() => {
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
        window.HTMLTextAreaElement.prototype,
        'value',
      )?.set
      nativeInputValueSetter?.call(textarea!, 'yes')
      textarea!.dispatchEvent(new Event('change', { bubbles: true }))
    })

    const sendBtn = container.querySelector<HTMLButtonElement>('.hub-modal__send-btn')
    expect(sendBtn).not.toBeNull()

    // Click Send — this creates a RelayClient and starts the Promise
    act(() => {
      sendBtn!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // At this point a RelayClient instance should exist but sendInput not yet called
    // (send only fires after onOpen)
    expect(mockRelayInstances.length).toBeGreaterThanOrEqual(1)
    const sendClient = mockRelayInstances[mockRelayInstances.length - 1]
    expect(sendClient.sendInput).not.toHaveBeenCalled()

    // Simulate the WS opening — sendInput should now be called
    act(() => {
      sendClient.triggerOpen()
    })

    expect(sendClient.sendInput).toHaveBeenCalledOnce()
    expect(sendClient.sendInput).toHaveBeenCalledWith(expect.stringMatching(/yes\n/))

    // Advance 100ms so the post-send timer fires (settles → close)
    await act(async () => {
      vi.advanceTimersByTime(100)
      await Promise.resolve()
    })

    expect(sendClient.close).toHaveBeenCalled()
    // onClose is called after successful send
    expect(onClose).toHaveBeenCalled()
  })

  it('CR-03-01b: timeout path closes the client and sends NO text (zero sendInput calls)', async () => {
    const onClose = vi.fn()
    const { container } = renderBriefing({ remote: false, onClose })

    await act(async () => {
      await Promise.resolve()
    })

    // Type text into the textarea via React's change event simulation
    const textarea = container.querySelector<HTMLTextAreaElement>('textarea')
    act(() => {
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
        window.HTMLTextAreaElement.prototype,
        'value',
      )?.set
      nativeInputValueSetter?.call(textarea!, 'will timeout')
      textarea!?.dispatchEvent(new Event('change', { bubbles: true }))
    })

    const sendBtn = container.querySelector<HTMLButtonElement>('.hub-modal__send-btn')
    act(() => {
      sendBtn!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    expect(mockRelayInstances.length).toBeGreaterThanOrEqual(1)
    const sendClient = mockRelayInstances[mockRelayInstances.length - 1]

    // Advance 5000ms so the CR-03 timeout fires (never called triggerOpen)
    await act(async () => {
      vi.advanceTimersByTime(5000)
      await Promise.resolve()
    })

    // The timeout path closes the WS and NEVER calls sendInput
    expect(sendClient.close).toHaveBeenCalled()
    expect(sendClient.sendInput).not.toHaveBeenCalled()
    // onClose is NOT called on timeout (send failed; user sees error)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('CR-03-01c: a late onOpen after timeout does NOT call sendInput (settled guard)', async () => {
    const { container } = renderBriefing({ remote: false })

    await act(async () => {
      await Promise.resolve()
    })

    const textarea = container.querySelector<HTMLTextAreaElement>('textarea')
    act(() => {
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
        window.HTMLTextAreaElement.prototype,
        'value',
      )?.set
      nativeInputValueSetter?.call(textarea!, 'late text')
      textarea!?.dispatchEvent(new Event('change', { bubbles: true }))
    })

    const sendBtn = container.querySelector<HTMLButtonElement>('.hub-modal__send-btn')
    act(() => {
      sendBtn!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    expect(mockRelayInstances.length).toBeGreaterThanOrEqual(1)
    const sendClient = mockRelayInstances[mockRelayInstances.length - 1]

    // Advance past the 5s timeout (settled = true)
    await act(async () => {
      vi.advanceTimersByTime(5000)
      await Promise.resolve()
    })

    // Simulate the late onOpen arriving AFTER the timeout
    act(() => {
      sendClient.triggerOpen()
    })

    // sendInput must NOT be called — the settled guard prevents post-abandon send
    expect(sendClient.sendInput).not.toHaveBeenCalled()
  })
})

// ---- TAIL-01: Behavioral tests — remote tail rendered from WS snapshot (WR-07) ----

describe('HubBriefingModal TAIL-01: remote tail from WS snapshot (behavioral)', () => {
  beforeEach(() => {
    mockRelayInstances.length = 0
    vi.useFakeTimers()
    vi.mocked(GetSessionStyledTailLines).mockClear()
  })

  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('TAIL-01a: remote=true renders tail lines from onOutput snapshot (not "No recent output available")', async () => {
    // Phase 139 / CARD-05: remote path now uses headless xterm + serializeAsHTML (no <pre>).
    // serializeAsHTML renders to a div.hub-modal__tail-remote via dangerouslySetInnerHTML.
    // The HTML output contains the text — check via innerHTML or container textContent.
    const remoteSession = makeSession({ id: 'r1', hostname: 'remote-host' })
    const { container } = renderBriefing({ session: remoteSession, remote: true })

    // A RelayClient should have been constructed for the tail fetch
    expect(mockRelayInstances.length).toBeGreaterThanOrEqual(1)
    const tailClient = mockRelayInstances[0]
    expect(tailClient.opts?.remote).toBe(true)

    // Simulate the WS opening and delivering scrollback snapshot bytes
    const encoder = new TextEncoder()
    act(() => {
      tailClient.triggerOpen()
      // Deliver some output bytes (plain text — no ANSI in this simple test)
      tailClient.triggerOutput(encoder.encode('line-one\nline-two\n'))
    })

    // Advance 500ms (past the 150ms idle timer in the remote tail path)
    await act(async () => {
      vi.advanceTimersByTime(500)
      await Promise.resolve()
    })

    // The remote tail should be rendered in the .hub-modal__tail-remote div
    // (serializeAsHTML output contains the text in spans or as text nodes)
    const tailDiv = container.querySelector('.hub-modal__tail-remote')
    expect(tailDiv).not.toBeNull()
    // serializeAsHTML output should include "line-one" text
    expect(tailDiv!.innerHTML).toContain('line-one')
    // Must NOT show the "No recent output available" placeholder
    expect(container.textContent).not.toContain('No recent output available')
  })

  it('TAIL-01b: remote=true does NOT call GetSessionStyledTailLines (local-only API)', async () => {
    const remoteSession = makeSession({ id: 'r2', hostname: 'peer-host' })
    renderBriefing({ session: remoteSession, remote: true })

    // Allow any async setup to settle
    await act(async () => {
      await Promise.resolve()
    })

    // GetSessionStyledTailLines must NOT be called — the remote path uses the WS snapshot
    expect(GetSessionStyledTailLines).not.toHaveBeenCalled()
  })

  it('TAIL-01c: remote=false (local) DOES call GetSessionStyledTailLines and NOT RelayClient for tail', async () => {
    // Phase 139 / CARD-05: local path now calls GetSessionStyledTailLines instead of GetSessionTailLines
    // Clear instances — the local path should not open a RelayClient for the tail
    mockRelayInstances.length = 0
    const localSession = makeSession({ id: 'loc1', hostname: '' })
    renderBriefing({ session: localSession, remote: false })

    await act(async () => {
      await Promise.resolve()
    })

    expect(GetSessionStyledTailLines).toHaveBeenCalledWith('loc1', 20)
    // No RelayClient should have been created (local path skips WS)
    expect(mockRelayInstances.length).toBe(0)
  })
})
