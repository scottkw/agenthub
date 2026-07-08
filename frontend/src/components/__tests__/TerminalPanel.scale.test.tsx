/**
 * Phase 157 VIEW-04/05 — TerminalPanel guest honor + scale behavioral tests.
 * Phase 175-03 BUG-01 — extended with floor-aware guest viewport (readability
 * floor + horizontal-scroll fallback) behavioral tests.
 *
 * Guest path: a RelayClient onResize(cols,rows) callback calls term.resize(cols,rows)
 *   then applies a CSS transform via computeGuestViewport, toggling the
 *   .terminal-guest--scroll-x class when the natural scale falls below the
 *   readability floor. Guest never calls sendResize.
 * Host path: fitTerminal + sendResize on open preserved; no transform applied,
 *   scroll-x class never toggled.
 *
 * Source-inspection assertions pin the isGuest gate and computeGuestViewport import;
 * behavioral assertions use mock-captured RelayClient callbacks to drive onResize.
 *
 * jsdom limitation: clientWidth/clientHeight default to 0, so the natural scale is
 * always 0 (min(0/gridW, 0/gridH)) unless a test explicitly overrides clientWidth/
 * clientHeight via Object.defineProperty. Without an override, computeGuestViewport
 * clamps to DEFAULT_GUEST_MIN_SCALE with overflowX=true (0 is below any floor) — a
 * test-env artifact, not a bug. Tests that need to exercise the above-floor /
 * below-floor branches explicitly override clientWidth/clientHeight on the
 * .terminal-session-container node before invoking onResize.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import src from '../TerminalPanel.tsx?raw'

// ─── Hoisted shared state (shared between mock factories and test body) ───────

const hoisted = vi.hoisted(() => {
  return {
    // Captured from the most-recent RelayClient constructor call
    lastCallbacks: null as Record<string, unknown> | null,
    // Track sendResize calls so we can assert host vs guest behavior
    sendResizeCalls: [] as Array<[number, number]>,
    // Track term.resize calls
    termResizeCalls: [] as Array<[number, number]>,
    // The most recent terminal element (to inspect style.transform)
    lastElement: null as HTMLElement | null,
  }
})

// ─── Module mocks ─────────────────────────────────────────────────────────────

vi.mock('../../lib/relayClient', async () => {
  // Capture callbacks from the most recent RelayClient construction.
  // The onOpen callback simulates the WS-open event; tests call it directly.
  class MockRelayClient {
    constructor(_p: number, _id: string, cbs: Record<string, unknown>) {
      hoisted.lastCallbacks = cbs
      hoisted.sendResizeCalls = []
    }
    sendInput() {}
    sendResize(cols: number, rows: number) { hoisted.sendResizeCalls.push([cols, rows]) }
    sendChat() {}
    sendTyping() {}
    sendAliasSet() {}
    sendSessionInject() {}
    close() {}
  }
  return {
    RelayClient: MockRelayClient,
    MSG_OUTPUT: 0x01,
    MSG_RESIZE: 0x02,
    MSG_INPUT: 0x10,
    MSG_RESIZE2: 0x11,
    MSG_PING: 0x12,
    MSG_CHAT: 0x30,
    MSG_CHAT_SEND: 0x31,
    MSG_PRESENCE: 0x32,
    MSG_TYPING: 0x33,
    MSG_ALIAS_SET: 0x34,
    MSG_SESSION_INJECT: 0x35,
    MSG_INJECT_ERROR: 0x36,
    encodeInputFrame: () => new Uint8Array(),
    encodeResizeFrame: () => new Uint8Array(),
    encodeTypingFrame: () => new Uint8Array(),
    encodeAliasSetFrame: () => new Uint8Array(),
    encodeChatSendFrame: () => new Uint8Array(),
    encodeSessionInjectFrame: () => new Uint8Array(),
    parseServerFrame: () => ({ type: 'unknown' }),
  }
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
    public loadAddon = vi.fn()
    public open = vi.fn((container: HTMLElement) => {
      const el = document.createElement('div')
      container.appendChild(el)
      this.element = el
      hoisted.lastElement = el
    })
    public attachCustomKeyEventHandler = vi.fn()
    public onData = vi.fn(() => ({ dispose: () => {} }))
    public onResize = vi.fn(() => ({ dispose: () => {} }))
    public write = vi.fn()
    public dispose = vi.fn()
    public clearTextureAtlas = vi.fn()
    public refresh = vi.fn()
    public resize = vi.fn((cols: number, rows: number) => {
      // xterm updates cols/rows on resize — replicate for recomputeScale reads
      this.cols = cols
      this.rows = rows
      hoisted.termResizeCalls.push([cols, rows])
    })
    constructor(_opts: Record<string, unknown>) {
      this._core = {
        _renderService: {
          dimensions: { css: { cell: { width: 9, height: 17 } } },
          clear: vi.fn(),
        },
      }
    }
  }
  return { Terminal: MockTerminal }
})

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: class {
    proposeDimensions() { return { cols: 80, rows: 24 } }
    activate() {}
    fit() {}
  },
}))
vi.mock('@xterm/addon-image', () => ({
  ImageAddon: class { activate() {} dispose() {} },
}))
vi.mock('@xterm/addon-unicode11', () => ({
  Unicode11Addon: class { activate() {} },
}))
vi.mock('@xterm/addon-webgl', () => ({
  WebglAddon: class {
    onContextLoss() { return { dispose: () => {} } }
    activate() {}
    dispose() {}
  },
}))
vi.mock('@xterm/addon-clipboard', () => ({
  ClipboardAddon: class { activate() {} dispose() {} },
}))
vi.mock('@xterm/addon-search', () => ({
  SearchAddon: class {
    activate() {}
    dispose() {}
    onDidChangeResults() { return { dispose: () => {} } }
    findNext() {}
    findPrevious() {}
    clearDecorations() {}
  },
}))
vi.mock('@xterm/addon-web-links', () => ({
  WebLinksAddon: class { constructor(_h?: unknown, _o?: unknown) {} activate() {} dispose() {} },
}))
vi.mock('@xterm/addon-serialize', () => ({
  SerializeAddon: class { activate() {} dispose() {} serialize() { return '' } },
}))
vi.mock('@xterm/addon-progress', () => ({
  ProgressAddon: class { activate() {} dispose() {} onChange() { return { dispose: () => {} } } },
}))

vi.mock('../../lib/webglProbe', () => ({ isSoftwareWebGL: () => false }))
vi.mock('../../lib/isXtermFocused', () => ({ isXtermFocused: () => false }))
vi.mock('../../lib/touchScrollHandler', () => ({ attachTouchScroll: () => () => {} }))
vi.mock('../../lib/urlSafety', () => ({
  isAllowedScheme: () => true,
  getRisk: () => null,
}))
vi.mock('../../lib/openLink', () => ({
  openLink: () => {},
  isModifierPressed: () => false,
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  SetSearchConfig: () => Promise.resolve(),
}))
vi.mock('../../wailsjs/go/models', () => ({
  daemon: { SearchConfig: class { constructor(_o: unknown) {} } },
}))

// jsdom has no ResizeObserver. Stub with a no-op.
class MockResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
;(globalThis as unknown as { ResizeObserver: typeof MockResizeObserver }).ResizeObserver =
  MockResizeObserver

// ─── Import SUT after all mocks ───────────────────────────────────────────────
import { TerminalPanel } from '../TerminalPanel'
import { DEFAULT_GUEST_MIN_SCALE } from '../../lib/terminalScale'

// ─── Shared test helpers ──────────────────────────────────────────────────────

function makeBaseProps(overrides: Partial<React.ComponentProps<typeof TerminalPanel>> = {}) {
  return {
    sessionId: 'test-session',
    isActive: true,
    relayPort: 0,
    fontSize: 14,
    onFontSizeChange: () => {},
    theme: { background: '#000', foreground: '#fff' },
    ...overrides,
  } as React.ComponentProps<typeof TerminalPanel>
}

// ─── Source-inspection tests ──────────────────────────────────────────────────

describe('Phase 157 TerminalPanel — source gates (VIEW-04/05)', () => {
  it('imports computeGuestViewport from terminalScale (BUG-01 floor-aware helper)', () => {
    expect(src).toContain("from '../lib/terminalScale'")
    expect(src).toContain('computeGuestViewport')
  })

  it('computes isGuest = remote || !!wsURL', () => {
    expect(src).toMatch(/isGuest\s*=\s*remote\s*\|\|\s*!!wsURL/)
  })

  it('calls term.resize inside the onResize callback (guest honor)', () => {
    expect(src).toContain('term.resize(Math.max(1, cols), Math.max(1, rows))')
  })

  it('guest path: does not call client.sendResize from onOpen', () => {
    // The onOpen for guests is gated by isGuest — the sendResize call is in the !isGuest branch
    // Verify the ternary structure is present
    expect(src).toMatch(/isGuest\s*\?\s*undefined\s*:\s*\(\)\s*=>\s*\{/)
  })

  it('guest path: calls recomputeScale() after term.resize (Pitfall 5 order)', () => {
    // In the onResize callback block: term.resize THEN recomputeScale
    const onResizeBlock = src.slice(
      src.indexOf('onResize: isGuest'),
      src.indexOf('onResize: isGuest') + 300,
    )
    const resizeIdx = onResizeBlock.indexOf('term.resize')
    const scaleIdx = onResizeBlock.indexOf('recomputeScale()')
    expect(resizeIdx).toBeGreaterThanOrEqual(0)
    expect(scaleIdx).toBeGreaterThanOrEqual(0)
    expect(resizeIdx).toBeLessThan(scaleIdx)
  })

  it('guest ResizeObserver path does NOT call fitTerminal', () => {
    // Find the guest branch of the isActive effect and verify it calls recomputeScale, not fitTerminal
    const guestBranch = src.slice(
      src.indexOf('GUEST path (VIEW-04/05)'),
      src.indexOf('HOST path: fit terminal'),
    )
    expect(guestBranch).toContain('recomputeScale()')
    // Must not contain a fitTerminal() call (comments mentioning the name are OK)
    expect(guestBranch).not.toMatch(/fitTerminal\s*\(/)
  })

  it('host ResizeObserver path still calls fitTerminal', () => {
    const hostBranch = src.slice(
      src.indexOf('HOST path: fit terminal'),
      src.indexOf('Apply font size changes'),
    )
    expect(hostBranch).toContain('fitTerminal(')
  })

  it('CSS contains transform-origin: top left for .xterm', () => {
    // This is checked via a separate style.css read in style tests;
    // here we just confirm the recomputeScale output uses scale()
    expect(src).toContain("term.element.style.transform = `scale(${s})`")
  })
})

// ─── Behavioral tests — guest path ───────────────────────────────────────────

describe('Phase 157 TerminalPanel — guest path behavioral (VIEW-04/05)', () => {
  let container: HTMLElement
  let root: Root

  beforeEach(() => {
    hoisted.lastCallbacks = null
    hoisted.sendResizeCalls = []
    hoisted.termResizeCalls = []
    hoisted.lastElement = null
    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(() => {
    try { flushSync(() => root.unmount()) } catch { /* ignore */ }
    container.remove()
    vi.clearAllMocks()
  })

  function mountGuest(remote?: boolean, wsURL?: string) {
    flushSync(() => {
      root.render(
        React.createElement(TerminalPanel, makeBaseProps({ remote, wsURL })),
      )
    })
  }

  // VIEW-04: guest honors 0x02 → term.resize
  it('guest (remote=true): onResize callback calls term.resize with correct cols/rows', () => {
    mountGuest(true, undefined)
    expect(hoisted.lastCallbacks).not.toBeNull()
    const onResize = hoisted.lastCallbacks?.['onResize'] as
      | ((cols: number, rows: number) => void)
      | undefined
    expect(onResize).toBeTypeOf('function')

    flushSync(() => { onResize?.(80, 24) })

    expect(hoisted.termResizeCalls).toContainEqual([80, 24])
  })

  it('guest (wsURL set): onResize callback calls term.resize with correct cols/rows', () => {
    mountGuest(undefined, 'wss://host/sessions/s1/ws?cap=tok')
    const onResize = hoisted.lastCallbacks?.['onResize'] as
      | ((cols: number, rows: number) => void)
      | undefined
    expect(onResize).toBeTypeOf('function')

    flushSync(() => { onResize?.(132, 50) })

    expect(hoisted.termResizeCalls).toContainEqual([132, 50])
  })

  // VIEW-04: clamp to >= 1 (T-157-03 — zero-dim guard, xterm rejects resize(0,0))
  it('guest: clamps cols/rows to min 1 when zero-dim frame arrives', () => {
    mountGuest(true)
    const onResize = hoisted.lastCallbacks?.['onResize'] as
      | ((cols: number, rows: number) => void)
      | undefined

    flushSync(() => { onResize?.(0, 0) })

    expect(hoisted.termResizeCalls).toContainEqual([1, 1])
  })

  // VIEW-05: guest applies CSS transform after onResize
  it('guest: onResize applies a scale(…) transform to term.element', () => {
    mountGuest(true)
    const onResize = hoisted.lastCallbacks?.['onResize'] as
      | ((cols: number, rows: number) => void)
      | undefined

    flushSync(() => { onResize?.(80, 24) })

    expect(hoisted.lastElement?.style.transform).toMatch(/^scale\(/)
  })

  // D-03 invariant: guests never drive PTY resize
  it('guest (remote=true): sendResize is never called (onOpen is undefined or no-op)', () => {
    mountGuest(true)
    // Simulate WS open — even if onOpen fires, it must not call sendResize for guests
    const onOpen = hoisted.lastCallbacks?.['onOpen'] as (() => void) | undefined
    if (typeof onOpen === 'function') flushSync(() => { onOpen() })
    expect(hoisted.sendResizeCalls).toHaveLength(0)
  })

  it('guest (wsURL set): sendResize is never called', () => {
    mountGuest(undefined, 'wss://host/sessions/s/ws?cap=x')
    const onOpen = hoisted.lastCallbacks?.['onOpen'] as (() => void) | undefined
    if (typeof onOpen === 'function') flushSync(() => { onOpen() })
    expect(hoisted.sendResizeCalls).toHaveLength(0)
  })

  // Scale cap: transform never exceeds scale(1)
  it('guest: scale cap — transform never exceeds scale(1) even if container > grid', () => {
    mountGuest(true)
    const onResize = hoisted.lastCallbacks?.['onResize'] as
      | ((cols: number, rows: number) => void)
      | undefined

    // In jsdom clientWidth=0 so scale=0; just assert transform is set, not undefined
    flushSync(() => { onResize?.(80, 24) })

    const t = hoisted.lastElement?.style.transform ?? ''
    expect(t).toMatch(/^scale\(/)
    // Extract the scale value from 'scale(X)' — must be <= 1
    const match = t.match(/^scale\(([\d.]+)\)/)
    if (match) {
      const s = parseFloat(match[1])
      expect(s).toBeLessThanOrEqual(1)
    }
  })

  // BUG-01: narrow guest viewport — natural scale falls below the readability
  // floor, so the scale clamps at the floor and the horizontal-scroll fallback
  // class is applied instead of shrinking further.
  it('guest: narrow container clamps to the readability floor and adds the scroll-x class', () => {
    mountGuest(true)
    const node = container.querySelector('.terminal-session-container') as HTMLElement
    // grid = 80cols*9px x 24rows*17px = 720x408. A 50x50 container is far below
    // the floor at any scale > 0.
    Object.defineProperty(node, 'clientWidth', { value: 50, configurable: true })
    Object.defineProperty(node, 'clientHeight', { value: 50, configurable: true })
    const onResize = hoisted.lastCallbacks?.['onResize'] as
      | ((cols: number, rows: number) => void)
      | undefined

    flushSync(() => { onResize?.(80, 24) })

    expect(hoisted.lastElement?.style.transform).toBe(`scale(${DEFAULT_GUEST_MIN_SCALE})`)
    expect(node.classList.contains('terminal-guest--scroll-x')).toBe(true)
    expect(hoisted.sendResizeCalls).toHaveLength(0)
  })

  // BUG-01: wide guest viewport (container >= grid) — natural scale caps at 1.0
  // (never upscale, VIEW-04/05 invariant), which is always >= the floor, so no
  // horizontal-scroll fallback is applied.
  it('guest: wide container caps at scale(1) with no scroll-x class', () => {
    mountGuest(true)
    const node = container.querySelector('.terminal-session-container') as HTMLElement
    // grid = 720x408; a 2000x2000 container is far larger than the grid.
    Object.defineProperty(node, 'clientWidth', { value: 2000, configurable: true })
    Object.defineProperty(node, 'clientHeight', { value: 2000, configurable: true })
    const onResize = hoisted.lastCallbacks?.['onResize'] as
      | ((cols: number, rows: number) => void)
      | undefined

    flushSync(() => { onResize?.(80, 24) })

    expect(hoisted.lastElement?.style.transform).toBe('scale(1)')
    expect(node.classList.contains('terminal-guest--scroll-x')).toBe(false)
  })
})

