// Phase 147-01: HelpSearch test stubs (RED until Plan 03).
//
// Asserts search input label, clear button, empty-state, and mark highlight
// behaviors. All DOM-render assertions WILL fail until HelpSearch.tsx is
// implemented in Plan 03 — this is the intended RED state for Wave 0.

import { describe, it, expect, vi, afterEach } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'

// ============================================================
// Render helper
// ============================================================

type SearchEntry = { sectionId: string; sectionLabel: string; text: string }

// Dynamically import HelpSearch so compilation errors in the (not-yet-existing)
// component surface at test time as "module not found" rather than crashing the
// entire suite. The tests below are expected to fail at render time.
let HelpSearch: React.ComponentType<{
  query: string
  results: ReadonlyArray<SearchEntry>
  onQueryChange: (raw: string) => void
  onJumpToSection: (id: string) => void
}>

try {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  HelpSearch = require('../HelpSearch').HelpSearch
} catch {
  // Component doesn't exist yet (RED state) — tests will fail at render
  HelpSearch = () => null as unknown as React.ReactElement
}

function renderHelpSearch(
  overrides: Partial<{
    query: string
    results: ReadonlyArray<SearchEntry>
    onQueryChange: (raw: string) => void
    onJumpToSection: (id: string) => void
  }> = {},
) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const defaultProps = {
    query: '',
    results: [] as SearchEntry[],
    onQueryChange: vi.fn(),
    onJumpToSection: vi.fn(),
  }
  act(() => {
    root.render(<HelpSearch {...defaultProps} {...overrides} />)
  })
  return { container, root }
}

let container: HTMLElement
let root: ReturnType<typeof createRoot>

afterEach(() => {
  try {
    root.unmount()
  } catch {
    // Ignore — RED state may not have rendered anything
  }
  container?.remove()
})

// ============================================================
// Search input label (D-12 visible label requirement)
// ============================================================

describe('HelpSearch: visible label "Search help…" (Phase 147)', () => {
  it('renders a visible label element with text "Search help…"', () => {
    ;({ container, root } = renderHelpSearch())
    const label = container.querySelector('label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Search help…')
  })

  it('renders a search input associated with the label', () => {
    ;({ container, root } = renderHelpSearch())
    const input = container.querySelector('input[type="search"], input.help-search__input')
    expect(input).not.toBeNull()
  })
})

// ============================================================
// Clear button
// ============================================================

describe('HelpSearch: clear button (Phase 147)', () => {
  it('renders a clear button with aria-label="Clear search"', () => {
    ;({ container, root } = renderHelpSearch({ query: 'hello' }))
    const clearBtn = container.querySelector('button[aria-label="Clear search"]')
    expect(clearBtn).not.toBeNull()
  })
})

// ============================================================
// Empty state
// ============================================================

describe('HelpSearch: empty state (Phase 147)', () => {
  it('shows .help-search__empty with No results for "{query}" when query non-empty AND results empty', () => {
    ;({ container, root } = renderHelpSearch({ query: 'zzzzunknown', results: [] }))
    const emptyEl = container.querySelector('.help-search__empty')
    expect(emptyEl).not.toBeNull()
    expect(emptyEl!.textContent).toContain('zzzzunknown')
  })

  it('does NOT show .help-search__empty when query is empty', () => {
    ;({ container, root } = renderHelpSearch({ query: '', results: [] }))
    const emptyEl = container.querySelector('.help-search__empty')
    expect(emptyEl).toBeNull()
  })

  it('does NOT show .help-search__empty when results are non-empty', () => {
    const results: SearchEntry[] = [
      { sectionId: 'help-faq', sectionLabel: 'FAQ', text: 'DevTools zzzzunknown disabled' },
    ]
    ;({ container, root } = renderHelpSearch({ query: 'zzzzunknown', results }))
    const emptyEl = container.querySelector('.help-search__empty')
    expect(emptyEl).toBeNull()
  })
})

// ============================================================
// Highlight <mark> element
// ============================================================

describe('HelpSearch: <mark class="help-search__mark"> highlight (Phase 147)', () => {
  it('wraps matched substring in <mark class="help-search__mark">', () => {
    const results: SearchEntry[] = [
      {
        sectionId: 'help-getting-started',
        sectionLabel: 'Getting Started',
        text: 'Click the New Session button to start',
      },
    ]
    ;({ container, root } = renderHelpSearch({ query: 'New Session', results }))
    const mark = container.querySelector('mark.help-search__mark')
    expect(mark).not.toBeNull()
    expect(mark!.textContent).toContain('New Session')
  })
})
