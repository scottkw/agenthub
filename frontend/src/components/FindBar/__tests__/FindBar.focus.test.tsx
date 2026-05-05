/**
 * Phase 94 SRC-01 — FindBar focus-conditioning + Esc-closes-bar tests.
 *
 * The full TerminalPanel keydown integration (Cmd-F gating) is verified by
 * source-inspection in TerminalPanel.hot-swap.test.tsx (Plan 94-03 Task 2).
 * This file pins the helper-side invariants AND the contract that Esc fired
 * inside the find bar invokes onClose (UI-SPEC §"Closing the Find Bar").
 *
 * RESEARCH §"Pattern 2 — Focus-conditioned Cmd-F"; UI-SPEC §"Opening the Find Bar".
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { isXtermFocused } from '../../../lib/isXtermFocused'
import { FindBar } from '../FindBar'

function renderFindBar(propsOverride: Partial<React.ComponentProps<typeof FindBar>> = {}) {
  const props: React.ComponentProps<typeof FindBar> = {
    query: '',
    onQueryChange: vi.fn(),
    matchCount: 0,
    currentMatchIndex: -1,
    searchOptions: { regex: false, caseSensitive: false, wholeWord: false },
    onSearchOptionsChange: vi.fn(),
    onNext: vi.fn(),
    onPrev: vi.fn(),
    onClose: vi.fn(),
    ...propsOverride,
  }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(FindBar, props))
  })
  return { container, root, props }
}

describe('FindBar — focus conditioning (SRC-01)', () => {
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
    document.body.querySelectorAll('div, input').forEach((n) => {
      if (!n.closest('[data-vitest-keep]')) n.remove()
    })
  })

  it('isXtermFocused returns false when termContainer is null', () => {
    expect(isXtermFocused(null)).toBe(false)
  })

  it('isXtermFocused returns true when activeElement is inside termContainer', () => {
    const term = document.createElement('div')
    document.body.appendChild(term)
    const child = document.createElement('input')
    child.tabIndex = 0
    term.appendChild(child)
    child.focus()
    expect(isXtermFocused(term)).toBe(true)
  })

  it('isXtermFocused returns false when activeElement is in a modal sibling (Pitfall #1)', () => {
    const term = document.createElement('div')
    document.body.appendChild(term)
    const modal = document.createElement('div')
    document.body.appendChild(modal)
    const modalInput = document.createElement('input')
    modalInput.tabIndex = 0
    modal.appendChild(modalInput)
    modalInput.focus()
    expect(isXtermFocused(term)).toBe(false)
  })

  it('Escape on the find-bar container calls onClose', () => {
    const onClose = vi.fn()
    ;({ container, root } = renderFindBar({ onClose }))
    const bar = container.querySelector('.find-bar') as HTMLElement | null
    expect(bar).not.toBeNull()
    flushSync(() => {
      bar!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
