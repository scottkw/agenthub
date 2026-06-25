# Phase 152: Relay Protocol + Identity + Presence - Research

**Researched:** 2026-06-25
**Domain:** Go WebSocket relay protocol extension — identity stamping, presence roster, typing indicators
**Confidence:** HIGH (all findings from direct codebase inspection of the shipped Phase 151 code)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01** Default alias derived from tailnet identity (MagicDNS hostname / login name). Owner shows as a `"local"`-origin entry. No forced "pick a name" gate.

**D-02** Alias is daemon-persisted, keyed by the composite identity (TailnetID + origin). Survives reconnect, late-join, and daemon restart. Storage is daemon-owned. The persistence key is the composite (TailnetID + origin) key from D-04, not bare TailnetID. Complements Phase 151 per-message AuthorAlias snapshot (the live alias is a separate, mutable presence attribute).

**D-03** Presence is per-person, collapsed. Multiple connections from the same composite key collapse into one presence entry (e.g. `Ken — 2 connections`). Reference-count connections per person key; entry goes `disconnected` only when the last connection drops.

**D-04** Person key is the composite `TailnetID + origin`. `origin` is `"local"` (desktop Wails owner, relay/loopback path) or `"web"` (web-share browser, webserver/Tailscale path). Same remote peer's multiple browser tabs share one composite key and collapse. Desktop owner (`local` origin) vs same-machine browser (`web` origin, same tailnet node) have different origins and thus different composite keys — two distinct presence entries.

**D-05** Named typing indicator with overflow rollup: `Ken is typing…` → `Ken and Sam are typing…` → `Ken, Sam +2 typing…`. Typing frame carries the typer's identity/alias. Timings locked by success criteria: ≤500 ms appear, 5 s idle clear, clear-on-disconnect, never stored.

**D-06** RO-cap web viewers are full chat participants. Presence, posting, typing, and @mention all unrestricted. The RO cap gates only the terminal (PTY input).

### Claude's Discretion

**Wire protocol shape** — Extend MetaPayload / MsgMeta (0x20) vs. new dedicated frame-type constants. Both directions required: client→server for set-alias and typing-start/stop; server→client for presence roster and typing roster. Suggested default: 0x21–0x2F for client→server verbs, fan rosters via MsgMeta/BroadcastMeta. Planner decides.

**Abrupt-disconnect detection** feeding the typing/presence TTL — clean WS close vs. ping/pong-timeout; reuse existing MsgPing 0x12 where possible.

**Alias validation** — length cap, allowed charset, trimming, uniqueness. Default: bounded length, printable-only, non-unique allowed.

### Deferred Ideas (OUT OF SCOPE)

None. Adjacent items (`@session` injection, chat UI, notifications, Markdown export) are scoped to Phases 153–155.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| IDENT-01 | Each participant is identified by their tailnet ID and a self-chosen alias, both visible to all participants | WhoIs resolution at WS upgrade; TailnetID on Subscriber; alias broadcast via MsgPresence |
| IDENT-02 | A user can set and change their alias; the local owner and a same-machine web client resolve to a single, correctly-disambiguated participant | MsgAliasSet frame; composite key (TailnetID + origin) ensures two distinct presence entries |
| PRESENCE-01 | Each participant's presence (connected / disconnected) is shown to all participants | Hub presence roster with refcount; BroadcastPresence on Subscribe/Unsubscribe |
| PRESENCE-02 | Typing indicators show when another participant is composing (debounced, volatile, never stored, with a server-side TTL so they clear on abrupt disconnect) | MsgTyping bidirectional; server-side time.AfterFunc 5s TTL per personKey; cancelled on Unsubscribe |

</phase_requirements>

## Summary

Phase 152 layers identity, presence, and typing onto the existing relay WebSocket connection without introducing a parallel connection. All new behavior rides the existing Hub subscriber model via three new frame type bytes — no MetaPayload extension. The key design tensions were: (1) whether to extend the existing MsgMeta JSON envelope vs. using dedicated frame constants; (2) where the 5-second typing TTL lives and what event fires it; and (3) what concrete bounds govern alias text. All three are resolved by the findings below.

The Phase 151 codebase (shipped and confirmed) provides the exact integration points. The `Subscriber` struct in `hub.go` and the two WS handler goroutines in `relay/server.go` and `webserver/server.go` are the mutation surfaces. The Hub already has the locking pattern, the BroadcastMeta fan-out pattern, and the NotifyViewerCount reference-count lifecycle — all three are directly reused for presence and typing.

The Tailscale `local.Client.WhoIs` call is a single local socket roundtrip at WS upgrade time. The returned `Node.Key.String()` is the stable TailnetID; `Node.ComputedName` is the MagicDNS base name used as the default alias. The desktop owner's TailnetID is the sentinel `"local"` — it is NOT the real node key — which means a same-machine browser (which does get the real node key from WhoIs) produces a different composite key and therefore a different presence entry (satisfying success criterion 5 without any extra logic).

**Primary recommendation:** Use dedicated frame-type constants 0x30–0x34 (the range established in Phase 151 ARCHITECTURE.md research and confirmed in STATE.md). Define all five constants now; dispatch only 0x32/0x33/0x34 in Phase 152. Use server-side `time.AfterFunc` per-personKey typing timers inside the Hub. Cap alias at 32 runes, printable Unicode only. Persist aliases in `~/.config/agenthub/aliases.json` via a new lightweight daemon-side AliasStore.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| TailnetID resolution (web path) | API / Backend (webserver.handleWSSRelay) | — | WhoIs is a daemon-local socket call; must happen server-side before any subscriber is created |
| TailnetID for desktop owner (relay path) | API / Backend (relay.handleSession) | — | All loopback connections use the sentinel "local"; no Tailscale call needed |
| Composite person key computation | API / Backend (both handlers) | — | key = TailnetID + ":" + origin; computed once at subscribe time, stored on Subscriber |
| Presence roster (refcount + broadcast) | API / Backend (relay.Hub) | — | Hub owns subscriber lifecycle; presence state mirrors subscriber map under hub.mu |
| Typing TTL timer | API / Backend (relay.Hub) | — | Timers must survive connection loss; Hub outlives individual subscribers |
| Alias persistence | API / Backend (daemon AliasStore) | — | Daemon-owned per D-02; must survive daemon restart |
| Default alias derivation | API / Backend (webserver/relay handlers) | — | WhoIs ComputedName → LoginName prefix → fallback |
| MsgAliasSet / MsgTyping dispatch | API / Backend (both read pump goroutines) | — | Application-protocol frames parsed in the existing read pump switch |
| Presence / typing display | Browser / Client | — | Out of scope for Phase 152; Phase 154/155 responsibility |

