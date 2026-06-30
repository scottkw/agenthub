# Phase 154: Desktop Chat UI — Pattern Map

**Mapped:** 2026-06-26
**Files analyzed:** 8 (5 new + 3 modified)
**Analogs found:** 8 / 8

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/Hub/ChatPanel.tsx` | component | event-driven (WS) | `frontend/src/components/TerminalPanel.tsx` | role-match (both own a RelayClient + ResizeObserver lifecycle) |
| `frontend/src/components/Hub/ChatMessage.tsx` | component | transform (render) | `frontend/src/components/Hub/SessionCard.tsx` (ATTN row) | role-match |
| `frontend/src/components/Hub/MentionPopover.tsx` | component | request-response | `frontend/src/components/Hub/SessionCard.tsx` (`hub-card__menu`) | role-match (dropdown keyboard-nav pattern) |
| `frontend/src/components/Hub/ChatDaySeparator.tsx` | component | transform (render) | `frontend/src/components/Hub/SessionCard.tsx` (section labels) | partial-match |
| `frontend/src/components/Hub/ChatBadge.tsx` | component | transform (render) | `frontend/src/components/Hub/SessionCard.tsx` (ATTN badge + `hub-card__badge`) | role-match |
| `frontend/src/components/Hub/HubInteractiveModal.tsx` | component | request-response | itself (existing) | exact — small modification |
| `frontend/src/components/Hub/SessionCard.tsx` | component | transform (render) | itself (existing) | exact — prop addition only |
| `frontend/src/lib/relayClient.ts` | utility | event-driven (WS) | itself (existing) | exact — stub completion |
| `internal/relay/server.go` (read pump) | middleware | event-driven | itself (existing `case MsgSessionInject`) | exact — add parallel case |
| `internal/relay/hub.go` (`HandleChatSend`) | service | CRUD | `HandleInject` (lines 492–551) | exact — same pattern, fewer guards |

---

## Pattern Assignments

### `frontend/src/lib/relayClient.ts` (utility, event-driven — stub completion)

**Analog:** itself (lines 1–211 already read above)

**Existing constants pattern** (lines 1–12):
```typescript
// Binary framing constants matching internal/relay/protocol.go
export const MSG_OUTPUT  = 0x01  // server → client
export const MSG_INPUT   = 0x10  // client → server
// PATTERN: one-liner per constant, hex literal, arrow-comment direction + description
```

**Existing encode frame builder pattern** (lines 63–69 — `encodeTypingFrame`):
```typescript
export function encodeTypingFrame(typing: boolean): Uint8Array {
  const encoded = new TextEncoder().encode(JSON.stringify({ typing }))
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_TYPING
  frame.set(encoded, 1)
  return frame
}
// PATTERN: 1-byte type prefix + JSON body. COPY EXACTLY for encodeChatSendFrame and encodeSessionInjectFrame.
```

**Existing parseServerFrame JSON case pattern** (lines 104–119):
```typescript
case MSG_PRESENCE: {
  try {
    const json = new TextDecoder().decode(data.slice(1))
    const parsed = JSON.parse(json) as { participants?: PresenceEntry[] }
    return { type: 'presence', participants: parsed.participants ?? [] }
  } catch {
    return { type: 'unknown' }
  }
}
// PATTERN: decode slice(1), JSON.parse with try/catch returning { type: 'unknown' } on failure.
// Add MSG_CHAT (0x30) and MSG_INJECT_ERROR (0x36) cases using this exact template.
```

**Existing RelayClientCallbacks interface** (lines 130–134):
```typescript
export interface RelayClientCallbacks {
  onOutput: (data: Uint8Array) => void
  onOpen?: () => void
  onClose?: () => void
}
// PATTERN: required onOutput; optional callbacks with ?.
// Extend: add onPresence?, onTyping?, onChat?, onInjectError? (all optional — backward compat).
```

**Existing ws.onmessage dispatch** (lines 167–173):
```typescript
this.ws.onmessage = (event: MessageEvent) => {
  const frame = parseServerFrame(new Uint8Array(event.data as ArrayBuffer))
  if (frame.type === 'output') {
    callbacks.onOutput(frame.payload)
  }
  // resize frames from server are informational; terminal resize is driven client-side
}
// PATTERN: parseServerFrame → switch on frame.type → callback?.()
// Replace the if-chain with a switch; add cases for 'chat', 'inject_error', 'presence', 'typing'.
```

**New constants to add** (after line 12, following existing comment block style):
```typescript
// MsgChat (0x30) and MsgChatSend (0x31) are Phase 154 dispatch stubs — wired here.
export const MSG_CHAT            = 0x30  // server → client: chat message broadcast
export const MSG_CHAT_SEND       = 0x31  // client → server: post chat message
export const MSG_SESSION_INJECT  = 0x35  // client → server: inject text into PTY
export const MSG_INJECT_ERROR    = 0x36  // server → client: inject rejected (SEC-01 or oversize)
```

**New ChatMessage interface** (mirror of `internal/relay/protocol.go` — CRITICAL: use `alias` not `authorAlias`):
```typescript
export interface ChatMessage {
  v: number            // schema version (always 1)
  id: string
  sessionID: string
  authorID: string     // "local" for desktop owner
  alias: string        // JSON tag is "alias" in Go — DO NOT use authorAlias
  content: string
  mentions?: string[]  // AuthorIDs of mentioned participants
  sessionInject?: boolean
  ts: number           // UNIX milliseconds
}
```

---

### `frontend/src/components/Hub/ChatPanel.tsx` (component, event-driven)

**Analog:** `frontend/src/components/TerminalPanel.tsx` (RelayClient lifecycle)
**Secondary analog:** `frontend/src/components/Hub/HubInteractiveModal.tsx` (modal shell structure)

**RelayClient construction pattern** — copy from TerminalPanel's `clientRef` pattern:
```typescript
// TerminalPanel constructs RelayClient in a useEffect with sessionId dep:
const clientRef = useRef<RelayClient | null>(null)
useEffect(() => {
  if (!isActive) return
  const client = new RelayClient(relayPort, sessionId, {
    onOutput: ...,
    onOpen: ...,
    onClose: ...,
  })
  clientRef.current = client
  return () => { client.close(); clientRef.current = null }
}, [sessionId, relayPort, isActive])
// PATTERN for ChatPanel: same lifecycle but pass onChat, onPresence, onTyping, onInjectError instead.
// ChatPanel opens its OWN RelayClient (separate from TerminalPanel's — D-02 rationale in RESEARCH.md).
```

**WS-first, then HTTP history** (Pitfall 5 avoidance — subscribe before fetch):
```typescript
useEffect(() => {
  // 1. Open WS first (start receiving live frames)
  const client = new RelayClient(relayPort, sessionId, { onChat: handleChat, ... })
  // 2. Fetch history after WS is connecting (deduplicate by id using seenIds Set)
  loadChatHistory(relayPort, sessionId).then(msgs => { /* dedupe + prepend */ })
  return () => client.close()
}, [sessionId, relayPort])
```

**Unread badge state pattern** (D-09 — window focus/blur, NOT visibilitychange — Pitfall 8):
```typescript
// From RESEARCH.md Pattern 8 — copy exactly:
const [windowFocused, setWindowFocused] = useState(document.hasFocus())
useEffect(() => {
  const onFocus = () => setWindowFocused(true)
  const onBlur = () => setWindowFocused(false)
  window.addEventListener('focus', onFocus)
  window.addEventListener('blur', onBlur)
  return () => { window.removeEventListener('focus', onFocus); window.removeEventListener('blur', onBlur) }
}, [])
```

**Press-and-hold gesture** (D-08 — RESEARCH.md Pattern 5):
```typescript
const holdTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
const [isHolding, setIsHolding] = useState(false)

