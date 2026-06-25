# Feature Research

**Domain:** Session-scoped human-to-human chat side channel (v4.1 Session Chat)
**Researched:** 2026-06-25
**Confidence:** MEDIUM (cross-checked against design-doc agreements + websearch patterns; LOW-confidence web results elevated to MEDIUM where corroborated by multiple independent references)

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Enter = send, Shift+Enter = newline** | Universal convention: Slack, Teams, Discord, WhatsApp all agree. Any dev who uses chat arrives with this hardwired. Swapping the defaults is a fast way to generate complaints. | LOW | Keyboard event handler on textarea/contenteditable. Send button as mouse fallback. No server work. |
| **@mention autocomplete popover** | Standard in every team chat tool since Slack introduced it. Typing `@` with no popover feels archaic. | MEDIUM | `@` triggers inline popover above cursor, filtered by typed chars. Up/Down to navigate, Enter to select, Escape to dismiss. `react-mentions-ts` or DIY. Participants list = connected peers fetched from relay. |
| **@session as a special autocomplete target** | The core differentiating action for this product. It must be discoverable via the same `@` flow, not a separate command syntax that users have to memorize. | LOW (if @mention is done) | Pre-seed `@session` as the first item (pinned at top) in the mentions list with a distinct icon (e.g. terminal icon). One-way only: inject message text into PTY stdin. No round-trip. |
| **Presence: who is connected right now** | In a session-scoped chat there are at most a handful of participants. Users expect to see who else is in the room. Missing this is disorienting. | LOW | Binary connected / disconnected derived from existing relay WebSocket lifecycle. No separate heartbeat required. Show as avatar dots or a "2 connected" chip in the chat header. |
| **Typing indicator ("Ken is typing...")** | Users expect this from every real-time chat. Without it the chat feels like asynchronous email. | MEDIUM | Emit `typing` event max once per 3 s while user is composing. Emit `stopped` after 5 s with no keystroke. Relay fans both out. Client shows "X is typing..." until `stopped` arrives or 6 s timeout. Volatile: never stored, never replayed. Uses the same relay fan-out as the terminal, routed on a new message type. |
| **Late-join scrollback (full history on open)** | Collaborators connecting mid-session expect to catch up on what was said. Missing this means every late joiner is flying blind. | MEDIUM | Daemon stores messages. On chat-open the client fetches full thread from daemon REST endpoint. Scroll to bottom on first load. CSS `overflow-anchor` or JS anchor element to pin to bottom as new messages arrive; stop auto-scrolling if user has scrolled up intentionally. Session-scoped threads are expected to be short (<500 messages), so load-all is fine; no pagination needed for v1. |
| **Message timestamps** | Every chat interface shows when each message was sent. Absence feels broken. | LOW | Show wall-clock time (HH:MM) on each message. Hover reveals full ISO-8601 datetime. Group consecutive messages from the same sender within ~60 s under a single avatar (Slack-style compact grouping) to reduce visual noise. |
| **Day separators** | Slack, Discord, Discourse all do this. Sessions that run overnight or resume across a day boundary need the visual break. | LOW | Centered horizontal rule with "Today" / "Yesterday" / "Wed Jun 24" label. Sticky at top while scrolling through history. For same-day sessions (the common case) this separator is invisible. |
| **Identity display: alias + tailnet ID** | Agreed design decision. Trust comes from the tailnet ID (system-verified); legibility comes from the alias (self-chosen). Both must be visible. | LOW | Sender line: `**alias** (tailnet-node-id)`. Alias is set on first-use with a prompt (max 32 chars, stored per-tailnet-peer in daemon). No login screen, no registration — preserves AgentHub's zero-config ethos. |
| **Unread count badge when chat is not open** | If a collaborator sends a message while the user is looking at the terminal, they need a signal to look at the chat panel. Without this, chat messages are silently lost. | MEDIUM | Numeric badge on the chat toggle button (desktop) and on the Hub session card (when not in the modal). Clear the count when chat panel is visible/focused. @mention gets an additional highlight color (orange/amber vs neutral gray for plain unreads). Depends on existing Hub session-card component. |

