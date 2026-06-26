/**
 * ChatPanel.tsx — Phase 154-05 (GREEN implementation)
 *
 * Right slide-over drawer that owns its own RelayClient WebSocket subscription
 * (separate from TerminalPanel's — Pattern 2, D-02 overlay mode), renders the
 * virtualized message thread with sticky day separators, loads late-join scrollback,
 * shows empty/loading/error states, and computes unread accrual (D-09).
 *
 * Plan 154-05 deliverables:
 *   - buildItems(messages): groups by day + marks consecutive same-author runs
 *   - mergeWithDedup(current, incoming, seenIds): WS+history deduplication
 *   - accrueUnread(prev, message, currentUserTailnetID): pure unread transition
 *   - getRowStyle(isActiveSeparator, start): sticky vs absolute positioning
 *   - loadChatHistory(relayPort, sessionId): GET /api/chat/{id}/history
 *   - ChatPanel component: subscription, history, virtualizer, states, unread
 *
 * Window focus tracking uses window focus/blur events (Pitfall 8, WKWebView).
 * See RESEARCH.md for why the alternative is avoided.
 *
 * The composer (send path) and HubInteractiveModal integration are added in plan 154-06.
 * The `open` prop drives the CSS translateX slide; ChatPanel stays mounted while the
 * session modal is open so unread accrues while the drawer is closed (D-09).
 */
import React, {
  useRef,
  useEffect,
  useState,
  useMemo,
  useCallback,
} from 'react'
import { useVirtualizer, defaultRangeExtractor } from '@tanstack/react-virtual'
import { RelayClient } from '../../lib/relayClient'
import type { ChatMessage, PresenceEntry } from '../../lib/relayClient'
import { ChatMessage as ChatMessageComponent } from './ChatMessage'
import { ChatDaySeparator, formatDaySeparator } from './ChatDaySeparator'
import {
  ChatBubbleLeftIcon,
  ExclamationCircleIcon,
} from '@heroicons/react/24/outline'

// ── Types ──────────────────────────────────────────────────────────────────

/** Item union fed to the virtualizer. Separators and messages share the same flat array. */
export type VirtualItem =
  | { type: 'message'; message: ChatMessage; isConsecutive: boolean }
  | { type: 'separator'; label: string; isoDate: string }

/** Unread badge state: total count + mention flag. */
export interface UnreadState {
  count: number
  hasMention: boolean
}

/** Loading phases for the thread region. */
type LoadingPhase = 'connecting' | 'loading-history' | 'ready' | 'error'

// ── Props ──────────────────────────────────────────────────────────────────

export interface ChatPanelProps {
  /** Session to subscribe to. */
  sessionId: string
  /** Local relay port (127.0.0.1:{relayPort}). */
  relayPort: number
  /** Whether the drawer is currently open (controls CSS translateX). */
  open: boolean
  /**
   * TailnetID of the current user.
   * Defaults to "local" for the desktop owner (RESEARCH Open Question 3).
   * Phase 155 passes the real Tailscale node pubkey for web-share viewers.
   */
  currentUserTailnetID?: string
  /** Called whenever the unread count or mention flag changes. */
  onUnreadChange?: (count: number, hasMention: boolean) => void
}

// ── Pure helpers (exported for unit tests) ─────────────────────────────────

/** Returns the YYYY-M-D local-time key used to detect day boundaries. */
function getDayKey(tsMs: number): string {
  const d = new Date(tsMs)
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
}

/**
 * Derive a hue angle (0–359) from a tailnetID string via polynomial hash.
 * Used for supplementary avatar colour in the presence roster.
 */
function tailnetIdToHue(tailnetID: string): number {
  let hash = 0
  for (let i = 0; i < tailnetID.length; i++) {
    hash = (hash * 31 + tailnetID.charCodeAt(i)) >>> 0
  }
  return hash % 360
}

/**
 * Build the flat VirtualItem array from a sorted message list.
 *
 * - Inserts a separator item before each new calendar day.
 * - Marks same-author messages within 5 min of the previous as `isConsecutive`.
 * - Resets consecutive tracking at every day boundary.
 *
 * Exported for unit tests — pure, no DOM access.
 */
