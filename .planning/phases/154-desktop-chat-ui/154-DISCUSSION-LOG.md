# Phase 154: Desktop Chat UI - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-26
**Phase:** 154-desktop-chat-ui
**Areas discussed:** Panel placement & layout, Message thread visual style, @mention + @session confirm, Unread / notification behavior

---

## Panel placement & layout

### Q1 — How ChatPanel coexists with the terminal in the session modal

| Option | Description | Selected |
|--------|-------------|----------|
| Slide-over drawer (right) | Terminal full-bleed; chat toggle opens a right drawer; closing returns to full terminal | ✓ |
| Side-by-side split | Two always-visible columns (~65/35); costs terminal width | |
| Tab toggle (swap) | Terminal \| Chat tabs swap a full-bleed panel; can't watch terminal while chatting | |

**User's choice:** Slide-over drawer (right)

### Q2 — What happens to the terminal when the drawer opens

| Option | Description | Selected |
|--------|-------------|----------|
| Overlay (terminal unchanged) | Drawer floats over the right edge; no PTY resize; part of terminal covered | |
| Push (terminal shrinks) | Terminal column shrinks → PTY resize/reflow; nothing covered | ✓ |
| You decide | Defer to research/planning | |

**User's choice:** Push (terminal shrinks)
**Notes:** Accepts the per-open/close PTY reflow cost; planner must route through TerminalPanel's resize + max-wins arbitration.

### Q3 — Default drawer state on modal open

| Option | Description | Selected |
|--------|-------------|----------|
| Closed by default | Opens to full terminal; user opens drawer via badged toggle | ✓ |
| Remember last state | Persist + restore open/closed per user/session | |
| Open if unread exist | Auto-open when unread messages are waiting | |

**User's choice:** Closed by default

---

## Message thread visual style

### Q1 — Thread layout

| Option | Description | Selected |
|--------|-------------|----------|
| Slack-style avatar rows | Flat left-aligned rows; avatar + alias + timestamp header; collapse consecutive same-author | ✓ |
| Chat bubbles (own-right) | Own messages right-aligned in colored bubble; weaker signal for colorblind user; burns width | |

**User's choice:** Slack-style avatar rows

### Q2 — How a message that @mentions you is made distinct (colorblind-safe)

| Option | Description | Selected |
|--------|-------------|----------|
| Left accent bar + tint + icon | Solid left bar + background tint + @you chip — three redundant signals | ✓ |
| Background tint + bold @name | Tint + bold token; both subtle | |
| Left accent bar only | Pure shape/position signal; minimal | |

**User's choice:** Left accent bar + tint + icon
**Notes:** Must not rely on color alone (user is colorblind); shape + position + glyph carry it independently.

### Q3 — How a @session inject (SessionInject:true) renders

| Option | Description | Selected |
|--------|-------------|----------|
| System-style line + icon | Distinct system/meta row with terminal glyph + "→ injected into terminal" caption | ✓ |
| Normal bubble + badge | Ordinary row + small "→ injected" chip; reads as a regular message | |

**User's choice:** System-style line + icon

---

## @mention + @session confirm

### Q1 — @session placement in the autocomplete popover

| Option | Description | Selected |
|--------|-------------|----------|
| Pinned on top, visually set apart | Always first, own divided section + agent glyph, above humans | ✓ |
| Inline, sorted alphabetically | Mixed with humans; can scroll out of view | |
| Pinned on top, styled like the rest | First by position only, no section/divider | |

**User's choice:** Pinned on top, visually set apart

### Q2 — Deliberate-confirm gesture for @session inject (MENTION-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Press-and-hold the send button | Send becomes hold-to-inject (~600ms); tap/Enter never injects | ✓ |
| Two-step confirm dialog | Enter opens a confirm popover; second click injects; interruptive | |
| Enter sends, Cmd/Ctrl+Enter injects | Distinct chord; invisible learned affordance; weaker guarantee | |

**User's choice:** Press-and-hold the send button
**Notes:** Plain Enter only ever sends a normal chat message; only the hold emits MsgSessionInject 0x35 — the client half of the Phase 153 structural guarantee.

---

## Unread / notification behavior

### Q1 — What counts as "unread"

| Option | Description | Selected |
|--------|-------------|----------|
| When drawer not focused/open | Unread accrues when drawer closed, other session, or window backgrounded | ✓ |
| Only when modal closed/other session | Open drawer = "in" chat; misses messages arriving while open-but-unfocused | |
| You decide | Defer to planning | |

**User's choice:** When drawer not focused/open
**Notes:** Planner to confirm focus/visibility detectability in the Wails webview (reused on web in Phase 155).

### Q2 — Badge scope on chat toggle + Hub card

| Option | Description | Selected |
|--------|-------------|----------|
| Count all + distinct @mention state | Total unread count; distinct accent + @ glyph state when mentioned | ✓ |
| Two separate counts | Total + separate mention count; busy on small card | |
| Single plain count | Just total; mentions only distinct inside the thread | |

**User's choice:** Count all + distinct @mention state
**Notes:** @mention badge state colorblind-safe via the glyph, not color alone.

---

## Claude's Discretion

- CHAT-04 day-separator sticky implementation (must interoperate with @tanstack/react-virtual)
- Composer auto-grow max rows (react-textarea-autosize)
- Avatar/identity coloring (alias text always present; color never the sole identifier)
- Empty / loading / first-message panel states
- rehype-sanitize schema (start from default GitHub schema; may tighten, never loosen — no rehype-raw)

## Deferred Ideas

- Web-share chat UI + Markdown export + cross-surface parity gate → Phase 155
- Native tray / OS @mention notification (NOTIF-F1) — deferred, needs cross-surface story
- Triple-backtick code-block heuristic rendering (CHAT-F1) — out of Markdown scope
- Remember-last drawer state / auto-open on unread — considered for D-03, not adopted
