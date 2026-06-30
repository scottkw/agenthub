# Phase 152: Relay Protocol + Identity + Presence — Pattern Map

**Mapped:** 2026-06-25
**Files analyzed:** 9 (3 new, 6 modified)
**Analogs found:** 9 / 9

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/relay/protocol.go` (modified) | protocol constants + structs | request-response | `internal/relay/protocol.go` (existing) | self — extend |
| `internal/relay/hub.go` (modified) | fan-out hub | event-driven | `internal/relay/hub.go` (existing) | self — extend |
| `internal/relay/server.go` (modified) | WS handler / read pump | request-response | `internal/relay/server.go` (existing) | self — extend |
| `internal/webserver/server.go` (modified) | WS handler / read pump | request-response | `internal/relay/server.go` | exact (structurally identical read-pump) |
| `internal/daemon/alias_store.go` (new) | service / JSON store | CRUD | `internal/daemon/chat.go` + `engine.go:loadSettingsFromDisk` | role-match |
| `internal/relay/hub_presence_test.go` (new) | test | event-driven | `internal/relay/hub.go` (tested via existing tests) | role-match |
| `internal/relay/protocol_presence_test.go` (new) | test | request-response | `internal/relay/protocol.go` (existing) | role-match |
| `internal/daemon/alias_store_test.go` (new) | test | CRUD | `internal/daemon/chat.go` (existing tests pattern) | role-match |

---

## Pattern Assignments

### `internal/relay/protocol.go` — add chat/presence frame constants + payload structs

**Analog:** The file itself — copy the existing constant block and `MakeMeta`/`MetaPayload` patterns.

**Existing constants block** (lines 11–22) — mirror this style for the new 0x30–0x34 block:
```go
const (
    MsgOutput  byte = 0x01 // PTY stdout/stderr → client
    MsgResize  byte = 0x02
    MsgTitle   byte = 0x03
    MsgInput   byte = 0x10
    MsgResize2 byte = 0x11
    MsgPing    byte = 0x12

    // MsgMeta is the server-to-client metadata push channel (JSON payload).
    // Reserved range 0x20-0x2F for future server-push frame types.
    MsgMeta byte = 0x20
)
```

Add immediately after — new block:
```go
// Chat and presence frame types — 0x30-0x3F reserved for chat/presence.
const (
    MsgChat     byte = 0x30 // server → client: deliver chat message (JSON ChatMessage)     [Phase 154 dispatch]
    MsgChatSend byte = 0x31 // client → server: send chat message (JSON content)            [Phase 154 dispatch]
    MsgPresence byte = 0x32 // server → client: full presence roster (JSON PresencePayload) [Phase 152]
    MsgTyping   byte = 0x33 // bidirectional: typing-start/stop (JSON TypingPayload)         [Phase 152]
    MsgAliasSet byte = 0x34 // client → server: set/update alias (JSON AliasPayload)        [Phase 152]
)
```

**Existing `MetaPayload` / `MakeMeta` pattern** (lines 62–73) — copy this for each new payload type:
```go
type MetaPayload struct {
    ViewerCount *int `json:"viewerCount,omitempty"`
}

