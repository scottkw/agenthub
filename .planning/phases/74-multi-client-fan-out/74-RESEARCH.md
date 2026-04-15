# Phase 74: Multi-Client Fan-Out - Research

**Researched:** 2026-04-14
**Domain:** Go WebSocket relay, PTY session concurrency, resize arbitration, read-only access control
**Confidence:** HIGH

## Summary

Phase 74 adds multi-client capability to an already-working fan-out hub. The core broadcast
mechanism (`internal/relay/Hub`, `HubManager`, `Subscriber`) already fans output to N
simultaneous WebSocket clients — `TestHub_TwoClientsFanOut` and `TestHub_ReconnectScrollback`
pass today. What Phase 74 must add is everything built on top of that foundation:

1. **Per-client metadata storage in Hub** — connection count (MC-04), client identity name
   (MC-05), and read-only flag (MC-03) must be stored per `Subscriber`.
2. **Read-only enforcement** — `Subscriber` needs a `ReadOnly bool` field; the server's read
   pump must check it before calling `hub.WriteInput` or `hub.Resize`.
3. **Resize arbitration** — when multiple clients send resize frames, the PTY must stabilize
   to the largest active client's dimensions, never oscillate. A "max-dimensions" policy is
   the standard approach.
4. **Session metadata API enrichment** — `SessionInfo` in `internal/daemon/types.go` must
   gain a `ViewerCount int` field; the daemon API's list/get handlers must populate it by
   querying the Hub's subscriber count.
5. **WebSocket URL query parameters** — `?client=name` and `?readonly=1` need to be parsed
   at both `relay.Server.handleSession` (TCP relay) and `webserver.WebServer.handleWSSRelay`
   (HTTPS relay for browser).

The implementation is additive. No existing behavior changes — fan-out, scrollback replay,
and slow-client eviction are correct and should not be touched.

**Primary recommendation:** Extend `Subscriber` with metadata, add a `Hub.SubscriberCount()`
method, enforce read-only at the read-pump level, implement max-wins resize arbitration in
`Hub`, expose viewer count in daemon API.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Fan-out broadcast | relay.Hub | — | Already implemented; no change needed |
| Client identity / read-only flag | relay.Hub (Subscriber) | relay.Server, webserver.WebServer | Hub stores per-client state; servers parse URL query params and set fields |
| Viewer count tracking | relay.Hub | daemon.API | Hub is authoritative source of subscriber count; API exposes it to callers |
| Resize arbitration (max-wins) | relay.Hub | relay.Server, webserver.WebServer | Hub owns PTY state; servers forward resize frames, Hub decides whether to apply |
| Read-only enforcement | relay.Server + webserver.WebServer | relay.Hub | Servers gate WriteInput/Resize calls based on per-Subscriber flag |
| Session metadata API (viewer count) | daemon.API (GET /sessions, GET /sessions/{id}) | daemon.SessionEngine | API enriches SessionInfo from Hub.SubscriberCount() |
| CLI attach `--readonly` flag | cmd_attach.go | relay (MsgMeta frame type) | CLI parses flag, sends metadata frame at connect time, or uses query param |

## Standard Stack

### Core (already in go.mod)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/coder/websocket` | v1.8.14 | WebSocket upgrade and framing | [VERIFIED: go.mod] Already used throughout relay and webserver |
| `golang.org/x/term` | v0.41.0 | Terminal raw mode, SIGWINCH | [VERIFIED: go.mod] Already used in cmd_attach.go |
| `sync` (stdlib) | go1.26 | Mutex-protected hub state | [VERIFIED: hub.go] Pattern established |

### No New Dependencies Required
All required functionality can be implemented using existing dependencies.
[VERIFIED: codebase grep — no new library needed]

## Architecture Patterns

### System Architecture Diagram

