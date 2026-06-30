# Phase 161: Chat-Sidebar Alias Control — Pattern Map

**Mapped:** 2026-06-28
**Files analyzed:** 5 (2 primary code + 3 test) + 2 read-only backend confirms (+1 optional Open-Q1 backend touch)
**Analogs found:** 5 / 5 (every new/modified file has an in-tree analog; backend is reuse-only)

> This phase is ~95% frontend glue against the already-shipped Phase 152 alias backend.
> Every analog below lives in the same files that will be modified — the planner can copy
> patterns from the surrounding code in the same file it edits.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/lib/relayClient.ts` (ADD `sendAliasSet`) | utility (WS client) | request-response (client→server frame) | `RelayClient.sendChat` / `sendSessionInject` (same file, lines 299-311) + `encodeAliasSetFrame` (same file, 94-100) | exact (same file, sibling methods) |
| `frontend/src/components/Hub/ChatPanel.tsx` (ADD alias header control) | component | event-driven (user input → WS send + presence echo) | The `.chat-panel__header` block + composer send handlers in the same file (header 688-737; `handleSend` 570-578; `clientRef` 314/450) | exact (same component) |
| `frontend/src/style.css` (ADD `.chat-panel__alias-*` styles) | config (CSS) | n/a | `.chat-panel__header` / `__title` / `__roster` block (6698-6749) | exact |
| `frontend/src/lib/relayClient.test.ts` (ADD `sendAliasSet` 0x34 assertion) | test | request-response | `encodeAliasSetFrame` describe block (222-239) + `RelayClient URL construction` WebSocket-stub block (242-289) | exact |
| `frontend/src/components/Hub/ChatPanel.test.tsx` (ADD control-render + RO-enabled + validation-mirror tests) | test | event-driven | MockRelayClient (`sendChat`/`sendSessionInject` stubs, 26-58) + existing render tests | exact |

**Read-only backend (confirm, do NOT modify):**

| File | Role | Why no change |
|------|------|---------------|
| `internal/relay/server.go:357-373` | server read pump | `MsgAliasSet` already dispatches: ValidateAlias → `sub.Alias` → `AliasSetFn` → `hub.UpdateAlias` → `NotifyPresence`. NOT RO-gated. |
| `internal/webserver/server.go:1202-1218` | server read pump | Identical dispatch on the web path. NOT RO-gated. |
| `internal/relay/protocol.go:200-215` | model (validator) | `ValidateAlias` is the authority the client must mirror. |
| `internal/daemon/alias_store.go` | model (store) | Global per-`personKey` JSON store; written through `AliasSetFn`. |

**Conditional backend (only if Open-Q1 → self-identity frame — planner decides):**

| File | Role | Analog |
|------|------|--------|
| `internal/relay/server.go:297-301` + `internal/webserver/server.go:1150-1156` | server (on-connect direct write) | The pre-scrollback `conn.Write(ctx, MessageBinary, MakeResizeFrame(...))` is the exact insertion point + pattern for a one-way `MsgSelf` frame. |

---

## Pattern Assignments

### `frontend/src/lib/relayClient.ts` — ADD `sendAliasSet(alias)` (utility, request-response)

**Analog:** `RelayClient.sendChat` / `sendSessionInject` in the same file.

The encoder already exists — do NOT write a new one. `encodeAliasSetFrame` (lines 94-100) and the `MSG_ALIAS_SET = 0x34` constant (line 13) are already present and already imported into scope:

```typescript
// EXISTING — relayClient.ts:90-100 (encoder is done; currently test-only)
export function encodeAliasSetFrame(alias: string): Uint8Array {
  const encoded = new TextEncoder().encode(JSON.stringify({ alias }))
  const frame = new Uint8Array(1 + encoded.length)
  frame[0] = MSG_ALIAS_SET
  frame.set(encoded, 1)
  return frame
}
```

**Core pattern to copy** — the new method is a one-to-one mirror of the existing senders (lines 299-311). Place it immediately after `sendChat`:

```typescript
// EXISTING sibling methods to mirror — relayClient.ts:299-311
/** Send a chat message to the relay. */
sendChat(content: string): void {
  if (this.ws.readyState === WebSocket.OPEN) {
    this.ws.send(encodeChatSendFrame(content))
  }
}

