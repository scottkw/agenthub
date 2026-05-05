/**
 * Phase 94 SRC-01 — FindBar dismiss tests.
 *
 * UI-SPEC §"Closing the Find Bar": Esc and the close-button click both
 * invoke onClose. Focus-restore-to-xterm lives in TerminalPanel's
 * handleSearchClose (Task 2) — exercised by source-inspection there.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { FindBar } from '../FindBar'

function renderFindBar(onClose: () => void) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(FindBar, {
        query: '',
        onQueryChange: vi.fn(),
        matchCount: 0,
        currentMatchIndex: -1,
        searchOptions: { regex: false, caseSensitive: false, wholeWord: false },
        onSearchOptionsChange: vi.fn(),
        onNext: vi.fn(),
        onPrev: vi.fn(),
        onClose,
      }),
    )
  })
  return { container, root }
}

describe('FindBar — dismiss (SRC-01)', () => {
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

  it('Escape keydown on the find-bar container invokes onClose', () => {
    const onClose = vi.fn()
    ;({ container, root } = renderFindBar(onClose))
    const bar = container.querySelector('.find-bar') as HTMLElement | null
    expect(bar).not.toBeNull()
    flushSync(() => {
      bar!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('clicking the close button invokes onClose', () => {
    const onClose = vi.fn()
    ;({ container, root } = renderFindBar(onClose))
    const closeBtn = container.querySelector('[aria-label="Close find bar"]') as HTMLButtonElement | null
    expect(closeBtn).not.toBeNull()
    flushSync(() => {
      closeBtn!.click()
    })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('Escape and close-button combine to two onClose invocations across both paths', () => {
    const onClose = vi.fn()
    ;({ container, root } = renderFindBar(onClose))
    const bar = container.querySelector('.find-bar') as HTMLElement
    const closeBtn = container.querySelector('[aria-label="Close find bar"]') as HTMLButtonElement
    flushSync(() => {
      bar.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    flushSync(() => {
      closeBtn.click()
    })
    expect(onClose).toHaveBeenCalledTimes(2)
  })
})