export function buildItems(messages: ChatMessage[]): VirtualItem[] {
  const items: VirtualItem[] = []
  let lastDay: string | null = null
  let lastAuthorID: string | null = null
  let lastTs: number | null = null

  for (const msg of messages) {
    const dayKey = getDayKey(msg.ts)

    if (dayKey !== lastDay) {
      // New calendar day: insert separator and reset consecutive tracking
      items.push({
        type: 'separator',
        label: formatDaySeparator(msg.ts),
        isoDate: dayKey,
      })
      lastDay = dayKey
      lastAuthorID = null
      lastTs = null
    }

    const isConsecutive =
      lastAuthorID === msg.authorID &&
      lastTs !== null &&
      msg.ts - lastTs < 5 * 60 * 1000 // < 5 minutes

    items.push({ type: 'message', message: msg, isConsecutive })
    lastAuthorID = msg.authorID
    lastTs = msg.ts
  }

  return items
}

/**
 * Merge `incoming` messages into `current`, skipping any whose id is already in `seenIds`.
 * Mutates `seenIds` as it goes (caller owns the Set via useRef).
 *
 * Returns the same `current` reference when nothing new is added (avoids re-render).
 * Exported for unit tests — pure aside from the seenIds mutation.
 */
export function mergeWithDedup(
  current: ChatMessage[],
  incoming: ChatMessage[],
  seenIds: Set<string>,
): ChatMessage[] {
  const additions: ChatMessage[] = []
  for (const msg of incoming) {
    if (!seenIds.has(msg.id)) {
      seenIds.add(msg.id)
      additions.push(msg)
    }
  }
  if (additions.length === 0) return current
  return [...current, ...additions]
}

/**
 * Pure unread-accrual transition: given the previous state, an incoming message,
 * and the current user's TailnetID, return the next UnreadState.
 *
 * Callers decide WHEN to call this (only when !open || !windowFocused).
 * Clearing is a separate concern (useEffect on open+windowFocused).
 * Exported for unit tests — pure, no side effects.
 */
export function accrueUnread(
  prev: UnreadState,
  message: ChatMessage,
  currentUserTailnetID: string,
): UnreadState {
  const hasMention =
    prev.hasMention ||
    !!(message.mentions?.includes(currentUserTailnetID))
  return { count: prev.count + 1, hasMention }
}

/**
 * Return the CSS style object for a virtualizer row.
 *
 * Active sticky separator (CHAT-04):
 *   position:sticky, top:0, zIndex:2 — NO transform.
 *   CSS transform creates a new containing block which breaks position:sticky
 *   (Pitfall 1 in RESEARCH.md). The separator must NOT receive a translateY.
 *
 * All other rows:
 *   position:absolute + translateY (standard tanstack/react-virtual pattern).
 *
 * Exported for unit tests — pure.
 */
export function getRowStyle(
  isActiveSeparator: boolean,
  start: number,
): React.CSSProperties {
  if (isActiveSeparator) {
    // NO transform — Pitfall 1: transform overrides position:sticky
    return { position: 'sticky', top: 0, zIndex: 2 }
  }
  return {
    position: 'absolute',
    top: 0,
    left: 0,
    transform: `translateY(${start}px)`,
  }
}

/**
 * Fetch late-join chat history from the local relay.
 * Called AFTER the WebSocket is opened (Pitfall 5: WS-first to avoid gap).
 * Returns empty array on any failure — live WS messages continue regardless.
 */
export async function loadChatHistory(
  relayPort: number,
  sessionId: string,
): Promise<ChatMessage[]> {
  try {
    const resp = await fetch(
      `http://127.0.0.1:${relayPort}/api/chat/${sessionId}/history`,
    )
    if (!resp.ok) return []
    return resp.json() as Promise<ChatMessage[]>
  } catch {
    return []
  }
}

// ── Component ──────────────────────────────────────────────────────────────

/**
 * ChatPanel — the receive-and-display half of the chat UI.
 *
 * Opens its own RelayClient WebSocket subscription (separate from TerminalPanel's).
 * Maintains message state, presence roster, typing indicators, and unread count.
 * The send composer and HubInteractiveModal integration are added in plan 154-06.
 *
 * D-02: The drawer is absolutely positioned over the right edge of the terminal
 * (overlay mode). The terminal width never changes — no PTY resize is triggered.
 */
