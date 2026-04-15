# Phase 74: Multi-Client Fan-Out - Pattern Map

**Mapped:** 2026-04-14
**Files analyzed:** 7 files to be created or modified
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/relay/hub.go` | service (hub core) | event-driven, fan-out | `internal/relay/hub.go` itself | self — additive extension |
| `internal/relay/server.go` | middleware / handler | request-response, event-driven | `internal/relay/server.go` itself | self — additive extension |
| `internal/webserver/server.go` | middleware / handler | request-response, event-driven | `internal/relay/server.go` | exact role+data-flow match (duplicate read pump) |
| `internal/daemon/types.go` | model | transform | `internal/daemon/types.go` itself | self — additive extension |
| `internal/daemon/engine.go` | service | CRUD | `internal/daemon/engine.go` itself | self — additive extension |
| `cmd_attach.go` | CLI handler | request-response | `cmd_attach.go` itself | self — additive extension |
| `internal/relay/hub_test.go` | test | — | `internal/relay/hub_test.go` itself | self — new test functions appended |

---

## Pattern Assignments

### `internal/relay/hub.go` (service, event-driven)

**Analog:** `internal/relay/hub.go` (self-extension)

**Existing Subscriber struct** (lines 9–15) — extend this, do not replace:
```go
type Subscriber struct {
    Msgs      chan []byte
    CloseSlow func()
}
```

**Target shape after extension** (add three fields to `Subscriber`):
```go
type Subscriber struct {
    Msgs      chan []byte
    CloseSlow func()

    // MC-03: if true, input frames from this client are discarded by the server read pump.
    ReadOnly bool

    // MC-05: optional client identity name from ?client= query param.
    Name string

    // MC-06: last reported terminal dimensions from this client.
    // Read/written under hub.mu.
    Cols int
    Rows int
}
```

**Existing Hub struct fields** (lines 19–31) — add `ptyCols`/`ptyRows` after `closed`:
```go
type Hub struct {
    sessionID  string
    reader     io.Reader
    writer     io.Writer
    scrollback *Scrollback
    resizeFn   func(cols, rows int) error

    mu          sync.Mutex
    subscribers map[*Subscriber]struct{}
    done        chan struct{}
    closed      bool
    closeOnce   sync.Once
    // MC-06: current PTY dimensions (max-wins arbiter tracks these).
    ptyCols int
    ptyRows int
}
```

**Existing `Resize` method** (lines 50–55) — keep as-is for backward compat; add `ResizeClient` alongside:
```go
// Resize calls the resize callback registered at construction time.
// Kept for backward compatibility. New callers should use ResizeClient.
func (h *Hub) Resize(cols, rows int) error {
    if h.resizeFn != nil {
        return h.resizeFn(cols, rows)
    }
    return nil
}
```

**New `SubscriberCount` method** — follows the same mu-lock pattern as `Subscribe` (line 60):
```go
// SubscriberCount returns the number of currently subscribed clients. (MC-04)
func (h *Hub) SubscriberCount() int {
    h.mu.Lock()
    defer h.mu.Unlock()
    return len(h.subscribers)
}
```

**New `ResizeClient` method** — max-wins arbiter; note unlock-before-resizeFn anti-pattern avoidance:
```go
// ResizeClient stores the subscriber's reported dimensions and calls resizeFn
// only when the new maximum across all subscribers exceeds the current PTY size.
// This implements the max-wins policy: PTY only grows, never shrinks. (MC-06)
//
// IMPORTANT: resizeFn is called AFTER releasing hub.mu to avoid contending
// the broadcast drain loop with a potentially blocking PTY syscall.
func (h *Hub) ResizeClient(sub *Subscriber, cols, rows int) error {
    h.mu.Lock()
    sub.Cols = cols
    sub.Rows = rows

    maxCols, maxRows := 0, 0
    for s := range h.subscribers {
        if s.Cols > maxCols {
            maxCols = s.Cols
        }
        if s.Rows > maxRows {
            maxRows = s.Rows
        }
    }
    needResize := (maxCols > 0 || maxRows > 0) && (maxCols != h.ptyCols || maxRows != h.ptyRows)
    if needResize {
        h.ptyCols = maxCols
        h.ptyRows = maxRows
    }
    h.mu.Unlock() // release BEFORE calling resizeFn

    if needResize && h.resizeFn != nil {
        return h.resizeFn(maxCols, maxRows)
    }
    return nil
}
```

**Lock pattern to follow** — copy from `Subscribe` (lines 59–63):
```go
func (h *Hub) Subscribe(sub *Subscriber) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.subscribers[sub] = struct{}{}
}
```

---

### `internal/relay/server.go` (handler, request-response)

**Analog:** `internal/relay/server.go` (self-extension)

**Existing `handleSession` handler** (lines 43–128) — the entry point to modify for query param parsing. The `sub` construction block at lines 65–70 is where new fields are set:
```go
sub := &Subscriber{
    Msgs: make(chan []byte, 256),
}
sub.CloseSlow = func() {
    conn.Close(websocket.StatusPolicyViolation, "too slow")
}
```

**Target shape** — parse query params before constructing `sub`:
```go
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")

    // MC-03, MC-05: parse client metadata from URL query params at upgrade time.
    readonly := r.URL.Query().Get("readonly") == "1" || r.URL.Query().Get("readonly") == "true"
    clientName := r.URL.Query().Get("client")
    if len(clientName) > 64 {
        clientName = clientName[:64] // MC-05: cap at 64 chars (security: input validation)
    }

    hub, ok := s.manager.Get(sessionID)
    // ... (unchanged)

    sub := &Subscriber{
        Msgs:     make(chan []byte, 256),
        ReadOnly: readonly,
        Name:     clientName,
    }
    sub.CloseSlow = func() {
        conn.Close(websocket.StatusPolicyViolation, "too slow")
    }
    // ...
