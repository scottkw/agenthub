/**
 * Phase 94 Plan 94-04 — FindBar cancel-on-close (SRC-03 cancellation half).
 *
 * Verifies the cancellation contract from two angles:
 *
 *   1. FindBar component contract — clicking the close button OR pressing
 *      Esc inside the find bar invokes onClose synchronously. (Overlaps
 *      with FindBar.dismiss.test.tsx by design: the overlap pins the
 *      interaction surface; the unique value here is the Esc-while-non-input-
 *      element-focused case from RESEARCH Pitfall #3 — Esc must work even
 *      when the input is NOT the focused element.)
 *
 *   2. TerminalPanel handleSearchClose source-inspection — the close
 *      handler MUST call BOTH `clearTimeout(debounceTimerRef.current)` AND
 *      `searchAddonRef.current?.clearDecorations()`, AND reset
 *      `debounceTimerRef.current = null` after clearing. Verified via
 *      `?raw` source-inspection because jsdom cannot exercise the runtime
 *      cancel path (requires a real xterm + SearchAddon). This is the
 *      Pitfall #10 mitigation regression gate (RESEARCH §"Find Bar Unmount
 *      Doesn't Clear Pending Debounce").
 *
 * Runs in jsdom (Vitest) — does NOT need chromedp. The full chromedp
 * "rapidly type, then close before debounce fires" simulation is out of
 * scope for this plan; Plan 94-03 already implemented the runtime path.
 *
 * See: 94-VALIDATION.md row 04-perf wave 3; 94-RESEARCH.md Pitfall #10.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { FindBar } from '../FindBar'
import terminalPanelSrc from '../../TerminalPanel.tsx?raw'

function renderFindBar(props: Partial<Parameters<typeof FindBar>[0]> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const fullProps = {
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
  flushSync(() => {
    root.render(React.createElement(FindBar, fullProps))
  })
  return { container, root, props: fullProps }
}

describe('FindBar — cancellation (SRC-03)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

  beforeEach(() => {
    vi.useFakeTimers()
  })

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
  })

  it('clicking the close button invokes onClose synchronously', () => {
    const onClose = vi.fn()
    ;({ container, root } = renderFindBar({ onClose, query: 'foo', matchCount: 5, currentMatchIndex: 2 }))
    const closeBtn = container.querySelector<HTMLButtonElement>('.find-bar__close')
    expect(closeBtn).not.toBeNull()
    flushSync(() => {
      closeBtn!.click()
    })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('Esc keydown on the find-bar container invokes onClose even when a non-input element is focused (Pitfall #3)', () => {
    // RESEARCH Pitfall #3: "Esc Inside Input Doesn't Bubble Up to onKeyDown"
    // — the keydown handler MUST live on the find-bar container, not just on
    // the input, so Esc works while a toggle button (e.g. case-sensitive)
    // has focus instead of the search input.
    const onClose = vi.fn()
    ;({ container, root } = renderFindBar({ onClose }))
    const caseToggle = container.querySelector<HTMLButtonElement>('.find-bar__toggle--case')
    expect(caseToggle).not.toBeNull()
    caseToggle!.focus()
    flushSync(() => {
      caseToggle!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('TerminalPanel handleSearchClose calls clearTimeout(debounceTimerRef.current) (Pitfall #10)', () => {
    // Source-inspection of TerminalPanel.tsx (Plan 94-03 implementation).
    // jsdom cannot exercise this path because the SearchAddon construction
    // requires a real xterm canvas; the test inspects the source string
    // for the required cancel symbols instead.
    expect(terminalPanelSrc).toMatch(/clearTimeout\(\s*debounceTimerRef\.current\s*\)/)
  })

  it('TerminalPanel handleSearchClose calls searchAddonRef.current?.clearDecorations() (Pitfall #10)', () => {
    expect(terminalPanelSrc).toMatch(/searchAddonRef\.current\?\.clearDecorations\(\)/)
  })

  it('TerminalPanel handleSearchClose resets debounceTimerRef.current = null after clearing the timeout', () => {
    // Belt-and-suspenders invariant: not only is the timeout cleared, the
    // ref is nulled so the next mount starts from a known-clean state.
    expect(terminalPanelSrc).toMatch(/debounceTimerRef\.current\s*=\s*null/)
  })
})
