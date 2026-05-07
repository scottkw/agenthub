/**
 * Phase 94 WR-01 / SC-4 / SRC-04 — FindBar slide-in animation wiring.
 *
 * Asserts the JS wiring (NOT the visual feel — that's a manual UAT in
 * 94-VERIFICATION.md). On mount, the .find-bar container must carry the
 * `find-bar--entering` modifier class so CSS `.find-bar--entering` (which
 * sets translateY(-100%) + opacity 0) is the initial state. On the next
 * animation frame, the modifier must drop so the base rule kicks in
 * (`transform: translateY(0); opacity: 1; transition: 200ms ease`) and the
 * browser observes the class flip and runs the slide-in transition.
 *
 * UI-SPEC §"Animation" lines 197-202 mandates 200ms slide-in with
 * translateY(-100%) → 0 + opacity 0 → 1.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { act } from 'react'
import { FindBar } from '../FindBar'

function render(props: Partial<React.ComponentProps<typeof FindBar>> = {}) {
  const full: React.ComponentProps<typeof FindBar> = {
    query: '',
    onQueryChange: vi.fn(),
    matchCount: 0,
    currentMatchIndex: -1,
    searchOptions: { regex: false, caseSensitive: false, wholeWord: false },
    onSearchOptionsChange: vi.fn(),
    onNext: vi.fn(),
    onPrev: vi.fn(),
    onClose: vi.fn(),
    ...props,
  }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(FindBar, full))
  })
  return { container, root }
}

describe('FindBar — slide-in animation wiring (WR-01 / SC-4)', () => {
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
  })

  it('applies find-bar--entering on initial render', () => {
    ;({ container, root } = render())
    const bar = container.querySelector('[role="search"]') as HTMLElement | null
    expect(bar).not.toBeNull()
    expect(bar!.className).toContain('find-bar--entering')
    expect(bar!.classList.contains('find-bar')).toBe(true)
  })

  it('removes find-bar--entering after one animation frame', async () => {
    ;({ container, root } = render())
    const bar = container.querySelector('[role="search"]') as HTMLElement | null
    expect(bar).not.toBeNull()
    expect(bar!.className).toContain('find-bar--entering')

    // Flush a single requestAnimationFrame tick. jsdom polyfills RAF as
    // setTimeout(fn, 0); wrapping in act() ensures the resulting state
    // update flushes before we re-query the DOM.
    await act(async () => {
      await new Promise<void>((resolve) => {
        requestAnimationFrame(() => resolve())
      })
    })

    expect(bar!.className).not.toContain('find-bar--entering')
    expect(bar!.classList.contains('find-bar')).toBe(true)
  })

  it('cleanup cancels the pending RAF on unmount (no late state update)', async () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    ;({ container, root } = render())
    // Unmount IMMEDIATELY before the RAF fires.
    flushSync(() => root!.unmount())
    root = undefined
    // Now flush the RAF — if cancelAnimationFrame failed, React would
    // attempt setState on an unmounted component and console.error.
    await act(async () => {
      await new Promise<void>((resolve) => {
        requestAnimationFrame(() => resolve())
      })
    })
    expect(errorSpy).not.toHaveBeenCalled()
    errorSpy.mockRestore()
  })
})