## Standard Stack

### Core (no new dependencies — all capabilities already present)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `tailscale.com/client/local` | v1.98.3 (in go.mod) | `local.Client.WhoIs` → peer TailnetID + hostname | Already used for `GetCertificate` in webserver; zero new dep [VERIFIED: direct-codebase-inspection of go.mod] |
| `encoding/json` | stdlib | PresencePayload / TypingPayload / AliasPayload marshal | All existing relay frames already use this encoding path |
| `sync` | stdlib | AliasStore.mu (RWMutex); Hub presence/typing roster locking | Mirrors existing hub.mu pattern |
| `time` | stdlib | `time.AfterFunc` for typing TTL timers | No external timer library needed |

### No New Packages

Phase 152 installs zero new packages — all required capabilities exist in the shipped go.mod and stdlib. [VERIFIED: direct-codebase-inspection]

## Package Legitimacy Audit

No external packages are added in this phase. The only packages used are:
- `tailscale.com` v1.98.3 — already in go.mod, used since Phase 87 [VERIFIED: direct-codebase-inspection]
- Standard library only for all new code

**Packages removed due to SLOP verdict:** none
**Packages flagged as suspicious SUS:** none

## Architecture Patterns

### System Architecture Diagram

```
Client (browser or Wails)        Server (daemon process)
         │                                │
         │  [upgrade WS]                  │
         │─────────────────────────────►  │  handleWSSRelay or handleSession
         │                                │    1. lc.WhoIs(r.RemoteAddr)   ←──── tailscaled (local socket)
         │                                │    2. compute personKey = TailnetID+":"+origin
         │                                │    3. look up alias from AliasStore (or use default)
         │                                │    4. hub.Subscribe(sub)   ─────────────────────►  Hub
         │                                │    5. Hub adds sub to subscribers map               │
         │                                │    6. Hub increments presenceRoster[personKey]      │
         │                                │    7. NotifyPresence(hub)   ◄─── BroadcastPresence  │
         │  ◄── MsgPresence (0x32) ──────────────────── fan-out to ALL subscribers ────────────┘
         │
         │  [user types in composer]
         │─── MsgTyping {typing:true} (0x33) ─────────►  read pump goroutine
         │                                                   hub.UpdateTyping(personKey, alias, true)
         │                                                   → reset 5s TTL timer for personKey
         │                                                   → build TypingPayload
         │  ◄── MsgTyping broadcast (0x33) ────────────────  fan-out to ALL OTHER subscribers
         │
         │  [5s TTL fires / user stops typing]
         │  ◄── MsgTyping {typing:false} broadcast ─────────  timer goroutine → hub.UpdateTyping(..., false)
         │
         │  [user sets alias via UI]
         │─── MsgAliasSet {alias:"ken"} (0x34) ────────►  read pump goroutine
         │                                                   validate alias (≤32 runes, printable)
         │                                                   AliasStore.Set(personKey, alias)
         │                                                   sub.Alias = alias
         │                                                   update presenceRoster[personKey].Alias
         │  ◄── MsgPresence (full roster) ─────────────────  BroadcastPresence → all subscribers
         │
         │  [client disconnects]
         │  read pump returns → readDone closes → handler exits
         │                                         defer hub.Unsubscribe(sub)
         │                                           → decrement presenceRoster[personKey].ConnCount
         │                                           → if ConnCount == 0: remove entry, cancel typing timer
         │  ◄── MsgPresence (updated roster) ──────────────  BroadcastPresence → remaining subscribers
```

### Recommended Project Structure

New files introduced by Phase 152:

```
internal/relay/
├── protocol.go       # + MsgChatSend/MsgChat/MsgPresence/MsgTyping/MsgAliasSet constants
│                     # + PresenceEntry, PresencePayload, TypingPayload, AliasPayload structs
│                     # + MakePresenceFrame, MakeTypingFrame, MakeAliasSetFrame helpers
├── hub.go            # + Subscriber.{TailnetID, Origin, PersonKey, Alias} fields
│                     # + Hub.{presenceRoster, typingRoster} maps (under hu.mu)
│                     # + BroadcastPresence, UpdateTyping, NotifyPresence methods
│                     # + Subscribe updated to init presence state
│                     # + Unsubscribe updated to decrement refcount + cancel typing timer
internal/daemon/
├── alias_store.go    # NEW: AliasStore (JSON file persistence of alias map)
│                     #   ~/.config/agenthub/aliases.json → map[personKey]alias
internal/relay/
├── hub_presence_test.go  # NEW: unit tests for presence refcount, typing TTL, BroadcastPresence
├── protocol_presence_test.go  # NEW: frame encode/decode round-trip for new frame types
internal/daemon/
├── alias_store_test.go   # NEW: AliasStore get/set/persist/reload
internal/webserver/
├── server.go         # MODIFIED: WhoIs call in handleWSSRelay; MsgAliasSet/MsgTyping dispatch
internal/relay/
├── server.go         # MODIFIED: identity stamping in handleSession; MsgAliasSet/MsgTyping dispatch
```

### Pattern 1: Dedicated Frame Type Constants (NOT MetaPayload Extension)

**Decision (resolving Gray Area 1):** Use dedicated frame types in the 0x30-0x3F range, NOT MetaPayload extension. Rationale documented here:

**Why NOT extend MetaPayload:**
- MsgMeta (0x20) is server→client only per the existing comment "Reserved range 0x20-0x2F for future server-push frame types." `MsgTyping` must be bidirectional, which a MetaPayload extension cannot express.
- MetaPayload currently carries `ViewerCount`; mixing viewer count, presence roster, and typing roster in one JSON struct conflates semantically distinct frame purposes and makes partial-update semantics ambiguous.
- The Phase 151 ARCHITECTURE.md research (produced as part of the v4.1 milestone design) already settled on 0x30-0x34, and STATE.md "Modified files" lists "protocol.go (frame constants 0x30–0x34)" — this is the established baseline. [VERIFIED: direct-codebase-inspection of .planning/STATE.md and .planning/research/ARCHITECTURE.md]

**Backward compatibility:** The TypeScript `relayClient.ts:parseServerFrame` function has an explicit `default: return { type: 'unknown' }` at line 61. Unknown frame types (including the new 0x32/0x33 server→client frames) are returned as `{ type: 'unknown' }` and silently ignored in the `onmessage` handler. Old clients that haven't been updated to handle presence/typing frames will silently ignore them. [VERIFIED: direct-codebase-inspection of frontend/src/lib/relayClient.ts:61]

