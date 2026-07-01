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

// Import mocked runtime bindings for assertion (Phase 166 FUI-05)
import { ClipboardSetText, BrowserOpenURL } from '../../wailsjs/wailsjs/runtime/runtime'

interface RenderOpts {
  browseEnabled?: boolean
  writeURL?: string
  writeCode?: string
  // Phase 166 FUI-04/FUI-05 Internet section
  funnelActive?: boolean
  funnelUrl?: string | null
  warmingUp?: boolean
  warmupTimedOut?: boolean
  onDisableFunnel?: () => void
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
        funnelActive: opts.funnelActive,
        funnelUrl: opts.funnelUrl,
        warmingUp: opts.warmingUp,
        warmupTimedOut: opts.warmupTimedOut,
        onDisableFunnel: opts.onDisableFunnel,
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

// ---------------------------------------------------------------------------
// Phase 166 — FUI-04/FUI-05 — Internet (public) section
// ---------------------------------------------------------------------------
const mockedClipboardSetText = ClipboardSetText as ReturnType<typeof vi.fn>
const mockedBrowserOpenURL = BrowserOpenURL as ReturnType<typeof vi.fn>

describe('SessionSharePanel — FUI-05 Internet (public) section', () => {
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

  it('does not render the Internet section when funnel is not engaged', () => {
    ;({ container, root } = renderPanel())
    expect(container!.querySelector('.hub-share-internet-section')).toBeNull()
  })

  it('warmingUp renders the "Starting up… (TLS warming up)" state and no live URL actions', () => {
    ;({ container, root } = renderPanel({ warmingUp: true }))
    const section = container!.querySelector('.hub-share-internet-section')
    expect(section).not.toBeNull()
    expect(container!.textContent).toMatch(/Starting up… \(TLS warming up\)/)
    // No live "Copy URL" action while warming up
    const copyUrl = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Copy URL',
    )
    expect(copyUrl).toBeUndefined()
  })

  it('funnelActive + funnelUrl renders the read-only URL with Copy URL / Open / QR actions', () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
    }))
    expect(container!.textContent).toMatch(/Public URL \(read-only\)/)
    expect(container!.innerHTML).toContain('https://sess.tail-scale.ts.net/')
    const labels = Array.from(container!.querySelectorAll('button')).map((b) => b.textContent?.trim())
    expect(labels).toContain('Copy URL')
    expect(labels.some((l) => l === 'Open')).toBe(true)
  })

  it('Copy URL uses ClipboardSetText (not navigator.clipboard)', async () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
    }))
    const copyBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Copy URL',
    ) as HTMLElement
    await flushSync(() => copyBtn.click())
    expect(mockedClipboardSetText).toHaveBeenCalledWith('https://sess.tail-scale.ts.net/')
  })

  it('Open uses BrowserOpenURL with the funnel URL', async () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
    }))
    const openBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Open' && b.getAttribute('aria-label')?.includes('public'),
    ) as HTMLElement
    await flushSync(() => openBtn.click())
    expect(mockedBrowserOpenURL).toHaveBeenCalledWith('https://sess.tail-scale.ts.net/')
  })

  it('does NOT render a public write link in the Internet section (D-12)', () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
      writeURL: 'https://example.com/w?cap=SECRET_WRITE',
    }))
    const section = container!.querySelector('.hub-share-internet-section')!
    // The write cap token must never appear inside the Internet (public) section.
    expect(section.innerHTML).not.toContain('SECRET_WRITE')
  })

  it('clicking "Disable internet share" invokes onDisableFunnel (single click, no confirm)', async () => {
    const onDisableFunnel = vi.fn()
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
      onDisableFunnel,
    }))
    const disableBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Disable internet share',
    ) as HTMLElement
    expect(disableBtn).not.toBeUndefined()
    await flushSync(() => disableBtn.click())
    expect(onDisableFunnel).toHaveBeenCalledTimes(1)
  })

  it('warmupTimedOut renders the timeout error copy', () => {
    ;({ container, root } = renderPanel({ warmupTimedOut: true }))
    expect(container!.querySelector('.hub-share-internet-section__error')).not.toBeNull()
    expect(container!.textContent).toMatch(/Connection timed out\. Try disabling and re-enabling\./)
  })
})