func MakeMeta(p MetaPayload) []byte {
    b, _ := json.Marshal(p) // MetaPayload is always serialisable
    frame := make([]byte, 1+len(b))
    frame[0] = MsgMeta
    copy(frame[1:], b)
    return frame
}
```

New payload structs follow the same pointer-omitempty style; helper `Make*` functions follow the same 1-byte-prefix pattern. Implement `MakePresenceFrame(p PresencePayload) []byte` and `MakeTypingFrame(p TypingPayload) []byte` identically to `MakeMeta`.

**`validateAlias` placement:** Put in this file (alongside frame constants) so both read pumps import it from one place. Pattern: same file as related constants (see how `ParseFrame` lives alongside constant definitions at line 52).

---

### `internal/relay/hub.go` — add identity fields to Subscriber + presence/typing roster to Hub

**Analog:** The file itself. All additions follow existing mutex and non-blocking fan-out patterns.

**Existing `Subscriber` struct** (lines 9–26) — append new fields after `Rows int`:
```go
type Subscriber struct {
    Msgs      chan []byte
    CloseSlow func()
    ReadOnly  bool
    Name      string  // ?client= hint
    Cols      int
    Rows      int
    // Phase 152: identity
    TailnetID  string
    Origin     string
    PersonKey  string  // TailnetID + ":" + Origin
    Alias      string
    AliasSetFn func(personKey, alias string) // callback; avoids import cycle with daemon
}
```

**Existing `Hub` struct** (lines 30–46) — append new maps under `mu`:
```go
type Hub struct {
    // ... existing fields unchanged ...
    mu          sync.Mutex
    subscribers map[*Subscriber]struct{}
    done        chan struct{}
    closed      bool
    closeOnce   sync.Once
    ptyCols     int
    ptyRows     int
    // Phase 152: presence/typing — guarded by mu
    presenceRoster map[string]*presenceState  // personKey → collapsed state
    typingRoster   map[string]*time.Timer     // personKey → 5s TTL timer
}
```

**`NewHub` constructor** (lines 51–61) — must initialize both new maps alongside `subscribers`:
```go
return &Hub{
    // existing ...
    subscribers:    make(map[*Subscriber]struct{}),
    done:           make(chan struct{}),
    // Phase 152
    presenceRoster: make(map[string]*presenceState),
    typingRoster:   make(map[string]*time.Timer),
}
```

**Existing `Subscribe`** (lines 74–78) — extend (keep existing body, add presence refcount):
```go
func (h *Hub) Subscribe(sub *Subscriber) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.subscribers[sub] = struct{}{} // existing — do not remove
    if sub.PersonKey != "" {
        if s, ok := h.presenceRoster[sub.PersonKey]; ok {
            s.ConnCount++
        } else {
            h.presenceRoster[sub.PersonKey] = &presenceState{
                TailnetID: sub.TailnetID, Origin: sub.Origin,
                Alias: sub.Alias, ConnCount: 1,
            }
        }
    }
}
```

**Existing `Unsubscribe`** (lines 81–85) — change signature to `(bool)`, extend body:
```go
func (h *Hub) Unsubscribe(sub *Subscriber) (presenceChanged bool) {
    h.mu.Lock()
    defer h.mu.Unlock()
    delete(h.subscribers, sub) // existing — do not remove
    if sub.PersonKey != "" {
        if s, ok := h.presenceRoster[sub.PersonKey]; ok {
            s.ConnCount--
            if s.ConnCount <= 0 {
                delete(h.presenceRoster, sub.PersonKey)
                if t, ok := h.typingRoster[sub.PersonKey]; ok {
                    t.Stop()
                    delete(h.typingRoster, sub.PersonKey)
                }
                presenceChanged = true
            }
        }
    }
    return
}
```

**Existing `BroadcastMeta`** (lines 172–183) — copy verbatim as `BroadcastPresence` and `BroadcastExcept`:
```go
// BroadcastMeta — exact reference for BroadcastPresence (copy, rename, same body)
func (h *Hub) BroadcastMeta(frame []byte) {
    h.mu.Lock()
    defer h.mu.Unlock()
    for sub := range h.subscribers {
        select {
        case sub.Msgs <- frame:
        default:
            go sub.CloseSlow()
        }
    }
}

// BroadcastExcept — same body, one exclusion guard added
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

**`UpdateTyping` locking discipline** — mirrors `ResizeClient` (lines 100–126): acquire mu → mutate state → release mu → call external function AFTER release. Never hold mu across a broadcast:
```go
// ResizeClient reference: mu released BEFORE calling resizeFn (line 120-125)
h.mu.Unlock() // release BEFORE calling resizeFn
if needResize && h.resizeFn != nil {
    return h.resizeFn(maxCols, maxRows)
}
```
`UpdateTyping` must release `h.mu` before calling `NotifyTyping` — same discipline.

**`h.closed` guard in timer callback** — follow the existing `Shutdown` pattern (lines 186–193):
```go
func (h *Hub) Shutdown() {
    h.closeOnce.Do(func() {
        h.mu.Lock()
        h.closed = true   // ← this field exists; timer callbacks must check it
        h.mu.Unlock()
        close(h.done)
    })
}
```

---

### `internal/relay/server.go` — identity stamping in `handleSession` + read pump dispatch

**Analog:** The file itself — `handleSession` lines 177–302.

**Subscriber creation** (lines 218–225) — extend with identity fields before `hub.Subscribe`:
```go
// Current pattern (lines 218-225):
sub := &Subscriber{
    Msgs:     make(chan []byte, 256),
    ReadOnly: readonly,
    Name:     clientName,
}
sub.CloseSlow = func() {
    conn.Close(websocket.StatusPolicyViolation, "too slow")
}

// Phase 152 addition — set identity fields BEFORE Subscribe:
sub.TailnetID = "local"
sub.Origin    = "local"
sub.PersonKey = "local:local"
sub.Alias     = alias   // from AliasStore lookup or engine.hostname fallback
sub.AliasSetFn = func(key, a string) { aliasStore.Set(key, a) }
```

