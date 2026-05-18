/**
 * Phase 112 UI-01 — Behavioral test for onContextLoss handler order.
 *
 * Mocks @xterm/addon-webgl so we can:
 *   - capture the onContextLoss listener that TerminalPanel registers
 *   - record dispose() invocations on a shared timeline
 *   - fire the captured listener synchronously (simulating xterm's emitter)
 *   - assert that the parent-prop callback (onWebGLContextLost) is invoked
 *     with 'context-loss' BEFORE webglAddon.dispose() runs.
 *
 * RED on main (pre-fix): current source disposes first; the synchronous
 * teardown aborts the continuation so parentCb may not even run, OR it runs
 * after dispose — both cases fail the load-bearing notify-before-dispose
 * timeline assertion.
 *
 * GREEN after Task 2 reorder (RESEARCH §5 pattern): notify first, then
 * queueMicrotask-deferred dispose.
 *
 * Note (jsdom limitation): xterm Terminal can't fully render under jsdom
 * because it requires real <canvas> + WebGL context. We mock @xterm/xterm
 * minimally so the hot-swap useEffect runs to the WebglAddon registration.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'

// vi.mock() factories are hoisted above all other top-level code, so any
// classes / state they reference must also be hoisted. vi.hoisted() is the
// official escape hatch.
const hoisted = vi.hoisted(() => {
  const timeline: string[] = []
  type MockWebglAddon = {
    capturedOnContextLoss: (() => void) | null
    dispose: ReturnType<typeof import('vitest')['vi']['fn']>
    onContextLoss: (cb: () => void) => { dispose: () => void }
    activate: (t: unknown) => void
  }
  const state: { lastWebglAddon: MockWebglAddon | null } = { lastWebglAddon: null }
  return { timeline, state }
})

vi.mock('@xterm/addon-webgl', async () => {
  const { vi } = await import('vitest')
  class MockWebglAddon {
    public capturedOnContextLoss: (() => void) | null = null
    public dispose = vi.fn(() => {
      hoisted.timeline.push('dispose')
    })
    public onContextLoss(cb: () => void): { dispose: () => void } {
      this.capturedOnContextLoss = cb
      return { dispose: () => {} }
    }
    public activate(_t: unknown): void { /* no-op */ }
    constructor() {
      hoisted.state.lastWebglAddon = this as unknown as typeof hoisted.state.lastWebglAddon
    }
  }
  return { WebglAddon: MockWebglAddon }
})

vi.mock('@xterm/xterm', async () => {
  const { vi } = await import('vitest')
  class MockTerminal {
    public cols = 80
    public rows = 24
    public options: Record<string, unknown> = {}
    public unicode = { activeVersion: '6' as string }
    public element: HTMLElement | null = null
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    public _core: any
    public loadAddon = vi.fn((addon: { activate?: (t: unknown) => void }) => {
      if (typeof addon?.activate === 'function') addon.activate(this)
    })
    public open = vi.fn((container: HTMLElement) => {
      this.element = document.createElement('div')
      container.appendChild(this.element)
    })
    public attachCustomKeyEventHandler = vi.fn()
    public onData = vi.fn(() => ({ dispose: () => {} }))
    public onResize = vi.fn(() => ({ dispose: () => {} }))
    public write = vi.fn()
    public dispose = vi.fn()
    public clearTextureAtlas = vi.fn()
    public refresh = vi.fn()
    public resize = vi.fn()
    constructor(_opts: Record<string, unknown>) {
      this._core = {
        _renderService: {
          dimensions: { css: { cell: { width: 9, height: 17 } } },
          clear: vi.fn(),
        },
        _createRenderer: vi.fn(),
      }
    }
  }
  return { Terminal: MockTerminal }
})

// Stub the other addons the hot-swap effect may construct/dispose.
vi.mock('@xterm/addon-fit', () => ({
  FitAddon: class { activate(_t: unknown) {} proposeDimensions() { return { cols: 80, rows: 24 } } },
}))
vi.mock('@xterm/addon-image', () => ({
  ImageAddon: class { activate(_t: unknown) {} dispose() {} },
}))
vi.mock('@xterm/addon-unicode11', () => ({
  Unicode11Addon: class { activate(_t: unknown) {} },
}))
vi.mock('@xterm/addon-clipboard', () => ({
  ClipboardAddon: class { activate(_t: unknown) {} dispose() {} },
}))
vi.mock('@xterm/addon-search', () => ({
  SearchAddon: class {
    activate(_t: unknown) {}
    dispose() {}
    onDidChangeResults() { return { dispose: () => {} } }
    findNext() {}
    findPrevious() {}
    clearDecorations() {}
  },
}))
vi.mock('@xterm/addon-web-links', () => ({
  WebLinksAddon: class { constructor(_h?: unknown, _o?: unknown) {} activate(_t: unknown) {} dispose() {} },
}))
vi.mock('@xterm/addon-serialize', () => ({
  SerializeAddon: class { activate(_t: unknown) {} dispose() {} serialize() { return '' } },
}))
vi.mock('@xterm/addon-progress', () => ({
  ProgressAddon: class { activate(_t: unknown) {} dispose() {} onChange() { return { dispose: () => {} } } },
}))

