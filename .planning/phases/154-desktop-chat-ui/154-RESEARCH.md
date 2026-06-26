# Phase 154: Desktop Chat UI - Research

**Researched:** 2026-06-26
**Domain:** React / TypeScript frontend — chat panel, virtualized message list, safe Markdown, WebSocket framing, Wails webview
**Confidence:** MEDIUM (all library findings verified against installed packages in frontend/node_modules; protocol findings from reading actual source files)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** ChatPanel is a right slide-over drawer, 360px fixed width, inside the session modal. Chat toggle button carries the unread badge.
- **D-02:** Overlay mode — the drawer is absolutely positioned over the right edge of the terminal; the terminal column keeps full width and NO PTY resize is triggered. (Changed from push mode 2026-06-26 to avoid disturbing the host-authority PTY model adopted for Issue #109. See Pattern 7 / Pitfall 6 — both superseded.)
- **D-03:** Drawer starts closed; user opens via badged toggle.
- **D-04:** Slack-style avatar rows; consecutive-sender collapse. No own-right bubbles (colorblind user).
- **D-05:** @mention-of-me = three simultaneous signals: solid left accent bar (3px) + background tint + `@you` chip. Color is never the sole signal.
- **D-06:** `SessionInject:true` message renders as a system-style line with separator + `CommandLineIcon` + "→ injected into terminal" caption. Not a normal bubble.
- **D-07:** `@session` always pinned as first item in the mention autocomplete popover, in its own "Agent" section, never scrolled away by filtering.
- **D-08:** Send button switches to "Inject" state when `@session` is in the composer. Press-and-hold ~600ms fires `MsgSessionInject 0x35`. A tap, Enter, or autocomplete Enter NEVER injects.
- **D-09:** Unread accrues whenever the drawer is not open-and-focused (closed, different session viewed, or window backgrounded).
- **D-10:** Badge shows total unread count; if any unread message @mentions current user, badge switches to `@` glyph state. Two separate counts rejected.

### Claude's Discretion
- CHAT-04 day-separator mechanics: CSS `position: sticky` vs virtualizer-aware header — planner chooses; must interoperate with `@tanstack/react-virtual`.
- Composer auto-grow cap: `react-textarea-autosize` `maxRows` value (`maxRows={6}` per UI-SPEC).
- Avatar/identity coloring: deterministic hue hash on TailnetID; alias text always present.
- Empty/loading/first-message states: standard treatments.
- `rehype-sanitize` schema: start from GitHub default; may only tighten.

### Deferred Ideas (OUT OF SCOPE)
- Web-share chat UI, Markdown export, cross-surface parity gate → Phase 155.
- Native tray / OS notification on @mention (NOTIF-F1) → deferred.
- Triple-backtick code-block heuristic rendering (CHAT-F1) → out of scope.
- Remember-last drawer state / auto-open on unread → rejected in D-03.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CHAT-01 | Send and receive messages in a per-session chat thread; Enter sends, Shift+Enter inserts a newline | relayClient.ts wiring section — encodeChatSendFrame, keyboard handling pattern |
| CHAT-02 | Message stream shows author (alias + tailnet ID), timestamp (HH:MM, ISO-8601 on hover), day separators | ChatMessage wire type, timezone/formatting pattern |
| CHAT-03 | Composer auto-grows with input (capped); message bodies render Markdown safely | react-textarea-autosize API, react-markdown + rehype-sanitize pattern |
| CHAT-04 | Day separators stick to the top of the stream while scrolling | @tanstack/react-virtual v3 sticky pattern — rangeExtractor + hybrid position |
| MENTION-01 | Typing `@` opens filterable, keyboard-navigable autocomplete popover over participants + pinned `@session` | Mention popover architecture; composer-bottom anchor (no caret tracking needed) |
| NOTIF-01 | In-app unread badge on chat toggle and Hub session card; @mention visually distinct | Focus detection strategy; badge state lift to HubPanel |
| NOTIF-02 | Messages that mention current user's alias are highlighted in the stream | ChatMessage.Mentions field contains AuthorIDs; compare against current user TailnetID |
| SEC-03 | Markdown rendering cannot execute injected scripts/HTML; no rehype-raw; XSS payloads inert | defaultSchema verified — strip:['script'], no event handler attrs; pattern confirmed |
</phase_requirements>

---

## Summary

Phase 154 is a pure frontend phase that wires and renders the chat side-channel built by Phases 151–153. The backend is complete and locked. However, two backend file changes are still required: `case MsgChatSend:` dispatch is explicitly a "Phase 154 dispatch stub" in `internal/relay/protocol.go` and is absent from both `internal/relay/server.go` and `internal/webserver/server.go` read pumps — these stubs must be completed before the UI can receive echoed messages. All other server/protocol elements are already implemented.

The frontend work centers on five integration points: (1) extending `relayClient.ts` with chat constants, frame builders, and a fully-dispatching `ws.onmessage`; (2) implementing `ChatPanel.tsx` with a `@tanstack/react-virtual` message list that uses a hybrid sticky/absolute positioning pattern for day separators; (3) safe Markdown via `react-markdown 10.1.0` + `rehype-sanitize 6.0.0` (both already installed, rehype-raw is not installed); (4) the press-and-hold inject gesture on the Send button; and (5) unread badge state management lifted through HubPanel to SessionCard.

Two npm packages are NOT yet in `frontend/package.json`: `@tanstack/react-virtual@^3.14.3` and `react-textarea-autosize@^8.5.9`. Both must be added and installed as Wave 0 before any component work.

**Primary recommendation:** Build in waves: (0) add npm packages + MsgChatSend dispatch stubs, (1) relayClient.ts extension + late-join HTTP scrollback fetch, (2) ChatPanel drawer + virtualizer + day separators, (3) composer + mention popover + press-and-hold, (4) unread badge state + SessionCard wiring, (5) SEC-03 Markdown + UAT verification against the #79 design comp.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Chat message send/receive | API / Backend (relay WS) | Frontend Client | Messages are brokered through the relay hub; the UI only sends encoded frames and receives broadcasts |
| Message thread rendering | Browser / Client | — | Pure UI rendering; no server involvement beyond data receipt |
| Markdown sanitization | Browser / Client | — | SEC-03 is a render-time concern; the backend stores raw content; the frontend sanitizes on display |
| Unread badge state | Browser / Client | — | Derived from messages received + drawer focus state — all client-side |
| Day separator stickiness | Browser / Client | — | CSS + virtualizer control; no server involvement |
| Chat history (late join) | API / Backend (HTTP) | Frontend Client | `GET /api/chat/{id}/history` already implemented; client fetches on mount |
| Press-and-hold gesture | Browser / Client | — | Pure interaction pattern; only the resulting `MsgSessionInject` frame goes to the server |
| @mention detection | Browser / Client | — | `ChatMessage.Mentions` field (AuthorIDs) compared against current user's TailnetID |

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@tanstack/react-virtual` | `^3.14.3` | Virtualized message list | Architecture-locked in STATE.md; 16.8M downloads/week; handles dynamic heights + sticky headers |
| `react-textarea-autosize` | `^8.5.9` | Auto-growing composer textarea | Architecture-locked in STATE.md; 7.7M downloads/week; drop-in `<textarea>` replacement |
| `react-markdown` | `10.1.0` | Markdown rendering | Already installed; official remarkjs component for React |
| `rehype-sanitize` | `^6.0.0` | XSS sanitization | Already installed; hast-util-sanitize 5.0.2 provides the GitHub-style defaultSchema |
| `remark-gfm` | `^4.0.1` | GFM extensions (tables, strikethrough, task lists) | Already installed; locked by CHAT-03 |
| `@heroicons/react` | `^2.2.0` | Icons | Already installed; used throughout the codebase |

### Not Required (already installed)
`react`, `react-dom`, `typescript` — all existing project dependencies.

### NOT Installed — Must Add in Wave 0
```bash
pnpm --filter frontend add @tanstack/react-virtual@^3.14.3 react-textarea-autosize@^8.5.9
```

After install, also add TypeScript types:
```bash
pnpm --filter frontend add -D @types/react-textarea-autosize
```

Note: `@tanstack/react-virtual` ships its own types — no separate `@types` package needed.

**Version verification:** [VERIFIED: npm registry] — `npm view @tanstack/react-virtual@3.14.3 version` → `"3.14.3"`. `npm view react-textarea-autosize@8.5.9 version` → `"8.5.9"`.

---

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `@tanstack/react-virtual` | npm | ~3 yrs (v3) | 16.8M/wk | github.com/TanStack/virtual | SUS (too-new flag: v3.14.4 released day-of-research) | Flagged — planner must add checkpoint. Note: package is well-established; SUS is a recency artifact |
| `react-textarea-autosize` | npm | ~8 yrs | 7.7M/wk | github.com/Andarist/react-textarea-autosize | OK | Approved |
| `react-markdown` | npm | already installed | — | github.com/remarkjs/react-markdown | OK (already installed) | Approved |
| `rehype-sanitize` | npm | already installed (6.0.0) | — | github.com/rehypejs/rehype-sanitize | OK (already installed) | Approved |
| `remark-gfm` | npm | already installed (4.0.1) | — | — | OK (already installed) | Approved |

**Packages removed due to [SLOP] verdict:** none

**Packages flagged as suspicious [SUS]:** `@tanstack/react-virtual` — flagged because a new version (3.14.4) was published on the day this research was conducted (2026-06-26), triggering the recency heuristic. The package itself is the canonical TanStack virtualizer with 16.8M weekly downloads, confirmed GitHub source at TanStack/virtual, and is architecture-locked in STATE.md. The planner should add a `checkpoint:human-verify` task before the install step, but this is low-risk.

**SEC-03 gate:** `rehype-raw` is NOT installed in `frontend/node_modules` and is NOT in `frontend/package.json`. [VERIFIED: npm registry check confirmed absence].

---

## Architecture Patterns

### System Architecture Diagram

```
[Composer textarea]          [ChatPanel RelayClient WS]
     |                               |
     | encodeChatSendFrame()         | onmessage → parseServerFrame()
     | encodeSessionInjectFrame()    |
     ↓                               ↓
[ws://127.0.0.1:{port}/sessions/{id}/ws]
              |
              | relay/server.go read pump (MUST ADD case MsgChatSend)
              ↓
         [relay.Hub]
         BroadcastChat(MakeChatFrame(msg))  ←── chatAppendFn → ChatStore.AppendMessage
              |
              ↓
     [All WS subscribers]
     TerminalPanel WS: discards 0x30 (no callback)
     ChatPanel WS: dispatches to onChat → virtualizer row render

[ChatPanel mount]
     |
     ├─→ GET /api/chat/{id}/history → ChatMessage[] (late-join scrollback)
     └─→ WS subscribe (new frames only)
```

### Recommended Project Structure

```
frontend/src/components/Hub/
├── ChatPanel.tsx           # Full drawer: thread + composer + toggle button
├── ChatMessage.tsx         # Single message row (avatar/header + body, consecutive collapse)
├── ChatDaySeparator.tsx    # Sticky day separator row for virtualizer
├── MentionPopover.tsx      # @ autocomplete: @session pinned + participants
└── ChatBadge.tsx           # Unread badge (count state + @mention state)
frontend/src/lib/relayClient.ts  # Extended with chat constants, builders, callbacks
```

### Pattern 1: relayClient.ts Extension

**What:** Add MSG_CHAT + MSG_CHAT_SEND constants, ChatMessage TypeScript interface, new frame builders, extended callbacks, and full-dispatch `ws.onmessage`.

**Current state (from reading `frontend/src/lib/relayClient.ts`):**
- `RelayClientCallbacks` only has `onOutput`, `onOpen`, `onClose`
- `parseServerFrame` handles `MSG_OUTPUT`, `MSG_RESIZE`, `MSG_PRESENCE`, `MSG_TYPING`; no `MSG_CHAT` (0x30) or `MSG_INJECT_ERROR` (0x36)
- `ws.onmessage` only routes `output` frames; presence and typing are parsed but DISCARDED
- Missing constants: `MSG_CHAT = 0x30`, `MSG_CHAT_SEND = 0x31`, `MSG_SESSION_INJECT = 0x35`, `MSG_INJECT_ERROR = 0x36`

**Required additions:** [VERIFIED: source code read]

```typescript
// Source: frontend/src/lib/relayClient.ts (additions for Phase 154)
export const MSG_CHAT           = 0x30  // server → client: chat message
export const MSG_CHAT_SEND      = 0x31  // client → server: send chat message
export const MSG_SESSION_INJECT = 0x35  // client → server: inject into PTY
export const MSG_INJECT_ERROR   = 0x36  // server → client: inject rejected

// Mirror of internal/relay/protocol.go ChatMessage
export interface ChatMessage {
  v: number           // schema version (always 1)
  id: string
  sessionID: string
  authorID: string    // TailnetID — "local" for desktop owner
  alias: string       // JSON tag is "alias", NOT "authorAlias"
  content: string
  mentions?: string[] // AuthorIDs of mentioned participants
  sessionInject?: boolean
  ts: number          // UNIX milliseconds
}

// ServerFrame union — add chat + inject_error variants
export type ServerFrame =
  | { type: 'output'; payload: Uint8Array }
  | { type: 'resize'; cols: number; rows: number }
  | { type: 'presence'; participants: PresenceEntry[] }
  | { type: 'typing'; personKey: string; alias: string; typing: boolean }
  | { type: 'chat'; message: ChatMessage }
  | { type: 'inject_error'; reason: string }
  | { type: 'unknown' }

// Extended callbacks
export interface RelayClientCallbacks {
  onOutput: (data: Uint8Array) => void
  onPresence?: (participants: PresenceEntry[]) => void
  onTyping?: (personKey: string, alias: string, typing: boolean) => void
  onChat?: (message: ChatMessage) => void
  onInjectError?: (reason: string) => void
  onOpen?: () => void
  onClose?: () => void
}

// New frame builders
export function encodeChatSendFrame(content: string): Uint8Array {
  const encoded = new TextEncoder().encode(JSON.stringify({ content }))
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_CHAT_SEND
  frame.set(encoded, 1)
  return frame
}

export function encodeSessionInjectFrame(text: string): Uint8Array {
  const encoded = new TextEncoder().encode(JSON.stringify({ text }))
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_SESSION_INJECT
  frame.set(encoded, 1)
  return frame
}
```

The `ws.onmessage` handler must be updated to dispatch ALL frame types via callbacks (presence, typing, chat, inject_error), not just output.

**Backward compatibility:** `TerminalPanel` constructs `RelayClient` with `{ onOutput, onOpen, onClose }`. The new optional callbacks (`onPresence?`, `onTyping?`, `onChat?`, `onInjectError?`) use `?:` in the interface and are called with `?.()` — TerminalPanel continues to work unchanged.

### Pattern 2: ChatPanel RelayClient — Separate Subscription

**Decision rationale:** ChatPanel opens its OWN RelayClient WebSocket connection (separate from TerminalPanel's), giving two WS connections per open session modal.

**Why separate, not shared:** [VERIFIED: source code read]
- TerminalPanel owns its `RelayClient` internally via `clientRef.current` — no external access
- TerminalPanel's lifecycle is `[sessionId]` dep-array, not accessible from HubInteractiveModal
- The relay Hub's fan-out broadcasts to ALL subscribers — both connections receive all frame types
- TerminalPanel's connection discards non-output frames (its callbacks ignore them)
- ChatPanel's connection discards output frames (no `onOutput` callback registered)
- This is the same pattern used elsewhere: multiple clients can subscribe to the same Hub

**Connection URL:** Same endpoint as TerminalPanel: `ws://127.0.0.1:{relayPort}/sessions/{sessionId}/ws`

### Pattern 3: @tanstack/react-virtual v3 — Sticky Day Separators

**What:** Virtualized list where day separator items stay sticky at the top while scrolling.

**Item array structure:** [CITED: deepwiki.com/TanStack/virtual/4.4-sticky-headers-and-footers]

```typescript
// Source: TanStack Virtual v3 sticky pattern (deepwiki + official docs)
type VirtualItem =
  | { type: 'message'; message: ChatMessage; isConsecutive: boolean }
  | { type: 'separator'; label: string; isoDate: string }

// Build flat items array by grouping messages by day
function buildItems(messages: ChatMessage[]): VirtualItem[] { ... }
```

**Sticky CSS pattern:** Active sticky separator = `position: sticky; top: 0; z-index: 2; NO transform`.
All other items (messages + non-active separators) = `position: absolute; top: 0; transform: translateY(${item.start}px)`.

**rangeExtractor:** Override to always include the active sticky separator index:

```typescript
// Source: TanStack Virtual v3 sticky example pattern
import { useVirtualizer, defaultRangeExtractor } from '@tanstack/react-virtual'

const separatorIndices = useMemo(
  () => items.reduce<number[]>((acc, item, i) =>
    item.type === 'separator' ? [...acc, i] : acc, []), [items])

const virtualizer = useVirtualizer({
  count: items.length,
  getScrollElement: () => parentRef.current,
  estimateSize: (i) => items[i].type === 'separator' ? 28 : 60,
  measureElement: (el) => el.getBoundingClientRect().height,
  rangeExtractor: (range) => {
    // Find active sticky separator (highest separator index <= startIndex)
    const activeSep = [...separatorIndices]
      .reverse()
      .find((i) => i <= range.startIndex) ?? separatorIndices[0]
    const range_ = defaultRangeExtractor(range)
    const set = new Set(range_)
    if (activeSep !== undefined) set.add(activeSep)
    return [...set].sort((a, b) => a - b)
  }
})
```

**Render loop:**

```tsx
// Source: TanStack Virtual v3 sticky + absolute hybrid
const activeSeparatorIndex = /* highest sep index <= virtualizer.range.startIndex */

{virtualizer.getVirtualItems().map((virtualRow) => {
  const item = items[virtualRow.index]
  const isActiveSeparator = item.type === 'separator' && virtualRow.index === activeSeparatorIndex

  return (
    <div
      key={virtualRow.key}
      ref={virtualizer.measureElement}
      data-index={virtualRow.index}
      style={isActiveSeparator
        ? { position: 'sticky', top: 0, zIndex: 2 }  // NO transform
        : { position: 'absolute', top: 0, left: 0,
            transform: `translateY(${virtualRow.start}px)` }
      }
    >
      {item.type === 'separator'
        ? <ChatDaySeparator label={item.label} />
        : <ChatMessage ... />}
    </div>
  )
})}
```

**Critical:** Remove `transform` from the active sticky separator — CSS `position: sticky` is overridden by `transform` if both are present.

### Pattern 4: Safe Markdown (SEC-03)

**Installed versions:** [VERIFIED: npm registry + node_modules read]
- `react-markdown@10.1.0` — className prop removed in v10; wrap with `<div>` instead
- `rehype-sanitize@6.0.0` using `hast-util-sanitize@5.0.2`
- `remark-gfm@4.0.1`
- `rehype-raw`: NOT installed, NOT in package.json

**defaultSchema verified from `frontend/node_modules/.pnpm/hast-util-sanitize@5.0.2/.../schema.js`:** [VERIFIED: source file read]
- `strip: ['script']` — `<script>` elements stripped entirely from the HAST tree
- `img` attributes: only `[...aria, 'longDesc', 'src']` — no `onerror`, no event handlers
- `a` attributes: only `[...aria, 'dataFootnoteBackref', 'dataFootnoteRef', ['className', 'data-footnote-backref'], 'href']`
- `attributes['*']` does NOT include any event handler names — all `on*` handlers are blocked by omission
- protocols for `src`: only `['http', 'https']` — `javascript:` blocked

**Usage pattern (SEC-03 compliant):**

```tsx
// Source: rehype-sanitize readme + hast-util-sanitize schema.js (installed in project)
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeSanitize from 'rehype-sanitize'

// No rehype-raw — HTML is disabled by default in react-markdown v10
// rehype-sanitize is belt-and-suspenders protection
<div className="chat-msg__body">
  <Markdown
    remarkPlugins={[remarkGfm]}
    rehypePlugins={[rehypeSanitize]}
  >
    {message.content}
  </Markdown>
</div>
```

**XSS proofs:**
- `<script>alert(1)</script>` → stripped from HAST before React render (no HTML element output)
- `<img src=x onerror=alert(1)>` → `onerror` stripped by attribute allowlist; output: `<img src="x">`

**Note on rehype-raw:** Do NOT add rehype-raw even "to allow some HTML." Adding it before rehype-sanitize would parse raw HTML nodes first and rely entirely on the sanitizer; the CONTEXT.md and SEC-03 both prohibit rehype-raw.

### Pattern 5: Press-and-Hold Inject Gesture (D-08)

**What:** The Inject button (when @session is in composer) requires a 600ms hold to fire `MsgSessionInject`. A tap releases before 600ms and fires nothing. Enter never injects.

**Implementation:** [ASSUMED — standard React pointer event pattern; no library lookup needed]

```tsx
// Source: standard browser pointer events + CSS animation pattern
const holdTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
const [isHolding, setIsHolding] = useState(false)
const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches

function handlePointerDown(e: React.PointerEvent) {
  e.currentTarget.setPointerCapture(e.pointerId)  // track pointer even if it leaves button
  setIsHolding(true)
  holdTimerRef.current = setTimeout(() => {
    setIsHolding(false)
    fireInject()
  }, 600)
}

function handlePointerUp() {
  if (holdTimerRef.current) {
    clearTimeout(holdTimerRef.current)
    holdTimerRef.current = null
  }
  setIsHolding(false)
}

// Apply class for fill animation
<button
  className={`chat-composer__inject-btn${isHolding ? ' chat-composer__inject-btn--holding' : ''}`}
  onPointerDown={handlePointerDown}
  onPointerUp={handlePointerUp}
  onPointerCancel={handlePointerUp}
>
```

```css
/* CSS fill animation — GPU-composited scaleX transform on ::before */
.chat-composer__inject-btn { position: relative; overflow: hidden; }
.chat-composer__inject-btn::before {
  content: '';
  position: absolute; inset: 0;
  background: var(--hub-accent);
  transform: scaleX(0);
  transform-origin: left;
  transition: none;
}
.chat-composer__inject-btn--holding::before {
  transform: scaleX(1);
  transition: transform 600ms linear;
}

@media (prefers-reduced-motion: reduce) {
  .chat-composer__inject-btn--holding::before { transition: none; }
}
```

**Accidental-Enter safety:** Keyboard Enter in the composer textarea fires `encodeChatSendFrame` (via `onKeyDown`, not button click) — the button's `onClick` is a no-op. The inject button has no `form` association and no keyboard shortcut. The only inject path is a completed 600ms hold.

### Pattern 6: Composer @-Mention Popover Positioning

**Key insight:** The UI-SPEC positions the MentionPopover "directly above the composer (bottom-anchored), same width as composer." This means **no caret coordinate tracking is needed** — the popover anchors to the composer container's top edge, not to the `@` character's position. [CITED: 154-UI-SPEC.md §8 @mention Autocomplete Popover]

**Simple implementation:**

```tsx
// Position relative to composer wrapper div; no getBoundingClientRect of caret
<div className="chat-composer__popover-anchor" style={{ position: 'relative' }}>
  {mentionOpen && (
    <MentionPopover
      style={{ position: 'absolute', bottom: '100%', left: 0, right: 0 }}
      participants={participants}
      filter={mentionFilter}
      onSelect={handleMentionSelect}
    />
  )}
  <TextareaAutosize ... />
</div>
```

**Trigger:** `@` typed in textarea → parse text from last `@` to caret position for filter string → open popover. On selection, replace `@filter` with `@alias` in the textarea and close.

### Pattern 7: D-02 Overlay Mode — Drawer Floats Over the Terminal (no resize)

> **SUPERSEDED 2026-06-26.** This pattern originally described *push mode* (the terminal column
> shrinking by 360px, driving a PTY resize through `TerminalPanel`'s ResizeObserver). D-02 changed
> to **overlay mode** to keep the host PTY grid stable for the Issue #109 host-authority screen-share
> model. The resize machinery below is no longer used by the drawer; it is retained only as the
> reference for how `TerminalPanel` resize works in general (still relevant to Issue #109's own phase).

**Overlay implementation in HubInteractiveModal:** Keep the body as a single full-bleed terminal and
absolutely position the drawer over its right edge. The terminal width never changes, so the
ResizeObserver does NOT fire on drawer toggle and no PTY resize is sent:

```tsx
// HubInteractiveModal changes for D-02 (overlay)
return (
  <div className="hub-modal__body hub-modal__body--interactive">  {/* position: relative */}
    <TerminalPanel ... isActive={open} />   {/* full width, unchanged by the drawer */}
    <ChatPanel sessionId={session.id} relayPort={relayPort} open={chatOpen} ... />
  </div>
)
```

```css
.hub-modal__body--interactive { position: relative; }       /* containing block */
.chat-panel { position: absolute; top: 0; right: 0; bottom: 0; width: 360px; z-index: 5; }
/* slide: translateX(100%) → translateX(0), 220ms ease-out; prefers-reduced-motion: reduce → instant */
```

**Why this is correct for #109:** push mode would call `client.sendResize(cols, rows)` on every drawer
toggle, changing the host PTY grid. Under Issue #109's host-authority model the host is the single
source of truth for PTY size and guests conform to it — a drawer-driven resize would ripple a re-conform
to every connected guest. Overlay mode leaves the PTY untouched. Tradeoff: the drawer covers ~360px of
the terminal while open.

**`isActive` prop:** Stays `true` while the session modal is open (the modal-open prop, unchanged by the
drawer). Since the overlay never resizes the terminal, there is no resize-timing concern — see Pitfall 6
(also superseded).

### Pattern 8: Unread Badge State Lift

**State ownership:** ChatPanel tracks `unreadCount` and `hasMention` internally, then lifts state up via a callback prop `onUnreadChange(sessionId, count, hasMention)`.

**Chain:** `ChatPanel` → `HubInteractiveModal` → `HubModal` → `HubPanel` → `SessionCard`

**Alternative (simpler for Phase 154):** Since there is only ONE session modal open at a time, lift unread state to `HubInteractiveModal` and pass it back up to the Hub session card via an existing session-level callback. The planner chooses the exact prop-threading path.

**Unread accrual logic (D-09):**

```typescript
// Unread accrues when: drawer is closed OR window is not focused
const [drawerOpen, setDrawerOpen] = useState(false)
const [windowFocused, setWindowFocused] = useState(document.hasFocus())

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

const shouldAccrue = !drawerOpen || !windowFocused

// On each incoming MsgChat frame:
if (shouldAccrue) {
  setUnreadCount(c => c + 1)
  if (msg.mentions?.includes(myTailnetID)) setHasMention(true)
}

// Clear when drawer opens and window is focused:
useEffect(() => {
  if (drawerOpen && windowFocused) {
    setUnreadCount(0)
    setHasMention(false)
  }
}, [drawerOpen, windowFocused])
```

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Virtualized message list | Custom scroll handler + DOM recycling | `@tanstack/react-virtual` | Handles dynamic heights, sticky headers, scroll restoration, keyboard nav — 5k+ loc of edge cases |
| Auto-growing textarea | Manual `rows` calculation or `height` JS | `react-textarea-autosize` | Handles cross-browser height calculation, IME input, copy-paste resize, contentEditable pitfalls |
| Markdown XSS sanitization | Custom regex/string replacement | `rehype-sanitize` (already installed) | The hast-util-sanitize schema handles DOM clobbering, protocol stripping, attribute allowlisting — regex misses Punycode, Unicode homoglyphs, new HTML5 attrs |
| Caret position for popover | `textarea-caret` or mirror-div technique | Bottom-anchor to composer div | The UI-SPEC specifies bottom-anchored popover — no caret tracking needed, simpler, no cross-browser font-metrics pitfalls |
| Press-and-hold debouncing | Interval polling | `setTimeout` + `clearTimeout` on `pointerup` | Standard reliable pattern; intervals drift and can fire after unmount |
| Binary frame encoding | Custom byte packing | Extend existing `encode*Frame` pattern in `relayClient.ts` | Consistency with existing codebase; reduces integration bugs |

**Key insight:** The existing relay client encoding pattern (1-byte type prefix + JSON body as UTF-8) is the standard for all client→server frames in this project. Follow it exactly for `MsgChatSend` and `MsgSessionInject`.

---

## Common Pitfalls

### Pitfall 1: Sticky Separator + Virtualizer Transform Conflict
**What goes wrong:** Applying `transform: translateY(...)` to a day separator that's supposed to be `position: sticky` causes the sticky behavior to be overridden — the browser creates a new stacking context and sticky positioning fails.
**Why it happens:** CSS `transform` on an element creates a new containing block, which breaks `position: sticky` (sticky needs an ancestor scroll container, not a transformed containing block).
**How to avoid:** Active sticky separator gets ONLY `position: sticky; top: 0; z-index: 2` — remove the transform. Non-active separators and all message rows use the absolute + transform pattern.
**Warning signs:** Day separator scrolls with content instead of sticking to top.

### Pitfall 2: RelayClient Callback Interface Not Backward-Compatible
**What goes wrong:** Changing `RelayClientCallbacks` to require `onPresence/onTyping/onChat` breaks `TerminalPanel` (which constructs `RelayClient` with only `onOutput/onOpen/onClose`).
**Why it happens:** TypeScript requires all non-optional interface members; if the new callbacks are added as required, TerminalPanel's object literal no longer satisfies the interface.
**How to avoid:** Mark all new callbacks as optional (`onPresence?: ...; onTyping?: ...; onChat?: ...`). Call them with optional chaining (`callbacks.onChat?.(msg)`).
**Warning signs:** TypeScript error at TerminalPanel's RelayClient construction.

### Pitfall 3: MsgChatSend Dispatch Missing on Server
**What goes wrong:** ChatPanel sends `MsgChatSend (0x31)` frames, but they're silently ignored by the server's read pump. Messages appear to send locally but no echo comes back, and other participants never see the message.
**Why it happens:** `relay/server.go` and `internal/webserver/server.go` read pumps have NO `case MsgChatSend:` — this is explicitly a "Phase 154 dispatch stub" in `internal/relay/protocol.go`. The comment at relay/server.go:369 clarifying "MsgChatSend never writes to PTY" is inside the `MsgSessionInject` case, not a separate dispatch case.
**How to avoid:** Add `case MsgChatSend:` in BOTH server read pumps (relay AND webserver). Call a new `hub.HandleChatSend(sub, content)` method that calls `chatAppendFn` + `BroadcastChat`. See server-side work section below.
**Warning signs:** Messages "send" with no echo; other clients see nothing.

### Pitfall 4: Parsing the Wrong JSON Tag for Author Alias
**What goes wrong:** The `ChatMessage` TypeScript interface uses `authorAlias` but the Go struct uses JSON tag `"alias"` — the alias is always undefined on received messages.
**Why it happens:** Go struct field is `AuthorAlias string json:"alias"` (not `"authorAlias"`). From `internal/relay/protocol.go:254`: `AuthorAlias string \`json:"alias"\``.
**How to avoid:** TypeScript interface must be `alias: string` (not `authorAlias`). [VERIFIED: protocol.go source read]
**Warning signs:** `message.alias` is undefined; author header shows empty.

### Pitfall 5: Chat History + WS Gap
**What goes wrong:** ChatPanel loads history via HTTP GET, then subscribes via WS. Messages sent between the HTTP response and WS subscription are missed.
**Why it happens:** Unlike terminal scrollback (the Hub replays from its scrollback buffer on WS connect), chat history is served from `ChatStore.Messages()` which is a snapshot, not replayed via WS. There's no chat scrollback in the WS handshake.
**How to avoid:** Subscribe WS FIRST (to start receiving live frames), then fetch history. Deduplicate by message ID: the HTTP history may overlap with WS messages received during the fetch. Use a `Set<string>` of received IDs.
**Warning signs:** Intermittent missed messages on first load, more noticeable on slow connections.

### Pitfall 6: `isActive=false` During Drawer Animation Breaks PTY Resize
> **SUPERSEDED 2026-06-26 (D-02 → overlay mode).** This pitfall only existed under *push mode*, where
> the drawer resized the terminal column and relied on the ResizeObserver firing mid-animation. In
> overlay mode the drawer never changes the terminal width, so there is no resize to break. Keeping
> `isActive=true` while the modal is open is still correct for the terminal lifecycle, but it is no
> longer load-bearing for the drawer. Retained for history.

**What went wrong (push mode):** If `isActive` was set to `false` during the 220ms drawer open animation, the ResizeObserver in TerminalPanel disconnected and `fitTerminal` never fired when the column width stabilized.
**Why it happened:** The `isActive` effect in TerminalPanel returns early when `isActive === false`, disconnecting the ResizeObserver.
**How it was avoided:** Keep `isActive === true` at all times while the session modal is open. The `isActive` prop gates the terminal on whether the session modal IS open. (In overlay mode this is automatic — the drawer does not touch the terminal.)

### Pitfall 7: Enter Key Injects Instead of Sends
**What goes wrong:** User presses Enter to send a message while `@session` is in the composer; the message is injected into the PTY.
**Why it happens:** The textarea `onKeyDown` handler routes Enter to whatever send action is current; if "inject" state is active, it might call `fireInject`.
**How to avoid:** Enter in the composer textarea ALWAYS calls `encodeChatSendFrame` regardless of `@session` presence. The inject action is ONLY triggerable via the 600ms press-and-hold on the button. The two code paths must never share the same keyboard handler branch. [CITED: 154-CONTEXT.md D-08]
**Warning signs:** Press Enter with `@session` in message → terminal receives input.

### Pitfall 8: `visibilityState` Unreliable in WKWebView
**What goes wrong:** Relying on `document.visibilityState` or `visibilitychange` events to detect window focus fails on macOS (WKWebView). Messages continue to accrue unread even when the window is visible.
**Why it happens:** WKWebView on macOS has known bugs where `visibilitychange` does not fire reliably. [CITED: developer.apple.com/forums/thread/733769]
**How to avoid:** Use `window` `focus`/`blur` events instead of `visibilitychange`. These are reliably dispatched in WKWebView and WebView2. The `document.hasFocus()` call on initialization correctly reflects the starting state.
**Warning signs:** Unread badge doesn't clear when window is switched back into focus on macOS.

---

## Undeclared Server-Side Work (CRITICAL)

The additional_context states "no new server/protocol work" but `internal/relay/protocol.go` explicitly marks `MsgChat (0x30)` and `MsgChatSend (0x31)` as **"Phase 154 dispatch stubs"**. Reading the actual source code confirms:

**`internal/relay/server.go` read pump** (verified at lines 324–388): Has `case MsgInput`, `case MsgResize2`, `case MsgPing`, `case MsgTyping`, `case MsgAliasSet`, `case MsgSessionInject`. **No `case MsgChatSend:`.**

**`internal/webserver/server.go` read pump** (verified at lines 1124–1188): Same omission.

**Required additions (small but blocking):**

1. Add `hub.HandleChatSend(sub *Subscriber, content string) error` method to `internal/relay/hub.go`:
   - Gate: any subscriber (not gated on ReadOnly per D-06 — chat is accessible to RO clients)
   - Sanitize content with `SanitizeChatContent(content)` (already exists in Phase 153)
   - Call `chatAppendFn` to persist + get stamped message
   - Call `BroadcastChat(MakeChatFrame(msg))` to fan out to all subscribers

2. Add `case relay.MsgChatSend:` to `internal/relay/server.go` read pump (~8 lines)

3. Add `case relay.MsgChatSend:` to `internal/webserver/server.go` read pump (~8 lines)

This is the "wire the stub" work. The BroadcastChat infrastructure (`hub.BroadcastChat`, `relay.MakeChatFrame`, `chatAppendFn` wiring in engine.go) is already complete.

---

## Code Examples

### 1. Wire MsgChat in parseServerFrame

```typescript
// Source: frontend/src/lib/relayClient.ts (Phase 154 addition)
case MSG_CHAT: {
  try {
    const json = new TextDecoder().decode(data.slice(1))
    const msg = JSON.parse(json) as ChatMessage
    return { type: 'chat', message: msg }
  } catch {
    return { type: 'unknown' }
  }
}

case MSG_INJECT_ERROR: {
  try {
    const json = new TextDecoder().decode(data.slice(1))
    const parsed = JSON.parse(json) as { reason: string }
    return { type: 'inject_error', reason: parsed.reason ?? 'unknown error' }
  } catch {
    return { type: 'unknown' }
  }
}
```

### 2. Chat History Late-Join Load

```typescript
// Source: pattern from chat_routes.go API contract
async function loadChatHistory(relayPort: number, sessionId: string): Promise<ChatMessage[]> {
  const url = `http://127.0.0.1:${relayPort}/api/chat/${sessionId}/history`
  const resp = await fetch(url)
  if (!resp.ok) return []
  return resp.json() as Promise<ChatMessage[]>
}
```

### 3. hub.HandleChatSend (Go — server addition)

```go
// Source: internal/relay/hub.go (Phase 154 addition)
// HandleChatSend accepts a normal chat message from a subscriber, persists
// it via chatAppendFn, and broadcasts a MsgChat frame to all subscribers.
// Unlike HandleInject, chat send is NOT gated on ReadOnly — D-06 / REQUIREMENTS.md:
// "SEC-01: read-only capability holders cannot post CHAT MESSAGES or trigger injection"
// Wait — re-read SEC-01: "cannot post chat messages" — RO IS gated on chat send!
func (h *Hub) HandleChatSend(sub *Subscriber, content string) error {
    if sub.ReadOnly {
        return ErrReadOnly  // SEC-01: RO clients cannot post chat messages
    }
    if content = SanitizeChatContent(content); content == "" {
        return nil // empty after sanitize — silent no-op
    }
    h.mu.Lock()
    fn := h.chatAppendFn
    h.mu.Unlock()
    if fn == nil {
        return errors.New("relay: chat not available for this session")
    }
    msg, err := fn(ChatMessage{
        AuthorID:    sub.TailnetID,
        AuthorAlias: sub.Alias,
        Content:     content,
    })
    if err != nil {
        return err
    }
    h.BroadcastChat(MakeChatFrame(msg))
    return nil
}
```

**SEC-01 re-confirmation:** REQUIREMENTS.md SEC-01 says "Read-only capability holders cannot POST CHAT MESSAGES or trigger injection." So `HandleChatSend` MUST gate on `!sub.ReadOnly`.

### 4. Day separator date formatting

```typescript
// Source: [ASSUMED] — standard Intl.DateTimeFormat pattern
function formatDaySeparator(isoDate: string): string {
  const date = new Date(isoDate)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today.getTime() - 86400000)
  const msgDay = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  
  if (msgDay.getTime() === today.getTime()) return 'Today'
  if (msgDay.getTime() === yesterday.getTime()) return 'Yesterday'
  return new Intl.DateTimeFormat('en-US', { weekday: 'short', month: 'short', day: 'numeric' }).format(date)
}
```

### 5. Avatar hue derivation (D-04 Claude's Discretion)

```typescript
// Source: [ASSUMED] — deterministic hash-to-hue pattern
function tailnetIdToHue(tailnetID: string): number {
  let hash = 0
  for (let i = 0; i < tailnetID.length; i++) {
    hash = (hash * 31 + tailnetID.charCodeAt(i)) >>> 0
  }
  return hash % 360
}
// Usage: background: `hsl(${tailnetIdToHue(tailnetID)}, 55%, 45%)`
// Foreground: white (contrast checked against hsl range at 45% lightness)
```

---

## Validation Architecture

Framework: vitest (installed, `vitest@^4.1.0` in devDependencies)

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CHAT-01 | Enter sends MsgChatSend frame; Shift+Enter inserts newline | unit | `vitest run frontend/src/lib/relayClient.test.ts` | ❌ Wave 0 |
| CHAT-02 | ChatMessage fields displayed (alias, tailnetID, HH:MM timestamp) | unit | `vitest run frontend/src/components/Hub/ChatMessage.test.tsx` | ❌ Wave 0 |
| CHAT-03 | Composer grows to maxRows=6; Markdown renders safely | unit | `vitest run frontend/src/components/Hub/ChatPanel.test.tsx` | ❌ Wave 0 |
| CHAT-04 | Day separator sticky: correct CSS applied to active separator | unit | `vitest run frontend/src/components/Hub/ChatDaySeparator.test.tsx` | ❌ Wave 0 |
| MENTION-01 | @-mention popover opens on @, @session pinned first, keyboard nav works | unit | `vitest run frontend/src/components/Hub/MentionPopover.test.tsx` | ❌ Wave 0 |
| NOTIF-01 | Unread badge shows count; @mention badge shows @ glyph | unit | `vitest run frontend/src/components/Hub/ChatBadge.test.tsx` | ❌ Wave 0 |
| NOTIF-02 | @mention row has accent bar + tint + @you chip | unit | `vitest run frontend/src/components/Hub/ChatMessage.test.tsx` | ❌ Wave 0 |
| SEC-03 | `<script>alert(1)</script>` renders as text; `<img onerror=...>` strips onerror | unit | `vitest run frontend/src/components/Hub/ChatPanel.test.tsx -t "sec-03"` | ❌ Wave 0 |
| D-08 | Press-and-hold < 600ms fires nothing; >= 600ms fires inject; Enter never injects | unit | `vitest run frontend/src/components/Hub/ChatPanel.test.tsx -t "inject"` | ❌ Wave 0 |
| D-08 (UAT defer) | Inject indicator "→ injected into terminal" visible for SessionInject:true messages | manual | Phase 154 UAT checklist | — |

### Wave 0 Gaps
- [ ] `frontend/src/lib/relayClient.test.ts` — encodeChatSendFrame, encodeSessionInjectFrame, parseServerFrame (chat, inject_error cases)
- [ ] `frontend/src/components/Hub/ChatPanel.test.tsx` — composer send/receive, SEC-03 XSS, press-and-hold
- [ ] `frontend/src/components/Hub/ChatMessage.test.tsx` — field rendering, @mention signals, inject indicator
- [ ] `frontend/src/components/Hub/MentionPopover.test.tsx` — popover behavior, @session pinned first
- [ ] `frontend/src/components/Hub/ChatBadge.test.tsx` — unread count, @mention state
- [ ] `frontend/src/components/Hub/ChatDaySeparator.test.tsx` — sticky CSS, date formatting

### Test Infrastructure
vitest is installed and configured. `pnpm test` in `frontend/` runs the full suite.

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | Chat is session-scoped by the relay's existing session auth |
| V4 Access Control | yes | SEC-01: `ReadOnly` gate in `HandleChatSend` and `HandleInject` (server-side, already implemented) |
| V5 Input Validation | yes | SEC-03: rehype-sanitize (client); SanitizeChatContent (server, existing Phase 153) |
| V6 Cryptography | no | — |

### Known Threat Patterns for Chat Rendering

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Stored XSS via Markdown | Spoofing/Tampering | rehype-sanitize defaultSchema (no event handlers, no script, no javascript:) |
| HTML injection via raw element | Tampering | rehype-raw NOT installed; raw HTML disabled by default in react-markdown v10 |
| PTY injection via accidental Enter | Elevation of privilege | D-08 press-and-hold; Enter always routes to chat send not inject |
| DOM clobbering via `id`/`name` attrs | Spoofing | hast-util-sanitize prefixes clobbered attrs with `user-content-` |
| Session inject by RO client | Elevation of privilege | SEC-01: `HandleChatSend` and `HandleInject` both gate on `!sub.ReadOnly` (server-side) |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Press-and-hold implemented with `setTimeout` + `clearTimeout` + CSS `scaleX` transition | Pattern 5 (press-and-hold) | No library exists that does this differently; risk is low |
| A2 | Avatar hue derivation: `hash(tailnetID) % 360` with `hsl(h, 55%, 45%)` | Code Examples §5 | If hue at 45% lightness doesn't pass WCAG against white text on all hues, avatar text may be low-contrast — but alias text is always present so avatar is supplementary |
| A3 | `window` `focus`/`blur` events are reliable in Wails WKWebView (macOS) | Pitfall 8, Unread pattern | If window focus/blur also has WKWebView bugs, unread clearing may be unreliable; workaround: also use IntersectionObserver on the drawer panel as tertiary signal |
| A4 | The planner routes unread state up through HubInteractiveModal → HubPanel → SessionCard | Unread Badge State Lift | If the prop-threading depth is too high, a context solution (React context) may be preferable; planner decides |
| A5 | `SanitizeChatContent` already exists from Phase 153 and can be reused in `HandleChatSend` | Server-side wiring | If `SanitizeChatContent` is not exported from the relay package, it may need to be re-used or duplicated |

---

## Open Questions

1. **HandleChatSend — SEC-01 read-only gate**
   - What we know: REQUIREMENTS.md SEC-01 says RO clients "cannot post chat messages." The chat send path doesn't exist yet.
   - What's unclear: Should the server NAK the sender with a `MsgInjectError` (0x36) style frame, or just silently drop?
   - Recommendation: Silent drop (same as MsgTyping for consistency). NAK adds a new client error display path not in scope for Phase 154.

2. **ChatPanel RelayClient URL and scrollback replay**
   - What we know: The WS endpoint replays terminal scrollback on connect. Chat history is served separately via HTTP GET.
   - What's unclear: Should the ChatPanel WS connection open a `readonly` URL flag (e.g. `?readonly=1`)? Desktop owner is always RW; this only matters for future web use. Phase 155 reuses this.
   - Recommendation: No `readonly` flag on the desktop ChatPanel's WS. The desktop owner is always the relay "local:local" identity (RW).

3. **Mention detection: by alias vs. by AuthorID**
   - What we know: `ChatMessage.Mentions` contains `AuthorID` strings (TailnetIDs), not aliases. The current user's TailnetID is `"local"` for the desktop owner.
   - What's unclear: Phase 155 (web share) clients have real Tailscale node pubkeys as TailnetIDs. The desktop ChatPanel only needs to detect `Mentions.includes("local")`.
   - Recommendation: Use `"local"` as the current user's TailnetID for Phase 154 desktop detection. Phase 155 can pass the real TailnetID from the web identity system.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| pnpm | npm install | ✓ | detected in project | — |
| Node.js | frontend build | ✓ | detected | — |
| `@tanstack/react-virtual@3.14.3` | CHAT-04 | ✗ | NOT INSTALLED | Must install in Wave 0 |
| `react-textarea-autosize@8.5.9` | CHAT-03 | ✗ | NOT INSTALLED | Must install in Wave 0 |
| `react-markdown@10.1.0` | SEC-03 | ✓ | 10.1.0 installed | — |
| `rehype-sanitize@6.0.0` | SEC-03 | ✓ | 6.0.0 installed | — |
| `remark-gfm@4.0.1` | CHAT-03 | ✓ | 4.0.1 installed | — |
| `rehype-raw` | SEC-03 (MUST NOT be present) | ✗ | NOT installed | Correct absence — do not install |

**Missing dependencies blocking execution:**
- `@tanstack/react-virtual@^3.14.3` — required before any virtualizer code; must install in Wave 0
- `react-textarea-autosize@^8.5.9` — required for composer; must install in Wave 0

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `react-virtualized` / `react-window` for lists | `@tanstack/react-virtual` | TanStack v3 (2023) | Headless virtualizer; works with any DOM structure; required for sticky + dynamic heights |
| `className` prop on `<Markdown>` | Wrap with `<div className="...">` | react-markdown v10 (Feb 2025) | Breaking change from v9 |
| `rehype-raw + rehype-sanitize` for safe HTML | `rehype-sanitize` only (no `rehype-raw`) | Project design decision (SEC-03) | No raw HTML in chat bodies; sanitizer is belt-and-suspenders |
| `document.visibilityState` for focus detection | `window.focus/blur` events | Wails WKWebView reality | visibilitychange unreliable in WKWebView macOS |

**Deprecated/outdated:**
- `react-virtualized` and `react-window`: older virtualization libraries; TanStack Virtual is the current standard and the locked choice for this project
- `rehype-raw`: not installed and must not be installed; raw HTML pass-through bypasses remark's safe text parsing

---

## Sources

### Primary (MEDIUM confidence — verified against installed files)
- `frontend/src/lib/relayClient.ts` — frame constants, callback interface, relay client pattern [VERIFIED: source file read]
- `internal/relay/protocol.go` — ChatMessage wire type (JSON tags confirmed), frame constants, Phase 154 stub annotations [VERIFIED: source file read]
- `internal/relay/hub.go` — BroadcastChat, HandleInject, SetChatAppendFn, HubManager subscription pattern [VERIFIED: source file read]
- `internal/relay/server.go` — read pump dispatch (confirmed missing MsgChatSend case) [VERIFIED: source file read]
- `internal/webserver/server.go` — web server read pump (confirmed missing MsgChatSend case) [VERIFIED: source file read]
- `internal/daemon/chat.go` — ChatStore API, late-join scrollback source [VERIFIED: source file read]
- `internal/daemon/chat_routes.go` — `/api/chat/{id}/history` and `/api/chat/{id}/export` endpoints [VERIFIED: source file read]
- `frontend/node_modules/.pnpm/hast-util-sanitize@5.0.2/.../schema.js` — complete defaultSchema object [VERIFIED: source file read]
- `frontend/node_modules/rehype-sanitize/index.js` — ESM export pattern [VERIFIED: source file read]
- `frontend/src/components/Hub/HubInteractiveModal.tsx` — current modal structure [VERIFIED: source file read]
- `frontend/src/components/Hub/SessionCard.tsx` — badge pattern for unread extension [VERIFIED: source file read]
- `frontend/src/components/TerminalPanel.tsx` — isActive/ResizeObserver/fitTerminal pattern [VERIFIED: source file read]
- `frontend/package.json` — confirmed installed/missing packages [VERIFIED: source file read]

### Secondary (LOW confidence — web search)
- deepwiki.com/TanStack/virtual/4.4-sticky-headers-and-footers — sticky headers pattern [CITED: web source]
- developer.apple.com/forums/thread/733769 — WKWebView visibilitychange unreliability [CITED: Apple Developer Forums]
- react-markdown changelog.md — v10 className prop removal [CITED: github.com/remarkjs/react-markdown]

### Tertiary (ASSUMED — training knowledge)
- Press-and-hold gesture implementation with `setTimeout/clearTimeout` — standard browser pattern
- Avatar hue hash derivation
- Day separator date formatting with `Intl.DateTimeFormat`

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all installed package versions verified from node_modules; missing packages verified on npm registry
- Protocol/wiring facts: HIGH — read actual Go and TypeScript source files
- Architecture patterns: MEDIUM — relayClient extension pattern derived from existing code; virtualizer sticky from documentation
- Wails webview focus: MEDIUM — Apple Developer Forums confirm WKWebView visibilitychange unreliability; window focus/blur recommendation is standard fallback

**Research date:** 2026-06-26
**Valid until:** 2026-07-26 (stable libraries; Wails focus behavior stable)