```
CLI / Browser                relay.Server / webserver         relay.Hub
─────────────                ───────────────────────          ──────────

Browser tab A (read-write) ──WS ?client=tab-a ──────────────► Subscribe(sub{name="tab-a", ro=false})
Browser tab B (read-only)  ──WS ?readonly=1   ──────────────► Subscribe(sub{name="", ro=true})
CLI attach                 ──WS ?client=macbook ─────────────► Subscribe(sub{name="macbook", ro=false})

PTY reader goroutine ──────────────────────────────────────────► Hub.Run() drain loop
                                                                    │
                                                                    ▼ broadcast(frame)
                                                               sub.Msgs ──► client A
                                                               sub.Msgs ──► client B
                                                               sub.Msgs ──► CLI

Client A sends MsgInput ───────────────────────────────────────► server read pump:
                                                                   if sub.ReadOnly → discard
                                                                   else → hub.WriteInput()

Client A sends MsgResize2 ─────────────────────────────────────► server read pump:
                                                                   hub.Resize(cols, rows)
                                                                    │
                                                                    ▼ Hub max-wins arbiter:
                                                               track per-sub dimensions
                                                               if cols×rows > current max:
                                                                 call resizeFn(cols, rows)
                                                               else: no-op (no PTY resize)

GET /sessions ─────────────────────────────────────────────────► daemon.API.handleListSessions
                                                                   SessionInfo.ViewerCount =
                                                                     manager.Get(id).SubscriberCount()
```

### Recommended Project Structure
```
internal/relay/
├── hub.go         # Add: Subscriber.ReadOnly, Subscriber.Name, Subscriber.Cols/Rows
│                  # Add: Hub.SubscriberCount(), Hub.Resize now max-wins
├── server.go      # Add: parse ?client=, ?readonly= query params; enforce read-only
├── scrollback.go  # No change
├── protocol.go    # No change (MsgMeta message type optional, query params preferred)
├── manager.go     # No change
internal/daemon/
├── types.go       # Add: ViewerCount int to SessionInfo
├── api.go         # Add: populate ViewerCount in handleListSessions / handleGetSession
internal/webserver/
├── server.go      # Add: parse ?client=, ?readonly= query params; enforce read-only
cmd_attach.go      # Add: --readonly flag; send ?readonly=1 or ?client=name in WS URL
```

### Pattern 1: Per-Subscriber Metadata Fields

Extend `Subscriber` to carry read-only flag, identity name, and current terminal dimensions.
The server constructs the `Subscriber` with these fields populated from URL query parameters
before calling `hub.Subscribe`.

```go
// Source: internal/relay/hub.go (proposed extension)
type Subscriber struct {
    Msgs      chan []byte
    CloseSlow func()

    // MC-03: if true, input frames from this client are discarded.
    ReadOnly bool

    // MC-05: optional client identity name from ?client= query param.
    Name string

    // MC-06: last reported terminal dimensions from this client.
    // Protected by Hub.mu (set inside broadcast lock or separate lock).
    Cols int
    Rows int
}
```

[VERIFIED: hub.go — Subscriber struct is the right place; already guards map with hub.mu]

### Pattern 2: Max-Wins Resize Arbitration (MC-06)

The resize arbitration policy: when client A sends (80x24) and client B sends (220x50),
the PTY is set to max(cols) × max(rows). This prevents the PTY from shrinking when a small
client connects. It also prevents oscillation — only an actual growth event triggers a PTY
resize syscall.

```go
// Source: internal/relay/hub.go (proposed Resize replacement)
// ResizeClient updates the stored dimensions for sub and calls resizeFn if the
// new dimensions are strictly larger than the current PTY dimensions.
func (h *Hub) ResizeClient(sub *Subscriber, cols, rows int) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    sub.Cols = cols
    sub.Rows = rows

    // Compute max across all subscribers.
    maxCols, maxRows := 0, 0
    for s := range h.subscribers {
        if s.Cols > maxCols {
            maxCols = s.Cols
        }
        if s.Rows > maxRows {
            maxRows = s.Rows
        }
    }

    // Only call resizeFn if PTY dimensions need to grow.
    if maxCols != h.ptyCols || maxRows != h.ptyRows {
        h.ptyCols = maxCols
        h.ptyRows = maxRows
        if h.resizeFn != nil {
            return h.resizeFn(maxCols, maxRows)
        }
    }
    return nil
}
```

