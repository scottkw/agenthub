import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { LocalNetworkBanner } from '../LocalNetworkBanner'

function renderBanner(props: { visible: boolean; tailscaleConnected: boolean; onOpenURL: (url: string) => void }) {
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

  it('renders banner content when visible and tailscale not connected', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: false, onOpenURL }))
    expect(container.textContent).toContain('Local network mode active')
    expect(container.textContent).toContain('Install Tailscale')
    const statusEl = container.querySelector('[role="status"]')
    expect(statusEl).not.toBeNull()
  })

  it('returns null when not visible', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: false, tailscaleConnected: false, onOpenURL }))
    expect(container.firstChild).toBeNull()
  })

  it('calls onOpenURL with tailscale download link when CTA clicked', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: false, onOpenURL }))
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
    ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: false, onOpenURL }))
    const statusEl = container.querySelector('[role="status"]')
    expect(statusEl).not.toBeNull()
  })

  it('banner contains warning icon', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: false, onOpenURL }))
    expect(container.textContent).toContain('\u26a0')
  })

  it('shows upgrading message when tailscaleConnected is true', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: true, onOpenURL }))
    expect(container.textContent).toContain('upgrading to Tailscale')
    expect(container.textContent).toContain('Tailscale detected')
  })

  it('does not show CTA button when tailscaleConnected is true', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: true, onOpenURL }))
    const buttons = container.querySelectorAll('button')
    expect(buttons.length).toBe(0)
  })

  it('shows Install Tailscale button when tailscaleConnected is false', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: false, onOpenURL }))
    const buttons = container.querySelectorAll('button')
    const ctaBtn = Array.from(buttons).find((b) => b.textContent?.includes('Install Tailscale'))
    expect(ctaBtn).not.toBeUndefined()
  })
})