**Constants to add in `internal/relay/protocol.go`:**

```go
// Source: direct codebase inspection of internal/relay/protocol.go
// (current constants: MsgOutput 0x01, MsgResize 0x02, MsgTitle 0x03,
//  MsgInput 0x10, MsgResize2 0x11, MsgPing 0x12, MsgMeta 0x20)

// Chat and presence frame types — 0x30-0x3F reserved for chat/presence.
const (
    MsgChat     byte = 0x30 // server → client: deliver chat message (JSON ChatMessage)     [Phase 154 dispatch]
    MsgChatSend byte = 0x31 // client → server: send chat message (JSON content)            [Phase 154 dispatch]
    MsgPresence byte = 0x32 // server → client: full presence roster (JSON PresencePayload) [Phase 152]
    MsgTyping   byte = 0x33 // bidirectional: typing-start/stop (JSON TypingPayload)         [Phase 152]
    MsgAliasSet byte = 0x34 // client → server: set/update alias (JSON AliasPayload)        [Phase 152]
)
```

Define all five now to lock the wire protocol before Phase 154 implements MsgChat/MsgChatSend dispatch. Phase 152 only DISPATCHES 0x32, 0x33, 0x34 — the chat-message frames are stubs until Phase 154.

**Payload structs to add in `internal/relay/protocol.go`:**

```go
// PresenceEntry describes one participant in the presence roster.
// PersonKey = TailnetID + ":" + origin — the stable collapse key.
type PresenceEntry struct {
    PersonKey string `json:"personKey"`
    TailnetID string `json:"tailnetID"`  // "local" for desktop owner, node pubkey for web
    Origin    string `json:"origin"`     // "local" or "web"
    Alias     string `json:"alias"`
    ConnCount int    `json:"connCount"`  // active connections for this person key
}

// PresencePayload is the JSON body of MsgPresence frames (server → client).
// Full roster on every change — clients replace, not patch.
type PresencePayload struct {
    Participants []PresenceEntry `json:"participants"`
}

// TypingPayload is the JSON body of MsgTyping frames (bidirectional).
// Client → server: PersonKey and Alias are left empty (server fills them from Subscriber).
// Server → client: PersonKey and Alias are populated before broadcast.
type TypingPayload struct {
    PersonKey string `json:"personKey,omitempty"`
    Alias     string `json:"alias,omitempty"`
    Typing    bool   `json:"typing"`
}

// AliasPayload is the JSON body of MsgAliasSet frames (client → server).
type AliasPayload struct {
    Alias string `json:"alias"`
}
```

**Helper functions:**

```go
// MakePresenceFrame encodes a MsgPresence frame.
func MakePresenceFrame(p PresencePayload) []byte { ... }

// MakeTypingFrame encodes a MsgTyping frame.
func MakeTypingFrame(p TypingPayload) []byte { ... }
```

### Pattern 2: Hub Presence Roster with Reference Counting

**Subscriber struct additions** (`internal/relay/hub.go`):

```go
// Source: internal/relay/hub.go:9 (existing Subscriber struct)
type Subscriber struct {
    Msgs      chan []byte
    CloseSlow func()
    ReadOnly  bool
    Name      string // existing ?client= hint
    Cols      int
    Rows      int

    // Phase 152 identity fields — set once at subscribe time, read by read pump
    TailnetID string // "local" or Tailscale node public key string
    Origin    string // "local" (relay loopback) or "web" (webserver Tailscale)
    PersonKey string // TailnetID + ":" + Origin — the collapse key (D-04)
    Alias     string // current display name (mutable via MsgAliasSet)
}
```

**Hub struct additions:**

```go
type Hub struct {
    // ... existing fields ...

    // Phase 152 presence/typing state — guarded by mu
    presenceRoster map[string]*presenceState  // personKey → state
    typingRoster   map[string]*time.Timer     // personKey → 5s TTL timer
}

type presenceState struct {
    TailnetID string
    Origin    string
    Alias     string
    ConnCount int
}
```

**Subscriber lifecycle with presence (Subscribe):**

```go
func (h *Hub) Subscribe(sub *Subscriber) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.subscribers[sub] = struct{}{} // existing
    if sub.PersonKey != "" {
        if s, ok := h.presenceRoster[sub.PersonKey]; ok {
            s.ConnCount++
        } else {
            h.presenceRoster[sub.PersonKey] = &presenceState{
                TailnetID: sub.TailnetID,
                Origin:    sub.Origin,
                Alias:     sub.Alias,
                ConnCount: 1,
            }
        }
    }
    // Caller must call NotifyPresence(hub) after Subscribe returns (outside mu)
}
```

**Subscriber lifecycle with presence (Unsubscribe):**

```go
// Unsubscribe removes the subscriber and returns true if a presence broadcast is warranted.
// Callers call NotifyPresence(hub) when true is returned, outside mu.
func (h *Hub) Unsubscribe(sub *Subscriber) (presenceChanged bool) {
    h.mu.Lock()
    defer h.mu.Unlock()
    delete(h.subscribers, sub) // existing
    if sub.PersonKey != "" {
        if s, ok := h.presenceRoster[sub.PersonKey]; ok {
            s.ConnCount--
            if s.ConnCount <= 0 {
                delete(h.presenceRoster, sub.PersonKey)
                // Cancel typing timer for this person
                if t, ok := h.typingRoster[sub.PersonKey]; ok {
                    t.Stop()
                    delete(h.typingRoster, sub.PersonKey)
                }
                presenceChanged = true
            }
        }
    }
    return presenceChanged
}
```

NOTE: The existing `Unsubscribe` signature returns nothing. The changed signature must be accounted for in both call sites (`relay/server.go` and `webserver/server.go`). The existing `defer func() { hub.Unsubscribe(sub); NotifyViewerCount(hub) }()` pattern becomes:

```go
defer func() {
    presenceChanged := hub.Unsubscribe(sub)
    NotifyViewerCount(hub)
    if presenceChanged {
        NotifyPresence(hub)
    }
}()
```

**NotifyPresence function (analogous to NotifyViewerCount):**

```go
// NotifyPresence pushes a MsgPresence frame with the current roster to all subscribers.
// Must be called AFTER Subscribe/Unsubscribe returns (outside hub.mu).
func NotifyPresence(hub *Hub) {
    roster := hub.CurrentPresence() // acquires + releases mu
    frame := MakePresenceFrame(PresencePayload{Participants: roster})
    hub.BroadcastPresence(frame)   // acquires + releases mu internally
}

func (h *Hub) CurrentPresence() []PresenceEntry {
    h.mu.Lock()
    defer h.mu.Unlock()
    entries := make([]PresenceEntry, 0, len(h.presenceRoster))
    for key, s := range h.presenceRoster {
        entries = append(entries, PresenceEntry{
            PersonKey: key,
            TailnetID: s.TailnetID,
            Origin:    s.Origin,
            Alias:     s.Alias,
            ConnCount: s.ConnCount,
        })
    }
    return entries
}
```