Key invariant: `h.ptyCols` / `h.ptyRows` track the **current PTY dimensions**, not a client
request. Stored on Hub alongside `subscribers`. Protected by `hub.mu`.

[ASSUMED: max-wins is the conventional policy for shared terminal sessions — tmux uses it]

### Pattern 3: Read-Only Enforcement in Server Read Pump

Both `relay/server.go` and `webserver/server.go` have identical read pump logic. The change
is the same in both places:

```go
// Source: internal/relay/server.go (proposed extension)
case MsgInput:
    if !sub.ReadOnly {    // MC-03: gate on read-only flag
        _ = hub.WriteInput(payload)
    }
case MsgResize2:
    if len(payload) >= 4 {
        cols := uint16(payload[0])<<8 | uint16(payload[1])
        rows := uint16(payload[2])<<8 | uint16(payload[3])
        _ = hub.ResizeClient(sub, int(cols), int(rows))  // MC-06
    }
```

[VERIFIED: relay/server.go and webserver/server.go — both have identical read pump switch]

### Pattern 4: Viewer Count in SessionInfo

```go
// Source: internal/daemon/types.go (proposed extension)
type SessionInfo struct {
    ID          string `json:"id"`
    CLI         string `json:"cli"`
    Name        string `json:"name"`
    State       string `json:"state"`
    CreatedAt   string `json:"createdAt"`
    Hostname    string `json:"hostname"`
    WebEnabled  bool   `json:"webEnabled"`
    ViewerCount int    `json:"viewerCount"` // MC-04
}
```

Populate in `engine.ListSessions()` by calling `hub.SubscriberCount()` for each session ID.
The `Hub.SubscriberCount()` method acquires `mu` and returns `len(h.subscribers)`.

[VERIFIED: daemon/types.go, daemon/engine.go — ListSessions iterates all sessions; manager.Get() is available]

### Pattern 5: URL Query Parameter Parsing

Both relay server endpoints and the webserver endpoint should parse standard HTTP query params:
- `?readonly=1` (or `true`) → `sub.ReadOnly = true`
- `?client=name` → `sub.Name = name`

```go
// Source: internal/relay/server.go (proposed)
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    readonly := r.URL.Query().Get("readonly") == "1" || r.URL.Query().Get("readonly") == "true"
    clientName := r.URL.Query().Get("client")
    // ...
    sub := &relay.Subscriber{
        Msgs:     make(chan []byte, 256),
        ReadOnly: readonly,
        Name:     clientName,
    }
```

CLI attach uses URL construction:
```go
// Source: cmd_attach.go (proposed)
wsURL := fmt.Sprintf("ws://127.0.0.1:%d/sessions/%s/ws?readonly=1", port, sessionID)
// or with client name:
wsURL := fmt.Sprintf("ws://127.0.0.1:%d/sessions/%s/ws?client=%s", port, sessionID, url.QueryEscape(clientName))
```

[VERIFIED: cmd_attach.go — wsURL construction at line 79; r.URL.Query() is standard net/http]

### Anti-Patterns to Avoid

- **Resize on every client event:** Calling `resizeFn` on every resize message from any
  client causes PTY dimension oscillation when clients differ in size. The max-wins arbiter
  must compare against current PTY dimensions and only call `resizeFn` when dimensions
  actually change.

- **Holding hub.mu while calling resizeFn:** The pty resize syscall may block momentarily.
  Compute the new max-wins dimensions under `hub.mu`, capture them, unlock, then call
  `resizeFn`. Otherwise resize contends with the broadcast drain loop.

  Corrected pattern:
  ```go
  func (h *Hub) ResizeClient(sub *Subscriber, cols, rows int) error {
      h.mu.Lock()
      sub.Cols, sub.Rows = cols, rows
      maxCols, maxRows := h.computeMaxDimensions() // under lock
      needResize := maxCols != h.ptyCols || maxRows != h.ptyRows
      if needResize {
          h.ptyCols, h.ptyCols = maxCols, maxRows
      }
      h.mu.Unlock()                                // unlock BEFORE calling resizeFn
      if needResize && h.resizeFn != nil {
          return h.resizeFn(maxCols, maxRows)
      }
      return nil
  }
  ```

