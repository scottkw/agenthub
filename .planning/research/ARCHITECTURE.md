# Architecture Research

**Domain:** Per-session human-to-human chat integrated into Go/Wails + React app (AgentHub v4.1)
**Researched:** 2026-06-25
**Confidence:** HIGH — based on direct codebase inspection (relay, hub, webserver, daemon, capability packages)

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CLIENT SURFACES                                      │
│                                                                               │
│  ┌──────────────────────────────┐   ┌──────────────────────────────────────┐ │
│  │  Desktop GUI (Wails/React)   │   │  Web-share browser (React /app/ SPA) │ │
│  │  HubInteractiveModal         │   │  terminal.html + chat panel          │ │
│  │  + ChatPanel.tsx (NEW)       │   │  + chat panel (NEW)                  │ │
│  │  WS: ws://127.0.0.1:<relay>  │   │  WS: wss://<tailnet>:7443/sessions   │ │
│  │  No cap token (loopback)     │   │  ?cap=<token> (capability-gated)     │ │
│  └──────────────┬───────────────┘   └─────────────────┬────────────────────┘ │
└─────────────────┼───────────────────────────────────────┼──────────────────┘
                  │                                         │
    ┌─────────────▼─────────────┐         ┌────────────────▼─────────────────┐
    │  relay.Server             │         │  webserver.WebServer             │
    │  internal/relay/server.go │         │  internal/webserver/server.go    │
    │  (MODIFIED)               │         │  (MODIFIED)                      │
    │  - /sessions/{id}/ws      │         │  - /sessions/{id}/ws             │
    │  - /sessions/{id}/chat    │         │  - /api/sessions/{id}/chat       │
    │    (REST history, NEW)    │         │    (REST history, cap-gated, NEW) │
    │  - MsgChatSend dispatch   │         │  - MsgChatSend dispatch          │
    │    in read pump           │         │    in read pump                  │
    │  - WhoIs: "local" (skip)  │         │  - WhoIs: lc.WhoIs(remoteAddr)   │
    └─────────────┬─────────────┘         └────────────────┬─────────────────┘
                  │                                         │
    ┌─────────────▼─────────────────────────────────────────▼─────────────────┐
    │                           relay.Hub                                       │
    │                     internal/relay/hub.go (MODIFIED)                     │
    │                                                                           │
    │  Existing: Run() PTY drain → scrollback → broadcast(MsgOutput)           │
    │  Existing: BroadcastMeta(MsgMeta viewer count)                           │
    │  NEW:      BroadcastChat(MsgChat frame) — fan-out to all subscribers     │
    │  NEW:      chatHandler callback (registered by daemon, called on recv)   │
    └─────────────────────────────────────┬────────────────────────────────────┘
                                          │
    ┌─────────────────────────────────────▼────────────────────────────────────┐
    │                        daemon.SessionEngine                               │
    │                      internal/daemon/engine.go (MODIFIED)                │
    │                                                                           │
    │  Existing: session registry, pty backend, hub manager, settings          │
    │  NEW:      chatStores map[string]*ChatStore  (sessionID → store)         │
    │  NEW:      KillSession extended → delete ChatStore + chat file           │
    └─────────────────────────────────────┬────────────────────────────────────┘
                                          │
    ┌─────────────────────────────────────▼────────────────────────────────────┐
    │                           daemon.ChatStore  (NEW)                         │
    │                      internal/daemon/chat.go                              │
    │                                                                           │
    │  In-memory slice + flat JSON file                                         │
    │  ~/.config/agenthub/chats/<sessionID>.json                               │
    │  Append(msg) → persists atomically (temp+rename)                         │
    │  Messages() → returns full thread for late-join scrollback                │
    │  Export() → renders Markdown for download                                 │
    │  Delete() → removes the JSON file                                         │
    └──────────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | New or Modified |
