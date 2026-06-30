# Phase 152: Relay Protocol + Identity + Presence - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-25
**Phase:** 152-relay-protocol-identity-presence
**Areas discussed:** Alias default & persistence, Presence granularity & disambiguation, Typing indicator display, Participant scope (RO viewers)

---

## Alias — default

| Option | Description | Selected |
|--------|-------------|----------|
| Derive from tailnet | Auto-default from tailnet identity (MagicDNS hostname / login name); owner shows as `You (local)` | ✓ |
| Generic placeholder | Show "Guest"/"Anonymous" until set | |
| Force-pick on join | Block participation until an alias is typed | |

**User's choice:** Derive from tailnet
**Notes:** Preserves the passwordless / zero-config ethos — no gate, no identical placeholders.

## Alias — storage

| Option | Description | Selected |
|--------|-------------|----------|
| Daemon-persisted per tailnet-ID | Daemon remembers alias keyed by identity; survives reconnect/late-join/restart | ✓ |
| Connection-scoped (in-memory) | Alias lives only for the current WS connection | |
| Client-remembered | Browser/desktop stores alias locally and re-sends | |

**User's choice:** Daemon-persisted per tailnet-ID
**Notes:** Refined during Area 2 — the persistence key is the composite (TailnetID + origin), not bare TailnetID, so owner and same-machine browser keep separate aliases.

---

## Presence — granularity

| Option | Description | Selected |
|--------|-------------|----------|
| Per-connection | Each live WS connection is its own presence entry (two tabs = two entries) | |
| Per-person, collapsed | Collapse a participant's multiple connections into one entry ("Ken — 2 devices") | ✓ |

**User's choice:** Per-person, collapsed
**Notes:** Needs reference-counting per person key; entry goes disconnected only when the last connection drops.

## Presence — disambiguation (owner vs same-machine browser)

| Option | Description | Selected |
|--------|-------------|----------|
| Composite key: identity + origin | Key = tailnet-ID + origin (desktop-owner vs web); same node → two keys → two entries; stable across reconnect | ✓ |
| Origin + per-connection nonce | Always distinct, but reconnect = new nonce = lost alias continuity | |
| Owner is always "local" sentinel | Owner hard-coded to AuthorID "local"; web clients use real tailnet-ID | |

**User's choice:** Composite key: identity + origin
**Notes:** The composite (TailnetID + origin) is ALSO the per-person collapse key from the previous question. Same-peer tabs collapse; owner (local origin) vs same-machine browser (web origin) stay distinct — satisfies success-criterion 5. Per-connection nonce rejected as it would break the daemon-persisted-alias decision.

---

## Typing indicator — display

| Option | Description | Selected |
|--------|-------------|----------|
| Named, list with overflow | "Ken is typing…" → "Ken and Sam are typing…" → "Ken, Sam +2 typing…"; frame carries typer identity | ✓ |
| Anonymous count | "2 people typing…", no names | |
| Single most-recent only | Only ever show one "<alias> is typing…" | |

**User's choice:** Named, list with overflow
**Notes:** Anonymous count is strictly worse in a 2–3 person side-channel. Typing frame must carry identity/alias. Timings (≤500 ms / 5 s / never stored) locked by success criteria.

---

## Participant scope — read-only viewers

| Option | Description | Selected |
|--------|-------------|----------|
| Full chat participants | RO viewers appear in presence, can post, type, are @mention-able; RO cap gates only the terminal | ✓ |
| Presence-only (read chat, can't post) | RO viewers show in presence and read, but can't send/type | |
| RW-only chat | Chat & presence restricted to RW + owner; RO viewers invisible | |

**User's choice:** Full chat participants
**Notes:** Chat is human-to-human; the RO/RW cap governs the terminal (and, in Phase 153, the separately RW-gated `@session` inject). Matches "participants = humans connected to the session" from the discovery doc.

---

## Claude's Discretion

- **Wire protocol shape** — extend `MetaPayload`/`MsgMeta 0x20` vs. new dedicated frame-type constants (both directions needed). Suggested default noted in CONTEXT.md; planner decides.
- **Abrupt-disconnect detection** feeding the typing/presence TTL (clean close vs. ping/pong timeout; reuse `MsgPing 0x12`).
- **Alias validation** — length cap, charset, trimming; non-unique allowed (identity is the TailnetID).

## Deferred Ideas

None — discussion stayed within phase scope. Adjacent items (`@session` injection, chat UI, notifications, Markdown export) are already separately scoped to Phases 153–155.
