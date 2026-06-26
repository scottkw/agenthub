# Requirements: AgentHub v4.1 — Session Chat

**Defined:** 2026-06-25
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Milestone goal:** Give the humans connected to a session a side-channel chat thread inside AgentHub — so collaborators talk to each other (and pipe prompts to the agent) without leaving for Slack/Discord. Closes Issue #79.

> Design record: `.planning/notes/session-chat-discovery.md` · Research: `.planning/research/SUMMARY.md`

## v1 Requirements

Requirements for the v4.1 release. Each maps to a roadmap phase.

### CHAT — Messaging core

- [ ] **CHAT-01**: A user can send and receive messages in a per-session chat thread; Enter sends, Shift+Enter inserts a newline.
- [ ] **CHAT-02**: The message stream shows each message's author (alias + tailnet ID), a timestamp (HH:MM, full ISO-8601 on hover), and day separators between calendar days.
- [ ] **CHAT-03**: The composer auto-grows with input (capped), and message bodies render Markdown safely (`remark-gfm` only, sanitized — no raw HTML).
- [ ] **CHAT-04**: Day separators stick to the top of the stream while scrolling.

### IDENT — Identity

- [ ] **IDENT-01**: Each participant is identified by their tailnet ID and a self-chosen alias, both visible to all participants.
- [x] **IDENT-02**: A user can set and change their alias; the local owner and a same-machine web client resolve to a single, correctly-disambiguated participant.

### MENTION — Mentions & the agent bridge

- [ ] **MENTION-01**: Typing `@` opens an autocomplete popover over the session's participants (plus the pinned `@session` target), filterable and keyboard-navigable.
- [ ] **MENTION-02**: `@session <text>` injects the message into the agent's PTY as a prompt — one-way only (the agent's reply appears in the terminal, not the chat); gated to read/write-capability holders; sanitized before injection; the chat shows a "→ injected into terminal" indicator.
- [ ] **MENTION-03**: `@session` injection requires a deliberate confirm step (e.g. a short press-and-hold) to prevent accidental prompts into the agent.

### PRESENCE — Liveness

- [ ] **PRESENCE-01**: Each participant's presence (connected / disconnected) is shown to all participants.
- [ ] **PRESENCE-02**: Typing indicators show when another participant is composing (debounced, volatile, never stored, with a server-side TTL so they clear on abrupt disconnect).

### PERSIST — Durability

- [x] **PERSIST-01**: The chat thread is persisted by the daemon for the session's life and survives daemon/app restart.
- [x] **PERSIST-02**: A participant joining late loads the full thread scrollback.
- [x] **PERSIST-03**: The thread is deleted when its session is deleted; a hard per-session message cap bounds store growth.

### EXPORT — Export

- [ ] **EXPORT-01**: A user can download a chat thread as a Markdown file, from both the desktop GUI and the web-share surface.

### NOTIF — In-app notifications

- [ ] **NOTIF-01**: An in-app unread badge appears on the chat toggle and the Hub session card when there are unread messages; an `@mention` of the current user is visually distinct.
- [ ] **NOTIF-02**: Messages that mention the current user's alias are highlighted in the stream.

### PARITY — Cross-surface

- [ ] **PARITY-01**: Every Session Chat feature behaves identically on the desktop GUI and the web-share browser surface (release-blocking).

### SEC — Security

- [ ] **SEC-01**: Read-only capability holders cannot post chat messages or trigger `@session` injection (enforced server-side, not by UI suppression).
- [ ] **SEC-02**: Text injected via `@session` into the PTY is sanitized — C0 control characters and terminal escape sequences stripped, newlines collapsed, exactly one trailing newline appended.
- [ ] **SEC-03**: Markdown message rendering cannot execute injected scripts/HTML (no `rehype-raw`; XSS payloads render inert) on either surface.

### INSTALL — Install links & distribution (orthogonal; `docs/install-links-fix.md`)

