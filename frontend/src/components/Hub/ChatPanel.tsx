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
import TextareaAutosize from 'react-textarea-autosize'
import { useVirtualizer, defaultRangeExtractor } from '@tanstack/react-virtual'
import { RelayClient } from '../../lib/relayClient'
import type { ChatMessage, PresenceEntry } from '../../lib/relayClient'
import { ChatMessage as ChatMessageComponent } from './ChatMessage'
import { ChatDaySeparator, formatDaySeparator } from './ChatDaySeparator'
import { MentionPopover } from './MentionPopover'
import {
  ChatBubbleLeftIcon,
  ExclamationCircleIcon,
  PaperAirplaneIcon,
  CommandLineIcon,
  ArrowDownTrayIcon,
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
  // ── Phase 155 additions (all optional → non-breaking for desktop) ──────────
  /** WS URL override for web-share (wss://host/sessions/{id}/ws?cap=…). When
   *  absent, RelayClient builds ws://127.0.0.1:{relayPort}/… (desktop). */
  wsURL?: string
  /** HTTP base URL for history/export API calls (web-share: window.location.origin;
   *  desktop: falls back to http://127.0.0.1:{relayPort}). */
  apiBaseURL?: string
  /** Cap token forwarded opaquely as ?cap= on all web-surface API calls. Never
   *  decoded client-side (signing key is server-only). */
  capToken?: string
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
 * Client mirror of Go ValidateAlias (internal/relay/protocol.go:ValidateAlias).
 *
 * Rules — must match exactly:
 *   1. Trim leading/trailing whitespace.
 *   2. Reject empty result.
 *   3. Count code points with Array.from (NOT String.length) — mirrors Go []rune cast.
 *   4. Reject (not truncate) if code-point count exceeds 32.
 *   5. Reject any code point < 0x20 (C0) or 0x7F..0x9F (C1).
 *
 * Defense-in-depth: server ValidateAlias is authoritative; server sends NO NAK on
 * rejection, so this is the ONLY user feedback path for invalid input (RESEARCH Pitfall 4).
 * Exported for unit tests — pure, no side effects.
 */
export function validateAlias(raw: string): string | null {
  const trimmed = raw.trim()
  if (trimmed === '') return null
  const runes = Array.from(trimmed) // code points, mirrors Go []rune — NOT .length
  if (runes.length > 32) return null // reject — do NOT truncate
  for (const ch of runes) {
    const cp = ch.codePointAt(0)!
    if (cp < 0x20 || (cp >= 0x7f && cp <= 0x9f)) return null // C0 / C1
  }
  return trimmed
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
 * Fetch late-join chat history from the local relay or webserver.
 * Called AFTER the WebSocket is opened (Pitfall 5: WS-first to avoid gap).
 * Returns empty array on any failure — live WS messages continue regardless.
 *
 * Phase 155: opts.apiBaseURL / opts.capToken enable web-share surface.
 * Without ?cap= on web-share, the webserver returns 401 → error state fires
 * (Pitfall 2 — cap param is mandatory on web-share).
 */
export async function loadChatHistory(
  relayPort: number,
  sessionId: string,
  opts?: { apiBaseURL?: string; capToken?: string },
): Promise<ChatMessage[]> {
  try {
    const base = opts?.apiBaseURL ?? `http://127.0.0.1:${relayPort}`
    const capParam = opts?.capToken ? `?cap=${encodeURIComponent(opts.capToken)}` : ''
    const resp = await fetch(`${base}/api/chat/${sessionId}/history${capParam}`)
    if (!resp.ok) return []
    return resp.json() as Promise<ChatMessage[]>
  } catch {
    return []
  }
}

/**
 * Download a URL via a hidden anchor element (Content-Disposition download).
 * The server sets `Content-Disposition: attachment; filename="chat-{id}.md"`.
 */
function triggerExport(url: string): void {
  const a = document.createElement('a')
  a.href = url
  a.download = ''
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
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
  wsURL,
  apiBaseURL,
  capToken,
}: ChatPanelProps): React.ReactElement {
  // ── Core state ─────────────────────────────────────────────────────────
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [participants, setParticipants] = useState<PresenceEntry[]>([])
  const [typingEntries, setTypingEntries] = useState<{ key: string; alias: string }[]>([])
  const [phase, setPhase] = useState<LoadingPhase>('connecting')
  const [unread, setUnread] = useState<UnreadState>({ count: 0, hasMention: false })
  const [windowFocused, setWindowFocused] = useState(document.hasFocus())

  // ── Composer state (plan 154-06) ────────────────────────────────────────
  const [draft, setDraft] = useState('')
  const [mentionOpen, setMentionOpen] = useState(false)
  const [mentionFilter, setMentionFilter] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const [isHolding, setIsHolding] = useState(false)

  // ── Phase 155 PARITY-01 SC-3: read-only cap suppression (defense-in-depth) ─
  // Fail-safe default: true. Resolved from server's /info perms on web-share.
  // Desktop (no capToken) is always writable — set to false immediately.
  // Server gating is authoritative (HandleChatSend/HandleInject); this is UX only.
  const [isReadOnly, setIsReadOnly] = useState(true)

  // ── Phase 161 ALIAS-UI-01: alias control state ───────────────────────────
  /** Self-identity set from MsgSelf (0x37) frame on WS connect. Used for web pre-fill. */
  const [selfIdentity, setSelfIdentity] = useState<{ personKey: string; alias: string } | null>(null)
  /** Whether the alias edit input is currently active. */
  const [aliasEditing, setAliasEditing] = useState(false)
  /** Alias draft value while editing. */
  const [aliasDraft, setAliasDraft] = useState('')
  /** Client-side validation error (shown instead of calling sendAliasSet). */
  const [aliasError, setAliasError] = useState('')

  // ── Refs ────────────────────────────────────────────────────────────────
  /** Stable Set of seen message IDs — prevents WS+history duplicates. */
  const seenIdsRef = useRef(new Set<string>())
  /** Scroll container for the virtualizer. */
  const parentRef = useRef<HTMLDivElement>(null)
  /** Composer textarea element. */
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  /** Relay client ref — lets composer handlers send frames (plan 154-06). */
  const clientRef = useRef<RelayClient | null>(null)
  /** Stable draft ref for the inject timer closure (D-08). */
  const draftRef = useRef('')
  draftRef.current = draft  // inline sync on every render (liveRef pattern)
  /** Press-and-hold timer ref (D-08). */
  const holdTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  /** Previous items.length — used to detect new-message arrivals for auto-scroll. */
  const prevItemCountRef = useRef(0)
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

  // ── Phase 155 PARITY-01 SC-3: resolve RO status from server /info ────────
  // Precedent: web/assets/terminal.js:73-105 — same endpoint/perms logic.
  // Desktop (no capToken): always RW. Web-share: fetch once on mount.
  // NEVER JWT-decode client-side — server's /info response is the authority.
  useEffect(() => {
    if (!capToken) {
      // Desktop owner: always writable
      setIsReadOnly(false)
      return
    }
    // Web-share: resolve from server's /info perms, fail-safe is read-only (default state)
    const base = apiBaseURL ?? `http://127.0.0.1:${relayPort}`
    fetch(`${base}/api/sessions/${encodeURIComponent(sessionId)}/info?cap=${encodeURIComponent(capToken)}`)
      .then(r => r.ok ? r.json() : Promise.reject(r))
      .then((info: { perms?: string }) => {
        // Whole-token membership — matches capability.HasPerm semantics (no substring match)
        const perms = (info?.perms ?? '').split(',').map((s: string) => s.trim())
        setIsReadOnly(!perms.includes('write'))
      })
      .catch(() => {
        // Any fetch error → leave isReadOnly=true (fail-safe)
      })
  }, [sessionId, capToken, apiBaseURL, relayPort])

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

  // ── Hold timer cleanup on unmount ───────────────────────────────────────
  useEffect(() => {
    return () => {
      if (holdTimerRef.current) {
        clearTimeout(holdTimerRef.current)
      }
    }
  }, [])

  // ── Phase 155: Export URL builder (component-scoped — captures props) ───
  function buildExportURL(): string {
    const base = apiBaseURL ?? `http://127.0.0.1:${relayPort}`
    const capParam = capToken ? `?cap=${encodeURIComponent(capToken)}` : ''
    return `${base}/api/chat/${sessionId}/export${capParam}`
  }

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
    setDraft('')
    setMentionOpen(false)
    // Phase 161: reset alias control state on session change
    setSelfIdentity(null)
    setAliasEditing(false)
    setAliasError('')
    clientRef.current = null

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
      onSelf: (personKey, alias) => {
        // Phase 161: store self-identity for alias pre-fill on both surfaces
        setSelfIdentity({ personKey, alias })
      },
      onInjectError: (reason) => {
        console.debug('[ChatPanel] inject error:', reason)
      },
      onOpen: () => {
        // WS connected — fetch history AFTER WS opens to avoid gap (Pitfall 5)
        // Phase 155: pass apiBaseURL+capToken so web-share appends ?cap= (Pitfall 2)
        setPhase('loading-history')
        loadChatHistory(relayPort, sessionId, { apiBaseURL, capToken })
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
    }, { wsURL }) // Phase 155: wsURL override for web-share surface
    clientRef.current = client

    return () => {
      client.close()
      clientRef.current = null
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId, relayPort, wsURL, apiBaseURL, capToken]) // handleChat is stable (useCallback with no deps)

  // ── Composer derived values ─────────────────────────────────────────────
  /** True when the draft contains "@session" — switches Send → Inject button. */
  const hasAtSession = draft.includes('@session')

  /** Participants filtered by the mention filter string. */
  const filteredParticipants = useMemo(
    () =>
      mentionFilter
        ? participants.filter(p =>
            p.alias.toLowerCase().includes(mentionFilter.toLowerCase()),
          )
        : participants,
    [participants, mentionFilter],
  )
  const totalOptions = 1 + filteredParticipants.length

  // ── Composer handlers ───────────────────────────────────────────────────

  function handleDraftChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    const text = e.target.value
    setDraft(text)
    draftRef.current = text

    // Detect @ mention trigger: scan from last @ to caret for filter string
    const pos = e.target.selectionStart ?? text.length
    const textBefore = text.slice(0, pos)
    const atIdx = textBefore.lastIndexOf('@')

    if (atIdx >= 0) {
      const fragment = textBefore.slice(atIdx + 1)
      // No whitespace between @ and cursor = valid trigger
      if (!/\s/.test(fragment)) {
        setMentionOpen(true)
        setMentionFilter(fragment)
        setActiveIndex(0)
        return
      }
    }

    if (mentionOpen) {
      setMentionOpen(false)
      setMentionFilter('')
    }
  }

  function handleMentionSelect(alias: string) {
    const pos = textareaRef.current?.selectionStart ?? draft.length
    const textBefore = draft.slice(0, pos)
    const atIdx = textBefore.lastIndexOf('@')
    const beforeAt = draft.slice(0, atIdx)
    const afterCursor = draft.slice(pos)
    const newDraft = `${beforeAt}${alias} ${afterCursor}`
    setDraft(newDraft)
    draftRef.current = newDraft
    setMentionOpen(false)
    setMentionFilter('')
    setActiveIndex(0)
    textareaRef.current?.focus()
  }

  function handleMentionClose() {
    setMentionOpen(false)
    setMentionFilter('')
  }

  function handleTextareaKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (mentionOpen) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActiveIndex(i => (i + 1) % totalOptions)
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActiveIndex(i => (i - 1 + totalOptions) % totalOptions)
        return
      }
      if (e.key === 'Enter') {
        e.preventDefault()
        if (activeIndex === 0) {
          handleMentionSelect('@session')
        } else {
          const p = filteredParticipants[activeIndex - 1]
          if (p) handleMentionSelect(`@${p.alias}`)
        }
        return
      }
      if (e.key === 'Escape' || e.key === 'Tab') {
        if (e.key === 'Escape') e.preventDefault()
        setMentionOpen(false)
        setMentionFilter('')
        return
      }
    }

    // CRITICAL (Pitfall 7 / D-08): Enter ALWAYS routes to chat-send, NEVER to inject.
    // The inject path is ONLY reachable through the completed 600ms press-and-hold.
    // Phase 163-02 ROCHAT-01: RO clients can post chat; isReadOnly guard removed here.
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      const text = draftRef.current
      if (text.trim()) {
        clientRef.current?.sendChat(text)
        setDraft('')
        draftRef.current = ''
        setMentionOpen(false)
      }
    }
    // Shift+Enter: no preventDefault — default textarea behavior inserts newline
  }

  function handleSend() {
    // Phase 163-02 ROCHAT-01: RO clients can post chat; isReadOnly guard removed.
    const text = draftRef.current
    if (text.trim()) {
      clientRef.current?.sendChat(text)
      setDraft('')
      draftRef.current = ''
    }
  }

  // ── Phase 161 ALIAS-UI-01: alias control helpers ─────────────────────────

  /**
   * Current alias for the header + edit pre-fill.
   *
   * MsgSelf (0x37) is emitted only ONCE on WS connect, so selfIdentity.alias is a frozen
   * connect-time snapshot. After the user changes their alias, the server re-broadcasts the
   * presence roster but does NOT re-emit MsgSelf (ALIAS-UI-02). The header must therefore
   * track the LIVE roster entry for the self person — keyed by selfIdentity.personKey, which
   * matches the roster key the server assigns (`local:local` desktop / `tailnetID:web`) —
   * not the stale selfIdentity.alias.
   *
   * Priority:
   *   1. Live presence entry for the self personKey (updates immediately on alias change)
   *   2. selfIdentity.alias — connect-time seed, used before the roster includes self (web pre-fill)
   *   3. Empty string (no identity yet — neutral placeholder)
   */
  const selfPersonKey = selfIdentity?.personKey ?? 'local:local'
  const currentAlias =
    participants.find(p => p.personKey === selfPersonKey)?.alias ??
    selfIdentity?.alias ??
    ''

  /**
   * Commit the alias edit: validate via validateAlias (client mirror of Go ValidateAlias),
   * show a client-side error on null, or call sendAliasSet on success.
   *
   * CRITICAL: NO isReadOnly early-return — the alias control is the D-06 RO exception.
   * Only handleSend/handleInjectPointerDown remain RO-gated. (RESEARCH Pitfall 1)
   */
  function handleAliasCommit() {
    const validated = validateAlias(aliasDraft)
    if (validated === null) {
      setAliasError('Alias must be 1–32 characters with no control characters.')
      return
    }
    setAliasError('')
    clientRef.current?.sendAliasSet(validated)
    setAliasEditing(false)
  }

  function handleInjectPointerDown(e: React.PointerEvent<HTMLButtonElement>) {
    // Phase 155 PARITY-01 SC-3: short-circuit inject gesture when read-only
    if (isReadOnly) return
    // setPointerCapture keeps tracking the pointer even after it leaves the button
    try { e.currentTarget.setPointerCapture(e.pointerId) } catch { /* jsdom fallback */ }
    setIsHolding(true)
    holdTimerRef.current = setTimeout(() => {
      holdTimerRef.current = null
      setIsHolding(false)
      const text = draftRef.current
      if (text.trim()) {
        clientRef.current?.sendSessionInject(text)
        setDraft('')
        draftRef.current = ''
      }
    }, 600)
  }

  function resetInjectHold() {
    if (holdTimerRef.current) {
      clearTimeout(holdTimerRef.current)
      holdTimerRef.current = null
    }
    setIsHolding(false)
  }

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

  // ── Auto-scroll to bottom when new messages arrive ───────────────────────
  // Standard chat UX: pin to newest message on initial history load and on
  // each inbound message when the user is already near the bottom.
  // Uses prevItemCountRef to distinguish first load (0 → N) from incremental
  // updates (N → N+1). The virtualizer is intentionally omitted from deps —
  // it is a fresh object on every render; including it would cause an
  // infinite loop. scrollToIndex is stable across renders (internal useCallback).
  useEffect(() => {
    const newCount = items.length
    if (newCount === 0) {
      prevItemCountRef.current = 0
      return
    }
    const el = parentRef.current
    const isFirstLoad = prevItemCountRef.current === 0
    const isNearBottom =
      !el || el.scrollHeight - el.scrollTop - el.clientHeight < 150
    if (isFirstLoad || isNearBottom) {
      // scrollToIndex with align:'end' scrolls the BOTTOM of the last item
      // into view (not the top), ensuring the newest message is fully visible.
      virtualizer.scrollToIndex(newCount - 1, { align: 'end' })
    }
    prevItemCountRef.current = newCount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [items.length]) // virtualizer intentionally omitted — see comment above

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
      {/* ── Header: title + alias control + presence roster + export ──── */}
      <div className="chat-panel__header">
        <span className="chat-panel__title">Chat</span>
        {/* Phase 161 ALIAS-UI-01: global display-name control.
            Enabled on ALL surfaces (GUI tab, Hub modal, web-share guest).
            NOT disabled when isReadOnly — D-06 exception: alias set ≠ chat send.
            title conveys global scope: "Your global display name across all sessions". */}
        <div
          className="chat-panel__alias"
          title="Your global display name — shown to all chat participants across all sessions"
        >
          {aliasEditing ? (
            <div className="chat-panel__alias-edit">
              <input
                type="text"
                className="chat-panel__alias-input"
                value={aliasDraft}
                onChange={e => setAliasDraft(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter') { e.preventDefault(); handleAliasCommit() }
                  if (e.key === 'Escape') { setAliasEditing(false); setAliasError('') }
                }}
                aria-label="Your global display name"
                // NOT disabled — alias control is the D-06 RO exception
                autoFocus
              />
              <button
                type="button"
                className="chat-panel__alias-save"
                onClick={handleAliasCommit}
                aria-label="Save display name"
              >
                ✓
              </button>
            </div>
          ) : (
            <button
              type="button"
              className="chat-panel__alias-label"
              onClick={() => {
                setAliasDraft(currentAlias)
                setAliasError('')
                setAliasEditing(true)
              }}
              aria-label={`Chatting as ${currentAlias || '(no name set)'}. Click to edit your global display name.`}
            >
              <span className="chat-panel__alias-name">{currentAlias || '(set name)'}</span>
              {' ✏️'}
            </button>
          )}
          {aliasError && (
            <div
              className="chat-panel__alias-error"
              role="alert"
              style={{ fontSize: 11, color: 'var(--hub-text-dim)' }}
            >
              {aliasError}
            </div>
          )}
        </div>
        {/* Presence roster: up to 3 avatars + overflow count.
            chat-presence is the frozen Playwright selector (UI-SPEC §5). */}
        <div
          className="chat-panel__roster chat-presence"
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
        {/* Phase 155 EXPORT-01: Export chat as Markdown (available in both RO and RW modes) */}
        <button
          type="button"
          className="chat-panel__export-btn"
          data-chat-export
          aria-label="Export chat as Markdown"
          title="Export chat as Markdown"
          onClick={() => triggerExport(buildExportURL())}
          style={{
            width: 36,
            height: 36,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            color: 'var(--hub-text-muted)',
            borderRadius: 'var(--hub-radius-sm)',
          }}
        >
          <ArrowDownTrayIcon style={{ width: 18, height: 18 }} aria-hidden="true" />
        </button>
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
      {/* chat-typing is the frozen Playwright selector (UI-SPEC §5). */}
      <div
        className="chat-panel__typing chat-typing"
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

      {/* ── Composer (plan 154-06) ───────────────────────────────────── */}
      <div className="chat-panel__composer">
        {/* Relative wrapper: containing block for MentionPopover (Pattern 6) */}
        <div className="chat-panel__composer-inner">
          {/* MentionPopover — bottom-anchored above the textarea wrapper */}
          {mentionOpen && (
            <MentionPopover
              participants={participants}
              filter={mentionFilter}
              activeIndex={activeIndex}
              onSelect={handleMentionSelect}
              onClose={handleMentionClose}
            />
          )}

          {/* Composer textarea */}
          <TextareaAutosize
            ref={textareaRef}
            className="chat-composer__textarea"
            value={draft}
            onChange={handleDraftChange}
            onKeyDown={handleTextareaKeyDown}
            minRows={1}
            maxRows={6}
            placeholder={
              hasAtSession
                ? '@session — hold Send to inject'
                : 'Message…'
            }
            aria-label="Type a message"
          />

          {/* Send button (normal) / Inject button (@session in draft) */}
          {hasAtSession ? (
            /* CRITICAL (D-08 / Pitfall 7): inject button has NO form association
               and NO keyboard shortcut. The only path to inject is the completed
               600ms press-and-hold. Enter always routes to handleTextareaKeyDown
               → sendChat on a strictly separate code path. */
            <button
              type="button"
              className={`chat-composer__inject-btn${isHolding ? ' chat-composer__inject-btn--holding' : ''}`}
              aria-label="Hold to inject into terminal"
              onPointerDown={handleInjectPointerDown}
              onPointerUp={resetInjectHold}
              onPointerCancel={resetInjectHold}
            >
              <CommandLineIcon className="chat-composer__inject-icon" aria-hidden="true" />
              <span>Inject</span>
            </button>
          ) : (
            <button
              type="button"
              className="chat-composer__send-btn"
              // Phase 163-02 ROCHAT-01: data-chat-send present so Playwright can query it.
              // RO clients can now post chat; only inject remains RO-gated.
              data-chat-send
              aria-label="Send message"
              onClick={handleSend}
              disabled={!draft.trim()}
            >
              <PaperAirplaneIcon className="chat-composer__send-icon" aria-hidden="true" />
              <span>Send</span>
            </button>
          )}
        </div>
        {/* Phase 163-02 ROCHAT-01: Read-only label removed — RO clients are chat participants.
            isReadOnly is retained to gate the inject gesture (handleInjectPointerDown). */}
      </div>
    </div>
  )
}

export default ChatPanel