/** Inject text into the PTY via the relay. */
sendSessionInject(text: string): void {
  if (this.ws.readyState === WebSocket.OPEN) {
    this.ws.send(encodeSessionInjectFrame(text))
  }
}
```

New method (copy the shape exactly):

```typescript
/** Set/update this participant's global display alias. */
sendAliasSet(alias: string): void {
  if (this.ws.readyState === WebSocket.OPEN) {
    this.ws.send(encodeAliasSetFrame(alias))
  }
}
```

**Naming constraint:** a test mock already stubs the method name `sendAliasSet` (`TerminalPanel.scale.test.tsx:50` per research) — use exactly `sendAliasSet`, no alternate name.

---

### `frontend/src/components/Hub/ChatPanel.tsx` — ADD alias control in `.chat-panel__header` (component, event-driven)

**Analog:** the header render block + the composer send handlers in the same file.

**1. Live relay client is already held** (no new wiring) — `clientRef` is assigned in the subscription effect and used by every send handler:

```typescript
// EXISTING — ChatPanel.tsx:314 (decl), :450 (assign), :570-578 (use)
const clientRef = useRef<RelayClient | null>(null)
// ...
clientRef.current = client            // line 450, inside the subscription useEffect
// ...
function handleSend() {
  if (isReadOnly) return              // ← do NOT copy this RO guard for alias (see Anti-Pattern)
  const text = draftRef.current
  if (text.trim()) {
    clientRef.current?.sendChat(text) // ← copy THIS invocation shape: clientRef.current?.sendAliasSet(validated)
    setDraft('')
    draftRef.current = ''
  }
}
```

The alias commit handler copies `handleSend`'s `clientRef.current?.send…` shape but calls `sendAliasSet` and **omits the `isReadOnly` early-return**.

**2. Header insertion point** — the control mounts inside `.chat-panel__header`, alongside the title/roster/export (lines 688-737):

```typescript
// EXISTING — ChatPanel.tsx:688-737 (insertion site)
<div className="chat-panel__header">
  <span className="chat-panel__title">Chat</span>
  <div className="chat-panel__roster chat-presence" aria-label={...}>
    {participants.slice(0, 3).map(p => ( /* avatar initials only — no name labels */ ))}
  </div>
  <button type="button" className="chat-panel__export-btn" ...>  {/* export icon button */}
