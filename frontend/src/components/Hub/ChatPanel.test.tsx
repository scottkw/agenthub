/**
 * ChatPanel.test.tsx — Phase 154-05 (TDD)
 *
 * Task 1 tests (RED 1 → GREEN 1):
 *   - buildItems: day grouping, separator insertion, consecutive-author collapse
 *   - mergeWithDedup: WS+history deduplication by message id
 *   - Subscription: separate RelayClient constructed on mount, closed on unmount
 *   - Loading state: "Connecting…" immediately on mount
 *   - Empty state: "No messages yet" after WS opens with zero messages
 *
 * Task 2 tests (RED 2 → GREEN 2):
 *   - getRowStyle (sticky): active separator gets NO transform; others get translateY
 *   - accrueUnread: increments count, sets hasMention, preserves once set
 *   - Component-level clear: onUnreadChange(0,false) fires when open+focused
 *
 * Architecture note: tests operate on exported pure helpers without requiring
 * virtualizer pixel measurement — jsdom returns 0 for getBoundingClientRect.
 * Sticky behaviour during live scroll is a manual UAT (154-06 TESTING.md §5).
 */
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'

// ── Mock state (hoisted so it is available when vi.mock factory runs) ──────

const mocks = vi.hoisted(() => ({
  ctorCount: 0,
  lastPort: 0,
  lastSessionId: '',
  lastCallbacks: null as Record<string, unknown> | null,
  mockClose: vi.fn(),
  mockSendChat: vi.fn(),
  mockSendSessionInject: vi.fn(),
  mockSendAliasSet: vi.fn(),
}))

// ── Mock RelayClient ───────────────────────────────────────────────────────
// ChatPanel opens its OWN RelayClient (Pattern 2 — separate subscription).
// The mock uses a class (not vi.fn().mockImplementation(arrow)) so it is
// constructable with `new` and doesn't hit "is not a constructor" in vitest.

vi.mock('../../lib/relayClient', async (importActual) => {
  const actual = await importActual<typeof import('../../lib/relayClient')>()
  return {
    ...actual,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    RelayClient: class MockRelayClient {
      constructor(port: number, sessionId: string, cbs: unknown) {
        mocks.ctorCount++
        mocks.lastPort = port
        mocks.lastSessionId = sessionId
        mocks.lastCallbacks = cbs as Record<string, unknown>
      }
      close() { mocks.mockClose() }
      sendChat(content: string) { mocks.mockSendChat(content) }
      sendSessionInject(text: string) { mocks.mockSendSessionInject(text) }
      sendAliasSet(alias: string) { mocks.mockSendAliasSet(alias) }
    } as unknown as (typeof actual)['RelayClient'],
  }
})

// ── Mock fetch (for loadChatHistory) ──────────────────────────────────────

const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

// ── Imports from file under test ───────────────────────────────────────────
// NOTE: placed after vi.mock — vitest hoists vi.mock but imports must follow.

import {
  buildItems,
  mergeWithDedup,
  accrueUnread,
  getRowStyle,
  validateAlias,
  clampChatWidth,
  CHAT_WIDTH_MIN,
  CHAT_WIDTH_MAX,
  CHAT_WIDTH_DEFAULT,
  ChatPanel,
} from './ChatPanel'
import type { UnreadState } from './ChatPanel'
import type { ChatMessage, PresenceEntry } from '../../lib/relayClient'

// ── Helpers ────────────────────────────────────────────────────────────────

function makeMsg(overrides: Partial<ChatMessage> = {}): ChatMessage {
  return {
    v: 1,
    id: 'msg-001',
    sessionID: 'sess-001',
    authorID: 'local',
    alias: 'Alice',
    content: 'Hello',
    ts: Date.UTC(2026, 5, 26, 14, 0, 0), // 2026-06-26 14:00 UTC
    ...overrides,
  }
}

/** Mount ChatPanel into a fresh container; returns unmount helper. */
function mountPanel(props: Partial<Parameters<typeof ChatPanel>[0]> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(
      <ChatPanel
        sessionId="sess-001"
        relayPort={8080}
        open={false}
        {...props}
      />,
    )
  })
  return {
    container,
    root,
    unmount: () => {
      act(() => { root.unmount() })
      container.remove()
    },
  }
}

// ── Global setup ───────────────────────────────────────────────────────────

beforeEach(() => {
  mocks.ctorCount = 0
  mocks.lastPort = 0
  mocks.lastSessionId = ''
  mocks.lastCallbacks = null
  mocks.mockClose.mockClear()
  mocks.mockSendChat.mockClear()
  mocks.mockSendSessionInject.mockClear()
  mocks.mockSendAliasSet.mockClear()
  mockFetch.mockReset()
  // Default: history endpoint returns empty list
  mockFetch.mockResolvedValue({
    ok: true,
    json: () => Promise.resolve([]),
  })
})

afterEach(() => {
  vi.clearAllMocks()
})

// ═══════════════════════════════════════════════════════════════════════════
// TASK 1 — subscription + history + buildItems + states
// ═══════════════════════════════════════════════════════════════════════════

// ── buildItems ─────────────────────────────────────────────────────────────

