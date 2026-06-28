# Phase 161: Chat-Sidebar Alias Control — Research

**Researched:** 2026-06-28
**Domain:** Frontend wiring (React/TS ChatPanel + RelayClient) over an already-shipped Go relay/webserver alias backend (Phase 152)
**Confidence:** HIGH (this is a codebase-wiring investigation; every claim below is traced to a specific file/line in the live tree)

## Summary

The Phase 152 alias backend is **complete and symmetric across both server paths**. `MsgAliasSet` (0x34), `AliasStore` (`aliases.json`, global per-`personKey`), and `ValidateAlias` (≤32 runes, no C0/C1) all exist, and both the relay read pump (`internal/relay/server.go:357-373`) and the webserver read pump (`internal/webserver/server.go:1202-1218`) dispatch `MsgAliasSet` identically: validate → `sub.Alias = newAlias` → persist via `AliasSetFn` → `hub.UpdateAlias` → `NotifyPresence` (full-roster rebroadcast). Neither path gates alias-set on `sub.ReadOnly` (D-06) — a verified, intentional behavior (`TestWebReadOnlyCanChat`).

**Open Question (a) is definitively resolved: use the existing relay-client send path. No Wails binding is needed.** `ChatPanel` already constructs and owns a live `RelayClient` (`clientRef`, ChatPanel.tsx:314/416-450) on **every** surface — loopback relay on desktop, webserver WS on web-share. That same client already exposes `sendChat()` / `sendSessionInject()`. The *only* missing piece on the send side is a `sendAliasSet(alias)` method on `RelayClient` (the encoder `encodeAliasSetFrame` already exists at relayClient.ts:94, currently called only in tests; a test mock at TerminalPanel.scale.test.tsx:50 already references a `sendAliasSet()` method name). Add that method + a control inside the shared `ChatPanel`, and all three host surfaces (`HubInteractiveModal`, `TerminalChatHost`, `WebShareSessionView`) inherit it with **zero changes** — cross-surface parity by construction.

The one genuine design gap is **pre-fill / self-identification (decided approach d)**. There is no "this is you" marker in the presence roster. On desktop the user's `personKey` is the constant `"local:local"`, so the current alias is a trivial roster lookup with no wire change. On web-share the client cannot identify its own roster entry (its `personKey` is `tailnetID:web` and the client never learns its own `tailnetID`). Resolving web pre-fill requires a small additive server→client signal. See Open Questions for the concrete recommendation.

**Primary recommendation:** Add `RelayClient.sendAliasSet(alias)` (wraps existing `encodeAliasSetFrame`); add a "chatting as: «name» ✏️" affordance in `ChatPanel`'s `.chat-panel__header`; keep the control **enabled even when `isReadOnly`** (alias-set is not RO-gated); client-side-validate against the `ValidateAlias` rules (using code-point length, not `String.length`); communicate the alias is a **global** display name. For web pre-fill, add a one-way server→client self-identity frame on connect (recommended) — see Open Question 1.

<user_constraints>
## User Constraints (from ROADMAP "Approach — decided 2026-06-27")

> No CONTEXT.md exists for this phase. The ROADMAP "Approach (decided 2026-06-27)" block is the locked design context and is treated as the user's decisions.

### Locked Decisions
- **Surface the already-built Phase 152 backend — do NOT rebuild it.** `MsgAliasSet` (0x34), `AliasStore` (`~/.config/agenthub/aliases.json`), `ValidateAlias`, and `ChatMessage.alias`/`PresenceEntry.alias` already exist and must be reused.
- The control lives in the **shared `ChatPanel` sidebar/header**, NOT in Settings. (Web-share has no Settings page; the shared component gives GUI tab + Hub modal + web-share guest the control from one implementation.)
- (a) Prefer sending `MsgAliasSet` over the **existing relay client** (reuse the wire path; avoid new Wails bindings) — confirm against live client wiring. **→ Confirmed viable; see Architecture.**
- (b) The control must clearly communicate the alias is a **global** display name (per `personKey`), not per-session.
- (c) Keep the shared `ChatPanel` **un-forked** so all surfaces inherit it (parity by construction).
- (d) Pre-fill with the user's current/default alias (Tailnet computed name for web guests, owner default/hostname for desktop).
- ALIAS-UI-02 respects `ValidateAlias` (≤32 runes, no control chars) and updates author name + presence-roster name immediately for all participants.

