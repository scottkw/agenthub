# Phase 153: @session PTY Bridge — Research

**Researched:** 2026-06-26
**Domain:** Go daemon / relay protocol security — PTY injection gating, sanitization, binary frame protocol
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01: Phase 153 is a backend/protocol security slice.** Build the daemon inject handler, the dedicated inject frame, the RW-cap gate, the sanitizer, and the persist+broadcast of the `SessionInject:true` message. Validate end-to-end via Go unit/integration tests and direct WS frames — **not** through a composer UI. The visible composer trigger, the press-and-hold gesture, and the rendered indicator are deferred to Phase 154 where the composer exists.

**D-02: Injection is anchored to a dedicated inject frame/verb** (e.g. a new `MsgSessionInject` constant), structurally separate from a normal chat-send frame. A stray Enter, an autocomplete keypress, or an ordinary chat message can **never** reach the PTY, because only this distinct frame triggers `WriteInput`. This _is_ the "deliberate confirm" guarantee at this phase's layer.

**D-03: Sanitizer policy — strip beyond the literal C0+escape list:**
- Strip C0 control characters (0x00–0x1F excluding use as newline collapse marker)
- Strip terminal escape sequences: CSI (ESC [ ... final) and OSC (ESC ] ... ST/BEL)
- Collapse embedded newlines (LF/CR/CRLF) to single spaces
- **Also** strip C1 controls (0x80–0x9F as Unicode runes U+0080–U+009F)
- **Also** strip Unicode bidi overrides (RLO, LRO, PDF, LRI, RLI, FSI, PDI, LRM, RLM, ALM)
- Then append exactly one trailing `\n`
- Invariant: only printable text + exactly one trailing `\n` ever reaches `WriteInput`

**D-04: Server rejects + returns an explicit error/NAK frame.** The daemon refuses the inject (no PTY write) for any RO-cap holder and emits an explicit error frame the client can surface later. The gate is **server-side** and must hold against a hand-crafted WS frame regardless of any client-side suppression.

### Claude's Discretion

- **Exact frame constant + range.** The concrete byte value for the inject verb and the error/NAK frame shape. Planner decides, consistent with `protocol.go` conventions.
- **Which WS paths carry inject.** Both entry paths must enforce the gate. Planner decides whether the owner path needs the inject verb at all or only the web path does.
- **Inject message authorship/alias.** What `AuthorAlias`/identity the persisted `SessionInject:true` message carries — reuse the existing identity-stamping path.
- **Optional hardening (confirm token, rate-limit, audit log).** Not required by the success criteria. Planner may propose in the threat model.

### Deferred Ideas (OUT OF SCOPE)

- **Press-and-hold gesture UI + rendered "→ injected into terminal" indicator + RO affordance hiding** — Phase 154.
- **Confirm token / rate-limiting / audit logging of injects** — optional hardening; not locked scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MENTION-02 | `@session <text>` injects the message into the agent's PTY as a prompt — one-way only; gated to RW-capability holders; sanitized before injection; the chat shows a "→ injected into terminal" indicator. | D-02 (dedicated frame), D-04 (RW gate), D-03 (sanitizer), ChatMessage.SessionInject already persisted+broadcast |
| MENTION-03 | `@session` injection requires a deliberate confirm step to prevent accidental prompts into the agent. | D-02: only the dedicated MsgSessionInject frame triggers WriteInput; normal chat-send (MsgChatSend 0x31) never writes to PTY |
| SEC-01 | Read-only capability holders cannot post chat messages or trigger `@session` injection (enforced server-side, not by UI suppression). | D-04: RO gate in read-pump switch, !sub.ReadOnly, must hold against hand-crafted WS frame; both WS paths verified |
| SEC-02 | Text injected via `@session` into the PTY is sanitized — C0 control characters and terminal escape sequences stripped, newlines collapsed, exactly one trailing newline appended. | D-03: SanitizePTYText state machine; corpus test covers LF/CR/CRLF/NUL/CSI/OSC/C1/bidi |
</phase_requirements>

---

## Summary

Phase 153 is a pure Go backend slice with no frontend component. It adds one new binary frame verb to the relay protocol, one sanitizer function, and one case to the WebSocket read-pump switch in both entry paths (relay loopback and web-share). The danger surface is narrow and load-bearing: `Hub.WriteInput` writes directly to PTY stdin, so the RW-cap gate (SEC-01) and the sanitizer (SEC-02) must both be bulletproof before any UI rides on top.

The codebase is well-prepared. Phase 152 already stamped per-connection identity onto `Subscriber` (TailnetID, Alias, PersonKey, ReadOnly), which the inject handler reads. Phase 151 already landed `ChatMessage.SessionInject` as a field in `relay.ChatMessage`, and `ChatStore.Export()` already renders `_injected into terminal_` for those messages. Phase 152 also established `ValidateAlias` as the canonical rune-scanning style for input validation — the sanitizer mirrors that style with a state machine for multi-character escape sequences.

The primary open design question (resolved below as a recommendation) is the wiring pattern for ChatStore access from inside the read pump, which lives in the `relay` package and cannot import `daemon`. The existing callback pattern (AliasSetFn on Subscriber, chatHistoryProvider on WebServer) is the right model.