describe('buildItems — day grouping and consecutive-author collapse', () => {
  it('returns empty array for no messages', () => {
    expect(buildItems([])).toEqual([])
  })

  it('inserts a separator before the first message of the day', () => {
    const msg = makeMsg({ ts: new Date(2025, 0, 1, 12, 0, 0).getTime() })
    const items = buildItems([msg])
    expect(items[0].type).toBe('separator')
    expect(items[1].type).toBe('message')
    expect(items.length).toBe(2)
  })

  it('marks the first message in a group as NOT consecutive', () => {
    const items = buildItems([makeMsg()])
    const msgItem = items.find(i => i.type === 'message')
    expect(msgItem?.type === 'message' && msgItem.isConsecutive).toBe(false)
  })

  it('marks same-author message within 5 min as consecutive', () => {
    const base = new Date(2025, 0, 1, 12, 0, 0).getTime()
    const msgs = [
      makeMsg({ id: 'm1', authorID: 'alice', ts: base }),
      makeMsg({ id: 'm2', authorID: 'alice', ts: base + 2 * 60 * 1000 }), // 2 min later
    ]
    const items = buildItems(msgs)
    const msgItems = items.filter(i => i.type === 'message')
    expect(msgItems[0].type === 'message' && msgItems[0].isConsecutive).toBe(false)
    expect(msgItems[1].type === 'message' && msgItems[1].isConsecutive).toBe(true)
  })

  it('does NOT mark as consecutive when gap > 5 min', () => {
    const base = new Date(2025, 0, 1, 12, 0, 0).getTime()
    const msgs = [
      makeMsg({ id: 'm1', authorID: 'alice', ts: base }),
      makeMsg({ id: 'm2', authorID: 'alice', ts: base + 6 * 60 * 1000 }), // 6 min later
    ]
    const items = buildItems(msgs)
    const msgItems = items.filter(i => i.type === 'message')
    expect(msgItems[1].type === 'message' && msgItems[1].isConsecutive).toBe(false)
  })

  it('does NOT mark as consecutive when author differs', () => {
    const base = new Date(2025, 0, 1, 12, 0, 0).getTime()
    const msgs = [
      makeMsg({ id: 'm1', authorID: 'alice', ts: base }),
      makeMsg({ id: 'm2', authorID: 'bob', ts: base + 60 * 1000 }), // 1 min later, different author
    ]
    const items = buildItems(msgs)
    const msgItems = items.filter(i => i.type === 'message')
    expect(msgItems[1].type === 'message' && msgItems[1].isConsecutive).toBe(false)
  })

  it('inserts a new separator when messages cross a day boundary', () => {
    const day1 = new Date(2025, 0, 1, 23, 59, 0).getTime()
    const day2 = new Date(2025, 0, 2, 0, 1, 0).getTime()
    const msgs = [
      makeMsg({ id: 'm1', ts: day1 }),
      makeMsg({ id: 'm2', ts: day2 }),
    ]
    const items = buildItems(msgs)
    const separators = items.filter(i => i.type === 'separator')
    expect(separators.length).toBe(2)
  })

  it('resets consecutive tracking at a day boundary (same author, same day, <5 min but crosses midnight)', () => {
    const day1 = new Date(2025, 0, 1, 23, 58, 0).getTime()
    const day2 = new Date(2025, 0, 2, 0, 1, 0).getTime() // crosses midnight, 3 min later
    const msgs = [
      makeMsg({ id: 'm1', authorID: 'alice', ts: day1 }),
      makeMsg({ id: 'm2', authorID: 'alice', ts: day2 }),
    ]
    const items = buildItems(msgs)
    const msgItems = items.filter(i => i.type === 'message')
    // Second message is first in a new day — NOT consecutive even though same author + <5 min
    expect(msgItems[1].type === 'message' && msgItems[1].isConsecutive).toBe(false)
  })
})

// ── mergeWithDedup ─────────────────────────────────────────────────────────

describe('mergeWithDedup — WS+history deduplication by message id', () => {
  it('adds new messages to an empty list', () => {
    const seenIds = new Set<string>()
    const msg = makeMsg({ id: 'msg-001' })
    const result = mergeWithDedup([], [msg], seenIds)
    expect(result.length).toBe(1)
    expect(result[0].id).toBe('msg-001')
  })

  it('deduplicates when the same message id appears twice (WS then history)', () => {
    const seenIds = new Set<string>()
    const msg = makeMsg({ id: 'dup' })
    // First add: from WS
    const after1 = mergeWithDedup([], [msg], seenIds)
    // Second add: same id from history
    const after2 = mergeWithDedup(after1, [msg], seenIds)
    expect(after2.length).toBe(1) // NOT 2
  })

  it('records seen IDs in the provided set (mutation)', () => {
    const seenIds = new Set<string>()
    const msg = makeMsg({ id: 'seen' })
    mergeWithDedup([], [msg], seenIds)
    expect(seenIds.has('seen')).toBe(true)
  })

  it('appends genuinely new messages', () => {
    const seenIds = new Set<string>()
    const m1 = makeMsg({ id: 'm1' })
    const m2 = makeMsg({ id: 'm2' })
    const after1 = mergeWithDedup([], [m1], seenIds)
    const after2 = mergeWithDedup(after1, [m2], seenIds)
    expect(after2.length).toBe(2)
    expect(after2[1].id).toBe('m2')
  })

  it('returns the same reference when no new messages are added (avoids re-render)', () => {
    const seenIds = new Set<string>(['dup'])
    const msg = makeMsg({ id: 'dup' })
    const current = [msg]
    const result = mergeWithDedup(current, [msg], seenIds)
    expect(result).toBe(current) // same array ref
  })
})

// ── ChatPanel subscription ─────────────────────────────────────────────────