function handlePointerDown(e: React.PointerEvent) {
  e.currentTarget.setPointerCapture(e.pointerId)
  setIsHolding(true)
  holdTimerRef.current = setTimeout(() => { setIsHolding(false); fireInject() }, 600)
}
function handlePointerUp() {
  if (holdTimerRef.current) { clearTimeout(holdTimerRef.current); holdTimerRef.current = null }
  setIsHolding(false)
}
// CRITICAL: Enter in textarea ALWAYS calls encodeChatSendFrame — never fireInject.
// The inject path is ONLY reachable through the completed 600ms hold (D-08 / Pitfall 7).
```

**virtualizer setup** (RESEARCH.md Pattern 3 — sticky day separators):
```typescript
import { useVirtualizer, defaultRangeExtractor } from '@tanstack/react-virtual'

const virtualizer = useVirtualizer({
  count: items.length,
  getScrollElement: () => parentRef.current,
  estimateSize: (i) => items[i].type === 'separator' ? 28 : 60,
  measureElement: (el) => el.getBoundingClientRect().height,
  rangeExtractor: (range) => {
    const activeSep = [...separatorIndices].reverse().find(i => i <= range.startIndex) ?? separatorIndices[0]
    const range_ = defaultRangeExtractor(range)
    const set = new Set(range_)
    if (activeSep !== undefined) set.add(activeSep)
    return [...set].sort((a, b) => a - b)
  }
})
```

**render loop — CRITICAL sticky vs transform** (Pitfall 1 — no transform on active sticky):
```tsx
{virtualizer.getVirtualItems().map((vRow) => {
  const item = items[vRow.index]
  const isActiveSep = item.type === 'separator' && vRow.index === activeSepIndex
  return (
    <div
      key={vRow.key}
      ref={virtualizer.measureElement}
      data-index={vRow.index}
      style={isActiveSep
        ? { position: 'sticky', top: 0, zIndex: 2 }             // NO transform — Pitfall 1
        : { position: 'absolute', top: 0, left: 0,
            transform: `translateY(${vRow.start}px)` }}
    >
      {item.type === 'separator' ? <ChatDaySeparator label={item.label} /> : <ChatMessage ... />}
    </div>
  )
})}
```

---

### `frontend/src/components/Hub/ChatMessage.tsx` (component, transform)

**Analog:** `frontend/src/components/Hub/SessionCard.tsx` — ATTN badge row (lines 446–466) for colorblind-safe multi-signal pattern

**@mention-of-me three-signal pattern** (D-05 — copy the ATTN approach):
```tsx
// From SessionCard ATTN: icon + text label + border color — three independent signals.
// For ChatMessage @mention: accent bar (position:absolute left:0, 3px) + tint + chip.
// Same principle: if CSS color is invisible, bar shape + chip glyph still communicate state.
<div
  className={`chat-msg ${isMentionOfMe ? 'chat-msg--mention' : ''}`}
  role="listitem"
  aria-label={isMentionOfMe ? `${alias} mentioned you: ${content}` : undefined}
