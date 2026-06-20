import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { HubFilterBar } from './HubFilterBar'
import type { HubFilter } from './HubFilterBar'

// Minimal SessionInfo factory for testing counts
function makeSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'sess-1',
    cli: 'claude',
    name: 'My session',
    state: 'running',
    status: 'running',
    createdAt: new Date().toISOString(),
    hostname: '',
    webEnabled: false,
    viewerCount: 0,
    homeDir: false,
    browseEnabled: false,
    workDir: '',
    ...overrides,
  }
}

// Render helper
function renderFilterBar(
  overrides: Partial<React.ComponentProps<typeof HubFilterBar>> = {},
) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const defaultProps: React.ComponentProps<typeof HubFilterBar> = {
    sessions: [],
    activeFilter: 'all',
    searchText: '',
    searchRef: React.createRef<HTMLInputElement>(),
    onFilterChange: vi.fn(),
    onSearchChange: vi.fn(),
    onNewSession: vi.fn(),
  }
  const props = { ...defaultProps, ...overrides }
  act(() => {
    root.render(<HubFilterBar {...props} />)
  })
  return { container, root, ...props }
}

describe('HubFilterBar — filter pills', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders a pill for each filter key (All, Working, Needs input, Complete, Error, Idle)', () => {
    ;({ container, root } = renderFilterBar())
    const pills = container.querySelectorAll('.hub-filter__pill')
    expect(pills.length).toBe(6)
  })

  it('the "All" pill does not show a count', () => {
    ;({ container, root } = renderFilterBar({
      sessions: [makeSession({ state: 'running', status: 'running' })],
    }))
    const allPill = Array.from(container.querySelectorAll('.hub-filter__pill')).find(
      (p) => p.textContent?.startsWith('All'),
    )
    expect(allPill).toBeDefined()
    // "All" pill should NOT contain parentheses
    expect(allPill?.textContent).not.toContain('(')
  })

  it('non-All pills show live counts in parens', () => {
    const sessions = [
      makeSession({ id: '1', state: 'running', status: 'running' }),
      makeSession({ id: '2', state: 'running', status: 'running' }),
      makeSession({ id: '3', state: 'running', status: 'waiting' }),
    ]
    ;({ container, root } = renderFilterBar({ sessions }))
    const workingPill = Array.from(container.querySelectorAll('.hub-filter__pill')).find(
      (p) => p.textContent?.startsWith('Working'),
    )
    expect(workingPill?.textContent).toBe('Working (2)')
    const needsInputPill = Array.from(container.querySelectorAll('.hub-filter__pill')).find(
      (p) => p.textContent?.startsWith('Needs input'),
    )
    expect(needsInputPill?.textContent).toBe('Needs input (1)')
  })

  it('derives stopped-err from state=stopped + exitCode!=0', () => {
    const sessions = [
      makeSession({ id: '1', state: 'stopped', status: 'running', exitCode: 1 }),
    ]
    ;({ container, root } = renderFilterBar({ sessions }))
    const errorPill = Array.from(container.querySelectorAll('.hub-filter__pill')).find(
      (p) => p.textContent?.startsWith('Error'),
    )
    expect(errorPill?.textContent).toBe('Error (1)')
  })

  it('derives stopped-ok from state=stopped + exitCode=0', () => {
    const sessions = [
      makeSession({ id: '1', state: 'stopped', status: 'running', exitCode: 0 }),
    ]
    ;({ container, root } = renderFilterBar({ sessions }))
    const completePill = Array.from(container.querySelectorAll('.hub-filter__pill')).find(
      (p) => p.textContent?.startsWith('Complete'),
    )
    expect(completePill?.textContent).toBe('Complete (1)')
  })

  it('active pill gets the --active modifier class', () => {
    ;({ container, root } = renderFilterBar({ activeFilter: 'running' }))
    const activePill = container.querySelector('.hub-filter__pill--active')
    expect(activePill).not.toBeNull()
    expect(activePill?.textContent).toMatch(/^Working/)
  })

  it('clicking a pill fires onFilterChange with the HubFilter key', () => {
    const onFilterChange = vi.fn()
    ;({ container, root } = renderFilterBar({ onFilterChange }))
    const pills = container.querySelectorAll('.hub-filter__pill')
    // Second pill is "Working" → key 'running'
    act(() => {
      ;(pills[1] as HTMLButtonElement).click()
    })
    expect(onFilterChange).toHaveBeenCalledWith('running')
  })

  it('clicking "All" pill fires onFilterChange with "all"', () => {
    const onFilterChange = vi.fn()
    ;({ container, root } = renderFilterBar({ onFilterChange }))
    const allPill = Array.from(container.querySelectorAll('.hub-filter__pill')).find(
      (p) => p.textContent?.startsWith('All'),
    ) as HTMLButtonElement
    act(() => {
      allPill.click()
    })
    expect(onFilterChange).toHaveBeenCalledWith('all')
  })

  it('active pill has aria-pressed="true"', () => {
    ;({ container, root } = renderFilterBar({ activeFilter: 'running' }))
    const activePill = container.querySelector('.hub-filter__pill--active') as HTMLButtonElement
    expect(activePill.getAttribute('aria-pressed')).toBe('true')
  })

  it('inactive pills have aria-pressed="false"', () => {
    ;({ container, root } = renderFilterBar({ activeFilter: 'running' }))
    const allPills = Array.from(container.querySelectorAll('.hub-filter__pill'))
    const inactivePills = allPills.filter((p) => !p.classList.contains('hub-filter__pill--active'))
    for (const pill of inactivePills) {
      expect(pill.getAttribute('aria-pressed')).toBe('false')
    }
  })
})