### Pattern 3: Typing TTL via server-side time.AfterFunc

**Decision (resolving Gray Area 2 — typing TTL):**

Server-side `time.AfterFunc(5*time.Second, callback)` stored per personKey in `hub.typingRoster`. This is the ONLY reliable path for clearing typing indicators on abrupt disconnect. The 5s TTL fires regardless of how the connection dies (clean close, TCP timeout, OS kill).

**Why the existing MsgPing is not enough for typing TTL:**

The client's keep-alive `MsgPing 0x12` is sent every 30 seconds from `relayClient.ts` and received as a no-op by the server. [VERIFIED: direct-codebase-inspection of frontend/src/lib/relayClient.ts:96-101 and relay/server.go:281] This 30s interval is too coarse for the 5s typing TTL. A server-side timer is the correct mechanism.

**For presence disconnect:** The existing Subscribe/Unsubscribe lifecycle already handles both cases correctly:
- Clean WS close → `conn.Read(ctx)` returns error → read pump goroutine exits → `readDone` closes → write pump selects on `readDone` and returns → `defer hub.Unsubscribe(sub)` fires.
- Abrupt disconnect → TCP keepalive (OS-level, typically 60-75s) fires → `conn.Read(ctx)` returns error → same path as above.
No additional server-side ping mechanism is needed for PRESENCE. The TCP-level timeout eventually fires and presence clears. Presence accuracy within seconds is less critical than typing (a connected-but-silent user is fine; a stuck "is typing" indicator is annoying).

**Typing TTL implementation:**

```go
// UpdateTyping updates the typing state for the given person key and schedules
// or cancels the 5s auto-clear TTL. Must be called OUTSIDE hub.mu.
func (h *Hub) UpdateTyping(personKey, alias string, typing bool) {
    h.mu.Lock()
    if !typing {
        // Explicit stop — cancel timer
        if t, ok := h.typingRoster[personKey]; ok {
            t.Stop()
            delete(h.typingRoster, personKey)
        }
        h.mu.Unlock()
        // Broadcast typing=false to all subscribers
        NotifyTyping(h, personKey, alias, false)
        return
    }
    // Start / reset timer
    if t, ok := h.typingRoster[personKey]; ok {
        t.Stop()
    }
    h.typingRoster[personKey] = time.AfterFunc(5*time.Second, func() {
        h.mu.Lock()
        delete(h.typingRoster, personKey)
        h.mu.Unlock()
        NotifyTyping(h, personKey, alias, false)
    })
    h.mu.Unlock()
    NotifyTyping(h, personKey, alias, true)
}
```

**Timer callback safety:** The `time.AfterFunc` callback fires in a new goroutine. It acquires `hub.mu` briefly to clean up the typingRoster, then releases mu before calling `NotifyTyping` (which internally acquires mu for the fan-out). This matches the established "compute under mu, broadcast outside mu" pattern. [VERIFIED: pattern confirmed in existing BroadcastMeta and NotifyViewerCount implementations]

**Read pump additions (identical in both relay/server.go and webserver/server.go):**

```go
case MsgTyping:
    var tp TypingPayload
    if json.Unmarshal(payload, &tp) == nil {
        hub.UpdateTyping(sub.PersonKey, sub.Alias, tp.Typing)
    }

case MsgAliasSet:
    var ap AliasPayload
    if json.Unmarshal(payload, &ap) == nil {
        newAlias := validateAlias(ap.Alias)
        if newAlias != "" {
            sub.Alias = newAlias
            // Persist alias via daemon callback (see AliasStore below)
            if setAlias := sub.AliasSetFn; setAlias != nil {
                setAlias(sub.PersonKey, newAlias)
            }
            // Update presenceRoster entry
            hub.UpdateAlias(sub.PersonKey, newAlias)
            NotifyPresence(hub)
        }
    }
```

NOTE: `sub.AliasSetFn` is a new callback field on Subscriber (analogous to `CloseSlow`) that the caller wires to AliasStore.Set to avoid an import cycle between relay and daemon.

### Pattern 4: Identity Stamping at WS Upgrade

**Web path — `handleWSSRelay` in `internal/webserver/server.go`:**

Call `lc.WhoIs` immediately after `websocket.Accept` returns, using `r.RemoteAddr` (the TLS connection's TCP-level remote address, which is the peer's Tailscale IP:port).

```go
// Source: internal/webserver/tailscale.go pattern; adapted for handleWSSRelay
var lc local.Client
tailnetID := "unknown"
defaultAlias := ""
if who, err := lc.WhoIs(r.Context(), r.RemoteAddr); err == nil && who.Node != nil {
    tailnetID = who.Node.Key.String() // "nodekey:hexhex..." — stable across restarts
    if who.Node.ComputedName != "" {
        defaultAlias = who.Node.ComputedName // MagicDNS base name e.g. "ken-macbook"
    } else if who.UserProfile != nil && who.UserProfile.LoginName != "" {
        // Email "alice@example.com" → prefix "alice"
        defaultAlias = strings.SplitN(who.UserProfile.LoginName, "@", 2)[0]
    }
}
personKey := tailnetID + ":web"
alias := getOrDefaultAlias(personKey, defaultAlias) // AliasStore lookup
```

`lc.WhoIs` is a local socket call to tailscaled (not a network call). It completes in milliseconds. Creating `var lc local.Client` inline matches the pattern already used in `tailscale.go:CheckHealth` and `startTailscale`. [VERIFIED: direct-codebase-inspection of internal/webserver/tailscale.go:84-99 and internal/webserver/server.go:384]

**What WhoIs returns for same-tailnet node connecting via web:**

When the desktop owner opens the session's web URL in a browser on the same machine, `r.RemoteAddr` is the machine's own Tailscale IP (e.g., `100.x.x.x:PORT`). `lc.WhoIs` returns the local node's own `Node.Key.String()` — a real nodekey string, NOT the sentinel `"local"`. This means:
- Desktop owner (relay/loopback path): TailnetID = `"local"`, Origin = `"local"`, personKey = `"local:local"`
- Same-machine browser (webserver/Tailscale path): TailnetID = `"nodekey:abc..."`, Origin = `"web"`, personKey = `"nodekey:abc...:web"`