>
  {/* Left accent bar — rendered via CSS ::before on .chat-msg--mention */}
  {/* Background tint — rgba(122,162,247,0.08) via .chat-msg--mention class */}
  {isFirstInGroup && (
    <div className="chat-msg__header">
      <div className="chat-msg__avatar" style={{ background: `hsl(${hue}, 55%, 45%)` }}>
        {alias[0]?.toUpperCase()}
      </div>
      <span className="chat-msg__alias">{alias}</span>
      <span className="chat-msg__tailnet-id">({authorID})</span>
      {isMentionOfMe && <span className="chat-msg__you-chip" aria-hidden="true">@</span>}
      <time className="chat-msg__time" title={new Date(ts).toISOString()}>{formatHHMM(ts)}</time>
    </div>
  )}
  <div className={`chat-msg__body ${!isFirstInGroup ? 'chat-msg__body--consecutive' : ''}`}>
    <Markdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeSanitize]}>{content}</Markdown>
  </div>
</div>
```

**SessionInject indicator** (D-06 — system-style row, NOT a normal message):
```tsx
{message.sessionInject && (
  <div className="chat-msg__inject-indicator" aria-label={`Injected into terminal by ${alias}`}>
    <hr className="chat-msg__inject-rule" />
    <span className="chat-msg__inject-caption">
      <CommandLineIcon className="chat-msg__inject-icon" aria-hidden="true" />
      → injected into terminal
    </span>
  </div>
)}
```

**Avatar hue derivation** (Claude's Discretion — deterministic from tailnetID):
```typescript
function tailnetIdToHue(tailnetID: string): number {
  let hash = 0
  for (let i = 0; i < tailnetID.length; i++) {
    hash = (hash * 31 + tailnetID.charCodeAt(i)) >>> 0
  }
  return hash % 360
}
// background: hsl(${tailnetIdToHue(authorID)}, 55%, 45%)
// foreground: white — alias text is always present (avatar is supplementary)
```

**Safe Markdown pattern** (SEC-03 — no rehype-raw):
```tsx
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeSanitize from 'rehype-sanitize'
// NO rehype-raw import — rehype-raw is not installed and must not be added (SEC-03)
<div className="chat-msg__body">
  <Markdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeSanitize]}>
    {message.content}
  </Markdown>