|-----------|---------------|-----------------|
| `internal/daemon/chat.go` | Per-session message store: persist, load, export | NEW |
| `internal/relay/protocol.go` | Binary frame type constants + payload structs for chat/presence/typing | MODIFIED |
| `internal/relay/hub.go` | BroadcastChat fan-out; chatHandler callback registration | MODIFIED |
| `internal/relay/server.go` | MsgChatSend dispatch in read pump; REST history route | MODIFIED |
| `internal/webserver/server.go` | Cap-gated REST history; WhoIs identity at WS upgrade; MsgChatSend dispatch | MODIFIED |
| `internal/daemon/engine.go` | Instantiate + register ChatStore per session; wire KillSession deletion | MODIFIED |
| `internal/daemon/api.go` | Daemon-socket REST route for chat history (desktop GUI load path) | MODIFIED |
| `frontend/src/components/Hub/ChatPanel.tsx` | Chat UI panel for desktop modal (Wails surface) | NEW |
| `frontend/src/components/Hub/HubInteractiveModal.tsx` | Embed ChatPanel alongside terminal | MODIFIED |
| Web chat UI (in /app/ SPA or terminal.html) | Chat panel for web-share surface | NEW |

## Recommended Project Structure

New files introduced by v4.1:

```
internal/daemon/
├── chat.go               # ChatStore: message store, persist, export (NEW)
├── chat_test.go          # unit tests for ChatStore (NEW)
internal/relay/
├── protocol.go           # + MsgChat, MsgChatSend, MsgPresence, MsgTyping consts
│                         # + ChatMessage, PresencePayload structs (MODIFIED)
frontend/src/components/Hub/
├── ChatPanel.tsx         # desktop chat panel component (NEW)
├── ChatPanel.test.tsx    # tests (NEW)
web/
├── assets/chat.css       # chat panel styles for web surface (NEW)
```

Chat file storage:

```
~/.config/agenthub/
├── settings.json             (existing)
├── chats/
│   └── <sessionID>.json      (NEW — one per live session, deleted with session)
```

## Architectural Patterns

### Pattern 1: Extend Relay Protocol with New Frame Types

**What:** Add new single-byte message type constants to `internal/relay/protocol.go` for chat, presence, and typing. Use the same binary framing (`[typeByte][payload...]`) as `MsgOutput`, `MsgMeta`, etc.

**When to use:** Any new server-push or client-to-server channel that must fan-out to all session subscribers simultaneously.

**Trade-offs:** Keeps a single WebSocket connection per session on each surface. Both the PTY stream and the chat stream ride the same connection. Subscribers already receive all frame types; the UI filters by type byte. The main risk is that the Hub's drain goroutine runs per-session, not per-client — chat fan-out must use the same non-blocking `BroadcastChat` pattern as `BroadcastMeta` to avoid blocking PTY drain.

**New constants (in `internal/relay/protocol.go`):**

```go
// Chat frame types — range 0x30-0x3F reserved for chat/presence.
MsgChat     byte = 0x30 // server → client: new chat message (JSON payload)
MsgChatSend byte = 0x31 // client → server: send a chat message
MsgPresence byte = 0x32 // server → client: presence list update (JSON payload)
MsgTyping   byte = 0x33 // bidirectional: typing indicator
MsgAliasSet byte = 0x34 // client → server: set/update display alias

// ChatMessage is the canonical wire type for a chat message.
type ChatMessage struct {
    ID            string   `json:"id"`
    SessionID     string   `json:"sessionID"`
    AuthorID      string   `json:"authorID"`    // stable tailnet node key or "local"
    AuthorAlias   string   `json:"alias"`
    Content       string   `json:"content"`
    Mentions      []string `json:"mentions,omitempty"`
    SessionInject bool     `json:"sessionInject,omitempty"` // true when @session triggered
    TimestampMs   int64    `json:"ts"`          // UNIX milliseconds
}
```

### Pattern 2: Hub chatHandler Callback (Daemon Owns Persistence)

**What:** The relay Hub must not import the daemon package (import cycle). The daemon registers a callback on the Hub at session creation. When the relay read pump receives `MsgChatSend`, it calls this callback. The callback persists the message and returns a canonical `ChatMessage`. The relay then calls `hub.BroadcastChat`.