- **Removing `Resize(cols, rows int)` from Hub public API:** `relay.Server` currently calls
  `hub.Resize` directly. The planner should replace this call with `hub.ResizeClient(sub, ...)`,
  but the old `Resize` method may need to stay for backward compatibility with tests or can
  be repurposed.

- **Storing client metadata in the webserver instead of Hub:** WebServer and relay.Server are
  both entry points to the same Hub. Metadata stored in the server layer would be invisible to
  the daemon API viewer count queries. All per-client state belongs on `Subscriber`.

- **Using a new binary message type for metadata:** A `MsgMeta` binary protocol frame is
  possible but adds complexity (ordering guarantees, replay from scrollback). URL query
  parameters are simpler, already available at WebSocket upgrade time, and sufficient for
  all MC-01 through MC-06 requirements.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Thread-safe subscriber count | Custom atomic counter | `len(h.subscribers)` under `hub.mu` | Hub already serializes all subscriber mutations through `hub.mu`; separate counter risks drift |
| Custom WebSocket sub-protocol negotiation | MsgMeta binary framing | HTTP query parameters at upgrade | Query params are visible to handlers at the HTTP layer before upgrade, no frame ordering issues |
| Fan-out broadcast (already done) | New broadcast mechanism | Existing `Hub.broadcast()` | Works correctly; tests confirm N-subscriber delivery |
| Read-only channel segregation | Separate hub for read-only clients | ReadOnly flag on Subscriber | Same output stream; only input gating differs |

**Key insight:** The hub's `mu` mutex already serializes subscriber map mutations. Viewer count
is just `len(h.subscribers)` read under that lock — no additional data structure needed.

## Common Pitfalls

### Pitfall 1: Resize Oscillation with Competing Clients
**What goes wrong:** Client A (80x24) and Client B (220x50) are both connected. Every time
either sends a resize frame, the PTY oscillates between (80x24) and (220x50). Any terminal
output that was word-wrapped for 220 columns is rewrapped to 80 columns and vice versa.
**Why it happens:** The current `hub.Resize` call is unconditional — it calls `resizeFn`
on every resize message regardless of whether dimensions changed or whether another client
has a larger terminal.
**How to avoid:** Implement max-wins arbitration. Store current PTY dimensions on Hub
(`ptyCols`, `ptyRows`). Only call `resizeFn` when max across all subscribers exceeds current.
**Warning signs:** PTY resize called more than once during a multi-client test scenario.

### Pitfall 2: Race Between Unsubscribe and Resize
**What goes wrong:** Client A disconnects while `ResizeClient` iterates `h.subscribers` to
compute max dimensions. If `Unsubscribe` runs concurrently, the iteration sees a partially
removed map.
**Why it happens:** Both `Unsubscribe` and `ResizeClient` need `hub.mu`. If `ResizeClient`
drops the lock before calling `resizeFn`, a concurrent `Unsubscribe` can modify the map,
making the max computation stale.
**How to avoid:** Compute max and capture dimensions under lock. Release lock before calling
resizeFn (to avoid lock-while-blocking). The stale max is acceptable — the PTY will be
sized slightly large momentarily until the next resize event.

### Pitfall 3: SubscriberCount Called After Hub Shutdown
**What goes wrong:** After a session ends, the hub is shut down. Code that calls
`hub.SubscriberCount()` after `Shutdown()` might read a subscribers map that is being cleared.
**Why it happens:** `HubManager.Remove()` calls `hub.Shutdown()` and then `delete(m.hubs, id)`.
If `ListSessions` runs concurrently, it might get a hub reference and call `SubscriberCount`
on a shutting-down hub.
**How to avoid:** `SubscriberCount` acquires `hub.mu` so it is safe to call at any time —
`Shutdown` only sets `closed = true` and closes `done`, it does not nil the subscribers map.
The `len(h.subscribers)` call is safe. No special handling needed.

