// Phase 147-01: HelpSectionNav test stubs (RED until Plan 03).
//
// Asserts section-nav renders one button per section, aria-current on the active
// section, and onSectionChange fires on click. WILL fail until HelpSectionNav.tsx
// is implemented in Plan 03 — this is the intended RED state for Wave 0.

import { describe, it, expect, vi, afterEach } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'

// ============================================================
// Render helper
// ============================================================

// Phase 147-02: Component now exists — use static import (GREEN state).
// The try/catch require() pattern used in RED state is incompatible with
// Vitest's CJS resolver (which does not try .tsx extensions).
import { HelpSectionNav } from '../HelpSectionNav'

function renderHelpSectionNav(
  overrides: Partial<{
    activeSection: string
    onSectionChange: (id: string) => void
    contentPaneRef: React.RefObject<HTMLDivElement | null>
  }> = {},
) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const defaultContentPane = { current: document.createElement('div') } as React.RefObject<HTMLDivElement | null>
  const defaultProps = {
    activeSection: 'help-getting-started',
    onSectionChange: vi.fn(),
    contentPaneRef: defaultContentPane,
  }
  act(() => {
    root.render(<HelpSectionNav {...defaultProps} {...overrides} />)
  })
  return { container, root, ...defaultProps, ...(overrides as Partial<typeof defaultProps>) }
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
// Section buttons render
// ============================================================

describe('HelpSectionNav: renders one button per section (Phase 147)', () => {
  it('renders at least two section nav buttons (Getting Started + FAQ)', () => {
    ;({ container, root } = renderHelpSectionNav())
    // The nav renders one <button> per section (Getting Started, FAQ at minimum)
    const buttons = container.querySelectorAll('button')
    expect(buttons.length).toBeGreaterThanOrEqual(2)
  })

  it('renders a button for the Getting Started section', () => {
    ;({ container, root } = renderHelpSectionNav())
    const buttons = Array.from(container.querySelectorAll('button'))
    const gettingStartedBtn = buttons.find((b) =>
      b.textContent?.includes('Getting Started'),
    )
    expect(gettingStartedBtn).not.toBeUndefined()
  })

  it('renders a button for the FAQ section', () => {
    ;({ container, root } = renderHelpSectionNav())
    const buttons = Array.from(container.querySelectorAll('button'))
    const faqBtn = buttons.find((b) =>
      b.textContent?.includes('Frequently Asked Questions') ||
      b.textContent?.includes('FAQ'),
    )
    expect(faqBtn).not.toBeUndefined()
  })
})

// ============================================================
// aria-current reflects activeSection prop
// ============================================================

describe('HelpSectionNav: aria-current on active section (Phase 147)', () => {
  it('active section button carries aria-current="true"', () => {
    ;({ container, root } = renderHelpSectionNav({ activeSection: 'help-getting-started' }))
    // The button for help-getting-started should have aria-current="true"
    const buttons = Array.from(container.querySelectorAll('button'))
    const activeBtn = buttons.find((b) =>
      b.getAttribute('aria-current') === 'true',
    )
    expect(activeBtn).not.toBeUndefined()
    expect(activeBtn!.textContent).toContain('Getting Started')
  })

  it('active section button has class help-nav__link--active', () => {
    ;({ container, root } = renderHelpSectionNav({ activeSection: 'help-getting-started' }))
    const activeBtn = container.querySelector('.help-nav__link--active')
    expect(activeBtn).not.toBeNull()
  })

  it('inactive section buttons do NOT carry aria-current="true"', () => {
    ;({ container, root } = renderHelpSectionNav({ activeSection: 'help-getting-started' }))
    const buttons = Array.from(container.querySelectorAll('button'))
    const inactiveBtns = buttons.filter(
      (b) => b.getAttribute('aria-current') !== 'true',
    )
    // There should be at least one inactive button (FAQ)
    expect(inactiveBtns.length).toBeGreaterThanOrEqual(1)
  })
})

// ============================================================
// onSectionChange fires on click
// ============================================================

describe('HelpSectionNav: onSectionChange fires on button click (Phase 147)', () => {
  it('clicking a section button calls onSectionChange with that section id', () => {
    const onSectionChange = vi.fn()
    ;({ container, root } = renderHelpSectionNav({
      activeSection: 'help-getting-started',
      onSectionChange,
    }))
    const buttons = Array.from(container.querySelectorAll('button'))
    const faqBtn = buttons.find((b) =>
      b.textContent?.includes('Frequently Asked Questions') ||
      b.textContent?.includes('FAQ'),
    ) as HTMLButtonElement
    expect(faqBtn).not.toBeUndefined()
    act(() => { faqBtn.click() })
    expect(onSectionChange).toHaveBeenCalledWith(expect.stringContaining('faq'))
  })
})