**Primary recommendation:** Add `Hub.SetChatAppendFn` + `Hub.BroadcastChat` to the relay package; wire the callback in `engine.go`'s `CreateSession` immediately after both the Hub and the ChatStore are constructed. The read-pump cases in `relay/server.go` and `webserver/server.go` then call `hub.HandleInject(sub, text)`, which gates, sanitizes, writes PTY, appends, and broadcasts — all in one atomic operation on the hub level.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| RW-cap gate (SEC-01) | API / Backend (relay read pump) | — | Gate is enforced at the WebSocket message handler; sub.ReadOnly is stamped from signed JWT claims or loopback query param; must hold server-side |
| PTY stdin write | API / Backend (Hub.WriteInput) | — | Hub owns the PTY writer; relay package; no UI involvement |
| Text sanitization (SEC-02) | API / Backend (relay package) | — | Pure function in relay package; called before WriteInput; no UI involvement |
| Persist inject event | API / Backend (daemon ChatStore) | — | ChatStore.AppendMessage is in daemon package; reached via callback to avoid import cycle |
| Broadcast inject to all subscribers | API / Backend (Hub.BroadcastChat) | — | Hub fan-out to all Subscriber.Msgs channels; relay package |
| RO-cap NAK frame | API / Backend (relay read pump → sub.Msgs) | — | Error frame returned to originating subscriber only; no PTY write |
| "→ injected into terminal" indicator | Frontend (Phase 154) | — | Rendering of SessionInject:true messages is Phase 154 work; this phase only PERSISTS the flag |

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `unicode/utf8` | Go 1.26.3 | Rune decoding for sanitizer state machine | Already used throughout codebase; no new import |
| Go stdlib `unicode` | Go 1.26.3 | Unicode category checks (IsPrint, bidi codepoint ranges) | Already in go.mod transitively |
| `github.com/coder/websocket` | existing pinned | WebSocket binary frame write (NAK reply) | Already used in relay/server.go and webserver/server.go |
| `encoding/json` | stdlib | InjectPayload / InjectErrorPayload serialization | Consistent with all other frame payloads |

**No new Go modules required.** All capabilities are met by existing dependencies in `go.mod`.

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `testing` | Go 1.26.3 | Sanitizer corpus test + adversarial WS frame test | Consistent with every existing test in the repo |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Rune-scan state machine for sanitizer | `regexp.MustCompile` | Regex is more compact but harder to audit for completeness; a state machine makes every byte-class decision explicit and maps directly to the test corpus |
| Hub.SetChatAppendFn callback | Passing ChatStore into Hub at NewHub | Passing the concrete *daemon.ChatStore would create a relay→daemon import cycle; callbacks break the cycle |
| MsgSessionInject in 0x30-0x3F chat range | New range 0x21-0x2F | The 0x21-0x2F range is labeled "server-push" in protocol.go comments; chat/presence verbs all live in 0x30-0x3F; consistency favors extending the chat range |

**Installation:** No new packages — `go build ./...` installs nothing new.

---

## Package Legitimacy Audit

Not applicable. This phase introduces no new external packages. All implementation uses existing `go.mod` entries and Go stdlib.

---

## Architecture Patterns

### System Architecture Diagram

```
Client (RW-cap)        Client (RO-cap)
     |                      |
     | MsgSessionInject(0x35)| MsgSessionInject(0x35)
     v                      v
 +-----------------------------------------+
 |   WS Read Pump (relay/server.go         |
 |              webserver/server.go)        |
 |                                         |
 |   case MsgSessionInject:                |
 |     if sub.ReadOnly → NAK frame → sub  |
 |     else:                               |
 |       text = parse InjectPayload.Text   |
 |       hub.HandleInject(sub, text)       |
 +-----------------------------------------+
           |
           v
 +---------------------------+
 | Hub.HandleInject(sub,text)|
 |   SanitizePTYText(text)   |
 |          |                |
 |          v                |
 |   Hub.WriteInput(bytes)   | → PTY stdin
 |          |                |
 |   chatAppendFn(msg)       | → ChatStore.AppendMessage
 |          |                |      (callback, no import cycle)
 |   Hub.BroadcastChat(frame)| → all Subscriber.Msgs channels
 +---------------------------+
           |
           v
  MsgChat(0x30) frame → all subscribers (including RO viewers)
  (contains ChatMessage{ SessionInject:true, AuthorID, AuthorAlias, Content })
```

### Recommended Project Structure

```
internal/
├── relay/
│   ├── protocol.go          # Add MsgSessionInject(0x35), MsgInjectError(0x36),
│   │                        #   InjectPayload, InjectErrorPayload,
│   │                        #   MakeSessionInjectFrame, MakeInjectErrorFrame, MakeChatFrame
│   ├── sanitize.go          # NEW: SanitizePTYText state machine
│   ├── sanitize_test.go     # NEW: corpus test (LF/CR/CRLF/NUL/CSI/OSC/C1/bidi)
│   ├── hub.go               # Add chatAppendFn field, SetChatAppendFn,
│   │                        #   BroadcastChat, HandleInject
│   ├── server.go            # Add case MsgSessionInject to read-pump switch
│   └── server_inject_test.go # NEW: relay-path RO rejection test + write-path smoke
├── webserver/
│   ├── server.go            # Add case relay.MsgSessionInject to read-pump switch
│   └── inject_test.go       # NEW: web-share path RO rejection test (adversarial frame)
└── daemon/
    └── engine.go            # Wire hub.SetChatAppendFn after CreateSession
```

### Pattern 1: Dedicated Frame Verb as Structural Guard (D-02)

**What:** The inject verb `MsgSessionInject` (byte 0x35) is distinct from `MsgChatSend` (0x31). The read-pump switch only calls `WriteInput` for `MsgSessionInject`; `MsgChatSend` (Phase 154) will only append to chat, never touch the PTY. No code path connects a normal chat message to PTY stdin.

**When to use:** Whenever a privileged side-effect (PTY write) must be structurally impossible from a normal user action. The frame boundary IS the confirm step at this protocol layer.

**Example** (adding to protocol.go):
```go
// Source: internal/relay/protocol.go (existing pattern — extend this block)
const (
    // ... existing chat/presence constants at 0x30–0x34 ...
    MsgSessionInject byte = 0x35 // client → server: inject text into session PTY (RW only)
    MsgInjectError   byte = 0x36 // server → client: inject rejected (RO cap or sanitizer error)
)

// InjectPayload is the JSON body of a MsgSessionInject frame (client → server).
type InjectPayload struct {
    Text string `json:"text"`
}

// InjectErrorPayload is the JSON body of a MsgInjectError frame (server → client).
type InjectErrorPayload struct {
    Reason string `json:"reason"`
}

// MakeChatFrame encodes a ChatMessage as a MsgChat frame (server → client).
func MakeChatFrame(msg ChatMessage) []byte {
    b, _ := json.Marshal(msg)
    frame := make([]byte, 1+len(b))
    frame[0] = MsgChat
    copy(frame[1:], b)
    return frame
}

// MakeInjectErrorFrame encodes a reason string as a MsgInjectError frame.
func MakeInjectErrorFrame(reason string) []byte {
    b, _ := json.Marshal(InjectErrorPayload{Reason: reason})
    frame := make([]byte, 1+len(b))
    frame[0] = MsgInjectError
    copy(frame[1:], b)
    return frame
}
```