describe('ChatPanel — separate RelayClient subscription', () => {
  it('constructs exactly one RelayClient on mount', () => {
    const { unmount } = mountPanel()
    expect(mocks.ctorCount).toBe(1)
    unmount()
  })

  it('passes the correct relayPort to RelayClient', () => {
    const { unmount } = mountPanel({ relayPort: 9999 })
    expect(mocks.lastPort).toBe(9999)
    unmount()
  })

  it('passes the correct sessionId to RelayClient', () => {
    const { unmount } = mountPanel({ sessionId: 'my-session' })
    expect(mocks.lastSessionId).toBe('my-session')
    unmount()
  })

  it('registers an onChat callback on the RelayClient', () => {
    const { unmount } = mountPanel()
    expect(mocks.lastCallbacks).not.toBeNull()
    expect(typeof mocks.lastCallbacks?.onChat).toBe('function')
    unmount()
  })

  it('calls RelayClient.close() on unmount', () => {
    const { unmount } = mountPanel()
    expect(mocks.mockClose).not.toHaveBeenCalled()
    unmount()
    expect(mocks.mockClose).toHaveBeenCalledOnce()
  })
})

// ── ChatPanel states ───────────────────────────────────────────────────────

describe('ChatPanel — loading and empty states', () => {
  it('renders loading text "Connecting" before the WS opens', () => {
    const { container, unmount } = mountPanel()
    const loading = container.querySelector('.chat-panel__loading')
    expect(loading).not.toBeNull()
    expect(loading?.textContent).toContain('Connecting')
    unmount()
  })

  it('renders the empty state after WS opens with no messages', async () => {
    mockFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    const { container, unmount } = mountPanel()

    await act(async () => {
      // Simulate WS open → triggers loadChatHistory
      ;(mocks.lastCallbacks?.onOpen as (() => void) | undefined)?.()
    })

    expect(container.querySelector('.chat-panel__empty')).not.toBeNull()
    expect(container.textContent).toContain('No messages yet')
    expect(container.textContent).toContain('Send a message to start the conversation.')
    unmount()
  })

  it('renders the drawer title "Chat" in the header', () => {
    const { container, unmount } = mountPanel()
    const title = container.querySelector('.chat-panel__title')
    expect(title?.textContent).toBe('Chat')
    unmount()
  })
})

// ═══════════════════════════════════════════════════════════════════════════
// TASK 2 — virtualizer sticky style + unread accrual
// ═══════════════════════════════════════════════════════════════════════════

// ── getRowStyle — sticky separator (CHAT-04, Pitfall 1) ───────────────────

describe('sticky separator row style (CHAT-04)', () => {
  it('active separator: position:sticky, top:0, zIndex:2 and NO transform', () => {
    const style = getRowStyle(true, 100)
    expect(style.position).toBe('sticky')
    expect(style.top).toBe(0)
    expect(style.zIndex).toBe(2)
    // CRITICAL: no transform on the sticky row (Pitfall 1 — transform breaks position:sticky)
    expect(Object.keys(style)).not.toContain('transform')
  })

  it('active separator: NO width property (must not break sticky separator layout)', () => {
    // Adding width to the separator branch would regress CHAT-04 sticky behaviour (Pitfall 1)
    const style = getRowStyle(true, 100)
    expect(Object.keys(style)).not.toContain('width')
  })

  it('non-active row: position:absolute with translateY transform', () => {
    const style = getRowStyle(false, 100)
    expect(style.position).toBe('absolute')
    expect(style.top).toBe(0)
    expect(style.left).toBe(0)
    expect(style.transform).toBe('translateY(100px)')
  })

  it('non-active row: width is "100%" (CHAT-LAYOUT-01 root-cause fix — bounds row to thread width so WEBCHAT-06 ellipsis engages)', () => {
    // Without width:'100%', the absolutely-positioned row shrinks to its content width,
    // so the existing ellipsis on .chat-msg__tailnet-id has no bounded container to truncate within.
    const style = getRowStyle(false, 120)
    expect(style.width).toBe('100%')
  })

  it('non-active row: right is 0 (prevents overflow beyond right edge)', () => {
    const style = getRowStyle(false, 120)
    expect(style.right).toBe(0)
  })

  it('non-active row translateY uses exact start value', () => {
    expect(getRowStyle(false, 0).transform).toBe('translateY(0px)')
    expect(getRowStyle(false, 456).transform).toBe('translateY(456px)')
    expect(getRowStyle(false, 999).transform).toBe('translateY(999px)')
  })
})

// ── accrueUnread — D-09 unread accrual pure helper ────────────────────────

describe('unread accrual (D-09 — pure helper)', () => {
  const noMentionMsg = makeMsg({ id: 'nm', mentions: [] })
  const mentionMsg = makeMsg({ id: 'mm', mentions: ['current-user'] })

  it('increments count by 1 per message', () => {
    const next = accrueUnread({ count: 0, hasMention: false }, noMentionMsg, 'current-user')
    expect(next.count).toBe(1)
  })

  it('accumulates count across consecutive calls', () => {
    let state: UnreadState = { count: 0, hasMention: false }
    state = accrueUnread(state, makeMsg({ id: 'a' }), 'me')
    state = accrueUnread(state, makeMsg({ id: 'b' }), 'me')
    state = accrueUnread(state, makeMsg({ id: 'c' }), 'me')
    expect(state.count).toBe(3)
  })

  it('sets hasMention=true when message.mentions includes currentUserTailnetID', () => {
    const next = accrueUnread({ count: 0, hasMention: false }, mentionMsg, 'current-user')
    expect(next.hasMention).toBe(true)
  })

  it('does NOT set hasMention for a different user in mentions', () => {
    const next = accrueUnread({ count: 0, hasMention: false }, mentionMsg, 'other-user')
    expect(next.hasMention).toBe(false)
  })

  it('does NOT set hasMention when mentions is undefined', () => {
    const msg = makeMsg({ id: 'x', mentions: undefined })
    const next = accrueUnread({ count: 0, hasMention: false }, msg, 'me')
    expect(next.hasMention).toBe(false)
  })

  it('preserves hasMention=true once set (sticky flag)', () => {
    const first = accrueUnread({ count: 0, hasMention: false }, mentionMsg, 'current-user')
    expect(first.hasMention).toBe(true)
    // Next message has no mention — hasMention should stay true
    const second = accrueUnread(first, noMentionMsg, 'current-user')
    expect(second.hasMention).toBe(true)
  })
})