// Stub RelayClient (it tries to open a real WebSocket).
vi.mock('../../lib/relayClient', () => ({
  RelayClient: class {
    constructor(_p: number, _id: string, _cbs: unknown) {}
    sendInput() {}
    sendResize() {}
    close() {}
  },
  MSG_OUTPUT: 0x01,
  MSG_RESIZE: 0x02,
  MSG_INPUT: 0x10,
  MSG_RESIZE2: 0x11,
  MSG_PING: 0x12,
  encodeInputFrame: () => new Uint8Array(),
  encodeResizeFrame: () => new Uint8Array(),
  parseServerFrame: () => ({ type: 0 }),
}))

vi.mock('../../lib/webglProbe', () => ({
  isSoftwareWebGL: () => false,
}))

vi.mock('../../wailsjs/go/main/App', () => ({
  SetSearchConfig: () => Promise.resolve(),
}))

vi.mock('../../wailsjs/go/models', () => ({
  daemon: {
    SearchConfig: class { constructor(_o: unknown) {} },
  },
}))

// jsdom doesn't ship ResizeObserver.
class MockResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
;(globalThis as unknown as { ResizeObserver: typeof MockResizeObserver }).ResizeObserver = MockResizeObserver

// ---------- SUT ----------
// Import AFTER the vi.mock() calls above so the SUT picks up the mocks.
import { TerminalPanel } from '../TerminalPanel'

describe('Phase 112 UI-01: TerminalPanel onContextLoss notify-before-dispose (behavioral)', () => {
  let container: HTMLElement
  let root: Root
  let parentCb: ReturnType<typeof vi.fn>

  beforeEach(() => {
    hoisted.timeline.length = 0
    hoisted.state.lastWebglAddon = null
    parentCb = vi.fn((reason: string) => {
      hoisted.timeline.push(`notify:${reason}`)
    })
    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(() => {
    try { flushSync(() => root.unmount()) } catch { /* ignore */ }
    container.remove()
    vi.clearAllMocks()
  })

  function mountPanel() {
    flushSync(() => {
      root.render(
        React.createElement(TerminalPanel, {
          sessionId: 'test-session',
          isActive: true,
          relayPort: 0,
          fontSize: 14,
          onFontSizeChange: () => {},
          theme: { background: '#000', foreground: '#fff' },
          pluginConfig: {
            webgl: true,
            clipboard: false,
            unicode11: false,
            image: false,
            search: false,
            webLinks: false,
            serialize: false,
            progress: false,
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
          } as any,
          onWebGLContextLost: parentCb as unknown as (
            reason: 'context-loss' | 'software-rasterized',
          ) => void,
        }),
      )
    })
  }

  it('registers an onContextLoss listener with the WebglAddon instance', () => {
    mountPanel()
    expect(hoisted.state.lastWebglAddon).not.toBeNull()
    expect(hoisted.state.lastWebglAddon!.capturedOnContextLoss).toBeTypeOf('function')
  })

  it('invokes parent onWebGLContextLost("context-loss") when xterm emitter fires', async () => {
    mountPanel()
    const cb = hoisted.state.lastWebglAddon!.capturedOnContextLoss!
    cb()
    // Flush microtasks so any deferred dispose runs.
    await Promise.resolve()
    await Promise.resolve()
    expect(parentCb).toHaveBeenCalledTimes(1)
    expect(parentCb).toHaveBeenCalledWith('context-loss')
  })

  it('records notify BEFORE dispose in the shared call-order timeline (load-bearing — Issue #55 root cause)', async () => {
    mountPanel()
    const cb = hoisted.state.lastWebglAddon!.capturedOnContextLoss!
    cb()
    // Flush microtasks so the deferred dispose runs.
    await Promise.resolve()
    await Promise.resolve()
    const notifyIdx = hoisted.timeline.indexOf('notify:context-loss')
    const disposeIdx = hoisted.timeline.indexOf('dispose')
    expect(notifyIdx, 'parent callback must have run').toBeGreaterThanOrEqual(0)
    expect(disposeIdx, 'dispose must have been called (deferred)').toBeGreaterThanOrEqual(0)
    expect(notifyIdx, 'notify must precede dispose — Issue #55 root cause').toBeLessThan(disposeIdx)
  })
})