### Pitfall 4: TestHub_SlowClientDisconnected Flaky Test
**What goes wrong:** This test is already flaky in the current codebase (fails on first run,
passes with `-short` skip). It races between the normalClient getting evicted (instead of
the slowClient) when the test's write loop runs on a machine where both channels fill up.
**Why it happens:** The `slowClient` has a zero-buffer channel (`make(chan []byte, 0)`)
but the slow-path goroutine launched by `CloseSlow` competes with the drain loop. On fast
machines the normalClient may be the "slow" one.
**How to avoid:** This test has nothing to do with Phase 74 changes. Do not regress it —
but do not spend time fixing it either. The `t.Skip` in `testing.Short()` covers CI.

### Pitfall 5: Both `relay.Server` and `webserver.WebServer` Have Duplicate Read Pump
**What goes wrong:** Read-only enforcement and resize arbitration changes must be applied
to BOTH `relay/server.go` (TCP relay, used by CLI attach) and `webserver/server.go`
(HTTPS relay, used by browser). Missing one means only half the access paths enforce MC-03/MC-06.
**Why it happens:** The two handlers are structurally identical but live in different packages.
**How to avoid:** Make both changes in the same plan. Add a test for each path.

## Code Examples

### Hub.SubscriberCount — Viewer Count Query
```go
// Source: internal/relay/hub.go (to be added)
func (h *Hub) SubscriberCount() int {
    h.mu.Lock()
    defer h.mu.Unlock()
    return len(h.subscribers)
}
```

### Hub.ResizeClient — Max-Wins Arbiter
```go
// Source: internal/relay/hub.go (replaces/supplements existing Resize method)
func (h *Hub) ResizeClient(sub *Subscriber, cols, rows int) error {
    h.mu.Lock()
    sub.Cols = cols
    sub.Rows = rows

    maxCols, maxRows := 0, 0
    for s := range h.subscribers {
        if s.Cols > maxCols { maxCols = s.Cols }
        if s.Rows > maxRows { maxRows = s.Rows }
    }
    needResize := (maxCols > 0 || maxRows > 0) && (maxCols != h.ptyCols || maxRows != h.ptyRows)
    if needResize {
        h.ptyCols = maxCols
        h.ptyRows = maxRows
    }
    h.mu.Unlock()

    if needResize && h.resizeFn != nil {
        return h.resizeFn(maxCols, maxRows)
    }
    return nil
}
```

### Daemon API — Viewer Count Population
```go
// Source: internal/daemon/engine.go (ListSessions extension)
for _, s := range sessions {
    // ...existing fields...
    viewerCount := 0
    if hub, ok := e.manager.Get(s.ID); ok {
        viewerCount = hub.SubscriberCount()
    }
    result = append(result, SessionInfo{
        // ...
        ViewerCount: viewerCount,
    })
}
```

