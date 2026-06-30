---
phase: "155-web-share-chat-ui-cross-surface-parity-gate"
plan: "04"
subsystem: "e2e parity gate + playwright fixture chat wiring + TESTING.md"
tags: ["parity", "playwright", "e2e", "chat", "export", "testing"]
status: complete

dependency_graph:
  requires: ["155-01", "155-03"]
  provides: ["PARITY-01 e2e gate", "EXPORT-01 e2e gate", "Playwright fixture chat wiring"]
  affects: ["TESTING.md", "cmd/playwright-fixture/main.go", "frontend/e2e/chat-parity.spec.ts"]

tech_stack:
  added: []
  patterns:
    - "Playwright two-context broadcast test with unique Date.now()-suffix messages (Pitfall 5 avoidance)"
    - "Playwright download event assertion for EXPORT-01 (waitForEvent('download'))"
    - "admin-side /__test__/hub-status diagnostic endpoint for broadcast debug"
    - "ChatPanel auto-scroll via prevItemCountRef + virtualizer.scrollToIndex (Pitfall 5 virtualizer guard)"

key_files:
  created:
    - frontend/e2e/chat-parity.spec.ts
  modified:
    - cmd/playwright-fixture/main.go
    - internal/relay/hub.go
    - frontend/src/components/Hub/ChatPanel.tsx
    - TESTING.md

decisions:
  - "viewerAppUrl(env) used for RO SC-3 test — viewer cap has Perms='read' (no write); same fixture cap model as files e2e"
  - "SC-3 adversarial send: fill+Enter UI path (client guard blocks it) instead of raw WS frame — proves client gate; fixture ChatStore AppendMessage + BroadcastChat are the server gate already proven in Phase 154 Go tests"
  - "Auto-scroll fix (prevItemCountRef) is required for broadcast e2e correctness: virtualizer only renders visible items; without scrollToIndex, Page2 test locator never finds the broadcast message in the DOM"
  - "Hub-status diagnostic endpoint uses manager.Get(sessionID) at request time — not a closure capture — so subscriber count reflects live state"
  - "Live Playwright e2e deferred to orchestrator with hard timeout per critical constraint (prior executor hung)"

metrics:
  duration: "~30 minutes (continuation run)"
  completed_date: "2026-06-26"
  tasks_completed: 3
  files_modified: 5
---

# Phase 155 Plan 04: Playwright Fixture Chat Wiring + Cross-Surface Parity Gate Summary

Wire the Playwright fixture to serve chat (seeded ChatStore + ChatHistory/Export providers + RO cap) and ship `frontend/e2e/chat-parity.spec.ts` — the release-blocking PARITY-01/EXPORT-01 cross-surface parity gate; finalize TESTING.md.

## What Was Built

### Task 1 + 2: Playwright Fixture Chat Wiring + chat-parity.spec.ts

The fixture (`cmd/playwright-fixture/main.go`) already had ChatStore + provider wiring committed in the prior executor run (`dff0e20a`, `aaa65eed`). This run finished the remaining WIP and committed:

- `/__test__/hub-status` admin endpoint (subscriberCount + chatAppendFnWired + hubFound) for broadcast e2e diagnostics
- `internal/relay/hub.go` `ChatAppendFnWired()` diagnostic method (used by hub-status)
- `frontend/src/components/Hub/ChatPanel.tsx` auto-scroll fix (prevItemCountRef + useEffect → `virtualizer.scrollToIndex(newCount-1, {align:'end'})`)
- `frontend/e2e/chat-parity.spec.ts` — 8 tests across 6 `test()` blocks:
  - **PARITY-01 SC-1 broadcast**: two RW contexts exchange a unique `parity-broadcast-${Date.now()}` message; Page2 sees it via hub.BroadcastChat
  - **PARITY-01 SC-1 presence**: `.chat-presence` element attaches on both clients after both open chat
  - **PARITY-01 SC-1 unread badge**: Page2 `.chat-badge` appears when Page1 sends while Page2 has chat closed
  - **PARITY-01 SC-1 typing slot**: `.chat-typing` element attaches in DOM (timing behavioral check deferred to M-19)
  - **PARITY-01 SC-1 @mention**: seeded `Mentions:["local"]` message renders as `.chat-msg--mention`
  - **PARITY-01 SC-3 RO gate**: viewer Send button `toBeDisabled()`; adversarial fill+Enter → no new `.chat-msg` (beforeCount == afterCount)
  - **EXPORT-01 SC-2 download**: `[data-chat-export]` click triggers download; filename `/^chat-.*\.md$/`; content has `---`, `session:`, `exported_at:`
  - **PARITY-01 SC-4 inject indicator**: seeded `SessionInject:true` message renders as `.chat-msg--inject`

The fixture seeds 4 messages: RW hello, viewer hello, one `SessionInject:true`, one `Mentions:["local"]`. All selectors are FROZEN (UI-SPEC §5 verbatim).

### Task 3: TESTING.md Finalized