These are different composite keys by construction, satisfying success criterion 5 (no silent identity merge) WITHOUT any special-casing logic. [ASSUMED: WhoIs behavior for self-IP — based on tailscale source analysis; verify with live UAT in Phase 155]

**Relay/loopback path — `handleSession` in `internal/relay/server.go`:**

```go
// All loopback connections are the daemon owner; no WhoIs needed.
tailnetID := "local"
origin := "local"
personKey := "local:local"
alias := initialOwnerAlias // from hub.OwnerAlias (set at Hub creation by daemon)
```

The owner's initial alias is derived from `engine.hostname` (already captured via `os.Hostname()` at daemon startup). The engine passes this to the Hub at creation or provides it via a callback. [VERIFIED: direct-codebase-inspection of internal/daemon/engine.go:295 — hostname field]

### Pattern 5: Alias Persistence (AliasStore)

**Decision (resolving Gray Area 3 — alias validation + persistence design):**

New file `internal/daemon/alias_store.go`. The AliasStore is global (not per-session) because alias identity is per-person, not per-session — the same user wants the same alias across all their sessions.

```
~/.config/agenthub/
├── settings.json    (existing)
├── chats/           (existing — Phase 151)
└── aliases.json     (NEW — Phase 152: map[personKey]alias)
```

**AliasStore API:**

```go
type AliasStore struct {
    mu       sync.RWMutex
    filePath string
    aliases  map[string]string // personKey → alias
}

func NewAliasStore(configDir string) (*AliasStore, error) { ... }
func (a *AliasStore) Get(personKey string) (string, bool) { ... }
func (a *AliasStore) Set(personKey, alias string) error { ... }  // validates + persists
```

The AliasStore is held on `SessionEngine` (alongside `chatStores`). Access is via callbacks on the Subscriber to avoid import cycles (same pattern as `resizeFn` on Hub and `chatHandler` on Hub in Phase 151 ARCHITECTURE.md).

**Alias validation bounds (resolving Gray Area 3 — concrete values):**

```go
// validateAlias returns the alias if valid and non-empty after trimming,
// or "" if the alias should be rejected.
func validateAlias(raw string) string {
    trimmed := strings.TrimSpace(raw)
    if trimmed == "" {
        return ""
    }
    runes := []rune(trimmed)
    if len(runes) > 32 {
        return "" // reject (not truncate — caller should inform the sender)
    }
    for _, r := range runes {
        // Reject C0 controls (U+0000-U+001F) and C1 controls (U+007F-U+009F)
        if r < 0x0020 || (r >= 0x007F && r <= 0x009F) {
            return ""
        }
    }
    return trimmed
}
```

**Rationale for concrete bounds:**
- 32 runes (not bytes): Unicode-safe; fits on one line in the rollup copy `Ken, Sam +2 typing…`; half the existing `clientName` cap of 64 [VERIFIED: internal/relay/server.go:183 and internal/webserver/server.go:986]
- Trim-then-reject (not truncate): rejecting preserves intent and forces clients to send a valid alias; truncation silently changes the user's input
- Non-unique: TailnetID is the stable identity; alias is a human label
- "printable" Unicode: excludes ASCII control chars (C0: 0x00-0x1F) and C1 controls (0x7F-0x9F), which could interfere with terminal rendering on the web-share surface

### Anti-Patterns to Avoid