### Claude's Discretion
- Exact affordance shape (inline-editable "you" roster entry vs. "chatting as: «name» ✏️" header control).
- Mechanism for web self-identification/pre-fill (self-frame vs. `/info` extension vs. optimistic) — research recommends below.
- Client-side validation/error-display UX.

### Deferred Ideas (OUT OF SCOPE)
- Settings → Profile alias section (explicitly moved out to the chat sidebar).
- Read-only-can-post-chat reconciliation — that is **Phase 163** (ROCHAT-01/02), not this phase. (Note: alias-set is *already* un-gated for RO; this phase does not touch the chat-send RO gate.)
- Rebuilding/altering the alias persistence, validation, or dispatch backend.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ALIAS-UI-01 | User sets their alias from the shared chat sidebar; available on GUI tab, Hub modal, and web-share guest — cross-surface parity via shared `ChatPanel`. | Control added inside `ChatPanel` (shared by all 3 hosts) + `RelayClient.sendAliasSet()`. Zero host-component changes needed (HubInteractiveModal.tsx, TerminalChatHost.tsx, WebShareSessionView.tsx all mount the same `<ChatPanel>`). |
| ALIAS-UI-02 | Set alias persists via the Phase 152 `AliasStore`/`MsgAliasSet` path and immediately updates author name + presence-roster name for all participants; respects `ValidateAlias`. | Both server read pumps already persist + `NotifyPresence`-rebroadcast on `MsgAliasSet`. `onPresence` callback updates the roster live (ChatPanel.tsx:419). New chat messages snapshot `sub.Alias`; **past messages are not renamed** (per-message `AuthorAlias` snapshot — see Pitfall 3). |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Alias input UI + client validation | Frontend (shared `ChatPanel`) | — | One shared React component is the single home for all surfaces (parity by construction). |
| Send alias change over the wire | Frontend `RelayClient` → WS | — | Reuse the live WS already owned by `ChatPanel`; symmetric with `sendChat`. |
| Validate / persist / authorize alias | Go server read pump + `AliasStore` | — | Server is authoritative (`ValidateAlias`, global per-`personKey` persistence). Already built. |
| Rebroadcast updated identity | Go `Hub` (`NotifyPresence`) | — | Full-roster rebroadcast already wired on `MsgAliasSet`. |
| Self-identity / current-alias source for pre-fill | Go server (on-connect) → Frontend | — | The one missing seam; the client has no "self" marker today (see Open Q1). |

## Standard Stack

No new libraries. This phase is pure wiring against the existing stack.

### Core (already in tree)
| Item | Location | Purpose |
|------|----------|---------|
| `RelayClient` | `frontend/src/lib/relayClient.ts` | Owns the per-session WS; add `sendAliasSet()` here. `[VERIFIED: codebase]` |
| `encodeAliasSetFrame(alias)` | `relayClient.ts:94-100` | Already encodes `[0x34, ...JSON{alias}]`; currently test-only. `[VERIFIED: codebase]` |
| `ChatPanel` | `frontend/src/components/Hub/ChatPanel.tsx` | Shared drawer; owns `clientRef` + presence roster render (`.chat-panel__header`). Add control here. `[VERIFIED: codebase]` |
| `ValidateAlias` | `internal/relay/protocol.go:200-215` | Server-authoritative rules to mirror client-side. `[VERIFIED: codebase]` |
| `AliasStore` | `internal/daemon/alias_store.go` | Global per-`personKey` persistence (`aliases.json`, 0600). `[VERIFIED: codebase]` |

**Installation:** none — no `package.json` or `go.mod` changes anticipated (unless a new self-identity frame is added, which is still pure stdlib + existing modules).

## Package Legitimacy Audit

Not applicable — this phase installs **no external packages** (npm or Go). All work reuses existing in-tree modules.

## Architecture Patterns

### System Architecture Diagram

