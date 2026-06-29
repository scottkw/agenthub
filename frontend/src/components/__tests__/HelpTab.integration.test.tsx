// Phase 147 (WR-01 / CR-01): HelpTab render-based integration test.
//
// The sibling HelpTab.test.tsx is source-gate-only (readFileSync + substring
// match) and never mounts the component — which is exactly why CR-01 (section
// anchors never exist, so nav / scroll-spy / search-jump are dead no-ops)
// shipped green. This test ACTUALLY renders <HelpTab/> and exercises the three
// getElementById-dependent code paths against the real DOM:
//
//   1. Sections render with the expected ids (#help-getting-started, #help-faq).
//   2. Clicking a HelpSectionNav item scrolls the matching section into view
//      (scrollIntoView is jsdom-unimplemented — we stub it and assert it was
//      called on the correct <section> element) and marks it active.
//   3. Typing a query (through the 200ms debounce) yields results whose
//      "jump to section" resolves a REAL section element (scrollIntoView fires
//      on a non-null target).
//
// This test FAILS against the pre-fix code (HelpTab rendered one concatenated
// <HelpContent> with no #help-* section wrappers, so getElementById → null and
// scrollIntoView is never called) and PASSES after the CR-01 fix.

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'

// Mock the Wails runtime so BrowserOpenURL-using children mount cleanly in jsdom.
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}))

import { HelpTab } from '../HelpTab'

let container: HTMLElement
let root: ReturnType<typeof createRoot>
let scrollIntoViewMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  vi.useFakeTimers()
  // jsdom does not implement scrollIntoView; stub it on the prototype so every
  // element (including our <section> anchors) gets the spy.
  scrollIntoViewMock = vi.fn()
  Element.prototype.scrollIntoView = scrollIntoViewMock as unknown as typeof Element.prototype.scrollIntoView

  container = document.createElement('div')
  document.body.appendChild(container)
  root = createRoot(container)
  act(() => {
    root.render(<HelpTab />)
  })
})

afterEach(() => {
  act(() => {
    root.unmount()
  })
  container.remove()
  vi.useRealTimers()
  vi.restoreAllMocks()
})

// ============================================================
// 1. Section anchors exist in the DOM (CR-01 core regression)
// ============================================================

describe('HelpTab integration: section anchors render with expected ids (Phase 147)', () => {
  it('renders a #help-getting-started section element', () => {
    const el = document.getElementById('help-getting-started')
    expect(el).not.toBeNull()
    expect(el!.tagName.toLowerCase()).toBe('section')
  })

  it('renders a #help-faq section element', () => {
    const el = document.getElementById('help-faq')
    expect(el).not.toBeNull()
    expect(el!.tagName.toLowerCase()).toBe('section')
  })

  it('section wrappers carry the help-content__section class (live CSS selector)', () => {
    const sections = container.querySelectorAll('section.help-content__section')
    expect(sections.length).toBe(3)
  })

  it('renders a #help-chat section element', () => {
    const el = document.getElementById('help-chat')
    expect(el).not.toBeNull()
    expect(el!.tagName.toLowerCase()).toBe('section')
    expect(el!.textContent).toContain('Chat')
  })

  it('rendered section actually contains its markdown heading text', () => {
    const el = document.getElementById('help-getting-started')
    expect(el!.textContent).toContain('Getting Started')
    const faq = document.getElementById('help-faq')
    expect(faq!.textContent).toContain('Frequently Asked Questions')
  })
})

// ============================================================
// 2. Nav click scrolls the right section + marks it active
// ============================================================

describe('HelpTab integration: nav click scrolls section into view (Phase 147)', () => {
  it('clicking the FAQ nav button calls scrollIntoView on the #help-faq section', () => {
    const faqSection = document.getElementById('help-faq')!
    const faqScroll = vi.fn()
    // Per-element spy so we can assert the click targeted THIS section.
    faqSection.scrollIntoView = faqScroll as unknown as typeof faqSection.scrollIntoView

    const navButtons = Array.from(
      container.querySelectorAll('.help-tab__nav button'),
    ) as HTMLButtonElement[]
    const faqNav = navButtons.find((b) =>
      b.textContent?.includes('Frequently Asked Questions'),
    )
    expect(faqNav).not.toBeUndefined()

    act(() => {
      faqNav!.click()
    })

    expect(faqScroll).toHaveBeenCalledTimes(1)
  })

  it('clicking the FAQ nav button marks that nav item active (aria-current)', () => {
    const navButtons = Array.from(
      container.querySelectorAll('.help-tab__nav button'),
    ) as HTMLButtonElement[]
    const faqNav = navButtons.find((b) =>
      b.textContent?.includes('Frequently Asked Questions'),
    )!
    act(() => {
      faqNav.click()
    })
    // After click, HelpTab.setActiveSection('help-faq') re-renders the nav with
    // aria-current on the FAQ button.
    const active = container.querySelector('.help-tab__nav button[aria-current="true"]')
    expect(active).not.toBeNull()
    expect(active!.textContent).toContain('Frequently Asked Questions')
  })
})

// ============================================================
// 3. Search jump resolves a real section element
// ============================================================

describe('HelpTab integration: search result jump resolves a real section (Phase 147)', () => {
  it('typing a query then clicking a result scrolls a non-null section into view', () => {
    const input = container.querySelector('#help-search-input') as HTMLInputElement
    expect(input).not.toBeNull()

    // Type a query that matches FAQ content ("Tailscale" appears in faq.md).
    act(() => {
      const setter = Object.getOwnPropertyDescriptor(
        window.HTMLInputElement.prototype,
        'value',
      )!.set!
      setter.call(input, 'Tailscale')
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })

    // Advance past the 200ms debounce so results compute.
    act(() => {
      vi.advanceTimersByTime(250)
    })

    const resultItems = Array.from(
      container.querySelectorAll('.help-search__result'),
    ) as HTMLElement[]
    expect(resultItems.length).toBeGreaterThan(0)

    // Clicking a result calls handleJumpToSection → getElementById(sectionId)
    // → scrollIntoView. Pre-fix that element was null and scrollIntoView never
    // fired; post-fix it resolves a real <section>.
    scrollIntoViewMock.mockClear()
    act(() => {
      resultItems[0].click()
    })
    expect(scrollIntoViewMock).toHaveBeenCalled()
  })

  it('the search index actually produces results for known content', () => {
    const input = container.querySelector('#help-search-input') as HTMLInputElement
    act(() => {
      const setter = Object.getOwnPropertyDescriptor(
        window.HTMLInputElement.prototype,
        'value',
      )!.set!
      setter.call(input, 'session')
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    act(() => {
      vi.advanceTimersByTime(250)
    })
    const resultItems = container.querySelectorAll('.help-search__result')
    expect(resultItems.length).toBeGreaterThan(0)
  })
})
