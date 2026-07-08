/**
 * Phase 173 / SM-04 + SM-06 — TailnetTab tests.
 *
 * Redistributed from SessionSharePanel.test.tsx's Read-Only/Full-Access-link
 * and scope-clarity assertions. Highest-value new test (SM-04/T-173-01): the
 * public-write markers (`hub-funnel-write-gate` class + hold-to-confirm
 * button) are ABSENT from this tab's rendered output.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { TailnetTab } from '../SessionShare/TailnetTab'

vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  GetCapabilityQRCode: vi.fn().mockResolvedValue(''),
}))

interface RenderOpts {
  browseEnabled?: boolean
  readURL?: string
  writeURL?: string
  readCode?: string
  writeCode?: string
}

function renderTab(opts: RenderOpts = {}) {
  const c = document.createElement('div')
  document.body.appendChild(c)
  const r = createRoot(c)
  flushSync(() => {
    r.render(
      React.createElement(TailnetTab, {
        readURL: opts.readURL ?? 'https://example.com/r',
        writeURL: opts.writeURL ?? 'https://example.com/w',
        readCode: opts.readCode ?? 'read-code',
        writeCode: opts.writeCode ?? 'write-code',
        browseEnabled: opts.browseEnabled ?? false,
      }),
    )
  })
  return { container: c, root: r }
}

describe('TailnetTab', () => {
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

  it('renders exactly two ShareLinkCards: Read-Only Link and Full Access Link', () => {
    ;({ container, root } = renderTab())
    const cards = container!.querySelectorAll('.share-linkcard')
    expect(cards.length).toBe(2)
    const titles = Array.from(container!.querySelectorAll('.share-linkcard__title')).map(
      (el) => el.textContent,
    )
    expect(titles).toEqual(['Read-Only Link', 'Full Access Link'])
  })

  it('renders both URLs and both join codes', () => {
    ;({ container, root } = renderTab({
      readURL: 'https://example.com/r?cap=READ_TOKEN',
      writeURL: 'https://example.com/w?cap=WRITE_TOKEN',
      readCode: 'read-code-x',
      writeCode: 'write-code-y',
    }))
    expect(container!.innerHTML).toContain('READ_TOKEN')
    expect(container!.innerHTML).toContain('WRITE_TOKEN')
    expect(container!.textContent).toContain('read-code-x')
    expect(container!.textContent).toContain('write-code-y')
  })

  it('browseEnabled=false shows "watch only" / no-file-access scope text for both cards', () => {
    ;({ container, root } = renderTab({ browseEnabled: false }))
    const text = container!.textContent ?? ''
    expect(text).toContain('cannot send input or browse files')
    expect(text).toContain('Full control of the live session (send input) plus file browsing. Use the toggle above to also allow file editing.')
  })

  it('browseEnabled=true reflects file browsing in both cards\' scope text', () => {
    ;({ container, root } = renderTab({ browseEnabled: true }))
    const text = container!.textContent ?? ''
    expect(text).toContain('Watch the live session and browse files read-only — cannot send input.')
    expect(text).toContain('Full control of the live session (send input) plus file browsing and editing.')
  })

  // SM-04 / T-173-01 negative wall-off assertion — the write-gate markup and
  // hold-to-confirm button must NEVER appear in the Tailnet tab.
  it('SM-04: the public-write gate markup and hold-to-confirm button are ABSENT', () => {
    ;({ container, root } = renderTab())
    expect(container!.querySelector('.hub-funnel-write-gate')).toBeNull()
    expect(container!.querySelector('.hub-funnel-write-gate__hold-btn')).toBeNull()
    expect(container!.textContent).not.toContain('PUBLIC WRITE ACCESS — COMMAND EXECUTION')
    expect(container!.textContent).not.toContain('Enable public write access')
  })
})
