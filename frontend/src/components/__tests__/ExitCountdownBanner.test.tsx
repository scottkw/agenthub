import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { ExitCountdownBanner } from '../ExitCountdownBanner'

function renderBanner(countdown: number, onKeepOpen: () => void = vi.fn()) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(ExitCountdownBanner, { countdown, onKeepOpen }))
  })
  return { container, root }
}

describe('ExitCountdownBanner', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root?.unmount()
    container?.remove()
    vi.clearAllMocks()
  })

  it('renders banner with countdown message', () => {
    ;({ container, root } = renderBanner(4))
    expect(container.textContent).toContain('Agent exited cleanly')
    expect(container.textContent).toContain('4s')
  })

  it('renders Keep Open button', () => {
    ;({ container, root } = renderBanner(3))
    const btn = container.querySelector('.exit-countdown-banner__keep-open')
    expect(btn).not.toBeNull()
    expect(btn?.textContent).toBe('Keep Open')
  })

  it('calls onKeepOpen when button clicked', () => {
    const onKeepOpen = vi.fn()
    ;({ container, root } = renderBanner(5, onKeepOpen))
    const btn = container.querySelector('.exit-countdown-banner__keep-open') as HTMLButtonElement
    flushSync(() => { btn.click() })
    expect(onKeepOpen).toHaveBeenCalledOnce()
  })

  it('has role="alert" for accessibility', () => {
    ;({ container, root } = renderBanner(2))
    const banner = container.querySelector('.exit-countdown-banner')
    expect(banner?.getAttribute('role')).toBe('alert')
  })

  it('displays countdown in dedicated span', () => {
    ;({ container, root } = renderBanner(1))
    const countdownEl = container.querySelector('.exit-countdown-banner__countdown')
    expect(countdownEl?.textContent).toBe('1s')
  })
})