- **Section 2**: Playwright count 7→8; total 499→500; added chat-parity.spec.ts to manifest note
- **Section 4**:
  - Fixed PARITY-01 row: `WebShareSessionView.tsx` → `WebShareSessionView.test.tsx` (Rule 1 bug — path pointed to component file, not test file)
  - Added PARITY-01 → `frontend/e2e/chat-parity.spec.ts` (Playwright e2e gate)
  - Added EXPORT-01 → `frontend/e2e/chat-parity.spec.ts` (download assertion)
  - Added NOTIF-02 → `frontend/src/components/Hub/ChatMessage.test.tsx` (minor gap from Phase 154 VERIFICATION — @mention highlight was noted in CHAT-02 Notes but lacked a dedicated NOTIF-02 row)
- **Section 5**: Added Category M (Web-Share Chat Parity) with M-24 (@session inject via web-share in native WebView — live PTY + inject broadcast)
- `bash tests/check-traceability-paths.sh`: exits 0

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] ChatPanel auto-scroll fix**
- **Found during:** Task 2 implementation
- **Issue:** The `@tanstack/react-virtual` virtualizer only renders DOM nodes for visible list items. When `hub.BroadcastChat` delivers a new message to Page2, the message is appended to `items[]` but the virtualizer does not scroll to the bottom — the new message is outside the render window and invisible to Playwright `.chat-msg` locators. The PARITY-01 SC-1 broadcast test would always time out waiting for the broadcast message.
- **Fix:** Added `prevItemCountRef` + `useEffect` that calls `virtualizer.scrollToIndex(newCount - 1, { align: 'end' })` on first load (`prevItemCountRef.current === 0`) and on each new message when the user is already near bottom (< 150px from bottom). `virtualizer` intentionally omitted from deps (fresh object every render; including it loops).
- **Files modified:** `frontend/src/components/Hub/ChatPanel.tsx`
- **Commit:** `68d3928c`

**2. [Rule 2 - Missing Critical] Hub-status diagnostic endpoint + ChatAppendFnWired()**
- **Found during:** Task 1 (WIP from prior executor)
- **Issue:** The broadcast e2e depends on `hub.chatAppendFn` being wired — without it `HandleChatSend` returns an error and no broadcast fires. A diagnostic endpoint was needed to confirm live hub state during test failures.
- **Fix:** Added `/__test__/hub-status` to the admin mux (reports `subscriberCount`, `chatAppendFnWired`, `hubFound`) and `ChatAppendFnWired()` to `relay.Hub` (reads `h.chatAppendFn != nil` under lock).
- **Files modified:** `cmd/playwright-fixture/main.go`, `internal/relay/hub.go`
- **Commit:** `68d3928c`

**3. [Rule 1 - Bug] Fixed PARITY-01 traceability row in TESTING.md**
- **Found during:** Task 3 (TESTING.md review)
- **Issue:** The PARITY-01 row added in Phase 155-03 pointed to `frontend/src/components/Hub/WebShareSessionView.tsx` (the component file) instead of `frontend/src/components/Hub/WebShareSessionView.test.tsx` (the test file). The traceability-paths.sh script passed because both files exist, but the semantic intent of the path column is a test file.
- **Fix:** Changed `.tsx` → `.test.tsx` in the path column.
- **Files modified:** `TESTING.md`
- **Commit:** `8576557d`

### Out of Scope — Deferred Live Playwright Run

Per the critical constraint in the execution brief: the live `npx playwright test chat-parity` run was NOT executed. The prior executor hung on exactly this step for >1 hour. The orchestrator will run the e2e parity gate under a controlled hard timeout. All non-interactive verifications passed:

- `go build -tags playwrightfixture ./cmd/playwright-fixture/` — OK
- `go build ./...` — OK
- `cd frontend && npx tsc --noEmit` — OK
- `cd frontend && npx vitest run src/components/Hub/` — 344/344 passed (16 files)
- `bash tests/check-traceability-paths.sh` — exits 0

## Commits

| Commit | Type | Description |
|--------|------|-------------|
| `dff0e20a` | feat (prior executor) | Wire ChatStore + chat providers into playwright fixture |
| `aaa65eed` | fix (prior executor) | Add broadcast wiring + frozen Playwright CSS selectors |
| `68d3928c` | feat (this executor) | Add chat-parity spec + hub-status diagnostic + ChatPanel auto-scroll |
| `8576557d` | docs (this executor) | Finalize TESTING.md — Playwright registration, traceability, Phase 155 manual UAT |

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| `frontend/e2e/chat-parity.spec.ts` exists | FOUND |
| `cmd/playwright-fixture/main.go` exists | FOUND |
| `internal/relay/hub.go` exists | FOUND |
| `frontend/src/components/Hub/ChatPanel.tsx` exists | FOUND |
| `TESTING.md` exists | FOUND |
| Commit dff0e20a (prior executor Task 1a) | FOUND |
| Commit aaa65eed (prior executor Task 1b) | FOUND |
| Commit 68d3928c (this executor Task 1+2) | FOUND |
| Commit 8576557d (this executor Task 3) | FOUND |
| `go build -tags playwrightfixture ./cmd/playwright-fixture/` | OK |
| `go build ./...` | OK |
| `bash tests/check-traceability-paths.sh` | OK (exits 0) |
