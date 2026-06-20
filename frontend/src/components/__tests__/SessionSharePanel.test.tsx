/**
 * Phase 137 / SHARE-06 — SessionSharePanel simplified contract (post-CAP-05).
 *
 * The CAP-05 two-gate tests (owner-write prop, viewer opt-in state, confirm dialog)
 * are retired in Phase 137 — the write link is now always shown when writeURL/writeCode
 * are provided (no viewer opt-in gate). The browseEnabled prop controls scope text only.
 *
 * Surviving behaviors:
 *   - Write link row always rendered when writeURL/writeCode provided (no gate)
 *   - Scope text reflects browseEnabled prop: "Watch only" vs "Watch + browse files"
 *   - QR/clipboard actions work (SHARE-04 — unchanged)
 *   - Read code renders as plain text (unchanged)
 *   - "Join code:" label present (unchanged)
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { SessionSharePanel } from '../SessionSharePanel'

// Mock Wails runtime modules used by SessionSharePanel
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  GetCapabilityQRCode: vi.fn().mockResolvedValue(''),
}))

interface RenderOpts {
  browseEnabled?: boolean
  writeURL?: string
  writeCode?: string
}

function renderPanel(opts: RenderOpts = {}) {
  const c = document.createElement('div')
  document.body.appendChild(c)
  const r = createRoot(c)
  flushSync(() => {
    r.render(
      React.createElement(SessionSharePanel, {
        sessionId: 'sess-1',
        readURL: 'https://example.com/r',
        writeURL: opts.writeURL ?? 'https://example.com/w',
        readCode: 'read-code',
        writeCode: opts.writeCode ?? 'write-code',
        browseEnabled: opts.browseEnabled ?? false,
      })
    )
  })
  return { container: c, root: r }
}

describe('SessionSharePanel — simplified write link (post-CAP-05)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

  afterEach(() => {
    if (root) {
      flushSync(() => root!.unmount())
      root = undefined
    }
    if (container) {
      container.remove()
      container = undefined
    }
    vi.clearAllMocks()
  })

  it('write link row always rendered when writeURL/writeCode are provided (no gate)', () => {
    ;({ container, root } = renderPanel({
      writeURL: 'https://example.com/w?cap=WRITE_TOKEN_ALWAYS_VISIBLE',
    }))
    // The write URL must be in the DOM immediately — no opt-in required
    expect(container!.innerHTML).toContain('WRITE_TOKEN_ALWAYS_VISIBLE')
  })

  it('write code always rendered when writeCode is provided (no gate)', () => {
    ;({ container, root } = renderPanel({ writeCode: 'explicit-write-code' }))
    expect(container!.textContent).toContain('explicit-write-code')
  })

  it('no locked placeholder row present (CAP-05 two-gate removed)', () => {
    ;({ container, root } = renderPanel())
    // The old two-gate surfaced a locked placeholder when gates were closed.
    // Post-CAP-05 there should be no locked/hidden placeholder.
    const lockedRow = container!.querySelector('[data-testid="full-access-link-locked"]')
    expect(lockedRow).toBeNull()
  })
})

describe('SessionSharePanel — scope text (browseEnabled prop)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

  afterEach(() => {
    if (root) {
      flushSync(() => root!.unmount())
      root = undefined
    }
    if (container) {
      container.remove()
      container = undefined
    }
    vi.clearAllMocks()
  })

  it('browseEnabled=false shows "watch only" scope text (no file access)', () => {
    ;({ container, root } = renderPanel({ browseEnabled: false }))
    const text = container!.textContent ?? ''
    // Should state that view-only has no file access
    expect(text).toMatch(/cannot send input or browse files|watch only|no file access/i)
  })

  it('browseEnabled=true reflects file browsing in scope text', () => {
    ;({ container, root } = renderPanel({ browseEnabled: true }))
    const text = container!.textContent ?? ''
    // Should reference browsing or file access in scope
    expect(text).toMatch(/browse|file access|full control/i)
  })
})

describe('SessionSharePanel — join code text display (unchanged)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

  afterEach(() => {
    if (root) {
      flushSync(() => root!.unmount())
      root = undefined
    }
    if (container) {
      container.remove()
      container = undefined
    }
    vi.clearAllMocks()
  })

  it('renders the read code as plain text immediately (no user click required)', () => {
    ;({ container, root } = renderPanel())
    expect(container!.textContent).toContain('read-code')
    expect(container!.textContent).toContain('Join code:')
  })

  it('renders the write code as plain text (no gate in post-CAP-05 panel)', () => {
    ;({ container, root } = renderPanel({ writeCode: 'write-code' }))
    expect(container!.textContent).toContain('write-code')
  })
})

// v3.5 UAT relabel (#24): scope clarity assertions (unchanged from prior version).
describe('SessionSharePanel — link scope clarity (v3.5 relabel)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

  afterEach(() => {
    if (root) {
      flushSync(() => root!.unmount())
      root = undefined
    }
    if (container) {
      container.remove()
      container = undefined
    }
    vi.clearAllMocks()
  })

  it('states the View-Only scope: session view only, no file access', () => {
    ;({ container, root } = renderPanel())
    expect(container!.textContent).toContain('cannot send input or browse files')
  })

  it('states the Full Access scope: full session control plus file browsing', () => {
    ;({ container, root } = renderPanel())
    expect(container!.textContent).toContain('Full control of the live session')
  })
})