```
                         ┌──────────────────────── shared ChatPanel.tsx ───────────────────────┐
   user types alias ───► │  alias control (header)  ──► client ValidateAlias mirror            │
                         │        │ (enabled even if isReadOnly)                                │
                         │        ▼                                                             │
                         │  clientRef.sendAliasSet(alias)  ──► RelayClient.sendAliasSet()       │
                         └────────────────────────────────┬────────────────────────────────────┘
                                                           │ WS frame [0x34, JSON{alias}]
                          desktop (loopback)               │                 web-share (wss + ?cap=)
                  ws://127.0.0.1:{port}/sessions/{id}/ws ◄──┴──► wss://host/sessions/{id}/ws?cap=
                                  │                                                  │
                   internal/relay/server.go read pump            internal/webserver/server.go read pump
                   case MsgAliasSet (357-373)                     case MsgAliasSet (1202-1218)
                                  │  (identical logic, NOT RO-gated — D-06)          │
                                  ▼                                                  ▼
                   ValidateAlias → sub.Alias=new → AliasSetFn (persist AliasStore) → hub.UpdateAlias
                                  │                                                  │
                                  └──────────────► NotifyPresence(hub) ◄─────────────┘
                                                           │ MsgPresence (full roster) to ALL clients
                                                           ▼
                         every ChatPanel.onPresence → setParticipants(...)  (roster name updates live)
                         next ChatMessage snapshots sub.Alias as AuthorAlias (past messages unchanged)
```

### Pattern 1: Add the send method to `RelayClient` (mirror `sendChat`)
**What:** A one-line method symmetric with the existing chat/inject senders.
**Where:** `relayClient.ts`, next to `sendChat` (line ~300).
**Example:**
```typescript
// Source: relayClient.ts (existing sendChat pattern, lines 299-311)
/** Set/update this participant's global display alias. */
sendAliasSet(alias: string): void {
  if (this.ws.readyState === WebSocket.OPEN) {
    this.ws.send(encodeAliasSetFrame(alias))   // encoder already exists at line 94
  }
}
```
**Naming note:** the existing scale-test mock already stubs `sendAliasSet() {}` (TerminalPanel.scale.test.tsx:50) — match that name to avoid a second method alias.

### Pattern 2: The control is a `ChatPanel` header affordance, invoked via `clientRef`
**What:** Inline control in `.chat-panel__header` (alongside the roster + export button). On commit, call `clientRef.current?.sendAliasSet(validated)`.
**Where:** ChatPanel.tsx render, `.chat-panel__header` block (lines 689-737); CSS at `frontend/src/style.css:6698` (`.chat-panel__header`), `:6711` (`__title`), `:6719` (`__roster`).
**Live-update path:** no local state echo strictly required — the server's `NotifyPresence` rebroadcast lands in the existing `onPresence` callback (ChatPanel.tsx:419) and updates the roster for everyone, including the sender. Optionally optimistically reflect the new value in the control until the roster confirms.

### Pattern 3: Pre-fill the current alias
- **Desktop:** find the roster entry where `personKey === 'local:local'` → its `.alias`. The owner key is a hard constant (relay/server.go:265). No wire change. Default before any set = engine hostname (`ownerDefaultAlias = a.engine.hostname`, api.go:278 / server.go:266).
- **Web-share:** the client has no self marker (see Anti-Patterns + Open Q1). Default before any set = Tailscale `ComputedName` (webserver/server.go:1126-1130). Requires a self-identity source.

### Anti-Patterns to Avoid
- **Reusing `isReadOnly` to disable the alias control.** Alias-set is intentionally *not* RO-gated (D-06; `TestWebReadOnlyCanChat`). The chat **Send** button is disabled for RO, but the **alias** control must stay enabled for RO guests. Do not copy the `disabled={isReadOnly}` pattern from the Send button.
- **Using `alias.length` for the 32-rune check.** `ValidateAlias` counts Go runes (code points). JS `String.length` counts UTF-16 units — emoji/astral chars diverge. Use `Array.from(alias).length` (or spread) to mirror the server, or the client will accept strings the server silently rejects.
- **Guessing the web client's own roster entry by index/connCount.** Multiple web guests share `origin:"web"`; only the per-connection `personKey` disambiguates, and the client doesn't know its own. Don't heuristically pick "the last/highest" entry.
- **Adding a Wails `GetAlias`/`SetAlias` binding.** Unnecessary — `ChatPanel` already has a live relay client on the desktop path. Confirmed no such binding exists today (grep of `app.go`/`api.go` shows only the callback wiring, no exported getter/setter).
- **Forking `ChatPanel` per surface.** Breaks parity-by-construction (c). The whole point is one component.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Alias validation | A new client/server validator | `ValidateAlias` (server) + a code-point mirror of its exact rules (client) | Single source of truth already exists; divergence causes silent server-side drops. |
| Alias persistence | Any new store / localStorage | `AliasStore` via `MsgAliasSet` | Global per-`personKey` JSON store already shipped, 0600, restart-surviving. |
| Sending the frame | Hand-rolled WS send | `RelayClient.sendAliasSet` + `encodeAliasSetFrame` | Encoder + framing already written and unit-tested. |
| Broadcasting the change | Any new broadcast path | `NotifyPresence` (already called on `MsgAliasSet`) | Full-roster rebroadcast already wired on both paths. |

