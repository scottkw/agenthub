/**
 * ChatMessage component tests — Phase 154-03
 *
 * Covers:
 *   - First-in-group render (avatar, alias, tailnet ID, timestamp with ISO title)
 *   - Consecutive collapse (no header, chat-msg__body--consecutive applied)
 *   - @mention-of-me three-signal treatment (NOTIF-02, D-05, colorblind-safe)
 *   - Session-inject indicator (D-06)
 *   - SEC-03: XSS payloads render inert (no script element, no onerror attribute)
 *   - Helper exports: tailnetIdToHue, formatHHMM
 *
 * SEC-03 note: the payload strings used as test inputs below are confined to
 * this file. They must NOT appear in the component source or its comments —
 * the component describes mitigations by concept only.
 */
import { describe, it, expect, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import {
  ChatMessage as ChatMessageComponent,
  tailnetIdToHue,
  formatHHMM,
} from './ChatMessage'
import type { ChatMessage } from '../../lib/relayClient'

// ── helpers ───────────────────────────────────────────────────────────────

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
  return {
    v: 1,
    id: 'msg-001',
    sessionID: 'session-001',
    authorID: 'node:abc123',
    alias: 'Alice',
    content: 'Hello world',
    ts: Date.UTC(2026, 5, 26, 14, 30, 0), // 2026-06-26T14:30:00.000Z
    ...overrides,
  }
}

function renderMessage(
  message: ChatMessage,
  options: { isFirstInGroup?: boolean; isMentionOfMe?: boolean } = {},
): { container: HTMLElement; root: ReturnType<typeof createRoot> } {
  const { isFirstInGroup = true, isMentionOfMe = false } = options
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(
      <ChatMessageComponent
        message={message}
        isFirstInGroup={isFirstInGroup}
        isMentionOfMe={isMentionOfMe}
      />,
    )
  })
  return { container, root }
}

// ── tailnetIdToHue ────────────────────────────────────────────────────────

describe('tailnetIdToHue', () => {
  it('returns a number in [0, 360)', () => {
    const hue = tailnetIdToHue('node:abc123')
    expect(hue).toBeGreaterThanOrEqual(0)
    expect(hue).toBeLessThan(360)
  })

  it('is deterministic — same input returns same hue', () => {
    expect(tailnetIdToHue('node:abc123')).toBe(tailnetIdToHue('node:abc123'))
  })

  it('returns 0 for an empty string (degenerate)', () => {
    expect(tailnetIdToHue('')).toBe(0)
  })
})

// ── formatHHMM ────────────────────────────────────────────────────────────

describe('formatHHMM', () => {
  it('formats 14:30 UTC as "14:30" (no AM/PM)', () => {
    const ts = Date.UTC(2026, 5, 26, 14, 30, 0)
    // Use local-clock formatting — the result depends on the test host timezone,
    // so assert format shape (HH:MM) not exact digits.
    const result = formatHHMM(ts)
    expect(result).toMatch(/^\d{2}:\d{2}$/)
  })

  it('zero-pads single-digit hours and minutes', () => {
    // Construct a local midnight moment — hour 0, minute 5.
    const now = new Date()
    now.setHours(0, 5, 0, 0)
    const result = formatHHMM(now.getTime())
    expect(result).toBe('00:05')
  })
})

// ── First-in-group render ─────────────────────────────────────────────────

describe('ChatMessage — first-in-group', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders the chat-msg root element', () => {
    ;({ container, root } = renderMessage(makeMessage()))
    expect(container.querySelector('.chat-msg')).not.toBeNull()
  })

  it('renders the header row (.chat-msg__header)', () => {
    ;({ container, root } = renderMessage(makeMessage()))
    expect(container.querySelector('.chat-msg__header')).not.toBeNull()
  })

  it('renders the avatar with an uppercase alias initial', () => {
    ;({ container, root } = renderMessage(makeMessage({ alias: 'Alice' })))
    const avatar = container.querySelector('.chat-msg__avatar')
    expect(avatar).not.toBeNull()
    expect(avatar?.textContent).toBe('A')
  })

  it('renders the alias text', () => {
    ;({ container, root } = renderMessage(makeMessage({ alias: 'Alice' })))
    const alias = container.querySelector('.chat-msg__alias')
    expect(alias).not.toBeNull()
    expect(alias?.textContent).toBe('Alice')
  })

  it('renders the tailnet ID in parens', () => {
    ;({ container, root } = renderMessage(makeMessage({ authorID: 'node:abc123' })))
    const tidEl = container.querySelector('.chat-msg__tailnet-id')
    expect(tidEl).not.toBeNull()
    expect(tidEl?.textContent).toContain('node:abc123')
  })

  it('renders a <time> element with the ISO-8601 datetime in the title attribute', () => {
    const ts = Date.UTC(2026, 5, 26, 14, 30, 0) // 2026-06-26T14:30:00.000Z
    ;({ container, root } = renderMessage(makeMessage({ ts })))
    const timeEl = container.querySelector('time.chat-msg__time')
    expect(timeEl).not.toBeNull()
    expect(timeEl?.getAttribute('title')).toBe(new Date(ts).toISOString())
  })

  it('renders the message body (.chat-msg__body)', () => {
    ;({ container, root } = renderMessage(makeMessage({ content: 'Hello world' })))
    const body = container.querySelector('.chat-msg__body')
    expect(body).not.toBeNull()
    expect(body?.textContent).toContain('Hello world')
  })

  it('does NOT apply chat-msg__body--consecutive in first-in-group mode', () => {
    ;({ container, root } = renderMessage(makeMessage(), { isFirstInGroup: true }))
    expect(container.querySelector('.chat-msg__body--consecutive')).toBeNull()
  })
})