### Pattern 2: SanitizePTYText State Machine (D-03)

**What:** A rune-scanning state machine in `internal/relay/sanitize.go`. Mirrors `ValidateAlias` style (rune loop, no regex) but adds states for multi-character escape sequences (CSI/OSC). Output invariant: only printable text + exactly one trailing `\n`.

**When to use:** Any text from an untrusted client before it reaches `WriteInput`. Never skip it.

**Example** (new file `internal/relay/sanitize.go`):
```go
// Source: internal/relay/sanitize.go (new — mirrors ValidateAlias style)
package relay

import "strings"

// SanitizePTYText sanitizes user-supplied text before it is written to PTY stdin.
// It collapses CR/LF/CRLF newlines to single spaces, strips C0 control characters
// (0x00–0x1F), strips terminal escape sequences (CSI and OSC), strips C1 controls
// (U+0080–U+009F), strips Unicode bidi-override characters (U+202A–U+202E,
// U+2066–U+2069, U+200E, U+200F, U+061C), then appends exactly one trailing '\n'.
//
// Output invariant: the returned string contains only printable text followed by
// exactly one newline. This is the only text that must ever reach Hub.WriteInput.
func SanitizePTYText(input string) string {
    const (
        stateNormal    = iota
        stateEscape    // saw ESC (0x1B); next byte decides CSI/OSC/other
        stateCSI       // inside ESC [ ... ; skip until final byte 0x40–0x7E
        stateOSC       // inside ESC ] ... ; skip until BEL or ESC
        stateOSCEscape // inside OSC, saw ESC; next '\' ends it
    )
    state := stateNormal
    var b strings.Builder
    b.Grow(len(input) + 1)
    runes := []rune(input)
    for _, r := range runes {
        switch state {
        case stateNormal:
            switch {
            case r == '\n' || r == '\r':
                b.WriteRune(' ') // collapse newline to space
            case r == 0x1B:
                state = stateEscape
            case r >= 0x00 && r <= 0x1F:
                // C0 control (excluding 0x1B handled above): skip
            case r == 0x7F:
                // DEL: skip
            case r >= 0x80 && r <= 0x9F:
                // C1 control (Unicode U+0080–U+009F): skip
            case isBidiOverride(r):
                // Unicode bidi override: skip (CVE-2021-42574 class)
            default:
                b.WriteRune(r)
            }
        case stateEscape:
            switch r {
            case '[':
                state = stateCSI
            case ']':
                state = stateOSC
            default:
                state = stateNormal // discard ESC + this byte
            }
        case stateCSI:
            if r >= 0x40 && r <= 0x7E {
                state = stateNormal // final byte consumed; CSI complete
            }
            // parameter/intermediate bytes (0x20–0x3F): remain in CSI
        case stateOSC:
            switch r {
            case 0x07: // BEL terminates OSC
                state = stateNormal
            case 0x1B:
                state = stateOSCEscape
            }
            // all other bytes: remain in OSC (skip)
        case stateOSCEscape:
            if r == '\\' {
                state = stateNormal // ESC \ = String Terminator
            } else {
                state = stateOSC // not ST; remain in OSC
            }
        }
    }
    result := strings.TrimRight(b.String(), " ")
    return result + "\n"
}

// isBidiOverride returns true for Unicode bidi override / directional formatting
// characters that can be used to spoof terminal output (Trojan Source class).
func isBidiOverride(r rune) bool {
    switch r {
    case 0x061C, // ARABIC LETTER MARK
        0x200E, 0x200F, // LRM, RLM
        0x202A, 0x202B, 0x202C, 0x202D, 0x202E, // LRE, RLE, PDF, LRO, RLO
        0x2066, 0x2067, 0x2068, 0x2069: // LRI, RLI, FSI, PDI
        return true
    }
    return false
}
```

### Pattern 3: Hub Callback Wiring (chatAppendFn / import-cycle break)

**What:** `Hub` gains a `chatAppendFn func(ChatMessage) (ChatMessage, error)` field set by `engine.go` after both Hub and ChatStore are created. The relay package never imports daemon — the callback closure captures the `*ChatStore` reference without an explicit type dependency.

**When to use:** Any time the relay read pump needs to call a daemon-layer function. Same pattern as `resizeFn` in `NewHub`, `AliasSetFn` in Subscriber, `getAlias`/`setAlias` in relay.Server.

**Example** (additions to `internal/relay/hub.go`):
```go
// Source: internal/relay/hub.go (existing Hub struct — add field)
// Phase 153: persist+broadcast callback. Set by engine.go after CreateSession.
// Nil-safe: HandleInject skips persist+broadcast if unset (PTY write still occurs).
chatAppendFn func(ChatMessage) (ChatMessage, error)

// SetChatAppendFn wires the persist callback. Must be called before the first
// WebSocket connection is accepted for this session (i.e. from engine.go
// CreateSession, not from a read pump). Safe to call concurrently with mu.
func (h *Hub) SetChatAppendFn(fn func(ChatMessage) (ChatMessage, error)) {
    h.mu.Lock()
    h.chatAppendFn = fn
    h.mu.Unlock()
}

// BroadcastChat sends a MsgChat frame to all subscribers (fan-out identical
// to BroadcastPresence). The frame is pre-encoded by the caller.
func (h *Hub) BroadcastChat(frame []byte) {
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

// HandleInject is called by the read pump when a MsgSessionInject frame arrives.
// It: (1) gates on !sub.ReadOnly, (2) sanitizes, (3) writes to PTY stdin,
// (4) persists via chatAppendFn, (5) broadcasts the MsgChat frame to all subs.
// Returns ErrReadOnly when sub.ReadOnly == true (caller sends NAK frame).
func (h *Hub) HandleInject(sub *Subscriber, text string) error {
    if sub.ReadOnly {
        return ErrReadOnly
    }
    sanitized := SanitizePTYText(text)
    if err := h.WriteInput([]byte(sanitized)); err != nil {
        return err
    }
    if h.chatAppendFn != nil {
        msg, err := h.chatAppendFn(ChatMessage{
            AuthorID:      sub.TailnetID,
            AuthorAlias:   sub.Alias,
            Content:       text, // persist original, not sanitized
            SessionInject: true,
        })
        if err == nil {
            h.BroadcastChat(MakeChatFrame(msg))
        }
    }
    return nil
}
```

