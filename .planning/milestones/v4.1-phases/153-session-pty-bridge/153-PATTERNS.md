# Phase 153: @session PTY Bridge - Pattern Map

**Mapped:** 2026-06-26
**Files analyzed:** 9 (4 create, 5 modify)
**Analogs found:** 9 / 9

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/relay/protocol.go` (modify) | protocol/config | request-response | itself — existing `MakePresenceFrame`/`MakeTypingFrame`/`MakeAliasSetFrame` + `ValidateAlias` | exact |
| `internal/relay/sanitize.go` (create) | utility | transform | `internal/relay/protocol.go` `ValidateAlias` (lines 157–172) | role-match |
| `internal/relay/sanitize_test.go` (create) | test | transform | `internal/relay/server_identity_test.go` table-test structure | role-match |
| `internal/relay/hub.go` (modify) | service | event-driven | itself — `BroadcastMeta`/`BroadcastPresence`/`ResizeClient` + `AliasSetFn` callback pattern | exact |
| `internal/relay/server.go` (modify) | middleware | request-response | itself — `case MsgInput:` RW-gated block (lines 324–329) | exact |
| `internal/webserver/server.go` (modify) | middleware | request-response | itself — `case relay.MsgInput:` block (lines 1116–1122) + `readonly := claims.Perms == "read"` (line 1008) | exact |
| `internal/daemon/engine.go` (modify) | service | CRUD | itself — `hub := e.manager.Create(...)` + `chatStore, _ := NewChatStore(...)` wiring pattern (lines 413, 442–453) | exact |
| `internal/relay/server_inject_test.go` (create) | test | request-response | `internal/relay/server_identity_test.go` — `setupIdentityTestServer`, `dialIdentityWS`, `waitForPresenceFrame` helpers | exact |
| `internal/webserver/inject_test.go` (create) | test | request-response | `internal/relay/server_identity_test.go` — test server setup + adversarial WS frame pattern | role-match |

---

## Pattern Assignments

### `internal/relay/protocol.go` (modify — add constants + frame builders)

**Analog:** itself, lines 76–142 (existing chat/presence constant block + `Make*Frame` builders)

**Existing constant block** (lines 76–85) — extend after `MsgAliasSet`:
```go
// Chat and presence frame types — 0x30-0x3F reserved for chat/presence.
const (
    MsgChat     byte = 0x30
    MsgChatSend byte = 0x31
    MsgPresence byte = 0x32
    MsgTyping   byte = 0x33
    MsgAliasSet byte = 0x34  // last allocated in Phase 152
    // Phase 153: inject verb and NAK — next in the 0x30-0x3F chat/presence range.
    MsgSessionInject byte = 0x35 // client → server: inject text into PTY (RW only)
    MsgInjectError   byte = 0x36 // server → client: inject rejected (RO cap or error)
)
```

**Existing Make*Frame pattern** (lines 118–142) — copy verbatim structure for two new builders:
```go
// MakePresenceFrame (existing, lines 118–124) — exact pattern to copy:
func MakePresenceFrame(p PresencePayload) []byte {
    b, _ := json.Marshal(p) // always serialisable
    frame := make([]byte, 1+len(b))
    frame[0] = MsgPresence
    copy(frame[1:], b)
    return frame
}
// New builders follow identical shape:
//   MakeChatFrame(msg ChatMessage) []byte  { frame[0] = MsgChat; ... }
//   MakeInjectErrorFrame(reason string) []byte  { frame[0] = MsgInjectError; ... }
```

**New payload structs** — follow `TypingPayload`/`AliasPayload` style (lines 106–115):
```go
type InjectPayload struct {
    Text string `json:"text"`
}
type InjectErrorPayload struct {
    Reason string `json:"reason"`
}
```

**ValidateAlias** (lines 157–172) — style reference for `SanitizePTYText`; note the rune loop, explicit C0/C1 rejection, and no-regex approach:
```go
func ValidateAlias(raw string) string {
    trimmed := strings.TrimSpace(raw)
    if trimmed == "" { return "" }
    runes := []rune(trimmed)
    if len(runes) > 32 { return "" }
    for _, r := range runes {
        if r < 0x0020 || (r >= 0x007F && r <= 0x009F) {
            return "" // C0 or C1 control character
        }
    }
    return trimmed
}
```

---

### `internal/relay/sanitize.go` (create — new pure utility)

**Analog:** `internal/relay/protocol.go` `ValidateAlias` (lines 157–172)

**Package declaration** — same package as protocol.go:
```go
package relay