**Subscribe + NotifyViewerCount defer** (lines 229–234) — extend to also call `NotifyPresence`:
```go
// Current pattern (lines 229-234):
hub.Subscribe(sub)
NotifyViewerCount(hub)
defer func() {
    hub.Unsubscribe(sub)
    NotifyViewerCount(hub)
}()

// Phase 152 pattern:
hub.Subscribe(sub)
NotifyViewerCount(hub)
NotifyPresence(hub)   // NEW
defer func() {
    presenceChanged := hub.Unsubscribe(sub) // now returns bool
    NotifyViewerCount(hub)
    if presenceChanged {
        NotifyPresence(hub)
    }
}()
```

**Read pump switch** (lines 267–284) — add two new cases after existing ones:
```go
// Existing cases — reference:
case MsgInput:  // lines 269-276
case MsgResize2: // lines 277-282
case MsgPing:   // line 283 — no-op

// Phase 152 additions:
case MsgTyping:
    var tp TypingPayload
    if json.Unmarshal(payload, &tp) == nil {
        hub.UpdateTyping(sub.PersonKey, sub.Alias, tp.Typing)
    }
case MsgAliasSet:
    var ap AliasPayload
    if json.Unmarshal(payload, &ap) == nil {
        if newAlias := validateAlias(ap.Alias); newAlias != "" {
            sub.Alias = newAlias
            if sub.AliasSetFn != nil {
                sub.AliasSetFn(sub.PersonKey, newAlias)
            }
            hub.UpdateAlias(sub.PersonKey, newAlias)
            NotifyPresence(hub)
        }
    }
```

**`NotifyViewerCount`** (lines 368–372) — exact template to copy as `NotifyPresence`:
```go
func NotifyViewerCount(hub *Hub) {
    count := hub.SubscriberCount()
    frame := MakeMeta(MetaPayload{ViewerCount: &count})
    hub.BroadcastMeta(frame)
}
// Copy as:
func NotifyPresence(hub *Hub) {
    roster := hub.CurrentPresence()
    frame := MakePresenceFrame(PresencePayload{Participants: roster})
    hub.BroadcastPresence(frame)
}
```

---

### `internal/webserver/server.go` — WhoIs call + read pump dispatch in `handleWSSRelay`

**Analog:** `internal/relay/server.go:handleSession` (structurally identical; copy the same additions).

**`var lc local.Client` pattern** (from `internal/webserver/tailscale.go` lines 97–98) — zero-value instantiation, then method call:
```go
// Existing pattern in tailscale.go:97-98:
var lc local.Client
return checkHealth(ctx, lc.StatusWithoutPeers, "", realPrefsFunc(&lc))

// Phase 152 web-path addition — after websocket.Accept succeeds (line 1009):
var lc local.Client
tailnetID := "unknown"
defaultAlias := ""
if who, err := lc.WhoIs(r.Context(), r.RemoteAddr); err == nil && who.Node != nil {
    tailnetID = who.Node.Key.String()
    if who.Node.ComputedName != "" {
        defaultAlias = who.Node.ComputedName
    } else if who.UserProfile != nil && who.UserProfile.LoginName != "" {
        defaultAlias = strings.SplitN(who.UserProfile.LoginName, "@", 2)[0]
    }
}
personKey := tailnetID + ":web"
alias := aliasStore.GetOrDefault(personKey, defaultAlias)
```

**`handleWSSRelay` subscriber creation** (lines 1015–1020) — same extend-before-subscribe pattern as relay/server.go; same read pump switch additions (copy from relay/server.go pattern above, substituting `relay.MsgTyping`, `relay.MsgAliasSet`, `relay.ValidateAlias`).

**`handleWSSRelay` Unsubscribe defer** (lines 1027–1030) — same NotifyPresence extension as relay/server.go above.

---

### `internal/daemon/alias_store.go` (NEW)

**Analog:** `internal/daemon/chat.go` (JSON-based store) + `internal/daemon/engine.go:loadSettingsFromDisk/saveSettingsToDisk`.