</div>
```

Research recommends a `chatting as: «name» ✏️` control here rather than an inline-editable roster entry (the roster renders avatar initials only — line 705 `{p.alias[0]?.toUpperCase()}` — so inline-edit there is a larger change).

**3. Pre-fill source (current alias)** — read from the existing presence roster state, no new state needed:
- **Desktop:** `participants.find(p => p.personKey === 'local:local')?.alias`. Owner key is the hard constant (relay/server.go:265). Default before any set = engine hostname.
- **Web-share:** the client has no self marker today (`currentUserTailnetID` defaults to `'local'`, line 279; `WebShareSessionView` does not pass it). See Open Question 1 below — planner picks the mechanism.

`participants` is already live state, updated by `onPresence` (line 419), which the server's `NotifyPresence` rebroadcast lands in after every `MsgAliasSet`. **No local echo is strictly required** — the roster confirms the change for everyone including the sender. Optionally optimistically reflect the value until the roster confirms.

**4. Client-side validation — MIRROR `ValidateAlias` exactly** (server gives NO NAK on reject, so client validation is the ONLY user feedback path — Pitfall 4). Use code-point length, not `String.length`:

```typescript
// NEW — client mirror of internal/relay/protocol.go:200-215 ValidateAlias
function validateAlias(raw: string): string | null {
  const trimmed = raw.trim()
  if (trimmed === '') return null
  const runes = Array.from(trimmed)          // code points, mirrors Go []rune — NOT .length
  if (runes.length > 32) return null         // reject, do NOT truncate
  for (const ch of runes) {
    const cp = ch.codePointAt(0)!
    if (cp < 0x20 || (cp >= 0x7f && cp <= 0x9f)) return null  // C0 / C1
  }
  return trimmed
}
```

**5. Error display** — there is no existing inline alias-error affordance; the closest analog is the read-only label pattern (lines 904-912, a small muted `<div>` under the composer). Reuse that styling shape for a client-validation message. Keep it client-only (no promised server-error display — Pitfall 4).

**Behavioral note (Pitfall 3):** past messages do NOT rename — `ChatMessage.alias` is a per-message snapshot. Only the roster updates live + NEW messages carry the new author name. Scope ALIAS-UI-02 verification accordingly.

---

### `frontend/src/style.css` — ADD `.chat-panel__alias-*` (config/CSS)

**Analog:** the `.chat-panel__header` token-based block (6698-6749). Reuse the same CSS custom properties:

```css
/* EXISTING tokens to reuse — style.css:6698-6716 */
.chat-panel__header {
  height: 48px; display: flex; align-items: center;
  padding: 0 12px; gap: 8px;
  border-bottom: 1px solid var(--hub-border);
  background: var(--hub-surface);
}
.chat-panel__title {
  font-size: var(--hub-font-size-heading); font-weight: 600;
  color: var(--hub-text-primary); flex: 1;
}
```

Use the same `var(--hub-*)` tokens (`--hub-text-muted`, `--hub-text-dim`, `--hub-radius-sm`, `--hub-surface`, `--hub-border`) the header/export button already use (see inline export-btn styles, ChatPanel.tsx:722-733). Note `.chat-panel__title` has `flex:1` — adding the alias control to the header means the title no longer needs to absorb all slack; lay the new control out with the existing `gap: 8px`.

---

### `frontend/src/lib/relayClient.test.ts` — ADD `sendAliasSet` → 0x34 assertion (test)

**Analog:** the `encodeAliasSetFrame` describe block (222-239) + the WebSocket-stub `RelayClient URL construction` block (242-289). The encoder is already covered; the new test asserts the **method** sends the frame.

```typescript
// EXISTING encoder coverage to extend — relayClient.test.ts:222-239
describe('encodeAliasSetFrame', () => {
  it('produces frame with leading byte 0x34', () => {
    expect(encodeAliasSetFrame('ken')[0]).toBe(0x34)
  })
  it('JSON body parses to {alias:"ken"}', () => { /* ... */ })
  it('handles alias with unicode characters', () => { /* café */ })
})
```

For the method test, copy the WebSocket-stub harness (lines 247-265): stub `WebSocket` with `send: vi.fn()`, force `readyState = WebSocket.OPEN (1)`, construct `RelayClient`, call `client.sendAliasSet('ken')`, assert the stubbed `send` was called with a `Uint8Array` whose `[0] === 0x34` and whose JSON body parses to `{alias:'ken'}`.

---

### `frontend/src/components/Hub/ChatPanel.test.tsx` — ADD control-render / RO-enabled / validation tests (test)

**Analog:** the hoisted `MockRelayClient` (26-58) — extend it with a `sendAliasSet` stub exactly like the existing `sendChat` / `sendSessionInject` stubs:

```typescript
// EXISTING mock to extend — ChatPanel.test.tsx:26-58
const mocks = vi.hoisted(() => ({
  /* ... */
  mockSendChat: vi.fn(),
  mockSendSessionInject: vi.fn(),
  // ADD: mockSendAliasSet: vi.fn(),
}))