import "strings"
```

**Core pattern** — mirrors ValidateAlias's `[]rune(input)` range loop, but adds a `state` int for multi-character CSI/OSC escape sequence detection. Key differences from ValidateAlias: (1) does not return "" on bad input — strips and continues; (2) adds `stateEscape`/`stateCSI`/`stateOSC`/`stateOSCEscape` states; (3) calls `strings.TrimRight` on the result before appending exactly one `\n`.

**Output invariant** (load-bearing — must be stated in godoc): only printable text + exactly one trailing `\n` ever exits this function. This is the sole gate before `Hub.WriteInput`.

**Bidi check helper** — standalone `isBidiOverride(r rune) bool` with explicit codepoint switch; see RESEARCH.md Pattern 2 for the full codepoint list (U+061C, U+200E, U+200F, U+202A–U+202E, U+2066–U+2069).

---

### `internal/relay/sanitize_test.go` (create — corpus test)

**Analog:** `internal/relay/server_identity_test.go` table-driven style

**Package declaration:**
```go
package relay
```

**Table structure** — copy the `cases := []struct{ name, input, want string }` pattern; each case asserts `SanitizePTYText(input) == want`. Required corpus categories (per D-03 / SEC-02):
- Plain text passthrough
- LF, CR, CRLF → single space (newline collapse)
- Null byte stripped
- C0 BEL stripped
- DEL (0x7F) stripped
- C1 NEL (U+0085) stripped
- CSI sequences stripped (color, cursor movement)
- OSC sequences stripped (BEL-terminated and ST-terminated)
- Bidi override stripped (RLO U+202E, LRM U+200E)
- Empty input → `"\n"`
- Only-newlines input → `"\n"` (spaces trimmed)
- Mixed attack vector → only safe text + `\n`

**Test runner pattern** (copy from identity test):
```go
for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
        got := SanitizePTYText(tc.input)
        if got != tc.want {
            t.Errorf("SanitizePTYText(%q) = %q, want %q", tc.input, got, tc.want)
        }
    })
}
```

---

### `internal/relay/hub.go` (modify — add chatAppendFn field + SetChatAppendFn + BroadcastChat + HandleInject)

**Analog:** existing Hub struct + `BroadcastMeta` (lines 234–245) + `ResizeClient` (lines 163–193) + `Subscriber.AliasSetFn` (line 33)

**New field** — add to Hub struct after existing `presenceRoster`/`typingRoster` block (~line 68):
```go
// Phase 153: persist+broadcast callback. Wired by engine.go after Hub+ChatStore
// are both constructed. Nil-safe: HandleInject skips persist+broadcast when nil.
chatAppendFn func(ChatMessage) (ChatMessage, error)
```

**SetChatAppendFn** — mirrors `Subscriber.AliasSetFn` assignment pattern; acquires `h.mu`:
```go
func (h *Hub) SetChatAppendFn(fn func(ChatMessage) (ChatMessage, error)) {
    h.mu.Lock()
    h.chatAppendFn = fn
    h.mu.Unlock()
}
```

**BroadcastChat** — copy exactly from `BroadcastMeta` (lines 234–245); change frame[0] source only:
```go
// BroadcastMeta (lines 234-245) — exact pattern to copy for BroadcastChat:
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

**HandleInject** — CRITICAL discipline: do NOT hold `h.mu` during `WriteInput` or `chatAppendFn` call (mirrors ResizeClient pattern at lines 163–193 which calls `h.resizeFn` AFTER `h.mu.Unlock()`). `sub.ReadOnly` is read without a lock (set once at subscribe time, never mutated). Return `ErrReadOnly` sentinel (new package-level error) on RO.

```go
// ResizeClient unlock-before-IO discipline (lines 189-193) — HandleInject mirrors this:
h.mu.Unlock() // release BEFORE the blocking PTY resize syscall
if needResize {
    return h.resizeFn(maxCols, maxRows)
}
```

---

### `internal/relay/server.go` (modify — add `case MsgSessionInject:` to read-pump switch)

**Analog:** `case MsgInput:` block (lines 324–329)

**Existing RW-gated MsgInput case** (lines 324–330):
```go
case MsgInput:
    if !sub.ReadOnly { // MC-03: discard input for read-only clients
        filtered := absorber.Filter(payload)
        if len(filtered) > 0 {
            _ = hub.WriteInput(filtered)
        }
    }
```

**New inject case** — insert after `case MsgAliasSet:` block (~line 363), before the closing `}` of the switch:
```go
case MsgSessionInject:
    // SEC-01: RW cap required. Gate is server-side; must hold against a
    // hand-crafted WS frame regardless of any client-side suppression (D-04).
    // MsgChatSend (0x31): chat message only, NEVER writes to PTY (D-02).
    var ip InjectPayload
    if json.Unmarshal(payload, &ip) != nil || ip.Text == "" {
        continue // malformed frame: ignore silently (same as MsgTyping/MsgAliasSet)
    }
    if err := hub.HandleInject(sub, ip.Text); err != nil {
        select {
        case sub.Msgs <- MakeInjectErrorFrame(err.Error()):
        default:
            go sub.CloseSlow()
        }
    }
```

