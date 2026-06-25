# Project Research Summary

**Project:** AgentHub v4.1 — Session Chat
**Domain:** Per-session human-to-human chat side channel (Go daemon + React 19 frontend)
**Researched:** 2026-06-25
**Confidence:** HIGH (architecture from direct codebase inspection; stack from confirmed go.mod / package.json; features/pitfalls from design record + cross-checked patterns)

## Executive Summary

AgentHub v4.1 adds a real-time human-to-human chat thread to each session so participants connected via Tailscale can coordinate without leaving the tool. The scope was deliberately reframed away from the original prototype, which assumed the agent would participate as a chat author. That premise fails against the real PTY: agent output is a TUI paint stream with no discrete turns, not a message API. The agreed design is a one-way bridge — humans chat with each other; `@session` injects a message into the agent's PTY stdin; the agent's reply appears in the terminal only. This constraint makes the feature buildable in a single milestone without importing a hard AI-segmentation problem.

The implementation strategy is conservative and proven: zero new Go modules, two small npm additions (`@tanstack/react-virtual`, `react-textarea-autosize`), and hand-rolled `@mention` (~80 lines). Every capability the feature needs — WebSocket relay fan-out (`coder/websocket` v1.8.14), Tailscale peer identity (`tailscale.com/client/tailscale.LocalClient.WhoIs`), PTY stdin injection, and Markdown rendering (`react-markdown` + `rehype-sanitize` + `remark-gfm`) — is already in `go.mod` or `frontend/package.json`. The primary new work is a JSONL-per-session message store (stdlib), relay protocol extension (new frame-type bytes on the existing WS connection), and the React chat panel shared across both surfaces.

The dominant risks are security in nature: a read-only cap holder sending chat messages or `@session` injections via the new input path; control-character injection through `@session` into PTY stdin; and XSS from Markdown rendering on the web surface. All three are definitively preventable with targeted enforcement at the daemon layer, but each requires explicit test coverage rather than assumption. Cross-surface parity (desktop GUI and web-share browser must behave identically) is a standing release-blocking rule that must be wired into every phase, not addressed at the end.

## Key Findings

### Recommended Stack

The stack adds nothing to Go and only two small, focused npm packages. All five new server-side capabilities (message store, relay fan-out, Tailscale identity, PTY injection, Markdown export) use modules already present in `go.mod`. The JSONL append-only file store (one per session, `~/.config/agenthub/chats/<sessionID>.jsonl`) was chosen over `modernc.org/sqlite` and `bbolt` because the data is bounded in volume, needs no cross-session queries, and a mutex-guarded `os.File` append is simpler and dependency-free for a sequential list. The relay extension uses `coder/websocket`'s existing `MessageText`/`MessageBinary` frame split — chat rides the same WS connection as PTY, eliminating duplicate auth plumbing.

**Core technologies:**
- `coder/websocket` v1.8.14 (already in go.mod): relay WS — add `MessageText` branch for chat frames; no new dep
- `tailscale.com` v1.98.3 (already in go.mod): peer identity via `LocalClient.WhoIs(ctx, remoteAddr)` at WS upgrade
- stdlib `encoding/json` + `os` + `sync.Mutex`: JSONL message store — zero new Go dep
- `@tanstack/react-virtual` v3.14.3 (NEW): virtualized message list with React 19 `useFlushSync: false`
- `react-textarea-autosize` v8.5.9 (NEW): auto-growing composer; 1.3 KB gzipped, ships TS types
- `react-markdown` v10.1.0 + `rehype-sanitize` + `remark-gfm` (already installed): message bubble rendering
- Hand-rolled `@mention` (~80 lines TS): `react-mentions` is 3 years unmaintained; React 19 untested; bounded participant list makes a library unnecessary

### Expected Features