### Pattern 4: Read-Pump Switch Case (relay path)

**What:** Add `case MsgSessionInject:` to the existing switch in `relay/server.go` read pump (around line 323). Gate is `!sub.ReadOnly`. On rejection, write NAK to `sub.Msgs` (non-blocking, since sub.Msgs is buffered 256 frames).

**Example** (addition to `internal/relay/server.go` read-pump switch, mirroring MsgInput gate):
```go
// Source: internal/relay/server.go:~325 — add after existing MsgInput case
case MsgSessionInject:
    // SEC-01: RW cap required. Gate is server-side; must hold against direct WS
    // frame injection regardless of any client-side suppression.
    var ip InjectPayload
    if json.Unmarshal(payload, &ip) != nil || ip.Text == "" {
        continue // malformed frame: ignore silently
    }
    if err := hub.HandleInject(sub, ip.Text); err != nil {
        // Send NAK frame to originating subscriber only.
        // Non-blocking: sub.Msgs is buffered 256 frames.
        select {
        case sub.Msgs <- MakeInjectErrorFrame(err.Error()):
        default:
            go sub.CloseSlow()
        }
    }
```

The identical case (with `relay.` package prefix on all symbols) is added to `webserver/server.go` handleWSSRelay's read-pump switch.

### Pattern 5: Engine Wiring (engine.go)

**What:** After `manager.Create(sessionID, ...)` returns the `*Hub` and `NewChatStore` constructs the `*ChatStore`, wire the callback.

**Example** (addition to `internal/daemon/engine.go` CreateSession, after both are available):
```go
// Source: internal/daemon/engine.go CreateSession (existing flow — wire after Hub+ChatStore ready)
// Phase 153: wire the inject persist callback so the relay read pump can
// append SessionInject messages without importing daemon.
hub.SetChatAppendFn(func(msg relay.ChatMessage) (relay.ChatMessage, error) {
    return chatStore.AppendMessage(msg)
})
```

### Anti-Patterns to Avoid

- **Gating inject on client-asserted query param:** The loopback relay path currently uses `?readonly=1` as a client-asserted hint for `sub.ReadOnly`. For the inject gate, `sub.ReadOnly` is the authoritative check — no secondary query-param check needed. The gate must survive a hand-crafted frame that omits or falsifies any URL parameter.
- **Storing sanitized text in ChatMessage.Content:** Persist the original `ip.Text` (pre-sanitize) in `ChatMessage.Content` so the chat thread shows what the user typed, not the sanitized form. Only the bytes sent to `WriteInput` are sanitized.
- **Skipping the NAK on malformed payload:** Return silently for JSON-unmarshal failures (same pattern as MsgTyping/MsgAliasSet). Only the RO-rejection case emits an explicit NAK frame.
- **Testing only the relay path:** Both WS entry paths must be tested for the RO rejection. The web path reads `claims.Perms == "read"` from a signed JWT — test it with a hand-crafted frame from a RO-JWT session.
- **Calling BroadcastChat while holding hub.mu:** All existing broadcast methods acquire mu internally. `HandleInject` must call `WriteInput` and `chatAppendFn` before calling `BroadcastChat`, and must not hold `mu` during any of these calls (mirrors ResizeClient discipline).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CSI/OSC escape detection | Custom byte-pair matching | State machine in SanitizePTYText | Escape sequences have variable-length parameter fields; simple prefix matching misses sequences with intermediate bytes |
| Cap verification | Custom JWT parsing | `claims.Perms == "read"` / `sub.ReadOnly` from existing `requireCapability` middleware | The web path already validates the JWT before handleWSSRelay runs; loopback path already sets ReadOnly from ?readonly=1 |
| Message ID generation | Custom rand | `daemon.AppendMessage` auto-fills `msg.ID` via `randomHexID()` when left empty | ChatStore.AppendMessage fills ID, TimestampMs, SchemaVersion, SessionID automatically when caller leaves them zero |
| Chat persistence | New file append logic | `ChatStore.AppendMessage` | Already implements the append-only JSONL pattern with cap enforcement, mutex serialization, and disk/mirror consistency |

**Key insight:** The most dangerous code in this phase is `hub.WriteInput`. Every byte that reaches it must pass through `SanitizePTYText` with no exceptions. There is no "fast path" for trusted sources — the relay loopback owner runs through the same sanitizer as the web-share participant.

---

## Verified Code Anchors

All line numbers verified against live code (2026-06-26):