**Import note:** `encoding/json` is already imported in `server.go` (used by `MsgTyping`/`MsgAliasSet` cases).

---

### `internal/webserver/server.go` (modify — mirror inject case in `handleWSSRelay` read-pump)

**Analog:** `case relay.MsgInput:` block (lines 1116–1122) and `case relay.MsgAliasSet:` (lines 1139–1155)

**Existing web-path RW gate** (lines 1007–1008):
```go
claims, _ := capability.ClaimsFromContext(r.Context())
readonly := claims.Perms == "read"
```

**Existing MsgInput case** (lines 1116–1122):
```go
case relay.MsgInput:
    if !sub.ReadOnly { // MC-03: discard input for read-only clients
        filtered := absorber.Filter(payload)
        if len(filtered) > 0 {
            _ = hub.WriteInput(filtered)
        }
    }
```

**New inject case** — add after `case relay.MsgAliasSet:` block (~line 1155), identical structure to relay path but with `relay.` prefix on all symbols:
```go
case relay.MsgSessionInject:
    // SEC-01: RW cap required. Gate is server-side; sourced from signed JWT
    // claims.Perms == "read" (line 1008) — cannot be bypassed via URL param.
    // MsgChatSend (0x31): chat message only, NEVER writes to PTY (D-02).
    var ip relay.InjectPayload
    if json.Unmarshal(payload, &ip) != nil || ip.Text == "" {
        continue
    }
    if err := hub.HandleInject(sub, ip.Text); err != nil {
        select {
        case sub.Msgs <- relay.MakeInjectErrorFrame(err.Error()):
        default:
            go sub.CloseSlow()
        }
    }
```

---

### `internal/daemon/engine.go` (modify — wire `hub.SetChatAppendFn` in `CreateSession`)

**Analog:** existing ChatStore wiring (lines 431–453) and `hub := e.manager.Create(id, ...)` (line 413)

**Existing wiring sequence** (lines 413, 442–453):
```go
hub := e.manager.Create(id, sess, sess, resizeFn)   // line 413
// ...
chatStore, chatStoreErr := NewChatStore(chatsBaseDir, id)  // line 442

e.mu.Lock()
// ...
if chatStoreErr != nil {
    log.Printf("chat: NewChatStore ...")
} else {
    e.chatStores[id] = chatStore   // line 452
}
e.mu.Unlock()
```

**New wiring** — add immediately after the `e.mu.Unlock()` block (~line 453), before the `status.Watch` goroutine:
```go
// Phase 153: wire the inject persist callback so the relay read pump can
// append SessionInject messages without importing daemon (import-cycle break).
// Nil-guard: chatStore is nil when NewChatStore failed (non-fatal).
if chatStore != nil {
    hub.SetChatAppendFn(func(msg relay.ChatMessage) (relay.ChatMessage, error) {
        return chatStore.AppendMessage(msg)
    })
}
```

**Import:** `internal/relay` is already imported in `engine.go` (`relay.NewHubManager()` at line 321).

---

### `internal/relay/server_inject_test.go` (create — relay-path inject tests)

**Analog:** `internal/relay/server_identity_test.go` — reuse `setupIdentityTestServer`, `dialIdentityWS` helpers verbatim

**Package declaration:**
```go
package relay
```

**Imports** (copy from `server_identity_test.go` lines 1–14):
```go
import (
    "context"
    "encoding/json"
    "io"
    "net/http/httptest"
    "strings"
    "sync/atomic"
    "testing"
    "time"

    "github.com/coder/websocket"
)
```

**Test helper reuse** — `setupIdentityTestServer` returns `(*httptest.Server, *HubManager, string)` and registers all t.Cleanup; `dialIdentityWS` accepts optional `query` string (pass `"readonly=1"` for RO client, `""` for RW client).

**PTY write counting pattern** (for SEC-01 proof):
```go
var ptyWriteCount atomic.Int32
countingWriter := writerFunc(func(p []byte) (int, error) {
    ptyWriteCount.Add(1)
    return len(p), nil
})
// ... create hub with countingWriter instead of io.Discard
// ... assert ptyWriteCount.Load() == 0 after RO inject attempt
```

Note: `writerFunc` is a `type writerFunc func([]byte) (int, error)` with `Write` method — check if already defined in the test package; if not, add locally.

**Required test functions:**
- `TestInject_ROCap_RelayPath` — RO client sends `MsgSessionInject`, expects `MsgInjectError` NAK + zero PTY writes
- `TestInject_RWCap_WritesToPTY` — RW client sends `MsgSessionInject`, expects non-zero PTY writes + `MsgChat` broadcast
- `TestInject_OnlyDedicatedFrame` — RW client sends `MsgChatSend` (0x31), verifies PTY write count stays zero

