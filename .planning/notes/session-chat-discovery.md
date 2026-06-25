---
title: Session Chat (#79) — Discovery & Design Record
date: 2026-06-25
context: gsd-explore discovery on GitHub issue #79 "AgentHub Session Chat"
issue: 79
status: feasible — routed to /gsd-new-milestone
---

# Session Chat (#79) — Discovery & Design Record

## Origin

GitHub issue #79 shipped a detailed implementation briefing + a standalone React prototype
(`agenthub-v4.0-redesign/AgentHub.Chat.Session.standalone.html`, 1.6 MB) modeling a session as a
Slack-style **collaborative chat thread** with multiple human members, presence/typing, @mentions,
a structured message/thread history, tool-output cards, and a simulated streaming agent reply.

Discovery reconciled that prototype against the **actual** codebase and reframed the feature.

## Key reconciliation finding

The prototype's premise (agent participates as a chat author; clean turn-based "Agent message + tool
card" stream) does **not** map onto AgentHub reality. A session is a single-agent **PTY terminal**;
the agent paints a TUI (boxes, spinners, redraws) and emits **no discrete messages**. Segmenting that
paint-stream into clean conversational turns is the same class of problem as the v4.0 mini-preview
tail garble (#96, solved with a headless VT emulator) — but an order of magnitude harder. The
literal-prototype path drags that hard problem in.

## Reframed concept (agreed with user)

A **human-to-human side-channel chat thread attached to (scoped to) each session** — so people
connected to a session can talk to *each other* inside AgentHub instead of leaving for Slack/Discord.

- **Participants:** humans connected to the session (desktop owner + tailnet web-share peers).
- **Identity:** tailnet ID **+** a self-chosen alias, **both visible to all** participants. Real
  identity (trustworthy, already known to the system via Tailscale) plus a friendly label. No login
  screen — preserves AgentHub's passwordless/zero-config ethos.
- **Mentions:**
  - `@alias` → mentions a teammate (normal chat mention).
  - `@session` → injects the message into the agent's PTY as a **prompt**. **One-way bridge** — the
    agent's reply appears in the **terminal**, not the chat. No PTY→message parsing. (Round-trip /
    "agent answers in chat" explicitly deferred — it reintroduces the hard segmentation problem.)
- **Lifetime:** **persisted for the session's life**, stored by the **daemon** (alongside session
  state). Survives app/daemon restarts; gives late joiners full scrollback. Thread dies when the
  session is deleted.
- **Export:** thread is **downloadable as Markdown**.
- **Cross-surface:** participants are inherently mixed (GUI owner + web peers), so chat must work on
  **both desktop GUI and web-share browser**. Per the standing rule, that parity is release-blocking.

## Why it's feasible (builds on existing infrastructure)

- **Stack match:** frontend is React 19 + TS + Vite + pnpm — the prototype's React modules port
  cleanly (no Vue/framework mismatch). The v4.0 Hub already ships in this exact stack.
- **Transport:** extend the existing web-share **WebSocket relay** for message fan-out + presence.
- **Bridge:** PTY stdin injection already exists (it's what typing in the terminal does).
- **Identity:** tailnet peer identity is already known to the system.
- **New work:** daemon-side **message store** (persist + late-join scrollback + Markdown export),
  a **chat UI panel** on both surfaces, and the **alias/mention** layer.

## Deferred / resolve-at-plan-time gray areas

- **UI placement:** chat panel within the session modal vs. a dedicated session-detail route (the
  prototype is a full page; v4.0 is Hub-modal-centric and retired separate sidebar pages).
- **Notifications:** tray toast / Hub-card badge on `@mention` when not viewing the chat.
- **Presence/typing fidelity:** real presence + typing indicators via the relay (prototype's were
  cosmetic timers).
- **Edge case:** the local owner and same-machine web clients aren't distinct tailnet peers — needs a
  disambiguation rule for identity.

## Out of scope (explicitly designed out)

- Agent-as-chat-author / agent replies streaming into the thread (round-trip bridge).
- Tool-output cards rendered from agent output.
- Multi-human chat that outlives the session (searchable archive) — different product.

## Routing

Milestone-sized (daemon store + relay extension + cross-surface UI). No active milestone at discovery
time; #79 was already a milestone candidate. User chose **Start the milestone now** →
`/gsd-new-milestone`.
