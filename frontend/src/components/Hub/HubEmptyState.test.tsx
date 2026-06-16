import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import { HubEmptyState } from './HubEmptyState'

// Render helper
function renderEmptyState(
  overrides: Partial<React.ComponentProps<typeof HubEmptyState>> = {},
) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const defaultProps: React.ComponentProps<typeof HubEmptyState> = {
    variant: 'no-sessions',
    onNewSession: vi.fn(),
    onClearFilter: vi.fn(),
  }
  const props = { ...defaultProps, ...overrides }
  act(() => {
    root.render(<HubEmptyState {...props} />)
  })
  return { container, root, ...props }
}

describe('HubEmptyState — no-sessions variant', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders the .hub__empty-state container', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-sessions' }))
    const el = container.querySelector('.hub__empty-state')
    expect(el).not.toBeNull()
  })

  it('renders the exact heading "No sessions yet"', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-sessions' }))
    const heading = container.querySelector('.hub__empty-heading')
    expect(heading).not.toBeNull()
    expect(heading?.textContent).toBe('No sessions yet')
  })

  it('renders the exact body "Create a session to start an AI coding agent."', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-sessions' }))
    const body = container.querySelector('.hub__empty-body')
    expect(body).not.toBeNull()
    expect(body?.textContent).toBe('Create a session to start an AI coding agent.')
  })

  it('renders a "New session" CTA button', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-sessions' }))
    const cta = container.querySelector('.hub__empty-cta')
    expect(cta).not.toBeNull()
    expect(cta?.textContent?.trim()).toBe('New session')
  })

  it('clicking "New session" CTA fires onNewSession', () => {
    const onNewSession = vi.fn()
    ;({ container, root } = renderEmptyState({ variant: 'no-sessions', onNewSession }))
    const cta = container.querySelector('.hub__empty-cta') as HTMLButtonElement
    act(() => {
      cta.click()
    })
    expect(onNewSession).toHaveBeenCalledTimes(1)
  })

  it('does NOT render a "Clear filter" button in the no-sessions variant', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-sessions' }))
    const buttons = container.querySelectorAll('button')
    const clearBtn = Array.from(buttons).find(
      (b) => b.textContent?.trim() === 'Clear filter',
    )
    expect(clearBtn).toBeUndefined()
  })
})

describe('HubEmptyState — no-matches variant', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders the .hub__empty-state container', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-matches' }))
    const el = container.querySelector('.hub__empty-state')
    expect(el).not.toBeNull()
  })

  it('renders the exact heading "No matching sessions"', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-matches' }))
    const heading = container.querySelector('.hub__empty-heading')
    expect(heading).not.toBeNull()
    expect(heading?.textContent).toBe('No matching sessions')
  })

  it('renders the exact body "Clear the filter or search to see all sessions."', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-matches' }))
    const body = container.querySelector('.hub__empty-body')
    expect(body).not.toBeNull()
    expect(body?.textContent).toBe('Clear the filter or search to see all sessions.')
  })

  it('renders a "Clear filter" CTA button', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-matches' }))
    const cta = container.querySelector('.hub__empty-cta')
    expect(cta).not.toBeNull()
    expect(cta?.textContent?.trim()).toBe('Clear filter')
  })

  it('clicking "Clear filter" CTA fires onClearFilter', () => {
    const onClearFilter = vi.fn()
    ;({ container, root } = renderEmptyState({ variant: 'no-matches', onClearFilter }))
    const cta = container.querySelector('.hub__empty-cta') as HTMLButtonElement
    act(() => {
      cta.click()
    })
    expect(onClearFilter).toHaveBeenCalledTimes(1)
  })

  it('does NOT render a "New session" button in the no-matches variant', () => {
    ;({ container, root } = renderEmptyState({ variant: 'no-matches' }))
    const buttons = container.querySelectorAll('button')
    const newSessionBtn = Array.from(buttons).find(
      (b) => b.textContent?.trim() === 'New session',
    )
    expect(newSessionBtn).toBeUndefined()
  })
})
