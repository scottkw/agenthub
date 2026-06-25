# Phase 152: Relay Protocol + Identity + Presence - Context

**Gathered:** 2026-06-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Add an **identity + live-presence + typing** layer on top of the existing web-share
WebSocket relay so that every participant in a session's chat is identified and their
join/leave and typing status propagate in real time across all connected clients
(desktop GUI owner + tailnet web-share peers).

**In scope (IDENT-01, IDENT-02, PRESENCE-01, PRESENCE-02):**
- Stamp each incoming WS connection with a `TailnetID` (`lc.WhoIs` for web clients,
  `"local"` for the Wails desktop owner) **+** an alias, before any message is stored.
- Set/change alias with propagation to all clients within one relay round-trip.
- Presence (connected / disconnected) shown to all participants in real time.
- Typing indicators: appear ≤500 ms, auto-clear after 5 s idle or on disconnect,
  **never persisted** to the JSONL log; server-side TTL clears them on abrupt drop.
- Owner and a same-machine browser client resolve to **two distinct** presence entries.

**Out of scope (later phases / explicitly designed out):**
- `@session` PTY injection + RW-cap gate + sanitization → **Phase 153**.
- Chat UI panels (desktop / web), Markdown rendering, unread notifications → **Phases 154–155**.
- Agent-as-chat-author / round-trip replies, tool-output cards, archive outliving the session.

</domain>

<decisions>
## Implementation Decisions

### Alias — default & persistence
- **D-01: Default alias is derived from the tailnet identity.** Before a participant
  picks a name, show a tailnet-derived default (MagicDNS hostname / login name, e.g.
  `ken`, `macbook`). The desktop owner shows as a `"local"`-origin entry (e.g.
  `You (local)`). No forced "pick a name" gate — preserves the passwordless / zero-config
  ethos. No generic "Guest" placeholder.
- **D-02: Alias is daemon-persisted, keyed by the composite identity (see D-04).** A
  chosen alias survives reconnect, late-join, and daemon restart — set once, sticks.
  Storage is daemon-owned (not client localStorage, not connection-scoped memory). The
  persistence key is the **composite (TailnetID + origin)** key from D-04, **not** bare
  TailnetID — so the owner and a same-machine browser keep separate persisted aliases.
  This complements Phase 151's per-message `AuthorAlias` *snapshot* (the live alias is a
  separate, mutable presence attribute).

### Presence — granularity & disambiguation
- **D-03: Presence is per-person, collapsed.** A participant's multiple live connections
  (two browser tabs, or desktop + browser) collapse into **one** presence entry
  (e.g. `Ken — 2 connections`). Requires reference-counting connections per person key;
  the entry goes `disconnected` only when the last connection for that key drops.
- **D-04: The "person" key is the composite `TailnetID + origin`.** `origin` ∈
  {`local` (desktop Wails owner), `web` (web-share browser)}. Consequences:
  - Same remote peer's multiple browser tabs → same composite key → collapse into one entry.
  - Desktop owner (`local` origin) vs a same-machine browser (`web` origin, same tailnet
    node) → **different origin → two distinct keys → two distinct entries.** This is the
    mechanism that satisfies success-criterion 5 (no silent identity merge).
  - **Rejected:** per-connection presence (would over-split same-peer tabs); origin +
    per-connection nonce (a fresh nonce per reconnect would break the daemon-persisted
    alias continuity from D-02).

### Typing indicator — display
- **D-05: Named typing indicator with overflow rollup.** Show aliases:
  `Ken is typing…` → `Ken and Sam are typing…` → past a threshold roll up to
  `Ken, Sam +2 typing…`. The typing frame therefore **must carry the typer's identity/
  alias**, not just a count. (Rejected: anonymous count — strictly worse in a 2–3 person
  side-channel; single-most-recent — hides concurrent typers.) Timings are locked by the
  success criteria (≤500 ms appear, 5 s idle clear, clear-on-disconnect, never stored).

### Participant scope — read-only viewers
- **D-06: Read-only (RO-cap) web viewers are full chat participants.** They appear in
  presence, can post messages, can type, and are `@mention`-able. The RO cap only ever
  gates the **terminal** (PTY input) — and, in Phase 153, the separately RW-gated
  `@session` injection. Chat is human-to-human and not cap-restricted. Matches the
  discovery-doc definition "participants = humans connected to the session." (Rejected:
  presence-only spectator tier; RW-only chat — both contradict the side-channel intent.)

### Claude's Discretion (defer to researcher / planner, with noted defaults)
- **Wire protocol shape.** Whether to extend the existing `MetaPayload` (`MsgMeta 0x20`,
  server→client) with optional presence/typing fields vs. add dedicated frame-type
  constants. Note both directions are needed: client→server for *set-alias* and
  *typing-start/stop*; server→client for *presence roster* and *typing roster*.
  Suggested default: reserve new frame-type constants in the `0x21–0x2F` range for the
  client→server verbs (set-alias, typing) and fan presence/typing rosters out via the
  existing `MsgMeta`/`BroadcastMeta` metadata path. Planner decides.