**Must have (table stakes — v1 launch):**
- Enter = send, Shift+Enter = newline
- `@mention` autocomplete popover (session participants + pinned `@session` entry)
- Presence indicator (connected/disconnected per participant)
- Typing indicators (debounced, volatile, never stored)
- Late-join scrollback (full thread on open, scroll-to-bottom, respects user scroll)
- Message timestamps (HH:MM, hover → full ISO-8601) + day separators
- Identity display: alias + tailnet ID on each message
- Unread badge on chat toggle button and Hub session card; `@mention` gets distinct color
- Cross-surface parity: every feature works identically on desktop GUI and web-share browser

**Should have (differentiators — v1 launch):**
- `@session` → PTY stdin injection (one-way, RW-cap gated, "→ injected into terminal" indicator) — unique to AgentHub
- Markdown thread export (download `.md` with YAML frontmatter)

**Defer to v1.x polish:**
- Sticky day separators (CSS `position: sticky`)
- `@mention` highlight for current user's alias
- Triple-backtick code-block heuristic rendering

**Anti-features (explicitly out for v1):** emoji reactions, reply threading, file uploads, edit/delete, read receipts, full-text search, agent-as-chat-author, cross-session persistent archive

### Architecture Approach

The architecture integrates with the existing daemon/relay/webserver/PTY stack through three new constructs: a `daemon.ChatStore` (JSONL file, per-session, deleted with the session), protocol extensions to `internal/relay/protocol.go` (new frame-type byte constants 0x30–0x34), and a `Hub.SetChatHandler` callback that decouples relay fan-out from daemon persistence without an import cycle. Both client surfaces use the same `ChatPanel.tsx` React component and the same relay frame types.

**Major components:**
1. `internal/daemon/chat.go` (NEW) — `ChatStore`: JSONL append, in-memory slice, `Messages()` for late-join replay, `Export()` for Markdown, `Delete()` on session teardown
2. `internal/relay/protocol.go` (MODIFIED) — `MsgChat`/`MsgChatSend`/`MsgPresence`/`MsgTyping`/`MsgAliasSet`; `ChatMessage` wire struct with stable `AuthorID` + snapshot `AuthorAlias`
3. `internal/relay/hub.go` (MODIFIED) — `BroadcastChat`, `BroadcastPresence`, `SetChatHandler` (callback pattern mirrors existing `resizeFn`)
4. `internal/relay/server.go` + `internal/webserver/server.go` (MODIFIED) — `MsgChatSend` dispatch in read pump; `lc.WhoIs` at WS upgrade (webserver only; relay uses `"local"` for loopback)
5. `internal/daemon/engine.go` (MODIFIED) — `chatStores map[string]*ChatStore`; `KillSession` calls `store.Delete()`
6. `frontend/src/components/Hub/ChatPanel.tsx` (NEW) — single React component shared by desktop modal and web SPA

### Critical Pitfalls

1. **RO cap holders posting chat / triggering `@session`** — Check `!sub.ReadOnly` and explicit write-perm before any `MsgChatSend` processing and before every `session.Write()` call in the daemon handler. Never rely on frontend button suppression alone. Grep all `session.Write` callsites added during the chat phase.

2. **Control characters in `@session` PTY injection** — Strip all C0 controls (`\x00`–`\x1f` except space/tab), collapse embedded newlines to a single space, strip CSI/OSC escape sequences, append exactly one `\n`. Apply sanitizer in the daemon handler, not in relay or UI. Unit-test with a newline/ctrl-char corpus.

3. **XSS via Markdown rendering on the web surface** — Never add `rehype-raw`; use `remark-gfm` only (same config as Phase 120 FileBrowserTab, a prior explicit security decision). Test by rendering `<img src=x onerror=alert(1)>` and confirming no `onerror` in the DOM.

4. **`@session` wired as Wails RPC instead of relay message** — Wails RPC silently does nothing on the web surface (cross-surface parity is release-blocking). Implement as `MsgChatSend` with `sessionInject: true`; same relay message on both surfaces; daemon receives and writes PTY.

5. **Local owner vs same-machine web client indistinguishable** — Define disambiguation rule before identity phase: Wails webview = `TailnetID "local"`; local browser = `WhoIs` node key; UI merges by `TailnetID` into one presence entry.

## Implications for Roadmap