```

**Existing read pump switch** (lines 98–110) — the `MsgInput` and `MsgResize2` cases to modify:
```go
switch msgType {
case MsgInput:
    _ = hub.WriteInput(payload)
case MsgResize2:
    if len(payload) >= 4 {
        cols := uint16(payload[0])<<8 | uint16(payload[1])
        rows := uint16(payload[2])<<8 | uint16(payload[3])
        _ = hub.Resize(int(cols), int(rows))
    }
case MsgPing:
    // Keep-alive — no-op.
}
```

**Target shape** — gate on `ReadOnly`, use `ResizeClient`:
```go
switch msgType {
case MsgInput:
    if !sub.ReadOnly { // MC-03: discard input for read-only clients
        _ = hub.WriteInput(payload)
    }
case MsgResize2:
    if len(payload) >= 4 {
        cols := uint16(payload[0])<<8 | uint16(payload[1])
        rows := uint16(payload[2])<<8 | uint16(payload[3])
        _ = hub.ResizeClient(sub, int(cols), int(rows)) // MC-06: max-wins arbiter
    }
case MsgPing:
    // Keep-alive — no-op.
}
```

---

### `internal/webserver/server.go` (handler, request-response)

**Analog:** `internal/relay/server.go` — exact structural match (duplicate read pump)

**Existing `handleWSSRelay`** (lines 382–466) — identical structure to relay/server.go. The same two changes apply in the same two locations.

**Existing `sub` construction block** (lines 404–409):
```go
sub := &relay.Subscriber{
    Msgs: make(chan []byte, 256),
}
sub.CloseSlow = func() {
    conn.Close(websocket.StatusPolicyViolation, "too slow")
}
```

**Target shape** — identical to relay/server.go pattern above, using `relay.` prefix on types:
```go
readonly := r.URL.Query().Get("readonly") == "1" || r.URL.Query().Get("readonly") == "true"
clientName := r.URL.Query().Get("client")
if len(clientName) > 64 {
    clientName = clientName[:64]
}