**When to use:** Whenever a relay-layer event needs side effects owned by a higher-level package without import cycles. This is the same inversion used for `resizeFn` on the Hub today.

**Hub additions:**

```go
// SetChatHandler registers the daemon-side callback.
func (h *Hub) SetChatHandler(fn func(senderID, alias, content string) (*ChatMessage, error))

// BroadcastChat fans out a MsgChat frame to all subscribers (non-blocking).
func (h *Hub) BroadcastChat(frame []byte)

// BroadcastPresence fans out a MsgPresence frame (called on connect/disconnect/alias change).
func (h *Hub) BroadcastPresence(frame []byte)
```

**Read pump addition (identical in relay/server.go and webserver/server.go):**

```go
case MsgChatSend:
    if fn := hub.ChatHandler(); fn != nil {
        msg, err := fn(sub.TailnetID, sub.Alias, string(payload))
        if err == nil {
            hub.BroadcastChat(relay.MakeChatFrame(*msg))
            if msg.SessionInject {
                hub.WriteInput(append([]byte(extractSessionPrompt(msg.Content)), '\n'))
            }
        }
    }
```

### Pattern 3: Flat JSON File Chat Store (Daemon-Persisted, Session-Scoped)

**What:** One JSON file per session at `~/.config/agenthub/chats/<sessionID>.json`. Array of `ChatMessage` appended atomically (temp+rename pattern used throughout the codebase). Loaded into memory on daemon restart. File deleted in `KillSession`.

**When to use:** Data must survive daemon restarts; volume is bounded by session lifetime; no cross-session query required.

**Trade-offs:** Zero new dependencies. Simple to reason about. Not suitable for cross-session search or very high message volumes (thousands+ per session), but chat is inherently lower throughput than PTY output. SQLite would add complexity without v4.1-scope benefit.

**Lifecycle binding in engine.go:**

```go
func (e *SessionEngine) KillSession(id string) error {
    // ... existing removes ...
    e.mu.Lock()
    if store, ok := e.chatStores[id]; ok {
        store.Delete()
        delete(e.chatStores, id)
    }
    e.mu.Unlock()
    return nil
}
```

## Data Flow

### Sending a Chat Message (Web Surface)

```
Browser (web-share tab)
    ↓ user types, hits Enter
ChatPanel constructs MsgChatSend frame (0x31 + JSON payload)
    ↓ WS send → wss://<tailnet>:7443/sessions/{id}/ws?cap=<token>
webserver.handleWSSRelay read pump
    ↓ ParseFrame → MsgChatSend
    ↓ calls hub.ChatHandler()(sub.TailnetID, sub.Alias, content)
        ↓ engine.ChatStore.Append(msg) → persists to chats/<id>.json
        returns canonical ChatMessage (daemon-assigned ID + timestamp)
    ↓ hub.BroadcastChat(MakeChatFrame(msg)) → all subscribers (desktop + web)
    if @session: hub.WriteInput(prompt + "\n") → PTY stdin → one-way
```

### Late Joiner (Loading Chat History)

```
New client connects to session WS
    ↓ server subscribes client, replays PTY scrollback (existing)
    ↓ client immediately calls:
        GET /sessions/{id}/chat              (desktop, relay mux)
        GET /api/sessions/{id}/chat?cap=X    (web, capability-gated)
    ↓ daemon returns ChatMessage array from in-memory store
    ↓ ChatPanel renders full history
    ↓ live MsgChat frames arrive via WS from this point
```

### @session PTY Injection

```
User sends: "@session refactor the auth module"
    ↓ daemon chatHandler detects "@session " prefix
    ↓ msg.SessionInject = true; extractSessionPrompt strips prefix
    ↓ hub.WriteInput([]byte("refactor the auth module\n")) → PTY stdin
    ↓ agent processes; reply appears in terminal stream only (not chat)
Chat message broadcast includes sessionInject:true → UI renders
"→ injected into terminal" indicator on the chat bubble.
```

### Presence + Typing Indicators