**Key insight:** This phase is ~95% frontend glue. The backend round-trip is done, symmetric, and test-covered on both surfaces. The risk is entirely in (1) the RO-enablement nuance and (2) the web pre-fill self-identity seam.

## Runtime State Inventory

This is a feature-add, not a rename/refactor, but the alias system is stateful, so:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `~/.config/agenthub/aliases.json` — global `personKey → alias` map (`AliasStore`). Setting an alias writes here. | None new — reuse. The control writes through the existing `AliasSetFn`. |
| Live service config | Presence roster lives in `Hub` memory; rebroadcast on every `MsgAliasSet`. | None — already wired. |
| OS-registered state | None. | None — verified (no scheduler/service touches alias). |
| Secrets/env vars | None — alias is not a secret; no env var governs it. | None. |
| Build artifacts | None. | None. |

**Note:** Because the alias is **global per person, not per session**, setting it in one session's chat changes the display name in every session for that `personKey`. This is the intended behavior (decided approach b) and the control copy must say so.

## Common Pitfalls

### Pitfall 1: Disabling the alias control for read-only guests
**What goes wrong:** Copying the Send-button's `disabled={isReadOnly}` blocks RO web guests from setting their name.
**Why:** `MsgAliasSet` is deliberately un-gated server-side (D-06, relay/server.go:360-361 + webserver/server.go:1205). RO guests *can* and *should* set an alias.
**How to avoid:** The alias control ignores `isReadOnly`. Only chat-send / `@session` inject remain RO-gated.
**Warning signs:** A web viewer with a read cap can't change their name in UAT.

### Pitfall 2: Rune-count mismatch
**What goes wrong:** Client accepts a 32-emoji name; server's `ValidateAlias` rejects (>32 runes) and silently drops (no NAK) — the name never changes and the user sees no error.
**How to avoid:** Mirror `ValidateAlias` exactly client-side: `trim`, reject empty, reject `Array.from(s).length > 32`, reject any code point `< 0x20` or in `0x7F..0x9F`. Show a client error before sending.
**Warning signs:** "I set my name but nothing happened."

### Pitfall 3: Expecting past messages to rename
**What goes wrong:** Tester sets a new alias and expects already-sent messages to relabel.
**Why:** `ChatMessage.AuthorAlias` is a **per-message snapshot** taken at send time (protocol.go:236-238; HandleChatSend uses `sub.Alias` at hub.go:689). Stored JSONL messages keep their original alias.
**How to avoid:** Scope ALIAS-UI-02 verification to: (a) presence roster updates immediately for all, and (b) *new* messages carry the new author name. Document the snapshot semantics so UAT doesn't flag it as a bug.

### Pitfall 4: No server NAK on invalid/failed alias
**What goes wrong:** On `ValidateAlias` failure or `AliasStore.Set` persist error, the server logs/ignores silently — no client feedback frame exists.
**How to avoid:** Client-side validation is the *only* user feedback path. Validate before send; for the rare persist failure, the absence of a roster update is the (weak) signal — acceptable for this phase, but do not promise inline server-error display.

### Pitfall 5: Web self-identity assumed available
**What goes wrong:** Pre-fill code assumes `currentUserTailnetID` identifies the web user. It defaults to `'local'` and is **not passed** by `WebShareSessionView` (it omits the prop). So a roster self-lookup matches nothing on web.
**How to avoid:** Resolve web self-identity explicitly (Open Q1). As a bonus, doing so also fixes the currently-broken web "mention-of-me" highlight (which today compares against `'local'`).

