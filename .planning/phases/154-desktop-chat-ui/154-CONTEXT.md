# Phase 154: Desktop Chat UI - Context

**Gathered:** 2026-06-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Build the **desktop GUI chat panel** — a `ChatPanel.tsx` mounted inside the session modal
(`HubInteractiveModal`) that turns the human-to-human session side-channel into a fully usable
UI on the **desktop surface only**. Covers CHAT-01, CHAT-02, CHAT-03, CHAT-04, MENTION-01,
NOTIF-01, NOTIF-02, SEC-03.

This is the **UI layer that rides on top of the server-side work already shipped in Phases
151–153**: the daemon `ChatStore` (JSONL persistence, cap, export), the relay/web frame set
(`0x30`–`0x35`), presence/typing/alias, and the RW-gated, sanitized one-way `@session→PTY`
inject path. Phase 154 **wires the `MsgChat` / `MsgChatSend` dispatch stubs** (currently no-op
placeholders in `relayClient.ts`) and **renders** everything the backend already broadcasts.

**In scope (Phase 154):**
- `ChatPanel.tsx` inside the session modal: send (Enter) / newline (Shift+Enter), auto-growing
  composer, message thread with author (alias + tailnet ID) + HH:MM timestamp (ISO-8601 on
  hover), day separators (sticky to top of viewport).
- Safe Markdown rendering of message bodies (`react-markdown` + `remark-gfm` +
  `rehype-sanitize`, **no `rehype-raw`**, no raw HTML) — SEC-03.
- `@` mention autocomplete popover (participants + pinned `@session`), keyboard-navigable.
- The **press-and-hold `@session` confirm gesture** in the composer (carried from Phase 153 —
  the structural guarantee exists; this is the affordance that emits the dedicated
  `MsgSessionInject 0x35` frame).
- Rendering the **"→ injected into terminal" indicator** for `SessionInject:true` messages
  (carried from Phase 153 — the data is broadcast; this is the visual treatment).
- Unread badges on the chat toggle **and** the Hub session card; `@mention`-of-me visual
  distinction (NOTIF-01/02).
- The two **Phase 153 UAT items deferred into 154** (see `<canonical_refs>`): the inject
  indicator render, and the accidental-Enter confirm UX.

**Out of scope (later phases / explicitly designed out):**
- The **web-share** chat UI + Markdown **export** + the cross-surface **parity gate** →
  **Phase 155** (shares this `ChatPanel.tsx`).
- Native tray / OS notification on `@mention` (NOTIF-F1) — deferred.
- Agent→chat round-trip / reply parsing — permanently out of scope (one-way bridge).
- Any new server/protocol work — frames `0x30`–`0x35`, the daemon store, and the inject path
  are locked upstream.

</domain>

<decisions>
## Implementation Decisions

