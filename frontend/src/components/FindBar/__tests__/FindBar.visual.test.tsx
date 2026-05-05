/**
 * Phase 94 SRC-04 — FindBar visual / aria contract.
 *
 * Verifies the verbatim aria-labels, structural classes, and the
 * "no decorations:" invariant via source-inspection (component must NOT
 * pass a `decorations` field to SearchAddon — SRC-04 theme.selectionBackground
 * compliance across all 138 themes).
 *
 * UI-SPEC §"Find Bar CSS Classes" + §"Copywriting Contract".
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { FindBar } from '../FindBar'
// ?raw imports the file as a string for source-inspection (Vite/Vitest feature).
import findBarSrc from '../FindBar.tsx?raw'

function render() {
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
        onClose: vi.fn(),
      }),
    )
  })
  return { container, root }
}

describe('FindBar — visual contract (SRC-04)', () => {
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

  it('outer container has class "find-bar", role="search", aria-label="Find in terminal"', () => {
    ;({ container, root } = render())
    const bar = container.querySelector('.find-bar')
    expect(bar).not.toBeNull()
    expect(bar!.getAttribute('role')).toBe('search')
    expect(bar!.getAttribute('aria-label')).toBe('Find in terminal')
  })

  it('input has placeholder "Find…", aria-label "Search", autocomplete "off", spellcheck "false"', () => {
    ;({ container, root } = render())
    const input = container.querySelector('.find-bar__input') as HTMLInputElement | null
    expect(input).not.toBeNull()
    expect(input!.placeholder).toBe('Find…')
    expect(input!.getAttribute('aria-label')).toBe('Search')
    expect(input!.getAttribute('autocomplete')).toBe('off')
    // React renders spellCheck={false} as the HTML attribute spellcheck="false";
    // assert via the attribute since jsdom does not always reflect the property.
    expect(input!.getAttribute('spellcheck')).toBe('false')
  })

  it('renders three toggle buttons with verbatim aria-labels', () => {
    ;({ container, root } = render())
    expect(container.querySelector('[aria-label="Case sensitive"]')).not.toBeNull()
    expect(container.querySelector('[aria-label="Regular expression"]')).not.toBeNull()
    expect(container.querySelector('[aria-label="Whole word"]')).not.toBeNull()
  })

  it('renders two nav buttons with verbatim aria-labels', () => {
    ;({ container, root } = render())
    expect(container.querySelector('[aria-label="Previous match"]')).not.toBeNull()
    expect(container.querySelector('[aria-label="Next match"]')).not.toBeNull()
  })

  it('close button has aria-label "Close find bar" and title "Close (Esc)"', () => {
    ;({ container, root } = render())
    const close = container.querySelector('[aria-label="Close find bar"]') as HTMLButtonElement | null
    expect(close).not.toBeNull()
    expect(close!.getAttribute('title')).toBe('Close (Esc)')
  })

  it('toggle buttons reflect aria-pressed from searchOptions prop', () => {
    const root2Container = document.createElement('div')
    document.body.appendChild(root2Container)
    const root2 = createRoot(root2Container)
    flushSync(() => {
      root2.render(
        React.createElement(FindBar, {
          query: '',
          onQueryChange: vi.fn(),
          matchCount: 0,
          currentMatchIndex: -1,
          searchOptions: { regex: true, caseSensitive: false, wholeWord: true },
          onSearchOptionsChange: vi.fn(),
          onNext: vi.fn(),
          onPrev: vi.fn(),
          onClose: vi.fn(),
        }),
      )
    })
    const regex = root2Container.querySelector('[aria-label="Regular expression"]')!
    const caseT = root2Container.querySelector('[aria-label="Case sensitive"]')!
    const word = root2Container.querySelector('[aria-label="Whole word"]')!
    expect(regex.getAttribute('aria-pressed')).toBe('true')
    expect(caseT.getAttribute('aria-pressed')).toBe('false')
    expect(word.getAttribute('aria-pressed')).toBe('true')
    flushSync(() => root2.unmount())
    root2Container.remove()
  })

  it('source does NOT pass `decorations:` to any SearchAddon API (SRC-04 theme.selectionBackground invariant)', () => {
    expect(findBarSrc).not.toMatch(/decorations:/)
  })

  it('source does NOT instantiate SearchAddon directly (controlled-component contract — parent owns the addon)', () => {
    expect(findBarSrc).not.toContain('new SearchAddon')
    expect(findBarSrc).not.toMatch(/from\s+['"]@xterm\/addon-search['"]/)
  })
})
