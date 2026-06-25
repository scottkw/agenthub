# Stack Research — v4.1 Session Chat

**Domain:** Per-session human-to-human chat side channel (Go daemon + React 19 frontend)
**Researched:** 2026-06-25
**Confidence:** MEDIUM (Go stdlib patterns: HIGH; npm versions: MEDIUM from pkg.go.dev + npmjs.com; Tailscale API: MEDIUM from official Go pkg docs)

This is a **subsequent-milestone STACK.md** — it covers ONLY new capabilities required for v4.1 Session Chat. The existing stack (Go 1.26.3 / Wails v2.10.2 daemon, React 19.2.x / TS / Vite 8 / pnpm frontend, xterm.js, coder/websocket v1.8.14 relay, tailscale.com v1.98.3, capability tokens, react-markdown 10.1.0 + rehype-sanitize + remark-gfm, @heroicons/react) is in-place and confirmed via go.mod and frontend/package.json. Do not re-survey or replace any of it.

---

## TL;DR

1. **Message store**: JSONL append-only file per session (`{data_dir}/chats/{session-id}.jsonl`). **No new Go dependency.** Stdlib `encoding/json` + `os` + `sync.Mutex`. Survives daemon restart, dies with session, trivially Markdown-exportable.

2. **Relay extension**: Binary frame = PTY data (unchanged). Text frame = JSON chat envelope. `coder/websocket` v1.8.14 **already in go.mod** handles both frame types — add one `MessageText` branch to the existing `handleSession` read loop. No second transport, no new library.

3. **Frontend chat UI**: Add two small npm packages (`@tanstack/react-virtual` v3.14.3 for the message list, `react-textarea-autosize` v8.5.9 for the composer). @mention is **hand-rolled** (bounded participant list; `react-mentions` is 3 years stale). Markdown rendering via `react-markdown` already installed.

4. **Tailscale identity**: `tailscale.com/client/tailscale.LocalClient.WhoIs(ctx, remoteAddr)` — **already in go.mod**. Returns `UserProfile.LoginName` (stable identity) and `UserProfile.DisplayName`. Call on WebSocket upgrade with `r.RemoteAddr`.

5. **Markdown export**: Server-side Go, `strings.Builder` + `fmt.Fprintf`. **No new dependency.**

---

## New Go Dependencies

**None required.** All five chat capabilities use modules already present in go.mod.

| Capability | Module (already in go.mod) |
|------------|---------------------------|
| Message store persistence | `encoding/json` + `os` + `sync` — stdlib |
| Chat fan-out over relay | `coder/websocket` v1.8.14 |
| Tailscale peer identity | `tailscale.com` v1.98.3 |
| Markdown export | `strings` + `fmt` + `io` — stdlib |
| Message routing | `net/http` mux — stdlib |

## New Frontend npm Dependencies

| Package | Version | Purpose | Why |
|---------|---------|---------|-----|
| `@tanstack/react-virtual` | `^3.14.3` | Virtualized message list | Has a dedicated chat/reverse-scroll example; React 19 compatible (set `useFlushSync: false`); headless — no CSS opinion; actively maintained; 1321 npm dependents |
| `react-textarea-autosize` | `^8.5.9` | Auto-growing composer textarea | 1.3 KB gzipped; ships own TS types; purpose-built; simple DOM wrapper with no React internals coupling |

### Already installed — no version change

| Package | Current Version | Role in Chat |
|---------|----------------|--------------|
| `react-markdown` | 10.1.0 | Render markdown in message bubbles |
| `rehype-sanitize` | ^6.0.0 | Sanitize user-supplied markdown content |
| `remark-gfm` | ^4.0.1 | GFM tables/strikethrough in messages |
| `@heroicons/react` | ^2.2.0 | Send button, presence dot, mention icon |

---

## Implementation Details by Capability

### (a) Daemon-side Message Store — JSONL, no new dep

One file per session at `{data_dir}/chats/{session-id}.jsonl`. Each line is a single JSON object terminated by `\n`:

```json
{"id":"<uuidv4>","ts":"2026-06-25T18:00:00Z","sender_id":"user@example.com","sender_alias":"alice","type":"message","content":"hello @bob"}
```

Message types: `"message"` (normal chat), `"system"` (join/leave events), `"inject"` (records `@session` PTY injections).

