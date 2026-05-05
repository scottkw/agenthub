/**
 * Phase 94 SRC-02 — FindBar match count rendering.
 *
 * Three render scenarios per UI-SPEC §"Match Count" + Copywriting Contract:
 *   1. query=""             → count element has --hidden modifier (visibility:hidden)
 *   2. query="foo", count=12, idx=2 → text "3 of 12", no --no-results
 *   3. query="xyz", count=0, idx=-1 → text "0 of 0", --no-results modifier
 *
 * jsdom does not compute CSS, so the no-results color (#f7768e) is asserted
 * via class membership only. Theme-pixel verification lives in 94-VERIFICATION.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { FindBar } from '../FindBar'

function render(props: Partial<React.ComponentProps<typeof FindBar>>) {
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

describe('FindBar — match count (SRC-02)', () => {
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

  it('empty query renders count element with --hidden modifier (visibility preserved layout)', () => {
    ;({ container, root } = render({ query: '', matchCount: 0, currentMatchIndex: -1 }))
    const count = container.querySelector('.find-bar__count') as HTMLElement | null
    expect(count).not.toBeNull()
    expect(count!.classList.contains('find-bar__count--hidden')).toBe(true)
    expect(count!.classList.contains('find-bar__count--no-results')).toBe(false)
  })

  it('non-empty query with matches renders "{i+1} of {n}" and no modifier classes', () => {
    ;({ container, root } = render({ query: 'foo', matchCount: 12, currentMatchIndex: 2 }))
    const count = container.querySelector('.find-bar__count') as HTMLElement | null
    expect(count).not.toBeNull()
    expect(count!.textContent).toBe('3 of 12')
    expect(count!.classList.contains('find-bar__count--hidden')).toBe(false)
    expect(count!.classList.contains('find-bar__count--no-results')).toBe(false)
  })

  it('non-empty query with zero matches renders "0 of 0" with --no-results modifier', () => {
    ;({ container, root } = render({ query: 'xyz', matchCount: 0, currentMatchIndex: -1 }))
    const count = container.querySelector('.find-bar__count') as HTMLElement | null
    expect(count).not.toBeNull()
    expect(count!.textContent).toBe('0 of 0')
    expect(count!.classList.contains('find-bar__count--no-results')).toBe(true)
    expect(count!.classList.contains('find-bar__count--hidden')).toBe(false)
  })

  it('count element has aria-live="polite" and aria-atomic="true" for screen reader updates', () => {
    ;({ container, root } = render({ query: 'foo', matchCount: 1, currentMatchIndex: 0 }))
    const count = container.querySelector('.find-bar__count') as HTMLElement | null
    expect(count).not.toBeNull()
    expect(count!.getAttribute('aria-live')).toBe('polite')
    expect(count!.getAttribute('aria-atomic')).toBe('true')
  })
})
