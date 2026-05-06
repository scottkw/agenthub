/**
 * Phase 94 WR-01 / SC-4 / SRC-04 — TerminalPanel exit-animation wiring.
 *
 * Asserts the JS wiring (NOT the visual feel — that's a manual UAT in
 * 94-VERIFICATION.md). On close (Esc / close button), TerminalPanel must:
 *   1. Add the .find-bar--exiting modifier to FindBar BEFORE unmount.
 *   2. Delay the actual unmount by 200ms via setTimeout (matches the
 *      longer of the two CSS exit durations — transform 200ms / opacity 150ms).
 *   3. Cancel the pending unmount timer if Cmd-F re-opens during the exit
 *      window (no zombie timer / no flicker).
 *   4. Synchronously cancel the debounce timer + clear decorations
 *      (Pitfall #10 — preserve the cancel-on-close contract).
 *
 * The runtime path (mounting xterm, firing key events) requires a real
 * <canvas> + WebGL context which jsdom does not provide; this file uses
 * the same source-inspection style as TerminalPanel.search.test.tsx
 * (the existing 81-test sweep). FindBar is exercised at runtime to verify
 * the `exiting` prop wiring (which is feasible in jsdom).
 *
 * UI-SPEC §"Animation" line 200 + §"Closing the Find Bar" lines 304-311.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import src from '../TerminalPanel.tsx?raw'
import { FindBar } from '../FindBar/FindBar'

describe('Phase 94 WR-01: TerminalPanel exit-animation wiring (source-inspection)', () => {
  it('declares findBarExiting state for the exit-animation flag', () => {
    expect(src).toMatch(/const\s+\[findBarExiting,\s*setFindBarExiting\]\s*=\s*useState\(false\)/)
  })

  it('declares findBarExitTimerRef for the delayed unmount timer', () => {
    expect(src).toContain('findBarExitTimerRef')
    // ≥4 references — declaration, set in close, clear in cancel-on-reopen,
    // clear in mount cleanup, clear+null inside the setTimeout body.
    const matches = src.match(/findBarExitTimerRef/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(4)
  })

  it('handleSearchClose sets findBarExiting=true BEFORE scheduling the unmount', () => {
    // Locate handleSearchClose body (callback declared with useCallback).
    const closeMatch = src.match(/handleSearchClose\s*=\s*useCallback\(\([^)]*\)\s*=>\s*\{([\s\S]*?)\},\s*\[\]\)/)
    expect(closeMatch).not.toBeNull()
    const body = closeMatch![1]
    expect(body).toContain('setFindBarExiting(true)')
    expect(body).toMatch(/setTimeout\([\s\S]+?,\s*200\s*\)/)
    // setFindBarOpen(false) must live INSIDE the setTimeout callback,
    // not at the top of the close handler — i.e. it must come AFTER
    // setFindBarExiting(true) in source order.
    const exitingIdx = body.indexOf('setFindBarExiting(true)')
    const openFalseIdx = body.indexOf('setFindBarOpen(false)')
    expect(exitingIdx).toBeGreaterThanOrEqual(0)
    expect(openFalseIdx).toBeGreaterThan(exitingIdx)
  })

  it('handleSearchClose synchronously clears decorations + debounce timer (Pitfall #10)', () => {
    const closeMatch = src.match(/handleSearchClose\s*=\s*useCallback\(\([^)]*\)\s*=>\s*\{([\s\S]*?)\},\s*\[\]\)/)
    expect(closeMatch).not.toBeNull()
    const body = closeMatch![1]
    expect(body).toContain('clearDecorations()')
    expect(body).toContain('clearTimeout(debounceTimerRef.current)')
    // These must run BEFORE the setTimeout schedule (synchronous cleanup).
    const clearDecIdx = body.indexOf('clearDecorations()')
    const setTimeoutIdx = body.search(/setTimeout\([\s\S]+?,\s*200\s*\)/)
    expect(clearDecIdx).toBeGreaterThanOrEqual(0)
    expect(setTimeoutIdx).toBeGreaterThan(clearDecIdx)
  })

  it('Cmd-F re-open path cancels the pending exit timer (no zombie state)', () => {
    // The handler must clear findBarExitTimerRef.current before flipping
    // findBarOpen back to true.
    expect(src).toMatch(/findBarExitTimerRef\.current[\s\S]+?clearTimeout\(findBarExitTimerRef\.current\)/)
    // setFindBarExiting(false) must be called when re-opening to drop the
    // .find-bar--exiting modifier in case the user closed mid-press.
    expect(src).toContain('setFindBarExiting(false)')
  })

  it('mount-effect cleanup clears findBarExitTimerRef (no leaked timer on unmount)', () => {
    // Look for clearTimeout(findBarExitTimerRef.current) reachable from the
    // mount-effect cleanup. Since cleanup is the only place that calls
    // term.dispose(), assert the timer-clear appears in the same file BEFORE
    // term.dispose() in source order.
    const termDisposeIdx = src.indexOf('term.dispose()')
    expect(termDisposeIdx).toBeGreaterThan(0)
    const before = src.slice(0, termDisposeIdx)
    expect(before).toContain('findBarExitTimerRef.current')
    expect(before).toMatch(/clearTimeout\(findBarExitTimerRef\.current\)/)
  })

  it('FindBar renders while findBarOpen OR findBarExiting is true (DOM-present during exit)', () => {
    // Render guard must be `(findBarOpen || findBarExiting) && pluginConfig?.search`.
    expect(src).toMatch(/\(findBarOpen\s*\|\|\s*findBarExiting\)\s*&&\s*pluginConfig\?\.search/)
  })

  it('threads exiting={findBarExiting} prop into FindBar', () => {
    expect(src).toMatch(/exiting=\{findBarExiting\}/)
  })
})

// ---- Runtime verification of the FindBar `exiting` prop wiring ----
// jsdom can mount FindBar standalone; this verifies the className composition
// for the exit-animation modifier without needing xterm.
describe('Phase 94 WR-01: FindBar exiting prop runtime wiring', () => {
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
    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)
    flushSync(() => {
      root!.render(React.createElement(FindBar, full))
    })
    return container.querySelector('[role="search"]') as HTMLElement
  }

  it('exiting=true applies .find-bar--exiting modifier', () => {
    const bar = render({ exiting: true })
    expect(bar.className).toContain('find-bar--exiting')
  })

  it('exiting=true suppresses .find-bar--entering (exiting wins over mount-state)', () => {
    const bar = render({ exiting: true })
    expect(bar.className).not.toContain('find-bar--entering')
  })

  it('exiting=false (default) does NOT apply the exiting modifier', () => {
    const bar = render({ exiting: false })
    expect(bar.className).not.toContain('find-bar--exiting')
  })
})
