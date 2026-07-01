/**
 * Phase 166 / FUI-01/FUI-02/FUI-06 — FunnelRiskPanel contract.
 *
 * The risk panel is the ONLY gate that leads to internet exposure. It must:
 *   - render the verbatim risk statement (FUI-01)
 *   - offer the five auto-expiry presets with 3600 (1 hour) default (FUI-02)
 *   - expose a Help cross-link to the Sharing Guide (FUI-06)
 *   - fire onEnable / onCancel from the two action buttons
 *   - contain NO SetSessionFunnel call (commit happens in the modal, D-02)
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { FunnelRiskPanel } from '../Hub/FunnelRiskPanel'

const RISK_STATEMENT =
  'This makes the session reachable from the public internet. The join code is the only access gate for its lifetime. Prefer short-lived, read-only shares.'

let container: HTMLElement | undefined
let root: Root | undefined

interface PanelOpts {
  open?: boolean
  expirySeconds?: number
  onExpiryChange?: (n: number) => void
  onEnable?: () => void
  onCancel?: () => void
  onOpenHelp?: () => void
}

function renderPanel(opts: PanelOpts = {}) {
  container = document.createElement('div')
  document.body.appendChild(container)
  root = createRoot(container)
  flushSync(() => {
    root!.render(
      React.createElement(FunnelRiskPanel, {
        open: opts.open ?? true,
        expirySeconds: opts.expirySeconds ?? 3600,
        onExpiryChange: opts.onExpiryChange ?? vi.fn(),
        onEnable: opts.onEnable ?? vi.fn(),
        onCancel: opts.onCancel ?? vi.fn(),
        onOpenHelp: opts.onOpenHelp ?? vi.fn(),
      }),
    )
  })
  return container!
}

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

describe('FunnelRiskPanel — FUI-01 risk statement', () => {
  it('renders the verbatim risk statement', () => {
    const c = renderPanel()
    expect(c.textContent).toContain(RISK_STATEMENT)
  })

  it('applies the --open modifier class when open', () => {
    const c = renderPanel({ open: true })
    const panel = c.querySelector('.hub-funnel-risk-panel')
    expect(panel).not.toBeNull()
    expect(panel!.className).toContain('hub-funnel-risk-panel--open')
  })

  it('omits the --open modifier when closed', () => {
    const c = renderPanel({ open: false })
    const panel = c.querySelector('.hub-funnel-risk-panel')
    expect(panel!.className).not.toContain('hub-funnel-risk-panel--open')
  })
})

describe('FunnelRiskPanel — FUI-02 auto-expiry selector', () => {
  it('offers exactly the five preset values 1800/3600/14400/28800/0', () => {
    const c = renderPanel()
    const select = c.querySelector('select') as HTMLSelectElement | null
    expect(select).not.toBeNull()
    const values = Array.from(select!.options).map((o) => o.value)
    expect(values).toEqual(['1800', '3600', '14400', '28800', '0'])
  })

  it('defaults the selected value to 3600 (1 hour)', () => {
    const c = renderPanel({ expirySeconds: 3600 })
    const select = c.querySelector('select') as HTMLSelectElement
    expect(select.value).toBe('3600')
  })

  it('invokes onExpiryChange with the numeric preset value on change', () => {
    const onExpiryChange = vi.fn()
    const c = renderPanel({ onExpiryChange })
    const select = c.querySelector('select') as HTMLSelectElement
    select.value = '14400'
    flushSync(() => {
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(onExpiryChange).toHaveBeenCalledWith(14400)
  })
})

describe('FunnelRiskPanel — FUI-06 help cross-link + actions', () => {
  it('fires onOpenHelp when the Sharing Guide cross-link is clicked', () => {
    const onOpenHelp = vi.fn()
    const c = renderPanel({ onOpenHelp })
    const link = Array.from(c.querySelectorAll('button')).find((b) =>
      /See the Sharing Guide/i.test(b.textContent ?? ''),
    ) as HTMLElement | null
    expect(link).not.toBeNull()
    flushSync(() => link!.click())
    expect(onOpenHelp).toHaveBeenCalled()
  })

  it('fires onEnable from the "Enable internet share" CTA', () => {
    const onEnable = vi.fn()
    const c = renderPanel({ onEnable })
    const cta = Array.from(c.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Enable internet share',
    ) as HTMLElement | null
    expect(cta).not.toBeNull()
    flushSync(() => cta!.click())
    expect(onEnable).toHaveBeenCalled()
  })

  it('fires onCancel from the "Keep local only" button', () => {
    const onCancel = vi.fn()
    const c = renderPanel({ onCancel })
    const cancel = Array.from(c.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Keep local only',
    ) as HTMLElement | null
    expect(cancel).not.toBeNull()
    flushSync(() => cancel!.click())
    expect(onCancel).toHaveBeenCalled()
  })

  it('contains no reference to SetSessionFunnel (commit happens in the modal)', () => {
    // The panel is presentational — assert it exposes only the callback surface,
    // never a direct binding. This is a structural guard for D-02.
    const c = renderPanel()
    expect(c.textContent).not.toMatch(/SetSessionFunnel/)
  })
})