// ── Component-level unread: clear on open+focus ───────────────────────────

describe('unread accrual — clear on open+focused (D-09, component level)', () => {
  it('clears unread via onUnreadChange(0,false) when drawer opens with window focused', async () => {
    const onUnreadChange = vi.fn()
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)

    // Mount with open=false
    act(() => {
      root.render(
        <ChatPanel
          sessionId="sess-001"
          relayPort={8080}
          open={false}
          onUnreadChange={onUnreadChange}
        />,
      )
    })

    // Force window to blurred state (messages will accrue)
    act(() => { window.dispatchEvent(new Event('blur')) })

    // Open WS and empty history
    await act(async () => {
      ;(mocks.lastCallbacks?.onOpen as (() => void) | undefined)?.()
    })

    // Accrue 2 messages while closed+blurred
    act(() => {
      const onChat = mocks.lastCallbacks?.onChat as ((m: ChatMessage) => void) | undefined
      onChat?.(makeMsg({ id: 'a', mentions: [] }))
      onChat?.(makeMsg({ id: 'b', mentions: [] }))
    })

    // Reset call tracking before the clear
    onUnreadChange.mockClear()

    // Focus window then open drawer → should clear unread
    act(() => {
      window.dispatchEvent(new Event('focus'))
      root.render(
        <ChatPanel
          sessionId="sess-001"
          relayPort={8080}
          open={true}
          onUnreadChange={onUnreadChange}
        />,
      )
    })

    expect(onUnreadChange).toHaveBeenCalledWith(0, false)

    act(() => { root.unmount() })
    container.remove()
  })
})

// ═══════════════════════════════════════════════════════════════════════════
// Phase 161 TASK 1 — validateAlias client mirror of Go ValidateAlias
// ═══════════════════════════════════════════════════════════════════════════

describe('validateAlias — client mirror of Go ValidateAlias (Array.from code-point parity)', () => {
  it('trims surrounding whitespace and returns trimmed string', () => {
    expect(validateAlias('  ken  ')).toBe('ken')
  })

  it('returns null for empty string', () => {
    expect(validateAlias('')).toBeNull()
  })

  it('returns null for whitespace-only string', () => {
    expect(validateAlias('   ')).toBeNull()
  })

  it('accepts exactly 32 code points', () => {
    const s = 'a'.repeat(32)
    expect(validateAlias(s)).toBe(s)
  })

  it('rejects 33 code points (reject, not truncate)', () => {
    expect(validateAlias('a'.repeat(33))).toBeNull()
  })

  it('rejects a C0 control char (tab 0x09)', () => {
    expect(validateAlias('hello\tthere')).toBeNull()
  })

  it('rejects a C0 control char (newline 0x0A)', () => {
    expect(validateAlias('hello\nthere')).toBeNull()
  })

  it('rejects a C1 control char (0x80)', () => {
    expect(validateAlias('hellothere')).toBeNull()
  })

  it('rejects a C1 control char at boundary (0x9F)', () => {
    expect(validateAlias('hellothere')).toBeNull()
  })

  it('accepts DEL (0x7E) — boundary just below the C1 block', () => {
    // 0x7E is '~', which is below 0x7F; 0x7F is DEL itself
    // Go rejects 0x7F..0x9F; 0x7E is valid
    expect(validateAlias('hello~there')).toBe('hello~there')
  })

  it('accepts a 32-emoji string (Array.from vs .length guard)', () => {
    // Each emoji is 2 UTF-16 code units (String.length = 64) but 1 code point.
    // Proves code-point counting via Array.from, not .length.
    const emoji32 = '😀'.repeat(32) // '😀' × 32; String.length = 64
    expect(validateAlias(emoji32)).toBe(emoji32)
  })

  it('rejects a 33-emoji string (code-point limit)', () => {
    const emoji33 = '😀'.repeat(33) // String.length = 66
    expect(validateAlias(emoji33)).toBeNull()
  })
})

// ═══════════════════════════════════════════════════════════════════════════
// TASK 1 (154-06) — inject gesture, composer keyboard, sec-03 integration
// ═══════════════════════════════════════════════════════════════════════════

// Helper: set the value of a textarea via the native setter (React picks it up)
function setTextareaValue(el: HTMLTextAreaElement | null, value: string) {
  if (!el) return
  const nativeSetter = Object.getOwnPropertyDescriptor(
    HTMLTextAreaElement.prototype,
    'value',
  )?.set
  if (nativeSetter) {
    nativeSetter.call(el, value)
  } else {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ;(el as any).value = value
  }
  el.dispatchEvent(new Event('input', { bubbles: true }))
}

