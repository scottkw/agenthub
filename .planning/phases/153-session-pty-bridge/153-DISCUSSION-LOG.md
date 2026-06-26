# Phase 153: @session PTY Bridge - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-25
**Phase:** 153-session-pty-bridge
**Areas discussed:** Scope of slice, Confirm contract, Sanitization policy, RO-holder rejection

---

## Scope of the slice

| Option | Description | Selected |
|--------|-------------|----------|
| Backend/protocol slice | Daemon inject handler + dedicated frame + RW gate + sanitizer + persist/broadcast; validate via Go tests + direct WS frames; trigger/gesture/indicator deferred to Phase 154 | ✓ |
| Minimal end-to-end trigger now | Build a throwaway minimal composer/trigger now, replaced by Phase 154's real composer | |

**User's choice:** Backend/protocol slice (Recommended)
**Notes:** Grounded in the codebase reality — no chat composer UI exists (CHAT-01 + MENTION-01 are Phase 154; relay read pump has no `MsgChat` case). Isolating the dangerous PTY-write path in its own hardened, test-driven phase is the intent.

---

## Deliberate-confirm contract (MENTION-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated inject frame | Injection only via a distinct frame/verb (e.g. MsgSessionInject); a stray Enter / normal message can never reach the PTY. Gesture built in Phase 154 | ✓ |
| Confirm token in protocol | Inject frame must carry a short-lived confirm token / two-step handshake | |
| Build press-and-hold now | Implement the gesture in this phase against a minimal trigger | |

**User's choice:** Dedicated inject frame (Recommended)
**Notes:** The distinct verb is the structural anti-accident guarantee; the press-and-hold gesture that emits it belongs in Phase 154 with the composer. Confirm-token left as optional hardening.

---

## Sanitization policy (SEC-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Collapse newlines, strip C1/bidi too | Embedded newlines → single spaces; one trailing `\n`; also strip C1 (0x80–0x9F) + Unicode bidi overrides beyond the literal C0+CSI/OSC list | ✓ |
| Reject multi-line messages | Server rejects any message containing embedded newlines | |
| Minimal — exactly the criteria | Strip only C0 + CSI/OSC, collapse newlines to one trailing newline | |

**User's choice:** Collapse newlines, strip C1/bidi too (Recommended)
**Notes:** Closes terminal-spoofing vectors beyond the success-criteria minimum while still accepting pasted multi-line prompts as a single-line injection.

---

## RO-holder rejection (SEC-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Server reject + error frame | Daemon rejects (no PTY write) and returns an explicit error/NAK frame; client affordance hiding deferred to Phase 154; proven against a hand-crafted WS frame | ✓ |
| Silent server drop | Daemon silently discards the RO inject with no client feedback | |

**User's choice:** Server reject + error frame (Recommended)
**Notes:** Explicit rejection fits the side-channel intent and gives the sender a signal. The server gate must hold even against a hand-crafted WS frame, independent of any client-side suppression.

---

## Claude's Discretion

- Exact inject frame byte constant + error/NAK frame shape (consistent with `protocol.go`).
- Which WS paths carry the inject verb (relay loopback owner vs web-share path).
- Authorship/alias the persisted `SessionInject:true` message carries (reuse Phase 152 identity stamping).
- Optional hardening — confirm token, rate-limiting, audit logging — may surface in the threat model but is not locked scope.

## Deferred Ideas

- Press-and-hold gesture UI + rendered "→ injected into terminal" indicator + client-side RO affordance hiding → Phase 154 (needs the composer).
- Confirm token / rate-limiting / audit logging of injects → optional hardening, not locked.
