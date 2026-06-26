/**
 * MentionPopover component tests — Phase 154-04
 *
 * Covers MENTION-01 / D-07 behaviors:
 *   1. With participants [alice, bob] and filter "" — "Agent" section + @session first,
 *      divider, then alice + bob in Section 2
 *   2. With filter "al" — participant list shows only alice; @session STILL first (never filtered)
 *   3. activeIndex=0 highlights @session (mention-popover__item--active + aria-selected=true)
 *   4. Clicking @session calls onSelect("@session"); clicking alice calls onSelect("@alice")
 *   5. Escape keydown calls onClose
 *
 * Test infrastructure follows the ChatMessage.test.tsx pattern (createRoot + act).
 */
import { describe, it, expect, afterEach, vi } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import { MentionPopover } from './MentionPopover'
import type { PresenceEntry } from '../../lib/relayClient'

// ── helpers ────────────────────────────────────────────────────────────────

function makeParticipant(alias: string, tailnetID?: string): PresenceEntry {
  return {
    personKey: `${tailnetID ?? `node:${alias}`}:web`,
    tailnetID: tailnetID ?? `node:${alias}`,
    origin: 'web',
    alias,
    connCount: 1,
  }
}

function renderPopover(props: {
  participants: PresenceEntry[]
  filter: string
  activeIndex: number
  onSelect: (alias: string) => void
  onClose: () => void
}): { container: HTMLElement; root: ReturnType<typeof createRoot> } {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<MentionPopover {...props} />)
  })
  return { container, root }
}

// ── tests ──────────────────────────────────────────────────────────────────

describe('MentionPopover', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    act(() => { root?.unmount() })
    container?.remove()
  })

  it('renders Agent section with @session first, divider, then participants when filter=""', () => {
    const onSelect = vi.fn()
    const onClose = vi.fn()
    ;({ container, root } = renderPopover({
      participants: [makeParticipant('alice'), makeParticipant('bob')],
      filter: '',
      activeIndex: 0,
      onSelect,
      onClose,
    }))

    // role=listbox container
    const listbox = container.querySelector('[role="listbox"]')
    expect(listbox).not.toBeNull()

    // Section label "Agent" is rendered
    const sectionLabel = container.querySelector('.mention-popover__section-label')
    expect(sectionLabel).not.toBeNull()
    expect(sectionLabel?.textContent).toContain('Agent')

    // @session option exists and is first
    const options = container.querySelectorAll('[role="option"]')
    expect(options.length).toBeGreaterThanOrEqual(3) // @session + alice + bob
    expect(options[0].textContent).toContain('@session')

    // Divider between sections
    expect(container.querySelector('.mention-popover__divider')).not.toBeNull()

    // Participants alice and bob appear
    const allText = listbox?.textContent ?? ''
    expect(allText).toContain('alice')
    expect(allText).toContain('bob')
  })

  it('@session remains first (never filtered) when filter="al" — D-07 invariant', () => {
    const onSelect = vi.fn()
    const onClose = vi.fn()
    ;({ container, root } = renderPopover({
      participants: [makeParticipant('alice'), makeParticipant('bob')],
      filter: 'al',
      activeIndex: 0,
      onSelect,
      onClose,
    }))

    const options = container.querySelectorAll('[role="option"]')
    // @session is STILL first option regardless of filter
    expect(options[0].textContent).toContain('@session')

    // alice appears (matches "al")
    const listbox = container.querySelector('[role="listbox"]')
    expect(listbox?.textContent).toContain('alice')

    // bob does NOT appear (no match for "al")
    expect(listbox?.textContent).not.toContain('bob')
  })

  it('highlights activeIndex=0 with mention-popover__item--active and aria-selected=true', () => {
    const onSelect = vi.fn()
    const onClose = vi.fn()
    ;({ container, root } = renderPopover({
      participants: [makeParticipant('alice')],
      filter: '',
      activeIndex: 0,
      onSelect,
      onClose,
    }))

    const options = container.querySelectorAll('[role="option"]')
    // Option 0 (@session) must be active
    expect(options[0].classList.contains('mention-popover__item--active')).toBe(true)
    expect(options[0].getAttribute('aria-selected')).toBe('true')
    // Option 1 (alice) must NOT be active
    expect(options[1]?.classList.contains('mention-popover__item--active')).toBe(false)
  })

  it('calls onSelect("@session") when @session row is clicked', () => {
    const onSelect = vi.fn()
    const onClose = vi.fn()
    ;({ container, root } = renderPopover({
      participants: [makeParticipant('alice')],
      filter: '',
      activeIndex: 0,
      onSelect,
      onClose,
    }))

    const sessionOption = container.querySelector('.mention-popover__item--session')
    expect(sessionOption).not.toBeNull()
    act(() => {
      sessionOption?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    expect(onSelect).toHaveBeenCalledWith('@session')
  })

  it('calls onSelect("@alice") when alice row is clicked', () => {
    const onSelect = vi.fn()
    const onClose = vi.fn()
    ;({ container, root } = renderPopover({
      participants: [makeParticipant('alice')],
      filter: '',
      activeIndex: 1,
      onSelect,
      onClose,
    }))

    const options = container.querySelectorAll('[role="option"]')
    // Option 1 should be alice
    const aliceOption = options[1]
    expect(aliceOption).not.toBeUndefined()
    act(() => {
      aliceOption?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    expect(onSelect).toHaveBeenCalledWith('@alice')
  })

  it('calls onClose when Escape is pressed (global keydown listener)', () => {
    const onSelect = vi.fn()
    const onClose = vi.fn()
    ;({ container, root } = renderPopover({
      participants: [],
      filter: '',
      activeIndex: 0,
      onSelect,
      onClose,
    }))

    act(() => {
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    expect(onClose).toHaveBeenCalled()
  })
})