</div>
```

---

### `frontend/src/components/Hub/MentionPopover.tsx` (component, request-response)

**Analog:** `frontend/src/components/Hub/SessionCard.tsx` — `hub-card__menu` dropdown (lines 362–431)

**Keyboard-dismissable dropdown pattern** (from SessionCard lines 267–296):
```typescript
// Close on Escape — global keydown listener while open:
useEffect(() => {
  if (!open) return
  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') { onClose(); }
    // MentionPopover also handles: ArrowUp/ArrowDown (move activeIndex), Enter (select)
  }
  document.addEventListener('keydown', handleKeyDown)
  return () => document.removeEventListener('keydown', handleKeyDown)
}, [open])
```

**Menu ARIA pattern** (from SessionCard lines 363–431):
```tsx
// SessionCard uses role="menu" + role="menuitem"
// MentionPopover uses role="listbox" + role="option" (autocomplete semantic)
<div className="mention-popover" role="listbox" style={{ position: 'absolute', bottom: '100%', left: 0, right: 0 }}>
  {/* Section 1: @session — ALWAYS first, never filtered (D-07) */}
  <div className="mention-popover__section-label" aria-hidden="true">Agent</div>
  <div
    role="option"
    className={`mention-popover__item mention-popover__item--session ${activeIndex === 0 ? 'mention-popover__item--active' : ''}`}
    aria-selected={activeIndex === 0}
    onClick={() => onSelect('@session')}
  >
    <CommandLineIcon className="mention-popover__session-icon" aria-hidden="true" />
    <span className="mention-popover__alias">@session</span>
    <span className="mention-popover__desc">Inject into terminal</span>
  </div>
  <hr className="mention-popover__divider" />
  {/* Section 2: filtered live participants */}
  {filteredParticipants.map((p, i) => (
    <div key={p.personKey} role="option" className="mention-popover__item" ... />
  ))}
</div>
```

**Bottom-anchor positioning** (RESEARCH.md Pattern 6 — no caret tracking needed):
```tsx
// Anchor to composer wrapper div, not caret position:
<div className="chat-composer__popover-anchor" style={{ position: 'relative' }}>
  {mentionOpen && <MentionPopover style={{ position: 'absolute', bottom: '100%', left: 0, right: 0 }} ... />}
  <TextareaAutosize ... />