### Panel placement & layout
- **D-01: Right slide-over drawer.** `ChatPanel` is a drawer that slides in from the right edge
  of the session modal, opened via a **chat toggle button** that carries the unread badge. The
  terminal is the primary surface; chat is opt-in. (Rejected: always-on side-by-side split —
  permanently costs terminal width for what is usually a solo session; tab-toggle swap — you
  can't watch the terminal while chatting, which weakens the `@session` "inject then watch the
  reply" flow.)
- **D-02: Overlay mode — the drawer floats over the terminal; the terminal is NOT resized.**
  Opening/closing the drawer slides a fixed 360px panel in over the right edge of the terminal
  (covering it) without changing the `TerminalPanel` width — so no PTY resize/reflow is triggered.
  The drawer is absolutely positioned against the modal body; the terminal column stays full-bleed.
  (**Changed from push mode on 2026-06-26.** Push mode resized the host PTY on every drawer toggle,
  which fights the host-authority "screen-share semantics" model adopted for GitHub Issue #109 — the
  host's terminal must be the single source of truth for the PTY grid size, with guests conforming.
  A drawer that resizes the host PTY would force every guest to re-conform on each toggle. Tradeoff
  accepted: the drawer covers ~360px of the terminal while chatting, which the user previously did
  not want but now prefers over disturbing the shared PTY.)
- **D-03: Closed by default.** The drawer starts closed when a session modal opens; the user
  opens it via the badged toggle. (Rejected: remember-last-state — adds a persistence layer for
  marginal benefit; auto-open-on-unread — prematurely couples drawer state to notification
  logic.)

### Message thread visual style
- **D-04: Slack-style avatar rows.** Flat, left-aligned rows: avatar + alias + timestamp
  header, message body below; collapse the header on consecutive messages from the same author.
  Matches the #79 design prototype (avatars + composer) and scales with `@tanstack/react-virtual`.
  (Rejected: own-right chat bubbles — right-alignment + color-fill is a weak signal for a
  colorblind user and wastes horizontal space in a ~360px drawer.)
- **D-05: `@mention`-of-me = three redundant, colorblind-safe signals.** A mentioned-row uses a
  **solid left accent bar + a subtle background tint + an `@you` chip/icon**. The row must read
  as "you were mentioned" even if the color/tint is imperceptible — shape (bar) + position +
  glyph carry it independently of hue. **Color alone is never the sole signal** (user is
  colorblind). (Rejected: tint + bold-token only — both signals are subtle; bar-only — robust
  but the user preferred the richer treatment.)
- **D-06: `@session` inject renders as a system-style line, not a normal bubble.** A
  `SessionInject:true` message shows the sender's text followed by a divider, a keyboard/arrow
  glyph, and a "→ injected into terminal" caption — clearly signaling "this went to the agent,
  not just the humans." Shape + icon + caption = colorblind-safe. (Rejected: normal bubble +
  small "→ injected" chip — reads as an ordinary human message at a glance.)

### @mention autocomplete & @session confirm
- **D-07: `@session` pinned on top, visually set apart.** In the `@` autocomplete popover,
  `@session` is always the first item, in its own pinned/divided section with an agent
  (terminal) glyph, above the live human participants. It never scrolls out of view while
  filtering and is unmistakably "the one that talks to the agent." The popover is filterable and
  keyboard-navigable (arrows + Enter to confirm) per MENTION-01. (Rejected: inline alphabetical
  — `@session` scrolls away and reads as just another participant; pinned-but-same-styling —
  relies on position alone.)
- **D-08: Press-and-hold the Send button to inject (MENTION-03 affordance).** When `@session` is
  the target in the composer, the Send button switches to an "Inject" state requiring a
  **press-and-hold** (e.g. ~600ms with a fill animation) to fire the dedicated
  `MsgSessionInject 0x35` frame. A tap, a stray Enter, or an autocomplete keypress **never**
  injects — plain Enter only ever sends a normal chat message. This is the UI half of the
  Phase 153 structural guarantee. (Rejected: two-step confirm dialog — interruptive on every
  inject; Cmd/Ctrl+Enter chord — invisible learned affordance, easier to fire by reflex, weaker
  "deliberate" guarantee.)

### Unread / notification behavior
- **D-09: "Unread" accrues whenever the drawer isn't open-and-focused.** Messages count as
  unread when the drawer is closed, when a different session is being viewed, or when the window
  is backgrounded; opening + viewing the drawer clears the count. (Rejected: only-when-modal-
  closed — a message arriving while the drawer is open-but-unfocused would silently fail to
  badge.) Planner: confirm what focus/visibility state is reliably detectable in the Wails
  webview (this same logic gets reused on web in Phase 155).
- **D-10: Badge shows total unread + a distinct `@mention` state.** Both the chat toggle and the
  Hub session card show the **total unread count**; if any unread message `@mentions` the
  current user, the badge switches to a **distinct state** (accent + `@` glyph / different
  shape — colorblind-safe via the glyph, not color alone). (Rejected: two separate counts — busy
  on a small Hub card; single plain count — loses the NOTIF-01 "mention is visually distinct"
  requirement at the badge level.)

### Claude's Discretion
- **CHAT-04 day-separator mechanics** — the requirement locks "day separators stick to the top
  of the stream while scrolling"; the exact sticky implementation (CSS `position: sticky` vs a
  virtualizer-aware header) is the planner's call, but it must interoperate with
  `@tanstack/react-virtual`.