// ── inject button — press-and-hold gesture (D-08, Pitfall 7) ──────────────

describe('inject button — press-and-hold gesture (D-08, Pitfall 7)', () => {
  it('-t inject: tap < 600ms fires nothing (pointerup before threshold)', () => {
    vi.useFakeTimers()
    const { container, unmount } = mountPanel()

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, '@session hello') })

    const injectBtn = container.querySelector('.chat-composer__inject-btn')
    act(() => {
      injectBtn?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerId: 1 }))
    })
    // Release before 600ms
    act(() => { vi.advanceTimersByTime(400) })
    act(() => {
      injectBtn?.dispatchEvent(new PointerEvent('pointerup', { bubbles: true, pointerId: 1 }))
    })

    expect(mocks.mockSendSessionInject).not.toHaveBeenCalled()

    vi.useRealTimers()
    unmount()
  })

  it('-t inject: hold >= 600ms fires exactly one inject frame', () => {
    vi.useFakeTimers()
    const { container, unmount } = mountPanel()

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, '@session hello') })

    const injectBtn = container.querySelector('.chat-composer__inject-btn')
    act(() => {
      injectBtn?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerId: 1 }))
    })
    // Advance to threshold
    act(() => { vi.advanceTimersByTime(600) })

    expect(mocks.mockSendSessionInject).toHaveBeenCalledOnce()
    expect(mocks.mockSendChat).not.toHaveBeenCalled()

    vi.useRealTimers()
    unmount()
  })

  it('-t inject: Enter with @session fires chat-send, NOT inject (Pitfall 7)', () => {
    const { container, unmount } = mountPanel()

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, '@session hello') })

    // Press Enter — must always route to chat-send, never inject
    act(() => {
      textarea?.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }),
      )
    })

    expect(mocks.mockSendChat).toHaveBeenCalledOnce()
    expect(mocks.mockSendSessionInject).not.toHaveBeenCalled()

    unmount()
  })
})

// ── composer — Enter send, Shift+Enter newline, @ mention popover ──────────

describe('composer — Enter send, Shift+Enter newline, @-mention popover', () => {
  it('-t composer: Enter sends via sendChat and clears the textarea', () => {
    const { container, unmount } = mountPanel()

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, 'Hello world') })

    act(() => {
      textarea?.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }),
      )
    })

    expect(mocks.mockSendChat).toHaveBeenCalledOnce()
    expect(mocks.mockSendChat).toHaveBeenCalledWith('Hello world')
    // Textarea should be cleared after send
    expect(container.querySelector('textarea')?.value ?? '').toBe('')

    unmount()
  })

  it('-t composer: Shift+Enter inserts newline and does NOT send', () => {
    const { container, unmount } = mountPanel()

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, 'Line one') })

    act(() => {
      textarea?.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Enter', shiftKey: true, bubbles: true, cancelable: true }),
      )
    })

    // Chat send must NOT have fired
    expect(mocks.mockSendChat).not.toHaveBeenCalled()

    unmount()
  })

  it('-t composer: typing @ opens MentionPopover', () => {
    const { container, unmount } = mountPanel()

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, '@ali') })

    // MentionPopover should be visible
    const popover = container.querySelector('.mention-popover')
    expect(popover).not.toBeNull()

    unmount()
  })

  it('-t composer: Enter in popover selects @session (index 0) and closes popover', () => {
    const { container, unmount } = mountPanel()

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, '@') })

    // Popover should be open with @session as first item
    expect(container.querySelector('.mention-popover')).not.toBeNull()

    // Press Enter to select active item (index 0 = @session)
    act(() => {
      textarea?.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }),
      )
    })

    // Popover should be closed after selection
    expect(container.querySelector('.mention-popover')).toBeNull()
    // Draft should contain @session
    expect(container.querySelector('textarea')?.value ?? '').toContain('@session')

    unmount()
  })
})

// ── sec-03 integration — echoed script payload renders inert ───────────────

describe('sec-03 integration — echoed script payload renders inert', () => {
  it('-t sec-03: onChat message with <script> payload — no <script> element in DOM', async () => {
    const { container, unmount } = mountPanel({ open: true })

    // Open WS and empty history
    await act(async () => {
      ;(mocks.lastCallbacks?.onOpen as (() => void) | undefined)?.()
    })

    // Deliver message with script content via onChat callback (simulates server echo)
    act(() => {
      const onChat = mocks.lastCallbacks?.onChat as ((m: ChatMessage) => void) | undefined
      onChat?.({
        v: 1,
        id: 'xss-1',
        sessionID: 'sess-001',
        authorID: 'attacker',
        alias: 'attacker',
        content: '<script>alert("xss")</script>',
        ts: Date.now(),
      })
    })

    // Message was received — empty state should be gone
    expect(container.querySelector('.chat-panel__empty')).toBeNull()
    // No <script> element should appear anywhere in the DOM
    expect(container.querySelector('script')).toBeNull()
    // No onerror attributes
    expect(container.innerHTML).not.toContain('onerror')

    unmount()
  })
})

// ═══════════════════════════════════════════════════════════════════════════
// Phase 161 TASK 2 — alias control render + RO-enable + commit + pre-fill
// ═══════════════════════════════════════════════════════════════════════════

/** Set the value of an <input> via the native setter (React synthetic onChange picks it up). */
function setInputValue(el: HTMLInputElement | null, value: string) {
  if (!el) return
  const nativeSetter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    'value',
  )?.set
  if (nativeSetter) {
    nativeSetter.call(el, value)
  } else {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ;(el as any).value = value
  }
  el.dispatchEvent(new Event('input', { bubbles: true }))
}