| Symbol | File | Actual Location | Notes |
|--------|------|-----------------|-------|
| `Subscriber.ReadOnly` | `internal/relay/hub.go` | line 18 | Field comment: "if true, input frames from this client are discarded by the server read pump. (MC-03)" |
| `Hub.WriteInput` | `internal/relay/hub.go` | lines 409–413 | `func (h *Hub) WriteInput(data []byte) error` — direct `h.writer.Write(data)` call |
| MsgInput (0x10) | `internal/relay/protocol.go` | line 16 | `MsgInput byte = 0x10 // Client keyboard input → PTY stdin` |
| MsgMeta (0x20) | `internal/relay/protocol.go` | line 22 | `MsgMeta byte = 0x20` — server-push frame; reserved range comment says 0x20-0x2F |
| MsgChat (0x30) | `internal/relay/protocol.go` | line 80 | Phase 154 dispatch stub; server → client |
| MsgChatSend (0x31) | `internal/relay/protocol.go` | line 81 | Phase 154 dispatch stub; client → server |
| MsgPresence (0x32) | `internal/relay/protocol.go` | line 83 | Phase 152; server → client |
| MsgTyping (0x33) | `internal/relay/protocol.go` | line 83 | Phase 152; bidirectional |
| MsgAliasSet (0x34) | `internal/relay/protocol.go` | line 84 | Phase 152; client → server — next available is **0x35** |
| `ChatMessage.SessionInject` | `internal/relay/protocol.go` | line 220 | `SessionInject bool \`json:"sessionInject,omitempty"\`` — field already exists |
| `ValidateAlias` | `internal/relay/protocol.go` | lines 157–172 | Style precedent for rune-scan sanitizer; strips C0 (0x00–0x1F) and C1 (0x7F–0x9F) |
| Read-pump switch (relay path) | `internal/relay/server.go` | lines 323–364 | MsgInput case at line 325; MsgTyping/MsgAliasSet deliberately not gated |
| `readonly` derivation (relay path) | `internal/relay/server.go` | line 209 | `readonly := r.URL.Query().Get("readonly") == "1" || ...` |
| `handleWSSRelay` | `internal/webserver/server.go` | line 999 | `readonly := claims.Perms == "read"` at line 1008 |
| Read-pump switch (web path) | `internal/webserver/server.go` | lines 1115–1157 | MsgInput case at line 1116; identical structure to relay path |
| `ChatStore.AppendMessage` | `internal/daemon/chat.go` | lines 244–308 | Signature: `func (s *ChatStore) AppendMessage(msg relay.ChatMessage) (relay.ChatMessage, error)` |
| `ChatStore.Export` | `internal/daemon/chat.go` | lines 321–341 | `_injected into terminal_` marker rendered at line 335 for `msg.SessionInject == true` |
| `engine.ChatStoreFor` | `internal/daemon/engine.go` | lines 297–302 | Returns (*ChatStore, bool) — no hub handle; hub accessed via `manager.Get` |
| `HubManager.Create` | `internal/relay/manager.go` | lines 26–38 | Returns `*Hub`; also calls `go hub.Run()` |
| `claims.Perms == "read"` (web gate) | `internal/webserver/server.go` | line 1008 | Sourced from signed JWT via `capability.ClaimsFromContext(r.Context())` |

**CONTEXT.md line-number drift:** The CONTEXT.md cited `handleWSSRelay` at "~:972/996"; actual start is line 999. The hub.WriteInput at ":409" is correct. ChatStore `AppendMessage/Export` at "~:316–335" in CONTEXT.md refers to Export (321) and the inject marker (335); AppendMessage starts at line 244. All other anchors match within ±5 lines. [VERIFIED: live code read]

---

## Common Pitfalls

### Pitfall 1: CRLF vs LF handling in the sanitizer
**What goes wrong:** A client sends `"hello\r\nworld"`. A naive sanitizer strips `\r` but not `\n`, producing `"hello\nworld\n"` — two newlines reach the PTY; the agent sees two separate commands.
**Why it happens:** Newline handling is platform-specific; web browsers often send CRLF.
**How to avoid:** The state machine collapses `\r` and `\n` (individually OR as a sequence) to a single space. `TrimRight` on trailing spaces before appending the single `\n` ensures no trailing spaces.
**Warning signs:** Test corpus includes `"hello\r\nworld"` → must produce `"hello world\n"`. [ASSUMED based on common terminal behavior]

### Pitfall 2: C1 controls as raw bytes vs UTF-8 encoded
**What goes wrong:** C1 controls (0x80–0x9F) arrive either as raw bytes (in Latin-1 strings) or as UTF-8 encoded runes (U+0080–U+009F). Go's `string` type is a byte slice; ranging over a string yields Unicode rune values. A raw byte 0x80 in a non-UTF-8 string decodes as `unicode.ReplacementChar` (U+FFFD), not U+0080.
**Why it happens:** The relay receives raw WebSocket binary frames. If the client sends a malformed UTF-8 payload containing raw 0x80–0x9F bytes, `[]rune(input)` replaces them with U+FFFD, which is already printable-safe. The risk case is a well-formed UTF-8 payload with Unicode C1 codepoints (U+0080–U+009F), which ARE valid UTF-8.
**How to avoid:** The sanitizer's rune-scan already handles this: `r >= 0x80 && r <= 0x9F` catches the Unicode C1 range explicitly. The test corpus includes a C1 codepoint such as `"\u0085"` (NEL, NEXT LINE — U+0085, a C1 that also acts as a newline in some terminals). [ASSUMED based on Go string semantics]

### Pitfall 3: Inject verb must NOT be dispatched by MsgChatSend case
**What goes wrong:** Phase 154 adds `case MsgChatSend (0x31):` to the read pump. If a developer accidentally adds PTY-write logic to MsgChatSend, the structural separation between chat and inject breaks.
**Why it happens:** Both frames carry text; the distinction is only by byte value.
**How to avoid:** MsgChatSend's case must explicitly NOT call WriteInput, HandleInject, or SanitizePTYText. Add a code comment in both read-pump switches: `// MsgChatSend: chat message only, NEVER writes to PTY (D-02)`.

### Pitfall 4: hub.mu held during WriteInput or chatAppendFn
**What goes wrong:** If `HandleInject` acquires `hub.mu` before calling `WriteInput` (which calls `h.writer.Write`) or `chatAppendFn` (which calls `ChatStore.AppendMessage` + disk I/O), it holds the mutex during blocking I/O. This starves the broadcast drain loop.
**Why it happens:** Developers familiar with mutex-guarded methods add a lock at the top of HandleInject.
**How to avoid:** Follow `ResizeClient` discipline: call `hub.mu.Unlock()` BEFORE any I/O. `HandleInject` should not hold `hub.mu` at all — `sub.ReadOnly` is read under no lock (it is set once at subscribe time and never mutated), and the callback + broadcast acquire their own internal locks.