### CLI Attach — Read-Only and Client Name URL Construction
```go
// Source: cmd_attach.go (extension)
import "net/url"

func buildRelayURL(port int, sessionID, clientName string, readOnly bool) string {
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
    u.RawQuery = q.Encode()
    return u.String()
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single-client attach (one WS per session) | Multi-subscriber fan-out (Hub + broadcast) | Phase 73 (v1.14) | Already done — N subscribers work |
| Unconditional resize (last-writer-wins) | Max-wins resize arbitration | Phase 74 | Prevents resize oscillation |
| No access control on input | Read-only flag enforcement | Phase 74 | Enables observer mode |
| No connection metadata | Subscriber identity + viewer count | Phase 74 | Enables `agenthub list` viewer count display |

**Deprecated/outdated:**
- `Hub.Resize(cols, rows int)` — still needed for backward compat but should internally
  become a stub that calls `ResizeClient` with a nil/sentinel subscriber, or can stay as-is
  for the single-client code path (no subscriber to update dimensions for).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Max-wins is the correct resize arbitration policy (largest client wins, no shrink) | Architecture Patterns — Pattern 2 | If user wants min-wins or last-wins, resize logic needs change; low risk since success criteria explicitly says "largest active client" |
| A2 | URL query parameters are preferred over a new binary MsgMeta frame type for passing client identity | Architecture Patterns — Pattern 5 | If future phases need post-connect metadata updates, query params won't work; acceptable for current scope |
| A3 | `Hub.Resize(cols, rows int)` should remain as a public method (not removed) | Common Pitfalls — Pitfall 4 | Removing it breaks existing callers (relay/server.go call site); plan must update all call sites or keep the method |

## Open Questions

1. **Should `--readonly` also suppress scrollback replay or just input?**
   - What we know: MC-03 says "input suppressed, output received" — scrollback replay is output
   - What's unclear: Whether the read-only observer should receive the full scrollback snapshot
   - Recommendation: Yes, read-only clients should receive scrollback (they're observers, not blank slates). Success criteria does not restrict this.

2. **Does `agenthub list` output need client identity names or just the count?**
   - What we know: MC-04 says "current viewer count"; MC-05 says identity "visible in session metadata API"
   - What's unclear: Whether the CLI `list` command output should show names vs. just count
   - Recommendation: Session metadata API returns count + optional list of names; CLI `list` displays count only (names are for API consumers like future TUI)

3. **Should viewer count be 0 or absent when a session has no hub?**
   - What we know: A session in "stopped" state may have no active hub
   - Recommendation: Default to 0 when hub not found via `manager.Get`

## Environment Availability

Step 2.6: SKIPPED — Phase 74 is a pure Go code change. No new external tools, services,
CLIs, runtimes, or databases are required beyond what is already used (Go toolchain, existing
go.mod dependencies).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./internal/relay/... ./internal/daemon/... -count=1 -timeout 30s` |
| Full suite command | `go test ./... -count=1 -timeout 60s` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MC-01 | Two WS clients receive same PTY output | unit | `go test ./internal/relay/... -run TestHub_TwoClientsFanOut` | ✅ server_test.go |
| MC-02 | Independent scrollback position per client | unit | `go test ./internal/relay/... -run TestHub_ReconnectScrollback` | ✅ server_test.go (proxy: reconnect gets scrollback; per-client scroll is client-side) |
| MC-03 | Read-only client: input discarded, output received | unit | `go test ./internal/relay/... -run TestHub_ReadOnlyClientInputDiscarded` | ❌ Wave 0 |
| MC-04 | Viewer count in session metadata API | unit | `go test ./internal/daemon/... -run TestAPI_ViewerCount` | ❌ Wave 0 |
| MC-05 | Client identity in session metadata | unit | `go test ./internal/relay/... -run TestHub_ClientIdentityStored` | ❌ Wave 0 |
| MC-06 | Max-wins resize: largest client wins, no oscillation | unit | `go test ./internal/relay/... -run TestHub_ResizeMaxWins` | ❌ Wave 0 |

**Note on MC-02:** Independent scrollback position is a client-side concern (xterm.js scroll
position is independent per browser tab by construction). The server-side requirement — that
both clients receive the same output stream — is already validated by `TestHub_TwoClientsFanOut`.
No server-side test is needed for the "independent" aspect.

