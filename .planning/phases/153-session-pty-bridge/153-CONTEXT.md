# Phase 153: @session PTY Bridge - Context

**Gathered:** 2026-06-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Build the **server-side `@session` injection path**: a RW-capability-gated, sanitized,
deliberately-confirmed, one-way bridge that writes a chat message's text into the agent's
live PTY stdin. Covers MENTION-02, MENTION-03, SEC-01, SEC-02.

**Critical sequencing reality:** there is **no chat composer UI yet** — CHAT-01 (the composer)
and MENTION-01 (the `@` autocomplete popover) are both **Phase 154**, and the relay read pump
has no `MsgChat` case at all today. Phase 153 is therefore a **daemon / protocol + security
slice**, not a UI phase. Two of the ROADMAP success criteria reference UI that does not exist
yet ("the chat thread shows a → injected into terminal indicator", "Enter-on-autocomplete");
those visual concerns are satisfied by *persisting/broadcasting* the inject message in this
phase and *rendering* it in Phase 154.

**In scope (Phase 153):**
- A daemon-side inject handler that takes an `@session` text, enforces the RW-cap gate
  **server-side** (SEC-01), sanitizes it (SEC-02), and writes it to the PTY via `Hub.WriteInput`.
- A **dedicated inject frame/verb** (distinct from a normal chat send) so injection can never
  be a side-effect of a stray keypress / normal message (MENTION-03 structural guarantee).
- Persist the inject as a `ChatMessage{ SessionInject: true }` and broadcast it so all
  participants' threads *can* show the "→ injected into terminal" indicator (rendering is 154).
- Server-side rejection (no PTY write) of inject attempts from RO-cap holders, returning an
  explicit error/NAK frame; proven to hold even against a hand-crafted WS frame.

**Out of scope (later phases / explicitly designed out):**
- The chat composer, the `@` autocomplete popover (MENTION-01), the **press-and-hold gesture
  UI**, and the rendered "→ injected into terminal" indicator → **Phase 154** (where the composer
  lives).
