/**
 * ChatDaySeparator component tests — Phase 154-03
 *
 * Covers:
 *   - Render structure: centered label between two hr rules with correct aria-label
 *   - formatDaySeparator returns "Today" for timestamps in the current day
 *   - formatDaySeparator returns "Yesterday" for timestamps in the previous day
 *   - formatDaySeparator returns a locale-short label ("Mon, Jun 23") for older dates
 *   - Component does NOT set position:sticky (parent virtualizer owns stickiness)
 */
import { describe, it, expect, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import { ChatDaySeparator, formatDaySeparator } from './ChatDaySeparator'

// ── render helper ──────────────────────────────────────────────────────────

function renderSeparator(label: string): {
  container: HTMLElement
  root: ReturnType<typeof createRoot>
} {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<ChatDaySeparator label={label} />)
  })
  return { container, root }
}

// ── ChatDaySeparator component ─────────────────────────────────────────────

describe('ChatDaySeparator — render structure', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders the .chat-day-sep root element', () => {
    ;({ container, root } = renderSeparator('Today'))
    expect(container.querySelector('.chat-day-sep')).not.toBeNull()
  })

  it('renders a .chat-day-sep__label with the provided label text', () => {
    ;({ container, root } = renderSeparator('Today'))
    const label = container.querySelector('.chat-day-sep__label')
    expect(label).not.toBeNull()
    expect(label?.textContent).toBe('Today')
  })

  it('renders exactly two .chat-day-sep__rule hr elements', () => {
    ;({ container, root } = renderSeparator('Today'))
    const rules = container.querySelectorAll('.chat-day-sep__rule')
    expect(rules.length).toBe(2)
  })

  it('sets aria-label to "Messages from {label}" on the root', () => {
    ;({ container, root } = renderSeparator('Today'))
    const root_el = container.querySelector('.chat-day-sep')
    expect(root_el?.getAttribute('aria-label')).toBe('Messages from Today')
  })

  it('aria-label uses the exact label prop — works for Yesterday too', () => {
    ;({ container, root } = renderSeparator('Yesterday'))
    const root_el = container.querySelector('.chat-day-sep')
    expect(root_el?.getAttribute('aria-label')).toBe('Messages from Yesterday')
  })

  it('renders label before a trailing hr (label is flanked by rules)', () => {
    ;({ container, root } = renderSeparator('Today'))
    const sep = container.querySelector('.chat-day-sep')!
    const children = Array.from(sep.children)
    // Expect: [hr, span, hr] — label in the middle
    expect(children[0].tagName).toBe('HR')
    expect(children[1].tagName).toBe('SPAN')
    expect(children[2].tagName).toBe('HR')
  })

  it('does NOT apply position:sticky inline style (parent virtualizer owns stickiness)', () => {
    ;({ container, root } = renderSeparator('Today'))
    const sep = container.querySelector('.chat-day-sep') as HTMLElement | null
    expect(sep?.style.position).not.toBe('sticky')
  })
})

// ── formatDaySeparator ────────────────────────────────────────────────────

describe('formatDaySeparator', () => {
  it('returns "Today" for a timestamp earlier today', () => {
    const now = new Date()
    // A moment earlier today (9am local)
    const todayAt9 = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 9, 0, 0)
    expect(formatDaySeparator(todayAt9.getTime())).toBe('Today')
  })

  it('returns "Today" for a timestamp at midnight today', () => {
    const now = new Date()
    const midnight = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0)
    expect(formatDaySeparator(midnight.getTime())).toBe('Today')
  })

  it('returns "Yesterday" for a timestamp anywhere in the previous day', () => {
    const now = new Date()
    const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1, 12, 0, 0)
    expect(formatDaySeparator(yesterday.getTime())).toBe('Yesterday')
  })

  it('returns a locale-short label (not Today or Yesterday) for a date 2 days ago', () => {
    const now = new Date()
    const twoDaysAgo = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 2, 12, 0, 0)
    const result = formatDaySeparator(twoDaysAgo.getTime())
    expect(result).not.toBe('Today')
    expect(result).not.toBe('Yesterday')
    // Should contain a weekday abbreviation and day number (e.g. "Mon, Jun 23")
    expect(result).toMatch(/\w+,?\s+\w+\s+\d+/)
  })

  it('returns a locale-short label for a distant past date', () => {
    const oldDate = new Date(2020, 0, 15, 10, 0, 0) // Jan 15 2020
    const result = formatDaySeparator(oldDate.getTime())
    expect(result).not.toBe('Today')
    expect(result).not.toBe('Yesterday')
    expect(typeof result).toBe('string')
    expect(result.length).toBeGreaterThan(0)
  })
})
