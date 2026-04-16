import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { LocalNetworkBanner } from '../LocalNetworkBanner'

interface LocalNetworkBannerProps {
  visible: boolean
  tailscaleConnected: boolean
  tailscaleInstalled: boolean
  tailscaleBinaryFound: boolean
  tailscaleDaemonUp: boolean
  platformHint: string
  onOpenURL: (url: string) => void
  onDismiss?: () => void
  className?: string
}

function renderBanner(props: Partial<LocalNetworkBannerProps> & { visible: boolean; onOpenURL: (url: string) => void }) {
  const fullProps: LocalNetworkBannerProps = {
    tailscaleConnected: false,
    tailscaleInstalled: false,
    tailscaleBinaryFound: false,
    tailscaleDaemonUp: false,
    platformHint: '',
    ...props,
  }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(LocalNetworkBanner, fullProps))
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

  it('renders Install Tailscale when not installed and not connected', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL }))
    expect(container.textContent).toContain('Local network mode active')
    expect(container.textContent).toContain('Install Tailscale')
    const ctaBtn = container.querySelector('button')
    expect(ctaBtn?.textContent).toContain('Install Tailscale')
  })

  it('returns null when not visible', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: false, onOpenURL }))
    expect(container.firstChild).toBeNull()
  })

  it('calls onOpenURL with tailscale download link when CTA clicked', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL }))
    const ctaBtn = container.querySelector('button')
    expect(ctaBtn).not.toBeNull()
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

  it('shows upgrading message when tailscaleConnected is true', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: true, tailscaleInstalled: true, tailscaleBinaryFound: true, tailscaleDaemonUp: true, onOpenURL }))
    expect(container.textContent).toContain('upgrading to Tailscale')
    expect(container.textContent).toContain('Tailscale detected')
  })

  it('does not show CTA button when tailscaleConnected is true', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: true, tailscaleInstalled: true, tailscaleBinaryFound: true, tailscaleDaemonUp: true, onOpenURL }))
    const buttons = container.querySelectorAll('button')
    expect(buttons.length).toBe(0)
  })

  it('shows "not connected" message when daemon is up but not connected', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, tailscaleDaemonUp: true, tailscaleBinaryFound: true, platformHint: 'darwin', onOpenURL }))
    expect(container.textContent).toContain('not connected')
  })

  it('shows Install Tailscale button only when not installed', () => {
    const onOpenURL = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL }))
    const buttons = container.querySelectorAll('button')
    const ctaBtn = Array.from(buttons).find((b) => b.textContent?.includes('Install Tailscale'))
    expect(ctaBtn).not.toBeUndefined()
  })

  it('shows daemon-stopped message when binary found but daemon down', () => {
    ;({ container, root } = renderBanner({
      visible: true,
      tailscaleBinaryFound: true,
      tailscaleDaemonUp: false,
      platformHint: 'darwin',
      onOpenURL: vi.fn(),
    }))
    expect(container.textContent).toContain('daemon not running')
    expect(container.textContent).toContain('Open Tailscale from Applications')
    const buttons = container.querySelectorAll('button')
    expect(buttons.length).toBe(0) // D-06: no action buttons
  })

  it('shows Linux daemon instruction when platformHint is linux', () => {
    ;({ container, root } = renderBanner({
      visible: true,
      tailscaleBinaryFound: true,
      tailscaleDaemonUp: false,
      platformHint: 'linux',
      onOpenURL: vi.fn(),
    }))
    expect(container.textContent).toContain('sudo systemctl start tailscaled')
  })

  it('shows Windows daemon instruction when platformHint is windows', () => {
    ;({ container, root } = renderBanner({
      visible: true,
      tailscaleBinaryFound: true,
      tailscaleDaemonUp: false,
      platformHint: 'windows',
      onOpenURL: vi.fn(),
    }))
    expect(container.textContent).toContain('Start menu or system tray')
  })

  it('renders dismiss button when onDismiss is provided', () => {
    const onDismiss = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL: vi.fn(), onDismiss }))
    const dismissBtn = container.querySelector('.local-network-banner__dismiss')
    expect(dismissBtn).not.toBeNull()
  })

  it('does not render dismiss button when onDismiss is not provided', () => {
    ;({ container, root } = renderBanner({ visible: true, onOpenURL: vi.fn() }))
    const dismissBtn = container.querySelector('.local-network-banner__dismiss')
    expect(dismissBtn).toBeNull()
  })

  it('calls onDismiss when dismiss button clicked', () => {
    const onDismiss = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL: vi.fn(), onDismiss }))
    const dismissBtn = container.querySelector('.local-network-banner__dismiss') as HTMLButtonElement
    flushSync(() => { dismissBtn.click() })
    expect(onDismiss).toHaveBeenCalledOnce()
  })

  it('applies className when provided', () => {
    ;({ container, root } = renderBanner({ visible: true, onOpenURL: vi.fn(), className: 'banner-exit' }))
    const banner = container.querySelector('.local-network-banner')
    expect(banner?.classList.contains('banner-exit')).toBe(true)
  })

  it('dismiss button has correct aria-label', () => {
    const onDismiss = vi.fn()
    ;({ container, root } = renderBanner({ visible: true, onOpenURL: vi.fn(), onDismiss }))
    const dismissBtn = container.querySelector('.local-network-banner__dismiss')
    expect(dismissBtn?.getAttribute('aria-label')).toBe('Dismiss local network notification')
  })
})