**Write:** Append one JSON line + `\n`; a per-session `sync.Mutex` serializes concurrent senders.

**Late-join replay:** Open file, scan from line 0 (streaming `bufio.Scanner`), send all messages to the new subscriber before switching to live fan-out.

**Restart survival:** File persists on disk; daemon reopens and resumes on restart.

**Cleanup:** Delete `{session-id}.jsonl` when the session is deleted — same lifecycle as the session's PTY.

**Markdown export:** HTTP handler iterates lines, formats with `strings.Builder` (see below). No library.

**Why not `modernc.org/sqlite` v1.53.0:** CGO-free and correct (confirmed v1.53.0, published 2026-06-21), but adds a new module dependency for data that needs no SQL queries, no cross-session joins, and is bounded in size. Overkill.

**Why not `go.etcd.io/bbolt` v1.5.0:** B+tree with bucket-serialization setup adds code complexity. Single writer anyway. No benefit over a mutex-guarded `os.File` for a simple sequential list. Overkill.

### (b) WebSocket Relay Extension — text/binary frame split, no new dep

`coder/websocket` v1.8.14 already handles both `websocket.MessageText` and `websocket.MessageBinary` on the same connection. The relay's `handleSession` read loop adds one branch:

```go
msgType, data, err := conn.Read(ctx)
switch msgType {
case websocket.MessageBinary:
    // existing PTY stdin injection path — zero change
case websocket.MessageText:
    // new: unmarshal JSON envelope, route to chat or presence handler
}
```

**JSON envelope** (all chat-family messages use this schema):

```json
{ "type": "<chat|presence|typing>", "payload": { ... } }
```

Payload shapes:

| type | payload fields |
|------|---------------|
| `chat` | `id` (uuid), `ts` (RFC3339), `sender_id`, `sender_alias`, `content` (markdown string) |
| `presence` | `participants: [{id, alias, online}]` — full list broadcast on every join/leave |
| `typing` | `sender_id`, `alias`, `typing: bool` — debounced client-side every 2 s, auto-cleared after 5 s silence |

**Fan-out:** The relay already has a subscriber list per session. Chat text frames broadcast to all subscribers. Presence frames broadcast to all subscribers. The existing per-session mutex covers the subscriber list operations.

**What this adds to the relay:** The `MessageText` branch above; a `ChatStore.Append()` call before broadcast; a `presenceMap` maintained alongside the subscriber list.

### (c) Frontend Chat UI — two new packages, one hand-rolled component

**Message list — `@tanstack/react-virtual` v3.14.3**

```tsx
const rowVirtualizer = useVirtualizer({
  count: messages.length,
  getScrollElement: () => scrollRef.current,
  estimateSize: () => 64,
  overscan: 10,
  useFlushSync: false,   // required for React 19 — eliminates flushSync lifecycle warning
})
```

TanStack Virtual's chat example covers pinned-to-bottom (auto-scroll when new message arrives, unless user has scrolled up) and initial scrollback load. Headless — no CSS imposed on the existing design language.

**Auto-growing composer — `react-textarea-autosize` v8.5.9**

```tsx
import TextareaAutosize from 'react-textarea-autosize'

<TextareaAutosize
  minRows={1}
  maxRows={6}
  value={draft}
  onChange={e => setDraft(e.target.value)}
  onKeyDown={handleComposerKey}  // Enter → send; Shift+Enter → newline
  placeholder="Message…"
/>
```

1.3 KB gzipped, ships TS types.

**@mention autocomplete — hand-rolled, zero new dependency**

The participant list in any single session is bounded (2–5 people in practice). Pattern:

1. In `onChange`, scan backwards from the cursor position for an `@` trigger not preceded by a non-whitespace character.
2. If trigger found, filter the current `participants[]` array by the partial alias typed after `@`.
3. Render a small `<ul>` popover positioned below the cursor. Keyboard nav: `ArrowUp/Down`, `Enter`/`Tab` to complete, `Escape` to dismiss.
4. Insert the completed `@alias ` text, replacing the trigger + partial.

This is approximately 80 lines of TypeScript. No library saves enough code to justify its installation at this participant count. `react-mentions` v4.4.10 is 3 years unmaintained (React 19 untested). Tiptap/ProseMirror is a full rich-text editor framework — far too heavy for a plain textarea with mention completion.

**Markdown rendering in message bubbles — `react-markdown` v10.1.0 (already installed)**