**waitFor helper** — copy `waitForPresenceFrame` pattern from `server_identity_test.go` lines 81–101 to make a `waitForFrameType(t, conn, msgType, label)` variant for inject tests.

---

### `internal/webserver/inject_test.go` (create — web-path RO rejection adversarial test)

**Analog:** `internal/relay/server_identity_test.go` — same test server setup style; uses `httptest.NewServer` + real JWT-signed capability

**Package declaration:**
```go
package webserver
```

**Key difference from relay test:** The web path's `readonly` comes from `claims.Perms == "read"` (line 1008), not from `?readonly=1`. The test must construct a real capability JWT with `Perms:"read"` and pass it as the `?cap=` query param (the same mechanism as existing webserver tests).

**Pattern for capability setup** — look at existing webserver test files for how they mint a `capability.Claims{Perms: "read"}` token and present it; the `requireCapability` middleware validates it before `handleWSSRelay` runs.

**Required test function:**
- `TestInjectRO_WebPath` — mint RO JWT, dial `handleWSSRelay`, send hand-crafted `MsgSessionInject` frame, assert `MsgInjectError` received + zero PTY writes. This is the adversarial proof that SEC-01 holds for the web entry path regardless of client-side suppression.

---

## Shared Patterns

### RW-cap gate (SEC-01)
**Source:** `internal/relay/server.go` lines 324–329 (`case MsgInput:` block)
**Apply to:** Both read-pump switch additions (`relay/server.go` and `webserver/server.go`)
```go
// Pattern: gate on sub.ReadOnly, which is set at subscribe time from either:
//   relay path: ?readonly=1 query param (server.go line 209)
//   web path:   claims.Perms == "read" from signed JWT (webserver/server.go line 1008)
if !sub.ReadOnly {
    // privileged action
}
// For inject: return ErrReadOnly via HandleInject → send NAK frame (not silent drop)
```

### Make*Frame builder convention
**Source:** `internal/relay/protocol.go` lines 118–142
**Apply to:** New `MakeChatFrame` and `MakeInjectErrorFrame` builders in `protocol.go`
```go
func MakeXxxFrame(p XxxPayload) []byte {
    b, _ := json.Marshal(p)   // payload always serialisable
    frame := make([]byte, 1+len(b))
    frame[0] = MsgXxx
    copy(frame[1:], b)
    return frame
}
```

### Callback field pattern (import-cycle break)
**Source:** `internal/relay/hub.go` line 33 (`Subscriber.AliasSetFn`) + engine.go lines 442–453 (ChatStore wiring)
**Apply to:** `Hub.chatAppendFn` + `hub.SetChatAppendFn` + engine.go wiring
- Field is a func type on the struct
- Set via a public setter that acquires `h.mu`
- Nil-checked before every call (`if h.chatAppendFn != nil`)
- Wired in `engine.go` `CreateSession` after both Hub and ChatStore are constructed

### Unlock-before-IO discipline
**Source:** `internal/relay/hub.go` `ResizeClient` lines 163–193 and `UpdateTyping` line 391
**Apply to:** `Hub.HandleInject`
```go
// ResizeClient pattern — call resizeFn AFTER releasing mu:
h.mu.Unlock() // release BEFORE blocking I/O
if needResize {
    return h.resizeFn(maxCols, maxRows)
}
// HandleInject: read sub.ReadOnly (no lock needed — set once), then call
// WriteInput + chatAppendFn + BroadcastChat all without holding h.mu.
```

### Non-blocking subscriber send
**Source:** `internal/relay/hub.go` `broadcast` (line 223) and `BroadcastMeta` (line 234)
**Apply to:** NAK frame send in read-pump inject case + `BroadcastChat`
```go
select {
case sub.Msgs <- frame:
default:
    go sub.CloseSlow()
}
```

### Malformed-frame silent-drop
**Source:** `internal/relay/server.go` lines 343–346 (`case MsgTyping:`)
**Apply to:** JSON unmarshal failure in `case MsgSessionInject:` in both read pumps
```go
var tp TypingPayload
if json.Unmarshal(payload, &tp) == nil {
    // process
}
// No else — malformed frames are silently ignored
// BUT: only RO rejection sends explicit NAK; malformed frames do NOT.
```

---

## No Analog Found

All files in this phase have strong analogs in the existing codebase. No files require falling back to RESEARCH.md patterns exclusively.

---

## Metadata

**Analog search scope:** `internal/relay/`, `internal/webserver/`, `internal/daemon/`
**Files scanned:** 8 source files read directly
**Pattern extraction date:** 2026-06-26
**Verification:** All line numbers confirmed against live code (per RESEARCH.md verified anchors)