- **Abrupt-disconnect detection feeding the typing/presence TTL.** Clean WS close vs.
  ping/pong-timeout detection for the 5 s typing TTL and presence `disconnected`.
  Reuse the existing keep-alive (`MsgPing 0x12`) where possible.
- **Alias validation** (length cap, allowed charset, trimming, uniqueness). Default:
  bounded length, printable-only, **non-unique allowed** (identity is the TailnetID, alias
  is just a label) — planner picks concrete bounds consistent with the chat line-size limits.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone / phase intent
- `.planning/notes/session-chat-discovery.md` — the v4.1 Session Chat design record;
  defines identity = tailnet ID + self-chosen alias (both visible), the passwordless
  ethos, and explicitly flags the owner-vs-same-machine-browser disambiguation and
  presence/typing fidelity as resolve-at-plan-time gray areas (now resolved above).
- `.planning/ROADMAP.md` §"Phase 152" — the 5 success criteria (TailnetID stamping,
  alias propagation, real-time presence, 500 ms/5 s typing TTL never-stored, distinct
  owner vs same-machine-browser entries).
- `.planning/REQUIREMENTS.md` — IDENT-01, IDENT-02, PRESENCE-01, PRESENCE-02.

### Phase 151 carry-forward (locked upstream — read before extending)
- `internal/relay/protocol.go` — `ChatMessage` (`AuthorID` = `"local"`/tailnet pubkey,
  `AuthorAlias` snapshot, `SchemaVersion`, `SessionInject`), frame-type constants
  (`MsgMeta 0x20`), and `MetaPayload` (`ViewerCount`) — the identity record + extensibility point.
- `internal/daemon/chat.go` — `ChatStore` (per-session JSONL store, `AppendMessage`
  fills `AuthorID`/`AuthorAlias`, `Export`, `Delete`). Phase 152 stamps identity/alias
  that `AppendMessage` records.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/relay/protocol.go:62` — `MetaPayload{ ViewerCount *int }` broadcast over
  `MsgMeta 0x20`: the natural carrier to extend with presence/typing rosters.
- `internal/relay/server.go:365` — `NotifyViewerCount()` already fans presence-like
  state (join/leave count) via `BroadcastMeta`; presence broadcasting can mirror this path.
- `internal/relay/hub.go` — `Hub.Subscribe/Unsubscribe` (74–85), `SubscriberCount()`
  (88–92), `broadcast()`/`BroadcastMeta()` (156–183); `Subscriber` already has
  `Name` (`?client=` hint), `ReadOnly`, `Cols/Rows`.
- `internal/capability/capability.go` — `Claims{ SID, Perms, GrantID }` + `HasPerm`;
  `Perms == "read"` is the RO signal (D-06: gates terminal, NOT chat).

### Established Patterns
- Two WS entry paths: **relay (loopback, owner)** `internal/relay/server.go:177`
  `handleSession()` (parses `?readonly=`, `?client=`) → origin = `"local"`; and
  **web-share (Tailscale-bound, cap-gated)** `internal/webserver/server.go:972`
  `handleWSSRelay()` (`readonly := claims.Perms == "read"`) → origin = `"web"`, needs
  `lc.WhoIs` to resolve TailnetID.
- Tailnet `LocalClient` already imported: `internal/tailnet/tailnet.go:23`,
  `internal/webserver/tailscale.go:7` (`local.Client`, `lc.Status`/`StatusWithoutPeers`)
  — `lc.WhoIs(remoteAddr)` is the resolution call to add on the web path.
- Per-session ChatStore wiring: `internal/daemon/engine.go` `chatStores` map (43),
  `ChatStoreFor()` (287), create in `CreateSession` (~418), delete on `KillSession` (~579).

### Integration Points
- **Identity stamping:** `handleWSSRelay` (web) resolves TailnetID via `lc.WhoIs`;
  relay `handleSession` (owner) stamps `"local"`. Both attach composite key (D-04) +
  current alias (D-02) to the `Subscriber` before any `AppendMessage`.
- **Subscriber struct** (`internal/relay/hub.go:9`) gains a TailnetID/origin/alias triple
  (alongside existing `Name`).
- **Presence/typing fan-out:** extend `MetaPayload` and/or new frame types; broadcast on
  Subscribe/Unsubscribe (presence) and on typing-start/stop frames (typing, with TTL).
- **Alias store:** new daemon-side persisted map keyed by composite (TailnetID+origin),
  or extend session state; consumed by both WS paths.

</code_context>

<specifics>
## Specific Ideas

- Owner label reads as a `"local"`-origin entry (e.g. `You (local)`), distinct from any
  same-machine browser entry — the visible proof of the no-silent-merge rule.
- Typing rollup copy pattern: `Ken is typing…` / `Ken and Sam are typing…` /
  `Ken, Sam +2 typing…`.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. (Adjacent items — `@session` injection,
chat UI, notifications, Markdown export — are already separately scoped to Phases
153–155 in the roadmap, not deferrals from this discussion.)

</deferred>

---

*Phase: 152-relay-protocol-identity-presence*
*Context gathered: 2026-06-25*