- Client-side hiding of the inject affordance for RO holders → **Phase 154** UI layer
  (defense-in-depth on top of this phase's server gate).
- Agent→chat round-trip / reply parsing — permanently out of scope (one-way bridge only).

</domain>

<decisions>
## Implementation Decisions

### Scope of the slice
- **D-01: Phase 153 is a backend/protocol security slice.** Build the daemon inject handler,
  the dedicated inject frame, the RW-cap gate, the sanitizer, and the persist+broadcast of the
  `SessionInject:true` message. Validate end-to-end via Go unit/integration tests and direct WS
  frames — **not** through a composer UI. The visible composer trigger, the press-and-hold
  gesture, and the rendered indicator are deferred to Phase 154 where the composer exists. This
  isolates and hardens the dangerous PTY-write path (with its own threat model + sanitization
  corpus) before any UI rides on top. (Rejected: a throwaway minimal trigger UI now — duplicates
  Phase 154 work and contradicts the 154/155 chat-UI split.)

### Deliberate-confirm contract (MENTION-03)
- **D-02: Injection is anchored to a dedicated inject frame/verb** (e.g. a new
  `MsgSessionInject` constant), structurally separate from a normal chat-send frame. A stray
  Enter, an autocomplete keypress, or an ordinary chat message can **never** reach the PTY,
  because only this distinct frame triggers `WriteInput`. This *is* the "deliberate confirm"
  guarantee at this phase's layer — the actual press-and-hold gesture that emits the frame is
  built in Phase 154. (Rejected for this phase: building the press-and-hold gesture now — needs
  the composer; a confirm-token/two-step handshake — deferred as possible hardening, see
  Discretion, but a distinct frame already delivers the anti-accident guarantee.)

### Sanitization policy (SEC-02)
- **D-03: Collapse newlines; strip beyond the literal C0+escape list.** The sanitizer enforces:
  strip C0 control characters, strip terminal escape sequences (CSI **and** OSC), collapse any
  embedded newlines to single spaces so the result is a one-line prompt, then append **exactly
  one** trailing `\n`. **Go beyond the success-criteria minimum:** also strip C1 controls
  (0x80–0x9F) and Unicode bidirectional overrides (RLO/LRO/PDF/etc.) to close terminal-spoofing
  vectors. (Rejected: rejecting multi-line messages outright — blocks legitimate pasted prompts;
  implementing only the literal C0+CSI/OSC list — leaves C1/bidi spoofing open.)
- The sanitizer unit-test corpus MUST cover: newline injection (LF/CR/CRLF), null bytes, CSI
  sequences, OSC sequences, C1 controls, and bidi overrides — and assert that only printable
  text plus exactly one trailing newline survives.

### RO-holder rejection (SEC-01)
- **D-04: Server rejects + returns an explicit error/NAK frame.** The daemon refuses the inject
  (no PTY write) for any RO-cap holder and emits an explicit error frame the client can surface
  later. The gate is **server-side** and must hold against a hand-crafted WS frame regardless of
  any client-side suppression. Client-side hiding of the affordance is deferred to Phase 154.
  (Rejected: silent server drop — the sender gets no signal their inject was refused; the
  side-channel intent favors an explicit rejection.)

### Claude's Discretion (defer to researcher / planner)
- **Exact frame constant + range.** The concrete byte value for the inject verb (e.g. in the
  `0x21–0x2F` client→server range reserved in Phase 152) and the error/NAK frame shape. Planner
  decides, consistent with `protocol.go` conventions.
- **Which WS paths carry inject.** Both entry paths must enforce the gate, but the relay loopback
  owner (`origin=local`) is implicitly RW; the web-share path keys off `claims.Perms`. Planner
  decides whether the owner path needs the inject verb at all or only the web path does.
- **Inject message authorship/alias.** What `AuthorAlias`/identity the persisted
  `SessionInject:true` message carries (the sender's, per Phase 152 identity stamping) — reuse
  the existing identity-stamping path; no new decision needed unless the planner finds a gap.
- **Optional hardening (confirm token, rate-limit, audit log).** A per-inject confirm token,
  rate-limiting, or audit logging of injects are *not required* by the success criteria. Planner
  may propose them in the threat model but they are not in the locked scope.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone / phase intent
- `.planning/notes/session-chat-discovery.md` — v4.1 Session Chat design record; defines the
  `@session` bridge as **one-way** (agent reply goes to the terminal, not chat; no PTY→message
  parsing) and notes "PTY stdin injection already exists (it's what typing in the terminal does)."
- `.planning/ROADMAP.md` §"Phase 153" — the 4 success criteria (exact-text PTY write +
  indicator; RO rejection holds against direct WS frame injection; daemon-side sanitization with
  a newline/null/CSI test corpus; deliberate-confirm so an accidental keypress does not inject).
- `.planning/REQUIREMENTS.md` — MENTION-02, MENTION-03, SEC-01, SEC-02 (and the broader
  cross-surface PARITY-01 / SEC-03 context landing in 154–155).

### Phase 151–152 carry-forward (locked upstream — read before extending)
- `internal/relay/protocol.go` — `ChatMessage{ AuthorID, AuthorAlias, SchemaVersion,
  SessionInject }` (the `SessionInject bool` field at ~:220 already exists), frame-type
  constants (`MsgInput 0x10` at :16, `MsgMeta 0x20`, the client→server verb range), and
  `MakeInputFrame`. The inject verb is a new constant added here.
- `internal/relay/server.go` — the WS read-pump switch (~:324) where `MsgInput` is RW-gated via
  `!sub.ReadOnly` and chat/typing/alias frames are deliberately **not** gated (D-06). The new
  inject case is added to this switch and IS RW-gated.
- `internal/relay/hub.go` — `Hub.WriteInput(data)` (:409) is the PTY-stdin write target;
  `Subscriber.ReadOnly` (:18) is the per-connection RW signal.
- `internal/webserver/server.go` — `handleWSSRelay` (~:972/996) sources `Subscriber.ReadOnly`
  from `claims.Perms == "read"` (D-24/SEC-04); the web inject path keys off the same claims.
- `internal/daemon/chat.go` — `ChatStore.AppendMessage` / `Export` (~:316–335) already renders
  the "injected into terminal" marker when `SessionInject == true`; the inject path persists
  through this store.
- `internal/capability/capability.go` — `Claims{ SID, Perms, GrantID }` + `HasPerm`;
  `Perms == "read"` is the RO signal that the server gate checks.
- `.planning/phases/152-relay-protocol-identity-presence/152-CONTEXT.md` — D-06 (RO cap gates
  terminal + `@session` only; chat is human-to-human), the two-WS-path model, and identity
  stamping the inject message reuses.
- `internal/relay/server.go` — `ValidateAlias` (control-char rejection) as the **style
  precedent** for the new sanitizer (printable-only, bounded, control-stripping).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/relay/hub.go:409` `Hub.WriteInput(data []byte)` — the exact PTY-stdin write the
  inject path calls after sanitization.
- `internal/daemon/chat.go:316` `ChatStore.AppendMessage` + `Export` — already persist and
  export-mark `SessionInject:true` messages; no new persistence work for the marker.
- `internal/relay/protocol.go:220` `ChatMessage.SessionInject` — the flag already exists; the
  inject handler sets it on the broadcast/persisted message.
- `ValidateAlias` (relay) — existing control-char/length validator to mirror for the sanitizer.

### Established Patterns
- Frame-type switch in `internal/relay/server.go` read pump: `MsgInput` is RW-gated
  (`!sub.ReadOnly`), other chat frames are not. The inject verb follows the `MsgInput` gating
  pattern, **not** the chat/typing/alias ungated pattern (D-06).
- `MakeInputFrame`/`Make*Frame` constructors in `protocol.go` — the new inject frame + its
  error/NAK reply follow this builder convention.
- Two WS entry paths: relay loopback owner (`origin=local`, implicitly RW) vs web-share
  (`claims.Perms`-gated) — both must enforce the inject gate.

### Integration Points
- New inject case in the `server.go` read-pump switch → cap check → sanitize → `WriteInput` →
  `AppendMessage(SessionInject:true)` → broadcast.
- Sanitizer is a new pure function (own package/file) so the test corpus targets it directly.
- Error/NAK frame fan-out back to the originating subscriber on RO rejection.

</code_context>

<specifics>
## Specific Ideas

- The dedicated inject frame is the load-bearing safety mechanism: "injection can only happen
  via this one verb" is what makes an accidental Enter structurally incapable of writing to the
  PTY — call this out explicitly in the threat model.
- Sanitizer output contract, stated as an invariant for the test corpus: *only printable text +
  exactly one trailing `\n` ever reaches `WriteInput`.*
- RO rejection must be proven against a **direct/hand-crafted WS frame**, not just a suppressed
  client — that adversarial test is the proof SEC-01 demands.

</specifics>

<deferred>
## Deferred Ideas

- **Press-and-hold gesture UI + rendered "→ injected into terminal" indicator + RO affordance
  hiding** — Phase 154 (needs the composer). This phase delivers the protocol/persistence they
  build on.
- **Confirm token / rate-limiting / audit logging of injects** — optional hardening; planner may
  raise in the threat model but not in locked scope.

None of the above are scope creep from this discussion — they are roadmap-sequenced into later
phases or explicitly optional.

</deferred>

---

*Phase: 153-session-pty-bridge*
*Context gathered: 2026-06-25*