### Pitfall 5: RO rejection test only uses relay path
**What goes wrong:** A test only tests `relay.Server`'s handleSession RO gate and passes. But the web path's `handleWSSRelay` has a different `readonly` derivation (`claims.Perms == "read"`) and a different read-pump switch. A developer who correctly gates relay path but forgets the web path passes all tests.
**How to avoid:** The adversarial RO-frame test must cover the web path (via `internal/webserver/inject_test.go`). Use a mock capability JWT with `Perms:"read"` and send a hand-crafted `MsgSessionInject` frame; verify `hub.WriteInput` is never called (mock PTY writer counts writes).

### Pitfall 6: content vs sanitized stored in ChatMessage
**What goes wrong:** Storing `sanitized` (the output of SanitizePTYText) in `ChatMessage.Content` means the chat thread shows truncated/normalized text. Users see their `@session` message with bidi characters stripped and newlines collapsed, which is confusing.
**How to avoid:** Pass `ip.Text` (the original pre-sanitized payload) to `chatAppendFn`. Only the bytes sent to `WriteInput` are sanitized. The chat thread records what the user intended, not the wire representation.

---

## Code Examples

### Sanitizer test corpus structure (internal/relay/sanitize_test.go)

```go
// Source: internal/relay/sanitize_test.go (new — covers SEC-02 invariant)
func TestSanitizePTYText(t *testing.T) {
    cases := []struct {
        name  string
        input string
        want  string
    }{
        {"plain text", "hello world", "hello world\n"},
        {"trailing space stripped", "hello  ", "hello\n"},
        {"LF newline collapsed", "hello\nworld", "hello world\n"},
        {"CR newline collapsed", "hello\rworld", "hello world\n"},
        {"CRLF newline collapsed", "hello\r\nworld", "hello world\n"},
        {"null byte stripped", "hel\x00lo", "hello\n"},
        {"C0 control stripped", "hel\x07lo", "hello\n"}, // BEL
        {"DEL stripped", "hel\x7flo", "hello\n"},
        {"C1 NEL stripped", "hel\u0085lo", "hello\n"},   // U+0085 NEXT LINE
        {"C1 control U+0080 stripped", "hel\u0080lo", "hello\n"},
        {"CSI clear screen", "hello\x1b[2Jworld", "helloworld\n"},
        {"CSI color", "hel\x1b[31mlo", "hello\n"},       // ESC [ 31 m
        {"OSC title", "hi\x1b]0;title\x07there", "hithere\n"},
        {"OSC with ST", "hi\x1b]0;title\x1b\\there", "hithere\n"},
        {"bidi RLO", "hel\u202elo", "hello\n"},           // RIGHT-TO-LEFT OVERRIDE
        {"bidi LRM", "hel\u200elo", "hello\n"},           // LEFT-TO-RIGHT MARK
        {"empty input", "", "\n"},
        {"only newlines", "\n\n\n", "\n"},                // all collapse to spaces, trimmed
        {"mixed attack", "cmd\x1b[A\x00\r\n;evil", "cmd evil\n"},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := SanitizePTYText(tc.input)
            if got != tc.want {
                t.Errorf("SanitizePTYText(%q) = %q, want %q", tc.input, got, tc.want)
            }
        })
    }
}
```

### Adversarial RO frame test structure (relay path)