// ─── Behavioral tests — host path ────────────────────────────────────────────

describe('Phase 157 TerminalPanel — host path invariance (VIEW-04/05)', () => {
  let container: HTMLElement
  let root: Root

  beforeEach(() => {
    hoisted.lastCallbacks = null
    hoisted.sendResizeCalls = []
    hoisted.termResizeCalls = []
    hoisted.lastElement = null
    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(() => {
    try { flushSync(() => root.unmount()) } catch { /* ignore */ }
    container.remove()
    vi.clearAllMocks()
  })

  function mountHost() {
    flushSync(() => {
      root.render(
        React.createElement(TerminalPanel, makeBaseProps({ remote: false, wsURL: undefined })),
      )
    })
  }

  // Host: sendResize fires from onOpen (existing behavior preserved)
  it('host: sendResize is called when onOpen fires (host drives PTY grid)', () => {
    mountHost()
    const onOpen = hoisted.lastCallbacks?.['onOpen'] as (() => void) | undefined
    expect(onOpen).toBeTypeOf('function')

    flushSync(() => { onOpen?.() })

    expect(hoisted.sendResizeCalls.length).toBeGreaterThan(0)
  })

  // Host: no onResize callback wired (host never honors 0x02)
  it('host: onResize callback is not provided (host never honors server 0x02)', () => {
    mountHost()
    const onResize = hoisted.lastCallbacks?.['onResize']
    // Host path: onResize must be undefined (Pitfall 7)
    expect(onResize).toBeUndefined()
  })

  // Host: no transform applied (host renders natively, never scaled)
  it('host: no CSS transform applied to term.element after open', () => {
    mountHost()
    // Simulate WS open
    const onOpen = hoisted.lastCallbacks?.['onOpen'] as (() => void) | undefined
    flushSync(() => { onOpen?.() })

    const t = hoisted.lastElement?.style.transform ?? ''
    // Host element must have NO scale transform
    expect(t).toBe('')
  })

  // BUG-01: host path never toggles the guest-only scroll-x fallback class —
  // the host PTY grid and its container are unaffected by the guest fix.
  it('host: never gains the terminal-guest--scroll-x class', () => {
    mountHost()
    const node = container.querySelector('.terminal-session-container') as HTMLElement
    const onOpen = hoisted.lastCallbacks?.['onOpen'] as (() => void) | undefined
    flushSync(() => { onOpen?.() })
    expect(node.classList.contains('terminal-guest--scroll-x')).toBe(false)
  })
})

// ─── CSS gate (style.css source inspection) ───────────────────────────────────

describe('Phase 157 style.css — guest scale CSS', () => {
  it('style.css contains transform-origin: top left for .xterm', () => {
    const { readFileSync } = require('node:fs')
    const { resolve, dirname } = require('node:path')
    const { fileURLToPath } = require('node:url')
    const __dir = dirname(fileURLToPath(import.meta.url))
    const css = readFileSync(resolve(__dir, '../../style.css'), 'utf-8')
    // Find the .xterm rule and verify it contains transform-origin: top left
    const xtermRuleIdx = css.indexOf('.xterm {')
    expect(xtermRuleIdx).toBeGreaterThan(-1)
    const ruleBlock = css.slice(xtermRuleIdx, css.indexOf('}', xtermRuleIdx) + 1)
    expect(ruleBlock).toContain('transform-origin: top left')
  })

  // BUG-01: the horizontal-scroll fallback class must enable overflow-x.
  it('style.css contains .terminal-guest--scroll-x with overflow-x: auto', () => {
    const { readFileSync } = require('node:fs')
    const { resolve, dirname } = require('node:path')
    const { fileURLToPath } = require('node:url')
    const __dir = dirname(fileURLToPath(import.meta.url))
    const css = readFileSync(resolve(__dir, '../../style.css'), 'utf-8')
    const ruleIdx = css.indexOf('.terminal-guest--scroll-x')
    expect(ruleIdx).toBeGreaterThan(-1)
    const ruleBlock = css.slice(ruleIdx, css.indexOf('}', ruleIdx) + 1)
    expect(ruleBlock).toContain('overflow-x: auto')
  })
})