sub := &relay.Subscriber{
    Msgs:     make(chan []byte, 256),
    ReadOnly: readonly,
    Name:     clientName,
}
sub.CloseSlow = func() {
    conn.Close(websocket.StatusPolicyViolation, "too slow")
}
```

**Existing read pump switch** (lines 436–447):
```go
switch msgType {
case relay.MsgInput:
    _ = hub.WriteInput(payload)
case relay.MsgResize2:
    if len(payload) >= 4 {
        cols := uint16(payload[0])<<8 | uint16(payload[1])
        rows := uint16(payload[2])<<8 | uint16(payload[3])
        _ = hub.Resize(int(cols), int(rows))
    }
case relay.MsgPing:
    // Keep-alive — no-op.
}
```

**Target shape** — same as relay/server.go:
```go
switch msgType {
case relay.MsgInput:
    if !sub.ReadOnly { // MC-03
        _ = hub.WriteInput(payload)
    }
case relay.MsgResize2:
    if len(payload) >= 4 {
        cols := uint16(payload[0])<<8 | uint16(payload[1])
        rows := uint16(payload[2])<<8 | uint16(payload[3])
        _ = hub.ResizeClient(sub, int(cols), int(rows)) // MC-06
    }
case relay.MsgPing:
    // Keep-alive — no-op.
}
```

---

### `internal/daemon/types.go` (model, transform)

**Analog:** `internal/daemon/types.go` (self-extension)

**Existing `SessionInfo` struct** (lines 4–12):
```go
type SessionInfo struct {
    ID         string `json:"id"`
    CLI        string `json:"cli"`
    Name       string `json:"name"`
    State      string `json:"state"`
    CreatedAt  string `json:"createdAt"`
    Hostname   string `json:"hostname"`
    WebEnabled bool   `json:"webEnabled"`
}
```

**Target shape** — append `ViewerCount` field (MC-04). The `WebEnabled` field was added by a prior phase using the same additive pattern:
```go
type SessionInfo struct {
    ID          string `json:"id"`
    CLI         string `json:"cli"`
    Name        string `json:"name"`
    State       string `json:"state"`
    CreatedAt   string `json:"createdAt"`
    Hostname    string `json:"hostname"`
    WebEnabled  bool   `json:"webEnabled"`
    ViewerCount int    `json:"viewerCount"` // MC-04: number of active WebSocket subscribers
}
```

---

### `internal/daemon/engine.go` (service, CRUD)

**Analog:** `internal/daemon/engine.go` (self-extension)

**Existing `ListSessions` method** (lines 134–157) — the loop body is where `ViewerCount` is populated. Follow the same pattern used for `WebEnabled` enrichment in `api.go` (lines 241–248), but done at the engine layer using `manager.Get`:
```go
func (e *SessionEngine) ListSessions() []SessionInfo {
    sessions := e.registry.List()
    result := make([]SessionInfo, 0, len(sessions))

    e.mu.RLock()
    defer e.mu.RUnlock()

    for _, s := range sessions {
        state := "running"
        if s.State == pty.StateStopped {
            state = "stopped"
        }
        name := e.tabNames[s.ID]
        result = append(result, SessionInfo{
            ID:        s.ID,
            CLI:       s.CLI,
            Name:      name,
            State:     state,
            CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
            Hostname:  e.hostname,
        })
    }
    return result
}
```

**Target shape** — add `ViewerCount` to the `SessionInfo` literal using `manager.Get`:
```go
for _, s := range sessions {
    state := "running"
    if s.State == pty.StateStopped {
        state = "stopped"
    }
    name := e.tabNames[s.ID]

    // MC-04: populate viewer count from hub subscriber count.
    // manager.Get acquires HubManager.mu; SubscriberCount acquires hub.mu.
    // Both are fine to call while holding e.mu.RLock (no lock ordering conflict).
    viewerCount := 0
    if hub, ok := e.manager.Get(s.ID); ok {
        viewerCount = hub.SubscriberCount()
    }

    result = append(result, SessionInfo{
        ID:          s.ID,
        CLI:         s.CLI,
        Name:        name,
        State:       state,
        CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
        Hostname:    e.hostname,
        ViewerCount: viewerCount,
    })
}
```

**Existing manager access pattern** — `e.manager.Get` is already used in `CreateSession` (line 114):
```go
hub := e.manager.Create(id, sess, sess, resizeFn)
```
`Get` follows the same pattern:
```go
if hub, ok := e.manager.Get(s.ID); ok {
    viewerCount = hub.SubscriberCount()
}
```

---

### `cmd_attach.go` (CLI handler, request-response)

**Analog:** `cmd_attach.go` (self-extension)

**Existing flag parsing pattern** (lines 31–43) — `--detach-key` is parsed from `args[1:]` with prefix matching. Follow this same pattern for `--readonly` and `--client`:
```go
detachKey := byte(0x1C)
for _, arg := range args[1:] {
    if len(arg) > 13 && arg[:13] == "--detach-key=" {
        val := arg[13:]
        switch val {
        case `ctrl-\`, "ctrl-backslash":
            detachKey = 0x1C
        default:
            if len(val) == 1 {
                detachKey = val[0]
            }
        }
    }
}
```

**Existing `wsURL` construction** (line 79):
```go
wsURL := fmt.Sprintf("ws://127.0.0.1:%d/sessions/%s/ws", port, sessionID)
```

**Target shape** — parse `--readonly` and `--client` flags, then build URL with query params using `net/url`:
```go
// Parse optional flags from args[1:].
detachKey := byte(0x1C)
readOnly := false
clientName := ""
for _, arg := range args[1:] {
    if arg == "--readonly" {
        readOnly = true
    } else if len(arg) > 9 && arg[:9] == "--client=" {
        clientName = arg[9:]
    } else if len(arg) > 13 && arg[:13] == "--detach-key=" {
        // ... existing detach key parsing unchanged ...
    }
}

// Build wsURL with query params (MC-03, MC-05).
u := url.URL{
    Scheme: "ws",
    Host:   fmt.Sprintf("127.0.0.1:%d", port),
    Path:   fmt.Sprintf("/sessions/%s/ws", sessionID),
}
q := url.Values{}
if readOnly {
    q.Set("readonly", "1")
}
if clientName != "" {
    q.Set("client", clientName)
}
if len(q) > 0 {
    u.RawQuery = q.Encode()
}
wsURL := u.String()
```

**Required new import** — `"net/url"` must be added to the existing import block (lines 3–19). The existing imports already include `"fmt"`, which is still needed.

---

### `internal/relay/hub_test.go` (test)

**Analog:** `internal/relay/hub_test.go` (self-extension — append new test functions)

**Existing `makeTestHub` helper** (lines 11–17) — reuse as-is for all new tests:
```go
func makeTestHub(t *testing.T) (*Hub, *io.PipeWriter) {
    t.Helper()
    r, w := io.Pipe()
    hub := NewHub("test-session", r, w, DefaultScrollbackBytes, nil)
    return hub, w
}
```

**Existing test structure pattern** (e.g., `TestHubTwoSubscribersBothReceive` lines 54–88):
- Use `makeTestHub` to get a hub + ptyWriter
- Construct `&Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}`
- Call `hub.Subscribe(sub)` then `go hub.Run()`
- Write to `ptyWriter` to simulate PTY output
- Use `select { case ...: ... case <-time.After(2 * time.Second): t.Fatal("timeout") }` for async assertions

**New tests to append** (MC-03, MC-04, MC-05, MC-06):

MC-03 test skeleton — read-only input discarded:
```go
func TestHub_ReadOnlyClientInputDiscarded(t *testing.T) {
    // Setup: two subscribers, one ReadOnly=true, verify WriteInput is NOT called
    // for the read-only one (test at the server layer via server_test.go, or
    // verify the ReadOnly flag is stored on Subscriber and server gates on it).
    // Hub layer test: just verify Subscriber.ReadOnly is stored correctly.
    hub, ptyWriter := makeTestHub(t)
    defer ptyWriter.Close()

    sub := &Subscriber{
        Msgs:      make(chan []byte, 256),
        CloseSlow: func() {},
        ReadOnly:  true,
    }
    hub.Subscribe(sub)
    if !sub.ReadOnly {
        t.Error("ReadOnly field not stored on Subscriber")
    }
}
```

MC-04 test skeleton — SubscriberCount:
```go
func TestHub_SubscriberCountTracksConcurrentSubscribers(t *testing.T) {
    hub, ptyWriter := makeTestHub(t)
    defer ptyWriter.Close()

    if hub.SubscriberCount() != 0 {
        t.Errorf("initial count: want 0, got %d", hub.SubscriberCount())
    }
    sub1 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
    hub.Subscribe(sub1)
    if hub.SubscriberCount() != 1 {
        t.Errorf("after subscribe: want 1, got %d", hub.SubscriberCount())
    }
    hub.Unsubscribe(sub1)
    if hub.SubscriberCount() != 0 {
        t.Errorf("after unsubscribe: want 0, got %d", hub.SubscriberCount())
    }
}
```

MC-05 test skeleton — client name stored:
```go
func TestHub_ClientNameStoredOnSubscriber(t *testing.T) {
    hub, ptyWriter := makeTestHub(t)
    defer ptyWriter.Close()

    sub := &Subscriber{
        Msgs:      make(chan []byte, 256),
        CloseSlow: func() {},
        Name:      "macbook",
    }
    hub.Subscribe(sub)
    if sub.Name != "macbook" {
        t.Errorf("Name: want %q, got %q", "macbook", sub.Name)
    }
}
```

MC-06 test skeleton — max-wins resize:
```go
func TestHub_ResizeMaxWinsPolicy(t *testing.T) {
    r, w := io.Pipe()
    resizeCalls := [][]int{}
    hub := NewHub("test-resize", r, w, DefaultScrollbackBytes, func(cols, rows int) error {
        resizeCalls = append(resizeCalls, []int{cols, rows})
        return nil
    })
    go hub.Run()
    defer w.Close()

    sub1 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
    sub2 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
    hub.Subscribe(sub1)
    hub.Subscribe(sub2)

    // sub1 claims 220x50; should trigger resize to 220x50
    _ = hub.ResizeClient(sub1, 220, 50)
    // sub2 claims 80x24; max is still 220x50, no new resize call expected
    _ = hub.ResizeClient(sub2, 80, 24)
    // sub1 claims 240x60; should trigger resize to 240x60
    _ = hub.ResizeClient(sub1, 240, 60)

    if len(resizeCalls) != 2 {
        t.Errorf("want 2 resize calls, got %d: %v", len(resizeCalls), resizeCalls)
    }
    if len(resizeCalls) >= 1 && (resizeCalls[0][0] != 220 || resizeCalls[0][1] != 50) {
        t.Errorf("first resize: want 220x50, got %dx%d", resizeCalls[0][0], resizeCalls[0][1])
    }
    if len(resizeCalls) >= 2 && (resizeCalls[1][0] != 240 || resizeCalls[1][1] != 60) {
        t.Errorf("second resize: want 240x60, got %dx%d", resizeCalls[1][0], resizeCalls[1][1])
    }
}
```

---

## Shared Patterns

### Mutex Lock Pattern
**Source:** `internal/relay/hub.go` lines 59–63 (`Subscribe`) and lines 66–70 (`Unsubscribe`)
**Apply to:** All new Hub methods (`SubscriberCount`, `ResizeClient`)
```go
h.mu.Lock()
defer h.mu.Unlock()
// ... operate on h.subscribers map ...
```
Exception for `ResizeClient`: defer is NOT used because `resizeFn` must be called after unlock. Use explicit `h.mu.Lock()` / `h.mu.Unlock()` with the resize call after.

### Query Parameter Parsing Pattern
**Source:** `internal/relay/server.go` — `r.PathValue("id")` (line 44); extend with `r.URL.Query().Get()`
**Apply to:** `internal/relay/server.go` `handleSession`, `internal/webserver/server.go` `handleWSSRelay`
```go
readonly := r.URL.Query().Get("readonly") == "1" || r.URL.Query().Get("readonly") == "true"
clientName := r.URL.Query().Get("client")
if len(clientName) > 64 {
    clientName = clientName[:64] // strip control-char injection surface
}
```

### WebSocket Subscriber Construction Pattern
**Source:** `internal/relay/server.go` lines 65–70, `internal/webserver/server.go` lines 404–409
**Apply to:** Both server read pump entry points
```go
sub := &Subscriber{ /* or &relay.Subscriber{ in webserver */ 
    Msgs:     make(chan []byte, 256),
    ReadOnly: readonly,
    Name:     clientName,
}
sub.CloseSlow = func() {
    conn.Close(websocket.StatusPolicyViolation, "too slow")
}
hub.Subscribe(sub)
defer hub.Unsubscribe(sub)
```

### JSON Response Pattern
**Source:** `internal/daemon/api.go` lines 204–209 (`writeJSON` helper)
**Apply to:** Any new API response that returns `ViewerCount` — already handled since `handleListSessions` calls `engine.ListSessions()` which returns the enriched `SessionInfo`
```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}
```

### Test Hub Setup Pattern
**Source:** `internal/relay/hub_test.go` lines 11–17 (`makeTestHub`)
**Apply to:** All new tests in `hub_test.go` and `server_test.go`
```go
func makeTestHub(t *testing.T) (*Hub, *io.PipeWriter) {
    t.Helper()
    r, w := io.Pipe()
    hub := NewHub("test-session", r, w, DefaultScrollbackBytes, nil)
    return hub, w
}
```

---

## No Analog Found

None — every file to be modified has a direct analog (itself or a structurally identical peer).

---

## Critical Anti-Patterns (Do Not Repeat)

| Anti-Pattern | Where It Would Occur | Correct Alternative |
|---|---|---|
| Call `resizeFn` while holding `hub.mu` | `Hub.ResizeClient` | Capture dimensions under lock, unlock, then call `resizeFn` |
| Unconditional `hub.Resize` in read pump | `relay/server.go` and `webserver/server.go` | Replace with `hub.ResizeClient(sub, ...)` |
| Call `hub.Resize` without updating `sub.Cols`/`sub.Rows` | Any new resize call site | Always use `hub.ResizeClient(sub, ...)` so dimensions are tracked |
| Store viewer count in a separate atomic counter | `Hub` struct | Use `len(h.subscribers)` under `hub.mu` — no drift possible |
| Parse client metadata in webserver layer only | `webserver/server.go` | Set fields on `Subscriber` struct (hub-owned) so all callers see them |

---

## Metadata

**Analog search scope:** `internal/relay/`, `internal/daemon/`, `internal/webserver/`, `cmd_attach.go`
**Files read:** hub.go, server.go (relay), server.go (webserver), manager.go, types.go, engine.go, api.go, cmd_attach.go, hub_test.go, server_test.go, api_test.go
**Pattern extraction date:** 2026-04-14
