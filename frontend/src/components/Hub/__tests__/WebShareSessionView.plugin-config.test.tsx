/**
 * WebShareSessionView.plugin-config.test.tsx — Phase 168 FIX-01 (#112)
 *
 * Restores live plugin-config + SSE hot-swap for web-share guests on the
 * /app/ surface, lost after the Phase 159 /sessions/{id} -> /app/ redirect.
 *
 * Tests (behavior block, 168-02-PLAN.md Task 2):
 *   1. A web guest (no Wails-provided pluginConfig prop) self-fetches
 *      /api/plugin-config?cap=<capToken> on mount and applies the returned
 *      config to TerminalPanel.
 *   2. An EventSource 'plugin-config' event updates the live config without
 *      remount/reload.
 *   3. When a Wails pluginConfig prop IS provided (desktop), no self-fetch
 *      or EventSource is created.
 *   4. On unmount, the fetch AbortController aborts and the EventSource
 *      closes.
 *
 * Architecture: TerminalPanel and ChatPanel are mocked (same technique as
 * WebShareSessionView.test.tsx) so xterm / WebSocket side effects never run
 * in jsdom. global.fetch and global.EventSource are stubbed per-test.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'

// ── Mock TerminalPanel ──────────────────────────────────────────────────────
// Captures pluginConfig so we can assert the effective (live-fetched or
// prop-provided) config reaches the terminal.

interface CapturedTerminalProps {
  pluginConfig?: unknown
  [key: string]: unknown
}

let capturedTerminalProps: CapturedTerminalProps | null = null

vi.mock('../../TerminalPanel', () => ({
  TerminalPanel: (props: CapturedTerminalProps) => {
    capturedTerminalProps = props
    return React.createElement('div', { 'data-testid': 'mock-terminal-panel' })
  },
}))

// ── Mock ChatPanel / ChatBadge ───────────────────────────────────────────────

vi.mock('../ChatPanel', () => ({
  ChatPanel: () => React.createElement('div', { 'data-testid': 'mock-chat-panel' }),
}))

vi.mock('../ChatBadge', () => ({
  ChatBadge: () => React.createElement('span', { 'data-testid': 'mock-chat-badge' }),
}))

// ── Import under test (after vi.mock — vitest hoists vi.mock automatically) ─

import { WebShareSessionView } from '../WebShareSessionView'

// ── Fake EventSource ─────────────────────────────────────────────────────────
// jsdom does not implement EventSource. This fake tracks every constructed
// instance (via `instances`) and every registered 'plugin-config' listener so
// tests can dispatch events and assert close() was called.

type PluginConfigListener = (ev: { data: string }) => void

class FakeEventSource {
  static instances: FakeEventSource[] = []
  url: string
  closed = false
  listeners: Record<string, PluginConfigListener[]> = {}

  constructor(url: string) {
    this.url = url
    FakeEventSource.instances.push(this)
  }

  addEventListener(type: string, listener: PluginConfigListener) {
    ;(this.listeners[type] ??= []).push(listener)
  }

  removeEventListener(type: string, listener: PluginConfigListener) {
    this.listeners[type] = (this.listeners[type] ?? []).filter((l) => l !== listener)
  }

  close() {
    this.closed = true
  }

  dispatch(type: string, data: unknown) {
    for (const listener of this.listeners[type] ?? []) {
      listener({ data: JSON.stringify(data) })
    }
  }
}

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
      act(() => {
        root.unmount()
      })
      container.remove()
    },
  }
}

async function flushMicrotasks() {
  await act(async () => {
    await Promise.resolve()
    await Promise.resolve()
  })
}

// ── Setup / teardown ─────────────────────────────────────────────────────────

let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  FakeEventSource.instances = []
  fetchMock = vi.fn().mockResolvedValue({
    ok: true,
    json: async () => ({ webglRenderer: true }),
  })
  vi.stubGlobal('fetch', fetchMock)
  vi.stubGlobal('EventSource', FakeEventSource as unknown as typeof EventSource)
})

afterEach(() => {
  capturedTerminalProps = null
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

// ── Tests ────────────────────────────────────────────────────────────────────

describe('WebShareSessionView — web-guest plugin-config self-fetch (FIX-01)', () => {
  it('self-fetches /api/plugin-config?cap=<capToken> on mount and applies it to TerminalPanel', async () => {
    const { unmount } = mountView({ sessionId: 'sess-123', capToken: 'tok-abc' })
    await flushMicrotasks()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url] = fetchMock.mock.calls[0] as [string]
    expect(url).toBe(`${window.location.origin}/api/plugin-config?cap=${encodeURIComponent('tok-abc')}`)

    expect(capturedTerminalProps?.pluginConfig).toEqual({ webglRenderer: true })
    unmount()
  })

  it('opens an EventSource on /api/plugin-config/stream?cap=<capToken>', async () => {
    const { unmount } = mountView({ sessionId: 'sess-123', capToken: 'tok-abc' })
    await flushMicrotasks()

    expect(FakeEventSource.instances).toHaveLength(1)
    expect(FakeEventSource.instances[0].url).toBe(
      `${window.location.origin}/api/plugin-config/stream?cap=${encodeURIComponent('tok-abc')}`,
    )
    unmount()
  })
})

describe('WebShareSessionView — SSE hot-swap (FIX-01)', () => {
  it('applies an updated config pushed via the plugin-config SSE event without remount', async () => {
    const { unmount } = mountView({ sessionId: 'sess-123', capToken: 'tok-abc' })
    await flushMicrotasks()

    expect(capturedTerminalProps?.pluginConfig).toEqual({ webglRenderer: true })

    const es = FakeEventSource.instances[0]
    act(() => {
      es.dispatch('plugin-config', { webglRenderer: false, unicode11: true })
    })

    expect(capturedTerminalProps?.pluginConfig).toEqual({ webglRenderer: false, unicode11: true })
    unmount()
  })
})

describe('WebShareSessionView — desktop (Wails pluginConfig prop) path (FIX-01)', () => {
  it('creates no fetch or EventSource when a Wails pluginConfig prop is provided', async () => {
    const providedConfig = { webglRenderer: true, unicode11: false } as never
    const { unmount } = mountView({ pluginConfig: providedConfig })
    await flushMicrotasks()

    expect(fetchMock).not.toHaveBeenCalled()
    expect(FakeEventSource.instances).toHaveLength(0)
    expect(capturedTerminalProps?.pluginConfig).toEqual(providedConfig)
    unmount()
  })
})

describe('WebShareSessionView — cleanup on unmount (FIX-01)', () => {
  it('aborts the fetch AbortController and closes the EventSource on unmount', async () => {
    const { unmount } = mountView({ sessionId: 'sess-123', capToken: 'tok-abc' })
    await flushMicrotasks()

    const [, init] = fetchMock.mock.calls[0] as [string, { signal: AbortSignal }]
    const signal = init.signal
    expect(signal.aborted).toBe(false)

    const es = FakeEventSource.instances[0]
    expect(es.closed).toBe(false)

    unmount()

    expect(signal.aborted).toBe(true)
    expect(es.closed).toBe(true)
  })
})