### Phase 1: Message Schema + ChatStore
**Rationale:** Everything is blocked without the persisted message format; schema changes here ripple into all subsequent phases.
**Delivers:** `daemon.ChatStore` (JSONL, `Messages()`, `Export()`, `Delete()`); `ChatMessage` wire struct in `protocol.go`; `chatStores` map in `SessionEngine`; `KillSession` teardown; REST history + export endpoints on relay mux (desktop) and webserver (cap-gated web); concurrent-write test under `-race`; 10 000-message hard cap enforced in `AppendMessage`.
**Addresses:** Late-join scrollback foundation; Markdown export foundation; daemon restart survival.
**Avoids:** Unbounded store growth; race conditions; reconnect duplication; RO-cap bypass (cap check wired before any Append).
**Research flag:** Standard Go stdlib patterns — skip research phase.

### Phase 2: Relay Protocol Extension + Hub Fan-Out + Identity
**Rationale:** Real-time delivery and identity attribution must both be in place before any message is stored with authorship.
**Delivers:** Frame-type constants 0x30–0x34; `BroadcastChat`/`BroadcastPresence`/`SetChatHandler` on Hub; `TailnetID`/`Alias` on Subscriber; `lc.WhoIs` at WS upgrade (webserver); `"local"` identity for loopback (relay); `MsgChatSend` + `MsgTyping` + `MsgAliasSet` dispatch in both read pumps; typing server-side TTL (5 s auto-expire + clear on WS close); 500 ms client throttle spec; 2/sec server-side rate limit on `MsgTyping`.
**Addresses:** Presence; typing indicators; live message fan-out; identity display; alias lifecycle.
**Avoids:** Presence flooding; stale typing after disconnect; alias spoofing; local owner vs same-machine web disambiguation.
**Research flag:** Standard patterns — skip research phase.

### Phase 3: `@session` PTY Bridge
**Rationale:** Depends on Phase 2 read pump dispatch; highest-security-risk capability deserves its own phase for isolation and test focus.
**Delivers:** Daemon `chatHandler` detects `"@session "` prefix; `extractSessionPrompt`; PTY sanitizer (C0 strip, newline collapse, CSI/OSC strip, single `\n` append); explicit write-perm check before `session.Write()`; `SessionInject: true` in broadcast; sanitizer unit tests with control-char corpus.
**Addresses:** `@session` → PTY injection (unique differentiator).
**Avoids:** PTY control-char injection; RO-cap `@session` bypass; Wails-RPC surface divergence (relay message path only).
**Research flag:** No new research needed. Planning must specify sanitizer test corpus explicitly.

### Phase 4: Desktop Chat UI
**Rationale:** Depends on Phases 1–3 for backend; frontend can scaffold in parallel with Phase 2 but wires end-to-end only after Phase 3.
**Delivers:** `ChatPanel.tsx` (TanStack Virtual message list; `react-textarea-autosize` composer; hand-rolled `@mention` popover; presence/typing indicators; unread badge; day separators; timestamps; `@session` "→ injected" indicator); `HubInteractiveModal.tsx` updated; `pnpm add @tanstack/react-virtual react-textarea-autosize`.
**Addresses:** All desktop table-stakes features; Enter=send; @mention autocomplete.
**Avoids:** XSS (`react-markdown` config — `remark-gfm` only); React 19 `useFlushSync: false`.
**Research flag:** TanStack Virtual chat example is published — skip research phase.

### Phase 5: Web-Share Chat UI + Cross-Surface Parity Gate
**Rationale:** `ChatPanel.tsx` is shared; wire into web SPA; verify cap-gated paths; cross-surface parity is release-blocking.
**Delivers:** `ChatPanel.tsx` in web SPA; cap-gated REST history + export verified on web surface; Markdown export download button; Playwright e2e for both desktop and web-share surfaces for all table-stakes features including `@session`; unread badge on Hub session card; parity checklist signed off.
**Addresses:** Markdown export (both surfaces); web-share `@session` injection verified; all cross-surface guarantees.
**Avoids:** `@session` silent failure on web; chat feature drift between surfaces; XSS on web-share.
**Research flag:** Needs planning attention on Playwright web-share surface setup (see MEMORY.md: browser UAT via isolated component harness; wails dev bridge has no PTY). Use isolated component harness pattern.