describe('alias control — Phase 161 ALIAS-UI-01/02 (render + RO-enable + commit + pre-fill)', () => {
  it('-t alias: renders the alias control element in the header', () => {
    const { container, unmount } = mountPanel()
    expect(container.querySelector('.chat-panel__alias')).not.toBeNull()
    unmount()
  })

  it('-t alias: alias label button is NOT disabled when isReadOnly (D-06 exception)', async () => {
    // Mount with capToken; default mockFetch returns [] → /info perms="" → isReadOnly=true
    const { container, unmount } = mountPanel({ capToken: 'test-cap' })

    // Flush all pending effects including the /info fetch
    await act(async () => { /* flush */ })

    const aliasBtn = container.querySelector('.chat-panel__alias-label') as HTMLButtonElement | null
    expect(aliasBtn).not.toBeNull()
    // The alias control must NOT be disabled even though isReadOnly is true (D-06 exception)
    expect(aliasBtn?.disabled).toBe(false)

    unmount()
  })

  it('-t alias: valid commit calls sendAliasSet exactly once with the validated alias', () => {
    const { container, unmount } = mountPanel() // no capToken → isReadOnly=false

    // Open the alias edit input
    const aliasBtn = container.querySelector('.chat-panel__alias-label') as HTMLButtonElement | null
    act(() => { aliasBtn?.click() })

    const input = container.querySelector('.chat-panel__alias-input') as HTMLInputElement | null
    expect(input).not.toBeNull()

    // Set a valid alias
    act(() => { setInputValue(input, 'Alice') })

    // Click the save button (type="button", onClick=handleAliasCommit)
    const saveBtn = container.querySelector('.chat-panel__alias-save') as HTMLButtonElement | null
    act(() => { saveBtn?.click() })

    expect(mocks.mockSendAliasSet).toHaveBeenCalledOnce()
    expect(mocks.mockSendAliasSet).toHaveBeenCalledWith('Alice')

    unmount()
  })

  it('-t alias: invalid alias (33 code points) does NOT call sendAliasSet and shows .chat-panel__alias-error', () => {
    const { container, unmount } = mountPanel()

    // Open the alias edit input
    const aliasBtn = container.querySelector('.chat-panel__alias-label') as HTMLButtonElement | null
    act(() => { aliasBtn?.click() })

    const input = container.querySelector('.chat-panel__alias-input') as HTMLInputElement | null
    // Set an alias that is too long (33 code points)
    act(() => { setInputValue(input, 'a'.repeat(33)) })

    const saveBtn = container.querySelector('.chat-panel__alias-save') as HTMLButtonElement | null
    act(() => { saveBtn?.click() })

    // Should NOT call sendAliasSet
    expect(mocks.mockSendAliasSet).not.toHaveBeenCalled()
    // Should show a visible error element
    expect(container.querySelector('.chat-panel__alias-error')).not.toBeNull()

    unmount()
  })

  it('-t alias: invalid alias with C0 control char does NOT call sendAliasSet', () => {
    const { container, unmount } = mountPanel()

    const aliasBtn = container.querySelector('.chat-panel__alias-label') as HTMLButtonElement | null
    act(() => { aliasBtn?.click() })

    const input = container.querySelector('.chat-panel__alias-input') as HTMLInputElement | null
    act(() => { setInputValue(input, 'bad\x01alias') })

    const saveBtn = container.querySelector('.chat-panel__alias-save') as HTMLButtonElement | null
    act(() => { saveBtn?.click() })

    expect(mocks.mockSendAliasSet).not.toHaveBeenCalled()
    expect(container.querySelector('.chat-panel__alias-error')).not.toBeNull()

    unmount()
  })

  it('-t alias: desktop pre-fill from local:local roster entry', () => {
    const { container, unmount } = mountPanel()

    // Deliver presence with a local:local entry (desktop owner constant)
    act(() => {
      const onPresence = mocks.lastCallbacks?.onPresence as ((p: PresenceEntry[]) => void) | undefined
      onPresence?.([{
        personKey: 'local:local',
        tailnetID: 'local',
        origin: 'local',
        alias: 'DesktopAlias',
        connCount: 1,
      }])
    })

    // The alias label should display the alias from the roster
    const aliasLabel = container.querySelector('.chat-panel__alias-label')
    expect(aliasLabel?.textContent).toContain('DesktopAlias')

    // Click to open edit — input should be pre-filled with the roster alias
    act(() => { (aliasLabel as HTMLButtonElement)?.click() })
    const input = container.querySelector('.chat-panel__alias-input') as HTMLInputElement | null
    expect(input?.value).toBe('DesktopAlias')

    unmount()
  })

  it('-t alias: web pre-fill from onSelf identity (MsgSelf 0x37)', () => {
    const { container, unmount } = mountPanel()

    // Deliver onSelf callback (simulates MsgSelf frame arriving on WS connect)
    act(() => {
      const onSelf = mocks.lastCallbacks?.onSelf as ((personKey: string, alias: string) => void) | undefined
      onSelf?.('node:abc123', 'WebAlias')
    })

    // The alias label should display the onSelf alias
    const aliasLabel = container.querySelector('.chat-panel__alias-label')
    expect(aliasLabel?.textContent).toContain('WebAlias')

    // Click to open edit — input should be pre-filled
    act(() => { (aliasLabel as HTMLButtonElement)?.click() })
    const input = container.querySelector('.chat-panel__alias-input') as HTMLInputElement | null
    expect(input?.value).toBe('WebAlias')

    unmount()
  })

  it('-t alias: header reflects the LIVE roster alias for self after an alias change (desktop)', () => {
    // Phase 161 ALIAS-UI-02 regression: MsgSelf (0x37) is emitted once on connect, so
    // selfIdentity.alias is a frozen connect-time snapshot. After the user changes their
    // alias, only the presence roster re-broadcasts — the header must track the live roster
    // entry for the self personKey, NOT the stale connect-time selfIdentity.alias.
    const { container, unmount } = mountPanel()

    // Connect-time identity (hostname default) arrives via MsgSelf.
    act(() => {
      const onSelf = mocks.lastCallbacks?.onSelf as ((personKey: string, alias: string) => void) | undefined
      onSelf?.('local:local', 'Mac.attlocal.net')
    })
    // Initial roster also carries the hostname default.
    act(() => {
      const onPresence = mocks.lastCallbacks?.onPresence as ((p: PresenceEntry[]) => void) | undefined
      onPresence?.([{ personKey: 'local:local', tailnetID: 'local', origin: 'local', alias: 'Mac.attlocal.net', connCount: 1 }])
    })
    expect(container.querySelector('.chat-panel__alias-label')?.textContent).toContain('Mac.attlocal.net')

    // User changes alias → server re-broadcasts presence (NO new MsgSelf).
    act(() => {
      const onPresence = mocks.lastCallbacks?.onPresence as ((p: PresenceEntry[]) => void) | undefined
      onPresence?.([{ personKey: 'local:local', tailnetID: 'local', origin: 'local', alias: 'Ken (Desktop)', connCount: 1 }])
    })

    // Header must now show the NEW alias, not the frozen connect-time value.
    expect(container.querySelector('.chat-panel__alias-label')?.textContent).toContain('Ken (Desktop)')
    expect(container.querySelector('.chat-panel__alias-label')?.textContent).not.toContain('Mac.attlocal.net')

    unmount()
  })

  it('-t alias: header reflects the LIVE roster alias for self after an alias change (web personKey)', () => {
    const { container, unmount } = mountPanel()

    // Web guest learns its own identity via MsgSelf on connect.
    act(() => {
      const onSelf = mocks.lastCallbacks?.onSelf as ((personKey: string, alias: string) => void) | undefined
      onSelf?.('node:abc123', 'WebOld')
    })
    // Roster re-broadcast after the guest sets a new alias (same personKey).
    act(() => {
      const onPresence = mocks.lastCallbacks?.onPresence as ((p: PresenceEntry[]) => void) | undefined
      onPresence?.([{ personKey: 'node:abc123', tailnetID: 'node', origin: 'remote', alias: 'WebNew', connCount: 1 }])
    })

    expect(container.querySelector('.chat-panel__alias-label')?.textContent).toContain('WebNew')
    expect(container.querySelector('.chat-panel__alias-label')?.textContent).not.toContain('WebOld')

    unmount()
  })

  it('-t alias: global scope communicated via title or aria-label', () => {
    const { container, unmount } = mountPanel()

    const aliasEl = container.querySelector('.chat-panel__alias')
    expect(aliasEl).not.toBeNull()
    // title or aria-label must convey that this is a global (not per-session) display name
    const titleText = aliasEl?.getAttribute('title') ?? ''
    const ariaLabel = aliasEl?.getAttribute('aria-label') ?? ''
    const combined = (titleText + ' ' + ariaLabel).toLowerCase()
    expect(combined).toMatch(/global|all session/i)

    unmount()
  })
})