---

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required for the baseline, but add meaningful value.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **@session -> PTY prompt injection (one-way bridge)** | Unique to AgentHub: lets collaborators co-pilot the AI agent without giving everyone keyboard control of the terminal. The "send this prompt to the agent" action is accessible to read-only web-share viewers. No other tool offers this in a terminal-session context. | MEDIUM | Reuses existing PTY stdin injection path. The relay must carry a `@session` message type to the daemon, which writes the message text to the session's PTY stdin. Only works for sessions the user has RW capability on -- enforce this gate. Display the injected text in the terminal verbatim; it appears in the agent's input line. Show an indicator in the chat (e.g. "-> injected into terminal") so the sender knows it was dispatched. |
| **Markdown thread export** | Lets teams keep a record of the collaborative session outside the tool. Important for AI-assisted coding flows where chat captures the decision trail. Rare in embedded session-scoped chat. | LOW-MEDIUM | Download button in chat panel header. Format: YAML frontmatter (session-id, session-name, agent, exported-at, participants with alias+id), then messages as `**Alias** (tailnet-node-id) [HH:MM:SS]` + body, day separators as `## YYYY-MM-DD`, @session messages marked with `> [injected into terminal]`. Follows conventions from ChatGPT/LLM export tools so the output drops into Obsidian/Logseq. |
| **@mention highlight for addressed participant** | When your alias is mentioned, your message row gets a different background (e.g. soft amber). Familiar from Slack/Discord but not all embedded chat tools implement it. Reduces the need to scan every message when returning to a busy thread. | LOW | CSS class toggle on message rows where text contains `@<my-alias>`. Client-side only; no server change. |
| **Sticky day-separator on scroll-back** | The separator label stays fixed at the top of the viewport as the user scrolls through history, providing temporal context without having to scroll back to find the separator. Slack does this; many tools do not. | LOW | CSS `position: sticky` on separator element inside the scrollable container. No server work. |

---

### Anti-Features (Scope Creep -- Explicitly Avoid in v1)

Features that seem reasonable but create more problems than they solve for this milestone.

