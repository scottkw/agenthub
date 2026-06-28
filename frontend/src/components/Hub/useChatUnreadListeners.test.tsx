import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// Mock RelayClient before importing the hook so the constructor is intercepted.
// The mock captures the callbacks object and exposes a close spy.
vi.mock('../../lib/relayClient', () => ({
  RelayClient: vi.fn().mockImplementation((_port: number, _id: string, callbacks: unknown) => ({
    callbacks,
    close: vi.fn(),
  })),
}))

// Mock accrueUnread from ChatPanel as a pure increment so tests assert on
// threading logic, not accrual math.
vi.mock('./ChatPanel', () => ({
  accrueUnread: vi.fn(
    (prev: { count: number; hasMention: boolean }, _msg: unknown, _id: string) => ({
      count: prev.count + 1,
      hasMention: false,
    }),
  ),
}))

import { RelayClient } from '../../lib/relayClient'
import { useChatUnreadListeners } from './useChatUnreadListeners'

// ---- Helpers ----------------------------------------------------------------

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
    workDir: '/home/user/project',
    ...overrides,
  }
}

const MockRelayClient = RelayClient as unknown as ReturnType<typeof vi.fn>

// ---- Tests ------------------------------------------------------------------

describe('useChatUnreadListeners', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('opens one RelayClient per non-modal session when relayPort > 0 and isActive', () => {
    const sessions = [
      makeSession({ id: 'sess-1' }),
      makeSession({ id: 'sess-2' }),
      makeSession({ id: 'sess-3' }),
    ]
    const onUnreadChange = vi.fn()

    renderHook(() =>
      useChatUnreadListeners(sessions, 8080, null, true, onUnreadChange),
    )

    // One RelayClient should be created for each session
    expect(MockRelayClient).toHaveBeenCalledTimes(3)
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-1', expect.any(Object))
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-2', expect.any(Object))
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-3', expect.any(Object))
  })

  it('excludes openModalSessionId from the subscription set', () => {
    const sessions = [
      makeSession({ id: 'sess-1' }),
      makeSession({ id: 'sess-modal' }),
      makeSession({ id: 'sess-3' }),
    ]
    const onUnreadChange = vi.fn()

    renderHook(() =>
      useChatUnreadListeners(sessions, 8080, 'sess-modal', true, onUnreadChange),
    )

    // Only 2 clients — sess-modal is excluded
    expect(MockRelayClient).toHaveBeenCalledTimes(2)
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-1', expect.any(Object))
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-3', expect.any(Object))
    // Verify the modal session was NOT subscribed
    const callArgs = MockRelayClient.mock.calls.map((c: unknown[]) => c[1])
    expect(callArgs).not.toContain('sess-modal')
  })

  it('calls onUnreadChange with correct (sessionId, count, hasMention) when onChat fires', () => {
    const sessions = [makeSession({ id: 'sess-a' })]
    const onUnreadChange = vi.fn()

    renderHook(() =>
      useChatUnreadListeners(sessions, 8080, null, true, onUnreadChange),
    )

    expect(MockRelayClient).toHaveBeenCalledTimes(1)
    // Retrieve the mock instance's captured callbacks
    const instance = MockRelayClient.mock.results[0].value as {
      callbacks: { onChat?: (msg: unknown) => void }
      close: ReturnType<typeof vi.fn>
    }
    const fakeMessage = { body: 'hello', mentions: [] }

    act(() => {
      instance.callbacks.onChat?.(fakeMessage)
    })

    expect(onUnreadChange).toHaveBeenCalledTimes(1)
    expect(onUnreadChange).toHaveBeenCalledWith('sess-a', 1, false)
  })

  it('closes every RelayClient on unmount (no leak)', () => {
    const sessions = [
      makeSession({ id: 'sess-x' }),
      makeSession({ id: 'sess-y' }),
    ]
    const onUnreadChange = vi.fn()

    const { unmount } = renderHook(() =>
      useChatUnreadListeners(sessions, 8080, null, true, onUnreadChange),
    )

    expect(MockRelayClient).toHaveBeenCalledTimes(2)

    const instances = MockRelayClient.mock.results.map(
      (r: { value: { close: ReturnType<typeof vi.fn> } }) => r.value,
    )

    unmount()

    // Both clients must have been closed
    for (const inst of instances) {
      expect(inst.close).toHaveBeenCalledTimes(1)
    }
  })

  it('constructs zero RelayClients when relayPort === 0', () => {
    const sessions = [
      makeSession({ id: 'sess-1' }),
      makeSession({ id: 'sess-2' }),
    ]
    const onUnreadChange = vi.fn()

    renderHook(() =>
      useChatUnreadListeners(sessions, 0, null, true, onUnreadChange),
    )

    expect(MockRelayClient).toHaveBeenCalledTimes(0)
  })

  it('constructs zero RelayClients when isActive is false', () => {
    const sessions = [makeSession({ id: 'sess-1' })]
    const onUnreadChange = vi.fn()

    renderHook(() =>
      useChatUnreadListeners(sessions, 8080, null, false, onUnreadChange),
    )

    expect(MockRelayClient).toHaveBeenCalledTimes(0)
  })
})