```
Client connects:
    ↓ server calls hub.BroadcastPresence(current participant list)
    ↓ all clients update their presence panel

Client types (debounced, ~500ms after last keystroke):
    ↓ client sends MsgTyping frame
    ↓ server re-broadcasts to all OTHER subscribers
    ↓ recipients show "{alias} is typing..." (auto-clears after 3s)
```

## Identity Resolution

### Web-share clients (Tailscale network)

At WS upgrade time in `webserver.handleWSSRelay`, call `lc.WhoIs(r.RemoteAddr)` using the `local.Client` that is already instantiated for `GetCertificate`:

```go
var nodeID, hostname string
if info, err := lc.WhoIs(r.Context(), r.RemoteAddr); err == nil {
    nodeID = info.Node.Key.String() // stable node public key
    hostname = info.Node.HostName
} else {
    nodeID = "local"
    hostname = serverHostname
}
```

`TailnetID` and `Alias` are stored on `relay.Subscriber` (two new fields). The alias defaults to `hostname` until the client sends `MsgAliasSet`.

### Desktop GUI (Wails surface, loopback)

All connections arrive from `127.0.0.1`; `WhoIs` returns an error or the local machine's own node. Use `TailnetID = "local"` consistently. The owner's initial alias defaults to `engine.hostname` (already captured at daemon init via `os.Hostname()`).

### Edge case: local owner opens the web URL on the same machine

When connecting via the Tailscale FQDN from the same machine, the source IP is the machine's own Tailscale IP. `lc.WhoIs` returns the local node's info. `TailnetID` becomes the local node key — the same node key the desktop GUI uses (also resolved from `lc.WhoIs` on the loopback path, or tagged as "local"). The UI should merge these under one participant entry keyed by `TailnetID`. The two WS connections remain distinct (each gets a unique `connectionID` for session tracking), but both appear as the same person in the presence list.

### Alias lifecycle

- Alias is not persisted server-side. Clients store it in localStorage and re-send `MsgAliasSet` on reconnect.
- Server caps alias at 64 chars (same pattern as `clientName` today).
- Every alias change triggers `hub.BroadcastPresence` so all clients see the updated name.

## Cross-Surface Shared Contract

Both surfaces (Wails desktop GUI, web-share browser) use the same `ChatMessage` JSON schema and the same relay frame types. The shared `ChatPanel.tsx` React component is compiled into the main frontend bundle (used via `staticAppFS` embed on the web surface) — no duplicate UI code.

| Channel | Desktop path (relay mux) | Web-share path (webserver mux) |
|---------|--------------------------|-------------------------------|
| History load | `GET /sessions/{id}/chat` | `GET /api/sessions/{id}/chat?cap=X` |
| Live messages | `MsgChat` on `ws://127.0.0.1:<relay>/sessions/{id}/ws` | Same frame type on `wss://` |
| Send | `MsgChatSend` frame | Same |
| Presence | `MsgPresence` frame | Same |
| Typing | `MsgTyping` frame | Same |
| Markdown export | `GET /sessions/{id}/chat/export` | `GET /api/sessions/{id}/chat/export?cap=X` |

## Dependency-Ordered Build Sequence

1. **Message schema + ChatStore** — define `relay.ChatMessage` wire type; implement `daemon.ChatStore`; wire into `SessionEngine` (instantiation + `KillSession` deletion). Nothing else can build without the schema.

2. **REST history + export endpoints** — `GET /sessions/{id}/chat` on relay mux (desktop); `GET /api/sessions/{id}/chat` on webserver (capability-gated); Markdown export endpoint. UI cannot render history without this.

3. **Protocol frame types + Hub extension** — add `MsgChat`, `MsgChatSend`, `MsgPresence`, `MsgTyping`, `MsgAliasSet` constants; add `BroadcastChat`, `BroadcastPresence`, `SetChatHandler` to Hub; add `TailnetID`/`Alias` fields to Subscriber. Schema must exist before this step.

4. **Identity resolution** — add `lc.WhoIs` call at WS upgrade in webserver; populate `TailnetID`/`Alias` on both relay.Server and webserver subscriber creation; handle `MsgAliasSet`. Must precede message attribution in step 5.

