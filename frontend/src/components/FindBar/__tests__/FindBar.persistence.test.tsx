/**
 * Phase 94 SRC-02 — FindBar persistence contract.
 *
 * Verifies the controlled-component contract: clicking each toggle button
 * fires onSearchOptionsChange with the flipped value. The actual
 * SetPluginSettings round-trip (daemon SearchConfig persistence) is
 * exercised by the parent's source-inspection in TerminalPanel.search.test.tsx
 * and by the daemon-side test in App.plugin-event.test.tsx (Plan 94-02).
 *
 * UI-SPEC §"Toggle Persistence"; Pitfall #2 — parent owns the persistence
 * call, FindBar is purely controlled.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { FindBar, type FindBarSearchOptions } from '../FindBar'

function render(initial: FindBarSearchOptions, onSearchOptionsChange: (o: FindBarSearchOptions) => void) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(FindBar, {
        query: 'foo',
        onQueryChange: vi.fn(),
        matchCount: 1,
        currentMatchIndex: 0,
        searchOptions: initial,
        onSearchOptionsChange,
        onNext: vi.fn(),
        onPrev: vi.fn(),
        onClose: vi.fn(),
      }),
    )
  })
  return { container, root }
}

describe('FindBar — persistence (SRC-02 controlled-component contract)', () => {
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

  it('clicking the Case-sensitive toggle fires onSearchOptionsChange with caseSensitive flipped', () => {
    const onChange = vi.fn()
    const initial: FindBarSearchOptions = { regex: false, caseSensitive: false, wholeWord: false }
    ;({ container, root } = render(initial, onChange))
    const btn = container.querySelector('[aria-label="Case sensitive"]') as HTMLButtonElement
    flushSync(() => btn.click())
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith({ regex: false, caseSensitive: true, wholeWord: false })
  })

  it('clicking the Regex toggle fires onSearchOptionsChange with regex flipped', () => {
    const onChange = vi.fn()
    const initial: FindBarSearchOptions = { regex: false, caseSensitive: true, wholeWord: false }
    ;({ container, root } = render(initial, onChange))
    const btn = container.querySelector('[aria-label="Regular expression"]') as HTMLButtonElement
    flushSync(() => btn.click())
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith({ regex: true, caseSensitive: true, wholeWord: false })
  })

  it('clicking the Whole-word toggle fires onSearchOptionsChange with wholeWord flipped', () => {
    const onChange = vi.fn()
    const initial: FindBarSearchOptions = { regex: true, caseSensitive: false, wholeWord: false }
    ;({ container, root } = render(initial, onChange))
    const btn = container.querySelector('[aria-label="Whole word"]') as HTMLButtonElement
    flushSync(() => btn.click())
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith({ regex: true, caseSensitive: false, wholeWord: true })
  })

  it('clicking an already-active toggle flips it OFF (true → false)', () => {
    const onChange = vi.fn()
    const initial: FindBarSearchOptions = { regex: true, caseSensitive: true, wholeWord: true }
    ;({ container, root } = render(initial, onChange))
    const btn = container.querySelector('[aria-label="Case sensitive"]') as HTMLButtonElement
    flushSync(() => btn.click())
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith({ regex: true, caseSensitive: false, wholeWord: true })
  })

  it('active toggle has aria-pressed="true" and the --active modifier class', () => {
    const onChange = vi.fn()
    const initial: FindBarSearchOptions = { regex: true, caseSensitive: false, wholeWord: false }
    ;({ container, root } = render(initial, onChange))
    const regex = container.querySelector('[aria-label="Regular expression"]') as HTMLButtonElement
    expect(regex.getAttribute('aria-pressed')).toBe('true')
    expect(regex.classList.contains('find-bar__toggle--active')).toBe(true)
    const caseT = container.querySelector('[aria-label="Case sensitive"]') as HTMLButtonElement
    expect(caseT.getAttribute('aria-pressed')).toBe('false')
    expect(caseT.classList.contains('find-bar__toggle--active')).toBe(false)
  })
})