// ═══════════════════════════════════════════════════════════════════════════
// Phase 163-02 — ROCHAT-01 / ROCHAT-02: RO can post chat; inject still gated
//
// Default mockFetch returns [] for history; capToken triggers /info fetch which
// also resolves to [] → info.perms undefined → isReadOnly stays true (fail-safe).
// await act(async () => {}) flushes all pending microtasks + useEffect chains.
// ═══════════════════════════════════════════════════════════════════════════

describe('ROCHAT-01/02 — RO chat-send enabled; inject gesture stays gated', () => {
  it('-t rochat-01: RO viewer Send button is NOT disabled (ROCHAT-01)', async () => {
    // capToken → /info returns [] → isReadOnly=true (fail-safe path)
    const { container, unmount } = mountPanel({ capToken: 'test-ro-cap' })
    // Flush /info fetch promise
    await act(async () => { /* flush */ })

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, 'hello from ro') })

    const sendBtn = container.querySelector('[data-chat-send]') as HTMLButtonElement | null
    expect(sendBtn).not.toBeNull()
    // ROCHAT-01: Send button must NOT be disabled for RO clients
    expect(sendBtn?.disabled).toBe(false)

    unmount()
  })

  it('-t rochat-01: RO viewer clicking Send calls sendChat with the draft (ROCHAT-01)', async () => {
    const { container, unmount } = mountPanel({ capToken: 'test-ro-cap' })
    await act(async () => { /* flush */ })

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, 'ro click send') })

    const sendBtn = container.querySelector('[data-chat-send]') as HTMLButtonElement | null
    act(() => { sendBtn?.click() })

    expect(mocks.mockSendChat).toHaveBeenCalledOnce()
    expect(mocks.mockSendChat).toHaveBeenCalledWith('ro click send')

    unmount()
  })

  it('-t rochat-01: RO viewer pressing Enter calls sendChat and clears textarea (ROCHAT-01)', async () => {
    const { container, unmount } = mountPanel({ capToken: 'test-ro-cap' })
    await act(async () => { /* flush */ })

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, 'ro enter send') })

    act(() => {
      textarea?.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }),
      )
    })

    expect(mocks.mockSendChat).toHaveBeenCalledOnce()
    expect(mocks.mockSendChat).toHaveBeenCalledWith('ro enter send')
    // Textarea must be cleared after send
    expect(container.querySelector('textarea')?.value ?? '').toBe('')

    unmount()
  })

  it('-t rochat-01: no "Read only" label renders when isReadOnly (ROCHAT-01)', async () => {
    const { container, unmount } = mountPanel({ capToken: 'test-ro-cap' })
    await act(async () => { /* flush */ })

    // The blanket "Read only" label was removed in Phase 163-02
    expect(container.textContent).not.toContain('Read only')

    unmount()
  })

  it('-t rochat-02: RO viewer press-and-hold does NOT call sendSessionInject (ROCHAT-02 inject gate)', async () => {
    vi.useFakeTimers()
    const { container, unmount } = mountPanel({ capToken: 'test-ro-cap' })
    // Flush /info fetch → isReadOnly=true
    await act(async () => { /* flush */ })

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement | null
    act(() => { setTextareaValue(textarea, '@session inject attempt') })

    // Drive the press-and-hold to completion (>= 600ms)
    const injectBtn = container.querySelector('.chat-composer__inject-btn')
    act(() => {
      injectBtn?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerId: 1 }))
    })
    act(() => { vi.advanceTimersByTime(600) })

    // RO inject must NOT fire — handleInjectPointerDown returns early when isReadOnly
    expect(mocks.mockSendSessionInject).not.toHaveBeenCalled()

    vi.useRealTimers()
    unmount()
  })
})