## Code Examples

### Both server read pumps (already shipped — do not modify)
```go
// Source: internal/relay/server.go:357-373  (webserver/server.go:1202-1218 is identical)
case MsgAliasSet:
    // NOT gated on sub.ReadOnly (D-06). ValidateAlias rejects control chars / over-length.
    var ap AliasPayload
    if json.Unmarshal(payload, &ap) == nil {
        if newAlias := ValidateAlias(ap.Alias); newAlias != "" {
            sub.Alias = newAlias
            if sub.AliasSetFn != nil {
                sub.AliasSetFn(sub.PersonKey, newAlias) // persists to global AliasStore
            }
            hub.UpdateAlias(sub.PersonKey, newAlias)
            NotifyPresence(hub) // full-roster rebroadcast to ALL clients
        }
    }
```

### `ValidateAlias` rules to mirror client-side
```go
// Source: internal/relay/protocol.go:200-215
func ValidateAlias(raw string) string {
    trimmed := strings.TrimSpace(raw)
    if trimmed == "" { return "" }
    runes := []rune(trimmed)
    if len(runes) > 32 { return "" }            // reject, do NOT truncate
    for _, r := range runes {
        if r < 0x0020 || (r >= 0x007F && r <= 0x009F) { return "" } // C0/C1
    }
    return trimmed
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Alias settable only by tests (`encodeAliasSetFrame` test-only) | UI control in shared `ChatPanel` invoking `RelayClient.sendAliasSet` | This phase (161) | Owner + guests can finally set their display name. |
| Proposed Settings → Profile alias section | Shared chat-sidebar control | Decided 2026-06-27 | Web-share has no Settings page; sidebar = one-component parity. |

**Deprecated/outdated:** none. The Phase 152 backend is current.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | A small one-way server→client self-identity frame is the cleanest way to give the web client its own `personKey`/alias for pre-fill. | Open Q1 | If the planner prefers an `/info` extension or optimistic-only pre-fill, the wire change is avoided but web pre-fill fidelity changes. This is a *design recommendation*, not a verified fact — planner decides. |
| A2 | No new npm/Go packages are required. | Standard Stack | Only wrong if the chosen pre-fill mechanism needs something exotic (it doesn't). |

## Open Questions

1. **How does the web-share client learn its own current alias for pre-fill (decided approach d)?** *(The main design decision for the planner. Desktop is already solved by the constant `personKey "local:local"` roster lookup — no wire change.)*
   - **What we know:** The presence roster (`MsgPresence`) has no "self" marker; the same broadcast bytes go to every client. The web client never learns its own `tailnetID`/`personKey`. `WebShareSessionView` doesn't even pass `currentUserTailnetID`. The web default alias (`ComputedName`) is known only server-side at WS-upgrade (`WhoIs`), not at the `/info` HTTP request.
   - **What's unclear:** Which self-identity mechanism to adopt.
   - **Recommendation (ranked):**
     1. **Self-identity frame (recommended).** Add a one-way server→client frame (e.g. `MsgSelf 0x37`) sent once on connect via direct `conn.Write` — right beside the existing pre-scrollback resize write (relay/server.go:297-301; webserver/server.go:1152-1156) — carrying `{personKey, alias}`. Parse in `relayClient.ts`; expose `onSelf(personKey, alias)`. **Uniform across both surfaces, additive (older/raw clients ignore unknown frame, consistent with how 0x30-0x36 were introduced), and it also lets `ChatPanel` mark the "you" roster entry and fixes the web "mention-of-me" highlight as a bonus.** This is a thin, one-way hint — it does not "rebuild" the alias backend (still uses `AliasStore`/`ValidateAlias`/`MsgAliasSet` unchanged).
     2. **Extend `GET /api/sessions/{id}/info`** (already fetched by `ChatPanel` on web mount, ChatPanel.tsx:353) to also return the caller's resolved alias. Lower-footprint on the wire, but `handleSessionInfo` (webserver/server.go:931-961) currently only has `claims.Perms` — it would need to run `WhoIs(r.RemoteAddr)` to resolve the alias/personKey, adding an IPC call to that handler. Desktop has no `/info` fetch, so this still needs the desktop roster-lookup path → two divergent code paths.
     3. **Optimistic / desktop-only pre-fill.** Desktop pre-fills from the `"local:local"` roster entry; web leaves the field empty with a placeholder ("e.g. your name") and just sends. Lowest cost, but partially misses (d) for web.

2. **Affordance shape:** inline-editable "you" entry in the roster vs. a "chatting as: «name» ✏️" header control. Claude's discretion — recommend the header "chatting as:" control because the roster currently renders avatar initials only (no name labels), so an inline-edit there is a larger render change. Either way it lives in `.chat-panel__header`.

## Environment Availability

Skipped — no external runtime dependencies. Pure in-repo frontend (TS/React) + Go change using the existing toolchain (`go`, `pnpm`/vite, vitest, Playwright).

## Validation Architecture

> `workflow.nyquist_validation` is absent from `.planning/config.json` → treated as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (backend) + vitest (frontend unit) + Playwright (cross-surface e2e) |
| Config file | `vitest` via `frontend/`; Playwright `frontend/playwright.config.*`; Go standard |
| Quick run command | `cd frontend && pnpm vitest run src/lib/relayClient.test.ts src/components/Hub/ChatPanel.test.tsx` |
| Full suite command | `go test ./... && cd frontend && pnpm vitest run && pnpm tsc --noEmit && pnpm exec playwright test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ALIAS-UI-01 | `RelayClient.sendAliasSet` emits a 0x34 frame with `{alias}` | unit | `pnpm vitest run src/lib/relayClient.test.ts` | ✅ extend (encodeAliasSetFrame tests at :222) |
| ALIAS-UI-01 | Control renders in `ChatPanel` header, enabled even when `isReadOnly` | unit | `pnpm vitest run src/components/Hub/ChatPanel.test.tsx` | ✅ extend |
| ALIAS-UI-01 | Cross-surface: alias set on web-share updates the other client's roster | e2e | `pnpm exec playwright test frontend/e2e/chat-parity.spec.ts` | ✅ extend (chat-parity.spec.ts) |
| ALIAS-UI-02 | `MsgAliasSet` persists + rebroadcasts presence (relay path) | unit (Go) | `go test ./internal/relay/ -run TestRelayIdentity_AliasPropagation` | ✅ exists (server_identity_test.go:170) |
| ALIAS-UI-02 | `MsgAliasSet` persists + rebroadcasts (web path) + RO can set | unit (Go) | `go test ./internal/webserver/ -run 'TestWebAliasPropagation|TestWebReadOnlyCanChat'` | ✅ exists (identity_test.go) |
| ALIAS-UI-02 | Client validation mirrors `ValidateAlias` (rune count, C0/C1) | unit | `pnpm vitest run src/components/Hub/ChatPanel.test.tsx` | ❌ Wave 0 (new validation tests) |