- [ ] **INSTALL-01**: The Linux install command on the Welcome screen works end-to-end — add `scripts/install.sh` (arch detect → fetch the latest GitHub-release tarball → verify SHA256 against `checksums.txt` → install the binary), point `WelcomeTab.tsx` at the working raw GitHub URL, add a `TESTING.md` manual item, and verify on a clean Linux box.
- [ ] **INSTALL-02**: The Welcome screen shows correct distribution strings — the winget command uses the real package id (`winget install scottkw.agenthub`) and the repo link reads `github.com/scottkw/agenthub`.
- [ ] **INSTALL-03**: The winget package is published in the catalog — complete the one-time `microsoft/winget-pkgs` first submission (provision `WINGET_TOKEN`, set `WINGET_FIRST_SUBMISSION=true`, trigger `distribute.yml`, shepherd the PR to merge, then reset the flag and remove `continue-on-error` from `submit-winget`) so `winget install scottkw.agenthub` installs on Windows. *(External dependency: Microsoft's `winget-pkgs` review/merge — completion is gated on PR acceptance.)*

## v2 / Future Requirements

Deferred — acknowledged but not in the v4.1 roadmap.

### Notifications

- **NOTIF-F1**: Native tray / OS notification when the user is `@mentioned` and not viewing that chat (reuses tray infra; needs a cross-surface story since the web surface has no tray).

### Rendering

- **CHAT-F1**: Triple-backtick code-block heuristic rendering in message bodies.

## Out of Scope

Explicitly excluded for v4.1 — anti-features from research and design decisions from discovery.

| Feature | Reason |
|---------|--------|
| Agent as a chat author / agent replies streamed into the thread | Requires segmenting the agent's TUI paint-stream into discrete messages — the hard problem discovery explicitly designed out; the only chat↔agent bridge is the one-way `@session` injection. |
| Agent tool-output cards rendered inside chat | Same root cause as above — depends on structured agent output the PTY doesn't provide. |
| Chat archive that outlives the session | Thread is scoped to the session's life by design; a searchable cross-session archive is a different product. |
| Emoji reactions | Scope creep; not core to session coordination. |
| Reply threading (threads-within-threads) | Over-complex for small, session-scoped teams. |
| File uploads / attachments | Storage + security surface; out of scope for a coordination side-channel. |
| Editing / deleting sent messages | Adds mutable-history complexity; export + ephemeral scope make it unnecessary for v1. |
| Read receipts | Presence + unread badges are sufficient; receipts add per-message bookkeeping. |
| Full-text search across threads | Threads are short-lived and session-scoped; not worth the index. |
| Direct messages between participants | Chat is session-scoped by definition. |

## Traceability

Which phase covers each requirement. Populated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CHAT-01 | Phase 154 | Pending |
| CHAT-02 | Phase 154 | Pending |
| CHAT-03 | Phase 154 | Pending |
| CHAT-04 | Phase 154 | Pending |
| IDENT-01 | Phase 152 | Pending |
| IDENT-02 | Phase 152 | Complete |
| MENTION-01 | Phase 154 | Pending |
| MENTION-02 | Phase 153 | Pending |
| MENTION-03 | Phase 153 | Pending |
| PRESENCE-01 | Phase 152 | Pending |
| PRESENCE-02 | Phase 152 | Pending |
| PERSIST-01 | Phase 151 | Complete |
| PERSIST-02 | Phase 151 | Complete |
| PERSIST-03 | Phase 151 | Complete |
| EXPORT-01 | Phase 155 | Pending |
| NOTIF-01 | Phase 154 | Pending |
| NOTIF-02 | Phase 154 | Pending |
| PARITY-01 | Phase 155 | Pending |
| SEC-01 | Phase 153 | Pending |
| SEC-02 | Phase 153 | Pending |
| SEC-03 | Phase 154 | Pending |
| INSTALL-01 | Phase 156 | Pending |
| INSTALL-02 | Phase 156 | Pending |
| INSTALL-03 | Phase 156 | Pending |

**Coverage:**

- v1 requirements: 24 total
- Mapped to phases: 24 (100% ✓)
- Unmapped: 0

---
*Requirements defined: 2026-06-25*
*Last updated: 2026-06-25 — traceability filled by roadmapper (v4.1 Session Chat)*
