/**
 * Phase 173 / SM-06 — ShareLinkCard structure + QR-target + desc-beneath
 * contract.
 *
 * Assertions are attribute/text/class based only (owner is colorblind; see
 * project memory) — no computed-color checks.
 *
 * Locks the Information-Disclosure mitigation (T-173-02): the QR fetch
 * target must be the join-code exchange URL (/join?code=...), never the raw
 * capability token.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { ShareLinkCard } from '../SessionShare/ShareLinkCard'

// Mock Wails runtime modules used by ShareLinkCard (same modules
// SessionSharePanel.tsx mocks — resolved paths are identical regardless of
// which component imports them).
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  GetCapabilityQRCode: vi.fn().mockResolvedValue('deadbeef'),
}))

import { ClipboardSetText, BrowserOpenURL } from '../../wailsjs/wailsjs/runtime/runtime'
import { GetCapabilityQRCode } from '../../wailsjs/go/main/App'

const mockedClipboardSetText = ClipboardSetText as ReturnType<typeof vi.fn>
const mockedBrowserOpenURL = BrowserOpenURL as ReturnType<typeof vi.fn>
const mockedGetCapabilityQRCode = GetCapabilityQRCode as ReturnType<typeof vi.fn>

const FULL_URL = 'https://example.ts.net/sessions/abc123?cap=SUPER_SECRET_TOKEN'
const CODE = 'JOIN-CODE-1'
const DESCRIPTION = 'Watch the live session only — cannot send input or browse files.'

function renderCard() {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(ShareLinkCard, {
        title: 'Read-Only Link',
        url: FULL_URL,
        code: CODE,
        description: DESCRIPTION,
      })
    )
  })
  return { container, root }
}

describe('ShareLinkCard — structure contract (SM-06)', () => {
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

  it('renders the title text', () => {
    ;({ container, root } = renderCard())
    expect(container!.textContent).toContain('Read-Only Link')
  })

  it('renders .share-linkcard__url with title= equal to the full URL (CSS-truncated, not JS-truncated)', () => {
    ;({ container, root } = renderCard())
    const urlEl = container!.querySelector('.share-linkcard__url')
    expect(urlEl).not.toBeNull()
    expect(urlEl!.getAttribute('title')).toBe(FULL_URL)
    expect(urlEl!.textContent).toBe(FULL_URL)
  })

  it('renders Copy, Open, and QR action buttons', () => {
    ;({ container, root } = renderCard())
    const actions = container!.querySelector('.share-linkcard__actions')
    expect(actions).not.toBeNull()
    const buttons = Array.from(actions!.querySelectorAll('button')).map((b) => b.textContent)
    expect(buttons).toContain('Copy')
    expect(buttons).toContain('Open')
    expect(buttons).toContain('QR')
  })

  it('renders the join code (CodeDisplay) and the scope description inside .share-linkcard__desc, positioned after the link', () => {
    ;({ container, root } = renderCard())
    expect(container!.textContent).toContain(CODE)
    const desc = container!.querySelector('.share-linkcard__desc')
    expect(desc).not.toBeNull()
    expect(desc!.textContent).toBe(DESCRIPTION)

    // Desc beneath: .share-linkcard__desc must follow .share-linkcard__top
    // and .share-linkcard__join in DOM order.
    const card = container!.querySelector('.share-linkcard')!
    const children = Array.from(card.children)
    const topIdx = children.findIndex((c) => c.classList.contains('share-linkcard__top'))
    const joinIdx = children.findIndex((c) => c.classList.contains('share-linkcard__join'))
    const descIdx = children.findIndex((c) => c.classList.contains('share-linkcard__desc'))
    expect(topIdx).toBeGreaterThanOrEqual(0)
    expect(joinIdx).toBeGreaterThan(topIdx)
    expect(descIdx).toBeGreaterThan(joinIdx)
  })
})

describe('ShareLinkCard — action wiring', () => {
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

  it('clicking Copy calls ClipboardSetText with the full URL', async () => {
    ;({ container, root } = renderCard())
    const copyBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent === 'Copy'
    )!
    await flushSync(() => copyBtn.click())
    expect(mockedClipboardSetText).toHaveBeenCalledWith(FULL_URL)
  })

  it('clicking Open calls BrowserOpenURL with the URL', () => {
    ;({ container, root } = renderCard())
    const openBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent === 'Open'
    )!
    flushSync(() => openBtn.click())
    expect(mockedBrowserOpenURL).toHaveBeenCalledWith(FULL_URL)
  })

  it('clicking QR calls GetCapabilityQRCode with a target containing /join?code= (join-URL, not the raw capability token)', async () => {
    ;({ container, root } = renderCard())
    const qrBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent === 'QR'
    )!
    await flushSync(() => qrBtn.click())
    expect(mockedGetCapabilityQRCode).toHaveBeenCalledTimes(1)
    const target = mockedGetCapabilityQRCode.mock.calls[0][0] as string
    expect(target).toContain('/join?code=')
    expect(target).toContain(CODE)
    // The raw capability token must never appear in the QR target.
    expect(target).not.toContain('SUPER_SECRET_TOKEN')
  })
})