// ── Consecutive collapse ──────────────────────────────────────────────────

describe('ChatMessage — consecutive (isFirstInGroup=false)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('does NOT render the header row when isFirstInGroup=false', () => {
    ;({ container, root } = renderMessage(makeMessage(), { isFirstInGroup: false }))
    expect(container.querySelector('.chat-msg__header')).toBeNull()
  })

  it('applies chat-msg__body--consecutive when isFirstInGroup=false', () => {
    ;({ container, root } = renderMessage(makeMessage(), { isFirstInGroup: false }))
    expect(container.querySelector('.chat-msg__body--consecutive')).not.toBeNull()
  })
})

// ── @mention-of-me (NOTIF-02 / D-05) — three independent signals ──────────

describe('ChatMessage — isMentionOfMe (colorblind-safe three-signal)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('Signal 1: adds chat-msg--mention class to the root row', () => {
    ;({ container, root } = renderMessage(makeMessage(), { isMentionOfMe: true }))
    const row = container.querySelector('.chat-msg')
    expect(row?.classList.contains('chat-msg--mention')).toBe(true)
  })

  it('Signal 2: renders a .chat-msg__you-chip element (glyph channel)', () => {
    ;({ container, root } = renderMessage(makeMessage(), { isMentionOfMe: true }))
    expect(container.querySelector('.chat-msg__you-chip')).not.toBeNull()
  })

  it('Signal 3: row has role="listitem" and aria-label containing "mentioned you"', () => {
    ;({ container, root } = renderMessage(makeMessage(), { isMentionOfMe: true }))
    const row = container.querySelector('[role="listitem"]')
    expect(row).not.toBeNull()
    expect(row?.getAttribute('aria-label')).toContain('mentioned you')
  })

  it('does NOT add chat-msg--mention when isMentionOfMe=false', () => {
    ;({ container, root } = renderMessage(makeMessage(), { isMentionOfMe: false }))
    const row = container.querySelector('.chat-msg')
    expect(row?.classList.contains('chat-msg--mention')).toBe(false)
  })

  it('does NOT render .chat-msg__you-chip when isMentionOfMe=false', () => {
    ;({ container, root } = renderMessage(makeMessage(), { isMentionOfMe: false }))
    expect(container.querySelector('.chat-msg__you-chip')).toBeNull()
  })
})

// ── Session-inject indicator (D-06) ───────────────────────────────────────

describe('ChatMessage — sessionInject indicator', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders .chat-msg__inject-indicator when sessionInject=true', () => {
    const message = makeMessage({ sessionInject: true })
    ;({ container, root } = renderMessage(message))
    expect(container.querySelector('.chat-msg__inject-indicator')).not.toBeNull()
  })

  it('renders the exact caption "→ injected into terminal"', () => {
    const message = makeMessage({ sessionInject: true })
    ;({ container, root } = renderMessage(message))
    const caption = container.querySelector('.chat-msg__inject-caption')
    expect(caption?.textContent).toContain('→ injected into terminal')
  })

  it('sets aria-label "Injected into terminal by {alias}" on the indicator', () => {
    const message = makeMessage({ alias: 'Alice', sessionInject: true })
    ;({ container, root } = renderMessage(message))
    const indicator = container.querySelector('.chat-msg__inject-indicator')
    expect(indicator?.getAttribute('aria-label')).toBe('Injected into terminal by Alice')
  })

  it('does NOT render .chat-msg__inject-indicator when sessionInject is absent', () => {
    const message = makeMessage() // no sessionInject
    ;({ container, root } = renderMessage(message))
    expect(container.querySelector('.chat-msg__inject-indicator')).toBeNull()
  })
})

// ── SEC-03: XSS safety ────────────────────────────────────────────────────
// These tests use unsafe payload strings as test INPUT. The payloads are
// confined to this file and this describe block — they must not appear in
// the component source. See component JSDoc for conceptual description only.

describe('ChatMessage — SEC-03 XSS safety (rehype-sanitize + react-markdown v10)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders a script-tag payload inert — no <script> element in DOM', () => {
    // Payload: raw script injection attempt
    const message = makeMessage({ content: '<script>alert(1)</script>' })
    ;({ container, root } = renderMessage(message))
    // ASSERT: no script element exists in the rendered DOM
    expect(container.querySelector('script')).toBeNull()
  })

  it('renders an img-onerror payload inert — no onerror attribute in DOM', () => {
    // Payload: event-handler injection attempt via img attribute
    const message = makeMessage({ content: '<img src=x onerror=alert(1)>' })
    ;({ container, root } = renderMessage(message))
    // ASSERT: no element carries an onerror attribute
    const withOnerror = container.querySelectorAll('[onerror]')
    expect(withOnerror.length).toBe(0)
  })
})