</div>
```

---

### `frontend/src/components/Hub/ChatDaySeparator.tsx` (component, transform)

**Analog:** `frontend/src/components/Hub/SessionCard.tsx` — section label/divider patterns (line 365, `hub-card__menu-divider`)

**Simple presentational component** — no close analog for sticky day separators; use spec directly:
```tsx
// UI-SPEC §11: centered date label with horizontal rules on both sides
// Sticky behavior comes from parent (ChatPanel virtualizer rangeExtractor) — not from this component
export function ChatDaySeparator({ label }: { label: string }) {
  return (
    <div className="chat-day-sep" aria-label={`Messages from ${label}`}>
      <hr className="chat-day-sep__rule" aria-hidden="true" />
      <span className="chat-day-sep__label">{label}</span>
      <hr className="chat-day-sep__rule" aria-hidden="true" />
    </div>
  )
}
// CSS applied by PARENT (ChatPanel) on the wrapper div:
// active:  { position: 'sticky', top: 0, zIndex: 2 }  — NO transform (Pitfall 1)
// inactive: { position: 'absolute', top: 0, transform: `translateY(...)` }
```

**Date formatting** (Claude's Discretion — standard Intl pattern):
```typescript
function formatDaySeparator(tsMs: number): string {
  const date = new Date(tsMs)
  const today = new Date(); today.setHours(0,0,0,0)
  const yesterday = new Date(today.getTime() - 86400000)
  const msgDay = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  if (msgDay.getTime() === today.getTime()) return 'Today'
  if (msgDay.getTime() === yesterday.getTime()) return 'Yesterday'
  return new Intl.DateTimeFormat('en-US', { weekday: 'short', month: 'short', day: 'numeric' }).format(date)
}
```

---

### `frontend/src/components/Hub/ChatBadge.tsx` (component, transform)

**Analog:** `frontend/src/components/Hub/SessionCard.tsx` — existing ATTN badge (`hub-card__attn-icon`, lines 447–450) and CLI badge (`hub-card__badge`, line 463)

**ATTN badge multi-signal pattern** (lines 447–450 — shape + glyph, never color alone):
```tsx
// Existing ATTN — BellAlertIcon shape carries the state; color is reinforcement:
{isAttention && (
  <span className="hub-card__attn-icon" aria-label="Needs attention">
    <BellAlertIcon aria-hidden="true" />
  </span>
)}
// COPY PATTERN for ChatBadge — @ glyph carries the mention state; color is reinforcement:
export function ChatBadge({ count, hasMention }: { count: number; hasMention: boolean }) {
  if (count === 0) return null
  return (
    <span
      className={`chat-badge ${hasMention ? 'chat-badge--mention' : ''}`}
      aria-label={hasMention
        ? `${count} unread message${count !== 1 ? 's' : ''}, including a mention`
        : `${count} unread message${count !== 1 ? 's' : ''}`}
    >
      {/* @ glyph replaces count when hasMention — glyph is the signal, not color */}
      {hasMention ? '@' : count}
    </span>
  )
}
// CSS: 18px filled circle, --hub-accent bg, --hub-bg foreground, weight 600
```

---

### `frontend/src/components/Hub/HubInteractiveModal.tsx` (component — modification)

**Analog:** itself (lines 1–58 already read)

**Current layout** (line 45–56 — single full-bleed terminal):
```tsx
return (
  <div className="hub-modal__body hub-modal__body--interactive">
    <TerminalPanel sessionId={session.id} isActive={open} ... />
  </div>
)
```

**Modified layout for D-02 overlay mode** — terminal stays full-bleed, drawer is absolutely positioned over it (no resize):
```tsx
// Add chatOpen state + onUnreadChange callback prop. The body stays a single
// full-bleed terminal; the drawer is absolutely positioned over its right edge.
// The terminal width never changes, so the drawer does NOT trigger a PTY resize.
return (
  <div className="hub-modal__body hub-modal__body--interactive">
    <TerminalPanel
      sessionId={session.id}
      isActive={open}  // modal-open prop — unchanged by the drawer; overlay never resizes the terminal
      ...
    />
    {/* Chat toggle button — always rendered (badge visible even when drawer closed) */}
    <button className="hub-modal__chat-toggle" onClick={() => setChatOpen(v => !v)} ...>
      <ChatBubbleLeftRightIcon />
      <ChatBadge count={unreadCount} hasMention={hasMention} />
    </button>
    {/* ChatPanel mounted whenever the modal is open (so unread accrues while closed, D-09);
        the `open` prop drives the slide, NOT conditional mounting */}
    <ChatPanel
      sessionId={session.id}
      relayPort={relayPort}
      open={chatOpen}
      onUnreadChange={(count, mention) => { setUnreadCount(count); setHasMention(mention) }}
    />
  </div>
)
// CSS: .hub-modal__body--interactive { position: relative }  /* containing block for the overlay */
// .chat-panel { position: absolute; top: 0; right: 0; bottom: 0; width: 360px; z-index: 5 } (D-01)
// Drawer animation: translateX(100%)→translateX(0), 220ms ease-out; prefers-reduced-motion: reduce → instant
// NOTE: no hub-modal__terminal-col wrapper and no flex row — the terminal is not pushed. The
//       drawer covers ~360px of the terminal while open (the accepted D-02 tradeoff for Issue #109).
```

**Props to add to `HubInteractiveModalProps`:**
```typescript
onUnreadChange?: (sessionId: string, count: number, hasMention: boolean) => void
```

---

### `frontend/src/components/Hub/SessionCard.tsx` (component — prop addition only)

**Analog:** itself — existing `isAttention` + `BellAlertIcon` badge pattern (lines 143–144, 447–450)

**Existing ATTN prop pattern** (lines 143–144):
```typescript
/** ATTN-01: true when isAttentionStatus(deriveHubStatus(session)) is true */
isAttention?: boolean
```

**New props to add** — mirror the ATTN pattern:
```typescript
/** Phase 154 / NOTIF-01: total unread chat message count for this session */
unreadCount?: number
/** Phase 154 / NOTIF-01: true when any unread message @mentions the current user */
hasChatMention?: boolean
```

**Badge render** — mount after existing ATTN icon (line 447), same slot area:
```tsx
{(unreadCount ?? 0) > 0 && (
  <ChatBadge count={unreadCount!} hasMention={hasChatMention ?? false} />
)}
// Position: overlay top-right of the chat icon/status area (UI-SPEC §10 — consistent with ATTN badge position)
```

---

### `internal/relay/hub.go` — new `HandleChatSend` method

**Analog:** `HandleInject` (lines 492–551) — same pattern, fewer security gates

**HandleInject pattern to copy** (lines 492–548):
```go
func (h *Hub) HandleInject(sub *Subscriber, text string) error {
    if sub.ReadOnly { return ErrReadOnly }                   // SEC-01 gate
    if len(text) > MaxInjectTextBytes { return ErrInjectTooLarge }  // size gate
    sanitized := SanitizePTYText(text)                       // PTY sanitizer
    if strings.TrimSpace(sanitized) == "" { return nil }
    if err := h.WriteInput([]byte(sanitized)); err != nil { return err }
    h.mu.Lock(); fn := h.chatAppendFn; h.mu.Unlock()        // unlock before IO
    if fn != nil {
        msg, err := fn(ChatMessage{...})
        if err != nil { return fmt.Errorf("%w: %v", ErrInjectNotRecorded, err) }
        h.BroadcastChat(MakeChatFrame(msg))
    }
    return nil
}
```

**HandleChatSend** — same structure, skip PTY write and use `SanitizeChatContent` directly:
```go
// HandleChatSend is called by the read pump when a MsgChatSend (0x31) frame arrives.
// Unlike HandleInject it does NOT write to PTY stdin — it only persists + broadcasts.
func (h *Hub) HandleChatSend(sub *Subscriber, content string) error {
    if sub.ReadOnly { return ErrReadOnly }              // SEC-01: RO cannot post chat
    if content = SanitizeChatContent(content); content == "" { return nil }
    h.mu.Lock(); fn := h.chatAppendFn; h.mu.Unlock()
    if fn == nil { return errors.New("relay: chat not available for this session") }
    msg, err := fn(ChatMessage{
        AuthorID:    sub.TailnetID,
        AuthorAlias: sub.Alias,
        Content:     content,
    })
    if err != nil { return err }
    h.BroadcastChat(MakeChatFrame(msg))
    return nil
}
```

---

### `internal/relay/server.go` — add `case MsgChatSend:` to read pump

**Analog:** existing `case MsgSessionInject:` (lines 365–386) and `case MsgTyping:` (lines 340–348)

**MsgTyping pattern** (lines 340–348 — simple unmarshal + hub call):
```go
case MsgTyping:
    var tp TypingPayload
    if json.Unmarshal(payload, &tp) == nil {
        hub.UpdateTyping(sub, tp.Typing)
    }