- **Composer auto-grow cap** — CHAT-03 locks "auto-grows (capped)"; the concrete max row count
  before internal scroll is discretion (`react-textarea-autosize` `maxRows`).
- **Avatar/identity coloring** — avatars are supplementary, but any color used to disambiguate
  authors must not be the *sole* identifier (alias text always present). Planner picks the
  derivation.
- **Empty / loading / first-message states** for the panel — standard treatments; planner's call.
- **`rehype-sanitize` schema** — start from the default GitHub schema; SEC-03 only requires that
  no `rehype-raw` is present and XSS payloads render inert. Planner may tighten, never loosen.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone / phase intent
- `.planning/notes/session-chat-discovery.md` — v4.1 Session Chat design record: human-to-human
  side-channel, one-way `@session` bridge, identity = tailnet ID + alias (both visible), and the
  explicit "UI placement: chat panel within the session modal" gray area this phase resolves
  (D-01).
- `.planning/ROADMAP.md` §"Phase 154: Desktop Chat UI" — the 5 success criteria (send/Enter +
  alias/ID/timestamp; `@` autocomplete with pinned `@session`; mention + unread-badge distinction;
  XSS payloads render inert; sticky day separators).
- `.planning/REQUIREMENTS.md` — CHAT-01..04, MENTION-01, NOTIF-01/02, SEC-03 (the locked
  requirement text for this phase).

### Phase 153 carry-forward into 154 (deferred UAT — MUST address)
- `.planning/phases/153-session-pty-bridge/153-VERIFICATION.md` `deferred[]` and
  `.planning/phases/153-session-pty-bridge/153-UAT.md` — the two items routed to Phase 154:
  (1) render the "→ injected into terminal" indicator for `SessionInject:true` messages (D-06);
  (2) the `@session` confirm UX that prevents an accidental Enter from injecting (D-08). Both
  must be exercised in Phase 154's UAT.
- `.planning/phases/153-session-pty-bridge/153-CONTEXT.md` — the inject contract (dedicated
  `MsgSessionInject` verb is the only PTY-write path; server-side RW gate; sanitizer invariant).

### Upstream server/protocol contracts (locked — read before wiring the UI)
- `frontend/src/lib/relayClient.ts` — frame constants `MSG_PRESENCE 0x32`, `MSG_TYPING 0x33`,
  `MSG_ALIAS_SET 0x34`; `MsgChat 0x30` / `MsgChatSend 0x31` are **Phase 154 dispatch stubs** to
  be wired here; `ServerFrame` union, `parseServerFrame`, `RelayClient` callbacks, and the
  `encode*Frame` builders the composer extends. `MsgSessionInject` is `0x35`.
- `internal/relay/protocol.go` — `ChatMessage{ AuthorID, AuthorAlias, SchemaVersion,
  SessionInject }` shape the UI renders; the `0x30`–`0x35` frame range.
- `internal/daemon/chat.go` — `ChatStore` (`NewChatStore(baseDir, sessionID)`, `AppendMessage`,
  `Export`, cap/`ErrChatCapReached`, `SessionInject` marker in `Export`) — the persistence/
  late-join scrollback source this UI displays.
- `internal/relay/server.go` / `internal/webserver/server.go` — the WS read-pump dispatch
  (`MsgChatSend`, inject case) the desktop client talks to; same paths Phase 155 reuses for web.

### Frontend mount point & reuse
- `frontend/src/components/Hub/HubInteractiveModal.tsx` — currently mounts `TerminalPanel`
  full-bleed; this is where the chat drawer + toggle attach (D-01/D-02).
- `frontend/src/components/Hub/SessionCard.tsx` — the Hub card that gets the unread badge (D-10).
- `agenthub-v4.0-redesign/AgentHub.Chat.Session.standalone.html` — the #79 React design
  prototype / **design comp** (avatars, composer, mention popover). Per project convention,
  render and compare the running ChatPanel against this comp during UI verification — token/ARIA
  checks alone are not sufficient.