// ═══════════════════════════════════════════════════════════════════════════
// Phase 164-02 — CHAT-LAYOUT-02: clampChatWidth helper + localStorage persistence
// ═══════════════════════════════════════════════════════════════════════════

describe('clampChatWidth — width clamp helper (CHAT-LAYOUT-02)', () => {
  it('clamps below-min input up to CHAT_WIDTH_MIN', () => {
    expect(clampChatWidth(100)).toBe(CHAT_WIDTH_MIN)
  })

  it('clamps above-max input down to CHAT_WIDTH_MAX', () => {
    expect(clampChatWidth(9999)).toBe(CHAT_WIDTH_MAX)
  })

  it('passes through an in-range value as integer (coercion)', () => {
    expect(clampChatWidth(420)).toBe(420)
  })

  it('returns CHAT_WIDTH_DEFAULT for NaN input (non-finite fallback)', () => {
    expect(clampChatWidth(NaN)).toBe(CHAT_WIDTH_DEFAULT)
  })

  it('returns CHAT_WIDTH_DEFAULT for Infinity input (non-finite fallback)', () => {
    expect(clampChatWidth(Infinity)).toBe(CHAT_WIDTH_DEFAULT)
  })

  it('returns CHAT_WIDTH_DEFAULT for -Infinity input (non-finite fallback)', () => {
    expect(clampChatWidth(-Infinity)).toBe(CHAT_WIDTH_DEFAULT)
  })

  it('CHAT_WIDTH_MIN < CHAT_WIDTH_MAX (bounds are finite and ordered)', () => {
    expect(CHAT_WIDTH_MIN).toBeGreaterThan(0)
    expect(CHAT_WIDTH_MAX).toBeGreaterThan(CHAT_WIDTH_MIN)
  })
})

// ── ChatPanel localStorage width persistence on mount ─────────────────────

describe('ChatPanel — localStorage width persistence on mount (CHAT-LAYOUT-02 D-04)', () => {
  beforeEach(() => {
    localStorage.clear()
    // Reset any --chat-panel-width set by previous test
    document.documentElement.style.removeProperty('--chat-panel-width')
  })

  afterEach(() => {
    localStorage.clear()
    document.documentElement.style.removeProperty('--chat-panel-width')
  })

  it('initializes to stored valid value on mount (D-04)', () => {
    localStorage.setItem('agenthub.chatPanelWidth', '500')
    const { unmount } = mountPanel()
    expect(document.documentElement.style.getPropertyValue('--chat-panel-width')).toBe('500px')
    unmount()
  })

  it('clamps an out-of-range stored value on mount (tampered localStorage — T-164-11)', () => {
    localStorage.setItem('agenthub.chatPanelWidth', '5000')
    const { unmount } = mountPanel()
    expect(document.documentElement.style.getPropertyValue('--chat-panel-width')).toBe(`${CHAT_WIDTH_MAX}px`)
    unmount()
  })

  it('falls back to CHAT_WIDTH_DEFAULT when no stored value exists', () => {
    // localStorage has no 'agenthub.chatPanelWidth' entry
    const { unmount } = mountPanel()
    expect(document.documentElement.style.getPropertyValue('--chat-panel-width')).toBe(`${CHAT_WIDTH_DEFAULT}px`)
    unmount()
  })

  it('falls back to CHAT_WIDTH_DEFAULT for a non-numeric stored value (NaN path)', () => {
    localStorage.setItem('agenthub.chatPanelWidth', 'not-a-number')
    const { unmount } = mountPanel()
    expect(document.documentElement.style.getPropertyValue('--chat-panel-width')).toBe(`${CHAT_WIDTH_DEFAULT}px`)
    unmount()
  })
})
