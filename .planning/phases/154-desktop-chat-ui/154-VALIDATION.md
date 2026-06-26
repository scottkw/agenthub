---
phase: 154
slug: desktop-chat-ui
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-26
---

# Phase 154 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (`vitest@^4.1.0` in `frontend/` devDependencies) + `go test` for server dispatch |
| **Config file** | `frontend/vitest.config.ts` (existing); Go uses standard `go test` |
| **Quick run command** | `pnpm --filter frontend test run <changed test file>` |
| **Full suite command** | `pnpm --filter frontend test run` && `go test ./internal/relay/... ./internal/webserver/...` |
| **Estimated runtime** | ~30 seconds (frontend) + ~15 seconds (Go dispatch) |

---

## Sampling Rate

- **After every task commit:** Run the quick run command for the changed test file
- **After every plan wave:** Run the full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Req ID | Behavior | Test Type | Automated Command | File Exists |
|--------|----------|-----------|-------------------|-------------|
| CHAT-01 | Enter sends MsgChatSend frame; Shift+Enter inserts newline | unit | `pnpm --filter frontend test run src/lib/relayClient.test.ts` | ❌ W0 |
| CHAT-02 | ChatMessage fields displayed (alias, tailnetID, HH:MM timestamp; ISO-8601 on hover) | unit | `pnpm --filter frontend test run src/components/Hub/ChatMessage.test.tsx` | ❌ W0 |
| CHAT-03 | Composer auto-grows to maxRows=6; Markdown renders | unit | `pnpm --filter frontend test run src/components/Hub/ChatPanel.test.tsx` | ❌ W0 |
| CHAT-04 | Day separator sticky: correct CSS applied to active separator | unit | `pnpm --filter frontend test run src/components/Hub/ChatDaySeparator.test.tsx` | ❌ W0 |
| MENTION-01 | @-mention popover opens on `@`, @session pinned first, keyboard nav works | unit | `pnpm --filter frontend test run src/components/Hub/MentionPopover.test.tsx` | ❌ W0 |
| NOTIF-01 | Unread badge shows count; @mention badge shows `@` glyph | unit | `pnpm --filter frontend test run src/components/Hub/ChatBadge.test.tsx` | ❌ W0 |
| NOTIF-02 | @mention row has accent bar + tint + @you chip | unit | `pnpm --filter frontend test run src/components/Hub/ChatMessage.test.tsx` | ❌ W0 |
| SEC-03 | `<script>alert(1)</script>` renders as text; `<img onerror=...>` strips onerror | unit | `pnpm --filter frontend test run src/components/Hub/ChatPanel.test.tsx -t "sec-03"` | ❌ W0 |
| D-08 | Press-and-hold < 600ms fires nothing; ≥ 600ms fires inject; Enter never injects | unit | `pnpm --filter frontend test run src/components/Hub/ChatPanel.test.tsx -t "inject"` | ❌ W0 |
| CHAT-01 (server) | `MsgChatSend` dispatch case wired in relay + webserver read pumps | unit | `go test ./internal/relay/... ./internal/webserver/...` | ❌ W0 |
| D-08 (UAT defer) | Inject indicator "→ injected into terminal" visible for `SessionInject:true` messages | manual | Phase 154 UAT checklist | — |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pnpm --filter frontend add @tanstack/react-virtual@^3.14.3 react-textarea-autosize@^8.5.9` — packages not yet installed (RESEARCH §Standard Stack)
- [ ] `frontend/src/lib/relayClient.test.ts` — encodeChatSendFrame, encodeSessionInjectFrame, parseServerFrame (chat, inject_error cases)
- [ ] `frontend/src/components/Hub/ChatPanel.test.tsx` — composer send/receive, SEC-03 XSS, press-and-hold
- [ ] `frontend/src/components/Hub/ChatMessage.test.tsx` — field rendering, @mention signals, inject indicator
- [ ] `frontend/src/components/Hub/MentionPopover.test.tsx` — popover behavior, @session pinned first
- [ ] `frontend/src/components/Hub/ChatBadge.test.tsx` — unread count, @mention state
- [ ] `frontend/src/components/Hub/ChatDaySeparator.test.tsx` — sticky CSS, date formatting
- [ ] Go test stubs for the new `MsgChatSend` dispatch case in `internal/relay/server.go` + `internal/webserver/server.go` (RESEARCH §Undeclared Server-Side Work)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Inject indicator renders for `SessionInject:true` messages | D-06 / Phase 153 carry-forward | Requires live daemon broadcast + visual render comparison against #79 design comp | Send an @session inject in a live session; confirm the system-style "→ injected into terminal" row renders (carry from 153 UAT) |
| Accidental-Enter never injects | D-08 / Phase 153 carry-forward | Gesture timing + PTY side-effect needs live PTY | With `@session` in composer, press Enter → confirm a normal chat message (no PTY write); hold Send 600ms → confirm inject |
| Day separators stay anchored while scrolling history | CHAT-04 | Sticky-during-scroll behavior in the virtualizer needs a live scroll | Scroll a multi-day thread; confirm the active day label stays pinned to viewport top |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 45s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