### Phase Ordering Rationale

- Store before transport before UI: JSONL schema is the contract everything serializes to; late schema changes require coordinated migration across three layers
- Identity in Phase 2 not Phase 4: `TailnetID` must be stamped on Subscriber before any message is stored; retrofitting after storage is wired requires a migration
- `@session` as a dedicated phase: highest-security-risk new capability; isolation prevents the sanitizer and cap check from being rushed as part of the UI phase
- Web UI last: shares `ChatPanel.tsx` built in Phase 4; parity gate belongs here as a wholistic cross-surface pass

### Research Flags

Phases with well-documented patterns (skip `--research-phase`): Phase 1, Phase 2, Phase 3, Phase 4.

Phases needing planning attention (not deep research, but explicit design decisions): **Phase 5** — Playwright web-share UAT harness has known gotchas documented in MEMORY.md; plan must specify exact mechanic for each parity check.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | go.mod + package.json confirmed; all capabilities already in-place; alternatives evaluated with specific version evidence |
| Features | MEDIUM | Design record (HIGH); web-search UX conventions (MEDIUM); feature boundaries from agreed discovery, not user research |
| Architecture | HIGH | Direct inspection of relay, hub, webserver, daemon, capability packages; component boundaries from actual code |
| Pitfalls | HIGH | All critical pitfalls derived from actual code paths (relay MC-03 line 269, engine.go:474, Phase 120 rehype-raw decision) |

**Overall confidence:** HIGH

### Gaps to Address

- **Alias uniqueness enforcement strategy:** reject vs auto-suffix — decide at Phase 2 plan time (reject is simpler)
- **Local owner vs same-machine web client:** disambiguation rule specified but not live-tested; confirm with two-tab UAT in Phase 5
- **`@session` UX confirmation timing:** 500 ms hold to prevent accidents — decide whether in Phase 3 or defer to v1.x
- **Typing indicator visual design:** must be visually distinct from Hub agent status dots; Phase 4 plan must specify exact icon + label choices

## Sources

### Primary (HIGH confidence)
- `/Users/ken/dev/agenthub/internal/relay/server.go` — ReadOnly enforcement (MC-03, line 269)
- `/Users/ken/dev/agenthub/internal/relay/hub.go` — subscriber model; BroadcastMeta pattern
- `/Users/ken/dev/agenthub/internal/webserver/server.go` — Tailscale bind; WhoIs pattern
- `/Users/ken/dev/agenthub/internal/daemon/engine.go` — session registry; KillSession; PTY write warning (line 474)
- `/Users/ken/dev/agenthub/internal/capability/capability.go` — HasPerm() whole-token split
- `/Users/ken/dev/agenthub/go.mod` — coder/websocket v1.8.14; tailscale.com v1.98.3; Go 1.26.3
- `/Users/ken/dev/agenthub/frontend/package.json` — react-markdown 10.1.0; rehype-sanitize; remark-gfm; React 19.2.x
- `.planning/notes/session-chat-discovery.md` — agreed design; one-way bridge; explicit out-of-scope decisions

### Secondary (MEDIUM confidence)
- `pkg.go.dev/tailscale.com/client/tailscale/apitype#WhoIsResponse` — Node + UserProfile fields confirmed
- `npmjs.com/@tanstack/react-virtual` — v3.14.3; React 19 `useFlushSync: false` documented
- `npmjs.com/react-textarea-autosize` — v8.5.9; 1.3 KB gzipped; ships TS types
- `npmjs.com/react-mentions` — v4.4.10; last published 3 years ago (confirms staleness)
- Web search: Enter=send conventions; @mention patterns; typing debounce; Markdown export formats

### Tertiary (context only)
- `agenthub-v4.0-redesign/AgentHub.Chat.Session.standalone.html` — original prototype; NOT the implementation model; informed what to design out

---
*Research completed: 2026-06-25*
*Ready for roadmap: yes*