### Sampling Rate
- **Per task commit:** the quick vitest command above (+ `go test ./internal/relay/ ./internal/webserver/` when touching Go).
- **Per wave merge:** `go test ./... && cd frontend && pnpm vitest run && pnpm tsc --noEmit`.
- **Phase gate:** full suite incl. Playwright green before `/gsd-verify-work`. (Reminder per MEMORY: run `tsc` in the frontend gate — vitest tolerates TS errors that `wails dev`/`vite build` reject.)

### Wave 0 Gaps
- [ ] `frontend/src/lib/relayClient.test.ts` — add `sendAliasSet` → 0x34-frame assertion (covers ALIAS-UI-01).
- [ ] `frontend/src/components/Hub/ChatPanel.test.tsx` — control render, RO-enabled, client `ValidateAlias` mirror, `clientRef.sendAliasSet` called on commit (ALIAS-UI-01/02).
- [ ] `frontend/e2e/chat-parity.spec.ts` — extend: alias set on one web client appears in the other's roster (ALIAS-UI-01 cross-surface).
- [ ] **If Open Q1 → self-identity frame:** Go round-trip + on-connect direct-write tests on **both** paths (relay/server.go + webserver/server.go), plus `relayClient.test.ts` parse/`onSelf` dispatch test.
- [ ] **TESTING.md (standing rule):** register new/extended test files in §2 (Suite Manifest), add §4 traceability rows for ALIAS-UI-01/02 (repo-relative `.ts`/`.tsx`/`.go` paths only), add a §5 manual `M-NN` item only if any behavior is live-app-only. Run `bash tests/check-traceability-paths.sh` before commit.