vi.mock('../../lib/relayClient', async (importActual) => {
  const actual = await importActual<typeof import('../../lib/relayClient')>()
  return {
    ...actual,
    RelayClient: class MockRelayClient {
      constructor(port, sessionId, cbs) { /* records ctor + callbacks */ }
      close() { mocks.mockClose() }
      sendChat(content: string) { mocks.mockSendChat(content) }
      sendSessionInject(text: string) { mocks.mockSendSessionInject(text) }
      // ADD: sendAliasSet(alias: string) { mocks.mockSendAliasSet(alias) }
    } as unknown as (typeof actual)['RelayClient'],
  }
})
```

New tests to add (Wave 0 gaps from RESEARCH §Validation Architecture):
- Control renders in the header, **enabled even when `isReadOnly === true`** (the critical RO-enable nuance — assert NOT disabled).
- Committing a valid alias calls `mocks.mockSendAliasSet(validated)`.
- Client `ValidateAlias` mirror: a 33-rune name and a name with a C0 char are rejected (no `sendAliasSet` call); a valid name passes.

To exercise `isReadOnly`, the test must drive the `/info` perms fetch — the existing `mockFetch` global (lines 62-63) + the RO-resolution effect (ChatPanel.tsx:341-363) is the analog for setting up a read-only render.

---

## Shared Patterns

### Frame send (client → server)
**Source:** `RelayClient.sendChat` (relayClient.ts:299-304)
**Apply to:** the new `sendAliasSet` method + its `clientRef.current?.sendAliasSet(...)` call site in ChatPanel.
**Rule:** guard on `this.ws.readyState === WebSocket.OPEN`, wrap a pre-existing `encode*Frame`, never hand-roll framing.

### Server-authoritative validation, client mirror
**Source:** `ValidateAlias` (internal/relay/protocol.go:200-215)
**Apply to:** the ChatPanel client-side `validateAlias`.
**Rule:** trim → reject empty → reject `Array.from(s).length > 32` → reject code points `< 0x20` or `0x7f..0x9f`. Code points, never UTF-16 `.length`. Reject (do not truncate). Server gives no NAK, so this is the only user feedback.

### Live identity echo via presence rebroadcast
**Source:** `onPresence: (p) => setParticipants(p)` (ChatPanel.tsx:419) ← server `NotifyPresence` (relay/server.go:372, webserver/server.go:1217)
**Apply to:** alias pre-fill + post-commit confirmation.
**Rule:** the roster is the single source of truth for "current alias" — read pre-fill from it; do not maintain a parallel authoritative alias store client-side.

### RO-aware controls — alias is the EXCEPTION
**Source:** composer send button `disabled={!draft.trim() || isReadOnly}` (ChatPanel.tsx:896) and `handleSend` early-return (line 571).
**Apply to:** NOTHING in the alias control — this is the anti-pattern to avoid (see below). Listed here so the planner explicitly does the opposite.

### On-connect one-way server→client frame (ONLY if Open-Q1 self-identity frame is chosen)
**Source:** pre-scrollback resize write — `conn.Write(ctx, websocket.MessageBinary, MakeResizeFrame(...))` (relay/server.go:297-301; webserver/server.go:1150-1156, where `defaultAlias = who.Node.ComputedName` is already resolved at 1126-1127).
**Apply to:** a `MsgSelf` (e.g. 0x37) frame carrying `{personKey, alias}`, written once on connect at that same site on BOTH paths; parse in `relayClient.ts` (mirror the `parseServerFrame` cases, lines 147-185) and expose an `onSelf` callback (mirror `onPresence`, lines 197/253-254). Additive — older/raw clients ignore unknown frame types (default case, line 187).

---

## No Analog Found

None. Every file to be created or modified has a direct in-tree analog — most in the very same file being edited. The single design gap (web self-identity / pre-fill) is an Open Question of mechanism, not a missing-analog: even the recommended fix (on-connect self-frame) has an exact structural analog in the pre-scrollback resize write.

## Open Question for the Planner (from RESEARCH §Open Questions)

**Web-share self-identity / pre-fill (decided approach d).** Desktop is solved by the `personKey === 'local:local'` roster lookup (no wire change). Web-share needs a self-identity source. Research ranks:
1. **Self-identity frame (recommended)** — `MsgSelf 0x37` one-way on connect (analog above). Uniform across both surfaces; also fixes the currently-broken web "mention-of-me" highlight (compares against `'local'` today).
2. **Extend `GET /api/sessions/{id}/info`** to return the caller's resolved alias — lower wire footprint but desktop has no `/info` fetch → two divergent code paths.
3. **Optimistic / desktop-only** — web field starts empty with a placeholder; lowest cost, partially misses (d) for web.

The choice determines whether any Go is touched this phase; options 1 carries the conditional backend files above. Pure-frontend (option 3) touches zero Go.

## Metadata

**Analog search scope:** `frontend/src/lib/`, `frontend/src/components/Hub/`, `frontend/src/style.css`, `internal/relay/`, `internal/webserver/`, `internal/daemon/`
**Files scanned:** relayClient.ts (full), ChatPanel.tsx (full), relayClient.test.ts (alias + URL blocks), ChatPanel.test.tsx (harness), style.css (chat-panel header block), relay/server.go (read pump + on-connect write), webserver/server.go (grep map), protocol.go (frame constructors), identity tests (both paths)
**Pattern extraction date:** 2026-06-28