### npm packages (from v4.1 architecture notes in STATE.md)
- `@tanstack/react-virtual` v3.14.3 — virtualized message list.
- `react-textarea-autosize` v8.5.9 — auto-growing composer (CHAT-03 cap).
- `react-markdown` 10.1.0 + `remark-gfm` ^4.0.1 + `rehype-sanitize` ^6.0.0 — already in
  `frontend/package.json`; the locked safe-Markdown stack (no `rehype-raw`).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/lib/relayClient.ts` — `RelayClient`, `parseServerFrame`, `ServerFrame` union,
  and `encodeTypingFrame`/`encodeAliasSetFrame` builders. The chat composer adds `MsgChatSend`
  and `MsgSessionInject` encoders and a `chat`/`presence`/`typing` render path off the existing
  callback surface. `MsgChat 0x30` / `MsgChatSend 0x31` are documented as **Phase 154 stubs**.
- `internal/daemon/chat.go` `ChatStore.Export` already emits the inject marker for
  `SessionInject:true` — the UI mirrors that semantic visually (D-06).
- `frontend/src/components/Hub/HubInteractiveModal.tsx` — the existing modal shell + `TerminalPanel`
  mount; the drawer + toggle slot in here without `HubInteractiveModal` owning a `RelayClient`
  (`TerminalPanel` owns it internally — the chat panel needs its own subscription path; planner
  to decide shared-vs-separate client).
- `frontend/src/components/Hub/SessionCard.tsx` — card surface for the Hub unread badge.

### Established Patterns
- Frame constants + `encode*Frame` builder convention in `relayClient.ts` (mirror for chat-send
  and inject frames).
- Hub modal + `TerminalPanel` full-bleed layout (D-02 is now overlay mode — the drawer floats over
  the terminal and does NOT use `TerminalPanel`'s resize path; the PTY is left untouched on toggle).
- Colorblind-safe signaling is a standing project rule — every status/notification cue carries a
  non-color channel (shape/glyph/text). D-05, D-06, D-10 all follow it.

### Integration Points
- New `MsgChatSend` dispatch case (client→server) + `MsgChat` render case (server→client)
  wiring the existing stubs to live frames.
- Drawer open/close → `TerminalPanel` resize → PTY reflow (D-02).
- Unread state derived from drawer focus/visibility (D-09) → badge on toggle (D-10) and a
  Hub-card badge channel up to `SessionCard`.
- Press-and-hold gesture → emits `MsgSessionInject 0x35` (the only frame the server injects on).

</code_context>

<specifics>
## Specific Ideas

- The press-and-hold Send button is the **load-bearing safety affordance** for MENTION-03 on the
  client side: only a deliberate hold emits `MsgSessionInject`; a tap/Enter/autocomplete-Enter
  cannot. Pair this with the server's "only `0x35` writes to the PTY" guarantee in the threat
  model / UAT.
- `@mention`-of-me and the inject indicator both deliberately use **3+ redundant channels**
  (shape + position + glyph/caption) so they survive the user's colorblindness — never rely on
  hue.
- Compare the rendered ChatPanel against the #79 standalone prototype during verification, not
  just token/ARIA checks (per the project's "verify UI against the design comp" rule).

</specifics>

<deferred>
## Deferred Ideas

- **Web-share chat UI, Markdown export, cross-surface parity gate** — Phase 155 (reuses this
  `ChatPanel.tsx`). PARITY-01 is release-blocking and lands there.
- **Native tray / OS notification on `@mention` when not viewing the chat** (NOTIF-F1) — deferred;
  needs a cross-surface story (web has no tray).
- **Triple-backtick code-block heuristic rendering** (CHAT-F1) — out of this phase's Markdown
  scope.
- **Remember-last drawer state / auto-open on unread** — considered for D-03, not adopted.

None of the above are scope creep from this discussion — they are roadmap-sequenced into later
phases or explicitly optional.

</deferred>

---

*Phase: 154-desktop-chat-ui*
*Context gathered: 2026-06-26*
