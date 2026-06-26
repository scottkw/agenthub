/**
 * ChatBadge component tests — Phase 154-04
 *
 * Covers NOTIF-01 / D-10: unread badge behaviors.
 *   1. count=0 → renders null (no DOM node)
 *   2. count=3, hasMention=false → shows "3" with singular/plural aria-label
 *   3. count=1, hasMention=false → uses singular aria-label "1 unread message"
 *   4. count=3, hasMention=true → renders "@" glyph (NOT "3") with aria-label
 *      containing "mention" — glyph + aria-label are two non-color signals (D-10)
 *
 * Test infrastructure follows the ChatMessage.test.tsx pattern (createRoot + act).
 */
import { describe, it, expect, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import { ChatBadge } from './ChatBadge'

function renderBadge(
  props: { count: number; hasMention: boolean },
): { container: HTMLElement; root: ReturnType<typeof createRoot> } {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<ChatBadge count={props.count} hasMention={props.hasMention} />)
  })
  return { container, root }
}

describe('ChatBadge', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    act(() => { root?.unmount() })
    container?.remove()
  })

  it('renders nothing (null) when count is 0', () => {
    ;({ container, root } = renderBadge({ count: 0, hasMention: false }))
    // No DOM node should be rendered
    expect(container.firstChild).toBeNull()
  })

  it('renders the count number as text for normal unread (count=3, hasMention=false)', () => {
    ;({ container, root } = renderBadge({ count: 3, hasMention: false }))
    const badge = container.querySelector('.chat-badge')
    expect(badge).not.toBeNull()
    expect(badge?.textContent).toBe('3')
    // Must NOT render the "@" glyph
    expect(badge?.textContent).not.toBe('@')
    expect(badge?.getAttribute('aria-label')).toBe('3 unread messages')
  })

  it('uses singular aria-label when count is 1 (1 unread message, not messages)', () => {
    ;({ container, root } = renderBadge({ count: 1, hasMention: false }))
    const badge = container.querySelector('.chat-badge')
    expect(badge).not.toBeNull()
    expect(badge?.getAttribute('aria-label')).toBe('1 unread message')
  })

  it('renders "@" glyph (NOT the count) when hasMention=true — glyph is the non-color signal (D-10)', () => {
    ;({ container, root } = renderBadge({ count: 3, hasMention: true }))
    const badge = container.querySelector('.chat-badge')
    expect(badge).not.toBeNull()
    // Primary non-color signal: "@" glyph replaces the count number
    expect(badge?.textContent).toBe('@')
    // Second non-color signal: aria-label must contain "mention" for screen readers
    expect(badge?.getAttribute('aria-label')).toBe('3 unread messages, including a mention')
    // chat-badge--mention CSS modifier applied
    expect(badge?.classList.contains('chat-badge--mention')).toBe(true)
  })
})
