import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { UpdateBanner } from '../UpdateBanner'
import type { UpdateInfo } from '../UpdateBanner'

// Mock BrowserOpenURL from Wails runtime
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}))

import { BrowserOpenURL } from '../../wailsjs/wailsjs/runtime/runtime'

const mockUpdate: UpdateInfo = {
  currentVersion: '1.0.0',
  latestVersion: '2.0.0',
  releaseURL: 'https://github.com/agenthub-dev/agenthub/releases/tag/v2.0.0',
}

function renderUpdateBanner(update: UpdateInfo, onDismiss: () => void, className?: string) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(UpdateBanner, { update, onDismiss, className }))
  })
  return { container, root }
}

describe('UpdateBanner', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root?.unmount()
    container?.remove()
    vi.clearAllMocks()
  })

  it('renders update version information', () => {
    ;({ container, root } = renderUpdateBanner(mockUpdate, vi.fn()))
    expect(container.textContent).toContain('1.0.0')
    expect(container.textContent).toContain('2.0.0')
  })

  it('renders Download Update button', () => {
    ;({ container, root } = renderUpdateBanner(mockUpdate, vi.fn()))
    const downloadBtn = container.querySelector('.update-banner__btn--download')
    expect(downloadBtn).not.toBeNull()
    expect(downloadBtn?.textContent).toContain('Download Update')
  })

  it('renders Dismiss button with correct aria-label', () => {
    ;({ container, root } = renderUpdateBanner(mockUpdate, vi.fn()))
    const dismissBtn = container.querySelector('.update-banner__btn--dismiss')
    expect(dismissBtn).not.toBeNull()
    expect(dismissBtn?.getAttribute('aria-label')).toBe('Dismiss update notification')
  })

  it('calls onDismiss when Dismiss button clicked', () => {
    const onDismiss = vi.fn()
    ;({ container, root } = renderUpdateBanner(mockUpdate, onDismiss))
    const dismissBtn = container.querySelector('.update-banner__btn--dismiss') as HTMLButtonElement
    flushSync(() => { dismissBtn.click() })
    expect(onDismiss).toHaveBeenCalledOnce()
  })

  it('has role="alert" for accessibility', () => {
    ;({ container, root } = renderUpdateBanner(mockUpdate, vi.fn()))
    const alertEl = container.querySelector('[role="alert"]')
    expect(alertEl).not.toBeNull()
  })

  it('has aria-live="polite"', () => {
    ;({ container, root } = renderUpdateBanner(mockUpdate, vi.fn()))
    const alertEl = container.querySelector('[aria-live="polite"]')
    expect(alertEl).not.toBeNull()
  })

  it('applies className when provided (for banner-exit animation)', () => {
    ;({ container, root } = renderUpdateBanner(mockUpdate, vi.fn(), 'banner-exit'))
    const banner = container.querySelector('.update-banner')
    expect(banner?.classList.contains('banner-exit')).toBe(true)
  })

  it('calls BrowserOpenURL with releaseURL on download click', () => {
    ;({ container, root } = renderUpdateBanner(mockUpdate, vi.fn()))
    const downloadBtn = container.querySelector('.update-banner__btn--download') as HTMLButtonElement
    flushSync(() => { downloadBtn.click() })
    expect(BrowserOpenURL).toHaveBeenCalledWith(mockUpdate.releaseURL)
  })

  it('renders "Update available:" message text', () => {
    ;({ container, root } = renderUpdateBanner(mockUpdate, vi.fn()))
    expect(container.textContent).toContain('Update available:')
  })
})
