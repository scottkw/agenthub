/**
 * Phase 99 PUI-02 — PluginToggleBanner render tests.
 *
 * Verbatim copy locked by 99-RESEARCH.md "Claude's Discretion":
 *   kind='unicode11' → "Open a new terminal session to apply the Unicode 11 change."
 *                       Auto-dismiss after 6000ms.
 *   kind='image'     → "Open a new terminal session to apply the Inline Images change."
 *                       Auto-dismiss after 6000ms.
 *
 * Both kinds auto-dismiss (unconditional) — differs from WebGLRecoveryBanner
 * where 'software-rasterized' is persistent.
 */
import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { PluginToggleBanner } from '../PluginToggleBanner'

function render(kind: 'unicode11' | 'image', onDismiss: () => void) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(PluginToggleBanner, { kind, onDismiss }))
  })
  return { container, root }
}

describe('PluginToggleBanner', () => {
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

  it("kind='unicode11' renders verbatim copy 'Open a new terminal session to apply the Unicode 11 change.'", () => {
    ;({ container, root } = render('unicode11', vi.fn()))
    expect(container.textContent).toContain('Open a new terminal session to apply the Unicode 11 change.')
  })

  it("kind='image' renders verbatim copy 'Open a new terminal session to apply the Inline Images change.'", () => {
    ;({ container, root } = render('image', vi.fn()))
    expect(container.textContent).toContain('Open a new terminal session to apply the Inline Images change.')
  })

  it('has role="status" and aria-live="polite" (a11y contract from analog)', () => {
    ;({ container, root } = render('unicode11', vi.fn()))
    const statusEl = container.querySelector('[role="status"]')
    expect(statusEl).not.toBeNull()
    expect(statusEl?.getAttribute('aria-live')).toBe('polite')
  })

  it('dismiss button has aria-label="Dismiss notification" and fires onDismiss when clicked', () => {
    const onDismiss = vi.fn()
    ;({ container, root } = render('unicode11', onDismiss))
    const btn = container.querySelector('[aria-label="Dismiss notification"]') as HTMLButtonElement | null
    expect(btn).not.toBeNull()
    flushSync(() => { btn!.click() })
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('outer element has className containing "webgl-recovery-banner" (BEM reuse for visual continuity)', () => {
    ;({ container, root } = render('unicode11', vi.fn()))
    const el = container.firstElementChild as HTMLElement | null
    expect(el?.className).toContain('webgl-recovery-banner')
  })

  describe('auto-dismiss timing', () => {
    beforeEach(() => {
      vi.useFakeTimers()
    })

    it("kind='unicode11' fires onDismiss after 6000ms (auto-dismiss)", () => {
      const onDismiss = vi.fn()
      ;({ container, root } = render('unicode11', onDismiss))
      expect(onDismiss).not.toHaveBeenCalled()
      vi.advanceTimersByTime(6000)
      expect(onDismiss).toHaveBeenCalledTimes(1)
    })

    it("kind='image' fires onDismiss after 6000ms (auto-dismiss — both kinds auto-dismiss, no conditional skip)", () => {
      const onDismiss = vi.fn()
      ;({ container, root } = render('image', onDismiss))
      expect(onDismiss).not.toHaveBeenCalled()
      vi.advanceTimersByTime(6000)
      expect(onDismiss).toHaveBeenCalledTimes(1)
    })

    it('timer is cleared on unmount (safe to call onDismiss after unmount but timer must not fire)', () => {
      const onDismiss = vi.fn()
      ;({ container, root } = render('unicode11', onDismiss))
      // Unmount before timer fires
      flushSync(() => root!.unmount())
      root = undefined
      vi.advanceTimersByTime(6000)
      // onDismiss should NOT have been called by the timer after unmount
      expect(onDismiss).not.toHaveBeenCalled()
    })
  })
})