```

**MsgSessionInject pattern** (lines 365–386 — unmarshal + hub call + NAK on error):
```go
case MsgSessionInject:
    var ip InjectPayload
    if json.Unmarshal(payload, &ip) != nil || ip.Text == "" { continue }
    if err := hub.HandleInject(sub, ip.Text); err != nil {
        log.Printf("relay: inject rejected: %v", err)
        select {
        case sub.Msgs <- MakeInjectErrorFrame(InjectErrorReason(err)):
        default: go sub.CloseSlow()
        }
    }
```

**New `case MsgChatSend:`** — copy MsgSessionInject pattern, use ChatSendPayload:
```go
case MsgChatSend:
    // Phase 154: wire the MsgChatSend (0x31) Phase 154 dispatch stub.
    // Chat send is NOT gated on ReadOnly here — HandleChatSend enforces SEC-01.
    var cp ChatSendPayload  // {Content string `json:"content"`}
    if json.Unmarshal(payload, &cp) != nil || cp.Content == "" { continue }
    if err := hub.HandleChatSend(sub, cp.Content); err != nil {
        log.Printf("relay: chat send rejected: %v", err)
        // Silent drop on error (per RESEARCH.md Open Question 1 recommendation)
    }
```

**Same case must be added to `internal/webserver/server.go` read pump** (lines 1124–1188) — identical pattern.

---

## Shared Patterns

### Colorblind-safe signaling (standing project rule)
**Source:** `frontend/src/components/Hub/SessionCard.tsx` STATUS_CONFIG (lines 36–60) + ATTN badge (lines 447–450)
**Apply to:** `ChatMessage.tsx` (@mention row), `ChatBadge.tsx`, `MentionPopover.tsx` (@session row)

Rule: every status/badge/indicator must carry at minimum 2 non-color channels (shape + glyph or shape + text). Color is always reinforcement only. Verify at hex constants in code — not by eye (user is colorblind).

```tsx
// Existing precedent: STATUS_CONFIG uses Icon (shape) + label (text), hex is reinforcement:
running: { Icon: ArrowPathIcon, label: 'Running', spin: true }
// Apply same principle:
// ChatBadge mention: @ glyph is the signal (not accent color)
// @mention row: accent bar (3px shape) + @you chip (glyph) + tint (color reinforcement)
// @session popover: CommandLineIcon (shape) + "Inject into terminal" (text)
```

### BEM CSS class naming
**Source:** `frontend/src/components/Hub/SessionCard.tsx` — `hub-card__*` classes
**Apply to:** All new components

Pattern: `{component-name}__{element}` with `{component-name}--{modifier}` for states.
```
chat-panel__*     ChatPanel
chat-msg__*       ChatMessage
chat-day-sep__*   ChatDaySeparator
mention-popover__* MentionPopover
chat-badge        ChatBadge (single-element component)
chat-composer__*  composer section of ChatPanel
```

All CSS goes in `frontend/src/style.css` (no CSS modules, no Tailwind — project convention).

### `--hub-*` CSS token usage
**Source:** `frontend/src/style.css` token definitions
**Apply to:** All new components

Tokens to use in this phase (from UI-SPEC §Color):
- Background: `--hub-bg`, `--hub-surface`, `--hub-surface-elevated`
- Text: `--hub-text-primary`, `--hub-text-muted`, `--hub-text-dim`
- Accent: `--hub-accent` (8 reserved uses — see UI-SPEC §Color)
- Border: `--hub-border`
- Hover tint: `--hub-sidebar-item-hover-bg` = `rgba(122, 162, 247, 0.08)` (for @mention row tint)

### React icon import pattern
**Source:** `frontend/src/components/Hub/SessionCard.tsx` lines 8–21
**Apply to:** All new components

```typescript
import {
  CommandLineIcon,
  ChatBubbleLeftRightIcon,
  PaperAirplaneIcon,
  // etc.
} from '@heroicons/react/24/outline'
// ALWAYS /24/outline — never /20/solid unless UI-SPEC explicitly requires it
```

### `prefers-reduced-motion` CSS guard
**Source:** RESEARCH.md Pattern 5 (press-and-hold fill animation)
**Apply to:** `ChatPanel.tsx` inject button fill animation, drawer slide-in animation

```css
@media (prefers-reduced-motion: reduce) {
  .chat-panel { transition: none; }
  .chat-composer__inject-btn--holding::before { transition: none; }
}
```

---

## No Analog Found

None. All files have analogs in the existing codebase.

---

## Metadata

**Analog search scope:** `frontend/src/components/Hub/`, `frontend/src/lib/`, `internal/relay/`
**Files read:** 6 source files (relayClient.ts, HubInteractiveModal.tsx, SessionCard.tsx, relay/server.go lines 310–404, hub.go lines 460–551, style.css badge/attn tokens)
**Pattern extraction date:** 2026-06-26