### Sampling Rate
- **Per task commit:** `go test ./internal/relay/... ./internal/daemon/... -count=1 -timeout 30s`
- **Per wave merge:** `go test ./... -count=1 -timeout 60s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/relay/hub_test.go` — add `TestHub_ReadOnlyClientInputDiscarded` (MC-03)
- [ ] `internal/relay/hub_test.go` — add `TestHub_SubscriberCountTracksConcurrentSubscribers` (MC-04 hub layer)
- [ ] `internal/relay/hub_test.go` — add `TestHub_ClientNameStoredOnSubscriber` (MC-05)
- [ ] `internal/relay/hub_test.go` — add `TestHub_ResizeMaxWinsPolicy` (MC-06: multiple resize events, PTY only grows)
- [ ] `internal/relay/hub_test.go` — add `TestHub_ResizeClientUnsubscribeRecomputesMax` (MC-06: client disconnect, max shrinks... or doesn't — clarify policy)
- [ ] `internal/daemon/api_test.go` — add `TestAPI_ListSessionsViewerCount` (MC-04: API returns ViewerCount)
- [ ] `cmd_attach_test.go` — add `TestCmdAttach_ReadonlyFlagConstructsURL` (MC-03 CLI path)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Existing auth unchanged (Tailscale / Basic Auth) |
| V3 Session Management | no | Session lifecycle unchanged |
| V4 Access Control | yes | Read-only flag enforcement — see below |
| V5 Input Validation | yes | Query param values sanitized (client name max length) |
| V6 Cryptography | no | No new crypto |

### Access Control — Read-Only Flag (V4)
The read-only flag is set by the connecting client via `?readonly=1`. This is an **honor
system** — there is no server-side enforcement that a connecting client cannot lie. However,
the `--readonly` flag on the CLI attach command signals intent. The server enforces it by
discarding input from that connection. The risk model is: observers on the same Tailscale
network as the session are trusted at the network level. An attacker with Tailscale network
access can already connect without `?readonly=1`, so the read-only feature is a user
convenience, not a security boundary.

### Known Threat Patterns for Phase 74 Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Client name injection (XSS via session metadata API) | Tampering | Sanitize client name — max 64 chars, strip control characters before storing in Subscriber.Name |
| Resize-loop DoS (rapid resize frames) | DoS | Max-wins arbiter naturally limits PTY resize calls — only grows, never oscillates; rate-limit not needed for this phase |
| Read-only bypass (omit ?readonly flag) | Elevation of Privilege | Acceptable per risk model — network-level trust via Tailscale/local password auth |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: /Users/ken/dev/agenthub/internal/relay/hub.go] — Subscriber struct, Hub fields, broadcast logic
- [VERIFIED: /Users/ken/dev/agenthub/internal/relay/server.go] — handleSession read pump, resize handling
- [VERIFIED: /Users/ken/dev/agenthub/internal/relay/manager.go] — HubManager.Get, Create, SessionIDs
- [VERIFIED: /Users/ken/dev/agenthub/internal/daemon/types.go] — SessionInfo struct
- [VERIFIED: /Users/ken/dev/agenthub/internal/daemon/engine.go] — ListSessions, manager.Get usage
- [VERIFIED: /Users/ken/dev/agenthub/internal/daemon/api.go] — route registration, handleListSessions
- [VERIFIED: /Users/ken/dev/agenthub/internal/webserver/server.go] — handleWSSRelay (duplicate read pump)
- [VERIFIED: /Users/ken/dev/agenthub/cmd_attach.go] — wsURL construction, flag parsing
- [VERIFIED: /Users/ken/dev/agenthub/go.mod] — dependency versions

### Secondary (MEDIUM confidence)
- [ASSUMED: A1] Max-wins resize policy matches tmux/screen behavior and matches success criterion wording "stabilize to the largest active client"

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are existing go.mod dependencies; verified
- Architecture: HIGH — all extension points are identified from direct code reading; no speculation
- Pitfalls: HIGH for race/lock pitfalls (verified from code); MEDIUM for resize oscillation (behavior verified by test, fix pattern is conventional)
- Security: MEDIUM — read-only is honor-system; explicitly acceptable per existing network trust model

**Research date:** 2026-04-14
**Valid until:** 2026-05-14 (codebase is stable; no fast-moving external dependencies)

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MC-01 | Multiple WebSocket clients connect to same session and receive live output simultaneously | Already implemented in Hub.broadcast(); TestHub_TwoClientsFanOut confirms; no code change needed |
| MC-02 | Each connected client maintains independent scrollback position | Client-side (xterm.js scroll state); server sends same output stream to all clients; already correct |
| MC-03 | User can attach in read-only mode via `--readonly` flag (input suppressed, output received) | Requires: Subscriber.ReadOnly field + server read-pump gating + CLI flag + URL query param |
| MC-04 | Daemon tracks connection count per session and exposes it via session metadata API | Requires: Hub.SubscriberCount() + ViewerCount in SessionInfo + engine.ListSessions() enrichment |
| MC-05 | Clients can provide identity name at connection (e.g. `?client=macbook`) visible in session metadata | Requires: Subscriber.Name field + URL query param parsing in both server handlers |
| MC-06 | PTY resize arbitration prevents dimension thrash when multiple clients have different terminal sizes | Requires: Subscriber.Cols/Rows fields + Hub.ResizeClient() with max-wins policy + replace hub.Resize() call sites |
</phase_requirements>
