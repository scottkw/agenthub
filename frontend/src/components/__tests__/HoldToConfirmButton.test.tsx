/**
 * Phase 173 / SM-07 — HoldToConfirmButton prefers-reduced-motion fallback.
 *
 * D-09 preserves the 3s hold-to-confirm SAFETY GATE CONTRACT as-is on the
 * non-reduced-motion path (see SessionSharePanel.test.tsx R1 assertions,
 * unchanged). This file covers the NEW, ADDITIVE reduced-motion branch:
 * a plain single-click confirm with no timed fill — still a deliberate
 * confirm action, just without the timed animation (SM-07/D-07).
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { HoldToConfirmButton } from '../SessionShare/shared'

function mockMatchMedia(reduced: boolean): () => void {
  const original = window.matchMedia
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: reduced && query.includes('prefers-reduced-motion'),
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })) as unknown as typeof window.matchMedia
  return () => {
    window.matchMedia = original
  }
}

function render(props: { disabled: boolean; onConfirm: () => void }): { container: HTMLElement; root: Root } {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(HoldToConfirmButton, props))
  })
  return { container, root }
}

describe('HoldToConfirmButton — prefers-reduced-motion fallback (SM-07/D-07)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined
  let restoreMatchMedia: (() => void) | undefined

  afterEach(() => {
    if (root) {
      flushSync(() => root!.unmount())
      root = undefined
    }
    if (container) {
      container.remove()
      container = undefined
    }
    restoreMatchMedia?.()
    restoreMatchMedia = undefined
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('reduced-motion: renders a plain confirm button with no hold-fill element, and a single click fires onConfirm exactly once', () => {
    restoreMatchMedia = mockMatchMedia(true)
    const onConfirm = vi.fn()
    ;({ container, root } = render({ disabled: false, onConfirm }))

    expect(container!.querySelector('.hub-funnel-write-gate__hold-fill')).toBeNull()
    expect(container!.textContent).toContain('Confirm')

    const btn = container!.querySelector('button') as HTMLButtonElement
    act(() => {
      btn.click()
    })
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('reduced-motion: disabled prop prevents confirm on click', () => {
    restoreMatchMedia = mockMatchMedia(true)
    const onConfirm = vi.fn()
    ;({ container, root } = render({ disabled: true, onConfirm }))

    const btn = container!.querySelector('button') as HTMLButtonElement
    expect(btn.disabled).toBe(true)
    act(() => {
      btn.click()
    })
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('non-reduced-motion: keeps the timed 3s hold — hold-fill element present, and a bare click does NOT fire onConfirm', () => {
    restoreMatchMedia = mockMatchMedia(false)
    vi.useFakeTimers()
    const onConfirm = vi.fn()
    ;({ container, root } = render({ disabled: false, onConfirm }))

    const fill = container!.querySelector('.hub-funnel-write-gate__hold-fill')
    expect(fill).not.toBeNull()
    expect(container!.textContent).toContain('Hold 3s to confirm')

    const btn = container!.querySelector('button') as HTMLButtonElement
    act(() => {
      btn.click()
    })
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('non-reduced-motion: completing the full 3s hold still fires onConfirm exactly once (byte-behavior unchanged)', () => {
    restoreMatchMedia = mockMatchMedia(false)
    vi.useFakeTimers()
    const onConfirm = vi.fn()
    ;({ container, root } = render({ disabled: false, onConfirm }))

    const btn = container!.querySelector('button') as HTMLButtonElement
    act(() => {
      btn.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerId: 1 }))
    })
    act(() => {
      vi.advanceTimersByTime(3000)
    })
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('non-reduced-motion: disabled prop prevents the hold from starting', () => {
    restoreMatchMedia = mockMatchMedia(false)
    const onConfirm = vi.fn()
    ;({ container, root } = render({ disabled: true, onConfirm }))

    const btn = container!.querySelector('button') as HTMLButtonElement
    expect(btn.disabled).toBe(true)
    expect(btn.getAttribute('aria-disabled')).toBe('true')
  })
})