**Store struct pattern** — copy `ChatStore` mu+filePath+in-memory-map idiom:
```go
// ChatStore reference (chat.go lines 84-89):
type ChatStore struct {
    mu        sync.Mutex
    filePath  string
    sessionID string
    messages  []relay.ChatMessage
}
```
```go
// AliasStore — same idiom, different payload:
type AliasStore struct {
    mu       sync.RWMutex  // RW: reads are frequent (every subscribe)
    filePath string
    aliases  map[string]string // personKey → alias
}
```

**Constructor pattern** — copy `NewChatStore` structure:
```go
// NewChatStore reference (chat.go lines 99-130):
func NewChatStore(baseDir, sessionID string) (*ChatStore, error) {
    // validate input
    // os.MkdirAll
    // derive filePath
    // containment check
    // store := &ChatStore{...}
    // store.loadFromDisk()
    // return store, nil
}
```
`NewAliasStore(configDir string) (*AliasStore, error)` follows the same shape; `filePath = filepath.Join(configDir, "aliases.json")`.

**Load from disk pattern** — copy `loadSettingsFromDisk` (engine.go lines 173–193):
```go
// engine.go:173-193 reference:
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
    data, err := os.ReadFile(settingsPath(dir))
    if err != nil {
        return // missing file is not an error (first run)
    }
    // pre-populate defaults
    if json.Unmarshal(data, &s) != nil { ... }
}
```
`AliasStore.loadFromDisk`: `os.ReadFile` → if `os.IsNotExist` return nil → `json.Unmarshal` into `map[string]string`.

**Save to disk pattern** — copy `saveSettingsToDisk` (engine.go lines 242–258):
```go
// engine.go:254-258 reference:
data, err := json.Marshal(s)
if err != nil { return }
_ = os.WriteFile(settingsPath(e.configDir), data, 0600)
```
`AliasStore.persist()` (called under mu from `Set`): `json.Marshal(a.aliases)` → `os.WriteFile(a.filePath, data, 0600)`.

**Delete / remove pattern** — copy `ChatStore.Delete` (chat.go lines 347–356): `os.Remove` + ignore `os.IsNotExist`.

**`daemonConfigDir()` reference** (engine.go lines 69–75) — production callers pass `daemonConfigDir()` into `NewAliasStore`; tests pass `t.TempDir()`. Same isolation pattern as `NewChatStore(baseDir, sessionID)`.

**`chatStores` map wiring** (engine.go lines 43, 287) — `aliasStore *AliasStore` field on `SessionEngine` alongside `chatStores map[string]*ChatStore`; instantiated in `NewSessionEngine` (line 295) alongside other stores.

---

### `internal/relay/hub_presence_test.go` (NEW)

**Analog:** The pattern established in existing hub tests. Copy the in-process Hub construction idiom.

**`NewHub` test construction pattern** — instantiate with an `io.Pipe()` reader/writer pair and `nil` resizeFn (mirrors existing hub test setup):
```go
pr, pw := io.Pipe()
hub := NewHub("test-session", pr, pw, 64*1024, nil)
```

**Subscriber test helper pattern** — create a subscriber with a buffered channel and a no-op CloseSlow:
```go
sub := &Subscriber{
    Msgs:      make(chan []byte, 16),
    CloseSlow: func() {},
    PersonKey: "local:local",
    TailnetID: "local",
    Origin:    "local",
    Alias:     "ken",
}
```

**Async broadcast drain** — non-blocking receive from `sub.Msgs` with `t.Helper()` wrapper; test the frame type byte directly (`frame[0] == MsgPresence`).

**`time.AfterFunc` TTL test** — use a very short TTL (1 ms in test mode) by exposing a `typingTTL time.Duration` field on Hub (default `5 * time.Second`, overridden in tests). Test asserts typing=false broadcast arrives within `100 * time.Millisecond`.

---

### `internal/relay/protocol_presence_test.go` (NEW)

**Analog:** Existing protocol tests. Copy round-trip test pattern:

```go
// Pattern: encode → decode → assert fields
frame := MakePresenceFrame(PresencePayload{
    Participants: []PresenceEntry{{PersonKey: "local:local", Alias: "ken", ConnCount: 1}},
})
if frame[0] != MsgPresence { t.Fatalf("wrong type byte") }
var p PresencePayload
if err := json.Unmarshal(frame[1:], &p); err != nil { t.Fatal(err) }
// assert p.Participants[0].Alias == "ken"
```

---

### `internal/daemon/alias_store_test.go` (NEW)