export function ChatPanel({
  sessionId,
  relayPort,
  open,
  currentUserTailnetID = 'local',
  onUnreadChange,
}: ChatPanelProps): React.ReactElement {
  // ── Core state ─────────────────────────────────────────────────────────
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [participants, setParticipants] = useState<PresenceEntry[]>([])
  const [typingEntries, setTypingEntries] = useState<{ key: string; alias: string }[]>([])
  const [phase, setPhase] = useState<LoadingPhase>('connecting')
  const [unread, setUnread] = useState<UnreadState>({ count: 0, hasMention: false })
  const [windowFocused, setWindowFocused] = useState(document.hasFocus())

  // ── Refs ────────────────────────────────────────────────────────────────
  /** Stable Set of seen message IDs — prevents WS+history duplicates. */
  const seenIdsRef = useRef(new Set<string>())
  /** Scroll container for the virtualizer. */
  const parentRef = useRef<HTMLDivElement>(null)
  /**
   * Current open/focus/tailnetID captured in a ref so the onChat callback
   * (in the long-lived RelayClient useEffect) reads non-stale values.
   */
  const liveRef = useRef({ open, windowFocused, currentUserTailnetID })
  liveRef.current = { open, windowFocused, currentUserTailnetID }

  // ── Window focus/blur tracking (focus/blur events — Pitfall 8, WKWebView) ─
  useEffect(() => {
    const onFocus = () => setWindowFocused(true)
    const onBlur = () => setWindowFocused(false)
    window.addEventListener('focus', onFocus)
    window.addEventListener('blur', onBlur)
    return () => {
      window.removeEventListener('focus', onFocus)
      window.removeEventListener('blur', onBlur)
    }
  }, [])

  // ── Clear unread when drawer is open AND window is focused (D-09) ───────
  useEffect(() => {
    if (open && windowFocused) {
      setUnread({ count: 0, hasMention: false })
    }
  }, [open, windowFocused])

  // ── Report unread changes upward via onUnreadChange ─────────────────────
  const onUnreadChangeRef = useRef(onUnreadChange)
  onUnreadChangeRef.current = onUnreadChange
  useEffect(() => {
    onUnreadChangeRef.current?.(unread.count, unread.hasMention)
  }, [unread])

  // ── RelayClient subscription (Pattern 2 — separate from TerminalPanel) ─
  const handleChat = useCallback((message: ChatMessage) => {
    setMessages(prev => mergeWithDedup(prev, [message], seenIdsRef.current))
    const { open: isOpen, windowFocused: isFocused, currentUserTailnetID: tailnetID } = liveRef.current
    if (!isOpen || !isFocused) {
      setUnread(prev => accrueUnread(prev, message, tailnetID))
    }
  }, [])

  useEffect(() => {
    // Reset state on session change
    setPhase('connecting')
    setMessages([])
    setParticipants([])
    setTypingEntries([])
    setUnread({ count: 0, hasMention: false })
    seenIdsRef.current = new Set()

    const client = new RelayClient(relayPort, sessionId, {
      onOutput: () => {}, // ChatPanel discards PTY output — only onChat matters
      onChat: handleChat,
      onPresence: (p) => setParticipants(p),
      onTyping: (personKey, alias, typing) => {
        setTypingEntries(prev =>
          typing
            ? prev.some(e => e.key === personKey)
              ? prev
              : [...prev, { key: personKey, alias }]
            : prev.filter(e => e.key !== personKey),
        )
      },
      onInjectError: (reason) => {
        console.debug('[ChatPanel] inject error:', reason)
      },
      onOpen: () => {
        // WS connected — fetch history AFTER WS opens to avoid gap (Pitfall 5)
        setPhase('loading-history')
        loadChatHistory(relayPort, sessionId)
          .then(history => {
            setMessages(prev => mergeWithDedup(prev, history, seenIdsRef.current))
            setPhase('ready')
          })
          .catch(() => {
            // History unavailable — live-only mode still works
            setPhase('ready')
          })
      },
      onClose: () => {
        setPhase('error')
      },
    })

    return () => { client.close() }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId, relayPort]) // handleChat is stable (useCallback with no deps)

  // ── Virtualizer setup ───────────────────────────────────────────────────
  const items = useMemo(() => buildItems(messages), [messages])

  /** Indices of separator items in the flat items array. */
  const separatorIndices = useMemo(
    () =>
      items.reduce<number[]>(
        (acc, item, i) => (item.type === 'separator' ? [...acc, i] : acc),
        [],
      ),
    [items],
  )

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: (i) => (items[i]?.type === 'separator' ? 28 : 60),
    rangeExtractor: (range) => {
      // Always include the active sticky separator (highest sep index ≤ startIndex)
      const activeSep =
        [...separatorIndices].reverse().find(i => i <= range.startIndex) ??
        separatorIndices[0]
      const range_ = defaultRangeExtractor(range)
      const set = new Set(range_)
      if (activeSep !== undefined) set.add(activeSep)
      return [...set].sort((a, b) => a - b)
    },
  })

  /** The active sticky separator index for this scroll position. */
  const activeSepIndex = useMemo(() => {
    const range = virtualizer.range
    if (!range || separatorIndices.length === 0) return separatorIndices[0]
    return (
      [...separatorIndices].reverse().find(i => i <= range.startIndex) ??
      separatorIndices[0]
    )
  }, [separatorIndices, virtualizer.range])

  // ── Typing indicator text ───────────────────────────────────────────────
  const typingText = useMemo(() => {
    if (typingEntries.length === 0) return ''
    if (typingEntries.length === 1) return `${typingEntries[0].alias} is typing…`
    if (typingEntries.length === 2)
      return `${typingEntries[0].alias} and ${typingEntries[1].alias} are typing…`
    return `${typingEntries.length} people are typing…`
  }, [typingEntries])

  // ── Render ─────────────────────────────────────────────────────────────
  return (
    <div
      className={`chat-panel${open ? ' chat-panel--open' : ''}`}
      aria-label="Chat"
      aria-hidden={!open}
      data-testid="chat-panel"
    >
      {/* ── Header: title + presence roster ──────────────────────────── */}
      <div className="chat-panel__header">
        <span className="chat-panel__title">Chat</span>
        {/* Presence roster: up to 3 avatars + overflow count */}
        <div
          className="chat-panel__roster"
          aria-label={`${participants.length} participants connected`}
        >
          {participants.slice(0, 3).map(p => (
            <div
              key={p.personKey}
              className="chat-panel__roster-avatar"
              style={{ background: `hsl(${tailnetIdToHue(p.tailnetID)}, 55%, 45%)` }}
              title={p.alias}
              aria-hidden="true"
            >
              {p.alias[0]?.toUpperCase() ?? '?'}
            </div>
          ))}
          {participants.length > 3 && (
            <span className="chat-panel__roster-overflow">
              +{participants.length - 3}
            </span>
          )}
        </div>
      </div>

      {/* ── Thread: scroll container for the virtualizer ─────────────── */}
      <div className="chat-panel__thread" ref={parentRef}>
        {/* Connecting (WS not yet open) */}
        {phase === 'connecting' && (
          <div className="chat-panel__loading" aria-live="polite">
            Connecting…
          </div>
        )}

        {/* Loading history (WS open, history fetch in progress, no messages yet) */}
        {phase === 'loading-history' && items.length === 0 && (
          <div className="chat-panel__loading" aria-live="polite">
            Loading history…
          </div>
        )}

        {/* Error (WS closed unexpectedly) */}
        {phase === 'error' && (
          <div className="chat-panel__error" role="alert">
            <ExclamationCircleIcon
              className="chat-panel__error-icon"
              aria-hidden="true"
            />
            Connection lost — reconnecting… or try refreshing the session.
          </div>
        )}

        {/* Empty (ready, no messages) */}
        {phase === 'ready' && items.length === 0 && (
          <div className="chat-panel__empty">
            <ChatBubbleLeftIcon
              className="chat-panel__empty-icon"
              aria-hidden="true"
            />
            <p className="chat-panel__empty-heading">No messages yet</p>
            <p className="chat-panel__empty-body">
              Send a message to start the conversation.
            </p>
          </div>
        )}

        {/* Virtualizer render loop (CHAT-04: sticky day separators) */}
        {items.length > 0 && (
          <div
            style={{
              height: `${virtualizer.getTotalSize()}px`,
              width: '100%',
              position: 'relative',
            }}
          >
            {virtualizer.getVirtualItems().map(vRow => {
              const item = items[vRow.index]
              const isActiveSep =
                item.type === 'separator' && vRow.index === activeSepIndex

              return (
                <div
                  key={vRow.key}
                  ref={virtualizer.measureElement}
                  data-index={vRow.index}
                  // getRowStyle: sticky for active separator, absolute+translateY for all others
                  // See RESEARCH Pitfall 1 — transform overrides position:sticky
                  style={getRowStyle(isActiveSep, vRow.start)}
                >
                  {item.type === 'separator' ? (
                    <ChatDaySeparator label={item.label} />
                  ) : (
                    <ChatMessageComponent
                      message={item.message}
                      isFirstInGroup={!item.isConsecutive}
                      isMentionOfMe={
                        !!(item.message.mentions?.includes(currentUserTailnetID))
                      }
                    />
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* ── Typing indicator slot (collapsible) ──────────────────────── */}
      <div
        className="chat-panel__typing"
        aria-live="polite"
        style={{
          maxHeight: typingText ? '24px' : '0',
          overflow: 'hidden',
          transition: 'max-height 150ms',
        }}
      >
        {typingText && (
          <span className="chat-panel__typing-text">{typingText}</span>
        )}
      </div>

      {/* ── Composer slot — filled in plan 154-06 ────────────────────── */}
      <div className="chat-panel__composer-slot" />
    </div>
  )
}

export default ChatPanel