```go
// Source: internal/relay/server_inject_test.go (new — proves SEC-01 server-side)
func TestInject_ROCapRejected_RelayPath(t *testing.T) {
    // PTY writer that counts writes — any write is a test failure.
    var ptyWriteCount atomic.Int32
    ptyReader, ptyWriter := io.Pipe()
    _ = ptyReader
    countingWriter := writerFunc(func(p []byte) (int, error) {
        ptyWriteCount.Add(1)
        return len(p), nil
    })

    const sessionID = "inject-ro-test"
    manager := NewHubManager()
    manager.Create(sessionID, io.NopCloser(ptyReader), countingWriter, nil)

    ts := httptest.NewServer(NewServer(manager, nil, nil))
    t.Cleanup(func() { ts.Close(); manager.Shutdown(); ptyWriter.Close() })

    // Dial as read-only client.
    wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/sessions/" + sessionID + "/ws?readonly=1"
    conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
    require.NoError(t, err)
    t.Cleanup(func() { conn.CloseNow() })

    // Send hand-crafted MsgSessionInject frame directly (bypasses any client-side suppression).
    payload, _ := json.Marshal(InjectPayload{Text: "evil command"})
    frame := append([]byte{MsgSessionInject}, payload...)
    require.NoError(t, conn.Write(context.Background(), websocket.MessageBinary, frame))

    // Must receive a MsgInjectError NAK frame within timeout.
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    _, rawMsg, err := conn.Read(ctx)
    require.NoError(t, err)
    require.Greater(t, len(rawMsg), 0)
    assert.Equal(t, MsgInjectError, rawMsg[0], "expected NAK frame type")

    // PTY stdin must have received zero bytes.
    assert.Equal(t, int32(0), ptyWriteCount.Load(), "PTY must not be written for RO client")
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Trust client-asserted `?readonly=1` | `sub.ReadOnly` from signed JWT claims (`claims.Perms == "read"`) for web path; query param for relay loopback | Phase 87/88 (D-24/SEC-04) | Web-path gate cannot be bypassed by URL manipulation; relay loopback remains query-param (loopback trust boundary) |
| No chat persistence | JSONL ChatStore with per-session cap | Phase 151 | AppendMessage is available for inject events; no new persistence needed |
| No identity stamping | Phase 152 identity on Subscriber (TailnetID, Alias, PersonKey) | Phase 152 | Inject messages have a real AuthorID/AuthorAlias without extra work |
| No SessionInject field | ChatMessage.SessionInject bool (already landed) | Phase 151 | Export already renders "injected into terminal" marker |

**Deprecated/outdated:**
- The "client→server verb range ~0x21-0x2F" mentioned in CONTEXT.md as reserved: the code's comment says this range is "server-push frame types." The inject verb should follow the chat range (0x30-0x3F) at 0x35, consistent with all existing client→server verbs (MsgInput 0x10, MsgChatSend 0x31, MsgAliasSet 0x34). [ASSUMED: inferred from actual protocol.go; the CONTEXT.md description predates the Phase 152 final constant allocation]

---

## Discretion Recommendations

The planner must decide these. Recommendations are provided to reduce deliberation time.

### Frame constant allocation

**Recommendation:** `MsgSessionInject = 0x35` (client→server), `MsgInjectError = 0x36` (server→client). This extends the established chat/presence range (0x30-0x3F) where all chat-related verbs live. The comment block in protocol.go already scopes this range to "chat/presence."

**Alternative:** Use `MsgError byte = 0x21` for a generic server→client error frame (in the server-push range 0x20-0x2F). Advantage: reusable for other server-side errors in future. Disadvantage: changes the semantic of the 0x21-0x2F range comment and may confuse Phase 154 developers.

### Which WS paths carry inject

**Recommendation:** Both paths implement `case MsgSessionInject:`. The relay loopback owner is the primary RW user of inject (they are always RW, `sub.ReadOnly == false`). The web path is where the SEC-01 gate is most critical. Both paths share the same `hub.HandleInject(sub, text)` implementation. Skipping the relay loopback path would mean the desktop owner cannot inject via their own relay connection — a functional gap.

### Inject message authorship

**Recommendation:** Use `sub.TailnetID` as `AuthorID` and `sub.Alias` as `AuthorAlias` — the same fields the future `MsgChatSend` case will use. The Phase 152 identity stamping already ensures these are set correctly on every Subscriber at subscribe time. No new logic needed.

### Optional hardening candidates (not locked scope)

The planner may note these in the threat model without implementing:
- **Rate limit:** Max N injects per subscriber per second (prevent PTY flood). Could be a simple token bucket in `HandleInject` or at the read-pump level.
- **Audit log:** `log.Printf("inject: authorID=%s session=%s bytes=%d", sub.TailnetID, h.sessionID, len(sanitized))` in HandleInject — metadata only, never content.
- **Max inject payload length:** Reject InjectPayload.Text > N bytes (e.g., 4096) before sanitizing — prevents CPU-expensive state machine runs on megabyte payloads.

---

## Validation Architecture

Nyquist validation is **ENABLED** (key absent from config → treat as enabled).

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package, Go 1.26.3 |
| Config file | none — `go test` discovers by convention |
| Quick run command | `go test -race -short -run TestSanitize\|TestInject ./internal/relay/... ./internal/webserver/...` |
| Full suite command | `go test -race -short ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MENTION-02 | RW-cap inject writes text to PTY stdin and broadcasts MsgChat frame | integration | `go test -race -short -run TestInject_RWCap ./internal/relay/...` | No — Wave 0: create `internal/relay/server_inject_test.go` |
| MENTION-03 | MsgChatSend frame does NOT write to PTY stdin; only MsgSessionInject does | unit | `go test -race -short -run TestInject_OnlyDedicatedFrame ./internal/relay/...` | No — Wave 0: same file |
| SEC-01 (relay path) | RO client sending MsgSessionInject receives NAK frame; WriteInput never called | integration | `go test -race -short -run TestInject_ROCap_RelayPath ./internal/relay/...` | No — Wave 0: create `internal/relay/server_inject_test.go` |
| SEC-01 (web path) | RO JWT client sending MsgSessionInject receives NAK frame; WriteInput never called | integration | `go test -race -short -run TestInject_ROCap_WebPath ./internal/webserver/...` | No — Wave 0: create `internal/webserver/inject_test.go` |
| SEC-02 | SanitizePTYText corpus: LF/CR/CRLF/NUL/CSI/OSC/C1/bidi all stripped; only printable + `\n` survives | unit | `go test -race -short -run TestSanitizePTYText ./internal/relay/...` | No — Wave 0: create `internal/relay/sanitize_test.go` |

### Sampling Rate

- **Per task commit:** `go test -race -short -run TestSanitize\|TestInject ./internal/relay/... ./internal/webserver/...`
- **Per wave merge:** `go test -race -short ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/relay/sanitize_test.go` — covers SEC-02, sanitizer corpus
- [ ] `internal/relay/server_inject_test.go` — covers MENTION-02, MENTION-03, SEC-01 relay path
- [ ] `internal/webserver/inject_test.go` — covers SEC-01 web path (adversarial RO JWT frame)

*(Note: `internal/relay/sanitize.go` and additions to `hub.go`, `protocol.go`, `server.go`, `webserver/server.go`, `engine.go` are implementation files, not test gap files.)*

**TESTING.md must be updated** (per repo standing convention in CLAUDE.md):
- Suite Manifest §2: Go count 359 → 362 (+3 new test files)
- Traceability §4: Add rows for MENTION-02, MENTION-03, SEC-01, SEC-02

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | Indirectly | JWT Claims verified by `requireCapability` middleware upstream of handleWSSRelay; relay path relies on loopback trust + optional ?readonly |
| V3 Session Management | No | WebSocket session lifecycle is managed; no auth session concept in relay protocol |
| V4 Access Control | **Yes** | `sub.ReadOnly` gate in read-pump switch; derived from signed JWT for web path; must hold server-side (SEC-01) |
| V5 Input Validation | **Yes** | `SanitizePTYText` before `WriteInput`; state machine strips C0/C1/CSI/OSC/bidi; corpus test covers all categories (SEC-02) |
| V6 Cryptography | No | No new crypto; existing HMAC-SHA256 JWT in `capability` package is unchanged |

