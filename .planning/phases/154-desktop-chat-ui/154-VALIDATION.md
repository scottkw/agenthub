---
phase: 154
slug: desktop-chat-ui
status: audited
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-26
audited: 2026-06-26
note: Plans implement all tests inline via TDD tasks (each task creates its own test file); there is no separate Wave-0 test-stub plan. Plan-checker verified Dimension 8 compliance 2026-06-26. Post-execution audit (2026-06-26) confirmed all 8 frontend test files (203 tests) + Go relay/webserver dispatch tests run green; both npm packages installed (154-02). wave_0_complete now true.
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

| Req ID | Behavior | Test Type | Automated Command | Status |
|--------|----------|-----------|-------------------|--------|
| CHAT-01 | Enter sends MsgChatSend frame; Shift+Enter inserts newline | unit | `pnpm --filter frontend test run src/lib/relayClient.test.ts` | ✅ green |
| CHAT-02 | ChatMessage fields displayed (alias, tailnetID, HH:MM timestamp; ISO-8601 on hover) | unit | `pnpm --filter frontend test run src/components/Hub/ChatMessage.test.tsx` | ✅ green |
| CHAT-03 | Composer auto-grows to maxRows=6; Markdown renders | unit | `pnpm --filter frontend test run src/components/Hub/ChatPanel.test.tsx` | ✅ green |
| CHAT-04 | Day separator sticky: correct CSS applied to active separator | unit | `pnpm --filter frontend test run src/components/Hub/ChatDaySeparator.test.tsx` | ✅ green (scroll pinning → manual) |
| MENTION-01 | @-mention popover opens on `@`, @session pinned first, keyboard nav works | unit | `pnpm --filter frontend test run src/components/Hub/MentionPopover.test.tsx` | ✅ green |
| NOTIF-01 | Unread badge shows count; @mention badge shows `@` glyph | unit | `pnpm --filter frontend test run src/components/Hub/ChatBadge.test.tsx` | ✅ green |
| NOTIF-02 | @mention row has accent bar + tint + @you chip | unit | `pnpm --filter frontend test run src/components/Hub/ChatMessage.test.tsx` | ✅ green |
| SEC-03 | `<script>alert(1)</script>` renders as text; `<img onerror=...>` strips onerror | unit | `pnpm --filter frontend test run src/components/Hub/ChatPanel.test.tsx -t "sec-03"` | ✅ green |
| D-02 | Overlay drawer: `isActive` unchanged by chatOpen; no `hub-modal__terminal-col` wrapper | unit | `pnpm --filter frontend test run src/components/Hub/HubInteractiveModal.test.tsx` | ✅ green (live PTY no-resize → manual) |
| D-08 | Press-and-hold < 600ms fires nothing; ≥ 600ms fires inject; Enter never injects | unit | `pnpm --filter frontend test run src/components/Hub/ChatPanel.test.tsx -t "inject"` | ✅ green (PTY echo + animation → manual) |
| CHAT-01 (server) | `MsgChatSend` dispatch case wired in relay + webserver read pumps | unit | `go test ./internal/relay/... ./internal/webserver/...` | ✅ green |
| D-08 (UAT defer) | Inject indicator "→ injected into terminal" visible for `SessionInject:true` messages | manual | Phase 154 UAT checklist | — manual |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · — manual-only*

> **Audit note (2026-06-26):** All automated rows re-run green during post-execution validation — 8 frontend test files (203 tests) + Go relay/webserver dispatch tests. D-02 added as an explicit row (covered by `HubInteractiveModal.test.tsx` structural assertions; live-PTY no-resize stays manual). No MISSING or PARTIAL gaps found; no auditor spawn required.

---

## Wave 0 Requirements

- [x] `pnpm --filter frontend add @tanstack/react-virtual@^3.14.3 react-textarea-autosize@^8.5.9` — both packages confirmed in `frontend/package.json`
- [x] `frontend/src/lib/relayClient.test.ts` — encodeChatSendFrame, encodeSessionInjectFrame, parseServerFrame (chat, inject_error cases)
- [x] `frontend/src/components/Hub/ChatPanel.test.tsx` — composer send/receive, SEC-03 XSS, press-and-hold
- [x] `frontend/src/components/Hub/ChatMessage.test.tsx` — field rendering, @mention signals, inject indicator
- [x] `frontend/src/components/Hub/MentionPopover.test.tsx` — popover behavior, @session pinned first
- [x] `frontend/src/components/Hub/ChatBadge.test.tsx` — unread count, @mention state
- [x] `frontend/src/components/Hub/ChatDaySeparator.test.tsx` — sticky CSS, date formatting
- [x] Go test for the new `MsgChatSend` dispatch case — `internal/relay/server_chatsend_test.go` + `internal/webserver/server_chatsend_test.go` (plus `hub_chatsend_test.go`, `server_inject_test.go`)

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
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-06-26 (plan-checker, all 12 dimensions PASS)

---

## Validation Audit 2026-06-26

Post-execution audit (State A) — re-ran every automated row and confirmed the planned test files exist and pass.

| Metric | Count |
|--------|-------|
| Requirements audited | 11 (10 automated + 1 manual-defer) |
| Gaps found | 0 |
| Resolved (auto) | 0 (none needed) |
| Escalated to manual-only | 0 |
| Automated tests green | 203 frontend (8 files) + Go relay/webserver dispatch |

**Evidence:**
- `pnpm test run` over all 8 Phase 154 frontend test files → **8 files, 203 tests passed**
- `go test ./internal/relay/... ./internal/webserver/... -run 'ChatSend|Inject' -count=1` → **ok / ok**
- All 8 frontend packages + 4 Go test files present on disk; both npm deps in `package.json`

**Result:** `nyquist_compliant: true` confirmed. Every requirement has automated verification except the four live-app behaviors (CHAT-04 scroll pinning, D-02 PTY no-resize, D-08 PTY echo, D-08 fill animation), which remain correctly manual-only (jsdom has no layout engine / no live PTY) and are registered as M-20…M-23 in TESTING.md §5.

**Note (out of audit scope, carried from VERIFICATION.md):** NOTIF-02 lacks a dedicated traceability row in TESTING.md §4 (covered under the CHAT-02 row's `ChatMessage.test.tsx`). Documentation-only — the requirement is implemented and tested.