**Analog:** `internal/daemon/chat.go` test pattern. Tests use `t.TempDir()` as `configDir`.

```go
// Pattern from NewChatStore tests:
store, err := NewChatStore(t.TempDir(), "sess1")
// Phase 152 analog:
store, err := NewAliasStore(t.TempDir())
```

Test: `Set` → `Get` → reload via `NewAliasStore(same dir)` → assert alias persisted.

---

## Shared Patterns

### Fan-out Non-blocking Send (applies to BroadcastPresence, BroadcastExcept)
**Source:** `internal/relay/hub.go:BroadcastMeta` lines 172–183
```go
func (h *Hub) BroadcastMeta(frame []byte) {
    h.mu.Lock()
    defer h.mu.Unlock()
    for sub := range h.subscribers {
        select {
        case sub.Msgs <- frame:
        default:
            go sub.CloseSlow()
        }
    }
}
```
**Apply to:** `BroadcastPresence` (identical), `BroadcastExcept` (same + `if sub == exclude { continue }`).

### NotifyViewerCount → NotifyPresence Template
**Source:** `internal/relay/server.go:368–372`
```go
func NotifyViewerCount(hub *Hub) {
    count := hub.SubscriberCount()
    frame := MakeMeta(MetaPayload{ViewerCount: &count})
    hub.BroadcastMeta(frame)
}
```
**Apply to:** `NotifyPresence` — same three-line shape: get state (outside mu) → encode frame → broadcast.

### Subscribe-before-snapshot + defer-Unsubscribe
**Source:** `internal/relay/server.go:229–234` and `internal/webserver/server.go:1025–1031`
Both handlers share the exact same pattern:
```go
hub.Subscribe(sub)
relay.NotifyViewerCount(hub)
defer func() {
    hub.Unsubscribe(sub)
    relay.NotifyViewerCount(hub)
}()
```
**Apply to:** Both handlers — extend to add `NotifyPresence` calls in the same positions.

### JSON Persistence (apply to AliasStore)
**Source:** `internal/daemon/engine.go:173–193` (load) and `242–258` (save)
- Load: `os.ReadFile` → ignore `IsNotExist` → `json.Unmarshal`
- Save: `json.Marshal` → `os.WriteFile(..., 0600)`
**Apply to:** `alias_store.go` — `loadFromDisk` and `persist`.

### `var lc local.Client` Zero-value Pattern
**Source:** `internal/webserver/tailscale.go:97–98`
```go
var lc local.Client
return checkHealth(ctx, lc.StatusWithoutPeers, ...)
```
**Apply to:** `handleWSSRelay` in `internal/webserver/server.go` — `var lc local.Client` → `lc.WhoIs(r.Context(), r.RemoteAddr)`.

### Read Pump Frame Dispatch
**Source:** `internal/relay/server.go:267–284` (switch on `msgType`)
```go
switch msgType {
case MsgInput:   ...
case MsgResize2: ...
case MsgPing:    // no-op
}
```
**Apply to:** Both handlers — add `case MsgTyping:` and `case MsgAliasSet:` cases after existing ones. Do NOT remove existing cases.

### TypeScript `parseServerFrame` Extension
**Source:** `frontend/src/lib/relayClient.ts:43–64`
```typescript
export function parseServerFrame(data: Uint8Array): ServerFrame {
  const typeByte = data[0]
  switch (typeByte) {
    case MSG_OUTPUT: return { type: 'output', payload: data.slice(1) }
    case MSG_RESIZE: { ... return { type: 'resize', cols, rows } }
    default:
      return { type: 'unknown' }  // ← backward-compat: unknown types silently ignored
  }
}
```
**Apply to:** `relayClient.ts` — add `MSG_PRESENCE = 0x32`, `MSG_TYPING = 0x33`, `MSG_ALIAS_SET = 0x34` constants at top (lines 1–6 style). Add `ServerFrame` union variants for presence and typing. Add `case MSG_PRESENCE:` and `case MSG_TYPING:` branches to `parseServerFrame`. The existing `default: return { type: 'unknown' }` ensures old clients ignore the new types safely — this is why no version negotiation is needed.

---

## No Analog Found

None. All files have close analogs in the existing codebase.

---

## Metadata

**Analog search scope:** `internal/relay/`, `internal/daemon/`, `internal/webserver/`, `frontend/src/lib/`
**Files scanned:** 8 source files fully read
**Pattern extraction date:** 2026-06-25