- **Extending MetaPayload for presence/typing:** MetaPayload is server→client only and currently carries viewer count. Adding presence and typing to it creates a semantically overloaded frame with three different kinds of content. Dedicated frame types are cleaner, already established in the Phase 151 ARCHITECTURE.md, and backward-compatible (old TS client ignores unknown type bytes via `default: return { type: 'unknown' }`).
- **Per-connection presence:** Each subscriber has its own presence entry rather than collapsing by personKey. This gives two entries for "Ken" when he has two browser tabs open, which contradicts D-03. Always collapse by personKey.
- **Storing typing indicators in ChatStore:** The 5-second TTL and the "never persisted" success criterion mean typing state must live exclusively in memory (Hub's typingRoster). Any code path that calls `AppendMessage` for a typing event is a bug.
- **Blocking the typing timer on hub.mu:** `time.AfterFunc` callbacks fire in their own goroutine. The callback must acquire `hub.mu` briefly (to clean up the timer entry), release it, then broadcast. Never hold `hub.mu` across a `BroadcastPresence` call — BroadcastMeta acquires mu internally and a second acquisition would deadlock.
- **Calling lc.WhoIs after WebSocket Accept fails:** The `websocket.Accept` call may return an error (bad origin, etc.) and `handleWSSRelay` already returns early in that case. WhoIs must only be called after a successful Accept.
- **Using TailnetID "local" as a real node key:** The relay/loopback path stamps TailnetID = "local" (sentinel). Never pass this through lc.WhoIs or treat it as a Tailscale node public key. It is simply a stable label for the daemon owner's loopback connections.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Typing debounce timer | Custom goroutine per subscriber with `time.Sleep` | `time.AfterFunc` per personKey in the Hub | AfterFunc fires in a separate goroutine, is cancellable with Stop(), and doesn't leak when Unsubscribe fires concurrently |
| Alias persistence | SQLite or embedded DB | Simple JSON file via `encoding/json` + `os.WriteFile` | Same pattern used for settings.json; no cross-identity queries needed; single writer (AliasStore.mu serializes) |
| Tailscale peer identity | Manual IP→hostname lookup | `local.Client.WhoIs` | Cryptographically verified by the Tailscale control plane; already in go.mod; correct for this use case |
| Frame parsing for new types | New binary decoder | Extend the existing `ParseFrame` + `json.Unmarshal` chain | All existing frame types use ParseFrame → type byte switch → json.Unmarshal; consistent and tested |
| Presence "full replace" vs "diff patch" | Delta presence updates | Full roster on every change | Roster is tiny (typical 2-5 participants); full replace eliminates client-side state reconciliation bugs; BroadcastPresence is identical to BroadcastMeta in cost |

**Key insight:** The Hub's subscriber map is already a reference-counted presence system — we're not adding a new concept, just making it visible to clients via a new frame type.

## Common Pitfalls

### Pitfall 1: WhoIs Called on Wrong Address

**What goes wrong:** `lc.WhoIs(ctx, r.RemoteAddr)` may fail or return wrong data if called before `websocket.Accept` (the request context is different) or if `r.RemoteAddr` is a loopback address (e.g., in test/dev where the webserver is bound to 127.0.0.1 instead of a Tailscale IP).

**Why it happens:** In dev mode the webserver may be bound to loopback. `lc.WhoIs("127.0.0.1:PORT")` returns a "peer not found" error (constant `local.ErrPeerNotFound`). [VERIFIED: direct-codebase-inspection of go/pkg/mod/tailscale.com@v1.98.3/client/local/local.go:330]

**How to avoid:** Always check `err == nil && who != nil && who.Node != nil` before using the WhoIs result. If WhoIs fails, fall back to `tailnetID = "unknown-" + r.RemoteAddr` (or a sanitized form) and `defaultAlias = ""`. This allows tests to run with a fake webserver without a live Tailscale daemon.

**Warning signs:** `500 ms` timing failures in tests where no Tailscale daemon is running. The webserver tests already mock WhoIs indirectly via the `local.Client` zero-value path.

### Pitfall 2: Timer Callback Firing After Hub Shutdown

**What goes wrong:** A `time.AfterFunc` typing timer fires after `hub.Shutdown()` is called. The callback tries to broadcast to an already-shut-down Hub, potentially panicking or sending to closed channels.

**Why it happens:** Hub shutdown is signaled by `close(h.done)` but `time.AfterFunc` timers don't observe this signal.

**How to avoid:** The timer callback should check `h.closed` under `hub.mu` before broadcasting:

```go
h.typingRoster[personKey] = time.AfterFunc(5*time.Second, func() {
    h.mu.Lock()
    if h.closed {
        h.mu.Unlock()
        return // hub already shut down; skip broadcast
    }
    delete(h.typingRoster, personKey)
    h.mu.Unlock()
    NotifyTyping(h, personKey, alias, false)
})
```

**Warning signs:** Panics or race detector hits in tests that shut down the Hub before all timers expire.

### Pitfall 3: Alias Validation Applied Inconsistently

**What goes wrong:** The `validateAlias` function is called in the relay read pump but not in the webserver read pump (or vice versa), allowing longer or control-char-containing aliases from one surface.

**Why it happens:** The two read pumps in `relay/server.go` and `webserver/server.go` are structurally identical but separately maintained. A fix in one is easily missed in the other.

**How to avoid:** Put `validateAlias` in `internal/relay/protocol.go` (alongside the frame type constants) so both handlers import it from the same package. Test both paths explicitly in integration tests.

**Warning signs:** The webserver-path handler returning 200 for aliases that the relay handler rejects (or vice versa).

### Pitfall 4: presenceRoster and typingRoster Not Initialized

**What goes wrong:** `Hub.presenceRoster` or `Hub.typingRoster` is `nil` at Subscribe time (zero-value map) → nil map assignment panics.

**Why it happens:** `NewHub` currently initializes `subscribers: make(map[*Subscriber]struct{})` but would not initialize the two new maps unless explicitly added. [VERIFIED: direct-codebase-inspection of internal/relay/hub.go:51-61]

**How to avoid:** Add `presenceRoster: make(map[string]*presenceState)` and `typingRoster: make(map[string]*time.Timer)` to the `NewHub` constructor return statement.

### Pitfall 5: MsgTyping Broadcast Echoes Back to Sender

**What goes wrong:** The typing indicator briefly flashes for the user who is typing (they see "you are typing" in their own UI).

**Why it happens:** `BroadcastPresence` fans out to ALL subscribers including the one whose Subscriber triggered the event.

**How to avoid:** For typing specifically, broadcast to all subscribers EXCEPT `sub` (the sender). This requires a new `BroadcastExcept(frame []byte, exclude *Subscriber)` helper:

```go
func (h *Hub) BroadcastExcept(frame []byte, exclude *Subscriber) {
    h.mu.Lock()
    defer h.mu.Unlock()
    for sub := range h.subscribers {
        if sub == exclude { continue }
        select {
        case sub.Msgs <- frame:
        default:
            go sub.CloseSlow()
        }
    }
}
```

`MsgPresence` (full roster) should still go to ALL subscribers including the sender, so they see the updated roster after an alias change.

### Pitfall 6: Alias Change Not Propagated on Reconnect

**What goes wrong:** User sets alias in session A. They reconnect. Their presence entry shows the default alias (MagicDNS name) instead of the persisted one.

**Why it happens:** The AliasStore lookup happens at Subscribe time. If the AliasStore callback is not wired to the Subscriber before Subscribe is called, the default alias is used for the new connection.

**How to avoid:** Set up the alias callback BEFORE calling `hub.Subscribe(sub)`. The ordering in both handlers must be: create sub → wire callbacks → look up alias from AliasStore → set sub.Alias → subscribe.

## Code Examples

### WhoIs Call Pattern (Web Path)

```go
// Source: adapted from internal/webserver/tailscale.go:84 (same lc.Client zero-value pattern)
// Called in handleWSSRelay after websocket.Accept succeeds.

var lc local.Client
tailnetID := "unknown"
defaultAlias := ""
if who, err := lc.WhoIs(r.Context(), r.RemoteAddr); err == nil && who.Node != nil {
    tailnetID = who.Node.Key.String() // e.g. "nodekey:abc123def456..."
    if who.Node.ComputedName != "" {
        defaultAlias = who.Node.ComputedName // MagicDNS base name
    } else if who.UserProfile != nil && who.UserProfile.LoginName != "" {
        defaultAlias = strings.SplitN(who.UserProfile.LoginName, "@", 2)[0]
    }
}
```

### Presence Fan-Out Pattern

```go
// Source: mirrors internal/relay/server.go:369 (NotifyViewerCount pattern)

// NotifyPresence pushes a MsgPresence frame to all subscribers (outside mu).
func NotifyPresence(hub *Hub) {
    roster := hub.CurrentPresence()
    frame := MakePresenceFrame(PresencePayload{Participants: roster})
    hub.BroadcastPresence(frame) // mirrors BroadcastMeta
}
```

### Subscribe + Unsubscribe with Presence

```go
// Source: internal/relay/server.go:229-234 (existing Subscribe + NotifyViewerCount pattern)
// Phase 152 addition: set identity on sub before subscribe, call NotifyPresence after.

sub := &relay.Subscriber{
    Msgs:      make(chan []byte, 256),
    ReadOnly:  readonly,
    Name:      clientName,
    TailnetID: tailnetID,    // NEW Phase 152
    Origin:    "local",      // NEW Phase 152 ("local" for relay, "web" for webserver)
    PersonKey: tailnetID + ":local", // NEW Phase 152
    Alias:     alias,        // NEW Phase 152 — from AliasStore or default
    AliasSetFn: func(key, a string) { aliasStore.Set(key, a) }, // NEW Phase 152
}
sub.CloseSlow = func() { conn.Close(websocket.StatusPolicyViolation, "too slow") }

hub.Subscribe(sub)
relay.NotifyViewerCount(hub)    // existing
relay.NotifyPresence(hub)       // NEW Phase 152 — full roster to all subscribers
defer func() {
    presenceChanged := hub.Unsubscribe(sub) // NOW returns bool
    relay.NotifyViewerCount(hub)
    if presenceChanged {
        relay.NotifyPresence(hub)
    }
}()
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Viewer count pushed via MsgMeta + MetaPayload | Same approach for viewer count; dedicated frame types for presence/typing | Phase 152 | Clean separation; MetaPayload stays minimal |
| No client identity on Subscriber | TailnetID + Origin + PersonKey + Alias on Subscriber | Phase 152 | Enables presence attribution and alias persistence |
| Alias in client localStorage, re-sent on reconnect | Alias in daemon aliases.json, persisted per composite key | Phase 152 | Survives daemon restart; owner alias same across sessions |

**Note on ARCHITECTURE.md discrepancy:** The Phase 151 ARCHITECTURE.md described alias as NOT persisted server-side (client localStorage). D-02 from CONTEXT.md overrides this: alias IS daemon-persisted. The planner must not rely on the ARCHITECTURE.md alias-persistence description. [VERIFIED: CONTEXT.md D-02 is the locked decision; ARCHITECTURE.md was written before the discuss-phase settled this]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `lc.WhoIs` returns the local node's own info when called with the local machine's Tailscale IP (same-machine browser case) | Identity Stamping Pattern, Pitfall 1 | If WhoIs returns ErrPeerNotFound for self-IP, the same-machine browser fallback uses "unknown" TailnetID — still distinct from "local", so presence still shows two entries, but with a less-informative TailnetID label |
| A2 | `Node.ComputedName` is the MagicDNS base name (e.g., "ken-macbook") and is always populated for same-tailnet nodes | Code Examples | If ComputedName is empty for connected nodes, the LoginName fallback fires; alias quality degrades but correctness is preserved |
| A3 | `time.AfterFunc` timers are safely stopped by `Timer.Stop()` without the callback having already started, when Stop() is called concurrently | Pitfall 2 | If Stop() loses the race (callback already started), the typing=false broadcast fires twice — idempotent for the client, just redundant |

## Open Questions

1. **AliasStore import cycle scope**
   - What we know: relay.Hub must not import daemon (existing invariant preserved across all prior phases)
   - What's unclear: whether to put AliasStore in relay (breaking if relay needs daemon types) or pass it as callbacks on Subscriber
   - Recommendation: Pass alias operations as callbacks on Subscriber (same pattern as resizeFn on Hub, chatHandler in Phase 151 ARCHITECTURE.md). AliasStore lives in daemon.

2. **Unsubscribe return value API change**
   - What we know: `hub.Unsubscribe(sub)` currently returns nothing; changing it to `(bool)` breaks all callers
   - What's unclear: whether to change the signature or add a new `UnsubscribeWithIdentity` method
   - Recommendation: Change the existing signature — there are exactly two callers (`relay/server.go` and `webserver/server.go`), both updated in this phase. Simpler than a parallel method.

3. **BroadcastExcept for MsgTyping**
   - What we know: typing broadcasts should skip the sender (avoid self-echoing)
   - What's unclear: whether to add a general `BroadcastExcept` or a specific `BroadcastTyping(frame, sender)` method
   - Recommendation: Add `BroadcastExcept(frame, *Subscriber)` — it's general enough to reuse and tests can verify the exclusion.

## Environment Availability

Step 2.6: SKIPPED — Phase 152 is pure Go code changes in existing packages. All external dependencies (Tailscale daemon for WhoIs at WS upgrade time) are verified present via go.mod. No new CLI tools, databases, or services are required.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package + `testing/race` |
| Config file | none (standard `go test`) |
| Quick run command | `go test -race -short ./internal/relay/... ./internal/daemon/...` |
| Full suite command | `go test -race ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| IDENT-01 | TailnetID stamped on Subscriber at subscribe time; visible in presence roster | unit | `go test -race ./internal/relay/ -run TestPresenceRosterTailnetID` | Wave 0 |
| IDENT-01 | Default alias from MagicDNS ComputedName (WhoIs) | unit (mock WhoIs) | `go test -race ./internal/relay/ -run TestDefaultAlias` | Wave 0 |
| IDENT-02 | Alias change via MsgAliasSet propagates to all subscribers within one round-trip | integration (in-process hub) | `go test -race ./internal/relay/ -run TestAliasSetsPropagate` | Wave 0 |
| IDENT-02 | Owner and same-machine browser resolve to distinct presence entries | unit (personKey logic) | `go test -run TestCompositePersonKey` | Wave 0 |
| IDENT-02 | Alias persists across daemon restart (AliasStore reload) | unit | `go test -race ./internal/daemon/ -run TestAliasStorePersist` | Wave 0 |
| PRESENCE-01 | ConnCount increments on Subscribe, decrements on Unsubscribe | unit | `go test -race ./internal/relay/ -run TestPresenceRefCount` | Wave 0 |
| PRESENCE-01 | presenceChanged=true returned from Unsubscribe when last connection drops | unit | `go test -race ./internal/relay/ -run TestUnsubscribePresenceChanged` | Wave 0 |
| PRESENCE-01 | BroadcastPresence sends MsgPresence (0x32) to all subscribers | unit | `go test -race ./internal/relay/ -run TestBroadcastPresence` | Wave 0 |
| PRESENCE-02 | TypingPayload round-trips through JSON marshal/unmarshal | unit | `go test ./internal/relay/ -run TestTypingPayloadRoundTrip` | Wave 0 |
| PRESENCE-02 | MsgTyping from client updates typingRoster; broadcast sent to all EXCEPT sender | unit | `go test -race ./internal/relay/ -run TestTypingBroadcastExcludeSender` | Wave 0 |
| PRESENCE-02 | Typing TTL fires after 5s and broadcasts typing=false | unit (fake timer) | `go test -race ./internal/relay/ -run TestTypingTTLExpiry` | Wave 0 |
| PRESENCE-02 | Unsubscribe cancels typing timer for that personKey | unit | `go test -race ./internal/relay/ -run TestTypingTimerCancelledOnUnsubscribe` | Wave 0 |
| PRESENCE-02 | Hub shutdown with active typing timer does not panic | unit | `go test -race ./internal/relay/ -run TestHubShutdownWithActiveTypingTimer` | Wave 0 |
| IDENT-02 | Same-machine browser vs owner: two distinct personKeys | manual + live UAT | Manual item M-NN (TESTING.md update required) | N/A |
| PRESENCE-02 | Typing appears ≤500ms from keystroke (client-side debounce) | manual + live UAT | Manual item M-NN | N/A |

**Manual-only justification:** The ≤500ms timing requires a real browser and real keypress timing. The owner-vs-same-machine-browser test requires two actual WS connections (Wails app + a real browser), which cannot be automated without the full Tailscale network and live daemon.

### Sampling Rate

- **Per task commit:** `go test -race -short ./internal/relay/... ./internal/daemon/...`
- **Per wave merge:** `go test -race ./...` (full suite)
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- `internal/relay/hub_presence_test.go` — covers PRESENCE-01 refcount, BroadcastPresence, PRESENCE-02 typing TTL, BroadcastExcept
- `internal/relay/protocol_presence_test.go` — covers frame encode/decode for PresencePayload, TypingPayload, AliasPayload (frame type constants 0x32/0x33/0x34)
- `internal/daemon/alias_store_test.go` — covers AliasStore Get/Set/persist/reload

No existing test files need modification for Wave 0 (new test files only).

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Tailscale provides the identity; WhoIs is the verification |
| V3 Session Management | no | Session lifecycle is existing relay Hub; no new session tokens |
| V4 Access Control | yes — D-06 enforcement | `sub.ReadOnly` check already gates PTY input; chat/presence does NOT add new access control requirements beyond existing `MsgInput` gate |
| V5 Input Validation | yes | `validateAlias` function on MsgAliasSet payload; JSON unmarshal on TypingPayload |
| V6 Cryptography | no | No new cryptographic operations; TailnetID comes from Tailscale's cryptographic identity |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Alias injection of control characters | Tampering | `validateAlias` rejects any rune < U+0020 or in C1 range (U+007F-U+009F); applied server-side in both read pumps |
| Alias spoofing another user's identity | Spoofing | Alias is a label only; TailnetID (from Tailscale's cryptographic WhoIs) is the authoritative identity; non-unique aliases are explicitly allowed (D-03) |
| Presence flooding via rapid connect/disconnect | Denial of Service | Hub's non-blocking BroadcastPresence with CloseSlow mirrors existing BroadcastMeta; slow subscribers are dropped, not blocking the fan-out |
| Typing storm (rapid MsgTyping frames) | Denial of Service | Server-side typing TTL resets on each frame (no accumulation); consider a server-side rate limit of 2/sec per personKey (Phase 151 ARCHITECTURE.md recommendation) — planner should include in implementation tasks |
| Alias persistence file path traversal | Tampering | AliasStore file lives at a fixed hardcoded path `aliases.json` under `daemonConfigDir()`; no user-controlled path component |
| RO cap holder triggering MsgAliasSet or MsgTyping | Privilege escalation | NOT a security risk — both operations are benign chat-layer actions explicitly included in D-06 (RO viewers are full chat participants). Only PTY input is gated on ReadOnly. |