5. **Relay read pump: MsgChatSend dispatch** — handle `MsgChatSend` in both `relay/server.go` and `webserver/server.go` read pumps; call chatHandler → persist → BroadcastChat; handle `MsgTyping` / `MsgAliasSet`. Steps 3 and 4 must be complete.

6. **@session PTY bridge** — detect `@session` prefix in daemon chatHandler; call `hub.WriteInput`. Requires step 5.

7. **Desktop chat UI** — `ChatPanel.tsx` (message list, input, presence/typing indicators); wire into `HubInteractiveModal.tsx`. Testable end-to-end once steps 2 and 5 are complete.

8. **Web-share chat UI** — same `ChatPanel.tsx` rendered via the web SPA; verify cap-gated REST + WS path. Requires step 7 (component complete) + step 4 (WhoIs in webserver).

9. **Cross-surface parity gate + Markdown export** — verify identical behavior on both surfaces; wire export button to the export endpoint.

## Anti-Patterns

### Anti-Pattern 1: Chat via Parallel WebSocket

**What people do:** Add a separate `/sessions/{id}/chat/ws` endpoint for chat-only traffic.
**Why it's wrong:** Doubles connections per viewer; requires separate auth plumbing on the web surface; presence is harder to correlate with PTY viewer count; the UI must manage two connection lifecycles. The `BroadcastMeta` pattern proves the existing Hub can already multiplex non-PTY server-push frames safely.
**Do this instead:** Extend the existing relay WS with new frame type bytes. `BroadcastChat` is structurally identical to `BroadcastMeta`.

### Anti-Pattern 2: Storing Identity Server-Side

**What people do:** Persist aliases and tailnet IDs in a server-side user table.
**Why it's wrong:** Breaks the zero-config/zero-login ethos. Creates a user management problem that the feature doesn't need. Tailnet membership is already the trust boundary.
**Do this instead:** `TailnetID` comes from Tailscale's `WhoIs` (cryptographically verified). Alias lives in client localStorage and is re-sent via `MsgAliasSet` on reconnect. Server never needs to store aliases beyond the current WS connection.

### Anti-Pattern 3: PTY Echo as Agent Reply in Chat

**What people do:** Parse PTY output after `@session` injection and attribute the next lines to the agent in the chat thread.
**Why it's wrong:** The agent paints a TUI — spinners, redraws, ANSI sequences — not discrete messages. Segmenting PTY output into clean conversational turns is the problem explicitly designed out in the discovery record (the same class as v4.0 mini-preview garble, an order of magnitude harder).
**Do this instead:** One-way bridge only. `SessionInject=true` on the chat message; the UI renders "→ injected into terminal." The agent's work appears in the terminal stream.

### Anti-Pattern 4: Blocking the Hub Drain Goroutine on Chat Persistence

**What people do:** Call `ChatStore.Append` (file I/O) inside the Hub's `broadcast` path or from the PTY drain goroutine.
**Why it's wrong:** The drain goroutine reads PTY output at full speed. Any blocking call stalls output fan-out for all subscribers.
**Do this instead:** `chatHandler` is called from the read pump goroutine of the subscriber that sent the message — not from the drain goroutine. File I/O blocks only that client's read pump. `BroadcastChat` then uses the same non-blocking send + `CloseSlow` pattern as all other broadcasts.

## Sources

- Direct inspection: `internal/relay/protocol.go`, `hub.go`, `server.go`
- Direct inspection: `internal/webserver/server.go`, `capability_mw.go`, `auth.go`, `tailscale.go`
- Direct inspection: `internal/daemon/engine.go`, `types.go`
- Direct inspection: `internal/capability/capability.go`
- Direct inspection: `internal/tailnet/tailnet.go`
- Design record: `.planning/notes/session-chat-discovery.md`
- Project context: `.planning/PROJECT.md` (v4.1 milestone target features)

---
*Architecture research for: AgentHub v4.1 Session Chat — integration with existing daemon/relay/PTY/Tailscale stack*
*Researched: 2026-06-25*