### Known Threat Patterns for this Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| RO client sends MsgSessionInject to write PTY | Elevation of Privilege | `!sub.ReadOnly` gate in read pump; NAK frame returned; zero PTY writes proven by adversarial test |
| Newline injection via `"cmd\nrm -rf /"` | Tampering | `SanitizePTYText` collapses `\n` to space; test corpus includes this case |
| CSI escape injection (`ESC[A` cursor-up etc.) | Tampering | State machine strips CSI sequences; test corpus includes color + movement codes |
| OSC escape injection (terminal title spoof `ESC]0;evil\x07`) | Spoofing | State machine strips OSC sequences including ST-terminated form |
| Trojan Source (bidi override) — CVE-2021-42574 | Spoofing | `isBidiOverride()` strips RLO/LRO/PDF/LRI/RLI/FSI/PDI/LRM/RLM/ALM |
| C1 control as newline (`U+0085 NEL`) | Tampering | `r >= 0x80 && r <= 0x9F` strips all C1 controls including NEL (U+0085) |
| PTY flood (rapid inject frames) | DoS | Optional rate limiting (discretion item; not locked scope) |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Storing `ip.Text` (pre-sanitize) in ChatMessage.Content is the right approach; sanitized text goes only to PTY | Code Examples (Pitfall 6) | If wrong, the persisted message would show stripped/modified text to all chat participants, causing UX confusion; correct by changing which value is passed to chatAppendFn |
| A2 | `MsgSessionInject = 0x35` is the next available constant in the chat range | Discretion Recommendations | If Phase 152 or an unreferenced phase allocates 0x35, collision; planner must verify and shift upward |
| A3 | `SanitizePTYText` should preserve original text in ChatMessage.Content and only sanitize the bytes going to WriteInput | Code Examples, Pattern 3 | If wrong: either raw PTY bytes are stored (security) or user sees corrupted chat (UX). Current design chooses UX fidelity over storing the sanitized form |
| A4 | CRLF should become `" "` (one space), not `""` (discarded) | Code Examples, Pitfall 1 | If collapsing to space is wrong (e.g., the agent expects literal newlines), the planner may want to discuss; but D-03 says "collapse" not "strip" |
| A5 | `hub.HandleInject` should not hold `hub.mu` during WriteInput or chatAppendFn | Common Pitfalls, Pattern 3 | If wrong, PTY write latency or disk I/O would block the broadcast fan-out loop; test with `-race` and a slow mock writer |

**Verified claims:** All code anchor line numbers, field names, constant values, and import cycle analysis are from direct `Read` tool inspection of the live codebase. [VERIFIED: live code read]

---

## Open Questions (RESOLVED)

1. **Should `MsgChatSend (0x31)` be dispatched in this phase or remain a stub?**
   - What we know: The comment in protocol.go marks MsgChatSend as "Phase 154 dispatch stub"; the read-pump switches currently have no `case MsgChatSend:` in their switch statements.
   - What's unclear: Phase 153's scope is inject only. Adding MsgChatSend dispatch here would pull in CHAT-01 scope.
   - RESOLVED: Leave MsgChatSend as a stub in Phase 153. The new `case MsgSessionInject:` is sufficient for this phase's requirements. (Plans follow this — no MsgChatSend dispatch case is added.)

2. **Does the relay loopback path need its inject verb at all for Phase 153 to meet all 4 success criteria?**
   - What we know: Success criterion 1 says "Sending `@session <text>` as a RW-cap holder causes exactly that text to appear in the session PTY stdin." Phase 154 (where the composer lives) will emit frames from the Wails webview, which goes through the relay loopback path.
   - What's unclear: The adversarial test (success criterion 2) focuses on RO WS frame injection, which is primarily the web path concern. The loopback path's `sub.ReadOnly` is always false for the desktop owner.
   - RESOLVED: Implement both paths in Phase 153. The loopback path is where the desktop owner will inject from Phase 154 onward; the gate still needs to be present and tested. (Plans 02 + 03 implement both WS paths.)

---

## Environment Availability

All code is in Go; no external runtime dependencies beyond the existing `go.mod`. Go 1.26.3 is installed. No new CLIs, services, or binaries are required for this phase.

---

## Sources

### Primary (HIGH confidence)

- `internal/relay/protocol.go` — live read; all constants, ChatMessage struct, ValidateAlias implementation verified [VERIFIED: live code read]
- `internal/relay/hub.go` — live read; WriteInput, Subscriber.ReadOnly, BroadcastPresence pattern, all line numbers verified [VERIFIED: live code read]
- `internal/relay/server.go` — live read; read-pump switch, MsgInput gate pattern, identity wiring verified [VERIFIED: live code read]
- `internal/webserver/server.go` — live read; handleWSSRelay, claims.Perms=="read" gate, read-pump switch verified [VERIFIED: live code read]
- `internal/daemon/chat.go` — live read; AppendMessage signature, Export with inject marker, ChatStore internals verified [VERIFIED: live code read]
- `internal/daemon/engine.go` — live read; ChatStoreFor, chatStores map, NewSessionEngine structure verified [VERIFIED: live code read]
- `internal/relay/manager.go` — live read; HubManager.Create returns *Hub; hub.SetChatAppendFn wiring point confirmed [VERIFIED: live code read]
- `internal/relay/server_identity_test.go` — live read; test helpers (setupIdentityTestServer, dialIdentityWS, waitForPresenceFrame) verified as model for inject tests [VERIFIED: live code read]
- `TESTING.md` — live read; current test counts (359 Go, 120 vitest, 7 Playwright, 1 build-script = 487 total) and traceability format [VERIFIED: live code read]

### Secondary (MEDIUM confidence)

- `internal/capability/capability.go` — live read; Claims.Perms field, HasPerm semantics verified [VERIFIED: live code read]
- `internal/daemon/chat_routes.go` — live read; wrapRelayWithChat outer-HTTP pattern; confirms WS read pump has no direct ChatStore access [VERIFIED: live code read]
- `internal/daemon/api.go` — live read; setChatProviders pattern, RelayHandler wiring, wrapRelayWithChat call [VERIFIED: live code read]

### Tertiary (LOW confidence)

- Trojan Source CVE-2021-42574 bidi override attack class — D-03 requirement for bidi stripping [ASSUMED: well-known CVE, standard recommendation]

---

## Metadata

**Confidence breakdown:**
- Code anchors (line numbers, signatures): HIGH — all verified via live Read
- Frame constant allocation recommendation (0x35/0x36): MEDIUM — inferred from protocol.go allocation pattern; planner must verify no other unreferenced allocation exists
- Sanitizer completeness (C1/bidi codepoint set): HIGH — rune ranges verified against Unicode spec conventions; isBidiOverride list matches standard bidi control characters
- Import cycle analysis: HIGH — daemon→relay and relay→daemon dependency graph verified from actual imports

**Research date:** 2026-06-26
**Valid until:** 2026-07-26 (stable protocol; constants won't drift unless Phase 154 allocates 0x35 first)