## Security Domain

> `security_enforcement` absent from config → treated as enabled.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | `ValidateAlias` server-side (authoritative) + code-point client mirror; rejects C0/C1 control chars (T-152-01). |
| V6 Cryptography | no | Alias is not sensitive; no crypto. |
| V4 Access Control | yes (by design un-gated) | Alias-set is intentionally available to RO caps (D-06). Cap/JWT continues to gate PTY input + `@session` inject — untouched here. |
| V2/V3 Auth/Session | no | No change to identity stamping or session auth. |

### Known Threat Patterns for {React render of server-broadcast alias}
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| XSS via crafted alias rendered in roster/author header | Tampering | Alias passes `ValidateAlias` (no control chars) server-side; React escapes text by default in roster (`title`/initial) and author header; chat body already uses rehype-sanitize (SEC-03). Alias is rendered as text, never as HTML. |
| RO guest impersonating via alias | Spoofing | Out of scope to prevent — aliases are user-chosen display names, not identity; the stable `personKey`/`tailnetID` remains the real identity in presence. |
| Over-long / control-char injection | Tampering | `ValidateAlias` reject (not truncate) ≤32 runes + C0/C1 block, on both server paths. |

**If Open Q1 adopts a self-identity frame:** it is server→client only and carries server-resolved values (already `ValidateAlias`-clean) — no new client→server trust surface. No new Go modules.

## Sources

### Primary (HIGH confidence — live codebase, this session)
- `internal/relay/server.go:250-373` — relay owner identity stamping (`local:local`) + `MsgAliasSet` dispatch.
- `internal/webserver/server.go:1116-1218, 925-961, 60-73` — web `WhoIs` identity + `MsgAliasSet` dispatch + `/info` (`sessionListItem`).
- `internal/relay/protocol.go:79-145, 187-215, 224-260` — frame constants, `AliasPayload`, `ValidateAlias`, `ChatMessage` snapshot semantics.
- `internal/daemon/alias_store.go` (whole) — global per-`personKey` JSON store.
- `internal/daemon/api.go:268-278, 506-542` — `SetIdentityProviders`/`SetAliasProviders` callback wiring; `ownerDefaultAlias = engine.hostname`; **no Wails alias binding**.
- `frontend/src/lib/relayClient.ts` (whole) — `encodeAliasSetFrame` (test-only), `RelayClient` send methods, callbacks.
- `frontend/src/components/Hub/ChatPanel.tsx` (whole) — `clientRef`, header/roster render, `isReadOnly`, `/info` probe.
- `frontend/src/components/Hub/{HubInteractiveModal,TerminalChatHost,WebShareSessionView}.tsx` — all mount the same `<ChatPanel>`.
- `internal/relay/server_identity_test.go:156-203`, `internal/webserver/identity_test.go` (via TESTING.md §4) — existing alias round-trip + RO-can-set tests.
- `frontend/src/style.css:6698-6745` — `.chat-panel__header/__roster` insertion point.
- `.planning/ROADMAP.md` (Phase 161 decided approach), `.planning/STATE.md` (roadmap evolution, key decisions).

### Secondary / Tertiary
- None required — no external/web research needed for an internal-wiring phase.

## Metadata

**Confidence breakdown:**
- Standard stack / wiring: HIGH — every claim traced to file:line in the live tree.
- Architecture (send path = relay client, no Wails binding): HIGH — `ChatPanel` owns `clientRef` on all surfaces; backend dispatch verified symmetric; binding absence grep-confirmed.
- Pitfalls (RO-enable, rune count, snapshot semantics): HIGH — backed by code + existing tests.
- Open Q1 mechanism choice: MEDIUM — the *constraint* (no self marker today) is HIGH/verified; the *recommended fix* is a design judgment (`[ASSUMED]` A1).

**Research date:** 2026-06-28
**Valid until:** ~2026-07-28 (stable internal code; re-verify line numbers if Phase 162/163 land first and touch `ChatPanel`/server read pumps).
