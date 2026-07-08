/**
 * Phase 173 / SM-04 + SM-06 — InternetReadOnlyTab tests.
 *
 * Redistributed from SessionSharePanel.test.tsx's FUI-05 Internet-section and
 * FNL-08 reusable-code assertions. Highest-value new test (SM-04/T-173-01):
 * the public-write markers (`hub-funnel-write-gate` class + hold-to-confirm
 * button) are ABSENT from this tab's rendered output.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { InternetReadOnlyTab } from '../SessionShare/InternetReadOnlyTab'

vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  GetCapabilityQRCode: vi.fn().mockResolvedValue(''),
}))

import { ClipboardSetText, BrowserOpenURL } from '../../wailsjs/wailsjs/runtime/runtime'

interface RenderOpts {
  funnelActive?: boolean
  funnelUrl?: string | null
  warmingUp?: boolean
  warmupTimedOut?: boolean
  publicReadCode?: string | null
  onDisableFunnel?: () => void
}

function renderTab(opts: RenderOpts = {}) {
  const c = document.createElement('div')
  document.body.appendChild(c)
  const r = createRoot(c)
  flushSync(() => {
    r.render(
      React.createElement(InternetReadOnlyTab, {
        funnelActive: opts.funnelActive,
        funnelUrl: opts.funnelUrl,
        warmingUp: opts.warmingUp,
        warmupTimedOut: opts.warmupTimedOut,
        publicReadCode: opts.publicReadCode,
        onDisableFunnel: opts.onDisableFunnel,
      }),
    )
  })
  return { container: c, root: r }
}

const mockedClipboardSetText = ClipboardSetText as ReturnType<typeof vi.fn>
const mockedBrowserOpenURL = BrowserOpenURL as ReturnType<typeof vi.fn>

describe('InternetReadOnlyTab', () => {
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

  it('warmingUp renders the "Starting up… (TLS warming up)" state and no live URL card', () => {
    ;({ container, root } = renderTab({ warmingUp: true }))
    expect(container!.textContent).toMatch(/Starting up… \(TLS warming up\)/)
    expect(container!.querySelector('.share-linkcard')).toBeNull()
  })

  it('warmupTimedOut renders the timeout error copy', () => {
    ;({ container, root } = renderTab({ warmupTimedOut: true }))
    expect(container!.querySelector('.hub-share-internet-section__error')).not.toBeNull()
    expect(container!.textContent).toMatch(/Connection timed out\. Try disabling and re-enabling\./)
  })

  it('funnelActive + publicReadCode renders a ShareLinkCard with the reusable /join entry URL (not the raw cap link)', () => {
    ;({ container, root } = renderTab({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=RO_TOKEN',
      publicReadCode: 'PUB1234',
    }))
    const card = container!.querySelector('.share-linkcard')
    expect(card).not.toBeNull()
    expect(container!.textContent).toMatch(/Public URL \(read-only\)/)
    expect(container!.innerHTML).toContain('https://sess.tail-scale.ts.net/join?code=PUB1234')
    // The raw capability token and the /sessions/{id} viewer path must NEVER be
    // handed to the public guest.
    expect(container!.innerHTML).not.toContain('RO_TOKEN')
    expect(container!.innerHTML).not.toContain('/sessions/abc123')
    expect(container!.textContent).toContain('Public join code (reusable):')
    const codeEls = Array.from(container!.querySelectorAll('[data-testid="join-code-text"]'))
    expect(codeEls.some((el) => el.textContent === 'PUB1234')).toBe(true)
  })

  it('Copy copies the reusable /join entry URL (ClipboardSetText, not the raw cap link)', async () => {
    ;({ container, root } = renderTab({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=RO_TOKEN',
      publicReadCode: 'PUB1234',
    }))
    const copyBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Copy',
    ) as HTMLElement
    await flushSync(() => copyBtn.click())
    expect(mockedClipboardSetText).toHaveBeenCalledWith('https://sess.tail-scale.ts.net/join?code=PUB1234')
  })

  it('Open opens the reusable /join entry URL (BrowserOpenURL, not the raw cap link)', async () => {
    ;({ container, root } = renderTab({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=RO_TOKEN',
      publicReadCode: 'PUB1234',
    }))
    const openBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Open',
    ) as HTMLElement
    await flushSync(() => openBtn.click())
    expect(mockedBrowserOpenURL).toHaveBeenCalledWith('https://sess.tail-scale.ts.net/join?code=PUB1234')
  })

  it('does NOT render a public join code row when publicReadCode is absent (degenerate fallback)', () => {
    ;({ container, root } = renderTab({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
    }))
    expect(container!.textContent).toMatch(/Public URL \(read-only\)/)
    expect(container!.textContent).not.toContain('Public join code (reusable):')
  })

  it('clicking "Disable internet share" invokes onDisableFunnel (single click, no confirm)', async () => {
    const onDisableFunnel = vi.fn()
    ;({ container, root } = renderTab({
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

  // SM-04 / T-173-01 negative wall-off assertion — the write-gate markup and
  // hold-to-confirm button must NEVER appear in the Internet Read-Only tab.
  it('SM-04: the public-write gate markup and hold-to-confirm button are ABSENT', () => {
    ;({ container, root } = renderTab({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
      publicReadCode: 'PUB1234',
    }))
    expect(container!.querySelector('.hub-funnel-write-gate')).toBeNull()
    expect(container!.querySelector('.hub-funnel-write-gate__hold-btn')).toBeNull()
    expect(container!.textContent).not.toContain('PUBLIC WRITE ACCESS — COMMAND EXECUTION')
    expect(container!.textContent).not.toContain('Enable public write access')
  })
})