Render each message's `content` field through the existing `react-markdown` + `rehype-sanitize` + `remark-gfm` pipeline already used by the file browser. No config change needed.

### (d) Markdown Export — server-side Go, no new dep

`GET /api/sessions/{id}/chat/export.md` (requires at minimum RO capability token):

```go
func (h *ChatHandler) exportMarkdown(w http.ResponseWriter, r *http.Request, sessionID, sessionName string) {
    msgs, _ := h.store.All(sessionID)
    var b strings.Builder
    fmt.Fprintf(&b, "# Session Chat: %s\nExported: %s\n\n", sessionName, time.Now().UTC().Format(time.RFC3339))
    for _, m := range msgs {
        switch m.Type {
        case "message", "inject":
            fmt.Fprintf(&b, "---\n\n**%s** (%s) — %s\n\n%s\n\n",
                m.SenderAlias, m.SenderID,
                m.Ts.Format("2006-01-02 15:04:05 UTC"),
                m.Content)
        case "system":
            fmt.Fprintf(&b, "*%s*\n\n", m.Content)
        }
    }
    w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
    w.Header().Set("Content-Disposition", `attachment; filename="chat-`+sessionID+`.md"`)
    io.WriteString(w, b.String())
}
```

Output is valid Markdown renderable in any editor or on GitHub. No Go library needed — this is string concatenation, not Markdown parsing.

### (e) Tailscale Identity — `LocalClient.WhoIs`, already in go.mod

```go
import "tailscale.com/client/tailscale"

var tsClient tailscale.LocalClient   // zero-value is usable

func identifyPeer(ctx context.Context, remoteAddr string) (loginName, displayName string, err error) {
    who, err := tsClient.WhoIs(ctx, remoteAddr)
    if err != nil {
        return "", "", err
    }
    // UserProfile and Node are guaranteed non-nil on success
    return who.UserProfile.LoginName, who.UserProfile.DisplayName, nil
}
```

- `UserProfile.LoginName` — stable tailnet identity (e.g., `ken@example.com`); use this as `sender_id` in stored messages.
- `UserProfile.DisplayName` — human-readable name; offer as default for the self-chosen alias.
- Call with `r.RemoteAddr` on WebSocket upgrade, before subscribing the client to the relay.
- Store `loginName` on the subscriber struct alongside the existing `client` name field.

**Local/loopback disambiguation:** When `r.RemoteAddr` is `127.0.0.1:*` or `[::1]:*` (desktop GUI owner connecting to their own daemon), `WhoIs` will fail or return no match — the local loopback is not a tailnet peer address. Treatment: assign a synthetic identity `"owner@local"` as `sender_id` with display name from `os.Hostname()`. The discovery doc (`.planning/notes/session-chat-discovery.md`) flags this as a gray area to finalize at plan time; this rule is the simplest defensible default.

---

## Installation

```bash
# Frontend — from the frontend/ directory
pnpm add @tanstack/react-virtual@^3.14.3 react-textarea-autosize@^8.5.9

# Backend — no go get needed; all modules already in go.mod
```