| Feature | Surface Appeal | Why Avoid | What to Do Instead |
|---------|---------------|-----------|-------------------|
| **Emoji reactions** | Users ask for them in every chat tool. Fun, expressive. | Adds a reactions store to the daemon message model, complicates the export format, adds UI chrome, and provides zero functional value in an AI coding session context. Pure social layer. | Skip entirely for v1. File as v2 candidate if user demand emerges post-launch. |
| **Threads-within-threads (replies)** | Keeps related discussion together. Familiar from Slack. | The design discovery note explicitly called out the prototype's "structured message/thread history" as added complexity to design out. Nesting creates infinite depth questions, complicates the export format, and is overkill for small sessions with 2-5 participants. | Flat thread only. Use @alias mentions to signal who you're talking to. |
| **File uploads / attachments** | Natural to want to paste a screenshot or share a file. | AgentHub already ships a full-featured sandboxed file browser. Duplicating file transfer in the chat layer doubles the attack surface, adds binary storage to the daemon message store, and has no clear advantage over "go to the Files tab." | Point users to the existing file browser for file sharing. |
| **Editing or deleting sent messages** | Standard in Slack/Discord. Users expect it. | Requires message versioning (or tombstoning) in the daemon store, "edited" indicators in the UI, and synchronization across all connected clients. For an ephemeral session-scoped thread this complexity is not justified. The thread dies with the session anyway. | Do not implement. If a user makes a mistake, they send a correction message. |
| **Read receipts ("Seen by X")** | Shows accountability for message consumption. | Adds per-participant read-cursor tracking to the daemon, adds "seen" events to the relay protocol, and creates social pressure in a context (AI coding) where the participant may simply be watching the terminal. | Trust that if the person is connected (presence dot), they can see the message. |
| **Full-text search of chat history** | Useful in long-running persistent channels. | Chat thread is session-scoped and short-lived. Sessions rarely produce thousands of messages. The thread dies with the session. Browser Ctrl+F over the rendered chat is sufficient. Adding daemon-side search indexing is pure over-engineering for v1. | Scrollback + browser find-in-page. Revisit if session threads grow large in practice. |
| **Persistent notifications (push / OS-level)** | Users expect to be notified even when the app is background. | AgentHub has no push notification infrastructure. Building it for chat alone is a large out-of-scope dependency. Tray-icon badge for the app is the extent of OS-level notification that already exists. | In-app unread badge (table stakes above) is sufficient. AgentHub is a tray-resident tool; the app is always "nearby." |
| **Message quoting / reply-to-specific-message** | Familiar from modern messengers (iMessage, WhatsApp, Telegram). | Requires message IDs in the reply payload, quoted-message rendering in the thread, and link-back UX. Significant UI and data-model complexity for a context where the thread is short and linear. | Use @alias mention to address a specific person. |
| **Rich inline Markdown rendering in chat bubbles** | Developers love Markdown. | Requires a full MD renderer in the message bubble, sanitization to prevent XSS from user-generated content, and CSS overrides. The input box is plain text + @mentions. Rendering full Markdown in received messages adds significant complexity and security surface. | Render plain text. Distinguish code by wrapping in a code block if the message body is a triple-backtick fenced block (simple heuristic, no full parser). Full rich MD is in scope for a v1.x polish pass after core is proven. |
| **Giphy / emoji picker** | Familiarity from consumer chat apps. | Adds external API dependency (Giphy) and heavyweight picker UI to a developer productivity tool. | Standard OS emoji input (Ctrl+Cmd+Space on macOS) works in the textarea already. |
| **Direct messages between participants** | Private side channels. | Chat is session-scoped. Private conversations between participants break the "shared context" model and create a separate message store and routing layer. | Out of scope. Participants who want a private channel already have Tailscale-connected machines -- they can DM natively. |
| **Agent-as-chat-author (round-trip replies)** | The original prototype design. Agent writes into the chat thread as a participant. | Explicitly designed out in discovery. The PTY output is a paint stream (TUI redraws, ANSI escape sequences), not discrete messages. Segmenting it into clean turns is the same order-of-magnitude problem as the v4.0 mini-preview garble (#96), but harder. Introduces the full agent segmentation problem. | One-way bridge via @session only. Agent replies appear in the terminal, not the chat. This constraint is the foundation of the feasibility argument for v1. |
| **Chat archive that outlives the session** | Persistent team history. | Explicitly out of scope in discovery. "Thread dies when the session is deleted" was an agreed constraint. Persistent cross-session archives are a different product (a team knowledge base). | Thread is downloadable as Markdown before deletion. That is the preservation mechanism. |

---

## Feature Dependencies

```
[Daemon message store]
    +--required by--> [Late-join scrollback]
    +--required by--> [Markdown export]
    +--required by--> [Message timestamps + day separators]

[WebSocket relay extension (new message type)]
    +--required by--> [Message fan-out to all participants]
    +--required by--> [Typing indicators]
    +--required by--> [Presence: connected/disconnected events]

[Participant identity model (tailnet ID + alias)]
    +--required by--> [@mention autocomplete popover (member list)]
    +--required by--> [Identity display on messages]
    +--required by--> [@mention highlight ("addressed me")]

[@mention autocomplete popover]
    +--enables-----> [@session PTY injection bridge]

[Chat panel UI (both surfaces)]
    +--required by--> [Unread badge on Hub card] (card must know if panel is open)

[Existing PTY stdin injection path]
    +--reused by---> [@session -> PTY bridge] (no new injection work needed)

[Existing Hub session-card component]
    +--extended by-> [Unread count badge]
```

### Dependency Notes

- **Daemon message store blocks almost everything**: timestamps, day separators, scrollback, and Markdown export all depend on a persistent store. It is the first thing to implement.
- **Relay extension gates real-time features**: fan-out, typing, and presence all need a new message type in the existing WebSocket relay. This is the second major work item, done in parallel with UI scaffolding.
- **Identity model (alias) gates @mention**: the alias must be stored and resolvable before autocomplete or identity display can work.
- **@mention must be working before @session bridge**: `@session` is just a special entry in the mention list; the autocomplete mechanism must exist first.
- **Cross-surface parity is release-blocking** (standing AgentHub rule): every table-stakes feature must work on both desktop GUI (Wails/React) and web-share browser (served HTML/React). There is no desktop-only shortcut.

---

## MVP Definition

### Launch With (v1 -- all table stakes + agreed differentiators)

- [ ] Enter = send, Shift+Enter = newline
- [ ] @mention autocomplete popover (participants + pinned @session entry)
- [ ] @session -> PTY stdin injection (one-way, RW-cap gated, "-> injected" confirmation in chat)
- [ ] Presence: connected/disconnected indicator for each participant
- [ ] Typing indicators (debounced, volatile, never stored)
- [ ] Late-join scrollback: full thread on open, scroll-to-bottom, respects user scroll
- [ ] Message timestamps (HH:MM, hover for full datetime)
- [ ] Day separators ("Today" / "Yesterday" / date)
- [ ] Identity display: alias + tailnet ID on each message
- [ ] Unread badge on chat toggle button and Hub card, cleared on focus; @mention gets distinct color
- [ ] Markdown thread export (download .md with YAML frontmatter)
- [ ] Cross-surface parity: identical feature set on desktop GUI and web-share browser

### Add After Validation (v1.x)

- [ ] Sticky day separators on scroll (CSS `position:sticky`) -- cosmetic polish, add in first sprint after launch
- [ ] @mention highlight for the current user's alias -- low-effort, high-signal, add with first polish pass
- [ ] Triple-backtick code block rendering (heuristic, no full MD parser) -- developers paste code; simple pre/code wrapping adds value
- [ ] Alias edit capability without session restart -- if user mis-types alias they need a way to fix it

### Future Consideration (v2+)

- [ ] Rich inline Markdown rendering -- only after confirming users want it; adds security surface
- [ ] Emoji reactions -- only if post-launch user feedback shows demand
- [ ] Reply-to-specific-message quoting -- only if threads grow long enough that linear attribution breaks down

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Daemon message store | HIGH | MEDIUM | P1 -- foundation |
| WebSocket relay extension | HIGH | MEDIUM | P1 -- foundation |
| Enter=send / Shift+Enter=newline | HIGH | LOW | P1 |
| Identity: alias + tailnet ID | HIGH | LOW | P1 |
| @mention autocomplete | HIGH | MEDIUM | P1 |
| @session PTY bridge | HIGH | LOW (reuses existing) | P1 |
| Late-join scrollback | HIGH | MEDIUM | P1 |
| Message timestamps + day separators | MEDIUM | LOW | P1 |
| Presence indicators | MEDIUM | LOW | P1 |
| Typing indicators | MEDIUM | MEDIUM | P1 |
| Unread badge (chat button) | HIGH | MEDIUM | P1 |
| Markdown export | MEDIUM | LOW | P1 |
| Unread badge on Hub card | MEDIUM | MEDIUM | P2 |
| @mention highlight (current user) | MEDIUM | LOW | P2 |
| Sticky day separators | LOW | LOW | P2 |
| Code-block heuristic rendering | MEDIUM | LOW | P2 |

**Priority key:**
- P1: Must have for launch (all table stakes + the two agreed differentiators: @session bridge, Markdown export)
- P2: Should have, add in first polish sprint post-launch
- P3: Nice to have, future consideration

---

## Competitor Feature Analysis

Context: AgentHub's session chat is not a general-purpose team chat product. The relevant comparison space is embedded, session-scoped, small-N collaborative tools -- not Slack/Discord at scale.

| Feature | Slack (reference) | Discord (reference) | tmux shared-session (prior art) | AgentHub v4.1 approach |
|---------|------------------|--------------------|---------------------------------|----------------------|
| Enter=send | Yes (default) | Yes (default) | N/A | Yes (Enter=send) |
| @mention autocomplete | Full member graph | Full member graph | N/A | Session participants only |
| Presence | Rich (active/away/DND/offline) | Rich | N/A | Binary: connected/disconnected |
| Typing indicators | Yes | Yes | No | Yes (debounced, volatile) |
| History on join | Full channel history | Configurable | No | Full session thread |
| Unread badge | Yes, per-channel | Yes | No | Yes, per-session-card |
| Day separators | Yes | Yes | No | Yes |
| Identity | Username + avatar | Username + avatar | Unix user | Tailnet peer ID + alias |
| Agent bridge | No | No | N/A (tmux paste) | @session -> PTY stdin (unique) |
| Markdown export | No (paid data export only) | No | No | Yes (unique in this space) |
| Reactions | Yes | Yes | No | Anti-feature: skip v1 |
| Thread replies | Yes | Yes (Forum channels) | No | Anti-feature: flat thread |
| File uploads | Yes | Yes | No | Anti-feature: use file browser |
| Editing/deleting | Yes | Yes | No | Anti-feature: skip |
| Read receipts | DMs only | No | No | Anti-feature: skip |
| Search | Full-text | Full-text | No | Anti-feature: browser Ctrl+F |

**Key takeaway:** AgentHub's session chat intentionally covers the full table-stakes UX that users associate with collaborative chat (enter=send, @mentions, presence, typing, timestamps, scrollback, unread) while eliminating all the social-layer features that are irrelevant or harmful in a small-team AI coding context. The `@session` bridge and Markdown export are the two genuine differentiators with no equivalent in the reference tools.

---

## Sources

- Discovery/design record: `.planning/notes/session-chat-discovery.md` (HIGH confidence -- agreed design)
- Project context: `.planning/PROJECT.md` (HIGH confidence -- project source of truth)
- Web search: chat UX Enter/Shift+Enter conventions, Discourse Meta discussion (MEDIUM, cross-checked across Slack/Teams/Discord/ChatGPT)
- Web search: @mention autocomplete patterns, CSS-Tricks "So You Want to Build an @mention Autocomplete Feature", `react-mentions-ts` library (MEDIUM)
- Web search: typing indicator debounce patterns, WhatsApp / Socket.IO patterns, DEV Community articles (MEDIUM, multiple independent sources)
- Web search: late-join scrollback, CSS overflow-anchor, Vonage chat pagination guide (MEDIUM)
- Web search: timestamp/day-separator conventions -- Slack, Discourse, Discord (MEDIUM)
- Web search: Markdown export format conventions for chat transcripts, ChatGPT exporters (MEDIUM)