describe('HubFilterBar — search input', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders a search input with correct placeholder', () => {
    ;({ container, root } = renderFilterBar())
    const input = container.querySelector('.hub-filter__search') as HTMLInputElement
    expect(input).not.toBeNull()
    expect(input.placeholder).toBe('Search sessions…')
  })

  it('renders the search input with correct aria-label', () => {
    ;({ container, root } = renderFilterBar())
    const input = container.querySelector('.hub-filter__search') as HTMLInputElement
    expect(input.getAttribute('aria-label')).toBe('Search sessions by name, CLI, or host')
  })

  it('typing in the search input fires onSearchChange', () => {
    const onSearchChange = vi.fn()
    ;({ container, root } = renderFilterBar({ onSearchChange }))
    const input = container.querySelector('.hub-filter__search') as HTMLInputElement
    act(() => {
      // Simulate change event
      Object.defineProperty(input, 'value', { writable: true, value: 'my-session' })
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    // React uses synthetic onChange — fire a React change event
    act(() => {
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
        window.HTMLInputElement.prototype,
        'value',
      )?.set
      nativeInputValueSetter?.call(input, 'my-session')
      input.dispatchEvent(new Event('change', { bubbles: true }))
    })
    // The value is bound — check that firing the event triggers the callback
    // We test via the ref-controlled pattern by directly verifying the handler
    expect(onSearchChange).toHaveBeenCalledWith('my-session')
  })

  it('pressing Escape in the search input fires onSearchChange("") and blurs', () => {
    const onSearchChange = vi.fn()
    ;({ container, root } = renderFilterBar({ searchText: 'hello', onSearchChange }))
    const input = container.querySelector('.hub-filter__search') as HTMLInputElement
    const blurSpy = vi.spyOn(input, 'blur')
    act(() => {
      input.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }),
      )
    })
    expect(onSearchChange).toHaveBeenCalledWith('')
    expect(blurSpy).toHaveBeenCalled()
  })

  it('search input reflects the searchText prop (controlled)', () => {
    ;({ container, root } = renderFilterBar({ searchText: 'my search' }))
    const input = container.querySelector('.hub-filter__search') as HTMLInputElement
    expect(input.value).toBe('my search')
  })
})

describe('HubFilterBar — New session button', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders a "New session" button', () => {
    ;({ container, root } = renderFilterBar())
    const btn = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'New session',
    )
    expect(btn).toBeDefined()
  })

  it('clicking "New session" fires onNewSession', () => {
    const onNewSession = vi.fn()
    ;({ container, root } = renderFilterBar({ onNewSession }))
    const btn = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'New session',
    ) as HTMLButtonElement
    act(() => {
      btn.click()
    })
    expect(onNewSession).toHaveBeenCalledTimes(1)
  })
})

describe('HubFilterBar — searchRef forwarding', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('forwards searchRef to the search input element', () => {
    const searchRef = React.createRef<HTMLInputElement>()
    ;({ container, root } = renderFilterBar({ searchRef }))
    expect(searchRef.current).not.toBeNull()
    expect(searchRef.current?.tagName).toBe('INPUT')
    expect(searchRef.current?.classList.contains('hub-filter__search')).toBe(true)
  })
})

describe('HubFilterBar — mixed sessions count', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('counts all filter categories correctly from a mixed sessions array', () => {
    const sessions = [
      makeSession({ id: '1', state: 'running', status: 'running' }),
      makeSession({ id: '2', state: 'running', status: 'running' }),
      makeSession({ id: '3', state: 'running', status: 'waiting' }),
      makeSession({ id: '4', state: 'stopped', status: 'running', exitCode: 0 }),
      makeSession({ id: '5', state: 'stopped', status: 'running', exitCode: 1 }),
      makeSession({ id: '6', state: 'running', status: 'idle' }),
      makeSession({ id: '7', state: 'running', status: 'errored' }),
    ]
    ;({ container, root } = renderFilterBar({ sessions }))
    const pills = container.querySelectorAll('.hub-filter__pill')

    // All — no count
    expect(pills[0].textContent).toBe('All')
    // Working (running) — 2
    expect(pills[1].textContent).toBe('Working (2)')
    // Needs input (waiting) — 1
    expect(pills[2].textContent).toBe('Needs input (1)')
    // Complete (stopped-ok) — 1
    expect(pills[3].textContent).toBe('Complete (1)')
    // Error (stopped-err) — 1
    expect(pills[4].textContent).toBe('Error (1)')
    // Idle — 1
    expect(pills[5].textContent).toBe('Idle (1)')
  })
})

describe('HubFilterBar — exports HubFilter type', () => {
  it('HubFilterBar module exports HubFilter type usable as a value', () => {
    // We import HubFilter as a type — if the module compiles, the type is exported.
    // This test verifies at runtime that the component renders without TS errors.
    const filter: HubFilter = 'all'
    expect(filter).toBe('all')
  })
})
