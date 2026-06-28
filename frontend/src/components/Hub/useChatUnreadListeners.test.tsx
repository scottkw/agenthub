/**
 * Tests for useChatUnreadListeners.
 *
 * @testing-library/react is not in this project's devDependencies, so we
 * mirror HubPanel.test.tsx's pattern: createRoot + act from React directly.
 * A thin HookRunner component calls the hook with whatever props are stored in
 * a mutable ref, allowing re-renders with new props (prop-update tests).
 */

// React 19 requires IS_REACT_ACT_ENVIRONMENT = true when using act() outside
// the built-in testing framework integrations. HubPanel.test.tsx relies on the
// same flag — set it before any imports so React picks it up at module init.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
;(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true

import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import type { Root } from 'react-dom/client'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// Mock RelayClient before importing the hook so the constructor is intercepted.
// Uses a regular `function` (not an arrow function) because arrow functions
// cannot be called with `new` — they are not constructor-compatible.
// The mock captures the callbacks object and exposes a close spy.
vi.mock('../../lib/relayClient', () => ({
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  RelayClient: vi.fn(function(this: any, _port: number, _id: string, callbacks: unknown) {
    this.callbacks = callbacks
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    this.close = vi.fn()
  }),
}))

// Mock accrueUnread from ChatPanel as a pure +1 increment so tests assert on
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

// Type alias for the mock instance shape produced by the constructor mock above.
type MockRelayInstance = {
  callbacks: { onChat?: (msg: unknown) => void; [key: string]: unknown }
  close: ReturnType<typeof vi.fn>
}

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

// Props bag threaded into the HookRunner component so we can re-render with
// new values without tearing down and re-creating the root.
interface HookProps {
  sessions: SessionInfo[]
  relayPort: number
  openModalSessionId: string | null
  isActive: boolean
  onUnreadChange: (sessionId: string, count: number, hasMention: boolean) => void
}

/** Minimal wrapper component that exercises the hook under test. */
function HookRunner(props: HookProps) {
  useChatUnreadListeners(
    props.sessions,
    props.relayPort,
    props.openModalSessionId,
    props.isActive,
    props.onUnreadChange,
  )
  return null
}

/** Mounts HookRunner and returns helpers for re-render, unmount. */
function mountHook(initialProps: HookProps): {
  rerender: (nextProps: HookProps) => void
  unmount: () => void
  root: Root
  container: HTMLDivElement
} {
  const container = document.createElement('div')
  document.body.appendChild(container)
  let root!: Root

  act(() => {
    root = createRoot(container)
    root.render(React.createElement(HookRunner, initialProps))
  })

  return {
    rerender: (nextProps: HookProps) => {
      act(() => {
        root.render(React.createElement(HookRunner, nextProps))
      })
    },
    unmount: () => {
      act(() => {
        root.unmount()
      })
      container.remove()
    },
    root,
    container,
  }
}

// ---- Tests ------------------------------------------------------------------

describe('useChatUnreadListeners', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

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

    const { unmount } = mountHook({
      sessions,
      relayPort: 8080,
      openModalSessionId: null,
      isActive: true,
      onUnreadChange,
    })

    expect(MockRelayClient).toHaveBeenCalledTimes(3)
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-1', expect.any(Object))
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-2', expect.any(Object))
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-3', expect.any(Object))

    unmount()
  })

  it('excludes openModalSessionId from the subscription set', () => {
    const sessions = [
      makeSession({ id: 'sess-1' }),
      makeSession({ id: 'sess-modal' }),
      makeSession({ id: 'sess-3' }),
    ]
    const onUnreadChange = vi.fn()

    const { unmount } = mountHook({
      sessions,
      relayPort: 8080,
      openModalSessionId: 'sess-modal',
      isActive: true,
      onUnreadChange,
    })

    // Only 2 clients — sess-modal is excluded
    expect(MockRelayClient).toHaveBeenCalledTimes(2)
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-1', expect.any(Object))
    expect(MockRelayClient).toHaveBeenCalledWith(8080, 'sess-3', expect.any(Object))

    const callArgs = MockRelayClient.mock.calls.map((c: unknown[]) => c[1])
    expect(callArgs).not.toContain('sess-modal')

    unmount()
  })

  it('calls onUnreadChange with correct (sessionId, count, hasMention) when onChat fires', () => {
    const sessions = [makeSession({ id: 'sess-a' })]
    const onUnreadChange = vi.fn()

    const { unmount } = mountHook({
      sessions,
      relayPort: 8080,
      openModalSessionId: null,
      isActive: true,
      onUnreadChange,
    })

    expect(MockRelayClient).toHaveBeenCalledTimes(1)

    // For constructor mocks, instances are in mock.instances (the `this` objects
    // produced by `new RelayClient(...)`), not mock.results[i].value.
    const instance = MockRelayClient.mock.instances[0] as MockRelayInstance
    const fakeMessage = { body: 'hello', mentions: [] }

    act(() => {
      instance.callbacks.onChat?.(fakeMessage)
    })

    expect(onUnreadChange).toHaveBeenCalledTimes(1)
    expect(onUnreadChange).toHaveBeenCalledWith('sess-a', 1, false)

    unmount()
  })

  it('closes every RelayClient on unmount (no leak)', () => {
    const sessions = [
      makeSession({ id: 'sess-x' }),
      makeSession({ id: 'sess-y' }),
    ]
    const onUnreadChange = vi.fn()

    const { unmount } = mountHook({
      sessions,
      relayPort: 8080,
      openModalSessionId: null,
      isActive: true,
      onUnreadChange,
    })

    expect(MockRelayClient).toHaveBeenCalledTimes(2)

    // For constructor mocks, instances are in mock.instances.
    const instances = MockRelayClient.mock.instances as MockRelayInstance[]

    unmount()

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

    const { unmount } = mountHook({
      sessions,
      relayPort: 0,
      openModalSessionId: null,
      isActive: true,
      onUnreadChange,
    })

    expect(MockRelayClient).toHaveBeenCalledTimes(0)
    unmount()
  })

  it('constructs zero RelayClients when isActive is false', () => {
    const sessions = [makeSession({ id: 'sess-1' })]
    const onUnreadChange = vi.fn()

    const { unmount } = mountHook({
      sessions,
      relayPort: 8080,
      openModalSessionId: null,
      isActive: false,
      onUnreadChange,
    })

    expect(MockRelayClient).toHaveBeenCalledTimes(0)
    unmount()
  })
})
