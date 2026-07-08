/**
 * Phase 173 / SM-03/SM-07 — ShareSegmentedControl a11y + roving-tabindex +
 * arrow-nav contract.
 *
 * All assertions are attribute/text/role/class based — never computed color
 * (owner is colorblind; see project memory). This is the only place the
 * ARIA/roving/arrow-nav contract is proven before the shell (plan 06) wires
 * this component in.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import {
  ShareSegmentedControl,
  type ShareSegmentedControlTab,
} from '../Hub/ShareSegmentedControl'

let container: HTMLElement | undefined
let root: Root | undefined

const TABS_PRE_CONFIRM: ShareSegmentedControlTab[] = [
  { id: 'tailnet', main: 'Tailnet', sub: 'Private' },
  { id: 'internet-ro', main: 'Internet', sub: 'Read-only', disabled: true },
  {
    id: 'internet-fa',
    main: 'Internet',
    sub: 'Full access',
    disabled: true,
    danger: true,
  },
]

const TABS_POST_CONFIRM: ShareSegmentedControlTab[] = [
  { id: 'tailnet', main: 'Tailnet', sub: 'Private' },
  { id: 'internet-ro', main: 'Internet', sub: 'Read-only' },
  { id: 'internet-fa', main: 'Internet', sub: 'Full access', danger: true },
]

function renderControl(opts: {
  tabs: ShareSegmentedControlTab[]
  active: string
  onSelect?: (id: string) => void
}) {
  container = document.createElement('div')
  document.body.appendChild(container)
  root = createRoot(container)
  flushSync(() => {
    root!.render(
      React.createElement(ShareSegmentedControl, {
        tabs: opts.tabs,
        active: opts.active as never,
        onSelect: (opts.onSelect ?? vi.fn()) as never,
      }),
    )
  })
  return container!
}

afterEach(() => {
  if (root) {
    flushSync(() => root!.unmount())
    root = undefined
  }
  if (container) {
    container.remove()
    container = undefined
  }
  vi.clearAllMocks()
})

describe('ShareSegmentedControl — ARIA tablist contract (SM-03)', () => {
  it('renders role=tablist with exactly three role=tab children', () => {
    const c = renderControl({ tabs: TABS_POST_CONFIRM, active: 'tailnet' })
    const tablist = c.querySelector('[role="tablist"]')
    expect(tablist).not.toBeNull()
    const tabs = c.querySelectorAll('[role="tab"]')
    expect(tabs.length).toBe(3)
  })

  it('sets aria-selected true on the active tab and false on others', () => {
    const c = renderControl({ tabs: TABS_POST_CONFIRM, active: 'internet-ro' })
    const tabs = Array.from(c.querySelectorAll('[role="tab"]'))
    const active = tabs.find((t) => t.getAttribute('aria-selected') === 'true')
    expect(active?.textContent).toContain('Read-only')
    const others = tabs.filter((t) => t !== active)
    expect(others.every((t) => t.getAttribute('aria-selected') === 'false')).toBe(true)
  })
})

describe('ShareSegmentedControl — roving tabindex (SM-07)', () => {
  it('gives the active tab tabIndex=0 and all others tabIndex=-1', () => {
    const c = renderControl({ tabs: TABS_POST_CONFIRM, active: 'internet-fa' })
    const tabs = Array.from(c.querySelectorAll('[role="tab"]')) as HTMLButtonElement[]
    const active = tabs.find((t) => t.textContent?.includes('Full access'))
    expect(active?.tabIndex).toBe(0)
    const others = tabs.filter((t) => t !== active)
    expect(others.every((t) => t.tabIndex === -1)).toBe(true)
  })
})

describe('ShareSegmentedControl — disabled segments (SM-05/SM-07)', () => {
  it('marks disabled Internet tabs aria-disabled + disabled with N/A sub text', () => {
    const c = renderControl({ tabs: TABS_PRE_CONFIRM, active: 'tailnet' })
    const tabs = Array.from(c.querySelectorAll('[role="tab"]')) as HTMLButtonElement[]
    const disabledTabs = tabs.filter((t) => t.getAttribute('aria-disabled') === 'true')
    expect(disabledTabs.length).toBe(2)
    for (const t of disabledTabs) {
      expect(t.disabled).toBe(true)
      const sub = t.querySelector('.share-seg__sub')
      expect(sub?.textContent).toBe('N/A')
    }
  })

  it('does not call onSelect when a disabled tab is clicked', () => {
    const onSelect = vi.fn()
    const c = renderControl({ tabs: TABS_PRE_CONFIRM, active: 'tailnet', onSelect })
    const tabs = Array.from(c.querySelectorAll('[role="tab"]')) as HTMLButtonElement[]
    const disabledTab = tabs.find((t) => t.getAttribute('aria-disabled') === 'true')
    expect(disabledTab).toBeDefined()
    flushSync(() => disabledTab!.click())
    expect(onSelect).not.toHaveBeenCalled()
  })
})

describe('ShareSegmentedControl — arrow-key navigation (SM-07)', () => {
  it('ArrowRight from the active enabled tab selects the next enabled tab (wraps, skips disabled)', () => {
    const onSelect = vi.fn()
    // Pre-confirm: only 'tailnet' enabled. ArrowRight from tailnet must wrap
    // back to itself (the only enabled segment) — proves disabled segments
    // are skipped rather than selected.
    const c = renderControl({ tabs: TABS_PRE_CONFIRM, active: 'tailnet', onSelect })
    const tabs = Array.from(c.querySelectorAll('[role="tab"]')) as HTMLButtonElement[]
    const activeTab = tabs.find((t) => t.getAttribute('aria-selected') === 'true')!
    flushSync(() => {
      activeTab.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }),
      )
    })
    expect(onSelect).toHaveBeenCalledWith('tailnet')
  })

  it('ArrowRight moves through all three enabled tabs and wraps at the end', () => {
    const onSelect = vi.fn()
    const c = renderControl({ tabs: TABS_POST_CONFIRM, active: 'tailnet', onSelect })
    const tabs = Array.from(c.querySelectorAll('[role="tab"]')) as HTMLButtonElement[]
    const activeTab = tabs.find((t) => t.getAttribute('aria-selected') === 'true')!
    flushSync(() => {
      activeTab.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }),
      )
    })
    expect(onSelect).toHaveBeenCalledWith('internet-ro')
  })

  it('ArrowLeft from the first enabled tab wraps to the last enabled tab', () => {
    const onSelect = vi.fn()
    const c = renderControl({ tabs: TABS_POST_CONFIRM, active: 'tailnet', onSelect })
    const tabs = Array.from(c.querySelectorAll('[role="tab"]')) as HTMLButtonElement[]
    const activeTab = tabs.find((t) => t.getAttribute('aria-selected') === 'true')!
    flushSync(() => {
      activeTab.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }),
      )
    })
    expect(onSelect).toHaveBeenCalledWith('internet-fa')
  })
})

describe('ShareSegmentedControl — colorblind-safe danger cue (SM-07)', () => {
  it('gives the danger tab the is-danger class and a warning glyph in its rendered text', () => {
    const c = renderControl({ tabs: TABS_POST_CONFIRM, active: 'internet-fa' })
    const tabs = Array.from(c.querySelectorAll('[role="tab"]')) as HTMLButtonElement[]
    const dangerTab = tabs.find((t) => t.className.includes('is-danger'))
    expect(dangerTab).toBeDefined()
    expect(dangerTab!.textContent).toContain('⚠')
  })
})