---

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| JSONL per-session file (stdlib) | `modernc.org/sqlite` v1.53.0 | New Go dep; SQL is overkill for a bounded sequential list with no cross-session queries; v1.53.0 published 2026-06-21, CGO-free and correct, but unnecessary here |
| JSONL per-session file (stdlib) | `go.etcd.io/bbolt` v1.5.0 | New Go dep; B+tree bucket setup complexity with no benefit over a mutex + append-only file for simple sequential chat |
| Text/binary WS frame split on existing relay | New `/chat` WebSocket endpoint | Two connections per session on the frontend; duplicates auth and relay infrastructure |
| Text/binary WS frame split on existing relay | Socket.IO / Centrifugo / Pusher | New server-side library or process; existing relay already does per-session fan-out |
| Hand-rolled @mention | `react-mentions` v4.4.10 | Last published 3 years ago; React 19 compatibility unknown; the hand-rolled approach is ~80 lines and avoids adding a stale dependency |
| `@tanstack/react-virtual` v3.14.3 | `react-window` | react-window is in maintenance mode; TanStack Virtual is the community successor with active development and a dedicated reverse-scroll chat example |
| Server-side Go Markdown export | Client-side JS export | Server already holds all messages; client export requires a round-trip fetch then formatting — adds complexity with no benefit |
| `react-markdown` v10.1.0 (already installed) | Any other Markdown renderer | Already present; rehype-sanitize already wired; no reason to add a second renderer |

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `modernc.org/sqlite` or any SQLite driver | New Go module for data that needs no SQL; JSONL covers all requirements | JSONL per-session file (stdlib) |
| `go.etcd.io/bbolt` | New Go module; B+tree overhead for a sequential append list | JSONL per-session file (stdlib) |
| Second WebSocket endpoint or second WS connection per session | Doubles frontend WS management; duplicates auth/relay infrastructure | Text frame on existing relay connection |
| Socket.IO / Pusher / Ably / Centrifugo | External service or heavyweight server library; violates zero-config ethos; existing relay already fan-outs | Extend `coder/websocket` relay with text frames |
| `react-mentions` v4.4.10 | 3 years unmaintained; React 19 untested; more code to integrate than hand-rolled | ~80 lines of hand-rolled TypeScript |
| Tiptap / ProseMirror | Full rich-text editor framework (100+ KB); way heavier than a plain textarea + mention completion | `react-textarea-autosize` + hand-rolled @mention |
| `stream-chat-react` or any hosted chat SDK | SaaS dependency; per-message pricing; requires external user accounts | Extend own relay |
| `github.com/gomarkdown/markdown` or any Go Markdown library | No parsing needed — the export produces a Markdown *string*, not rendered HTML | `strings.Builder` + `fmt.Fprintf` |
| Any new Go module for UUID generation | `crypto/rand` + `fmt.Sprintf("%x", ...)` is sufficient for unique message IDs | stdlib `crypto/rand` |

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `@tanstack/react-virtual@3.14.3` | React 19.2.x | Set `useFlushSync: false` in `useVirtualizer` options; confirmed in TanStack docs for React 19 |
| `react-textarea-autosize@8.5.9` | React 19.2.x | Simple controlled-component wrapper over native `<textarea>`; no React internals coupling; no known React 19 breakage |
| `react-markdown@10.1.0` | Already in frontend/package.json | No change; existing pipeline with rehype-sanitize already handles user content safely |
| `tailscale.com@v1.98.3` | Go 1.26.3 | Already in go.mod; `client/tailscale.LocalClient.WhoIs` returns `*apitype.WhoIsResponse` with `Node` + `UserProfile` (both non-nil on success) |
| `coder/websocket@v1.8.14` | Go 1.26.3 | Already in go.mod; `websocket.MessageText` and `websocket.MessageBinary` constants available |

---

## Sources

- `pkg.go.dev/modernc.org/sqlite` — v1.53.0 confirmed CGO-free, WAL support (FileControlPersistWAL), single-goroutine connections, published 2026-06-21 (MEDIUM confidence)
- `pkg.go.dev/go.etcd.io/bbolt` — v1.5.0, NextSequence(), byte-sorted cursor, single-writer model confirmed, published 2026-06-03 (MEDIUM confidence)
- `npmjs.com/@tanstack/react-virtual` — v3.14.3, React 19 `useFlushSync: false` note confirmed in TanStack docs (MEDIUM confidence)
- `npmjs.com/react-textarea-autosize` — v8.5.9, 1.3 KB gzipped, ships TS types (MEDIUM confidence)
- `npmjs.com/react-mentions` — v4.4.10, last published 3 years ago (MEDIUM confidence — confirms staleness)
- `pkg.go.dev/tailscale.com/client/tailscale/apitype#WhoIsResponse` — `Node` + `UserProfile` fields confirmed non-nil on success; `LoginName` + `DisplayName` fields confirmed (MEDIUM confidence)
- `tanstack.com/virtual/v3/docs/framework/react/react-virtual` — chat reverse-scroll example documented (MEDIUM confidence)
- `/Users/ken/dev/agenthub/go.mod` — confirmed `coder/websocket v1.8.14`, `tailscale.com v1.98.3`, `go 1.26.3`
- `/Users/ken/dev/agenthub/frontend/package.json` — confirmed `react-markdown@10.1.0`, `rehype-sanitize@^6.0.0`, `remark-gfm@^4.0.1`, `@heroicons/react@^2.2.0`, React 19.2.x

---
*Stack research for: v4.1 Session Chat — new capability additions only*
*Researched: 2026-06-25*