## Sources

### Primary (HIGH confidence — direct codebase inspection)
- `internal/relay/protocol.go` — existing frame type constants (MsgOutput 0x01 through MsgMeta 0x20); MetaPayload struct; ChatMessage struct [VERIFIED: direct-codebase-inspection]
- `internal/relay/hub.go` — Subscriber struct fields; Hub struct + mutex patterns; BroadcastMeta non-blocking fan-out; Subscribe/Unsubscribe lifecycle [VERIFIED: direct-codebase-inspection]
- `internal/relay/server.go` — handleSession read pump switch; NotifyViewerCount pattern; loopback identity ("local") [VERIFIED: direct-codebase-inspection]
- `internal/webserver/server.go` — handleWSSRelay structure; `var lc local.Client` creation pattern [VERIFIED: direct-codebase-inspection]
- `internal/webserver/tailscale.go` — CheckHealth `var lc local.Client` pattern; lc.WhoIs call site [VERIFIED: direct-codebase-inspection]
- `frontend/src/lib/relayClient.ts` — parseServerFrame `default: return { type: 'unknown' }` backward-compat path; 30s MsgPing interval [VERIFIED: direct-codebase-inspection]
- `internal/daemon/engine.go` — hostname field (os.Hostname at startup); chatStores map pattern; KillSession teardown [VERIFIED: direct-codebase-inspection]
- `internal/daemon/chat.go` — maxChatLineBytes = 1 MiB constant [VERIFIED: direct-codebase-inspection]
- `~/go/pkg/mod/tailscale.com@v1.98.3/client/tailscale/apitype/apitype.go` — WhoIsResponse struct: Node *tailcfg.Node, UserProfile *tailcfg.UserProfile [VERIFIED: direct-codebase-inspection]
- `~/go/pkg/mod/tailscale.com@v1.98.3/tailcfg/tailcfg.go` — Node.Key NodePublic, Node.ComputedName string, UserProfile.LoginName string [VERIFIED: direct-codebase-inspection]
- `~/go/pkg/mod/tailscale.com@v1.98.3/types/key/node.go` — NodePublic.String() returns "nodekey:hexhex..." [VERIFIED: direct-codebase-inspection]
- `.planning/research/ARCHITECTURE.md` — Phase 151 architecture decisions: 0x30-0x34 frame constants, callback pattern, subscriber identity fields [VERIFIED: direct-codebase-inspection]
- `.planning/STATE.md` — "Modified files: protocol.go (frame constants 0x30–0x34)" confirms the established range [VERIFIED: direct-codebase-inspection]

### Secondary (MEDIUM confidence)
- `~/go/pkg/mod/tailscale.com@v1.98.3/client/local/local.go:319` — `func (lc *Client) WhoIs(ctx context.Context, remoteAddr string) (*apitype.WhoIsResponse, error)` signature; ErrPeerNotFound constant [VERIFIED: direct-codebase-inspection]
- `~/go/pkg/mod/github.com/coder/websocket@v1.8.14/conn.go:213` — `func (c *Conn) Ping(ctx context.Context) error` available (not used in Phase 152) [VERIFIED: direct-codebase-inspection]

## Metadata

**Confidence breakdown:**
- Frame protocol design: HIGH — directly verified against existing code + Phase 151 established baseline
- WhoIs call pattern: HIGH — tailscale library signature verified; same-machine browser behavior ASSUMED for WhoIs self-IP
- Typing TTL via time.AfterFunc: HIGH — standard Go pattern; locking analysis confirmed against hub.go mutex usage
- Alias validation bounds: MEDIUM — 32-rune cap is a judgment call; the "no existing relevant precedent" gap is documented

**Research date:** 2026-06-25
**Valid until:** 2026-07-25 (Go relay protocol is stable; tailscale.com v1.98.3 is pinned in go.mod)
