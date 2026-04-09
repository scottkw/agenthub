import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { LocalNetworkBanner } from '../LocalNetworkBanner'

function renderBanner(props: { visible: boolean; onOpenURL: (url: string) => void }) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(LocalNetworkBanner, props))
  })
  return { container, root }
}

describe('LocalNetworkBanner', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root?.unmount()
    container?.remove()
  })

  it('renders banner content when visible', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL }))
    expect(container.textContent).toContain('Local network mode active')
    expect(container.textContent).toContain('Install Tailscale')
    const statusEl = container.querySelector('[role="status"]')
    expect(statusEl).not.toBeNull()
  })

  it('returns null when not visible', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: false, onOpenURL }))
    expect(container.firstChild).toBeNull()
  })

  it('calls onOpenURL with tailscale download link when CTA clicked', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL }))
    const buttons = container.querySelectorAll('button')
    const ctaBtn = Array.from(buttons).find((b) => b.textContent?.includes('Install Tailscale'))
    expect(ctaBtn).not.toBeUndefined()
    flushSync(() => {
      ctaBtn!.click()
    })
    expect(onOpenURL).toHaveBeenCalledWith('https://tailscale.com/download')
  })

  it('banner has role="status" for accessibility', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL }))
    const statusEl = container.querySelector('[role="status"]')
    expect(statusEl).not.toBeNull()
  })

  it('banner contains warning icon', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL }))
    expect(container.textContent).toContain('\u26a0')
  })
})
