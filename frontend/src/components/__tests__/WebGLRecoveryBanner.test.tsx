/**
 * Phase 93 WGL-02/WGL-03 — WebGLRecoveryBanner render tests.
 *
 * Verbatim copy from UI-SPEC §"Copywriting Contract":
 *   reason='context-loss'      → "Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact."
 *                                 Auto-dismiss after 8000ms.
 *   reason='software-rasterized' → "Hardware acceleration is unavailable on this device. Your terminal is using the standard renderer for the best experience."
 *                                 Persistent (no auto-dismiss).
 */
import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { WebGLRecoveryBanner } from '../WebGLRecoveryBanner'

function render(reason: 'context-loss' | 'software-rasterized', onDismiss: () => void) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(WebGLRecoveryBanner, { reason, onDismiss }))
  })
  return { container, root }
}

describe('WebGLRecoveryBanner', () => {
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
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it("reason='context-loss' renders verbatim recovery copy including 'Scrollback is intact.'", () => {
    ;({ container, root } = render('context-loss', vi.fn()))
    expect(container.textContent).toMatch(/Hardware-accelerated rendering recovered/)
    expect(container.textContent).toMatch(/Scrollback is intact\./)
  })

  it("reason='software-rasterized' renders verbatim preemption copy", () => {
    ;({ container, root } = render('software-rasterized', vi.fn()))
    expect(container.textContent).toMatch(/Hardware acceleration is unavailable/)
    expect(container.textContent).toMatch(/for the best experience/)
  })

  it('has role="status" and aria-live="polite" (accessibility contract)', () => {
    ;({ container, root } = render('context-loss', vi.fn()))
    const statusEl = container.querySelector('[role="status"]')
    expect(statusEl).not.toBeNull()
    expect(statusEl?.getAttribute('aria-live')).toBe('polite')
  })

  it('dismiss button has aria-label="Dismiss notification" and fires onDismiss when clicked', () => {
    const onDismiss = vi.fn()
    ;({ container, root } = render('context-loss', onDismiss))
    const btn = container.querySelector('[aria-label="Dismiss notification"]') as HTMLButtonElement | null
    expect(btn).not.toBeNull()
    flushSync(() => { btn!.click() })
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  describe('auto-dismiss timing', () => {
    beforeEach(() => {
      vi.useFakeTimers()
    })

    it("reason='context-loss' fires onDismiss after 8000ms (auto-dismiss)", () => {
      const onDismiss = vi.fn()
      ;({ container, root } = render('context-loss', onDismiss))
      expect(onDismiss).not.toHaveBeenCalled()
      vi.advanceTimersByTime(8000)
      expect(onDismiss).toHaveBeenCalledTimes(1)
    })

    it("reason='software-rasterized' does NOT auto-dismiss (persistent)", () => {
      const onDismiss = vi.fn()
      ;({ container, root } = render('software-rasterized', onDismiss))
      vi.advanceTimersByTime(60000)
      expect(onDismiss).not.toHaveBeenCalled()
    })
  })
})
